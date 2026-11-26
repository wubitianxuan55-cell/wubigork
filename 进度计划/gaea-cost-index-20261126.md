# 造价库本地模型索引 + 旧成本库全量退役（施工单与断点记录）

2026-11-26 · 状态：**六刀全部提交——A1 `abd65f42`、R `459723c2`、B1 `cb4e4c23`、B2 `9fbf8e2e`、A3 `90fa05ab`、V（见 §十）。
代码面收官；待令 = 打包、装机、真机走查、push。**
（第一个会话傍晚开工、上下文用尽中断；第二个会话下午做完 A1 收尾与 R、B1、B2、A3、V。）
接续/复核先读 §八（退役的结果与行为变化）、§九（B1 / B2 / A3）、§十（评测与性能）。

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
| R | 旧成本库全量退役 | **完成并单独提交**（清单见 §七，结果与验证见 §八） |
| B1 | 造价库三类索引文档 + 写入点打脏标记 | **完成并提交** `cb4e4c23`（见 §九） |
| B2 | `workcost.Searcher` 混合检索；`cost_project` 与界面列表接入 | **完成并提交**（提交号见 §九） |
| A3 | 应用接线：后台索引器随应用启动；索引固定 bge-m3；索引状态与补齐绑定 | **完成并提交**（见 §九；含界面状态钮与语义补充段） |
| V | 验证：单测回归 + 真实库同义词评测（纯关键词 vs 混合）+ 性能压测 + 门禁 + 记档 + 提交 | **完成并提交**（见 §十）；打包装机与真机走查等用户通知 |

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

## 七、R 退役义务清单（第二个会话建图，2026-11-26；施工单 §二.5 之外新发现的都标「新增」）

**新增发现——都是只服务旧库的代码，随旧库一并退役（库表照旧不删）：**

1. `internal/gaea/coststage`（五算对比，841 行含测试）：唯一消费方是删除清单里的 `gaea_cost_stage.go`，不碰 workcost / `gf_*`。
2. 造价库导出 `GaeaCostLibraryExport` 的 `kind=entry`（资料条目）+ `costExportEntry`：导的就是旧库 `cost_entries`。
3. `cost_library` 的 `seed` 动作（资料条目资源化）+ 绑定 `GaeaWorkcostSeedPreview` / `GaeaWorkcostSeedApply` + `workcost/seed.go`：
   直接读 `cost_entries` 灌工料机库，是旧库 → 造价库的迁移桥；旧库代码删掉后没有数据源。
4. `GaeaWorkcostRecompose` + `workcost/entrycache.go` 的 Recompose 一族：只重算旧库条目的 price。
5. `scripts/migrate_workcost/main.go`：旧库 → 造价库的一次性迁移脚本（裸 SQL 读 `cost_entries`），同上。

**要退役的绑定 53 + 3**：`memory_hub.go` 9 个（增删改查 / 分类 / 检索 / 检索页）、compose 3、graph 1、
import 3、import_vision 2、inquiry 6、projects 10、ref 6、stage 4、price 9；桥 3 个（SeedPreview / SeedApply / Recompose）。

**保留但要动的地方**：`memory_hub.go`（总览计数 / 图谱的 cost 段）、`app.go`（`startPriceCron`）、
`gaea_tasks.go`（两个价格抓取任务 handler）、`retrieval/semantic.go`（`DocText` 的 cost 分支）、
`gaea_semantic_index.go` / `gaea_semantic_search.go`（`kind="cost"` 分支）、提示词五处
（`gaea_templates.go`、`office_runmode.go`、`office_runmode_skills.go`、`office_runmode_systemprompt.go`、
`office_cost_tools.go`）、`control/controller_approval.go`、`wenxue_tool_policy.go`、`permission/permission.go`（注释）、
登记四处（`compact.go` / `spacetags.go` / `registrymap.go` / `toolgroups.go`）、`.gaea/commands/cost-estimate.md`。

**明确不动的**：`tasks.KindPriceFetch*` 常量（通用任务种类的字符串，库里的历史任务记录还认得它，测试也拿它当样例）；
归档文档；测试里把 `cost_search` 当任意字符串用的夹具。

