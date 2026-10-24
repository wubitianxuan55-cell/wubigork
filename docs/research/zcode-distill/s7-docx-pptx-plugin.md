# S7 · ZCode 官方 documents / presentations 插件拆解：可移植粒度评估

- 日期：2026-10-24
- 研究对象（只读）：`clones/zcode/apps/zcode-cli/packages/documents-plugin`、`.../presentations-plugin`
- 对照对象：gaea 本体（`internal/app/gaea_docx_edit.go`、`internal/app/gaea_pptx_edit.go`、`internal/office/docxedit`、`internal/office/pptxedit`、`internal/app/gaea_verify.go`、`internal/gaea/skill/builtins.go`）
- 产出定位：给 gaea 办公域做「资产搬运清单」，分「原样搬 / 必须改写 / 不该搬」三档，逐条给成本（S/M/L）与风险。
- 纪律：全部结论带 `file:line` 证据；读不到的地方直说「读不到」，不做推断式补全。

路径缩写（全文一致）：

| 缩写 | 真实路径 |
|---|---|
| `DOCX/` | `clones/zcode/apps/zcode-cli/packages/documents-plugin/skills/docx/` |
| `DP/` | `clones/zcode/apps/zcode-cli/packages/documents-plugin/` |
| `PPTX/` | `clones/zcode/apps/zcode-cli/packages/presentations-plugin/skills/pptx/` |
| `PP/` | `clones/zcode/apps/zcode-cli/packages/presentations-plugin/` |

---

## 0. 一句话结论

ZCode 的 docx 能力是 **「docx-js（Node）负责从零生成 + Python 负责文档级后处理/编辑 + 脚本机检 + LLM 视觉验收」** 的四段式管线，知识资产（412KB Markdown / 9,631 行）比代码资产（139KB Python / 3,546 行）重得多；gaea 的办公是 **「Go 字节级真编辑（修订制）+ LibreOffice 渲染 + 像素 diff 复核」**。因此**知识层几乎可以整块搬（prompt/规则/机检清单），执行层几乎全部要改写（Node/Python 双运行时 vs 单 Go 二进制），面板/协议层完全不该搬**。

---

## 1. 资产盘点与体量

| 维度 | documents-plugin | presentations-plugin |
|---|---|---|
| 文件数 / 总字节 | 45 / 599,715 | 6 / 66,994 |
| Markdown | 26 文件 / 412,514 B / 9,631 行 | 3 文件 / 64,244 B（SKILL.md 55,255 B 单文件） |
| Python | 6 文件 / 139,256 B / 3,546 行 | 0 |
| 目录结构 | `skills/docx/{routes,scenes,references,scripts,env_setup}` 四层 | `skills/pptx/SKILL.md` 单文件（无 routes/scenes/references/scripts） |
| 附带 Agent | `agents/visual-judge.md`（5,426 B） | 同文件、**SHA256 完全相同**（`472147…27221`） |
| MCP / UI | `plugin.json` 声明 `mcpServers.documents` + `ui.surfaces` | 同构，`presentations` |

体积分布（docx）：`references/` 201KB > `scenes/` 140KB > `routes/` 29KB > `scripts/` 139KB(Python) > `env_setup/` 44KB。

**结构性判断（重要）**：docx 是「重知识 + 分层」，pptx 是「重知识 + 单文件」。pptx 没有 `routes/scenes/references` 分层，583 行的 SKILL.md 一次性全量进上下文（见 §3 的 token 实测），它靠「Part 1 设计原则 / Part 2 pptxgenjs API」内部小标题做伪分层。

---

## 2. Q1 · 技能路由结构：总纲 → route → scene → reference 的渐进式披露

### 2.1 四层职责与字数

| 层 | 载体 | 体量 | 职责边界（证据） |
|---|---|---|---|
| L0 门面 | `SKILL.md` frontmatter `description` | 1 段（约 100 词） | 只做**触发词覆盖**：创建/编辑/转换/渲染/`Word转PDF`/`doc转docx` 全塞进一行（`DOCX/SKILL.md:6`）。这一行是常驻上下文的唯一成本 |
| L1 总纲 | `DOCX/SKILL.md` | 315 行 / 20,240 B ≈ **5.6k tok** | 六步路由（`:34-40`）、Task Router 表（`:70-78`）、Scene Router 表（`:80-94`）、全局格式硬规则（`:96-116`）、两层验证清单（`:118-189`）、LibreOffice 硬要求（`:220-267`）、引用协议（`:269-315`）。**不含任何具体文档类型的正文规则** |
| L2 route | `DOCX/routes/*.md` 5 个 | 95–207 行，合计 29,235 B ≈ 8.9k tok | 按**动作**分：`create`/`edit`/`format`/`read`/`comment`。每篇只写「这类动作的执行顺序 + 该读哪个 reference」 |
| L2' scene | `DOCX/scenes/*.md` 7 个 | 139–783 行，合计 140,511 B ≈ 42k tok | 按**文体**分：academic / contract / exam / official-doc / report / resume / copywriting。写「该类文档的结构模板 + 措辞纪律 + 场景字体/页边距覆盖 + 场景专属检查清单」 |
| L3 reference | `DOCX/references/*.md` 10 个 | 229–1,797 行，合计 201,083 B ≈ 64k tok | 按**知识主题**分：design-system（80KB）/ common-rules（25KB）/ decorations（20KB）/ chart-templates / toc / math-formulas / ooxml / docx-js-core / docx-js-advanced / faq |

关键设计点：**L1 里没有正文规则，只有路由表**。`DOCX/SKILL.md:35-40` 的六步是显式「先选 route，再选 scene，再无条件加载两个 reference」：

```
Step 1 任务类型 → route 文件（:35）
Step 2 业务场景 → scene 文件（:36）
Step 3 无条件加载 design-system.md（:37）
Step 4 无条件加载 common-rules.md（:38）
Step 5 按 route 执行（:39）
Step 6 跑生成后检查（:40）
```

**边界纪律值得抄**：TOC 规则号称 single source of truth，其他文件只许引用不许复述（`DOCX/references/toc.md:3`）；封面配方只许 `design-system.md` 定义，禁止复制函数（`DOCX/routes/create.md:44`："the **authoritative source** — do NOT duplicate the function"）。这条「单源 + 引用不复述」是 412KB 知识库没有自相矛盾的唯一原因。

### 2.2 真实场景走查：「合同」完整加载路径 + token 账

触发：「帮我写一份技术开发合作合同」。按 L1 路由表落到 `create` + `contract`。

| # | 加载物 | 触发依据 | 字节 | 估算 token |
|---|---|---|---|---|
| 1 | `SKILL.md` frontmatter description | 常驻（技能目录） | ~700 | ~180 |
| 2 | `DOCX/SKILL.md` | 技能入口，必读 | 20,240 | 5,571 |
| 3 | `DOCX/routes/create.md` | Task Router「Create/write/generate」(SKILL.md:74) | 12,030 | 3,234 |
| 4 | `DOCX/scenes/contract.md` | Scene Router「contract, agreement, NDA…」(SKILL.md:86) | 20,741 | 5,705 |
| 5 | `DOCX/references/design-system.md` | Step 3 无条件（SKILL.md:37） | 80,528 | 22,175 |
| 6 | `DOCX/references/common-rules.md` | Step 4 无条件（SKILL.md:38） | 24,868 | 6,829 |
| 7 | `DOCX/references/docx-js-core.md` | create route 第 4 步（create.md:10） | 10,062 | 2,771 |
| | **合计** | | **168,469 B** | **≈ 46.5k tok** |

