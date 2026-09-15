// mock/model.ts — 模型中心域（T6-10.1 拆分自 lib/mock.ts，方法体零改动；
// Models/ModelSwitchEstimate 契约对齐 Go，T6-10.4）。
// 注意 keepWarm/preloadPlan/engineFailover 会被整体重绑，必须经 s.keepWarm /
// s.preloadPlan / s.engineFailover 读写（setter 经 state.setKeepWarm 等
// 重绑），不能解构快照。
import type { AppBindings } from "../bridge";
import type { ModelSwitchEstimate } from "../types";
import type { MakeMockState } from "./state";

// GetEngineFailover/SetEngineFailover 是 legacy 绑定面（挂在后端 ModelB 门面，
// Go 同名，经 window.go.app.App 兼容代理按方法名路由），按 bridge.ts 的
// LegacySurfaceNames 约定不进 AppBindings——这里用局部方法类型补足，避免牵动
// AppBindings 双向类型锁（bindingNames 已含两名，drift 校验零变更）。
type LegacyModelMethods = {
  GetEngineFailover(): Promise<boolean>;
  SetEngineFailover(enabled: boolean): Promise<void>;
};

// Herdsman 方法族（B 线欠账）：模型中心「模型库/本地调度/受控测评」段经
// api/engines.ts（getHerdsmanCatalog 等）调用下列同名绑定。与上方故障转移
// 两名同为 legacy 绑定面（bridge.ts LegacySurfaceNames 显式排除，AppBindings
// 认领待后端合入后收口），故同样用局部类型补足、不牵动 AppBindings 双向锁。
// 返回形状按消费方解构字段局部重述（对齐 api/engines.ts 的 HerdsmanCatalog/
// HerdsmanLaunchPreset/HerdsmanModelStats/HerdsmanOpResult；api 层 import 会成
// api↔gaea 环），字段漂移由 mock-contract-herdsman 契约测试兜底。
type LegacyHerdsmanMethods = {
  HerdsmanModelCatalog(): Promise<{
    models: unknown[];
    total: number;
    installed: number;
    running: number;
    source: string;
    error?: string;
  }>;
  HerdsmanLaunchPresets(): Promise<unknown[]>;
  HerdsmanModelStats(): Promise<{
    total: number;
    since: string;
    per_model: unknown[];
    source: string;
    error?: string;
  }>;
  HerdsmanModelStart(model: string): Promise<HerdsmanOpResultMock>;
  HerdsmanModelStop(model: string): Promise<HerdsmanOpResultMock>;
  HerdsmanModelDownload(model: string): Promise<HerdsmanOpResultMock>;
  HerdsmanModelUninstall(model: string): Promise<HerdsmanOpResultMock>;
  // 模型调用统计汇总（模型中心「详细统计」段）：同属 legacy 绑定面。
  // 返回形状对齐 api/engines.ts ModelStatsSummary 消费方解构字段（api 层
  // import 会成 api↔gaea 环，故局部重述；字段漂移由契约测试兜底）。
  GetModelCallStats(): Promise<{
    total_calls: number;
    success_calls: number;
    fail_calls: number;
    total_tokens: number;
    input_tokens: number;
    output_tokens: number;
    total_duration_ms: number;
    avg_duration_ms: number;
    total_cost: number;
    trend: unknown[];
    per_model: unknown[];
  }>;
  // 引擎列表（模型中心引擎管理段 useEngineState / 设置·绘梦面板 getEngines）：
  // 同属 legacy 绑定面；空列表走引擎管理既有空态，不再报
  // 「GetEngines is not a function」横幅。
  GetEngines(): Promise<unknown[]>;
};

/** Herdsman 生命周期操作结果（对齐 Go HerdsmanOpResult：ok/status/message）。 */
type HerdsmanOpResultMock = { ok: boolean; status: string; message: string };

