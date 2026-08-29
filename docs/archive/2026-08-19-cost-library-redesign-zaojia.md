# 成本库重设计：参照 zaojia-database 蒸馏，2026-08-19

> 目标：用户在 GitHub 检索到 [BruceLee1024/zaojia-database](https://github.com/BruceLee1024/zaojia-database)
> （造价数据库｜本地工程造价资料库），要求参照蒸馏、重新设计 gaea 成本库，并把成本库
> 从「记忆中枢」二级分类提升为一级导航。

> 状态（2026-08-19 定稿）：已按「综合单价=一级、人材机=二级组成」落地并构建
> 桌面端。定稿决策与实现细节见
> [market-research-2026-08-cost-architecture-zonghe-danjia.md](market-research-2026-08-cost-architecture-zonghe-danjia.md)；
> 面板重设计（库规模/人材机构成/数据健康）见
> [CostLibraryPage.tsx](../frontend/src/pages/CostLibraryPage.tsx)。

## 0. 一句话结论

zaojia-database 的本质是**「个人造价数据与经验沉淀管理平台」**：定额库 / 清单库 /
材料设备库 / 项目报价 / 造价参考 / 复盘笔记围绕「采集 → 入库 → 编制 → 版本留痕 →
复用」组织，坚持本地优先、导入先预览确认、每个数字可追溯、AI 可降级。gaea 成本库
已经具备它的核心数据底座（条目 + 多级分类 + 价格历史 + 导入 + 价格源），差距在
**产品形态**：它只是记忆中枢里的一个二级 Tab，没有一级入口、没有工作台式的概览与
健康度视图。本次重设计 = ① 提升为一级导航「成本库」；② 以「概览 + 分库」工作台形态
重新组织现有能力；③ 用 zaojia-database 的「待补单价 / 完整度 / 最近工作」做概览骨架。

## 1. zaojia-database 蒸馏

### 1.1 项目定位

面向个人造价从业者的通用数据与经验沉淀平台，不限定工程类型。业务数据默认存本地
（IndexedDB / 授权本地文件夹），AI Key 仅存 localStorage。产品由「产品介绍落地页 +
工作台」组成，工作台模块：

| 模块 | 作用 | 蒸馏点 |
|---|---|---|
| 我的概览 | 继续最近工作，查看待补单价、待保存版本、资料完整度 | 工作台首页 = 待办 + 健康度 |
| 导入资料 | 统一导入定额/通用清单/项目清单/材料/设备 | 先识别表头/字段映射，预览确认后才写入 |
| 我的定额库/清单库/材料库/设备库 | 分库维护可复用资源 | 资源主数据 + 价格快照分离 |
| 我的项目 / 工程量清单 | 项目编制、报价版本留痕、审查、恢复 | 工作稿 → 不可变快照 → 对比/恢复 |
| 造价参考 | 案例极值/分位数/中位数/均值 | 沉淀案例再反哺报价 |
| AI 助手 / 复盘笔记 | 查造价、推荐定额、检查缺单价；经验沉淀 | AI 只做识别/检查/检索，可降级 |
| 数据与备份 | AI 配置、本地存储、JSON/ZIP 备份 | 本地优先，资料归使用者 |

### 1.2 产品设计五原则（照搬到 gaea 成本库）

1. **本地优先，资料由使用者掌控**：gaea 本就是桌面应用（Hephaestus.db），天然满足；
   重设计继续强调数据在本地，不默认把业务资料交给远端。
2. **先看得懂，再写进去**：导入 Excel/CSV/PDF/图片报价单 → 识别表头与数据区 → 展示
   预览 → 人工确认后才落库。gaea 的 CostImportModal 已实现，重设计把它做成一级页的
   一等公民入口。
3. **每个数字都应能追溯**：来源、导入记录、价格快照、更新时间共同回答「这个数从哪里来、
   何时生效、是否可信」。gaea 已有 source + cost_price_history + pricefeed，重设计把
   「来源」和「价格历史」在概览与条目视图里显性化。
4. **沉淀可复用的判断，而不只是堆积数据**：单价条目与分类、标签、状态（现行/草稿/已归档）
   关联，一次询价/测算的单价回流成本库供下次复用（办公侧 cost_save / CostLibraryPanel
   已打通，本次保留）。
5. **专业逻辑优先于炫技**：AI 只在导入识别、缺单价检查、检索精排环节提供可降级辅助；
   抓价结果必须人工确认发布。保持现有「无确认不落库」边界。

### 1.3 数据模型蒸馏（对照现状）

| zaojia-database | gaea 现状 | 差距 | 处理 |
|---|---|---|---|
| 定额/清单/材料/设备分库 | 单表 cost_entries + 多级分类树 | 无硬分库（靠分类路径） | 保留分类树，不拆表（个人量级） |
| 材料/设备价格 = 不可变快照（ex_factory / delivered / installed_composite 三口径 + 有效期 + 附件） | cost_price_history 快照历史 + pricefeed | 无三口径、无有效期、无附件 | P1：条目详情补口径/有效期字段 |
| 价格三要素：规格 + 地区 + 时间 | spec + updatedAt，无地区/期数 | 无地区维度 | P1：补 region/price_date |
| 缺单价标记 + 数据质量状态 | status（现行/草稿/已归档）+ price<=0 隐式 | 无显式健康度 | 本次：概览统计「待补单价 = price<=0 或草稿」 |
| 导入记录 / 来源行号可溯 | source 字段 + 导入入口 | 无原始行号 | P2：导入时记录原始行号 |

## 2. gaea 成本库现状盘点（代码事实）

- 存储：`Hephaestus.db` `cost_entries`（SchemaV2：name/title/category/category_path/
  unit/price/spec/source/tags/status/body/时间戳）+ `cost_categories` 分类树 +
  `cost_price_history` + `pricefeed`（Source/FetchRecord/History）。
- 后端：`internal/gaea/cost` Store（Save 同名 UPSERT / Get / Delete / Search，LIKE +
  BM25 检索）；App 绑定 `GaeaCostList/Search/Get/Save/Delete/Categories/...` + 导入
  （xlsx/csv/PDF/图片 + AI/视觉解析）+ 价格源抓取 + 价格历史 + 对比。
- 前端：记忆中枢 `MemoryHubPage` 的 cost Tab（`CostLibrary`：成本条目 / 价格源 /
  价格源仓库 三个子视图，含多级分类树、列表/表格双视图、批量操作、导入 Modal、
  价格历史、对比）；办公右侧 `CostLibraryPanel`（窄面板 + 一键插入上下文）。
- 概览：`MemoryHubOverview` 已含 `costCount`。

### 差距（vs zaojia-database 形态）

1. 无一级入口：成本库藏在「记忆中枢」二级分类里，功能再强也不像独立工作台。
2. 无概览/待办：没有「待补单价、草稿数、最近更新、完整度」的聚合视图。
3. 无「导入资料」一等入口：导入能力在条目工具栏里，靠文件选择，无工作台卡片直达。
4. 价格口径/地区/有效期缺失（P1，沿用现有调研路线）。

## 3. 重设计决策

### 3.1 一级导航：成本库独立成板块

- 新增一级板块 `cost`（Page = `CostLibraryPage`，icon = AccountBook，菜单序 5，紧跟
  「办公」之后），与 chat/novel/imagegen/gaea 并列；编程及后续板块顺延。**对外
  名称定为「造价数据库」**（用户命名，对齐 zaojia-database）；内部 id/表名保持
  `cost`/`cost_entries` 不变，避免大范围迁移。
- 记忆中枢移除 `cost` 二级分类（避免双入口；与「knowledge 并入记忆中枢不再单列」的
  前例是同一收敛逻辑，方向相反而已）。
- 办公右侧 `CostLibraryPanel` 保留：它是「办公任务内引用成本库」的通道；管理/编辑
  集中在新的成本库一级页，面板只做浏览 + 插入。

### 3.2 页面结构：概览 + 分库工作台

```
成本库（CostLibraryPage）
├─ 顶栏：成本库标识 + 一句话说明 + 快捷动作（导入文件 / 新建条目）
├─ 模块 Tab：概览 | 成本条目 | 价格源 | 价格仓库
└─ 概览模块（对标 zaojia「我的概览」）
   ├─ 统计卡：条目总数 / 待补单价 / 分类数 / 价格源数
   ├─ 最近更新：最近 6 条（继续最近工作）
   ├─ 待补单价清单：price<=0 或草稿条目（缺单价标记）
   └─ 快捷入口：导入资料 / 新建条目 / 价格源 / 价格仓库
```

### 3.3 保持的边界（沿用现有路线，不扩权）

- 不做企业级组价/定额引擎、不做云端团队共享、不做无人确认自动写入。
- 抓价结果必须人工确认发布；导入必须预览确认后落库。
- 检索继续按数据量分级（<5k LIKE/BM25 → 5k–50k FTS/BM25 → 大库 + 本地 rerank），
  本次不引入新检索基建。

## 4. 本次落地清单

- 后端 `internal/app/board/builtins.go`：新增 `cost` 板块 manifest（Page=CostLibraryPage、
  Bindings=CostB、菜单序 5），`CanonicalIDs` 加入 cost，记忆中枢 Nav 移除 cost。
- 前端 `frontend/src/boards/manifests.ts`：新增 `cost` canonical 板块 + COST_NAV，
  移除 MEMORYHUB_NAV 的 cost，图标注册表补 AccountBookOutlined。
- 新页面 `frontend/src/pages/CostLibraryPage.tsx`：概览 + 成本条目（复用
  CostLibraryView）/ 价格源（PriceSourcesPanel）/ 价格仓库（PriceSourcesRepository）。
- 注册：`main.tsx` registerPage + MainLayout legacy fallback；MemoryHubPage 移除 cost
  分类与渲染。
- 测试：Go board_manifest_test / manifest_test、前端 manifests.test / launcher.test
  对齐新菜单序与 fixture。

## 5. 后续（P1/P2，不进本次范围）

- ✅ 已落地（2026-08-19 第二轮蒸馏）：条目「价格口径（出厂/到场/安装综合）+ 有效期 +
  地区/期数」字段与录入/展示 UI（SchemaV9 + cost.Store + CostEntryModal/CostLibraryView）。
- ✅ 已落地：`source_row` 导入原始行号——Excel/CSV 横向表导入自动按物理行号记录
  （表头偏移计算），随条目保存、展示，office agent 输出可见；纵向参数表/AI 解析
  无法确定物理行号时记 0（不臆造）。
- ✅ 已落地（2026-08-19 第三轮蒸馏）：「项目/测算 → 成本库」沉淀闭环（对标 zaojia 的
  我的项目/工程量清单/版本留痕），见 §5.2；**前端工作台形态已在 §7 收敛决策中移除**，
  测算/沉淀能力保留为后端 + office agent 工具（cost_save/cost_indicators）。
- ✅ 已落地（2026-08-19 第四轮蒸馏）：造价参考（案例分位数/中位数/均值）+ 复盘笔记
  （结论/边界/风险/证据/可信度/复核状态），见 §5.3；**前端模块已在 §7 收敛决策中移除**，
  造价参考改为 office agent 工具（cost_indicators），复盘笔记并入知识库方向（不新开存储）。

### 5.1 第二轮蒸馏落地明细（2026-08-19）

参照 zaojia「价格三要素（规格+地区+时间）+ 三口径 + 有效期」与「每个数字可追溯」：

- SchemaV9：`cost_entries` 增加 `region`（地区）、`price_date`（价格时间/期数）、
  `price_type`（出厂价/到场价/安装综合价）、`valid_until`（有效期至）、`source_row`
  （导入原始行号）；`cost_price_history` 同步记录发布时的地区与口径。
- cost.Store Save/Get/Search 全链路支持新字段；检索 haystack 与 BM25 文本加入
  地区/口径/期数（"成都 螺纹钢" 也能召回）。
- 价格源抓取写回（GaeaPriceFetchApply）自动带出：地区 = 价格源配置的 area，
  期数 = 抓取记录 period；历史快照同时记录地区与口径——"哪个地区、什么口径、
  哪一期"可完整回看。
- 前端：CostEntryModal 新增 地区/价格时间·期数/价格口径（Select）/有效期至/原始行号；
  CostLibraryView 列表项与表格新增 地区·期数/口径 展示；office agent 的 cost_search
  输出表格增加 地区/期数/口径 三列，测算引用时可交代依据。

### 5.2 测算项目与沉淀闭环（第三轮蒸馏，2026-08-19）

对标 zaojia「我的项目 / 工程量清单 / 版本留痕」+「沉淀即调用」：

- SchemaV10：`cost_projects`（项目容器：类型/规模/工艺/状态）、`cost_estimate_items`
  （明细行：数量×单价自动算金额、可引用成本条目 name）、`cost_estimate_versions`
  （不可变版本快照：明细 JSON + 合计）。
- 后端 `internal/gaea/costproject` Store：项目/明细/版本 CRUD + 级联删除；项目 id
  用「纳秒时间戳 + 进程内序号」防碰撞。
- App 绑定（CostB，gen_bindings 自动归入）：GaeaCostProject* / GaeaCostEstimate* /
  GaeaCostEstimateSediment——沉淀 = 选中明细 UPSERT 回 cost_entries，引用既有条目时
  保留其规格/地区/口径/有效期，来源标注「测算沉淀：{项目名}」，缺单价行不沉淀；
  保存版本后项目状态自动置「已保存版本」，沉淀后置「已沉淀」。
- 前端「测算项目」模块（造价数据库一级板块内）：项目列表（条目数/合计/版本数/状态）
  → 项目详情（工程量清单编辑：从成本库搜索引用单价、手动加行、逐行保存/删除、合计）
  → 保存版本（不可变快照 + 版本备注）→ 版本历史（点开回看快照明细）→ 沉淀单价。
- 概览新增「测算项目」统计卡与快捷入口。
- 测试：costproject Store 全流程（金额计算/空项目拒绝版本/快照不可变/版本递增/级联删除）
  + app 层沉淀闭环（保留引用属性、缺单价不沉淀、状态流转）。

### 5.3 造价参考与复盘笔记（第四轮蒸馏，2026-08-19）

对标 zaojia「造价参考（案例分位数/中位数/均值 + 质量状态）」与「复盘笔记（结论/
适用边界/风险提示/证据来源/可信度/有效期/复核状态 + 引用次数）」：

- 造价参考**不落表**：指标由「已保存版本/已沉淀」测算项目的明细行实时聚合
  （costref.ComputeIndicators，按科目或一级分类），临时工作稿不参与对标；统计口径
  R-7 线性插值（Excel 同款），缺单价行排除；样本数 <3 前端标注「样本少」。
- SchemaV11：`cost_review_notes`（标题/结论/边界/风险/证据/可信度/有效期/状态/分类/
  项目类型/工艺/引用计数）；App 绑定 GaeaCostNote* + GaeaCostIndicators（CostB，
  gen_bindings 自动归入，绑定面 495 个方法，漂移检查通过）。
- 前端两个模块：造价参考（案例项目条 + 指标表 + 质量标注）、复盘笔记（关键词/状态
  筛选、新建/编辑弹窗、草稿→已确认、引用计数、删除）。
- 测试：costref Store（CRUD/过滤/引用计数）+ 指标纯函数（分位数/均值/缺单价排除/
  空输入）+ app 层（案例 vs 工作稿聚合、笔记闭环）。

至此 zaojia-database 蒸馏的四大块（数据模型、导入追溯、测算沉淀闭环、参考与复盘）
全部落地。后续可选：AI 复盘追问闭环（保存版本/沉淀后由 agent 生成草稿）、指标点击
下钻到来源明细、office agent 引用已确认笔记并 bump ref_count。

## 6. 接入方式评估：直接内嵌 zaojia-database vs 继续蒸馏（2026-08-19 追加）

### 6.1 「编程方式」是什么

编程板块（ProgrammingPage）内嵌的是**外部服务** DeepSeek Harness Web：本地起服务
（127.0.0.1:3080），gaea 负责 preflight / 启动 / 停止 / 日志，页面 iframe 全出血嵌入；
数据与任务都住在 Harness 自己的环境里，gaea 不做数据耦合。zaojia-database 是纯前端
静态应用（IndexedDB + 本地文件夹），技术上同样可以用「本地静态服务 + iframe」内嵌，
生命周期比 Harness 还简单（无构建、无端口依赖）。

### 6.2 割裂点：直接内嵌的代价

| 割裂面 | 现状（原生成本库） | 内嵌 zaojia-database 后 |
|---|---|---|
| 办公 agent | `cost_search` / `cost_save` 直读写 Hephaestus.db，cost-estimate 模板先查后沉淀 | agent 读不到 IndexedDB 里的数据，测算闭环断裂 |
| 办公右侧面板 | CostLibraryPanel 浏览 + 一键插入上下文 | 面板与内嵌库各是各的数据 |
| 记忆/检索 | MemoryHubOverview 统计 cost、三脑语义检索含成本、图谱节点含成本类型 | 全部感知不到内嵌库 |
| 价格源/历史 | pricefeed 订阅抓价 + 价格快照历史 | 内嵌库无抓价能力，现有能力被闲置 |
| 备份 | GAEA_DATA_ROOT 统一 ZIP 备份 | zaojia 自有 JSON/ZIP 备份，两套备份系统 |
| AI 助手 | 统一走 gaea agent + 办公模型配置 | zaojia 自带远端 LLM 配置（localStorage），重复且割裂 |
| UI/壳层 | 与 gaea 设计系统一致 | 自带 Tailwind/Material Symbols 的第二个工作台，视觉割裂 |

编程板块能接受割裂，是因为 Harness 是**独立开发工具**，与办公数据无耦合；造价数据是
办公工作流（测算/报价/沉淀/引用）的**业务数据**，割裂代价完全不同。

### 6.3 结论与建议

- **不采用直接内嵌作为正式形态**：会与办公产生数据、检索、备份、AI 四层割裂，且闲置
  已有的导入/抓价/历史能力。
- **继续蒸馏**：保留原生一级板块，按 §5 的 P1/P2 逐项把 zaojia 的设计（价格快照口径、
  有效期、导入行号追溯、版本留痕、案例指标）移植进 `cost_entries` 体系。
- 内嵌只保留一个用途：**参考对照工具**（开发/评估时临时打开 zaojia 工作台看它的
  交互细节），不承载真实数据。

## 7. 收敛决策：数据库就是数据库，办公只有一个入口（2026-08-19 终局）

用户在完成四轮蒸馏后定调：**造价数据库是数据中枢，不是第二个办公工作台；办公能力
（测算/对标/沉淀）只从办公板块这一个入口进出**。据此收敛：

1. **造价数据库一级板块回到四个模块**：概览 / 成本条目 / 价格源 / 价格仓库。
   测算项目、造价参考、复盘笔记三个前端工作台式模块从页面移除（组件文件删除，
   manifest 子导航同步）。
2. **后端与绑定保留**：`cost_projects` / `cost_estimate_*` / `cost_review_notes`
   （SchemaV10/V11）、CostB 绑定全部保留且有测试——它们是 agent 能力的数据底座，
   前端不暴露不等于废弃。
3. **能力归办公 agent**：
   - 测算与沉淀：办公侧既有 cost-estimate 模板（先 cost_search、完成后 cost_save）；
   - 造价参考：新增 `cost_indicators` 只读工具（内部 `internal/gaea/tool/builtin/
     cost_indicators.go`），按科目/分类返回案例单价分位数/均值/区间，测算时自动对标；
   - 复盘笔记：不新开前端入口，后续并入知识库（分类=成本复盘）复用 knowledge 体系。
4. **边界写死**：造价数据库只做「存、查、管、抓价」；一切「用」的动作（测算、对标、
   沉淀、复盘）都在办公板块发生。
