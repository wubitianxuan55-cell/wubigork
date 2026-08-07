# 任务进度

> 最后更新: 2026-08-08 00:10:00

## 2.25 通用办公欢迎页重设计（✅ 已完成，2026-08-08，v2.4.5）
- 移除土壤修复专项卡片（场地调查/标书/修复方案/污染风险/成本测算），改为 6 个通用办公核心能力：
  文档撰写/表格处理/格式转换/图表生成/报告拼装/知识沉淀
- 新增内置技能 chips：format-convert / chart-builder / doc-assemble / docx / xlsx / pdf，点击填入任务提示词
- 视觉升级：GAEA OFFICE + 「今天想做什么？」主视觉、能力卡 hover 顶部高光线 + 箭头、渐次入场动画
  （.welcome-rise-*，尊重 prefers-reduced-motion）；保留工作区/模型 pill 与最近会话并优化
- 文案清理：welcome.tagline 与 skeleton 加载描述改为通用办公，删除 locale 中土壤修复死条目
- 验证：tsc + vite build OK、vitest 37 例全过、Edge headless 实拍确认布局

## 2.24 文件预览与右侧面板精简（✅ 已完成，2026-08-07，v2.4.4）
- 右侧面板删除「消息」「报告」标签页（MessageNavigator/ReportPreviewPanel 组件一并删除），保留文件/统计，面板默认折叠
- 对话内文件可点击预览：Markdown 本地路径渲染为文件 chip、@ 附件可点，打开居中大尺寸预览弹层
  （FilePreviewModal：Esc/遮罩关闭、文件管理器定位、外部打开）
- 后端新增 GaeaPreview 统一预览接口：图片 dataURL、md/文本原文、docx/xlsx/pdf 转 Markdown 内联预览（含 OCR 回退），不支持格式给外部打开入口
- 转换引擎抽取至 internal/office/docmd（format_convert 工具与预览共用），修复 openpyxl inlineStr 丢表头
- 工作区「文件」面板预览同步升级为 Markdown 渲染；修复旧 ReadFile 字段不匹配导致的空预览
- 测试：新增 FilePreviewModal 3 例 + Markdown 文件链接 2 例，前端 37 例全过

## 2.23 内置工具精简 38→17（✅ 已完成，2026-08-07，v2.4.3）
- 按市场调研结论（同类产品 10~20 核心工具、文档用技能扩展）删除 21 个内置工具：
  计算器/压缩/电脑操作/甘特图/项目初始化/模板链/方案 agent 工具 + 被 ModelScope 技能覆盖的文档专项工具
- 删除与 run_skill 重叠的 parallel_skills（保留 RunDAG 管道基础设施）
- 保留 17 个核心工具；内置技能去工具依赖（chart-builder 用 bash+python 读表、doc-assemble 用 format_convert+bash）
- 修复 chart_gen Windows python3 商店别名劫持；安装 matplotlib 3.11.1；compact 死条目清理
- 前端能力抽屉工具列表重建为实际 7 组；README/CHANGELOG/wails.json 同步

## 2.22 通用办公改造 + ModelScope 文档技能（✅ 已完成，2026-08-07，v2.4.2）
- 「智能办公」精简为「通用办公」：删除 6 个土壤修复专项工具与 5 个土壤办公技能
- 内置子代理技能收敛为 format-convert / chart-builder / doc-assemble（wrapper 工具 + 测试）
- 安装 ModelScope docx/pdf/xlsx 技能至 ~/.codex/skills 与 .gaea/skills（127 文件）
- 环境：LibreOffice 26.2、npm 全局 docx、Python 依赖（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/pdf2image/pytesseract）
- 系统提示词与单模型纪律改写为通用办公；端到端验证 docx 生成→PDF→validate→comment→pack、xlsx recalc

## 2.21 设置面板重设计（✅ 已完成，2026-08-07）
- 审计结论：EnginePanel 为死代码（从未引用）；「方案」Tab 与「办公」重复且一半是纯文档；
  「语音」面板与聊天页 VoiceSettingsPanel 重复；「聊天」面板的主动搭话开关无人读取、清除会话与聊天板块重复
- 布局重设计：顶部 Tab 改为左侧功能分组导航（通用 / 聊天 / 小说 / 绘梦 / 办公 / 模型 / 关于），
  窄屏转横向滚动 chips；新增 `settings-page.css`，保留全局搜索并可过滤分组，激活分组自动切换
- 聊天（新 ChatPanel）：合并 AI 伴侣（称呼/性别）+ 默认人格（角色库管理提示）+ 语音核心项
  （语音对话/朗读回复/合成音色），移除死开关与重复数据管理；完整语音面板仍在聊天板块
