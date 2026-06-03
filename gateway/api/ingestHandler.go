package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/XiaoXianHW/ArcNode/gateway/storage"
)

// maxBatchEvents caps how many events a single ingest request may carry, so a
// malformed or malicious client cannot exhaust gateway memory.
const maxBatchEvents = 50000

// clockSkewToleranceSecs is how far a client timestamp may lead the server's
// clock before we treat it as skew and fall back to the receive time.
const clockSkewToleranceSecs = 300

// knownEventTypes is the set of accepted normalized event types. Events of any
// other type are rejected (counted as invalid) rather than silently stored.
var knownEventTypes = map[string]bool{
	"foreground_change": true,
	"process_start":     true,
	"process_exit":      true,
	"idle_start":        true,
	"idle_end":          true,
	"keyboard_shortcut": true,
	"system_sample":     true,
}

type initRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	Name       string `json:"name"`
	Platform   string `json:"platform"`
	SystemInfo struct {
		CPUBrand        string `json:"cpu_brand"`
		CPUCores        int    `json:"cpu_cores"`
		TotalMemory     int64  `json:"total_memory"`
		TotalDisk       int64  `json:"total_disk"`
		OSName          string `json:"os_name"`
		OSVersion       string `json:"os_version"`
		Architecture    string `json:"architecture"`
		BootTime        int64  `json:"boot_time"`
		Uptime          int64  `json:"uptime"`
		NetworkUpload   int64  `json:"network_upload"`
		NetworkDownload int64  `json:"network_download"`
	} `json:"system_info"`
}

func (s *Server) handleInit(c *gin.Context) {
	var req initRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d := &storage.Device{
		DeviceID:        req.DeviceID,
		Name:            req.Name,
		Platform:        req.Platform,
		CPUBrand:        req.SystemInfo.CPUBrand,
		CPUCores:        req.SystemInfo.CPUCores,
		TotalMemory:     req.SystemInfo.TotalMemory,
		TotalDisk:       req.SystemInfo.TotalDisk,
		OSName:          req.SystemInfo.OSName,
		OSVersion:       req.SystemInfo.OSVersion,
		Architecture:    req.SystemInfo.Architecture,
		BootTime:        req.SystemInfo.BootTime,
		Uptime:          req.SystemInfo.Uptime,
		NetworkUpload:   req.SystemInfo.NetworkUpload,
		NetworkDownload: req.SystemInfo.NetworkDownload,
	}
	if err := s.Store.UpsertDevice(d); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type eventBatch struct {
	DeviceID string          `json:"device_id" binding:"required"`
	Events   []incomingEvent `json:"events" binding:"required"`
}

type incomingEvent struct {
	EventID   string                 `json:"event_id"`
	DeviceID  string                 `json:"device_id"`
	Timestamp int64                  `json:"timestamp"`
	EventType string                 `json:"event_type"`
	Category  string                 `json:"category"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (s *Server) handleEvents(c *gin.Context) {
	var batch eventBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(batch.Events) > maxBatchEvents {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "batch too large", "max": maxBatchEvents,
		})
		return
	}
	if err := s.Store.TouchDevice(batch.DeviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx, err := s.Store.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().Unix()
	inserted := 0
	duplicates := 0
	invalid := 0
	typeCounts := map[string]int{}
	for _, ev := range batch.Events {
		evType := normalizeEventType(ev.EventType)
		if !knownEventTypes[evType] {
			invalid++
			continue
		}
		ts := sanitizeTimestamp(ev.Timestamp, now)
		typeCounts[evType]++
		processName, windowTitle, pid := extractMeta(ev.Metadata)
		category := ev.Category
		if category == "" {
			category = s.Classifier.Classify(processName, windowTitle)
		}
		stored := &storage.Event{
			EventID:     ev.EventID,
			DeviceID:    batch.DeviceID,
			Timestamp:   ts,
			ReceivedAt:  now,
			EventType:   evType,
			Category:    category,
			ProcessName: processName,
			WindowTitle: windowTitle,
			PID:         pid,
			Metadata:    ev.Metadata,
		}
		wasInserted, err := s.Store.InsertEvent(tx, stored)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !wasInserted {
			// Already stored (re-sent after a retry); skip derived state so
			// segments are not double-counted.
			duplicates++
			continue
		}
		if evType == "foreground_change" && processName != "" {
			if err := s.Store.UpsertSegment(tx, batch.DeviceID, processName, windowTitle, category,
				ts, s.SegmentGap); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, ok := typeCounts["system_sample"]; ok && n > 0 {
		log.Printf("ingest device=%s system_sample=%d total=%d", batch.DeviceID, n, inserted)
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok", "inserted": inserted, "duplicates": duplicates, "invalid": invalid,
	})
}

// sanitizeTimestamp guards against client clock skew. A non-positive timestamp
// (unset) falls back to the server receive time; a timestamp implausibly far in
// the future (clock running ahead) is clamped to the receive time. Otherwise
// the client-provided time is preserved so legitimate backfill from a spooled
// queue keeps its original time.
func sanitizeTimestamp(ts, now int64) int64 {
	if ts <= 0 {
		return now
	}
	if ts > now+clockSkewToleranceSecs {
		return now
	}
	return ts
}

func normalizeEventType(t string) string {
	switch t {
	case "ForegroundChange":
		return "foreground_change"
	case "ProcessStart":
		return "process_start"
	case "ProcessExit":
		return "process_exit"
	case "IdleStart":
		return "idle_start"
	case "IdleEnd":
		return "idle_end"
	case "KeyboardShortcut":
		return "keyboard_shortcut"
	case "SystemSample":
		return "system_sample"
	}
	return t
}

func extractMeta(m map[string]interface{}) (proc, title string, pid int) {
	if m == nil {
		return
	}
	if v, ok := m["process_name"].(string); ok {
		proc = v
	}
	if v, ok := m["window_title"].(string); ok {
		title = v
	}
	if v, ok := m["pid"].(float64); ok {
		pid = int(v)
	}
	return
}
