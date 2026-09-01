# 模块 3 · AI 底座：记忆与知识库

## 3. AI 底座 · 记忆与知识库（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

开源记忆基础设施在 2026 年完成「从向量库到记忆 OS」的跃迁，梯队清晰：

- **Mem0**（64.4k stars）2026-04 发布新记忆算法：单次 ADD-only 提取（无 UPDATE/DELETE）、实体链接、多信号检索（语义 + BM25 + 实体）、时间推理；LoCoMo 92.5 / LongMemEval 94.4，迁移指南显示开源版进入 v3 [1]。BM25+语义混合检索已成行业标配。
- **MemOS 2.0「星尘」**（11.1k stars）：L1 轨迹 / L2 策略 / L3 世界模型三层记忆，MemCube 可组合记忆立方体（支持隔离与受控共享），MemScheduler 异步调度；2026-05 memos-local-plugin 2.0 实现 100% 本地 SQLite，2026-08-17 接入 DeepSeek Harness。README 未提及遗忘机制，最接近的是自然语言「记忆反馈与修正」[2]。
- **Letta（MemGPT）**大改版：主仓库转为落地页，活跃开发移至 letta-code（agent harness + 桌面应用），Letta Cloud 提供跨设备同步 agent 记忆/身份/会话；记忆固化为可被 agent 用工具编辑、可跨 agent 挂载的 memory blocks [3][4]。
- **Zep Graphiti**（30.4k stars）：双时态知识图谱 + 自动事实失效，语义+BM25+图遍历混合检索，v0.17 自定义图驱动、Kuzu 弃用、嵌入式 FalkorDB Lite 降低部署门槛，仅自托管 [5]。
- **大厂原生记忆**：ChatGPT Memory 为「显式记住指定信息 + 回忆过往对话」双开关（后者 2025-04 上线），2025-09 起 ChatGPT Pulse 依据聊天历史生成晨报 [7]；Gemini 2025-02 向 Advanced 订阅者开放回忆过往对话，2026-03 Pixel Drop 推出 Gemini 代办与 Magic Cue 等主动个性化功能，2026-05 发布 Gemini Spark 代理 [8][9]；Claude 走客户端文件型 memory tool（/memories 目录，view/create/str_replace/insert/delete/rename 六命令，存储归应用方，可跨会话并与 compaction 组合）[6]。ChatGPT/Gemini 记忆在 2026 年的专项升级细节：**未核实**（官方帮助页不可访问、搜索引擎反爬）。
- **本地知识库**：RAGFlow（89.7k stars）2025-12 给 agent 加 Memory 组件、2026-06 支持飞书/Discord/Telegram 等渠道 [11]；AnythingLLM（65.4k stars）上线智能技能选择（宣称省 80% token）、MCP 支持，桌面版内置 LanceDB+本地嵌入的零配置 RAG [12]。

### 范式迁移（上轮调研以来的变化）

1. **记忆架构化**：提取、固化、调度、检索成为独立工程层（Mem0 算法层、MemOS L1-L3+调度器、Graphiti 双时态图），「存下对话摘要」不再是记忆产品。
2. **遗忘被重新定义为「失信 + 复核」而非删除**：LangChain 的 OpenWiki 自纠错记忆（2026-08-25）用「主张 ↔ 代码证据版本」锚点做确定性过期检测，过期主张标记保留、验证后修正或刷新，形成自纠错回路 [13]；Graphiti 的自动事实失效同思路。
3. **GraphRAG 降温、agentic RAG 成为默认**：Microsoft GraphRAG 官宣进入维护模式、不再接受新功能 PR [10]；RAGFlow 以 agentic workflow 为主线并内置 Memory。重型批量图谱构建让位于轻量动态图 + agent 按需检索。
4. **记忆所有权/可移植性成为行业叙事**：Harrison Chase《Own your intelligence》（2026-07-25）明确提出「不拥有 context 与记忆，就不拥有积累的智能」，要求学习成果可跨模型、跨系统迁移 [14]；Mem0 已为 Claude Code/Cursor/Codex 提供「让 harness 记住你的项目」插件 [1][15]。
5. **主动式记忆从概念到产品**：ChatGPT Pulse、Gemini Spark 均以「记忆 + 历史」主动送达，记忆竞争从「记得准」转向「主动给」。

### 对 gaea 的机会与威胁

