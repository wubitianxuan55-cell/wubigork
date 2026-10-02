# 全仓审计第 14 批 · 簇 A 安全（4 条并行线）· 2026-10-02

> 接续 [round-15（批次十三）](round-15-p1-batch13.md) §4.1 的「簇 A · 安全 / 边界」建议簇。
> 本批**不抬版本**、不动 CHANGELOG（发版仪式归用户）；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `a08e9129`（批次十三）；工作树干净（`git status --short` 空）。
> **文档定位**：本文中每一条结论都带 `file:line` 与「谁实测」来源标注（主代理实测 / 子代理实测），
> 与本仓既有纪律一致——**审计原文必须能被现场复核，行号错=整条作废**。

---

## 〇、主代理开工前预核（2026-10-02 晚，读码实测）

> 口径同上批：审计 fid 与行号是 2026-10-02 审计当时的**移动靶**，动手前一律按当前工作树复核。
> 下表是主代理**亲自 read/grep** 的结论，不是转述子代理。

| # | 审计项 | 主代理预核结论 | 关键证据（当前工作树） |
|---|---|---|---|
| 1 | **AP2-04** 书封生图绕过图像域与 play 安全护栏 | **半成立/需收窄**：模型与台账口径那一半**已被批次十二顺带修掉**（`effectiveSceneIllustrationModel`）；真实缺口 = **护栏旁路**（`applyImageSafeMode` 的三个生产注入点都不覆盖书封） | `internal/app/novel_bookcover.go:115`（模型已是生效值）、`:122`（`a.clientRef().GenerateImage` 直发）；护栏注入点仅 `gaea_tools.go:89`、`image_handler.go:313`、`image_handler.go:548`（定义 `play_guardrails.go:136`） |
| 2 | **AP5-09** 路径归一 15 处手写、穿越防护只在 1 处生效 | **成立，且数字要改**：`gaea_lint.go` 零校验（`..` 可逃出工作区）；**主代理独立清点 = 真判据站点 ≥17 处（审计记 15），跨 ≥8 个包**（`internal/app` 9 / `characterlib` / `scene` / `gaea/filewatch` / `gaea/backup` / `gaea/control` / `gaea/pins` / `gaea/tool/builtin` / `schedule`）⇒ 收口必须先做「复用/上提/保留」逐站点裁决，否则会变成第 6 份重复 | `internal/app/gaea_lint.go:20-23`（`filepath.Join(gaeaCwd(), rel)` 后直接 `os.Stat`/`ReadFile`）；同族判据：`internal/gaea/tool/builtin/confine.go:102` `within`、`internal/app/image_domain.go:166` `imagePathWithinAny`、`internal/app/gaea_ui_extra.go:699/:725` `withinReadRoots`/`withinWriteRoots`、`internal/gaea/wspath`（当前**只有** `skipdirs.go`）；独立清点命令与 19 行站点表见 §0.1 |
| 3 | **GA4-06** hook 的 `Sandbox` 全仓零填充 | **成立且性质明确**：字段注释与配置默认（`Sandbox.Bash = "enforce"`）都宣称有沙箱，全仓**无任何生产路径给它赋值** ⇒ hook 命令永不沙箱；另发现 `DefaultSpawner` **丢弃** `sandbox.Command` 的「是否真被包裹」bool（静默失败同族） | `internal/gaea/hook/hook.go:300-304`（字段+注释）、`:344`（构造 `SpawnInput` 不填）、`:387-389`（闸 + `argv, _ :=`）；`hook/runner.go:26`（`NewRunner` 唯一构造器）；唯一生产装配点 `internal/gaea/boot/boot.go:275`（同类 spec 先例见 `:221`，诚实降级先例见 `:222-226`）；平台事实 `internal/gaea/sandbox/seatbelt_windows.go:9-25` |
| 4 | **IN4-05 / IN4-08** 桌面操作「零权限与路径校验」 | **按现场必须收窄为「埋着的地雷」**：①4 层安全管道**存在**（设置级拦截 / 关闭黑名单 / 路径策略 / 确认+跳过确认）且每步都留痕；②`ExecuteUseComputer` 生产唯一调用点 `agent_loop_runner.go:254` 被 `:245 if r.Router != nil` 包着，而 **`RouterContext` 全仓只在测试里构造**、`AgentLoopRunner.Router` 生产零赋值 ⇒ 整条执行链今天**只经测试触达**；③同一处发现**今天就可达的真静默失败**：`Router == nil` 时 `use_computer` 分支不追加任何工具结果（无 else、无回执、无告警） | `internal/whisper/desktop_router.go:51-156`（4 层）；`internal/whisper/desktop_executor.go:40-134`（22 action switch）；`internal/whisper/agent_loop_runner.go:14-21`/`:34-39`/`:245-259`；`RouterContext{` 全仓仅测试命中 |
| 5 | **FE6-09** 打字机流缺取消守卫 | **成立**（至少 `ChatPanel` 这一份）：rAF 循环体无 unmounted/代际判断，循环尾还会回写 `onMessagesChange([...withAi])` 覆盖外部状态 | `frontend/src/components/ChatPanel.tsx:104-127`（循环 `:113-122`、回写 `:123-126`）；「另一份打字机在哪、守卫长什么样」由线 4 实测 |
| 6 | **FE7-11** 模块级可变游标在 `set` 更新器内改写 | **待线 4 定位**（预核只确认了现场特征：该文件有大量 `set((s) => { pushHistory(...); … })` 形态与「合并会话游标」注释） | `frontend/src/schedule/store.ts:308-481`（`set` 更新器群）、`:187`/`:264`/`:621`（游标注释） |

**由预核直接得出的本批收口原则**：
1. **线间足迹互斥按「Go 包」划**（本仓在册坑：同包/跨包并发写者会让 `go build` 在途变红，构建期任何测试都不具判据价值）。
   本批分线：线 1 = 路径/护栏（`internal/app` + `characterlib` + `scene` + `filewatch` + 新叶子包）；线 2 = `internal/gaea/hook` + `internal/gaea/boot`；线 3 = `internal/whisper`；线 4 = 前端两文件。
2. **审计原文不实/过时的一律纠错**（预核已产出 2 条：AP2-04 的半边已被前批修掉、IN4-05/08 的「零校验」与「实现无校验」都不成立）——纠错不是免责，是把修法对准真缺口。
3. **每线必须交「改坏能红」反向证据**；**审计没说的真缺陷照修**（如第 3 条的丢弃 bool、第 4 条的静默 no-op）——本仓既有口径：先说现场，再说审计。

---

### 〇.1 AP5-09 独立清点（主代理扫，用于收线时对拍子代理报告）

命令：`grep '"\.\."|HasPrefix\(.*\.\.[\\/]|Contains\(.*\.\.'` 扫 `internal/**`（63 命中）→ 剔测试/注释/文案后：

