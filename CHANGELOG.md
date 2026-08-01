# gaea · 多功能 AI 助手

## v1.6.2「方案模型条修复」(2026-08-02)

> 补丁：方案编写板块模型条不可见修复（OfficePage 主 return 为单行 Fragment，此前 JSX 未插入）。
> tag v1.6.2，构建 36,109,824 字节。

- OfficePage 顶部插入 FeatureModelBar（feature="office"），方案编写窗口显示绑定模型 + 启停
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.1「小说统一模型」(2026-08-02)

> 补丁：小说板块章节/角色/世界观 agent 统一接入 func_novel，整部小说用一个 LLM 模型（v1.6.0 已知限制 1 修复）。
> tag v1.6.1，构建 36,108,288 字节。

- worldview / character / chapter / analysis / outline 5 个 agent 全部加 featureModel + chat（带 func_novel 引擎覆盖），
  替换全部 `ChatSimpleStream(a.cfg.Model)` 调用 → `chat(ctx, ...)` + `ChatSimpleStreamWithOptions(EngineID: FuncNovelEngine)`
- 运行中切换小说模型即时生效（各 agent 动态读 cfg.FuncNovelEngine/Model）
- 验证：go test ./... 53 包全绿 + tsc -b + vite + wails build

## v1.6.0「语音交互 · 角色中心 · 功能模型」(2026-08-02)

> 语言交互全面落地：首页语音对话 + 核心 AI 助手 gaea（大地女神）+ 角色中心（助手即角色）+ 各功能板块独立模型绑定与资源监控。
> 23 提交（v1.5.0 后），tag v1.6.0，构建 36,108,288 字节。详见 releases/v1.6.0.md

### 首页语言交互（全新）
- 深空虚空首页：400px 大语言粒子球（连线网络+双环绕轨道+音量驱动）悬浮，8 透明玻璃卡片浮游，虚空微尘背景
- 「进入语音对话」本页直启麦克风：轻语 voiceManager 管道 → gaea 对话 → 识别/回复气泡 + TTS 朗读
- 布局响应式（粒子球随视口缩放）+ 语音卡片边框透明

### 核心 AI 助手 gaea = 大地女神盖亚
- 新增人格预设 gaea（首位全局默认）：大地之母，温厚沉稳包容滋养（五维 T85/I55/S20/O80/R50，goddess 标签）
- 启动确保 gaea 始终存在（修复旧数据缺失）；唯一「AI 助手」，其余标「角色」

### 角色中心（助手即角色）
- 管理中心改铺满主界面角色中心（角色卡网格 + 详情弹窗）；AI 角色剧照；小说↔轻语互传（参数统一）
- gaea 排第一 > 当前对话 > 启用 > 禁用

### 功能级模型绑定
- ai.Client per-call 引擎覆盖；config 10 键（chat/whisper/novel/office/gaea 引擎+模型）持久化
- 接入：聊天/轻语（含语音）/小说大纲；模型中心「功能模型绑定」UI
- 各窗口 FeatureModelBar（三态状态+一键启停）；底栏资源监控（CPU/内存/GPU+已启用模型）；超载弹窗警告

### 模型中心完善
- 恢复 ComfyUI 引擎（启停/状态/本机路径写死）；ComfyUI 图片模型（krea2/z-image-turbo）入墙
- 语音模型 STT/TTS 三段选择 + 持久化

### 已知限制（范围控制）
- 小说 agent 绑定仅大纲；办公 agent 绑定仅 UI；语音端到端依赖本地 herdsman 未 GUI 自动化实测

## v1.5.0「微信接通」(2026-08-02)

> 微信 ClawBot 通道全链路打通（微信发消息 → AI 自动回复），办公板块 v1.4.0 回归修复。
> 11 提交（v1.4.0 后），tag v1.5.0，构建 36,011,520 字节。
> 详见 releases/v1.5.0.md

### 微信 ClawBot 通道接通（核心）
- 认证修复：`Authorization: Bearer <token>` + `iLink-App-Id` + `iLink-App-ClientVersion`（数字编码 132099）
  + getUpdates 端点全小写 → 消除"会话过期"（errcode -14）
- 消息字段对齐**腾讯官方 openclaw-weixin**：`item_list[].type`（type=1 文本）+ client_id `{prefix}:{ts}-{hex}`
  （此前对齐社区 Rust SDK 误用 `item_type`，消息被静默丢弃 → 微信无回复的最终根因）
- 扫码绑定流程补全：need_verifycode 配对码二次查询（新绑定 WhisperWeixinQRStatusWithCode）、
  scaned_but_redirect/verify_code_blocked 处理；confirmed 返回完整 token + botId
- 会话过期状态透出：UI 显示"微信会话过期 · 需重新绑定"替代虚假"在线"
- 防脱敏 Token：前端含 `*` 用原值、后端含 `*` 拒绝
- 助手名字注入：BuildAckemCanonBlock 名字参数化 + Orchestrator.AssistantName，微信 AI 自称助手名
  （如"峨嵋"）而非默认"轻语"

### 办公板块修复（v1.4.0 回归）
- 右侧面板 Drawer 内联渲染（getContainer=false）+ 改为布局内 grid 列（340px）——不再跨页面残留/遮挡导航栏/凸出遮挡对话区
- 恢复对话区样式（修复 eb7d5e6 误删 chat-pane/transcript/markdown 样式族，输出框消失）
- 移除办公设置 + 明暗配色按钮（与系统重复，由主应用设置中心接管）

### 验证
- go test ./... 53 包全绿；tsc + vite build；wails build 产物 36,011,520 字节
- 微信全链路用户实测通过（收到消息 → AI 回复 → 发送成功）

## v1.4.0「架构统一」(2026-08-02)

> 后端四类重复收敛 + 前端两套 UI 体系统一。净删 6874 行（+555/-7429），10 提交，tag v1.4.0。
> 详见 releases/v1.4.0.md

