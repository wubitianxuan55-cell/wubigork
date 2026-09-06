# gaea 进度计划双工期口径设计（定额日历天 vs 工作日）

> 状态：**草案 · 待拍板**（2026-09-06）。承接拍板池欠账「双工期口径（定额日历天 vs 工作日）」——本文即拍板用设计方案，未获拍板前不动任何代码。
> 纪律沿用：AI 产建议、引擎裁决（fail-closed）；TS↔Go 镜像铁律（internal/schedule/ 与 frontend/src/schedule/ 同构，测试互为镜像）；gsched.json optional 字段演进零迁移（baseline/deadline/resources/aoaLayout 四次先例）；零新绑定。

---

## 0. 结论速览

- **推荐模型：任务级单位标记 `task.durationUnit?: 'wd' | 'cd'`（缺省 wd=工作日）**——`duration` 恒为「当前单位下的数值」，不新增第二工期字段，无双源漂移；与 mspdi 的 `Duration + DurationFormat`（7=天 / 8=日历天 elapsed）语义 1:1 对应。旧文件无此字段=全程工作日，**零迁移、零行为变化**。
- **换算语义**：cd 任务正推以 `wdToDate(es)`（es 所在工作日）为起锚点，按自然日走 `锚点 + duration` 天得完成边界，**边界落到其后第一个工作日**（ceil 吸附）得 ef；逆推走镜像回退。养护 28 自然日在周一~五日历下 ≈ **20 个工作日跨度**（对比误录 28 工作日失真 +8 个工作日）。混合口径下正逆推/时差/关键线路自洽（逆推定义「fwd(ls) ≤ lf 的最大 ls」，平段吸收周末，TF 语义不破）。
- **v1 fail-closed 收窄**：cd 任务**仅允许 FS 搭接**（养护「完成才可后续」主场景即 FS；SS/FF/SF 涉及 cd 会引入对 dur 的不动点迭代，列欠账放开，放开语义矩阵见 §3.2.5）。lag/里程碑/分组行/倒排校核口径全部不变。
- **成本口径**：work 分配按**等效工作日跨度（ef−es）**计费——养护期周末不记工日，与「元/工日」费率天然对齐（拍板项 3）。基线 dur 存等效跨度，漂移在工作日空间对比，口径不变。
- **刀序（4 刀，全程 0 绑定）**：刀1 模型+引擎（cpm 日历感知换算，无 cd 任务时快路径逐位等价）→ 刀2 agent 通道（patch_task.durationUnit + analyze 出「日历天任务」清单 + 技能条款）→ 刀3 板块 UI（工期列单位后缀+录入切换+拖拽换算）→ 刀4 mspdi 互通（DurationFormat 8 导出/导入，绑真机池）。每刀独立可发布、可单独 revert。
- **最大风险**：mspdi 中 elapsed 工期的 `Duration` 序列化口径（PT168H 还是 PT56H）与 Project 往返重算行为**未核实**——刀4 必须真机走查，不真机不发布刀4。

---

## 1. 现状与缺口（逐项已核，2026-09-06 读码）

### 1.1 现状基线

