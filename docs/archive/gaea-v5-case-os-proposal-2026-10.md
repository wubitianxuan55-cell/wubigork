# gaea V5 立项书：案头（Case OS）

> 从「会聊天的工具箱」到「可托付的个人工作操作系统」
>
> 2026-10-29 · 基线 v4.527.0 · 状态：立项建议稿（待拍板） · 分类：总纲 · 架构立项

---

## 0. 一页结论

**判断**：gaea 在 V4 已经证明「一个人 + 本地模型 + 真编辑 + 垂类数据」可以做出巨头产品做不到的深度；但 V4 的组织方式仍是「板块—页面—会话—工具调用」。这套方式在 527 个版本后已经触顶：功能越多，用户越难把 gaea 当成「办事的地方」，而不是「一堆功能的地方」。

**V5 的主轴**：把 gaea 从「以页面和会话组织的工具箱」重构为「以**事项（Case）—事务（Transaction）—证据（Evidence）**组织的个人工作操作系统」。用户带来一件事，gaea 交付一件**可验收、可回滚、可复用**的成果。

**六个转变**：

| # | V4 | V5 |
|---|---|---|
| 1 | 板块 / 页面是容器 | 事项是容器，板块降为「镜头」 |
| 2 | 聊天轮次是执行单位 | 事务是执行单位（幂等、检查点、可补偿） |
| 3 | 检索片段拼上下文 | 事项上下文编译器（预算内、带出处、缓存友好） |
| 4 | 686 个绑定 / 48 个内置工具 | ~48 个能力契约（类型化、风险分级、可验证） |
| 5 | 审批 = 弹窗 | 委托契约 = 风险分级 + 监督预算 + 自主度阶梯 + 回执 |
| 6 | 消息数 / 功能数 | 已验收事项数（North Star） |

**首个样板**：造价案卷——「12 份报价单 → 询价归一 → 组价 → 成本测算 xlsx + 说明 docx + 证据链」，端到端跑通，再复制到办公与小说。

**投入**：12 个月、5 个阶段、单人 + 子代理；第 6 周设 kill gate。**不重写**：Wails / Go / React / SQLite 全保留，V4 可回退。

**建议**：**立项**，按 §11 的 E 方案（案头 / Case OS）推进；先用 6 周 P0 验证内核（真实造价案卷、崩溃恢复、回滚），过了 kill gate 再进 P1。若 P0 失败，按 §11.2 退到「轻账本」或造价单点。

**请拍板**：见 §12 的 8 项。最关键的一项：**是否确认「事项—事务—证据」为 V5 唯一主轴**。

---

## 1. 调研方法与边界

本立项书由三条证据线合成：

1. **市场与竞品**：2026-01 至 2026-10 的公开报道、产品文档、行业报告摘要；覆盖通用助手、办公 Agent、本地客户端、垂类工具、基础设施五层。底稿见 `.gaea/v5-research/market-competitors.md`。
2. **社区与 GitHub**：Hacker News、Reddit、V2EX、知乎、少数派、即刻，以及 2026 年活跃的开源项目；重点看「用户实际怎么用、在哪弃用、怕什么」。底稿见 `.gaea/v5-research/community-sentiment.md`、`.gaea/v5-research/github-landscape.md`。
3. **代码与文档审计**：gaea 仓库 v4.527.0 的绑定面、运行时、事件日志、审批、办公 DAG、小说门禁、造价库；每条结构判断尽量带 `file:line`。底稿见 `.gaea/v5-research/v4-structural-audit.md`。

**局限（必须先说）**：

- 本次桌面研究无法读取部分付费报告与全文（工具对若干域名返回非公网 IP）；涉及具体数字处均标注来源与日期，拿不到精确值时写「报道口径」或「未获取」，不编造。
- 竞品功能以公开文档与搜索摘要为准，可能存在版本时差；结论尽量落在「结构判断」而非单点功能。
- 社区样本有幸存者偏差：发声者多为重度用户与开发者，沉默的大多数（尤其造价从业者）代表性不足。
- 代码审计是只读抽样，不是全仓逐行；V4 的实际行为以真机走查为准。

**评估一把尺**：一个提案算不算「革命性」，看它是否改变了**组织单位**与**执行契约**，而不是增加了功能；算不算「实用性」，看它能否**复用 V4 资产、在 12 个月内端到端跑通、并且比 V4 更快**。

---

## 2. 市场：五个已经发生的变化

### 2.1 从「给出答案」到「交付结果」

2026 年头部产品的叙事已经从「更聪明的聊天」转向「把事做完」：Claude Cowork 直接操作文件夹、产出文档；WorkBuddy / 灵犀把「项目」放在首屏；GenUI 被定义为「让 Agent 从给出答案走向交付结果」（阿里云《AI Agent Handbook》第 26 章）；Genspark 甚至开源了 AI 办公套件 GenOffice。

**对 gaea 的含义**：gaea 的「真编辑 docx/pptx/xlsx、产物面板、GenUI」方向是对的，但 V4 把这些能力挂在「板块」上，用户要先学会 gaea 的导航，才能开始办事。V5 必须让**产物**成为第一现场。

### 2.2 从「会话」到「事项」

竞品在 2026 年集体转向项目 / 任务空间：Cowork 有 Projects，WorkBuddy 有项目栏，灵犀有任务空间，Manus 有会话工作区。gaea 自己的路线图也早已拍板「板块制 → 任务空间制」「双空间硬隔离」，但 V4 的落地仍以板块页面为主，任务是会话里的一个卡片。

**对 gaea 的含义**：这不是「再加一个项目管理页」，而是把**事项**升级为内核对象：有目标、计划、产物、证据、决策、预算、验收、归档。会话只是事项的一种输入/输出界面。

### 2.3 从「模型能力」到「上下文工程」

2026 年行业讨论的焦点从「模型够不够强」转向「上下文给得对不对」：Economic Times 的「context engines」、Atlan 的「Context Just Went From Hype to Reality」、OFweek 的「上下文成为新的价值衡量基线」都在说同一件事。gaea 的阶段九「上下文即预算」已经做了双阈值压缩、混合召回、时序失效、KV 缓存友好，这是 V5 的现成地基。

**对 gaea 的含义**：V5 不该再造一个通用 RAG，而应做**事项级上下文编译器**：只给当前步骤必要的事实、证据、决策与产物摘要，带出处、可审计、预算硬约束。

### 2.4 从「能力展示」到「信任契约」

2026 年最刺眼的不是模型进步，而是事故：Claude Cowork 被报道「一不小心没了 11GB 文件」（InfoQ）、出现可逃逸虚拟机访问 Mac 文件的漏洞（The Hacker News，2026-07）、以及文件外泄风险（PromptArmor）；华为云社区则记录了「用户因为疲劳点错了一次审批」——审批疲劳本身成为安全漏洞。学术侧，arXiv 2606.08919《Oversight Has a Capacity》直接把「监督容量」当成可耗尽的资源；IEEE 的 AgentOps Replay、AgentDebugX 在补「可回放、可归因」的基础设施。

**对 gaea 的含义**：信任不是「多弹一个确认框」，而是**默认值 + 证据 + 回滚 + 监督预算**。谁能把「AI 到底改了什么、为什么、能不能退」讲清楚，谁就能拿到高价值、低容错的活（造价、投标、合同、长篇定稿）。

### 2.5 从「功能护城河」到「技能商品化」

2026-10，Anthropic 把 Agent Skills 标准捐给 Linux 基金会旗下的 Agentic AI Foundation（艾瑞、IT之家、Chinaz 报道）。这意味着技能格式、技能市场、甚至一部分技能内容会快速商品化；MCP 也早已是公开标准。

**对 gaea 的含义**：不要做技能市场（已在册不做），也不要指望「我支持某个技能格式」当卖点。真正的护城河是：**垂类能力契约 + 用户事项数据 + 可验证的交付记录**。技能只是能力的封装格式，不是资产本身。

### 2.6 竞争地图（五层）

| 层 | 代表 | 强在哪 | 弱在哪 | gaea 的位置 |
|---|---|---|---|---|
| 通用助手 | ChatGPT、Claude、Gemini、豆包、夸克、Kimi | 模型、分发、免费、生态 | 云端、通用、垂直深度浅、数据不在本地 | 不拼通用对话 |
| 办公 Agent | Cowork、WorkBuddy（腾讯）、灵犀（金山办公）、Copilot、Notion AI、Manus、Genspark | 项目空间、文件操作、企业集成 | 订阅制、云端、国内垂类数据弱、审计浅 | 用「事项 + 证据」打差异 |
| 本地客户端 | Cherry Studio、OpenHuman、kuse_cowork、AnythingLLM、LM Studio | 本地、隐私、模型自由 | 偏聊天/记忆，办公真编辑与垂类引擎弱 | 本地 + 真编辑 + 垂类 |
| 垂类工具 | 广联达、鲁班、品茗、Togal、NovelCrafter、Sudowrite、P6 | 领域数据、规范、工作流 | 单域、AI 浅、跨域弱、贵 | 跨域事项 + 个人所有权 |
| 基础设施 | MCP、Agent Skills、Temporal/Restate、记忆/上下文引擎、GenUI | 标准化、可复用 | 不直接服务终端用户 | 站在其上，不做平台 |

**空位**：目前没有人把「本地优先 + 真编辑 + 垂类深度 + 可审计事项」四件事同时做好。巨头有前两项但云化、通用；本地工具有前两项但浅；垂类工具有第三项但单域、封闭。这个交叉点就是 V5 的战场。

### 2.7 巨头战场：不要用 breadth 对抗 breadth

2026 年办公 Agent 已是巨头战争：36kr 报道「3 大巨头、19 款应用抢滩」；DoNews 称「AI 办公不止三国杀」；QuestMobile 2026 报告称办公 Agent「三强格局初显」；另有报道称 20 款办公 AI 产品月访问量合计破 3 亿次。这些数字的具体口径不一，但方向明确：**通用办公 Agent 是免费 + 分发 + 生态的战场**，个人开发者用功能数量去拼，必败。

