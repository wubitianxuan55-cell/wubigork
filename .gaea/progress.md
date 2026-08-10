# 任务进度

> 最后更新: 2026-08-10 12:35:00

## 开发中（v2.11.0 方向）：索引新鲜度 + pptx 提取 + 自动重建 + 总结文档

| 状态 | 任务 |
|------|------|
| ✅ | semantic.Ensure 内容感知：向量缺失或正文快照变化才重嵌（编辑过的条目自动刷新，修复「改内容仍被旧向量命中」的潜在问题） |
| ✅ | fileindex 支持 pptx：zip + slide XML 提取 <a:t> 文本（无第三方依赖），资料语义检索覆盖 PPT |
| ✅ | 文件索引自动维护：startFileIndexCron 启动即查 + 每 10 分钟增量重建（Ensure 内容感知 + Stale 清理），embedding 不可用静默跳过 |
| ✅ | 总结文档 docs/2026-08-office-knowledge-stack-optimization-summary.md：四大库能力矩阵、检索分层定稿、设计原则、验证状态、后续候选 |
| ✅ | 测试：Ensure 内容变更重嵌、pptx 提取；前端 111 例全绿，go test 全量 + vet 干净，tsc + vite build 通过 |

### 说明
- 自动维护采用 10 分钟轮询（无新依赖）；fsnotify 实时监听列为后续候选

## 开发中（v2.11.0 方向）：工作区文件语义索引（资料语义检索）

| 状态 | 任务 |
|------|------|
| ✅ | fileindex 包：扫描工作区支持的文本文件（md/txt/csv/docx/xlsx/pdf，≤2MB、≤300 个，跳过 .git/node_modules/.gaea 等），提取正文（docmd 转换 + 截断 2 万字符） |
| ✅ | 索引：复用 semantic_vectors（kind=file，id=相对路径），Ensure 增量向量化 + Stale 清理删除文件；GaeaFileIndexRebuild / GaeaFileSemanticSearch 绑定 |
| ✅ | 前端：搜索面板「语义」开关（本地 bge-m3 语义命中区，含相似度 % 与摘要，可预览/@引用）+「重建索引」按钮与结果提示；记忆中枢统一搜索并入「语义·文件」命中 |
| ✅ | 测试：fileindex 扫描/跳过/截断、App 重建+语义检索端到端（临时工作区 + fake embed）、搜索面板语义/重建索引用例；前端 111 例全绿，go test 全量 + vet 干净，tsc + vite build 通过 |

### 说明
- 资料语义检索与成本/知识/办公记忆共用同一向量表与 bge-m3，检索分层完整：文件关键词全文（wssearch）→ 语义命中（bge-m3）→ 可 @ 引用
- 后续候选：索引增量监听（文件变更自动重建）、pptx 文本提取支持

## 开发中（v2.11.0 方向）：办公记忆查重/合并

| 状态 | 任务 |
|------|------|
| ✅ | textsim 共享包：CJK 二元组集合 Dice 相似度（knowledgeimport 重构委托，办公记忆共用） |
| ✅ | GaeaMemoryDuplicates：办公记忆两两查重（描述+正文相似度，阈值 0.55，建议保留较早一项），按相似度降序 |
| ✅ | GaeaMemoryMerge：标签并集 + 来源事实描述追加「合并自」标记到目标正文 + 来源删除，返回目标名 |
| ✅ | 前端：OfficeMemoryLibrary 工具条「查重合并」按钮 → 弹窗列出疑似重复对（相似度 %）+ 单对合并/全部合并 + 结果提示 |
| ✅ | 测试：查重对（a⇄b）、合并（标签并集/来源删除/正文标记）；前端 111 例全绿（新增查重合并用例），go test 全量 + vet 干净，tsc + vite build 通过 |

### 说明
- 自动做梦/多会话会产生同名变体事实，查重合并按对处理（保留较早项），来源删除前无独立历史（facts 短、可溯源到 sourceSession）
- 后续候选：项目资料/材料的文件级语义索引（工作区文件向量化），或聊天记忆（轻语）查重

## 开发中（v2.11.0 方向）：知识库审核流 + 相似条目一键合并（P1 收尾）

| 状态 | 任务 |
|------|------|
| ✅ | GaeaKnowledgeReview：草稿 → 现行 的审核通过动作（可填审核人），动作留档版本历史；驳回保留原状态 |
| ✅ | GaeaKnowledgeMerge：相似条目合并进目标条目（标签并集、来源合并、来源条目留档「合并至」后删除、目标留档「合并自」） |
| ✅ | 前端：展开详情「审核通过」（仅草稿显示）、「合并相似…」弹窗（按相似度列出候选勾选合并）、合并结果提示 |
| ✅ | 测试：审核通过（状态/审核人/留档）、合并（标签并集/来源删除/双方留档）；前端 110 例全绿，go test 全量 + vet 干净，tsc + vite build 通过 |

### 说明
- 单用户本地工具：审核流是「草稿 → 现行」的确认发布动作，reviewer 字段可编辑，动作可溯源
- 知识库大优化闭环：导入/AI 提取 → 管理/批量 → 查重/合并 → 审核/版本 → 检索分层（语义召回+BM25+精排）→ 导出

## 开发中（v2.11.0 方向）：知识库去重/版本历史/批量导出（P1）

| 状态 | 任务 |
|------|------|
| ✅ | 查重：knowledgeimport.Similarity（CJK 二元组集合 Dice）+ FindSimilar + 导入预览相似条目标记（≥55% 提示「与「xxx」相似 N%，建议合并」） |
| ✅ | 版本历史：SchemaV6 knowledge_history 表；saveKnowledgeVersioned 内容变化时旧快照留档 + 版本号自增（面板保存与导入覆盖共用）；GaeaKnowledgeHistory 查询 |
| ✅ | 批量导出：GaeaKnowledgeExport 全部条目导出为 Markdown（frontmatter + 正文），默认 .gaea/exports/knowledge-<日期> |
| ✅ | 前端：KnowledgePanel 新建/编辑防抖查重提示（疑似重复 + 相似度）、展开详情「版本历史」弹窗、标题栏「导出」按钮；KnowledgeImportModal 相似条目红色徽标 |
| ✅ | 测试：相似度/查重/导入相似提示、版本自增与历史留档、导出文件；前端 110 例全绿，go test 全量 + vet 干净，tsc + vite build 通过 |

