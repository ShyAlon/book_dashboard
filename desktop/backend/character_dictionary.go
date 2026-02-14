package backend

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"book_dashboard/internal/forensics"
)

var properNamePattern = regexp.MustCompile(`\b[A-Z][a-z]{2,}\b`)
var sentenceExtractPattern = regexp.MustCompile(`[^.!?]+[.!?]?`)
var eventVerbPattern = regexp.MustCompile(`\b(arrived|left|discovered|revealed|decided|confronted|killed|died|escaped|found|lost|won|failed|confessed|attacked|agreed|refused|warned|promised|betrayed|collapsed|resigned|married|divorced|fled|returned|investigated|accused|admitted|exposed)\b`)
var causalMarkerPattern = regexp.MustCompile(`\b(because|therefore|after|before|when|then|suddenly|finally|meanwhile|later)\b`)
var dialogueOnlyPattern = regexp.MustCompile(`^\s*["']`)

const speechVerbAlternation = `(said|asked|replied|whispered|shouted|murmured|called|told|answered|cried|snapped)`

var speechVerbPattern = regexp.MustCompile(`(?i)\b` + speechVerbAlternation + `\b`)

var weakNameStopwords = map[string]struct{}{
	"what": {}, "maybe": {}, "not": {}, "well": {}, "yes": {}, "no": {}, "oh": {}, "ah": {}, "hmm": {},
	"however": {}, "anyway": {}, "therefore": {}, "meanwhile": {}, "then": {}, "also": {}, "still": {},
}

func buildCharacterDictionary(chapters []chapter) ([]CharacterEntry, []ChapterSummary, map[int]ChapterSummary) {
	type agg struct {
		entry CharacterEntry
	}
	entries := map[string]*agg{}
	chapterSummaries := make([]ChapterSummary, 0, len(chapters))
	chapterByID := map[int]ChapterSummary{}

	for _, ch := range chapters {
		events := deriveEvents(ch.text)
		summary := deriveSummary(ch.text, events)
		cs := ChapterSummary{Chapter: ch.index, Title: ch.title, Summary: summary, Events: events}
		chapterSummaries = append(chapterSummaries, cs)
		chapterByID[ch.index] = cs

		names := namesInText(ch.text)
		for _, name := range names {
			item, ok := entries[name]
			if !ok {
				item = &agg{entry: CharacterEntry{Name: name, Description: fmt.Sprintf("Appears in the manuscript across %d chapter(s).", 1), FirstSeenChapter: ch.index, LastSeenChapter: ch.index}}
				entries[name] = item
			}
			if ch.index < item.entry.FirstSeenChapter {
				item.entry.FirstSeenChapter = ch.index
			}
			if ch.index > item.entry.LastSeenChapter {
				item.entry.LastSeenChapter = ch.index
			}
			item.entry.TotalMentions += strings.Count(ch.text, name)
			item.entry.Chapters = append(item.entry.Chapters, CharacterChapterRecord{
				Chapter: ch.index,
				Title:   ch.title,
				Summary: summary,
				Actions: deriveActions(name, ch.text),
				Events:  events,
			})
		}
	}

	out := make([]CharacterEntry, 0, len(entries))
	for _, v := range entries {
		chapCount := len(v.entry.Chapters)
		v.entry.Description = fmt.Sprintf("Appears in %d chapter(s), from Ch %d to Ch %d.", chapCount, v.entry.FirstSeenChapter, v.entry.LastSeenChapter)
		out = append(out, v.entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalMentions == out[j].TotalMentions {
			return out[i].Name < out[j].Name
		}
		return out[i].TotalMentions > out[j].TotalMentions
	})
	sort.Slice(chapterSummaries, func(i, j int) bool { return chapterSummaries[i].Chapter < chapterSummaries[j].Chapter })
	return out, chapterSummaries, chapterByID
}

func buildHealthIssues(contradictions []forensics.Contradiction, chapterByID map[int]ChapterSummary) []HealthIssue {
	issues := make([]HealthIssue, 0, len(contradictions))
	for i, c := range contradictions {
		a := chapterByID[c.ChapterA]
		b := chapterByID[c.ChapterB]
		issues = append(issues, HealthIssue{
			ID:            fmt.Sprintf("issue-%03d", i+1),
			Entity:        c.EntityName,
			Severity:      c.Severity,
			Description:   c.Description,
			ChapterA:      c.ChapterA,
			ChapterB:      c.ChapterB,
			ContextA:      a.Summary,
			ContextB:      b.Summary,
			DictionaryRef: c.EntityName,
		})
	}
	return issues
}

