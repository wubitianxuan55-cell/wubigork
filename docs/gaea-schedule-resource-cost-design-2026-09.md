# gaea 进度计划资源成本能力设计（资源表 · 分配 · 成本 rollup）

> 状态：**草案 · 待拍板**（2026-09-06）。承接 releases/v4.121.0.md 欠账池「资源成本（动手前需先出设计方案拍板）」——本文即拍板用设计方案，未获拍板前不动任何代码。
> 纪律沿用：AI 产建议、引擎裁决（fail-closed）；取道不取器（零新依赖、不引入外部引擎）；工作日口径（工期/时距/费率周期均为工作日整数）；TS/Go 镜像引擎双侧同步。

---

## 0. 结论速览

- **模型**：Project 口径做减法——三类资源（工时 work / 材料 material / 成本 cost）+ 任务级分配（assignment）。成本公式确定化：任务成本 = 固定成本 + Σ分配成本；工时 = 工期(工作日) × 投入 × 费率 + 每次使用；材料 = 总量 × 单价；成本资源 = 固定金额。**不做 time-phased**（无 S 曲线/直方图），故 AccrueAt 等摊销口径 v1 不承载。
- **费率口径推荐「元/工日」**（国内工日惯例、与引擎工作日口径天然对齐），导出 mspdi 时 ÷8 换算 Project 的元/小时——Project 对齐是换算层的事，不进数据模型（拍板项 1）。
- **引擎**：新增 cost rollup 纯函数（输入 project + CPM 结果 → 任务/资源/总成本），与 cpm.ts/cpm.go 同范式 TS/Go 双侧镜像、测试互为镜像；Validate fail-closed 扩展（悬空引用/负费率/重复分配/分组行分配拒绝）。
- **绑定面全刀次 0 新增**：计划文件是 JSON 串过既有 GaeaScheduleLoad/Save 通道，agent 走既有 schedule_apply 的 ops 通道扩枚举——drift 预期维持 PASS@588。
- **刀序（4 刀）**：刀1 模型+引擎+持久化（0 绑定）→ 刀2 agent 通道（ops 扩展+三工具回执+技能条款，0 绑定）→ 刀3 板块 UI（资源弹层+甘特成本列+状态栏总成本，0 绑定）→ 刀4 mspdi 资源/成本互通（0 绑定+真机池）。每刀独立可发布、可单独 revert。
- **最大风险**：Project 打开含 Resources/Assignments 的 XML 时对 Assignment Cost 的重算/覆盖行为**未核实**，工日↔小时换算的正确性同样需真机裁决——刀4 必须配真机池走查，不真机不发布刀4。

---

## 1. 现状与缺口（逐项已核，2026-09-06 读码）

### 1.1 现状基线

| 层 | 现状 | 文件 |
|---|---|---|
| 数据模型 | `SchedProject{name,startDate,tasks,links,calendar?,baseline?,deadline?}`；`SchedTask{id,name,duration,level,progress,isMilestone?,mode?,manualStart?}`——**无资源/成本任何字段** | frontend/src/schedule/types.ts；internal/schedule/types.go（JSON 字段一一对应） |
| 引擎 | CPM（FS/SS/FF/SF+时距、正逆推、TF/FF、关键、手动模式）；工作日历互逆换算；基线快照/漂移；倒排校核——全部纯函数 fail-closed | cpm.ts/cpm.go、calendar.ts/calendar.go、baseline.ts/baseline.go、deadline.ts/deadline.go |
| 持久化 | 进度计划/当前计划.gsched.json：Go Load/Validate/Save（校验+原子写 fail-closed）；前端 normalizeProject 旧数据补缺省 | internal/schedule/project.go；store.ts |
| agent | schedule_get / apply（project 整量 + ops 增量：upsert_task/patch_task/remove_task/set_links/set_meta/auto_chain/set_baseline/clear_baseline）/ analyze（质检+叙事）；回执强制带总工期与关键数 | internal/gaea/tool/builtin/schedule_tools.go；internal/schedule/ops.go；技能条款 internal/gaea/skill/builtins.go |
| UI | 页头（工程名/开工/竣工/日历/基线）+ 工具栏 + 三视图（横道/单代号/双代号）+ 状态栏（共 N 项工作总工期 N 天/关键/基线漂移/倒排/工作制/同步指示）；甘特左表列：行号/任务名称/WBS/工期/开始/完成/模式/前置 | SchedulePage.tsx；GanttView.tsx |
| mspdi | **只通 Tasks+Calendars**：任务（大纲/工期/里程碑/进度/手动）、搭接、基准日历；导出无 `<Resources>`/`<Assignments>`，导入直接丢弃这两节 | frontend/src/schedule/mspdi.ts |

