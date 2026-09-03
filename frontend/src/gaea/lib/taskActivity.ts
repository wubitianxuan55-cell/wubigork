// taskActivity.ts — 子代理 task 卡 live 化的注入点（v4.26 对话流式重造）。
//
// Why：task 工具卡运行期间，主窗此前只有一张静态「运行中」卡——子代理的
// 实时动态（lastText/lastTool）只存在于右栏分工面板（GaeaAgentNetwork /
// GaeaSubagentRuns 轮询数据）。Codex 0.150.0 的方向是 "Report completed
// sub-agent activity on parent turns"：子代理活动要回投到主回合可见。
//
// How：渲染层与数据层解耦——本模块只持有一个 provider 槽位。主代理在 App
// 层把 AgentNetwork/SubagentRuns 轮询数据接进来：
//
//   setTaskCardActivityProvider((ref, args) => {
//     const run = runsView.runs.find(r => r.ref === ref);
//     return run ? { lastText: run.lastText, lastTool: run.lastTool, state: run.status } : undefined;
//   });
//
// 契约：
//  - fn 入参 ref：task 卡能解析到的子代理引用（sa_...；来自 args.continue_from
//    或 tool_result 的 "Subagent reference: sa_..." 行）。派发初期解析不到时
//    传空串 ""——provider 可自行回退：单个 running 分工直接回退；并行多个
//    running 时用第二参 args 的任务描述文本经 matchRunningRun 做唯一命中
//    匹配（0 或 ≥2 命中必须放弃，宁缺勿错）。返回 undefined 则卡片按现状
//    渲染，绝不因此报错。
//  - fn 入参 args（可选）：task 工具卡原始 args（JSON 字符串原样透传，形状
//    不做保证）。仅空 ref 回退匹配时消费；旧 provider 只声明 ref 一参即可，
//    多余实参被忽略，向后兼容。
//  - setTaskCardActivityProvider(null)：卸载注入（App 卸载/会话切换时调用），
//    卡片回退到默认渲染。
//  - 卡片侧以 1s tick（useNow）轮询取值，provider 内部数据更新后最迟 1s 上屏。

/** task 卡 live 预览数据（字段全部可选，缺省按现状渲染）。 */
export interface TaskCardActivity {
  /** 子代理最后一段 assistant 文本（Codex 式活动预览，单行截断）。 */
  lastText?: string;
  /** 最后一次工具调用摘要（name + 结果头）。 */
  lastTool?: string;
  /** 子代理状态（running/completed/failed 等原样透传）。 */
  state?: string;
}

/**
 * provider 签名：按子代理 ref 查活动动态；查不到返回 undefined。
 * 第二参 args（可选）为 task 卡原始 args，仅空 ref 回退匹配时消费——
 * 旧 provider 只声明 `(ref) => …` 即可：少参函数可赋给多参函数类型，
 * 运行期多余实参也被忽略，签名升级向后兼容。
 */
export type TaskCardActivityProvider = (
  ref: string,
  args?: unknown,
) => TaskCardActivity | undefined;

let provider: TaskCardActivityProvider | null = null;

/** 注入/卸载（null）活动数据 provider。重复注入以最后一次为准。 */
export function setTaskCardActivityProvider(fn: TaskCardActivityProvider | null): void {
  provider = typeof fn === "function" ? fn : null;
}

/** 卡片侧取数：未注入或 provider 抛错时返回 undefined（渲染不炸）。args 原样透传。 */
export function getTaskCardActivity(ref: string, args?: unknown): TaskCardActivity | undefined {
  if (!provider) return undefined;
  try {
    return provider(ref, args);
  } catch {
    return undefined;
  }
}

/** 是否已注入 provider：未注入（null）时 task 卡按 v4.26 之前的现状渲染。 */
export function hasTaskCardActivityProvider(): boolean {
  return provider !== null;
}

