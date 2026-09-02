# 微信助手专项市场调研（2026-09-02，基线 v4.38.0，绑定面 557）

模块制原始稿：`docs/research-2026-09-02b/`（codebase-recon 现状侦察 + capability-recon 能力底座侦察 + market-scan 市场扫描 v2）。
**定位（用户拍板）**：微信助手 = 通过微信与 gaea 对话进行各项工作——**对话聊天、出图、改图、收发文件、多微信并行**。本文为合成版：现状盘点 → 市场发现 → 差距 → 优化方向提案（供拍板）。

## 1. 现状盘点（对照定位五项）

| 能力 | 现状 | 底座 |
|---|---|---|
| 对话聊天 | ✅ 真机已通：文本→意图路由→轻语聊天（人格+联网搜索） | `whisper_state.go:66-105` |
| 出图 | ✅ 真机已通：「画一张…」→GenerateFreeImage→图片卡回推 | `intent_router.go:184-200` |
| **改图** | ❌ 入站图片只做 OCR 识别；无改图意图；无云端 edit 引擎 | 缺最后一公里：图片字节→`GenerateMedia.InitImage` |
| **收发文件** | ❌ 入站 type=4 只出占位提示；出站仅图片白名单 | file_item 字段留位+防御解析已有；产物登记表已有 |
| **多微信并行** | ⚠️ 架构已支持（每助手独立 Server/Token/人格，无全局锁），前端无管理台 | 后端 CRUD 12 绑定齐备，WeixinPage 只有只读列表 |

其他基线：官方 iLink ClawBot 通道（合法不封号）、扫码/配对码绑定、离线代办回推、限频/SSRF/截断防线、图片双向上传下载（真机定稿）。绑定面 12 方法。

## 2. 市场发现

1. **指令改图是成熟品类**：Qwen-Image-Edit（百炼 API 现货、多图输入、改字/增删物体/风格迁移）、SeedEdit 3.0（豆包/即梦）、Nano-Banana（Gemini）、FLUX Kontext。「发图+一句话→回编辑后图」与微信对话形态天然契合。GLM CogView 无图生图（代码已实证）。
2. **文件收发是同通道标配**：openclaw-weixin 逆向文档含 file_item 完整协议（可互证抓包）；生态实证场景 = 发文件给 AI 分析 + AI 产物（Word/Excel/PDF）发回用户——即 gaea 办公产物的回推形态。
3. **多微信并行是官方标配**：openclaw-weixin 官方支持多号同时登录、上下文隔离、每号扫码一次；多 Agent 人格隔离是通用范式。gaea 架构已对齐，缺管理台+两个并发 bug。
4. 通道格局（v1 结论）：官方 iLink 路线已是行业最优解；hook/协议库封号风险高，不碰。

## 3. 差距总表

| 维度 | 市场/竞品 | gaea | 结论 |
|---|---|---|---|
| 改图 | 四强模型 API 现货 | 零接入（本地 img2img 仅 ComfyUI 2 模型） | 最大价值缺口，云端引擎+意图接线一刀 |
| 文件收发 | 同通道标配 | 占位提示/仅图片 | 协议抓包+实装一刀 |
| 多号管理台 | 官方标配+隔离范式 | 后端齐/前端零 | 纯工程刀，无协议风险 |
| 并发正确性 | 隔离范式 | 同人格共享 orchestrator 覆盖名字（无锁）；Update 不回写 WxBotID/PortraitURL | 真 bug，随管理台刀收 |
| inpaint/参考槽 | roadmap 0-3 月规划 | 零 | 后置（依赖 edit 引擎落地） |

## 4. 优化方向提案（供拍板，建议刀序 C → A → B，D 搭车）

**C. 多微信并行管理台——纯工程刀，无协议风险，建议第一刀**
WeixinPage 从只读列表升级为助手管理台：助手卡（portrait/人格/状态徽标）+ 启停开关 + 新增/删除 + **逐助手扫码绑定**（修 `confirmBinding` 硬编码 `id:'gaea'`）+ per-assistant 会话过期重绑入口。后端 `WhisperAssistantList/Save/Delete` 已绑定零新增。随刀收两个真 bug：①同人格多助手共享 orchestrator 的 `AssistantName` 无锁覆盖（每号强制独立人格或加锁修共享）；②`manager.Update` 补回写 WxBotID/PortraitURL/VoiceGuide/Dims。体量：前端为主一整刀。红线：新增绑定方法先登记 gen_bindings explicitOverrides。

**A. 对话式改图——价值最高刀，云端 edit 引擎 + 微信接线**
①引擎：image backend 注册表（kind→factory）新增云端 edit 后端，首发 Qwen-Image-Edit（百炼 API 文档公开、多图输入、改字/增删物体/风格迁移全要），模型中心可绑；本地兜底 ComfyUI img2img（已有）。②意图：`intent.go` 新增 ActionEditImage（「把这张图…/改成…/去掉…」+ 图片上下文——缓存最近入站图，支持「刚才那张图」指代）。③接线：入站图片已有解密落盘路径（`resolveDownload`）→ 转成 `GenerateMedia.InitImage` → 产物 CardPath 走现成图片卡回推。vision 识图链路不动（红线）。体量：一整刀，后端为主。

**B. 文件收发双向——协议刀，抓包是硬前置**
入站：真机抓包定稿 file_item 下载协议（与 openclaw-weixin 逆向文档互证）→ 复用 media_download 防线（SSRF/20MiB/魔数）→ 类型识别 → 接桌面侧 document_import 提取文本进对话（「总结这个文件」）。出站：产物推送——办公产物读交付物登记表（docx/xlsx/pptx/zip）+ 绘梦图，经上传协议发文件卡片；降级链完整（文件→图片卡→文本路径卡，后两级已有）。体量：一整刀；前置抓包半步可与 A/C 并行准备。

**D. 出图增强搭车件（小）**
生图已通，搭 A 刀车：一次出 N 张供选、尺寸/风格快捷参数指令（「横版 16:9 赛博朋克」）。

**不建议本期立项**：企微通道/语音条/快捷指令面板（v1 提案，非本定位重点，挂回欠账池）；inpaint/IP-Adapter（等 A 落地后二期）；群聊（官方通道未开放）。

**刀序建议**：C（稳，先收用户点名的多微信并行+真 bug）→ A（改图，API 现货无协议风险）→ B（文件，唯一带抓包前置的刀）。文件所有权集中（weixin 包 + image backend + WeixinPage），单线推进。
