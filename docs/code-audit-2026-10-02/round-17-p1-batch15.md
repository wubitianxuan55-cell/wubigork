# 全仓审计第 15 批 · 簇 B 真 bug（3 条并行线）· 2026-10-02

> 接续 [round-16（批次十四）](round-16-p1-batch14.md) §4.1 的「簇 B · 真 bug / 竞态」候选短名单。
> 本批**不抬版本**、不动 CHANGELOG（发版仪式归用户）；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `295b6d2a`（批次十四）；工作树干净。
> **文档定位**：每条结论带 `file:line` 与「谁实测」标注；审计原文必须能被现场复核，行号错=整条作废。

---

## 〇、主代理开工前预核（2026-10-02，读码实测）

| # | 审计项 | 主代理预核结论 | 关键证据（当前工作树） |
|---|---|---|---|
| 1 | **IN2-13** `ConsistencyReport` 同名两套定义、JSON 字段不兼容 | **成立且比审计更轻**：`types.ConsistencyReport/ConsistencyIssue`（`internal/types/types.go:49-61`）**全仓零消费**——`types.Consistency` 限定引用 0 命中、`overall_note` 除定义外全仓 0 命中、prompts 无引用；graph 版（`internal/graph/consistency.go:19-35`）是唯一活形态（消费方 `internal/app/consistency_deep_handler.go:76/:100/:144` + `graph_handler.go:169` + 前端 `ConsistencyPanel.tsx`/`consistencyDeep.ts` 全走 `issues/total_issues/summary` 口径）。git 历史（`-S types.ConsistencyReport`）显示消费方已在 `3d84c7a7`（死代码大清理）与 `b6f092f7`（P0 清账）被清 ⇒ types 版是**纯死定义 + 同名撞车**（同名同 `issues` JSON tag 语义不同，谁摸到谁解析出空字段） | 修法 = 删 types 死定义（非改名非兼容层）；graph 版注释标注「全仓唯一形态」 |
| 2 | **GA1-09** retry 循环 run 包装分支永不可达 | **成立，且原意判明=同会话续跑（不是盲删）**：①函数 doc（`internal/gaea/agent/task.go:712-715`）明写「After each retry the same session accumulates messages so the sub-agent sees the full failure history」；②重试提示词（`:761-763`）「Previous attempt failed the verification… Fix the issues above」只对**记得上次尝试**的子代理有意义；③死分支成因 = 会话藏在 `runSubSession` 内部（`:636-648` `run==nil` 时 `sess = NewSession(sys)` 不外传），作者想把「新会话续跑」接上没接成（`:727-730` 只在 `run != nil` 时给 `subSession` 赋值，`:738` 判 `run == nil && subSession != nil` 恒假）。**真实缺陷**：`run == nil`（ephemeral 模式，`prepareRun:809-813` 无 transcript store 时返回 nil）时每次重试全新会话、失败历史不累积——与文档语义相反，重试烧在重复失败上 | 修法 = `runSubSession` 回传会话 + retry 循环首试后包 ephemeral `SubagentRun{Session}` + 后续尝试不再重复 `Seed`（`:652-654` 防重复注入） |
| 3 | **AP1-07** 取消残稿三态落盘三份重复 | **成立（弱）**：`saveCancelledPartial`（`internal/app/create_chapter_handler.go:428-473`，三态=空稿只发事件/正稿已存在→侧车另存/全新→写正稿）行为 today 正确，被 `streamCreateChapter` 三处取消出口同构调用（`:558` 连接前取消 `summaryStarted` 恒 false / `:571` ctx.Done / `:581` error 帧夹带取消）⇒ 纯收敛项，非行为 bug | 修法 = 收敛调用形态或「保留+书面理由」（不许为改而改） |
| 4 | **AP1-08** 重写写回三副本 | **成立**：`NovelApplyRewriteVersion`（`novel_rewrite_handler.go:305-372`）与 `NovelRestoreRewriteVersion`（`:394-447`）各带「场景拼接/整章」两路，四段「写回+blob 同步+版本落账」结构重复。**语义差异必须保留**：Apply 场景路有 `SceneOriginal` 完整性校验（`:334`）+ 快照 Capture（`:340`），Restore 场景路两者皆无（对称设计）；整章路有 `rebuildScenesFromBlob`（`:363`/`:438`，v4 场景章写回后重置单场景）。同族既有 helper `writeBackRewritten`（`novel_llm_rewrite_handler.go:32`）不含 rebuild 与版本落账 | 修法 = 抽共享 helper，行为逐字段不变（错误文案/slog/返回 map/快照时机） |

