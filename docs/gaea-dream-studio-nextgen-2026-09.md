# gaea 绘梦板块 下一代规划与目标「画室」草案（2026-09-05）

> 定位：为「继续优化迭代 gaea · 本轮绘梦板块」提供**外部调研结论 + 现状差距审计 +
> 下一代目标与刀序草案**。基线：gaea v4.97.0（2026-09-05）。
> **长期路线见 `docs/gaea-image-domain-longterm-plan-2026.md`**（本文档 = 调研 +
> 短期草案；长期文档 = 跨版本路线、目标体系与主题线）。
> 权威路线：`docs/gaea-nextgen-roadmap-2026.md` §6/§10/§13.3（绘梦 = 乐园·画室，
> 创作包；与小说共享「创作间」愿景）。本文是 §6 的一次 2026-09 增补与执行化修订。
> 调研窗口：2026-09-04/05 检索；结论均附来源；未能核实项如实标注。

---

## 0.5 定位修订（用户补充 2026-09-05 · 待拍板）

> 用户补充：「绘梦应该是 gaea 的『图』板块核心——可识图、生图、改图；图既服务小说
> （场景、封面等），也服务办公（流程图、思维导图等）；其它功能模块不自己造轮子，
> 需要图就直接调用绘梦，gaea 才能逐渐进化成一体。」

**我的客观判断（见正文回复）**：方向成立，且能治本——代码里目前确有 4–5 套并行的
「图」管线（绘梦工作台 / 角色剧照 / 章节配图与书封 / 办公 diagram 工具 / 办公与微信
识图 OCR），各自有 seam 但契约、资产、溯源元数据不统一，这正是「各造轮子」的痕迹。
但需要做一个**架构层面的澄清**，否则会撞上双空间红线与板块边界：

1. **能力域 ≠ 乐园板块**。建议把「绘梦」在架构上拆成两层：
   - **图像能力域（Image Hub，共享家底）**：识图 / 生图 / 改图 / 图示（图解）+ 统一
     素材资产、模型目录、插槽模板、溯源与成本。全 app 只有这一套引擎与契约。
   - **乐园·画室（play 前台）**：绘梦页继续是乐园的创作工作台，做沉浸式创作；
     工位通过自己的入口（办公配图 / 图示工具 / 识图）消费同一能力域，**不把乐园的
     角色资产/作品记忆注入工位**（保持「默认隔离、显式跨门」红线）。
2. **「图」是四类技术栈，不是一类**：识图（vision/OCR）、生图（扩散）、改图
   （编辑/inpaint）、图示（流程图/思维导图 = Mermaid/代码渲染等确定性图解）应统一在
   **契约层**（请求/响应/资产/能力标签），不在实现层强行合一——各引擎继续走自己的
   最优路径（本地 ComfyUI、云端旗舰、OCR 三级链、Mermaid 渲染互不替代）。
3. **统一入口 ≠ 全走绘梦 UI**：其它模块消费能力契约与资产层，不依赖画室页面/画室
   队列的内部形态；每模块保留自己的领域上下文（小说=角色一致+文风；办公=数据图表+
   图示；造价=OCR 报价+图表），共享的是引擎、资产、溯源与成本。

**待拍板项**：① 图像能力域落在哪个命名空间（`internal/ai` 扩为 imagehub / 新建
`internal/imagehub` / 维持现状只统一 app 门面）；② 识图是否把现有
`GaeaOCRText / GaeaRecognizeImage / visionOCRText / 模型中心识图链`收编为同一能力目录；
③ 办公思维导图（[gaea-office-mindmap-base-design-2026-09.md](gaea-office-mindmap-base-design-2026-09.md)
已拍板为 markdown 大纲 + 交互树）保持办公原生，只把「导出图/嵌入文档」的图资产交给
图像域登记，还是整体改走图像域。

**我的建议（2026-09-05 · 建议口径，待确认）**

