# gaea 进度板块 · 复刻斑马进度前的存量评估（舍/改判定书）

> **结论先行**：**不要重写。** 现有代码 14.2K 行源 + 12.3K 行测试、537 个 vitest 用例、
> 137 个 Go 测试函数，且**双代号引擎的教材口径已经和斑马同源**（唯一起终点、链式免虚箭、
> FS/SS/FF/SF 补虚箭、虚箭去重、事件正逆推、关键线路含关键虚箭、波形线=自由时差、过桥法）。
>
> 判定分布（按文件，共 55 个前端源文件 + 14 个 Go 源文件）：
>
> | 判定 | 文件数 | 说明 |
> |---|---|---|
> | **保留**（不动或只加字段透传） | ~40 | 内核、分析面板、导出互通、交互工具 |
> | **修改**（增量，不改结构） | ~12 | types / store / cpm / ganttCols / ganttGroup / TaskInspector / 绑定面 |
> | **改造**（保留渲染与算法，换数据源或参数化） | 3 | `aoa.ts` / `AoaView.tsx` / `ganttCols.ts` |
> | **新增**（斑马有 gaea 无） | 6 组 | 检查器 / 检视+前锋线 / 预警 / 视图快照 / 图面参数 / 期间计划 |
> | **整块舍弃** | **0** | 只有 3 处"策略级"舍弃（见 §5） |
>
> 依据：`docs/gaea-schedule-zpert8-network-re-20261114.md`（斑马网络图逆向，15 章 + 附录）
> 与 `gaea-schedule-zpert8-distill-20261114.md`（全产品面）。

---

## 1 存量盘点（实测）

### 1.1 规模

| 面 | 文件 | LOC | 备注 |
|---|---|---|---|
| 前端 `frontend/src/schedule` | 55 源 + 46 测试 | **10,039 / 7,566** | 14 个 `.tsx` 视图/面板 |
| Go `internal/schedule` | 14 源 + 17 测试 | **4,178 / 4,690** | CPM/日历/成本/基线/MPP/XLSX/Ops/索引 |
| App 绑定 `internal/app/gaea_schedule*` | 3 源 + 2 测试 | **890 / 916** | 11 个导出方法 |
| **合计** | | **≈15.1K 源 / 13.2K 测试** | 537 vitest 用例 + 137 Go 测试函数 |

### 1.2 已有能力（按斑马的对照面）

| 斑马能力 | gaea 现状 | 位置 |
|---|---|---|
| FS/SS/FF/SF + ±lag | ✅ | `types.ts:Link` / `cpm.go:forwardBound/backwardBound` |
| 双工期（工日/日历天） | ✅（gaea 用 `durationUnit`，斑马用"含假期"） | `types.ts:DurationUnit` |
| 总时差/自由时差 | ✅ | `cpm.ts` / `cpm.go:freeFloatPart` |
| 关键工作/关键线路 | ✅ | `cpm.ts` + `CriticalChain()` |
| 基线（原计划对比） | ✅ **3 槽 FIFO（强于斑马单"原计划"）** | `baseline.ts` / `baseline.go` |
| 资源/成本/EVM | ✅ | `cost.ts` / `evm.ts` / `ResourcePanel` / `EvmView` |
| DCMA 14 点质量体检 | ✅ **斑马没有** | `dcma.ts` + `dcma-thresholds.json` |
| 蒙特卡洛工期带 | ✅ **斑马没有** | `monte.ts` / `MonteView` |
| 双代号（AOA）引擎 | ✅ **教材口径 + 虚箭 + i<j 编号 + 真时标 x=ES** | `aoa.ts`（文件头注释即口径书） |
| 关系线正交/通道/过桥（跨线避让） | ✅ | `aoaLayout.ts:edgeSegs/assignChannels/findBridgeArcs` |
| 波形线 = 自由时差 | ✅ | `AoaView.tsx` 头注释 G4 |
| 单代号（PDM）视图 | ✅ | `PdmView.tsx` |
| MPP 导入 / MSPDI / XLSX | ✅（MPP **只读**） | `mpp.go` / `mspdi.ts` / `xlsx.go` |
| 变更 diff + 确认卡 + 回滚 | ✅ | `applyDiff.ts` / `opsSim.ts` / `ScheduleDiffCard` |
| 检视时间（多期）/前锋线/冻结 | ❌ | — |
| 预警状态机（档位/红绿灯/两算法） | ❌ | — |
| 排网方式 ASAP/ALAP + 要求时间约束 | ❌（只有 auto/manual） | `types.ts:TaskMode` |
| 视图快照（列+分组+筛选+样式） | ❌（只有 `view` 枚举切页） | `store.ts:ScheduleView` |
| 分组维度 14 种 | 部分（3 种：critical/mode/milestone） | `ganttGroup.ts` |
| 列可配置/列记忆 | ❌（静态 `GANTT_COLS`） | `ganttCols.ts` |
| 布图压实（页数窗/紧凑度/按纸张） | ❌ | — |
| 期间计划/工程量/定额库 | ❌ | — |

> 一句话：**gaea 的"计划编制 + 质量体检 + 概率分析 + 变更治理"已经比斑马深；
> 空的是"执行期治理（检视/前锋线/预警）"和"视图体系"两层** —— 与 D0 结论一致。

---

## 2 耦合度量（决定"改得动吗"）

前端模块被引用次数（in-degree，`from './x'` 计数）：

