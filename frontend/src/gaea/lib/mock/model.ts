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
};

/** Herdsman 生命周期操作结果（对齐 Go HerdsmanOpResult：ok/status/message）。 */
type HerdsmanOpResultMock = { ok: boolean; status: string; message: string };

type ModelMethods = Pick<
  AppBindings,
  | "Models" | "SetModel"
  | "KeepWarmGet" | "KeepWarmSet" | "PreloadPlanGet" | "PreloadPlanSet"
  | "ModelSwitchEstimate" | "Balance"
> & LegacyModelMethods & LegacyHerdsmanMethods;

export function buildModel(s: MakeMockState): ModelMethods {
  return {
    async Models() {
      // 契约对齐 Go GaeaModels（internal/app/gaea_ui_meta.go）：ref = 引擎ID + "/" + 模型，
      // provider = 引擎 ID，model 为空回退 "(默认)"，current = 活跃引擎；引擎默认模型取
      // internal/modelengine/engine.go 预置值（deepseek→deepseek-v4-pro、xai→grok-4.20、
      // herdsman/ollama 默认模型为空 → "(默认)"）。
      return [
        { ref: "xai/grok-4.20", provider: "xai", model: "grok-4.20", current: false },
        { ref: "ollama/(默认)", provider: "ollama", model: "(默认)", current: false },
        { ref: "herdsman/(默认)", provider: "herdsman", model: "(默认)", current: false },
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
    async ModelSwitchEstimate(engineID: string): Promise<ModelSwitchEstimate> {
      // 契约对齐 Go GaeaModelSwitchEstimate（internal/app/gaea_schedule.go）：
      // 非 herdsman 引擎恒为 hot（引擎常驻，waitSeconds=1）；herdsman 演示冷启动。
      if (engineID !== "herdsman") {
        return {
          engine: engineID,
          model: "",
          status: "hot",
          waitSeconds: 1,
          note: "引擎已就绪",
        };
      }
      return {
        engine: engineID,
        model: "",
        status: "cold",
        waitSeconds: 20,
        note: "本地模型需冷启动，实测约 15-20 秒（mock）",
      };
    },
    async Balance() {
      // Mirror the active mock provider: deepseek-flash carries a balance_url.
      const p = s.settings.providers.find((x) => x.name === s.settings.defaultModel);
      if (!p?.balanceUrl) return { available: false, display: "" };
      return { available: true, display: "¥128.50" };
    },

    // ── Herdsman 方法族（B 线欠账，v4.55 走查）─────────────────
    // 浏览器 ?mock=1 下原缺这批绑定，模型中心 load() 抛「... is not a
    // function」级 TypeError：调度段呈现「当前运行中的本地模型不可用」、
    // 模型库段呈现「模型目录不可用」错误横幅。契约取中性空态（空模型列表 +
    // 计数 0，不编造逼真模型名）：调度段读 running=0 →「0 个」、无 error →
    // 不再报错；模型库段空列表 → 「没有匹配的模型」空态；测评段 installed
    // 过滤为空。生命周期操作诚实返回 ok:false（mock 无 herdsman.exe 后端，
    // 假成功会误导走查；消费方 runOp 对 !ok 弹 message.error）。
    async HerdsmanModelCatalog() {
      return { models: [], total: 0, installed: 0, running: 0, source: "mock" };
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
  };
}