### 后端架构重构（-526 行）
- HTTP client 统一：netclient 提升共享层，22 处裸 `http.Client` 收敛到 `NewSimpleClient`
- 工具函数收敛：office 死代码 + asr/whisper 手写字符串函数替换标准库
- 配置系统整合：删除 gaea config 零消费者死域（ComfyUI/Statusline/Notify/Serve/LSP）
- AI 通道唯一化：删除 provider/stream_client.go，通道=前端→agent→bridge→ai.Client→modelengine

### 前端两套 UI 体系统一（-6300+ 行）
- 死代码清理：老栈 29 废弃组件 + gaea 7 死代码 + 4 死 hook/util + highlight.js（-5800 行）
- 令牌层统一：gaea 颜色引用老栈 M3 令牌，删独立主题系统，暗亮双向联动
- 图标体系统一：66 个 lucide → @ant-design/icons，lucide 依赖移除
- 布局壳 antd 化：Layout/Drawer/Tooltip 用 antd，功能面板按分层原则保留自绘
- 死样式清理：.app/workspace-panel 样式族删除（-690 行）

### 验证
- go test 77 包全绿；tsc + vite build 通过；wails build 产物 gaea.exe

## v1.3.0「未来感 UI」(2026-08-02)

> 未来感 AI 多功能助手平台：深空星云 × 玻璃拟态 × 霓虹光效全链路统一 + 设置中心整合全部参数。
> 五批 UI 改造（+718/-268 前端），tag v1.3.0，构建 36.1MB。

### 设计系统
- 12 套主题新增 glow/glassBg/auroraBg 三令牌（App.tsx 注入 CSS 变量）
- 未来感 CSS 层：赛博网格 + 星点 + 漂浮光球背景、玻璃拟态、霓虹描边卡片、发光状态点、流光扫描线

### 框架与界面
- 顶栏玻璃化 + logo 光晕 + 菜单霓虹选中态；AI 中枢首页（真实模型状态 + 霓虹卡片墙）
- 模态框/弹层全局玻璃化；主题色块发光胶囊；页面切换过渡 + 霓虹加载态
- 聊天界面：玻璃气泡 + 渐变用户气泡 + 霓虹输入栏；侧栏玻璃化
- 绘梦面板玻璃化 + 画廊 hover 发光；底栏霓虹状态条

### 设置中心（整合全部设置参数）
- SettingsPage 重写为 7 分组 Tabs（外观/工作空间/模型引擎/语音/绘梦/办公/系统）
- 新增 settings/ 8 组件：主题发光选择器、图片保存目录、推理强度、引擎编辑、语音健康检测与阈值、绘梦后端、办公摘要、v4 迁移
- api/settings.ts 补 5 封装；设置页隐藏 AI 控制台

### 验证
- go test ./... 全绿（54 包）；go build / go vet 通过
- npm run build + wails build 产物 gaea.exe（36.1MB）
## v1.2.0「架构瘦身」(2026-08-01)

> 后端臃肿治理：绑定层按域拆分（197 方法迁移）+ 死代码清理 + 办公板块设置写路径接通
> + staticcheck U1000 全仓库清零 + 原子写实现去重。5 批提交，净删 ~600 行。

### 绑定层拆分（App 聚合 + 嵌入提升）
- App 结构体从 20+ 平铺字段收敛为 core + 4 域 State 嵌入引用（writing/media/whisper/office）
- 197 个方法按域归位；Go 嵌入提升保证 Wails 绑定不变，前端零改动
- 子服务持 app 反向指针协调跨域调用

### 办公板块设置写路径接通（20 个存根）
- 新增配置持久化（~/.config/gaea/config.toml 原子写）+ 改配置 → 保存 → 重建 controller
- Agent 参数 / 权限 / 沙箱 / 模型设置真实生效；Provider 增删改转发模型中心
- 回退 / 更新类改为明确错误语义，前端隐藏回退菜单

### 死代码清理（staticcheck U1000 全仓库清零）
- gaea 引擎：notify 包（2 文件）+ filteredSchemas + incompleteCanonicalTodos + runTurn
- whisper：candidate_collector.go 整文件（179 行孤岛）+ emotion_fusion 7 函数 + 3 常量
- ai：buildFluxWorkflow（已被 Z-Image/Krea 取代）

### 原子写去重
- 重建 fileutil.AtomicWrite，5 处手写重复实现收敛，净删 41 行

### 验证
- go build / go vet / 53 包测试全绿；staticcheck U1000 清零；前端 tsc 通过
- 新增测试：配置持久化往返 + 嵌入提升编译期断言

## v1.1.0「质量工程」(2026-08-01)
## v1.1.0「质量工程」(2026-08-01)

> 稳定性加固 + 架构瘦身 + 测试防线建立：21 处 goroutine recover、SSE 阻塞修复、
> 死代码清理 902 行、四个核心模块测试覆盖大幅提升（modelengine 97.1% / office 35.6% / ai 24% / whisper 16.8%）。

### 稳定性（3 批提交）
- 21 处 goroutine 无 recover 防护：桌面应用 panic 崩溃问题根治，最严重为
  controller.runGuarded turn 执行 panic 导致前端永久"生成中"，修复后复位状态 + Emit TurnDone
- SSE 流式发送 select+ctx.Done 保护：取消后 goroutine + HTTP 连接阻塞泄漏根治（hang 类问题根因）

### 架构整理（5 批提交，净删 902 行）
- gaea 模块：管理命令组 / 模糊编辑移植 / 宪法文件 / 7 处未使用字段别名
- 其他模块：剧照旧实现 / 世界观上下文 / 章节迁移等
- staticcheck U1000 84→25 处（剩余为 whisper 对齐 ackem + ComfyUI 迭代相关）