### 1.2 缺口清单（本设计要补的）

1. 模型无资源维度：不存在资源表、费率、任务↔资源分配、任务固定成本。
2. 引擎无成本 rollup：任何口径的成本（任务/资源/汇总）都算不出，agent 也无从谈起。
3. ops 增量操作集无资源/分配操作：agent 无法对话式挂资源、调费率。
4. UI 无任何资源/成本呈现：无资源表、无成本列、状态栏无总成本。
5. mspdi 互通只有进度不通成本：与 Project 往返必丢资源与费率。
6. 办公侧计划摘要卡（v4.121 刀12）无成本 chip（欠账候选，不进本四刀主线）。

---

## 2. 外部口径调研摘要

### 2.1 MS Project 资源/成本模型（官方口径为主）

- **三类资源**（已核实，MS Learn「Type Element」Resource 枚举：0=Material 材料 / 1=Work 工时 / 2=Cost 成本）：
  - 工时资源（人/机械）：计费=标准费率+加班费率+每次使用成本（Cost/Use，每次分配计一次）；MaxUnits 表可用上限（100%=1 个满负荷当量）；标准费率默认按小时，可选日/周/月/年与材料标签口径。
  - 材料资源（消耗性：钢材/混凝土）：有计量单位标签（MaterialLabel）；费率=元/单位；分配分「随工期变量」（默认，用量=单价口径×工期）与「固定用量」（FixedMaterial）两种。
  - 成本资源（差旅/规费等一次性费用）：**无费率/无 MaxUnits/无 MaterialLabel**，金额直接记在分配上（ProjectPlan365 与 MS Support「Enter costs for resources」口径一致）。
- **成本摊销 AccrueAt**（已核实枚举：1=Start 开始 / 2=Prorated 按进度比例·缺省 / 3=End 完成）：标准/加班成本何时计入任务成本——它只对 time-phased 展示有意义。
- **任务总成本**（MS 支持文档口径）：任务成本 = 固定成本（Fixed Cost）+ 各资源分配成本（工时×费率 + 加班工时×加班费率 + Cost/Use + 成本资源金额）；分组/汇总行向上 rollup，项目总成本=全任务汇总。
- **挣值（EVM）**：Project 在成本+进度数据之上提供 PV/EV/AC、CV/SV、CPI/SPI 等（MS Support「Earned value analysis」）——本文点到为止，列入「明确不做」（§7）。
- **mspdi XML 互通字段**（已核实，MS Learn Resource/Assignment Elements and XML Structure）：
  - `<Resources><Resource>`：`UID` `Name` `Type`(0/1/2) `MaterialLabel` `MaxUnits` `AccrueAt` `StandardRate`(decimal) `StandardRateFormat` `OvertimeRate` `OvertimeRateFormat` `CostPerUse` `Cost` 以及分时段费率表 `<Rates><Rate><RatesFrom/RatesTo/RateTable/StandardRate/…>` 等。
  - `<Assignments><Assignment>`：`UID` `TaskUID` `ResourceUID` `Units`(float，1.0=100%) `Cost`(decimal) `FixedMaterial`(boolean) `Work` `OvertimeWork` `CostRateTable` `RemainingCost/ActualCost` 等。
  - **未核实**：`StandardRateFormat` 各枚举值（小时/日/周…的具体整数）；实施时以 XSD 注解+真机导出样本钉死，不凭记忆写死。
- **加班费率/加班工时**：Project 对工时资源分标准/加班两套费率——依赖「工时」独立口径（RegularWork/OvertimeWork），与 gaea 工期制不直接对应，v1 不做（§7）。

### 2.2 国产工具口径（斑马进度等，浅调研）

- 斑马进度计划（广联达）：有【资源】页签与「人/料/机计划资源管理」，按工作资源用量自动统计生成**每日/每月资源用量统计表**并可导出 Excel；官方有「巧用资源功能，管控计划成本目标」实操课（B 站第 37 期，标题级佐证）。→ 口径=资源挂工作 + 时段统计表。
- **未核实**：斑马费率表/成本目标的具体字段与公式（公开资料仅到功能名，无字段级文档）；Projexif 等其他国产工具未检索到公开字段级资料，不引申。
- 结论：时段统计（直方图/S 曲线）是国产主流形态，但它属于 time-phased 范畴，v1 不做（§7），先把「资源表+分配+成本 rollup」的地基打对。

