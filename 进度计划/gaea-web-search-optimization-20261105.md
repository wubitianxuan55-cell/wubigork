# gaea 网络搜索能力优化计划（2026-11-05）

> 事实源：众神殿会话 `pan_1793872913023_9c20fd7e7ab9`（三刀共识，四席全签）；
> 论证台账见 `docs/gaea-web-search-pantheon-digest-20261105.md`。
> 状态标记：✅已落地 / 🚧进行中 / ⏸待拍板或批次二。

## 0. 主源路线：✅ 已拍板 A 托管 API（2026-11-05 晚，用户拍板）

- **A 托管 API（已选）**：Tavily（免 1000 次/月，score 字段现成）或 Brave（免 2000 次/月）。零运维；查询外发+免费额度约束。
- ✅ 已接线（2026-11-05 晚）：Tavily key 已配（user env `TAVILY_API_KEY` setx + repo `.env`（gitignored）+ `%APPDATA%\gaea\config.toml` `[search] tavily_api_key_env="TAVILY_API_KEY"`）；运行中 gaea.exe 须重启生效。
- A 臂首批实测数据点（同句「2026年11月3日 人民币兑美元汇率」）：bing 672ms 先胜但 5 条全离题 vs tavily 1122ms 5 条全正题（当日中间价 6.7150 命中）——**首胜单槽丢主源活案例，刀2 主源保留窗实证支点**；tavily-only 探针 5/5 正题 key 链路通。
- Tavily 工具面：tvly CLI 0.2.1 已装已认证（api_key）；Tavily Agent Skills 已全局装入 `~/.zcode/skills`（重启/重扫后生效）。
- 影响收口：刀3=向导形态按 A 落（设置面板配置卡：选引擎+填 key+测试连接）；A 臂基线等 key 到位后跑（保留窗毫秒/主源入 top5 比例由此定数）。

## 1. 刀1 统一检索核 + 三层失败契约 + 四入口收敛 + 清洗边界 ✅（2026-11-05 晚落地，三包测试绿）

| 项 | 内容 | 文件 |
|----|------|------|
| 统一核 | `RunSearch(ctx, query, topK)` 从 Execute 抽出（编排首胜现状不动），逐引擎 `SearchAttempt{Name,Outcome(ok/empty/filtered-empty/error),Err,ElapsedMS}`；`ClassifySearchState` 枚举 OK/EMPTY/ALL_FAILED（DEGRADED 留刀2）；**策略过滤后空≠ALL_FAILED**；办公 `Execute` 输出零变化（JSON 数组+全败诊断文案原样） | `builtin/websearch.go`（+`websearch_core_test.go`） |
| 聊天适配器 | `runChatWebSearch`：清洗（8s ctx 硬超时）→ 统一核 → 状态分类 → 注入文本；`cleanChatSearchQuery` 下沉自 whisper 并**移除「在哪里/在哪儿/多少钱」后缀剥离**；OK 结果列表带 title/url/snippet/source | `app/chat_search_adapter.go`（新） |
| 三层契约 | ①状态枚举 ②注入头按状态选文案（OK=实时结果头；EMPTY=已搜索无相关结果+如实；ALL_FAILED=联网搜索暂不可用+如实）③plain 提示词删「不要说自己无法联网搜索」改「未获取到实时信息如实说明」（仅 plain，人格路径不加） | `app/chat_service.go` |
| 四入口 | chatWebSearch seam 换绑 adapter；人格路径 `prepareWhisperChatMessage` 同 adapter（失败不再静默，注入如实占位）；`WhisperWebSearch` 绑定委托 adapter（map 增 `state`，死 error 字段修活）；agent loop 走 `whisper.WebSearchToolExec` 委托缝（boot 注入 builtin，whisper 不 import builtin），死失败分支改活 | `chat_service.go`/`whisper_handler.go`/`agent_loop_runner.go`/`app.go Startup` |
| 删旧栈 | `internal/whisper/web_search.go` 整文件删除（Bing/DDG 解析+httpClient+cleanSearchQuery）+旧测试删除→清洗用例迁 app（删符号=源级守卫，绕过 adapter 即编译红） | whisper 包 |
| 不动 | 触发词表 `searchTriggers`；办公 query 原样（`site:` 不改写）；`Execute` JSON 数组输出；per-engine 15s/总 20s 超时 | — |

