package backend

import (
	"book_dashboard/internal/structure"
	"book_dashboard/internal/timeline"
	"fmt"
)

func buildTimeline(chapters []chapter, chapterSummaries []ChapterSummary) []timeline.Event {
	out := make([]timeline.Event, 0, 40)
	for _, ch := range chapters {
		markers := extractChapterMarkers(ch.text)
		if len(markers) == 0 {
			continue
		}
		for _, m := range markers {
			if len(out) >= 40 {
				return out
			}
			out = append(out, timeline.Event{
				TimeMarker: m,
				Event:      fmt.Sprintf("Ch %d %s: %s", ch.index, ch.title, firstWords(ch.text, 16)),
			})
		}
	}
	if len(out) > 0 {
		return out
	}

	summaryByChapter := make(map[int]ChapterSummary, len(chapterSummaries))
	for _, s := range chapterSummaries {
		summaryByChapter[s.Chapter] = s
	}

	for _, ch := range chapters {
		if len(out) >= 12 {
			break
		}
		summary := firstWords(ch.text, 16)
		if s, ok := summaryByChapter[ch.index]; ok {
			if len(s.Events) > 0 {
				summary = s.Events[0]
			} else if s.Summary != "" {
				summary = firstWords(s.Summary, 16)
			}
		}
		out = append(out, timeline.Event{TimeMarker: fmt.Sprintf("Chapter %d", ch.index), Event: summary})
	}
	return out
}

func buildBeats(chapters []chapter, chapterSummaries []ChapterSummary, chapterMetrics []ChapterMetric, timelineEvents []timeline.Event) []BeatResult {
	beats := make([]BeatResult, 0, len(structure.SaveTheCatWindows))
	total := len(chapters)
	if total == 0 {
		return beats
	}

	summaryByChapter := make(map[int]ChapterSummary, len(chapterSummaries))
	for _, s := range chapterSummaries {
		summaryByChapter[s.Chapter] = s
	}
	metricByChapter := make(map[int]ChapterMetric, len(chapterMetrics))
	for _, m := range chapterMetrics {
		metricByChapter[m.Index] = m
	}

	for _, bw := range structure.SaveTheCatWindows {
		start, end := structure.ChaptersInWindow(total, bw.StartRatio, bw.EndRatio)
		if start <= 0 || end <= 0 || start > total {
			continue
		}
		if end > total {
			end = total
		}
		snippets := make([]string, 0, end-start+1)
		for i := start - 1; i < end; i++ {
			chIndex := chapters[i].index
			if summary, ok := summaryByChapter[chIndex]; ok {
				if len(summary.Events) > 0 {
					snippets = append(snippets, firstWords(summary.Events[0], 22))
					continue
				}
				if summary.Summary != "" {
					snippets = append(snippets, firstWords(summary.Summary, 22))
					continue
				}
			}
			snippets = append(snippets, firstWords(chapters[i].text, 22))
		}
		reason := "Insufficient chapter text in beat window."
		isBeat := false
		if len(snippets) > 0 {
			reason = snippets[0]
			if len(snippets) > 1 {
				reason += " ... " + snippets[len(snippets)-1]
			}
			if m, ok := metricByChapter[start]; ok {
				reason += fmt.Sprintf(" (start genre=%s, timeline_markers=%d)", m.TopGenre, m.TimelineMarks)
			}
			isBeat = true
		}
		if len(timelineEvents) > 0 {
			reason += fmt.Sprintf(" [timeline_events=%d]", len(timelineEvents))
		}
		beats = append(beats, BeatResult{Name: bw.Name, StartChapter: start, EndChapter: end, IsBeat: isBeat, Reasoning: reason})
	}
	return beats
}
