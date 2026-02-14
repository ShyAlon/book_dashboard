package timeline

import "regexp"

type Event struct {
	TimeMarker string `json:"time_marker"`
	Event      string `json:"event"`
}

var markerRegex = regexp.MustCompile(`(?i)\b(next day|yesterday|today|tomorrow|last night|\d{4})\b`)

func ExtractMarkers(paragraph string) []string {
	return markerRegex.FindAllString(paragraph, -1)
}

func EventsFromText(text string, maxEvents int) []Event {
	if maxEvents <= 0 {
		maxEvents = 15
	}
	matches := markerRegex.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make([]Event, 0, min(maxEvents, len(matches)))
	for _, m := range matches {
		if len(out) >= maxEvents {
			break
		}
		start := max(0, m[0]-80)
		end := min(len(text), m[1]+120)
		fragment := text[start:end]
		marker := text[m[0]:m[1]]
		out = append(out, Event{
			TimeMarker: marker,
			Event:      compact(fragment),
		})
	}
	return out
}

func compact(s string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(s, " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
