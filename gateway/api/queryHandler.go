package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/XiaoXianHW/ArcNode/gateway/storage"
)

func (s *Server) handleListDevices(c *gin.Context) {
	devices, err := s.Store.ListDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if devices == nil {
		devices = []storage.Device{}
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (s *Server) handleGetDevice(c *gin.Context) {
	d, err := s.Store.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func (s *Server) handleQueryEvents(c *gin.Context) {
	q := storage.EventQuery{
		DeviceID:  c.Query("device_id"),
		Start:     parseInt64(c.Query("start")),
		End:       parseInt64(c.Query("end")),
		EventType: c.Query("type"),
		Category:  c.Query("category"),
		Limit:     int(parseInt64(c.DefaultQuery("limit", "200"))),
		Offset:    int(parseInt64(c.Query("offset"))),
	}
	events, err := s.Store.QueryEvents(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if events == nil {
		events = []storage.Event{}
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (s *Server) handleQuerySegments(c *gin.Context) {
	start, end := dateRange(c)
	q := storage.SegmentQuery{
		DeviceID: c.Query("device_id"),
		Start:    start,
		End:      end,
		Category: c.Query("category"),
		Limit:    int(parseInt64(c.DefaultQuery("limit", "0"))),
		Offset:   int(parseInt64(c.Query("offset"))),
	}
	segments, err := s.Store.QuerySegments(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if segments == nil {
		segments = []storage.Segment{}
	}
	c.JSON(http.StatusOK, gin.H{"segments": segments})
}

func (s *Server) handleCategoryStats(c *gin.Context) {
	start, end := dateRange(c)
	stats, err := s.Store.CategoryStats(c.Query("device_id"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if stats == nil {
		stats = []storage.CategoryStat{}
	}
	c.JSON(http.StatusOK, gin.H{"categories": stats, "start": start, "end": end})
}

func (s *Server) handleAppStats(c *gin.Context) {
	start, end := dateRange(c)
	limit := int(parseInt64(c.DefaultQuery("limit", "20")))
	apps, err := s.Store.AppStats(c.Query("device_id"), start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if apps == nil {
		apps = []storage.AppStat{}
	}
	c.JSON(http.StatusOK, gin.H{"apps": apps, "start": start, "end": end})
}

func (s *Server) handleShortcutStats(c *gin.Context) {
	start, end := dateRange(c)
	limit := int(parseInt64(c.DefaultQuery("limit", "20")))
	shortcuts, err := s.Store.ShortcutStats(c.Query("device_id"), start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if shortcuts == nil {
		shortcuts = []storage.ShortcutStat{}
	}
	c.JSON(http.StatusOK, gin.H{"shortcuts": shortcuts, "start": start, "end": end})
}

func (s *Server) handleSummary(c *gin.Context) {
	start, end := dateRange(c)
	deviceID := c.Query("device_id")
	cats, err := s.Store.CategoryStats(deviceID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	apps, err := s.Store.AppStats(deviceID, start, end, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	shortcuts, err := s.Store.ShortcutStats(deviceID, start, end, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	idle := storage.IdleStat{}
	if deviceID != "" {
		idle, err = s.Store.IdleStats(deviceID, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"start":      start,
		"end":        end,
		"categories": cats,
		"top_apps":   apps,
		"shortcuts":  shortcuts,
		"idle":       idle,
	})
}

func (s *Server) handleCategoryRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"rules": s.Classifier.Rules()})
}

func (s *Server) handleDailyStats(c *gin.Context) {
	days := int(parseInt64(c.DefaultQuery("days", "30")))
	if days <= 0 {
		days = 30
	}
	end := parseInt64(c.Query("end"))
	if end == 0 {
		end = time.Now().Unix()
	}
	start := endOfDay(end) - int64(days)*86400
	buckets, err := s.Store.DailyStats(c.Query("device_id"), c.Query("category"), start, endOfDay(end))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if buckets == nil {
		buckets = []storage.DailyBucket{}
	}
	c.JSON(http.StatusOK, gin.H{"days": buckets, "start": start, "end": endOfDay(end)})
}

func (s *Server) handleHeatmap(c *gin.Context) {
	days := int(parseInt64(c.DefaultQuery("days", "365")))
	if days <= 0 {
		days = 365
	}
	end := parseInt64(c.Query("end"))
	if end == 0 {
		end = time.Now().Unix()
	}
	endTs := endOfDay(end)
	start := endTs - int64(days)*86400
	buckets, err := s.Store.DailyStats(c.Query("device_id"), c.Query("category"), start, endTs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if buckets == nil {
		buckets = []storage.DailyBucket{}
	}
	var maxDur int64
	var totalDur int64
	for _, b := range buckets {
		totalDur += b.Duration
		if b.Duration > maxDur {
			maxDur = b.Duration
		}
	}
	cur, longest, active := streaks(buckets, endTs)
	c.JSON(http.StatusOK, gin.H{
		"days":           buckets,
		"start":          start,
		"end":            endTs,
		"max_duration":   maxDur,
		"total_duration": totalDur,
		"active_days":    active,
		"current_streak": cur,
		"longest_streak": longest,
	})
}

func (s *Server) handleProjectStats(c *gin.Context) {
	start, end := dateRange(c)
	limit := int(parseInt64(c.DefaultQuery("limit", "20")))
	projects, err := s.Store.ProjectStats(c.Query("device_id"), c.Query("category"), start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if projects == nil {
		projects = []storage.ProjectStat{}
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects, "start": start, "end": end})
}

func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func dateRange(c *gin.Context) (int64, int64) {
	if startQ := c.Query("start"); startQ != "" {
		start := parseInt64(startQ)
		end := parseInt64(c.Query("end"))
		if end == 0 {
			end = time.Now().Unix()
		}
		return start, end
	}
	date := c.Query("date")
	loc := time.Local
	var day time.Time
	if date == "" {
		day = time.Now().In(loc)
	} else {
		t, err := time.ParseInLocation("2006-01-02", date, loc)
		if err != nil {
			day = time.Now().In(loc)
		} else {
			day = t
		}
	}
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	end := startOfDay.Add(24 * time.Hour)
	return startOfDay.Unix(), end.Unix()
}

func endOfDay(ts int64) int64 {
	t := time.Unix(ts, 0).In(time.Local)
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Add(24 * time.Hour)
	return d.Unix()
}

func streaks(buckets []storage.DailyBucket, endTs int64) (current, longest, active int) {
	have := make(map[string]bool, len(buckets))
	for _, b := range buckets {
		if b.Duration > 0 {
			have[b.Date] = true
			active++
		}
	}
	end := time.Unix(endTs, 0).In(time.Local).Add(-time.Second)
	for d := end; ; d = d.AddDate(0, 0, -1) {
		if !have[d.Format("2006-01-02")] {
			break
		}
		current++
	}
	run := 0
	dates := make([]string, 0, len(buckets))
	for _, b := range buckets {
		dates = append(dates, b.Date)
	}
	var prev time.Time
	for _, ds := range dates {
		t, err := time.ParseInLocation("2006-01-02", ds, time.Local)
		if err != nil || !have[ds] {
			run = 0
			continue
		}
		if !prev.IsZero() && t.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
		prev = t
	}
	return
}
