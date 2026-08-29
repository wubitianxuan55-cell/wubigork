# gaea 3.0 愿景规划 — AI 图像生成市场与「绘梦板块」调研报告（r6）

> 调研对象：gaea 的「绘梦板块」——AI 图像生成工作台，多后端（本地 ComfyUI 跑 Flux / Z-Image-Turbo / Krea2 + 云端 xAI 等）、LoRA/风格预设、生成队列/历史/灯箱，与小说板块（角色立绘）和办公板块（方案配图）协作。
> 调研时间：2025—2026 年公开信息
> 调研方式：web_search 中英文混合检索（每主题 2—4 轮，共约 40 次查询），结论均附来源 URL；量级数据以公开报道为准并标注年份。
> 关联报告：r1（harness 生态）、r2（桌面中文市场）、r3（agent 记忆编排）、00-vision-thesis（愿景总纲）；gaea 内部设计：docs/2026-08-15-gaea3-vision-roadmap.md 、docs/2026-08-15-gaea3-architecture-design.md 、docs/gaea2/2026-08-13-绘梦板块重构设计.md 。

---

## 0. 摘要（先读这节）

1. **中文 AI 图像市场是"大厂免费流量战 + 垂直社区闷声赚钱"的双层结构**。上层：字节即梦、快手可灵等以 App 积分制收割大众流量（豆包 MAU 1.72 亿、可灵月活破 1200 万）；下层：LiblibAI 以"模型社区 + 在线生图"一年融资 1.3 亿美元（2025-10，当年中国 AI 应用最大单笔），吐司 TusiArt 做在线 ComfyUI，说明**垂直创作者付费意愿真实存在**（来源见 1.4、1.5）。
2. **Midjourney 的"霸权"只在英文市场**。约 40 人、无 VC、年收入 5—6 亿美元，但形态是 Discord bot + 网页编辑器，中文用户需翻墙 + Discord 注册；2025 年中文横评普遍结论是"国内商业场景选即梦"——**中文市场没有 Midjourney 心智，这是窗口**（见 1.1）。
3. **开源与本地生图已经成熟到"8G 显存跑 Flux 12B"**。Flux.1（Black Forest Labs，2025-11 估值 32.5 亿美元）、SD 3.5、字节 Seedream 4.0（2025-09 开源）均可本地运行；ComfyUI 是本地生图的"标准件"，Civitai 模型社区从 2024 年初 8.4 万模型增长到 2025 年数十万级（见 2.2、2.3）。
4. **"本地 + 云端"双引擎在图像生成上已是事实标准，但缺少产品化入口**。LiblibAI/吐司把 ComfyUI 搬上云按积分收费，桌面端 ComfyUI 免费但门槛高；**没有中文产品把"本地免费用量 + 云端质量模型 + 统一队列/历史/灯箱 + LoRA 角色一致性"整合成一个创作工作台**。这是绘梦板块的直接机会（见 4.1）。
5. **"生图嵌入创作流"已被验证但各自孤立**：网文侧有番茄 AI 插图（平台内封闭）、阅文 Claw（2026-03 发布）、NovelAI（海外订阅制证明付费意愿）；文档侧有 WPS AI 智能配图、Notion AI 生图（2026-03）、Gamma（a16z 投 6800 万美元，估值 21 亿美元）。**没有产品打通"写作→立绘→全书插图→封面→文档配图"的连续流**（见 3.1—3.5、4.2）。
6. **付费模式对照**：可灵 2024-10 单月流水破千万（会员制）；LiblibAI ARR 破 3 亿（2026-06）；Gamma 约 50 人年收入近 7 亿元人民币量级；美图 2025 经调整净利润 9.65 亿元（+64.7%）。**图像创作工具的订阅/积分商业模式在中文市场已被反复验证**（见 1.4、1.5、3.3）。
7. **反面教材同样重要**：百度文心一格 2025 年 6 月底停止服务（并入文小言）；阿里妙鸭相机 2026-05 传出团队解散。**大众娱乐型、单点爆款型的生图产品生命周期短，嵌入工作流的工具型产品反而活得更久**（见 1.4、1.5）。
8. **对 gaea 绘梦板块的结论**：不做"又一个大厂生图 App"，做"个人创作工作台"——本地 ComfyUI 内核（免费/无审查/可控/LoRA）+ 云端模型兜底（质量/速度）+ 生成队列/历史/灯箱 + 与小说（角色立绘一致性）和办公（方案配图）的数据联动。目标用户：网文作者、内容创作者、方案制作者，而非泛大众（见 4.3）。

---

## 1. AI 图像生成市场格局

### 1.0 市场总览（2024—2026 演变）

- **阶段一（2022—2023）：MJ 定义"图像生成 = 订阅制"**。Midjourney 以 Discord 社区+订阅制跑通"质量即付费"；Stable Diffusion 开源引爆本地部署；中国跟进文心一格、通义万相、混元、6pen 等一批工具（综述：https://flowpixai.com/ai-art/ai-big-tech-painting-case.html ）。
- **阶段二（2024）：国产 App 化 + 视频化**。可灵/即梦把战场从"生图"拉到"生视频"；文生图变成 App 内功能；LiblibAI、吐司等垂直平台崛起（见 1.4、1.5）。
- **阶段三（2025—2026）：开源大模型开源潮 + 商业化收敛**。Flux 估值 32.5 亿美元、Seedream 4.0/即梦 4.0 开源、可灵月活 1200 万、LiblibAI ARR 3 亿；同时文心一格停服、妙鸭解散、Sora 关停（AI 生成视频冰火两重天：https://blog.csdn.net/techforward/article/details/161004139 ）——**市场从"比谁模型好"进入"比谁嵌入的工作流深、离钱近"**。
- 市场数据口径：全球 AI 绘画设计市场报告（中国区域增长分析）：https://dxpress.gelonghui.com/p/2959775 、https://dxpress.gelonghui.com/p/5395678 ；AI 绘画行业报告（168report）：https://www.168report.com/reports/10616183/ai-painting-generation-software ；国内 AI 绘画市场现状综述：https://flowpixai.com/ai-news/ai-art-market-status.html 。

### 1.1 海外第一梯队：Midjourney

**形态与商业模式**
- 2022 年 7 月成立，拒绝 VC 融资，全员约 40 人，靠 Discord 社区运营起家；2026 年盘点给出"年收入 5—6 亿美元、人均创收约 300 万美元"的量级（https://megaoneai.com/blog/midjourney-revenue-business-model/ 、https://dev.to/andrew-ooo/how-midjourney-generates-3-million-per-employee-with-zero-vc-funding-and-no-marketing-budget-594l 、https://closedloop.sh/blog/discord-community-product-feedback-goldmine ）。
- 商业模式为**纯订阅制、无免费额度**：基础档约 10 美元/月、标准档约 30 美元/月，按 GPU 分钟计费，档位越高并发与保密度越高（megaoneai 同文）。
- 产品形态：Discord bot 指令 + Web 编辑器 + API（企业）；2023 年底推出 Web 版降低门槛，但**始终没有独立中文版**（百度百科词条：https://baike.baidu.com/item/midjourney/62827850 ）。
- 统计聚合站口径（用户/收入/份额）：https://morphed.app/stats/midjourney-statistics 。

**中文用户心智**
- 使用路径 = 科学上网 + Discord 注册 + 英文提示词，**对中文大众用户是"教程里的神、日常的麻烦"**。
- 2025 年中文横评普遍结论：出图质量 MJ 仍属第一梯队，但"中文提示词、价格、访问、合规"上国产平台（即梦等）胜出——什么值得买《Midjourney和即梦AI哪个更强？国内商业场景到底该选谁》（https://post.smzdm.com/p/a26ozmkq/ ）、PHP 中文网横评（https://m.php.cn/faq/2507183.html ）、1ai.net 五平台对比（https://www.1ai.net/37059.html ）、AI 绘画教程选型（https://flowpixai.com/tutorials/domestic-ai-painting-tutorial.html ）。
- 社区讨论（LINUX DO《佬友们现在AI画图用哪个模型比较好》）显示中文用户日常首选即梦/本地 Flux，MJ 仅作风格参照（https://linux.do/t/topic/713070/13 ）。
- 小红书/自媒体大量"AI 绘画平台大扒皮"测评进一步消解 MJ 光环（https://www.xiaohongshu.com/discovery/item/68a6d3e5000000001d03ba10 ）。
- **结论**：MJ 在中文市场是"口碑资产 > 实际使用"，其心智正在被国产平台与本地开源共同接管。