1. **图像能力域先落在 app 门面层，不做包级重构**：在既有 `Gaea*` 绑定门面上加一个
   「图像域协调入口」，定义五类能力原语（识图-读 / 识图-懂 / 生图 / 改图 / 图示），
   每类只留规范请求/响应与能力元数据（空间标签、后端、模型、成本、产物溯源），
   引擎实现一律不搬（`internal/ai` 图片后端 seam、`internal/gaea/vision`、
   office/docmd OCR seam 原地保留）。理由：`internal/ai` 与 `internal/app` 都已被审计
   点名膨胀，现在新建 `internal/imagehub` 或扩包只会先付搬迁成本；等契约稳定、调用方
   超过 5 个后再抽独立包，成本更低、风险更小。
2. **识图收编入口，不收编实现**：把识图收敛为两个能力原语——「读」（OCR/逐行取字）
   与「懂」（画面理解/问答/描述），微信、办公、模型中心、造价全部走同一能力目录路由，
   各自的引擎链（PaddleOCR/MinerU/Ovis、本地 vision 大模型等）作为该原语下的降级档
   保留。**不物理合并现有稳定实现**，只统一「谁来调、调哪个原语、回什么结构」。
3. **办公思维导图保持办公原生**：markdown 大纲 + 交互树已拍板，不整体改走图像域；
   只立一条边界——导图/图表要变成交付图（PNG 导出、嵌入 docx/pptx）时，产物经图像域
   统一登记溯源。图示类（流程图/时序/导图/架构图）整体归入图像域「图示」能力，但实现
   继续是 Mermaid/代码渲染，与扩散生图互不替代。

---

## 0. 读前摘要（一句话版）

绘梦的下一代，不是「再加几个模型/再加几个模板」，而是把板块从**生图工作台**升格为
乐园创作间里的**画室**：以**角色资产为锚（人设不跑）、以指令编辑为手感（不满意能修）、
以章节/场景为上下文（正文到成图一条流）**，让小说配图、书封、剧照、独立创作四个入口
共用同一条「角色 → 场景 → 成图 → 改图 → 归档」管线。

一句话目标：**把「抽卡碰运气」变成「确定性地画得出、改得动、攒得下」**。

---

## 1. 外部调研结论（2026-09 快照）

### 1.1 行业共识：生图单点工具已死，云端旗舰全线收敛到「生成 + 编辑 + 资产」

