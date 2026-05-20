package api

import (
	"sort"
	"strings"
	"time"

	"github.com/XiaoXianHW/ArcNode/gateway/category"
	"github.com/XiaoXianHW/ArcNode/gateway/storage"
)

func segmentQuery(deviceID, cat string, start, end int64) storage.SegmentQuery {
	return storage.SegmentQuery{
		DeviceID: deviceID,
		Start:    start,
		End:      end,
		Category: cat,
		Limit:    0,
		Offset:   0,
	}
}

func (s *Server) summaryFor(deviceID string, start, end int64) (map[string]interface{}, error) {
	cats, err := s.Store.CategoryStats(deviceID, start, end)
	if err != nil {
		return nil, err
	}
	apps, err := s.Store.AppStats(deviceID, start, end, 10)
	if err != nil {
		return nil, err
	}
	shortcuts, err := s.Store.ShortcutStats(deviceID, start, end, 10)
	if err != nil {
		return nil, err
	}
	idle := storage.IdleStat{}
	if deviceID != "" {
		idle, err = s.Store.IdleStats(deviceID, start, end)
		if err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{
		"start":      start,
		"end":        end,
		"categories": cats,
		"top_apps":   apps,
		"shortcuts":  shortcuts,
		"idle":       idle,
	}, nil
}

func aggregateLanguages(segs []storage.CodingSegment) []languageStat {
	agg := map[string]*languageStat{}
	for _, seg := range segs {
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
	return out
}

func (s *Server) flowFor(deviceID string, start, end int64) (map[string]interface{}, error) {
	dailyDur, err := s.Store.DailyStats(deviceID, "", start, end)
	if err != nil {
		return nil, err
	}
	switches, err := s.Store.DailySwitches(deviceID, start, end)
	if err != nil {
		return nil, err
	}
	idleSpans, err := s.Store.IdleSpans(deviceID, start, end)
	if err != nil {
		return nil, err
	}
	focus, err := s.Store.FocusBlocks(deviceID, "", start, end, 1500, 120)
	if err != nil {
		return nil, err
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
	swByDay := map[string]storage.SwitchBucket{}
	for _, sw := range switches {
		swByDay[sw.Date] = sw
	}
	days := make([]flowDay, 0, len(dailyDur))
	for _, b := range dailyDur {
		sw := swByDay[b.Date]
		f := focusByDay[b.Date]
		days = append(days, flowDay{
			Date:         b.Date,
			Active:       b.Duration,
			Idle:         idleByDay[b.Date],
			FocusSeconds: f,
			Switches:     sw.Switches,
			UniqueApps:   sw.Unique,
			Score:        computeFlowScore(b.Duration, f, sw.Switches, sw.Unique),
		})
	}
	return map[string]interface{}{"days": days, "start": start, "end": end}, nil
}

func (s *Server) fileStatsFor(deviceID string, start, end int64, limit int) ([]storage.FileStat, error) {
	segs, err := s.Store.CodingSegmentsForLang(deviceID, start, end)
	if err != nil {
		return nil, err
	}
	agg := map[string]*storage.FileStat{}
	for _, sg := range segs {
		match := fileExtRegex.FindString(sg.WindowTitle)
		if match == "" {
			continue
		}
		lang := category.DetectLanguage(sg.ProcessName, sg.WindowTitle)
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
	return out, nil
}

func (s *Server) videoFor(deviceID string, start, end int64) ([]storage.VideoRow, error) {
	segs, err := s.Store.VideoCandidates(deviceID, start, end)
	if err != nil {
		return nil, err
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
	return out, nil
}

func (s *Server) idleRatioFor(deviceID string, start, end int64) (map[string]interface{}, error) {
	idles, err := s.Store.IdleSpans(deviceID, start, end)
	if err != nil {
		return nil, err
	}
	daily, err := s.Store.DailyStats(deviceID, "", start, end)
	if err != nil {
		return nil, err
	}
	idleByDay := map[string]int64{}
	for _, iv := range idles {
		day := time.Unix(iv[0], 0).In(time.Local).Format("2006-01-02")
		idleByDay[day] += iv[1] - iv[0]
	}
	out := make([]map[string]interface{}, 0, len(daily))
	for _, b := range daily {
		out = append(out, map[string]interface{}{
			"date":   b.Date,
			"active": b.Duration,
			"idle":   idleByDay[b.Date],
		})
	}
	return map[string]interface{}{"days": out, "start": start, "end": end}, nil
}

func (s *Server) gameReportFor(deviceID string, start, end int64) ([]gameReport, error) {
	segs, err := s.Store.QuerySegments(storage.SegmentQuery{DeviceID: deviceID, Start: start, End: end, Category: "gaming"})
	if err != nil {
		return nil, err
	}
	agg := map[string]*gameReport{}
	dayMap := map[string]map[string]bool{}
	for _, sg := range segs {
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
	return out, nil
}

func (s *Server) weeklyReportFor(deviceID string, days int64) (weeklyReport, error) {
	if days <= 0 {
		days = 7
	}
	endTs := endOfDay(time.Now().Unix())
	start := endTs - days*86400
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
	if switches, err := s.Store.DailySwitches(deviceID, start, endTs); err == nil {
		var sw int64
		for _, x := range switches {
			sw += x.Switches
		}
		r.Switches = sw
	}
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
				r.AvgFlowScore = sum / float64(n)
			}
		}
	}
	return r, nil
}