（token 估算口径：CJK 字符按 1 tok/字，其余按 3.6 B/tok；本技能正文已被译成英文，CJK 仅数十个，实测 `contract.md` 20,404 字符里只有 51 个汉字。）

补充：合同场景若判断需要 TOC（>20 条，`routes/create.md:73`）再加 `toc.md` +3,117 tok；若带图表再加 `chart-templates.md` ≈ 3.8k tok。**冷启动一次 50k tok 量级**。

### 2.3 这套披露哪里做得好、哪里是假渐进

- 好：`routes` 与 `scenes` 正交（动作 × 文体），组合爆炸被压成 5+7 个文件；`faq.md` 323 行承接「bug 修复知识」而不污染主路径。
- 假：**Step 3/Step 4 无条件加载**，而 `design-system.md` 一项就占全路径的 **48%**（22.2k / 46.5k）。也就是说「渐进式披露」在 docx 里只省掉了 `decorations/faq/ooxml/docx-js-advanced` 这 4 个（约 43KB ≈ 12k tok），最大的一块仍是被强制全量读。合同场景实际只需要 `design-system.md` 的「调色板 + 表格样式 + 排版规则」三节（约 400 行），却付了 1,797 行的钱。
- 对比 pptx：单文件 54,573 字符 ≈ **15.2k tok** 一次读完，看似更贵，但没有二次加载（无 references 目录，`PP/skills/pptx/` 只有 `SKILL.md` + `LICENSE.txt`）。**在中等复杂度任务上，pptx 的单文件反而比 docx 的四层更省**——这是给 gaea 的启示：分层本身有成本，只有当「高频部分能被真正跳过」时才划算。

---

## 3. Q2 · 生成管线、CLI 接口、OOXML 合法性与样式继承

### 3.1 docx：**docx-js 生成 + Python 后处理/编辑**（不是 python-docx 单管线）

明确分工（证据密集，容易误判，逐条列）：

| 环节 | 技术 | 证据 |
|---|---|---|
| 从零生成 | **docx-js（Node/JS 库）**，代码里 `new Document/Packer/Paragraph/TextRun/TableOfContents` | `DOCX/routes/create.md:154-198`（完整 builder 示例）、`DOCX/references/docx-js-core.md`、依赖声明 `DOCX/SKILL.md:213`（`bun add docx` / `npm install docx`） |
| 运行方式 | `bun run generate.js` 或 `node generate.js` | `DOCX/routes/create.md:15` |
| TOC 占位注入 | **Python** 后处理 zip | `DOCX/scripts/add_toc_placeholders.py` |
| 页脚域代码修补 | **Python** 后处理 zip | `DOCX/scripts/fix_footer_fields.py` |
| 编辑既有文档 | **Python** 直操作解包后的 XML（`Document` 库 + `defusedxml.minidom`） | `DOCX/routes/edit.md:60-72`、`DOCX/scripts/document.py:1-27,649-716` |
| 简单批注 | python-docx（仅"Simple comment operations"） | `DOCX/SKILL.md:218` |
| 图表 | **matplotlib → PNG 嵌入**（非原生 chart） | `DOCX/SKILL.md:200-208`、`DOCX/references/chart-templates.md` |
| 复杂公式 | LaTeX → docx-js Math 对象；>3 层嵌套回退 matplotlib PNG | `DOCX/references/math-formulas.md:231-276` |
| 转 PDF / .doc→.docx / 渲图 | LibreOffice `--headless --convert-to` + `pdftoppm` | `DOCX/routes/edit.md:27,86`、`DOCX/references/docx-js-advanced.md:251-270` |

**结论**：docx 是**双运行时**管线（Node 生成 + Python 后处理），python-docx 只在「批注」和「TOC 自动提取标题」两个点上出现（`add_toc_placeholders.py:49` `from docx import Document`）。

### 3.2 pptx：**pptxgenjs 生成 + python-pptx 编辑 + playwright/sharp 备选**

| 环节 | 技术 | 证据 |
|---|---|---|
| 从零生成 | **pptxgenjs**（JS，坐标单位英寸，强制 13.33×7.5" 画布） | `PPTX/SKILL.md:186-188,112`；依赖 `PPTX/SKILL.md:677` |
| 编辑既有 deck | **Approach A：python-pptx（run 级替换，首选）**；Approach B：裸 OOXML（动画/切换/备注/主题） | `PPTX/SKILL.md:561-562` |
| 图表 | 一律原生 `addChart()`，**禁止**渲染成图（除桑基/网络图等无原生表示） | `PPTX/SKILL.md:424-431` |
| 从模板造新 deck | 模板页克隆（`deepcopy` slide element + 重建 rels）或 zip 级复制 | `PPTX/SKILL.md:540-541` |
| 文本提取 | markitdown | `PPTX/SKILL.md:676` |
| 渲图 | `soffice --convert-to pdf` → `pdftoppm -png -r 150` | `PPTX/SKILL.md:162-171` |

**注意一个反向事实**：pptx 有 python-pptx 这条编辑路径，docx 反而没有对称的"编辑用 python-docx"——docx 的编辑走自研 `Document`（XML/DOM 级），因为要支持修订与批注（§3.4）。

### 3.3 脚本暴露的 CLI 接口（可直接对照移植）

| 脚本 | 入口 | 参数 | 输入 | 输出 / 退出码 |
|---|---|---|---|---|
| `scripts/postcheck.py` | `main()` `:766-804` | `<docx_path>`（位置参数）、`--json`、`--strict`（`:769-771`） | 单个 .docx | stdout 人读报告或 JSON 数组；退出码：`0` 通过 / `2` 有 error / `1` strict 且有 warning（`:799-804`） |
| `scripts/add_toc_placeholders.py` | `main()` `:701-744` | `<docx_file>`（原地替换）、`--auto`、`--entries '<json>'`（`:705-713`） | .docx，TOC 域必须已存在 | 原地写回；退出码 `1`=TOC 结构错（`:738-741`） |
| `scripts/fix_footer_fields.py` | `main()` `:254-281` | `<docx_file>`（原地）、`--dry-run`（`:262-266`） | .docx | 原地写回；打印逐条变更（`:272-278`） |
| `scripts/document.py` | **无 CLI**，纯库 | `Document(unpacked_dir, rsid=None, track_revisions=False, author="Z.AI", initials="C")`（`:652-659`） | **已解包的目录**（不是 .docx） | `doc["word/document.xml"].get_node(...)` / `add_comment(start,end,text)` / `suggest_deletion(node)` / `doc.save()` |
| `env_setup/env_check.sh` | `source env_check.sh [--quiet]` `:5-7` | — | — | 导出 `DOCX_SKILL_DIR`/`FONT_DIR`；退出码 0/1（`:41`） |

一条硬约定值得抄：**脚本路径必须绝对化**，禁止 `python3 scripts/x.py` 裸调用，理由写在 `DOCX/SKILL.md:68`（"only works if cwd happens to be the skill directory"）。

### 3.4 OOXML 合法性 与 样式继承 分别靠什么保证

这是本插件最工程化的部分，共 5 道闸：

