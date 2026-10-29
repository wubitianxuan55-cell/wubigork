# gaea V5 · 开源 / GitHub 生态调研（个人 AI 工作台 · 事项操作系统 · 可信执行）

- 调研日期：2026-10-29
- 工作目录：`C:\AI\wubigrok`
- 目标对象：gaea（Windows 桌面个人 AI 工具，Wails + Go + React，本地优先，用户口径当前 v4.527）
  - 版本口径注记：`.gaea/AGENTS.md` 顶部速览记录的最新发布为 **v4.457.0（2026-10-07）**，与用户所述 v4.527 不一致，可能为 AGENTS.md 未回写；本报告不据此推断产品能力，仅登记该差异。
- 方法说明（重要）：本次网络环境**无法读取任何网页正文**。`web_fetch` 对 `github.com`、`raw.githubusercontent.com`、`www.qbitai.com`、`export.arxiv.org` 等一律返回 `resolves to a non-public IP address`（DNS/代理层被劫持或出口受限）。因此本报告的事实来源**仅为 `web_search` 返回的标题与摘要片段**。
  - 凡标注「未获取」者，均为拿不到精确数据；
  - 凡标注「（摘要推断）」者，为从标题/摘要语义推断，未经 README 原文核对；
  - **本报告不含任何编造的 star 数、功能清单或发布日期。**

---

## 一、结论摘要

1. **赛道已经分成三层，而不是一层**：①桌面客户端（Cherry Studio、ekko-studio、OpenHuman、kuse_cowork）②任务/工作舱运行时（PilotDeck 2.0、patchwork-os、muldro）③持久执行底座（Temporal、Restate、DBOS、Orkes）。gaea 目前在①，但 v5 的差异化必须落在②，因为①正在被开源免费化。
2. **2026 年的真正分水岭是「审批 + 回执」（approval + receipt）**。patchwork-os 把定位写成 "the decision layer for personal AI"：每笔风险动作必须停下等用户 yes，worker 靠历史战绩赚取自治度，每个决策留 receipt。muldro 用 world model + trust/approval 做同一件事。这与 gaea「可信执行」是同一命题，且对方已经把叙事占住了。
3. **PilotDeck 2.0 是 2026-10 最强的正面威胁**：清华 + 面壁智能联合开源，定位「面向任务制的 AI Agent 生产力平台，以 WorkSpace 为单位」，媒体标题称任务成本降 83%、记忆可查可改，且自我定位为 "Harness"。它以「工作舱」为单位组织事项，恰好是 gaea「事项操作系统」的同构竞争。
4. **持久化/可恢复执行已被基础设施化**：Temporal（约 22K stars）、Restate（journal + pydantic-ai 集成）、DBOS（Postgres 上的轻量 durable workflow）、Orkes（Durable Adaptive Graphs）。这意味着「断点续跑」不再是卖点，而是**及格线**；gaea 若自研执行日志，应直接借用事件溯源 + SQLite 的形态，而不是重造工作流引擎。
5. **技能（Skills）正在成为分发单位，而不是插件**：tiangolo/agentskills 给出规范（渐进式披露、文件夹式 skill），anthropics/skills 给参考实现，genoffice 甚至把「办公套件」直接打包成 agent skill + CLI 供 Claude Code / Codex / Cursor 调用；造价大师也以 `SKILL.md` 形式提供能力。**这是 gaea 最便宜的可借鉴点，也是最容易被绕过的一层**。
6. **办公/生成式 UI 已被开源吞掉一半**：awesome-llm-apps（约 117K stars）里有 `generative_ui_agents` 示例（chat 与 agent 共享状态的看板），genspark-ai/genoffice 提供 Docs/Sheets/Slides/PDF 全栈开源 + 内置 agent + CLI。gaea 的 office 能力若只是「能生成 docx」，已无壁垒。
7. **中国垂直场景几乎无人竞争，但也几乎无人验证**：造价大师（costing-master，GB/T 50500-2024、各省定额、广联达兼容）与小说类（novel-studio、书童 AI-Novel-Writer）都是**单点工具**，没有一个把它们装进「个人 AI OS + 事项图 + 可信执行」的壳里。这是 gaea V5 的空位。
8. **安全叙事出现裂缝，是 gaea 的机会**：OpenHuman 因「整合 100+ 应用、权限高度集中」被媒体质疑安全（BigGo 财经）；同期有文章断言 agent sandbox 层已商品化、护城河转移。谁能把「权限最小化 + 每步可审」做成默认体验，谁就拿到信任。

---

## 二、项目清单表

> Stars 列规则：有明确来源的写数值并注明出处类型；只有量级传闻的写「约」；查不到的写「未获取」。活跃度按 2026-10 前后是否有公开更新/媒体报道判断。

