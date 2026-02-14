package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var wordPattern = regexp.MustCompile(`[A-Za-z']+`)
var sentencePattern = regexp.MustCompile(`[.!?]+`)
var repeatedPunctPattern = regexp.MustCompile(`[!?.,]{2,}`)
var multiSpacePattern = regexp.MustCompile(`\s{2,}`)
var vowelPattern = regexp.MustCompile(`[aeiouy]`)
var hardClusterPattern = regexp.MustCompile(`[bcdfghjklmnpqrstvwxz]{6,}`)

func analyzeLanguage(chapters []chapter, text string) LanguageReport {
	base := heuristicLanguage(text)
	base.SpellingProvider = "heuristic"
	base.SafetyProvider = "heuristic"

	ltReport, ltErr := analyzeWithLanguageTool(chapters)
	if ltErr == nil {
		base.SpellingScore = ltReport.SpellingScore
		base.GrammarScore = ltReport.GrammarScore
		base.ReadabilityScore = ltReport.ReadabilityScore
		base.ProfanityScore = max(base.ProfanityScore, ltReport.ProfanityScore)
		base.SpellingProvider = "LanguageTool"
		base.Notes = append(base.Notes, ltReport.Notes...)
	} else {
		base.Notes = append(base.Notes, "Spelling & grammar provider: heuristic fallback")
		base.Notes = append(base.Notes, "LanguageTool unavailable: "+ltErr.Error())
	}

	safety, safetyErr := analyzeSafetyWithOllama(chapters, text)
	if safetyErr == nil {
		base.AgeCategory = safety.AgeCategory
		base.ProfanityScore = safety.ProfanityScore
		base.ExplicitScore = safety.ExplicitScore
		base.ViolenceScore = safety.ViolenceScore
		base.SafetyProvider = "Ollama"
		base.ProfanityInstances = max(base.ProfanityInstances, safety.ProfanityInstances)
		base.ExplicitInstances = max(base.ExplicitInstances, safety.ExplicitInstances)
		if safety.SafetyRationale != "" {
			base.Notes = append(base.Notes, "Ollama safety rationale: "+safety.SafetyRationale)
		}
	} else {
		base.Notes = append(base.Notes, "Ollama safety unavailable: "+safetyErr.Error())
	}
	base.HeuristicFallback = strings.EqualFold(base.SpellingProvider, "heuristic") || strings.EqualFold(base.SafetyProvider, "heuristic")
	if base.HeuristicFallback {
		base.Notes = append([]string{"Warning: heuristic fallback active. Verify dependency startup logs."}, base.Notes...)
	}

	return base
}