### 2.3 对 gaea 的取道不取器

取：三类资源分类、分配模型、成本=固定+分配 rollup 的确定性公式、mspdi 字段口径（互通所需最小子集）。
不取：任务三类型（fixed units/duration/work）联动方程、资源日历、资源平衡/超载自动排程、分时段费率表（Rates）、time-phased 数据、EVM 全套、加班费率——全部是「工时制引擎」的器，gaea 是工期制（工作日整数），取其道须重写成工期制口径。

---

## 3. 设计提案

### 3.1 数据模型扩展（TS/Go 双侧同构，JSON 字段一一对应）

```ts
/** 资源类型（对齐 Project：工时/材料/成本三类） */
export type ResourceType = 'work' | 'material' | 'cost'

/** 资源（Project 资源工作表的工期制子集） */
export interface SchedResource {
  id: string
  name: string
  type: ResourceType
  /** 材料计量单位（type=material 有意义：t/m³/…，展示与导出用） */
  unit?: string
  /** 标准费率（元）：work=元/工日；material=元/单位；cost 不用 */
  standardRate?: number
  /** 每次使用成本（元，每条分配计一次；work/material 可用） */
  costPerUse?: number
  /** 工时资源可用上限（默认 1；v1 仅承载与 mspdi 往返，不做平衡） */
  maxUnits?: number
}

/** 分配（任务↔资源；(taskId,resourceId) 唯一，无独立 id） */
export interface SchedAssignment {
  /** 叶任务 id（分组行禁止分配，fail-closed） */
  taskId: string
  resourceId: string
  /** 工时资源投入强度（默认 1；成本=工期×units×费率） */
  units?: number
  /** 材料固定总量（type=material：总量×单价，不随时长变） */
  quantity?: number
  /** 成本资源金额（type=cost：该分配的固定金额，元） */
  amount?: number
}
```

- `SchedTask` 增可选 `fixedCost?: number`（任务固定成本，元）。
- `SchedProject` 增可选 `resources?: SchedResource[]`、`assignments?: SchedAssignment[]`。
- 金额单位：元，number，引擎 rollup 后四舍五入到分（0.01）；校验拒绝 NaN/Infinity/负数。
- 按字段名引用（resourceId/taskId），禁止按下标——对齐 gbase 视图按列名引用的既有纪律。

### 3.2 引擎扩展：成本 rollup（纯函数，TS↔Go 镜像）

新增 `cost.ts` / `internal/schedule/cost.go`，与 cpm 同范式（纯函数、无 IO、无时钟）：

```
computeCosts(project, cpm) → CostResult
  CostResult {
    ok: boolean                    // CPM 未过（循环依赖）时 ok=false，成本不出（fail-closed）
    error?: string
    rows: Record<taskId, { fixed, assigned, total }>   // 叶任务
    byResource: Record<resourceId, number>             // 资源维度汇总
    total: number                  // 项目总成本 = Σ 叶任务 total
  }
```

- 分配成本公式（确定性，无隐式行为）：
  - work：`effDur(工作日) × units(默认1) × standardRate + costPerUse`
  - material：`quantity × standardRate + costPerUse`
  - cost：`amount`
- 任务成本：`fixedCost(默认0) + Σ 该任务分配成本`；分组行 = 子孙叶任务求和（复用扁平 WBS 滚动口径，同甘特汇总条，不经独立存储）。
- 里程碑 effDur=0：工时分配成本自然只剩 costPerUse——引擎不给里程碑特殊规则，公式自己说话。
- **fail-closed 校验**（并入 project.go Validate 与 normalizeProject）：
  - 分配引用的任务/资源必须存在；`(taskId,resourceId)` 重复拒绝；
  - 分组行（level=0）挂分配拒绝；负费率/负数量/负金额/负 units 拒绝；
  - work 分配不填费率且无 costPerUse → 合法（成本=0），但 schedule_analyze 出 finding「已分配未定价」（AI 建议层，非引擎拒绝）；cost 分配缺 amount → 合法（=0）+同上 finding。
- CPM 未过（环依赖）时成本 rollup 返回 ok=false：**成本不是进度的替代裁判，但服从同一裁决**。
- 镜像纪律：cost_test.ts 与 cost_test.go 同批场景同批期望值（沿用 cpm/baseline/deadline 三对先例的镜像方法）。