1. **生成端不手搓 XML**：docx-js / pptxgenjs 负责拼合法 OOXML（`create.md:154-198` 全程构件对象）。
2. **已知生成器缺陷的定点修补**（这是真正值钱的部分）：
   - fldChar 结构 bug：begin+instrText+separate 被合并进同一个 `<w:r>` → `add_toc_placeholders.py:240-283` `_fix_fld_char_structure()` 修；
   - `outlineLvl` 缺失导致 Word 更新域找不到标题 → `:178-239` `_fix_heading_outline_levels()`；
   - `settings.xml` 缺 `updateFields=true` → `:143-177` `_fix_update_fields()`；
   - 页脚裸 `PAGE` 被 WPS 忽略 → `fix_footer_fields.py:43-64` 四条正则（裸 PAGE / 已带格式 / `\* decimal` 错用 / 空 `<w:pgNumType/>`）分类修补；
   - 空 `<w:pgNumType/>` 混淆 WPS → `fix_footer_fields.py:55`。
3. **压缩包级清洗**：重打包前删掉 `*.xml`/`*.rels` 的纯空白文本节点与注释节点（`document.py:46-62` `_condense_xml`），再 `ZIP_DEFLATED` 重打（`:64-79` `_pack_document`）。
4. **样式继承**：三层——`styles.default.document` 全局默认（`create.md:183-186`）→ 段落级 `heading: HeadingLevel.X`（禁止用"加粗+大字"模拟，`create.md:136-140`）→ run 级 `font: { ascii, eastAsia }` 双字体槽。中文走 `eastAsia`、西文走 `ascii` 是硬规则（`common-rules.md:35-41`）。
   - 编辑场景的样式继承是**另一套**：`Document` 默认 `track_revisions` 关，插新元素时自动补 `w:rsidR/w:rsidRDefault/w:rsidP` 与 `w:author/w:date/w:id`（`document.py:84-90`），替换文本时把原 run 的 `<w:rPr>` 原样搬进 `w:del`/`w:ins`（`routes/edit.md:66-69`）。
5. **不校验 XML schema**：`postcheck.py:5-6` 明说 "this script does not check XML legality"，只查"人眼可见的排版质量"。**所以"合法性"是 1+2 保证的，"观感"是 3.4 之后的 postcheck 保证的**，两者不重叠。

---

## 4. Q3 · 质检闭环：postcheck 逐条 + visual-judge 协议 + 修复轮次

### 4.1 声明的检查项 vs 实际执行的检查项（名实不符，重要）

- `SKILL.md:187` 宣称 **"Automatically checks 14 business rules"**，并列举了"table margins / CJK indent / document cleanliness / report content quality"。
- `postcheck.py` 文件头 docstring（`:11-26`）列了 **15 项**。
- 实际注册执行的只有 **9 项**：`run_all_checks()` `:729-737` 的 7 个函数 + `:753` 的 TOC + `:759` 的图片纵横比。

| # | 实际检查 | 函数与行号 | 判据（关键阈值） | 严重度 |
|---|---|---|---|---|
| 1 | **空白页** | `check_blank_pages` `:103-214` | ①文档末段只有分页符无文字（`:112-125`）②连续空段 ≥5（`:147`）③节末 PageBreak + 下节 `w:type=nextPage` 双分页（`:150-165`）④含 PageBreak 的空段计数（排除节末段，`:167-195`）；"double/trailing/consecutive" 归 error，其余 warning（`:197-212`） | error/warning |
| 2 | **封面溢出** | `check_cover_overflow` `:645-722` | 首节字号 >88 half-pt（44pt，`:674`）；`spacing.before` >5000 twips（`:690`）；封面尾部空段 >2（`:709`） | error |
| 3 | **行距一致性** | `check_line_spacing` `:217-266` | 统计正文段 `w:spacing/@w:line` 众数；偏离占比 >20% 报 warning（`:259`）；跳过 `Heading*`/`Title` 样式（`:229-233`） | warning |
| 4 | **图片超宽** | `check_image_overflow` `:270-309` | `wp:extent/@cx` > 可用页宽 ×1.05（`:287-296`），twips→EMU 用 `×635` | error |
| 5 | **字体回退风险** | `check_font_fallback` `:417-448` | 白名单 20 个安全字体（宋体/黑体/雅黑/仿宋/楷体/TNR/Arial/Calibri…，`:419-428`），集合差集非空即报 | info |
| 6 | **标题层级跳级** | `check_heading_levels` `:452-485` | `Heading(\d+)` 序列相邻差 >1 即报（`:472-477`） | warning |
| 7 | **ShadingType 误用** | `check_shading_type` `:491-508` | 任何 `w:shd/@w:val="solid"` 即报（`:496-500`），WPS 会渲染成黑块 | error |
| 8 | **TOC 质量** | `check_toc` `:512-640` | ①有"目录"标题却无 TOC 域 → `TOC_FIELD_MISSING`（`:574-575`）②有 TOC 域但 0 个 Heading 段 → `TOC_NO_HEADINGS`（`:578-579`）③Heading 样式缺 `outlineLvl` → `TOC_OUTLINE_MISSING`（`:609-614`）④`settings.xml` 缺 `updateFields=true` → `TOC_UPDATE_DISABLED`（`:617-629`）；前三种 error（`:639`）。TOC 域探测三路：`instrText`/`fldSimple`/`sdt` 包裹（`:540-558`） | error/warning |
| 9 | **图片纵横比失真** | `check_image_aspect_ratio` `:312-414` | 读 `word/_rels/document.xml.rels` 建 rId→图片映射（`:320-340`），用 PIL 读原始像素（`:367-378`），与 `wp:extent` 显示比例比对，漂移 >10% 报 warning（`:385`）；**PIL 缺失/异常静默跳过**（`:377-378,393-398`） | warning |

实现细节里两个"诚实"的地方：
- `check_cover_separation` 被**主动删除**并留注释说明原因（`:488`：">15 elements is normal" 误报太多）——不是遗漏，是刻意取舍。
- 声称的"表格边距/cantSplit/CJK 缩进/文档清洁度/报告内容质量"5 项**根本没有实现函数**，只以 prompt 清单形式存在于 `SKILL.md:159-163`（表格跨页）与 `SKILL.md:124-131`（CJK 缩进）里。`SKILL.md:187` 的"14 条"是**营销口径**，不能作为 gaea 需求清单直接引用。

依赖面：`postcheck.py` 顶层 import 只有 `json/re/sys/zipfile/pathlib/xml.etree`（`:29-34`），PIL 是函数内 import 且失败即跳过 —— **除 PIL 外零第三方依赖**，这是它最容易被 gaea 复用的原因（见 §9）。

### 4.2 visual-judge 判定协议

文件：`DP/agents/visual-judge.md`（与 pdf/spreadsheets/presentations 三份 SHA256 完全一致）。

- **定位**：整套交付的"唯一视觉验收关"。要么派它、要么自己看图，**二者只能选一**，禁止先自检再派（`:3`："do not pre-screen the pages before dispatching"）。
- **输入**：只读**已渲染好的 PNG 路径** + 用户原始需求（`:15`）；明确禁止把 pptx/docx/HTML/PDF 当图片读（`:3`）。
- **权限**：`tools: [Read, Bash]`（`:7`），read-only，不写任何文件（`:11`）。
- **判定维度**（每页都查两类，`:19-22`）：
  1. 视觉资产：图/表/图标切题、正确、未变形（无拉伸/压扁/破坏性裁剪）、清晰未裁切、无假图；
  2. 版式构图：模块重叠、内容堆叠或隐藏、越界溢出、过挤、明显失衡。
