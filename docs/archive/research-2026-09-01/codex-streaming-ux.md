# 对话窗口流式输出重造调研：对齐 OpenAI Codex（2026-09-01）

问题背景：gaea 用户反馈「发送后对话窗口长时间无反应，但轨迹面板显示在干活」。本调研只做桌面资料核对（官方文档 learn.chatgpt.com/code.claude.com/cursor.com/code.visualstudio.com/docs.devin.ai + simonwillison.net），逐句标来源，未核实项明标。

## 1. Codex agent 执行期的对话流（2026 形态）

- Codex IDE 扩展：执行期间消息区不沉寂，完成任务后头部显示耗时行 "Worked for 6m 53s"（learn.chatgpt.com）。
- 编辑以「动词+目标+diffstat+操作」芯片呈现："Edited retry.ts+2−2" 且带 "Undo"；结果区用 "Validation passed:" 平铺要点、"Ask for follow-up changes" 引导续作；还可 "Continue in: Work locally / Cloud" 把会话在本地与云端间迁移（learn.chatgpt.com）。
- Codex CLI：提示行常驻上下文水位 "100% context left · ? for shortcuts"；/status 描述为 "Show the chat ID, context usage, and rate limits"（learn.chatgpt.com）。
- CLI 2026-08 更新："Added an interactive codex agents dashboard for searching, starting, opening, renaming, and stopping tasks"（0.149.0）、"Report completed sub-agent activity on parent turns"（0.150.0）——子代理活动回投到主回合（learn.chatgpt.com）。
- 待核实：网传 CLI 运行头部 "Working (Xs • esc to interrupt)" 与按动作折叠分组 "Ran N commands"，本次官方文档未检索到原文，标为未核实；也未发现「步数计数」UI（learn.chatgpt.com）。
- 最终答复何时开始流：官方未给逐字说明；多代理场景明确规定 "Codex waits until all requested results are available, then returns a consolidated response"（learn.chatgpt.com）。

## 2. 同类对照：Claude Code / Cursor / Copilot / Devin

- Claude Code VS Code：Focus view "hides tool calls, tool results, and thinking behind expandable rows, leaving your prompts and Claude's responses"，且 "Claude's latest to-do list stays visible"——工具细节折叠成可展开行、待办常驻，避免静默（code.claude.com）。
- Claude Code 终端：重复 MCP 调用折叠为 "Called slack 3 times" 一行，Ctrl+O 展开全量带时间戳；Esc 可中断且 "Claude keeps the work done so far"；排队消息列在输入框上方（code.claude.com）。
- Claude Code 面板底部常驻 context 用量指示，后台标签页用颜色区分 "blue means a permission request is pending, orange means Claude finished while the tab was hidden"（code.claude.com）。
- Cursor：工具间隙仍持续动作（"the agent continues reading files, making edits, or running commands"）；消息排队 Enter 入队、Cmd+Enter 立即发；追问 "delivered at the agent's next tool call instead of cutting off work mid-action"；checkpoint 自动建立并可在 chat timeline 点击回看文件快照（cursor.com）。Spinner 具体文案官方文档未载，未核实（cursor.com）。
- Copilot（VS Code）：终端命令 "you can see the output in real time"，可 "Select a changed file in the response to inspect its diff"，并可用 `chat.checkpoints.showFileChanges` 开启每请求文件变更摘要（code.visualstudio.com）；agent 模式强调自愈 "analyzes run-time errors with self-healing capabilities"（github.blog）。spinner 文案未在文档中检索到，未核实。
- Devin：会话内 "progress steps" 可点击回看，工具面板强调可观看："watch commands being executed"、"Watch Devin browse through documentation, test web applications it builds"（docs.devin.ai）。

## 3. 长前置阶段（首个 token 前）的反馈

- OpenAI 官方 prompting guide 要求：多步任务在第一次工具调用前先输出一句用户可见的确认+第一步说明（GPT-5.5 guide，经 simonwillison.net 转述）；作者实测 "it does make longer running tasks feel less like the model has crashed"（simonwillison.net）。
- Codex Goal 模式：goal 进度行 "appears above the composer"，可随时 pause/resume/edit/clear，截图中含 "Paused goal …10h 9m" 计时；支持 "Ask for a status recap" 主动要进度（learn.chatgpt.com）。
- Codex 侧聊："Start a temporary side chat without interrupting the main chat"，不打断主任务地询问状态（learn.chatgpt.com）。
- 完成靠通知兜底：Pets/系统通知提示 chat "needs input or is ready for review"（learn.chatgpt.com）。
- 未核实：Codex web 起始状态 "Scoping out the task" 一词未能在现行文档检索到（learn.chatgpt.com）。

## 4. 子代理/并行任务的行内呈现

- Codex subagents：CLI 用 "/agent to inspect and switch between agent threads while they run"；应用侧 "surfaces each subagent thread"；IDE 里 "active subagents appear above the composer"（learn.chatgpt.com）。
- Web 侧栏 Subagents 视图为 "read-only Active and Done lists"，条目带耗时（如 "Model api audit 2m"）与一句话结果，计数 "Done · 3"，状态词 "started working"（learn.chatgpt.com）。
- CLI 可从主线程处理非活动线程的审批，"press o to open that thread"；最终 "The main thread collects the subagent results into its final response"；并发由 `agents.max_concurrent_threads_per_session` 限制（learn.chatgpt.com）。
- Cursor：/multitask 让请求 "run async subagents in parallel instead of queuing your requests"；每个子代理 "runs in its own context window and returns a result to the main conversation"；计划页有 "Build in Parallel" 按钮；Agents Window 侧栏统一管理并可置顶长任务（cursor.com）。
- 未核实：主对话中「已派子代理」单行 + 实时进度的精确行形态，各家文档均只给入口与聚合规则，未见逐字 UI（learn.chatgpt.com/cursor.com）。

## 5. 对 gaea 的启示

1. 反馈必须内联在对话窗口：首个工具前先流式输出一行确认+第一步（对齐 GPT-5.5 指南），执行期逐动作插行（图标+动词+目标），耗时与上下文水位做成常驻头部（learn.chatgpt.com/simonwillison.net）。
2. 轨迹面板保留为「深度层」而非唯一反馈：默认折叠动作行、可展开（Focus view 逆操作 + "Called slack 3 times" 式聚合）（code.claude.com）。
3. 间隙防静默：等待审批/长命令时显示阶段文本与实时输出（Copilot real-time terminal、Devin 可观看面板），排队消息可见（cursor.com/code.claude.com/docs.devin.ai）。
4. 中断与重定向：Esc 级中断保留已完成工作，追问在下一个工具边界投递（code.claude.com/cursor.com）。
5. 子代理回投主回合并附耗时与一句话结果，完成后发系统通知（learn.chatgpt.com）。

## 来源

learn.chatgpt.com（/codex/ide、/codex/cli、/codex/changelog、/codex/reference/slash-commands、/codex/agent-configuration/subagents、/codex/long-running-work）；code.claude.com（/vs-code、/interactive-mode）；cursor.com（/docs/agent/overview、/docs/agent/plan-mode、/help/ai-features/multi-agent）；code.visualstudio.com（/docs/agents/run/chat-view）；github.blog（Copilot agent mode）；docs.devin.ai（get-started/devin-intro）；simonwillison.net（/tags/codex）。