- 办公：并入「方案编写模型」FeatureModelBar（原方案 Tab 唯一有效内容），删除「方案生成」纯文档列表
- 模型（新 ModelPanel）：找回丢失入口的全局「推理强度」（saveConfig reasoning_effort），
  展示当前模型，新增「前往模型中心」跳转（navigate 事件）
- 关于（新 AboutPanel）：版本卡片 + 系统信息（配置路径，替换 antd Descriptions 为普通定义列表，
  消除 jsdom 下 matchMedia 依赖与 labelStyle 弃用警告）+ 可折叠更新日志
- 清理：删除 EnginePanel/VoicePanel/ProposalPanel/WhisperPanel/SystemPanel 五个文件；
  SettingsSection 即时生效徽章改用主题 success 令牌
- 测试：新增 SettingsPage.test.tsx 4 例（七个分组渲染/默认通用/搜索过滤自动切换/关于分组），
  `npm run test` 共 32 例全过；`tsc` + `vite build` OK；`scripts/ci.ps1` CI OK；
  设置页主包 77.6KB → 56.8KB

## 2.20 网页调试桥接 + 办公引擎热加载（✅ 已完成，2026-08-07）
- 新增 `internal/httpbridge`：`POST /api/rpc` 反射分发全部 Wails 绑定方法（缺失尾参补零值、panic 转 error）、`GET /api/stream?id=` SSE 事件推送（15s keep-alive + connected 帧）、`/api/health` 存活探针；`core.emit` 统一发布到桥接订阅者（无 Wails 上下文也推送，网页/移动端调试与桌面端完全对齐）
- `main.go`：设置 `GAEA_HTTP_PORT`（如 8080）后自动启动桥接，Vite 把 `/api` 代理到桥接
- 前端 HTTP 模式：`runtimePolyfill` 补齐 EventsOn/EventsOff/EventsOnMultiple/EventsEmit，所有事件经 SSE 对齐桌面端；修复并发订阅只连首个事件的探测竞态（未就绪事件入队、成功后全部建连、失败可随下次订阅重探）；`bridge.ts` 将 `window.go.app.App.*` 代理到 `/api/rpc`
- 办公引擎热加载：`GaeaReload` 重新读取磁盘持久化配置并重建 controller（Agent 参数/权限/沙箱/技能路径/插件），成功返回工具/技能数量并广播 gaea-ready（kind=reloaded）令前端 store 重新拉取；失败保持旧引擎继续运行
- 前端入口：能力抽屉（MCP/工具/技能）标题行新增「热加载」按钮（成功后展示 N 工具 · N 技能，失败内联提示）；设置→办公新增「从磁盘热加载」；三语 i18n 同步
- 测试：httpbridge RPC 分发/未知方法/缺参补零/SSE 发布 2 例 + `TestGaeaReloadHotLoadsConfig`（磁盘改温度 0.2→0.85 热加载生效）共 3 例
- 验证：`go build/vet/test` 全绿、`tsc` + `vite build` OK、`scripts/ci.ps1` CI OK；版本 2.3.0 → 2.4.0（wails.json / versioninfo.rc / CHANGELOG / README）

## 2.19 角色详情：一键随机补齐 + 生成剧照（✅ 已完成，2026-08-07）
- 后端新增库作用域方法 `CharacterGenerateFill`：`character-generate-single` 模板（正确注入 name/worldview/story_thread）+ novel 功能级路由 → 只填空字段合并（role_type→roleType 归一、空 tags 才填、已有内容一律保留）；无项目时也能用（有项目则借题材/世界观）
- 后端新增 `CharacterGeneratePortrait`：按角色字段构建中文智能 prompt（外貌/身材/气质/标签 + 半身像构图，跳过空字段），复用 `GenerateFreeImage` 管线（ComfyUI/xAI/Herdsman/Ollama 自动恢复与落盘）
- 前端：详情「小说设定」分区标题行加「随机补齐」（提示已补齐 N 处/无空缺），立绘横幅右上加玻璃 pill「生成剧照」（返回后即时预览，保存才落库）；空名拦截
- 绑定：`wails generate module` 再生成 `CharacterGenerateFill` / `CharacterGeneratePortrait`
- 测试：Go 单测 5 用例（只填空合并/role_type 归一/空 tags/剧照 prompt 构建与跳空/摘要）+ 前端 25 用例 PASS（新增补齐回填、剧照更新横幅、空名拦截）
- 验证：`go build/vet/test`、`tsc -b`、`npm run build`、`scripts/ci.ps1` CI OK、`wails build` 成功
- 文档：`docs/superpowers/specs/2026-08-07-角色详情补齐与剧照-design.md` + `plans/2026-08-07-角色详情补齐与剧照.md`（Task 3 人工 UI 目检待验证）

