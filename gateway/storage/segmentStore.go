package storage

import (
	"database/sql"
	"strings"
	"time"
)

type Segment struct {
	ID          int64  `json:"id"`
	DeviceID    string `json:"device_id"`
	ProcessName string `json:"process_name"`
	WindowTitle string `json:"window_title,omitempty"`
	Category    string `json:"category,omitempty"`
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
	Duration    int64  `json:"duration"`
}

type SegmentQuery struct {
	DeviceID string
	Start    int64
	End      int64
	Category string
	Limit    int
	Offset   int
}

func (s *Store) UpsertSegment(tx *sql.Tx, deviceID, processName, windowTitle, category string, ts, gap int64) error {
	row := tx.QueryRow(`
		SELECT id, start_time, end_time FROM segments
		WHERE device_id=? AND process_name=? AND IFNULL(window_title,'')=IFNULL(?, '')
		ORDER BY end_time DESC LIMIT 1
	`, deviceID, processName, nullable(windowTitle))

	var id, startTime, endTime int64
	err := row.Scan(&id, &startTime, &endTime)
	if err == sql.ErrNoRows || ts-endTime > gap {
		_, err := tx.Exec(`
			INSERT INTO segments (device_id, process_name, window_title, category, start_time, end_time, duration)
			VALUES (?,?,?,?,?,?,0)
		`, deviceID, processName, nullable(windowTitle), nullable(category), ts, ts)
		return err
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE segments SET end_time=?, duration=?-start_time, category=COALESCE(NULLIF(category,''), ?) WHERE id=?`,
		ts, ts, nullable(category), id)
	return err
}

func (s *Store) QuerySegments(q SegmentQuery) ([]Segment, error) {
	var where []string
	var args []interface{}
	if q.DeviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, q.DeviceID)
	}
	if q.Start > 0 {
		where = append(where, "end_time>=?")
		args = append(args, q.Start)
	}
	if q.End > 0 {
		where = append(where, "start_time<=?")
		args = append(args, q.End)
	}
	if q.Category != "" {
		where = append(where, "category=?")
		args = append(args, q.Category)
	}
	sqlStr := `SELECT id, device_id, process_name, window_title, category, start_time, end_time, duration FROM segments`
	if len(where) > 0 {
		sqlStr += " WHERE " + strings.Join(where, " AND ")
	}
	sqlStr += " ORDER BY start_time ASC"
	if q.Limit > 0 {
		sqlStr += " LIMIT ? OFFSET ?"
		args = append(args, q.Limit, q.Offset)
	}
	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Segment
	for rows.Next() {
		var seg Segment
		var title, cat sql.NullString
		if err := rows.Scan(&seg.ID, &seg.DeviceID, &seg.ProcessName, &title, &cat,
			&seg.StartTime, &seg.EndTime, &seg.Duration); err != nil {
			return nil, err
		}
		seg.WindowTitle = title.String
		seg.Category = cat.String
		out = append(out, seg)
	}
	return out, rows.Err()
}

type CategoryStat struct {
	Category string `json:"category"`
	Duration int64  `json:"duration"`
	Count    int64  `json:"count"`
}

func (s *Store) CategoryStats(deviceID string, start, end int64) ([]CategoryStat, error) {
	var where []string
	var args []interface{}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if start > 0 {
		where = append(where, "end_time>=?")
		args = append(args, start)
	}
	if end > 0 {
		where = append(where, "start_time<=?")
		args = append(args, end)
	}
	sqlStr := `SELECT COALESCE(NULLIF(category,''),'uncategorized') AS cat, SUM(duration), COUNT(*) FROM segments`
	if len(where) > 0 {
		sqlStr += " WHERE " + strings.Join(where, " AND ")
	}
	sqlStr += " GROUP BY cat ORDER BY SUM(duration) DESC"
	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CategoryStat
	for rows.Next() {
		var c CategoryStat
		if err := rows.Scan(&c.Category, &c.Duration, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type AppStat struct {
	ProcessName string `json:"process_name"`
	Category    string `json:"category"`
	Duration    int64  `json:"duration"`
	Count       int64  `json:"count"`
}

func (s *Store) AppStats(deviceID string, start, end int64, limit int) ([]AppStat, error) {
	var where []string
	var args []interface{}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if start > 0 {
		where = append(where, "end_time>=?")
		args = append(args, start)
	}
	if end > 0 {
		where = append(where, "start_time<=?")
		args = append(args, end)
	}
	sqlStr := `SELECT process_name, COALESCE(NULLIF(category,''),'uncategorized'), SUM(duration), COUNT(*) FROM segments`
	if len(where) > 0 {
		sqlStr += " WHERE " + strings.Join(where, " AND ")
	}
	sqlStr += " GROUP BY process_name ORDER BY SUM(duration) DESC"
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
	var out []AppStat
	for rows.Next() {
		var a AppStat
		if err := rows.Scan(&a.ProcessName, &a.Category, &a.Duration, &a.Count); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type ShortcutStat struct {
	Shortcut    string `json:"shortcut"`
	Application string `json:"application,omitempty"`
	Count       int64  `json:"count"`
}

func (s *Store) ShortcutStats(deviceID string, start, end int64, limit int) ([]ShortcutStat, error) {
	var where []string
	args := []interface{}{}
	where = append(where, "event_type='keyboard_shortcut'")
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	if start > 0 {
		where = append(where, "timestamp>=?")
		args = append(args, start)
	}
	if end > 0 {
		where = append(where, "timestamp<=?")
		args = append(args, end)
	}
	sqlStr := `SELECT json_extract(metadata,'$.shortcut') AS sc,
		COALESCE(json_extract(metadata,'$.application'),'') AS app,
		COUNT(*) AS cnt FROM events`
	sqlStr += " WHERE " + strings.Join(where, " AND ")
	sqlStr += " GROUP BY sc, app ORDER BY cnt DESC"
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
	var out []ShortcutStat
	for rows.Next() {
		var sc ShortcutStat
		var scStr, appStr sql.NullString
		if err := rows.Scan(&scStr, &appStr, &sc.Count); err != nil {
			return nil, err
		}
		if !scStr.Valid {
			continue
		}
		sc.Shortcut = scStr.String
		sc.Application = appStr.String
		out = append(out, sc)
	}
	return out, rows.Err()
}

type IdleStat struct {
	IdleSeconds   int64 `json:"idle_seconds"`
	ActiveSeconds int64 `json:"active_seconds"`
}

func (s *Store) IdleStats(deviceID string, start, end int64) (IdleStat, error) {
	var out IdleStat
	rows, err := s.DB.Query(`
		SELECT timestamp, event_type FROM events
		WHERE device_id=? AND timestamp BETWEEN ? AND ? AND event_type IN ('idle_start','idle_end')
		ORDER BY timestamp ASC
	`, deviceID, start, end)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	var idleStart int64
	for rows.Next() {
		var ts int64
		var et string
		if err := rows.Scan(&ts, &et); err != nil {
			return out, err
		}
		if et == "idle_start" {
			idleStart = ts
		} else if et == "idle_end" && idleStart > 0 {
			out.IdleSeconds += ts - idleStart
			idleStart = 0
		}
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	// Bound the active window by the actual event range so an empty/early
	// day doesn't report a full 24h of "active" time. The window is
	// [first_event, min(last_event, end, now)].
	firstTs, lastTs, errR := s.eventRange(deviceID, start, end)
	if errR != nil {
		return out, errR
	}
	if firstTs == 0 || lastTs == 0 {
		return out, nil
	}
	now := time.Now().Unix()
	upper := lastTs
	if end < upper {
		upper = end
	}
	if now < upper {
		upper = now
	}
	if idleStart > 0 && upper > idleStart {
		out.IdleSeconds += upper - idleStart
	}
	total := upper - firstTs
	if total < 0 {
		total = 0
	}
	out.ActiveSeconds = total - out.IdleSeconds
	if out.ActiveSeconds < 0 {
		out.ActiveSeconds = 0
	}
	return out, nil
}

func (s *Store) eventRange(deviceID string, start, end int64) (int64, int64, error) {
	var first, last sql.NullInt64
	err := s.DB.QueryRow(`
		SELECT MIN(timestamp), MAX(timestamp) FROM events
		WHERE device_id=? AND timestamp BETWEEN ? AND ?
	`, deviceID, start, end).Scan(&first, &last)
	if err != nil {
		return 0, 0, err
	}
	return first.Int64, last.Int64, nil
}

// Classifier is the minimal interface used by ReclassifyAll so the storage
// package does not depend on category.
type Classifier interface {
	Classify(processName, windowTitle string) string
}

type ReclassifyResult struct {
	SegmentsScanned int64 `json:"segments_scanned"`
	SegmentsUpdated int64 `json:"segments_updated"`
	EventsScanned   int64 `json:"events_scanned"`
	EventsUpdated   int64 `json:"events_updated"`
}

// ReclassifyAll re-runs the classifier over stored segments and
// foreground_change events, updating any row whose stored category no
// longer matches what the classifier returns. Idle/shortcut/system rows
// keep their assigned category untouched.
func (s *Store) ReclassifyAll(cl Classifier) (ReclassifyResult, error) {
	var out ReclassifyResult
	if cl == nil {
		return out, nil
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`SELECT id, IFNULL(process_name,''), IFNULL(window_title,''), IFNULL(category,'') FROM segments`)
	if err != nil {
		return out, err
	}
	type update struct {
		id  int64
		cat string
	}
	var segUpdates []update
	for rows.Next() {
		var id int64
		var proc, title, cur string
		if err := rows.Scan(&id, &proc, &title, &cur); err != nil {
			rows.Close()
			return out, err
		}
		out.SegmentsScanned++
		next := cl.Classify(proc, title)
		if next != cur {
			segUpdates = append(segUpdates, update{id, next})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	for _, u := range segUpdates {
		var cat interface{}
		if u.cat == "" {
			cat = nil
		} else {
			cat = u.cat
		}
		if _, err := tx.Exec(`UPDATE segments SET category=? WHERE id=?`, cat, u.id); err != nil {
			return out, err
		}
	}
	out.SegmentsUpdated = int64(len(segUpdates))

	rows, err = tx.Query(`SELECT id, IFNULL(process_name,''), IFNULL(window_title,''), IFNULL(category,'') FROM events WHERE event_type='foreground_change'`)
	if err != nil {
		return out, err
	}
	var evUpdates []update
	for rows.Next() {
		var id int64
		var proc, title, cur string
		if err := rows.Scan(&id, &proc, &title, &cur); err != nil {
			rows.Close()
			return out, err
		}
		out.EventsScanned++
		next := cl.Classify(proc, title)
		if next != cur {
			evUpdates = append(evUpdates, update{id, next})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	for _, u := range evUpdates {
		var cat interface{}
		if u.cat == "" {
			cat = nil
		} else {
			cat = u.cat
		}
		if _, err := tx.Exec(`UPDATE events SET category=? WHERE id=?`, cat, u.id); err != nil {
			return out, err
		}
	}
	out.EventsUpdated = int64(len(evUpdates))

	if err := tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}
