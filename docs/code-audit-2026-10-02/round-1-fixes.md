# 第一轮修复记录（P0 止血刀）· 2026-10-02

> **来源**：`docs/code-audit-2026-10-02/README.md` §5「第一批：止血刀」中可并行、足迹互斥的条目。
> **编排**：7 条并行子代理线（4 槽位排队），主代理定契约、独立复核、跑集成门。
> **纪律**：每刀**先让测试红、再改绿**；禁跑 `go test ./...` / `scripts/ci.ps1` / `wails build`（避免与并发会话/并行线互相踩）；只碰各自足迹。
> **快照**：起始 HEAD `3d0210d1`（v4.454 时代）；工作树另有并发会话的未提交改动（本记录不含那些）。

---

## 一、已交付（7 刀）

### 刀1 · 配置保存丢段（审计 P0-1 / P0-2）

- **根因**：`RenderTOML` 是**有损归一化渲染**（只覆盖渲染器建模的段与键）；`Config.SaveTo` 直接用它整文件重写 → 一次「始终允许」审批（`boot.go:623-632` 的 `PersistAllowRule`）**或任何一次设置面板保存**（`app/gaea_ui_extra.go:147 gaeaApplyCfg → gaeaConfig.Save`）即把未建模的顶层段与被漏渲染的段内键从磁盘静默删除。
- **改动**：新增 `internal/gaea/config/render_preserve.go`（`RenderTOMLPreserving(c, existing)`，三层保留：根标量**前置** / 渲染器负责段做**键级合并** / 未知顶层段逐字追加）；`edit.go` 的 `SaveTo` 与 `config.go` 的 `WriteFile` 改走它（签名、原子写、权限位不变）；`render.go` 函数体零改动，仅注释标注「落盘请用 Preserving」。`rendererOwnedTopLevel = agent/providers/tools/permissions/space_profiles/sandbox/plugins`。
- **实测踩到的坑（已修）**：① 根标量按原位置写会被 BurntSushi 吞进上一张表（`[sandbox]` 之后的 `workspace = …` 读回空串）→ 根标量必须前置到第一张表之前；② `Load()` 对 `Providers` 是**整片替换**，故 `[[providers]]` 必须**按位次**配对（按 name 配对会把用户的 thinking/effort 补到别的条目）；③ 遗留 provider 条目必须连表头一起保留，否则掉进上一张表；④ `sandbox`/`plugins` 也在渲染器职责内（原清单漏列），当未知段整块保留会写出**重复表头 = 非法 TOML**。
- **验收（三重独立）**：
  - 线1 红证（临时探针 `GAEA_ZZ_LOSSY=1` 退化为旧路径）：`16 段丢 11 段`、`workspace 读回空串`、`known 段内 7 个键全丢`；绿证 4 条测试全 PASS（含钱测试「13 段逐字段零漂移」、真实配置形状、我追加的键级测试、连存 3 次收敛）。
  - 主代理端到端探针（`.tmp/configprobe`，生产同源链 `Load → AddPermissionRuleForSpace → Save`，**用真实配置的副本**、`APPDATA` 指向临时目录）：**修复前 16 段 → 5 段（丢 11）、`Session.Space` work→空**；**修复后 16 → 16、字段零漂移**。
  - 主代理注入非空值的合成夹具（`effort="high"`、`subagent_effort="low"`、`approval_timeout_secs=42`、`compact=true`、provider `thinking="adaptive"`/`effort="max"`）：**8 个字段往返全部保真、0 漂移**。
- **未覆盖**：`[space_profiles.X.permissions/.guardrails]` 与 `[[plugins]]` 的键级合并无专项用例（代码路径支持、既有测试仍绿）；用户注释**内容逐字保留但位置会挪到文件末尾保留区**；`internal/config` 经核实**不是**同类有损（read-modify-write + 全字段 marshal），未动。
- **用户数据保护**：原件一字未动；同目录留下 `config.toml.bak-20261002-audit-pre-RenderToML-fix`（5995 B）。

### 刀2 · 任务收件箱读-改-写无锁（审计 P0-3）