## 2.18 角色卡「详情档案」重设计（✅ 已完成，2026-08-07）
- 按 `redesign-existing-projects` skill 审计并重写 `CharacterLibEditor`：档案眉（编号/类型/可聊天/当前人格徽标）→ 21:9 立绘横幅（图片/首字占位 + 光泽 + 名称元数据叠层）→ 左侧身份栏（名称/立绘 URL/性别/年龄/定位/状态/标签/五维雷达+滑块/可聊天开关）→ 右侧卷宗正文（小说设定/对话样本/备注/聊天设定发丝线分区）→ 底部操作条（项目引用 + 取消/保存 pill）；字段与保存语义不变
- 档案卡整卡可点击进入详情：`CharacterCard` 可选 `onClick` + `role="button"`/`tabIndex`/Enter/Space 键盘支持，操作按钮 stopPropagation 不误触
- 列表页接线：卡片点击走 `openEdit`，详情同步当前人格徽标与列表编号
- 测试：新增 `CharacterLibEditor.test.tsx` 10 用例 + `CharacterCard` 点击/键盘 3 用例，共 22 用例 PASS
- 验证：`npx tsc -b` 无错误、`npm run build` 成功、`scripts/ci.ps1` CI OK、`wails build` 成功（build/bin/gaea.exe）
- 文档：`docs/superpowers/specs/2026-08-07-角色卡详情重设计-design.md` + `docs/superpowers/plans/2026-08-07-角色卡详情重设计.md`（Task 4 人工 UI 目检待验证）

## 2.17 角色库「人物档案卡」重设计（✅ 已完成，2026-08-07）
- 首次引入 Vitest 前端测试设施：`vitest@4 / jsdom@30 / @testing-library/react@16`，`test` 脚本 + vite test 配置 + setup 自动 cleanup
- 新增档案卡组件 `CharacterCard`（独立 CSS）：档案眉编号/类型/可聊天 → 4:3 立绘横幅（图片/首字占位 + 悬停光泽）→ 正文（名称/元数据发丝线/弧线引言/最多 3 个 # 标签）→ 底部（五维雷达 + 悬停操作区）；当前人格角标、已加入标记、`prefers-reduced-motion` 降级
- 列表页接入：网格改用 `CharacterCard`，头部发丝线分隔；当前人格状态（localStorage + `gaea-persona-changed` 事件同步）；清理旧内联卡片与未用导入
- 组件测试 9 用例（编号/元数据/弧线/标签/立绘/雷达/回调/当前人格/已加入）全部 PASS
- 验证：`npx tsc -b` 无错误、`npm run build` 成功、`scripts/ci.ps1` CI OK、`wails build` 成功（build/bin/gaea.exe）
- 计划文档 `docs/superpowers/plans/2026-08-07-角色库档案卡重设计.md` 勾选完成（Task 4 的 UI 目检与聊天联动回归待人工验证）

## 2.0 P0 基线加固（✅ 已完成）

- frontend/package.json 重建并纳入版本控制（基线可构建）
- 根 dist/ 由前端构建产出，go build/vet/test 全绿
- scripts/ci.ps1 基线闸门（build/vet/test/frontend）
- E01-E24 评估集重建：docs/evaluation-set.md（可达性标注）
- 首批自动化回归：E11 工作区沙箱 / E06 解析合并去重 / E01 工作流稳定
- 基线修复：empty.pdf 坏夹具重建、parseLoraNames map 顺序稳定、findChangelogPath cwd 优先（E21 类测试隔离）
- 基线：main v1.21.0（9093930），P0 在 codex/gaea2-p0-baseline 分支

## 2.0 P1 模型路由（✅ 已完成）

- routeModel 降级链：功能绑定 → 全局活跃 → 首个可用引擎；model.route 事件可观测
- novel 系收敛：章节/分支/风格（E03）；whisper 收敛（空模型直连修复）
- office/gaea 绑定回归（E09/E10）；GetModelRoute 绑定 + 模型中心"当前生效"展示
- 已知限制：CmdKEdit 无 EngineID 参数（模型名已路由，引擎随活跃引擎）；outlineAgent 在 1.x 从未初始化（死路径）

## 2.0 P2 三脑记忆（✅ 已完成）

- BrainStore 统一访问层：Read/Write/Search/Link/CrossRefs
- 主脑适配器（画像+知识库）、右脑适配器（hermes.db）、左脑适配器（方案）
- brain_links 跨脑关联索引（Hephaestus.db 零迁移建表，内存/SQLite 双模式）
- BrainWrite/BrainSearch/BrainCrossRefs 绑定 + 记忆中枢三脑检索区块
- 验收：右脑"甲方A 保守报价" → 跨脑检索同时命中右脑与左脑标书；现有数据零迁移

## 2.0 P3 主脑助手（✅ 已完成）

