# gaea 蒸馏 ProjectLibre：取道不取器——进度计划板块机制供给（2026-09-07）

> 状态：机制供给参考（蒸馏成果已全部落地：16 项差距清偿总账见 gaea-schedule-gap-vs-project-2026-09.md 收官总账，v4.132~v4.139 全 ✅）；clones/projectlibre 仍为只读参考（CPAL 零搬运）。
> 日期：2026-09-07 · gaea 基线 v4.135.0（绑定面 590）
> 上游：git://git.code.sf.net/p/projectlibre/code · master 0530be2（v1.9.8，Java 21）· **CPAL-1.0** · 只读克隆 `clones/projectlibre`（已 gitignore，永不入库、永不抄代码——CPAL 带署名展示+网络使用即分发条款）
> 调研方式：三个只读探索代理分头通读——core（调度/日历/基线/undo/主子项目）、ui（横道/网络/使用视图/交互/打印）、exchange+field+reports，共约 230 个关键文件。
> 定位：给 `docs/gaea-schedule-gap-vs-project-2026-09.md` 16 项差距做**机制供给**（Microsoft 清单只说"要什么"，这份说"怎么想"）。

---

## 0. 摘要

ProjectLibre 是 OpenProj 血统的开源 MS Project 克隆（原班人马出品），core 627 文件 + ui 445 文件 + 内嵌 MPXJ fork（约 20 种格式进出）。蒸馏结论：

- 16 项差距中 **8 项有直接可取机制**（基线比较/资源级日历/使用视图/多级分组/自定义字段/任务路径/任务检查器/主子项目），**1 项上游也空白**（项目模板——全库 grep 无 template 代码，只能自设计），豁免项 leveling 它也只有"字段建模"没有求解器（对话框被注释掉）。
- 五件套最有价值：①负数日期对称编码统一正逆推+哨兵任务；②差异式日历+交集物化；③工时轮廓=任务时间真相源；④快照式 11 基线槽+派生差值字段；⑤声明式 bar 样式表。
- 引擎纪律红线：CPAL 零代码搬运，`clones/` 只读参考；取道=思路进自家 TS/Go 镜像引擎。

---

## 1. 上游能力面（快照 0530be2）

| 模块 | 规模 | 内容 |
|------|------|------|
| projectlibre_core | 627 java | CPM（criticalpath 包）、日历、任务/资源/分配、基线快照、undo、field 框架、grouping 引擎、buffer（时标聚合查询，非关键链） |
| projectlibre_ui | 445 java | Swing：横道/网络(Pert)/使用视图/直方图/电子表格、timescale、打印分页、SVG 离屏导出 |
| projectlibre_exchange | — | 内嵌 MPXJ fork：mpp 二进制(POI+自研)/mspdi/mpx/planner/primavera 等读，mspdi 写；声明式字段映射转换器 |
| projectlibre_reports | 6 java | JasperReports 动态构表（报表定义全在 view.xml 元数据） |

**反面清单（上游自己的坑，不必跟随）**：无项目模板机制；公式字段架子在但 `setFormula` 直接 throw（TODO 未启用）；`MspExporter` 是空壳（导出走旧模型另一条路）；mpp 子项目导入半成品（TODO 注释）；无基线比较报表、无任务路径/驱动链 UI（数据原料在、概念没做）；UI 双击依赖线才能改搭接。

---

## 2. 引擎取道（E1–E9）

**E1 负数日期对称编码 + 哨兵任务（`pm/criticalpath/TaskSchedule.java`、`CriticalPath.java`）**
每个任务持 EARLY(-1)/LATE(+1) 两个 TaskSchedule，晚日程日期以**负数毫秒**表示，早推取 max、逆推取 min 共用同一份代码，日历 add/compare 入口统一处理负号自动反向。全图用 `<Start>/<End>` 两个零工期哨兵包裹：项目开始约束=start 哨兵的 SNET、截止倒排=end 哨兵的 FNLT——**倒排与正排是同一算法只翻方向**。
→ 取法：不重构现有 cpm.go/cpm.ts，作为**未来动逆推时的设计准绳**。v4.133 曾出"AOA 逆推锚点从未生效致假负时差"的 bug（planFinish 锚点游离于主算法外），哨兵化正是让锚点不再是特例的解法。下次触碰逆推时按此收口。

