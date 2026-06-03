package storage

// PruneOlderThan deletes events and segments whose primary timestamp is older
// than the given cutoff (unix seconds). It returns the number of rows removed
// from each table. Used to enforce a configurable retention window so a
// long-running self-hosted gateway does not grow without bound.
func (s *Store) PruneOlderThan(cutoff int64) (events int64, segments int64, err error) {
	res, err := s.DB.Exec(`DELETE FROM events WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, 0, err
	}
	events, _ = res.RowsAffected()

	res, err = s.DB.Exec(`DELETE FROM segments WHERE end_time < ?`, cutoff)
	if err != nil {
		return events, 0, err
	}
	segments, _ = res.RowsAffected()
	return events, segments, nil
}

// Vacuum reclaims disk space left behind after large deletes. Should be called
// sparingly (e.g. after a prune) since it rewrites the whole database file.
func (s *Store) Vacuum() error {
	_, err := s.DB.Exec(`VACUUM`)
	return err
}
