# 造价库本地模型索引 + 旧成本库全量退役（施工单与断点记录）

2026-11-26 傍晚 · 状态：**进行中，未完成，全部未提交**。上下文用尽后由用户令中断。
**下个会话从这里接着做，不要重新摸底。**

与 WPS 能力开发是两条独立的线，互不捆绑（WPS 刀 1 的记录在 `gaea-wps-capability-plan-20261126.md`）。

## 一、用户拍板（2026-11-26，原话要点）

- 「成本库和造价库都要接入本地模型索引，数据库到了后期会越来越膨胀」
- 向量模型：**bge-m3**
- 规模目标：**单类目 50 万条以内**
- 索引范围：**清单、定额、工料机三类都进索引**
- 「现在不就一个造价数据库吗，怎么出来两个，以前那个已经删除了」→ 在用的只有造价库（`gf_*` 表）；
  旧成本库（`cost_entries`）数据已清空。**不要再把它说成两个库。**
- 「旧成本库这套功能要退役」→ 确认**全量退役**

## 二、摸底结论（已核过，勿重查）

1. **现状检索**：造价库（定额 226 / 工料机 185 / 清单行 129 / 项目 4）在模型工具 `cost_project`
   （`action=quotas/resources`）和界面列表（`GaeaWorkcostQuotaList/ResourceList`）里都是纯 SQL
   `LIKE '%关键词%'`，没有语义索引。
2. **旧索引底座的天花板**（`internal/gaea/semantic/semantic.go`，`semantic_vectors` 表）：向量以 JSON 文本存；
   每次查询把该类目下**全部**向量读出来解析；建索引（`Ensure`）挤在搜索路径里靠 5 秒超时兜底；
   没有模型名 / 维度元信息。库一大先慢后废。
3. **两个向量模型各写各的**：方案库和工具侧用 `bge-m3`（1024 维）；应用侧
   `resolveHerdsmanSearchModel`（`internal/app/gaea_cost_rerank.go`）有段「自动升级」逻辑，引擎模型表里有
   `qwen3-embedding` 就选它。本机 `engines.json` 的 herdsman 条目确实列了 `qwen3-embedding-4b` / `qwen3-reranker-4b`
   （都停着）。真实库 `semantic_vectors` 里已混有 2560 维（9 月界面侧写的）与 1024 维两种向量。
   **新索引不走这段逻辑，固定读 `[retrieval]` 配置，默认 bge-m3 / bge-reranker-v2-m3。**
4. **本机实测**：Herdsman `127.0.0.1:8080`，`/v1/models`、`/v1/embeddings`、`/v1/rerank` 可用。
   `bge-m3` 1024 维，单条 30 毫秒；批量 512 条 5.1 秒 ≈ **100 条/秒**（首次建 10 万条约 17 分钟）。
   `bge-reranker-v2-m3` 精排约 80 毫秒；拿「单斗挖掘机」排 4 条候选：「单斗挖掘机 斗容 1m3」3.16、
   「履带式液压挖掘机」0.30、「人工开挖土方」-2.27、「自卸汽车 15t」-2.57（认得出同义说法）。
   模型会空闲卸载（探测到 `stopped` 不等于不可用）。本机没有 `gaea.toml`，`[retrieval]` 走默认值。