**gaea 不能做的**：通用写作、通用 PPT、通用会议纪要、通用搜索、通用编程。**gaea 能做的**：中国造价、进度计划、长篇小说一致性、CAD 读图、个人数据主权——这些是巨头不愿意弯腰、垂类厂商做不深的地方。

### 2.8 垂类：付费意愿在「错不起」的地方

- **造价**：广联达 2026-10 宣称土建提量准确率 98%；但社区实测文章「同一套图纸试了 6 款造价 AI 工具，免费的那款最贵」说明：用户不怕付费，怕的是**错了还看不出来**。造价基础岗收缩的讨论（服务新干线）进一步说明，工具的价值不是「替代人」，而是「让留下来的人更快、更敢签字」。
- **长篇写作**：NovelCrafter、Sudowrite 证明作者愿意为「一致性 /  bible / 上下文」付费；但它们的本地化、中文网文工艺、整书状态管理都不够。gaea 的场景制、实体卡、章级门禁是现成优势。
- **进度 / CAD**：P6、斑马进度、CAD 读图工具都是「专业贵、AI 浅」；gaea 的 CPM/DCMA/蒙特卡洛与本地 CAD 解析可以做成事项里的能力，而不是独立板块。

**结论**：垂类付费成立，但卖点必须是「可验收」而不是「AI 生成」。造价案卷的验收标准可以精确到「量、价、费、税」与「每一笔价格的来源」；小说案卷可以精确到「实体状态一致、伏笔回收、章级门禁通过」。这是通用 Agent 给不了的。

### 2.9 社区与 GitHub：验证了方向，也暴露了差距

2026 年 GitHub 上出现了一批「个人 AI OS / 决策层」项目，说明 V5 的方向不是臆想：

- **OpenHuman**：4 个月 31K stars，以「可读记忆」切入桌面代理——记忆是入口，但办公真编辑与垂类深度不足。
- **kuse_cowork**：Claude Cowork 的开源替代，证明「本地文件 + 多 Agent + 工作流」有真实需求。
- **patchwork-os**：自称「personal AI 的决策层」，本地优先，每个高风险动作必须用户点头，worker 靠历史记录赚取自主权，每个决定留 receipt——这几乎是 V5 信任模型的社区版预演。
- **muldro**：个人 AI OS，维护 world model，所有外部写入走 trust/approval，技术栈 Postgres/Redis/Qdrant/Neo4j——重，但方向一致。
- **PilotDeck（OpenBMB）**、清华 2026-10 开源的智能体操作系统（报道称任务成本降 83%、记忆可查可改）、arXiv 2603.08938 的 AgentOS：学术与基础设施层在把「Agent OS」概念化。
- **GenOffice、kaos-office、genpark office skill**：办公文档 Agent 已经开源化，单纯「能改 docx」不再是壁垒。
- **costing-master（造价大师）**：遵循 GB/T 50500-2024、覆盖投资估算到竣工决算、兼容各省定额与广联达——个人开发者已经在做造价垂类，gaea 的窗口不是无限的。
- **AgentDebugX、agentlens-audit、AgentOps Replay**：失败归因、审计回放正在成为独立工具层。

**差距**：这些项目大多面向英文/创始人场景（Gmail、Slack、GitHub），或者只做基础设施；没有一个把「中国造价/进度/小说 + 本地文件 + 事项账本」组合起来。**这正是 gaea 的空位，但窗口正在关闭**。

### 2.10 社区与 GitHub 的十条硬约束

社区调研（`.gaea/v5-research/community-sentiment.md`）与 GitHub 扫描（`.gaea/v5-research/github-landscape.md`）给出的不是「功能清单」，而是**采用门槛**。2026 年用户已经不问「你能不能做」，而问「你敢不敢让它碰我的文件」。十条硬约束必须写进架构，而不是写进营销页：

| # | 社区信号（证据） | 硬约束 | 落在 V5 哪一层 |
|---|---|---|---|
| 1 | SaaS-Bench：办公 computer-use 任务 Claude 通过率 <4%（量子位 2026-05） | 不做「全自动办公」叙事；默认半自动、逐步验收 | L2 验证器 + 验收 |
| 2 | Cowork 删 11GB 文件；疑似删 15 年家庭照片（InfoQ / AI Incident DB #1441） | 删除必须可撤销（回收站/快照），批量操作先 dry-run | L2 补偿器 + R3 |
| 3 | SharedRoot VM 逃逸可读 Mac 凭据；PromptArmor 文件外泄（2026-07 / 2026-01） | 目录×动作权限；凭据/系统目录硬 deny；沙箱边界可视化 | L2 策略 + L0 |
| 4 | 华为云复盘「用户因为疲劳点错了一次审批」；Claude Code 自动模式改默认 | 审批变「风险预算」：分级、批量、配额，而非取消 | L2 监督预算 |
| 5 | Cowork #78750：上下文压缩静默丢掉「已确认的工作」 | 已确认事实锁定，压缩后必须重新对齐并显式提示 | L3 上下文编译器 |
| 6 | OpenClaw 无视 3 次停止指令清空邮箱（2026-02） | 停止键必须是进程级硬中断；支持人工接管续跑 | L2 执行器 |
| 7 | 「没报错，但忘了」成为精确恐惧 | 每步落盘、可核验进度、失败要「响」 | L1 账本 |
| 8 | ClawHub 头部 skill 含恶意代码；3.3 万 Skills 多数死在安装 | 技能签名、权限清单、默认沙箱；安装前展示读写范围 | L2 技能 |
| 9 | 本地 30B 可跑，但「本地」目前只对愿意折腾的人成立 | 本地档位向导 + 能力边界提示，本地是默认路径 | L3 模型路由 |
| 10 | kuse 3 个月 1000 万美元 ARR；PilotDeck 以 WorkSpace 为单位 | 项目/工作台是一等公民，聊天只是入口 | L1 + L4 |

**GitHub 侧的五个架构确认**：

1. **事项/工作舱已被验证**：PilotDeck 2.0（清华 + 面壁）以 WorkSpace 为单位、自称 Harness，是与 V5 最同构的正面威胁；它证明「事项容器」不是臆想，但也意味着**叙事窗口正在关闭**。
2. **信任的完备形态 = 风险分级 + 审批闸 + 回执 + 沙箱**：patchwork-os 把「每笔风险动作停下等 yes、worker 靠战绩赚自治、每决策留 receipt」讲成了产品语言；muldro 用 world model + 外部写审批做同一件事。gaea 只做确认弹窗，会落后一代。
3. **持久执行已商品化**：Temporal 的确定性重放、Restate 的单二进制 journal、DBOS 的「数据库即编排器」说明——**不要引入 Temporal server**，在 SQLite 里用 `run/step/effect` 三表 + 幂等键 + 重放即可；断点续跑是及格线，不是卖点。
4. **技能是分发单位，不是插件**：Agent Skills 规范（`SKILL.md` + 渐进披露）已被 Linux 基金会托管；genoffice 用「编辑器 + CLI + skill」绕开宿主，造价大师用 `SKILL.md` 分发。gaea 的能力应可被外部 agent 调用，而不是把能力锁在壳里。
5. **生成式 UI 的关键是共享状态**：agent 改状态、UI 反映状态，而不是 agent 返回一段界面代码；这样审批、勾选、就地编辑才能在客户端零往返完成，并天然可回放。

> 结论：V5 的差异化不在「更会聊天」，而在「**更敢托付**」——事项有边界、动作有分级、结果有证据、失败可回滚、过程可接管。这恰好是社区与 GitHub 同时指向、而巨头与垂类厂商都还没做好的交叉点。

### 2.11 付费与规模：垂类不是小生意

- **造价数字化是几十亿级市场，且云订阅已被接受**：广联达 2025 年数字成本业务收入 47.79 亿元，其中云收入 34.57 亿元、占 72%（同花顺 2026-03-25 机构调研报道）。广联达数字造价市占 **>90%**，是典型「单一厂商把持 + 高毛利订阅」的市场（市场底稿口径）。这说明中国造价用户愿意为「持续更新的数据 + 可核验的工具」付年费，而不是只买一次性软件；也说明正面拼通用造价工具是硬碰硬，机会在「个人/小团队的证据链」而非「替代广联达」。
- **行业底数可查**：住建部已发布《2025 年工程造价咨询统计公报》（2026-09）；具体企业数/从业人数以公报原文为准，本调研未获取全文，不在此引用精确值。
- **办公 Agent 是巨头战场，但市场足够大**：易观 2026-10-20 称 20 款桌面端 AI 原生办公 Agent 月访问合计 **>3 亿次**；QuestMobile 2026-09「三强初显」：腾讯 WorkBuddy **3235.3 万 MAU**（环比 +647.1%）、百度搭子 977 万、千问办公 9 月初破 3000 万用户；AI 原生 App 大盘 5.46 亿 MAU，豆包 4.19 亿断层第一。中金称「AI+办公」打开万亿级市场；观点网引述 2026 年办公智能体市场规模预计 449 亿元。口径不一，但共同说明：通用层会被巨头用免费+分发打穿，垂类层仍有付费空间。
- **价格带已经分层**：ChatGPT Go $8/月（含广告）、Claude Cowork 下放 $20 Pro、M365 Copilot $30/席/月（2026 付费席位 1500 万、+160%）、千问办公高级 19 元/月、Claude Max $200/月。**$20 是心理价位天花板，¥19–30 是国内甜点**；gaea 不产品化，但能力设计要按「用户已在为这些价位付费」的标准做，而不是按「免费玩具」做。
- **通用个人 Agent 独立生存难**：36kr/钛媒体称 Manus「没能逃出巨头的游戏」；前豆包 PC 端负责人齐俊元创业 Today 7 个月融资 13 亿元（腾讯+蚂蚁），媒体仍定调「做 AI 助理但不好赚钱」；Kuse 毛利微正、Cursor 毛利率 -23%。而 kuse 用「NotebookLM 方法重做 Notion」3 个月做到 1000 万美元 ARR（媒体访谈口径）——**有明确工作台形态、有交付物的产品能活**。
- **失败案例划出边界**：豆包 2026-07-03 通知 AI 智能体 7-15 下线、10-15 清用户数据并引导迁收费「猫箱」，引发权益争议；某老板 50 万元做客服 Agent 上线一周关停（日均 2000 单→不足 50 单）。前者说明「平台型 agent 的存续风险」，后者说明「没有验收标准的 agent 不值钱」。
- **写作付费真实但薄**：NovelCrafter $4–20/月 + 自付 AI 成本、Sudowrite $10–59/月；番茄 67% 作者用 AI、一个月下架 4 万本书，85% 新人一年稿费不够买手机。愿意付的是「能过审/能连载」的效率，不是「AI 写作」本身——对应 V5 的小说镜头必须绑定「过审/一致性」验收，而不是「生成更多字」。
- **对 gaea 的含义**：gaea 不产品化、不融资，但可以用同样的尺子检验 V5——如果一件事不能让人「敢签字」，它就不值钱；如果能，用户愿意为它换掉现在的做法。

