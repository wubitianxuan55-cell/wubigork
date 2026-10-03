# gaea 全仓代码审计报告 · 2026-10-02

> **修复线程终态（2026-10-03）**：P0 25 条全清；四簇（安全/真 bug/口径单源/语义冲突）+ duplication 簇 + god-file 族 + 留池/上报未动池/观察池**全部清账或归拍板**。修复台账=`round-1~52`（批 1~50，免拍板活池清零）；**待您裁决的余量菜单=[decision-brief-2026-10-03.md](decision-brief-2026-10-03.md)**（结构大刀 7 + 绑定/UI 小拍板 7 + 等时机 2，每项含建议）。
>
> **本次交付**：27 个板块单元并行只读审计（并行子代理）+ 6 份分册综合 + 2 份机械扫描附录。
> **重点**：屎山代码（上帝文件/函数、复制粘贴与平行实现、死代码、吞错、层级倒挂、并发隐患、多套并行口径、兼容壳堆积）。
> **一句话结论**：静态门禁全绿（`go vet` / `golangci-lint` 0 issues），但**结构层是烂的**——27 个板块平均 **4.54 / 10**（10=干净，0=屎山），**512 条带 `file:line` 证据**的问题，其中 **25 条 P0**（静默丢数据、崩溃、真竞态、对用户撒谎式降级）。

| 分册 | 内容 | 单元 findings | P0 | 可执行刀 |
|---|---|---|---|---|
| [01 后端 gaea 内核分册](01-后端-gaea内核分册.md) | Agent/工具/记忆检索/控制面/上下文配置/造价域 | 111 | 5 | 14 |
| [02 后端应用层·小说与轻语分册](02-后端-应用层-小说与轻语分册.md) | 生成链、场景卡、章节计划、轻语/原罪/书源 | 38 | 2 | 21 |
| [03 后端应用层·办公/造价/进度与基础设施分册](03-后端-应用层-办公造价进度与基础设施分册.md) | 办公文档、造价、进度/DAG/任务、微信语音、绘梦、绑定面、记忆中枢 | 131 | 9 | 19 |
| [04 内部领域包与 whisper 分册](04-内部领域包与whisper分册.md) | 小说/角色/项目/书源域包、底座包、办公解析、whisper | 79 | 1 | 15 |
| [05 前端（React/TS）分册](05-前端分册.md) | gaea/components、lib、pages、schedule、genui | 135 | 5 | 21 |
| [06 横切面分册](06-横切面-重复实现与死代码分册.md) | 重复实现、死代码与僵尸资产、历史审计对账、复发型缺陷 | 18 | 3 | 14 |
| [附录 A · 机械扫描](appendix-A-mechanical.md) | 上帝文件/上帝函数 AST 统计、标记词频、死代码候选、复制粘贴证据、测试卫生、仓库体积 | — | — | — |
| [附录 B · 绑定面与依赖图](appendix-B-surface.md) | 743 绑定面的前端消费面、legacy api 层存活、139 包依赖图、资产存活 | — | — | — |

**合计 512 条**（P0 25 / P1 216 / P2 209 / P3 62），6 分册共给出 **104 条可执行刀**（刀号—目标—涉及文件—风险—验收口径）。

---

## 1. 审计口径与快照（**读结论前必看**）

| 项 | 值 |
|---|---|
| 审计起始快照 | HEAD `24926674`（release: v4.450.0），工作树已含未提交改动 |
| 审计期间 HEAD 前进到 | `cc954cdd`（release: v4.451.0）——**有并发会话在同一工作树里改码** |
| 工作树脏项 | 21 项（`README.md`、`wails.json`、`versioninfo.rc`、`prompts/*.json`、`internal/app/*.go`、`frontend/src/pages/CreatePage.tsx` 等，另含新增未跟踪文件 `internal/app/chapter_plan_position.go`） |
| 审计窗口内被写入的源文件 | 17 个（10:04–10:13，集中在小说计划/分支链与 CreatePage） |
| 审计窗口内并发会话的发布 | v4.451.0 提交 `cc954cdd` 落地；**v4.452.0 的发布说明同期已在写**（`.gaea/AGENTS.md` 速览可见） |

