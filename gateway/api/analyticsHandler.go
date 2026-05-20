package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/XiaoXianHW/ArcNode/gateway/category"
	"github.com/XiaoXianHW/ArcNode/gateway/storage"
)

// browserProcessTokens / terminalProcessTokens are used as a defensive
// filter on top of the new classifier so legacy mis-classified segments
// (recorded before the v3 process/title split) don't leak into the game
// or wellness reports.
var browserProcessTokens = []string{
	"chrome", "chromium", "msedge", "edge", "firefox", "safari", "brave",
	"opera", "vivaldi", "arc", "tor browser", "torbrowser", "thorium",
	"sidekick", "orion", "360se", "360chrome", "qqbrowser", "ucbrowser",
	"sogouexplorer", "zen browser",
}

var terminalProcessTokens = []string{
	"windowsterminal", "wt.exe", "powershell", "pwsh", "cmd.exe", "conhost",
	"iterm", "wezterm", "alacritty", "kitty", "warp", "hyper", "tabby",
	"konsole", "xterm", "rxvt", "termite", "ghostty", "rio.exe", "tilix",
	"qterminal", "deepin-terminal",
}

func isBrowserProcess(p string) bool {
	p = strings.ToLower(p)
	for _, k := range browserProcessTokens {
		if strings.Contains(p, k) {
			return true
		}
	}
	return false
}

func isTerminalProcess(p string) bool {
	p = strings.ToLower(p)
	for _, k := range terminalProcessTokens {
		if strings.Contains(p, k) {
			return true
		}
	}
	return false
}

func windowedRange(c *gin.Context, defaultDays int64) (int64, int64) {
	days := parseInt64(c.DefaultQuery("days", fmt.Sprintf("%d", defaultDays)))
	if days <= 0 {
		days = defaultDays
	}
	end := parseInt64(c.Query("end"))
	if end == 0 {
		end = time.Now().Unix()
	}
	endTs := endOfDay(end)
	start := endTs - days*86400
	return start, endTs
}

