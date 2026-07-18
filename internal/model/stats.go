package model

type StatsMetrics struct {
	InputToken     int64   `json:"input_token" gorm:"bigint"`
	OutputToken    int64   `json:"output_token" gorm:"bigint"`
	InputCost      float64 `json:"input_cost" gorm:"type:real"`
	OutputCost     float64 `json:"output_cost" gorm:"type:real"`
	WaitTime       int64   `json:"wait_time" gorm:"bigint"`
	RequestSuccess int64   `json:"request_success" gorm:"bigint"`
	RequestFailed  int64   `json:"request_failed" gorm:"bigint"`
}

type StatsTotal struct {
	ID int `gorm:"primaryKey"`
	StatsMetrics
}

type StatsHourly struct {
	Hour int    `json:"hour" gorm:"primaryKey"`
	Date string `json:"date" gorm:"not null"` // 记录最后更新日期，格式：20060102
	StatsMetrics
}

type StatsDaily struct {
	Date string `json:"date" gorm:"primaryKey"`
	StatsMetrics
}

type StatsModel struct {
	ID        int    `json:"id" gorm:"primaryKey"`
	Name      string `json:"name" gorm:"not null"`
	ChannelID int    `json:"channel_id" gorm:"not null"`
	StatsMetrics
}

type StatsChannel struct {
	ChannelID int `json:"channel_id" gorm:"primaryKey"`
	StatsMetrics
}

type StatsAPIKey struct {
	APIKeyID int `json:"api_key_id" gorm:"primaryKey"`
	StatsMetrics
}

// StatsSiteModelHourly 站点渠道按小时聚合的请求统计，
// 用于站点渠道页折线图，覆盖任意时间跨度的可用性趋势。
type StatsSiteModelHourly struct {
	Hour          int    `json:"hour" gorm:"primaryKey;autoIncrement:false"`
	SiteAccountID int    `json:"site_account_id" gorm:"primaryKey;index:idx_stats_site_model_lookup"`
	GroupKey      string `json:"group_key" gorm:"primaryKey;type:varchar(128);index:idx_stats_site_model_lookup"`
	ModelName     string `json:"model_name" gorm:"primaryKey;type:varchar(128);index:idx_stats_site_model_lookup"`
	Date          string `json:"date" gorm:"not null;type:varchar(8)"`
	LastRequestAt int64  `json:"last_request_at" gorm:"not null;default:0"`
	StatsMetrics
}

// StatsDimHourly 按 小时 × 渠道 × 模型 × APIKey 维度聚合的请求统计，
// 供多维度统计页的时间趋势、分布与交叉堆叠使用。
// 复合主键决定粒度；Date 冗余存本地日用于天粒度 GROUP BY；ChannelName/APIKeyName
// 为快照列（渠道/Key 删除后仍可展示历史名称）。
type StatsDimHourly struct {
	Hour        int    `json:"hour" gorm:"primaryKey;autoIncrement:false"` // epoch 秒 / 3600
	ChannelID   int    `json:"channel_id" gorm:"primaryKey"`
	ModelName   string `json:"model_name" gorm:"primaryKey;type:varchar(128)"`
	APIKeyID    int    `json:"api_key_id" gorm:"primaryKey"`
	Date        string `json:"date" gorm:"not null;type:varchar(8);index:idx_stats_dim_date"` // 本地日 20060102
	ChannelName string `json:"channel_name" gorm:"type:varchar(128)"`
	APIKeyName  string `json:"api_key_name" gorm:"type:varchar(128)"`
	StatsMetrics
	CacheReadToken  int64 `json:"cache_read_token" gorm:"bigint"`
	CacheWriteToken int64 `json:"cache_write_token" gorm:"bigint"`
	FtutTime        int64 `json:"ftut_time" gorm:"bigint"` // 首字耗时总和(ms)，除以成功数得均值
}

// Add aggregates another StatsMetrics into the current one.
func (s *StatsMetrics) Add(delta StatsMetrics) {
	s.InputToken += delta.InputToken
	s.OutputToken += delta.OutputToken
	s.InputCost += delta.InputCost
	s.OutputCost += delta.OutputCost
	s.WaitTime += delta.WaitTime
	s.RequestSuccess += delta.RequestSuccess
	s.RequestFailed += delta.RequestFailed
}
