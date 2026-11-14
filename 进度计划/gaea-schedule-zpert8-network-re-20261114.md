# 斑马进度 ZPert8 · 网络图子系统逆向（D1 深挖）· 2026-11-14

> **状态**：D1 完成。以网络图（双代号/单代号/时标/逻辑/横道网络）为重心把整个斑马进度过了一遍，
> 拿到**点-箭数据模型 + 图面状态机 + 布图/调图流水线 + 关系合法性矩阵 + 文件容器真身**。
> 承接 `gaea-schedule-zpert8-distill-20261114.md`（D0 全产品面），本文是网络图专题与**对 D0 两处推测的修正**。
>
> **样本**：`C:\Program Files\ZebraTech\ZPert8\ZPert.exe`（7.0.706.0）
> SHA256 `C61972FAA6A1A08AB846037564CAAD7B2AB23ABCEE3A11B27885DBD24E094E68`
>
> **方法**（无 Ghidra 路线，全部本地静态）：
> 1. `pefile` 解析节表 → VA↔文件偏移；
> 2. **`.pdata` 精确函数边界**（22,402 个函数）——不需要符号表也能做"引用字符串 → 函数"归属；
> 3. `capstone` 定点反汇编 + RIP 相对引用自动解析成它所指向的字符串；
> 4. RTTI 类型描述符扫描 + MSVC COL/vtable 探测（结论见 §1.3：**自定义类无 vtable 落盘**）；
> 5. 字符串簇聚类：把同区域提示文案的所有引用聚到一个函数 → 反推子系统边界。
>
> **工具落盘**（可复用）：`re-zpert/tools/pdata.py`（函数边界 + 字符串引用反查）、
> `re-zpert/tools/disasm.py`（定点反汇编 + 字符串注解）、`re-zpert/tools/vtable.py`（RTTI/COL 探测）。

---

## 0 一页结论

| # | 结论 | 强度 |
|---|---|---|
| 1 | **双代号是独立的数据结构**，不是横道图的渲染派生：`pert_prop{dots, begin_dot_id, job_type, compute_start_time, compute_stop_time, group_id}` + `pert_links` + `dots_size` | 实测（schema 键序） |
| 2 | **显示模式是 5 个具名枚举 + 内部 mode 字段**：`时标逻辑图 / 时标网络图 / 逻辑网络图 / 单代号网络图 / 横道图`（内存中连续 `char[5][16]` 表），另有"已废弃模式"检测分支与 `mode==6` 的独立几何分支 | 实测 |
| 3 | **图面 = 一个 60+ 键的状态块**（`pert_mode / is_stagger / is_hide_unimportant_link / dot_style / arrow_style / name_pos / head_height / cr_x..crl_y / layer_height1-2 / is_show_pert / is_show_src / is_show_script / is_show_splitline / frozen_timeruler_and_partition …`） | 实测（连续键序） |
| 4 | **布图是可重入的"排版—压图幅"流水线**：智能布图 / 智能调图 / 智能位错 / 重新布局关系 / 合并多余关系 / 分离重叠工作 / 合并多余的层 / 清除多余空层 + 紧凑度 + `字号不变布图` + `先布图后缩放` | 实测（命令面 + 失败文案） |
| 5 | **关系合法性按 6 种结果埋点**：建立 FS / SS / SF-合法 / SF-不合法 / FF-合法 / FF-不合法（埋点 81001–81006）→ FF/SF 是"组合合法"判定，不是无条件允许 | 实测（埋点表） |
| 6 | **`.zp` 容器真身（D0 修正）**：`zpbdata.ZPertFile{type, args}` → `ZPertFileArgs{pert_json, images, attachments, brotli_original_size, aes_iv, co_edit_version, timestamp, lock_num}` —— 计划正文是 **JSON → brotli 压缩 → AES（iv 单独存）**，不是我 D0 里"未知加密体"的模糊说法 | 实测（protobuf 字段名串） |
| 7 | 斑马**没有**把布图结果当"每节点坐标 pins"存盘；它存的是**图面参数 + 层号**（`layer`、`layer_height1/2`），坐标由布图算法现算 | 推断（强） |
| 8 | **`.zp` 的 protobuf 只有 18 个消息**（信封/授权/协作），**计划本体是 `pert_json` 里的 JSON**；已从嵌入的 `FileDescriptorProto` 解出**完整字段号与类型**（§11） | 实测（descriptor 解析） |
| 9 | **协作协议也在这 18 个消息里**：`Request{seq,type,project_uid,diff,token}` / `Response{seq,code,message,file,diffs,member,members,project_uid}` / `Diff{pert_diff,images,deletes[],insert_at[],version}` → 按位置索引的 op 序列，不是 CRDT（§11.2） | 实测 |
| 11 | **关系合法性已伪代码级收口**（§15）：`canConnect` 只判结构（同点/空端/层序/邻居），**不含类型参数**；类型由 `decideLinkType` 按"四侧是否已有非 FS 箭头"的位掩码判定 → FF/SF 的"不合法"真实含义是 **该点已有一支别种箭头落地**，不是搭接组合本身非法 | 实测（Ghidra 反编译 + 六个埋点文案长度全部对上） |
| 12 | **布图有硬页数窗**：一条路径页/格数 `[60,1000]`、另一条 `[1,200]`；`按纸张` 失败先回滚尺寸再报错（67 字文案） | 实测（§15.7） |
| 14 | **布图分两段**：`0x140507510`+`0x14051cf50` 是**按目标纵横比的迭代拟合器**（容差 0.003 / 0.03），不是分层；分层与时标映射不在这条路径上（§16.1） | 实测（Ghidra） |
| 15 | **`Dot[+0x04]` = 点在 dots 数组的下标**（`FUN_1401f5540` 统一重排），因此 `canConnect` 的排序键是**数组先后**而非层号（§16.2，更正 D1） | 实测 |
| 16 | **`Dot[+0x54]` 初值 -1** 是层号/状态的候选槽；`+0x38/+0x40/+0x48` 是三个侵入式链表头（§16.2） | 实测 + 推断 |
| 13 | **建关系允许端点不存在**：程序自动新建点（编号递增）并在冗余时合并重复点 ⇒ 双代号不需要用户手摆节点 | 实测（§15.5–15.6） |
| 10 | **授权模型带功能闸**：`RestrictedFunctions{columns,view_read_only,pert_read_only,resource_read_only,real_info_read_only,show_watermark,max_works_count,watermark_pattern,functions}` → 免费版限工作项数就靠 `max_works_count`（§11.3） | 实测 |

---

## 1 子系统地图

### 1.1 代码分布（.pdata 实测）

网络图/进度 UI 的逻辑集中在 **文件偏移 0x4f8000–0x541500**（VA `0x1404f8000`–`0x140541500`，约 330KB），
其中按字符串簇归属可分辨出这几组函数（地址 = 函数起点，`size` 由 .pdata 给出）：

| 功能簇 | 代表函数 | 证据（该函数引用的文案） |
|---|---|---|
| 子网络下钻/展开、AI 推荐、里程碑加入校验 | `0x1404f8ef6` | `请选择一个子网络`、`全部展开`、`关系的不合条件，该里程碑不能加入`、`关系的首尾节点必须连接到工作！` |
| 视图/列/筛选/期间计划/图幅压实 | `0x1404ff7cc` | `控制节点`、`存在多余逻辑关系，是否将其删除 ?`、`当前网络图宽度较小,是否按【单位】及【步长】设置调整宽度？` |
| 下钻/上钻 + 模板/标准工序 | `0x140515da6` | `请选择一个工作或里程碑进行下钻`、`已显示全部计划内容，不在下钻状态中`、`标准工序` |
| **布图/视图渲染主体** | `0x14051c5e0` | `当前字号下无法将图幅压缩至指定纸张大小和数量…1.使用更大的纸张；2.改变字号大小;3.使用更紧凑的布图设置`、`退出期间计划`、`{"一般检查得分":"%s","投标检查得分":"%s"}` |
| 结构变更（升级/降级/上移/下移/串并行/父子） | `0x140533c4a` | `该工作将连接到节点`、`该工作将与节点`、`断开联系`、`上移后存在不合理的关系，无法上移`、`{"增加类型":"串行工作"} / 并行工作 / 子工作 / 父工作` |
| 图面设置/右键菜单项注册 | `0x140533c4a` 同簇 | `年-月`、`月-天`、`日-时`、`刻度线疏密`、`不画/画小/画大刻度线`、`画工程历线`、`时间刻度…`、`图注…`、`分区…`、`插入空层`、`底色显示/全部显示/分区显示/不显示`、`名称排列 自动/横向/纵向` |
| 单独的网络图宽度守卫 | `0x14068d13c` | `当前网络图宽度较小,是否按【单位】及【步长】设置调整宽度？` |

`0x14051c5e0` 的内部结构（capstone 实测）值得记一笔：

```
mov  rcx, [rcx+0x2280]        ; 视图对象
mov  rax, [rcx+0x188]
sub  rax, [rcx+0x180]
cmp  rax, 8                   ; < 8 就走"图幅不足"分支
lea  rcx, [rbx+0x12e8]        ; 布局器/临时缓冲
call 0x1402c0ca0
mov  eax, 0x1b40              ; 模式 6 的几何块
mov  ecx, 0x19a8              ; 非模式 6 的几何块
cmp  dword ptr [rbx+0x11d8], 6
cmovne eax, ecx
movsd xmm8, [rax+rbx]         ; 取该模式的"图幅宽度/高度"基准
```

→ **mode 存在 `+0x11d8`（int）；每个模式有自己的一份几何基准块**（0x1b40 / 0x19a8），
且 `mode==6` 走单独分支 ⇒ 内部 mode 值不止 5 个具名模式。

### 1.2 类目录（RTTI 实测，网络图相关）

```
Scene                    // 网络图绘图场景
Layouter / SceneLayouter  // 布图引擎（两个：图元布局 / 场景布局）
SceneConflictChecker      // 场景级冲突检查（重叠/交叉检测）
PartRender                // 分区渲染
ZGArrowType               // 箭头类型枚举（作为类型被 RTTI 记录）
ZGanttState               // 横道/网络共用视图状态
Pert / Link / Work        // 内核对象
ZQtIntelligentLayoutDlg          // 「智能布图」对话框
ZQtIntelligentLayoutPrintDlg     // 「按纸张布图」对话框
PartitionSettingDlg              // 「分区…」对话框
GanttOptionsSetCtrlPageDlg       // 横道/网络选项页
WorksGanttSetDlg                 // 表格与横道设置
VisualizationDlg / ZVisualizationGrid  // 形象图
ZBasePertDoc / ZQtPertDoc / ZQtPertView / ZQtPertWindow / ZPertDocWindow  // 文档-视图-窗口
```