### 测试防线（5 批提交，+1900 行测试）
- modelengine 0%→97.1%、office/proposal 0%→35.6%、ai 7%→24%、whisper 13.1%→16.8%
- 修复 3 处测试暴露缺陷：GenerateSection 漏设 completed、UpdateSection/RemoveRawFile 静默失败

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`，完整说明见 `releases/v1.1.0.md`

## v1.0.0「品牌重塑 · 盖亚」(2026-08-01)

> wubigrok 正式更名为 **gaea**（盖亚，大地女神）——从「小说创作 Agent」升级为「多功能 AI 助手」。
> 全新品牌视觉：翡翠球体 + 破土嫩芽 + 灵感星芒，favicon / appicon 全量替换。

### 品牌重塑
- 全产品品牌替换：窗口标题「gaea · 多功能 AI 助手」、应用名/产物（gaea.exe）、UI 显示、文档、导出署名、图片下载前缀、日志文件
- Go module 重命名 `github.com/wubigork/wubigork` → `github.com/gaea/gaea`（233 个文件 import 同步）
- 新 logo 三件套：`frontend/public/favicon.svg` + `build/appicon.svg` + `build/appicon.png`（1024x1024）
- 版本号从 v5.x 重新起算为 **V1.0.0**（versioninfo.rc / wails.json / CHANGELOG）

### 数据兼容（老用户零丢失）
- 配置文件 `~/.gaea_config.json`（回退读取 `~/.wubigork_config.json`）
- 登录 token 回退读取 `.wubigork_token.json`（免重新登录）
- 项目标记目录 `.gaea/`（识别旧 `.wubigork/` 项目，v4 检测双向兼容）
- localStorage 键 `gaea_*`（回退读取旧键：聊天记录/人格/主题/绘梦模板保留）
- 内部 provider 注册名 `wubigrok` 保留（bridge provider 引擎兼容），UI 显示名全部为 gaea

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`

## v5.76.0「工程办公」(2026-07-31)

> gaeaW（土壤修复工程办公 AI 助手）完整移植：47 个工程工具 + Hermes/Hephaestus 双模型 agent + 6 个工程技能 + gaeaW 原生 UI，模型统一走 gaea 模型中心。

### 新板块：办公
- 后端移植 `internal/gaea/` 30 包（agent/tool/control/skill/command/plugin/knowledge/memory/boot/config），模型经 `provider/bridge` 接入模型中心（空模型动态跟随引擎切换）
- ai 包扩展工具调用支持（OpenAI 兼容 + SSE tool_calls 分片拼装，向后兼容）
- gaeaW 原生 UI 完整移植至 `frontend/src/gaea/`（App + 70 组件 + Tailwind v4），GaeaPage 渲染 gaeaW App
- 适配层：bridge.ts 90+ 方法名映射（Submit→GaeaSend 等），wubigrok 补齐 80+ Gaea* 绑定方法，事件格式精确对齐 WireEvent
- `.gaea/skills/` 6 个工程技能（场地调查/风险评估/修复设计/投标/数据报告）

### 发布
- 完整说明见 `releases/v5.76.0.md`

## v5.73.1「方案编写完善」(2026-07-31)

### 关键修复
- 嵌套章节（AI 大纲的 2/3 级）后端查找/更新全部改为递归展平，子章节可正常撰写/润色/改图/重命名
- 流式撰写内容真正落盘（原为副本指针，刷新即丢）
- 前端自动保存与图表插入支持子章节（updTree 递归更新）
- Word/MD 导出、覆盖/规范检查包含全部层级章节
- 浏览器上传招标文件改为 base64 落盘后转换（原路径为空必然失败）

### 新能力
- 大纲手工编辑：新增子章节/重命名/删除（含确认与编号重排），三个章节树均可用
- 流式撰写上下文补齐：需求、评分标准、废标条款、完整大纲、前一章节
- 失败时向前端发送 error 事件，不再卡死「生成中」

### 发布
- 完整说明见 `releases/v5.73.1.md`

## v5.20.0「精炼」(2025-07-26)

> 提示词全面重设计 + 死代码大清理 + 编译修复。净删 ~3000 行，prompts 23→15。

### 提示词重设计（4 个核心 prompt）
- **create-chapter**：硬编码字符串 → 模板化，「正在写这本书的作者」
- **chapter-generate**：「出版级」→「作者」，去 AI 味交由技能注入
- **plot-branch-browser**：「剧情策划人」→「story breaker」，固定 3 分支
- **worldview-agent**：「设定顾问」→「设定编辑」，代码块输出替换原文

### 死代码大清理（16 文件删除 + 后端精简）
- **删除 9 个废弃 prompt JSON**：brainstorm-ideas, chapter-review, outline-generate-detail, story-thread-chat, story-thread-generate, worldview-chat-section, worldview-check-consistency, worldview-generate-all, bootstrap-reference-summarize
- **删除 6 个前端组件**：AIAssistSheet, BrainstormModal, DialogueModal, StoryBibleModal, StoryBibleSteps, BeatToProse
- **删除 brainstorm_handler.go**（59 行）
- **Go handler 死代码清理**：chapter_handler (-178), copilot_handler (-237), create_chapter_handler (-82), outline_handler (-108), plot_branch_handler (-80), project_handler (-107), worldview_handler (-42)
- **核心模块精简**：chapter.go (-301), outline.go (-203), worldview.go (-215)
- **前端页面清理**：MainLayout (-60), CreatePage (-46)，其他页面移除死引用

### 编译修复
- chapter_handler.go 缺失函数闭合 `}` 修复
- copilot_handler.go 6 个未使用 import 清理
- chapter_handler.go 未使用 "strings" import 清理

## v5.17.0「重塑」(2025-07-26)

> 从 v5.7.1 分支重建。移除移动端代码，新增「创作」面板，章节节点树 + 分支系统。

### 移除
- 删除 `mobile/` 独立移动端应用
- 删除 `internal/mobile/` Go HTTP 服务
- 删除 MobileTabBar、MobileDrawer、useIsMobile 等全部移动端组件