验收（会话终稿锚定）：
- [x] 仓库无绕过 adapter 的旧直连（`whisper.WebSearch` 符号删除，编译级保证）
- [x] adapter 单测：「帮我搜索一下明天上海天气」→「明天上海天气」；「成都到上海在哪里买票」「这个多少钱」不被剥尾
- [x] 办公 `site:` query 经统一核原样不改写（RunSearch 透传测试）
- [x] 三状态注入头文案分叉有测试；策略滤空与 ALL_FAILED 拆分有测试
- [x] 全量门禁：go build/vet + 相关包测试绿

## 2. 基线先行：30 query 固定集 ✅（2026-11-05 晚实弹 150 记录）

- 档案：`docs/web-search-baseline-30q-20261105.json`（摘要）+ `.jsonl`（原始 150 记录，5 臂×30）。
- **主源入 top5 = 1/30（3.3%）**——首胜竞速量化实锤；tavily p50=2.4s/p95=3.2s；bing p50=0.6s；**DDG 28/30 失败=本机不可达，keyless 必达源只剩 bing**。
- 覆盖门校准：F30 多实体部分命中误判案例 → 收紧为过半实体命中；E 类 5/5 判对。

## 3. 刀2 首胜→短窗合并 + 主源保留窗 + 覆盖门 v0 ✅（2026-11-05 晚落地）

- `builtin/websearch_core.go`：RunSearch 返回 `SearchOutcome{Results,Attempts,State,DegradedReason,Diagnostics}`；主源保留窗（**3.5s，A 臂 p95 校准**）→主源为主+短窗合并（1.2s，URL 规范化去重 www./尾斜杠/utm_*，双来源并入 Source）→超窗/失败走 keyless 兜底并强制 `DEGRADED(primary-timeout|primary-error)`（合成超时尝试入诊断，禁止静默 bing-only 标 OK）。
- `builtin/websearch_coverage.go`：覆盖门 v0=剥日期/数字/泛词后**过半核心实体**命中 title+snippet（F30 校准），不过→`DEGRADED(coverage-generic-query|coverage-miss)`；办公 Execute JSON 数组契约零变化。
- 聊天 adapter：DEGRADED 注入文案（降级标注压过成功头），四态互异。
- Codex 桩测五条全绿（retention 保留/超窗降级/URL 去重双源/nil=EMPTY 解析败=ALL_FAILED/四态注入头）+ F30 部分命中回归 + 泛词/多实体过门用例。
- 验证跑（实弹 60 记录）：keyless 臂 E 类 5/5 判对、F 类 4/5（F27 判 OK 属正确——结果真相关）；primary-error 环境故障下 30/30 无伪 OK（降级语义实弹验证）。
- 余项：主源窗内 tavily 29/30 x509 证书失败为 fake-ip 代理出口环境噪声（基线跑同代码 29/30 ok），待环境恢复复测主源入 top5 比例；reranker 二档挂池（误杀率现值 0 触发）。

## 4. 刀3 主源向导 + 限时 keyless + 全败才指引 ✅（2026-11-05 晚落地）

- ✅ 设置面板办公分类「网络搜索」卡（SearchPanel）：Tavily/Brave key+SearXNG URL 三源任配其一即主源；测试连接=真实走统一检索核出网，ok/degraded=连通（降级原因如实携带）、全败附配置指引；可用性徽标。
- ✅ 后端三绑定 WebSearchSettingsGet/Set/TestConnection：key 落 ~/.env（dotenv boot 同源，不进 config.toml）+Setenv 立即生效+builtin 三件热重载；空串=清除。绑定面手写同构入 bindings_office.go+explicitOverrides（他方在途同树，未整跑生成器，下轮收敛）；wailsjs 待下次 build amend。
- 全败指引保持现位；人格路径/普通路径统一可见状态。
- 主源生效归因靠刀2 埋点（A 臂 OK 类 top5 主源出现率达标）。

## 5. 批次二（本批不做，防丢）

触发面升级（扩表 vs 4B 前置判别，先攒失败样本）；契约字段（日期/score/证据 ID）按消费方需求加；`ChatSend` 回传 `search_state` UI 可见性；Bing `ck/a` 跳转 URL 去重核验；public-searxng 并行化；搜索 TTL 缓存。

## 6. 纪律

- 一处一刀；不批量字符串补丁；不提前宣布全绿——门禁跑完再报。
- 办公 `Execute` 输出契约零变化（消费方是模型工具循环，不在本批动）。
- 工作树另有他方在途改动（boot sonic/voice UI），本刀提交只圈自己的文件。
