// cost.ts — CostBindings（AppBindings 分域接口之一，Go CostB 门面）：
// 造价库维护（workcost：库级维护 / 清单 / 定额 / 工料机）与跨库语义检索域。
// （旧成本库 / 价格源 / 组价 / 询价飞轮 / 五算对比 / 造价参考域 2026-11-26 退役，已摘。）

/** 资源价批量调价一行（Go workcost.BulkPriceRow 镜像）。 */
/** 跳过行（人可读原因）。 */
export interface CostBulkPriceSkip {
  row: number
  title: string
  reason: string
}

/** 预览的一条匹配（资源现状速览）。 */
export interface CostBulkPriceMatch {
  row: number
  resourceId: number
  code: string
  title: string
  spec?: string
  unit?: string
  oldPrice: number
  newPrice: number
  period?: string
  region?: string
  note?: string
  resourceKind?: string
}

/** 批量调价预览报告。 */
export interface CostBulkPricePreview {
  total: number
  matched: CostBulkPriceMatch[]
  skipped: CostBulkPriceSkip[]
  rowErrors: CostBulkPriceSkip[]
}

/** 批量调价应用回执。 */
export interface CostBulkPriceResult {
  updated: number
  skipped: CostBulkPriceSkip[]
  total: number
  applied: CostBulkPriceMatch[]
}

import type {
  SemanticHitView,
  SemanticIndexStatus,
  WorkcostApplyProjectResult,
  WorkcostBillItem,
  WorkcostBillQuotaLink,
  WorkcostBillProject,
  WorkcostCompose,
  WorkcostComposeOverride,
  WorkcostFeeResult,
  WorkcostProjectBundle,
  WorkcostQuota,
  WorkcostAuditReport,
  WorkcostQuotaFlywheelPreview,
  WorkcostQuotaFlywheelResult,
  WorkcostRateSet,
  WorkcostResource,
  WorkcostResourcePrice,
} from "../types";

