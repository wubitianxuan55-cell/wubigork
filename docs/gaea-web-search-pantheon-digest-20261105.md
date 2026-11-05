# 众神殿「网络搜索能力弱」论证资料整理

> 会话：`pan_1793872913023_9c20fd7e7ab9`（2026-11-05，议题「如何解决gaea的网络搜索能力弱的问题，聊天板块与办公板块都需要」）
> 四席 codex/zcode/dsh/grok × 3 轮，status=completed，第 3 轮四席全签一致。
> 本文是论证原文的结构化台账；实施计划见 `进度计划/gaea-web-search-optimization-20261105.md`。

## 一、根因：四因子复合（全部【工具核验】，行号为会话当时锚点）

| # | 因子 | 证据锚点 | 后果 |
|---|------|---------|------|
| 1 | **双栈分裂** | 聊天 `internal/whisper/web_search.go`（仅 Bing HTML 正则→DDG Lite 两腿、5 条、私有清洗）；办公 `internal/gaea/tool/builtin/websearch.go`（6 引擎执行器） | 修一处另一处不受益；能力演进不对称 |
| 2 | **「非空即当真」** | `web_search.go:38-49` 全失败返回 `"搜索「x」暂无结果", nil`（几乎不返 error）→ `chat_service.go:52` 的 `result != ""` 分支把伪失败包装成「实时搜索结果」注入；`whisper_handler.go:766` 同款；`whisper_handler.go:713-727` `WhisperWebSearch` 直连旧栈且 `error` 字段恒死（err 塞 map 返回 nil error） | 失败分支全是死代码；plain 提示词 `chat_service.go:23` 还要求「不要说自己无法联网搜索」反向施压 |
| 3 | **现网零主源** | 主持人 r1 直读 `%APPDATA%\gaea\config.toml:55-58`：`[search]` 三键全空串；零 key 下 `buildEngines()` 只剩 public-searxng/bing/ddg 三件套（`websearch_registry_test.go:157-159`） | keyless HTML 抓取脆：验证页/改版即空 |
| 4 | **首胜单槽丢弃后到结果** | `websearch.go:127-168`：`resultCh` 缓冲 1，首个非空即 return，迟到结果 select-default 直接丢弃；`engine_order` 只决定 goroutine 构建序，不构成胜者权重 | **配置了主源也会被先回的 bing 盖掉**——「配了 key 未必改善体感」 |

伴随实证（grok r2 keyless 探针）：「2026年11月3日 人民币汇率」→ bing 返回 3 条**非空全离题**（百度百科「2026年」/节假日日历/万年历）——问题不止「搜不到」，是「搜到了也不能信」，且注入头仍称「实时搜索结果」。

## 二、会话中被证伪/淘汰的主张（防复提）

1. ~~「未配置即出配置指引/配置墙」（zcode-r2）~~ ——被 grok-r2 keyless 探针实证 3:1 推翻：keyless 非空可用，未试就出墙切断唯一免费路径。**定论：限时 keyless → 降级标注 → 全败才指引**。
2. ~~「engine_order 作同位次权重即可让主源生效」（zcode-r2）~~ ——单槽首胜证伪（三席独立核验）。**定论：主源保留窗语义**。
3. ~~「发布时间/质量分/证据 ID 前置第一批」（codex-r1/r2）~~ ——全仓 grep `publishedTime|evidenceId|qualityScore` 零消费方，提出者自撤。**定论：Tavily score（已在 `websearch_engine.go:233` 解析、`:281-286` 被丢）等字段只作刀2 排序信号**。
4. ~~「6 引擎扇出=聊天委托即获质量跃迁」~~ ——零 key 实际可用仅 3 引擎且 public-searxng 对 6 实例串行（`websearch_engine.go:171-181`），短窗内大概率赶不上。**委托的真实收益=一份实现/域名策略/SSRF 防护/统一诊断，质量跃迁来自主源**。

## 三、收口共识（第 3 轮四席全签）：三刀同批 1→2→3

**刀1 统一检索核 + 三层失败契约 + 四入口收敛 + 清洗边界**
- 四入口全走 adapter：`chat_service.go:26`（chatWebSearch seam）、`whisper_handler.go:765`（人格路径）、`whisper/agent_loop_runner.go:274`（agent loop）、`whisper_handler.go:713`（WhisperWebSearch 绑定，直连旧栈+死 error 字段）；删除还是委托交实现者，验收断言无绕过旧直连。
- 三层契约：①返回值状态枚举（EMPTY/ALL_FAILED 不伪装成功，**策略过滤后空≠ALL_FAILED**，`websearch.go:149-152` 现将其混入引擎失败）→ ②注入头按状态选文案（`chat_service.go:53`/`whisper_handler.go:768` 的无条件「实时搜索结果」头废）→ ③plain 提示词删「不要说自己无法联网搜索」（**仅 plain，人格路径没有此句勿加**）。
- 清洗边界写死：清洗归聊天 adapter 层（`cleanSearchQuery` 现为 whisper 私有须导出/下沉）；办公/agent loop 的模型结构化 query（含 `site:` 操作符）**原样不改写**；验收含「帮我搜索一下明天上海天气→明天上海天气」且**不再剥「在哪里/多少钱」**（现 `web_search.go:81` 会剥）。
- `agent_loop_runner.go:275-277` 死失败分支随换绑一并改活。
- 触发词表（`searchTriggers` v4.9.1）**不动**；`whisper.WebSearch` 自研 Bing/DDG 解析删除（删符号=源级守卫，绕过即编译红）。
- 办公 `Execute` 维持 JSON 数组输出（改状态信封须同批改工具 Schema+前端 `tools.ts:320-323` 摘要，本批不做）；状态走内部诊断。

