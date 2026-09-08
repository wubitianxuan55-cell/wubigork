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