func namesInText(text string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 32)
	for _, n := range properNamePattern.FindAllString(text, -1) {
		if !shouldKeepCharacterCandidate(n, text) {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func shouldKeepCharacterCandidate(name, text string) bool {
	if isIgnoredEntityName(name) {
		return false
	}
	if _, ok := weakNameStopwords[strings.ToLower(name)]; !ok {
		return true
	}
	return hasStrongNameEvidence(name, text)
}

func hasStrongNameEvidence(name, text string) bool {
	// Possessive and vocative usage are strong indicators of a real character name.
	if strings.Contains(text, name+"'s") || strings.Contains(text, name+"’s") || strings.Contains(text, ","+name+",") {
		return true
	}

	quoted := regexp.QuoteMeta(name)
	titlePattern := regexp.MustCompile(`(?i)\b(mr|mrs|ms|dr|prof)\.?\s+` + quoted + `\b`)
	if titlePattern.MatchString(text) {
		return true
	}

	beforeSpeech := regexp.MustCompile(`(?i)\b` + quoted + `\b\s+\b` + speechVerbAlternation + `\b`)
	if beforeSpeech.MatchString(text) {
		return true
	}
	afterSpeech := regexp.MustCompile(`(?i)\b` + speechVerbAlternation + `\b\s+\b` + quoted + `\b`)
	if afterSpeech.MatchString(text) {
		return true
	}

	return false
}

func deriveEvents(text string) []string {
	sentences := splitSentences(text)
	type candidate struct {
		sentence string
		score    int
	}
	candidates := make([]candidate, 0, len(sentences))
	for _, s := range sentences {
		lower := strings.ToLower(s)
		score := 0
		if eventVerbPattern.MatchString(lower) {
			score += 4
		}
		if causalMarkerPattern.MatchString(lower) {
			score += 2
		}
		if len(extractChapterMarkers(s)) > 0 {
			score += 2
		}
		if properNamePattern.MatchString(s) {
			score++
		}
		if dialogueOnlyPattern.MatchString(s) && !eventVerbPattern.MatchString(lower) {
			score--
		}
		if len(strings.Fields(s)) < 6 {
			score--
		}
		if score > 0 {
			candidates = append(candidates, candidate{sentence: strings.TrimSpace(s), score: score})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return len(candidates[i].sentence) > len(candidates[j].sentence)
		}
		return candidates[i].score > candidates[j].score
	})

	out := make([]string, 0, 4)
	seen := map[string]struct{}{}
	for _, c := range candidates {
		if len(out) >= 4 {
			break
		}
		s := firstWords(c.sentence, 22)
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		out = append(out, firstWords(text, 22))
	}
	return out
}

func deriveActions(name, text string) []string {
	sentences := splitSentences(text)
	out := make([]string, 0, 3)
	needle := strings.ToLower(name)
	type candidate struct {
		sentence string
		score    int
	}
	candidates := make([]candidate, 0, 8)
	for _, s := range sentences {
		lower := strings.ToLower(s)
		if strings.Contains(lower, needle) {
			score := 1
			if eventVerbPattern.MatchString(lower) {
				score += 3
			}
			if causalMarkerPattern.MatchString(lower) {
				score++
			}
			candidates = append(candidates, candidate{sentence: strings.TrimSpace(s), score: score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	for _, c := range candidates {
		if len(out) >= 3 {
			break
		}
		out = append(out, firstWords(c.sentence, 18))
	}
	if len(out) == 0 {
		out = append(out, "No direct action sentence extracted.")
	}
	return out
}

func splitSentences(text string) []string {
	parts := sentenceExtractPattern.FindAllString(text, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func deriveSummary(text string, events []string) string {
	if len(events) >= 2 {
		return firstWords(events[0], 20) + " | " + firstWords(events[1], 20)
	}
	if len(events) == 1 {
		return firstWords(events[0], 24)
	}
	return firstWords(text, 36)
}