| 层 | 现状 | 文件 |
|---|---|---|
| 数据模型 | `SchedTask.duration: number`——唯一工期字段，**全程工作日整数**；时距 `SchedLink.lag` 亦工作日；里程碑工期视为 0 | frontend/src/schedule/types.ts；internal/schedule/types.go（JSON 字段一一对应） |
| CPM 引擎 | `computeCpm(tasks, links)` 纯函数 fail-closed：正推 EF=ES+effDur、逆推 LF−dur、TF/FF、关键、手动模式——**签名不含日历与开工日**，引擎对「日历换算」无感知 | cpm.ts / cpm.go；effDur（cpm.ts L18）是有效工期单一来源 |
| 日历工具 | `wdToDate/dateToWd/deadlineWorkdays` 双向换算（workweek+holidays，扫描上限 3650 天）；**只被视图/导出/倒排调用，CPM 不经它** | calendar.ts / calendar.go |
| 成本 | work 公式 `effDur(工作日)×units×费率+costPerUse`——effDur 直接取自任务字段 | cost.ts / cost.go（computeCosts 已接收 cpm 结果，rows 含 es/ef） |
| 基线 | 快照行存 `es/ef/dur(=effDur)/critical`，漂移在工作日序号空间对比 | baseline.ts / baseline.go |
| 倒排 | `deadlineWorkdays`（[开工,竣工] 工作日计数）vs `cpm.duration`，公式纯工作日制 | deadline.ts / deadline.go |
| mspdi | 导出 `<DurationFormat>7</DurationFormat>` **写死**、`Duration=PT{d*8}H`；导入 `parseDurationDays`（÷480 分钟）**忽略 DurationFormat**——Project 的 elapsed（日历天）任务导入会被静默当工作日 | mspdi.ts L89、L120-126 |
| UI | 甘特时间轴已按**自然日**绘制（每天一列+非工作日底纹，v4.112 斑马口径），条形 x=wd→自然日偏移，**跨周末自然变宽**；工期列 InputNumber 无单位表达；拖拽 resize 落点按最近工作日吸附（drag.ts） | GanttView.tsx；drag.ts |
| agent | 三工具回执口径「工期为工作日（按日历扣除周末/节假日）」「总工期 X 天」；技能条款「工期一律工作日整数」「汇报日期说明工作日序还是日历日期」 | internal/gaea/tool/builtin/schedule_tools.go；internal/gaea/skill/builtins.go |
| 摘要卡 | 办公侧 gschedSummary 的 duration/finishDate=工作日口径（wdToDate 推竣工） | gschedSummary.ts |

### 1.2 缺口清单（本设计要补的）

1. **自然日定时工作无法表达**：混凝土养护 28 天、油漆干燥期、嫁接成活观察期等按自然日走的工作，现只能录 28 **工作日** → 日历跨度 ≈40 天（28+2 个周末×…按 5/7 折算 39.2），横道条与竣工日期双双失真，且随节假日更多。
2. **引擎无换算通道**：CPM 签名不含 calendar/startDate，日历工具与引擎断开；日历天工期进不了排程。
3. **工期单一来源撑不住双口径**：effDur 是成本/基线/前锋线共用的工期单一来源，无单位语义。
4. **mspdi 单向失真**：导出恒 7（天），导入忽略 DurationFormat——Project 侧 `28 ed` 任务导回 gaea 变 28 工作日，静默放大。
5. **UI/agent 无口径表达**：工期列、拖拽、回执、技能纪律全部只有工作日一元词汇，AI 编计划时遇到养护类工序只能失真录入学：「工期一律工作日」条款与定额自然日场景直接冲突。

---

## 2. 外部口径调研摘要

### 2.1 MS Project：两条并存的机制（官方口径已核实）

Project 表达「日历天定时」有两条路，语义不同：

- **机制 A：工期单位 elapsed（本文取道）**——任务工期可带 `e` 前缀单位（`7 ed`=7 个日历天）。官方定义（MS Learn DurationFormat Element，已核实）：**"Elapsed time counts all time, including non-working time specified in the project, resource, or task calendar"**，例：7ed 从周一起算到下周日（同日历下 7d 会跳过周末）。工期数值与单位本就一体存于任务（Duration + DurationFormat），elapsed 只是格式枚举的一个子集：
  - mspdi `DurationFormat` 枚举（已核实）：3=分 5=时 **7=天 8=日历天(ed)** 9=周 11=月，elapsed 变体 4/6/8/10/12；
  - `Duration` 元素以 ISO8601 小时串存储（如 7 天=PT56H0M0S，按 MinutesPerDay=480 折算）——**elapsed 工期（8）在 XML 里的确切序列化（PT168H vs PT56H）未核实**，实施须以 XSD+真机导出样本钉死，不凭记忆写死。
- **机制 B：任务日历**——给任务挂独立基准日历（如 24 小时制/每周 7 天），工期内全程按该日历计工作。mspdi 经 `<TaskCalendarUID>`（+`IgnoreResourceCalendars`）承载（MS Learn CalendarUID Element 已核实；「24 小时日历做养护/干燥」为社区常用做法，二手佐证：Tensix/Consult Leopard/ProjectPlan365）。**未核实**：任务日历与资源日历/基准日历冲突时的逆推细节。