**刀2 首胜→短窗合并 + 主源保留窗 + 覆盖门 v0**
- 主源保留窗：主源在本路径截止内返回时 top5 必含 ≥1 条主源，否则只能标 `DEGRADED(primary-timeout|primary-error)`，禁止静默 bing-only 标 OK。窗长由 30 集 A 臂 elapsed 校准（2.5s/90% 均建议值未校准）；聊天 8s 硬上限；保留窗在编排层实现，不动 per-engine 15s/总截止 20s。
- 每引擎 `queried/won/elapsed` 埋点；`engine_order` 若改语义须改配置注释。
- 覆盖门 v0：剥日期/数字后核心实体命中 title+snippet（复用 `internal/gaea/search/search.go:55-81` bigram，防日期灌水）；误杀 >10% 退本机 reranker 二档（热 410ms 可用、冷启 4.4s 不进聊天热路径，dsh r2 实测）。
- Codex 桩测五条作门禁：①keyless 20ms+主源 300ms→结果必含主源且保留他路独有结果；②主源超窗→`DEGRADED(primary-timeout)` 不静默 OK；③同规范 URL 去重且记双来源；④nil 结果=EMPTY、请求/解析失败=ALL_FAILED；⑤三状态注入头文案分叉。live test 失败仅 Logf 不判红，只留观测不作门禁。

**刀3 主源向导 + 限时 keyless + 全败才指引**
- `[search]` 三键空：限时尝试 keyless（8s 内要么带回降级标记结果要么全败指引），指引保持现位（`websearch.go:191-197` 在全败诊断之后）。
- A 托管 API（Tavily 免 1000 次/月 score 现成 / Brave 免 2000 次/月）或 B 本机 SearXNG（docker 一行）——**交用户拍板**；主源生效归因靠刀2 埋点。

## 四、验收基线：30 query 固定集（dsh-r3 骨架 + grok/codex 口径）

字段：`id / query / category / expected_state(OK|DEGRADED|EMPTY) / must_hit / expected_source(primary|any) / max_first_ms / as_of`。
口径：评执行器不评触发词；`expected_state` 随臂变（keyless 臂 A1 允许 DEGRADED）；D18/D19 指称空预设 DEGRADED；`must_hit` **谁跑基线谁填**（防先射箭后画靶）；每条记 `as_of` 防日期漂移。

- **A 时效事实×5**：人民币兑美元汇率(2026-11-03)／上海天气(2026-11-03)／2026 国庆放假安排／沪深300 最新收盘／92号汽油最新价格
- **B 技术专业×5**：golang 1.24 release notes／React 19 useEffect 变化／SQLite WAL 优缺点／混凝土养护规范 GB／2026 新能源汽车购置税政策
- **C 实体人物×5**：DeepSeek 创始人／哪吒2 票房／2028 奥运会举办城市／OpenAI CEO／2026 诺贝尔物理学奖得主
- **D 口语意图×5**：最近有什么新电影／帮我查一下明天会不会下雨／那个做芯片的公司怎么样／这个多少钱／帮我搜索一下今天天气
- **E 对抗离题×5**：2026年11月3日 汇率／苹果 价格／最新 消息／人民币 汇率 今天／2026年11月3日 北京 天气 空气质量
- **F 无结果长尾×5**：qwzxvbn 2026 特定型号 参数／site:example.invalid gaea 测试（兼做办公 query 不清洗回归样本）／不存在小镇 房价 2026／zzqxj 空气净化器 滤芯 型号／火星 房地产 均价 2026

判定：A/B/C OK 类 recall@5 不低于首胜基线；E 类状态判对 ≥4；F 类不得判 OK；OK 类前 5 条离题率 ≤20%；P95 首结果 ≤3s（目标值）。

## 五、遗留台账（会话收口时在册）

主源保留窗毫秒数与主源入 top5 比例需 A 臂校准；覆盖门剥词规则/阈值未跑样本；30 集 `must_hit/expected_state` 未标注；Bing `ck/a` 跳转 URL 跨引擎去重未实网核验（`websearch_html.go:52-54` 原样存 href）；`ChatSend` 是否回传 `search_state` 给 UI（仅可见性不阻塞）；触发面失败样本零条（批次二：扩表 vs 4B 前置判别，先记 `shouldSearchWeb=false` 且用户追问「你搜一下」的会话凑 30 条）；public-searxng 六实例串行未并行化；搜索结果无 TTL 缓存；搜索设置前端入口/绑定缺失（刀3 范围）。

## 六、会话裁决链（分歧台账 issue-1~7 全 resolved）

issue-1（无实质命题）issue-2（契约字段前置）issue-3（状态契约归属）issue-4/5（未配置行为）issue-6（engine_order 权重）issue-7（WhisperWebSearch 直连）——全部 closed，去向见第三节刀谱。
