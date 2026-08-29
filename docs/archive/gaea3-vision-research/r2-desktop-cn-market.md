# gaea 3.0 愿景规划 — 桌面中文市场调研报告（r2）

> 调研对象：gaea（Wails 桌面 AI 助手，Go 后端 + React 前端，单用户本机；模块：聊天 / 轻语人格陪伴 / 小说创作 / 绘梦 / 办公 agent / 记忆中枢 / 模型中心；用户为中文个人用户）
> 调研时间：本次调研基于 2025—2026 年公开信息检索
> 调研方式：web_search 中英文混合检索，结论均附来源 URL；用户量级等数据以公开报道为准，标注年份

---

## 0. 摘要（先读这节）

1. **桌面 AI 助手是"薄壳客户端"与"本地运行器"两个物种**。Jan（本地优先）、LM Studio（本地推理 GUI）、AnythingLLM（本地 RAG）负责"本地"，Chatbox、Cherry Studio 负责"多模型客户端"，Enconvo 是 macOS 启动器形态。它们普遍不提供"单机智能体"能力，插件正在被 MCP 统一。gaea 的差异化空间在于：把这几类形态**揉进一个单用户本机应用**，而不是再做一个聊天壳。
2. **中文小说写作 AI 处于"平台收紧、工具增多"的夹缝**。彩云小梦（续写）、阅文妙笔（网文平台内嵌）、笔灵（工具站）、Lattics（长文知识库写作）并存；但晋江 2025 年 2 月打响"反 AI 创作"第一枪，网文平台对 AI 内容态度复杂。对 gaea 的意义：小说模块应面向**个人创作而非发表**，避开平台红线。
3. **中文 AI 陪伴是"上亿用户、不赚钱、监管收紧"的高风险高流量赛道**。星野（MiniMax）海外收入占比 73%，仍净亏 5.12 亿美元（2025 前三季）；2025 年底情感陪伴类 AI 迎来监管新规，星野下架大量智能体。数字生命/复活逝者是独立小赛道（硅基智能等），伦理与合规风险高。
4. **记忆产品分四条技术路线**：向量检索记忆（Mem0，2025 年获 2400 万美元 A 轮）、上下文状态管理（Letta，前 MemGPT）、时序知识图谱（Zep）、本地文件渐进披露（Napkin）。本地 RAG 助手（AnythingLLM、腾讯 ima 等）已普及"工作区/知识库 + 溯源"形态。gaea 记忆中枢可借鉴 Mem0/Napkin 的组合。
5. **本地跑 agent 的现实：推理成熟、agent 化不成熟**。Ollama/LM Studio/llama.cpp 做推理已经稳定，但本地小模型的工具调用（tool calling）是公认痛点：流式响应破坏工具调用、模型幻觉 XML 标签、小模型在工具任务上卡死。真能跑通的本地 agent 形态主要是：OpenClaw+Docker+Ollama、Dify+Ollama 工作流、Claude Desktop+MCP 式"云端大脑+本地工具"。**结论：个人用户本地 agent 应默认"本地推理 + 云端智能"混合，而非全本地**。

---

## 1. 本地 / 桌面 AI 助手市场

### 1.1 产品全景与定位

桌面 AI 客户端市场在 2024—2026 年快速分化，按形态可分为四类：

| 形态 | 代表产品 | 核心定位 | 能力边界 |
|---|---|---|---|
| 多模型聊天客户端 | Chatbox、Cherry Studio | 薄壳客户端，接各家 API | 聊天/提示词/部分 RAG，无本地推理 |
| 本地优先客户端 | Jan | 100% 离线跑开源模型 | 模型管理+聊天，生态不完整 |
| 本地推理运行器 | LM Studio、Ollama | 本地模型 GUI/守护进程 | 推理+OpenAI 兼容 API，无应用层 |
| 本地知识库应用 | AnythingLLM、Open WebUI | 私有文档 RAG 问答 | 工作区/文档问答，非通用助手 |
| 启动器/效率工具 | Enconvo | macOS 全局唤起 AI 工作流 | 插件化快捷操作，非聊天主体 |

### 1.2 各产品要点

**Chatbox（开源多模型客户端）**
- 官方自述 "Powerful AI Client"，跨平台桌面应用（Windows/macOS/Linux），主打"接入多个前沿 LLM 模型"的通用聊天客户端（GitHub: https://github.com/GZC888/chatbox ；中文 README: https://github.com/mz247/chatbox/blob/main/README-CN.md ）。
- 定位是**轻量聊天壳**：多 provider（OpenAI/Claude/Gemini/DeepSeek/Ollama 本地端点）、会话管理、提示词库，不做知识库/工作区/插件生态。AI 原生全景图收录：https://landscape.jimmysong.io/projects/chatbox/ 。

**Cherry Studio（国产开源多模型客户端）**
- 官网自述 "desktop client that supports for multiple LLM providers"，GitHub 仓库：https://github.com/CherryHQ/cherry-studio ；中文官网：https://cherrystudiocn.com/index.html ；文档站：https://docs.cherryai.com.cn/ 。
- 相对 Chatbox 更进一步：内置**预置应用/助手商店**（预设提示词应用：翻译、写作、程序员等，见 https://docs.cherryai.com.cn/docs/zhong-wen-fan-ti/cherry-studio/preview#yu-she-ying-yong ）、**知识库（RAG）**、联网搜索、与硅基流动 SiliconFlow 等国产平台的深度对接（https://docs.siliconflow.cn/en/usercases/use-siliconcloud-in-cherry-studio ）。社区文章称其为"从个人助手到企业级解决方案"（腾讯云开发者社区: https://cloud.tencent.com/developer/article/2617735 ）。
- 在国内以"DeepSeek 最佳客户端"心智出圈（百度智能云文章做 Cherry Studio 与 ChatBox 对比: https://cloud.baidu.com/article/3931431 ）。

**Jan（本地优先开源客户端，jan.ai）**
- 官方定位 "an open source alternative to ChatGPT that runs 100% offline on your computer"，多引擎支持（llama.cpp、TensorRT-LLM），OpenAI 兼容端点，也允许连接云端 provider（GitHub: https://github.com/Innovation-Labs-Technical-Hub/jan-app ；文档: https://www.jan.ai/docs/desktop/quickstart ）。
- 用户心智：中文社区称"20.5K 星星！将你的电脑变成 AI 计算机"（腾讯云: https://cloud.tencent.cn/developer/article/2472803 ）；另一篇称 39.2k Star（CSDN: https://blog.csdn.net/feikillyou/article/details/155708100 ）。
- 能力边界：**它是"本地模型管理器 + 聊天界面"，不是"本地智能体"**。ItsFoss 实测后结论是"I Went Back to Ollama"——作为 ChatGPT 替代品在生态完整性上不敌（https://itsfoss.com/jan-ai/ ）。Open WebUI 官方文档也对 Jan 与 Open WebUI 的取舍做了对比（https://docs.openwebui.com/alternatives/jan/ ）。

