package storage

import (
	"database/sql"
	"sort"
	"strings"
	"time"
)

type FocusBlock struct {
	Category  string `json:"category"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Duration  int64  `json:"duration"`
	Apps      int    `json:"apps"`
}

func (s *Store) FocusBlocks(deviceID, cat string, start, end, minDuration, maxGap int64) ([]FocusBlock, error) {
	where := []string{"end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if cat != "" {
		where = append(where, "category=?")
		args = append(args, cat)
	}
	q := `SELECT process_name, COALESCE(NULLIF(category,''),'uncategorized'), start_time, end_time
		FROM segments WHERE ` + strings.Join(where, " AND ") + ` ORDER BY start_time ASC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type seg struct {
		proc string
		cat  string
		s, e int64
	}
	var segs []seg
	for rows.Next() {
		var sg seg
		if err := rows.Scan(&sg.proc, &sg.cat, &sg.s, &sg.e); err != nil {
			return nil, err
		}
		segs = append(segs, sg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var blocks []FocusBlock
	for _, sg := range segs {
		if len(blocks) == 0 {
			blocks = append(blocks, FocusBlock{Category: sg.cat, StartTime: sg.s, EndTime: sg.e, Apps: 1})
			continue
		}
		last := &blocks[len(blocks)-1]
		if sg.cat == last.Category && sg.s-last.EndTime <= maxGap {
			if sg.e > last.EndTime {
				last.EndTime = sg.e
			}
			last.Apps++
		} else {
			blocks = append(blocks, FocusBlock{Category: sg.cat, StartTime: sg.s, EndTime: sg.e, Apps: 1})
		}
	}
	out := blocks[:0]
	for _, b := range blocks {
		b.Duration = b.EndTime - b.StartTime
		if b.Duration >= minDuration {
			out = append(out, b)
		}
	}
	return out, nil
}

type SwitchBucket struct {
	Date     string `json:"date"`
	Switches int64  `json:"switches"`
	Unique   int64  `json:"unique_apps"`
}

func (s *Store) DailySwitches(deviceID string, start, end int64) ([]SwitchBucket, error) {
	where := []string{"event_type='foreground_change'", "timestamp>=?", "timestamp<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT strftime('%Y-%m-%d', timestamp, 'unixepoch', 'localtime') AS day,
		COUNT(*) AS sw, COUNT(DISTINCT process_name) AS uq
		FROM events WHERE ` + strings.Join(where, " AND ") + ` GROUP BY day ORDER BY day ASC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SwitchBucket
	for rows.Next() {
		var b SwitchBucket
		if err := rows.Scan(&b.Date, &b.Switches, &b.Unique); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

type HourlySwitch struct {
	Weekday  int   `json:"weekday"`
	Hour     int   `json:"hour"`
	Switches int64 `json:"switches"`
}

func (s *Store) HourlySwitches(deviceID string, start, end int64) ([]HourlySwitch, error) {
	where := []string{"event_type='foreground_change'", "timestamp>=?", "timestamp<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT timestamp FROM events WHERE ` + strings.Join(where, " AND ")
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	grid := make(map[int]int64, 24*7)
	for rows.Next() {
		var ts int64
		if err := rows.Scan(&ts); err != nil {
			return nil, err
		}
		t := timeUnix(ts)
		key := int(t.Weekday())*24 + t.Hour()
		grid[key]++
	}
	out := make([]HourlySwitch, 0, 24*7)
	for w := 0; w < 7; w++ {
		for h := 0; h < 24; h++ {
			out = append(out, HourlySwitch{Weekday: w, Hour: h, Switches: grid[w*24+h]})
		}
	}
	return out, rows.Err()
}

type SessionBucket struct {
	Bucket   string `json:"bucket"`
	Min      int64  `json:"min"`
	Max      int64  `json:"max"`
	Count    int64  `json:"count"`
	Duration int64  `json:"duration"`
}

var sessionBuckets = []struct {
	label    string
	min, max int64
}{
	{"<1m", 0, 60},
	{"1-5m", 60, 300},
	{"5-15m", 300, 900},
	{"15-30m", 900, 1800},
	{"30-60m", 1800, 3600},
	{"1-2h", 3600, 7200},
	{"2-4h", 7200, 14400},
	{"4h+", 14400, 1 << 30},
}

func (s *Store) SessionDistribution(deviceID, cat string, start, end int64) ([]SessionBucket, error) {
	where := []string{"end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if cat != "" {
		where = append(where, "category=?")
		args = append(args, cat)
	}
	q := `SELECT duration FROM segments WHERE ` + strings.Join(where, " AND ")
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SessionBucket, len(sessionBuckets))
	for i, b := range sessionBuckets {
		out[i] = SessionBucket{Bucket: b.label, Min: b.min, Max: b.max}
	}
	for rows.Next() {
		var d int64
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		for i, b := range sessionBuckets {
			if d >= b.min && d < b.max {
				out[i].Count++
				out[i].Duration += d
				break
			}
		}
	}
	return out, rows.Err()
}

type FileStat struct {
	File     string `json:"file"`
	Language string `json:"language"`
	Duration int64  `json:"duration"`
	Count    int64  `json:"count"`
}

type ProjectDailyRow struct {
	Date     string `json:"date"`
	Project  string `json:"project"`
	Duration int64  `json:"duration"`
}

func (s *Store) ProjectDaily(deviceID, cat string, start, end int64) ([]ProjectDailyRow, error) {
	where := []string{"end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if cat != "" {
		where = append(where, "category=?")
		args = append(args, cat)
	}
	q := `SELECT strftime('%Y-%m-%d', start_time, 'unixepoch', 'localtime') AS day,
		IFNULL(window_title,''), SUM(duration)
		FROM segments WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY day, window_title ORDER BY day ASC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectDailyRow
	for rows.Next() {
		var r ProjectDailyRow
		if err := rows.Scan(&r.Date, &r.Project, &r.Duration); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type AppPair struct {
	A     string `json:"a"`
	B     string `json:"b"`
	Count int64  `json:"count"`
}

func (s *Store) AppPairs(deviceID string, start, end int64, limit int) ([]AppPair, error) {
	where := []string{"event_type='foreground_change'", "timestamp>=?", "timestamp<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT process_name FROM events WHERE ` + strings.Join(where, " AND ") + ` ORDER BY timestamp ASC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pairs := make(map[string]int64, 64)
	var prev string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		if prev != "" && prev != p {
			a, b := prev, p
			if a > b {
				a, b = b, a
			}
			pairs[a+"\x1f"+b]++
		}
		prev = p
	}
	out := make([]AppPair, 0, len(pairs))
	for k, v := range pairs {
		parts := strings.SplitN(k, "\x1f", 2)
		if len(parts) != 2 {
			continue
		}
		out = append(out, AppPair{A: parts[0], B: parts[1], Count: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, rows.Err()
}

type DailySedentary struct {
	Date           string `json:"date"`
	LongestStretch int64  `json:"longest_stretch"`
	StretchesOver  int    `json:"stretches_over_threshold"`
	TotalActive    int64  `json:"total_active"`
	TotalIdle      int64  `json:"total_idle"`
}

// IdleSpans returns idle (idle_start..idle_end) pairs and treats unmatched
// idle_start as open intervals up to `end`.
func (s *Store) IdleSpans(deviceID string, start, end int64) ([][2]int64, error) {
	where := []string{"event_type IN ('idle_start','idle_end')", "timestamp BETWEEN ? AND ?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT timestamp, event_type FROM events WHERE ` + strings.Join(where, " AND ") + ` ORDER BY timestamp ASC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var spans [][2]int64
	var openStart int64
	for rows.Next() {
		var ts int64
		var et string
		if err := rows.Scan(&ts, &et); err != nil {
			return nil, err
		}
		if et == "idle_start" {
			openStart = ts
		} else if openStart > 0 {
			spans = append(spans, [2]int64{openStart, ts})
			openStart = 0
		}
	}
	if openStart > 0 {
		spans = append(spans, [2]int64{openStart, end})
	}
	return spans, rows.Err()
}

func (s *Store) DailySedentary(deviceID string, start, end, threshold int64) ([]DailySedentary, error) {
	idles, err := s.IdleSpans(deviceID, start, end)
	if err != nil {
		return nil, err
	}
	type segLite struct {
		s, e int64
	}
	where := []string{"end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT start_time, end_time FROM segments WHERE ` + strings.Join(where, " AND ") + ` ORDER BY start_time ASC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var segs []segLite
	for rows.Next() {
		var sg segLite
		if err := rows.Scan(&sg.s, &sg.e); err != nil {
			return nil, err
		}
		segs = append(segs, sg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	loc := time.Local
	dayBuckets := map[string]*DailySedentary{}
	prevEnd := int64(0)
	prevDay := ""
	for _, sg := range segs {
		day := time.Unix(sg.s, 0).In(loc).Format("2006-01-02")
		b, ok := dayBuckets[day]
		if !ok {
			b = &DailySedentary{Date: day}
			dayBuckets[day] = b
		}
		b.TotalActive += sg.e - sg.s
		if prevEnd > 0 && day == prevDay {
			gap := sg.s - prevEnd
			if gap > 0 {
				if gap >= threshold {
					b.StretchesOver++
				}
				if gap > b.LongestStretch {
					b.LongestStretch = gap
				}
			}
		}
		prevEnd = sg.e
		prevDay = day
	}
	for _, iv := range idles {
		day := time.Unix(iv[0], 0).In(loc).Format("2006-01-02")
		b, ok := dayBuckets[day]
		if !ok {
			b = &DailySedentary{Date: day}
			dayBuckets[day] = b
		}
		b.TotalIdle += iv[1] - iv[0]
	}
	out := make([]DailySedentary, 0, len(dayBuckets))
	for _, b := range dayBuckets {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out, nil
}

type UncategorizedRow struct {
	ProcessName  string `json:"process_name"`
	SampleTitle  string `json:"sample_title"`
	Duration     int64  `json:"duration"`
	Count        int64  `json:"count"`
}

func (s *Store) UncategorizedTop(deviceID string, start, end int64, limit int) ([]UncategorizedRow, error) {
	where := []string{"(category IS NULL OR category='' OR category='uncategorized')", "end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT process_name,
		IFNULL((SELECT window_title FROM segments WHERE process_name=t.process_name AND window_title IS NOT NULL LIMIT 1), '') AS sample,
		SUM(duration), COUNT(*)
		FROM segments t WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY process_name ORDER BY SUM(duration) DESC`
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UncategorizedRow
	for rows.Next() {
		var r UncategorizedRow
		var sample sql.NullString
		if err := rows.Scan(&r.ProcessName, &sample, &r.Duration, &r.Count); err != nil {
			return nil, err
		}
		r.SampleTitle = sample.String
		out = append(out, r)
	}
	return out, rows.Err()
}

type LiveStatus struct {
	DeviceID     string `json:"device_id"`
	Online       bool   `json:"online"`
	Idle         bool   `json:"idle"`
	LastEventAt  int64  `json:"last_event_at"`
	LastSegment  *Segment `json:"last_segment,omitempty"`
	RecentApps   []AppStat `json:"recent_apps,omitempty"`
	IdleSince    int64  `json:"idle_since,omitempty"`
}

func (s *Store) LiveStatus(deviceID string, onlineWindow int64, recentWindow int64) (LiveStatus, error) {
	now := time.Now().Unix()
	out := LiveStatus{DeviceID: deviceID}
	row := s.DB.QueryRow(`SELECT MAX(timestamp) FROM events WHERE device_id=?`, deviceID)
	var ts sql.NullInt64
	if err := row.Scan(&ts); err == nil && ts.Valid {
		out.LastEventAt = ts.Int64
		out.Online = now-ts.Int64 <= onlineWindow
	}

	row = s.DB.QueryRow(`SELECT timestamp, event_type FROM events WHERE device_id=? AND event_type IN ('idle_start','idle_end') ORDER BY timestamp DESC LIMIT 1`, deviceID)
	var lastTs sql.NullInt64
	var lastType sql.NullString
	if err := row.Scan(&lastTs, &lastType); err == nil && lastType.Valid && lastType.String == "idle_start" {
		out.Idle = true
		out.IdleSince = lastTs.Int64
	}

	last := s.DB.QueryRow(`SELECT id, device_id, process_name, IFNULL(window_title,''), IFNULL(category,''), start_time, end_time, duration FROM segments WHERE device_id=? ORDER BY end_time DESC LIMIT 1`, deviceID)
	var seg Segment
	if err := last.Scan(&seg.ID, &seg.DeviceID, &seg.ProcessName, &seg.WindowTitle, &seg.Category, &seg.StartTime, &seg.EndTime, &seg.Duration); err == nil {
		out.LastSegment = &seg
	}

	apps, err := s.AppStats(deviceID, now-recentWindow, now, 5)
	if err == nil {
		out.RecentApps = apps
	}
	return out, nil
}

type SystemSample struct {
	Timestamp int64   `json:"timestamp"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	BatteryPct float64 `json:"battery_pct,omitempty"`
}

func (s *Store) SystemSamples(deviceID string, start, end int64) ([]SystemSample, error) {
	where := []string{"event_type='system_sample'", "timestamp>=?", "timestamp<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT timestamp,
		IFNULL(CAST(json_extract(metadata,'$.cpu') AS REAL),0),
		IFNULL(CAST(json_extract(metadata,'$.memory') AS REAL),0),
		IFNULL(CAST(json_extract(metadata,'$.battery_pct') AS REAL),0)
		FROM events WHERE ` + strings.Join(where, " AND ") + ` ORDER BY timestamp ASC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SystemSample
	for rows.Next() {
		var sm SystemSample
		if err := rows.Scan(&sm.Timestamp, &sm.CPU, &sm.Memory, &sm.BatteryPct); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

type VideoRow struct {
	Platform string `json:"platform"`
	Duration int64  `json:"duration"`
	Count    int64  `json:"count"`
}

// VideoSegments returns all candidate segments from which the API layer can
// derive video/streaming platform usage via title heuristics.
func (s *Store) VideoCandidates(deviceID string, start, end int64) ([]Segment, error) {
	where := []string{"end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT id, device_id, process_name, IFNULL(window_title,''), IFNULL(category,''), start_time, end_time, duration
		FROM segments WHERE ` + strings.Join(where, " AND ")
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Segment
	for rows.Next() {
		var sg Segment
		if err := rows.Scan(&sg.ID, &sg.DeviceID, &sg.ProcessName, &sg.WindowTitle, &sg.Category, &sg.StartTime, &sg.EndTime, &sg.Duration); err != nil {
			return nil, err
		}
		out = append(out, sg)
	}
	return out, rows.Err()
}
