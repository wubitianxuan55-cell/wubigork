# S5 · gaea 办公板块后端能力基线（v4.524.0）

- 审计对象：`C:\AI\wubigrok`（gaea 桌面端，Go + Wails），HEAD `e547f249`，`internal/app/app_info.go:12` `AppVersion = "4.524.0"`。
- 审计方式：**只读**。入口先行（`main.go` → `internal/app` 绑定面 → `internal/gaea/boot` 装配 → `internal/gaea/control` 会话壳 → `internal/gaea/agent` 执行环 → `internal/gaea/tool` 工具面），再按"办公相关"逐包看入口文件与关键路径；不对 13.5 万行逐行通读。
- 规模事实（本机实测，含测试）：`internal/gaea/agent` 175 个 `.go`/27,626 行；`internal/gaea/tool` 88 文件/15,030 行；`internal/gaea/control` 31 文件/7,097 行；`internal/gaea/memory` 64 文件/9,471 行；`internal/app` 为 Wails 绑定层（`gaea_handler.go` 861 行、`bindings_office.go` 222 行是生成物）。
- 未深潜（本文不作断言或标注"未确认"）：`browser/`、`cost*/`、`workcost/`、`sin`（小说/轻语）、`cad/`、`vision/` 的具体实现；前端渲染细节；`provider/` 各家协议差异。凡未读到实现细节者，文中标 **未确认**。
- 一句话基线：**gaea 的"办公后端"= 单进程 Controller → 单模型 Agent 循环 → 工具注册表 + 事件日志磁盘真相源；文档能力的本体是仓库内 `.gaea/skills` 的 Python 技能包（由模型经 `bash`/`run_skill` 驱动）与 Wails 绑定面（供 UI），Go 侧提供"装配、闸门、证据链、回滚、预览、导出"这几层产品化外围，外加只服务 UI 的原生编辑器（`internal/office/{docxedit,pptxedit,xlsxedit}`）。**

---

## 1 运行时骨架图（文字版）

### 1.1 装配（一次 `Build` 定型，之后按需重建）

`App.gaeaBuildController`（`internal/app/gaea_handler.go:108`）→ `boot.Build`（`internal/gaea/boot/boot.go:112`）。顺序如下（每条都是"谁被塞进谁"）：

| # | 动作 | 证据 |
|---|---|---|
| 1 | 办公配置：单 provider `gaea`（kind=`wubigrok` bridge），**ContextWindow 硬写 256000** | `gaea_handler.go:82-90` |
| 2 | sandbox bash 关、写入根=工作空间 | `gaea_handler.go:96-103` |
| 3 | sink 链：`event.Sync` → `session.EventLogSink`（先落盘再转发前端） | `boot.go:167-175`、`agent/session/sink.go:70-93` |
| 4 | `jobs.Manager`（内存态后台作业）| `boot.go:176` |
| 5 | 惰性子代理 store（目录随当前会话解析）| `boot.go:183-192` |
| 6 | 系统提示词装配 `buildSystemPrompt` | `boot.go:213`、`boot/sysprompt.go:43` |
| 7 | 权限策略 `permission.New(mode, allow, ask, deny)` + headlessGate | `boot.go:271-273` |
| 8 | hook runner（settings.json，项目需 trusted）| `boot.go:286-294` |
| 9 | 内置工具按空间注册 `addBuiltins` | `boot.go:238`、`boot.go:799`、`tool/builtin/spacetags.go:120` |
| 10 | `task` 子代理工具 + interrupt/list/read_result + session_search | `boot.go:301-352` |
| 11 | 记忆工具（remember/forget/promote/memory_get）| `boot.go:380-383` |
| 12 | 会话级元工具 ask / request_permission / exit_plan_mode | `boot.go:389-405` |
| 13 | `run_skill` / `install_skill` / `summarize_file` / `slash_command` | `boot.go:471-485`、`boot.go:584` |
| 14 | 前端注入工具 ExtraTools（image_gen / diagram / routine_llm / translate_text / fact_* / dag_plan / ocr / semantic_search / author_memory）| `gaea_handler.go:168-178`、`app/gaea_specialist_tools.go:18-24`、`boot.go:589-593` |
| 15 | 主执行器 `agent.New(..., Gate/AccessPolicy/Jobs/Compaction/Dispatcher/JournalDir/Spill)` | `boot.go:507-523` |
| 16 | TCCA 上下文核（Identity/Runtime/Skill/Flow 四层）| `boot.go:617-628`、`context/manager.go:20` |
| 17 | `control.New(Options{... Policy/Registry/CtxMgr/WorkspaceRoot})` | `boot.go:646-692` |
| 18 | 桌面端把 headlessGate 换成交互闸 + 接 Asker/Requester | `gaea_handler.go:202` → `control/controller.go:714-724` |

**系统提示词拼装点**（办公提示词的唯一真相在 Go 常量里，`prompts/` 目录全是小说模板）：

1. `cfg.ResolveSystemPrompt()` → 缺省 `config.DefaultSystemPrompt`（`internal/gaea/config/config_paths.go:14`，办公角色/原则/办公规范/本地工具/子代理/记忆六段）
2. `+ outputstyle`（`sysprompt.go:51`）`+ LanguagePolicy`（`:54`，`config_paths.go:67`）`+ genui.OfficePointer`（`:55`，`genui/handbook.go` embed `officepointer.md`）
3. `+ memory.Compose`（`:70`，画像 + 文档索引）
4. `+ 晨报预载`（`:81`，work 空间）、`+ 项目本体 `[MEM:]``（`:89`，work 空间）、`+ 固化事实正文`（`:100`）、`+ 常用资料 pins`（`:106`）
5. `+ skill.ApplyIndex`（`:134`，技能索引）
6. 执行器 system = 上面前缀 `+ "\n\n" + agent.SingleModelPromptFor(space)`（`boot.go:506`、`agent/single_prompt.go:89`）——后者是"规划→执行→验证→签退 / 事实底座 / 办公质量检查点 / 输出文件约定 / 落盘规范"（`single_prompt.go:15-71`）

> 两个细节值得注意：①`complete_step` 签退纪律同时写在基础提示词（`config_paths.go:28`）和执行纪律（`single_prompt.go:22`）；②提示词把 docx/xlsx/pdf/pptx 全指向 `run_skill`（`config_paths.go:35`、`single_prompt.go:20,45`），见 §8 缺口 1。

### 1.2 一次办公对话的完整事件链（编号=时序）

```
UI(Wails) → OfficeB.GaeaSend → Controller.Send/SendWithRaw → turnLoop(goroutine)
  → runTurnWithRaw: Compose(@refs/记忆/后台回执) → hooks.PromptSubmit
  → [event 模式] flushCheckpoint(fail-closed) + logUserMessage
  → AgentRunner.Run → runDirect: 循环{ stream → 工具批 → 结果入会话 }
  → 无工具调用: taskGate/goalGate/finalReadinessCheck → TurnResult
  → flushJournal(证据卡 JSONL) + Snapshot(.jsonl + checkpoint) + TurnDone
```

逐步证据：

