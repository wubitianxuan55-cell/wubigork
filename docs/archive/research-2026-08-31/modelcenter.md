# 模块 4：模型中心（引擎接入/路由/成本）市场调研复扫

## 4. 模型中心（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

**国产 API 生态：旗舰变贵、廉价/免费档扩容、缓存与峰谷计费普及。** DeepSeek 定价页现以 V4 系列为主（V4-Flash-0731、V4-Pro-0813、V4-Flash-Vision-Exp），1M 上下文 / 384K 输出，输入缓存命中价约为未命中的 1/31，非高峰时段再五折，并提供 Anthropic 兼容端点（https://api-docs.deepseek.com/quick_start/pricing）。智谱 GLM 已迭代到 GLM-5.3/5.2/5.1，GLM-5.3-Flash 促销价 $0.075/$0.25（至 2026-09-09），GLM-4.7-Flash、GLM-4.5-Flash、GLM-4.6V-Flash 完全免费，缓存输入另有约 0.19× 的专价（https://docs.z.ai/guides/overview/pricing）；GLM-5.2 主打 1M 无损上下文与长程 Coding Agent（https://docs.bigmodel.cn/cn/guide/models/text/glm-5.2）。Kimi 旗舰 K3 为 1M 上下文、缓存命中 $0.30 / 未命中 $3.00 / 输出 $15.00，另有 K2.7 Code 与 K2.6（视觉），Moonshot V1 于 2026-08-31 全平台下线，域名迁至 platform.kimi.ai（https://platform.kimi.ai/docs/pricing/chat、https://platform.kimi.ai/docs/pricing/chat-k3）。阿里百炼自身已成多厂商全模态货架：qwen3.8-max/3.7-plus 之外还上架 deepseek-v4-pro、kimi-k3、glm-5.3，并统一提供生图（qwen-image-3.0-pro、wan2.7-image-pro）、视频、TTS/ASR/Realtime、embedding/rerank（https://help.aliyun.com/zh/model-studio/models）。

**编码套餐订阅化。** GLM Coding Plan 改为积分制：Lite/Pro/Max 三档按「5 小时积分 + 周积分」计（2,000/12,000/28,000 与 10,000/60,000/140,000），非高峰时段积分五折抵扣、宣称最高省 92%；套餐覆盖 GLM-5.3 与 GLM-5.3-Flash，调用旧名自动切换，支持 Claude Code、OpenClaw、OpenCode、TRAE、CodeBuddy 等工具（https://docs.bigmodel.cn/cn/coding-plan/overview）。

**桌面客户端全面 Agent 化。** Cherry Studio 8 月连发 v2.0.x 多版：三种 agent 会话运行时（pi、DeepSeek Harness「dsh」、Claude Agent SDK）、MCP prompts/resources 进入输入框、MCP-over-HTTP（/v1/mcps）、供应商目录免发版热更新、统一模型连通性检测、用量分析改用 ECharts、本地模型 DirectML/CoreML 加速（https://github.com/CherryHQ/cherry-studio/releases）。LobeChat 品牌升级为 LobeHub，定位「下一代 Agent harness」「7×24 小时 Agent 运营」（https://lobehub.com/zh）。AnythingLLM 主打 on-device、本地私密（https://anythingllm.com/）。LM Studio 搜索摘要显示其出现「Bionic 工作与代码智能体」及云模型、LM Link 等入口（官网抓取超时，细节未核实）。开源个人 AI 助理 OpenClaw 成为现象级产品（https://openclaw.ai），智谱推出「AutoClaw（澳龙）——国内首款一键安装的本地 OpenClaw 客户端」，内置 50+ Skills 与 AutoGLM 浏览器能力（https://www.zhipuai.cn/zh/about）。

### 范式迁移（上轮调研以来的变化）

1. **计费模型从「按量单价」变「订阅积分 + 峰谷时段 + 缓存命中」三维**：GLM Coding Plan 积分制与自动换模型、DeepSeek 峰谷五折、各家缓存专价（上轮基线均无）。
2. **prompt 前缀缓存计费成头部引擎标配**：Anthropic 读取 0.1×、写入 1.25×（5 分钟）/2×（1 小时），自动与显式断点并存（https://platform.claude.com/docs/en/build-with-claude/prompt-caching）；OpenAI 默认开启、GPT-5.6+ 读 0.1×/写 1.25×、门槛 1024 token（https://developers.openai.com/api/docs/guides/prompt-caching）；DeepSeek、Kimi、GLM 均有缓存价（见上）。
3. **通用语义缓存产品化退潮**：GPTCache 明示不再适配新 API/模型，且存在误命中问题（https://github.com/zilliztech/GPTCache）；主流路径转向引擎原生前缀缓存计费。
4. **智能路由产品化成熟**：OpenRouter Auto Router 按任务分类（约 30 类）+ 社区 7 天消费份额排名 + cost_tier 档位路由、不收附加费（https://openrouter.ai/docs/guides/routing/routers/auto-router），provider 路由默认按价格负载均衡（https://openrouter.ai/docs/features/provider-routing）；LiteLLM 内置 cost-based/latency-based 等策略（https://docs.litellm.ai/docs/routing）；聚合网关 new-api（46.8k stars）提供渠道加权、失败重试与含缓存命中的用量计费统计（https://github.com/QuantumNous/new-api）。
5. **多模态目录统一管理在平台/网关层已常态化**（new-api 聚合 chat/image/audio/video/embedding/rerank/realtime；百炼多厂商货架），桌面端以 Cherry「目录热更新 + 统一连通性检测」最接近，但「chat/生图/语音/embedding 一处管理」仍无桌面客户端标杆。

