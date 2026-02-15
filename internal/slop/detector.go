package slop

import (
	_ "embed"
	"encoding/json"
	"math"
	"regexp"
	"strings"
)

//go:embed bad_words.json
var badWordsJSON []byte

var sentenceEnd = regexp.MustCompile(`[.!?]+`)
var wordPattern = regexp.MustCompile(`[A-Za-z']+`)
var paragraphSplit = regexp.MustCompile(`\n\s*\n+`)
var chapterHeadingPattern = regexp.MustCompile(`(?im)^\s*(chapter|ch\.?)\s+([0-9ivxlcdm]+)\s*[:\-]?\s*(.+)?$`)
var nonWordPattern = regexp.MustCompile(`[^a-z0-9\s]+`)
var multiSpacePattern = regexp.MustCompile(`\s+`)

// A compact subset of common trigrams used as a proxy for repetitive language.
var commonTrigrams = map[string]struct{}{
	"one of the":       {},
	"as well as":       {},
	"out of the":       {},
	"it was a":         {},
	"to be a":          {},
	"in the same":      {},
	"at the same":      {},
	"was one of":       {},
	"this is a":        {},
	"there was a":      {},
	"in order to":      {},
	"the end of":       {},
	"a lot of":         {},
	"the rest of":      {},
	"it is a":          {},
	"for the first":    {},
	"the beginning of": {},
}

type Report struct {
	Monotone                    bool
	MeanSentenceLength          float64
	SentenceLengthSD            float64
	BadWordDensity              float64
	LowOriginality              bool
	RepeatedBlockCount          int
	MaxBlockRepeat              int
	VerbatimDuplicationCoverage float64
	RepeatedPhraseCoverage      float64
	DramaticDensity             float64
	DramaticDensitySD           float64
	ExpansionMarkerCount        int
	OptimizationMarkerCount     int
	AISuspicionScore            int
	LikelyAIGenerated           bool
	Flags                       []string
}

func Analyze(text string) Report {
	words := tokenize(text)
	sentences := splitSentences(text)
	sd, mean := sentenceLengthStats(text)
	density := badWordDensity(words)
	lowOriginality := trigramCommonness(words) >= 0.90
	dupCoverage, repeatedBlockCount, maxRepeat := repeatedParagraphStats(text, len(words))
	repeatedPhraseCoverage := repeatedShingleCoverage(words, 12)
	dramaticDensity, dramaticDensitySD := dramaticProfile(sentences)
	expansionMarkerCount := expansionMarkerCount(text)
	optimizationMarkerCount := optimizationMarkerCount(text)

	flags := make([]string, 0, 7)
	monotone := sd < 4.0
	if monotone {
		flags = append(flags, "Monotone: sentence-length variability is unusually low")
	}
	if density > 0.015 {
		flags = append(flags, "High red-flag vocabulary density")
	}
	if lowOriginality {
		flags = append(flags, "Low Originality: trigram profile is overly common")
	}
	if dupCoverage >= 0.12 || maxRepeat >= 3 {
		flags = append(flags, "Verbatim repetition: large blocks are duplicated across the manuscript")
	}
	if repeatedPhraseCoverage >= 0.10 {
		flags = append(flags, "Repeated phrase lattice: long n-grams recur too frequently")
	}
	if dramaticDensity >= 0.055 && dramaticDensitySD <= 0.04 {
		flags = append(flags, "Uniform dramatic saturation: stylistic intensity is unusually constant")
	}
	if expansionMarkerCount > 0 {
		flags = append(flags, "Mechanical expansion markers detected (e.g., elaborated/duplicated chapter structure)")
	}

	aiScore := aiSuspicionScore(dupCoverage, repeatedPhraseCoverage, repeatedBlockCount, maxRepeat, dramaticDensity, dramaticDensitySD, expansionMarkerCount, optimizationMarkerCount)
	likelyAIGenerated := aiScore >= 45
	if likelyAIGenerated {
		flags = append(flags, "AI-generation risk is high based on repetition and style-structure signals")
	}

	return Report{
		Monotone:                    monotone,
		MeanSentenceLength:          mean,
		SentenceLengthSD:            sd,
		BadWordDensity:              density,
		LowOriginality:              lowOriginality,
		RepeatedBlockCount:          repeatedBlockCount,
		MaxBlockRepeat:              maxRepeat,
		VerbatimDuplicationCoverage: dupCoverage,
		RepeatedPhraseCoverage:      repeatedPhraseCoverage,
		DramaticDensity:             dramaticDensity,
		DramaticDensitySD:           dramaticDensitySD,
		ExpansionMarkerCount:        expansionMarkerCount,
		OptimizationMarkerCount:     optimizationMarkerCount,
		AISuspicionScore:            aiScore,
		LikelyAIGenerated:           likelyAIGenerated,
		Flags:                       flags,
	}
}