| 步 | 事件 | 证据 |
|---|---|---|
| 1 | 前端绑定 → 控制器 | `app/bindings_office.go:163` → `App.GaeaSend` → `control/controller.go:387`(`Send`) / `:400`(`SendWithRaw`) |
| 2 | 回合在跑则入队（上限 8 条，满则拒绝并 notice）| `controller.go:183`、`:406-418` |
| 3 | 起回合：`launchTurnLocked` 起 goroutine，`running=true` | `controller.go:333-339`、`:346`(`turnLoop`) |
| 4 | 解析 `@` 引用 → 组装输入 | `controller.go:427-444` |
| 5 | 回合状态 sidecar 置 running（崩溃残标 = "上次未完成"）| `controller.go:470-485`、`agent/session/state.go:23-58` |
| 6 | 首轮：TCCA `ProcessFirstTurn` + 运行时上下文合并进 L1 末尾 | `controller.go:491-503`、`agent/agent.go:546` |
| 7 | `Compose`：memory-update / session-facts / background-jobs / 记忆检索注入（**只加到 user 消息，不动缓存前缀**）| `controller/input.go:14` 起 |
| 8 | hooks（PromptSubmit 可阻断；Stop 收尾）| `controller.go:511-520` |
| 9 | event 模式：模型调用前 checkpoint flush（失败即中止回合）| `controller.go:524-533`、`agent/agent.go:568` |
| 10 | 进入 `runDirect`：trace/space/archive 盖章、evidence 重置、Journal defer、`TurnStarted` | `agent/agent_run.go:25-58` |
| 11 | 用户消息入会话 | `agent_run.go:65` |
| 12 | 主循环：consumeSteer → `midTurnMaybeCompact` → `Phase:思考中` → `stream` | `agent_run.go:125-149` |
| 13 | `stream`：组 system/tools → 发 `RequestHeader`（system+tools hash/前缀稳定判定）→ `prov.Stream` 流式 | `agent/agent_stream.go:29-74` |
| 14 | 流中：工具调用一开始就发半开卡片；**只读工具在参数流完即并发预执行** | `agent_stream.go:107-145` |
| 15 | usage 到手 → soft/ratio/force 三档压缩判定 | `agent_run.go:221-275`、`agent/compact.go:106-230` |
| 16 | assistant 消息（含 reasoning/tool_calls）入会话 + archive | `agent_run.go:281-300` |
| 17 | 有工具调用 → `executeBatch`：参数修复 → 参数风暴熔断 → `ToolDispatch` 事件 → 只读批并行（≤8）/写批串行 | `agent_run.go:377`、`agent/batch_executor.go:17-108,291-293` |
| 18 | 单次调用 `executeOne`：调度器检查（hook/gate）→ 路径访问策略 → 确定性预检查 → stale 守卫 → 读缓存 → ctx 盖章（sink/asker/permReq/ledger/journal/jobs/memory/spill）→ 执行 → 证据记账 → 结果压缩/泄洪 → 信封 | `agent/execute_one.go:21-365` |
| 19 | 结果入会话（tool 消息）→ complete_step 推进 canonical todo → 失败螺旋/重复/后台启停 检测与 nudge | `agent_run.go:381-470` |
| 20 | 无工具调用的终局：空答重试(3) → taskGate → goalGate → 证据就绪闸（未验证 todo）→ 返回 `TurnResult` | `agent_run.go:302-363`、`agent/agent.go:997-1018` |
| 21 | 收尾：`flushJournal`（证据卡 JSONL + markdown 投影）、mt_ 记录兜底、steer 队列 drain | `agent_run.go:38-57`、`:541-577` |
| 22 | 控制器收尾：记忆引用触达 → `Snapshot`（整文件 JSONL 原子写 + checkpoint）→ `TurnDone` | `controller.go:537-548`、`control/controller_session.go:263-288` |
| 23 | 宿主在 TurnDone 后跑"自动做梦"与自动命名 | `app/gaea_handler.go:118-123` |

### 1.3 谁拥有状态（关键：没有第二个权威）

| 状态 | 拥有者 | 生命周期 | 落盘 |
|---|---|---|---|
| 会话消息数组（模型可见面） | `agent.Session`（`AgentRunner.session`）| 进程内，回合中追加 | `session.Save` 整文件 JSONL 原子重写（`agent/session/save.go:26-40`）|
| 回合状态（running/turn/sessionPath/pendingSends/granted/approvals/asks）| `control.Controller`（`controller.go:41-148`，`mu` 保护）| 进程内 | `.state.json` sidecar 只记 running/摘要（`session/state.go:23`）|
| 事件日志（"模型可见必入日志"的真相源）| `session.EventLogSink`（append-only，seq=行数）| 磁盘 | `<id>.gaea-log.jsonl`（`session/log.go:181`）|
| 检查点（消息投影 + 已消费 seq）| Controller `flushCheckpoint` | 磁盘 | `<id>.gaea-checkpoint.json`（`log.go:194`、`controller_session.go:215-238`）|
| 待审批/待回答请求 | Controller `approvals`/`asks` map + 阻塞 goroutine | 进程内，重启即失 | 无 |
| 会话级权限授予 | Controller `granted`/`grantedRules` | 进程内，重启即失（除 `persist_allow` 回写配置）| 配置文件（`controller_approval.go:362-375`）|
| 子代理转录 | `agent.SubagentStore`（`subagent_store.go:146`）| 磁盘 | `<sessionDir>/subagents/sa_*.jsonl` + `.meta.json` |
| 证据卡/裁决 | `evidence.JournalStore` | 磁盘 | `<cwd>/.gaea/work/journal/*.jsonl`、`verdicts.jsonl` |
| 产物登记（只读折叠）| `trajectory.FoldDeliverables` 纯函数 | 派生 | 无独立存储（读事件日志）|
| 交付事实底座 | `factbase.Store` | 磁盘 | `<session>-facts.json`（`factbase/factbase.go:111`）|
| 后台作业 | `jobs.Manager` | 进程内，Close 即取消 | 无（`jobs/jobs.go:463`）|
| 长任务（抓价/索引）| `tasks.Manager` | 磁盘（sqlite）| `Hephaestus.db` tasks 表（`tasks/tasks.go:378`、`db/database.go:21`）|

---

## 2 工具清单（办公相关，按注册面枚举）

审批列口径：`只读免审` = `ReadOnly()==true` 且无硬规则时 `Policy.Decide` 直接 Allow（`permission/permission.go:114-128`）；`写类=Mode` = 写工具落 `Policy.Mode`，默认 `"ask"` 弹卡（`internal/gaea/config/config.go:810`：`Mode:"ask", Allow:["run_skill"]`）；`hard-ask` = 任何权限级别逐条确认（`control/controller_approval.go:21-28`）。

### 2.1 文件 / 命令 / 编辑（办公落盘主线）

| 工具 | 作用 | 参数要点 | 可写盘 | 审批 |
|---|---|---|---|---|
| `read_file` | 读文本（带行号/偏移/上限 2000 行）| `path, offset, limit, line_numbers`（`tool/builtin/readfile.go:45-54`）| 否 | 只读免审；结果进回合内缓存（`execute_one.go:134-144`）|
| `write_file` | 整文件覆盖写入（保留原编码/权限，父目录自动建）| `path, content`（`writefile.go:34-36`）| 是（受 write roots confine，`:68`）| 写类=Mode；写前快照+Journal 证据卡（`:80-100`）|
| `edit_file` | 锚点替换 | `path, old_string, new_string`（`editfile.go:35`）| 是 | 写类=Mode；证据卡（`:100`）|
| `edit_lines` | 行区间替换（1-based，空内容=删除）| `path, start_line, end_line, new_content`（`editlines.go:32-41`）| 是 | 写类=Mode；证据卡（`:121`）|
| `multi_edit` | 一次多编辑片段 | `edits[]`（`multiedit.go:32`）| 是 | 写类=Mode；证据卡（`:110`）|
| `move_file` | 移动/改名 | `source, destination`（`movefile.go`）| 是 | 写类=Mode；证据卡（`:130`）|
| `ls` / `grep` | 列目录 / 内容检索 | `path` / `pattern` | 否 | 只读免审 |
| `bash` | shell（前台/后台、plain/json 输出）| `command, run_in_background, output_format`（`bash.go:70`）| **是（任意写）** | 写类=Mode（ReadOnly=false，`bash.go:76`）；只读命令另有识别旁路（`permission/bash_readonly.go`）|
| `bash_output` / `kill_shell` / `wait` | 后台作业读输出/终止/等待 | job id | 否 | 写类=Mode（`bgjobs.go:35,115,168`）|
| `read_spill` | 取回泄洪的大工具结果全文 | spill id（`readspill.go:18`）| 否 | 只读免审 |
| `format_convert` | docx/xlsx/pptx/pdf → Markdown（PDF 支持页范围）| `path, output, pages`（`format_convert.go:27-37`）| 条件（给了 `output` 才写）| 写类=Mode（`ReadOnly()==false`，`:40`）|
| `summarize_file` | 大文件分块摘要（模型支撑）| 见 `largefile/tool.go:32` | 否 | 写类=Mode；被标 ModelBacked，落 mt_ 记录 |