// 受控测评五名 + 资源遥测（B 线欠账，v4.56 走查）：BenchmarkSection 经
// api/engines.ts（getBenchmarkList 等）调用 GaeaBenchmark*；ResourceMonitor /
// MainLayout / ModuleLauncher 调用 GetModelMonitor。六名与上方 Herdsman 族同为
// legacy 绑定面（bridge.ts LegacySurfaceNames 显式排除，AppBindings 不消费），
// 故同样局部重述返回形状、不牵动 AppBindings 双向类型锁。形状对齐
// api/engines.ts 的 BenchmarkRunSummary/BenchmarkRunDetail/StreamProbeResult 与
// pages/modelcenter/resource.ts 的 ResourceMonitorData（api/pages 层 import 会成
// api↔gaea 环）；字段漂移由 mock-contract-benchmark 契约测试兜底。
type LegacyBenchmarkMethods = {
  GaeaBenchmarkList(): Promise<unknown[]>;
  // Go 契约为 (string, error)：失败呈现 rejected promise（消费方 catch 弹错），
  // 无 ok:false 形状可用。
  GaeaBenchmarkStart(req: unknown): Promise<string>;
  GaeaBenchmarkDetail(id: string): Promise<BenchmarkRunDetailMock>;
  GaeaBenchmarkExport(id: string, dir: string): Promise<string>;
  GaeaBenchmarkStreamProbe(model: string): Promise<StreamProbeResultMock>;
  GetModelMonitor(): Promise<ResourceMonitorDataMock>;
};

/** 测评运行明细（对齐 api/engines.ts BenchmarkRunDetail 消费方解构字段）。 */
type BenchmarkRunDetailMock = {
  id: string;
  created_at: string;
  finished_at?: string;
  status: string;
  config: Record<string, unknown>;
  summary: {
    total_cases: number;
    succeeded: number;
    failed: number;
    canceled: number;
    avg_duration_ms: number;
    avg_ttft_ms: number;
    avg_tps: number;
  };
  cases: unknown[];
};

/** 流式探针结果（对齐 api/engines.ts StreamProbeResult：ok/error 键驱动红标）。 */
type StreamProbeResultMock = {
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
};

/** 资源遥测（对齐 pages/modelcenter/resource.ts ResourceMonitorData 三键）。 */
type ResourceMonitorDataMock = {
  engines?: unknown[];
  stats?: {
    cpu?: number;
    memTotal?: number;
    memUsed?: number;
    gpuName?: string;
    gpuUsage?: number;
    vramUsed?: number;
    vramTotal?: number;
  };
  comfyRunning?: boolean;
};

type ModelMethods = Pick<
  AppBindings,
  | "Models" | "SetModel"
  | "KeepWarmGet" | "KeepWarmSet" | "PreloadPlanGet" | "PreloadPlanSet"
  | "ModelSwitchEstimate" | "Balance"
  // v4.171 批次一 legacy 直调转正（ModelB 门面）：功能级模型绑定写 + 活跃模型读。
  | "SetFeatureModel" | "SetFeatureModelEnabled" | "GetActiveModel"
  // 路由学习四名（阶段七 7.1-2，ModelB 门面）：账本/建议/采纳/忽略。
  | "GaeaRouteLedger" | "GaeaRouteSuggestions" | "GaeaRouteSuggestionApply" | "GaeaRouteSuggestionIgnore"
> & LegacyModelMethods & LegacyHerdsmanMethods & LegacyBenchmarkMethods &
  LegacyModelHubMethods & LegacyOpencodeKeyMethods;

// HerdsmanCatalogModel 本地视图（对齐 internal/app/herdsman_catalog.go 的 json
// 字段；mock 层不 import Go 侧，字段漂移由 mock-contract-herdsman 兜底）。
type HerdsmanCatalogModelMock = {
  name: string;
  display_name: string;
  type: string;
  runtime?: string;
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
};