### 说明
- 版本历史仅在内容实际变化时留档（同名同正文不产生新版本），无噪音
- 后续候选：审核流（reviewer 确认后发布）、相似条目一键合并

## 开发中（v2.11.0 方向）：知识库大优化（导入/AI 提取/检索升级/批量管理）

| 状态 | 任务 |
|------|------|
| ✅ | knowledgeimport 包：md/txt 直接入库（标题=文件名、分类关键词猜测）、docx/pdf 走 docmd 提取正文、xlsx/csv 表头自动映射（标题/分类/阶段/专业/标签/来源/正文）；按标题匹配既有条目标「新增/将覆盖更新」 |
| ✅ | App 绑定：GaeaKnowledgeImportPreview / GaeaKnowledgeImportAIParse（office 模型多主题拆分结构化，body 保留要点）/ GaeaKnowledgeImportApply（批量落库）；bridge/mock/types 同步 |
| ✅ | 知识检索升级：knowledge_search 工具 + GaeaKnowledgeSearch 接入「关键词不足→bge-m3 持久化语义召回（复用 semantic_vectors）→ BM25/原排序 → >8 条 bge-reranker-v2-m3 精排」，失败自动回退 |
| ✅ | 前端：KnowledgeImportModal（勾选/修正/覆盖提示/AI 解析/确认导入）；KnowledgePanel 加导入入口 + 多选批量删除/批量改状态 |
| ✅ | 测试：knowledgeimport md/xlsx/覆盖匹配/slug 单测、知识语义召回集成；前端 110 例全绿（新增知识导入 2 例、面板批量/导入 2 例），tsc + vite build 通过，go test 全量 + vet 干净 |

### 说明
- 与成本库同套路：无确认不落库、来源可溯、检索分层（子串→语义召回→BM25→rerank）
- P1 候选：知识条目版本/审核流、相似条目去重合并提示、批量导出

## 开发中（v2.11.0 方向）：持久化向量索引 + 跨库统一语义检索（P1）

| 状态 | 任务 |
|------|------|
| ✅ | SchemaV5：semantic_vectors 共享表（(kind,id) 主键，vec JSON，doc 快照，updated_at） |
| ✅ | semantic 包：Ensure 增量向量化（分批 64，只处理缺失项）、SearchReady（只嵌 query + 余弦 topN）、SearchMany（多 kind 合并降序）、Stale（条目删除后清理向量） |
| ✅ | 成本语义召回改为持久化索引：cost_search 工具 + App GaeaCostSearch 走 semantic.Store，入库即增量写向量，查询只嵌 query（替代每查询全量批量，候选库规模不再影响单次成本） |
| ✅ | GaeaSemanticSearch 跨库统一语义检索：成本/知识/办公记忆三库合并（每库 top6 → 合并 top20），本地 bge-m3 零 token |
| ✅ | 前端：记忆中枢统一搜索接入语义命中（「语义·成本/知识/办公」标签，复用现有命中渲染）；bridge/mock/types 同步 |
| ✅ | 测试：semantic 包（增量 Ensure/搜索/SearchMany/Stale）、工具语义召回集成、真实 Herdsman bge-m3 持久化链路端到端；go test 全量 + vet 干净，前端 106 例全绿，tsc + vite build 通过 |

### 说明
- 检索分层最终形态：SQL 过滤 → 子串召回 →（不足时）bge-m3 持久化语义召回 → BM25 排序 →（>8 时）bge-reranker-v2-m3 精排；跨库统一语义检索复用同一向量表

## 开发中（v2.11.0 方向）：bge-m3 本地语义召回 + 架构方向确认

| 状态 | 任务 |
|------|------|
| ✅ | 实测 Herdsman bge-m3：/v1/embeddings 可用（1024 维、支持批量） |
| ✅ | retrieval.Embedder（OpenAI 兼容 /v1/embeddings，模型探测缓存、60s 超时、余弦相似度）+ SemanticRecall（关键词召回不足时按余弦补召回，排除已有结果，纯本地零 token） |
| ✅ | cost_search 工具 + App GaeaCostSearch 接入：关键词召回 <3 时自动触发本地语义召回（候选库 ≤500 批量向量化，更大走 P1 持久化索引），embedding 不可用/失败回退 |
| ✅ | 真实 Herdsman 端到端：「打桩设备 台班价」无关键词命中 → bge-m3 语义召回 HP300 液压振动锤 |
| ✅ | 架构方向确认（写入 docs/2026-08-office-cloud-local-architecture-review.md §6）：尽量免费（云端免费可用）、出图 XAI 优先再 Herdsman、不固化「常规→本地」路由（交给云端模型按需调度本地工具）、bge-m3 语义召回接入 |
| ✅ | 测试：embed 批量/余弦/语义召回/排除已有/不可用降级、工具语义召回集成；go test 全量 + vet 干净，前端 106 例全绿，tsc + vite build 通过 |

### 说明
- 检索分层最终形态：SQL 过滤 → 子串召回 →（不足时）bge-m3 语义召回 → BM25 排序 →（>8 时）bge-reranker-v2-m3 精排；全部本地，任一层失败自动降级
- P1：bge-m3 向量持久化索引（入库即向量化，替代每次全量批量）+ 跨库统一语义检索

## 开发中（v2.11.0 方向）：价格异常识别（P2）

| 状态 | 任务 |
|------|------|
| ✅ | pricefeed.DetectAnomalies：对「更新」候选对比最近历史发布价（无历史用现价）计算单期跳幅，±20% 阈值标记异常并给出原因（如「单期跳幅 +25.0%（基准 ¥3,000）」）；新增/无变化不判 |
| ✅ | 手动抓取（GaeaPriceFetch）与定时抓取（tickPriceCron）均接入异常检测，结果随 pending 记录持久化 |
| ✅ | 前端：待确认结果中异常条目显示红色「异常」徽标，悬停显示原因；默认仍可勾选发布（确认权在用户） |
| ✅ | 测试：异常/正常/无基准/新增/无变化判定、历史价优先于现价作基准；前端价格源面板异常徽标用例；go test 全量 + vet 干净，前端 106 例全绿，tsc + vite build 通过 |

