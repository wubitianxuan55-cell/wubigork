# 任务进度

> 最后更新: 2026-08-07 10:15:38

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