**线切分（Go 包互斥）**：线 1 = `internal/types` + `internal/graph`（主代理直做，删死定义 + 唯一形态标注）；线 2 = `internal/gaea/agent`（GA1-09）；线 3 = `internal/app` 两文件（AP1-07/08）。三线足迹不相交。

## 一、逐线落地

### 1.1 线 1 · types 死定义删除（IN2-13）——主代理直做

- 删 `internal/types/types.go` 的 `ConsistencyIssue`/`ConsistencyReport`（13 行，零消费实证见 §〇.1）。
- `internal/graph/consistency.go:30-33` 注释标注「全仓唯一一致性报告形态……勿在别处再造第二份（审计 IN2-13）」。
- 验证：`go build ./internal/types/ ./internal/graph/ ./internal/app/` + `go vet` exit 0；`go test ./internal/graph/ ./internal/types/` ok；全仓 grep `ConsistencyReport|ConsistencyIssue` 仅剩 graph 定义与 `graph.` 限定引用。
- 反向证据形态说明：纯死代码删除的「改坏能红」= 复活定义即编译冗余（不可红），安全性由**零消费 grep + 全量构建**证明；防复活由 graph 注释 + 本台账承担。

### 1.2 线 2 · GA1-09 同会话重试接活（子代理 + 主代理复核）

**修法**：`runSubSession` 签名改 `(string, *Session, error)`（`internal/gaea/agent/task.go:582-584`，4 个返回点带 sess）；6 个不需要会话的调用点改 `_`（`:338/:353/:361/:473/:489/:544`）；retry 循环接活死分支——首试后 `run = &SubagentRun{Session: subSession}`（ephemeral wrapper，`Ref==""`）+ `seed = nil`（`task.go:750-753`）。

**子代理对预核的证据强化**：`Session.Seed` 是**整体替换**语义（`internal/gaea/agent/session/session.go:52-56`，`s.Messages = append(make([]provider.Message,0,len(msgs)), msgs...)`）⇒ `seed = nil` 不是防御性微调而是**必须项**——重试重注种子会把刚累积的失败历史整段抹掉。

**次生影响四项逐一读码核实**（主代理复核确认）：①MarkRunning/TrackProgress 闸 `run.Ref != ""` ⇒ wrapper no-op；②subJournal 同闸 no-op；③`subSink(ctx, run)` 的 `refSrc` 两形态都返回 `""`，Text 增量维持有意丢弃；④`subagentRunRef(run)==""` ⇒ SessionID 不登记，与 ephemeral 现状一致。`finalizeRun` 见 `Ref==""` 原样返回，零 transcript 写。

**测试**（`task_retry_session_test.go` 新增）：`flakyCheckBash`（首败后过）+ `TestRetryUntilSameSessionAccumulatesHistory`——断言第二次子代理请求**包含首试 assistant 回复**、恰 4 条消息（`[sys, user, assistant, retry-user]`，同时钉死 seed 重注）、check 恰跑 2 次、输出取二试答案。

**反向证据**（主代理亲手重做变异，不依赖子代理自述）：wrapper 段短路（`if false &&`）→ `TestRetryUntilSameSessionAccumulatesHistory` 红，红原文如实呈现旧行为形态（第二条请求只剩 `[system, user 重试提示]`，无首试回复）→ 还原复绿（`ok 0.257s`）。`TEMP-MUTATE|REVERSE-TEST|if false` 足迹内 0 命中。

