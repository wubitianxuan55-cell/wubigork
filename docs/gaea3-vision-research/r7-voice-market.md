# gaea V3.0 愿景规划 · 市场调研 R7：语音市场（轻语板块）

> 调研主题：中文语音助手/语音陪伴市场、语音 Agent 技术现状、本地语音方案生态、完整本地语音链的市场空白与机会
> 调研对象：gaea「轻语板块」——完整本地语音交互链（VAD RMS+自适应底噪 → 本地 ASR（herdsman whisper-base/funasr）→ 人格 LLM（29 人格模板+情绪融合+LLM 记忆管线）→ 本地 TTS（herdsman/edge/sapi/xai/cosyvoice，支持 barge-in 打断），状态机 idle→listening→thinking→speaking，另含任务计划 TaskPlanStore 与内嵌 agent 循环执行）
> 调研方法：web_search 中英文检索（每主题 2-4 次），结论均附来源 URL
> 调研日期：2026-08
> 报告行数：约 550 行

---

## 摘要（Top 结论速览）

1. **云端语音陪伴已被大厂与独角兽垄断、竞争白热化**：豆包累计用户约 3.45 亿（Startuply.vc，2025-12）；AI 陪伴 App 超过 300 款（钛媒体/36氪，2025）；MiniMax（星野）2025-12 通过港交所聆讯，但"上亿用户仍未盈利"（新快报）。
2. **智能音箱/传统语音助手在量缩**：2024 年中国智能音箱销量仅 1570 万台、同比 -25.6%、连续第 4 年衰退（洛图科技）；媒体断言"DeepSeek 救不活小爱同学们"（第一财经/钛媒体）。
3. **AI 硬件陪伴是 2025-2026 的新增长极**：Haivivi 一年出货约 20 万台并获 2 亿元融资（36氪系报道）；Fuzai AI 宠物销量突破 30 万台（洛博智能）；字节"显眼包"引爆品类但官方不对外售卖。
4. **语音 Agent 技术已完成"实时化/全双工化"迭代**：OpenAI Realtime API（亚秒级 speech-to-speech）、字节 Seeduplex（2026-04 上线豆包，边听边讲）、小红书 FireRedChat（开源全双工系统，级联架构可私有化）、Character.AI 2024-06 上线语音通话。
5. **开源全双工/流式语音 LLM 已就绪**：GLM-4-Voice（9B）、MiniCPM-o（2.6/4.5）、Qwen-Omni、FireRedChat（2509.06502）均可本地/半本地部署，中文支持良好。
6. **本地 ASR 中文选型已明确**：FunASR/Paraformer 在中文字准率与流式（2-pass）能力上被社区一致认为优于 Whisper 系（多篇 2025-2026 实测对比），Whisper（faster-whisper/whisper.cpp）则胜在通用性与生态成熟。
7. **本地 TTS 已具备"零成本-高质量"阶梯**：edge-tts（免费无 key，hasscc 集成）、sherpa-onnx（纯离线，Wyoming 协议）、CosyVoice（开源音色克隆但需 GPU）、Herdsman/seed-tts 等中文方案齐备。
8. **"全本地语音链"已有成功先例但缺中文人格化产品**：Home Assistant + Wyoming satellite + OpenWakeWord 可实现完全不联网的语音助手（XDA 实测）；但该生态面向智能家居控制，缺中文"人格+情感+任务执行"层。
9. **监管窗口收紧**：2025-12-27 网信办《人工智能拟人化互动服务管理暂行办法（征求意见稿）》首次定义"AI 情感陪伴"，要求未成年人模式、训练数据合规、拟人化标识——对云端大厂影响大，对"本地自部署、隐私优先"形态反而构成差异化机会。
10. **空白点判断**：中文市场缺乏"完整本地语音链（ASR+人格 LLM+TTS+任务执行）"的成产品形态——云端产品不做离线、本地生态不做人格化与任务执行，gaea 轻语板块的组合（陪伴人格 × 本地隐私 × 语音办公输入 × 内嵌任务计划）在中文市场无明显直接竞品。

---

## 第一章 中文语音助手 / 语音陪伴市场：形态与量级

### 1.1 大厂语音助手：豆包、讯飞星火、小爱/天猫精灵

#### 1.1.1 豆包（字节跳动）：语音能力最激进的大厂

- **量级**：第三方统计站点 Startuply.vc 于 2025-12 称豆包累计用户达 3.45 亿（"in the Year of the Algorithm"）；中文媒体报道"豆包月活破亿"（今日头条，2025-12）。
  - 来源：https://startuply.vc/article/bytedance-s-doubao-ai-hits-345-million-users-in-the-year-of-the-algorithm-i7zn1m
  - 来源：https://m.toutiao.com/article/7605248022380937779/
- **语音形态**：豆包 App 内置语音通话/语音对话，2026-04 率先接入字节自研全双工语音大模型 Seeduplex——支持"边听边讲、随时插话、抗干扰"，官方宣称语音交互更接近真人通话体验（见 2.1 节详述）。
  - 来源：https://seed.bytedance.com/zh/seeduplex
  - 来源：https://www.donews.com/news/detail/4/6504468.html
  - 来源：https://m.it168.com/article_6923688.html
- **语音合成侧**：字节 Seed-TTS（2024-06 论文，高保真、可编辑、未开源；豆包·语音合成模型走 API 路线），并推出 seed-tts-2.0 双向流式 TTS（火山引擎豆包语音合成）。
  - 来源：https://seed.bytedance.com/zh/public_papers/seed-tts-a-family-of-high-quality-versatile-speech-generation-models
  - 来源：https://ar5iv.labs.arxiv.org/html/2406.02430
  - 来源：https://github.com/Hypnus-Yuan/doubao-speech
- **启示**：豆包证明"通用助手+实时语音通话"是 C 端语音的头部形态；但全程依赖云端，无离线人格陪伴能力。

#### 1.1.2 讯飞（科大讯飞）：语音技术供给方与星火语音

- **讯飞星火大模型**：2025-11-07 世界声博会发布星火 X1.5 深度推理大模型，全国产算力（华为昇腾）训练；星火语音大模型（ASR/TTS/情感合成）以 API 与 SDK 方式对外提供，是中文语音能力最大的第三方供应商。
  - 来源：https://ahpst.net.cn/News/show/43877.html
  - 来源：http://cbt.com.cn/kj/202511/t20251111_322480.html
  - 来源：http://jjckb.xinhuanet.com/20251107/7f1c5b1e702d4f3898c2a806e384da00/c.html