**LM Studio（本地推理 GUI）**
- 定位：在桌面上运行 GGUF 开源模型的图形界面 + OpenAI 兼容本地 API server。版本迭代可见其能力：0.3.14 加入多 GPU 控制（https://beta.lmstudio.ai/blog/lmstudio-v0.3.14 ）、0.3.15 支持 RTX 50 系列并改进 API 工具调用（https://beta.lmstudio.ai/blog/lmstudio-v0.3.15 ）、0.3.19（https://beta.lmstudio.ai/blog/lmstudio-v0.3.19 ）。
- 用户心智：本地推理的"标准件"，常与 Open WebUI、AnythingLLM、开发工具组合使用；skywork 的 2025 对比文章把 LM Studio 列入主流桌面 AI 工具（https://skywork.ai/blog/ai-agent/claude-desktop-vs-chatgpt-perplexity-copilot-lm-studio-2025-comparison/ ）。
- 能力边界：**推理运行器，非应用平台**——不提供知识库、工作区、agent 编排；工具调用仅通过 API 暴露给外部程序。

**AnythingLLM（本地知识库/RAG 应用）**
- 定位：本地私有化 RAG 知识库，核心概念是 **workspace（工作区）**——每个工作区挂不同文档集与模型，文档问答可溯源（官方文档: https://docs.anythingllm.com/chatting-with-documents/introduction ；RAG 原理说明: https://docs.anythingllm.com/chatting-with-documents/rag-in-anythingllm ）。v1.8.5 迭代中，另有移动端（https://docs.anythingllm.com/mobile/overview ）。
- 中文社区大量"Ollama+AnythingLLM 本地知识库搭建"教程（百度云: https://cloud.baidu.com/article/3676944 、https://cloud.baidu.com/article/3891916 ；DataCamp: https://www.datacamp.com/zh/blog/anythingllm ），说明它是**国内个人用户搭建本地知识库的默认选择之一**。
- 能力边界：**文档问答工具，不是助手**——没有主动记忆、没有跨模块调度。

**Enconvo（macOS AI 启动器）**
- 定位：macOS 上的 **AI Agent Launcher**，150+ 插件、情境自动化、全局快捷键唤起，类 Raycast 的效率工具形态（官方文档: https://docs.enconvo.com/docs/intro ；介绍页: https://www.toolmage.com/en/tool/enconvo/ ；中文介绍: https://pidoutv.com/sites/33511.html ）。
- 能力边界：**系统级快捷操作**（唤起、摘录、翻译、总结等），不承载长对话与人格，是"桌面 AI 效率层"而非"桌面 AI 助手"。

**Open WebUI（开源 Web UI）**
- 开源 Web 界面，支持 Ollama/OpenAI API 等，功能密度高（聊天、RAG、多用户、工具）。XDA 评测称"two months of updates… I'd pick it over ChatGPT's interface for local LLMs"（https://www.XDA-Developers.com/after-two-months-open-webui-updates-pick-chatgpt-interface-local-llms/ ）。dev.to 有 5 款本地 AI UI 对比（Open WebUI / LM Studio / SillyTavern / Jan / 自研）：https://dev.to/purpledoubled/i-compared-5-local-ai-uis-open-webui-lm-studio-sillytavern-jan-and-my-own-1453 。

### 1.3 插件 / 工作区形态观察

- **MCP（Model Context Protocol）已成为桌面客户端插件事实标准**：Cherry Studio 等国产客户端与 Claude Desktop 均支持 MCP；中文社区已有"MCP 客户端盘点"类文章（CSDN: https://adg.csdn.net/694cf0d35b9f5f31781a9446.html ）。gaea 3.0 的插件层应直接兼容 MCP，而不是自造协议。
- 工作区形态分两种：**知识工作区**（AnythingLLM 的 workspace：文档集+模型绑定+问答溯源）与**助手商店**（Cherry Studio 的预置应用：提示词包形式的垂直助手）。gaea 的记忆中枢可借鉴前者，模型中心可借鉴后者。

### 1.4 对 gaea 的启示

- 市场已有：聊天壳（Chatbox）、本地运行器（LM Studio/Ollama）、本地 RAG（AnythingLLM）、启动器（Enconvo）。**缺的是"单机、单用户、跨模块"的桌面智能体**：一个应用里同时有聊天、陪伴人格、写作、绘图、办公 agent、记忆，且全部本地数据。这正是 gaea 的定位空隙。
- 模型中心应同时面向"本地模型（Ollama/LM Studio 兼容端点）"与"云端 API"，且默认混合调度——与第 5 节的现实结论一致。

---

### 1.5 桌面助手功能矩阵（2025—2026 实测能力对比）

| 能力维度 | Chatbox | Cherry Studio | Jan | LM Studio | AnythingLLM | Open WebUI | Enconvo |
|---|---|---|---|---|---|---|---|
| 本地模型接入 | 可接 Ollama 端点 | 可接本地端点 | 内置引擎 | 原生 | 原生(Ollama) | 原生 | 依赖外部 |
| 多云端 API | ✅ | ✅(DeepSeek/硅基流动等) | ✅ | ❌ | ✅ | ✅ | ✅ |
| 知识库/RAG | ❌ | ✅ | ❌ | ❌ | ✅(核心) | ✅ | ❌ |
| 插件/扩展 | ❌ | ✅(预置应用+MCP) | ❌ | ❌ | ❌ | ✅(工具/管道) | ✅(150+插件) |
| 会话/人格管理 | 基本 | 助理角色 | 基本 | ❌ | 基本 | 多用户 | ❌ |
| 系统级操作(快捷键/自动化) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅(核心) |
| 主动 agent/调度 | ❌ | 部分(知识库问答) | ❌ | ❌ | 部分(文档任务) | 部分 | 部分(工作流) |

**要点**：没有任何一款产品同时覆盖"本地推理 + 多模型 + 记忆 + 插件 + 系统操作"。gaea 的模块化单机形态在功能矩阵上天然位于空白区，但代价是每一项都要自研或对接成熟组件（Ollama 端点、MCP、sqlite-vec 等）。

### 1.6 国产桌面客户端的用户心智差异

- Cherry Studio 在国内的传播点是"**DeepSeek 客户端**"（百度智能云测评文章以此为卖点: https://cloud.baidu.com/article/3931431 ）与"**免费接入多家国产模型**"，用户是"多模型白嫖党/换模型党"。
- Chatbox 的传播点是"**轻量、跨平台、开箱即用**"，用户是"单一或少量模型的重度聊天用户"。
- Jan/LM Studio 的用户是"**隐私敏感 + 折腾党**"（技术型个人用户），心智是"我的模型我做主"，对 GUI 完成度容忍度高。
- 结论：中文个人用户选客户端的第一理由是"模型接入便利 + 免费"，第二才是"隐私"。gaea 主打"单机智能体"时，必须在模型中心把"接自家 API + 接国产模型 + 接本地模型"做成零门槛，否则会被当"又一个 Cherry Studio"。

---

## 2. 中文小说写作 AI 市场

### 2.1 产品形态与用户心智