// resolveTaskRef 从 task 工具卡的 args/output 里解析子代理引用。
//  - args.continue_from：续跑子代理时模型显式传的 sa_ 引用（运行前可得）；
//  - output 的 "Subagent reference: sa_..." 行：后端 finalizeRun 追加
//    （internal/gaea/agent/subagent_store.go FormatSubagentReference），
//    仅 tool_result 到达后可得。
// 都没有 → 返回 ""（provider 收到空串，自行决定是否回退匹配）。
export function resolveTaskRef(args: string, output?: string): string {
  if (args) {
    try {
      const a = JSON.parse(args) as Record<string, unknown>;
      if (typeof a.continue_from === "string" && a.continue_from.startsWith("sa_")) {
        return a.continue_from;
      }
    } catch {
      // args 非 JSON：忽略，继续从 output 找
    }
  }
  const m = output ? output.match(/Subagent reference:\s*(sa_[A-Za-z0-9_.-]+)/) : null;
  return m ? m[1] : "";
}

// taskResultSummary 从 task 工具的 output 提取单行结果摘要（v4.26：task
// 完成卡此前 summarize 返回空串，除嵌套步数外没有任何结果信息）。取第一条
// 非空行（跳过 "Subagent reference:" 引用行），超 80 字截断；error 时由
// 卡片错误区展示，这里返回空串。
export function taskResultSummary(output?: string, error?: string): string {
  if (error || !output) return "";
  for (const line of output.split("\n")) {
    const t = line.trim();
    if (!t || t.startsWith("Subagent reference:")) continue;
    return t.length > 80 ? `${t.slice(0, 80)}…` : t;
  }
  return "";
}

// ── matchRunningRun：空 ref 并行多 running 时的文本匹配（宁缺勿错）──────
// Why：task 派发初期 args 不带 ref、output 未回，task 卡拿不到子代理引用；
// 单个 running 可整卡回退，但并行多子代理同时 running 时无法靠数量判定——
// 只能用 args 里的任务描述文本与各 run.task 做归一化匹配，唯一命中才绑定。
// 结构化最小接口：不 import types.ts，App 侧 SubagentRunView 天然满足；
// 泛型 T 保住入参数组元素的具体类型，调用方拿回的仍是 SubagentRunView。
export interface SubagentRunLike {
  ref?: string;
  status?: string;
  /** 任务摘要（transcript 首条 user 消息）；空/缺失的 run 跳过不参与匹配。 */
  task?: string;
}

// taskDescOf 从 task 工具卡的 args 里取任务描述文本（description 优先，
// prompt 回退——与 lib/tools.ts subjectOf 对 task 的取法同源）。args 可能是
// 原始 JSON 字符串（ToolCard item.args 原样透传）或已解析对象；任何形状
// 异常都返回空串（匹配自然放弃，不炸）。
function taskDescOf(args: unknown): string {
  let a: unknown = args;
  if (typeof a === "string") {
    try {
      a = JSON.parse(a);
    } catch {
      return "";
    }
  }
  if (!a || typeof a !== "object") return "";
  const o = a as Record<string, unknown>;
  for (const key of ["description", "prompt"]) {
    const v = o[key];
    if (typeof v === "string" && v.trim()) return v.trim();
  }
  return "";
}

/**
 * matchRunningRun：args 任务描述 ↔ run.task 归一化（trim）双向包含匹配。
 * Why 只有 trim：派发双方都以中文任务描述为主，大小写折叠收益极小反而
 * 扩大误配面；「前缀命中」是「包含命中」的子集，统一用双向 contains 判定。
 * 命中语义：desc 包含 run.task 或 run.task 包含 desc 任一成立（覆盖模型
 * 短摘要 vs transcript 长首条消息两个方向）。
 * 返回：恰好一个 running run 命中 → 该 run；0 或 ≥2 命中、args 无描述
 * 文本 → null（宁缺勿错：绝不把别的子代理动态安到错误卡片上）。
 */
export function matchRunningRun<T extends SubagentRunLike>(
  args: unknown,
  runs: readonly T[],
): T | null {
  const desc = taskDescOf(args);
  if (!desc) return null;
  let hit: T | null = null;
  for (const r of runs) {
    if (r.status !== "running") continue;
    const t = typeof r.task === "string" ? r.task.trim() : "";
    if (!t) continue;
    if (t.includes(desc) || desc.includes(t)) {
      if (hit) return null; // 第二个命中即歧义：立即放弃
      hit = r;
    }
  }
  return hit;
}
