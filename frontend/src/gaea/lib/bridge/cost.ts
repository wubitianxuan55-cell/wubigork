// cost.ts — CostBindings（AppBindings 分域接口之一，Go CostB 门面）：
// 成本库/价格源/组价/询价飞轮/五算对比/造价参考与复盘笔记域。
import type {
  CostAdjustSuggestion,
  CostAttribution,
  CostCategory,
  CostCompareRow,
  CostComposeRecord,
  CostComposeView,
  CostEntry,
  CostEstimateItem,
  CostEstimateVersion,
  CostImportPreview,
  CostIndicator,
  CostInquiryRecord,
  CostInquiryScanFinding,
  CostProject,
  CostProjectSummary,
  CostReviewNote,
  CostStageCompareRow,
  CostStageDeviation,
  CostStageValue,
  CostSummary,
  PriceFetchRecord,
  PriceHistory,
  PriceSource,
  SemanticHitView,
  SemanticIndexStatus,
  TaskView,
} from "../types";

export interface CostBindings {
  // ── 成本库 ──
  CostList(): Promise<CostSummary[]>;
  CostSearch(query: string, category: string, status: string): Promise<CostSummary[]>;
  CostGet(name: string): Promise<CostEntry | null>;
  CostSave(e: CostEntry): Promise<void>;
  CostDelete(name: string): Promise<void>;
  // CostImportPreview 解析 xlsx/csv 报价单/测算表为可确认的成本条目。
  CostImportPreview(path: string): Promise<CostImportPreview>;
  // CostImportAIParse 用办公功能模型把表格行归一化为成本条目（AI 解析）。
  CostImportAIParse(path: string): Promise<CostImportPreview>;
  // CostImportApply 批量写入确认后的成本条目，返回成功条数。
  // v4.6：inquirySource 非空（如 "OCR报价"）时，报价单行自动幂等写入询价库
  // （PDF/图片报价单飞轮反向接线）。
  CostImportApply(rows: CostEntry[], inquirySource?: string): Promise<number>;
  // CostImportVisionPreview 解析 PDF（文字型/扫描件 OCR）/图片报价单为可确认的
  // 成本条目；preview.source 标注识别来源（pdf_text / pdf_scan / image）。
  CostImportVisionPreview(path: string): Promise<CostImportPreview>;
  // 多级分类：分类树（含计数）、新建/改名、删除（有子节点或条目时拒绝）。
  CostCategories(): Promise<CostCategory[]>;
  CostCategorySave(parentId: number, name: string, sort: number, id: number): Promise<number>;
  CostCategoryDelete(id: number): Promise<void>;
  // ── 价格源（定时抓取价格更新）──
  PriceSources(): Promise<PriceSource[]>;
  PriceSourceSave(src: PriceSource): Promise<void>;
  PriceSourceDelete(id: string): Promise<void>;
  // PriceFetch 立即抓取价格源（异步任务，T5-1）：提交任务入队立即返回任务
  // 视图，进度/完成经 gaea-task 事件推送；完成后读 PriceFetches 取 pending 记录。
  PriceFetch(id: string): Promise<TaskView>;
  // PriceFetchAll 一键抓取全部启用的价格源（异步任务），逐源进度事件推送，
  // 失败明细在任务结果/消息里。
  PriceFetchAll(): Promise<TaskView>;
  PriceFetches(): Promise<PriceFetchRecord[]>;
  // PriceFetchApply 确认发布抓取结果（按标题选择），返回写入条数。
  PriceFetchApply(fetchId: string, titles: string[]): Promise<number>;
  PriceFetchIgnore(fetchId: string): Promise<void>;
  // PriceHistory 返回某成本条目的价格历史（新→旧）。
  PriceHistory(name: string): Promise<PriceHistory[]>;
  // SemanticSearch 跨库统一语义检索（成本/知识/办公记忆，本地 bge-m3）。
  // SemanticIndexStatus 语义索引覆盖状态（成本条目总数/已覆盖/模型可用性）。
  SemanticIndexStatus(): Promise<SemanticIndexStatus>;
  // SemanticIndexBackfill 显式补齐：缺失或正文变化才重嵌（幂等）。
  SemanticIndexBackfill(): Promise<{ total: number; updated: number; indexed: number; missing: number }>;
  SemanticSearch(query: string): Promise<SemanticHitView[]>;
  // CostCompare 返回某成本条目的多来源比价明细（现价/历史/价格源抓取候选）。
  CostCompare(name: string): Promise<CostCompareRow[]>;
  // ── 测算项目与沉淀闭环（我的项目/工程量清单/版本留痕 → 沉淀回成本库）──
  CostProjectSave(p: CostProject): Promise<string>;
  CostProjectList(): Promise<CostProjectSummary[]>;
  CostProjectGet(id: string): Promise<CostProject | null>;
  CostProjectDelete(id: string): Promise<void>;
  CostEstimateItemSave(i: CostEstimateItem): Promise<number>;
  CostEstimateItemDelete(id: number): Promise<void>;
  CostEstimateItems(projectId: string): Promise<CostEstimateItem[]>;
  CostEstimateVersionSave(projectId: string, note: string): Promise<CostEstimateVersion>;
  CostEstimateVersions(projectId: string): Promise<CostEstimateVersion[]>;
  // CostEstimateSediment 沉淀选中明细行回成本库（UPSERT cost_entries），返回条数。
  CostEstimateSediment(projectId: string, itemIds: number[]): Promise<number>;
  // ── 造价参考与复盘笔记（案例指标 + 经验沉淀）──
  // CostIndicators 造价参考指标：group=title（按科目）| category（按一级分类）。
  CostIndicators(group: string): Promise<CostIndicator[]>;
  // CostAttribution 归因对标（v4.6.1）：项目明细 vs 参考指标带宽，产出
  // 差幅等级/贡献金额/主因 TopDrivers（参考池排除本项目）。
  CostAttribution(projectId: string): Promise<CostAttribution>;
  CostNoteSave(n: CostReviewNote): Promise<number>;
  CostNoteList(query: string, status: string): Promise<CostReviewNote[]>;
  CostNoteDelete(id: number): Promise<void>;
  CostNoteBumpRef(id: number): Promise<void>;
  // CostGraph 成本知识图谱（v4.8）：scope=tree 分类聚合总览（默认）| entry 条目
  // 展开（focus=分类路径或项目 ID）；limit=节点上限（<=0 或 >600 归一 600）。
  // 返回 JSON 串（CostGraphView），由前端 JSON.parse（与 CostImportApply 等先例
  // 不同：该视图含大量节点/边，走字符串通道避免绑定层结构映射开销）。
  CostGraph(scope: string, focus: string, limit: number): Promise<string>;
  // ── v4.2 造价 AI 化：AI 组价 + 询价飞轮 + 五算对比 ──
  // CostCompose AI 组价：清单描述 → 相似检索（关键词+语义）→ 价格带推荐 +
  // 证据链 + LLM 人材机拆解；band=null 表示成本库无相似条目。无确认不落库。
  CostCompose(desc: string, unit: string): Promise<CostComposeView>;
  // CostComposeApply 确认组价建议并回写成本库（UPSERT），返回条目 name。
  CostComposeApply(v: CostComposeView): Promise<string>;
  // CostComposeRecords 组价确认记录回看（v4.158 复核闭环）：entryName 非空 =
  // 该条目全部记录（createdAt 倒序）；空串 = 全库最近 20 条。snapshot 为确认
  // 组价时的完整视图（含 band/recommendedPrice/components/evidence/checks）。
  CostComposeRecords(entryName: string): Promise<CostComposeRecord[]>;
  // ── 询价飞轮（四源归一数据点：信息价/OCR报价/供应商比价/手动询价）──
  // CostInquiryScan 库级异常扫描（只读）：离散/跳变/过期未标记/陈旧四类发现。
  CostInquiryScan(): Promise<CostInquiryScanFinding[]>;
  CostInquirySave(r: CostInquiryRecord): Promise<number>;
  CostInquiryList(query: string, limit: number): Promise<CostInquiryRecord[]>;
  CostInquiryDelete(id: number): Promise<void>;
  // CostInquiryExpiring 到期预警：valid_until <= today+days 的数据点。
  CostInquiryExpiring(days: number): Promise<CostInquiryRecord[]>;
  // CostInquiryAdjust 调差建议：成本库条目 vs 最新询价数据点（|差幅|>2%）。
  CostInquiryAdjust(): Promise<CostAdjustSuggestion[]>;
  // ── 五算对比（估/概/预/结/决）──
  CostStageSave(v: CostStageValue): Promise<void>;
  CostStages(projectId: string): Promise<CostStageValue[]>;
  CostStageCompare(projectId: string): Promise<CostStageCompareRow[]>;
  CostStageDeviations(projectId: string): Promise<CostStageDeviation[]>;
}
