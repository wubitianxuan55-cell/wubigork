// cost.ts — CostBindings（AppBindings 分域接口之一，Go CostB 门面）。
//
// 2026-11-25 造价数据库推倒重建后的口径：板块是三级浏览面（清单 / 定额 /
// 工料机）——读三级数据、现算综合单价、删除、导出、体检，以及 2026-11-26 接入的
// 语义索引（状态 / 补齐 / 清单检索）；2026-11-27 补上库页编辑面（直接修正录入
// 数据）：清单行与定额挂接、定额含含量行、工料机与调价、项目口径与费率。
// 走 store 的既有保存入口，语义索引打脏标记随写自动生效。批量导入 / 对标 /
// 询价 / 资料条目仍归办公「造价」运行模式（cost_library / cost_project_apply）。
//
// 口径：综合单价 = Σ(消耗量×资源价)，**只含人材机**；管理费/利润/税只在项目
// 合计层跑一次。外委独立行类但并入材料桶。
import type {
  CostIndexStatusView,
  SemanticHitView,
  WorkcostBillItem,
  WorkcostBillProject,
  WorkcostBillQuotaLink,
  WorkcostCompose,
  WorkcostComposeOverride,
  WorkcostLibraryFilter,
  WorkcostQuota,
  WorkcostRateSet,
  WorkcostResource,
  WorkcostResourcePrice,
  WorkcostUsageRef,
} from "../types";

export interface CostBindings {
  // ── 造价数据库（三级浏览面）──
  // CostHealthScan 数据健康六类扫描（零价/缺价子目/悬挂引用/重复防护/编码混用/
  // 不合规矩的数据），只读。
  CostHealthScan(): Promise<Record<string, unknown>>;
  // CostLibraryExport 列表导出 xlsx：kind=bill 清单｜quota 定额（主表+含量明细附表）｜
  // resource 工料机｜template 测算样表。filter 与板块列表的当前筛选同口径（专业/分部/
  // 章节/类别/分类/地区/期数，见 WorkcostLibraryFilter），outPath 为目标 .xlsx 完整
  // 路径（已存在则拒绝覆盖）。返回写出路径。
  CostLibraryExport(
    kind: string,
    filter: WorkcostLibraryFilter,
    outPath: string,
  ): Promise<string>;
  // WorkcostResourceList 工料机列表（kind/keyword 可空 = 不过滤）。
  WorkcostResourceList(kind: string, keyword: string): Promise<WorkcostResource[]>;
  // WorkcostResourceGet 按 id 读工料机；不存在返回 null。
  WorkcostResourceGet(id: number): Promise<WorkcostResource | null>;
  // WorkcostResourceDelete force=false 时拒绝删除仍被定额引用的资源。
  WorkcostResourceDelete(id: number, force: boolean): Promise<void>;
  // WorkcostResourcePrices 某资源的调价历史（新→旧）。
  WorkcostResourcePrices(id: number): Promise<WorkcostResourcePrice[]>;
  // WorkcostResourceUsage 某工料机被哪些定额含量行引用（三级下钻的回链）。
  WorkcostResourceUsage(code: string): Promise<WorkcostUsageRef[]>;
  // WorkcostQuotaGet 按编码读定额（含含量行）；不存在返回 null。
  WorkcostQuotaGet(code: string): Promise<WorkcostQuota | null>;
  // WorkcostQuotaList 定额列表（specialty/keyword 可空 = 不过滤；不含含量行）。
  WorkcostQuotaList(specialty: string, keyword: string): Promise<WorkcostQuota[]>;
  // WorkcostQuotaDelete 删除定额及含量行；仍被清单项引用时拒绝（不留悬空引用）。
  WorkcostQuotaDelete(code: string): Promise<void>;
  // WorkcostQuotaCompose 按定额核算综合单价（只含人材机）。overrides 支持项目级
  // 覆盖含量/单价（key=资源编码；负值表示不覆盖该项）。
  WorkcostQuotaCompose(
    code: string,
    overrides: Record<string, WorkcostComposeOverride> | null,
  ): Promise<WorkcostCompose>;
  // WorkcostQuotaComposeMany 批量核算（清单参考价列专用）：一次资源索引复用，
  // key=定额编码；不存在的编码不在结果里。
  WorkcostQuotaComposeMany(codes: string[]): Promise<Record<string, WorkcostCompose>>;
  // WorkcostBillProjects 清单项目列表（含封面、费率与清单项数）。
  WorkcostBillProjects(): Promise<WorkcostBillProject[]>;
  // WorkcostBillItems 某项目的分部分项清单（quotaCodes 为挂接定额码聚合）。
  WorkcostBillItems(projectId: number): Promise<WorkcostBillItem[]>;
  // WorkcostBillItemDelete 删除一条清单项（定额挂接引用随行清理）。
  WorkcostBillItemDelete(id: number): Promise<void>;
  // WorkcostBillProjectDelete 整体删除项目（封面+清单+独占定额；共享定额与
  // 资源保留）。返回删除的独占定额数。
  WorkcostBillProjectDelete(id: number): Promise<number>;
  // WorkcostBillSearch 清单行跨项目检索：带关键词走混合检索（编码精确优先，相关度序，
  // 含同义说法召回）；不带关键词按列表顺序返回前 200 条。projectId 非 0 时只看该项目。
  WorkcostBillSearch(keyword: string, specialty: string, projectId: number): Promise<WorkcostBillItem[]>;