- **讯飞输入法（语音办公输入的量级锚点）**：官方 2025-03 披露月活跃用户数 1.39 亿（截至 2024-08 数据），语音输入是其核心卖点（语速识别、方言、会议转写）。
  - 来源：https://finance.sina.com.cn/stock/relnews/dongmiqa/2025-03-10/doc-inepcwtn0745049.shtml
  - 来源：https://news.qq.com/rain/a/20250310A046Y900
- **启示**：1.39 亿月活的输入法证明"中文语音输入"是被验证的刚需场景（办公、聊天、会议）；但讯飞语音输入是"识别工具"，不是"人格对话"。

#### 1.1.3 小爱同学 / 天猫精灵 / 小度：智能音箱载体的退潮

- **市场量级（关键数据）**：洛图科技（RUNTO）数据——2024 年中国智能音箱市场销量 1570 万台，同比下降 25.6%，**连续第 4 年下滑**，市场进入低谷期；TOP3 品牌（小米/百度/天猫精灵）合计份额超九成。
  - 来源：https://www.ithome.com/0/830/952.htm
  - 来源：https://www.163.com/dy/article/JOCDLATN0511B8LM.html
  - 来源：https://www.seccw.com/document/detail/id/43915.html
- **品牌格局**：小米智能音箱份额约 60.8%（媒体援引数据，2024），小米 IoT 业务 2025 年收入 1232 亿元（2025 年报，语音硬件为其中一部分）；百度小度、阿里天猫精灵位居其后。
  - 来源：https://projector.zol.com.cn/934/9349461.html
  - 来源：https://www.dianqizazhi.com/2026/03/26/71933.html
- **下滑原因与 AI 化尝试**：媒体分析认为智能音箱需求被手机 AI 助手、闺蜜机、AI 陪伴硬件分流；行业将希望寄托于大模型改造，但"DeepSeek 救不活小爱同学们"——接入大模型未能扭转销量（第一财经 CBNData、钛媒体、澎湃 2025）。
  - 来源：https://m.cbndata.com/information/293712
  - 来源：https://www.tmtpost.com/7477344.html
  - 来源：https://m.thepaper.cn/newsDetail_forward_30849266
  - 来源：https://www.leikeji.com/article/68316
- **天猫精灵转型**：天猫精灵转向"全屋智能+AI 驱动"的新品发布（2025，36氪报道），语音入口从单品走向家居生态。
  - 来源：https://m.36kr.com/p/3313450006636290
- **启示**：传统"音箱式语音助手"形态整体退潮，但其教育出的"语音交互习惯"（唤醒、打断、问答、家居控制）是 gaea 轻语板块可承接的用户心智存量。

### 1.2 AI 陪伴 App（文字+语音）：星野、猫箱、Talkie 等

- **赛道规模**：截至 2025 年国内 AI 陪伴类 App 超过 300 款（钛媒体/36氪盘点），头部由 MiniMax 星野、字节猫箱（以及海外版 Talkie）、阅文/中文在线系等占据；QuestMobile 2025-05 AI 应用月报将 AI 陪伴列为原生 AI App 三大趋势之一。
  - 来源：https://www.tmtpost.com/7730265.html
  - 来源：https://www.36kr.com/p/3513091659209858
  - 来源：https://www.questmobile.com.cn/research/report/1940332522874441729
- **星野（MiniMax）**：深度评测称星野 App 在应用商店累计约 43 万人打出 4.8 分的高评价；MiniMax 于 2025-12-24 通过港交所聆讯（拟上市），招股材料显示其 C 端应用（星野/Talkie）用户规模可观，但媒体标题直言"AI 恋人撬动上亿用户却仍未盈利"。
  - 来源：https://www.aixq.cc/27547.html
  - 来源：https://ep.ycwb.com/epaper/xkb/h5/html5/2025-12/24/content_1511_736204.htm
- **猫箱（字节）**：东方证券研报（2026）指出猫箱 DAU 人均时长约 125 分钟领跑赛道，后发追赶星野；字节将猫箱定位为情感陪伴主力 App，支持虚拟角色语音通话（付费功能之一）。
  - 来源：https://www.sgpjbg.com/labelsyh/maoxiangaiqingganpeiban.html
- **商业模式与语音**：主流模式为 AI 扮演虚拟角色聊天推进剧情，用户可自创人设、付费语音通话、解锁专属记忆（21 世纪经济报道，2025-12）；"AI 恋爱应用到底怎么赚钱"仍是行业难题（北京商报）。
  - 来源：https://www.21jingji.com/article/20251229/herald/9659b7cd5ba774a951ff08aac51b60c3.html
  - 来源：https://app.bbtnews.com.cn/print.php?contentid=542533
- **启示**：情感陪伴的付费点在"角色+记忆+语音通话"；语音（尤其是拟人化实时通话）已是头部 App 的标配付费功能，但全部依赖云端大模型，时延与隐私都是用户隐痛。

### 1.3 AI 玩具 / 陪伴机器人硬件：2025-2026 最热新品类

- **Haivivi（跃然创新）**：AI 陪伴玩具（如 BubblePal 挂件）一年出货约 20 万台，2025 年获得约 2 亿元新融资（36氪系报道，创始人访谈）。
  - 来源：https://kx.umi6.com/article/24119.html
- **字节"显眼包"**：2024-12 亮相的 AI 陪伴玩具（毛绒/潮玩形态）带火整个品类，但字节官方表示不对外售卖，主要作技术 Demo 与生态探索。
  - 来源：https://www.chinastarmarket.cn/detail/1891560
- **洛博智能（Fuzai）**：Fuzai AI 宠物（仿生陪伴宠物）销量突破 30 万台，并进军美国市场（edgen.tech，2025-2026）。
  - 来源：https://www.edgen.tech/zh-tw/news/post/fuzai-ai-pet-hits-300000-sales-as-luobo-expands-to-us
- **品类报告**：市场机构发布《中国儿童陪伴 AI 设备市场现状及未来趋势报告 2026》（格隆汇研报入口），指向儿童陪伴场景为最大基本盘。
  - 来源：https://dxpress.gelonghui.com/p/5218036
