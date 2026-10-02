# 25 条 P0 的当前状态复核（2026-10-02 第二轮）

> 快照：HEAD=`3d0210d1 fix: 删除 v4.453 库侧主角关系测试——随 v4.454 回退（漏提交致 HEAD 编译失败）`（2026-10-02 11:41:11 +0800）；复核时间=2026-10-02 12:28（+08:00）；口径=读当前工作树源码，不跑测试/构建。
>
> 编号口径：本表沿用 `README.md` §3 的 25 行序号（即任务书列出的 P0#1..#25），并在「源 id」列给出 `machine/findings_all.json` 里的 fid，便于与分册/JSON 对账（README 的 #15/#16 与 JSON 的 FE7-01/FE7-02 顺序相反，源 id 列即为此而设）。
>
> 只读声明：本轮除本文件外未修改任何文件；未执行 `go test` / `go build` / `wails build` / `golangci-lint`。

## 总览

| # | 源 id | P0 标题 | 状态 | 当前证据（file:line + ≤2 行摘录或「该行已不存在」） | 备注 |
|---|---|---|---|---|---|
| 1 | GA5-01 | `RenderTOML` 只渲染 9 段，保存即丢 14 个配置段 | 本轮已修 | `internal/gaea/config/render_preserve.go:48` `func RenderTOMLPreserving(c *Config, existing string) string {`；`config/edit.go:294-295` `existing := readFileOrEmpty(path)` → `AtomicWrite(path, []byte(RenderTOMLPreserving(c, existing)), 0o644)` | 落盘全走保留渲染；`RenderTOML` 仅剩测试与保留器内部调用（全仓非测试调用点已不存在） |
| 2 | GA5-02 | 一次 `persist_allow` 审批即触发配置段丢失落盘 | 本轮已修 | `boot.go:623-631` 仍是 `config.Load()` → `AddPermissionRuleForSpace` → `pcfg.Save()`，但 `config/edit.go:301-306` 的 `Save()` → `SaveTo` → 保留渲染；`config/config.go:840` `WriteFile` 同改 | 链路未动、落盘口径已换；待建「审批后重读断言段文本仍在」的绑定断言 |
| 3 | AP4-02 | 收件箱读改写无锁且有跨会话写者 | 本轮已修 | `gaea_task_inbox.go:98` `var taskInboxMu sync.RWMutex`；`taskInboxSave:191-192` 写锁、`GaeaTaskInboxSetStatus:300`、`GaeaTaskInboxDelete:332`、`GaeaTaskInboxList:267` RLock | 生产代码里已无游离的裸 `load/save` 调用点（其余命中全在 `_test.go`）；进程内口径已封，跨进程未封 |
| 4 | AP1-01 | 收敛闭环 goroutine 无登记无取消无 WG | 仍开放 | `converge_handler.go:177` `go func() {`（函数内无 `registerChapterGen`/`chapterGenWG`/`ctx.Err()`）；同族 `create_chapter_handler.go:237-239` 有 `chapterGenWG.Add(1)` | 一行最小修法：`177` 前接 `key := chapterGenKey(chapterNum,""); ctx,cancel,err := a.registerChapterGen(key,chapterNum,"")`，`177` 后补 `defer a.unregisterChapterGen(key,cancel)`，循环内以 `ctx.Err()` 判停，外层 `a.chapterGenWG.Add(1)`+`defer Done()` |
| 5 | AP1-02 | 逐场景生成协程漏计 WG，`waitGensDone` 提前返回 | 部分修（登记已补，WG 仍缺） | 已补：`scene_cards_handler.go:210` `ctx, cancel, err := a.registerChapterGen(...)`、`219` `defer a.unregisterChapterGen(key, cancel)`；仍缺：`218` `go func() {` 之前无 `a.chapterGenWG.Add(1)`，函数内无 `Done()`（全包 `chapterGenWG` 命中只有 `app.go:105` 声明与 `create_chapter_handler.go:237/239/752/754/761/763`） | 一行最小修法：`218` 前加 `a.chapterGenWG.Add(1)`，`219` 后再加 `defer a.chapterGenWG.Done()`（与 create_chapter 三行同款） |
| 6 | GA3-01 | knowledge 后端吞 DB 错误，查重失效致重复入库 | 仍开放 | `knowledge/sqlite.go:77-82` `func (b *sqliteBackend) List() []EntrySummary {` / `if err != nil { return nil }`；接口仍是无 error 单返回值（`store.go:20`），消费点 `knowledgeimport.go:163` `for _, s := range store.List()` | 一行最小修法：`List` 改 `([]EntrySummary, error)` 并在 `err` 分支 `return nil, err`，`MatchRows`（`knowledgeimport.go:159`）改在其调用点把失败上抛中止预览 |
| 7 | GA3-02 | 进程级全局 `memory_search` 索引被多控制器覆盖 | 仍开放 | `tool/builtin/memory_search.go:21` `var memorySearchIndex *memory.SearchIndex`；写点 `boot/sysprompt.go:109` 与 `control/controller_memory.go:285` `builtin.SetMemorySearchIndex(next.Search)` | `controller_memory.go:282` 的 gen 守卫只解决单控制器回退，跨控制器仍是「最后写者赢」；最小修法：索引挂 Controller 并经 `context` 传给 `Execute` |
| 8 | GA5-03 | trajectory 折叠 `assistantRecord` 返回 nil 被解引用 | 仍开放 | `trajectory/fold.go:332-333` `r := f.assistantRecord(e)` / `r.Assistant.Usage = &Usage{`；被调方 `fold.go:441-443` `if f.cur == nil { return nil }` | 一行最小修法：`332` 后加 `if r == nil { return }`（或按 `applyCompaction:359-363` 的既有分流把记录落到 `f.between`）；绑定入口统一 recover 中间件仍缺 |
| 9 | AP4-01 | 三套并行编排各带队列与状态机 | 仍开放 | `gaea_dag.go:514-528` `dagStart` 自建 `dagCancels` + goroutine；`gaea_task_inbox.go` 仍是独立状态文件 + 自建状态机；`gaea/tasks/tasks.go` 未动 | 属「需先拍板再动刀」的结构项，非一行可修；同族判定见 06 分册刀8 |
| 10 | AP4-04 | DAG 执行器自建并发模型绕过任务闸口 | 仍开放 | `gaea_dag.go:545-548` `for _, node := range wave {` / `wg.Add(1)` / `go func() {`；`596` `_, ref, err := runner(gaeaAgent.WithSpace(ctx, ...), prompt, emit)` | 全包无按空间的子代理信号量（`local_concurrency` 仅出现在注释：`app.go:256`、`gaea_tasks.go:64`、`herdsman_lifecycle.go:22`）；最小修法：`go func()` 前先取一个来自 config 的信号量 |
| 11 | AP4-03 | 成本导入直写 UPSERT 与 `Store.Save` 双写路径 | 仍开放 | `gaea_cost_import.go:252-264` `const costEntryUpsertSQL = \`\nINSERT INTO cost_entries(...)`（第二份 SQL 常量仍在）；`266-275` `marshalCostTags` 副本仍在 | 最小修法：cost 包导出 `SaveTx(tx, e)` 复用唯一 SQL 常量，app 侧删常量与 `marshalCostTags` |
| 12 | AP6-01 | `voice.Manager` 回调字段无锁写读 | 仍开放 | `voice/voice_manager.go:212-214` `func (m *Manager) SetWhisperChatFn(fn WhisperChatFn) {` / `m.whisperChatFn = fn`；`218-220` `SetTTSSynthesizeFn` 同样裸写 | 紧邻的 `ApplyConfig:225-227`/`GetConfig:232-234` 都已走 `m.mu`，仅回调字段未纳入；最小修法：两个 setter 加 `m.mu.Lock()`，读点锁内快照 |
| 13 | AP6-02 | `whisperSessions` 跨锁窗口双建 orchestrator 后互相覆盖 | 仍开放 | `whisper_handler.go:51-56` `whisperSessionsMu.RLock()` / `if orch, ok := whisperSessions[sessionID]; ok { ... }` / `whisperSessionsMu.RUnlock()`；`99` 才 `whisper.NewOrchestrator`，`110-112` 才回写 | 查表→建实例→回写三段跨锁；最小修法：`51` 改 `Lock` 并 double-check，或改 per-key `sync.Once`/singleflight |
| 14 | AP7-01 | 生成链无单飞，进度与取消是全局单槽会串台 | 仍开放 | `image_handler.go:97-101` `a.imageGenID++` / `ctx, cancel := context.WithCancel(parent)` / `a.imageGenCancel = cancel`（`99` 覆盖式写入，无 busy 拒绝）；进度仍是单槽：`174-185` `updateComfyTaskProgress` 写 `a.comfyTaskStatus/Percent/Node`、`197` `GetComfyUITaskProgress()` 无参数 | `92` 的 `imageGenMu` 只护字段不护「并发拒绝」；最小修法：`92` 锁内加 `if a.imageGenRunning { return nil,nil,0 }` 式拒绝并返回中文 busy 错误 |
| 15 | FE7-01 | 多基线 `baselines` 被 Go 保存链路静默擦除 | 本轮已修 | `internal/schedule/types.go:155` `Baselines []Baseline \`json:"baselines,omitempty"\``（`148-154` 注明「此前 Go 无此字段…静默删掉用户第 2/3 条基线」）；回归锁 `internal/app/gaea_schedule_file_test.go:487-511` `TestGaeaScheduleRoundTripKeepsBaselines` 断言 `len(Baselines)==3` + 槽位顺序 + `savedAt/rows` | 前端 `frontend/src/schedule/types.ts:115` 的 `baselines?: SchedBaseline[]` 与 Go 侧已对齐 |
| 16 | FE7-02 | `assignment` 数值字段 TS 可选 / Go 零值 | 本轮已修 | `internal/schedule/types.go:122` `Units *float64`、`125` `Quantity *float64`、`127` `Amount *float64`（`106-110` 注明「审计 FE7-02」三态口径）；回归锁 `gaea_schedule_file_test.go:518-533` `TestGaeaScheduleRoundTripKeepsAssignmentTriState` | 同文件 `112-115` 如实登记未动残留：`Resource` 数值字段与 `Task.ManualStart/FixedCost` 仍是裸值+`omitempty`（其中 `MaxUnits` 有真实分叉），本 P0 范围内的 Assignment 三字段已修 |
| 17 | FE2-01 | 表格视图全选调未定义函数，点击抛错且无反馈 | 无法判定（与当前源码不符） | `frontend/src/gaea/components/CostLibraryView.tsx:1086-1087` 调用 `setAllSelected()` / `clearSelection()`，而这两个**组件内函数声明存在于 `1126-1131`**（`function setAllSelected() { onSelectAll(new Set(rows.map((e) => e.name))) }`）；文件自 `c48b834e`（v4.386.0）后无改动 | 缺的证据：可跑 `tsc`/vitest 的环境（本轮禁跑构建）。静态读码看不到「引用未定义函数」，故不判「仍开放」；同时确无「点表头复选框→全部行选中」的用例（`onSelectAll` 仅 `544`/`822`/`1060` 三处，无测试断言） |
| 18 | FE6-01 | `genui_validate` 把 `diffs`/`series` 当节点，合法规格被判非法 | 本轮已修 | `genui/validate.go:120-121` `v.checkData(path, typ, obj)` / `v.walkChildren(path, typ, obj, depth)`；`walkChildren:128-159` 只对 `row/col/grid/card/list.items` 与 `tabs`/`accordion` 行内 `items` 递归；`checkDataRows(..., "diffs", obj["diffs"], "path","newText")`（`239`） | 用例已入库：`validate_test.go:64` `TestValidateSpec_DiffRowsAreDataNotNodes`、`:81` `TestValidateSpec_ChartSeriesAreDataNotNodes`、`:112` 其它数据数组 |
| 19 | GA5-01(已上表 1) | `RenderTOML` 只渲染 9 段 | 见 #1 | 同 #1 | README #19 与 #1 是同一条（`render.go:13-225`） |
| 20 | GA5-02(已上表 2) | `persist_allow` 审批触发丢段落盘 | 见 #2 | 同 #2 | README #20 与 #2 是同一条（`boot.go:623-632`） |
| 21 | GA5-03(已上表 8) | trajectory `assistantRecord` nil 解引用 | 见 #8 | 同 #8 | README #21 与 #9 是同一条（`fold.go:332-339`） |
| 22 | IN2-01 | `ExtractJSON` 取最后一个右花括号，多对象即解析失败 | 本轮已修 | `util/util.go:92-100` `func ExtractJSON(s string) string {` → `extractFromJSONFence` / `extractBalancedJSON`；`157-187` `scanBalanced` 深度配平 + 字符串内 `escaped` 状态机，`181-183` `depth <= 0` 即返回 | 旧实现（首个 `{` → 最后 `}`）在当前文件中已不存在；同族副本 `internal/bookimport/reconstruct.go:227-231` 已改为 `json.Unmarshal([]byte(util.ExtractJSON(resp)), &v)` 转调（该文件在工作树中为未提交改动 + 新增 `reconstruct_extractjson_test.go`）→ 该副本**本轮已修** |
| 23 | X1-01 | heredoc 转义坑在册复发 5 次，无任何守卫 | 仍开放 | `CHANGELOG.md:8,11,38,50,93` 五处踩坑记录（例：`:11` 「Bash heredoc 把 `\n` 降级真实换行进 Go 字符串（在册坑再踩）」、`:8` 「第三次踩」、`:38` 「四犯」）；新增守卫 `scripts/check-primitives.mjs:389-395` 三项检测=截断原语/原子写/前端旁路，**不含 heredoc**，且自述「warn 档 exit 0（未接 CI，接门禁前先拍板验收线）」 | 一行最小修法：在 `check-primitives.mjs` 增第 4 项检测「新增 `.go` 内容含 `<</EOF` / python 写文件 heredoc 形态即报」，并以 `--strict` 接 CI |
| 24 | X1-02 | Windows unlinkat/TempDir 竞态三度复发打挂 CI | 仍开放 | 根因仍在：`scene_cards_handler.go:218` 的协程未计 `chapterGenWG`（见 #5），而 `create_chapter_cancel_test.go:101-122` 的 `waitGensDone` 明确「只等登记表会在 Windows 上与 `t.TempDir()` 清理竞态（unlinkat directory not empty，v4.233 在册 flaky 的根因），故最终以 `chapterGenWG` 为准」；审计建议的 `newTestAppWithTempDir(t)` helper 当前**不存在**（`waitGensDone` 仅 `create_chapter_cancel_test.go:105` 定义，仍靠 13 处调用点手工调用） | 一行最小修法：先把 #5 的 `chapterGenWG.Add(1)` 补上，再抽 `newTestAppWithTempDir(t)`（内部 `t.Cleanup` 先 `waitGensDone` 再删目录） |
| 25 | X1-03 | 三重 `*core` 嵌入导致 nil 嵌入 panic 三犯 | 仍开放 | `app/app.go:272-277` 现为**五重**嵌入：`*core` / `*writingState` / `*mediaState` / `*whisperState` / `*officeState`；`app.go:80-81,134-135,176-177,228-229` 四个 state 各自再嵌 `*core`；唯一装配点 `app.go:308` `a := &App{core: c}` | 仍是可裸构造的导出零值：`internal/app` 全包测试大量 `a := &App{}`（如 `gaea_dag_test.go:160,214,245,277,305`、`gaea_cost_projects_test.go:39`、`gaea_export_test.go:31`），`bindingNames:14` 亦 `&App{}`；无 fail-fast 断言。最小修法：加 `func (a *App) assertAssembled()` 并在绑定入口首行调用，或收敛到唯一构造函数 |