**彩云小梦（彩云科技，北京彩彻区明科技）**
- 中文 AI 小说续写 App 的鼻祖（2019 年前后上线，iOS 应用页: https://apps.apple.com/cn/story/id1585749129 ；百度百科: https://baike.baidu.com/item/%E5%BD%A9%E4%BA%91%E5%B0%8F%E6%A2%A6/59403428 ）。核心形态：**用户写开头，AI 续写/接龙**，支持故事设定、多分支；用户心智是"卡文救星 / 追更替代"（36 氪 AI 测评两篇: https://36aidianping.com/note-detail/3568010593719853 、https://36aidianping.com/note-detail/3568010593720397 ）。
- 母公司彩云科技 2024 年 11 月推出基于 DCFormer 架构的通用大模型（新华网: http://www.news.cn/tech/20241113/80d5b68aba9c445da0ad94d5467a308f/c.html ），说明**从应用到模型自研**的路径；App 商店数据可见其收入/下载趋势（AppStoreSpy: https://appstorespy.com/ios-apple-store/1564619616-trends-revenue-statistics-downloads-ratings ）。
- 形态总结：**独立 App + 社区续写**，面向"读者转作者"的轻量创作者。

**阅文妙笔 / 作家助手妙笔版（阅文集团）**
- 2023 年 7 月发布国内首个网文大模型"阅文妙笔"及配套应用"作家助手妙笔版"（中国作家网转载综述: https://wyb.chinawriter.com.cn/Pad/content/202507/04/content79915.html ；百度百科: https://baike.baidu.com/item/%E4%BD%9C%E5%AE%B6%E5%8A%A9%E6%89%8B%E5%A6%99%E7%AC%94%E7%89%88/63227250 ）。
- 2025 年演进：发布"妙笔通鉴""漫剧助手"，AI 赋能网文创作与 IP 改编（网易: https://m.163.com/dy/article/KC3PV3K10514CDBK.html ）；推出行业首个"千万字理解能力"AI 应用，可实时分析网文（搜狐: https://www.sohu.com/a/944943008_100116740 ）；2025 年 2 月网文领域率先部署 DeepSeek（广州日报: https://epaper.gzdaily.cn/news/html/2025/02/15/content_873_879639.htm ）。
- 形态总结：**平台内嵌式写作辅助**（大纲、设定、润色、长文分析），绑定阅文作家生态，面向发表级创作。

**笔灵 AI（ibiling.cn）**
- AI 小说写作工具站：首页 AI 全篇创作、小说工具箱、写作课堂、小说素材库（https://ibiling.cn/novel-navigation/detail/1 ）；有移动端 App（https://sj.qq.com/appdetail/com.ibiling.app ）；百度百科词条: https://baike.baidu.com/item/%E7%AC%94%E7%81%B5/67287703 。
- 形态总结：**模板化写作工具**（人设/大纲/全篇生成），面向网文新手批量生产，行业测评常列名（CSDN 8 款 AI 写小说工具盘点: https://aitool.csdn.net/6a4e1b5010ee7a33f2895e79.html ）。

**Lattics（类脑知识库 + 长文写作）**
- 定位 "Brain-like knowledge base with AI writing & deep research"，macOS/iOS App（Apple App Store: https://apps.apple.com/ao/app/lattics-brain-like-writing/id1575605022?mt=12 ；Product Hunt: https://www.producthunt.com/products/lattics-2 ）。以"类脑"卡片组织写作素材，官方示范学术论文全流程工作流（https://lattics.com/zh-CN/review/lattics-academic-paper-workflow ）。
- 形态总结：**知识库驱动的长文写作**（研究+素材+成文一体），用户心智是"写作工作台"而非"小说生成器"。与 gaea 的"小说创作 + 记忆中枢"组合最接近的现成对标。

**Midreal（海外互动叙事）**
- 硅谷团队（创始人陈锴杰）的生成式互动叙事平台，"choose your own adventure"式 AI 故事（GamesBeat 2024: https://gamesbeat.com/midreal-generative-ai-storytelling/ ；测评: https://aiseekertools.com/tools/midreal ；公司信息: https://www.cbinsights.com/company/midreal ）。中文社区曾讨论其"AI 夺回网文界"叙事（机器之心: https://cloud.tencent.cn/developer/article/2376909 ）。形态：**AI 自主延续 + 玩家选择分支**，是"AI 作者"而非"写作工具"。
- SudoWrite（海外，小说家辅助）
- 面向职业小说家的 AI 写作工具，核心功能 Story Engine（故事引擎）、Describe、Write 续写；官方博客展示高产作者的真实写作栈（https://sudowrite.com/blog/ai-fiction-writing-stack/ ；AI 共写方法论: https://sudowrite.com/blog/co-writing-with-ai/ ）；Reedsy 评测: https://reedsy.com/blog/guide/book-writing-software/sudowrite/ 。形态：**co-writing 工具（人主导、AI 辅助）**，不承诺自动成书。

### 2.2 平台与监管态度（关键背景）

- **晋江 2025-02-17 打响"反 AI 创作"第一枪**：发布《关于 AI 辅助写作使用、判定的试运行公告》，划定 AI 辅助写作的安全线（中国作家网学术综述: http://www.chinawriter.com.cn/n1/2025/0905/c404027-40557712.html ）。
- 网文平台被 AI"重新定义"而非杀死：起点、番茄等平台仍主导分发，但 AI 内容冲击创作生态（DoNews/界面: https://m.jiemian.com/article/13655654.html ；投资界: https://m.pedaily.cn/news/557522 ）。
- 2025 中国网络文学发展研究报告：**作者群体对 AI 态度普遍复杂**（中国社会科学网: https://www.cssn.cn/skgz/bwyc/202604/t20260420_5981165.shtml ）。
- 里程碑事件：2024 年国内首部百万字 AI 小说《天命使徒》（华东师范大学研讨会，见中国作家网综述）；AI 短剧"单篇赚两万"的造富叙事（PConline: https://www.pconline.com.cn/focus/1736/17361632.html ）。
- 结论：**中文网文平台对"AI 创作内容"收紧，但对"AI 辅助工具"暧昧**。gaea 小说模块应定位"个人创作助手"（设定/灵感/润色/世界书），避开"代写发表"心智，这与轻语陪伴的"私人"属性天然一致。

---

### 2.3 写作工具形态对比

| 产品 | 形态 | 用户 | 心智关键词 | 商业模式 |
|---|---|---|---|---|
| 彩云小梦 | App + 社区续写 | 轻量创作者/读者 | 卡文救星、AI 接龙 | 订阅/会员 |
| 作家助手妙笔版 | 网文平台内嵌 | 阅文签约/驻站作者 | 大纲、设定、润色 | 平台生态 |
| 妙笔通鉴/漫剧助手 | 平台级 AI 应用 | 编辑/作者/IP 方 | 千万字分析、IP 改编 | 平台生态 |
| 笔灵 | 工具站 + App | 网文新手/批量写手 | 全篇生成、模板 | 会员/按量 |
| Lattics | 桌面知识库写作 | 长文作者/研究者 | 类脑、素材管理 | 买断/订阅 |
| Midreal | 互动叙事平台 | 英文互动小说玩家 | AI 自主叙事 | 订阅 |
| SudoWrite | 网页工具 | 职业英文小说家 | co-writing、故事引擎 | 订阅($10-30/月) |

### 2.4 用户心智与内容生态要点