### 创作面板（全新）
- **三栏布局**：左侧节点树 | 中间编辑器 | 右侧 AI 控制台（全局复用）
- **CreateChapter API**：一步到位，跳过对话和大纲阶段，直接生成正文
- **三分支构思**：AI 读取设定 + 前文摘要 → 生成 3 个剧情方向 → 用户选择 → 生成正文
- **节点树系统**：每章永久节点（ID、摘要、parent_id），支持分支/覆盖/删除/重生成
- **摘要注入**：每章生成后自动提取摘要存入节点树，前文注入改为节点摘要拼接

### AI 控制台增强
- 展开 REQ 查看完整 SYSTEM/USER prompt
- 展开 OK 查看 AI 完整响应（修复 response 事件缺失 content 字段）
- 固定宽度 380px + alignSelf: stretch

### 后端
- `internal/app/create_chapter_handler.go` — CreateChapter 一步生成+保存
- 正文末尾 `---CHAPTER_SUMMARY---` 标记自动分离摘要
- `GenerateOutlineWithDialogue` prompt 限制章节数，避免浪费 tokens

## v5.7.0「凝练」(2026-07-06)

> 7 轮组件化拆分迭代：42 文件变更，+3,244 / -1,946，(window as any) 和 @ts-ignore 全面清零

### ⚛️ 组件化重构（7 轮）

#### R3 — 去重 & 组件化
- **BranchSelectorPanel**: 提取 PlotBranchModal + NextChapterModal 共享分支选择 UI（187 行）
- **StoryBibleSteps**: StoryBibleModal 5 步子组件拆分，主组件 489→288 行
- **净效果**: +701/-448，两 Modal 各减少 ~90 行重复代码

#### R4 — 书架模块
- **提取 4 子组件**: WelcomePage / BrainstormModal / CreateNovelModal / ProjectCardItem
- **HomePage**: 640→280 行 (-56%)，Ctrl+N 快捷键 + 骨架屏
- **Utility**: 提取 formatRelativeTime + delay 到 utils/time.ts
- **类型化**: 新增 BrainstormIdea 接口，消除 any
- **移动端**: 统一 btnBase 样式，移除无效 CSS animation

#### R5 — 世界观模块
- **API 抽象层**: api/worldview.ts，消除 6 处 @ts-ignore
- **SectionNav**: 共享维度导航（桌面固定侧栏 / 移动端下拉面板），消除 ~90 行重复
- **ConsistencyReport / MapFullscreen**: 提取行内子组件
- **WorldviewPage**: 490→328 行 (-33%)

#### R6 — 角色模块
- **API 抽象层**: api/character.ts，消除 11 处 @ts-ignore
- **RelationshipModal / OrganizationEditModal / PortraitLightbox**: 提取行内 Modal
- **OrgField**: 移入 CharacterFormHelpers.tsx
- **renderCharEditor()**: 消除桌面/移动端 ~40 行重复
- **CharacterPage**: 489→396 行 (-19%)

#### R7 — 写作模块
- **Wails 类型导入**: 消除 7 处 (window as any)
- **ReviewResult**: 审稿组件（评分圆环 + 优势/不足/改进建议）
- **OutlinePanel**: 大纲面板独立组件（折叠/展开 + 卷/章树形渲染）
- **超长行拆分**: 最长行 620→314 字符，提取 createTabData / handleFinalize / handleNextChapterGenerate

#### R8 — 绘梦模块
- **API 抽象层**: api/image.ts，消除 12 处 (window as any)
- **ResultGallery 增强**: 添加 onDelete 支持
- **PromptBar / CustomTemplateModal**: 提取底部输入栏和模板编辑弹窗
- **ImageGenPage**: 665→521 行 (-22%)

#### R9 — 设置面板
- **API 抽象层**: api/settings.ts，消除 ~15 处 (window as any) + @ts-ignore
- **SettingsCard**: 统一卡片组件，消除 7 处重复 cardStyle
- **SettingField**: 统一「标签 + Input/Select」模式
- **handleToggleMobile / handleMigrate**: 提取命名函数
- **SettingsPage**: 475→356 行 (-25%)

### 🧹 全局清理
- **(window as any) 清零**: 前端所有页面全部替换为 wailsjs/go/app/App 类型导入
- **@ts-ignore 清零**: 通过 API 抽象层消除所有类型跳过
- **catch(_) 清零**: 全部替换为 catch(err) + console.error
- **mountedRef 移除**: React 18 自动批处理已解决

### 📦 构建
```
wails build -o build/bin/wubigork-v5.7.0.exe
```
- 新增 28 文件，修改 14 文件

## v5.6.0「移动端远程操控」(2026-07-03)

> 移动端完整功能实现：RPC 桥接、SSE 流式、静态文件服务、CORS、QR 码、IP 检测

### 📱 移动端功能
- **HTTP RPC 调度器** (`internal/mobile/rpc.go`) — 泛型反射调用 App 方法，黑名单过滤生命周期方法
- **SSE 流式推送** (`internal/mobile/sse.go`) — StreamHub 事件广播 + EventSource 频道
- **前端桥接层** (`frontend/src/api/bridge.ts`) — 非 Wails 环境自动创建 `window.go.app.App` HTTP 代理
- **Runtime 多填充** (`frontend/src/api/runtimePolyfill.ts`) — `EventsOn/Off/Emit` + SSE EventSource 多填充
- **静态文件服务** — embed.FS 生产模式 + `dist/` 前缀兼容 + SPA fallback
- **CORS 中间件** — 手机浏览器跨域访问支持
- **IP 检测** — 虚拟网卡过滤（Docker/WSL/Hyper-V 等），优先 WiFi 真实 IP
- **端口预检** — `net.Listen` 预检端口可用性，占用时立即报错

### 🐛 修复
- **移动端页面白屏** — `serveStaticOrSPA` 路径前导斜杠导致 embed.FS 拒绝读取
- **事件名不一致** — 前端 `prose-stream` → `beat-prose-stream`
- **render 崩溃** — HomePage `card.title` null 守卫
- **IP 错误** — UDP 拨号法回退到 Docker IP，接口名过滤法替代

### 📦 构建
```
wails build -o build/bin/wubigork-v5.6.0.exe
```
- 新增 4 文件，修改 12 文件