### 2.1 本地优先桌面 Agent / 个人 AI OS

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想（一句） | 可借鉴点 | 对 gaea 的威胁 / 差距 |
|---|---|---|---|---|---|---|
| **OpenHuman** | `github.com/tinyhumansai/openhuman`（另有 `senamakel/openhuman` 分支镜像） | **31K**（4 个月达成，腾讯云社区文章标题；未获取精确当日值） | 高：2026-10 仍在被媒体与 issue 讨论（issue #2422） | 桌面代理把「记忆」做成**人类可读、可编辑**的资产，而非向量黑箱；并整合 100+ 应用做统一入口 | 「可读记忆」的产品化表达——记忆是用户能打开看、能改的文件；这比 mem0 式黑箱更契合本地优先 | 差距最大的一项：gaea 的记忆若仍是内部 DB，用户无法审计。威胁：31K 星 4 个月的增速本身就是分发优势 |
| **kuse-ai/kuse_cowork** | `github.com/kuse-ai/kuse_cowork`（镜像 `leemgs/kuse_cowork`）；产品页 `kuse.ai/open-cowork` | 未获取（sourcepulse 收录，无星数） | 中高：2026 年持续以 "Open-Source Claude Cowork Alternative" 定位推广 | 把 Claude Cowork 的「虚拟电脑 + 交付物」模式开源化：agent 在隔离环境里操作文件/浏览器，产出可交付成果 | 「交付物中心」而非「对话中心」的任务收尾方式；沙箱内操作的可复现性 | 威胁：直接对标「桌面 cowork」，且品牌叙事清晰。差距：gaea 缺少「一次任务 = 一份可交付文件」的强约束 |
| **IvenKooLab/ekko-studio** | `github.com/IvenKooLab/ekko-studio`（组织镜像亦见 `EKKOLearnAI/ekko-studio`、`ekkolearnai/hermes-studio`） | 未获取 | 中：README 有中英双语，2026 年在更 | 本地优先 AI 工作台：多 agent 对话 + 编码 + 可视化工作流，桌面与 Web 双端（摘要称基于 Vue 3 + Koa 的多平台 AI 助手管理） | 同一份工作台同时出桌面与 Web；可视化工作流编排降低「事项」的表达门槛 | 差距：gaea 的编排仍偏隐性。威胁中等：技术栈不同（Vue/Koa vs React/Go），但功能面重叠 |
| **Cherry Studio** | `github.com/CherryHQ/cherry-studio` | **40K+**（theagenttimes 文章标题 "Crosses 40K Stars"） | 高：2026 年仍是桌面 agent hub 的代表 | 一体化 AI 生产力工作台：聊天 + 自主 Agent + 300+ 助手预设，多模型供应商聚合 | 助手/预设的市场化组织方式；多 provider 聚合的工程细节 | 威胁：40K 星 + 成熟桌面壳，是 gaea 在「桌面 AI 客户端」层的最大同类。差距：gaea 的助手生态与预设分发几乎为零 |
| **OpenClaw** | `github.com/openclaw/openclaw`（文档 `docs.openclaw.ai`；生态报告见 `Clawland-AI/picclaw`） | **约 300K**（腾讯云文章标题 "GitHub 史上最快 30 万星"，量级传闻，未获取精确值） | 极高：2026 年现象级项目，已有企业侧记忆集成（阿里云 PolarDB Agent Memory） | **赢的不是模型，是消息总线**：以 gateway/消息总线为核心，把多渠道（GitHub、Linear、Notion 等）接入同一 agent | 总线式架构让「渠道」成为插件而非核心；这是 gaea 接微信/飞书/钉钉时应抄的形态 | 威胁：生态规模碾压，且已被云厂商做企业记忆插件。差距：gaea 无插件总线，渠道耦合在应用内 |
| **Oolab-labs/patchwork-os** | `github.com/Oolab-labs/patchwork-os`；npm `patchwork-os` | 未获取（npm 有 0.2.0-beta / 1.2.0-beta 版本线） | 高：beta 迭代频繁 | **个人 AI 的决策层**：本地优先运行时，agent 跨 GitHub/Slack/Gmail 等 45+ 服务自动化，但**每个风险动作停下等 yes**，worker 从战绩中赚取自治度，每个决策留 receipt；自带 MCP bridge（177 个 IDE 工具给 Claude Code） | ①「风险分级 + 逐级授权」的自治度模型 ②receipt 作为一等公民 ③policy 属于用户、模型属于用户、机器属于用户的三段式叙事 | **最直接的范式威胁**：它已经把「可信执行」讲成了产品语言。gaea 若只做「确认弹窗」，在叙事上落后一代 |
| **shivadharmi/muldro** | `github.com/shivadharmi/muldro` | 未获取 | 中：2026 年新项目 | 面向创始人的个人 AI OS：多 agent 运行时感知 Gmail/Calendar/Slack/GitHub，维护 **world model**，做规划，并把每一次外部写操作挡在 trust + approval 之后；Python/FastAPI + Next.js + MCP + Postgres/Redis/Qdrant/Neo4j | ①world model 作为「事项图」的实体层 ②外部写操作单独设闸（读自由、写审批）③多存储分工（关系/缓存/向量/图） | 威胁：创始人场景 = 高频 SaaS 上下文，正是 gaea 想切的人群。差距：gaea 无 world model，事项之间无实体关系 |