func heuristicLanguage(text string) LanguageReport {
	words := wordPattern.FindAllString(strings.ToLower(text), -1)
	sentences := sentencePattern.Split(text, -1)
	wordCount := len(words)
	if wordCount == 0 {
		return LanguageReport{SpellingScore: 0, GrammarScore: 0, ReadabilityScore: 0, AgeCategory: "Unknown"}
	}

	profanityWords := map[string]struct{}{"fuck": {}, "shit": {}, "damn": {}, "bitch": {}, "asshole": {}, "bastard": {}}
	explicitWords := map[string]struct{}{"sex": {}, "nude": {}, "naked": {}, "erotic": {}, "orgasm": {}, "penetration": {}}
	violenceWords := map[string]struct{}{"blood": {}, "kill": {}, "murder": {}, "gun": {}, "knife": {}, "stab": {}, "violent": {}}

	profanityCount := 0
	explicitCount := 0
	violenceCount := 0
	suspiciousSpelling := 0
	for _, w := range words {
		if _, ok := profanityWords[w]; ok {
			profanityCount++
		}
		if _, ok := explicitWords[w]; ok {
			explicitCount++
		}
		if _, ok := violenceWords[w]; ok {
			violenceCount++
		}
		if looksMisspelled(w) {
			suspiciousSpelling++
		}
	}

	lowerStartIssues := 0
	longSentenceIssues := 0
	totalSentenceWords := 0
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		tokens := wordPattern.FindAllString(s, -1)
		totalSentenceWords += len(tokens)
		if len(tokens) > 35 {
			longSentenceIssues++
		}
		if len(s) > 0 {
			r := []rune(strings.TrimSpace(s))
			if len(r) > 0 && r[0] >= 'a' && r[0] <= 'z' {
				lowerStartIssues++
			}
		}
	}

	repeatedPunct := len(repeatedPunctPattern.FindAllString(text, -1))
	multiSpace := len(multiSpacePattern.FindAllString(text, -1))
	grammarIssues := lowerStartIssues + longSentenceIssues + repeatedPunct + multiSpace

	spellingScore := clamp100(100 - (suspiciousSpelling * 300 / max(1, wordCount)))
	grammarScore := clamp100(100 - (grammarIssues * 200 / max(1, len(sentences))))

	avgSentenceLen := 16.0
	if len(sentences) > 0 {
		avgSentenceLen = float64(totalSentenceWords) / float64(max(1, len(sentences)))
	}
	readabilityPenalty := int(abs(avgSentenceLen-18.0) * 3.2)
	readabilityScore := clamp100(100 - readabilityPenalty)

	profanityScore := clamp100(profanityCount * 1000 / max(1, wordCount))
	explicitScore := clamp100(explicitCount * 1200 / max(1, wordCount))
	violenceScore := clamp100(violenceCount * 900 / max(1, wordCount))

	ageCategory := "All Ages"
	switch {
	case explicitScore >= 8 || profanityScore >= 10:
		ageCategory = "Adult 18+"
	case explicitScore >= 4 || profanityScore >= 6 || violenceScore >= 8:
		ageCategory = "Mature 16+"
	case profanityScore >= 2 || violenceScore >= 4:
		ageCategory = "Teen 13+"
	}

	notes := []string{
		fmt.Sprintf("Average sentence length: %.1f words", avgSentenceLen),
		fmt.Sprintf("Potential spelling anomalies: %d", suspiciousSpelling),
		fmt.Sprintf("Grammar issue heuristics triggered: %d", grammarIssues),
	}

	return LanguageReport{
		SpellingScore:      spellingScore,
		GrammarScore:       grammarScore,
		ReadabilityScore:   readabilityScore,
		AgeCategory:        ageCategory,
		ProfanityScore:     profanityScore,
		ExplicitScore:      explicitScore,
		ViolenceScore:      violenceScore,
		ProfanityInstances: profanityCount,
		ExplicitInstances:  explicitCount,
		Notes:              notes,
	}
}

type languageToolResponse struct {
	Matches []struct {
		Rule struct {
			Category struct {
				ID string `json:"id"`
			} `json:"category"`
		} `json:"rule"`
	} `json:"matches"`
}

func analyzeWithLanguageTool(chapters []chapter) (LanguageReport, error) {
	endpoint := os.Getenv("LANGUAGETOOL_URL")
	if endpoint == "" {
		endpoint = "http://localhost:8010/v2/check"
	}
	client := &http.Client{Timeout: 45 * time.Second}

	grammarIssues := 0
	spellingIssues := 0
	styleIssues := 0
	totalWords := 0
	for _, ch := range chapters {
		totalWords += len(strings.Fields(ch.text))
		vals := url.Values{}
		vals.Set("language", "en-US")
		vals.Set("text", ch.text)
		req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(vals.Encode()))
		if err != nil {
			return LanguageReport{}, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return LanguageReport{}, err
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return LanguageReport{}, fmt.Errorf("status %d", resp.StatusCode)
		}
		var lt languageToolResponse
		if err := json.Unmarshal(body, &lt); err != nil {
			return LanguageReport{}, err
		}
		for _, m := range lt.Matches {
			cat := strings.ToUpper(m.Rule.Category.ID)
			switch {
			case strings.Contains(cat, "TYPOS") || strings.Contains(cat, "SPELL"):
				spellingIssues++
			case strings.Contains(cat, "STYLE"):
				styleIssues++
			default:
				grammarIssues++
			}
		}
	}

	if totalWords == 0 {
		totalWords = 1
	}
	spellingScore := clamp100(100 - (spellingIssues * 700 / totalWords))
	grammarScore := clamp100(100 - ((grammarIssues + styleIssues) * 900 / totalWords))
	readabilityScore := clamp100((spellingScore + grammarScore) / 2)

	return LanguageReport{
		SpellingScore:    spellingScore,
		GrammarScore:     grammarScore,
		ReadabilityScore: readabilityScore,
		ProfanityScore:   0,
		Notes: []string{
			"Spelling & grammar provider: LanguageTool",
			fmt.Sprintf("LanguageTool issues: grammar=%d spelling=%d style=%d", grammarIssues, spellingIssues, styleIssues),
		},
	}, nil
}