- **分格式专属关注点**（`:26-29`）：pptx 看"投影距离一眼落地"（文字压卡片、单卡半空、跨页一致性）；docx 看"阅读距离"（近空白页、标题孤行、跨页断框、**TOC 条目无页码**、图渲成空白、页眉页脚页码跨节连续性）；xlsx 看 `####` 截断/错误值/打印切片；pdf 看多栏断流/孤立标题。
- **输出协议**：**每页一行 JSON，按页序，pass 页也要输出，无散文**（`:41-48`）：
  ```
  {"page": 4, "verdict": "fail", "issues": [{"category": "Design", "problem": "chart overlaps the caption text below it", "evidence": "bar chart's bottom edge extends into the caption line on page 4"}]}
  ```
  `category ∈ {Spec, Visual, Design, Unverified}`；**任一条不满足即 fail**；无法确认必须标 `Unverified`（不是猜 pass）。
- **证据纪律**：一处根因多处症状只报一条、归到主导类；图内 = Visual，元素间 = Design，违背 brief = Spec（且必须引用 brief 原文，没给 brief 就不许发明约束）；禁止编造像素值（`:37-39`）。
- **模型**：`account:bigmodel-individual-coding-plan/GLM-5.3-Flash`，`thoughtLevel: max`（`:5-6`）——即"视觉验收"是**另一个模型 + 最高思考档**的独立 pass，不是主 agent 自评。
- **阈值**：**没有分数阈值**。判定是逐页二元的 pass/fail + 结构化 issue，不是打分制（全文无 score/threshold 字段）。

### 4.3 修复循环最多几轮？——**读不到**

- `visual-judge.md:3` 把轮次写死在别处：「Dispatch grouping, what to pass in, and **the repair loop follow the delivery protocol's visual gate in your system prompt**」。
- 我检索了整个 `clones/zcode` 仓库（`grep "delivery protocol|visual gate"`），命中只有 4 份 visual-judge.md 自身的 description，**仓库内没有该 delivery protocol 的定义**（属 ZCode 系统提示词，不在源码树内）。
- 仓库内能找到的只有两条弱约束：
  - `PPTX/SKILL.md:158`："**Visual QA once only**"（pptx 明确只做一遍视觉 QA）；
  - `DOCX/SKILL.md:189`："After generating any document, MUST run postcheck.py and **fix all ❌ errors**"（机检必须清零，但没说最多几轮）。
- 结论：**「最多 N 轮」这个数字在本仓库不存在**。可确证的闭环结构是「机检清零（无限轮，直到 0 error）→ 视觉验收一遍（once only）」，而不是固定轮次。给 gaea 复刻时应自行定死（建议 3 轮上限 + 每轮只修 issue 列表内的项），并明确写进 prompt —— 这正是 ZCode 的做法（把轮次外置到系统提示词），复刻时必须显式化。

---

## 5. Q4 · 领域知识资产：10 条最硬的，可直接变 gaea prompt / 校验规则

按「能直接落成规则或机检」排序，逐条摘录要点。

### ① GB/T 9704 公文版式全套数值（可直接机检）
`DOCX/scenes/official-doc.md:265-330` + `common-rules.md:59-71`
- 页边距：上 3.7cm=**2098**、下 3.5cm=**1984**、左 2.8cm=**1588**、右 2.6cm=**1474**（twips）。
- 字体：红头 方正小标宋 26pt(size 52)；标题 小标宋/宋体加粗 22pt(44)；**正文 仿宋 三号 16pt(32)**；一级标题黑体、二级楷体、三级仿宋加粗。
- **行距固定 28pt = `line: 560`**，首行缩进 **640** twips。
- 禁：封面、TOC、页眉、除红头外的任何颜色、图表（`:359-368`）。
- 红头/文号/版记**只在显式要求时启用**（`:44-51` `needsRedHeader()` 正则）。

### ② 中文号数字号 ↔ half-points 换算表（一个常量表顶一堆换算 bug）
`common-rules.md:73-89`：初号 42pt/84、小初 36/72、一号 26/52、小一 24/48、二号 22/44、小二 18/36、**三号 16/32**、小三 15/30、四号 14/28、小四 12/24、**五号 10.5/21**、小五 9/18、六号 7.5/15。
配合 `SKILL.md:107-116` 单位表：1cm=567tw、1in=1440tw、1pt=20 half-pt、A4=11906×16838 tw。

### ③ WPS / Word 双引擎兼容黑名单（12 条禁 + 10 条必检）
`common-rules.md:169-230`。最硬的 6 条：
1. **`ShadingType.SOLID` 在 WPS 渲染成纯黑** → 永远用 `CLEAR`（`:184`）；
2. **exact 高行里用 `verticalAlign: center/bottom`** WPS 忽略 → 用 `verticalAlign:"top"` + `spacing.before`（`:179`）；
3. **表格列宽用 DXA 会触发 WPS tblGrid 全 100 的 bug** → 一律 `WidthType.PERCENTAGE`（`:216-230`）；
4. **用制表位对齐** 在两引擎宽度不一致 → 改用无框表格（`:187`）；
5. **用文字字符画横线**（`───`）宽度不可控且会截断 → 用段落边框 `border.bottom`（`:177`）；
6. **封面外层表忘了 `allNoBorders`** → Word 把边框算进高度，溢出几 twip 就在第 2 页冒出**空白页**（`:178`，含机理说明，非常值钱）。

### ④ 封面 16838 高度预算 + 只许用 R1–R7 配方（把"排版玄学"变成算术）
`common-rules.md:202-216`、`design-system.md:926-965`
- 封面必须放在**独立 section** 且 section 边距全 0（`:209`）；
- 外层包装表行高 `{ value: 16838, rule: "exact" }`（`:210`）；
- 内容总高 ≤ **15638 twips**（留 1200 安全边距，`SKILL.md:105`）；
- 封面标题必须过 `calcTitleLayout()`，禁止硬编码 >40pt（`SKILL.md:168`）；间距必须过 `calcCoverSpacing()`（`:169`）。
- ⚠️ 实测缺口：`design-system.md` 只有 R1–R5 的 `buildCoverRX()` 实现（`:985,1102,1213,1328,1554`），**R6/R7 只出现在调色板与路由里，没有 builder 代码** —— 直接搬会踩空（详见 §8 风险）。

### ⑤ TOC 三步法 + 三段式页码 + `\* arabic` 陷阱（真实高频 bug）
`toc.md:5-15,126-227` + `fix_footer_fields.py:1-25`
- 三步：docx-js 出**空** TOC 域 → `add_toc_placeholders.py --auto` 灌可见占位 → 用户在 Word 里"更新域"。
- "目录"标题**绝不能**用 Heading 样式（否则 TOC 索引自己，`toc.md:30,247`）。
- `TableOfContents` 之后**必须**紧跟含 PageBreak 的段落（`toc.md:56-57`，称"#1 TOC formatting failure"）。
- 三段式页码：封面无页码 → 前置罗马 `start=1` → 正文阿拉伯 `start=1`（`toc.md:130-163`）；`pageNumbers` **必须嵌在 `page:{}` 内**（`:165-182`）。
- **`\* decimal` 是错的**：`decimal` 是 docx-js 的枚举值，不是 Word 域开关，会导致页码渲染成 "1decimal"；阿拉伯数字正确开关是 `\* arabic`（`toc.md:192`、`SKILL.md:155`、`fix_footer_fields.py:50-53` 专门写正则修这个 bug）。

