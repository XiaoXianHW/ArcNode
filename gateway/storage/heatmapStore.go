package storage

import (
	"strings"
)

type DailyBucket struct {
	Date     string `json:"date"`
	Duration int64  `json:"duration"`
	Count    int64  `json:"count"`
}

func (s *Store) DailyStats(deviceID, category string, start, end int64) ([]DailyBucket, error) {
	where := []string{"end_time>=?", "start_time<=?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if category != "" {
		where = append(where, "category=?")
		args = append(args, category)
	}
	sqlStr := `SELECT strftime('%Y-%m-%d', start_time, 'unixepoch', 'localtime') AS day,
		SUM(duration), COUNT(*) FROM segments
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY day ORDER BY day ASC`
	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyBucket
	for rows.Next() {
		var b DailyBucket
		if err := rows.Scan(&b.Date, &b.Duration, &b.Count); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

type ProjectStat struct {
	WindowTitle string `json:"window_title"`
	ProcessName string `json:"process_name"`
	Duration    int64  `json:"duration"`
	Count       int64  `json:"count"`
}

func (s *Store) ProjectStats(deviceID, category string, start, end int64, limit int) ([]ProjectStat, error) {
	where := []string{"IFNULL(window_title,'')<>''"}
	args := []interface{}{}
	if start > 0 {
		where = append(where, "end_time>=?")
		args = append(args, start)
	}
	if end > 0 {
		where = append(where, "start_time<=?")
		args = append(args, end)
	}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if category != "" {
		where = append(where, "category=?")
		args = append(args, category)
	}
	sqlStr := `SELECT window_title, process_name, SUM(duration), COUNT(*) FROM segments
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY window_title, process_name ORDER BY SUM(duration) DESC`
	if limit <= 0 {
		limit = 20
	}
	sqlStr += " LIMIT ?"
	args = append(args, limit)
	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectStat
	for rows.Next() {
		var p ProjectStat
		if err := rows.Scan(&p.WindowTitle, &p.ProcessName, &p.Duration, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