### 1.3 vtable 探测的负结果（重要，避免后续白跑）

`Layouter` / `SceneLayouter` / `Scene` 的 RTTI **TypeDescriptor 全都在**（`.?AVLayouter@@` 等，
位于 `.data`，前置 `??_7type_info` 指针 = `0x140f930f0`），但整个映像里**没有任何 dword 引用这些 TD 的 RVA**，
即 **COL（CompleteObjectLocator）与 vtable 未落盘**。结论：

- 这三类没有可枚举的虚函数表可抄（编译器已 devirtualize / 类本身非多态）；
- 后续若要摸算法，**只能按字符串簇定位函数再读汇编**（本文做法），或装 Ghidra 做伪代码。

---

## 2 显示模式（网络图的"五视图"真身）

### 2.1 具名枚举表（内存原样，`char[5][16]`）

```
0x1408e93f0  时标逻辑图
0x1408e9400  时标网络图
0x1408e9410  逻辑网络图
0x1408e9420  单代号网络图
0x1408e9430  横道图
```

语义（来自软件自带说明文案，原文）：

| 模式 | 官方定义 |
|---|---|
| 时标网络图 | `用活动的定位和长度表示活动历时的项目网络图，是含网络逻辑的横道图` |
| 时标逻辑图 | 同族；**按时间坐标**但只表达逻辑（不落资源/进度装饰） |
| 逻辑网络图 | `只按时间顺序，不按时间坐标表示，可以以最紧凑的形式完全表示清所有逻辑关系` |
| 单代号网络图 | `以节点表示工序，实箭线指明工序之间的关系，不如双代号形象直观` |
| 横道图 | `又称甘特图（Gantt chart）` |

### 2.2 运行时

- `pert_mode` 同时出现在**视图设置**与 `zpbdata.ProductInfo/Ribbon`（记忆上次用的模式）；
- 字段位置实测 `[view+0x11d8]`（int）；
- `检测到已经废弃的网络图显示模式：` —— 存在旧模式值迁移分支（老文件里的 mode 会被改名/降级）；
- `检测到甘特图模式下但是焦点在网络图中` —— 模式与焦点是**两套状态**，会打架并提示；
- `useable_view_type` —— 视图可用类型集合（不是所有模式对所有计划都可选）；
- 切换命令：`计划-网络图` / `计划-表格网络图` / `空白计划-网络图` / `空白计划-表格网络图` / `空白计划-横道图` / `空白计划-创建`（新建时的入口差异）。

---

## 3 双代号数据模型（本次最大收获）

### 3.1 `pert_prop`：点-箭本体

从 protobuf/JSON 字段串的**连续键序**（顺序即序列化顺序）实读：

```
… info_base { total_time | resources | sources_kind | kind | code | res_graphs
             | task_id_seq | next_calendar_id }
  pert_prop { begin_dot_id | dots | notes | job_type
             | compute_start_time | compute_stop_time | group_id }
  pert_links
  pen_set | color_set | p_colorset | <一堆 fontcolor/color> …
```

配套的文档级字段（同一串里，靠前）：

```
local_idx | class_code | store_type | frozen_time | plan_state | last_count_time
is_in_schedule_analysis_state | period_plan | schedule_analyse | is_auto_compute_ask_stop_time
guid | create_time | calendars | default_calendar_id | timescale | gantt_timescale
dots_size | legend | res_kinds | shared_norms | local_work_sites | local_responsible_teams
src_graphs | zpert_version | zpert_product | version_records | reference_update_log
schedule_type | first_child | index | next_sibling | progresses | groups | show_root
group_by_manager | grid_rows | current_view_idx | views_list | lock_manager | extension
```

判读：

| 键 | 含义 | 对我们的价值 |
|---|---|---|
| `dots_size` / `dots` / `begin_dot_id` | **节点（事件）数组 + 起始节点 id**；`dots_size` 是数量，说明 `dots` 是 repeated | 双代号 = 点-箭，节点是**一等持久化实体**，不是渲染中间产物 |
| `job_type` | 工作类型（`ordinary / free_time / milestone / progress / frozen / frame / note_border / parent / new / part` 这一组样式类别里取值） | 决定箭头/条的外形档位 |
| `compute_start_time` / `compute_stop_time` | **该工作的 CPM 计算开始/完成**（落库） | 斑马把计算结果也持久化，便于打开即显示而不重算 |
| `group_id` | 分区归属（分区是 group） | 分区不是 WBS 层级而是独立实体 |
| `pert_links` | 箭头集合（工作 → 关系的绘图侧表示） | 与 `Link` 逻辑关系是一对多投影（一个逻辑关系可拆成线段） |
| `timescale` / `gantt_timescale` | **网络图与横道图各有一套刻度状态** | 两视图不同时间刻度是常态，gaea 需分开存 |
| `frozen_time` / `last_count_time` | 冻结时刻 / 上次统计时刻（前锋线拉直相关） | 与 D0 的"检视/拉直"接上 |
| `progresses` / `progress_resolution` / `progress_details` | 多期进度（检视序列） | 印证 D0 结论：检视是一等模型 |
| `views_list` / `current_view_idx` / `useable_view_type` | 视图是持久化对象列表 | 印证"视图=命名快照" |
| `schedule_analyse` / `is_in_schedule_analysis_state` | 计划分析态（= 进度对比分析视图的状态） | 分析态与编辑态互斥（对应 D0 的"非进度检视状态"） |

### 3.2 `Work` 侧与网络图相关的字段

同一串字段表（Work 的持久化序列）里出现：

```
… collapsed | auto_duration | draw_type | pert_height | layer | work_site | zerendanwei
  source_count | work_code | last_plan_start_time | last_plan_stop_time | last_is_key_work
  start_percent | stop_percent | leader | leader_id | department | is_typical_deviation
  creator | … | time_history | parent_idx | external_info | custom_values | project_values
```

- **`layer`**：工作所在层号（布图的输入/输出之一）；
- **`pert_height`**：网络图里该工作的行高（每个工作可在图上有独立高度）；
- `draw_type` + `start_percent/stop_percent`：条/箭的绘制参数（与 D0 一致）；
- `collapsed` / `parent_idx`：折叠与父引用（分区/父子）。

> **判定**：斑马**不存每节点坐标**。它持久化 `layer`（层）、`pert_height`（行高）、图面参数，
> 打开时由 `Layouter` 现算几何 —— 与 gaea 现在"每节点 pins"（`aoaLayout.pins`）是**相反的策略**。

---

## 4 图面状态机（网络图/横道的全部可调项）

以下键序是从二进制里同一串字段表**原样**取出的（顺序 = 结构体/JSON 顺序），
按功能分组给出，标注实测的界面控件名（对话框里 `xxx_` 形式的 objectName 可反查控件）：

### 4.1 视图/图面开关

| 键 | 界面控件的原文 | 作用 |
|---|---|---|
| `pert_mode` | — | 显示模式（§2） |
| `is_show_pert` | — | 是否显示网络图 |
| `is_show_grid_editor` | `显示表格编辑(左侧/状态栏)` | 左侧表格编辑区 |
| `is_show_script` | — | 是否显示"脚本/工序说明"层（`pert_script`、`script0`、`script1`） |
| `is_show_src` | `在网络图中含/只画/不含资源曲线` | 资源曲线叠加 |
| `is_stagger` | `智能位错` | **错层排布**（相邻节点纵向错开，减少箭头交叉） |
| `is_hide_unimportant_link` | `隐藏非重要关系线` | 只保留关键/重要关系 |
| `is_show_splitline` | `显示打印分割线` | 出图分页线 |
| `frozen_timeruler_and_partition` | `冻结分区与刻度` | 滚动时钉住时标与分区列 |
| `is_flag_show` / `is_bold_out_frame` | — | 里程碑旗标 / 外框加粗 |
| `is_show_finish_ratio` | `显示完成占比` | 条内百分比 |
| `is_show_freetime` / `is_link_show_freetime` / `is_freetime_same_color` | — | 自由时差线三开关 |
| `is_link_same_style` / `is_parent_show_single_line` / `is_show_duration_in_sections` | — | 关系线统一风格 / 父工作单行 / 分区内显示工期 |
| `is_solid_milestone` | — | 实心里程碑 |
| `is_show_estimate_plan` / `is_show_estimate_plan_in_schedule_analysis_state` | — | 预测计划叠加（含分析视图下是否叠加） |
| `is_show_external_depend_work_in_timenet` | — | **时标网络图里显示外部依赖工作**（跨计划依赖） |

### 4.2 节点与箭头几何（双代号专有）

| 键 | 说明 |
|---|---|
| `dot_style` | 节点样式（界面控件 `dot_style_`） |
| `cycle_radius_` | **圆节点半径**（界面上直接叫 cycle_radius） |
| `arrow_style` | 箭头样式（`arrow_style_`）；`ZGArrowType` 为其类型 |
| `head_height` | 箭头头部高度 |
| `name_pos` / `name_style` / `name_align_` | 工作名称排布：自动 / 横向 / 纵向 + 对齐 |
| `cr_x` / `cr_y` / `crl_x` / `crl_y` | 圆节点/圆角矩形的位置与尺寸参数（c=circle, crl=circle-radius-left 一线） |
| `layer_height1` / `layer_height2` | **两层高度**（时标网络图上下双层的层高） |
| `pert_height`（Work 级） | 单个工作在图上的行高 |

### 4.3 线宽/字体的完整族（说明它是"逐元素可调"）

```
线宽：unoverkeyjob_width | unoverjob_width | link_width | milestone_width | annoborder_width
      freetime_line_width | jobsline_width | frozentime_width
高度：work_height | gantt_work_height | src_height
字体：work_logfont | num_logfont | milestone_logfont | title_logfont | script_logfont
      script1_logfont | jobtype_logfont | src_logfont | src_label_logfont
      annotation_logfont | asked_logfont | table_logfont
颜色：srcfontcolor | src_label_font_color | workfontcolor | numfontcolor | titlefontcolor
      scriptfontcolor | script1fontcolor | jobtypefontcolor | milestone_fontcolor
      unoverjobcolor | unoverkeyjobcolor | framecolor | virtualjobcolor | milestonecolor
      annobordercolor | progresslinecolor | jobslinecolor | frozentimecolor
      srcsumcolor | srcdistcolor | srcsscolor | level_colors
```

**`virtualjobcolor` 的存在证明双代号有"虚工作"概念**（虽然界面文案里没有"虚工作"三字），
即：箭线可以是实工作也可以是虚箭（仅表达逻辑），并可单独配色。