**E2 增量 CPM：拓扑链表+脏标记代数（`PredecessorTaskList`、`calculationStateCount`）**
改动的任务若不影响关键路径只跑正推；唯一前继的后继直接内联传日期；三步进 stateCount 区分三遍并兼做脏标记。父任务用 PARENT_BEGIN/END 双引用在排程链表里"夹住"子任务，摘要聚合是同一次遍历的 O(1)。
→ 取法：现阶段不动（我们全量重排在千级任务内够用）。记录为性能预案：任务数上数千+连续拖动卡顿时按此改造。

**E3 差异式日历+展平缓存+周粗调（`WorkingCalendar`/`CalendarDefinition`）**
日历只存相对 base 的**差异**（周历覆盖+日期异常=节假日/调休），读取时展平成具体日历并缓存、修改即失效。热路径 add() 先按整周工时粗跳再补异常差值、余下逐天精调——复杂度取决于异常天数而非工期长度。
→ 取法：差距 **#13 资源级日历**的实现底座。我们 calendar.go 已是"基历+例外"形态，核对两点：有无展平缓存、有无整周粗调；资源个人日历直接按 E4 叠加。

**E4 任务日历 ∩ 资源日历交集物化（`AssignmentDetail.getEffectiveWorkCalendar`）**
资源休假不靠运行时逐日合并，而是把"任务日历∩资源日历"的**几何交集预计算物化**挂到分配上；交集为空报错回退任务日历；基线还把当时日历克隆冻结（基线重算不受后续改历影响）。
→ 取法：#13 照此实现——分配粒度存交集日历，天然得到"个人休假生效+交集为空检测+基线不漂移"三件事。

**E5 工时轮廓=任务时间真相源（`AssignmentDetail`+`PersonalContour`）**
分配=开始+delay+工期+桶序列轮廓（每桶{时长,单位%}）。split=插 0% 桶；开工=actual 段与剩余段分离；甘特多段条/直方图全是"遍历轮廓生成工作区间"一个机制的投影。
→ 取法：差距 **#12 使用视图+超载检测**的前置抽象——引入"任务的时间分布可枚举为工作区间"这一层（我们已有日历，缺区间生成器），之后逐日分桶累加=纯只读聚合，不碰排程状态。

**E6 leveling=levelingDelay 字段（豁免项机制认知）**
上游没有调配求解器：只给每任务/每分配存一个 levelingDelay（工时毫秒），排程时在依赖日期之上统一 `calendar.add(begin, levelingDelay)` 消化，CPM/undo/序列化全部免费。**调配做成"输出延时建议值写回字段"，引擎内核不污染。**
→ 取法：维持豁免；若未来做"超载一键顺延建议"，按此形态——算法在分析侧，结果是一个字段。

**E7 快照式多基线+派生差值（`SnapshotList` 11 槽 + configuration.xml 派生字段）**
11 个命名基线槽（baseline+baseline1..10），每槽克隆 {start,finish,duration,cost,work}+分配轮廓+**冻结日历**；基线字段全部 readOnly（注释原话："允许改基线很危险"）；差异不设 diff 结构，而是派生字段（当前−基线槽 i）；"存快照"本身可 undo。
→ 取法：差距 **#11 基线汇总+版本比较**。我们刀7 只有单基线，扩成 N 槽（2–3 个够国内签证 1→7 场景起步）+差值列由派生函数出，不写 diff 算法。xlsx 往返加 baselineStart/Finish 列即可带基线出行。

