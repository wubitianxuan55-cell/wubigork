# 调研原始稿：pptx 真编辑技术底座（2026-09-05）

> 为 docs/gaea-pptx-edit-design-2026-09.md 提供原始依据。两部分：仓库内源码事实（逐文件核实）与
> 外部库调研（GitHub/官网快照）。**调研阶段未改任何代码。** 查不到的标「待验证」。

## 一、仓库内源码事实（pptx/docx/xlsx 三通道现状）

### 1. docx 编辑通道：Go 标准库自研字节级手术（与 gooxml 无关）

- 文件：`internal/office/docxedit/docxedit.go`（全 826 行）。import 仅标准库：
  `archive/zip`、`encoding/xml`、`regexp`、`bytes`、`time` 等。**零第三方依赖。**
- 核心机制（包头注释原文口径）：「不解析/重建整份 OOXML，只对 document.xml 中命中的段落做
  字节级手术，保证『框选即改、其余内容与版式零扰动』，与 gaea『AI 不啃完整 OOXML』的原则一致」。
- 关键实现点（pptx 同构评估的依据）：
  - `xml.Decoder.InputOffset()` 精确记录每个 token 的原始字节区间；`parseParagraphs` 收集
    段落（w:p）→ run（w:r，含 rPr 原始字节区间）→ 文本段（w:t，含属性子串）三级模型。
  - `locateSpan`：先精确子串匹配，失败后空白折叠模糊匹配（rune 级坐标映射还原原区间）。
  - `blocker` 保护：选区覆盖 drawing/pict/object/fldChar/instrText/tab/br 等特殊元素时拒绝并提示缩小选区。
  - 修订写入：`rebuildParagraph` 把被选中文本包进 `w:del`（delText）、插入 `w:ins`（新文），
    `w:id` 取全文档最大 id+1 递增；`w:author="gaea AI"`；首尾空白补 `xml:space="preserve"`。
  - `AcceptChanges`/`RejectChanges`：按作者扁平化 w:del/w:ins（accept: 删 del 留 ins；
    reject: 删 ins、delText 还原为 t）。
  - 写回：`writeDocx` 用临时文件 + `os.Rename` 原子替换；zip 内条目顺序原样保留。
- App 绑定：`internal/app/gaea_docx_edit.go`——`GaeaOfficeEditText`（AI 生成替换文）、
  `GaeaDocxApplyEdit`（修订式写入→返回新预览）、`GaeaDocxAcceptChanges`（接受/拒绝全部修订）。
- **注意：docx 框选即改不落证据链**（gaea_docx_edit.go 全文无 evidence 调用）——版本时间线
  看不到这类编辑；xlsx Apply 则落（见下）。这是两通道现状的差异。

### 2. xlsx 编辑通道：excelize 直编 + Plan→Apply + 证据链

- 文件：`internal/office/xlsxedit/xlsxedit.go`。依赖 `github.com/xuri/excelize/v2 v2.11.0`（go.mod）。
- 范式：`BuildContext`（表格上下文 JSON）→ AI 规划 ops JSON（`Client.XlsxEditOps`，
  internal/ai/copilot.go）→ **临时副本试运行**（合法性+真实摘要）→ 与原文件 diff 出变更清单
  （`XlsxPlanResult.Changes`）→ 用户批准后 `GaeaXlsxApplyEdit` 执行 → LibreOffice 重算公式。
- 证据链：`internal/app/gaea_xlsx_edit.go appendXlsxEvidence`——work 空间才落 Journal（JSONL），
  记 `Tool:"xlsx_apply"` + opsJSON（≤SummaryLimit 才落）+ `BaselinePath`（应用前快照），
  `Status: pending_verify`。回滚走 `GaeaRollbackRecord`（gaea_verify.go：基线快照回写；
  目标已被手工修改则拒绝——「零覆盖」红线）。

### 3. pptx 现状：只读 + 生成，无任何编辑工具

- **读侧（v4.28 B2）** `internal/app/gaea_pptx.go`：
  - `GaeaPptxOutline`：内置 python 脚本（`pptxOutlinePy` 常量，临时 .py 落盘注入，规避 -c
    命令行长度/引号陷阱）用 python-pptx 解析 slides/shapes/text_frame → stdout JSON 大纲
    （index/title/texts/shapeCount；每页 texts ≤40 条、单条截断 200 rune）。PATH 上的 python
    + 60s 超时；python/python-pptx 缺失走结构化错误（Available=false），降级不中断。
  - `previewPptx`：soffice→PDF（懒渲染，缓存 `.gaea/cache/pptx-preview/<hash>.pdf`）→
    poppler pdftoppm 低 DPI（64）逐页 PNG（`<hash>-pages/`）。缓存键 = sha256(路径小写 + size +
    ModTime.UnixNano)（gaea_pptx.go:319）——**文件被写回后缓存自动失效**，编辑后预览刷新零成本。
    最多 60 页，TTL 7 天清理。
