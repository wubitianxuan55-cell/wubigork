# dsh-office 蒸馏调研论证（2026-11-13）

> 调研批，未动刀。上游=github.com/copylee711/dsh-office v1.2.0（2069af2d，2026-10-23），本地快照 `.tmp/dsh-office-upstream-src`（depth-1 克隆）。本文是唯一事实源，拍板前不写代码。

## 0. 结论卡

**值得蒸馏，推荐立项。** 核心资产是一份 9893 行的 C# COM 自动化 helper（独立进程、stdio JSON 行协议），它把我方办公板块唯一缺失的能力位——「在用户打开的 Office/WPS 里实时编辑」——连同成册的坑（COM 重试过滤、WPS 全套怪癖、脚注 story 坐标、实例生命周期、模态忙等）一起送上门，MIT 许可。helper 是自包含单文件，移植面小、坑册价值极高；我方无头 OOXML 手术管线不动，两者是互补而非替代。

本机可行性已实证：WPS 已装、csc.exe（.NET Framework 4.8 in-box 编译器）在、LibreOffice 在；MS Office 未装——上游对 WPS 恰有大量专门适配（脚注集合序号怪癖、滚动跟随、公式预转 OMML 等正是为 WPS 而设）。

## 1. 上游盘点

### 1.1 身份与成熟度

| 项 | 事实 |
|---|---|
| 形态 | DSH（DeepSeek Harness）Cordis 插件 `@copylee/dsh-office`，npm 发布 |
| 许可 | MIT（移植需保留版权头） |
| 规模 | src 1.6k 行 TS + helper/OfficeHelper.cs 9893 行 C# 5 + docs/verification.md 1075 行 |
| 成熟度 | 88 commits、v1.2.0、24 stars；verification.md 是 0.1.0→1.2.0 逐版本真机验收账（每版「改动/实测/没验证的」三段），工程纪律与本项目同构 |
| 平台 | 仅 Windows；微软 Office 或 WPS Office；.NET Framework 4.8（Win10/11 自带） |

### 1.2 架构

- **宿主层**（src/，TS）：注册 8 个工具、系统提示词段、设置页、技能引导回。与 DSH 的耦合面很小（`defineTool`/attachments/systemPrompt 三处注入），**工具层逻辑与协议层可整体照抄成 Go**。
- **helper 层**（helper/OfficeHelper.cs，C# 5）：运行时由系统 `csc.exe` 编译（约 1 秒，按源码 sha256 前 12 位缓存在 `%LOCALAPPDATA%\dsh-office\helper-<hash>\`），常驻进程，stdin/stdout 逐行 JSON（`{id, cmd, ...}` → `{id, ok, result|error}`），事件行（`ready`/`event`）旁路广播。单命令串行，超时即杀重启。
- helper 每条命令结束释放全部 COM 引用（防"引用挂着→用户关不掉 Office"）；注册 COM MessageFilter 对 COM 忙/拒做自动重试。

### 1.3 资产地图（OfficeHelper.cs 段落 → 行数）

| 段 | 行数 | 内容 |
|---|---|---|
| 框架（app/document 生命周期 + commands + 窗口/卡片/跟随） | ~1300 | 附着正在运行的 Office/WPS；静默模式隐藏实例；Explorer 双击竞态处理；迷你卡片；跟随滚动 |
| equations（LaTeX→公式） | 725 | Word 自带 LaTeX 解析器优先；WPS 不识别时插件自转 OMML 预转；公式编号 `\tag{}` 靠右；兼容模式降级 |
| Word（段落/表格/图片/脚注/修订） | 1465 | 段落号+expect 防错位；模板大单元格展开；整批一次撤销；不拆词换行 |
| citations（交叉引用域） | 226 | `\cite{}` 上标跳转 + 参考文献表 |
| styles/page layout | 59 | style_format/page_setup/页码/页眉/TOC |
| diagrams（Word 内流程图） | 329 | Word 原生形状，直角连线自动寻路，仍可逐件编辑 |
| Excel + data work | 1418 | 列概况 profile（>60 行自动概况）；清洗/透视/图表全家桶 |
| PowerPoint + designed slides + more layouts | ~4100 | 主题系统（11 套+背景纹理）+ 25 种成品版式 + grid/canvas 自由布局 + SmartArt/形状图表 |
| edit batches + rendering | 260 | 批量执行器；`office_render` 不依赖截图、后台出图；总览拼图 |

### 1.4 工具面与交互设计亮点（无关 COM 也值得抄）