### 2.12 市场结论：四条可防守的轴

1. **垂类深度**：造价、进度、小说、CAD——通用 Agent 不会弯腰做的领域。
2. **本地与数据主权**：文件不出机、模型可本地、事项账本可导出。
3. **可验证 / 可回滚**：每一笔写入有证据、有回执、能撤销；高容错成本的活才敢交给 gaea。
4. **个人上下文复利**：事项做得越多，实体、决策、技能越厚，换一个通用 Agent 迁移不走。

**一句话**：gaea 不该做「更全的 AI」，该做「更敢托付的 AI」。

---

## 3. V4 诊断：为什么不能靠继续加功能

### 3.1 四个结构天花板

**（1）组织单位错了：板块 / 页面是开发史，不是用户价值**

V4 的导航从 13 个板块瘦到 9 个，首页改成 12 张卡，但本质仍是「功能目录」：用户要先知道「这件事属于哪个板块」，才能开始。可现实里一件事往往跨板块——一次投标要造价 + 进度 + 文档 + CAD；一本小说要大纲 + 实体 + 章节 + 质检。板块是开发分工的投影，不是工作的投影。

**（2）执行单位错了：会话 / 轮次承载不了长活**

`internal/gaea/event/event.go` 定义的是**回合级 UI 事件流**（TurnStarted / ToolDispatch / TurnDone），不是可重放的业务账本。真正可重放的是 `internal/gaea/agent/session/log.go` 的会话日志（seq = 已写行数、payload 无损、模型可见必入），但它只记录「对话」，不记录「步骤/事务/验收」。执行侧 `agent_run.go:18-19` 只有 `Run → runDirect` 一条路径，主循环在 `:126`；中断恢复靠 `interrupted_recovery.go` 把文本块塞回上下文，进程重启即丢——`agent.go:566/571` 的 checkpoint 是会话 flush，不是可续跑的步骤机。办公 DAG 有拓扑执行，但它是办公域的；小说门禁是小说域的；造价组价是造价域的——每个域各有一套「半程持久化」，没有统一的事务语义。

**（3）能力单位错了：686 个绑定 / 48 个内置工具，模型和人都被淹没**

实测 11 个门面共 **686 个导出方法**（OfficeB 201 / NovelB 129 / CostB 70 / CoreB 53 / MemoryB 51 / ModelB 51 / VoiceB 33 / ImageB 30 / SinB 28 / CharlibB 22 / ChatB 18），前端 `wailsjs/go/app/*.d.ts` 同为 686；`.gaea/AGENTS.md:159` 仍写 635、v4.457.0 写 654，均已漂移。更关键的是门面方法体清一色 `return b.a.Xxx(...)` 纯委托，11 个门面共享同一个 `*App`（`bindings_manifest.go:11-23`）——这是**命名分层，不是依赖分层**，加一个「对案子操作」只会让 686 继续涨。工具侧同样膨胀：`internal/gaea/tool/builtin` 有 46 个文件 / **48 个工具**（browser 11、bash/jobs 4、cost 5、schedule 3，另有 office/read/write/edit/docrender、memory/knowledge、cad_ask、chart/diagram、genui、vision、ocr 等），再加 `task`/`run_skill`/`install_skill`/`summarize_file`；文档里的「17 个内置工具」已过时。`internal/app` 591 文件 / 117,871 行，God 类型 `App` 冻结不拆；60 张表里 `case_id` 命中 **0**。对模型来说，工具太多会稀释选择；对人来说，686 个方法意味着每个功能都要单独维护、单独测试、单独解释。V4 的「瘦身」在删死代码，但没有改变「绑定即能力」的生成方式。

**（4）信任单位错了：审批是弹窗，而且在子代理上系统性退化为放行**

`internal/gaea/control/controller_approval.go` 里已有好种子：hardAsk 工具逐条确认、计划批准后自动放行、`schedule_apply` 整计划通道强制确认 + diff 预览、会话级放行。但它们是**按工具散落**的：没有统一风险分级、没有监督预算、没有事项级授权、没有「同类操作批量批准」、没有「历史成功率换自主权」、没有把「证据」作为放行条件。更严重的是 `boot/boot.go:273` 的 `headlessGate := permission.NewGate(policy, nil)`：`:262-267` 注释自陈「子代理无 UI，继承同一 gate」，而 `permission.go:259-261/281-283` 在 Approver 为 nil 时把 `ask` 直接放行——**所有 task / DAG 节点 / run_skill 派生会话的写盘与命令都不弹卡**。结果是两个极端：要么弹窗太多（审批疲劳），要么子代理全放（事故）。

### 3.2 V4 已经做对的事（V5 的地基）

| 资产 | 位置（示例） | V5 怎么用 |
|---|---|---|
| 会话事件日志（append-only + seq 身份 + 折叠重建 + 编译指纹） | `internal/gaea/agent/session/log.go:3-9`、`sink.go` | **直接继承**为案卷事实层（加 `case_id` 分区与 `case.*` 事件类） |
| 办公真编辑 | `internal/office`（docxedit/xlsxedit/pptxedit/ooxml 共 3,265 行） | 事项产物画布；写入走事务 |
| 造价数据库 | `internal/gaea/workcost`（14 文件） | 造价能力包 + 价格证据 |
| 小说门禁 | `internal/novelreview/review.go:98`（8 维 + rubric） | 小说事项的验收器 |
| 记忆事件日志 | `internal/gaea/memory/eventlog.go` | 记忆域投影源；与案卷日志分域，不混用 |
| 证据卡 + journal + 回滚（雏形） | `evidence/journal.go`、`app/gaea_verify.go:256-316` | **改造后继承**：8KB 摘要换成 sha256 + 版本 + 事务边界 |
| 上下文编译 | 阶段九双阈值/混合召回/缓存 | 事项上下文编译器 |
| GenUI | `frontend/src/genui`（3,095 行）+ `internal/gaea/genui` | 事项内的交互式产物 |
| 本地引擎 | tianxuan / herdsman / OCR | 能力路由的本地执行端 |
| 双空间 | 工位 / 乐园 | 事项的 principal 隔离（play 不落 journal 的问题要补） |
| 计划批准 / diff 确认 / 回滚 | 审批桥、schedule 通道 | 事务运行时的策略原语 |
| 发布/CI/文档守卫 | scripts/ci.ps1、check-docs.mjs | V5 的工程纪律 |

### 3.3 必须换掉的四个假设

1. **「功能越多，产品越强」** → 功能越多，选择成本越高；V5 以「一件事能不能做完」衡量，不以功能数衡量。
2. **「会话是产品的中心」** → 会话是入口之一；事项才是中心。
3. **「页面是功能的容器」** → 页面是镜头；事项是容器。
4. **「审批 = 弹窗」** → 审批是委托契约的一部分；默认值、证据、回滚、监督预算共同构成信任。

### 3.4 连续性：V5 不是重写

- 保留 Wails v2 + Go 1.26 + React/TS/AntD/Vite + SQLite。
- 新增 `internal/caseos` 包，不动的域先通过适配器接入；V4 绑定保留为 legacy。
- 前端新壳与旧板块并存，按「镜头」逐个迁移；V4 模式可回退。
- 数据库只加表/加列，不做破坏性迁移；旧项目可导入为事项（尽力而为）。

### 3.5 审计结论：天花板不在绑定面，在运行时与编排层

2026-10-29 的只读结构审计（`.gaea/v5-research/v4-structural-audit.md`，65 行证据表，全部 `文件:行` 可复核）给出三个判断：

1. **绑定面不是天花板，是天花板的下游**：686 个方法可以原样搬运（零重构风险），但它承载不了 case 维度；每加一个「对案子操作」就是再往门面加方法。真正要换的是**状态所有权**，不是命名空间。
2. **最大的坑是运行时**：无 step 持久化、无 resume、审批在子代理上全放行、8KB 摘要当变更真值。这四项里任何一项不修，「事项可托付」就是空话。
3. **70% 的迁移工作量在把 `internal/app` 里的编排抽出来**：`gaea_dag.go` 806 行、`gaea_verify.go` 回滚、`gaea_deliverable_registry.go` 产物登记、`gaea_ui_extra.go` 1,112 行，都长在 God 类型上；而 office / docmd / schedule / cad / ComfyUI 这些包边界干净，搬起来是低风险。

| 迁移层级 | 内容 | 工期 |
|---|---|---|
| **低**（机械改动） | 15 张表加可空 `case_id` + 索引；`knowledge`/`cost_entries` 的 `name UNIQUE` 改 `UNIQUE(case_id,name)`；13 个页面组件零改动降级为 lens；文档数字订正（635→686、17→48、facts→60 表） | 1 周内 |
| **中**（改签名不改语义） | `routeModel(feature)` → `Resolve(ctx, caseID, feature)`；产物登记键加 `CaseID`；写类绑定加 case 上下文（读类靠 session→case 映射）；`tasks` 补 `case_id` | 2–4 周 |
| **高**（新建/重构） | step 持久化 + resume；审批授权域重构（含 `boot.go:273` 装配）；App 编排抽成 case runtime；证据真值换 hash/版本/事务边界 | 1–2 个季度 |

**审计给出的六个 P0**，V5 首批必须处理：①子代理审批全放行；②无 step 持久化；③8KB 摘要当真值；④文档数字成体系漂移；⑤`knowledge`/`cost_entries` 全局唯一命名；⑥play 空间不落 journal、审计链断点。前四项直接决定「敢不敢托付」，后两项决定「账本能不能自洽」。