### 4.4 图签 / 图例 / 项目信息

```
project_info { gantt_cname | gantt_fname | write_password | cloud_project_id
             | cloud_project_catalog_id | cloud_project_com_id
             | gantt_denomi | denominator | date_gap }
TitleInfo    { title_group_ | pert_c_name_ | pert_f_name_ | gantt_c_name_ | gantt_f_name_
             | title_blank_ | is_rect_name_ | is_line_name_ | is_title_none_ }
legend       { pert_legend_settings_ | legend_position_settings_{visible_, position_}
             | warning_num_settings_{warning_num_} | picture_bar_settings_{picture_bar_}
             | is_show_symbol_ | symbol_style_ | describe_font_ | title_font_ }
```

- `pert_c_name_` / `pert_f_name_` = 网络图的**起始节点名 / 终止节点名**（推断：与 `gantt_c_name_/gantt_f_name_`（横道图的
  开工/竣工名称）成对出现，c=commence、f=finish；界面上这两个框在图签里，且没有任何中文串直接叫"起点/节点名"，
  所以这是**键名推断**而非文案实证）；
- `is_rect_name_ / is_line_name_ / is_title_none_` = 图签三种形态（矩形框 / 单线 / 无）；
- `date_gap` = 时间刻度步长（对应"单位与步长"里的步长）；
- `warning_num_` = 图上显示的预警个数。

---

## 5 布图 / 调图流水线

### 5.1 命令面（实测全部）

| 命令 | 语义 |
|---|---|
| `智能布图` | 从零算整体布局（进入 `ZQtIntelligentLayoutDlg` 取参数） |
| `按纸张` / `ZQtIntelligentLayoutPrintDlg` | **按纸张布图**（以图幅为目标反推间距/字号） |
| `智能调图` | 在现有布局上做局部优化 |
| `智能位错` | 相邻节点错层（`is_stagger`） |
| `重新布局关系` | 只重排关系线（不动节点） |
| `合并多余关系` | 删传递冗余（配合 `存在多余逻辑关系，是否将其删除 ?` 的确认） |
| `分离重叠工作` | 处理节点/工作重叠（`SceneConflictChecker` 的产物） |
| `合并多余的层` / `清除多余空层` / `删除当前空层` / `插入空层` | 层（layer）维度的整理 |
| `先布图后缩放` vs `字号不变布图` | **两种布图策略**：先算布局再整体缩放 / 锁字号只调间距 |
| `1：1显示` / `整图显示` / `显示整图` / `等比缩放` / `窗口缩放` / `框选放大` / `放大/缩小网络图` / `横向压缩/撑长图形` / `纵向压缩/撑长图形` | 视图变换族（不改变布局数据，只改视口/变换） |

### 5.2 图幅压实（排版失败的处置链）

原文（`0x14051ca54` 引用，函数 `0x14051c5e0`）：

> `当前字号下无法将图幅压缩至指定纸张大小和数量，建议您做如下尝试：`
> `1.使用更大的纸张； 2.改变字号大小; 3.使用更紧凑的布图设置。`

配套：`打印纸张过小，请选择较大的打印纸张！`、`当前网络图宽度较小,是否按【单位】及【步长】设置调整宽度？`、
`总工期为0时该功能不可用`。

**四维约束在数据里是实名字段**（§14 全键表实证）：

| 维度 | 字段 | 说明 |
|---|---|---|
| 紧凑度 | **`layout_density`** | 就是文案里的"更紧凑的布图设置" |
| 纵横比 | **`aspect_ratio`** | 压缩时保持的比例约束 |
| 纸张 | `paper_size` · `paper_size_mm` · `paper_name` · `orientation` · `top_margin` · `bottom_margin` · `from_page` | 目标图幅 |
| 张数 | `print_preview_horizon_page_num` · `print_preview_vertical_page_num` · `print_page_number` | "纸张大小和数量" |

→ 布局是**四维约束**（节点数/层数 × 字号 × 间距（紧凑度） × 纸张），算法目标是压进"纸张大小 × 张数"。
这条链 gaea 完全没有（gaea 的 `aoaLayout.ts` 只做固定间距的自动布局 + 手动 pins）。

### 5.3 布图输入（从字段反推）

1. 逻辑图（`Link` + 分区/父子层级）→ 分层（`layer`）；
2. 时间（时标类模式）→ 横坐标由 CPM 的 `plan_start/plan_stop` 映射（时标网络图 = 位置+长度表示历时）；
3. 图面参数（`layer_height1/2`、`cycle_radius`、`head_height`、字体度量、`date_gap`）→ 尺寸；
4. 冲突消解（`SceneConflictChecker`：重叠/交叉）→ `智能位错` / `分离重叠工作`；
5. 压实（字号 or 紧凑度 or 纸张）→ 输出。

---

## 6 关系合法性矩阵（拦截面）

### 6.1 埋点即口径

内存里有一张连续的表（同一数组里还有 `网络图编辑` 与 140001 系列 AI 埋点）：

```
81001  建立FS关系
81002  建立SS关系
81003  建立SF关系-合法
81004  建立SF关系-不合法
81005  建立FF关系-合法
81006  建立FF关系-不合法
```

> **D1 更正**：此前从 UTF-16 误读成 `18001…18006`，Ghidra 反编译里读到的相邻 id 串是 **`81001/81002/…`**
> （`0x1408e9570 = '81003'` 实测），以本文为准。

判读：**FS / SS 无条件允许；FF / SF 需要合法性判定，且结果是"允许/拒绝"两种埋点**。
这与 D0 从文案读到的 `建立FF关系-不合法`、`建立SF关系-不合法` 完全对上 —— 现在有了编号级证据。

合法性的判定依据（同簇文案）：`逻辑关系不对，两点不能连接！`、`逻辑关系不对，该工作不能加入！`、
`关系的首尾节点必须连接到工作！`、`存在循环依赖，请重新添加！`、`延隔时间不能大于9999`、
`间隔值为正整数，请重新输入！`、`若要创建FS-N的关系，请在延隔时间值之前加上负号`。

> 留给 gaea 的判断题：**FF/SF 的"不合法"到底是"会造成同层反向/自环"还是"与分区跨层冲突"**，
> 需要在 Ghidra 里读 §6.2 那个判定函数才能定死。当前已知它是**在"两点连接"层面判定**（不是全放行）。

### 6.2 交互式建关系的四条路径（都有文案）

| 路径 | 文案 |
|---|---|
| 拖拽到节点 | `左键拖拽至节点添加关系`、`该工作将连接到节点`、`该工作将与节点` |
| 双击前后 | `双击向前添加关系`、`双击向后添加关系`、`双击添加前置工作`、`双击添加后置工作` |
| 键盘 | `键盘增加网络图`、`键盘浏览网络图` |
| 断开 | `断开联系`（节点/关系解绑） |

---

## 7 交互与非图面能力（网络图视角）

| 能力 | 文案/键 | 说明 |
|---|---|---|
| **下钻/上钻（子网络）** | `请选择一个工作或里程碑进行下钻`、`请选择一个子网络`、`已显示全部计划内容，不在下钻状态中`、`上钻`/`下钻` | 双代号支持"进入某工作的子网络"的层级导航 —— gaea 完全没有 |
| 在位编辑 | `网络图在位编辑`、`在位编辑工作名称`、`在位编辑工期`、`在位编辑标题`、`表格新建工作` | 图上直接改 |
| 分区 | `分区…`、`网络图区域分隔设置`、`双击插入分区`、`右键设置分区`、`分区显示/全部显示/不显示`、`至层级N设为分区` | 分区是图上泳道 |
| 成组/解组 | `成组`、`解组`、`请选择一个父工作进行解组！` | |
| 框选/多选 | `框选放大`、`选择`、`请选择左表计划任务` | |
| 键盘 | `键盘增加工作`、`键盘增加网络图`、`键盘浏览网络图`、`快捷键助手` | 全键盘可操作 |
| 图纸密码 | `网络计划图的密码设置` | 图上锁（不是文件锁） |
| 形态 | `形象图`、`楼层形象进度图`、`交叉表`、`流水` | AEC 专有视图 |
| 打印 | `打印预览-横道图`、`按纸张`、`铺满纸张`、`适应纸张`、`按宽度/高度适应纸张`、`显示打印分割线`、`横道图打印-{纸张大小,纸张方向,页边距,标题位置,标题设置,图注位置,图注设置,时间刻度,边框线宽,边框颜色,无缩放,外边框加粗,打印确认,打印页码}` | 出图参数面极大 |

---

## 8 文件容器真身（**修正 D0 §3.3**）

D0 里我写的是"protobuf 信封 + 高熵加密体（未定）"。本轮从 protobuf 描述符字符串读到确切结构：

```
zpbdata.ZPertFile      { type, args: ZPertFileArgs }
zpbdata.ZPertFileArgs  { pert_json, images, attachments,
                         brotli_original_size, aes_iv, co_edit_version, timestamp, lock_num }
zpbdata.Attachment     { name, data }
```

**结论**：计划正文 = `pert_json`，先 **brotli 压缩**（`brotli_original_size` 记明文长度），
再 **AES 加密**（`aes_iv` 单独存在 args 里，说明是 CBC/CTR 类带 IV 的模式）。
这与我 D0 实测的字节结构完全吻合：

```
08 02                     field1 = 2 (version)
12 16 [08 <ver> 12 10 <16B>]   field2 = 22B 头，其中 16B 就是 aes_iv
1a <varint len> <高熵体>   field3 = 加密后的 pert_json
```

- 因此 `.zp/.gzp/.gdb` **同构**（只是 `type` 不同：计划 / 装饰 / 资源库 / 视图库）；
- `co_edit_version` + `lock_num` = 协作编辑版本号与锁计数；
- `timestamp` = 保存时间。

> 产品含义不变（D0 §12 仍然成立）：**不追这个格式**。但这条现在可以写进对外口径了：
> "斑马的工程文件是 protobuf 封装的 AES+brotli JSON，且带协作版本号" —— 说清了为什么互操作必须走 MPP/XML/Excel。

---

## 9 gaea 落地映射（网络图专题）

### 9.1 现状（按代码实读）

