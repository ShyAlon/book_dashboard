package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"book_dashboard/internal/timeline"
)

var knownPlotStructures = []string{
	"Save the Cat",
	"Three Act",
	"Hero's Journey",
	"Fichtean Curve",
}

type plotLLMResult struct {
	SelectedStructure      string                 `json:"selected_structure"`
	Reasoning              string                 `json:"reasoning"`
	StructureProbabilities map[string]float64     `json:"structure_probabilities"`
	Beats                  []plotLLMBeatCandidate `json:"beats"`
}

type plotLLMBeatCandidate struct {
	Name         string `json:"name"`
	StartChapter int    `json:"start_chapter"`
	EndChapter   int    `json:"end_chapter"`
	IsBeat       bool   `json:"is_beat"`
	Reasoning    string `json:"reasoning"`
}

type PlotInputs struct {
	Chapters         []chapter
	ChapterSummaries []ChapterSummary
	ChapterMetrics   []ChapterMetric
	TimelineEvents   []timeline.Event
	GenreScores      []GenreScore
	GenreProvider    string
	GenreReasoning   string
}

func analyzePlotStructure(in PlotInputs) ([]BeatResult, PlotStructureReport) {
	fallbackBeats := buildBeats(in.Chapters, in.ChapterSummaries, in.ChapterMetrics, in.TimelineEvents)
	fallback := PlotStructureReport{
		Provider:          "heuristic",
		SelectedStructure: "Save the Cat",
		Probabilities: []PlotStructureProbability{
			{Name: "Save the Cat", Probability: 0.55},
			{Name: "Three Act", Probability: 0.30},
			{Name: "Fichtean Curve", Probability: 0.10},
			{Name: "Hero's Journey", Probability: 0.05},
		},
		Reasoning: "Heuristic fallback from chapter-position windows.",
	}
	if len(in.Chapters) == 0 {
		return fallbackBeats, fallback
	}

	model := strings.TrimSpace(os.Getenv("OLLAMA_STRUCTURE_MODEL"))
	if model == "" {
		model = strings.TrimSpace(os.Getenv("OLLAMA_GENRE_MODEL"))
	}
	if model == "" {
		model = strings.TrimSpace(os.Getenv("OLLAMA_LANGUAGE_MODEL"))
	}
	if model == "" {
		model = "llama3.1:8b"
	}

	client := &http.Client{Timeout: 120 * time.Second}
	prompt := buildPlotPrompt(in)
	payload := map[string]any{
		"model":   model,
		"prompt":  prompt,
		"stream":  false,
		"format":  "json",
		"options": map[string]any{"temperature": 0},
	}
	raw, _ := json.Marshal(payload)
	resp, err := client.Post(ollamaGenerateEndpoint(), "application/json", bytes.NewReader(raw))
	if err != nil {
		fallback.Reasoning += " Ollama unavailable: " + err.Error()
		return fallbackBeats, fallback
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fallback.Reasoning += fmt.Sprintf(" Ollama status=%d.", resp.StatusCode)
		return fallbackBeats, fallback
	}

	var out ollamaGenreResponse
	if err := json.Unmarshal(body, &out); err != nil {
		fallback.Reasoning += " Ollama decode failed: " + err.Error()
		return fallbackBeats, fallback
	}
	jsonText := extractJSONObject(out.Response)
	if jsonText == "" {
		fallback.Reasoning += " No JSON in response."
		return fallbackBeats, fallback
	}

	var parsed plotLLMResult
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		fallback.Reasoning += " JSON parse failed: " + err.Error()
		return fallbackBeats, fallback
	}

	beats := normalizeLLMBeats(parsed.Beats, in.Chapters, fallbackBeats)
	probs := normalizeStructureProbabilities(parsed.StructureProbabilities)
	selected := strings.TrimSpace(parsed.SelectedStructure)
	if selected == "" {
		selected = selectHighestStructure(probs, fallback.SelectedStructure)
	}
	reason := strings.TrimSpace(parsed.Reasoning)
	if reason == "" {
		reason = "LLM selected structure based on chapter-level event progression."
	}

	return beats, PlotStructureReport{
		Provider:          "ollama:" + model,
		SelectedStructure: selected,
		Probabilities:     probs,
		Reasoning:         reason,
	}
}

