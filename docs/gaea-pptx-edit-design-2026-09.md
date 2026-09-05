# Gaea pptx 真编辑设计：大纲驱动文本框改写 + 快照回滚 + 结构化对比（方案待拍板）

> 2026-09-05 设计文档，**调研与方案阶段未改任何代码**。背景：战略层已拍板「Office 真编辑是全场空白，
> 招牌场景=框选→改→落盘回写」（docs/market-research-2026-09-05.md）；docx（框选即改+修订制）与
> xlsx（Plan→Apply+直编）均已落地，**pptx 是三件套中唯一没有编辑能力的缺口**。红线沿用：简化界面
> ≠删除功能；Word/Excel 编辑能力全量保留换壳不换芯（本文不动 docx/xlsx 通道，只在 pptx 侧补齐）。
> 原始调研稿：docs/research-2026-09-05b/pptx-edit-survey.md（源码事实逐文件核实 + 外部库快照）。

## 0. 结论速览

- **现状**：pptx「读」侧已齐（python-pptx 大纲卡 + soffice/poppler 逐页缩略，v4.28）、「写」侧只有
  从零生成（create_pptx.py / exportPptx / CrossEmbed 同走此脚本），**对既有 pptx 的原地编辑为零**；
  builtin 工具面无任何 pptx 写工具，gaea_pptx.go 包头注释明写「真编辑为远期项」——本文就是收这个远期项。
- **推荐技术路径：Go 自研 `internal/office/pptxedit`，与 docxedit 同构**（zip 原样重打包 +
  `encoding/xml InputOffset` 字节级定位 + DrawingML a:p/a:r/a:t 三级解析 + 直接替换写回），
  零新依赖、架构与测试先例齐备；修订制在 pptx **无格式层对应物**（格式无 w:ins/w:del 等价物，
  PowerPoint 官方审查靠 Compare 合并对比），故交互范式改走 **xlsx 同款「对比预览→用户批准→落盘 +
  基线快照回滚」**，证据链复用 xlsx_apply 的 App 层快照先例。
- **交互形态**：pptx 预览是 PNG（无法 DOM 级框选），框选即改的 pptx 对应形态 = **大纲卡升级为
  「页→文本框」两级导航 + 编辑面板**（左侧逐页缩略锚定版式，右侧文本框原文/指令/AI 改写/双栏对比/
  应用），AI 生成复用 OfficeEditText（文档类型无关，零改动）。
- **分期**：刀1 数据层（pptxedit 包 + 应用绑定 + 证据链）→ 刀2 编辑面（大纲扩展+编辑面板）→
  刀3 对比（pptxTextDiff + 版本时间线接入）→ 刀4（可选）修改队列泛化。每期独立可验收、可回退。

## 1. 现状盘点（源码逐条写实，详见原始调研稿）

| 通道 | 技术底座 | 编辑范式 | 证据链/回滚 | 前端预览 |
|---|---|---|---|---|
| docx | **Go 标准库自研字节级手术**（internal/office/docxedit，零第三方依赖；与 gooxml 无关——gooxml 只用于 internal/export/docx.go 生成） | 框选即改：选中→OfficeEditText 生成→`ApplyTrackedReplace` 以 w:del+w:ins 修订写入→接受/拒绝修订 | **不落证据链**（gaea_docx_edit.go 无 evidence 调用）；回滚缺位 | docx-preview@^0.4.0 renderAsync 浏览器保真渲染，DOM 文本可框选 |
| xlsx | **excelize v2.11 直编**（internal/office/xlsxedit）+ LibreOffice 重算 | Plan→Apply：AI 规划 ops→临时副本试运行→变更清单→用户批准→应用 | **落 Journal**（xlsx_apply 卡 + opsJSON + BaselinePath 快照）；GaeaRollbackRecord 零覆盖回滚 | XlsxPreview 结构化表格 + 虚拟滚动 |
| **pptx** | 读：内置 python 脚本 python-pptx（GaeaPptxOutline）+ soffice→PDF→poppler 缩略（previewPptx）；写：create_pptx.py 从零生成（exportPptx/ppt-deck 模板/CrossEmbed 三路同源） | **无编辑**。大纲卡只有「针对第 N 页修改」→ 往 composer 插指令模板，靠 agent 侧工具间接处理且无 pptx 写工具可调 | 无 | 逐页 PNG 缩略 + 页锚点（PNG 无文本可选性）；版本对比落 unsupported |

