// 记忆中枢领域语义色单一数据源。
//
// Why: 三个组件此前各自维护同一张领域色表（GraphView TYPE_COLORS /
// MaterialsLibrary Pin / WhisperMemoryLibrary 情节记忆），audit 10-ui-audit.md
// 指出「同一色值 3 处重复维护」。领域分类色是语义色（图谱节点/库徽标/图标
// 按库类型区分），审计明确保留——这里收敛为单源，三处消费同一常量。
//
// How to apply: 新增领域分类时只改本文件 + TYPE_LABELS；组件内
// `import { DOMAIN_COLORS } from "../../lib/domainColors"` 按 key 取色。

/** 记忆库领域分类 → 主题语义色。 */
export const DOMAIN_COLORS: Record<string, string> = {
  profile: "var(--gaea-chart-5)",
  office: "var(--gaea-chart-2)",
  whisper: "var(--gaea-chart-6)",
  wenxue: "var(--gaea-chart-9)",
  material: "var(--gaea-chart-4)",
  database: "var(--gaea-chart-8)",
  cost: "var(--gaea-chart-10)",
  entity: "var(--gaea-chart-7)",
  event: "var(--gaea-chart-muted)",
  source: "var(--gaea-chart-11)",
};

/** 领域分类 → 中文标签（图谱图例/库徽标共用）。 */
export const DOMAIN_LABELS: Record<string, string> = {
  profile: "画像",
  office: "办公记忆",
  whisper: "聊天记忆",
  material: "项目资料",
  cost: "成本",
  entity: "记忆实体",
  event: "记忆事件",
  source: "来源",
};

/** 领域分类 key 清单（图谱图例/筛选用，稳定顺序）。 */
export const DOMAIN_KEYS = ["profile", "office", "whisper", "material", "cost", "entity", "event", "source"] as const;