### 2.2 Agent OS / 运行时

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想 | 可借鉴点 | 威胁 / 差距 |
|---|---|---|---|---|---|---|
| **OpenBMB PilotDeck（2.0）** | `github.com/OpenBMB/PilotDeck`；文档 `pilotdeck.openbmb.cn`；镜像 `atomgit.com/OpenBMB/PilotDeck` | 未获取（GitHub 热榜级，无数值来源） | **极高**：2026-10 发布 2.0，量子位/搜狐/人人都是产品经理/北软协均有报道 | 面向任务制的 agent 生产力平台，**以 WorkSpace（工作舱）为单位**组织任务；官方英文副标题强调 "redefining operational boundaries and memory evolution"；2.0 自我定位为 **Harness**，主打长难任务稳定交付 | ①「工作舱」= 事项的容器，比「会话」更接近操作系统语义 ②记忆演进（可查可改）作为 OS 级能力 ③把「长难任务交付率」当作核心指标而非 token 数 | **最强正面威胁**：清华 + 面壁的品牌、开源、以及「工作舱」概念与 gaea「事项操作系统」几乎同构。速度上它已到 2.0 |
| **清华 2026-10 智能体操作系统（与上条同一项目，已查清）** | 同 PilotDeck：`github.com/OpenBMB/PilotDeck` | 未获取 | 极高 | 媒体口径：「智能体操作系统，任务成本降 **83%**，AI 记忆**可查可改**」。可交叉验证的第三方标题为《清华联合面壁智能开源 PilotDeck 2.0，让智能体记忆可查可改》 | 成本指标的表述方式（降 83% 相对基线未说明，引用时须标注「媒体口径，基线未获取」） | 结论：**「清华 2026-10 智能体 OS」= PilotDeck 2.0**，不是独立项目。此前把它当作两个项目的判断应撤销 |
| **arXiv AgentOS** | `arxiv.org/abs/2603.08938` — *AgentOS: From Application Silos to a Natural Language-Driven Data Ecosystem* | 不适用（论文） | 2026-03 预印本，被多方引用 | 论证从「应用孤岛」走向「自然语言驱动的数据生态」，即 OS 层抽象应围绕数据/意图而非 app | 为「事项即一等公民」提供学术合法性；可用于 gaea 对外白皮书引用 | 差距：gaea 无理论叙事，容易被当成工具而非 OS |
| **字节 GUI Agent（UI-TARS）** | 字节跳动 UI-TARS 系列（桌面端 `UI-TARS-desktop`） | **26K**（c114 文章标题 "突破26k Star"） | 高：2026 年登 GitHub 热榜，被称为豆包手机核心技术 | 纯视觉 GUI agent：不依赖 DOM/API，直接看屏操作，跨应用泛化 | 视觉兜底通道——当 gaea 的目标应用无 API（如国产造价软件、旧版 Office 插件）时的最后手段 | 威胁：字节以手机/桌面双端压过来。差距：gaea 目前只有结构化工具调用，缺视觉兜底 |

### 2.3 持久化 / 可恢复执行

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想 | 可借鉴点 | 威胁 / 差距 |
|---|---|---|---|---|---|---|
| **Temporal** | `github.com/temporalio/temporal` | **约 22K**（east196 调研文章标题 "22K stars"） | 高：6 年老牌，2026 仍活跃 | 事件溯源式 workflow：确定性重放（replay）+ 历史事件日志，代码即状态机 | 确定性重放是「崩溃后精确恢复到同一步」的唯一可靠做法；gaea 的任务日志应 append-only | 差距：gaea 若用「快照 + 重试」，会在 LLM 非确定性调用上出错。不必引入 Temporal，但必须借其日志模型 |
| **Restate** | `restate.dev`（GitHub 组织 restatedev） | 未获取 | 高：2026 与 pydantic-ai 有官方集成文章 | 单二进制 durable execution：journal 记录每步副作用，SDK 内嵌到应用进程，无需独立集群 | 轻量形态更适合桌面端——**单进程 + journal 文件**，而非服务端集群 | 这是 gaea 最该抄的工程形态：把 journal 落到本地 SQLite，而不是跑一个 Temporal server |
| **DBOS** | `github.com/dbos-inc/dbos-transact-py`（PyPI `dbos`） | 未获取 | 高：有 Vercel AI SDK 集成文档 | 把 durable workflow 建在 **Postgres** 上，用数据库事务兼作执行日志，无额外编排服务 | 「数据库即编排器」——gaea 已有 SQLite，可直接把状态机表当 journal | 差距：gaea 缺「步骤幂等键」概念，重跑会重复扣费/重复写文件 |
| **pydantic-ai durable execution** | `pydantic.dev/docs/ai/capabilities/durable_execution/temporal/` | 不适用（框架能力） | 高：2026 有 Temporal 与 Restate 两条后端路径 | 把 agent 的 tool 调用声明为 durable step，框架负责重放与副作用隔离 | 以「类型化 agent + 可插拔持久后端」的方式，避免框架锁定 | gaea 的 Go 侧等价物是「tool 调用 = 可重放 step」，需在协议层定义 |
| **Orkes Durable Adaptive Graphs** | `orkes.io/content/ai-cookbook/dynamic-workflows` | 未获取（商业 + 开源 Conductor 系） | 高：2026 有 AI cookbook 专章 | 图在执行中**动态改写**：agent 可在运行时增删节点，而不是预编译 DAG | 事项图天然是动态图——用户的计划会变，图必须可编辑且可回滚 | 差距：gaea 的事项关系若为静态表，无法表达「计划变更」 |