**既有用例处置：零改动**。`grep retry_until|max_retries internal/gaea/agent/*_test.go` 仅 `task_retry_gate_test.go`（gate 语义，不钉会话身份）与 `task_fork_test.go`（fork+retry 互斥）——无任何既有测试钉死「重试=新会话」旧行为。

### 1.3 线 3 · AP1-07/08 写回与取消出口收敛（子代理 + 主代理复核）

**AP1-08（主修）**：新增两个写回 helper（`novel_rewrite_handler.go`，未新建生产文件）：
- `writeBackRewriteVersionScene`（`:321-352`，`applying` 分流）：Apply 路=读场景→`SceneOriginal` 逐字完整性校验→`SceneNew` 非空→应用前快照（失败仅 warn 继续）→写回 `SceneNew`；Restore 路=`SceneOriginal` 非空校验（**在读场景之前**，序不变）→读场景→写回 `SceneOriginal`，**无完整性校验、无快照**（「无条件还原」是刻意语义，helper doc 明写）。共用尾：Content+WordCount→`sm.Write`→`syncBlobFromScenes`。
- `writeBackRewriteVersionWhole`（`:355-361`）：`WriteChapter`（错误前缀 `写回正文失败`/`恢复正文失败` 由调用方逐字传入，`%s: %w` 产物逐字节同原）→`rebuildScenesFromBlob`。
- 落账（Status/AppliedAt 或 RestoredAt/RestoredFrom）、slog、返回 map 全部留在调用点——那是 Apply/Restore 的真差异。
- **与 `writeBackRewritten` 不复用的理由**：错误文案不同（转调即改文案，违反行为不变约束）、不写 `Meta.WordCount`、不含校验/快照/rebuild/落账、文件在足迹外。

**AP1-07（收敛）**：`saveCancelledPartial` 三态本体与签名不动（既有测试直调）；三处 9 参同构调用收敛为循环顶部闭包 `savePartial(summaryStarted)`（`create_chapter_handler.go:505-507`），7 个稳定实参归一，唯逐处取值的 `summaryStarted` 在各出口显式（`:565` 字面量 `false` = 连接前恒未进摘要段的语义保留）。

**行为不变证明（主代理逐段亲核 diff 后确认）**：错误文案、slog 级别与文案、返回 map 键值、校验顺序（读场景→完整性→SceneNew→快照→写回）、快照时机（写回前/内容=旧文/标签 `局部重写应用前`+`partial-apply`/失败仅 warn）、落账字段写序、WordCount 口径（`len([]rune(text))`）逐项同原。**现状钉子**（`novel_rewrite_writeback_pin_test.go` 新增 522 行 7 Test）：四条路径 Happy（含快照恰好 +1、Label/Trigger/WordCount 断言、v4 rebuild 对称重置单场景）+ 错误分支文案逐字（含完整性拒绝后零副作用、Apply 幂等篡改磁盘不重写、校验顺序钉死）+ slog 捕获逐字。**时序证明**：钉子先对原始代码跑绿→重构→一字不改复绿。

**反向证据**（主代理亲手重做变异 A）：helper 中拍丢 Apply 应用前快照 → `TestApplyRewritePin_ScenePathHappy` 红（`应用应恰好落 1 个快照，得到 0`）→ 还原复绿。子代理另做变异 B（给恢复路补完整性校验）→ `TestRestoreRewritePin_ScenePathNoIntegrityNoSnapshot` 红（`恢复（场景被改过也不得拦）`）。

**主代理独立复跑**：`go test ./internal/app/ -run 'TestApplyRewritePin|TestRestoreRewritePin|TestSaveCancelledPartial'` → 8/8 PASS；`go test ./internal/gaea/agent/ ./internal/graph/ ./internal/types/` 全 ok；`go build ./...` exit 0；gofmt 改动文件 0 命中。

