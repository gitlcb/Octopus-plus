package op

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
	"gorm.io/gorm/clause"
)

// 多维度小时聚合桶：小时 × 渠道 × 模型 × APIKey。
// 语义：桶保留「该小时全量累计」，落库用整行 replace（OnConflict UpdateAll），
// 三库兼容；flush 后不清空当前小时桶，只清理 2 小时前旧桶，避免重复计数。
type statsDimKey struct {
	Hour      int
	ChannelID int
	ModelName string
	APIKeyID  int
}

const (
	statsDimHistoryWindow = 90 * 24 * time.Hour
	// 桶 key 上限，防 ModelName（自由字符串）异常膨胀。
	statsDimBucketLimit = 4096
	statsDimModelMaxLen = 128
	statsDimNameMaxLen  = 128
)

var statsDimCache = make(map[statsDimKey]*model.StatsDimHourly)
var statsDimCacheLock sync.Mutex

func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// StatsDimUpdate 记录一次请求到对应的维度小时桶。
// channelID/apiKeyID 为 0（失败请求无最终渠道）时照常记录，前端显示 unknown，
// 否则失败请求会在分布里消失、成功率失真。
func StatsDimUpdate(
	channelID int,
	channelName string,
	modelName string,
	apiKeyID int,
	apiKeyName string,
	metrics model.StatsMetrics,
	cacheRead int64,
	cacheWrite int64,
	ftut int64,
) {
	modelName = truncateRunes(strings.TrimSpace(modelName), statsDimModelMaxLen)
	if modelName == "" {
		modelName = "unknown"
	}
	channelName = truncateRunes(strings.TrimSpace(channelName), statsDimNameMaxLen)
	apiKeyName = truncateRunes(strings.TrimSpace(apiKeyName), statsDimNameMaxLen)

	now := time.Now()
	hour := int(now.Unix() / 3600)
	date := now.Format("20060102")

	key := statsDimKey{Hour: hour, ChannelID: channelID, ModelName: modelName, APIKeyID: apiKeyID}

	statsDimCacheLock.Lock()
	defer statsDimCacheLock.Unlock()

	entry, ok := statsDimCache[key]
	if !ok {
		if len(statsDimCache) >= statsDimBucketLimit {
			log.Warnw("stats_dim.bucket_limit_exceeded", "limit", statsDimBucketLimit, "model", modelName)
			return
		}
		entry = &model.StatsDimHourly{
			Hour:      hour,
			ChannelID: channelID,
			ModelName: modelName,
			APIKeyID:  apiKeyID,
			Date:      date,
		}
		statsDimCache[key] = entry
	}
	// 快照名以最后一次为准（渠道/Key 改名时展示最新）。
	if channelName != "" {
		entry.ChannelName = channelName
	}
	if apiKeyName != "" {
		entry.APIKeyName = apiKeyName
	}
	entry.StatsMetrics.Add(metrics)
	entry.CacheReadToken += cacheRead
	entry.CacheWriteToken += cacheWrite
	entry.FtutTime += ftut
}

// StatsDimSaveDB 把内存桶整行 replace 入库，并清理 2 小时前的旧桶。
// 由 stats 后台任务调用。
func StatsDimSaveDB(ctx context.Context) error {
	statsDimCacheLock.Lock()
	if len(statsDimCache) == 0 {
		statsDimCacheLock.Unlock()
		return nil
	}
	rows := make([]model.StatsDimHourly, 0, len(statsDimCache))
	for _, entry := range statsDimCache {
		rows = append(rows, *entry)
	}
	// 只清理已封口的旧小时桶（当前小时与上一小时保留，保证 replace 语义正确）。
	cutoff := int(time.Now().Unix()/3600) - 1
	for k := range statsDimCache {
		if k.Hour < cutoff {
			delete(statsDimCache, k)
		}
	}
	statsDimCacheLock.Unlock()

	dbConn := db.GetDB().WithContext(ctx)
	return dbConn.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "hour"}, {Name: "channel_id"}, {Name: "model_name"}, {Name: "api_key_id"},
		},
		UpdateAll: true,
	}).CreateInBatches(&rows, 200).Error
}

// StatsDimCleanup 删除超出保留窗口的历史行。
func StatsDimCleanup(ctx context.Context) error {
	cutoff := int(time.Now().Add(-statsDimHistoryWindow).Unix() / 3600)
	return db.GetDB().WithContext(ctx).
		Where("hour < ?", cutoff).
		Delete(&model.StatsDimHourly{}).Error
}

// StatsDimRow 是聚合查询的单行结果（供 handler 直接序列化）。
type StatsDimRow struct {
	Time int64  `json:"time,omitempty"` // bucket 起始 epoch 秒（total 时为 0）
	Date string `json:"date,omitempty"` // 天粒度日期
	Key  string `json:"key"`            // 维度键（model_name / channel_id / api_key_id / "all"）
	Label string `json:"label"`         // 展示名

	InputToken      int64   `json:"input_token"`
	OutputToken     int64   `json:"output_token"`
	InputCost       float64 `json:"input_cost"`
	OutputCost      float64 `json:"output_cost"`
	WaitTime        int64   `json:"wait_time"`
	RequestSuccess  int64   `json:"request_success"`
	RequestFailed   int64   `json:"request_failed"`
	CacheReadToken  int64   `json:"cache_read_token"`
	CacheWriteToken int64   `json:"cache_write_token"`
	FtutTime        int64   `json:"ftut_time"`
}

// StatsDimParams 聚合查询参数。
type StatsDimParams struct {
	GroupBy string // none | model | channel | apikey
	Bucket  string // hour | day | total
	From    int64  // unix 秒
	To      int64  // unix 秒
	Limit   int    // >0 时 top-N，其余归并为 __other__
}