### 2.2 文档/图表/检索/图像（办公产出与取证）

| 工具 | 作用 | 参数要点 | 可写盘 | 审批 |
|---|---|---|---|---|
| `chart_gen` | 统计图（bar/line/pie/scatter → png/svg，走 python）| `type, ... , output`（`chart_gen.go:35`）| 是 | 写类=Mode |
| `diagram_gen` | Mermaid 渲染成 png（内置版，python）| `type, topic, output`（`diagram_gen.go:38`）| 是 | 写类=Mode |
| `diagram` | **App 版**：LLM 生成 Mermaid 存 `.mmd`（`app/gaea_diagram.go:120`）| `type, topic, extra` | 是 | 写类=Mode |
| `vision` | 图片理解（本地视觉模型）| `image_path`（`vision.go:18`）| 否 | 只读免审 |
| `ocr` | 图片/扫描件文字提取（本地 OCR 服务）| `image_path, language`（`app/gaea_specialist_tools.go:39-47`）| 否 | 只读免审 |
| `screen_capture` | 截屏 | —（`screenshot.go:25`）| 是（系统路径，.gaea/uploads）| 写类=Mode |
| `web_search` / `web_fetch` | 联网检索/抓页 | query / url（`websearch.go:59`、`webfetch.go:39`）| 否 | 只读免审 |
| `semantic_search` | 跨库/工作区语义检索（scope=all/file）| `query, scope`（`gaea_specialist_tools.go:112-121`）| 否 | 只读免审 |
| `knowledge_search` / `knowledge_add` | 工程知识库 | — | add 写 | add=hard-ask（`controller_approval.go:24`）|
| `memory_search` / `memory_get` / `remember` / `forget` / `promote_session_facts` | 跨会话记忆 | —（`memory/remember.go:29` 等）| remember/forget/promote 写 | **hard-ask** 三项（`:22-27`）|
| `fact_add` / `fact_list` / `fact_clear` | 会话级"事实底座"（交付物统一口径）| `key, value, source, category`（`app/gaea_factbase.go:78`）| 是（`<session>-facts.json`）| 写类=Mode（`fact_add/clear` 非只读）|
| `routine_llm` / `translate_text` | 本地常规模型文本处理 / 翻译 | —（`app/gaea_routine_llm.go:23`、`app/gaea_translate.go:295`）| 否 | 写类=Mode（保守）|
| `cost_search` / `cost_compose` / `cost_indicators` / `cost_save` | 成本库检索/组价/指标/写入 | — | save 写 | save=**hard-ask**（`:23`）|
| `schedule_get` / `schedule_analyze` / `schedule_apply` | 进度计划读/分析/写（`project` 整量通道最危险）| — | apply 写 | apply 的 project 通道=**任何级别强确认**（`controller_approval.go:51-53`）|

### 2.3 编排 / 元工具（办公流程骨架）

| 工具 | 作用 | 参数要点 | 可写盘 | 审批 |
|---|---|---|---|---|
| `task` | 派发隔离子代理（前台/后台/结构化 JSON/重试/续跑/fork）| `prompt, description, write_paths, tools, max_steps, run_in_background, output_schema, retry_until, continue_from, fork`（`agent/task.go:275-291`）| 间接（子代理可写）| 写类=Mode（`task.go:297`）|
| `run_skill` / `read_skill` | 按名执行技能（inline 或子代理）/ 按需读技能正文 | `name`、`task`（`skill/tools.go:38`、`readskill.go:60`）| 间接 | run_skill 在默认 allow 白名单里（`config.go:810`）；写盘发生在子任务内部仍受闸门 |
| `install_skill` | 安装/落盘技能 | —（`skill` 包）| 是 | **hard-ask**（`:28`）|
| `slash_command` | 自定义命令/技能按名展开 | `name, args`（`command/slashtool.go:56`）| 间接 | 写类=Mode |
| `ask` | 结构化多选提问（阻塞等答）| `questions[1-4]{header,question,options[2-4],multiSelect}`（`agent/ask.go:31-63`）| 否 | 只读免审（`ask.go:68`）|
| `request_permission` | 申请权限规则（"Tool"/"Tool(glob)"）| `tool, subject, reason`（`agent/request_permission.go:42`）| 否 | 自身即审批通道（`controller_approval.go:167-204`）|
| `exit_plan_mode` | 计划模式提交计划待批 | plan markdown（`agent/plan_mode.go:63`）| 否 | 走 ask 通道 |
| `todo_write` / `complete_step` | 计划清单 / 签退（证据匹配）| `todos[]` / `step`（`builtin/todo.go:32`、`completestep.go:41`）| 否 | 只读/写类混合（todo 写会话态）|
| `interrupt_agent` / `subagent_list` / `read_subagent_result` | 中断/枚举/取回子代理 | `ref`（`agent/interrupt_tool.go:30` 等）| 否 | 写类=Mode（interrupt）|
| `dag_plan` | 把多文件任务规划成流水线（只规划不执行）| `goal, run_id, nodes[{id,title,prompt,depends_on,risk}]`（`app/gaea_dag.go:44-59`）| 是（`.gaea/work/dag/<id>.json`）| 写类=Mode；PersistWrite（子代理不可见）|
| `genui_validate` | 校验 GenUI 围栏 JSON | spec（`genui_validate.go:22`）| 否 | 只读免审 |
| `session_search` | 过往会话全文检索（FTS5）| query（`sessionquery/tool.go:25`）| 否 | 只读免审 |
| `browser_*`（11 个）| 浏览器观察/操作 | url/snapshot 等 | 间接（网页可上传）| shared 空间，写类=Mode |

> 注：`complete_subtask` 只注册进子代理注册表（`agent/task.go:520`），父代理不可见。

---

## 3 上下文与压缩

### 3.1 分层与阈值

- 上下文窗口：桌面办公 provider 硬写 **256,000** token（`app/gaea_handler.go:89`），boot 里 `entry.ContextWindow==0` 时兜底 1,000,000（`boot.go:154-156`）。
- 三档水位常量：soft `0.5` / ratio `0.8` / force `0.9`，目标折叠比 `0.5`，尾预算 `16384` token，最少保留 5 条消息（`agent/compact.go:34-42`）；配置可覆写（`config/config.go:614-616`、`boot.go:951-958`）。→ 默认实际触发点 ≈ 128k / 205k / 230k。
- 触发点四处：①采样后真实 usage（`compact.go:106-127`）；②回合内两次采样之间用估算值（`compact.go:136-150`）；③provider 报上下文溢出后的自愈（`compact.go:249-283`）；④手动 `/compact`（`compact.go:412`）。
- 判定目标：`prompt` 优先取 `usage.PromptTokens`，缺失时回退 `compaction.LastPrompt`（`compact.go:110-119`）。

