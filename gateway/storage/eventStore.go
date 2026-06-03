package storage

import (
	"database/sql"
	"encoding/json"
	"strings"
)

type Event struct {
	ID          int64                  `json:"id,omitempty"`
	EventID     string                 `json:"event_id,omitempty"`
	DeviceID    string                 `json:"device_id"`
	Timestamp   int64                  `json:"timestamp"`
	ReceivedAt  int64                  `json:"received_at,omitempty"`
	EventType   string                 `json:"event_type"`
	Category    string                 `json:"category,omitempty"`
	ProcessName string                 `json:"process_name,omitempty"`
	WindowTitle string                 `json:"window_title,omitempty"`
	PID         int                    `json:"pid,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type EventQuery struct {
	DeviceID  string
	Start     int64
	End       int64
	EventType string
	Category  string
	Limit     int
	Offset    int
}

// InsertEvent stores an event, deduplicating on event_id. It reports whether a
// new row was actually inserted (false means the event_id was already present
// and the insert was ignored), so callers can avoid double-counting derived
// state such as timeline segments when an agent re-sends a batch.
func (s *Store) InsertEvent(tx *sql.Tx, e *Event) (inserted bool, err error) {
	var meta sql.NullString
	if len(e.Metadata) > 0 {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return false, err
		}
		meta = sql.NullString{String: string(b), Valid: true}
	}
	var pid sql.NullInt64
	if e.PID > 0 {
		pid = sql.NullInt64{Int64: int64(e.PID), Valid: true}
	}
	var receivedAt sql.NullInt64
	if e.ReceivedAt > 0 {
		receivedAt = sql.NullInt64{Int64: e.ReceivedAt, Valid: true}
	}
	res, err := tx.Exec(`
		INSERT OR IGNORE INTO events (event_id, device_id, timestamp, received_at, event_type, category, process_name, window_title, pid, metadata)
		VALUES (?,?,?,?,?,?,?,?,?,?)
	`, nullable(e.EventID), e.DeviceID, e.Timestamp, receivedAt, e.EventType, nullable(e.Category),
		nullable(e.ProcessName), nullable(e.WindowTitle), pid, meta)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) QueryEvents(q EventQuery) ([]Event, error) {
	var where []string
	var args []interface{}
	if q.DeviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, q.DeviceID)
	}
	if q.Start > 0 {
		where = append(where, "timestamp>=?")
		args = append(args, q.Start)
	}
	if q.End > 0 {
		where = append(where, "timestamp<=?")
		args = append(args, q.End)
	}
	if q.EventType != "" {
		where = append(where, "event_type=?")
		args = append(args, q.EventType)
	}
	if q.Category != "" {
		where = append(where, "category=?")
		args = append(args, q.Category)
	}
	sqlStr := `SELECT id, device_id, timestamp, event_type, category, process_name, window_title, pid, metadata FROM events`
	if len(where) > 0 {
		sqlStr += " WHERE " + strings.Join(where, " AND ")
	}
	sqlStr += " ORDER BY timestamp DESC"
	if q.Limit <= 0 {
		q.Limit = 200
	}
	sqlStr += " LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := s.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var cat, proc, title, meta sql.NullString
		var pid sql.NullInt64
		if err := rows.Scan(&e.ID, &e.DeviceID, &e.Timestamp, &e.EventType,
			&cat, &proc, &title, &pid, &meta); err != nil {
			return nil, err
		}
		e.Category = cat.String
		e.ProcessName = proc.String
		e.WindowTitle = title.String
		if pid.Valid {
			e.PID = int(pid.Int64)
		}
		if meta.Valid && meta.String != "" {
			_ = json.Unmarshal([]byte(meta.String), &e.Metadata)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullable(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

type EventTypeCount struct {
	EventType string `json:"event_type"`
	Count     int64  `json:"count"`
}

func (s *Store) EventTypeCounts(deviceID string, start, end int64) ([]EventTypeCount, error) {
	where := []string{"timestamp BETWEEN ? AND ?"}
	args := []interface{}{start, end}
	if deviceID != "" {
		where = append(where, "device_id=?")
		args = append(args, deviceID)
	}
	q := `SELECT event_type, COUNT(*) FROM events WHERE ` + strings.Join(where, " AND ") + ` GROUP BY event_type ORDER BY COUNT(*) DESC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventTypeCount
	for rows.Next() {
		var c EventTypeCount
		if err := rows.Scan(&c.EventType, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
