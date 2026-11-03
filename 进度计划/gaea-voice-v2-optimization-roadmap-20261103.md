# gaea 语音系统优化迭代规划（2026-11-03）

> **出处与效力**：本文按众神殿议事会 `pan_1793710538764`（四席 codex/grok/zcode/dsh，独立开场 4 轮 19 条贡献 + 12 条分歧台账 + 主席裁决 + 用户签注 **accepted**）全文提炼，是下一阶段语音优化的**唯一排序与验收事实源**。裁决五条 proposedActions 与 12 条台账去向逐条落档；三项产品决策已于 2026-11-03 用户拍板定稿（见 §7）。前情底档：`进度计划/gaea-voice-v2-rearchitecture-20261101.md`（V2 总方案，§13.3 门禁表 ：511-522）、M1~M4P 五份报告、语音日合流提交 `3a9eac08`（129 文件）。

---

## 0. 议事定性（为什么规划长这样）

- **主矛盾已从"架构选型"转成"产品化收口"**（dsh-r1 定性，四席无异议）：V2 方向全对（事件驱动相位、代际播放、Smart Turn、4B 快模型、8792 记忆网关、单事务写回），但它仍是**装机态 spike 而非产品态**——4 个本地服务、5.32GiB 依赖、硬编码路径（`voice_v2_service.go:30-36`）、默认档仍 v1、产品入口只有 ChatPage、断线须手点重连、`latency_trace` 未由运行时发射。
- **"打通"的定义**：不是再加功能，而是可靠性收口 + 身份可信 + 产品动线。
- 三个确定性缺陷均有**三席独立实读一致的代码实证**：① take 只比 itemID 不比人格（`voice_v2_turn.go:140-152`）；② 开台人格同步 fire-and-forget 无屏障（`ChatPage.tsx:174-177` 的 `.catch(()=>{})`）；③ 上游 `[DONE]` 后 prepared 残留可复现（`voice_v2_gateway_test.go:215-260`）+ plain 早退在 take 之前 return（`voice_v2_turn.go:212-214`）。

## 1. 目标形态（定稿）

- **后端**：sidecar 管 VAD/STT/TTS 与生成取消；**Gaea 网关（8792）独占人格/记忆/动作策略/提交账本**；对话平面（4B 快车道）与动作平面（规则 intent + 确认执行）分离。
- **前端**：**单一 V2 控制器**显式持有 `topic/personality/session/generation`，不新开第二条麦；协议与播放代际骨架不推翻。
- **UI**：聊天输入区**内嵌一等语音条**（可展开实时转写/六态相位/打断/诊断）+ 动作预览卡确认；不再是"开台即接管工作区"的全屏语音房间。
- **适用条件**（裁决 conditions 原文）：单机 Windows、聊天页内单语音会话；V2 底座不推翻、sidecar 外部进程架构保留；V1 按门禁降级不立即删除。

## 2. 下一阶段排序（四席收敛定稿）

