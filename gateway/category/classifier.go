package category

import "strings"

type Classifier struct {
	rules map[string][]string
}

func New(rules map[string][]string) *Classifier {
	normalized := make(map[string][]string, len(rules))
	for cat, kws := range rules {
		lower := make([]string, 0, len(kws))
		for _, k := range kws {
			k = strings.TrimSpace(strings.ToLower(k))
			if k != "" {
				lower = append(lower, k)
			}
		}
		normalized[cat] = lower
	}
	return &Classifier{rules: normalized}
}

func (c *Classifier) Classify(processName, windowTitle string) string {
	if c == nil {
		return ""
	}
	hay := strings.ToLower(processName + " " + windowTitle)
	if hay == " " {
		return ""
	}
	for cat, kws := range c.rules {
		for _, k := range kws {
			if strings.Contains(hay, k) {
				return cat
			}
		}
	}
	return ""
}

func (c *Classifier) Rules() map[string][]string {
	out := make(map[string][]string, len(c.rules))
	for k, v := range c.rules {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}