**拿旧库条目当夹具的保留测试（逐个：删旧库功能用例，或换夹具）**：`gaea_cost_library_export_test.go`、
`gaea_workcost_test.go`、`office_cost_library_test.go`、`gaea_specialist_tools_test.go`、`gaea_unified_search_test.go`、
`memory_graph_test.go`、`gaea_retrieval_eval_test.go`、`gaea_semantic_search_test.go`、`gaea_semantic_index_test.go`、
builtin `p3_predicates_test.go`。

**前端**：绑定登记（`bindingNames.ts` / `legacyBindings.ts` / `mock/bindingSignatures.ts` 由 `gen_bindings` 再生）、
`lib/bridge/mappings.ts`、`lib/tools.ts` 工具目录、`lib/taskTemplates.ts`、mock、locale（gaea + wenxue 两套）、
相关测试、wailsjs 生成件。

## 八、R 退役执行结果（第二个会话，2026-11-26 下午）

**结论**：旧成本库全量退役完成并单独提交；库表与那 106 个分类节点没动（只是不再有代码读写）。

**删除面**（83 个文件、约 1.9 万行）：七个后端包（施工单列的六个 + 漏列的 `coststage`）、app 旧库文件 23 个
（含测试）、builtin 三工具（`cost_search` / `cost_save` / `cost_indicators`）及其测试、迁移脚本
`scripts/migrate_workcost`、workcost 的 `seed.go` / `entrycache.go` 及其测试（活代码共用的 `isKnownKind` 搬进
`store.go` 保留）、`retrieval/semantic.go`、`gaea_semantic_index.go`、死技能 `.gaea/skills/cost-compose`。

**改名归位（不删）**：`gaea_cost_rerank.go → gaea_search_clients.go`（去掉两个旧库函数）；`builtin/cost_tools.go`
里的通用检索客户端拆成 `builtin/retrieval_clients.go`（`costEmbedder` / `costReranker` → `localEmbedder` /
`localReranker`，方案库两处调用点跟着改）；`gaea_cost_marshal_test.go → gaea_bridge_client_inject_test.go`（只留桥接注入回归）。

**施工单 §二.5 之外一并退役的（都是只服务旧库的代码）**：`coststage` 包；造价导出 `kind=entry`（资料条目）；
`cost_library` 的 `seed` 动作 + `GaeaWorkcostSeedPreview` / `Apply`；`GaeaWorkcostRecompose`；迁移脚本；
cost-compose 技能（它描述的组价校验引擎早已不在内核里）；交付面板「沉淀到成本库」菜单项（办公 / 文学两板）；
`GaeaSemanticIndexBackfill`（成本类向量索引补齐）；文学侧整套旧库登记（名字清单、两份签名清单、映射、空间归属、
桥接接口 38 个方法、mock 实现、38 个类型）；两板 mock 共享状态里的旧库样例数据、办公侧 5 个旧库类型。

**行为变化（用户可感，汇报时要讲）**：
1. 模型工具面少三件（`cost_search` / `cost_save` / `cost_indicators`）。造价提示词（模板 / 运行模式 / 技能 /
   系统提示词 / `cost_project_apply` 描述 / `cost-estimate` 命令 / 前端任务模板与 mock 模板）里「cost_search 查历史单价」
   改为查造价库工料机价（`cost_project action=resources`），「cost_save 沉淀」整步删除。
2. 交付面板的表格产物不再有「沉淀到成本库」按钮（两板）；三语词条 `deliverPanel.deposit*` 一并撤。
3. 记忆中枢总览少「成本库」计数（`costCount` 字段撤）；「最近更新」改取办公记忆（含文学脑）最近一次写入时间
   （原先只看旧库条目）。记忆图谱不再有 cost 节点。
4. 造价库导出只剩 bill / quota / resource / template；`cost_library` 不再有 `seed` 动作。
5. 审批：`cost_save` 不再是硬确认工具（remember / cost_library / proposal_save 等不变，相关测试样例换成 remember）。
6. 任务调度器缺省优先级去掉价格抓取两项、启动不再拉起价格源定时抓取；`tasks.KindPriceFetch*` 常量保留
   （历史任务记录仍可读）。