- **生成侧**：
  - `internal/app/gaea_export.go exportPptx`：Markdown → slides spec JSON →
    `create_pptx.py`（.gaea/skills/pptx/scripts）从零构建 16:9 deck（python-pptx，
    空白版式 slide_layouts[6] + 文本框，Microsoft YaHei，a:ea 东亚字体同步设置）。
  - 任务模板 `ppt-deck`（internal/app/gaea_templates.go）：提示词引导 agent 走「pptx 技能」
    （即 create_pptx.py）产出 .pptx 到 .gaea/exports/。
  - CrossEmbed 嵌 PPT：`internal/app/gaea_crosslink.go embedPptxWithChart` →
    `crosslink.BuildPptxSpec`（图表页+数据明细页 spec）→ **同样走 create_pptx.py 生成新文件**
    （非向既有 pptx 追加）。
- **编辑侧：不存在**。`internal/gaea/tool/builtin` 无任何 pptx 写工具；format_convert
  （pptx→md 经 internal/office/docmd markitdown）与 chart_gen（xlsx 图表）均为只读或非 pptx。
  gaea_pptx.go 包头注释明确「真编辑 pptx（python-pptx 写回）为远期项」。
- **前端**：`PptxOutline.tsx` 大纲卡（每页「标题+摘要」+「针对第 N 页修改」→ composer 插入
  指令模板，不直接发送）；FilePreview 逐页缩略 + 页锚点。pptx 预览是 PNG，**无 DOM 级文本
  可选性**——docx 式框选在此渲染形态上不可行。

### 4. 前端 docx 预览与配套纯逻辑（可复用面）

- `DocxPreview.tsx`：`docx-preview@^0.4.0` 的 `renderAsync` 浏览器内保真渲染；框选即改
  （选中文字→预设动作[润色/精简/翻译/扩写]+自定义指令→AI 替换→修订写入→接受/拒绝）；
  修改队列（`docxAnnotationQueue.ts`）串行批量执行，单条与批量互斥。
- `docxAnnotationQueue.ts`（326 行纯逻辑，零 DOM/零后端依赖）：队列状态机（pending/running/
  done/failed/skipped 白名单迁移）、摘录归一化（空白收敛）+ `locateExcerpt` 再定位（多处命中
  取第一处且 unique=false 如实上报）、`runQueue` 串行编排（每条执行前对最新全文再定位）。
  **deps 注入式设计（generate/apply/readText 回调）理论上可直接换 pptx 通道复用。**
- `docxTextDiff.ts`：段级 LCS diff 纯函数（全量 DP 矩阵，与 lib/diff.diffLines 同算法，
  「行」换「段落」）；口径注释明确「段内改写呈现为一对相邻 del+add」。
- `versionCompare.ts`：版本对比数据层——text（行 diff）/docx（JSZip+DOMParser 提取段落→
  段级 LCS）/xlsx（结构化单元格 diff）/其余 kind:"unsupported"（UI 降级并排预览）。
  **pptx 当前落 unsupported。** `VersionTimeline.tsx`（v4.95 起）对比体统一走 ChangesDiff
  渲染器（v4.87 的改蓝配对+字符级高亮+上下文折叠可白得）。

### 5. 证据链与基线快照（pptx 可复用的既有设施）

- `internal/gaea/evidence/journal.go`：`ChangeRecord.BaselinePath`（写盘前整文件基线快照，
  Verifier 通道 B 视觉 diff 与 Rollback 回滚原料）；`StageBaseline/StageBaselineTo`
  （快照到 `.gaea/work/rollback/`，无台账/未配置时静默降级）。
- 写盘工具（write_file/edit_file/multi_edit/edit_lines/move_file）经 `evidence.RecordChange`
  自动上报（internal/gaea/agent/execute_one.go:148 注释口径）。
- 回滚：`GaeaRollbackRecord`（internal/app/gaea_verify.go）——基线回写 + 回滚前再快照 +
  「目标已被手工修改则拒绝」（对 write_file 的 AfterSummary 与当前内容 ClampSummary 比对）。
- **App 层自写文件的先例：xlsx_apply 手动快照 + Append**（gaea_xlsx_edit.go:186）——pptx
  编辑照抄此口径即可接入版本时间线/回滚，且顺手可补 docx_apply 的同类欠账（另行拍板）。

### 6. AI 文本编辑提示词（可复用面）

- `internal/ai/copilot.go OfficeEditText`：文档类型无关（纯文本进/纯文本出；核心规则=仅改
  选中文本、遵循指令、关键信息不变、输出纯文本、保留原语气）。pptx 文本框改写**直接复用**，
  唯一要补的是 pptx 场景约束（要点短句化、长度相近防溢出）——加参数或旁路提示词均可。

## 二、外部库调研（2026-09-05 网络快照）

### 1. unidoc/unioffice
- 链接：https://github.com/unidoc/unioffice ｜ License：**商业双授权**（源码可见，运行需
  license key；前身为 AGPL，v1.1 起转商业）。
