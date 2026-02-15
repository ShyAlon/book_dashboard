export type LogLine = { time: string; level: string; stage: string; message: string; detail: string };
export type GenreScore = { genre: string; score: number };

export type ChapterMetric = {
  index: number;
  title: string;
  wordCount: number;
  timelineMarks: number;
  topGenre: string;
  topGenreScore: number;
  genreBreakdown: GenreScore[];
};

export type Contradiction = {
  EntityName: string;
  Attribute: string;
  ValueA: string;
  ValueB: string;
  ChapterA: number;
  ChapterB: number;
  Description: string;
  Severity: string;
};

export type HealthIssue = {
  id: string;
  entity: string;
  severity: string;
  description: string;
  chapterA: number;
  chapterB: number;
  contextA: string;
  contextB: string;
  dictionaryRef: string;
};

export type ChapterSummary = {
  chapter: number;
  title: string;
  summary: string;
  events: string[];
};

export type CharacterChapterRecord = {
  chapter: number;
  title: string;
  summary: string;
  actions: string[];
  events: string[];
};

export type CharacterEntry = {
  name: string;
  description: string;
  firstSeenChapter: number;
  lastSeenChapter: number;
  totalMentions: number;
  chapters: CharacterChapterRecord[];
};

export type AIDetectionReport = {
  document_id: string;
  p_ai_doc: number | null;
  ai_coverage_est: number | null;
  p_ai_max: number | null;
  confidence_doc: number | null;
  flags: string[];
  errors: Array<{ stage: string; message: string; type: string; retryable: boolean }>;
  traces: Array<{ name: string; duration_ms: number; status: string }>;
  windows: Array<{
    window_id: string;
    start_word: number;
    end_word: number;
    p_ai: number;
    confidence: number;
    signals: {
      duplication: { score: number | null; evidence: Array<{ type: string; summary: string; spans: Array<{ start: number; end: number }> }> };
      lm_smoothness: { score: number | null };
      style_uniformity: { score: number | null };
      polish_cliche: { score: number | null };
      language_tool: { score: number | null };
    };
    top_evidence: Array<{ type: string; summary: string; spans: Array<{ start: number; end: number }> }>;
  }>;
  word_count: number;
};

export type DashboardData = {
  bookTitle: string;
  wordCount: number;
  mhdScore: number;
  logs: LogLine[];
  contradictions: Contradiction[];
  healthIssues: HealthIssue[];
  aiReport: AIDetectionReport;
  slopReport: {
    Monotone: boolean;
    MeanSentenceLength: number;
    SentenceLengthSD: number;
    BadWordDensity: number;
    LowOriginality: boolean;
    RepeatedBlockCount: number;
    MaxBlockRepeat: number;
    VerbatimDuplicationCoverage: number;
    RepeatedPhraseCoverage: number;
    DramaticDensity: number;
    DramaticDensitySD: number;
    ExpansionMarkerCount: number;
    OptimizationMarkerCount: number;
    AISuspicionScore: number;
    LikelyAIGenerated: boolean;
    Flags: string[];
  };
  timeline: Array<{ time_marker: string; event: string }>;
  beats: Array<{ name: string; startChapter: number; endChapter: number; isBeat: boolean; reasoning: string }>;
  genreScores: GenreScore[];
  chapterMetrics: ChapterMetric[];
  chapterSummaries: ChapterSummary[];
  characterDictionary: CharacterEntry[];
  chapterCount: number;
  compTitles: Array<{ title: string; tier: string }>;
  language: {
    spellingScore: number;
    grammarScore: number;
    readabilityScore: number;
    ageCategory: string;
    spellingProvider: string;
    safetyProvider: string;
    heuristicFallback: boolean;
    profanityScore: number;
    explicitScore: number;
    violenceScore: number;
    profanityInstances: number;
    explicitInstances: number;
    notes: string[];
  };
  projectLocation: string;
  system: {
    overall: string;
    initializing: boolean;
    ollama: { name: string; running: boolean; ready: boolean; lastError: string; detail: string; missing: boolean; installHint: string; installCommand: string };
    languageTool: { name: string; running: boolean; ready: boolean; lastError: string; detail: string; missing: boolean; installHint: string; installCommand: string };
    traces: Array<{ time: string; level: string; message: string; detail: string }>;
  };
  runStats: {
    runId: string;
    sourceName: string;
    lastAction: string;
    status: string;
    startedAt: string;
    completedAt: string;
    chapterCount: number;
    segmentCount: number;
    timelineCount: number;
    contradictionCount: number;
    slopFlagCount: number;
  };
};

export type TabName = "ai" | "structure" | "market" | "language" | "dictionary";
export type LogFilter = "ALL" | "INFO" | "ANALYSIS" | "RISK";

export const emptyData: DashboardData = {
  bookTitle: "Untitled",
  wordCount: 0,
  mhdScore: 0,
  logs: [],
  contradictions: [],
  healthIssues: [],
  aiReport: {
    document_id: "",
    p_ai_doc: null,
    ai_coverage_est: null,
    p_ai_max: null,
    confidence_doc: null,
    flags: [],
    errors: [],
    traces: [],
    windows: [],
    word_count: 0,
  },
  slopReport: {
    Monotone: false,
    MeanSentenceLength: 0,
    SentenceLengthSD: 0,
    BadWordDensity: 0,
    LowOriginality: false,
    RepeatedBlockCount: 0,
    MaxBlockRepeat: 0,
    VerbatimDuplicationCoverage: 0,
    RepeatedPhraseCoverage: 0,
    DramaticDensity: 0,
    DramaticDensitySD: 0,
    ExpansionMarkerCount: 0,
    OptimizationMarkerCount: 0,
    AISuspicionScore: 0,
    LikelyAIGenerated: false,
    Flags: [],
  },
  timeline: [],
  beats: [],
  genreScores: [],
  chapterMetrics: [],
  chapterSummaries: [],
  characterDictionary: [],
  chapterCount: 0,
  compTitles: [],
  language: {
    spellingScore: 0,
    grammarScore: 0,
    readabilityScore: 0,
    ageCategory: "Unknown",
    spellingProvider: "",
    safetyProvider: "",
    heuristicFallback: false,
    profanityScore: 0,
    explicitScore: 0,
    violenceScore: 0,
    profanityInstances: 0,
    explicitInstances: 0,
    notes: [],
  },
  projectLocation: "",
  system: {
    overall: "IDLE",
    initializing: true,
    ollama: { name: "ollama", running: false, ready: false, lastError: "", detail: "", missing: false, installHint: "", installCommand: "" },
    languageTool: { name: "languagetool", running: false, ready: false, lastError: "", detail: "", missing: false, installHint: "", installCommand: "" },
    traces: [],
  },
  runStats: {
    runId: "",
    sourceName: "",
    lastAction: "",
    status: "",
    startedAt: "",
    completedAt: "",
    chapterCount: 0,
    segmentCount: 0,
    timelineCount: 0,
    contradictionCount: 0,
    slopFlagCount: 0,
  },
};
