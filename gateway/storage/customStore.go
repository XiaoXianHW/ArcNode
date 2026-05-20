package storage

import (
	"strings"
	"time"

	"github.com/XiaoXianHW/ArcNode/gateway/category"
)

// CustomKeyword is a user-defined classification rule. Scope is either
// "process" (matched against process_name only — default) or "title"
// (matched against window_title only).
type CustomKeyword struct {
	ID        int64  `json:"id"`
	Category  string `json:"category"`
	Keyword   string `json:"keyword"`
	Scope     string `json:"scope"`
	CreatedAt int64  `json:"created_at"`
}

const (
	ScopeProcess = "process"
	ScopeTitle   = "title"
)

func normalizeScope(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == ScopeTitle {
		return ScopeTitle
	}
	return ScopeProcess
}

func (s *Store) ListCustomKeywords() ([]CustomKeyword, error) {
	rows, err := s.DB.Query(`SELECT id, category, keyword, IFNULL(scope,'process'), created_at FROM custom_keywords ORDER BY category ASC, scope ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomKeyword
	for rows.Next() {
		var k CustomKeyword
		if err := rows.Scan(&k.ID, &k.Category, &k.Keyword, &k.Scope, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) AddCustomKeyword(catName, keyword, scope string) (CustomKeyword, error) {
	catName = strings.ToLower(strings.TrimSpace(catName))
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	scope = normalizeScope(scope)
	now := time.Now().Unix()
	res, err := s.DB.Exec(
		`INSERT INTO custom_keywords (category, keyword, scope, created_at) VALUES (?, ?, ?, ?)
			ON CONFLICT(category, keyword, scope) DO UPDATE SET created_at=created_at`,
		catName, keyword, scope, now)
	if err != nil {
		return CustomKeyword{}, err
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		_ = s.DB.QueryRow(`SELECT id, created_at FROM custom_keywords WHERE category=? AND keyword=? AND scope=?`,
			catName, keyword, scope).Scan(&id, &now)
	}
	return CustomKeyword{ID: id, Category: catName, Keyword: keyword, Scope: scope, CreatedAt: now}, nil
}

func (s *Store) DeleteCustomKeyword(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM custom_keywords WHERE id=?`, id)
	return err
}

// CustomKeywordMap returns the user's custom keywords grouped by category
// and split into the (process, title) shape expected by the classifier.
func (s *Store) CustomKeywordMap() (map[string]category.Rules, error) {
	list, err := s.ListCustomKeywords()
	if err != nil {
		return nil, err
	}
	out := make(map[string]category.Rules, 8)
	for _, k := range list {
		cur := out[k.Category]
		if k.Scope == ScopeTitle {
			cur.Title = append(cur.Title, k.Keyword)
		} else {
			cur.Process = append(cur.Process, k.Keyword)
		}
		out[k.Category] = cur
	}
	return out, nil
}