- ModuleRegistry 模块注册与统一派发 + RunModule 绑定（gaea/whisper/novel/office/imagegen）
- 主脑意图识别（规则分类）+ MainBrainChat 后端能力（跨脑材料 + 模块派发 + 汇总）
- 定位：可选编排入口，不经由任何模块的直接路径；前端主脑页不做（留待 3.0）
- 验收：一句话任务派发到正确模块，跨脑材料随结果返回；模块直达路径不受影响

## 2.0 P4 模块互联验收 + 3.0 接口预留（✅ 已完成）

- 方案生成注入跨脑记忆（buildBrainMaterials：右脑甲方偏好 → 写作上下文，最多 3 条去重）
- 完成判据回归：右脑"保守报价" → 方案上下文可命中；主脑一句话派发可用
- 3.0 模块协议文档：docs/gaea2/module-protocol.md（ModuleRegistry/RunModule/MainBrainChat/三脑访问）
- 基线加固：TestChatSimpleStream flaky 修复（SSE Flush + token 过期边界）

## 2.0 P5 发布 2.0.1（✅ 已完成）

- 版本号：wails.json / versioninfo.rc / CHANGELOG → 2.0.1
- 构建：wails build 成功（build/bin/gaea.exe 38MB），SHA256SUMS-v2.0.1.txt 已生成
- 发布文档：releases/v2.0.1.md；Wails 绑定更新（RunModule/BrainSearch/MainBrainChat 等）
- 备份：scripts/backup.ps1（whisper_data/novels/配置 → backups/ 时间戳目录，已运行验证）
- 全量验收：scripts/ci.ps1 CI OK，工作区干净

## 2.0 发布与推送（✅ 已完成）

- 桌面端最终构建：wails build → build/bin/gaea.exe（38MB），已复制到桌面
- 校验和：releases/SHA256SUMS-v2.0.1.txt（b57b4452…）
- Git 标签：v2.0.1（annotated）
- 远程推送：main + v2.0.1 → origin

| 状态 | 任务 |
|------|------|
| ⬜ | Write tests |
| ✅ | Add parser |

## 2.1 模型中心完善（✅ 已完成，2026-08-07）

- 引擎状态持久化：`whisper_data/engines.json`（enabled / base_url / default_model / models / 最近连接状态缓存），启动自动恢复，任何变更自动落盘；API Key 不入状态文件
- 修复 `active_engine_id` 只存不读：`config.Load()` 恢复全局活跃引擎（此前重启必然回退 xai）
- 稳定引擎顺序：`GetEngines` 固定 xai → ollama → herdsman → deepseek（消除 map 随机序）
- 连接状态可观测：`EngineConfig.status` 缓存 + 前端「已连接/失败 + 模型数 + 上次检查时间」；测试连接/刷新后即时回填
- 前端修复：DeepSeek Key 脱敏字段映射（masked）、挂载时加载真实活跃模型、ComfyUI 端口死代码清理、保存 Key 后清空输入框（避免把脱敏串当 Key 保存）
- 验收：新增 8 个回归测试（持久化往返/排除 Key/未知引擎忽略/顺序稳定/状态缓存/状态随文件恢复/ActiveEngineID 读取），`scripts/ci.ps1` CI OK

## 2.2 功能级模型启停（✅ 已完成，2026-08-07）

- 语义修复：FeatureModelBar「启动/停用」此前切换整个引擎的 enabled（停用轻语=全局关掉 xAI），改为功能级开关 `func_*_enabled`（只影响该功能路由，停用后回退全局）
- 后端：config 新增 5 个 `func_*_enabled` 键（默认启用、`*bool` 区分显式停用）；`SetFeatureModelEnabled`/`GetFeatureModelEnabled` 绑定；`routeModel` 功能绑定步骤增加启用门控；重新绑定自动恢复启用
- 前端：`useFeatureModel` 返回 enabled 并监听事件；FeatureModelBar 状态细分「运行中/已停用/引擎已停用/未绑定」，启停调功能级接口；模型中心「功能绑定」卡片新增启用开关
- 绑定：wailsjs 再生成（`wails build`），仅新增 Get/SetFeatureModelEnabled 两个方法
- 验收：新增 6 个回归测试（enabled 配置往返/停用回退全局/重绑恢复/未知功能报错/持久化），E03/E09/E10 路由回归保持绿，`scripts/ci.ps1` CI OK

## 2.3 Cmd+K 编辑引擎路由（✅ 已完成，2026-08-07）