### 对 gaea 的机会与威胁

**机会**
- gaea 的 GLM coding=端点适配面临失真风险，也是升级契机：编码套餐已积分制化、旧模型名自动切到 GLM-5.3，静态模型目录若不热更新，模型名与用量口径都会过时（来源同上 bigmodel.cn）。
- 「成本仪表」在桌面端仍是空白位：Cherry 用量分析刚重设计、聚合网关才有缓存命中统计，gaea 做「按端点/属性的用量 + 缓存命中率 + 积分余量」即可差异化。
- 本地-云端自动路由 v1 不必做语义缓存：以「隐私/本地优先 → 缓存友好 → 峰谷错峰」为目标函数即可吃到 2026 计费红利（各家缓存价差 3–10 倍、DeepSeek 峰谷 2 倍）。

**威胁**
- Cherry Studio v2 的统一连通性检测、目录热更新、用量分析直接覆盖 gaea 模型中心核心体验，且 8 月连发 5 版迭代极快。
- OpenClaw/AutoClaw + 编码套餐把个人用户的「引擎选择」收编为订阅权益（且注明 OpenClaw 走次级调度/尽力交付），桌面助手的模型接入层被上游平台化。
- 旗舰单价高企（K3 输出 $15/M）+ 积分额度制增加用量统计与告警复杂度；coding 套餐「仅限指定工具」存在限制第三方客户端的政策风险（未核实是否长期收紧）。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

**0-3 月**
1. 成本仪表 v0：按引擎/云端属性/端点聚合 token、缓存命中率、估算成本；GLM coding 端点按「积分」口径展示并显示 5 小时/周额度余量与非高峰五折提示。
2. 静态模型目录改可热更新，建立 GLM 5.x 别名映射（5.2/5.1→5.3 自动切换的对端展示）。
3. 连接测试升级：chat ping 连发两次验证前缀缓存命中，把「缓存可用性」纳入云端属性。

**下个 3-6 月**
4. 本地-云端自动路由 v1：目标函数=本地优先 → 缓存命中最大化 → 峰谷价，采用「成本档位」设计（参考 OpenRouter cost_tier）而非复杂打分器。
5. prompt 前缀稳定性工程：固定系统提示词与记忆段前缀，吃满各家 0.1× 缓存价（注意各家最低缓存门槛 1024 token 起）。
6. 多模态目录统一 UI：chat/生图/TTS/ASR/embedding/rerank 一张引擎卡管理（对齐 new-api 类型聚合口径）。

**愿景 6-12 月**
7. 面向委托式任务的配额调度：编码套餐积分在多任务/子任务间的分配与熔断。
8. 本地语义缓存仅作实验特性评估（GPTCache 教训：误命中、停止适配新模型），默认不做。

### 参考来源

- DeepSeek API 定价（V4/缓存/峰谷/Anthropic 端点）：https://api-docs.deepseek.com/quick_start/pricing
- Z.ai GLM 按量定价与免费档：https://docs.z.ai/guides/overview/pricing
- GLM Coding Plan 套餐概览（积分/峰谷/工具支持）：https://docs.bigmodel.cn/cn/coding-plan/overview
- GLM-5.2 模型文档（1M 无损上下文）：https://docs.bigmodel.cn/cn/guide/models/text/glm-5.2
- 智谱关于页（AutoClaw 本地 OpenClaw 客户端）：https://www.zhipuai.cn/zh/about
- Kimi 定价索引（K3/K2.7 Code/K2.6、V1 下线）：https://platform.kimi.ai/docs/pricing/chat
- Kimi K3 定价（缓存命中价）：https://platform.kimi.ai/docs/pricing/chat-k3
- 阿里云百炼模型列表（多厂商全模态货架）：https://help.aliyun.com/zh/model-studio/models
- Cherry Studio Releases（v2.0.x，2026-08）：https://github.com/CherryHQ/cherry-studio/releases
- LobeHub 官网与文档（Agent harness 定位）：https://lobehub.com/zh 、https://lobehub.com/zh/docs/usage/start
- AnythingLLM（on-device 定位）：https://anythingllm.com/
- LM Studio（Bionic/云模型/LM Link，仅搜索摘要，未核实）：https://lmstudio.ai/login 、https://lm-studio.cn/
- OpenClaw 官网与文档：https://openclaw.ai 、https://docs.openclaw.ai/zh-CN
- OpenClaw 趋势参考（2026-02 指南）：https://zhuanlan.zhihu.com/p/2002485126714644013
- OpenRouter Provider Routing：https://openrouter.ai/docs/features/provider-routing
- OpenRouter Auto Router：https://openrouter.ai/docs/guides/routing/routers/auto-router
- LiteLLM Router 策略：https://docs.litellm.ai/docs/routing
- new-api 网关（聚合/计费/缓存命中统计）：https://github.com/QuantumNous/new-api
- GPTCache（维护状态与局限）：https://github.com/zilliztech/GPTCache
- Anthropic Prompt Caching 计费：https://platform.claude.com/docs/en/build-with-claude/prompt-caching
- OpenAI Prompt Caching 计费：https://developers.openai.com/api/docs/guides/prompt-caching