| 模块 | importers | 含义 |
|---|---|---|
| `types.ts` | **79** | 全板块的类型脊柱。**加可选字段是安全的**（无解构依赖），改既有字段名才是灾难 |
| `cpm.ts` | **30** | 计算入口被所有视图/面板消费 |
| `store.ts` | **21** | 中心状态；动它等于动 21 个消费方 |
| `calendar.ts` | 15 | 时间口径 |
| `aoa.ts` | 9 | 双代号转换（视图 + 导出 + 布局） |
| `ganttCols.ts` | 7 | 列定义 |
| `aoaLayout.ts` | 6 | 布局/走线 |
| `baseline.ts` / `cost.ts` / `deadline.ts` | 6 / 5 / 5 | 分析面 |

跨目录消费方（schedule 之外）：`pages/SchedulePage.tsx`、`pages/SchedulePage.test.tsx`、
`pages/schedule/{CalendarEditor,ExportDialog,ProjectsManagePanel}.tsx`。

**判读**：
- 类型层 79 依赖 → **只能增量演进**（可选字段 + 缺省值），任何"改名字/改形状"都要付 79 处回归；
- `store.ts` 21 依赖 + 732 行 + 自带 history/sync 测试（`store.history.test.ts` 98 行 / `store.purity.test.ts` 121 行 / `store.sync.test.ts` 386 行）
  → 这是**唯一的高风险修改点**，必须"加动作、不改既有动作语义"；
- `aoa.ts` 9 依赖但**纯函数**、有 291 行单测 → 改造风险可控，是性价比最高的一刀。

---

## 3 逐模块判定

### 3.1 内核（数学/时间/成本/分析）—— **保留，两处扩展**

| 文件 | LOC | 判定 | 理由 / 要改什么 |
|---|---|---|---|
| `cpm.ts` / `cpm.go` | 247 / 416 | **修改** | 加 ALAP（越晚越好）与 `askStart/askFinish` 约束夹取；加"自由口径最晚时间"（斑马 `latest_*_by_free`，见 RE §4.2）。**双份镜像必须同步**（`types.go` 头注释即纪律） |
| `calendar.ts` / `calendar.go` | 138 / 223 | 保留 | 只需加 `firstDayCounts`（斑马 `首日是否参与计算-是/否`），默认沿用现口径零迁移 |
| `cost.ts` / `cost.go` | 93 / 117 | 保留 | 斑马四口径单价（计划/实际/合同/产值）可后置按需加 |
| `baseline.ts` / `baseline.go` | 138 / 159 | 保留 | 已强于斑马（3 槽 vs 单"原计划"） |
| `deadline.ts` / `deadline.go` | 49 / 52 | 保留 | 倒排校核已有 |
| `analysis.go` / `analysis.ts`(无) | 148 | 保留 | 叙述生成 |
| `monte.ts` / `evm.ts` / `dcma.ts` | 179 / 110 / 342 | 保留 | 斑马没有，是 gaea 的差异化资产；`dcma.ts` 的 `dataDate` 参数可直接复用给检视 |

### 3.2 数据模型与持久化 —— **修改（增量），1 处高风险**

| 文件 | LOC | 判定 | 要改什么 |
|---|---|---|---|
| `types.ts` | 195 | **修改** | 追加**全可选**字段：`TReview[] reviews`、`Project.aoa?`（点-箭持久）、`Project.networkStyle?`、`Project.views?`、`Task.askStart?/askFinish?/scheduleType?/layer?`。旧 `.gsched.json` 零迁移 |
| `store.ts` | 732 | **修改（高风险）** | 加 `addReview/removeReview/straightenForwardLine/freeze`、`setView` 快照、`setNetworkStyle`；**不改既有动作语义**，并在 `store.sync.test.ts` 补"新字段往返不丢"（已踩过 `baselines` 被静默抹掉的坑，types.go 有记录） |
| `api.ts` | 144 | 修改 | 新绑定面 3–5 个（见 3.5） |
| `project.go` | 290 | 修改 | `Validate` 增新区块校验（fail-closed，沿用现风格） |
| `ops.go` | 642 | 修改 | 增 `setReview/clearReview/setNetworkStyle/setView` 四类 op（有 `ops_golden_test.go` 保证不放宽既有口径） |
| `index.go` | 325 | 保留 | 工程索引与当前指针不受影响 |
| `applyDiff.ts` / `opsSim.ts` | 303 / 405 | 保留 | diff/回滚机制已够；只加新 op 类型 |
| `gschedSummary.ts` | 90 | 修改 | 摘要带上检视次数/预警计数（给 agent 用） |

### 3.3 三视图与渲染 —— **AOA 一簇"改造"，其余小改**