- 能力：纯 Go 的 docx/xlsx/**pptx** 读写处理，PresentationML 支持面最全（官方宣称目标
  「most compatible and highest-performance」）。unioffice-examples 有 pptx 系列示例。
- 维护：活跃（商业公司 unidoc.io 支撑）。
- 结论：能力上限最高，但**引入商业授权成本** + 与仓库现有「零商业依赖」惯例冲突。

### 2. carmel/gooxml（仓库既有依赖，本模块缓存实测）
- 链接：https://github.com/carmel/gooxml ｜ go.mod 锁定 v0.0.0-20220216072414-40ff56130850。
- License：AGPL-3.0（LICENSE 文件头 + LICENSE.commercial 并存；包头版权声明 Baliance 2017）。
- 实测（GOMODCACHE 本地目录）：**自带 `presentation` 包**（presentation.go/read.go/slide.go/
  slidemaster.go/placeholder.go/textbox.go），`presentation.Read` 可解包既有 pptx，
  Slide 有 `PlaceHolders()/GetPlaceholderByIndex/AddTextBox`，`Slide.X()` 透出 *pml.Sld。
- 边界：API 以「创建」为主（TextBox.AddParagraph 等），**对既有文件文本 run 级改写的
  高层 API 薄弱**（需下探 pml schema 自行操作）；仓库用它只做 docx 生成
  （internal/export/docx.go）；上游 2022 年后未见更新（以模块代理版本为准，待验证）。
- 风险：AGPL 传染对闭源桌面分发不利（当前仓库未见 AGPL 合规处置的先例说明，待拍板）。

### 3. 小型 Go pptx 库（均不可作为主选）
- hurtener/pptx-go（Muprprpr/Go-pptx 的 fork）：MIT/Apache，宣称 create/read/modify+流式；
  **1 star / 1 fork**（2026-09-05 GitHub 页面快照），能力无文档佐证。不可依赖。
- referefref/gopptx：find-and-replace 式替换（ReplaceSlideContent/ReplaceNotesSlideContent/
  ReplaceImage），**0 star / 8 commits**，纯字符串替换无 run/格式保证。不可依赖。
- kenny-not-dead/gopptx：纯 Go 读写 pptx（pkg.go.dev 收录），活跃度低（待验证，未深查）。
- 结论：Go 生态「轻量 pptx 编辑」无可用成熟品——这解释了仓库读侧选 python-pptx 的原因。

### 4. python-pptx（仓库既有运行时依赖）
- 链接：https://github.com/scanny/python-pptx ｜ License：**MIT**。
- 能力：官方定位即「creating, reading, and **updating**」pptx；text_frame 级编辑成熟
  （段落/run/字体/对齐/自动缩放）。已知坑：issue #285——`Paragraph.text` 赋值会把整段 runs
  摊平成单一 run 丢格式；保格式须逐 run 编辑（与 docxedit 的 run 级手术同一问题域）。
- 维护：v1.0.2（v1.0.0 于 2024-08-03）；此后发版放缓（Snyk 口径 12 个月无新版本；
  社区 PR 活动持续到 2025-09），PyPI 下载量 ~156 万/天；有 power-pptx / python-pptx-ng 等
  续维护 fork 可作后备。
- 仓库现状：读侧大纲、生成侧 create_pptx.py 都依赖它；`ensure_pptx()` 会自动 pip install。

### 5. 浏览器端 pptx 渲染（预览/框选候选，仅远期参考）
- pptx-preview.js（develop365 演示站）：pptx→HTML 浏览器内预览。
- PPTXjs（pptx.js.org）：老牌 pptx→HTML 查看器。
- PptxViewJS（gptsci/pptxviewjs）：Canvas 渲染，纯客户端。
- 共性风险：保真度参差、维护活跃度普遍弱（对比 docx-preview 的成熟度），用于「框选即改」
  的 DOM 文本可选性需要逐库验证（均未验证，待验证；不进本方案承诺范围）。

### 6. 关键格式事实：pptx 没有修订制（track changes）
- Microsoft 支持文档口径：PowerPoint 的修订审查靠 **Review→Compare**（两版合并对比出
  Revisions 任务窗格）+ 批注，**不存在 Word 式内联修订标记**（w:ins/w:del 无对应物；
  社区与第三方文档一致确认：PowerPoint 无法像 Word 那样自动跟踪修订）。
  - 来源：https://support.microsoft.com/en-us/powerpoint/track-changes-in-your-presentation 、
    https://slidemodel.com/how-to-track-changes-in-powerpoint/ 、
    https://pptproductivity.com/blog/how-to-track-changes-in-powerpoint-using-review-compare-feature
- 结论：docx「修订制」在 pptx **无格式层对应物**，修订语义须重定义为「应用前对比预览 +
  应用后基线快照回滚」（xlsx Plan→Apply 同款范式），或接受「先落盘可撤销」。
- DrawingML 文本模型：`ppt/slides/slideN.xml` 的 `p:txBody/a:p`（段落）→`a:r`（run，含
  `a:rPr`）→`a:t`（文本），与 WML 的 w:p/w:r/w:t 三级结构同构（OOXML 常识 + gooxml pml
  schema 印证；不同点：字段 `a:fld`、换行 `a:br`、表格 `a:tbl` 嵌套 txBody）。
