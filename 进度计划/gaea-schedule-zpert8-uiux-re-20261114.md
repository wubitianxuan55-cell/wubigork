# 斑马进度 ZPert 8 · 前端 UX/UI 逆向调研（2026-11-14）

> **样本**：`C:\Program Files\ZebraTech\ZPert8\ZPert.exe`（7.0.706.0，22.3 MB，SHA256 `C61972FA…4E094E68`）
> **本轮中心**：前端 **UX / UI**（信息架构、控件体系、交互模型、视觉与文案），**不改 gaea 代码**，只做取证与对标。
> **方法**：PE 静态取证为主（无 Ghidra：本机 `GHIDRA_INSTALL_DIR` 未配置，且该样本 `.pdata` 已损坏）——
> 自建「代码区段化 + 字符串引用索引」（capstone 线性切块 + RIP 相对引用扫描 + 就地多编码解码），
> 把「界面文案 ↔ 代码位置」建起来，按指令地址顺序复原建界面/注册命令的顺序。
> **配套证据**：`C:\AI\zpert-re\`（**仓库外**，见 §12 清单），工具：`zpert-re/tools/uiindex3.py`、`uiscan.py`、`uicmd.py`、`uifind.py`。
> **与既有文档的关系**：`gaea-schedule-zpert8-distill-20261114.md`（内核/口径）、`…-network-re-20261114.md`（网络图/协议）、
> `…-replication-assess-20261114.md`（复刻判定）已覆盖算法与数据；**本文只补 UI 面，且以"能照着做界面"为标准取证。**

---

## 0. 结论先行（七条）

| # | 结论 | 分量 |
|---|---|---|
| 1 | **功能区是数据驱动的**：`<页>/<面板>/<条目组>` 三级路径 + `<页>-<命令>` 命令键 + 分段命令 ID 组成一张注册表，页签/面板由路径前缀生成，因此**支持用户自定义功能区**（命令键/路径是稳定标识；`Ribbon@zpbdata` 实为**授权受限项**而非布局，见 §13.1 勘误） | 决定复刻方式：不是硬编码菜单，而是"命令表 → 渲染" |
| 2 | **UI 栈 = Qt6 Widgets + SARibbon + SAFrameless + QWK**（不是 Web）：ribbon 用开源 SARibbon（Category/Pannel/Gallery/ContextCategory/QuickAccess），窗口是无边框自绘（`QWK::WidgetWindowAgent` + `SAFramelessHelper`） | 决定了"复制它的观感"要用什么控件模型 |
| 3 | **表格与图是自绘 C++ 网格**：`ZWorksGridWidget`/`ZBaseGrid`/`ZQtGridView` + 数百个 `*GridCell` 列类型；画布 `ZQtPertView`/`ZGanttState`/`CGanttPrint` | 列 = 类型对象，列顺序/宽度是数据 |
| 4 | **一个文档窗口承载"表格 + 图"双栏**：`show_grid`/`show_pert` 切换按钮、表格编辑（左）显示/隐藏、网络图区域分隔设置；图模式 5 种（时标网络图/横道图/逻辑网络图/单代号网络图/表格网络图） | gaea 的 GanttView/AoaView/PdmView 是"切页"，斑马是"同屏双栏 + 切图" |
| 5 | **交互密度极高**：上下文菜单 12 项、过滤器菜单族（文本/数值/日期/枚举/异常标记/进度滞后）、颜色与线宽选择器、日期弹出、格式刷、在位编辑、20 个自定义光标 | 专业用户的"手感"来自这些细节 |
| 6 | **AI 是"内联网格助手"而不是聊天窗**：图标 + 浮框 + 键盘采纳（Enter / Shift+Enter / Delete / Esc），埋点事件族 `12040xx` 覆盖采纳/不采纳/失去焦点 | 直接可对标 gaea 的 AI 计划 diff 交互 |
| 7 | **主题/外观是可切换的**：6 套 QSS（win7 / office2013 / office2016-blue / office2021-blue / dark / dark2）+ 偏好设置里的**护眼模式 / 黑白模式**开关；图标按 20…66px 与 100/125/150% 三档位图 | 分值高且**成本低**，是最好抄的一块 |

---

## 1. 取证环境与方法（可复现）

| 项 | 内容 |
|---|---|
| 为什么不用 Ghidra | 本机未配置 `GHIDRA_INSTALL_DIR`；且样本 `.pdata` 内容为高熵乱码（节名 `.l'&` 异常），`RUNTIME_FUNCTION` 不可用 → 无法按函数边界聚合 |
| 自建索引 | `uiindex3.py`：capstone 线性反汇编 `.text`（7.9 MB），以 `0xCC/0x90` 填充与解码失败切块（51,386 块 ≈ 函数）；再扫全部 RIP 相对寻址（`lea/mov r64,[rip+d32]`）95,812 处，就地解码 52,730 条字符串引用 |
| 编码竞争规则 | 中文按 UTF-16 读、标识符按 ASCII 读；用「CJK 码位高字节 ∈ 0x4E..0x9F」统计判别"显示/关系"这类两字节全可打印的歧义串（否则会被读成 `>f:y`） |
| 顺序语义 | 同一代码块内**按指令地址排序**的字符串引用 ≈ 源码里"建控件/加条目"的顺序 → 页签字段顺序、命令注册顺序可复原 |
| 局限 | 块边界是近似函数；少数标签因指针落在串中间被截断（如"另存为"→"存为"）；页签**显示顺序**未取证（见 §11） |

---

## 2. 技术栈与渲染架构

| 组件 | 证据 | 说明 |
|---|---|---|
| Qt 6.7.3（广联达 fork） | `Qt6Core/Gui/Widgets/Network/Pdf/Svg/SvgWidgets/PrintSupport/Xml.dll`（2024-09 构建） | 全部 UI 走 Widgets，无 QML（RTTI 无 `QQuick*`） |
| **SARibbon** | RTTI 70+ 类：`SARibbonBar/Category/Pannel/ContextCategory/Gallery/QuickAccessBar/ApplicationButton/ActionsManager/TabBar/MainWindow…`；qrc `/SARibbon/image/resource/*` | 功能区外壳来自开源 SARibbon；`SARibbonMainWindow` 即主窗基类 |
| **SAFramelessHelper** | 字符串 `SAFramelessHelper` | 无边框 + 命中测试（自绘标题栏/拖动区） |
| **QWK** | `QWKWidgets.dll`/`QWKCore.dll`；`QWK::WidgetWindowAgent::setTitleBar/setHitTestVisible` | 窗口代理：标题栏、阴影、拉伸边框 |
| 自绘网格 | `ZBaseGrid`、`ZQtGridView`、`ZWorksGridWidget`、`ZWorksGridTopHdg/SideHdg`、`ZGridCell` + ~200 个 `*GridCell` 子类 | 虚拟化网格、表头/侧头分离、每列一个 cell 类型 |
| 图/画布 | `ZQtPertView`、`ZBasePertView`、`ZGanttState`、`CGanttPrint`、`ZQtPreviewView` | 网络图/横道图绘制与打印预览 |
| CEF 嵌入 | `QCefView.dll` + `CefView\libcef.dll`(216 MB)、`chrome_100/200_percent.pak` | 只挂：登录/云计划/授权/AI 助手/帮助/购买页 —— **业务界面不在网页里** |
| 打印 | `Qt6Pdf.dll`、`Qt6PrintSupport.dll`、`CGanttPrint`、`PrintPreviewEventFilter`、`QPrintPreviewWidget` | 打印预览是 Qt 原生 |
| 图标体系 | qrc `/ribbon/resource/ribbon20/25/30/33/35/40/41/50/58/66.png`、`status_bar_16/20/24/28/32.png` | **同一功能的 10 档 ribbon 图标尺寸**（按 ribbon 高度/DPI 选择） |
| 高分屏位图 | `/res_balance/resource/*_100/125/150.bmp`、`/setting/resource/*_small.bmp` | 100/125/150% 三档资源 → DPI 感知是"多套位图"而非矢量 |
| 主题 | qrc `/theme/resource/theme-{win7,office2013,office2016-blue,office2021-blue,dark,dark2}.qss` | 6 套 QSS 皮肤（SARibbon 主题族 + 暗色） |
| 调色板 | `/resource/system_light_palette.dat`、`ribbonTheme`、`SARibbonTheme` | 主题名存储键存在（偏好项落盘） |
| 国际化 | `translations\ZPertQt_zh_CN.qm` 仅 **0.3 KB**（业务文案几乎全部硬编码在 exe 的 UTF-16 字面量里）；`qt_zh_CN.qm` 142 KB 是 Qt 自带 | 无独立 i18n 表 → 复刻文案只能从 exe 抽 |

**判断**：斑马是「**C++ 排程内核 + 本地自绘网格 + ribbon 外壳 + CEF 点缀**」。要做观感对标，控件模型应选
「ribbon/命令表 + 虚拟化表格 + 自绘画布」，而不是"再写一套网页表格"。

---

## 3. 主窗口骨架

证据：`SARibbonMainWindow`、`SARibbonQuickAccessBar`、`SARibbonApplicationButton`（`/ribbon/resource/application_btn.svg`）、
`ZTitleTabBar`、`ZPertMainWindow`/`ZPertDocWindow`/`ZPertWindow`/`ZPertDocWindow` + `QMdiSubWindow`、`QDockWidget`、
`ZPertStatusBarQMenu`、`ZWorksGridWindow`、`ZQtGridWindow`。

```
┌ 应用按钮(logo) ┬──────────────── 快速访问栏 ────────────────┬ 窗口按钮 ┐   ← 无边框自绘标题栏
├ 页签: 文件 | 计划 | 资源 | 管控 | 协作 | 视图 | 帮助 ────────────────┤   ← SARibbonTabBar
├ 面板行(每页多面板，面板内分"大按钮/小按钮/画廊/下拉") ─────────────────┤   ← SARibbonCategory/Pannel
├ 文档标签栏(多文档) ZTitleTabBar ────────────────────────────────┤   ← MDI 文档（ZPertDocWindow）
│ ┌──── 表格(左, 可隐藏"显示表格编辑") ────┬──── 图(右: 网络图/横道图) ────┐ │
│ │ ZWorksGridWidget (列头/侧头/正文/冻结) │ ZQtPertView (时标刻度/箭线/节点) │ │
│ └───────────────────────────────────────┴────────────── 泊靠面板(右侧预警) ┘ │
└ 状态栏(含 status_bar 图标族 + QMenu 弹出) ─────────────────────────────┘
```

- **同屏双栏**：`:/button/resource/show_grid` / `show_pert` 一对切换图标 = 表格/图独立显隐；另有"隐藏表格编辑(左侧)/显示表格编辑(左侧)"命令；
  "网络图区域分隔设置"命令 = 双栏分割比例可调。
- **泊靠面板**：文案「进度检视状态下…**右侧泊靠窗体**展示预警信息」→ 右侧是可停靠的预警/检查面板（`QDockWidget`）。
- **状态栏**：`ZPertStatusBarQMenu` + `status_bar_*.png` → 状态栏条目可点击弹菜单（缩放/单位/筛选一类）。
- **多文档**：`文件-关闭所有文件`、`QMdiSubWindow`、`恢复未保存的文件` → 多计划标签页 + 崩溃恢复。

---

## 4. 功能区信息架构（本轮最有价值的复原）

### 4.1 数据模型（推断自注册表结构，证据见 §4.2）

一个命令条目 ≈ `{ 命令键 "<页>-<命令>", 路径 "<页>/<面板>[/<条目组>]", 命令ID "1xxxx…9xxxx", 动作GUID "5000000000xxxxxxxxxx", 显示名 }`：

- **路径**决定它挂在哪：页签 = 第一段，面板 = 第二段，条目组 = 第三段（如 `计划/视图/分组`）。
- **命令键**是稳定标识（埋点、快捷键、自定义功能区、二次开发都靠它）。
- **命令 ID 分段**：`1xxxx` 文件/打印、`2xxxx` 计划、`3xxxx` 资源、`4xxxx` 管控/期间、`5xxxx` 帮助/反馈、`6xxxx` 导航、
  `7xxxx` 检查/里程碑、`8xxxx` 关系合法性校验、`9xxxx` 授权/云/AI 商业、`11xxxxx/12xxxxx` AI 交互埋点。

