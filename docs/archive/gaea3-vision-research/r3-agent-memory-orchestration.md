# gaea 3.0 愿景调研 r3：Agent 长期记忆与多 Agent 编排（公开信息综述）

> 调研范围：web_search 公开资料（中英文），约 30 次检索，覆盖 5 个主题。
> 产出定位：为 gaea 3.0 愿景的「P3 统一记忆与身份」「P4 治理下的自主性」两条支柱提供外部证据与工程参照。
> 结论颗粒度：具体到项目名/论文/产品/年份，每条关键结论附来源 URL。
> 说明：部分链接为第三方转载或聚合站，权威出处（arXiv / 官方博客 / GitHub 仓库）已优先给出。

---

## 目录

0. 摘要：最重要的 10 条结论
1. Agent 长期记忆架构最佳实践
   1.1 范式一：记忆即数据层（Memory as a Data Layer）
   1.2 范式二：程序化记忆与记忆→技能（Memory-to-Skill）
   1.3 Claude Code 的记忆文件约定（CLAUDE.md / SKILL.md / hooks）
   1.4 企业级记忆分层与生产教训
2. 多 Agent 编排模式
   2.1 总体谱系：工作流 vs 自主（Anthropic 的二分法）
   2.2 orchestrator-worker 模式
   2.3 subagent 委托（Claude Code subagents）
   2.4 handoff 交接（OpenAI Swarm / Agents SDK）
   2.5 workflow fan-out（LangGraph Send / map-reduce / 并行子代理）
3. 目标驱动 / 自主运行的 Agent 产品化现状
   3.1 从 AutoGPT/BabyAGI 到「受控自主」的教训
   3.2 产品化现状盘点（2024–2026）
   3.3 形态归纳：goal loop / cron+agent / 事件触发
4. 会话回放 / 分叉 / 时间旅行的工程实现
   4.1 消费端产品：ChatGPT branching、Claude 的 rewind 与 fork
   4.2 框架端：LangGraph time travel（checkpointer / get_state / update_state）
   4.3 基建与调试工具：record/replay/fork 生态
   4.4 事件日志事实源与可观测性
5. 本地优先 AI 的隐私卖点与「个人 AI 助理」信任架构
   5.1 local-first 运动与 AI 的交汇
   5.2 隐私卖点的真实产品案例
   5.3 信任架构：连续性 / 因果性 / 政策门控 / 记忆防投毒
6. 交叉结论：对 gaea 3.0 的启示（要点式）
7. 来源索引（按主题分组）

---

## 0. 摘要：最重要的 10 条结论

1. **记忆架构的行业共识已从「提示词注入」转向「记忆即数据层」**：Letta（MemGPT 演化）、Mem0、Zep 三家把记忆做成独立数据服务/库，agent 通过 API 读写，记忆与上下文解耦——这与 gaea 愿景中「统一记忆层」的方向一致（来源见 1.1）。
2. **记忆不是单一数据库，而是分层结构**：CoALA（2023，arXiv:2309.02427）给出 working / episodic / semantic / procedural 四分类，被后续所有记忆产品引用；生产实现普遍是「上下文内核心记忆 + 向量/图长期记忆 + 技能库」三段式。
3. **CLAUDE.md 已成为「程序化记忆」的事实标准**：Anthropic 官方最佳实践是分层记忆文件（全局/项目/子目录）+ 自动加载 + hooks 维护 + SKILL.md 技能——即「记忆文件是配置与代码，不是聊天记录」。
4. **多 Agent 编排的工程形态收敛为四类**：workflow（固定 DAG）、orchestrator-worker（规划者+执行者）、subagent 委托（Claude Code 形态：子代理 = 带模型/工具配置的 markdown 文件）、handoff（OpenAI Swarm/Agents SDK：会话控制权交接）。Anthropic 2024 年的结论是「最成功的 agent 往往是最简单的」（来源见 2.1）。
5. **fan-out 并行是深度研究类任务收益最大的模式**：Anthropic 多 agent 研究系统（2025）报告检索质量提升约 90%（准确率 65.8%→78.2%），并验证了并行 fan-out + 汇总层的价值；LangGraph 的 Send API 是这一模式的工业实现。
6. **自主运行的 agent 已产品化，但形态是「受控自主」**：Dust Triggers（2025）、ChatGPT 定时任务（2025-11）、Claude Code Desktop 计划任务（2025）、飞书 aily 任务模式（2025-12）、Google Gemini 2.5 Spark 后台运行（2026-05）均已上线——共同点是「睡眠期跑、醒来审」，全部带审批/通知收尾。
7. **会话分叉/时间旅行已是框架一等公民**：LangGraph 的 checkpointer + get_state/update_state 支持任意节点 fork；消费端 ChatGPT 2025-09 上线分支（branching），Claude Code 用户大量请求 rewind 与「分叉但不新建会话」（GitHub issue #9279 等）——证明「会话事实源 + 可回放」是用户能感知的强需求。
8. **回放/分叉已从功能演进为独立工具品类**：agent-replay、rewind-agent、tracefork、agent-vcr、aivcs、Agent-Git、salamander-db 等 2025-2026 涌现的「agent 版本控制 / 时间旅行调试」项目，共同思路 = 事件日志事实源 + 可持久化 checkpoint + 语义化 fork/merge。
9. **本地优先 AI 的隐私卖点已有真实产品背书**：Ente（加密相册公司）2025-2026 推出纯本地 LLM 应用 Ensu；Apple Intelligence 以「设备端 + Private Cloud Compute 可验证承诺」作为核心卖点（2024）；local-first 运动（Ink & Switch，2019 起）明确要求「网络只是可选优化，不是事实源」。
10. **「个人 AI 助理」的信任架构正在从「宣传承诺」走向「可验证工程」**：2025-10 的 Stateful Intelligence 论文提出连续性/因果性/尊严三支柱；SuperLocalMemory（2026，arXiv:2603.02240）把「记忆投毒防御」做成贝叶斯信任模型；产品侧出现 policy-gated agency（Jarvis）、正确性 by architecture（Aegis）等本地优先实现。

---

## 1. Agent 长期记忆架构最佳实践

### 1.1 范式一：记忆即数据层（Memory as a Data Layer）

三个代表性项目（全部开源，2023-2026 持续迭代）把「记忆」从上下文技巧升级为独立数据层：

