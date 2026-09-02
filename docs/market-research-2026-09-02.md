# 模型中心专项市场调研（2026-09-02，基线 v4.34.0，绑定面 552）

模块制原始稿：`docs/research-2026-09-02/`（desktop-clients / web-apps / local-engines，3 个调研子代理产出）。
本文为合成版：现状盘点 → 竞品发现 → 差距 → 优化方向提案（供拍板）。

## 1. gaea 模型中心现状盘点（代码侦察）

- 10 分区：总览 / 语言模型 / 图片生成 / 语音模型 / 专业模型 / 模型库(Herdsman) / 受控测评 / 检索质量 / 功能绑定 / 引擎管理（本地调度+引擎卡）；右侧统计/资源 Inspector。
- 引擎 8 个内置硬编码（`internal/modelengine/engine.go` 种子 + Key 按 type 硬编码分支）：云端 xai(OAuth)/deepseek/glm(std|coding 双端点)/opencode-go/opencode-zen，本地 ollama/herdsman/cosyvoice(TTS)。`EngineConfig.APIKey` 字段已有但仅内置引擎使用。
- 引擎级能力：BaseURL 编辑（仅本地引擎显示——v4.10 防 Key 粘错框防线）、启停/批量启停、连通测试（延迟+模型数+错误回显，随 engines.json 持久化）、刷新模型列表（OpenAI /models）、每引擎默认模型 + 全局活跃引擎/模型。
- 模型级能力：卡片置顶（modelPrefs）、默认徽标、GLM coding 套餐 alias 诚实注记、本地模型启停；kind 由后端判型（llm/tts/stt/image/embedding/rerank/ocr）。
- 功能绑定：5 功能（chat/novel/office/characterlib/routine）绑 engine+model+enabled，回退全局；routeSource 已有 feature/global/fallback 三态。
- 本地调度：保活 + 启动自动预载（T5-3）；Herdsman 模型库：目录/下载/启动/停止/卸载/统计。
- 统计测评：调用统计（按引擎/模型聚合+趋势+汇率）、成本重排（gaea_cost_rerank）、总览高失败率模型榜、受控测评、检索质量评测、流式探针（gaea_stream_probe）。
- GLM 静态目录 glm_catalog.json 仅 id+kind，无能力/上下文/价格字段；目录热更新/GLM 积分口径/成本仪表 v0 为 08-31 复扫已立欠账（计费三件套）。

## 2. 竞品关键发现（三路汇总）

**桌面客户端（Cherry Studio/Chatbox/NextChat/ChatWise/BoltAI）**
- 自定义服务商（OpenAI 兼容 BaseURL+Key，任意添加）是 4/5 家标配——gaea 全家都没有，是最大硬缺口。
- 自动拉取模型列表 4/5 家有——gaea 已有（refreshEngineModels），不落后。
- 能力标签：仅 Cherry 有（手工标 函数/推理/视觉/网络），全行业无自动检测；价格/上下文展示普遍缺——gaea 若做元数据体系即超行业平均。
- 用量统计仅 Cherry 有——gaea StatsSection 已领先。
- 参数粒度天花板：BoltAI 生成级（top-p/top-k/penalties）+ Context Profiles；NextChat 面具绑定模型+参数；Chatbox「未设置」兼容推理模型。

**Web 端（LobeChat/Open WebUI/AnythingLLM/LibreChat）**
- LobeChat 能力标记一体语法 `<128000:vision:fc:reasoning>`（上下文+能力）；多 Key random/turn 轮询。
- Open WebUI 多连接聚合 + Prefix ID 区分同名模型；模型级 capabilities 开关。
- LibreChat tokenConfig 按模型声明 prompt/completion/context/cache 价格 + maxContextTokens——价格目录的成熟同构物；参数体系最细（端点级 addParams/dropParams + 每模型覆盖）。
- AnythingLLM Model Router：规则式按消息路由 provider+model——与 gaea 欠账「自动路由 v1」同类，验证方向。
- 四家全部未见故障转移/健康巡检（仅 LobeChat Key 轮询）——主动健康巡检+故障转移是行业空白，且 gaea 已有 status 持久化缓存与 fallback 路由语义，边际成本低。