### 3.2 一次压缩做什么（顺序即优先级）

1. **soft 通知**：只发 notice，不动消息，一次性（`compact.go:161-167`）。
2. **剪枝**：`PruneStaleToolResults`（旧工具结果 elide + 归档），免费，剪完若低于水位就返回（`compact.go:189-197`；实现在 `agent/prune.go`）。
3. **经济学预判**：估算"最省摘要后仍≥触发水位"则直接跳过；固定前缀已超 50% 窗口时要求 ≥25% 节省（`compact.go:321-344`）。
4. **原文归档**：折叠区先写 archive JSONL（`compact.go:351-357`），摘要末尾回填归档路径（`:366-370`）。
5. **LLM 摘要**：`summaryInstruction`（`compact.go:73-98`）要求按 7 个固定标题产出（Standing facts / Goal / Decisions / Files & code / Commands & outcomes / Errors & fixes / Pending），明确"UI 围栏是组件不是内容，不得回写 JSON"；失败或超时（90s，`:67`）→ 机械摘要兜底（`:359-364`）。
6. **shrink 硬校验**：摘要体积必须小于被折叠区间，否则退回机械摘要（`compact.go:376-386`）。
7. **重写会话**：`system + 首个 user + 旧 digest + 新 digest + 尾部`（`compact.go:286,388-396`），`RewriteVersion++`（缓存前缀作废）。
8. **熔断**：连续 2 次压缩仍压不下去 → `compactStuck` 暂停自动压缩；会话长度变化即自动解锁（`compact.go:211-229`、`:174-184`）。

### 3.3 连续性怎么保持

| 机制 | 说明 | 证据 |
|---|---|---|
| 固定前缀 | system(L1/L2) + 首个可 pin 的 user 轮 + 既有 digest 恒保留 | `compact.go:548-574`（首个 user 上限 1500 token 或窗口 15%）|
| 用户轮逐字保留 | 按确定性预算保留用户原话，超出则计入 `Dropped` 并显性告警 | `compact.go:594-651`、`:401-407` |
| `[[keep]]` 标记 | 用户可用 `[[keep]]/[keep]/<keep>` 钉住一轮 | `compact.go:714-745`、`KeepPolicy`(`agent.go:197-210`) |
| KeepErrors / KeepProtected | 失败输出与 `read_skill/memory_search` 等基础上下文不被摘要 | `compact.go:653-666`、`agent.go:197-210` |
| 渐进注入 | 压缩前把 `progress.md` 计划进度塞进摘要指令的 Pending 段 | `compact.go:200-204` |
| 中断交接 | 流错误/预算阻断/就绪闸失败 → 折叠回执注入下一条用户消息 | `agent/interrupted_recovery.go`、`agent_run.go:61-63` |
| 尾预算下限 | 尾保留 ≥ max(25% 窗口, 2000)，上限 8000 token | `compact.go:487-500` |
| 定量计费 | 锚定式 token meter（真实 usage 锚 + 表面差值） | `agent/tokenmeter.go`、`agent.go:663-671` |

### 3.4 上下文视图如何统计

- 纯函数折叠：`contextview.FoldTimeline(entries, window, retention)`（`contextview/fold.go:32`），retention 默认 200 次请求（`types.go:7`）。
- 六分类：`system / tools / user / inject / assistant / tool`（`types.go:10-22`）；每次请求一根趋势柱（`RequestRecord`，`types.go:37-70`），带 `Delta`（对比上一步）与 `Prefix`（相邻请求前缀稳定判定：stable/prefixBytes/appendCount/rewriteCount/hash）。
- 数据来源是 `request_header` 事件（`agent_stream.go:54-68` → `session/log.go:111-123`）+ `usage` + `tool_dispatch/tool_result`；上下文事件（inject/compact/prune/switch/mode）单列（`types.go:102-111`）。
- 另有"模型可见 surface 节点"（seq/cat/tokens/预览/Gone=被取代的 seq）供浏览器与归档回放（`types.go:113-120`）。
- 前端绑定：`GaeaContextView` / `GaeaContextNodeDetail`（`bindings_office.go:49,48`）；轨迹/产物折叠同源（`trajectory/fold.go`、`trajectory/deliverable.go:46`）。

---

## 4 权限 / 审批

### 4.1 判定逻辑（三层）

1. **纯策略** `permission.Policy.Decide`：优先级 `deny > ask > allow > (只读→Allow) > Mode`（`permission/permission.go:114-128`）。
2. **Gate.Check**：`AlwaysAsk` 集合优先（非交互时仍放行以保自治）；`bash` 非只读但命令属只读白名单时降级为只读；`Ask` 且有 Approver 时走审批（`permission.go:257-299`，bash 只读识别 `permission/bash_readonly.go`）。
3. **Controller gateApprover**：`auto = autoApprove || permLevel!="ask"`；`hardAsk` 工具与 `schedule_apply` 的 project 通道无论级别都弹卡；其余 auto 直接放行（`control/controller_approval.go:34-58`）。

### 4.2 白名单与强制集

- 缺省策略：`mode="ask"`、`allow=["run_skill"]`（`config/config.go:810`）→ **默认每个写工具（write_file/edit_file/bash/task/format_convert…）都会弹一张审批卡**。
- `hardAskTools`（逐条确认、不记会话放行、不自动）＝`cost_save / remember / forget / knowledge_add / promote_session_facts / install_skill`（`controller_approval.go:21-28`）；可被空间策略参数化（`boot.go:664`、`controller.go:729-734`）。
- 规则形态：`Tool` 或 `Tool(subject-glob)`，glob 支持 `*`/`?` 且跨分隔符（`permission.go:63-76,199-223`）。
- subject 提取键序：`command > file_path > path > pattern > source > url`（`permission.go:171`）——bash 看命令、文件工具看路径。
- 路径边界是独立一层：`tool.AccessPolicy`（read roots/write roots/显式选文件）在 gate 之后、执行之前生效（`boot.go:227`、`execute_one.go:94-100`、`tool/access.go`）。
- 会话级授予：`granted`（精确 tool+subject）与 `grantedRules`（request_permission 批的 glob 规则），重启即失（`controller.go:109-115`）。

### 4.3 UI 往返协议（事件 + 绑定方法）

```
agent → gate.Check → gateApprover.Approve → requestApproval → promptApproval
      → sink.Emit(ApprovalRequest{ID, Tool, Subject, Reason, Request})
      → [前端弹卡] → OfficeB.GaeaApprove(id, decision)
      → Controller.Approve: reply chan → 阻塞方继续
```

- 请求事件字段：`ID/Tool/Subject/Reason/Request`（`controller_approval.go:263-266`）。
- 决策串 6 种：`allow_once / allow_session / persist_allow / deny / abort / timeout`（`controller.go:152-161`）。
  - `persist_allow`：先记会话 granted，再经回调回写策略文件（`controller_approval.go:297-314`、`boot.go:670-679`）；hard-ask 下退化为 `allow_session`（`:300-303`）。
  - `abort`：拒绝并 `Cancel()` 整个回合（`:291-296`）；`deny`：只拒这一次、回合继续（`:337-339`）；`timeout`：按拒绝处理并发 notice（`:345-355`，超时由 `ApprovalTimeout` 配置）。
- `ask` 通道同构：`AskRequest` 事件 → `GaeaAnswer(id, answers)`（`controller.go:739-777`、`bindings_office.go:28`）。
- 串行约束：`promptMu` 保证**同一时刻只有一个待答弹窗**（工具审批与 ask 共用）：字段在 `controller.go:99`，审批侧加锁在 `controller_approval.go:245-246`，ask 侧加锁在 `controller.go:739-741`。
- 自动放行的两个来源：`permLevel != "ask"`（`GaeaSetPermLevel`，`controller.go:780`）与"刚批准的 plan"（`autoApprove`，`controller_approval.go:38-40`）。