### 1.2 Flux / Black Forest Labs（本地生图新标准）

**公司形态**
- Black Forest Labs 由原 Stability AI 核心成员创立，2024-08 发布 Flux.1 系列（dev/schnell 开源、pro 走 API），随后进入 Midjourney"劲敌"叙事；2025-11/12 完成 **3 亿美元 B 轮、估值 32.5 亿美元**，欧洲 AI 独角兽（https://news.crunchbase.com/ai/image-generator-europe-unicorn-black-forest-labs-raise/ 、https://finance.yahoo.com/news/black-forest-labs-raises-300m-140813374.html 、https://sifted.eu/articles/black-forest-labs-300m-series-b 、https://tech.eu/2025/12/01/black-forest-labs-secures-300m-series-b-at-325b-valuation/ ）。
- FT 中文网将其与 MJ 并列视为"挑战中美 AI 领跑者的欧洲公司"（http://app001.ftchinese.com/interactive/242956 ）。

**开源与本地运行**
- Flux.1 dev 以非商用许可开源、schnell 为 Apache 2.0（模型卡：https://huggingface.co/wangkanai/flux-dev-fp8 ）；12B 参数文本编码器偏重，但社区已跑通 **8G 显存部署**（NGA《8G显存本地部署12B参数的Flux模型》：https://ngabbs.com/read.php?tid=41092735 ）。
- 显存需求聚合站：Flux.1 Dev 在 ComfyUI/Diffusers 下的各档位（https://willitrunai.com/image-models/flux-1-dev ）；12GB（RTX 3060 12G）训练 FLUX LoRA 的完整指南（https://github.com/marhensa/kohya-config ）；SDNQ 量化插件省 50—75% 显存（https://github.com/EnragedAntelope/comfyui-sdnq ）。
- 中文社区"Flux 本地部署 + 云端远程使用"教程成为主流内容（cpolar：https://www.cpolar.com/blog/stable-diffusion-3-5-is-deployed-locally-and-remotely-to-generate-ai-images ）；云端部署指南（GigaGPU 教程：https://gigagpu.com/flux1-deployment-guide/ ）。
- **对 gaea 的意义**：Flux 是当前"本地质量天花板"之一，绘梦板块本地后端选 Flux（配合 Z-Image-Turbo / Krea2 等轻量模型做速度档）符合社区主流技术栈。

### 1.3 Stable Diffusion / ComfyUI 开源生态

**生态规模**
- Stable Diffusion 2022-08 开源引爆个人生图，形成"SD 1.5 → SDXL → SD 3.5"三代模型谱系（部署资料：https://www.cpolar.com/blog/stable-diffusion-3-5-is-deployed-locally-and-remotely-to-generate-ai-images ）。
- **Civitai** 成为全球最大模型社区：2024 年初第三方数据洞察统计约 **8.4 万个模型**（NGA《C站8.4万个模型的数据洞察》：https://bbs.nga.cn/read.php?tid=39002962 ，小红书转载：https://www.xiaohongshu.com/discovery/item/6595511b000000001e0054df ）；官方《2025 Review》（https://civitai.com/articles/24324/2025-review ）；第三方统计聚合 Stable Diffusion Statistics 2026（https://morphed.app/stats/stable-diffusion-statistics ）。
- **ComfyUI**：节点式工作流工具，GitHub 最活跃的生成式 AI 项目之一；官方桌面版 ComfyUI Desktop 与云端 ComfyUI Cloud 并存（https://raw.githubusercontent.com/Comfy-Org/Comfy-Desktop/main/README.md 、OSS 数据：https://ossaihub.com/tool/comfyui/ 、2025-08 生态复盘 The ComfyUI Revolution：https://42.uk/promptus-lite/august-2025/ ）。
- 中文"1000 张工作流"类资料与安装教程极多，是中文创作者学习本地生图的主要入口（腾讯云开发者：https://cloud.tencent.cn/developer/article/2378928 、CSDN 新手安装：https://blog.csdn.net/ice829/article/details/143250864 ）。
- 中文独立评测对 Civitai 前景的看法（内容审核/商业化的两面性）：https://www.aixq.cc/31147.html ；模型网站排行榜（FlowPix 主观榜）：https://www.flowpixai.com/ai-tools/ai-painting-model-websites.html 。

**生态对中文用户的意义**
- SD/ComfyUI 生态是"免费、模型多、可商用选项多、无审查、可离线"的代名词；工具型评测《开源本地AI绘画工作流：可控、免费、可进化的Stable Diffusion实践指南》（https://blog.csdn.net/weixin_33324197/article/details/161268935 ）和 ToolChamber《Why Pros Are Moving to Open-Source》（https://toolchamber.com/stable-diffusion-3-5-review/ ）都指向专业用户迁移开源的动机。
- **但原生 ComfyUI 面向工程师**：节点图、依赖冲突、模型管理（下载/路径/版本）、显存配置——个人创作者并不直接使用它。**这中间层正是 LiblibAI / 吐司 / 绘梦要填的位置**（见 2.1、4.1）。

### 1.4 中国阵营：字节即梦 / 快手可灵 / 阿里通义万相 / 腾讯混元 / 百度文心一格

**字节即梦（Dreamina）**
- 即梦 AI 是字节的图像/视频生成 App（海外 Dreamina），深度绑定豆包生态；2025 年迭代到即梦 3.0/4.0，底层文生图模型 **Seedream 4.0 于 2025-09 前后开源**，可本地/Docker 部署（CSDN《2025年9月9日首发！即梦 4.0 接口开发全攻略：开源 + Docker 部署》：https://blog.csdn.net/wwwzhouhui/article/details/151628773 ；技术解读：http://cnnetsun.cn/news/1662526.html ）。
- 字节在 AI 内容侧的投入量级：AI 短剧/短内容生态 90 天做到约 6.5 亿美元收入规模（海外视角报道：https://www.ainchina.com/blog/china-ai-drama-revolution-bytedance-650m-empire-2026/ ）；"字节的 AI 时间表：从豆包到即梦"复盘（https://www.sohu.com/a/1057621245_116457 ）。
- 用户量级参照：QuestMobile 2025-08 显示豆包 MAU 1.72 亿（中国原生 AI App 第一）、DeepSeek 1.45 亿、Kimi 967 万（https://www.bianews.com/news/details?id=223880 ）；2025-10 全国 AI 应用移动端月活突破 7 亿（人民网：http://sc.people.com.cn/BIG5/n2/2025/1030/c346366-41395926.html ）。
- 即梦与"AI 短剧风口"绑定讨论（界面新闻：https://www.jiemian.com/article/14412206.html 、创业邦：https://m.cyzone.cn/article/833124.html ）；Seedance 2.0 排队问题暴露"免费积分 + 高峰排队"的体验代价（每日经济新闻：https://www.stcn.com/article/detail/3658074.html ）。
- 商业模式：免费积分 + 会员订阅（积分制），无独立客户端订阅心智，靠豆包全家桶导流；图像生成对它是"引流功能"，视频/短剧才是营收重点。