| 站点 | 形态 | 语义类别（主代理初判） |
|---|---|---|
| `internal/app/gaea_lint.go:20-23` | **零校验** | 漏点（必修） |
| `internal/app/gaea_listdir.go:61` | cleanRel 前缀/片段 | 穿越片段 |
| `internal/app/gaea_pdf.go:50` | clean 前缀 | 穿越片段 |
| `internal/app/gaea_preview.go:84` | cleanRel 前缀/片段 | 穿越片段 |
| `internal/app/gaea_ui_extra.go:595` / `:640` | IsAbs + clean + 前缀/片段 | 穿越片段（两处逐字相似=重复） |
| `internal/app/image_domain.go:186`（`:134`/`:302` 消费） | rel 前缀 | **落在允许根内**（唯一） |
| `internal/app/characterlib_handler.go:262` | rel==".." / `../` 前缀 | 穿越片段 |
| `internal/app/gaea_deliverable_zip.go:38` | clean `../` 前缀 | 穿越片段 |
| `internal/app/gaea_schedule_file.go:342` | 逐段 `seg==".."` | 段级 |
| `internal/app/intent_router.go:336` | rel 前缀 | 穿越片段 |
| `internal/characterlib/portrait.go:210` | 分隔符 + `..` 片段 | 文件名/ID 净化 |
| `internal/scene/scene.go:229` | `!Contains(sceneID,"..")` | **sceneID 合法性**（≠路径包含） |
| `internal/gaea/filewatch/filewatch.go:218` | Rel 前缀 | 穿越片段 |
| `internal/gaea/backup/backup.go:350` | clean 前缀 / 含 `..\` | 归档条目名 |
| `internal/gaea/control/attachments.go:164` / `:181` | `HasPrefix(clean, root+sep)` | **落在根内**（语义最近） |
| `internal/gaea/pins/pins.go:119` | `../`、`/../` | 引用/URL 口径 |
| `internal/gaea/tool/builtin/confine.go:102/:107` | `within(root,path)` | **落在根内**（既有叶子判据） |
| `internal/schedule/index.go:321` | 逐段 `seg==".."` | 段级 |

**口径要求（写进线 1 报告，收线逐条核）**：`站点 → 语义类别 → 处置（改调单源 / 上提 / 保留） → 理由 → 用例覆盖`。
语义不同的（sceneID、pins、backup、schedule 段级）**宁可保留并写明理由**，不许硬并（先例：批次十三线 3 拒绝硬并两套 diff 实现）。

---

## 一、逐线落地

> 本节按线分节。**每条结论都标来源**（`主代理复核` = 我亲自读 diff / 跑探针；`线 N 实测` = 子代理报告且我按证据复核）。
> 子代理报告全文不进本文（避免自述当实测），只保留可复核的行号与命令。

### 1.1 线 1 · 路径归一与穿越单源（AP5-09）+ 书封护栏（AP2-04）

**结论：两条都成立，且 AP5-09 比审计原文更严重。** 单源落在**已有叶子包** `internal/gaea/wspath`（未造第二份），**30 个调用点改调**（24×`ResolveRelWithin` + 6×`Within`）。

| 项 | 落地形态 |
|---|---|
| 原语 | `wspath/resolve.go:47` `Within(root,target) bool`（Rel 口径唯一包含性判据）；`:86` `ResolveRelWithin(root,rel) (string,error)`（Clean+Abs+包含性；空串/`.`→root；绝对→`ErrPathAbsolute`；`C:foo`→`ErrPathVolume`；`..`逃逸→`ErrPathEscape`） |
| 用例 | `wspath/resolve_test.go`（合法 8 形态 + 非法逐条错因 + 大小写/跨盘符/无卷根/空 root fail-closed）；`internal/app/wspath_guard_matrix_test.go`（**入口 × 恶意输入**矩阵：19 入口 + 绝对/卷相对分列 + 合法输入回归锁 + 诱饵文件防「碰巧报不存在」） |
| 书封（AP2-04） | `novel_bookcover.go:118` `Prompt: applyImageSafeMode(b.String(), playGuardrails().ImageSafeMode)` —— 口径与 `image_handler.go:313` 逐字一致（安全段只进请求、台账仍记原始提示词）；未改走 `generateImageInternal` 的理由写在代码注释里（统一入口会接管 size 策略/落盘目录/并发槽 ⇒ 书封 3:4 契约与 exports 落点会回归） |

**站点 → 旧口径 → 新口径 → 是否行为变化（线 1 报告 + 主代理逐站读 diff 复核）**：

| 站点 | 旧口径 | 新口径 | 变化 |
|---|---|---|---|
| `gaea_lint.go:30` | **零校验** `Join(cwd,rel)` | `ResolveRelWithin` | **收窄（真漏点）** |
| `gaea_preview.go:90` | 有前缀判据但**返回逃逸路径**（赌 Stat 失败） | 越界→空路径 | **收窄（真穿越）** |
| `gaea_docx_edit.go:59/94`、`gaea_pptx_edit.go:34`、`gaea_xlsx_edit.go`（6 处）、`gaea_crosslink.go:60/97`、`gaea_xlsx_chart.go:48` | **零校验** `Join`（crosslink 输出还含 `MkdirAll`） | `ResolveRelWithin` | **收窄（写侧逃逸族）** |
| `gaea_verify.go:42` | **零校验** `Join`（Journal 是不可信磁盘面） | `ResolveRelWithin` + 签名改 `(string,error)` | **收窄**（不再越界复核/回滚写盘） |
| `gaea_schedule_file.go:82`（索引指针）/`:123` | **零校验** `Join` / 段级门后 `Join` | `ResolveRelWithin` | **收窄 / 冗余防线** |
| `gaea_listdir.go:63`、`gaea_pdf.go:50`、`gaea_ui_extra.go:601/651`、`image_domain.go:183`、`gaea_deliverable_zip.go:51`、`gaea_tasks.go:558`、`characterlib_handler.go:264`、`intent_router.go:337` | 各自手写 Clean/前缀/Rel 判据（含**两份逐字重复**） | `ResolveRelWithin`/`Within` | **等价**（`image_domain` 唯一边界 `path==root` 由拒改允，产物恒为文件故不可达） |
| `gaea_ui_extra.go:729/746`（`withinReadRoots`/`withinWriteRoots`）、`filewatch.go:222`、`builtin/confine.go:105` | 手写 `HasPrefix(abs,root+sep)`；`HasPrefix(rel,"..")`；包内自带 Rel | `wspath.Within`（builtin 处**上提**） | **对齐文件系统**：大小写不敏感（原 `c:\ws\x` 被误拒）、`root==abs` 计入、相对 root 按 cwd 解析；**修假阴性**：合法文件「..foo」不再被静默丢弃 |

**既有判据 → 复用/上提/保留（主代理要求表，线 1 逐条给理由）**：**上提** `builtin/confine.go:102 within`（唯一逐条同义者，叶子包防依赖倒挂）；**复用** `imagePathWithinAny` / `withinReadRoots` / `withinWriteRoots` / `characterlib_handler` / `filewatch`；**保留（语义不同，不硬并）** `control/attachments.go:164/:181`（返回相对名+固定子根+Lstat 逐段拒符号链接）、`pins.go:119`（丢弃式返回空串、更严）、`backup.go:350`（zip 条目名口径）、`checkScheduleRel`/`validateScheduleRel`（段级更严，硬并会**放宽**）、`scene.go:218`（ID 白名单，其 `..` 判据实为死代码）、`portrait.go:208`（段清洗→SHA256 回退）。

**反向证据（线 1 跑，报告给红/绿原文；主代理已按同样纪律要求所有线）**：
1. `resolve.go` 的包含性检查改 `if false` → `TestResolveRelWithin` 红 **7 子用例**；`TestWspathGuardMatrix` 红 **68 条断言**（原文含 `GaeaDocumentLint（本批漏点） 未拒绝穿越输入 "../outside/escape.md"`、`GaeaPreview（resolvePreviewPath） 未拒绝穿越输入 …`）→ 还原后 `TEMP-MUTATE` 0 命中、复跑 ok。
2. `novel_bookcover.go:118` 改回 `safePrompt := b.String()` → `TestGaeaGenerateBookCoverSafeModePrompt` 红（`安全模式提示词应为「基线+安全段」逐字节相等`）→ 还原复跑 ok。

**主代理独立复核（不依赖其自述）**：`go test ./internal/gaea/wspath/...` → **ok 0.516s**；`go test ./internal/app/ -run TestWspathGuardMatrix -v` → **3 个 Test 全 PASS**（含 8 个合法输入子用例）；逐站读 diff 复核了上表 30 处改动；`internal/app` 与全仓 `go build ./...` → exit 0。

**主代理裁决（线 1 的三问）**：
- **裁决 1（绝对路径旁路保留）→ 接受保留**：绝对路径是前端/用户手输的合法用法（OfficePanel 路径输入框、附件/素材预览），本批一律不收紧（口径：**只改判据来源，不改允许面**）。`GaeaDocumentLint` 与 `GaeaConvertToPdf` 的读根门不同源 ⇒ **留池**（要补须单开一刀并同步 OfficePanel 交互）。
- **裁决 2（`withinReadRoots`/`withinWriteRoots` 语义变化）→ 接受**：大小写不敏感与 `root==abs` 计入是**对齐文件系统**；「相对 root 按 cwd 解析」与 `builtin` confiner 口径一致，且 `cfg.WriteRoots()` 明确可能返回相对根——旧行为（相对根恒不匹配 ⇒ 静默失效）才是要消灭的形态。
- **裁决 3（`filewatch` 索引「..foo」）→ 接受为假阴性修复**：被静默丢弃的合法文件重新进索引，与「文件监听对工作区内文件全覆盖」的宣称一致。

**收线加固（主代理做，线 1 已停）**：`matrixApp()` 由 `&App{}` 改为 `&App{core: &core{}}`——`emit` 是 `*core` 方法（`app.go:314`，`ctx==nil` 时直接 return），原写法一旦未来输入走到 `GaeaPreview` 的 `.doc/.xls/.pdf/pptx` 分支即 **nil 嵌入 panic**（P0#25 同族）。改后守卫档位由 `bare`（高危）降到 `core-only`（中危），「助手内 + bare 放大器」计数**不变（4 处，无新增）**；矩阵复跑 `ok 0.370s`。

### 1.2 线 2 · hook 沙箱接线（GA4-06）

**主代理复核（读 diff + 逐处对账，2026-10-02）**：

| 项 | 落地形态 | 复核结论 |
|---|---|---|
| 接线方式 | `hook.NewRunner(...).WithSandbox(bashSpec)`（`internal/gaea/boot/boot.go:275-284`）；`Runner` 新增不可导出字段 `sandbox sandbox.Spec` + `WithSandbox` 加法式设值（`internal/gaea/hook/runner.go:24-63`） | **不破坏既有调用面**：`NewRunner` 签名未变——主代理实测 `NewRunner(` 全仓命中 **22 处**（定义 1 `runner.go` + 生产装配 1 `boot.go:280` + 测试调用 **20**）；其中**改前既有测试调用 14 处一字未动**（`git show HEAD:...runner_test.go` 计数），新增 6 处属本批用例。与 bash 工具共用同一 `bashSpec`（`boot.go:221`） |
| 注入点 | `Runner.spawn`（`runner.go:66-105`）唯一注入：所有事件路径（`PermissionRequest`/`PreToolUse`/`PostToolUse`/`PromptSubmit`/`Stop`/`SessionStart`/`SessionEnd`/`SubagentStop`/`Notification`/`PostLLMCall`/`PreCompact`）从 `r.spawner` 改为 `r.spawn` | **收敛正确**：审计的缺陷形态正是「唯一生产构造器从不填」，逐站点填=下次再漏；单点注入 + 事件路径全改，无遗漏 |
| 诚实降级 | ①`enforce` 但平台不可用 → 不入 spec + `warnSandboxUnavailableOnce()`（`sync.Once`，经既有 `notify` 通道，一次性）；②平台可用但**本次包裹失败** → `SpawnResult.sandboxApplied`（不可导出，spawner→runner 信号）为 false → 同一告警 | **修掉两处静默**：原 `hook.go` 的 `argv, _ := sandbox.Command(...)` 丢弃 bool（=把失败读成成功）已改为 `argv, ok := hookSandboxCommand(...)` + `if ok` 才 `sandboxed=true`（`hook.go:399-423`）；判据不再用 `sandbox.Available()` 顶替 `Command` 的 bool（Windows 有 WSL2 仍可能 `WrapCommand` 失败，`sandbox/seatbelt_windows.go:15-24`） |
| 测试 seam | `hookSandboxCommand`（包级变量，`hook.go:326-333`）+ `Runner.sandboxAvailable`（字段，`runner.go:33-37`） | 用例不依赖真机 WSL2；**余量**：包级变量 seam 与字段 seam 风格不统一，且 `t.Parallel()` 下会竞争（本包当前无 parallel），已登记 |

**主代理追回的一处过程事故（值得进在册）**：收线预检时在 `hook.go` 读到 `// REVERSE-TEST: bool discarded` 未还原的变异标记，且 `sandboxed` 被无条件置真——**若不查，这就是「反向验证的中间态被当成最终态提交」**（把「平台可用」当「本次已沙箱」，又是一处静默撒谎）。已要求线 2 还原并改判据；核实后 `REVERSE-TEST|TEMP-MUTATE` 全仓 0 命中。

**主代理裁决线 2 提出的两处偏离/行为变更（2026-10-02）**：

| 项 | 线 2 的方案 | 主代理裁决 | 依据 |
|---|---|---|---|
| 去掉 `&& sandbox.Available()` 前置门 | 只认 `sandbox.Command` 的 bool 为唯一权威（`Available()` 仅用于「是否提前提醒」） | **接受** | `sandbox.Command` 内部已按平台自查（darwin `LookPath("sandbox-exec")` / linux `bwrap` / windows `DetectWSL2`+`WrapCommand`）；在 spawner 里再问一次 `Available()` 会引入第二个可能不一致的权威，正是「丢 bool 才撒谎」的成因 |
| enforce 默认下「多一条告警」 | 不做静默；接受每次会话一条 Notice | **接受**（并纠正其描述） | 主代理复核调用链：`Runner.<事件>` 先判 `Enabled()`（`len(hooks)>0`），`hook.Run` 只在 **命中事件的 hook** 循环体内调用 spawner（`hook.go:370-378`）⇒ **没有配置 hook 的用户零告警**，有 hook 的用户在**第一条真正要跑的 hook 之前**看到一次。噪声边界＝「有 hook 且真要跑」，「每次 boot 无脑刷一条」的说法不成立 |

### 1.3 线 3 · 桌面操作现场复核（IN4-05 / IN4-08）

**主代理实测（亲跑线 3 留在树里的 4 个临时探针：`go test ./internal/whisper/ -run TestZZProbe -v`）**——这部分证据**不依赖子代理是否交报告**：

| 探针 | 结果 | 结论 |
|---|---|---|
| `TestZZProbeInjection` | **PASS**（含控制组 A/B） | 控制组 A（未转义）**真的建出 `PWNED`**、控制组 B（转义）没建 ⇒ 探针有检出能力；7 类载荷（`'`、`$()`、`&`、反引号 n、双引号、换行、反引号 q）在 `openAppTarget`（`:317-326`）/`closeAppTarget`（`:328-341`）/`trashItem`（`:343-356`）上**全部 OK=false 且 marker 未创建** ⇒ **单引号上下文的命令注入不成立**（三处都做了 `'`→`''`） |
| `TestZZProbePathPolicy` | **PASS** | 真绕过：`\\?\C:\Windows\System32\evil.dll`、`\\localhost\C$\Windows\System32\…` 经 `normalizePath` 后仍 `hardblock=false`（前缀比较被「扩展长度/UNC 形态」绕开）；**不绕过**：`C:/Windows/System32/evil.dll` 归一后变反斜杠形态 → `hardblock=true` |
| `TestZZProbeNameVariants` | **PASS** | `\\?\` / UNC / 尾点 / 尾空格形态一律 `policyOK=true, hardblock=false`；`C:\WINDOWS\SYSTEM32\` 与常规形态正常拦截。**建文件全部 `Access is denied`**（无管理员权限）⇒ 定性必须是「**策略层绕过**」，不是「已证明能写进 System32」 |
| `TestZZProbeWildcardBypass` | **PASS** | **绕过实证（判据层）**：`explorer.exe` → `blockedByMap=true`（黑名单生效）；`explorer*` / `EXPLORER*` / `explo?er` / `explo?er.exe` / `*e` / `[a-z]xplorer` / `explorer*.exe` → `blockedByMap=false`，且 `Get-Process -Name '<该形态>'` **确实匹配到 `explorer`**；`*` 单字符更是匹配到本机约 200 个进程名 ⇒ 第 2 层黑名单可被通配符整体绕开（探针只列举进程名，未调 `CloseMainWindow`，无破坏） |
| `TestZZProbeTrashItem` | **PASS** | **推翻主代理的疑点**：我曾怀疑 `trashItem` 的 `$shell.Namespace(0)+ParseName(全路径)` 会查不到而「宣称删除实则不删」——实测绝对路径 `res.OK=true`、文件**确实消失**、且回收站里查到 `zz_trash_probe`（两条）⇒ **该功能可用**；只有传裸文件名（非路径）时才 `OK=false`，属预期。**这条不进缺口清单**（主代理推断被实测推翻，如实登记） |

**主代理独立复核（不受子代理报告影响）的两条实证缺口**：①`Get-Process -Name` 支持通配符（我实测 `(Get-Process -Name 'power*' …)` → `powershell`）⇒ 第 2 层关闭黑名单（`desktop_router.go:199-216` 只做精确匹配）可被 `explorer*` / `*` 绕过；②第 3 层写硬阻断（`:310-322` 只有两条 `HasPrefix`）可被 `\\?\` / UNC 形态绕过。两条都是「**判据空间 vs 执行空间分叉**」同族。

**审计原文按现场收窄**（详表见 §五）：IN4-05 的「零权限与路径校验」不成立（4 层管道在位）；IN4-08 的「注释称禁 `..` 逃逸、实现无校验」不成立（该注释本就明说「不做 cwd 圈定」，且 `Clean` 会消解 `..`）。

**线 3 定性（三条审计原文，主代理按其报告与自己跑的探针复核）**：

| 审计原文 | 判定 | 依据 |
|---|---|---|
| **IN4-05**「22 action 巨型 switch，**零权限与路径校验**」 | **收窄（不是撤销）** | switch 确在（`desktop_executor.go:40-134`），但权限/路径校验在 `desktop_router.go:51-156` 四层管道、逐层 `appendAuditOrFail` 留痕；「零校验」不成立。真实形态是**埋雷**：`RouterContext` 全仓仅测试构造、`DefaultAgentLoopRunner:34-39` 不设 Router ⇒ 今日只经测试触达、**一旦接线即生效** |
| **IN4-08**「`normalizePath` 注释称禁 `..` 逃逸，实现无校验」 | **撤销（行号失效=整条作废）** | 现树 `desktop_router.go:265-293` 注释**明说「不做 cwd 圈定」**，审计所引 `line 270-276` 的「禁止 .. 逃逸」字样**在现树不存在**（旧快照）；实测 `normalizePath("..\\..\\..\\Windows\\System32\\evil.dll","C:\\Users\\u\\docs")` = `C:\Windows\System32\evil.dll` 且随即被硬阻断（`desktop_test.go:89-106` 已钉死） |
| **IN4-09**（审计对 `'`→`''` 的评价） | **就命令注入这一面撤销** | 控制组 A/B 自证探针有检出能力（未转义→**真建出 `PWNED`**；转义→不建），7 类载荷 × 3 函数全部 `OK=false` 且 marker 未创建（主代理独立跑出同样结果）⇒ 单引号上下文下转义正确。**同族真缺口换了位置**（见下 ①） |

**本线真实缺口与落地（全部在 `internal/whisper/**`，+202/−20）**：

| # | 缺口 | 落地 |
|---|---|---|
| ① | **关闭黑名单可被通配符绕过**（判据 `desktop_router.go:199-216` 精确匹配 × 执行 `Get-Process -Name` 支持 `*?[]`） | 判据侧 `hasProcessNameWildcard` 形状拒绝 + 执行侧改 `Where-Object { $_.ProcessName -ieq … }`（**双侧同源**） |
| ② | **写硬阻断可被 `\\?\`/UNC 形态绕过**（前缀比较 × Windows 路径语义） | 新增 `canonicalizeForMatch` + `withinRoot`，`isSensitivePath`/`isHardBlockedWritePath` 改**边界化归一比较**（含 `System32foo` 不误拦的边界负例） |
| ③ | **`pathTo` 为空时静默落到进程 CWD**（`filepath.Dir("")=="."`） | copy/move 缺目标路径 → 硬阻断「缺少目标路径参数」 |
| ④ | **下载类 action 借 `AppActions` 早退漏过归一与硬阻断**（实测 `evaluatePathPolicy(ActionDownloadAndInstall,"C:\Windows\System32\evil.exe")` 曾 `OK=true`），而执行侧下载后**直接 `shellOpen` 执行** | 判据侧补「下载类落点同样受系统目录硬阻断」＋执行侧 `DesktopExecContext.Confirmed`（**零值 fail-closed**，语义＝「已过确认层」；`desktop_router.go:130` 传 `true`）；**真修法（下载后独立确认 UI）留池+拍板** |
| ⑤ | **`Router==nil` 时 `use_computer` 静默无结果**（今日可达：模型发动作→零 tool result→`shouldContinue=false`→`AllPassed=true` 收尾） | `agent_loop_runner.go:259-267` 追加诚实回执（走**既有** tool result 通道，零新增绑定/事件） |

**反向证据（线 3 跑，5 组，红→绿，全部还原；`TEMP-REVERSE-EVIDENCE` grep 0 命中）**：

| # | 改坏什么 | 红原文节选 |
|---|---|---|
| R1 | `hasProcessNameWildcard` 恒 false | 12 条；`TestExecuteUseComputer_WildcardCloseBlocked: target="*" 审计结果应为 blocked, got "allowed"` |
| R2a | `withinRoot` 回到裸 `ToLower` 前缀比较 | `isHardBlockedWritePath("\\?\C:\Windows\System32\x.dll") 应为 true` 等 |
| R2b | `withinRoot` 去掉段边界 | `TestPathPrefixBoundaryNotOverTightened: System32foo 落点不在 System32 内，不应硬阻断` |
| R3 | `pathTo` 空值守卫短路 | `copy_path 缺 pathTo 应硬阻断: {OK:true …}`（copy+move 各 3 条） |
| R4 | 执行器 `Confirmed` 门短路 | `未确认的 download_and_install 应被拒绝: {OK:false Content:Get "https://example.invalid/x.exe": EOF}` ← **真的把请求发到网络层**，证明旧写法无门 |
| R5 | `Router==nil` 诚实回执分支短路 | `got 0 条: []`；`诚实回执应计入工具结果, got {ToolRounds:1 TotalResults:0 AllPassed:true}` |

**既有用例处置（2 处，均为「旧用例把已定缺口当期望」）**：`agent_loop_runner_test.go:130`（旧断言 `len(results)==0` = 静默失败即期望）与 `:161`（旧断言 `AllPassed==true` = 空结果即「通过」）——按「允许改期望但须单列」的口径改，未改语义、未动其它包（主代理复核：两处都是把**已被本批判定为缺陷**的行为当期望，属合法改期望）。

**主代理复核与登记的两处**：
- `trashItem` **两次实测结果不一致**：主代理那次 `res.OK=true`、文件消失、回收站查到条目；线 3 那次把 `go test` **挂死 600s**（`$folder.ParseName(全路径)` + `InvokeVerb('delete')` 走 Shell 模态确认）。⇒ 按「**有阻塞风险、能否稳定移入回收站未证实**」登记为**存疑**，建议单开一刀（加超时/改 API）——**本批不硬造兼容**。
- **接线点（`internal/app` 侧，本批未动，留池+拍板）**：`RouterContext` 需由设置页权限开关 + 前端确认回调填充；`DefaultAgentLoopRunner` 需传 `Router`（含 `SessionID`）；`SetDesktopAgentSessionAutoApprove`/`SetTaskPlanDeleteAutoApprove` 需被设置/任务计划侧调用——否则四层管道与 22 action 永远只经测试触达。

### 1.4 线 4 · 前端两条真 bug（FE6-09 / FE7-11）

**主代理复核（读 diff，2026-10-02）**：

| 项 | 落地形态 | 复核结论 |
|---|---|---|
| FE7-11 副作用出更新器 | `frontend/src/schedule/store.ts`：`pushHistory(...)`/`newId(...)` 一律移到动作体内、`set` 更新器改纯函数返回（约 15 个动作）；`addTask`/`addGroup`/`upsertResource` 的 id **在 set 之前**生成，并保持 `idSeq` 的原有消费顺序（`upsertResource` 有 id 时不消费，注释写明） | **语义保留**：`activateBaseline`/`removeBaseline` 的早退（不入史）顺序不变；800ms 合并窗口（`:248-262`）与 `HISTORY_CAP=50` 未动 ⇒ 用户可见行为不变，只是重放/双跑不再推进游标 |
| FE7-11 附带真修复（审计未列） | `setTaskAssignments` 的「分组行禁挂分配」fail-closed 判据原先在 `pushHistory` **之后** ⇒ 被拒路径仍会压栈并清空 redo 栈；本批把判据提到 `pushHistory` 之前 | **真 bug**：拒绝一个非法操作不该改变撤销/重做历史。与批次十三「备份回滚部分失败」同族（都是「失败路径留下副作用」） |
| 主代理独立复跑（前端） | `npm.cmd run test -- --run src/components/ChatPanel.test.tsx` → **1 file / 6 tests passed**；`--run src/schedule/store.purity.test.ts` → **1 file / 4 tests passed**；`--run src/pages/NovelSettingPage.test.tsx` → **1 file / 27 tests passed**；`--run src/schedule`（全包）→ **42 files / 571 tests passed**（含 `store.history` / `store.sync` / `ResourcePanel`）；`tsc --noEmit -p tsconfig.app.json` → 0 error；`eslint <4 文件>` → 0 error | **独立于子代理自述**（唯一前置条件是会话中途放开沙箱，见 §3.2） |

**主代理执行的反向证据（线 4 的会话跑不了 vitest，配方由它给、变异与还原由我做；两个文件均以 SHA256 证明字节级还原）**：

| 变异 | 命令 | 红原文（要点） | 还原 |
|---|---|---|---|
| **A** 删 `ChatPanel.tsx` 循环后守卫（`:150-152`） | `vitest run src/components/ChatPanel.test.tsx` | `Tests 2 failed \| 4 passed`：**「卸载后循环不再写状态」**+**「切话题中止打字流」**两例红 | SHA256 回到 `5F85E750…C5720` |
| **B** 删 `ChatPanel.tsx` tick 首行守卫（`:141-142`） | 同上 | `Tests 1 failed \| 5 passed`：**只有「切话题中止打字流」**红（「卸载」仍绿 ⇒ 两处守卫**各自承重、非重复**） | 同上（哈希一致） |
| **C** 把 `pushHistory` 放回 `store.ts` `updateTask` 更新器 | `vitest run src/schedule/store.purity.test.ts` | `Tests 1 failed \| 3 passed`：**「updateTask：重放更新器不刷新 800ms 合并游标（不吞撤销步）」**红 | SHA256 回到 `09F7440E…8AD8E` |
| **D** 把 `newId('t')` 放回 `store.ts` `addTask` 更新器 | 同上 | `Tests 1 failed \| 3 passed`：**「addTask：同一更新器双跑不重复消费 idSeq（id 不变）」**红（＝审计原文漏掉的那处硬证据） | 同上（哈希一致） |

变异全部还原后复跑：**3 files / 37 tests passed**；`TEMP-MUTATE\|REVERSE-TEST` 在改动文件里 **0 命中**。

**主代理对线 4 两条 impact 纠错的独立复核**：
- **FE6-09 impact 不完整（成立）**：流式气泡渲染条件是 `ChatPanel.tsx:481 {streaming && streamText && (…)}`，而唯一宿主 `pages/NovelSettingPage.tsx:406-414` **不传 `streaming`**（默认 false）⇒ 真实页面上打字机文本**根本不显示**；用户可见后果主要是「回合结束把切书后刚重置的列表覆盖回旧对话」+「卸载后循环继续 setState」。
- **FE7-11 impact 不成立（成立）**：原文「`set` 里再 `set` ⇒ past/project 落不同批次 ⇒ undo/redo 跳步」按 zustand 5.0.15 语义无法复现（`pushHistory` 的 `setState` 先同步提交，外层再以「已更新 state」为基底合并 patch）⇒ 真实后果只有**更新器不纯**（双跑多消费 `idSeq` / 续期 `lastPush.at` ⇒ 吞撤销步）与「更新器求值期嵌套写 store」的反模式。

**线 4 三项裁决（主代理）**：①「切话题」启发式判据**接受**（保留 ≥1 条本会话 id 即不中止；边界用例「裁剪/分页仍含本会话 id 不误判」已绿），**残留风险**（把本会话消息全部裁掉会误判）与持久修法（父层显式 `conversationKey`/`key` 或 `onCancel`，在 `pages/NovelSettingPage.tsx`，本批足迹外）**进留池**；②`setTaskAssignments` 拒绝路径不再空压栈/清 future＝**接受为真 bug 修**（与 `activateBaseline`/`removeBaseline` 既有口径一致，redo 栈被清属用户可见数据损失）；③`setBaseline`（`store.ts:318-327`）同族残留**未改**，**单列一刀进留池**（它不在 FE7-11 定义内，避免扩面）。
| FE6-09 打字机守卫 | `frontend/src/components/ChatPanel.tsx`：新增 `typingCancelRef` + `ownedIdsRef`；卸载 cleanup 置位；**切话题**按「父层消息列表不再含本组件写过的 id」置位；`handleSendImpl` 开新回合复位；三处早退（`await onSend` 后、rAF 循环内每帧首行、循环结束后写终态前）以 `catch` 内再一处 | **与 canonical 同口径**：对齐 `frontend/src/hooks/useChatStream.ts:219`/`:230`（T6-3.3 两处守卫）；节奏未改（仍每帧 +3 字符，**未**顺手同步成按时间推进） |
| 余量（登记） | 切话题判据是**启发式**（按 id 归属）：父层若「保留同一批消息但整体重建」不会误判（id 比对），但父层若**裁剪**掉本组件的旧消息（如分页丢弃），会误判为切话题并中止在途打字 | 已登记为观察项；修法需父层显式给「话题 id」而非靠消息列表推断（拍板项） |


## 二、跨线裁决与缝合

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **足迹互斥按 Go 包划**（本批：线 1 = `internal/app`+`characterlib`+`scene`+`filewatch`+`builtin`+`wspath`；线 2 = `hook`+`boot`；线 3 = `whisper`；线 4 = 前端两文件） | **有效**：三条 Go 线并行期间，主代理在中途即跑通 `go build ./internal/app/` 与 `go build ./...`（exit 0）⇒ 无跨线签名互撞。**仍守既有纪律**：同包不并行、并发期不跑全量（禁止「边写边跑全量」=编译竞态自伤） |
| 2 | **主代理线索 → 子代理实测的双轨**（通配符绕过、`\\?\` 绕过两条由主代理先实证再交线 3） | **有效且必需**：线 3 独立复现两条并补出 ③④ 两条（`pathTo` 空值、下载类策略盲区）；若只等子代理报告，这两条大概率随「IN4-05 不成立」一起被销账 |
| 3 | **主代理自身的两次被推翻/被拦下** | ①`trashItem`「宣称删除实则不删」的推断**被实测推翻**（功能可用）；②线 2 树里的 `// REVERSE-TEST: bool discarded` **变异中间态**被收线预检拦下（否则就是「把平台可用当本次已沙箱」的静默撒谎提交）⇒ **两件事都进文档**，不因为「是我说的」就留白 |
| 4 | **冲突证据不装作一致** | `trashItem` 两条实测相反（主代理：成功移入回收站并可查条目；线 3：`go test` 挂死 600s）——**两条并列登记为「有阻塞风险、稳定性未证实」**，不许二选一写成结论 |
| 5 | **`check-test-ctors` +29 → +1 的评审与收线加固** | 线 1 按「方案 A」收敛为单一 `matrixApp()`；主代理复审发现该 helper 是 `&App{}`，而 `emit` 是 `*core` 方法（`app.go:314`）⇒ 未来输入一旦走到 `GaeaPreview` 的非 `.md` 分支即 **nil 嵌入 panic** ⇒ 收线改为 `&App{core: &core{}}`（`core.emit` 在 `ctx==nil` 时 return），守卫档位 `bare`→`core-only`，**放大器计数不变（4）**，基线刷新 363→364 后复跑「无新增」 |
| 6 | **线 3 提出给执行器加 `Confirmed bool`** | **接受但限定语义**：只允许写成「**已过确认层**」（不许写成「用户已确认」）；今天唯一生产路径恒为 true ⇒ 其价值是**零值 fail-closed**（未来新入口绕过确认层即被拒），并用 R4 证明「旧写法真的会发请求到网络层」；真修法（下载后独立确认 UI）**留池+拍板** |
| 7 | **两条线都撞沙箱边界（管道子进程 EPERM）** | 线 4 与线 3 均无法自跑 vitest/探针；**主代理代跑**（会话中途放开后）：前端 3 文件 37 例、`src/schedule` 571 例、`internal/whisper` 与线 3 探针全跑——**「子代理跑不了 ≠ 不验证」**，配方由它给、执行与红/绿原文由主代理出 |
| 8 | **`.tmp` 通配符删除越界（主代理过失）** | 见 §5.1：删线 4 的 4 个探针时连带删掉 6 个更早会话遗留的临时脚本；`.tmp/` 已 gitignore、无 tracked 删除。**纪律补充：删子代理临时物必须逐文件名，禁通配符**（`.tmp` 是共享暂存区） |
| 9 | **审计 fid 在本批的销账粒度** | 簇 A 四项全部落地；其中 IN4-08 与 IN4-09 判定**撤销**（不为它们造修法），IN4-05 **收窄**并转为 5 条真实缺口修法 + 接线点留池——**撤销也是结论**，且需带现场证据（行号失效/控制组探针） |


## 三、本批结论与量化收益（收线时填）

### 3.1 门禁与守卫（主代理亲跑）

**本地快闸（口径声明：这是「本地快闸」＝ `scripts/ci.ps1` 的步骤集，不是 GitHub Actions）**

| 步骤 | 结果 |
|---|---|
| `version drift check` | OK（三处版本一致 4.454.0） |
| `.tmp hygiene guard` | OK（256MB ≤ 512MB，无需清理） |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `golangci-lint`（钉 v2.14.0） | **0 issues** |
| `go test ./... -count=1` | **exit 0**（`internal/app` ok 113.965s；全仓无 FAIL） |
| frontend `lint` | exit 0（**0 errors** / 3 warnings，均为存量：`CreatePage.prevChapter`、`GhostText.test.waitFor`、`GhostText` ref cleanup） |
| frontend `build` | exit 0（`✓ built in 20.20s`，`dist/index.html` 在位） |
| frontend `vitest`（全量） | **427 files / 3712 tests passed**（批次十三为 425/3702 ⇒ 本批 +2 文件/+10 例） |
| E 系列回归守卫 | OK |
| `check-docs` 卫生四查 | OK（孤儿 0 / 悬空 0 / 指令预算内 / .ps1 编码合规） |

**守卫**

| 守卫 | 结果 | 备注 |
|---|---|---|
| `node scripts/check-primitives.mjs --strict` | **OK 无新增**（截断 37 / 原子写 8 / 前端 5 / heredoc 0） | 本批不欠基线刷新 |
| `powershell scripts/check-bindings-drift.ps1` | **OK @744** | 绑定面零变更（本批改了 20+ 个绑定方法的内部实现，签名一个未动） |
| `node scripts/check-contract-drift.mjs` | **无新增漂移**（warn 档 exit 0；5 条三态提示为存量） | — |
| `node scripts/check-test-ctors.mjs` | 363 → **364**（已评审刷新：线 1 的 `matrixApp()`，主代理收线加固为 `&App{core:&core{}}` ⇒ `core-only` 中危、放大器计数不变）；复跑「无新增」 | — |
| `node scripts/check-docs.mjs` | OK，但 **`.gaea/AGENTS.md` 61402 B 已近 65536 B 预算**（58000 B 警告线）⇒ 回写时必须分流 | 见 §六 |

**关键环境事实：`scripts/ci.ps1` 不能在「后台任务」里跑，必须前台跑（本批两次实测）**

两次以**后台 job** 方式执行 `scripts/ci.ps1`，均在 `go test ./...` 步大面积失败，**每一条失败都是** `open/mkdir C:\AI\wubigrok\.tmp\…: Access is denied`（涉及 `internal/app`、`channels/weixin`、`characterlib`、`config`、`export`、`gaea/backup`、`scene`、`snapshot`、`gaea/tool/builtin`、`whisper/db/repos` 十个包）。**判别实验**：
1. 用同样 `TMP/TEMP/GOCACHE` 在**前台**单跑这 10 个包 → **全部 ok**（含 `internal/app` 110.891s）；
2. 前台 `go test ./... -count=1`（与 ci.ps1 同口径）→ **exit 0**；
3. `clean-tmp.ps1` 已读码排除嫌疑（它只在 >512MB 时按模式删，本次未触发，不写权限）。
⇒ **判别变量是「后台 job 的进程树拿不到 `.tmp` 写权限」，不是代码、也不是并行负载**。台账：`ci-full.log` / `ci-full2.log`（两次失败原文）与 `.tmp` 前台复跑结果并存，按环境事实登记，**不作为本批红**。

### 3.3 本批量化收益

| 项 | 变化 |
|---|---|
| 路径穿越判据 | 手写实现（清点 **19 个零校验解析站点 + 12 处有判据站点**）→ **1 个叶子包 2 个原语** `wspath.Within` / `wspath.ResolveRelWithin`，**30 个调用点**改调；`builtin` 的 `within` **上提**为转调 |
| 真漏洞（审计未列或低估） | `gaea_lint.go`（零校验）、`gaea_preview.go`（**伪拒绝**：返回逃逸路径）、docx/pptx/xlsx×6/crosslink×2/xlsx_chart（**写侧零校验**，crosslink 还能在工作区外 `MkdirAll`）、`gaea_verify.go`（Journal 不可信面参与回滚写盘）、`gaea_schedule_file.go`（索引指针）、`gaea_deliverable_zip.go`（`\\` 形态漏拦）⇒ **13 个真修复** |
| 假阴性 | `filewatch` 的 `HasPrefix(rel,"..")` 误丢合法文件「..foo」→ 修复 |
| hook 沙箱 | `SpawnInput.Sandbox` **零生产填充** → 装配点接上 + **唯一注入点** `Runner.spawn`；`sandbox.Command` 的 bool 不再被丢（两处静默失败修复） |
| 前端不纯更新器 | `idSeq`（模块级游标）与 `pushHistory`（有状态副作用）从 **16 处** `set` 更新器内移出；附带修 `setTaskAssignments` 拒绝路径仍清 redo 栈的真 bug |
| 打字机流守卫 | 全仓 2 份实现中缺守卫的 1 份补齐（4 处早退 + 2 个 effect，与 canonical 同口径） |
| 桌面操作判据 | 5 条真实缺口（通配符黑名单、`\\?\`/UNC 前缀、`pathTo` 空值、下载类策略盲区、`Router==nil` 静默）⇒ 判据与执行**双侧同源** |
| 测试 | 新增 `wspath/resolve_test.go`、`wspath_guard_matrix_test.go`、`desktop_hardening_test.go`、`ChatPanel.test.tsx`、`store.purity.test.ts`；守卫基线 `test-ctors` 363→**364**（已评审：`core-only`，放大器计数不变） |

### 3.4 余量（如实登记，未硬凑）

1. **AP5-09 未覆盖**：符号链接/目录联接逃逸（需 `EvalSymlinks`，`wspath` 包注释已写明不覆盖）；`snapshot.go:29` 与 `scene.go:218` 两份同名 `validSceneID`；`internal/schedule` 与 app 层两份段级校验（**都更严，硬并会放宽**）；`GaeaDocumentLint` 的读根门与 `GaeaConvertToPdf` 不同源（**留池**，要补须同步 OfficePanel 交互）。
2. **GA4-06 未覆盖**：hook 命令在 `WriteRoots` 受限平台上的实际行为（本机无 WSL2/bwrap/sandbox-exec，**真包裹测不到**）；`hookSandboxCommand` 是**包级变量 seam**（与 `Runner.sandboxAvailable` 字段 seam 风格不统一，`t.Parallel()` 下会竞争——本包当前无 parallel）。
3. **IN4-05 未接线**：`RouterContext` 的生产构造者、`DefaultAgentLoopRunner` 传 `Router`（含 `SessionID`）、两个 session 级自动批准设置——**都在 `internal/app` 侧，本批未动**（留池+拍板）；`canonicalizeForMatch` 只做**词法**归一，8.3 短名与符号链接未覆盖（需 IO，不在纯策略函数里做）。
4. **`trashItem` 存疑**：两次实测结果不一致（成功移入回收站 / 挂死 600s）⇒ 建议单开一刀（加超时或换 API）再定性。
5. **FE6-09 判据是启发式**：父层若把本会话消息**全部**裁掉会误判为切话题；持久修法（父层显式 `conversationKey`/`key` 或 `onCancel`）在 `pages/NovelSettingPage.tsx`，**足迹外**。
6. **FE7-11 同族残留**：`store.ts:318-327 setBaseline` 的 `pushHistory` 在 `if (!r.ok) return` **之前** ⇒ 快照失败也空压栈 + 清 future（**单列一刀**）。
7. **真机/环境欠账**（承接）：`-race` 只在 Actions（本机无 gcc）；FE6-02 真机拔麦走查仍未做；本批门禁前段撞到的 `.tmp` 写入瞬时拒绝（见 §3.2）未复现，**按环境瞬时性登记**。



### 3.2 本批环境事实（与代码无关，但会决定门禁红在哪一侧）

| 项 | 实测（主代理，2026-10-02） | 对策 |
|---|---|---|
| `GOCACHE` 默认位置不可写（**仅限本批前段**） | 前段（workspace-write 沙箱）：`go env GOCACHE` = `%LOCALAPPDATA%\go-build`，`New-Item` 探针 → **访问被拒绝** | 前段一律带 `GOCACHE=<ws>\.tmp-gocache`。**注意**：`go build` 有时仍返回 0——那只是**全部命中缓存**，不代表缓存可写（本批踩过一次，靠文件探针才定性） |
| `GOMODCACHE` 写入被拒（仅限前段） | 前段 `go build ./...` 仍 **exit 0**，stderr 有 `writing stat cache: …\go\pkg\mod\cache\…info.tmp: Access is denied` | 非致命（模块已下载，只写 stat 元数据）；`ci.ps1` 的 `Invoke-Native` 只认 `$LASTEXITCODE` |
| **前端的 esbuild EPERM 边界（仅限前段）** | 前段 `vitest run`（无论 `npx` 还是 `npm run test`）必然 `Error: spawn EPERM`（esbuild 服务进程以管道 stdio 派生，被沙箱挡）；`tsc -b` 也被拒（写 `frontend/node_modules/.tmp/*.tsbuildinfo`） | 前段用两条替代验证：`tsc --noEmit -p tsconfig.app.json`（**0 error**）与 `npx eslint <改动文件>`（**0 error**）；vitest 待沙箱放宽后补跑 |
| **会话中途策略变更** | 用户把文件策略改为 **danger-full-access**（审批提示同时关闭） | 上述三条边界消失：vitest 实跑通过（见 §3.1）；后续不再申请升级（会话内**禁止**设置 `sandbox_permissions`） |
| `-race` | 本机无 gcc（在册事实），竞争检测只在 GitHub Actions | 本批不声明 race 结论 |



## 四、留池与待拍板

### 4.0 本批新增留池与拍板（簇 A 收口带出的）

**新增留池（不需拍板，可排队）**：
1. `store.ts:318-327 setBaseline` 同族残留（`pushHistory` 在失败判据之前 ⇒ 快照失败也空压栈+清 future）——单列一刀。
2. `trashItem` 阻塞风险（两次实测不一致）——单独实测 + 加超时/换 API。
3. 符号链接/目录联接逃逸（`wspath` 不覆盖，需 `EvalSymlinks`）；`snapshot.go:29` 与 `scene.go:218` 两份 `validSceneID`；app 与 `internal/schedule` 两份段级校验。
4. `check-test-ctors` 仍是 **warn 档**（未接 CI strict）——验收线（是否允许 `core-only`）需拍板后接入。

**新增拍板项**：
- **桌面 agent 接线（IN4-05 的根）**：`RouterContext` 生产构造者 + `DefaultAgentLoopRunner` 传 `Router`（含 `SessionID`）+ 两个 session 级自动批准设置的调用方——**不做这一步，四层管道与 22 action 永远只经测试触达**；(a) 接线（M）/ (b) 明确弃用并删执行器（S，但要接受能力回退）/ (c) 维持现状（只留埋雷）。
- **`download_and_install` 真修法**：下载后**独立**的用户确认 UI/流程（当前只有 `Confirmed` 零值 fail-closed 纵深防御）。
- **`GaeaDocumentLint` 读根门**：是否与 `GaeaConvertToPdf` 同源（会咬 OfficePanel 手输绝对路径）。
- **FE6-09 话题标识显式化**：父层传 `conversationKey`/`key` 或 `onCancel`，替掉「按消息 id 归属」的启发式。

### 4.1 下一批候选短名单（按 round-15 §4.1 四簇，主代理本轮预核状态标注）

**簇 A · 安全 / 边界**（**本批已落地**：AP2-04 书封护栏、AP5-09 路径单源、GA4-06 hook 沙箱、IN4-05 收窄为 5 条真实缺口、IN4-08/IN4-09 撤销——见 §一、§五）

**簇 B · 真 bug / 竞态**（FE6-09 / FE7-11 已在本批线 4 吃掉；余下候选）

| fid | 问题 | 位置（审计当时） | 主代理预核（2026-10-02 晚） |
|---|---|---|---|
| FE6-09 | 打字机流两份，其中一份没有取消守卫 | `frontend/src/components/ChatPanel.tsx:113-122` | **本批线 4** |
| FE7-11 | 模块级可变游标在 `set` 更新器内被改写 | `frontend/src/schedule/store.ts` | **本批线 4**（主代理已锁定：`idSeq` `:207-208` 经 `newId` 在 `:359`/`:376-377` 的更新器内自增；`pushHistory` `:248-262` 是有状态副作用却被塞进数十个更新器） |
| IN2-13 | `ConsistencyReport` 同名两套定义、JSON 字段不兼容 | `internal/graph/consistency.go:19-35` | **预核成立（两套已定位）**：另一套在 `internal/types/types.go:49-61`。字段面确不兼容——graph 版 issue 有 `category/entity_name/location/evidence/branch`、report 有 `total_issues/summary`；types 版 issue 是 `section`、report 是 `overall_note`。两者同名同 JSON tag（`issues`）语义不同 ⇒ 消费端按哪套解析都可能拿到空字段 |
| GA1-09 | retry 循环里的 run 包装分支**永不可达** | `internal/gaea/agent/task.go:726-739` | **预核成立（真死路径）**：`:727-730` 只在 `run != nil` 时给 `subSession` 赋值，`:738` 却判 `run == nil && subSession != nil` ⇒ 条件恒假。修法=要么删死分支，要么补上「run 为 nil 时用新建 session 续跑」的真语义（须先判定哪个是原意，不可盲删） |
| AP1-07 / AP1-08 | 取消残稿三态落盘三份重复 / 重写写回三副本 | `internal/app/create_chapter_handler.go:455-470` / `novel_rewrite_handler.go:328-356` | 未预核（属 `internal/app`，与线 1 同包 ⇒ **必须单独一波**，不可与本批并发） |

**簇 C · 口径单源（consistency）**：IN1-09（AI 味判据三套严重度）· IN1-05（`Profile` vs `Fingerprint`）· GA3-07（知识库检索双口径）· IN3-02（日历语义 Go 四处 + TS 一份镜像）· FE4-04（绑定方法名三份手工清单）· GA6-01/GA6-04/GA6-06（成本条目 UPSERT 三份 / 检索三路 / 包级 BM25 隐式契约）。

**簇 D · 死代码与僵尸资产**：IN4-04（桌面能力 3 条路由、2 条 0 生产调用——**本批预核已独立佐证**：`RouterContext` 生产零构造）· GA4-09（`SkillLayer.Learner` 桩）· GA2-03（`CompactDescriptor` 无条件生效）· FE3-03（漂移检查只认「名字被认领」，死绑定无法发现——**实现口径必须先按「前端驼峰名 + `mappings.ts` 反查」**，见批次十三误判教训）· X1-11（前端 mock 是第三份手写契约）。

### 4.2 待拍板（仍未动，需用户裁决）

- **AP4-01** 三套并行编排合并（DAG / 收件箱 / `gaea/tasks`）：(a) 抽共用原语（M）/ (b) 统一内核（L/XL）/ (c) 维持现状。
- **X1-03** 五重 `*core` 嵌入裸构造 panic：(a) 生成器注入 `assertAssembled()` + 227 处测试桩迁移（L）/ (b) nil-safe 访问器（M）/ (c) 维持守卫。
- **`estimateTokens` 四份口径**（P5，批次十一/十二/十三三度入池）：口径真不同，合一会改变预算行为。

## 五、审计原文纠错与遗留登记

> 口径：**审计结论必须能被现场复核，行号错=整条作废**；本批坚持「先按现场定性，再按缺口修」，纠错不是免责。

| # | 审计/前批原文 | 本批实测（谁测） | 处置 |
|---|---|---|---|
| 1 | AP2-04「书封生图**绕过图像域与 play 安全护栏**」 | **一半已被批次十二顺带修掉**：`:115` 早就是 `effectiveSceneIllustrationModel(a.cfg)`、`:169` 台账记生效 backend/model；**护栏旁路仍在**（3 个注入点不含书封）（主代理预核 + 线 1 复核） | 本批按「护栏旁路」修（§一 1.1） |
| 2 | AP5-09「路径归一 **15 处**手写，穿越防护只在 1 处生效」 | **数字偏小**：主代理独立清点 = 真判据站点 **≥17 处、跨 ≥8 包**（§〇.1 的 19 行表）；更严重的是原文没写的**写侧逃逸族**——`gaea_docx_edit.go`（Apply/AcceptChanges）、`gaea_pptx_edit.go`、`gaea_xlsx_edit.go`（Plan/Apply/SetCell/RowOps/ColOps/Recalc）、`gaea_crosslink.go`（xlsx 源 **与输出**）、`gaea_lint.go` 原为**无校验 Join**（可写/读工作区外）；`gaea_deliverable_zip.go:38` 的 `HasPrefix(clean,"../")` 在 Windows 上**拦不住反斜杠形态**（主代理逐站读码，线 1 落地） | 全部改调 `wspath` 单源（§一 1.1）；`filewatch.go` 顺带修「`..foo` 合法文件被误丢」的过度拒绝 |
| 3 | IN4-05「22 种桌面操作巨型 switch，**零权限与路径校验**」 | **不成立**（按现场收窄）：`desktop_router.go:51-156` 有 4 层管道（设置级拦截 / 关闭黑名单 / 路径策略 / 确认+跳过确认）且每步 `appendAuditOrFail` 留痕；执行器只是**分层之后**的那一层。**真实可达状态更弱**：`ExecuteUseComputer` 生产唯一调用点 `agent_loop_runner.go:254` 被 `:245 if r.Router != nil` 包着，而 `RouterContext` 全仓**只在测试里构造** ⇒ 整链今天只经测试触达（埋雷定性）。**真缺口**在判据层：①关闭黑名单可被**通配符**绕过（实测 `explorer*`→`blockedByMap=false` 且真匹配到 `explorer`；`*` 匹配约 200 个进程名）②写硬阻断可被 `\\?\` / UNC 形态绕过（`hardblock=false`；实际写盘被 ACL 拒，属**策略层**绕过）（主代理跑线 3 探针 + 独立 Go 探针） | 修法＝判据与执行**双侧同源**（§一 1.3，收线时定稿） |
| 4 | IN4-08「`normalizePath` 注释称禁 `..` 逃逸，**实现无校验**」 | **不成立**：该注释本就明说「**不做** cwd 圈定——绝对路径是桌面操作的合法输入」，且 `filepath.Clean` 会消解 `..`（实测 `..\..\..\Windows\System32\evil.dll` + cwd → `C:\Windows\System32\evil.dll` 且 `hardblock=true`）；另探针实测三处 PowerShell 拼接都做了 `'`→`''`，7 类注入载荷**全部未建出 marker**（控制组 A/B 证明探针有检出能力）⇒ **命令注入亦不成立**（主代理跑探针） | 按「收窄/撤销」登记；不再为它造修法 |
| 5 | AP8-08 之外，主代理曾怀疑 `trashItem`「宣称删除实则不删」 | **主代理推断被实测推翻**：绝对路径 `res.OK=true`、文件消失、回收站里查到 `zz_trash_probe` ⇒ 功能可用（主代理跑探针） | **不进缺口清单**（如实登记「推断被推翻」） |
| 6 | 线 2 报「enforce 默认下**每次 boot** 多一条 hook 告警」 | `Runner.<事件>` 先判 `Enabled()`，`hook.Run` 只在**命中事件的 hook** 循环体内调 spawner ⇒ **没配 hook 的用户零告警**（主代理读码纠正措辞） | 行为变更本身接受（§一 1.2 裁决表） |
| 7 | FE7-11 impact「`pushHistory` 内部再 `setState` 形成重入，历史栈与 project 落不同批次，undo/redo 出现跳步」 | **不成立**：`pushHistory` 的 `setState({past,future})` 先同步提交，外层 `setState` 再以「已更新 state」为基底合并 patch，`past/future` 不会丢（线 4 按 zustand 5.0.15 源码逐行核对；主代理复核其推理与用例 D 的红/绿一致） | 真实后果改述为两条：**更新器不纯**（双跑多消费 `idSeq`／续期 `lastPush.at` ⇒ **吞撤销步**，已用变异 C/D 实测红）+ 更新器求值期嵌套写 store 的反模式（当前无可观测故障） |
| 8 | FE6-09 impact「仍会对已卸载组件 `setStreamText`」 | **句子对、描述不完整**：真实页面上打字机文本不渲染（`ChatPanel.tsx:481` 需 `streaming`，宿主 `NovelSettingPage.tsx:406-414` **不传**）⇒ 用户可见后果是「回合结束把切书后刚重置的列表**覆盖回旧对话**」+「卸载后循环继续 setState」，不是「用户看见打字继续跑」（线 4 提出，主代理读码复核成立） | 按此改述；修法不变（本批已修） |

### 5.1 收线附带处置（如实登记）

- **`.tmp` 清理越界（主代理过失）**：线 4 的 4 个沙箱探针（`vitest-sandbox-preload.cjs` / `worker-probe.cjs` / `spawn-probe.cjs` / `child-echo.cjs`）要求删除；主代理执行 `Remove-Item .tmp\*.cjs` 时**连带删掉了 `.tmp` 下 6 个更早会话遗留的临时脚本**（`diag-sin-composer.cjs` / `diag-sin-ui.cjs` / `retry-illustrate.cjs` / `shot-antbtn.cjs` / `shot-cost-light.cjs` / `verify-cleanup-illustrate.cjs`）。**影响面已核**：`.tmp/` 在 `.gitignore:86` 内、`git status` 无任何 tracked 删除、这些文件本就被 `clean-tmp.ps1` 定期清理。**教训**：删「某子代理产生的临时物」必须**按文件名逐个删**，禁通配符批量删——`.tmp` 是共享暂存区，不是某个子代理的私有目录。
- **探针删净复核**：线 3 的 5 个探针（`zz_probe_injection/path/variant/wildcard_test.go` + `zz_close_positive_test.go`）与线 4 的 4 个 `.cjs` 全部删除，`Test-Path` 全 False、`git status` 零残留；另删除本批缓存与临时目录（`.tmp-gocache` / `.tmp-gotmp` / `.tmp-probe-removed` / `.tmp\zz-pathcheck`）。

## 六、交付物与记忆回写

| 项 | 状态 |
|---|---|
| 代码 | 36 个文件修改 + 7 个新文件（3 个新测试 + 2 个新原语 + 2 个新守卫用例文件）；**绑定面 744 零变更**；未抬版本、未动 CHANGELOG（发版仪式归用户） |
| 本文档 | `docs/code-audit-2026-10-02/round-16-p1-batch14.md` |
| 守卫基线 | `docs/code-audit-2026-10-02/machine/testctors-baseline.json`（363→**364**，已评审） |
| 进度记忆 | `.gaea/progress.md` 顶部新增「非版本刀：全仓审计第 14 批·簇 A 安全」一条（含坑/教训 7 条与留池/拍板清单） |
| 项目记忆 | `.gaea/AGENTS.md` 顶部速览新增本批条目；**并触发指令预算分流**：`node scripts/check-docs.mjs` 报 `66229 B > 65536 B` ⇒ 迁最老 **7 条**非版本刀（品牌资产补齐 / 壳内全板块渲染健康巡检 / DAG 6.3 真机走查 / v4.337~v4.343 真机走查班 / todos 对账清账 / 对比度普查收账 / v4.270 留池补验收官）入 `docs/archive/agents-version-history-2026-09.md`（机械搬运：行首标记精确取行、零转录）⇒ **66229 → 56508 B**，`check-docs` 由 FAIL 转 OK 且不再压 58000 B 警告线 |
| 分流事故与修复 | 首次写入 AGENTS.md 时突破预算，**harness 把「指令载荷」截短了约 997 B**（磁盘文件未受损，已用「工作树尾部 == HEAD 尾部」核过）；随后按上面的分流把文件压回预算内，并把「删/迁一律走机械搬运 + 复核字节数」写进 §5.1 与本批教训 |