两条路对比：机制 A 把「日历天」收在**工期单位**里，模型最小、单基准日历架构零冲突；机制 B 引入多日历管理（日历表、任务↔日历引用、多日历逆推），是 Project 全量日历体系的第一步。

### 2.2 国产工具口径（斑马进度等，浅调研）

- 斑马进度的工期口径**由日历驱动**：默认日历不设假期即连续自然日，插入假期/周休后按工作日（官方社区《系统学习文档》「设置日历及假期」章节 + 知乎官方教程，论坛级佐证）；有「特殊工序单独设置日历」操作（官方抖音教程标题级佐证，即任务日历类似物）。**未核实**：斑马是否支持任务级工期单位（elapsed 类）——公开资料未见字段级文档。
- 社区高频问题「做出来的工期比招标要求多一天」（斑马论坛帖）佐证：国内合同/定额工期口径混杂「自然日连续时间」与「含首尾计数」，双口径需求真实存在。
- P6 的作业日历（Activity Calendar）同属机制 B 路线——常识级印象，**本文未逐条核实，不引申**。

### 2.3 对 gaea 的取道不取器

- **取：机制 A「工期单位」**——`durationUnit:'cd'` 与 Project 的 Duration+DurationFormat(8) 语义 1:1，单基准日历架构零新增管理面，与 gaea「工期制（工作日整数）」模型是加一个枚举的关系。
- **不取：机制 B 任务日历/多日历**——需要日历资源表、任务↔日历绑定、多日历正逆推与换算，是「日历制引擎」的器；gaea 单基准日历（workweek+holidays）已覆盖施工停工窗口表达，为养护类工序引入多日历是杀鸡用牛刀（拍板项 1 备选留档）。
- **不取**：周/月/小时等其余 DurationFormat 单位、lag 的 elapsed（LagFormat 8）——时距恒工作日（§7）。

---

## 3. 设计提案

### 3.1 模型：单位标记（推荐）vs 第二工期字段（备选）

```ts
/** 工期单位（v3 双工期口径刀1）：缺省 wd=工作日（现状口径，旧文件零迁移）；cd=日历天（自然日定时） */
export type DurationUnit = 'wd' | 'cd'
// SchedTask 增：
//   durationUnit?: DurationUnit   —— duration 恒为「当前单位下的数值」
```

- `SchedTask.durationUnit?: 'wd' | 'cd'`（Go 侧 `DurationUnit string \`json:"durationUnit,omitempty"\``，同构）；缺省/空=wd，**所有既有读码点行为不变**。
- `duration` 语义随单位：unit=cd 时存**自然日数**（整数 ≥0）。不新增第二工期字段——`durationCal` 方案（备选）会造出「两个工期字段谁是真值」的双源问题：引擎/成本/基线/UI 每个读点都要裁决优先级，还须维护「duration 与 durationCal 同步或清零」的额外不变量，normalize 容错也修不了语义冲突（只能 Validate 拒）。单位标记方案里工期数值永远只有一份。
- fail-closed 校验（并入 Validate + normalizeProject，两道闸分工同现状）：
  - `durationUnit` 非空时必须为 `wd|cd`，否则拒绝落盘（normalize 丢弃回落 wd）；
  - cd 仅限**叶任务且非里程碑**（里程碑 effDur=0 口径不动；分组行 duration 恒 0 同 fixedCost 先例禁止 cd）；
  - cd 数值上限 3650（≈10 年，与 calendar 扫描上限同源，防呆一致）。
- **零迁移**：旧文件无字段=wd；persist merge / importProject 经 normalizeProject 补缺省，沿用 resources 刀先例。
- 模型对比一览：

| 方案 | CPM | 成本/基线 | mspdi | 双源风险 | 结论 |
|---|---|---|---|---|---|
| `durationUnit?: 'wd'\|'cd'`（推荐） | 读点+单位分支 | effDur 换源（§3.3） | Duration+Format 1:1 | 无（单值字段） | **推荐** |
| `durationCal?: number` 第二字段 | 同级 | 同级 | 需字段映射 | 有（优先级/同步不变量） | 备选留档 |
| 任务日历（机制 B） | 多日历正逆推重写 | 需按日历逐日计 | CalendarUID 全套 | 引入日历实体 | 明确不做（§7） |

### 3.2 引擎换算语义（TS↔Go 镜像，纯函数 fail-closed）

