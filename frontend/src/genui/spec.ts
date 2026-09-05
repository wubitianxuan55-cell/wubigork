// GenUI spec language — gaea 蒸馏 dsh-genui 的声明式组件语言。
//
// 模型在回答里输出 ```genui / ```dsh-ui 围栏，围栏内 JSON 规格经 guard.ts
// 校验/修复后由 GenuiBlock 渲染为白名单组件。规格语言只在本文件声明，
// guard 与渲染器共享同一份类型与上限常量。

/** 统一资源上限（单一权威常量；Go 侧 genui_validate 的注释以本表为准）。 */
export const GENUI_LIMITS = {
  maxDepth: 8,
  maxNodes: 200,
  maxString: 2000,
  maxCode: 12000,
  maxFenceBody: 65536,
  maxGridCols: 12,
  maxTabs: 12,
  maxAccordionItems: 24,
  maxListItems: 50,
  maxOptions: 50,
  maxTableRows: 50,
  maxTableCols: 12,
  maxChartPoints: 60,
  maxTimelineItems: 24,
  maxSteps: 24,
  maxKeyValuePairs: 24,
  maxQuizOptions: 8,
  maxDiffs: 20,
  maxAction: 64,
} as const;

export type Tone =
  | "success"
  | "warn"
  | "danger"
  | "accent"
  | "info"
  | "error";

export type TextSize = "h1" | "h2" | "h3" | "body" | "muted" | "caption";
export type ButtonTone = "primary" | "danger" | "success" | "ghost";
export type InputType = "text" | "email" | "password";
export type ChartKind = "bars" | "line" | "donut";

/** 布局 */
export interface GenuiText {
  type: "text";
  content: string;
  size?: TextSize;
  center?: boolean;
}
export interface GenuiRow {
  type: "row";
  items: GenuiNode[];
  wrap?: boolean;
  gap?: number;
}
export interface GenuiCol {
  type: "col";
  items: GenuiNode[];
  wrap?: boolean;
  gap?: number;
}
export interface GenuiGrid {
  type: "grid";
  cols: number;
  items: GenuiNode[];
}
export interface GenuiCard {
  type: "card";
  title?: string;
  items: GenuiNode[];
}
export interface GenuiDivider {
  type: "divider";
}
export interface GenuiSpacer {
  type: "spacer";
}

/** 展示 */
export interface GenuiStat {
  type: "stat";
  label: string;
  value: string;
  delta?: string;
}
export interface GenuiBadge {
  type: "badge";
  label: string;
  tone?: "success" | "warn" | "danger" | "accent";
  icon?: string;
}
export interface GenuiProgress {
  type: "progress";
  label?: string;
  value: number;
  valueLabel?: string;
}
export interface GenuiKeyValue {
  type: "keyvalue";
  pairs: { key: string; value: string }[];
}
export interface GenuiList {
  type: "list";
  items: (string | GenuiNode)[];
}
export interface GenuiTable {
  type: "table";
  columns: string[];
  rows: (string | number)[][];
}
export interface GenuiTimeline {
  type: "timeline";
  items: { title: string; desc?: string; time?: string }[];
}
export interface GenuiCallout {
  type: "callout";
  tone?: "info" | "success" | "warning" | "error";
  title?: string;
  content: string;
}
export interface GenuiSteps {
  type: "steps";
  current?: number;
  steps: { title: string; desc?: string }[];
}
export interface GenuiAvatar {
  type: "avatar";
  name: string;
  color?: string;
}
export interface GenuiCopy {
  type: "copy";
  label?: string;
  text: string;
}

/** 轻图表（手写 SVG，不引 ECharts） */
export interface GenuiChart {
  type: "chart";
  kind?: ChartKind;
  data: { label: string; value: number; color?: string }[];
  series?: { label: string; color?: string; data: { label: string; value: number; color?: string }[] }[];
}

