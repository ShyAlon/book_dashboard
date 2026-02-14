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

// A compact subset of common trigrams used as a proxy for repetitive language.
var commonTrigrams = map[string]struct{}{
	"one of the":      {},
	"as well as":      {},
	"out of the":      {},
	"it was a":        {},
	"to be a":         {},
	"in the same":     {},
	"at the same":     {},
	"was one of":      {},
	"this is a":       {},
	"there was a":     {},
	"in order to":     {},
	"the end of":      {},
	"a lot of":        {},
	"the rest of":     {},
	"it is a":         {},
	"for the first":   {},
	"the beginning of": {},
}

type Report struct {
	Monotone           bool
	MeanSentenceLength float64
	SentenceLengthSD   float64
	BadWordDensity     float64
	LowOriginality     bool
	Flags              []string
}

func Analyze(text string) Report {
	words := tokenize(text)
	sd, mean := sentenceLengthStats(text)
	density := badWordDensity(words)
	lowOriginality := trigramCommonness(words) >= 0.90

	flags := make([]string, 0, 3)
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

	return Report{
		Monotone:           monotone,
		MeanSentenceLength: mean,
		SentenceLengthSD:   sd,
		BadWordDensity:     density,
		LowOriginality:     lowOriginality,
		Flags:              flags,
	}
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
	sentences := sentenceEnd.Split(text, -1)
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