**快手可灵（Kling）**
- 2024-06 上线；2024-08 官方口径：用户超 160 万、生成视频超 1600 万条（新京报：https://www.bjnews.com.cn/detail/1724735582129538.html ）；2024-10 程一笑：**单月流水超千万、9 月月活超 150 万**（https://www.cnstock.com/commonDetail/318721 、https://www.bjnews.com.cn/detail/1732109282129033.html 、界面：https://www.jiemian.com/article/12013718.html ）。
- 到 2026-01 前后，可灵 AI **月活突破 1200 万**（格隆汇/中金在线异动快讯：https://www.gelonghui.com/news/5154470 、http://3g.cnfol.com/sc_stock/gushizhibo/20260121/31960084.shtml ）；快手 2025 Q1 营收 326 亿元（https://huacheng.gz-cmc.com/pages/2025/05/27/SF13895079bed232d4e66d4a96bef3cf.html ）。
- 行业心智："国产 AI 视频第一梯队、落地最实、离钱最近"，但 2026 年被批评"可灵不灵了"——视频侧心智被即梦抢走，图像侧未独立成产品（界面新闻：https://www.jiemian.com/article/14016972.html ）。
- 商业模式：快影 App 内会员订阅（月度/年度），是国产"会员制跑通"的样板；对 gaea 的启示：**先跑通单点（视频/生图）再谈心智，心智会被更强模型快速稀释**。

**阿里通义万相**
- 通义万相 2023-07 发布：文生图/风格化/图像编辑，集成于通义 App/网页端；阿里系 AI 应用 2025 年登声量榜（https://finance.eastmoney.com/a/202601303636937224.html 、QuestMobile 2025 榜单：https://news.ifeng.com/c/8rCHlF2LXGa ）。
- 定位为**大模型 App 内的功能**，无独立生图社区与生态；BAT+字节 AI 绘图产品案例综述见 FlowPix（https://flowpixai.com/ai-art/ai-big-tech-painting-case.html ）；国内 AI 绘画网站汇总（FlowPix）：https://www.flowpixai.com/ai-art/domestic-ai-painting-sites.html 。

**腾讯混元**
- 混元生图（2023-10 起）集成于腾讯元宝/微信生态；官方技术百科列出的应用场景含**网文配图**、设计素材、电商等（腾讯云开发者技术百科：https://cloud.tencent.com/developer/techpedia/2486/19454 ）。
- 腾讯元宝 2025 年靠买量冲榜（月活第二梯队），但**混元生图没有独立产品心智**；元宝 2026 年买量腰斩（36氪 5 月 AI 月报：https://m.36kr.com/p/3328589608134153 ）——买量换来的流量不可持续。

**百度文心一格（反面案例）**
- 文心一格 2022 年上线，是百度独立的 AI 绘画网站/App；**2025 年 6 月底停止独立服务，功能并入文小言**（百度百科词条：https://baike.baidu.com/item/文心一格/63170732 ；文心一言 4.0 更名文小言背景：https://cloud.baidu.com/article/3552341 ）。
- 教训：**大众向、独立、免费的生图站点在"大厂免费流量战"里没有生存位**；独立工具必须绑定创作工作流或垂直人群才有护城河。

### 1.5 中国垂直社区与工具：LiblibAI / 吐司 / 美图

**LiblibAI（哩布哩布，运营主体演语科技）**
- 2023 年成立，定位 AI 图像生成平台 + 模型/工作流社区（在线生图、LoRA 训练、模型托管、一键同款）；创始人"90 后字节前高管"（百度百科：https://baike.baidu.com/item/LiblibAI/65434067 ）。
- **2025-10 完成 1.3 亿美元 B 轮，当年中国 AI 应用最大单笔融资**，投资方含蚂蚁（https://www.cs.com.cn/ssgs/gsxw/202510/t20251023_6518895.html 、界面：https://m.jiemian.com/article/13504148.html 、https://m.jiemian.com/article/13505364.html 、证券日报：http://m.zqrb.cn/gscy/qiyexinxi/2025-10-24/A1761269742719.html ）。
- 2026-06 报道：**B+ 轮近 3 亿美元、估值超 200 亿元、ARR 破 3 亿**（21 世纪经济报道：https://www.21jingji.com/article/20260625/herald/10ad11c2e12d70d1207cc79f6874e2e5.html 、https://m.jiemian.com/article/14609845.html 、KuCoin 转载：https://www.kucoin.com/zh-hant/news/flash/ai-application-layer-unicorn-liblibai-secures-300m-in-b-round-valued-at-over-20b ）。
- **对 gaea 的意义**：LiblibAI 证明"中文创作者为模型/在线生图付费"成立，但其形态是**平台（网页+App）而非个人工作台**，不嵌入写作/文档流；且重心是"模型电商"而非"创作流"。

**吐司 TusiArt（吐司 AI）**
- 国内"在线 ComfyUI 生图 + 模型分享 + 一键同款工作流"平台（CSDN 介绍：https://blog.csdn.net/2401_84760719/article/details/142636181 、导航收录：https://www.toolmage.com/zh-hans/tool/tusiart/ 、https://prompt.cn/sites/9445.html 、App Store 数据页：https://app.sensortower.com/overview/6458545837?country=CN ）。
- 形态上把"ComfyUI 工作流"商品化：用户传工作流、平台云端跑、按积分收费——**验证了"云端化 ComfyUI"需求，但同样不绑定创作场景**。

**美图（Meitu）**
- 美图 2025 年经调整净利润 9.65 亿元（+64.7%）、海外 MAU 破 1 亿，靠 AI 转型（https://gna.org.gh/2026/03/meitu-2025-annual-results-adjusted-net-profit-surges-64-7-yoy-to-a-record-rmb-965-million-driven-by-ai-transformation/ 、https://www.streetinsider.com/GetNews/Meitu’s+2025+Globalisation+Milestone/26235098.html 、投中网《400亿美图，靠AI重生了》：https://m.pedaily.cn/news/553826 ）。
- 反例：阿里妙鸭相机（2023 年 9.9 元 AI 写真爆款）2026-05 传出团队解散（https://finance.eastmoney.com/a/202605213745403187.html 、https://news.qq.com/rain/a/20260523A03K6Q00 ）——**单点爆款生命周期短，工具+工作流才持久**。

### 1.6 商业模式对比表

| 产品 | 形态 | 商业模式 | 量级（公开报道，标年份） |
|---|---|---|---|
| Midjourney | Discord/网页生成器 | 纯订阅 $10 起/月，无免费档 | 年收入 5—6 亿美元（2025-26 估算）、约 40 人 |
| Flux / Black Forest Labs | 开源模型 + API | dev/schnell 开源、pro API 计费 | B 轮 3 亿美元、估值 32.5 亿美元（2025-11） |
| Stable Diffusion / ComfyUI | 开源软件生态 | 免费；周边平台收费 | Civitai 8.4 万模型（2024 初）→ 数十万级（2025） |
| 即梦（字节） | App/网页 | 免费积分 + 会员 | 豆包 MAU 1.72 亿（2025-08）；Seedream 4.0 开源 |
| 可灵（快手） | App（快影） | 会员订阅 | 月活 150 万（2024-09）→ 1200 万（2026-01）；单月流水破千万（2024-10） |
| 通义万相 / 混元 | 大模型 App 内功能 | 随会员/免费 | 无独立生图心智 |
| 文心一格（百度） | 独立站点/App | 免费 | 2025-06 停服并入文小言 |
| LiblibAI | 平台（社区 + 在线生图） | 积分/会员/B 端 | 1.3 亿美元 B 轮（2025-10）；ARR 破 3 亿（2026-06） |
| 吐司 TusiArt | 平台（在线 ComfyUI） | 积分 | 社区平台，无公开营收 |
| 美图 | App 矩阵 | 订阅 | 2025 经调整净利 9.65 亿元 |

（表格内各数字来源见 1.1—1.5 各条 URL）

### 1.7 中文用户心智小结

