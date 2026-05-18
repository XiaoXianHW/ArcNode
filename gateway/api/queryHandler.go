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
	endOfDay := startOfDay.Add(24 * time.Hour)
	return startOfDay.Unix(), endOfDay.Unix()
}