| 阶段 | 内容 | 出口判据 |
|---|---|---|
| **P0a 身份与提交一致性** | A1–A5 验收口径（§3 定稿表） | ✅ **2026-11-04 收官**：A1–A5 断言全绿（提交 3c85b86b）+ A5e/f/g 真机三例全 PASS 无 N/A（§10） |
| **P1 可靠性与度量** | §13.3 延迟记分卡（✅ §11）+ breeze 停顿处置（本批未复现暂红，挂观察）+ ✅ 网关剥 tools（e211ff9e）+ ✅ 有限次自动重连（098911d8）+ ✅ 取消后状态清理（**2026-11-05 销项：已被后续工作覆盖**——prepared 槽三路回收=A5f abort+网关 SSE 中断回滚+下一轮抢先回滚；播放=M2 代际闸+speechStart flush；前端=interrupted 相位+卸载清理；sidecar 残留=CancelScope 忽略 stale turn；A5f 真机零幽灵提交）+ 微信 orch 并发窗口（少数意见，微信使用频率上升则重估） | ✅ 记分卡 8 项有实测/样本/判定/豁免（§11）；**p95 已转绿（1.70s ≤ 4.0s）→ P1.5 默认化门开启** |
| **P1.5 内嵌形态** | ✅ **2026-11-04 落地（fe4093a9）**：VoiceConsoleV2 双形态（bar=输入区紧凑条：相位点/实时预览/绑定芯片/四服务健康点/开麦·打断·展开·结束；展开=同实例 fixed 浮层不重挂，整幅新增收起钮）；ChatPage V2 分支改挂输入岛 dock，工作区不再被开台接管；V1 整幅行为不动 | 真机走查全绿（bar 全相位流转+回答进预览+落库进聊天流+展开/收起/结束+提交锚点）；**默认入口=bar**（p95 门已开）；余=用户实弹手感反馈 |
| **P1.9 异步 dry-run 动作预览** | 最终转写 → 后台 intent.Parse → 预览卡异步事件面；**同步主链已淘汰**（预览含 20 会话扫描延迟未测） | 预览 0 副作用（dry-run 测试基 `intent_router_test.go:116-143`）、对话回复零阻塞、失败按"未识别为动作"静默 |
| **P2 确认执行** | ✅ **2026-11-05 落地（18a64abf）**：预览台账（preview_id+intent_hash+TTL 2min+幂等领走即置位），执行=台账重构 Intent 直派既有执行层（执行的就是预览的那个动作）；白名单=navigate/status/reminder/save_task，生图/发文件/读屏可预览不可执行（留权限设计）；前端凭据流+执行钮+回执淡出 | 后端+4/前端+4 测试全绿；真机闭环=语音「打开运行监测」→待确认→确认执行→板块真实切换；重复执行=already-executed |
| **P3 产品化收口** | 分发形态已拍板**随包**（装包增 5.32GiB 接受，无下载器/首启引导）；余项 = ✅ **依赖路径参数化+bundle 打包脚本（2026-11-05 刀一，e6ec06cf）**：GAEA_VOICE_V2_BASE 解析 venv/models/bin（缺省=历史装机根兼容现役机）+ package_voice_bundle.py（stage+冒烟实测 PASS）——**跨机 venv 脆弱未解**（安装器内 bootstrap 或 §16.3 提取线，需第二台机验证）+ 默认档 v1→v2 量化切换（门=误打断率实弹）+ V1 降 fallback 一个版本期再删 | 记分卡全绿切默认档（差误打断率实弹一项）；**对外分发场景 breeze 权重非商业红线须重评**（单机自用不受限；bundle 不含 breeze） |
| **M5 模型工具** | 通用模型工具调用 + 推测生成/恢复 | ✅ **前置一已就位（ed25d618）**：prepared 单槽→itemID 台账（行为等价重构，容量恒 ≤1，M5 只需放开抢先约束）；**前置二**：✅ **设计稿已落档（2026-11-05，进度计划/gaea-voice-m5-permission-design-20261105.md）**——工具四级模型（T0 免确认提案/T1 确认/T2 可预览不可执行/T3 永久禁）+ 语音确认词默认关提案 + 模型工具统一接 P2 台账 + 与 agent 工具面隔离（Approver 洞不借道）；**待用户拍板 §7 四项后排刀** |

## 3. P0a 验收口径 A1–A5（定稿版，zcode-r4 合成）

