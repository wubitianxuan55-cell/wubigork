// types/cost.ts — 造价库（workcost）与语义索引状态类型。
// （旧成本库 / 价格源 / 组价 / 询价 / 五算对比的类型已随旧成本库退役拆除，2026-11-26。）

// SemanticIndexStatus 语义索引状态（各 kind 的向量条数 / 模型可用性；与 Go app 侧同构）。
export interface SemanticIndexStatus {
  available: boolean;
  counts: Record<string, number>; // kind → 向量条数
  detail?: Record<string, string>;
  error?: string; // 模型 / 存储不可用时的说明
}

// ── 工料法成本数据库（工料机资源库 + 消耗定额库 + 综合单价核算）──────────
//
// 口径（实测模版，旺平矿业/什邡五表产物）：
//   综合单价 = Σ(消耗量 × 资源价)，**只含人材机**，不含管理费/利润/税金；
//   管理费/利润/规费/税只在项目合计层跑一次（ComposeProjectFees）。
//   外委是独立行类，但三费归集并入材料桶（材料 = SUMIF(材料)+SUMIF(外委)）。

// WorkcostKind 工料机类别 + 外委。kind 值为中文，与 Go 侧常量一致。
export type WorkcostKind = "人工" | "材料" | "机械" | "外委";

// WorkcostResource 工料机资源（gf_resources）：独立主数据，有基准价与现行价。
// 核算取用价 = 现行价优先、为 0 回退基准价（EffectivePrice）。
export interface WorkcostResource {
  id: number;
  code: string;
  kind: WorkcostKind;
  title: string;
  spec: string;
  unit: string;
  basePrice: number;
  currentPrice: number;
  categoryPath: string;
  source: string;
  supplier: string;
  region: string;
  priceDate: string;
  priceType: string;
  validUntil: string;
  lossRate: number;
  note: string;
  tags: string[];
  status: string;
  createdAt?: string;
  updatedAt?: string;
}

// WorkcostResourcePrice 资源调价历史（gf_resource_prices）：信息价按期发布，
// 每次发布留一条，旧价保留可回看环比。
export interface WorkcostResourcePrice {
  id: number;
  resourceId: number;
  price: number;
  period: string;
  region: string;
  priceType: string;
  source: string;
  fetchedAt: string;
  note: string;
}

// WorkcostQuotaItem 定额子目的一条工料机消耗行（标准含量）。
export interface WorkcostQuotaItem {
  id?: number;
  quotaCode: string;
  resourceCode: string;
  kind: WorkcostKind;
  title: string;
  spec?: string;
  unit: string;
  quantity: number;
  resourcePrice?: number;
  lossRate?: number;
  note?: string;
  sort?: number;
}

// WorkcostQuota 消耗定额（gf_quotas）：某清单子目每单位消耗的工料机标准含量。
export interface WorkcostQuota {
  id: number;
  code: string;
  /** 工作簿原始清单编码（编码统一可能改写 code，溯源靠它）。 */
  origCode?: string;
  title: string;
  specialty: string;
  chapter: string;
  unit: string;
  categoryPath: string;
  baseLabor: number;
  baseMaterial: number;
  baseMachine: number;
  source: string;
  region: string;
  priceDate: string;
  note: string;
  status: string;
  items?: WorkcostQuotaItem[];
  /** 被多少条清单项引用（读时聚合）——企业定额库复用度。 */
  usageCount?: number;
  createdAt?: string;
  updatedAt?: string;
}

// WorkcostComposeLineView 组成行的核算视图（含派生的资源费与占比）。
export interface WorkcostComposeLineView {
  kind: WorkcostKind;
  title: string;
  unit: string;
  quantity: number;
  price: number;
  lossRate: number;
  amount: number;
  sharePct: number;
}

// WorkcostCompose 综合单价分析结果——**只含人材机**。
// materialFee 已含外委（口径同实测模版汇总行）。
export interface WorkcostCompose {
  laborFee: number;
  materialFee: number;
  machineFee: number;
  outsourcedFee: number;
  otherFee: number;
  subtotal: number;
  compositePrice: number;
  zeroLines: number;
  lines: WorkcostComposeLineView[];
  warnings: string[];
}

// WorkcostBillProject 清单项目（五表封面 + 录入费率；一次导入=一个项目）。
// 费率是录入数据——加总计算不在库里发生，导出五表活公式工作簿由 Excel 算。
export interface WorkcostBillProject {
  id: number;
  name: string;
  fileName: string;
  source: string;
  location: string;
  duration: string;
  pricing: string;
  managementRate: number;
  regulatoryRate: number;
  profitRate: number;
  taxRate: number;
  profitIncludesRegulatory: boolean;
  controlPrice: number;
  itemCount: number;
  createdAt?: string;
  updatedAt?: string;
}

// WorkcostBillItem 分部分项清单项：编码/名称/单位/工程量 × 引用定额。
// quotaCode 非空=套定额（综合单价导出时现算参考）；priceOverride>0=手填单价。
// listPrice=原表综合单价快照（导入时从工作簿落库留痕，SchemaV34；0/缺省=无快照）。
export interface WorkcostBillItem {
  id: number;
  projectId: number;
  code: string;
  title: string;
  unit: string;
  division: string;
  quantity: number;
  quantityExpr: string;
  quotaCode: string;
  feature: string;
  priceOverride: number;
  /** 原表综合单价快照（对照显形与案例样本用，不参与现算）。 */
  listPrice?: number;
  sort: number;
  /** 挂接的全部定额码（1:N 组合，gf_bill_quota_links 聚合；quotaCode 是主定额）。 */
  quotaCodes?: string[];
}