| 文件 | LOC | 判定 | 理由（关键） |
|---|---|---|---|
| `aoa.ts` | 419 | **改造** | 保留 `buildAoa` 的教材口径与 `anchor` 机制；增两条：① **有持久 dots 时优先用其 id/num，不再每次重建事件身份**（现口径下拓扑一改，事件 id 与 pin 全漂移）；② 坐标映射参数化以支持 5 种模式（斑马：时标逻辑图/时标网络图/逻辑网络图/单代号网络图/横道图）。约 +150 行、改约 40 行 |
| `aoaLayout.ts` | 351 | 保留为主 | `edgeSegs/assignChannels/findBridgeArcs/snapPt/applyPins` 全是资产（斑马的"跨线避让/过桥"GA 已有）。只把 `AOA_PINS_MAX=5000` 的全量落盘降级为**差量覆盖** |
| `AoaView.tsx` | 400 | **改造** | 渲染/拖拽/波形线/图例全保留；增模式切换与图面参数（`dot_style/arrow_style/name_pos/head_height/layer_height` 等，见 RE §4.2） |
| `PdmView.tsx` | 300 | 小改 | 加"时标/逻辑"两模式（斑马 `单代号网络图` 与 `时标逻辑图`） |
| `GanttView.tsx` | 527 | 修改 | 加"横道网络图"叠加（关系线 + 时标刻度）；列从静态改配置 |
| `ganttCols.ts` | 84 | **改造** | `GANTT_COLS` 是硬编码常量数组（含宽度），无法支撑"插入列/列设置/视图记忆"。改成"列注册表 + 每视图列集" |
| `ganttGroup.ts` | 254 | 修改 | 分组维度 3（critical/mode/milestone）→ 斑马 14（工期/总时差/负责人/里程碑/关键/工作分类/工作部位/执行状态/资源名称/前置数/后置数/自定义） |
| `ganttFilter.ts` / `ganttLinks.ts` / `layout.ts` / `drag.ts` / `wheelZoom.ts` / `aoaRuler.ts` / `pathDriver.ts` | 62/48/120/63/43/50/143 | 保留 | 通用交互与几何工具，斑马对应能力都有且不冲突 |

### 3.4 面板与交互 —— **保留，TaskInspector 一改**

| 文件 | LOC | 判定 | 备注 |
|---|---|---|---|
| `TaskInspector.tsx` | 240 | 修改 | 加"要求开始/要求完成、执行状态、预警状态、检视回填"四组字段 |
| `ResourcePanel` / `BaselinesPanel` / `ScheduleDiffCard` / `DcmaView` / `EvmView` / `MonteView` / `UsageView` / `WhatIfPanel` / `TemplatesPanel` / `ChatPane` | 345/245/221/190/114/92/192/97/87/130 | 保留 | 与斑马口径不冲突；`ChatPane` 是 agent 入口，正好承载"AI 改计划" |
| `advice.ts` / `whatif.ts` / `templates.ts` / `chatPrefs.ts` | 126/78/109/101 | 保留 | |

### 3.5 导出与互通 —— **保留 + 补一个缺口**

| 文件 | LOC | 判定 | 备注 |
|---|---|---|---|
| `ganttExport.ts` / `networkExport.ts` | 457 / 376 | 保留 | 斑马是 QPainter+libxl 自绘；gaea 已有 png/pdf/xlsx 全套 |
| `mspdi.ts` / `exportArtifact.ts` / `exportMeta.ts` / `imgPdf.ts` | 244/107/76/78 | 保留 | MSPDI 双向已有 |
| `mpp.go` / `mpp_rows.go` / `mpp_layout.go` | 560/279/141 | 保留 | **只读解析**；斑马有"导出 MPP"，gaea 缺 → **新增**（`mpp_export.go`，不是改 mpp.go） |
| `xlsx.go` | 617 | 保留 | 双向已有 |
| `gaea_schedule*.go`（绑定面） | 890 | 修改 | 新增 `GaeaScheduleAddReview` / `StraightenForwardLine` / `SetNetworkStyle` / `SaveView` 等 4–6 个方法 |

### 3.6 新增（斑马有、gaea 完全没有）

| 新文件 | 对应斑马 | 工作量 |
|---|---|---|
| `checker.ts` (+`check.go` 镜像) | 检查器 C1–C15（RE §8.1） | 0.5 刀 |
| `reviews.ts` + 前锋线几何 | 检视时间/拉直/冻结/对比表（RE §5.1–5.2） | 1.5 刀 |
| `warn.ts` | 预警状态机：五档 + 红绿灯 + 两算法（RE §5.3） | 0.5 刀 |
| `views.ts` | 视图 = 列+分组+筛选+样式快照（RE §6.1） | 0.5 刀 |
| `networkStyle.ts` | 图面 60+ 参数（RE §4） | 0.5 刀 |
| 期间计划 / 工程量 / 定额库 | 期间窗口 + 当期量（RE §5.5 / §7） | 2 刀+（按客户触发） |

---

## 4 四个"硬冲突"（决定改动方式，不是要不要改）

### 4.1 双代号的"点身份"：派生 anchor vs 持久 dots

- **gaea**：`buildAoa()` 每次从 tasks+links 重建事件，`AoaNode.id` 与 `num` 是**派生结果**，
  只有 `anchor`（成员键字典序最小者）跨拓扑变更稳定，pins 靠它匹配（`aoaLayout.test.ts` 钉死）。
- **斑马**：`dots` **持久化**，带递增 id 与数组序号（`Dot[+0]=id`、`Dot[+4]=序号`，RE §16.2），
  建关系时还会**自动建点/合并重复点**（`newDot`/`mergeDot`，RE §15.5–15.6）。
- **冲突后果**：拓扑一改（插入工作/改层级），gaea 的事件编号与 pin 位置会变，斑马不会。
- **处置**：`Project.aoa?: {dots, arrows, beginDotId}` 作为**可选权威源**；存在时 `buildAoa` 用它，
  不存在时按现口径派生。**不是重写 `buildAoa`，是给它一个数据源分支**。

### 4.2 显示模式：1 种 vs 5 种

斑马 5 模式（RE §2.1，内存 `char[5][16]` 实证）：`时标逻辑图 / 时标网络图 / 逻辑网络图 / 单代号网络图 / 横道图`，
且**每种模式有自己的一块几何基准**（RE §15.7）。gaea 现在：`ScheduleView` 枚举 7 个**页面**（gantt/pdm/aoa/usage/dcma/monte/evm），
AOA 内部只有"自动时标/手动 pins"两态。
- **处置**：`ScheduleView` 不动（页面级）；在 AOA/PDM 内部加 `renderMode: 'timescaled'|'logic'|'pdmscaled'|'ganttNetwork'`，
  `buildAoa(opts)` 按模式切坐标映射。**加参数，不改枚举**。