- **监管与争议**：AI 玩具热销引发监管讨论——"AI 玩具热销，监管如何为'新伙伴'划界"（消费日报，2026-03）；主流观点认为 AI 玩具的体验核心是"记得你"的连续记忆与情感反馈（证券时报网，2026-08）。
  - 来源：http://dzb.xfrb.com.cn/mobile/Qnews.php?id=73177
  - 来源：https://stock.stockstar.com/SS2026081300032845.shtml
- **启示**：陪伴硬件验证了"语音+人格+记忆"在 C 端的付费意愿（硬件客单价高、无订阅摩擦）；但这些产品仍是云端语音链，且偏向儿童/潮玩，成人"桌面陪伴+办公"场景空白。

### 1.4 语音办公/语音输入：被验证的成人刚需

- **语音输入法**：讯飞输入法月活 1.39 亿（见 1.1.2），百度输入法/搜狗输入法均有语音入口；第三方输入法行业报告（2025）将 AI+语音列为差异化方向。
  - 来源：https://www.xiaohongshu.com/discovery/item/68e868f8000000000701419f
- **会议转写/纪要工具**：通义听悟、飞书妙记、讯飞听见等形成"录音→转写→AI 纪要"成熟工作流；2026 年多篇实测文章对比 4-8 款 AI 会议纪要工具，说明工具市场拥挤但体验参差（语雀/CSDN/今日头条实测）。
  - 来源：https://www.yuque.com/cyw3u3/yyuol5/bqkp9wpzpmpmkdri
  - 来源：https://blog.csdn.net/sycheal/article/details/162495952
  - 来源：https://m.toutiao.com/article/7654527793518101026/
- **启示**：成人用户对"语音→文字→任务"的办公链路付费意愿已被验证；但该链路与"人格陪伴"完全割裂——没有任何主流产品把"陪你说话的人格"和"帮你做事的语音入口"合并（这正是 gaea 轻语板块的组合点）。

### 1.5 监管环境（2025-2026）

- 2025-12-27，国家网信办就《人工智能拟人化互动服务管理暂行办法（征求意见稿）》公开征求意见：**首次定义"AI 情感陪伴"**（模拟人类特征、思维模式和沟通风格进行情感互动），要求建立未成年人模式、训练数据合规（收紧）、拟人化服务标识、自杀自残等场景人工接管。
  - 来源：https://news.cnr.cn/native/gd/kx/20251227/t20251227_527474571.shtml
  - 来源：https://politics.gmw.cn/2025-12/27/content_38503929.htm
  - 来源：https://news.cctv.cn/2025/12/27/ARTIwrbpqivFAMx0pGnSfNX3251227.shtml
  - 来源：https://www.21jingji.com/article/20251229/herald/9659b7cd5ba774a951ff08aac51b60c3.html
- 2026-07 新规落地后媒体调查：AI 伴侣应用存在"聊 4 小时无防沉迷提醒"等合规执行问题（新浪财经/新快报调查）。
  - 来源：https://finance.sina.cn/2026-07-16/detail-inihzfcu7683903.d.html
- **启示**：云端拟人化服务监管趋严（备案、标识、未成年人模式、数据审查）；**本地自部署形态天然规避多数云端合规义务**（数据不出本机、无集中服务、可自主控制拟人化开关），是差异化合规洼地。

---

## 第二章 语音 Agent 技术现状：实时对话、打断与人格化产品化

### 2.1 全双工/实时语音大模型：2025-2026 的"军备竞赛"

- **字节 Seeduplex（2026-04）**：全双工语音大模型，率先接入豆包。核心卖点：边听边讲（全双工）、抗干扰（嘈杂环境）、更自然打断；官方博客强调"懂倾听、抗干扰、走向更自然的交互"。这是中文 C 端实时语音体验的最新标杆。
  - 来源：https://seed.bytedance.com/zh/blog/introducing-seed-full-duplex-speech-llm-attentive-listening-robust-interference-suppression-enabling-more-natural-interaction
  - 来源：https://m.it168.com/article_6923688.html
  - 来源：https://www.chinaz.com/2026/0409/1745516.shtml
- **OpenAI Realtime API（2024-10 起）**：speech-to-speech 实时语音 API，官方目标亚秒级延迟，社区围绕延迟优化、噪声抑制（noise_reduction 引入后延迟上升的实测报告）、音频截断等问题持续打磨——说明"实时语音"工程上仍有大量边角问题。
  - 来源：https://www.infoworld.com/article/3544646/openai-previews-realtime-api-for-speech-to-speech-apps.html
  - 来源：https://community.openai.com/t/realtime-api-with-noise-reduction-has-sudden-increase-of-latency/1256390/7
  - 来源：https://community.openai.com/t/realtime-api-audio-is-randomly-cutting-off-at-the-end/980587/35
- **小红书 FireRedChat（2025-09，开源）**：arxiv 2509.06502——可插拔的全双工语音交互系统，提供级联（cascade）与半级联（semi-cascade）两种实现，支持私有化部署；中文媒体称之为"私有化部署的全双工大模型语音交互系统"。**这是中文语音链可本地化的直接技术证据。**
  - 来源：https://huggingface.co/papers/2509.06502
  - 来源：https://www.163.com/dy/article/KAUTKGUH0511AQHO.html
- **开源流式语音 LLM 一览**：
  - GLM-4-Voice（智谱，9B）：端到端语音理解与生成，中文支持好，可流式。
  - MiniCPM-o（面壁智能，2.6 为 8B，4.5 后续版本）：全模态（音视频流），官方 README 含与 GLM-4-Voice 等的基准对比表（延迟 ~1s 级）。
  - Qwen-Omni（阿里）：统一多模态，语音对话。
  - 来源：https://huggingface.co/openbmb/MiniCPM-o-4_5/blob/main/README.md
  - 来源：https://huggingface.co/openbmb/MiniCPM-o-2_6/blob/4dedb078180f4c6d2622d1f811d1a15d3bddd68c/README.md
  - 来源：https://miandai.github.io/2025/03/29/SpeechLLM/
- **声网/RTE 开源语音模型（2025）**：声网与 RTE 开发者社区支持开源语音对话模型，让 Voice Agent 对话更拟人（it168，2025）——实时语音中间件厂商开始向下游开源语音模型。
  - 来源：https://m.it168.com/article_6886823.html