| 现有 | 位置 | 缺口 |
|---|---|---|
| 双代号视图 | `frontend/src/schedule/AoaView.tsx`(23.8KB)、`aoa.ts`(21KB) | 只有 AOA 渲染，未见"点-箭持久化模型" |
| AOA 手动布点 | `aoaLayout.ts`(15.4KB)：`AoaLayout.pins` | **策略相反**：gaea 存死坐标，斑马存层+参数现算 |
| 单代号 | `PdmView.tsx`(15.7KB) | 无时标/逻辑双模式 |
| 横道 | `GanttView.tsx` + `ganttGroup/ganttCols/ganttFilter` | 无"时标网络图（横道叠网络）" |
| 后端 | `internal/schedule/aoa*` 无（Go 侧只透传 `AoaLayout`） | 双代号无服务端模型 |

### 9.2 建议的最小可实现集（按价值排序）

**N1 点-箭模型落库（0.5 刀）**
```ts
interface AoaDot { id: string; no?: number; layer: number; workIds: string[]; isBegin?: boolean }
interface AoaArrow { id: string; fromDot: string; toDot: string; workId?: string; isDummy: boolean }
interface AoaGraph { dots: AoaDot[]; arrows: AoaArrow[]; beginDotId?: string }
```
在 `Project` 上加 `aoa?: AoaGraph`（与斑马 `pert_prop{begin_dot_id,dots}`、`pert_links` 同构），
`aoaLayout.pins` 降级为"人工微调覆盖"，**默认由 `layer` + 参数现算**（= 斑马策略）。
验收：同一计划两次打开布局稳定（层号稳定 + 参数不变即坐标不变）；pins 只覆盖非空键。

**N2 图面状态块（0.5 刀）**
把 §4 的键收成 `NetworkStyle`（`dotStyle/cycleRadius/arrowStyle/headHeight/namePos/layerHeight1/2/
isStagger/isHideUnimportantLink/isShowSrc/isShowFreetime/isLinkShowFreetime/showSplitline/
frozenTimerulerAndPartition/`+线宽字体色族）。落 `.gsched.json`，默认值给出与斑马同名片面对齐的一套。

**N3 布图流水线（1.5 刀，核心）**
```
layoutGraph(project, cpm, style, opts) -> { dots, arrows, layers, extent }
 1) 分层：拓扑层 + 分区/父子约束        (layer)
 2) 时标模式：x 由 plan_start/plan_stop 映射，len = duration
 3) 逻辑模式：x 由层深度紧凑排布（最紧凑）
 4) 虚箭插入：当箭头跨越层直连会与他箭交叉时，插入 isDummy 箭头（分离节点）
 5) 冲突消解：节点/箭头交叉检测 → 位错（stagger）/ 分离重叠
 6) 压实：compact 档位 × 字号 × 纸张 → 返回是否成功 + 失败原因枚举
```
失败原因照斑马三条给 `{code:'paper_too_small'|'font_too_large'|'need_compact', hint}`。

**N4 关系合法性（0.3 刀）**
把 D0 的 C10–C12 与本文 §6.1 合并：`isLinkLegal(fromType,toType,fromTask,toTask,lag)` 返回
`{legal:boolean, code:'self_loop'|'cycle'|'layer_conflict'|'dot_unbound'|'lag_range'}`；
UI 侧对 FF/SF 必须先判后画（对应斑马的 18003–18006 六埋点，gaea 侧同理记 6 个事件名便于对拍）。

**N5 下钻/上钻（0.5 刀）**
`AoaView` 支持"进入某工作的子网络"：把该工作的子任务集合作为子图渲染，面包屑 `上钻`。

### 9.3 明确不做的

- **不存每节点坐标到计划文件**（学斑马：存层+参数）；
- **不做"按纸张布图"的第一版**（纸张驱动反推间距是锦上添花，先做 `compact` 档）；
- **不做图纸密码**（`网络计划图的密码设置`）。

---

## 10 仍未解 & 下一步取证点位（给后续会话）

| 未解 | 需要的动作 | 精确点位 |
|---|---|---|
| `Layouter` 的实际分层算法（层号如何分配、是否做重心排序减少交叉） | Ghidra 伪代码 | 函数簇 `0x1404f8000–0x140541500`；先读 `0x14051c5e0`（含 mode 分支与图幅判定） |
| `SceneConflictChecker` 的冲突判据（矩形相交？线段相交？） | 同上 | RTTI 有类名无 vtable（§1.3），只能按字符串/常量找入口 |
| FF/SF 的"不合法"判定细节 | **已定位到函数**（§13）：`0x14050ca90` canConnect + `0x140274360` 核心判定；要公式化需伪代码 | 同上 |
| 虚箭（dummy）什么时候插入 | 同上 | `virtualjobcolor` 的消费点 + `dots_size` 的写点 |
| 时标映射的取整规则（半工作日/跨假期） | 同上 | `date_gap`、`timescale` 的消费点 |
| **"断点"是什么**：`计划存在父子结构或要求时间，断点处理可能会对计划产生影响，不建议进行处理。` —— 全库唯一一句，无第二处文案可佐证 | 同上 | 该句引用点（`0x14050c98d` 附近） |
| `控制节点`（`0x140500e06` 引用）：双代号里的"控制节点"是约束点还是人工锚点 | 同上 | 该函数 `0x1404ff7cc` 内 |
| CEF 侧 JS 桥（AI 面板 → 网络图命令） | `QCefView.dll` + 运行期 | 已知 CEF 窗口命令：`open_cef_window/get_cef_window_info/set_cef_window_show/update_cef_window` |

**把 Ghidra 装上就绪**：`GHIDRA_INSTALL_DIR` 指向 Ghidra 12.1.x 解包目录 + 本机 JDK（REA `binary_session`
当前报 `not_configured`）。装好后本文 §10 前五行可一次做完；本文已给出全部函数起点地址，不必重新定位。

---

## 11 `.zp` 完整 protobuf schema（FileDescriptorProto 实证）

protobuf C++ 生成代码会把整个 `FileDescriptorProto`（`.proto` 的二进制自我描述）作为字符串常量嵌进 `.rdata`。
在文件偏移 `0x89a7f0` 找到 `zpbdata.proto` 的描述符并解出 **18 个消息、无顶层枚举**（工具：
`re-zpert/tools/protodesc.py`；产物：`re-zpert/evidence/zpbdata_schema.{txt,json}`）。

### 11.1 信封与文件

```
message ZPertFile {                        message ZPertFileArgs {
  int32       type            = 1;           int32  brotli_original_size = 1;
  ZPertFileArgs args          = 2;           bytes  aes_iv               = 2;
  bytes       pert_json       = 3;           int32  co_edit_version      = 3;
  repeated bytes images       = 4;           int64  timestamp            = 4;
  repeated Attachment attachments = 5;       string lock_num             = 5;
}                                          }
message Attachment { string name = 1; bytes data = 2; }
```

→ **计划正文在 `ZPertFile.field 3 = pert_json`**（我 D1 早期从字段名串读到顺序，此处补上字段号）。
`args` 里的 `aes_iv` 与 `brotli_original_size` 决定了解密链：**AES 解密 → brotli 解压 → JSON**。

### 11.2 协作协议（云/多人）

```
message Request  { string seq=1; int32 type=2; string project_uid=3; bytes diff=4; string token=5; }
message Response { string seq=1; int32 code=2; string message=3; bytes file=4;
                   repeated bytes diffs=5; string member=6; repeated string members=7;
                   string project_uid=8; }
message Diff     { string pert_diff=1; repeated bytes images=2;
                   repeated int32 deletes=3; repeated int32 insert_at=4; int32 version=5; }
```

判读：**`Diff` 是按位置索引的插入/删除 op 序列**（`insert_at[]` + `deletes[]`）+ 一个 `version` 做乐观并发，
`pert_diff` 是字符串形式的差异描述。这不是 CRDT/OT，而是"版本号 + 位置 op"的简单模型。
> gaea 若要做多人改计划，这个模型比 CRDT 好实现：**plan 版本号 + op 列表 + 冲突时整份回传**。

### 11.3 授权与功能闸

```
message RestrictedFunctions {
  repeated Ribbon ribbons=1; repeated string sides=2; int64 columns=3;
  bool view_read_only=4; bool pert_read_only=5; bool resource_read_only=6;
  bool real_info_read_only=7; bool show_watermark=8; int32 max_works_count=9;
  bytes watermark_pattern=10; int64 functions=11;
}
message ProductInfo { string product_id=1; string product_name_suffix=2;
  repeated string aecore_spids=3; repeated string google_spids=4;
  RestrictedFunctions restricted_functions=5; int64 stop_date=6; int64 start_date=7; }
message ProductInfoDatabase { repeated CheckLockMetadata check_locks=1;
  repeated string permanent_locks=2; repeated ProductIdToInfoEntry product_id_to_info=6;
  AdvancedFunctionSpecialStyleButton advanced_style_button=7; }
message Ribbon { string button=1; int32 pert_mode=2; }   // 记住上次用的显示模式
message ZMachileId { … }  message ZSDKLicense { … }  message ZLicenseFile { … }
message ProductFunctionList { … }
```

产品情报（不抄，但要知道它在防什么）：

- **`max_works_count`** —— 免费/试用版**按工作项数限流**（这在 D0 的 `工作数目达到20项` 文案上有对应：AI 推荐样本上限）；
- `view_read_only / pert_read_only / resource_read_only / real_info_read_only` —— **按视图粒度只读**；
- `show_watermark + watermark_pattern` —— 授权异常时**在图面上打水印**（对应 `盗版提醒`、`piracy_limited.html`）；
- `Ribbon{button, pert_mode}` —— 连"上次停在哪个显示模式"都写进产品库。

---

## 12 `pert_json` 的 JSON 结构（键序实证）

### 12.1 方法

计划本体是 JSON，写入方用的是 rapidjson。键名字面量以 NUL 分隔连续排布在
`.rdata` 的一段池里（VA `0x140894300`–`0x140894b00`，共 **116 个键**，见附录 C），
而**代码里这些键的引用顺序 = 序列化发射顺序**。按函数聚类并保序输出，即得结构。

### 12.2 三个序列化函数（按引用站点排序还原）

**（a）文档主体 · `0x140245749`**

```
gczero_time · gcstop_time
info_base {
  total_time · resources ·
  sources_kind { kind · code · res_graphs { task_id_seq · next_calendar_id } }
}
pert_prop {
  begin_dot_id · dots · notes · job_type ·
  compute_start_time · compute_stop_time · group_id
}
kind · code · pert_links
pen_set · color_set · p_colorset · srcfontcolor · src_label_font_color
```

**（b）颜色与项目信息 · `0x140266126`**