---

## 5 会话与任务

### 5.1 存储形态（**不是 sqlite**；sqlite 只管任务/记忆/知识/成本）

| 文件 | 内容 | 证据 |
|---|---|---|
| `.gaea/sessions/[<space>/]<id>.jsonl` | 会话消息整文件投影（每次 Save 全量重写，原子写）| `agent/session/save.go:26-40`、`config/config_paths.go:115` |
| `<id>.gaea-log.jsonl` | append-only 事件日志（seq=行数，单次 Write 追加，非法 JSON 拒写，torn-tail 自动修复）| `session/log.go:4-9,226-316` |
| `<id>.gaea-checkpoint.json` | 消息投影 + 已消费 log seq + space | `log.go:194-204`、`controller_session.go:215-238` |
| `<id>.state.json` | running/摘要/时间（中断标记）| `session/state.go:23-28` |
| `<id>-facts.json` | 事实底座 | `factbase/factbase.go:111-122` |
| `subagents/sa_*.jsonl` + `.meta.json` | 子代理转录 + 侧车 | `subagent_store.go:764-820` |
| `.gaea/work/journal/*.jsonl` + `verdicts.jsonl` | 证据卡 + 复核裁决 | `evidence/journal.go:205-260,321` |
| `.gaea/work/dag/<id>.json` | 文件流水线运行档 | `app/gaea_dag.go:3-8` |
| `Hephaestus.db` | tasks / facts(记忆) / 知识库 / 成本库（WAL，Schema 迁移到 V21）| `db/database.go:21-28`、`db/schema.go` |

双格式并存：`log_format` 缺省 `event`（事件日志 + checkpoint），可退回 `legacy`（纯 jsonl，`controller_session.go:127-151`）；事件模式下 `Save` 仍写 legacy 镜像（`save.go:46-70`）。

### 5.2 恢复能力

- 正常恢复：`ResumeFromDisk` → 旧格式先迁移 → `LoadWithFormat("event")` = `Restore(checkpoint, log)`（checkpoint 消息 + seq 之后 tail 重放，torn-tail 修复）→ `Resume` 注入执行器并续接 checkpoint flusher（`controller_session.go:97-125`、`save.go:101-115`）。
- 崩溃恢复：`state.json` 的 `running=true` 残留可被识别（`session/state.go:1-5`）；子代理侧 `CleanupStaleRunning` 把 running/queued 判 failed（`subagent_store.go:902`）。
- 回合中崩溃最多丢一个在飞回合：每回合结束 `Snapshot`（`controller.go:538-548`），模型调用前 checkpoint 是 **fail-closed**（失败即中止回合，`agent.go:565-573`）。
- 未持久化的诚实边界：中断交接记录（`pendingRecovery`）、steer 队列、待审批请求、后台作业均为内存态，进程重启即失（`agent.go:425-429` 注释明确写为"诚实边界"）。

### 5.3 并发 / 排队 / 后台

| 维度 | 规则 | 证据 |
|---|---|---|
| 会话回合 | 单飞：`running` 标志 + `tryStartTurn` 原子起转 | `controller.go:319-327` |
| 用户消息排队 | 回合中入 `pendingSends`，上限 **8**，FIFO 逐条起新回合并各发 TurnDone | `controller.go:181-183,346-370,406-418` |
| 回合内插话（steer） | 队列 + 终结前关门 drain，未消费落会话+Notice 不静默丢 | `agent/agent.go:946-992`、`agent_run.go:35-41` |
| 工具批并行 | 只读批并行，上限 **8**；写/混合批串行；流式期间只读工具已并发预执行 | `batch_executor.go:100-108,291-293`、`agent_stream.go:118-145` |
| 子代理并行 | 专用信号量，默认 **3**、上限 8，超出 FIFO 等槽（前端显示 queued） | `subagent_limit.go:29-36,48-62` |
| 后台作业（bash/子代理） | `jobs.Manager` 内存态；完成回执下回合以 `<background-jobs>` 注入 | `jobs/jobs.go:126-158,463`、`control/input.go` |
| 长任务（持久） | `tasks.Manager`：sqlite 表 + worker + 空间配额 + 优先级 + 指数退避重试 + 重启续跑 | `tasks/tasks.go:378,798-818,820-929` |
| 任务种类（当前仅 4 类） | `price_fetch / price_fetch_all / file_index / outline_reconstruct` | `tasks/tasks.go:28-33`、`app/gaea_tasks.go:50-53` |

---

## 6 子代理

- **派生方式**：`task` 工具（`agent/task.go:268`）。同一消息里多个 task 调用真并行（受子代理信号量），严格有序步骤要求写进同一 prompt（`task.go:271-272` 描述）。
- **隔离**：
  - 独立 `Session`，**不继承父 L1+L2**，用 `DefaultTaskSystemPrompt`（`task.go:26,783-786`），可省约 5 万 token；
  - 子注册表 = 父注册表 ∩ `tools` 白名单 − `SubagentMetaTools`（task/run_skill/install_skill/interrupt/request_permission/exit_plan_mode 等，`task.go:126-158,510-577`），保证**委托只有一层**；
  - `fork=true` 时以父会话消息做种子（前缀字节一致以吃 KV 缓存），与 background/continue_from/retry_until 互斥（`task.go:346-361,865-867`）；
  - 空间继承父会话（`WithSpace`/`SpaceFromContext`，`agent.go:104-126`）。
- **模型与成本**：可整体换子代理 provider（`SetSubagentProvider`，`boot.go:324-335`），也可 per-skill 覆盖（`boot.go:424-430`）；用量累加回父（`mergeSubUsage`，`task.go:588`）。
- **回传**：只有最终答复进父上下文；完成时额外发一条 `SubagentMessage` 事件供 UI 显示（`task.go:892-904`）；结果包 `<task_result>` 壳（`task.go:54`）；`output_schema` 不合法则在**同一会话内**纠偏重入（≤2 次）后如实降级（`task.go:868-891`）；子代理若提交过结构化报告，父侧附"完成裁决"（`childCompletionVerdict`，`task.go:918-922`）。
- **落盘与续跑**：`subagents/sa_<ts>_<rand>.jsonl` + `.meta.json`（状态/标题/工具面/模型/空间/出发快照，`subagent_store.go:61-108`）；运行中 1s 快照（`:692-733`）；`continue_from` 续跑（`:465-531`，单飞 claim 防双跑 `:152-157`）；崩溃清理（`:902`）。
- **反向通道**：父可用 `read_subagent_result` 分页取回最终答复、`interrupt_agent` 中断、`subagent_list` 枚举（`boot.go:337-346`）；用户可在子代理 tab 追问（Side Chat，`boot.go:357-370`）。
- **DAG 流水线**：`dag_plan` 只规划（`app/gaea_dag.go:30-42`），执行由宿主按波次逐节点 `TaskTool.RunNew` 起**全新子代理**（`gaea_dag.go:3-8`），节点产物按 `sa_` ref 归因。

---

## 7 技能与记忆（含 evidence / trajectory）

### 7.1 技能加载