7. `semantic_search` 工具与统一语义检索的类目只剩 办公记忆 / 工作区文件（「成本库」一类撤）；索引状态视图撤
   `costTotal` / `costIndexed`。
8. 文学侧 deny 名单与办公侧工具空间分类表去掉这三个工具名（它们已不存在）。

**验证**（全部在本提交的工作树上实跑）：
- Go：`go build ./...`、`go vet ./...` 过；**app 包 1363 例 4 分片全绿**；非 app 包 139 个包**零回归**——
  剩余 5 例红在退役前提交 `abd65f42` 的独立工作树上同样红（agent `TestStaleGuardBlocksMoveFileWithoutRead`、
  trajectory `TestFoldDeliverablesGolden`、proposal `TestMigrationFromV35`、workcost
  `TestApplyProjectBundle_RegionsFactsAndCodeAlias`、voice/realtime `TestPipelineMultiTurnNoSnowball`）；
  另 3 例是 ucrt64 前置 PATH 顶掉 markitdown 的环境假红（builtin 的 `format_convert` 两例 +
  `TestProposalImageEndToEndDocxConvertAndSave`），不带前置单跑即绿。
- 前端：`tsc -b` 过；vitest 全量 4778 例 4777 绿，唯一 1 例红是模型中心 `SchedulingSection` 的等待超时
  （出现在与 Go 分片测试并行时；隔离单跑 8 次过 7，该文件本刀未动）；knip 退出码 0。
- 门禁：`registry_map_doc -check` OK（地图已重导，diff 只有三个工具与两行谓词类代表工具）；`check-bindings-drift.ps1`
  OK（665 个方法）；`check-docs.mjs` OK；`gen_bindings` 三份清单无变化，完备性测试随排除清单清理重生成。
- wailsjs：实测漂移只有 `models.ts` 里 `MemoryHubOverview.costCount` 两行，已同步（门面文件里没有任何已删方法）。
  下次 `wails build` 照常再生复核。

**未做 / 待令**：打包、装机、push（等用户通知）；真实库不迁移、不删表。

## 九、B1 / B2 执行结果（第二个会话，2026-11-26 傍晚）

**B1 三类索引文档 + 写入打脏标记（`cb4e4c23`，`internal/gaea/workcost/index.go`）**
- 文档正文只放「是什么」，不放会频繁变动的数：定额 = 编码 + 名称 + 专业 + 章节 + 单位 + 分类路径 + 含量行资源名
  （按 sort,id 排好再拼，正文哈希稳定）；工料机 = 编码 + 名称 + 规格 + 类型 + 分类路径 + 单位（不含单价）；
  清单 = 编码 + 名称 + 项目特征 + 单位 + 项目名（不含工程量 / 综合单价；键是行 ID）。空字段整段不要。
- 全量文档源 `Store.IndexSources()`：定额按编码键集分页，含量行资源名在 Go 侧拼（SQLite 派生表里不能引用
  外层列，一条 SQL 做不了；含量行没填资源名时回查资源表；取含量行前先关定额游标，防单连接库自锁）。
- **打脏标记没打在几十个写路径上，而是打在库句柄上**：`Open` 把 `*sql.DB` 包成 `dirtyDB / dirtyTx / dirtyStmt`，
  写语句成功后按语句里的表名映射类目调 `vecindex.MarkDirty`（`gf_quotas` / `gf_quota_items` → wc_quota、
  `gf_resources` → wc_res、`gf_bill_items` / `gf_projects` → wc_bill）。现有与将来的写路径一律覆盖；
  事务回滚后多打的标记无害。`Store.DB()` 返回裸句柄，经它写库的调用方不经标记，靠索引器 10 分钟兜底对账。
  三个 `backfill*GCodes` 改收 `executor` 接口。
- 测试 4 项 `-race` 全过，含集成用例「Store 写入 → 打脏 → 索引器增量对账：只嵌新条目、调价不重嵌、删除清向量」。

