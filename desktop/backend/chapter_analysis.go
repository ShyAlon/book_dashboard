package backend

import (
	"fmt"
	"regexp"
	"strings"

	"book_dashboard/internal/timeline"
)

var chapterHeaderPattern = regexp.MustCompile(`(?i)^\s*(chapter|ch\.)\s+([0-9ivxlcdm]+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)\b.*`)
var chapterInlinePattern = regexp.MustCompile(`(?i)\b(chapter|ch\.)\s+([0-9ivxlcdm]+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)\b`)

func splitChapters(text string) []chapter {
	if chunks := splitByInlineHeaders(text); len(chunks) >= 3 {
		return chunks
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]chapter, 0, 64)
	var currentTitle string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		idx := len(out) + 1
		title := currentTitle
		if title == "" {
			title = fmt.Sprintf("Chapter %d", idx)
		}
		out = append(out, chapter{index: idx, title: title, text: strings.Join(current, "\n")})
		current = nil
	}

	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if chapterHeaderPattern.MatchString(trim) {
			flush()
			currentTitle = trim
			continue
		}
		if trim != "" {
			current = append(current, trim)
		}
	}
	flush()

	if len(out) > 0 {
		return out
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []chapter{{index: 1, title: "Chapter 1", text: ""}}
	}
	const chunkSize = 2500
	for i := 0; i < len(words); i += chunkSize {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		idx := len(out) + 1
		out = append(out, chapter{index: idx, title: fmt.Sprintf("Chapter %d", idx), text: strings.Join(words[i:end], " ")})
	}
	return out
}

func splitByInlineHeaders(text string) []chapter {
	matches := chapterInlinePattern.FindAllStringIndex(text, -1)
	if len(matches) < 2 {
		return nil
	}

	out := make([]chapter, 0, len(matches))
	for i := 0; i < len(matches); i++ {
		start := matches[i][0]
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		chunk := strings.TrimSpace(text[start:end])
		if chunk == "" {
			continue
		}
		title := extractTitle(chunk, i+1)
		out = append(out, chapter{
			index: len(out) + 1,
			title: title,
			text:  chunk,
		})
	}
	return out
}

func extractTitle(chunk string, fallback int) string {
	line := firstWords(chunk, 8)
	if chapterHeaderPattern.MatchString(line) {
		return line
	}
	return fmt.Sprintf("Chapter %d", fallback)
}

func extractChapterMarkers(text string) []string {
	return timeline.ExtractMarkers(text)
}

func firstWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, " ")
}