**Letta（前身 MemGPT，2023-10 论文 / 2024 改名）**
- 核心设计：把操作系统分页思想搬进 LLM 上下文——**核心记忆块（core memory blocks）常驻上下文**（如 persona、human、user 等命名块），**档案记忆（archival memory）溢出到向量存储**，通过「自我编辑」（self-editing：agent 自己调用记忆工具增删改）在上下文中管理自己的记忆。
- 关键概念：记忆块（Memory Blocks）是「上下文管理的钥匙」，每个块有固定 token 预算；agent 通过 `/memory` 工具读写，不依赖 LLM 每次重新"记住"。
- 2025-2026 引入 sleep-time compute（睡眠期后台整理记忆）、记忆模块化。社区对它的评价是「最完整的 agent 记忆实现之一」。
- 来源：
  - 记忆块官方博客：https://www.letta.com/blog/memory-blocks
  - 记忆系统 DeepWiki：https://deepwiki.com/letta-ai/letta/3-memory-system
  - 社区问答「How does memory work in Letta」：https://forum.letta.com/t/how-does-memory-work-in-letta/93/2
  - 框架分析（含与 Hindsight 对比，2026）：https://vectorize.io/articles/hindsight-vs-letta

**Mem0（2024 创立，YC 背书，论文 arXiv:2504.19413，2025-04）**
- 核心设计：**通用记忆层（universal memory layer）**，与具体模型/框架解耦；从对话中提取事实，写入短期记忆，经「ADD / UPDATE / DELETE / NOOP」四类操作演化成长期记忆；支持向量检索 + 图关系。
- 论文要点：提出生产级记忆服务需要「提取、去重、更新、检索、失效」完整管线；对齐工作（alignment）决定提取质量；记忆操作必须显式、可审计。
- 平台架构：客户端 SDK（各类框架）+ 服务器端向量库/图库（Qdrant、Neo4j 等）+ REST API，多用户多 agent 隔离。
- 来源：
  - 论文（HuggingFace 页）：https://huggingface.co/papers/2504.19413
  - 论文（arXiv 镜像）：https://arxiv.org/abs/2504.19413
  - 官方架构文档：https://raw.githubusercontent.com/mem0ai/mem0/main/skills/mem0/references/architecture.md
  - 与 CortexDB 架构对比（2026）：https://cortexdb.ai/blog/cortexdb-vs-mem0

**Zep / Graphiti（2024 起）**
- 核心设计：**时序知识图谱（temporal knowledge graph）**。把 agent 记忆建成「实体-关系-事件」图，每条边带生效/失效时间（bi-temporal），支持「当时是什么状态」的查询（如"上个月用户在哪里工作"）；社区摘要（community summaries）在图上做滚动压缩。
- 论文：arXiv:2501.13956（2025-01）。
- 定位：企业级 agent 记忆，突出「可更新、可失效、可回溯」——这正是个人助理记忆（画像会变、事实会过期）需要的语义。
- 来源：
  - 论文页：https://arxiv.org/abs/2501.13956（另见 https://hub.baai.ac.cn/paper/043747d3-ea7a-4440-a6cc-acc45a220379）
  - 官方文档（Graphiti 概览）：https://help.getzep.com/graphiti/getting-started/overview
  - 与 Hindsight 对比（2026）：https://vectorize.io/articles/hindsight-vs-zep

**配套论文：CoALA（Cognitive Architectures for Language Agents，普林斯顿，2023，arXiv:2309.02427）**
- 提出 agent 记忆四分类：**工作记忆（working memory，即上下文）、情景记忆（episodic，历史经验）、语义记忆（semantic，事实与知识）、程序化记忆（procedural，技能与流程）**；并定义「记忆动作」（从记忆中读、向记忆中写、记忆内部操作）。几乎所有后续记忆产品（Letta/Mem0/Zep）都在用这套词汇表描述自己。
- 来源：
  - 论文页：https://arxiv.org/abs/2309.02427
  - 中文解读（BAAI hub）：https://hub.baai.ac.cn/view/30377
  - 相关项目清单（awesome-language-agents）：https://github.com/leehanchung/awesome-language-agents

**记忆分类的中文综述资料**
- 腾讯云《GraphRAG、Agentic RAG 与 AI 原生记忆系统的工程化实战》：https://cloud.tencent.com.cn/developer/article/2707864
- Atlan《Types of AI Agent Memory: Episodic, Semantic, Procedural and More》（含 CoALA 对照表）：https://atlan.com/know/types-of-ai-agent-memory/
- SSW Rules《Do you know the types of AI agent memory?》：https://www.ssw.com.au/rules/ai-agent-memory-systems
- arXiv 2602.06052《Rethinking Memory Mechanisms of Foundation Agents: A Survey》（2026 综述）：https://ar5iv.labs.arxiv.org/html/2602.06052

### 1.2 范式二：程序化记忆与记忆→技能（Memory-to-Skill）

- **程序化记忆（procedural memory）**：CoALA 定义的最晚被产品化的一类记忆——「agent 会怎么做」，对应 Claude Code 的 CLAUDE.md/SKILL.md、OpenAI 的 custom instructions、CrewAI 的 skills。
- **Memory-to-Skill 观点**（Zilliz 2025 中文专栏《重新理解 Agent 记忆：从 CoALA 到 Memory-to-Skill》）：重复出现的记忆应固化为技能/工具，而不是反复检索——「记忆的终点是技能」。这直接解释了为什么 CLAUDE.md 比向量记忆更受欢迎：它是最廉价的程序化记忆。
  - 来源：https://zilliz.com.cn/blog/have-we-got-agent-memory-all-wrong
- **「工作记忆折叠 / 会话档案化 / 记忆演化」**（BAAI hub 中文长文，2025）：提出会话结束后应把工作记忆「折叠」进长期记忆（归档、摘要、演化），与 DSH/gaea 的「会话 → 事实源 → 记忆」链路高度同构。
  - 来源：https://hub.baai.ac.cn/view/52036
- 阿里云开发者社区《AI Agent 长期记忆机制设计：从短期上下文到持久知识存储》：https://developer.aliyun.com/article/1752360

### 1.3 Claude Code 的记忆文件约定（CLAUDE.md / SKILL.md / hooks）

Claude Code 把「记忆文件」做成了事实标准，其约定对任何桌面 agent（含 gaea）都可直接借鉴：

- **CLAUDE.md 分层**：全局（`~/.claude/CLAUDE.md`）→ 项目根（`./CLAUDE.md`）→ 子目录（如 `./src/CLAUDE.md`），按工作目录自动加载最近层级的文件，可 `#` 引入其他文件（import 语法）。内容约定：项目背景、编码规范、常用命令、用户偏好、易错点。
  - Anthropic 官方博客《Using CLAUDE.md files》：
    - https://claude.com/blog/using-claude-md-files
  - 官方帮助中心《Give Claude context: CLAUDE.md and better prompts》：https://support.claude.com/en/articles/14553240