**独立佐证（时效性证据）**：v4.452.0 的发布说明自陈「**heredoc 反斜杠 n 工具层降级（第三次踩）**」，与本报告 P0 第 23 条（heredoc 转义坑在册复发、无守卫）**独立互证**——这类复发型缺陷仍在以「每次消耗一个发版窗口」的节奏复发，且本次审计窗口正好覆盖了它复发的那一次。

**三条因此而来的限制**：

1. **移动靶**：单元审计员读到的是「读取当时」的内容，行号与行数为当时值。热区（小说章节计划/分支、CreatePage、`app_info.go`、prompts）可能已漂移；06 分册 §五 W1 已记录现场复测差异（例：单元记 `gaea_ui_extra.go` 1188 行 / `orchestrator.go` 1033 行，主编复测 994 / 857，本次终稿复核为 **1070 / 971**——三方不一致本身就是「并发改码」的直接证据）。**复核任何条目请以当前工作树为准，并按 `git log` 判断该文件是否在审计后被改过。**
2. **评分口径**：本报告分数是**烂代码密度**（10=干净，0=屎山），**不是**可用性/完成度评分。2026-09-26 那轮审计档的「综合 7.0/10」是另一种口径，两者**不可直接比较**（那轮判「功能达成 + 工程纪律」，这轮判「结构债」）。
3. **只读纪律已守住**：全部审计员被禁止改源码。`git status` 里所有改动都能归因到并发会话；本审计**只新增** `docs/code-audit-2026-10-02/`。三个 P0 由我（主代理）逐行核对过源码，见下表「复核」列 ✅。

**单元补审说明（AP7）**：绘梦/图像/视觉单元（AP7）首轮由工作流编排的审计员完成并落盘，但工作流层回报该 agent 失败；主代理随后**重做了该单元**，`units/AP7.json` 为第二轮结果（20 条 / 1 条 P0，每行 evidence 已按工作树当前行号逐行回验 0 处不匹配）。03 分册综合时用的是第一轮内容——两轮**结论一致**（同一条 P0：三链共用包级单槽进度/取消导致并发串台），仅在个别 P1/P2 条目上互换，故分册内计数与本报告单元级合计可能相差 ±1，**细节一律以终稿 `units/AP7.json` 与本报告为准**。

---

## 2. 数字总览

### 2.1 规模与门禁基线

| 项 | 文件数 | 非测试 LOC | 测试文件 | 测试 LOC |
|---|---|---|---|---|
| Go（`internal/`） | 874 | **203,872** | 849 | 153,259 |
| TS/TSX（`frontend/src`） | 713 | **148,135** | 414 | 60,033 |
| **合计（非测试）** | **1,587** | **352,007** | 1,263 | 213,292 |

- `go vet ./...` → exit 0，0 条输出（2026-10-02 本机实测）
- `golangci-lint run ./...`（v2.14.0，errcheck / govet / ineffassign / staticcheck / unused / misspell 全开）→ **`0 issues.`**
- **含义**：lint 可查的那一层已经清零。本报告 512 条**没有一条**是 lint 能抓的——全部是结构、契约、并发、口径层面的债。把门禁当健康证明是本仓最大的认知陷阱。

### 2.2 严重度 × 类别矩阵（512 条）