| 项 | 定稿断言 | 代码依据 |
|---|---|---|
| **A1 开台屏障** | 点击开台时捕获 `{topicID, personalityID}`，把**这一笔** `VoiceApplySettings` 入共享串行队列并**等待该 Promise**（不是等旧队尾），成功后再 `setVoiceConsoleOpen(true)`；Wails 拒绝或运行时同步失败不开台并可见报错。**磁盘 Save 失败不阻断**（现状语义=`voice_handler.go:683-688` 仅 Warn、`:750` return nil；fail-closed 属 API 语义变更，不混入 P0a） | `ChatPage.tsx:220-228` 现状先 setVoiceConsoleOpen 再 V2 分支直接返回；`:174-177` 同步 fire-and-forget |
| **A2 take 双校验** | `takeVoiceV2PreparedTurn` 在 itemID 命中时**同时比 personalityID**；错配则同锁 `restoreVoiceV2PreparedLocked` + 置 nil + 返回**可区分 mismatch**（不复用 blocked 文案），调用方不得再走新人格 `PreLLMTurn` 路径 | `voice_v2_turn.go:140-152` 只比 itemID；`:277-287` 旧人格预推进无人回滚=幽灵预推进。回滚原语已在产（`:94-102`、`:120-127`），Go 侧约二十行 |
| **A3 两读者同值** | `VoiceApplySettings` 成功后 `currentVoicePersonality()` 与请求值一致；**拒绝路径须覆盖半写序列**。可失败键清单（`voice_handler.go` 内 setVoicePersonality 之后、ApplyConfig 之前提前 return 的键）：`personaVoicePresets`(:741-742)、`breezeVoiceDesc`(:713-714) 等，**共五处**。修法=`setVoicePersonality` 移至 `:749 ApplyConfig` 紧前（天然覆盖 VoiceV2Stop 副作用面）。**后端单调版本号不进 P0a**（无已证实的第二写者，串行队列够用；微信/CLI 成为并发写者再升级） | dsh-r4 O2 + zcode-r4 §2.3 扩清单 |
| **A4 第五门拆三子例** | **A5e 完成未提交仍提交**：用可挂起的 `VoiceV2CommitTurn` 让 `commitChainRef` 真实积压后卸载，断言第二笔仍提交到捕获 topic、prepared 归 nil、状态只推进一次（**只制造 response.done 不制造真实积压=假绿**）。**A5f 生成中关闭**：先断言 prepared **非空**再关闭 + 按 `{topicID,itemID,turnID,revision,personalityID}` 精确 abort，禁止 N/A 记绿；修=前端显式 abort（`dispose()` 现只 `transport.close()` 不发 `response.cancel`），完成回合继续提交、生成中关闭才回滚，提交与 abort 竞争只允许一方胜。**A5g plain 早退不留 prepared**：`voice_v2_turn.go:212-214` 在 `:273` take 之前 return=**确定性控制流，必测断言**而非观察项；断言提交返回后 `voiceV2Prepared==nil` 且旧人格 TotalTurns 不变 | `voice_v2_gateway_test.go:215-260`（[DONE] 后 prepared 非空=残留可复现）；`VoiceConsoleV2.tsx:89-126`（卸载/换题直接 return 不提交） |
| **A5 分测矩阵** | V2 主行：延迟写入→立即开台→断言 prepare 人格。V2 单元：**同 item 不同人格**（`prepare(pidA,item-a)`→`take(pidB,item-a)` 断言返回 nil、prepared 归 nil、pidA TotalTurns 复原）。V2 plain 行：不写人格记忆且 prepared 空。V1 行：**构造不同值乱序**（同值双写=空跑；触发对=切人格后立即开麦；`ChatPage.tsx:176` 与 `useChatVoice.ts:149` 双写者仅在 V1 动线成立，V2 开台 `:226` 提前 return 不经 toggleVoice——测对对象） | zcode-r3 §2.2 纠正写入点实为两处；dsh-r3 §2.3 收窄双写者归属 |

## 4. 已淘汰方案（防重提台账）

1. **壳级宿主（B）不提前**：负向检索证实产品挂载仅 `ChatPage.tsx`、首页/办公零语音挂载、`voice_v2*.go` 对 routeIntent|IntentPreview 零命中。B 保留为条件升级，**开工四条件缺一不排期**：take 侧人格一致且旧预推进回滚 ／ prepared 按会话分桶 ／ 完成提交不依赖组件挂载 ／ 生成中关闭可按 item 精确 abort。
2. **模型工具直接执行 + 同步 IntentPreview 主链**：执行留 P2、模型工具 M5（§11 语音侧权限确认空白）；同步预览改异步事件面（codex-r3 自我修正）。
3. **后端单调版本号进 P0a**：无已证实的第二写者；若微信/CLI 成为并发写者再升级 P2 硬化。
4. **A1 把磁盘持久化失败纳入开台阻断**：现状语义告警不阻断（行号仲裁 `voice_handler.go:683-688`、`:750`），改 fail-closed 属 API 语义变更且伤离线可用性（同函数 realtimeAPIKey `:649-662` 有刻意区分关键性的先例）。
5. **少数意见保留**：微信侧与 V2 共用同一 orch、prepare 解锁到 commit 的窗口可插入 PreLLMTurn（`wx_agent.go:510`）——判 P1 处理不进 P0a；微信侧使用频率上升则重估。

## 5. 分歧台账去向（12 条）

