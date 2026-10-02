// Code generated from internal/gaea/genui/limits.go — DO NOT EDIT MANUALLY.
// 单一真相源在 Go 侧（审计 2026-10-02 X1-07）；重新生成：
//   GAEA_GENUI_LIMITS_UPDATE=1 go test -run TestGenuiLimitsSync ./internal/gaea/genui/
// 由 limits_sync_test.go 的 TestGenuiLimitsSync 逐字节校验（手改必红）。

/** Go 侧 genui 上限常量（唯一真相源；spec.ts 在此之上补渲染器专属上限）。 */
export const GENUI_GO_LIMITS = {
  maxDepth: 8,
  maxNodes: 200,
  maxString: 2000,
  maxCode: 12000,
  maxFenceBody: 65536,
  maxGridCols: 12,
  maxTableRows: 50,
  maxTableCols: 12,
  maxOptions: 50,
  maxChartPoints: 60,
} as const;

/** Go 侧合法组件 type 白名单（唯一真相源；spec.ts 用它构造 GENUI_NODE_TYPES）。 */
export const GENUI_GO_NODE_TYPES: readonly string[] = [
  "accordion",
  "avatar",
  "badge",
  "button",
  "callout",
  "card",
  "chart",
  "checkbox",
  "code",
  "col",
  "copy",
  "diff",
  "divider",
  "grid",
  "input",
  "json",
  "keyvalue",
  "list",
  "progress",
  "quiz",
  "radio",
  "row",
  "select",
  "slider",
  "spacer",
  "stat",
  "steps",
  "submit",
  "switch",
  "table",
  "tabs",
  "text",
  "textarea",
  "timeline",
];