| 类别 | P0 | P1 | P2 | P3 | 合计 |
|---|---|---|---|---|---|
| duplication（复制粘贴/平行实现） | 3 | 65 | 63 | 4 | **135** |
| consistency（多套并行口径） | 2 | 33 | 47 | 9 | **91** |
| error-swallow（吞错/静默失败） | 4 | 9 | 21 | 2 | 36 |
| coupling（层级倒挂/跨层耦合） | 1 | 22 | 7 | 4 | 34 |
| concurrency（并发/竞态） | 9 | 23 | 2 | 0 | **34** |
| dead-code（死代码/僵尸资产） | 0 | 11 | 9 | 12 | 32 |
| complexity（复杂度） | 0 | 7 | 16 | 3 | 26 |
| safety（安全） | 5 | 8 | 8 | 4 | 25 |
| god-file（上帝文件） | 0 | 19 | 2 | 0 | 21 |
| legacy-residue（兼容壳/历史残留） | 1 | 4 | 6 | 9 | 20 |
| perf-smell | 0 | 3 | 14 | 2 | 19 |
| doc-drift（注释与实现漂移） | 0 | 3 | 7 | 8 | 18 |
| god-func（上帝函数） | 0 | 9 | 2 | 0 | 11 |
| test-smell | 0 | 0 | 5 | 5 | 10 |
| **合计** | **25** | **216** | **209** | **62** | **512** |

**读法**：`duplication + consistency = 226 条（44.1%）`是绝对主因——这个仓的烂不是「某个文件写得差」，而是**同一件事没有唯一实现**。`concurrency` 的 P0 密度最高（34 条里 9 条 P0，26%），说明并发是当前最容易真咬人的面。

### 2.3 评分总表（单元级，低分=屎山重；完整表见 [machine/readme_blocks.md](machine/readme_blocks.md)）

| 分册 | 单元 | 板块 | 分 | P0 | P1 |
|---|---|---|---|---|---|
| 03 | AP4 | app 造价/进度/DAG/任务 | **3.0** | 4 | 7 |
| 01 | GA5 | gaea 上下文与配置 | **3.0** | 3 | 6 |
| 02 | AP1 | app 小说生成链 | 4.0 | 2 | 10 |
| 03 | AP3 | app 办公文档与交付 | 4.0 | 0 | 5 |
| 03 | AP8 | app 绑定面与主干 | 4.0 | 0 | 11 |
| 03 | AP9 | app 记忆中枢/brain/herdsman | 4.0 | 2 | 6 |
| 05 | FE2 | 前端 gaea/components 其余 + memoryhub | 4.0 | 1 | 7 |
| 05 | FE3 | 前端桥接与 mock 双实现面 | 4.0 | 0 | 6 |
| 05 | FE5 | 前端 pages + layouts + boards | 4.0 | 0 | 5 |
| 01 | GA3 | gaea 记忆与检索 | 4.0 | 2 | 6 |
| 04 | IN1 | 内部小说领域包 | 4.0 | 0 | 10 |
| 04 | IN4 | internal/whisper | 4.0 | 0 | 8 |
| 03 | AP7 | app 绘梦/图像/视觉 | 4.5 | 1 | 10 |
| 01 | GA6 | gaea 造价域 | 4.5 | 0 | 6 |
| 04 | IN3 | 内部办公/解析/日程包 | 4.5 | 0 | 13 |
| 03 | AP6 | 微信/语音/TTS/ASR | 4.6 | 2 | 8 |
| … | … | 其余 11 个单元 5.0–6.0（最干净：GA1/GA2 `gaea/agent`、`gaea/tool` = 6.0） | | | |

分册均值：01 = 4.75、02 = 4.75、03 = 4.16、04 = 4.38、05 = 4.71、06 = 5.00。

---

## 3. 屎山 TOP 榜（表内 25 行 = **22 条独立 P0** + 3 行重复 + 复核状态）

> 「复核 ✅」= 主代理亲自 read 源码逐行核对过（不是转述子代理）。`位置` 为审计当时行号。

