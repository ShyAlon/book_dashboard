package prompts

import (
	"fmt"
	"strings"
)

const StructureAnalysisTemplate = `SYSTEM: You are a senior literary editor.
INPUT: %s
TASK: Determine if this section represents the "%s" of the story.
CRITERIA:
- Catalyst: A life-changing event.
- Midpoint: A false victory or false defeat.
- All is Lost: A moment of total hopelessness.
OUTPUT: JSON { "is_beat": boolean, "reasoning": string }`

const CompTitlesTemplate = `SYSTEM: You are a book market analyst.
INPUT: %s
TASK: List 5 comparable titles published in the last 10 years.
CONSTRAINT: Do not invent titles. If unsure, state "Unknown".
OUTPUT: JSON array.`

func StructurePrompt(chapterSummary, beatName string) string {
	return strings.TrimSpace(fmt.Sprintf(StructureAnalysisTemplate, chapterSummary, beatName))
}

func CompTitlesPrompt(synopsis string) string {
	return strings.TrimSpace(fmt.Sprintf(CompTitlesTemplate, synopsis))
}