### 说明
- 阈值暂定 ±20%（常量 anomalyJumpPct），后续可按材料类别差异化；异常仅提示不阻断，发布仍需用户确认

## 开发中（v2.11.0 方向）：成本库检索 BM25 本地排序层

| 状态 | 任务 |
|------|------|
| ✅ | bm25 包：中英文混合分词（CJK 重叠二元组 + 字母数字词，两字词去重）→ BM25 打分（k1=1.2, b=0.75），纯本地零 token |
| ✅ | cost.Search 接入：子串召回（多词 AND/字段 OR）→ BM25 相关度排序（TF 密度高的排前）→ 无查询词保持 name 序 |
| ✅ | 修复排序覆盖 bug：底部按 name 的最终排序把 BM25/精排顺序覆盖，改为仅空查询时按 name 排 |
| ✅ | 检索分层定型：SQL 分类/状态过滤 → Go 子串召回 → BM25 本地排序 →（候选>8 时）Herdsman bge-reranker-v2-m3 精排，任一层失败自动降级 |
| ✅ | 测试：bm25 分词/排序/无命中单测、cost BM25 顺序断言；go test 全量 + vet 干净，前端 106 例全绿，真实 Herdsman 端到端仍通过 |

### 说明
- BM25 是纯本地词法排序：零 token、零网络，与云端 API 完全无关；后续数据量大也只需换索引实现（FTS5 可选），排序语义不变

## 开发中（v2.11.0 方向）：成本库检索升级 — 本地语义精排（bge-reranker-v2-m3）

| 状态 | 任务 |
|------|------|
| ✅ | 实测 Herdsman：/v1/models 已列出 bge-reranker-v2-m3；/v1/rerank 对「液压振动锤 300kW 台班」正确排序（HP300 > 挖掘机 > 水泥） |
| ✅ | retrieval 包：OpenAI 兼容 /v1/rerank 客户端（模型探测缓存 60s、15s 超时、topN、失败可回退） |
| ✅ | cost_search 工具 + App GaeaCostSearch 接入：SQL/Go 粗召回（>8 条）→ 本地精排 topN → 失败自动回退原结果；纯本地推理，不消耗云端 token |
| ✅ | 修复既有检索 bug：modernc/sqlite 对 6 列 OR LIKE 长链存在返回空集的怪癖（如按单位「台班」搜索 0 命中）；关键词匹配改为 Go 侧精确子串（多词 AND、字段 OR），顺带支持「液压振动锤 台班」这类跨字段多词查询 |
| ✅ | 测试：retrieval 客户端（httptest 排序/可用性/HTTP 错误）、cost_search 精排顺序 + 失败回退、真实 Herdsman 端到端（多词召回 + 连通）；go test 全量 + vet 干净，前端 106 例全绿，tsc + vite build 通过 |

### 说明
- BM25/FTS5 是纯本地词法排序，零 token、零网络；本轮的 rerank 走 Herdsman 也是本地推理，同样不耗云端 API token
- 精排开关式生效：Herdsman 模型在则启用，不在/失败自动回退 SQL；HERDSMAN_BASE_URL（默认 http://localhost:8080）与 HERDSMAN_RERANK_MODEL（默认 bge-reranker-v2-m3）可覆盖

## 开发中（v2.11.0 方向）：成本库价格源 P1（订阅/定时抓取/价格历史）

| 状态 | 任务 |
|------|------|
| ✅ | SchemaV4：price_sources / price_fetch / cost_price_history 三表（Hephaestus.db 迁移链追加） |
| ✅ | pricefeed 包：抓取（浏览器 UA + 自定义 Cookie 头，30s 超时）→ HTML 价格表解析（名称/规格/单位/含税 + 首个地区报价列，价格归一化）→ 与 cost_entries 匹配（更新/新增/无变化 + 差额/环比）；四川真实页面解析 24 行通过、live 抓取验证通过 |
| ✅ | 价格源 CRUD + 抓取记录（pending 待确认）+ 价格历史存储（pricefeed.Store） |
| ✅ | App 绑定：GaeaPriceSources/Save/Delete、GaeaPriceFetch/Fetches/Apply/Ignore、GaeaPriceHistory；bridge/mock/types 同步 |
| ✅ | 定时调度：startPriceCron（启动即查 + 每 30 分钟轮询，frequency_hours>0 且到期的源自动抓取存 pending），Shutdown 停止 |
| ✅ | 前端 PriceSourcesPanel（记忆中枢 + 办公侧共用）：源列表/新建/编辑（含 Cookie 自定义头）/删除、立即抓取、待确认结果（变更高亮 + 环比 + 勾选发布/忽略）；办公侧条目行加「价格历史」弹窗 |
| ✅ | 测试：pricefeed 解析/匹配/httptest 抓取/store CRUD 单测；前端 106 例全绿（新增价格源 2 例），tsc + vite build 通过，go test 全量 + go vet 干净 |

### 说明
- 用户提供的两个源：四川造价信息网（202.61.90.35:8032 period=758）可直连抓取；重庆施工造价信息网（cqsgczjxx.org）是瑞数 JS 挑战站（HTTP 412），需在价格源配置里粘贴浏览器 Cookie，后端已支持自定义请求头
- 抓取结果一律 pending → 用户勾选确认发布（写回 cost_entries + 价格历史快照）或忽略；无确认不写库
- 未做：检索 BM25/rerank（数据量上来后接 Herdsman bge-reranker-v2-m3）、价格异常识别

## 开发中（v2.11.0 方向）：成本库功能深化 P0（导入/AI 提取/管理增强）

