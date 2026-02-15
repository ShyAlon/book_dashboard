package aidetect

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Input struct {
	DocumentID string `json:"document_id"`
	Text       string `json:"text"`
	Language   string `json:"language"`
}

type ErrorEntry struct {
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Retryable bool   `json:"retryable"`
}

type SpanTrace struct {
	Name       string `json:"name"`
	DurationMs int64  `json:"duration_ms"`
	Status     string `json:"status"`
}

type EvidenceSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Evidence struct {
	Type    string         `json:"type"`
	Summary string         `json:"summary"`
	Spans   []EvidenceSpan `json:"spans"`
}

type DuplicationSignal struct {
	Score    *float64   `json:"score"`
	Evidence []Evidence `json:"evidence"`
}

type ScalarSignal struct {
	Score *float64 `json:"score"`
}

type WindowSignals struct {
	Duplication  DuplicationSignal `json:"duplication"`
	LMSmoothness ScalarSignal      `json:"lm_smoothness"`
	StyleUniform ScalarSignal      `json:"style_uniformity"`
	PolishCliche ScalarSignal      `json:"polish_cliche"`
	LanguageTool ScalarSignal      `json:"language_tool"`
}

type WindowReport struct {
	WindowID    string        `json:"window_id"`
	StartWord   int           `json:"start_word"`
	EndWord     int           `json:"end_word"`
	PAI         float64       `json:"p_ai"`
	Confidence  float64       `json:"confidence"`
	Signals     WindowSignals `json:"signals"`
	TopEvidence []Evidence    `json:"top_evidence"`
}

type Report struct {
	DocumentID    string         `json:"document_id"`
	PAIDoc        *float64       `json:"p_ai_doc"`
	AICoverageEst *float64       `json:"ai_coverage_est"`
	PAIMax        *float64       `json:"p_ai_max"`
	ConfidenceDoc *float64       `json:"confidence_doc"`
	Flags         []string       `json:"flags"`
	Windows       []WindowReport `json:"windows"`
	Errors        []ErrorEntry   `json:"errors"`
	Traces        []SpanTrace    `json:"traces"`
	WordCount     int            `json:"word_count"`
}

type Config struct {
	WindowWords           int
	StrideWords           int
	DupNGramN             int
	NearDupThreshold      float64
	DupOverrideMinWords   int
	CoverageTrigger       float64
	Bias                  float64
	EnableLanguageTool    bool
	EnableLMSmoothness    bool
	LanguageToolTimeoutMs int
	LanguageToolStride    int
	LanguageToolMaxWindow int
	LanguageToolMaxFails  int
	LMSmoothnessTimeoutMs int
}

type LanguageToolScorer interface {
	ScoreWindow(ctx context.Context, text string) (float64, error)
}

type LMSmoothnessScorer interface {
	ScoreWindow(ctx context.Context, text string) (float64, error)
}

type Logger interface {
	Log(level, stage, message, detail string)
}

func DefaultConfig() Config {
	return Config{
		WindowWords:           getenvInt("AI_WINDOW_WORDS", 900),
		StrideWords:           getenvInt("AI_STRIDE_WORDS", 450),
		DupNGramN:             getenvInt("AI_DUP_NGRAM_N", 10),
		NearDupThreshold:      getenvFloat("AI_NEAR_DUP_THRESHOLD", 0.18),
		DupOverrideMinWords:   getenvInt("AI_DUP_OVERRIDE_MIN_WORDS", 250),
		CoverageTrigger:       getenvFloat("AI_COVERAGE_TRIGGER", 0.03),
		Bias:                  getenvFloat("AI_BIAS", -0.20),
		EnableLanguageTool:    getenvBool("AI_ENABLE_LANGUAGE_TOOL", true),
		EnableLMSmoothness:    getenvBool("AI_ENABLE_LM_SMOOTHNESS", false),
		LanguageToolTimeoutMs: getenvInt("AI_LANGUAGETOOL_TIMEOUT_MS", 5000),
		LanguageToolStride:    getenvInt("AI_LANGUAGETOOL_STRIDE", 3),
		LanguageToolMaxWindow: getenvInt("AI_LANGUAGETOOL_MAX_WINDOWS", 24),
		LanguageToolMaxFails:  getenvInt("AI_LANGUAGETOOL_MAX_FAILS", 3),
		LMSmoothnessTimeoutMs: getenvInt("AI_LM_TIMEOUT_MS", 5000),
	}
}