- **Steering 四件套**（Anthropic 2025-09 官方博客）：CLAUDE.md（长期偏好）+ skills（程序化技能，SKILL.md 带 frontmatter 声明触发场景）+ hooks（在事件点自动注入上下文/执行脚本，例如"文件变化后自动更新记忆文件"）+ subagents（见 2.3）。这是「记忆 = 配置 + 代码 + 自动化维护」的完整回答。
  - https://claude.com/blog/steering-claude-code-skills-hooks-rules-subagents-and-more
- **记忆维护自动化**：官方推荐用 hooks/脚本把「跑完任务后更新 CLAUDE.md」变成自动行为（例如自动追加"本项目的坑"）；社区出现大量「auto memory」工具，让 CLAUDE.md 成为活文档。
- 中文资料：
  - 腾讯云《从 0 到 1：用 CLAUDE.md 搭建永远懂你的项目环境》：https://cloud.tencent.cn/developer/article/2596824
  - claude-code-handbook《Memory System》章节：https://github.com/vitoworleone/claude-code-handbook/blob/main/docs/manual/part-04-context/ch-10-memory-system.md

**对 gaea 的直接启示**：轻语「三脑」命名空间 + CLAUDE.md 式分层文件可以融合——「人格档案（memory block）+ 项目记忆（分层 CLAUDE.md）+ 技能（SKILL.md）」，而不是再堆第 9 个记忆库。

### 1.4 企业级记忆分层与生产教训

- **分层是共识**：生产系统普遍分「上下文内（core/working）+ 检索外存（向量/图/文档）+ 技能库（procedural）」三层；短期记忆（会话内）与长期记忆（跨会话）分离，长期记忆再分事实（semantic）与经历（episodic）。
- **记忆要可失效、可更新、可审计**：Zep 论文的核心论点；Mem0 的 UPDATE/DELETE 操作；企业合规要求「用户可删除记忆」（GDPR 类要求）。
- **记忆与上下文管理是两件事**（Atlan 2026 文章）：context management 管「这次对话放什么进窗口」，memory management 管「跨会话持久化什么」——gaea 的「会话」与「记忆库」分离正是这个分工。
  - https://atlan.com/know/ai-agent-context/context-management-vs-memory-management-ai-agents/
- **agentic-memory 参考库**（社区维护的 agent 记忆论文/项目索引，含 Mem0/Zep 逐篇分析）：https://github.com/lhl/agentic-memory

---

## 2. 多 Agent 编排模式

### 2.1 总体谱系：工作流 vs 自主（Anthropic 的二分法）

- Anthropic 2024-12《Building Effective Agents》给出业界最常引用的分类：**workflows**（LLM 与工具在预置代码路径中协作）vs **agents**（LLM 动态决定自己的流程与工具调用）；并明确「最成功的 agent 往往是最简单的」——先用最少 agent 解决，必要时再上复杂编排。
  - Simon Willison 的解读：https://simonwillison.net/2024/Dec/20/building-effective-agents/
  - 中文综述（腾讯云）：https://cloud.tencent.com.cn/developer/article/2701818
- 多 agent 模式可归纳为四类工程形态（LangChain/Anthropic 官方文档共识）：workflow（DAG）、orchestrator-worker、subagent 委托、handoff 交接。

### 2.2 orchestrator-worker 模式

- **定义**：orchestrator（规划者）把任务分解为子任务，分发给 worker 并行执行，再汇总结果；子任务可再递归。适合「任务边界明确、需要并行」的场景。
- **Anthropic 官方 cookbook**（patterns-agents-orchestrator-workers，2025）给出可运行示例（代码审查、研究等）。
  - https://platform.claude.com/cookbook/patterns-agents-orchestrator-workers
- **生产案例：Anthropic 多 agent 研究系统**（2025-02 博客，报告复杂信息任务）：单 agent 准确率 65.8%；orchestrator + 并行 fan-out 研究子代理 + 汇总后达到 78.2%；检索质量提升约 90%；成本约增加 15 倍——「贵但准」是深度研究模式的明确权衡。
  - 全文收录（ZenML LLMOps 数据库）：https://www.zenml.io/llmops-database/building-production-multi-agent-research-systems-with-claude
  - 中文解读（90% 效率提升）：https://cloud.tencent.cn/developer/article/2532752
- **LangGraph 官方实现**：supervisor 模式（一个 supervisor LLM 路由给多个 worker）是文档主打模式之一。
  - 官方多 agent 概念文档：https://langchain-ai.github.io/langgraph/concepts/multi_agent/（中文：https://langgraph.com.cn/agents/multi-agent.1.html）
  - 社区模式总结（mastering-langgraph-agent-skill）：https://github.com/SpillwaveSolutions/mastering-langgraph-agent-skill/blob/main/references/multi-agent-patterns.md

### 2.3 subagent 委托（Claude Code subagents）

- **形态**：Claude Code 的子代理是 `.claude/agents/<name>.md` 文件——声明式定义：名称、描述（何时被主代理召唤）、模型、工具白名单、系统提示词。主代理（orchestrator）按需 spawn 子代理，子代理返回结果文本，不共享主上下文（上下文隔离）。
- **官方建议**（Anthropic 博客《How and when to use subagents in Claude Code》，2025）：把「独立任务、可并行、上下文需求不同」的工作交给 subagent；收益是上下文预算隔离 + 并行度；代价是子代理看不到主对话，需要把任务背景写清楚。
  - https://claude.com/blog/subagents-in-claude-code
- **生态实践**：社区普遍使用「子代理再生子代理」（spawn-of-spawn）把任务树展开；2025 年中出现「Swarms 模式」讨论（claude code 隐藏多代理模式），说明用户对多 subagent 并行有强需求。
  - https://www.xda-developers.com/set-up-claude-way-anthropic-now-recommends-sub-agents/
  - https://korben.info/en/claude-code-enable-hidden-swarms-mode.html
  - 2026 实践指南（Tembo）：https://www.tembo.io/blog/claude-code-subagents
  - 配置指南（Builder.io）：https://site.builder.io/blog/claude-code-subagents
- **关键工程点**：子代理 = 声明文件（非代码），这给了 gaea 一个现成的「技能/子代理即配置」产品形态——与愿景 P2 板块声明式一致。