```
workfontcolor · numfontcolor · titlefontcolor · scriptfontcolor · script1fontcolor
jobtypefontcolor · milestone_fontcolor · unoverjobcolor · unoverkeyjobcolor · framecolor
virtualjobcolor · milestonecolor · annobordercolor · progresslinecolor · jobslinecolor · frozentimecolor
project_info { gantt_cname · gantt_fname · write_password ·
               cloud_project_id · cloud_project_catalog_id · cloud_project_com_id ·
               gantt_denomi · denominator · date_gap }
pert_script · script0
jobtype_setting { task_types }
forign_name · pert_row · group_shows · gantt_style { … }
```

**（c）图面与样式 · `0x14027852d`（115 处引用，最长）**

```
srcsumcolor · srcdistcolor · srcsscolor
gantt_style { show · horz_align · vert_align · line_width · line_color ·
              show_background · name_orientation · level_colors }
styles  { ordinary · free_time · milestone · progress · frozen · frame · note_border · parent · new · part }
font_set · print
network {
  layer_height1 · layer_height2 · pert_mode · is_stagger · is_hide_unimportant_link ·
  is_show_grid_editor · is_show_pert · is_show_script · is_show_src ·
  pen_set · dot_style · arrow_style · name_pos · head_height ·
  cr · crl · cr_x · cr_y · crl_x · crl_y · is_rect_name
}
color_set · p_colorset · gantt_cname · gantt_fname
```

> 该函数里同一组键出现两遍（第二遍多了 `is_show_script` / `cr_y` / `crl_x`），
> 说明是**同一函数里的读/写两分支或多版本兼容**，不是两个结构。

**（d）视图项 · `0x1402a6e15`**：`kind · code · forign_name · color_set · p_colorset · pert_mode · is_show_grid_editor`
→ `views_list[]` 的每个元素就带这几个键（**视图记住自己的显示模式与是否显示表格编辑**，印证 D0 的"视图=命名快照"）。

### 12.3 结构判读（给 gaea 的映射建议）

| JSON 层级 | 键 | 语义 | gaea 对应 |
|---|---|---|---|
| 根 | `info_base{…}` | 项目基础/日历/资源库引用 | `Project` |
| 根 | `pert_prop{dots,…}` | **双代号点集 + 起始点 + 计算时间** | 新增 `Project.aoa` |
| 根 | `pert_links[]` | 箭头集合 | `Project.aoa.arrows` |
| 根 | `gantt_style{…}` | 横道样式 | `NetworkStyle` |
| 根 | `styles{ordinary,free_time,…}` | 各状态样式 | `NetworkStyle.styles` |
| 根 | `network{…}`（§4 那 60+ 键） | **网络图全部图面参数** | `NetworkStyle` |
| 根 | `views_list[]` | 视图快照（含 `pert_mode`） | 新增 `Project.views` |
| 根 | `project_info{gantt_cname,gantt_fname,…}` | 项目名/图签名/密级 | `Project.meta` |
| 根 | `pen_set/color_set/p_colorset/*fontcolor` | 画笔/配色/字体 | `NetworkStyle.palette` |

> **结论（对 gaea 最有用的一条）**：斑马的"网络图"不是一个视图，而是
> **`pert_prop`（拓扑）+ `network`（图面参数）+ `views_list[].pert_mode`（视图 → 模式）+ `layer`（每工作层号）**
> 四件套。gaea 只要把 `aoaLayout.pins` 换成"层号 + 图面参数 + 现算"，就与斑马同构了。

---

## 13 关系合法性判定：代码级证据（部分解）

D1 只知道"FF/SF 有合法/不合法两种结果"。本轮从"两点不能连接"这条文案的引用点逆推，把调用链挖出来了：

### 13.1 调用链

```
0x1404e7cf0  关系创建入口（引用 '逻辑关系不对，两点不能连接！' @0x1404e7de3）
   …
   mov rcx, rsi
   call 0x14050ca90        ; ← canConnect(ctx)：返回 bool
   test al, al
   jne  0x1404e7e77        ; true → 继续创建
   mov  byte ptr [rsi+0x178], 1
   → 弹 '逻辑关系不对，两点不能连接！'（字符串 id 0x40）并中止
```

同族还有一个返回**错误码**的判定入口（`0x1404e7c41` 起），返回 `0x98/0x99/0x9b`（152/153/155）
并带各自的提示 id（`3` / `0x30` / `0x35`）——说明"建关系"的失败原因是一个**有限错误码集**，
不是一句通用提示。

### 13.2 `canConnect`（`0x14050ca90`）结构还原

```
bool canConnect(ctx)                                     // ctx 是带 vtable 的关系上下文
  a = ctx.vcall(2) -> (obj + 0x20)                        // 端点 A 实体
  b = ctx.vcall(4) -> (obj + 0x20)                        // 端点 B 实体
  if (a == b) return false;                               // ① 同一个点
  if (!a || !b) return false;                             // ② 端点缺失（= "节点必须连接到工作"）
  if (b->[+4] >= a->[+4]) swap(a,b);                      // ③ 按 [+4] 整数排序：小 → 大
  if (!sub_140274360(ctx, small, large)) return false;     // ④ 核心判定
  if (!ctx.vcall(0x20)) return false;                      // ⑤ 守卫
  ctx.vcall(0x18, 0);
  for (p in large->[+8 .. +0x10]) { t = sub_1401cb3d0(p, ctx); sub_1401ccb20(t, *p, 1, 0, -1); }
  for (p in large->[+0x20 .. +0x28]) { … 同上 … }          // ⑥ 把两端已有箭头逐项搬/并到新点
  return true;
```

**关键观察**：

1. 端点实体有个**整数字段 `[+4]` 用来比较先后**——在点-箭结构里最自然的解释是**层号/序号**（与 §3.2 的 `layer` 呼应）；
2. 实体上有**两个指针数组**（`+8..+0x10` 与 `+0x20..+0x28`）——对应**入箭集合 / 出箭集合**；
3. 判定分两段：先 `sub_140274360` 做"能否连"，再逐项处理两端箭头（合并重复点/续接箭头）。

### 13.3 核心判定 `sub_140274360`

```
bool sub(ctx, a, b)                                       // a、b 为已排序的两端
  if (a == b) return false;
  copy(a->[+8..+0x10])   → v1                            // A 的第一个邻居表
  copy(a->[+0x20..+0x28]) → v2                            // A 的第二个表
  for (x in v1) sub_1402d7b60(x, b);                       // 逐项判定/剔除
  for (x in v2) sub_1402d7ac0(x, b);
  if (ctx.vcall(0x20)) { … } else { 反向再来一遍 … }
  …
```

> **判读（推断，明确标注）**：`0x14050ca90 + 0x140274360` 合起来是**"能不能在这两个点之间加一支箭"**的
> 结构判定：同点、端点缺失、层序（`[+4]`）、已有邻居是否会与新箭冲突/重复。FF/SF 之所以有"不合法"分支，
> 最可能就是它们的箭尾/箭头落在与 `[+4]` 顺序相反的一端（或需要插虚箭才能表达）。
> **要把它写成 gaea 的规则需要伪代码**——这是 Ghidra 一下就能收的活（函数边界已给出）。

### 13.4 附带的两个副产品

- `0x140619141`：延隔时间校验函数，引用 `间隔值为正整数，请重新输入！`（配 D0 §3.1 的 ±9999 闸门）；
- `0x140318410` / `0x1402d7b60` / `0x1402d7ac0` / `0x1401cb3d0` 是这条链上的公共小工具（集合查重/搬运）。

---

## 14 文档 JSON 全键表（219 键）与四条修正

把 `layer_height1` 所在的键池整段（VA `0x1408949e8` 前后约 0x1400 字节）按内存顺序解出，
得到**计划文档的完整键表（219 个）**——这是 `pert_json` 的权威词汇表，比 §12 的三段序列更完整。
全表见**附录 D**。其中几条直接修正了前文的判断：

### 14.1 里程碑治理是"项目级开关"，不是硬编码

| 键 | 含义 | 前文修正 |
|---|---|---|
| `milestoneValidationRanks` | **里程碑校验范围**（按级别，对应界面 `里程碑校验范围：`） | D0 §5.4 说"必须填写偏差原因"是强制；实际是**按级别可配** |
| `completedMilestoneAttachmentRequired` | 已完成的里程碑必须上传附件（开关） | 同上：是开关，默认开 |
| `delayedMilestoneReasonResolutionRequired` | 延期/超期里程碑必须填偏差原因与解决措施（开关） | 同上 |
| `approvedPlan` | 审批通过后的锁定标志 | 对应文案"企业已审批通过的计划不允许变更计划信息" |

### 14.2 企业管控也是字段

| 键 | 含义 | 对应文案 |
|---|---|---|
| `restrictActualTime` | 是否启用"实际时间不得早于管控线" | `按照企业管控要求，实际开始时间/实际完成时间不得早于[%s]之前` |
| `actualTimeEarliestDays` | 管控线的天数阈值 | 同上 |
| `forbidStraightenForwardLine` | 禁止拉直前锋线 | `企业端已禁止审批通过的计划使用拉直前锋线功能` |

### 14.3 打印/图幅是一整块

`paper_size · paper_size_mm · paper_name · orientation · top_margin · bottom_margin · from_page ·
print_page_number · print_preview_horizon_page_num · print_preview_vertical_page_num ·
layout_density · aspect_ratio`，加上横道内容块
`bar_height · is_show_separator · is_show_links · left/right/top/bottom_show_content`，
再加表格头样式 `brush · fonts · top_hdg_font/style/bkcolor · side_hdg_* · grid_row_color ·
focus_rect_color · grid_blank_bkcolor · lock_line_color · parent_work_style`。

> 也就是说：**"按纸张布图"的输入全在数据里**（纸张 + 张数 + 紧凑度 + 纵横比 + 字号），
> 不需要额外状态。gaea 若做 N3 的压实，照这 5 个字段建 `PrintTarget` 即可。

### 14.4 云/项目身份

`catalog_name · project_name · cloud_plan_id · bim5d_plan_id · cloud_catalog_id · cloud_company_id ·
bim5d_org_id · bim5d_org_type · cloudt_project_id · tenant_id · org_id · org_type · is_cloud ·
contract_nodes · project_type · f_name`。

---

## 15 Ghidra 收口：关系合法性与节点构造（伪代码实证）

### 15.1 环境与产物

| 项 | 值 |
|---|---|
| JDK | Temurin 21.0.13+9（`C:\Tools\jdk-21.0.13+9`） |
| Ghidra | 12.1.4 PUBLIC 20260921（`C:\Tools\ghidra_12.1.4_PUBLIC`） |
| 项目 | `C:\Tools\ghidra-proj\ZPertProj`（已保存，后续查询秒级） |
| 分析耗时 | 1046 s 全量分析（导入 + 自动分析） |
| 导出脚本 | `re-zpert/ghidra/DumpDecomp.java`（可选地址 → 反编译 C + 调用者/被调用者） |
| 产物 | `re-zpert/evidence/decomp1.c`（26.6KB）/ `decomp2.c`（69.4KB）/ `decomp3.c`（7.3KB） |

