# 桌面端 LLM 客户端「模型/服务商管理」调研（原始稿，2026-09-02）

调研子代理产出，来源均为官方网站/文档/GitHub（2026-09 现状）。合成结论见 `docs/market-research-2026-09-02.md`。

## 1. Cherry Studio
- 服务商配置：内置约 40 家服务商（OpenAI、Gemini、Anthropic、硅基流动等），含「密钥、API 地址、代理」三项通用配置；页面顶部有服务商启用开关；支持「添加」自定义服务商，可从 OpenAI 类型/Gemini 类型/智谱类型/云端预设创建（来源：https://docs.cherryai.com.cn/pre-basic/providers/providers 、https://docs.cherryai.com.cn/pre-basic/providers/zi-ding-yi-fu-wu-shang）
- 模型列表管理：「管理」弹窗支持添加/删除模型、按名称前缀自动分组（可改分组模板 GPT-4o/Claude/Gemini 等）、每个模型可填「模型 ID、模型名称、分组、描述」；「获取模型列表」按钮可从服务商 API 拉取（来源：同上 providers 页）
- 能力标签：模型名旁有「函数（工具）、推理、视觉、网络」标签开关，手工标记，无自动检测；描述字段可标注价格、上下文长度（来源：providers 页）
- 连通性测试：「检查」按钮检测密钥/API 地址有效性，按服务商返回不同信息（如余额查询仅官方 API 支持、模型可用性）（来源：providers 页）
- 默认模型：设置→默认模型，可为默认/话题命名/翻译/快速助手等分别指定；配合助手/话题可各自绑定模型（来源：https://docs.cherryai.com.cn/pre-basic/settings/default-models）
- 亮点：设置-关于内置「模型用量统计」页（模型列表排序、数据缓存、图表可视化）；价格信息靠描述/分组自填，无内置价目库（来源：https://docs.cherryai.com.cn/changelog/v1-6-4 ）

## 2. Chatbox
- 服务商配置：内置 OpenAI/Azure/Claude/Gemini/Ollama/GLM 等；可「添加自定义提供方」（数量不限），API 模式目前仅 OpenAI 兼容（原生 Anthropic 格式仍为社区诉求）；配置项含名称、API Key、API 域名(Host)、路径(Path)（来源：https://help.aliyun.com/zh/model-studio/chatbox 、https://github.com/chatboxai/chatbox/issues/2591 ）
- 模型列表管理：每提供方下有模型列表可手动增删（如添加 embedding 模型）；新版「支持从远程获取最新模型列表」自动拉取，并有自动纠正常见配置错误的兼容逻辑（来源：https://chatboxai.app/zh/help-center/changelog 、https://chatboxai.app/zh/guide/work-mode/configuration ）
- 能力：官方指南用文档化「能力速查表」（文本/视觉/推理/速度/价格星级）介绍 Vision/Reasoning/Tool Use，应用内无用户可编辑标签、未见自动检测（来源：https://chatboxai.app/zh/guide/concepts/models ）
- 参数：Temperature/Top P/Max Tokens 为全局设置，支持设为「未设置」以兼容不支持这些参数的推理模型（来源：同上 models 页）
- 测试/分组/价格：连通性检查按钮未见；模型下拉按提供方分组；应用内未见价格与上下文长度展示（来源：未见）
- 亮点：本地数据存储、团队协作共享 API 资源（Pro）、DALL-E-3 生图；社区曾提议添加提供方时自动扫描 /v1/models（Issue #2110，属建议而非现状）（来源：https://github.com/chatboxai/chatbox/issues/2110 ）

## 3. NextChat
- 服务商配置：以单提供方切换为主（OpenAI/Azure/Google/Anthropic/DeepSeek 等），配置项为 API Key + 自定义接口地址(BASE_URL)，可通过 BASE_URL 接任何 OpenAI 兼容服务（来源：https://github.com/ChatGPTNextWeb/NextChat/blob/main/docs/user-manual-cn.md 、https://docs.infini-ai.com/gen-studio/integrations/use-nextchat.html ）
- 模型列表管理：内置模型列表硬编码；v2.13.0 起支持 `+模型名@厂商` 添加、`-模型名@厂商` 隐藏、`名=别名` 重命名的语法（服务端 CUSTOM_MODELS 环境变量或客户端设置），自定义模型暂不支持图片输入（来源：https://github.com/ChatGPTNextWeb/NextChat/issues/5001 、#5493、https://github.com/ChatGPTNextWeb/ChatGPT-Next-Web/blob/main/README_CN.md ）
- 连通性测试：未见「检查」按钮；自动拉取服务商模型列表亦未见官方文档（仅社区讨论）（来源：未见）
- 参数粒度：全局设置含 model/temperature/top_p/max_tokens/presence_penalty/frequency_penalty；「面具(Mask)」可绑定各自的模型+参数+上下文，实现按场景预设（来源：user-manual-cn.md）
- 亮点：面具（预设对话）体系、话题自动命名/压缩上下文、自部署环境变量化配置；无价格/用量/能力标签体系（来源：user-manual-cn.md）