### 2.4 handoff 交接（OpenAI Swarm / Agents SDK）

- **Swarm**（OpenAI Solutions 团队，2024-10 开源，教育性质）：两个核心抽象——**routine**（含指令与工具的函数）与 **handoff**（agent 之间显式移交会话控制权的特殊工具调用）；agent 是 stateless 的，交接即「把会话状态转交给另一个 agent」。
  - 仓库：https://github.com/openai/swarm
  - 官方 cookbook《Orchestrating Agents: Routines and Handoffs》：https://developers.openai.com/cookbook/examples/orchestrating_agents
- **OpenAI Agents SDK**（2025-03 正式发布，Swarm 的生产级后继）：agent + handoffs + guardrails + sessions（会话管理）；强调「最小原语」（loop 与 handoff 两个核心概念）。
  - 仓库：https://github.com/openai/openai-agents-python
  - 解读（2026）：https://futureagi.com/blog/what-is-openai-agents-sdk-2026/
- **handoff 与 subagent 的区别**（工程常识，多份资料印证）：handoff 是「接棒」（控制权转移，前 agent 退出），subagent 是「委派」（父 agent 保持控制、回收结果）。LangGraph 两者都支持（Send 派发 vs 命令式路由）。
  - LangGraph handoff 示例：https://github.com/kennethleungty/Handoffs-in-LangGraph-Multi-Agent-Systems

### 2.5 workflow fan-out（LangGraph Send / map-reduce / 并行子代理）

- **Send API**：LangGraph 的核心 fan-out 原语——从一个节点向多个「动态生成的」节点并发派发状态，配合 map-reduce 做并行处理与归并（官方文档 + 社区示例）。
  - 官方并行化概念：https://langchain-ai.github.io/langgraph/concepts/parallelization/
  - map-reduce 示例：https://github.com/martimfasantos/ai-agents-frameworks/blob/main/langgraph/16_map_reduce.py
- **工程价值**：深度研究、批量审查、多文件改造等「同一任务拆 N 份」场景；注意点——fan-out 前要把任务切分清楚（切分质量决定并行收益），fan-in 后要做冲突归并。
- 中文教程《LangGraph 多智能体系统完整教程》：https://latenode.com/blog/ai-frameworks-technical-infrastructure/langgraph-multi-agent-orchestration/langgraph-multi-agent-systems-complete-tutorial-examples
- 生产级 starter kit（7 种模式 + MCP 集成）：https://github.com/ac12644/langgraph-starter-kit
- 并行子代理参考实现（Tensorlake）：https://docs.tensorlake.ai/applications/parallel-sub-agents

---

## 3. 目标驱动 / 自主运行的 Agent 产品化现状

### 3.1 从 AutoGPT/BabyAGI 到「受控自主」的教训

- AutoGPT/BabyAGI（2023-03 爆火）证明了「目标循环（goal loop：目标→计划→执行→反馈→再计划）」的吸引力，也暴露了无边界自主的问题：死循环、成本失控、不可控。
  - 澎湃《AutoGPT 和 BabyAGI 是 AGI 的更进一步吗》：https://www.thepaper.cn/newsDetail_forward_22747517
  - 中文对比分析（AutoGPT vs BabyAGI）：https://cloud.tencent.com.cn/developer/article/2540699
- 2024-2026 的行业共识：**自主必须是受控的**——目标循环保留，但加上预算上限、审批闸门、人工确认点、审计日志。这正是 gaea 办公板块已有的「审批/沙箱/审计三件套」所对应的外部趋势。

### 3.2 产品化现状盘点（2024–2026）

按形态分四类，均有真实产品/公司/时间点：

**A. 定时/触发式任务（cron + agent）——主流形态**
- **Dust Triggers**（2025）：Dust 平台上线「Triggers」，让 agent「在你睡觉时工作」——定时或事件触发运行 agent 流程，结果留在工作区供次日审阅。https://front-edge.dust.tt/blog/introducing-triggers-your-agents-working-while-you-sleep
- **ChatGPT 定时任务**（2025-11 起）：ChatGPT 加入 scheduled tasks，「学会在你睡觉时工作」。https://aidash.news/2025/11/12/ai-dash-76-chatgpt-just-learned-to-work-while-you-sleep/
- **Claude Code Desktop 计划任务**（2025-12）：官方支持 schedule recurring tasks（cron 式），任务在本地运行，完成通知。https://code.claude.com/docs/en/desktop-scheduled-tasks（中文：https://code.claude.com/docs/zh-CN/desktop-scheduled-tasks）
- **飞书 aily「任务模式」**（2025-12，新华社报道）：从「回答问题」走向「执行工作」，支持任务编排与定时执行——国内办公场景的 agent 定时任务落地。http://www.news.cn/tech/20251216/dd271d59b3ef4ae49f4a84ace61f5832/c.html
- **Google Gemini 2.5 Spark**（2026-05 报道）：支持「proactive/后台运行 + MCP + 关机后继续」，派对策划等长任务在你合上电脑后继续跑。https://www.businessinsider.com/google-ai-agent-spark-proactive-run-background-mcp-gemini-2026-5
- **Glean「Digital Agents」**（2025-02）：企业搜索公司让客户构建「在你睡觉时工作」的数字化员工。https://www.businessinsider.com/ai-search-company-glean-launches-digital-agents-for-businesses-2025-2

**B. 开源「夜间任务」工具**
- **nightcrawler**（2025）：Claude Code 的隔夜自主循环——研究工具包、arXiv/Semantic Scholar 论文监控、知识合成、结构化交接（episodic execution + handoff）。https://github.com/thebasedcapital/nightcrawler
- **hermes-agent**（2024-2025）：「The agent that grows with you」的长期运行个人 agent（自述受 Nous Research 影响，中文 README 有）。https://github.com/Adkid-Zephyr/hermes-agent
- **kage-ai**（2025）：per-project 的 AI 原生 cron 任务执行器。https://pypi.org/project/kage-ai/0.2.6/
- **claude-code-scheduler**：把 Claude 变「自动驾驶」的 cron 调度器。https://github.com/jshchnz/claude-code-scheduler
- **automagik-spark**：调度 workflow、7x24 自动化，对接 LangFlow。https://github.com/namastexlabs/automagik-spark
- 工程文章《Nightly Tech-Debt Burners: Scheduling Agents to Clean Your Repo》（2025）：用定时 agent 清理技术债的实践。https://www.kinde.com/learn/ai-for-software-engineering/ai-devops/nightly-tech-debt-burners-scheduling-agents-to-clean-your-repo/