### 15.2 建关系是三段式（`FUN_1404e7cf0`）

```
int createLink(doc, docNode, workA, op, workB):
  graph = doc[0x5b]                      // 点-箭图对象
  graph.vcall(0x290)(slot=2, workA)      // 登记端点 A（slot 2）
  graph.vcall(0x290)(slot=4, workB)      // 登记端点 B（slot 4）
  if (slot4 != null && canConnect(graph) == false):
      SetError(0x40);  MessageBox("逻辑关系不对，两点不能连接！");   // 0x140be2870, len=14
      return 0
  if (op ∈ {10, 12}):                     // 10 / 12 是两种"建关系"操作码
      graph.vcall(0x290)(slot=0, slot4)
      r = splice(graph, op)               // FUN_14050cc40
      if (r == -1):
          MessageBox("逻辑关系不对，不能连接！  ");                   // 0x140be29b0, len=15
          return 0
  return 1
```

- 端点用 **slot 号**存取（2 = 起点工作、4 = 终点工作、0 = 临时位）；
- 失败走统一 **错误码 0x40** + 一条消息，**不落任何图数据**。

### 15.3 `canConnect`（`FUN_14050ca90`）——注意：**它不接类型参数**

```
bool canConnect(graph):
  a = graph.vcall(0x298)(2)?.obj[+0x20]
  b = graph.vcall(0x298)(4)?.obj[+0x20]
  if (a == b || a == 0 || b == 0) return false          // ① 同点 / 端点缺失
  (lo, hi) = a[+4] <= b[+4] ? (a,b) : (b,a)             // ② 按 [+4] 整数排序（层/序号）
  if (!mergeCheck(graph, lo, hi)) return false           // ③ 合并可行性（FUN_140274360）
  if (!graph.vcall(0x20)()) return false                 // ④ 守卫
  graph.vcall(0x18, 0)
  for (arrow in hi[+8 .. +0x10])  rebindOut(arrow, graph)   // ⑤ 逐个把 hi 的箭头改挂
  for (arrow in hi[+0x20 .. +0x28]) rebindIn(arrow, graph)
  return true
```

**`FUN_140274360` 的真身（本轮一并读通）**：

```
bool mergeCheck(graph, lo, hi):
  if (lo == hi) return false
  out = snapshot(hi[+8..+0x10])          // hi 的首侧箭头
  in  = snapshot(hi[+0x20..+0x28])       // hi 的次侧箭头
  for (x in out) rebindOut(x, hi)        // 先挂到 hi
  for (x in in)  rebindIn(x, hi)
  if (!graph.vcall(0x20)()) {            // 守卫失败 → 改挂到 lo（回滚）
      for (x in out) rebindOut(x, lo)
      for (x in in)  rebindIn(x, lo)
      return false
  }
  mergeDot(graph, lo)                    // 成功 → 合并 lo
  return true
```

而两个 146 字节的小工具**不是判定函数，而是"改挂/移动箭头端点"**：

```
void rebindOut(arrow, newDot):           // FUN_1402d7b60
  old = arrow[+0x30]                      // 箭头当前所属的点（起点侧）
  remove(old[+8..+0x10], arrow)
  if (newDot) append(newDot[+8..+0x18], arrow)
  arrow[+0x30] = newDot

void rebindIn(arrow, newDot):            // FUN_1402d7ac0（对称，用 arrow[+0x28] / dot[+0x20..0x30]）
```

**由此确定点-箭的对象布局**：

| 对象 | 偏移 | 含义 |
|---|---|---|
| Dot | `[+4]` | 整数（层号/序号，用于排序） |
| Dot | `[+8..+0x10]` | 首侧箭头集合（出） |
| Dot | `[+0x20..+0x28]` | 次侧箭头集合（入） |
| Arrow | `[+0x28]` | 次端点（入端所在点） |
| Arrow | `[+0x30]` | 首端点（出端所在点） |
| Arrow | `[+0x70]` | **搭接类型**（1 = FS，见 §15.4） |
| Arrow | `[+0x48] ← [+0x80]` | 拼接时"复制时间/坐标"（§15.5） |

**关键推论（D1 §13 的收敛）**：`canConnect` 里没有任何 FS/SS/FF/SF 分支 —— 它是
**"这两个端点能不能合并成一个点"** 的判定 + 合并动作（`mergeDot`），与搭接类型无关。
类型合法性在另一路（§15.4）。D1 §13.3 里"FF/SF 不合法可能因为箭尾箭头落反"的猜测**不成立**。

### 15.4 类型判定与六埋点（`FUN_14036c640`）★ 本次最重要的一条

```
int decideType(dotA, dotB):            // 只看箭头类型 != 1 的分布；1 = FS
  m = 0
  if (dotA):
      if (any(arrow in A.first  where arrow[+0x70] != 1)) m |= 8
      if (any(arrow in A.second where arrow[+0x70] != 1)) m |= 4
  if (dotB && dotB != dotA):
      if (any(arrow in B.first  where arrow[+0x70] != 1)) m |= 2
      if (any(arrow in B.second where arrow[+0x70] != 1)) m |= 1
  t = 2
  switch (m):
    case 6:            t = 3;            // fallthrough → FF/SF 分支
    case 2: case 10: case 14:
        for (arrow in B.first):
            if (arrow.vcall(0x20)):                  // 该箭头"已落地/已确认"
                if (t == 3) { log("建立SF关系-不合法"); return 0 }   // 埋点 81004
                else        { log("建立FF关系-不合法"); return 0 }   // 埋点 81006
        if (t == 2) { log("建立FF关系-合法"); return 2 }             // 81005
        if (t == 3) { log("建立SF关系-合法"); return 3 }             // 81003
        break
    case 4: case 5: case 7:
        log("建立SS关系"); return 1                                  // 81002
  default:
        log("建立FS关系"); return 0                                  // 81001
```

**证据强度**：六个埋点文案的 `len` 参数与文案长度逐一吻合（`FUN_1400decb0(str, len)` 传的分别是
`6 / 6 / 9 / 10 / 9 / 10`，对应 `建立FS关系`=6、`建立SS关系`=6、`建立SF关系-合法`=9、
`建立SF关系-不合法`=10、`建立FF关系-合法`=9、`建立FF关系-不合法`=10），且相邻 id 串为
`81001…81006`。这不是"猜"，是反编译里读出来的分支与常量。

**结论（可写进 gaea 的规则）**：

| 掩码 m（8·A首 + 4·A次 + 2·B首 + 1·B次，位=该侧存在非 FS 箭头） | 判定 |
|---|---|
| 0 | FS（默认） |
| 4 / 5 / 7 | SS |
| 2 / 10 / 14 | FF；若 B 的首侧已有"已落地"箭头 → **FF 不合法**（拒建） |
| 6 | SF；同上条件 → **SF 不合法**（拒建） |
| 其余 | 落 default → 按 FS |

> 换句话说：**FF / SF 的"不合法"不是搭接组合本身非法，而是"该点上已经有一支别种箭头落地了"**
> —— 一个点只能承载一种"起点侧/终点侧"语义，冲突时拒绝。这正好对应双代号里"节点只能有一种
> 进出语义、否则要插虚箭"的施工习惯。

### 15.5 拼接与落库（`FUN_14050cc40`）

```
int splice(graph, op):                    // op = 10 | 12
  A = graph.slot(2).obj[+0x20]; B = graph.slot(4).obj[+0x20]
  if (op ∉ {10,12} || A == 0) return -2
  if (B == A[+0x28] || B == A[+0x30]) return -1        // 已经连着 → 拒
  if (op == 12):
      if (B == 0) { d = newDot(graph, -1); bind(A, d) }        // 自动补一个点！
      else {
          if (count(B'[+0x30].first) > 1) bind(A, B)
          if (count(B'[+0x30].second) != 0 && count(B'[+0x30].first) == 1)
              for (k in B'[+0x30].second) k[+0x48] = k[+0x80]   // 复制时间/坐标
          if (count(first) == 1 && count(second) < 1) mergeDot(graph, B)   // 合并点
      }
  else /* op == 10 */:  … 对称分支（阈值 1 / 7 等）…
  if (!graph.vcall(0x20)()) return -1
  graph.vcall(0x18, 0); rebind(A); return 0
```

判读：**建关系时允许"端点不存在"**——这时程序**自动新建一个点**（`FUN_140274ba0`），
并在合适条件下**合并重复点**（`FUN_1402729a0`）。这就是双代号图能一直保持"点-箭"形态、
不需要用户手工摆节点的原因。

### 15.6 建点 / 合并点

```
Dot* newDot(graph, pos):                 // FUN_140274ba0
  d = malloc(0x60)
  graph[+8] += 1                          // 图的号数计数器
  init(d)
  if (d[0] == -1) d[0] = ++graph[+8]      // 节点编号（递增）
  d[1] = (graph.dots.end - graph.dots.begin) / 8   // 节点序号 = 数组下标
  insertAt(graph.dots[+0x40], pos, d)     // pos = -1 → 追加
```

- **节点有两个身份**：`d[0]` = 编号（递增、可显示在图签/编号圈里），`d[1]` = 数组序号（重排后变）；
- `mergeDot(graph, d)`（`FUN_1402729a0`）把 d 的两组箭头搬到目标点上并删除 d。

### 15.7 布图主函数（`FUN_14051c5e0`）的实证

```
void layout/reflow(view, param_2, bool report, uint targetX, uint targetY):
  if (view[0x2280].contentSize() < 8) return;            // 内容还没量出来
  FUN_1402c0ca0(view + 0x12e8)
  base = (view[+0x11d8] == 6) ? 0x1b40 : 0x19a8;         // ★ 每模式一块"宽度基准"
  dVar1 = *(double*)(view + base)
  savedW = view[+0x11c8]; savedH = view[+0x1284]
  if (view[+0x11dc]):                                    // 需要重算尺寸
      real = FUN_1403abf80(view[0x2280])                 // 实际内容尺寸
      if (dVar1 < real):
          base2 = (view[+0x11d8] == 6) ? 0x1b28 : 0x1990 // ★ 每模式一块"高度基准"
          *(double*)(view + base2 + 0x18) = real
          FUN_1402c4b40()                                // 触发布图
  FUN_14051cf50(view, tmp, param_2, view[+0x1280] == 2)  // 预布局
  …
  if (view[+0x1280] == 2):
      r = FUN_140508ad0(view, real, param_2, &out)        // 分支 A
      ok = (60 <= out <= DAT_140974ec8) && (r - 60 < 941) // ★ 页/格数 ∈ [60, 1000]
      if ok: FUN_140525be0(view, savedW, r, param_2)      // 提交
      else : 回滚尺寸 + 重算 + 弹 67 字文案（0x140bf5f60, len=0x43）
  else:
      r = FUN_140507510(view, real, param_2)              // 分支 B
      ok = (60 <= out <= …) && (r - 1 < 200)              // ★ 页数 ∈ [1, 200]
      if ok: 提交（并 FUN_14050cf60(view,…)）
      else : 回滚 + 同一句 67 字文案
```