type ollamaResponse struct {
	Response string `json:"response"`
}

type safetyResult struct {
	AgeCategory        string `json:"age_category"`
	ProfanityScore     int    `json:"profanity_score"`
	ExplicitScore      int    `json:"explicit_score"`
	ViolenceScore      int    `json:"violence_score"`
	ProfanityInstances int    `json:"profanity_instances"`
	ExplicitInstances  int    `json:"explicit_instances"`
	SafetyRationale    string `json:"safety_rationale"`
}

func analyzeSafetyWithOllama(chapters []chapter, text string) (safetyResult, error) {
	endpoint := ollamaGenerateEndpoint()
	model := os.Getenv("OLLAMA_LANGUAGE_MODEL")
	if model == "" {
		model = "llama3.1:8b"
	}

	sample := buildSafetySample(chapters, text)
	prompt := "You are a strict content classifier for book publishing. Return JSON only with keys: age_category, profanity_score, explicit_score, violence_score, profanity_instances, explicit_instances, safety_rationale. Scores are 0-100." + "\n\nTEXT:\n" + sample
	payload := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
		"options": map[string]any{
			"temperature": 0,
		},
	}
	raw, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(raw))
	if err != nil {
		return safetyResult{}, err
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return safetyResult{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out ollamaResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return safetyResult{}, err
	}
	jsonText := extractJSONObject(out.Response)
	if jsonText == "" {
		snippet := strings.TrimSpace(out.Response)
		if len(snippet) > 220 {
			snippet = snippet[:220] + "..."
		}
		if snippet == "" {
			return safetyResult{}, fmt.Errorf("no JSON in model response (empty response)")
		}
		return safetyResult{}, fmt.Errorf("no JSON in model response: %q", snippet)
	}
	var sr safetyResult
	if err := json.Unmarshal([]byte(jsonText), &sr); err != nil {
		return safetyResult{}, err
	}
	sr.ProfanityScore = clamp100(sr.ProfanityScore)
	sr.ExplicitScore = clamp100(sr.ExplicitScore)
	sr.ViolenceScore = clamp100(sr.ViolenceScore)
	if sr.AgeCategory == "" {
		sr.AgeCategory = "Unknown"
	}
	return sr, nil
}

func buildSafetySample(chapters []chapter, text string) string {
	if len(chapters) == 0 {
		return firstWords(text, 2500)
	}
	parts := make([]string, 0, len(chapters))
	for i, ch := range chapters {
		if i >= 12 {
			break
		}
		parts = append(parts, fmt.Sprintf("[Ch %d %s] %s", ch.index, ch.title, firstWords(ch.text, 180)))
	}
	return strings.Join(parts, "\n")
}

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Handle fenced markdown payloads first.
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 3 {
			s = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func looksMisspelled(w string) bool {
	if len(w) <= 2 {
		return false
	}
	if vowelPattern.FindString(w) == "" {
		return true
	}
	return hardClusterPattern.FindString(w) != ""
}
