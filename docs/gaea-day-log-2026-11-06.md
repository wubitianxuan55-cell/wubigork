# 当日总账（2026-11-06 凌晨）：聊天板块核心优化第一批+实弹基线+臂B定案

> 追记：本日实际推进三批——①四卫生切片（见下）②实弹基线对照（13b65f37 tag vs
> 16b52dda，docs/gaea-chat-probe-baseline-20261106.md）③臂B 四刀+实弹验证
> （docs/gaea-chat-probe-armb-20261106.md）。两轮实弹全部 8124 qwen35（与 herdsman
> 槽位同一份 GGUF）+TempDir 隔离+搜索腿罐头，挂具 GAEA_PROBE=1 门控。

## 追加提交索引（臂B，5 笔）

| 提交 | 内容 |
|---|---|
| d99ee720 | 探针挂具+基线对照报告（五桶×5+15轮+重启差分+并发首轮全字段） |
| 0a69f2c5 | 臂B刀① Stage 晋级：中性轮计正向延续（Stranger 57 轮不晋级=实弹主靶） |
| 7bfff3fb | 臂B刀② EvtCold 义务豁免：短祈使/任务句+短问句不判冷，问句词表补「谁」 |
| 03debeeb | 臂B刀③ ≤15 字钳制豁免祈使/任务句 |
| 7fad82b4 | 臂B刀④ 压制文案拆分：限长不禁话题，回应用户句为义务（六行全换口径） |
| fba8e27b | 臂B 对照报告+主计划定臂收口 |

## 臂B 实弹结论

Stage STRANGER→FAMILIAR；whisper 46/46 恒注→连续段 0；五桶正文 23→45 字；
Trust 38.5→50.5；钳制 21→11（寒暄限长保留）；零回归零空回复。A′ 纯文案臂
不再需要（实弹双重证伪）。待办一条：重启续接指代消解（最近未完成任务优先）
随多轮续接刀入设计。

## 再追加：来源字段契约（a859524c）

chat_history 行级 source="raw"（Exchange.Source，四写入点盖章：chat 主链/
语音 V2/微信 agent）；persist 只写有标记行=遗留旧行零改写零误标；未知形态
不回灌。无 schema 迁移（rows_json 无模式扩展），action-8 的副本演练迁移面
不存在（旧数据一字不动）。三例契约测试+全量回归绿。

## 三追加：多轮续接刀（1e60f5c8）

旧天气链坏例（明天天气→四川成都→我要天气→成都 全转城市介绍）收口：
PendingSearch 随 state_json 持久化（重启续接同口径）+实体短答判定
（IsSearchFollowupAnswer：2~10 字/非问句/非祈使/非冷淡）+一次合并检索
（上轮查询词+实体）；回合末即消费/清除宁漏勿误；cleanChatSearchQuery 剥
「我要/我想」前缀。三场景测试过，全量回归绿。余：有界工具循环（等真实
办事坏例）。

> 依据众神殿两场连议收官结论（pan_1793883599669 + pan_1793885604229，议题「角色（带人格）
> 聊天胡乱说话、不作为」，用户已确认小结）：「胡乱说话/不作为」拆成来源污染、错误压制、
> 假成功、连续性缺口四类病因，首批四切片互相独立可单独回滚。主计划=进度计划/
> gaea-chat-persona-optimization-20261105.md。本批 5 笔提交，全部只读论证→代码落地，
> 压制文案包/续接/工具循环按裁决挂池未动。

## 提交索引（5 笔）

| 提交 | 刀 | 一句话 |
|---|---|---|
| 13b65f37 | 刀0 | 探针字段基建：TurnTrace L3 增 GateLevel/StrategyLevel/ClampInjected/InjectedFactIDs + whisperChatCore 收尾探针日志（raw/llm 长度哈希、历史面、终态、ttft）+ `chat-persona-baseline` tag |
| cfff7a0f | 刀1 | 空正文/取消守卫：非取消空正文有界重试一次仍空→显式 error；空轮/取消轮不推 WM/PromptHistory/不 FinalizeTurn/不入记忆；appendChatExchange 移到 cancelled 判定后；AppendExchange 空 assistant fail-loud |
| f429cecd | 刀2 | 生日显式化：删 nil→now 隐式默认；gaea 显式接 06-11（死常量首次接线+ParseAckemBirthdayMMDD）；存量等值行自愈（非 gaea 清 nil/gaea 置 06-11，幂等+先备份） |
| a931d216 | 刀3 | 搜索入口 raw/llm 双通道：Raw 进 TurnPlan/PreLLMTurn/WM/FinalizeTurn/记忆，增广块只进 llmUserMsg+PromptHistory；KV 缓存契约逐字节不变 |
| 16b52dda | 刀4 | 重启预热+发布序：先恢复后发布（并发首轮拿不到半成品）；chat_history 干净行 ≤4 轮回灌 PromptHistory（脏行五标记跳过，重启不再失忆） |

## 当场逮到的真问题（测试先行抓出）