- **问题**：`internal/app/gaea_task_inbox.go` 的 `taskInboxSave` = `loadTaskInbox → 改切片 → saveTaskInbox`（temp+rename 整写），**全文件无锁**；写者来自 UI 面板 / Ctrl+K / 语音 `execSaveTask` / 微信通道，并发时后写者用旧快照覆盖 → 静默丢卡；`MaxTasks=500` 上限判定同样基于过期切片。
- **改动**：新增包级 `taskInboxMu sync.RWMutex`；`taskInboxSave`（新建+编辑）、`GaeaTaskInboxSetStatus`、`GaeaTaskInboxDelete` 三处 RMW **整段**进临界区；`GaeaTaskInboxList` 取 RLock 求一致快照；`loadTaskInbox`/`saveTaskInbox` 保持裸原语并在注释写明「调用方必须持锁」（防嵌套自死锁）。仅 2 文件 +98 行，零签名/格式变更。
- **红 → 绿**：
  - 红（`TestTaskInboxConcurrentSaveNoLostUpdate`，20 goroutine 并发新建）：`并发新建 20 条后 List 应 20 条，实得 1 条` ＋ 3 次 `rename ... Access is denied`。
  - 绿：`-run TaskInbox` ok 0.723s；`-count=5` ok 2.295s；`-run 'TaskInbox|SaveTask|RouteIntent'` ok 1.692s；`go vet`/`gofmt` 干净。
- **未覆盖**：锁是**进程内**的——双开 app 仍存在跨进程丢更新（temp+rename 只防半截文件），需文件锁或单实例约束（已写入注释，另立一刀）。本机无 gcc，`-race` 无法本地跑（race 门在 Actions）。

### 刀3 · `ExtractJSON` 取最后一个右花括号（审计 P0-22）

