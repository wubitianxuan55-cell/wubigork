// 上下文六分类的单一事实源（审计 P1 FE1-02：inspector 此前未导出而本地重声明
// GROUPS/ROW_LABELS，与 ContextView 的 CATS/CAT_BROWSE_SHORT 双源漂移）。
// ContextView（分类面板/事件徽标）与 context/inspector（分类浏览）都从这里派生。
import type { DictKey } from "../../locales/en";

export interface ContextCatDef {
  /** 分类 key（surface 节点 cat 与六分类 key 同字面量）。 */
  key: "system" | "tools" | "user" | "inject" | "assistant" | "tool";
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