- 发现根按优先级：`project > custom > global > builtin`（`skill/skill.go:26-35`），扫描 `.gaea/.agents/.agent/.claude` 四个约定目录下的 `skills/`（`skill.go:157-182` + `config/config_paths.go:152` `ConventionDirs`）。
- 解析：`SKILL.md` 目录式或 `<name>.md`；frontmatter 支持 `allowed-tools`（子代理技能的工具白名单）、`run-as`（inline/subagent）、`model`（`skill.go:54-66,311-340,426-459`）；正文支持引用式加载（`loadBodyWithReferences`，`:388`）。
- 内置 8 个技能（`skill/builtins.go:21-241`）：`office-gongwen`（GB/T 9704 公文）、`genui`、`cad-takeoff`、`office-edit`（xlsx/docx/pptx/gbase 编辑纪律）、`schedule-edit`（进度计划 + CPM）、`format-convert`、`chart-builder`、`doc-assemble`。
- 入口收敛为两级：`read_skill`（按需读正文折进当前回合，`builtin/readskill.go:60`）与 `run_skill`（inline 折入 / subagent 隔离执行，`skill/tools.go:38`、`boot.go:416-470`）；顶层包装工具（format-convert/chart-builder/doc-assemble）已删除（`boot.go:407-415` 注释）。
- **磁盘事实（关键，实测绘）**：项目约定目录 `.gaea/skills/` 已 git 跟踪 158 个文件，内容为完整 Python 技能包——`docx`（62 文件：`scripts/create_docx.py`、`scripts/office/{pack,unpack,soffice,validate}.py`、OOXML XSD 校验器与模板）、`xlsx`（58）、`pdf`（14）、`pptx`（2：`SKILL.md` + `scripts/create_pptx.py`）、`gongwen-9704`（8）、`cost-compose` / `schedule-dcma` / `skill-creator` / `novel-agents|deslop|review`；仓库根 `skills/` 只是额外两个小说技能（`story-deslop`、`story-long-write`）。内置技能表 8 项（`builtins.go`）与这些文件技能**不重复**，同名时项目/用户文件优先。见 §8 缺口 1。

### 7.2 记忆注入

- 存储：sqlite `facts` 表（`memory/sqlite.go:11-18`，库 `Hephaestus.db`），按项目 slug + 空间（work/play）隔离。
- 注入（全部在**缓存稳定前缀**里，只在装配期发生一次）：`memory.Compose`（画像 + 文档索引，`sysprompt.go:70`）、晨报预载（`:81`）、项目本体 `[MEM:]`（`:89`）、固化事实正文（`:100`）、常用资料 pins（`:106`）。
- 会话中新增记忆**不进前缀**：折叠成 `<memory-update>` 附在下一条 user 消息（`control/input.go:14-30`），下个会话才并入前缀。
- 会话级临时事实：`remember(session=true)` → 每回合以 `<session-facts>` 重注入（`input.go` 同段）。
- 回传侧：回复里的 `[MEM:key]` 被解析并 touch（`controller.go:561-571`）；出口闸会剥离悬空引用键（`boot.go:531-544`、`agent_stream.go:201-207`）。
- 自动"做梦"：每成功回合后台整理记忆（`app/gaea_handler.go:118-123` → `app/gaea_dream.go`），产出必须过建议闸。

### 7.3 evidence / trajectory 各干什么

| 包 | 职责 | 证据 |
|---|---|---|
| `evidence`（回合内回执）| `Ledger` 记录本回合每次工具调用回执（工具名/参数/成败/paths/write），供 `complete_step` 校验"这一步真的有工具证据"；`strictVerify` 只在 Plan Mode 开 | `evidence/evidence.go:32-127` |
| `evidence`（变更台账 + Journal）| `ChangeLedger` 回合内收集 → 回合末 `flushJournal` 写 `journal/*.jsonl`（并导出 markdown 投影）；**只在 work 空间落盘**（play 红线丢弃）；`StageBaseline` 写整文件 `.before` 快照供回滚/视觉复核 | `evidence/journal.go:35-138,191-260`、`agent_run.go:541-577` |
| `evidence`（复核）| `Verdict` 双通道：A 结构/引用（xlsx 重算 + ops 声明比对、move 源/目标、文本 before/after 摘要比对），B 视觉（soffice→PDF→pdftoppm→像素差，阈值 2%/20%），渲染不可用降级 warn | `app/gaea_verify.go:75-235` |
| `evidence`（回滚）| 基线快照原子写回；"目标已被手工修改"守卫拒绝覆盖；回滚动作自身再生成一张带快照的证据卡（可撤销恢复）| `app/gaea_verify.go:238-316` |
| `trajectory` | 事件日志 → 轨迹/时间线/产物登记/Agent 网络（纯函数折叠，可黄金测试）| `trajectory/fold.go`、`deliverable.go:46-57`、`agentnet.go` |
| `contextview` | 事件日志 → 六分类上下文统计、趋势、前缀稳定、surface 节点 | 见 §3.4 |

---

## 8 缺口清单（按严重度排序）

> 判定纪律：每条只写"代码里能证实的缺失/单一路径/外置依赖"，不写"我觉得应该有"。凡属取舍（非缺陷）另标。

### G1（严重）文档"真编辑"是两套并行实现：模型只能走 Python 技能，Go 原生编辑器只服务 UI

- 证据 A：仓库 `.gaea/skills/` 已是完整 Python 技能包（`git ls-files .gaea/skills | wc -l` = 158；`docx/scripts/create_docx.py` 15KB、`pptx/scripts/create_pptx.py` 6.5KB、`xlsx/scripts/*`、`docx/scripts/office/validate.py` + XSD）。
- 证据 B：模型面**没有**任何 `docx_*`/`xlsx_*`/`pptx_*` 工具：`tool/builtin/*` 的 `Name()` 全表无文档类写工具（§2.1/2.2），`app` 的 ExtraTools 只有 `image_gen/diagram/routine_llm/translate_text/fact_*/dag_plan/ocr/semantic_search/author_memory`（`gaea_handler.go:168-178`）。
- 证据 C：Go 侧已有成熟原生编辑器 `internal/office/{docxedit,pptxedit,xlsxedit}`（修订模式写入 `w:del/w:ins`、ppt run 级替换继承 rPr、xlsx 单元格/行列操作 + 公式重算），但只经 Wails 绑定暴露给前端（`bindings_office.go:79,123,215`；实现 `app/gaea_docx_edit.go:50`、`app/gaea_pptx_edit.go:19`、`app/gaea_xlsx_edit.go:107`）。
- 证据 D：提示词把文档任务全部指向 `run_skill`（`config_paths.go:35`、`single_prompt.go:20,45`），导出也走 python 脚本（`app/gaea_export.go:124,156`，`findSkillScript` 按 `.gaea/skills → <cwd>/skills → ~/.codex/skills → ~/.gaea/skills` 查找）。
- 影响：①同一能力两套实现必然漂移（预览/版式校验/修订在 Go，生成/编辑在 Python）；②模型拿不到"修订模式""单元格/行列操作"这些已实现的结构化编辑，只能生成整文件；③办公硬依赖变成 python 运行时（外加 soffice/poppler 渲染链，见 §9-4）；④**技能是"项目作用域"资产**：技能发现用 `ProjectRoot: cwd`（`boot/sysprompt.go:132`），`GaeaSwitchWorkspace` 会持久化新工作区并重建引擎（`app/gaea_ui_extra.go:1045-1087`），而切换时只安装"任务模板命令"（`ensureTaskTemplateCommands`，`app/gaea_templates.go:96-114`，写 `.gaea/commands`），**不复制技能**——换到没有 `.gaea/skills` 的工作区后，docx/xlsx/pptx/pdf 只能靠用户级目录（`~/.gaea|.claude|.agents|.agent/skills`）兜底，否则办公文档能力消失；`findSkillScript` 的第一候选正是 `gaeaCwd()/.gaea/skills`（`app/gaea_export.go` 同函数）。
- 借鉴含义：任何"文档能力蒸馏"第一步要选边：把原生编辑能力**工具化**（给模型 schema + 权限 + 证据链），还是继续加厚 Python 技能；两者混用会持续制造 G2/G3 类缺口。

