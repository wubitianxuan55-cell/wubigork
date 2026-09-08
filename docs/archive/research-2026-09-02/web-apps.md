# Web 端 LLM 应用「模型中心」调研（原始稿，2026-09-02）

调研子代理产出，来源均为官方文档/GitHub（2026-09 现状）。合成结论见 `docs/market-research-2026-09-02.md`。

## 一、LobeChat / LobeHub
- **服务商管理**：内置 40+ 服务商（设置→模型服务商），填 API Key、可选代理 Base URL/自托管端点、服务商独立开关；2025-01-22 起支持添加任意 OpenAI 兼容「自定义 AI Provider」，可编辑现有配置。来源：[使用多模型服务商](https://lobehub.com/zh/docs/usage/providers)、[自定义 AI Provider 更新日志](https://lobehub.com/zh/changelog/2025-01-22-new-ai-provider)
- **多 Key**：API Key 支持逗号分隔多个，`API_KEY_SELECT_MODE=random|turn` 实现随机/轮询取 Key（自部署）。来源：[环境变量 Basic](https://lobehub.com/zh/docs/self-hosting/environment-variables/basic)
- **模型列表**：UI 内勾选启停模型、手动输入模型 ID 添加、「获取模型列表」自动拉取、每服务商「连通性检查」按钮；部署级 `OPENAI_MODEL_LIST` 等语法：`+`添加/`-`隐藏/`-all,+gpt-4o` 白名单、`模型名->部署名=展示名` 别名。来源：[模型列表自定义](https://lobehub.com/zh/docs/self-hosting/advanced/model-list)、[模型服务商环境变量](https://lobehub.com/zh/docs/self-hosting/environment-variables/model-provider)
- **能力标记**：模型可标 `<128000:vision:fc:reasoning>`（上下文长度+视觉/函数调用/推理/文件等标签）；自定义模型条目可逐项设置能力。来源：同上 + 更新日志
- **参数默认值**：`DEFAULT_AGENT_CONFIG`（default=供应商/模型，含 temperature 等）+ `SYSTEM_AGENT` 为翻译/摘要等系统助手分别指定模型；用户会话内可覆盖。来源：[环境变量 Basic](https://lobehub.com/zh/docs/self-hosting/environment-variables/basic)
- **管理员 vs 用户**：产品定位单用户/自服务，模型配置存用户侧（服务端 DB 模式按用户保存），未见管理员集中管控模型可见性/配额。
- **独有亮点**：「发现」页生态（助手/模型/插件目录）；系统助手独立选型；多 Key 轮询。故障转移/负载均衡（服务商级）：未见。

## 二、Open WebUI
- **连接管理**：管理面板 Settings→Admin→Connections：可添加多条 OpenAI 兼容连接和多条 Ollama 连接，各自 Base URL/API Key、独立启用开关；模型自动发现（/v1/models）。来源：[Connect a Provider](https://docs.openwebui.com/getting-started/quick-start/connect-a-provider/)
- **聚合与区分**：所有连接的模型聚合到一个选择器；连接可设 Prefix ID 前缀区分不同上游的同名模型（已知有 bug #28929）；连接级/环境变量（OPENAI_API_MODELS）可做模型白名单过滤。来源：[OpenAI Compatible](https://docs.openwebui.com/getting-started/quick-start/connect-a-provider/starting-with-openai-compatible/)、[Issue #28929](https://github.com/open-webui/open-webui/issues/28929)
- **用户级直连**：Direct Connections 允许普通用户自带 OpenAI 兼容端点绕过后端直连（管理员可禁用）。来源：[Direct Connections](https://docs.openwebui.com/features/chat-conversations/direct-connections/)
- **可见性控制**：Workspace/Model Builder 创建的模型有 Public/Private/Group 三级可见性（默认 Private，按组授权）；配合角色/组权限体系。来源：[Models](https://docs.openwebui.com/features/workspace/models/)
- **参数默认值**：Admin→Models→Model Defaults 设实例级默认参数；每个 workspace 模型可再覆盖（系统提示词、高级参数、温度等）；聊天中亦可临时覆盖。来源：[API Endpoints Reference](https://docs.openwebui.com/reference/api-endpoints/)
- **健康/负载均衡**：未见原生健康检查与故障转移；同一模型 ID 出现在多连接时仅聚合展示，真正负载均衡需前置 LiteLLM/Olla/Portkey 等网关。来源：[Discussion #22345](https://github.com/open-webui/open-webui/discussions/22345)
- **能力标记**：模型级 capabilities 开关（vision 等）；上下文长度/价格信息：未见集中展示。
- **独有亮点**：Model Builder 可基于任意基础模型创建"预设模型"并绑定知识库、工具、Filter Function（可改写模型行为）；模型置顶/标签。

## 三、AnythingLLM
- **服务商管理**：系统级单选式——Admin→LLM Preference 从 30+ 集成提供商中选一个；Generic OpenAI（Base Path + API Key + Chat Model Name）接入任意 OpenAI 兼容端点。来源：[Language Models](https://docs.anythingllm.com/features/language-models)、[OpenAI Generic](https://docs.anythingllm.com/setup/llm-configuration/cloud/openai-generic)
- **模型列表**：按所选 provider 自动拉取其模型列表，部分需手填模型名；白/黑名单、别名、排序：未见。
- **管理员 vs 用户**：管理员设系统默认，多用户模式下每个工作区可在 Chat Settings 覆盖 LLM 提供商与模型（工作区按用户授权），普通用户无法改系统级设置。来源：[LLM Configuration Overview](https://docs.useanything.com/setup/llm-configuration/overview)
- **Model Router（独有）**：规则式按消息动态路由——按关键词/条件把每条消息发给不同 provider+model，而非锁定单一模型。来源：[What is the Model Router?](https://docs.anythingllm.com/model-router/overview)、[Setup](https://docs.anythingllm.com/model-router/setup)
- **参数默认值**：LLM Preference 内设 temperature/max tokens/context length（视 provider 而定）；工作区级再覆盖（温度、历史长度等）。来源：[Configuration](https://docs.anythingllm.com/configuration)
- **能力标记/价格**：未见集中能力标签与价格信息（Generic OpenAI 仅可填 context window）。
- **独有亮点**：Desktop 版可「Import custom model」导入本地 GGUF 模型文件（[Import an LLM](https://docs.anythingllm.com/import-custom-models)）；聊天 LLM、嵌入模型、Agent LLM、TTS/STT 四类模型分列独立选择。健康检查/故障转移/负载均衡/用量配额：未见。

## 四、LibreChat
- **连接管理**：声明式 librechat.yaml——endpoints.custom[] 定义任意多个端点：name/apiKey/baseURL/models/headers/iconURL/titleModel 等；内置端点（OpenAI/Anthropic/Google/Bedrock 等）经 env 配置。来源：[AI Endpoints](https://www.librechat.ai/docs/configuration/librechat_yaml/ai_endpoints)、[Custom Endpoint 结构](https://www.librechat.ai/docs/configuration/librechat_yaml/object_structure/custom_endpoint)
- **Key 管理（独有）**：apiKey 支持 env 引用、经管理 API 灌入后加密存储（apiKeyPreview 预览）、`user_provided` 用户自带 Key（BYOK，baseURL 亦可 user_provided）；请求头可注入 `{{LIBRECHAT_USER_ID/EMAIL/NAME}}`。来源：同上
- **模型列表**：models.list 手工声明（顺序即下拉顺序）或 models.fetch=true 自动拉取（可配 fetchEndpoint、userIdQuery 按用户过滤）、defaultModel 指默认；无独立黑白名单（用 list 实现）。来源：同上
- **管理员 vs 用户**：yaml 全局生效即管理员级控制；modelSpecs 的 showInModelList 控制是否只呈现预设卡；未见端点级用户组 ACL（Agents 资源有 ACL）。
- **参数体系（最细）**：defaultParamsEndpoint+defaultParams 设全局默认；每端点 addParams/dropParams/customParams.paramDefinitions（含 default 默认值与枚举范围）；modelSpecs.preset 锁定参数+maxContextTokens；titleModel/titleConvo 单独指定标题模型。来源：[Custom Params](https://www.librechat.ai/docs/configuration/librechat_yaml/object_structure/custom_params)、[Model Specs](https://www.librechat.ai/docs/configuration/librechat_yaml/object_structure/model_specs)
- **能力/价格**：tokenConfig 可按模型声明 prompt/completion/context/cache 价格（对接余额扣费）；maxContextTokens 声明上下文；能力标签（vision/tools）：未见集中标记。
- **独有亮点**：rateLimits 全局+按端点、按 IP/用户限流（[librechat.yaml](https://www.librechat.ai/docs/configuration/librechat_yaml)）；Billing/Balance token 余额体系；modelSpecs 预设卡（图标/描述/开场白 starters）。健康检查/故障转移/负载均衡：未见（可经 reverse proxy 外置）。

## 横向对比速览
| 维度 | LobeChat | Open WebUI | AnythingLLM | LibreChat |
|---|---|---|---|---|
| 多服务商 | 40+ 内置+自定义 | 多连接聚合 | 单选+Generic | yaml 声明任意多端点 |
| 多 Key/轮询 | 有（random/turn） | 未见 | 未见 | user_provided/加密存储 |
| 模型白名单 | MODEL_LIST 语法 | 连接级/ENV | 未见 | models.list |
| 可见性控制 | 未见 | Public/Private/Group | 系统级 vs 工作区 | 全局 yaml+modelSpecs |
| 连通检查 | 有 | 未见 | 未见 | 未见 |
| 负载均衡/故障转移 | 仅 Key 轮询 | 未见（需外置网关） | 未见 | 未见 |
| 能力/价格标记 | 能力标签+上下文 | capabilities 开关 | 未见 | tokenConfig 价格 |
| 独有亮点 | 发现页生态、系统助手分模型 | Model Builder 绑定知识/工具/Filter | Model Router 规则路由、GGUF 导入 | BYOK+限流+余额计费 |