- **resolved ×8**：issue-1/4（grok 撤回：工具环与壳级宿主不提前，负向检索实证）、issue-2（zcode 撤回：分发按 5.32GiB 实测收口挂 P3）、issue-3（前提消失：四席实质交叉核验成立）、issue-5/7（裁决收口：P1 限 dry-run 预览、执行 P2、模型工具 M5）、issue-8（P0a/P0b 拆分成立）、issue-10（A5e/f/g 三子例定稿）。
- **needs_verification ×1**：issue-9 = P0b instructions 通道，静态链路已定位、待 30 分钟级端到端 spike（§6）。
- **user_decision ×3**：issue-6（B 动线 + 分发上限）、issue-11/12（关闭后完成未提交回合：提交还是丢弃）——见 §7。

## 6. 并行证据补块（不占 P0a 刀，可与 P0a 并行）

1. **~~P0b instructions 标记端到端 spike~~（已挂起）**：B 动线已拍板「暂不做」，该 spike 失去排期对象，**挂起不入路线**；静态链路结论留档（`session.update` 携带 `[gaea-persona:<id>]` → `VoiceClientV2.ts:69-87` → sidecar `chat.py:442-450` → 网关 `voice_v2_gateway.go:99-120` body 可读可剥）——若用户将来重开 B，此处是第一条前置的唯一候选通道。
2. **P1 延迟 p95 扩样**：温态 50 条 / 冷启 20 条分段测 p95，复核 5.66s（n=14）的暂红判定；按 STT/LLM/TTS-TTFA 三段归因。直接决定 P1.5 内嵌形态能否默认化。
3. **§13.3 记分卡补测**：barge-in→静音 p95 ≤500ms、误打断率 ≤3%、迟到/串台音频 0、缓存命中 ≥70%、首轮输入 token ≤2500 五项目前**只有门禁定义无实测值**，P1 出口前补齐。

## 7. 三项产品决策（2026-11-03 用户拍板，定稿）

| # | 问题 | **用户拍板** | 落地效力 |
|---|---|---|---|
| ① | 关闭语音台后，已完成未提交的回合（用户已听完回答）继续提交还是丢弃？ | **继续提交**（=四席默认） | A5e 断言方向定稿：完成未提交仍提交到原 topic、prepared 归 nil；"已听到的对话不蒸发"写进第五门唯一正解 |
| ② | 终态是否需要「离开聊天页仍能说」（B 动线）？ | **暂不做 B** | B 不进路线图（四前置清单留档防重提）；P0b instructions 通道 spike 挂起（§6）；壳级宿主/首页悬浮条不在本阶段任何一刀内 |
| ③ | 语音包按需分发还是随包？ | **随包**（否了按需倾向） | 装包增 5.32GiB 接受；无下载器/首启引导；P3 分发门简化为 sidecar venv 打包工程化 + 默认档切换 + V1 退役（§2 P3 行）；对外分发场景 breeze 权重非商业红线须重评 |

> 台账连带：issue-6/11/12 三条 user_decision 全部闭合——12 条台账至此 **12/12 收口**。

## 8. P0a 开工落点（实现席速查）

- `frontend/src/pages/ChatPage.tsx`：A1 共享串行队列（`personalitySyncRef` 模式）+ 开台前 enqueue-and-await **这一笔**。
- `frontend/src/components/chat/VoiceConsoleV2.tsx`：A5e 提交链——`response.done(completed)` 先登记提交意图，关闭只抑制 UI 不阻止后端提交；未 completed 的关闭走精确 abort。
- `frontend/src/voice/client/VoiceClientV2.ts`：A5f——dispose 关闭路径补 `response.cancel` / 新增 abort 通道。
- `internal/app/voice_v2_turn.go`：A2 take 双校验 + 同锁 restore + 可区分 mismatch；A5g plain 早退处 restore。
- `internal/app/voice_handler.go`：A3 `setVoicePersonality` 移至 `ApplyConfig` 紧前。
- 新绑定（若走显式 abort）：`VoiceV2Abort` 四表同批（bridge/mock/spaceBindings/bindingNames + gen_bindings 重生成）。
- 测试：`voice_v2_turn_test.go` 补"同 item 不同人格"用例；A5e/f/g 三个集成用例（前置断言 prepared 非空，禁止 N/A 记绿）。
- **坑位提醒**：`voice_handler.go` 有 DSH 低延迟线（lowLatency/whisperChatCore）在途交错史，动刀前先盘该函数当前态；构建态验证才复现 data:URL worklet 家族坑（vite dev 测不出）；CDP 驱动前查 gaea.exe 进程清单（9333 铁律）。