### ⑥ 合同：5 类型模板 + 交易闭合 10 问 + 纯黑排版
`contract.md`
- 类型路由 `selectContractType()` 正则（`:20-28`）；5 套条款骨架（买卖 18 段 `:44-62`、转让 12 段、NDA 12 段、框架 13 段、单方条款 11 段）。
- **交易闭合 10 问**（`:162-177`，最值钱）：有履行期限→必须写逾期后果；有付款节点→必须写条件/方式/票据；有交付→必须写验收标准与方法、异议期；有解除权→条件/通知/生效日/解除后结算；有违约责任→必须对应前文主要义务；涉 IP/数据/商业秘密→单独写归属/许可范围/使用限制；有保密→范围/例外/期限/违约后果；有不可抗力→通知义务/减损义务/后续协商；有通知条款→地址/联系人/邮箱；单方免责条款→加人工复核批注。
- 措辞纪律：`shall / has the right to / shall not / may` 四档对应强制/授权/禁止/裁量（`:148-152`）；禁"相关方/必要时"这类模糊主语（`:156`）。
- 版式：**无封面、无 TOC（<20 条）、无装饰、全篇纯黑 `"000000"`、行距 1.5x（line: 360）**（`:422-429`）；甲乙双方信息**必须无框表格对齐**（`:305-362`）；金额必须大写+数字双写（`:412-418`）。

### ⑦ 公文正文层次序数 + 中文日期 + 六角括号（可正则机检）
`official-doc.md:198-250`
- 层次：`一、` → `（一）` → `1.` → `（1）`，禁跳级、禁 Markdown 符号（`:217-221`）。
- 日期必须中文数字且带 **〇**（"二〇二六年三月三十一日"，`:246-247`）——注意不是"零"。
- 文号用**六角括号〔〕**，不是方括号（`:248-249`）。
- 标题四段式「发文机关 + 关于 + 事项 + 文种」（`:127-132`）；结束语必须与文种匹配（通知"特此通知"/函"请复函"/批复"此复"，`:184-188`）。

### ⑧ 学术论文：三线表 + 图表题注位置 + GB/T 7714 + 引文上标
`academic.md`
- **表题在上、图题在下**，五号宋体居中；图表公式**必须正文引用**（`:567-610`）。
- 三线表硬实现：外框 top/bottom size 4，表头下 size 2，其余全 NONE，表头行 `tableHeader:true cantSplit:true`（`:665-692`）。
- 编号按章：图 3-1 / 表 2-1 / 式 (3-1)（`:557-565`）；公式居中 + 编号右对齐靠**制表位**（`:591-603`）。
- 参考文献 GB/T 7714 四类格式（`[J]/[M]/[D]/[EB/OL]`，`:630-636`），正文 `[n]` 上标（`:621`），同一文献同号（`:626`）；篇数按学位层级给区间（`:652-659`）。
- 真实性红线：不许编造统计量/显著性/访谈对象/文献（`:698-704`）。

### ⑨ 试卷：答题空间与分值映射 + 作文格 1.25 倍 + 分隔禁装饰
`exam.md`
- 答题行数↔分值：2–4 分→3–4 行；5–8→6–8；10–12→8–10；14–20→10–14（`:352-359`）。
- 作文格：格数 = 要求字数 × 1.25，20 列，行数向上取整（`:387-404`）；英文作文**必须有横线**，字数↔行数表（`:426-446`）。
- 题与题之间**只用间距**（`spacing.before:200`），禁灰线/色块/符号分隔（`:344-350`）。
- 图纸规格：练习 A4 纵、正式考 A3 横双栏、答题卡 A4 纵（`:16-20`）；双栏 `column:{count:2, space:720}`（`:50`）。

### ⑩ 图表/公式的"印刷级"约束
`design-system.md:1698-1736` + `chart-templates.md:317-386` + `math-formulas.md:231-276`
- 图表色板：默认**莫兰迪低饱和** 8 色，单图**最多 5 色**，超出用同色深浅；饼/条相邻色相间隔 ≥60°（`:1706-1720`）。
- **纵横比铁律**：只给宽度、用真实像素算高度（`image-size` / `PIL`），否则饼图变椭圆、雷达变菱形（`:1724-1731`、`chart-templates.md:319-351`）。
- 200 DPI、`figsize=(10,6)`、饼/雷达必须 `(8,8)`、>6 个 x 标签旋转 45°、图例 >4 系列移出绘图区（`:1722-1736`、`:376-386`）。
- CJK 字体必须在 matplotlib 里显式绑定，否则全是方块（`chart-templates.md:380`）。
- 公式回退分界线：嵌套 >2 层/矩阵/复杂积分/分段函数 → matplotlib PNG；其余走 docx-js Math 对象（`math-formulas.md:271-276`）。

---

## 6. Q5 · 工程洁癖：env_setup、字体策略、失败降级

### 6.1 Python 环境：**没有 venv，全系统级安装 + marker 文件**
- 文档层面只要求 `python3` + `pip install defusedxml`（`env_setup/setup.md:44-51`）；Windows 侧 `python -m pip install defusedxml`（`:81`）。
- 没有 `python -m venv`、没有 requirements.txt / lock、没有版本钉住。隔离靠**两个 marker 文件**：字体装完写 `.office-skill-fonts-installed`（`setup.md:98`、`setup_windows.ps1:195-197,233`），下次看到 marker 就跳过下载。
- 依赖清单极薄（`SKILL.md:210-218`）：pandoc / docx(npm) / LibreOffice / Poppler / defusedxml / python-docx。**没有 pandas、没有 lxml 显式声明**（`document.py:37-38` 用的是 `defusedxml.minidom`）。

### 6.2 字体：CDN 单源 + 扁平落盘 + 平台目录差异 + Windows 要写注册表
- **策略是 local-font-first**：`SKILL.md:18-20` 明确"先看用户本机已有字体，优先用本机；内置/下载仅作 fallback；**未经用户确认不得装字体**"。
- 清单：`env_setup/font_list.txt` 实测 **80 行**（`setup.md:97` 写 78，文档漂移 2 条），路径形如 `/usr/share/fonts/truetype/chinese/NotoSansSC[wght].ttf` —— 这是**CDN 相对路径**（镜像 Linux 安装路径），不是安装目标路径。
- CDN 基址 `https://z-cdn.chatglm.cn/office-skill/fonts/`（`setup.md:96`），按目录分 12 组（lxgw-wenkai 6 / noto-serif-sc 9 / chinese 14 / dejavu 8 / english 12 / freefont 12 / liberation 12 …，`:103-118`）。
- 安装目标按平台：mac `~/Library/Fonts`、Linux `~/.local/share/fonts` + `fc-cache -f`、Windows `%LOCALAPPDATA%\Microsoft\Windows\Fonts`（**per-user，免管理员**，`:120-126`）。
- Windows 额外一步：**必须写注册表** `HKCU:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`（`setup_windows.ps1:222-224`），否则部分应用看不到字体。
- 文件名含 `[` `]` 需 URL 编码（`setup.md:99,143`）。