### 2.4 工作/事项图与决策层

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想 | 可借鉴点 | 威胁 / 差距 |
|---|---|---|---|---|---|---|
| **@claxedo/workgraph** | `npmjs.com/package/@claxedo/workgraph` | 未获取（npm 包存在，README 未能读取，功能描述未获取） | 未获取 | （摘要推断）以 work graph 为单位的任务/依赖编排包 | 命名与抽象值得关注：「work graph」而非「task list」 | 无法评估威胁：需在有网环境读 npm README |
| **ranausmanai/agent-workgraph** | `github.com/ranausmanai/agent-workgraph` | 未获取 | 中：2026 新项目 | 把 AI 编码会话**沉淀为可复用的工程记忆、evals、briefs 与 skills** | 「一次会话 = 一份可复用资产」的转化管线；对 gaea 的复盘/沉淀功能是直接模板 | 差距：gaea 的任务结束后资产消散，无法复用 |
| **case management 范式** | 代表实现分散（法律/保险 agent 套件，如 `dsh-legal-suite`、`judicialmind/legal-agents`，均为 npm 包，未逐一验证） | 未获取 | 未获取 | 以「案件/事项」为主键，聚合证据、时间线、决策与产出物，而非以对话为主键 | 事项 = 主键 + 证据链 + 时间线；这正是「事项操作系统」的数据模型 | 差距：gaea 主键仍是会话/文件，事项不是一等实体 |
| **project-based agent workspace** | 同上，PilotDeck 的 WorkSpace 是最强代表 | — | — | 以项目/工作舱隔离上下文、权限、产物 | 上下文按事项隔离，避免跨任务污染 | 见 2.2 |

### 2.5 可信 / 可审计

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想 | 可借鉴点 | 威胁 / 差距 |
|---|---|---|---|---|---|---|
| **agentlens-audit** | PyPI `agentlens-audit`（0.3.0）；相关 `agentlens-io`（0.7.0） | 不适用（PyPI 包） | 未获取 | **未能读取包描述，功能未获取**（不可臆测） | 仅登记：Python 生态已出现专门的 agent 审计包，说明「审计」正在独立成层 | 无法评估。建议后续有网时补 README |
| **AgentOps** | `github.com/AgentOps-AI/agentops` | 未获取 | 高：2026 有 AG2 官方集成文档 | agent 监控与调试：会话追踪、事件记录、可回放（摘要推断："monitoring and debugging"） | 会话级 trace 的时间轴 UI，是「可信」最直观的表达 | 差距：gaea 无可视化 trace 时间轴，用户无法回看「AI 当时为什么这么做」 |
| **computer-use traceability** | 代表实现 `aaronlab/browsertrace`（记录浏览器状态与模型 I/O 以调试 computer-use agent） | 未获取 | 中 | 把「屏幕状态 + 模型输入输出」成对固化，使每一步可复现、可追责 | 视觉 agent 的黑箱问题用「状态快照对」解决 | 差距：gaea 若上视觉通道，缺这套取证设施 |
| **prompt injection / VM escape 后的开源防护** | 代表实现 `AlexanderMattTurner/agent-glovebox`（gVisor 沙箱相关提交）、`asbox`（agent-sandbox-os，PyPI） | 未获取 | 2026 年新项目密集出现 | 把 agent 关进 microVM/gVisor 级沙箱，用系统调用层隔离替代「提示词层防御」 | 结论：**提示词层防御已被证伪，隔离必须下沉到内核/VM 层** | 差距：Wails 桌面端要落地需评估 Windows 侧（AppContainer/Job Object/HCS）成本 |
| **产业判断** | 文章《Commoditized: The Agent Sandbox Layer Has Become a Utility — and the Moat Is Moving》 | — | 2026 | 沙箱层已商品化，护城河转向「策略/审计/信任」 | 佐证 gaea 不应自研沙箱内核，应买/借，把力气放在 policy 与 receipt | — |

### 2.6 技能与上下文

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想 | 可借鉴点 | 威胁 / 差距 |
|---|---|---|---|---|---|---|
| **Agent Skills 规范** | `github.com/tiangolo/agentskills`；规范站 `agentskills.io/specification` | 未获取 | 高：规范站有 client implementation 指南（"adding skills support"） | 技能 = 文件夹（`SKILL.md` + 资源），按需**渐进式披露**进上下文，而非一次性全量注入 | ①技能即目录，天然可版本化、可分发 ②渐进披露省 token ③跨客户端可移植 | **最便宜的可借鉴项**：gaea 应把内置能力改成 skill 包，同时获得生态兼容（Claude Code/Codex 可直接用） |
| **anthropics/skills** | `github.com/anthropics/skills` | 未获取 | 高：官方参考实现仓库 | 官方技能样例集合，示范 SKILL.md 的写法与边界 | 直接照抄目录结构与元数据字段 | 差距：gaea 无技能格式，能力硬编码 |
| **awesome-context-engineering** | `github.com/yzfly/awesome-context-engineering`；另有 `royalpinto007/awesome-context-engineering`（自称 auto-updating） | 未获取 | 高：两个活跃维护版本 | 上下文工程的知识地图：压缩、检索、结构化、上下文腐烂（context rot）治理 | 用于校准 gaea 的上下文预算策略与压缩阈值 | 差距：gaea 缺显式的上下文预算与压缩策略文档 |
| **mem0 / 上下文引擎** | `github.com/mem0ai/mem0` | 未获取 | 高：2026 仍是记忆层基准对照（多篇对比文，如 mnem vs mem0） | 自动抽取 + 更新用户记忆，向量 + 图混合检索 | 记忆的「抽取-更新-遗忘」三段式 | 与 OpenHuman 相反路线：mem0 是黑箱自动化，OpenHuman 是可读资产。**gaea 应选可读优先、自动化辅助** |

