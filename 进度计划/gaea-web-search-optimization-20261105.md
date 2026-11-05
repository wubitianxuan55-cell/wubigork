# gaea 网络搜索能力优化计划（2026-11-05）

> 事实源：众神殿会话 `pan_1793872913023_9c20fd7e7ab9`（三刀共识，四席全签）；
> 论证台账见 `docs/gaea-web-search-pantheon-digest-20261105.md`。
> 状态标记：✅已落地 / 🚧进行中 / ⏸待拍板或批次二。

## 0. 主源路线：✅ 已拍板 A 托管 API（2026-11-05 晚，用户拍板）

- **A 托管 API（已选）**：Tavily（免 1000 次/月，score 字段现成）或 Brave（免 2000 次/月）。零运维；查询外发+免费额度约束。
- 待用户动作：注册 Tavily（tavily.com）或 Brave（api.search.brave.com）→ 设环境变量 `TAVILY_API_KEY`/`BRAVE_API_KEY` → `[search]` 对应 `*_api_key_env` 填变量名（刀3 向导落地后改为面板内配置）。
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

## 2. 基线先行：30 query 固定集 ⏸（keyless 臂可与刀2 并行）

- 建 `docs/web-search-baseline-30q-20261105.json`（6×5 终稿表，digest 第四节），keyless 臂先跑：每引擎 queried/won/elapsed + 状态 + 离题标注。
- `must_hit/expected_state/as_of` 谁跑基线谁填；A 臂等主源拍板。
- 产出定数：保留窗毫秒数、覆盖门阈值、DDG 可达性、主源入 top5 比例。**不得先写死再凑。**

## 3. 刀2 首胜→短窗合并 + 主源保留窗 + 覆盖门 v0 ⏸（依赖刀1+基线定数）

- 主源保留窗语义（top5 必含主源否则 DEGRADED）；短窗收齐已完成引擎+URL 规范化去重；每引擎埋点。
- 覆盖门 v0（剥日期/数字后核心实体命中，复用 `search/search.go` bigram；误杀>10% 退 reranker 二档）。
- Codex 桩测五条作门禁（keyless 20ms vs 主源 300ms 必含主源／超窗=DEGRADED／同 URL 去重／nil=EMPTY 解析败=ALL_FAILED／三状态注入头分叉）。
- 验收：30 集 A/B/C OK 类 recall@5 ≥ 首胜基线；E 状态判对≥4；F 零误判 OK；单引擎行为等价现状。

## 4. 刀3 主源向导 + 限时 keyless + 全败才指引 ⏸（依赖主源拍板+刀2 埋点）

- 设置面板搜索配置卡（A/B 选择+测试连接）——当前前端零搜索配置绑定（`tools.ts:320-323` 只数编号），四表同批。
- 全败指引保持现位；人格路径/普通路径统一可见状态。
- 主源生效归因靠刀2 埋点（A 臂 OK 类 top5 主源出现率达标）。

## 5. 批次二（本批不做，防丢）

触发面升级（扩表 vs 4B 前置判别，先攒失败样本）；契约字段（日期/score/证据 ID）按消费方需求加；`ChatSend` 回传 `search_state` UI 可见性；Bing `ck/a` 跳转 URL 去重核验；public-searxng 并行化；搜索 TTL 缓存。

## 6. 纪律

- 一处一刀；不批量字符串补丁；不提前宣布全绿——门禁跑完再报。
- 办公 `Execute` 输出契约零变化（消费方是模型工具循环，不在本批动）。
- 工作树另有他方在途改动（boot sonic/voice UI），本刀提交只圈自己的文件。
