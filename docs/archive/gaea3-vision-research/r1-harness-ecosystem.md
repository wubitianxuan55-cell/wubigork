# gaea 3.0 愿景规划 · 调研报告 R1：Agent Harness 与开放协议生态

> **背景**：为 gaea 3.0（Wails 桌面 AI 助手：Go 后端 + React 前端，单用户本机，模块含聊天/轻语人格/小说创作/绘梦/办公 agent/记忆中枢/模型中心）向 DeepSeek Harness（DSH）式 agent harness 架构靠拢收集公开信息。
> **调研时间**：2026-08（本报告所有数据截至检索日，星标/ARR/服务器数量等指标随时间变化）
> **方法**：中英文 web 检索（每主题多轮）+ 关键页面精读（官方博客 / 官方仓库 / 行业媒体 / arXiv 论文 / 知乎·博客园技术社区）+ 与本地 DSH checkout 交叉验证。
> **范围**：① DSH 公开信息；② 2025–2026 agent harness / coding agent 格局；③ MCP / Skills / ACP / A2A 协议生态；④ 个人 AI 助手"工作台化"趋势。

---

## 摘要：10 条核心结论（速览）

1. **DeepSeek Harness（dsh）是真实的开源产品**：DeepSeek AI 于 2026-08-13/14 发布 v0.1 开发者预览版（MIT 协议），定位对标 OpenAI Codex 与 Anthropic Claude Cowork，公开公式"Model+Harness=Agent"。来源：[IT之家](https://www.ithome.com/0/989/446.htm)
2. **dsh 最核心的设计是"一切皆插件"**：模型、工具、技能、会话、沙箱、存储、循环、调度、UI 全部由 Cordis 插件组合而成，连 agent 循环、存储、UI 都可替换；主流 agent（Claude Code/Codex）只开放"外挂层"，dsh 开放"运行时层"。来源：[博客园深度对比](https://www.cnblogs.com/qq8864/articles/22479803)、[IT之家](https://www.ithome.com/0/989/446.htm)
3. **dsh 的理论底座是北大+DeepSeek 论文《A Programming Paradigm for Spatiotemporal Composability》**（Cordis 元框架）：时间可组合性=可逆效应（卸载回滚副作用）、空间可组合性=反应式余效应（依赖就绪才加载），为插件可替换/可回滚/热替换（HMR）提供形式化保证。来源：[论文仓](https://github.com/cordiverse/paper/blob/main/paper.pdf)、[智东西](https://www.zhidx.com/p/584897.html)
4. **dsh 的"轨迹"设计（append-only 事件日志）与主流 agent 的会话转录理念一致**：系统提示词、思维链、工具调用与结果、子 Agent 调度、每次上下文注入全部落盘，恢复/分叉/检索/回放共享同一事件流；dsh 用 zstd 压缩的 JSONL 事件日志存储（~/.dsh/sessions/*.jsonl.zstd）。来源：[智东西](https://www.zhidx.com/p/584897.html)、[session-persistence-jsonl](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/session/session-persistence-jsonl/README.md)
5. **编码 agent 已成 2025–2026 最热赛道且商业化爆发**：Claude Code 约 $2.5B ARR（2026-04）、"史上增长最快的开发者产品"；Anthropic 整体 ARR $14B（2026-02）；OpenAI Codex CLI（2025-05 开源）、Google Gemini CLI（2025-06-25 开源）、阿里 Qwen Code（2025-08，~25.9k 星）、OpenCode（2026-03 破 120k 星）悉数入场。来源：[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/25/claude-code-25b-arr-fastest-ai-developer-tool-billion-dollar-revenue)、[saastr](https://www.saastr.com/anthropic-just-hit-14-billion-in-arr-up-from-1-billion-just-14-months-ago/)、[star-history](https://www.star-history.com/qwenlm/qwen-code/)、[TheAgentTimes](https://theagenttimes.com/articles/opencode-crosses-120k-github-stars-as-open-source-coding-age-3e4b556f)
6. **实证研究显示编码 agent 已在 GitHub 大规模普及**：arXiv 2601.18341（2026-01）对 129,134 个项目的统计显示 2025 上半年采用率达 15.85%–22.60% 且仍在上升；2026-06 后续分析称新项目采用率已翻倍。来源：[EmergentMind 论文页](https://www.emergentmind.com/papers/2601.18341)、[codex.danielvaughan.com](https://codex.danielvaughan.com/2026/06/18/agentic-very-much-coding-agent-adoption-doubled-new-github-projects-codex-cli-team-configuration/)
7. **MCP 已成事实标准且规模巨大**：2026-04 时活跃公共服务器超 10,000 个、月 SDK 下载 9,700 万（16 个月达到 React 三年的量级）；官方 MCP Registry 2026-03 列 12,000+ 条目；2025-12 捐赠给 Linux Foundation 旗下 Agentic AI Foundation（AAIF，创始成员 Anthropic/Block/OpenAI，146 家成员）。来源：[MCP 官方一周年博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
8. **协议分层已清晰：MCP（agent↔工具，垂直）＋ A2A（agent↔agent，水平）＋ ACP（agent↔编辑器/客户端）**：A2A 于 2025-06 由 Google 捐赠 Linux Foundation（AWS 加入）；ACP 由 Zed 发起（Gemini CLI 为参考实现，Claude Code/Goose 已支持），2026-01-28 ACP Registry 随 JetBrains 上线。来源：[Zed ACP 进展报告](https://zed.dev/blog/acp-progress-report)、[Zed ACP Registry](https://zed.dev/blog/acp-registry)、[Linux Foundation](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents)
9. **Skills（SKILL.md）成为第二项跨 agent 标准**：Anthropic 2025-10 发布 Agent Skills；格式被 Claude Code/Codex/Cursor/Gemini CLI 等采纳，dsh 社区已有 dsh-skillport 兼容转换工具；有观点认为"一个 SKILL.md 可取代 CLAUDE.md、AGENTS.md、.cursorrules"。来源：[Simon Willison](https://simonwillison.net/2025/Dec/19/agent-skills/)、[dsh-skillport](https://github.com/Jesse-njx/dsh-skillport)、[dev.to](https://dev.to/creeta/one-skillmd-replaces-claudemd-agentsmd-and-cursorrules-5a70)
10. **个人 AI 助手正在"工作台化"**：2026 年腾讯 WorkBuddy、字节 TRAE Work、阿里 QoderWork 等纷纷从聊天框转向"类 Office"工作台（文档/项目/协作/版本）；Claude Cowork、ChatGPT Work、Gemini Computer Use 让助手从"能说"变"会干"，ChatGPT Work 可连续工作数小时。来源：[腾讯云开发者社区](https://cloud.tencent.com.cn/developer/article/2708771)、[IT之家 Cowork](https://www.ithome.com/0/912/701.htm)、[澎湃 ChatGPT Work](https://m.thepaper.cn/newsDetail_forward_33586633)

---

## 〇、2024-11 → 2026-08 关键时间线（速查）

- **2024-11-25**：Anthropic 开源 MCP（Model Context Protocol），从"给 Claude 接本地工具"的实验开始。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)
- **2025-02**：Claude Code 公开面世；OpenAI Codex 云版上线（"AI 程序员上线，人类仅需点按钮"）。[澎湃](https://m.thepaper.cn/newsdetail_forward_30833637)
- **2025-04**：Google 发布 A2A（Agent-to-Agent Protocol）；Zed 发布 ACP（Agent Client Protocol，与 Google 合作把 Gemini CLI 带入 Zed）。[腾讯云解读](https://cloud.tencent.cn/developer/article/2664955)、[Zed](https://zed.dev/blog/acp-progress-report)
- **2025-05**：OpenAI Codex CLI 开源（TS 启动器 + Rust 二进制）。[百度百科](https://baike.baidu.com/item/Codex%20CLI/65594415)、[MIT AI Agent Index](https://aiagentindex.mit.edu/2025/codex/)
- **2025-06-25**：Google 开源 Gemini CLI；同月 Google 将 A2A 捐赠 Linux Foundation（AWS 加入支持）。[china.org.cn](http://www.china.org.cn/world/Off_the_Wire/2025-06/26/content_117947933.shtml)、[Google Developers Blog](https://developers.googleblog.com/en/google-cloud-donates-a2a-to-linux-foundation/)、[Linux Foundation](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents)
- **2025-08**：阿里通义推出 Qwen Code（CLI + IDE 双形态，Qwen3-Coder 系）。[百度百科](https://baike.baidu.com/item/Qwen%20Code/66293865)、[Qwen 官方博客](https://qwenlm.github.io/blog/qwen3-coder/)
- **2025-09**：官方 MCP Registry 上线（初始批次服务器，一周年时 407% 增长、近 2,000 条目）。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)
- **2025-10**：Anthropic 发布 Claude Agent SDK（Python/TS，开源）与 Agent Skills（SKILL.md）；Claude Sonnet 4.5 同期发布。来源：[Open Source For You](https://www.opensourceforu.com/2025/10/anthropic-releases-open-source-claude-agent-sdk-alongside-claude-sonnet-4-5-breakthrough/)、[Simon Willison](https://simonwillison.net/2025/Dec/19/agent-skills/)
- **2025-11-25**：MCP 一周年 + 新规范发布（授权扩展、SEP 治理、Working/Interest Groups）。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)
- **2025-11/12**：Claude Code 推出 Agent Extensions / 插件能力；Anthropic 发布 Claude Cowork（"面向所有人"的 Claude Code），新增 RBAC 与 MCP 工具级权限控制。[Zed 博客提及](https://zed.dev/blog/acp-registry)、[IT之家](https://www.ithome.com/0/912/701.htm)、[digitaltoday](https://www.digitaltoday.co.kr/cn/view/46886)
- **2025-12**：Anthropic 将 MCP 捐赠给 Linux Foundation 新成立的 Agentic AI Foundation（AAIF），创始成员 Anthropic/Block/OpenAI。[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **2026-01-28**：ACP Registry 上线（Zed + JetBrains），可在 JetBrains IDE 中查找连接 ACP 编码 agent。[Zed](https://zed.dev/blog/acp-registry)、[JetBrains](https://blog.jetbrains.com/ai/2026/01/acp-agent-registry/)
- **2026-01~06**：ChatGPT Work / ChatGPT agent mode / Codex 支持 Windows 电脑控制等 computer use 能力集中落地。[澎湃](https://m.thepaper.cn/newsDetail_forward_33586633)、[腾讯云](https://cloud.tencent.cn/developer/article/2695965)
- **2026-03**：OpenCode 破 120k 星（Hacker News 登顶）；官方 MCP Registry 达 12,000+ 条目。[TheAgentTimes](https://theagenttimes.com/articles/opencode-crosses-120k-github-stars-as-open-source-coding-age-3e4b556f)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **2026-04**：MCP 生态破万（10,000+ 活跃服务器、月 9,700 万 SDK 下载）。[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **2026-06**：腾讯 WorkBuddy / 字节 TRAE Work / 阿里 QoderWork 集体转向"类 Office"工作台。[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)
- **2026-08-13/14**：DeepSeek-V4-Pro-0813 开源 + **DeepSeek Harness v0.1 公测（MIT）**，同步开放 npm 插件生态。[IT之家](https://www.ithome.com/0/989/446.htm)、[智东西](https://www.zhidx.com/p/584897.html)

---

## 一、DeepSeek Harness 公开信息

### 1.1 是什么：产品定位与发布事实

- **产品定义**：DeepSeek Harness（dsh）是 DeepSeek AI 开发的开源 agent harness。"Harness"的通俗定义是把模型转化为智能体的工具：在 LLM 之外，它负责调度上下文、工具、任务状态、反馈与边界，把模型能力落地到真实环境，完成从理解需求到交付成果的完整闭环。[IT之家](https://www.ithome.com/0/989/446.htm)
- **发布事件**：2026-08-13/14，DeepSeek 在发布 DeepSeek-V4-Pro-0813 开源模型的同时推出 Harness v0.1 开发者预览版（MIT 协议），并同步开放配套 npm 插件生态；发布前 2 小时官方先宣布 V4-Pro 正式版与 API 大幅涨价，"闪电二连击"引爆社交媒体。[IT之家](https://www.ithome.com/0/989/446.htm)、[智东西](https://www.zhidx.com/p/584897.html)
- **V4-Pro-0813 背景数据**：MoE 架构，总参数 1.6 万亿、推理时每 token 激活 490 亿；100 万 token 上下文、最大 38.4 万 token 输出；支持思考/非思考模式、JSON 结构化输出、Tool Calls、Responses API 与 Anthropic API；权重 MIT 开源，被视为"将模型底座从对特定算力的依赖中解耦"。[IT之家](https://www.ithome.com/0/989/446.htm)
- **团队**：负责人崔添翼，2026-08-02 面向全球公开征集内测用户，短时间吸引数十名全球开发者报名；早期已有"DeepSeek Harness 团队"公众号注册信号。[IT之家](https://www.ithome.com/0/989/446.htm)
- **产品定位**：官方"精准对标 OpenAI Codex 及 Anthropic Claude Cowork，旨在成为一款主打编程和办公场景的 AI 生产力工具"；招聘信息中给出公式 **"Model+Harness=Agent"**。[IT之家](https://www.ithome.com/0/989/446.htm)
- **如何运行**：`npx @deepseek-ai/dsh web` 一键启动 Web UI（默认 http://127.0.0.1:3080）；源码方式 `git clone github.com/deepseek-ai/deepseek-harness && pnpm install && pnpm run build && pnpm dsh web`；核心命令 `dsh --profile <name>` 启动指定配置，支持 Web/Headless 等模式并可插件扩展。[官方 README](https://github.com/deepseek-ai/deepseek-harness/blob/master/README.md)、[IT之家](https://www.ithome.com/0/989/446.htm)
- **许可与实现**：MIT；仓库主体 TypeScript（97.1%），12,293 次提交（截至 2026-08-14 附近 master）；发布状态 v0.1.0-rc.x，官方明确"兼容性破坏性变更会频繁发生"。[官方 README](https://github.com/deepseek-ai/deepseek-harness/blob/master/README.md)

### 1.2 社区认知与热度

- **星标增长曲线**：发布半小时 GitHub 星数破 1 万、智东西发稿时超 3 万（2026-08-14）；另有口径：约 1.5 小时 2.2 万星、"GitHub 史上增长最快开源项目"；首日 28k 星；dtinsight 报道称"一夜 5 万星，Agent 界的 Android"；博客园插件开发者（2026-08-14 晚）称关注量 78.8k★。[智东西](https://www.zhidx.com/p/584897.html)、[sohu](https://www.sohu.com/a/1062527577_115128)、[locdd](https://locdd.com/t/topic/80277/8)、[dtinsight](https://www.dtinsight.com.cn/sys-nd/4072.html)、[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **正面评价**："Agent 界的 Android / 乐高玩具"，可以随心意组装成任何样子；开源让用户掌握未来（"闭源为用户提供现在，开源让用户自己掌握未来"）；"Agent 终于不是黑盒了"。[智东西](https://www.zhidx.com/p/584897.html)、[dtinsight](https://www.dtinsight.com.cn/sys-nd/4072.html)、[博客园](https://www.cnblogs.com/AI-life/articles/22466870)
- **争议/保留意见**：部分开发者认为"它更像一个开发框架，而非 Coding Agent"；功能与生态当时尚不成熟；TS 运行时性能不如 Rust 单体（Codex/AtomCode）；发布 24 小时内已有人写"一整本书"（dsh-handbook）也说明迭代极快。[智东西](https://www.zhidx.com/p/584897.html)、[博客园](https://www.cnblogs.com/qq8864/articles/22479803)、[博客园 2](https://www.cnblogs.com/itech/p/22467523)
- **重要澄清**：dsh **没有**采用社区先前猜测的 Pi Agent 架构，也未套用 DeepSeek 缓存命中率极高的 Reasonix；差异化完全来自 Cordis 插件系统。[智东西](https://www.zhidx.com/p/584897.html)
- **生态信号**：GitHub 设 `dsh-plugin` topic 便于插件发现；社区出现 dsh-handbook（0 到 1 深度手册，中英 PDF）、learn-dsh（拆解教学课程 + 简易教学版实现）、dsh-TUI、dsh-session-export、dsh-notify、dsh-session-report、dsh-article-publish、dsh-skillport（把 Claude Code/Codex/Cursor/Gemini CLI 的 SKILL.md 迁入 dsh）等。[dsh-handbook](https://github.com/Electricitysheep/dsh-handbook)、[learn-dsh](https://github.com/onychen/learn-dsh)、[dsh-skillport](https://github.com/Jesse-njx/dsh-skillport)、[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **国际认知**：VentureBeat 称其为 "open source rival to Claude Code"，并点出与 V4-Pro API 涨价同期发布；Gigazine（日）亦报道 v0.1 发布。[VentureBeat](https://venturebeat.com/technology/deepseek-harness-launches-as-open-source-rival-to-claude-code-alongside-v4-pro-on-api-with-higher-prices)、[Gigazine](https://gigazine.net/news/20260814-deepseek-harness-v0-1/)
- **行业定位参考**：AI 原生全景图（landscape.jimmysong.io）已收录 DSH 条目。[AI 原生全景图](https://landscape.jimmysong.io/zh/projects/deepseek-harness/)

### 1.3 架构：为什么"一切皆插件"是运行时级的

- **核心原则**：模型、工具、技能、会话、沙箱、存储、循环、调度、UI……所有 Agent 能力均由插件组合而成，**没有"核心"**；base bundle 也只是第一个插件组合；开发者无需改动源码即可独立选择、替换或扩展任一能力。[IT之家](https://www.ithome.com/0/989/446.htm)、[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **底层框架 Cordis**：元框架只负责插件的加载、卸载与依赖关系；所有具体组件都是 Cordis 插件，插件通过 Cordis 服务与事件彼此协作，可在配置层自由组合。理念来自北京大学与 DeepSeek 联合署名论文《A Programming Paradigm for Spatiotemporal Composability》（paper.pdf 在 github.com/cordiverse/paper）。[智东西](https://www.zhidx.com/p/584897.html)、[IT之家](https://www.ithome.com/0/989/446.htm)、[53ai](https://www.53ai.com/news/OpenSourceLLM/2026081415798.html)
- **时间可组合性 = 可逆效应（revertible effects）**：组件卸载时运行时完整回滚其全部副作用，每个上下文变换携带逆变换；插件通过 `ctx.effect(() => cleanup)` 注册清理逻辑；HMR 热替换（改配置/代码不重启换插件）的安全保证来自此机制。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **空间可组合性 = 反应式余效应（reactive coeffects）**：上下文一旦变化，主动通知符合余效应规格的组件；`inject: ['sessions']` 声明依赖（就绪才加载、消失自动卸载）；session/event 事件流是导出、通知、报表等插件的协作基础。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **形式化意义**：论文把两维度统一为"上下文类型"，给出动态组合演算元理论——插件的可替换、可回滚、可重组是有证明的，不是工程巧合。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)、[论文仓](https://github.com/cordiverse/paper/blob/main/paper.pdf)
- **内置四种模式（每种默认加载不同插件集）**：
  - 标准模式：提供完整工具组合；
  - PTC 模式（Programmatic Tool Calling）：由模型生成一段代码来组合多轮工具调用（实测写贪吃蛇 1 分 05 秒）；
  - 极简模式：仅保留一个 shell 工具与一个文件编辑工具，用于最小环境下的模型基准测试（实测贪吃蛇 50 余秒，不做多余动作）；
  - 创造模式：检查当前运行时、在内存中试验 Cordis 插件，据此组合并创作新的模式。[IT之家](https://www.ithome.com/0/989/446.htm)、[智东西](https://www.zhidx.com/p/584897.html)
- **"轨迹"（trajectory）：append-only 会话日志**：模型看到的一切——系统提示词、思维链、工具调用与结果、子 Agent 调度、每一次上下文注入——都写入只追加的会话日志；轨迹视图按来源展示；**恢复、分叉、检索与回放共享同一事件流**。[智东西](https://www.zhidx.com/p/584897.html)
- **实测表现（智东西）**：88 页论文翻译耗时 22 分钟、首 Token 平均 1.4 秒、缓存命中率 98%、输入 6.6M token / 输出 72.7k token；任务中自动派发 10 个子代理；PDF 工具首次使用需现装（"一穷二白"按需装插件）。[智东西](https://www.zhidx.com/p/584897.html)
- **存储实现**：官方 dsh-session-persistence-jsonl（zstd 压缩事件日志，本地位于 ~/.dsh/sessions/*.jsonl.zstd），另有 sqlite 变体；会话子系统为事件溯源式（session/event 订阅、只读事件日志）。[session-persistence-jsonl](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/session/session-persistence-jsonl/README.md)、[session 文档](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/session.zh.md)、[core 文档](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/core.zh.md)
- **插件机制一览（社区开发者实测）**：
  - 模型适配：`ctx.llm.registerAdapter(names, adapter)`，支持任意 OpenAI 兼容端点；
  - 工具：`ctx.tools.register(defineTool({name, parameters, execute}))`；
  - 技能（Skills）：官方内置 dsh-code-review、dsh-doc-standards 等，会话内自动注入 skill 目录；
  - 会话：事件溯源服务 ctx.sessions（session/event 订阅、session.events 只读日志）；
  - 沙箱/权限：dsh-sandbox-local 等插件，workspace 围栏；
  - 存储：dsh-session-persistence-jsonl（zstd 事件日志）；
  - agent 循环：dsh-agent-loop（turn/step 驱动、并行调度；工具执行流水线：schema 校验→冻结→调度器 prepare）；
  - 调度/任务：Fiber 生命周期状态机 + ctx.jobs.start()（长任务），headless 退出前 dispose flush 兜底；
  - UI：dsh-web-app（Web）、社区 dsh-TUI（终端）——web/headless/tui 三表面同一内核。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **自进化能力**：在 Cordis 协议下，模型可以在不打断任务的前提下写插件、装插件、动态加载，从海量实例中筛出好插件融回主线——"自进化是架构特性而非功能"（对比 openJiuwen 的 Skill 自进化在 Skill 定义层，dsh 在运行时层）。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)、[CSDN](https://csdnnews.blog.csdn.net/article/details/163747298)
- **本机交叉验证**：本机 checkout（C:/AI/deepseek-harness，deepseek-ai/deepseek-harness 的 fork）packages 结构证实公开描述：session / subagent（含 subagent-acp、subagent-claude-code、subagent-codex）/ shell（bash-sandbox、pwsh-sandbox）/ skill / sandbox / storage / web / workflow / workspace / spill / schedule / typert / boot（app-boot）/ apps/desktop 等包；vendor 内置 cordis/cosmokit/hmr/schemastery；依赖树含 @agentclientprotocol/sdk、@modelcontextprotocol/server-filesystem、@openai/codex、e2b、playwright——**DSH 已内建 ACP 协议支持、可把 Claude Code / Codex 作为子 agent 驱动、自带 MCP 服务器与云沙箱桥接**。（本地仓库观察，与公开仓库结构一致）

### 1.4 与主流 agent 框架的差异（公开对比）

| 维度 | Claude Code | Codex CLI | AtomCode | openJiuwen 生态 | DeepSeek Harness (dsh) |
|---|---|---|---|---|---|
| 开源 | 闭源产品 | 开源（TS 启动器 + Rust 二进制，2025） | MIT | Apache-2.0 | MIT |
| 实现 | 闭源 | Rust 核心 + TS 薄壳 | Rust 分层（kernel/capabilities/coding/tui/daemon） | Python 为主多语言（agent-protocol 为 C++ SDK，agent-core 有 Java 版） | TypeScript + Cordis |
| 模型绑定 | Anthropic | OpenAI | 任意 OpenAI 兼容 | 华为 MaaS / OpenAI 兼容 / 本地 | 任意 OpenAI 兼容 + 自家 |
| 扩展单元 | MCP / skills / hooks / subagents | MCP / hooks / skills | skills / MCP / memory | Tool 生态 + Skill Hub + Agent SDK + 可视化工作流 | 插件（运行时级） |
| 可组合粒度 | 外挂层（只能加，不能改） | 外挂层（核心不可替换） | 能力模块化，运行时固定 | 能力/智能体/工作流层，运行时是框架 | 运行时层（循环/存储/沙箱/UI 全可换） |
| 特色 | 生态最大、打磨深 | 模型强、云/本地双形态 | 纯 Rust 性能、100% AI 生成 | 分布式 swarm、Skill 自进化、DeepSearch、可视化编排 | 可逆效应理论、全插件化、自进化潜力 |

- 核心区别一句话：Claude Code/Codex 开放"能力"，AtomCode 开放"分层源码"，openJiuwen 开放"平台与编排"，**dsh 开放"运行时本身"**（"安卓 vs iPhone"的比喻）。[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)
- openJiuwen（华为系）生态共 17 个仓库：agent-core（Agent SDK）、agent-runtime（分布式运行部署）、agent-dx（分布式执行）、agent-tools（Tool 生态）、agent-memory（长期记忆）、agent-protocol（**MCP/A2A 协议 C++ SDK**）、agent-studio（零码/低码可视化工作流编排）、skillhub（Skill 托管分发，兼容 ClawHub）、jiuwenswarm（多智能体协作）、deepsearch（深度搜索）、relay（多智能体协作平台）等。[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)
- 诚实的另一面：dsh 是 developer preview（破坏性变更频繁）、TS 运行时性能不如 Rust 单体（Codex/AtomCode）、生态成熟度远不如 Claude Code；但其"运行时全开放 + 可逆效应理论"是独有卖点，78.8k 星关注者"赌的是架构方向"。[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)

---

## 二、2025–2026 主流 agent harness / coding agent 格局

### 2.1 玩家全景与关键数据

| 产品 | 发布/开源时间 | 形态 | 关键数据/事实 | 来源 |
|---|---|---|---|---|
| Claude Code | 2025-02 公开 | 终端 CLI + IDE 插件 + Agent SDK | 约 $2.5B ARR（2026-04）；"史上增长最快的开发者产品"；Anthropic 整体 $14B ARR（2026-02，Bloomberg） | [AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/25/claude-code-25b-arr-fastest-ai-developer-tool-billion-dollar-revenue)、[saastr](https://www.saastr.com/anthropic-just-hit-14-billion-in-arr-up-from-1-billion-just-14-months-ago/)、[Bloomberg](https://www.bloomberg.com/news/articles/2026-02-20/the-surprise-hit-that-made-anthropic-into-an-ai-juggernaut-mlve4nc2)、[Product Podcast](https://productschool.com/resources/product-podcast/meaghan-choi-anthropic-claude-code-fastest-growing-dev-tool) |
| OpenAI Codex CLI | 云版 2025-02，CLI 开源 2025-05 | TS 启动器 + Rust 二进制 | 云/本地双形态；已支持 Windows 电脑控制、手机远程派活 | [百度百科](https://baike.baidu.com/item/Codex%20CLI/65594415)、[MIT AI Agent Index](https://aiagentindex.mit.edu/2025/codex/)、[腾讯云](https://cloud.tencent.cn/developer/article/2695965) |
| Gemini CLI | 2025-06-25 开源 | 终端 agent（支持 subagents、AGENTS.md） | 与 Zed 联合成为 ACP 参考实现；Gemini 3.5 Flash 内置 computer use | [china.org.cn](http://www.china.org.cn/world/Off_the_Wire/2025-06/26/content_117947933.shtml)、[InfoWorld](https://www.infoworld.com/article/4012067/google-unveils-gemini-cli-for-developers.html)、[Zed](https://zed.dev/blog/bring-your-own-agent-to-zed)、[gemini-cli subagents 文档](https://github.com/google-gemini/gemini-cli/blob/main/docs/core/subagents.md) |
| Qwen Code | 2025-08（阿里通义） | CLI + IDE 插件（Qwen3-Coder 系） | ~25.9k 星（star-history 全球 #1460）；v0.6.0；另有 Rust 重写 qwen-code-rust | [百度百科](https://baike.baidu.com/item/Qwen%20Code/66293865)、[star-history](https://www.star-history.com/qwenlm/qwen-code/)、[qwen-code 仓库](https://github.com/QwenLM/qwen-code)、[qwen-code-rust](https://github.com/hscale/qwen-code-rust) |
| OpenCode | 2024 底出现，2025 爆发 | 开源终端 agent（TS/Rust，插件体系） | 2026-03 破 120k 星、Hacker News 登顶；模型无关/零成本；"主 Agent + 子 Agent 分层" | [TheAgentTimes](https://theagenttimes.com/articles/opencode-crosses-120k-github-stars-as-open-source-coding-age-3e4b556f)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/05/opencode-120k-github-stars-zero-cost-model-agnostic-coding-agent)、[阿里云](https://developer.aliyun.com/article/1743990)、[腾讯云 7 插件实测](https://cloud.tencent.cn/developer/article/2661551) |
| Cursor 2.0 | 2025-12/2026-01 | AI-first IDE | 自研代码模型 Composer（250 tokens/s、MoE+强化学习）；8 个代理并行；速度提升 4 倍 | [Cursor 官方博客](https://cursor.ac.cn/blog/2-0)、[InfoQ](https://www.infoq.cn/article/qlbwols6tlq36paygyf5)、[InfoWorld](https://www.infoworld.com/article/4081431/cursor-2-0-adds-coding-model-ui-for-parallel-agents.html) |
| Windsurf (Cognition) | 2025 全年迭代 | AI IDE（Cascade agent） | Wave 13 引入 SWE-1.5 快模型、多会话并行 + git worktree；2025 Gartner MQ AI 代码助手 Leader | [ZOL](https://ai.zol.com.cn/1106/11066810.html)、[Cognition](https://cognition.com/blog/swe-1-5)、[WWT](https://www.wwt.com/article/partner-pov-cognition-windsurf-named-a-leader-in-the-2025-gartnerr-magic-quadranttm-for-ai-code-assistants) |
| Claude Cowork | 2025-12/2026-01 | "面向所有人"的 Claude Code（桌面/Web/Chrome） | RBAC + MCP 工具级权限；跨设备会话；Anthropic 用 AI 仅一周半做出 Cowork；阿里、MiniMax 等跟随入场 | [IT之家](https://www.ithome.com/0/912/701.htm)、[IT之家 2](https://www.ithome.com/0/913/115.htm)、[eWeek](https://www.eweek.com/news/claude-cowork-chrome-cross-device-sessions/)、[NBD](https://m.nbd.com.cn/articles/2026-02-05/4251318.html) |
| ChatGPT Work / Operator | 2026-01~06 | 云端 agent + computer use | ChatGPT Work 可连续工作数小时、融入业务流程；Operator 为 computer use 智能体 | [澎湃](https://m.thepaper.cn/newsDetail_forward_33586633)、[coasty.ai](https://coasty.ai/blog/openai-operator-review-2026-20260402)、[DataCamp 对比](https://www.datacamp.com/zh/blog/chatgpt-work-vs-claude-cowork) |

- **格局结论**：2025 年形成"四大 CLI agent（Claude Code / Codex CLI / Gemini CLI / Qwen Code）+ 两大 AI IDE（Cursor / Windsurf）+ 开源黑马（OpenCode 等）"格局；2026 年竞争重心从"模型"转向"harness 工程"（"The Harness Matters More Than the Model"；COSCUP 2026 亦有专题演讲《透過 ClaudeCode、Gemini CLI、Codex 來了解 harness》）。[知乎终端助手指南](https://zhuanlan.zhihu.com/p/2024146096939614452)、[Zylos 研究](https://zylos.ai/en/research/2026-02-21-ai-agent-cli-frameworks/)、[dev.to](https://dev.to/zira125/claude-code-vs-codex-vs-gemini-cli-the-harness-matters-more-than-the-model-h69)、[COSCUP 2026](https://pretalx.coscup.org/coscup-2026/talk/M8WTHA/)

### 2.2 采用率实证：编码 agent 已在 GitHub 大规模普及

- **arXiv 2601.18341《Agentic Much? Adoption of Coding Agents on GitHub》**（2026-01-26 发表，cs.SE）：
  - 样本：129,134 个 GitHub 项目（首次大规模研究）；
  - 结论：2025 上半年编码 agent 采用率估计 **15.85%–22.60%**，"对于一个只有几个月历史的技术来说非常高"，且仍在上升；
  - 采用者画像：横跨项目成熟度全谱系、包括成熟组织、覆盖多种语言与主题；
  - 提交层面：agent 辅助的提交比纯人类提交更大，且功能与修 bug 占比高；
  - 方法：利用 agent 留下的显式痕迹（co-authored commits / PR）识别。[EmergentMind](https://www.emergentmind.com/papers/2601.18341)、[arxivlens](https://arxivlens.com/paperview/details/agentic-much-adoption-of-coding-agents-on-github-3623-7fa51de8)、[ADTmag](https://adtmag.com/articles/2026/05/27/ai-coding-agents-are-already-spreading-across-github.aspx)
- **后续信号**：2026-06 分析《Agentic Very Much》称新 GitHub 项目中编码 agent 采用率已翻倍（并讨论对 Codex CLI 团队配置的含义）。[codex.danielvaughan.com](https://codex.danielvaughan.com/2026/06/18/agentic-very-much-coding-agent-adoption-doubled-new-github-projects-codex-cli-team-configuration/)
- **另一研究**：arXiv 2603.23802《How are AI agents used? Evidence from 177,000 MCP tools》基于 GitHub 上 MCP 服务器清单实证 agent 工具使用行为。[arXiv](https://ar5iv.labs.arxiv.org/html/2603.23802)
- **值得跟踪的索引**：MIT AI Agent Index（aiagentindex.mit.edu）对 Codex 等 agent 做了年度收录。[MIT AI Agent Index](https://aiagentindex.mit.edu/2025/codex/)

### 2.3 主流产品逐个看

#### 2.3.1 Claude Code（Anthropic，闭源）

- 能力演进：2025-02 公开后快速迭代，2025 年内陆续加入 subagents、hooks、MCP 支持、Agent Skills（2025-10）、Agent SDK（2025-10 开源 Python/TS）、Agent Extensions / 插件（2025-11）与 agent teams；CHANGELOG 可追溯全部演进。[Anthropic 官方博客](https://www.anthropic.com/news/enabling-claude-code-to-work-more-autonomously)、[Claude Code CHANGELOG](https://github.com/anthropics/claude-code/blob/4dc23d0275ff615ba1dccbdd76ad2b12a3ede591/CHANGELOG.md)、[Claude 官方博客（插件）](https://claude.com/blog/claude-code-plugins)、[Open Source For You](https://www.opensourceforu.com/2025/10/anthropic-releases-open-source-claude-agent-sdk-alongside-claude-sonnet-4-5-breakthrough/)
- 商业数据：约 $2.5B ARR（2026-04）；被 Anthropic 设计负责人称为"历史上增长最快的营收产品"；Bloomberg（2026-02）称 Claude Code 是把 Anthropic 推向 AI 巨头位置的"意外爆款"。[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/25/claude-code-25b-arr-fastest-ai-developer-tool-billion-dollar-revenue)、[Product Podcast](https://productschool.com/resources/product-podcast/meaghan-choi-anthropic-claude-code-fastest-growing-dev-tool)、[Bloomberg](https://www.bloomberg.com/news/articles/2026-02-20/the-surprise-hit-that-made-anthropic-into-an-ai-juggernaut-mlve4nc2)
- 架构事实（VILA-Lab《Dive into Claude Code》系统拆解，arXiv 2604.14228）：子 agent 通过 "sidechain transcripts" 委派；内置多种 agent 定义；多实例协调用文件锁而非消息队列/分布式协调器；会话转录为可读文件（社区 recensa 可"read, search, replay, audit every session"）。[VILA-Lab](https://github.com/VILA-Lab/Dive-into-Claude-Code)、[arXiv 2604.14228](https://ar5iv.labs.arxiv.org/html/2604.14228)、[recensa](https://github.com/S40911120/recensa)、[cnblogs 解读](https://www.cnblogs.com/jesse123/p/22406128)
- 扩展模型：MCP / skills / hooks / subagents / agent teams / plugins，全部为"外挂层"扩展（核心闭源不可改）。[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)、[towardsai 解读](https://pub.towardsai.net/claude-code-extensions-explained-skills-mcp-hooks-subagents-agent-teams-plugins-9294907e84ff)

#### 2.3.2 OpenAI Codex CLI（开源）

- 形态：TS 启动器 + Rust 二进制（开源），云端 Codex（Web/IDE 版）闭源；支持 MCP、hooks、skills；有沙箱与审批模式。[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)、[百度百科](https://baike.baidu.com/item/Codex%20CLI/65594415)
- 2026 扩展：Codex 支持 Windows 电脑控制（computer use），手机可远程派活——"这些方向的公司要火了"。[腾讯云](https://cloud.tencent.cn/developer/article/2695965)
- 关键评价：开源 ≠ 可组合——Codex 仍是"单体核心 + 外挂扩展"，agent loop 本身不可通过插件替换，这是与 dsh 的关键区别。[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)

#### 2.3.3 Gemini CLI（Google，开源）

- 2025-06-25 开源；支持 subagents（官方文档 core/subagents.md）、AGENTS.md 等；作为 ACP 参考实现与 Zed 深度合作。[china.org.cn](http://www.china.org.cn/world/Off_the_Wire/2025-06/26/content_117947933.shtml)、[gemini-cli subagents 文档](https://github.com/google-gemini/gemini-cli/blob/main/docs/core/subagents.md)、[Zed BYOA](https://zed.dev/blog/bring-your-own-agent-to-zed)
- 生态位：谷歌系 coding agent 代表，与 Qwen Code 一起是"模型厂商自家 CLI"路线的典型。[deeplearning.ai 课程](https://www.deeplearning.ai/courses/gemini-cli-code-and-create-with-an-open-source-agent)、[ai-agent-guidebook](https://github.com/ai-infra-curriculum/ai-agent-guidebook/blob/main/guides/gemini-cli/README.md)

#### 2.3.4 Qwen Code（阿里，开源）

- 2025-08 由阿里云通义千问团队推出；官方定位 "an open-source AI coding agent that lives in your terminal"，CLI + IDE 双形态；配套 Qwen3-Coder 模型（"Agentic Coding in the World"）；~25.9k 星。[qwen-code 仓库](https://github.com/QwenLM/qwen-code)、[Qwen 官方博客](https://qwenlm.github.io/blog/qwen3-coder/)、[百度百科](https://baike.baidu.com/item/Qwen%20Code/66293865)、[star-history](https://www.star-history.com/qwenlm/qwen-code/)
- 2026 动态：v0.6.0 系列持续迭代，另有社区 Rust 重写 qwen-code-rust（含 ACP 支持文档）。[newreleases](https://newreleases.io/project/github/QwenLM/qwen-code/release/v0.6.0-nightly.20251226.3787e955)、[qwen-code-rust](https://github.com/hscale/qwen-code-rust)

#### 2.3.5 OpenCode（开源黑马）

- 增长：2026-03 破 120k 星、Hacker News 登顶，被称"模型无关、零成本、改写模型锁定规则"的终端编码 agent；"甚至 Anthropic 的法律威胁也没能拖慢它"。[TheAgentTimes](https://theagenttimes.com/articles/opencode-crosses-120k-github-stars-as-open-source-coding-age-3e4b556f)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/05/opencode-120k-github-stars-zero-cost-model-agnostic-coding-agent)、[topaiproduct](https://topaiproduct.com/2026/03/20/opencode-crossed-120k-github-stars-and-even-anthropics-legal-threats-couldnt-slow-it-down/)
- 架构：主 Agent 与子 Agent 分层架构；插件体系（hook 支持响应式子 agent 派生、模型继承等）；社区大量配置/插件教程（"7 个常用插件实测"）。[阿里云](https://developer.aliyun.com/article/1743990)、[opencode issue #20387](https://github.com/anomalyco/opencode/issues/20387)、[腾讯云](https://cloud.tencent.cn/developer/article/2661551)

#### 2.3.6 Cursor 2.0（Anysphere，闭源 IDE）

- 2025-12/2026-01 发布：自研代码模型 Composer（约 250 tokens/s 生成、MoE + 强化学习）；多代理 UI，最多 8 个代理并行；整体速度提升 4 倍。[Cursor 官方博客](https://cursor.ac.cn/blog/2-0)、[InfoQ](https://www.infoq.cn/article/qlbwols6tlq36paygyf5)、[InfoWorld](https://www.infoworld.com/article/4081431/cursor-2-0-adds-coding-model-ui-for-parallel-agents.html)、[BAAI](https://hub.baai.ac.cn/view/49954)

#### 2.3.7 Windsurf（Cognition，闭源 IDE）

- Wave 13：引入 SWE-1.5（Cognition 自己的"快速 agent 模型"）、多会话并行与 git worktree 全面升级；Cascade 是其 agent 体系；Cognition（Windsurf）被评为 2025 Gartner MQ AI 代码助手 Leader。[ZOL](https://ai.zol.com.cn/1106/11066810.html)、[Cognition](https://cognition.com/blog/swe-1-5)、[WWT](https://www.wwt.com/article/partner-pov-cognition-windsurf-named-a-leader-in-the-2025-gartnerr-magic-quadranttm-for-ai-code-assistants)、[Windsurf 文档](https://docs.windsurf.com/zh/windsurf/cascade/cascade)

#### 2.3.8 Claude Cowork（Anthropic，2025-12/2026-01）

- 定位："面向所有人版本的 Claude Code 助手"，把编程 agent 能力带给普通用户；支持 Chrome 浏览器、跨设备会话；新增 RBAC 与 MCP 工具级权限控制；"Anthropic 用 AI 写智能体，Claude 仅用一周半做出 Cowork"；国内媒体称"Claude Cowork 爆火，阿里、MiniMax 等悉数入场"。[IT之家](https://www.ithome.com/0/912/701.htm)、[IT之家 2](https://www.ithome.com/0/913/115.htm)、[eWeek](https://www.eweek.com/news/claude-cowork-chrome-cross-device-sessions/)、[digitaltoday](https://www.digitaltoday.co.kr/cn/view/46886)、[NBD](https://m.nbd.com.cn/articles/2026-02-05/4251318.html)、[DataCamp 对比](https://www.datacamp.com/zh/blog/chatgpt-work-vs-claude-cowork)

### 2.4 架构共性：agent harness 的"标准零件"

各家表面不同，但 2025–2026 已收敛出一套共同架构组件：

1. **Agent 循环（loop）**：Claude Code 的动态工作流本质是"while 循环"驱动的 agent 循环（"一个 while 循环凭什么干掉状态机"）；OpenCode 采用主/子 Agent 分层；dsh 的 dsh-agent-loop 提供 turn/step 驱动与并行调度；VILA-Lab 论文系统拆解了 Claude Code 的设计空间。[乐小野拆解](https://cloud.tencent.cn/developer/article/2685109)、[阿里云](https://developer.aliyun.com/article/1743990)、[arXiv 2604.14228](https://ar5iv.labs.arxiv.org/html/2604.14228)、[VILA-Lab](https://github.com/VILA-Lab/Dive-into-Claude-Code)
2. **事件日志 / 会话转录**：Claude Code 把每轮会话存为 JSONL 转录（recensa 等工具可读/搜/回放/审计）；dsh 的 append-only 轨迹流；OpenCode 亦有会话回放。**"每次运行都有迹可循"已是主流 harness 标配**。[recensa](https://github.com/S40911120/recensa)、[智东西](https://www.zhidx.com/p/584897.html)
3. **工具（Tools）+ MCP**：工具调用从"各家私有 schema"收敛到 MCP 统一协议（详见第三章）；原生工具集（文件/shell/搜索/浏览器）仍由各家实现，MCP 负责长尾集成。[Morph](https://www.morphllm.com/agent-client-protocol)、[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)
4. **沙箱 / 权限**：Claude Code 有精细权限系统（允许/拒绝/询问），Cowork 新增 RBAC 与 MCP 工具级权限；dsh 提供 dsh-sandbox-local / bash-sandbox / pwsh-sandbox（workspace 围栏）；e2b（云端沙箱）成为通用后端；社区有 opengap 等"Isolation Policy（状态与凭据隔离级别）"规范。[digitaltoday](https://www.digitaltoday.co.kr/cn/view/46886)、[博客园](https://www.cnblogs.com/qq8864/articles/22479803)、[opengap](https://github.com/open-gitagent/opengap/blob/HEAD/spec/SPECIFICATION.md)
5. **插件 / 扩展机制**：Claude Code 2025-11 推出 Agent Extensions 与插件；OpenCode 有插件系统（hook 支持响应式子 agent 派生）；dsh 是运行时级插件；Cursor 用 rules + MCP；Codex 用 hooks + skills。[Zed 博客](https://zed.dev/blog/acp-registry)、[opencode issue](https://github.com/anomalyco/opencode/issues/20387)、[Claude 插件博客](https://claude.com/blog/claude-code-plugins)
6. **AGENTS.md（项目指令文件）**：2025 年成为开放事实标准——仓库根目录用自然语言给 agent 下项目级指令的 Markdown；InfoQ 2025-08 报道 "AGENTS.md Emerges as Open Standard"；GitHub 研究将其作为 agent 采用代理指标；heise 讨论其 token 成本争议（"helpful briefing or token hog"）；dsh 官方仓库自带 AGENTS.md + CLAUDE.md（symlink）。[InfoQ](https://www.infoq.com/news/2025/08/agents-md/)、[socket.dev](https://socket.dev/blog/agents-md-gains-traction-as-an-open-format-for-ai-coding-agents)、[heise](https://www.heise.de/en/background/AGENTS-md-Helpful-agent-briefing-or-token-hog-11245317.html)、[arXiv 2601.18341](https://ar5iv.labs.arxiv.org/html/2601.18341)、[dsh AGENTS.md](https://github.com/deepseek-ai/deepseek-harness/blob/master/AGENTS.md)
7. **Skills（技能）**：Anthropic 2025-10 发布 Agent Skills（SKILL.md），"先给技能目录、任务匹配才展开完整说明书"，省 token、可移植；跨 agent 采纳中（详见第三章）。[腾讯云实战](https://cloud.tencent.com.cn/developer/article/2698677)、[Simon Willison](https://simonwillison.net/2025/Dec/19/agent-skills/)
8. **ACP（Agent Client Protocol）**：agent 与编辑器/客户端之间的协议（详见第三章）；Gemini CLI、Claude Code、Goose 已接入，Codex/Aider/Cursor 适配中；2026-01-28 ACP Registry 上线。[Zed](https://zed.dev/blog/acp-progress-report)、[Zed Registry](https://zed.dev/blog/acp-registry)
9. **子 Agent（subagents）**：Claude Code 内置多种子 agent（VILA-Lab 拆出 6 种内置 agent 定义，sidechain transcripts 委派）；dsh 把子 agent 做成可插拔（subagent-claude-code / subagent-codex / subagent-acp / subagent-in-process-driver 等）；Cursor 2.0 支持 8 个并行 agent；OpenCode 主/子分层。[VILA-Lab](https://github.com/VILA-Lab/Dive-into-Claude-Code)、[dsh packages/subagent](https://github.com/deepseek-ai/deepseek-harness/tree/master/packages/subagent)、[InfoQ](https://www.infoq.cn/article/qlbwols6tlq36paygyf5)

---

## 三、MCP / Skills / ACP / A2A 协议生态现状

### 3.1 MCP（Model Context Protocol）：工具层的"USB-C"

- **定义与定位**：一个标准 JSON-RPC 接口，规范 agent 与外部工具/数据源之间的工具发现、调用与结果处理；官方自评"从一个小型开源实验变成连接数据与应用到 LLM 的事实标准"。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **时间线**：2024-11-25 开源 → 2025-09 官方 MCP Registry 上线 → 2025-11-25 一周年规范发布（授权扩展、SEP 治理、Working/Interest Groups，SEP-1302 设立工作组/兴趣组机制）→ 2025-12 捐赠 Linux Foundation 旗下 Agentic AI Foundation（AAIF）→ 2026-03 Registry 12,000+ → 2026-04 生态破万。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **规模数据（2026-04 口径）**：
  - 活跃公共 MCP 服务器 **10,000+**；
  - 月 SDK 下载 **9,700 万**（对比：React 约 3 年才达 1 亿月下载，MCP 16 个月）；
  - 官方 MCP Registry（2026-03，Anthropic/GitHub/Microsoft/PulseMCP 支持）列出 **12,000+** 服务器条目，top50 服务器月搜索 62.2 万+；
  - TypeScript SDK 有 **34,700+** 依赖的 npm 项目；
  - 社区 HuggingFace 注册表镜像：5,033 个（2026-06-21）→ 5,833 个（2026-07-25）。[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)、[HF Vinkius/mcp-registry](https://huggingface.co/datasets/Vinkius/mcp-registry)
- **服务器类别分布**：开发工具 1,200+（GitHub/GitLab/Jira/Docker/K8s）、数据库 800+（PostgreSQL/MongoDB/MySQL/Snowflake）、业务应用 950+（Salesforce/HubSpot/Notion/Stripe/Workday）、通信 600+（Slack/Gmail/Teams/Discord）、云与 DevOps 700+（AWS/GCP/Azure/Terraform/Grafana）、分析与可观测 400+（Datadog/Prometheus 等）；GitHub 官方 MCP server 是部署最广的。[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **官方生态案例（一周年博客列举）**：Notion、Stripe、GitHub、Hugging Face、Postman 等官方 MCP server；"如果你能想到一个场景，大概率已有对应 MCP server"。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)
- **治理**：AAIF 创始成员 Anthropic、Block、OpenAI；铂金赞助 AWS、Google、Microsoft、Bloomberg、Cloudflare；截至 2026 初 146 家成员；社区活动 MCP Dev Summit / MCP Night / MCP Dev Days，官方调试器 MCP Inspector。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **客户端采纳**：ChatGPT、Gemini、Cursor、Microsoft Copilot Studio、VS Code、Windows 11、JetBrains、Windsurf、Claude Desktop 等全部内置 MCP；OpenAI 高管表态"OpenAI 从早期就贡献 MCP，现为 ChatGPT 与开发者平台的关键部分"。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)、[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **安全问题**：行业讨论"安全债务"（17 个 SEP、权限/授权仍是短板）；官方一周年即发布 authorization extensions。[AgentMarketCap 安全文](https://agentmarketcap.ai/blog/2026/04/06/mcp-18-months-5800-servers-security-debt-enterprise-adoption)、[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)
- **中文生态**：2026 上半年全览文章梳理协议进展与工具链；腾讯云开发者社区开设"MCP 广场"。[LearnAgent](https://learnagent.org/library/updates/mcp-ecosystem-2026-h1/)、[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)
- **对自研产品的含义**：MCP 已是"接就活、不接就死"的集成层；单用户本机助手也应至少作为 MCP client 提供工具接入，理想情况可提供本机 MCP server（如文件/笔记/图片生成能力）。

### 3.2 Agent Skills（SKILL.md）：第二项跨 agent 标准

- **定义**：Anthropic 2025-10 随 Claude 发布 Agent Skills——用 `SKILL.md`（frontmatter 含 name/description 等）描述可复用能力，把"完整说明书"拆为"技能目录 + 按需展开"，省 token、可移植；"换了一个思路：先给 Agent 一份技能目录，只有任务匹配时才展开完整说明书"。[腾讯云实战](https://cloud.tencent.com.cn/developer/article/2698677)、[Simon Willison](https://simonwillison.net/2025/Dec/19/agent-skills/)
- **采纳**：Claude Code / Codex / Cursor / Gemini CLI 等陆续支持 SKILL.md；社区出现 skills-mcp（用 MCP 暴露技能）、agentpatterns-ai（标准收集）、ClawHub / Skill Hub（技能分发，openJiuwen 的 skillhub 兼容 ClawHub）；atlan 2026 报告分析其格式与采纳。[skills-mcp](https://github.com/skills-mcp/skills-mcp)、[agentpatterns](https://github.com/agentpatterns-ai/website/blob/main/standards/agent-skills-standard.md)、[atlan](https://atlan.com/know/ai-agent/ai-agent-skills/what-are-agent-skills/)
- **激进观点**："One SKILL.md replaces CLAUDE.md, AGENTS.md, and .cursorrules"——技能文件可能统一替代各家项目指令格式。[dev.to](https://dev.to/creeta/one-skillmd-replaces-claudemd-agentsmd-and-cursorrules-5a70)
- **dsh 关联**：dsh 官方内置技能（dsh-code-review、dsh-doc-standards 等），会话内自动注入 skill 目录；社区 dsh-skillport 把 Claude Code/Codex/Cursor/Gemini CLI 的 SKILL.md 迁入 dsh（"Every skill you already have — works in DSH"）。**SKILL.md 已是跨 harness 通用技能格式**。[dsh-skillport](https://github.com/Jesse-njx/dsh-skillport)、[博客园](https://www.cnblogs.com/qq8864/articles/22479803)

### 3.3 ACP（Agent Client Protocol）：agent ↔ 编辑器/客户端

- **发起**：Zed 为"把更多 agent 带进 Zed"设计（与其逐个集成，不如做一个任何 agent 都能实现的协议）；与 Google 合作把 Gemini CLI 做成参考实现；理念是"agent 专注逻辑，编辑器负责 UX"。[Zed 进展报告](https://zed.dev/blog/acp-progress-report)、[Zed BYOA](https://zed.dev/blog/bring-your-own-agent-to-zed)
- **生态（2025-10 时点）**：
  - 客户端：Neovim（CodeCompanion、avante.nvim）、Emacs（agent-shell）、marimo notebook；进行中：Eclipse、Toad；
  - Agent：Gemini CLI（参考实现）、Claude Code（Zed SDK adapter）、Goose（Square 开源 agent）；进行中：Codex（官方适配 + 社区版）、Aider、Cursor（社区 adapter）。[Zed 进展报告](https://zed.dev/blog/acp-progress-report)
- **Registry**：2026-01-28 "The ACP Registry is Live"，Zed 与 JetBrains 联合发布，可在 JetBrains IDE 查找并连接 ACP 编码 agent。[Zed Registry](https://zed.dev/blog/acp-registry)、[JetBrains 博客](https://blog.jetbrains.com/ai/2026/01/acp-agent-registry/)、[heise](https://www.heise.de/en/news/Integrate-AI-Agents-into-the-Editor-JetBrains-and-Zed-Release-ACP-Registry-11160709.html)
- **采纳信号**：Google 与 OpenAI 先后采纳 Anthropic 主导的 ACP（ZDNet："Google joins OpenAI in adopting Anthropic's protocol for connecting AI agents"）。[ZDNet](https://www.zdnet.com/article/google-joins-openai-in-adopting-anthropics-protocol-for-connecting-ai-agents-why-it-matters/)
- **dsh 关联**：dsh 内置 subagent-acp 包（把 ACP agent 作为子 agent 驱动）。[dsh packages/subagent](https://github.com/deepseek-ai/deepseek-harness/tree/master/packages/subagent)

### 3.4 A2A（Agent2Agent）：agent ↔ agent

- **定义**：Google 2025-04 发布，解决"独立 agent 之间如何发现彼此、协商任务委派、交换结果"的水平互操作问题；业界比喻为"OS 中的进程间通信"。[腾讯云解读](https://cloud.tencent.cn/developer/article/2664955)
- **治理**：2025-06-25 Google 将 A2A 捐赠 Linux Foundation（新设 Agent2Agent 项目，与 MCP 同归基金会治理），AWS 宣布加入支持；维基百科已收录词条。[Google Developers Blog](https://developers.googleblog.com/en/google-cloud-donates-a2a-to-linux-foundation/)、[Linux Foundation 新闻稿](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents)、[Digitimes](https://apps.digitimes.com/news/a20250625PD205/google-ai-agent-linux-aws.html)、[维基百科](https://zh.wikipedia.org/zh-hk/Agent2Agent)
- **与 MCP 分工**：MCP = 垂直连接（agent→工具/数据）；A2A = 水平连接（agent↔agent）；2026 生产系统两者分层并用（客服 agent 用 MCP 查 CRM/知识库/工单，用 A2A 把复杂问题转交给其他模型构建的专业 agent）。[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)

### 3.5 协议格局小结

- **三层协议栈已成共识**：MCP（工具层）＋ A2A（agent 间层）＋ ACP（客户端/编辑器层）；三者同归开放治理（AAIF/LF 等），"协议战争"在 2026 已基本结束，进入"分层共建"阶段；
- 主流 harness 同时在实现三类协议：openJiuwen 的 agent-protocol 提供 MCP/A2A 的 C++ SDK；DSH 本机依赖树同时出现 @agentclientprotocol/sdk 与 @modelcontextprotocol/*；qwen-code-rust 文档亦含 ACP 章节。[博客园对比文](https://www.cnblogs.com/qq8864/articles/22479803)、[qwen-code-rust ACP 文档](https://github.com/hscale/qwen-code-rust/blob/main/docs/acp.md)
- **术语对照**：MCP ≈ 设备总线（工具）；A2A ≈ 进程间通信（agent 间）；ACP ≈ 显示器/外设接口（客户端-agent 会话）。[Morph 对比文](https://www.morphllm.com/agent-client-protocol)

---

## 四、个人 AI 助手的"工作台化"趋势

### 4.1 从"聊天框"到"工作台"：2026 年的明确转向

- **标志性事件（2026-06）**：腾讯 WorkBuddy 增加 Teams、项目看板、资产库、版本历史；字节跳动将 SOLO 更名为 **TRAE Work**（从独立开发者工具扩展为面向各类专业人士的 AI 工作空间）；阿里整合 **QoderWork、悟空、MuleRun** 产品线，形成统一企业 AI 生产力产品。[腾讯云开发者社区（WebOffice 专栏）](https://cloud.tencent.com.cn/developer/article/2708771)
- **三家起点不同、终点趋同**：腾讯（AI 编程助手 → 个人工作助手）、阿里（编程 Agent → 任务执行工具）、字节（TRAE 开发者底盘 → 编程 Agent）都长出了"文档/设计/幻灯片/项目管理/协作/版本管理"能力。[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)
- **核心论点**：聊天框可以表达目标（做报告/生成 PPT/整理项目计划/分析销售数据），但**无法承载工作成果**——报告要交付文档、经营分析要表格图表、项目要分工/进度/版本；"聊天框可以成为工作的起点，却很难成为工作的容器"。[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)
- **第一波尝试**："聊天负责沟通，成果需要独立的空间承载"——Claude Artifacts、Microsoft Copilot Pages、ChatGPT Canvas；2026 年演化为完整工作台：QoderWork 提供设计/幻灯片/写作工作空间（生成后可继续调整、比较、导出）；WorkBuddy 增加项目/团队/任务/版本（把个人与 AI 的对话扩展为团队分工交接的工作流）；TRAE Work 把需求、设计、代码、交付放进同一上下文。[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)
- **为什么是编程 Agent 先转型**：编程 Agent 最早跑通完整执行闭环（理解任务→使用工具→修改成果→检查结果→持续迭代），最早意识到"仅仅生成文本不够，必须进入真实工作环境、操作真实文件、维护任务状态、交付可继续使用的成果"；该模式正从软件开发外溢到研究、数据分析、写作、设计、项目管理。[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)
- **反面路线 OfficeCLI**：另一条路线是"Office 能力直接开放给 Agent"（创建/读取/修改/检查 Office 文件，而非让 agent 模拟人点击 GUI）；两条路线终将汇合为"上层是人的工作空间，下层是 Agent 可调用的能力，中间由任务/流程/数据/权限连接"。[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)
- **"人和 Agent 双原生"**：未来 Office 同时存在面向人的使用方式（阅读/编辑/审阅/协作/确认）与面向 Agent 的使用方式（读取/生成/修改/推动流程/检查结果）；竞争指标从"功能是否丰富"转向"能否被 Agent 调用、能否承载完整任务、能否控制 Agent 权限、能否记录/检查/回滚 Agent 操作"。[腾讯云](https://cloud.tencent.com.cn/developer/article/2708771)

### 4.2 桌面与工作场景的 agent 卡位战

- 中文媒体用"桌面办公 Agent 卡位战：谁先占领你的电脑？"、"腾讯谷歌们的 AI 工位新战事：让马维斯们替打工人打工靠谱吗？"、"从 ChatGPT 到 WorkBuddy：谁会成为下一代 AI 工作台？"来描述这一轮竞争；2026 WAIC 主题被概括为从"能说"迈向"会干"。[钛媒体](https://www.tmtpost.com/8099969.html)、[界面新闻](https://www.jiemian.com/article/14910643.html)、[OFweek](https://www.ofweek.com/ai/2026-06/ART-201712-8110-30689162.html)、[腾讯云](https://cloud.tencent.com.cn/developer/article/2709015)、[搜狐](https://news.sohu.com/a/1051880219_122014422)
- **海外对标**：a16z 提出"The AI-Native Office Suite——Can AI Do Work For You?"；钛媒体英文版报道"The Office Agent Race Shifts From Chatbots to Organizational Work"；OpenDataScience 提出"Personal AI Agents Are The Next Operating Layer For Work"；Skyclaw 等 2026 个人 AI agent 产品崛起。[a16z](https://a16z.com/the-ai-native-office-suite-can-ai-do-work-for-you/)、[tmtpost 英文](https://www.tmtpost.com/8092632.html)、[OpenDataScience](https://opendatascience.com/personal-ai-agents-are-the-next-operating-layer-for-work/)、[skywork](https://skywork.ai/blog/ai-agent/skyclaw-rise-personal-ai-agents/)
- **福布斯 2025 AI 50**："AI Agent 全面崛起，应用层才是 2025 真正的主战场"。[BAAI 转载](https://hub.baai.ac.cn/view/44849)

### 4.3 "能动手"：computer use 类能力成为工作台标配

- **OpenAI**：ChatGPT Work 发布（2026-06，AI 助手可连续工作数小时、深度融入业务流程）；Operator 作为 computer use 智能体（评测视角："keep asking you to do its job"说明人机协作仍是难点）；Codex 支持 Windows 电脑控制、手机远程派活。[澎湃](https://m.thepaper.cn/newsDetail_forward_33586633)、[163](https://m.163.com/dy/article/L1F5K9HT0511BLFD.html)、[腾讯云](https://cloud.tencent.cn/developer/article/2695965)、[coasty.ai](https://coasty.ai/blog/openai-operator-review-2026-20260402)
- **Google**：Gemini 3 系列支持 computer use（Gemini API 提供 Computer Use 能力）；Gemini 3.5 Flash 把"看屏幕 + 控制电脑"做成内置工具，主推企业信任与安全；媒体同步警示"攻击者已开始针对 AI agent"。[Google AI for Developers](https://ai.google.dev/gemini-api/docs/computer-use)、[TheNextWeb](https://thenextweb.com/news/google-gemini-3-5-flash-computer-use-built-in-tool)、[Search Engine Journal](https://www.searchenginejournal.com/google-gemini-can-now-control-your-computer-hackers-are-already-targeting-ai-agents/580578/)
- **Anthropic**：Claude Cowork 把 Claude Code 的 agent 能力带给普通用户，强调 RBAC 与 MCP 工具级权限；"Claude Cowork 爆火，阿里、MiniMax 等悉数入场"。[IT之家](https://www.ithome.com/0/912/701.htm)、[NBD](https://m.nbd.com.cn/articles/2026-02-05/4251318.html)
- **趋势归纳**："从能说迈向会干"——2025 年 AI 产品比"说得好"，2026 年比"干得成"；能干的前提是：工具（MCP）、沙箱/权限（安全边界）、事件日志（可追溯）、工作台（成果容器）。[搜狐](https://news.sohu.com/a/1051880219_122014422)

### 4.4 对"单用户本机桌面助手"（gaea 场景）的含义

- 个人 AI 助手的演进路径可概括为：**聊天框 → 带工具的执行器 → 带会话/轨迹/状态的工作台 → 可被 Agent 调用的本地能力平台**；
- 2025–2026 证据链表明：**工作台不是聊天框的增强，而是新的产品形态**（文档、项目、版本、任务、权限、协作是骨架，Agent 是执行层）；
- 单用户本机产品对应组合：轻量工作台 + 本地文件系统/沙箱 + 会话事件日志 + 跨会话记忆 + 模型/工具/技能可插拔；
- 本机（local-first）仍有生态位：桌面办公 Agent 卡位战报道指出"谁占领你的电脑"决定下一轮入口；本地沙箱（bash-sandbox/pwsh-sandbox、e2b）、本地会话存储（JSONL/SQLite）是主流 harness 标配，也是 dsh 直接示范的形态。[钛媒体](https://www.tmtpost.com/8099969.html)、[博客园](https://www.cnblogs.com/qq8864/articles/22479803)

---

## 五、对 gaea 3.0 的初步启示（供后续调研细化）

### 5.1 架构对标清单

若 gaea 3.0 向 DSH 靠拢，建议优先吸收以下可迁移特性：

1. **会话/轨迹事件溯源**：append-only 事件日志（系统提示词/思维链/工具调用与结果/上下文注入），支持恢复、分叉、检索、回放；存储可用 JSONL（可 zstd 压缩）或 SQLite。[智东西](https://www.zhidx.com/p/584897.html)、[dsh session 包](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/session/session-persistence-jsonl/README.md)
2. **能力插件化**：模型适配器（OpenAI 兼容即插即用）、工具注册表、技能（SKILL.md）、存储、沙箱、UI 分层解耦；即使不做完整 Cordis，也应定义"工具/技能/模型"三张注册表与生命周期（安装/卸载/热更新）。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
3. **内置运行模式**：标准 / 极简（基准测试）/ 自动化（类 PTC：模型生成脚本组合多步调用）/ 创造（运行时试验）——模式即插件组合的预设。[IT之家](https://www.ithome.com/0/989/446.htm)
4. **子 agent 与外部 agent 驱动**：本机可把 Claude Code / Codex / 任意 ACP agent 作为"外部子 agent"驱动（DSH 的 subagent-claude-code / subagent-codex / subagent-acp 即示范），用报告/消息机制回传结果。[dsh packages/subagent](https://github.com/deepseek-ai/deepseek-harness/tree/master/packages/subagent)
5. **可观测性**：每次运行"有迹可循"——轨迹视图、token/成本统计、错误回放；这是 2026 主流 harness 的标配。[智东西](https://www.zhidx.com/p/584897.html)

### 5.2 协议与标准接入优先级

| 标准 | 优先级 | 理由（事实依据） |
|---|---|---|
| MCP（client 至少，server 可选） | P0 | 10,000+ 服务器、全主流客户端支持、2026 事实标准；不接等于放弃整个工具生态 | [AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects) |
| AGENTS.md | P0 | 2025 成为开放事实标准，GitHub 仓库普遍采用；"项目即上下文"的载体 | [InfoQ](https://www.infoq.com/news/2025/08/agents-md/) |
| Skills（SKILL.md） | P1 | 跨 Claude Code/Codex/Cursor/Gemini CLI/dsh 的通用技能格式；技能库可移植 | [Simon Willison](https://simonwillison.net/2025/Dec/19/agent-skills/)、[dsh-skillport](https://github.com/Jesse-njx/dsh-skillport) |
| ACP | P2 | agent↔编辑器协议；若 gaea 提供桌面 UI 并想接入外部 agent，或自身作为"agent 宿主"时再接入；DSH 已有参考实现 | [Zed](https://zed.dev/blog/acp-progress-report) |
| A2A | P3 | agent↔agent 水平互操作；单用户本机产品短期无刚需，多 agent 场景（如轻语人格/办公 agent 协作）可后期考虑 | [Linux Foundation](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents) |

### 5.3 工作台化对 gaea 模块的映射（初步）

| gaea 现有模块 | 工作台化方向（依据第四章事实） |
|---|---|
| 聊天 | 保留为"入口/指挥层"，不再作为唯一界面；会话事件日志化（恢复/分叉/回放） |
| 轻语人格 | 可视为"特殊 persona 插件/技能"：人格=上下文模板+行为约束（SKILL.md/AGENTS.md 式描述） |
| 小说创作 | 需要"成果容器"：文档/章节/版本管理（对齐 QoderWork 工作空间、Artifacts 思路） |
| 绘梦 | 作为工具注册表的一项能力（本机 MCP server 或原生工具），生成物进入资产库 |
| 办公 agent | 直接对标"编程 Agent → 工作台"路径：任务（理解→工具→修改→检查→交付）+ 项目/版本视图 |
| 记忆中枢 | 对齐跨会话记忆事实：dsh 会话事件溯源 + OpenCode/Claude 会话转录 + openJiuwen agent-memory；建议事件日志与"记忆摘要"双层 |
| 模型中心 | 对齐模型适配器：OpenAI 兼容端点即插即用（dsh ctx.llm.registerAdapter 思路）+ 本地模型 |

### 5.4 定位与差异化建议（事实支撑）

- 单用户本机产品的护城河：本地隐私、离线可用、低延迟、可深度定制（对应"谁能占领你的电脑"的入口之争与 local-first 生态位）；[钛媒体](https://www.tmtpost.com/8099969.html)
- dsh 的"运行时全开放"正是此类产品的架构方向：不做"第二个 Claude Code 外挂"，而做"可自进化的本机工作台"。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- 注意：DSH 团队自己也承认开发期产品粗糙（"预览版本，可能存在很多粗糙之处"），gaea 3.0 不必一次性追求 DSH 的完整插件体系，可先做"能力注册表 + 事件日志 + 工作台"三件套。[智东西](https://www.zhidx.com/p/584897.html)

---

## 附录 A：术语表

- **Harness**：把模型转化为智能体的框架层（上下文调度、工具、任务状态、反馈与边界）——"Model + Harness = Agent"。[IT之家](https://www.ithome.com/0/989/446.htm)
- **Cordis**：DSH 底层插件元框架（加载/卸载/依赖），理念出自北大+DeepSeek 论文。[智东西](https://www.zhidx.com/p/584897.html)
- **可逆效应（revertible effects）**：组件卸载时回滚其副作用的机制，支撑插件热替换。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **反应式余效应（reactive coeffects）**：上下文变化主动通知依赖组件，支撑插件间协作。[博客园](https://www.cnblogs.com/qq8864/articles/22479803)
- **PTC（Programmatic Tool Calling）**：模型生成代码组合多轮工具调用的模式（dsh 内置）。[IT之家](https://www.ithome.com/0/989/446.htm)
- **轨迹（trajectory）**：dsh 的 append-only 会话事件日志与回放视图。[智东西](https://www.zhidx.com/p/584897.html)
- **MCP（Model Context Protocol）**：agent↔工具/数据协议，2024-11 开源，2025-12 入 LF/AAIF。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)
- **A2A（Agent2Agent）**：agent↔agent 协议，2025-04 Google 发布，2025-06 入 Linux Foundation。[Linux Foundation](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents)
- **ACP（Agent Client Protocol）**：agent↔编辑器/客户端协议，Zed 发起，2026-01 发布官方 Registry。[Zed](https://zed.dev/blog/acp-progress-report)、[Zed Registry](https://zed.dev/blog/acp-registry)
- **Agent Skills（SKILL.md）**：可复用技能描述格式，Anthropic 2025-10 发布，跨 agent 采纳。[Simon Willison](https://simonwillison.net/2025/Dec/19/agent-skills/)
- **AGENTS.md**：项目级 agent 指令文件，2025 年成为开放事实标准。[InfoQ](https://www.infoq.com/news/2025/08/agents-md/)
- **AAIF（Agentic AI Foundation）**：Linux Foundation 下治理 MCP（及 A2A）的基金会，2025-12 成立。[AgentMarketCap](https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects)
- **SEP（Specification Enhancement Proposal）**：MCP 规范增强提案机制。[MCP 官方博客](https://modelcontextprotocol.info/blog/first-mcp-anniversary/)
- **computer use**：让 agent 像人一样操作电脑屏幕/应用的能力（Operator、Gemini Computer Use、Codex 电脑控制）。[Google AI for Developers](https://ai.google.dev/gemini-api/docs/computer-use)

## 附录 B：关键来源 URL 汇总

**DeepSeek Harness**
- 官方仓库：https://github.com/deepseek-ai/deepseek-harness （README / README.zh.md、AGENTS.md、docs/subsystems/*、packages/session/session-persistence-jsonl、packages/subagent/*）
- Cordis 论文：https://github.com/cordiverse/paper/blob/main/paper.pdf
- IT之家（公测 + 插件生态 + V4-Pro 参数）：https://www.ithome.com/0/989/446.htm
- 智东西（实测 + 轨迹 + 数据）：https://www.zhidx.com/p/584897.html
- 博客园（架构哲学与主流对比）：https://www.cnblogs.com/qq8864/articles/22479803
- 36氪（黑色鲸鱼）：https://www.36kr.com/p/3938566998834308
- 品玩：https://www.pingwest.com/a/316436
- VentureBeat：https://venturebeat.com/technology/deepseek-harness-launches-as-open-source-rival-to-claude-code-alongside-v4-pro-on-api-with-higher-prices
- Gigazine：https://gigazine.net/news/20260814-deepseek-harness-v0-1/
- dsh-handbook：https://github.com/Electricitysheep/dsh-handbook ；learn-dsh：https://github.com/onychen/learn-dsh ；dsh-skillport：https://github.com/Jesse-njx/dsh-skillport

**格局 / 架构共性**
- arXiv 2601.18341（Agentic Much，采用率研究）：https://www.emergentmind.com/papers/2601.18341
- arXiv 2604.14228（Dive into Claude Code）：https://ar5iv.labs.arxiv.org/html/2604.14228 ；配套仓库 https://github.com/VILA-Lab/Dive-into-Claude-Code
- Claude Code $2.5B ARR：https://agentmarketcap.ai/blog/2026/04/25/claude-code-25b-arr-fastest-ai-developer-tool-billion-dollar-revenue
- Anthropic $14B ARR：https://www.saastr.com/anthropic-just-hit-14-billion-in-arr-up-from-1-billion-just-14-months-ago/ ；Bloomberg：https://www.bloomberg.com/news/articles/2026-02-20/the-surprise-hit-that-made-anthropic-into-an-ai-juggernaut-mlve4nc2
- Cursor 2.0：https://cursor.ac.cn/blog/2-0 ；https://www.infoq.cn/article/qlbwols6tlq36paygyf5
- Windsurf Wave 13：https://ai.zol.com.cn/1106/11066810.html ；SWE-1.5：https://cognition.com/blog/swe-1-5
- Claude Cowork：https://www.ithome.com/0/912/701.htm ；https://www.ithome.com/0/913/115.htm
- Qwen Code：https://baike.baidu.com/item/Qwen%20Code/66293865 ；https://www.star-history.com/qwenlm/qwen-code/ ；https://github.com/QwenLM/qwen-code
- OpenCode 120k：https://theagenttimes.com/articles/opencode-crosses-120k-github-stars-as-open-source-coding-age-3e4b556f
- Zylos CLI 框架研究：https://zylos.ai/en/research/2026-02-21-ai-agent-cli-frameworks/
- AGENTS.md（InfoQ）：https://www.infoq.com/news/2025/08/agents-md/
- ACP 与 MCP 对比（Morph）：https://www.morphllm.com/agent-client-protocol

**协议生态**
- MCP 官方一周年：https://modelcontextprotocol.info/blog/first-mcp-anniversary/
- MCP 10,000 服务器：https://agentmarketcap.ai/blog/2026/04/14/mcp-ecosystem-10000-servers-protocol-network-effects
- MCP 安全债：https://agentmarketcap.ai/blog/2026/04/06/mcp-18-months-5800-servers-security-debt-enterprise-adoption
- HF MCP 注册表镜像：https://huggingface.co/datasets/Vinkius/mcp-registry ；MCP 2026 H1 全览（中文）：https://learnagent.org/library/updates/mcp-ecosystem-2026-h1/
- A2A 捐赠 LF：https://developers.googleblog.com/en/google-cloud-donates-a2a-to-linux-foundation/ ；https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents
- ACP 进展报告：https://zed.dev/blog/acp-progress-report ；ACP Registry：https://zed.dev/blog/acp-registry ；JetBrains：https://blog.jetbrains.com/ai/2026/01/acp-agent-registry/
- Google/OpenAI 采纳 ACP：https://www.zdnet.com/article/google-joins-openai-in-adopting-anthropics-protocol-for-connecting-ai-agents-why-it-matters/
- skills-mcp：https://github.com/skills-mcp/skills-mcp ；Agent Skills（Simon Willison）：https://simonwillison.net/2025/Dec/19/agent-skills/

**工作台化**
- 2026 AI 产品转向"类 Office"工作台（WorkBuddy/TRAE Work/QoderWork）：https://cloud.tencent.com.cn/developer/article/2708771
- 从 ChatGPT 到 WorkBuddy：https://cloud.tencent.com.cn/developer/article/2709015
- 桌面办公 Agent 卡位战：https://www.tmtpost.com/8099969.html ；https://www.jiemian.com/article/14910643.html
- AI 工位新战事：https://www.ofweek.com/ai/2026-06/ART-201712-8110-30689162.html
- ChatGPT Work：https://m.thepaper.cn/newsDetail_forward_33586633
- Gemini Computer Use：https://ai.google.dev/gemini-api/docs/computer-use ；Gemini 3.5 Flash：https://thenextweb.com/news/google-gemini-3-5-flash-computer-use-built-in-tool
- a16z AI-Native Office Suite：https://a16z.com/the-ai-native-office-suite-can-ai-do-work-for-you/
- Office Agent Race：https://www.tmtpost.com/8092632.html ；Personal AI Agents as OS layer：https://opendatascience.com/personal-ai-agents-are-the-next-operating-layer-for-work/

---

## 附录 C：调研方法与局限

**方法**：每主题 2-4 轮中英文检索（共 14 轮 web_search）+ 关键页面精读约 12 个（官方仓库 GitHub、MCP 官方博客、Zed 官方博客、IT之家/智东西/36氪等媒体、arXiv 论文页、博客园/知乎技术社区、腾讯云开发者社区）+ 与本机 DSH checkout（C:/AI/deepseek-harness）packages 结构交叉验证。

**局限与口径说明**：

- 星标、ARR、MCP 服务器数量、SDK 下载量等均为**检索日快照**，随时间变化；同一指标存在多个报道口径（如 dsh 星标：半小时 1 万/首日 28k/一夜 5 万/78.8k★），报告已并列标注来源与时间；
- 部分中文媒体标题存在修辞成分（如"一夜 5 万星""GitHub 史上增长最快"），引用时应以官方或可复核数据为准；
- dsh 处于 developer preview，官方明示会有兼容性破坏性变更，其仓库结构、包名与能力可能在数周内变化；
- Claude Code ARR、Cursor Composer 参数等商业/产品细节来自二手报道，未逐一与厂商财报核对；
- openJiuwen、AtomCode 等相对小众产品的信息密度依赖单一来源（博客园对比文），后续如纳入规划建议单独核验；
- 未覆盖：定价与商业模式细节、许可证合规细节、具体 benchmark 方法论——建议后续轮次专项调研。

---

## 附录 D：主要风险与不确定性（对 gaea 规划的提示）

1. **协议仍在快速演进**：MCP 授权扩展（2025-11）、ACP Registry（2026-01）上线不久，A2A 治理刚起步（2025-06 入 LF）；gaea 接入时应预留协议抽象层（client 接口可插拔），避免绑定某一协议的当前版本。
2. **安全是硬门槛**：MCP 生态有"安全债务"讨论（17 个 SEP、权限/授权短板）；computer use 类能力发布的同时已有"攻击者针对 AI agent"的警示——本机 agent 的沙箱、权限、审批、操作回滚要优先设计（对齐 Claude Cowork 的 RBAC + MCP 工具级权限、dsh 的 sandbox 插件）。
3. **dsh 生态尚处早期**：官方明示 breaking changes，直接复用其插件体系有兼容性风险；建议 gaea 只吸收其**模式与理念**（事件溯源会话、能力注册表、运行模式预设、子 agent 桥接），不绑定其运行时与包名。
4. **形态代差风险**：若 gaea 仍停留在"聊天框 + 单项工具"，与 2026 年的 WorkBuddy / TRAE Work / QoderWork / Claude Cowork / ChatGPT Work 存在形态代差；"工作台化"不是可选增强，而是主流产品共识方向。
5. **差异化依赖本地优先**：单用户本机 + 本地模型与云端巨头的"能力差"客观存在，需靠隐私、离线、低延迟、深度定制与"可自进化"弥补；dsh 的"运行时全开放"与 78.8k 星社区证明该路线有真实需求。

---

*报告完。数据检索时间：2026-08；所有数据为公开报道/官方文档/论文，星标、ARR、服务器数量等指标随时间变化，引用时请注明口径与日期。*