**B2 混合检索 `workcost.Searcher`（`internal/gaea/workcost/search.go`）**
- 管线：编码 / 自有码精确 → 关键词（SQL LIKE：名称 / 编码 / 规格 / 章节 / 分类 / 项目特征）→ 向量（查询词向量化
  一次，类目内存矩阵取 top-N）→ `vecindex.Order` 融合 → 候选超过 8 条且精排可用时重排 → 每条标 code / keyword /
  semantic。编码精确命中只返回它，不再补近似。
- 向量路任何一步不可用（模型没开 / 索引没建 / 维度对不上）一律静默让路，关键词路照常。查询词为空 = 按原
  列表顺序返回。专业 / 类型 / 项目过滤对关键词路与向量路都生效（向量路命中的行按键回表时套过滤）。
- 接入：
  - `cost_project` 的 quotas / resources 带关键词一律走混合检索；非关键词命中的标〈编码〉/〈语义〉；
    **新增 `action=bills` 跨项目清单行检索**（keyword / specialty / project_id），不新增工具。
    造价提示词两处补了用法（查询词写自然说法、〈语义〉的含义、bills）。
  - 界面 `GaeaWorkcostQuotaList` / `GaeaWorkcostResourceList`：只是浏览或类目索引没建时保持原列表口径
    （全量返回，既有行为不变）；索引就绪且带关键词时走混合检索（前 200 条）。
  - 模型依赖经 `builtin.LocalEmbedder` / `LocalReranker` 固定读 `[retrieval]` 配置（与方案库同一套客户端），
    不走 `resolveHerdsmanSearchModel`。应用侧装配在 `internal/app/gaea_cost_search.go`（含测试注入缝
    `SetCostIndexForTest`）。
- 测试：workcost 6 项（关键词退化 / 编码短路 / 同义词召回 / 精排重排 / 向量路让路 / 清单跨项目）+ app 3 项
  （工具标注与短路 / bills / 界面绑定「索引就绪才走」）全过。
- B2 后的全量：`go build` / `go vet` 过；app 包 1366 例全绿；非 app 零回归（失败清单与 §八 相同）。

**A3 应用接线（见本提交）**
- 后台索引器：`internal/app/gaea_cost_indexer.go` 的 `startCostIndexer`（`app.go` 启动处、任务调度器之后起；
  `Once` + 取消函数 + panic 恢复，关闭时停）。`Indexer.Run` 改成**启动首轮立即对账**（原先要等满 3 秒合并窗口——
  开机那一刻没有可合并的写入）。`vecindex.Indexer` 补了「最近一轮对账的错误」记录（`LastError`）与 `Model` / `Kinds`。
- 模型固定读 `[retrieval]`：`builtin.LocalEmbedModel()` 取模型名（缺省 bge-m3），`LocalEmbedder` / `LocalReranker`
  取客户端。顺手修了 B2 留下的隐患：检索后端 kind 未知时客户端是 nil 指针，装进接口会变成「非 nil 接口包着 nil」，
  调用方的 nil 判断失效——`costIndexEmbedder` 现在显式返回 nil。
- 绑定三个：`GaeaCostIndexStatus`（三类各自：条数 / 已建向量数 / 模型 / 最近写入 / 最近错误；索引器是否在跑；
  只读、不探测模型）、`GaeaCostIndexBackfill(kind)`（显式补齐，空 = 全部；模型不可用如实报错）、
  `GaeaWorkcostBillSearch(keyword, specialty, projectId)`（清单行跨项目检索，给板块清单级用）。
- **纠正 B2 的一处说法**：B2 给两个列表绑定加了混合检索，但造价数据库板块只在挂载时用空关键词拉全量快照、
  搜索框是前端本地子串过滤，所以那一步对板块的搜索框**并没有生效**。A3 把真正的界面接入补上了：
  关键词非空且该级索引已建时，页面防抖 250ms 调一次后台混合检索，把「通不过本地关键词过滤」的结果
  （纯语义命中）作为**补充段**放在主表之后（小标题「名称不含『关键词』、意思相近」，最多 30 条）；
  本地零匹配而语义有命中时不出空态、直接给补充段；索引没建 / 绑定不可用 / 请求失败都没有补充段，本地过滤照常。
