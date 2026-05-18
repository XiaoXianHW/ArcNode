package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/XiaoXianHW/ArcNode/gateway/storage"
)

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
	DeviceID string         `json:"device_id" binding:"required"`
	Events   []incomingEvent `json:"events" binding:"required"`
}

type incomingEvent struct {
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

	inserted := 0
	for _, ev := range batch.Events {
		evType := normalizeEventType(ev.EventType)
		processName, windowTitle, pid := extractMeta(ev.Metadata)
		category := ev.Category
		if category == "" {
			category = s.Classifier.Classify(processName, windowTitle)
		}
		stored := &storage.Event{
			DeviceID:    batch.DeviceID,
			Timestamp:   ev.Timestamp,
			EventType:   evType,
			Category:    category,
			ProcessName: processName,
			WindowTitle: windowTitle,
			PID:         pid,
			Metadata:    ev.Metadata,
		}
		if _, err := s.Store.InsertEvent(tx, stored); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if evType == "foreground_change" && processName != "" {
			if err := s.Store.UpsertSegment(tx, batch.DeviceID, processName, windowTitle, category,
				ev.Timestamp, s.SegmentGap); err != nil {
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
	c.JSON(http.StatusOK, gin.H{"status": "ok", "inserted": inserted})
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