| # | 单元 | 问题 | 位置 | 一刀修法 | 复核 |
|---|---|---|---|---|---|
| 1 | GA5 | **`RenderTOML` 只渲染 8 个段，保存即丢 13 个配置段**（`[memory] [dream] [session] [space] [plugins] [skills] [search] [network] [tasks] [retrieval] [vision] [markdown_converter] [workspace]`） | `internal/gaea/config/render.go:13-225` | Render 前先 `toml.Decode` 保留未识别段，或补全 13 段；加「Default+全段赋值 → Render → Decode → 逐字段 DeepEqual」用例（当前必红） | ✅ |
| 2 | GA5 | 一次「始终允许」审批即触发上面的丢段落盘（`PersistAllowRule` → `Load` → `Save` → `RenderTOML`） | `internal/gaea/boot/boot.go:623-632` → `config/config.go:837` | 同上修好后自动消失；另加「审批后重读 gaea.toml 断言段文本仍在」的绑定断言 | ✅ |
| 3 | AP4 | **任务收件箱读-改-写全程无锁**（`loadTaskInbox` → 改切片 → `saveTaskInbox` 原子整写）；跨会话并发写者（UI/语音/微信）必丢一条 | `internal/app/gaea_task_inbox.go:153-219` | 加包级 mutex 或改为文件锁 + 版本号；补并发写用例（当前无任何锁符号） | ✅ |
| 4 | AP1 | **收敛闭环后台 goroutine 无登记、无取消、无 WaitGroup**：切书后仍持旧 `project.Manager` 写盘，可与整章/逐场景生成同时改同一章 | `internal/app/converge_handler.go:177-232` | 纳入既有 `chatStreamCancels` 式登记表 + `waitGensDone` WG + 同章互斥（复用章节生成闸） | — |
| 5 | AP1 | 逐场景生成协程漏计 WG，`waitGensDone` 提前返回 → 重新引入已根治的 Windows unlinkat flaky | `internal/app/scene_cards_handler.go:209-219` | 每场 `wg.Add/Done`，收尾 `waitGensDone` 后再返回 | — |
| 6 | GA3 | knowledge 后端把 DB 错误吞成空切片 → `MatchRows` 查重全失效 → **重复入库且提示「不覆盖」** | `internal/gaea/knowledge/sqlite.go:77-82` → `knowledgeimport.go:163` | `List/ReadAll` 返回 error 并让导入预览中止；补「注入 db 失败 → 断言报错」用例 | — |
| 7 | GA3 | 进程级全局可变 `memory_search` 索引被多控制器互相覆盖 → work/play 跨空间串味 + goroutine 竞争 | `internal/gaea/tool/builtin/memory_search.go:19-21`（写点 `control/controller_memory.go:285`） | 索引按 space/controller 实例化（去掉包级 var），补 `-race` 双控制器用例 | — |
| 8 | GA5 | 轨迹折叠遇残缺会话日志（首条即 usage/text 增量）返回 nil 被解引用 → 看板 panic（Wails 绑定无 recover） | `internal/gaea/trajectory/fold.go:332-339` | 空守卫 + 三份残缺夹具用例；绑定入口加统一 recover 中间件 | — |
| 9 | AP4 | **三套并行编排**（DAG / 任务调度器 / 收件箱）各带队列与状态机，横切能力必须写三遍（取消/重试/续跑语义互不一致） | `internal/app/gaea_dag.go:512-528` | 先统一任务身份与状态机（见 06 刀8），再谈合并入口 | — |
| 10 | AP4 | DAG 执行器自建并发模型，绕过任务闸口：8 节点并行会在 `local_concurrency=1` 前提下同时拉起 8 个模型会话 | `internal/app/gaea_dag.go:545-548` | 节点执行走同一并发闸（信号量来自 config），并与 07 点同批做 | — |
| 11 | AP4 | 成本导入直写 UPSERT 与 `Store.Save` 双写路径；BM25 缓存靠「调用方记得手动失效」的隐式契约 | `internal/app/gaea_cost_import.go:252-264` | 收敛到唯一写入口 `SaveTx`（01 分册刀7/06 分册刀3） | — |
| 12 | AP6 | `voice.Manager` 回调字段无锁写读：启动刷新 goroutine 与语音回合并发构成真 data race | `internal/voice/voice_manager.go:212-220` | 改 `atomic.Pointer[callbacks]`；`go test -race` 覆盖生产接线形态 | — |
| 13 | AP6 | `whisperSessions` 跨锁窗口双建 orchestrator 后互相覆盖（GUI/微信/语音三入口同抢人格） | `internal/app/whisper_handler.go:49-118` | 建实例放锁内（或 `singleflight`），重复灌历史一并修 | — |
| 14 | AP7 | 绘梦生成链无单飞，进度与取消是**包级单槽**：三链并发生成必串台、取消只中断最后一条（前端串行队列只是规避） | `internal/app/image_handler.go:97-102` | 每任务独立进度槽 + 生成单飞；前端撤掉代偿队列 | — |
| 15 | AP9 | 数字生命读取全吞错：表名/列名变了显示「0 个角色」，JSON 坏了该角色凭空消失且无日志 | `internal/app/herdsman_digitallife.go:105-117` | 分「真无数据」与「读失败」两态并上报；错误至少进日志与面板提示 | — |
| 16 | AP9 | Herdsman CLI 超时只杀父进程，模型服务器成孤儿进程 → 显存打满 | `internal/app/herdsman_lifecycle.go:84-97` | 杀进程树（Windows Job Object / `taskkill /T`），超时后校验端口释放 | — |
| 17 | FE2 | ~~表格视图「全选」调用未定义函数~~ **〔复核撤销：判据不成立〕** `setAllSelected`/`clearSelection` 定义在同文件 `1126-1131`（函数声明提升，点击不会抛 ReferenceError，且该文件自 v4.386.0 未改过）。**真实缺口降级为 P2**：表头全选没有测试覆盖 | `frontend/src/gaea/components/CostLibraryView.tsx:1126-1131`（定义处） | 补「点表头复选框 → 全部行进 selected」用例即可 | 主代理 + 复核线亲核 |
| 18 | FE6 | `genui_validate` 把数据数组 `diffs` / `series` 当子节点递归 → **合法规格被判非法**，模型据此反复改写甚至放弃组件 | `internal/gaea/genui/validate.go:113-121` | 区分「节点数组」（items/tabs/steps/options）与「数据数组」（diffs/series）；Go 与 TS 常量集合双向断言 | ✅ |
| 19 | FE6 | 麦克风不可用静默转「模拟模式」：界面照样显示在听、AI 照样回（走文本），用户无法分辨 | `frontend/src/hooks/useVoiceChat.ts:380-383` | 显式降级态 `mic-unavailable` + 警示条（对齐本仓其它诚实降级路径） | — |
| 20 | FE7 | **多基线 `baselines` 被 Go 保存链路静默擦除**（前端类型有、Go 侧无字段，防抖自动保存即清空槽位） | `frontend/src/schedule/types.ts:115-116` | Go 侧补 `Baselines` 字段 + 往返测试 `len(Baselines)==3` 顺序不变 | — |
| 21 | FE7 | `assignment` 的 `Quantity`/`Amount` 是裸 `float64` + `omitempty`：**显式 0 经一次往返被抹成缺键**〔复核更正：方向是「0 → 缺失」，不是「缺失 → 0」；`Units` 早已是 `*float64`〕 | `internal/schedule/types.go` | 改 `*float64` 三态 + 绑定层往返断言（本轮已修） | 线5 实测更正 |
| 22 | IN2 | `ExtractJSON` 取**最后一个**右花括号 → 回复含多个 JSON 对象时解析失败（25 处生产调用点） | `internal/util/util.go:82-96` | 改为括号配平扫描 + 单测覆盖「正文含两个对象」 | — |
| 23 | X1 | heredoc 转义坑在册**复发 5 次**，无任何机器守卫 | `CHANGELOG.md:2,29,41,918,922,928` | 提交前文本守卫（禁 `\n` 进 Go 字面量的脚本化检查） | — |
| 24 | X1 | Windows unlinkat/TempDir 竞态**三度复发打挂 CI** | `CHANGELOG.md:55,59,922` | 收尾纪律机器化（`waitGensDone` 缺失检测）+ flaky 分类器 | — |
| 25 | X1 | 三重 `*core` 嵌入导致 nil 嵌入 panic **三犯** | `internal/app/app.go:273-275` | 装配改为显式构造器（禁裸 `&App{}` 测试桩）或启动断言 | — |

