// 上下文六分类的单一事实源（审计 P1 FE1-02：inspector 此前未导出而本地重声明
// GROUPS/ROW_LABELS，与 ContextView 的 CATS/CAT_BROWSE_SHORT 双源漂移）。
// ContextView（分类面板/事件徽标）与 context/inspector（分类浏览）、context/cards
// （汇总条图例）都从这里派生；i18n 收口批把六分类语义色 CAT_COLORS 也收编至此
// （此前定义在 ContextView、cards 另抄一份色值，三处同值靠人工对齐）。
import type { DictKey } from "../../locales/en";

/** 分类 key（surface 节点 cat 与 ContextCategory 六字段同字面量）。 */
export type ContextCatKey = "system" | "tools" | "user" | "inject" | "assistant" | "tool";

export interface ContextCatDef {
  /** 分类 key（surface 节点 cat 与六分类 key 同字面量）。 */
  key: ContextCatKey;
  /** 分类组行全名 i18n 键（contextview.cat*）。 */
  labelKey: DictKey;
  /** 节点行短名 i18n 键（contextview.browse*）。 */
  browseKey: DictKey;
}

export const CONTEXT_CATS: readonly ContextCatDef[] = [
  { key: "system", labelKey: "contextview.catSystem", browseKey: "contextview.browseSystem" },
  { key: "tools", labelKey: "contextview.catTools", browseKey: "contextview.browseTools" },
  { key: "user", labelKey: "contextview.catUser", browseKey: "contextview.browseUser" },
  { key: "inject", labelKey: "contextview.catInject", browseKey: "contextview.browseInject" },
  { key: "assistant", labelKey: "contextview.catAssistant", browseKey: "contextview.browseAssistant" },
  { key: "tool", labelKey: "contextview.catTool", browseKey: "contextview.browseTool" },
] as const;

/** cat → 节点行短名 i18n 键（delta 徽标/浏览行共用）。 */
export const CONTEXT_CAT_BROWSE: Record<string, DictKey> = Object.fromEntries(
  CONTEXT_CATS.map((c) => [c.key, c.browseKey]),
);

// 六分类语义色（效果图对齐：系统蓝/工具橙/用户绿/注入紫/助手深蓝/工具青）。
export const CAT_COLORS: Record<ContextCatKey, string> = {
  system: "#3b82f6", // hex-exempt 上下文六分类语义色（可视化调色板）
  tools: "#f59e0b", // hex-exempt 上下文六分类语义色
  user: "#22c55e", // hex-exempt 上下文六分类语义色
  inject: "#a855f7", // hex-exempt 上下文六分类语义色
  assistant: "#1e40af", // hex-exempt 上下文六分类语义色
  tool: "#06b6d4", // hex-exempt 上下文六分类语义色
};