- **大众心智 = "免费积分 + 会员"**：即梦/可灵教会中文用户"每天免费生几张、想快点就买会员"；直接付费订阅（如 MJ 式 $30/月）在中文大众市场不可行，但在创作者（LiblibAI 用户）中可行。
- **质量心智正在迁移**：2025 年中文评测普遍认为即梦 4.0/Seedream 与 MJ 差距收窄甚至局部反超（https://m.php.cn/faq/2507183.html 、https://post.smzdm.com/p/a26ozmkq/ 、https://www.1ai.net/37059.html ）；本地派（Flux/SDXL）以"免费+可控"守住专业用户（https://toolchamber.com/stable-diffusion-3-5-review/ ）。
- **"没有中文 Midjourney"是结构性事实**：大厂做"流量入口"（即梦/可灵），垂直平台做"模型/素材电商"（LiblibAI/吐司），**缺一个面向个人创作全流程（写作/文档/图）的集成工作台**。
- 对 gaea 的定位启发：**不做流量入口，不做模型电商，做"创作流中间的图引擎"**——用户因为要"给小说配图/给方案配图"而来，而不是因为"想玩 AI 生图"而来。

### 1.8 格局判断：大厂 / 垂直 / 开源 / 海外独立产品的四方博弈

- **大厂（字节/快手/阿里/腾讯/百度）**：把生图当"获客功能"，模型迭代快、免费积分烧钱换月活；除 Seedream 等少数开源动作外，模型不开放本地化、不开放创作工具层（见 1.4 各条）。
- **垂直平台（LiblibAI/吐司）**：做"模型电商 + 云生图"，商业化最健康（LiblibAI ARR 破 3 亿，2026-06），但离"创作流"远——用户找模型/生完图即走，无留存场景、无写作/文档联动（见 1.5 各条）。
- **开源生态（SD/Flux/ComfyUI/Civitai）**：技术与内容最丰富（模型数十万、工作流无数），但"用户体验"断层——节点图、依赖、显存把大众挡在门外（见 2.1—2.4）。
- **海外独立产品（MJ/NovelAI/Gamma）**：以订阅制验证"为质量/为场景付费"，但中文服务缺位、不接中文创作流（见 1.1、3.2、3.3）。
- **对绘梦板块的定位**：站在这四方的交点上——用大厂的模型能力（云端 API）、垂直平台的模型生态（LoRA/工作流）、开源的免费内核（本地 ComfyUI）、独立产品的订阅心智（创作工具付费），拼成"个人创作工作台"。**四方的缝隙就是绘梦的地盘**。

---

## 2. 「本地 ComfyUI 工作流」生态

### 2.1 ComfyUI 的形态分层

- **桌面版**：ComfyUI Desktop（官方，Comfy-Org，2024 年末起分发）：https://raw.githubusercontent.com/Comfy-Org/Comfy-Desktop/main/README.md ；此前社区打包的 desktop 仓库已归档并入官方（https://www.ghtrending.com/project/Comfy-Org/desktop ）。
- **管理器/插件**：ComfyUI-Manager 等社区节点生态是"一键装模型/装节点"的事实标准；中文教程把"ComfyUI 安装 + 1000 张工作流"做成爆款资料（https://cloud.tencent.cn/developer/article/2378928 、CSDN 新手安装：https://blog.csdn.net/ice829/article/details/143250864 ）。
- **在线版**：ComfyUI Cloud（官方按需付费）、国内吐司/哩布哩布"上传工作流→云端跑"模式（见 1.5）。
- **量级参照**：GitHub 星标/趋势榜持续名列前茅（2025-08 社区复盘 The ComfyUI Revolution：https://42.uk/promptus-lite/august-2025/ ；OSS 聚合：https://ossaihub.com/tool/comfyui/ ）；Steam 级分发暂无公开数据，但中文教程生态密度（CSDN/腾讯云/NGA/小红书）远超同类工具。
- **对 gaea 的意义**：ComfyUI 是"生图内核"，绘梦板块应**把 ComfyUI 当作后端引擎封装**（headless API 模式），而不是让用户直接操作节点图——用户看到的是"预设/风格/LoRA 按钮"。

### 2.2 本地生图的现实门槛（显存 / 模型 / 速度）

- **显存档位（实测/官方资料）**：
  - 8GB：可跑 SD 1.5、低量化 Flux（NGA 实测《8G显存本地部署12B参数的Flux模型》：https://ngabbs.com/read.php?tid=41092735 ）；
  - 12GB（RTX 3060 12G）：可跑 Flux.1 dev 且能训练 FLUX LoRA（kohya 配置指南：https://github.com/marhensa/kohya-config ）；
  - 24GB（4090）：SDXL/Flux 全精度 + 大批量无压力（Flux.1 Dev 各档显存需求表：https://willitrunai.com/image-models/flux-1-dev ）；
  - 量化/蒸馏进一步压门槛：SDNQ 省 50—75% 显存（https://github.com/EnragedAntelope/comfyui-sdnq ）。
  - **结论：8G 起步、12G 舒适、24G 自由；2020 年后的中高端显卡即可覆盖主流用例，中文用户"为本地生图升级显卡"的消费习惯已形成**（NGA/贴吧大量装机帖佐证）。
- **模型大小**：SD 1.5 约 2—4GB、SDXL 约 7GB、Flux.1 dev 约 12—24GB（含文本编码器）、Seedream 4.0 开源权重需按官方说明部署（CSDN 部署攻略：https://blog.csdn.net/wwwzhouhui/article/details/151628773 ）。**下载一次模型 5—20GB 是常态，磁盘管理（多模型并存、版本冲突）是真实痛点**——绘梦板块的"模型中心"可以解决。
- **速度**：消费卡（4060—4090）单张 512—1024px 图约 3—30 秒/张（视模型与步数）；部署指南（https://gigagpu.com/flux1-deployment-guide/ 、https://www.cpolar.com/blog/stable-diffusion-3-5-is-deployed-locally-and-remotely-to-generate-ai-images ）。**速度上本地足以支撑"批量生图 + 灯箱挑选"交互，但高峰批量时云端更划算**——这正是"本地为主、云端兜底"路由策略的依据。

### 2.3 工作流社区（Civitai / OpenArt / LiblibAI / 吐司）

- **Civitai**：全球最大模型+工作流社区，8.4 万模型（2024 初，NGA 数据洞察：https://bbs.nga.cn/read.php?tid=39002962 ）→ 2025 官方年度回顾（https://civitai.com/articles/24324/2025-review ）；下载量级第三方统计（https://morphed.app/stats/stable-diffusion-statistics ）。内容以二次元/写实 LoRA、工作流分享为主。
- **OpenArt**：面向"工作流模板 + 在线生图"的海外平台（模型/模板市场，与 ComfyUI 深度绑定），并入"AI 绘画模型网站"中文榜单（https://www.flowpixai.com/ai-tools/ai-painting-model-websites.html ）。
- **LiblibAI / 吐司**：中文版"模型电商 + 在线生图"，见 1.5；吐司主打"ComfyUI 工作流一键同款"（https://blog.csdn.net/2401_84760719/article/details/142636181 ）。
- **工作流内容形态**：以"文生图基础流、图生图/局部重绘流、角色一致性流（参考图 + LoRA + IPAdapter）、放大修复流（Upscale）、风格化流"五大类为主；中文教程池极深（https://cloud.tencent.cn/developer/article/2378928 ）。
- **对 gaea 的意义**：工作流社区证明"模板化/预设化"是降低门槛的唯一路径——**绘梦板块的 LoRA/风格预设本质是"把工作流变成用户可点选的按钮"**，且 gaea 可在本地把社区工作流"导入为预设"。

### 2.4 个人用户本地生图的比例与动机

- **动机（公认四点）**：
  1. **免费**：无单张/积分成本，生多少不心疼（CSDN 本地工作流指南：https://blog.csdn.net/weixin_33324197/article/details/161268935 ）；
  2. **无审查**：本地跑开源模型没有平台内容审核（社区舆论《本地部署开源免费比付费还强》：https://www.toutiao.com/article/7669301579530371627/ ）；
  3. **可控**：模型/LoRA/参数/种子全自主，可复现（ToolChamber 专业用户迁往开源的评测：https://toolchamber.com/stable-diffusion-3-5-review/ ）；
  4. **隐私/离线**：本地权重自持，生成内容不出本机。