关键细节（选型依据）：

- docxedit 的技术内核可平移：`xml.Decoder.InputOffset()` 记录 token 原始字节区间 → 段落(w:p)/
  run(w:r，含 rPr 区间)/文本段(w:t)三级模型 → 精确匹配+空白折叠模糊匹配定位 → blocker 保护
  （drawing/tab/br 等覆盖即拒绝）→ 只重建命中段落、其余字节原样 → 临时文件原子替换。
  **DrawingML 文本结构与之同构**：`p:txBody/a:p`（段落）→`a:r`（run，含 `a:rPr`）→`a:t`（文本）。
- pptx 预览缓存键 = sha256(路径+size+ModTime)（gaea_pptx.go:319），**文件写回后自动失效**——
  编辑落盘后预览刷新零额外成本。
- 证据链设施齐备可复用：`evidence.StageBaseline/StageBaselineTo`（.gaea/work/rollback/ 快照）、
  `GaeaRollbackRecord`（基线回写 + 「目标已被手工修改则拒绝」）、VersionTimeline 的
  baselinePath 对比数据层（text/docx/xlsx 三 kind 已成，pptx 补第四种）。
- 前端纯逻辑可复用：`docxAnnotationQueue.ts` 是 deps 注入式（generate/apply/readText 回调），
  换绑 pptx 通道即可复用；`docxTextDiff.ts` 段级 LCS 可直接以「形状文本序列」复用；
  v4.87 ChangesDiff 的改蓝配对+字符级高亮+折叠对 pptx 对比白得。
- pptx 文本的特殊形态（docx 没有的增量）：slide 文件序需要 `presentation.xml` sldIdLst + rels
  映射到 `ppt/slides/slideN.xml`（大纲卡页码与物理文件的桥）；文本还可能藏在表格(a:tbl)、
  组合形状(p:grpSp)、备注页(p:notesSlide)、字段(a:fld)——分级支持见 §5 范围。

## 2. 技术选型

| 选项 | License | 维护状态 | pptx 能力边界 | 评估 |
|---|---|---|---|---|
| **A. Go 自研 pptxedit（docxedit 同构）** | 无新依赖 | 自研自养；docxedit 先例 + 单测基建现成 | 文本框/表格文本 run 级改写；字号字体=复用命中 run 的 rPr 原字节（天然保格式）；母版/主题/动画不触碰=字节不动 | **推荐**。与 gaea「AI 不啃完整 OOXML、其余字节零扰动」原则同源；编辑链路不依赖 python 在场（比现有读侧更可靠） |
| B. python-pptx 写回脚本 | MIT | v1.0.2（2024-08 后发版放缓；社区活跃，fork 后备多）——**待验证**长期维护 | text_frame 成熟（段落/run/字体）；已知坑 issue#285：Paragraph.text 赋值摊平 runs 丢格式，须逐 run 编辑 | 备选。与读/生成侧同栈，但把编辑绑死在 PATH python（缺失即功能不可用）；lxml 全量重序列化对未建模 part 的扰动需逐项验证，违背「零扰动」偏好 |
| C. carmel/gooxml presentation（仓库既有依赖） | **AGPL-3.0**（LICENSE.commercial 并存） | 上游 2022 后未见更新（模块代理版本口径，待验证） | Read 可解包既有 pptx；Slide/PlaceHolder/TextBox API 以创建为主，既有文本 run 级改写需下探 pml schema 自行操作 | 不推荐。高层 API 薄弱（改造工作量≈自研）+ AGPL 传染对闭源桌面分发不利 |
| D. unidoc/unioffice（商业版） | 商业双授权 | 活跃（unidoc.io 支撑） | PresentationML 支持面最全 | 不推荐首轮。商业授权成本 + 破坏「零商业依赖」惯例；能力上限高但本期范围（文本改写）用不到 |
| E. 小型库（hurtener/pptx-go 1 star、referefref/gopptx 0 star 等） | MIT 等 | 无社区 | 无 run/格式保证 | 全部排除。Go 生态「轻量 pptx 编辑」无成熟品——这正是仓库读侧当年选 python-pptx 的原因 |

