package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/XiaoXianHW/ArcNode/gateway/category"
)

func (s *Server) handleHourlyStats(c *gin.Context) {
	days := int(parseInt64(c.DefaultQuery("days", "30")))
	if days <= 0 {
		days = 30
	}
	end := parseInt64(c.Query("end"))
	if end == 0 {
		end = time.Now().Unix()
	}
	endTs := endOfDay(end)
	start := endTs - int64(days)*86400
	buckets, err := s.Store.HourlyStats(c.Query("device_id"), c.Query("category"), start, endTs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var maxDur int64
	for _, b := range buckets {
		if b.Duration > maxDur {
			maxDur = b.Duration
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"buckets":      buckets,
		"start":        start,
		"end":          endTs,
		"max_duration": maxDur,
	})
}

func (s *Server) handleBalanceStats(c *gin.Context) {
	days := int(parseInt64(c.DefaultQuery("days", "14")))
	if days <= 0 {
		days = 14
	}
	end := parseInt64(c.Query("end"))
	if end == 0 {
		end = time.Now().Unix()
	}
	endTs := endOfDay(end)
	start := endTs - int64(days)*86400
	rows, err := s.Store.DailyCategoryStats(c.Query("device_id"), start, endTs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"rows":  rows,
		"start": start,
		"end":   endTs,
	})
}

type languageStat struct {
	Language string `json:"language"`
	Duration int64  `json:"duration"`
	Count    int64  `json:"count"`
}

func (s *Server) handleLanguageStats(c *gin.Context) {
	days := int(parseInt64(c.DefaultQuery("days", "30")))
	if days <= 0 {
		days = 30
	}
	end := parseInt64(c.Query("end"))
	if end == 0 {
		end = time.Now().Unix()
	}
	endTs := endOfDay(end)
	start := endTs - int64(days)*86400
	segments, err := s.Store.CodingSegmentsForLang(c.Query("device_id"), start, endTs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agg := make(map[string]*languageStat, 16)
	for _, seg := range segments {
		lang := category.DetectLanguage(seg.ProcessName, seg.WindowTitle)
		if lang == "" {
			lang = "Other"
		}
		l, ok := agg[lang]
		if !ok {
			l = &languageStat{Language: lang}
			agg[lang] = l
		}
		l.Duration += seg.Duration
		l.Count++
	}
	out := make([]languageStat, 0, len(agg))
	for _, l := range agg {
		out = append(out, *l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Duration > out[j].Duration })
	c.JSON(http.StatusOK, gin.H{
		"languages": out,
		"start":     start,
		"end":       endTs,
	})
}