### 4.3 列与分组是静态常量

`GANTT_COLS` 是常量数组（含宽度），`ganttGroup` 只吃 3 个字段 → 撑不起"视图快照/14 分组/插入列"。
- **处置**：列改"注册表 + 视图列集"，分组改"字段插件表"。**这是唯一需要动结构的改造**（84 + 254 行），
  但两个文件都是纯函数、单测齐全（`ganttGroup.test.ts` 10.4KB / `ganttCols.test.ts` 5.2KB），可控。

### 4.4 CPM 双份镜像

前端 `cpm.ts` 与后端 `cpm.go` 是**手写镜像**（`types.go` 头注释："修改口径必须两侧同步，前端测试与 Go 测试互为镜像"）。
- **处置**：本条不改，但要**在加 ALAP/约束时严格执行双向同步 + 两侧测试**（否则 agent 算的和界面显示的分叉）。
  这是"修改 vs 重写"里最容易被忽视的成本项。

---

## 5 真正的"舍弃"清单（只有 3 处，且都是策略级）

| # | 舍弃什么 | 位置 | 为什么 | 怎么处置 |
|---|---|---|---|---|
| 1 | "每次重建事件 id、以字典序最小 anchor 作身份"的**默认口径** | `aoa.ts`（约 30 行） | 斑马 dots 持久、编号稳定；继续这样 pin 与编号必然漂移 | 保留 anchor 作**兜底**，有持久 dots 时优先用其 id |
| 2 | `AOA_PINS_MAX = 5000` 的**全量坐标落盘**默认 | `aoaLayout.ts`（约 10 行） | 斑马根本不存坐标（只存层号+参数，RE §16.4），全量 pins 会让计划文件膨胀且与"参数现算"冲突 | 降级为"仅存手动差量"，序列化边界加 pruning（已有 `prunePins` 可复用） |
| 3 | `GANTT_COLS` 作为**唯一真源**的静态列定义 | `ganttCols.ts`（84 行重写） | 无法支撑插入列/列设置/视图记忆 | 改成注册表 + 视图列集；**语义（列宽/可隐藏/固定列）全部保留** |

> 除此之外**没有任何文件需要整块删掉**。`aoa.ts`/`AoaView.tsx`/`ganttCols.ts` 三个是"改造"，
> 其余全是"保留 + 增量"。**"弃掉重写"在当前存量下是不划算的**：12.3K 行测试与 537 用例
> 是重写的隐藏成本，而这些用例覆盖的正是斑马也没做错的数学口径。

---

## 6 迁移顺序（与斑马复刻刀单对齐）

| 步 | 动作 | 触达文件 | 风险 |
|---|---|---|---|
| **0** | 本判定书定稿 | — | — |
| **1** | ✅ **已完成（2026-11-14）**：`types.ts`/`types.go` 增量四块（`reviews`/`aoa`/`networkStyle`/`views`）+ 归一与 fail-closed 校验 + 往返测试。**能存能读，未接 UI、CPM 不消费** | `types.ts`(+165) / `blocks.ts`(新) / `store.ts`(+7) / `blocks.test.ts`(新) / `types.go`(+136) / `blocks.go`(新) / `blocks_test.go`(新) / `project.go`(+5) | 中（store 21 依赖）→ 实测零回归 |
| **2** | ✅ **已完成（2026-11-14）**：N1 点-箭持久化接进 `buildAoa`（消费侧）+ `aoaGraphFromBuild` 固化（生产侧，首次手动布点触发）+ pins 降级为手动差量（`pinsToPersist`） | `aoa.ts` / `aoaLayout.ts` / `AoaView.tsx` / `store.ts` / `SchedulePage.tsx` / `ScheduleTool.tsx` | 中低 → 实测 669 例全绿 |
| **3** | ✅ **已完成（2026-11-14，仅前端）**：`checker.ts` 18 条规则 + `TaskInspector` 抽屉段 + 两处调用点接入。**Go 侧刻意不做 check.go**（避免第三套口径，理由见 §11.1） | `checker.ts`(新) / `checker.test.ts`(新) / `TaskInspector.tsx` / `SchedulePage.tsx` / `ScheduleTool.tsx` | 低 → 实测 690 例全绿 |
| **4** | ✅ **已完成（2026-11-14，模型+面板）**：`reviews.ts`（加权完成量/三态/期际对比/拉直冻结/当期计划 vs 实际）+ `ReviewPanel`（打点·列表·对比·拉直·删除）+ store 三动作。**图上画前锋线线体留到下一步**（需动 GanttView SVG） | `reviews.ts`(新) / `ReviewPanel.tsx`(新) / `store.ts` / `SchedulePage.tsx` | 中 → 实测 710 例全绿 |
| **5** | 预警状态机 | warn(新)/TaskInspector/导出 | 低 |
| **6** | 视图快照 + 列注册表 + 14 分组 | ganttCols(改造)/ganttGroup/views(新) | 高（唯一结构改造） |
| **7** | 模式矩阵（5 模式） | aoa/AoaView/PdmView/GanttView | 中 |
| **8** | MPP 导出 / 期间计划 / 定额库 | mpp_export(新)/… | 低–中 |