---

## 4. V5 定位与总体架构

### 4.1 一句话

**gaea V5「案头」= 可托付的个人工作操作系统：每一件被托付的事，都有目标、计划、产物、证据、决策与可回滚的执行记录。**

### 4.2 三根支柱 + 一个复利

| 支柱 | 定义 | 解决 V4 的什么 |
|---|---|---|
| **事项（Case）** | 有目标、范围、验收标准、预算、产出物的持久工作对象 | 板块/页面不是用户价值单位 |
| **事务（Transaction）** | 一次能力调用的最小执行单元：幂等、检查点、可补偿、可审计 | 会话轮次承载不了长活 |
| **证据（Evidence）** | 产物中的每个关键断言都能追到来源/计算/决策 | 审批弹窗不等于信任 |
| **技能结晶（Crystallization）** | 结案后把计划、决策、修复蒸馏成可复用技能 | 做完一次，下一次不从零开始 |

### 4.3 六个转变（革命性所在）

| 维度 | V4 | V5 | 为什么是革命性的 |
|---|---|---|---|
| 组织 | 板块 / 页面 | 事项 / 镜头 | 用户按「事」组织，不按「功能」组织 |
| 执行 | 聊天轮次 + 工具调用 | 事务 + 计划 DAG | 长活可断点、可恢复、可撤销 |
| 上下文 | 检索片段 | 事项上下文编译器 | 只给必要信息，带出处，预算硬约束 |
| 能力 | 686 绑定 / 48 内置工具 | ~48 能力契约 | 模型选择变少，行为变可预测 |
| 信任 | 审批弹窗 | 委托契约 + 监督预算 + 回执 | 从「点确认」到「可托付」 |
| 价值 | 消息数 / 功能数 | 已验收事项数 | 产品好坏可度量 |

### 4.4 不是什么

- **不是重写**：不换框架、不换语言、不换存储；是换主轴。
- **不是平台 / 市场**：不做技能市场、不做插件生态、不做多租户；在册边界继续有效。
- **不是全自动员工**：保留人在环；高风险动作必须可解释、可批准、可回滚。
- **不是通用办公套件**：不拼广度；只做「有验收标准的深度事项」。
- **不是 24/7 常驻**：按需运行，事项可暂停/恢复，不做昼夜运转。

### 4.5 架构原则

1. **本地优先**：文件、账本、模型默认在本机；云端是可选能力，不是前提。
2. **确定性轨道 + LLM 判断**：算术、规范、格式用确定性引擎；LLM 负责理解、规划、表达。绝不让 LLM 算钱。
3. **最小权限**：能力契约声明副作用；默认拒绝；按事项授权。
4. **不许静默写入**：任何用户文件写入必须有 base hash、diff 预览、回执、补偿路径。
5. **空间硬隔离**：工位/乐园是 principal，记忆、事项、策略默认不跨界。
6. **一个事实源**：账本是真相，投影可重建；不养第二套状态。
7. **先删后加**：V5 每加一个抽象，必须能删掉对应的一批旧代码/绑定。
8. **可回放**：任何事项能在另一台机器/另一个时间重放（至少到决策层）。

### 4.6 革命性检验：改的是「单位」与「契约」，不是功能

「革命性」很容易变成自我感动。本立项书只认五条结构检验，五条全过才算：

| # | 检验 | V4 | V5 | 判定 |
|---|---|---|---|---|
| 1 | 组织单位是否改变 | 板块/页面是容器 | 事项是容器，板块降为镜头 | ✅ |
| 2 | 执行单位是否改变 | 聊天轮次 + 工具调用 | 事务（幂等/检查点/补偿） | ✅ |
| 3 | 信任模型是否改变 | 审批弹窗 | 委托契约 + 监督预算 + 回执 | ✅ |
| 4 | 能力供给是否改变 | 绑定即能力（686） | 契约即能力（~48，带风险与证据） | ✅ |
| 5 | 价值计量是否改变 | 消息数/功能数 | 已验收事项数 | ✅ |

**不算革命性的部分**（必须诚实）：模型不是自研的、框架没换、存储没换、交互范式仍以桌面为主。V5 的革命性只发生在**软件结构**层——但这一层恰好是过去 527 个版本没有动过的。

### 4.7 实用性检验：复用、时间、可比、可退

「实用性」也不能靠愿景。四条硬检验：

| # | 检验 | 证据 | 判定 |
|---|---|---|---|
| 1 | 复用 V4 资产 | 办公真编辑、造价库、小说门禁、记忆日志、GenUI、本地引擎、双空间全部保留；新代码集中在 `internal/caseos` | ✅ 约 80% 域代码不改 |
| 2 | 12 个月内端到端 | P0 六周跑通 10 步真实造价案卷（headless），P1 十周出可用壳 | ✅ 有 kill gate |
| 3 | 比 V4 更快 | 首个产物时间 -30%、token/产物 -30%；同一任务做 A/B | ⚠️ 待基线测量 |
| 4 | 可回退、可维护 | V4 壳保留一个版本周期；绑定数只降不升；单人 + 子代理可维护 | ✅ |

**革命性与实用性的张力**，用三条设计消解：

1. **内核新、壳渐进**：`caseos` 是新包，旧板块通过适配器接入；不搞「大爆炸重写」。
2. **能力先契约、实现后补**：P0 只 12 个能力，够跑通造价案卷即可；其余能力按需迁。
3. **每个抽象都能删旧代码**：新增 `Case`/`Transaction`/`Capability` 的同时，必须删掉对应的一批绑定或工具；只加不删的提案不批。

---

## 5. 核心架构

### 5.1 分层

```
┌──────────────────────────────────────────────────────────────┐
│ L4 案头 Shell                                                │
│  事项列表 │ 事项工作区（概览/计划/产物/证据/账本/复盘） │ 检查器 │
│  镜头：办公 · 造价 · 小说 · 进度 · CAD · 绘梦 │ 快捷工具抽屉 │
├──────────────────────────────────────────────────────────────┤
│ L3 上下文与记忆                                              │
│  事项上下文编译器 │ 实体时序图 │ 决策账本 │ 模型路由 │ 检索    │
├──────────────────────────────────────────────────────────────┤
│ L2 事务运行时                                                │
│  规划器 │ 执行器（租约/幂等/检查点） │ 验证器 │ 补偿器 │ 策略  │
│  能力注册表（~48 契约） │ 调度器 │ 监督预算 │ 技能结晶        │
├──────────────────────────────────────────────────────────────┤
│ L1 案卷内核                                                  │
│  案卷分区（继承会话日志） │ 哈希链/快照/重放 │ 事项/计划/步骤  │
├──────────────────────────────────────────────────────────────┤
│ L0 资产层                                                    │
│  内容寻址 Blob │ SQLite WAL │ FTS5 │ 向量（可选） │ 文件监视  │
└──────────────────────────────────────────────────────────────┘
```

### 5.2 L0 资产层

- **内容寻址**：导入的源文件、生成的产物、快照都算 SHA-256；原件留在用户目录，账本存「路径 + 哈希 + 版本」，不把用户文件搬进黑盒。
- **SQLite WAL**：单机、零运维、事务可靠；沿用 gaea 现有 SQLite 经验（Hephaestus.db 单连接池的教训要吸取：事项库单独连接池，避免长事务互相饿死）。
- **FTS5 + 可选向量**：中文分词先用现有方案；向量只做补充，不做唯一召回。
- **文件监视**：外部改动可检测；base hash 不匹配时进入「冲突」状态，不静默覆盖。

### 5.3 L1 案卷内核

**一条日志，两种视图**（继承 session log，不另造第二套）

V4 的 `session/log.go:3-9` 已经给出四条硬保证：seq = 已写行数、payload 无损 JSON、「模型可见必入日志」、`request_header` 带编译指纹；前端 `controller_state.ts:57-109` 完全靠它折叠重建。V5 不新建「事件总线」，而是把案卷事件作为 session log 的 `case.*` 分区：

```go
LogEntry{Seq, Ts, Kind, Space, Payload}   // 现有结构不变
Kind = "case.created" | "case.plan.proposed" | ...
Payload = {case_id, type, actor, data, refs}  // 大产物只放 blob_hash + path
```

- **`case_events` 是分区视图（物化），不是第二事实源**：可 DROP 重建；真相仍是 session log 文件。这个关系照抄 `memory_events` / `mem_graph_*` 的分层（`schema.go:411-416`）。
- **哈希链只加在 case 分区**：`hash = sha256(prev_hash || canonical_json(entry) || at)`，`case_events` 存 `seq/prev_hash/hash` 供校验；不改 session log 的写入协议，历史日志不受影响。
- **重放复用现有读法**：`session.ReadLogRepaired` + 新增纯函数 `FoldCase(entries) -> CaseState`；不引入第二套序列化。
- **大产物不塞 payload**：payload 只放 `blob_hash + path + meta`，内容在内容寻址存储里；避免日志膨胀。

```
case_events(            -- 物化视图，可重建
  case_id, seq, type, actor, payload_json,
  prev_hash, hash, at, recorded_at
)
```

- 每个事项一条哈希链；定期快照（每 N 个事件或每个阶段结束）加速重放。
- 事件类型是封闭集合：`case.created`、`charter.updated`、`plan.proposed`、`plan.approved`、`step.started`、`transaction.committed`、`artifact.published`、`evidence.attached`、`decision.recorded`、`approval.granted`、`case.accepted`、`case.archived`、`case.reopened`。
- **重放**是纯函数：`replay(events) -> CaseState`；任何投影（列表、看板、图谱）都可重建。
- **时间旅行**：`case at seq` 可查看任一时刻的产物版本与决策；`fork at seq` 可做「如果当时选另一个方案」。

**聚合根**：