**C. 会话内自主（agent 自己决定跑多久）**
- Claude Code 的「headless 模式」与 /loop、连续运行（loop until done）成为高阶用法；社区自建 session manager（token 预算监控、长计划管理）。https://github.com/StanislavBG/claude-code-session-manager
- Anthropic 官方 issue #4785「Proactive, Scheduled Hooks for Automation (Cron-like)」（2025）：说明官方在向「主动型定时自动化」演进，目前靠外部调度。https://github.com/anthropics/claude-code/issues/4785

**D. 企业「数字员工」**
- 见 Glean（B 类已列）；另有大量 RPA+LLM 融合产品（本报告不展开，属办公自动化赛道）。

### 3.3 形态归纳：goal loop / cron+agent / 事件触发

| 形态 | 触发 | 退出条件 | 代表产品 |
|---|---|---|---|
| goal loop（会话内） | 用户给目标 | 目标达成/预算耗尽/用户打断 | Claude Code headless、DeepResearch 类 |
| cron+agent（定时） | 时钟 | 单次运行结束 + 结果通知 | ChatGPT 定时任务、Claude Code Desktop、Dust Triggers |
| 事件触发（事件） | 文件/消息/系统事件 | 任务完成 | Dust Triggers、Gemini 2.5 Spark |
| 常驻数字员工（长期） | 持续 | 无（按 SLA 运行） | Glean Digital Agents、hermes-agent |

共性：**「睡眠期跑、醒来审」——运行是无人值守的，但收尾必须有人**（通知、审批、报告）。对 gaea：夜间任务 = 目标循环 + cron 调度 + 结果摘要进会话 + 审批闸门，三件套全部已在办公板块有雏形。

---

## 4. 会话回放 / 分叉 / 时间旅行的工程实现

### 4.1 消费端产品：ChatGPT branching、Claude 的 rewind 与 fork

- **ChatGPT「分支」功能**（2025-09 上线）：用户可从任意消息分叉出新线程，「一个会话做多件事」；Ars Technica 评论指出这挑战了「对话是线性记录」的旧心智模型。
  - https://arstechnica.com/ai/2025/09/chatgpts-new-branching-feature-is-a-good-reminder-that-ai-chatbots-arent-people/
  - 36氪报道（中文）：https://www.36kr.com/p/3453602336593541
- **Claude 应用**：桌面/移动端支持「rewind（回退重发）」；Claude Code 用户对「rewind 而不创建新 fork」「从特定消息 split & rollback」有强烈诉求（GitHub issue 长期高赞）——说明**分叉的粒度与「不丢上下文」是核心痛点**。
  - Issue #9279（rewind 不创建 fork）：https://github.com/anthropics/claude-code/issues/9279
  - Issue #64993（split & rollback）：https://github.com/anthropics/claude-code/issues/64993
  - Issue #16236（conversation branching）：https://github.com/anthropics/claude-code/issues/16236
  - 会话管理官方博客（session management / 1M context）：https://claude.com/blog/using-claude-code-session-management-and-1m-context

### 4.2 框架端：LangGraph time travel（checkpointer / get_state / update_state）

- **实现机制**：每个节点运行后把状态写入 checkpointer（内存/SQLite/Postgres 等），得到「状态时间线」；`get_state` 读取历史状态，`update_state` 修改历史状态后继续运行 = 分叉（fork）；配合双写检查点支持「回放与分支」。
- **工程含义**：时间旅行是「持久化 checkpoint + 状态可序列化 + 可再入执行」三件事的组合——对任何 agent 框架都成立。
- 来源：
  - 官方 checkpointer 文档：https://docs.langchain.com/oss/python/langgraph/checkpointers（另见 https://langchain-ai.github.io/langgraph/concepts/persistence/）
  - 中文教程《LangGraph 新手村：时间旅行——浏览历史、分叉时间线与修改过去》：https://developer.aliyun.com/article/1732982
  - 代码示例（time_travel.py）：https://github.com/martimfasantos/ai-agents-frameworks/blob/main/langgraph/13_time_travel.py
  - LangSmith 的 time travel（HITL 恢复）：https://docs.langchain.com/langsmith/human-in-the-loop-time-travel

### 4.3 基建与调试工具：record/replay/fork 生态（2025-2026 密集涌现）

这是 2025-2026 一个明显的工具品类爆发，全部围绕「agent 运行的版本控制与时间旅行」：

- **agent-replay**（2025）：100% 本地、SQLite 驱动的 agent 时间旅行调试 CLI——重放执行轨迹、对比行为 diff、fork 运行测试修复、AI 评估与安全护栏。https://github.com/clay-good/agent-replay
- **rewind-agent**（2025-2026）：「AI agent 的 Chrome DevTools」——记录、检查、fork、重放、diff。https://pypi.org/project/rewind-agent/0.17.0/
- **tracefork**：位级精确 record/replay、任意步骤 fork、带置信区间的因果 blame。https://pypi.org/project/tracefork/
- **agent-vcr**：重放、编辑、断点续跑而不重跑。https://github.com/ixchio/agent-vcr
- **aivcs**（2025）：「AI Agent Version Control System」——状态提交、分支、语义化 merge。https://github.com/stevedores-org/aivcs
- **Agent-Git**：面向 LangGraph 生态的「agent 版本控制 + 开放分支 + RL MDP」基础设施层。https://github.com/MAS-Infra-Layer/Agent-Git
- **ContextTimeMachine**：跨会话快照/恢复 LLM 上下文，时间旅行、分支对话、checkpoint 回放（不重跑推理）。https://github.com/dakshjain-1616/ContextTimeMachine
- **salamander-db**（2025）：Rust 嵌入式事件溯源引擎，内置 time-travel 与 forking，定位就是「AI agent memory」数据库。https://github.com/rdelprete/salamander-db
- **smithers**：agent 平台/长程运行基础设施（会话持久化与恢复相关）。https://github.com/smithersai/smithers

**共性架构**：事件日志（event log）作为事实源 + checkpoint 快照 + 分支语义 + 语义化合并。这与 DSH 的「会话日志事实源」、愿景 P1 的「事件日志事实源 → 会话可回放/分叉/恢复」完全同构，且外部生态已经验证了用户价值。

### 4.4 事件日志事实源与可观测性