1. **建连期取消走 callErr 早退**：首 token 前取消被误报为故障（error 终态）而非
   cancelled——原逻辑该分支在 cancelled 判断之前；且 ChatStreamMessages 错误路径
   cancel 返回 nil，直接调用=panic（守卫测试当场红）。
2. **sin 导出测试种数据撞防线**：AppendExchange 拒空后，历史空行改走 AppendMessage
   单条口种入（导出跳过旧库空行的测试语义保留）。
3. **heal 测试初版漏恢复序**：orch 未先 LoadCompanionStateFromDB 就 heal，等值门
   直接早退——生产序=先恢复后自愈，测试照此修正。

## 验收

- 新增测试 12 例全绿：空流/仅reasoning/取消空轮三终态用例+AppendExchange 防线、
  生日显式化三例+自愈迁移三例、双通道唯一标记两例、重启回灌+并发发布序两例
- 全量：`go build ./...`、`go vet`、`internal/app` 全包（~139s）、`internal/whisper/...`
  （prefix_stability 三契约/并发/生日）全绿
- 双通道验收口径：唯一标记只出现在本轮模型输入与 PromptHistory（后者=本批明确不改
  的既定口径，跨轮重放治理挂池）；WM/chat.db 全原文

## 余项（挂池，见主计划 §三）

五桶+15轮+重启差分实弹基线（刀0 字段已备，需本地引擎空闲+授权，先 baseline tag
后修复 HEAD 各跑一遍）→ 压制两臂实验（A′ 文案 vs 意图/分类门控，六条文案五代码位）
→ 来源字段契约立项 → 多轮续接（HEAD 复现门槛）→ 有界工具循环（真实办事坏例门槛）。

## 打包记录（2026-11-06）

25 提交上 origin/main（d8aa6b3b..e4dc8da8，门禁 build/vet 0 复验后推）：
网络搜索线 8 笔（451b5d1c 统一检索核…a335b46e Tavily 记档）+ 聊天核心优化线
17 笔（13b65f37 探针基建/刀0 tag → cfff7a0f 刀1 → f429cecd 刀2 → a931d216 刀3 →
16b52dda 刀4 → d99ee720 实弹基线 → 0a69f2c5..7fad82b4 臂B四刀 → fba8e27b 臂B报告 →
ea0c5e12 day-log → a859524c 来源契约 → e4f6a9a7 记档 → 1e60f5c8 多轮续接 →
e4dc8da8 收尾记档）。工作树干净，baseline tag `chat-persona-baseline`=13b65f37 随推。

## 桌面构建记档（2026-11-06 10:34，v4.533.0）

- **版本真源=wails.json info.productVersion**（build/windows/info.json 模板消费）；
  仓根 versioninfo.rc 是不被 wails build 消费的历史遗留（已顺手四处对齐 4.533.0
  无害留档；v4.532.0 当年只改它的 FileVersion 一处=改在了死文件上，真源当时对了吗？
  已核：wails.json 4.532.0 是对的）。坑=GetVersionInfo 读 exe 版本恒空（历史各版皆然），
  版本辨识走 build time+哈希+本记档
- 产物 build/bin/gaea.exe 59,849,216B sha256 前缀 c117d477（44d32d0a 为 4.532.0 串首建
  已被覆盖）+三 DLL；wails build ~2min
- 装机 scripts/gaea_install.ps1：四件套哈希对齐 C:\AI\gaea-app，桌面 gaea.lnk 更新
- 拉起验证：杀旧壳 37324 → 新壳 PID 33800 @ 10:35:18 > 构建 10:34:52
- 本壳含聊天核心优化全线（探针字段/四守卫/生日/双通道/重启回灌/臂B/来源契约/续接）

## 构建方案更新：桌面散件清理改默认（2026-11-06 10:43）

- 病灶：gaea_install.ps1 的 -CleanDesktop 清理能力自拍板起一直默认关，历次装机
  没带开关 → 桌面残留散件 gaea.exe（v4.532.0，Nov 5 20:50）+3 DLL——双击即跑旧版
  （2026-10-31 旧壳翻车同款风险源）
- 方案更新：桌面散件清理（gaea.exe/3 DLL/gaea-go.bat 精确名匹配）改为**默认执行**，
  取证场景用 -KeepDesktopFiles 反向豁免；用户数据（桌面 .gaea/）永不触碰
- 新方案实装：散件 4 件清零，桌面只剩 gaea.lnk+.gaea；新壳 PID 42580 @ 10:43:09
  > build 10:34:52，INSTALL OK
- 范围留档：C:\AI\gaea-voice-v2（语音 V2 遗留：models/sherpa-zh-14m+venv）属语音线
  资产判定范围，本批不动；装机根 C:\AI\gaea-app 与 build/bin 均干净（当前版四件套）

## 工作空间整理（2026-11-06）

