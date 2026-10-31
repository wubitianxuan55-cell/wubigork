export namespace app {

	export class AgentView {
	    temperature: number;
	    maxSteps: number;
	    systemPrompt: string;
	    subagentTemperature: number;
	    effort: string;
	    subagentEffort: string;

	    static createFrom(source: any = {}) {
	        return new AgentView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.temperature = source["temperature"];
	        this.maxSteps = source["maxSteps"];
	        this.systemPrompt = source["systemPrompt"];
	        this.subagentTemperature = source["subagentTemperature"];
	        this.effort = source["effort"];
	        this.subagentEffort = source["subagentEffort"];
	    }
	}
	export class BalanceInfo {
	    available: boolean;
	    display: string;
	    err?: string;

	    static createFrom(source: any = {}) {
	        return new BalanceInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.display = source["display"];
	        this.err = source["err"];
	    }
	}
	export class BenchmarkCase {
	    model_name: string;
	    variant_id: string;
	    context_size: number;
	    status: string;
	    started_at?: string;
	    ended_at?: string;
	    duration_ms: number;
	    ttft_ms_avg: number;
	    ttft_ms_p95: number;
	    input_tokens: number;
	    output_tokens: number;
	    total_tokens: number;
	    cached_tokens: number;
	    second_duration_ms: number;
	    second_ttft_ms_avg: number;
	    prompt_tokens_tps: number;
	    output_tokens_tps: number;
	    prefill_speedup_ratio: number;
	    prefill_ms_saved: number;
	    prompt_ms: number;
	    predicted_ms: number;
	    response_excerpt?: string;
	    effective_launch_params?: Record<string, any>;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new BenchmarkCase(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_name = source["model_name"];
	        this.variant_id = source["variant_id"];
	        this.context_size = source["context_size"];
	        this.status = source["status"];
	        this.started_at = source["started_at"];
	        this.ended_at = source["ended_at"];
	        this.duration_ms = source["duration_ms"];
	        this.ttft_ms_avg = source["ttft_ms_avg"];
	        this.ttft_ms_p95 = source["ttft_ms_p95"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.cached_tokens = source["cached_tokens"];
	        this.second_duration_ms = source["second_duration_ms"];
	        this.second_ttft_ms_avg = source["second_ttft_ms_avg"];
	        this.prompt_tokens_tps = source["prompt_tokens_tps"];
	        this.output_tokens_tps = source["output_tokens_tps"];
	        this.prefill_speedup_ratio = source["prefill_speedup_ratio"];
	        this.prefill_ms_saved = source["prefill_ms_saved"];
	        this.prompt_ms = source["prompt_ms"];
	        this.predicted_ms = source["predicted_ms"];
	        this.response_excerpt = source["response_excerpt"];
	        this.effective_launch_params = source["effective_launch_params"];
	        this.error = source["error"];
	    }
	}
	export class BenchmarkPromptRequest {
	    user_prompt: string;
	    temperature: number;
	    top_p: number;
	    top_k: number;
	    repeat_penalty: number;
	    max_tokens: number;
	    stream: boolean;
	    timeout_seconds: number;

	    static createFrom(source: any = {}) {
	        return new BenchmarkPromptRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_prompt = source["user_prompt"];
	        this.temperature = source["temperature"];
	        this.top_p = source["top_p"];
	        this.top_k = source["top_k"];
	        this.repeat_penalty = source["repeat_penalty"];
	        this.max_tokens = source["max_tokens"];
	        this.stream = source["stream"];
	        this.timeout_seconds = source["timeout_seconds"];
	    }
	}
	export class BenchmarkRequest {
	    model_names: string[];
	    variants: string[];
	    context_sizes: number[];
	    cache_reuse_mode: string;
	    warmup_count: number;
	    repeat_count: number;
	    concurrency: number;
	    request: BenchmarkPromptRequest;

	    static createFrom(source: any = {}) {
	        return new BenchmarkRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_names = source["model_names"];
	        this.variants = source["variants"];
	        this.context_sizes = source["context_sizes"];
	        this.cache_reuse_mode = source["cache_reuse_mode"];
	        this.warmup_count = source["warmup_count"];
	        this.repeat_count = source["repeat_count"];
	        this.concurrency = source["concurrency"];
	        this.request = this.convertValues(source["request"], BenchmarkPromptRequest);
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
	export class BenchmarkSummary {
	    total_cases: number;
	    succeeded: number;
	    failed: number;
	    canceled: number;
	    avg_duration_ms: number;
	    avg_ttft_ms: number;
	    avg_tps: number;

	    static createFrom(source: any = {}) {
	        return new BenchmarkSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_cases = source["total_cases"];
	        this.succeeded = source["succeeded"];
	        this.failed = source["failed"];
	        this.canceled = source["canceled"];
	        this.avg_duration_ms = source["avg_duration_ms"];
	        this.avg_ttft_ms = source["avg_ttft_ms"];
	        this.avg_tps = source["avg_tps"];
	    }
	}
	export class BenchmarkRunDetail {
	    id: string;
	    created_at: string;
	    finished_at?: string;
	    status: string;
	    config: BenchmarkRequest;
	    summary: BenchmarkSummary;
	    cases: BenchmarkCase[];

	    static createFrom(source: any = {}) {
	        return new BenchmarkRunDetail(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = source["created_at"];
	        this.finished_at = source["finished_at"];
	        this.status = source["status"];
	        this.config = this.convertValues(source["config"], BenchmarkRequest);
	        this.summary = this.convertValues(source["summary"], BenchmarkSummary);
	        this.cases = this.convertValues(source["cases"], BenchmarkCase);
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
	export class BenchmarkRunSummary {
	    id: string;
	    created_at: string;
	    finished_at?: string;
	    status: string;
	    model_names: string[];
	    variants: string[];
	    context_sizes: number[];
	    summary?: BenchmarkSummary;

	    static createFrom(source: any = {}) {
	        return new BenchmarkRunSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = source["created_at"];
	        this.finished_at = source["finished_at"];
	        this.status = source["status"];
	        this.model_names = source["model_names"];
	        this.variants = source["variants"];
	        this.context_sizes = source["context_sizes"];
	        this.summary = this.convertValues(source["summary"], BenchmarkSummary);
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

	export class BookHealthChapter {
	    chapterNum: number;
	    words: number;
	    outlineIssues: number;
	    outlineErrors: number;
	    qualityIssues: number;
	    qualityErrors: number;
	    aiTasteScore: number;

	    static createFrom(source: any = {}) {
	        return new BookHealthChapter(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterNum = source["chapterNum"];
	        this.words = source["words"];
	        this.outlineIssues = source["outlineIssues"];
	        this.outlineErrors = source["outlineErrors"];
	        this.qualityIssues = source["qualityIssues"];
	        this.qualityErrors = source["qualityErrors"];
	        this.aiTasteScore = source["aiTasteScore"];
	    }
	}
	export class ForeshadowLintFinding {
	    code: string;
	    severity: string;
	    foreshadowId: string;
	    itemDesc: string;
	    message: string;
	    chapter?: string;

	    static createFrom(source: any = {}) {
	        return new ForeshadowLintFinding(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.severity = source["severity"];
	        this.foreshadowId = source["foreshadowId"];
	        this.itemDesc = source["itemDesc"];
	        this.message = source["message"];
	        this.chapter = source["chapter"];
	    }
	}
	export class ForeshadowLintReport {
	    totalChapters: number;
	    items: number;
	    planted: number;
	    hinted: number;
	    revealed: number;
	    longTerm: number;
	    findings: ForeshadowLintFinding[];

	    static createFrom(source: any = {}) {
	        return new ForeshadowLintReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalChapters = source["totalChapters"];
	        this.items = source["items"];
	        this.planted = source["planted"];
	        this.hinted = source["hinted"];
	        this.revealed = source["revealed"];
	        this.longTerm = source["longTerm"];
	        this.findings = this.convertValues(source["findings"], ForeshadowLintFinding);
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
	export class BookHealthReport {
	    totalChapters: number;
	    chapters: BookHealthChapter[];
	    contractIssueChapters: number;
	    qualityIssueChapters: number;
	    worstAiTaste?: BookHealthChapter;
	    analyzedChapters: number;
	    foreshadow: ForeshadowLintReport;

	    static createFrom(source: any = {}) {
	        return new BookHealthReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalChapters = source["totalChapters"];
	        this.chapters = this.convertValues(source["chapters"], BookHealthChapter);
	        this.contractIssueChapters = source["contractIssueChapters"];
	        this.qualityIssueChapters = source["qualityIssueChapters"];
	        this.worstAiTaste = this.convertValues(source["worstAiTaste"], BookHealthChapter);
	        this.analyzedChapters = source["analyzedChapters"];
	        this.foreshadow = this.convertValues(source["foreshadow"], ForeshadowLintReport);
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
	export class SkillView {
	    name: string;
	    description: string;
	    scope: string;
	    runAs: string;

	    static createFrom(source: any = {}) {
	        return new SkillView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.scope = source["scope"];
	        this.runAs = source["runAs"];
	    }
	}
	export class ToolView {
	    name: string;
	    description: string;

	    static createFrom(source: any = {}) {
	        return new ToolView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class ServerView {
	    name: string;
	    transport: string;
	    status: string;
	    tools: number;
	    prompts: number;
	    resources: number;
	    error?: string;
	    toolList?: ToolView[];

	    static createFrom(source: any = {}) {
	        return new ServerView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.transport = source["transport"];
	        this.status = source["status"];
	        this.tools = source["tools"];
	        this.prompts = source["prompts"];
	        this.resources = source["resources"];
	        this.error = source["error"];
	        this.toolList = this.convertValues(source["toolList"], ToolView);
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
	export class CapabilitiesView {
	    servers: ServerView[];
	    skills: SkillView[];

	    static createFrom(source: any = {}) {
	        return new CapabilitiesView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], ServerView);
	        this.skills = this.convertValues(source["skills"], SkillView);
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
	export class ChapterReviewEvidenceView {
	    paragraph: number;
	    excerpt: string;

	    static createFrom(source: any = {}) {
	        return new ChapterReviewEvidenceView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paragraph = source["paragraph"];
	        this.excerpt = source["excerpt"];
	    }
	}
	export class ChapterReviewDimensionView {
	    id: string;
	    label: string;
	    verdict: string;
	    severity?: string;
	    detail: string;
	    advice?: string;
	    evidence?: ChapterReviewEvidenceView[];

	    static createFrom(source: any = {}) {
	        return new ChapterReviewDimensionView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.verdict = source["verdict"];
	        this.severity = source["severity"];
	        this.detail = source["detail"];
	        this.advice = source["advice"];
	        this.evidence = this.convertValues(source["evidence"], ChapterReviewEvidenceView);
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

	export class ChapterReviewPayload {
	    chapterNum: number;
	    platform: string;
	    platformLabel: string;
	    words: number;
	    verdict: string;
	    counts: Record<string, number>;
	    dimensions: ChapterReviewDimensionView[];
	    advisories?: string[];

	    static createFrom(source: any = {}) {
	        return new ChapterReviewPayload(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterNum = source["chapterNum"];
	        this.platform = source["platform"];
	        this.platformLabel = source["platformLabel"];
	        this.words = source["words"];
	        this.verdict = source["verdict"];
	        this.counts = source["counts"];
	        this.dimensions = this.convertValues(source["dimensions"], ChapterReviewDimensionView);
	        this.advisories = source["advisories"];
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
	export class ChatMessageInput {
	    role: string;
	    content: string;
	    extra: string;

	    static createFrom(source: any = {}) {
	        return new ChatMessageInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.extra = source["extra"];
	    }
	}
	export class ChatMessagesPageResult {
	    messages: chat.Message[];
	    hasMore: boolean;
	    oldestSeq: number;

	    static createFrom(source: any = {}) {
	        return new ChatMessagesPageResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messages = this.convertValues(source["messages"], chat.Message);
	        this.hasMore = source["hasMore"];
	        this.oldestSeq = source["oldestSeq"];
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
	export class CommandInfo {
	    name: string;
	    description: string;
	    hint?: string;
	    kind: string;

	    static createFrom(source: any = {}) {
	        return new CommandInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.hint = source["hint"];
	        this.kind = source["kind"];
	    }
	}
	export class ContextInfo {
	    used: number;
	    window: number;
	    compactRatio?: number;
	    compactLeft?: number;

	    static createFrom(source: any = {}) {
	        return new ContextInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.used = source["used"];
	        this.window = source["window"];
	        this.compactRatio = source["compactRatio"];
	        this.compactLeft = source["compactLeft"];
	    }
	}
	export class ConvertPdfResult {
	    path: string;
	    name: string;
	    size: number;
	    source: string;

	    static createFrom(source: any = {}) {
	        return new ConvertPdfResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.source = source["source"];
	    }
	}
	export class CostCategoryView {
	    id: number;
	    parentId: number;
	    name: string;
	    sort: number;
	    count: number;
	    children?: CostCategoryView[];

	    static createFrom(source: any = {}) {
	        return new CostCategoryView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentId = source["parentId"];
	        this.name = source["name"];
	        this.sort = source["sort"];
	        this.count = source["count"];
	        this.children = this.convertValues(source["children"], CostCategoryView);
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
	export class CostCompareRow {
	    source: string;
	    period: string;
	    price: number;
	    diffPct: number;
	    fetchedAt: string;
	    kind: string;

	    static createFrom(source: any = {}) {
	        return new CostCompareRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.period = source["period"];
	        this.price = source["price"];
	        this.diffPct = source["diffPct"];
	        this.fetchedAt = source["fetchedAt"];
	        this.kind = source["kind"];
	    }
	}
	export class CostComponentView {
	    kind: string;
	    title: string;
	    unit?: string;
	    quantity?: number;
	    price?: number;
	    amount?: number;
	    note?: string;
	    sort?: number;

	    static createFrom(source: any = {}) {
	        return new CostComponentView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.unit = source["unit"];
	        this.quantity = source["quantity"];
	        this.price = source["price"];
	        this.amount = source["amount"];
	        this.note = source["note"];
	        this.sort = source["sort"];
	    }
	}
	export class CostComposeEvidence {
	    name: string;
	    title: string;
	    category: string;
	    unit: string;
	    spec: string;
	    price: number;
	    source: string;
	    region: string;
	    priceDate: string;
	    priceType: string;

	    static createFrom(source: any = {}) {
	        return new CostComposeEvidence(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.category = source["category"];
	        this.unit = source["unit"];
	        this.spec = source["spec"];
	        this.price = source["price"];
	        this.source = source["source"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.priceType = source["priceType"];
	    }
	}
	export class CostComposeView {
	    description: string;
	    unit: string;
	    band?: cost.PriceBand;
	    recommendedPrice: number;
	    reason: string;
	    components?: CostComponentView[];
	    componentsNote?: string;
	    llmUsed: boolean;
	    evidence: CostComposeEvidence[];
	    checks?: cost.ComposeCheck[];

	    static createFrom(source: any = {}) {
	        return new CostComposeView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.unit = source["unit"];
	        this.band = this.convertValues(source["band"], cost.PriceBand);
	        this.recommendedPrice = source["recommendedPrice"];
	        this.reason = source["reason"];
	        this.components = this.convertValues(source["components"], CostComponentView);
	        this.componentsNote = source["componentsNote"];
	        this.llmUsed = source["llmUsed"];
	        this.evidence = this.convertValues(source["evidence"], CostComposeEvidence);
	        this.checks = this.convertValues(source["checks"], cost.ComposeCheck);
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
	export class CostComposeRecord {
	    id: number;
	    entryName: string;
	    createdAt: string;
	    llmUsed: boolean;
	    snapshot?: CostComposeView;

	    static createFrom(source: any = {}) {
	        return new CostComposeRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.entryName = source["entryName"];
	        this.createdAt = source["createdAt"];
	        this.llmUsed = source["llmUsed"];
	        this.snapshot = this.convertValues(source["snapshot"], CostComposeView);
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

	export class CostEntry {
	    name: string;
	    title: string;
	    code?: string;
	    category: string;
	    categoryPath: string;
	    unit: string;
	    price: number;
	    laborFee?: number;
	    materialFee?: number;
	    machineFee?: number;
	    componentCount?: number;
	    spec: string;
	    source: string;
	    region?: string;
	    priceDate?: string;
	    priceType?: string;
	    validUntil?: string;
	    sourceRow?: number;
	    tags: string[];
	    status: string;
	    // Go type: time
	    updatedAt: any;
	    managementFee?: number;
	    profitFee?: number;
	    advanceFee?: number;
	    taxRate?: number;
	    components?: CostComponentView[];
	    body: string;
	    // Go type: time
	    createdAt: any;

	    static createFrom(source: any = {}) {
	        return new CostEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.code = source["code"];
	        this.category = source["category"];
	        this.categoryPath = source["categoryPath"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.laborFee = source["laborFee"];
	        this.materialFee = source["materialFee"];
	        this.machineFee = source["machineFee"];
	        this.componentCount = source["componentCount"];
	        this.spec = source["spec"];
	        this.source = source["source"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.priceType = source["priceType"];
	        this.validUntil = source["validUntil"];
	        this.sourceRow = source["sourceRow"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.managementFee = source["managementFee"];
	        this.profitFee = source["profitFee"];
	        this.advanceFee = source["advanceFee"];
	        this.taxRate = source["taxRate"];
	        this.components = this.convertValues(source["components"], CostComponentView);
	        this.body = source["body"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class CostImportRowView {
	    name: string;
	    title: string;
	    code: string;
	    category: string;
	    unit: string;
	    price: number;
	    laborFee?: number;
	    materialFee?: number;
	    machineFee?: number;
	    managementFee?: number;
	    profitFee?: number;
	    advanceFee?: number;
	    taxRate?: number;
	    components?: CostComponentView[];
	    body?: string;
	    spec: string;
	    source: string;
	    sourceRow?: number;
	    status: string;
	    existingName: string;
	    existingPrice: number;
	    matchNote: string;
	    raw: string;
	    skip: boolean;
	    skipReason: string;

	    static createFrom(source: any = {}) {
	        return new CostImportRowView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.code = source["code"];
	        this.category = source["category"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.laborFee = source["laborFee"];
	        this.materialFee = source["materialFee"];
	        this.machineFee = source["machineFee"];
	        this.managementFee = source["managementFee"];
	        this.profitFee = source["profitFee"];
	        this.advanceFee = source["advanceFee"];
	        this.taxRate = source["taxRate"];
	        this.components = this.convertValues(source["components"], CostComponentView);
	        this.body = source["body"];
	        this.spec = source["spec"];
	        this.source = source["source"];
	        this.sourceRow = source["sourceRow"];
	        this.status = source["status"];
	        this.existingName = source["existingName"];
	        this.existingPrice = source["existingPrice"];
	        this.matchNote = source["matchNote"];
	        this.raw = source["raw"];
	        this.skip = source["skip"];
	        this.skipReason = source["skipReason"];
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
	export class CostImportPreview {
	    path: string;
	    fileName: string;
	    columns: string[];
	    unmapped: string[];
	    rows: CostImportRowView[];
	    message: string;
	    aiUsed: boolean;
	    source: string;

	    static createFrom(source: any = {}) {
	        return new CostImportPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.columns = source["columns"];
	        this.unmapped = source["unmapped"];
	        this.rows = this.convertValues(source["rows"], CostImportRowView);
	        this.message = source["message"];
	        this.aiUsed = source["aiUsed"];
	        this.source = source["source"];
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

	export class CostSummary {
	    name: string;
	    title: string;
	    code?: string;
	    category: string;
	    categoryPath: string;
	    unit: string;
	    price: number;
	    laborFee?: number;
	    materialFee?: number;
	    machineFee?: number;
	    componentCount?: number;
	    spec: string;
	    source: string;
	    region?: string;
	    priceDate?: string;
	    priceType?: string;
	    validUntil?: string;
	    sourceRow?: number;
	    tags: string[];
	    status: string;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new CostSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.code = source["code"];
	        this.category = source["category"];
	        this.categoryPath = source["categoryPath"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.laborFee = source["laborFee"];
	        this.materialFee = source["materialFee"];
	        this.machineFee = source["machineFee"];
	        this.componentCount = source["componentCount"];
	        this.spec = source["spec"];
	        this.source = source["source"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.priceType = source["priceType"];
	        this.validUntil = source["validUntil"];
	        this.sourceRow = source["sourceRow"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class CostSearchPage {
	    items: CostSummary[];
	    total: number;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new CostSearchPage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], CostSummary);
	        this.total = source["total"];
	        this.error = source["error"];
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

	export class CrossEmbedInput {
	    xlsxRel: string;
	    sheet: string;
	    range: string;
	    chartType: string;
	    title: string;
	    into: string;
	    output: string;

	    static createFrom(source: any = {}) {
	        return new CrossEmbedInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.xlsxRel = source["xlsxRel"];
	        this.sheet = source["sheet"];
	        this.range = source["range"];
	        this.chartType = source["chartType"];
	        this.title = source["title"];
	        this.into = source["into"];
	        this.output = source["output"];
	    }
	}
	export class CrossEmbedResult {
	    path: string;
	    name: string;
	    size: number;
	    chartPath: string;

	    static createFrom(source: any = {}) {
	        return new CrossEmbedResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.chartPath = source["chartPath"];
	    }
	}
	export class DatabaseProjectStats {
	    path: string;
	    name: string;
	    current: boolean;
	    sessionCount: number;
	    archivedCount: number;
	    sessionBytes: number;
	    uploadCount: number;
	    uploadBytes: number;
	    latestMod: number;

	    static createFrom(source: any = {}) {
	        return new DatabaseProjectStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.current = source["current"];
	        this.sessionCount = source["sessionCount"];
	        this.archivedCount = source["archivedCount"];
	        this.sessionBytes = source["sessionBytes"];
	        this.uploadCount = source["uploadCount"];
	        this.uploadBytes = source["uploadBytes"];
	        this.latestMod = source["latestMod"];
	    }
	}
	export class DatabaseOverview {
	    projects: DatabaseProjectStats[];
	    totalSessions: number;
	    totalArchived: number;
	    totalUploads: number;
	    totalBytes: number;

	    static createFrom(source: any = {}) {
	        return new DatabaseOverview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projects = this.convertValues(source["projects"], DatabaseProjectStats);
	        this.totalSessions = source["totalSessions"];
	        this.totalArchived = source["totalArchived"];
	        this.totalUploads = source["totalUploads"];
	        this.totalBytes = source["totalBytes"];
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

	export class DeliverableRegistryView {
	    available: boolean;
	    entries: trajectory.DeliverableEntry[];
	    total: number;

	    static createFrom(source: any = {}) {
	        return new DeliverableRegistryView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.entries = this.convertValues(source["entries"], trajectory.DeliverableEntry);
	        this.total = source["total"];
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
	export class DirEntry {
	    name: string;
	    isDir: boolean;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new DirEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	    }
	}
	export class ExportDeliverableInput {
	    markdown: string;
	    format: string;
	    title: string;
	    template: string;
	    cover: boolean;
	    toc: boolean;
	    header: string;
	    footer: string;

	    static createFrom(source: any = {}) {
	        return new ExportDeliverableInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.markdown = source["markdown"];
	        this.format = source["format"];
	        this.title = source["title"];
	        this.template = source["template"];
	        this.cover = source["cover"];
	        this.toc = source["toc"];
	        this.header = source["header"];
	        this.footer = source["footer"];
	    }
	}
	export class ExportDeliverableResult {
	    path: string;
	    name: string;
	    format: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new ExportDeliverableResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.format = source["format"];
	        this.size = source["size"];
	    }
	}
	export class FactView {
	    key: string;
	    value: string;
	    source?: string;
	    category?: string;
	    updatedAt: number;

	    static createFrom(source: any = {}) {
	        return new FactView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.source = source["source"];
	        this.category = source["category"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class FactBaseView {
	    facts: FactView[];
	    markdown: string;
	    count: number;
	    path: string;

	    static createFrom(source: any = {}) {
	        return new FactBaseView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.facts = this.convertValues(source["facts"], FactView);
	        this.markdown = source["markdown"];
	        this.count = source["count"];
	        this.path = source["path"];
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

	export class FilePickResult {
	    path: string;
	    name: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new FilePickResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	    }
	}
	export class FilePreview {
	    path: string;
	    markdown: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new FilePreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.markdown = source["markdown"];
	        this.size = source["size"];
	    }
	}
	export class FileSearchHit {
	    path: string;
	    name: string;
	    isDir: boolean;
	    size: number;
	    modTime: number;

	    static createFrom(source: any = {}) {
	        return new FileSearchHit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	    }
	}
	export class FileSemanticHit {
	    path: string;
	    score: number;
	    snippet: string;

	    static createFrom(source: any = {}) {
	        return new FileSemanticHit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.score = source["score"];
	        this.snippet = source["snippet"];
	    }
	}
	export class FingerprintIssueView {
	    start: number;
	    end: number;
	    reason: string;
	    severity: string;
	    suggestion: string;
	    excerpt: string;

	    static createFrom(source: any = {}) {
	        return new FingerprintIssueView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.reason = source["reason"];
	        this.severity = source["severity"];
	        this.suggestion = source["suggestion"];
	        this.excerpt = source["excerpt"];
	    }
	}
	export class FingerprintScorePayload {
	    chapterNum: number;
	    refExists: boolean;
	    score: number;
	    delta?: number;
	    issues: FingerprintIssueView[];

	    static createFrom(source: any = {}) {
	        return new FingerprintScorePayload(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterNum = source["chapterNum"];
	        this.refExists = source["refExists"];
	        this.score = source["score"];
	        this.delta = source["delta"];
	        this.issues = this.convertValues(source["issues"], FingerprintIssueView);
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
	export class FingerprintSummary {
	    sentenceMean: number;
	    sentenceSd: number;
	    paraMean: number;
	    ttr1000: number;
	    dialogRatio: number;
	    fourCharRatio: number;
	    connectiveDensity: number;
	    adjAdvDensity: number;
	    topBigrams: string[];
	    topTrigrams: string[];
	    authorSignWords: string[];

	    static createFrom(source: any = {}) {
	        return new FingerprintSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sentenceMean = source["sentenceMean"];
	        this.sentenceSd = source["sentenceSd"];
	        this.paraMean = source["paraMean"];
	        this.ttr1000 = source["ttr1000"];
	        this.dialogRatio = source["dialogRatio"];
	        this.fourCharRatio = source["fourCharRatio"];
	        this.connectiveDensity = source["connectiveDensity"];
	        this.adjAdvDensity = source["adjAdvDensity"];
	        this.topBigrams = source["topBigrams"];
	        this.topTrigrams = source["topTrigrams"];
	        this.authorSignWords = source["authorSignWords"];
	    }
	}
	export class FingerprintStatusPayload {
	    exists: boolean;
	    builtAt?: string;
	    chapters?: number;
	    chars?: number;
	    summary?: FingerprintSummary;

	    static createFrom(source: any = {}) {
	        return new FingerprintStatusPayload(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.builtAt = source["builtAt"];
	        this.chapters = source["chapters"];
	        this.chars = source["chars"];
	        this.summary = this.convertValues(source["summary"], FingerprintSummary);
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

	export class ForeshadowCleanupResult {
	    deleted: number;
	    rolledBack?: number;
	    resetManual?: number;

	    static createFrom(source: any = {}) {
	        return new ForeshadowCleanupResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deleted = source["deleted"];
	        this.rolledBack = source["rolledBack"];
	        this.resetManual = source["resetManual"];
	    }
	}
	export class ForeshadowDueItem {
	    id: string;
	    title: string;
	    status: string;
	    plantedIn?: string;
	    targetIn?: string;
	    urgency: number;
	    overdueBy?: number;
	    longTerm?: boolean;

	    static createFrom(source: any = {}) {
	        return new ForeshadowDueItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.plantedIn = source["plantedIn"];
	        this.targetIn = source["targetIn"];
	        this.urgency = source["urgency"];
	        this.overdueBy = source["overdueBy"];
	        this.longTerm = source["longTerm"];
	    }
	}
	export class ForeshadowDueReport {
	    chapterNum: number;
	    urgent: ForeshadowDueItem[];
	    soon: ForeshadowDueItem[];

	    static createFrom(source: any = {}) {
	        return new ForeshadowDueReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterNum = source["chapterNum"];
	        this.urgent = this.convertValues(source["urgent"], ForeshadowDueItem);
	        this.soon = this.convertValues(source["soon"], ForeshadowDueItem);
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


	export class ForeshadowStatsReport {
	    total: number;
	    pending: number;
	    planted: number;
	    hinted: number;
	    resolved: number;
	    partiallyResolved: number;
	    abandoned: number;
	    longTermCount: number;
	    overdueCount: number;
	    currentChapter?: number;

	    static createFrom(source: any = {}) {
	        return new ForeshadowStatsReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.pending = source["pending"];
	        this.planted = source["planted"];
	        this.hinted = source["hinted"];
	        this.resolved = source["resolved"];
	        this.partiallyResolved = source["partiallyResolved"];
	        this.abandoned = source["abandoned"];
	        this.longTermCount = source["longTermCount"];
	        this.overdueCount = source["overdueCount"];
	        this.currentChapter = source["currentChapter"];
	    }
	}
	export class GaeaReloadResult {
	    tools: number;
	    skills: number;

	    static createFrom(source: any = {}) {
	        return new GaeaReloadResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools = source["tools"];
	        this.skills = source["skills"];
	    }
	}
	export class GaeaResyncItem {
	    kind: string;
	    id: string;
	    text: string;
	    reasoning: string;
	    streaming: boolean;
	    level: string;
	    name: string;
	    args: string;
	    toolId: string;
	    output: string;
	    err: string;
	    status: string;
	    readOnly: boolean;
	    truncated: boolean;
	    parentId: string;
	    subagentRef: string;

	    static createFrom(source: any = {}) {
	        return new GaeaResyncItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.id = source["id"];
	        this.text = source["text"];
	        this.reasoning = source["reasoning"];
	        this.streaming = source["streaming"];
	        this.level = source["level"];
	        this.name = source["name"];
	        this.args = source["args"];
	        this.toolId = source["toolId"];
	        this.output = source["output"];
	        this.err = source["err"];
	        this.status = source["status"];
	        this.readOnly = source["readOnly"];
	        this.truncated = source["truncated"];
	        this.parentId = source["parentId"];
	        this.subagentRef = source["subagentRef"];
	    }
	}
	export class GaeaResyncResult {
	    seq: number;
	    items: GaeaResyncItem[];

	    static createFrom(source: any = {}) {
	        return new GaeaResyncResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.items = this.convertValues(source["items"], GaeaResyncItem);
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
	export class GaeaSummaryResult {
	    path: string;
	    totalPages: number;
	    chars: number;
	    chunks: number;
	    summary: string;

	    static createFrom(source: any = {}) {
	        return new GaeaSummaryResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.totalPages = source["totalPages"];
	        this.chars = source["chars"];
	        this.chunks = source["chunks"];
	        this.summary = source["summary"];
	    }
	}
	export class GitCommitInfo {
	    hash: string;
	    subject: string;
	    author?: string;
	    ts?: number;

	    static createFrom(source: any = {}) {
	        return new GitCommitInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.subject = source["subject"];
	        this.author = source["author"];
	        this.ts = source["ts"];
	    }
	}
	export class GitFileStatus {
	    path: string;
	    x: string;
	    y: string;
	    staged: boolean;
	    untracked: boolean;
	    deleted: boolean;
	    modified: boolean;
	    renamed: boolean;

	    static createFrom(source: any = {}) {
	        return new GitFileStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.staged = source["staged"];
	        this.untracked = source["untracked"];
	        this.deleted = source["deleted"];
	        this.modified = source["modified"];
	        this.renamed = source["renamed"];
	    }
	}
	export class GitStatus {
	    isRepo: boolean;
	    branch?: string;
	    ahead?: number;
	    behind?: number;
	    files: GitFileStatus[];
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new GitStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRepo = source["isRepo"];
	        this.branch = source["branch"];
	        this.ahead = source["ahead"];
	        this.behind = source["behind"];
	        this.files = this.convertValues(source["files"], GitFileStatus);
	        this.error = source["error"];
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
	export class GraphLink {
	    source: string;
	    target: string;
	    type: string;

	    static createFrom(source: any = {}) {
	        return new GraphLink(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.target = source["target"];
	        this.type = source["type"];
	    }
	}
	export class GraphNode {
	    id: string;
	    name: string;
	    type: string;
	    desc: string;
	    val: number;

	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.desc = source["desc"];
	        this.val = source["val"];
	    }
	}
	export class HerdsmanCatalogModel {
	    name: string;
	    display_name: string;
	    type: string;
	    runtime: string;
	    inference_engines?: string[];
	    capabilities?: string[];
	    installed: boolean;
	    running: boolean;
	    status: string;
	    run_status?: string;
	    quantization?: string;
	    parameter_count?: number;
	    active_parameters?: number;
	    is_moe?: boolean;
	    file_size?: number;
	    llama_cpp_variants?: string[];
	    hint?: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanCatalogModel(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.display_name = source["display_name"];
	        this.type = source["type"];
	        this.runtime = source["runtime"];
	        this.inference_engines = source["inference_engines"];
	        this.capabilities = source["capabilities"];
	        this.installed = source["installed"];
	        this.running = source["running"];
	        this.status = source["status"];
	        this.run_status = source["run_status"];
	        this.quantization = source["quantization"];
	        this.parameter_count = source["parameter_count"];
	        this.active_parameters = source["active_parameters"];
	        this.is_moe = source["is_moe"];
	        this.file_size = source["file_size"];
	        this.llama_cpp_variants = source["llama_cpp_variants"];
	        this.hint = source["hint"];
	    }
	}
	export class HerdsmanCatalog {
	    models: HerdsmanCatalogModel[];
	    total: number;
	    installed: number;
	    running: number;
	    source: string;
	    error?: string;
	    installed_bytes?: number;
	    disk_total?: number;
	    disk_free?: number;
	    disk_error?: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanCatalog(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.models = this.convertValues(source["models"], HerdsmanCatalogModel);
	        this.total = source["total"];
	        this.installed = source["installed"];
	        this.running = source["running"];
	        this.source = source["source"];
	        this.error = source["error"];
	        this.installed_bytes = source["installed_bytes"];
	        this.disk_total = source["disk_total"];
	        this.disk_free = source["disk_free"];
	        this.disk_error = source["disk_error"];
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

	export class HerdsmanDigitalCharacter {
	    id: string;
	    name: string;
	    gender: string;
	    identity: string;
	    worldview: string;
	    text_model: string;
	    intimacy: number;
	    trust: number;
	    safety: number;
	    conflict: number;
	    last_interacted_at: string;
	    memory_summary: string;
	    highlights?: string[];
	    memory_event_count: number;
	    reinforcement: number;
	    updated_at: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanDigitalCharacter(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.gender = source["gender"];
	        this.identity = source["identity"];
	        this.worldview = source["worldview"];
	        this.text_model = source["text_model"];
	        this.intimacy = source["intimacy"];
	        this.trust = source["trust"];
	        this.safety = source["safety"];
	        this.conflict = source["conflict"];
	        this.last_interacted_at = source["last_interacted_at"];
	        this.memory_summary = source["memory_summary"];
	        this.highlights = source["highlights"];
	        this.memory_event_count = source["memory_event_count"];
	        this.reinforcement = source["reinforcement"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class HerdsmanDigitalEvent {
	    type: string;
	    title: string;
	    summary: string;
	    occurred_at: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanDigitalEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.occurred_at = source["occurred_at"];
	    }
	}
	export class HerdsmanDigitalLife {
	    available: boolean;
	    source: string;
	    error?: string;
	    warnings: string[];
	    character_count: number;
	    timeline_events: number;
	    state_commits: number;
	    world_events: number;
	    memory_events: number;
	    memory_summaries: number;
	    relationships: number;
	    turn_traces: number;
	    characters: HerdsmanDigitalCharacter[];
	    recent_timeline: HerdsmanDigitalEvent[];
	    recent_world: HerdsmanDigitalEvent[];

	    static createFrom(source: any = {}) {
	        return new HerdsmanDigitalLife(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.source = source["source"];
	        this.error = source["error"];
	        this.warnings = source["warnings"];
	        this.character_count = source["character_count"];
	        this.timeline_events = source["timeline_events"];
	        this.state_commits = source["state_commits"];
	        this.world_events = source["world_events"];
	        this.memory_events = source["memory_events"];
	        this.memory_summaries = source["memory_summaries"];
	        this.relationships = source["relationships"];
	        this.turn_traces = source["turn_traces"];
	        this.characters = this.convertValues(source["characters"], HerdsmanDigitalCharacter);
	        this.recent_timeline = this.convertValues(source["recent_timeline"], HerdsmanDigitalEvent);
	        this.recent_world = this.convertValues(source["recent_world"], HerdsmanDigitalEvent);
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
	export class HerdsmanLaunchPreset {
	    model: string;
	    engine: string;
	    port: number;
	    started_at: string;
	    options: Record<string, any>;

	    static createFrom(source: any = {}) {
	        return new HerdsmanLaunchPreset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.engine = source["engine"];
	        this.port = source["port"];
	        this.started_at = source["started_at"];
	        this.options = source["options"];
	    }
	}
	export class HerdsmanModelStat {
	    model: string;
	    type: string;
	    runtime: string;
	    calls: number;
	    succeeded: number;
	    failed: number;
	    input_tokens: number;
	    output_tokens: number;
	    total_duration_ms: number;
	    avg_duration_ms: number;
	    avg_ttft_ms: number;
	    avg_prompt_tps: number;
	    avg_predicted_tps: number;
	    last_called_at: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanModelStat(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.type = source["type"];
	        this.runtime = source["runtime"];
	        this.calls = source["calls"];
	        this.succeeded = source["succeeded"];
	        this.failed = source["failed"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_duration_ms = source["total_duration_ms"];
	        this.avg_duration_ms = source["avg_duration_ms"];
	        this.avg_ttft_ms = source["avg_ttft_ms"];
	        this.avg_prompt_tps = source["avg_prompt_tps"];
	        this.avg_predicted_tps = source["avg_predicted_tps"];
	        this.last_called_at = source["last_called_at"];
	    }
	}
	export class HerdsmanModelStats {
	    total: number;
	    since: string;
	    per_model: HerdsmanModelStat[];
	    source: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanModelStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.since = source["since"];
	        this.per_model = this.convertValues(source["per_model"], HerdsmanModelStat);
	        this.source = source["source"];
	        this.error = source["error"];
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
	export class HerdsmanOpResult {
	    ok: boolean;
	    status: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanOpResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class HerdsmanOperation {
	    id: string;
	    kind: string;
	    model: string;
	    status: string;
	    stage?: string;
	    progress: number;
	    artifacts: number;
	    created_at: string;
	    completed_at?: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanOperation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.model = source["model"];
	        this.status = source["status"];
	        this.stage = source["stage"];
	        this.progress = source["progress"];
	        this.artifacts = source["artifacts"];
	        this.created_at = source["created_at"];
	        this.completed_at = source["completed_at"];
	    }
	}
	export class HerdsmanOperations {
	    total: number;
	    items: HerdsmanOperation[];
	    source: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new HerdsmanOperations(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.items = this.convertValues(source["items"], HerdsmanOperation);
	        this.source = source["source"];
	        this.error = source["error"];
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
	export class HistoryMessage {
	    role: string;
	    content: string;
	    toolName?: string;
	    toolArgs?: string;
	    toolId?: string;
	    toolOutput?: string;
	    subagentRef?: string;

	    static createFrom(source: any = {}) {
	        return new HistoryMessage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.toolName = source["toolName"];
	        this.toolArgs = source["toolArgs"];
	        this.toolId = source["toolId"];
	        this.toolOutput = source["toolOutput"];
	        this.subagentRef = source["subagentRef"];
	    }
	}
	export class Hit {
	    brain: string;
	    entity: string;
	    text: string;
	    score: number;

	    static createFrom(source: any = {}) {
	        return new Hit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.brain = source["brain"];
	        this.entity = source["entity"];
	        this.text = source["text"];
	        this.score = source["score"];
	    }
	}
	export class ImageDirectorPlan {
	    model: string;
	    prompt: string;
	    negative: string;
	    style: string;
	    loras: string[];
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new ImageDirectorPlan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.prompt = source["prompt"];
	        this.negative = source["negative"];
	        this.style = source["style"];
	        this.loras = source["loras"];
	        this.reason = source["reason"];
	    }
	}
	export class ImageDirectorRequest {
	    need: string;
	    model: string;
	    aspect: string;
	    maxAttempts: number;
	    seed: number;

	    static createFrom(source: any = {}) {
	        return new ImageDirectorRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.need = source["need"];
	        this.model = source["model"];
	        this.aspect = source["aspect"];
	        this.maxAttempts = source["maxAttempts"];
	        this.seed = source["seed"];
	    }
	}
	export class ImageDirectorStep {
	    stage: string;
	    attempt: number;
	    detail: string;

	    static createFrom(source: any = {}) {
	        return new ImageDirectorStep(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stage = source["stage"];
	        this.attempt = source["attempt"];
	        this.detail = source["detail"];
	    }
	}
	export class ImageDirectorVerdict {
	    pass: boolean;
	    score: number;
	    issues: string[];
	    fix: string;

	    static createFrom(source: any = {}) {
	        return new ImageDirectorVerdict(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pass = source["pass"];
	        this.score = source["score"];
	        this.issues = source["issues"];
	        this.fix = source["fix"];
	    }
	}
	export class imageItem {
	    image: string;
	    seed: number;
	    time: number;
	    prompt: string;
	    model: string;
	    size: string;
	    kind?: string;
	    file_path?: string;
	    asset_id?: string;
	    parent_id?: string;

	    static createFrom(source: any = {}) {
	        return new imageItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.seed = source["seed"];
	        this.time = source["time"];
	        this.prompt = source["prompt"];
	        this.model = source["model"];
	        this.size = source["size"];
	        this.kind = source["kind"];
	        this.file_path = source["file_path"];
	        this.asset_id = source["asset_id"];
	        this.parent_id = source["parent_id"];
	    }
	}
	export class ImageDirectorResult {
	    images: imageItem[];
	    plan?: ImageDirectorPlan;
	    verdict?: ImageDirectorVerdict;
	    steps: ImageDirectorStep[];
	    attempts: number;
	    accepted: boolean;

	    static createFrom(source: any = {}) {
	        return new ImageDirectorResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.images = this.convertValues(source["images"], imageItem);
	        this.plan = this.convertValues(source["plan"], ImageDirectorPlan);
	        this.verdict = this.convertValues(source["verdict"], ImageDirectorVerdict);
	        this.steps = this.convertValues(source["steps"], ImageDirectorStep);
	        this.attempts = source["attempts"];
	        this.accepted = source["accepted"];
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


	export class ImageHubUsageModelRow {
	    model: string;
	    backend?: string;
	    count: number;
	    unitCost?: string;
	    estCost?: string;

	    static createFrom(source: any = {}) {
	        return new ImageHubUsageModelRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.backend = source["backend"];
	        this.count = source["count"];
	        this.unitCost = source["unitCost"];
	        this.estCost = source["estCost"];
	    }
	}
	export class ImageHubMonthlyUsageReport {
	    month: string;
	    space: string;
	    total: number;
	    freeCount: number;
	    unpriced: number;
	    estimated: string;
	    byModel: ImageHubUsageModelRow[];

	    static createFrom(source: any = {}) {
	        return new ImageHubMonthlyUsageReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.space = source["space"];
	        this.total = source["total"];
	        this.freeCount = source["freeCount"];
	        this.unpriced = source["unpriced"];
	        this.estimated = source["estimated"];
	        this.byModel = this.convertValues(source["byModel"], ImageHubUsageModelRow);
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

	export class IntentResult {
	    reply: string;
	    cardPath?: string;
	    handled: boolean;
	    action?: string;
	    target?: string;

	    static createFrom(source: any = {}) {
	        return new IntentResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reply = source["reply"];
	        this.cardPath = source["cardPath"];
	        this.handled = source["handled"];
	        this.action = source["action"];
	        this.target = source["target"];
	    }
	}
	export class JobView {
	    id: string;
	    kind: string;
	    label: string;
	    status: string;
	    startedAt: number;

	    static createFrom(source: any = {}) {
	        return new JobView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.startedAt = source["startedAt"];
	    }
	}
	export class KnowledgeEntry {
	    name: string;
	    title: string;
	    category: string;
	    phase: string;
	    discipline: string;
	    tags: string[];
	    status: string;
	    version: number;
	    author: string;
	    reviewer: string;
	    source: string;
	    body: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new KnowledgeEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.category = source["category"];
	        this.phase = source["phase"];
	        this.discipline = source["discipline"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.reviewer = source["reviewer"];
	        this.source = source["source"];
	        this.body = source["body"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class KnowledgeHistoryView {
	    name: string;
	    title: string;
	    version: number;
	    category: string;
	    phase: string;
	    discipline: string;
	    tags: string[];
	    status: string;
	    author: string;
	    reviewer: string;
	    source: string;
	    body: string;
	    changedAt: string;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new KnowledgeHistoryView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.version = source["version"];
	        this.category = source["category"];
	        this.phase = source["phase"];
	        this.discipline = source["discipline"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.author = source["author"];
	        this.reviewer = source["reviewer"];
	        this.source = source["source"];
	        this.body = source["body"];
	        this.changedAt = source["changedAt"];
	        this.note = source["note"];
	    }
	}
	export class KnowledgeImportRowView {
	    name: string;
	    title: string;
	    category: string;
	    phase: string;
	    discipline: string;
	    tags: string[];
	    status: string;
	    source: string;
	    body: string;
	    existingName: string;
	    matchNote: string;
	    similarName: string;
	    similarNote: string;
	    raw: string;
	    skip: boolean;
	    skipReason: string;

	    static createFrom(source: any = {}) {
	        return new KnowledgeImportRowView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.category = source["category"];
	        this.phase = source["phase"];
	        this.discipline = source["discipline"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.body = source["body"];
	        this.existingName = source["existingName"];
	        this.matchNote = source["matchNote"];
	        this.similarName = source["similarName"];
	        this.similarNote = source["similarNote"];
	        this.raw = source["raw"];
	        this.skip = source["skip"];
	        this.skipReason = source["skipReason"];
	    }
	}
	export class KnowledgeImportPreview {
	    path: string;
	    fileName: string;
	    columns: string[];
	    unmapped: string[];
	    rows: KnowledgeImportRowView[];
	    message: string;
	    aiUsed: boolean;

	    static createFrom(source: any = {}) {
	        return new KnowledgeImportPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.columns = source["columns"];
	        this.unmapped = source["unmapped"];
	        this.rows = this.convertValues(source["rows"], KnowledgeImportRowView);
	        this.message = source["message"];
	        this.aiUsed = source["aiUsed"];
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

	export class KnowledgeSummary {
	    name: string;
	    title: string;
	    category: string;
	    phase: string;
	    tags: string[];
	    status: string;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new KnowledgeSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.category = source["category"];
	        this.phase = source["phase"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class MCPServerInput {
	    name: string;
	    transport: string;
	    command: string;
	    args: string[];
	    url: string;
	    env: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new MCPServerInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.transport = source["transport"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.url = source["url"];
	        this.env = source["env"];
	    }
	}
	export class MemoryArchivedView {
	    name: string;
	    title?: string;
	    description: string;
	    type: string;
	    kind: string;
	    archivedAt: string;

	    static createFrom(source: any = {}) {
	        return new MemoryArchivedView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.kind = source["kind"];
	        this.archivedAt = source["archivedAt"];
	    }
	}
	export class MemoryArchivedPage {
	    items: MemoryArchivedView[];
	    total: number;
	    limit: number;
	    offset: number;
	    retentionDays: number;

	    static createFrom(source: any = {}) {
	        return new MemoryArchivedPage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], MemoryArchivedView);
	        this.total = source["total"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	        this.retentionDays = source["retentionDays"];
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

	export class MemoryDoc {
	    path: string;
	    scope: string;
	    body: string;

	    static createFrom(source: any = {}) {
	        return new MemoryDoc(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.scope = source["scope"];
	        this.body = source["body"];
	    }
	}
	export class MemoryDuplicateView {
	    keep: string;
	    keepTitle: string;
	    dup: string;
	    dupTitle: string;
	    score: number;

	    static createFrom(source: any = {}) {
	        return new MemoryDuplicateView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keep = source["keep"];
	        this.keepTitle = source["keepTitle"];
	        this.dup = source["dup"];
	        this.dupTitle = source["dupTitle"];
	        this.score = source["score"];
	    }
	}
	export class MemoryEvalReport {
	    memoryEnabled: boolean;
	    morningPreload: boolean;
	    projectBrief: boolean;
	    spaceModeOn: boolean;
	    preloadPresent: boolean;
	    briefPresent: boolean;
	    preloadRunes: number;
	    briefRunes: number;
	    preloadBudget: number;
	    briefBudget: number;
	    entryCount: number;
	    refCount: number;
	    pinnedTotal: number;
	    pinnedInBrief: number;
	    missingPinned?: string[];
	    violations?: string[];
	    passed: boolean;
	    note?: string;

	    static createFrom(source: any = {}) {
	        return new MemoryEvalReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.memoryEnabled = source["memoryEnabled"];
	        this.morningPreload = source["morningPreload"];
	        this.projectBrief = source["projectBrief"];
	        this.spaceModeOn = source["spaceModeOn"];
	        this.preloadPresent = source["preloadPresent"];
	        this.briefPresent = source["briefPresent"];
	        this.preloadRunes = source["preloadRunes"];
	        this.briefRunes = source["briefRunes"];
	        this.preloadBudget = source["preloadBudget"];
	        this.briefBudget = source["briefBudget"];
	        this.entryCount = source["entryCount"];
	        this.refCount = source["refCount"];
	        this.pinnedTotal = source["pinnedTotal"];
	        this.pinnedInBrief = source["pinnedInBrief"];
	        this.missingPinned = source["missingPinned"];
	        this.violations = source["violations"];
	        this.passed = source["passed"];
	        this.note = source["note"];
	    }
	}
	export class MemoryFact {
	    name: string;
	    title?: string;
	    description: string;
	    type: string;
	    body: string;
	    lastUsedAt?: string;
	    sourceSession?: string;
	    sourceMessage?: string;
	    pinned?: boolean;

	    static createFrom(source: any = {}) {
	        return new MemoryFact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.body = source["body"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.sourceSession = source["sourceSession"];
	        this.sourceMessage = source["sourceMessage"];
	        this.pinned = source["pinned"];
	    }
	}
	export class MemoryGraphView {
	    nodes: GraphNode[];
	    links: GraphLink[];

	    static createFrom(source: any = {}) {
	        return new MemoryGraphView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], GraphNode);
	        this.links = this.convertValues(source["links"], GraphLink);
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
	export class MemoryHubOverview {
	    knowledgeCount: number;
	    profileCount: number;
	    officeCount: number;
	    costCount: number;
	    whisperCount: number;
	    pinnedCount: number;
	    sessionCount: number;
	    uploadCount: number;
	    latestUpdated: string;

	    static createFrom(source: any = {}) {
	        return new MemoryHubOverview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.knowledgeCount = source["knowledgeCount"];
	        this.profileCount = source["profileCount"];
	        this.officeCount = source["officeCount"];
	        this.costCount = source["costCount"];
	        this.whisperCount = source["whisperCount"];
	        this.pinnedCount = source["pinnedCount"];
	        this.sessionCount = source["sessionCount"];
	        this.uploadCount = source["uploadCount"];
	        this.latestUpdated = source["latestUpdated"];
	    }
	}
	export class MemoryLifecycleItem {
	    name: string;
	    title?: string;
	    description?: string;
	    kind: string;
	    type: string;
	    space?: string;
	    score: number;
	    daysIdle: number;
	    lastUsedAt?: string;

	    static createFrom(source: any = {}) {
	        return new MemoryLifecycleItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.kind = source["kind"];
	        this.type = source["type"];
	        this.space = source["space"];
	        this.score = source["score"];
	        this.daysIdle = source["daysIdle"];
	        this.lastUsedAt = source["lastUsedAt"];
	    }
	}
	export class MemoryLifecycleView {
	    pinned: MemoryLifecycleItem[];
	    decaying: MemoryLifecycleItem[];
	    activeCount: number;
	    archivedCount: number;
	    retentionDays: number;
	    staleAfterDays: number;

	    static createFrom(source: any = {}) {
	        return new MemoryLifecycleView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pinned = this.convertValues(source["pinned"], MemoryLifecycleItem);
	        this.decaying = this.convertValues(source["decaying"], MemoryLifecycleItem);
	        this.activeCount = source["activeCount"];
	        this.archivedCount = source["archivedCount"];
	        this.retentionDays = source["retentionDays"];
	        this.staleAfterDays = source["staleAfterDays"];
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
	export class MemoryScope {
	    scope: string;
	    path: string;

	    static createFrom(source: any = {}) {
	        return new MemoryScope(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scope = source["scope"];
	        this.path = source["path"];
	    }
	}
	export class MemorySuggestionView {
	    id: string;
	    name: string;
	    title?: string;
	    description: string;
	    type: string;
	    body: string;
	    reason: string;
	    evidence: string[];

	    static createFrom(source: any = {}) {
	        return new MemorySuggestionView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.body = source["body"];
	        this.reason = source["reason"];
	        this.evidence = source["evidence"];
	    }
	}
	export class MergeSuggestionView {
	    id: string;
	    keep: string;
	    archive: string;
	    keepTitle?: string;
	    archiveTitle?: string;
	    keepUpdatedAt?: string;
	    archiveUpdatedAt?: string;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new MergeSuggestionView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.keep = source["keep"];
	        this.archive = source["archive"];
	        this.keepTitle = source["keepTitle"];
	        this.archiveTitle = source["archiveTitle"];
	        this.keepUpdatedAt = source["keepUpdatedAt"];
	        this.archiveUpdatedAt = source["archiveUpdatedAt"];
	        this.reason = source["reason"];
	    }
	}
	export class SkillSuggestionView {
	    id: string;
	    name: string;
	    description: string;
	    scope: string;
	    body: string;
	    reason: string;
	    evidence: string[];

	    static createFrom(source: any = {}) {
	        return new SkillSuggestionView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.scope = source["scope"];
	        this.body = source["body"];
	        this.reason = source["reason"];
	        this.evidence = source["evidence"];
	    }
	}
	export class MemorySuggestionsView {
	    memories: MemorySuggestionView[];
	    skills: SkillSuggestionView[];
	    merges?: MergeSuggestionView[];
	    generatedAt: string;
	    available: boolean;
	    source: string;

	    static createFrom(source: any = {}) {
	        return new MemorySuggestionsView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.memories = this.convertValues(source["memories"], MemorySuggestionView);
	        this.skills = this.convertValues(source["skills"], SkillSuggestionView);
	        this.merges = this.convertValues(source["merges"], MergeSuggestionView);
	        this.generatedAt = source["generatedAt"];
	        this.available = source["available"];
	        this.source = source["source"];
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
	export class MemoryView {
	    docs: MemoryDoc[];
	    facts: MemoryFact[];
	    scopes: MemoryScope[];
	    storeDir: string;
	    available: boolean;
	    enabled: boolean;
	    dreamMode: string;

	    static createFrom(source: any = {}) {
	        return new MemoryView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.docs = this.convertValues(source["docs"], MemoryDoc);
	        this.facts = this.convertValues(source["facts"], MemoryFact);
	        this.scopes = this.convertValues(source["scopes"], MemoryScope);
	        this.storeDir = source["storeDir"];
	        this.available = source["available"];
	        this.enabled = source["enabled"];
	        this.dreamMode = source["dreamMode"];
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

	export class Meta {
	    label: string;
	    subagentLabel?: string;
	    ready: boolean;
	    startupErr?: string;
	    eventChannel: string;
	    cwd: string;
	    bypass: boolean;
	    permLevel: string;

	    static createFrom(source: any = {}) {
	        return new Meta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.subagentLabel = source["subagentLabel"];
	        this.ready = source["ready"];
	        this.startupErr = source["startupErr"];
	        this.eventChannel = source["eventChannel"];
	        this.cwd = source["cwd"];
	        this.bypass = source["bypass"];
	        this.permLevel = source["permLevel"];
	    }
	}
	export class ModelInfo {
	    ref: string;
	    provider: string;
	    model: string;
	    current: boolean;
	    local?: boolean;
	    status?: string;
	    label?: string;

	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.current = source["current"];
	        this.local = source["local"];
	        this.status = source["status"];
	        this.label = source["label"];
	    }
	}
	export class ModelSwitchEstimate {
	    engine: string;
	    model: string;
	    status: string;
	    wait_seconds: number;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new ModelSwitchEstimate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.model = source["model"];
	        this.status = source["status"];
	        this.wait_seconds = source["wait_seconds"];
	        this.note = source["note"];
	    }
	}
	export class NovelBookSourceCandidate {
	    kind: string;
	    source: string;
	    title: string;
	    author?: string;
	    url: string;
	    host?: string;
	    hasRule: boolean;
	    latestChapter?: string;

	    static createFrom(source: any = {}) {
	        return new NovelBookSourceCandidate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.source = source["source"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.url = source["url"];
	        this.host = source["host"];
	        this.hasRule = source["hasRule"];
	        this.latestChapter = source["latestChapter"];
	    }
	}
	export class NovelBookSourceEnginesPayload {
	    path: string;
	    rules: booksource.SearchEngineRule[];

	    static createFrom(source: any = {}) {
	        return new NovelBookSourceEnginesPayload(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.rules = this.convertValues(source["rules"], booksource.SearchEngineRule);
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
	export class NovelBookSourceImportStart {
	    jobId: string;

	    static createFrom(source: any = {}) {
	        return new NovelBookSourceImportStart(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	    }
	}
	export class NovelBookSourceSearchResult {
	    candidates: NovelBookSourceCandidate[];
	    warnings?: string[];

	    static createFrom(source: any = {}) {
	        return new NovelBookSourceSearchResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidates = this.convertValues(source["candidates"], NovelBookSourceCandidate);
	        this.warnings = source["warnings"];
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
	export class NovelBookSourceTocPreview {
	    total: number;
	    sample: booksource.TocEntry[];
	    truncated: boolean;

	    static createFrom(source: any = {}) {
	        return new NovelBookSourceTocPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.sample = this.convertValues(source["sample"], booksource.TocEntry);
	        this.truncated = source["truncated"];
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
	export class NovelImportResult {
	    path: string;
	    title: string;
	    chapter_count: number;
	    total_words: number;
	    encoding?: string;
	    split_strategy?: string;
	    warnings?: bookimport.Warning[];

	    static createFrom(source: any = {}) {
	        return new NovelImportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.chapter_count = source["chapter_count"];
	        this.total_words = source["total_words"];
	        this.encoding = source["encoding"];
	        this.split_strategy = source["split_strategy"];
	        this.warnings = this.convertValues(source["warnings"], bookimport.Warning);
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
	export class OutlineReconstructItem {
	    chapterNumber: number;
	    chapterFrom?: number;
	    chapterTo?: number;
	    title: string;
	    summary: string;
	    scenes?: string[];
	    characters?: string[];
	    keyPoints?: string[];
	    emotion?: string;
	    goal?: string;

	    static createFrom(source: any = {}) {
	        return new OutlineReconstructItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterNumber = source["chapterNumber"];
	        this.chapterFrom = source["chapterFrom"];
	        this.chapterTo = source["chapterTo"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.scenes = source["scenes"];
	        this.characters = source["characters"];
	        this.keyPoints = source["keyPoints"];
	        this.emotion = source["emotion"];
	        this.goal = source["goal"];
	    }
	}
	export class OutlineReconstructPreview {
	    aiUsed: boolean;
	    projectTitle: string;
	    description?: string;
	    theme?: string;
	    genre?: string;
	    narrativePerspective?: string;
	    targetWords?: number;
	    items: OutlineReconstructItem[];
	    tier?: string;
	    segmentSize?: number;
	    warnings?: string[];

	    static createFrom(source: any = {}) {
	        return new OutlineReconstructPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aiUsed = source["aiUsed"];
	        this.projectTitle = source["projectTitle"];
	        this.description = source["description"];
	        this.theme = source["theme"];
	        this.genre = source["genre"];
	        this.narrativePerspective = source["narrativePerspective"];
	        this.targetWords = source["targetWords"];
	        this.items = this.convertValues(source["items"], OutlineReconstructItem);
	        this.tier = source["tier"];
	        this.segmentSize = source["segmentSize"];
	        this.warnings = source["warnings"];
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
	export class NovelOutlineReconstructTaskState {
	    taskId: string;
	    status: string;
	    message?: string;
	    error?: string;
	    preview?: OutlineReconstructPreview;

	    static createFrom(source: any = {}) {
	        return new NovelOutlineReconstructTaskState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.error = source["error"];
	        this.preview = this.convertValues(source["preview"], OutlineReconstructPreview);
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
	export class NovelSearchHit {
	    node_id: string;
	    title: string;
	    chapter_num: number;
	    branch?: string;
	    snippet: string;
	    title_hit: boolean;
	    match_index: number;
	    paragraph_index: number;
	    char_offset: number;
	    match_len: number;
	    total_hits: number;
	    chapter_count: number;

	    static createFrom(source: any = {}) {
	        return new NovelSearchHit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.node_id = source["node_id"];
	        this.title = source["title"];
	        this.chapter_num = source["chapter_num"];
	        this.branch = source["branch"];
	        this.snippet = source["snippet"];
	        this.title_hit = source["title_hit"];
	        this.match_index = source["match_index"];
	        this.paragraph_index = source["paragraph_index"];
	        this.char_offset = source["char_offset"];
	        this.match_len = source["match_len"];
	        this.total_hits = source["total_hits"];
	        this.chapter_count = source["chapter_count"];
	    }
	}


	export class PermissionsView {
	    mode: string;
	    allow: string[];
	    ask: string[];
	    deny: string[];

	    static createFrom(source: any = {}) {
	        return new PermissionsView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.allow = source["allow"];
	        this.ask = source["ask"];
	        this.deny = source["deny"];
	    }
	}
	export class PptxSlideOutline {
	    index: number;
	    title: string;
	    texts: string[];
	    shapeCount: number;

	    static createFrom(source: any = {}) {
	        return new PptxSlideOutline(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.title = source["title"];
	        this.texts = source["texts"];
	        this.shapeCount = source["shapeCount"];
	    }
	}
	export class PptxOutlineView {
	    available: boolean;
	    error?: string;
	    slides: PptxSlideOutline[];

	    static createFrom(source: any = {}) {
	        return new PptxOutlineView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.error = source["error"];
	        this.slides = this.convertValues(source["slides"], PptxSlideOutline);
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

	export class PreviewPage {
	    page: number;
	    dataUrl: string;

	    static createFrom(source: any = {}) {
	        return new PreviewPage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.dataUrl = source["dataUrl"];
	    }
	}
	export class PreviewResult {
	    path: string;
	    name: string;
	    ext: string;
	    size: number;
	    kind: string;
	    body: string;
	    dataUrl: string;
	    error: string;
	    truncated?: boolean;
	    totalPages?: number;
	    pages?: PreviewPage[];
	    hint?: string;

	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.ext = source["ext"];
	        this.size = source["size"];
	        this.kind = source["kind"];
	        this.body = source["body"];
	        this.dataUrl = source["dataUrl"];
	        this.error = source["error"];
	        this.truncated = source["truncated"];
	        this.totalPages = source["totalPages"];
	        this.pages = this.convertValues(source["pages"], PreviewPage);
	        this.hint = source["hint"];
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
	export class PriceApplyResult {
	    applied: number;
	    resourceSynced: number;
	    resourceUnmatched: number;

	    static createFrom(source: any = {}) {
	        return new PriceApplyResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.resourceSynced = source["resourceSynced"];
	        this.resourceUnmatched = source["resourceUnmatched"];
	    }
	}
	export class ProfileFactView {
	    name: string;
	    title: string;
	    description: string;
	    type: string;
	    kind: string;
	    tags: string[];
	    body: string;

	    static createFrom(source: any = {}) {
	        return new ProfileFactView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.kind = source["kind"];
	        this.tags = source["tags"];
	        this.body = source["body"];
	    }
	}
	export class ProjectCard {
	    title: string;
	    genre: string;
	    style: string;
	    path: string;
	    word_count: number;
	    chapter_count: number;
	    created_at: string;
	    last_opened_at: string;
	    mature?: string;

	    static createFrom(source: any = {}) {
	        return new ProjectCard(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.genre = source["genre"];
	        this.style = source["style"];
	        this.path = source["path"];
	        this.word_count = source["word_count"];
	        this.chapter_count = source["chapter_count"];
	        this.created_at = source["created_at"];
	        this.last_opened_at = source["last_opened_at"];
	        this.mature = source["mature"];
	    }
	}
	export class SessionMeta {
	    path: string;
	    preview: string;
	    title?: string;
	    turns: number;
	    modTime: number;
	    current: boolean;
	    pinned: boolean;
	    archived: boolean;
	    interrupted: boolean;
	    spaceId?: string;

	    static createFrom(source: any = {}) {
	        return new SessionMeta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.preview = source["preview"];
	        this.title = source["title"];
	        this.turns = source["turns"];
	        this.modTime = source["modTime"];
	        this.current = source["current"];
	        this.pinned = source["pinned"];
	        this.archived = source["archived"];
	        this.interrupted = source["interrupted"];
	        this.spaceId = source["spaceId"];
	    }
	}
	export class ProjectGroup {
	    path: string;
	    name: string;
	    current: boolean;
	    sessions: SessionMeta[];
	    archived: SessionMeta[];
	    modTime: number;

	    static createFrom(source: any = {}) {
	        return new ProjectGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.current = source["current"];
	        this.sessions = this.convertValues(source["sessions"], SessionMeta);
	        this.archived = this.convertValues(source["archived"], SessionMeta);
	        this.modTime = source["modTime"];
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
	export class PromptPreviewResult {
	    systemPrompt: string;
	    warnings: string[];

	    static createFrom(source: any = {}) {
	        return new PromptPreviewResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.systemPrompt = source["systemPrompt"];
	        this.warnings = source["warnings"];
	    }
	}
	export class PromptSaveResult {
	    saved: boolean;
	    issues: promptstore.Issue[];
	    version: number;

	    static createFrom(source: any = {}) {
	        return new PromptSaveResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saved = source["saved"];
	        this.issues = this.convertValues(source["issues"], promptstore.Issue);
	        this.version = source["version"];
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
	export class PromptTemplateMeta {
	    key: string;
	    category: string;
	    description: string;
	    source: string;
	    hasOverride: boolean;
	    overrideActive: boolean;
	    version: number;
	    updatedAt: number;

	    static createFrom(source: any = {}) {
	        return new PromptTemplateMeta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.source = source["source"];
	        this.hasOverride = source["hasOverride"];
	        this.overrideActive = source["overrideActive"];
	        this.version = source["version"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class PromptTemplateDetail {
	    meta: PromptTemplateMeta;
	    template: prompt.Template;
	    base: prompt.Template;

	    static createFrom(source: any = {}) {
	        return new PromptTemplateDetail(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], PromptTemplateMeta);
	        this.template = this.convertValues(source["template"], prompt.Template);
	        this.base = this.convertValues(source["base"], prompt.Template);
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

	export class ProviderView {
	    name: string;
	    kind: string;
	    baseUrl: string;
	    models: string[];
	    default: string;
	    apiKeyEnv: string;
	    keySet: boolean;
	    balanceUrl: string;
	    contextWindow: number;
	    oauthKind: string;
	    oauthReady: boolean;

	    static createFrom(source: any = {}) {
	        return new ProviderView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.baseUrl = source["baseUrl"];
	        this.models = source["models"];
	        this.default = source["default"];
	        this.apiKeyEnv = source["apiKeyEnv"];
	        this.keySet = source["keySet"];
	        this.balanceUrl = source["balanceUrl"];
	        this.contextWindow = source["contextWindow"];
	        this.oauthKind = source["oauthKind"];
	        this.oauthReady = source["oauthReady"];
	    }
	}
	export class RetrievalEvalQuery {
	    query: string;
	    expected: string[];
	    topHits: string[];
	    recall: number;

	    static createFrom(source: any = {}) {
	        return new RetrievalEvalQuery(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.expected = source["expected"];
	        this.topHits = source["topHits"];
	        this.recall = source["recall"];
	    }
	}
	export class RetrievalEvalReport {
	    total: number;
	    threshold: number;
	    recallAt10: number;
	    passed: boolean;
	    perQuery: RetrievalEvalQuery[];
	    note?: string;

	    static createFrom(source: any = {}) {
	        return new RetrievalEvalReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.threshold = source["threshold"];
	        this.recallAt10 = source["recallAt10"];
	        this.passed = source["passed"];
	        this.perQuery = this.convertValues(source["perQuery"], RetrievalEvalQuery);
	        this.note = source["note"];
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
	export class ReviewPlatformView {
	    id: string;
	    label: string;
	    form: string;

	    static createFrom(source: any = {}) {
	        return new ReviewPlatformView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.form = source["form"];
	    }
	}
	export class RouteLedgerTotal {
	    calls: number;
	    tokens_in: number;
	    tokens_out: number;
	    cost_cny: number;
	    avg_ms: number;
	    success_rate: number;

	    static createFrom(source: any = {}) {
	        return new RouteLedgerTotal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.calls = source["calls"];
	        this.tokens_in = source["tokens_in"];
	        this.tokens_out = source["tokens_out"];
	        this.cost_cny = source["cost_cny"];
	        this.avg_ms = source["avg_ms"];
	        this.success_rate = source["success_rate"];
	    }
	}
	export class RouteLedgerView {
	    generated_at: string;
	    total: RouteLedgerTotal;
	    features: modelengine.FeatureUsageSummary[];

	    static createFrom(source: any = {}) {
	        return new RouteLedgerView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generated_at = source["generated_at"];
	        this.total = this.convertValues(source["total"], RouteLedgerTotal);
	        this.features = this.convertValues(source["features"], modelengine.FeatureUsageSummary);
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
	export class RouteSuggestionsView {
	    generated_at: string;
	    suggestions: routesuggest.Suggestion[];

	    static createFrom(source: any = {}) {
	        return new RouteSuggestionsView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generated_at = source["generated_at"];
	        this.suggestions = this.convertValues(source["suggestions"], routesuggest.Suggestion);
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
	export class SandboxView {
	    bash: string;
	    network: boolean;
	    workspaceRoot: string;
	    allowWrite: string[];

	    static createFrom(source: any = {}) {
	        return new SandboxView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bash = source["bash"];
	        this.network = source["network"];
	        this.workspaceRoot = source["workspaceRoot"];
	        this.allowWrite = source["allowWrite"];
	    }
	}
	export class SavingsView {
	    ref_price_per_mtok: number;
	    would_cost_cloud: number;
	    saved: number;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new SavingsView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref_price_per_mtok = source["ref_price_per_mtok"];
	        this.would_cost_cloud = source["would_cost_cloud"];
	        this.saved = source["saved"];
	        this.note = source["note"];
	    }
	}
	export class SceneBibleCharView {
	    name: string;
	    roleType?: string;
	    status?: string;
	    location?: string;
	    currentState?: string;
	    careerMain?: string;
	    careerSub?: string[];
	    items?: string[];
	    knownBy?: string[];

	    static createFrom(source: any = {}) {
	        return new SceneBibleCharView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.roleType = source["roleType"];
	        this.status = source["status"];
	        this.location = source["location"];
	        this.currentState = source["currentState"];
	        this.careerMain = source["careerMain"];
	        this.careerSub = source["careerSub"];
	        this.items = source["items"];
	        this.knownBy = source["knownBy"];
	    }
	}
	export class SceneBibleView {
	    pov?: string;
	    title?: string;
	    location?: string;
	    timeOfDay?: string;
	    mood?: string;
	    tags?: string[];
	    characters: SceneBibleCharView[];
	    povView?: string;
	    hiddenFacts: string[];
	    foreshadows: string[];
	    memories: string[];
	    timeAnchor?: string;
	    style?: string;
	    thread?: string;

	    static createFrom(source: any = {}) {
	        return new SceneBibleView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pov = source["pov"];
	        this.title = source["title"];
	        this.location = source["location"];
	        this.timeOfDay = source["timeOfDay"];
	        this.mood = source["mood"];
	        this.tags = source["tags"];
	        this.characters = this.convertValues(source["characters"], SceneBibleCharView);
	        this.povView = source["povView"];
	        this.hiddenFacts = source["hiddenFacts"];
	        this.foreshadows = source["foreshadows"];
	        this.memories = source["memories"];
	        this.timeAnchor = source["timeAnchor"];
	        this.style = source["style"];
	        this.thread = source["thread"];
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
	export class ScheduleCreateResult {
	    rel: string;
	    current: string;

	    static createFrom(source: any = {}) {
	        return new ScheduleCreateResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rel = source["rel"];
	        this.current = source["current"];
	    }
	}
	export class ScheduleLoadResult {
	    path: string;
	    exists: boolean;
	    project: string;

	    static createFrom(source: any = {}) {
	        return new ScheduleLoadResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.exists = source["exists"];
	        this.project = source["project"];
	    }
	}
	export class ScheduleOpenResult {
	    current: string;

	    static createFrom(source: any = {}) {
	        return new ScheduleOpenResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	    }
	}
	export class ScheduleProjectSummary {
	    rel: string;
	    name: string;
	    archived: boolean;
	    duration: number;
	    taskCount: number;
	    ok: boolean;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new ScheduleProjectSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rel = source["rel"];
	        this.name = source["name"];
	        this.archived = source["archived"];
	        this.duration = source["duration"];
	        this.taskCount = source["taskCount"];
	        this.ok = source["ok"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ScheduleProjectsResult {
	    current: string;
	    projects: ScheduleProjectSummary[];

	    static createFrom(source: any = {}) {
	        return new ScheduleProjectsResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.projects = this.convertValues(source["projects"], ScheduleProjectSummary);
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
	export class ScheduleSaveResult {
	    path: string;
	    savedAt: string;
	    duration: number;
	    critical: number;

	    static createFrom(source: any = {}) {
	        return new ScheduleSaveResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.savedAt = source["savedAt"];
	        this.duration = source["duration"];
	        this.critical = source["critical"];
	    }
	}
	export class SemanticGraphView {
	    nodes: GraphNode[];
	    links: GraphLink[];
	    eventCount: number;
	    entityCount: number;
	    danglingRefs: string[];
	    projectedSeq: number;

	    static createFrom(source: any = {}) {
	        return new SemanticGraphView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], GraphNode);
	        this.links = this.convertValues(source["links"], GraphLink);
	        this.eventCount = source["eventCount"];
	        this.entityCount = source["entityCount"];
	        this.danglingRefs = source["danglingRefs"];
	        this.projectedSeq = source["projectedSeq"];
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
	export class SemanticHitView {
	    kind: string;
	    name: string;
	    score: number;
	    text: string;

	    static createFrom(source: any = {}) {
	        return new SemanticHitView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.score = source["score"];
	        this.text = source["text"];
	    }
	}
	export class SemanticIndexStatus {
	    available: boolean;
	    counts: Record<string, number>;
	    detail?: Record<string, string>;
	    error?: string;
	    costTotal?: number;
	    costIndexed?: number;

	    static createFrom(source: any = {}) {
	        return new SemanticIndexStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.counts = source["counts"];
	        this.detail = source["detail"];
	        this.error = source["error"];
	        this.costTotal = source["costTotal"];
	        this.costIndexed = source["costIndexed"];
	    }
	}


	export class SessionStatsView {
	    available: boolean;
	    stats: session.TokenStats;

	    static createFrom(source: any = {}) {
	        return new SessionStatsView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.stats = this.convertValues(source["stats"], session.TokenStats);
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
	export class SettingsView {
	    defaultModel: string;
	    subagentModel: string;
	    subagentModels: Record<string, string>;
	    subagentSkills: string[];
	    providers: ProviderView[];
	    permissions: PermissionsView;
	    sandbox: SandboxView;
	    agent: AgentView;
	    configPath: string;
	    providerKinds: string[];
	    bypass: boolean;

	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultModel = source["defaultModel"];
	        this.subagentModel = source["subagentModel"];
	        this.subagentModels = source["subagentModels"];
	        this.subagentSkills = source["subagentSkills"];
	        this.providers = this.convertValues(source["providers"], ProviderView);
	        this.permissions = this.convertValues(source["permissions"], PermissionsView);
	        this.sandbox = this.convertValues(source["sandbox"], SandboxView);
	        this.agent = this.convertValues(source["agent"], AgentView);
	        this.configPath = source["configPath"];
	        this.providerKinds = source["providerKinds"];
	        this.bypass = source["bypass"];
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
	export class SimilarView {
	    name: string;
	    title: string;
	    score: number;

	    static createFrom(source: any = {}) {
	        return new SimilarView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.score = source["score"];
	    }
	}
	export class SinBookSourceBook {
	    title: string;
	    path: string;
	    sizeBytes: number;
	    modifiedAt: string;

	    static createFrom(source: any = {}) {
	        return new SinBookSourceBook(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}
	export class SinBookSourceDownloadStart {
	    jobId: string;

	    static createFrom(source: any = {}) {
	        return new SinBookSourceDownloadStart(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	    }
	}
	export class SinMessagesPageResult {
	    messages: chat.Message[];
	    has_more: boolean;

	    static createFrom(source: any = {}) {
	        return new SinMessagesPageResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messages = this.convertValues(source["messages"], chat.Message);
	        this.has_more = source["has_more"];
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
	export class SinNotesView {
	    notes: string[];
	    outline: string;

	    static createFrom(source: any = {}) {
	        return new SinNotesView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notes = source["notes"];
	        this.outline = source["outline"];
	    }
	}
	export class SkillCaptureInput {
	    name: string;
	    description: string;
	    task: string;
	    solution: string;

	    static createFrom(source: any = {}) {
	        return new SkillCaptureInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.task = source["task"];
	        this.solution = source["solution"];
	    }
	}
	export class SkillCaptureResult {
	    name: string;
	    description: string;
	    path: string;
	    reloaded: boolean;
	    tools: number;
	    skills: number;

	    static createFrom(source: any = {}) {
	        return new SkillCaptureResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.path = source["path"];
	        this.reloaded = source["reloaded"];
	        this.tools = source["tools"];
	        this.skills = source["skills"];
	    }
	}
	export class SkillDistillCandidateView {
	    id: string;
	    pattern: string[];
	    repeat: number;
	    sessions: string[];
	    evidence: string[];
	    lastAt: number;

	    static createFrom(source: any = {}) {
	        return new SkillDistillCandidateView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pattern = source["pattern"];
	        this.repeat = source["repeat"];
	        this.sessions = source["sessions"];
	        this.evidence = source["evidence"];
	        this.lastAt = source["lastAt"];
	    }
	}
	export class SkillDistillView {
	    candidates: SkillDistillCandidateView[];
	    available: boolean;
	    generatedAt: string;

	    static createFrom(source: any = {}) {
	        return new SkillDistillView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidates = this.convertValues(source["candidates"], SkillDistillCandidateView);
	        this.available = source["available"];
	        this.generatedAt = source["generatedAt"];
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
	export class SkillDraft {
	    name: string;
	    description: string;
	    scenario: string;
	    steps: string[];
	    cautions: string[];

	    static createFrom(source: any = {}) {
	        return new SkillDraft(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.scenario = source["scenario"];
	        this.steps = source["steps"];
	        this.cautions = source["cautions"];
	    }
	}
	export class SkillRecordResult {
	    draft: SkillDraft;
	    replay: string;
	    preview: string;

	    static createFrom(source: any = {}) {
	        return new SkillRecordResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.draft = this.convertValues(source["draft"], SkillDraft);
	        this.replay = source["replay"];
	        this.preview = source["preview"];
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


	export class SlashArgItem {
	    label: string;
	    insert: string;
	    hint: string;
	    descend: boolean;

	    static createFrom(source: any = {}) {
	        return new SlashArgItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.insert = source["insert"];
	        this.hint = source["hint"];
	        this.descend = source["descend"];
	    }
	}
	export class SlashArgsResult {
	    items: SlashArgItem[];
	    from: number;

	    static createFrom(source: any = {}) {
	        return new SlashArgsResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], SlashArgItem);
	        this.from = source["from"];
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
	export class SpaceActiveView {
	    space: string;
	    modeOn: boolean;
	    exportsDir: string;
	    workDir: string;

	    static createFrom(source: any = {}) {
	        return new SpaceActiveView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.space = source["space"];
	        this.modeOn = source["modeOn"];
	        this.exportsDir = source["exportsDir"];
	        this.workDir = source["workDir"];
	    }
	}
	export class SpaceOption {
	    id: string;
	    title: string;
	    desc: string;

	    static createFrom(source: any = {}) {
	        return new SpaceOption(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.desc = source["desc"];
	    }
	}
	export class SpaceProfileView {
	    space: string;
	    gaea: string;
	    gaeaResolved: string;
	    gaeaOk: boolean;
	    models: Record<string, string>;
	    permMode: string;
	    permHardAskCount: number;
	    permHardAskBySpace: boolean;
	    guardrailsOn: boolean;
	    modeOn: boolean;

	    static createFrom(source: any = {}) {
	        return new SpaceProfileView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.space = source["space"];
	        this.gaea = source["gaea"];
	        this.gaeaResolved = source["gaeaResolved"];
	        this.gaeaOk = source["gaeaOk"];
	        this.models = source["models"];
	        this.permMode = source["permMode"];
	        this.permHardAskCount = source["permHardAskCount"];
	        this.permHardAskBySpace = source["permHardAskBySpace"];
	        this.guardrailsOn = source["guardrailsOn"];
	        this.modeOn = source["modeOn"];
	    }
	}
	export class StoryHealthReport {
	    current_chapter: number;
	    findings: types.StoryHealthFinding[];
	    counts: Record<string, number>;

	    static createFrom(source: any = {}) {
	        return new StoryHealthReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current_chapter = source["current_chapter"];
	        this.findings = this.convertValues(source["findings"], types.StoryHealthFinding);
	        this.counts = source["counts"];
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
	export class StreamProbeResult {
	    model: string;
	    ok: boolean;
	    ttft_ms: number;
	    chunks: number;
	    tokens: number;
	    duration_ms: number;
	    max_gap_ms: number;
	    avg_gap_ms: number;
	    completed: boolean;
	    interrupted: boolean;
	    error?: string;
	    response_start?: string;

	    static createFrom(source: any = {}) {
	        return new StreamProbeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.ok = source["ok"];
	        this.ttft_ms = source["ttft_ms"];
	        this.chunks = source["chunks"];
	        this.tokens = source["tokens"];
	        this.duration_ms = source["duration_ms"];
	        this.max_gap_ms = source["max_gap_ms"];
	        this.avg_gap_ms = source["avg_gap_ms"];
	        this.completed = source["completed"];
	        this.interrupted = source["interrupted"];
	        this.error = source["error"];
	        this.response_start = source["response_start"];
	    }
	}
	export class SubagentRunView {
	    ref: string;
	    status: string;
	    kind: string;
	    tool?: string;
	    model?: string;
	    toolScope?: string[];
	    task: string;
	    answer: string;
	    toolCalls: number;
	    lastText?: string;
	    lastTool?: string;
	    followUpError?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new SubagentRunView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.status = source["status"];
	        this.kind = source["kind"];
	        this.tool = source["tool"];
	        this.model = source["model"];
	        this.toolScope = source["toolScope"];
	        this.task = source["task"];
	        this.answer = source["answer"];
	        this.toolCalls = source["toolCalls"];
	        this.lastText = source["lastText"];
	        this.lastTool = source["lastTool"];
	        this.followUpError = source["followUpError"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class SubagentRunsView {
	    available: boolean;
	    runs: SubagentRunView[];
	    total: number;
	    running: number;

	    static createFrom(source: any = {}) {
	        return new SubagentRunsView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.runs = this.convertValues(source["runs"], SubagentRunView);
	        this.total = source["total"];
	        this.running = source["running"];
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
	export class SubagentTranscriptMessage {
	    role: string;
	    name?: string;
	    content?: string;
	    reasoning?: string;
	    toolCalls?: provider.ToolCall[];
	    toolCallId?: string;

	    static createFrom(source: any = {}) {
	        return new SubagentTranscriptMessage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.name = source["name"];
	        this.content = source["content"];
	        this.reasoning = source["reasoning"];
	        this.toolCalls = this.convertValues(source["toolCalls"], provider.ToolCall);
	        this.toolCallId = source["toolCallId"];
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
	export class SubagentTranscriptView {
	    ref: string;
	    task: string;
	    followUpError?: string;
	    messages: SubagentTranscriptMessage[];

	    static createFrom(source: any = {}) {
	        return new SubagentTranscriptView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.task = source["task"];
	        this.followUpError = source["followUpError"];
	        this.messages = this.convertValues(source["messages"], SubagentTranscriptMessage);
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
	export class TaskInboxView {
	    id: string;
	    title: string;
	    space: string;
	    status: string;
	    source: string;
	    action?: string;
	    target?: string;
	    session?: string;
	    note?: string;
	    createdAt: number;
	    updatedAt: number;

	    static createFrom(source: any = {}) {
	        return new TaskInboxView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.space = source["space"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.session = source["session"];
	        this.note = source["note"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class TaskOutputView {
	    tail: string;
	    truncated: boolean;

	    static createFrom(source: any = {}) {
	        return new TaskOutputView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tail = source["tail"];
	        this.truncated = source["truncated"];
	    }
	}
	export class TaskTemplate {
	    name: string;
	    title: string;
	    description: string;
	    prompt: string;

	    static createFrom(source: any = {}) {
	        return new TaskTemplate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.prompt = source["prompt"];
	    }
	}

	export class WorkspaceSearchHit {
	    path: string;
	    name: string;
	    size: number;
	    modTime: number;
	    score: number;
	    snippet: string;
	    truncated: boolean;
	    skipped?: string;

	    static createFrom(source: any = {}) {
	        return new WorkspaceSearchHit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.score = source["score"];
	        this.snippet = source["snippet"];
	        this.truncated = source["truncated"];
	        this.skipped = source["skipped"];
	    }
	}
	export class UnifiedSearchView {
	    keyword: WorkspaceSearchHit[];
	    semantic: SemanticHitView[];
	    brain: Hit[];
	    files: FileSemanticHit[];

	    static createFrom(source: any = {}) {
	        return new UnifiedSearchView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = this.convertValues(source["keyword"], WorkspaceSearchHit);
	        this.semantic = this.convertValues(source["semantic"], SemanticHitView);
	        this.brain = this.convertValues(source["brain"], Hit);
	        this.files = this.convertValues(source["files"], FileSemanticHit);
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
	export class UpdateInfo {
	    version: string;
	    notes: string;

	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.notes = source["notes"];
	    }
	}
	export class UploadFileRow {
	    name: string;
	    path: string;
	    size: number;
	    modTime: number;
	    kind: string;

	    static createFrom(source: any = {}) {
	        return new UploadFileRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.kind = source["kind"];
	    }
	}
	export class UsageSide {
	    calls: number;
	    success_calls: number;
	    fail_calls: number;
	    input_tokens: number;
	    output_tokens: number;
	    total_tokens: number;
	    cache_hit_tokens: number;
	    cache_miss_tokens: number;
	    total_duration_ms: number;
	    cost: number;
	    engines: string[];

	    static createFrom(source: any = {}) {
	        return new UsageSide(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.calls = source["calls"];
	        this.success_calls = source["success_calls"];
	        this.fail_calls = source["fail_calls"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.cache_hit_tokens = source["cache_hit_tokens"];
	        this.cache_miss_tokens = source["cache_miss_tokens"];
	        this.total_duration_ms = source["total_duration_ms"];
	        this.cost = source["cost"];
	        this.engines = source["engines"];
	    }
	}
	export class UsageOverview {
	    cloud: UsageSide;
	    local: UsageSide;
	    savings: SavingsView;
	    cache_hit_tokens: number;
	    cache_miss_tokens: number;
	    cache_hit_rate: number;

	    static createFrom(source: any = {}) {
	        return new UsageOverview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cloud = this.convertValues(source["cloud"], UsageSide);
	        this.local = this.convertValues(source["local"], UsageSide);
	        this.savings = this.convertValues(source["savings"], SavingsView);
	        this.cache_hit_tokens = source["cache_hit_tokens"];
	        this.cache_miss_tokens = source["cache_miss_tokens"];
	        this.cache_hit_rate = source["cache_hit_rate"];
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

	export class WhisperReplayLine {
	    turnIndex: number;
	    role: string;
	    text: string;

	    static createFrom(source: any = {}) {
	        return new WhisperReplayLine(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turnIndex = source["turnIndex"];
	        this.role = source["role"];
	        this.text = source["text"];
	    }
	}
	export class WhisperEpisodeReplayView {
	    id: string;
	    summary: string;
	    dominantEmotion: string;
	    emotionalIntensity: number;
	    keywords: string[];
	    // Go type: time
	    createdAt: any;
	    sourceSessionId: string;
	    startTurn: number;
	    endTurn: number;
	    dialogue: WhisperReplayLine[];
	    replayable: boolean;

	    static createFrom(source: any = {}) {
	        return new WhisperEpisodeReplayView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.dominantEmotion = source["dominantEmotion"];
	        this.emotionalIntensity = source["emotionalIntensity"];
	        this.keywords = source["keywords"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.sourceSessionId = source["sourceSessionId"];
	        this.startTurn = source["startTurn"];
	        this.endTurn = source["endTurn"];
	        this.dialogue = this.convertValues(source["dialogue"], WhisperReplayLine);
	        this.replayable = source["replayable"];
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
	export class WhisperAnchorReplayView {
	    anchorId: string;
	    anchorDate: string;
	    anchorType: string;
	    domain: string;
	    summary: string;
	    emotionalValence: number;
	    emotionalIntensity: number;
	    linkedFactSummaries: string[];
	    episodeReplay?: WhisperEpisodeReplayView;
	    replayable: boolean;

	    static createFrom(source: any = {}) {
	        return new WhisperAnchorReplayView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anchorId = source["anchorId"];
	        this.anchorDate = source["anchorDate"];
	        this.anchorType = source["anchorType"];
	        this.domain = source["domain"];
	        this.summary = source["summary"];
	        this.emotionalValence = source["emotionalValence"];
	        this.emotionalIntensity = source["emotionalIntensity"];
	        this.linkedFactSummaries = source["linkedFactSummaries"];
	        this.episodeReplay = this.convertValues(source["episodeReplay"], WhisperEpisodeReplayView);
	        this.replayable = source["replayable"];
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
	export class WhisperAnchorView {
	    id: string;
	    anchorDate: string;
	    anchorType: string;
	    recurrenceRule: string;
	    domain: string;
	    summary: string;
	    emotionalValence: number;
	    emotionalIntensity: number;
	    linkedFactIds: string[];

	    static createFrom(source: any = {}) {
	        return new WhisperAnchorView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.anchorDate = source["anchorDate"];
	        this.anchorType = source["anchorType"];
	        this.recurrenceRule = source["recurrenceRule"];
	        this.domain = source["domain"];
	        this.summary = source["summary"];
	        this.emotionalValence = source["emotionalValence"];
	        this.emotionalIntensity = source["emotionalIntensity"];
	        this.linkedFactIds = source["linkedFactIds"];
	    }
	}

	export class WhisperEpisodeView {
	    id: string;
	    summary: string;
	    dominantEmotion: string;
	    emotionalIntensity: number;
	    keywords: string[];
	    startTurn: number;
	    endTurn: number;
	    // Go type: time
	    createdAt: any;
	    sourceSessionId: string;

	    static createFrom(source: any = {}) {
	        return new WhisperEpisodeView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.dominantEmotion = source["dominantEmotion"];
	        this.emotionalIntensity = source["emotionalIntensity"];
	        this.keywords = source["keywords"];
	        this.startTurn = source["startTurn"];
	        this.endTurn = source["endTurn"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.sourceSessionId = source["sourceSessionId"];
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
	export class WhisperMemoryView {
	    id: string;
	    domain: string;
	    subcategory: string;
	    subject: string;
	    summary: string;
	    weight: number;
	    confidence: number;
	    tier: string;
	    status: string;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new WhisperMemoryView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.domain = source["domain"];
	        this.subcategory = source["subcategory"];
	        this.subject = source["subject"];
	        this.summary = source["summary"];
	        this.weight = source["weight"];
	        this.confidence = source["confidence"];
	        this.tier = source["tier"];
	        this.status = source["status"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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


	export class WorkspaceView {
	    path: string;
	    name: string;
	    current: boolean;

	    static createFrom(source: any = {}) {
	        return new WorkspaceView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.current = source["current"];
	    }
	}
	export class XlsxChartInput {
	    rel: string;
	    sheet: string;
	    refs: string;
	    chartType: string;
	    title: string;

	    static createFrom(source: any = {}) {
	        return new XlsxChartInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rel = source["rel"];
	        this.sheet = source["sheet"];
	        this.refs = source["refs"];
	        this.chartType = source["chartType"];
	        this.title = source["title"];
	    }
	}
	export class XlsxChartResult {
	    path: string;
	    name: string;
	    sheet: string;
	    anchor: string;
	    labels: number;
	    labelList: string[];
	    values: number[];
	    chartType: string;
	    title: string;

	    static createFrom(source: any = {}) {
	        return new XlsxChartResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.sheet = source["sheet"];
	        this.anchor = source["anchor"];
	        this.labels = source["labels"];
	        this.labelList = source["labelList"];
	        this.values = source["values"];
	        this.chartType = source["chartType"];
	        this.title = source["title"];
	    }
	}
	export class XlsxEditResult {
	    preview: string;
	    summary: string;
	    applied: number;

	    static createFrom(source: any = {}) {
	        return new XlsxEditResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preview = source["preview"];
	        this.summary = source["summary"];
	        this.applied = source["applied"];
	    }
	}
	export class XlsxPlanResult {
	    ops: string;
	    summary: string;
	    changes: xlsxedit.Change[];
	    total: number;
	    truncated: boolean;

	    static createFrom(source: any = {}) {
	        return new XlsxPlanResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ops = source["ops"];
	        this.summary = source["summary"];
	        this.changes = this.convertValues(source["changes"], xlsxedit.Change);
	        this.total = source["total"];
	        this.truncated = source["truncated"];
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
	export class ZipDeliverableResult {
	    path: string;
	    name: string;
	    entries: number;
	    bytes: number;

	    static createFrom(source: any = {}) {
	        return new ZipDeliverableResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.entries = source["entries"];
	        this.bytes = source["bytes"];
	    }
	}
	export class chapterArtEntry {
	    chapter: number;
	    asset_id?: string;
	    path: string;
	    created_at: string;

	    static createFrom(source: any = {}) {
	        return new chapterArtEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter = source["chapter"];
	        this.asset_id = source["asset_id"];
	        this.path = source["path"];
	        this.created_at = source["created_at"];
	    }
	}
	export class imageHubAssetView {
	    id: string;
	    kind: string;
	    path: string;
	    mime?: string;
	    space: string;
	    source_board: string;
	    capability: string;
	    backend?: string;
	    model?: string;
	    cost?: string;
	    created_at: string;
	    prompt_truncate?: string;
	    params?: Record<string, any>;
	    parent_id?: string;

	    static createFrom(source: any = {}) {
	        return new imageHubAssetView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.mime = source["mime"];
	        this.space = source["space"];
	        this.source_board = source["source_board"];
	        this.capability = source["capability"];
	        this.backend = source["backend"];
	        this.model = source["model"];
	        this.cost = source["cost"];
	        this.created_at = source["created_at"];
	        this.prompt_truncate = source["prompt_truncate"];
	        this.params = source["params"];
	        this.parent_id = source["parent_id"];
	    }
	}

}

export namespace assistant {

	export class Assistant {
	    id: string;
	    name: string;
	    personalityId: string;
	    wxToken: string;
	    wxBotId: string;
	    wxUserId: string;
	    enabled: boolean;
	    portraitUrl: string;
	    voiceGuide?: string;
	    gender?: string;
	    tags?: string[];
	    dims?: whisper.PersonalityDims;

	    static createFrom(source: any = {}) {
	        return new Assistant(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.personalityId = source["personalityId"];
	        this.wxToken = source["wxToken"];
	        this.wxBotId = source["wxBotId"];
	        this.wxUserId = source["wxUserId"];
	        this.enabled = source["enabled"];
	        this.portraitUrl = source["portraitUrl"];
	        this.voiceGuide = source["voiceGuide"];
	        this.gender = source["gender"];
	        this.tags = source["tags"];
	        this.dims = this.convertValues(source["dims"], whisper.PersonalityDims);
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

export namespace board {

	export class BreadcrumbSpec {
	    anchorTo?: string;

	    static createFrom(source: any = {}) {
	        return new BreadcrumbSpec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anchorTo = source["anchorTo"];
	    }
	}
	export class IntentDecl {
	    id: string;
	    handler: string;

	    static createFrom(source: any = {}) {
	        return new IntentDecl(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.handler = source["handler"];
	    }
	}
	export class NavChild {
	    id: string;
	    label: string;
	    page?: string;
	    icon?: string;

	    static createFrom(source: any = {}) {
	        return new NavChild(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.page = source["page"];
	        this.icon = source["icon"];
	    }
	}
	export class NavSpec {
	    children?: NavChild[];

	    static createFrom(source: any = {}) {
	        return new NavSpec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.children = this.convertValues(source["children"], NavChild);
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
	export class Manifest {
	    id: string;
	    label: string;
	    icon: string;
	    page: string;
	    lazy: boolean;
	    space?: string;
	    keepAlive?: boolean;
	    layout?: string;
	    shortcut?: string;
	    menuOrder?: number;
	    inMenu?: boolean;
	    breadcrumb?: BreadcrumbSpec;
	    isHome?: boolean;
	    nav?: NavSpec;
	    featureModel?: string;
	    bindings?: string[];
	    intents?: IntentDecl[];
	    tools?: string[];

	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.page = source["page"];
	        this.lazy = source["lazy"];
	        this.space = source["space"];
	        this.keepAlive = source["keepAlive"];
	        this.layout = source["layout"];
	        this.shortcut = source["shortcut"];
	        this.menuOrder = source["menuOrder"];
	        this.inMenu = source["inMenu"];
	        this.breadcrumb = this.convertValues(source["breadcrumb"], BreadcrumbSpec);
	        this.isHome = source["isHome"];
	        this.nav = this.convertValues(source["nav"], NavSpec);
	        this.featureModel = source["featureModel"];
	        this.bindings = source["bindings"];
	        this.intents = this.convertValues(source["intents"], IntentDecl);
	        this.tools = source["tools"];
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

export namespace bookimport {

	export class Warning {
	    code: string;
	    message: string;
	    level: string;

	    static createFrom(source: any = {}) {
	        return new Warning(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.level = source["level"];
	    }
	}

}

export namespace booksource {

	export class CrawlRule {
	    concurrency?: number;
	    minIntervalMs?: number;
	    maxIntervalMs?: number;
	    maxRetries?: number;
	    retryMinMs?: number;
	    retryMaxMs?: number;

	    static createFrom(source: any = {}) {
	        return new CrawlRule(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.concurrency = source["concurrency"];
	        this.minIntervalMs = source["minIntervalMs"];
	        this.maxIntervalMs = source["maxIntervalMs"];
	        this.maxRetries = source["maxRetries"];
	        this.retryMinMs = source["retryMinMs"];
	        this.retryMaxMs = source["retryMaxMs"];
	    }
	}
	export class SearchEngineRule {
	    name: string;
	    url: string;
	    queryFormat?: string;
	    result: string;
	    title: string;
	    link?: string;
	    disabled?: boolean;
	    comment?: string;
	    linkParam?: string;
	    redirectHosts?: string[];
	    crawl?: CrawlRule;

	    static createFrom(source: any = {}) {
	        return new SearchEngineRule(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.queryFormat = source["queryFormat"];
	        this.result = source["result"];
	        this.title = source["title"];
	        this.link = source["link"];
	        this.disabled = source["disabled"];
	        this.comment = source["comment"];
	        this.linkParam = source["linkParam"];
	        this.redirectHosts = source["redirectHosts"];
	        this.crawl = this.convertValues(source["crawl"], CrawlRule);
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
	export class TocEntry {
	    title: string;
	    url: string;
	    order: number;

	    static createFrom(source: any = {}) {
	        return new TocEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.order = source["order"];
	    }
	}

}

export namespace browser {

	export class ObserveView {
	    available: boolean;
	    url: string;
	    title: string;
	    image: string;
	    width: number;
	    height: number;
	    updatedAt: number;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new ObserveView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.image = source["image"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.updatedAt = source["updatedAt"];
	        this.error = source["error"];
	    }
	}

}

export namespace characterlib {

	export class Character {
	    id: string;
	    name: string;
	    kind: string;
	    gender?: string;
	    age?: string;
	    tags?: string[];
	    portraitUrl?: string;
	    referenceImages?: string[];
	    galleryImages?: string[];
	    referenceScores?: number[];
	    roleType?: string;
	    personality?: string;
	    background?: string;
	    appearance?: string;
	    figure?: string;
	    motivation?: string;
	    arc?: string;
	    status?: string;
	    notes?: string;
	    dialogueSamples?: string[];
	    chatEnabled: boolean;
	    dims: whisper.PersonalityDims;
	    voiceGuide?: string;
	    behaviorRules?: string;
	    emotionLogic?: string;
	    hiddenPersona?: whisper.PersonalityDims;
	    assistantId?: string;
	    createdAt: string;
	    updatedAt: string;
	    hidden: boolean;

	    static createFrom(source: any = {}) {
	        return new Character(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.gender = source["gender"];
	        this.age = source["age"];
	        this.tags = source["tags"];
	        this.portraitUrl = source["portraitUrl"];
	        this.referenceImages = source["referenceImages"];
	        this.galleryImages = source["galleryImages"];
	        this.referenceScores = source["referenceScores"];
	        this.roleType = source["roleType"];
	        this.personality = source["personality"];
	        this.background = source["background"];
	        this.appearance = source["appearance"];
	        this.figure = source["figure"];
	        this.motivation = source["motivation"];
	        this.arc = source["arc"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	        this.dialogueSamples = source["dialogueSamples"];
	        this.chatEnabled = source["chatEnabled"];
	        this.dims = this.convertValues(source["dims"], whisper.PersonalityDims);
	        this.voiceGuide = source["voiceGuide"];
	        this.behaviorRules = source["behaviorRules"];
	        this.emotionLogic = source["emotionLogic"];
	        this.hiddenPersona = this.convertValues(source["hiddenPersona"], whisper.PersonalityDims);
	        this.assistantId = source["assistantId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.hidden = source["hidden"];
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
	export class ImportFieldConflict {
	    characterId: string;
	    characterName: string;
	    field: string;
	    fieldLabel: string;
	    libraryValue: string;
	    projectValue: string;

	    static createFrom(source: any = {}) {
	        return new ImportFieldConflict(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.characterId = source["characterId"];
	        this.characterName = source["characterName"];
	        this.field = source["field"];
	        this.fieldLabel = source["fieldLabel"];
	        this.libraryValue = source["libraryValue"];
	        this.projectValue = source["projectValue"];
	    }
	}
	export class ImportPreview {
	    import: number;
	    fill: number;
	    conflicts: ImportFieldConflict[];

	    static createFrom(source: any = {}) {
	        return new ImportPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.import = source["import"];
	        this.fill = source["fill"];
	        this.conflicts = this.convertValues(source["conflicts"], ImportFieldConflict);
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
	export class ProjectCharacter {
	    projectId: string;
	    characterId: string;
	    roleInProject?: string;
	    arcState?: string;
	    status?: string;
	    joinedAt: string;

	    static createFrom(source: any = {}) {
	        return new ProjectCharacter(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.characterId = source["characterId"];
	        this.roleInProject = source["roleInProject"];
	        this.arcState = source["arcState"];
	        this.status = source["status"];
	        this.joinedAt = source["joinedAt"];
	    }
	}

}

export namespace chat {

	export class Message {
	    id: number;
	    topic_id: string;
	    role: string;
	    content: string;
	    extra?: string;
	    seq: number;
	    created_at: string;

	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.topic_id = source["topic_id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.extra = source["extra"];
	        this.seq = source["seq"];
	        this.created_at = source["created_at"];
	    }
	}
	export class Topic {
	    id: string;
	    title: string;
	    mode: string;
	    created_at: string;
	    updated_at: string;
	    preview?: string;

	    static createFrom(source: any = {}) {
	        return new Topic(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.mode = source["mode"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.preview = source["preview"];
	    }
	}

}

export namespace contextview {

	export class CatDelta {
	    cat: string;
	    items?: number;
	    tokens: number;

	    static createFrom(source: any = {}) {
	        return new CatDelta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cat = source["cat"];
	        this.items = source["items"];
	        this.tokens = source["tokens"];
	    }
	}
	export class Category {
	    system: number;
	    tools: number;
	    user: number;
	    inject: number;
	    assistant: number;
	    tool: number;

	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = source["system"];
	        this.tools = source["tools"];
	        this.user = source["user"];
	        this.inject = source["inject"];
	        this.assistant = source["assistant"];
	        this.tool = source["tool"];
	    }
	}
	export class CompactTiers {
	    soft: number;
	    ratio: number;
	    force: number;

	    static createFrom(source: any = {}) {
	        return new CompactTiers(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.soft = source["soft"];
	        this.ratio = source["ratio"];
	        this.force = source["force"];
	    }
	}
	export class ContextEvent {
	    kind: string;
	    seq: number;
	    delta: number;
	    source?: string;
	    turn: number;
	    step: number;
	    ts: number;

	    static createFrom(source: any = {}) {
	        return new ContextEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.seq = source["seq"];
	        this.delta = source["delta"];
	        this.source = source["source"];
	        this.turn = source["turn"];
	        this.step = source["step"];
	        this.ts = source["ts"];
	    }
	}
	export class CostRate {
	    inputPer1M?: number;
	    outputPer1M?: number;
	    cacheHitPer1M?: number;
	    currency?: string;

	    static createFrom(source: any = {}) {
	        return new CostRate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPer1M = source["inputPer1M"];
	        this.outputPer1M = source["outputPer1M"];
	        this.cacheHitPer1M = source["cacheHitPer1M"];
	        this.currency = source["currency"];
	    }
	}
	export class ToolTiming {
	    name: string;
	    calls: number;
	    ms: number;

	    static createFrom(source: any = {}) {
	        return new ToolTiming(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.calls = source["calls"];
	        this.ms = source["ms"];
	    }
	}
	export class ContextTiming {
	    wallMs?: number;
	    ttftMs?: number;
	    genMs?: number;
	    calls?: number;
	    toolsMs?: number;
	    toolCalls?: number;
	    tools?: ToolTiming[];

	    static createFrom(source: any = {}) {
	        return new ContextTiming(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wallMs = source["wallMs"];
	        this.ttftMs = source["ttftMs"];
	        this.genMs = source["genMs"];
	        this.calls = source["calls"];
	        this.toolsMs = source["toolsMs"];
	        this.toolCalls = source["toolCalls"];
	        this.tools = this.convertValues(source["tools"], ToolTiming);
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
	export class FileActivity {
	    seq: number;
	    ts: number;
	    turn: number;
	    step: number;
	    tool: string;
	    action: string;
	    path: string;
	    added?: number;
	    removed?: number;
	    hits?: number;

	    static createFrom(source: any = {}) {
	        return new FileActivity(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.ts = source["ts"];
	        this.turn = source["turn"];
	        this.step = source["step"];
	        this.tool = source["tool"];
	        this.action = source["action"];
	        this.path = source["path"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.hits = source["hits"];
	    }
	}
	export class SurfaceNode {
	    seq: number;
	    cat: string;
	    tokens: number;
	    text?: string;
	    gone?: number;
	    tool?: string;
	    err?: boolean;

	    static createFrom(source: any = {}) {
	        return new SurfaceNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.cat = source["cat"];
	        this.tokens = source["tokens"];
	        this.text = source["text"];
	        this.gone = source["gone"];
	        this.tool = source["tool"];
	        this.err = source["err"];
	    }
	}
	export class PrefixVerdict {
	    stable: boolean;
	    prefixBytes?: number;
	    appendCount?: number;
	    rewriteCount?: number;
	    hash?: string;
	    prevHash?: string;

	    static createFrom(source: any = {}) {
	        return new PrefixVerdict(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stable = source["stable"];
	        this.prefixBytes = source["prefixBytes"];
	        this.appendCount = source["appendCount"];
	        this.rewriteCount = source["rewriteCount"];
	        this.hash = source["hash"];
	        this.prevHash = source["prevHash"];
	    }
	}
	export class RequestDelta {
	    items: number;
	    tokens: number;
	    byCat?: CatDelta[];
	    approx?: boolean;
	    first?: boolean;

	    static createFrom(source: any = {}) {
	        return new RequestDelta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = source["items"];
	        this.tokens = source["tokens"];
	        this.byCat = this.convertValues(source["byCat"], CatDelta);
	        this.approx = source["approx"];
	        this.first = source["first"];
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
	export class RequestRecord {
	    seq: number;
	    ts: number;
	    turn: number;
	    step: number;
	    category: Category;
	    briefUser?: string;
	    briefIn?: string[];
	    briefResp?: string;
	    briefUserSeq?: number;
	    briefRespSeq?: number;
	    promptTokens?: number;
	    outputTokens?: number;
	    cacheHitTokens?: number;
	    cacheMissTokens?: number;
	    estimated?: boolean;
	    delta?: RequestDelta;
	    prefix?: PrefixVerdict;

	    static createFrom(source: any = {}) {
	        return new RequestRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.ts = source["ts"];
	        this.turn = source["turn"];
	        this.step = source["step"];
	        this.category = this.convertValues(source["category"], Category);
	        this.briefUser = source["briefUser"];
	        this.briefIn = source["briefIn"];
	        this.briefResp = source["briefResp"];
	        this.briefUserSeq = source["briefUserSeq"];
	        this.briefRespSeq = source["briefRespSeq"];
	        this.promptTokens = source["promptTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.cacheHitTokens = source["cacheHitTokens"];
	        this.cacheMissTokens = source["cacheMissTokens"];
	        this.estimated = source["estimated"];
	        this.delta = this.convertValues(source["delta"], RequestDelta);
	        this.prefix = this.convertValues(source["prefix"], PrefixVerdict);
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
	export class Stats {
	    turns: number;
	    steps: number;
	    injects: number;
	    compacts: number;
	    prunes: number;
	    toolCalls: number;
	    images: number;
	    cacheHitPercent?: number;
	    costEstimate?: number;

	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turns = source["turns"];
	        this.steps = source["steps"];
	        this.injects = source["injects"];
	        this.compacts = source["compacts"];
	        this.prunes = source["prunes"];
	        this.toolCalls = source["toolCalls"];
	        this.images = source["images"];
	        this.cacheHitPercent = source["cacheHitPercent"];
	        this.costEstimate = source["costEstimate"];
	    }
	}
	export class ContextTimeline {
	    ok: boolean;
	    window: number;
	    current: Category;
	    stats: Stats;
	    requests: RequestRecord[];
	    events: ContextEvent[];
	    nodes: SurfaceNode[];
	    archive: SurfaceNode[];
	    files: FileActivity[];
	    timing?: ContextTiming;
	    rate?: CostRate;
	    compact?: CompactTiers;

	    static createFrom(source: any = {}) {
	        return new ContextTimeline(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.window = source["window"];
	        this.current = this.convertValues(source["current"], Category);
	        this.stats = this.convertValues(source["stats"], Stats);
	        this.requests = this.convertValues(source["requests"], RequestRecord);
	        this.events = this.convertValues(source["events"], ContextEvent);
	        this.nodes = this.convertValues(source["nodes"], SurfaceNode);
	        this.archive = this.convertValues(source["archive"], SurfaceNode);
	        this.files = this.convertValues(source["files"], FileActivity);
	        this.timing = this.convertValues(source["timing"], ContextTiming);
	        this.rate = this.convertValues(source["rate"], CostRate);
	        this.compact = this.convertValues(source["compact"], CompactTiers);
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



	export class NodeImage {
	    ref: string;
	    path: string;
	    refCwd?: string;
	    exists?: boolean;
	    width?: number;
	    height?: number;
	    scaledW?: number;
	    scaledH?: number;
	    stdTokens?: number;
	    highTokens?: number;

	    static createFrom(source: any = {}) {
	        return new NodeImage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.path = source["path"];
	        this.refCwd = source["refCwd"];
	        this.exists = source["exists"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.scaledW = source["scaledW"];
	        this.scaledH = source["scaledH"];
	        this.stdTokens = source["stdTokens"];
	        this.highTokens = source["highTokens"];
	    }
	}
	export class NodeDetail {
	    seq: number;
	    kind: string;
	    ts?: number;
	    tool?: string;
	    args?: string;
	    output?: string;
	    err?: string;
	    truncated?: boolean;
	    text?: string;
	    imageRefs?: string[];
	    images?: NodeImage[];
	    lines?: number;
	    clamped?: boolean;

	    static createFrom(source: any = {}) {
	        return new NodeDetail(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.kind = source["kind"];
	        this.ts = source["ts"];
	        this.tool = source["tool"];
	        this.args = source["args"];
	        this.output = source["output"];
	        this.err = source["err"];
	        this.truncated = source["truncated"];
	        this.text = source["text"];
	        this.imageRefs = source["imageRefs"];
	        this.images = this.convertValues(source["images"], NodeImage);
	        this.lines = source["lines"];
	        this.clamped = source["clamped"];
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

export namespace cost {

	export class BandSource {
	    name: string;
	    title: string;
	    category: string;
	    unit: string;
	    spec: string;
	    source: string;
	    region: string;
	    priceDate: string;
	    priceType: string;
	    price: number;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new BandSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.category = source["category"];
	        this.unit = source["unit"];
	        this.spec = source["spec"];
	        this.source = source["source"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.priceType = source["priceType"];
	        this.price = source["price"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ComposeCheck {
	    level: string;
	    row: number;
	    msg: string;

	    static createFrom(source: any = {}) {
	        return new ComposeCheck(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.row = source["row"];
	        this.msg = source["msg"];
	    }
	}
	export class PriceBand {
	    samples: number;
	    min: number;
	    max: number;
	    mean: number;
	    median: number;
	    p25: number;
	    p75: number;
	    spreadPct: number;
	    outliers: number;
	    confidence: string;
	    sources: BandSource[];

	    static createFrom(source: any = {}) {
	        return new PriceBand(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.samples = source["samples"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.mean = source["mean"];
	        this.median = source["median"];
	        this.p25 = source["p25"];
	        this.p75 = source["p75"];
	        this.spreadPct = source["spreadPct"];
	        this.outliers = source["outliers"];
	        this.confidence = source["confidence"];
	        this.sources = this.convertValues(source["sources"], BandSource);
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

export namespace costinquiry {

	export class AdjustSuggestion {
	    entryName: string;
	    entryTitle: string;
	    entryPrice: number;
	    latestPrice: number;
	    latestDate: string;
	    latestSource: string;
	    diff: number;
	    diffPct: number;
	    unit: string;
	    level: string;
	    predictedNext?: number;
	    predictionNote?: string;

	    static createFrom(source: any = {}) {
	        return new AdjustSuggestion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entryName = source["entryName"];
	        this.entryTitle = source["entryTitle"];
	        this.entryPrice = source["entryPrice"];
	        this.latestPrice = source["latestPrice"];
	        this.latestDate = source["latestDate"];
	        this.latestSource = source["latestSource"];
	        this.diff = source["diff"];
	        this.diffPct = source["diffPct"];
	        this.unit = source["unit"];
	        this.level = source["level"];
	        this.predictedNext = source["predictedNext"];
	        this.predictionNote = source["predictionNote"];
	    }
	}
	export class Record {
	    id: number;
	    title: string;
	    spec: string;
	    unit: string;
	    price: number;
	    source: string;
	    supplier: string;
	    region: string;
	    priceDate: string;
	    validUntil: string;
	    note: string;
	    status: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.source = source["source"];
	        this.supplier = source["supplier"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.validUntil = source["validUntil"];
	        this.note = source["note"];
	        this.status = source["status"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ScanFinding {
	    kind: string;
	    severity: string;
	    title: string;
	    detail: string;
	    refIds: number[];

	    static createFrom(source: any = {}) {
	        return new ScanFinding(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.severity = source["severity"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.refIds = source["refIds"];
	    }
	}

}

export namespace costproject {

	export class ProjectSummary {
	    id: string;
	    name: string;
	    projectType: string;
	    scale: string;
	    craft: string;
	    status: string;
	    note: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    itemCount: number;
	    total: number;
	    versionCount: number;

	    static createFrom(source: any = {}) {
	        return new ProjectSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.projectType = source["projectType"];
	        this.scale = source["scale"];
	        this.craft = source["craft"];
	        this.status = source["status"];
	        this.note = source["note"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.itemCount = source["itemCount"];
	        this.total = source["total"];
	        this.versionCount = source["versionCount"];
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

export namespace costref {

	export class Indicator {
	    key: string;
	    unit: string;
	    samples: number;
	    min: number;
	    max: number;
	    mean: number;
	    median: number;
	    p25: number;
	    p75: number;

	    static createFrom(source: any = {}) {
	        return new Indicator(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.unit = source["unit"];
	        this.samples = source["samples"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.mean = source["mean"];
	        this.median = source["median"];
	        this.p25 = source["p25"];
	        this.p75 = source["p75"];
	    }
	}

}

export namespace dag {

	export class Node {
	    id: string;
	    title: string;
	    prompt: string;
	    dependsOn?: string[];
	    status: string;
	    ref?: string;
	    outputs?: string[];
	    error?: string;
	    runCount: number;
	    steerCount?: number;
	    acceptedAt?: string;
	    risk?: string;
	    approved?: boolean;

	    static createFrom(source: any = {}) {
	        return new Node(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.prompt = source["prompt"];
	        this.dependsOn = source["dependsOn"];
	        this.status = source["status"];
	        this.ref = source["ref"];
	        this.outputs = source["outputs"];
	        this.error = source["error"];
	        this.runCount = source["runCount"];
	        this.steerCount = source["steerCount"];
	        this.acceptedAt = source["acceptedAt"];
	        this.risk = source["risk"];
	        this.approved = source["approved"];
	    }
	}
	export class RunView {
	    id: string;
	    goal: string;
	    createdAt: string;
	    updatedAt: string;
	    nodes: Node[];
	    derived: string;

	    static createFrom(source: any = {}) {
	        return new RunView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.goal = source["goal"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.nodes = this.convertValues(source["nodes"], Node);
	        this.derived = source["derived"];
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
	export class Template {
	    id: string;
	    name: string;
	    goal: string;
	    nodes: Node[];
	    createdAt: string;
	    sourceRunId?: string;

	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.goal = source["goal"];
	        this.nodes = this.convertValues(source["nodes"], Node);
	        this.createdAt = source["createdAt"];
	        this.sourceRunId = source["sourceRunId"];
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

export namespace event {

	export class AskAnswer {
	    questionId: string;
	    selected: string[];

	    static createFrom(source: any = {}) {
	        return new AskAnswer(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.questionId = source["questionId"];
	        this.selected = source["selected"];
	    }
	}

}

export namespace evidence {

	export class ChangeRecord {
	    id: string;
	    sessionId: string;
	    space: string;
	    turn: number;
	    tool: string;
	    target: string;
	    beforeSummary: string;
	    afterSummary: string;
	    baselinePath?: string;
	    opsJson?: string;
	    model?: string;
	    at: number;
	    status: string;

	    static createFrom(source: any = {}) {
	        return new ChangeRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.space = source["space"];
	        this.turn = source["turn"];
	        this.tool = source["tool"];
	        this.target = source["target"];
	        this.beforeSummary = source["beforeSummary"];
	        this.afterSummary = source["afterSummary"];
	        this.baselinePath = source["baselinePath"];
	        this.opsJson = source["opsJson"];
	        this.model = source["model"];
	        this.at = source["at"];
	        this.status = source["status"];
	    }
	}
	export class Verdict {
	    id: string;
	    status: string;
	    channelA?: string;
	    channelB?: string;
	    note?: string;
	    at: number;
	    channelBRatio?: number;
	    channelBPages?: number;
	    channelBArtifacts?: string;

	    static createFrom(source: any = {}) {
	        return new Verdict(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.channelA = source["channelA"];
	        this.channelB = source["channelB"];
	        this.note = source["note"];
	        this.at = source["at"];
	        this.channelBRatio = source["channelBRatio"];
	        this.channelBPages = source["channelBPages"];
	        this.channelBArtifacts = source["channelBArtifacts"];
	    }
	}

}

export namespace herdsman {

	export class LanExposure {
	    config_path: string;
	    exposed: boolean;
	    port: number;
	    config_missing: boolean;
	    parse_error?: string;
	    guidance?: string;

	    static createFrom(source: any = {}) {
	        return new LanExposure(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config_path = source["config_path"];
	        this.exposed = source["exposed"];
	        this.port = source["port"];
	        this.config_missing = source["config_missing"];
	        this.parse_error = source["parse_error"];
	        this.guidance = source["guidance"];
	    }
	}

}

export namespace modelengine {

	export class EngineStatus {
	    id: string;
	    connected: boolean;
	    model_count: number;
	    error?: string;
	    last_checked?: string;
	    latency_ms?: number;

	    static createFrom(source: any = {}) {
	        return new EngineStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.connected = source["connected"];
	        this.model_count = source["model_count"];
	        this.error = source["error"];
	        this.last_checked = source["last_checked"];
	        this.latency_ms = source["latency_ms"];
	    }
	}
	export class ModelInfo {
	    id: string;
	    owned_by: string;
	    status?: string;
	    kind?: string;
	    name?: string;
	    alias_of?: string;
	    context_length?: number;
	    max_output?: number;
	    price_in?: number;
	    price_out?: number;
	    currency?: string;
	    unit?: string;
	    free?: boolean;
	    caps?: string[];
	    price_note?: string;
	    points_in?: number;
	    points_cached?: number;
	    points_out?: number;
	    points_peak?: number;

	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.owned_by = source["owned_by"];
	        this.status = source["status"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.alias_of = source["alias_of"];
	        this.context_length = source["context_length"];
	        this.max_output = source["max_output"];
	        this.price_in = source["price_in"];
	        this.price_out = source["price_out"];
	        this.currency = source["currency"];
	        this.unit = source["unit"];
	        this.free = source["free"];
	        this.caps = source["caps"];
	        this.price_note = source["price_note"];
	        this.points_in = source["points_in"];
	        this.points_cached = source["points_cached"];
	        this.points_out = source["points_out"];
	        this.points_peak = source["points_peak"];
	    }
	}
	export class EngineConfig {
	    id: string;
	    name: string;
	    type: string;
	    label?: string;
	    color?: string;
	    icon?: string;
	    is_local?: boolean;
	    base_url: string;
	    api_key?: string;
	    enabled: boolean;
	    default_model: string;
	    models?: ModelInfo[];
	    status?: EngineStatus;
	    user_price_in?: number;
	    user_price_out?: number;

	    static createFrom(source: any = {}) {
	        return new EngineConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.label = source["label"];
	        this.color = source["color"];
	        this.icon = source["icon"];
	        this.is_local = source["is_local"];
	        this.base_url = source["base_url"];
	        this.api_key = source["api_key"];
	        this.enabled = source["enabled"];
	        this.default_model = source["default_model"];
	        this.models = this.convertValues(source["models"], ModelInfo);
	        this.status = this.convertValues(source["status"], EngineStatus);
	        this.user_price_in = source["user_price_in"];
	        this.user_price_out = source["user_price_out"];
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

	export class EngineSubtotal {
	    tokens: number;
	    calls: number;
	    estimated_cost_cny: number;

	    static createFrom(source: any = {}) {
	        return new EngineSubtotal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tokens = source["tokens"];
	        this.calls = source["calls"];
	        this.estimated_cost_cny = source["estimated_cost_cny"];
	    }
	}
	export class FeatureUsageSummary {
	    feature: string;
	    engine_id: string;
	    model: string;
	    calls: number;
	    tokens_in: number;
	    tokens_out: number;
	    cost_cny: number;
	    avg_ms: number;
	    success_rate: number;

	    static createFrom(source: any = {}) {
	        return new FeatureUsageSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feature = source["feature"];
	        this.engine_id = source["engine_id"];
	        this.model = source["model"];
	        this.calls = source["calls"];
	        this.tokens_in = source["tokens_in"];
	        this.tokens_out = source["tokens_out"];
	        this.cost_cny = source["cost_cny"];
	        this.avg_ms = source["avg_ms"];
	        this.success_rate = source["success_rate"];
	    }
	}

	export class ModelUsageStats {
	    feature?: string;
	    engine_id: string;
	    model: string;
	    call_count: number;
	    success_count: number;
	    fail_count: number;
	    input_tokens: number;
	    output_tokens: number;
	    total_tokens: number;
	    cache_hit_tokens?: number;
	    cache_miss_tokens?: number;
	    total_duration_ms: number;
	    estimated_cost?: number;
	    currency?: string;
	    billing_mode?: string;
	    last_error?: string;
	    last_called_at?: string;

	    static createFrom(source: any = {}) {
	        return new ModelUsageStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feature = source["feature"];
	        this.engine_id = source["engine_id"];
	        this.model = source["model"];
	        this.call_count = source["call_count"];
	        this.success_count = source["success_count"];
	        this.fail_count = source["fail_count"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.cache_hit_tokens = source["cache_hit_tokens"];
	        this.cache_miss_tokens = source["cache_miss_tokens"];
	        this.total_duration_ms = source["total_duration_ms"];
	        this.estimated_cost = source["estimated_cost"];
	        this.currency = source["currency"];
	        this.billing_mode = source["billing_mode"];
	        this.last_error = source["last_error"];
	        this.last_called_at = source["last_called_at"];
	    }
	}
	export class TrendPoint {
	    time: string;
	    calls: number;
	    success_calls: number;
	    fail_calls: number;
	    input_tokens: number;
	    output_tokens: number;
	    total_tokens: number;
	    cost: number;

	    static createFrom(source: any = {}) {
	        return new TrendPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.calls = source["calls"];
	        this.success_calls = source["success_calls"];
	        this.fail_calls = source["fail_calls"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.cost = source["cost"];
	    }
	}
	export class ModelStatsSummary {
	    total_calls: number;
	    success_calls: number;
	    fail_calls: number;
	    total_tokens: number;
	    input_tokens: number;
	    output_tokens: number;
	    cache_hit_tokens?: number;
	    cache_miss_tokens?: number;
	    total_duration_ms: number;
	    avg_duration_ms: number;
	    total_cost: number;
	    trend: TrendPoint[];
	    per_model: ModelUsageStats[];
	    engines?: Record<string, EngineSubtotal>;
	    since?: string;
	    usd_to_cny: number;
	    catalog_version?: string;
	    catalog_source?: string;
	    per_feature?: FeatureUsageSummary[];

	    static createFrom(source: any = {}) {
	        return new ModelStatsSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_calls = source["total_calls"];
	        this.success_calls = source["success_calls"];
	        this.fail_calls = source["fail_calls"];
	        this.total_tokens = source["total_tokens"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.cache_hit_tokens = source["cache_hit_tokens"];
	        this.cache_miss_tokens = source["cache_miss_tokens"];
	        this.total_duration_ms = source["total_duration_ms"];
	        this.avg_duration_ms = source["avg_duration_ms"];
	        this.total_cost = source["total_cost"];
	        this.trend = this.convertValues(source["trend"], TrendPoint);
	        this.per_model = this.convertValues(source["per_model"], ModelUsageStats);
	        this.engines = this.convertValues(source["engines"], EngineSubtotal, true);
	        this.since = source["since"];
	        this.usd_to_cny = source["usd_to_cny"];
	        this.catalog_version = source["catalog_version"];
	        this.catalog_source = source["catalog_source"];
	        this.per_feature = this.convertValues(source["per_feature"], FeatureUsageSummary);
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

export namespace pantheon {

	export class AgentStatus {
	    id: string;
	    name: string;
	    kind: string;
	    available: boolean;
	    executable?: string;
	    version?: string;
	    reason?: string;
	    capabilities?: string[];

	    static createFrom(source: any = {}) {
	        return new AgentStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.available = source["available"];
	        this.executable = source["executable"];
	        this.version = source["version"];
	        this.reason = source["reason"];
	        this.capabilities = source["capabilities"];
	    }
	}
	export class Attachment {
	    name: string;
	    path: string;
	    mime: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.mime = source["mime"];
	        this.size = source["size"];
	    }
	}
	export class Contribution {
	    id: string;
	    phase: string;
	    round?: number;
	    agentId: string;
	    agentName: string;
	    anonLabel?: string;
	    content: string;
	    confidence?: string;
	    consensus?: string;
	    runMode?: string;
	    createdAt: number;
	    durationMs?: number;

	    static createFrom(source: any = {}) {
	        return new Contribution(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.phase = source["phase"];
	        this.round = source["round"];
	        this.agentId = source["agentId"];
	        this.agentName = source["agentName"];
	        this.anonLabel = source["anonLabel"];
	        this.content = source["content"];
	        this.confidence = source["confidence"];
	        this.consensus = source["consensus"];
	        this.runMode = source["runMode"];
	        this.createdAt = source["createdAt"];
	        this.durationMs = source["durationMs"];
	    }
	}
	export class Event {
	    seq: number;
	    sessionId: string;
	    phase: string;
	    round?: number;
	    agentId?: string;
	    kind: string;
	    text?: string;
	    toolId?: string;
	    toolName?: string;
	    toolStatus?: string;
	    toolInput?: string;
	    toolOutput?: string;
	    at: number;
	    durationMs?: number;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.sessionId = source["sessionId"];
	        this.phase = source["phase"];
	        this.round = source["round"];
	        this.agentId = source["agentId"];
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.toolId = source["toolId"];
	        this.toolName = source["toolName"];
	        this.toolStatus = source["toolStatus"];
	        this.toolInput = source["toolInput"];
	        this.toolOutput = source["toolOutput"];
	        this.at = source["at"];
	        this.durationMs = source["durationMs"];
	        this.error = source["error"];
	    }
	}
	export class EvidenceItem {
	    name: string;
	    content: string;

	    static createFrom(source: any = {}) {
	        return new EvidenceItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.content = source["content"];
	    }
	}
	export class InputDigest {
	    kind: string;
	    name: string;
	    path?: string;
	    sha256?: string;
	    size?: number;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new InputDigest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.sha256 = source["sha256"];
	        this.size = source["size"];
	        this.error = source["error"];
	    }
	}
	export class Participant {
	    agentId: string;
	    name: string;
	    role: string;
	    status: string;
	    startedAt?: number;
	    finishedAt?: number;
	    durationMs?: number;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new Participant(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agentId = source["agentId"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.durationMs = source["durationMs"];
	        this.error = source["error"];
	    }
	}
	export class SignOff {
	    decision: string;
	    note?: string;
	    at: number;
	    verdictSha256?: string;

	    static createFrom(source: any = {}) {
	        return new SignOff(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.decision = source["decision"];
	        this.note = source["note"];
	        this.at = source["at"];
	        this.verdictSha256 = source["verdictSha256"];
	    }
	}
	export class Verdict {
	    summary: string;
	    recommendation: string;
	    conditions?: string[];
	    disagreements?: string[];
	    risks?: string[];
	    evidenceNotes?: string[];
	    chairman: string;
	    partial: boolean;
	    createdAt: number;

	    static createFrom(source: any = {}) {
	        return new Verdict(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.recommendation = source["recommendation"];
	        this.conditions = source["conditions"];
	        this.disagreements = source["disagreements"];
	        this.risks = source["risks"];
	        this.evidenceNotes = source["evidenceNotes"];
	        this.chairman = source["chairman"];
	        this.partial = source["partial"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class StartRequest {
	    title: string;
	    topic: string;
	    context: string;
	    successCriteria: string;
	    workspace?: string;
	    evidence?: EvidenceItem[];
	    attachments?: Attachment[];
	    agentIds: string[];
	    chairmanAgentId: string;
	    rounds: number;
	    untilConsensus?: boolean;
	    maxRounds?: number;
	    readOnly: boolean;
	    protocol?: string;
	    parentSessionId?: string;

	    static createFrom(source: any = {}) {
	        return new StartRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.topic = source["topic"];
	        this.context = source["context"];
	        this.successCriteria = source["successCriteria"];
	        this.workspace = source["workspace"];
	        this.evidence = this.convertValues(source["evidence"], EvidenceItem);
	        this.attachments = this.convertValues(source["attachments"], Attachment);
	        this.agentIds = source["agentIds"];
	        this.chairmanAgentId = source["chairmanAgentId"];
	        this.rounds = source["rounds"];
	        this.untilConsensus = source["untilConsensus"];
	        this.maxRounds = source["maxRounds"];
	        this.readOnly = source["readOnly"];
	        this.protocol = source["protocol"];
	        this.parentSessionId = source["parentSessionId"];
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
	export class Session {
	    id: string;
	    title: string;
	    status: string;
	    phase: string;
	    currentRound?: number;
	    createdAt: number;
	    updatedAt: number;
	    request: StartRequest;
	    participants: Participant[];
	    events: Event[];
	    contributions: Contribution[];
	    verdict?: Verdict;
	    signoff?: SignOff;
	    resultPath?: string;
	    inputManifest?: InputDigest[];
	    warnings?: string[];
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.phase = source["phase"];
	        this.currentRound = source["currentRound"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.request = this.convertValues(source["request"], StartRequest);
	        this.participants = this.convertValues(source["participants"], Participant);
	        this.events = this.convertValues(source["events"], Event);
	        this.contributions = this.convertValues(source["contributions"], Contribution);
	        this.verdict = this.convertValues(source["verdict"], Verdict);
	        this.signoff = this.convertValues(source["signoff"], SignOff);
	        this.resultPath = source["resultPath"];
	        this.inputManifest = this.convertValues(source["inputManifest"], InputDigest);
	        this.warnings = source["warnings"];
	        this.error = source["error"];
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
	export class SessionSummary {
	    id: string;
	    title: string;
	    workspace?: string;
	    status: string;
	    phase: string;
	    createdAt: number;
	    updatedAt: number;
	    agentCount: number;
	    verdictSummary?: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new SessionSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.workspace = source["workspace"];
	        this.status = source["status"];
	        this.phase = source["phase"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.agentCount = source["agentCount"];
	        this.verdictSummary = source["verdictSummary"];
	        this.error = source["error"];
	    }
	}



	export class WorkspaceOption {
	    path: string;
	    name: string;
	    current: boolean;
	    recent?: boolean;

	    static createFrom(source: any = {}) {
	        return new WorkspaceOption(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.current = source["current"];
	        this.recent = source["recent"];
	    }
	}

}

export namespace pptxedit {

	export class SlideText {
	    index: number;
	    paragraphs: string[];

	    static createFrom(source: any = {}) {
	        return new SlideText(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.paragraphs = source["paragraphs"];
	    }
	}

}

export namespace pricefeed {

	export class Candidate {
	    title: string;
	    spec: string;
	    unit: string;
	    price: number;
	    tax: string;
	    existingName: string;
	    existingPrice: number;
	    status: string;
	    diff: number;
	    diffPct: number;
	    anomaly: boolean;
	    anomalyReason: string;
	    resourceCode?: string;
	    resourceTitle?: string;

	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.tax = source["tax"];
	        this.existingName = source["existingName"];
	        this.existingPrice = source["existingPrice"];
	        this.status = source["status"];
	        this.diff = source["diff"];
	        this.diffPct = source["diffPct"];
	        this.anomaly = source["anomaly"];
	        this.anomalyReason = source["anomalyReason"];
	        this.resourceCode = source["resourceCode"];
	        this.resourceTitle = source["resourceTitle"];
	    }
	}
	export class FetchRecord {
	    id: string;
	    sourceId: string;
	    sourceName: string;
	    url: string;
	    period: string;
	    fetchedAt: string;
	    status: string;
	    candidates: Candidate[];

	    static createFrom(source: any = {}) {
	        return new FetchRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceId = source["sourceId"];
	        this.sourceName = source["sourceName"];
	        this.url = source["url"];
	        this.period = source["period"];
	        this.fetchedAt = source["fetchedAt"];
	        this.status = source["status"];
	        this.candidates = this.convertValues(source["candidates"], Candidate);
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
	export class History {
	    name: string;
	    title: string;
	    unit: string;
	    price: number;
	    source: string;
	    period: string;
	    region?: string;
	    priceType?: string;
	    fetchedAt: string;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new History(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.source = source["source"];
	        this.period = source["period"];
	        this.region = source["region"];
	        this.priceType = source["priceType"];
	        this.fetchedAt = source["fetchedAt"];
	        this.note = source["note"];
	    }
	}
	export class Source {
	    id: string;
	    name: string;
	    url: string;
	    parser: string;
	    frequencyHours: number;
	    area: string;
	    headers?: Record<string, string>;
	    enabled: boolean;
	    lastFetchAt: string;
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.parser = source["parser"];
	        this.frequencyHours = source["frequencyHours"];
	        this.area = source["area"];
	        this.headers = source["headers"];
	        this.enabled = source["enabled"];
	        this.lastFetchAt = source["lastFetchAt"];
	        this.createdAt = source["createdAt"];
	    }
	}

}

export namespace prompt {

	export class ConstraintDef {
	    must: string[];
	    forbidden: string[];
	    style?: string[];

	    static createFrom(source: any = {}) {
	        return new ConstraintDef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.must = source["must"];
	        this.forbidden = source["forbidden"];
	        this.style = source["style"];
	    }
	}
	export class InputDef {
	    priority: string;
	    label: string;
	    order?: number;

	    static createFrom(source: any = {}) {
	        return new InputDef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.priority = source["priority"];
	        this.label = source["label"];
	        this.order = source["order"];
	    }
	}
	export class OutputDef {
	    format: string;
	    description: string;

	    static createFrom(source: any = {}) {
	        return new OutputDef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.description = source["description"];
	    }
	}
	export class Template {
	    name: string;
	    system: string;
	    task: string;
	    input_sections: Record<string, InputDef>;
	    output: OutputDef;
	    constraints: ConstraintDef;
	    version?: string;
	    category?: string;
	    description?: string;
	    parameters?: string[];

	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.system = source["system"];
	        this.task = source["task"];
	        this.input_sections = this.convertValues(source["input_sections"], InputDef, true);
	        this.output = this.convertValues(source["output"], OutputDef);
	        this.constraints = this.convertValues(source["constraints"], ConstraintDef);
	        this.version = source["version"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.parameters = source["parameters"];
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

export namespace promptstore {

	export class BundleImportOutcome {
	    key: string;
	    action: string;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new BundleImportOutcome(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.action = source["action"];
	        this.reason = source["reason"];
	    }
	}
	export class BundleImportStats {
	    total: number;
	    keptSystemDefault: number;
	    convertedToCustom: number;
	    createdOrUpdate: number;
	    skippedInvalid: number;
	    skippedUnknown: number;
	    skippedDuplicate: number;

	    static createFrom(source: any = {}) {
	        return new BundleImportStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.keptSystemDefault = source["keptSystemDefault"];
	        this.convertedToCustom = source["convertedToCustom"];
	        this.createdOrUpdate = source["createdOrUpdate"];
	        this.skippedInvalid = source["skippedInvalid"];
	        this.skippedUnknown = source["skippedUnknown"];
	        this.skippedDuplicate = source["skippedDuplicate"];
	    }
	}
	export class BundleImportResult {
	    applied: boolean;
	    statistics: BundleImportStats;
	    outcomes: BundleImportOutcome[];

	    static createFrom(source: any = {}) {
	        return new BundleImportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.statistics = this.convertValues(source["statistics"], BundleImportStats);
	        this.outcomes = this.convertValues(source["outcomes"], BundleImportOutcome);
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

	export class Issue {
	    code: string;
	    severity: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.severity = source["severity"];
	        this.message = source["message"];
	    }
	}

}

export namespace provider {

	export class ToolCall {
	    id: string;
	    name: string;
	    arguments: string;

	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	    }
	}

}

export namespace routesuggest {

	export class Candidate {
	    engine_id: string;
	    model: string;
	    is_local: boolean;
	    success_rate?: number;
	    avg_ms: number;
	    cost_cny: number;
	    samples: number;
	    ttft_ms?: number;
	    tps?: number;

	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine_id = source["engine_id"];
	        this.model = source["model"];
	        this.is_local = source["is_local"];
	        this.success_rate = source["success_rate"];
	        this.avg_ms = source["avg_ms"];
	        this.cost_cny = source["cost_cny"];
	        this.samples = source["samples"];
	        this.ttft_ms = source["ttft_ms"];
	        this.tps = source["tps"];
	    }
	}
	export class Endpoint {
	    engine_id: string;
	    model: string;

	    static createFrom(source: any = {}) {
	        return new Endpoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine_id = source["engine_id"];
	        this.model = source["model"];
	    }
	}
	export class Suggestion {
	    id: string;
	    feature: string;
	    reason: string;
	    evidence: string;
	    from: Endpoint;
	    to: Candidate;
	    score_gap: number;

	    static createFrom(source: any = {}) {
	        return new Suggestion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.feature = source["feature"];
	        this.reason = source["reason"];
	        this.evidence = source["evidence"];
	        this.from = this.convertValues(source["from"], Endpoint);
	        this.to = this.convertValues(source["to"], Candidate);
	        this.score_gap = source["score_gap"];
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

export namespace session {

	export class TokenStats {
	    promptTokens: number;
	    completionTokens: number;
	    totalTokens: number;
	    cacheHitTokens: number;
	    cacheMissTokens: number;
	    reasoningTokens: number;
	    usageCount: number;
	    cost: number;
	    currency?: string;
	    mainCost: number;
	    subagentCost: number;

	    static createFrom(source: any = {}) {
	        return new TokenStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptTokens = source["promptTokens"];
	        this.completionTokens = source["completionTokens"];
	        this.totalTokens = source["totalTokens"];
	        this.cacheHitTokens = source["cacheHitTokens"];
	        this.cacheMissTokens = source["cacheMissTokens"];
	        this.reasoningTokens = source["reasoningTokens"];
	        this.usageCount = source["usageCount"];
	        this.cost = source["cost"];
	        this.currency = source["currency"];
	        this.mainCost = source["mainCost"];
	        this.subagentCost = source["subagentCost"];
	    }
	}

}

export namespace skillstats {

	export class StatView {
	    name: string;
	    calls: number;
	    ok: number;
	    lastAt: number;

	    static createFrom(source: any = {}) {
	        return new StatView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.calls = source["calls"];
	        this.ok = source["ok"];
	        this.lastAt = source["lastAt"];
	    }
	}

}

export namespace standard {

	export class Issue {
	    element: string;
	    found: boolean;
	    note: string;
	    spec?: string;

	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.element = source["element"];
	        this.found = source["found"];
	        this.note = source["note"];
	        this.spec = source["spec"];
	    }
	}
	export class LintReport {
	    path: string;
	    issues: Issue[];
	    passed: boolean;
	    summary: string;

	    static createFrom(source: any = {}) {
	        return new LintReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.issues = this.convertValues(source["issues"], Issue);
	        this.passed = source["passed"];
	        this.summary = source["summary"];
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

export namespace tasks {

	export class Task {
	    id: string;
	    kind: string;
	    label: string;
	    status: string;
	    progress: number;
	    message: string;
	    error: string;
	    retryCount: number;
	    maxRetries: number;
	    payload: string;
	    result: string;
	    createdAt: number;
	    startedAt: number;
	    finishedAt: number;
	    spaceId?: string;
	    session_id?: string;
	    outputTail?: string;
	    outputTruncated?: boolean;
	    exitCode?: number;

	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.message = source["message"];
	        this.error = source["error"];
	        this.retryCount = source["retryCount"];
	        this.maxRetries = source["maxRetries"];
	        this.payload = source["payload"];
	        this.result = source["result"];
	        this.createdAt = source["createdAt"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.spaceId = source["spaceId"];
	        this.session_id = source["session_id"];
	        this.outputTail = source["outputTail"];
	        this.outputTruncated = source["outputTruncated"];
	        this.exitCode = source["exitCode"];
	    }
	}

}

export namespace trajectory {

	export class AgentNode {
	    id: string;
	    name: string;
	    kind: string;
	    status: string;
	    model?: string;
	    task?: string;
	    toolCalls: number;
	    errors: number;
	    tokens: number;
	    firstTs?: number;
	    lastTs?: number;
	    children?: AgentNode[];

	    static createFrom(source: any = {}) {
	        return new AgentNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.model = source["model"];
	        this.task = source["task"];
	        this.toolCalls = source["toolCalls"];
	        this.errors = source["errors"];
	        this.tokens = source["tokens"];
	        this.firstTs = source["firstTs"];
	        this.lastTs = source["lastTs"];
	        this.children = this.convertValues(source["children"], AgentNode);
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
	export class AgentNetwork {
	    ok: boolean;
	    window: number;
	    root: AgentNode;

	    static createFrom(source: any = {}) {
	        return new AgentNetwork(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.window = source["window"];
	        this.root = this.convertValues(source["root"], AgentNode);
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

	export class ApprovalRec {
	    tool?: string;
	    subject?: string;

	    static createFrom(source: any = {}) {
	        return new ApprovalRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.subject = source["subject"];
	    }
	}
	export class AskRec {
	    question?: string;

	    static createFrom(source: any = {}) {
	        return new AskRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.question = source["question"];
	    }
	}
	export class Usage {
	    promptTokens?: number;
	    completionTokens?: number;
	    cacheHitTokens?: number;
	    cacheMissTokens?: number;
	    reasoningTokens?: number;

	    static createFrom(source: any = {}) {
	        return new Usage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptTokens = source["promptTokens"];
	        this.completionTokens = source["completionTokens"];
	        this.cacheHitTokens = source["cacheHitTokens"];
	        this.cacheMissTokens = source["cacheMissTokens"];
	        this.reasoningTokens = source["reasoningTokens"];
	    }
	}
	export class AssistantRec {
	    text?: string;
	    reasoning?: string;
	    usage?: Usage;

	    static createFrom(source: any = {}) {
	        return new AssistantRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.reasoning = source["reasoning"];
	        this.usage = this.convertValues(source["usage"], Usage);
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
	export class CompactRec {
	    trigger?: string;
	    summary?: string;

	    static createFrom(source: any = {}) {
	        return new CompactRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trigger = source["trigger"];
	        this.summary = source["summary"];
	    }
	}
	export class DeliverableEntry {
	    path: string;
	    tool: string;
	    turn: number;
	    updatedAt: number;
	    touches: number;

	    static createFrom(source: any = {}) {
	        return new DeliverableEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.tool = source["tool"];
	        this.turn = source["turn"];
	        this.updatedAt = source["updatedAt"];
	        this.touches = source["touches"];
	    }
	}
	export class HeaderRec {
	    system?: string;
	    toolCount: number;
	    tokens: number;
	    change?: string;

	    static createFrom(source: any = {}) {
	        return new HeaderRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = source["system"];
	        this.toolCount = source["toolCount"];
	        this.tokens = source["tokens"];
	        this.change = source["change"];
	    }
	}
	export class SubagentRec {
	    ref?: string;
	    text?: string;
	    parentId?: string;

	    static createFrom(source: any = {}) {
	        return new SubagentRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.text = source["text"];
	        this.parentId = source["parentId"];
	    }
	}
	export class ToolRec {
	    id: string;
	    name: string;
	    args?: string;
	    output?: string;
	    err?: string;
	    truncated?: boolean;
	    readOnly?: boolean;
	    status: string;
	    parentId?: string;

	    static createFrom(source: any = {}) {
	        return new ToolRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.args = source["args"];
	        this.output = source["output"];
	        this.err = source["err"];
	        this.truncated = source["truncated"];
	        this.readOnly = source["readOnly"];
	        this.status = source["status"];
	        this.parentId = source["parentId"];
	    }
	}
	export class UserRec {
	    text: string;

	    static createFrom(source: any = {}) {
	        return new UserRec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	    }
	}
	export class Record {
	    seq: number;
	    kind: string;
	    ts: number;
	    durationMs?: number;
	    step?: number;
	    user?: UserRec;
	    header?: HeaderRec;
	    assistant?: AssistantRec;
	    tool?: ToolRec;
	    compact?: CompactRec;
	    ask?: AskRec;
	    approval?: ApprovalRec;
	    subagent?: SubagentRec;

	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.kind = source["kind"];
	        this.ts = source["ts"];
	        this.durationMs = source["durationMs"];
	        this.step = source["step"];
	        this.user = this.convertValues(source["user"], UserRec);
	        this.header = this.convertValues(source["header"], HeaderRec);
	        this.assistant = this.convertValues(source["assistant"], AssistantRec);
	        this.tool = this.convertValues(source["tool"], ToolRec);
	        this.compact = this.convertValues(source["compact"], CompactRec);
	        this.ask = this.convertValues(source["ask"], AskRec);
	        this.approval = this.convertValues(source["approval"], ApprovalRec);
	        this.subagent = this.convertValues(source["subagent"], SubagentRec);
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


	export class TurnEnd {
	    seq: number;
	    ts: number;
	    err?: string;

	    static createFrom(source: any = {}) {
	        return new TurnEnd(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.ts = source["ts"];
	        this.err = source["err"];
	    }
	}
	export class Turn {
	    turn: number;
	    startedAt?: number;
	    durationMs?: number;
	    end?: TurnEnd;
	    records: Record[];

	    static createFrom(source: any = {}) {
	        return new Turn(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turn = source["turn"];
	        this.startedAt = source["startedAt"];
	        this.durationMs = source["durationMs"];
	        this.end = this.convertValues(source["end"], TurnEnd);
	        this.records = this.convertValues(source["records"], Record);
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
	export class Trajectory {
	    ok: boolean;
	    turns: Turn[];
	    betweenTurns?: Record[];

	    static createFrom(source: any = {}) {
	        return new Trajectory(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.turns = this.convertValues(source["turns"], Turn);
	        this.betweenTurns = this.convertValues(source["betweenTurns"], Record);
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

export namespace tts {

	export class TTSParams {
	    speed: number;
	    pitch: number;
	    style: string;
	    emotion: string;

	    static createFrom(source: any = {}) {
	        return new TTSParams(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.speed = source["speed"];
	        this.pitch = source["pitch"];
	        this.style = source["style"];
	        this.emotion = source["emotion"];
	    }
	}

}

export namespace types {

	export class AnalysisScores {
	    pacing: number;
	    engagement: number;
	    coherence: number;
	    overall: number;
	    score_justification: string;

	    static createFrom(source: any = {}) {
	        return new AnalysisScores(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pacing = source["pacing"];
	        this.engagement = source["engagement"];
	        this.coherence = source["coherence"];
	        this.overall = source["overall"];
	        this.score_justification = source["score_justification"];
	    }
	}
	export class AnalysisScene {
	    location: string;
	    atmosphere: string;
	    duration: string;

	    static createFrom(source: any = {}) {
	        return new AnalysisScene(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.location = source["location"];
	        this.atmosphere = source["atmosphere"];
	        this.duration = source["duration"];
	    }
	}
	export class PlotPoint {
	    content: string;
	    type: string;
	    importance: number;
	    impact: string;
	    keyword: Keyword;

	    static createFrom(source: any = {}) {
	        return new PlotPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.type = source["type"];
	        this.importance = source["importance"];
	        this.impact = source["impact"];
	        this.keyword = this.convertValues(source["keyword"], Keyword);
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
	export class OrgMemberChangeV2 {
	    character_name: string;
	    change_type: string;
	    position?: string;
	    loyalty_hint?: number;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new OrgMemberChangeV2(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.character_name = source["character_name"];
	        this.change_type = source["change_type"];
	        this.position = source["position"];
	        this.loyalty_hint = source["loyalty_hint"];
	        this.reason = source["reason"];
	    }
	}
	export class OrganizationStateChangeV2 {
	    org_name: string;
	    power_value?: number;
	    destroyed?: boolean;
	    member_changes?: OrgMemberChangeV2[];

	    static createFrom(source: any = {}) {
	        return new OrganizationStateChangeV2(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.org_name = source["org_name"];
	        this.power_value = source["power_value"];
	        this.destroyed = source["destroyed"];
	        this.member_changes = this.convertValues(source["member_changes"], OrgMemberChangeV2);
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
	export class CharacterStateChangeV2 {
	    name: string;
	    old_state?: string;
	    new_state?: string;
	    psychological_change?: string;
	    key_event?: string;
	    survival_status?: string;
	    relationship_changes?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new CharacterStateChangeV2(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.old_state = source["old_state"];
	        this.new_state = source["new_state"];
	        this.psychological_change = source["psychological_change"];
	        this.key_event = source["key_event"];
	        this.survival_status = source["survival_status"];
	        this.relationship_changes = source["relationship_changes"];
	    }
	}
	export class EmotionalArc {
	    primary_emotion: string;
	    intensity: number;
	    curve: string;
	    secondary_emotions?: string[];

	    static createFrom(source: any = {}) {
	        return new EmotionalArc(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.primary_emotion = source["primary_emotion"];
	        this.intensity = source["intensity"];
	        this.curve = source["curve"];
	        this.secondary_emotions = source["secondary_emotions"];
	    }
	}
	export class Conflict {
	    types: string[];
	    parties?: string[];
	    level: number;
	    description: string;
	    resolution_progress: number;

	    static createFrom(source: any = {}) {
	        return new Conflict(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.types = source["types"];
	        this.parties = source["parties"];
	        this.level = source["level"];
	        this.description = source["description"];
	        this.resolution_progress = source["resolution_progress"];
	    }
	}
	export class ForeshadowHit {
	    title: string;
	    content: string;
	    type: string;
	    strength: number;
	    subtlety: number;
	    category: string;
	    is_long_term: boolean;
	    related_characters?: string[];
	    estimated_resolve_chapter?: number;
	    reference_chapter?: number;
	    reference_stable_id?: string;
	    keyword: Keyword;

	    static createFrom(source: any = {}) {
	        return new ForeshadowHit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.content = source["content"];
	        this.type = source["type"];
	        this.strength = source["strength"];
	        this.subtlety = source["subtlety"];
	        this.category = source["category"];
	        this.is_long_term = source["is_long_term"];
	        this.related_characters = source["related_characters"];
	        this.estimated_resolve_chapter = source["estimated_resolve_chapter"];
	        this.reference_chapter = source["reference_chapter"];
	        this.reference_stable_id = source["reference_stable_id"];
	        this.keyword = this.convertValues(source["keyword"], Keyword);
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
	export class Keyword {
	    text: string;
	    pos: number;
	    length: number;

	    static createFrom(source: any = {}) {
	        return new Keyword(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.pos = source["pos"];
	        this.length = source["length"];
	    }
	}
	export class Hook {
	    type: string;
	    content: string;
	    strength: number;
	    position: string;
	    keyword: Keyword;

	    static createFrom(source: any = {}) {
	        return new Hook(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.content = source["content"];
	        this.strength = source["strength"];
	        this.position = source["position"];
	        this.keyword = this.convertValues(source["keyword"], Keyword);
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
	export class AnalysisResultV2 {
	    hooks: Hook[];
	    foreshadows: ForeshadowHit[];
	    conflict: Conflict;
	    emotional_arc: EmotionalArc;
	    character_states: CharacterStateChangeV2[];
	    organization_states?: OrganizationStateChangeV2[];
	    plot_points: PlotPoint[];
	    scenes: AnalysisScene[];
	    pacing: string;
	    dialogue_ratio: number;
	    description_ratio: number;
	    scores: AnalysisScores;
	    plot_stage: string;
	    suggestions: string[];
	    summary: string;

	    static createFrom(source: any = {}) {
	        return new AnalysisResultV2(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hooks = this.convertValues(source["hooks"], Hook);
	        this.foreshadows = this.convertValues(source["foreshadows"], ForeshadowHit);
	        this.conflict = this.convertValues(source["conflict"], Conflict);
	        this.emotional_arc = this.convertValues(source["emotional_arc"], EmotionalArc);
	        this.character_states = this.convertValues(source["character_states"], CharacterStateChangeV2);
	        this.organization_states = this.convertValues(source["organization_states"], OrganizationStateChangeV2);
	        this.plot_points = this.convertValues(source["plot_points"], PlotPoint);
	        this.scenes = this.convertValues(source["scenes"], AnalysisScene);
	        this.pacing = source["pacing"];
	        this.dialogue_ratio = source["dialogue_ratio"];
	        this.description_ratio = source["description_ratio"];
	        this.scores = this.convertValues(source["scores"], AnalysisScores);
	        this.plot_stage = source["plot_stage"];
	        this.suggestions = source["suggestions"];
	        this.summary = source["summary"];
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


	export class Annotation {
	    type: string;
	    title?: string;
	    content: string;
	    importance?: number;
	    pos: number;
	    length?: number;
	    tags?: string[];

	    static createFrom(source: any = {}) {
	        return new Annotation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.importance = source["importance"];
	        this.pos = source["pos"];
	        this.length = source["length"];
	        this.tags = source["tags"];
	    }
	}
	export class ChapterAnalysisResult {
	    chapter_num: number;
	    chapter_file?: string;
	    analyzed_at?: string;
	    engine?: string;
	    model?: string;
	    analyzer_source?: string;
	    result: AnalysisResultV2;

	    static createFrom(source: any = {}) {
	        return new ChapterAnalysisResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_num = source["chapter_num"];
	        this.chapter_file = source["chapter_file"];
	        this.analyzed_at = source["analyzed_at"];
	        this.engine = source["engine"];
	        this.model = source["model"];
	        this.analyzer_source = source["analyzer_source"];
	        this.result = this.convertValues(source["result"], AnalysisResultV2);
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
	export class FinalizeForeshadowSync {
	    planted_count: number;
	    resolved_count: number;
	    created_count: number;
	    errors?: string[];

	    static createFrom(source: any = {}) {
	        return new FinalizeForeshadowSync(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.planted_count = source["planted_count"];
	        this.resolved_count = source["resolved_count"];
	        this.created_count = source["created_count"];
	        this.errors = source["errors"];
	    }
	}
	export class StageRecapEntry {
	    start: number;
	    end: number;
	    text: string;
	    finalized_at: string;
	    source_chapter: number;

	    static createFrom(source: any = {}) {
	        return new StageRecapEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.text = source["text"];
	        this.finalized_at = source["finalized_at"];
	        this.source_chapter = source["source_chapter"];
	    }
	}
	export class FinalizeApplied {
	    stage_recap?: StageRecapEntry;
	    characters: number;
	    foreshadows?: FinalizeForeshadowSync;

	    static createFrom(source: any = {}) {
	        return new FinalizeApplied(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stage_recap = this.convertValues(source["stage_recap"], StageRecapEntry);
	        this.characters = source["characters"];
	        this.foreshadows = this.convertValues(source["foreshadows"], FinalizeForeshadowSync);
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
	export class ChapterFinalization {
	    version: number;
	    chapter_num: number;
	    branch?: string;
	    finalized_at: string;
	    note?: string;
	    applied?: FinalizeApplied;
	    skipped?: string;
	    errors?: string[];

	    static createFrom(source: any = {}) {
	        return new ChapterFinalization(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.chapter_num = source["chapter_num"];
	        this.branch = source["branch"];
	        this.finalized_at = source["finalized_at"];
	        this.note = source["note"];
	        this.applied = this.convertValues(source["applied"], FinalizeApplied);
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
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
	export class ChapterPlan {
	    sub_index: number;
	    title: string;
	    plot_summary: string;
	    key_events: string[];
	    character_focus: string[];
	    emotional_tone: string;
	    narrative_goal: string;
	    conflict_type: string;
	    ending_type: string;
	    estimated_words?: number;

	    static createFrom(source: any = {}) {
	        return new ChapterPlan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sub_index = source["sub_index"];
	        this.title = source["title"];
	        this.plot_summary = source["plot_summary"];
	        this.key_events = source["key_events"];
	        this.character_focus = source["character_focus"];
	        this.emotional_tone = source["emotional_tone"];
	        this.narrative_goal = source["narrative_goal"];
	        this.conflict_type = source["conflict_type"];
	        this.ending_type = source["ending_type"];
	        this.estimated_words = source["estimated_words"];
	    }
	}










	export class PlanDeviation {
	    chapterNum: number;
	    hasPlan: boolean;
	    analyzed: boolean;
	    missingEvents?: string[];
	    endingMismatch: boolean;
	    plannedEnding?: string;
	    actualEnding?: string;
	    emotionDrift: boolean;
	    plannedEmotion?: string;
	    actualEmotion?: string;
	    duplicateEvents?: string[];
	    summary?: string;
	    nextSuggestion?: string;

	    static createFrom(source: any = {}) {
	        return new PlanDeviation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterNum = source["chapterNum"];
	        this.hasPlan = source["hasPlan"];
	        this.analyzed = source["analyzed"];
	        this.missingEvents = source["missingEvents"];
	        this.endingMismatch = source["endingMismatch"];
	        this.plannedEnding = source["plannedEnding"];
	        this.actualEnding = source["actualEnding"];
	        this.emotionDrift = source["emotionDrift"];
	        this.plannedEmotion = source["plannedEmotion"];
	        this.actualEmotion = source["actualEmotion"];
	        this.duplicateEvents = source["duplicateEvents"];
	        this.summary = source["summary"];
	        this.nextSuggestion = source["nextSuggestion"];
	    }
	}
	export class PlanProblem {
	    code: string;
	    severity: string;
	    message: string;
	    evidence?: string;

	    static createFrom(source: any = {}) {
	        return new PlanProblem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.severity = source["severity"];
	        this.message = source["message"];
	        this.evidence = source["evidence"];
	    }
	}
	export class PlanGateReport {
	    chapterNum: number;
	    allowed: boolean;
	    hasPlan: boolean;
	    missing?: string[];
	    planProblems?: PlanProblem[];
	    outlineIssues?: PlanProblem[];
	    blocking: boolean;

	    static createFrom(source: any = {}) {
	        return new PlanGateReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterNum = source["chapterNum"];
	        this.allowed = source["allowed"];
	        this.hasPlan = source["hasPlan"];
	        this.missing = source["missing"];
	        this.planProblems = this.convertValues(source["planProblems"], PlanProblem);
	        this.outlineIssues = this.convertValues(source["outlineIssues"], PlanProblem);
	        this.blocking = source["blocking"];
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


	export class PreserveConfig {
	    preserve_structure: boolean;
	    preserve_dialogues?: string[];
	    preserve_plot_points?: string[];
	    preserve_character_traits: boolean;

	    static createFrom(source: any = {}) {
	        return new PreserveConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preserve_structure = source["preserve_structure"];
	        this.preserve_dialogues = source["preserve_dialogues"];
	        this.preserve_plot_points = source["preserve_plot_points"];
	        this.preserve_character_traits = source["preserve_character_traits"];
	    }
	}
	export class RewriteVersion {
	    id: string;
	    chapter_num: number;
	    mode: string;
	    status: string;
	    source: string;
	    suggestion_idxs?: number[];
	    custom_instructions?: string;
	    focus_areas?: string[];
	    preserve_elements?: PreserveConfig;
	    start_pos?: number;
	    end_pos?: number;
	    length_mode?: string;
	    target_word_count?: number;
	    scene_id?: string;
	    scene_original?: string;
	    scene_new?: string;
	    scene_start?: number;
	    scene_end?: number;
	    original_content: string;
	    original_word_count: number;
	    new_content?: string;
	    new_word_count?: number;
	    similarity?: number;
	    before_ai_score?: number;
	    after_ai_score?: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    applied_at?: any;
	    // Go type: time
	    restored_at?: any;
	    restored_from?: string;

	    static createFrom(source: any = {}) {
	        return new RewriteVersion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chapter_num = source["chapter_num"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.suggestion_idxs = source["suggestion_idxs"];
	        this.custom_instructions = source["custom_instructions"];
	        this.focus_areas = source["focus_areas"];
	        this.preserve_elements = this.convertValues(source["preserve_elements"], PreserveConfig);
	        this.start_pos = source["start_pos"];
	        this.end_pos = source["end_pos"];
	        this.length_mode = source["length_mode"];
	        this.target_word_count = source["target_word_count"];
	        this.scene_id = source["scene_id"];
	        this.scene_original = source["scene_original"];
	        this.scene_new = source["scene_new"];
	        this.scene_start = source["scene_start"];
	        this.scene_end = source["scene_end"];
	        this.original_content = source["original_content"];
	        this.original_word_count = source["original_word_count"];
	        this.new_content = source["new_content"];
	        this.new_word_count = source["new_word_count"];
	        this.similarity = source["similarity"];
	        this.before_ai_score = source["before_ai_score"];
	        this.after_ai_score = source["after_ai_score"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.applied_at = this.convertValues(source["applied_at"], null);
	        this.restored_at = this.convertValues(source["restored_at"], null);
	        this.restored_from = source["restored_from"];
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
	export class RewriteVersionIndex {
	    id: string;
	    chapter_num: number;
	    mode: string;
	    status: string;
	    similarity?: number;
	    // Go type: time
	    created_at: any;

	    static createFrom(source: any = {}) {
	        return new RewriteVersionIndex(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chapter_num = source["chapter_num"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.similarity = source["similarity"];
	        this.created_at = this.convertValues(source["created_at"], null);
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

	export class StoryHealthFinding {
	    code: string;
	    severity: string;
	    message: string;
	    ref?: string;

	    static createFrom(source: any = {}) {
	        return new StoryHealthFinding(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.severity = source["severity"];
	        this.message = source["message"];
	        this.ref = source["ref"];
	    }
	}

}

export namespace whisper {

	export class GraphEdge {
	    from: string;
	    to: string;
	    type: string;
	    weight: number;
	    emotionLabel?: string;

	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.type = source["type"];
	        this.weight = source["weight"];
	        this.emotionLabel = source["emotionLabel"];
	    }
	}
	export class GraphNode {
	    id: string;
	    name: string;
	    type: string;
	    weight: number;

	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.weight = source["weight"];
	    }
	}
	export class PersonalityDims {
	    T: number;
	    I: number;
	    S: number;
	    O: number;
	    R: number;

	    static createFrom(source: any = {}) {
	        return new PersonalityDims(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.T = source["T"];
	        this.I = source["I"];
	        this.S = source["S"];
	        this.O = source["O"];
	        this.R = source["R"];
	    }
	}
	export class PersonalityPreset {
	    id: string;
	    label: string;
	    gender: string;
	    dims: PersonalityDims;
	    tags?: string[];
	    hiddenPersona?: PersonalityDims;
	    requiresAdult18?: boolean;
	    voiceGuide?: string;

	    static createFrom(source: any = {}) {
	        return new PersonalityPreset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.gender = source["gender"];
	        this.dims = this.convertValues(source["dims"], PersonalityDims);
	        this.tags = source["tags"];
	        this.hiddenPersona = this.convertValues(source["hiddenPersona"], PersonalityDims);
	        this.requiresAdult18 = source["requiresAdult18"];
	        this.voiceGuide = source["voiceGuide"];
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
	export class Subgraph {
	    nodes: GraphNode[];
	    edges: GraphEdge[];

	    static createFrom(source: any = {}) {
	        return new Subgraph(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], GraphNode);
	        this.edges = this.convertValues(source["edges"], GraphEdge);
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
	export class TurnTraceL5 {
	    toolCalls?: string[];

	    static createFrom(source: any = {}) {
	        return new TurnTraceL5(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolCalls = source["toolCalls"];
	    }
	}
	export class TurnTraceL4 {
	    wrote: boolean;

	    static createFrom(source: any = {}) {
	        return new TurnTraceL4(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wrote = source["wrote"];
	    }
	}
	export class TurnTraceL3 {
	    silent: boolean;
	    tierBChars: number;
	    factsUsed?: number;
	    embeddingHits?: number;
	    topicSource?: string;
	    emergenceType?: string;

	    static createFrom(source: any = {}) {
	        return new TurnTraceL3(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silent = source["silent"];
	        this.tierBChars = source["tierBChars"];
	        this.factsUsed = source["factsUsed"];
	        this.embeddingHits = source["embeddingHits"];
	        this.topicSource = source["topicSource"];
	        this.emergenceType = source["emergenceType"];
	    }
	}
	export class TurnTraceL2 {
	    aff: number;
	    sec: number;
	    aro: number;
	    dom: number;
	    label: string;

	    static createFrom(source: any = {}) {
	        return new TurnTraceL2(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aff = source["aff"];
	        this.sec = source["sec"];
	        this.aro = source["aro"];
	        this.dom = source["dom"];
	        this.label = source["label"];
	    }
	}
	export class TurnTraceL1 {
	    trust: number;
	    rifts: number;
	    stage: string;
	    atmosphere?: string;

	    static createFrom(source: any = {}) {
	        return new TurnTraceL1(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trust = source["trust"];
	        this.rifts = source["rifts"];
	        this.stage = source["stage"];
	        this.atmosphere = source["atmosphere"];
	    }
	}
	export class TurnTraceL0 {
	    type: string;
	    intensity: number;
	    sincerity?: number;

	    static createFrom(source: any = {}) {
	        return new TurnTraceL0(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.intensity = source["intensity"];
	        this.sincerity = source["sincerity"];
	    }
	}
	export class TurnTrace {
	    turn: number;
	    l0: TurnTraceL0;
	    l1: TurnTraceL1;
	    l2: TurnTraceL2;
	    l3: TurnTraceL3;
	    l4: TurnTraceL4;
	    l5?: TurnTraceL5;
	    // Go type: time
	    timestamp?: any;

	    static createFrom(source: any = {}) {
	        return new TurnTrace(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turn = source["turn"];
	        this.l0 = this.convertValues(source["l0"], TurnTraceL0);
	        this.l1 = this.convertValues(source["l1"], TurnTraceL1);
	        this.l2 = this.convertValues(source["l2"], TurnTraceL2);
	        this.l3 = this.convertValues(source["l3"], TurnTraceL3);
	        this.l4 = this.convertValues(source["l4"], TurnTraceL4);
	        this.l5 = this.convertValues(source["l5"], TurnTraceL5);
	        this.timestamp = this.convertValues(source["timestamp"], null);
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

export namespace workcost {

	export class ApplyProjectResult {
	    project: string;
	    resourceNew: number;
	    resourceUpd: number;
	    quotaNew: number;
	    quotaUpd: number;
	    lines: number;
	    errors: string[];

	    static createFrom(source: any = {}) {
	        return new ApplyProjectResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project = source["project"];
	        this.resourceNew = source["resourceNew"];
	        this.resourceUpd = source["resourceUpd"];
	        this.quotaNew = source["quotaNew"];
	        this.quotaUpd = source["quotaUpd"];
	        this.lines = source["lines"];
	        this.errors = source["errors"];
	    }
	}
	export class AuditEvidence {
	    ref: string;
	    source: string;
	    support: string;

	    static createFrom(source: any = {}) {
	        return new AuditEvidence(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.source = source["source"];
	        this.support = source["support"];
	    }
	}
	export class AuditFinding {
	    kind: string;
	    target: string;
	    detail: string;
	    evidence: AuditEvidence[];

	    static createFrom(source: any = {}) {
	        return new AuditFinding(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.target = source["target"];
	        this.detail = source["detail"];
	        this.evidence = this.convertValues(source["evidence"], AuditEvidence);
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
	export class AuditReport {
	    billGaps: AuditFinding[];
	    quotaGaps: AuditFinding[];
	    auditedProj: string;
	    projectHits: number;

	    static createFrom(source: any = {}) {
	        return new AuditReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.billGaps = this.convertValues(source["billGaps"], AuditFinding);
	        this.quotaGaps = this.convertValues(source["quotaGaps"], AuditFinding);
	        this.auditedProj = source["auditedProj"];
	        this.projectHits = source["projectHits"];
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
	export class BillItem {
	    id: number;
	    projectId: number;
	    code: string;
	    title: string;
	    unit: string;
	    division: string;
	    quantity: number;
	    quantityExpr: string;
	    quotaCode: string;
	    feature: string;
	    priceOverride: number;
	    sort: number;
	    quotaCodes: string[];

	    static createFrom(source: any = {}) {
	        return new BillItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.code = source["code"];
	        this.title = source["title"];
	        this.unit = source["unit"];
	        this.division = source["division"];
	        this.quantity = source["quantity"];
	        this.quantityExpr = source["quantityExpr"];
	        this.quotaCode = source["quotaCode"];
	        this.feature = source["feature"];
	        this.priceOverride = source["priceOverride"];
	        this.sort = source["sort"];
	        this.quotaCodes = source["quotaCodes"];
	    }
	}
	export class BillProject {
	    id: number;
	    name: string;
	    fileName: string;
	    source: string;
	    location: string;
	    duration: string;
	    pricing: string;
	    managementRate: number;
	    regulatoryRate: number;
	    profitRate: number;
	    taxRate: number;
	    profitIncludesRegulatory: boolean;
	    controlPrice: number;
	    itemCount: number;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new BillProject(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.fileName = source["fileName"];
	        this.source = source["source"];
	        this.location = source["location"];
	        this.duration = source["duration"];
	        this.pricing = source["pricing"];
	        this.managementRate = source["managementRate"];
	        this.regulatoryRate = source["regulatoryRate"];
	        this.profitRate = source["profitRate"];
	        this.taxRate = source["taxRate"];
	        this.profitIncludesRegulatory = source["profitIncludesRegulatory"];
	        this.controlPrice = source["controlPrice"];
	        this.itemCount = source["itemCount"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class BillQuotaLink {
	    id: number;
	    billItemId: number;
	    quotaCode: string;
	    quantity: number;
	    sort: number;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new BillQuotaLink(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.billItemId = source["billItemId"];
	        this.quotaCode = source["quotaCode"];
	        this.quantity = source["quantity"];
	        this.sort = source["sort"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class BulkPriceMatch {
	    row: number;
	    resourceId: number;
	    code: string;
	    title: string;
	    spec?: string;
	    unit?: string;
	    oldPrice: number;
	    newPrice: number;
	    period?: string;
	    region?: string;
	    note?: string;
	    resourceKind?: string;

	    static createFrom(source: any = {}) {
	        return new BulkPriceMatch(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row = source["row"];
	        this.resourceId = source["resourceId"];
	        this.code = source["code"];
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.oldPrice = source["oldPrice"];
	        this.newPrice = source["newPrice"];
	        this.period = source["period"];
	        this.region = source["region"];
	        this.note = source["note"];
	        this.resourceKind = source["resourceKind"];
	    }
	}
	export class BulkPriceSkip {
	    row: number;
	    title: string;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new BulkPriceSkip(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row = source["row"];
	        this.title = source["title"];
	        this.reason = source["reason"];
	    }
	}
	export class BulkPricePreview {
	    total: number;
	    matched: BulkPriceMatch[];
	    skipped: BulkPriceSkip[];
	    rowErrors: BulkPriceSkip[];

	    static createFrom(source: any = {}) {
	        return new BulkPricePreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.matched = this.convertValues(source["matched"], BulkPriceMatch);
	        this.skipped = this.convertValues(source["skipped"], BulkPriceSkip);
	        this.rowErrors = this.convertValues(source["rowErrors"], BulkPriceSkip);
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
	export class BulkPriceResult {
	    updated: number;
	    skipped: BulkPriceSkip[];
	    total: number;
	    applied: BulkPriceMatch[];

	    static createFrom(source: any = {}) {
	        return new BulkPriceResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updated = source["updated"];
	        this.skipped = this.convertValues(source["skipped"], BulkPriceSkip);
	        this.total = source["total"];
	        this.applied = this.convertValues(source["applied"], BulkPriceMatch);
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

	export class ClassifyCount {
	    labor: number;
	    material: number;
	    machine: number;
	    outsourced: number;

	    static createFrom(source: any = {}) {
	        return new ClassifyCount(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.labor = source["labor"];
	        this.material = source["material"];
	        this.machine = source["machine"];
	        this.outsourced = source["outsourced"];
	    }
	}
	export class HealthFinding {
	    id: number;
	    code: string;
	    title: string;
	    kind?: string;
	    unit?: string;
	    detail?: string;

	    static createFrom(source: any = {}) {
	        return new HealthFinding(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.code = source["code"];
	        this.title = source["title"];
	        this.kind = source["kind"];
	        this.unit = source["unit"];
	        this.detail = source["detail"];
	    }
	}
	export class CodeMismatchGroup {
	    title: string;
	    items: HealthFinding[];

	    static createFrom(source: any = {}) {
	        return new CodeMismatchGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.items = this.convertValues(source["items"], HealthFinding);
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
	export class ComposeLineView {
	    kind: string;
	    title: string;
	    unit: string;
	    quantity: number;
	    price: number;
	    lossRate: number;
	    amount: number;
	    sharePct: number;

	    static createFrom(source: any = {}) {
	        return new ComposeLineView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.unit = source["unit"];
	        this.quantity = source["quantity"];
	        this.price = source["price"];
	        this.lossRate = source["lossRate"];
	        this.amount = source["amount"];
	        this.sharePct = source["sharePct"];
	    }
	}
	export class Compose {
	    laborFee: number;
	    materialFee: number;
	    machineFee: number;
	    outsourcedFee: number;
	    otherFee: number;
	    subtotal: number;
	    compositePrice: number;
	    zeroLines: number;
	    lines: ComposeLineView[];
	    warnings: string[];

	    static createFrom(source: any = {}) {
	        return new Compose(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.laborFee = source["laborFee"];
	        this.materialFee = source["materialFee"];
	        this.machineFee = source["machineFee"];
	        this.outsourcedFee = source["outsourcedFee"];
	        this.otherFee = source["otherFee"];
	        this.subtotal = source["subtotal"];
	        this.compositePrice = source["compositePrice"];
	        this.zeroLines = source["zeroLines"];
	        this.lines = this.convertValues(source["lines"], ComposeLineView);
	        this.warnings = source["warnings"];
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

	export class DedupeResult {
	    kept: string;
	    merged: number;
	    redirectedItems: number;
	    redirectedPrices: number;

	    static createFrom(source: any = {}) {
	        return new DedupeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kept = source["kept"];
	        this.merged = source["merged"];
	        this.redirectedItems = source["redirectedItems"];
	        this.redirectedPrices = source["redirectedPrices"];
	    }
	}
	export class DuplicateGroup {
	    kind: string;
	    title: string;
	    detail?: string;

	    static createFrom(source: any = {}) {
	        return new DuplicateGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	    }
	}
	export class FeeResult {
	    directFee: number;
	    managementFee: number;
	    profitFee: number;
	    regulatoryFee: number;
	    measuresFee: number;
	    contingency: number;
	    preTaxTotal: number;
	    taxFee: number;
	    total: number;
	    controlPrice: number;
	    controlDiff: number;
	    controlUtilPct: number;

	    static createFrom(source: any = {}) {
	        return new FeeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.directFee = source["directFee"];
	        this.managementFee = source["managementFee"];
	        this.profitFee = source["profitFee"];
	        this.regulatoryFee = source["regulatoryFee"];
	        this.measuresFee = source["measuresFee"];
	        this.contingency = source["contingency"];
	        this.preTaxTotal = source["preTaxTotal"];
	        this.taxFee = source["taxFee"];
	        this.total = source["total"];
	        this.controlPrice = source["controlPrice"];
	        this.controlDiff = source["controlDiff"];
	        this.controlUtilPct = source["controlUtilPct"];
	    }
	}

	export class HealthReport {
	    zeroPrice: HealthFinding[];
	    quotaNoPrice: HealthFinding[];
	    brokenRef: HealthFinding[];
	    duplicates: DuplicateGroup[];
	    codeMismatch: CodeMismatchGroup[];
	    totalFindings: number;

	    static createFrom(source: any = {}) {
	        return new HealthReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.zeroPrice = this.convertValues(source["zeroPrice"], HealthFinding);
	        this.quotaNoPrice = this.convertValues(source["quotaNoPrice"], HealthFinding);
	        this.brokenRef = this.convertValues(source["brokenRef"], HealthFinding);
	        this.duplicates = this.convertValues(source["duplicates"], DuplicateGroup);
	        this.codeMismatch = this.convertValues(source["codeMismatch"], CodeMismatchGroup);
	        this.totalFindings = source["totalFindings"];
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
	export class SkippedRow {
	    sheet: string;
	    row: number;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new SkippedRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sheet = source["sheet"];
	        this.row = source["row"];
	        this.reason = source["reason"];
	    }
	}
	export class ProjectQuantity {
	    code: string;
	    title: string;
	    value: number;
	    unit: string;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new ProjectQuantity(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.title = source["title"];
	        this.value = source["value"];
	        this.unit = source["unit"];
	        this.note = source["note"];
	    }
	}
	export class ProjectFeeParams {
	    managementRate: number;
	    regulatoryRate: number;
	    profitRate: number;
	    taxRate: number;
	    controlPrice: number;
	    managementFormula: string;
	    taxNote: string;

	    static createFrom(source: any = {}) {
	        return new ProjectFeeParams(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.managementRate = source["managementRate"];
	        this.regulatoryRate = source["regulatoryRate"];
	        this.profitRate = source["profitRate"];
	        this.taxRate = source["taxRate"];
	        this.controlPrice = source["controlPrice"];
	        this.managementFormula = source["managementFormula"];
	        this.taxNote = source["taxNote"];
	    }
	}
	export class ProjectConsumption {
	    seq: string;
	    itemCode: string;
	    itemTitle: string;
	    itemUnit: string;
	    kind: string;
	    resourceCode: string;
	    resourceTitle: string;
	    unit: string;
	    quantity: number;
	    price: number;
	    priceFormula: string;
	    amount: number;

	    static createFrom(source: any = {}) {
	        return new ProjectConsumption(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.itemCode = source["itemCode"];
	        this.itemTitle = source["itemTitle"];
	        this.itemUnit = source["itemUnit"];
	        this.kind = source["kind"];
	        this.resourceCode = source["resourceCode"];
	        this.resourceTitle = source["resourceTitle"];
	        this.unit = source["unit"];
	        this.quantity = source["quantity"];
	        this.price = source["price"];
	        this.priceFormula = source["priceFormula"];
	        this.amount = source["amount"];
	    }
	}
	export class ProjectItem {
	    code: string;
	    title: string;
	    unit: string;
	    division: string;
	    feature: string;
	    quantity: number;
	    quantityExpr: string;
	    categoryPath: string;

	    static createFrom(source: any = {}) {
	        return new ProjectItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.title = source["title"];
	        this.unit = source["unit"];
	        this.division = source["division"];
	        this.feature = source["feature"];
	        this.quantity = source["quantity"];
	        this.quantityExpr = source["quantityExpr"];
	        this.categoryPath = source["categoryPath"];
	    }
	}
	export class ProjectResource {
	    code: string;
	    kind: string;
	    title: string;
	    spec: string;
	    unit: string;
	    price: number;
	    priceFormula: string;
	    source: string;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new ProjectResource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.price = source["price"];
	        this.priceFormula = source["priceFormula"];
	        this.source = source["source"];
	        this.note = source["note"];
	    }
	}
	export class ProjectBundle {
	    path: string;
	    fileName: string;
	    project: string;
	    location: string;
	    duration: string;
	    pricing: string;
	    resources: ProjectResource[];
	    items: ProjectItem[];
	    lines: ProjectConsumption[];
	    fee: ProjectFeeParams;
	    quantities: ProjectQuantity[];
	    skipped: SkippedRow[];
	    warnings: string[];
	    sheets: string[];

	    static createFrom(source: any = {}) {
	        return new ProjectBundle(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.project = source["project"];
	        this.location = source["location"];
	        this.duration = source["duration"];
	        this.pricing = source["pricing"];
	        this.resources = this.convertValues(source["resources"], ProjectResource);
	        this.items = this.convertValues(source["items"], ProjectItem);
	        this.lines = this.convertValues(source["lines"], ProjectConsumption);
	        this.fee = this.convertValues(source["fee"], ProjectFeeParams);
	        this.quantities = this.convertValues(source["quantities"], ProjectQuantity);
	        this.skipped = this.convertValues(source["skipped"], SkippedRow);
	        this.warnings = source["warnings"];
	        this.sheets = source["sheets"];
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





	export class QuotaItem {
	    id: number;
	    quotaCode: string;
	    resourceCode: string;
	    kind: string;
	    title: string;
	    spec: string;
	    unit: string;
	    quantity: number;
	    resourcePrice: number;
	    lossRate: number;
	    note: string;
	    sort: number;

	    static createFrom(source: any = {}) {
	        return new QuotaItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.quotaCode = source["quotaCode"];
	        this.resourceCode = source["resourceCode"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.quantity = source["quantity"];
	        this.resourcePrice = source["resourcePrice"];
	        this.lossRate = source["lossRate"];
	        this.note = source["note"];
	        this.sort = source["sort"];
	    }
	}
	export class Quota {
	    id: number;
	    code: string;
	    origCode: string;
	    title: string;
	    specialty: string;
	    chapter: string;
	    unit: string;
	    categoryPath: string;
	    baseLabor: number;
	    baseMaterial: number;
	    baseMachine: number;
	    source: string;
	    region: string;
	    priceDate: string;
	    note: string;
	    status: string;
	    items: QuotaItem[];
	    usageCount: number;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new Quota(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.code = source["code"];
	        this.origCode = source["origCode"];
	        this.title = source["title"];
	        this.specialty = source["specialty"];
	        this.chapter = source["chapter"];
	        this.unit = source["unit"];
	        this.categoryPath = source["categoryPath"];
	        this.baseLabor = source["baseLabor"];
	        this.baseMaterial = source["baseMaterial"];
	        this.baseMachine = source["baseMachine"];
	        this.source = source["source"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.note = source["note"];
	        this.status = source["status"];
	        this.items = this.convertValues(source["items"], QuotaItem);
	        this.usageCount = source["usageCount"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class QuotaFlywheelRow {
	    resourceCode: string;
	    kind: string;
	    title: string;
	    spec?: string;
	    unit?: string;
	    quantity: number;
	    coverage: number;
	    min: number;
	    max: number;
	    majority: boolean;

	    static createFrom(source: any = {}) {
	        return new QuotaFlywheelRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resourceCode = source["resourceCode"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.quantity = source["quantity"];
	        this.coverage = source["coverage"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.majority = source["majority"];
	    }
	}
	export class QuotaFlywheelMember {
	    code: string;
	    origCode: string;
	    source: string;
	    itemCount: number;

	    static createFrom(source: any = {}) {
	        return new QuotaFlywheelMember(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.origCode = source["origCode"];
	        this.source = source["source"];
	        this.itemCount = source["itemCount"];
	    }
	}
	export class QuotaFlywheelCandidate {
	    origCode: string;
	    newCode: string;
	    title: string;
	    unit: string;
	    members: QuotaFlywheelMember[];
	    rows: QuotaFlywheelRow[];
	    sources: number;

	    static createFrom(source: any = {}) {
	        return new QuotaFlywheelCandidate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.origCode = source["origCode"];
	        this.newCode = source["newCode"];
	        this.title = source["title"];
	        this.unit = source["unit"];
	        this.members = this.convertValues(source["members"], QuotaFlywheelMember);
	        this.rows = this.convertValues(source["rows"], QuotaFlywheelRow);
	        this.sources = source["sources"];
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

	export class QuotaFlywheelSkip {
	    key: string;
	    title: string;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new QuotaFlywheelSkip(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.title = source["title"];
	        this.reason = source["reason"];
	    }
	}
	export class QuotaFlywheelPreview {
	    clusters: number;
	    candidates: QuotaFlywheelCandidate[];
	    skipped: QuotaFlywheelSkip[];

	    static createFrom(source: any = {}) {
	        return new QuotaFlywheelPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusters = source["clusters"];
	        this.candidates = this.convertValues(source["candidates"], QuotaFlywheelCandidate);
	        this.skipped = this.convertValues(source["skipped"], QuotaFlywheelSkip);
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
	export class QuotaFlywheelResult {
	    saved: string[];
	    skipped: QuotaFlywheelSkip[];

	    static createFrom(source: any = {}) {
	        return new QuotaFlywheelResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saved = source["saved"];
	        this.skipped = this.convertValues(source["skipped"], QuotaFlywheelSkip);
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



	export class RateSet {
	    managementRate: number;
	    regulatoryRate: number;
	    profitRate: number;
	    taxRate: number;

	    static createFrom(source: any = {}) {
	        return new RateSet(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.managementRate = source["managementRate"];
	        this.regulatoryRate = source["regulatoryRate"];
	        this.profitRate = source["profitRate"];
	        this.taxRate = source["taxRate"];
	    }
	}
	export class RecomposeResult {
	    scanned: number;
	    withComp: number;
	    updated: number;
	    unchanged: number;
	    errors: string[];

	    static createFrom(source: any = {}) {
	        return new RecomposeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanned = source["scanned"];
	        this.withComp = source["withComp"];
	        this.updated = source["updated"];
	        this.unchanged = source["unchanged"];
	        this.errors = source["errors"];
	    }
	}
	export class Resource {
	    id: number;
	    code: string;
	    kind: string;
	    title: string;
	    spec: string;
	    unit: string;
	    basePrice: number;
	    currentPrice: number;
	    categoryPath: string;
	    source: string;
	    supplier: string;
	    region: string;
	    priceDate: string;
	    priceType: string;
	    validUntil: string;
	    lossRate: number;
	    note: string;
	    tags: string[];
	    status: string;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new Resource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.code = source["code"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.basePrice = source["basePrice"];
	        this.currentPrice = source["currentPrice"];
	        this.categoryPath = source["categoryPath"];
	        this.source = source["source"];
	        this.supplier = source["supplier"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.priceType = source["priceType"];
	        this.validUntil = source["validUntil"];
	        this.lossRate = source["lossRate"];
	        this.note = source["note"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ResourcePrice {
	    id: number;
	    resourceId: number;
	    price: number;
	    period: string;
	    region: string;
	    priceType: string;
	    source: string;
	    fetchedAt: string;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new ResourcePrice(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.resourceId = source["resourceId"];
	        this.price = source["price"];
	        this.period = source["period"];
	        this.region = source["region"];
	        this.priceType = source["priceType"];
	        this.source = source["source"];
	        this.fetchedAt = source["fetchedAt"];
	        this.note = source["note"];
	    }
	}
	export class SeedCandidate {
	    sourceName: string;
	    title: string;
	    spec: string;
	    unit: string;
	    kind: string;
	    price: number;
	    categoryPath: string;
	    source: string;
	    region: string;
	    priceDate: string;
	    priceType: string;
	    note: string;

	    static createFrom(source: any = {}) {
	        return new SeedCandidate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceName = source["sourceName"];
	        this.title = source["title"];
	        this.spec = source["spec"];
	        this.unit = source["unit"];
	        this.kind = source["kind"];
	        this.price = source["price"];
	        this.categoryPath = source["categoryPath"];
	        this.source = source["source"];
	        this.region = source["region"];
	        this.priceDate = source["priceDate"];
	        this.priceType = source["priceType"];
	        this.note = source["note"];
	    }
	}
	export class SeedPreview {
	    candidates: SeedCandidate[];
	    counts: ClassifyCount;
	    skipped: Record<string, number>;
	    total: number;

	    static createFrom(source: any = {}) {
	        return new SeedPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidates = this.convertValues(source["candidates"], SeedCandidate);
	        this.counts = this.convertValues(source["counts"], ClassifyCount);
	        this.skipped = source["skipped"];
	        this.total = source["total"];
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
	export class SeedResult {
	    created: number;
	    updated: number;
	    skipped: number;
	    counts: ClassifyCount;
	    errors: string[];

	    static createFrom(source: any = {}) {
	        return new SeedResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = source["created"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	        this.counts = this.convertValues(source["counts"], ClassifyCount);
	        this.errors = source["errors"];
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

export namespace xlsxedit {

	export class Change {
	    sheet: string;
	    cell: string;
	    before: string;
	    after: string;
	    formula?: string;

	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sheet = source["sheet"];
	        this.cell = source["cell"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.formula = source["formula"];
	    }
	}

}

