# 进度计划多工程管理（后端持久化多工程）设计方案

> [已归档 2026-09-09] 已实施：v4.139.0（a4777bf0）刀1+刀2 全量落地——每文件一工程+索引+切换器+管理动作+agent 跟随。本文保留为设计依据与 §5 拍板问题/§6 风险的裁定记录。

> 状态：**草案 / 待拍板**（2026-09-06）。承接拍板池欠账「多工程管理」「后端持久化多工程」（releases/v4.110.0 起挂账，v4.124.0 仍在池）。本文只做调研与设计提案，未改任何代码。
> 纪律对齐：AI 产建议引擎裁决（引擎/校验/CPM 口径不因多工程松动）；绑定面 588=权威口径（本提案含绑定面精确预估与诚实漂移分析）；gsched.json schema optional 演进零迁移（本文不动 Project schema）。

---

## 0. 结论速览

- **文件布局：每文件一工程**。`进度计划/` 目录下多个 `.gsched.json`（`isScheduleFilePath` 后缀识别既有），`进度计划/当前计划.gsched.json` 保留为兼容别名；工程注册表+当前指针落 `.gaea/schedule/index.json`（工作区 `.gaea/` 元数据惯例）。单文件多工程数组与 SQLite 工程库均不推荐（§3.1）。
- **绑定面：推荐 +2（588→590）+ 签名级扩展 1 处**。新增 `GaeaScheduleProjects()`（列表+指针视图）与 `GaeaScheduleProjectOpen(rel)`（切指针）；`GaeaScheduleSave` 加可选 rel 参数（1 处签名扩展）。**诚实回答**：drift 检查只比对方法名集合（check-bindings-drift.ps1 用 `gen_bindings -names` 逐名 diff），签名变化**不破 588 口径、drift 仍 PASS**，属「签名级漂移」而非「方法数漂移」，项目有 v4.96 GaeaListDir 签名例外先例（§3.2）。
- **切换语义：当前指针收敛在 Go 侧索引文件**。板块 Load/Save 与 agent 三工具缺省 path 都解析同一指针——「对话改的 = 板块打开的」纪律跨多工程保持；显式 path 永远优先（§3.3）。
- **UI：页头工程切换器**（Select 下拉，含新建/管理入口），默认仅「当前计划」一项，老用户零感知（§3.4）。
- **agent：v1 不加 list_projects 工具、不做 analyze 跨工程对比**。三工具 path 参数已有多工程天然支持；缺省 path 跟随当前指针（§3.5）。办公摘要卡「设为当前计划」从「复制内容」升级为「切指针」（§3.6）。
- 分刀：刀1 切换+列表（588→590）→ 刀2 管理动作（590→592）→ 刀3（agent 对比通道，可不做）。拍板问题 8 项（§5）。

---

## 1. 现状与缺口（读码核实）

### 1.1 单工程制现状

| 层 | 现状 | 文件 |
|---|---|---|
| 路径常量 | `DefaultRelPath = "进度计划/当前计划.gsched.json"`（Go 唯一权威，前端 `SCHEDULE_FILE_PATH` 常量镜像） | internal/schedule/project.go:19；frontend/src/schedule/gschedSummary.ts:16 |
| 引擎 | `Load(path)`/`Save(path, project)` **已参数化**——校验+CPM fail-closed+临时文件原子写+MkdirAll，天然支持任意路径 | internal/schedule/project.go:23-67 |
| 绑定 | `GaeaScheduleLoad()` **真零参数**；`GaeaScheduleSave(projectJSON)` 单参数。两者内部**硬编码 DefaultRelPath**，返回 Path 字段 | internal/app/gaea_schedule_file.go:37-79（注意：internal/app/gaea_schedule.go 是模型保活/预载，与计划文件无关） |
| 门面 | OfficeB 纯委托（gen_bindings 生成物） | internal/app/bindings_office.go:129-130 |
| agent 工具 | schedule_get/apply/analyze 均有**可选 `path` 参数**，`resolveSchedulePath` 缺省回落 DefaultRelPath；apply 有写根 confine | internal/gaea/tool/builtin/schedule_tools.go:34-39 |
| 前端 store | 单工程 zustand + localStorage persist（`gaea.schedule.v1`，缓存+迁移源）；水合/防抖 800ms 自动保存/15s 轻扫回读，`lastSyncedRaw` 单文件基准 | frontend/src/schedule/store.ts:348-410 |
| 页面 | 页头：标题+工程名 Input（改的是文件内 `project.name` 字段，非文件名）+开工日期+视图切换；底部状态栏同步指示器 title 硬编码当前路径；AoaView 只吃 graph/tasks props 与路径无关 | frontend/src/pages/SchedulePage.tsx |
| 办公摘要卡 | `isCurrentPlan` 按 `endsWith(SCHEDULE_FILE_PATH)` 判定；当前计划才有「打开/周报」；非当前计划有「设为当前计划」=**复制内容覆盖当前计划文件** | frontend/src/gaea/components/ScheduleFileCard.tsx:43-76 |
| mock | `mockScheduleFile` 单文件会话内存 + `window.__mockScheduleFile` 走查钩子 | frontend/src/gaea/lib/mock/office.ts:33-48 |