- 修复 P1 已知限制「CmdKEdit 无 EngineID 参数」：此前 `Client.CmdKEdit` 只用模型名、引擎随活跃引擎，`App.CmdKEdit` 直接传全局 `cfg.Model`——小说编辑器里的 AI 改写完全忽略 novel 功能绑定
- `Client.CmdKEdit` 新增 engineID 参数并透传 `ChatSimpleOptions.EngineID`（空=活跃引擎回退，兼容既有调用）
- `App.CmdKEdit` 改走 `routeModel("novel")`：绑定引擎+模型进入请求，模型名不再取全局陈旧值
- 防御：`a.ctx` 为空时回退 `context.Background()`（测试/异常路径不 panic）
- 验收：新增 2 个回归测试（ai 层 EngineID 路由命中绑定引擎、app 层 novel 绑定生效且不误走 xAI），`scripts/ci.ps1` CI OK

## 2.4 xAI OAuth 回归验证（✅ 已完成，2026-08-07）

- 修复：`DiscoverEndpoints` 此前硬编码 `https://auth.x.ai/...`，`cfg.OIDCDiscoveryURL` 与 `XAI_OIDC_DISCOVERY_URL` 环境变量完全失效（刷新链路也因此无法用测试桩验证）→ 改为取配置 URL（默认不变）
- 加固：`exchangeCodeForToken` 从无超时的 `http.PostForm` 改为 15s 超时客户端（与 `RefreshAccessToken` 一致，E04 换 token 挂起场景可及时失败）
- 回归（E04/E13 状态更新）：新增 8 个测试——discovery 配置化、换 token 成功（verifier/challenge 齐全）/500/403/空 verifier、刷新 token 成功/500/403；referrer=wubigork 已有 TestBuildAuthURL 断言
- 验收：`scripts/ci.ps1` CI OK；docs/evaluation-set.md 中 E04/E13 状态改为「已转回归测试」

## 2.5 前端 E 系列核对 + 回归守卫（✅ 已完成，2026-08-07）

- 逐项核对四项前端历史修复均已在代码落地：
  - E16 QUICK_REPLIES 常量在模块顶层（WhisperPage.tsx:54，组件在 :77）
  - E22 AI 控制台降级动画禁用（index.css `html.gaea-raf-degraded .ai-console-panel`）
  - E23 WebView2 rAF 节流降级（main.tsx ensureRAF：帧率<30fps 降级 setTimeout(16ms) + index.css antd motion enter/leave 禁用）
  - E24 记忆中枢 3D 图谱用 3d-force-graph（GraphView 默认导入 + 正确初始化，MemoryHubPage 已挂载）
- 新增 `scripts/frontend-e-check.mjs` 静态回归守卫（无前端测试框架时的不变式检查），接入 `scripts/ci.ps1`——四项任一回归即 CI 拦截
- 验收：守卫本机运行全过，`scripts/ci.ps1` CI OK；docs/evaluation-set.md 中 E16/E22/E23/E24 状态改为「已核实修复 + 回归守卫」

## 2.6 小说链路审计：剧照泄漏 + 全局模型回退（✅ 已完成，2026-08-07）

- E02 剧照泄漏（确认存在并修复）：角色对话/详情/批量生成的 prompt 把整个 `CharacterFile`（含 ComfyUI base64 剧照）序列化注入；大纲 `loadCharsContext`、章节分析 `Analyze` 同样泄漏 → 新增 `types.Character.PromptView`/`CharacterFile.PromptView`（剥离 portrait_url，深拷贝），五处注入统一收口
- E03 全局模型回退（确认存在并修复）：角色/大纲/章节/分析/世界观 5 个 agent 的 `chat()` 在 novel 未绑定时强制 `model = cfg.Model`（把 xAI 默认名 grok-4.20 发给非 xAI 活跃引擎会 404）→ 改为留空让客户端按活跃引擎解析默认模型（等价 routeModel 全局路径）；章节/分支/Cmd+K/风格此前已走 `routeModel("novel")`
- 新增 6 个回归测试：角色 Chat/详情 prompt 无剧照、未绑定留空解析、绑定生效、PromptView 剥离与深拷贝
- 验收：`scripts/ci.ps1` CI OK；docs/evaluation-set.md 中 E02/E03 状态更新

## 2.7 发布 2.1.0（✅ 已完成，2026-08-07）

- 版本号：wails.json / versioninfo.rc / CHANGELOG / README → 2.1.0（versioninfo.rc 顺手修正陈旧的 ProductVersion 1.13.0）
- 构建：wails build 成功（build/bin/gaea.exe 38MB），已复制到桌面 + releases/gaea-v2.1.0.exe
- 校验和：releases/SHA256SUMS-v2.1.0.txt（d76d93ac…）
- 发布文档：releases/v2.1.0.md（2.1–2.6 六轮迭代摘要）
- 备份：scripts/backup.ps1 已运行（whisper_data/配置 → backups/ 时间戳目录）
- 全量验收：scripts/ci.ps1 CI OK，工作区干净
- Git 标签：v2.1.0（annotated）；远程推送：main + v2.1.0 → origin
- 说明：wails v2.13 不把根目录 versioninfo.rc 编译进 exe（历史版本同），版本资源缺失为仓库既有状态

