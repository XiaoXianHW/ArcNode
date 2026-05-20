package api

import (
	"sort"

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