1. **8 工具瘦面**：open/status/read/edit/render/save/close/help，全部批量操作收进 `office_edit` 的 ops 数组（一次调用一整步，省模型回合）。
2. **按需操作清单**：GUIDES（Word/Excel/PPT 三份排版教学，合计 ~100 行）不进 schema，随该 app 首次 open/read 一次性发放，之后 `office_help` 再取——上下文成本控制的样板。
3. **expect 防错位**：Word 按「段落号+开头文字」定位，对不上即拒绝并要求重读，防"改错位置"。
4. **unseen 视觉检查回路**：编辑后未看过渲染图则 `office_save` 自动附总览图，强制交付前核对；保存前对 Word 自动跑 `fill_gaps` 排版补空。
5. **settle 收尾**：agent 回合结束钩子里统一「卡片消失 + 静默模式打开的文档保存关闭」。
6. **技能引导回**：模型首次加载宿主自带的文件式 office-docx/pptx 技能时，用一段替换文本把它引回实时工具（第二次请求即放行）——不是禁用，是单次重定向。
7. **vision 探测降级**：模型不支持图片时 render 直接改口"用 office_read 核对"。

## 2. 我方现状对照（gap 面）

我方办公板块 = **无头 OOXML 手术管线**（探索代理盘点，file:line 详见对话记录）：

| 能力位 | 我方 | 上游 | 差距定性 |
|---|---|---|---|
| 编辑既有 docx/pptx/xlsx | docx 修订制字节手术（docxedit.go）/ pptx run 级替换（pptxedit.go）/ xlsx excelize+LibreOffice 重算 | COM 让应用自己改 | **互补**：我方适合无人值守/批量/无 Office 环境；上游适合人机协同 |
| 文件被占用时 | filelock sidecar + RenameWithRetry 兜底（会失败） | 直接在打开它的应用里改（这正是卖点） | **缺口** |
| 实时可见/可协同编辑 | 无（写后 800ms 防抖 reloadSignal 刷预览） | 全程实时 | **缺口** |
| LaTeX→原生公式 | 零 OMML 代码 | 725 行方程段 + omml.ts | **缺口** |
| 原生脚注/交叉引用/题注域 | 无 | 成套 | **缺口** |
| 模型侧渲染检查回路 | 部分（pptx→PDF→PNG 预览管线在；但「保存前强制总览核对」纪律未闭环） | unseen 追踪+总览自动附 | **可借鉴**（无头管线也能接） |
| PPT 成品版式引擎 | create_pptx.py（spec JSON 生成） | ~4100 行主题+版式引擎 | **可选蒸馏** |
| Excel 数据概况/清洗 | 无 profile 概况 | 1418 行 | **可选蒸馏** |
| 越界写防护 | PublishGuard（bash 扫描隔离） | 不需要（COM 不写盘到工作区） | 我方独有，保留 |
| 证据链/回滚 | Journal 证据卡 + rollback 快照 | 无 | 我方独有，COM 路线需补接 |

## 3. 论证：为什么蒸馏

1. **坑册不可复得**。9893 行 C# 的主体不是功能而是坑：COM MessageFilter 重试、WPS 脚注"集合序号≠显示编号"且"Index 相对 Range 收集"的怪癖、WPS 每次改动把窗口拉回 caret、PowerPoint 单进程被 Explorer 双击顶出的竞态、模态对话框忙等 5 秒、兼容模式无公式、公式经剪贴板进出 WPS 演示转矢量图……这些只有真机逐版踩出来，重新开发等于重踩一遍。
2. **架构契合**。helper 是独立 exe + 行协议，gaea 已有成熟的子进程纪律（proc.HideWindow 叠加 CREATE_NO_WINDOW、JobObject 兜底、外部引擎收割器、herdsman 管道分诊）——spawn 一个 helper 与现有模式同构；Go 侧 client（200 行 TS 的等价物）是薄层。
3. **零编译依赖**。csc.exe 是 Windows 自带（本机已实证在），无 node-gyp/Python/LibreOffice 新依赖；编译缓存按源码哈希，升级即自然重编。
4. **互补不冲突**。上游自己也保留文件式技能兜底（office-xlsx 数据分析走代码更好），并在系统提示词里明说分工。我方 office_generate/docx_apply/pptx_apply/xlsx_apply 全部保留，COM 工具族作为「实时协同档」并列，按 Office 安装与否动态注册（上游 installedApps 模式可直接抄）。
5. **许可干净**。MIT，保版权头即可入库。

## 4. 蒸馏路线选项

### 路线 A：整体移植（推荐）

