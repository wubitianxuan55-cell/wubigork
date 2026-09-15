// gaea/lib/types — 纯类型声明域拆分入口（瘦身 P3 结构版1：巨文件拆分）。
// 域类型模块见 ./types/（wire/session/context/git/trajectory/agent/shell/files/
// office/subagent/deliverables/resync/capabilities/memory/knowledge/settings/
// updater/whisper/weixin/cost/search/tasks/programming + shared）。
// 本文件仅做类型 re-export：不引入任何运行时值，消费方
// `import type { X } from "./lib/types"` 行为不变。
export * from "./types/shared";
export * from "./types/wire";
export * from "./types/session";
export * from "./types/context";
export * from "./types/git";
export * from "./types/trajectory";
export * from "./types/agent";
export * from "./types/shell";
export * from "./types/files";
export * from "./types/office";
export * from "./types/subagent";
export * from "./types/deliverables";
export * from "./types/resync";
export * from "./types/capabilities";
export * from "./types/memory";
export * from "./types/knowledge";
export * from "./types/settings";
export * from "./types/updater";
export * from "./types/whisper";
export * from "./types/weixin";
export * from "./types/cost";
export * from "./types/search";
export * from "./types/tasks";
export * from "./types/programming";

// ── 7.3-1 任务收件箱：本地重述（herdsman 防环先例，不依赖 AppModels 再生；
// 字段名/顺序与后端 internal/app.TaskInboxView 的 json 标签逐一同——camelCase
// 形状由 Go 侧测试钉死，前端只按此形状消费）──
// 状态机：pending→doing→done；pending|doing→abandoned；终态不接受迁移
// （CanTransition 的前端镜像见 TaskInboxPanel 的 nextActions 纯函数）。
export type TaskInboxStatus = "pending" | "doing" | "done" | "abandoned";

export interface TaskInboxView {
  id: string; // "ti-"+12hex（后端生成，实体非重算候选）
  title: string; // NormalizeTitle 产物（≤120 rune）
  space: string; // "work" | "play"（唯一合法值，双空间隔离红线）
  status: TaskInboxStatus;
  source: string; // ctrlk | palette | voice | weixin | inbox（来源审计链）
  action?: string; // 意图动作（navigate/…；手动新建留空）
  target?: string; // 意图目标（板块 id 等）
  session?: string; // 源会话路径（审计链；V1 跳转=板块粒度）
  note?: string;
  createdAt: number; // unix 毫秒
  updatedAt: number; // unix 毫秒
}