**威胁**：
- 大厂把「记得你 + 主动服务」变成默认体验，用户预期水涨船高；gaea 做梦 2.0 的主动预取尚未做，正好卡在这条新预期线上。
- MemOS 本地插件 2.0（100% 本地 + 三层记忆 + 智能去重 + Memory Viewer）与 gaea 三脑叙事正面重叠，且有论文与基准背书；MemCube 的「隔离 + 受控共享」是「记忆分区」最强公开先例——gaea「双空间」独创性被削弱。但注意：各家分区均按 user/session/agent 边界（Mem0 三级、Letta shared blocks、MemCube），未见按「生活领域」（工作 vs 角色人格）分区且互不检索的先例，gaea 的隔离语义仍独特。
- LoCoMo/LongMemEval 已成通用记分牌，gaea 无公开可复现指标，宣传缺第三方背书。

**机会**：
- 「记忆所有权」叙事正热，gaea 天然本地优先，可把「可查看、可删除、可导出迁移」做成对大厂云记忆的差异化卖点（个人版「遗忘权」）。
- 开源方案多需 Neo4j/Qdrant 等服务栈（MemOS docker compose、Graphiti），gaea 的单机 SQLite + BM25+语义零配置路线在 Windows 个人机上部署门槛最低。
- GraphRAG 进维护模式印证重型图谱不适合个人机；gaea 宜做实体链接 + 时态失效的「轻图谱」，而非全文档社区摘要。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

**0-3 月**：
- 做梦 2.0（DistillMerge）补齐冲突消解语义：借鉴 OpenWiki 的「主张-证据」思路，两条记忆冲突时并存并标记待复核，而非静默覆盖；蒸馏结果按 Mem0 新算法的 LoCoMo/LongMemEval 口径跑一组公开可复现评测。
- 上线「主动预取」MVP：开机/空闲时把高频工作记忆预装配进上下文预算（对标 Pulse 晨报形态，但纯本地）。
- 记忆管理面板补全：单条查看/编辑/删除 + 全量导出（JSON/Markdown），把「遗忘权」落成功能。

**下个 3-6 月**：
- 实体链接 + 轻时态失效（借鉴 Graphiti 双时态）：知识库与记忆分区实体互链，过期事实自动标记 stale。
- fileindex 与记忆检索合流：个人文件与对话记忆走统一混合检索，context 预算调度从雏形升级为显式「召回→评分→预算装配」管线。
- 双空间引入 MemCube 式「受控共享」白名单（如工作空间可引用知识分区、角色空间不可），把隔离从设计主张升级为可验证机制。

**愿景 6-12 月**：
- 记忆可移植性：gaea 记忆格式可导出（对齐 Mem0/JSON-LD）、跨机迁移、支持导入大厂商聊天导出，呼应「own your memory」叙事。
- 对齐 MemOS 的 L2 策略记忆（个人工作流偏好）与 L3 世界模型（个人知识图谱）；本地小模型驱动的后台做梦升级为「睡眠时固化」（sleep-time 式反思 + 主动预取联动）。

### 参考来源

1. Mem0 GitHub（新算法 2026-04、基准、v3）：https://github.com/mem0ai/mem0
2. MemOS GitHub（2.0 星尘、L1-L3、MemCube、本地插件）：https://github.com/MemTensor/MemOS
3. Letta GitHub（letta-code、Letta Cloud 同步）：https://github.com/letta-ai/letta
4. Letta 文档 · Memory Blocks：https://docs.letta.com/guides/agents/memory
5. Zep Graphiti GitHub（双时态图、事实失效、FalkorDB Lite）：https://github.com/getzep/graphiti
6. Anthropic memory tool 文档：https://platform.claude.com/docs/en/agents-and-tools/tool-use/memory-tool
7. Wikipedia · ChatGPT（Memory 双开关、Pulse 晨报）：https://en.wikipedia.org/wiki/ChatGPT
8. Wikipedia · Gemini（回忆过往对话、Spark 代理）：https://en.wikipedia.org/wiki/Gemini_(chatbot)
9. Google Blog · 2026-03 Pixel Drop（Gemini 代办/Magic Cue）：https://blog.google/products-and-platforms/devices/pixel/march-2026-pixel-drop/
10. Microsoft GraphRAG GitHub（维护模式声明）：https://github.com/microsoft/graphrag
11. RAGFlow GitHub（Memory 组件、渠道接入）：https://github.com/infiniflow/ragflow
12. AnythingLLM GitHub（技能选择、MCP、桌面本地 RAG）：https://github.com/Mintplex-Labs/anything-llm
13. LangChain Blog · Building Self-Correcting Memory in OpenWiki（2026-08-25）：https://www.langchain.com/blog/self-correcting-memory-openwiki
14. LangChain Blog · Own your intelligence（2026-07-25）：https://www.langchain.com/blog/own-your-intelligence
15. Mem0 文档（Claude Code/Cursor/Codex 插件、自托管）：https://docs.mem0.ai