**E8 逐依赖自由时差=驱动判定（`TaskSchedule.calcDependencyDate`、`Task.getFreeSlack`）**
上游缓存了每任务的 dependencyDate（所有前继推算值的 **max**）和逐依赖 free slack（=0 即该依赖在驱动）。没有 UI 概念但数据原料齐全。
→ 取法：差距 **#8 任务路径分析 + #9 任务检查器**一步到位的正解——正推时给每条依赖留贡献值，任务开始日期=argmax 的那条即驱动依赖；高亮=沿驱动边 BFS 前驱/后继链；检查器=列出各依赖的 free slack 与驱动标记。这两个小刀的引擎侧成本因此很低。

**E9 子项目=任务节点+外部任务占位符（`SubProj`/`ExternalTaskManager`）**
子项目在主项目里是一个任务节点（持有子 Project），打开时子任务拼进主拓扑链、约束"不早于子项目自身开始"；跨项目依赖用 isExternal 影子任务占位，真实项目打开/关闭时**换入换出**（换入时检测循环则 disable，换出时把实任务日程拷回占位符）。
→ 取法：差距 **#15 多工程**。印证既有拍板池设计（每文件一工程+索引）可行；跨文件依赖先用占位符+冻结日期，不必做跨文件实时重算。

---

## 3. UI 取道（U1–U6）

**U1 声明式 bar 样式表（`BarFormat`/`BarStyles`+谓词+图层号+槽位）**
"哪类任务画什么条"是规则数据不是 if-else：每条 format = 谓词（critical/summary/milestone 公式）+ from/to 字段 + 三段 shape + **layer**（1000–1499 背景→500–999 连线→1–499 前景）+ **row**（1=主条、2..12=基线槽位行）。摘要黑五边形帽、里程碑菱形、关键红条、松弛斜纹条、基线小条全是不同 format。右键"条样式"菜单直接由规则表生成（样式表即菜单）。渲染前空跑一遍把每行 bar 的 bbox 存回行对象，命中测试零重算。
→ 取法：差距 **#8 路径高亮**（插一条更高优先级的 format 即可）、PDF 对比余项 **bar 标注**（独立 AnnotationRenderer 管线：字段值画在条尾+偏移）、**里程碑旗标**、基线叠加条，都变成"加规则数据"。建议把现有横道渲染收敛为 rule[]+layer 分桶，一次到位。

**U2 纯线性 timescale+双级 TimeIterator（`TimeScaleManager`/`TimeIterator`）**
刻度=有序档位数组（zoom=下标±1），ratio=minWidth/minDuration 纯线性，toX/toTime O(1)；TimeIterator 惰性产出双级区间（主级跨边界才产出，次级逐格），刻度头/条/直方图/使用视图列全共享同一 converter 与 iterator。
→ 取法：对照现有刻度实现核对——只要存在"按列查时刻"的反查逻辑就收敛为纯函数换算；双代号时标网络的时刻定位直接受益。低风险核对项，不单开刀。

**U3 单一可见行序列+filter→sort→group 管道+合成组头行（`NodeCacheTransformer`/`NodeGrouper`）**
表格 DOM、canvas 条、网络节点全部渲染自**同一个可见行数组**（rowIndex×rowHeight=y），左右滚动/选中天然对齐。多级分组=递归扫描切组+造**合成 GroupNode**（就是一个 summary 行），折叠/汇总/缩进复用 WBS 摘要全部机制；空摘要枝剪除；hiddenSorter 保 WBS 序、userSorter 用户序。
→ 取法：差距 **#16 排序与多级分组**的完整蓝图。v4.135 已有筛选（两遍法），升级路径：visibleRows 成为唯一状态源（含 `__groupLevel/__isGroup/collapsed`），分组在状态层递归插组头行即可，行右键/菜单零改动。