/** 代码展示 */
export interface GenuiCode {
  type: "code";
  lang?: string;
  code: string;
}
export interface GenuiJson {
  type: "json";
  value: unknown;
}
export interface GenuiDiff {
  type: "diff";
  diffs: { path: string; oldText?: string | null; newText: string }[];
}

/** 交互 */
export interface GenuiButton {
  type: "button";
  label: string;
  tone?: ButtonTone;
  full?: boolean;
  small?: boolean;
  icon?: string;
  action?: string;
}
export interface GenuiInput {
  type: "input";
  label?: string;
  placeholder?: string;
  value?: string;
  inputType?: InputType;
  action?: string;
  id?: string;
}
export interface GenuiSelect {
  type: "select";
  label?: string;
  options: string[];
  selected?: number;
  action?: string;
  id?: string;
}
export interface GenuiCheckbox {
  type: "checkbox";
  label: string;
  checked?: boolean;
  action?: string;
}
export interface GenuiSwitch {
  type: "switch";
  label: string;
  checked?: boolean;
  action?: string;
}
export interface GenuiRadio {
  type: "radio";
  label?: string;
  options: string[];
  selected?: number;
  action?: string;
  group?: string;
  answer?: string | number;
  explanation?: string;
}
export interface GenuiSlider {
  type: "slider";
  label?: string;
  min?: number;
  max?: number;
  step?: number;
  value?: number;
  action?: string;
  id?: string;
}
export interface GenuiTextarea {
  type: "textarea";
  label?: string;
  placeholder?: string;
  rows?: number;
  value?: string;
  action?: string;
  id?: string;
}
export interface GenuiSubmit {
  type: "submit";
  label?: string;
  action?: string;
  groups?: string[];
  resetAction?: string;
}
export interface GenuiTabs {
  type: "tabs";
  tabs: { label: string; items: GenuiNode[] }[];
}
export interface GenuiAccordion {
  type: "accordion";
  items: { title: string; items: GenuiNode[] }[];
}
export interface GenuiQuiz {
  type: "quiz";
  question: string;
  options: { label: string; correct?: boolean; feedback?: string }[];
  explanation?: string;
  id?: string;
  action?: string;
}

export type GenuiNode =
  | GenuiText
  | GenuiRow
  | GenuiCol
  | GenuiGrid
  | GenuiCard
  | GenuiDivider
  | GenuiSpacer
  | GenuiStat
  | GenuiBadge
  | GenuiProgress
  | GenuiKeyValue
  | GenuiList
  | GenuiTable
  | GenuiTimeline
  | GenuiCallout
  | GenuiSteps
  | GenuiAvatar
  | GenuiCopy
  | GenuiChart
  | GenuiCode
  | GenuiJson
  | GenuiDiff
  | GenuiButton
  | GenuiInput
  | GenuiSelect
  | GenuiCheckbox
  | GenuiSwitch
  | GenuiRadio
  | GenuiSlider
  | GenuiTextarea
  | GenuiSubmit
  | GenuiTabs
  | GenuiAccordion
  | GenuiQuiz;

export interface GenuiSpec {
  /** 块标题（banner）。 */
  title?: string;
  /** 根 items 垂直间距。 */
  gap?: number;
  /** panel:true → 投递办公会话面板（消息流内只显示占位 chip）。 */
  panel?: boolean;
  /** 仅 panel 有效：把本规格并入现有面板（同 label tab 追加/新 tab 增加/items 尾接）。 */
  append?: boolean;
  items: GenuiNode[];
}

/** 全部合法 type（guard 与 validate 共用）。 */
export const GENUI_NODE_TYPES: ReadonlySet<string> = new Set([
  "text", "row", "col", "grid", "card", "divider", "spacer",
  "stat", "badge", "progress", "keyvalue", "list", "table", "timeline",
  "callout", "steps", "avatar", "copy",
  "chart", "code", "json", "diff",
  "button", "input", "select", "checkbox", "switch", "radio", "slider",
  "textarea", "submit", "tabs", "accordion", "quiz",
]);

export const GENUI_FENCE_LANGS: ReadonlySet<string> = new Set(["genui", "dsh-ui"]);
