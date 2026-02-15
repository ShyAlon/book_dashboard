package backend

import (
	"time"

	"book_dashboard/internal/aidetect"
	"book_dashboard/internal/slop"
)

func InitialDashboard() DashboardData {
	return DashboardData{
		BookTitle:           "No Manuscript Loaded",
		WordCount:           0,
		MHDScore:            0,
		Logs:                []LogLine{{Time: time.Now().Format("15:04:05.000"), Level: "INFO", Stage: "BOOT", Message: "Ready", Detail: "Use Pick File or Analyze File to start."}},
		Contradictions:      nil,
		HealthIssues:        nil,
		AIReport:            aidetect.Report{Flags: []string{}, Windows: []aidetect.WindowReport{}, Errors: []aidetect.ErrorEntry{}, Traces: []aidetect.SpanTrace{}},
		SlopReport:          slop.Report{},
		Timeline:            nil,
		Beats:               nil,
		PlotStructure:       PlotStructureReport{},
		GenreScores:         nil,
		GenreProvider:       "",
		GenreReasoning:      "",
		ChapterMetrics:      nil,
		ChapterSummaries:    nil,
		CharacterDictionary: nil,
		ChapterCount:        0,
		CompTitles:          nil,
		Language:            LanguageReport{AgeCategory: "Unknown"},
		ProjectLocation:     "",
		RunStats: RunStats{
			Status:     "IDLE",
			RunID:      "",
			LastAction: "Awaiting input",
		},
		System: SystemDiagnostics{
			Overall:      "IDLE",
			Initializing: true,
			Ollama:       ServiceStatus{Name: "ollama"},
			LanguageTool: ServiceStatus{Name: "languagetool"},
			Traces:       []ServiceTrace{},
		},
	}
}