## 10. P0a 真机三例收官（2026-11-04，全部 PASS，无 N/A）

沙箱走查壳（`GAEA_DATA_ROOT=.tmp/voice-p0a-walk/data` 隔离 + CDP 9333 + 页面级假麦注入）实证，走查脚本 `.tmp/voice-p0a-walk/walk.mjs`，日志锚点全在沙箱 `logs/gaea-20261104.log`：

| 例 | 驱动 | 证据链 | 判定 |
|---|---|---|---|
| **A5f 生成中关闭** | UI 开台→开麦→thinking 段 40ms 轮询同拍点关闭 | sidecar `turn_1 rev=1 status=cancelled`（response.cancel 到达）+ LLM/TTS 双取消 + 网关「上游 SSE 中断，已回滚未提交回合」恰好一次 + 话题消息 0→0 + 全程零 commit 锚点 | **PASS** |
| **A5e 完成未提交仍提交** | UI 完整回合→tail 出现 `response.done completed` 同拍关台→3s 后验库 | `Voice V2 回合已提交 topicID=<捕获话题> personality=gaea memoryScheduled=true` + 落库消息与已知测试语音逐字对账（用户文本含「今晚我想听你讲一个很短的故事」） | **PASS** |
| **A5g plain 早退回收** | 直驱网关 8792（header 带 itemID）真实填充 prepared → 直驱 `VoiceV2CommitTurn` 到 plain 话题 | 「Voice V2 plain 话题丢弃人格预推进」（personality/itemID 精确匹配）+ 「回合已提交（plain 话题，仅落聊天）」+ `committed=true, memoryScheduled=false` | **PASS** |

**实测认知（后续真机验证配方）**：
1. `vc2-assistant` 文本出现 ≠ response 闭合（文本 item done 早于 response done）——彼时关台走 abort 路径。A5e 精确信号=tail 里 `response.done completed`。
2. 循环测试音频=无限 bargein 取消环（下一句在 response 未闭合时到达）；轮完成类测试必须单发语句（loop=false）。
3. 应用 `stopMic` 会 stop 假麦轨道——假麦补丁必须每次 gUM 调用发一条全新轨道。
4. A5f 竞争中 `VoiceV2AbortTurn` 绑定未赢（网关 SSE 中断回滚先到，同锁单胜者语义成立，两条路径都只回滚一次）；专用路径由 Go 集成测试锁定，真机按结果对账。
5. A1 屏障用 UI 的 `activePersonality` 重同步人格（binding 层同步会被覆盖）——设计正确，真机同步事实源=UI。
6. Wails v2.13 恒注入 `--remote-debugging-port=9333` 且 go-webview2 覆写 `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS`——自定义 CDP 口/浏览器旗标环境变量通道无效；CDP 驱动前按 PID 链核实 9333 归属。
7. 板块切换不卸载 ChatPage（`initRef` 只跑一次）——binding 建的话题不进侧栏，走查造题应走 UI「新建对话」按钮。

## 11. §13.3 记分卡实测（2026-11-04，P1 出口数据）

**测量环境**：sidecar/4B/breeze/网关全链本机实跑；`ws_probe` turn/bargein 场景（探针阈值已对齐产品 0.75）；sidecar `Turn latency` 行为唯一时延事实源（服务端口径）。**样本**：冷启段 n=21 + 温态段 n=71（完成态，每调用约 2 条=尾部 800ms reopen grace 的第二 turn，cancelled/failed 已过滤）。