### 1.2 绑定签名机制（本设计的口径核心，逐层核实）

1. **588 的口径 = 导出绑定方法名集合**。`go run ./scripts/gen_bindings -names` 当前输出 588 个方法名；bindingNames.ts 恰好 588 条。
2. **drift 检查只比名字**。scripts/check-bindings-drift.ps1：`gen_bindings -names` 输出 ↔ bindingNames.ts 数组逐名 Compare-Object，**不含签名**。CI 门禁 ci.yml「Check bindings drift」即此。
3. **签名变化的真实波及面**（诚实清单）：
   - wails 生成物 `frontend/wailsjs/go/app/OfficeB.d.ts`：`GaeaScheduleLoad():Promise<...>` 会变为 `GaeaScheduleLoad(arg1?:string)` 形态——机械重生成，非手工维护；
   - bridge.ts 的 AppBindings 接口声明 + LegacySurfaceNames 映射（名字不变，仅类型签名行）——tsc 门禁覆盖；
   - mock/office.ts 的对应 mock 方法签名同步。
   - **结论：给既有绑定加可选参数 ≠ 绑定面漂移**（588 与 drift PASS 均不破），但属于**签名级扩展**，须同步上述三处。项目有先例：v4.96 GaeaListDir 返回值签名例外（`[]DirEntry` → `([]DirEntry, error)`），登记口径为「方法名/绑定数 581 不变」。
4. **wails 缺参语义**：JS 侧按位置传参，少传参数 Go 侧反射调用按零值补位（`Load()` 不传 → `rel==""` → 走缺省分支）。此语义需真机走查钉死（§6 风险 1）。
5. **新绑定成本**：gen_bindings 扫描 App 导出方法自动归板块（`GaeaSchedule*` 前缀默认落 office，explicitOverrides 已登记先例），新增方法须重生成 bindingNames.ts + bridge 分类同步——+N 个方法即 588→588+N，这正是「零新绑定强偏好」要控的量。

### 1.3 缺口

- 板块与绑定的 Load/Save 目标固定，无法打开第二个计划文件（办公板块虽能预览任意 .gsched.json，板块不可编辑它）。
- 无工程列表/切换/新建/归档的任何通道（绑定、工具、UI 三层皆无）。
- agent 显式 path 已可读写任意 .gsched.json（**多工程的 agent 通道其实已存在**），缺的只是「缺省跟随什么」与「发现工程列表」。
- 「设为当前计划」是复制语义——多工程下会破坏源文件独立性（复制后两文件同源漂移）。

---

## 2. 参照

### 2.1 项目内先例（主参照）