// mergedDimEntry 用于 DB 行 + 内存桶合并。
type mergedDimKey struct {
	Bucket string // date 或 hour 字符串或 "total"
	Dim    string // 维度键
}

// StatsDimQuery 聚合查询：DB 历史 + 内存未刷盘桶合并，按 bucket×dimension 求和。
func StatsDimQuery(ctx context.Context, p StatsDimParams) ([]StatsDimRow, error) {
	fromHour := int(p.From / 3600)
	toHour := int(p.To / 3600)

	var rows []model.StatsDimHourly
	if err := db.GetDB().WithContext(ctx).
		Where("hour >= ? AND hour <= ?", fromHour, toHour).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	// 合并未刷盘内存桶。
	statsDimCacheLock.Lock()
	for k, entry := range statsDimCache {
		if k.Hour >= fromHour && k.Hour <= toHour {
			rows = append(rows, *entry)
		}
	}
	statsDimCacheLock.Unlock()

	agg := make(map[mergedDimKey]*StatsDimRow)
	labels := make(map[string]string) // 维度键 -> 最新展示名

	for i := range rows {
		r := &rows[i]
		dim, label := dimKeyLabel(p.GroupBy, r)
		if label != "" {
			labels[dim] = label
		}
		bucket := bucketOf(p.Bucket, r)
		mk := mergedDimKey{Bucket: bucket, Dim: dim}
		row, ok := agg[mk]
		if !ok {
			row = &StatsDimRow{Key: dim}
			switch p.Bucket {
			case "day":
				row.Date = r.Date
				row.Time = dayStartUnix(r.Date)
			case "hour":
				row.Time = int64(r.Hour) * 3600
			}
			agg[mk] = row
		}
		row.InputToken += r.InputToken
		row.OutputToken += r.OutputToken
		row.InputCost += r.InputCost
		row.OutputCost += r.OutputCost
		row.WaitTime += r.WaitTime
		row.RequestSuccess += r.RequestSuccess
		row.RequestFailed += r.RequestFailed
		row.CacheReadToken += r.CacheReadToken
		row.CacheWriteToken += r.CacheWriteToken
		row.FtutTime += r.FtutTime
	}

	out := make([]StatsDimRow, 0, len(agg))
	for _, row := range agg {
		if lbl, ok := labels[row.Key]; ok && lbl != "" {
			row.Label = lbl
		} else {
			row.Label = row.Key
		}
		out = append(out, *row)
	}

	// top-N 归并（仅对分维度且指定 limit 生效）。
	if p.Limit > 0 && p.GroupBy != "none" {
		out = applyTopN(out, p.Bucket, p.Limit)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Time != out[j].Time {
			return out[i].Time < out[j].Time
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func dimKeyLabel(groupBy string, r *model.StatsDimHourly) (string, string) {
	switch groupBy {
	case "model":
		return r.ModelName, r.ModelName
	case "channel":
		return itoa(r.ChannelID), r.ChannelName
	case "apikey":
		return itoa(r.APIKeyID), r.APIKeyName
	default: // none
		return "all", "all"
	}
}

func bucketOf(bucket string, r *model.StatsDimHourly) string {
	switch bucket {
	case "day":
		return r.Date
	case "total":
		return "total"
	default: // hour
		return itoa(r.Hour)
	}
}

// applyTopN 按总费用取前 N 个维度，其余合并为 __other__（保持每个时间桶）。
func applyTopN(rows []StatsDimRow, bucket string, limit int) []StatsDimRow {
	costByDim := make(map[string]float64)
	for i := range rows {
		costByDim[rows[i].Key] += rows[i].InputCost + rows[i].OutputCost
	}
	type kv struct {
		k string
		v float64
	}
	dims := make([]kv, 0, len(costByDim))
	for k, v := range costByDim {
		dims = append(dims, kv{k, v})
	}
	sort.Slice(dims, func(i, j int) bool { return dims[i].v > dims[j].v })
	keep := make(map[string]struct{})
	for i := 0; i < len(dims) && i < limit; i++ {
		keep[dims[i].k] = struct{}{}
	}
	if len(dims) <= limit {
		return rows
	}

	out := make([]StatsDimRow, 0, len(rows))
	otherByBucket := make(map[string]*StatsDimRow)
	for i := range rows {
		if _, ok := keep[rows[i].Key]; ok {
			out = append(out, rows[i])
			continue
		}
		bk := bucketKeyOf(bucket, rows[i])
		o, ok := otherByBucket[bk]
		if !ok {
			o = &StatsDimRow{Key: "__other__", Label: "__other__", Time: rows[i].Time, Date: rows[i].Date}
			otherByBucket[bk] = o
		}
		o.InputToken += rows[i].InputToken
		o.OutputToken += rows[i].OutputToken
		o.InputCost += rows[i].InputCost
		o.OutputCost += rows[i].OutputCost
		o.WaitTime += rows[i].WaitTime
		o.RequestSuccess += rows[i].RequestSuccess
		o.RequestFailed += rows[i].RequestFailed
		o.CacheReadToken += rows[i].CacheReadToken
		o.CacheWriteToken += rows[i].CacheWriteToken
		o.FtutTime += rows[i].FtutTime
	}
	for _, o := range otherByBucket {
		out = append(out, *o)
	}
	return out
}

func bucketKeyOf(bucket string, r StatsDimRow) string {
	switch bucket {
	case "day":
		return r.Date
	case "total":
		return "total"
	default:
		return itoa(int(r.Time / 3600))
	}
}

func dayStartUnix(date string) int64 {
	t, err := time.ParseInLocation("20060102", date, time.Local)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