### 6.3 Windows 支持：真做了，不是口头
- 独立 PowerShell 脚本 `setup_windows.ps1`（9.7KB），`param()` + `$ErrorActionPreference="Continue"`（`:10,14`）；联网探测 pypi（`:47`）。
- `env_check.sh` 是 bash-only（`:20-24` 只判 Darwin/Linux），Windows 侧没有对称的 `env_check.ps1` —— **这是移植时的空缺**（`setup.md:40` 让 Windows 直接跑 `setup_windows.ps1`，无轻量 check）。
- LibreOffice 在 Windows 上要 `setx` 追加 PATH，且明确"`setx` 只对新 shell 生效，当前会话用全路径"（`setup.md:319-326`）。

### 6.4 失败降级：这里最能看出产品性格
- **一处硬：LibreOffice 不许替代。** `SKILL.md:220-267` 用整节写 "HARD REQUIREMENT"，明确禁止：用本机 Word/WPS/Pages 顶替、以"包太大"为由跳过、问用户"要不要用 Word 省时间"、静默跳过 PDF/视觉检查（`:227-234`）。要求顺序执行"解释必要性→告知体积→从清华 TUNA 镜像装→注册 PATH→复验 `soffice --version`"（`:236-264`），且给出金句"**no install attempt = no substitute program**"（`:266`）。同一段话在 `routes/edit.md:30-35,89-91`、`env_check.sh:10-13` 重复三遍。
- **另一处软：** `env_check.sh` 把 LibreOffice 归为 `required_on_demand`，缺失**不让 check 失败**（`:10-13,37`）；`postcheck.py` 的 PIL 缺失、图片纵横比检查失败一律 `info` 且 pass（`:393-398`）；`add_toc_placeholders.py` 对非 TOC 结构错误只打印不改退出码。
- **具体降级路径**：TOC 空 → 灌占位条目让用户先看到标题（`toc.md:124`）；复杂公式 → PNG（`math-formulas.md:271`）；WPS 字体缺 → `STXiaoBiaoSong → FZXiaoBiaoSong-S13 → SimSun Bold` 三级回退（`official-doc.md:292-301`）；字体缺失 → 白名单告警而不阻断（`postcheck.py:417-448`）。

### 6.5 几处"洁癖的反面"（复刻时要避开）
- `add_toc_placeholders.py:725` 写的是 `elif args.auto or True:` —— 恒真分支，`--auto` 形同虚设，永远是自动模式。
- `SKILL.md:187` 的"14 条检查"与 `postcheck.py` 实际 9 条不符（§4.1）。
- `setup.md:97` 的"78 fonts"与实际 80 条不符。
- `visual-judge.md` 在 4 个插件里复制了 4 份（SHA256 相同）——**同一份规则四处维护**，改一处必漏三处；gaea 若复刻应做成单源引用。

---

## 7. gaea 侧现状（对照基线，全部实测）

| 能力 | gaea 现状 | 证据 |
|---|---|---|
| docx 真编辑 | **修订制**：`w:del`+`w:ins` 写入，字节级手术，其余字节不动 | `internal/office/docxedit/docxedit.go:1-8`、`ApplyTrackedReplace` `:27`、`AcceptChanges` `:53`、`RejectChanges` `:58`、`flattenChanges` `:63` |
| docx 编辑绑定 | 落盘前基线快照 + Journal 证据卡（`docx_apply`/`docx_accept`），不可逆操作才留证据 | `internal/app/gaea_docx_edit.go:50-78,86-115,121-134` |
| pptx 真编辑 | run 级文本替换，`a:rPr` 原字节继承，**无修订制**（pptx 无等价物） | `internal/office/pptxedit/pptxedit.go:4-11,61`、绑定 `internal/app/gaea_pptx_edit.go:19-58` |
| pptx 段落定位 | `LocateText` 纯 Go，零 python | `internal/office/pptxedit/pptxedit.go:32`；绑定注释明确"纯 Go 零 python" `gaea_pptx_edit.go:63-64` |
| 渲染 | soffice→PDF（独立 profile 防锁冲突）+ poppler 逐页 PNG | `internal/app/gaea_pdf.go:19-22,149-205`、`internal/app/gaea_pptx.go:256-271` |
| 视觉复核 | **像素 diff + 页数联合判定**，阈值 0.02 pass / 0.20 warn | `internal/app/gaea_verify.go:60-68,75-109` |
| 降级 | soffice/poppler 缺失 → `warn: 视觉渲染降级（…仅结构复核）`，不 fail | `internal/app/gaea_verify.go:79-82,87` |
| 办公纪律 prompt | 已有 `office-edit` 内置技能：先读后写 / 写后回读 / 宁拒不误改 / 逐 run 编辑 / soffice 取证 / `[FORMAT_*]` 错误路由 | `internal/gaea/skill/builtins.go:44-104` |
| docx 从零生成 | 弱：gooxml 只做 Title/Heading1/段落，**不设复杂排版/图片**（源码自述） | `internal/export/docx.go:10-14,17-66` |

**关键差异（决定搬运策略）**：

1. gaea 的 docx 是**「改用户已有文档」**范式；ZCode 的 docx 是**「从零造文档」**范式（`routes/create.md` 占 207 行，是主路径）。所以 ZCode 那套 80KB 封面/调色板/配方，对 gaea 主路径**几乎无用**，只在"生成新 docx"这一支线上有价值。
2. gaea 的视觉复核是**相对 diff**（before/after 像不像），ZCode 的 visual-judge 是**绝对验收**（这一页能不能交付）。二者互补，不重叠。
3. gaea 的写入路径有**唯一收口**（docxedit/pptxedit + 证据链 + 回滚快照）；ZCode 是"agent 写脚本自己落盘"。搬 ZCode 的脚本进 gaea，等于**开第二条写入路径**，这是最大的架构风险。
4. 任务描述里把 `internal/gaea/config/render_preserve.go` 归到"办公/渲染保留"是**误归**：该文件是 **TOML 配置**的保留式渲染（保住用户手写但渲染器不认识的配置段/键，`render_preserve.go:9-28`），与 docx 样式继承无关。**不要**拿它当"样式继承"的参照物。

---

## 8. 搬运判定矩阵

成本口径：**S** ≤ 半天（纯文本搬运/小改）；**M** 1–3 天（需要新代码或新校验通道）；**L** ≥ 1 周（跨运行时/跨范式）。

### 8.1 可以原样搬（纯 prompt / 规则 / 清单，零执行依赖）