## 4. ChatWise
- 服务商配置：Settings → Providers，「+」添加，预设或自定义 OpenAI 兼容 / Anthropic 兼容服务，填 API Key + Endpoint；代理默认跟随系统 HTTP 代理（来源：https://docs.chatwise.app/custom-provider 、https://docs.chatwise.app/proxy ）
- 模型列表管理：若服务商 API 支持 /models 端点可一键自动拉取模型，也可手动添加（来源：https://docs.chatwise.app/custom-provider ）
- 能力：特色「辅助视觉模型」——为纯文本模型自动生成图片描述从而获得图像理解；PDF 原生处理按模型支持情况（Claude/Gemini/Mistral）区分；无用户可编辑能力标签（来源：https://docs.chatwise.app/auxiliary-visual 、https://docs.chatwise.app/documents ）
- 默认模型/绑定：按对话切换模型，未见独立的默认模型设置页与收藏/排序（来源：未见，https://docs.chatwise.app/ ）
- 亮点：自定义提供方支持 Responses API；Agent 模式内置 Bash/Write/Edit 等工具；ChatWise Search 免 Key 本地抓取 Google 结果供所有模型用（来源：https://chatwise.app/changelog 、https://docs.chatwise.app/agent 、https://docs.chatwise.app/web-search ）

## 5. BoltAI (Mac)
- 服务商配置：Settings → Models，「+」按提供方类型添加（OpenAI/Anthropic/Gemini/Mistral/Azure/Bedrock/xAI/Perplexity/Ollama/LM Studio 及「OpenAI 兼容服务器」表单）；BYOK，API Key 可用口令加密，本地 SQLite 存储（来源：https://docs.boltai.com/docs/start/use-another-ai-service 、https://boltai.com/help/how-to-setup-custom-openai-server 、https://boltai.com/ ）
- 模型列表管理：宣称 300+ 模型统一工作区；changelog 提及改进「自定义提供方的模型发现（model discovery）」即自动拉取模型列表（含带 API 版本/查询参数的端点）（来源：https://boltai.com/ 、https://boltai.com/changelog ）
- 默认模型：可设默认模型，并按聊天或按 Agent 覆盖，随时切换；多 Agent 对话可并排对比多个模型（来源：https://boltai.com/#faq ）
- 参数粒度：生成级细粒度控制——temperature、max tokens、top-p/top-k、penalties 及系统指令；另有「Context Profiles」一键切换面向编码/写作/研究的整套配置（来源：https://boltai.com/ ）
- 能力标签/价格展示：未见应用内能力标签自动检测与价格/上下文长度展示（仅营销口径 vision-enabled models）（来源：未见）
- 亮点：错误信息改进——透出上游服务商原始错误详情；修复「新项目错误选择提供方/丢失 Base Agent 模型与生成设置」类问题；原生 Mac + 全局快捷键/截图问答（来源：https://boltai.com/ 首页 changelog 区）

## 横向小结
- 自动拉取模型列表：Cherry Studio / ChatWise / Chatbox（新版）/ BoltAI 均支持；NextChat 靠语法维护列表
- 连通性测试按钮：仅 Cherry Studio 有明确「检查」（含余额/可用性）；其余未见
- 能力标签：Cherry Studio（手工四类标签）最完整；ChatWise 用「辅助视觉模型」另辟蹊径；无一家自动检测
- 价格/用量：Cherry Studio 有用量统计页+描述自填价格；其余未见系统化展示
- 参数粒度：BoltAI（生成级 top-p/top-k/penalties）与 NextChat（面具绑定）最细；Chatbox 有「未设置」兼容设计