- **可观测性三件套**：LangSmith（LangChain 官方）、Langfuse（开源自托管）、Laminar（轻量）——统一做 trace、session 分组、回放（含时间旅行）、评估。2026 对比文：https://laminar.sh/blog/2026-01-29-laminar-vs-langfuse-vs-langsmith-llm-observability-compared
- 多 agent 调试工具盘点（2026，含 time-travel debug 定位）：https://futureagi.com/blog/best-multi-agent-debugging-tools-2026/
- MLflow 的 LLM tracing 对比：https://mlflow.org/articles/best-llm-tracing-tools-for-multi-agent-systems-in-2026/
- 要点：**日志结构（session/turn/tool_call 的树形 trace）决定回放/审计能力**；事件溯源（event sourcing）是「会话即事实源」的工程底座。

---

## 5. 本地优先 AI 的隐私卖点与「个人 AI 助理」信任架构

### 5.1 local-first 运动与 AI 的交汇

- **Ink & Switch《Local-first software》**（2019 起持续）：七条原则——你的数据归你（ownership）、多设备（multi-device）、离线可用、网络是可选优化而非事实源……2024 起该实验室把方向转向「tool for thought + AI」。
  - https://www.inkandswitch.com/
  - Wikipedia：https://en.wikipedia.org/wiki/Local-first_software
  - 中文语境解读（《What Is Local-First Software? And Why It Matters for AI Tools》）：https://www.llmnesia.com/blog/what-is-local-first-software
- **Local-First AI 生态盘点**（2026，Moryflow《Local-First AI Tools in 2026: The Definitive Landscape》）：本地模型运行器（Ollama/LM Studio）、本地知识库、本地 agent 记忆等已成独立品类。https://moryflow.com/blog/local-first-ai-tools

### 5.2 隐私卖点的真实产品案例

- **Ente Ensu**（2025-2026）：以端到端加密相册闻名的 Ente 推出纯本地 LLM 聊天应用 Ensu——「AI 在你设备上运行，数据不出设备」，把隐私公司信誉直接转嫁给 AI 产品。
  - 官方博客：https://ente.com/blog/ensu/
  - FAQ（数据/模型/删除策略）：https://ente.photos/help/ensu/faq/
  - App Store：https://apps.apple.com/us/app/ensu-entes-local-llm/id6758197006
- **Apple Intelligence 与 Private Cloud Compute**（2024-06 发布）：核心卖点 = 尽可能设备端处理 + 云端部分用 PCC（可验证的「Apple 也读不到」承诺，含公开审计）；The Verge 评论「Apple 的 AI 成败系于隐私承诺」。
  - https://www.theverge.com/ai-artificial-intelligence/946705/apple-private-cloud-compute-ai-siri-intelligence-wwdc
  - 9to5Mac 详解（2024-10）：https://9to5mac.com/2024/10/11/apple-intelligence-privacy-features-heres-what-you-should-know/
  - Apple 官方隐私页：https://www.apple.com/au/privacy/features/
- **开源本地优先个人助理**（2025-2026 密集涌现）：
  - **PrivacyCopilot**（Rust/Tauri 桌面）：加密存储、SQLite 历史、向量搜索、provider 可换。https://github.com/mihaibc/PrivacyCopilot
  - **nous-core**：家庭多 agent 系统，跑在树莓派 5 + Jetson Orin NX 上，「no cloud, no subscriptions, your data stays home」。https://github.com/Discod73/nous-core
  - **Fomi**：完全本地的 OS 交互插件式助理，「built to respect your privacy」。https://github.com/Rourugin/Fomi-AI-assistant
  - **Jarvis**：Windows/iPhone 本地优先个人智能，persistent memory + policy-gated agency。https://github.com/YuvrajKashyap/jarvis
  - **synapse**：本地优先的「个人记忆操作系统」（SQLite，agent 长期记忆）。https://github.com/Danialsamadi/synapse
  - **Aegis**：本地优先、护栏化认知系统——「工具执行的正确性来自架构而非模型能力」，目标小模型（2B-13B）可跑。https://github.com/mwkloh/aegis
  - **AMD GAIA**（本地 agent 框架）的 agent UI 计划：https://amd-gaia.ai/docs/plans/agent-ui

### 5.3 信任架构：连续性 / 因果性 / 政策门控 / 记忆防投毒

2025-2026 出现了一批把「个人 AI 助理的信任」工程化的论文与设计（不是营销话术，是可实现机制）：

- **Stateful Intelligence**（2025-10，Zenodo）：提出「有状态的智能」——信任三支柱：**连续性**（continuity，记忆跨会话）、**因果性**（causality，agent 行为可归因到数据/指令）、**尊严**（dignity，用户知情与控制）。对个人助理产品的信任设计是直接蓝图。
  - https://zenodo.org/records/17438011
- **Domain-Calibrated Trust in Stateful AI Systems**（2025-11）：把上述框架落地为具体机制（context-aware HITL、dispositional scaffolding）。
  - https://zenodo.org/records/17604302
- **Dignity-First Artificial Intelligence**（SSRN，2025-2026）：隐私、伦理与人类能动性的尊严优先框架。
  - https://papers.ssrn.com/sol3/papers.cfm?abstract_id=6532179
- **SuperLocalMemory**（2026，arXiv:2603.02240）：本地优先多 agent 记忆 + **贝叶斯信任防御对抗记忆投毒**（memory poisoning）——指出「本地」本身不够，还要防御恶意记忆注入（如通过文档/邮件投毒 agent 记忆）。这是「本地优先 = 更安全」叙事的重要补充（本地防外部泄露，但防不了注入，需要信任机制）。
  - https://huggingface.co/papers/2603.02240
- **产品侧的机制落地**：
  - Jarvis 的 policy-gated agency（策略门控的自主执行）；
  - Aegis 的「正确性 by architecture」；
  - 本地优先的「open and auditable」设计哲学（IronClaw 文档：https://mintlify.wiki/logicminds/ironclaw/concepts/philosophy）。
- **营销 vs 工程的分界**：隐私卖点（数据不出设备）是获客钩子；信任架构（连续性/可审计/可删除/防投毒/审批门控）才是留存与合规的工程。对 gaea：办公板块的「审批/沙箱/审计」+ 本地优先 = 现成的信任架构骨架，缺的是把它升级为可验证（如事件日志可回放即「可审计」的实现）。

---

## 6. 交叉结论：对 gaea 3.0 的启示（要点式）