func Analyze(in Input, cfg Config, lt LanguageToolScorer, lm LMSmoothnessScorer, logger Logger) Report {
	report := Report{
		DocumentID: in.DocumentID,
		Flags:      []string{},
		Windows:    []WindowReport{},
		Errors:     []ErrorEntry{},
		Traces:     []SpanTrace{},
	}
	if strings.TrimSpace(in.Language) != "" && !strings.EqualFold(in.Language, "en") {
		report.Errors = append(report.Errors, ErrorEntry{
			Stage:     "bad_input",
			Message:   "language must be en",
			Type:      "bad_input",
			Retryable: false,
		})
		return report
	}
	startAll := time.Now()

	var normalized string
	withSpan(&report, "normalize_text", func() error {
		normalized = normalizeText(in.Text)
		return nil
	})

	words := splitWords(normalized)
	report.WordCount = len(words)

	var windows []wordWindow
	withSpan(&report, "segment_windows", func() error {
		windows = segmentWindows(words, cfg.WindowWords, cfg.StrideWords)
		if len(windows) == 0 {
			return fmt.Errorf("segmentation produced no windows")
		}
		return nil
	})
	if len(report.Errors) > 0 && report.PAIDoc == nil {
		return report
	}

	if logger != nil {
		logger.Log("ANALYSIS", "AI", "AI detection run started", fmt.Sprintf("document_id=%s words=%d windows=%d", in.DocumentID, len(words), len(windows)))
	}

	paraDupMap := map[string][]paragraphLoc{}
	shingleSets := make([]map[string]struct{}, len(windows))
	withSpan(&report, "duplication_scan", func() error {
		paraDupMap = buildParagraphHashIndex(normalized, words)
		for i, w := range windows {
			shingleSets[i] = shingleSet(words[w.Start:w.End], cfg.DupNGramN)
		}
		return nil
	})

	ltUnavailable := false
	lmUnavailable := false
	dupSignals := make([]float64, len(windows))
	styleSignals := make([]float64, len(windows))
	polishSignals := make([]float64, len(windows))
	ltSignals := make([]*float64, len(windows))
	lmSignals := make([]*float64, len(windows))
	windowEvidences := make([][]Evidence, len(windows))
	overrideLongDup := make([]bool, len(windows))
	overrideDupWords := make([]int, len(windows))

	for i, w := range windows {
		windowWords := words[w.Start:w.End]
		windowText := strings.Join(windowWords, " ")
		dupScore, dupEvidence, longestDupWords := windowDupSignal(i, w, windows, windowWords, paraDupMap, shingleSets, cfg.NearDupThreshold, cfg.WindowWords)
		dupSignals[i] = dupScore
		windowEvidences[i] = dupEvidence
		overrideDupWords[i] = longestDupWords
		if longestDupWords >= cfg.DupOverrideMinWords {
			overrideLongDup[i] = true
		}

		styleSignals[i] = styleUniformityScore(windowText)
		polishSignals[i] = polishClicheScore(windowWords, windowText)
	}

	withSpan(&report, "language_tool_run", func() error {
		if !cfg.EnableLanguageTool || lt == nil {
			ltUnavailable = true
			report.Errors = append(report.Errors, ErrorEntry{
				Stage:     "language_tool_run",
				Message:   "language tool scorer unavailable",
				Type:      "tool_unavailable",
				Retryable: true,
			})
			return nil
		}
		failCount := 0
		failType := ""
		failMessage := ""
		successCount := 0
		attemptedCount := 0
		consecutiveFailures := 0
		for i, w := range windows {
			if !shouldRunLanguageTool(i, len(windows), cfg, attemptedCount) {
				continue
			}
			attemptedCount++
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.LanguageToolTimeoutMs)*time.Millisecond)
			score, err := lt.ScoreWindow(ctx, strings.Join(words[w.Start:w.End], " "))
			cancel()
			if err != nil {
				ltUnavailable = true
				failCount++
				consecutiveFailures++
				if failType == "" {
					failType = classifyToolErr(err)
				}
				if failMessage == "" {
					failMessage = err.Error()
				}
				if consecutiveFailures >= maxInt(1, cfg.LanguageToolMaxFails) {
					break
				}
				continue
			}
			s := clamp01(score)
			ltSignals[i] = &s
			successCount++
			consecutiveFailures = 0
		}
		if failCount > 0 {
			msg := failMessage
			if msg == "" {
				msg = "language tool scorer failed"
			}
			report.Errors = append(report.Errors, ErrorEntry{
				Stage:     "language_tool_run",
				Message:   fmt.Sprintf("%s (%d/%d sampled windows failed)", msg, failCount, maxInt(1, attemptedCount)),
				Type:      defaultIfEmpty(failType, "exception"),
				Retryable: true,
			})
		}
		return nil
	})

	withSpan(&report, "lm_scoring_run", func() error {
		if !cfg.EnableLMSmoothness || lm == nil {
			lmUnavailable = true
			report.Errors = append(report.Errors, ErrorEntry{
				Stage:     "lm_scoring_run",
				Message:   "lm smoothness scorer unavailable",
				Type:      "tool_unavailable",
				Retryable: true,
			})
			return nil
		}
		failCount := 0
		failType := ""
		failMessage := ""
		for i, w := range windows {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.LMSmoothnessTimeoutMs)*time.Millisecond)
			score, err := lm.ScoreWindow(ctx, strings.Join(words[w.Start:w.End], " "))
			cancel()
			if err != nil {
				lmUnavailable = true
				failCount++
				if failType == "" {
					failType = classifyToolErr(err)
				}
				if failMessage == "" {
					failMessage = err.Error()
				}
				if failCount >= 3 {
					break
				}
				continue
			}
			s := clamp01(score)
			lmSignals[i] = &s
		}
		if failCount > 0 {
			msg := failMessage
			if msg == "" {
				msg = "lm scorer failed"
			}
			report.Errors = append(report.Errors, ErrorEntry{
				Stage:     "lm_scoring_run",
				Message:   fmt.Sprintf("%s (%d/%d windows failed)", msg, failCount, len(windows)),
				Type:      defaultIfEmpty(failType, "exception"),
				Retryable: true,
			})
		}
		return nil
	})

	withSpan(&report, "score_windows", func() error {
		for i, w := range windows {
			weights := signalWeights(!lmUnavailable && lmSignals[i] != nil)
			signals := WindowSignals{
				Duplication: DuplicationSignal{
					Score:    floatPtr(dupSignals[i]),
					Evidence: windowEvidences[i],
				},
				LMSmoothness: ScalarSignal{Score: lmSignals[i]},
				StyleUniform: ScalarSignal{Score: floatPtr(styleSignals[i])},
				PolishCliche: ScalarSignal{Score: floatPtr(polishSignals[i])},
				LanguageTool: ScalarSignal{Score: ltSignals[i]},
			}

			sum := weights.Duplication*dupSignals[i] + weights.StyleUniform*styleSignals[i] + weights.PolishCliche*polishSignals[i]
			if lmSignals[i] != nil {
				sum += weights.LMSmoothness * *lmSignals[i]
			}
			if ltSignals[i] != nil {
				sum += weights.LanguageTool * *ltSignals[i]
			}
			p := sigmoid(sum + cfg.Bias)

			conf := 0.6
			if dupSignals[i] > 0.0 || len(windowEvidences[i]) > 0 {
				conf += 0.15
			}
			agree := 0
			if dupSignals[i] > 0.6 {
				agree++
			}
			if styleSignals[i] > 0.6 {
				agree++
			}
			if polishSignals[i] > 0.6 {
				agree++
			}
			if lmSignals[i] != nil && *lmSignals[i] > 0.6 {
				agree++
			}
			if ltSignals[i] != nil && *ltSignals[i] > 0.6 {
				agree++
			}
			if agree >= 3 {
				conf += 0.10
			}
			if lmSignals[i] == nil {
				conf -= 0.20
			}
			if w.End-w.Start < 600 {
				conf -= 0.10
			}
			conf = clamp01(conf)

			topEvidence := topEvidence(windowEvidences[i], 3)
			if overrideLongDup[i] {
				p = math.Max(p, 0.90)
				conf = math.Max(conf, 0.80)
				topEvidence = append(topEvidence, Evidence{
					Type:    "duplication",
					Summary: "long duplicate span",
					Spans:   []EvidenceSpan{{Start: w.Start, End: minInt(w.End, w.Start+overrideDupWords[i])}},
				})
			}

			report.Windows = append(report.Windows, WindowReport{
				WindowID:    fmt.Sprintf("w-%03d", i),
				StartWord:   w.Start,
				EndWord:     w.End,
				PAI:         clamp01(p),
				Confidence:  conf,
				Signals:     signals,
				TopEvidence: topEvidence,
			})
		}
		return nil
	})

	withSpan(&report, "aggregate_document", func() error {
		if len(report.Windows) == 0 {
			report.Errors = append(report.Errors, ErrorEntry{
				Stage:     "aggregate_document",
				Message:   "no windows to aggregate",
				Type:      "exception",
				Retryable: false,
			})
			return nil
		}
		maxP := 0.0
		covNum := 0.0
		covDen := 0.0
		type sc struct {
			p  float64
			c  float64
			pw float64
		}
		top := make([]sc, 0, len(report.Windows))

		for _, w := range report.Windows {
			pw := clamp01(w.PAI * w.Confidence)
			if w.PAI > maxP {
				maxP = w.PAI
			}
			length := float64(maxInt(1, w.EndWord-w.StartWord))
			covNum += w.PAI * w.Confidence * length
			covDen += length
			top = append(top, sc{p: w.PAI, c: w.Confidence, pw: pw})
		}
		coverage := 0.0
		if covDen > 0 {
			coverage = covNum / covDen
		}
		sort.Slice(top, func(i, j int) bool { return top[i].pw > top[j].pw })
		limit := minInt(10, len(top))
		topPWMean := 0.0
		cn := 0.0
		cd := 0.0
		for i := 0; i < limit; i++ {
			topPWMean += top[i].pw
			cn += top[i].c
			cd += 1.0
		}
		if limit > 0 {
			topPWMean /= float64(limit)
		}
		confDoc := 0.0
		if cd > 0 {
			confDoc = cn / cd
		}
		coverageSignal := 0.0
		if coverage > cfg.CoverageTrigger {
			den := maxFloat(0.01, 0.35-cfg.CoverageTrigger)
			coverageSignal = clamp01((coverage - cfg.CoverageTrigger) / den)
		}
		// Conservative doc aggregation to avoid saturating on long manuscripts with many medium windows.
		pDoc := clamp01(0.50*topPWMean + 0.35*maxP + 0.15*coverageSignal)

		if maxP >= 0.85 {
			report.Flags = append(report.Flags, "ai_chunk_detected")
		}
		if coverage >= 0.35 {
			report.Flags = append(report.Flags, "widespread_ai_signal")
		}
		if hasDupFlag(report.Windows) {
			report.Flags = append(report.Flags, "possible_stitching")
		}
		if coverage >= cfg.CoverageTrigger {
			report.Flags = append(report.Flags, "coverage_trigger_exceeded")
		}

		report.PAIDoc = floatPtr(clamp01(pDoc))
		report.AICoverageEst = floatPtr(clamp01(coverage))
		report.PAIMax = floatPtr(clamp01(maxP))
		report.ConfidenceDoc = floatPtr(clamp01(confDoc))
		return nil
	})

	if logger != nil {
		errCount := len(report.Errors)
		logger.Log("ANALYSIS", "AI", "AI detection run completed", fmt.Sprintf("document_id=%s words=%d windows=%d errors=%d p_ai_doc=%.3f coverage=%.3f p_ai_max=%.3f duration_ms=%d lm_available=%t lt_available=%t",
			in.DocumentID, report.WordCount, len(report.Windows), errCount, deref(report.PAIDoc), deref(report.AICoverageEst), deref(report.PAIMax),
			time.Since(startAll).Milliseconds(), !lmUnavailable, !ltUnavailable))
	}
	return report
}