| 聚合 | 关键字段 | 说明 |
|---|---|---|
| Case | id, space, title, goal, acceptance, status, autonomy, budget, parent_id | 事项 |
| Plan | id, case_id, version, dag_json, status, approved_by | 计划 |
| Step | id, plan_id, node_key, capability_id, input_json, status, attempt, lease_until | 步骤 |
| Transaction | id, step_id, capability_id, idempotency_key, input_hash, output_hash, before_hash, after_hash, status, compensation_json | 事务 |
| Artifact | id, case_id, kind, path, blob_hash, version, parent_hash, provenance_json | 产物 |
| Evidence | id, case_id, artifact_id, source_type, locator, quote, hash | 证据 |
| Decision | id, case_id, topic, chosen, rejected_json, rationale, valid_until | 决策 |
| Entity | id, space, case_id, type, name, state_json, valid_from, valid_to | 实体 |
| Approval | id, case_id, step_id, risk_tier, requested_json, decision, reason | 审批 |
| Skill | id, name, version, contract_json, plan_template, eval_json, usage_count | 技能 |

### 5.4 L2 事务运行时

**规划器（Planner）**

- 输入：事项 charter、可用能力契约、历史相似事项、用户偏好。
- 输出：计划 DAG（JSON Schema 校验）+ 成本/时间/风险估计 + 验收标准。
- **模板优先**：造价测算、办公报告、小说章节、进度更新都有模板；LLM 只在模板上做参数化与局部改写，避免每次从零规划。
- 计划是**提案**，默认需要用户批准（低风险模板可自动批准）；计划变更若提高风险，必须重新批准。

**执行器（Executor）**

- 每个步骤一个租约（lease），防并发重复执行；心跳续租。
- 每个能力调用一个事务，带 `idempotency_key = case_id + step_id + capability_id + input_hash`。
- 重试前先判定「副作用是否已发生」：查账本/验 before-after hash/调用能力的 `verify`。
- 检查点：步骤边界写快照；崩溃后从最后一个已提交事务恢复。
- 取消：协作式取消；已发生的副作用走补偿。

**验证器（Verifier）**

- 每个步骤声明 postcondition（文件存在、单元格值、字数、实体一致、CPM 通过、价格来源非空）。
- 每个产物声明 acceptance checks；结案前必须全过，或用户显式豁免并记录理由。
- 验证失败 → 修复步骤（重试/换能力/降级）→ 仍失败则挂起等人工。

**补偿器（Compensator）**

- 能力契约声明 `reversible` 与 `compensation`：`restore_blob`（文件）、`delete_row`（数据库）、`restore_snapshot`（办公文档）、`none`（不可逆，必须 R3）。
- 补偿按事务逆序执行；补偿本身也记账。
- 不可逆操作（对外发送、发布、付款、覆盖无备份文件）一律 R3，逐次确认。

**策略引擎（Policy）**

- Principal：用户 + 空间 + 事项 + 能力 + 路径/网络范围。
- **per-case 授权域**：批准一次「改这个目录的 xlsx」只在该事项内有效；换事项重新授权。这是对 `boot.go:273` 全局 headlessGate 的直接替代。
- **子代理继承受限域，而不是放行**：task/DAG 节点从父事项拿一个「域 + 风险上限 + 预算」的租约；超出即挂起等父级/用户，绝不允许 `Approver=nil → ask 放行`。
- 决策：allow / deny / ask；ask 的粒度：once / pattern / case / always。
- 风险分级 R0–R3（见 §5.7）；risk 必须挂在**动作**上（能力契约字段），不能只挂在 DAG 节点上。
- 监督预算：每事项/每小时最多 N 次打断；同类合并；超预算自动降级为「先做草稿，批量确认」。
- 自主度阶梯：manual → supervised → delegated → trusted；按能力+场景统计成功率，达标升级，事故降级。

**调度器（Scheduler）**

- 本地队列，按事项优先级/截止时间/依赖调度；并发上限可配。
- 不做常驻后台；应用关闭即暂停，重启恢复。
- 子代理是「带租约的步骤」：拿一个子 DAG，返回事务与产物，不返回散文。

### 5.5 L3 上下文与记忆

**事项上下文编译器**：

```
输入：charter + plan state + 最近事务 + 产物摘要 + 实体状态 + 决策 + 证据 + 用户偏好
处理：选择 → 打分 → 打包 → 标注出处 → 预算校验 → 稳定前缀（KV 缓存）
输出：带 [E12] 引用 ID 的上下文块 + 预算报告
```

- 预算：每步骤 token 上限；软阈值 75% 压缩、硬阈值 88% 截断（沿用阶段九）。
- 出处：每个块带 `source_type + locator + hash`；模型引用证据时必须用 ID。
- 稳定前缀：系统提示、能力契约、事项 charter 放前面且不轻易变；提高缓存命中。
- 跨事项引用必须显式授权；默认只读当前事项。

**实体时序图**：`valid_from / valid_to`，支持「当时的状态」查询；小说实体、造价项目、进度对象共用。

**决策账本**：记录「选了什么、否了什么、为什么、何时失效」；上下文编译器优先注入，避免重复讨论。

**模型路由**：按能力选本地/云端；隐私级别、成本、质量三要素；沿用 tianxuan/herdsman。

### 5.6 L4 案头 Shell

**布局**：

- 左：空间切换（工位/乐园）+ 事项列表（按最近/截止/状态）+ 收件箱（拖入文件即立卷）。
- 中：事项工作区，六个视图——概览、计划、产物、证据、账本、复盘。
- 右：检查器——来源、证据、决策、审批、成本、版本。
- 顶：事项标题 + 状态 + 自主度 + 预算 + 全局命令栏（Ctrl+K）。

**六个视图**：

1. **概览**：目标、验收标准、进度、下一步、风险、预算消耗。
2. **计划**：DAG，节点状态、证据、审批、重试；可拖拽调整（变更需重新批准）。
3. **产物**：原生画布（docx/pptx/xlsx/novel/cost/Gantt/CAD/image）；选中即问，改动走 diff。
4. **证据**：来源列表、断言-证据映射、缺口提示。
5. **账本**：时间线、每笔事务的回执（改了什么、为什么、怎么退）。
6. **复盘**：重放、对比、结案报告、技能结晶。

**镜头（Lenses）**：办公、造价、小说、进度、CAD、绘梦。镜头不是板块页面，而是「同一批事项/产物的领域视图 + 领域能力面板」。用户可以直接进镜头做单点工具（保留 V4 的即时性），也可以把结果「升格为事项」。

**快捷工具抽屉**：翻译、OCR、润色、单次问答等不需要事项的场景保留；一键「升格为事项」把临时对话变成可追踪的案卷。

**双空间**：工位（暖纸/理性）与乐园（暗色/感性）是 principal；事项、记忆、策略默认隔离；跨空间引用需显式授权。

### 5.7 风险分级与监督预算

| 级别 | 定义 | 默认策略 | 例子 |
|---|---|---|---|
| R0 | 只读 | 自动，留痕 | 读文件、检索、OCR |
| R1 | 可逆的本地写入（事项沙箱/副本） | 自动，留回执 | 写草稿、生成中间表 |
| R2 | 写用户文件 / 调外部 API（可逆） | 首次批准，之后可按模式批量 | 改 xlsx、发草稿、下载 |
| R3 | 不可逆 / 高成本 / 对外发布 | 每次批准 + 二次确认 | 覆盖原件、发布、付款、删除 |

**监督预算**（直接回应审批疲劳）：

- 每个事项默认最多 6 次打断；超过后切换为「批量审批包」：把同类改动打包成一张 diff 卡。
- 同一能力在同一事项内连续成功 N 次，可升级为「通知而非询问」；一次失败/撤销即降级。
- 审批卡必须包含：改了什么（diff）、为什么（计划步骤）、证据（来源）、怎么退（补偿）。
- 记录「审批耗时/拒绝率/撤销率」，作为自主度调整依据。

### 5.8 能力契约（~48 项）

能力契约是 V5 对 V4「绑定爆炸」的结构性回答。字段：

```json
{
  "id": "office.xlsx.write_range",
  "version": "1.0",
  "input_schema": {},
  "output_schema": {},
  "effect": "write_local",
  "risk": "R2",
  "reversible": true,
  "compensation": "restore_blob",
  "evidence": ["before_hash", "after_hash", "diff"],
  "idempotent": true,
  "cost": {"tokens": "none", "time_ms": 80},
  "scope": ["work", "play"]
}
```

能力族（12 族，约 48 项）：

| 族 | 能力（示例） | 主要风险 |
|---|---|---|
| 摄取 | file.read / file.watch / web.fetch / ocr / cad.read / clipboard.read | R0–R1 |
| 办公 | docx.read/write / pptx.read/write / xlsx.read/write / convert / template.fill | R1–R2 |
| 文本 | draft / rewrite / summarize / translate / extract | R1 |
| 小说 | entity.upsert / chapter.write / consistency.check / foreshadow.track | R1–R2 |
| 造价 | price.lookup / quota.match / compose / variance.check | R0–R2 |
| 进度 | cpm.compute / progress.update / delay.analyze | R0–R2 |
| 图像 | comfy.run / image.edit | R2 |
| 知识 | index / search / cite | R0–R1 |
| 系统 | approval.request / artifact.publish / case.close | R1–R3 |
| 元 | plan.compile / skill.distill / eval.run | R1 |
| 外部 | http.request / email.draft（不直接发） | R3 |
| 本地 | model.route / engine.status | R0 |

**关键纪律**：LLM 只能看到契约摘要（id、用途、输入输出、风险），看不到 686 个绑定；新增功能必须先立契约，再写实现；契约有版本与弃用期。

### 5.9 事项生命周期

```
立事（Intake）
  → 谋案（Plan）
  → 办事（Execute）
  → 验收（Accept）
  → 归档（Close）
  → 复盘（Replay / Fork）
```

| 阶段 | 用户看到 | 系统动作 | 出口 |
|---|---|---|---|
| 立事 | 一句话/拖文件/选模板 | 抽取目标、约束、来源、验收标准；生成 charter 草案 | charter 确认 |
| 谋案 | 计划 DAG、成本/时间/风险 | 模板 + LLM 规划；能力绑定；审批策略 | 计划批准 |
| 办事 | 步骤进度、产物、审批卡 | 事务执行、检查点、证据、验证 | 所有步骤完成或挂起 |
| 验收 | 验收清单、diff、证据缺口 | 跑 acceptance checks；生成结案报告 | 用户接受/返工 |
| 归档 | 事项包、技能草案 | 冻结产物、更新实体/记忆、蒸馏技能 | 归档 |
| 复盘 | 时间线、重放、对比 | 可重放/分叉/导出审计包 | — |