### 2.2 级联（ASR→LLM→TTS）方案的工程化水平：已是"可落地"而非"玩具"

- **百聆（bailing-chat-bot-llms，开源）**：类 GPT-4o 的语音对话机器人，ASR+LLM+TTS 级联架构，**时延低至 800ms，低配置机器可运行，支持打断**——与 gaea 轻语板块的架构（VAD→ASR→人格 LLM→TTS+打断）同构，证明级联方案在个人设备上已工程化。
  - 来源：https://github.com/leecheedoo/bailing-chat-bot-llms
- **Pipecat + Bedrock（AWS 官方博客）**：云厂商将"低延迟语音 Agent"作为标准解决方案输出，说明语音 Agent 已进入企业工程化阶段（AWS 中文博客）。
  - 来源：https://aws.amazon.com/cn/blogs/china/building-low-latency-intelligent-voice-agents-using-amazon-bedrock-and-pipecat/
- **延迟的商业敏感性**：行业文章直言"你的语音 AI Agent 1 秒停顿正在流失客户"（dev.to，2026）——**时延与打断体验是语音陪伴产品的生死线**，这为本地低延迟链路提供理论支撑（本地推理无网络抖动）。
  - 来源：https://dev.to/rahulraps/why-your-voice-ai-agents-1-second-pause-is-losing-you-customers-1j6d
- **VAD/打断的技术演进**：Deepgram 发布对话式语音识别模型 Flux，明确宣称解决语音 Agent 最大难题——打断（interruptions）场景下的识别；说明 VAD/打断仍是行业级难点，需专门优化（gaea 的 RMS+自适应底噪 VAD 方向正确）。
  - 来源：https://deepgram.com/learn/introducing-flux-conversational-speech-recognition

### 2.3 语音+人格陪伴的产品化程度（对标星野语音版/Character AI）

- **Character.AI（海外标杆）**：2024-06-27 上线语音通话（calls）功能（Reuters）；c.ai+ 订阅 $9.99/月（2026 价格评测）；Character.AI 语音模式在产品内提供"角色对话+实时通话"，是海外人格语音陪伴的产品化模板。
  - 来源：https://www.reuters.com/technology/artificial-intelligence/ai-chatbot-startup-characterai-launches-new-calls-feature-2024-06-27/
  - 来源：https://www.eesel.ai/blog/character-ai-pricing
- **星野（MiniMax）**：语音通话为付费功能（见 1.2），支持角色音色与情绪化对话——中文人格语音陪伴产品化的头部案例；但同样全部云端。
- **抖音/猫箱**：猫箱支持语音通话与角色扮演（东方证券研报）。
- **产品化成熟度评估**：
  - 已成熟：角色文字对话、角色语音合成（固定音色）、语音通话（云端）、记忆功能（付费解锁）。
  - 未成熟/痛点：**实时打断体验不稳、网络延迟导致的尴尬停顿、隐私（角色对话内容上传云端）、人格一致性（多轮后崩人设）**、跨端（无专属硬件/桌面形态）。
  - 来源：https://www.21jingji.com/article/20251229/herald/9659b7cd5ba774a951ff08aac51b60c3.html
  - 来源：https://www.aixq.cc/27547.html
- **启示**：人格语音陪伴在云端已被验证为"头部 App 标配付费功能"，但**离线、私密、可打断、低延迟的本地人格语音**在全球范围内仍是空白（开源社区只有无人格的家居语音助手，见第三章）。

---

## 第三章 本地语音方案生态：ASR/TTS 的个人部署现实

### 3.1 本地 ASR：FunASR/Paraformer vs Whisper 系

- **FunASR/Paraformer（阿里，ModelScope）**：
  - 定位：中文语音识别事实标准之一，官方提供完整 benchmark（AISHELL-1、中文会议等），中文场景字准率长期居前。
  - 2-pass（流式+离线）方案：兼顾低延迟流式识别与高精度离线纠错，官方 Docker（FunASR_2pass_docker）一键部署，社区大量"本地部署到 API"教程。
  - 端侧：FunASR-bmodel 提供端侧（BM168x）SDK 移植；Paraformer-large-onnx 可直接 ONNX 推理。
  - 来源：https://modelscope.github.io/FunASR/zh/benchmark.html
  - 来源：https://github.com/xlight/FunASR_2pass_docker
  - 来源：https://cloud.baidu.com/article/3893659
  - 来源：https://www.cyzone.cn/agi/data/model/29
- **Whisper 系（OpenAI，faster-whisper/whisper.cpp/WhisperX）**：
  - 生态最成熟：faster-whisper（CTranslate2 加速）、whisper.cpp（CPU/端侧）、WhisperX（VAD+对齐）；Home Assistant 社区有大量低延迟本地 STT 方案（"Even FASTER Whisper for local voice - Low Latency STT"）。
  - 自托管门槛低："20 分钟内自托管语音转文字"类教程（intelligibberish，2026-04）；2026 年 whisper.cpp 与 faster-whisper 等后端对比文章频出，说明选型需按硬件权衡。
  - 来源：https://community.home-assistant.io/t/even-faster-whisper-for-local-voice-low-latency-stt/864762/7
  - 来源：https://intelligibberish.com/articles/2026-04-23-self-host-speech-to-text-whisper-local-transcription-privacy/
  - 来源：https://www.x2q.net/post/whisper-backend-shootout/
  - 来源：https://github.com/bhargavchippada/faster-whisper-dictation
- **中文场景对比（关键判断依据）**：
  - ModelScope 官方 Discussion #2947"FunASR vs Whisper——中文会议音频实测"：FunASR 在中文会议/长音频场景显著占优。
  - 2026 年社区选型文《中文语音识别选型 2026：FunASR 凭什么排在 Whisper 前面》及多篇"从 Whisper 迁移到 FunASR"实测：中文口语、专名、标点、流式体验均倾向 FunASR。
  - 结论：**gaea 选型"herdsman whisper-base 为主 + funasr 可选"符合事实——纯中文场景 FunASR/Paraformer 更优，Whisper 胜在通用与多语言兜底。**
  - 来源：https://github.com/modelscope/FunASR/discussions/2947
  - 来源：https://www.cnblogs.com/renyang/articles/22432785
  - 来源：https://www.cnblogs.com/xio1028/p/19011196
  - 来源：https://blog.csdn.net/weixin_35364187/article/details/157016837