### G2（严重）主路径写盘不进证据链、不可回滚、不进产物登记（审计只覆盖 read/write/edit 族）

- 证据 A：`evidence.RecordChange` 的调用点只有 `write_file / edit_file / edit_lines / multi_edit / move_file / schedule_apply`（`tool/builtin/writefile.go:94`、`editfile.go:100`、`editlines.go:121`、`multiedit.go:110`、`movefile.go:130`、`schedule_tools.go:311`）。
- 证据 B：产物登记白名单是"写类 5 种 + 生成/导出类 3 种（format_convert/chart_gen/diagram_gen）"，注释明确 `bash` 不登记（`evidence/evidence.go:380-433`）。
- 证据 C：而办公主路径恰恰是 `bash` + python 技能（G1），以及 `diagram`（App 版，`app/gaea_diagram.go:120` 写 `.mmd`）——都不在白名单里。
- 影响：docx/pptx/xlsx 由脚本生成时，没有 `.before` 基线、Journal 无卡、Verifier 无法复核、Rollback 无法回滚、交付物台账看不到；"证据链/回滚"只对纯文本编辑成立。

### G3（严重）模型没有"版式/渲染验证"工具：提示词要求"执行后必验证"，但文档类验证只在 UI 面

- 证据 A：`single_prompt.go:21`、`config_paths.go:28` 要求每步先验证再 `complete_step`，"不接受纯 manual 声明"。
- 证据 B：模型可用的验证工具只有 `read_file`（文本行）/`format_convert`（转 Markdown，有损）/`bash`（可跑外部脚本，但无产品化断言）；`diagram_gen/chart_gen` 只产图不校验。
- 证据 C：真正的版式验证在后端绑定里：`GaeaPreview`（docx-preview/xlsx 渲染/pptx→PDF 逐页，`app/gaea_preview.go:132-293`）、`GaeaDocumentLint`（`app/gaea_lint.go:17`）、`GaeaVerifyRecord` 的双通道（`app/gaea_verify.go:75-235`）——全部无 agent 工具入口。
- 影响：`complete_step` 的"证据"对文档类只能退化为"文件存在 + 摘要串匹配"（`gaea_verify.go:173-188` 的 default 分支就是文本 contains），版式回归无人自证。

### G4（高）"统一交付出口"（一稿多用）不可被模型调用

- 证据：`GaeaExportDeliverable` 支持 md→docx/pptx/xlsx/pdf + 封面/目录/页眉/页脚/模板（`app/gaea_export.go:22-120`），但只有 Wails 绑定（`bindings_office.go:82`），全仓库无工具包装（工具名枚举无 export 类）。
- 影响：提示词强调的"所有交付物基于同一事实底座生成"（`single_prompt.go:25-27`）在模型侧只能靠"分别跑技能脚本"实现；一致性靠模型自觉，而不是同一出口的格式保证。

### G5（高）办公长任务没有任务化/后台通道；能持久化的只有 4 类非办公任务

- 证据 A：`tasks` 调度器注册的 kind 只有 `price_fetch / price_fetch_all / file_index / outline_reconstruct`（`tasks/tasks.go:28-33`、`app/gaea_tasks.go:50-53`），**没有**"批量文档生成/交付流水线"类任务。
- 证据 B：`jobs.Manager` 是纯内存（`jobs/jobs.go:126-158`，`Close` 即 `cancel`，`:463`），进程退出即失。
- 证据 C：DAG 流水线（唯一的多文件编排）必须由用户在任务中心起跑，节点=子代理（`app/gaea_dag.go:3-8`），没有"夜间批处理/定时"接入。
- 影响：长文档批产只能挂在一个回合里，关窗/取消即断；与"办公"最相关的持久化调度能力缺失。

### G6（高）审批默认逐工具弹卡且全局串行——并行子代理/批量写盘会被单张卡阻塞

- 证据 A：默认 `mode="ask"`、allow 只有 `run_skill`（`config/config.go:810`）→ 每个 `write_file/edit_file/bash/task/format_convert` 都要一张卡。
- 证据 B：`promptMu` 保证同一时刻只有一个待答弹窗，工具审批与 `ask` 共用（字段 `control/controller.go:99`；审批加锁 `controller_approval.go:245-246`；ask 加锁 `controller.go:739-741`）；而工具批/子代理是真并行的（`batch_executor.go:100`、`subagent_limit.go`）。
- 证据 C：没有"本轮全部批准/按计划预授权/批量批准"的通道（决策枚举只到 `persist_allow` 单规则，`controller.go:152-161`）。
- 影响：多子代理并行写盘时用户体验是"排队点卡"，这也解释了为什么产品侧要提供 `permLevel=auto/yolo`（但那等于放弃普通写工具的逐条确认；`hardAsk` 硬门在 yolo 下仍保留，`controller.go:785-808`）。

### G7（中高）交付物台账依赖事件日志 + 仅按会话：legacy 会话与跨会话检索都不可用

- 证据：`GaeaDeliverableRegistry` 只读 `<session>.gaea-log.jsonl`，读不到或无日志即 `Available=false`（`app/gaea_deliverable_registry.go:25-39`）；折叠上限 200 条（`trajectory/deliverable.go:18`）。
- 影响：没有"工作区交付物库"（跨会话/跨项目查产物、查版本、查归属），与 G2 叠加后办公室最需要的"我到底产出了哪些文件"只能按会话看。

### G8（中）事实底座存在并发丢更新风险，且不进压缩摘要/子代理上下文

- 证据 A：每次调用都 `factbase.NewStore(path)` 新建实例（`app/gaea_factbase.go:34-40`），而 `Store.Add` 是"读-改-写"且锁是**实例级**（`factbase/factbase.go:126-138,98-106`）→ 并行子代理/并行工具同时 `fact_add` 会互相覆盖（无进程级锁/文件锁）。
- 证据 B：压缩摘要指令只注入 `progress.md` 进度（`compact.go:200-204`），没有把事实底座折进摘要；摘要模板也无"事实/口径"段（`compact.go:73-98` 的 7 段里只有 Standing facts 泛指）。
- 证据 C：`fact_*` 工具未标 `PersistWrite`（`app/gaea_factbase.go:76-78` 只实现 Name/Description/Schema/ReadOnly/Execute），子代理注册表不会剔除它 → 子代理会写**父会话**的 facts 文件（`gaeaFactBaseStore` 取 `gaeaCtrl().SessionPath()`，`app/gaea_factbase.go:34-39`）。
- 影响：一稿多用的"唯一真相"在多智能体场景下不可靠；压缩后模型只能靠自己记得去 `fact_list`。

### G9（中）DAG 流水线只有"节点=子代理 + 人工验收"一种执行器，无产物校验/自动回流

- 证据：`dag_plan` 只规划不执行（`app/gaea_dag.go:41`）；节点执行走 `TaskTool.RunNew`（`gaea_dag.go:3-8`）；验收由绑定 `GaeaDagNodeAccept` 人工触发（`bindings_office.go:56-57`）；节点风险分级只有 normal/high 的暂停审批（`gaea_dag.go:55`）。
- 影响：流水线产物没有"断言式验收"（如 xlsx 重算零错、docx 页数与版式阈值），仍要人点。

### G10（中）压缩摘要模板是"工程向"，办公语义段缺失

