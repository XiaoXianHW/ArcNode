package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func (s *Server) mcpTools() []mcpTool {
	dateArg := schemaObject(map[string]interface{}{
		"device_id": schemaString("Optional device ID."),
		"date":      schemaString("YYYY-MM-DD. Defaults to today."),
	}, nil)
	daysArg := schemaObject(map[string]interface{}{
		"device_id": schemaString("Optional device ID."),
		"category":  schemaString("Optional category filter."),
		"days":      schemaInt("Number of days back to include."),
	}, nil)
	return []mcpTool{
		{
			Name:        "list_devices",
			Description: "List devices reporting into the ArcNode gateway.",
			InputSchema: schemaObject(nil, nil),
		},
		{
			Name:        "get_summary",
			Description: "Daily summary: per-category time, top apps, top shortcuts, idle/active split.",
			InputSchema: dateArg,
		},
		{
			Name:        "get_categories",
			Description: "Time spent by category for a given day.",
			InputSchema: dateArg,
		},
		{
			Name:        "get_apps",
			Description: "Top applications by usage time on a given day.",
			InputSchema: schemaObject(map[string]interface{}{
				"device_id": schemaString("Optional device ID."),
				"date":      schemaString("YYYY-MM-DD."),
				"limit":     schemaInt("Maximum number of apps."),
			}, nil),
		},
		{
			Name:        "get_segments",
			Description: "Continuous activity segments for a given day. Use to reconstruct the day's timeline.",
			InputSchema: schemaObject(map[string]interface{}{
				"device_id": schemaString("Optional device ID."),
				"date":      schemaString("YYYY-MM-DD."),
				"category":  schemaString("Optional category filter."),
			}, nil),
		},
		{
			Name:        "get_heatmap",
			Description: "Per-day duration over a range, for GitHub-style heatmaps.",
			InputSchema: daysArg,
		},
		{
			Name:        "get_daily",
			Description: "Per-day duration bars for a category (last N days).",
			InputSchema: daysArg,
		},
		{
			Name:        "get_projects",
			Description: "Top window-title groups, useful for figuring out which projects took most time.",
			InputSchema: schemaObject(map[string]interface{}{
				"device_id": schemaString("Optional device ID."),
				"category":  schemaString("Optional category filter (often 'coding')."),
				"date":      schemaString("YYYY-MM-DD."),
				"limit":     schemaInt("Maximum number of projects."),
			}, nil),
		},
		{
			Name:        "get_languages",
			Description: "Programming language breakdown for the 'coding' category over the last N days.",
			InputSchema: daysArg,
		},
		{
			Name:        "get_hourly",
			Description: "Aggregated activity grid (weekday x hour) over the last N days. Use to find peak hours.",
			InputSchema: daysArg,
		},
		{
			Name:        "get_balance",
			Description: "Daily category balance over the last N days (for stacked area charts).",
			InputSchema: daysArg,
		},
		{
			Name:        "get_rules",
			Description: "Effective classifier rules (builtin + custom keywords).",
			InputSchema: schemaObject(nil, nil),
		},
		{
			Name:        "list_custom_keywords",
			Description: "List all user-defined custom classification keywords.",
			InputSchema: schemaObject(nil, nil),
		},
	}
}

func schemaObject(props map[string]interface{}, required []string) map[string]interface{} {
	obj := map[string]interface{}{"type": "object"}
	if props == nil {
		props = map[string]interface{}{}
	}
	obj["properties"] = props
	if required != nil {
		obj["required"] = required
	}
	return obj
}

func schemaString(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

func schemaInt(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": desc}
}

func (s *Server) handleMCP(c *gin.Context) {
	var req mcpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, mcpResponse{
			JSONRPC: "2.0",
			Error:   &mcpError{Code: -32700, Message: "parse error: " + err.Error()},
		})
		return
	}
	resp := mcpResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "arcnode-gateway",
				"version": "0.2.0",
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{"listChanged": false},
			},
		}
	case "ping":
		resp.Result = map[string]interface{}{}
	case "notifications/initialized", "notifications/cancelled":
		c.Status(http.StatusNoContent)
		return
	case "tools/list":
		resp.Result = map[string]interface{}{"tools": s.mcpTools()}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp.Error = &mcpError{Code: -32602, Message: err.Error()}
			break
		}
		out, err := s.callMCPTool(p.Name, p.Arguments)
		if err != nil {
			resp.Result = map[string]interface{}{
				"isError": true,
				"content": []map[string]interface{}{{"type": "text", "text": err.Error()}},
			}
			break
		}
		payload, _ := json.MarshalIndent(out, "", "  ")
		resp.Result = map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": string(payload)}},
		}
	default:
		resp.Error = &mcpError{Code: -32601, Message: "method not found: " + req.Method}
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) callMCPTool(name string, raw json.RawMessage) (interface{}, error) {
	args := map[string]interface{}{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, err
		}
	}
	str := func(k string) string {
		if v, ok := args[k].(string); ok {
			return v
		}
		return ""
	}
	intArg := func(k string, def int64) int64 {
		switch v := args[k].(type) {
		case float64:
			return int64(v)
		case string:
			var n int64
			fmt.Sscanf(v, "%d", &n)
			if n == 0 {
				return def
			}
			return n
		}
		return def
	}
	dateRangeFor := func(date string) (int64, int64) {
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		t, err := time.ParseInLocation("2006-01-02", date, time.Local)
		if err != nil {
			t = time.Now()
		}
		startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		return startOfDay.Unix(), startOfDay.Add(24 * time.Hour).Unix()
	}
	daysRange := func(def int64) (int64, int64) {
		days := intArg("days", def)
		end := endOfDay(time.Now().Unix())
		start := end - days*86400
		return start, end
	}
	switch name {
	case "list_devices":
		return s.Store.ListDevices()
	case "get_categories":
		start, end := dateRangeFor(str("date"))
		return s.Store.CategoryStats(str("device_id"), start, end)
	case "get_apps":
		start, end := dateRangeFor(str("date"))
		return s.Store.AppStats(str("device_id"), start, end, int(intArg("limit", 20)))
	case "get_segments":
		start, end := dateRangeFor(str("date"))
		return s.Store.QuerySegments(segmentQuery(str("device_id"), str("category"), start, end))
	case "get_summary":
		start, end := dateRangeFor(str("date"))
		return s.summaryFor(str("device_id"), start, end)
	case "get_heatmap":
		start, end := daysRange(365)
		buckets, err := s.Store.DailyStats(str("device_id"), str("category"), start, end)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"days": buckets, "start": start, "end": end}, nil
	case "get_daily":
		start, end := daysRange(30)
		buckets, err := s.Store.DailyStats(str("device_id"), str("category"), start, end)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"days": buckets, "start": start, "end": end}, nil
	case "get_projects":
		start, end := dateRangeFor(str("date"))
		return s.Store.ProjectStats(str("device_id"), str("category"), start, end, int(intArg("limit", 20)))
	case "get_languages":
		start, end := daysRange(30)
		segs, err := s.Store.CodingSegmentsForLang(str("device_id"), start, end)
		if err != nil {
			return nil, err
		}
		return aggregateLanguages(segs), nil
	case "get_hourly":
		start, end := daysRange(30)
		return s.Store.HourlyStats(str("device_id"), str("category"), start, end)
	case "get_balance":
		start, end := daysRange(14)
		return s.Store.DailyCategoryStats(str("device_id"), start, end)
	case "get_rules":
		return s.Classifier.Rules(), nil
	case "list_custom_keywords":
		return s.Store.ListCustomKeywords()
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}
