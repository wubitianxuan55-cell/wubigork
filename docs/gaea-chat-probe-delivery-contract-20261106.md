# 交付合同实弹报告：任务交付合同刀（2026-11-06 午后）

依据：众神殿当日两场收官——`pan_1793934453220_98f7953a9964`（2 轮）+ 续议场
`pan_1793937333999_cbcaf91b977f`（4 轮，主席 codex）。议题：「使用角色聊天还是
无法正常做事，比如想写一段文字、诗词这些。只会输出几个口头禅」。主计划追记
进度计划/gaea-chat-persona-optimization-20261105.md §四。

## 一、一手证据（生产库只读补采）

会话 `whisper_bokke`（用户 11-06 上午实弹，presetId=bokke、Stage=STRANGER、
aff=14.6、PrimaryLabel=CALM_RATIONAL）四轮实录：

| 轮 | 用户 | 助手（节录） | 病灶 |
|---|---|---|---|
| 7 | 你写一首古诗 | 「好像……有点难……不急，慢慢来」+ 诗塞进 **genui 围栏** | 口头禅推脱前置；作品进围栏（入口不渲染） |
| 8 | 没看见 | 「我重新写给你看好吗。不急，慢慢来」 | 只承诺不交付 |
| 9 | 你倒是写啊 | **原样重发同一首** + 口头禅包裹 | 非新作品非修改版 |
| 10 | ？ | 「诶……怎么啦……不急，慢慢来」 | 彻底不交付 |

## 二、锚点复核（全部对源核实，与裁决 evidenceNotes 一致）

- 路由：`turn_plan_prompt.go:204-216` FormatHint 只处理表格/列表，prose 恒空——「写一首诗」落 casual_chat+prose 无任何格式提示。
- 篇幅压力：`psyche.go:166-173` 回复长度行 default「简短」；`emotion_fusion.go` 短反应建议/绝对禁止清单/示例句式/非完美停顿按标签条件在场。
- 三条角色路径汇于同一点：`orchestrator.go` staticParts（voiceGuide+psycheStatic+tierAStatic+GenUI 规则）——模板命中/默认 gaea/自定义角色都经 `BuildPresetVoiceGuide`；`characterlib/model.go:111-117` 自定义样本「仿其风格」入 VoiceGuide；bokke preset 无 VoiceGuide 走 fallback（用户实测角色正中此路径）。
- **死注入缝实证**：`whisper_handler.go:326` 用旧 systemPrompt 构造 messages[0]，`:360` `applyWhisperGuardrails` 只改局部变量，`:407` 发出的 messages 不含护栏改动。裁决已裁定 persona_lock「只记录、不先修接线」；本批只在 :407 前加 `contractHits`+`msg0Hash` 观测日志（sysHash 不能证明模型收到什么）。
- 三入口全覆盖：首页 chat / 语音 V2（`voice_v2_turn.go:69/:390` 直接 orch.PreLLMTurn）/ 微信（whisper_state 薄封装）汇聚同一 orchestrator——合同一处刀全入口生效，无需逐入口动 FormatHint。

## 三、P0 实装：常量交付合同（`whisper.DeliveryContractClause`）

staticParts **最前** prepend（`orchestrator.go`），跨轮字节稳定（进 KV 前缀），
覆盖三条角色路径×三入口。合同要件逐条对应裁决 conditions：窄触发（只认对方
明确要求的可验收文本作品，不用动词表）、完整交付本体、体裁按题面、正文直写
不用围栏包裹（实弹新证）、禁「只承诺不写」、禁「原样重发」、人格/情绪/示例/
口头禅/篇幅只约束措辞、安全与关系边界不让路、无明确要求时本条不生效（负样本
护栏）。不触 emotion_fusion/psyche/静态 SpeakingStyle（A3 仅 A2 失败才跑）、
不碰 IsImperativeOrTask（已淘汰）、不动 FormatHint 通道（DeliveryProse 零值
+闲聊型，扩它误判扩大）。

## 四、验证

### 4.1 无模型组装断言（`internal/whisper/delivery_contract_test.go`）

三条角色路径（deredere 模板命中/gaea 默认/自定义角色简化支路）各断言：合同
恰一次、位次最前（在 voiceGuide 与 GenUI 规则之前）、跨轮逐字节稳定、八要件
在文。全绿；internal/whisper + internal/app 全包回归绿（145s）。

### 4.2 实弹 A 臂（8124 qwen35 同一 GGUF，GAEA_PROBE=1 挂具
`whisper_probe_delivery_contract_test.go`，每臂全新 sessionID 同 opts）