**U4 交互状态机+ghost 预览+命中缓存（`GraphInteractor`/`GanttInteractor`）**
hover 即判状态（move/resize/progress/link/split）并定光标；拖动全程只画 XOR 影子不改数据；跨行自动切"拉依赖线"状态；pointerup 才把"意图+时长增量"交领域服务并入 undo 栈。
→ 取法：对照刀9 拖拽实现核对三点：命中区是否用渲染后缓存的 bar bbox（别逐条 hit-test）、拖动是否零数据变更纯 ghost、提交是否单 action 进 undo。查漏补缺不重写。

**U5 同一 paint(ctx, clip) 三用（`offline_graphics`/`GraphPageable`）**
屏幕/SVG 导出/打印共用一个离屏视图的 `paint(g, clip)`：clip 决定画什么，printBounds 自报分块数，分页=逐 tile 调用。
→ 取法：刀D1/D2 已各自实现导出，收敛方向明确：图面构建器签名统一为 `draw(ctx, viewport)`，打印分页与 PNG/PDF 逐 tile 复用同一函数。下次触碰导出（如 bar 标注）时顺手收口，不单开刀。

**U6 使用视图分桶列+直方图（`UsageDetailView`/`TimeSpreadSheet`/`ChartModel.computeHistogram`）**
使用视图="左明细+右时间表"，每个 TimeInterval 一列，取值="字段×时间区间"二维；不可编辑桶涂灰。直方图=分桶累计工作量阶梯面积+**可用性曲线**粗框叠加，柱超出线即超载；选中集可过滤直方图统计范围。
→ 取法：差距 **#12** 的视图蓝图（引擎侧靠 E5 区间枚举）。资源刀4（mspdi 绑真机）之后的资源使用小刀照此做。

次级：辅助线统一背景 pass（今日线/状态日期线/拖拽参考线/非工作日底纹同一条管线，我们已有底纹+今日线，新增参考线时归入）；任务信息对话框六页签（General 含基线日期对照、前驱/后继页签超链接可跳转沿链查看——#9 检查器的交互形态直接抄）。

---

## 4. 交换/字段取道（X1–X3）

**X1 声明式字段映射表+区间展开+值转换器管道（`FieldUtil.convertFields`+各 Converter）**
三元组 {本地字段, 外部字段, 转换器} 一张表双向用（from 布尔），`cost:1:10` 区间语法展开扩展列，null 一律跳过（"未设置"语义），值转换器可插拔（DateUTC/截断文本…）。
→ 取法：xlsx 7 列往返（刀D3）从硬编码泛化为表驱动=加列即加行；mspdi 解析（资源余刀4）直接挂同一张表的第二个格式，我们的"口径 7 列"就是 mspdi 列子集。

**X2 固定槽位自定义字段+字段注册表（`CustomFieldsImpl`+configuration.xml `<field>`）**
自定义字段=按类型定长数组（text×30/number×20/flag×20/date×10…），字段声明带类型/格式/列宽/**别名**/汇总类型(sum|avg|min|max)/readOnly；别名覆盖显示名（正好解决"上报口径列名≠系统字段名"，刀D3 表头别名归一是它的特例）；报表列宽/聚合/格式全由这份元数据驱动。
→ 取法：差距 **#14 自定义字段**——Go 侧一份字段注册表（schema 内置），任务/资源各挂固定槽位数组，xlsx/mspdi/未来报表共用元数据。公式字段不做（上游自己都没启用）。

**X3 mspdi 进出要点（`MspImporter` 阶段序+`MSPDISerializer`）**
导入固定阶段序：Options→Calendars→Resources→Tasks(**含 11 个基线快照直接填槽**)→Dependencies→Header；写侧 JAXB 整树序列化。
→ 取法：余刀4 绑真机时按此阶段序做导入管线；基线槽与 mspdi baseline1..10 天然对称（E7 的额外收益）。

---

## 5. 差距清单 × 机制对照（刀路供给）

