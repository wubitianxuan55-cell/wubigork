// KnowledgePanel 拆分——共享常量（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件整体迁出，逐字节照搬；行为零变化。

export const CATEGORIES = ["all", "规范标准", "工程案例", "经验总结", "材料工艺", "法规政策", "调查报告", "设计方案", "其他"];
export const PHASES = ["all", "调查", "设计", "施工", "验收", "运维", "全程"];
export const STATUSES = ["all", "现行", "已归档", "常用", "草稿"];
/** 批量操作条「改状态…」的目标集合（与 STATUSES 顺序不同：不含 all）。 */
export const BATCH_STATUS_OPTIONS = ["现行", "草稿", "常用", "已归档"];

// 可见焦点环（v3 规范：--gaea-glow 描边）。全局样式会把 :focus-visible 的
// outline 置 none，这里用 Tailwind 工具类显式恢复，保证键盘可达。
export const FOCUS_RING = "focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-[var(--gaea-glow)]";