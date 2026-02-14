package backend

import (
	"book_dashboard/internal/forensics"
	"book_dashboard/internal/slop"
	"book_dashboard/internal/timeline"
)

type DashboardData struct {
	BookTitle           string                    `json:"bookTitle"`
	WordCount           int                       `json:"wordCount"`
	MHDScore            int                       `json:"mhdScore"`
	Logs                []LogLine                 `json:"logs"`
	Contradictions      []forensics.Contradiction `json:"contradictions"`
	HealthIssues        []HealthIssue             `json:"healthIssues"`
	SlopReport          slop.Report               `json:"slopReport"`
	Timeline            []timeline.Event          `json:"timeline"`
	Beats               []BeatResult              `json:"beats"`
	PlotStructure       PlotStructureReport       `json:"plotStructure"`
	GenreScores         []GenreScore              `json:"genreScores"`
	GenreProvider       string                    `json:"genreProvider"`
	GenreReasoning      string                    `json:"genreReasoning"`
	ChapterMetrics      []ChapterMetric           `json:"chapterMetrics"`
	ChapterSummaries    []ChapterSummary          `json:"chapterSummaries"`
	CharacterDictionary []CharacterEntry          `json:"characterDictionary"`
	ChapterCount        int                       `json:"chapterCount"`
	CompTitles          []CompTitle               `json:"compTitles"`
	Language            LanguageReport            `json:"language"`
	ProjectLocation     string                    `json:"projectLocation"`
	RunStats            RunStats                  `json:"runStats"`
	System              SystemDiagnostics         `json:"system"`
}

type LogLine struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Stage   string `json:"stage"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

type BeatResult struct {
	Name         string `json:"name"`
	StartChapter int    `json:"startChapter"`
	EndChapter   int    `json:"endChapter"`
	IsBeat       bool   `json:"isBeat"`
	Reasoning    string `json:"reasoning"`
}

type PlotStructureProbability struct {
	Name        string  `json:"name"`
	Probability float64 `json:"probability"`
}

type PlotStructureReport struct {
	Provider          string                     `json:"provider"`
	SelectedStructure string                     `json:"selectedStructure"`
	Probabilities     []PlotStructureProbability `json:"probabilities"`
	Reasoning         string                     `json:"reasoning"`
}

type GenreScore struct {
	Genre string  `json:"genre"`
	Score float64 `json:"score"`
}

type ChapterMetric struct {
	Index          int          `json:"index"`
	Title          string       `json:"title"`
	WordCount      int          `json:"wordCount"`
	TimelineMarks  int          `json:"timelineMarks"`
	TopGenre       string       `json:"topGenre"`
	TopGenreScore  float64      `json:"topGenreScore"`
	GenreProvider  string       `json:"genreProvider"`
	GenreReasoning string       `json:"genreReasoning"`
	GenreBreakdown []GenreScore `json:"genreBreakdown"`
}

type CompTitle struct {
	Title string `json:"title"`
	Tier  string `json:"tier"`
}

type ChapterSummary struct {
	Chapter int      `json:"chapter"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Events  []string `json:"events"`
}

type CharacterChapterRecord struct {
	Chapter int      `json:"chapter"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Actions []string `json:"actions"`
	Events  []string `json:"events"`
}

type CharacterEntry struct {
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	FirstSeenChapter int                      `json:"firstSeenChapter"`
	LastSeenChapter  int                      `json:"lastSeenChapter"`
	TotalMentions    int                      `json:"totalMentions"`
	Chapters         []CharacterChapterRecord `json:"chapters"`
}

type HealthIssue struct {
	ID            string `json:"id"`
	Entity        string `json:"entity"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
	ChapterA      int    `json:"chapterA"`
	ChapterB      int    `json:"chapterB"`
	ContextA      string `json:"contextA"`
	ContextB      string `json:"contextB"`
	DictionaryRef string `json:"dictionaryRef"`
}

type LanguageReport struct {
	SpellingScore      int      `json:"spellingScore"`
	GrammarScore       int      `json:"grammarScore"`
	ReadabilityScore   int      `json:"readabilityScore"`
	AgeCategory        string   `json:"ageCategory"`
	SpellingProvider   string   `json:"spellingProvider"`
	SafetyProvider     string   `json:"safetyProvider"`
	HeuristicFallback  bool     `json:"heuristicFallback"`
	ProfanityScore     int      `json:"profanityScore"`
	ExplicitScore      int      `json:"explicitScore"`
	ViolenceScore      int      `json:"violenceScore"`
	ProfanityInstances int      `json:"profanityInstances"`
	ExplicitInstances  int      `json:"explicitInstances"`
	Notes              []string `json:"notes"`
}

type RunStats struct {
	RunID              string `json:"runId"`
	SourceName         string `json:"sourceName"`
	LastAction         string `json:"lastAction"`
	Status             string `json:"status"`
	StartedAt          string `json:"startedAt"`
	CompletedAt        string `json:"completedAt"`
	ChapterCount       int    `json:"chapterCount"`
	SegmentCount       int    `json:"segmentCount"`
	TimelineCount      int    `json:"timelineCount"`
	ContradictionCount int    `json:"contradictionCount"`
	SlopFlagCount      int    `json:"slopFlagCount"`
}

type SystemDiagnostics struct {
	Overall      string         `json:"overall"`
	Initializing bool           `json:"initializing"`
	Ollama       ServiceStatus  `json:"ollama"`
	LanguageTool ServiceStatus  `json:"languageTool"`
	Traces       []ServiceTrace `json:"traces"`
}

type ServiceStatus struct {
	Name           string `json:"name"`
	Running        bool   `json:"running"`
	Ready          bool   `json:"ready"`
	LastError      string `json:"lastError"`
	Detail         string `json:"detail"`
	Missing        bool   `json:"missing"`
	InstallHint    string `json:"installHint"`
	InstallCommand string `json:"installCommand"`
}

type ServiceTrace struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

type chapter struct {
	index int
	title string
	text  string
}
