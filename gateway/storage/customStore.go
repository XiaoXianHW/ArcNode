package storage

import (
	"strings"
	"time"
)

type CustomKeyword struct {
	ID        int64  `json:"id"`
	Category  string `json:"category"`
	Keyword   string `json:"keyword"`
	CreatedAt int64  `json:"created_at"`
}

func (s *Store) ListCustomKeywords() ([]CustomKeyword, error) {
	rows, err := s.DB.Query(`SELECT id, category, keyword, created_at FROM custom_keywords ORDER BY category ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomKeyword
	for rows.Next() {
		var k CustomKeyword
		if err := rows.Scan(&k.ID, &k.Category, &k.Keyword, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) AddCustomKeyword(category, keyword string) (CustomKeyword, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	now := time.Now().Unix()
	res, err := s.DB.Exec(
		`INSERT INTO custom_keywords (category, keyword, created_at) VALUES (?, ?, ?)
			ON CONFLICT(category, keyword) DO UPDATE SET created_at=created_at`,
		category, keyword, now)
	if err != nil {
		return CustomKeyword{}, err
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		_ = s.DB.QueryRow(`SELECT id, created_at FROM custom_keywords WHERE category=? AND keyword=?`,
			category, keyword).Scan(&id, &now)
	}
	return CustomKeyword{ID: id, Category: category, Keyword: keyword, CreatedAt: now}, nil
}

func (s *Store) DeleteCustomKeyword(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM custom_keywords WHERE id=?`, id)
	return err
}

func (s *Store) CustomKeywordMap() (map[string][]string, error) {
	list, err := s.ListCustomKeywords()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, 8)
	for _, k := range list {
		out[k.Category] = append(out[k.Category], k.Keyword)
	}
	return out, nil
}