- **问题**：`internal/util/util.go` 用「第一个 `{` + 最后一个 `}`」切 JSON；回复里出现两个对象或后文再提 `{}` 即切出非法 JSON。实际受影响：**21 处生产调用点**（审计报 25，线3 精确普查后更正，并补出 3 处漏列的调用点）。
- **改动**：重写为**括号配平状态机**（`scanBalanced`，处理字符串内花括号与转义）＋ ` ```json ` 围栏优先；优先级：对象 > 结构化数组（首元素是 `{`/`[`）> 兜底第一个完整片段；无完整片段时**原样返回**（与旧行为一致）。签名/包路径不变 → 调用点零改动。
- **红 → 绿**：红证据用 `go test -overlay=` 把 `util.go` 映射到 HEAD 的字节一致副本（blob hash 已校验），**磁盘文件从未回退**：9 条 FAIL（含 `{"a":1} {"b":2}` 与 `[{"op":1},{"op":2}]` 被剥掉方括号）。绿：`./internal/util/...` ok（56 PASS / 0 FAIL）。
- **顺带上报**：`internal/bookimport/reconstruct.go:218-239` 是**同一 bug 类的第二份实现**（已另开线7 处理）；`scripts/test_herdsman_models.go:173` 是第三份局部拷贝（脚本件，仅登记）。
- **未覆盖**：括号类型不匹配不检测（`{"a":1]` 返回该片段而非整串，两者都非法、下游路径相同）；正文里合法的结构化方括号字面量仍可能抢先（支持 `[` 的固有代价）。

### 刀4 · GenUI 校验器把数据数组当子节点（审计 P0-18）

- **问题**：`internal/gaea/genui/validate.go` 的 `walk` 把 `diffs`（diff 行数据）与 `series`（chart 数据序列）当节点数组递归 → 逐行误报「缺少 type 字段」，最多刷满 30 条，并侵占节点预算；模型据此**反复改写甚至放弃组件**。
- **改动**：拆成 `checkData`（数据字段校验）与 `walkChildren`（只递归真子节点容器：`row/col/grid/card.items`、`list.items`(允许字符串)、`tabs[].items`、`accordion[].items`）；`maxErrors=30` 常量化并注明与 TS 侧 50 的口径分叉。
- **红 → 绿**：新增 8 条用例，其中 **5 条是反向用例**（`row.items`/`card.items`/`tabs 行 items`/`accordion 行 items`/`list` 对象元素缺 `type` **必须仍报错**），证明不是把校验关掉；`./internal/gaea/genui/...` ok。
- **主代理独立复核**：对照 `frontend/src/genui/spec.ts` 逐个核过——节点容器集合与 `walkChildren` 覆盖**完全一致**（`timeline.items`/`steps` 确是数据行，未被误当节点）。
- **未覆盖**：Go/TS 常量与白名单仍是**两份事实源**（本轮只修误判，未收敛同源；`spec.ts` 与 `validate.go` 的集合一致性尚无双向断言）。

### 刀5 · 进度计划数据契约（审计 P0-20 / P0-21 ＋ 新发现第三条）

- **问题**（线5 实测复核后的准确口径）：
  1. `schedule.Project` **无 `Baselines` 字段** → 前端多基线（v4.137 引入）经 Go 一次保存即被整量覆盖抹掉；**agent 通道同因**（`internal/gaea/tool/builtin/schedule_tools.go:255-277` 也是 Load→ApplyOps→Save）。
  2. **审计原文方向写反了**：`Units` 早已是 `*float64`；真正残留的是 `Quantity`/`Amount` 裸值 + `omitempty` —— 真实方向是 **显式 0 → 缺键**（不是缺键 → 0）。
  3. **本轮新发现**：`SchedResource.calendar`（资源级个人日历，前端 `ResourcePanel.tsx`/`usage.ts` 消费）在 Go `Resource` 里**完全不存在** → 同因静默丢失，性质同为 P0。
- **改动**：`types.go` 加 `Baselines []Baseline`（纯透传，FIFO/同名替换口径仍唯一在前端）、`Quantity/Amount float64 → *float64`（nil=不落键，0=落 0）、`Calendar *Calendar`；消费点 `project.go`/`cost.go`/`analysis.go` 改显式判空（与前端 `??` 缺省逐位一致）。**前端零改动**（`types.ts` 键名与可选性本就正确）。
- **红 → 绿**：三条往返用例经**绑定层生产函数** `GaeaScheduleSave/GaeaScheduleLoad` 红（`期望 3 条，读回 0 条`；`显式 quantity=0 丢失`；`calendar 被静默擦除`）→ 绿；`./internal/schedule/...` 全包 ok；`tsc -b` EXIT=0；vitest 定向 7 文件 137 例全绿；**`ops_golden.fixture.json` 零改动**（证明指针化对非零值逐字节兼容）；零迁移（旧文件行为不变）。
- **未覆盖（线5 登记在案，建议各立一刀）**：① `schedule_apply set_baseline` op 不 append 到 `Baselines`（agent 存的基线不进前端槽位）；② `usage.ts` 的 `maxUnits ?? 1` 与 Go 零值分叉（显式 0 被放大为 1）；③ 活跃基线 `null` → 缺键（消费点均真值判断，行为一致）；④ 缺「`types.ts` ⊆ Go json tag」的跨语言漂移闸。

### 刀6 · 原语平行实现守卫（warn 档 ＋ 真实基线）

- **交付**：`scripts/check-primitives.mjs`（默认 warn 恒 exit 0 / `--strict` 有超基线新增才 exit 1 / `--json` 机读 / `--write-baseline` 重建）、`docs/code-audit-2026-10-02/machine/primitive-baseline.json`（真实首跑，`head=3d0210d180d6`）、`machine/README.md`（用法/口径/验收线/待拍板项）。**未接 CI**（避免打断正在发版的并发会话）。
- **实测口径**：① 截断原语顶层函数 **41**（审计的 41 只在 `grep -i` 下可复现，大小写敏感仅 37）＋方法形式 1 = **42**；② `internal/app` 原子写旁路（`os.CreateTemp`/`ioutil.TempFile`）**13**；③ 前端保存/解码旁路 **5**（`a.download=` 3 含 1 测试、`atob(` 2）。与主代理独立复核一致。
- **负例验证**：临时探针令 `--strict` exit 1 且精确报出新增位置；warn 档仍 exit 0；探针删净并 `Test-Path` 复核；B/C 两条检测在 `%TEMP%` 隔离树验证（避免在仓内留 Go/TS 探针打断别人的 `go build`）。
- **建议验收线（接 CI 前拍板）**：截断 41 → ≤5；`internal/app` 原子写 13 → 0；前端旁路 5 → 0；之后一次性把 `--strict` 接进门禁。

### 刀7 · `bookimport` 的第二份 JSON 提取实现（P0-22 同类）

- **问题**：`internal/bookimport/reconstruct.go` 自带一份 `strings.IndexAny(s,"{[")` + `LastIndexByte(s, closeCh)` 的提取器，与 `util.ExtractJSON` 同属一个 bug 类（线3 越界上报）。
- **改动**：改为 `util.ExtractJSON` 的**薄包装**（无 import 环：util 只依赖 stdlib），逐字保留本包原契约「找不到即报错、标量/null 不算结构」；顺手删掉只为旧实现服务的私有 `stripCodeFence`。调用点（`CallJSON` → `app/novel_import_ai.go:354/379`）零改动。
- **红 → 绿**：红 4 个子用例（两对象拼接 / 正文+对象+后文 `{}` / 字符串内花括号 / 嵌套后噪声）→ 绿全包 ok；两个「护栏」用例（对象数组先取、裸围栏容错）改前改后均 PASS（证明没丢原容错面）；端到端 `TestNovelOutlineReconstruct` ok。
- **残留**：第三份同类实现 `scripts/test_herdsman_models.go:173`（脚本件）仍在；未处理「正文先出现合法 JSON 噪声」；`util` 的「对象优先于纯标量数组」在 `[1,2] {"a":1}` 这类回复上与旧实现结果不同（`ExpectArray` 下旧成功 / 新重试）。

---

## 二、25 条 P0 的当前状态（只读复核）

复核线（HEAD `3d0210d1`，只读、不跑测试）逐条读**当前**代码后产出 `round-1-p0-status.md`；主代理另亲核撤销/更正了其中 3 条判据。

| 口径 | 数量 |
|---|---|
| 表内行数 | 25 行（其中 #19/#20/#21 与 #1/#2/#8 是同一条） |
| **独立 P0** | **22 条** |
| 本轮已修 | 7 条独立（#1/#2 配置、#3 收件箱、#15/#16 进度契约、#18 GenUI、#22 ExtractJSON）＋ 非编号内的 bookimport 副本 |
| 部分修 | 1 条（#5 逐场景协程：登记已补、WG 仍缺） |
| 撤销 | 1 条（#17 前端全选：`setAllSelected`/`clearSelection` 定义在同文件 1126-1131，声明提升，原判据不成立；降级 P2「缺表头全选用例」） |
| 更正 | 方向 1 条（#21 真实方向是「显式 0 → 缺键」）；计数 1 条（#22 生产调用点 **21** 处，非 25） |
| **仍开放** | **10 条独立**：#4/#5 后台链登记纪律 · #6/#7 检索与知识库吞错/全局索引 · #8 崩溃面 · #9/#10/#11 三套编排与并发闸 · #12/#13 微信语音并发 · #14 绘梦单槽进度 · #23/#24/#25 三类复发型缺陷无守卫 |

**复核线给出的两条「差一行就闭环」高性价比项**：

1. #5 `internal/app/scene_cards_handler.go:218` 前补 `a.chapterGenWG.Add(1)` + 协程首行 `defer a.chapterGenWG.Done()`；
2. #4 `internal/app/converge_handler.go:177` 改走既有 `registerChapterGen`（并入取消登记表 + WG）。

> 完整逐条证据（当前行号 + 修复后新位置）见 [round-1-p0-status.md](round-1-p0-status.md)。


| 线 | 任务 | 状态 |
|---|---|---|
| 线1 | 配置保存丢段（P0-1/2）：`RenderTOMLPreserving` 段级 + **键级**保留（`[agent] effort`、`[tools] compact` 这类*已知段内部*未渲染键同样会丢——主代理用真实配置键对账后追加的要求） | 进行中（工作树内在途） |
| 线7 | `internal/bookimport/reconstruct.go` 的同类 `ExtractJSON` 副本 | 进行中 |
| 线V | 25 条 P0 在**当前工作树**是否仍成立（只读复核）→ `round-1-p0-status.md` | 进行中 |

## 三、额外的触发面发现（主代理亲核）

`internal/app/gaea_ui_extra.go:147 gaeaApplyCfg` = **设置面板保存通道**（`mutate → gaeaConfig.Save → gaeaRebuildLocked`），而该面板自身就写 `cfg.Agent.Effort/SubagentEffort`（186-191 行）。修复前 `Save → SaveTo → RenderTOML`（有损整文件重写），因此：

- **不是只有点「始终允许」才丢**：**保存一次设置**即触发整文件重写；
- 面板刚填的 `effort`/`subagent_effort` **直接落不了盘**（而 `boot.go:191/302` 真的会用它们设置 provider 的 effort 参数）；`[tools] compact`（`boot.go:454` 消费）同理。

**真实数据冒烟（修复前，用你 `%APPDATA%\gaea\config.toml` 的副本跑生产同源链 `Load → AddPermissionRuleForSpace → Save`）**：

| 指标 | 修复前 |
|---|---|
| 段数 | 16 → **5** |
| 丢失的段 | `dream` `markdown_converter` `memory` `network` `retrieval` `search` `session` `skills` `space` `tasks` `vision`（**11 个**） |
| 字段漂移 | `Session.Space`：`work` → 空 |
| 探针规则写入 | 是（说明这次写入真的发生了） |
| 用户原件 | 未触碰（仍 5995 B / 2026-09-24；已在同目录留 `.bak-20261002-audit-pre-RenderToML-fix`） |

> 修复后需复跑同一探针，期望：16/16 段保留、字段零漂移，**并且** `[agent] effort`/`subagent_effort`/`[tools] compact` 这类段内键也保住（这正是线1 的键级保留要求）。

## 四、口径与局限

1. 本轮**只做 P0 止血**，未碰 P1/P2（三套编排合并、绑定面 322 零引用瘦身、上帝文件拆分、mock 契约机械化等仍挂池，见审计 README §5 第二/三批）。
2. 全部改动**不动绑定面**、不动生成物（`bindingNames.ts`/`bridges`/`wailsjs`）、不动版本号与 CHANGELOG（发版仪式归用户/并发会话）。
3. 本机无 gcc → `-race` 只能靠 Actions；并发/锁类改动（刀2、刀5）在 CI 侧仍需 race job 复核。
4. 并行线的编译瞬时态会互相影响（实测：`render_preserve.go` 在途时 `internal/app` 包 build failed）——**集成门必须在所有写者停止后跑**。
5. **集成门实测（收尾）**：本地快闸 `scripts/ci.ps1` 全程通过——version-drift / clean-tmp / `go build` / `go vet` / golangci（钉版 v2.14.0）/ `go test ./... -count=1` / 前端 eslint + build + 全量 vitest（**414 文件 / 3596 例**）/ E 系守卫 / check-docs，脚本自身以 `CI OK` 收尾。外层管道返回的 exit 1 来自 node 的 stderr 警告被 PowerShell 记为 `NativeCommandError`（本仓在册的 stderr 陷阱，见 `ci.ps1` 头部注释），故主代理**直接重跑**了两项关键闸复核：`go test ./... -count=1` → exit 0、0 条 FAIL；`golangci-lint run ./...` → **0 issues**。
6. **门禁抓到的问题（教训：子代理自测绿 ≠ 接门禁绿）**：首轮快闸在 golangci 挂掉——线1 新文件 `render_preserve.go` 留了 2 个**未被使用**的辅助函数（`blockStringValue`、`keyLine.strValue`）与 1 个只写不读的结构字段 `val`；主代理清理后重跑才全绿。**作业书应加一条「无用代码自检」**（新建文件里不得留只被自己引用的辅助函数/只写字段），否则每轮都会有这样的收尾返工。
7. **本轮不做绑定面/生成物/版本**：未改 `bindingNames.ts`、`bridge/**`、`wailsjs/**`、`frontend/**`（线5 实测前端无需改动）、未抬版本、未动 CHANGELOG 与 AGENTS 速览——发版仪式归用户；`scripts/check-primitives.mjs` 仍是 warn 档，接 CI 前需拍板验收线。