| 先例 | 形态 | 对本设计的启示 |
|---|---|---|
| **造价数据库多项目**（internal/gaea/costproject + internal/app/gaea_cost_projects.go） | SQLite（Hephaestus.db）承载 Project/Item/Version；绑定 GaeaCostProjectSave/List/Get/Delete + Estimate*（+8）；版本=不可变快照 | **DB 制的反例教材**：测算项目是纯应用内资产（用户不需要拿文件去别处），计划文件则是**用户可见的共享资产**（办公预览/摘要卡/mspdi 互通/外部工具直开）——DB 化会牺牲可见性。但其「列表视图带统计（ItemCount/Total/VersionCount）」的 ProjectSummary 形态值得抄：GaeaScheduleProjects 返回值带 duration/taskCount/ok 摘要 |
| **双空间**（internal/app/gaea_spaces.go） | 「当前空间」指针持久化为配置键（session.space → 用户配置文件）；GaeaSpaceList/Active/Activate +3 绑定 | 「当前指针放哪」的直接先例。但配置键是**用户级**（跨工作区全局一份）——多工作区下 A 区指针指向 B 区文件语义错；计划资产随工作区走，指针应随工作区（索引文件）而非用户配置 |
| **记忆中枢**（internal/gaea/memory） | space 谓词读端隔离（ListInSpace）；file backend 项目域目录（projects/&lt;slug&gt;/memory） | 「按域分区、读端过滤」的分域思路；目录分区先例 |
| **工作区 .gaea/ 元数据惯例** | .gaea/exports、.gaea/play/exports、.gaea/skills、.gaea/uploads | 索引落 `.gaea/schedule/index.json` 合惯例（与用户资产目录「进度计划/」分离，防误删） |
| **办公思维导图/多维表设计**（docs/gaea-office-mindmap-base-design-2026-09.md，格式基准） | markdown 大纲=权威格式（用户可直读、diff 白得）；视图配置 sidecar；「视图不是新格式」 | 多工程同理：**gsched.json 文件=权威，索引只是注册表**；不发明容器格式 |

### 2.2 外部工具浅调研

- **MS Project**：单 .mpp=单工程；多工程=主工程插入子工程（consolidated）+资源共享池——文件间链接脆弱，是业界公认运维痛点。反证「每文件一工程 + 松耦合列表」比「容器+内嵌子工程」健康。
- **Primavera P6**：数据库 EPS 树承载多工程（企业协同形态）——与造价 DB 制同款取舍，gaea 单机桌面形态不取。
- **斑马进度**（docs/research-2026-09-06/Glodon-Zebra-斑马进度.md）：桌面单机文件制，多项目靠看板人工汇总；其空档恰是「agent 长驻盯多工程」——本设计保持文件制即保住了这条差异化通道。
- 结论：**文件制多工程（每文件一工程+轻索引）** 与 gaea「文件是板块与 agent 共享资产」哲学一致，外部两派（文件容器派/数据库派）的教训都不指向第三条路。

---

## 3. 设计提案

### 3.1 文件布局：每文件一工程 + 索引（正面回答问题 1）

**推荐：A. 目录多文件 + 索引**

```
进度计划/
  当前计划.gsched.json        ← 兼容别名：默认工程的落点（老用户零感知）
  办公楼二期.gsched.json      ← 每文件一工程（新工程文件名由 Go 生成安全 slug）
  ...
.gaea/schedule/index.json    ← 工程注册表 + 当前指针（非权威，可重建，见下）
```

索引形态（示意，字段以实施定稿）：

```json
{
  "version": 1,
  "current": "进度计划/当前计划.gsched.json",
  "projects": [
    { "rel": "进度计划/当前计划.gsched.json", "name": "当前计划", "archived": false, "updatedAt": "..." }
  ]
}
```

理由（含 agent 契合度）：

1. **agent path 参数天然契合**：三工具的 `path` 就是文件粒度——每文件一工程时「path=工程」，零参数扩容；单文件多工程数组则 path 无法定位工程，必须给三工具加 project_id 维度（三份 schema+compact 通道全动），代价根本不成比例。
2. **schema 零迁移**：Project 结构一字不动（纪律：schema optional 演进零迁移）；数组方案要在 Project 之上发明容器 schema 并改 Validate。
3. **用户可见性**：办公板块/文件管理器/mspdi 往来/git diff 全部按文件工作，损坏只影响单工程；容器方案把所有工程锁进一个文件，冲突面=全部工程。
4. **引擎零改动**：Load/Save/Validate/ComputeCpm 全部已按单文件工作，多文件只是调用次数变多。

