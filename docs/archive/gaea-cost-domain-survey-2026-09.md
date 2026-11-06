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

> **2026-09-10 状态：六项全部收口**（v4.178/v4.194/v4.195/v4.196/v4.204/v4.209）。逐项落地记录如下，新缺口另立调研、不在本文追加。

1. **条目匹配键弱**（v4.178.0 首刀落地：SchemaV17 code 列 + NormalizeCode 归一化 + 导入「定额编号/清单编码」列映射 + 编码优先匹配[带码未命中宁新增勿误配] + costref 归因 matchedBy=code + 工具面/UI 贯通；存量条目编码等带码样本导入自然补全，不做标题猜测回填）：全靠标题字符串精确/子串匹配的缺口已闭合主链，编码维度后续按使用反馈加深。
2. **组价流式/分步交互**（v4.204.0 半刀：多方案对照已落地——ComposeModal P25/中位/P75 点选推荐档+cost_compose `mode` 参数+三档对照行；**不**拆逐步盖章。流式打字机/费率政策对照仍不做：前者规划禁止盖章化，后者费率入库仅展示追溯不参与计算）。
3. **五算与版本快照不贯通**（✅ v4.194.0 收口：FiveCalcPanel 每阶段「带入」钮——测算项目不可变版本快照合计一键带入，确认制+备注溯源「带入:〈项目名〉v〈n〉」，(project_id,stage) UPSERT 天然覆盖旧值；**组价结果带入明确不做**——组价=单条目单价、阶段值=项目总量，量纲不同，混进五算造错账，组价进测算项目走既有明细行引用链路）。原文「阶段值手填」的缺口即此刀闭合。
4. **询价库级异常扫描报表**（✅ v4.195.0 收口：GaeaCostInquiryScan + CostInquiryPanel「⑥ 库级扫描」——同标题离散/相邻期跳变/有效期已过仍现行/最新期数超一年 四类体检全量扫，severity+人话 detail+refIds 定位）。**预测置信区间仍欠**（单变量线性回归无区间估计），等真实样本再评估。
5. **检索索引被动维护**（✅ v4.196.0 收口：semantic.Store.Coverage 内容口径覆盖判定[向量在且正文快照一致] + GaeaSemanticIndexBackfill 显式补齐[10 分钟预算，与搜索路径共用 Ensure 幂等] + CostLibraryView 索引 chip 三态[全覆盖/部分可补/未启用]；测评集补 13-15 清单级综合单价查询）。
6. **人材机含量合理性基线**（✅ v4.209.0 收口：cost.CheckContentBaseline 纯函数——拆解组件含量 vs 相似条目池同标题同单位含量分位带[P25/P75，R-7]，带外 warn、带内静默、同值退化带 ±5% 容差、样本 <3 不比对、标题/单位归一精确匹配[一方缺失宁缺勿误]；接进 GaeaCostCompose Checks 与金额自洽结论同列，**零新结构零新绑定**）。**外部「行业含量区间」数据源明确不引入**——基线源=用户自己的库（私人记忆哲学），样本数即诚实口径；若未来有可靠行业数据源再评估另刀。

## 边界（材料立场，维持不做）

不做算量赛道（QuantifAI/Togal 重资源内卷）；不做企业级定额引擎/云协同；费率入库仅展示追溯不参与计算；「无感建库」个人版属远期。
