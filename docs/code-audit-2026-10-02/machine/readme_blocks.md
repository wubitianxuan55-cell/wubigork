### 评分总表（单元级，低分=屎山重）

| 分册 | 单元 | 板块 | 评分 | findings | P0 | P1 |
|---|---|---|---|---|---|---|
| 03 | AP4 | app 造价/进度/DAG/任务（含 internal/gaea/t | 3 | 20 | 4 | 7 |
| 01 | GA5 | gaea 上下文与配置（context/contextview/se | 3 | 20 | 3 | 6 |
| 02 | AP1 | app 小说生成链（章节/角色/大纲/场景/伏笔/分支/计划/评测） | 4 | 18 | 2 | 10 |
| 03 | AP3 | app 办公文档与交付（含 internal/office、inte | 4 | 17 | 0 | 5 |
| 03 | AP8 | app 绑定面与主干（app.go/bindings/module/ | 4 | 18 | 0 | 11 |
| 03 | AP9 | app 记忆中枢/brain/herdsman/角色库/看板 | 4 | 14 | 2 | 6 |
| 05 | FE2 | 前端 gaea/components 其余 + memoryhub  | 4 | 18 | 1 | 7 |
| 05 | FE3 | 前端桥接与 mock 双实现面 | 4 | 21 | 0 | 6 |
| 05 | FE5 | 前端 pages + layouts + boards | 4 | 20 | 0 | 5 |
| 01 | GA3 | gaea 记忆与检索（memory/factbase/semanti | 4 | 20 | 2 | 6 |
| 04 | IN1 | 内部小说领域包（character/characterlib/cha | 4 | 20 | 0 | 10 |
| 04 | IN4 | internal/whisper | 4 | 19 | 0 | 8 |
| 03 | AP7 | app 绘梦/图像/视觉 + internal/visual | 4.5 | 20 | 1 | 10 |
| 01 | GA6 | gaea 造价域（cost/costimport/costinqui | 4.5 | 16 | 0 | 6 |
| 04 | IN3 | 内部办公/解析/日程包（office/docmd/schedule/ | 4.5 | 20 | 0 | 13 |
| 03 | AP6 | app 微信/语音/TTS/ASR + internal/voice | 4.6 | 20 | 2 | 8 |
| 03 | AP5 | app 记忆/技能/梦想/搜索/校验杂项 | 5 | 22 | 0 | 10 |
| 05 | FE1 | 前端 gaea/components 大文件组 | 5 | 17 | 0 | 5 |
| 05 | FE4 | 前端 gaea/lib 工具库 + hooks + app + lo | 5 | 19 | 0 | 6 |
| 05 | FE6 | 前端 components + genui + 其它基础设施 | 5 | 20 | 2 | 8 |
| 01 | GA4 | gaea 控制面与运行时（control/boot/jobs/pro | 5 | 16 | 0 | 10 |
| 04 | IN2 | 内部底座包（ai/modelengine/config/netcli | 5 | 20 | 1 | 13 |
| 06 | X1 | 历史审计对账与复发型缺陷（横切） | 5 | 18 | 3 | 10 |
| 02 | AP2 | app 轻语/原罪/书源/导入/书架 | 5.5 | 20 | 0 | 5 |
| 05 | FE7 | 前端 schedule 板块 | 6 | 20 | 2 | 10 |
| 01 | GA1 | gaea/agent（Agent 内核） | 6 | 20 | 0 | 9 |
| 01 | GA2 | gaea/tool+genui+command+vision+dag | 6 | 19 | 0 | 6 |

平均分 **4.54 / 10**（27 单元，等权）

### 严重度 × 类别矩阵

