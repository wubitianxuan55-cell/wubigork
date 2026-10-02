# 第十五轮修复记录（批次十三：P1 池清账 + 守卫精化）· 2026-10-02

> **承接**：批次十二（`03e0fc0c`，round 14）。批次十二在 §四 留池里留下 **P1~P8** 八条「已实测、已定位、但不在本批切口」的项，本批按「**不需拍板、可一刀落**」的优先级把其中的 P1/P2/P3/P4/P7 与两条 app 内 P1（AP4-07/AP8-08）+ 一条批次十一的接线欠账（GA3-13）做成四线。
> **范式**：4 条互斥并行子代理线 + 主代理定契约、收线缝合、全量门禁。**两个新纪律**（来自批次十二的事故）：①每条线被要求「接近上下文上限/长时间无进展时先用 `send_message` 把阶段性结论发回主代理」；②主代理不把「已派出」当「已完成」、不把子代理自述混同为自己的实测。
> **本批不做（需要拍板）**：AP4-01 三套编排合并、X1-03 五重 `*core` 嵌入、`estimateTokens` 四份口径合一（P5）——这三条仍在拍板池。

---

## 〇、本批候选与来源（承接 round 14 §四）

| 本批 fid/编号 | round 14 池位 | 内容 | 归属线 |
|---|---|---|---|
| D5 / P1 | 本批新入池 P1 | `upsert_task` 无「工期为负」闸（**真缺口**，实测 `Duration:-5` → `err=nil` 且落库） | 线 1 |
| X1-12 残 / P2 | 本批新入池 P2 | `ci.yml` 仍整包排除 `internal/tts`，理由引用的 `TestEdgeTTS` 全仓不存在 | 线 2 |
| X1-12 / P3 | 本批新入池 P3 | 前端 flaky 分类器注释自称「含 4 组夹具自测」，实测零自测 | 线 2 |
| X1-13 精化 / P7 | 本批新入池 P7 | 遮蔽闸归因切片靠字典序（超基线必红但明细可能指错条目） | 线 2 |
| FE7-09 残 / P4 | 本批新入池 P4 | 计划 rel 的第 4/第 5 份前端副本（`ScheduleApplyRollback.tsx` / `mock/office/schedule.ts`） | 线 3 |
| FE4-02 | 审计 P1（未触碰） | 工具参数 → diff 两套平行实现（`planDiff` vs `tools`，`changes` 第三份） | 线 3 |
| GA4-11 前端侧 | 批次十一余量 5 | 备份回滚「部分失败返回 error」是否被前端按错误展示 | 线 3 |
| AP4-07 | 审计 P1（未触碰） | 文件索引入队去重三入口两种空间口径（play 任务不被看见 → 索引抖动） | 线 4 |
| AP8-08 | 审计 P1（未触碰） | 板块 ID 白名单第二份（`intent_llm.go` 硬编码，含已删 `code`、缺 `schedule/sin/knowledge`） | 线 4 |
| GA3-13 接线 | 批次十一余量 4 | `knowledge.Service.MigrationState()` 已导出但 app 启动路径/面板未接 | 线 4 |

---

## 一、已落地

### 线 1 · 工期负值闸（D5 真缺口）+ 跨语言 golden fixture 同步

