package backend

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"book_dashboard/internal/chunk"
	"book_dashboard/internal/slop"
	"book_dashboard/internal/timeline"
	"book_dashboard/internal/workspace"
)

func BuildDashboard(bookTitle, sourceName string, source []byte, text string, onProgress ProgressFn) DashboardData {
	started := time.Now()
	runID := "run-" + started.Format("20060102-150405.000")
	stats := RunStats{
		RunID:      runID,
		SourceName: sourceName,
		LastAction: "Analyze Manuscript",
		Status:     "RUNNING",
		StartedAt:  started.Format(time.RFC3339),
	}

	logs := []LogLine{}
	addLog := func(level, stage, message, detail string) {
		if os.Getenv("MHD_TRACE_PROGRESS") == "1" {
			fmt.Printf("%s [ANALYSIS] [%s] [%s] %s | %s\n", time.Now().Format("15:04:05.000"), level, stage, message, detail)
		}
		logs = append(logs, LogLine{
			Time:    time.Now().Format("15:04:05.000"),
			Level:   level,
			Stage:   stage,
			Message: message,
			Detail:  detail,
		})
	}

	addLog("INFO", "BOOT", "Run started", fmt.Sprintf("id=%s source=%s", runID, sourceName))
	progress(onProgress, 2, "BOOT", "Run started")
	addLog("INFO", "WORKSPACE", "Workspace initialization started", "")
	progress(onProgress, 6, "WORKSPACE", "Initializing workspace")

	workspaceRoot, err := workspace.EnsureDefault()
	if err != nil {
		addLog("RISK", "WORKSPACE", "Workspace initialization failed", err.Error())
	} else {
		addLog("INFO", "WORKSPACE", "Workspace ready", workspaceRoot)
	}

	projectPath := ""
	reportPath := ""
	if workspaceRoot != "" {
		project, projectErr := workspace.CreateProjectWithSource(workspaceRoot, bookTitle, sourceName, source)
		if projectErr != nil {
			addLog("RISK", "PROJECT", "Project initialization failed", projectErr.Error())
		} else {
			projectPath = project.Root
			reportPath = project.ReportPath
			addLog("ANALYSIS", "PROJECT", "Project created", project.Root)
		}
	}
	progress(onProgress, 12, "PROJECT", "Project initialized")

	words := len(strings.Fields(text))
	chapters := splitChapters(text)
	stats.ChapterCount = len(chapters)
	addLog("ANALYSIS", "CHAPTER", "Chapter scan completed", strconv.Itoa(len(chapters))+" chapters")
	progress(onProgress, 18, "CHAPTER", fmt.Sprintf("%d chapters detected", len(chapters)))

	segments := chunk.SlidingWindow(text, 1500, 200)
	stats.SegmentCount = len(segments)
	addLog("ANALYSIS", "INGEST", "Chunking completed", strconv.Itoa(len(segments))+" segments")
	progress(onProgress, 24, "INGEST", fmt.Sprintf("%d segments created", len(segments)))

	chapterMetrics := make([]ChapterMetric, 0, len(chapters))
	genreClassifier := newGenreClassifier()
	allGenreRaw := map[string]float64{}
	genreReasoningLines := make([]string, 0, len(chapters))
	providerHits := map[string]int{}
	for idx, ch := range chapters {
		chapterProgressStart := 24
		chapterProgressEnd := 50
		if len(chapters) > 0 {
			chapterProgressStart = 24 + int(float64(idx)/float64(len(chapters))*26.0)
			chapterProgressEnd = 24 + int(float64(idx+1)/float64(len(chapters))*26.0)
		}
		chapterProgressMid := chapterProgressStart + (chapterProgressEnd-chapterProgressStart)/2
		progress(onProgress, chapterProgressStart, "CHAPTER", fmt.Sprintf("Chapter %d/%d: classifying genre", idx+1, len(chapters)))

		genreDecision := genreClassifier.classifyChapter(ch)
		chGenres := genreDecision.Scores
		progress(onProgress, chapterProgressMid, "CHAPTER", fmt.Sprintf("Chapter %d/%d: extracting timeline markers", idx+1, len(chapters)))
		markCount := len(extractChapterMarkers(ch.text))
		topName, topScore := topGenre(chGenres)
		providerHits[genreDecision.Provider]++
		for _, g := range chGenres {
			allGenreRaw[g.Genre] += g.Score
		}
		genreReasoningLines = append(genreReasoningLines, fmt.Sprintf("Ch%d (%s): %s", ch.index, genreDecision.Provider, genreDecision.Reasoning))
		chapterMetrics = append(chapterMetrics, ChapterMetric{
			Index:          ch.index,
			Title:          ch.title,
			WordCount:      len(strings.Fields(ch.text)),
			TimelineMarks:  markCount,
			TopGenre:       topName,
			TopGenreScore:  topScore,
			GenreProvider:  genreDecision.Provider,
			GenreReasoning: genreDecision.Reasoning,
			GenreBreakdown: topNGenres(chGenres, 4),
		})
		addLog("ANALYSIS", "CHAPTER", fmt.Sprintf("Read chapter %d", ch.index), fmt.Sprintf("title=%s words=%d top_genre=%s provider=%s timeline_markers=%d", ch.title, len(strings.Fields(ch.text)), topName, genreDecision.Provider, markCount))
		progress(onProgress, chapterProgressEnd, "CHAPTER", fmt.Sprintf("Chapter %d/%d: metrics complete", idx+1, len(chapters)))
	}
	characterDictionary, chapterSummaries, chapterSummaryByID := buildCharacterDictionary(chapters)
	addLog("ANALYSIS", "DICTIONARY", "Character dictionary built", fmt.Sprintf("characters=%d chapters=%d", len(characterDictionary), len(chapterSummaries)))

	genreScores := normalizeGenreScores(allGenreRaw)
	if len(genreScores) == 0 {
		genreScores = scoreGenresForText(text)
	}
	globalGenreProvider := dominantProvider(providerHits)
	globalGenreReasoning := strings.Join(genreReasoningLines, "\n")
	if len(globalGenreReasoning) > 2400 {
		globalGenreReasoning = globalGenreReasoning[:2400]
	}

	slopReport := slop.Analyze(text)
	stats.SlopFlagCount = len(slopReport.Flags)
	addLog("ANALYSIS", "SLOP", "Statistical scan completed", fmt.Sprintf("flags=%d sd=%.2f", len(slopReport.Flags), slopReport.SentenceLengthSD))
	for _, flag := range slopReport.Flags {
		addLog("RISK", "SLOP", flag, "")
	}
	progress(onProgress, 56, "SLOP", "Statistical language pass complete")

	contradictions := detectHeuristicContradictions(chapters)
	healthIssues := buildHealthIssues(contradictions, chapterSummaryByID)
	stats.ContradictionCount = len(healthIssues)
	if len(healthIssues) > 0 {
		addLog("RISK", "FORENSICS", "Consistency contradictions found", strconv.Itoa(len(healthIssues)))
	} else {
		addLog("INFO", "FORENSICS", "No contradictions detected by heuristic pass", "")
	}
	progress(onProgress, 68, "FORENSICS", "Consistency checks complete")

	timelineEvents := buildTimeline(chapters, chapterSummaries)
	stats.TimelineCount = len(timelineEvents)
	if len(timelineEvents) == 0 {
		timelineEvents = defaultTimeline()
		addLog("INFO", "TIMELINE", "No explicit timeline markers found", "")
	} else {
		addLog("ANALYSIS", "TIMELINE", "Timeline markers extracted", strconv.Itoa(len(timelineEvents)))
	}
	progress(onProgress, 76, "TIMELINE", "Timeline reconstruction complete")

	beats, plotStructure := analyzePlotStructure(PlotInputs{
		Chapters:         chapters,
		ChapterSummaries: chapterSummaries,
		ChapterMetrics:   chapterMetrics,
		TimelineEvents:   timelineEvents,
		GenreScores:      genreScores,
		GenreProvider:    globalGenreProvider,
		GenreReasoning:   globalGenreReasoning,
	})
	addLog("ANALYSIS", "STRUCTURE", "Plot structure evaluated", fmt.Sprintf("beats=%d selected=%s provider=%s", len(beats), plotStructure.SelectedStructure, plotStructure.Provider))
	progress(onProgress, 84, "STRUCTURE", "Structural beat mapping complete")

	language := analyzeLanguage(chapters, text)
	addLog("ANALYSIS", "LANGUAGE", "Language diagnostics completed", fmt.Sprintf("spelling=%d grammar=%d age=%s", language.SpellingScore, language.GrammarScore, language.AgeCategory))
	if language.HeuristicFallback {
		addLog("RISK", "LANGUAGE", "Heuristic fallback active", fmt.Sprintf("spelling_provider=%s safety_provider=%s", language.SpellingProvider, language.SafetyProvider))
	}
	for _, note := range language.Notes {
		if strings.Contains(strings.ToLower(note), "unavailable") {
			addLog("RISK", "LANGUAGE", "Language dependency unavailable", note)
		}
	}
	progress(onProgress, 94, "LANGUAGE", "Language quality analysis complete")

	compTitles := []CompTitle{{Title: "The Silent Patient", Tier: "Blockbuster"}, {Title: "The Maidens", Tier: "Blockbuster"}, {Title: "Wrong Place Wrong Time", Tier: "Mid-list"}, {Title: "Rock Paper Scissors", Tier: "Mid-list"}, {Title: "Unknown", Tier: "Unknown"}}

	mhdScore := 100 - (len(healthIssues) * 10) - (len(slopReport.Flags) * 6) - ((100 - language.GrammarScore) / 5) - ((100 - language.SpellingScore) / 5)
	if mhdScore < 0 {
		mhdScore = 0
	}
	addLog("INFO", "SCORING", "MHD score calculated", strconv.Itoa(mhdScore))

	data := DashboardData{
		BookTitle:           bookTitle,
		WordCount:           words,
		MHDScore:            mhdScore,
		Logs:                logs,
		Contradictions:      contradictions,
		HealthIssues:        healthIssues,
		SlopReport:          slopReport,
		Timeline:            timelineEvents,
		Beats:               beats,
		PlotStructure:       plotStructure,
		GenreScores:         genreScores,
		GenreProvider:       globalGenreProvider,
		GenreReasoning:      globalGenreReasoning,
		ChapterMetrics:      chapterMetrics,
		ChapterSummaries:    chapterSummaries,
		CharacterDictionary: characterDictionary,
		ChapterCount:        len(chapters),
		CompTitles:          compTitles,
		Language:            language,
		ProjectLocation:     projectPath,
		RunStats:            stats,
	}

	stats.CompletedAt = time.Now().Format(time.RFC3339)
	stats.Status = "DONE"
	data.RunStats = stats

	if reportPath != "" {
		report := workspace.Report{
			BookTitle:      data.BookTitle,
			WordCount:      data.WordCount,
			MHDScore:       data.MHDScore,
			Contradictions: len(data.Contradictions),
			SlopFlags:      data.SlopReport.Flags,
			Analysis: map[string]any{
				"chapter_count":        data.ChapterCount,
				"run_stats":            data.RunStats,
				"system":               data.System,
				"health_issues":        data.HealthIssues,
				"language":             data.Language,
				"genre_scores":         data.GenreScores,
				"genre_provider":       data.GenreProvider,
				"genre_reasoning":      data.GenreReasoning,
				"chapter_metrics":      data.ChapterMetrics,
				"chapter_summaries":    data.ChapterSummaries,
				"character_dictionary": data.CharacterDictionary,
				"timeline":             data.Timeline,
				"beats":                data.Beats,
				"plot_structure":       data.PlotStructure,
				"slop_report":          data.SlopReport,
				"comp_titles":          data.CompTitles,
				"project_location":     data.ProjectLocation,
			},
		}
		if err := workspace.SaveReport(reportPath, report); err != nil {
			addLog("RISK", "REPORT", "report persistence failed", err.Error())
		} else {
			addLog("INFO", "REPORT", "Report persisted", reportPath)
		}
	}

	addLog("INFO", "BOOT", "Run completed", stats.RunID)
	data.Logs = logs
	progress(onProgress, 100, "DONE", "Analysis complete")
	return data
}

func defaultTimeline() []timeline.Event {
	return []timeline.Event{{TimeMarker: "Unknown", Event: "No explicit time markers detected."}}
}
