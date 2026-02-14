export namespace backend {
	
	export class BeatResult {
	    name: string;
	    startChapter: number;
	    endChapter: number;
	    isBeat: boolean;
	    reasoning: string;
	
	    static createFrom(source: any = {}) {
	        return new BeatResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.startChapter = source["startChapter"];
	        this.endChapter = source["endChapter"];
	        this.isBeat = source["isBeat"];
	        this.reasoning = source["reasoning"];
	    }
	}
	export class GenreScore {
	    genre: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new GenreScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.genre = source["genre"];
	        this.score = source["score"];
	    }
	}
	export class ChapterMetric {
	    index: number;
	    title: string;
	    wordCount: number;
	    timelineMarks: number;
	    topGenre: string;
	    topGenreScore: number;
	    genreProvider: string;
	    genreReasoning: string;
	    genreBreakdown: GenreScore[];
	
	    static createFrom(source: any = {}) {
	        return new ChapterMetric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.title = source["title"];
	        this.wordCount = source["wordCount"];
	        this.timelineMarks = source["timelineMarks"];
	        this.topGenre = source["topGenre"];
	        this.topGenreScore = source["topGenreScore"];
	        this.genreProvider = source["genreProvider"];
	        this.genreReasoning = source["genreReasoning"];
	        this.genreBreakdown = this.convertValues(source["genreBreakdown"], GenreScore);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChapterSummary {
	    chapter: number;
	    title: string;
	    summary: string;
	    events: string[];
	
	    static createFrom(source: any = {}) {
	        return new ChapterSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter = source["chapter"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.events = source["events"];
	    }
	}
	export class CharacterChapterRecord {
	    chapter: number;
	    title: string;
	    summary: string;
	    actions: string[];
	    events: string[];
	
	    static createFrom(source: any = {}) {
	        return new CharacterChapterRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter = source["chapter"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.actions = source["actions"];
	        this.events = source["events"];
	    }
	}
	export class CharacterEntry {
	    name: string;
	    description: string;
	    firstSeenChapter: number;
	    lastSeenChapter: number;
	    totalMentions: number;
	    chapters: CharacterChapterRecord[];
	
	    static createFrom(source: any = {}) {
	        return new CharacterEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.firstSeenChapter = source["firstSeenChapter"];
	        this.lastSeenChapter = source["lastSeenChapter"];
	        this.totalMentions = source["totalMentions"];
	        this.chapters = this.convertValues(source["chapters"], CharacterChapterRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompTitle {
	    title: string;
	    tier: string;
	
	    static createFrom(source: any = {}) {
	        return new CompTitle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.tier = source["tier"];
	    }
	}
	export class ServiceTrace {
	    time: string;
	    level: string;
	    message: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceTrace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.detail = source["detail"];
	    }
	}
	export class ServiceStatus {
	    name: string;
	    running: boolean;
	    ready: boolean;
	    lastError: string;
	    detail: string;
	    missing: boolean;
	    installHint: string;
	    installCommand: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.running = source["running"];
	        this.ready = source["ready"];
	        this.lastError = source["lastError"];
	        this.detail = source["detail"];
	        this.missing = source["missing"];
	        this.installHint = source["installHint"];
	        this.installCommand = source["installCommand"];
	    }
	}
	export class SystemDiagnostics {
	    overall: string;
	    initializing: boolean;
	    ollama: ServiceStatus;
	    languageTool: ServiceStatus;
	    traces: ServiceTrace[];
	
	    static createFrom(source: any = {}) {
	        return new SystemDiagnostics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overall = source["overall"];
	        this.initializing = source["initializing"];
	        this.ollama = this.convertValues(source["ollama"], ServiceStatus);
	        this.languageTool = this.convertValues(source["languageTool"], ServiceStatus);
	        this.traces = this.convertValues(source["traces"], ServiceTrace);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunStats {
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
	
	    static createFrom(source: any = {}) {
	        return new RunStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.sourceName = source["sourceName"];
	        this.lastAction = source["lastAction"];
	        this.status = source["status"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	        this.chapterCount = source["chapterCount"];
	        this.segmentCount = source["segmentCount"];
	        this.timelineCount = source["timelineCount"];
	        this.contradictionCount = source["contradictionCount"];
	        this.slopFlagCount = source["slopFlagCount"];
	    }
	}
	export class LanguageReport {
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
	
	    static createFrom(source: any = {}) {
	        return new LanguageReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spellingScore = source["spellingScore"];
	        this.grammarScore = source["grammarScore"];
	        this.readabilityScore = source["readabilityScore"];
	        this.ageCategory = source["ageCategory"];
	        this.spellingProvider = source["spellingProvider"];
	        this.safetyProvider = source["safetyProvider"];
	        this.heuristicFallback = source["heuristicFallback"];
	        this.profanityScore = source["profanityScore"];
	        this.explicitScore = source["explicitScore"];
	        this.violenceScore = source["violenceScore"];
	        this.profanityInstances = source["profanityInstances"];
	        this.explicitInstances = source["explicitInstances"];
	        this.notes = source["notes"];
	    }
	}
	export class PlotStructureProbability {
	    name: string;
	    probability: number;
	
	    static createFrom(source: any = {}) {
	        return new PlotStructureProbability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.probability = source["probability"];
	    }
	}
	export class PlotStructureReport {
	    provider: string;
	    selectedStructure: string;
	    probabilities: PlotStructureProbability[];
	    reasoning: string;
	
	    static createFrom(source: any = {}) {
	        return new PlotStructureReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.selectedStructure = source["selectedStructure"];
	        this.probabilities = this.convertValues(source["probabilities"], PlotStructureProbability);
	        this.reasoning = source["reasoning"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HealthIssue {
	    id: string;
	    entity: string;
	    severity: string;
	    description: string;
	    chapterA: number;
	    chapterB: number;
	    contextA: string;
	    contextB: string;
	    dictionaryRef: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.entity = source["entity"];
	        this.severity = source["severity"];
	        this.description = source["description"];
	        this.chapterA = source["chapterA"];
	        this.chapterB = source["chapterB"];
	        this.contextA = source["contextA"];
	        this.contextB = source["contextB"];
	        this.dictionaryRef = source["dictionaryRef"];
	    }
	}
	export class LogLine {
	    time: string;
	    level: string;
	    stage: string;
	    message: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new LogLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.stage = source["stage"];
	        this.message = source["message"];
	        this.detail = source["detail"];
	    }
	}
	export class DashboardData {
	    bookTitle: string;
	    wordCount: number;
	    mhdScore: number;
	    logs: LogLine[];
	    contradictions: forensics.Contradiction[];
	    healthIssues: HealthIssue[];
	    slopReport: slop.Report;
	    timeline: timeline.Event[];
	    beats: BeatResult[];
	    plotStructure: PlotStructureReport;
	    genreScores: GenreScore[];
	    genreProvider: string;
	    genreReasoning: string;
	    chapterMetrics: ChapterMetric[];
	    chapterSummaries: ChapterSummary[];
	    characterDictionary: CharacterEntry[];
	    chapterCount: number;
	    compTitles: CompTitle[];
	    language: LanguageReport;
	    projectLocation: string;
	    runStats: RunStats;
	    system: SystemDiagnostics;
	
	    static createFrom(source: any = {}) {
	        return new DashboardData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bookTitle = source["bookTitle"];
	        this.wordCount = source["wordCount"];
	        this.mhdScore = source["mhdScore"];
	        this.logs = this.convertValues(source["logs"], LogLine);
	        this.contradictions = this.convertValues(source["contradictions"], forensics.Contradiction);
	        this.healthIssues = this.convertValues(source["healthIssues"], HealthIssue);
	        this.slopReport = this.convertValues(source["slopReport"], slop.Report);
	        this.timeline = this.convertValues(source["timeline"], timeline.Event);
	        this.beats = this.convertValues(source["beats"], BeatResult);
	        this.plotStructure = this.convertValues(source["plotStructure"], PlotStructureReport);
	        this.genreScores = this.convertValues(source["genreScores"], GenreScore);
	        this.genreProvider = source["genreProvider"];
	        this.genreReasoning = source["genreReasoning"];
	        this.chapterMetrics = this.convertValues(source["chapterMetrics"], ChapterMetric);
	        this.chapterSummaries = this.convertValues(source["chapterSummaries"], ChapterSummary);
	        this.characterDictionary = this.convertValues(source["characterDictionary"], CharacterEntry);
	        this.chapterCount = source["chapterCount"];
	        this.compTitles = this.convertValues(source["compTitles"], CompTitle);
	        this.language = this.convertValues(source["language"], LanguageReport);
	        this.projectLocation = source["projectLocation"];
	        this.runStats = this.convertValues(source["runStats"], RunStats);
	        this.system = this.convertValues(source["system"], SystemDiagnostics);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	

}

export namespace forensics {
	
	export class Contradiction {
	    EntityName: string;
	    Attribute: string;
	    ValueA: string;
	    ValueB: string;
	    ChapterA: number;
	    ChapterB: number;
	    Description: string;
	    Severity: string;
	
	    static createFrom(source: any = {}) {
	        return new Contradiction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.EntityName = source["EntityName"];
	        this.Attribute = source["Attribute"];
	        this.ValueA = source["ValueA"];
	        this.ValueB = source["ValueB"];
	        this.ChapterA = source["ChapterA"];
	        this.ChapterB = source["ChapterB"];
	        this.Description = source["Description"];
	        this.Severity = source["Severity"];
	    }
	}

}

export namespace slop {
	
	export class Report {
	    Monotone: boolean;
	    MeanSentenceLength: number;
	    SentenceLengthSD: number;
	    BadWordDensity: number;
	    LowOriginality: boolean;
	    Flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Monotone = source["Monotone"];
	        this.MeanSentenceLength = source["MeanSentenceLength"];
	        this.SentenceLengthSD = source["SentenceLengthSD"];
	        this.BadWordDensity = source["BadWordDensity"];
	        this.LowOriginality = source["LowOriginality"];
	        this.Flags = source["Flags"];
	    }
	}

}

export namespace timeline {
	
	export class Event {
	    time_marker: string;
	    event: string;
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time_marker = source["time_marker"];
	        this.event = source["event"];
	    }
	}

}