**结论：仍开放 15 条 / 已修 10 条（其中「本轮已修」7 条：源 id GA5-01/GA5-02/AP4-02/FE7-01/FE7-02/FE6-01/IN2-01；另 3 条为重复行 #19/#20/#21 —— 真实独立条目计 **25 行 = 22 条独立 P0 + 3 条重复行**，「本轮已修 7 条 / 部分修 1 条 / 仍开放 10 条 / 无法判定 1 条」（另有 bookimport 同族副本 1 条本轮已修，非 25 条编号内）。**

---

## 逐条

> 编号读法（重要）：小节标题里的「（源 X）」= `machine/findings_all.json` 的 fid，是**唯一权威对账键**——请用它在总览表的「源 id」列查行号。小节标题的数字是审计员当时的读序（1..13、15..25，无 14），与总览表行号在校验后对齐如下：§1–§10 分别=表行 1–10；**§11（源 AP6-01）=表行 12、§12（源 AP6-02）=表行 13**（表行 11=AP4-03 成本导入双写不在逐条正文内，其证据与最小修法已写进总览表第 11 行）；§13（源 AP7-01）=表行 14（README 行 14 与行 13 是同一条 finding 的两种写法，故本节同时承载表行 13/14）；§15（源 FE7-01）=表行 15、§16（源 FE7-02）=表行 16、§17（源 FE2-01）=表行 17、§18（源 FE6-01）=表行 18；§19/§20/§21 是重复行（表行 19/20/21），分别指回 §1/§2/§8，不重出正文；§22–§25=表行 22–25。合计逐条正文 22 节 = 25 行 − 3 条重复行 + 1 条「表行 13/14 同 finding 合并」。