type wordWindow struct {
	Start int
	End   int
}

type paragraphLoc struct {
	Start int
	End   int
}

type weights struct {
	Duplication  float64
	LMSmoothness float64
	StyleUniform float64
	PolishCliche float64
	LanguageTool float64
}

func signalWeights(lmAvailable bool) weights {
	if lmAvailable {
		return weights{
			Duplication:  0.35,
			LMSmoothness: 0.30,
			StyleUniform: 0.20,
			PolishCliche: 0.10,
			LanguageTool: 0.05,
		}
	}
	return weights{
		Duplication:  0.50,
		LMSmoothness: 0.0,
		StyleUniform: 0.30,
		PolishCliche: 0.15,
		LanguageTool: 0.05,
	}
}

func withSpan(report *Report, name string, fn func() error) {
	start := time.Now()
	status := "ok"
	if err := fn(); err != nil {
		status = "error"
		report.Errors = append(report.Errors, ErrorEntry{
			Stage:     name,
			Message:   err.Error(),
			Type:      "exception",
			Retryable: false,
		})
	}
	report.Traces = append(report.Traces, SpanTrace{
		Name:       name,
		DurationMs: time.Since(start).Milliseconds(),
		Status:     status,
	})
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ToLower(text)
	text = punctStripper.ReplaceAllString(text, " ")
	text = multiSpace.ReplaceAllString(strings.TrimSpace(text), " ")
	return text
}