| 状态 | 任务 |
|------|------|
| ✅ | 市场调研：docs/market-research-2026-08-office-cost-library-v2.md（广联达/慧讯网/造价通/千问/飞书/行情通/1688 竞品矩阵；检索分级 + bge-reranker-v2-m3 本地精排方案，Herdsman /v1/embeddings+/v1/rerank 已就绪） |
| ✅ | 导入解析：internal/gaea/costimport（xlsx/csv 读取复用 excelize + fileutil/encoding 编码探测；表头关键词自动映射名称/规格/单位/单价/来源/分类；价格归一化 ¥/元/千分位；既有条目匹配标「新增/将覆盖更新」；无确认不落库） |
| ✅ | App 绑定：GaeaCostImportPreview / GaeaCostImportAIParse（office 功能模型流式归一化，复用 GaeaSummarizeFile 的 provider 链路）/ GaeaCostImportApply（批量落库）；bridge/mock/types 同步 |
| ✅ | 共享编辑弹窗 CostEntryModal：记忆中枢 CostLibrary 与办公侧 CostLibraryPanel 共用新建/编辑表单 |
| ✅ | 导入弹窗 CostImportModal：候选条目表格可勾选/修正（名称/分类/单位/单价/规格/来源/状态）、覆盖更新提示、AI 智能解析一键切换、确认后批量入库 |
| ✅ | 管理增强：办公侧成本库 Tab 增加新建/编辑/删除 + 多选批量删除/批量改状态 + 导入入口；cost.Store.Search 补 name/tags 命中 |
| ✅ | 测试：costimport 单测（CSV/xlsx 映射、价格归一化、覆盖匹配、TSV RawTable）+ cost Search 补字段用例；前端 104 例全绿（新增导入弹窗 2 例、批量删除/编辑 2 例），tsc + vite build 通过，go test 全量 + go vet 干净 |

### 说明
- 解析复用通用办公基建：excelize（xlsxpreview 同款）、fileutil/encoding（format_convert 同款）、office 功能模型 provider 链路（GaeaSummarizeFile 同款）；差异仅在「成本条目」的列映射/价格归一化专用层
- PDF/图片报价单导入走通用链（format_convert/OCR → 文本）再 AI 提取，列为 P1
- P1 未做：价格源订阅 + 定时抓取（app 内 ticker 调度）、价格历史表、检索 BM25/rerank（Herdsman 装 bge-reranker-v2-m3 后开关式启用）

## 开发中（v2.11.0 方向）：通用办公 × 记忆中枢「成本库」打通（P0）

| 状态 | 任务 |
|------|------|
| ✅ | 市场调研：docs/market-research-2026-08-office-cost-cards.md（广联达/千问/WPS/腾讯/飞书/M365/Kimi/Notion/ChatGPT/BOQ-AI 竞品矩阵，结论=成本数据独立结构化中枢，双向打通） |
| ✅ | cost_search / cost_save 内置代理工具（builtin/cost_tools.go）：读/写 Hephaestus.db cost_entries（与记忆中枢 CostLibrary 同库），同名 UPSERT、标题自动生成稳定 name、分类概览 + 关键词/分类/状态过滤 + limit；单测通过 |
| ✅ | cost-estimate 模板升级：prompt 引导「先 cost_search 引用历史单价 → 完成后再 cost_save 沉淀（来源标注本次项目/文件）」 |
| ✅ | 办公右侧「成本库」Tab（CostLibraryPanel）：浏览/搜索/筛选成本条目，一键把结构化单价插入输入框（ComposerInsertStore requestText）；编辑仍留在记忆中枢页 |
| ✅ | 产物面板「沉淀到成本库」：xlsx/csv/et/ods 产物悬停出现 Coins 操作，一键把 cost_save 指令插入输入框 |
| ✅ | 记忆中枢概览 CostCount：MemoryHubOverview 增 costCount（后端/类型/mock/首页卡片同步显示） |
| ✅ | 测试：go test 全量 + go vet 干净；前端 100 例全绿（新增成本库面板 2 例、产物沉淀 2 例），tsc + vite build 通过 |

### 说明
- 打通链路：测算前 agent 用 cost_search 查单价 → 完成测算后 cost_save 写回成本库 → 用户侧右侧「成本库」Tab 浏览/一键引用 → 产物面板一键生成沉淀指令
- 术语按用户确认统一为「成本库」（成本卡 → 成本库）
- P1 未做：结构化装配+来源归属、相似名称/别名匹配、测算表自动解析入库；明确不做企业组价/定额引擎、云端同步、无确认自动写入

## 开发中（v2.11.0 方向）：大文件处理 — 实时 OCR 进度 UI

| 状态 | 任务 |
|------|------|
| ✅ | 扫描件 PDF 预览实时进度：GaeaPreview 走 docmd.ConvertLimitProgress，逐页 OCR 经事件通道发布 preview_progress（path/done/total） |
| ✅ | 前端 usePreviewProgress hook：FilePreview / FilePreviewModal 订阅事件，OCR 期间显示「OCR 识别中 x/N 页…」 |
| ✅ | 修复事件契约：progress 内携带 path（此前仅顶层有 path，hook 永远匹配不上——测试暴露的真实 bug） |
| ✅ | 测试：FilePreviewModal 新增扫描件 OCR 进度用例（mock 模拟逐页事件），前端 96 例全绿，tsc + vite build 通过，go test 全量 + go vet 干净 |

### 说明
- 文本型 PDF 不经 OCR，无进度事件（保持原「加载中」）；进度只在扫描件逐页识别时出现
- 进度经既有 gaea-event 通道发布，Wails/HTTP 桥接均生效；测试环境无 Wails 时静默跳过

## 开发中（v2.11.0 方向）：大文件处理 P1 落地

| 状态 | 任务 |
|------|------|
| ✅ | 全文搜索大文件提示（P1-④）：wssearch.Hit 增 Truncated/Skipped——>5MB 文本不索引但文件名命中时返回可见跳过原因（指引 summarize_file）；>20 万字符文档标「索引截断」；搜索面板显示琥珀色提示 |
| ✅ | md/text 预览截断标记（P1-⑤）：readPreviewCapped 2MB 上限 + UTF-8 安全截断 + 可见标记；前端截断横幅改为通用文案（PDF 页数 / 文件过大） |
| ✅ | 多文件「摘要的摘要」（P1-⑥）：largefile.SummarizeFiles 逐文件 map-reduce → 合并总览 pass；summarize_file 工具支持 paths 数组（多文件） |
| ✅ | 资料卡片大文件标注 + 摘要后引用（P1-⑦）：MaterialsPanel 大文件（>5MB）徽标 + Sparkles 摘要按钮 → GaeaSummarizeFile（app 层 bridge provider + largefile）→ 摘要文本插入输入框（ComposerInsertStore requestText/consumeText） |
| ✅ | 测试：wssearch 跳过/截断标记、app 摘要绑定错误路径、largefile 多文件合并单测通过；前端 95 例全绿（新增摘要按钮用例），tsc + vite build 通过，go test 全量 + go vet 干净 |