#### 3.2.1 正推（forward）

- 记 `anchor(es) = wdToDate(startDate, es, calendar)`（es 所在工作日；es 本身即工作日序号，手动任务 manualStart 同理）。
- wd 任务（现状，逐位不变）：`ef = es + effDur(t)`。
- cd 任务：`边界日 = anchor(es) + duration`（**自然日加法**）；`ef = 第一个 ≥ 边界日 的工作日序号`（ceil 吸附；边界恰为工作日即当日）。
- **例**（Mon–Fri、无节假日，es=0 起 28cd）：锚点=第 0 工作日（周一），锚点+28 自然日=第 4 周周一 → ef=第 20 个工作日序号，**等效工作日跨度 ef−es=20**；误录 28wd 则跨度 28——失真 +8 个工作日（≈11 个日历天）被本设计消除。边界落在周六/周日时（27cd、26cd）同样吸附到同一 ef：**26~28cd 在整周工作制下等效跨度相同**（吸附折叠，见 3.2.3 逆推如何消化）。

#### 3.2.2 签名与快路径（侵入面控制）

- TS：`computeCpm(tasks, links, ctx?)` 增可选第三参 `{ calendar?, startDate? }`——全部既有调用点（store/gschedSummary/cost/mspdi 测试等）零改动可编译；SchedulePage 传入 project.calendar/startDate。
- Go：新增 `ComputeCpmCal(tasks, links, cal *Calendar, start string)`，`ComputeCpm` 委托为 `ComputeCpmCal(tasks, links, nil, "")`（Go 无默认参，先例式双入口）。
- **快路径铁律**：任务表无任何 `unit==='cd'` 时走既有代码路径，**结果逐位等于现状**（cpm 既有 18+ 用例零改动全绿即证明）。cd 存在时才启用日历换算分支。calendar/start 缺失而有 cd 任务 → fail-closed 返回 ok=false（错误原文明示「缺日历无法计算日历天任务」，不静默）。

#### 3.2.3 逆推与关键线路（混合口径自洽性）

- 关键定义：`fwd(es) = 上述正推映射`（单调不减，且**有平段**——周末吸附使相邻 es 映射同 ef）。
- 逆推：`ls = 最大的工作日序号 s 使 fwd(s) ≤ lf`（「最迟开始：再晚一个工作日，完成边界就越过 lf」）。实现=镜像回退：从 `wdToDate(lf)` 起自然日回退 duration 天、向下吸附到工作日，再用有界循环（上限=SCAN_LIMIT 同源）双侧校正至满足上式；确定性由构造保证，平段/节假日用例测试钉死。
- 性质（测试必钉）：`fwd(ls) ≤ lf < fwd(ls+1)`；wd 任务退化为 `ls = lf − dur`（与现状逐位一致）。
- **时差/关键**：TF=ls−es、FF、critical=tf===0 公式不动。平段自动给出正确语义：cd 任务晚开工 N 个工作日若因周末吸收不推移竣工，则 TF>0（真实富余）；压线时 TF=0 且标关键。**cd 任务可入关键线路**（养护期常是关键约束），手动 cd 任务沿 manual 既有规则（不参与关键、不回传约束，ef=fwd(manualStart)）。
- 搭接边界组合的**逐项矩阵**（FS/SS/FF/SF 的 forward/backward 公式中何处在用 dur）已逐项推导：凡边界项引用「本任务 dur 且 dur 随 es 变」的组合产生不动点迭代——FS 全组合无此问题（FS forward 不含 durTo；FS backward 的 durTo 取 to 在逆推序中已定的 lf−ls）。
- **v1 收窄（fail-closed）**：cd 任务**仅允许 FS 搭接**（from/to 任一端），SS/FF/SF 涉及 cd → Validate 拒绝落盘、引擎错误原文明示。理由：①主场景「养护/干燥完成 → 后续工序」天然 FS；②cd 作人工流水（SS+时距）本应按工作日表达（人工作业按工作制走），单位本就不该是 cd；③避免 v1 引入不动点迭代。放开顺序欠账：FF-from → SS-to → （SF 与 to 端不动点组合最后）。
- **lag 恒工作日**：FS from cd 任务的 lag 仍是工作日序号加法（边界已吸附为工作日序号，代数自洽）；不引入 elapsed lag（§7）。

