# 市场对照：子代理「完整对话 / 实时输出」显示（2026-09-03）

> 调研目标：gaea 在「子代理会话 tab 如何像主代理一样直接显示完整对话/实时
> 输出」上的差距。取证方式：官方文档 / 官方 changelog / GitHub issue / 产品
> 评测，检索时间 2026-09-03；无法直接验证的界面细节标注「未证实」，不臆造。

## 结论先行

1. **主流已把「子代理 = 可点开的完整会话线程」做成默认能力**：Codex（app/
   CLI/IDE/web）、Qwen Code、Cursor、Devin 都在运行时暴露子代理线程，点开
   即可看正在进行的思考/工具/结果；不是“跑完只给一句话摘要”。
2. **主线程与子线程职责分离是共识**：主线程只收最终摘要/结果，避免上下文
   污染；完整中间过程放子线程/面板/日志，两级都可回看。
3. **透明化在增强**：Qwen Code 直接把「时长 + token + 可展开思考与工具过程」
   做成前台 pill + 底部常驻面板；Coze/Dify 提供按请求/节点的完整 I/O 与耗时
   Trace；Codex/Claude/Cursor 的 transcript 独立持久化、可恢复继续。
4. **不是所有产品默认全量内联**：Devin 后台子代理默认隐藏原始输出、父代总结
   后按 f 才内联显示；Claude Code 主会话派生的子代理不在 agent view 单独成行，
   社区因此出现 claude-esp 等“把隐藏子代理输出捞出来”的第三方工具——所以
   “默认完整可见”是加分项而非标配。
5. **对 gaea 的直接含义**：UI 形态（顶栏动态 tab + 子代理行点击打开）方向正确；
   真正缺口是**真实引擎的 transcript 持久化未接线 + 前端靠轮询整份拉取**，
   导致真机没有数据可显示。对齐市场的关键刀是后端接线 + 事件驱动实时渲染，
   而不是再造一套“摘要式”视图。

## 产品对照表