### 5.10 与 V4 的映射

| V4 | V5 | 迁移方式 |
|---|---|---|
| 板块导航 | 镜头 + 快捷工具 | 逐板块改 lens，旧路由保留 |
| 会话 | 事项的一种输入/输出 | 新会话可「升格为事项」 |
| 办公 DAG | 通用计划 DAG 的一个模板 | 适配器包装，节点变步骤 |
| 小说项目 | 小说事项模板 | 文件结构保留，加 case_id 映射 |
| 造价库 | 造价能力包 | 数据不动，加契约 |
| 记忆事件日志 | 事项账本的一个投影源 | 保留，增加 case_id/actor 关联 |
| 审批桥 | 策略引擎 | 保留 hardAsk，增加风险分级/预算 |
| 产物面板 | 产物视图 | 统一 artifact 登记 |
| GenUI | 事项内交互组件 | 不变，增加证据绑定 |

### 5.11 执行日志：SQLite 三表，不引入 Temporal

GitHub 扫描的关键结论：持久执行已商品化（Temporal / Restate / DBOS / Orkes），桌面端**不需要也养不起**一个 workflow server。V5 借语义、不借部署；`runs/steps/effects` 是 `case_events` 的**执行投影**，可从案卷日志重建，只用于 SQL 查询与崩溃恢复，不是第二事实源。

```sql
runs(id, case_id, plan_id, status, started_at, ended_at)
steps(id, run_id, seq, capability_id, input_hash, idempotency_key UNIQUE,
      status, attempt, lease_until, output_hash)
effects(id, step_id, effect_type, target, before_hash, after_hash,
        compensation_json, committed_at)
```

- **至少一次 + 效果去重**：步骤可重试，`effects` 按 `idempotency_key` 去重；写文件前查 before_hash，已生效则跳过。
- **确定性重放**：LLM 调用也记为事件（`llm.request` / `llm.response`），重放时用已存响应，不重新采样；这是与 Temporal 同构的最小实现。
- **租约 + 心跳**：执行中的步骤持租约；进程崩溃后，启动扫描过期租约，从最后一个已提交 effect 继续。
- **补偿栈**：按事务逆序执行 `compensation_json`；补偿本身也写 `effects`，保证「回滚也可审计」。
- **快照**：每 N 个事件或每个阶段结束做一次状态快照，重放从最近快照开始；避免账本变长后重放变慢。

### 5.12 Windows 桌面隔离与硬中断

Wails 把 Go 与 WebView 放在同一进程，工具直接在进程内跑——这是 V4 事故的温床，也是 V5 必须改的地方：

- **执行进程分离**：`gaea-worker.exe` 承载能力调用，主进程通过本地 IPC 派活；worker 放进 **Job Object**，主进程退出/用户点停止 → `TerminateJobObject`，进程级硬中断，而不是「再发一条消息让它停」。
- **文件写入三态**：读原文件 → 写事项沙箱副本 → 验收后 `publish` 到用户路径（带 base hash、diff、回执）。未 publish 的写入不出沙箱。
- **凭据与系统目录硬 deny**：`~/.ssh`、浏览器配置、微信/QQ 数据目录、系统盘关键路径；目录×动作矩阵，默认只读。
- **接管（Takeover）**：暂停队列、保留 worker 状态、打开检查器；用户改完可「继续」，从当前步骤接着跑，而不是从头再来。
- **沙箱边界可视化**：设置页展示「它现在能碰什么」——可读目录、可写目录、可联网域名、当前授权；这比任何隐私承诺都有效。

### 5.13 技能与回执：把「可信」变成可打印的东西

**技能**沿用 Agent Skills 的目录形态（`SKILL.md` + 资源 + 可选脚本），但多两样 gaea 特有的东西：

1. **能力契约清单**：技能声明它需要哪些 capability、什么风险级别、读写哪些目录；安装前展示，默认沙箱运行。
2. **评测集**：技能自带 3–10 个验收样例；没过的技能不进「可信技能」列表。

**回执（Receipt）**是 V5 与 patchwork-os 叙事拉开差距的地方。海外产品把 receipt 当开发者日志；中国重资产业务需要的是**可打印、可归档、可对外出示的「操作凭证/存档单」**：

- 内容：事项编号、步骤、改了什么（before/after hash + diff 摘要）、依据（证据 ID）、谁批准、何时、怎么退。
- 形态：HTML/PDF 可打印；可导出「事项审计包」（账本 + 产物 + 证据 + 回执）zip，用于竣工资料/审计/对账。
- 价值：造价、投标、合同、长篇定稿这些「错了要负责」的场景，回执本身就是交付物的一部分。

---

## 6. 首个样板：造价案卷

选造价，不选办公，理由：**验收标准最硬、证据最值钱、V4 资产最厚、竞品最痛**。

### 6.1 场景脚本

> 用户拖入 12 份供应商报价单（PDF/图片/Excel），说：「按 2024 定额和本地信息价，做一份这个项目的成本测算，输出 xlsx 和说明 docx，标出每一笔价格的来源和偏离。」

### 6.2 步骤与证据

| 步骤 | 能力 | 产物 | 证据 |
|---|---|---|---|
| 1 摄取 | ocr / xlsx.read | 12 份文本 + 表格 | 原文件 hash、页码/单元格 |
| 2 归一 | price.lookup / extract | 统一物料表 | 每行→原文位置 |
| 3 匹配 | quota.match | 定额子目 | 定额编号 + 章节 |
| 4 组价 | compose | 综合单价 | 人材机明细 + 费率 |
| 5 对标 | variance.check | 偏离清单 | 信息价/历史价 |
| 6 出表 | xlsx.write | 成本测算 xlsx | before/after hash + diff |
| 7 出说明 | docx.write | 说明 docx | 每个断言→证据 ID |
| 8 验收 | eval.run | 验收报告 | 量价费税核对 |

### 6.3 验收标准（机器可查）

- 12 份报价单全部有来源定位，缺一份即不通过。
- 每一笔材料价有「来源类型 + 定位 + 时间」；无来源的价格标红并阻断结案。
- 合价 = 量 × 综合单价，逐行复算误差 0。
- 与信息价偏离 > 10% 的项必须有人工说明或决策记录。
- xlsx 打开后格式不丢、公式可复算；docx 目录/编号正确。
- 所有产物有 before/after hash；可一键回滚到导入前。

### 6.4 与 V4 流程对比

| 维度 | V4 | V5 |
|---|---|---|
| 入口 | 造价板块 → 清单库 → 组价 → 办公导出 | 拖文件 → 事项 |
| 中断恢复 | 会话重开，上下文靠记忆 | 从最后一个事务继续 |
| 证据 | 部分价格有来源，散落 | 每笔价格强制证据 |
| 回滚 | 部分通道有 | 全链路可回滚 |
| 复用 | 手工再走一遍 | 结案蒸馏成「报价测算法」技能 |
| 验收 | 人看 | 机器验收 + 人复核 |

---

## 7. 路线图与门槛

| 阶段 | 周期 | 交付 | 出口判据 | 失败动作 |
|---|---|---|---|---|
| P0 内核冲刺 | 6 周 | `internal/caseos`：账本、事项/计划/步骤/事务、能力注册表、执行器、CLI、重放 | 10 步真实造价案卷 headless 跑通；kill -9 可恢复；回滚成功 | 停止，回到 V4 加功能 |
| P1 案头 MVP + 造价 | 10 周 | 新壳（事项列表/工作区/检查器）、造价模板、12 能力、审批策略 | 3 个真实案卷；时间 ≤ V4；证据 100%；回滚可用 | 缩到单模板，或换办公样板 |
| P2 办公 + 小说 | 12 周 | 办公报告/演示模板；小说章节事项；技能结晶 v1 | 小说 10 章过一致性门禁；技能在 2 个案卷复用 | 砍小说，保办公 |
| P3 上下文 + 自主度 | 12 周 | 事项上下文编译器、决策账本、监督预算、模型路由 | 每产物 token -30%；打断 -50%；零无证据产物 | 只保留编译器 |
| P4 迁移 + 硬化 | 12 周 | 镜头迁移、V4 数据导入、评测集、安全审查、V5.0 | 80% 周工作在新壳；P0 不变量全过 | 延长，不硬发 |

**投入估算**（单人 + 子代理）：

- 12 个月，约 1,200–1,600 人时；其中 P0 约 180 人时。
- 现金成本主要是模型 API 与电费；本地模型可把边际成本压到接近 0。
- 最大成本不是写代码，是**迁移与双轨维护**；用「冻结新板块 + 适配器 + 逐镜头迁移」控制。

**Kill criteria（第 6 周）**：

- 不能在无 UI 情况下重放一个真实案卷 → kill。
- 事务/补偿实现需要改动超过 30% 现有域代码 → 降级为「轻账本」方案。
- 单步延迟 > 2× V4 且无优化路径 → 重新设计执行器。

### 7.1 P0 六周逐周计划（可执行版）

| 周 | 交付 | 验收（机器可查） |
|---|---|---|
| W1 | `internal/caseos` 骨架；`cases/case_events/plans/steps/transactions/artifacts/evidence` 表；哈希链 append + replay；CLI `case new/list/show/replay` | 同一事件序列重放两次，状态 JSON 完全一致；篡改任一事件 → 校验失败 |
| W2 | 能力契约注册表；6 个能力（`file.read`/`file.write`/`xlsx.read`/`xlsx.write_range`/`docx.read`/`docx.write`）；R0–R3 策略；回执 v0 | 每个能力有契约 JSON + 输入输出 schema；R2 写入必须带 base hash；回执可打印 |
| W3 | 模板规划器（造价模板）；计划 DAG；执行器（租约/幂等键/检查点） | 注入 kill -9 后重启，从最后已提交 effect 继续；重复执行不产生二次写入 |
| W4 | 验证器（postcondition）；补偿器（`restore_blob`/`delete_row`/`restore_snapshot`）；证据登记 | 回滚到导入前状态；每个产物有 before/after hash；验收清单可跑 |
| W5 | 12 份真实报价单接入（OCR/xlsx）；`price.lookup`/`quota.match` 走现有造价库；出 xlsx + docx + 证据报告 | 12/12 有来源定位；合价复算误差 0；偏离 >10% 有说明 |
| W6 | 端到端演练 + A/B 基线（同任务 V4 vs V5）+ kill gate 评审 | 10 步案卷 headless 跑通；时间/token 有数；不达标 → 停止 |

