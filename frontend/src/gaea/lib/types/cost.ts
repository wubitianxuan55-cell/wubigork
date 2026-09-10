// ── 成本库类型 ───────────────────────────────────────────────────────────


// ── 成本库 ──────────────────────────────────────────────────────────
export interface CostSummary {
  name: string;
  title: string;
  // 定额编码/清单编码（归一化：半角大写、去空白；空=未录入）。
  code?: string;
  category: string;
  // 完整分类路径：一级/二级/…/叶子（多级分类保存与树形过滤依据）。
  categoryPath: string;
    unit: string;
    price: number;
    // 人材机二级汇总（综合单价子目，元）。
    laborFee?: number;
    materialFee?: number;
    machineFee?: number;
    // 人材机组成行数（综合单价子目的二级明细规模）。
    componentCount?: number;
    spec: string;
  source: string;
  // 价格三要素蒸馏（zaojia-database）：地区 + 价格时间/期数；口径与有效期辅助判断可信度。
  region?: string;
  priceDate?: string;
  priceType?: string;
  validUntil?: string;
  sourceRow?: number;
  tags: string[];
  status: string;
  // 新建/导入时前端不发送时间戳（Go 端 time.Time 不接受空串），留空由后端置零。
  updatedAt?: string;
}
  // 综合单价子目的人材机组成明细行（二级）。
  export interface CostComponent {
    kind: string; // 人工/材料/机械（可组合标签，如 人工+机械）
    title: string;
    unit?: string;
    quantity?: number; // 含量/数量（0=未解析出）
    price?: number; // 资源单价（0=未解析出）
    amount?: number; // 金额（含量×单价，含损耗）
    note?: string; // 原始行表达式（含损耗系数，追溯用）
    sort?: number;
  }

  export interface CostEntry extends CostSummary {
    // 费率（仅展示追溯，不参与计算）：管理费/利润/垫资 为金额（元），税率为百分比。
    managementFee?: number;
    profitFee?: number;
    advanceFee?: number;
    taxRate?: number;
    components?: CostComponent[];
    body: string;
    createdAt?: string;
  }

// 成本分类树节点（多级：parentId 自引用，children 为子树）。
export interface CostCategory {
  id: number;
  parentId: number;
  name: string;
  sort: number;
  count: number;
  children?: CostCategory[];
}

// ── 测算项目与沉淀闭环（zaojia-database 蒸馏：我的项目/工程量清单/版本留痕）──

// CostProject 测算项目容器（一次报价/测算工作）。
export interface CostProject {
  id: string;
  name: string;
  projectType: string;
  scale: string;
  craft: string;
  status: string; // 编制中 / 已保存版本 / 已沉淀
  note: string;
  createdAt?: string;
  updatedAt?: string;
}

// CostProjectSummary 项目列表视图（含条目数/合计/版本数）。
export interface CostProjectSummary extends CostProject {
  itemCount: number;
  total: number;
  versionCount: number;
}

// CostEstimateItem 测算明细行（工程量清单行；金额=数量×单价自动计算）。
export interface CostEstimateItem {
  id?: number;
  projectId: string;
  name: string; // kebab 稳定名（沉淀为成本条目 name）
  title: string;
  categoryPath: string;
  unit: string;
  code?: string; // 定额编码/清单编码（归因对标/组价检索锚点）
  quantity: number;
  price: number;
  amount?: number;
  entryName?: string; // 引用的成本条目 name（可空=手动估价）
  source?: string;
  note?: string;
  sort?: number;
  createdAt?: string;
  updatedAt?: string;
}

// CostEstimateVersion 不可变版本快照（保存时对明细行 JSON 快照 + 合计）。
export interface CostEstimateVersion {
  id: number;
  projectId: string;
  version: number;
  total: number;
  snapshot: string;
  note: string;
  createdAt: string;
}

// CostIndicator 造价参考指标（对案例项目明细行的价格聚合，实时计算不落表）。
export interface CostIndicator {
  key: string; // 科目标题 或 一级分类名
  unit: string;
  samples: number;
  min: number;
  max: number;
  mean: number;
  median: number;
  p25: number;
  p75: number;
}

// v4.6.1 归因对标（CostAttribution）：项目明细 vs 参考指标带宽的逐行对标。
export interface CostAttributionItem {
  title: string;
  unit: string;
  quantity: number;
  price: number;
  amount: number;
  refSamples: number;
  refMedian: number;
  refP25: number;
  refP75: number;
  diffPct: number;
  level: string; // 高/正常/低/无参考
  contribution: number; // (price-refMedian)×quantity
}

export interface CostAttribution {
  projectId: string;
  projectName: string;
  totalAmount: number;
  refTotal: number;
  totalDiff: number;
  totalDiffPct: number;
  items: CostAttributionItem[];
  topDrivers: CostAttributionItem[];
  summary: string;
}

