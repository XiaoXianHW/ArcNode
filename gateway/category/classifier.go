package category

import (
	"sort"
	"strings"
	"sync"
)

// Rules holds keywords matched against process_name and window_title
// separately. Process matches always take priority over title matches.
type Rules struct {
	Process []string `json:"process"`
	Title   []string `json:"title"`
}

// priorityOrder defines a stable iteration order so the more specific
// categories beat the generic ones (e.g. terminal beats coding, gaming
// beats browsing).
var priorityOrder = []string{
	"terminal",
	"coding",
	"ai_tools",
	"design",
	"gaming",
	"video",
	"music",
	"communication",
	"social",
	"reading",
	"productivity",
	"browsing",
}

type Classifier struct {
	mu      sync.RWMutex
	builtin map[string]Rules
	custom  map[string]Rules
	merged  map[string]Rules
	order   []string
}

func New(rules map[string]Rules) *Classifier {
	c := &Classifier{
		builtin: normalizeRules(rules),
		custom:  map[string]Rules{},
	}
	c.rebuild()
	return c
}

// Classify returns the best-matching category or "" when no rule matches.
//
// Rules:
//   - process_keywords always take precedence over title_keywords.
//   - The only exception: when the process match landed in the very
//     generic "browsing" category, a title hit on a more specific
//     category (video, ai_tools, …) is allowed to upgrade the result.
//
// This guards against the classic title-hijack false positives such as
// Chrome with a window title containing the word "Minecraft" being
// classified as gaming.
func (c *Classifier) Classify(processName, windowTitle string) string {
	if c == nil {
		return ""
	}
	proc := strings.ToLower(strings.TrimSpace(processName))
	title := strings.ToLower(strings.TrimSpace(windowTitle))
	c.mu.RLock()
	defer c.mu.RUnlock()

	procCat := ""
	if proc != "" {
		for _, cat := range c.order {
			for _, k := range c.merged[cat].Process {
				if k != "" && strings.Contains(proc, k) {
					procCat = cat
					break
				}
			}
			if procCat != "" {
				break
			}
		}
	}

	titleCat := ""
	if title != "" {
		for _, cat := range c.order {
			for _, k := range c.merged[cat].Title {
				if k != "" && strings.Contains(title, k) {
					titleCat = cat
					break
				}
			}
			if titleCat != "" {
				break
			}
		}
	}

	if procCat == "" {
		return titleCat
	}
	if titleCat != "" && (procCat == "browsing") {
		return titleCat
	}
	return procCat
}

// Rules returns a deep copy of the active merged rule set keyed by category.
func (c *Classifier) Rules() map[string]Rules {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]Rules, len(c.merged))
	for k, v := range c.merged {
		out[k] = Rules{Process: append([]string{}, v.Process...), Title: append([]string{}, v.Title...)}
	}
	return out
}

// Builtin returns just the built-in defaults (without custom overlay).
func (c *Classifier) Builtin() map[string]Rules {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]Rules, len(c.builtin))
	for k, v := range c.builtin {
		out[k] = Rules{Process: append([]string{}, v.Process...), Title: append([]string{}, v.Title...)}
	}
	return out
}

// SetCustom replaces the custom overlay with the given rules and rebuilds.
func (c *Classifier) SetCustom(rules map[string]Rules) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.custom = normalizeRules(rules)
	c.rebuild()
}

func (c *Classifier) rebuild() {
	merged := make(map[string]Rules, len(c.builtin)+len(c.custom))
	for cat, r := range c.builtin {
		merged[cat] = Rules{Process: append([]string{}, r.Process...), Title: append([]string{}, r.Title...)}
	}
	for cat, r := range c.custom {
		cur := merged[cat]
		cur.Process = dedupAppend(cur.Process, r.Process)
		cur.Title = dedupAppend(cur.Title, r.Title)
		merged[cat] = cur
	}
	c.merged = merged
	c.order = orderCategories(merged)
}

func orderCategories(rules map[string]Rules) []string {
	seen := map[string]bool{}
	order := make([]string, 0, len(rules))
	for _, cat := range priorityOrder {
		if _, ok := rules[cat]; ok {
			order = append(order, cat)
			seen[cat] = true
		}
	}
	tail := make([]string, 0)
	for cat := range rules {
		if !seen[cat] {
			tail = append(tail, cat)
		}
	}
	sort.Strings(tail)
	return append(order, tail...)
}

func dedupAppend(base, extra []string) []string {
	seen := map[string]bool{}
	for _, k := range base {
		seen[k] = true
	}
	for _, k := range extra {
		k = strings.TrimSpace(strings.ToLower(k))
		if k != "" && !seen[k] {
			base = append(base, k)
			seen[k] = true
		}
	}
	return base
}

func normalizeRules(rules map[string]Rules) map[string]Rules {
	out := make(map[string]Rules, len(rules))
	for cat, r := range rules {
		cat = strings.ToLower(strings.TrimSpace(cat))
		if cat == "" {
			continue
		}
		out[cat] = Rules{
			Process: normalizeList(r.Process),
			Title:   normalizeList(r.Title),
		}
	}
	return out
}

func normalizeList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, k := range in {
		k = strings.TrimSpace(strings.ToLower(k))
		if k == "" || seen[k] {
			continue
		}
		out = append(out, k)
		seen[k] = true
	}
	return out
}

// FlattenForUI returns category → flat keyword list for backward-compatible UI
// surfaces (kept around for old clients that only know the flat shape).
func FlattenForUI(rules map[string]Rules) map[string][]string {
	out := make(map[string][]string, len(rules))
	for cat, r := range rules {
		merged := make([]string, 0, len(r.Process)+len(r.Title))
		merged = append(merged, r.Process...)
		merged = append(merged, r.Title...)
		out[cat] = merged
	}
	return out
}