### 说明
- P1-⑤ 的 docx 预览保持 docx-preview 整读渲染（前端库渲染），不做后端截断，避免丢失格式保真
- PDF 页数预估未在资料列表做（需整读文件成本高）；以体积徽标 + 摘要按钮替代

## 开发中（v2.11.0 方向）：大文件处理 P0 落地

| 状态 | 任务 |
|------|------|
| ✅ | docmd PDF 页数上限（P0-③）：ConvertLimit/ConvertLimitProgress——默认 500 页守卫（DefaultMaxPDFPages），空规格收敛为 1-500、显式 pages 规格裁剪越界段、整体超限报错；返回 total+truncated 供上层标注 |
| ✅ | 分页 OCR（P0-③）：扫描件回退只渲染请求范围（pdftoppm -f/-l），不再整本出图；逐页进度回调 (done,total)；format_convert 超限自动追加截断提示 |
| ✅ | @ 大文件智能策略（P0-②）：office 文档（docx/doc/xls/xlsx/pdf）@ 引用转 Markdown 注入头部（PDF 限前 20 页），替代 "binary not shown"；大文本截断标记追加 summarize_file / read_file offset 指引；UTF-8 安全截断 |
| ✅ | 大文件分块摘要管线（P0-①）：新包 internal/gaea/largefile——ExtractText（office 经 docmd、文本直读 100MB 守卫）+ 段落感知 ChunkText（2 万字符/块）+ map-reduce Summarize（逐块摘要 → 超阈值再「摘要的摘要」合并 pass） |
| ✅ | summarize_file 工具：boot 注入会话 provider 注册（读模型可见）；输出文件/PDF 页数/字符数/分块数与结构化摘要；readonly |
| ✅ | 预览截断标记：PreviewResult 增 truncated/totalPages，超大 PDF 预览正文追加「已截断共 N 页仅前 500 页」提示；前端 FilePreview/FilePreviewModal 显示琥珀色截断横幅 |
| ✅ | 测试：docmd 页数上限/规格裁剪/OCR 进度参数单测、refs @office 注入与截断标记单测、largefile 分块/map-reduce/合并 pass/工具单测、boot 全量通过；前端 94 例全绿，tsc + vite build 通过，go test 全量 + go vet 干净 |

### 说明
- 大文件策略与「@ 按需加载 + 记忆自动做梦」上下文纪律一致：上下文只放头部/摘要，细节按需再读
- 实时 OCR 进度 UI 未接（预览为同步调用）；进度回调已在 docmd 层就绪，后续可挂事件通道

## 调研（2026-08-10）：大文件处理 — 第六轮竞品优点蒸馏

> 产出 docs/market-research-2026-08-office-large-files.md
> 主题：办公板块集中在大文件——体积/页数上限、超限提示、分块摘要、按需读取、流式 OCR

### 蒸馏出的可落地优点（P0）
1. 大文件分块摘要管线（map-reduce）：超过阈值（如 >300 页 / >2MB）的 docx/pdf/txt/md
   先分块摘要再合并成结构化文件摘要（章节/要点/数据表），摘要入上下文、细节按块再读
   ——对标千问 500 页超长文、WPS 读完整本书、豆包分段摘要、M365 摘要指南
2. @ 大文件智能策略：>64KB 引用不再硬截断，改为「注入目录+摘要 / 按需分页分段读取 /
   明确提示超限」，复用 refs.go 注入点——对标 Claude Code grep+offset、WPS 分页 OCR、
   aily 智能切片
3. PDF 页数上限 + 流式分页 OCR：预览/转换默认 300-500 页守卫，扫描件 OCR 按页流式 +
   进度回传 + 可取消——对标 Gemini 1000 页、千问 500 页、MinerU 流式、WPS 分页 OCR

### 蒸馏出的可落地优点（P1）
4. 全文搜索大文件提示与分块索引（>5MB 跳过 / 20 万字符截断改为可见标注 + 按页/chunk 分块索引）
5. 超大预览懒加载 + 截断标记（docx/pdf/md 补 xlsx 已有的 Truncated 标记）
6. 多文件「摘要的摘要」（一次引用多个大文件先逐文件摘要再合并总览）
7. 资料卡片大文件标注（MaterialsPanel 显示大小 + 页数预估 + 「摘要后引用」）

### 明确不做
- 把体积上限放宽到云端级（ChatGPT 512MB / Claude 500MB）——本地工具定位，靠策略解决
- 云端存储 / 跨设备大文件同步（本地优先）
- 大文件全量塞进上下文（坚持摘要 + 按需读取，与 @ 按需加载纪律一致）

## 开发中（v2.11.0 方向）：开工前 P1 深化落地