### 1（源 GA5-01）`RenderTOML` 只渲染 9 段，保存即丢 14 个配置段 —— 本轮已修

- 保留器落地：`internal/gaea/config/render_preserve.go:48-76`。键入口 `rendererOwnedTopLevel`（`25-33`：agent/providers/tools/permissions/space_profiles/sandbox/plugins）与 `rendererOwnedScalars`（`36-39`：default_model/language）之外的全部顶层块/标量**逐字保留**并追加在渲染结果之后（`55-74`）。
- 落盘口全部改道：`config/edit.go:290-296` `SaveTo`、`config/config.go:840-842` `WriteFile` 均走 `RenderTOMLPreserving(c, readFileOrEmpty(path))`；`edit.go:301-306` `Save()` 走 `SaveTo`。
- 旧渲染器仍在（`render.go:20`），但**全仓非测试调用点只剩 `render_preserve.go:49` 一处**（其余 11 处命中全在 `_test.go`），即「直接拿有损渲染整文件重写」的路径已不存在。
- 回归锁：`render_preserve_test.go:66`（空 existing 时逐字节等价 `RenderTOML`）+ `:98` 注释所述「改一处再落盘 → 再 Load 逐字段不许变」用例。
- 残余（不属本条）：审批链路的「保存后重读断言段文本仍在」绑定断言未单独建（见 #2）。