**推荐：A 为主选，B 作为 A 的对照验证工具**（单测里可用 python-pptx 生成 golden fixture、
交叉验证写回结果可被第三方库正常打开；生成侧 create_pptx.py 照旧）。A 的实现要点：

1. zip 读写照抄 docxedit（条目顺序原样保留、临时文件 + os.Rename 原子替换）。
2. 解析目标文件由「word/document.xml 单文件」变「slide 文件集」：先解析
   `ppt/presentation.xml`（p:sldIdLst 顺序）+ `ppt/_rels/presentation.xml.rels` 得到
   「页码 → slide part 路径」映射（一次性 ~百行，含 .pptx 缺 rels 的结构化报错）。
3. 段落解析器从 WML 移植为 DrawingML：a:p/a:r/a:t + a:rPr 原字节区间；blocker 换
   a:br/a:fld/a:tab；a:t 的 `xml:space` 语义与 WML 一致处理（待验证：DrawingML 尾随空格
   保留行为与 WML 的差异，实现时以 fixture 实测为准）。
4. 写入无修订制：`ApplyTextReplace(slideIdx, target, replacement)` 直接替换命中 a:t 序列
   （跨 run 拆分复用 docxedit 的 run 分组重建模型；新文本继承最后受影响 run 的 rPr 原字节）。
5. 应用前 `evidence.StageBaselineTo` 快照 + Journal Append（对齐 xlsx_apply 口径），
   回滚直接走 GaeaRollbackRecord 零改动。

## 3. 交互设计（对标 docx「框选即改 + 修订制」，按 pptx 的现实形态重述）

### 3.1 修订制的 pptx 语义（如实给方案）

pptx 格式**没有内联修订标记**（PowerPoint 官方审查路径是 Review→Compare 合并对比 + 批注；
w:ins/w:del 无对应物）。因此 docx 的「修订写入→预览可见→接受/拒绝」三段式在 pptx 有两种诚实译法：

| 方案 | 流程 | 取舍 |
|---|---|---|
| **P1（推荐）：xlsx 同款 Plan→Apply 审阅制** | 选中文本框→指令→AI 改写→**前端双栏对比（原文/新文）→用户点「应用」才写盘**→预览刷新；写盘前自动基线快照，应用后可从版本时间线一键回滚 | 落盘前有明确批准动作，与「无修订标记可还原」的格式现实最匹配；xlsx 用户已被教育过该范式 |
| P2：docx 同款「先落盘可撤销」 | 点应用即写盘→缩略图可见变化→不满意走回滚 | 少一步确认，但错误改动已落盘（只靠快照兜底），对演示文稿这种「交付前最后一步」的文件风险偏高 |

推荐 P1。无论 P1/P2，「修订可见性」一律由 §3.3 的结构化对比承担，UI 文案如实标注
「PPT 格式不支持修订标记，应用即生效，可随时恢复到应用前版本」。

### 3.2 编辑入口（预览是 PNG，框选不可行 → 大纲驱动）

- **PptxOutline 大纲卡升级为「页→文本框」两级导航**：每页条目可展开文本框清单
  （后端大纲已回 texts 数组，需补 shapeId/序号与文本预览截断）；点文本框条目 → 打开编辑面板。