**不推荐 B：单文件多工程数组**——path 契合崩坏（理由 1）、新容器 schema（理由 2）、外部工具不可直读单工程（理由 3），三头全损，仅省一个索引文件。
**不推荐 C：SQLite 工程库（造价先例）**——计划是用户可见共享资产（§2.1），DB 化牺牲办公预览/mspdi/git 全链路；且 agent 的写根 confine 按 path 工作天然适配文件制。

**索引的定位（诚实边界）**：索引是**缓存与指针，不是权威**。工程事实真相=目录扫描（`进度计划/*.gsched.json`）；索引缺失/损坏/漂移（用户手删文件）时，Go 侧 Load 索引失败即扫描目录重建（文件内 project.name 顺带回填），宁重建勿报错。

### 3.2 绑定面：+2 与签名扩展的诚实账（正面回答问题 2）

**候选方案对比：**

| 方案 | 内容 | 绑定面 | 评估 |
|---|---|---|---|
| **A（推荐）** | 新增 `GaeaScheduleProjects()` / `GaeaScheduleProjectOpen(rel)`；`GaeaScheduleSave(projectJSON, rel?)` 加可选 rel；`GaeaScheduleLoad()` 签名不动（内部改读索引 current） | **588→590（+2）**，签名扩展 1 处 | 语义最干净：Save 显式带目标 path，规避切换竞态（§3.3）；列表由 Go 算摘要（Load+Analyze 逐文件，工程数量级=个位数，开销可接受，结果缓存进索引） |
| B | 同 A 但 Save 也不动（读写全收敛到索引 current） | 588→590（+2），签名扩展 0 处 | 零签名漂移，但防抖保存与切指针存在时序竞态（切换瞬间旧内容可能写进新文件），store 层要加切换冲刷/丢弃逻辑——省 1 处签名换一类状态机复杂度 |
| C（零新增） | 前端用既有 GaeaListDir 列 .gsched.json + parseSchedSummary 纯函数出摘要 + GaeaWriteFile 写索引；Load/Save 完全不动（认 current） | **588 不变** | 可行（前端 parseSchedSummary 已能出全套摘要），但①索引写走通用 WriteFile 无原子写；②工程注册表一致性责任全压前端，重命名/归档/冲突演化必撞墙；③「切换」实质是前端写索引文件，Go 无从校验 rel 合法性（可写任意路径） |
| D | 不动 Load/Save，全走新绑定：+List/+Open/+SaveAs/+Create… | 588→591+ | 每多一个通道就多一份「两套保存路径」的口径分叉，违背轻量绑定偏好 |

**推荐 A**，理由收束：

- 588 的权威口径是**方法名集合**，A 方案 +2 一目了然可审计；签名扩展 1 处有 GaeaListDir 先例背书，且 drift PASS 不受影响（§1.2）。
- Save 加可选 rel 是**防竞态的结构性需要**（板块保存永远显式带上装载时的工程文件，切换只改指针不动保存目标），不是为绕开新绑定的技巧。
- C 方案作为拍板备选保留（若「绑定面 588 不可动」被裁定为硬红线，C 能交付 v1 全部体验，代价见评估列）。

### 3.3 切换语义：「当前工程」指针与 agent 缺省 path（正面回答问题 3）

**指针存哪——三候选裁定：**

| 候选 | 裁定 | 理由 |
|---|---|---|
| **索引文件**（`.gaea/schedule/index.json` 的 current 字段） | **推荐** | Go/前端/agent 三方同源（都经 Go 读）；随工作区走，多工作区各一个指针天然正确；重启保持 |
| 前端 localStorage | 排除作为唯一真相 | agent 缺省 path 在 Go 侧解析，前端指针 Go 看不见 → agent 缺省不跟随切换，破坏「对话改的=板块打开的」核心纪律；只可作 UI 偏好缓存 |
| Go 侧内存态 / 用户配置键 | 排除 | 内存态重启丢；用户配置键跨工作区全局一份（双空间先例的局限），A 区指针指向 B 区文件语义错 |