- 证据：`summaryInstruction` 明写 "engineering office assistant"，7 段标题为 Standing facts / Goal / Decisions / Files & code / Commands & outcomes / Errors & fixes / Pending（`compact.go:73-98`）；`Files & code` 用"signatures/line locations/data shapes"的工程口径描述。
- 影响：长办公会话（几十页文档、多个 sheet、多个交付版本）压缩后，"哪一版 docx、哪个 sheet、哪一页"这类结构信息只以散文留在摘要里；没有"交付物清单/版式约束/事实口径"结构化段。

### G11（中）预览/导出/编辑三类能力只有 Wails 绑定，没有模型面或非桌面面

- 证据：`bindings_office.go` 里 `GaeaPreview/GaeaExportDeliverable/GaeaDocxApplyEdit/GaeaPptxApplyEdit/GaeaXlsxApplyEdit/GaeaVerifyRecord/GaeaDocumentLint` 全是纯委托（`:75-126,211`），无工具注册；`bindings_office.go:1` 为生成物（`scripts/gen_bindings`）。
- 影响：蒸馏出的能力若只做"后端函数"而不做"工具 schema + 权限语义"，模型永远用不到；反之做工具又会与 UI 的路径/权限校验（`wspath.ResolveRelWithin`，如 `gaea_docx_edit.go:59`）产生两套入口。

### G12（轻微）工具名/白名单漂移：`diagram` 与 `diagram_gen` 双份，产物登记只认后者

- 证据：`diagram_gen` 在白名单（`evidence/evidence.go:402-411`），但 App 注册给模型的是 `diagram`（`app/gaea_diagram.go:53`），且提示词写的是 "diagram：流程图…"（`config_paths.go:46`）；`diagram` 产出的 `.mmd` 不登记为交付物。
- 影响：图表类交付物漏登（低危，但说明"工具名 → 台账"是维护点）。

### 非缺口（取舍，记录以免误判）

- **会话不是 sqlite 而是 JSONL + 事件日志**：为"模型可见必入日志/可回放/可折叠"服务（`session/log.go:4-9`、`sink.go:1-6`），不是缺陷。
- **中断交接/steer/审批为内存态**：注释自认"进程重启丢失=诚实边界"（`agent.go:425-429`）。
- **子代理只有最终答案回传**：设计目标是保护父上下文预算（`task.go:271-272` 描述、`:892-904` 注释），且已补 `read_subagent_result` 反向通道。

---

## 9 技术债与约束（会限制"从 ZCode 蒸馏"的可借鉴面）

1. **Go 单进程 + Wails 绑定生成**：`bindings_office.go` 是生成物（`// Code generated by scripts/gen_bindings; DO NOT EDIT.`），新增绑定要重跑生成并过 `scripts/check-bindings-drift.ps1`、`TestBindingsCompleteness`（`test` 侧对账）。任何"把 UI 能力工具化"的蒸馏都要同时改三处：实现、绑定、生成物。
2. **只有一个 engine 装配入口**：`boot.Build` 把 provider/tools/gate/hooks/ctxMgr/jobs/store 全部定型（`boot.go:112-704`）；换模型/改空间/改权限都走"重建 controller"（`gaea_handler.go` 的 Rebuild 路径）。增量能力若想"运行时热插"，只能在 registry/sink 上做（MCP 是既有先例：`controller_mcp.go:61`）。
3. **sqlite 单库 + 迁移链**：`Hephaestus.db`（WAL）已到 SchemaV21，`db/schema.go` 39KB 是长链迁移；tasks/memory/knowledge/cost 共享一库。新增"办公任务类型"需要加表/加列并写迁移，不是纯 Go 类型改动。
4. **Windows 优先 + 外部二进制依赖**：soffice/poppler 缺失时视觉复核降级 warn（`gaea_verify.go:78-88`）；bash 缺失时回落 Windows PowerShell 并告警（`boot.go:233-237`）；python 是技能脚本的硬依赖。任何"文档渲染验证"方案都要能吃这三种降级态。
5. **双格式持久化（legacy/event）长期并存**：`Save` 双写、`ResumeFromDisk` 先迁移再恢复（`save.go:46-70`、`controller_session.go:101-122`）；派生层（contextview/trajectory/deliverable）只认事件日志，legacy 会话一律空态 → 新派生能力的"可用性"受格式开关影响。
6. **缓存前缀纪律是硬约束**：`runDirect` 每轮把 activeSchemas 重置回基线（`agent_run.go:72-77`），压缩/rewind 会 `RewriteVersion++` 主动失效缓存（`compact.go:396-397`）；任何"每轮注入动态上下文"的蒸馏都必须走 user 消息折叠（`input.go`）而不是改 system。
7. **回调/接口 seam 很多，改动面宽**：`Asker`/`PermissionRequester`/`Gate`/`ToolHooks`/`Runner` 五个接口（`agent.go:30-186`）都由 control/boot 装配；新增一类"人机往返"（如计划审批、批量审批）必须同时落三处：agent 接口、controller 实现、前端事件+绑定。
8. **单会话单回合 + 内存态在飞状态**：`running` 单飞（`controller.go:319`）意味着"并行多会话/并行多任务"在进程模型上是新增物，不是配置项；后台作业内存态（G5）也意味着"夜跑/长任务"要先补持久化。
9. **双空间过滤已在装配期固化**：`work/play` 按 `spacetags.go` 物理过滤（`:27-97,120`），未入表工具缺省 shared 且有"全表覆盖"测试守卫（`spacetags_test.go:40`）；新增办公工具必须显式入表，否则静默 shared。
10. **可维护性**：`agent` 包 175 文件/2.76 万行、`tool` 88 文件/1.5 万行且无 `registry.go`（注册表在 `tool/tool.go:192`）、`boot.go` 959 行单函数 600+ 行；改动集中在"装配"与"执行环"两处大文件，评审成本高。

---

## 附：本报告用到的"一次读完就能定位"的关键锚点

| 主题 | 锚点 |
|---|---|
| 办公装配 | `internal/app/gaea_handler.go:108`、`internal/gaea/boot/boot.go:112` |
| 办公提示词 | `internal/gaea/config/config_paths.go:14`、`internal/gaea/agent/single_prompt.go:15`、`internal/gaea/boot/sysprompt.go:43` |
| 主循环 | `internal/gaea/agent/agent_run.go:23`、`agent/agent_stream.go:29`、`agent/batch_executor.go:17`、`agent/execute_one.go:21` |
| 压缩 | `internal/gaea/agent/compact.go:17-42,106,155,287,476` |
| 权限/审批 | `internal/gaea/permission/permission.go:114,257`、`internal/gaea/control/controller_approval.go:21,146,233` |
| 会话持久化 | `internal/gaea/agent/session/log.go:181,226`、`session/save.go:26,101`、`control/controller_session.go:97,263` |
| 任务 | `internal/gaea/tasks/tasks.go:28,378,798`、`internal/app/gaea_tasks.go:40` |
| 子代理 | `internal/gaea/agent/task.go:268,783`、`subagent_store.go:61,146,465`、`subagent_limit.go:29` |
| 证据链 | `internal/gaea/evidence/journal.go:35,205`、`internal/app/gaea_verify.go:75,124,238` |
| 产物/轨迹/上下文视图 | `internal/gaea/trajectory/deliverable.go:46`、`internal/gaea/contextview/types.go:10,37`、`internal/gaea/contextview/fold.go:32` |
| 事实底座 | `internal/gaea/factbase/factbase.go:111,126`、`internal/app/gaea_factbase.go:34,78` |
| 办公 UI 专用编辑器（模型不可见） | `internal/app/gaea_docx_edit.go:50`、`gaea_pptx_edit.go:19`、`gaea_xlsx_edit.go:107`、`gaea_export.go:45`、`gaea_preview.go:132` |