- **读者端**：AI 续写被当作"追更替代/同人创作"（彩云小梦测评文章标题即"追更太痛苦了，我还是自己写吧": https://36aidianping.com/note-detail/3568010593719853 ），说明中文用户对"AI 写故事"有真实需求，但期待的是**互动与陪伴式创作**，不是冷冰冰的批量生成。
- **作者端**：分两派——平台签约作者用"妙笔版"式辅助（不触碰红线），独立写手用笔灵式全篇生成（大量内容会撞上平台审核与"反 AI"规则）。
- **平台端**：晋江 2025-02 反 AI 公告（http://www.chinawriter.com.cn/n1/2025/0905/c404027-40557712.html ）之后，主流平台对"AI 全篇生成"内容普遍设限；但"AI 辅助工具"（设定、润色、灵感）仍被默许。gaea 应主打后者。
- **内容标准**：《2025 中国网络文学发展研究报告》指出作者对 AI 态度复杂（https://www.cssn.cn/skgz/bwyc/202604/t20260420_5981165.shtml ）——机会在于"帮作者写得更好"而不是"替作者写"。

---

## 3. 中文 AI 陪伴 / 数字生命趋势

### 3.1 头部产品与量级

**星野（MiniMax 国内）/ Talkie（MiniMax 出海）**
- 星野是国内 AI 陪伴头部 App（用户可创建"智能体/星野"角色聊天）；Talkie 是 MiniMax 出海版，主要营收来源。
- **财务现实（IPO 招股书，2025 年 12 月港交所聆讯）**：2025 年前三季度收入 5343.7 万美元、净亏损 5.12 亿美元、毛利率 23.3%、**海外市场收入占比 73.1%**（21 经济: https://m.21jingji.com/timeline/5201b332b6d10fe98ff18b8a68d0ac36.html ）；现金储备超 11 亿美元（新京报: https://m.bjnews.com.cn/detail/1766325559129059.html ）。
- 用户量级："AI 恋人撬动上亿用户却仍未盈利"，星野母公司 MiniMax 通过港交所聆讯（新快报 2025-12-24: https://ep.ycwb.com/epaper/xkb/h5/html5/2025-12/24/content_1511_736204.htm ）。
- 星野 + 猫箱（字节系 AI 陪伴 App）的合计流量"和 Kimi 一个量级"（钛媒体 2025-12: https://www.tmtpost.com/7392953.html ；转载: https://ai-kit.cn/2419.html ）。
- 星野获米哈游、腾讯投资（钛媒体: https://www.tmtpost.com/7358826.html ）——大厂与游戏公司押注 AI 陪伴。

**赛道降温信号**
- 2025 年 11 月"星野下架大量智能体"，"赛博爱情崩盘"论调出现（21 财经: https://m.21jingji.com/article/20251129/herald/c4cf0b7853b66ba95873b68fb2dee2f2_zaker.html ）。
- 2025 年 12 月"下载量暴跌八成，AI 社交涨不动了"（澎湃: https://m.thepaper.cn/newsDetail_forward_31086524 ）；36 氪称"投放、下载量全面腰斩"（https://m.36kr.com/p/3253625617196293 ）。
- **2025-12-29 情感陪伴类 AI 迎新规**：训练数据要求收紧（21 经济: https://www.21jingji.com/article/20251229/herald/9659b7cd5ba774a951ff08aac51b60c3.html ；南方财经: https://static.nfnews.com/content/202512/29/c12033897.html ）。监管对"拟人化情感陪伴"开始划界。

**Character.AI（海外参照）**
- 2025 年 10 月因青少年死亡诉讼，宣布限制 18 岁以下用户的聊天（Ars Technica: https://arstechnica.com/information-technology/2025/10/after-teen-death-lawsuits-character-ai-will-restrict-chats-for-under-18-users/ ）；CEO 提出 "AI Friends" 愿景（PYMNTS: https://www.pymnts.com/news/artificial-intelligence/2025/character-ai-ceo-envisions-future-with-ai-friends/ ）。用户与收入统计见 Axis Intelligence: https://axis-intelligence.com/character-ai-statistics/ 。

**其他中文陪伴/人格产品**
- 小冰（北京红棉小冰科技）：X Eva 是 AI 克隆人 App（百度百科: https://baike.baidu.com/item/X%20eva/63058852 ）；CEO 李笛主张"AI 必须懂得情感"（南方周末: https://www.infzm.com/contents/285722 ）。2025 年出现"克隆人平台停服"报道，折射商业化困境（南方+ : https://www.nfnews.com/content/K3BDX0e0oY.html ）。
- 大厂/AI 公司竞品：智谱清言、字节猫箱、以及各类"AI 女友"产品；行业观察称 AI 陪伴下载量下滑但粘性（日聊时长）高（艾瑞: https://news.iresearch.cn/content/202508/531576.shtml ——"AI 女友赛道半年吸金超 5 亿，头部玩家用户日聊 75 分钟"）。

### 3.2 数字生命 / 永生话题的产品化现状

- **AI 复活逝者已成真实小生意**：2025 年清明节期间中外媒体集中报道"用 AI 复活亲人"（新华英文: http://english.news.cn/20250406/0de3c703fead4741826ca1a614d69bc1/c.html ；人民网英文: http://en.people.cn/n3/2025/0406/c90000-20298379.html ；ChinaDaily: https://global.chinadaily.com.cn/a/202504/07/WS67f32678a3104d9fd381dd43.html ）。
- 产品与公司：南京超级头脑（Silicon Intelligence 硅基智能）——"AI 复活"头部公司，2025 年 11 月递交港股 IPO，宣传"8 万个数字人今年开始赚钱"（东方财富: https://finance.eastmoney.com/a/202511073557660656.html ；公司介绍: https://btw.media/zh/silicon-intelligence ）；36 氪报道"9.9 元用 AI 复活亲人：是技术的安慰，还是情感的幻觉？"（https://36kr.com/p/3553685486828681 ）与"红杉 1600 万刀押注数字永生"（https://36kr.com/p/3352583775695489 ）。
- 前沿探索：DeepMind 研究"生成幽灵"（Generative Ghosts）让逝者"赛博重生"（36 氪: https://www.36kr.com/p/3260657440554757 ）。
- 伦理讨论：新华网 2025-04 专题"永生的数字生命，你能接受吗？"（http://www.xinhuanet.com/20250404/5793c0d5c8764614a0fd4af6daf7354a/c.html ）。
- 对 gaea 的启示：**"数字永生"在国内的合规路径未通，但"把一个人的记忆/人格数据私有化保存"（本地数字遗产）是可行的差异化**——单机本地存储天然规避了云端伦理与数据合规问题，这是 gaea 相对星野/猫箱的独特位置。同时需注意：情感陪伴新规（训练数据收紧）对云端产品影响大，本地自托管影响小。

---

### 3.3 陪伴产品商业现实对比

| 产品 | 公司 | 用户/收入现实 | 形态 | 问题 |
|---|---|---|---|---|
| 星野 | MiniMax | 上亿用户量级、海外收入占 73.1%、净亏 5.12 亿美元(2025 前三季) | UGC 智能体+聊天 | 不盈利、监管收紧、智能体下架风波 |
| Talkie | MiniMax | 出海主力、主要收入来源 | UGC 智能体 | 依赖海外市场 |
| 猫箱 | 字节跳动系 | 与星野流量同量级 | 角色聊天 | 同上 |
| Character.AI | CAI | 海外头部；2025-10 因诉讼限制 18 岁以下 | 角色聊天 | 安全合规诉讼 |
| X Eva | 小冰 | 克隆人 App；2025 年出现停服报道 | AI 克隆人 | 商业化失败迹象 |
| 硅基智能 | 南京超级头脑 | 2025-11 递交港股 IPO，"8 万数字人" | 数字人/复活服务 | 伦理争议 |
| 各类"AI 女友" | 数十家 | 赛道半年吸金超 5 亿(艾瑞 2025-08) | 恋爱陪伴 | 留存与合规 |