> **本轮（2026-10-02 第一轮止血刀）已修**：#1/#2（配置保存丢段）、#3（任务收件箱无锁）、#18（GenUI 校验误判）、#20（多基线被抹）、#21（数值三态，方向已更正）、#22（`ExtractJSON` 多对象）；逐条改动与红→绿证据见 [round-1-fixes.md](round-1-fixes.md)，25 条在**当前工作树**上的复验结论见 [round-1-p0-status.md](round-1-p0-status.md)。
>
> **复核更正（2026-10-02 复核线 + 主代理亲核）**：① 上表 25 行中 **#19/#20/#21 与 #1/#2/#8 是同一条**（GA5-01/02/03 重复行），故**独立 P0 实为 22 条**；② **#17 撤销**（判据不成立，见该行）；③ 复核后仍未开放的独立条目集中在：后台链登记纪律（#4/#5）、检索与知识库吞错/全局索引（#6/#7）、崩溃面（#8）、三套编排与并发闸（#9/#10/#11）、微信语音并发（#12/#13）、绘梦单槽进度（#14）、三类复发型缺陷无守卫（#23/#24/#25）。

**P0 的性质分布**：25 条里 **10 条本质是「静默」**（丢配置、吞 DB 错、擦基线、吞表错、假装在听、静默串稿…）；8 条是并发/登记纪律（#3#4#5#10#12#13#14）；5 条是数据契约（#20#21#22 等）；2 条是崩溃。**静默失败和并发登记是当前最高性价比的止血面**。