## 2.8 聊天×轻语后端合并（✅ 已完成，2026-08-07）

- 统一入口：`App.ChatSend(topicID, message, mode)`——`plain` 走普通对话（featureModel("chat")），人格 ID 走轻语 Orchestrator（记忆/情绪），persona 返回保留情绪/信任/轮次等元数据
- 统一会话存储：新增 `internal/chat`（chat.db，SQLite，与 office.db 同模式）——话题 CRUD + 消息（seq 自增、级联删除）；绑定 `ChatTopicsList/Create/Rename/Delete/MessagesList`
- 绑定合并：`func_whisper_*` → `func_chat_*`——旧配置自动迁移（chat 显式配置优先，`func_whisper_enabled=false` 同步为 chat 停用）；`Get/SetFeatureModel("whisper")` 与启停均走 chat 别名；写入侧删除 whisper 键，微信/助手/主脑模块统一经 chat 路由
- 防御：`core.emit` 预检 Wails ctx（"events" 标记），非 Wails 上下文跳过发射——修复测试中异步记忆写入经 routeModel emit 触发 `log.Fatalf` 杀进程的问题（生产行为不变）
- 验收：新增 7 个测试（whisper→chat 迁移/chat 优先、chat store CRUD/级联删除、ChatSend plain/persona 落库与元数据）；wails build 再生成绑定（+6 方法）；`scripts/ci.ps1` CI OK
- 遗留：前端两页合并 + localStorage 话题导入 chat.db 留待下一轮

## 2.9 聊天×轻语前端合并 + 市场调研（✅ 已完成，2026-08-07）

- 市场调研：`docs/market-research-2026-08-chat-ui.md`（Claude 式居中流、陪伴产品临场感、CDT 暗模式红线、Liquid Glass、会话可读性、错误就近、无障碍）→ 落地为本次设计决策
- 设计：desktop AI 陪伴聊天 · 单人重度用户 · 克制暗色玻璃语言，沿用 gaea M3 tokens（night 系列），Liquid Glass 为 web 近似（外层壳 + 内芯高光）；dials variance 6 / motion 5 / density 5
- 前端合并：ChatPage 重写为单一聊天板块——模式切换条（普通对话 / 轻语人格）、人格状态条（CompanionAvatar + 情绪/信任/轮次只读元数据）、右侧记忆抽屉（状态/记忆/追踪三页）、Claude 式消息流（助手通栏文本 max 820 + 用户右侧轻量胶囊）、双层玻璃输入岛、错误内联 + 重试、快捷情绪 chips、语音输入；删除 WhisperPage.tsx
- 会话可读性：ChatTopicSidebar 升级为标题 + 首条预览 + 模式徽章（`chat.Topic.preview` 由 ListTopics 子查询填充，避免 N+1）
- 数据迁移：旧 localStorage 话题（`gaea_chat_topics` / `gaea_whisper_topics` + legacy 键）启动时经 `ChatImportTopic` 导入 chat.db 后清理本地键；新增 `ChatTopicSetMode`（模式切换持久化）与 `ChatTopicClear`（清空对话真删消息）
- 导航收敛：MainLayout 移除独立「轻语」菜单项；首页启动器轻语卡片改为一键进入聊天板块 persona 模式（`utils/chatNav.ts` 信号 + 事件）
- 无障碍/防降级：消息区 `role="log" aria-live="polite"`、focus-visible 环、`prefers-reduced-motion` 禁用打字机与入场动画、`prefers-reduced-transparency` 玻璃降级；emoji 图标全部替换为 antd icons
- 验收：新增 4 个测试（ChatImportTopic 导入/跳过非法角色、ChatTopicSetMode、ChatTopicClear）；E16 守卫迁移到 ChatPage + 新增 E25 合并不变量守卫；`scripts/ci.ps1` CI OK；`wails build` 成功（build/bin/gaea.exe 38MB）

## 2.10 全局统一角色库（架构重构版，✅ 已完成，2026-08-07）

> 推翻 2.10 初版"双 Tab 套壳"设计，按用户三点反馈重构：①角色不绑定小说；②面向大量角色；③小说角色与轻语角色是同一资产。