### 3.4 陪伴赛道关键信号（对 gaea 的决策输入）

- **需求被验证**：星野+猫箱流量与 Kimi 同量级（https://www.tmtpost.com/7392953.html ），头部用户日聊 75 分钟（https://news.iresearch.cn/content/202508/531576.shtml ）——"陪伴"是真需求。
- **盈利未验证**：MiniMax 招股书显示高增长伴随巨亏（https://m.21jingji.com/timeline/5201b332b6d10fe98ff18b8a68d0ac36.html ）；AI 社交下载量暴跌八成（https://m.thepaper.cn/newsDetail_forward_31086524 ）。
- **监管已落地**：2025-12-29 情感陪伴类 AI 新规，训练数据要求收紧（https://www.21jingji.com/article/20251229/herald/9659b7cd5ba774a951ff08aac51b60c3.html ）。云端大厂的"拟人化陪伴"成本上升。
- **gaea 的机会**：本地单机陪伴（人格数据不出本机、无云端训练依赖）天然规避新规核心条款；"数字生命/数字遗产"（把用户或亲人的记忆数据本地化保存）是情感叙事上的差异化入口，且可绕过"复活逝者"的伦理红线——**定位为"记忆守护"而非"人形复活"**。

---

## 4. 个人知识库 / 记忆产品

### 4.1 记忆技术四条路线（对标 gaea 记忆中枢）

| 路线 | 代表 | 记忆形态 | 特点 |
|---|---|---|---|
| 向量检索记忆 | Mem0 | 从对话抽取事实→向量库→检索注入 | "memory layer"中间件，跨会话持久上下文 |
| 上下文状态管理 | Letta（前 MemGPT） | 自我编辑记忆 + 上下文调度 | agent 框架级，论文出身 |
| 时序知识图谱 | Zep | 实体/关系随时间演化的知识图谱 | 企业级，强调"何时发生" |
| 本地文件渐进披露 | Napkin | markdown/文件系统为记忆，按需披露 | local-first、可读、可审计 |

**Mem0（"AI 记忆层"）**
- 开源项目，自述 "Universal memory layer for AI Agents"，从会话中抽取并持久化记忆，与 LangGraph/LangChain 等集成（GitHub: https://github.com/mem0ai/mem0 ；官方博客: https://mem0.ai/blog/how-to-build-a-production-ai-agent-with-langgraph-and-mem0 ）。
- 商业化验证：2025-10-28 完成 2400 万美元 A 轮（YC、Peak XV、Basis Set 领投）（TechCrunch: https://techcrunch.com/2025/10/28/mem0-raises-24m-from-yc-peak-xv-and-basis-set-to-build-the-memory-layer-for-ai-apps/ ）——**"记忆即服务"是被资本验证的方向**。技术细节：用嵌入向量做记忆检索（https://mem0.ai/blog/how-mem0-uses-embeddings-and-why-we-are-evaluating-nvidia-nemotron-3-embed ）。
- 形态总结：**API/中间件**，托管或自托管，服务任意 agent；对单机应用的价值是"抽取—存储—检索"的记忆管线范式。

**Letta（前 MemGPT）**
- "stateful agents framework with memory, reasoning, and context management"，框架级内存管理（GitHub: https://github.com/lettalang/letta ；框架分析: https://github.com/larsderidder/framework-analysis/blob/main/tier-1/letta.md ；PyPI: https://pypi.org/project/letta-nightly/ ）。
- 核心思想：agent 像操作系统管理内存一样管理自己的上下文（MemGPT 论文路线），记忆可被 agent 自编辑。形态：**agent 运行框架**（偏开发者），非终端产品。

**Zep（企业级 agent 记忆）**
- "agent memory at enterprise scale"，基于**时序知识图谱**（temporal knowledge graph）的记忆架构（论文 2501.13956: https://axi.lims.ac.uk/paper/2501.13956 ；官方文档: https://help.getzep.com/v3/memory-for-agent-frameworks ）。提供与 LangChain/LangGraph/CrewAI/n8n/Microsoft Agent Framework 的集成（PyPI: https://pypi.org/project/zep-ms-agent-framework/ ；n8n 节点: https://www.npmjs.com/package/n8n-nodes-zep-memory-v3 ）。
- 形态总结：**知识图谱式记忆中间件**，强调实体关系与时序，适合"谁/何时/什么关系"的记忆问答。

**Napkin（本地优先 agent 知识系统）**
- 开源项目，自述 "Knowledge system for agents. Local-first, file-based, progressively disclosed."——**本地优先、基于文件、渐进披露**（GitHub: https://github.com/Michaelliv/napkin ；作者博客"Building napkin - a memory system for agents": https://michaellivs.com/blog/building-napkin-memory-system-for-agents/ ；设计文档: https://github.com/Michaelliv/napkin/blob/4d939149fc64720d7566735bfc22789f8b9a9251/docs/agent-memory-progressive-disclosure.md ）。
- 与 gaea 最契合：**记忆=用户可读的本地文件**（而非黑盒向量库），按需把相关片段披露给模型。单机应用天然适合这种"记忆可审计"形态。

### 4.2 本地/个人 RAG 助手形态

- **AnythingLLM**：工作区 + 文档问答 + 溯源（见 1.2）。个人用户"本地知识库默认选择"。
- **腾讯 ima（AI 知识管家）**：国产个人知识库 App 的代表，2025 年接入元宝，Copilot 功能全面开放（此前超 10 万人排队）（21 经济 2026-05: https://m.21jingji.com/article/20260525/herald/927e1691a2c9f27f8f8fc51f9f052f9f_zaker.html ；凤凰科技: https://tech.ifeng.com/c/8tyhc1zY8EO ）。形态：**云端个人知识库**（笔记/文档/网页收藏 + AI 问答 + 溯源）。
- 本地轻量实现案例（开源）：Local-KB（SQLite + FTS5 + 余弦重排，零外部依赖: https://github.com/thejoshualewis/Local-KB ）、RAG-chatbot-local（FAISS + Ollama + Streamlit: https://github.com/RohinV/RAG-chatbot-local ）、LocalAI + Elasticsearch 个人知识助手（Elastic 官方博客: https://www.elastic.co/search-labs/kr/blog/local-rag-personal-knowlege-assistant-localai-elasticsearch ）。
- 对 gaea 的启示：**记忆中枢应"文件可读 + 向量检索 + 时序图谱"三者分层**（Napkin 式本地文件层做底、Mem0 式抽取层做记忆写入、Zep 式图谱层做人物/事件关系），而不是只做一个向量库。

---

### 4.3 记忆形态分层对比（对照 gaea 记忆中枢）

