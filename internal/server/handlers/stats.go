package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/stats").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/today", http.MethodGet).
				Handle(getStatsToday),
		).
		AddRoute(
			router.NewRoute("/daily", http.MethodGet).
				Handle(getStatsDaily),
		).
		AddRoute(
			router.NewRoute("/hourly", http.MethodGet).
				Handle(getStatsHourly),
		).
		AddRoute(
			router.NewRoute("/total", http.MethodGet).
				Handle(getStatsTotal),
		).
		AddRoute(
			router.NewRoute("/apikey", http.MethodGet).
				Handle(getStatsAPIKey),
		).
		AddRoute(
			router.NewRoute("/dimension", http.MethodGet).
				Handle(getStatsDimension),
		)
}

func getStatsToday(c *gin.Context) {
	resp.Success(c, op.StatsTodayGet())
}

func getStatsDaily(c *gin.Context) {
	statsDaily, err := op.StatsGetDaily(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, statsDaily)
}

func getStatsHourly(c *gin.Context) {
	resp.Success(c, op.StatsHourlyGet())
}

func getStatsTotal(c *gin.Context) {
	resp.Success(c, op.StatsTotalGet())
}

func getStatsAPIKey(c *gin.Context) {
	resp.Success(c, op.StatsAPIKeyList())
}

var statsDimGroupBy = map[string]struct{}{"none": {}, "model": {}, "channel": {}, "apikey": {}}
var statsDimBucket = map[string]struct{}{"hour": {}, "day": {}, "total": {}}

// getStatsDimension 通用多维度聚合:
//
//	group_by=none|model|channel|apikey  bucket=hour|day|total
//	from,to=unix秒(缺省近7天)  limit=top-N(可选)
func getStatsDimension(c *gin.Context) {
	groupBy := c.DefaultQuery("group_by", "none")
	if _, ok := statsDimGroupBy[groupBy]; !ok {
		resp.Error(c, http.StatusBadRequest, "invalid group_by")
		return
	}
	bucket := c.DefaultQuery("bucket", "day")
	if _, ok := statsDimBucket[bucket]; !ok {
		resp.Error(c, http.StatusBadRequest, "invalid bucket")
		return
	}

	now := time.Now().Unix()
	from := parseInt64(c.Query("from"), now-7*24*3600)
	to := parseInt64(c.Query("to"), now)
	if to < from {
		from, to = to, from
	}
	limit := int(parseInt64(c.Query("limit"), 0))
	if limit < 0 {
		limit = 0
	}

	rows, err := op.StatsDimQuery(c.Request.Context(), op.StatsDimParams{
		GroupBy: groupBy,
		Bucket:  bucket,
		From:    from,
		To:      to,
		Limit:   limit,
	})
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, gin.H{"bucket": bucket, "group_by": groupBy, "rows": rows})
}

func parseInt64(s string, def int64) int64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}
