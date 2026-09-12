# gaea 市场调研（2026-09-12）

> 定位基线 = `docs/gaea-competitive-landscape-2026.md`（2026-08-29，差异化论证在彼档；
> 本档与其结论一致：本地办公 agent×私人记忆×造价垂直×陪伴的组合无人重合）。
> 本档 = **两周增量刷新 + 迭代候选提取**：新增信号=Copilot agentic GA（2026-04-22）、
> LM Studio Bionic、记忆层 bi-temporal 共识、广联达 DATA+AI 全流程、陪伴市场 2026 规模数。
> 口径：只记录**已发布的产品事实**与**对 gaea 的含义**，不替用户宣布主轴。
> 候选一律交拍板池（§5）。
> 边界对账：本文候选已过删除史/拍板边界筛——T3 平台化、MCP 大全、跨设备记忆包、
> 审阅制当书斋默认、新板块、算量、企业云协同、GoalCard 系、进度 PM 功能对照表、
> 造价广联达功能对照表，均**不在候选内**（见 §5 逐条标注）。
> 检索时间 2026-09-12；市场数字为第三方估计，量级参考用，不当决策依据。

---

## 1. 办公 AI（gaea 核心板块的正面战场）

**市场事实**
- 微软 2026-04-22 宣布 Copilot 的 **agentic 能力在 Word/Excel/PPT 正式 GA**：多步、
  App 内原生操作；2025-09 起的 Agent Mode + Office Agent 路线兑现。2026-07 起
  Word/Excel/PPT agent 可直接在 Copilot Chat 里被调用做跨 App 编排。
  （microsoft.com/microsoft-365/blog/2026/04/22/、techcommunity 2026-07 what's new）
- WPS AI 2.0（2026 升级）：主题/长文档一键成 PPT（逻辑拆解→配模板→排版）、
  合同审查、公文生成、PDF 总结；主打格式兼容+基础功能免费。
- 飞书智能伙伴：定位「字节 AI 工作平台」，重团队协作与组织场景。

**对 gaea 的含义**
- 「agent 直接在文档里多步干活 + 管家/聊天里调 App agent」被巨头验证为正确形态——
  正是 gaea 已有的形态（真编辑三件套 + 办公流水线 DAG + 管家调办公）。方向对，
  竞争在加剧（LM Studio Bionic 也在做本地 agent 出文档/幻灯片，见 §8）。
- 差距点在**交付体验**：WPS/MS 把「一句话→成品文档」做到了大众可用；gaea 的 DAG
  交付物打磨（成品直出、少盖章）是可对照的迭代方向，且完全落在「办公主角位」内。

## 2. 记忆（gaea 记忆OS）

**市场事实**
- 开发者记忆层三分：**Mem0**（采用度最高，~55k stars，LongMemEval 93.4% 自报，
  基准争议在案）、**Zep/Graphiti**（**双时间轴 bi-temporal 知识图谱**，自报 71.2%）、
  **Letta/MemGPT**（agent-OS 式记忆）。结论式共识：「没有赢家，各家赌『记住』的
  定义不同」。Medium 六维对比（2026-05）。
- 消费侧：ChatGPT/Claude 内建记忆被评测认为**仍简陋**（扁平摘要式，不可查不可管）。
- 2026 年记忆系统关键词：自适应衰减（adaptive decay）、离线能力、可评测
  （LongMemEval 类基准）。

**对 gaea 的含义**
- gaea v4.210 语义图谱（memory_events 投影三向边）+ v4.213 三态生命周期
  （固化/衰减/可查）与市场前沿**同构**：事件日志≈Zep 的时间轴，DecayScore≈adaptive
  decay。方向再次被验证。
- 市场给出的两个可借鉴增量：① **双时间轴口径**（事实发生时间 vs 记录写入时间，
  Zep 核心卖点）——gaea 事件已有 At 字段，补口径是小刀；② **记忆质量可评测**
  （repo 已有 retrieval-eval-set.md，可对齐 LongMemEval 思路）。均属「加厚私人记忆」
  过滤内合规项。