## 二、跨线裁决与缝合

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **足迹互斥按 Go 包划**：线 1 = `types`+`graph`（主代理）、线 2 = `agent`、线 3 = `app` 两文件 | **有效**：三线并行期间主代理多次跑通 `go build ./...`（exit 0），无跨线签名互撞；两子代理报告均确认未触碰他人足迹 |
| 2 | **线 1 主代理直做**（删 13 行死定义不值得一个子代理波次） | **有效**：零消费 grep + 全量构建即安全性证明；防复活由 graph 注释 + 本台账承担 |
| 3 | **Seed 替换语义（子代理 → 预核的修正）** | 预核说「防重复注入」；实测 `Seed` 是替换 ⇒ 不置 nil 会**抹掉**累积历史——比预判更强，已进 doc 注释与钉子测试（4 消息形状） |
| 4 | **主代理双轨反向验证**：两条子代理线的变异均由主代理亲手各重做一组（线 2 短路 wrapper、线 3 拍丢快照） | 红→绿原文独立取得，「子代理跑不了 ≠ 不验证」升级为「子代理跑过 ≠ 只信自述」 |
| 5 | **IN2-13 处置 = 删死定义，非改名非兼容层** | 依据：`types.Consistency` 限定引用 0 命中、`overall_note` 除定义 0 命中、消费方已在 `3d84c7a7`/`b6f092f7` 清掉——改名会给死代码续命，兼容层给不存在的问题造机制 |
| 6 | **GA1-09 原意判定（审计要求「不可盲删」）** | 三证据链判定为「补真语义」：函数 doc 宣称累积（`:712-715`）＋重试提示词预设子代理记得上次（`:761-763`）＋死分支是接线未遂的半成品形态（会话藏在 `runSubSession` 不外传）——删死分支会让「文档语义与实现相反」的静默矛盾永久化 |

## 三、本批结论与量化收益

### 3.1 门禁与守卫（主代理亲跑，前台）

**本地快闸（`scripts/ci.ps1` 前台单次跑通，CI OK / exit 0）**

| 步骤 | 结果 |
|---|---|
| `version drift check` | OK（三处版本一致 4.454.0，本批不抬版本） |
| `.tmp hygiene guard` | OK（265MB ≤ 512MB） |
| `go build ./...` / `go vet ./...` | exit 0 |
| `golangci-lint`（钉 v2.14.0） | **0 issues**（首轮抓到线 3 新测试文件 1 处 gofmt 不齐，见下） |
| `go test ./... -count=1` | exit 0（全仓无 FAIL） |
| frontend `lint` / `build` | exit 0 / ok |
| frontend `vitest`（全量） | **427 files / 3712 tests passed**（与批 14 持平——本批零前端改动） |
| E 系列回归守卫 | OK |
| `check-docs` 卫生四查 | OK（AGENTS.md **56548 B** 预算内、孤儿 0 / 悬空 0） |

**守卫（收线单独复跑）**：`check-primitives --strict` 无新增（截断 37 / 原子写 8 / 前端 5 / heredoc 0）· `check-test-ctors` **364 = 基线零新增**（两个新测试文件都用既有 helper，无新裸构造）· `check-contract-drift` 无新增漂移（warn 档 exit 0）· `check-bindings-drift` **OK@744**（本批零绑定面变更）。

**过程事故（如实登记）**：首轮 ci 的 golangci 步抓到 `novel_rewrite_writeback_pin_test.go:237` gofmt 不齐（struct literal 对齐）——线 3 子代理自验只跑了 `go build`/`go vet`/`go test`（gofmt 仅 golangci 强制），报告里「gofmt clean」只覆盖它显式检查过的生产文件。**修法**=`gofmt -w` 单文件修复后复跑全量 ci 绿。**教训**：子代理自验清单缺 golangci 时，「格式干净」声明只覆盖它检查过的文件；收线全量 ci 仍是最后一道真闸（本批再次实证其价值）。