### 2.7 生成式 UI 与办公

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想 | 可借鉴点 | 威胁 / 差距 |
|---|---|---|---|---|---|---|
| **awesome-llm-apps / generative_ui_agents** | `github.com/Shubhamsaboo/awesome-llm-apps`（子目录 `generative_ui_agents`） | **约 117K**（阿里云文章标题 "11.7 万 Star"） | 极高 | 生成式 UI starter：**chat 与 agent 共享同一份状态**，UI 由 agent 的输出驱动渲染看板 | 共享状态是生成式 UI 的关键——不是「agent 返回 HTML」，而是「agent 改状态，UI 反映状态」 | 差距：gaea 若用「返回 markdown/组件 JSON」，会缺少双向状态绑定，交互无法就地完成 |
| **genspark-ai/genoffice** | `github.com/genspark-ai/genoffice`（另有镜像 `FrI3nd7/genoffice`） | 未获取 | 高：2026-09 有归档快照，官网 `genoffice.ai` 多语种 | 免费开源 AI 办公套件：Docs/Sheets/Slides/PDF/Markdown/HTML 编辑器 + 内置 agent + `genoffice` CLI + **agent skill**，让 Claude Code/Codex/Cursor 本地直接生成真实 .docx/.xlsx/.pptx；BYO key；三平台 | ①「GUI 套件 + CLI + skill」三件套，同一能力三种入口 ②CLI 输出一行摘要或 `--json`，天然可被 agent 调用 ③BYO key + 本地文件 | **直接威胁 gaea 的 office 模块**：它把「AI 生成真 Office 文件」做成了开源标准件，且可被任意 coding agent 调用，绕开 gaea |
| **kaos-office** | PyPI `kaos-office`（0.1.0 / 0.1.0a7） | 不适用（PyPI） | 未获取 | （摘要推断）以 agent 框架方式提供 office 文档处理能力的 Python 包 | 能力包化、pip 可装，便于嵌入 | 无法评估细节：未获取文档 |
| **genpark office skill** | `github.com/Alpha-Park/genpark-genoffice-document-agent-suite-skill` | 未获取 | 中 | GenPark 的 agent skill：多格式文档引擎 + agentic synthesizer，覆盖 docx/xlsx/pptx/pdf/md | skill 化封装的又一个样本：**办公能力正在变成 skill 而非 SDK** | 差距：gaea 的 office 能力不可被外部 agent 复用，反之亦然 |

### 2.8 垂直场景

| 项目 | Repo / URL | Stars | 最近活跃度 | 核心架构思想 | 可借鉴点 | 威胁 / 差距 |
|---|---|---|---|---|---|---|
| **jayden-lg/costing-master（造价大师）** | `github.com/jayden-lg/costing-master`（含 `SKILL.md`） | 未获取 | 中：2026 项目，README 完整 | 中国建设工程造价全流程 AI 助手：遵循 **GB/T 50500-2024**，覆盖投资估算→竣工决算，兼容各省定额与广联达；能力以 `SKILL.md` 形式给出 | ①规范号入 prompt/校验（合规可验证）②省级定额的差异化数据层 ③以 skill 而非 app 分发 | **与 gaea 场景高度重合**。威胁：单点工具但专业度高。差距：gaea 若做造价，必须补齐定额库与广联达互通 |
| **heavenrousy/neuro-estimator** | 未获取（检索仅命中无关的学术 NEUROeSTIMator，Iowa 研究项目，2023） | 未获取 | 未获取 | **未获取**——未能确认该 repo 存在或其用途 | — | 无法评估。建议有网时按 GitHub 用户页直查 |
| **EthanYoQ/AI-Novel-Writer（书童 Pagekin）** | `github.com/EthanYoQ/AI-Novel-Writer` | 未获取 | 高：2026 活跃，Windows & macOS | 开源免费 AI 长篇小说桌面软件：设定→蓝图→正文→审稿→定稿全链路；支持拆解仿写、续写、改写；可导入写作 Skill（如去 AI 味）；模型自配、作品本地 | ①「流水线阶段」而非「一次性生成」 ②可导入 skill 改变文风 ③本地存稿、无模型绑定 | 威胁：桌面 + 本地 + 全链路，与 gaea 小说场景正面撞。差距：gaea 缺「审稿/定稿」这类质量闸门 |
| **Xiaoyangy/novel-studio** | `github.com/Xiaoyangy/novel-studio` | 未获取 | 中高：2026 项目 | 本地优先长篇创作引擎：**多智能体世界推演 + 按弧（arc）规划 + RAG 长程记忆 + 逐章审核 + 断点恢复** | ①按弧规划 = 长任务的层级分解 ②逐章审核 = 质量闸门 ③断点恢复 = durable execution 的创作版 | 差距：这三件（弧规划/审核/恢复）正是 gaea 长任务缺的骨架 |

---

## 三、五条架构启示

### 启示 1：事项（WorkSpace / Work Graph）必须是持久化的一等实体，而不是会话的副产物

证据链：PilotDeck 以 WorkSpace 为组织单位；patchwork-os 的决策单元是「一笔风险动作」；muldro 用 world model 承载实体关系；agent-workgraph 把会话转成可复用资产；case management 以案件为主键。
对 gaea 的含义：v5 的数据模型应把 `Task/事项` 提到与 `Conversation` 同级甚至更高，事项下挂：目标、证据、产物、审批记录、执行日志、复用资产。会话只是事项的一种视图。

### 启示 2：持久执行 = append-only 日志 + 确定性重放，桌面端用 SQLite 落地即可