// 走查样例（v4.100 T1 画室「模型目录」）：6 条 = 5 条图像相关（生图/改图/视频/
// 识图各有着落）+ 1 条纯 LLM（验证画室视图按能力字段过滤、LLM 不混入）。
// hint 文案取 Go herdsmanModelHint 的真实映射输出；档位/成本/上下文目录本就
// 不携带，样例不编造（未知留空）。
const HERDSMAN_CATALOG_SAMPLE: HerdsmanCatalogModelMock[] = [
  {
    name: "z-image-turbo",
    display_name: "Z-Image Turbo",
    type: "image",
    capabilities: ["text-to-image", "image-to-image"],
    installed: true,
    running: true,
    status: "installed",
    run_status: "running",
    file_size: 20401094656,
    hint: "本地文生图（19GB）：绘梦板块 herdsman 后端",
  },
  {
    name: "flux",
    display_name: "FLUX",
    type: "image",
    capabilities: ["text-to-image"],
    installed: true,
    running: false,
    status: "installed",
    run_status: "stopped",
    file_size: 23843624960,
    hint: "本地文生图（19GB）：绘梦板块 herdsman 后端",
  },
  {
    name: "ltx-video",
    display_name: "LTX Video",
    type: "video",
    capabilities: ["text-to-video", "image-to-video"],
    installed: false,
    running: false,
    status: "uninstalled",
    run_status: "stopped",
  },
  {
    name: "ovis2-8b",
    display_name: "Ovis2 8B",
    type: "vision",
    capabilities: ["image-understanding", "ocr"],
    installed: true,
    running: false,
    status: "installed",
    run_status: "stopped",
    file_size: 17179869184,
  },
  {
    name: "paddleocr",
    display_name: "PaddleOCR",
    type: "ocr",
    capabilities: ["ocr"],
    installed: true,
    running: false,
    status: "installed",
    run_status: "stopped",
    hint: "快速 OCR（约 90ms）：中文混合场景有错字，失败自动回退 OvisOCR2",
  },
  {
    // 非图像条目：画室目录视图应过滤掉（仅模型中心「模型库」可见）。
    name: "qwen3-14b",
    display_name: "Qwen3 14B",
    type: "llm",
    capabilities: ["chat", "thinking"],
    installed: true,
    running: false,
    status: "installed",
    run_status: "stopped",
    quantization: "Q4_K_M",
    parameter_count: 14700000000,
    file_size: 9126805504,
    llama_cpp_variants: ["Q4_K_M", "Q8_0"],
  },
];

// Model Hub（Unsloth）本地引擎（6cd891df 落库，legacy 直调面同款局部类型补足）。
// 语义：Set 成功=浏览器内存态联动（makeMockApp 每次新建即回未配置，诚实无持久化）；
// Start 无本地引擎进程，诚实失败（rejected promise）——照 GetProgrammingWebStatus
// 「浏览器 mock 恒为未运行」先例。
type LegacyModelHubMethods = {
  SetModelHubKey(apiKey: string): Promise<void>;
  GetModelHubKeyStatus(): Promise<{ configured: boolean; masked: string }>;
  StartModelHubModel(modelID: string): Promise<void>;
};

// OpenCode Go/Zen Key（v4.4x 既有的 legacy 直调族，modelcenter 引擎管理同屏消费；
// 补 mock 消「引擎管理运行态报错」同族欠账）。语义同上：内存态联动、无持久化。
type LegacyOpencodeKeyMethods = {
  SetOpencodeGoKey(apiKey: string): Promise<void>;
  GetOpencodeGoKeyStatus(): Promise<{ configured: boolean; masked: string }>;
  SetOpencodeZenKey(apiKey: string): Promise<void>;
  GetOpencodeZenKeyStatus(): Promise<{ configured: boolean; masked: string }>;
};