**纪律**（沿用本仓既有）：每步先写失败测试；CPM 改动必须双份镜像同步；`store` 只加动作不改语义；
每步跑定向测试，收口跑一次全量（vitest 537 例 + Go 137 函数）。

---

## 7 判定依据（可复核的硬证据）

| 结论 | 证据 |
|---|---|
| 规模与测试面 | §1.1 实测（LOC/测试函数/用例计数命令见附录） |
| 双代号教材口径已同源 | `aoa.ts` 文件头注释列出的 6 条转换规则 + `AoaView.tsx` 头注释（虚箭/波形线/过桥/JGJ-T-121） |
| 斑马 dot 持久化 | RE §15.5–15.6（`newDot`/`mergeDot`/`splice`）、§16.2（Dot 结构） |
| 斑马 5 模式与每模式几何块 | RE §2.1（`char[5][16]` 表）、§15.7（`0x1b40/0x19a8` 分支） |
| 斑马不存节点坐标 | RE §3.2（`layer`/`pert_height` 落盘）、§16.4 |
| 斑马列/分组/视图 | RE §6.1（10 内置视图）、§6.2（14 分组维度）、§14（219 键） |
| 检查器规则集 | RE §8.1（C1–C15） |
| 检视/前锋线/预警 | RE §5.1–5.3 |

## 8 附录：复核命令

```powershell
# 规模
Get-ChildItem frontend\src\schedule -File -Filter *.ts* | Where-Object {$_.Name -notmatch '\.test\.'} |
  ForEach-Object { [pscustomobject]@{ f=$_.Name; loc=(Get-Content $_.FullName | Measure-Object -Line).Lines } } |
  Sort-Object loc -Descending
# 用例数
(Get-ChildItem frontend\src\schedule -File -Filter '*.test.ts*' | Select-String "^\s*(it|test)\(").Count
(Get-ChildItem internal\schedule -File -Filter '*_test.go' | Select-String '^func Test').Count
# 引用度
Select-String -Path frontend\src\schedule\*.ts* -Pattern "from './types'" | Measure-Object
```



---

## 9 第 1 步落地记录（承载位，2026-11-14）

**范围**：只做"能存能读不丢"，不接 UI、不进 CPM、不改任何既有语义。

| 文件 | 变更 | 行数 |
|---|---|---|
| `frontend/src/schedule/types.ts` | 新增 `SchedReview(Actual)` / `SchedAoaDot` / `SchedAoaArrow` / `SchedAoaGraph` / `SchedNetworkStyle` / `SchedRenderMode` / `SchedView(Sort)`，`SchedProject` 增四个**全可选**字段 | +165 |
| `frontend/src/schedule/blocks.ts`（新） | 四个 normalize 纯函数 + 上限常量 + `isISODate`；风格对齐 `normalizeAoaLayout` | 320 |
| `frontend/src/schedule/store.ts` | `normalizeProject` 末尾接四行归一（`...p` 已保证未知键透传，这里补**形状归一**） | +7 |
| `internal/schedule/types.go` | 同构 Go 结构体 + `Project` 四个 `omitempty` 字段（`Reviews/Aoa/NetworkStyle/Views`） | +136 |
| `internal/schedule/blocks.go`（新） | `validateBlocks` + 四个子校验（枚举白名单/区间/唯一性/引用完整性/规模上限） | 300 |
| `internal/schedule/project.go` | `Validate` 末尾接 `validateBlocks`（fail-closed 闸） | +5 |

**契约要点**（后续刀单必须遵守）：

1. **缺省=旧行为**：四块全 `undefined`/`nil` 时 `.gsched.json` **一个键都不多写**（有专项用例断言）；旧文件零迁移。
2. **前后端分工固定**：前端 normalize「逐条容错丢弃，保证板块能打开」；Go `Validate`「发现非法就拒绝保存并给可读原因」。两侧样例集互为镜像（`blocks.test.ts` ↔ `blocks_test.go`）。
3. **布尔三态**：Go 侧 `Stagger`/`HideUnimportantLink` 用 `*bool`（显式 `false` 与"未设置"语义不同），前端用 `boolean | undefined`。
4. **时间口径**：`Review.at` 与 `CpmResult` 同为「工作日序号（相对开工日）」；`date` 只作展示与人工校核。
5. **时距闸门**：箭线 `lag` 沿用斑马 ±9999（逆向报告 §3.1）。

**验证（本轮实跑）**：

| 门 | 命令 | 结果 |
|---|---|---|
| Go 定向 | `go test ./internal/schedule/ -count=1` | ok（0.64s，含新增 5 个用例组） |
| Go 上游 | `go test ./internal/app/ -run 'Schedule|BindingsCompleteness'` | ok（0.77s） |
| 前端定向 | `vitest run src/schedule/blocks.test.ts` | 20 passed |
| 前端回归 | `vitest run src/schedule` | **49 文件 / 639 例全绿**（60.6s） |
| 类型 | `tsc -b` | 0 错 |
| Lint | `eslint` 改动四文件 | 0 错 |

**下一步**（判定书 §6 第 2 步）：N1 点-箭持久化接进 `buildAoa` —— 走"有 `Project.aoa` 就用它、没有就按现口径派生"的分支，
并把 `aoaLayout` 的 pins 从"主存"降为"手动差量"。


---

## 10 第 2 步落地记录（N1 点-箭持久化，2026-11-14）

**范围**：把 `Project.aoa` 真正接进双代号渲染链，让**编号跨拓扑稳定**（斑马 `dots` 口径），
并把手动布点从"坐标快照"降级为"手动差量"。