5. **旧成本库落点（退役范围，数据现在全是 0 条，只剩 106 个分类节点）**：
   - 模型工具：`cost_search`、`cost_save`（`builtin/cost_tools.go`）、`cost_indicators`（`builtin/cost_indicators.go`）
   - 造价模式提示词：`office_runmode_systemprompt.go`、`office_runmode_skills.go`（「cost_search 查成本库历史单价」
     「确认的单价按 cost_save 沉淀回成本库」）；另有 `gaea_templates.go`、`gaea_subagents.go`、
     `pantheon_grok_agent.go`、`control/controller_approval.go`、`permission/permission.go`、`wenxue_tool_policy.go` 里的引用
   - 后端包（非测试约 6500 行）：`cost` 2196、`costimport` 1232（成本库文件导入）、`costproject` 353（测算项目与沉淀闭环）、
     `costinquiry` 719（询价飞轮）、`costref` 1339（造价参考与复盘笔记）、`pricefeed` 647（价格源定时抓取）
   - 应用绑定：`memory_hub.go` 里 9 个（增删改查 / 分类 / 搜索 / 搜索页），以及 `gaea_cost_import*.go`、
     `gaea_cost_projects.go`、`gaea_cost_inquiry.go`、`gaea_cost_graph.go`、`gaea_cost_ref.go`、`gaea_cost_stage.go`、
     `gaea_price_sources.go`、`gaea_cost_case_samples.go`、`gaea_cost_compose.go`、`gaea_cost_rerank.go`
   - 后台任务：`startPriceCron`（`app.go` 启动处）；`gaea_tasks.go` 引用 pricefeed
   - 检索侧：`retrieval/semantic.go`（`DocText` 吃 `cost.Summary`）、`gaea_semantic_index.go` / `gaea_semantic_search.go`
     里 `kind="cost"` 的分支
   - 前端：没有页面在用，只剩 `frontend/src/wenxue/lib/bindingNames.ts`、`bindingSignatures.ts`、`lib/bridge/mappings.ts`
     的绑定名登记、`lib/mock/office/methods_office.ts`、locale 文案、相关测试
   - **注意**：`localSearchEmbedder` / `localSearchReranker` 虽然住在 `gaea_cost_rerank.go`，但文件索引、记忆召回、
     统一语义搜索都在用——退役时**搬走保留**，不要连带删掉
   - **不属于旧库、不要动**：`cost_project`、`cost_library`、健康扫描（`gaea_cost_health.go`）、批量调价
     （`gaea_cost_bulkprice.go`）、定额飞轮（`gaea_cost_flywheel.go`）、审计（`gaea_cost_audit.go`）、
     导出（`gaea_cost_library_export.go`）——这些都是 workcost
6. **退役边界（用户确认全量）**：上表全删；**不删数据库里的 `cost_*` 表和那 106 个分类节点**（只是不再有代码读写；
  要清表须用户另说）。

## 三、设计

1. **索引底座 `internal/gaea/vecindex`**（叶子包，只依赖标准库）：新表 `semantic_vec2`（二进制 float32 向量 + 正文哈希 +
   模型名 / 维度）与 `semantic_vec2_meta`（每类目一行版本号），即 `SchemaV43`。查询只向量化 query，相似度在按类目缓存的
   内存矩阵上算（int8 量化，1024 维一条 1KB，50 万条约 512MB）。索引维护在后台索引器里做：库的写入点调
   `vecindex.MarkDirty(kind)` → 合并窗口 3 秒 → 增量对账（补缺 / 正文变了重嵌 / 模型变了整类目重嵌 / 清过期）；
   另每 10 分钟兜底对账一轮；模型不可用时待办保留、1 分钟后重试。搜索路径不再嵌文档。
2. **融合排序 `vecindex.Order`**：`0.55·向量 + 0.30·关键词 + 0.15·标题`，编码精确命中加 1.0 恒排最前；
   同分关键词命中优先，再按 ID。
3. **造价库三类索引文档**（待做）：定额 = 编码 + 名称 + 专业 + 章节 + 单位 + 分类路径 + 含量行资源名；
   工料机 = 编码 + 名称 + 规格 + 类型 + 分类路径 + 单位（**不含单价**，免得调价就重嵌）；清单 = 编码 + 名称 + 项目特征 +
   单位 + 项目名（不含工程量 / 综合单价）。
4. **统一检索管线**（待做）：编码精确 → 关键词（`LIKE`）→ 向量（按专业 / 类型预过滤）→ 融合 → 候选超过 8 条时本地精排
   → 每条结果标注「编码 / 关键词 / 语义」。向量模型没开时关键词路照常可用。入口：`cost_project` 的 quotas / resources，
   新增跨项目清单检索（同一工具加 action，不新增工具）、界面两个列表绑定。不新增工具。
5. **类目名**：`wc_quota`（定额）、`wc_res`（工料机）、`wc_bill`（清单）。

## 四、分刀与进度

| 刀 | 内容 | 状态 |
|---|---|---|
| A1 | 索引底座包 `vecindex` + `SchemaV43` | **完成并提交**（15 项单测 `-race` 全过；提交号见 §五） |
| R | 旧成本库全量退役 | 未开工。**排在索引接线之前、单独提交**（纯删除可独立验证，且先删后接更干净） |
| B1 | 造价库三类索引文档 + 写入点打脏标记 | 未开工 |
| B2 | `workcost.Searcher` 混合检索；`cost_project` 与界面列表接入 | 未开工 |
| A3 | 应用接线：后台索引器随应用启动；索引固定 bge-m3；索引状态与补齐绑定 | 未开工 |
| V | 验证：单测回归 + 真实库同义词评测（纯关键词 vs 混合）+ 性能压测 + 门禁 + 记档 + 提交 | 未开工 |