var punctStripper = regexp.MustCompile(`[^a-z0-9\s\.\?\!\n]+`)
var multiSpace = regexp.MustCompile(`[ \t]+`)
var multiNewLine = regexp.MustCompile(`\n{3,}`)
var sentenceSplit = regexp.MustCompile(`[.!?]+`)
var wordFinder = regexp.MustCompile(`[a-z0-9']+`)

func splitWords(text string) []string {
	return wordFinder.FindAllString(text, -1)
}

func segmentWindows(words []string, windowWords, strideWords int) []wordWindow {
	if windowWords <= 0 {
		windowWords = 900
	}
	if strideWords <= 0 {
		strideWords = windowWords / 2
	}
	if len(words) == 0 {
		return []wordWindow{{Start: 0, End: 0}}
	}
	if len(words) <= windowWords {
		return []wordWindow{{Start: 0, End: len(words)}}
	}
	out := []wordWindow{}
	for start := 0; start < len(words); start += strideWords {
		end := start + windowWords
		if end > len(words) {
			end = len(words)
		}
		out = append(out, wordWindow{Start: start, End: end})
		if end == len(words) {
			break
		}
	}
	return out
}

func buildParagraphHashIndex(normalized string, words []string) map[string][]paragraphLoc {
	idx := map[string][]paragraphLoc{}
	rawParas := strings.Split(multiNewLine.ReplaceAllString(normalized, "\n\n"), "\n\n")
	cursor := 0
	for _, p := range rawParas {
		pw := splitWords(p)
		if len(pw) < 40 {
			cursor += len(pw)
			continue
		}
		h := sha1Hash(strings.Join(pw, " "))
		start := cursor
		end := cursor + len(pw)
		idx[h] = append(idx[h], paragraphLoc{Start: start, End: minInt(end, len(words))})
		cursor = end
	}
	return idx
}