1. **记忆层不要做第 N 个库，要做数据层**：参照 Mem0/Letta 的「核心记忆块（上下文内）+ 长期存储（向量/图）+ 操作 API（增删改查/归档/失效）」三段式；gaea 的 8 个记忆库应收敛为「一个数据层 + 命名空间/视图」，三脑（main/left/right）作为命名空间而非物理库。
2. **会话档案化是记忆的地基**：外部共识（BAAI 长文、CoALA、Zep）与 DSH 范式一致——事件日志事实源 → 会话结束折叠为记忆 → 记忆可溯源。愿景 P1（事件日志）与 P3（统一记忆）必须串成一条链。
3. **CLAUDE.md/SKILL.md 是程序化记忆的事实标准**：gaea 应把「记忆文件分层（全局/板块/项目）+ hooks 自动维护 + skill 声明式」直接搬进来，比自造记忆格式成本更低、生态可兼容。
4. **多 agent 编排先做两类**：subagent 委托（上下文隔离 + 声明式子代理文件）与 orchestrator-worker（跨板块任务：写小说需绘梦封面、办公出图、记忆供素材——正是 Anthropic 研究系统的场景）；handoff 与 Send fan-out 留到有明确用例再上，遵循「最成功的 agent 最简」。
5. **自主性产品化的正确形态是「睡眠期跑、醒来审」**：定时/事件触发 + 目标循环 + 结果进会话 + 通知/审批收尾。gaea 的夜间任务建议从「轻语日记/记忆整理」「跨板块素材收集」两个低风险场景起步。
6. **会话分叉/回放直接抄成熟模式**：LangGraph 的 checkpointer + get_state/update_state 语义、ChatGPT branching 的 UX（从任意消息分叉）、Claude Code 的 rewind 诉求（分叉不丢上下文）。DSH 的事件日志 + 快照即实现底座。
7. **信任架构 = 隐私卖点 + 可验证工程**：本地优先（数据不出设备）+ 事件日志可审计 + 审批门控 + 记忆可删除 + 防投毒（记忆写入也要审批/校验）——后者（SuperLocalMemory 指出的投毒风险）在 gaea 的「知识库/记忆可被外部文档影响」场景下尤其要提前设计。
8. **市场窗口判断**：外部生态证明「本地优先 + 有状态 + agent 化」是 2025-2026 的活跃方向（Ente Ensu、nous-core、Claude Code Desktop 计划任务、飞书 aily 任务模式），但「个人全任务域本地 agent 工作台」仍是空白——与愿景论文的判断一致。

---

## 7. 来源索引（按主题分组）

### 主题 1：agent 长期记忆架构
- Letta 记忆块博客：https://www.letta.com/blog/memory-blocks
- Letta 记忆系统 DeepWiki：https://deepwiki.com/letta-ai/letta/3-memory-system
- Letta 社区问答：https://forum.letta.com/t/how-does-memory-work-in-letta/93/2
- Hindsight vs Letta：https://vectorize.io/articles/hindsight-vs-letta
- Mem0 论文（HF）：https://huggingface.co/papers/2504.19413
- Mem0 论文（arXiv）：https://arxiv.org/abs/2504.19413
- Mem0 架构文档：https://raw.githubusercontent.com/mem0ai/mem0/main/skills/mem0/references/architecture.md
- CortexDB vs Mem0：https://cortexdb.ai/blog/cortexdb-vs-mem0
- Zep 论文：https://arxiv.org/abs/2501.13956
- Zep Graphiti 文档：https://help.getzep.com/graphiti/getting-started/overview
- Hindsight vs Zep：https://vectorize.io/articles/hindsight-vs-zep
- CoALA 论文：https://arxiv.org/abs/2309.02427
- CoALA 中文解读：https://hub.baai.ac.cn/view/30377
- 记忆类型（Atlan）：https://atlan.com/know/types-of-ai-agent-memory/
- 记忆类型（SSW）：https://www.ssw.com.au/rules/ai-agent-memory-systems
- 记忆机制综述（arXiv 2602.06052）：https://ar5iv.labs.arxiv.org/html/2602.06052
- Memory-to-Skill（Zilliz 中文）：https://zilliz.com.cn/blog/have-we-got-agent-memory-all-wrong
- 工作记忆折叠/会话档案化（BAAI）：https://hub.baai.ac.cn/view/52036
- 阿里云长期记忆设计：https://developer.aliyun.com/article/1752360
- 腾讯云 GraphRAG/记忆系统：https://cloud.tencent.com.cn/developer/article/2707864
- agentic-memory 参考库：https://github.com/lhl/agentic-memory

### 主题 1.3：CLAUDE.md 约定
- Anthropic 官方 CLAUDE.md 博客：https://claude.com/blog/using-claude-md-files
- Claude 帮助中心：https://support.claude.com/en/articles/14553240
- Steering 四件套博客：https://claude.com/blog/steering-claude-code-skills-hooks-rules-subagents-and-more
- 腾讯云中文教程：https://cloud.tencent.cn/developer/article/2596824
- claude-code-handbook 记忆章节：https://github.com/vitoworleone/claude-code-handbook/blob/main/docs/manual/part-04-context/ch-10-memory-system.md

### 主题 2：多 agent 编排
- Building Effective Agents（Simon Willison 解读）：https://simonwillison.net/2024/Dec/20/building-effective-agents/
- Anthropic 年度总结中文：https://cloud.tencent.com.cn/developer/article/2701818
- orchestrator-worker cookbook：https://platform.claude.com/cookbook/patterns-agents-orchestrator-workers
- Anthropic 多 agent 研究系统（ZenML 收录）：https://www.zenml.io/llmops-database/building-production-multi-agent-research-systems-with-claude
- 中文解读（90% 提升）：https://cloud.tencent.cn/developer/article/2532752
- LangGraph multi-agent 概念：https://langchain-ai.github.io/langgraph/concepts/multi_agent/
- LangGraph 中文文档：https://langgraph.com.cn/agents/multi-agent.1.html
- LangGraph 模式总结：https://github.com/SpillwaveSolutions/mastering-langgraph-agent-skill/blob/main/references/multi-agent-patterns.md
- LangGraph starter kit：https://github.com/ac12644/langgraph-starter-kit
- Claude Code subagents 官方博客：https://claude.com/blog/subagents-in-claude-code
- subagents 实践（XDA）：https://www.xda-developers.com/set-up-claude-way-anthropic-now-recommends-sub-agents/
- Swarms 模式讨论：https://korben.info/en/claude-code-enable-hidden-swarms-mode.html
- Tembo subagents 指南：https://www.tembo.io/blog/claude-code-subagents
- OpenAI Swarm：https://github.com/openai/swarm
- Routines and Handoffs cookbook：https://developers.openai.com/cookbook/examples/orchestrating_agents
- OpenAI Agents SDK：https://github.com/openai/openai-agents-python
- LangGraph handoff 示例：https://github.com/kennethleungty/Handoffs-in-LangGraph-Multi-Agent-Systems
- LangGraph 并行化：https://langchain-ai.github.io/langgraph/concepts/parallelization/
- map-reduce 示例：https://github.com/martimfasantos/ai-agents-frameworks/blob/main/langgraph/16_map_reduce.py
- Tensorlake 并行子代理：https://docs.tensorlake.ai/applications/parallel-sub-agents