**任务 1 · `upsert_task` 补负工期闸（已落地，判据单源）**
- `internal/schedule/project.go:196-224` 新增 `validateDuration(t Task, idInMsg bool) error` —— 负工期判据**唯一实现**（范式对齐同文件的 `validateDurationUnit`），三调用点=`Validate`（`:92`，原内联整段删除、文案逐字不变）/ `upsert_task`（`ops.go:119-125`，置于 level 之后 progress 之前，与 `Validate` 同序）/ `patch_task`（`ops.go:193-209`，移入既有「生效态块」，**校验先于任何赋值**——顺带消掉批次十二 D4 同族的「失败留半改状态」）。
- 证据：`TestNegativeDurationThreePathsAgree`（Save→Validate / upsert / patch 三路径同结论矩阵：Save 报「任务 A 工期为负」且 `os.Stat` 目标 `IsNotExist`=**不落盘**；upsert「第 1 条操作失败：任务 C 工期为负」；patch「第 1 条操作失败：工期为负」；经 `ruleText` 去条序号+去 id 前缀后三处**逐字一致**）、`TestUpsertNegativeDurationRejectedAndNotPersisted`（新增态/同 id 覆盖态 `reflect.DeepEqual(p.Tasks, before.Tasks)` 均通过——旧实现会追加 `{ID:C Duration:-5}`；多 op 序列 fail-closed：A.Duration 保持 2）、`TestPatchNegativeDurationRejectedAndNotPersisted`（组合 `{durationUnit:cd, duration:-1}` 拒绝且**无半改**）、`TestNegativeDurationLegalStatesNotHurt`（0/正数/**缺 duration**=0 三例不误伤，回执含「工期 N 工作日」）。
- **反向证据**（4 组，mutate→FAIL→还原→PASS）：锚点 `ops.go:123`（upsert 闸换成 `_ = validateDuration`）→ `TestNegativeDurationThreePathsAgree` FAIL「upsert_task{Duration:-5} 应被拒绝（旧实现 err=nil 且把 -5 落进计划）」+ `TestUpsertNegativeDurationRejectedAndNotPersisted` FAIL「实际 err=nil，任务表=[… Duration:-5 …]」**逐字复现批次十二 D5** + `TestApplyOpsGoldenFixture` FAIL（错误口径与 after 状态双向漂移）；还原 → ok 0.869s。

**任务 2 · TS `opsSim` + fixture 跨语言同步（已落地）**
- `frontend/src/schedule/opsSim.ts:55-67` 新增 `negativeDurationError(t, idInMsg)`（Go `validateDuration` 的 TS 镜像、同判据同文案），upsert 分支 `:107-112`（`idInMsg=true`）/ patch 分支 `:167-173`（`idInMsg=false`，沿用 golden 冻结原文）。
- **fixture 是生成物**（唯一权威源=`internal/schedule/ops_golden_test.go` 的 `goldenCases()`，`go test ./internal/schedule/ -run TestApplyOpsGoldenFixture -update-golden` 重生成；TS `opsSim.golden.test.ts` 读同一份对拍）⇒ **按生成路径再生，未手改**；再生前先拿到漂移证据：FAIL「用例数漂移：fixture=65 computed=66」。
- `git diff --numstat ops_golden.fixture.json` = **112 / 0（纯新增、零删除）**，新增 2 例：index 8「upsert 负工期」（D5 缺口本体的拒绝路径）、index 17「patch 既有负工期只改 level（不误伤）」（**有意收窄边界的跨语言锁定**）；既有 65 条逐字节不变，`patch 负工期` 原文未变。
- TS 反向证据：把 `negativeDurationError` 的 `if (t.duration >= 0) return null` 改成恒返回 null → `opsSim.golden.test.ts` 4 例 FAIL（upsert 负工期 / patch 负工期 / 同一探针触及 duration / 缺 duration 与显式 0 放行）；还原 → 85 passed。

**预审建议的裁决（主代理建议「闸宽外溢」，线 1 有据驳回，主代理采纳）**：我预审时提议「patch 只要生效态为负就拒绝」，线 1 判定应**有意收窄**并给出理由——`ops` 文件头契约是「ops 只校验自己的改动，应用后由调用方统一 `Validate` + CPM fail-closed」；`validateDurationUnit` 在生效态块里判是因为 patch 的 level/isMilestone 会**制造**新的 cd 违规，而**既有** `Duration<0` 不是本次 patch 造成的非法态。外溢有两个坏处：与工期无关的补丁（只改 level/isMilestone）被拒、错误文案指向用户没碰的字段，且**同一条 op 在不同计划上随机成败**。收窄后不变量没丢（该计划照旧过不了 `Validate`/`Save`）。
- 落地为 `ops.go:204` 的 `if pt.Duration != nil { validateDuration(eff,false) }` 门 + 注释，TS 侧同门；
- 探针两侧齐备：Go `TestPatchTaskUntouchedNegativeDurationNotRejected`（构造 `Duration=-5` 的 Project → `patch_task{Level:0}` **放行**、`{IsMilestone:true}` 同放行、`{Duration:5}` 修复放行、`{Duration:-1}` 拒绝；并断言该计划 `Validate`/`Save` 仍拒绝）+ TS 手工探针 3 例（明示非 fixture 驱动）+ fixture 级跨语言锁定；
- **边界反向证据**：Go 锚点 `ops.go:204`（摘掉门）→ `TestPatchTaskUntouchedNegativeDurationNotRejected` FAIL「只改 level 的补丁不应被既有负工期绊住」+ golden FAIL；TS 锚点（patch 分支顶部硬插 `if (t.duration < 0) throw`）→ 2 例 FAIL；均还原后绿。

**主代理补做/追加的裁决**：①「`Duration` 缺省→0 不误伤」已在两侧钉住（Go 表驱动 + TS `?? 0` + 既有 golden 用例）；②本批**不做**「`ApplyOps` 出口即合法」（尾部挂 `Validate`）——那会改动全部 op 的既有语义与整份 fixture，登记为独立刀具（见 §四）。

**线 1 余量**：patch 通道文案仍不带 id 前缀（被 fixture 冻结，与 upsert/Validate 并存，延续批次十二登记项）；`ApplyOps` 单独调用不保证整计划合法（本文件既有契约）；fixture 为生成物，ops 语义改动必须走 `-update-golden`（本批实测该机制两次生效）。

### 线 2 · 守卫与 CI 精化

**任务 1（P2）`ci.yml` 的 `internal/tts` 过期排除（已收口）**
- 删除 `pkgs=$(go list ./... | grep -v /internal/tts)` → `pkgs=$(go list ./...)`（只动 backend job 这一处；race job 与前端 job 逐步骤对照 HEAD 未动）。
- **平台核查（这条是本任务的真正结论）**：`internal/tts` **无任何 `//go:build` tag**，也没有 `_windows.go`/`_linux.go` 变体；但 `sapi.go:55` 用 `syscall.SysProcAttr{HideWindow: true}`（**Windows-only 字段**）——线 2 实测交叉编译 `GOOS=linux GOARCH=amd64 go build ./internal/tts/` → `sapi.go:55:41: unknown field HideWindow`，exit 1 ⇒ **该包只能在 windows runner 上编译运行**，backend job（windows-latest）是本仓唯一能覆盖它的位置，纳入安全；ubuntu 的 race job 不能含它（本来也没有）。
- 也就是说旧注释**双重不实**：①`TestEdgeTTS` 全仓零命中（测试不存在）；②真原因是平台而非「依赖外网」。
- 宿主依赖排查：全包测试无 `exec.Command`/`powershell`/网络调用（`provider_test.go:260` 注明「不调用 `SynthesizeWithMime`（会拉起 PowerShell 子进程）」，`NewWinTTS` 只构造+断言），唯一真机项 `TestLiveVoxCPM2` 由 `HERDSMAN_LIVE != "1" → t.Skip` 自动跳过。证据：`go test -count=1 ./internal/tts/...` → `ok 1.488s`（PASS=65 / SKIP=1 / FAIL=0）。
- **ci.yml 验证（比读 YAML 强）**：python-yaml 递归结构 diff（HEAD↔工作树 5 条差异路径全在预期位置、jobs 集合不变、race job 逐步骤相等）+ 四份 run 块 `bash -n` 全过 + **stub `go` 端到端跑失败路径**：工作树口径把 `internal/tts` 交给了 `go test`（HEAD 版没有）且 `REGRESSION → STEP_EXIT=1`（失败仍红，不静默）。
- **余量**：本机 Win11 + go1.26.6 vs `windows-latest`（Windows Server）——最终证明仍是改后第一次 CI 跑。

**任务 2（P3）前端 flaky 分类器零自测（已收口）+ 堵掉一个真静默放行洞**
- `frontend/scripts/classify-vitest-failures.mjs`（80 → 243 行）重构为可测形态（`readFlakyList`/`readReport`/`classify`/`applyExpectFailure`/`quiet`/`selftest`/`main`），**8 组夹具 / 14 条断言**，覆盖：全绿 / 失败⊆清单（含复跑清单落盘）/ 含清单外失败（并断言报点只列清单外那条）/ 空清单+有失败 / 无失败事件但首跑非零（带 flag→UNATTRIBUTED，不带→OK）/ 报告缺失→2、不可解析→2、**可解析但无 `testResults`→PARSE-FAIL/2**。
- 新增 `--expect-failure`（与后端同义：首跑非零却归不出失败文件 → UNATTRIBUTED/exit 1，**不许静默放行**）；`ci.yml` 前端 job 新增 `Vitest flaky classifier selftest` step + 注释改成与事实一致 + 首跑非零路径加 `--expect-failure`。
- 证据：`--selftest` 14/14 exit 0；真报告 + 真清单的 CLI 端到端四条（REGRESSION/1、FLAKY-RETRY/0 + 复跑清单落盘、`{}`→PARSE-FAIL/2、无失败事件+flag→UNATTRIBUTED/1）；**负例自测**：把夹具③期望改反 → `FAIL ③ 报点只列清单外那条`、1/14、exit 1，还原后 14/14 且文件 SHA256 与改前**一字不差**（`2785800f…`）。
- **附带发现（真静默放行洞，已堵）**：原实现遇到**报告里没有 `testResults` 数组**（vitest 崩在配置/收集阶段、报告被截断、写出 `{}`）会落 `OK: 无失败` / exit 0 —— CI 只在首跑非零时才调分类器，等于把「套件启动崩了」**放行成绿**。现按 PARSE-FAIL/exit 2 红。**这条不在原 X1-12 记录里，是本批新发现**（详见 §5.1.1）。

**任务 3（P7）X1-13 遮蔽闸归因（已收口）**
- `scripts/gen_bindings/main.go`：`const shadowBaselineMax = 2` → **显式在册清单** `var shadowBaseline = []shadowPair{…}`（列名字 + 接收者 + 文件:行，作人读登记）；归因改 `shadowDiff` **按名字集合差**（`added`/`gone`/`moved`）；`shadowDriftReport{code, lines}` 让**打印与断言共用同一份事实**。
- 判据：实测有/在册无 → 新增 → exit 1 逐条列新增名；**在册有/实测无 → 「消失」也 exit 1**（提示「这不是新增／请同步在册清单」），避免清单悄悄过期；**位置漂移仅提示不拦门**（避免「代码上移几行就拦门」的脆性）。判闸仍在写生成物之前（默认路径同样先判）。
- 新增 `scripts/gen_bindings/shadow_baseline_test.go`（4 例）→ `go test ./scripts/gen_bindings/` ok。
- **负例自测（隔离夹具 `%TEMP%\shadow-neg`，未碰仓库 `internal/app` —— 那是别条线足迹）**：① 新增字典序最靠前的 `AardvarkModel` → 新实现 `exit 1` 且**只列** `AardvarkModel :148`；**同一夹具跑 HEAD 旧二进制 → 报的却是 `SetFeatureModelEnabled :147`（在册条目被当成新增）**——批次十二登记的「字典序切片可能指错条目」由此从推断变**实测**；② 移除在册项 → exit 1 只报「消失」，输出里**没有**「新增遮蔽」段；③ 还原 base → exit 0 且夹具 SHA256 与首次 base 完全一致（`64ac54f8…`）；④ 默认（写文件）路径 exit 1 且 `internal/app` 下**零 `bindings_*.go`**（无半截产物）。
- 证据（green）：`-shadow-diag` 打印在册 2 项 / 实测 2 项 + 明细，exit 0；`-names` 仍 744 行；`check-bindings-drift.ps1` → `OK@744`。
- **余量（有意收紧，可由主代理单点回退）**：「在册项消失」也判红——理由=显式清单必须同步；若倾向只告警，改 `reportShadowDrift` 那一处返回码即可。口径未变：本闸数「真正被丢掉的实现」（当前 2），与审计「约 281 份全仓同名对（含门面重复）」仍是两个口径。

**线 2 足迹**：仅 4 个文件（`ci.yml`、`frontend/scripts/classify-vitest-failures.mjs`、`scripts/gen_bindings/main.go`、新增 `scripts/gen_bindings/shadow_baseline_test.go`）；明确未改 `scripts/ci.ps1`（BOM 复核 EF BB BF 仍在）、`scripts/classify-go-failures.mjs`、`frontend/known-flaky.txt`、仓库内 `internal/app/**`。

### 线 3 · 前端收敛与可见性

**任务 1（P4）计划 rel 第 4/第 5 份副本并单源（已修）**
- `gaea/components/ScheduleApplyRollback.tsx`（:3 import、:25-28 删自写 const）与 `gaea/lib/mock/office/schedule.ts`（:13 import、删原 :16 const）改为 import 零依赖 `schedule/paths.ts`；`paths.test.ts` 由 3 处消费点断言**扩到 5 处**（新增「组件渲染口径」——mock 给 `target===SCHEDULE_DEFAULT_REL` 的记录，组件缺省路径必须命中并渲染按钮；「mock 走查态指针口径」）+ **并单源源码锁**（两个消费文件不得再自写 rel 字面量）。
- 模块环前提已否：`paths.ts` 仍零 import（用例 3 源码断言「模块内不出现任何 import」）。
- 证据：`vitest run src/schedule/paths.test.ts` → 3 passed；关联面 `ScheduleApplyRollback.test.tsx` 2 ✓ / `mock-office-schedule-projects` 9 ✓ / `store.sync` 14 ✓ / `changes` 15 ✓。**反向证据**：把 mock 的 import 改回漂移字面量 `进度计划/当前计划X.gsched.json` → 2 例 FAIL（值不等 + 源码锁命中），还原后 SHA256 一致、3 passed。
- 余量：①「目录前缀」那族仍是 4 处副本（`进度计划/${name}.gsched.json`，mock Create/Copy，属**命名约定**不是缺省 rel，未并）；② `bridge/office.ts:60`、`pages/SchedulePage.tsx:8` 注释里仍有字面量（非代码）；③ Go `DefaultRelPath` 的跨语言一致性仍靠字面量锁 + 后端校验兜底。

**任务 2（FE4-02）工具参数 → diff 两套平行实现（已修，审计描述成立、未证伪）**
- 新增**单一解析器** `frontend/src/gaea/lib/toolArgs.ts`（166 行，零 import）：`parseWriteArgs` → `ParsedWriteArgs{hunks[{index,old,new}], content?, degrade?, editsCount, lineRange?}`；`planDiff.buildChangeDiff` 与 `tools.diffsFor`/`diffStatFor` 只留**展示策略**，降级文案全部单源到 `toolArgs.ts`。
- **收益可定量**：写类工具（`edit_file`/`multi_edit`/`write_file`/`edit_lines`）的字段口径自此**集中在一处**；锁它的证据 = `toolArgs.test.ts` **24 条等价矩阵 + 6 条解析器契约 + 3 条加固**（33 例）。
- **行为等价优先**（方法论）：先在**未改动实现**的前提下把等价矩阵写成测试（重构前全绿）——期间发现两处是**我把期望写错**（`plusMinus("", x)` 实测 del=1，因 `diffLines` 把空串 split 出一个空行），按真实现状修正期望而非改码；重构后**同一测试文件一字未改**仍全绿。
- **唯一非等价变更（有意，已登记 + 有用例）**：args 是合法 JSON 但**非对象**（典型 `"null"`）时，旧两条实现都在读属性处 TypeError **炸穿变更面板**；现按空对象走「字段缺失」降级。数字/字符串/数组在旧实现下本就等价于空对象，只有 `null` 从崩溃变降级。
- **未复用 `changes.extractChangedPaths` 的理由（写进 ts 头注释）**：它的契约是「一次调用 → **全部**被改动路径」（`path`/`file_path`/`notebook_path`/`source`/`destination` + `paths`/`file_paths` 数组 + `edits[].path`），服务 `WRITE_TOOL_NAMES` 全量工具（含 `delete_symbol` 等无内容片段者）的**聚合计数**；解析器的契约是「内容级视图 → 唯一主目标 path + 可还原片段」——键集合、基数、工具集合三者都不同，硬并会把聚合语义塞进内容视图。
- 证据：重构后 `toolArgs` 33 passed；关联面 15 文件 **171 passed**；**两次反向证据**（`hunks.push`→`unshift` 改错 hunk 顺序 → 3 例 FAIL；`NOTE_WRITE_FILE` 文案尾部加字 → 2 例 FAIL；两次还原后 SHA256 均与改前一致）。
- 余量：`plusMinus("", x)` 计 del=1（两侧一致的既有口径，未改）；三处两侧历史分歧**刻意保留并逐条锁死**（CRLF 归一/原文 · multi_edit 空 old 丢弃/保留 · 「编辑 N」连号/原始下标 + edits 缺失→stat=null vs 空数组→0/0）。

**任务 3（GA4-11 前端侧）备份回滚「部分失败」可见性（已修）**
- **改前确切行为（前后对照）**：`DataPanel.tsx` 的 catch **只有一句** `message.error(err.message)`——**antd 瞬时 toast**（无 DOM 持久节点、无「部分失败」措辞、无任何用例覆盖）；这段 try/catch 早于批次十二就在，批次十二把后端改成「部分失败返回非 nil error」后，这条 catch 才第一次真正被部分失败路径命中 ⇒ 前端此前是「只有瞬时 toast + 零用例」，不是「已有持久呈现」。
- 改动：`DataPanel.tsx:41-72` 新增 `RollbackFailure` 态 + 与 `backup.go` 文案对齐的正则 `toRollbackFailure`；`:145-158` catch 落状态（保留 toast 作即时反馈）；`:161-177` **页面内持久 Alert**（`data-testid=settings-rollback-failure`、可关闭、无 modal）。测试 3 例（完整成功 / 部分失败点名项数+可重试说明+原文两通道 / 整体失败通用文案），`DataPanel.test.tsx` 7 passed。
- **反向证据**：删掉 catch 里的 `setRollbackFailure(...)`（=退回改前）→ 2 例 FAIL（部分失败 `expected 1 to be greater than or equal to 2`、整体失败找不到「回滚失败」），而「完整成功」仍 ✓（**证明红灯定向**）；还原后 SHA256 一致、7 passed。
- 余量：`done===false` 仍走 `message.success('没有可回滚的恢复前备份')`（非失败态，语义更像 info，另案）；失败后不重新 `load()`；计数解析依赖后端文案正则（换文案只退化为通用标题，有兜底用例）。

**线 3 的方法论产出（进在册）**：**「先固化现状、再重构」的等价矩阵**——重构前把两套实现的分叉**逐条测出来**（发现「期望写错」两处并纠正期望而非改码），重构后测试文件一字不改仍全绿；这是「行为等价」可证的正确姿势，比「重构完看看测试过不过」强一个量级。

### 线 4 · app 内 P1（三条）

**任务 1（AP4-07）文件索引入队去重三入口两套空间口径（已修，含审计因果链纠错）**
- 唯一入口 `internal/app/gaea_tasks.go:388 subFileIndexTaskIn` 实际名 `submitFileIndexTaskIn(space, session, label, reason)`：`HasActiveInSpace(space)` + `SubmitSpaceSession(space, session)` **同取一个 space**；`space` 空/非法 → work；返回 `(nil, nil)` = 去重命中。空间单点 `:411 fileIndexSpace()` = **调用方当前生效空间**（play 生效→play；mode off / 引擎未初始化→work）。`gaea_file_index.go:34/75` 两入口改走它（**错误文案逐字不变**），6 处 watch 调用点补 label。
- 单源实证：`grep` 全仓只剩 `gaea_tasks.go:394/:397` 两条调用。
- **收敛前的三入口口径差异清单（动手前逐处读码，见 §二 引述）**：manual/cron 判据=**全局跨空间** `HasActive`、落库 `Submit`（space 隐式缺省 work）、无 space/session 参数；watch 判据=`HasActiveInSpace(work)`、落库 `SubmitSpace(work)`、显式 work。差异=「判据空间两处全局 vs 一处 work」（play 在途对 watch 不可见；play 在途又会让 manual/cron 误拒 work 刷新）+「落库空间隐式 vs 显式」（当时同值但不同源）+ session 三处恒空。
- **审计因果链纠错（实测）**：审计引的 `novel_import_task.go:55` 提交的是 `KindOutlineReconstruct`；全仓 `KindFileIndex` 只有 3 个提交点且**今天没有任何 play 空间的 file_index 生产者** ⇒ 审计叙述的「play 任务不被看见 → 与 watch 并发抖动」**当前不可达**；真实可达缺陷=「判决空间三元不一致 + 一旦将来出现 play 生产者即跨空间不可见/互吞」的**类缺陷**。
- 证据：新增 `gaea_file_index_space_test.go` 三用例（play 在途时 work 入队不被吞 / 同空间必去重 / play 侧判据仍认得自己的在途 + mode off 回退 work + 三入口判据同源）；**改坏能红** M1（`fileIndexSpace` 写死 work）→ FAIL「work 在途不得吞掉 play 入队」、M2（判据换回全局 `HasActive`）→ 三用例全 FAIL。
- 余量：生效空间切换窗口内仍可能两个全量任务并发（per-space 去重是审计指定口径；彻底单例需 tasks 层 workspace-global 单例 kind，**属拍板**）；`applyIncrementalFileIndex`（watch 增量）不经队列，与全量任务仍可并发——另一条，未动。

**任务 2（AP8-08）板块白名单第二份 → 单源（已修，且**多收两份**）**
- 审计只列了 `intent_llm.go:56-57` 提示词；线 4 实测**同族漂移还有两处**：`internal/app/wx_agent.go:71`（工具 schema 参数说明）与 `:315`（失败回执）——同样含已删的 `code`、缺 `schedule/sin/knowledge`，且 `:67` 说明里的中文名写死「编程/轻语/造价库」。
- 单源实现：新增 `internal/app/board_manifests.go:23 boardIDsFromManifests` / `:35 boardLabelsFromManifests`（空清单→`""`）/ `:49 boardIDList` / `:54 boardLabelList`（均取 `a.GetBoardManifests()`）；`intent_llm.go:48 intentFallbackSystemPrompt(boards)` + `:77` 用 `boardIDList()`（`boards==""` 时**不写**「只能是：」约束、不 panic）；`wx_agent.go:66 wxAgentBoardNavigateTool` + `:95 (a *App) wxAgentToolSchemas()`（6 工具顺序不变、原 var 保留 5 静态工具，净 +40/-7）+ `:344` 回执动态。
- 证据：新增 `board_id_single_source_test.go` 三用例（**提示词 ID 集 == manifest 集双向断言**；schema 说明 + 回执两处同断言 + 展示名枚举 == manifest 且不含「编程」；清单为空降级不 panic 不编造）；**改坏能红** M3（提示词手写回旧清单）→ FAIL（缺 knowledge/sin/schedule、多出 code/home）、M4（wx 工具说明手写回旧清单）→ FAIL（同上 + 仍列举编程 + 缺展示名）。
- 余量：工具说明给模型看的中文枚举由固定串变为 **manifest 展示名**（有意收敛；模型仍可传中文名经别名表解析）；前端 `frontend/src/boards/manifests.ts` 是另一份静态清单（不在该线足迹，未动），**Go/TS 无跨语言断言**——登记为下一批候选。

**任务 3（GA3-13 接线）知识库迁移失败可见（已接）**
- 新增 `internal/app/gaea_knowledge_migration.go`：`:28 knowledgeMigrationStateFn`（读状态缝）、`:35 reportKnowledgeMigrationState`——**仅 Failed 时** `slog.Warn`（含 Reason）+ 经**既有** `gaea-event` notice（`gaeaNoticeSink` 缝 → 内核 emit，与 `gaeaBackgroundPanicNotice`/`reportCostReadError` 同路，**未新造通道**）；`app.go:545` 启动路径（charlib+stores 阶段）调用。**未加任何绑定方法**（绑定面零变更；面板走既有 notice 消费面）。
- 证据：新增 `gaea_knowledge_migration_test.go` 两用例（失败态 1 条 notice 含 Reason + `slog` level=WARN 含 Reason；success/not-ran 两态**零 notice 零 WARN**）；**改坏能红** M5（读状态后直接 return）→ FAIL「迁移失败应发 1 条 notice，实际 0 条」。
- 余量：启动即首次打开知识库（SQLite 打开 + 旧目录 glob，成本小且是「启动即可见」的唯一时机）；面板若要**常驻**提示条需新增读取型绑定——按纪律未加，**留拍板**。

**线 4 足迹**：全部在 `internal/app/**`（M：`gaea_file_index.go`/`gaea_tasks.go`/`gaea_tasks_test.go`/`board_manifests.go`/`intent_llm.go`/`wx_agent.go`/`wx_agent_test.go`/`app.go`；新增：`gaea_file_index_space_test.go`/`board_id_single_source_test.go`/`gaea_knowledge_migration.go`/`gaea_knowledge_migration_test.go`）；**新增 7 个符号全未导出**（无导出方法签名变化）；未动其它线/主代理文件。

---

---

## 二、跨线裁决与缝合（主代理收线清单）

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **线 1 反向驳回主代理的「闸宽外溢」建议** | 采纳线 1 判据：`ops` 的文件契约是「**只校验本次改动**，应用后由调用方统一 `Validate` + CPM fail-closed」；`validateDurationUnit` 在生效态块里判是因为 patch 的 level/isMilestone 会**制造**新的 cd 违规，而**既有** `Duration<0` 不是本次 patch 造成的非法态。外溢的坏处：与工期无关的补丁被拒、文案指向用户没碰的字段、**同一条 op 在不同计划上随机成败**。故 `patch` 只在本 op 触及 duration 时校验（`ops.go:204` 的 `if pt.Duration != nil` 门 + TS 同门），并用两侧探针 + 一条新 fixture 用例锁死边界。**主代理独立复现了 D5 反向证据**（把 upsert 闸改成只调用不拦 → `err=nil` 且 `{ID:C Duration:-5}` 落计划 → 还原 `ok 1.102s`，`TEMP-MUTATE` 残留 0）。 |
| 2 | **跨语言契约：fixture 是生成物** | `ops_golden.fixture.json` 的唯一权威源是 `internal/schedule/ops_golden_test.go` 的 `goldenCases()`（`-update-golden` 重生成），Go 与 TS **共读同一份**。本批 `numstat = 112/0`（**纯新增、零删除**，既有 65 条逐字节不变）——这是 golden fixture 该有的改法：只加不改，任何既有行为漂移都会在两侧同时暴露而不是被新期望值掩盖。 |
| 3 | **审计因果链纠错（AP4-07）** | 审计引 `novel_import_task.go:55` 说「play 提交索引任务」——实测该提交是 `KindOutlineReconstruct`，全仓 `KindFileIndex` 只有 3 个提交点且**当前无 play 生产者** ⇒ 「正在抖动」的叙述不成立；按「**埋着的地雷**」定性（判据三元不一致 + 将来一有 play 生产者即跨空间不可见/互吞），修法照审计口径落地。**文档必须留这条更正**，避免后续会话当真事故去救。 |
| 4 | **审计只列一处、实为两处（AP8-08）** | 除 `intent_llm.go` 提示词外，`wx_agent.go:71`（工具 schema 说明）与 `:315`（失败回执）同族漂移，`wx_agent.go:67` 的中文名还写死「编程/轻语/造价库」⇒ 由线 4 一并单源化（4 处清单 → `board_manifests.go` 的 manifest 派生 + `wxAgentToolSchemas()`）。 |
| 5 | **GA3-13 接线不加绑定** | 迁移失败经**既有** `gaea-event` notice（`gaeaNoticeSink` 缝，与 `gaeaBackgroundPanicNotice`/`reportCostReadError` 同路）+ `slog.Warn`，**绑定面零变更**；面板若要常驻提示条需新增读取型绑定 ⇒ 留拍板。 |
| 6 | **线 3 的「唯一非等价变更」** | `args` 是合法 JSON 但为 `null`（非对象）时：旧两条实现都 TypeError **炸穿变更面板** → 现按空对象走字段缺失降级。**有意变更，已登记 + 有用例**（数字/字符串/数组在旧实现下本就等价于空对象，只有 `null` 从崩溃变降级）。 |
| 7 | **线 2 的「消失也红」** | 遮蔽在册清单：新增遮蔽 → exit 1；**在册项消失 → 也 exit 1**（提示「这不是新增／请同步在册清单」），避免清单悄悄过期；位置漂移**仅提示不拦门**。理由与单点回退位置已写明（改 `reportShadowDrift` 一处返回码即可）。 |
| 8 | **线 3 未复用 `changes.extractChangedPaths`** | 采纳其理由：该函数契约是「一次调用 → **全部**被改动路径」（服务 `WRITE_TOOL_NAMES` 全量工具的**聚合计数**），与「内容级视图 → 唯一主目标 path + 可还原片段」的键集合/基数/工具集合三者都不同，硬并会把聚合语义塞进内容视图。 |
| 9 | **主代理预核 GA4-11 误判（自我更正）** | 我按 **Go 绑定名** grep 断定「前端零消费面」，被线 3 用实证推翻（前端写**驼峰别名** `app.DataBackupRollback()`，经 `mappings.ts:147` 运行时改写 Go 名）。教训已登记，并直接影响下一批 FE3-03 的实现口径。 |
| 10 | **守卫基线刷新** | `check-test-ctors`：线 4 新增 4 处测试药桩（两处纯函数测板块清单、两处 `&App{core:&core{}}` 测 notice）⇒ 人工评审 → `--write-baseline`（363 = 359 + 4），复跑「无新增」。其余守卫本批**无需刷**：`check-primitives --strict` 无新增（截断 37 / 原子写 8 / 前端 5 / heredoc 0）、`check-bindings-drift` OK@744（绑定面零变更）、`check-contract-drift` 无新增、`check-docs` OK。 |

---

## 三、本批结论与余量

### 3.1 交付与验收

| 项 | 结果 |
|---|---|
| **真缺口** | `upsert_task` 负工期闸（批次十二 D5）——判据单源 `validateDuration`，Go/TS 两侧同判据同文案，golden fixture 跨语言锁定（含「不误伤」用例） |
| **守卫机器化** | `ci.yml` 删 `internal/tts` 过期排除（含 Windows-only 字段实证）· 前端 flaky 分类器 8 组/14 断言 + **堵掉「无 `testResults` 被放行成绿」** · 遮蔽闸改显式清单 + 名字集合差（新增/消失分开报、位置漂移仅提示） |
| **前端收敛** | rel 第 4/5 份副本并单源（5 消费点断言 + 源码锁）· 写类工具字段口径收敛到 `toolArgs.ts` 一处（24 等价矩阵 + 6 契约 + 3 加固）· 备份回滚部分失败**页面内持久提示**（改前只有 antd 瞬时 toast） |
| **app P1** | 索引入队去重**唯一入口**（判据与落库同取一个 space、跨空间互不吞并）· 板块白名单**四处清单**统一由 manifest 派生（含 `wx_agent` 两处审计未列）· 知识库迁移失败经既有 notice 可视（不加绑定） |
| 静态门禁 | `go build ./...`=0、`go vet ./internal/app/...`=0、`gofmt -l` 空 |
| Go 测试 | 主代理独立复跑：`internal/schedule` 1.026s、`internal/app` **106.7s** + board 0.500s、`internal/tts` 1.488s（线 2 实测）——均 ok |
| 前端 | 主代理独立复跑：`vitest` 6 文件 **86 passed**（paths/toolArgs/planDiff/tools/DataPanel/ScheduleApplyRollback）；线 3 另跑关联面 15 文件 171 passed；`tsc -b`=0、`eslint`=0 problem |
| 守卫 | 五份：`primitives --strict` OK · `test-ctors` 无新增（基线已刷至 363）· `bindings-drift` OK@744 · `contract-drift` 无新增 · `check-docs` OK（校验和 447 / 发布说明 636） |
| **全量本地快闸** | **`scripts/ci.ps1` → `CI OK` / `CI_SCRIPT_EXIT=0`**（`version-drift` → `clean-tmp` → `go build` → `go vet` → **golangci-lint v2.14.0 = 0 issues** → `go test ./... -count=1` → 前端 `lint` → `build` → **vitest 425 文件 / 3702 例全绿** → E 系列回归守卫 → `check-docs` 卫生四查）。主代理另起一次独立全量 vitest 复核：`Test Files 425 passed (425) / Tests 3702 passed (3702) / vitest_exit=0`（批次十二为 424 / 3660）。 |

### 3.2 本批量化收益

| 项 | 变化 |
|---|---|
| 工期校验段 | 负工期判据 **3 份 → 1**（`validateDuration`），Go/TS 各一份镜像且 fixture 逐字节共读 |
| 遮蔽闸报点 | 「字典序切片猜新增」→ **名字集合差**（新增/消失/漂移三态分流） |
| 前端分类器 | 零自测 → **8 组 / 14 断言**，并堵掉 1 个静默放行洞 |
| CI 覆盖 | `internal/tts` **整包 66 例**从「白丢」回到门禁 |
| 计划 rel 常量（前端） | 4 份 → **1 份**（另 4 处「目录前缀」属命名约定，未并） |
| 写类工具字段口径 | 3 处（planDiff / tools / 各自降级文案）→ **1 处**（`toolArgs.ts`） |
| 板块 ID 清单 | **4 处硬编码 → 1 处 manifest 派生**（审计只列 1 处，实为 4 处） |
| 文件索引入队 | 3 套判据/落库写法 → **1 个入口**（判据与落库同 space） |
| 知识库迁移失败 | 「只有 API 层状态」→ **启动即 notice + WARN 可见**（零新增绑定） |
| 测试裸构造（守卫口径） | 359 → **363**（+4 测试药桩，已评审入基线） |

### 3.3 余量（如实登记，未硬凑）

1. **线 1**：patch 通道负工期文案仍不带 id 前缀（fixture 冻结，与 upsert/Validate 并存的登记项）；`ApplyOps` 单独调用不保证整计划合法（既有契约；要「出口即合法」需单开一刀在尾部挂 `Validate`）。
2. **线 2**：`internal/tts` 纳入 CI 的**最终证明仍是改后第一次 CI 跑**（本机 Win11+go1.26.6 vs `windows-latest` Windows Server）；`--expect-failure` 让 CI 更严（有意，方向安全）。
3. **线 3**：rel「目录前缀」族仍 4 处（命名约定）；Go `DefaultRelPath` 无跨语言编译期锁（靠字面量锁 + 后端校验）；`plusMinus("", x)` 计 del=1（两侧一致的既有口径）；三处两侧历史分歧**刻意保留并逐条锁死**。
4. **线 4**：生效空间切换窗口内仍可能两个全量索引任务并发（彻底单例需 tasks 层 workspace-global 单例 kind ⇒ **拍板**）；`applyIncrementalFileIndex` 不经队列；`frontend/src/boards/manifests.ts` 与 Go manifest **无跨语言断言**；知识库面板常驻提示条需新增读取型绑定（留拍板）。
5. **真机/环境欠账**（承接）：`-race` 只在 Actions；FE6-02 真机拔麦走查仍未做。

---

## 四、留池与待拍板

（收线后填写本批余量；下面「下一批候选短名单」为批次十三开工时预排，收线时按实测校正）

### 4.1 下一批候选短名单（审计 P1 中仍未触碰，按价值分四簇）

> 口径：来自 `machine/findings_all.json` 的 P1 全集减去批次一~十三已处理项；fid 与行号为 2026-10-02 审计值，**动手前必须按当前工作树实测**。

**簇 A · 安全 / 边界（建议优先，用户可感风险）**

> **主代理开工前预核（2026-10-02）**：
> ① `GA4-06` **成立且性质明确**——`internal/gaea/hook/hook.go:304` 定义 `Sandbox *sandbox.Spec`，`:387` 判 `in.Sandbox != nil && in.Sandbox.Mode == "enforce"` 才走 `sandbox.Command(...)`；但全仓**无任何生产路径给 hook 的 `Sandbox` 赋值**（grep 只命中 `gaea_ui_extra.go:116` 的 UI 视图、`config.go:758` 的 `SandboxConfig` 默认值、walkthrough 局部变量）⇒ **hook 执行永不沙箱**，而注释与配置默认（`Sandbox.Bash = "enforce"`）都宣称有。修法二选一：**接上**（hook 调用点传 `cfg.Sandbox`，与默认意图一致）或**删字段+改注释**（不让代码撒谎）；建议前者。与批次十二的 FE3-06 同族（写着有防护、实际没生效）。
> ② `AP2-04` **只剩一半**（审计原文需按现场收窄）：**模型/后端那一半已被批次十二顺带修掉**——`novel_bookcover.go:115` 现在走 `effectiveSceneIllustrationModel(a.cfg)`（不再是写死的 `coverImageModel`），:169 台账记生效后端/模型。**仍未修的是护栏旁路**：`applyImageSafeMode` 全仓只有 3 个注入点（`gaea_tools.go:89`、`image_handler.go:313`、`image_handler.go:548`），书封走 `clientRef().GenerateImage`（`novel_bookcover.go:122`）**不在其中** ⇒ play 空间的「小说」书封提示词不受 `image_safe_mode` 约束（同板块章节插图与原罪插图都受）。修法=改走 `mediaState.generateImageInternal`（或至少在该处补 `applyImageSafeMode` 并写明理由）。

> ③ `AP5-09` **成立，且是「重复实现 + 漏点」**（审计只说了重复）：同一语义现有 5+ 种手写判据——`gaea_listdir.go:61`、`gaea_pdf.go:50`、`gaea_preview.go:84`、`gaea_ui_extra.go:595/:640` 判 `..` 前缀/片段；`characterlib/portrait.go:210` 判 `/\` 与 `..`；`filewatch.go:218` 判 `Rel` 前缀；`scene.go:229` 判 sceneID；`image_domain.go:134/:302` 走 `imagePathWithinAny`（唯一「落在允许根内」口径）。**漏点**：`internal/app/gaea_lint.go:20-23` 零校验——`path = filepath.Join(gaeaCwd(), rel)` 后直接 `os.Stat/ReadFile`，`rel="../../x"` 会被 `Join` 清洗成工作区外路径。批次十四落法=叶子包落**唯一** `ResolveRelWithin(root, rel) (string, error)`（口径=Clean 后必须在 root 内），全部站点改调 + **参数化矩阵用例**（每个入口都拒绝 `../` 与绝对路径），而不是逐个功能各写一遍。

| fid | 问题 | 位置（审计当时） |
|---|---|---|
| AP2-04 | 书封生图**绕过图像域与 play 安全护栏**（模型口径已于批次十二修） | `internal/app/novel_bookcover.go:108-119` |
| AP5-09 | 路径归一 **15 处手写**，穿越防护只在 1 处生效 | `internal/app/gaea_lint.go:20-23` 等 |
| GA4-06 | hook 的 `Sandbox` 字段**全仓零填充**——命令实际不沙箱 | `internal/gaea/hook/hook.go:300-304` |
| IN4-05 | 22 种桌面操作巨型 switch，**零权限与路径校验** | `internal/whisper/desktop_executor.go:40-134` |
| IN4-08 | `normalizePath` 注释称禁 `..` 逃逸，**实现无校验** | `internal/whisper/desktop_executor.go` |

**簇 B · 真 bug / 竞态**

| fid | 问题 | 位置 |
|---|---|---|
| FE6-09 | 打字机流两份，**其中一份没有取消守卫** | `frontend/src/components/ChatPanel.tsx:113-122` |
| FE7-11 | **模块级可变游标在 set 更新器内被改写** | `frontend/src/schedule/store.ts` |
| IN2-13 | `ConsistencyReport` 同名两套定义、**JSON 字段不兼容** | `internal/graph/consistency.go:19-35` |
| GA1-09 | retry 循环里的 run 包装分支**永不可达**（死路径掩盖真分支） | `internal/gaea/agent/task.go:726-739` |
| AP1-07 / AP1-08 | 取消残稿三态落盘三份重复 / 重写写回三副本（同族漂移风险） | `internal/app/create_chapter_handler.go:455-470` / `novel_rewrite_handler.go:328-356` |

**簇 C · 口径单源（consistency，收敛型）**

| fid | 问题 |
|---|---|
| IN1-09 | AI 味判据三套严重度（blocker/S1/自定义） |
| IN1-05 | 两套文风互不相识：`Profile` 与 `Fingerprint` |
| GA3-07 | 知识库检索双口径合分，权重与阈值硬编码 |
| IN3-02 | 日历语义 Go 四处 + 前端 TS 一份镜像 |
| FE4-04 | 绑定方法名三份手工清单互锁 |
| GA6-01 / GA6-04 / GA6-06 | 成本条目 UPSERT 三份 / 检索三路 / 包级 BM25 缓存隐式契约 |

**簇 D · 死代码与僵尸资产（低风险清理）**

| fid | 问题 |
|---|---|
| IN4-04 | 桌面能力 3 条路由，2 条 0 生产调用 |
| GA4-09 | `SkillLayer` 的 `Learner` 桩与版本晋升全是空壳 |
| GA2-03 | `CompactDescriptor` 无条件生效，全量 Schema 成死资产且已漂移 |
| FE3-03 | 漂移检查只认「名字被认领」，死绑定无法发现 |
| X1-11 | 前端 mock 是第三份手写契约，已实际分叉咬人 |

### 4.2 待拍板（本批未动，仍需用户裁决）

- **AP4-01** 三套并行编排合并（DAG / 收件箱 / `gaea/tasks`）：(a) 抽共用原语（M）/ (b) 统一内核（L/XL）/ (c) 维持现状。
- **X1-03** 五重 `*core` 嵌入裸构造 panic：(a) 生成器注入 `assertAssembled()` + 227 处测试桩迁移（L）/ (b) nil-safe 访问器（M）/ (c) 维持守卫。
- **`estimateTokens` 四份口径**（P5，批次十一/十二两次入池）：口径真不同，合一会改变预算行为。

---

## 五、审计原文纠错与遗留登记

### 5.0 主代理开工前预核（收线时用线报告对账）

| 项 | 预核结论（主代理实跑 grep/read） | 状态 |
|---|---|---|
| AP8-08 板块白名单漂移 | **完全成立**：`internal/app/intent_llm.go:56` 硬编码 `home chat novel imagegen gaea cost code`——含**已删除的 `code`**，缺 `schedule/sin/knowledge/modelcenter/characterlib/settings/weixin/memoryhub`。**线 4 逐处读码又发现同族两处**（`wx_agent.go:71` 工具 schema 说明、`:315` 失败回执，且 `:67` 中文名写死「编程/轻语/造价库」）⇒ 实为 **4 处清单** | **已落地**（统一由 `board_manifests.go` 的 manifest 派生；M3/M4 两条改坏能红） |
| AP4-07 索引去重口径 | **三入口差异成立**（`gaea_file_index.go:35/:80` 全局 `HasActive` + `Submit` 缺省空间；`gaea_tasks.go:376/:379` `HasActiveInSpace(work)` + `SubmitSpace(work)`），但**审计的因果链引错一处**：它引的 `novel_import_task.go:55` 提交的是 `KindOutlineReconstruct`（**不是** `KindFileIndex`）⇒ 「play 的 file_index 任务从哪来」不许照抄审计。**线 4 已实测定案（2026-10-02）**：全仓 `KindFileIndex` 只有 **3 个提交点**（manual `gaea_file_index.go:38` / cron `:83` / watch `gaea_tasks.go:379`），**今天不存在任何 play 空间的 file_index 生产者** ⇒ 审计叙述的「play 任务不被看见 → 与 watch 并发抖动」**当前不可达**；真实可达缺陷是「同一件事判据空间三元不一致 + 一旦将来出现 play 生产者即跨空间不可见/互吞」的**类缺陷**。修法=三入口统一取 `fileIndexSpace()`（当前生效空间，mode off→work）+ `submitFileIndexTaskIn` 唯一入口 + per-space 去重 + 跨空间互不吞并 | **已落地**（线 4，含 M1/M2 两条改坏能红） |
| FE4-02 diff 两套平行实现 | **成立**：`frontend/src/gaea/lib/planDiff.ts:44` `buildChangeDiff` 与 `frontend/src/gaea/lib/tools.ts:87/119` `diffsFor`/`diffStatFor` 各自解析 `edit_file/multi_edit/write_file/edit_lines`（含各自降级文案）；线 3 补充实测：`diffsFor` 此前**零测试**、`diffStatFor` 只有 3 条 | **已落地**（新增零依赖 `toolArgs.ts` 单一解析器） |
| GA4-11 备份回滚「前端文案」 | ~~预核判定：前端无消费面~~ **〔主代理误判，已被线 3 实证推翻〕** 我按 **Go 绑定名** `GaeaDataBackupRollback` grep，必然零命中——前端写的是**驼峰别名** `app.DataBackupRollback()`，经 `bridge/mappings.ts:147` 在运行时改写为 Go 名（`proxy.ts:158` `gaeaToGaea[prop] ?? prop`）。真实消费面：`frontend/src/components/settings/DataPanel.tsx:115`（`handleRollback` 里 `await app.DataBackupRollback()`，由 :149 回滚按钮触发，:118 catch 只丢 antd `message` 瞬时 toast）；面板活代码（`pages/SettingsPage.tsx:20/:118`）；另有 `bridge/office.ts:116` 声明、`mock/office/methods_office.ts:908` mock、`spaceBindings.ts:594` 分面 | **线 3 按原要求落地**（补持久提示条 + 两态用例 + 改坏能红）；**教训**：绑定名 grep 必须「前端驼峰名 + mappings 反查」，只按 Go 名 grep 会**系统性漏报**（本次即误判为零消费面）——该教训直接影响批次十四候选 **FE3-03 死绑定检测**的实现口径 |

### 5.1 收线后的原文纠错与遗留

#### 5.1.1 线 2 阶段报告带出的两条「超出任务书」发现（阶段结论，待正式报告复核）

| # | 发现 | 证据（线 2 阶段实测） | 为什么重要 |
|---|---|---|---|
| 1 | **前端分类器存在真实静默放行洞**：报告里**没有 `testResults` 字段**时（vitest 在**收集阶段**就崩了），旧实现落 **OK / exit 0** ——「套件根本没跑起来」被当成「全绿」 | `frontend/scripts/classify-vitest-failures.mjs` 重构后该情形改判 **PARSE-FAIL / exit 2**，并新增 `--expect-failure`（UNATTRIBUTED 兜底） | 这正是 P3「判据不可信」家族的**更严重形态**：不只是「注释自称有自测」，而是「崩溃可被读成绿」。**CI 的绿可能来自一次没跑起来的 vitest** |
| 2 | **遮蔽闸误报原 bug 被实证复现**：同一夹具下，**HEAD 旧二进制**把在册条目 `SetFeatureModelEnabled :147` 报成「新增」 | 线 2 在 `%TEMP%\shadow-neg` 重建同形 `internal/app` 夹具：新增字典序最靠前的 `AardvarkModel` → **新**二进制 exit 1 且只列 `AardvarkModel :148`；**旧**二进制报的却是 `SetFeatureModelEnabled :147`（在册条目） | 批次十二我登记的那条余量（「字典序切片在新增名靠前时明细可能指错条目」）在此**从推断变成实测**；照旧二进制提示去「处置新增」会把基线写坏——这就是 P7 要修的真风险 |

#### 5.1.2 其余原文纠错与遗留

| # | 审计/前批原文 | 本批实测 | 处置 |
|---|---|---|---|
| 1 | AP4-07「play 的 file_index 任务不被看见 → 与 watch 并发抖动」 | `novel_import_task.go:55` 提交的是 `KindOutlineReconstruct`；`KindFileIndex` 全仓仅 3 个提交点且**当前无 play 生产者** ⇒ 「正在抖动」不成立 | 按**类缺陷**（判据三元不一致 + 潜在跨空间互吞）定性并修；§二 第 3 条留痕 |
| 2 | AP8-08 只列 `intent_llm.go` 一处白名单 | 实为 **4 处**（`intent_llm.go` 提示词 + `wx_agent.go:71/:315` + 中文名枚举） | 一并单源化；§二 第 4 条 |
| 3 | 批次十一余量 5「备份回滚部分失败的前端文案需确认」 | 前端**有**消费面（驼峰别名 `app.DataBackupRollback()` → `mappings.ts:147` 改写 Go 名）；改前只有瞬时 toast | 主代理误判已更正（§5.0）；线 3 补持久 Alert |
| 4 | X1-12 只记「CI 后端整批重试」（前批已修） | 本批新发现：前端分类器在「报告无 `testResults`」时把**崩溃读成绿**（OK/exit 0） | 已堵为 PARSE-FAIL/exit 2（§5.1.1 第 1 条） |
| 5 | X1-13「条数基线 + 字典序切片」 | 旧二进制在同一夹具下把**在册条目**报成新增（实测复现） | 已改名字集合差（§5.1.1 第 2 条） |
| 6 | `ci.yml` 注释「`internal/tts` 的 `TestEdgeTTS` 依赖外网」 | 该测试全仓不存在**且**真原因是平台（`sapi.go` 用 Windows-only `HideWindow`） | 双重不实已纠正；整包纳入 windows backend job（§一 线 2） |

**下一批建议（承接 §4.1 四簇）**：先打**簇 A 安全**（GA4-06 hook 沙箱零填充 · AP2-04 书封护栏旁路 · AP5-09 路径穿越「重复+漏点」· IN4-05/IN4-08 桌面操作零校验），再按簇 B/C/D 推进；FE3-03 死绑定检测必须按**前端驼峰名 + `mappings.ts` 反查**实现（本批误报教训）。