/** WorkcostBillQuotaLink 清单项挂接的一条定额引用（1:N 组合）。 */
export interface WorkcostBillQuotaLink {
  id: number;
  billItemId: number;
  quotaCode: string;
  /** 该定额参与合价的工程量；0 = 跟随清单工程量。 */
  quantity: number;
  sort: number;
  createdAt: string;
  updatedAt: string;
}

// WorkcostRateSet 取费参数（项目级）。
export interface WorkcostRateSet {
  managementRate: number;
  regulatoryRate: number;
  profitRate: number;
  taxRate: number;
}

// WorkcostFeeResult 项目合计层取费结果（费用汇总下半段）。
export interface WorkcostFeeResult {
  directFee: number;
  managementFee: number;
  profitFee: number;
  regulatoryFee: number;
  measuresFee: number;
  contingency: number;
  preTaxTotal: number;
  taxFee: number;
  total: number;
  controlPrice: number;
  controlDiff: number;
  controlUtilPct: number;
}

// WorkcostComposeOverride 项目级覆盖含量/单价（key=资源编码；
// 含量或单价传负值表示不覆盖该项）。
export interface WorkcostComposeOverride {
  quantity: number;
  price: number;
}

// ── 项目工作簿（五表模版）解析与落库 ──

export interface WorkcostProjectResource {
  code: string;
  kind: WorkcostKind;
  title: string;
  spec: string;
  unit: string;
  price: number;
  priceFormula: string;
  source: string;
  note: string;
}

export interface WorkcostProjectConsumption {
  seq: string;
  itemCode: string;
  itemTitle: string;
  itemUnit: string;
  kind: WorkcostKind;
  resourceCode: string;
  resourceTitle: string;
  unit: string;
  quantity: number;
  price: number;
  priceFormula: string;
  amount: number;
}

export interface WorkcostProjectItem {
  code: string;
  title: string;
  unit: string;
  division: string;
  feature: string;
  quantity: number;
  quantityExpr: string;
  categoryPath: string;
}

export interface WorkcostProjectFeeParams {
  managementRate: number;
  regulatoryRate: number;
  profitRate: number;
  taxRate: number;
  controlPrice: number;
  managementFormula: string;
  taxNote: string;
}

export interface WorkcostProjectQuantity {
  code: string;
  title: string;
  value: number;
  unit: string;
  note: string;
}

export interface WorkcostSkippedRow {
  sheet: string;
  row: number;
  reason: string;
}

export interface WorkcostProjectBundle {
  path: string;
  fileName: string;
  project: string;
  location: string;
  duration: string;
  pricing: string;
  resources: WorkcostProjectResource[];
  items: WorkcostProjectItem[];
  lines: WorkcostProjectConsumption[];
  fee: WorkcostProjectFeeParams;
  quantities: WorkcostProjectQuantity[];
  skipped: WorkcostSkippedRow[];
  warnings: string[];
  sheets: string[];
}

export interface WorkcostApplyProjectResult {
  project: string;
  resourceNew: number;
  resourceUpd: number;
  quotaNew: number;
  quotaUpd: number;
  lines: number;
  errors: string[];
}

// ── 存量资源化（成本条目 → 工料机资源库）──

// ── 企业定额自生成（11.6；roadmap 造价愿景#5）────────────────────

// WorkcostQuotaFlywheelMember 聚类成员（同工作内容的一个定额变体）。
export interface WorkcostQuotaFlywheelMember {
  code: string;
  origCode: string;
  source: string;
  itemCount: number;
}

// WorkcostQuotaFlywheelRow 候选企业定额的一条聚合含量行（数量=各变体中位数）。
export interface WorkcostQuotaFlywheelRow {
  resourceCode: string;
  kind: string;
  title: string;
  spec?: string;
  unit?: string;
  quantity: number;
  coverage: number;
  min: number;
  max: number;
  majority: boolean;
}

// WorkcostQuotaFlywheelCandidate 一个工作内容的企业定额候选。
export interface WorkcostQuotaFlywheelCandidate {
  origCode: string;
  newCode: string;
  title: string;
  unit: string;
  members: WorkcostQuotaFlywheelMember[];
  rows: WorkcostQuotaFlywheelRow[];
  sources: number;
}

export interface WorkcostQuotaFlywheelSkip {
  key: string;
  title: string;
  reason: string;
}

export interface WorkcostQuotaFlywheelPreview {
  clusters: number;
  candidates: WorkcostQuotaFlywheelCandidate[];
  skipped: WorkcostQuotaFlywheelSkip[];
}

export interface WorkcostQuotaFlywheelResult {
  saved: string[];
  skipped: WorkcostQuotaFlywheelSkip[];
}

// ── 跨项目对标查漏（11.3；复核增强而非替代）────────────────────

// WorkcostAuditEvidence 一条溯源引用（支撑本结论的项目/定额）。
export interface WorkcostAuditEvidence {
  ref: string;
  source: string;
  support: string;
}

// WorkcostAuditFinding 一条查漏线索（kind: bill_gap | quota_gap）。
export interface WorkcostAuditFinding {
  kind: string;
  target: string;
  detail: string;
  evidence: WorkcostAuditEvidence[];
}

export interface WorkcostAuditReport {
  billGaps: WorkcostAuditFinding[];
  quotaGaps: WorkcostAuditFinding[];
  auditedProj: string;
  projectHits: number;
}