## 3. 造价（两翼之一）

**市场事实**
- 广联达 2026-04-23 发布「DATA+AI·造价更轻松」：**AI组价专家、AI智能提量/算量、
  AI清标、AI询比价、无感建库**，配 AI 开放平台（CAD 识别、组价方案推荐），
  主打企业级全流程。
- 个人级/轻量级造价 AI 在国内检索面基本空白（广联达生态外）。

**对 gaea 的含义**
- gaea AI 组价 v4.2 在产 + 组价确认进库（=「无感建库」的私人版）已占住个人级席位；
  广联达的算量/清标/询比价是企业交易环节能力，且算量/企业云协同**已拍板不做**。
- 唯一值得留观察窗的：价格带/询价数据从哪来（gaea 组价已有价格带+证据链）。
  数据源问题需用户拍板，不自行动。

## 4. 进度（两翼之二）

**市场事实**
- ClickUp Brain（笔记/聊天→结构化任务、工作区问答）、Asana AI（智能状态、负载）、
  monday.com（做成完整 agentic 平台）——2026 年 PM 三强全上了 AI，但全部是
  **任务/协作层** AI。
- **MS Project + Copilot 在检索面存在感很弱**；CPM/双代号/专业排程的 AI 化在
  个人级市场是空席位。

**对 gaea 的含义**
- gaea 进度基本功（双代号/DCMA 体检/挣值/蒙特卡洛/双工期搭接）在个人级没有对标品，
  是真稀缺席位；但「不单独立项做 Project、进度只作为办公产物被 schedule_* 引用」
  已拍板——**不翻案**，此条只作为「维持现状就有差异化」的证据记录。

## 5. 候选提名表（交拍板池，不替用户宣布）

| # | 候选 | 依据 | 与既有拍板关系 |
|---|---|---|---|
| 1 | 记忆双时间轴口径（事实时间 vs 记录时间） | Zep bi-temporal 是市场记忆层核心卖点；gaea memory_events 已有 At | 合规（加厚私人记忆） |
| 2 | 记忆质量可评测（对齐 LongMemEval 思路，接 retrieval-eval-set.md） | 市场共识「可评测」是 2026 记忆关键词 | 合规（加厚私人记忆） |
| 3 | 办公流水线交付物「成品直出」打磨（对照 MS Agent Mode/WPS 一键成稿体验） | §1：巨头把「少盖章→成品」做成大众可用 | 合规（办公主角位；非计划卡、非新房间） |
| 4 | 价格带数据源观察窗 | 广联达询比价印证需求；gaea 组价已有价格带 | **需拍板**（涉外部数据源，不自行动） |

**明确不提名**（对账删除史）：平台化/MCP 大全/跨设备记忆包（在册排除）；算量/清标/
询比价/企业协同（拍板不做+停手信号表「广联达对照表」）；PM 任务协作层 AI
（停手信号表「项目管理功能对照表」）；GoalCard 式目标代理（v3.6.0 撤下）；
新板块；小说侧脑图/社区类扩张（停手信号）。

## 6. 小说（闲庭外环）

**市场事实**
- Sudowrite 2026 共识卖点：**Story Bible 一致性**（人物/设定/lore 长篇不漂移）+
  Draft 全章生成 + 自研文学模型 **Muse**；用户抱怨风格定制少——风格可定制是
  Novelcrafter 等对手的差异点。

**对 gaea 的含义**
- Story Bible ≈ gaea 角色库+项目工作副本+快照语义（阶段四已收官），方向被验证；
  「风格可定制」= gaea AI 味词表外置（v4.225 words.json）的同一问题域，已是数据资产，
  无需新机制。
- Muse 提示「专用文学微调小模型」路线存在；gaea 模型中心已支持本地模型，留观察不立项。

## 7. AI 陪伴（闲庭）