- **"每模式一块几何基准"** 落实了 §2.2 的推断：`pert_mode` 不只是选渲染样式，还选**基准尺寸块**
  （`0x1b40/0x1b28` 与 `0x19a8/0x1990` 两组，`mode==6` 用前者）；
- **张数/页数有硬上限**：一条路径 `[60, 1000]`、另一条 `[1, 200]` —— 这就是 `按纸张` 布图
  会失败的真实原因（不是"排不下"而是"超出页数窗"）；
- 失败路径**先回滚尺寸再报错**（`*(double*)(base2+0x18) = dVar1` + `FUN_1402c4b40`），
  所以点"取消"不会留下半成品布局 —— 这正是 gaea N3 该抄的失败语义。

### 15.8 gaea 可直接落地的规则（伪代码，替代 D1 §9.2 的 N4 草稿）

```ts
// 1) 结构可连（与类型无关）
function canConnect(g: AoaGraph, a: DotId, b: DotId): boolean {
  if (a === b) return false
  const A = dot(g, a), B = dot(g, b)
  if (!A || !B) return false
  const [lo, hi] = orderBy(A.layer, B.layer, A, B)     // 按 [+4] 整数（= layer）排序
  if (!neighborsCompatible(g, lo, hi)) return false    // 同点/重复箭/邻居冲突
  return true
}

// 2) 类型判定 + 合法性（斑马口径）
function decideLinkKind(g: AoaGraph, a: DotId, b: DotId):
  { kind: 'FS'|'SS'|'FF'|'SF', legal: boolean, code: 81001|81002|81003|81004|81005|81006 } {
  const A = dot(g, a), B = dot(g, b)
  const hasNonFS = (d, side) => g.arrows.some(x => x.dot === d && x.side === side && x.kind !== 'FS')
  let m = 0
  if (A) { if (hasNonFS(a,'out')) m |= 8; if (hasNonFS(a,'in')) m |= 4 }
  if (B && b !== a) { if (hasNonFS(b,'out')) m |= 2; if (hasNonFS(b,'in')) m |= 1 }
  let kind = 'FS', code = 81001
  if (m === 4 || m === 5 || m === 7) { kind = 'SS'; code = 81002 }
  else if (m === 2 || m === 10 || m === 14) kind = 'FF'
  else if (m === 6) kind = 'SF'
  if (kind === 'FF' || kind === 'SF') {
    const landed = g.arrows.some(x => x.dot === b && x.side === 'out' && x.landed)
    if (landed) return { kind, legal: false, code: kind === 'SF' ? 81004 : 81006 }
    code = kind === 'SF' ? 81003 : 81005
  }
  return { kind, legal: true, code }
}

// 3) 提交：端点不存在就补点（双代号必须）
function commitLink(g, a, b, kind, lag) {
  if (b === undefined) b = addDot(g, -1)              // 斑马：自动建点
  bindArrow(g, a, b, kind, lag)
  if (redundant(g, a, b)) mergeDot(g, b)              // 斑马：合并重复点
}
```

### 15.9 仍未解（已大幅收窄）

| # | 未解 | 现状 |
|---|---|---|
| 1 | ~~`neighborsCompatible` 的判据~~ | **已读通**（§15.3）：`FUN_140274360` 是"合并两个点"（改挂箭头 + `vcall(0x20)` 守卫 + 失败回滚 + `mergeDot`），两个 146 字节工具是 `rebindOut/rebindIn` |
| 2 | `LAYOUT` 分层算法（谁决定 `layer`） | 布图主函数只做"量-比-压-回滚"，分层发生在 `FUN_140507510`（5552 字节）/`FUN_14051cf50`，未展开 |
| 3 | `SceneConflictChecker` 判据 | 仍未定位到函数（§1.3 无 vtable） |
| 4 | 虚箭插入时机 | 已确认"端点缺失自动建点"（§15.5），但"何时必须建虚箭而非合并"未读死 |
| 5 | `vcall(0x20)` 在两个类上的确切语义（点=可合并？箭头=已落地？） | 只能确定它是"谓词且决定成败"，具体判据在其虚表实现里 |

> 复现入口：`re-zpert/tools/ghidra_decomp.ps1 -Addrs 0x140507510 -Out .tmp_re\x.c`（工程已保存，秒级）。

---

## 16 布图拟合器与点结构（第二轮 Ghidra）

### 16.1 `FUN_140507510` / `FUN_14051cf50` 不是"分层算法"，是**按目标纵横比的迭代拟合器**

两个函数都是这个形状（`FUN_14051cf50` 1179B 为外层、`FUN_140507510` 5552B 为主体，
都由 `0x14051c5e0` 布图主函数调用）：

```
fit(view, targetAspect /*param_3*/, …, int maxW, int maxH):
  量出内容 宽 iVar15 / 高 iVar13（含 top_hdg 等表头占位 +2、分区 -3 等修正）
  ratio = 高/宽
  while (|ratio - targetAspect| > 容差):          // 容差常量实测
      if (ratio > target)  调整列宽/行高 → 重算 iVar15/iVar13
      else                 调整另一维      → 重算
      收敛失败 → 走 §15.7 的回滚 + 67 字提示
```

实测常量（`disasm --str` 读 double）：

| 地址 | 值 | 用途 |
|---|---|---|
| `DAT_140bf7198` | **0.003** | 内层收敛容差 |
| `DAT_140bf71a0` | **0.03** | 外层收敛容差 |
| `DAT_1408bd218` | **0.1** | 尺寸合法性下界 |
| `DAT_140818460` | **1.0** | 基准比 |
| `DAT_1407f8ba0` | **0.5** | 单步修正系数 |

循环体里对 `view[+0x11c8]/[+0x11cc]`（行高/列宽类）或 `view[+0x1284]` 反复修正，
并用 `0x492492492492492` 之类的哨兵做容量校验（`FUN_1400dac80` = 抛错）。
**没有任何"按拓扑层给点分配行号"的循环** —— 分层不在这一层。

### 16.2 点（Dot）的真实结构（`FUN_1401f4b40` 148B 构造器）

对象 0x60 字节，布局实测（注意构造器参数是 `undefined4*`，`+N` 是 **dword 下标**）：

| 偏移 | 初始化 | 含义 |
|---|---|---|
| `+0x00` | `= id` | 点 id（`newDot` 里递增分配） |
| `+0x04` | `= 序号` | **在 dots 数组中的下标**（见下） |
| `+0x08 / +0x10 / +0x18` | 0 | 向量 A：`begin / end / cap`（首侧箭头集合） |
| `+0x20 / +0x28 / +0x30` | 0 | 向量 B：`begin / end / cap`（次侧箭头集合） |
| `+0x38 / +0x40 / +0x48` | `DAT_141125778`（共享哨兵） | **三个侵入式链表头**（前驱点/后继点/引用表一类） |
| `+0x54` | **`0xFFFFFFFF` = -1** | 候选"层号/状态"，初值 -1（**未验证**，见 §16.4） |
| `+0x58` | 0 | 保留 |

**`+0x04` 的语义已被证实**：`FUN_1401f5540(dots)` 就是

```
for (i = 0; i < dots.size(); i++) dots[i]->[+4] = i;
```

即**插入/删除点后把全表序号重排**。所以：

> **D1 §15.3 更正**：`canConnect` 里"按 `[+4]` 排序"不是按层号，而是**按点在 dots 数组中的先后**
> （`lo` = 数组里靠前的点，`hi` = 靠后的点）。合并方向由数组顺序决定，合并后序号会重排。

### 16.3 `canConnect` 尾段也不是 rebind（再更正一处）

`canConnect` 成功分支对 `hi` 的两个箭头集合逐个做的是：

```
obj = FUN_1401cb3d0(&tmp, graph)              // 206B：构造一个临时上下文
FUN_1401ccb20(obj, arrow, 1, 0, 0xffffffff, &out)   // 2309B：通用"登记/更新"例程
FUN_1401cb8c0(&tmp)                            // 析构
```

**改挂箭头（rebindOut/rebindIn）只发生在 `FUN_140274360`（mergeCheck）里**，不在 `canConnect` 尾段。
`FUN_1401ccb20` 的 2309 字节未展开，具体语义（登记到布局队列 / 重算几何）**未定**。

### 16.4 "分层在哪"——当前最佳判断（假设，标注清楚）

证据链：

1. 布图主函数只做**量尺寸 → 拟合 → 提交/回滚**，没有拓扑遍历；
2. 点位是**时间轴驱动**的（时标类模式 `x` 由 `plan_start/plan_stop` 映射，§2.1 的官方定义原文：
   `用活动的定位和长度表示活动历时`）；
3. `layer` 是 **Work 上的持久字段**（§3.2，随 `pert_json` 落盘），而 Dot 上只有一个初值 -1 的 `[+0x54]`。

→ **判断**：斑马的分层不是"每次渲染重算的拓扑分层"，而是
**时标模式下 x 直接由时间定、`layer` 由用户/自动调图写入 Work 并持久化**；
`[+0x54]` 若被证实是层号，则说明**点这一级也缓存了一份层号**（便于快速取行）。

**下一步钩子（一条命令可查）**：找写 `Dot[+0x54]` 的代码，就是分层/自动调图的落点。

```
pwsh re-zpert/tools/ghidra_decomp.ps1 -Addrs <候选地址> -Out .tmp_re\x.c
# 或不走 Ghidra：用 capstone 扫 `C7 4? 54 00 00 00 ?? ?? ?? ??`（mov dword [reg+0x54], imm32）
```

### 16.5 未解表更新

