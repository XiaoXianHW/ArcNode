package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func insert(t *testing.T, s *Store, e *Event) bool {
	t.Helper()
	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	ok, err := s.InsertEvent(tx, e)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return ok
}

func TestInsertEventDeduplicatesByEventID(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().Unix()
	e := &Event{EventID: "evt-1", DeviceID: "dev-1", Timestamp: now, EventType: "foreground_change"}

	if !insert(t, s, e) {
		t.Fatal("first insert should report inserted=true")
	}
	if insert(t, s, e) {
		t.Fatal("re-sending the same event_id should report inserted=false")
	}

	rows, err := s.QueryEvents(EventQuery{DeviceID: "dev-1"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 stored event after dedup, got %d", len(rows))
	}
}

func TestInsertEventWithoutIDAlwaysInserts(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().Unix()
	// Legacy agents send no event_id; those must never be deduplicated.
	for i := 0; i < 3; i++ {
		e := &Event{DeviceID: "dev-1", Timestamp: now, EventType: "system_sample"}
		if !insert(t, s, e) {
			t.Fatal("event without event_id should always insert")
		}
	}
	rows, err := s.QueryEvents(EventQuery{DeviceID: "dev-1"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 stored events, got %d", len(rows))
	}
}

func TestPruneOlderThan(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().Unix()
	old := now - 100
	insert(t, s, &Event{EventID: "old", DeviceID: "d", Timestamp: old, EventType: "system_sample"})
	insert(t, s, &Event{EventID: "new", DeviceID: "d", Timestamp: now, EventType: "system_sample"})

	ev, _, err := s.PruneOlderThan(now - 50)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if ev != 1 {
		t.Fatalf("expected 1 pruned event, got %d", ev)
	}
	rows, err := s.QueryEvents(EventQuery{DeviceID: "d"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 || rows[0].Timestamp != now {
		t.Fatalf("expected only the recent event to remain, got %+v", rows)
	}
}