**切换语义：**

- `GaeaScheduleProjectOpen(rel)` 只做两件事：校验 rel 合法（在索引注册表内或存在且后缀合法）→ 原子写索引 current。**不搬运数据、不改文件**。
- 板块切换流程：Open → 重水合（走既有 initScheduleSync 水合分支）→ 保存目标显式绑定新 rel（方案 A 的 Save(rel)）。切换前若有 dirty 防抖在途，先冲刷保存旧工程再切（store 层逻辑，与现有 lastSyncedRaw 基准同范式）。
- **agent 缺省 path 跟随当前指针**：`resolveSchedulePath` 缺省分支从「常量 DefaultRelPath」改为「读索引 current，索引不可用回落 DefaultRelPath」。显式 path 永远优先。这样「对话改的 = 板块打开的」在多工程下继续成立；工程上下文一致性由指针的文件真相保证（对话中用户在板块切工程，agent 下一次缺省调用即落新工程——这是特性不是缺陷，与「板块=当前工程工作台」的语义一致）。

### 3.4 UI（正面回答问题 4）

- **工程切换器**：页头「进度计划」标题右侧 antd Select——列未归档工程（显示 project.name，当前项高亮），底部「+ 新建工程」「管理工程…」两个入口。与现有页头元素（工程名 Input/开工日期/视图 Segmented）并排，风格沿用。
- **新建**：弹窗输入工程名 → Go 生成安全 slug 文件名（同名加序号；**此后文件名不再改**——agent 显式 path 引用稳定性优先）→ 空 Project 落盘（走 schedule.Save 全套校验）→ 索引登记 → 自动切为当前。
- **重命名**：下拉「管理工程…」→ 列表（名称/总工期/任务数/更新时间）→ 行内改名=改**文件内 project.name**（既有 renameProject 通道，落盘自动回写）而非文件改名——文件名机械化的理由同上。
- **归档/删除**：管理列表行内「归档」（索引标 archived，文件保留原位，切换器不再显示；可反归档）与「删除」（Popconfirm 二次确认 + 文案注明不可恢复；Go 绑定物理删文件+索引摘除）。删除当前工程时先切到剩余第一个工程。
- **兼容（老用户零感知）**：索引不存在 → 首次进入板块时扫描 `进度计划/*.gsched.json` 重建（通常只有「当前计划」一项，current=DefaultRelPath）；切换器只有一项时收窄为纯显示（不占交互成本）。SCHEDULE_FILE_PATH 常量降级为**缺省值**，板块状态栏 title、办公卡 isCurrentPlan 改读 store 的 currentPath（缺省仍为常量值）。
- **状态栏**：同步指示器 hover 从硬编码路径改为当前工程 rel；多工程下增加工程名段（可选，刀2）。

### 3.5 agent（正面回答问题 5）

- **三工具不加参数**：path 已有多工程天然支持（§3.1 理由 1）。仅 Description 文案微调（「缺省=当前计划」→「缺省=当前工程（可在进度计划板块切换）」），compact 通道同步。
- **list_projects 工具：v1 不做**。理由：① agent 发现工程的通道已存在——读 `.gaea/schedule/index.json`（readfile）或 listdir `进度计划/`，都是既有通用工具；② 工具面零增是纪律偏好，专用工具只有在「通用通道无法表达」时才值得；③ v1 先看对话实测：若模型找不到工程列表成为高频失败模式，刀3 再拍板。
- **analyze 扩多工程对比：v1 不做**。理由：跨工程对比没有确定性口径（不同工程任务集/日历不同，「哪个工程风险大」是 AI 叙事层判断不是 CPM 引擎裁决——强行做会破「AI 产建议引擎裁决」的分工）；引擎保持单工程纯函数。agent 若被要求对比，可对多个 path 依次 schedule_get 后自行综合，通道已通。
- **办公摘要卡「设为当前计划」联动**：语义从「复制内容覆盖当前计划文件」升级为「`GaeaScheduleProjectOpen(rel)` 切指针 + 通知板块重水合」（notifyScheduleFileChanged 既有事件复用）。理由：复制语义在多工程下制造两份同源文件，漂移后无从收拾；切换语义保持「一工程一文件」。原复制行为不再需要；「打开」按钮从仅当前计划扩展为任意计划（点击=切换+跳板块，绑定已有）。升级后 notCurrent 文案同步改。

