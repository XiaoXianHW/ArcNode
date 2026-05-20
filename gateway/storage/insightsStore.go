package storage

import "strings"

type HourlyBucket struct {
	Weekday  int   `json:"weekday"`
	Hour     int   `json:"hour"`
	Duration int64 `json:"duration"`
	Count    int64 `json:"count"`
}

// HourlyStats returns activity duration per (weekday, hour) bucket. The
// distribution is computed by clipping each segment to the requested window
// then projecting onto hour cells.
func (s *Store) HourlyStats(deviceID, category string, start, end int64) ([]HourlyBucket, error) {
	where := []string{"end_time>?", "start_time<?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if category != "" {
		where = append(where, "category=?")
		args = append(args, category)
	}
	sqlStr := `SELECT start_time, end_time FROM segments WHERE ` + strings.Join(where, " AND ")
	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grid := make(map[int]int64, 24*7)
	counts := make(map[int]int64, 24*7)
	for rows.Next() {
		var s0, s1 int64
		if err := rows.Scan(&s0, &s1); err != nil {
			return nil, err
		}
		if s0 < start {
			s0 = start
		}
		if s1 > end {
			s1 = end
		}
		if s1 <= s0 {
			continue
		}
		distribute(s0, s1, grid, counts)
	}
	out := make([]HourlyBucket, 0, 24*7)
	for w := 0; w < 7; w++ {
		for h := 0; h < 24; h++ {
			key := w*24 + h
			out = append(out, HourlyBucket{
				Weekday: w, Hour: h, Duration: grid[key], Count: counts[key],
			})
		}
	}
	return out, rows.Err()
}

func distribute(s0, s1 int64, grid map[int]int64, counts map[int]int64) {
	const hour = int64(3600)
	for cur := s0; cur < s1; {
		w, h := weekdayHour(cur)
		boundary := cur + (hour - cur%hour)
		if boundary > s1 {
			boundary = s1
		}
		key := w*24 + h
		grid[key] += boundary - cur
		counts[key]++
		cur = boundary
	}
}

func weekdayHour(ts int64) (int, int) {
	// Use UTC-anchored local time: SQLite-side queries use 'localtime', match here.
	// rusqlite/golang both report timezone-naive seconds; convert via builtin.
	t := timeUnix(ts)
	wd := int(t.Weekday())
	return wd, t.Hour()
}

type DailyCategoryRow struct {
	Date     string `json:"date"`
	Category string `json:"category"`
	Duration int64  `json:"duration"`
}

func (s *Store) DailyCategoryStats(deviceID string, start, end int64) ([]DailyCategoryRow, error) {
	where := []string{"end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	sqlStr := `SELECT strftime('%Y-%m-%d', start_time, 'unixepoch', 'localtime') AS day,
		COALESCE(NULLIF(category,''),'uncategorized') AS cat,
		SUM(duration)
		FROM segments WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY day, cat ORDER BY day ASC, SUM(duration) DESC`
	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyCategoryRow
	for rows.Next() {
		var r DailyCategoryRow
		if err := rows.Scan(&r.Date, &r.Category, &r.Duration); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type CodingSegment struct {
	ProcessName string
	WindowTitle string
	Duration    int64
}

func (s *Store) CodingSegmentsForLang(deviceID string, start, end int64) ([]CodingSegment, error) {
	where := []string{"category='coding'", "end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	sqlStr := `SELECT process_name, IFNULL(window_title,''), SUM(duration) FROM segments
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY process_name, window_title`
	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CodingSegment
	for rows.Next() {
		var c CodingSegment
		if err := rows.Scan(&c.ProcessName, &c.WindowTitle, &c.Duration); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