- 界面：状态栏的「语义索引」状态钮（`cost-library/IndexStatus.tsx`：就绪 / 建设中 已建·总数 / 待补齐 / 未建；
  悬停给三类明细与错误；点击补齐 → 重读状态 → 提示；空库不出钮）。
- 配套：`lib/types/cost.ts` 两个视图类型、`lib/bridge/cost.ts` 接口、`mappings.ts`、`spaceBindings.ts`、
  `bindingNames.ts`（+3）、`mock/bindingSignatures.ts`、`legacyBindings.ts`、`mock/cost.ts`（dev 态如实报未建）；
  wailsjs 的 `CostB.d.ts/.js` 与 `models.ts` 两个类手工同步（下次 `wails build` 再生复核）。
  退役遗留的 `officeState.priceCron*` 三个空壳字段与关闭钩子一并清掉。
- 测试：vecindex +1（错误记录）、app +2（索引器随启动与状态流转 / 模型不可用如实报错）、
  页面 +6（状态钮三态与补齐 / 索引未建不出补充段 / 语义补充段 / 本地零匹配给补充段）、`IndexStatus` 8 例。
- A3 后的全量：`go vet` 过；app 包 1368 例全绿；非 app 零回归；`tsc -b` 过；相关 vitest 13 个文件 113 例全过；
  `check-bindings-drift`（668 个方法，无新增死绑定）/ `registry_map_doc -check` / `check-docs` / knip 均 OK。

**V 的做法与结果见 §十。**

## 十、V 真实库同义词评测与性能（第二个会话，2026-11-26 傍晚）

**评测口径**：拿真实库的**副本**（`%APPDATA%\gaea\Hephaestus.db` 复制到系统临时目录，评完即删，没碰活库），
用本机 Herdsman 的 bge-m3 / bge-reranker-v2-m3 给三类建索引，对 19 个「名称里不含查询词」的同义说法
（含两个英文词）比较「纯关键词（LIKE）」与「混合检索」的 top-8。期望条目是照着库名人工点的对应项，共 61 个。
可复跑：`internal/gaea/workcost/search_live_test.go`（默认跳过；`GAEA_COST_EVAL_DIR` 给副本目录、
`GAEA_COST_EVAL_BASE` 给模型地址才跑）。
库规模：定额 261 / 工料机 225 / 清单 159（项目 5）——比第一次摸底时又长了些。

**结果**：

| 指标 | 纯关键词 | 混合检索 |
|---|---|---|
| 期望条目 top-8 命中 | 1 / 61（2%） | 42 / 61（69%） |
| 关键词一条都对不上的查询 | — | 19 个里 15 个被向量路召回 |

- 强项：挖机→挖掘机 4/4、吊车→汽车吊 3/3、excavator→挖掘机 4/4、dump truck→自卸汽车 4/4、
  办公室搭建→板房 3/3、扬尘治理 5/5、基坑排水→基坑降水 2/2、地面硬化 3/3、污泥脱水 2/2、洗车槽 4/5。
- **弱项（如实记，没修）**：铲车→装载机 0/4、砼→商品混凝土 0/5——行话缩写向量模型认不出来；
  翻斗车→自卸汽车 2/4；渣土外运在定额类目 0/3（向量路给的是「建渣污泥密闭运输」这类邻近项，算相关、
  但不是我点的期望）、在清单类目 1/4；泥水分离设备 1/2。想补这两个缩写，可以在关键词路加一张小的
  同义词表（未做，挂待定）。
- 增量对账旁证：同一副本第二次跑评测，三类都是「新嵌 0 条」——真实库上的增量对账确实生效。

**性能**（本机）：

| 项 | 实测 |
|---|---|
| 建索引吞吐 | 645 条 / 8.9 秒 ≈ 72 条/秒（含文档源读取与哈希比对） |
| 纯关键词单次 | 约 1 毫秒 |
| 向量路单次（含查询词向量化） | 约 19 毫秒 |
| 完整混合单次（含精排，候选封顶 16 篇） | 约 235 毫秒（封顶前 430 毫秒，那时候选 32 篇） |
| 内存矩阵 10 万条 × 1024 维 | 建矩阵 0.41 秒、占内存约 69 MB、单次检索 10 毫秒 |
| 线性外推 50 万条（用户拍板的规模上限） | 建矩阵约 2 秒、内存约 350 MB、单次检索约 52 毫秒 |