### 3.6 数据流小结（改什么/不改什么）

**不改**：internal/schedule 全包（Load/Save/Validate/CPM/ops/基线/倒排）；三工具 schema 与执行主体；gsched.json schema；localStorage persist 结构（仍只缓存当前工程）。
**改**：gaea_schedule_file.go（Load 读索引、Save 加 rel、两个新绑定）；schedule_tools.go resolveSchedulePath 缺省分支；store.ts（currentPath 状态、切换重水合、SCHEDULE_FILE_PATH 降级缺省）；SchedulePage 页头切换器；ScheduleFileCard 联动语义；mock/office.ts（多文件会话内存 Map）；bridge.ts 类型；wailsjs 重生成。

---

## 4. 分刀方案（每刀独立可发布，绑定面精确到个数）

| 刀 | 内容 | 绑定面 | 测试重心 | 体量 |
|---|---|---|---|---|
| **刀1（本设计主体）** | 索引读写+扫描重建；`GaeaScheduleProjects`/`GaeaScheduleProjectOpen` 新绑定；Save 加可选 rel；agent 缺省 path 跟随指针；前端切换器+重水合；老用户零感知兼容 | **588→590（+2）**，签名扩展 1 处（Save），drift PASS@590 | Go：索引重建/指针切换/Save rel/缺省解析 6~8 例；TS：切换重水合/竞态冲刷/索引降级 6~8 例；mock 走查：切换→编辑→agent 缺省写→回读全链 | 中刀 |
| **刀2（管理动作）** | 新建（slug 落盘+登记+自动切换）；重命名（文件内 name）；归档/删除绑定 `GaeaScheduleProjectArchive(rel, archived)`、`GaeaScheduleProjectDelete(rel)`；管理列表 UI；状态栏工程名 | **590→592（+2）**，drift PASS@592 | Go：slug 冲突/删当前工程级联切指针/归档过滤 5~6 例；TS：管理列表 5~6 例；走查：新建→改名→归档→删除 | 小刀 |
| **刀3（可不做）** | agent list_projects 工具；analyze 跨工程对比叙事；办公摘要卡多工程对比视图 | 若做：+1 工具（非绑定，工具数另计）或 +1 绑定 | 另案 | 待观察后拍板 |

刀1 发布后即可独立满足拍板池「多工程管理」的最小完整闭环（列表/切换/agent 跟随/办公卡联动）；刀2 补管理动作；刀3 视实测。

---

## 5. 拍板问题清单（每点带推荐）

1. **文件布局**：A 目录多文件+索引（推荐）vs B 单文件数组 vs C SQLite。→ 推荐 A（§3.1）。
2. **索引落点**：`.gaea/schedule/index.json`（推荐，与用户资产目录分离防误删）vs `进度计划/index.json`（目录自包含但易被当垃圾清掉）。→ 推荐 `.gaea/schedule/index.json`。
3. **当前指针**：索引文件（推荐）vs localStorage vs 用户配置键。→ 索引文件（§3.3）。
4. **绑定面方案**：A +2+签名扩展 1 处（推荐）vs B +2 零签名扩展（吃竞态复杂度）vs C 零新增（一致性责任压前端）vs D +4 全新绑定。→ 推荐 A；若「588 不可动」为硬红线则 C（§3.2）。
5. **agent 缺省 path**：跟随当前指针（推荐）vs 固定 DefaultRelPath（板块切了 agent 不知道，破坏「对话改的=板块打开的」）。→ 跟随（§3.3）。
6. **归档语义**：索引标 archived 文件不动（推荐，文件是用户资产，agent 显式 path 引用不失效）vs 移入 `进度计划/归档/` 子目录（路径变，外部引用断）vs 仅物理删除。→ 标 archived；删除独立动作+二次确认（§3.4）。
7. **「设为当前计划」语义**：切指针（推荐）vs 保留复制。→ 切指针，复制废弃（§3.5）。
8. **工程身份**：列表显示文件内 project.name、文件名 slug 化后不改（推荐，agent path 引用稳定）vs 文件名即身份（改名=改文件名，外部引用全断）。→ 前者（§3.4）。