### 主题 3：目标驱动/自主 agent 产品化
- Dust Triggers：https://front-edge.dust.tt/blog/introducing-triggers-your-agents-working-while-you-sleep
- ChatGPT 定时任务：https://aidash.news/2025/11/12/ai-dash-76-chatgpt-just-learned-to-work-while-you-sleep/
- Claude Code Desktop 计划任务：https://code.claude.com/docs/en/desktop-scheduled-tasks
- 飞书 aily 任务模式（新华社）：http://www.news.cn/tech/20251216/dd271d59b3ef4ae49f4a84ace61f5832/c.html
- Gemini 2.5 Spark 后台运行：https://www.businessinsider.com/google-ai-agent-spark-proactive-run-background-mcp-gemini-2026-5
- Glean Digital Agents：https://www.businessinsider.com/ai-search-company-glean-launches-digital-agents-for-businesses-2025-2
- nightcrawler：https://github.com/thebasedcapital/nightcrawler
- hermes-agent：https://github.com/Adkid-Zephyr/hermes-agent
- kage-ai：https://pypi.org/project/kage-ai/0.2.6/
- claude-code-scheduler：https://github.com/jshchnz/claude-code-scheduler
- automagik-spark：https://github.com/namastexlabs/automagik-spark
- 夜间技术债清理：https://www.kinde.com/learn/ai-for-software-engineering/ai-devops/nightly-tech-debt-burners-scheduling-agents-to-clean-your-repo/
- Claude Code 定时自动化 issue：#4785 https://github.com/anthropics/claude-code/issues/4785
- AutoGPT/BabyAGI 中文分析：https://cloud.tencent.com.cn/developer/article/2540699
- 澎湃 AutoGPT 报道：https://www.thepaper.cn/newsDetail_forward_22747517

### 主题 4：会话回放/分叉/时间旅行
- ChatGPT branching（Ars）：https://arstechnica.com/ai/2025/09/chatgpts-new-branching-feature-is-a-good-reminder-that-ai-chatbots-arent-people/
- 36氪中文报道：https://www.36kr.com/p/3453602336593541
- Claude Code issue #9279：https://github.com/anthropics/claude-code/issues/9279
- Claude Code issue #64993：https://github.com/anthropics/claude-code/issues/64993
- Claude Code issue #16236：https://github.com/anthropics/claude-code/issues/16236
- Claude 会话管理博客：https://claude.com/blog/using-claude-code-session-management-and-1m-context
- LangGraph checkpointer：https://docs.langchain.com/oss/python/langgraph/checkpointers
- LangGraph persistence：https://langchain-ai.github.io/langgraph/concepts/persistence/
- 阿里云时间旅行中文教程：https://developer.aliyun.com/article/1732982
- LangSmith time travel：https://docs.langchain.com/langsmith/human-in-the-loop-time-travel
- agent-replay：https://github.com/clay-good/agent-replay
- rewind-agent：https://pypi.org/project/rewind-agent/0.17.0/
- tracefork：https://pypi.org/project/tracefork/
- agent-vcr：https://github.com/ixchio/agent-vcr
- aivcs：https://github.com/stevedores-org/aivcs
- Agent-Git：https://github.com/MAS-Infra-Layer/Agent-Git
- ContextTimeMachine：https://github.com/dakshjain-1616/ContextTimeMachine
- salamander-db：https://github.com/rdelprete/salamander-db
- smithers：https://github.com/smithersai/smithers
- Laminar vs Langfuse vs LangSmith：https://laminar.sh/blog/2026-01-29-laminar-vs-langfuse-vs-langsmith-llm-observability-compared
- 多 agent 调试工具盘点：https://futureagi.com/blog/best-multi-agent-debugging-tools-2026/

### 主题 5：本地优先 AI 与信任架构
- Ink & Switch：https://www.inkandswitch.com/
- Local-first software（Wikipedia）：https://en.wikipedia.org/wiki/Local-first_software
- Local-first 与 AI 工具：https://www.llmnesia.com/blog/what-is-local-first-software
- Local-First AI 生态（Moryflow）：https://moryflow.com/blog/local-first-ai-tools
- Ente Ensu 博客：https://ente.com/blog/ensu/
- Ensu FAQ：https://ente.photos/help/ensu/faq/
- Ensu App Store：https://apps.apple.com/us/app/ensu-entes-local-llm/id6758197006
- Apple PCC（The Verge）：https://www.theverge.com/ai-artificial-intelligence/946705/apple-private-cloud-compute-ai-siri-intelligence-wwdc
- Apple Intelligence 隐私（9to5Mac）：https://9to5mac.com/2024/10/11/apple-intelligence-privacy-features-heres-what-you-should-know/
- Stateful Intelligence：https://zenodo.org/records/17438011
- Domain-Calibrated Trust：https://zenodo.org/records/17604302
- Dignity-First AI：https://papers.ssrn.com/sol3/papers.cfm?abstract_id=6532179
- SuperLocalMemory 论文：https://huggingface.co/papers/2603.02240
- PrivacyCopilot：https://github.com/mihaibc/PrivacyCopilot
- nous-core：https://github.com/Discod73/nous-core
- Fomi：https://github.com/Rourugin/Fomi-AI-assistant
- Jarvis：https://github.com/YuvrajKashyap/jarvis
- synapse：https://github.com/Danialsamadi/synapse
- Aegis：https://github.com/mwkloh/aegis
- IronClaw 设计哲学：https://mintlify.wiki/logicminds/ironclaw/concepts/philosophy
- AMD GAIA：https://amd-gaia.ai/docs/plans/agent-ui

---

> 撰写说明：本报告为 r3（agent 记忆与多 agent 编排）；与 r1（本地市场）、r2（编码 agent/工作台）、r4（会话与数据产品）并行，供愿景合成使用。所有结论均基于公开可访问来源，未包含内部项目信息。