func splitSentences(text string) []string {
	raw := sentenceEnd.Split(text, -1)
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func badWordDensity(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	var raw []string
	_ = json.Unmarshal(badWordsJSON, &raw)
	bad := make(map[string]struct{}, len(raw))
	for _, w := range raw {
		bad[strings.ToLower(strings.TrimSpace(w))] = struct{}{}
	}
	matches := 0
	for _, w := range words {
		if _, ok := bad[w]; ok {
			matches++
		}
	}
	return float64(matches) / float64(len(words))
}

func sentenceLengthStats(text string) (sd float64, mean float64) {
	sentences := splitSentences(text)
	lengths := make([]float64, 0, len(sentences))
	for _, s := range sentences {
		count := float64(len(tokenize(s)))
		if count > 0 {
			lengths = append(lengths, count)
		}
	}
	if len(lengths) == 0 {
		return 0, 0
	}

	total := 0.0
	for _, l := range lengths {
		total += l
	}
	mean = total / float64(len(lengths))
	if len(lengths) == 1 {
		return 0, mean
	}

	var variance float64
	for _, l := range lengths {
		d := l - mean
		variance += d * d
	}
	variance /= float64(len(lengths))
	return math.Sqrt(variance), mean
}

func trigramCommonness(words []string) float64 {
	if len(words) < 3 {
		return 0
	}
	total := 0
	common := 0
	for i := 0; i+2 < len(words); i++ {
		total++
		tri := words[i] + " " + words[i+1] + " " + words[i+2]
		if _, ok := commonTrigrams[tri]; ok {
			common++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(common) / float64(total)
}

func tokenize(text string) []string {
	parts := wordPattern.FindAllString(strings.ToLower(text), -1)
	return parts
}

func repeatedParagraphStats(text string, totalWords int) (coverage float64, repeatedBlocks int, maxRepeat int) {
	paras := paragraphSplit.Split(text, -1)
	type paraStat struct {
		count int
		words int
	}
	stats := map[string]*paraStat{}
	for _, p := range paras {
		tokens := tokenize(p)
		if len(tokens) < 35 {
			continue
		}
		key := normalizeBlock(p)
		if key == "" {
			continue
		}
		item := stats[key]
		if item == nil {
			item = &paraStat{words: len(tokens)}
			stats[key] = item
		}
		item.count++
		if len(tokens) > item.words {
			item.words = len(tokens)
		}
	}
	dupWords := 0
	for _, st := range stats {
		if st.count > 1 {
			repeatedBlocks++
			dupWords += st.words * st.count
			if st.count > maxRepeat {
				maxRepeat = st.count
			}
		}
	}
	if totalWords <= 0 {
		return 0, repeatedBlocks, maxRepeat
	}
	return float64(dupWords) / float64(totalWords), repeatedBlocks, maxRepeat
}

func repeatedShingleCoverage(words []string, size int) float64 {
	if size <= 1 || len(words) < size {
		return 0
	}
	total := len(words) - size + 1
	counts := make(map[string]int, total)
	for i := 0; i+size <= len(words); i++ {
		key := strings.Join(words[i:i+size], " ")
		counts[key]++
	}
	dup := 0
	for _, c := range counts {
		if c > 1 {
			dup += c
		}
	}
	return float64(dup) / float64(total)
}

func normalizeBlock(s string) string {
	s = strings.ToLower(s)
	s = nonWordPattern.ReplaceAllString(s, " ")
	s = multiSpacePattern.ReplaceAllString(strings.TrimSpace(s), " ")
	return s
}

func dramaticProfile(sentences []string) (mean float64, sd float64) {
	if len(sentences) == 0 {
		return 0, 0
	}
	densities := make([]float64, 0, len(sentences))
	for _, sentence := range sentences {
		tokens := tokenize(sentence)
		if len(tokens) < 4 {
			continue
		}
		hits := 0
		for _, t := range tokens {
			if _, ok := dramaticLexicon[t]; ok {
				hits++
			}
		}
		densities = append(densities, float64(hits)/float64(len(tokens)))
	}
	if len(densities) == 0 {
		return 0, 0
	}
	total := 0.0
	for _, v := range densities {
		total += v
	}
	mean = total / float64(len(densities))
	if len(densities) == 1 {
		return mean, 0
	}
	var variance float64
	for _, v := range densities {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(densities))
	return mean, math.Sqrt(variance)
}

func expansionMarkerCount(text string) int {
	lower := strings.ToLower(text)
	count := 0
	for _, marker := range expansionMarkers {
		count += strings.Count(lower, marker)
	}
	count += repeatedChapterHeadingCount(text)
	return count
}

func repeatedChapterHeadingCount(text string) int {
	matches := chapterHeadingPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0
	}
	seen := map[string]int{}
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		title := normalizeBlock(m[3])
		if len(title) < 10 {
			continue
		}
		seen[title]++
	}
	reused := 0
	for _, count := range seen {
		if count > 1 {
			reused++
		}
	}
	return reused
}

func optimizationMarkerCount(text string) int {
	lower := strings.ToLower(text)
	count := 0
	for _, marker := range optimizationMarkers {
		count += strings.Count(lower, marker)
	}
	return count
}

func aiSuspicionScore(dupCoverage, repeatedPhraseCoverage float64, repeatedBlocks, maxRepeat int, dramaticDensity, dramaticDensitySD float64, expansionCount, optimizationCount int) int {
	score := 0
	score += minInt(55, int(dupCoverage*180.0))
	score += minInt(35, int(repeatedPhraseCoverage*120.0))
	if repeatedBlocks > 0 {
		score += minInt(15, repeatedBlocks*3+maxInt(0, maxRepeat-1)*2)
	}
	if dramaticDensity >= 0.05 && dramaticDensitySD <= 0.05 {
		score += minInt(15, int(dramaticDensity*220.0))
	}
	score += minInt(10, expansionCount*3)
	score += minInt(5, optimizationCount)
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
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

var dramaticLexicon = map[string]struct{}{
	"blood": {}, "fear": {}, "grave": {}, "ghost": {}, "tomb": {}, "dark": {}, "hollow": {}, "fatal": {}, "doom": {}, "despair": {},
	"metallic": {}, "sterile": {}, "iron": {}, "claw": {}, "clawed": {}, "claws": {}, "scream": {}, "screamed": {}, "shattered": {}, "ruin": {},
	"infinite": {}, "eternal": {}, "perfect": {}, "perfection": {}, "obedience": {}, "compliance": {}, "unforgiving": {}, "abyss": {}, "haunting": {}, "haunted": {},
}

var expansionMarkers = []string{
	"elaborated version",
	"expanded version",
	"revised version",
	"chapter rewrite",
	"version 2",
	"version ii",
}

var optimizationMarkers = []string{
	"efficiency",
	"compliance",
	"optimization",
	"directive",
	"flagged",
}