**本地引擎（Ollama/LM Studio/Jan/llama.cpp/vLLM/GPT4All）**
- 下载前硬件评估/推荐量化：仅 Jan（兼容性检查器）与 LM Studio 做了——明确产品机会点；gaea Herdsman 目录下载目前无显存评估。
- llama.cpp 已内置 /health、/metrics、`-fit` 自动适配显存——Herdsman 底座可低成本透出健康与适配信息。
- 生命周期管理（TTL 自动卸载/keep_alive/多模型并行驻留）是共性基线——gaea 保活+预载已有，不落后。
- 磁盘占用/批量删除释放空间（Jan）是低成本补课项。

## 3. 差距与机会（按竞品对照）

| 维度 | 行业水位 | gaea 现状 | 结论 |
|---|---|---|---|
| 自定义引擎 | 桌面标配 | 8 内置硬编码 | 最大缺口 |
| 模型能力/价格/上下文元数据 | 零散手工（Cherry 描述自填、LibreChat yaml） | 仅 id+kind | 可做超行业：目录化+诚实展示 |
| 目录热更新 | 竞品内置服务商由官方维护 | glm_catalog.json 静态（GLM 积分制已证失真风险） | 已立欠账，必做 |
| 故障转移/健康巡检 | 行业空白（含 4 Web 端） | 手动测试+失败率榜 | 差异化机会 |
| 模型级试调用 | 无（Cherry 仅服务商级） | 引擎级测试+流式探针在测评区 | 低成本挪/复用 |
| 参数粒度 | BoltAI/LibreChat 最细 | 无模型级参数 | 次优先（可绑功能而非模型） |
| 本地硬件预检 | Jan/LM Studio 仅两家 | 无 | 机会点，依赖 Herdsman 透出 |
| 用量统计/成本 | 仅 Cherry 有统计 | 已领先（趋势+汇率+成本重排） | 保持，接真实价格即 v0 仪表 |

## 4. 优化方向提案（供拍板，建议刀序）

**A. 自定义引擎（OpenAI 兼容自定义服务商）——补课刚需，建议第一刀**
用户可添加任意 OpenAI 兼容端点（中转站/硅基流动/私有网关/新版 OpenRouter）：名称+BaseURL+Key+启停，引擎卡可编辑/删除。后端 `engines.json` 本就是通用 EngineConfig，新增用户引擎条目持久化 + `keyIfNeeded` 回退 `engine.APIKey` + 约 4~6 个新绑定方法；前端引擎管理加表单。红线：v4.10「云端引擎不露地址框」防线延伸——自定义引擎地址框与 Key 框分离 + URL 格式校验（防 Key 粘进地址框，旧事故复发）。体量：一整刀。

**B. 模型目录热更新 + 能力/价格元数据体系——计费三件套的根，建议第二刀**
ModelInfo 扩展 capabilities(vision/tools/reasoning)/context_length/price(in/out) 字段；glm_catalog.json 升级为带版本号远程热更新+本地缓存兜底（积分制口径随版本走）；模型卡诚实展示能力/上下文/价格徽标；成本仪表 v0 用真实价格算（替代仅汇率换算的现态）。同构 LibreChat tokenConfig + LobeChat 能力标记，且是自动路由 v1 的前置（目标函数需要价格数据）。体量：一整刀，后端为主。

**C. 健康巡检 + 故障转移 v0——差异化机会，建议第三刀**
后台定时巡检已启用引擎（复用 status 持久化缓存），总览「需要关注」升级为主动告警（连续失败→徽标+通知）；调用失败时按功能绑定回退链自动切换引擎（可开关，默认关），routeSource 已有 fallback 语义可承接。行业空白=护城河叙事。体量：中刀（巡检半刀可先发）。

**D. 本地模型磁盘/健康补课（小件，可搭车）**
Herdsman 模型库卡补磁盘占用展示与「删除释放 N GB」提示；透出 llama.cpp /health 到引擎卡（若 Herdsman 版本支持）。体量：半刀搭车项。

**不建议现在做**：模型级参数体系（BoltAI/LibreChat 级）——gaea 场景是功能绑定制，参数应绑功能而非裸模型，等自动路由 v1 一并设计；多 Key 轮询（单用户桌面场景价值低）；本地量化档选择 UI（依赖 Herdsman 底座改造，另立项）。

刀序建议：A → B → C（+D 搭车）；与 08-31 复扫「模型中心计费三件套」欠账的承接关系：B=三件套本体升级版，A 是复扫未覆盖的新增刚需，C 为差异化增量。