**市场事实**
- 第三方估计分歧大但方向一致：AI companion 市场 2026 年 $48B（Grand View 窄口径
  ~$9B）→ 2033 年 $318B 预测，CAGR 高两位数；Character.AI 2025 年收入 ~$50M
  （+66%），MAU 2000万~4500万口径不一，日均交互 1.85 亿+。
- 监管/伦理关注升温（情感依赖、未成年人保护，Ada Lovelace Institute 等）。

**对 gaea 的含义**
- 高粘性赛道验证闲庭作为 gaea 差异化粘性来源的价值；gaea 口径是「闲庭陪聊不写进
  办公 AGENTS」「轻语主动开口极稀」——与市场重度沉浸路线不同，**维持口径**。
- 监管趋势值得产品口径知悉（情感依赖边界），不动架构。

## 8. 本地优先 AI（gaea 的底层定位）

**市场事实**
- 2026 本地生态分层成熟：llama.cpp（运行时）→ Ollama（runner）→ LM Studio/Jan
  （桌面）→ agentic 工作台；**「本地默认/隐私」已是基线不再是差异点**。
- **LM Studio 发布 Bionic**：本地/开源模型直接产出文档、幻灯片、PDF、软件——
  本地 agent 工作台正面进入 gaea 的席位。

**对 gaea 的含义（战略结论）**
1. 「本地+隐私」单独不再是卖点，gaea 的护城河必须压在
   **真编辑三件套 × 私人记忆 × 专业域（造价/进度）× 闲庭粘性** 的组合上——
   这个组合在检索面上无人重合。
2. LM Studio Bionic 是最直接的竞品信号（本地 agent 出办公成品），办公交付体验
   （§5 候选3）因此从「锦上添花」升级为「防御性优先级」。

---

## 来源

- [Copilot agentic capabilities GA（MS 365 Blog 2026-04-22）](https://www.microsoft.com/en-us/microsoft-365/blog/2026/04/22/copilots-agentic-capabilities-in-word-excel-and-powerpoint-are-generally-available/)
- [Agent Mode & Office Agent 发布（2025-09-29）](https://www.microsoft.com/en-us/microsoft-365/blog/2025/09/29/vibe-working-introducing-agent-mode-and-office-agent-in-microsoft-365-copilot/)
- [What's New in M365 Copilot 2026-07](https://techcommunity.microsoft.com/blog/microsoft-copilot-blog/what%E2%80%99s-new-in-microsoft-365-copilot--july-2026/4538332)
- [WPS AI 2.0 / 文档工具横评](https://www.wps.cn/article/SjedqsTp.html)
- [Mem0 vs Letta vs Zep 对比](https://www.digitalapplied.com/blog/open-source-agent-memory-mem0-letta-zep-compared)
- [AI Agent Memory 六维对比（2026-05）](https://medium.com/@wasowski.jarek/i-compared-5-ai-agent-memory-systems-across-6-dimensions-none-wins-6a658335ed0a)
- [AI Memory Systems 2026（decay/offline/MCP 维度）](https://www.getfeather.store/theory/ai-memory-systems-comparison-2026)
- [广联达 DATA+AI 解决方案](https://m.glodon.com/mobile/solution/detail/175)
- [AI PM 工具 2026（ClickUp Blog）](https://clickup.com/blog/ai-project-management-tools/)
- [Manus vs Genspark 架构分析](https://www.cnblogs.com/cloudrivers/p/19622133)、[四大智能体复杂任务对比](https://zhuanlan.zhihu.com/p/1913273897475901249)
- [Sudowrite 2026 创作工具对比](https://sudowrite.com/blog/best-ai-for-creative-writing-in-2026-tested-and-compared/)
- [AI Companion Market（Grand View）](https://www.grandviewresearch.com/industry-analysis/ai-companion-market-report)
- [10 Best Local AI Assistants 2026（Vellum）](https://www.vellum.ai/blog/best-local-ai-assistants)
- [LM Studio（Bionic）](https://lmstudio.ai/)