- **PptxEditPanel 编辑面板**（FilePreview 右栏或弹层，左缩略图右面板）：
  1. 原文只读展示（取自大纲/详情接口，含所在页与形状定位）；
  2. 预设动作（润色/精简/翻译/扩写）+ 自定义指令——与 DocxPreview 同一套动作语义；
  3. 「生成」→ 复用 `GaeaOfficeEditText`（纯文本进/出，零改动；pptx 场景约束「要点短句化、
     长度相近防溢出」以附加提示词参数传入，不动 docx 既有调用）；
  4. 双栏对比（原文/新文，字符级差异高亮直接复用 ChangesDiff 的字符高亮管线）；
  5. 「应用」→ `GaeaPptxApplyEdit(rel, slideIdx, shapeRef, originalText, replacement)`
     → 后端定位校验（原文不匹配则拒绝，宁拒不误改）→ 快照 → 写盘 → 返回新预览（缓存自动失效）。
- 定位锚点用「slideIdx + 原文摘录」（与 docx 框选即改同款文本匹配语义），不依赖 shapeId——
  shapeId 在外部工具往返后可能变化，文本摘录与用户所见一致且可校验；shapeId 仅作辅助提示。

### 3.3 预览与版本对比

- **预览**：维持既有 soffice→PDF→poppler 缩略管线（保真度=LibreOffice 渲染口径，字体缺失时
  与 PowerPoint 实际效果有差异，见 §5）。不引浏览器端 pptx 渲染库（pptx-preview.js/PPTXjs/
  PptxViewJS 保真与维护性均未达 docx-preview 水位，列为远期观察项）。
- **pptxTextDiff 分层**（前端纯函数，JSZip 解 ppt/slides/slideN.xml，docxText 同套路）：
  - 层1 slide 级：页数/页序/标题列表 diff（成本最低，先行）；
  - 层2 **形状级（本期目标）**：同页文本框文本序列做 LCS（docxTextDiff 的「段落序列」换成
    「形状文本序列」，算法零改动）；页序对齐后逐页比对；
  - 层3 run 级字符高亮：层2 产出的相邻 del+add 对交给 v4.87 ChangesDiff 自动获得字符级配对
    高亮（白得，不用新写）。
  - `versionCompare.compareVersionWithCurrent` 增 `kind:"pptx"`，VersionTimeline 的
    unsupported 分支对 .pptx 收口。

## 4. 分期刀序（每版 1-2 刀惯例；每期独立验收、独立可回退）

| 刀 | 内容 | 验收面 | 档位 |
|---|---|---|---|
| **刀1 数据层** | 新包 `internal/office/pptxedit`：slide 序映射 + DrawingML 段落解析 + `LocateText/ApplyTextReplace` + 原子写回；App 层 `GaeaPptxApplyEdit`（快照+Journal+返回新预览）；绑定面 +1（579→580，gen_bindings 再生） | pptxedit 包单测（fixture pptx：命中/跨 run/未命中/表格内/blocker 拒绝/原子性）；Go 全量 0 FAIL；drift PASS | **中刀**（docxedit 有蓝本，slide 映射为新增量） |
| **刀2 编辑面** | 大纲接口补文本框级详情（shapeRef+文本预览）；PptxOutline 两级导航；PptxEditPanel（预设动作/生成/双栏对比/应用）；GaeaOfficeEditText 附加 pptx 场景约束参数 | vitest（面板状态机+对比渲染）；?mock=1 真机走查（大纲→选文本框→改→应用→缩略图刷新全链路）；i18n 三语 | 中刀 |
| **刀3 结构化对比** | pptxTextDiff.ts（层1+层2）；versionCompare kind:"pptx"；VersionTimeline 接入 ChangesDiff | vitest 纯函数单测（页对齐/形状 LCS/降级 unsupported 口径）；走查（编辑后版本时间线出红绿对比） | 小刀 |
| **刀4（可选）修改队列泛化** | docxAnnotationQueue deps 换绑 pptx 通道（generate=OfficeEditText、apply=GaeaPptxApplyEdit、readText=大纲全文拼接），多处文本框批改一次提交 | vitest（复用既有队列测试模型换 deps）；走查多页批改 | 小-中刀 |

刀序原则：数据层先行（刀1 独立可用——绑定面先通，agent 侧即可经 GaeaCallTool 试验）；
刀2/刀3 可并行拆分；刀4 视刀2 使用反馈拍板。

## 5. 风险与边界