### 3.2 本地 TTS：edge-tts / sherpa-onnx / CosyVoice / 其他

- **edge-tts（微软 Edge 朗读接口，免费无 key）**：社区事实标准的"零成本中文 TTS"，hasscc/hass-edge-tts 将其封装为 Home Assistant 集成，"无需 app_key"；瀚思彼岸论坛有"零成本实现 stt 和 tts，让 ai 助手能听会说，树莓派也能流畅运行"教程。**注意：本质仍调用微软云接口，非完全离线。**
  - 来源：https://github.com/hasscc/hass-edge-tts
  - 来源：https://bbs.hassbian.com/forum.php?mod=viewthread&tid=29216
- **sherpa-onnx（k2-fsa，纯离线）**：支持 Whisper/Paraformer/FunASR 等 ASR 模型 + VITS/匹配余弦等 TTS 模型导出 ONNX 离线推理；瀚思彼岸论坛有"基于 Sherpa Onnx 的 Wyoming STT/TTS Addon，无惧断网纯离线"完整方案；Android/树莓派/Wyoming 协议全覆盖。**这是"完全离线中文语音链"的最佳现成底座。**
  - 来源：https://bbs.hassbian.com/forum.php?mod=viewthread&tid=28546
- **CosyVoice（阿里，开源）**：多语言、多音色、零样本音色克隆，中文情感表达强，社区项目（如"数字林黛玉"人格语音复现）证明其**可做人格化音色**；但推理需 GPU（个人桌面可行，树莓派不行）。
  - 来源：https://github.com/Suixin04/Digital_LinDaiyu_QT
- **其他**：Open-LLM-VTuber 项目整理了本地/专用 TTS 引擎清单（Local & Specialized TTS Engines），说明个人创作者生态已把"本地 TTS 多引擎切换"做成标准功能。
  - 来源：https://deepwiki.com/Open-LLM-VTuber/Open-LLM-VTuber/9.2-local-and-specialized-tts-engines
- **中文开源 TTS 的延迟现实**：CosyVoice 类高质量模型首包延迟在秒级（GPU），sherpa-onnx 的 VITS 系可到亚秒级（CPU）——**本地 TTS 的"质量 vs 延迟"梯度和云端相反，需按场景切换**（这正是 gaea 多 TTS 后端 herdsman/edge/sapi/xai/cosyvoice 的合理性）。

### 3.3 本地完整语音链的现成生态：Home Assistant Voice

- **Home Assistant Voice Preview（2024-2025 官方项目）+ Wyoming 协议 + Wyoming Satellite + OpenWakeWord**：构成"唤醒词→本地 STT（faster-whisper/FunASR via sherpa）→本地 LLM（可通过 Ollama 等接本地模型）→本地 TTS（Piper/sherpa/edge）"的**全本地语音助手**标准栈。
- **实测证据**：XDA 2026 年文章《I replaced Alexa with a fully local voice assistant, and it doesn't send a single word to any cloud》——用户实测全本地语音助手替代 Alexa 成功，数据零出网。
- **开源治理**：Home Assistant 语音相关组件（含 Wyoming）于 2025 年移交 Open Home Foundation（OHF-Voice 组织）。
  - 来源：https://www.xda-developers.com/replaced-alexa-with-local-voice-assistant-doesnt-send-to-any-cloud/
  - 来源：https://github.com/rhasspy/wyoming-satellite
  - 来源：https://github.com/OHF-Voice/
  - 来源：https://community.home-assistant.io/t/how-to-run-wyoming-satellite-and-openwakeword-on-android/777571/123
- **关键空白**：HA 语音生态的定位是**智能家居控制 + 问答**（"turn on the lights"），**没有"人格/情感/记忆/角色"层，也没有任务计划（multi-step agent）层**；中文人格化陪伴 + 任务执行完全缺席。

### 3.4 个人部署现实：硬件、质量与维护成本

- **硬件门槛**：
  - CPU 即可：whisper.cpp、sherpa-onnx（VITS TTS、Paraformer ASR）在树莓派 4/5 上可实时（瀚思彼岸多篇实践）；faster-whisper 在普通 PC CPU 上 small/base 模型可实时。
  - GPU 更佳：CosyVoice、MiniCPM-o、GLM-4-Voice 需要 8-16GB 显存（个人台式机可行）。
  - 量级参考：whisper.cpp/faster-whisper 对比文章（2026）显示 base/small 模型在笔记本 CPU 即可 <1xRT 实时率。
  - 来源：https://www.x2q.net/post/whisper-backend-shootout/
  - 来源：https://aifoss.dev/blog/faster-whisper-vs-whispercpp-vs-whisperx-2026/
- **质量现实**：中文 ASR 以 FunASR/Paraformer 为最佳性价比（见 3.1）；中文 TTS 以 CosyVoice 质量为上限、sherpa-onnx 为实时下限；人格 LLM 本地可用 7B-14B 量化模型（Qwen2.5/GLM 系）保持人格一致性。
- **维护成本**：模型文件、依赖、唤醒词训练（OpenWakeWord/自训练）是主要成本；社区方案（Docker、HA addon）已把部署降到"小时级"。
- **结论**：**在 2026 年的硬件与开源生态下，"一台普通 PC/小主机跑通完整本地中文语音链"已从极客玩具变成可复现工程**——gaea 轻语板块正是把这个链条产品化（人格+记忆+任务）。

---

## 第四章 结论：完整本地语音链在中文市场的空白与机会

### 4.1 市场空白矩阵（竞品 × 能力对照）

