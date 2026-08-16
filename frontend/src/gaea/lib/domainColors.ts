// 记忆中枢领域语义色单一数据源。
//
// Why: 三个组件此前各自维护同一张领域色表（GraphView TYPE_COLORS /
// MaterialsLibrary Pin / WhisperMemoryLibrary 情节记忆），audit 10-ui-audit.md
// 指出「同一色值 3 处重复维护」。领域分类色是语义色（图谱节点/库徽标/图标
// 按库类型区分），审计明确保留——这里收敛为单源，三处消费同一常量。
//
// How to apply: 新增领域分类时只改本文件 + TYPE_LABELS；组件内
// `import { DOMAIN_COLORS } from "../../lib/domainColors"` 按 key 取色。

/** 记忆库领域分类 → 语义色（固定色值：领域色不随主题，图谱节点/徽标靠色区分）。 */
export const DOMAIN_COLORS: Record<string, string> = {
  knowledge: "#818cf8", // indigo
  profile: "#a78bfa", // violet
  office: "#34d399", // emerald
  whisper: "#f472b6", // pink：聊天/情节记忆
  material: "#38bdf8", // sky：项目资料（固定常用文件）
  cost: "#fbbf24", // amber：成本条目
};

/** 领域分类 → 中文标签（图谱图例/库徽标共用）。 */
export const DOMAIN_LABELS: Record<string, string> = {
  knowledge: "知识",
  profile: "画像",
  office: "办公记忆",
  whisper: "聊天记忆",
  material: "项目资料",
  cost: "成本",
};

/** 领域分类 key 清单（图谱图例/筛选用，稳定顺序）。 */
export const DOMAIN_KEYS = ["knowledge", "profile", "office", "whisper", "material", "cost"] as const;