---

## 4. 系统性债 TOP 8（跨板块同根因——比单个文件更值得先动）

| # | 债 | 证据（跨板块） | 为什么排这么前 |
|---|---|---|---|
| 1 | **基础设施原语被平行重写** | 截断函数 33~41 份（06 分册刀4：`grep -c '^func truncate\|^func clip'` = 41）；原子写 5 份绕开 `fileutil.AtomicWrite`；`estimateTokens` 4 份；BM25 2 份（k1=1.5 / k1=1.2）；分词器 5 份；工具名单 6 份；跳过目录 6 份 | 每加一个功能就多一份副本，且**副本会漂移**（已在 token 预算、召回质量、成本算法上实际漂移） |
| 2 | **上帝装配包与上帝文件** | `internal/app` 一个包 **import 110 个包**（第二名 `gaea/boot` 27）；Go 单函数 ≥100 行 **110 个**（≥200 行 19 个；最差 `boot.Build` **551 行**、`config.Load` 449、`agent_run.runDirect` 449、`schedule/ops.applyOne` 434）；前端 4 个千行页面（CreatePage 1501）与 4 个千行组件 | 任何横切改动都会扫到这些文件；拆不动 → 只能继续糊 |
| 3 | **三套编排并存** | DAG（`gaea_dag.go` 792）/ 任务调度器（`gaea/tasks/tasks.go` 1244）/ 收件箱（`gaea_task_inbox.go`）；收件箱还无锁 | 取消、重试、重启续跑三套语义，未来每个横切能力都要写三遍 |
| 4 | **绑定面死面 43%** | 743 个方法中 **322 个（43.3%）在前端（含 legacy `api/` 层）零引用**，31 个仅测试引用，287 个只有 1 处生产引用；而漂移闸只校「名字被认领」不校「有人调用」（[附录 B](appendix-B-surface.md) B1） | 743 名 × 4 处手工清单（bindingNames / bridge / spaceBindings / d.ts）+ mock 1.4 万行都要跟着维护 |
| 5 | **两套前端调用面并存** | `frontend/src/api/**`（18~53 个导入方的 legacy `window.go.app.App` 代理）与 `gaea/lib/bridge/**`（4 层同名路由代理）同时活着；页面还直接 import `wailsjs` 绕过门控 | 门控/错误归一/空间校验只在其中一条路上生效 |
| 6 | **同一概念两套包** | `internal/config` vs `internal/gaea/config`；`internal/memory` vs `internal/gaea/memory`；`internal/context` vs `internal/gaea/context`；`internal/search` vs `internal/gaea/search`；`internal/skill` vs `internal/gaea/skill`；`internal/core` 与 `internal/office` 是逐行平迁副本 | 改一处必漏一处（Twin package 7 组，见附录 B3） |
| 7 | **whisper 整域是逐文件转录 + 未接线** | 19.4k 行（123 文件）；`ConfirmSvc`/`DeliveryCoord` 只构造不读取；桌面路由 3 条中 2 条零生产调用；情绪涌现两套状态源 | 功能「在册」但行为不生效——是最容易被误判为「已完成」的债 |
| 8 | **复发型缺陷无守卫** | heredoc 转义 ×5、Windows unlinkat 竞态 ×3、`*core` nil 嵌入 panic ×3；`CHANGELOG.md` 1.28 MB / 6987 行是唯一账本 | 同类坑靠再踩一次登记，每次消耗一个发版窗口 |