- **比例（推断，无权威官方数）**：三点间接证据——①Civitai 下载量级达数十亿次（https://morphed.app/stats/stable-diffusion-statistics ）；②ComfyUI 本地生态长期霸榜 GitHub 趋势与中文教程池（https://42.uk/promptus-lite/august-2025/ 、https://cloud.tencent.cn/developer/article/2378928 ）；③LiblibAI/吐司"云化 ComfyUI"能收上费（见 1.5），说明"想用 ComfyUI 但不想折腾本地"的人群同样巨大。
  **推断：中文个人创作者中"本地派"与"云积分派"并存；本地派是少数但高价值（专业、重度、长期、愿为体验付费）**。
- 反面声音：本地生态的安装/依赖/工作流复杂度劝退大众（Diffusion Bee 与 Comfy 桌面版的"易用性对比"长期是社区话题：https://cask.news/compare/diffusionbee-vs-comfy ；iOS 本地生图 App（ImageLab）做"本地 AI 图像"的轻量尝试：https://apps.apple.com/cn/app/imagelab-本地人工智能图像/id6756362014 ）——**易用性封装 = 产品机会**。

### 2.5 典型本地生图架构拆解（绘梦可复用的工程现实）

- **软件栈**：Python + PyTorch + ComfyUI（或 diffusers），通过 headless 模式/API 暴露生成接口；ComfyUI 原生支持 API 调用与队列管理（Comfy-Org/Comfy-Desktop README：https://raw.githubusercontent.com/Comfy-Org/Comfy-Desktop/main/README.md ）。
- **硬件现实**：单卡消费级 GPU 即可；显存不足时量化（FP8/GGUF/NF4）与分块加载（SDNQ 省 50—75% 显存：https://github.com/EnragedAntelope/comfyui-sdnq ）；多模型并存靠磁盘（单个模型 5—20GB，见 2.2）。
- **生成流程**：提示词（含 LoRA/负面词/采样参数）→ 队列 → 出图 → 后处理（放大/降噪/局部重绘）→ 落盘；参数可复现（种子固定）。
- **角色一致性工作流**：参考图 + LoRA + IPAdapter/角色分区是社区标准做法（Auto-NovelAI-Refactor：https://github.com/zhulinyv/Auto-NovelAI-Refactor ）；一张角色 LoRA 在 12G 卡上训练约 10—30 分钟（kohya 指南：https://github.com/marhensa/kohya-config ）。
- **云端兜底**：无卡用户/高峰批量走 API（xAI/Seedream 等），路由策略按"本地空闲优先、超时切云端"。
- **对绘梦的意义**：这些能力全部已开源成熟，**产品化工作在于封装**（预设、队列、历史、灯箱、成本显示），而非从零实现生图引擎——开发重心应放在"工作台"而非"生成器"。

### 2.6 模型选择决策表（本地后端候选）

| 模型 | 显存门槛 | 权重大小 | 质量档 | 许可（商用） | 适配场景 |
|---|---|---|---|---|---|
| SD 1.5 | 4—6G | 2—4GB | 入门 | 大部分开源 | 轻量/风格化 |
| SDXL | 6—8G | 约 7GB | 中 | 开源（商用条款见官方） | 通用/写实 |
| SD 3.5 | 8—12G | 约 7GB | 中高 | 有商用条款 | 通用 |
| Flux.1 dev | 8—12G（量化）/16G+（全精度） | 12—24GB | 高 | 非商用 | 质量优先 |
| Seedream 4.0（开源） | 按官方文档 | 较大 | 高 | 开源（Apache 2.0） | 中文语义/质量 |
| Z-Image-Turbo / Krea2（gaea 内部轻量档） | 低 | 小 | 速度档 | —（内部模型） | 快速预览/批量 |

（来源：Flux 显存表 https://willitrunai.com/image-models/flux-1-dev ；8G 实测 https://ngabbs.com/read.php?tid=41092735 ；SD 部署 https://www.cpolar.com/blog/stable-diffusion-3-5-is-deployed-locally-and-remotely-to-generate-ai-images ；Seedream 部署 https://blog.csdn.net/wwwzhouhui/article/details/151628773 ；Flux LoRA 训练 https://github.com/marhensa/kohya-config ）

---

## 3. 图像生成 × 写作 / 文档的集成产品

### 3.0 需求侧证据：为什么"配图"是创作刚需而非锦上添花

- **网文侧**：番茄/阅文的"AI 插图"功能上线即被作者使用（小红书体验帖：https://www.xiaohongshu.com/discovery/item/64e78303000000000a01b44f ），"即梦+醒图自制小说封面"教程成为爆款内容（https://m.toutiao.com/article/7485656464057745959/ ）——封面+插图是网文作者的真实高频需求（新书发布要封面、章节推广要图）。
- **内容创作侧**：公众号/小红书配图是运营刚需，AI 图像类工具在国内 AI 应用下载榜上长期是主力（QuestMobile 2025 榜单：https://news.ifeng.com/c/8rCHlF2LXGa 、AI 应用榜：https://m.163.com/dy/article/KN47JSH1051481US.html ）。
- **办公侧**：演示/方案文档"无图不成文"，WPS AI 智能配图功能上线即主推（https://bbs.wps.cn/topic/47219 ），Gamma 靠"AI 生成演示内容（含配图）"做到 21 亿美元估值（https://www.chaincatcher.com/article/2218927 ）——配图是被验证的付费场景。
- **结论**："生成一张图"是低频娱乐，"给创作内容配图"是高频工作；**嵌入创作流的生图产品吃的是后者的心智，这正是绘梦板块与"生图 App"的本质区别**。

### 3.1 网文配图与小说封面

- **平台内功能（封闭）**：
  - 番茄小说有"AI 插图"功能（2023 年起作者可用，小红书体验帖：https://www.xiaohongshu.com/discovery/item/64e78303000000000a01b44f ）；
  - 阅文"作家助手妙笔版"（火山引擎驱动）提供 AI 辅助创作（百度百科：https://baike.baidu.com/item/作家助手妙笔版/63227250 、品玩《行业首家！阅文部署DeepSeek，"作家助手"升级三大辅助创作功能》：https://www.pingwest.com/w/302142 ）；
  - 2026-03 阅文推出作家专属 AI 创作产品 **Claw**（https://finance.eastmoney.com/a/202603153672452246.html 、北京市科委转载：https://kw.beijing.gov.cn/xwdt/kcyx/xwdtyqqy/202603/t20260316_4557563.html 、腾讯新闻：https://news.qq.com/rain/a/20260315A072J800 ）。
  - **特点：都在平台内，作者离开平台就用不了；且插图功能多为"配图"级，无角色一致性管理**。
- **独立封面工具（单点）**：
  - 即梦+醒图自制小说封面教程成为爆款内容（2025-03，https://m.toutiao.com/article/7485656464057745959/ ）；
  - 在线"AI 小说封面生成器"大量出现（EYUAI：https://www.eyuseo.com/zh/image-generator/novel-cover-generator 、pagepop 教程：https://www.pagepop.cn/learn/236/ 、https://www.pagepop.cn/learn/476/ ）。
  - **特点：只解决封面一张图，不解决全书插图与角色一致性**。
- **开源写作系统内置生图**：
  - NovelFlow-AI（GitHub）宣称"AI 网文一条龙：题材策划、人物设定、多 Agent 写作、封面生成、自动审稿"（https://github.com/myshisheng/NovelFlow-AI ）；
  - 实验性项目"先用大模型生成提示词再生图/视频"（https://github.com/zhoutingunl/patched-but-still-broken/issues/27 ）。
  - **特点：个人开发者证明"写作→提示词→生图"管线可行，但工程粗糙、无云端、无队列/历史管理**。
- **中文 AI 写作工具群（配图能力现状）**：马良写作（中文长篇作者工具，把 NovelAI 列为竞对但主业写作：https://maliangwriter.com/compare/maliang-vs-novelai/ ）、墨星写作（https://www.sparkx.zone/tools/mxxz-ai.html ）、彩云小梦（续写向，见 r2 报告）——**普遍无成熟内置生图，这是绘梦板块的插入点**。