func sha1Hash(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func shingleSet(words []string, n int) map[string]struct{} {
	if n <= 0 {
		n = 10
	}
	out := map[string]struct{}{}
	if len(words) < n {
		return out
	}
	for i := 0; i+n <= len(words); i++ {
		key := sha1Hash(strings.Join(words[i:i+n], " "))
		out[key] = struct{}{}
	}
	return out
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union <= 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func windowDupSignal(i int, w wordWindow, windows []wordWindow, windowWords []string, paraDupMap map[string][]paragraphLoc, shingleSets []map[string]struct{}, nearDupThreshold float64, windowSize int) (float64, []Evidence, int) {
	evidence := []Evidence{}
	dupParaCount := 0
	longestDupWords := 0

	// Exact paragraph duplication evidence
	for _, locs := range paraDupMap {
		if len(locs) < 2 {
			continue
		}
		for _, loc := range locs {
			if rangesOverlap(w.Start, w.End, loc.Start, loc.End) {
				dupParaCount++
				spanStart := maxInt(w.Start, loc.Start)
				spanEnd := minInt(w.End, loc.End)
				spanLen := maxInt(0, spanEnd-spanStart)
				if spanLen > longestDupWords {
					longestDupWords = spanLen
				}
				evidence = append(evidence, Evidence{
					Type:    "duplication",
					Summary: "exact duplicated paragraph hash",
					Spans:   []EvidenceSpan{{Start: spanStart, End: spanEnd}},
				})
				break
			}
		}
	}

	maxJac := 0.0
	maxJacWindow := -1
	for j := range windows {
		if i == j || absInt(i-j) < 2 {
			continue
		}
		jac := jaccard(shingleSets[i], shingleSets[j])
		if jac > maxJac {
			maxJac = jac
			maxJacWindow = j
		}
	}
	if maxJac >= nearDupThreshold {
		approxSpan := int(maxJac * float64(windowSize))
		if approxSpan > longestDupWords {
			longestDupWords = approxSpan
		}
		evidence = append(evidence, Evidence{
			Type:    "duplication",
			Summary: fmt.Sprintf("near-duplication with %s (jaccard=%.2f)", windowID(maxJacWindow), maxJac),
			Spans:   []EvidenceSpan{{Start: w.Start, End: minInt(w.End, w.Start+maxInt(1, approxSpan))}},
		})
	}

	dupScore := clamp01(0.25*math.Min(1.0, float64(dupParaCount)/3.0) + 0.75*clamp01(maxJac/0.35))
	return dupScore, evidence, longestDupWords
}

func styleUniformityScore(windowText string) float64 {
	sentences := sentenceSplit.Split(windowText, -1)
	lengths := []float64{}
	commas := 0
	semis := 0
	dashes := strings.Count(windowText, "—")
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		commas += strings.Count(s, ",")
		semis += strings.Count(s, ";")
		lengths = append(lengths, float64(len(splitWords(s))))
	}
	if len(lengths) == 0 {
		return 0
	}
	mean, sd := meanStd(lengths)
	punctRate := float64(commas+semis+dashes) / float64(maxInt(1, len(splitWords(windowText))))
	mattr := mattrScore(splitWords(windowText), 200)
	// Low sd + stable punctuation + lower lexical variation => more uniform.
	a := clamp01((8.0 - sd) / 8.0)
	b := clamp01((0.04 - punctRate) / 0.04)
	c := clamp01((0.62 - mattr) / 0.30)
	_ = mean
	return clamp01(0.55*a + 0.20*b + 0.25*c)
}

func polishClicheScore(words []string, windowText string) float64 {
	if len(words) == 0 {
		return 0
	}
	intensifiers := 0
	for _, w := range words {
		if _, ok := intensifierLexicon[w]; ok {
			intensifiers++
		}
	}
	intDensity := float64(intensifiers) / float64(len(words)) * 1000.0
	frameHits := 0
	for _, re := range stockFramePatterns {
		frameHits += len(re.FindAllStringIndex(windowText, -1))
	}
	sentenceCount := maxInt(1, len(sentenceSplit.Split(windowText, -1)))
	frameRate := float64(frameHits) / float64(sentenceCount) * 1000.0
	return clamp01(0.6*clamp01(intDensity/22.0) + 0.4*clamp01(frameRate/45.0))
}

func mattrScore(words []string, n int) float64 {
	if len(words) == 0 {
		return 0
	}
	if len(words) <= n {
		seen := map[string]struct{}{}
		for _, w := range words {
			seen[w] = struct{}{}
		}
		return float64(len(seen)) / float64(len(words))
	}
	sum := 0.0
	count := 0
	for i := 0; i+n <= len(words); i += n / 2 {
		seen := map[string]struct{}{}
		for _, w := range words[i : i+n] {
			seen[w] = struct{}{}
		}
		sum += float64(len(seen)) / float64(n)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func meanStd(values []float64) (mean, sd float64) {
	if len(values) == 0 {
		return 0, 0
	}
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	if len(values) == 1 {
		return mean, 0
	}
	variance := 0.0
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(values))
	return mean, math.Sqrt(variance)
}

func topEvidence(in []Evidence, limit int) []Evidence {
	if len(in) <= limit {
		return in
	}
	return in[:limit]
}

func hasDupFlag(windows []WindowReport) bool {
	for _, w := range windows {
		if len(w.Signals.Duplication.Evidence) > 0 {
			return true
		}
	}
	return false
}

func classifyToolErr(err error) string {
	if err == nil {
		return "exception"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline exceeded"):
		return "timeout"
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "unavailable"):
		return "tool_unavailable"
	default:
		return "exception"
	}
}

func windowID(i int) string {
	if i < 0 {
		return "w-unknown"
	}
	return fmt.Sprintf("w-%03d", i)
}

func rangesOverlap(aStart, aEnd, bStart, bEnd int) bool {
	return aStart < bEnd && bStart < aEnd
}

func sigmoid(z float64) float64 {
	return 1.0 / (1.0 + math.Exp(-z))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func floatPtr(v float64) *float64 {
	v = clamp01(v)
	return &v
}

func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shouldRunLanguageTool(index, total int, cfg Config, attempted int) bool {
	if total <= 0 {
		return false
	}
	maxWindows := cfg.LanguageToolMaxWindow
	if maxWindows <= 0 {
		maxWindows = total
	}
	if attempted >= maxWindows {
		return false
	}
	if index == 0 || index == total-1 {
		return true
	}
	stride := cfg.LanguageToolStride
	if stride <= 1 {
		return true
	}
	return index%stride == 0
}

func getenvInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getenvFloat(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

func getenvBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if raw == "" {
		return fallback
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

var intensifierLexicon = map[string]struct{}{
	"very": {}, "extremely": {}, "utterly": {}, "absolutely": {}, "perfectly": {}, "incredibly": {}, "deeply": {}, "completely": {},
	"terrifying": {}, "chilling": {}, "unmistakable": {}, "frantic": {}, "desperate": {}, "inevitable": {}, "unforgiving": {},
}

var stockFramePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bthe unmistakable\b`),
	regexp.MustCompile(`\bthe final\b`),
	regexp.MustCompile(`\bthe only\b`),
	regexp.MustCompile(`\bthe world\b`),
	regexp.MustCompile(`\ba data point\b`),
	regexp.MustCompile(`\bthe protocol\b`),
}