**V 顺手改的一处**：精排候选从「全部融合候选」封顶到 16 篇（`searchRerankDocs`）。精排耗时与篇数成正比
（bge-reranker-v2-m3 约 12 毫秒/篇），融合序靠后的本来就排不进前几名——延迟减半，top-8 召回只少 1 个。

**V 后的验证**：vecindex 包 `-race` 全过（含新增的 1024 维性能护栏）；app 包 1368 例全绿；
workcost 只剩那一例已知存量红。

**没做 / 待令**：打包、装机、真机走查（板块状态钮与补充段）、push。

## 六、坑与纪律

- `go test -race` 在本机默认 PATH 下报 `cgo: C compiler "gcc" not found`，要先前置 ucrt64。
- 用了 Python 的 heredoc 补丁文件，落盘后要重新 `gofmt`；Edit 工具在文件被脚本 / gofmt 改过后要先 Read。
- 退役要守的纪律：义务清单先行；新增/退役工具要对账三处（注册 / `spacetags` / `compact.go`，守卫会拦），
  另有 `registry_map_doc.go -check`、`check-bindings-drift.ps1`、前端 knip；数据库表不删。
- 历史存量红（与本线无关，未动；第二个会话在退役前提交的独立工作树上实证过）：`agent` 的
  `TestStaleGuardBlocksMoveFileWithoutRead`、`trajectory` 的 multi_edit 金样本、`proposal` 的 `TestMigrationFromV35`、
  `workcost` 的 `TestApplyProjectBundle_RegionsFactsAndCodeAlias`、`voice/realtime` 的 `TestPipelineMultiTurnNoSnowball`。
- 存量环境假红（ucrt64 前置 PATH 顶掉系统 python → markitdown 不可用）：builtin 的 `format_convert` 两例 +
  `TestProposalImageEndToEndDocxConvertAndSave`（注意它在 **builtin** 包，不在 proposal 包）；不带前置单跑即绿。
- `go test -exec` 的包装器**必须给绝对路径**（`C:\AI\wubigrok\scripts\cgo_test_dll_prep.cmd`）：go 把测试进程的
  工作目录设成包目录，相对路径从那里解析不到，表现为整盘 0.001 秒全红。app 包直接 `go test ./internal/app/`
  不带包装器 = `0xc0000135`。
- 基线对照别归错包：`-run` 的用例不在那个包里时输出 `ok … [no tests to run]`，看上去是绿，其实是没跑。
- ZCode 工具：`git mv` 之后新路径要先 Read 才能 Write；同一个文件「脚本改 → 读 → 编辑」放进同一批会并行，
  Edit 拿到的是旧快照（要分批串行）；自写上下文工具给命中加的【】是高亮，别当成原文去匹配。
- 批量替换脚本写成「每处必须恰好命中 N 次、任一处不符整文件不改」，这次多次靠它挡住了敲错的 old 串。
- `registry_map_doc.go -check` 在本机是「检出任一文件就报漂移」的假红：它拿生成器的 LF 输出和工作树文件逐字节比，
  而 `core.autocrlf=true` 让检出件是 CRLF。判真伪的办法：重导后 `git status` 无差异 = 内容一致；想让门自己变绿，
  先把工作树那份转成 LF 再跑（提交不受影响，仓库存的就是 LF）。
- Go 的「typed nil」坑：`*T` 为 nil 时直接赋给接口变量，接口不等于 nil（类型信息还在），调用方的 `x == nil` 判断
  失效、一调方法就崩。工厂函数可能返回 nil 指针时，装接口前先判空再返回裸 nil（`costIndexEmbedder` 的修法）。
- 前端手写 wailsjs 生成件要照生成器的格式：字段名是合法标识符就用普通属性（`updatedAt: string;`），带 omitempty
  才写成 `"lastError"?: string;`，构造器里对应 `this["lastError"] = …`（`this."x"` 不是合法 TS）。