| 类别 | P0 | P1 | P2 | P3 | 合计 |
|---|---|---|---|---|---|
| duplication | 3 | 65 | 63 | 4 | 135 |
| consistency | 2 | 33 | 47 | 9 | 91 |
| error-swallow | 4 | 9 | 21 | 2 | 36 |
| coupling | 1 | 22 | 7 | 4 | 34 |
| concurrency | 9 | 23 | 2 | 0 | 34 |
| dead-code | 0 | 11 | 9 | 12 | 32 |
| complexity | 0 | 7 | 16 | 3 | 26 |
| safety | 5 | 8 | 8 | 4 | 25 |
| god-file | 0 | 19 | 2 | 0 | 21 |
| legacy-residue | 1 | 4 | 6 | 9 | 20 |
| perf-smell | 0 | 3 | 14 | 2 | 19 |
| doc-drift | 0 | 3 | 7 | 8 | 18 |
| god-func | 0 | 9 | 2 | 0 | 11 |
| test-smell | 0 | 0 | 5 | 5 | 10 |
| **合计** | **25** | **216** | **209** | **62** | **512** |

### P0 全表（25 条）

| # | 单元 | 标题 | 位置 | 工作量 | 影响（摘） |
|---|---|---|---|---|---|
| 1 | AP1 | 收敛闭环 goroutine 无登记无取消无 WG | `internal/app/converge_handler.go:177-232` | M | ①用户切书/关闭项目后，该协程仍持旧 project.Manager 继续读写，正是 AGENTS.md 记录过的『切书后残留无主协程』家族；②同一章可以被收敛修补与整章生成/逐场景生成同时改写，作者看到的是静默串稿；③ |
| 2 | AP1 | 逐场景生成协程漏计 WG，waitGensDone 提前返回 | `internal/app/scene_cards_handler.go:209-219` | S | waitGensDone 在逐场景生成仍在写盘时就返回，TempDir 清理撞上未落盘文件——把已根治的 unlinkat flaky 从整章生成路径搬到场景路径重新引入；AGENTS.md 的『任何 CreateCha |
| 3 | AP4 | 三套并行编排各带队列与状态机 | `internal/app/gaea_dag.go:512-528` | L | 任何横切能力必须写三遍且无法共享：DAG 节点执行既不进 tasks 表也不进收件箱，任意一处的取消/重试/重启续跑语义改动都与另两处失配；「跑一条流水线」与「跑一个任务」在 UI 上是两类东西，用户无法用同一入口看全在 |
| 4 | AP4 | 收件箱读改写无锁且有跨会话写者 | `internal/app/gaea_task_inbox.go:153-156` | S | 两条并发写必然丢一条（后写者用自己那份旧快照覆盖）：语音说「存个任务」与微信同时说、或 UI 开启一条任务时正有语音落库，都会静默少一条卡；MaxTasks=500 上限判定同样基于可能过期的切片，可被并发写突破。 |
| 5 | AP4 | 成本导入直写 UPSERT 与 Store.Save 双写路径 | `internal/app/gaea_cost_import.go:252-264` | M | cost 表新增列/改语义时 app 侧静默漂移：导入写入的行缺新列且不报错；BM25 排序缓存依赖调用方记得手动失效，任何新调用点漏调即出现「导入后搜不到」的陈旧结果；两套归一化（normalizeCostEntryF |
| 6 | AP4 | DAG 执行器自建并发模型绕过任务闸口 | `internal/app/gaea_dag.go:545-548` | M | 一条 8 节点全并行的流水线会在 work 空间同时拉起 8 个模型子代理会话，与「local_concurrency=1，同一时间只服务一个模型」的部署前提直接冲突（gaea_schedule.go:6-8 注释同源） |
| 7 | AP6 | voice.Manager 回调字段无锁写读，语音回合与启动刷新竞态 | `internal/voice/voice_manager.go:212-220` | S | 启动末尾刷新 goroutine 与语音回合并发时构成真实 data race（Actions race job 可判红）；最坏情况下 speak/runReply 读到半个接口值或旧回调，表现为语音回合丢回复且日志无线 |
| 8 | AP6 | whisperSessions 跨锁窗口双建 orchestrator 后互相覆盖 | `internal/app/whisper_handler.go:49-118` | M | GUI/微信/语音三入口同时首次进入同一人格时各建一个 orchestrator，后写者覆盖先写者：先建实例上的回合状态推进被丢弃，且 restoreWhisperState 会对同一 hermes.db 会话重复灌历史 |
| 9 | AP7 | 生成链无单飞，进度与取消是全局单槽会串台 | `internal/app/image_handler.go:97-102` | M | 两条链并发时先发起者的进度被后来者覆盖、取消只中断最后提交的那条；v4.258.0 CHANGELOG 已实录「并发会让两条进度条抢同一百分比」，v4.427.0 只在前端加串行队列规避（后端仍开放），任何新调用方（ht |
| 10 | AP9 | 数字生命读取全吞错：库变了显示 0，记录凭空消失 | `internal/app/herdsman_digitallife.go:105-117` | S | Herdsman 升级后表名/列名变化 → 面板静默显示「角色 0 个」而不是报错；characters.data JSON 坏了 → 该角色从列表消失且无任何日志。用户唯一线索是「数据不见了」。 |
| 11 | AP9 | Herdsman CLI 超时只杀父进程，模型服务器成孤儿 | `internal/app/herdsman_lifecycle.go:84-97` | M | 超时后模型进程仍加载在显存里；用户重试 → 再起一个 → 显存被打满，后续一切本地模型功能失败，而 gaea 端只会显示「CLI 调用失败: context deadline exceeded」。 |
| 12 | FE2 | 表格视图全选调未定义函数，点击抛错且无反馈 | `frontend/src/gaea/components/CostLibraryView.tsx:1084-1090` | S | 事件回调抛出的异常不被 ErrorBoundary 捕获，只是 window 未捕获错误：勾选后不会全选（checked 由 selected.size===rows.length 驱动，因此也不打勾），用户看到的是「点 |
| 13 | FE6 | genui_validate 把 diffs/series 当节点，合法规格被判非法 | `internal/gaea/genui/validate.go:113-121` | S | 模型按系统提示词调用 genui_validate 自检时，任何含 diff 或 chart.series 的合法规格都会拿到 ❌（最多 30 条错误刷满），模型会据此反复改写甚至放弃该组件；预算被影子节点侵占还可能让大 |
| 14 | FE6 | 麦克风不可用静默转「模拟模式」，界面无提示 | `frontend/src/hooks/useVoiceChat.ts:380-383` | S | 用户戴着没插稳的麦克风说完整段话，页面全程显示在听、AI 也照常「回复」（走文本通道），实际没有采集到任何音频；用户无法从 UI 分辨「真在听」与「在假装听」，属于对用户撒谎式降级，而本仓其它降级路径（如 Chapter |
| 15 | FE7 | 多基线 baselines 被 Go 保存链路静默擦除 | `frontend/src/schedule/types.ts:115-116` | M | 用户保存第 2/3 个基线后，一旦 800ms 防抖自动保存触发（或下一次 agent 写盘/轻扫回读），baselines 槽位全部从文件消失、前端项目状态被文件内容覆盖；CHANGELOG v4.137.0 明写「b |
| 16 | FE7 | assignment 数值字段 TS 可选/Go 零值，往返往返被改写 | `internal/schedule/types.go:107-111` | S | 分配字段经一次 Go 往返后 0 与缺失不可区分：前端 normalizeProject 的 validNum 接受 undefined 也接受 0，成本 computeCosts 对 0 与 undefined 分支处 |
| 17 | GA3 | knowledge 后端吞 DB 错误，查重失效致重复入库 | `internal/gaea/knowledge/sqlite.go:77-82` | M | knowledgeimport.MatchRows（knowledgeimport.go:163 store.List()）拿不到既有条目时，每条导入行都判为「新增」并提示「不覆盖」，用户确认后按同名字覆盖写库形成静默重 |
| 18 | GA3 | 进程级全局 memory_search 索引被多控制器互相覆盖 | `internal/gaea/tool/builtin/memory_search.go:19-21` | M | play 会话刷新后，work 会话的 memory_search 工具会检索到 play 空间的记忆索引（与 wssearch 里 .gaea/play 硬隔离的 S1.2 双空间红线相反），并存在 goroutine |
| 19 | GA5 | RenderTOML 只渲染 9 段，落盘即丢 14 个配置段 | `internal/gaea/config/render.go:13-225` | L | 任何走 RenderTOML 的保存都会静默把 [memory] enabled=false、[dream] mode、[session] log_format=legacy、[space] mode=off、[sear |
| 20 | GA5 | 一次 persist_allow 审批即触发配置段丢失落盘 | `internal/gaea/boot/boot.go:623-632` | M | 用户在审批卡上点一次「始终允许」，或改一次权限模式，就会连带丢失 memory/dream/session/space/search/network/tasks 等段。今日新加的 P0/P1 特性（space 维度、dr |
| 21 | GA5 | trajectory 折叠 assistantRecord 返回 nil 被解引用 | `internal/gaea/trajectory/fold.go:332-339` | S | 残缺/迁移/被截断的会话日志（首条即为 usage 或 text 增量）会让轨迹看板 panic；Wails 绑定无 recover（本仓既有「recover() 抓不住 map 并发写导致整桌面崩溃」的同类教训），实际 |
| 22 | IN2 | ExtractJSON 取最后一个右花括号，多对象即解析失败 | `internal/util/util.go:82-96` | S | 25 处生产调用点（copilot.go:264、chapter.go:57/171/200、outline.go:173/344/670、whisper/memory_ingest.go:118/291、app/nov |
| 23 | X1 | heredoc 转义坑在册复发 5 次，无任何守卫 | `CHANGELOG.md:2,29,41,918,922,928` | S | 每次复发至少损坏一处测试串或字面量（v4.442『Go 测试串三处断裂』），需重建+复跑+发版；修复动作与『新功能』争夺同一个发版窗口。 |
| 24 | X1 | Windows unlinkat/TempDir 竞态三度复发打挂 CI | `CHANGELOG.md:55,59,922` | M | CI 红灯可信度受损（同一形态两天两度打挂），每个涉及落盘的 app 系测试都有 5~10s 级偶发红；排查每次耗时以小时计。 |
| 25 | X1 | 三重 *core 嵌入导致 nil 嵌入 panic 三犯 | `internal/app/app.go:273-275` | M | 同一类崩溃已三次进 CHANGELOG：v4.406『App 字面量测试缺 mediaState 初始化——嵌入指针 nil…panic』、v4.407『&mediaState{} 裸初始化嵌入 core 为 nil…r |

### 分册 roll-up（单元级合计；去重后的 分册 内部计数见各分册 §六）

| 分册 | 单元数 | 评分均值 | findings | P0 | P1 | P2 | P3 | 分册内拆刀数 |
|---|---|---|---|---|---|---|---|---|
| 01 后端 gaea 内核 | 6 | 4.75 | 111 | 5 | 43 | 51 | 12 | 14 |
| 02 后端应用层·小说与轻语 | 2 | 4.75 | 38 | 2 | 15 | 15 | 6 | 21 |
| 03 后端应用层·办公/造价/进度/基础 | 7 | 4.16 | 131 | 9 | 57 | 47 | 18 | 19 |
| 04 内部领域包与 whisper | 4 | 4.38 | 79 | 1 | 44 | 29 | 5 | 15 |
| 05 前端 React/TS | 7 | 4.71 | 135 | 5 | 47 | 63 | 20 | 21 |
| 06 横切面 | 1 | 5.00 | 18 | 3 | 10 | 4 | 1 | 1 |