### 3.3 本批量化收益

| 项 | 变化 |
|---|---|
| IN2-13 | 同名异构两套 `ConsistencyReport/Issue`（`issues` 同 tag 语义不同）→ **graph 一套**；types 死定义 −13 行；graph 版注释标注唯一形态防复活 |
| GA1-09 | retry_until ephemeral 模式每次重试丢失败历史（与 doc 宣称相反的静默矛盾）→ **同会话累积**（文档语义成真）；恒假死分支消灭；Seed 替换语义钉进注释与测试 |
| AP1-08 | Apply/Restore × 场景/整章四段重复写回体 → **2 个 helper**；Apply（完整性校验+快照）与 Restore（无条件还原）的语义差异由钉子测试钉死 |
| AP1-07 | 三处 9 参同构取消出口 → **1 个闭包**（7 稳定实参归一） |
| 测试 | +2 文件：`task_retry_session_test.go`（1 Test）、`novel_rewrite_writeback_pin_test.go`（7 Test，522 行）；既有测试文件零改动 |

### 3.4 余量（如实登记，未硬凑）

1. `writeBackRewritten`（`novel_llm_rewrite_handler.go:32`）与新 helper 的进一步合并需统一错误文案（跨文件消费方），**留池**。
2. GA1-09 的 `run!=nil`（continue_from/持久 store）路径行为不变（本就同会话）；fork+retry 组合既有互斥闸不涉及。
3. `Session.Seed` 替换语义本身无第二消费方受影响（fork 种子单次注入路径未变）。

## 四、留池与待拍板

### 4.1 下一批候选（接 round-16 §4.1 未尽项）

- **簇 C · 口径单源**：IN1-09（AI 味判据三套严重度）· IN1-05（`Profile` vs `Fingerprint`）· GA3-07（知识库检索双口径）· IN3-02（日历语义 Go 四处 + TS 镜像）· FE4-04（绑定方法名三份手工清单）· GA6-01/GA6-04/GA6-06（成本条目族）。
- **簇 D · 死代码与僵尸资产**：IN4-04（桌面能力 0 生产调用路由——批 14 已实证埋雷，等接线拍板）· GA4-09（`SkillLayer.Learner` 桩）· GA2-03 · FE3-03 · X1-11。
- **既有拍板池不变**：桌面 agent 接线三件套、`download_and_install` 独立确认 UI、`GaeaDocumentLint` 读根门、FE6-09 conversationKey、AP4-01 三编排合并、X1-03 裸构造、`estimateTokens` 四口径。

### 4.2 本批新增留池

1. `writeBackRewritten` 与 `writeBackRewriteVersionScene/Whole` 的远期合并（需动错误文案，跨消费方）。
2. AP1-07 的 `saveCancelledPartial` 三态本体与 converge/scene_cards 的 cancelled 事件形态仍是两套事件词汇（`partialSaved` vs 裸 `cancelled`）——语义不同不硬并，登记备查。

## 五、审计原文纠错与遗留登记

| # | 审计原文 | 本批实测（谁测） | 处置 |
|---|---|---|---|
| 1 | IN2-13「消费端按哪套解析都可能拿到空字段」 | **危害表述偏重**：types 版全仓零消费（grep 实证），不存在「哪个消费端拿错」——真实形态是**同名撞车地雷**（下一个摸到 types 包的人会拿错）＋两套结构漂移的维护噪声（主代理预核） | 按死定义删除处置；graph 标注唯一形态 |
| 2 | GA1-09「要么删死分支，要么补真语义（须先判定原意）」 | **原意判明=补真语义**（三证据链，见 §二.6）；死分支不是笔误而是接线未遂（主代理读码） | 按接活处置，非盲删 |
| 3 | AP1-07「三态落盘三份重复」 | **「三份」指三处调用出口**（`:558/:571/:581`），三态判定本体始终单一——审计表述易误读为三份实现（主代理读码） | 收敛调用形态；本体不动 |