| # | 未解 | 现状 |
|---|---|---|
| 1 | ~~`neighborsCompatible`~~ | 已读通（§15.3 / §16.3）：mergeCheck + rebind + 守卫 + 回滚 |
| 2 | 分层算法 | 已收窄为"时标驱动 + `layer` 持久字段 + `Dot[+0x54]` 候选"（§16.4），**落点待证** |
| 3 | `SceneConflictChecker` 判据 | 仍未定位（无 vtable，需从"分离重叠工作"命令的反向调用找） |
| 4 | 虚箭插入时机 | 已确认"端点缺失自动建点"（§15.5）；"必须建虚箭而非合并"的条件未读 |
| 5 | `vcall(0x20)` 语义 | 已确认它挂在 **graph 上下文类**（不是 Dot），是"合并/落地前的一次可行性查询"；判据在其虚表实现里 |
| 6 | `FUN_1401ccb20`（2309B） | canConnect 尾段对每个箭头调用的通用例程，语义未展开 |

---

## 附录 A 本次新增证据文件
| 文件 | 内容 |
|---|---|
| `re-zpert/tools/pdata.py` | .pdata 函数边界 + "引用某段字符串的函数"反查（本次主力工具） |
| `re-zpert/tools/disasm.py` | 定点反汇编 + RIP 引用自动注解成字符串 |
| `re-zpert/tools/vtable.py` | RTTI/COL/vtable 探测（本次结论：自定义类无 vtable） |
| `re-zpert/tools/protodesc.py` | 解嵌入的 FileDescriptorProto → `.zp` 完整 schema |
| `re-zpert/evidence/zpbdata_schema.txt` / `.json` | `zpbdata.proto` 18 个消息的字段号/类型全表 |
| `re-zpert/ghidra/DumpDecomp.java` | Ghidra 头less 反编译脚本（地址 → C 源 + 调用图） |
| `re-zpert/evidence/decomp1..5.c` | 关系合法性 / 布图 / 建点 / 拟合器 / 辅助例程 的反编译源码（§15–§16 依据，共 ~190KB） |
| `re-zpert/evidence/net_cn.json` | 网络图关键词簇（网络图/双代号/时标/节点/箭线/布图/关系线/图注/刻度…） |
| `re-zpert/evidence/net_ids.txt` | 网络图相关 ASCII 键（pert_*/dot/arrow/layout/legend…） |
| `re-zpert/evidence/qmeta.txt` / `qmeta_sig.txt` | UTF-16 ASCII 串与签名式串（Qt/C++ 侧） |

## 附录 B 网络图命令全表（原文，按功能）

```
视图切换  计划-网络图 · 计划-表格网络图 · 空白计划-网络图 · 空白计划-表格网络图 · 空白计划-横道图
显示模式  时标逻辑图(1) 时标网络图(2) 逻辑网络图(3) 单代号网络图(4) 横道图(5)   [括注=枚举序]
布图      智能布图 · 按纸张 · 智能调图 · 智能位错 · 重新布局关系 · 合并多余关系 ·
          分离重叠工作 · 合并多余的层 · 清除多余空层 · 插入空层 · 删除当前空层 ·
          先布图后缩放 · 字号不变布图
视图变换  1：1显示 · 整图显示 · 显示整图 · 等比缩放 · 窗口缩放 · 框选放大 ·
          放大网络图 · 缩小网络图 · 横向压缩图形 · 横向撑长图形 · 纵向压缩图形 · 纵向撑长图形
图面设置  网络图样式 · 时间刻度… · 图注… · 分区… · 文字和线条样式… · 特殊格式… ·
          网络图区域分隔设置 · 网格开关 · 底色显示/全部显示/分区显示/不显示 ·
          名称排列(自动/横向/纵向) · 冻结分区与刻度 · 显示打印分割线 ·
          隐藏非重要关系线 · 在网络图中含/只画/不含资源曲线 · 网络图显示预警个数
刻度      年-月 · 月-天 · 日-时 · 刻度线疏密 · 不画刻度线 · 画小刻度线 · 画大刻度线 · 画工程历线
编辑      网络图在位编辑 · 在位编辑工作名称/工期/标题 · 左键拖拽至节点添加关系 ·
          双击向前/向后添加关系 · 键盘增加网络图 · 键盘浏览网络图 · 断开联系 ·
          上钻 · 下钻 · 展开/折叠工作 · 全部展开 · 成组 · 解组 · 分区(至层级1..9)
关系     建立FS关系 · 建立SS关系 · 建立SF关系-合法/不合法 · 建立FF关系-合法/不合法
打印      打印预览-横道图 · 按纸张 · 铺满纸张 · 适应纸张 · 按宽度/高度适应纸张 ·
          横道图打印-*(纸张大小/方向/页边距/标题/图注/时间刻度/边框线宽/边框颜色/无缩放/外边框加粗/打印确认/打印页码)
```

---

## 附录 C 计划 JSON 键池全表（116 键，内存原序 · VA 0x140894300）

```
_ · gczero_time · gcstop_time · info_base · total_time · resources · sources_kind · kind · code · res_graphs · task_id_seq · next_calendar_id · pert_prop · begin_dot_id · dots · notes · job_type · compute_start_time · compute_stop_time · group_id · pert_links · pen_set · color_set · p_colorset · srcfontcolor · src_label_font_color · workfontcolor · numfontcolor · titlefontcolor · scriptfontcolor · script1fontcolor · jobtypefontcolor · milestone_fontcolor · unoverjobcolor · unoverkeyjobcolor · framecolor · virtualjobcolor · milestonecolor · annobordercolor · progresslinecolor · jobslinecolor · frozentimecolor · project_info · gantt_cname · gantt_fname · write_password · cloud_project_id · cloud_project_catalog_id · cloud_project_com_id · gantt_denomi · denominator · date_gap · pert_script · script0 · jobtype_setting · task_types · forign_name · pert_row · group_shows · gantt_style · show · horz_align · vert_align · line_width · line_color · show_background · name_orientation · level_colors · srcsumcolor · srcdistcolor · srcsscolor · ordinary · free_time · milestone · progress · frozen · frame · note_border · parent · new · part · font_set · print · layer_height1 · layer_height2 · pert_mode · is_show_grid_editor · is_show_pert · is_show_script · is_show_src · is_stagger · is_hide_unimportant_link · dot_style · arrow_style · name_pos · head_height · cr · crl · cr_x · cr_y · crl_x · crl_y
```

> 取法：把 VA `0x140894300`–`0x140894b00` 里 NUL 分隔的 ASCII 串按内存顺序列出（数字串 `0`/`2`/`4`
> 是相邻数据或长度常量，不是键）。这 116 个键就是 `pert_json` 文档主体的全部键名；
> 与 §4 的图面键表（另一段池）合起来即完整图面 schema。

---

## 附录 D 计划文档 JSON 全键表（219 键，内存原序 · VA 0x1408949e8 段）

``````
progress_resolution | completion_ratio | password | cloud | zpert | draw_time | global | current_time | last_current_time | jobtype_width | max_workid | gczero_time | gcstop_time | info_base | total_time | resources | sources_kind | kind | code | res_graphs | task_id_seq | next_calendar_id | pert_prop | begin_dot_id | dots | notes | job_type | compute_start_time | compute_stop_time | group_id | pert_links | pen_set | color_set | p_colorset | srcfontcolor | src_label_font_color | workfontcolor | numfontcolor | titlefontcolor | scriptfontcolor | jobtypefontcolor | milestone_fontcolor | unoverjobcolor | unoverkeyjobcolor | framecolor | virtualjobcolor | milestonecolor | annobordercolor | progresslinecolor | jobslinecolor | frozentimecolor | project_info | gantt_cname | gantt_fname | write_password | cloud_project_id | cloud_project_catalog_id | cloud_project_com_id | gantt_denomi | denominator | date_gap | pert_script | jobtype_setting | task_types | forign_name | pert_row | group_shows | gantt_style | show | horz_align | vert_align | line_width | line_color | show_background | name_orientation | level_colors | srcsumcolor | srcdistcolor | srcsscolor | ordinary | free_time | milestone | progress | frozen | frame | note_border | parent | new | part | font_set | print | layer_height1 | layer_height2 | pert_mode | is_show_grid_editor | is_show_pert | is_show_script | is_show_src | is_stagger | is_hide_unimportant_link | dot_style | arrow_style | name_pos | head_height | crl | cr_x | cr_y | crl_x | crl_y | is_rect_name | is_line_title | title_blank | is_flag_show | is_bold_out_frame | src_height | fee_unit | is_show_splitline | frozen_timeruler_and_partition | name_style | is_solid_milestone | is_freetime_same_color | is_show_freetime | is_link_show_freetime | is_link_same_style | is_parent_show_single_line | is_show_duration_in_sections | tags | desc | is_show_finish_ratio | is_show_estimate_plan | is_show_estimate_plan_in_schedule_analysis_state | default_duration_days | is_show_external_depend_work_in_timenet | gantt_print | useable_view_type | unoverkeyjob_width | unoverjob_width | link_width | milestone_width | annoborder_width | freetime_line_width | jobsline_width | frozentime_width | work_height | gantt_work_height | work_logfont | num_logfont | milestone_logfont | title_logfont | script_logfont | script1_logfont | jobtype_logfont | src_logfont | src_label_logfont | annotation_logfont | asked_logfont | table_logfont | top_margin | bottom_margin | paper_size | orientation | print_preview_horizon_page_num | print_preview_vertical_page_num | layout_density | aspect_ratio | paper_size_mm | paper_name | catalog_name | project_name | cloud_plan_id | bim5d_plan_id | cloud_catalog_id | cloud_company_id | bim5d_org_id | bim5d_org_type | cloudt_project_id | tenant_id | org_id | org_type | is_cloud | contract_nodes | from_page | project_type | restrictActualTime | actualTimeEarliestDays | forbidStraightenForwardLine | milestoneValidationRanks | completedMilestoneAttachmentRequired | delayedMilestoneReasonResolutionRequired | approvedPlan | f_name | bar_height | is_show_separator | is_show_links | left_show_content | right_show_content | top_show_content | bottom_show_content | brush | fonts | top_hdg_font | side_hdg_font | top_hdg_style | border_style | side_hdg_style | top_hdg_bkcolor | side_hdg_bkcolor | grid_blank_bkcolor | focus_rect_color | focus_drag_rect_color | grid_row_color | check_box_color | lock_line_color | parent_work_style | title | scale | print_page_number | gantt_adjust | color
``````

> 取法同附录 C：NUL 分隔 ASCII 串按内存顺序；已过滤数字串与相邻二进制（只留下含下划线或纯字母且长度不少于 3 的串）。
> 与附录 C 的 116 键合起来：**附录 C 偏文档主体与图面，附录 D 覆盖打印/纸张/云身份/企业管控/表格头样式**。
> 其中 `layout_density` / `aspect_ratio` / `paper_size_mm` 把 §5.2 的图幅压实四维约束落到了数据字段上。