| # | 资产 | 落点 | 成本 | 风险 |
|---|---|---|---|---|
| A1 | **中文号字表 + 单位换算表 + 1.3x 行距 + 占位符【】规范**（`common-rules.md:73-110`、`SKILL.md:107-116`） | `office-edit` 技能追加一节常量表 | **S** | 低。仅与 gaea 现有"保留原格式"哲学有轻微张力：这些是**新建**文档才用的默认值，须标注适用面（新建 vs 编辑） |
| A2 | **WPS/Word 兼容黑名单 12 条 + 10 条必检**（`common-rules.md:169-230`） | 新增校验规则集 + prompt checklist | **S** | 中。半数条目（allNoBorders/16838 wrapper/DXA 列宽）**只对 docx-js 生成物成立**；对用户已有文档无效，直接照搬会产生假告警。需按"生成路径/编辑路径"分档 |
| A3 | **visual-judge 验收协议**（`DP/agents/visual-judge.md` 全文） | gaea Verifier 的"绝对质量"新通道 prompt（渲染 gaea 已有） | **S**（协议）+ **S**（接渲染，链路已通） | 中。① 需一个"看图"能力（gaea 现无 VLM 通道，需确认模型路由）；② 与现有像素 diff 必须**明确分工**，否则同一次改动被两套标准判两次 |
| A4 | **合同场景知识包**（`scenes/contract.md` 5 模板 + 闭合 10 问 + 纯黑/无封面/1.5x） | 新建 `office-contract` 场景 prompt | **S** | 低（纯知识）。风险在**法律责任**：ZCode 自己写了"不给正式法律意见"（`:184-188`），gaea 搬时必须保留这条免责边界 |
| A5 | **公文 GB/T 9704 数值包**（`official-doc.md:265-330` + 层次序数/日期〇/〔〕`:198-250`） | 场景 prompt + 机检规则（页边距/行距/字体名可正则断言） | **S**（prompt）/**M**（机检） | 低。中国办公刚需，且数值全部可机检，投入产出比最高的一档 |
| A6 | **TOC 三步法 + `\* arabic` 陷阱 + 三段式页码**（`toc.md` 全文 + `SKILL.md:147-157`） | `office-edit` 技能补 TOC 专章 | **S** | 低-中。gaea 当前完全不碰 TOC（docxedit 只改正文 run），搬进来是**纯增量**；但"用户打开 Word 更新域"这一步必须交给用户，不能承诺自动页码 |
| A7 | **图表/公式印刷级约束**（`design-system.md:1698-1736`、`chart-templates.md:317-386`、`math-formulas.md:231-276`） | 提示词 + 可选机检（纵横比漂移 >10% 可查） | **S** | 低 |
| A8 | **学术/简历/试卷/报告四场景知识**（`academic.md`/`resume.md`/`exam.md`/`report.md`） | 场景 prompt 库 | **S/M** | 低。纯知识；`exam.md` 的作文格/答题行数需要表格能力，gaea 生成路径弱（见 B1） |
| A9 | **「单源 + 引用不复述」的组织纪律**（`toc.md:3`、`create.md:44`） | gaea skill 体系编写规范 | **S** | 无。这是纯方法论收益 |

### 8.2 必须改写（有执行依赖 / 跨运行时 / 跨范式）

| # | 资产 | 为什么必须改 | 改写方案 | 成本 | 风险 |
|---|---|---|---|---|---|
| B1 | **docx 生成管线**（docx-js + `bun`/`node`，`create.md:154-198`） | gaea 是单 Go 二进制；`internal/export/docx.go` 用 gooxml 只能出 Title/Heading/段落，**无表格/图片/分节/TOC** | 三选一：① 允许 `bash + node -e`（gaea 已有 bash 工具与 node 约定，`builtins.go:248,255`）；② Go 侧补 gooxml 能力（工作量大且上游弱）；③ 改成"模板+占位符填充"（放弃自由排版） | **M**（①）/**L**（②） | **高**：① 引入 Node 运行时依赖（体积/离线安装），且与"Go 侧真编辑"形成**两条写盘路径**，必须收口到同一证据链；② 质量天花板低 |
| B2 | **`scripts/document.py`（XML 编辑 + 修订 + 批注）** | gaea 已有 `docxedit` 覆盖修订（apply/accept/reject）；python 库的价值在 **批注四件套**（comments/commentsExtended/commentsIds/commentsExtensible，`document.py:698-702`、模板 `scripts/templates/*.xml`）与**按行号/属性精确定位**（`:717-748`） | 只搬两件：① 批注写入（Go 补 `comments*.xml` part + rels + content-types，~400 行）；② 定位原语（`get_node(tag,attrs,line_number)`）可作 docxedit 的定位增强 | **M** | **中-高**：两套修订实现并存 → `w:id`/`w:author` 冲突、rsid 冲突；批注与修订交叉时（批注锚点落在 `w:ins` 内）语义易错。必须**只保留 gaea 一套修订**，python 侧不再产修订 |
| B3 | **`postcheck.py` 9 条机检** | 是 Python；但**零第三方依赖**（`postcheck.py:29-34`），且判据全在 OOXML 字节层 | ① 最省：Go 侧 `exec python postcheck.py --json` 包一层，当 Verifier 通道 A 的补充；② 干净：用 Go 重写 9 条（zip + encoding/xml，约 600–800 行）——**推荐②**，因为①会把 Python 运行时塞进桌面端分发 | **M** | **中**：重写要保持阈值一致（换行距 20%、图宽 105%、纵横比 10%、封面 44pt/5000tw/2 空段），否则两边判定不一致；`check_image_aspect_ratio` 依赖图像解码，Go 需 `image/png`+`image/jpeg`（标准库够用） |
| B4 | **`add_toc_placeholders.py` + `fix_footer_fields.py`** | Python + 正则手术；且它们会**改写 document.xml/footer 字节** | Go 移植（zip 读写已有 `ooxml.ReadZip/WriteAtomic`，`pptxedit.go:114-118` 说明该层已收口）；正则改 token 级解析 | **M** | **中-高**：① 与修订制冲突——在后处理阶段改 `document.xml`，若文档已含 `w:ins/w:del`，字符串级替换可能落进修订标记内部；② `add_toc_placeholders` 有恒真 bug（`:725`）；③ 只能对**gaea 自己生成的、未含修订的**文件跑，须加守卫 |
| B5 | **pptx 的 python-pptx 编辑路径**（`PPTX/SKILL.md:561-663`，含 budget 规则 `:613-626`） | gaea 已有 Go `pptxedit`（run 级替换 + rPr 继承） | **只搬规则不搬代码**：`budget = orig_len × 1.1`、`role = label/body` 判定（`h<1.5cm 或 orig_len≤8 或 font≥20pt`）、"先扩框再缩字"优先级（`:630-652`）、`\x0b` 软换行归一（`:658`）、删页从高索引起（`:662`） | **S**（规则） | 低。这些是 gaea `pptxedit` 可直接消费的约束；但 gaea 现在的 `pptxedit` 没做溢出判定，加上后属行为变更，需测试 |
| B6 | **字体 CDN + env_setup 全套** | 会**改用户机器**（装字体、写注册表、setx PATH） | 不建议自动执行；改为"检测 + 提示 + 用户手动装"（与 ZCode 自己的 local-font-first 一致，`SKILL.md:18-20`） | **M** | **高**：桌面应用静默写 `HKCU` 字体注册表属越界动作；且 80 个字体（含 12 个 3–10MB 的 CJK）显著膨胀安装体积 |

### 8.3 不该搬

| # | 资产 | 不搬理由（硬证据） |
|---|---|---|
| C1 | **封面配方 R1–R7 / design-system.md 80KB / decorations.md 20KB** | ① 与 gaea 范式冲突：gaea 是"改用户文档"，不新建封面；② **R6/R7 只有路由和调色板，没有 `buildCoverRX()` 实现**（`design-system.md` 仅 `:985,1102,1213,1328,1554` 五个 builder）；③ 依赖 docx-js 的 `Table/Paragraph/TextRun` 对象模型，Go 侧要重写全部 5 个 builder ≈ L 成本换"造封面"这一个低频场景；④ `decorations.md` 537 行是纯装饰几何，与"公文/合同"的克制审美相悖 |
| C2 | **LibreOffice「HARD REQUIREMENT」安装话术 + 强制装机流程**（`SKILL.md:220-267`、`edit.md:30-35`、`env_check.sh:10-13`） | ① 与 gaea 现有降级哲学**直接冲突**：gaea 明确"soffice 不可用 → warn 降级、仅结构复核"（`gaea_verify.go:79-82`）；② 该话术是给"云端 agent 有 shell 权限"的场景写的，桌面应用不能命令用户装几百 MB 软件或代问"要不要用 Word"；③ gaea 已有 `findSoffice()` 多平台探测（`gaea_pdf.go:185-205`），比这套话术更工程化 |
| C3 | **`::zcode-file-citation{}` 引用协议**（`SKILL.md:269-315`）+ **plugin.json / MCP server / UI panel** | ① 引用语法是 ZCode 专有 UI 渲染器（`artifact_kind`/`page_number`/`slide_id`），gaea 无对应渲染，搬了只会输出字面文本；② `plugin.json` 的 `mcpServers` 指向 `${ZCODE_PLUGIN_ROOT}/dist/server.mjs`（`DP/.zcode-plugin/plugin.json:16-25`），**该 dist 产物不在源码树内**（实测 `documents-plugin/` 下无 `dist/`），无法搬；③ `ui.surfaces` 依赖 `ui://documents/panel.html` + ZCode Desktop workspace 协议，gaea 有自有预览体系 |
| C4 | **pptx 55KB 的「AI 生成感」避坑清单 + 全套设计规范**（`PPTX/SKILL.md:9-183`） | 属于"从零造 deck"范式；gaea 的 pptx 只做 run 级文本替换，**不决定版式/配色/字体**（`pptxedit.go:4-11` 明确"母版/主题/动画零扰动"）。搬进来会诱导 agent 去改用户模板，违背 gaea 的保留式编辑 |
| C5 | **`env_setup/setup_windows.ps1` 的注册表写字体** | 桌面应用写 `HKCU\...\Fonts` 静默改用户系统字体表（`setup_windows.ps1:222-224`），属高副作用低收益动作；且 marker 机制（`:195-197`）在 gaea 的 `.gaea/` 缓存体系里已有更合适的落点 |

---

## 9. 最值得搬的 6 条（含成本）+ 落地顺序建议

| 序 | 条目 | 成本 | 为什么排这个位置 |
|---|---|---|---|
| 1 | **visual-judge 验收协议**（A3） | **S** | 补 gaea Verifier 缺的"绝对质量"维度；渲染链路（soffice+pdftoppm）已通，只缺一个 prompt + 一次派发。**零代码风险**，收益最大 |
| 2 | **postcheck 9 条机检用 Go 重写**（B3） | **M** | 把"排版质量"从 prompt 软约束变成机检硬约束，正好接进 `gaea_verify.go` 通道 A。判据全在字节层，无第三方依赖 |
| 3 | **公文 GB/T 9704 数值包 + 中文号字表**（A5+A1） | **S**（prompt）/ **M**（机检） | 中国办公最高频、最可机检的一类；页边距/行距/字体/序数/日期/括号全部可 grep 断言 |
| 4 | **WPS/Word 兼容黑名单**（A2） | **S** | 直接降低"用户打开是黑格/空白页/页码乱码"这类投诉；但**必须分"生成路径/编辑路径"**再写，否则假告警 |
| 5 | **TOC 三步法 + `\* arabic` 坑**（A6） | **S** | gaea 现在完全不碰 TOC，是纯增量能力；且这两个坑是真实用户事故（页码显示 "1decimal"） |
| 6 | **合同场景知识包**（A4） | **S** | 纯知识、复用率高（NDA/框架/服务条款同源）；注意保留"不给正式法律意见"的免责边界 |

可选第 7：`pptx budget 规则`（B5，S）——gaea `pptxedit` 加上 `orig_len × 1.1` 上限即可显著减少溢出投诉。

---

## 10. 证据强度与读不到清单（诚实边界）

**完整读过（全文）**：两份 `plugin.json`、`agents/visual-judge.md`、`DOCX/SKILL.md`、`routes/create.md`、`routes/edit.md`、`scenes/contract.md`、`scenes/official-doc.md`、`references/toc.md`、`scripts/postcheck.py`（808 行）、`env_setup/setup.md`、`env_setup/env_check.sh`、`fix_footer_fields.py`（头 120 行 + 尾 36 行）、`add_toc_placeholders.py`（头 120 行 + 尾 59 行）、`document.py`（头 90 行 + `Document` 类 649-828）、`skills/docx-viewer/SKILL.md`、`PPTX/SKILL.md` 的 1-190 / 424-573 / 552-691 段。

**抽样/仅取结构**：`references/common-rules.md`（1-115、160-299，其余按标题抽样）、`design-system.md`（1-60、1690-1797 + 全文标题与关键正则，**中间 1,600 行未逐行读**）、`scenes/academic.md`（555-719 + 标题）、`scenes/exam.md`（12-141、330-449 + 标题）、`scenes/resume.md`（1-150）、`scenes/report.md`（1-120）、`math-formulas.md`（217-276）、`chart-templates.md`（310-386）、`faq.md`/`decorations.md`/`ooxml.md`/`docx-js-core.md`/`docx-js-advanced.md`（**仅标题级**）、`routes/format.md`/`read.md`/`comment.md`（**未读**）、`scripts/utilities.py`（仅 grep 函数名）、`PPTX/SKILL.md` 190-424 / 692-783（**未读**）。

**明确读不到 / 不存在**：
1. **修复循环轮次上限**：定义在 ZCode 系统提示词的 "delivery protocol's visual gate"（`visual-judge.md:3` 指向），**本仓库内无该文本**；仓库内 grep `delivery protocol|visual gate` 只命中 4 份 visual-judge.md 自身。
2. **`dist/server.mjs`（MCP server 实现）**：`plugin.json` 引用，源码树内无此产物 → MCP 工具面的真实能力**无法评估**。
3. **R6/R7 封面 builder 实现**：`design-system.md` 只有 R1–R5 的 `buildCoverRX()`。
4. **design-system.md 的 60–1690 行**（约 1,630 行，含 7 套封面配方全文与 14 个调色板定义）：只读了标题与关键正则命中行，**配方内部细节未逐行核对**。
5. **`scenes/copywriting.md`**（139 行）、`routes/format.md`/`read.md`/`comment.md`：未读。
6. `PPTX/SKILL.md` 的 190–424（pptxgenjs 文本/形状/图片/表格 API 细节）与 692–783（LibreOffice 硬要求后半 + 引用协议）：未读。

---

## 11. 一句话给决策者

**搬知识、不搬运行时**：visual-judge 协议 + postcheck 9 条 + 公文/合同/兼容性三套规则，合计约 **3–4 人日（S+S+M+M）** 就能给 gaea 补上"办公交付质量"这一整层；而 docx-js 生成管线与 Python 后处理要花 **L 级成本**去换取 gaea 主路径（改用户文档）用不上的"从零造文档"能力 —— 除非 gaea 明确要把"新建 docx 交付"做成主功能，否则这一档应当继续压在 `bash + node` 的临时路径上，不进 Go 主干。
