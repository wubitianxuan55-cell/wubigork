# ZCode 蒸馏给办公板块：调研与论证

> 对象：[zai-org/ZCode](https://github.com/zai-org/ZCode)（Z.ai 的 AI 编程工作台：桌面 / Web / 终端三形态）
> 基线：本地克隆 `clones/zcode`，`main@3c5b227`（2026-10-20，README 标 v3.16.2）
> 日期：2026-10-24 · 状态：**调研 + 论证，未改任何代码**
> 方法：8 路并行子代理（ZCode 侧 6 路 / gaea 侧 2 路）+ 主代理交叉复核，全部结论要求 `file:line`
> 依据：`docs/research/zcode-distill/` 下 s1~s8 共 8 份子报告（约 54 万字节），本文只做收敛与论证
> **后续**：[分析篇](zcode-office-distill-analysis-2026-10.md) 对本文 7 处结论做了交叉校验与修正（P0 前置、Q4 成本、上下文条条目删除、N8 移出等），**冲突时以分析篇为准**。

---

## 0. 结论（先看这五条）

**① ZCode 对办公板块的价值，不在它的 agent 骨架，在它的"文档工艺"。**
它把 docx/pptx/xlsx/pdf 做成四个**官方插件**：技能四层（总纲→route→scene→reference）、确定性机检脚本、独立视觉验收子代理、按格式分发的查看面板。gaea 的办公骨架已经比它适配本地（真编辑/修订制/Plan→Apply/证据链），缺的正是这层"怎么做才像人做的"。

**② 两边不是同一种 agent，抄错一层就是负债。**
coding agent 的真相源是**仓库最终状态**（可 diff、可测试、可幂等重放），所以它敢自动判定完成、自动重试、靠 git 回滚。办公 agent 的真相源是**用户对这份交付物的认可**，docx/pptx 是状态化、不可幂等、局部编辑成本高的产物。→ 自动判定完成/自动续跑这类机制在办公会变成风险；**可判定的那一小部分（版式、页码、字体、溢出、口径）恰恰是办公最缺的确定性门禁**。

**③ 四个真空，四个补法。**

| 真空 | gaea 现状 | ZCode 的补法 | 结论 |
|---|---|---|---|
| V1 工艺知识 | 有纪律（office-edit 铁律）、无版面知识 | 7 本场景手册 + 排版规则库 | 搬内容，做成技能 |
| V2 确定性门禁 | 只有文本级体检 + before/after 像素 diff | postcheck / audit / validate 规则集 | 搬规则，Go 重写 |
| V3 视觉验收 | 有 judge，但只看文本证据 | `visual-judge` 逐页 JSON 判决 | 搬协议，做子代理 |
| V4 指认 | 引用是纯文本，模型与用户不在同一版 | 锚点语法 + `report_view` + `capture_view` | 搬契约，改预览 |

**④ 蒸馏的形态必须是"技能 + 规则 + 协议"，不是"代码移植"。**
gaea 长期规划明令：「领域规范/模板/规则一律技能化（代码只进通用机制）」（`docs/gaea-longterm-plan-2026-10.md:29`）。这条恰好把 ZCode 最大的一块资产（四个办公插件合计约 2MB，其中纯技能与规则文本约 1MB）变成"允许搬"的形态，同时挡掉它最大的一块负债（Python/TS 渲染栈）。

**⑤ 本轮的性价比最高五项（三项 S、两项 M，合计约 5~6 人日，可在一周内验证）。**
中文排版阈值表 · WPS/Word 兼容黑名单 · docx 机检 9 条 · docx 锚点 + 视图回报 · PDF 文本 CJK 部首归一。
合计约 5~6 人日，全部落在既有模块内，不新增板块（治理红线 `gaea-longterm-plan-2026-10.md:5`）。

---

## 1. 调研口径

### 1.1 我们调研了什么

ZCode 仓库 1.8 万份受管文件（`git ls-files` 18266），不能通读。按"是否可能影响办公板块"切了八路：

| 路 | 覆盖 | 产出 |
|---|---|---|
| s1 | agent 运行时内核（loop / 权限 / 压缩 / 队列 / 重试 / 子代理） | 33 个机制 M1–M33 + 8 条不采 |
| s2 | 工具层（文件 IO / 结果预算 / 错误码 / 两阶段 apply） | 12 项 ROI 排序 |
| s3 | GenUI 与 agent 交互面（上下文条 / 审批 / 交付物 / 引用） | 15 条候选打分 |
| s4 | 扩展模型（skills / agents / plugins / MCP / hooks） | 5 个技能建议 |
| s5 | **gaea 办公后端基线**（骨架、工具表、证据链、缺口） | 缺口 Top 8 |
| s6 | **gaea 办公前端基线**（组件地图、交互缺口） | 缺口 Top 8 |
| s7 | ZCode docx / pptx 插件逐文件拆解 | 10 条知识资产 + 搬运矩阵 |
| s8 | ZCode xlsx / pdf 插件逐文件拆解 | 14 条规则集 + 10 个源码坑 |

另做了三项主代理亲读：`documents-plugin/skills/docx/SKILL.md`（315 行全文）、`office-runtime/src/CONTRACT.md` + `domain/anchor.ts`、`office-viewer` 设计规格（137 行）。

### 1.2 许可（先看这个再谈搬运）

| 对象 | 许可 | 证据 |
|---|---|---|
| ZCode 主仓 | Apache-2.0 | `clones/zcode/LICENSE:2` |
| 四个办公插件的技能与脚本 | Apache-2.0（Copyright (c) 2026 Z.ai） | `documents-plugin/skills/docx/LICENSE.txt` |
| 查看面板运行时 `office-runtime` | MIT | `docs/plugins/2026-10-15-office-viewer-builtin-panels-spec.md:15` |

**结论：可以搬，但必须留版权与 NOTICE。** 一个坑：四个 `SKILL.md` 的 frontmatter 写的是 `license: Proprietary`（`docx/SKILL.md:7`），与同目录 `LICENSE.txt`（Apache-2.0）矛盾。以 LICENSE.txt 为准，但**正式搬运前需要一次法务口径确认**，并在 `THIRD-PARTY-NOTICES` 里落条目。

---

## 2. ZCode 结构解剖（只留与办公有关的）

### 2.1 三层形态，一个内核

桌面（Electron）/ Web / 终端三形态共用 `apps/zcode-cli` 的 agent 内核；`packages/*` 是宿主与共享 UI。**这套多宿主 + 远程（SSH/WSL）+ owner/lease 的架构，对单进程 Wails 是纯负债**（s1：AGENTS.md:62/64/68-69、`persisted-remote-session-path-repair.ts` 184 行）。

### 2.2 四个办公插件（本次调研的真正标的）

`apps/zcode-cli/packages/{documents,presentations,spreadsheets,pdf}-plugin/`，每个包含四件套：

```
.zcode-plugin/plugin.json   插件清单（skills / agents / mcpServers / ui.surfaces）
agents/visual-judge.md      只读视觉验收子代理（独立模型、max 思考档）
skills/<fmt>/               SKILL.md 路由 + routes/ + scenes/ + references/ + scripts/
skills/<fmt>-viewer/        查看面板技能：open_document / show_in_viewer / capture_view
```

插件内 `plugin.json` 的三段声明值得记住（`documents-plugin/.zcode-plugin/plugin.json:15-40`）：技能（给模型）、MCP server（持有文档状态）、UI surface（给用户看）。**"服务持数据、页面持视图"这条纪律是它最干净的设计**（s4）。

### 2.3 技能是四层渐进披露

以 docx 为例（实测字数与 token 估算来自 s7）：

| 层 | 文件 | 体量 | 何时加载 |
|---|---|---|---|
| L0 总纲 | `SKILL.md` | 315 行 / ≈5.6k tok | 命中技能即全文 |
| L1 路由 | `routes/{create,edit,format,comment,read}.md` | 3~12KB | 按意图选一 |
| L2 场景 | `scenes/{report,contract,official-doc,resume,academic,exam,copywriting}.md` | 5~34KB | 按关键词选一 |
| L3 参考 | `references/{common-rules,design-system,ooxml,toc,math,chart-templates,faq,decorations,docx-js-*} .md` | 8~80KB | 按步骤拉 |

**实测拐点**：合同场景冷启动一次性 ≈ **46.5k tok**，其中 `design-system.md`（80KB）一项占 48%，且被 `SKILL.md:37` 无条件要求加载。也就是说"分层"在 docx 上只省掉了约 12k token，**并没有真正实现按需**（S7 §2）。这条对 gaea 是反向教训，不是正面样板。

### 2.4 生成与验收是两条分开的线

- **生成**：docx-js（Node）从零排版 → Python 做 zip/OOXML 后处理；pptx 用 pptxgenjs + python-pptx；xlsx 用 openpyxl 写活公式；pdf 有四条互不共享的栈（ReportLab / HTML+Playwright / Tectonic LaTeX / LibreOffice）。
- **验收**：`postcheck.py`（结构机检）→ `visual-judge` 子代理（看渲染页 PNG，逐页一行 JSON 判决）→ 回修。

**这条"生成与验收分离、验收由独立模型做"的工序，是 ZCode 最值得抄的一条工程思想。**

### 2.5 但它是半成品，别照单全收

调研里挖到的实证（都在子报告里带行号）：

- `SKILL.md:187` 宣称 `postcheck.py` 查 **14 条**，实际 `run_all_checks` 只跑 **9 条**；表格边距、cantSplit、CJK 缩进、文档清洁度、报告内容质量五项**没有实现函数**（s7 §3）。
- `pdf_qa.py` 的 `LAST_PAGE_MIN_FILL` 定义被注释、`:118` 仍引用——一旦启用即 NameError，而 `run_qa` 恰好没调用它（s8 §3.6）。
- 封面配方 R1–R7 只有路由与调色板，**没有 `buildCoverRX()` 实现**；R6/R7 文档与实现不对应（s7 §4）。
- `Tectonic` 内置二进制仅 macOS arm64；`convert.html` 引用不存在的 `html2pdf.js`；Creative 画布写死 720×960（s8 §3.6）。
- 两个插件 `SKILL.md` 标 Proprietary、`LICENSE.txt` 是 Apache-2.0（见 §1.2）。

**论证含义**：把 ZCode 当"标杆"是对的，当"标准答案"是错的。凡是要搬的规则，必须逐条在读源码时确认"它真的实现了"，否则会把它的欠账搬成我们的欠账。

---

## 3. 本质差异：为什么不能照搬 coding agent 的自主动线

### 3.1 真相源不同

| | coding agent | 办公 agent |
|---|---|---|
| 真相源 | 仓库最终状态 | 用户对交付物的认可 |
| 可判定性 | 高（编译/测试/diff/幂等重放） | 低（"这份合同读起来对不对"） |
| 单位成本 | 改错一行，重跑即可 | 改错一页，格式可能全崩 |
| 回滚 | git | 需要文件级 checkpoint + 修订制 |
| 完成判据 | 测试绿 | 人拍板 |

### 3.2 由差异推出的三条设计律

**律一：把"可判定"的尽量变成机器门禁，把"不可判定"的尽量变成人的低成本确认。**
办公里可判定的是：页边距、字体回退、行距、标题层级、TOC 页码域、表格跨页、图片纵横比、单元格错误值、列宽溢出、口径一致性。这些恰好是 ZCode 的 `postcheck` / `audit` / `validate` 覆盖的，而 gaea 只在**文本层**有（`GaeaDocumentLint` 把 docx 转 Markdown 后查红头要素/造价表式，`internal/app/gaea_lint.go:48-66`），**OOXML 层一条都没有**；`Verifier` 又只做 before/after 像素差异（`internal/app/gaea_verify.go:66-67`：0.02 pass / 0.20 warn）。不可判定的是"内容好不好"，那要人看——所以锚点、定位、并排预览比"自动打勾"值钱。

**律二：写操作的粒度必须细到"页/sheet/选区"，而不是"文件"。**
ZCode 的权限链是 15 层有序判定 + 规则号（s1 M1），但它 90% 的规则服务 shell（`permission/service.ts:622-624` 的 `isDestructiveTool` 只认 Bash）。办公要的是"删页要问、读页不问、覆盖要问、另存不问"。gaea 现在的审批是文件级 + 全局单槽串行（`controller_approval.go:245`），并行子代理会排队点卡（s5 G6）。

**律三：跨压缩的连续性，在办公里等于"素材与大纲不能丢"。**
ZCode 的 compact 四件套（续接话术、transcript 可回溯、已读文件重放、计划文件引用）是真东西（s1 M2）。但 gaea 的 compact 阈值已经更细（0.5/0.8/0.9，`internal/gaea/agent/compact.go:34-37`），缺的是**摘要模板没有交付物与口径段**（`compact.go:73-98` 是纯工程向措辞，s5 中危项）。这条不用搬机制，改模板即可。

---

## 4. 四个真空：gaea 办公到底缺什么

### V1 工艺知识真空（有纪律，没工艺）

gaea 的办公技能写得很好，但都是"作业纪律"：

- `office-edit`：先读后写、写后回读、宁拒不误改、逐 run 编辑、soffice 渲染取证（`internal/gaea/skill/builtins.go:45-100`）
- `office-gongwen`：GB/T 9704 版面、字体字号、结构层次、常见错误清单（`internal/gaea/skill/gongwen_body.go:1-89`）
- `schedule-edit`：CPM 裁决、工作日口径、基线对比（`builtins.go:106-178`）

**问题不在质量，在覆盖**：目前只有"公文"一种文档体裁有版面知识。报告、合同、简历、学术、试卷、提案——全无。而 ZCode 有 7 本 scene（合计约 150KB），每本含：6 种模板结构、章节内容要求、内容质量约束、图表占位约定（`docx/scenes/report.md` 目录：19/35/48/114/128/161/224 行）。

**判据性证据**：gaea 的系统提示词里关于文档的要求只有一句"报告/方案必须先列大纲再填充"（`internal/gaea/agent/single_prompt.go:45`）。它没有任何排版参数。

### V2 确定性门禁真空（有 diff，没质检）

gaea 的验收是三件套：结构回读（技能里的纪律）、渲染取证（`soffice` → PDF → PNG）、before/after 像素 diff（`gaea_verify.go:75-124`）。另有一层文本体检：`GaeaDocumentLint` 用 `docmd.Convert` 把 docx 转成 Markdown，再交给 `internal/office/standard` 的可插拔检查器（现有两个：`GB/T 9704 红头要素`、`造价工程表式`，`standard/registry.go`），但**它只判内容要素，不判版式**，且只在 UI 绑定上。

**缺的是"绝对质量"判定**：一份从来没被改过的、第一次生成的报告，diff 恒为 0，Verifier 会给 pass——但它的 TOC 可能是空的、页码可能是 `1decimal`、标题可能跳级。这正是 ZCode `postcheck` 的位置。

ZCode 真实实现的 9 条（s7 §3 逐条核过函数体）：空白页、封面溢出、行距一致性、图片越界、图片纵横比、字体回退、标题跳级、ShadingType 误用、TOC 质量。加上 xlsx 的 10 类（错误串/零值/隐式数组/越界/含表头/小聚合/模式不一致/禁用函数/文本伪公式/.rels 绝对路径）与 poster 的 31 条文本规则（s8 §6.1）。**这些规则全部是纯文本/纯算术判据，没有一条依赖 Python。**

### V3 视觉验收真空（有 judge，但只看文本）

gaea 有独立 judge（`agent/judge.go:14-26`），判"目标是否真的达成"，证据来自 transcript。这是文本证据。

ZCode 的 `visual-judge` 判的是**渲染页**：给它 PNG 路径，逐页返回 `{"page":3,"verdict":"fail","issues":[{"category":"Design","problem":"chart overlaps the caption text below it","evidence":"..."}]}`（`documents-plugin/agents/visual-judge.md` 的 frontmatter description 与 "Output — one JSON line per page" 段）。它按格式给了专门的检查项——pptx 看"投影距离"、docx 看"分页伪影/TOC 无页码/图空白"、xlsx 看"#### 截断/错误值/宽表切页"、pdf 看"跨页边距/多栏断流"。

**这两者不重复**：judge 判"做没做对"，visual-judge 判"看起来能不能交"。gaea 现在只有前者。

**可行性证据**：gaea 已经具备全部原料——`soffice` 转 PDF、`pdftoppm` 逐页 PNG（`gaea_verify.go:51-56` 的 `verifyRenderPages = docmd.RenderPDFPages`），只是这些能力**只挂在 UI 绑定上**（`bindings_office.go:211` `GaeaVerifyRecord`），模型调不到（s5 G3）。

### V4 指认真空（引用是纯文本）

gaea 的"引用到对话"是把选中文字塞进输入框（`DocxPreview.tsx:83-89`、`onQuoteSelection`）。模型收到的是一段文字，不知道它来自第几段、第几页、哪个单元格；回答时也无法"指回去"。

ZCode 的锚点是三方共享的地址：`<workspace-relative path>#<locator>`（`office-runtime/src/domain/anchor.ts:1-11`）：

```
docx  p:14 · p:3-9 · t:2 · t:2/r3c1
pptx  slide:3 · slide:2-4 · slide:3/shape:7
pdf   page:5 · page:2-4
xlsx  Revenue · Revenue!B2 · Revenue!B2:F9
```

配套四个动作（`office-runtime/src/CONTRACT.md:12-20`）：`open_document`（开+回大纲）、`read_document(anchor)`、`search_document`、`show_in_viewer(targets, note)`（滚到并高亮，用户可逐条步进）、`get_view`（用户在看什么/选了什么）、`capture_view`（把当前渲染位置截成 PNG，给视觉判断用）。

**这是本次调研里"投入最小、体验差异最大"的一项**：gaea 已有 `sidebar_open`（`tool/builtin/sidebar_open.go:38`，纯 UI 动作、只读、可并行），再加"定位到锚点"和"回报当前视图"就是完整闭环。

---

## 5. 蒸馏清单

分级：**K** 知识 / **Q** 门禁 / **A** 锚点与感官 / **C** 契约。成本 S≤1 人日、M≤1 周、L>1 周。

### 5.1 K 知识层（搬文本，不搬代码）

| 编号 | 项 | 来源 | gaea 落点 | 成本 |
|---|---|---|---|---|
| K1 | docx 场景工艺包（report / contract / official-doc / resume 四本起） | `docx/scenes/*.md`、`references/common-rules.md`、`references/toc.md` | `internal/gaea/skill/` 新增 `office_docx.go`（沿 `gongwen_body.go` 体例），场景文本走 embed | M |
| K2 | 中文排版阈值表（twips/pt 换算、字号表、行距 312、缩进 480/420、GB/T 9704 页边距 2098/1984/1588/1474） | `references/common-rules.md:5-116`、`scenes/official-doc.md`、`gongwen_body.go` 现有数值 | `internal/gaea/office/typography.go` 常量 + 技能引用 | S |
| K3 | WPS/Word 兼容黑名单 | `references/common-rules.md:169-233` | 并入 Q1 规则集 + 技能提示 | S |
| K4 | xlsx 设计令牌与财务模板结构（12 套色板、auto_fit 的 CJK 1.7 宽估算、finance/finance_lite） | `xlsx/templates/{base,palettes}.py`、`engines/design.md`、`scenes/finance*.md` | `internal/gaea/office/xlsxstyle` + 技能 | M |
| K5 | PDF 排版判据（Measure-first、Max_Width 铁律、降级阶梯 14/12pt、页码五区制、末页 40/25%、大节 25% 换页、0.382 黄金锚） | `pdf/typesetting/{overflow,pagination,geometry,fill-engine}.md` | 导出管线检查（soffice 产出后断言）+ 技能 | M |
| K6 | 技能分层重构（把 `office-edit` 一整段拆成 route + references，实现真正的二级加载） | 反向借鉴：ZCode 的分层没做到按需（§2.3） | `internal/gaea/skill/skill.go`（现 `:384-422` 一次性拼正文）+ `index.go` | M |

**K1 的验收判据**：同一份"季度经营分析报告"在两版技能下各生成一次，对照清单（TOC 域存在 + 页码格式 + 页边距 + 行距 + 标题层级 + 图表题注）通过率 ≥90%，且正文可读性人工盲评不降。
**K1 的风险**：与"修订制"冲突——这套工艺只适用于**新建文档**，改既有文档仍走 `docxedit` 的 run 级手术。技能开头必须写明分流，否则模型会拿 docx-js 去重排用户的文件（这是最严重的失败模式）。

### 5.2 Q 门禁层（搬规则，Go 重写）

| 编号 | 项 | 来源 | 落点 | 成本 |
|---|---|---|---|---|
| Q1 | docx 机检 9 条 | `docx/scripts/postcheck.py`（实测实现项） | `internal/office/standard` 扩 OOXML 检查器（现有 Registry 只收文本级），入口沿用 `GaeaDocumentLint`，并补模型可调工具 | M |
| Q2 | xlsx 机检 10 类 | `spreadsheets-plugin/skills/xlsx/xlsx.py`（audit/scan/validate） | 挂 `xlsxedit.ApplyOps` 后置 | M |
| Q3 | HTML/poster 31 条文本规则（仅当做 PDF 模板导出时） | `pdf/scripts/poster_validate.py` | `internal/gaea/office/pdfqa` | M |
| Q4 | visual-judge 子代理 | `agents/visual-judge.md` 全文 | `internal/gaea/agent` 新增只读子代理模板 + 渲染工具 | M |
| Q5 | 三层验收分工定义 | —（我们自己的设计） | 文档 + 代码注释 | S |
| Q6 | 导出前"体检报告"呈现 | `postcheck` 输出形态 | 交付卡片 | S |

**Q1 的设计要点**（这是本轮最像"工程"的一项）：

1. **只做能断言的**。ZCode 的 14 条里 5 条没实现（§2.5），我们不做"宣称做了"的规则。首版 9 条：空白页（section 尾 PageBreak + 连续空段）、TOC 存在性与 `w:instrText` 的 `\* arabic`、标题跳级、行距一致性、CJK 首行缩进、图片纵横比、表格跨页（`cantSplit`/`tblHeader`）、字体回退（`w:eastAsia` 缺失）、`ShadingType` 黑块。
2. **Go 直读 OOXML，不起 Python**。gaea 的 `internal/gaea/office/ooxml` 已有条目序保真与解析能力，规则全是字符串/结构判断，加 Python 运行时会把"桌面分发体积"这条线弄脏。
3. **判据必须分档**：新建文档与编辑既有文档的检查项不同（编辑不该要求"有封面"）。

**Q4 的协议**（照搬 ZCode 的三条硬约束，这三条是它最有价值的部分）：
- **只读**：不许改文件，只能 Read 预渲染的 PNG；不许看 pptx/docx 原文冒充看图。
- **逐页一行 JSON**：`category ∈ {Spec, Visual, Design, Unverified}`，每条必须给"看到了什么"的具体证据，禁止泛泛美化建议。
- **只能有一个视觉门**：派了 visual-judge 就不要再自己看一遍图（ZCode 明文禁止重复评审，`visual-judge.md` 的 description 段）。gaea 要补一条：**与 `Verifier` 的像素 diff 分工写死**——diff 判回归（改前改后是否一致），judge 判绝对质量（这一页能不能交）。

### 5.3 A 锚点与感官层（搬契约，改前端）

| 编号 | 项 | 来源 | 落点 | 成本 |
|---|---|---|---|---|
| A1 | 锚点语法 + `show_in_viewer` 定位（docx p/t、xlsx Sheet!A1、pptx slide/shape、pdf page） | `office-runtime/src/domain/anchor.ts`、`CONTRACT.md:12-17` | `internal/gaea/office/anchor.go` + 前端 `lib/anchor.ts` + 定位工具 | M |
| A2 | `report_view`：用户选中/滚动 → 模型可知 | `CONTRACT.md:9,35-37` | 新工具 `get_view` + 预览节流上报 | S |
| A3 | `capture_view`：模型截图自己的第 N 页 | `CONTRACT.md:20` | 新工具（复用 `docmd.RenderPDFPages`） | S |
| A4 | 引用三态 + 每轮文件摘要 + 交付物"打开方式" | `gen-ui/domain/references.ts`、`docs/ui/task-file-change-summary.md:97-107`、`ArtifactOpenAsFileButton.tsx:156-213` | `frontend/src/gaea`（GenUI 解析、轮尾摘要、`DeliverableCards`） | S |

**A1 的最小切口**：先只做 docx 的 `p:N` / `t:N/rRcC` 与 xlsx 的 `Sheet!A1`。pptx 的 `shape:ID` 需要读 `cNvPr` id，值和 PPT 内部对象绑定，收益/成本比不如前两者。
**A2/A3 的价值**：这两条把"用户在看哪"和"模型看到了什么"变成同一份事实。有了它，用户说"这里改一下"，模型不必反问是哪一段；模型说"第 4 页图表压住题注"，用户可以一键跳过去核。
**A4 的取舍**：ZCode 的分段"打开方式"按钮（默认 App / 已装编辑器 / 在资源管理器显示 / 复制绝对路径）依赖宿主枚举本地编辑器，gaea 是 Wails 单进程，只做"预览 / 定位 / 复制路径"三项即可（s3 报告已标注 `lib/bridge` 未核实，落地前需确认）。

### 5.4 C 契约层（搬思想，改语义）

| 编号 | 项 | 来源 | 落点 | 成本 |
|---|---|---|---|---|
| C1 | 投影版本（`projectionVersion` + `based_on_version`）替代 size+mtime 指纹 | `write.ts:278-304` 的 CAS 思想 | `agent/file_observation` + `office` 编辑前置检查 | M |
| C2 | 工具结果结构化截断（office 结果不做字节截断） | `result-serialization.ts:175-232` | `tool/envelope.go` | S |
| C3 | 可编程错误码（`TARGET_NOT_FOUND` / `AMBIGUOUS` / `STALE_PROJECTION`） | `EditErrorCode` | `tool/builtin/errcode.go` 扩枚举 | S |
| C4 | 两阶段 apply（docx 结构化 op：计划全验 → 统一落盘） | `apply-patch.ts:67-216` | `internal/gaea/office/docxedit` 暴露 `PlanOps/ApplyOps`，与 `xlsxedit` 对齐 | M |

**C1 的论证**：ZCode 的"写前 CAS"前提是"模型读过字节"。**这个前提在 Office 上不成立**——模型从没拥有过 docx 的字节，它只有预览投影（`xlsxpreview` 抽样、docx 渲染、pptx 缩略）。所以 gaea 要做的是"投影版本"：预览时记下 `projectionVersion`，编辑时要求 `based_on_version` 匹配，不匹配就重读。s2 报告把这条列为"最该补的主线索"，我同意，且它的收益会同时收敛 stale / 幂等 / 自检 / 交付四处。

**C4 的现状**：xlsx 已经有了（`xlsxedit.PlanOps/ApplyOps`，`bindings_office.go:215,218`），docx/pptx 还是"选区替换"单点（`:79`、`:123`）。C4 就是把 xlsx 的范式复制到 docx——但只在模型侧需要，UI 的"框选即改"保持现状。

---

## 6. 不采清单（含"看似该抄其实不该"）

| 编号 | 不采 | 理由 |
|---|---|---|
| N1 | 插件市场、安装/更新/卸载、CDN seed、MCP 宿主、UI Plugin（MCP Apps + ui.surfaces） | 单机 Wails 无生态诉求，白背攻击面；gaea 已有 genui 与办公预览，再引一套就是两套页面协议。只保留"服务持数据、页面持视图"这条纪律。**长期规划红线：不做技能市场/分发平台**（`gaea-longterm-plan-2026-10.md:27`） |
| N2 | shell 中心的权限模型、终端、bash 工具族（37 个文件） | 办公的权利维度是"改哪个文档哪一部分"，不是"能不能执行命令"。gaea 的权限已按 read/write/path 分（`permission.Policy.Decide`），缺的是细分维度（页/sheet），不是 shell 语义 |
| N3 | git 工作流（`git-snapshot.ts` 把 `git status` 注入上下文） | 办公要文件级 checkpoint，不要版本库语义。gaea 的 `JournalStore` + 基线快照已经是更对的形态 |
| N4 | Electron 多宿主 + owner/lease + SSH/WSL 远程链路 | 单进程 Wails，纯负债 |
| N5 | Python/TS 渲染重栈（Tectonic、Chromium 海报链、ReportLab 封面引擎） | 体积、离线、字体度量、faux-bold 全要重做，L~XL。gaea 走 `soffice` 更划算 |
| N6 | 封面配方 R1–R7 + `design-system.md`（80KB） | 与"只改用户文档、不造封面"的修订制范式冲突；且 R6/R7 没有实现（§2.5），搬了会踩空。**只取其中"封面标题排版 10 条规则"的文字部分**（`common-rules.md:285-365`） |
| N7 | LibreOffice "HARD REQUIREMENT" 强制装机话术 | 与 gaea 的降级策略正面冲突（`gaea_verify.go:79-82`：soffice 不可用 → `warn: 视觉渲染降级`）。桌面应用不能命令用户装几百 MB。可以把"未装则视觉验收不可用"如实告知，但不能替用户做决定 |
| N8 | goal verifier 的 fail-open + 自动续跑、skill 作为可改运行时（allowed-tools/model/effort/fork）的载荷 | 前者在办公会把"没做完"判成"做完了"；后者 ZCode 自己已降级为默认 ask，gaea 的 `runas`+`model`+`allowed-tools` 已够，再往前是权限后门 |
| N9 | ZCode 的 compact / 队列 / 重试全套语义 | gaea 阈值更细、队列已有 Steer/撤回（`ComposerQueueList`），净增量小。只取"摘要模板补交付物与口径段"一条（§3.2 律三） |

---

## 7. 分期与验收

**约束**：不新增板块（`:5`）；域规则一律技能化（`:29`）；每刀三项验证（typing/lint/测试 + 走查 + 回归）。

### P0 · 一周内可验证（约 5~6 人日）

| 项 | 交付 | 验收判据 |
|---|---|---|
| K2 | `typography.go` 常量 + 技能引用 | 单测断言：1cm=567twips、1pt=20half-points、A4=11906×16838、行距 312、缩进 480/420、GB/T 9704 四边距 |
| K3 | 兼容黑名单（文字 + 规则） | 构造 5 个反例 docx，机检 5/5 命中 |
| Q1 | docx 机检首版 9 条 | 9 条各有正/反例测试；对 3 份真实交付物跑，命中 ≥1 真问题 |
| A1+A2 | docx 锚点 + `get_view` | 用户选中第 12 段 → 模型回答里出现 `p:12`；模型 `show_in_viewer(p:12)` → 面板滚动并高亮 |
| 附 | PDF 文本 CJK 部首归一（U+2F00–2FD5 / U+2E80–2EF3） | 用 `⽂`（U+2F42）构造 PDF，`knowledgeimport.ExtractText` 输出为 `文`；检索命中 |

> 附项来自 ZCode `office-runtime/src/domain/cjk.ts` 的实证（`docs/plugins/2026-10-15-office-viewer-builtin-panels-spec.md:118-120`）：Chrome/Skia 打印的 PDF 把部分汉字 ToUnicode 写成部首兼容字符，画面正确但复制/查找/引用全错。gaea 的 `knowledgeimport.go:59-69`（docx/pdf → Markdown）与知识库检索**完全没有这层归一**，属于确定性 bug，不是增强。

### P1 · 2~3 周（5 项，M）

K1（先 report + contract 两本）· K6（技能分层）· Q2（xlsx 机检）· Q4（visual-judge）· C1（投影版本）
**出口判据**：一份 30 页报告从"生成 → 机检 → 视觉验收 → 回修"跑通，模型自主完成其中三轮，失败项以 JSON 形式回到对话、由人拍板是否继续；同一文档二次编辑不会因投影陈旧而错改。

### P2 · 观察（不进本轮）

K4（xlsx 令牌与财务模板）· K5（PDF 排版判据）· Q3（HTML 规则）· A3（`capture_view`）· A4（交付物打开方式）· C2/C3/C4（契约层余项）· pptx/xlsx 锚点。

### 与长期规划的对位

- 阶段九 9.1（compact 双阈值）→ **不另立项**，只改摘要模板（§3.2 律三）。
- 阶段十一 11.5（办公真编辑立场）→ 本轮正是它的"工艺与门禁"支撑。
- 阶段十一 11.6（办公规范包 GB/T 9704 技能化）→ 已由 `office-gongwen` 完成；K2 是把其中数值抽成常量，K1 是把它扩成场景族。

---

## 8. 风险与红线

| 风险 | 说明 | 对冲 |
|---|---|---|
| **双轨写盘** | gaea 现在有两条路：模型 `bash`+Python 技能 改文档，UI 走 Go 原生 `docxedit`。二者对同一文件的并发改写会互相踩（s5 G1/G2） | 任何新工具都必须走 Go 原生 op 或明确声明"Python 路径"，且写后进证据链；`bash` 写盘补 `RecordChange`（s2 §8） |
| **双门禁** | Verifier 像素 diff + visual-judge + postcheck 三套判定，用户会看到互相矛盾的结论 | Q5 写死分工：结构机检（绝对）→ 视觉 judge（绝对）→ diff（回归）。冲突时以机检为准，judge 只报"能否交" |
| **Prompt 膨胀** | ZCode 合同场景冷启动 46.5k tok（§2.3）。gaea 的 L1 前缀是缓存稳定区，技能索引/事实/晨报全拼在里面（s4 发现：`boot/sysprompt.go:43-134`） | K6 必须做真二级加载；技能正文不进 L1；K1 每本控制在 ≤8k tok |
| **字体与体积** | docx/xlsx 的版面判据依赖具体字体；缺少字体时判据会误报 | 判据分"字体无关"（页边距/行距/层级）与"字体相关"（回退/宽度）两档，后者降为 warn |
| **许可** | `SKILL.md` 标 Proprietary、`LICENSE.txt` 是 Apache-2.0（§1.2） | 搬运前一次法务确认；保留 NOTICE；不搬商标与品牌素材 |
| **视觉语言** | 办公页现在是"Codex 对话 + 星枢面板 + Luminous Glass"三套并存（s6 判断） | 本轮所有前端改动只做加法，不动玄墨 v11 视觉；视觉统一单独立项 |

---

## 9. 反方案：为什么不这么干

**反方案 A：整体移植 ZCode 的办公插件（Python + TS）。**
不成立。三个硬理由：① 宿主不兼容——它的插件靠 MCP server + UI surface 分发（`plugin.json:15-40`），gaea 是单进程 Wails，没有 MCP 宿主；② 体积——技能脚本 + Python 依赖 + Playwright/Tectonic 会让安装包翻倍；③ 写盘路径分裂——模型走 Python、UI 走 Go，正是 s5 已经认定的 G1/G2 两个严重缺口，移植只会加深。**正确做法是把它的"知识"搬进技能、把"规则"重写成 Go、把"协议"改造成宿主原生的工具。**

**反方案 B：先做插件市场 / 技能市场。**
红线。长期规划明令不做（`:27`），且单机产品没有生态诉求。

**反方案 C：先做 PDF 排版引擎（Tectonic/Playwright）。**
投入产出比最差。gaea 的 PDF 出口目前是 `soffice` 转换（`gaea_pdf.go`），够用；排版判据（K5）可以只做"检查"不做"排版"——先能指出"第 7 页末页填充率 12%，建议并页"，就已经比现在强。

**反方案 D：照搬 ZCode 的"自动继续直到完成"。**
办公的完成判据是人。gaea 已有 goal gate + judge（`agent/judge.go`），再叠一层自动续跑，会把"用户没确认"误判成"任务未完成"，反复烧 token 且改动用户文件。

---

## 10. 关键证据索引

| 结论 | 证据 |
|---|---|
| 四个办公插件及其四件套 | `clones/zcode/apps/zcode-cli/packages/{documents,presentations,spreadsheets,pdf}-plugin/` |
| 技能四层与 46.5k tok 冷启动 | `documents-plugin/skills/docx/SKILL.md:33-94`；s7 §2 |
| postcheck 宣称 14 实跑 9 | `SKILL.md:187` vs `scripts/postcheck.py` `run_all_checks` |
| visual-judge 协议 | `documents-plugin/agents/visual-judge.md`（description + Output 段） |
| 锚点语法 | `office-runtime/src/domain/anchor.ts:1-11` |
| 查看面板四动作 | `office-runtime/src/CONTRACT.md:12-20` |
| CJK 部首归一 | `docs/plugins/2026-10-15-office-viewer-builtin-panels-spec.md:118-120` |
| 许可 | `LICENSE:2`、`skills/docx/LICENSE.txt`、规格 `:15` |
| gaea 无原生办公工具 | `internal/gaea/tool/builtin/` 全量（46 个工具，无 docx/pptx/xlsx） |
| gaea 办公编辑只在绑定层 | `internal/app/bindings_office.go:79,123,215` |
| gaea 只有像素 diff | `internal/app/gaea_verify.go:66-67` |
| gaea 有文本级文档体检 | `internal/app/gaea_lint.go:48-66`、`internal/office/standard/registry.go:21-54` |
| gaea 有文本 judge | `internal/gaea/agent/judge.go:14-26` |
| gaea 有 sidebar_open | `internal/gaea/tool/builtin/sidebar_open.go:38` |
| gaea 技能是整段正文 | `internal/gaea/skill/builtins.go:45-100`、`skill/skill.go:384-422` |
| gaea 文档提示词只有一句 | `internal/gaea/agent/single_prompt.go:45` |
| gaea compact 阈值 | `internal/gaea/agent/compact.go:34-37` |

---

## 11. 子报告索引

| 文件 | 内容 |
|---|---|
| [`s1-agent-runtime.md`](research/zcode-distill/s1-agent-runtime.md) | 33 个运行时机制、8 条不采、迁移矩阵 |
| [`s2-tooling.md`](research/zcode-distill/s2-tooling.md) | 工具契约四件套、Office 水土不服三条、12 项 ROI |
| [`s3-genui-ux.md`](research/zcode-distill/s3-genui-ux.md) | 交互 15 条候选、5 抄 3 不抄、P0 路线 |
| [`s4-extension.md`](research/zcode-distill/s4-extension.md) | 声明式子代理、hook 信任、技能分层、5 个技能建议 |
| [`s5-gaea-office-baseline.md`](research/zcode-distill/s5-gaea-office-baseline.md) | gaea 办公后端骨架与缺口 Top 8 |
| [`s6-gaea-office-ux.md`](research/zcode-distill/s6-gaea-office-ux.md) | gaea 办公前端组件地图与交互缺口 Top 8 |
| [`s7-docx-pptx-plugin.md`](research/zcode-distill/s7-docx-pptx-plugin.md) | docx/pptx 逐文件拆解、10 条知识资产 |
| [`s8-xlsx-pdf-plugin.md`](research/zcode-distill/s8-xlsx-pdf-plugin.md) | xlsx/pdf 管线、14 条规则集、10 个源码坑 |