| 层面 | 现成方案 | 形态 | gaea 落法建议 |
|---|---|---|---|
| 底层存储 | Napkin / SQLite / markdown | 文件系统即记忆，可读可审计 | 用户可浏览/编辑的记忆文件（如 `memory/` 目录），导出=所有权 |
| 事实抽取 | Mem0 式管线 | 从对话抽取 (实体,属性,关系) 写入 | 聊天/写作/办公 agent 统一写回记忆中枢 |
| 检索 | Mem0/AnythingLLM 式向量库 | 嵌入检索 + 相关度注入 | sqlite-vec 或本地嵌入模型，全部离线 |
| 关系/时序 | Zep 式轻量图谱 | 谁/何时/什么关系 | 轻量实体图谱（人物、故事角色、项目） |
| 注入策略 | Letta 式上下文管理 | 记忆按需加载进上下文 | 每轮对话按相关性注入，控制 token 预算 |
| 会话记忆 | 各聊天客户端 | 单会话上下文 | 会话树/分支（写作与陪伴场景需要回滚） |

### 4.4 关键洞察

- 资本方向：Mem0 的 2400 万美元 A 轮（https://techcrunch.com/2025/10/28/mem0-raises-24m-from-yc-peak-xv-and-basis-set-to-build-the-memory-layer-for-ai-apps/ ）证明"记忆层"是被 VC 认可的基础设施方向；但个人单机应用不需要"记忆服务"，需要的是**本地记忆管线**。
- 用户心智：AnythingLLM 与腾讯 ima 教育了中文用户"知识库 = 上传文档 + 问答溯源"（https://tech.ifeng.com/c/8tyhc1zY8EO ）。gaea 记忆中枢若只做"聊天记忆"会被视为弱产品；必须同时覆盖**文档知识库 + 对话记忆 + 人物/项目关系**，且全部本地。
- 隐私卖点：本地记忆 = 无云端训练 = 对新规免疫，是 gaea 相对 ima 这类云端知识库的差异化（ima 的数据在腾讯云）。

---

## 5. 桌面 agent 的本地模型生态

### 5.1 推理层：成熟

- **Ollama**：本地模型事实标准（GGUF 分发 + OpenAI 兼容 API + 模型库），个人用户心智"一条命令跑模型"。
- **LM Studio**：桌面 GUI + 本地 API server，多 GPU/RTX 50 支持（见 1.2）。
- **llama.cpp / llama-server**：底层引擎，提供官方 function calling 支持（llama.cpp 官方文档: https://mintlify.wiki/ggml-org/llama.cpp/advanced/function-calling ）。
- 这些层的稳定性没问题，问题在上一层。

### 5.2 工具调用（tool calling）：公认痛点

本地模型跑 agent 的核心障碍是**工具调用不可靠**，多个一手 issue 佐证：
- 流式响应会破坏工具调用，需要 stream:false 回退（OpenClaw issue #5769: https://github.com/openclaw/openclaw/issues/5769 ）。
- 部分模型根本"不支持工具调用"（Ollama issue #6704: https://github.com/ollama/ollama/issues/6704 ）。
- Ollama 拉取 hf.co 模型时不套用内置渲染/解析器，导致工具调用不可靠（Ollama issue #17636: https://github.com/ollama/ollama/issues/17636 ）。
- 模型会幻觉输出 `<tool_call>` XML 标签而非 JSON（LiquidAI LFM2-24B 讨论: https://huggingface.co/LiquidAI/LFM2-24B-A2B/discussions/7 ）。
- 小模型在工具调用任务上会"卡死/停滞"，需要专门修复（unsloth PR #4769: https://github.com/unslothai/unsloth/pull/4769 ）。
- 生态上出现了专门为工具调用微调的小模型（如 RefinedToolCallV5-3b: https://huggingface.co/RefinedNeuro/RefinedToolCallV5-3b 、limbic-tool-use-0.5B-32K: https://huggingface.co/Mungert/limbic-tool-use-0.5B-32K-GGUF ），但能力天花板明显。

### 5.3 真能跑通的本地 agent 形态（个人用户实测路径）

1. **OpenClaw（前 Clawdbot）+ Ollama（Docker 一键）**：本地 agent 网关的成熟样板，社区提供安全加固的一键脚本（https://github.com/builtbyV/openclaw-ollama-setup ）；Ionos 教程（https://www.ionos.co.uk/digitalguide/server/configuration/openclaw-ollama/ ）、Seeed Jetson 教程（https://wiki.seeedstudio.com/local_openclaw_on_recomputer_jetson/ ）、中文教程（腾讯云: https://cloud.tencent.cn/developer/article/2648199 ）——说明"真能跑通"，但都强调**模型选择与兼容性配置**。
2. **Dify + Ollama 本地 agent 工作流**：图形化编排 + 本地推理，"零成本搭建私有 AI Agent（Ollama+DeepSeek+Dify）"（百度云: https://cloud.baidu.com/article/4207987 ；Dify 接入本地模型实战: https://cloud.baidu.com/article/3363167 ；CSDN 教程: https://blog.csdn.net/2302_80329073/article/details/160184099 ）。适合 RAG + 简单工具流。
3. **Claude Desktop + MCP（云端大脑 + 本地工具）**：MCP 让桌面客户端成为本地执行层（腾讯云: https://cloud.tencent.com.cn/developer/article/2674529 ；DesktopCommander 把 AI 变成本地执行层: https://cloud.tencent.cn/developer/article/2708602 ）。这是目前**个人用户真正在用**的"桌面 agent"主流——大脑在云端、工具在本地。
4. **桌面工具 + Ollama 混合**：Hermes（桌面 AI 工具）接 Ollama 成为流行组合（ZDNet 2026-06: https://www.zdnet.com/article/hermes-ollama-hands-on-desktop-ai-tool/ ；中文评测: https://ai.zhiding.cn/2026/0611/3190238.shtml ；本地跑 Hermes + Qwen3.5-27B 踩坑记录: https://cloud.tencent.cn/developer/article/2660521 ——"踩坑记录"本身说明现实）。

### 5.4 现实结论（对 gaea 模型中心）

- **推理能本地，智能难本地**：本地小模型做聊天/续写/RAG 足够；做稳定多步 agent（规划+工具调用+记忆调度）仍不成熟。
- 主流个人用户方案是**本地推理 + 云端模型混合**（Ollama 跑轻任务/隐私任务，云端跑复杂 agent），或干脆"云端大脑 + 本地工具"（MCP）。
- gaea 模型中心应设计为**多后端适配层**：本地（Ollama/LM Studio/llama.cpp server）与云端（OpenAI 兼容 API）统一接口，按任务自动路由，并暴露"工具调用质量"提示（如检测到本地小模型时降级为无工具模式）。

---

### 5.5 本地 agent 能力阶梯（个人用户实际可达到的水平）