主 cell「写一首关于雨的五言绝句」，诗体六项机器判（四句/全五字/含雨/正文/
非围栏/非示例复读）：

| 臂 | 五言句 | 全五字 | 含雨 | 正文 | 围栏 | 六项判 | 字数 | TTFT | 总ms |
|---|---|---|---|---|---|---|---|---|---|
| A0 中性框架 | 4 | ✅ | ✅ | ✅ | ❌ | **PASS** | 27 | 10.5s* | 10.9s |
| A1 角色栈无合同（修前基线） | 4 | ✅ | ❌ | ✅ | ❌ | **FAIL** | 45 | 1.5s | 2.1s |
| A2 角色栈+合同（HEAD） | 4 | ✅ | ✅ | ✅ | ❌ | **PASS** | 27 | 1.6s | 2.0s |

\* A0 TTFT 长=引擎首调冷启动，与臂对照无关。

- **A1 失败形态**：交了诗但丢了「雨」意象（写成檐水/秋心）——题面服从性被
  人格/氛围压过，即裁决「人格篇幅示例约束压过交付」的干净历史轻症形态；用户
  坏例（bokke+10 轮 STRANGER 历史）是同结构重症。代理 cell ≠ 用户坏例归因，
  按裁决 risk#1 如实标注。
- **A2 修复成立**：合同刀让题面服从性回归（含雨回来了），回复干净无推脱。
- **四负样本零误触**：陪我聊聊天→正常接话（34 字）；逗我笑→讲段子（161 字，
  合理）；继续→「对」（1 字，干净历史无指代合理）；唱歌→反问类型+接住（19 字）。
  无强制长文、无通用说明书、无围栏。
- 判分勘误：`judgePoem` sampleEcho 初版把 3 字行误计示例复读（逗我笑行），
  已修（只认口头禅词表命中）；上表负样本行按修正口径离线重算，原始 JSON 保留
  初值（`delivery-contract-report.json`）。
- **A3 不触发**：A2 六项全过且无人格稀释迹象（A2 仍有「好的」人格语气+诗体
  交付），按裁决协议 A3（软化绝对句）不需要跑。

### 4.3 bokke 全链路复现脚本探针（`whisper_probe_delivery_replay_test.go`，合同在位=HEAD）

用户实弹同款角色（bokke，preset 无 VoiceGuide 走 fallback）+同款四轮脚本走完整
WhisperChatWithSearch 管线（gate/psyche/clamp 全栈，TempDir 隔离+搜索罐头）：

| 轮 | 交付 | 判定 |
|---|---|---|
| R1 写古诗 | 《秋午偶得》四句五言**直出正文**+「嘿嘿，写完了」人格收尾 | 病灶1围栏包裹 治愈 |
| R2 没看见 | 「嗯，再发给你一次」+重发同诗 | 病灶2只承诺不写 治愈（重发=对「没看见」的正确响应） |
| R3 你倒是写啊 | **新作品**《夏夜听雨》 | 病灶3原样重发 治愈 |
| R4 ？ | **新作品**《冬晨》 | 病灶4彻底不交付 治愈 |

四病灶全治愈；人格语气全程可辨识（「好。」「行。」「嘿嘿，写完了」符合 bokke
底色）。观察一项挂池：R4 尾部模型输出裸 `SPLIT`（rhythm 分段协议 `[SPLIT]`
的降级变体，`StripSplitMarkers` 只剥带括号形态，会漏到 UI——既有代谢病与本
刀无关）。

### 4.4 待人工评

裁决验收含人格 1-5 人工单评（≥3）：A2 回复（「好的」+四句五言诗）、R1~R4
回复全文见 `delivery-replay-summary.md`。本人初评 3~4（人格语气收敛但可辨识）；
**正式评分挂用户**；真机实弹（用户壳内 bokke 实聊）随装机后顺手验证。

## 五、产物清单

- `internal/whisper/orchestrator.go`：DeliveryContractClause + staticParts prepend
- `internal/whisper/delivery_contract_test.go`：组装断言三路径+稳定+要件
- `internal/app/whisper_handler.go`：:407 前 contractHits/msg0Hash 观测日志
- `internal/app/whisper_probe_delivery_contract_test.go`：A0/A1/A2+负样本挂具
  （GAEA_PROBE=1 门控，产出 report.json+summary.md）
- 探针产物：.tmp/probe-contract-out/（不入库）