- **统一数据模型**：新增 `internal/characterlib`（SQLite `characterlib.db`）。一张 `characters` 表承载全部角色——小说字段（定位/性格/背景/外貌/身材/动机/弧线/状态/备注/对话样本）+ 聊天字段（五维 TISOR/口吻指南/行为规则/情感逻辑/可聊天）长在同一行，另有来源类型（builtin/custom/assistant）、剧照、标签、时间戳
- **角色全局化，小说只是引用**：`project_characters` 关联表（项目目录 + 角色 + 项目内定位/弧线状态/状态），同一角色可被多本小说引用，删除小说页角色只移除引用、角色保留在库；项目 `characters.json` 变为从库物化的工作副本，小说 Agent 注入链路零改动
- **聊天直接用库**：29 个内置人格启动种子化为可编辑角色；`getOrCreateOrch` 优先从库解析人格（`ToPreset` 把行为规则/情感逻辑并入 voiceGuide）；`WhisperGetPersonalities` 返回库内可聊天角色，聊天人格列表与角色库同源；assistant 记录降级为"微信通道配置"（CharacterSave 自动同步/创建 assistant）
- **面向大量角色**：分页查询 + 名称/标签/性格搜索 + 类型筛选 + 可聊天筛选；`CharacterList` 绑定分页返回
- **双向一致**：打开/新建项目自动导入（幂等，按 ID 优先、名称去重合并，薄记录不覆盖丰富全局字段）；小说页 SaveCharacter/DeleteCharacter/GenerateCharacters/ChatCharacter 写后自动回写全局库；角色库"同步到项目"把引用物化回 characters.json
- **前端重写**：单一统一列表（搜索/筛选/分页/统一卡片），统一编辑器 `CharacterLibEditor`（基础/小说/聊天三段 + 五维滑杆 + 剧照 + 对话样本）；操作：设为聊天人格、加入当前项目/移出、同步、导入、删除（内置软隐藏）；删除初版双 Tab 与 WhisperRolePanel
- 新增绑定：CharacterList/Get/Save/Delete、ImportProject/ListByProject/Associate/Dissociate/SyncProject
- 验收：新增 10 个回归测试（characterlib 种子幂等/搜索筛选分页/内置软删自定义硬删/导入去重幂等/项目弧线状态物化/助手镜像/ToPreset；App 层保存→助手+聊天桥接、跨项目引用、物化回写、JSON 往返）；`scripts/ci.ps1` CI OK；`wails build` 成功（build/bin/gaea.exe 38MB）

## 2.12 单向约束：小说只使用角色库，角色面板改抽卡（✅ 已完成，2026-08-07）

- 按用户要求补上硬约束：**小说只是角色库的引用方，任何小说侧写入都不能反向污染全局角色**。删除此前"小说页写角色自动回写全局库"的反向路径（SaveCharacter/GenerateCharacters/ChatCharacter 不再触碰角色库；打开项目不再自动导入）
- 导入改为**只增不改**：`ImportProjectCharacters` ID 命中 → 仅补项目关联，绝不动库内记录；ID 不存在 → 以项目 ID 新建；不做名称合并（避免 ID 重映射破坏项目内关系引用）。旧项目数据在小说面板横幅提示"一次性迁移"
- **同步防误清保护**：`CharacterSyncProject` 检测到项目里还有未入库角色时拒绝覆盖，先导入再同步，防止同步把旧数据冲掉
- **小说角色面板重写**（CharacterPage）：删除"新建角色/批量生成/角色 Agent 对话/AI 补全/剧照生成/导入轻语"全部自建路径；改为「抽卡」——`CharacterDrawRandom` 从全局库随机抽（数量/性别/标签/可聊天过滤），抽中即加入本书；卡片点击只编辑**项目内覆盖**（定位/弧线状态/状态，写 `project_characters` 关联表），全局设定只读并引导去角色库编辑；组织/关系仍为项目内数据保留原能力
- 新增绑定：`CharacterDrawRandom`、`CharacterSetProjectState`；前端新增 4 个回归测试（小说保存不污染库/同步拒绝未入库/抽卡过滤与上限/导入只增不改）
- 验收：`scripts/ci.ps1` CI OK；`wails build` 成功（build/bin/gaea.exe 38MB）

## 2.13 聊天只选角色：聊天内角色选择器（✅ 已完成，2026-08-07）

- 确认产品方向：聊天板块只做「选角色」，其余角色管理/编辑/生成全部在角色库
- 新增 `PersonaPicker` 聊天内选择器：模式条「切换角色」按钮 + 人格空状态「选择角色」入口；列表来自角色库可聊天角色（`CharacterList(chatOnly=true)`），带搜索（名称/标签）、头像/类型标签、当前标记；选中即 `handleSwitchPersonality`（清内存会话→持久化全局人格→切换当前话题模式）；底部「去角色库管理角色」
- 清理聊天内残留管理入口文案：「虚拟助手管理」→「角色库管理」，空状态改为「选择角色」+「去角色库管理角色」
- 审计结论（供下一步决策）：人格注入为结构化分块（口吻指南 + 角色状态 Tier A + 心理状态 + Tier B 记忆块 + 运行时上下文）；记忆按 `whisper_<角色ID>` 隔离状态/历史，但**事实/情节/知识图谱恢复时未按会话过滤**（`LoadFactsFromDB` 全量加载），存在跨角色事实串扰——待确认语义后修复
- 验收：`npm run build`、`scripts/ci.ps1` CI OK、`wails build` 成功（build/bin/gaea.exe 38MB）