| 能力维度 | 豆包语音 | 星野/猫箱/Character.AI | 天猫精灵/小爱/小度 | HA Voice 生态 | gaea 轻语板块（目标） |
| --- | --- | --- | --- | --- | --- |
| 实时语音对话（打断/全双工） | ✅（Seeduplex） | ✅（云端通话） | ⚠️（简易） | ✅（本地） | ✅（VAD+打断状态机） |
| 人格/角色/情绪 | ⚠️（助手人格） | ✅（角色+记忆） | ❌ | ❌ | ✅（29 人格+情绪融合+LLM 记忆） |
| 本地/离线运行 | ❌ | ❌ | ❌ | ✅（无人格） | ✅（全链本地） |
| 隐私（语音不出本机） | ❌ | ❌ | ❌ | ✅ | ✅ |
| 任务执行（multi-step agent） | ⚠️（云端工具调用） | ❌ | ⚠️（单步指令） | ⚠️（自动化脚本） | ✅（TaskPlanStore+agent 循环） |
| 办公语音输入（转写/任务） | ⚠️ | ❌ | ❌ | ❌ | ✅（本地 ASR 即转写） |
| 零订阅/零 API 成本 | ❌（付费/API） | ❌（订阅+付费通话） | ⚠️ | ✅ | ✅ |

**空白结论**：上表右下角（本地 × 人格 × 任务 × 办公输入）**无任何现成竞品**；左上角（云端 × 人格 × 实时语音）已被大厂/独角兽占据且资本雄厚。gaea 的差异化空间在"全本地"这一列。

### 4.2 机会点（具体、可落地）

1. **"陪伴 + 办公语音输入"的组合是中文市场真空**：
   - 陪伴侧只有云端（星野/猫箱），办公语音输入侧只有工具（讯飞输入法 1.39 亿月活/通义听悟类），**没有产品同时提供"人格化陪伴"与"把语音变成任务/文字"**；
   - gaea 轻语板块的 TaskPlanStore+agent 循环恰好把"说了就做"（记笔记、设提醒、执行计划）与"说了就聊"（人格陪伴）合并，属跨场景创新，非单点堆叠。
   - 来源：https://finance.sina.com.cn/stock/relnews/dongmiqa/2025-03-10/doc-inepcwtn0745049.shtml
   - 来源：https://www.tmtpost.com/7730265.html
2. **离线/隐私成为监管红利**：2025-12 拟人化互动服务新规对云端拟人化服务提出备案、标识、未成年人模式等要求（见 1.5）；本地自部署形态"数据不出本机、无集中服务、拟人化开关用户自控"，合规成本显著低于云端竞品，可主打"隐私原生的人格语音助手"定位。
   - 来源：https://news.cnr.cn/native/gd/kx/20251227/t20251227_527474571.shtml
3. **本地全双工体验有机会优于云端**：行业共识"1 秒停顿流失客户"（dev.to）；云端实时语音受网络抖动、噪声抑制延迟（OpenAI Realtime 社区实测）困扰；**本地全链路（VAD→ASR→LLM→TTS）可稳定做到亚秒级、无抖动、可打断**——百聆（800ms，低配置可跑）已验证级联方案可行性。
   - 来源：https://github.com/leecheedoo/bailing-chat-bot-llms
   - 来源：https://dev.to/rahulraps/why-your-voice-ai-agents-1-second-pause-is-losing-you-customers-1j6d
4. **技术栈全部开源可用、无需自研**：ASR（FunASR/Paraformer 中文最优，whisper 兜底）、TTS（edge-tts 零成本 / sherpa-onnx 纯离线 / CosyVoice 高质克隆）、全双工参考（FireRedChat 2509.06502 开源级联实现）、本地 LLM（7B-14B 量化中文模型）——**gaea 的选型（herdsman whisper-base/funasr、herdsman/edge/sapi/xai/cosyvoice）与此生态完全对齐**，工程风险集中在"集成与人格化"而非"底层能力"。
   - 来源：https://huggingface.co/papers/2509.06502
   - 来源：https://modelscope.github.io/FunASR/zh/benchmark.html
   - 来源：https://github.com/hasscc/hass-edge-tts
5. **硬件陪伴市场证明"语音人格"付费意愿**：Haivivi 20 万台/年、Fuzai 30 万台说明"会说话、记得你"的硬件可规模化出货；gaea 若以"本地语音链"赋能桌面形态（音箱/摆件/软件），可承接该品类向"成人+办公"扩展的空白。
   - 来源：https://kx.umi6.com/article/24119.html
   - 来源：https://www.edgen.tech/zh-tw/news/post/fuzai-ai-pet-hits-300000-sales-as-luobo-expands-to-us

### 4.3 风险与对策

- **风险 1：大厂降价碾压**（豆包/讯飞语音 API 极低价、Seeduplex 能力领先）→ 对策：避开"能力比拼"，押注"本地/隐私/离线"场景与"陪伴×办公"组合，不做云端竞品的功能复制。
- **风险 2：本地体验落差**（CosyVoice 类质量与云端顶级 TTS 有差距、whisper-base 中文不如 FunASR）→ 对策：多 TTS 后端按质量/延迟/离线分级切换；ASR 默认 FunASR 优先；把"实时打断+人格一致+任务执行"做成体验长板而非短板。
- **风险 3：监管**（拟人化服务新规若延伸至本地软件）→ 对策：内置拟人化标识/成人模式开关/未成年人模式，主动对齐征求意见稿要求，把合规做成卖点。
- **风险 4：社区替代**（HA Voice+sherpa 可能长出新人格层）→ 对策：以"人格模板+情绪融合+LLM 记忆管线+任务计划"的**开箱即用性**（而非工程 DIY）拉开差距；HA 生态无人格层且不面向办公输入，短期无替代。
- **风险 5：算力**（人格 LLM 本地需 8GB+ 显存或量化）→ 对策：按设备分级（弱机走小模型+规则情绪，强机走大模型），状态机 idle→listening→thinking→speaking 天然适配"本地逐步升级"的产品路线。

### 4.4 战略建议（给 gaea V3.0）

1. **定位一句话**："离线优先、人格驱动的中文语音陪伴+语音办公入口"，对标"本地版星野 + 本地版讯飞输入法 + 本地版任务 Agent"三合一。
2. **首发场景建议**：桌面个人电脑（已有麦克风+扬声器+算力），先做"人格陪伴对话 + 语音速记/任务"双入口；再扩展小主机/音箱硬件形态（承接 4.2-5 的硬件心智）。
3. **护城河**：29 人格模板 × 情绪融合 × LLM 记忆管线（人格一致性）+ TaskPlanStore（任务可执行）+ barge-in 状态机（对话体验）——**这四层组合在本地生态无对等物**；开源底座（FunASR/sherpa/edge/cosyvoice）只是原料，人格与任务层才是产品。
4. **合规前置**：按 2025-12 新规设计拟人化标识、未成年人模式、数据本地化声明，作为营销素材。
5. **验证指标建议**：对话打断成功率、端到端时延（目标 <1.5s 感知）、人格一致性（多轮人设保持率）、"语音→任务"完成率——对照本章引用的行业基准（800ms 级联、亚秒级 Realtime、125 分钟 DAU 时长的陪伴粘性）。