| 层级 | 能力 | 本地可行度 | 代表路径 | 现实说明 |
|---|---|---|---|---|
| L1 推理 | 聊天/续写/翻译/摘要 | ✅ 成熟 | Ollama/LM Studio/llama.cpp | GGUF 模型 + 显存足够即可 |
| L2 检索 | RAG 问答 | ✅ 成熟 | Ollama + AnythingLLM / Dify | 分块+嵌入+检索，教程多 |
| L3 单步工具 | 单次工具调用(搜索/计算) | ⚠️ 可用但有坑 | MCP + 云端模型；本地需选支持 function calling 的模型 | 流式/格式问题普遍(见 5.2) |
| L4 多步 agent | 规划+多工具+记忆 | ❌ 不稳 | OpenClaw + Ollama(需调优)；Dify 工作流(半自动) | 小模型工具调用卡死、幻觉 XML 标签 |
| L5 桌面自动化 | 读屏/点击/文件操作 | ❌ 实验性 | Claude Desktop + MCP(云端大脑) | 本地模型尚难胜任 |

**结论**：个人用户在本地跑 agent 的现实是——**L1/L2 已经完全落地，L3 可落地需挑模型，L4/L5 要靠云端大脑**。gaea 的办公 agent 模块若设计成"完全本地"，只能承诺 L2；要承诺 L4 必须做混合路由（本地模型做理解与生成，复杂规划走云端或预置模板）。

### 5.6 模型选择现实（中文个人用户 2025—2026）

- 本地可跑的中文可用模型：Qwen 系列（14B/32B 量化）、DeepSeek 蒸馏版（R1 蒸馏 7B/14B）、GLM 系列小模型；中文社区实操文（https://cloud.tencent.cn/developer/article/2660521 ）显示"本地跑 Hermes + Qwen3.5-27B"需要 32GB 以上显存级别的配置且要踩大量坑。
- 结论：**gaea 模型中心必须按硬件分级**（8GB/16GB/32GB 显存对应不同模型档位），并提供"本地档位不足时自动建议云端"的降级路径，否则个人用户开局即受挫。

---

## 6. 对 gaea 3.0 的整体启示（合并）

1. **形态定位**：桌面市场缺"单机单用户的跨模块智能体"，gaea 的模块组合（聊天+陪伴+写作+绘图+办公 agent+记忆）本身就是差异化，无需再造聊天壳。
2. **记忆中枢**：采用"本地文件（Napkin 式，可读可审计）+ 向量检索（Mem0 式）+ 关系图谱（Zep 式，轻量版）"分层设计；数字遗产/人格数据本地私有化是相对云端陪伴产品的合规优势。
3. **小说创作**：定位"个人创作助手"（灵感/设定/润色/世界书），避开网文平台反 AI 红线；参考 Lattics 的知识库写作工作台形态。
4. **陪伴模块**：流量赛道（星野/猫箱）已证明需求，但盈利与监管双难；gaea 走"本地、私密、无广告订阅"路线，与监管新规（云端训练数据收紧）错位竞争。
5. **模型中心**：默认混合调度（本地推理 + 云端智能），兼容 MCP 插件生态，对本地小模型工具调用做降级处理。
6. **插件层**：直接兼容 MCP，不造私有协议（Cherry Studio/Claude Desktop 已验证）。

---

## 7. 来源 URL 汇总（按主题）

### 主题 1：桌面 AI 助手
- https://github.com/GZC888/chatbox （Chatbox）
- https://github.com/mz247/chatbox/blob/main/README-CN.md （Chatbox 中文说明）
- https://landscape.jimmysong.io/projects/chatbox/ （AI 原生全景图）
- https://cherrystudiocn.com/index.html （Cherry Studio 官网）
- https://docs.cherryai.com.cn/ （Cherry Studio 文档）
- https://cloud.tencent.com/developer/article/2617735 （Cherry Studio 深度解析）
- https://cloud.baidu.com/article/3931431 （Cherry Studio 与 ChatBox 对比）
- https://docs.siliconflow.cn/en/usercases/use-siliconcloud-in-cherry-studio （硅基流动接入 Cherry Studio）
- https://www.jan.ai/docs/desktop/quickstart （Jan 快速开始）
- https://github.com/Innovation-Labs-Technical-Hub/jan-app （Jan GitHub）
- https://itsfoss.com/jan-ai/ （Jan 实测：回到 Ollama）
- https://docs.openwebui.com/alternatives/jan/ （Open WebUI vs Jan）
- https://cloud.tencent.cn/developer/article/2472803 （Jan 20.5K stars）
- https://blog.csdn.net/feikillyou/article/details/155708100 （Jan 39.2k Star）
- https://beta.lmstudio.ai/blog/lmstudio-v0.3.14 （LM Studio 多 GPU）
- https://beta.lmstudio.ai/blog/lmstudio-v0.3.15 （LM Studio RTX50+工具调用）
- https://skywork.ai/blog/ai-agent/claude-desktop-vs-chatgpt-perplexity-copilot-lm-studio-2025-comparison/ （桌面工具对比）
- https://docs.anythingllm.com/chatting-with-documents/introduction （AnythingLLM 工作区）
- https://docs.anythingllm.com/chatting-with-documents/rag-in-anythingllm （AnythingLLM RAG 原理）
- https://cloud.baidu.com/article/3676944 （Ollama+AnythingLLM 本地知识库）
- https://docs.enconvo.com/docs/intro （Enconvo 文档）
- https://www.toolmage.com/en/tool/enconvo/ （Enconvo 介绍）
- https://dev.to/purpledoubled/i-compared-5-local-ai-uis-open-webui-lm-studio-sillytavern-jan-and-my-own-1453 （5 款本地 UI 对比）
- https://www.XDA-Developers.com/after-two-months-open-webui-updates-pick-chatgpt-interface-local-llms/ （Open WebUI 评测）
- https://adg.csdn.net/694cf0d35b9f5f31781a9446.html （开源 MCP 客户端盘点）

### 主题 2：中文小说写作 AI
- https://apps.apple.com/cn/story/id1585749129 （彩云小梦 iOS）
- https://baike.baidu.com/item/%E5%BD%A9%E4%BA%91%E5%B0%8F%E6%A2%A6/59403428 （彩云小梦百科）
- https://36aidianping.com/note-detail/3568010593719853 （彩云小梦测评）
- https://appstorespy.com/ios-apple-store/1564619616-trends-revenue-statistics-downloads-ratings （彩云小梦商店数据）
- http://www.news.cn/tech/20241113/80d5b68aba9c445da0ad94d5467a308f/c.html （彩云 DCFormer 大模型）
- https://wyb.chinawriter.com.cn/Pad/content/202507/04/content79915.html （网文 AI 写作综述）
- https://baike.baidu.com/item/%E4%BD%9C%E5%AE%B6%E5%8A%A9%E6%89%8B%E5%A6%99%E7%AC%94%E7%89%88/63227250 （作家助手妙笔版）
- https://m.163.com/dy/article/KC3PV3K10514CDBK.html （妙笔通鉴/漫剧助手）
- https://www.sohu.com/a/944943008_100116740 （阅文千万字理解 AI 应用）
- https://epaper.gzdaily.cn/news/html/2025/02/15/content_873_879639.htm （网文率先部署 DeepSeek）
- https://ibiling.cn/novel-navigation/detail/1 （笔灵 AI 小说）
- https://sj.qq.com/appdetail/com.ibiling.app （笔灵移动端）
- https://aitool.csdn.net/6a4e1b5010ee7a33f2895e79.html （8 款写小说工具盘点）
- https://apps.apple.com/ao/app/lattics-brain-like-writing/id1575605022?mt=12 （Lattics）
- https://www.producthunt.com/products/lattics-2 （Lattics Product Hunt）
- https://lattics.com/zh-CN/review/lattics-academic-paper-workflow （Lattics 学术写作工作流）
- https://gamesbeat.com/midreal-generative-ai-storytelling/ （Midreal 发布）
- https://www.cbinsights.com/company/midreal （Midreal 公司信息）
- https://cloud.tencent.cn/developer/article/2376909 （机器之心：AI 夺回网文界）
- https://sudowrite.com/blog/ai-fiction-writing-stack/ （SudoWrite 作者写作栈）
- https://reedsy.com/blog/guide/book-writing-software/sudowrite/ （SudoWrite 评测）
- https://www.chinawriter.com.cn/n1/2025/0905/c404027-40557712.html （晋江反 AI 创作公告分析）
- https://m.jiemian.com/article/13655654.html （AI 重新定义起点番茄）
- https://www.cssn.cn/skgz/bwyc/202604/t20260420_5981165.shtml （2025 网文报告：作者对 AI 态度复杂）
- https://www.pconline.com.cn/focus/1736/17361632.html （AI 短剧单篇两万）