#### 3.2.4 镜像边界

- 换算纯函数落 calendar.ts / calendar.go（`cdToBoundary(startISO, es, cd, cal)` 与 `cdLatestStart(...)`，或收敛为带 ctx 的 effDur 变体——实施时定，两侧同名同签名同期望）；cpm.ts/cpm.go 引入分支；**测试互为镜像**（同批场景同批期望值，沿用 cpm/baseline/deadline/cost 四对先例）：平段吸附、节假日窗口、26/27/28cd 同 ef、逆推双侧校正、手动 cd、混合计划总工期、无 cd 快路径逐位等价。
- gsched.json 演进：新字段 optional 零迁移（第五次，同 baseline/deadline/resources/aoaLayout 先例）；Validate 扩 §3.1 条款，Load/Save 一体生效（agent 与板块同受约束）。

#### 3.2.5 欠账放开矩阵（v1 后逐格放行用，实施前重推）

| cd 任务角色 | FS | SS | FF | SF |
|---|---|---|---|---|
| 作为 to（后继） | ✅ v1 | ✅ 可放（forward 不含 durTo；backward durFrom 为前置） | ⚠ 需不动点（forward 含 durTo） | ⚠ 需不动点 |
| 作为 from（前置） | ✅ v1 | ⚠ 需不动点（backward 含 durFrom） | ✅ 可放（backward 不含 durFrom） | ⚠ 需不动点 |

### 3.3 对既有功能的影响面（逐项裁决）

| 功能 | 裁决 | 说明 |
|---|---|---|
| **成本公式** | work 分配工期换源：`effDur(task)` → `row.ef − row.es`（cpm rows 的等效工作日跨度） | computeCosts 已接收 cpm 结果，零签名变化；wd 任务两值恒等（pin 测试），cd 任务=养护窗口内工作日数——**养护期周末不记工日**，与「元/工日」费率口径自洽。material/cost/fixedCost 不随时长，不受影响 |
| **基线** | 快照行 `dur` 改存 `row.ef − row.es`（等效跨度）；漂移仍在工作日空间 | wd 任务逐位不变；cd 任务日历/位置变化导致等效跨度变化时 durDrift 如实呈现（「等效工作日跨度随位置/日历变化」是诚实口径，非 bug）。基线不加 durCal 字段（拍板项 4） |
| **倒排校核** | **零改动** | targetWorkdays/currentDuration 均 cpm.duration 工作日口径；cd 任务贡献的是吸附后的工作日边界，可行性裁决公式不变 |
| **gschedSummary/摘要卡** | **零改动** | duration/finishDate 保持工作日口径 |
| **mspdi 导出** | cd 任务：`<DurationFormat>8</DurationFormat>` + `Duration` 存日历天口径（序列化待真机钉死）；wd 任务恒 7 不动 | 导出 Start/Finish 本就按 wdToDate 出日历日期，cd 任务两端日期自动吻合 |
| **mspdi 导入** | 读 `DurationFormat`：8 → `durationUnit:'cd'`，duration=自然日数；其余格式维持现状按天折算 | 修复「ed 任务导回被静默当工作日」的单向失真；未知格式回落 7 并标警告（沿用宽松容错） |
| **甘特绘制** | **条形零改动**（自然日轴上 dayNo(es)~dayNo(ef) 自动画出真实日历跨度，周末底纹透条）；工期列/悬停/前锋线取点补单位口径 | 前锋线 frontierWd 的 dur 改用等效跨度（进度×等效工期），检查日换算已按工作日 |
| **拖拽 resize** | cd 任务右缘落点=新 ef（工作日吸附不变），回写 `duration = wdToDate(newEf) − anchor(es)` 自然日数 | move 转手动语义不变（manualStart 恒工作日序号） |
| **PdmView/AoaView** | 节点/图例工期标注补单位后缀「(日历)」；图例「时间单位」注记补一句 | 计算不经视图，仅文案 |
| **agent 回执** | 「总工期 X 天」表述补「（工作日）」；cd 任务的 ops 回执文案「工期 28 天（日历）」 | §3.5 |

### 3.4 UI（推荐：单位后缀切换，不做双值列）