| 状态 | 任务 |
|------|------|
| ✅ | 工作区全文搜索（轻量 RAG）：新包 internal/gaea/wssearch——扫描工作区 docx/xlsx/pdf/md/txt/csv 等正文（docmd 转办公文档、文本直读、正文按路径+mtime+size 缓存），中文 bigram 分词 + TF-IDF 打分，返回命中片段；噪音跳过 .git/node_modules/dist/build/.cache/.codegraph/.tianxuan/.reasonix/.tmp* 及 .gaea/sessions\|archive\|cache，.gaea/exports 交付产物可索引 |
| ✅ | 前端「搜索」标签页（WorkspaceSearchPanel）：右侧面板第四个标签 + 命令面板「工作区全文搜索」入口，300ms 防抖，命中片段可预览 / 一键 @ 引用 / 外部打开；bridge WorkspaceSearch → GaeaWorkspaceSearch（含 mock） |
| ✅ | 常用资料固定（P1-②）：新包 internal/gaea/pins——清单持久化 <工作区>/.gaea/pinned.json（去重、≤20、防 ../ 逃逸）；GaeaPinnedMaterials/PinMaterial/UnpinMaterial 绑定；资料面板每行图钉 + 顶部「已固定」区（取消固定、新会话自动带入提示） |
| ✅ | 固定资料自动装配：boot 系统提示词组装末尾追加 pins.Block(cwd)——文本类附正文摘要（单文件 ≤1600 字符、整块 ≤8000）、办公文档列名按需读取（装配而非灌输）；仅新会话/重建时注入 |
| ✅ | 任务模板库（P1-③）：internal/app/gaea_templates.go 内置 8 个模板（周报/会议纪要/成本测算/方案大纲/数据分析/文档转换/报告拼装/演示文稿）——欢迎页新增「任务模板」区（一键填入结构化指令）；ensureTaskTemplateCommands 幂等落盘 .gaea/commands/*.md（不覆盖用户文件），/ 菜单与 Submit 通过既有自定义命令管线直接解析 |
| ✅ | 修复：命令发现改为工作区根——config.CommandDirsAt(cwd)，boot.Build 用 opts.Cwd 加载命令（此前 .gaea/commands 按进程目录发现，切换工作区后模板命令不可见）；与技能/记忆发现口径一致 |
| ✅ | 记忆中枢关联：三脑检索并入工作区全文搜索（文件命中同框展示、可预览/@引用）；首页新增「项目资料」卡片（PinnedCount 入 MemoryHubOverview）与 MaterialsLibrary 库面板（固定/取消、预览，与办公面板共用 pinned.json）；固定资料以 material 节点入记忆 3D 图谱（GraphView 增加颜色/标签/过滤） |
| ✅ | 记忆生命周期（P1-⑤）：SchemaV3 迁移 facts 增加 last_used_at/source_session/source_message；Save 持久化（修订不重置使用时间）、List/Get 回读；memory_get 读取即 Touch（高频信号） |
| ✅ | 轻量检索与压缩注入（P1-⑤）：memory.RecallBlock（关键词 + 时间 + 高频排序，默认预算 800 rune；procedural 常驻、episodic 标签触发、相关 semantic 命中带入）替代逐轮全量注入；ProfileBlock 改为近期/高频排序 + 600 rune 画像预算 |
| ✅ | 记忆可控与溯源（P1-④）：记忆面板标题栏「记忆开关」（GaeaSetMemoryEnabled → 配置持久化 + 引擎重建；关闭后系统提示词与逐轮上下文不再注入画像/规则/事实，磁盘记忆保留可管理）；事实卡片展示来源会话/最近使用（source_session/last_used_at） |
| ✅ | 方法论自动候选（P1-⑥）：GaeaMemorySuggestions 真实实现——procedural 记忆按主题词聚类（≥2 条同主题 → workflow-<主题> 技能候选，证据=记忆名、正文=各记忆内容，最多 5 条）；接受走 GaeaCaptureSkill 固化 + 热加载；契约对齐前端 memories/skills/generatedAt/available/source（修复旧 Facts/Skills 不匹配） |
| ✅ | 测试：wssearch/pins 单测、boot 工作区命令发现集成测试、app 层搜索/固定/模板/记忆中枢关联单测通过；新增记忆生命周期（SQLite 溯源字段 + Touch + 修订不重置）、RecallBlock 排序与预算、ProfileBlock 预算、技能聚类（≥2 同主题 / <2 不候选）单测通过；前端 vitest 93 例全绿，tsc + vite build 通过，go test 全量 + go vet 干净 |

### 说明
- 工作区全文搜索先做关键词/分词（TF-IDF + CJK bigram），向量检索按需再上（与调研一致）
- 固定资料注入时机 = 新会话/工作区切换/引擎重建；会话中途固定不注入当前会话（提示词已缓存）

## 当前版本

- **v2.11.0（正式发布，2026-08-10）**：通用办公大优化——四大库（成本/知识/
  办公记忆/工作区文件）能力闭环 + 本地语义检索栈（bge-m3 语义召回 + BM25 +
  bge-reranker-v2-m3 精排，零 token）。产物 releases/gaea-v2.11.0.exe（41.4MB，
  SHA256 已归档），桌面端同步；go test 全量 + vet 干净、前端 111 例全过、
  tsc/vite/wails build 通过，git tag v2.11.0。

## 发布动作（v2.11.0）

- 版本号对齐：app_info.go / wails.json / frontend/package.json / versioninfo.rc → 2.11.0
- CHANGELOG.md v2.11.0 条目补「本轮追加：四大库大优化」并改为正式发布；
  releases/v2.11.0.md + releases/CHANGELOG-v2.11.0.txt 发布说明；releases/README.md
  版本表补 v2.11.0 行；SHA256SUMS-v2.11.0.txt 已生成
- 构建：结束旧实例（build/bin/gaea.exe 被运行锁定）→ wails build -clean 成功 →
  产物复制 releases/gaea-v2.11.0.exe（41,370,624 字节）
- 记忆更新：办公自动记忆新增事实 release-2026-08-10-v2-11-0（Hephaestus.db facts，
  已用临时 Go 助手写入并验证，临时脚本已清理）

## 发布动作（v2.10.1）

- 版本号对齐：app_info.go / wails.json / frontend/package.json / versioninfo.rc → 2.10.1
- CHANGELOG.md 新增 v2.10.1 条目；README 通用办公能力更新；releases/v2.10.1.md 发布说明；
  releases/README.md 版本表补 v2.10.0 + v2.10.1 两行（v2.10.0 当时漏记）
- .gitignore 补充运行/测试产物（.tmp* / cpu.out / xlsx.out / *.test.exe / .gaea/exports|sessions）

## 开发中（v2.10.1 方向）：开工前 P0 落地

| 状态 | 任务 |
|------|------|
| ✅ | @ 文件引用增强：Composer @ 菜单改为「目录内浏览 / 最近使用 / 工作区跨目录搜索」三合一（统一 AtEntry），文件行显示扩展名徽标（docx/xlsx/pdf/md/png…），选过的文件进 localStorage 最近使用（≤20） |
| ✅ | 后端 GaeaFileSearch：工作区文件名搜索（大小写不敏感、深度 ≤6、限 30 条、跳过 .git/node_modules/dist/build/.tmp* 等噪音目录、中文命中），与 @ 菜单打通（bridge FileSearch → GaeaFileSearch，含 mock） |
| ✅ | 测试：后端 GaeaFileSearch（中文命中/噪音目录跳过/limit/深目录不 panic）通过；前端 88 例全绿（新增 FileMenu 徽标用例），tsc + vite build 通过 |
| ✅ | 工作区资料概览（P0-③）：右侧新增「资料」Tab（MaterialsPanel）——docx/xlsx/pptx/pdf/md/txt/csv 按 文档/表格/演示/PDF 分组、最新在前；行内「预览 / 一键 @ 引用 / 外部打开」；引用通过 useComposerInsertStore 插入输入框（面板→Composer 打通） |
| ✅ | 后端 GaeaMaterials：资料文件按修改时间倒序（限 100、深度 ≤5、跳过噪音目录、只收 office/文本扩展名） |
| ✅ | 测试：后端 GaeaMaterials（类型过滤/噪音跳过/最新在前/路径无前导斜杠）通过；前端 89 例全绿（新增 MaterialsPanel 分组+引用+预览用例），tsc + vite build 通过 |
| ✅ | 开工前计划卡片（P0-①）：AutoPlan 门控打通——AgentRunner.Plan 用同一 provider 一次性生成开工计划；Controller 在回合前（非简单查询时）以 ask 卡片询问「确认执行 / 先调整」，确认才执行，取消则通知并结束；计划生成失败自动回退为直接执行 |
| ✅ | 接线：boot.Options.AutoPlan ← config auto_plan（ask/on 开启）；办公板块 gaeaLoadConfig 默认 ask；前端 AskCard 提示改为 Markdown 渲染（计划以列表呈现） |
| ✅ | 测试：AgentRunner.Plan（成功/失败）、Controller askPlanApproval（确认/调整/空执行器回退）通过；agent/control/boot/app 全量测试通过，go vet 干净；前端 89 例全绿，tsc + vite build 通过 |

### 开工前 P0 全部落地（③ @ 引用增强 + 资料概览 + 计划卡片）

## 调研（2026-08-10）：开工前阶段 — 第五轮竞品优点蒸馏

> 产出 docs/market-research-2026-08-office-prework-context.md
> 主题：任务发出后、AI 动手前的文件交互与调研启动

### 蒸馏出的可落地优点（P0）
1. 开工前计划卡片：auto-plan 结果渲染为「计划卡片」（步骤/待读资料/将用工具/待确认），
   用户确认后再开干——对标 WorkBuddy Plan 模式、千问办公任务步骤清单
2. @ 文件引用增强：@ 菜单支持跨目录搜索 + 最近使用 + 按扩展名分组——对标 WorkBuddy @
   文件列表、Claude @ 按需加载
3. 工作区资料概览：切换/首次进入工作区时列出 docx/xlsx/pdf/md/csv 资料视图，一键
   @引用为上下文——对标千问办公选文件夹授权、aily 关联知识

### 蒸馏出的可落地优点（P1）
4. 工作区全文搜索（轻量 RAG，SQLite FTS5 索引文本文件）
5. 项目常用资料装配（pin 常用文件 → 新会话自动进上下文）
6. 任务模板库（欢迎页能力卡 + slash 升级为预置任务模板）

### 明确不做
- 云端知识库 / 团队共享上下文（个人工具定位）
- 整个文件夹全量塞进 prompt（坚持 @ 按需加载 + 检索装配）

## 开发中（v2.10.1 方向）：记忆自进化 P0 落地

| 状态 | 任务 |
|------|------|
| ✅ | 自动做梦（空闲自整理）v1：gaeaBuildController sink 在 TurnDone（成功）后触发后台整理——取最后一轮对话 → office 模型提炼（Kimi 二问纪律：只记稳定事实、≤5 事实/≤3 笔记、宁缺毋滥）→ 写入长期记忆 |
| ✅ | 写入路径：Controller.SaveDreamFacts（与 promote 同一 Store.Save 路径，按 name slug 去重、更新即修订），笔记走 QuickAdd（user/project/local 作用域） |
| ✅ | 防御：单飞（同时只跑一个）、实质内容门槛（≥2 条消息且助手输出 ≥100 字符，寒暄不整理）、90s 超时、失败静默跳过 |
| ✅ | 用户反馈：整理成功后发 Notice 事件「已自动整理记忆：新增 N 条事实、M 条笔记」，前端聊天内可见 |
| ✅ | 测试：parseDreamOutput（围栏/前缀/坏 JSON/空名过滤）、dreamTurnMessages（只取最后一轮 user/assistant）、dreamWorthwhile（寒暄不触发）、输入截断；Controller.SaveDreamFacts 去重/更新/空值跳过；internal/app 与 control 全量测试通过，go vet 干净 |

### 说明（自动回忆① 与生命周期③ 现状）
- 自动回忆注入（①）大部分已存在：memory.Compose 把文档索引 + ProfileBlock（user 语义事实自动聚合画像）+ ProceduralBlock（过程性规则每轮注入）折进系统提示词；剩余差距是「高频/近期排序与来源时间戳」，留作细化
- 记忆生命周期（③）部分已存在：sqlite facts 表有 created_at/updated_at、Archive 软删除；剩余差距是 last_used_at 高频排序与冲突修订链

## 调研（2026-08-10）：记忆与自进化 — 第四轮竞品优点蒸馏

> 产出 docs/market-research-2026-08-office-memory-evolution.md

### 蒸馏出的可落地优点（P0）
1. 自动回忆注入：会话开始时把用户画像 + 高频/近期 facts + 项目记忆压缩成「记忆上下文」
   注入提示词（带来源与时间戳）——对标 Claude MEMORY.md 前 200 行 / WPS 记忆自动带入
2. 空闲自整理（自动做梦）：会话结束（turn_done）后台归纳对话 → 新事实/偏好/踩坑/技能
   候选入库，复用 compact 提取 + fact 稳定 slug 去重，写入频率受控（Kimi 二问思路）
3. 记忆生命周期：facts 增加 created_at/last_used_at/source_message；同 slug 冲突修订链、
   使用计数参与注入排序、过期候选仅提示不自动删——对标 WPS 夜间整理 / ChatGPT Dreaming

### 蒸馏出的可落地优点（P1）
4. 记忆可控与溯源：MemoryPanel 显示来源会话/消息可跳转 + 「不再记住」+ 记忆开关
5. 轻量检索与压缩注入：facts/画像按「关键词+时间+高频」排序，注入预算 500-800 token
6. 方法论自动候选：多次同类任务后自动提示可沉淀技能（千问办公组织级 Skill 的个人版）

### 明确不做
- 团队共享记忆 / 组织级 Skill 权限（个人工具定位）
- 云端记忆存储与跨设备同步
- 自动删除用户记忆（只提示候选，删除须用户确认）

## 开发中（v2.10.1 方向）：P1 深化落地

| 状态 | 任务 |
|------|------|
| ✅ | 产物溯源：会话产物面板每条记录生成它的轮次（turn），悬停「跳转到生成它的消息」（MessageSquare）→ 收起面板并滚动到对应轮次（对标 AutoGPT Open chat / Claude Provenance） |
| ✅ | 编辑后自动回写刷新：useUpdatedFilesStore 变化 → 工作区文件树自动刷新（替代手动刷新按钮） |
| ✅ | 粘贴表格即数据：tableData.ts 识别 CSV/TSV 表格块（≥2 行、列数一致、≥2 列），Composer 显示「已识别表格数据 N×M」提示条 + 「发送时转为 Markdown 表格」开关（默认开），普通发送与 Shift+Enter 纠正发送均生效 |
| ✅ | 测试：前端 87 例全绿（新增 tableData 5 例、产物溯源 1 例），tsc + vite build 通过 |

## 开发中（v2.10.1 方向）：P0 成果交付收尾体验落地

| 状态 | 任务 |
|------|------|
| ✅ | 会话产物面板：右侧新增「产物」Tab（DeliverablesPanel），从会话消息提取交付文件（去重、最新在前），点击预览，悬停打开/定位/复制路径；空状态引导（对标 Kimi 工作空间 / 千问办公产物面板） |
| ✅ | 交付卡片增强：图片缩略图（AttachmentDataURL 前端加载，无新依赖）、复制路径动作、预览内编辑后「已更新」徽标 |
| ✅ | 已更新徽标链路：useUpdatedFilesStore + DocxPreview（应用修订/接受拒绝）/ XlsxPreview（AI 编辑/写单元格/重算/行列操作）成功后 markUpdated |
| ✅ | 测试：前端 81 例全绿（新增 DeliverablesPanel 3 例、DeliverableCards 已更新用例），tsc + vite build 通过 |
| ⏸️ | Office（xlsx/docx/pptx/pdf）首屏缩略图：需 LibreOffice→PDF→PNG 管线（pdftoppm/PyMuPDF 打包或依赖决策），暂缓，避免引入环境依赖 |

## 调研（2026-08-10）：对话内成果交付与文件交互 — 第三轮竞品优点蒸馏

> 产出 docs/market-research-2026-08-office-deliverable-ux.md

### 蒸馏出的可落地优点（P0）
1. 会话产物面板：右侧文件树之上新增「本次会话产物」视图（复用 findFileMentions +
   交付扩展名过滤，按时间倒序，点击预览/悬停打开/定位/复制路径）——对标 Kimi 工作空间、
   千问办公右侧产物面板、Claude artifact 网格
2. 交付卡片增强：图片缩略图（AttachmentDataURL）、xlsx/pptx 首屏预览图、复制路径动作、
   预览内编辑后卡片自动标「已更新」——对标豆包缩略图卡片、千问办公改完即交付

### 蒸馏出的可落地优点（P1）
3. 产物溯源：文件 ↔ 生成它的消息/任务跳转（对标 AutoGPT Open chat / Claude Provenance）
4. 编辑后自动回写刷新：DocxApplyEdit/XlsxEdit 成功后文件树/预览/卡片自动刷新
5. 粘贴即数据：Composer 粘贴 CSV/表格文本自动识别为结构化上下文

## 开发中（v2.10.1 方向）：输出文本文件引用全量可点击预览

| 状态 | 任务 |
|------|------|
| ✅ | 参考 openclaw / llama.cpp 同类实现，把"文件路径识别"从字符串正则替换改为 mdast（remark 插件）AST 层处理：代码块/行内代码/已有链接/公式天然跳过，无占位符保护 hack |
| ✅ | 聊天正文（含流式未稳定尾部）、工具输出、ask 提问中的文件引用均可点击打开内置预览：覆盖绝对路径（C:\…、/…）、相对路径（exports/…、.gaea/exports/…）、关键词引导裸文件名（输出文件：xxx.docx） |
| ✅ | 后端 GaeaPreview 支持裸文件名在常见输出目录（exports / .gaea/exports / outputs / docs / uploads / attachments / templates 等）中自动解析 |
| ✅ | gaea 输出约定：交付/生成/保存文件时正文必须给出 [文件名](路径) 可点击链接，路径不进代码块 |
| ✅ | 交付物附件卡片（对齐千问办公/Kimi）：回复尾部把正文中的文件引用去重渲染成"图标+文件名+扩展名"卡片，整卡点击预览，悬停提供外部打开/定位（DeliverableCards，接入 AssistantMessage） |
| ✅ | 测试：前端 77 例全绿（Markdown 渲染 / mdast 插件 / 流式尾部 / FileLinkText / 交付卡片 / 识别逻辑），tsc + vite build 通过，后端 GaeaPreview 裸文件名解析用例通过 |

## 当前版本

- **v2.10.0（正式发布）**：通用办公三阶段闭环（前期解析 / 中期编辑 / 后期输出）
  全部落地，桌面端 gaea.exe 已打包发布（Windows x64，约 40MB）。

## 本阶段已完成

| 状态 | 任务 |
|------|------|
| ✅ | 办公前期解析：docx/xlsx/pdf → Markdown 提速 + 扫描件 OCR 四级管线 + 事实底座 |
| ✅ | 办公中期编辑：Word 框选即改/修订、Excel 单元格级编辑（插行插列、公式重算） |
| ✅ | 办公后期输出：统一交付出口（docx/pptx/xlsx/md）+ 模板 + 成本测算模板 |
| ✅ | 跨应用联动：Excel 数据 → 图表 → 嵌入 Word/PPT 并随数据同步 |
| ✅ | Codex 式文件预览布局：右侧文件树 + 主区域可拖宽预览 |
| ✅ | 真实模型验收：DeepSeek 绑定自主生成成本测算表（公式 + 原生图表） |

## 已按用户决定收窄的范围

- 排序 / 筛选 / 条件格式不在 gaea 内做预览层实现（不写回文件），后期交给
  Excel/WPS 专业软件处理；gaea 保留真实落盘的编辑能力。

## 备忘

- 发布动作：版本号对齐 v2.10.0（app_info.go / wails.json / package.json），
  产物 releases/gaea-v2.10.0.exe，桌面端同步，git 打 tag v2.10.0。