// CostReviewNote 复盘笔记（结论/边界/风险/证据/可信度/有效期/复核状态）。
export interface CostReviewNote {
  id?: number;
  title: string;
  conclusion: string;
  boundary: string;
  risk: string;
  evidence: string;
  confidence: string; // 高/中/低
  validUntil?: string;
  status: string; // 草稿 / 已确认
  category?: string;
  projectType?: string;
  craft?: string;
  refCount?: number;
  createdAt?: string;
  updatedAt?: string;
}

// ── v4.8 成本知识图谱（CostGraph，后端返回 JSON 串前端解析）──────────

// CostGraphNode 图节点：type ∈ category|entry|project|item|indicator|inquiry|note。
// val 为金额量级（分类=子树合计/条目=单价/项目=总合计/明细=金额/指标=中位数/
// 询价=单价/笔记=引用数），供节点定半径；meta 为点击弹窗展示的结构化明细。
export interface CostGraphNode {
  id: string;
  name: string;
  type: string;
  desc: string;
  val: number;
  meta?: Record<string, string>;
}

// CostGraphEdge 图边：type ∈ belongs_to(category→entry)|contains(project→item)|
// references(item→entry)|benchmarks(item→indicator)|suggests(inquiry→entry)|
// notes(note→category)；meta.matchedBy ∈ entry_name|title（匹配方式溯源）。
export interface CostGraphEdge {
  source: string;
  target: string;
  type: string;
  weight: number;
  meta?: Record<string, string>;
}

// CostGraphStats 图规模统计（nodeCount/edgeCount 为截断后的实际数量）。
export interface CostGraphStats {
  truncated: boolean;
  nodeCount: number;
  edgeCount: number;
  countsByType: Record<string, number>;
}

export interface CostGraphView {
  nodes: CostGraphNode[];
  edges: CostGraphEdge[];
  stats: CostGraphStats;
}

// 导入预览中的一条候选成本条目（前端可编辑后确认导入）。
export interface CostImportRow {
  name: string;
  title: string;
  code?: string;
  category: string;
  unit: string;
  price: number;
  // 综合单价架构：人材机二级 = 合计 + 组成明细；费率仅展示追溯。
  laborFee?: number;
  materialFee?: number;
  machineFee?: number;
  managementFee?: number;
  profitFee?: number;
  advanceFee?: number;
  taxRate?: number;
  components?: CostComponent[];
  body?: string;
  spec: string;
  source: string;
  // 原始工作表物理行号（1-based；0=无法确定，如纵向参数表/AI 解析）。
  sourceRow?: number;
  status: string;
  existingName: string;
  existingPrice: number;
  matchNote: string; // 新增 / 将覆盖更新（现价 ¥xxx）
  raw: string;
  skip: boolean;
  skipReason: string;
}

// 文件导入解析结果（无确认不落库，确认走 CostImportApply）。
export interface CostImportPreview {
  path: string;
  fileName: string;
  columns: string[];
  unmapped: string[];
  rows: CostImportRow[];
  message: string;
  aiUsed: boolean;
  // 导入文件来源类型：xlsx/csv 表格解析；pdf_text 文字型 PDF；
  // pdf_scan 扫描件（OCR）；image 图片报价单（OCR）。
  source?: "xlsx" | "csv" | "pdf_text" | "pdf_scan" | "image";
}

// ── 价格源（定时抓取价格更新）──────────────────────────────────
export interface PriceSource {
  id: string;
  name: string;
  url: string;
  parser: string; // sc_table：造价信息网价格表
  frequencyHours: number; // 0=仅手动
  area: string;
  headers?: Record<string, string>;
  enabled: boolean;
  lastFetchAt: string;
  createdAt: string;
}

export interface PriceCandidate {
  title: string;
  spec: string;
  unit: string;
  price: number;
  tax: string;
  existingName: string;
  existingPrice: number;
  status: "更新" | "无变化" | "新增";
  diff: number;
  diffPct: number;
  anomaly: boolean; // 偏离历史价格区间（价格异常）
  anomalyReason: string;
}

export interface PriceFetchRecord {
  id: string;
  sourceId: string;
  sourceName: string;
  url: string;
  period: string;
  fetchedAt: string;
  status: "pending" | "applied" | "ignored";
  candidates: PriceCandidate[];
}

export interface PriceHistory {
  name: string;
  title: string;
  unit: string;
  price: number;
  source: string;
  period: string;
  fetchedAt: string;
  note: string;
}
// CostCompareRow 是一条比价明细：同一成本条目在多个来源/时段的报价对比。
// kind: current=成本库现价 / history=历史快照 / fetch=价格源抓取候选。
export interface CostCompareRow {
  source: string; // 来源（如「成本库」「四川造价信息网」「XX租赁」）
  period: string; // 期号/时段（如 "758"；成本库现价可为空）
  price: number; // 该来源报价
  diffPct: number; // 与基准价（通常为成本库现价）的偏差百分比，基准行为 0
  fetchedAt: string; // 获取时间（ISO 字符串，可为空）
  kind: "current" | "history" | "fetch";
}