### 3.2 角色立绘与一致性（NovelAI 与中文对标）

- **NovelAI（海外标杆）**：订阅制（约 $10/$25 档），二次元绘图 + 小说场景强绑定，靠"立绘/角色一致性 + 小说配图"心智立住（中文教程：https://tw.cyberlink.com/blog/ai-image-generator/5596/novel-ai-alternative 、评测：https://blog.pixai.art/en/novelai-review-features-pricing-pros-cons-and-is-it-worth-using-in-2026/ 、配图教程：https://m.php.cn/faq/2047251.html 、AppFollow 数据：https://apps.appfollow.io/ios/book-cover-maker-novelart/6444536767 ）。**证明"为角色图/小说配图付费"在创作者中成立**。
- **开源角色一致性技术栈已完全成熟**：Auto-NovelAI-Refactor 支持批量文生图/局部重绘/角色参考/反推 tag/超分降噪（https://github.com/zhulinyv/Auto-NovelAI-Refactor ）；LoRA + 参考图（IPAdapter/角色分区）是社区标准做法——**绘梦板块的"角色库"可直接复用**。
- **中文对标缺失**：国内没有"中文版 NovelAI 绘图侧"的独立产品；中文写作者只能拼凑"番茄插图（平台内）+ LiblibAI 模型 + 手动整理"。
- **与 gaea 的联动**：gaea 小说板块已有角色库设计（docs/2026-08-07-角色库档案卡重设计-design.md 、docs/2026-08-07-角色详情补齐与剧照.md 、docs/2026-08-07-角色卡详情重设计.md ）——绘梦板块做"角色→LoRA/参考图→立绘/剧照"管线，是自然延伸。

### 3.3 文档 / PPT / 方案配图（WPS AI / Notion / Gamma）

- **WPS AI（金山办公，国内办公心智第一）**：
  - 演示组件提供"AI 生成素材/智能配图"，一键给 PPT 配图（WPS 官方论坛：https://bbs.wps.cn/topic/47219 、https://bbs.wps.cn/topic/94447 、https://bbs.wps.cn/topic/86119 ）；
  - "AI 设计室/设计助手"可 AI 生图并定制模板（https://bbs.wps.cn/topic/87254 、https://m.it168.com/article_6873834.html 、中国日报：http://ex.chinadaily.com.cn/exchange/partners/82/rss/channel/cn/columns/sz8srm/stories/WS6752c6b4a310b59111da77c3.html ）。
  - **特点：通用素材级配图，无"方案/业务图"专业化，且后端不可控（用户无法选模型/风格/LoRA）**。
- **Notion AI 生图（海外知识库标杆）**：2026-03 官方发布"用 Notion AI 生成图片"（Notion 官方发布页：http://joon.blue/nb/releases/2026-03-09 、AI Dev Setup 分析：https://aidevsetup.com/insider/notion-ai-adds-image-generation-what-cms-builders-need-to-know ）——**知识/文档产品内置生图的趋势确认**。
- **Gamma（AI 演示标杆）**：约 50 人团队、AI 生成 PPT（含 AI 配图/版式），2025 年完成 **6800 万美元 B 轮（a16z 领投）估值 21 亿美元**，媒体称"52 个人用 AI 做 PPT，年赚 7 亿"（https://www.finet.hk/newscenter/print_content/6912a05a800d457d2ca0e95c 、https://www.chaincatcher.com/article/2218927 、https://m.163.com/dy/article/KFHA9KLH0511805E.html 、https://m.sohu.com/a/955008071_122014422/ 、Oman Observer 英文稿：https://www.omanobserver.om/article/1179511/scitech/technology/gamma-a-powerpoint-for-the-ai-era-raises-68-million ）。**证明"文档配图作为 AI 演示体验的一部分"能撑起独角兽估值**。
- Gamma vs Notion AI 完整对比（2026）：https://services-ia.lafrenchtech-grandeprovence.fr/zh/ai-bijiao/gamma-vs-notion-ai 。
- **对 gaea 的意义**：办公板块的"方案配图"应比 WPS 更进一步——支持**选风格/选后端/出示意图与架构图模板**，并复用方案上下文（大纲/章节）自动生成配图描述（对齐办公板块：docs/superpowers/plans/2026-08-05-方案编写板块-P6-排版导出.md 、docs/superpowers/plans/2026-08-05-方案编写板块-P3-大纲与撰写引擎.md ）。

### 3.4 素材库与版权

- 生成素材进入素材库的版权/合规是 To B 场景的关键：国内"千库/摄图/视觉中国"类平台对 AI 素材有明确合规口径；多模态 AI 应用商业化报告提到美图、万兴、福昕加速 AI 商业化（民生证券：https://www.sgpjbg.com/labelsyh/duomotaiaiyingyongqianjing.html ）。
- 海外社区对"AI 素材进图库"的版权争论持续（SD 生态统计页有相关内容：https://morphed.app/stats/stable-diffusion-statistics ）。
- **对 gaea 的意义**：办公板块"方案配图"输出应默认标记 AI 生成并保留生成参数（可追溯），避免版权争议；本地后端（用户自持模型权重）天然规避"平台生成内容版权"争议；素材库侧做"我的生成"（灯箱）而非"公共图库"。
- 另注意：gaea 已存在 DREAM_WRITE_POLICY 与 ADULT_MODE 政策（docs/DREAM_WRITE_POLICY.md 、docs/ADULT_MODE.md），绘梦板块的"无审查"仅限于技术能力，产品层面仍需遵循既有政策。

### 3.5 「生图嵌入创作流」产品案例汇总表

| 创作流 | 产品 | 嵌入方式 | 局限 |
|---|---|---|---|
| 网文平台 | 番茄 AI 插图、阅文 Claw/妙笔 | 平台内配图/辅助创作 | 封闭、离台即失、无角色一致性 |
| 独立写作 | NovelAI、NovelFlow-AI | 小说配图/立绘/封面生成 | 二次元向或工程粗糙 |
| 办公/演示 | WPS AI 智能配图、Notion AI、Gamma | 文档/PPT 内一键配图 | 通用素材级，后端不可控 |
| 设计平台 | LiblibAI、吐司、OpenArt | 模型/工作流社区 + 在线生图 | 不绑定写作/文档流 |
| （空白） | **gaea 绘梦板块** | **本地+云端多后端、LoRA 角色库、队列/历史/灯箱，联动小说与办公** | — |

### 3.6 相邻场景：AI 漫画 / 动态立绘 / 封面电商

- **AI 漫画/条漫**：漫画脸描述生成、同人创作、游戏立绘、轻小说配图等场景的落地文章（CSDN：https://blog.csdn.net/weixin_31974443/article/details/157008190 ）；AI 短剧风口（即梦/可灵，见 1.4、1.0）带动"立绘 → 视频"的资产需求。
- **3D/角色资产化**：Meshy 等工具把"角色艺术 → 3D 模型"做成独立产品（https://www.meshy.ai/zh-Hant/3d-tools/book-character-generator ）——角色资产化是长期方向，与 gaea 角色库天然衔接。
- **封面/立绘"约稿替代"市场**：AI 小说封面在线工具与 NovelArt 等封面 App 说明"封面/立绘"是高频付费场景（EYUAI：https://www.eyuseo.com/zh/image-generator/novel-cover-generator 、AppFollow：https://apps.appfollow.io/ios/book-cover-maker-novelart/6444536767 ）。
- **对绘梦的意义**：角色立绘是"小说 → 漫画/短剧/3D/封面电商"资产链的起点；gaea 若把立绘资产与角色库绑定，可为后续"剧照/分镜/封面导出"留扩展位。

---

## 4. 结论：多后端 + 嵌入创作工作流的中文市场空白与机会

### 4.1 空白一：多后端（本地 ComfyUI + 云端）没有消费级产品入口