// 脱敏口径对齐 internal/app/model_engine_handler.go maskKeyStatus：
// >8 位=前 4 + **** + 后 4，其余 ****，空=未配置。
function maskMockKey(key: string): { configured: boolean; masked: string } {
  if (!key) return { configured: false, masked: "" };
  if (key.length <= 8) return { configured: true, masked: "****" };
  return { configured: true, masked: key.slice(0, 4) + "****" + key.slice(-4) };
}

export function buildModel(s: MakeMockState): ModelMethods {
  // Key 内存态（makeMockApp 每次新建=重置；?mock=1 刷新即回未配置，诚实语义）
  let modelHubKey = "";
  let opencodeGoKey = "";
  let opencodeZenKey = "";
  return {
    async Models() {
      // 契约对齐 Go GaeaModels（internal/app/gaea_ui_meta.go，v4.126 刀1）：
      // 按引擎展开全部对话模型（本地引擎置前、local/status/label 透传），空清单
      // 引擎回退 "(默认)" 一条。ref = 引擎ID + "/" + 模型，current = 活跃引擎
      // 且为引擎默认模型。演示两个 ollama 模型（一加载一未加载）与 modelhub
      // stopped 条目，供切换器分组/徽标/预估确认走查。
      return [
        { ref: "ollama/qwen3:8b", provider: "ollama", model: "qwen3:8b", current: false, local: true, status: "running" },
        { ref: "ollama/qwen3:14b", provider: "ollama", model: "qwen3:14b", current: false, local: true },
        { ref: "modelhub/ollama-manifest:tiny", provider: "modelhub", model: "ollama-manifest:tiny", current: false, local: true, status: "stopped", label: "Tinyrick Q6" },
        { ref: "herdsman/(默认)", provider: "herdsman", model: "(默认)", current: false, local: true },
        { ref: "xai/grok-4.20", provider: "xai", model: "grok-4.20", current: false },
        { ref: "deepseek/deepseek-v4-pro", provider: "deepseek", model: "deepseek-v4-pro", current: true },
      ];
    },
    async SetModel() {},
    async KeepWarmGet() {
      return s.keepWarm;
    },
    async KeepWarmSet(enabled: boolean) {
      s.setKeepWarm(enabled);
    },
    async PreloadPlanGet() {
      return s.preloadPlan;
    },
    async PreloadPlanSet(enabled: boolean) {
      s.setPreloadPlan(enabled);
    },
    // 故障转移开关（C 刀）：Get 读 / Set 写回 mock state（默认 false，会话内
    // 一致）。与 KeepWarmGet/Set 同构；真实后端持久化与失败换引擎重试在
    // ModelB 门面（并行线），mock 只需锁「可读布尔 + 写回一致」契约。
    async GetEngineFailover() {
      return s.engineFailover;
    },
    async SetEngineFailover(enabled: boolean) {
      s.setEngineFailover(enabled);
    },
    async ModelSwitchEstimate(engineID: string, model = ""): Promise<ModelSwitchEstimate> {
      // 契约对齐 Go GaeaModelSwitchEstimate（internal/app/gaea_schedule.go，
      // v4.126 刀2）：云端引擎恒 hot；本地引擎按目标模型演示——ollama 未在
      // 运行列表 → cold（不假报秒数），herdsman → cold 20s（实测口径），
      // modelhub 未加载 → cold 引导一键加载。
      if (engineID !== "herdsman" && engineID !== "ollama" && engineID !== "modelhub") {
        return { engine: engineID, model, status: "hot", waitSeconds: 1, note: "引擎已就绪" };
      }
      if (engineID === "ollama" && model.endsWith(":8b")) {
        return { engine: engineID, model, status: "hot", waitSeconds: 1, note: "模型已加载，可直接切换" };
      }
      const notes: Record<string, string> = {
        herdsman: "本地模型需冷启动，实测约 15-20 秒（mock）",
        ollama: "模型已安装未加载，切换后首次对话需等待加载（mock）",
        modelhub: "模型未加载，切换后首次对话需等待加载（mock）",
      };
      return {
        engine: engineID,
        model,
        status: "cold",
        waitSeconds: engineID === "herdsman" ? 20 : 0,
        note: notes[engineID] ?? "",
      };
    },
    async Balance() {
      // Mirror the active mock provider: deepseek-flash carries a balance_url.
      const p = s.settings.providers.find((x) => x.name === s.settings.defaultModel);
      if (!p?.balanceUrl) return { available: false, display: "" };
      return { available: true, display: "¥128.50" };
    },
    // ── v4.171 批次一 legacy 直调转正（Go ModelB）──────────────────────
    async SetFeatureModel(_feature: string, _engineID: string, _modelName: string) {
      // mock: no-op——浏览器开发无持久化功能级绑定（真实实现写 config 并生效；
      // 与 feature-model-changed 事件的联动在消费侧，mock 不编造事件）。
    },
    async SetFeatureModelEnabled(_feature: string, _enabled: boolean) {
      // mock: no-op——同上，无配置可写。
    },
    async GetActiveModel() {
      // 对齐 Go GetActiveModel 未初始化（engineMgr/client 为空）返回空串；
      // mock 无活跃引擎，不编造模型名（api/settings.ts getActiveModel
      // 与 useEngineState 均有空串兜底）。
      return "";
    },

    // ── Herdsman 方法族（B 线欠账，v4.55 走查）─────────────────
    // 浏览器 ?mock=1 下原缺这批绑定，模型中心 load() 抛「... is not a
    // function」级 TypeError。v4.100 T1 起 HerdsmanModelCatalog 从「中性空态」
    // 改为带图像模型走查样例（画室「模型目录」tab 需要样例才能走查分层展示；
    // 授权刀：mock 样例数据追加，方法形状/语义不变——计数仍从列表派生、无
    // error 键，SchedulingSection/模型库段不再报错的不变量由契约测试继续锁定，
    // 断言已从「空态」改为「一致性」，见 mock-contract-herdsman.test.ts）。
    // 样例口径：模型名取项目既有口径（z-image-turbo/flux/LTX/ovis/paddleocr，
    // 与 mock/imagehub.ts 资产样例的 model 字段同源），字段逐一对齐 Go
    // HerdsmanCatalogModel（status/run_status 取真实 CLI 词汇 installed/
    // uninstalled/running/stopped）；目录本就不携带档位/成本/上下文元数据，
    // 样例同样不编造（诚实留空）。模型中心「模型库」段在 mock 下会显示这些
    // 样例（原为空态）——名称与画室视图同源，走查两侧一致。
    async HerdsmanModelCatalog() {
      // 计数从列表派生（与 Go HerdsmanModelCatalog 同一不变量），无 error 键。
      const models = HERDSMAN_CATALOG_SAMPLE;
      return {
        models,
        total: models.length,
        installed: models.filter((m) => m.installed).length,
        running: models.filter((m) => m.running).length,
        source: "mock",
      };
    },
    async HerdsmanLaunchPresets() {
      return [];
    },
    async HerdsmanModelStats() {
      return { total: 0, since: "", per_model: [], source: "mock" };
    },
    async HerdsmanModelStart(_model: string): Promise<HerdsmanOpResultMock> {
      return { ok: false, status: "unavailable", message: "浏览器开发环境无 Herdsman 后端，操作不可用（mock）" };
    },
    async HerdsmanModelStop(_model: string): Promise<HerdsmanOpResultMock> {
      return { ok: false, status: "unavailable", message: "浏览器开发环境无 Herdsman 后端，操作不可用（mock）" };
    },
    async HerdsmanModelDownload(_model: string): Promise<HerdsmanOpResultMock> {
      return { ok: false, status: "unavailable", message: "浏览器开发环境无 Herdsman 后端，操作不可用（mock）" };
    },
    async HerdsmanModelUninstall(_model: string): Promise<HerdsmanOpResultMock> {
      return { ok: false, status: "unavailable", message: "浏览器开发环境无 Herdsman 后端，操作不可用（mock）" };
    },
    // 调用统计空聚合（v4.56 走查）：total_calls=0 让 StatsSection 走既有
    // 「暂无数据」空态，不再渲染「统计加载失败：… is not a function」横幅。
    async GetModelCallStats() {
      return {
        total_calls: 0,
        success_calls: 0,
        fail_calls: 0,
        total_tokens: 0,
        input_tokens: 0,
        output_tokens: 0,
        total_duration_ms: 0,
        avg_duration_ms: 0,
        total_cost: 0,
        trend: [],
        per_model: [],
      };
    },
    // ── 路由学习四名（阶段七 7.1-2；归因 tab 走查用：给一组有数据的账本
    // + 一条本地引擎建议样本，便于 ?mock=1 目检建议卡/分组表/未标记桶）──
    async GaeaRouteLedger() {
      return {
        generated_at: "2026-09-15T12:00:00+08:00",
        total: { calls: 120, tokens_in: 80000, tokens_out: 30000, cost_cny: 0.42, avg_ms: 2400, success_rate: 0.95 },
        features: [
          { feature: "chat", engine_id: "xai", model: "grok-4", calls: 80, tokens_in: 60000, tokens_out: 22000, cost_cny: 0.38, avg_ms: 2600, success_rate: 0.94 },
          { feature: "chat", engine_id: "deepseek", model: "deepseek-chat", calls: 20, tokens_in: 10000, tokens_out: 4000, cost_cny: 0.02, avg_ms: 1800, success_rate: 0.95 },
          { feature: "novel", engine_id: "glm", model: "glm-5", calls: 20, tokens_in: 10000, tokens_out: 4000, cost_cny: 0.02, avg_ms: 2200, success_rate: 0.9 },
          { feature: "", engine_id: "xai", model: "grok-4", calls: 5, tokens_in: 3000, tokens_out: 1000, cost_cny: 0.01, avg_ms: 2000, success_rate: 1 },
        ],
      };
    },
    async GaeaRouteSuggestions() {
      return {
        generated_at: "2026-09-15T12:00:00+08:00",
        suggestions: [
          {
            id: "chat|xai>grok-4>herdsman>qwen3:32b",
            feature: "chat",
            reason: "同功能下 herdsman/qwen3:32b 评分领先当前 0.31（成功率 0.97 vs 0.94，单次成本 0.0000 vs 0.0047 CNY）",
            evidence: "候选：样本 64 次、均时长 6800ms、本地引擎（零成本）；当前：样本 80 次、均时长 2600ms",
            from: { engine_id: "xai", model: "grok-4" },
            to: { engine_id: "herdsman", model: "qwen3:32b", is_local: true, success_rate: 0.97, cost_cny: 0 },
            score_gap: 0.31,
          },
        ],
      };
    },
    async GaeaRouteSuggestionApply(_id: string): Promise<void> {
      // mock 环境无真实引擎管理：确认语义即可（真实链路=SetFeatureModel 改绑）
    },
    async GaeaRouteSuggestionIgnore(_id: string): Promise<void> {},
    // 引擎列表空态（v4.57 走查抓到的既有缺口）：引擎管理段空表，
    // 不再报「GetEngines is not a function」横幅。
    async GetEngines() {
      return [];
    },

    // ── 受控测评五名 + 资源遥测（B 线欠账，v4.56 走查）──────────
    // 查询类（List/Detail）中性空态：空列表/零计数、绝不 throw——BenchmarkSection
    // load() 的 Promise.all 里 list 一旦 rejected 会 message.warning「无法连接
    // Herdsman 测评接口」，空列表则走既有「暂无测评记录」EmptyState。
    async GaeaBenchmarkList() {
      return [];
    },
    async GaeaBenchmarkDetail(id: string): Promise<BenchmarkRunDetailMock> {
      return {
        id,
        created_at: "",
        finished_at: "",
        // mock 无运行记录；非 succeeded/running/pending/failed 的字串让消费方
        // statusTone 落 neutral 灰标，不误报成功/失败。
        status: "not_found",
        config: {
          model_names: [],
          variants: [],
          context_sizes: [],
          cache_reuse_mode: "",
          warmup_count: 0,
          repeat_count: 0,
          concurrency: 0,
          request: {
            user_prompt: "",
            temperature: 0,
            top_p: 0,
            top_k: 0,
            repeat_penalty: 0,
            max_tokens: 0,
            stream: false,
            timeout_seconds: 0,
          },
        },
        summary: { total_cases: 0, succeeded: 0, failed: 0, canceled: 0, avg_duration_ms: 0, avg_ttft_ms: 0, avg_tps: 0 },
        cases: [],
      };
    },
    // 动作类诚实失败：Go 契约为 (string, error)，Wails 失败即 rejected promise
    //（经 bridge invoke 归一为 BridgeError，message 保留原文），消费方 catch 后
    // message.error「发起测评失败/导出失败」——与 Herdsman ok:false 同「不假
    // 成功」原则（假成功会让走查误以为测评已发起）。
    async GaeaBenchmarkStart(_req: unknown): Promise<string> {
      throw new Error("浏览器开发环境无 Herdsman 测评后端，无法发起受控测评（mock）");
    },
    async GaeaBenchmarkExport(_id: string, _dir: string): Promise<string> {
      throw new Error("浏览器开发环境无 Herdsman 测评后端，报告不可导出（mock）");
    },
    // 流式探针不 throw：StreamProbeResult 自带 ok/error 键，消费方 probeBadge 对
    // !ok 渲染「✗ {error}」红标——诚实失败形状跟随 Go 契约（与 Start/Export 的
    // rejected promise 语义等价，只是形状不同）。
    async GaeaBenchmarkStreamProbe(model: string): Promise<StreamProbeResultMock> {
      return {
        model,
        ok: false,
        ttft_ms: 0,
        chunks: 0,
        tokens: 0,
        duration_ms: 0,
        max_gap_ms: 0,
        avg_gap_ms: 0,
        completed: false,
        interrupted: false,
        error: "浏览器开发环境无本地模型后端，流式探针不可用（mock）",
      };
    },
    // 资源遥测中性空态：无本地引擎 / 无占用读数 / ComfyUI 未运行——ResourceMonitor
    // 呈现「无本地引擎」chip + 0% 条，不编造真实 CPU/显存数值。
    async GetModelMonitor(): Promise<ResourceMonitorDataMock> {
      return { engines: [], stats: {}, comfyRunning: false };
    },
    // ── Model Hub（Unsloth）与 OpenCode Key：浏览器内存态联动，无持久化 ──
    async SetModelHubKey(apiKey: string): Promise<void> {
      if (!apiKey || !apiKey.trim()) throw new Error("API Key 不能为空");
      modelHubKey = apiKey.trim();
    },
    async GetModelHubKeyStatus() {
      return maskMockKey(modelHubKey);
    },
    async StartModelHubModel(_modelID: string): Promise<void> {
      throw new Error("浏览器 dev mock 无本地 Unsloth Studio 进程，模型加载不可用（mock）");
    },
    async SetOpencodeGoKey(apiKey: string): Promise<void> {
      if (!apiKey || !apiKey.trim()) throw new Error("API Key 不能为空");
      opencodeGoKey = apiKey.trim();
    },
    async GetOpencodeGoKeyStatus() {
      return maskMockKey(opencodeGoKey);
    },
    async SetOpencodeZenKey(apiKey: string): Promise<void> {
      if (!apiKey || !apiKey.trim()) throw new Error("API Key 不能为空");
      opencodeZenKey = apiKey.trim();
    },
    async GetOpencodeZenKeyStatus() {
      return maskMockKey(opencodeZenKey);
    },
  };
}