## 2.16 发布 2.2.0「统一角色库」（✅ 已完成，2026-08-07）

- 版本号：wails.json / versioninfo.rc / CHANGELOG / README → 2.2.0
- 构建：wails build 成功（build/bin/gaea.exe 38MB），已复制到桌面 + releases/gaea-v2.2.0.exe
- 校验和：releases/SHA256SUMS-v2.2.0.txt（89715c6e…）
- 发布文档：releases/v2.2.0.md（2.10–2.15 六轮迭代摘要：统一角色库 / 小说单向抽卡 / 聊天选角色 / 记忆隔离 / 状态归集 / 取消轻语）
- 备份：scripts/backup.ps1 已运行（whisper_data/配置 → backups/20260807-150559）
- 全量验收：scripts/ci.ps1 CI OK，提交后打 tag v2.2.0 并推送 origin

## 2.14 角色记忆隔离（方案 A：每个角色只恢复/保存自己的记忆，✅ 已完成，2026-08-07）

- 用户确认方案 A：事实/情节/知识图谱全部按角色会话隔离，不做跨角色共享
- 恢复侧：新增 `LoadFactsFromDBForSession` / `LoadEpisodesFromDBForSession`（按 `source_session_id` 过滤）；知识图谱表无会话列，按三元组 `source_fact_ids` 命中本会话事实 ID 过滤归属；每个角色只灌入自己的记忆
- 持久化侧：事实维持按 ID 合并；**情节从全量替换改为按会话合并**（本会话内存版替换，其他会话保留 DB 版）；**知识图谱从全量替换改为按归属合并**（命中本会话事实的三元组以内存为准，其余保留），杜绝 A 角色写回时冲掉 B 角色的记忆
- 无来源事实的三元组视为全局遗留：不注入任何角色、也不在写回时删除（保守保留）
- 更新既有 `TestWhisperMemoryPersistRoundTrip` 为隔离语义；新增 `TestWhisperMemoryRestore_IsolatedBySession` / `TestWhisperPersist_PreservesOtherSessions`
- 验收：`go test ./...`、`scripts/ci.ps1` CI OK、`wails build` 成功（build/bin/gaea.exe 38MB）

## 2.15 状态/记忆/追踪归集角色库 + 取消「轻语」称谓（✅ 已完成，2026-08-07）

- 聊天面板只留对话：移除右侧记忆抽屉（状态/记忆/追踪三页）、记忆管理弹窗、人格状态条上的情绪/信任/轮次/裂痕展示；保留头像/名字与情绪氛围色（视觉反馈，非状态面板）
- 新增角色库「记忆」入口（可聊天角色卡片）：`CharacterMemoryModal` 三个 Tab——状态（WhisperEmotionPanel + 欲望面板）、记忆（WhisperMemoryList 只读分组 + 管理弹窗）、追踪（WhisperTracePanel）
- 追踪按角色持久化：`turn_traces` 新增 `session_id` 列（Schema V12 迁移），每轮对话写入归属会话；新增 `WhisperGetTraces` 绑定按会话读取；`WhisperGetState`/`WhisperGetFacts` 改为无会话时自动恢复（角色库查看任意角色状态不再要求先聊过）
- 取消「轻语」称谓（统称聊天）：模式条「轻语·X」→「角色·X」；设置页/语音设置/模型中心/记忆中枢/首页启动器等全部改「聊天」或「聊天记忆」；首页删除重复的「轻语」启动卡片；删除 chatNav 死信号
- 验收：`npm run build`、`scripts/ci.ps1` CI OK、`wails build` 成功（build/bin/gaea.exe 38MB）

## 2.11 去除成人限制（私人非商用，✅ 已完成，2026-08-07）

- 用户声明软件私人使用、不商业化 → 移除成人内容门禁，成人内容默认开启
- 后端：`NewOrchestrator` 默认 `AdultMode: true`（成人状态机/表达策略/记忆隐私分级随会话创建即生效）；`WhisperSetAdultMode` 保留方法但强制 true（兼容旧调用）；成人内容引擎内的安全阀门（hard stop / 拒绝词 / 关系阶段门控 / 未成年红线）原样保留
- 前端：设置面板删除「我确认已年满 18 岁」复选与「成人模式」开关（`ageConfirmed18` / `gaea_whisper_adult_mode` 一并移除）；人格下拉与角色库卡片去除 18+ 标记；ChatPage 删除无用 adultMode state；`WhisperRolePanel` 删除 adultMode 死参数
- 验收：`npm run build`、`scripts/ci.ps1` CI OK、`wails build` 成功