export interface CostBindings {
  // ── 造价库维护（库级，workcost）──
  // CostBulkPricePreview 解析资源价批量调价 Excel 并预览匹配结果（三期 3.4，
  // 宁缺勿误：跳过行带人可读原因）。
  CostBulkPricePreview(xlsxBase64: string): Promise<CostBulkPricePreview>;
  // CostBulkPriceApply 作者确认后批量应用（整批事务：写调价历史+推现行价）。
  CostBulkPriceApply(xlsxBase64: string): Promise<CostBulkPriceResult>;
  // CostQuotaFlywheelPreview 扫描跨项目定额含量分歧（orig_code 簇），提出
  // 企业定额候选（11.6；宁缺勿误：不成立簇带人可读原因）。
  CostQuotaFlywheelPreview(): Promise<WorkcostQuotaFlywheelPreview>;
  // CostQuotaFlywheelApply 作者点名聚类词干后落 ENT-* 企业定额（含量=各变体
  // 中位数；只覆写自有命名空间，重生成=飞轮进化）。
  CostQuotaFlywheelApply(origCodes: string[]): Promise<WorkcostQuotaFlywheelResult>;
  // CostAuditScan 跨项目对标查漏（11.3）：确定性引擎（宁缺勿误闸）+逐条溯源
  // 引用；bill_gap 针对被审项目（0=跳过），quota_gap 全语料逐同名组。
  CostAuditScan(projectID: number): Promise<WorkcostAuditReport>;
  // CostHealthScan 数据健康五类扫描（零价/缺价子目/悬挂引用/重复防护/编码混用）。
  CostHealthScan(): Promise<Record<string, unknown>>;
  // CostDedupeResources 资源去重（keep 保留，merges 合并+引用重定向）。
  CostDedupeResources(keepId: number, mergeIds: number[]): Promise<Record<string, unknown>>;
  // SemanticSearch 跨库统一语义检索（办公记忆/工作区文件，本地 bge-m3）。
  // SemanticIndexStatus 语义索引状态（各 kind 向量条数 / 模型可用性）。
  SemanticIndexStatus(): Promise<SemanticIndexStatus>;
  SemanticSearch(query: string): Promise<SemanticHitView[]>;
  // ── 工料法成本数据库（工料机资源库 + 消耗定额库 + 综合单价核算）──
  // 口径：综合单价 = Σ(消耗量×资源价)，**只含人材机**；管理费/利润/税只在
  // 项目合计层跑一次（WorkcostProjectFees）。外委独立行类但并入材料桶。
  WorkcostResourceSave(r: WorkcostResource): Promise<WorkcostResource>;
  // WorkcostResourceGet 按 id 读资源；不存在返回 null。
  WorkcostResourceGet(id: number): Promise<WorkcostResource | null>;
  // WorkcostResourceList 资源列表（kind/keyword 可空 = 不过滤）。
  WorkcostResourceList(kind: string, keyword: string): Promise<WorkcostResource[]>;
  // WorkcostResourceDelete force=false 时拒绝删除仍被定额引用的资源。
  WorkcostResourceDelete(id: number, force: boolean): Promise<void>;
  // WorkcostResourceSetPrice 登记调价（写历史 + 推进现行价）——信息价一涨，
  // 引用该资源的综合单价在下次核算时同步重算。
  WorkcostResourceSetPrice(
    id: number,
    price: number,
    period: string,
    region: string,
    priceType: string,
    source: string,
    note: string,
  ): Promise<WorkcostResource>;
  WorkcostResourcePrices(id: number): Promise<WorkcostResourcePrice[]>;
  // WorkcostQuotaSave 新建/更新消耗定额（含工料机含量行，整组替换）。
  WorkcostQuotaSave(q: WorkcostQuota): Promise<WorkcostQuota>;
  WorkcostQuotaGet(code: string): Promise<WorkcostQuota | null>;
  WorkcostQuotaList(specialty: string, keyword: string): Promise<WorkcostQuota[]>;
  WorkcostQuotaDelete(code: string): Promise<void>;
  // WorkcostQuotaCompose 按定额核算综合单价（只含人材机）。overrides 支持项目级
  // 覆盖含量/单价（key=资源编码；负值表示不覆盖该项）。
  WorkcostQuotaCompose(
    code: string,
    overrides: Record<string, WorkcostComposeOverride> | null,
  ): Promise<WorkcostCompose>;
  // WorkcostQuotaComposeMany 批量核算（清单库参考价列专用）：一次资源索引复用，
  // key=定额编码；不存在的编码不在结果里。
  WorkcostQuotaComposeMany(codes: string[]): Promise<Record<string, WorkcostCompose>>;
  // WorkcostProjectFees 项目合计层取费：直接费 → 企管/利润/规费 → 税前合计 →
  // 增值税 → 含税总造价（含招标控制价对照）。
  // profitIncludesRegulatory 决定利润基数是否含规费（百锦路=false，市政道路=true）。
  WorkcostProjectFees(
    directFee: number,
    rates: WorkcostRateSet,
    profitIncludesRegulatory: boolean,
    measures: number,
    contingency: number,
    controlPrice: number,
  ): Promise<WorkcostFeeResult>;
  // ── 分部分项清单（录入数据，不做加总）──
  // WorkcostBillProjects 清单项目列表（含封面、费率与清单项数）。
  WorkcostBillProjects(): Promise<WorkcostBillProject[]>;
  // WorkcostBillItems 某项目的分部分项清单。
  WorkcostBillItems(projectId: number): Promise<WorkcostBillItem[]>;
  // WorkcostBillItemSave 新增/更新一条清单项（同项目同编码走更新；code 空
  // 自动生成 M 序号）。工程量与引用定额都是录入数据——合价导出时由 Excel 算。
  WorkcostBillItemSave(item: WorkcostBillItem): Promise<WorkcostBillItem>;
  // WorkcostBillItemDelete 删除一条清单项（1:N 定额挂接引用随行清理）。
  WorkcostBillItemDelete(id: number): Promise<void>;
  // WorkcostBillProjectRatesSave 保存项目费率（录入数据）。
  WorkcostBillProjectRatesSave(
    id: number,
    rates: WorkcostRateSet,
    profitIncludesRegulatory: boolean,
    controlPrice: number,
  ): Promise<void>;
  // WorkcostBillProjectDelete 整体删除导入的项目（封面+清单+独占定额；
  // 共享定额与资源保留）。返回删除的独占定额数。
  WorkcostBillProjectDelete(id: number): Promise<number>;
  // WorkcostBillQuotaLinks 清单项挂接的定额引用（1:N 组合，SchemaV30）。
  WorkcostBillQuotaLinks(billItemId: number): Promise<WorkcostBillQuotaLink[]>;
  // WorkcostBillQuotaAttach 挂接定额（同清单同定额幂等更新；quantity≤0=跟随清单工程量）。
  WorkcostBillQuotaAttach(billItemId: number, quotaCode: string, quantity: number): Promise<WorkcostBillQuotaLink>;
  // WorkcostBillQuotaDetach 解挂一条定额引用。
  WorkcostBillQuotaDetach(billItemId: number, linkId: number): Promise<void>;
  // WorkcostProjectParse 解析测算工作簿（只解析不落库；硬契约只有综合单价表）。
  WorkcostProjectParse(path: string): Promise<WorkcostProjectBundle>;
  // WorkcostProjectApply 把解析结果落库（工料机价格→资源库，综合单价定额块→定额库）。
  WorkcostProjectApply(path: string): Promise<WorkcostApplyProjectResult>;
  // （WorkcostProjectExport / WorkcostProjectExportToWorkspace 已随旧版渲染器删除：
  //   它产出旧「行类」版式、导不回；测算样表走 CostLibraryExport kind=template。）
}