> 这条模型解释了为什么它能做「自定义功能区 + 快捷键助手 + 埋点字典」三件事共用一套命令表。

### 4.2 页签与命令清单（命令键复原，节选）

| 页签 | 证据路径/键 | 命令（原文） |
|---|---|---|
| **文件** | `文件-*`（`10001…10013`、`10019`） | 打开向导、新建文件、打开文件、从其他系统新建、保存、另存为、导出、打印设置、打印、偏好设置、关闭所有文件、云计划保存、服务器设置 |
| **计划** | `计划/视图/计划编制`、`计划/视图/分组`、`计划/大纲`、`计划/高级`、`计划/上下文菜单` | **视图面板**：自定义视图、内置视图 9 种（计划编制/进度管控/期间计划/期间进度报告/资源/工程量计划/进度对比分析/计划变动分析/里程碑列表）、保存视图、视图管理、上传云视图、时标网络图、横道图、逻辑网络图、单代号网络图；**分组面板**：不分组/执行状态/工作分类/总时差/关键工作/负责人/工作部位/里程碑/资源名称/工作名称/工期/前置工作数/后置工作数/自定义分组/隐藏父工作/父工作树形显示/父工作合并行显示；**编辑**：复制/粘贴/删除/格式刷/查找/分类；**格式**：字体/字号/字体颜色/填充/线条宽度；**大纲**：折叠/升级/降级/上钻/下钻/上移/下移；**插入**：工作/关系/里程碑/选择；**高级**；**上下文菜单** |
| **资源** | `资源/智能优化`、`30001…30027` | 资源图设置（添加自定义/挂接资源图/确定/不显示资源图）、资源库维护（新增/保存/导入导出/从 excel 导入导出）、定额库维护、工期固定资源均衡、资源限量工期最短、资源统计表导出、更改检视/反馈时间、历史前锋线对比分析 |
| **管控** | `管控/调整`、`40001…40029` | 期间进度报告、计划变动分析、进度对比分析、版本、日期范围、导出进度报告（期间范围设置）、按预测时间 |
| **协作** | `协作-上传协作/开始协作/停止协作`、`本地协作-邀请协作/公开协作/停止协作` | 云协作与本地协作两条链路 |
| **视图** | `视图-系统内置视图*`、`视图-本计划自定义视图`、`视图-系统自定义视图` | 12 项：9 个内置视图 + 本计划自定义视图 + 系统自定义视图 + 系统内置视图 |
| **帮助** | `帮助-点击/搜索`、`建议反馈-导航栏/工具栏`、`上传云端-模板库` | 帮助中心、反馈、模板库 |
| **上下文（右键）** | `计划/上下文菜单`（`20097…20109`） | 添加工作、升级、降级、上移、下移、成组、解组、复制、粘贴、格式刷、删除、文字颜色 |

### 4.3 打印（独立页签，16 项 + 12 项通用）

`打印-*`（`10021…10029`）+ `横道图打印-*`（`10035…10050`）：
**打印确认 / 纸张方向 / 纸张大小 / 页边距 / 标题设置 / 图注设置 / 边框颜色 / 边框线宽 / 外边框加粗 /
宽度调整为一页 / 整体调整为一页 / 无缩放 / 打印页码 / 时间刻度 / 标题位置 / 图注位置 / 打印预览-横道图**。
配套位图：`内置图片\横版-打印色.png`、`横版-显示色.png`、`竖版-打印色.png`、`竖版-显示色.png` → **图签（标题栏）横/竖两版 × 打印色/显示色两态**。
另有 `ZQtIntelligentLayoutDlg`（智能布图）、`ZQtIntelligentLayoutPrintDlg`（按纸张布图）→ 打印不是"缩小视图"，是**重排布图**。

---

## 5. 视图体系与图上显示

| 维度 | 事实 | 证据 |
|---|---|---|
| 图模式 | **5 种**：时标网络图 / 横道图 / 逻辑网络图 / 单代号网络图 / 表格网络图 | `空白计划-横道图|网络图|表格网络图`（新建计划三选一）；`20014…20017`；qrc `grid_pert_%1.bmp` |
| 表格/图混合 | `网格+网络`（`grid_pert`）是独立形态 | qrc `:/setting/resource/{gantt,pert,grid_pert}_%1.bmp` 及 `_down` 变体 = 4 态按钮 |
| 时间标尺 | **双标尺**：上方"工程标尺（年-月-日）"、下方"进度标尺（星期）"；单位/间隔可设、刻度按月对齐、可插入/删除/上移/下移自定义刻度段、可局部缩放、字号可设 | 时间刻度对话框 §7.2 |
| 图上内容开关 | 4 个：`show_dot_time`（节点时间）、`show_duration`（工期）、`show_six_para`（六参数）、`show_start_stop_time`（起止时间） | qrc `:/setting/resource/show_*_small.bmp` |
| 节点/关系样式 | 节点样式、圆节点半径（界面控件名 `cycle_radius`）、箭线颜色、虚工作颜色 | `dot_style_`、`cycle_radius_`、`virtualjobcolor`（既有网络图取证） |
| 进度检视 | 前锋线（进度折线）+ 历史前锋线对比 + 拉直前锋线（按原计划工期/按原计划时间/按预计剩余时间）+ 更改检视/反馈时间 | `拉直前锋线-*`、`更改检视/反馈时间`、`历史前锋线对比分析` |
| 分区 | 网络图中"工作"可显示为分区；右键编辑分区、分区设置对话框 | `网络图中显示为分区`、`右键编辑分区`、`PartitionSettingDlg` |
| 预警/校验标记 | 列内标记：`external_warn.png`、`plan_time_warn_red/green.png`；右侧泊靠预警面板 | qrc `/grid/resource/*` |
| 水印 | `watermark_3581.png`（盗版/受限态水印） | qrc `/grid/resource/watermark_3581.png` |
| 图标化叙事 | 内置图片里有**施工机械/人物/旗帜/五角星**（挖掘机、塔吊、吊车、混凝土车、工人、安全帽、电梯、黑旗、红/黄/蓝/绿/粉五角星与小旗）| `内置图片\*.png` → 里程碑/关键节点的**图形符号库** |

---

## 6. 工作信息面板（`WorkPagesDialog`）—— 页签与字段（顺序即界面顺序）

**页签集合**（命令键证据，`工作信息-*`）：**概况 / 关系 / 资源和统计 / 高级 / 自定义字段**（5 个）。
**布局树证据**：`region_140679b20`（`WorkPagesDialog` → `verticalLayout` → `tabWidget` → 各页 `verticalLayout_N`/`widget_N`，共 124 个 objectName）。
**字段证据**：`region_140678710`（字段标签地址单调递增，顺序即建控件顺序）。

| 页 / 块 | 字段（原文顺序） | objectName 佐证 |
|---|---|---|
| **概况** | 名称、工期(+单位)、类型、分类、网络图中显示为分区、工期自动计算、计划开始、计划完成、预测开始、预测完成、工作部位、负责人、备注 | `labelName/lineEditName`、`labelDuration/comboUnit`、`labelType/comboType`、`labelGroup/comboGroup`、`checkPartition`、`checkAutoCompute`、`labelPlanStart/labelPlanStop`、`labelEstimateStart/labelEstimateStop`、`labelSite` |
| （工程量/统计块，归属页签按字段语义推断，**未取证页签标题**） | 工程量、累计完成量、执行状态（未开始/进行中/已完成）、累计实际完成(%)、典型偏差、实际开始、预计完成、实际完成、偏差原因、显示于网络图、解决措施、执行 | — |
| **关系** | 前置工作列表（`WorkPagesDialog` 内含 `NameEditFilter` 成员） | — |
| **高级 / 高级设置** | 外文名称、工作编码、要求开始、日历、要求完成、仅预警、排网方式、相对开始、相对完成、计算时间（预测）、最早开始、最早完成、最晚开始、最晚完成、最晚开始（自由）、最晚完成（自由）、自由时差、总时差 | — |
| **自定义字段** | 用户自定义列（与 `ZColumnSettingDialog`、`CustomGridCell` 配套） | `CustomGridCell`、`customFields` 概念对位 |
| 付费墙 | **「该页签功能为高级功能，请购买后使用。」**（`0x14067877e`）—— 付费门在**页签粒度** | — |
| 页内提示条 | `labelUltimateHint_summary`（带 `color: red;` 行内样式） | 红色警示条独立控件，不占用字段区 |

> 对 gaea 的意义：这 4 个页签正好对应 gaea 的 `TaskInspector`；差异是斑马把**六参数（最早/最晚/自由时差/总时差）直接铺进"高级"页签**，
> 而且"相对开始/相对完成 + 排网方式 + 仅预警"是**排网策略**字段（不是展示字段）。

---

## 7. 设置体系（对话框字段级证据）

### 7.1 偏好设置（`PreferenceSettingDialog`，0x140689480）

| 分组 | 项 |
|---|---|
| 全局设置 | 计价方式、标注文种、计划设置、**开启护眼模式**、**开启黑白模式**、定时备份（分钟）、加工作时出对话框、恢复为默认设置 |
| 服务器设置 | 服务器设置（云/协作端） |

- 「护眼模式 / 黑白模式」+ 6 套 QSS = **外观可切换**（护眼≈浅米色，黑白≈高对比/黑白稿）。
- 「加工作时出对话框」= 批量/快速录入模式开关（专业用户要"连打不入弹窗"）。

### 7.2 时间刻度（`GanttOptionsSetCtrlPageDlg`/`OptionsSetCtrlPageDlg`，0x14069d050）

`时间刻度 → 刻度设置`：上方刻度=工程标尺（年-月-日）、下方刻度=进度标尺（星期）、单位、间隔、刻度按月对齐、
时间刻度线（画小刻度线/画大刻度线/画工程日历线/不画刻度线）、`间隔 = …`、格式修改、标尺线、刻度线、时间刻度字体、
**时标刻度局部缩放**、刻度段 插入/删除/上移/下移。

### 7.3 其它设置入口（71 个 `*Dlg/Dialog` 类，按域归类）

| 域 | 对话框 |
|---|---|
| 全局/项目 | `PreferenceSettingDialog`、`GlobleSetDlg`、`PlanSettingDlg`、`PlanSetCtrlPageDlg`、`OptionsSetCtrlPageDlg`、`GanttOptionsSetCtrlPageDlg`、`ScriptSetCtrlPageDlg`、`DiarySetCtrlPageDlg` |
| 视图/列/样式 | `ZColumnSettingDialog`、`ZQtSaveViewDlg`、`ZQtManageViewDlg`、`WorksStyleSetDlg`、`WorksGanttSetDlg`、`SymbolStyleDialog`、`TextLineStyleDlg`、`SpecialFormatDlg`、`TitleInfoDlg`、`VisualizationDlg`、`ZFontDialog` |
| 分组/筛选 | `ZQGroupBySettingDlg`、`GroupsDlg`、`SyncGroupsDlg`（+ `ZQtTextFilterMenu/EnumFilterMenu/DateFilterMenu/NumbericalFilterMenu/AbnormalFlagFilterMenu/MainFilterMenu/ProgressDelayNumbericalFilterMenu`） |
| 计划编辑 | `ZConfirmPlanDlg`、`ZQtChangeCountTimeDlg`、`BatchCopyByNumDlg`、`BatchCopyByPartitionDlg`、`BatchGenerationDlg`、`PartitionSettingDlg`、`MilestoneAdjustDlg`、`MilestoneCompletenessDlg`、`MilestoneTracebackDlg` |
| 资源 | `ResDBEditDlg`、`ResDBPathSettingDlg`、`ResFormDlg`、`ResListDlg`、`ResBalanceDlg`、`ResLimitedDlg`、`ResGraphPageDlg`、`NormDataDlg`、`NormDBEditDlg` |
| 进度/分析 | `ZQtScheduleAnalyzeDialog`、`ZQtHistoryProgressDlg`、`ZQtFrozenProgressDlg`、`ZQtPeriodPlanDateRangeDialog`、`ZQtDateRangeDialog`、`ProgressDialog`、`ZProgressHintDialog` |
| 网络图 | `ZQtIntelligentLayoutDlg`、`ZQtIntelligentLayoutPrintDlg` |
| 云/授权/关于 | `CloudServerDlg`、`AboutDlg`、`FileInfoDlg`、`ShortcutAssistantDlg`、`FindDlg`、`NoteDialog` |
| 通用输入件 | `ZFDateDialog`/`ZFDateRangeDialog`/`ZFNumbericalDialog`/`ZFTextDialog`（+ `ZQtF*` 版本）= 统一"就地编辑弹层" |