- **阿里 Qwen-Image-3.0-Pro**（2026-08-05 全量上线）：**同一个模型 ID 同时做生成与
  编辑**；支持 4.5k token 长输入、复杂版面/图中图、10px 小字与多语言文字渲染；
  约 0.18 元/张（来源：[阿里云百炼图片生成与编辑](https://www.alibabacloud.com/help/zh/model-studio/image-model)；
  [阿里云产品动态 2026-08-04](https://cn.aliyun.com/product/news/30190)）。
- **智谱 GLM-Image**（2026-01 开源、2026-08 文档页更新）：自回归 + 扩散解码器混合
  架构，面向海报/PPT/科普图等知识密集版面，0.1 元/张（来源：
  [智谱官方文档](https://docs.bigmodel.cn/cn/guide/models/image-generation/glm-image)；
  [证券时报 2026-01-13](https://jnzstatic.cs.com.cn/zzb/htmlInfo/113558.html)）。
- **字节 Seedream 5.0 Pro**：生成 + 编辑统一，支持空间标记、涂鸦线稿、图层拆分，
  可产出故事板/信息图（[官方页](https://seed.bytedance.com/zh/seedream5_0_pro)）。
- **NovelAI Diffusion V5**（2026-08-21 上线）：**22 个角色同屏**、角色位置控制、
  **原生透明背景抠图**、自然语言描述一次生成**整页多格漫画**（[weavai 中文复盘
  2026-08-29](https://weavai.app/blog/zh-cn/2026/08/29/novelai-diffusion-v5%e4%b8%8a%e7%ba%bf%ef%bc%9a22%e4%ba%ba%e5%90%8c%e6%a1%86%e4%b8%8e%e5%8e%9f%e7%94%9f%e6%8a%a0%e5%9b%be/)；
  [中文综述 2026-08-23](https://wenqu123.com/news-detail-1139.html)）。
- **Midjourney V8.2 图像编辑模型**（2026-08-28 开放测试）：引入**多图参考生成机制**
  取代 omni-reference，网页编辑器成为主入口（[百科条目](https://baike.baidu.com/item/Omni-Reference/67326276)）。

**对 gaea 的含义**：gaea 的「图生图 = 整图低 denoise 重绘」已被行业拉开代差；
但云端编辑能力现在**按张计价极低（0.1–0.18 元/张）且大多走 OpenAI 兼容/同模型
ID 编辑**，补课成本远低于两年前的预期。

### 1.2 角色一致性已从「加分项」变成「入场券」，且中文网文场景被实锤验证

- FLUX.2 原生支持**最多 10 张参考图**；PuLID-Flux-2 / IP-Adapter-Flux 提供零训练
  人脸/角色锁定（约 11–12GB VRAM 可跑）（来源：[comfyui-character-gen 技能说明
  2026](https://raw.githubusercontent.com/NeverSight/skills_feed/refs/heads/main/data/skills-md/mckruz/comfyui-expert/comfyui-character-gen/SKILL.md)；
  [Roxabi 实现讨论 #419](https://github.com/Roxabi/roxabi-factory/issues/419)）。
- 中文社区（NGA 2026-08）：起点/番茄书中 **AI 角色图和名场面图穿插已越来越普及**，
  番茄自带文学生图，读者反响正面（[NGA 讨论](https://ngabbs.com/read.php?tid=47348466)）；
  阅文作家助手 Claw 也把「综合章节内容生成配图」列为作家技能（[扬子晚报
  2026-03-14](https://m.yzwb.net/wap/news/4930385.html)）。
- IP-Adapter 类方案在中文教程中的一致性认可度约 80–90%，瓶颈是**门槛高、要拼装
  工作流**（[FlowPixAI 教程 2026-06](https://flowpixai.com/ai-art/ai-painting-novel-illustration.html)）——
  这正是 gaea「画室 + 角色库」要填的位置。
- 工具侧验证：Sudowrite Visualize 把**角色卡的 Physical Description 直接注入成图**
  （[官方文档](https://docs.sudowrite.com/using-sudowrite/1ow1qkGqof9rtcyGnrWUBS/visualize/4MxYzYduLVPbGceg6zMMCb)）；
  阿里开源 **LumenX**（2026-08，小说→动态视频全链）以**角色三视图/场景定调图/道具
  参考图**为美术统一手段（[GitHub alibaba/lumenx](https://github.com/alibaba/lumenx)）；
  GitHub 新项目 Inline-Studio / Kohya-LoRA-Tool / Krea-2-Trainer 均在做
  「本地一键角色 LoRA/画风锁定」的桌面化（[Inline-Studio](https://github.com/inlineresearch/Inline-Studio)；
  [Kohya-LoRA-Tool](https://github.com/l1934332574-maker/Kohya-LoRA-Tool)；
  [阿里云开发者 Krea-2-Trainer 2026-09-02](https://developer.aliyun.com/article/1760379)）。

### 1.3 工作流与模板被平台化：「工作流即应用、一键同款」成为社区标准

- **ComfyUI App Mode / App Builder / ComfyHub**（2026-03-11 发布）：节点图折叠为
  「输入提示词 → 生成」的干净应用界面，工作流可共享 URL、可上 ComfyHub；
  云端与本地同享。海外社区主流声音欢迎（[ComfyUI 官方博客](https://blog.comfy.org/p/from-workflow-to-app-introducing)；
  [中文站](https://comfyui.org/zh/from-workflow-to-app-introducing)；
  [站长之家 2026-03-11](https://www.chinaz.com/ainews/26162.shtml)）。
- 国内平台侧：**LiblibAI** 已到约 3000 万用户 / 50 万模型 / 17 万创作者上传训练
  LoRA，但 2026-06 起被「模型版权迷局」讨论缠身（[百家号 2026-06-19](https://baijiahao.baidu.com/s?id=1868375417972702192)）；
  **吐司**做「在线 ComfyUI + 模型分享 + 一键同款」；**一拍 Image / 无界AI** 均以
  「模板即点即用 → 社区分享 → 赚积分」为留存环（[一拍 App 商店页
  2026-07](https://apps.apple.com/tw/app/%E4%B8%80%E6%8B%8Dimage/id6785806286)）。
- Civitai 2026 年在 Buzz 积分商业化上越来越激进，功能复杂化引发用户抱怨
  （[somake.ai 中文评测 2026-04](https://www.somake.ai/zh_CN/blog/civitai-review)）。

**对 gaea 的含义**：① 模板必须从「提示词文本 + 尺寸」升级为**带插槽的资产**
（角色槽/风格槽/参考图槽/模型档），否则会被「一键同款」平台在体验上超越；
② ComfyUI 已经替我们教育了用户「工作流 = 应用」，gaea 的 ComfyUI 后端应顺势
提供**参数化工作流收藏/一键套用**（本地不依赖 ComfyHub 账号）；
③ 做「模板/工作流交换」可以，做「公共模型社区」不是 gaea 的牌。

### 1.4 本地侧：技术门槛在降，但「开箱即用」仍是社区最痛的点

- ComfyUI 2026 年动态 VRAM 默认启用（低显存也能跑大模型）、ComfyUI Desktop 官方
  AMD ROCm Windows 支持落地（[Gigazine 2026-03-31](https://gigazine.net/news/20260401-comfyui-dynamic-vram/)；
  [ComfyUI 官方博客 2026-01](https://blog.comfy.org/p/official-amd-rocm-support-arrives)）。
- FLUX.2 Klein 4B Apache 2.0 开放权重、亚秒级、约 13GB VRAM 可跑；FLUX Kontext /
  Qwen-Image-Edit-2511 让本地指令式编辑与局部重绘进入「可部署」区
  （[innfactory 2026-06](https://innfactory.ai/de/ki-modelle/flux/)；
  [Spheron 部署指南 2026-06](https://www.spheron.network/blog/deploy-open-source-ai-image-editing-models-gpu-cloud-2026/)；
  [CSDN Qwen-Image-Edit-2511 教程](https://wenku.csdn.net/doc/9qecqqtwk9iz)）。
- 但社区吐槽依然密集：节点图陡峭、更新快导致教程/工作流失效、「一整天生不出图」、
  本地整合包解压即用是雷区（[cocoloop 讨论 2026-06](https://www.cocoloop.cn/t/topic/5680/21)；
  [PTT AI_Art 2026-06](https://www.pttweb.cc/bbs/AI_Art/M.1781249837.A.5EB)；
  [什么值得买 2026-06-12](https://post.smzdm.com/p/az8mezor/)）。
- MiniMax H3 开源（2026-08）带 ComfyUI 官方工作流与多参考生成（图最多 9 张 +
  视频/音频各 3），但单分区约 135GB、需 SGLang/vLLM-Omni 级部署——**属于云端/远端
  队列档，不属于个人桌面默认档**（[MiniMax 本地部署文档
  2026-08-27](https://platform.minimaxi.com/docs/guides/local-deploy-h3)；
  [太平洋科技 2026-08-16](https://www.pconline.com.cn/zhizao/article/1643334.html)）。

**对 gaea 的含义**：本地档继续做「ComfyUI 之上的省心壳」（模型/LoRA/工作流管理、
统一队列历史、开箱模板），同时**新能力先接云端、后补本地**（编辑、多参考、视频），
不要被单一大模型的本地显存门槛卡住产品节奏。

### 1.5 版权/合规：社区最大噪音，也是本地工具的机会

LiblibAI 的模型版权争议、Civitai 的商用风险提示，说明「公共模型市场」是雷区；
gaea 的本地优先 + 用户自持权重 + 生成参数可溯源，天然规避「平台生成内容版权」争议
（上一轮 r6 调研已立此结论，本轮再次被 LiblibAI 事件验证）。

---

## 2. 现状审计：gaea 绘梦已有什么、差什么

### 2.1 已有（代码事实，v4.97.0 基线）

| 面 | 现状 | 证据 |
|---|---|---|
| 工作台 | 三模式（文生图/图生图/文生视频）+ 三栏画廊工作台 + 队列/历史/灯箱 + 自定义模板 | `frontend/src/pages/ImageGenPage.tsx`、`components/imagegen/*` |
| 引擎 | ComfyUI 本地（krea2/z-image-turbo/flux 文生图；img2img 低 denoise；LTX t2v）+ OpenAI 兼容云/本地（xAI、herdsman、ollama 等）+ GLM | `internal/ai/image_comfyui.go`、`image_openai.go`、`image_glm.go`、`image_backend.go` |
| 角色接线（半程） | 角色库已有 `reference_images/gallery_images` 字段；角色编辑器可「以参考图生成剧照」；灯箱可「设为剧照」 | `internal/characterlib/store.go:162`、`CharacterLibEditor`、`Lightbox.tsx:187` |
| 小说联动（半程） | 章节「生成配图」→ `GenerateSceneIllustration`；书架「生成封面」→ `GaeaGenerateBookCover` | `ChapterPage.tsx:959`、`NovelPage.tsx:139`、`internal/app/chapter_handler.go:128` |

### 2.2 关键断点（下一代的直接战场）

1. **绘梦页不认识角色**：灯箱只能把成图「设为某角色的剧照」，不能「选某角色 →
   带他的参考图 → 生成新场景图」。角色库的参考图资产与绘梦工作台之间是**单向弱连接**。
2. **编辑链路薄弱**：img2img 只有整图低 denoise 重绘（更像重抽不是修图）；
   无指令式编辑（说人话改图）、无蒙版/局部重绘、无「锁定角色/构图只改 X」。
3. **模板不是资产**：模板仍是「提示词 + 尺寸」文本预设，无角色槽/风格槽/参考图槽，
   不能导入导出 ComfyUI JSON（百万级生态无法接入）。
4. **三条生图管线互不相通**：绘梦工作台、章节配图、角色剧照/书封各自为政，
   参数（角色参考/风格/模型）不透传，产物不回流（无统一「画室素材库/章节插图库」）。
5. **模型目录无分层心智**：模型靠引擎列表被动发现 + 少量硬编码默认值；没有
   质量/能力（生图/编辑/多参考/视频）/成本标签，也没有「本地免费 vs 云端按张」的
   透明感；按张计量缺失。
6. **叙事链断层**：单图能力齐备，但「正文 → 名场面列表 → 分镜 → 多图叙事（条漫/
   故事板/整页漫画）」无产品化入口；图生视频只有 t2v（LTX），无 i2v/首尾帧延续。

---

## 3. 下一代定位与目标

### 3.1 定位一句话

绘梦从「生图工具」升格为乐园创作间内的**画室**：以角色资产为锚、以指令编辑为手感、
以章节/场景为上下文，把「角色 → 场景 → 成图 → 改图 → 归档」做成一条连续创作流；
小说配图、书封、剧照、独立创作四个入口共用同一条管线，产物统一回到**画室素材库**。

### 3.2 三支柱（能力目标）

| 支柱 | 目标表述 | 行业对标 |
|---|---|---|
| ① 人设可控 | 角色参考图成为一等输入：选角色即可稳定复现，跨场景/跨批次不跑脸；预留 LoRA 深化 | FLUX.2 多参考 / PuLID-Flux-2 / Sudowrite Visualize / NovelAI V5 |
| ② 图可修改 | 「不满意 → 重抽」降级为「不满意 → 说人话改 / 圈选局部重绘 / 锁定角色只改其它」 | Qwen-Image-3.0-Pro 同模型编辑 / Seedream 5 Pro / Midjourney V8.2 编辑器 |
| ③ 图能成篇 | 章节正文 → 场景/角色识别 → 批量出图 → 故事板/配图/书封，产物进素材库可回溯可重出 | LumenX / 蛙蛙「文→图→动态」/ NovelAI 整页漫画 |

支撑轴：**多后端分层目录**（本地 ComfyUI 免费档 · 云端旗舰按张档 · 编辑/多参考能力
标签 + 成本透明），全 play 空间、与工位零交叉。

### 3.3 可度量目标（成功后看什么）

- **一致性**：角色库选角后「一次成图可用率」≥ 70%（人工验收：脸部/服饰可辨识）；
  参考槽在小说配图/剧照/书封的渗透率 ≥ 60%。
- **编辑闭环**：编辑类操作占绘梦生成行为的 ≥ 20%；同一张图形成「变体簇」沉淀。
- **创作链**：有章节配图的项目占比明显提升；正文到可用配图的平均步数下降 ≥ 50%。
- **资产沉淀**：项目平均「画室素材库」条目数上升；每张图可追溯（模型/参数/角色/
  后端/成本/时间戳）。
- **工程红线**：产物全部落 `.gaea/play/exports`；不新增公共社区/账号体系；
  无「任务/审计/积分」等工位词汇进入乐园文案；每刀独立提交可回退。

### 3.4 非目标（本轮明确不做）

- 不做 LiblibAI/吐司式**公共模型社区、云渲染市场、账号与社交体系**。
- 不自研扩散模型/训练内核；LoRA 训练只做「向导 + 装配本地已有工具」。
- 不新增导航板块；板块名保持「绘梦」，体验代号可称「画室」。
- 不与工位（办公/造价/记忆）打通任何数据流；章节/书封/角色素材只在 play 域流动。
- 不默认接入外部模型下载站/工作流市场（版权与安全红线），仅支持用户显式导入。

---

## 4. 分阶段规划与刀序草案

### 阶段一「画室起手式」（现在，0–2 月，建议 4–6 个小版本）

> 目标：把「一致性可点选、模型可分层、改图能说人话」做出来，全部复用现有后端
> seam，不新开板块。

- **刀 A 画室资产面板（前端为主）**：ImageGenPage 左侧新增「创作资产」区——
  角色槽（选角色自动带参考图/立绘）、模板槽、近期作品；模式入口语义化
  （文生图 / 以图改图 / 参考人设 / 章节配图），为后续能力立骨架。**绑定面零变更**。
- **刀 B 一致性参考槽（后端 ai seam + ComfyUI）**：`ImageGenerationRequest` 增
  `ReferenceImages`（角色/风格/场景多张 + 权重）+ `ReferenceMethod`
  （img2img / ip-adapter / pulid）；ComfyUI 工作流按模型族注入（krea2/z-image 先
  img2img 低 denoise 近似，flux 上 IP-Adapter/PuLID 节点）；不支持参考的后端
  **诚实降级提示**而非静默丢弃。角色库参考图 + 绘梦角色槽在此贯通。
- **刀 C 指令编辑「改图」MVP（云端先行）**：新增编辑请求（图 + 人话指令 + 可选
  保留区域语义），接 **Qwen-Image-3.0-Pro**（同模型 ID 生成 + 编辑）与 GLM-Image
  （若官方编辑端点可用）；前端灯箱/结果卡「改图」按钮 + 原图/新图对照 + 成本显示。
  本地 Qwen-Image-Edit / FLUX Kontext 工作流列为本地档 B 计划。
- **刀 D 章节配图管线 v2（小说 × 画室收口）**：`GenerateSceneIllustration` /
  `GaeaGenerateBookCover` 支持传角色参考图与风格槽；配图产物登记项目「画室素材库」
  （章节插图列表：重生成变体/替换/设为书封素材）；小说侧与绘梦页共用灯箱查看。
- **刀 E 模型目录分层 + 画室消耗（轻量）**：模型卡片带 能力/档位/成本 徽标；
  每次生成记录「张数 × 后端单价」，画室侧折叠面板显示「本月画室消耗」（本地免费
  即显示 0 元），为留存与未来计量铺垫。文案用「画室消耗/创作记录」，不用积分话术。

### 阶段二「画室长出手脚」（下个，3–6 月）

- **蒙版局部重绘 + 扩图 + 抠图**：canvas 圈选/笔刷蒙版 + ComfyUI inpaint 工作流
  （Qwen-Image-Edit / FLUX Kontext / SD inpaint）；透明底抠图（segment/RMBG），
  支持导出 PNG 透明立绘（对齐 NovelAI V5 原生透明卖点）。
- **角色资产 v2**：一键产出角色「设定卡/多姿态套图」；跨图一致性评分
  （face-sim + 参考相似度，低分提示补充参考/重抽）；角色画廊化管理。
- **LoRA 训练向导（实验）**：选角色已有成图 → 打标 → 调用本机 kohya/Inline-Studio
  类工具 → 产物装入 ComfyUI loras 目录自动可挂载；明确标注显存/时长门槛，
  不做云训练。
- **故事板 MVP**：章节 → 场景/人物抽取（复用 chapter/analysis）→ 分镜条目
  （角色 + 动作 + 景别 + 构图提示）→ 按角色参考批量出「角色一致故事板条」，
  可重排/换角/导出；先做 1–2 角色同图实验。
- **模板资产化 v1**：自定义模板升级为带插槽预设；支持导入 ComfyUI JSON
  （白名单节点校验防注入，沿用现有后端工作流构建器纪律）；收藏/版本化。

### 阶段三「画室成叙事工场」（愿景，6–12 月）

- **全本视觉化**：整书名场面扫描 → 批量配图 → 图文版导出（HTML/Markdown/PDF）；
  书封多方案 + 标题文字渲染验证（Qwen-Image 系文字能力）。
- **动态化**：图生视频（i2v/首尾帧/运镜参数）——云端 Wan/qwen 档先行，本地 LTX 档
  跟进；「动态立绘/角色表情集」与轻语联动仅在用户显式允许时评估（不破空间红线）。
- **素材库资产化**：个人作品集 + 版权留痕（AI 标记/参数/来源/许可提示）导出。
- **模板/工作流交换（可选）**：只做用户间模板/工作流文件交换，不做公共生图社区。

---

## 5. 关键取舍与风险

| 风险/取舍 | 判断 | 对策 |
|---|---|---|
| 云端编辑 API 可用性 | Qwen-Image-3.0-Pro 同模型编辑为已验证方向，但端点/参数可能变化 | 刀 C 先做「适配层 + 能力探测」，失败诚实提示；本地编辑 B 计划并行储备 |
| 本地编辑模型显存门槛 | Qwen-Image-Edit 基座 20B 级，8G 卡用户跑不动 | 先云端补体验，本地档等 2511/量化档验证后再默认开启 |
| ComfyUI 更新快、工作流易碎 | gaea 已用「显式工作流构建器映射」而非盲跑任意 JSON | 维持模型 → 工作流显式表；新节点能力小步验证再入 |
| LoRA 训练门槛 | 训练=显存/时间重投入，不适合默认主路 | 定位为「向导化实验特性」，默认主路仍是参考图槽 |
| 版权合规 | 参考图/模型来源是雷区 | 仅用户自有图做参考；不接公共模型下载/工作流市场；模型许可提示 |
| 需求过热（想一次做完） | 三支柱全部一次做 = 巨版不可验收 | 阶段一刀 A–E 每刀独立可发布可回退，按现有 release 纪律走 |

---

## 6. 参考来源（2026-09-04/05 检索，重点页已核实；未核实项已在上文标注）

- Qwen-Image-3.0-Pro：https://www.alibabacloud.com/help/zh/model-studio/image-model 、
  https://cn.aliyun.com/product/news/30190
- GLM-Image：https://docs.bigmodel.cn/cn/guide/models/image-generation/glm-image 、
  https://jnzstatic.cs.com.cn/zzb/htmlInfo/113558.html
- Seedream 5.0 Pro：https://seed.bytedance.com/zh/seedream5_0_pro
- NovelAI V5（2026-08-21）：https://weavai.app/blog/zh-cn/2026/08/29/novelai-diffusion-v5%e4%b8%8a%e7%ba%bf%ef%bc%9a22%e4%ba%ba%e5%90%8c%e6%a1%86%e4%b8%8e%e5%8e%9f%e7%94%9f%e6%8a%a0%e5%9b%be/
- Midjourney V8.2 编辑模型：https://baike.baidu.com/item/Omni-Reference/67326276
- ComfyUI App Mode/App Builder/ComfyHub：https://blog.comfy.org/p/from-workflow-to-app-introducing 、
  https://comfyui.org/zh/from-workflow-to-app-introducing
- ComfyUI Dynamic VRAM / AMD ROCm：https://gigazine.net/news/20260401-comfyui-dynamic-vram/ 、
  https://blog.comfy.org/p/official-amd-rocm-support-arrives
- FLUX.2 / PuLID / 角色一致性：https://github.com/Roxabi/roxabi-factory/issues/419 、
  https://raw.githubusercontent.com/NeverSight/skills_feed/refs/heads/main/data/skills-md/mckruz/comfyui-expert/comfyui-character-gen/SKILL.md
- Qwen-Image-Edit 本地化：https://www.spheron.network/blog/deploy-open-source-ai-image-editing-models-gpu-cloud-2026/ 、
  https://wenku.csdn.net/doc/9qecqqtwk9iz
- LiblibAI 规模与版权争议：https://baijiahao.baidu.com/s?id=1868375417972702192
- Civitai 2026 商业化讨论：https://www.somake.ai/zh_CN/blog/civitai-review
- 网文 AI 插图实况：https://ngabbs.com/read.php?tid=47348466 、
  https://flowpixai.com/ai-art/ai-painting-novel-illustration.html 、
  https://m.yzwb.net/wap/news/4930385.html
- LumenX（小说→动态视频）：https://github.com/alibaba/lumenx
- Sudowrite Visualize：https://docs.sudowrite.com/using-sudowrite/1ow1qkGqof9rtcyGnrWUBS/visualize/4MxYzYduLVPbGceg6zMMCb
- 本地 LoRA 工具：https://github.com/inlineresearch/Inline-Studio 、
  https://github.com/l1934332574-maker/Kohya-LoRA-Tool 、
  https://developer.aliyun.com/article/1760379
- MiniMax H3 本地部署：https://platform.minimaxi.com/docs/guides/local-deploy-h3
- 社区痛点（本地 ComfyUI 门槛）：https://www.cocoloop.cn/t/topic/5680/21 、
  https://www.pttweb.cc/bbs/AI_Art/M.1781249837.A.5EB

> 与内部基线关系：本文修订 `docs/gaea-nextgen-roadmap-2026.md` §6「绘梦」优先级——
> 「指令编辑/局部重绘」「模板插槽参数化」「角色参考图槽」三项**仍未落地**，顺位
> 提前为阶段一；「章节/书封联动 MVP」已在 v4.3 半程完成，阶段一以**参数透传 +
> 素材回流**收口。创作包复扫（2026-08-31）中「GLM-Image 分层 + 角色卡注入 +
> 用量统计」建议也已并入刀 E/刀 B。