func buildPlotPrompt(in PlotInputs) string {
	var b strings.Builder
	b.WriteString("You are a senior story analyst. Determine which plot structure best matches the manuscript.\n")
	b.WriteString("Return JSON only with keys: selected_structure, reasoning, structure_probabilities, beats.\n")
	b.WriteString("Allowed selected_structure values: Save the Cat, Three Act, Hero's Journey, Fichtean Curve.\n")
	b.WriteString("structure_probabilities: object with exactly those four keys; values are 0..1 and sum to 1.\n")
	b.WriteString("beats: array of objects with keys name,start_chapter,end_chapter,is_beat,reasoning.\n")
	b.WriteString("Use concise reasoning tied to chapter evidence.\n\n")

	if len(in.GenreScores) > 0 {
		top := in.GenreScores[0]
		b.WriteString(fmt.Sprintf("Global genre signal: top=%s(%.2f), provider=%s\n", top.Genre, top.Score, in.GenreProvider))
	}
	if in.GenreReasoning != "" {
		b.WriteString("Genre reasoning sample: " + firstWords(in.GenreReasoning, 60) + "\n")
	}
	if len(in.TimelineEvents) > 0 {
		b.WriteString("Timeline anchors:\n")
		for i, ev := range in.TimelineEvents {
			if i >= 12 {
				break
			}
			b.WriteString(fmt.Sprintf("- %s => %s\n", ev.TimeMarker, firstWords(ev.Event, 16)))
		}
	}
	b.WriteString("\nChapter evidence:\n")

	summaryByChapter := make(map[int]ChapterSummary, len(in.ChapterSummaries))
	for _, s := range in.ChapterSummaries {
		summaryByChapter[s.Chapter] = s
	}
	metricByChapter := make(map[int]ChapterMetric, len(in.ChapterMetrics))
	for _, m := range in.ChapterMetrics {
		metricByChapter[m.Index] = m
	}

	for _, ch := range in.Chapters {
		events := deriveEvents(ch.text)
		if s, ok := summaryByChapter[ch.index]; ok && len(s.Events) > 0 {
			events = s.Events
		}
		b.WriteString(fmt.Sprintf("[Chapter %d] %s\n", ch.index, ch.title))
		if s, ok := summaryByChapter[ch.index]; ok && s.Summary != "" {
			b.WriteString("Summary: " + firstWords(s.Summary, 30) + "\n")
		} else {
			b.WriteString("Summary: " + firstWords(ch.text, 30) + "\n")
		}
		if m, ok := metricByChapter[ch.index]; ok {
			b.WriteString(fmt.Sprintf("Metrics: words=%d timeline_marks=%d top_genre=%s score=%.2f\n", m.WordCount, m.TimelineMarks, m.TopGenre, m.TopGenreScore))
		}
		if len(events) > 0 {
			if len(events) > 4 {
				events = events[:4]
			}
			b.WriteString("Events: " + strings.Join(events, " | ") + "\n")
		}
	}
	return b.String()
}

func normalizeStructureProbabilities(raw map[string]float64) []PlotStructureProbability {
	base := map[string]float64{}
	for _, name := range knownPlotStructures {
		base[name] = 0.001
	}
	for k, v := range raw {
		for _, allowed := range knownPlotStructures {
			if strings.EqualFold(strings.TrimSpace(k), allowed) && v > 0 {
				base[allowed] = v
			}
		}
	}
	sum := 0.0
	for _, v := range base {
		sum += v
	}
	out := make([]PlotStructureProbability, 0, len(base))
	for _, name := range knownPlotStructures {
		p := base[name] / sum
		out = append(out, PlotStructureProbability{Name: name, Probability: p})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Probability > out[j].Probability })
	return out
}

func selectHighestStructure(probs []PlotStructureProbability, fallback string) string {
	if len(probs) == 0 {
		return fallback
	}
	return probs[0].Name
}

func normalizeLLMBeats(raw []plotLLMBeatCandidate, chapters []chapter, fallback []BeatResult) []BeatResult {
	total := len(chapters)
	if total == 0 {
		return fallback
	}
	if len(raw) == 0 {
		return fallback
	}
	out := make([]BeatResult, 0, len(raw))
	for _, b := range raw {
		start := b.StartChapter
		end := b.EndChapter
		if start <= 0 {
			start = 1
		}
		if end <= 0 {
			end = start
		}
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		if start > end {
			start = end
		}
		name := strings.TrimSpace(b.Name)
		if name == "" {
			name = "Beat"
		}
		reason := strings.TrimSpace(b.Reasoning)
		if reason == "" {
			reason = firstWords(chapters[start-1].text, 18)
		}
		out = append(out, BeatResult{
			Name:         name,
			StartChapter: start,
			EndChapter:   end,
			IsBeat:       b.IsBeat,
			Reasoning:    reason,
		})
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