### 3.3 持久化：gsched.json schema 演进与旧文件兼容

- 新字段全部 optional，**旧文件零迁移可读**：无 resources/assignments/fixedCost 即「无资源维度」，一切现状行为不变（与 baseline/deadline 两次演进同法，文件无版本号亦无需加）。
- `normalizeProject` 扩展：缺省补 `resources: []`、`assignments: []`；坏形字段（非数组/字段类型不符）丢弃——容错进板块，校验拒绝落盘在 Go Validate（fail-closed 两道闸，分工同现状）。
- `Validate` 扩展 §3.2 所列 fail-closed 条款（Load/Save 一体生效，agent 与板块同受约束）。
- 办公侧 `parseSchedSummary`（gschedSummary.ts）后续可加 `totalCost` chip——欠账候选（§6），不在本四刀主线。

### 3.4 UI（推荐版面）

推荐「**资源弹层 + 甘特成本列 + 状态栏总成本**」三件套，**不做第四 Segmented 视图**——资源是计划的一个维度（Project 也以资源工作表/列呈现，不是第四张图）：

1. **工具栏「资源」按钮**（Popover/Modal，与基线弹层同范式）：资源表（名称/类型/费率（带单位标注 元/工日、元/单位）/每次使用/已分配任务数/资源成本合计）+ 增删改；类型切换联动字段显隐（cost 资源隐藏费率，显金额口径说明）。
2. **甘特左表加「成本」列**（列尾，右对齐，宽≈72）：叶任务=明细合计（含固定成本），分组行=子孙汇总（灰显汇总口径）；无任何资源/成本数据时列值留空不占视觉。列宽预算：现 COLS 合计已较满，「成本」列入列配置数组即可，横道区自适应。
3. **行级分配编辑**：任务行「资源」Popover（复用前置 Popover 范式）：资源多选 + units/quantity/amount 输入 + 该任务固定成本输入；提交=整体替换该任务分配集（与 set_links「整体替换入边」同语义）。
4. **状态栏总成本段**：`总成本 ¥N`（有资源/成本数据时常驻；无数据不显示——诚实呈现，不留「¥0」假象）。
5. 单代号/双代号视图不动；拖拽改工期等既有交互不动（工期变了成本自动变——引擎裁决的价值）。

### 3.5 agent 工具接线（零新工具，三件套扩容）

- **ops 增量操作集**（internal/schedule/ops.go 扩 Type 枚举，前端无镜像必要——ops 只在 Go 侧执行，沿用现状）：
  - `upsert_resource` / `patch_resource`（指针三态）/ `remove_resource`（级联删其分配）；
  - `set_assignments`：整体替换某任务的分配集（set_links 同范式）；
  - `patch_task` 增 `fixedCost`。
- **schedule_get**：回执增 `resources` / `assignments` / `costs{total,byTask,byResource}`。
- **schedule_apply**：回执**强制带写入后总成本**（与总工期/关键数同列——「成功≠正确」纪律延伸：改了费率必须让模型核对总成本变化）；compact 通道（compact.go 的 compactDesc/compactSchema）同步维护。
- **schedule_analyze**：成本叙事数据（总成本、成本 Top5 任务、按资源汇总）+ 新 findings（已分配未定价 / 负值被拒在 apply 层不入 analyze）。
- **技能条款**：internal/gaea/skill/builtins.go schedule 技能 Body 增「资源与成本」分节（三类资源口径、费率单位纪律、先建资源再挂分配、成本改动的回执核对要求）。

### 3.6 mspdi 导入导出互通范围（刀4）

| 方向 | 内容 | 口径 |
|---|---|---|
| 导出 | `<Resources>`：UID/Name/Type(0/1/2)/MaterialLabel/StandardRate/CostPerUse/MaxUnits/AccrueAt(固定写 2=Prorated)；`<Assignments>`：TaskUID/ResourceUID/Units/FixedMaterial/Cost | 费率换算：元/工日 ÷8 → 元/小时（StandardRateFormat 待实施时按 XSD/真机钉死）；不导出 Rates 分时段表/加班费率（模型没有，诚实不导） |
| 导入 | 读 Resources/Assignments 回填 SchedResource/SchedAssignment；成本资源读 Assignment Cost → amount | 宽松容错沿现状：缺失字段走缺省；Type 非法值回落 work 并标警告 |
| 边界 | Work/RegularWork 等工时字段不导入（工期制模型无承载）；Task 的 Cost 字段 → fixedCost（若与分配重算冲突以固定成本为准并标警告） | 真机池必查：Project 打开导出文件是否重算/覆盖 Cost；斑马进度打开行为同查 |

