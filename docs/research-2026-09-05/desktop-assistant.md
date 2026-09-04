# 桌面/自托管 AI 助手竞品调研（2026-09-05）

调研方式：逐个访问 GitHub 仓库 README 与 Releases 页面（WebFetch），辅以站内搜索核实 LobeHub IM 渠道。数据为当日快照，星数为约数。

## 总览

| 项目 | 星数 | License | 最近 release | 形态 | 办公文档 | MCP | 微信/IM |
|---|---|---|---|---|---|---|---|
| lobehub/lobe-chat | ~82.2k | LobeHub Community（自定义） | v2.2.16（09-04） | 自托管 Web，桌面 Canary 中 | 未核实（README 无 docx/xlsx 编辑） | 是（1 万+插件/MCP） | 是（官方 Channels 含微信） |
| CherryHQ/cherry-studio | ~51.4k | AGPL-3.0 + 商业版 | v2.0.12（09-04） | 桌面 Win/Mac/Linux | 仅解析 Office/PDF，无真编辑 | 是（MCP 市场规划中） | 无 |
| chatboxai/chatbox | ~41.6k | GPLv3 | v1.23.1（09-02） | 桌面+移动+Web（社区版/Pro） | 无文档能力提及 | 社区版 README 未提及 | 无 |
| open-webui/open-webui | ~151k | Open WebUI License（保品牌条款） | v0.11.3（08-31） | 自托管 Web（pip/Docker/K8s） | 强 RAG 解析，无原地编辑 | 是（MCP/MCPO/OpenAPI） | 无（伴随 agent 走 Telegram/WhatsApp） |
| janhq/jan | ~44.3k | Apache-2.0 | v0.8.4（07-23） | 桌面（Tauri，本地 llama.cpp） | 未提及 | 是 | 无 |
| Mintplex-Labs/anything-llm | ~65.6k | MIT | 未核实（本次未查 release 页） | 桌面+Docker 自托管+云模板 | 解析 DOCX/PDF 入 RAG，非编辑 | 是 | 无 |

## 分项记录

### 1. lobehub/lobe-chat（已演进为 lobehub/lobehub）
- 链接：https://github.com/lobehub/lobe-chat ｜ ~82.2k stars
- 定位已从"聊天框架"升级为"AI Agent 平台/首席智能体指挥官"：Agent Builder、Agent Groups、Pages 多智能体协作写作、定时任务、Projects、个人记忆（白盒结构化记忆）。
- 形态：自托管 Web 为主（Vercel/Docker 一键部署），官方托管 app.lobehub.com；release 出现 Desktop Canary v2.2.17-canary.1，桌面版在灰度。License 为自定义社区许可证，商用需审条款。
- 办公文档：README 未覆盖文件上传/解析，更无 docx/xlsx 编辑——未核实到任何办公编辑能力。
- MCP：宣称 1 万+ 工具与 MCP 兼容插件，配套 plugin index/gateway。
- IM：官方 Channels 文档列出 Discord、Slack、Telegram、LINE、QQ、微信、飞书/Lark——六者中唯一明确支持微信渠道的。
- gaea 相对位置：LobeHub 赢在生态广度与 agent 编排、且微信渠道已成其标配，但其 agent 停留在"对话/调度"层，没有 Office 文件真编辑；gaea 的"框选即改+直编落盘"在它的能力面上不存在，但微信入口的独占性已不成立。

### 2. CherryHQ/cherry-studio
- 链接：https://github.com/CherryHQ/cherry-studio ｜ ~51.4k stars ｜ AGPL-3.0（社区版）+ 商业授权
- 桌面客户端（Win/Mac/Linux），300+ 预置助手、多模型同聊、知识库（企业版强化）。最近 release v2.0.12（2026-09-04），迭代极快。
- 办公文档：支持 Text/图片/Office/PDF 的解析处理，路线图有文档预处理、OCR、笔记；无任何 docx/xlsx 真编辑声明。
- MCP：已支持，MCP Marketplace 在路线图。模型广度：云端大厂+Ollama/LM Studio 本地。微信：无集成（仅 QQ 群/Telegram 社区；小程序支持在路线图，属客户端形态而非 IM 机器人）。
- gaea 相对位置：cherry 是国内用户心智最强的"桌面多模型工作台"，活跃度和完成度都高，但它是"聊天+知识库"范式，gaea 以真编辑+微信助手切入的是它没占的"产出与送达"位。

### 3. chatboxai/chatbox
- 链接：https://github.com/chatboxai/chatbox ｜ ~41.6k stars ｜ GPLv3（社区版，另有闭源 Pro）
- 全平台（桌面三端+iOS/Android+Web），主打轻量好用的聊天客户端：Markdown/LaTeX、DALL-E 出图、提示词库、API 资源团队共享。最近 release v1.23.1（2026-09-02）。
- 办公文档：README 无文档上传/解析/编辑能力。MCP：社区版 README 未提及。微信：无。
- gaea 相对位置：重叠最小、威胁最低；其"全家桶全平台+双版本商业化"说明桌面 AI 客户端有付费空间，但其功能纵深浅，不构成办公场景对手。

