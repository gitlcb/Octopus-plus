package op

import (
	"context"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
	"gorm.io/gorm/clause"
)

// StatsDimBackfill 一次性从最近的 relay_logs 回填 StatsDimHourly 多维度聚合表，
// 让首次启用统计页的实例立即看到历史数据。已回填则跳过。窗口 90 天。
//
// 注意:relay_logs 只有 RequestAPIKeyName(名称,无 ID),故回填的 APIKeyID 统一为 0,
// APIKeyName 存快照;后续实时打点才有真实 APIKeyID。按 api key 维度分组时,
// 历史数据会归并到 key=0(展示用名称)。
func StatsDimBackfill(ctx context.Context) {
	done, err := SettingGetBool(model.SettingKeyStatsDimBackfilled)
	if err == nil && done {
		return
	}

	startTime := time.Now()
	cutoff := startTime.Add(-statsDimHistoryWindow).Unix()

	const pageSize = 200
	type aggKey struct {
		Hour      int
		ChannelID int
		ModelName string
	}
	aggregated := make(map[aggKey]*model.StatsDimHourly)

	add := func(r *dimBackfillLogRow) {
		modelName := truncateRunes(strings.TrimSpace(r.ActualModelName), statsDimModelMaxLen)
		if modelName == "" {
			modelName = truncateRunes(strings.TrimSpace(r.RequestModelName), statsDimModelMaxLen)
		}
		if modelName == "" {
			modelName = "unknown"
		}
		hour := int(r.Time / 3600)
		k := aggKey{Hour: hour, ChannelID: r.ChannelId, ModelName: modelName}
		entry, ok := aggregated[k]
		if !ok {
			if len(aggregated) >= statsDimBucketLimit*8 {
				return // 回填期间硬上限,防异常库炸内存
			}
			entry = &model.StatsDimHourly{
				Hour:      hour,
				ChannelID: r.ChannelId,
				ModelName: modelName,
				APIKeyID:  0,
				Date:      time.Unix(r.Time, 0).Format("20060102"),
			}
			aggregated[k] = entry
		}
		if r.ChannelName != "" {
			entry.ChannelName = truncateRunes(r.ChannelName, statsDimNameMaxLen)
		}
		if r.RequestAPIKeyName != "" {
			entry.APIKeyName = truncateRunes(r.RequestAPIKeyName, statsDimNameMaxLen)
		}
		entry.InputToken += int64(r.InputTokens)
		entry.OutputToken += int64(r.OutputTokens)
		entry.InputCost += r.Cost // relay_logs 仅有合计 Cost,记入 input_cost 侧
		entry.WaitTime += int64(r.UseTime)
		if r.Success {
			entry.RequestSuccess++
		} else {
			entry.RequestFailed++
		}
		if r.CacheReadTokens != nil {
			entry.CacheReadToken += int64(*r.CacheReadTokens)
		}
		if r.CacheWriteTokens != nil {
			entry.CacheWriteToken += int64(*r.CacheWriteTokens)
		}
		entry.FtutTime += int64(r.Ftut)
	}

	var lastID int64
	for {
		var batch []dimBackfillLogRow
		if err := db.GetDB().WithContext(ctx).
			Model(&model.RelayLog{}).
			Where("time >= ? AND id > ?", cutoff, lastID).
			Order("id ASC").
			Limit(pageSize).
			Find(&batch).Error; err != nil {
			log.Warnf("stats dim backfill: scan logs failed: %v", err)
			return
		}
		if len(batch) == 0 {
			break
		}
		for i := range batch {
			lastID = batch[i].ID
			add(&batch[i])
		}
		if len(batch) < pageSize {
			break
		}
	}

	if len(aggregated) == 0 {
		_ = SettingSetString(model.SettingKeyStatsDimBackfilled, "true")
		log.Infof("stats dim backfill: no data, marked complete (took %s)", time.Since(startTime))
		return
	}

	rows := make([]model.StatsDimHourly, 0, len(aggregated))
	for _, v := range aggregated {
		rows = append(rows, *v)
	}

	const upsertChunk = 500
	dbConn := db.GetDB().WithContext(ctx)
	for i := 0; i < len(rows); i += upsertChunk {
		end := i + upsertChunk
		if end > len(rows) {
			end = len(rows)
		}
		// DoNothing:回填不覆盖实时打点已写入的行(避免与运行中累加冲突)。
		if err := dbConn.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "hour"}, {Name: "channel_id"}, {Name: "model_name"}, {Name: "api_key_id"},
			},
			DoNothing: true,
		}).Create(rows[i:end]).Error; err != nil {
			log.Warnf("stats dim backfill: insert chunk failed: %v", err)
			return
		}
	}

	if err := SettingSetString(model.SettingKeyStatsDimBackfilled, "true"); err != nil {
		log.Warnf("stats dim backfill: failed to mark complete: %v", err)
		return
	}
	log.Infof("stats dim backfill done: %d rows in %s", len(rows), time.Since(startTime))
}

// dimBackfillLogRow 精简扫描行:刻意不含 request_content/response_content 等大字段,
// GORM 按字段裁剪 SELECT,避免 OOM。不要加内容字段。
type dimBackfillLogRow struct {
	ID                int64
	Time              int64
	ChannelId         int
	ChannelName       string
	ActualModelName   string
	RequestModelName  string
	RequestAPIKeyName string
	InputTokens       int
	OutputTokens      int
	CacheReadTokens   *int
	CacheWriteTokens  *int
	Ftut              int
	UseTime           int
	Cost              float64
	Success           bool
}
