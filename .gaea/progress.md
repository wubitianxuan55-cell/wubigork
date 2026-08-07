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

| 状态 | 任务 |
|------|------|
| ⬜ | Write tests |
| ✅ | Add parser |