### 主题 3：AI 陪伴 / 数字生命
- https://m.21jingji.com/timeline/5201b332b6d10fe98ff18b8a68d0ac36.html （MiniMax 招股书数据）
- https://m.bjnews.com.cn/detail/1766325559129059.html （MiniMax 现金储备）
- https://ep.ycwb.com/epaper/xkb/h5/html5/2025-12/24/content_1511_736204.htm （AI 恋人上亿用户未盈利）
- https://www.tmtpost.com/7392953.html （星野+猫箱流量与 Kimi 同量级）
- https://www.tmtpost.com/7358826.html （星野获米哈游腾讯投资）
- https://m.21jingji.com/article/20251129/herald/c4cf0b7853b66ba95873b68fb2dee2f2_zaker.html （星野下架智能体）
- https://m.thepaper.cn/newsDetail_forward_31086524 （AI 社交下载量暴跌）
- https://m.36kr.com/p/3253625617196293 （AI 社交投放下载腰斩）
- https://www.21jingji.com/article/20251229/herald/9659b7cd5ba774a951ff08aac51b60c3.html （情感陪伴 AI 新规）
- https://arstechnica.com/information-technology/2025/10/after-teen-death-lawsuits-character-ai-will-restrict-chats-for-under-18-users/ （Character.AI 限龄）
- https://axis-intelligence.com/character-ai-statistics/ （Character.AI 统计）
- https://baike.baidu.com/item/X%20eva/63058852 （X Eva 克隆人）
- https://www.infzm.com/contents/285722 （小冰李笛谈情感）
- https://www.nfnews.com/content/K3BDX0e0oY.html （克隆人平台停服）
- https://news.iresearch.cn/content/202508/531576.shtml （AI 女友赛道半年吸金 5 亿）
- http://english.news.cn/20250406/0de3c703fead4741826ca1a614d69bc1/c.html （清明 AI 复活亲人）
- https://global.chinadaily.com.cn/a/202504/07/WS67f32678a3104d9fd381dd43.html （AI 哀悼服务）
- https://finance.eastmoney.com/a/202511073557660656.html （硅基智能 IPO，8 万数字人）
- https://36kr.com/p/3553685486828681 （9.9 元 AI 复活亲人）
- https://36kr.com/p/3352583775695489 （红杉 1600 万刀数字永生）
- https://www.36kr.com/p/3260657440554757 （DeepMind 生成幽灵）
- http://www.xinhuanet.com/20250404/5793c0d5c8764614a0fd4af6daf7354a/c.html （新华网：数字生命伦理）

### 主题 4：记忆 / 知识库
- https://github.com/mem0ai/mem0 （Mem0 GitHub）
- https://techcrunch.com/2025/10/28/mem0-raises-24m-from-yc-peak-xv-and-basis-set-to-build-the-memory-layer-for-ai-apps/ （Mem0 2400 万美元 A 轮）
- https://mem0.ai/blog/how-to-build-a-production-ai-agent-with-langgraph-and-mem0 （Mem0+LangGraph）
- https://github.com/lettalang/letta （Letta GitHub）
- https://pypi.org/project/letta-nightly/ （Letta PyPI）
- https://github.com/larsderidder/framework-analysis/blob/main/tier-1/letta.md （Letta 框架分析）
- https://help.getzep.com/v3/memory-for-agent-frameworks （Zep 文档）
- https://axi.lims.ac.uk/paper/2501.13956 （Zep 时序知识图谱论文）
- https://github.com/Michaelliv/napkin （Napkin GitHub）
- https://michaellivs.com/blog/building-napkin-memory-system-for-agents/ （Napkin 作者博客）
- https://docs.anythingllm.com/chatting-with-documents/introduction （AnythingLLM 工作区）
- https://tech.ifeng.com/c/8tyhc1zY8EO （ima 接入元宝）
- https://m.21jingji.com/article/20260525/herald/927e1691a2c9f27f8f8fc51f9f052f9f_zaker.html （ima Copilot 全面开放）
- https://github.com/thejoshualewis/Local-KB （Local-KB 轻量本地知识库）
- https://www.elastic.co/search-labs/kr/blog/local-rag-personal-knowlege-assistant-localai-elasticsearch （LocalAI+ES 本地 RAG）

### 主题 5：本地模型 agent 生态
- https://github.com/openclaw/openclaw/issues/5769 （流式破坏工具调用）
- https://github.com/ollama/ollama/issues/6704 （模型不支持工具调用）
- https://github.com/ollama/ollama/issues/17636 （hf.co 渲染解析器问题）
- https://mintlify.wiki/ggml-org/llama.cpp/advanced/function-calling （llama.cpp 函数调用）
- https://huggingface.co/LiquidAI/LFM2-24B-A2B/discussions/7 （模型幻觉 XML 标签）
- https://github.com/unslothai/unsloth/pull/4769 （小模型工具调用卡死修复）
- https://huggingface.co/RefinedNeuro/RefinedToolCallV5-3b （工具调用微调小模型）
- https://github.com/builtbyV/openclaw-ollama-setup （OpenClaw+Ollama 一键部署）
- https://www.ionos.co.uk/digitalguide/server/configuration/openclaw-ollama/ （OpenClaw+Ollama 教程）
- https://wiki.seeedstudio.com/local_openclaw_on_recomputer_jetson/ （Jetson 本地 OpenClaw）
- https://cloud.tencent.cn/developer/article/2648199 （OpenClaw 接入本地 Ollama）
- https://cloud.baidu.com/article/4207987 （Ollama+DeepSeek+Dify 零成本私有 Agent）
- https://cloud.baidu.com/article/3363167 （Dify 接入本地大模型）
- https://cloud.tencent.com.cn/developer/article/2674529 （MCP 详解）
- https://cloud.tencent.cn/developer/article/2708602 （DesktopCommander 本地执行层）
- https://www.zdnet.com/article/hermes-ollama-hands-on-desktop-ai-tool/ （Hermes+Ollama 实测）
- https://cloud.tencent.cn/developer/article/2660521 （本地跑 Hermes+Qwen3.5-27B 踩坑）

---

*报告完。数据截至检索时点；用户量级均来自公开报道，未经独立核验。*