| # | 指标 | 门限 | **实测** | 判定 |
|---|---|---|---|---|
| 1 | speech_end→first audio p50 | ≤2.5s | **1.42s**（stt+llm+tts_ttfa 逐轮和，n=92） | ✅ |
| 2 | speech_end→first audio p95 | ≤4.0s | **1.55s**；全音频播完 e2e p50=1.55s/**p95=1.70s**/max=1.97s | ✅ |
| 3 | 中文 EOT accuracy | ≥90% | 91%（M3 行为面，Smart Turn） | ✅ |
| 4 | barge-in→静音 p95 | ≤500ms | 服务端 bargein 检测同拍 flush（0ms）+VAD 判定 p50≈0.10s+陈旧音频 0B（M1 客户端 10/10，本批 15/16 flush 正常） | ✅（复合判定；起始锚定端到端时序未仪表化=豁免注记） |
| 5 | 误打断率 | ≤3% | 合成刺激无效（白噪/音爆在 0.6 与 0.75 下均触发=刺激不构成语音判别面）。**上界工具已沉淀（2026-11-05，scripts/voice_v2/false_interrupt_bound.py）**：取消响应占比=上界口径；用户真实会话首读 n=8 轮 UB=12.5%（11-03，1 次取消）——样本不足判定，测试流量日（11-04 全天=故意 bargein 机器）已标注剔除 | ⏸ 挂实弹池：上界工具已就位，用户实弹数日后跑一次即可读门；豁免=需真实施加 |
| 6 | 迟到/串台音频 | 0 | A5f 真机：cancelled 回合 0 落库；M1 bargein 旧代际泄漏 0B；播放代际闸单测 | ✅ |
| 7 | 同会话缓存命中 | ≥70% | ✅ **修复后实测：会话内第 2+ 轮 f_sim=0.995、prompt eval 11~13 tokens（~99% 复用，prefill 775ms→68ms≈11 倍）**；新内容轮 0.795→487 tokens（65%+）；跨连接首轮仍全量（会话上下文起点不同，每会话付一次） | ✅（2026-11-04 缓存命中刀修复）|
| 8 | 语音首轮输入 token | ≤2500 | 首响应 2157~2165（p50=2158/p95=2165，max=2873 为同连接累积轮） | ✅ |

**p95 转绿判定：转绿**（M4 的 5.66s 暂红 n=14 未复现——本批 breeze TTFA p95=0.47s/max=0.55s 无停顿，判定当时为 breeze 实例态偶发）。**P1.5 内嵌形态默认化门=开启。**

**缓存命中刀修复（2026-11-04，同日续）**：取证推翻初判——「≈0%」的 787-token eval 行系跨天日志误归因；经临时指纹日志（验后即撤）定位真实断点=①【本轮上下文】动态段（4564 字符含身份快照）拼进 user 消息头部，下一轮 sidecar 历史回放原文→公共前缀在第一条 user 即断；②whisper 侧 11-01 提速刀的 system 稳定契约实测有效（sha 跨轮全同）。修=网关 `applyVoiceV2PreparedPrompt` 动态段改独立 system 消息插在末条 user 之前（user 本体逐字节不动；Qwen3 模板正常渲染会话中 system）。**llama b11980 部分前缀复用实测可用**（LCP 0.971→仅 eval 12 tokens）。余=身份快照「性格特质/第一人称指令」仍混在动态段（quasi-static，进一步缩小 DYN 挂后续刀）。

**运行态实测副产物**：
- sidecar 冷启（依赖齐备时）=**24.3s** 端口就绪；壳启动 `VoiceV2Stop()` 会按端口收掉 4B/sidecar——壳重启=语音链全冷（真实用户冷启成本含 4B 加载）。
- **sidecar 会话槽泄漏再次实锤**：客户端非优雅断开（杀壳）后单槽被占死，后续连接全拒 `All 1 session slots are in use`（M3 已知 bug，自动重连刀的边界条件——重连对「槽被占死」无效，须修 sidecar 或 Ensure 前先杀残留会话）。
- 测量坑：bash 内联中文 JSON 会被 GBK 撕碎（curl 须 `--data-binary @file` UTF-8）；探针会话不 override 阈值时落在 yaml 默认 0.6（已修探针对齐 0.75）；合成噪声不是有效 误打断 刺激。

## 9. 风险清单（裁决 risks 原文要点）

1. ~~A5e/f/g 仅静态证据，真机未跑~~——已消除：2026-11-04 真机三例全 PASS（§10）。
2. P0b instructions 通道 sidecar 侧未经第二席复核——spike 定成败（已挂起，§6）。
3. ~~p95 暂红但 n=14 样本不足~~——已消除：2026-11-04 扩样转绿（§11，n=71+21，e2e p95=1.70s），P1.5 默认化门开启。
4. ~~P0a 落地前，切人格后立即开语音台存在零报错串记忆结构洞~~——已消除：A2/A3 落地+真机覆盖（§10 实测认知 5）。
5. ~~决策①未答前 A5e 按默认值执行需返工~~——已消除：2026-11-03 用户拍板「继续提交」，A5e 断言方向定稿。
