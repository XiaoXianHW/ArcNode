package category

import (
	"strings"
	"sync"
)

type Classifier struct {
	mu      sync.RWMutex
	builtin map[string][]string
	custom  map[string][]string
	merged  map[string][]string
}

func New(rules map[string][]string) *Classifier {
	c := &Classifier{
		builtin: normalize(rules),
		custom:  map[string][]string{},
	}
	c.rebuild()
	return c
}

func (c *Classifier) Classify(processName, windowTitle string) string {
	if c == nil {
		return ""
	}
	hay := strings.ToLower(processName + " " + windowTitle)
	if hay == " " {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for cat, kws := range c.merged {
		for _, k := range kws {
			if strings.Contains(hay, k) {
				return cat
			}
		}
	}
	return ""
}

func (c *Classifier) Rules() map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string][]string, len(c.merged))
	for k, v := range c.merged {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func (c *Classifier) Builtin() map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string][]string, len(c.builtin))
	for k, v := range c.builtin {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func (c *Classifier) SetCustom(rules map[string][]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.custom = normalize(rules)
	c.rebuild()
}

func (c *Classifier) rebuild() {
	merged := make(map[string][]string, len(c.builtin)+len(c.custom))
	for cat, kws := range c.builtin {
		merged[cat] = append([]string{}, kws...)
	}
	for cat, kws := range c.custom {
		seen := map[string]bool{}
		for _, k := range merged[cat] {
			seen[k] = true
		}
		for _, k := range kws {
			if k != "" && !seen[k] {
				merged[cat] = append(merged[cat], k)
				seen[k] = true
			}
		}
	}
	c.merged = merged
}

func normalize(rules map[string][]string) map[string][]string {
	out := make(map[string][]string, len(rules))
	for cat, kws := range rules {
		cat = strings.ToLower(strings.TrimSpace(cat))
		if cat == "" {
			continue
		}
		lower := make([]string, 0, len(kws))
		seen := map[string]bool{}
		for _, k := range kws {
			k = strings.TrimSpace(strings.ToLower(k))
			if k != "" && !seen[k] {
				lower = append(lower, k)
				seen[k] = true
			}
		}
		out[cat] = lower
	}
	return out
}