func (s *Server) handleFocusBlocks(c *gin.Context) {
	start, end := windowedRange(c, 7)
	minDur := parseInt64(c.DefaultQuery("min_duration", "1500")) // 25min
	maxGap := parseInt64(c.DefaultQuery("max_gap", "120"))
	blocks, err := s.Store.FocusBlocks(c.Query("device_id"), c.Query("category"), start, end, minDur, maxGap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if blocks == nil {
		blocks = []storage.FocusBlock{}
	}
	// summary
	var total int64
	var longest int64
	dayMap := map[string]int64{}
	for _, b := range blocks {
		total += b.Duration
		if b.Duration > longest {
			longest = b.Duration
		}
		day := time.Unix(b.StartTime, 0).In(time.Local).Format("2006-01-02")
		dayMap[day] += b.Duration
	}
	dailyKeys := make([]string, 0, len(dayMap))
	for k := range dayMap {
		dailyKeys = append(dailyKeys, k)
	}
	sort.Strings(dailyKeys)
	daily := make([]gin.H, 0, len(dailyKeys))
	for _, k := range dailyKeys {
		daily = append(daily, gin.H{"date": k, "duration": dayMap[k]})
	}
	c.JSON(http.StatusOK, gin.H{
		"blocks":          blocks,
		"start":           start,
		"end":             end,
		"total_focus":     total,
		"longest":         longest,
		"daily":           daily,
		"min_duration":    minDur,
		"max_gap_seconds": maxGap,
	})
}

func (s *Server) handleSwitches(c *gin.Context) {
	start, end := windowedRange(c, 14)
	daily, err := s.Store.DailySwitches(c.Query("device_id"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hourly, err := s.Store.HourlySwitches(c.Query("device_id"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if daily == nil {
		daily = []storage.SwitchBucket{}
	}
	c.JSON(http.StatusOK, gin.H{
		"daily":  daily,
		"hourly": hourly,
		"start":  start,
		"end":    end,
	})
}

type flowDay struct {
	Date         string  `json:"date"`
	Active       int64   `json:"active_seconds"`
	Idle         int64   `json:"idle_seconds"`
	FocusSeconds int64   `json:"focus_seconds"`
	Switches     int64   `json:"switches"`
	UniqueApps   int64   `json:"unique_apps"`
	Score        float64 `json:"score"`
}

func (s *Server) handleFlow(c *gin.Context) {
	start, end := windowedRange(c, 14)
	deviceID := c.Query("device_id")
	dailyDur, err := s.Store.DailyStats(deviceID, "", start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	switches, err := s.Store.DailySwitches(deviceID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	idleSpans, err := s.Store.IdleSpans(deviceID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	focus, err := s.Store.FocusBlocks(deviceID, "", start, end, 1500, 120)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	idleByDay := map[string]int64{}
	for _, iv := range idleSpans {
		day := time.Unix(iv[0], 0).In(time.Local).Format("2006-01-02")
		idleByDay[day] += iv[1] - iv[0]
	}
	focusByDay := map[string]int64{}
	for _, b := range focus {
		day := time.Unix(b.StartTime, 0).In(time.Local).Format("2006-01-02")
		focusByDay[day] += b.Duration
	}
	switchByDay := map[string]storage.SwitchBucket{}
	for _, sw := range switches {
		switchByDay[sw.Date] = sw
	}

	days := make([]flowDay, 0, len(dailyDur))
	for _, b := range dailyDur {
		sw := switchByDay[b.Date]
		f := focusByDay[b.Date]
		idle := idleByDay[b.Date]
		fd := flowDay{
			Date:         b.Date,
			Active:       b.Duration,
			Idle:         idle,
			FocusSeconds: f,
			Switches:     sw.Switches,
			UniqueApps:   sw.Unique,
		}
		fd.Score = computeFlowScore(b.Duration, f, sw.Switches, sw.Unique)
		days = append(days, fd)
	}
	c.JSON(http.StatusOK, gin.H{
		"days":  days,
		"start": start,
		"end":   end,
	})
}

func computeFlowScore(active, focus, switches, unique int64) float64 {
	if active <= 0 {
		return 0
	}
	focusRatio := float64(focus) / float64(active)
	if focusRatio > 1 {
		focusRatio = 1
	}
	// switches-per-hour normalized: 0 switches → 1, 60+ → 0
	swRate := float64(switches) / (float64(active) / 3600.0)
	swScore := 1 - (swRate / 60.0)
	if swScore < 0 {
		swScore = 0
	}
	// unique app spread: 5 apps ideal, 25+ very bad
	uniqScore := 1 - float64(unique-5)/20.0
	if uniqScore > 1 {
		uniqScore = 1
	}
	if uniqScore < 0 {
		uniqScore = 0
	}
	score := 0.5*focusRatio + 0.3*swScore + 0.2*uniqScore
	return clamp01(score) * 100
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func (s *Server) handleSessions(c *gin.Context) {
	start, end := windowedRange(c, 14)
	buckets, err := s.Store.SessionDistribution(c.Query("device_id"), c.Query("category"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"buckets": buckets, "start": start, "end": end})
}

var fileExtRegex = regexp.MustCompile(`([\w\-.]+\.[A-Za-z0-9]{1,8})\b`)

func (s *Server) handleFileStats(c *gin.Context) {
	start, end := windowedRange(c, 30)
	deviceID := c.Query("device_id")
	limit := int(parseInt64(c.DefaultQuery("limit", "30")))
	segments, err := s.Store.CodingSegmentsForLang(deviceID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agg := map[string]*storage.FileStat{}
	for _, sg := range segments {
		title := sg.WindowTitle
		if title == "" {
			continue
		}
		match := fileExtRegex.FindString(title)
		if match == "" {
			continue
		}
		lang := category.DetectLanguage(sg.ProcessName, title)
		f, ok := agg[match]
		if !ok {
			f = &storage.FileStat{File: match, Language: lang}
			agg[match] = f
		}
		f.Duration += sg.Duration
		f.Count++
	}
	out := make([]storage.FileStat, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Duration > out[j].Duration })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"files": out, "start": start, "end": end})
}

func (s *Server) handleProjectDaily(c *gin.Context) {
	start, end := windowedRange(c, 30)
	rows, err := s.Store.ProjectDaily(c.Query("device_id"), c.Query("category"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []storage.ProjectDailyRow{}
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows, "start": start, "end": end})
}

func (s *Server) handleAppPairs(c *gin.Context) {
	start, end := windowedRange(c, 14)
	limit := int(parseInt64(c.DefaultQuery("limit", "30")))
	pairs, err := s.Store.AppPairs(c.Query("device_id"), start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if pairs == nil {
		pairs = []storage.AppPair{}
	}
	c.JSON(http.StatusOK, gin.H{"pairs": pairs, "start": start, "end": end})
}

var videoPlatforms = map[string][]string{
	"YouTube":   {"youtube", "youtu.be"},
	"Bilibili":  {"bilibili", "哔哩哔哩", "B站"},
	"Twitch":    {"twitch.tv"},
	"Netflix":   {"netflix"},
	"Disney+":   {"disney+", "disneyplus"},
	"Prime":     {"prime video"},
	"HBO":       {"hbo max", "max."},
	"iQIYI":     {"iqiyi", "爱奇艺"},
	"Tencent":   {"v.qq.com", "腾讯视频"},
	"Youku":     {"youku", "优酷"},
	"AbemaTV":   {"abema"},
	"Niconico":  {"nicovideo", "niconico"},
	"VLC":       {"vlc"},
	"MPV":       {"mpv"},
	"PotPlayer": {"potplayer"},
	"IINA":      {"iina"},
}

func (s *Server) handleVideoStats(c *gin.Context) {
	start, end := windowedRange(c, 30)
	segs, err := s.Store.VideoCandidates(c.Query("device_id"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agg := map[string]*storage.VideoRow{}
	for _, sg := range segs {
		hay := strings.ToLower(sg.WindowTitle + " " + sg.ProcessName)
		for platform, keys := range videoPlatforms {
			matched := false
			for _, k := range keys {
				if strings.Contains(hay, strings.ToLower(k)) {
					matched = true
					break
				}
			}
			if matched {
				r, ok := agg[platform]
				if !ok {
					r = &storage.VideoRow{Platform: platform}
					agg[platform] = r
				}
				r.Duration += sg.Duration
				r.Count++
				break
			}
		}
	}
	out := make([]storage.VideoRow, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Duration > out[j].Duration })
	c.JSON(http.StatusOK, gin.H{"platforms": out, "start": start, "end": end})
}

func (s *Server) handleIdleRatio(c *gin.Context) {
	start, end := windowedRange(c, 14)
	deviceID := c.Query("device_id")
	idles, err := s.Store.IdleSpans(deviceID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	daily, err := s.Store.DailyStats(deviceID, "", start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	idleByDay := map[string]int64{}
	for _, iv := range idles {
		day := time.Unix(iv[0], 0).In(time.Local).Format("2006-01-02")
		idleByDay[day] += iv[1] - iv[0]
	}
	out := make([]gin.H, 0, len(daily))
	for _, b := range daily {
		out = append(out, gin.H{
			"date":   b.Date,
			"active": b.Duration,
			"idle":   idleByDay[b.Date],
		})
	}
	c.JSON(http.StatusOK, gin.H{"days": out, "start": start, "end": end})
}

func (s *Server) handleSedentary(c *gin.Context) {
	start, end := windowedRange(c, 14)
	threshold := parseInt64(c.DefaultQuery("threshold", "3600"))
	rows, err := s.Store.DailySedentary(c.Query("device_id"), start, end, threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []storage.DailySedentary{}
	}
	c.JSON(http.StatusOK, gin.H{"days": rows, "threshold": threshold, "start": start, "end": end})
}

func (s *Server) handleSuggestions(c *gin.Context) {
	start, end := windowedRange(c, 14)
	limit := int(parseInt64(c.DefaultQuery("limit", "20")))
	rows, err := s.Store.UncategorizedTop(c.Query("device_id"), start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []storage.UncategorizedRow{}
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "start": start, "end": end})
}

func (s *Server) handleSystemSamples(c *gin.Context) {
	start, end := windowedRange(c, 1)
	samples, err := s.Store.SystemSamples(c.Query("device_id"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if samples == nil {
		samples = []storage.SystemSample{}
	}
	c.JSON(http.StatusOK, gin.H{"samples": samples, "start": start, "end": end})
}

func (s *Server) handleLive(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		deviceID = c.Query("device_id")
	}
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}
	live, err := s.Store.LiveStatus(deviceID, 120, 24*3600)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, live)
}

func (s *Server) handleExportSegmentsCSV(c *gin.Context) {
	start, end := dateRange(c)
	segments, err := s.Store.QuerySegments(storage.SegmentQuery{
		DeviceID: c.Query("device_id"),
		Start:    start,
		End:      end,
		Category: c.Query("category"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="segments.csv"`)
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"id", "device_id", "process_name", "window_title", "category", "start_time", "end_time", "duration"})
	for _, sg := range segments {
		_ = w.Write([]string{
			fmt.Sprintf("%d", sg.ID),
			sg.DeviceID, sg.ProcessName, sg.WindowTitle, sg.Category,
			fmt.Sprintf("%d", sg.StartTime),
			fmt.Sprintf("%d", sg.EndTime),
			fmt.Sprintf("%d", sg.Duration),
		})
	}
	w.Flush()
}

func (s *Server) handleExportEventsJSON(c *gin.Context) {
	limit := int(parseInt64(c.DefaultQuery("limit", "10000")))
	q := storage.EventQuery{
		DeviceID:  c.Query("device_id"),
		Start:     parseInt64(c.Query("start")),
		End:       parseInt64(c.Query("end")),
		EventType: c.Query("type"),
		Limit:     limit,
	}
	events, err := s.Store.QueryEvents(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if events == nil {
		events = []storage.Event{}
	}
	c.Header("Content-Disposition", `attachment; filename="events.json"`)
	c.JSON(http.StatusOK, gin.H{"events": events})
}

type weeklyReport struct {
	Start             int64                `json:"start"`
	End               int64                `json:"end"`
	TotalActive       int64                `json:"total_active"`
	TotalFocus        int64                `json:"total_focus"`
	TopCategories     []storage.CategoryStat `json:"top_categories"`
	TopApps           []storage.AppStat    `json:"top_apps"`
	TopLanguages      []languageStat       `json:"top_languages"`
	TopGames          []storage.AppStat    `json:"top_games"`
	AvgFlowScore      float64              `json:"avg_flow_score"`
	BestDay           string               `json:"best_day"`
	BestDayDuration   int64                `json:"best_day_duration"`
	LongestFocus      int64                `json:"longest_focus"`
	BusiestHour       int                  `json:"busiest_hour"`
	BusiestWeekday    int                  `json:"busiest_weekday"`
	Switches          int64                `json:"context_switches"`
}

func (s *Server) handleWeeklyReport(c *gin.Context) {
	days := parseInt64(c.DefaultQuery("days", "7"))
	if days <= 0 {
		days = 7
	}
	endTs := endOfDay(time.Now().Unix())
	start := endTs - days*86400
	deviceID := c.Query("device_id")
	r := weeklyReport{Start: start, End: endTs}
	if cats, err := s.Store.CategoryStats(deviceID, start, endTs); err == nil {
		r.TopCategories = cats
		for _, ca := range cats {
			r.TotalActive += ca.Duration
		}
	}
	if apps, err := s.Store.AppStats(deviceID, start, endTs, 10); err == nil {
		r.TopApps = apps
	}
	if focus, err := s.Store.FocusBlocks(deviceID, "", start, endTs, 1500, 120); err == nil {
		for _, b := range focus {
			r.TotalFocus += b.Duration
			if b.Duration > r.LongestFocus {
				r.LongestFocus = b.Duration
			}
		}
	}
	if daily, err := s.Store.DailyStats(deviceID, "", start, endTs); err == nil {
		for _, b := range daily {
			if b.Duration > r.BestDayDuration {
				r.BestDayDuration = b.Duration
				r.BestDay = b.Date
			}
		}
	}
	if hourly, err := s.Store.HourlyStats(deviceID, "", start, endTs); err == nil {
		var maxDur int64
		hWeekday := map[int]int64{}
		for _, b := range hourly {
			if b.Duration > maxDur {
				maxDur = b.Duration
				r.BusiestHour = b.Hour
			}
			hWeekday[b.Weekday] += b.Duration
		}
		var maxWd int64
		for w, d := range hWeekday {
			if d > maxWd {
				maxWd = d
				r.BusiestWeekday = w
			}
		}
	}
	// languages
	if segs, err := s.Store.CodingSegmentsForLang(deviceID, start, endTs); err == nil {
		la := map[string]*languageStat{}
		for _, sg := range segs {
			lang := category.DetectLanguage(sg.ProcessName, sg.WindowTitle)
			if lang == "" {
				continue
			}
			l, ok := la[lang]
			if !ok {
				l = &languageStat{Language: lang}
				la[lang] = l
			}
			l.Duration += sg.Duration
			l.Count++
		}
		all := make([]languageStat, 0, len(la))
		for _, l := range la {
			all = append(all, *l)
		}
		sort.Slice(all, func(i, j int) bool { return all[i].Duration > all[j].Duration })
		if len(all) > 5 {
			all = all[:5]
		}
		r.TopLanguages = all
	}
	// games
	if games, err := s.Store.AppStats(deviceID, start, endTs, 30); err == nil {
		filtered := make([]storage.AppStat, 0, 5)
		for _, a := range games {
			if a.Category == "gaming" {
				filtered = append(filtered, a)
				if len(filtered) >= 5 {
					break
				}
			}
		}
		r.TopGames = filtered
	}
	// flow average
	if switches, err := s.Store.DailySwitches(deviceID, start, endTs); err == nil {
		var sw int64
		for _, x := range switches {
			sw += x.Switches
		}
		r.Switches = sw
	}
	// quick flow avg
	var avg float64
	if daily, err := s.Store.DailyStats(deviceID, "", start, endTs); err == nil {
		if switches, err := s.Store.DailySwitches(deviceID, start, endTs); err == nil {
			swMap := map[string]storage.SwitchBucket{}
			for _, x := range switches {
				swMap[x.Date] = x
			}
			focusByDay := map[string]int64{}
			if focus, err := s.Store.FocusBlocks(deviceID, "", start, endTs, 1500, 120); err == nil {
				for _, b := range focus {
					day := time.Unix(b.StartTime, 0).In(time.Local).Format("2006-01-02")
					focusByDay[day] += b.Duration
				}
			}
			var sum float64
			var n int
			for _, b := range daily {
				sw := swMap[b.Date]
				score := computeFlowScore(b.Duration, focusByDay[b.Date], sw.Switches, sw.Unique)
				sum += score
				n++
			}
			if n > 0 {
				avg = sum / float64(n)
			}
		}
	}
	r.AvgFlowScore = avg
	c.JSON(http.StatusOK, r)
}

type gameReport struct {
	ProcessName    string  `json:"process_name"`
	Title          string  `json:"title"`
	TotalDuration  int64   `json:"total_duration"`
	Sessions       int64   `json:"sessions"`
	AvgSession     float64 `json:"avg_session"`
	MaxSession     int64   `json:"max_session"`
	FirstPlayed    int64   `json:"first_played"`
	LastPlayed     int64   `json:"last_played"`
	UniqueDays     int     `json:"unique_days"`
}

func (s *Server) handleGameReport(c *gin.Context) {
	start, end := windowedRange(c, 365)
	deviceID := c.Query("device_id")
	segs, err := s.Store.QuerySegments(storage.SegmentQuery{
		DeviceID: deviceID,
		Start:    start,
		End:      end,
		Category: "gaming",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agg := map[string]*gameReport{}
	dayMap := map[string]map[string]bool{}
	for _, sg := range segs {
		if isBrowserProcess(sg.ProcessName) || isTerminalProcess(sg.ProcessName) {
			continue
		}
		key := sg.ProcessName
		r, ok := agg[key]
		if !ok {
			r = &gameReport{ProcessName: key, Title: sg.ProcessName, FirstPlayed: sg.StartTime, LastPlayed: sg.EndTime}
			agg[key] = r
			dayMap[key] = map[string]bool{}
		}
		r.TotalDuration += sg.Duration
		r.Sessions++
		if sg.Duration > r.MaxSession {
			r.MaxSession = sg.Duration
		}
		if sg.StartTime < r.FirstPlayed {
			r.FirstPlayed = sg.StartTime
		}
		if sg.EndTime > r.LastPlayed {
			r.LastPlayed = sg.EndTime
		}
		day := time.Unix(sg.StartTime, 0).In(time.Local).Format("2006-01-02")
		dayMap[key][day] = true
	}
	out := make([]gameReport, 0, len(agg))
	for k, r := range agg {
		if r.Sessions > 0 {
			r.AvgSession = float64(r.TotalDuration) / float64(r.Sessions)
		}
		r.UniqueDays = len(dayMap[k])
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalDuration > out[j].TotalDuration })
	c.JSON(http.StatusOK, gin.H{"games": out, "start": start, "end": end})
}

func (s *Server) handleEventCounts(c *gin.Context) {
	deviceID := c.Query("device_id")
	start, end := windowedRange(c, 7)
	counts, err := s.Store.EventTypeCounts(deviceID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"counts": counts, "start": start, "end": end})
}