1. **工期列**：InputNumber 后缀单位切换（antd addonAfter 下拉/点击循环：`工作日`/`日历`）；wd 行不显后缀（现状视觉零变化），cd 行显「日历」chip（与模式列手动 chip 同范式）。**不做双值展示**——等效工作日跨度是引擎派生值，摆进表格会诱导用户手改派生值（拍板项 7）。
2. **录入交互**：切换单位**不换算数值**（28 wd → 切 cd 即 28 自然日——与 Project 的 d↔ed 切换同口径，数值=用户声明口径）；分组行/里程碑不出现单位切换入口。
3. **悬停/提示**：cd 任务条 title 补「日历天 28（等效 20 工作日，2026-03-02 ~ 2026-03-30）」——等效跨度透明但不可编辑。
4. 状态栏「共 N 项工作 · 总工期 N 工作日」补齐单位词（现状已是工作日语义，仅文案补词）。

### 3.5 agent 工具接线（零新工具，三件套扩容）

- **ops（internal/schedule/ops.go）**：`patch_task` 增 `DurationUnit *string`（指针三态，'wd'|'cd'，回执「工期口径→日历天」）；`upsert_task` 的 Task 直接带 durationUnit；校验同 §3.1（枚举/里程碑/分组行/cd 上限/FS-only），违规错误原文回传。compact 通道（compact.go）同步维护（v4.122 刀2 先例）。
- **schedule_get**：tasks 带 durationUnit；描述文案补「durationUnit=cd 表示日历天（自然日定时），等效工作日跨度见 cpm 行 ef−es」。
- **schedule_analyze**：**新增「日历天任务」清单**（cdTasks：name/duration(自然日)/等效工作日跨度/锚点日期）——混排计划自检「哪几项按自然日定时、占多少日历窗口」；配套 finding：cd 任务挂了非 FS 搭接（不该发生，闸在 apply，此处防御）。
- **schedule_apply 回执**：patch cd 任务的 changes 文案带「天（日历）」；总工期行「总工期 X 天（工作日）」统一补词。
- **技能条款（builtins.go）**：工期分节增段——「混凝土养护/干燥/成活期等**自然日定时**工作用 `durationUnit:'cd'`，数值=自然日数；定额/合同工期（工作日口径）禁止误标 cd；cd 任务搭接仅 FS；成本仍按等效工作日跨度计（元/工日不因日历天放大）」；汇报口径节补「混排计划先讲日历天任务清单」。

### 3.6 mspdi 互通范围（刀4）

| 方向 | 内容 | 口径 |
|---|---|---|
| 导出 | cd 任务 `DurationFormat=8`；Duration 序列化按真机钉死结果实现（PT{d×24}H 假说优先，XSD 注解+真机样本裁决） | 不导出任务日历（模型无承载，诚实不导） |
| 导入 | DurationFormat=8 → cd；其余格式现状折算 | 宽松容错沿现状；Project 任务日历（TaskCalendarUID）**不导入**（丢弃，现状已丢弃），如需表达走导入后人工改 cd（欠账候选：导入警告清单） |
| 边界 | 往返目标：cd 任务进出不丢口径、数值不变 | **真机池必查**：Project 重开导出文件对 8 的显示与重算、WPS/斑马行为；不真机不发布刀4 |

---

## 4. 分刀方案（每刀独立可发布、可单独 revert）

| 刀 | 内容 | 绑定面 | 测试面 | 可发布性 |
|---|---|---|---|---|
| **刀1 模型+引擎** | types.ts/types.go 增 durationUnit；calendar 换算纯函数对；cpm 日历感知分支+ctx 签名+快路径；cost 换源 ef−es；baseline dur 换源；normalize/Validate 扩展 | **0** | cpm/calendar 镜像 cd 用例 ~14（平段/节假日/逆推校正/快路径等价/FS-only 拒绝）+ cost/baseline 回归 pin | 可发布：数据层先行无 UI 入口，行为对老用户零变化；revert=删字段（文件向后兼容） |
| **刀2 agent 通道** | ops patch_task.durationUnit + upsert 校验；三工具回执口径与 cdTasks 清单；技能条款；compact | **0** | ops 用例 ~6 + 回执 ~3 + skill 用例 ~2 | 可发布：agent 可先对话式表达养护工期跑通全链；revert 独立 |
| **刀3 板块 UI** | 工期列单位切换+chip、悬停等效提示、拖拽 resize 换算、前锋线等效工期、图例/状态栏文案 | **0** | 组件测试 ~8（单位切换不改值/chip 显隐/resize 回写自然日/分组里程碑无入口）+ ?mock=1 走查 | 可发布：纯 UI 层；revert 独立 |
| **刀4 mspdi 互通** | 导出 Format 8 + 导入读 8；真机钉死序列化 | **0** | mspdi.test.ts ~5（roundtrip cd 往返/未知格式回落）+ **真机池** | 可发布前置=真机走查通过；revert 独立 |

