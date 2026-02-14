package backend

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"book_dashboard/internal/forensics"
)

var eyesPattern = regexp.MustCompile(`(?i)\b([A-Z][a-z]+)\b[^.\n]{0,45}\beyes\b[^.\n]{0,25}\b(blue|brown|green|hazel|gray|grey)\b`)
var agePattern = regexp.MustCompile(`(?i)\b([A-Z][a-z]+)\b[^.\n]{0,35}\b(?:age|aged)\b[^0-9\n]{0,10}([0-9]{1,3})\b`)
var lifePattern = regexp.MustCompile(`(?i)\b([A-Z][a-z]+)\b[^.\n]{0,30}\b(dead|alive)\b`)

func detectHeuristicContradictions(chapters []chapter) []forensics.Contradiction {
	profiles := make([]forensics.ChapterProfile, 0, 256)
	for _, ch := range chapters {
		entityAttrs := map[string]map[string]string{}
		for _, m := range eyesPattern.FindAllStringSubmatch(ch.text, -1) {
			name := strings.TrimSpace(m[1])
			if isIgnoredEntityName(name) {
				continue
			}
			if entityAttrs[name] == nil {
				entityAttrs[name] = map[string]string{}
			}
			entityAttrs[name]["eyes"] = strings.ToLower(m[2])
		}
		for _, m := range agePattern.FindAllStringSubmatch(ch.text, -1) {
			name := strings.TrimSpace(m[1])
			if isIgnoredEntityName(name) {
				continue
			}
			if entityAttrs[name] == nil {
				entityAttrs[name] = map[string]string{}
			}
			entityAttrs[name]["age"] = m[2]
		}
		for _, m := range lifePattern.FindAllStringSubmatch(ch.text, -1) {
			name := strings.TrimSpace(m[1])
			if isIgnoredEntityName(name) {
				continue
			}
			if entityAttrs[name] == nil {
				entityAttrs[name] = map[string]string{}
			}
			entityAttrs[name]["dead"] = strconv.FormatBool(strings.EqualFold(m[2], "dead"))
		}
		for name, attrs := range entityAttrs {
			profiles = append(profiles, forensics.ChapterProfile{Chapter: ch.index, Name: name, Attributes: attrs})
		}
	}
	raw := forensics.DetectContradictions(profiles)
	return filterContradictions(raw)
}

func filterContradictions(raw []forensics.Contradiction) []forensics.Contradiction {
	out := make([]forensics.Contradiction, 0, len(raw))
	for _, c := range raw {
		// Allow natural progression from alive(false dead) to dead(true dead).
		if c.Attribute == "dead" && strings.EqualFold(c.ValueA, "false") && strings.EqualFold(c.ValueB, "true") && c.ChapterB > c.ChapterA {
			continue
		}
		out = append(out, c)
	}
	return out
}

func isIgnoredEntityName(name string) bool {
	if name == "" {
		return true
	}
	ignore := []string{"He", "She", "They", "Them", "Their", "The", "This", "That", "There", "You", "We", "I", "It", "His", "Her", "Our", "Your", "A", "An", "And", "But"}
	return slices.Contains(ignore, name)
}