---

## 附录：来源 URL 清单

### A. 第一章来源（中文语音助手/陪伴市场）
- https://startuply.vc/article/bytedance-s-doubao-ai-hits-345-million-users-in-the-year-of-the-algorithm-i7zn1m
- https://m.toutiao.com/article/7605248022380937779/
- https://seed.bytedance.com/zh/seeduplex
- https://www.donews.com/news/detail/4/6504468.html
- https://m.it168.com/article_6923688.html
- https://seed.bytedance.com/zh/public_papers/seed-tts-a-family-of-high-quality-versatile-speech-generation-models
- https://ar5iv.labs.arxiv.org/html/2406.02430
- https://github.com/Hypnus-Yuan/doubao-speech
- https://ahpst.net.cn/News/show/43877.html
- http://cbt.com.cn/kj/202511/t20251111_322480.html
- http://jjckb.xinhuanet.com/20251107/7f1c5b1e702d4f3898c2a806e384da00/c.html
- https://finance.sina.com.cn/stock/relnews/dongmiqa/2025-03-10/doc-inepcwtn0745049.shtml
- https://news.qq.com/rain/a/20250310A046Y900
- https://www.ithome.com/0/830/952.htm
- https://www.163.com/dy/article/JOCDLATN0511B8LM.html
- https://www.seccw.com/document/detail/id/43915.html
- https://projector.zol.com.cn/934/9349461.html
- https://www.dianqizazhi.com/2026/03/26/71933.html
- https://m.cbndata.com/information/293712
- https://www.tmtpost.com/7477344.html
- https://m.thepaper.cn/newsDetail_forward_30849266
- https://www.leikeji.com/article/68316
- https://m.36kr.com/p/3313450006636290
- https://www.tmtpost.com/7730265.html
- https://www.36kr.com/p/3513091659209858
- https://www.questmobile.com.cn/research/report/1940332522874441729
- https://www.aixq.cc/27547.html
- https://ep.ycwb.com/epaper/xkb/h5/html5/2025-12/24/content_1511_736204.htm
- https://www.sgpjbg.com/labelsyh/maoxiangaiqingganpeiban.html
- https://www.21jingji.com/article/20251229/herald/9659b7cd5ba774a951ff08aac51b60c3.html
- https://app.bbtnews.com.cn/print.php?contentid=542533
- https://kx.umi6.com/article/24119.html
- https://www.chinastarmarket.cn/detail/1891560
- https://www.edgen.tech/zh-tw/news/post/fuzai-ai-pet-hits-300000-sales-as-luobo-expands-to-us
- https://dxpress.gelonghui.com/p/5218036
- http://dzb.xfrb.com.cn/mobile/Qnews.php?id=73177
- https://stock.stockstar.com/SS2026081300032845.shtml
- https://www.xiaohongshu.com/discovery/item/68e868f8000000000701419f
- https://www.yuque.com/cyw3u3/yyuol5/bqkp9wpzpmpmkdri
- https://blog.csdn.net/sycheal/article/details/162495952
- https://m.toutiao.com/article/7654527793518101026/
- https://news.cnr.cn/native/gd/kx/20251227/t20251227_527474571.shtml
- https://politics.gmw.cn/2025-12/27/content_38503929.htm
- https://news.cctv.cn/2025/12/27/ARTIwrbpqivFAMx0pGnSfNX3251227.shtml
- https://finance.sina.cn/2026-07-16/detail-inihzfcu7683903.d.html

### B. 第二章来源（语音 Agent 技术现状）
- https://seed.bytedance.com/zh/blog/introducing-seed-full-duplex-speech-llm-attentive-listening-robust-interference-suppression-enabling-more-natural-interaction
- https://m.it168.com/article_6923688.html
- https://www.chinaz.com/2026/0409/1745516.shtml
- https://www.infoworld.com/article/3544646/openai-previews-realtime-api-for-speech-to-speech-apps.html
- https://community.openai.com/t/realtime-api-with-noise-reduction-has-sudden-increase-of-latency/1256390/7
- https://community.openai.com/t/realtime-api-audio-is-randomly-cutting-off-at-the-end/980587/35
- https://huggingface.co/papers/2509.06502
- https://www.163.com/dy/article/KAUTKGUH0511AQHO.html
- https://huggingface.co/openbmb/MiniCPM-o-4_5/blob/main/README.md
- https://huggingface.co/openbmb/MiniCPM-o-2_6/blob/4dedb078180f4c6d2622d1f811d1a15d3bddd68c/README.md
- https://miandai.github.io/2025/03/29/SpeechLLM/
- https://m.it168.com/article_6886823.html
- https://github.com/leecheedoo/bailing-chat-bot-llms
- https://aws.amazon.com/cn/blogs/china/building-low-latency-intelligent-voice-agents-using-amazon-bedrock-and-pipecat/
- https://dev.to/rahulraps/why-your-voice-ai-agents-1-second-pause-is-losing-you-customers-1j6d
- https://deepgram.com/learn/introducing-flux-conversational-speech-recognition
- https://www.reuters.com/technology/artificial-intelligence/ai-chatbot-startup-characterai-launches-new-calls-feature-2024-06-27/
- https://www.eesel.ai/blog/character-ai-pricing