### 10.1 契约（唯一口径，两侧都靠它）

- **`SchedAoaDot.id` 就是锚点键**：`S` / `T` / `start:<taskId>` / `end:<taskId>`。
  这条契约让"持久图元"与既有的 `anchor`（pin 的匹配键）天然对齐——不需要第二套身份映射。
- 命中持久点的节点：用其 `num`/`layer`，并带上 `dotId`；未命中者按原顺序补号、**跳过已占用编号**（编号必须唯一，是"箭线 i<j"显示的前提）。
- 一个都没命中（脏文件/全删过）→ 退回派生编号（`numSource='derived'`），不让坏数据扰动现有图。

### 10.2 消费侧（`buildAoa`）

```ts
buildAoa(tasks, links, { planFinish, persisted?: SchedAoaGraph })
  → nodes[i] { num, dotId?, layer?, anchor, ... }   // num 取自持久点
  → edges[i] { ..., arrowId? }                       // 两端都有 dotId 时按 from→to 反查
  → AoaGraph { numSource: 'derived' | 'persisted' }   // 界面/测试可据此判断
```

- 持久编号生效时 `nodes` **按 num 升序**（数组序=编号序=渲染序=i<j 断言同源）；派生口径下保持原拓扑序（旧行为逐字节不变，有专项用例）。
- 时间/几何**完全不受编号覆盖影响**（`x` 仍由 `es` 决定，`y` 仍走 band+重心行序）。

### 10.3 生产侧（`aoaGraphFromBuild`）

派生图 → 持久图元，**只在用户第一次手动布点时触发**（`AoaView.commitPin`，且 `graph.ok && nodes.length>0`）：

- 理由：手动 pins 的 key 是 `anchor`，一旦有人动过布局就需要"编号稳定"，否则下次打开编号变了、pin 的视觉归属也变了；凭空给所有工程固化只会制造历史包袱。
- `dot.layer` 初值取当前 `band`，并**优先沿用已有持久层号**（给后续"手动分层/智能位错"刀留位）。
- `arrow.id` 用两端锚点拼 `${from}>${to}`（跨拓扑稳定），不用内部自增 id。
- `beginDotId = 'S'`；`S`/`T` 打 `isBegin`/`isEnd`。

### 10.4 pins 降级（`pinsToPersist`）

新增纯函数，与 `prunePins` 的差别只有一条：**与当前自动位一致的键不落盘**。

| 键的处境 | `prunePins` | `pinsToPersist` |
|---|---|---|
| 失配（拓扑已变，anchor 不在了） | 剪 | 剪 |
| 与自动位相同（拖回原位/坐标快照残留） | 留 | **剪** |
| 真被搬动过 | 留 | 留 |

`AoaView.commitPin` 改用 `pinsToPersist`；重置按钮抽成 `resetPins`（语义不变）。
效果：`AOA_PINS_MAX=5000` 不再是"整图坐标快照上限"，而是"真手动布点数上限"。

### 10.5 验证（本轮实跑）

| 门 | 命令 | 结果 |
|---|---|---|
| 前端定向 | `vitest run src/schedule/aoa.test.ts src/schedule/aoaLayout.test.ts src/schedule/AoaView.test.tsx` | 79 passed（aoa 32 / layout 31 / AoaView 16） |
| 前端回归 | `vitest run src/schedule src/pages/SchedulePage.test.tsx src/pages/schedule` | **50 文件 / 669 例全绿**（39.0s） |
| 类型 | `tsc -b` | 0 错 |
| Lint | `eslint src/schedule` | 0 错 |
| Go | `go test ./internal/schedule/` + `./internal/app/ -run 'Schedule|BindingsCompleteness'` | ok / ok |

新增用例（13 例）：持久编号覆盖与排序、部分命中补号唯一、全不命中退回派生、箭线 id 标注、
固化后回灌编号一致、固化后插入新任务老编号不变、`prev.layer` 优先、arrow id 稳定、
`pinsToPersist` 三种处境。

### 10.6 已知边界

1. `layer` 目前只**承载不消费**（渲染仍走 band+重心行序）：留给"手动分层/智能位错"刀。
2. 固化后**不会自动清理**已删任务的点/箭（斑马也留着——`mergeDot` 只在建关系时合并）。文件会随删改缓慢变长，上限由 Go `validateBlocks`（20000/60000）兜底。
3. `numSource` 尚未在界面上提示（只在图对象与测试里可见）。


---

## 11 第 3 步落地记录（S1 检查器，2026-11-14）

### 11.1 为什么"只做前端、不做 check.go"

判定书原计划是 `checker.ts` + `check.go` 镜像。实施时查了 Go 侧现状：

| Go 侧已有 | 位置 | 与检查器的关系 |
|---|---|---|
| fail-closed 结构闸 | `project.go:Validate` | 悬空/自环/非法类型/分组行挂分配/工期越界——**保存时拒绝**，检查器只是提前告知 |
| 循环依赖裁决 | `cpm.go`（`存在循环依赖：…`） | 检查器直接引用 `cpm.error`，不分叉 |
| 语义 finding | `schedule_tools.go:schedule_analyze` 的 `checks` | 空分组/孤立/无前置/无收口/无里程碑/未定价，**字符串口径** |

再加一份 Go `check.go` 会变成**第三套口径**（Validate 一套、analyze 字符串一套、check.go 一套），
且没有消费者（agent 已用 analyze）。所以本步**刻意只做前端**：结构化 `Diagnostic` 只在 UI 侧存在，
措辞与 Go analyze 对齐。若将来要上移到 Go，正确做法是把 `analysis.go` 的 findings **结构化**（一次改两处消费），
而不是新增 check.go。