// ── v4.2 造价 AI 化（docs/gaea-v42-cost-ai-design.md）────────────────
// PriceBand 是相似清单的价格带推荐（Go cost.PriceBand 视图：分位数 R-7 口径）。
export interface PriceBandSource {
  name: string;
  title: string;
  category: string;
  unit: string;
  spec: string;
  source: string;
  region: string;
  priceDate: string;
  priceType: string;
  price: number;
  updatedAt: string;
}
export interface PriceBand {
  samples: number;
  min: number;
  max: number;
  mean: number;
  median: number;
  p25: number;
  p75: number;
  spreadPct: number;
  outliers: number;
  confidence: string;
  sources: PriceBandSource[];
}
// CostComposeEvidence 组价证据链一条：溯源字段（来源/地区/期数/口径）即证据格式。
export interface CostComposeEvidence {
  name: string;
  title: string;
  category: string;
  unit: string;
  spec: string;
  price: number;
  source: string;
  region: string;
  priceDate: string;
  priceType: string;
}
// CostComposeCheck 组价合理性校验一条（v4.158 AI 组价复核闭环）：LLM 拆解后才
// 可能返回，旧响应无此字段（可选，向后兼容）。row=组件行下标（-1=全局提示）。
export interface CostComposeCheck {
  level: "warn" | "info";
  row: number;
  msg: string;
}
// CostComposeView 是 AI 组价建议（无确认不落库；band=null 表示成本库无相似条目）。
export interface CostComposeView {
  description: string;
  unit: string;
  band: PriceBand | null;
  recommendedPrice: number;
  reason: string;
  components?: CostComponent[];
  componentsNote?: string;
  llmUsed: boolean;
  evidence: CostComposeEvidence[];
  // v4.158 合理性校验（LLM 拆解后才可能有；无字段/空数组 = 无校验，前端零变化）。
  checks?: CostComposeCheck[];
}
// CostComposeRecord 组价确认记录（v4.158 复核闭环）：每次确认组价（CostComposeApply）
// 对完整视图留痕快照，供条目详情回看「组价依据」。snapshot 为确认时的完整视图
// （含 band/recommendedPrice/components/evidence/checks），置 null 表示快照缺失。
export interface CostComposeRecord {
  id: number;
  entryName: string;
  createdAt: string;
  llmUsed: boolean;
  snapshot: CostComposeView | null;
}
// ── 询价飞轮（四源归一：信息价/OCR报价/供应商比价/手动询价）──
export interface CostInquiryRecord {
  id: number;
  title: string;
  spec: string;
  unit: string;
  price: number;
  source: string;
  supplier: string;
  region: string;
  priceDate: string;
  validUntil: string;
  note: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}
// CostAdjustSuggestion 调差建议：成本库条目 vs 最新询价数据点（|差幅|>2%）。
// CostInquiryScanFinding 库级异常扫描发现（询价库内部自洽体检，与调差建议互补）。
export interface CostInquiryScanFinding {
  kind: string; // 离散/跳变/过期/陈旧
  severity: string; // 关注/异常
  title: string; // 涉及标题（组=归一化标题，单条=原标题）
  detail: string; // 人话描述（含关键数值）
  refIds: number[]; // 涉及记录 id
}

export interface CostAdjustSuggestion {
  entryName: string;
  entryTitle: string;
  entryPrice: number;
  latestPrice: number;
  latestDate: string;
  latestSource: string;
  diff: number;
  diffPct: number;
  unit: string;
  // v4.6 询价异常检测 + 价格预测：差幅分级（正常/关注/异常）与询价序列
  // 线性回归下期预测价（0/缺省 = 数据点太少无可预测）。
  level?: string;
  predictedNext?: number;
  predictionNote?: string;
}
// ── 五算对比（估/概/预/结/决，coststage）──
export interface CostStageValue {
  id: number;
  projectId: string;
  stage: string;
  amount: number;
  date: string;
  note: string;
  createdAt: string;
  updatedAt: string;
}
// CostStageCompareRow 五算对比行：固定 5 阶段顺序，缺阶段 hasValue=false。
export interface CostStageCompareRow {
  stage: string;
  amount: number;
  hasValue: boolean;
  prevStage: string;
  hasPrev: boolean;
  chainDiff: number;
  chainDiffPct: number;
  baseDiff: number;
  baseDiffPct: number;
}
// CostStageDeviation 相邻阶段偏差特征（level: 正常/关注/异常，供复盘诊断）。
export interface CostStageDeviation {
  fromStage: string;
  toStage: string;
  fromAmount: number;
  toAmount: number;
  diff: number;
  diffPct: number;
  direction: string;
  level: string;
  suggestion: string;
}