- **归档 101 个 9 月收官批次文档（git mv 零损失）**：docs/gaea-*2026-09*+slim 系列
  +8 月设计稿 49 个 → docs/archive/（惯例沿用，archive 达 102）；进度计划 202609xx
  批次 53 个 → 进度计划/archive/（新建）。两目录活跃面：docs 76（day-log/活跃线/
  规范/审计台账）、进度计划 20（voice/chat-persona/web-search 等 10 月后线）
- **物理清未管理沙箱 5 个**（均 gitignore 不入库，关键数据已录 git 报告）：.tmp/
  （探针报告已录 docs/gaea-chat-probe-*）、.tmp-cad-poc/（报告在 docs）、.tmp-pdf/、
  .tmp_contract_review/（项目成控管理协议.docx 挪 temp/ 保留）、.dsh-vision-router/
- **保留待拍板**（恢复点/备份，不可逆删除交用户）：backups/（8 月 costdb 重组前
  备份+README）、.acl-recovery/（10-10 ACL 恢复点）、.audit/（10 月审计脚本配套）、
  temp/（cmp_* 对擂证据+活跃沙箱）、.agent-teams/

- 附记：整理提交 7c73f79d 被 git add -A 卷入 frontend/wailsjs/go/models.ts——
  实为今日装机 wails build 自动同步的刀0 探针字段（TurnTraceL3 gateLevel 等
  四字段），内容正确无功能影响；语义上应随 v4.533.0 版本批，已推不改史。

## 交付合同刀（2026-11-06 午后，第二批）

依据当日两场众神殿收官（pan_1793934453220 两轮+续议场 pan_1793937333999 四轮，
议题「角色聊天无法正常做事，写文字/诗词只出口头禅」）。一手证据生产库只读补采
`whisper_bokke` 四轮（诗进 genui 围栏→「没看见」→只承诺不写→原样重发→彻底不
交付）。P0=常量交付合同 `whisper.DeliveryContractClause` prepend staticParts 最前
（跨轮字节稳定，三条角色路径×三入口一处刀全覆盖；死注入缝 :360 只记录不修，
:407 前加 contractHits/msg0Hash 观测）。实弹 A 臂（8124 qwen35）：A0 中性六项
PASS / A1 修前基线丢「雨」意象 FAIL / A2+合同 PASS，四负样本零误触，A3 不触发，
判分 sampleEcho 勘误（3 字行误计已修+离线重算）。报告
docs/gaea-chat-probe-delivery-contract-20261106.md；主计划追记 §四。待用户：
人格 1-5 人工单评+真机 bokke 实弹验证。

## 再追加：bokke 复现探针+v4.534.0 装机（2026-11-06 午后二段）

- **bokke 全链路复现探针**（5d04289d）：用户实弹同款角色+同款四轮脚本走完整
  WhisperChatWithSearch 管线——四病灶全治愈（R1 诗直出正文无围栏/R2 对「没看见」
  正确重发/R3 R4 均新作品《夏夜听雨》《冬晨》，人格语气全程可辨识）。挂池：
  R4 模型输出裸 SPLIT（[SPLIT] 分段协议降级变体，StripSplitMarkers 只剥带括号
  形态——既有代谢病非本刀引入）。
- **发版**：wails.json 4.533.0→4.534.0（f57100ae）；origin/main=f57100ae 三提交
  上推（0b211895 合同刀/5d04289d 复现探针/f57100ae 版本）。
- **装机**：C:\AI\gaea-app 四件套哈希核对（gaea.exe sha256 073C2604FBF0…，
  60,114,944B）+桌面 gaea.lnk+散件清理+新壳 PID 44688（14:29:23>构建 14:16:46，
  PID AFTER BUILD OK）。冒烟：smoke.ps1 两次挂死在探活环（Invoke-WebRequest 在
  本会话 PS 链路挂死、90s 死线未触发，脚本问题非产品问题）；改直启等价冒烟
  PASS（桥接 18999 起、/api/health {"status":"ok"}）。
- **build.bat 环境坑（新，待 bisect）**：本会话 build.bat 三连败于绑定生成子进
  程——①0xc0000135（DLL_NOT_FOUND；PATH 前置 build/voice-dlls 可过）②三败点=
  「open frontend/wailsjs/runtime/package.json: Access is denied」，定位到
  wails v2.13 库 generateBindings 的 RemoveAll(runtime)+Extract 路径；而
  `wails generate module` 与裸 `wails build -s`（同 CGO/CC/PATH，无 TMP 重定向
  无 npm 步）均成功并完整重建 runtime 三文件（14:14 mtime 实证）。头号嫌疑=
  build.bat 的 TMP/TEMP 重定向到 .tmp（或 npm 步残留句柄），未定论；本批以裸
  build 出货（本批零前端改动，dist 无需重建），build.bat bisect 留下批。
- ACL 观察（.acl-recovery 关联）：frontend/wailsjs 树 ACE 含非法 authority 怪
  SID（S-1-4-538931142-703053989:W,D,DC）+他机 CodexSandboxUsers 残留；icacls
  /verify 全过、当前 token 读/写/创建实测全通，非本次故障因，留 .acl-recovery
  拍板面。