### 11.2 规则集（18 条，`checker.ts`）

| code | 严重级 | 定位 | 口径来源 |
|---|---|---|---|
| `cycle` | error | 项目级 | Go `cpm.error` 原文 |
| `link-dangling` | error | link | Go Validate |
| `link-self` | error | link | Go Validate |
| `link-bad-type` | error | link | Go Validate |
| `group-link` | error | link+task | 斑马「分区上存在关系」 |
| `group-assignment` | error | task | Go Validate（分组行禁挂分配） |
| `milestone-duration` | error | task | 斑马里程碑工期视为 0 |
| `duration-negative` / `duration-cd-over` | error | task | Go `validateDuration` / `validateDurationUnit` |
| `redundant-link` | warn | link | 斑马「多余关系」 |
| `negative-lag` | warn | link | 斑马/行业（负搭接） |
| `isolated-task` | warn | task | Go analyze「孤立项」 |
| `no-predecessor` | warn | task | Go analyze（首个非手动叶豁免） |
| `open-end` | warn | task | Go analyze（末位叶豁免） |
| `empty-group` | warn | task | Go analyze |
| `no-links` | warn | 项目级 | 新增（全计划零搭接） |
| `no-milestone` | info | 项目级 | Go analyze |
| `unpriced-assignment` | info | task | Go analyze `unpriced` |
| `high-float` / `near-critical` | info | task | 斑马「总时差」分组 + 行业阈值（20 / 2 天） |

设计取舍：
- **关键工作不逐条报**（会刷屏）：交给 DCMA 与横道红条；
- `hint` 只写"下一步"（对齐 Go 的 `可先 ops auto_chain 补缺省串联` 措辞），不放 `fix` 按钮描述——
  避免出现"UI 没有的修复动作"这种死字段；
- 与 `dcma.ts` 的分工写进文件头：**DCMA = 算比率的健康度报表，checker = 逐条可定位诊断**，规则重叠是刻意的。

### 11.3 接入

- `TaskInspector` 新增可选 `diagnostics?: Diagnostic[]`，抽屉里渲染「检查发现（N）」段：
  严重级标签（阻断/可疑/提示）+ 一句说明 + 「下一步：…」；空数组不渲染（干净任务保持简洁）。
- `SchedulePage` / `ScheduleTool` 各算一次 `checkProject(project, cpm)`，按 `inspectId` 过滤后传入；
  项目级只带 `cycle` 进抽屉（其余项目级提示属于整份计划，不该挂到某个任务上）。

### 11.4 验证

| 门 | 结果 |
|---|---|
| `checker.test.ts` | **19 例**（每条规则一正一负 + 排序 + 汇总 + 坏形输入不抛） |
| `TaskInspector.test.tsx` | 14 例（新增 2 例：渲染严重级/建议；无诊断不渲染） |
| 回归 `src/schedule` + pages/schedule | **51 文件 / 690 例全绿**（41.0s） |
| `tsc -b` / `eslint src/schedule` | 0 错 |
| Go `internal/schedule` + `internal/app(Schedule|Bindings)` | ok / ok |

### 11.5 下一步

判定书 §6 第 4 步 **S2 检视 + 前锋线**：`reviews.ts`（读 `Project.reviews`，算每期 `当期/累计完成`、`实际开始/完成` 派生）
+ 横道/AOA 上的前锋线几何 + 拉直（写回 `Task.progress` 与 `review.frozen`）+ 对比表导出。
消费第 1 步落好的 `reviews` 承载位。


---

## 12 第 4 步落地记录（S2 检视 + 拉直，2026-11-14）

**范围**：把第 1 步落好的 `Project.reviews` 承载位变成可用的检视模型 + 面板。
图上"前锋线线体"留到下一步（要动 GanttView 的 SVG 层，单独一刀更安全）。

### 12.1 三条口径（写进 `reviews.ts` 文件头，都是斑马行为不是我们的发明）

1. **历史不可覆盖**：多期并列；同一 `at` 新增 = 改那一期（`canAddReview` 拒绝并存），
   且**不能插入早于已冻结期的新期**（会让前锋线倒流）。
2. **拉直要把实际写回计划再冻结**：`straighten()` 写回 `progress` 并把该期 `frozen=true`；
   已冻结的期再拉直返回 `frozen=true / changed=0`。回写**只升不降**
   （`Math.max(当前, 该期)`）——避免把已完成又改回进行中。
3. **完成量按有效工期加权**：逐任务百分比算术平均没有意义（1 天任务与 100 天任务等价），
   分母不含里程碑与分组行；全是里程碑时退化为达成率。

### 12.2 模型 API（`reviews.ts`，256 行纯函数）

| 函数 | 作用 |
|---|---|
| `makeReview(project, at, date)` | 用当前进度打一期快照；`progress=100` 顺带填 `finish=at`（唯一可由"进度+数据日期"确定的信息） |
| `reviewPercent` / `taskStatus` | 该期完成%（回填优先，缺省回落 progress）/ 三态（未开始·进行中·已完成） |
| `projectPercent` | 整份计划加权完成量 |
| `latestReview` / `reviewAtOrBefore` / `sortedReviews` | 期序列定位 |
| `compareReviews(project, from, to)` | 期际对比：只列**有变化**的行 + 整体完成量差 |
| `straighten(project, at)` | 拉直：写回 + 冻结；返回 `{project, changed, frozen}` |
| `reviewProgress(project, review, cpm)` | 当期**计划应完成 vs 实际完成**（用 CPM 行算到该工作日的 elapsed，clamp 到工期） |
| `reviewWorkday` / `reviewDate` / `defaultReviewDate` | 日期↔工作日换算（走 `calendar.ts` 唯一口径） |
| `canAddReview` | 三道闸：重复 / 上限 30 / 早于已冻结期 |
| `frontlineWorkday` | 前锋线所在工作日（最新期的 at，供下一步画线） |