### 2（源 GA5-02）一次 `persist_allow` 审批即触发配置段丢失落盘 —— 本轮已修

- `internal/gaea/boot/boot.go:623-632` 的回调体还是 `config.Load()` → `pcfg.AddPermissionRuleForSpace(space, "allow", rule)` → `return pcfg.Save()`（该行已不存在「直接 `RenderTOML`」的落盘）。
- 丢段的实际写点已换口径：`config/edit.go:294-295`（`SaveTo` 读旧文件 → 保留渲染 → `fileutil.AtomicWrite`），`config/config.go:840-842`（`WriteFile` 同）。因此本条 P0 的「一次审批即删掉 `[memory]/[dream]/…`」已不成立。
- 待补（审计原刀要求、当前仍缺）：审批后重读 gaea.toml 断言段文本仍在的**绑定断言**。

### 3（源 AP4-02）收件箱读改写无锁且有跨会话写者 —— 本轮已修

- `internal/app/gaea_task_inbox.go:98` `var taskInboxMu sync.RWMutex`，`75-97` 注释明确「saveTaskInbox 的原子性只解决半截文件，解决不了丢更新——丢更新只能把 load…save 整段放进同一临界区」。
- 覆盖点（读现场）：`taskInboxSave:191-192`（新建 + 编辑 + MaxTasks 判定同锁内）、`GaeaTaskInboxList:267-269`（RLock）、`GaeaTaskInboxSetStatus:300-301`、`GaeaTaskInboxDelete:332-333`。
- 裸原语的调用纪律已写到函数注释：`loadTaskInbox:103-105`、`saveTaskInbox:120-122`；全仓 `loadTaskInbox(`/`saveTaskInbox(` 的生产命中（`195/230/252/268/303/316/335/339`）全部落在上述锁区间内，其余命中在 `gaea_task_inbox_test.go`。
- 已如实登记的未解风险（`95-97` 注释）：跨进程（双开 app）仍是文件级竞态，属另一刀。

### 4（源 AP1-01）收敛闭环 goroutine 无登记无取消无 WG —— 仍开放

- `internal/app/converge_handler.go:177` `go func() {`：协程体内（`178-232`）无 `registerChapterGen`、无 `a.chapterGenWG.Add/Done`、无 `ctx.Err()` 判停；闭包仍持有函数入口 `157` 的 `pm := a.getPM()`，整轮循环经 `convergeRound`/`saveConvergeVersion` 直写章节文件。
- 同族对照仍在：`create_chapter_handler.go:237-239` `a.chapterGenWG.Add(1)` / `defer a.chapterGenWG.Done()`。
- **一行最小修法**：`177` 前接 `key := chapterGenKey(chapterNum, "")` + `ctx, cancel, err := a.registerChapterGen(key, chapterNum, "")`（err 非空即返回），`177` 后首行 `defer a.unregisterChapterGen(key, cancel)`，外层 `a.chapterGenWG.Add(1)` + `defer a.chapterGenWG.Done()`，`for r := 1; ...` 循环首行加 `if ctx.Err() != nil { return }`。

### 5（源 AP1-02）逐场景生成协程漏计 WG，`waitGensDone` 提前返回 —— 部分修

- **已修的部分**（登记/取消/注销）：`internal/app/scene_cards_handler.go:209-213` `key := chapterGenKey(chapterNum, "")` / `ctx, cancel, err := a.registerChapterGen(key, chapterNum, "")`；`218-219` `go func() {` / `defer a.unregisterChapterGen(key, cancel)`；协程内 `222` 有 `if ctx.Err() != nil`。
- **仍开放的部分**（WG 记账）：`218` 之前无 `a.chapterGenWG.Add(1)`，协程内无 `defer a.chapterGenWG.Done()`。全包 `chapterGenWG` 命中：`app.go:105`（声明）、`create_chapter_handler.go:237/239/752/754/761/763`——`scene_cards_handler.go` 零命中。
- 影响面按 `create_chapter_cancel_test.go:101-104` 的注释仍在：`waitGensDone` 的两级等待以 `chapterGenWG` 为准；未计账的逐场景协程使「WG 归零」不再等于「本章生成链全部收尾」。
- **一行最小修法**：`218` 前插 `a.chapterGenWG.Add(1)`，`219` 后插 `defer a.chapterGenWG.Done()`。

