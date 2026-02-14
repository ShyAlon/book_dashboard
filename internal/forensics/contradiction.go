package forensics

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type ChapterProfile struct {
	Chapter    int
	Name       string
	Aliases    []string
	Attributes map[string]string
}

type Contradiction struct {
	EntityName  string
	Attribute   string
	ValueA      string
	ValueB      string
	ChapterA    int
	ChapterB    int
	Description string
	Severity    string
}

func DetectContradictions(profiles []ChapterProfile) []Contradiction {
	normalized := map[string][]ChapterProfile{}
	for _, p := range profiles {
		key := canonicalName(p.Name, p.Aliases)
		normalized[key] = append(normalized[key], p)
	}

	var out []Contradiction
	for entity, items := range normalized {
		seen := map[string]struct {
			value   string
			chapter int
		}{}
		for _, profile := range items {
			for k, v := range profile.Attributes {
				k = strings.TrimSpace(strings.ToLower(k))
				v = strings.TrimSpace(v)
				if prev, ok := seen[k]; ok && !strings.EqualFold(prev.value, v) {
					out = append(out, Contradiction{
						EntityName: entity,
						Attribute:  k,
						ValueA:     prev.value,
						ValueB:     v,
						ChapterA:   prev.chapter,
						ChapterB:   profile.Chapter,
						Description: fmt.Sprintf(
							"%s changed for %s: %q in Ch%d but %q in Ch%d",
							k, entity, prev.value, prev.chapter, v, profile.Chapter,
						),
						Severity: severityFor(k),
					})
					continue
				}
				seen[k] = struct {
					value   string
					chapter int
				}{value: v, chapter: profile.Chapter}
			}
		}
	}

	return out
}

func canonicalName(name string, aliases []string) string {
	candidates := append([]string{name}, aliases...)
	for i := range candidates {
		candidates[i] = strings.TrimSpace(strings.ToLower(candidates[i]))
	}
	slices.Sort(candidates)
	filtered := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c != "" {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return "unknown"
	}
	dedup := map[string]struct{}{}
	for _, c := range filtered {
		dedup[c] = struct{}{}
	}
	keys := slices.Collect(maps.Keys(dedup))
	slices.Sort(keys)
	return keys[0]
}

func severityFor(attribute string) string {
	switch attribute {
	case "dead", "alive":
		return "HIGH"
	case "age", "name", "eyes":
		return "MED"
	default:
		return "LOW"
	}
}