---

## 4. 分刀方案（每刀独立可发布、可单独 revert）

| 刀 | 内容 | 绑定面（新增 Go 绑定方法） | 测试面 | 可发布性 |
|---|---|---|---|---|
| **刀1 模型+引擎+持久化** | types.ts/types.go 增 SchedResource/SchedAssignment/fixedCost；cost.ts/cost.go rollup 镜像；normalizeProject/Validate 扩展；gsched.json 静默承载 | **0**（JSON 过既有 Load/Save 通道） | cost 镜像用例 ~12-16 + normalize/Validate ~8；Go 镜像同批 | 可发布：数据层先行无 UI 入口，行为对老用户零变化；revert=删字段（文件向后兼容） |
| **刀2 agent 通道** | ops 增 upsert_resource/patch_resource/remove_resource/set_assignments + patch_task.fixedCost；schedule_get/apply/analyze 成本回执与 findings；compact 通道；技能条款 | **0**（三工具已注册，扩 schema 与回执） | ops.go 用例 ~8 + schedule_tools 回执 ~4 + schedule_skill_test ~2 | 可发布：agent 可先对话式挂资源跑通全链（文件经板块只读呈现成本于状态栏之外亦可）；revert 独立 |
| **刀3 板块 UI** | 资源弹层 + 甘特成本列 + 任务资源 Popover + 状态栏总成本段 | **0** | 组件测试 ~12-14（弹层 CRUD/成本列汇总/Popover 整体替换/状态栏条件显隐）+ ?mock=1 全链走查 | 可发布：纯 UI 层；revert 只回退 UI，数据与 agent 通道不受损 |
| **刀4 mspdi 互通** | 导出 Resources/Assignments + 导入回读；费率工日↔小时换算纯函数 | **0** | mspdi.test.ts ~8（含换算边界/Type 非法回落）+ **真机池**（MS Project/WPS/斑马开导出文件、导回往返） | 可发布前置=真机走查通过；revert 独立（互通是外缘，回退不影响内部模型） |

刀序理由：引擎先行保证「AI 产建议、引擎裁决」从第一天成立；agent 通道第二，用户可在 UI 落地前经对话验证模型口径；UI 第三让版面拍板（§5 拍板项 5）不阻塞数据层；mspdi 最后且绑定真机池——互通的外部行为最不可控，放最外缘。

---

## 5. 拍板问题清单（每点给推荐项，拍板后生效）

| # | 决策 | 推荐 | 备选 |
|---|---|---|---|
| 1 | 费率单位口径 | **元/工日**（国内工日惯例；与工作日口径天然对齐；mspdi 导出 ÷8 换算） | 元/小时（Project 字段全对齐，但引擎与录入全要 ×8 往返，易错） |
| 2 | 工时资源成本公式 | **工期×units×费率+每次使用**（fixed-units 简化口径：改工期成本自动变，引擎裁决） | 引入独立工时字段+任务三类型（fixed units/duration/work 联动方程，Project 全口径，复杂度高，违背工期制） |
| 3 | 材料资源口径 | **固定总量×单价**（清单工程量×综合单价的国内惯例；不随时长变） | 随工期变量（Project 默认：单价口径×工期；需引入「单位时间用量」概念） |
| 4 | 分组行 fixedCost | **禁止**（fail-closed；汇总唯一口径=子孙求和，避免双源） | 允许（Project 允许汇总行固定成本；需定义汇总=Σ子孙+自身） |
| 5 | UI 版面 | **资源弹层+甘特成本列+状态栏总成本**（资源是维度不是视图） | 第四 Segmented「资源」视图（资源使用表：资源×任务矩阵；留 v2 口） |
| 6 | mspdi 互通方向 | **双向**（导出+导入；往返不丢资源费率） | 仅导出（省真机池一半工作量，但导入丢成本违背互通初衷） |
| 7 | 资源日历/资源平衡/加班费率/超载自动排程 | **v1 一律不做**（§7 已列；maxUnits 仅承载供导出） | 超载检测 finding 先行（需时段重叠分析，成本不小，建议欠账） |
| 8 | EVM | **明确不做、不预留字段**（JSON optional 演进天然可后加，不必预占） | 基线成本字段先行（在基线快照加 cost 列；若用户后续要 EVM 再拍板） |

---

## 6. 风险与欠账