刀序理由：引擎先行保证「AI 产建议、引擎裁决」从第一天成立且旧用例逐位不破；agent 第二让养护场景先在对话里验证换算口径；UI 第三（版面拍板项不阻塞数据层）；mspdi 最后绑真机池——外部行为最不可控，放最外缘。

---

## 5. 拍板问题清单（每点给推荐项，拍板后生效）

| # | 决策 | 推荐 | 备选 |
|---|---|---|---|
| 1 | 模型承载 | **`durationUnit?: 'wd'\|'cd'` 单位标记**（单值无双源；mspdi 1:1；零迁移） | durationCal 第二工期字段（双源优先级问题）；任务日历机制 B（多日历体系，明确不做） |
| 2 | 换算边界语义 | **起锚=es 所在工作日，完成边界=首个 ≥ 锚点+自然日数 的工作日（ceil 吸附）**；FS 后继即边界日开工 | floor（完成吸附到前一个工作日——后继会与养护尾日重叠一天，违背「完成才可后续」） |
| 3 | 成本口径 | **work 分配按等效工作日跨度（ef−es）计费**（元/工日自洽，养护周末不记工日） | 按 cd 计费（费率语义变「元/日历天」，与既定「元/工日」拍板冲突） |
| 4 | 基线 dur 口径 | **存等效工作日跨度**，漂移在工作日空间（不加 durCal 字段） | 基线行加自然日字段（双口径漂移并排，读数负担大，需求未现） |
| 5 | cd 任务搭接 | **v1 仅 FS**（fail-closed；SS/FF/SF 按欠账矩阵逐格放行） | v1 全搭接（引入不动点迭代，复杂度前置） |
| 6 | mspdi 方向 | **双向**（导出 Format 8 + 导入读 8，往返不丢口径；真机钉死） | 仅导入（修复失真）不导出（省真机，但 Project 侧养护变工作日失真） |
| 7 | UI 形态 | **工期列单位后缀切换；切单位不换算数值；不做双值列**（等效跨度进悬停提示） | 双值列并排（诱导改派生值）；切单位换算数值（28d→40cd 静默放大，反直觉） |
| 8 | agent analyze | **出「日历天任务」清单**（自然日数/等效跨度/锚点日期）+ 回执口径补词 | 不出清单（混排计划自检缺口，AI 汇报易漏口径） |
| 9 | 时距/里程碑/分组行 | **lag 恒工作日、里程碑与分组行禁 cd**（fail-closed，错误原文明示） | lag 引入 elapsed（LagFormat 8，逆推语义复杂化，无场景驱动） |

---

## 6. 风险与欠账

| 风险 | 对策 |
|---|---|
| **mspdi elapsed 的 Duration 序列化与 Project 往返重算行为未核实**（本设计最大不确定性） | 刀4 绑真机池：导出→真机开→改值→导回→diff；序列化以 XSD+真机样本钉死；不真机不发布刀4 |
| 平段（多 es 映射同 ef）导致逆推 LS 依赖「≤lf 的最大 s」定义，实现走样会出难察的时差错 | 定义钉死在 §3.2.3 + 性质测试 `fwd(ls)≤lf<fwd(ls+1)`；平段/节假日/月末边界用例镜像两侧 |
| 换算性能（cd 任务正逆推触发日历扫描，O(3650) 上限/次） | 快路径使 wd-only 计划零开销；cd 任务通常个位数；逆推校正循环有界（SCAN_LIMIT 同源），测试含 10 年边界 |
| 混合口径的认知负担（用户/AI 混淆 28 与 20） | UI chip+悬停等效提示；回执与技能条款强制口径词；analyze 出 cdTasks 清单先讲口径 |
| 日历编辑联动（改节假日 → cd 等效跨度变） | CPM 每次全量重算无缓存，天然一致；基线漂移如实呈现跨度变化（§3.3，非静默） |
| 单位切换误操作（wd↔cd 数值不换算） | 录入入口 chip 带悬停说明「切单位不换算数值」；patch_task 回执显式报口径变化 |

