package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"book_dashboard/internal/aidetect"
)

type aiLogger struct {
	add func(level, stage, message, detail string)
}

func (l aiLogger) Log(level, stage, message, detail string) {
	if l.add != nil {
		l.add(level, stage, message, detail)
	}
}

type aiLanguageToolScorer struct {
	endpoint string
	client   *http.Client
}

func newAILanguageToolScorer() aidetect.LanguageToolScorer {
	endpoint := strings.TrimSpace(os.Getenv("LANGUAGETOOL_URL"))
	if endpoint == "" {
		endpoint = "http://localhost:8010/v2/check"
	}
	return aiLanguageToolScorer{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 8 * time.Second},
	}
}

func (s aiLanguageToolScorer) ScoreWindow(ctx context.Context, text string) (float64, error) {
	vals := url.Values{}
	vals.Set("language", "en-US")
	vals.Set("text", text)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, strings.NewReader(vals.Encode()))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	var parsed languageToolResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	wordCount := len(strings.Fields(text))
	if wordCount == 0 {
		return 0, nil
	}
	issuesPer1K := float64(len(parsed.Matches)) * 1000.0 / float64(wordCount)
	// Cleaner prose is a weak AI-likelihood signal; keep this conservative.
	return clampAIDetect(1.0 - (issuesPer1K / 45.0)), nil
}

func clampAIDetect(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
