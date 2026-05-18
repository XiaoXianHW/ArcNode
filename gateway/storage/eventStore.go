package storage

import (
	"database/sql"
	"encoding/json"
	"strings"
)

type Event struct {
	ID          int64                  `json:"id,omitempty"`
	DeviceID    string                 `json:"device_id"`
	Timestamp   int64                  `json:"timestamp"`
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

func (s *Store) InsertEvent(tx *sql.Tx, e *Event) (int64, error) {
	var meta sql.NullString
	if len(e.Metadata) > 0 {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return 0, err
		}
		meta = sql.NullString{String: string(b), Valid: true}
	}
	var pid sql.NullInt64
	if e.PID > 0 {
		pid = sql.NullInt64{Int64: int64(e.PID), Valid: true}
	}
	res, err := tx.Exec(`
		INSERT INTO events (device_id, timestamp, event_type, category, process_name, window_title, pid, metadata)
		VALUES (?,?,?,?,?,?,?,?)
	`, e.DeviceID, e.Timestamp, e.EventType, nullable(e.Category),
		nullable(e.ProcessName), nullable(e.WindowTitle), pid, meta)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
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