| 产品 | 形态 | 子代理显示方式 | 实时性 | 显示内容 | 完成后 | 回看/继续 | 证据 |
|---|---|---|---|---|---|---|---|
| OpenAI Codex（桌面 app / CLI / IDE / ChatGPT Work） | 原生 Coding Agent | app：主线程 activity 里点开子代理线程；CLI：`/agent` 切换线程；IDE：background-agent panel 展开→打开线程；Web Work：Active/Done 列表→点完成项看详情 | 线程打开即看运行中内容（官方文档口径） | 完整对话线程（含其工作与结果）；主线程另收 summary | 摘要回投主聊天，线程可保留 | transcript 线程可继续/停止/关闭；GitHub issue 显示该交互有时缺任务首条/历史线程不浮出（未证实是否全场景） | [Codex docs](https://www.codex-docs.com/en/docs/agent-configuration/subagents)；[issue #34202](https://github.com/openai/codex/issues/34202)；[issue #16358](https://github.com/openai/codex/issues/16358) |
| Claude Code / Agent SDK | 终端 + SDK | 后台会话有独立 agent view（行状态 → peek 最近输出 → attach 进完整对话）；主会话内派生的 subagent 不单独成行，UI 只显示进度行 | agent view 行摘要 ≤15s 刷新；attach 后完整 | 背景会话完整 transcript；subagent 原始终端 UI 隐藏思考/工具细节，需 `--verbose` 或第三方流 | 父线程收最终消息；subagent transcript 独立持久化 | SDK 可 resume 同一会话恢复 subagent 全历史；UI 可 attach/detach | [agent-view docs](https://code.claude.com/docs/en/agent-view)；[SDK subagents zh](https://code.claude.com/docs/zh-CN/agent-sdk/subagents) |
| Cursor | IDE 插件/桌面 | Agents Window 聚合所有 agent；subagent 后台运行并落盘 `~/.cursor/subagents/`；side chat 是另一路“持久完整对话” | 聊天内流式；subagent 原始输出默认留在子上下文 | 子代理独立上下文；父线程只收 final summary | 可 resume 已完成 subagent 延续上下文 | Agents Window 里 Cmd+K 全文搜索 transcript；可 resume | [Cursor docs/subagents](https://cursor.com/docs/subagents.md)（页面直连不稳，结论以检索摘录为准）；[side-chat changelog](https://cursor.com/en-US/changelog/side-chat)；[AgentPatterns 评测](https://agentpatterns.ai/tools/cursor/multitask-subagents/) |
| Qwen Code（通义编码 CLI） | 终端 Agent | 底部常驻 LiveAgentPanel；前台子代理以 pill 出现，展开即看完整推理与工具调用；并行子代理各自独立状态 | 状态/耗时/token 实时刷新；可及时中断 | 推理、工具调用过程、时长、token | 父线程收子代理结果 | pill 展开回看；`/resume` 会话恢复 | [Qwen Code 周报 2026-05-14](https://qwenlm.github.io/qwen-code-docs/zh/blog/updates/weekly-update-2026-05-14/) |
| Devin（CLI/Desktop） | 任务 Agent | 后台子代理：输入区下方状态指示器 → Enter 开子代理面板（profile/标题/状态/耗时/工具调用数）；前台子代理内联在当前会话 | 前台内联实时；后台只给状态 | 官方明示“你无法直接看到后台子代理原始输出”，父代理总结关键发现与操作 | 后台完成后通知父代理；可恢复 | 面板按 f 把后台子代理切前台 → 输出内联显示；面板 reload 后保留 | [Devin zh docs](https://docs.devin.ai/zh/cli/subagents) |
| Cline / Roo Code | VS Code 扩展 | 单面板内嵌：主对话流显示 Cline 子代理进度行/结果；Roo 以同一 chat 面板展示行动 | 主对话流式 | Cline 子代理只读研究，返回 report 给主代理；Roo 聊天历史含请求/响应/动作 | 报告回主线程 | Cline History 可恢复完整任务会话（未证实子代理逐条回看） | [Cline Subagents](https://docs.cline.bot/features/subagents)；[Roo chat interface](https://roocodeinc.github.io/Roo-Code/basic-usage/the-chat-interface/) |
| 腾讯 CodeBuddy | IDE | Craft Agent（主）自动调 agentic Subagents，独立上下文、不污染主会话 | agentic 模式中途不可干预（等待结果或中断整轮） | 官方未强调逐条展开；重点在 agent 配置 | 结果返回主 Agent | manual 模式可替代主 Agent 使用 | [CodeBuddy subagents 指南](https://cloud.tencent.com.cn/document/product/1831/134515) |
| Manus | 网页任务 Agent | 计划→执行两步可见；执行中实时看状态/操作（移动端也可） | 实时 | 任务计划、浏览器/终端操作、产物；官方未给出逐 token transcript 式展示（未证实内部子代理原始流） | 交付物 + 步骤回放 | 计划任务可按执行卡/标签回看历史结果 | [Manus 官方博客](https://manus.im/zh-cn/blog/manus-schedules)；[第三方拆解](https://www.layer3labs.io/guides/manus-ai-explained) |
| Coze / 扣子（编程） | 可视化编排 | Trace 页按请求列节点；点开查看节点级完整输入输出与耗时；消息日志可查请求链路；可导出 Excel | 运行后 Trace（3-180 天留存） | 每节点 I/O、耗时、token | 线上运维式回看 | Trace 看板筛选/导出 | [扣子查看日志和 Trace](https://docs.coze.cn/guides/view_running_log) |
| Dify | 可视化编排 | Agent 节点会话汇总到日志；工作流 Trace 显示每节点 input/output/token/耗时 | 事件流推送 node_started/node_finished | 节点级完整输入输出 | 日志留存 | View Logs → Tracing | [Dify Agent docs](https://docs.dify.ai/zh/cloud/use-dify/build/new-agent/overview)（英文详版 DeepWiki 佐证事件流） |

> GitHub Copilot、Windsurf、Gemini CLI、Aider 等本次未充分取证（未证实项不
> 下结论）；Gemini/Jules 后台任务展示模式与 Codex/Devin 同类，建议下一轮补充
> 实测截图再定稿。

## 共性设计（可直接借鉴）

1. **两级信息模型**：父线程 = 决策与摘要；子线程 = 完整过程。开子代理 = 打开
   一个与主线程同构、可继续的对话（Codex 线程、Claude attach、Cursor
   subagent resume、Devin 前台内联）。
2. **运行中透明**：至少提供“正在做什么”的一行摘要/状态；进阶产品（Qwen、
   Codex、Cursor）把完整思考与工具调用逐步可展开，且运行中即可点开，不必等
   完成。
3. **元信息伴随**：profile/模型、耗时、工具调用数、token 消耗与状态一起展示
   （Devin 面板、Qwen pill、Claude agent view 行状态）。
4. **持久化独立于主会话**：子代理 transcript 自成文件，主会话压缩/结束后仍可
   恢复（Claude SDK、Cursor `~/.cursor/subagents/`、Devin reload 保留）。
5. **控制闭环**：停止、前台/后台切换、恢复继续是标配（Codex steer/stop、
   Claude attach、Devin Ctrl+B/f/x、Cursor resume、Qwen 中断）。

## gaea 差距与建议

- **差距 1（真机数据源）**：`SubagentStore` 的 JSONL/meta 落盘已实现但生产
  boot 未用 `taskTool.WithTranscripts(...)` 接线 → 真实引擎下没有
  `subagents/*.jsonl`，子代理 tab 无内容可显示（mock 模式除外）。
- **差距 2（实时渲染）**：前端是 3s 轮询 + 事件节流整份拉取，非逐段/逐字
  增量；对应市场“运行中即可点开看完整过程”的体验需事件直推。
- **建议落地顺序**：
  1. P0：boot 接线 SubagentStore（每个 subagent step 追加 transcript 增量 +
     meta 状态），让真机开始产生数据；同时补一条子代理文本/tool 增量事件。
  2. P0：SubagentThread 改事件订阅增量渲染（可复用主对话 MemoMarkdown 的
     streaming 路径），运行中打开 tab 即实时跟随。
  3. P1：子代理面板行显示 profile/模型/耗时/token（Devin/Qwen 口径），主对话
     完成卡 = 摘要 + “打开完整线程”入口（Codex/Cursor 口径）。
  4. P1：恢复继续（失败/取消/完成后续问）与停止/切前后台（Claude/Devin 口径）。

## 参考链接

- Codex Subagents（官方文档镜像/开发者文档）：https://www.codex-docs.com/en/docs/agent-configuration/subagents 、https://developers.openai.com/codex/subagents
- Claude Code Agent View：https://code.claude.com/docs/en/agent-view ；SDK 子代理（zh）：https://code.claude.com/docs/zh-CN/agent-sdk/subagents
- Cursor Subagents / Side Chats：https://cursor.com/docs/subagents.md 、https://cursor.com/en-US/changelog/side-chat
- Qwen Code 周报（子 Agent 可视化）：https://qwenlm.github.io/qwen-code-docs/zh/blog/updates/weekly-update-2026-05-14/
- Devin CLI 子 Agent（zh）：https://docs.devin.ai/zh/cli/subagents
- Cline Subagents：https://docs.cline.bot/features/subagents ；Roo Code Chat Interface：https://roocodeinc.github.io/Roo-Code/basic-usage/the-chat-interface/
- CodeBuddy Subagents：https://cloud.tencent.com.cn/document/product/1831/134515
- Coze 日志与 Trace：https://docs.coze.cn/guides/view_running_log ；Dify Agent：https://docs.dify.ai/zh/cloud/use-dify/build/new-agent/overview
- Manus 计划任务/执行回看：https://manus.im/zh-cn/blog/manus-schedules ；第三方拆解：https://www.layer3labs.io/guides/manus-ai-explained
