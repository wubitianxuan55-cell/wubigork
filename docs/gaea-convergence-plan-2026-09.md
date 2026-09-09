# gaea 收敛计划（2026-09-08 立项）

> **状态：🔄 活跃权威**（收敛期执行账 = 本文件 + `.gaea/todos.md`；全域路线图权威仍为
> `gaea-nextgen-roadmap-2026.md`，本文件是其收敛期叠加层，冲突时以「不加新板块」为先）。
> **2026-09-08 起分工**：本文件=治理层（节奏/拍板/不做清单）；瘦身执行层（原 §3 架构止血、
> §5.2 模块可藏）已全面化为 `gaea-slim-masterplan-2026-09.md`（七轨道六阶段权威）。
>
> **立项依据（2026-09-08 全量体检）**：Go 3921 测试函数全量绿；vitest 2677 例/304 文件
> （满并发双套件下 32 例超时假红、隔离复跑全绿）；图谱 30,936 节点/132,488 边，`office`
> 272 入/70 出枢纽化、`whisper` 143 入/96 出；44 天 937 commits/324 发布说明；无
> package-lock、无 LICENSE。结论：工程纪律是最大资产，「单人 × 高速 × 广度」是最大风险，
> 收敛期目标 = 压掉其中至少两个变量。

## 0. 拍板项（用户拍板前不动的两件事）

**已拍板（2026-09-08，用户裁定）**：

1. **性质路线 = A（终极个人工具）**——所有决策按自用标准；B（产品化）降为期权，
   不作承诺、不设季度门槛义务；§5 产品期权仅在用户重新拍板后激活。
2. **LICENSE = 私有 All Rights Reserved**——根目录 LICENSE 已落；若未来开放部分模块
   源代码，另行发布开源许可证覆盖对应模块并与本文件区隔（LICENSE 尾注已写明）。

## 1. W1 卫生刀（2026-09-08 启动，目标一周清完）

| # | 刀 | 验收 | 状态 |
|---|---|---|---|
| 1.1 | `frontend/package-lock.json` 入库（.gitignore 放行）+ CI `npm install`→`npm ci` | `npm ci --dry-run` 通过；CI 与本地装出同一棵依赖树 | 🔄 本轮 |
| 1.2 | vitest 抗抖：`testTimeout` 提至 15s + CI 前端 job 加一次 flaky retry（对齐 Go job 形态） | 双套件满并发同机重跑全量无超时假红 | 🔄 本轮 |
| 1.3 | DeliverablesPanel `{page:'office'}` 过期 id 潜伏 bug（v4.121 遗留）复现钉死 | 失败用例先行 + 根因修复 + 回归锁 | 🔄 本轮 |
| 1.4 | WebView2 壳内残留面扫描：`input[type=file]`（characterlib/imagegen×2/NovelSetting/Schedule×4）+ `a[download]`（export.ts/useImageGenHistory/ImageGenPage）+ print 路径 | 逐点位风险表 + 修复刀序（审计先行，修复另立刀，沿用 v4.162 的 PickFiles/ReadFileB64/saveExportBlob 模式） | 🔄 本轮 |

## 2. W2 知识外化刀（对冲 bus factor=1）

1. **30 分钟上手文档**：面向「懂 Go/React 第一次开仓」的读者，讲透绑定门面委托规约
   （v4.162 坑）、gen_bindings/drift、发布流程、测试口径。验收 = 陌生 agent 只读该文档
   即可正确提交一个带新绑定的小改动。✅ docs/gaea-getting-started-30min-2026-09.md（2026-09-09）
2. **progress.md 瘦身**：244KB 归档为年表，活跃文件只留开放项（todos.md 形态）。
   ✅ 归档至 docs/archive/progress-history-2026-09.md（2026-09-09）
3. **个人路径清洗**：本机绝对路径（盘符+工作目录）移出活跃设计文档，样例说明改指
   fixtures/占位。✅ 4 文档 5 处清洗（2026-09-09；docs/archive/ 历史证据不清洗）

## 3. W3-4 架构止血刀（不求重构，只求止血）

1. **office 枢纽解耦**：先 trace `whisper→office`（96 调用）与 `modelengine→office`
   （62 调用）逐条归类（文件工具/会话记忆/历史误用），把事实上的公共内核（journal/
   evidence/文件读写）抽为中立包，三个板块平级依赖。此刀优先——它降低后续每一刀的碰撞面。
2. **巨文件四件套拆分**（每刀独立提交、绑定面零变更、drift PASS、vitest 全绿）：
   - `frontend/src/gaea/App.tsx`（85KB）：按板块拆容器，App 只留壳+装配；
   - `frontend/src/gaea/lib/bridge.ts`（85KB）：按 bindingNames 所属域自动切分，index 聚合，类型不变；
   - `frontend/src/schedule/GanttView.tsx`（65KB）：渲染/交互/刻度三分（aoaRuler 先例）；
   - `internal/config/config.go`（59KB）：按域拆文件（同包多文件零迁移成本）。

## 4. 节奏改革（即刻生效，先于一切技术刀）

1. **周度 release train**：日常 commit 照常，版本号每周打一次；发布说明周度合并。
2. **拍板池 14 天过期**：挂超过 14 天的「待拍板/若做」项自动降级为不做，想做再激活。
3. **每 4 周一个零功能周**：只还欠账、走查、写文档；允许不发新版本。

## 5. 产品期权（B 路线拍板后才做，顺序不可换）

1. 造价刀路池打穿（条目匹配键→组价分步→五算快照→清单级测评集），**先于**任何新模块；
2. 模块可藏：小说/轻语/绘梦/微信折叠进「实验室」（feature flag，藏≠删）；
3. 10 分钟上手标准：解压→AI 改 Excel 并出 diff 确认卡 ≤10 分钟 ≤3 次点击。

## 6. 不做清单（收敛期红线）

macOS/Linux 适配 · MCP 生态大全 · 重写/换框架 · 一切新板块（含 DSH 阶段四/五）·
为想象中的用户写代码。

## 7. 度量（周报四行）

欠账净减数 · 巨文件数（>50KB 源文件）· 真机走查池存量 · CI 假红次数。