---

## 6. 风险与欠账

1. **wails 缺参=零值语义未真机验证**（方案 A 的 Save 可选 rel 依赖它）：mock 层不走 wails 验证不了；刀1 走查必须含真机冒烟（老调用形态 `Save(json)` 不传 rel）。兜底：若语义不符，退方案 B（Save 不动）只改 Go 内部分支，绑定面账不变。
2. **索引与文件系统漂移**（用户手动增删文件）：扫描重建兜底（§3.1），但「手动放进来的文件不自动出现在切换器」还是「每次全扫」需刀1 定稿（推荐：Load 索引时轻扫目录 diff，成本=一次 ReadDir）。
3. **切换竞态**：防抖保存在途时切工程——store 层冲刷逻辑是刀1 的测试重点（§4 刀1）；方案 A 的 Save(rel) 已从结构上消解「写错文件」，剩余竞态只是 UI 态。
4. **agent 显式 path 写未登记文件**（如 `进度计划/临时.gsched.json`）：Save 成功但不在注册表 → 下次扫描 diff 时收编（风险 2 的副产物，诚实呈现不静默吞）。
5. **localStorage 缓存只存当前工程**：切换后旧缓存被覆盖——缓存本就定位「离线兜底+迁移源」，多工程下降级为「当前工程缓存」，文档口径同步。
6. **欠账转移**：mspdi 导入「导入为新工程」、基线跨工程复制、多工程资源池——不在本设计，刀2 后入候选池。

## 7. 明确不做

- 单文件多工程数组容器（§3.1 方案 B，三头全损）。
- SQLite 工程库（§3.1 方案 C；造价先例不适用于用户可见资产）。
- 跨工程资源池 / 工程间依赖 / 多工程汇总报表 / P6 EPS 式企业结构（单机形态无关）。
- agent list_projects 工具与 analyze 跨工程对比（v1；§3.5，刀3 视实测）。
- 工程级权限/多人协作/云同步。
- gsched.json schema 任何变更（纪律：optional 演进零迁移）。

## 8. 参考

- releases/v4.110.0.md:64、v4.111.0.md:74（「后端持久化/多工程管理」首次挂账）；v4.113.0.md:54、v4.115.0.md:28、v4.124.0.md:55（拍板池延续）。
- docs/gaea-office-mindmap-base-design-2026-09.md（格式基准：文件=权威、视图配置 sidecar、纯函数解析）。
- docs/gaea-schedule-resource-cost-design-2026-09.md、docs/gaea-schedule-aoa-manual-layout-design-2026-09.md（绑定面零增先例与文档口径先例）。
- internal/schedule/project.go；internal/app/gaea_schedule_file.go；internal/app/bindings_office.go；scripts/gen_bindings/main.go；scripts/check-bindings-drift.ps1；.github/workflows/ci.yml（drift 门禁）。
- internal/gaea/tool/builtin/schedule_tools.go（三工具 path 参数）。
- internal/gaea/costproject/（造价多项目 DB 制先例）；internal/app/gaea_spaces.go（当前指针配置键先例）；internal/gaea/memory/store.go（分域读谓词先例）。
- frontend/src/schedule/{store.ts, api.ts, gschedSummary.ts}；frontend/src/pages/SchedulePage.tsx；frontend/src/gaea/components/ScheduleFileCard.tsx；frontend/src/gaea/lib/mock/office.ts。
- docs/research-2026-09-06/Glodon-Zebra-斑马进度.md（外部浅调研主材；MS Project/P6 为通识口径，未逐条核证）。