> **模式**：「`ZQtF*` 系列」是**统一的就地编辑弹层**（日期/日期区间/数值/文本），网格里双击就弹这一个族，
> 而不是每个单元格自造控件 —— 这是网格手感一致的关键（gaeа 的 `CustomFieldCell` 类做法可对照）。

---

## 8. 交互模型

| 交互 | 事实 | 证据 |
|---|---|---|
| 右键上下文（结构操作） | 12 项固定序：添加工作 / 升级 / 降级 / 上移 / 下移 / 成组 / 解组 / 复制 / 粘贴 / 格式刷 / 删除 / 文字颜色 | `计划/上下文菜单` `20097…20109` |
| 右键上下文（工作/画布，带"…"） | 表格和横道样式… / 文字和线条样式… / 当前工作文字和线条样式… / 执行… / **资源和统计…** / 取消显示为分区 / 插入前置工作 / 插入后置工作 / 展开折叠工作；同菜单内嵌即时校验「升级将会导致错误工作关系，请重新选择降级工作」 | `region_14053f9e2` |
| 格式刷 | 独立命令 + 光标态 + **快捷键**（`格式刷快捷键`） | `格式刷`、`cn2.txt:格式刷快捷键`、`StyleBrush.cur` |
| 列级筛选 | 每类列有专属过滤菜单：文本/数值/日期/枚举/异常标记/进度滞后数值/主过滤 | `ZQt*FilterMenu`、`FilterMenuCreater` |
| 颜色/线宽 | `SAColorMenu`+`SAColorPaletteGridWidget`（调色盘网格）、`ZQtLineWidthButton`（含下拉） | RTTI + `color_line.svg`、`font_color.svg` |
| 日期输入 | 单元格内下拉（`datetime_dropdown.svg`）+ `ZFDateRangeDialog` | qrc + RTTI |
| 光标语言 | **20 个自定义光标**：`cur_anchor`(锚点)、`DRAGCOPY`、`LEFTHAND/RIGHTHND`、`PENCIL`、`RES`(资源)、`SCRIPT`、`TIMER`、`JOBTYPE`、`DOTCROSS`、`UPDNCROW/UPDOWNCR`、`StyleBrush`、`titlename`… | qrc `/cursor/resource/*` |
| 在位编辑 | 在位编辑工作名称、在位编辑工期、双击改名 | `在位编辑工作名称`、`在位编辑工期` |
| 快捷键助手 | `ShortcutAssistantDlg`（可查全部快捷键的帮助弹窗） | RTTI |
| 拖拽 | `ZGDropGrid`/`ZQtDropGrid`（拖拽落格）、成组/解组/升降级 = 结构拖拽 | RTTI |
| 分组渲染 | `GroupHiearchyGridCell`、`CollapsedWorkRender`（折叠父工作）、`GroupIntervalGrid` | RTTI |
| 焦点/焦点错位防护 | 文案「检测到甘特图模式下但是焦点在网络图中」（0x…）：**焦点与图模式不一致时有显式提示** | 中文串 |
| AI 内联 | 图标 `ai_assistant.svg`/`ai_new.svg`；浮框 4 动作（单行/多行 × 采纳/不采纳）；键位 **Enter 采纳 / Shift+Enter 多行采纳 / Delete 单行不采纳 / Esc 多行不采纳**；埋点 20+ 条（工作项推荐、名称补全、临时工作升降级、失去焦点采纳、百分比采纳） | `浮框-*`、`失去焦点-*`、`AI图标-*`、`工作名称补全*` |
| 校验交互 | 关系合法性埋点：`建立SF关系-合法/不合法`、`建立FF关系-合法/不合法` → 非法关系**当场提示而非事后报错** | `81001…81006` |

---

## 9. 文案与引导体系（可直接对齐 gaea 的文案规范）

1. **工具提示 = 一句话说明书**（打开向导里每个入口都有）：`新建文件`→「建立一个新的计划。」；
   `导出图片`→「将计划导出为svg/png/jpg的图片格式。如果需要将文件导入word中，我们建议使用svg格式文件（矢量图文件，不失真）。」；
   `导出PROJECT`→「将计划导出为xml/mpp格式的文件，可在project中打开查看编辑。在project中选择【打开】-【浏览】，格式选择XML（*.xml）或MPP（*.mpp），然后导入…」（带步骤）。
2. **错误提示带"位置 + 原值 + 新值 + 补救办法"**：
   「第%d行工作"%s"的计划开始时间 %s 将位于项目要求开始时间（%s）之前，软件已自动将其移动至项目要求开始时间（%s）。如果希望将该工作计划开始时间设置为 %s，请更改项目要求开始时间。」
3. **编辑期硬校验**：「的计划完成时间不得早于其计划开始时间，请重新输入计划完成时间」。
4. **商业/合规提示成族**（可作反面参照）：`盗版-文件锁定提醒`、`盗版-导出限制提醒`、`盗版-打印限制提醒`（各含"点击购买"埋点）、
   `工作数量达上限-弹窗提醒`、`版本不支持-弹窗提醒`、`标记-临期续费购买`、`间隔验权-弹窗报错`、`验证失败退出`、
   `用户-云授权`、`AI 助手（弹购买页）`、`该页签功能为高级功能，请购买后使用。`
5. **迁移引导**：`进度对比分析功能已迁移至左上角【视图】下，请单击【%s】选择【进度对比分析】视图进行查看；点击确定将自动为您切换到…`
   → **功能搬家时给"一键跳转"而不是只给说明**（这条对 gaea 版本演进很有用）。
6. **客服与失败兜底**：`在%s目录下，未找到合法的系统视图文件ViewDB.gdb，请将系统视图文件放进该目录下或联系客服【010-56616031】反馈出现原因` → 把"缺什么文件、放哪、找谁"一次说全。

---

## 10. 与 gaea 现 UI 的对标（只读摸底，不动代码）

gaea 侧现状（`frontend/src`）：`SchedulePage`（42 KB）总装 + `GanttView/GanttTable/GanttToolbar` + `AoaView`(点-箭) + `PdmView` + `DcmaView` + `EvmView` + `MonteView` + `ResourcePanel` + `ReviewPanel` + `TaskInspector` + `WhatIfPanel` + `BaselinesPanel` + `TemplatesPanel` + `ExportDialog` + `ScheduleTool`。

| 能力 | 斑马 | gaea 现状 | 差距性质 | 优先级 |
|---|---|---|---|---|
| 功能区/命令表 | 数据驱动（路径+命令键+ID），可自定义、可埋点、可查快捷键 | 工具条按视图硬编码（`GanttToolbar` 等） | **架构差异**：想加"自定义功能区/快捷键助手/命令搜索"必须先有命令表 | P0（做地基，不做皮） |
| 同屏"表格+图"双栏 | 同文档双栏 + 独立显隐 + 分隔可调 | Gantt 里表格与时间轴同屏；网络图是**切视图** | 中：需要"表格+网络图同屏"能力才有斑马式编排 | P1 |
| 图模式 | 5 种（含表格网络图混合态） | Gantt / Aoa(点箭) / Pdm(单代号) / Dcma / Evm / Monte | 形态接近，缺"时标网络图/表格网络图"与"同一计划多形态并存" | P1 |
| 时间标尺 | 双标尺（工程标尺+进度标尺）+ 自定义刻度段 + 局部缩放 + 刻度字号 | 单一时间轴（`GanttTimeline`） | 专业度差距明显 | P1 |
| 图上显示开关 | 节点时间/工期/六参数/起止时间 4 开关 | 部分（六参数在 Inspect/表格） | 低成本高观感 | P1 |
| 工作信息面板 | 5 页签（概况/关系/资源和统计/高级/自定义字段）+ 高级页签含六参数与排网策略字段 + 页内红色提示条 | `TaskInspector` 单面板 | 信息架构要重排 | P1 |
| 列级筛选/分组 | 每类列专属过滤菜单 + 17 种分组维度 + 树形/合并行 | `ganttFilter`/`ganttGroup`（已有基础） | 维度与交互细化 | P2 |
| 就地编辑弹层族 | `ZF*` 统一族（日期/区间/数值/文本） | 各单元格自造（`CustomFieldCell` 等） | 一致性 | P2 |
| 颜色/线宽/字体 | 调色盘网格 + 线宽下拉 + 字体对话框 | 有基础样式能力 | 观感 | P2 |
| 打印 | 16 项设置 + 图签横竖×打印/显示色 + 智能布图/按纸张布图 + 原生预览 | `ganttExport`/`imgPdf`/`networkExport` | 输出形态差异（斑马是"重排布图"） | P2（先做"按纸张"） |
| 主题/外观 | 6 QSS + 护眼/黑白模式 + 10 档图标 + 3 档 DPI 位图 | 设计令牌体系（`design-system/`） | gaea 有更好地基，缺"护眼/黑白"两个预设 | P0（低成本高感知） |
| AI 交互 | 网格内联图标 + 浮框 + Enter/Shift+Enter/Delete/Esc 采纳 + 20 埋点 | `ScheduleDiffCard`/`ScheduleApplyRollback`/`ReviewPanel`（卡片式 diff） | **范式不同**：斑马"就地采纳"，gaea"卡片确认"；可互补（内联 + 卡片复核） | P0（拍板项） |
| 新手引导 | 打开向导（每个入口带工具提示）+ 空白计划三选一 + 模板库 | `TemplatesPanel` | 引导话术与"三选一"起步 | P1 |

> 一句判断：**斑马赢在"命令表 + 双栏 + 标尺 + 就地编辑"的专业密度；gaea 赢在设计令牌、AI 卡片化与可测试性。**
> 抄斑马不要抄"堆功能"，要抄它**用一套命令表把功能、快捷键、埋点、自定义串起来**这件事。

---

## 11. 未解 / 下一步（本轮明确没打透的）

| # | 未解 | 需要什么 |
|---|---|---|
| 1 | **页签显示顺序**未取证（路径只给出"属于哪页"，顺序在数据里或建页代码里） | 找 `SARibbonBar::addCategoryPage` 的调用序列；或读 `Ribbon@zpbdata` 默认布局 |
| 2 | **快捷键全表** | `ShortcutAssistantDlg` 的数据源（可能在 `ViewDB.gdb`/设置文件/SQLite）；或 CEF 帮助页 |
| 3 | **表格列头顺序与默认宽度** | 列定义可能随 `ViewDB.gdb`（系统视图文件）下发 → 需要样本工程 + 该 gdb |
| 4 | **主题默认值与切换 UI 位置** | 偏好设置只看到"护眼/黑白"开关；6 套 QSS 的选择入口仍未定位（可能在 CEF 设置页或注册表） |
| 5 | **CEF 侧 AI 面板协议** | 与上次结论一致：要看 `QCefView.dll` 与运行期 CEF 消息，而不是 `ZPert.exe` |
| 6 | 运行期观感（真实字号/间距/动画） | 需要启动实机截图（本轮只做静态取证，未启动进程） |

---