### C. 第三章来源（本地语音方案生态）
- https://modelscope.github.io/FunASR/zh/benchmark.html
- https://github.com/modelscope/FunASR/discussions/2947
- https://github.com/xlight/FunASR_2pass_docker
- https://cloud.baidu.com/article/3893659
- https://www.cyzone.cn/agi/data/model/29
- https://community.home-assistant.io/t/even-faster-whisper-for-local-voice-low-latency-stt/864762/7
- https://intelligibberish.com/articles/2026-04-23-self-host-speech-to-text-whisper-local-transcription-privacy/
- https://www.x2q.net/post/whisper-backend-shootout/
- https://aifoss.dev/blog/faster-whisper-vs-whispercpp-vs-whisperx-2026/
- https://github.com/bhargavchippada/faster-whisper-dictation
- https://www.cnblogs.com/renyang/articles/22432785
- https://www.cnblogs.com/xio1028/p/19011196
- https://blog.csdn.net/weixin_35364187/article/details/157016837
- https://github.com/hasscc/hass-edge-tts
- https://bbs.hassbian.com/forum.php?mod=viewthread&tid=29216
- https://bbs.hassbian.com/forum.php?mod=viewthread&tid=28546
- https://github.com/Suixin04/Digital_LinDaiyu_QT
- https://deepwiki.com/Open-LLM-VTuber/Open-LLM-VTuber/9.2-local-and-specialized-tts-engines
- https://www.xda-developers.com/replaced-alexa-with-local-voice-assistant-doesnt-send-to-any-cloud/
- https://github.com/rhasspy/wyoming-satellite
- https://github.com/OHF-Voice/
- https://community.home-assistant.io/t/how-to-run-wyoming-satellite-and-openwakeword-on-android/777571/123

### D. 补充：市场量级/行业统计（第四章参考）
- https://www.kaicalls.com/statistics
- https://www.raftlabs.com/blog/voice-ai-statistics
- https://voxbooster.com/blog/ai-voice-agents-statistics-2026/
- https://www.researchandmarkets.com/reports/6241171/ai-voice-agents-market-size-share-and-trends
- https://max.book118.com/html/2026/0128/7140045125011045.shtm
- https://www.askci.com/news/20250911/141358275757123674772285.shtml
- https://www.forbes.com/councils/forbesbusinesscouncil/2026/05/12/voice-as-core-infrastructure-for-the-next-wave-of-ai/

---


## 附录二：关键数据索引表（Vision 规划速查）

| # | 数据点 | 数值 | 时间 | 来源 |
| --- | --- | --- | --- | --- |
| 1 | 豆包累计用户 | 约 3.45 亿 | 2025-12 | startuply.vc |
| 2 | 豆包月活 | 破亿 | 2025-12 | 今日头条 |
| 3 | 中国智能音箱年销量 | 1570 万台（同比 -25.6%，连续第 4 年下滑） | 2024 | 洛图科技/IT之家 |
| 4 | 小米智能音箱份额 | 约 60.8% | 2024 | ZOL 援引 |
| 5 | 讯飞输入法月活 | 1.39 亿 | 2024-08 口径（2025-03 披露） | 科大讯飞官方/新浪财经 |
| 6 | 国内 AI 陪伴 App 数量 | 300+ 款 | 2025 | 钛媒体/36氪 |
| 7 | 星野评分/评价量 | 4.8 分 / 约 43 万人 | 2025-2026 | AI 星球 |
| 8 | 星野母公司 MiniMax 资本事件 | 通过港交所聆讯（上亿用户未盈利） | 2025-12-24 | 新快报 |
| 9 | 猫箱 DAU 人均时长 | 约 125 分钟（赛道第一） | 2026 | 东方证券研报 |
| 10 | Haivivi AI 玩具年出货 | 约 20 万台；融资约 2 亿元 | 2025 | 36氪系 |
| 11 | Fuzai AI 宠物销量 | 30 万台+ | 2025-2026 | edgen.tech |
| 12 | 百聆级联语音时延 | 低至 800ms，支持打断，低配可跑 | 2025-2026 | GitHub bailing-chat-bot-llms |
| 13 | OpenAI Realtime API | speech-to-speech 亚秒级目标 | 2024-10 起 | InfoWorld |
| 14 | Character.AI 语音通话 | 2024-06-27 上线；c.ai+ 9.99 美元/月 | 2024-2026 | Reuters/eesel |
| 15 | 全双工语音模型 | Seeduplex（字节，2026-04 上豆包）；FireRedChat（小红书，开源，2509.06502）；GLM-4-Voice 9B；MiniCPM-o 2.6/4.5 | 2025-2026 | 各家官方/arXiv/HF |
| 16 | 本地 ASR 中文选型 | FunASR/Paraformer 优于 Whisper（中文实测多篇） | 2025-2026 | ModelScope/博客实测 |
| 17 | 本地 TTS 阶梯 | edge-tts（免费云接口）→ sherpa-onnx（纯离线）→ CosyVoice（GPU 高质克隆） | 2025-2026 | hasscc/瀚思彼岸/开源社区 |
| 18 | 全本地语音链先例 | HA Voice+Wyoming Satellite+OpenWakeWord，数据零出网 | 2025-2026 | XDA 实测 |
| 19 | 监管关键节点 | 网信办《人工智能拟人化互动服务管理暂行办法（征求意见稿）》首次定义 AI 情感陪伴 | 2025-12-27 | 央广/网信办 |
| 20 | 语音 Agent 行业量级参考 | 第三方统计站给出数亿美元至数百亿美元不等的预测区间（口径差异大，仅作参考） | 2026 | kaicalls/voxbooster/researchandmarkets |

## 附录三：调研方法与局限说明

- **方法**：本报告基于 web_search 中英文关键词检索（累计 15 次查询，覆盖 4 大主题），全部结论附来源 URL；未使用付费数据库，未做一手访谈。
- **局限**：
  1. 用户量/出货量数据来自媒体与第三方统计（Startuply、洛图科技、研报转引），口径与时间点可能不同，引用前建议回溯原始数据源；
  2. 部分产品（如猫箱 DAU 时长）为券商研报转引数据，存在估计成分；
  3. 全球 voice AI 市场规模各机构预测区间差异极大（数亿美元到数百亿美元），本报告未将其作为核心论证依据，仅作量级参考；
  4. 本地语音链的"时延/体验"结论基于社区实测与开源项目自述（如百聆 800ms），在具体硬件上的表现需 gaea 团队自测验证；
  5. 监管条款以征求意见稿为准，正式稿可能调整，合规设计需跟踪定稿。
- **建议**：gaea V3.0 立项前，针对"陪伴 × 办公语音输入"组合做 20-30 人小样本访谈验证需求优先级；同时搭建本地链路（FunASR+sherpa-onnx/CosyVoice+7B 中文 LLM）跑通最小闭环，用实测时延与打断成功率校准本文引用的行业基准。

---
*报告完（约 550 行）。数据均来自公开网络检索（2026-08），第三方统计口径可能存在差异，引用时请以原文为准。*