### 6（源 AP4-01）三套并行编排各带队列与状态机 —— 仍开放

- DAG 侧自建在途登记与取消：`internal/app/gaea_dag.go:514-528`（`ctx, cancel := context.WithCancel(context.Background())` / `ga.dagCancels.Store(id, cancel)` / 自建 goroutine + `Delete(id)`）；`444-445` 查取消、`151` 判在途。
- 收件箱侧仍是独立状态文件 + 自建状态机：`gaea_task_inbox.go:106/123`（JSON 全量覆盖模型）与 `taskinbox.Sort/FilterBySpace` 口径。
- 任务调度器未动：`internal/gaea/tasks/tasks.go`（本文件未在审计后改动）。
- 本条是「先统一任务身份与状态机、再谈合并入口」的结构决策（README §5 已把它与 06 分册刀8 绑定），没有一行修法；本轮不给修法以避免越权拍板。

### 8（源 GA5-03）trajectory 折叠 `assistantRecord` 返回 nil 被解引用 —— 仍开放

- 调用点 `internal/gaea/trajectory/fold.go:332-339`：`r := f.assistantRecord(e)` 后直接 `r.Assistant.Usage = &Usage{...}`。
- 被调方 `fold.go:440-443`：`if f.cur == nil { return nil }`——「首条即 usage 增量」时 `f.cur == nil`（`applyHeader` 尚未建块），返回 nil。
- 对照既有分流：`applyCompaction:359-363` 对同种情况走 `f.between = append(f.between, rec)`，说明分流惯例已在包内。
- **一行最小修法**：`332` 之后加 `if r == nil { return }`（若要保留信息，则改为落 `f.between`）；Wails 绑定入口统一 `recover` 中间件（审计同条要求）当前仍未见（`app.go:477/551/591` 是逐链 recover，非绑定入口统一件）。

### 7（源 GA3-02）进程级全局 `memory_search` 索引被多控制器覆盖 —— 仍开放

- `internal/gaea/tool/builtin/memory_search.go:19-21`：`func SetMemorySearchIndex(idx *memory.SearchIndex) { memorySearchIndex = idx }` / `var memorySearchIndex *memory.SearchIndex`；读点 `:52` `idx := memorySearchIndex`。
- 两个写点都在册：`internal/gaea/boot/sysprompt.go:109` `builtin.SetMemorySearchIndex(mem.Search)`、`internal/gaea/control/controller_memory.go:285` `builtin.SetMemorySearchIndex(next.Search)`（锁内换入）。
- `controller_memory.go:275-287` 的 gen 守卫只保证「同控制器内的旧 Load 不回退」，**不改变跨控制器（work/play 两个 Controller）互相覆盖包级变量的语义**。
- **一行最小修法**：把 `*memory.SearchIndex` 收进 `Controller` 实例并随 `context` 传入 `Execute`（`memory_search.go:51` 的 `Execute(_ context.Context, ...)` 第一个参数现被丢弃，正是接入点）。

### 6（源 GA3-01）knowledge 后端吞 DB 错误，查重失效致重复入库 —— 仍开放

- `internal/gaea/knowledge/sqlite.go:77-82`：`func (b *sqliteBackend) List() []EntrySummary {` / `rows, err := b.db.Query(...)` / `if err != nil { return nil }`——错误被吞成空切片。
- 表层同构：`knowledge/store.go:20` `List() []EntrySummary`（接口无 error）、`89-93` Store 层透传、`150-158` `fileBackend.List` 走 `ReadAll()`（`168-169` 亦吞 glob 错）。
- 消费点：`internal/gaea/knowledgeimport/knowledgeimport.go:159-169` `MatchRows` 用 `store.List()` 建 `byTitle`；库读失败 → 空表 → 每行 `MatchNote = "新增"`（`185`），即「重复入库且提示不覆盖」的静默路径成立。
- **一行最小修法**：`List` 改 `([]EntrySummary, error)` 接口 + `sqlite.go:80` `return nil, err`，`MatchRows` 改 `([]Row, error)` 并由 `internal/app/gaea_knowledge_import.go:177` 上抛中止预览。

### 10（源 AP4-04）DAG 执行器自建并发模型绕过任务闸口 —— 仍开放

- `internal/app/gaea_dag.go:539-548`：波内 `for _, node := range wave { node := node; wg.Add(1); go func() {`——一波 N 节点全量并发，`wg` 只做收尾等待，不做准入。
- 执行体 `596`：`_, ref, err := runner(gaeaAgent.WithSpace(ctx, gaeaSessionSpace()), prompt, emit)`——直连子代理执行器，不经 `internal/gaea/tasks` 的队列/并发限制。
- 全仓 `local_concurrency` 仅出现在注释（`app.go:256`、`gaea_tasks.go:64`、`herdsman_lifecycle.go:22`、`gaea_schedule.go:7`），本包无按空间信号量。
- **一行最小修法**（按审计原刀）：在 `go func()`（`548`）前先取一个来自 config 的信号量（`sem <- struct{}{}` + `defer func(){ <-sem }()`），信号量上限与 tasks 的 `max_concurrent` 同源。


### 11（源 AP6-01）`voice.Manager` 回调字段无锁写读 —— 仍开放