| 风险 | 口径 | 防线 |
|---|---|---|
| 版式漂移 | 只改 a:t 文本、rPr/layout/master/theme 字节不动=格式天然保真；但文本变长可能溢出文本框 | AI 提示词约束长度相近；对比预览让用户落盘前看见新文；溢出属 PowerPoint 打开期行为，如实告知不承诺 |
| 字体缺失 | 缩略预览是 soffice 渲染口径，机器缺字体时预览与 PowerPoint 实际效果有出入 | 编辑不引入新字体（继承原 run）；预览失真属既有读侧问题不在本期放大 |
| 文本定位歧义 | 同文本多处出现（模板化 deck 常见） | docx 同款口径：多处命中取第一处且 unique=false 如实上报；定位不到拒不写（宁拒不误改） |
| 特殊形态文本 | 表格/组合形状/备注页/图表内嵌文本/SmartArt | 刀1 范围=slide 正文文本框（含组内与表格内 a:t 的自然命中，docxedit 对表格段落的处理先例）；备注页/图表文本列为边界外（拒绝+提示），范围扩大待拍板 |
| 母版与主题、动画多媒体 | slideMaster/slideLayout/theme/p:timing/媒体 part 一律不触碰（字节不动） | zip 重打包条目原样保留（docxedit 先例）；单测断言未命中文件字节零变化 |
| 文件损坏 | 写坏 pptx=用户交付物报废 | 临时文件+原子替换（docxedit 先例）+ 应用前基线快照 + GaeaRollbackRecord 回滚 + python-pptx/soffice 打开验证入单测（对照组 B 的复用） |
| .ppt 旧格式 | 不支持 | 沿大纲卡既有文案：请先另存为 .pptx |

## 6. 本方案待拍板项

1. **技术路径**：推荐 A（Go 自研 pptxedit 同构 docxedit）；备选 B（python-pptx 写回）。
   C/D/E 已陈述排除理由，如无异议按 A 出刀1。
2. **应用范式**：推荐 P1（xlsx 同款 Plan→Apply，落盘前对比+批准）；备选 P2（docx 同款先落盘
   可回滚）。影响刀2 面板交互与文案。
3. **编辑范围**：刀1 是否包含表格内/组内文本框命中（推荐：包含——解析器自然覆盖、零额外成本）；
   备注页（notesSlide）与图表内文本是否进范围（推荐：本期边界外）。
4. **证据链口径**：pptx 编辑落 Journal（推荐，对齐 xlsx_apply）；顺带是否补 docx_apply 不落
   证据链的既有欠账（推荐：另立小刀，不动 docx 编辑行为本身，红线不受影响）。
5. **刀4 修改队列**是否随首轮交付（推荐：刀2 上线观察后再定）。
6. **远期项是否立项**：浏览器端 pptx 结构化渲染（真·框选即改的前提）、母版/主题级改动、
   图表数据编辑——本期均不承诺，仅登记。

## 7. 参考资料

- 原始调研稿：docs/research-2026-09-05b/pptx-edit-survey.md（源码事实 + 外部库快照 + 引用链接）
- 战略依据：docs/market-research-2026-09-05.md（Office 真编辑全场空白、招牌场景口径）、
  docs/market-research-2026-09-03c.md（文件交付调研）
- 既有设计：docs/gaea-office-upgrade-plan-2026-09.md（B2 pptx 最小交互来源）、
  docs/gaea-edit-tools-design.md（工具层设计口吻先例）
- 关键源码：internal/office/docxedit/docxedit.go、internal/office/xlsxedit/xlsxedit.go、
  internal/app/gaea_pptx.go、internal/app/gaea_docx_edit.go、internal/app/gaea_xlsx_edit.go、
  internal/app/gaea_export.go、internal/app/gaea_crosslink.go、internal/gaea/evidence/journal.go、
  frontend/src/gaea/components/DocxPreview.tsx、frontend/src/gaea/lib/docxAnnotationQueue.ts、
  frontend/src/gaea/lib/docxTextDiff.ts、frontend/src/gaea/lib/versionCompare.ts