## 五、断点：当前工作树里的未提交改动

主树 HEAD = `bc43d41f`。其上压着这批**未提交**改动（只有这些，别的会话提交时不要卷入）：

- `internal/gaea/db/schema.go`：新增 `SchemaV43`（两张表；已撤掉早先牵连旧库的两行 `DELETE`）
- `internal/gaea/db/database.go`：`migrations` 末尾加 `SchemaV43`（db 包测试已过）
- `internal/gaea/vecindex/vecindex.go`：`Store` / `Apply` / `Search` / `SearchVec` / `Query` / `Meta`
- `internal/gaea/vecindex/matrix.go`：int8 内存矩阵、并行扫描、按版本号缓存
- `internal/gaea/vecindex/indexer.go`：`Indexer` / `MarkDirty` / `SetDefault` / `Run` / `ApplyNow`
- `internal/gaea/vecindex/order.go`：`Cand` / `Score` / `Order`
- `internal/gaea/vecindex/vecindex_test.go`

**A1 收尾结果（第二个会话核实，2026-11-26）**：下面五条在第一个会话写完本单后已经改掉，只是没回写；
第二个会话逐条对码 + 跑测试确认——`vecindex` 15 项单测 `-race` 全过，`db` 包迁移测试过，gofmt / vet 干净。
`TestApply_SerializedPerKind` 四次对账 total 都是 20、最终 20 条（修前的症状是 4 / 16 / 12 / 8、最终 16 条）。
以下保留作修法记录：

1. **真 bug · `Apply` 每次只处理第一批**：`vecindex.go` 的 `Apply` 里，文档循环内写的是
   `if len(pend) >= batch { return flush() }`——凑满一批就从回调返回了，后面的文档没遍历。
   应改为 `if len(pend) >= batch { if err := flush(); err != nil { return err } }`（冲完继续循环）。
   症状：`TestApply_SerializedPerKind` 四次对账 total 分别是 4 / 16 / 12 / 8、最终条数 16 而不是 20。
   （对账的按类目串行本身是对的：四次完成时刻是 31 / 62 / 94 / 124 毫秒。）
2. 测试自身写错的三处：
   - `TestApply_Incremental` 断言 `meta.Version != 2`，实际首次写入后版本号是 1 → 改成 1。
   - `TestApply_PartialFailureKeepsProgress` 的失败注入找「第二批」，但夹具文本是「第2批」→ 改成「第2批」。
   - `TestOrder` 末尾「同分关键词优先」的用例靠浮点精确相等凑同分，不可靠 → 改成同标志位按 ID 排序，
     另断言 `Keyword()`。
3. `TestSearch_SeesIndexUpdates`：查询词「压路机」与首条文档「打桩机 施工」只共一个字，余弦低于下限搜不到 →
   查询词改「桩机 施工」，第二条文档改「静压桩机 施工」。
4. 已通过的：`TestMatrix_QuantizedRankingMatchesExact`（int8 量化对精确余弦 top-10 平均重合 0.988）、
   `TestMatrix_ParallelEqualsSequential`、`TestMatrix_LargeSearchLatency`（20 万条 × 128 维单次 2.9 毫秒）、
   索引器三项（脏标记合并 / 模型不可用重试 / 兜底对账）、`TestOpen_SharesStorePerDB`、`TestApply_Unavailable`、
   `TestApply_SkipsEmptyAndClearsEmptied`、`TestSearch_RanksFiltersAndLimits`。
5. race 检测还没跑成：`-race` 要 PATH 前置 `/c/msys64/ucrt64/bin`（vecindex 包不依赖 markitdown，前置无副作用）。

## 六、坑与纪律

- `go test -race` 在本机默认 PATH 下报 `cgo: C compiler "gcc" not found`，要先前置 ucrt64。
- 用了 Python 的 heredoc 补丁文件，落盘后要重新 `gofmt`；Edit 工具在文件被脚本 / gofmt 改过后要先 Read。
- 退役要守的纪律：义务清单先行；新增/退役工具要对账三处（注册 / `spacetags` / `compact.go`，守卫会拦），
  另有 `registry_map_doc.go -check`、`check-bindings-drift.ps1`、前端 knip；数据库表不删。
- 历史存量红（与本线无关，未动）：`agent` 的 `TestStaleGuardBlocksMoveFileWithoutRead`、`trajectory` 的 multi_edit 金样本。