- 现状拼图（每一块都已存在，但互不相通）：
  - ① 本地 ComfyUI：免费但门槛高（节点/依赖/显存/模型管理，见 2.2、2.4）；
  - ② 云端 ComfyUI（吐司/LiblibAI）：按积分收费、质量与免费额度受限（见 1.5）；
  - ③ 国产大厂 App（即梦/可灵）：只给"官方模型 + 积分"，**不给用户自己的本地模型/LoRA**（见 1.4）；
  - ④ 海外 MJ/Flux pro：纯云端订阅、无本地（见 1.1、1.2）。
- **没有任何产品把"本地算力（免费、无限次、离线、可放任意开源模型与 LoRA）+ 云端模型（无显卡用户兜底、高质量、快速）"统一成一个队列/历史/灯箱工作台**。这就是绘梦板块的差异化底座：本地为主、云端兜底、自动路由、用户无感切换后端。
- 量级机会参照：可灵月活 150 万→1200 万的爬升（2024—2026，见 1.4）说明中文 AI 图像/视频创作用户盘子是**千万级**；LiblibAI ARR 破 3 亿（2026-06，见 1.5）说明垂直创作者付费成立；全国 AI 应用移动端月活破 7 亿（2025-10，http://sc.people.com.cn/BIG5/n2/2025/1030/c346366-41395926.html ）说明人群基本盘足够大。

### 4.2 空白二：生图没有真正"嵌入创作工作流"

- **写作侧**：平台功能封闭（番茄/阅文），独立工具单点（封面生成器）或二次元向（NovelAI）；**没有"从章节文本一键产出全书插图 + 角色立绘一致性 + 封面导出"的连续流**（见 3.1、3.2）。
- **文档侧**：WPS/Notion/Gamma 都是"通用配图"，**没有面向方案/投标文档的"业务示意图、架构图、配图风格统一"能力**，且后端不可控（见 3.3）。
- **素材侧**：生成的图没有进入可复用的个人素材库；灯箱/历史/风格预设（LoRA）是强需求，但只有 LiblibAI 这类平台在做，**且不与写作/办公场景联动**（见 3.4）。
- **机会表述**：绘梦板块 = 小说板块的"立绘/插图引擎" + 办公板块的"方案配图引擎" + 独立的"生图工作台"（多后端、队列/历史/灯箱、LoRA/风格预设），三个入口共用一套资产与后端路由。**中文市场没有竞品同时覆盖这三者**（LiblibAI 只有平台、WPS 只有文档、NovelAI 只有小说、即梦只有大众流量）。

**竞品定位矩阵（谁已经占了哪些格子）**：

| 维度 | 本地免费 | 云端质量 | 多后端统一 | 角色一致性 | 嵌入写作 | 嵌入文档 |
|---|---|---|---|---|---|---|
| 即梦/可灵 | ✗ | ✓ | ✗ | 弱（无角色库） | 平台内封闭 | ✗ |
| LiblibAI/吐司 | ✗ | ✓（积分） | ✗ | 弱 | ✗ | ✗ |
| ComfyUI 本地 | ✓ | ✗ | ✗ | ✓（需手动） | ✗ | ✗ |
| NovelAI | ✗ | ✓ | ✗ | ✓ | ✓（封闭） | ✗ |
| WPS/Notion/Gamma | ✗ | ✓ | ✗ | ✗ | ✗ | ✓（通用配图） |
| **绘梦（目标）** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

（各格依据见 1.4、1.5、2.1—2.4、3.1—3.4；**没有任何现成产品占满一行**，这是机会的直观证明。）

### 4.3 对 gaea 绘梦板块的具体建议（按优先级）

1. **后端架构：本地优先 + 云端兜底 + 显式路由**。默认路由本地 ComfyUI（Flux/SDXL/Z-Image-Turbo/Krea2 等），无显卡或高峰期走云端 API（xAI/即梦 Seedream 等）；队列与历史统一落盘，用户可查看"哪张图哪个后端、多少积分/时间"——成本卡片可复用办公板块成本库设计（docs/market-research-2026-08-office-cost-cards.md 、docs/market-research-2026-08-office-cost-routing.md ）。
2. **角色一致性 = LoRA + 参考图，做"角色库"**：从小说板块角色卡一键创建/选择 LoRA 与参考图，全书插图保持同一人设（对齐 gaea 角色库：docs/2026-08-07-角色库档案卡重设计-design.md 、docs/2026-08-07-角色详情补齐与剧照.md 、docs/2026-08-07-角色卡详情重设计.md ）。这是中文市场最缺、NovelAI 已验证付费的环节。
3. **嵌入而非附加**：小说章节内"选中文本→生成插图"、办公方案内"选中段落→生成配图/示意图"，产物自动进素材库（灯箱），可拖入文档导出（对齐办公板块排版导出管线：docs/superpowers/plans/2026-08-05-方案编写板块-P6-排版导出.md ）。
4. **免费额度策略**：本地生成不扣积分（电费即成本），云端按积分计——这是"本地+云端"模式相对纯云端平台的获客与留存杠杆；也是对"大厂免费积分战"的差异化回应。
5. **模型/预设中心**：管理本地模型下载（5—20GB/个的现实痛点，见 2.2）、LoRA 库、风格预设（工作流模板化，见 2.3）；支持导入 Civitai/LiblibAI 工作流为预设。
6. **合规底线**：生成内容标记 AI、保留参数可追溯；本地权重自持规避平台版权争议；遵循既有 DREAM_WRITE_POLICY 与 ADULT_MODE 政策（docs/DREAM_WRITE_POLICY.md 、docs/ADULT_MODE.md）。

### 4.4 风险与提醒

- **模型许可证**：Flux.1 dev 非商用、SD 3.5 有商用条款、Seedream 4.0 开源可商用——本地后端默认面向个人创作，商用输出需提示用户核对模型许可（Flux 模型卡：https://huggingface.co/wangkanai/flux-dev-fp8 、SD 部署资料：https://www.cpolar.com/blog/stable-diffusion-3-5-is-deployed-locally-and-remotely-to-generate-ai-images ）。
- **云端 API 依赖**：xAI 等 API 价格与可用性波动（Seedream/Seedance 排队之痛已有先例：https://www.stcn.com/article/detail/3658074.html ）——多后端路由本身即对冲，但需做成本缓存与失败降级。
- **大厂挤压**：即梦/可灵把"生图"做成免费功能（见 1.4），绘梦不能与它们拼大众流量，必须赢在"工作流嵌入 + 本地可控 + 角色一致性"。
- **参考标杆的寿命教训**：文心一格停服、妙鸭解散、Sora 关停（见 1.4、1.5、1.0）——**工具型、嵌入创作流的产品才有长期性**；避免做成"又一个 AI 生图小工具"。
- **技术迭代风险**：模型代际（SD 3.5 → Flux → Seedream → 4.0+）半年一换，绘梦板块的内核（ComfyUI 后端 + 多模型抽象）天然抗迭代，这是选"多后端"而非"绑定单一模型"的又一理由。

### 4.5 目标用户画像与功能规格建议

- **用户画像（按优先级）**：
  1. **网文作者**（起点/番茄/晋江生态，个人创作者）：要立绘一致性、章节插图、封面导出；付费意愿中等（会员制）；
  2. **内容创作者/自媒体**（公众号/小红书/视频封面）：要风格统一、批量出图、灯箱管理；
  3. **方案制作者**（gaea 办公板块用户）：要方案配图、示意图、风格统一、可导出；
  4. **生图爱好者（本地派）**：要模型管理、LoRA、工作流导入。