---

## 5. 合并拆刀路线（6 分册共 104 刀，这里给主干）

**刀序总原则（06 分册主张，我采纳）**：**先建守卫，再收敛**。本仓已有实证——上一轮审计（2026-09-26）定下的验收线未守住：`internal/app` 文件数/行数反涨、截断函数从 8 处涨到 41 个定义。没有机器守卫的收敛 = 下一版必回涨。

### 第一批：止血刀（P0，建议立刻拍板）

| 刀 | 目标 | 关键文件 | 验收口径 |
|---|---|---|---|
| **A0** | **先建 4 条机器守卫**（06 刀1）：禁新增 `truncate*/clip*` 定义、禁 `internal/app` 新 `os.CreateTemp`、禁裸 `a.download=`/`atob`、`gen_bindings shadowed` 超基线即 fail | `frontend/eslint-rules/`、`scripts/`、`gen_bindings`、`ci.yml` | 三条负例提交必须被 CI 拒 |
| **A1** | 配置段丢失（P0#1#2） | `gaea/config/render.go`、`config/edit.go`、`boot.go:623-632` | Default+全段 → Render → Decode 逐字段全等（当前必红） |
| **A2** | 崩溃与竞态（P0#3#7#12#13） | `gaea_task_inbox.go`、`memory_search.go`、`voice_manager.go`、`whisper_handler.go` | 并发写不丢条 + `-race` 双 Controller 零报告 |
| **A3** | 吞错改显式（P0#6#8#15） | `knowledge/sqlite.go`、`trajectory/fold.go`、`herdsman_digitallife.go` | 注入失败 → 断言返回 error / 不 panic / 面板可见 |
| **A4** | 后台链登记纪律（P0#4#5#10） | `converge_handler.go`、`scene_cards_handler.go`、`gaea_dag.go` | 切书后无残留协程；`waitGensDone` 真等；DAG 并发受闸 |
| **A5** | 数据契约往返（P0#20#21#22） | `internal/schedule/types.go`、`gaea_schedule_file.go`、`internal/util/util.go` | 基线 3 份往返不丢；0 与缺失可区分；多对象 JSON 可解析 |
| **A6** | 前端即时可感的错（P0#17#19） | `CostLibraryView.tsx`、`useVoiceChat.ts` | 全选可用 + 麦克风降级有提示条 |
| **A7** | GenUI 校验器与单一真相源（P0#18，06 刀2） | `gaea/genui/validate.go`、`frontend/src/genui/*` | 含 `diffs`/`series` 的合法规格判 OK；Go/TS 常量双向断言 |
| **A8** | 绘梦生成串台（P0#14） | `image_handler.go`、`image_comfyui_proc.go` | 三链并发生成各自进度、取消精确到任务 |