- `internal/voice/voice_manager.go:211-214`：`func (m *Manager) SetWhisperChatFn(fn WhisperChatFn) {` / `m.whisperChatFn = fn`（无 `m.mu`）。
- `:216-220`：`SetTTSSynthesizeFn` 同款裸写。第三个回调字段（`SetTTSSynthesizeFn` 之外的 ASR/Whisper 回调）按注释体例应同批处理。
- 对照：同文件 `ApplyConfig:225-227` / `GetConfig:232-234` / `GetState:238-239` 均已走 `m.mu`；`degradeRealtime:198-203` 也在锁内改 `rtActive`——即「回调字段」是本类型里唯一漏网的一组。
- **一行最小修法**：`212-214` 与 `218-220` 各加 `m.mu.Lock()` / `m.mu.Unlock()`，并在读点（`handleReply`/`speak`/`WhisperReady`）锁内取一次快照。

### 12（源 AP6-02）`whisperSessions` 跨锁窗口双建 orchestrator 后互相覆盖 —— 仍开放

- `internal/app/whisper_handler.go:51-56`：`whisperSessionsMu.RLock()` → `if orch, ok := whisperSessions[sessionID]; ok { ... }` → `whisperSessionsMu.RUnlock()`（读锁只盖查表）。
- 建实例在锁外：`:99` `orch := whisper.NewOrchestrator(sessionID, *preset)`（`100-108` 还接线 `DataRoot`/`FTSSearch`）；回写在 `:110-112` `whisperSessionsMu.Lock()` / `whisperSessions[sessionID] = orch` / `Unlock()`（无 double-check，后建者覆盖先建者）。
- 三入口（GUI/微信/语音）同抢人格时，两个 goroutine 可各建一个 orch，先入表者被静默覆盖；`114-116` 的 `restoreWhisperState` 也只作用于自己那份。
- **一行最小修法**：`51` 的 `RLock` 改 `Lock`，查到即 `Unlock` + `return`（写锁内建实例，或把 `99-112` 交给 `sync.Once`/`singleflight` per sessionID）。

### 13（源 AP7-01）生成链无单飞，进度与取消是全局单槽会串台 —— 仍开放

- 单飞缺失：`internal/app/image_handler.go:91-102` `beginImageGen` 在 `imageGenMu` 内**无条件** `a.imageGenID++`（`97`）、`a.imageGenCancel = cancel`（`99`）、`a.imageGenRunning = true`（`100`），没有「已有在跑则拒绝」的分支；并发调用即互相覆盖取消句柄。
- 进度仍是全局单槽：`174-185` `updateComfyTaskProgress` 写 `a.comfyTaskStatus/Elapsed/Percent/Node` 四个字段；`187-194` `clearComfyTaskProgress` 一次清空；`197-205` `GetComfyUITaskProgress()` **无 genID 参数**。
- 收尾按 `imageGenID == id` 判轮次（`104-115`）与取消（`125-134`），说明「轮次」概念已在，但缺「拒绝并发」与「按任务分槽的进度」。
- **一行最小修法**：`beginImageGen` 锁内加 `if a.imageGenRunning { return nil, nil, 0 }`（调用方转中文 busy 错误）；进度改 `map[uint64]comfyTaskProgress` 并给 `GetComfyUITaskProgress(genID uint64)` 加参数。

### 15（源 FE7-01）多基线 `baselines` 被 Go 保存链路静默擦除 —— 本轮已修

- Go 侧字段已补：`internal/schedule/types.go:155` `Baselines []Baseline \`json:"baselines,omitempty"\``；`148-154` 注释写明「此前 Go 无此字段，而 `GaeaScheduleSave` 是 Unmarshal→MarshalIndent 整量覆盖写盘，等于每次防抖自动保存都静默删掉用户第 2/3 条基线」，并声明本字段为**透传承载**（顺序/FIFO 口径仍在前端 `baseline.ts`）。
- 前端契约侧：`frontend/src/schedule/types.ts:115` `baselines?: SchedBaseline[]`（与 Go 逐字对齐）。
- 回归锁（本轮新增，属未提交改动）：`internal/app/gaea_schedule_file_test.go:487-511` `TestGaeaScheduleRoundTripKeepsBaselines`——断言 3 条、槽位顺序 `["开工版","签证A","签证B"]` 逐位相等、`savedAt`/`rows` 随行、单数 `baseline` 活跃指针并存。

### 16（源 FE7-02）`assignment` 数值字段 TS 可选 / Go 零值 —— 本轮已修

- `internal/schedule/types.go:116-128` `Assignment`：`Units *float64`（`122`）、`Quantity *float64`（`125`）、`Amount *float64`（`127`），`omitempty` 与「nil=不落键，非 nil 的 0=落 0」口径写在 `106-110`，且注释直接点名「审计 FE7-02」。
- 前端同为可选：`frontend/src/schedule/types.ts:86-90`（`units?/quantity?/amount?: number`）。
- 回归锁：`gaea_schedule_file_test.go:518-533` `TestGaeaScheduleRoundTripKeepsAssignmentTriState`——缺 `units` 时断言**不落键**，显式 `quantity=0` 时断言键存在且值为 `0`。
- 同文件如实登记的未动残留（`112-115`）：`Resource` 数值字段与 `Task.ManualStart/FixedCost` 仍是裸值 + `omitempty`，其中只有 `MaxUnits` 存在真实分叉（前端 `res.maxUnits ?? 1`）。本条 P0 的三个字段已修，残留属新登记项。

### 17（源 FE2-01）表格视图全选调未定义函数 —— 无法判定（与当前源码不符）