**P0 退出物**：一个可重放的 `.case` 包（事件账本 + 产物 + 证据 + 回执）、一份基线报告、一份「继续/停止」决议。**没有 UI 也算成功**——P0 验证的是内核，不是界面。

---

## 8. 指标

| 维度 | 指标 | V4 基线（待测） | V5 目标（12 个月） |
|---|---|---|---|
| North Star | 每周已验收事项数 | 0（无此概念） | ≥ 5 |
| 信任 | R2+ 产物证据覆盖率 | 未统计 | 100% |
| 信任 | 回滚成功率 | 部分通道 | 100% |
| 信任 | 审批打断/事项 | 未统计 | ≤ 6，且 -50% |
| 效率 | 首个产物时间 | 待测 | -30% |
| 效率 | tokens / 已验收产物 | 待测 | -30% |
| 复利 | 技能复用案卷占比 | 0 | ≥ 30% |
| 质量 | 造价量价费税复算误差 | 待测 | 0 |
| 质量 | 小说章级门禁通过率 | 待测 | ≥ 90% 一次通过 |
| 可靠 | 崩溃恢复成功率 | 未统计 | 100% |
| 可靠 | 步骤失败自动恢复率 | 未统计 | ≥ 80% |

**基线测法**：选 3 个真实任务（10 份报价测算、月度报告 docx、小说一章），在 V4 上记录时间、token、返工、人工核对点；V5 用同一任务对比。

---

## 9. 风险与对策

| # | 风险 | 概率 | 影响 | 对策 | 触发信号 |
|---|---|---|---|---|---|
| 1 | 过度抽象，做成「架构玩具」 | 中 | 高 | P0 只做一条真实案卷；6 周 kill gate | 2 周内跑不通端到端 |
| 2 | 双轨维护拖垮 | 高 | 高 | 冻结新板块；适配器；逐镜头迁移 | 同一 bug 修两遍 |
| 3 | 事件账本膨胀/变慢 | 中 | 中 | 快照 + 压缩 + blob 分离；事件瘦身 | 重放 > 2s |
| 4 | LLM 不可靠，验收不过 | 高 | 中 | 确定性引擎 + 验证器 + 人工门 | 返工率 > 30% |
| 5 | 审批疲劳复发 | 中 | 高 | 监督预算 + 批量 + 自主度阶梯 | 打断/事项 > 8 |
| 6 | 文件事故 | 低 | 极高 | base hash + diff + 补偿 + 沙箱 | 任何静默覆盖 |
| 7 | 范围蔓延到 6 个域 | 高 | 高 | 每阶段只做一个域；其余留镜头 | 同时改 3 个域 |
| 8 | 单人维护成本 | 高 | 中 | 契约收敛绑定；删旧代码；不建市场 | 绑定数不降反升 |
| 9 | 用户（自己）不用 | 中 | 高 | 首个样板必须比 V4 快；保留快捷工具 | 一周 0 案卷 |
| 10 | 模型许可/合规 | 低 | 中 | 本地模型许可清单；研究许可不商用 | 引入新模型未审 |

---

## 10. 迁移策略

按审计的三级成本推进，不搞「大爆炸」：

| 阶段 | 动作 | 验收 |
|---|---|---|
| **低（1 周内）** | 15 张表加可空 `case_id` + 索引；`knowledge`/`cost_entries` 改 `UNIQUE(case_id,name)`；13 个页面组件零改动，注册方式从 board 改 lens；订正文档数字（686/48/60） | 旧数据可读；`case_id` 为空时行为与 V4 一致 |
| **中（2–4 周）** | `routeModel(feature)` → `Resolve(ctx, caseID, feature)`；产物登记加 `CaseID`；写类绑定加 case 上下文；`tasks` 补 `case_id` | 同一功能在 V4/V5 两边结果一致；写类操作都可追到事项 |
| **高（1–2 季度）** | step 持久化 + resume；审批授权域（含 `boot.go:273` 重装配）；把 `gaea_dag.go`/`gaea_verify.go`/`gaea_deliverable_registry.go` 的编排抽成 case runtime；证据换 hash/版本/事务边界 | 崩溃可续跑；子代理无越权；回滚按 hash 校验 |

**四条纪律**：

1. **并存**：新壳挂 `/next`（或开关），旧板块保留；V4 数据只读兼容。
2. **加列不加表**：能给现有表加 `case_id` / `space` 的，不复制表；不能加的，建映射表。
3. **适配器**：办公 DAG、小说项目、造价项目各写一个 adapter，把旧 API 包成能力契约；域包（office/docmd/schedule/cad/comfy）原样搬。
4. **冻结 + 回退**：P1 起停止新板块功能，只修 P0 bug；V5 出问题可切回 V4 壳，账本不丢、事项可导出。

---

## 11. 边界：V5 不做什么

沿用并强化在册边界：

- 不做 GoalCard、MCP 大全、T3 平台化、跨设备记忆包。
- 不做算量、企业云协同、编程板块融合、壳外 computer use。
- 不做全自动写作、Neo4j、重写/换框架。
- 不做订阅聚合、浏览器/OS 深度集成、端到端实时语音主攻、IM 入口反转。
- 不做技能市场/分发平台、不做多租户、不做 24/7 常驻。
- 不把「通用办公广度」作为目标；只做有验收标准的深度事项。

### 11.1 备选方案：为什么是 E，不是 A/B/C/D

| 方案 | 一句话 | 革命性 | 实用性 | 主要风险 | 结论 |
|---|---|---|---|---|---|
| A 继续加功能（V4.x） | 在板块制里继续加页面/工具 | 0 | 5 | 12 个月后仍是工具箱；绑定继续膨胀 | 否 |
| B 全面重写（换栈） | 新语言/新框架/新存储 | 3 | 1 | 违反「不重写」约束；单人维护崩 | 否 |
| C 云 SaaS / 平台 | 多租户、技能市场、插件生态 | 2 | 2 | 巨头战场；违背本地优先与在册边界 | 否 |
| D 只做造价单点 | 把造价做成独立专业工具 | 2 | 4 | 丢掉跨域事项与个人上下文复利；正面撞广联达 | 备选（P0 失败的退路） |
| **E 案头 / Case OS** | 事项—事务—证据主轴，镜头承载垂类 | **5** | **4** | 过度抽象、双轨维护、6 周跑不通 | **建议** |

**判定规则**：革命性 ≥4 且实用性 ≥4 才立项；只有 E 同时过线。D 是 P0 失败后的退路——即使退回造价单点，「事务 + 证据 + 回执」仍然成立，不会白做。

### 11.2 如果 P0 失败，退到哪

三种退法，按优先级：

1. **退到「轻账本」**：不做完整 Case OS，只把 `case_events` + 回执 + 回滚接到 V4 的造价/办公流程上；成本约 3 周，仍能解决「敢不敢签字」。
2. **退到造价单点**：把 12 份报价单的流程做成独立工具（D 方案），保留证据链；放弃跨域。
3. **退回 V4**：删除 `caseos` 实验代码，V4 不受影响；把调研结论存档，等下一次窗口。

---

## 12. 拍板项

| # | 问题 | 建议 | 不拍的后果 |
|---|---|---|---|
| 1 | 是否确认「事项—事务—证据」为 V5 唯一主轴？ | **确认** | 继续在板块制里加功能，12 个月后仍是工具箱 |
| 2 | 首个样板选造价还是办公？ | **造价**（验收硬、证据值钱、资产厚） | 选办公会变成通用办公 Agent，正面撞巨头 |
| 3 | 是否接受 12 个月 / 5 阶段 / 6 周 kill gate？ | **接受** | 无限期重构，日常使用被拖累 |
| 4 | P1 起是否冻结新板块功能？ | **冻结**（只修 P0） | 双轨维护，永远迁不完 |
| 5 | 账本是否采用哈希链？ | **是**（审计价值 > 成本） | 事后无法证明「没被改过」 |
| 6 | R2 是否允许「首次批准后按模式批量」？ | **允许**，但每批有 diff 卡 | 审批疲劳，用户关掉审批 |
| 7 | 是否保留 V4 壳作为只读兼容层？ | **保留**（一个版本周期） | 迁移风险全压给用户 |
| 8 | 是否把「绑定数 ≤ 100」作为 V5.0 硬指标？ | **是**（以实测 686 为基线，先 ≤100，最终 ~48 契约） | 绑定继续膨胀，模型选择困难 |

---

## 附录 A：主要来源

> 外部链接为公开报道/产品文档/论文索引；部分仅读到标题与摘要，正文未能抓取，已在文中标注不确定性。

**市场与竞品**