## v5.5.0「精炼」— 全量代码优化 + 工程质量加固 (2026-07-03)

> 两轮迭代：后端去重 + 前端巨型组件拆分集成 + 死代码清理 + 类型安全加固

### 🏗 后端重构（12项）
- **EstimateTokens 去重**: context+memory→util，修复中文检测用 unicode.Is
- **ExtractJSON 去重**: style→util
- **401 重试合并**: ai/client.go 三合一 → refreshAndRetry()
- **config.Save 重构**: 18-case switch→map 注册模式
- **marker 解析抽象**: ParseMarkedSections()，worldview/character 共用
- **for i:=1;;i++→ForEachChapter**: 8处死循环统一，缺失章节 continue 而非 break
- **chapter.Generate 拆分**: 200行→3函数(buildGenerateContext/streamAndRetry/postProcess)
- **ChatStream 拆分**: 请求构建+SSE 解析分离
- **mobile 死代码清理**: 删除 SPAHandler，修复 _ = path
- **app.go 类型安全**: comfyUICmd 删除，distFS interface{}→fs.FS
- **skill YAML 加固**: Windows \r\n 兼容，修正 scan() 注释
- **slog.Warn 统一**: 35+处→Agent.warnRead()

### ⚛️ 前端重构（11项）
- **ChapterPage**: 1170→227 行 (-80%)，集成 ChapterEditor/AIAssistSheet/useChapterStream
- **OutlinePage**: 1188→350 行 (-70%)，集成 OutlinePanel/ThreadPanel/DialogueModal/useOutlines
- **StoryBibleModal**: 488行→5步子组件+路由壳
- **PlotBranchModal+NextChapterModal**: 共享 usePlotBranch hook
- **4 新 hooks**: useChapterStream/useOutlines/useWailsEvent/usePlotBranch
- **errorHandler**: handleError+wrapAsync 统一错误处理
- **OutlineNode 类型冲突修复**: api/outlines.ts 删除重复定义
- **StepCreate.tsx**: onChange 死代码修复
- **HomePage Creating**: mount guard 防 unmount 后 setState
- **LorebookModal/SkillModal**: AbortController cleanup
- **TTSPlayer**: onStatusChange ref 冻结

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.5.0.exe -ldflags="-s -w" .
wails build -o build/bin/wubigork-v5.5.0.exe
```
- 新增 16 文件，修改 ~25 文件，删除 ~900 行
- 零新依赖

## v5.4.0「锻造」— 稳定性修复 + 本地生图增强 + 功能打磨 (2026-07-03)

> 修复 13 项 Bug，新增 ComfyUI 一键启停、系统状态监控、角色 Agent、图片管理增强

### 🐛 Bug 修复（13项）
- **xAI 默认模型名**: `"flux"` → `"grok-imagine-image-quality"`，修复云端生图失败
- **xAI API size 参数**: API 不再接受 `size`，请求中清除该字段
- **ComfyUI 编码崩溃**: Python 3.13 + GBK 控制台无法编码 emoji → 补丁 logger.py + prestartup_script.py
- **Login() 后 ComfyUI 丢失**: 重新登录后自动恢复后端配置
- **小说切换数据不同步**: 5 页面 (世界观/角色/大纲/章节/画布) 监听 projectPath 重新加载
- **Token 刷新无超时**: `RefreshAccessToken` 用 `http.DefaultClient` → 15s 超时，防止永久卡死
- **生成失败消息**: 返回具体原因而非笼统的「所有生成尝试均失败」
- **ComfyUI 执行错误**: 错误解析索引 msgArr[2]→msgArr[1]，正确提取异常信息
- **右侧面板隐藏**: 绘梦右侧栏改为始终显示（文件夹按钮在无历史时也可见）
- **CMD 弹窗**: `getCPUUsage` 调用 wmic 时隐藏窗口
- **防重复提交**: 绘梦生成按钮加 useRef 锁

### ✨ 新功能
- **ComfyUI 一键启停**: 绘梦顶部 🟢/⚫ 状态 + 启动/停止按钮，设置页配置安装路径
- **系统状态监控**: 绘梦右侧栏显示 CPU% + GPU 显存占用，3s 刷新
- **角色 Agent**: 角色页底部 ChatPanel，AI 对话自动创建/编辑角色
- **图片单张删除**: 画廊和历史缩略图支持逐张删除
- **图片文件夹快捷打开**: 右侧栏 📁小说图片 / 🖼生成图片 目录
- **世界观地图双击全屏**: AI 地图和 3D 势力图支持双击放大
- **图片自动保存**: 未配置专用目录时自动存到 `<小说>/images/`

### 🎨 UI/UX
- 导航 `项目` → `书架`，其余文案 `项目` → `小说`
- 大纲页卷结构面板拉长填满窗口
- 清理冗余文件回收 ~79MB

### 📦 构建
- `build/bin/wubigork-v5.4.0.exe` — 16MB

---
## v5.3.0「暗夜」— 全局配色重设计 (2026-07-03)

> 暗夜系列 6 套主题 — 手工调色，表面色与强调色分离，完全替换 M3 Tonal Palette

### 🎨 主题系统重做
- **6 套暗夜主题**: 暗夜青（默认）/ 暗夜紫 / 暗夜玫 / 暗夜金 / 暗夜苔 / 暗夜墨
- **表面色与强调色分离**: 不再从单一 seed 派生所有颜色，每套有独立的表面色温
- **手工调色**: 替换 M3 `generateTonalPalette` 自动生成，消除同色系单调问题
- **亮色模式**: 6 套暗夜主题各自对应一套亮色变体
- **零破坏**: 旧 localStorage 中的主题名自动回退到默认暗夜青

### 🔧 累积修复 (v5.2.1 → v5.3.0)
- Z-Image sampler 节点引用 14→13
- 大纲 prompt 矛盾约束（恰好5卷 vs 保留不修改）
- Lightbox z-index 1000→1050（剧照全屏被遮挡）
- AI 绘梦模板系统：类别下拉 + 自定义 + 去重风格

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.3.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **3 文件变更**: +70/-188 行
- **零新依赖**

---
## v5.2.0「凝形」— Bug修复+工程质量+性能优化 (2026-07-03)

> 基于 v5.1.0 审计，修复 24 项问题：10 Bug修复 + 8 工程质量 + 6 性能与体验

### 🐛 Bug 修复（10项）
- **Z-Image 尺寸参数**: width/height 正确映射到 Z-Image ratio（1:1/3:2/4:3/16:9/2:1），不再被忽略
- **Context cancel 泄漏**: `copilot.go` `_ = cancel` → `defer cancel()`，防止 goroutine 泄漏
- **io.ReadAll 错误忽略**: ComfyUI `queuePrompt`/`checkHistory` 两处错误现在正确返回
- **os.MkdirAll 错误忽略**: `export.go` 目录创建失败不再静默继续
- **NovelsDir 硬编码**: `D:\AI\xiaoshuo` → `~/wubigork-novels`，跨系统兼容
- **CancelGhost 空实现**: 真正取消进行中的 Ghost 补全 goroutine，前端收到 `cancelled` 事件
- **事件监听器清理**: `EventsOn('xai-output')` 添加 useEffect cleanup 防止内存泄漏
- **静默吞异常**: 全站 `catch (_) {}` → 带标签的 `console.error`
- **loadOutlines 重复**: ChapterPage/OutlinePage 重复实现 → 提取到 `api/outlines.ts`

### 🏗 工程质量（8项）
- **API 抽象层**: 新建 `frontend/src/api/outlines.ts`，封装后端调用
- **Wails 类型声明**: 新建 `frontend/src/types/wails.d.ts`，`AppAPI` 接口声明 100+ 方法，消除 `@ts-ignore`
- **TTS 配置持久化**: `config.go` 新增 `tts_port`/`tts_backend`/`tts_speed` 的 Load/Save
- **Save() 常量化**: 定义 19 个 `Key*` 常量替代硬编码字符串，防止拼写错误
- **SafeGo 工具**: `util/util.go` 新增 `SafeGo(fn)` — goroutine + panic recover + 日志
- **requirePM() 辅助**: `app.go` 新增读锁获取项目的方法，消除 handler 重复的 nil 检查
- **废弃文件清理**: 删除 `internal/ai/context.go`

### ⚡ 性能与体验
- **移动端 API 对接**: `mobile/handlers.go` `ProjectsProvider` 回调，替换硬编码 JSON 占位
- **统一日志**: 新建 `frontend/src/utils/logger.ts`，四级日志（debug/info/warn/error），生产环境可关闭
- **页面保活**: `MainLayout` `visitedPages` Set 机制，切 tab 不再销毁组件丢失状态
- **大纲五阶段固定卷**: `ContinueOutline(5)` 全量替换（起承转合终），简化 AI 生成流程
- **Z-Image NSFW LoRA**: 工作流新增 `LoraLoaderModelOnly` 节点 (strength 0.7)
- **世界地图图片**: `SaveWorldMapImage`/`GetWorldMapImage` Wails API，base64 存取

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.2.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **27 文件变更**: +1095/-346 行
- **新增 3 文件**: api/outlines.ts / types/wails.d.ts / utils/logger.ts
- **删除 1 文件**: internal/ai/context.go
- **零新依赖**

---
## v5.1.0「绘梦师」— Z-Image-Turbo 极速生图 + AI 绘梦工作台 + 角色剧照打通 (2026-07-03)

> 本地 Z-Image-Turbo 8 步生图（48s）、AI 绘梦双栏工作台重做、角色剧照与 AI 绘梦打通、角色详卡三段式布局

### ⚡ Z-Image-Turbo 本地生图集成
- **新模型支持**: ComfyUI-ZImagePowerNodes + GGUF Q5_K_M (~5.2GB) + Qwen3-4B Q4_K_M (~2.5GB) text encoder
- **8 步极速**: ZSamplerTurbo2Simple，48 秒出图（Flux 需 60-90 秒 20 步）
- **双模型切换**: Flux Dev / Z-Image-Turbo，前/后端完整支持，配置持久化
- **VRAM 优化**: UNet partial offload ~4GB 加载 + 1.4GB 卸载，8GB 显存刚好够用
- **v5.0.0 补全**: 下载脚本 + 工作流测试验证 + 节点参数自动探测

### 🎨 AI 绘梦工作台重做
- **双栏布局**: 左栏 320px 控制面板 / 右栏自适应结果区，桌面端专业工作台体验
- **负向 Prompt**: 可折叠输入区，Flux 工作流节点 8 注入
- **种子控制**: InputNumber + 🎲 随机，精准复现同一张图
- **批量生成**: 一次 1-4 张，循环提交 + 每张独立计时
- **20 个预设模板**: 创作/写实/风格/构图 4 大类，点击自动填入正/负向 prompt
- **全屏灯箱**: 键盘导航（← → Esc），显示种子/模型/尺寸/耗时，一键下载/重用参数
- **会话历史**: 底部横向缩略图画廊，点击回溯，支持清空
- **进度预估**: 按钮下方显示「预计 ~90s」/「上次 48s」，根据模型和数量动态计算
- **6 个新组件**: PromptPanel / GenControls / GenButton / ResultGallery / Lightbox / HistoryStrip
- **移动端适配**: 单栏控制面板 + 底部 Drawer 结果抽屉

### 👤 角色剧照与 AI 绘梦打通
- **剧照后端统一**: GeneratePortrait 改用 `cfg.ImageModel`（支持本地 Z-Image-Turbo/Flux），中文 prompt
- **AI 绘梦→角色剧照**: Lightbox 新增「设为剧照」下拉选择器，选角色自动保存到 `portraits/<id>.png` 并写回 `characters.json`
- **角色卡片缩略图**: 卡片列表圆形头像显示 `portrait_url`（有图时），无图时回退图标
- **SetCharacterPortrait API**: Go `character.Agent.SetPortrait()` + Wails 绑定

### 📋 角色详卡三段式重做
- **概览区**: 剧照 160×160 左侧主视觉 + 姓名/类型/状态/性格标签 + 操作按钮
- **Tab 分组**: 「📋 档案」（基本信息+性格+外貌+身材+动机+弧光+背景）/「🔗 关系」（组织+人物关系+出场章节）
- **剧照三态**: 有图悬浮重生成 / 生成中 Spin / 无图虚线框 CTA
- **删除按钮**: 移至右上角，减小误触

### 🔧 修复
- **config.go**: `comfyui_url` Save case 空白无赋值（严重bug）→ 正确赋值 `ComfyUIURL`
- **config.go**: `image_save_dir` Save case 错误写入了 `ComfyUIURL` → 修正为写入 `ImageSaveDir`
- **config.go**: `ImageModel` 配置字段新增 + Load/Save 支持

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.1.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **新增 7 文件**: 6 个前端组件 + 模板数据
- **修改 9 文件**: Go 后端 5 + 前端 4
- **零新 npm 依赖**

---

## v5.0.0「织梦者·移动」— 移动端远程操控 + M3 设计语言 (2026-07-02)

> 手机远程操控桌面端、Material Design 3 设计迁移、ComfyUI LoRA 生图链、20 commits

### 🎨 Material Design 3 全站迁移
- **M3 Tonal Palette**: 轻量内联实现，5 套主题从种子色自动生成 13 级色调调色板
- **Ant Design Token 驱动**: ConfigProvider 模拟 M3 视觉（surface/surfaceContainer/elevation/outline）
- **CSS 变量标准化**: `--md-sys-color-*` / `--md-sys-elevation-*` M3 命名体系
- **Layered Glass → M3 Surface**: `.glass-panel` → `.md-surface-container`，移除全站 backdrop-filter
- **M3 交互**: CSS-only ripple 涟漪效果、`.md-card` elevation hover lift、触控 44px 最小区域
- **25 变量兼容填充**: 旧 CSS 变量名 → M3 token 映射，零破坏迁移

### 📱 移动端远程操控
- **HTTP 服务**: Go `internal/mobile/` 包 — LAN IP 检测 + 二维码生成 + SPA fallback
- **响应式双导航**: 桌面端 Sidebar / 移动端 Bottom TabBar + Drawer + AppBar
- **Container Query 布局**: `useMediaQuery` hook + 3 级断点（compact/medium/expanded）
- **通用移动组件**: MobileSheet（底部滑出面板）、LongPressable（长按 500ms + 震动反馈）
- **全页面适配**: 9 个页面 + 5 个组件 — 3D 关系图→2D SVG 降级、Canvas→Grid、编辑器→全屏+Sheet
- **设置页**: Switch 开关 + 二维码面板，手机扫码即连

### 🎨 ComfyUI 集成增强
- **内嵌 Web UI**: ImageGenPage 一键切换 iframe 控制台，无需另开浏览器
- **LoRA 链**: Flux.1 Dev → Realism (0.8) → NSFW (0.6)，LoraLoaderModelOnly 级联
- **URL 持久化**: `GetConfig('comfyui_url')` 自动加载

### 🔧 工程质量
- **零新依赖**: 前端无新 npm 包，Go 纯 `net/http` 标准库
- **Wails 绑定**: 3 个新方法 `StartMobileServer` / `StopMobileServer` / `GetMobileServerStatus`
- **20 commits**: 每个 Task 独立审查（Spec ✅ + Quality Approved），2 个 fix round
- **前后端双编译**: `tsc -b && vite build` + `go build` 零错误

### 📦 构建
```bash
wails build                              # 生产包 (build/bin/wubigork.exe, 16MB)
go build -o build/bin/wubigork-v5.0.0.exe .  # 开发包 (不含嵌入 dist)
```

---

## v4.0.0「织梦者」— AI 原生叙事工作室 (2026-06-29)

> 对标 Scrivener + Sudowrite + Cursor + Obsidian + Notion，从「AI 辅助写作工具」跃升为「人机共创的叙事操作系统」

### 🏗 数据基础: 场景优先架构
- **场景引擎**: 章节由场景组合，场景级 CRUD + 元数据（POV/地点/情感/标签），`Stitch()` 拼接编辑
- **快照系统**: 行级 diff 存储，自动快照 + 时间线恢复 + 双栏对比
- **v4 项目结构**: `chapters/NNN/scenes/` 目录，非破坏性 v3→v4 迁移（原件备份到 `_v3_backup/`）

### ✍️ AI 协写系统 (对标 Cursor + Sudowrite)
- **Ghost Text**: 内联 AI 补全，800ms 防抖，Tab 接受/Esc 取消，光标追踪
- **Cmd+K**: 6 个预设指令 + 自定义指令，选中文本 → AI 原地编辑 → diff 预览
- **Diff Review**: 行级 diff（绿色新增/红色删除），⌘Y 接受/⌘N 拒绝
- **Beat-to-Prose**: 两栏布局，AI 生成叙事节拍 → 逐个展开为正文，完成自动存为 v4 场景

### 🕸 叙事知识图谱 (对标 Obsidian + Notion)
- **[[wiki-link]]**: 双向链接解析 + 反向链接索引 + 未链接提及检测
- **EntityDB**: 6 种实体类型（角色/地点/物品/事件/概念/组织），关联查询
- **一致性守护**: 跨章属性冲突检测（眼睛/发色）+ 角色状态异常（Dead→出场）+ 时间线校验
- **StoryGraph**: Canvas 力导向 2D 图谱 + 悬停高亮 + 6 色图例

### 🧠 上下文智能引擎 (对标 NovelAI + Cursor)
- **语义记忆**: BM25 检索 + 中文 2-gram 分词，零外部依赖
- **Lorebook 2.0**: 触发式关键词注入 + Token 预算可视化（7 分区堆叠条形图）
- **多模型路由**: 8 种任务类型 → 最优参数自动映射（温度/Token/推理深度）
- **@-mention**: 弹出实体选择器，@角色/@地点/@概念 精确控制 AI 上下文

### 📊 视觉叙事工作台 (对标 Scrivener Corkboard)
- **StoryTimeline**: 水平滚动时间线 + 彩色情绪卡片 + 字数比例条 + 悬停详情
- **EmotionCurve**: Canvas 贝塞尔曲线 + tension/valence 切换 + 渐变填充
- **CharacterHeatmap**: 角色×章节矩阵 + POV 红色标记 + 颜色强度
- **CanvasCards**: SVG 连线软木板 + 30%-200% 缩放

### 🌐 生态平台
- **导出 2.0**: HTML 导出 + 3 个 Compile 模板（网文阅读/出版审阅/极简纯净）+ 暗色模式
- **写作仪表盘**: 8 个写作成就 + 统计总览 + 章节柱状图 + 目标进度
- **风格档案**: AI 分析 3 章样本 → 8 维风格特征 → 自动注入 prompt

### 🔧 工程质量
- **Go**: 6 个新包 (scene/snapshot/graph/memory/context/visual) + 34 项测试
- **React**: 14 个新组件 + TypeScript 零错误
- **API**: 60+ Wails 绑定方法
- **兼容**: v3 项目完全兼容，迁移可手动触发

---

## v3.1.0 — Layered Glass 视觉重设计 (2026-06-21)

### 🎨 UI 全面升级 — "Layered Glass"
- **设计系统**: 17 个新 CSS 设计令牌 (accent-rgb, shadow-sm/md/lg/glow, border-subtle, bg-deep/base/elevated/glass, radius-sm/md/lg/xl, transition-fast/normal/slow)，4 套主题统一渐变+辉光
- **玻璃态全站**: 所有面板/卡片 `backdrop-filter: blur(8px)` + 半透明背景 + 柔和边框替代 thin 1px 实线
- **导航重设计**: 顶栏 sticky glass + 菜单去下划线/pill 选中态 + 底栏玻璃态
- **XAI 控制台**: 从等宽字体调试面板 → 圆角玻璃浮层 + 系统字体 + 彩色左边线
- **首页**: 项目卡片 glass-card 悬停上浮 + 空状态品牌水印 + 按钮辉光
- **写作页**: 玻璃侧边栏 + pill 标签页 + 编辑器内凹 inset 阴影 + 工具栏统一玻璃按钮
- **对话框**: Ant Design Modal 全局玻璃覆盖 + 弹簧入场动画 + 遮罩 blur
- **骨架屏**: 三处 loading 从裸 Spin 升级为 Ant Design Skeleton 骨架屏
- **无障碍**: `prefers-reduced-motion` 全局尊重

### ✨ 功能增强
- **TTS 语音朗读**: VoxCPM 本地神经网络合成 + Edge TTS 在线自然语音 + Windows SAPI 零延迟
- **写作页标签页优化**: 可横向滚动 + 关闭未保存确认弹窗
- **设置页**: 工作空间目录可配置

### 🔧 代码质量
- **DRY 清理**: 消除 9 处 `C()` 重复定义 → 统一 `import { C } from utils/theme`；消除 RelationGraph 颜色映射重复；提取 `sortNodes` 到 `utils/outline.ts`
- **错误处理**: 消除 15+ 处 `json.Marshal`/`io.ReadAll` 静默吞错误
- **死代码清理**: 删除 `internal/xai/` + `pkg/novel/` 重复函数 ~350 行

### 🛠 修复
- Edge TTS WebSocket 重写 + 引擎链降级 (SAPI → Edge → VoxCPM)
- VoxCPM 编译为静态链接，消除 DLL 依赖
- exec.Command 隐藏控制台窗口，消除朗读时弹 cmd
- NovelsDir 默认值不再依赖配置文件

---

## v3.0.0 — 革命性跃升 (2026-06-20)

### 🚀 核心创新
- **剧情分支选择器**: AI 推理 3-5 个下一章方向，用户选用或手工录入，自动写入大纲并同步角色/世界观
- **全书发展编辑**: AI 审读全书生成 1500 字编辑信 + 5 维评分 + 角色弧光诊断，对标 $3500 人工编辑
- **自我演化引擎**: 每章生成后自动分析 → 建议 Lorebook 词条 → 追加世界观空维度 → 记录伏笔变化
- **跨模块协作**: 大纲角色点击跳转角色页 + 角色出场章节查询 + 世界观一致性一键编辑

### 🎯 v2.0.0 功能 (2026-06-20)
- Story Bible 引导式创建（5 步向导：灵感 → 一键生成 → 角色优化 → 大纲优化 → 开写）
- 情节画布（水平时间线可视化全文章节 + 角色色条 + 品质情绪标注）
- Skill 管理面板（浏览/导入/创建自定义写作风格 Markdown）

### 💡 v1.5.0 功能 (2026-06-20)
- Lorebook 词条系统（定义概念 → AI 写作时自动注入相关上下文）
- 写作统计仪表盘（每章字数柱状图 + 品质趋势进度条）
- AI 审稿（5 维评分环形图 + 改进建议高亮卡片）
- Brainstorm 脑暴面板（输入题材 → AI 生成 6 个核心点子 → 选用创建）

### 🔧 v1.4.0 功能 (2026-06-20)
- 右键 AI 操作（选中正文 → 丰富描写/扩展场景/重写此段）
- 多章节标签页（同时编辑多个章节，独立状态）
- 大纲拖拽排序（Ant Design Tree draggable + 批量保存）
- 专注写作模式（全屏沉浸，隐藏侧栏和 Agent）

### 🛠 v1.3.2 修复 (2026-06-20)
- 大素材归纳压缩（先 AI 归纳关键信息再注入 prompt，防止上下文溢出）

---

**技术栈**: Go + Wails v2 + React 19 + Ant Design 6 + Three.js + d3-force-3d  
**构建**: `go build -o build/bin/wubigork-v4.0.0.exe .`  
**许可证**: MIT