- 现场：`frontend/src/gaea/components/CostLibraryView.tsx:1081-1090` 表头复选框 `onChange` 调 `setAllSelected()`（`1086`）/ `clearSelection()`（`1087`）。
- 这两个标识符**在同一组件函数体内有声明**：`1126-1128` `function setAllSelected() { onSelectAll(new Set(rows.map((e) => e.name))); }`、`1129-1131` `function clearSelection() { onSelectAll(new Set()); }`；`onSelectAll` 来自 `TableViewProps`（`822`，实现方传 `setSelected`，`544`）。函数声明提升，`1086/1087` 的调用在静态语义上**有绑定**。
- git 侧：该文件最后改动为 `c48b834e`（2026-09-22，v4.386.0），工作树对该文件**无未提交改动**，即审计后未被并发线改过——审计当时若真读到 `ReferenceError`，只能是读到了不完整的文件视图。
- **缺哪条证据**：可执行的类型检查/单测（`tsc` / vitest）——本轮纪律禁跑构建，故不能 100% 排除其它编译级问题。另确认：确无「点表头复选框 → 全部行进 selected」的用例（用户要求的第一步已满足「补实现或删功能」的现状，第二步用例仍缺）。
- 结论写法：判 **无法判定**（不是「仍开放」——静态证据不支持未定义函数结论；也不是「已修」——本轮无人改过该文件，且缺可执行证据）。

### 19（源 GA5-01，同 #1）`RenderTOML` 只渲染 9 段 —— 见 #1

README §3 第 19 行与第 1 行为同一条（位置均为 `internal/gaea/config/render.go:13-225`）。结论=本轮已修，证据同 #1。

### 20（源 GA5-02，同 #2）`persist_allow` 审批触发丢段落盘 —— 见 #2

README §3 第 20 行与第 2 行为同一条（位置均为 `internal/gaea/boot/boot.go:623-632`）。结论=本轮已修，证据同 #2。

### 21（源 GA5-03，同 #8）trajectory `assistantRecord` nil 解引用 —— 见 #8

README §3 第 21 行与第 8 行为同一条（位置均为 `internal/gaea/trajectory/fold.go:332-339`，当前行号未漂移）。结论=仍开放，证据与最小修法同 #8。

### 22（源 IN2-01）`ExtractJSON` 取最后一个右花括号 —— 本轮已修