### 4. open-webui/open-webui
- 链接：https://github.com/open-webui/open-webui ｜ ~151k stars（六者最高）｜ Open WebUI License（要求保留品牌，非标准开源，二次分发需谨慎）
- 自托管 Web 平台（pip/Docker/Compose/K8s，可完全离线）。最近 release v0.11.3（2026-08-31），日更级节奏。
- 文档能力：检索侧极强——9 种向量库、Tika/Docling/Mistral OCR/PaddleOCR 抽取、BM25+向量混合检索、全网搜索 RAG；Notes 富文本编辑器可让 AI 改写选中文本。但结论明确：无 docx/xlsx 原地编辑，是抽取/检索范式。
- MCP：支持 MCP、MCPO、OpenAPI 工具服务器；Models & Agents 可绑定指令+工具+知识并分权。多模型同聊+竞技场评测。微信：无（伴随产品 Open WebUI Computer 可从 Telegram/WhatsApp 唤起）。
- gaea 相对位置：生态与工程化标杆，但它是"服务端 RAG 问答"，不碰文件成品；gaea 面向"交付一份改好的文档"，赛道正交。

### 5. janhq/jan
- 链接：https://github.com/janhq/jan ｜ ~44.3k stars ｜ Apache-2.0（六者中唯一标准宽松许可+全离线）
- 桌面（Tauri），llama.cpp 本地推理（Vulkan/Metal/CUDA/ROCm）+ 云端 Provider，本地 OpenAI 兼容 API（localhost:1337）。最近 release v0.8.4（2026-07-23）：OpenAI 兼容翻译网关、Responses API、原生 web search。
- MCP：核心特性（agentic 能力）。办公文档：README 未提及任何解析/编辑。微信：无。
- gaea 相对位置：jan 卡位"本地隐私模型运行时"，与办公秘书场景几乎不冲突；其 Apache-2.0 + 离线卖点提示 gaea 可强调"数据不出本机"的办公合规叙事。

### 6. 补充：Mintplex-Labs/anything-llm
- 链接：https://github.com/Mintplex-Labs/anything-llm ｜ ~65.6k stars ｜ MIT
- 桌面+Docker 自托管+云模板三形态；Agent Flows 无代码 agent 编排、定时任务、技能智能选择、动态模型路由、持久记忆；"Open Computer"在研。文档仅解析嵌入（PDF/TXT/DOCX）入 RAG，非编辑。MCP：是。微信：无。
- gaea 相对位置：anything-llm 是"私有 ChatGPT+doc chat"，agent 自动化走得快但同样不产出 Office 成品；release 信息本次未查，未核实。

## 综合观察：gaea 的差异化机会

1. **Office 真编辑是全场空白，且应坚持"换壳不换芯"**：六家无一具备 docx 框选即改/xlsx 直编——全部停在"上传解析→RAG→文字回答"。这是 gaea 最硬的差异化，机会在于把它做成可被感知的招牌场景（框选→改→落盘回写原文档），营销对标"AI 改文档"而非"AI 聊天"；实现上继续以真实 Office 文档引擎为核、AI 只做外壳与指令层，不做自研文档内核。
2. **微信入口独占性已破，需换卖点**：LobeHub Channels 官方支持微信/飞书/QQ 等（生态里还有 dsh-im-gateway、OpenClaw 等聚合网关），"能接微信"本身不再是壁垒。gaea 微信助手的卖点应升级为"微信里收到→gaea 改好一份 docx/xlsx 回传"的闭环交付，即 IM 作为 Office 编辑的入口与出口，而非单纯聊天机器人。
3. **"交付物可观测"对打"黑盒 agent"**：各家竞相堆 agent 编排（Agent Flows、Agent Groups、定时任务），但产出是聊天文本。gaea 的上下文可观测/证据链能力可包装为"每处修改可溯源、过程可见"，正好回应办公用户对 AI 改稿最大的顾虑（改了什么、为什么改、原文是否被破坏）。
4. **模型广度是别人的主场，不必正面追**：竞品普遍 10+ 云厂商+Ollama 本地。gaea 的多模型管理做到"够用、可配、可切换"即可，把工程资源押在文档链路与微信闭环上；同时可借 jan 的启示补一条"本地模型/数据不出本机"的合规叙事。
5. **注意许可证风险差异**：open-webui（保品牌条款）、lobe-chat（自定义社区许可）、cherry（AGPL）都不适合直接抄代码/换皮复用；gaea 自研路线反而规避了这类传染/品牌约束，可作为对外表述的合规优势。

## 信息来源
- 各仓库 README/Releases：github.com/lobehub/lobe-chat、CherryHQ/cherry-studio、chatboxai/chatbox、open-webui/open-webui、janhq/jan、Mintplex-Labs/anything-llm
- LobeHub Channels 渠道文档：lobehub.com/zh/docs/usage/channels/overview（含微信/飞书/QQ 等支持列表）