| 风险 | 对策 |
|---|---|
| **Project 对导出 XML 中 Assignment Cost 的重算/覆盖行为未核实**（本设计最大不确定性） | 刀4 绑定真机池：导出→真机开→改值→导回→diff；不真机不发布刀4；StandardRateFormat 枚举以 XSD+真机样本钉死 |
| TS/Go 双实现镜像负担（成本是第二套 rollup） | 镜像测试纪律沿用（cpm/baseline/deadline 三对先例）；公式表 §3.2 钉死口径，测试互为镜像 |
| 单位混乱（元/工日 vs 元/小时 vs 材料单位） | 数据模型只存元/工日与元/单位；UI 与技能条款显式标注单位；换算只存在于 mspdi 导出层纯函数 |
| 旧文件/第三方脏数据（负费率、悬空引用） | normalize 容错读 + Validate fail-closed 落盘双道闸；agent 错误原文回传（现状纪律） |
| schedule_apply 回执膨胀（成本字段） | compact 通道同步维护；byTask/byResource 仅 get/analyze 带，apply 回执只带 total（+改动任务行成本） |
| 工期拖拽导致成本静默变化（用户无感涨价） | 状态栏总成本常驻+甘特成本列实时反映；后续欠账：成本变化量进「改动待保存」提示语 |

**欠账池候选（本设计动完后再排）**：办公摘要卡总成本 chip；资源使用视图（资源×任务矩阵，拍板项 5 备选）；超载/峰值检测 finding；月度资源统计表（斑马口径的 time-phased 子集）；基线成本快照（拍板项 8 备选）；多币种。

## 7. 明确不做（v1 及可见将来，理由随列）

- **EVM 全套**（PV/EV/AC/CV/SV/CPI/SPI）：依赖实际成本跟踪（actuals），gaea 是计划期工具，无实耗数据源；
- **time-phased 成本/资源直方图、S 曲线、月度现金流量**：需逐日摊铺引擎与 AccrueAt 摊销口径，与「成本 rollup 确定化」是两代复杂度，需求出现时另案；
- **加班费率/加班工时**：依赖工时制（RegularWork/OvertimeWork），工期制模型无承载；
- **资源日历**（资源级工作制）：成本公式只用任务工作日工期，资源日历只影响产能不影响 v1 成本；
- **资源平衡/自动错峰**：触碰排程裁决权——引擎只裁决，重排由人拍板（与倒排校核「不自动改排程」同一立场）；
- **分时段费率表（Rates/RatesFrom/To）与费率升级**：单费率已覆盖国内综合单价惯例；
- **多币种/汇率**：元定死，字段不带币种；
- **外部引擎/依赖**：零新依赖纪律不变，成本引擎自研纯函数。

## 8. 参考

- MS Learn：Type Element（Resource 0=Material/1=Work/2=Cost；Task 0=fixed units/1=fixed duration/2=fixed work）—— https://learn.microsoft.com/en-us/office-project/xml-data-interchange/type-element-multiple-parents
- MS Learn：Resource Elements and XML Structure（StandardRate/CostPerUse/AccrueAt/Rates/MaterialLabel/MaxUnits 等）—— https://learn.microsoft.com/en-us/office-project/xml-data-interchange/resource-elements-and-xml-structure
- MS Learn：Assignment Elements and XML Structure（TaskUID/ResourceUID/Units/Cost/FixedMaterial 等）—— https://learn.microsoft.com/en-us/office-project/xml-data-interchange/assignment-elements-and-xml-structure
- MS Support：Enter costs for resources；Earned value analysis for the rest of us —— https://support.microsoft.com/en-us/project/enter-costs-for-resources ； https://support.microsoft.com/zh-cn/project/earned-value-analysis-for-the-rest-of-us
- ProjectPlan365：Cost Resources / Cost Tracking（成本资源无费率、Cost/Use 每分配一次等口径佐证，二手）—— https://www.projectplan365.com/articles/cost-resources/
- 广联达斑马进度：人/料/机计划资源管理（知乎专栏）与产品页（功能名级佐证，字段未核实）—— https://zhuanlan.zhihu.com/p/272751030 ； https://www.glodon.com/product/204.html
- 既有基准：docs/gaea-office-mindmap-base-design-2026-09.md（格式与刀序范式）；internal/schedule/* 与 frontend/src/schedule/*（现状全量读码，2026-09-06）；releases/v4.113.0.md、v4.121.0.md（欠账出处与绑定面口径）