- `internal/util/util.go:92-100`：`func ExtractJSON(s string) string {` → `extractFromJSONFence`（```json 围栏优先）→ `extractBalancedJSON`（`105-127`，从左到右对每个 `{`/`[` 起点的候选做配平 + `json.Valid`）→ 找不到则原样返回（向后兼容）。
- 配平实现 `scanBalanced:157-187`：`depth` + `inString` + `escaped` 状态机，`181-183` `depth <= 0` 即返回 `s[start:i+1]`——字符串内的 `{`/`}`/`\"` 不参与深度。
- 旧算法（首个 `{` 到最后一个 `}`）在当前文件中已不存在。
- 同族副本本轮也已收口：`internal/bookimport/reconstruct.go:227-231` `func ExtractJSON(resp string) (any, error) { var v any; if err := json.Unmarshal([]byte(util.ExtractJSON(resp)), &v); err != nil {...}`，注释 `221-226` 明确「本包不再自带第二份实现」；工作树中该文件为未提交改动并新增 `internal/bookimport/reconstruct_extractjson_test.go`。**（非 25 条编号内，附带登记）**

### 23（源 X1-01）heredoc 转义坑在册复发 5 次，无任何守卫 —— 仍开放

- 复发现场（`CHANGELOG.md` 当前行号）：`:8` 「Bash heredoc 反斜杠 n 降级真实换行（第三次踩）」、`:11` 「Bash heredoc 把 `\n` 降级真实换行进 Go 字符串（在册坑再踩）」、`:38` 「heredoc 反斜杠 n 降级在册坑四犯（Go 测试串三处断裂，chr(92) 修复）」、`:50`（「在册坑三犯」）、`:93`（「Bash 工具 heredoc 的 \n 会降级成真实换行」）。README 引用的 918/922/928 在当前文件中对应位置已漂移（现为 `:927/931/937` 的同类记录）。
- **守卫仍不存在**：新增的 `scripts/check-primitives.mjs`（未跟踪新文件）只检测三类——截断原语定义、`internal/app` 的 `os.CreateTemp`/`ioutil.TempFile`、前端 `a.download=`/`atob(`（见其 `38-41` 正则常量）；其自述为「warn 档，可复跑，暂不接 CI」（`389-395`，`--strict` 才 exit 1，且 CI 没有引用该脚本：`.github/workflows/ci.yml` 无 `check-primitives` 命中）。scripts/ 与 `.github/` 全目录检索 `heredoc` 零命中。
- **一行最小修法**：在 `check-primitives.mjs` 增第 4 个类别「新增 `.go`/`.ts` 内容含 heredoc 写文件形态」并在 CI 以 `--strict` 调用。

### 24（源 X1-02）Windows unlinkat/TempDir 竞态三度复发打挂 CI —— 仍开放

- 根因（Go 侧记账缺口）仍在：`internal/app/scene_cards_handler.go:218-219` 的逐场景协程只做 `registerChapterGen`/`unregisterChapterGen`，未计 `chapterGenWG`（详见 #5）；`waitGensDone` 的既有注释正是这条坑的自陈证据——`create_chapter_cancel_test.go:101-104`：「取消路径会先删登记，协程仍有『已生成部分落盘』尾步；只等登记表会在 Windows 上与 `t.TempDir()` 清理竞态（unlinkat directory not empty，v4.233 在册 flaky 的根因），故最终以 `chapterGenWG` 为准」。
- 审计建议的「测试基建 helper」**不存在**：全包无 `func newTestAppWithTempDir(`；`waitGensDone` 仅定义在 `create_chapter_cancel_test.go:105`，仍靠 13 处调用点手工调用（`create_chapter_plan_gate_test.go:421/464/484/504/548`、`ctx_compiler_test.go:84`、`novel_revolution_axis_test.go:276`、`plan_loop_e2e_test.go:321`、`story_spine_handler_test.go:207`、`style_digest_handler_test.go:136` 等）——即「写盘协程尾步必须等完」仍是个人纪律而非默认路径。
- CHANGELOG 侧复发记录仍在（当前行号）：`:64`「随版收口 `TestCreateChapterDigestInjection` 的 unlinkat 竞态（刀5 测试漏补 waitGensDone）」、`:68`「随版收口两处 HTTP 断言测试的 Windows unlinkat 竞态（…… v4.233 在册坑复发）」、`:931`（根因描述与本次修法）。前端的 flaky 分类器已存在（`frontend/scripts/classify-vitest-failures.mjs`，CI `ci.yml:124` 调用）——那是 vitest 面，**不覆盖** Go 侧 unlinkat。
- **一行最小修法**：先补 #5 的 `chapterGenWG.Add(1)/Done()`，再抽 `newTestAppWithTempDir(t)`（内部注册 `t.Cleanup`：先 `waitGensDone` 再删 TempDir），把现有 13 处手工调用替换为该 helper。

### 25（源 X1-03）三重 `*core` 嵌入导致 nil 嵌入 panic 三犯 —— 仍开放

- 现场：`internal/app/app.go:272-277` 现为**五重**嵌入——
  ```
  *core
  *writingState
  *mediaState
  *whisperState
  *officeState
  ```
  且四个 state 各自再嵌一份 `*core`：`app.go:80-81`（`writingState`）、`134-135`（`mediaState`）、`176-177`（`whisperState`）、`228-229`（`officeState`）。审计当时的「三重」行号 `273-275` 仍是这三行（`*core`/`*writingState`/`*mediaState`），未消失。
- 唯一装配点：`app.go:308` `a := &App{core: c}`（`309-317` 挂各 state）——构造函数是隐式的，`App` 仍是可裸构造的导出零值。
- 裸构造面仍旧很大：`internal/app` 内 `a := &App{}` / `(&App{}).Xxx` 大量存在（如 `gaea_dag_test.go:160,214,245,277,305`、`gaea_cost_projects_test.go:39`、`gaea_export_test.go:31,50,80,94,132`、`bindings_completeness_test.go:14`、`board_manifest_test.go:31,57,99,150`），说明「nil 嵌入」仍靠调用点自觉。
- 无 fail-fast：全包检索无 `assertAssembled` 式启动断言；部分绑定自保（`gaea_cost_import_vision.go:511`、`gaea_cost_compose.go:281`、`gaea_schedule.go:113` 等 `a == nil || a.core == nil` 判空）是**逐点**防御而非装配断言。
- **一行最小修法**：加 `func (a *App) assertAssembled()`（校验 `core` 与四个 state 非 nil，否则 panic 并提示用唯一构造函数）并在绑定入口首行调用；或把 `core` 从四个 state 的嵌入里去掉，改为显式字段。

---

## 附：口径与局限

1. **快照与移动靶**：本复核全部行号均以当前工作树（HEAD `3d0210d1`，含未提交改动）为准，逐条用 read/grep 现场核对；`audit` 旧行号仅在「与当前一致」时才引用。已知审计后 HEAD 又推进了 v4.451/452/453/454 四次提交，另有并行修复线在工作树内改码：`git status` 当前为 **21 项已跟踪改动 + 27 项未跟踪**（未跟踪里含 `internal/gaea/config/render_preserve.go`、`scripts/check-primitives.mjs`、`internal/bookimport/reconstruct_extractjson_test.go`、`internal/gaea/config/render_preserve_test.go`、`internal/gaea/config/zz_probe_providers_test.go` 等）。本复核**未新增或修改**任何被跟踪文件。
2. **未跑门禁**：按纪律**未执行** `go test` / `go build` / `wails build` / `golangci-lint` / `tsc` / `vitest`。因此所有「本轮已修」结论都是**源码证据级**（新代码位置 + 回归锁存在），不是「测试已绿」级；判断 P0#16（CostLibraryView）为「无法判定」正因缺这一层可执行证据。
3. **编号与去重**：README §3 的 25 行里有 3 行（#19/#20/#21）与前面的 #1/#2/#8 是同一条 finding（`render.go:13-225`、`boot.go:623-632`、`fold.go:332-339`），fid 分别为 GA5-01/GA5-02/GA5-03；因此 25 行对应 **22 条独立 P0**。总览表保留 25 行不合并（便于与 README 对账），并在「源 id」列标明对应关系。
4. **证据强度分级**：本轮把「本轮已修」限定为「在**当前工作树**读到修复代码位置（含新增未提交文件）且能看到对应回归锁/新增用例」；仅当 P0 的原始证据行**已不存在**时才允许判「已修」。P0#14（baselines）与 #15（assignment）的新增回归测试文件 `gaea_schedule_file_test.go` 在工作树中为未提交改动，回归锁能否通过需实跑（本轮未跑）。
5. **未覆盖的相邻面**：`internal/bookimport/reconstruct.go` 的 `ExtractJSON` 同族副本（任务书点名的另一条线）已在本轮顺带核到并判「本轮已修」，它不在 25 条编号内，仅作附带登记。另：P0#15 的 `Resource`/`ManualStart`/`FixedCost` 裸值残留、P0#3 的跨进程收件箱竞态、P0#8 的绑定入口统一 recover，都是条目内已如实登记的**残余**，本轮只标注不扩项。
6. **不越权拍板**：P0#6（三套编排合一）与 #10（DAG 并发闸）的「合并入口/统一状态机」属需要拍板的架构决策（README §5 已与 06 分册刀8、刀序原则绑定），本轮只给现状证据与「先上信号量」式短期止血修法，不代替拍板。