- **功能规格（对齐现有绘梦面板设计：docs/superpowers/plans/2026-08-05-绘梦面板重设计.md 、docs/superpowers/specs/2026-08-05-绘梦面板重设计-design.md 、docs/gaea2/2026-08-13-绘梦板块重构设计.md ）**：
  - **后端路由**：本地 ComfyUI（默认）+ 云端 API（兜底），队列/历史/灯箱统一，每张图记录"后端/耗时/成本"；
  - **角色库**：小说角色卡 ↔ LoRA/参考图 ↔ 立绘/剧照，全书一致性；
  - **预设中心**：风格预设、LoRA 预设、工作流导入（兼容 Civitai/LiblibAI 导出格式）；
  - **嵌入点**：小说章节"选中文本 → 生成插图"、办公方案"选中段落 → 生成配图"；
  - **成本显示**：本地（电费口径）与云端（积分口径）分列，支持限额与切换；
  - **合规**：AI 生成标记、参数可追溯、遵循 DREAM_WRITE_POLICY / ADULT_MODE（docs/DREAM_WRITE_POLICY.md 、docs/ADULT_MODE.md）。
- **商业模型（可选，供规划参考）**：个人免费（本地算力）+ 云端积分（可选充值）+ 进阶会员（LoRA 训练、批量队列、多后端优先级）；参照可灵会员（单月流水破千万）与 LiblibAI 订阅（ARR 破 3 亿）的量级空间（见 1.4、1.5）。

---

## 5. 参考来源清单（按主题）

### 主题一：市场格局
- Midjourney 收入与商业模式（2026 盘点）：https://megaoneai.com/blog/midjourney-revenue-business-model/
- Midjourney 统计（2026）：https://morphed.app/stats/midjourney-statistics
- Midjourney Discord 社区复盘：https://closedloop.sh/blog/discord-community-product-feedback-goldmine
- Midjourney 百度百科：https://baike.baidu.com/item/midjourney/62827850
- BFL B 轮 3 亿美元、估值 32.5 亿：https://news.crunchbase.com/ai/image-generator-europe-unicorn-black-forest-labs-raise/ 、https://finance.yahoo.com/news/black-forest-labs-raises-300m-140813374.html 、https://sifted.eu/articles/black-forest-labs-300m-series-b
- Flux.1 Dev 显存档位：https://willitrunai.com/image-models/flux-1-dev
- 8G 显存跑 Flux：https://ngabbs.com/read.php?tid=41092735
- 即梦/Seedream 4.0 开源部署：https://blog.csdn.net/wwwzhouhui/article/details/151628773
- 豆包 MAU 1.72 亿（QuestMobile 2025-08）：https://www.bianews.com/news/details?id=223880
- 字节 AI 时间表：https://www.sohu.com/a/1057621245_116457
- 即梦与 AI 短剧：https://www.jiemian.com/article/14412206.html
- 可灵月活/流水：https://www.cnstock.com/commonDetail/318721 、https://www.bjnews.com.cn/detail/1732109282129033.html 、https://www.bjnews.com.cn/detail/1724735582129538.html
- 可灵月活 1200 万（2026-01）：https://www.gelonghui.com/news/5154470
- 可灵不灵了？（2026 复盘）：https://www.jiemian.com/article/14016972.html
- 通义/混元案例综述：https://flowpixai.com/ai-art/ai-big-tech-painting-case.html
- 混元生图应用场景：https://cloud.tencent.com/developer/techpedia/2486/19454
- 文心一格停服（百度百科）：https://baike.baidu.com/item/文心一格/63170732
- LiblibAI 1.3 亿美元 B 轮：https://www.cs.com.cn/ssgs/gsxw/202510/t20251023_6518895.html 、https://m.jiemian.com/article/13504148.html
- LiblibAI B+ 轮/估值/ARR：https://www.21jingji.com/article/20260625/herald/10ad11c2e12d70d1207cc79f6874e2e5.html
- 吐司 TusiArt：https://blog.csdn.net/2401_84760719/article/details/142636181
- 美图 2025 年报：https://gna.org.gh/2026/03/meitu-2025-annual-results-adjusted-net-profit-surges-64-7-yoy-to-a-record-rmb-965-million-driven-by-ai-transformation/
- 妙鸭解散：https://finance.eastmoney.com/a/202605213745403187.html
- 中文横评（MJ vs 即梦）：https://post.smzdm.com/p/a26ozmkq/ 、https://m.php.cn/faq/2507183.html 、https://www.1ai.net/37059.html
- 全球/中国 AI 绘画市场报告：https://dxpress.gelonghui.com/p/2959775 、https://www.168report.com/reports/10616183/ai-painting-generation-software

### 主题二：本地 ComfyUI
- ComfyUI Desktop：https://raw.githubusercontent.com/Comfy-Org/Comfy-Desktop/main/README.md
- ComfyUI 星标/趋势：https://ossaihub.com/tool/comfyui/ 、https://42.uk/promptus-lite/august-2025/
- 安装/工作流资料：https://cloud.tencent.cn/developer/article/2378928
- Civitai 8.4 万模型洞察：https://bbs.nga.cn/read.php?tid=39002962
- Civitai 2025 Review：https://civitai.com/articles/24324/2025-review
- SD 生态统计（2026）：https://morphed.app/stats/stable-diffusion-statistics
- 本地生图动机/指南：https://blog.csdn.net/weixin_33324197/article/details/161268935
- 专业用户迁往开源：https://toolchamber.com/stable-diffusion-3-5-review/
- FLUX LoRA 12G 训练：https://github.com/marhensa/kohya-config
- SDNQ 量化省显存：https://github.com/EnragedAntelope/comfyui-sdnq
- 本地 vs 云端部署 Flux：https://www.cpolar.com/blog/stable-diffusion-3-5-is-deployed-locally-and-remotely-to-generate-ai-images 、https://gigagpu.com/flux1-deployment-guide/

### 主题三：写作/文档集成
- 番茄 AI 插图：https://www.xiaohongshu.com/discovery/item/64e78303000000000a01b44f
- 作家助手妙笔版：https://baike.baidu.com/item/作家助手妙笔版/63227250
- 阅文 Claw（2026-03）：https://finance.eastmoney.com/a/202603153672452246.html
- 阅文部署 DeepSeek：https://www.pingwest.com/w/302142
- NovelAI 教程/评测：https://tw.cyberlink.com/blog/ai-image-generator/5596/novel-ai-alternative 、https://blog.pixai.art/en/novelai-review-features-pricing-pros-cons-and-is-it-worth-using-in-2026/
- NovelFlow-AI（开源网文一条龙）：https://github.com/myshisheng/NovelFlow-AI
- Auto-NovelAI-Refactor（批量/角色参考）：https://github.com/zhulinyv/Auto-NovelAI-Refactor
- 即梦+醒图封面教程：https://m.toutiao.com/article/7485656464057745959/
- WPS AI 配图/素材：https://bbs.wps.cn/topic/47219 、https://bbs.wps.cn/topic/94447 、https://bbs.wps.cn/topic/87254
- Notion AI 生图（2026-03）：http://joon.blue/nb/releases/2026-03-09
- Gamma 融资/收入：https://www.finet.hk/newscenter/print_content/6912a05a800d457d2ca0e95c 、https://www.chaincatcher.com/article/2218927 、https://m.163.com/dy/article/KFHA9KLH0511805E.html
- 多模态 AI 应用商业化（民生证券）：https://www.sgpjbg.com/labelsyh/duomotaiaiyingyongqianjing.html

---

## 6. 调研方法与局限

- **方法**：web_search 中英文混合检索约 40 次查询，覆盖 2024—2026 年公开报道、官方文档、社区数据洞察、产品官网；每条关键结论附来源 URL。
- **局限**：
  1. 国内产品用户/收入数据多来自媒体报道（券商研报、界面/新京报/21 世纪等），无官方审计口径，量级以"约/破/超"表述并标注年份；
  2. 本地生图用户比例无权威统计，以 Civitai 下载量、GitHub 趋势、云平台收费三组间接证据推断并明确标注"推断"；
  3. 部分海外数据（如 Midjourney 收入）为第三方估算，存在口径差异；
  4. 调研截至时点的 2026 年新动态（模型代际、融资、产品关停）可能未覆盖，落地前建议复核关键数字。

*报告完。调研基于 2025—2026 年公开信息，量级数据均标注年份并附来源；未验证的社区推断（如本地生图比例）已明确标注为推断。*
