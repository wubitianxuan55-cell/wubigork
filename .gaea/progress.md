# 任务进度

> 最后更新: 2026-08-07 13:00:00

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