证据链：Temporal 的 replay 模型、Restate 的单二进制 journal、DBOS 的「数据库即编排器」、pydantic-ai 的可插拔 durable backend、Orkes 的动态图改写。
对 gaea 的含义：不要引入 Temporal server；在 SQLite 建三张表（`run` / `step` / `effect`），每个 tool 调用带幂等键，启动时重放未完成 step。动态图改写对应「事项图可编辑 + 变更留痕」。这同时满足「可恢复」与「可审计」两个目标。

### 启示 3：可信执行的最小完备集 = 风险分级 + 审批闸 + 回执（receipt）+ 沙箱

证据链：patchwork-os（每笔风险动作停下等 yes、worker 靠战绩赚自治、每决策留 receipt）、muldro（外部写操作单独设闸）、微软「approval 必须在模型之下」的观点、agent-glovebox/asbox（提示词防御已被证伪，隔离下沉到 VM）、sandbox 层商品化的产业判断。
对 gaea 的含义：四件事缺一不可。尤其「回执」——它是用户事后追责与信任累积的凭据，比审批弹窗更重要。审批粒度应按「副作用类型 × 可逆性」分级，而非统一 yes/no。

### 启示 4：技能是分发单位，办公能力应该「GUI + CLI + Skill」三入口

证据链：agentskills 规范（渐进披露）、anthropics/skills、genoffice（编辑器 + CLI + skill，且 CLI 支持 `--json` 供 agent 调用）、genpark office skill、造价大师的 `SKILL.md`。
对 gaea 的含义：v5 把内置能力重写为 skill 目录（`SKILL.md` + 资源 + 可选脚本），既省上下文又获得跨客户端可移植性；同时为每个能力提供 CLI 与 JSON 输出，使 gaea 自己既是 agent 宿主，也是别的 agent 的工具提供方——这是被 genoffice 绕开后的反制手段。

### 启示 5：生成式 UI 的关键是「共享状态」，不是「生成组件」

证据链：awesome-llm-apps 的 generative_ui_agents starter（chat 与 agent 共享状态）；DSH 自身以「白名单组件树」渲染而非自由 HTML。
对 gaea 的含义：v5 的 UI 层应定义一份**可被 agent 读写的视图状态**（如事项看板状态），agent 输出的是状态变更而非界面代码；渲染层负责把状态映射成组件。这样「审批」「勾选」「就地编辑」都在客户端完成，零往返，且天然可回放。

---

## 四、对 gaea V5 的威胁与空位

### 4.1 威胁（按紧迫度排序）

1. **PilotDeck 2.0（最高）**：同构概念（工作舱 ≈ 事项）、清华+面壁品牌、开源、已到 2.0、媒体口径有 83% 成本与「记忆可查可改」。若 gaea 的「事项操作系统」叙事晚于它落地，会被视为跟随者。
2. **patchwork-os / muldro（叙事威胁）**：它们已经把「个人 AI OS 面向 Gmail/Slack/GitHub 的创始人场景 + 审批 + receipt + world model」讲完整。这套叙事在英文开源社区传播效率远高于中文，会先占住「可信个人 AI」的心智。
3. **OpenClaw（~300K stars，生态威胁）**：消息总线 + 插件生态 + 云厂商企业记忆集成。它的规模会吸走「渠道接入」类需求。
4. **genoffice（功能替代）**：开源、三平台、CLI + skill，任何人可在 Claude Code/Codex 里直接生成真 Office 文件。gaea 的 office 模块不再是差异点。
5. **Cherry Studio（同层竞争）**：40K+，桌面 AI 客户端 + 300+ 助手，直接分流「我就想要个桌面 AI 工具」的用户。
6. **造价大师 / 小说工具（垂直替代）**：单点但专业度足够；用户可能「gaea 管事项 + 垂直工具干活」，gaea 沦为壳。

### 4.2 空位（gaea 可占）

1. **中文办公/造价的「事项 + 证据链」**：没有任何开源项目把 GB/T 50500-2024、各省定额、广联达互通装进「事项图 + 审批回执」的壳。造价天然是「多版本、多依据、需签字」的流程，receipt 在这里有真实法律/商务价值，而不是产品修辞。
2. **长篇小说作为「长任务」的持久执行样本**：novel-studio 有弧规划与断点恢复，AI-Novel-Writer 有全链路与 skill 导入，但两者都没有「审批闸 + 可审计」——而小说创作恰恰需要「作者批准关键转折」。把 durable execution + approval 用在创作上，是没人做的组合。
3. **Windows 原生 + 本地优先的可信执行**：主流实现要么是 Electron/Web（Cherry、ekko），要么是 Python/FastAPI + Next.js（muldro），要么是 Web 优先（patchwork）。**Wails + Go 的 Windows 原生形态 + AppContainer/Job Object 级隔离**，在国产办公软件（广联达、CAD、旧版 Office 插件）场景里是天然优势，且没人认真做。
4. **「可读记忆」的中文落地**：OpenHuman 证明了可读记忆的吸引力，但它是英文、SaaS 上下文。中文场景的记忆应该是「项目档案 / 甲方要求 / 定额依据 / 人物设定」这些可被用户直接打开审阅的文件夹——这比向量库更贴合中国用户对「资料」的心智。
5. **回执的中文化**：patchwork-os 的 receipt 是英文开发者语汇。在中国办公场景，它应该叫「操作凭证/存档单」，能打印、能盖章、能进竣工资料——这是把技术概念转成商业价值的机会。

### 4.3 场景差异（必须写进 V5 定位）

