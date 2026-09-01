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
//   setTaskCardActivityProvider((ref) => {
//     const run = runsView.runs.find(r => r.ref === ref);
//     return run ? { lastText: run.lastText, lastTool: run.lastTool, state: run.status } : undefined;
//   });
//
// 契约：
//  - fn 入参 ref：task 卡能解析到的子代理引用（sa_...；来自 args.continue_from
//    或 tool_result 的 "Subagent reference: sa_..." 行）。派发初期解析不到时
//    传空串 ""——provider 可自行回退（如返回当前唯一 running 分工的动态），
//    返回 undefined 则卡片按现状渲染，绝不因此报错。
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

/** provider 签名：按子代理 ref 查活动动态；查不到返回 undefined。 */
export type TaskCardActivityProvider = (ref: string) => TaskCardActivity | undefined;

let provider: TaskCardActivityProvider | null = null;

/** 注入/卸载（null）活动数据 provider。重复注入以最后一次为准。 */
export function setTaskCardActivityProvider(fn: TaskCardActivityProvider | null): void {
  provider = typeof fn === "function" ? fn : null;
}

/** 卡片侧取数：未注入或 provider 抛错时返回 undefined（渲染不炸）。 */
export function getTaskCardActivity(ref: string): TaskCardActivity | undefined {
  if (!provider) return undefined;
  try {
    return provider(ref);
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
