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
)

var genreKeywords = map[string][]string{
	"Thriller": {"murder", "chase", "conspiracy", "killer", "threat", "secret", "escape", "investigation", "spy"},
	"Mystery":  {"clue", "detective", "evidence", "suspect", "crime", "missing", "investigate", "alibi"},
	"Romance":  {"love", "kiss", "heart", "desire", "romance", "wedding", "passion", "affair"},
	"Fantasy":  {"magic", "kingdom", "sword", "dragon", "spell", "prophecy", "portal", "myth"},
	"Sci-Fi":   {"ship", "quantum", "android", "orbit", "signal", "colony", "lab", "protocol", "variant"},
	"Literary": {"memory", "silence", "family", "identity", "childhood", "grief", "dream", "voice"},
}

var genreOrder = []string{"Thriller", "Mystery", "Romance", "Fantasy", "Sci-Fi", "Literary"}

func scoreGenresForText(text string) []GenreScore {
	lower := strings.ToLower(text)
	raw := map[string]float64{}
	for genre, words := range genreKeywords {
		total := 1.0
		for _, w := range words {
			total += float64(strings.Count(lower, w))
		}
		raw[genre] = total
	}
	return normalizeGenreScores(raw)
}

type genreDecision struct {
	Provider  string
	Reasoning string
	Scores    []GenreScore
}

type genreClassifier struct {
	endpoint string
	model    string
	client   *http.Client

	consecutiveFailures int
	lastErr             string
}

func newGenreClassifier() *genreClassifier {
	model := strings.TrimSpace(os.Getenv("OLLAMA_GENRE_MODEL"))
	if model == "" {
		model = strings.TrimSpace(os.Getenv("OLLAMA_LANGUAGE_MODEL"))
	}
	if model == "" {
		model = "llama3.1:8b"
	}
	return &genreClassifier{
		endpoint: ollamaGenerateEndpoint(),
		model:    model,
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (g *genreClassifier) classifyChapter(ch chapter) genreDecision {
	// Keep trying Ollama per chapter; only short-circuit after repeated hard failures.
	if g.consecutiveFailures < 3 {
		sample := buildGenreSample(ch.text)
		for attempt := 0; attempt < 3; attempt++ {
			if llm, err := g.classifyWithOllama(sample); err == nil {
				g.consecutiveFailures = 0
				return genreDecision{
					Provider:  "ollama:" + g.model,
					Reasoning: llm.Reasoning,
					Scores:    llm.Scores,
				}
			} else {
				g.lastErr = err.Error()
			}
		}
		g.consecutiveFailures++
	}

	scores := scoreGenresForText(ch.text)
	topName, topScore := topGenre(scores)
	reason := fmt.Sprintf("Heuristic fallback using keyword frequencies across full chapter text (top genre=%s %.2f).", topName, topScore)
	if g.lastErr != "" {
		reason += " Ollama unavailable: " + g.lastErr
	}
	return genreDecision{
		Provider:  "heuristic",
		Reasoning: reason,
		Scores:    scores,
	}
}

type ollamaGenreResponse struct {
	Response string `json:"response"`
}

type genreLLMResult struct {
	TopGenre   string             `json:"top_genre"`
	Reasoning  string             `json:"reasoning"`
	GenreScore map[string]float64 `json:"genre_scores"`
}

func (g *genreClassifier) classifyWithOllama(sample string) (genreDecision, error) {
	prompt := "You are a senior fiction editor. Classify manuscript excerpt genre mixture." +
		" Return JSON only with keys: top_genre, reasoning, genre_scores." +
		" genre_scores must include exactly these keys with 0-1 floats that sum to 1: Thriller, Mystery, Romance, Fantasy, Sci-Fi, Literary." +
		" reasoning should be concise and cite observed signals.\n\nTEXT:\n" + sample

	payload := map[string]any{
		"model":   g.model,
		"prompt":  prompt,
		"stream":  false,
		"format":  "json",
		"options": map[string]any{"temperature": 0},
	}
	raw, _ := json.Marshal(payload)
	resp, err := g.client.Post(g.endpoint, "application/json", bytes.NewReader(raw))
	if err != nil {
		return genreDecision{}, err
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return genreDecision{}, fmt.Errorf("status %d", resp.StatusCode)
	}

	var out ollamaGenreResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return genreDecision{}, err
	}
	jsonText := extractJSONObject(out.Response)
	if jsonText == "" {
		return genreDecision{}, fmt.Errorf("no JSON in model response")
	}

	var parsed genreLLMResult
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return genreDecision{}, err
	}
	scores := normalizeGenreScores(fillMissingGenres(parsed.GenreScore))
	if len(scores) == 0 {
		return genreDecision{}, fmt.Errorf("empty genre scores")
	}

	reason := strings.TrimSpace(parsed.Reasoning)
	if reason == "" {
		topName, _ := topGenre(scores)
		reason = "Model classification favored " + topName + " from sampled chapter windows."
	}

	return genreDecision{
		Provider:  "ollama:" + g.model,
		Reasoning: reason,
		Scores:    scores,
	}, nil
}

func fillMissingGenres(raw map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for _, k := range genreOrder {
		out[k] = 0.001
	}
	for k, v := range raw {
		if _, ok := out[k]; ok && v > 0 {
			out[k] = v
		}
	}
	return out
}

func ollamaGenerateEndpoint() string {
	base := strings.TrimSpace(os.Getenv("OLLAMA_URL"))
	if base == "" {
		return "http://127.0.0.1:11434/api/generate"
	}
	if strings.Contains(base, "/api/generate") {
		return base
	}
	return strings.TrimSuffix(base, "/") + "/api/generate"
}

func buildGenreSample(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	if len(words) <= 900 {
		return strings.Join(words, " ")
	}

	window := 300
	head := strings.Join(words[:window], " ")
	midStart := len(words)/2 - window/2
	if midStart < 0 {
		midStart = 0
	}
	midEnd := midStart + window
	if midEnd > len(words) {
		midEnd = len(words)
	}
	mid := strings.Join(words[midStart:midEnd], " ")
	tailStart := len(words) - window
	if tailStart < 0 {
		tailStart = 0
	}
	tail := strings.Join(words[tailStart:], " ")
	return "[START]\n" + head + "\n[MIDDLE]\n" + mid + "\n[END]\n" + tail
}

func normalizeGenreScores(raw map[string]float64) []GenreScore {
	if len(raw) == 0 {
		return nil
	}
	sum := 0.0
	for _, v := range raw {
		sum += v
	}
	out := make([]GenreScore, 0, len(raw))
	for k, v := range raw {
		score := 0.0
		if sum > 0 {
			score = v / sum
		}
		out = append(out, GenreScore{Genre: k, Score: score})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func topGenre(scores []GenreScore) (string, float64) {
	if len(scores) == 0 {
		return "Unknown", 0
	}
	return scores[0].Genre, scores[0].Score
}

func topNGenres(scores []GenreScore, n int) []GenreScore {
	if n >= len(scores) {
		return scores
	}
	return scores[:n]
}

func dominantProvider(hits map[string]int) string {
	if len(hits) == 0 {
		return "heuristic"
	}
	name := ""
	max := -1
	for k, v := range hits {
		if v > max {
			max = v
			name = k
		}
	}
	if name == "" {
		return "heuristic"
	}
	return name
}
