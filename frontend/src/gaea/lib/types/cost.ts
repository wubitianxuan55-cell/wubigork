// ── 成本库类型 ───────────────────────────────────────────────────────────
// 2026-11-25 造价数据库推倒重建后只留仍在消费的 wire 镜像：成本条目摘要与分类
// （统一检索链）、价格源记录、工料法三级数据（工料机 / 定额 / 清单）。旧板块的
// 资料条目编辑 / 导入预览 / 询价 / 对标 / 图谱 / 五算 / 飞轮等类型随界面退役。
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

// 成本分类树节点（多级：parentId 自引用，children 为子树）。
export interface CostCategory {
  id: number;
  parentId: number;
  name: string;
  sort: number;
  count: number;
  children?: CostCategory[];
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
  /** 预匹配到的工料机资源（唯一命中才有值）——发布时可同步推进现行价。 */
  resourceCode?: string;
  resourceTitle?: string;
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

// ── 工料法成本数据库（工料机资源库 + 消耗定额库 + 综合单价核算）──────────
//
// 口径（实测模版，旺平矿业/什邡五表产物）：
//   综合单价 = Σ(消耗量 × 资源价)，**只含人材机**，不含管理费/利润/税金；
//   管理费/利润/规费/税只在项目合计层跑一次（ComposeProjectFees）。
//   外委是独立行类，但三费归集并入材料桶（材料 = SUMIF(材料)+SUMIF(外委)）。

// WorkcostKind 工料机类别 + 外委。kind 值为中文，与 Go 侧常量一致。
export type WorkcostKind = "人工" | "材料" | "机械" | "外委";

/**
 * 读侧的类别：规范值是四类之一；库定规矩之前的存量行会带着别的取值原样回来
 * （界面归到「类别待整理」一桶）。写入面仍只收 WorkcostKind。
 */
export type WorkcostKindRead = WorkcostKind | (string & Record<never, never>);

// WorkcostResource 工料机资源（gf_resources）：独立主数据，有基准价与现行价。
// 核算取用价 = 现行价优先、为 0 回退基准价（EffectivePrice）。
export interface WorkcostResource {
  id: number;
  code: string;
  kind: WorkcostKindRead;
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

// WorkcostUsageRef 一条引用：谁在用这条数据（Go workcost.UsageRef 镜像）。
// 定额的引用方是清单项（owner=所属项目，quantity=挂接工程量）；工料机的引用方
// 是定额含量行（unit=定额计量单位，quantity=含量）。
export interface WorkcostUsageRef {
  id: number;
  code: string;
  title: string;
  owner?: string;
  unit?: string;
  quantity: number;
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
  /** 专业——定额的固有属性：随来源定、定了不改；空=尚未标注（待整理）。 */
  specialty: string;
  chapter: string;
  /** 借鉴生成的溯源（「借鉴 <专业> <码>」）：本专业没有可用定额时以外专业定额为底生成；空=非借鉴。 */
  derivedFrom?: string;
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
// specialty/region/pricePeriod 是规范值（库定规矩）：专业、地区取字典值，计价期
// YYYY-MM；空=尚未标注（待整理）。region 与封面地址 location 分开。
export interface WorkcostBillProject {
  id: number;
  name: string;
  fileName: string;
  source: string;
  location: string;
  duration: string;
  pricing: string;
  specialty: string;
  region: string;
  pricePeriod: string;
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
  /** 清单项专业（读取时已归一：行自身为空则取项目默认专业；都空=待整理）。 */
  specialty?: string;
  /** 各挂接定额的工程量（定额码 → 工程量；0=跟随清单工程量）。详情里「消耗量 = 工程量 × 含量」用它。 */
  linkQuantities?: Record<string, number>;
}

// WorkcostLibraryFilter 造价库列表的筛选口径（板块列表与导出共用，保证「导出（当前
// 筛选）」导的就是界面上看到的那些行）。零值=不过滤；值为 NONE 哨兵（见 cost-library/logic）
// =只要该字段为空的行（未标注 / 未细分的那一桶）。
export interface WorkcostLibraryFilter {
  projectId: number;
  keyword: string;
  specialty: string;
  division: string;
  chapter: string;
  kind: string;
  category: string;
  region: string;
  period: string;
}

// WorkcostComposeOverride 项目级覆盖含量/单价（key=资源编码；
// 含量或单价传负值表示不覆盖该项）。
export interface WorkcostComposeOverride {
  quantity: number;
  price: number;
}

// ── 项目工作簿（五表模版）解析与落库 ──

// ── 存量资源化（成本条目 → 工料机资源库）──

// ── 企业定额自生成（11.6；roadmap 造价愿景#5）────────────────────

// ── 跨项目对标查漏（11.3；复核增强而非替代）────────────────────