> **路径约定（2026-11-14 起）**：逆向证据与工具**已移出 gaea 仓库**，统一放在 `C:\AI\zpert-re\`（`evidence/`、`tools/`、`ghidra/`）。
> 本文所有 `zpert-re/...` 均指该根目录；早期两份报告（`…-distill…`、`…-network-re…`）里写的 `re-zpert/...` 是**旧路径**，等价于 `C:\AI\zpert-re\...`。
> 仓库内**只保留这份报告**，不落任何样本、二进制、扫描产物或提取资源。
## 12. 证据与产物清单

| 文件 | 内容 |
|---|---|
| `zpert-re/tools/uiindex3.py` | 区段化 + RIP 引用扫描 + 多编码解码；`--build/--str/--region/--top` |
| `zpert-re/tools/uiscan.py` | Qt 类名/objectName/快捷键/路径抽取 → `ui_dialogs.txt`/`ui_objnames.txt`/`ui_shortcuts.txt`/`ui_cn_paths.txt` |
| `zpert-re/tools/uicmd.py` | 命令键 → 页签分组 → `ui_tab_commands.txt`、`ui_tab_paths.txt` |
| `zpert-re/tools/uifind.py` | 多探针定位"某对话框/某面板"的构建区块 |
| `zpert-re/evidence/ui/index3.json` | 15,265 个代码块的字符串引用索引（5.5 MB，可反复查询） |
| `zpert-re/evidence/ui/region_140017b00.txt` | **命令/埋点注册表**（916 条：路径、命令键、命令 ID、动作 GUID、AI 埋点族） |
| `zpert-re/evidence/ui/region_140014e30.txt` | 命令名 ↔ 动作 GUID 映射（`5000000000…`，602 条） |
| `zpert-re/evidence/ui/region_140678710.txt` | 工作信息字段表（概况 / 工程量统计块 / 高级，含付费墙文案） |
| `zpert-re/evidence/ui/ui_layouts.txt` | **20 个 Qt Designer 布局树**（WorkPagesDialog 124 objects、OptionsSetCtrlPage、ScriptSetCtrlPage、PlanSetCtrlPage、GanttOptionsSetCtrlPage、FrozenProgressDialog、ResGraphDialog、ColumnSettingDialog…） |
| `zpert-re/evidence/ui/region_14053f9e2.txt` | 工作/画布右键菜单（带"…"的 9 项 + 升级校验文案） |
| `zpert-re/evidence/ui/region_1406a5e90.txt` | `ScriptSetCtrlPage` 布局树（脚本设置页） |
| `zpert-re/evidence/ui/region_140679b20.txt` | `WorkPagesDialog` 的 Qt 布局树（objectName：`labelName/lineEditName/labelDuration/tabWidget/…`） |
| `zpert-re/evidence/ui/region_140689480`… | 偏好设置字段、时间刻度设置字段、打开向导工具提示 |
| 既有证据 | `zpert-re/evidence/{rtti_plain.txt(772 类), qmeta.txt, cn2/3/4.txt}`、`.tmp_re/zpert/bugreport/`（崩溃报告器 duilib 皮肤 + `UISkin.xml` + `blue_bg/*`） |

> 附注：`bugreport.zip` 里是一套 **duilib 皮肤**（`<Window>`/`bkimage="file='blue_bg\bg_up.png'"`/`float="true"`），
> 与主程序 Qt 无关 —— 那是独立崩溃上报器，别把它当斑马主界面的皮肤体系。



---

## 13. 第二轮补证（2026-11-14 续：配置源、视觉参数、勘误）

> 本轮只做静态取证，未启动进程、未改 gaea 代码。新增证据：`region_1406ca660.txt`（快捷键助手布局）、
> `ui_qss.txt`（80 条内联样式）、`region_140361a40.txt`（主题表）、`region_1406c8490.txt`（关于框）、`region_14053f9e2.txt`（工作右键菜单）。

### 13.1 勘误：`Ribbon@zpbdata` 不是功能区布局，是**授权受限项**

`zpbdata.proto` 里的真实定义（descriptor @ `0x89a7f0`，`zpert-re/evidence/zpbdata_schema.txt`）：

```
message Ribbon            { string button = 1; int32 pert_mode = 2; }
message RestrictedFunctions {
  repeated Ribbon ribbons = 1;  repeated string sides = 2;  int64 columns = 3;
  bool view_read_only = 4; bool pert_read_only = 5; bool resource_read_only = 6;
  bool real_info_read_only = 7; bool show_watermark = 8; int32 max_works_count = 9;
  bytes watermark_pattern = 10; int64 functions = 11;
}
```

→ `Ribbon` = **单个受限按钮**（按钮键 + 图模式），挂在 `ProductInfo.restricted_functions` 下面，由**授权档位下发**。
我上一版把它写成"功能区布局的持久化"是错的；功能区结构仍由命令表（键 + 路径）驱动，布局本身没有找到独立数据源（见 §13.3 顺序复原法）。

### 13.2 授权档位直接改造 UI（这条比"功能区自定义"更影响观感）

`RestrictedFunctions` 的字段就是一套**界面能力开关**：`view/pert/resource/real_info` 各自 `read_only`、
`show_watermark` + `watermark_pattern`（水印位图，对应 qrc `/grid/resource/watermark_3581.png`）、
`columns`（可用列数上限）、`max_works_count`（工作数上限，对应文案「工作数量达上限-弹窗提醒」）、`functions`（功能位图）。
**同一份安装包，授权档位不同 → 界面能看见/能编辑的东西不同**。做对标时"高级功能请购买"这类占位不是营销文案，是**渲染分支**。

### 13.3 页签/面板顺序：用"路径首次出现地址"复原（部分解决 §11-1）

命令表里路径与命令键**按建表顺序排列**，因此首次出现地址 ≈ 界面顺序。`计划` 页签实测面板序：

| 序 | 面板 | 首现地址 | 面板内命令（节选） |
|---|---|---|---|
| 1 | **视图** | `0x1400194da` | 计划编制（9 个内置视图：计划编制/进度管控/期间计划/期间进度报告/资源/工程量计划/进度对比分析/计划变动分析/里程碑列表）、保存视图、视图管理、上传云视图、时标网络图、横道图、逻辑网络图、单代号网络图；子组「分组」17 项 |
| 2 | **编辑** | `0x140019eae` | 复制、粘贴、删除、格式刷、查找、分类 |
| 3 | **格式** | `0x14001a06a` | 字体、字号、字体颜色、填充、线条宽度 |
| 4 | **大纲** | `0x14001a17e` | 折叠、升级、降级、上钻、下钻、上移、下移 |
| 5 | **插入** | `0x14001a3e2` | 工作、关系、里程碑、选择 |
| 6 | **高级** | `0x14001abe6` | 推导/排网相关命令（`20075`/`20083`/`20090` 组：首日是否参与计算-是/否、预警阈值 15/30/60/100 天、预警天数降低 5/10 天、关闭后置/下级工作推荐等） |
| — | 上下文菜单 | `0x14001afa8` | 12 项（见 §8），非面板 |

`资源` 页签面板序（按首现）：**资源图设置**（`0x14001b3fe`）→ **资源库维护**（`0x14001b604`）→ **定额库维护**（`0x14001b6e2`）→ **智能优化**（`0x14001b7c0`）→ **资源统计表**（`0x14001b97c`）→ **数据**（`0x14001ba5a`）。
`管控` 页签：**更改检视/反馈时间**（`0x14001bb82`）→ **历史前锋线对比分析**（`0x14001bbcc`）→ **调整**（`0x14001bdd2`，含进度对比分析/期间进度报告/计划变动分析）→ **版本**（`0x14001cbcd`）。
`视图` 页签 12 项与 `文件`/`打印` 命令集见 §4.2；`文件` 页签内部顺序仍未取证（不同区块混排）。

> 置信度：面板**集合**高（路径原文），面板**顺序**中（依赖"地址顺序≈建表顺序"这一假设，未做反例检验）。

### 13.4 视觉参数实测（这是复刻观感最硬的一手材料）

`ui_qss.txt` 共 80 条内联样式，落在 SARibbon 主题 QSS 之外，说明**新控件是"自带样式的独立弹层"**，色板明显是 **Arco Design 中性色 + 蓝主色**：

| 用途 | 实测值 |
|---|---|
| 数据表格 | `QTableWidget{border:1px solid #E2E8F0; gridline-color:#E2E8F0; font-size:14px;}`；`QHeaderView::section{background:#F8FAFC; color:#4E5969; border:none; border-bottom:1px solid #E2E8F0; border-right:1px solid …}` |
| 正文/次要文字 | `#4E5969`（次要）`#1D2129`（正文）`#FF7D00`（警示橙），字号 12px / 14px |
| 文档浮菜单 | `#ZDocFloatMenu{background:transparent;border:none}`；`#ZDocFloatMenuPanel{background:#fff; border:1px solid rgba(117,117,117,0.4); border-radius:8px}` |
| 新手教练浮层 | `#FeatureCoachTitle{font-size:14px;font-weight:600;color:#1D2129}`、`#FeatureCoachBody{12px;#4E5969}`、`#FeatureCoachCta{background:#2572E5;…}` |
| 到期续费卡 | `#RenewWidget{background:#FBF2F0}`、hover `#FFDBD3`；`#renewBtn{background:#FF6F49;color:#fff;border-radius:10px;padding:4px 12px}`、hover `#DD2F11` |
| 就地编辑 | `ZQtInplaceEdit{padding:0px;}`（网格内编辑框零内边距，保证行高不变） |
| 遮罩 | `background-color:rgba(0,0,0,0.5)`（模态蒙层） |
| 树/按钮 | `QTreeWidget{border:0}`、`QPushButton{border:none}`（去边框化） |
| 日历 | `QCalendarWidget QWidget#qt_calendar_navigationbar{background:#F0F0F0}`，按钮透明背景 + 黑色文字（中文化外观） |
| 快捷键助手 | 白 `rgb(255,255,255)` 与 `rgb(240,240,240)` **斑马纹行**、`scrollArea` 承载，两列 label 对 |

**判断**：斑马存在**两代视觉并存** —— 老壳（SARibbon/Office 主题 QSS + 位图图标）与新弹层（Arco 色板 + 14px/圆角/软阴影）。
复刻时若只抄老壳会显得"旧"，只抄新弹层会丢专业密度；gaeа 的设计令牌体系正好能同时覆盖两者。

### 13.5 主题：默认 `theme-office2013`（推断），底色来自 `Fusion` + 调色板文件

`region_140361a40`（7 条）是主题选择分支：`win7 → office2013 → office2016-blue → office2021-blue → dark → dark2`，末尾再次出现 `theme-office2013.qss`（典型 `default:` 分支写法）→ **默认 office2013**（置信中）。
`region_1405bb940`：`Fusion` + `:/resource/system_light_palette.dat` + 内联 QSS → 程序**先把 style 设为 Fusion，再加载调色板，再叠 QSS**。
与偏好设置里的「开启护眼模式 / 开启黑白模式」是两个独立维度：模式开关 + 6 套主题。

### 13.6 快捷键：助手是**运行时表**，硬编码的显式加速键只有 6 条

- `ShortcutAssistant`（`region_1406ca660`）是一个 `scrollArea + 交替底色 label 对` 的滚动表格 → 内容**运行时填充**，所以静态串里查不到键位表。
- 全样本显式加速键串仅：`Ctrl+0`、`Ctrl+1`、`Ctrl+/`、`Shift+Alt+Left`、`Shift+Alt+Right`、`Shift+Enter`（另有 AI 采纳用 `Enter`/`Delete`/`Esc` 三个语义键）。
- 推断：大部分快捷键用 Qt 数字键枚举（`Qt::Key_F2` 等）硬编码，或随 `ViewDB.gdb` 下发 → **要拿全表必须走运行期或拿到 gdb**（写进 §11 未解）。

### 13.7 `ViewDB.gdb` / `ResDB.gdb`：UI 的另一半配置源（本轮确认存在，未取得）

- 字符串：`ViewDB.gdb`、`ResDB.gdb`；RTTI：`ViewDB` 类 + `shared_ptr<ViewDB>` + 回调 `bool(shared_ptr<ViewDB>, const wstring&)`。
- 出错文案（§9 第 6 条）：目录下找不到合法 `ViewDB.gdb` 就报错并给客服电话 → **它是外部数据文件，不是内嵌资源**。
- 安装目录与 `%APPDATA%\ZebraTech` 均**没有**该文件（本机未真正运行过该软件）→ 列头顺序、列宽、系统视图定义、可能的快捷键表都在这个 gdb 里。
- **下一步要么拿到一份 `ViewDB.gdb`（用户导出/云视图下载），要么走运行期抓取**；纯静态到这儿是尽头。

### 13.8 关于框与合规展示（顺手取得）

`region_1406c8490`：产品名「斑马进度计划2026」、`版本号：0.0.0.0`（构建时替换的占位）、
「广联达科技股份有限公司 1992-2026」、「斑马进度官网」→ `https://www.zpert.com`、
投诉电话「4000-166-166 转8 再转7号键」、Qt LGPLv3 合规段落 + 三个 `github.com/GlodonOpenSrc/Qt6.7.3_Glodon` 链接（源码位置/GPLv3/LGPLv3）、按钮「关闭」，窗体 objectName = `About`。
→ 可作 gaea 关于页的字段清单参照（产品名/版本/版权/官网/反馈电话/第三方许可）。

### 13.9 本轮结论修订

| 上一版说法 | 修订 |
|---|---|
| `Ribbon@zpbdata` = 功能区布局持久化 | ❌ 实为授权受限项（`Ribbon{button, pert_mode}`），见 §13.1 |
| 页签顺序未取证 | 🟡 面板顺序已按路径首现复原（§13.3）；页签之间的顺序仍缺证据 |
| 主题"6 套 QSS + 护眼/黑白" | ✅ 补充：默认 office2013（推断）、底座 Fusion + `system_light_palette.dat`、另有 80 条内联样式（Arco 色板） |
| 快捷键未取证 | 🟡 确认助手是运行时表；显式加速键仅 6 条；全表依赖 gdb 或运行期 |

### 13.10 视觉素材实测（直接看图，不启动程序）

`C:\Program Files\ZebraTech\ZPert8\内置图片\` 是**可直接复用的素材库**，按用途分三类：

| 类别 | 文件 | 实测内容 |
|---|---|---|
| **图例（图签说明）** | `横版-显示色.png`(900×233)、`横版-打印色.png`、`竖版-显示色.png`、`竖版-打印色.png`(416×500) | 同一份 **8 项图例** 的 4 个变体（横/竖版 × 显示/打印色）。8 项依次为：**关键工作**（红色粗箭头）、**实工作**（实线箭头）、**挂起工作**（虚线箭头）、**虚工作**（点线箭头）、**里程碑**（矩形框+箭头）、**时差**（绿色波浪线+箭头）、**组件**（细线上串小矩形）、**结束旗帜**（红旗）。打印色版文字为深蓝、线条纯黑/纯红/纯绿（CMYK 友好）；显示色版文字为青色系 |
| **里程碑/标记符号** | `五角星{红,黄,蓝,绿,粉,黑}.png`、`小旗{同 6 色}.png` | 512×512 扁平拟物图标（小旗带左上高光与淡投影），6 色一套 → 里程碑按**颜色**分级/分类，星与旗两套符号可互换 |
| **施工机械/人员符号** | `挖掘机`(794×610)、`塔吊`/`塔吊2`、`吊车`、`混凝土车`/`混凝土车2`、`工人`、`安全帽`、`电梯` | **单色剪影**（纯黑填充、无渐变）→ 上色/打印/缩放到 16px 仍然可辨，明显是为"图上插图形符号"设计的 |
| 品牌/支持卡 | `斑马进度.png`(1408×683) | 橙(#E85C33 系)+深灰双色版式：左上斑马纹 logo「斑马进度 / 广联达旗下品牌」、右上深灰条「软件供应商与技术顾问支持」、中部「斑马网络计划」+ 电话 `010-56616031`、右侧微信二维码、底部橙灰条 |

**可复刻要点**：
1. 图例是**图片**而不是绘制出来的 → 打印/显示两套配色靠换图，实现成本极低；gaea 若做图例，用 SVG 组件 + 令牌换色更划算。
2. 机械/人员符号是**单色剪影**：这条决定了"图上符号"能跟着主题变色（`fill` 替换），比多色插画实用。
3. 里程碑用"星/旗 × 6 色"表达分类，而不是用文字 → 图面信息密度高的关键手段。

---

## 14. 第三轮：视觉规范从 rcc 还原（可直接照抄的令牌）

> 手段：Ghidra 反编译主题函数 + 对 `ZPert.exe` 内 Qt 资源（rcc）做 zlib 暴力还原（`zpert-re/tools/rccdump.py`、`rccraw.py`）。
> 产物：`zpert-re/evidence/rcc/`（6 套 QSS 原文 + 35 PNG + 31 SVG + 4 CUR + 索引）。
> 结论：**主题不是"整体换肤"，而是只换 SARibbon 外壳的一层 QSS**；内容区（表格/画布/弹层）另有自己的内联样式（见 §13.4）。

### 14.1 主题应用函数（Ghidra 反编译原文）

```c
void FUN_140361a40(QWidget *w, undefined4 theme)      // 调用者：FUN_140363a20 / FUN_140363950 / FUN_140341630（设置页与启动）
{
  QFile f;
  switch (theme) {
    case 0: f.setFileName(":/theme/resource/theme-office2013.qss");      break;
    case 1: f.setFileName(":/theme/resource/theme-office2016-blue.qss"); break;
    case 2: f.setFileName(":/theme/resource/theme-office2021-blue.qss"); break;
    case 3: f.setFileName(":/theme/resource/theme-win7.qss");            break;
    case 4: f.setFileName(":/theme/resource/theme-dark.qss");            break;
    case 5: f.setFileName(":/theme/resource/theme-dark2.qss");           break;
    default: f.setFileName(":/theme/resource/theme-office2013.qss");     // 默认 = office2013
  }
  if (f.open(QIODevice::ReadOnly)) w->setStyleSheet(QString::fromUtf8(f.readAll()));
}
```

→ **主题枚举顺序 `0..5` 与默认值 `office2013` 是代码事实**（上一版只是"推断"，现在钉死）。
→ 只对**传入的那个 widget** 设样式表（不是 `qApp->setStyleSheet`），所以是"局部换肤"，这也解释了为什么新弹层能自带另一套视觉。

### 14.2 六套主题与还原文件的对应（按调色板语义判定）

| 枚举 | 主题文件 | 还原产物 | 主色（功能区底 / 选中页签 / 按钮 hover） |
|---|---|---|---|
| 0（默认） | `theme-office2013.qss` | `qss_e247ef3f8142.qss` | 白底 `#FFFFFF`；页签文字 `#5779AF`；经典 Office 2013 白 |
| 1 | `theme-office2016-blue.qss` | `qss_93f3e5237f4d.qss` | 蓝条 `#225497`；页签 `#D2E4FF`；内容区 `#F1F1F1`；hover `#3E6DB6` |
| 2 | `theme-office2021-blue.qss` | `qss_26d560683c9c.qss` | 灰白条 `#E5E3E5`；选中页签 `#2760A7`；内容 `#FFFFFF`；圆角 4px |
| 3 | `theme-win7.qss` | `qss_6e683fdacf5b.qss` | 灰条 `#E3E6E8`；按钮 hover `#FDEEB3`（米黄）+ focus 边框 `#FCBF21`；应用按钮**竖向蓝色渐变** |
| 4 | `theme-dark.qss` | `qss_0603a1343583.qss` | 深灰 `#2B2B2B`；按钮 `#B2B2B2`；文字 `#242424`；hover `#A0A0A0` |
| 5 | `theme-dark2.qss` | `qss_92eda49357aa.qss` | 深蓝黑 `#192430`；面板 `#344253`；hover/按下 `#435C76`；强调描边 `#2A8DB5`；文字 `#DADADA`（**注释写明参考 AutoCAD 2021**） |

> 两套深色自带作者注释：`theme-dark` = "dark style theme"；`theme-dark2` = "参考 AutoCAD 2021 的界面配色"，
> 并列出语义：深色背景 `#192430` / 浅色背景 `#344253` / 编辑区 `#4d5973` / 字体 `#dadada` / 按钮 `#344253` / hover `#435c76` / 选中边框 `#2a8db5`。

### 14.3 跨主题一致的度量（照抄即可）

| 项 | 值 | 出现范围 |
|---|---|---|
| 页签最小宽度 | `min-width: 50px` | 全部 6 套 `SARibbonTabBar::tab` |
| 工具按钮圆角 | `border-radius: 4px` | office2016 / office2021 |
| 面板分隔条颜色 | `#BEC0C2` | office2013 / office2016 / office2021 |
| 页签未选中/选中 | 未选中文字降一档灰、选中给底色块或强调色文字；`hover:!selected` 单独给边框 | 全部 6 套 |
| 应用按钮（左上 logo） | win7 主题用 4 段 `qlineargradient` 竖向渐变（`#467FBD→#2A5FAC→#1A4088→#419ACF`），其余主题为纯色或透明 | — |
| 快速访问栏 | 5/6 套为 `transparent`；win7 主题写成 `background-color:#FF0000`（红!）——**这是一处遗留 bug 式写法**，说明该分支长期未被视觉验收 | `SARibbonQuickAccessBar` |

### 14.4 图标体系（这次拿到了真实组织结构）

| 资源 | 尺寸 | 结构结论 |
|---|---|---|
| `ribbon20/25/30/33/35/40/41/50/58/66.png` | 4080×20、5100×25、6120×30、6732×33、7140×35、8160×40、8364×41、10200×50、11832×58、13464×66 | **雪碧条：宽度 ÷ 高度 = 204**，即 **204 个图标 × 10 档尺寸**（图标是横向排列的等宽切片，不是单独文件） |
| `status_bar_16/20/24/28/32.png` | 288×16、360×20、432×24、504×28、576×32 | 同样是雪碧条：**18 个图标 × 5 档** |
| SVG（31 个） | 16/20/24/32 为主，另有 144×171、197×120 插图 | 新弹层用的矢量图标（Arco 风格，见 §13.4） |
| PNG 零散 | 1600×900（欢迎/帮助图）、814×592、805×595、322×282、240×200、128×128×4、44×44×2、25×25×2 | 引导图/示意图/状态图标 |
| 竖切条 | 12×70、11×70 各 3 个 | 滚动条/分隔拖拽条切片（状态相关） |
| CUR（4 个） | — | 配合 §8 的 20 个光标族使用 |

**复刻要点**：斑马的图标是"**一张图 204 槽 + 按 DPI 选整张**"，代价是每加一个命令要重排雪碧条；
gaeа 用 SVG 单图标 + 令牌换色更省事，但**必须保留"同一命令在多档尺寸下像素对齐"**这个约束（20/25/30/33/35/40 六档在 ribbon 折叠时混用）。

### 14.5 本轮新增可复用产物

| 路径 | 内容 |
|---|---|
| `zpert-re/evidence/rcc/qss_*.qss` | **6 套主题 QSS 原文**（带作者注释），可直接读成令牌表 |
| `zpert-re/evidence/rcc/png_*.png` / `svg_*.svg` / `cur_*.cur` | 还原的图标与素材（雪碧条未切分，切片规则见 §14.4） |
| `zpert-re/evidence/rcc/index.tsv` | 资源清单（类型/字节/文件名） |
| `zpert-re/tools/rccdump.py`、`rccraw.py`、`qsstokens.py`、`assetinv.py` | 资源还原与令牌抽取工具链 |
| `zpert-re/evidence/decomp_ui1.c` | 主题函数 / 命令表初始化 / 列设置 / 快捷键助手的 Ghidra 反编译 |

---

## 15. 第四轮：界面"处理逻辑"（不是长什么样，而是怎么跑起来）

> 手段：Ghidra 反编译具体实现函数（`zpert-re/evidence/decomp_ui2.c`）。这轮回答的是"同屏双栏怎么搭、格子里怎么编辑、右键菜单怎么装配、无边框窗口怎么拖"。

### 15.1 文档子窗口 = QMdiSubWindow + 内层 QMainWindow + 自绘 ZSplitter

`FUN_1405a1030` — `ZPertDocWindow` 构造（vftable `ZPertDocWindow::vftable`，基类 `QMdiSubWindow`）：

```c
QMdiSubWindow::QMdiSubWindow(this, parent, 0);
setAttribute(this, 0x37, true);          // Qt::WA_DeleteOnClose
setMinimumSize(this, 400, 300);          // 文档最小 400×300

// ① 外框：objectName = "centralWidget"，背景 #F9F9F9
widget->setObjectName("centralWidget");
widget->setStyleSheet("#centralWidget { background-color: #F9F9F9; }");
lay = new QHBoxLayout(widget);  lay->setContentsMargins(0, 8, 0, 0);   // 顶部留 8px

// ② 内层再套一个 QMainWindow（用来挂中央控件 + 后续 dock/status）
QMainWindow *inner = new QMainWindow(0);

// ③ 分栏：自定义 ZSplitter（不是裸 QSplitter）
ZSplitter *sp = new ZSplitter(Qt::Horizontal, 0);
sp->setContentsMargins(0,0,0,0);
sp->setHandleWidth(2);                    // 分隔条 2px
sp->setOpaqueResize(true);                // 拖动实时重绘
sp->setStyleSheet("ZSplitter { border:1px solid #E2E8F0; }"
                  "ZSplitter::handle { background-color:#E2E8F0; }");
sp->setMinimumWidth(10);

// ④ 两栏：左 = 表格控件（0x48 字节对象，FUN_1404d6cf0），右 = 画布（0x618 字节，FUN_14056d100）
QWidget *grid = FUN_1404d6cf0(sp, ctx);
QWidget *canvas = FUN_14056d100(sp, ctx);
installEventFilter(grid,   qApp->something);     // ← 应用级事件过滤器（全局快捷键/悬停）
installEventFilter(canvas, qApp->something);
sp->addWidget(grid);  sp->addWidget(canvas);

inner->setCentralWidget(sp);
QMdiSubWindow::setWidget(this, widget);

// ⑤ 4 对浮动按钮（分隔条上的折叠/切换钮），样式统一
QPushButton *b = new QPushButton(this);
b->setStyleSheet("QPushButton { border:none; back... }");   // 同一串复用 4 次

// ⑥ 分隔条被拖动 → 自定义槽，同步浮动按钮位置
connect(sp, &QSplitter::splitterMoved, this, FUN_1405d1940, ctx(this, sp, btnA, btnB));
connect(sp, <第二信号>,                this, FUN_1405d2880, ctx);

// ⑦ 默认分栏比例：1 : 1
QList<int> sizes = {1, 1};
sp->setSizes(sizes);
```

**可复刻结论**：
- 斑马是"**文档窗口里再套一层 QMainWindow**"（所以每个文档能独立拥有自己的工具栏/状态栏/停靠面板），不是"全局一个主窗 + 中央切换"。
- 分栏几何是硬编码的：**2px 分隔条 / `#E2E8F0` / 最小 10px / 默认 1:1**；拖动时用自定义槽同步两个浮动按钮，而不是靠布局自适应。
- 表格与画布都装了**应用级 eventFilter** → 键盘/滚轮这类交互是"集中拦截再分发"，这也解释了 §8 的"焦点在图还是表格"提示为何存在。

### 15.2 单元格就地编辑：6 步固定流程（`FUN_1404cd2a0`）

```c
editor = new ZQtInplaceEdit(parent);                     // : QLineEdit
editor->setStyleSheet("ZQtInplaceEdit { padding:0px; }"); // 零内边距，保证不撑高行
editor->setFont(cellFont);                               // 继承单元格字体（param_3）
editor->setAlignment(0x84);                              // Qt::AlignLeft | Qt::AlignVCenter
editor->setGeometry(QRect(cellRect.x, cellRect.y, w, h));// 宽度/高度取自单元格 rect
editor->setText(cellText);                               // 预填当前值
editor->selectAll();                                     // 全选（直接覆盖输入）
editor->setFocus(Qt::OtherFocusReason);                  // 7 = OtherFocusReason
editor->show();
connect(editor, &QLineEdit::editingFinished, this, FUN_1404d2720);  // 提交
```

→ 这套"**padding:0 + 继承字体 + 左对齐垂直居中 + 全选 + 编辑完成提交**"就是 §8 说的"就地编辑弹层族 `ZF*`"的落地形态；一致性来自**一个构造函数**，不是每个单元格各写一套。

### 15.3 上下文菜单：按"命中类型"追加条目（数据驱动，不写死 QMenu 序列）

`FUN_14053e5e0(doc, hitCtx, hitType)`：

- `switch (hitType)` 分 0..N 支，每支用 `push_back`（`FUN_140453630` / `FUN_1404539e0`）往同一个 vector 里追加菜单项；
- 每个菜单项结构 **0x48 字节**，内含：菜单项 id（如 `0x8605`、`0x2`）、文本键、以及**启用条件**；
- 启用条件从文档状态直接读，例：`if (*(char*)(doc + 0x11b4) != 0) item->enabled = 1;` —— 即"某状态下才多出这一项"；
- 菜单 id 空间与命令表 id 空间（§4.1 的 `1xxxx…9xxxx`）**不是同一套**：菜单项另有 `0x86xx` 段。
- 上层调用者：`FUN_1404ed850` / `FUN_1404edc40`（命中测试后的分发），说明"右键"走的是**统一的命中类型枚举**，而不是按控件类型分别弹菜单。

**对比 gaea**：gaea 的右键菜单目前散在各视图组件里（GanttView/AoaView/PdmView 各写一套）。斑马这条路子（命中类型 → 统一追加器）正好是 gaea 想要"右键一致 + 按上下文增删项"的现成范式。

### 15.4 CEF 桥：12 个查询覆盖登录窗的全部界面交互

登录窗（`webres/*.html`）与原生之间只有 `cefQuery` 一条通道；代码里被**逐个字符串比较分发**的查询名共 12 个（每个名在二进制里都恰好被一处代码引用）：

| 查询名 | 方向 | 作用 | 界面处理含义 |
|---|---|---|---|
| `get_login_method` | JS→原生 | 取登录方式 | 决定显示短信/密码/加密锁哪几块 |
| `startup_login_result` | JS→原生 | 启动登录结果 | 返回 `is_success` 则页面自关（§13 用过） |
| `change_login_status` | JS→原生 | 切换登录态（离线/加密锁） | 内含许可判定（破解那条线打的就是这） |
| `exec_license_manager` | JS→原生 | 拉起加密锁管理器 | `appname:'config'` 参数 |
| `close_window` | JS→原生 | 关闭登录窗 | — |
| `get_monitor_info` | JS→原生 | 显示器尺寸 | 计算窗口宽高 |
| `get_current_dpi_scale` | JS→原生 | DPI 缩放 | 100/125/150% 适配 |
| `update_cef_window` | JS→原生 | **设窗口几何/圆角/拖拽区** | `style:2148007936`、`round_r:0`、`dragable_area:[580,120,1250,680]`、`is_dragable_area_inverse:true` |
| `set_cef_window_show` | JS→原生 | 显示窗口 | 先算尺寸再显示，避免闪烁 |
| `reload_login_url` | JS→原生 | 换服务器后重载 | 服务器设置弹层用 |
| `set_cloud_server_config` | JS→原生 | 写服务器地址 | 落 `HKCU\Software\ZebraTech\斑马进度计划\Options\ServerHost` |
| `open_url` | JS→原生 | 外部浏览器打开 | 驱动下载/购买页 |

窗口几何规则（页面侧 JS 原逻辑）：基准 **1440×800** × DPI 缩放，且不超过显示器尺寸；`dragable_area` 用**反向拖拽区**（除标题/内容区外都可拖），这就是无边框窗口的"握把"实现方式。

### 15.5 本节对 gaea 的直接可用项

| 斑马做法 | gaea 现状 | 建议 |
|---|---|---|
| 文档窗口内套 QMainWindow（每文档独立工具/状态栏） | 单页 + 视图切换 | 若要做"多计划并排"，先定"每文档独立壳"还是"全局壳"（架构级） |
| 双栏 ZSplitter 2px `#E2E8F0`、默认 1:1、拖动同步浮动钮 | Gantt 内表格+时间轴一体 | 表格/网络图同屏时按此几何复刻，最容易"看起来一样" |
| 统一 `ZQtInplaceEdit` 构造（padding:0/继承字体/全选/提交） | 各单元格自造 | 抽一个 `InlineEditor` 工厂，六步照搬 |
| 命中类型 → 统一菜单追加器（0x48 字节条目 + enable 谓词） | 各视图各写 | 建"上下文动作注册表"，按 (视图, 命中类型) 增删 |
| CEF 桥 12 查询 + 反向拖拽区 | 无 CEF（纯 Web） | 无需照搬，但"先算尺寸再显示 + 反向拖拽区"两个技巧可用于桌面壳 |

---

## 16. 第五轮：视图库表结构与命中测试链路（含文件存放位置）

> 手段：Ghidra 反编译（产物 `C:\AI\zpert-re\evidence\decomp_view.c`、`decomp_viewdb.c`）。
> 本轮把"列头顺序到底存在哪"和"右键菜单凭什么弹"这两条查到了函数级。

### 16.1 视图定义（列/分组/筛选）确实存在 `ViewDB.gdb`，且走运行期适配器

读取（`FUN_1405bd410`，871 B）：

```c
QDir dir(doc + 0x48);                       // 视图库目录 = 文档对象偏移 0x48 上的路径
QString path = dir.absoluteFilePath("ViewDB.gdb");
if (!QFile::exists(path)) { 提示 "...未找到合法的系统视图文件ViewDB.gdb..." ; return; }
if (!FUN_14031e4c0(store, &msg)) { 提示 "打开系统视图文件失败！\r\n请联系客服…" ; }
```

写入（`FUN_1405c8600`，782 B）：

```c
QGuiApplication::setOverrideCursor(QCursor(Qt::WaitCursor));   // 保存时沙漏光标
path = QDir(doc+0x48).absoluteFilePath("ViewDB.gdb");
int rc = FUN_140310170(storePtr, &msg, &out);
if (rc == -1) 提示 "保存系统视图文件数据序列化失败。" ;
QGuiApplication::restoreOverrideCursor();
```

关键结论：
- **视图定义（列集合/列宽/排序/分组/筛选）不内嵌在 exe 里，而是落在文档目录下的 `ViewDB.gdb`**；`路径 = <文档目录> + "ViewDB.gdb"`。
- 真正的序列化不在这些函数里，而是 `FUN_140310170`（写）/ `FUN_14031e4c0`（读）→ 二者都通过**运行期绑定的适配器接口**转发：
  `FUN_14031e4c0` 里是 `(*(*(PTR_PTR_14109ea98))[0x10])(… , &handle, &msg)`，即调用全局接口指针的虚槽；
  `FUN_140310170` 侧再落到 `FUN_1403105d0` / `FUN_1401e0590` 等具体实现。
- 因此 **`.gdb` 的物理格式不是静态常量可判的**（不是 SQLite：导入表里没有 sqlite3；也不是 protobuf——protobuf 只用于工程文件 `ZPertFile`，见 network-re 报告）。要钉死格式，需要**一份真实的 `ViewDB.gdb` 样本**，或在有授权的机器上让它"保存视图"生成一份。
- 与之配套的两个入口：`视图管理`（`ZQtManageViewDlg`）与`保存视图`（`ZQtSaveViewDlg`）——**这两处就是"列头顺序/列宽"的唯一真源**，也解释了 gaea 侧为什么无法从 exe 里静态复原列顺序。

### 16.2 命中测试 → 菜单分发（右键为什么不按控件走）

`FUN_1404ed850`（979 B，画布/表格共用的分发器）：

```c
plVar = FUN_1403c16d0(doc, hitType, point);        // ① 命中测试：给 hitType + 点
if (plVar && objectOf(plVar) && !isSomething(doc)) {
     FUN_1404ea130(app, doc, objectOf(plVar));     // ② 命中了对象 → 对象级处理（选中/弹出对象菜单）
     return;
}
FUN_1403c41a0(doc, point, &out);                   // ③ 没命中 → 空白区处理（空白菜单/框选起点）
```

- **统一命中**：`FUN_1403c16d0(doc, hitType, pt)` 用 `(命中类型, 坐标)` 两个参数查询 —— 表格/画布/表头/分隔条共用一套查询，不是按控件各写一套。
- **两条分支**：命中对象 → `FUN_1404ea130`；未命中 → `FUN_1403c41a0`（空白右键菜单、框选起点）。§15.3 的 `FUN_14053e5e0(hitType)` 就挂在②这条上（按命中类型追加菜单项）。
- 调用者列表为空（`// callers:` 空）说明它是**经函数指针/信号表注册**进来的（Qt 槽或虚调用），不是直接 `call` —— 这也是 gaea 要仿"统一命中"时容易漏的一层。

### 16.3 文件存放位置（长期约定）

**仓库内只放文字报告，不放任何样本/二进制/产物。**

| 类别 | 位置 | 说明 |
|---|---|---|
| 报告 | `C:\AI\wubigrok\进度计划\gaea-schedule-zpert8-*.md` | 4 份：uiux（本文）/ network-re / distill / replication-assess |
| 逆向材料根 | `C:\AI\zpert-re\`（**仓库外**） | 见 `README.md`；总 33 MB / 325 文件 |
| 取证产物 | `C:\AI\zpert-re\evidence\` | `ui\`（index3.json 索引 + 区块字符串 + 布局树）、`rcc\`（6 套 QSS + 图标素材）、`decomp_*.c`（Ghidra 反编译）、字符串/RTTI/protobuf 描述符 |
| 工具 | `C:\AI\zpert-re\tools\` | 29 个脚本，路径已全部改指向新根（本轮批量修正 15 个硬编码路径） |
| 草稿 | `C:\AI\zpert-re\scratch\` | 探针脚本、早期反编译、日志、截图；可随时删 |
| 被分析样本 | `C:\Program Files\ZebraTech\ZPert8\` | 全程只读，未做任何修改 |
| Ghidra 工程 | `C:\Tools\ghidra-proj\ZPertProj`（392 MB） | 已全量分析，反编译秒级出结果 |

**命名约定**：反编译产物 `decomp_<主题>.c`（`decomp_ui1/2`、`decomp_view`、`decomp_viewdb`、`decomp_auth1`）；区块导出 `evidence\ui\region_<入口地址>.txt`；索引统一 `evidence\ui\index3.json`（旧 `index.json`/`index2.json` 已废）。

**复现命令**（路径已可直用）：

```powershell
# 查文案/字符串落在哪个代码块
python C:\AI\zpert-re\tools\uiindex3.py --str 视图管理

# 反编译任意地址（地址必须加引号）
C:\AI\zpert-re\tools\ghidra_decomp.ps1 -Out C:\AI\zpert-re\evidence\decomp_next.c -Addrs "0x1405bd410"

# 重新还原主题/图标资源
python C:\AI\zpert-re\tools\rccdump.py ; python C:\AI\zpert-re\tools\rccraw.py
```

> 纪律：任何新产物只落 `C:\AI\zpert-re\`；`git status` 里除报告外不应出现第二行。

### 16.4 本轮遗留

| # | 未解 | 需要什么 |
|---|---|---|
| 1 | `ViewDB.gdb` 物理格式（签名/压缩/加密） | 一份真实 gdb，或在授权机上"保存视图"生成 |
| 2 | `hitType` 枚举完整取值（0..N 语义） | 反编译 `FUN_1403c16d0` 的分支表（下一步） |
| 3 | `.gdb` 适配器接口的具体实现类 | 追 `FUN_1403105d0` / `FUN_1401e0590` 的 vtable |

---

## 17. 第六轮：命中类型枚举 0–6 与坐标换算（可直接照抄的交互内核）

> 产物：`C:\AI\zpert-re\evidence\decomp_hit.c`（`FUN_1403c16d0` / `FUN_1403c41a0` / `.gdb` 写路径）。

### 17.1 `hitType` 枚举（7 个取值，代码事实）

`FUN_1403c41a0(doc, point, int* out)` 把"点到哪儿了"归纳成一个整数，供 §15.3 的菜单追加器 `FUN_14053e5e0(hitType)` 使用：

| 值 | 判定条件（原文顺序） | 语义（据上下文推断） |
|---|---|---|
| **6** | `doc->mode == 6` 时**整块命中都返回 6**；仅当"不在表格矩形内 且 y < doc+0x11c"时提前返回 | **专用图模式**（`doc+0x11d8 == 6`，对照 §5 五种图之一）下的统一命中 |
| **5** | 命中 `doc+0x100` 区域，或 `x < doc->[0x100]`（左侧阈值） | **行头 / 侧表头** |
| **2** | 命中 `doc+0xb0`（表格主体矩形），或 `y >= doc->[0xb8]` | **单元格主体** |
| **0** | `y < doc->[0xac]`，或命中 `doc+0xa0` | **顶部列表头** |
| **4** | 命中 `doc+0xc0` 或 `doc+0xd0` | **分栏/分隔条**（两段，对应 §15.1 的 ZSplitter 把手） |
| **3** | 命中 `doc+0xe0`，或 `y > doc->[0xcc]` | **底部区域**（状态/滚动条带） |
| **1** | 命中 `doc` 且不属于以上任何一支（且 `mode==1` 时会先查 `doc+0xf0`） | **画布/图区** |

要点：
- **命中优先级是硬编码的 if 链**，不是空间索引；`2` 是默认值，其余靠"先判先得"。
- **图模式直接改命中语义**：`mode==6` 时整个坐标系退化成单一类型；`mode==1` 时才走 `doc+0xf0` 这条特有分支。
- 这些"区域矩形"都挂在文档对象上（`doc+0xa0/0xb0/0xc0/0xd0/0xe0/0x100`），布局变化时由布局代码回写 —— 所以**命中结果天然跟随窗口尺寸/分栏比例**，无需另建索引。

### 17.2 行命中：行高累加式反查（`FUN_1403c16d0`）

```c
if (isInEditMode(doc+8)) {
    for (row in rows(doc + 0x1c8)) {                 // 先按"行对象自带矩形"精确命中
        if (pointInRect(row->rect(row+0x10), pt)) return row;
    }
}
if (pointInTableArea(doc + 0x110, pt) && !pointInHeader(doc, pt)) {
    int off = (doc->mode == 6) ? 0x1b28 : 0x1990;    // ← 坐标系随图模式切换的两个布局区
    int y = pt.y - topOfRows(doc + off);             // 相对行区顶部
    int acc = 0, idx = 0;
    for (row in rows(doc + 0x240)) {                 // 逐行累加真实行高（虚函数 vtbl[0x50]）
        acc += row->height();
        idx++;
        if (y < acc) break;
    }
    if (acc < y) idx += (y - acc) / doc->defaultRowHeight(doc+0xcf8) + 1;   // 超出可见行→按默认行高外推
    if (idx - 1 < rowCount(doc)) return rowAt(doc + 0x240, idx - 1);
}
return 0;
```

**三个可复刻结论**：
1. **行高不是常量**：每行通过虚函数返回自己的高度，命中靠**顺序累加**（O(n)，没有前缀和也没有二分）。行数上万时这是性能隐患——gaea 若照抄必须换成前缀和/二分。
2. **默认行高 `doc+0xcf8`** 只用于"超出可见行"的外推（虚拟滚动的空白区）。
3. **坐标系有两套布局偏移（0x1990 / 0x1b28）**，由 `doc+0x11d8`（图模式）决定 —— 这就是"切图后命中错位"这类 bug 的结构性根源，也解释了 §8 里那条"检测到甘特图模式下但是焦点在网络图中"的提示为何存在。

### 17.3 `.gdb` 格式：本轮仍未定（给出否定证据）

| 检查项 | 结果 |
|---|---|
| 文件魔数 | 二进制里搜到的 `GDB9` 位于**指令字节内**（`8b 47 44 42 39`），非字符串常量 → 无可用魔数 |
| 压缩/加密依赖 | 导入表里**没有** zlib/brotli/AES 任何入口（与 `ZPertFile` 一致：brotli 是静态链进 exe 的） |
| 写路径结构 | `FUN_1403105d0` 里以 `0x10` / `0x20` 步长访问桶（`uVar9*0x10`、`uVar9*0x20`），并带 `0x4000000000000` / `0x3000000000000` 这类**打包常量** → 像是"按 key 序列化的 map"，但字段顺序/编码无法从常量反推 |
| 结论 | **必须拿一份真实 `ViewDB.gdb`**（授权机上"保存视图"生成，或让客服提供样例）才能定头部与编码；纯静态到此为止 |

### 17.4 对 gaea 的落地建议

```ts
// 建议的命中模型（照抄斑马的语义，换实现）
export type HitZone =
  | 'colHeader'    // 0
  | 'canvas'       // 1
  | 'cell'         // 2
  | 'statusBar'    // 3
  | 'splitter'     // 4
  | 'rowHeader'    // 5
  | 'mode6Whole';  // 6  —— 斑马是"某种图模式下整体一个区"，gaea 若不做这种模式可删
```
- 命中判定**按优先级 if 链**写，顺序就是 §17.1 表里的顺序；不要先算最具体的（行/列）再回退，否则分隔条永远抢不到。
- 行反查**别照抄顺序累加**：`rowHeights` 建前缀和 + 二分，O(log n)。
- `mode` 千万别和坐标偏移耦合（斑马那两套 0x1990/0x1b28 就是耦合证据）；gaea 应让"视图几何"只有一个真源，图模式只影响渲染。

### 17.5 本轮新增/变更文件（位置按 §16.3 约定）

| 文件 | 位置 | 说明 |
|---|---|---|
| `decomp_hit.c` | `C:\AI\zpert-re\evidence\` | 命中枚举 + 行反查 + `.gdb` 写路径 |
| `decomp_view.c` / `decomp_viewdb.c` | 同上 | 上轮：视图库读写、适配器 |
| 报告 | `进度计划\gaea-schedule-zpert8-uiux-re-20261114.md` | 本节即 §17 |

> 仓库 `git status` 仍应只有报告一行；所有 `.c` 反编译产物留在 `C:\AI\zpert-re\evidence\`。

---

## 18. 前端逆向收口：UI 复刻页与效果图

> 做法：把前 17 节反编译得到的**几何、配色、图标雪碧条**写成一个可渲染的 HTML 复刻页，用无头 Edge 截图出效果图。
> 位置：`C:\AI\zpert-re\mockup\`（**仓库外**，按 §16.3 约定）。
> 性质：**还原性复刻**，不是原程序截图；示意数据（工程名/日期/责任人）为构造值。

### 18.1 产物

| 文件 | 说明 |
|---|---|
| `zpert-re\mockup\zpert-ui.html` | 复刻页源码（office2013 浅色主题，CSS 变量驱动） |
| `zpert-re\mockup\zpert-dark.html` | 同一份 + `body.dark2`，即 AutoCAD 2021 深色主题 |
| `zpert-re\mockup\shots\zpert-light.png` | 浅色效果图 1610×1010 |
| `zpert-re\mockup\shots\zpert-dark.png` | 深色效果图 1610×1010 |
| `zpert-re\mockup\shots\icons.png` | 图标雪碧条索引总览（204 格，用于核对图标↔命令） |
| `zpert-re\mockup\assets\ribbon20.png` 等 | 从 exe 还原的原始雪碧条（20/66px、状态栏 16/32px） |

复现命令：

```powershell
& "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe" --headless=new --disable-gpu `
  --hide-scrollbars --window-size=1610,1010 `
  --screenshot="C:\AI\zpert-re\mockup\shots\zpert-light.png" `
  "file:///C:/AI/zpert-re/mockup/zpert-ui.html"
```

### 18.2 复刻页里每条参数的出处

| 界面元素 | 复刻取值 | 依据（本报告章节） |
|---|---|---|
| 页签集合与顺序 | 文件 / 计划 / 资源 / 管控 / 协作 / 视图 / 帮助 | §4.2 命令键前缀 |
| 「计划」页签面板序 | 视图 → 编辑 → 格式 → 大纲 → 插入 → 高级 | §13.3 路径首现地址 |
| 面板内命令 | 时标网络图/横道图/逻辑网络图/单代号网络图、保存视图/视图管理/分组、复制/粘贴/删除/查找/格式刷、字体/字号/字体颜色/填充/线条宽度、折叠/升级/降级/上移/下移/上钻、工作/关系/里程碑、智能布图/按纸张/设置/预警/检查 | §4.2 命令表 |
| 功能区配色 | 底 `#FFFFFF`、页签文字 `#5779AF`（选中加深）、面板分隔 `#BEC0C2`、按钮 hover `#DCE9F7` | §14.2 office2013 令牌 |
| 深色主题 | 底 `#192430`、面板 `#344253`、hover `#435C76`、强调 `#2A8DB5`、字 `#DADADA` | §14.2 `theme-dark2` |
| 表格 | 表头 `#F8FAFC`/`#4E5969`、网格线 `#E2E8F0`、字号 14px、行高 30px | §13.4 内联 QSS + §17.2 行高 |
| 同屏双栏 | 左右各 50%（`setSizes({1,1})`）、分隔条 **2px**、`#E2E8F0`、最小 10px | §15.1 反编译 |
| 画布 | 双标尺（月 / W 周）、月刻度竖线、横道条、里程碑菱形、**前锋线虚线** | §5（双标尺、前锋线）+ §15.1 |
| 右侧泊靠 | 「预警信息」面板，滞后/进度偏差/关系冲突/里程碑四类条目 | §3 泊靠窗 + §9 文案风格 |
| 状态栏 | 行数·分组数 / 工作·关系数 / 关键工作数 / 检视日期 / 缩放 100% | §3 状态栏 + §8 缩放 |
| 图标 | 直接切 204 格雪碧条（`ribbon20.png`），大小按钮 32px / 20px、快速访问 22px | §14.4 雪碧条结构 |
| 标题栏 | 无边框自绘：应用按钮 + 快速访问（保存/撤销/重做）+ 窗口键 | §3 / §15.4 |

### 18.3 复刻页**没有**照抄的三处（有意为之）

1. **行命中的 O(n) 累加**：复刻页固定 30px 行高；真实实现是逐行累加虚函数高度（§17.2），已在 §17.4 标注"不要学"。
2. **图标↔命令的对应**：204 格没有符号名，复刻页里的映射是我按图形语义挑的（`icons.png` 可核对），**不是**程序里的真实绑定。
3. **`.gdb` 驱动的列集**：复刻页的 8 列（序号/工作名称/工期/计划开始/计划完成/前置工作/负责人/进度）是**按 §6 的字段表构造的示意列**；真实列集与列宽仍由 `ViewDB.gdb` 决定（§16.1、§16.4）。

### 18.4 前端逆向的完成度盘点

| 维度 | 状态 | 证据 |
|---|---|---|
| 技术栈与控件模型 | ✅ 完成 | §2（Qt6 + SARibbon + SAF + QWK + CEF） |
| 功能区信息架构（页签/面板/命令/ID 分段） | ✅ 完成 | §4、§13.3 |
| 视图体系与图上显示 | ✅ 完成 | §5 |
| 对话框/设置/文案 | ✅ 完成（71 个对话框类 + 20 个布局树 + 文案族） | §6–§9、§14 |
| 视觉规范（6 套主题 + 图标组织） | ✅ 完成 | §14 |
| 界面处理逻辑（双栏/就地编辑/菜单/CEF 桥） | ✅ 完成 | §15 |
| 命中与坐标换算（0–6 枚举） | ✅ 完成 | §17 |
| 表格列集与列宽 | ❌ 阻塞 | 只在 `ViewDB.gdb`（§16.1） |
| 快捷键全表 | ❌ 阻塞 | 运行时表 + 数字键枚举（§13.6） |
| `.gdb` 物理格式 | ❌ 阻塞 | 无魔数/无压缩导入，需样本（§17.3） |

> 结论：**除三处依赖外部数据文件的项之外，前端逆向可以认为收口**；这三项都需要一份真实 `ViewDB.gdb`（授权机"保存视图"生成）才能继续。

---

## 19. 第七轮：官方资料对照 + ViewDB 解密成功（三处阻塞全部解除）

> 本轮两件事：① 按用户要求去**官网与社区**取证；② 顺着社区线索在本机找到真实 `ViewDB.gdb`，**把视图库解密出来了**。
> 新增产物：`C:\AI\zpert-re\evidence\viewdb\`（3 个原始容器 + 3 个解密 JSON）、`tools\viewdb_dec.py`、`viewdb_open.py`、`viewdb_report.py`。

### 19.1 官方/社区资料（URL 可查）

| 来源 | 拿到的信息 |
|---|---|
| [【学习】斑马进度计划软件常用快捷键介绍](https://bbs.zpert.com/t/topic/41) | **官方 17 条快捷键**：`Ctrl+N` 新建、`Ctrl+S` 保存、`Ctrl+Alt` 快速建组件、`Ctrl+A` 全选、`Ctrl+Z` 撤销、`Ctrl+Y` 恢复、`Ctrl+左键节点代号` 连关系、`Ctrl+左键` 多选、`Ctrl+左键双击竖线` 断关系、`Ctrl+C/V` 复制粘贴、`Delete` 删除、`Shift+拖拽` 改工期、`Ctrl+F` 查找替换、`Ctrl+Shift+P` 打印设置、`Ctrl+P` 打印预览、`Ctrl+Tab` 多计划窗口切换、`中键` 缩放/拖拽 |
| [斑马进度计划软件中自定义视图功能如何使用](https://bbs.zpert.com/t/topic/16377) | **视图 = 屏幕信息 + 列表信息 + 分组信息 + 筛选信息** 四要素；三种屏幕形态 = **表格 / 表格+网络图 / 表格+横道**；**列的新增/关闭走"表头右键 → 插入列/隐藏"**，列设置改表头名，自定义字段加列；视图分四类：系统内置/自定义/本计划/云；云视图需企业管理员 |
| [斑马进度计划软件系统学习文档（依照最新2025）](https://bbs.zpert.com/t/topic/120) | 完整功能地图（编制/管控/资源/AI/看板/交互），并列出子帖：设置列别显示、添加自定义列、设置列宽及行高、护眼模式与黑白模式、保存文件格式说明、备份功能、智能调图 |
| [下载-登陆说明](https://bbs.zpert.com/t/topic/65098) | 官方安装包：`https://zpert.oss-cn-beijing.aliyuncs.com/ZPert_Setup_2026.exe`（Win7/8 用 `_2025`）；短信/密码两种登录 |

**与静态取证的交叉验证**：官方帖说的"三种屏幕形态"正好对应 qrc 里的 `gantt/pert/grid_pert` 三套四态图标（§5）；"表头右键插入列/隐藏"对应 `ZColumnSettingDialog` 与 `ZQt*FilterMenu`（§7.3、§8）；快捷键里 `Ctrl+Tab` 印证 §3 的多文档标签、`Ctrl+P / Ctrl+Shift+P` 印证打印面板（§4.3）。

### 19.2 解密链路（这次是真解开了）

**① 文件位置**：注册表 `HKCU\Software\ZebraTech\斑马进度计划\Options\ResDBPath = C:/Users/wubi/Documents/ZPert`，该目录下安装包自带：

| 文件 | 大小 | 解密后 |
|---|---|---|
| `ViewDB.gdb` | 2,989 B | **150,828 B JSON**（视图库） |
| `ResDB.gdb` | 5,101 B | 32,247 B JSON（资源/定额库） |
| `BimDeco.gzp` | 9,229 B | 77,451 B JSON（示例工程，含 tasks/links/calendars/views） |

**② 容器结构**（与 `.zpet` 同一套 `zpfdata.ZPertFile`）：`type=2` + `args{brotli_original_size, aes_iv(16B)}` + `field3 = AES(brotli(json))`。

**③ 密钥不是常量推导，而是运行期算出来的**：反编译给出整条 KDF —— `FUN_1401f5710` 里拼出 **36 字节 seed**（UTF-16 中文标语"版权保护…"），`FUN_1401f5db0` 做 **SHA-256**，再用 **MT19937** 生成 16 字节随机数与数据异或；`FUN_14031ed50/14031ecb0` 是 AES 的状态推进（OFB 形态）。

**④ 抓密钥**：直接用 Frida 挂 `FUN_14031ed50(ctx, key32, iv16)`，程序启动时自己把参数交出来：

```
key32 = 013a156cb1660e3c469c69bfaf4acf19017f79ebb7c60e390aaca6cb8160ca0f
ResDB  iv = 25df510dc1b9c313827ece22a12bff0c   ← 与文件 args.aes_iv 完全一致
ViewDB iv = c1e6fa403fa8ce9c22c329267ea97e7e   ← 同上
```

**⑤ 离线解密参数**：`AES-256-CBC`（key 同上，iv 取容器里的 `aes_iv`）→ 去掉 PKCS#7 填充（ViewDB 去 15 字节、ResDB 去 10 字节）→ `brotli` 解压 → JSON。三个文件全部成功。

> **安全结论**：密钥对所有机器、所有文件都一样（同一把 key 既解安装包自带 gdb、也解示例工程），IV 每文件随机但**明文存在容器里**。也就是说 `.zpet/.gdb` 的"加密"只防明文浏览，不防解密——拿到 key 后任何人可读。**gaeа 若做类似容器，别用固定密钥 + 明文 IV。**

### 19.3 解密结果一：9 个内置视图（与命令表推断完全一致）

`ViewDB.gdb.json` → `views_list`（9 个），每个视图的键：
`id / name / view_mode / view_source / is_show_grid_editor / is_show_pert_editor / pert_mode / display_mode / groupby_setting / view_columns_info / lock_col / index_column_width / column_filters`

| # | 视图 | 表格 | 图 | 冻结列 `lock_col` | 索引列宽 |
|---|---|---|---|---|---|
| 0 | 计划编制 | ✅ | ✅ | 3 | 40 |
| 1 | 进度管控 | ✅ | ✅ | 3 | 40 |
| 2 | 期间计划 | ✅ | ✅ | 1 | 40 |
| 3 | 期间进度报告 | ✅ | ✅ | 1 | 40 |
| 4 | 资源 | ✅ | ✅ | 3 | 40 |
| 5 | 工程量计划 | ✅ | ✅ | 3 | 40 |
| 6 | 进度对比分析 | ✅ | ❌ | 3 | 40 |
| 7 | 计划变动分析 | ✅ | ❌ | 3 | 40 |
| 8 | 里程碑列表 | ✅ | ❌ | 3 | 40 |

（官方 6.0 帖列了 8 个，2026 版多出"里程碑列表"，与我 §4.2 从命令表数出的 9 个一致。）

### 19.4 解密结果二：真实列集与列宽（`计划编制`）

`view_columns_info` 共 **117 项**（其中 21 项是同名"上级目录"，每个大纲层级一列）。可见列（`is_visible=true`）按顺序：

| 序 | 列名 | 宽度 | id | 备注 |
|---|---|---|---|---|
| 0 | 分区 | 40 | 10010 | `data_type=3`（枚举/标记） |
| 1 | 标记 | 40 | — | 打卡/星标列 |
| 2 | **工作名称** | **200** | — | 主列（`lock_col=3` → 索引+分区+标记 冻结） |
| 3 | 工期(工日) | 100 | — | |
| 4 | 前置工作 | 90 | — | |
| 5 | 计划开始 | 110 | — | |
| 6 | 计划完成 | 110 | — | |

其余视图的代表性可见列：
- **进度管控**：…工作部位 100 / 负责人 100 / 执行状态 100 / 实际开始 100 / 累计实际完成(%) 120 / 预计完成 100 / 实际完成 100 / **完成时间差异 120** / 偏差原因 300 / 解决措施 300
- **资源**：资源名称 120 / 资源计划总量 120
- **工程量计划**：合同单价 90 / 工程量 100 / 单位 60 / 当期计划量 120 / 一~三级部门 100×3 / 一~三级负责人 100×3 / 累计完成量 120 / 累计实际完成(%) 120
- **进度对比分析**：执行状态 100 / 实际开始·完成 / 预测开始·完成 / **关键工作 100 / 预测关键工作 100** / 开始·完成·工期差异 120 / 影响关键线路 120

> 这段直接坐实了 §16.1 的判断：**列集/列宽/分组/筛选全部来自 `ViewDB.gdb`**，exe 里没有列定义。

### 19.5 复刻页升级为"真实列"

`C:\AI\zpert-re\mockup\zpert-ui.html` 的表格已换成 `计划编制` 的真实可见列与真实宽度（40/40/40/200/100/90/110/110），并补了官方口径的**表头右键菜单**（插入列 / 隐藏该列 / 列设置… / 自定义字段… / 列宽自适应）。两套主题效果图已重渲染：

`shots\zpert-light.png`（office2013）、`shots\zpert-dark.png`（dark2，AutoCAD 2021 配色）

### 19.6 前端逆向完成度（更新）

| 维度 | 状态 |
|---|---|
| 技术栈/功能区/视图体系/对话框/视觉/交互/处理逻辑/命中 | ✅ 完成（§2–§17） |
| **表格列集与列宽** | ✅ **解除**（§19.4 真实数据） |
| **快捷键** | ✅ **解除**（§19.1 官方 17 条 + §13.6 程序内建） |
| **`.gdb` 物理格式** | ✅ **解除**（§19.2 容器 + KDF + 解密参数） |
| 运行期观感（动画/真实间距） | ⚠️ 仅复刻页近似，未做真实截图对比 |

> 结论：**前端逆向到此可以判定完成**。剩余只有"真机截图逐像素比对"这类验收工作，不再有需要逆向的未知项。