| 维度 | 海外创始人场景（patchwork-os、muldro、OpenHuman） | gaea 的中国办公/造价/小说场景 |
|---|---|---|
| 上下文来源 | Gmail / Slack / GitHub / Calendar（API 齐全、OAuth 标准） | 广联达、CAD、Excel 定额库、微信/钉钉、纸质/扫描件（API 缺失，需视觉与文件解析） |
| 动作风险 | 发邮件、改 issue、写日历（可撤销） | 改工程量、出结算书、发标书（**不可撤销、有法律后果**） |
| 可信的证明方式 | receipt 作为产品特性 | 凭证需可打印、可归档、可对外出示（竣工资料/审计） |
| 协作对象 | 个人 + SaaS | 个人 + 甲方/审计/施工方（多方对账） |
| 长任务形态 | 多步 SaaS 自动化 | 长周期项目（数月）+ 长篇创作（数十万字） |
| 记忆内容 | 联系人、偏好、历史对话 | 定额依据、规范条款、人物设定、甲方口径 |

结论：gaea V5 不应模仿「Gmail/Slack/GitHub 创始人 OS」，而应做「**中文重资产业务的个人 AI OS**」——事项重、证据重、审批重、周期长。这恰好是海外项目结构上不愿进入的市场。

---

## 五、参考链接

> 所有链接均为 `web_search` 返回的 URL；本次环境无法 `web_fetch` 正文（non-public IP），引用内容仅来自标题/摘要。

**本地优先 / 个人 AI OS**
- OpenHuman 31K stars 报道：https://cloud.tencent.com.cn/developer/article/2686137
- OpenHuman 仓库（tinyhumansai/openhuman，issue #2422 佐证）：https://github.com/tinyhumansai/openhuman/issues/2422
- OpenHuman 安全质疑（权限集中）：https://finance.biggo.com/news/izm6Op4BtCxy99G5gE2-
- kuse-ai/kuse_cowork：https://github.com/kuse-ai/kuse_cowork ｜产品页：https://www.kuse.ai/open-cowork/features
- ekko-studio：https://github.com/IvenKooLab/ekko-studio ｜中文 README：https://github.com/EKKOLearnAI/ekko-studio/blob/main/README_zh.md
- Cherry Studio 40K stars：https://theagenttimes.com/articles/cherry-studio-crosses-40k-stars-the-desktop-agent-hub-the-west-hasnt-noticed-yet
- Cherry Studio 仓库：https://github.com/CherryHQ/cherry-studio
- OpenClaw 30 万星与消息总线：https://cloud.tencent.com.cn/developer/article/2756312
- OpenClaw 文档：https://docs.openclaw.ai/plugins/github
- patchwork-os：https://github.com/Oolab-labs/patchwork-os ｜npm：https://www.npmjs.com/package/patchwork-os
- muldro：https://github.com/shivadharmi/muldro

**Agent OS / 运行时**
- PilotDeck 仓库：https://github.com/OpenBMB/PilotDeck ｜中文 README：https://github.com/OpenBMB/PilotDeck/blob/main/README.zh.md
- PilotDeck 产品文档（WorkSpace 工作舱）：https://pilotdeck.openbmb.cn/PilotDeck_产品简介.html
- 量子位：清华开源智能体 OS，任务成本降 83%，记忆可查可改：https://www.qbitai.com/2026/10/503935.html
- PilotDeck 2.0 与「记忆可查可改」：https://ai-blog.cloud/tool/pilotdeck-2-0-open-source-agent-os/
- PilotDeck 2.0 Harness 发布报道：https://bsia.org.cn/site/content/35237.html
- 清华 Agent 工作台 2.0（人人都是产品经理）：https://www.woshipm.com/ai/6478827.html
- arXiv AgentOS（2603.08938）：https://arxiv.org/abs/2603.08938
- 字节 GUI Agent 26K stars：https://www.c114.net.cn/industry/58837.html

**持久执行**
- Temporal 22K stars 调研：https://east196.cn/index.php/2026/08/29/temporal-调研-22k-stars-的持久执行-workflow-引擎-为什么-6-年老牌开源能跟-airflow-拼刺刀/
- Temporal 仓库：https://github.com/temporalio/temporal
- Restate × pydantic-ai：https://pydantic.dev/articles/restate-durable-execution-pydanticai
- pydantic-ai durable execution（Temporal 后端）：https://pydantic.dev/docs/ai/capabilities/durable_execution/temporal/index.md
- DBOS：https://pypi.org/project/dbos/3.2.0/ ｜Vercel AI 集成：https://docs.dbos.dev/integrations/vercel-ai
- Orkes Durable Adaptive Graphs：https://orkes.io/content/ai-cookbook/dynamic-workflows
- 产业判断「durable execution 是下一个 agent 基础设施子赛道」：https://forkast.news/durable-execution-is-the-next-agent-infrastructure-sub-sector/

**事项图 / 决策层**
- @claxedo/workgraph：https://www.npmjs.com/package/@claxedo/workgraph
- agent-workgraph：https://github.com/ranausmanai/agent-workgraph

**可信 / 审计**
- agentlens-audit（PyPI）：https://pypi.org/project/agentlens-audit/0.3.0/
- AgentOps：https://github.com/AgentOps-AI/agentops
- browsertrace（computer-use 可追溯）：https://github.com/aaronlab/browsertrace
- agent-glovebox（gVisor 沙箱）：https://github.com/AlexanderMattTurner/agent-glovebox
- 微软「approval 必须放在模型之下」：https://radar.zurb.com/article/microsoft-agents-cannot-be-trusted-so-put-the-approval-below-the-model
- 沙箱层商品化：https://tech.yahoo.com/ai/meta-ai/articles/commoditized-agent-sandbox-layer-become-202439275.html