**欠账池候选（本设计动完后再排）**：SS/FF/SF×cd 的不动点迭代放开（§3.2.5 矩阵逐格）；导入 Project 任务日历转 cd 的迁移助手（含警告清单）；斑马/定额文件导入的口径探测；等效跨度进甘特工期列的灰显双读数（若用户实测仍要）。

## 7. 明确不做（v1 及可见将来，理由随列）

- **任务日历/多日历体系**（机制 B：日历实体、任务↔日历引用、TaskCalendarUID 承载）：为养护场景引入全量日历体系不成比例；单基准日历+工期单位已覆盖主场景；
- **lag 的日历天（elapsed lag / LagFormat 8）**：时距是搭接逻辑参数，恒工作日；无真实场景驱动；
- **周/月/小时等其余工期单位**（DurationFormat 3/5/9/11 及各自 elapsed）：工期制模型只认 wd/cd 两元；
- **cd 任务的资源日历/24h 计费**：成本口径已裁死等效工作日跨度（拍板项 3）；
- **倒排自动压缩/自动改排程**：维持「引擎裁决、AI 建议、用户拍板」既有立场；
- **小数工期**：两口径均整数（cd 上限 3650 整数，wd 现状整数）。

## 8. 参考

- MS Learn：DurationFormat Element（3=分/5=时/7=天/8=日历天…；"Elapsed time counts all time, including non-working time…"；7ed 示例）—— https://learn.microsoft.com/en-us/office-project/xml-data-interchange/durationformat-element
- MS Learn：Task.DurationFormat 枚举（Office 15 对象模型，elapsed 变体全集）—— https://learn.microsoft.com/en-us/previous-versions/office/project-class/gg176883(v=office.15)
- MS Learn：CalendarUID Element（Task 日历引用；TaskCalendarUID/IgnoreResourceCalendars 二手佐证）—— https://learn.microsoft.com/en-us/office-project/xml-data-interchange/calendaruid-element
- MS Support：设置项目的常规工作日和工作时间（基准日历 24 小时制等）—— https://support.microsoft.com/zh-cn/project/set-the-general-working-days-and-times-for-a-project
- 任务日历做养护/干燥的实操口径（二手）：https://tensix.com/how-to-assign-a-task-calendar-in-microsoft-project/ ；https://consultleopard.com/the-difference-between-project-calendar-and-task-calendar-in-ms-project/ ；https://www.projectplan365.com/articles/set-a-calendar-to-a-task/
- 斑马进度（论坛/教程级佐证，字段未核实）：官方学习文档「设置日历及假期」 http://bbs.zpert.com/t/topic/120 ；假期设置教程 https://zhuanlan.zhihu.com/p/433328760 ；特殊工序日历（官方抖音）https://www.douyin.com/shipin/7328265743064320050 ；招标工期差一天帖 http://bbs.zpert.com/t/topic/65063
- 既有基准：docs/gaea-schedule-resource-cost-design-2026-09.md（格式与刀序范式、元/工日拍板先例）；docs/gaea-office-mindmap-base-design-2026-09.md（文档格式基准）；releases/v4.111.0.md（日历口径从自然日修正为工作日的原始决策 + mspdi 务实子集）、releases/v4.122.0.md（资源成本模型、工期进成本公式、镜像测试纪律）
- 现状全量读码（2026-09-06）：frontend/src/schedule/{types,cpm,calendar,cost,baseline,deadline,mspdi,drag,gschedSummary,store,GanttView}.ts(x)；internal/schedule/{types,cpm,calendar,project,ops,analysis}.go；internal/gaea/tool/builtin/schedule_tools.go；internal/gaea/skill/builtins.go；frontend/src/pages/SchedulePage.tsx