| 差距项（gap 文档编号） | 上游机制 | 取法 | 建议刀路 |
|---|---|---|---|
| #8 任务路径分析 | E8 驱动判定+U1 高亮 format | 正推留依赖贡献值，沿驱动边 BFS；高亮=加一条 bar 规则 | 小刀（引擎半日+视图半日） |
| #9 任务检查器 | E8 逐依赖 free slack+六页签对话框 | 面板列依赖+free slack+驱动标记，前驱名可点击跳转 | 小刀（复用 isLinkBinding 亦可） |
| #10 项目模板 | **上游空白** | 自设计（plan JSON 克隆+模板库目录） | 排队，无可抄 |
| #11 基线比较 | E7 多槽快照+派生差值 | 扩 N 槽+差值列函数，不写 diff | 排队 |
| #12 使用视图+超载 | E5 区间枚举+U6 分桶/直方图 | 先做区间生成器，再做分桶聚合+可用性曲线 | 排队（资源刀4 后） |
| #13 资源级日历 | E3 差异日历+E4 交集物化 | 分配粒度存∩日历+空交集回退 | 排队 |
| #14 自定义字段 | X2 固定槽位+注册表 | 字段注册表+类型化槽位数组 | 排队 |
| #15 多工程 | E9 占位符换入换出 | 印证既有设计；跨文件依赖先占位符 | 待拍板（设计已落档） |
| #16 排序多级分组 | U3 可见行序列+合成组头行 | visibleRows 单一状态源+递归组头 | 排队 |
| mspdi（资源余刀4） | X1 映射表+X3 阶段序 | 表驱动转换+六阶段导入 | 进行中刀路直接吸收 |
| bar 标注/旗标（PDF 余项） | U1 标注管线+format | 加规则数据即可 | 顺带进下次导出刀 |
| 引擎逆推加固 | E1 负数对称+哨兵 | 不重构；动逆推时按此收口 | 设计准绳 |
| 性能预案 | E2 增量 CPM | 千级以上再启用 | 记录在案 |
| leveling（豁免） | E6 levelingDelay 字段 | 若做建议值：分析侧算、字段落、引擎消化 | 维持豁免 |

---

## 6. 不抄清单（红线）

- **零代码搬运**：CPAL-1.0 含署名展示+网络使用即分发条款，`clones/projectlibre` 永不编译、永不引用、永不拷文件，只读思路。
- Swing 具体件（JSplitPane/JFreeChart/JasperReports/XOR 绘制/UndoManager）——平台不对，取其架构不取其件。
- undo 机制：上游命令逆操作式 vs 我们 v4.135 快照式（上限 50+800ms 合并）——**维持现状**，不做迁移；仅记一条"字段层写入收敛"作 Go 侧长期方向。
- 公式字段（上游 TODO 未启用）、leveling 求解器（上游没有）、关键链 buffer（上游 buffer 包只是时标聚合查询，不是 CCPM）。

## 7. 来源

- 克隆：`clones/projectlibre`（git.code.sf.net/p/projectlibre/code，master 0530be2，2026-09-07）
- 引擎深读：`projectlibre_core/src/com/projectlibre1/pm/criticalpath/{TaskSchedule,CriticalPath}.java`、`pm/calendar/{WorkingCalendar,CalendarDefinition}.java`、`pm/assignment/{AssignmentDetail,contour/*}.java`、`pm/tasks/SnapshotList.java`、`undo/*`、`pm/task/{SubProj,ExternalTaskManager}.java`
- UI 深读：`projectlibre_ui/src/com/projectlibre1/pm/graphic/{gantt/timescale/network/views}`、`graphic/configuration/BarFormat.java`、`offline_graphics/*`、`print/GraphPageable.java`
- 交换/字段：`projectlibre_exchange/src/com/projectlibre/core/pm/exchange/{MspImporter,ProjectConverter}.java`、`net/sf/mpxj/mpp/MPPReader.java`、`projectlibre_core/src/com/projectlibre1/{field,configuration/configuration.xml}`
- 关联文档：`docs/gaea-schedule-gap-vs-project-2026-09.md`（16 项差距）、`docs/gaea-schedule-standards-digest-2026-09.md`（JGJ/T 121 规范蒸馏）