- **保真照搬**：helper/OfficeHelper.cs 原样入库（保留 MIT 版权头），Go 侧新增 `internal/office/comhelper`（spawn+行协议 client+编译缓存 `%LOCALAPPDATA%\gaea\office-helper\`，哈希策略同上游）。
- **工具层重写**：8 工具映射进 gaea extra-app 注册表（`gaea_handler.go:882-906` gaeaExtraTools 装配点），遵守新增工具三处纪律（契约行+静态表/自声明+文档重导）+ spacetags（work 域）+ RegistryMap 守卫。
- **分波**：
  - 波1 = 协议层 + open/status/read/edit/render/save/close/help 核八件 + Word ops（含公式/脚注/引用）→ 真机走查；
  - 波2 = Excel data work + PPT 成品版式引擎；
  - 波3 = 迷你卡片/跟随/静默/settle + 技能引导回 + unseen 保存检查闭环。
- **工作量粗估**：波1 约 2~3 天（大头是 Go client + 8 工具接线 + Journal/预览刷新接链）；波2/3 各 1~2 天（上游代码本身不用改，主要是适配面）。

### 路线 B：选择性蒸馏（不引入 COM）

只抄 1.4 的设计件到我方无头管线：unseen 总览检查、按需 guides、expect 防错位、Excel profile 语义进 xlsx_apply。收益是 1/4，坑册全放弃。

### 路线 C：只蒸馏 OMML/LaTeX（最小刀）

omml.ts（218 行）+ helper equations 段的 WPS 预转逻辑，接进 docxedit。公式是真实缺口，但脱离实时编辑语境价值有限。

## 5. 风险与红线核对

| 项 | 核对结果 |
|---|---|
| 本机无 MS Office | 用 WPS 档，上游专门适配过；限制清单（README「限制」节）随文档入库：WPS 文字窗口不跟随滚动、演示公式转矢量图不可再编、窗口跟随在 WPS 缺失 |
| 动态注册降级 | 探测 Office 安装与否决定注册与否（上游 installedApps 单飞缓存模式），没装则工具不出现，无头管线照旧 |
| PublishGuard | 只挂 bash 工具，COM 工具不走 bash，不冲突；但 COM 编辑的文件若在工作区，写后需派发与 office 写类工具一致的 reloadSignal（FilePreview.tsx:57-63 总线）与 Journal 证据卡（appendOfficeEvidence），否则前端预览与证据链断链 |
| 写路径守卫 | COM 工具的 path 参数须过 WritePathResolver 声明写路径纪律 |
| 记忆写入确认红线 | 不涉及（不动持久化记忆） |
| 规范知识技能化红线 | GUIDES 是工具操作指引，进工具帮助文本合规；公文体例/论文格式类领域规范不新增进内核 |
| UI 简化红线 | 迷你卡片是新增功能非删减；v1 可不做（挂拍板3） |
| 上下文成本 | 8 工具 schema ~3KB + 按需 guides；可用上游「首次使用发放」模式压成本 |
| csc.exe 依赖 | Win10/11 自带，缺 .NET Framework 4.8 功能时给出上游同款报错指引；编译缓存路径不落工作区外红线管辖（%LOCALAPPDATA%\gaea\ 是用户资产目录惯例） |
| 剪贴板副作用 | PowerPoint 公式经剪贴板（保存/恢复原内容）、Excel 区域渲染走剪贴板——须在工具文档如实声明 |
| 冲突会话 | 双执行体同树风险：helper 源码单文件一次入库，Go 侧新包独立目录，撞车面小 |
| 提方向先核对删除史 | 不沾 GoalCard/MCP/平台化/跨设备记忆排除项 |

## 6. 拍板池

1. **立项路线**：A 整体移植 / B 只借鉴设计 / C 只 OMML——推荐 A。
2. **工具命名与并存面**：`office_*` 前缀与现有 `docx_apply` 等并存（COM 档 vs 无头档），还是收编成统一 `office_edit` 门面？推荐前者（改动面小、降级自然）。
3. **波3 体验件**（迷你卡片/跟随/逐字/静默/技能引导回）v1 带不带？推荐静默+跟随带、迷你卡片挂二波。
4. **PPT 版式引擎**（上游 ~4100 行最大单体）波2 移植还是砍？推荐移植——它同时利好我方 create_pptx.py 的替代面。
5. **引导回策略**：是否复制 preferLive（我方 .gaea/skills/docx、pptx 技能首次加载时引导到 COM 工具）？推荐照抄语义但以 gaea 技能装载机制实现。
6. **清场**：`.tmp/dsh-office-upstream-src` 克隆在拍板后保留（移植底稿）还是删除？推荐保留至波1 收官。

## 7. 附：本机实证记录（2026-11-13）

- WPS：HKLM/HKCU Kingsoft Office 6.0 Common 注册表在，用户目录在 → **WPS 已装**
- Microsoft Office 16.0 InstallRoot / Office16 目录：无 → **MS Office 未装**
- LibreOffice：C:\Program Files\LibreOffice\program\soffice.exe 在
- csc.exe：C:\Windows\Microsoft.NET\Framework64\v4.0.30319\csc.exe 在
