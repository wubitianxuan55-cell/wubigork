# gaea 造价域现状盘点（2026-09-09，file:line 口径）

> 定位：v4.158.0「AI 组价复核闭环」的前置调研存档，供后续造价刀路的现状基线。**重要校正：roadmap §15 与 2026-08 审计中的部分造价「欠账」描述已过时**——AI 组价 v4.2 已在产、语义检索三层栈已建、询价异常检测/预测已存在。引用欠账前先对本文核对。

## 已兑现（勿再当欠账）

| 旧表述 | 实际 |
|---|---|
| 「AI 组价是路线图现在项，未启动」 | **v4.2 已落地在产**：清单描述→分类定位→相似检索(关键词+语义+精排)→价格带推荐(P25-P75+置信度+证据链)→LLM 人材机拆解→确认回写。`internal/app/gaea_cost_compose.go`、`internal/gaea/cost/priceband.go`、`ComposeModal.tsx`（v4.158 增 checks/留痕） |
| 「成本条目无嵌入索引」 | 检索三层栈已建：SQL+BM25（cost.go:275-368 Search）→语义向量召回（gaea_cost_rerank.go:76 semanticCostRecall，semantic_vectors 表 V5）→本地 reranker 精排（:129）；Recall@10≥0.8 测评框架在 gaea_retrieval_eval.go + docs/retrieval-eval-set.md（12 条查询） |
| 「询价无异常检测/预测」 | costinquiry.go:364 adjustLevel 三档(<5%/5-15%/>15%)、:292 SuggestAdjustments 调差、:416 PredictNext 线性回归预测；OCR 报价→询价库幂等飞轮 v4.6 已接（gaea_cost_import.go:314-370） |

## 数据模型（SQLite Hephaestus.db，schema V1-V16）

- 成本条目 `cost_entries`（V2）：name/title/category(_path)/unit/price + 人材机三费 + 费率四项（仅展示追溯不参与计算）+ **溯源五元组** Source/Region/PriceDate/PriceType/ValidUntil + SourceRow + Components 二级表（cost.go:21-65）；分类树 V7、价格三要素 V9、测算项目/版本快照 V10、**组价记录 V16**（v4.158）。
- 测算项目 costproject（明细 Item 引用 EntryName 弱关联+不可变版本快照）；五算 coststage（估概预结决五阶段静态值+对比+偏差档位）；询价库 costinquiry（四源归一+幂等键）；价格源 pricefeed（抓取 Candidate+Anomaly，无确认不写库）。

## 关键 Seam

- 检索/向量：`internal/gaea/retrieval/embed.go` NewEmbedder + `rerank.go` NewReranker + `internal/gaea/semantic`（Store.Open/Search）。
- LLM 范式：成本域统一 `routeSensitiveLocal`（敏感域本地优先，model_router.go:82）→ `provider.NewLLM().Stream()` 聚合（Temperature=0+JSON 约束；在产先例=组价拆解 composeLLMDecompose:147、导入解析 GaeaCostImportAIParse）。
- 归因对标：costref.ComputeAttribution（P25/P75 带宽+贡献金额+Top10，标题精确匹配）+ GaeaCostGraph 知识图谱。
- 绑定面：CostB 43 个（GaeaCost* 34+GaeaPrice*9；v4.158 后 +GaeaCostComposeRecords）。

## 现存缺口（后续刀路池，按价值排序）

1. **条目匹配键弱**：全靠标题字符串精确/子串匹配，无定额/清单编码体系（组价检索与归因对标的共同上游）。
2. **组价流式/分步交互**：LLM 拆解一次性返回（无打字机/分步确认）；多方案（不同分位/费率政策）对比无。
3. **五算与版本快照不贯通**：阶段值手填，不从预算版快照/组价结果带入。
4. **询价库级异常扫描报表**：异常检测只作用于单点调差建议；预测单变量线性回归无置信区间。
5. **检索索引被动维护**：仅检索时按需增量向量化，无后台任务；测评集 12 条偏材料/台班，清单级综合单价描述覆盖少。
6. **人材机含量合理性基线**：v4.158 已上规则校验（金额一致性/非正值/合计偏差）；与相似条目含量的对照基线（行业含量区间）未建。

## 边界（材料立场，维持不做）

不做算量赛道（QuantifAI/Togal 重资源内卷）；不做企业级定额引擎/云协同；费率入库仅展示追溯不参与计算；「无感建库」个人版属远期。