### 第二批：收敛刀（P1，季度级；每刀都须有守卫兜底）

原语收敛（截断→`strutil`、原子写→`fileutil.AtomicWrite`、token 估算×4→1、BM25/分词→1、清单→1）；成本域三合一（01 刀7/13 + 06 刀3）；工具元数据唯一源（01 刀6）；`internal/core`↔`internal/office` 合并（06 刀9）；图片后端解析 5→1（06 刀10）；LCS diff 四合一（05 刀10）；相对时间/防抖 hook/通知条收口（05 刀11~13）；绑定面三清单统一 + 死绑定检测（05 刀5）。

### 第三批：结构刀（拆上帝，纯搬移零功能；先锁定黄金测试）

`boot.Build`（551 行）、`gaea_ui_extra.go`（1070）、`image_handler.go`（1011）、`whisper_handler.go`（943）、`controller.go`（963）、`agent.go`（975）/`agent_run.go`、`contextview/fold.go`（939）、`websearch.go`（886）、`project.go`（1011）、`mpp.go`（873）、`docxedit.go`（776）、`CostLibraryView.tsx`（1133）、`CreatePage.tsx`（1501）、`App.tsx`（1041）。

> 每份分册 §四 都有完整的「刀号 | 目标 | 涉及文件 | 风险 | 验收口径」表，本表只是主干；执行时以分册为准。

---

## 6. 证据与可复跑性

| 资产 | 说明 |
|---|---|
| `units/<ID>.json`（27 份） | 每个板块单元的原始证据：verdict / score / metrics / hotspots / findings（含 `file`+`line`+`evidence`+`why`+`impact`+`fix`+`effort`+`prior_audit`）/ open_questions |
| `machine/scan.py` `scan2.py` `render_appendix*.py` | 机械扫描：上帝文件/函数、标记词频、长行复制、重复块、同名文件、死导出候选、测试卫生、仓库体积 → [附录 A](appendix-A-mechanical.md) |
| `machine/dead_bindings.py` `imports.py` `api_layer.py` | 绑定面消费面、139 包依赖图、legacy api 层存活面 → [附录 B](appendix-B-surface.md) |
| `machine/godfunc`（源码 `.tmp/godfunc/main.go`） | 用 `go/parser`+`go/ast` 精确统计函数跨度（花括号计数会被字符串里的 `{}` 污染，已弃用该口径） |
| `machine/aggregate.py` `readme_blocks.py` | 27 份单元 JSON 汇总结算（严重度矩阵、评分表、P0 全表） |
| `_AUDIT_BRIEF.md` / `_SYNTH_BRIEF.md` | 审计员与分册主编的作业书（口径公开，便于复现同类审计） |

---

## 7. 局限与未定论

1. **只读审计**：未跑测试/构建/真机走查（避免与并发会话的改码互相干扰）。所有「修法/验收口径」是可执行提案，**尚未实测**。
2. **322 个零引用绑定 ≠ 可以删**：其中一部分是给 CLI（`gaea ...`）、HTTP 面（`GAEA_HTTP_PORT` 模式）或外部脚本留的导出。AGENTS 里既有的「零调用者绑定删除待拍板（宽口径 36）」与本报告窄口径 322 **差一个量级，两套口径必须先对齐**再动手。
3. **移动靶**：见 §1。复核热区结论前先 `git log -1 --format=%h -- <file>` 确认审计后是否被改过。
4. **未定论项 78 条**分散在 6 份分册 §五「观察项 / 未定论」，包括：`internal/gaea/genui` 常量到底几份事实源（三份单元结论并存，06 分册 W8 待裁决）、`internal/app` 行数验收线是「≤800 行」还是「>50KB」（两条口径并存，06 W2）、mock 契约是否机器化、甘特库 spike 等——这些**需要拍板而非改码**。
5. **评分主观性**：单元分数由审计员按证据自评，量级可比、小数位不必当真；判断优先级请用 P0/P1 与「系统性债 TOP 8」，不要用分数排序做排刀唯一依据。