### 12.3 面板与接入（`ReviewPanel.tsx`，211 行）

- 结构照 `BaselinesPanel`（只收 `cpm`，工程与动作全走 store，可整体塞进任何 Popover）；
- 三段：打一期检视（日期缺省今天、早于开工日则取开工日）+ 期列表（工作日/日期/完成量/冻结标签/拉直·删除）
  + 当期两列（计划应完成%·实际完成%·偏差）+ 期际对比表（只列变化行）+ 前锋线位置提示；
- 接入：`SchedulePage` 工程信息条在「基线」左边加「检视」按钮（期数 >0 时高亮）。
  `ScheduleTool` 不加（与基线一致——工具面板只放轻量视图）。

### 12.4 store 三动作

`addReview(at,date)` / `removeReview(at)` / `straightenReview(at)`：都走 `pushHistory`（可撤销），
`canAddReview` / `frozen` 的拒绝原因**原样返回**给面板显示（不静默）。

### 12.5 验证

| 门 | 结果 |
|---|---|
| `reviews.test.ts` | **14 例**（加权非算术平均、三态、期序列、按工期算的计划量、拉直只升不降、冻结幂等、三道闸） |
| `ReviewPanel.test.tsx` | **6 例**（引导态、打点落库、同期禁用、拉直写回+冻结+按钮禁用、期际对比行、删除） |
| 回归 `src/schedule` + pages/schedule | **53 文件 / 712 例全绿**（35.8s，含 GanttView 前锋线跟随 2 例） |
| `tsc -b` / `eslint`（schedule + SchedulePage） | 0 错 |
| Go `internal/schedule` + `internal/app(Schedule|Bindings)` | ok / ok |

### 12.6 已知边界

1. **横道图已接线**（`GanttView` 的 `showFront`/`checkDate` 初值取最新一期检视：有检视默认开线并停在该期）；**AOA 视图的前锋线线体仍未画**（`frontlineWorkday()` 已备好，AoaView 侧待做）。
2. 检视的 `actuals` 目前只能由「当前进度快照」生成，**不支持逐任务手填实际开始/完成**（面板暂无编辑态）；
   数据结构（`ReviewActual.start/finish`）与 `taskStatus` 已经支持，补编辑 UI 即可。
3. `reviewProgress` 的计划量用「该工作日已跨过的工期」近似（非按日历逐日摊）；对均匀施工的任务准确，
   对"前松后紧"的实际分布只能算参照值——斑马的当期量口径同样是均匀摊（`pert_json` 无逐日曲线）。


### 12.7 补记：横道图前锋线跟随最新检视（同轮接线）

查 GanttView 时发现**前锋线早就实现了**（`GanttOverlay` 的 `sched-front-check/line/dot` +
`GanttToolbar` 的开关与日期框 + `frontierWd()` 按进度比例算折点），缺的只是**数据源**：
它一直由纯视图态 `checkDate`（默认今天）驱动，跟 `Project.reviews` 没关系。

本轮接通（3 行）：`GanttView` 的 `showFront`/`checkDate` 初值改为
**有最新检视 → 默认开线并停在那一期的工作日**（`reviewDate()` 反算 ISO），无检视仍回落到今天；
用户依旧可用工具栏覆盖（视图态不入文件）。
新增 2 例：有检视 → `.sched-front-check`/`.sched-front-line` 出现；无检视 → 不画。

> 至此"打一期检视 → 图上自动站到那一天 → 末端偏差由 `reviewProgress` 给出 → 拉直写回并冻结"
> 这条链闭合。AOA 视图的线体仍未画（下一刀）。

---

## 14 MPP 二进制导出判定（2026-11-14 后端复核，暂不立项）

判定书原列「S4 MPP 导出（0.5–1 刀）」。后端复核后**不按原估立项**：

1. **写 ≠ 读的反向**：现有 `mpp.go` 是只读蒸馏（MPXJ 口径字段子集 + mscfb 只读流）。
   写 MPP 需要：CFB 写入器（stdlib 无，mscfb 只读）+ Props/TBkndTask/Rsc/Assn/Cons/Cal
   六流完整 VarMeta/Var2Data/FixedMeta/FixedData 生成 + 真 Project 打开验证样本——
   MPXJ 本尊都只读不写 MPP（ProjectLibre 导出走 XML），读取子集表反向凑不出可被
   Project 接受的文件。
2. **斑马自证**：其对外交换文案是「导出XML格式，该格式可以在Project中转换打开」
   （MSPDI 主通道）；`导出 MPP 失败` 字串疑为失败路径兜底，非主通道。
3. **判定**：互操作闭环由 xlsx（2026-11-14 已贯通 ask_finish 上报列，commit 110f8553）
   + MSPDI XML（已有，前端 mspdi.ts）+ MPP 只读导入承担。若未来确需真二进制导出，
   单独立项（估 3+ 刀：CFB 写入器 → 六流生成 → 真机 Project 打开验证），不并入复刻主线。