  // ── 库页编辑面（2026-11-27：直接修正录入数据，不用绕回造价模式）──
  // WorkcostResourceSave 保存工料机（同身份更新、新身份插入；id>0 定点改那行）。
  // 返回落库后的资源。身份（类别+名称+规格+单位+地区）撞别行时 reject。
  WorkcostResourceSave(r: WorkcostResource): Promise<WorkcostResource>;
  // WorkcostResourceSetPrice 登记一次调价（写调价历史 + 推进现行价）。期数归一
  // YYYY-MM；地区须与该行的归属地区一致（或空）。
  WorkcostResourceSetPrice(
    id: number,
    price: number,
    period: string,
    region: string,
    priceType: string,
    source: string,
    note: string,
  ): Promise<WorkcostResource>;
  // WorkcostQuotaSave 保存定额（含含量行整组替换；items 里资源编码空缺的元数据
  // 由后端按资源索引补齐）。专业/地区须在字典里，期数归一 YYYY-MM。
  WorkcostQuotaSave(q: WorkcostQuota): Promise<WorkcostQuota>;
  // WorkcostBillItemSave 保存一条清单项（同项目同编码更新；sort=0 保持原序，
  // specialty 空=保持原值）。
  WorkcostBillItemSave(item: WorkcostBillItem): Promise<WorkcostBillItem>;
  // WorkcostBillQuotaLinks 清单项挂接的定额引用（1:N）。
  WorkcostBillQuotaLinks(billItemId: number): Promise<WorkcostBillQuotaLink[]>;
  // WorkcostBillQuotaAttach 挂接定额（同清单同定额幂等更新工程量；0=跟随清单）。
  WorkcostBillQuotaAttach(billItemId: number, quotaCode: string, quantity: number): Promise<WorkcostBillQuotaLink>;
  // WorkcostBillQuotaDetach 解挂一条定额引用（linkId 须归属该清单项）。
  WorkcostBillQuotaDetach(billItemId: number, linkId: number): Promise<void>;
  // WorkcostBillProjectRatesSave 保存项目费率（四费率 + 利润基数含规费开关 + 控制价）。
  WorkcostBillProjectRatesSave(
    id: number,
    rates: WorkcostRateSet,
    profitIncludesRegulatory: boolean,
    controlPrice: number,
  ): Promise<void>;
  // WorkcostBillProjectProfileSave 保存项目口径（专业/地区/计价期；空=保持原值，
  // 规范值须在字典里、期数归一 YYYY-MM）。
  WorkcostBillProjectProfileSave(id: number, specialty: string, region: string, pricePeriod: string): Promise<void>;

  // ── 语义索引（2026-11-26 造价库接入本地模型索引）──
  // CostIndexStatus 造价库语义索引的状态（三类各自：条数 / 已建向量数 / 模型 / 最近错误）。
  // 只读，不探测模型。
  CostIndexStatus(): Promise<CostIndexStatusView>;
  // CostIndexBackfill 显式补齐索引（缺的补、正文变了的重嵌）；kind 空 = 全部；返回补齐后的
  // 状态。模型不可用时 reject，不静默假装补齐。
  CostIndexBackfill(kind: string): Promise<CostIndexStatusView>;

  // ── 语义检索（Go 侧 GaeaSemanticSearch）：声明挂在造价分域，检索链统一消费 ──
  SemanticSearch(query: string): Promise<SemanticHitView[]>;
}