- Claude Cowork 官方架构概览：https://support.claude.com/zh-CN/articles/14479288
- Claude Cowork 配置（只读文件夹/沙箱）：https://claude.com/docs/third-party/claude-desktop/configuration
- InfoQ：Claude Cowork「没了 11GB 文件」：https://www.infoq.cn/article/0diNmKkv2y0MXOHKKXD2
- The Hacker News：Cowork VM 逃逸：https://thehackernews.com/2026/07/claude-cowork-flaw-could-let-ai-agent.html
- PromptArmor：Cowork 文件外泄：https://www.promptarmor.com/resources/claude-cowork-exfiltrates-files
- 华为云社区：Claude 的隔离与约束（审批疲劳）：https://bbs.huaweicloud.com/blogs/495775
- 36kr：3 大巨头 19 款应用抢滩：https://eu.36kr.com/zh/p/4035697724266375
- DoNews：AI 办公不止三国杀：https://www.donews.com/article/detail/5331/108944.html
- 钛媒体：个人 Agent 与企业 Agent 分化：https://m.tmtpost.com/8157089
- 广联达造价 AI（98%）：https://www.guandian.cn/article/20261014/613362.html
- 广联达 2025 数字成本收入 47.79 亿 / 云 34.57 亿：http://stock.10jqka.com.cn/20260325/c675521155.shtml
- 住建部《2025 年工程造价咨询统计公报》：https://js.shaanxi.gov.cn/sy/yw/zjglfw/zjgl/202609/t20260911_3689500.html
- 中金：AI+办公万亿市场：https://wap.stockstar.com/detail/IG2026102700013762
- 观点网：2026 办公智能体市场预计 449 亿元：https://www.guandian.hk/article/20260825/590648.html
- 36kr：Manus 没能逃出巨头的游戏：https://www.36kr.com/p/4019308837885576
- 易观 2026-10-20：20 款桌面端 AI 办公 Agent 月访问合计 >3 亿（来源见 `.gaea/v5-research/market-competitors.md`）
- QuestMobile 2026-09：腾讯 WorkBuddy 3235.3 万 MAU、百度搭子 977 万、千问办公破 3000 万（来源见同上）
- 观点网：2026 办公智能体市场 449 亿元：https://www.guandian.hk/article/20260825/590648.html
- 同花顺：广联达数字成本 47.79 亿 / 云 34.57 亿：http://stock.10jqka.com.cn/20260325/c675521155.shtml
- 广联达造价 AI 98%（厂商口径）：https://www.guandian.cn/article/20261014/613362.html
- 豆包 AI 智能体下线与数据清理争议、50 万元客服 Agent 一周关停（来源见 `.gaea/v5-research/market-competitors.md`）
- 市场底稿（含价格带/写作/造价深挖）：`.gaea/v5-research/market-competitors.md`
- V4 结构审计（65 行证据表）：`.gaea/v5-research/v4-structural-audit.md`
- 服务新干线：造价基础岗收缩：https://www.fwxgx.com/articles/248555
- 腾讯云：从基准高分到生产可用：https://cloud.tencent.com.cn/developer/article/2751406
- 阿里云：GenUI / AI Agent Handbook：https://developer.aliyun.com/article/1755090
- 艾瑞：Agent Skills 捐赠：https://news.iresearch.cn/content/202610/570482.shtml
- IT之家：Agent Skills 社区共治：https://www.ithome.com/1/016/811.htm
- Chinaz：Agent Skills 标准捐给社区：https://www.chinaz.com/ainews/31831.shtml
- OFweek：上下文成为价值基线：https://iot.ofweek.com/2026-10/ART-132214-8420-30709061.html
- InfoQ：human-in-the-loop 变成「闭着眼睛点确认」：https://www.infoq.cn/article/5qWsLD6JV8N2zDgGuhK9
- 量子位：清华开源智能体操作系统（成本降 83%）：https://www.qbitai.com/2026/10/503935.html
- 量子位：SaaS-Bench 办公 computer-use 通过率 <4%：https://www.qbitai.com/2026/05/424277.html
- The Next Web：SharedRoot VM 逃逸：https://thenextweb.com/news/claude-cowork-sandbox-escape-mac-files-sharedroot
- GitHub：Cowork #78750 上下文压缩丢弃已确认工作：https://github.com/anthropics/claude-code/issues/78750
- 星岛日报：OpenClaw 无视停止指令清空邮箱：https://www.singtaousa.com/2026/02/24/news/usa/ai-tool-openclaw-wipes-the-inbox-of-metas-ai-alignment-director-despite-repeated-commands-to-stop-executive-had-to-manually-terminate-the-ai-to-stop-the-bot-from-continuing-to-erase-data/
- 掘金：Agent Skills 2.8 亿安装但无安全规范：https://juejin.cn/post/7699431070982340658
- 安天：OpenClaw 技能市场投毒分析：https://m.antiy.cn/research/notice%26report/research_report/OpenClaw_AI_Poisoning_Attack_Analysis.html
- kuse 3 个月 1000 万美元 ARR（访谈口径）：http://api.aitntnews.com/newDetail.html?newId=21301
- Nylas：The State of Agentic AI H2 2026：https://www.nylas.com/agentic-ai-report-h2-2026/
- Nasdaq：Enterprises Hitting Agentic AI Inflection Point：https://www.nasdaq.com/press-release/new-global-report-finds-enterprises-hitting-agentic-ai-inflection-point-2026-01-22

**论文/研究**

- Oversight Has a Capacity：https://export.arxiv.org/abs/2606.08919
- Files as Memory, Modes as Principals：https://zenodo.org/records/20577663
- Securing Agents With Tracked Capabilities：https://dl.acm.org/doi/pdf/10.1145/3786335.3813127
- AgentOps Replay：https://ieeexplore.ieee.org/document/11465473
- AgentDebugX：https://browse-export.arxiv.org/abs/2607.18754
- Reframing LLM Agent Security as an Agent-Human Interaction Problem：https://export.arxiv.org/abs/2605.24309
- AgentOS：https://arxiv.org/abs/2603.08938
- Babysitting AI Agents：https://export.arxiv.org/abs/2610.34748

**社区/GitHub**

- OpenHuman（31K stars 报道）：https://cloud.tencent.com.cn/developer/article/2686137
- kuse_cowork：https://github.com/kuse-ai/kuse_cowork
- patchwork-os：https://github.com/Oolab-labs/patchwork-os
- muldro：https://github.com/shivadharmi/muldro
- PilotDeck：https://atomgit.com/OpenBMB/PilotDeck
- GenOffice：https://github.com/genspark-ai/genoffice
- costing-master：https://github.com/jayden-lg/costing-master
- Agent Skills 规范：https://github.com/tiangolo/agentskills
- awesome-context-engineering：https://github.com/royalpinto007/awesome-context-engineering
- Pega Case Management：https://design.pega.com/patterns/cases/
- 少数派：本地 AI + 自动捕获知识管理：https://ios.sspai.com/post/99595
- V2EX：研发提效 AI 工作流讨论：https://global.v2ex.co/t/1190167
- V2EX：Agentic Coding 两年：https://global.v2ex.co/t/1221802

## 附录 B：能力契约清单（首版）

见 §5.8 表；P0 只实现 12 项：

1. `file.read`
2. `file.write`（带 base hash）
3. `xlsx.read`
4. `xlsx.write_range`
5. `docx.read`
6. `docx.write`
7. `ocr`
8. `web.fetch`
9. `price.lookup`
10. `llm.complete`
11. `approval.request`
12. `artifact.publish`

## 附录 C：数据表 DDL（草案）

```sql
CREATE TABLE cases (
  id TEXT PRIMARY KEY,
  space TEXT NOT NULL,
  title TEXT NOT NULL,
  goal TEXT,
  acceptance_json TEXT,
  status TEXT NOT NULL,
  autonomy TEXT NOT NULL DEFAULT 'supervised',
  budget_json TEXT,
  parent_id TEXT,
  created_at INTEGER NOT NULL,
  closed_at INTEGER
);

CREATE TABLE case_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  case_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  type TEXT NOT NULL,
  actor TEXT,
  payload_json TEXT NOT NULL,
  prev_hash TEXT NOT NULL,
  hash TEXT NOT NULL,
  at INTEGER NOT NULL,
  recorded_at INTEGER NOT NULL,
  UNIQUE(case_id, seq)
);

CREATE TABLE plans (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  dag_json TEXT NOT NULL,
  status TEXT NOT NULL,
  approved_by TEXT,
  approved_at INTEGER
);

CREATE TABLE steps (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  plan_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  capability_id TEXT NOT NULL,
  input_json TEXT,
  status TEXT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 0,
  lease_until INTEGER,
  started_at INTEGER,
  ended_at INTEGER
);

CREATE TABLE transactions (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  capability_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  input_hash TEXT,
  output_hash TEXT,
  before_hash TEXT,
  after_hash TEXT,
  status TEXT NOT NULL,
  compensation_json TEXT,
  at INTEGER NOT NULL
);

CREATE TABLE artifacts (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  path TEXT,
  blob_hash TEXT,
  version INTEGER NOT NULL,
  parent_hash TEXT,
  provenance_json TEXT,
  at INTEGER NOT NULL
);

CREATE TABLE evidence (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  artifact_id TEXT,
  source_type TEXT NOT NULL,
  locator TEXT,
  quote TEXT,
  hash TEXT,
  at INTEGER NOT NULL
);

CREATE TABLE decisions (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  topic TEXT NOT NULL,
  chosen TEXT NOT NULL,
  rejected_json TEXT,
  rationale TEXT,
  valid_until INTEGER,
  at INTEGER NOT NULL
);

CREATE TABLE entities (
  id TEXT PRIMARY KEY,
  space TEXT NOT NULL,
  case_id TEXT,
  type TEXT NOT NULL,
  name TEXT NOT NULL,
  state_json TEXT,
  valid_from INTEGER NOT NULL,
  valid_to INTEGER
);

CREATE TABLE approvals (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  step_id TEXT,
  risk_tier TEXT NOT NULL,
  requested_json TEXT,
  decision TEXT NOT NULL,
  reason TEXT,
  at INTEGER NOT NULL
);

CREATE TABLE capabilities (
  id TEXT NOT NULL,
  version TEXT NOT NULL,
  contract_json TEXT NOT NULL,
  deprecated_at INTEGER,
  PRIMARY KEY(id, version)
);

CREATE TABLE skills (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  contract_json TEXT,
  plan_template TEXT,
  eval_json TEXT,
  usage_count INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);
```

## 附录 D：术语

- **事项 / Case**：一件被托付的工作，有目标、验收、预算、产物、证据。
- **事务 / Transaction**：一次能力调用的最小执行单元，幂等、可检查、可补偿。
- **证据 / Evidence**：支撑产物断言的来源、计算或决策。
- **能力契约 / Capability Contract**：能力的类型化声明（输入输出、副作用、风险、补偿、证据要求）。
- **镜头 / Lens**：领域视图，不是页面容器。
- **监督预算 / Oversight Budget**：一个事项允许的打断次数上限。
- **自主度 / Autonomy**：manual → supervised → delegated → trusted。
- **技能结晶 / Crystallization**：结案后把计划与修复蒸馏为可复用技能。
