# 市场调研复扫 · 模块 6：创作包（小说创作 + 绘梦）

## 6. 创作包 · 小说 + 绘梦（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

**小说创作**：海外 Sudowrite 已把「自研模型 + 资产库」做成完整闭环——Muse 1.5 专为小说散文训练、支持章节级输出与 agentic 工作流（[Muse 页，2026-08-29](https://sudowrite.com/muse)）；Story Bible/Story Engine 把角色、世界观、大纲作为每次生成的"记忆"，第三方评为"同类最扎实"；2026-08 上线 Ballad 1.1（批量生成省最高 70% 积分）与 Visualize 场景配图（2,500 积分/张）（[评测，2026-08-29 更新](https://www.varoo.cn/tools/sudowrite.html)；[定价页](https://sudowrite.com/pricing)）。定价 $10/$22/$44 三档，Max 档积分 12 个月滚动、年付约五折。国内蛙蛙写作（自研 Weaver 模型）主打角色卡 + 角色记忆模式（生成时自动校验初始设定防 OOC）与 Agent 工作流，并打通"小说→剧本→漫剧视频（自动角色绘制、镜头拆分、配音）"；按字数计费（约 ¥9.9/万字、¥68.9/50 万字包、3/7 天无限卡），无月卡（[评测指南，2026-01-21](https://aiproducthub.cn/sites/wawawriter-ai-smart-writing-assistant-tutorial.html)；[官网](https://wawawriter.com/app/)）。NovelAI 发布 Diffusion V5："更锐利细节、多角色、整页漫画一个模型"，多角色分离控制 + Vibe Transfer + 局部重绘（[官网](https://novelai.net/)）。基线报告中的阅文作家助手·妙笔（千万字级理解）仍为平台锁定型（内部基线：`docs/gaea-competitive-landscape-2026.md`）。彩云小梦旧域名 301 跳转至 [xiaomengai.com](https://www.xiaomengai.com/)（页面 JS 渲染，现状未核实）；笔灵 AI 2026 年动态未核实（多轮检索未获有效结果）。

**图像生成**：字节 Seedream 5.0 Pro（官网 © 2026；知乎测评 2026-07-09 与"GPT Image 2"对标）把生成+编辑统一架构推进到交互式编辑：响应空间标记与涂鸦线稿、支持图层拆分，可产出故事板/信息图等高密度版式（[官方页](https://seed.bytedance.com/zh/seedream5_0_pro)；Seedream 4.0 统一架构见 [官方页 2025-09](https://seed.bytedance.com/zh/seedream4_0)；[知乎测评（403，据检索摘要）](https://zhuanlan.zhihu.com/p/2058627990901241271)）。Gemini Nano Banana 已列名 Gemini API 官方文档（2026-08-24），Gemini 3.1 Pro 主打设计意图理解（[API 文档](https://ai.google.dev/gemini-api/docs?hl=zh-cn)；[DeepMind](https://deepmind.google/models/gemini/pro/)）；"GPT Image 2"仅见于上述知乎标题，未直接核实。智谱侧 GLM-Image 已升为图像旗舰（复杂指令遵循、文字渲染突出），CogView-4 为常规档、CogView-3-Flash 免费，模型概览未见独立图像编辑模型（[BigModel 文档](https://docs.bigmodel.cn/cn/guide/start/model-overview)）。开源侧 ComfyUI 官网主推 App Mode 简化工作流、降低新手门槛，仍是本地可控性代表（[comfy.org](https://comfy.org/)）；即梦海外版 Dreamina 开始聚合 Seedream/Seedance/GPT 等多家模型（[官网](https://dreamina.capcut.com/zh-tw/)）。可灵、通义万相 2026 动态未核实（检索受限）。

### 范式迁移（上轮调研以来的变化）

1. **编辑范式**：局部重绘/蒙版 →「指令编辑 + 图层化」成为云端旗舰标配（Seedream 5 Pro 图层拆分与批注改图、Gemini 对话式编辑）；gaea 对 img2img 的"诚实拒绝"对应的正是这一缺口。
2. **一致性**：从单图一致性走向「多角色同图 + 整页漫画」（NovelAI V5）与「角色记忆模式」（蛙蛙）；角色资产从写作工具附属品升级为文图共用资产。
3. **文图联动**：Sudowrite Visualize、蛙蛙"小说→剧本→漫剧视频"、Seedream 5 Pro 故事板，说明"叙事→分镜→成片"管线已进入头部产品；个人端"角色库 ↔ 一致性插画"闭环仍稀缺。
4. **计费与留存**：纯月订阅弱化，转向按量积分 + 滚动（Sudowrite 12 个月滚动、NovelAI Anlas 可纯按次购买）、年付大折（约五折）、字数包/无限日卡（蛙蛙，刻意无月卡）、"无限档 + 公平使用限额"（NovelAI V5 附加限额）；提示词/工作流共享社区成为留存抓手。行业系统性留存率数据未核实。

### 对 gaea 的机会与威胁

**机会**：①「创作间」把角色/世界观做成一套共用数据，恰好命中行业从"文本资产"走向"文图共用资产"的方向，是双模块合并的天然护城河；②本地优先 + URL 统一转 data URL 落盘，契合创作者版权敏感（蛙蛙/Sudowrite 均以"内容归用户、不用于训练"为核心卖点）；③智谱 GLM-Image 旗舰 + CogView-3-Flash 免费档便于分层接模型、控成本；④乐园空间 + 个人长期记忆可做出"越用越懂你的人物与世界设定"，是云端通用工具没有的粘性来源。
**威胁**：①云端旗舰（Seedream 5 Pro/Gemini）的指令编辑与一致性能力断层领先，gaea 绘梦除文生图外近乎空白，差距在拉大；②蛙蛙类产品已覆盖"文→图→视频"全链且定价透明，中文网文作者迁移成本低；③通用大模型（长上下文 + 免费/低价）持续挤压轻量续写、润色类功能的独立价值。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **0-3 月**：绘梦模型目录分层接入 GLM-Image（旗舰质量）与 CogView-3-Flash（免费兜底）；角色库/设定页字段一键注入生图提示词（角色卡→提示词模板，对齐 Story Bible 思路）；小说侧上线低成本高感知的「设定/伏笔一致性检查」。
- **3-6 月**：实验多角色同图与简单分镜排版（对标 NovelAI V5 多角色控制）；跟进 ComfyUI App Mode 趋势，评估本地引擎升级或工作流导入可行性；绘梦引入用量（张数/积分）统计与"创作消耗报告"，为将来计费与留存设计铺垫。
- **愿景 6-12 月**：创作间「叙事→插画→分镜漫画」管线（对标蛙蛙漫剧视频与 NovelAI 整页漫画）；世界观/角色资产跨模块复用（轻语角色扮演、小说封面生成）；若商业化，采用"积分 + 滚动 + 年付折扣"组合而非硬月费（2026 年观察到的主流范式）。

### 参考来源

- Sudowrite 定价：https://sudowrite.com/pricing （2026-08 核实）
- Sudowrite Muse 模型页（2026-08-29）：https://sudowrite.com/muse
- Sudowrite 中文评测（2026-08-29 更新，含 Muse 1.5/Story Bible/Visualize/Ballad 1.1）：https://www.varoo.cn/tools/sudowrite.html
- 蛙蛙写作评测指南（2026-01-21）：https://aiproducthub.cn/sites/wawawriter-ai-smart-writing-assistant-tutorial.html
- 蛙蛙写作官网：https://wawawriter.com/app/
- NovelAI 官网（Diffusion V5）：https://novelai.net/
- Seedream 5.0 Pro 官方页：https://seed.bytedance.com/zh/seedream5_0_pro
- Seedream 4.0 官方页（2025-09，生成+编辑统一架构）：https://seed.bytedance.com/zh/seedream4_0
- Seedream 5.0 Pro 知乎测评（2026-07-09，403 仅据检索摘要）：https://zhuanlan.zhihu.com/p/2058627990901241271
- Gemini API 文档（Nano Banana，2026-08-24）：https://ai.google.dev/gemini-api/docs?hl=zh-cn
- Gemini 3.1 Pro：https://deepmind.google/models/gemini/pro/
- 智谱 BigModel 模型概览（GLM-Image/CogView-4/CogView-3-Flash）：https://docs.bigmodel.cn/cn/guide/start/model-overview
- ComfyUI 官网（App Mode）：https://comfy.org/
- Dreamina（即梦海外版，多模型聚合）：https://dreamina.capcut.com/zh-tw/
- 彩云小梦跳转目标（现状未核实）：https://www.xiaomengai.com/
- gaea 内部基线报告：`docs/gaea-competitive-landscape-2026.md`（2026-08-29）

> 调研说明：本节基于 2026-08-31 的网络检索与页面核实；DuckDuckGo 被验证码拦截、部分 Bing 中文查询被污染或过滤、知乎/打开 AI 官网返回 403，故笔灵 AI、可灵、通义万相 2026 年动态与行业留存率数据均标注「未核实」，未编造任何产品或功能。