**技能与上下文**
- Agent Skills 规范：https://agentskills.io/specification
- tiangolo/agentskills：https://github.com/tiangolo/agentskills
- anthropics/skills：https://github.com/anthropics/skills
- awesome-context-engineering：https://github.com/yzfly/awesome-context-engineering ｜https://github.com/royalpinto007/awesome-context-engineering
- mem0 对比（mnem vs mem0）：https://github.com/Uranid/mnem/blob/main/docs/src/comparisons/mem0.md

**生成式 UI / 办公**
- awesome-llm-apps（117K stars 报道）：https://developer.aliyun.com/article/1759443
- generative_ui_agents 子目录：https://github.com/Shubhamsaboo/awesome-llm-apps/tree/main/generative_ui_agents
- genspark-ai/genoffice：https://github.com/genspark-ai/genoffice ｜官网：https://genoffice.ai/
- GenOffice 发布博客：https://www.genspark.ai/blog/genoffice-open-source-ai-office
- kaos-office（PyPI）：https://pypi.org/project/kaos-office/0.1.0/
- genpark office skill：https://github.com/Alpha-Park/genpark-genoffice-document-agent-suite-skill

**垂直**
- 造价大师：https://github.com/jayden-lg/costing-master ｜SKILL.md：https://github.com/jayden-lg/costing-master/blob/main/SKILL.md
- 书童 AI-Novel-Writer：https://github.com/EthanYoQ/AI-Novel-Writer
- novel-studio：https://github.com/Xiaoyangy/novel-studio

---

## 附录 A：与 gaea 现状的对齐（依据 `.gaea/AGENTS.md`，非外部调研）

外部范式要落到 gaea 已有资产上才有意义。按 AGENTS.md 记载的现状逐条对齐：

| gaea 现状（AGENTS.md 记载） | 对应外部范式 | V5 的最小动作 |
|---|---|---|
| **单模型架构**：一个 executor 完成规划与执行，无独立规划器；内置 17 个核心工具 | PilotDeck 的 WorkSpace / Orkes 的动态图 | 不要加第二个 planner。把「事项」做成确定性状态机，executor 只负责在状态上选下一步 |
| 绑定面 635 个导出方法，11 个板块门面（CoreB/OfficeB/MemoryB/CostB/…） | genoffice 的「GUI + CLI + Skill」三入口 | tool 面已收敛，改造为「可重放 step + 幂等键」的边际成本低（启示 2 可立即落地） |
| 记忆 = SQLite `%APPDATA%\gaea\Hephaestus.db` facts 表（按项目 slug 隔离）+ **文档记忆 AGENTS.md 层级** | OpenHuman「可读记忆」 | gaea 其实**已经有可读记忆的雏形**（AGENTS.md 层级就是人类可读、可编辑的记忆）。V5 只需把它产品化：让用户能在界面里打开、编辑、看到「AI 记住了什么」 |
| **双空间硬隔离**：工位（办公/造价/资料）与乐园（轻语/小说/绘梦）记忆分区、模型策略各配、上下文永不跨界，跨空间仅用户显式发起 | muldro 的 world model 分域；patchwork-os 的 policy | 这是竞品**没有**的结构性资产，且正好对应「重资产业务 vs 创作」两种信任模型。V5 应把它从「隔离」升级为「两套 policy + 两套 receipt」 |
| 写入类操作「全部硬性确认（含 yolo）」、子代理不继承持久化写入工具 | patchwork-os 审批闸 | 已有审批雏形，**缺 receipt**：把每次确认落成可查凭证（启示 3 的缺口就在这一项） |
| 造价域已有「工料法成本数据库」（工料机资源库 + 综合单价分析 + 五表，v4.457.0） | costing-master（GB/T 50500-2024、省级定额、广联达） | 数据层已深于多数开源项目；差距在「规范条款可核验」与「广联达互通」两点 |
| 本地产 AI 底座 Herdsman（localhost:8080/v1，含对话/视觉/OCR/ASR/TTS/生图） | Restate/DBOS 的单机 durable 形态；OpenClaw 的总线 | 本地单机是优势：journal 落 SQLite 即可，不需要服务端集群 |
| 根 LICENSE = 私有 All Rights Reserved；未来开源须另发许可证 | — | **借鉴开源代码前必须逐项核许可证**（本次未能获取多数项目 license，见缺口清单第 5 条），只借架构思想、不复制代码 |

一句话：gaea 缺的不是能力，而是**把已有能力组织成「事项 + 证据 + 回执」的骨架**；而这恰好是 2026 年开源生态刚刚开始做、且都没做到中国重资产业务里的事。

---

## 附：数据缺口清单（下一轮需补）

1. 全部项目的精确 star 数与最近 commit 时间（本次环境无法访问 GitHub API / 页面）。
2. `@claxedo/workgraph`、`agentlens-audit`、`kaos-office` 的实际功能（README/包描述未能读取）。
3. `heavenrousy/neuro-estimator` 是否存在及其定位（检索全量误命中学术项目 NEUROeSTIMator）。
4. PilotDeck 2.0「任务成本降 83%」的基线定义与评测方法（仅媒体口径）。
5. kuse_cowork、ekko-studio、OpenHuman 的许可证（license 影响可借鉴方式）。
