# s8 · ZCode「表格 / PDF」插件拆解与 gaea 移植判定

> 调研对象：`clones/zcode/apps/zcode-cli/packages/spreadsheets-plugin`（xlsx）与 `.../pdf-plugin`（pdf），只读。
> 对照目标：`C:\AI\wubigrok`（gaea，Go + Wails）。
> 方法：全量文件清单 + 结构化 grep 定位 + 局部精读；所有结论带 `文件:行号`。
> 路径别名（下文 `$X` / `$P` 均指此）：
> - `$X` = `clones/zcode/apps/zcode-cli/packages/spreadsheets-plugin/skills/xlsx`
> - `$P` = `clones/zcode/apps/zcode-cli/packages/pdf-plugin/skills/pdf`

---

## 0. 一页结论

| 维度 | xlsx 插件 | pdf 插件 |
|---|---|---|
| 本质 | **openpyxl + pandas + LibreOffice 重算** 的「写码式」工作台，价值在规范 + 校验命令 | **四条独立排版管线**（ReportLab / Tectonic / Playwright / LibreOffice）+ 一套跨管线排版规范 |
| 最强资产 | `templates/base.py`（设计令牌单一真源）、`quality/pipeline.md`（三角色质检流程）、`xlsx.py audit/scan/validate` 规则 | `typesetting/`（overflow / pagination / fill-engine / cover / geometry）、`poster_validate.py check-html/check-tex`、`design_engine.py` 色板与 WCAG 审计 |
| 代码可移植性 | 中：规则是纯正则/阈值，可 Go 化；生成侧重（openpyxl 生态） | 低：渲染全靠 Python/Node 三方库（ReportLab、Playwright、Tectonic），Go 无等价物 |
| 对 gaea 的定位 | **规则层原样搬、重建层改写** | **规范层原样搬（文本）、执行层不要搬（换 Go 原生渲染器）** |

三条最关键判断：

1. **能原样搬的是"知识"，不是"引擎"。** `typesetting/*.md`（~90KB 规则）、`briefs` 的铁律节选、`quality/pipeline.md` 的陷阱表、`palettes.py` 的 12 套色板数据，都是与实现解耦的纯文本/纯数据，拷进 gaea 的 prompt 资产与校验器即可（成本 S）。
2. **要改写的是"校验器"和"色板/测量数学"。** `audit/scan/validate`、`check-html`、`check-tex`、`palette-cascade + WCAG audit` 都是**纯规则 + 纯算术**，Go 重写成本 M 且风险低；`pdf_qa.py`（PyMuPDF）成本 L 且 Go 侧无等价物，建议先用 Python 侧车。
3. **不该搬的是宿主耦合与渲染栈。** 插件清单/MCP/侧栏 UI、`visual-judge` agent、`env_setup`、VBA 注入、`xlsx.py pivot`（名为 PivotTable 实为 Python 聚合，且与自家 `pipeline.md:104` 自相矛盾）都不该进 gaea。

**许可提示**：两个技能目录的 `SKILL.md` front-matter 写 `license: Proprietary`（`$X/SKILL.md:7`、`$P/SKILL.md:7`），但随文件分发的 `$X/LICENSE.txt:1-13` / `$P/LICENSE.txt:1-13` 是 **Apache-2.0**（Copyright Z.ai）。搬文本/代码前需确认以 LICENSE.txt 为准并在 gaea 侧保留 NOTICE/出处。

---

## 1. 取材与体量

### 1.1 xlsx 插件（`spreadsheets-plugin`）

| 文件 | 体量 | 作用 |
|---|---|---|
| `.zcode-plugin/plugin.json` | 1.4KB | 插件清单：MCP server `dist/server.mjs` + UI surface `ui://spreadsheets/panel.html` |
| `agents/visual-judge.md` | 5.4KB | 视觉验收 agent（模型绑定 `GLM-5.3-Flash`，只读页面 PNG 出 JSON 判决） |
| `skills/xlsx/SKILL.md` | 349 行 / 18.9KB | 主入口：意图闸门 + 场景路由 + 设计原则 + 工具链 |
| `skills/xlsx/scenes/*.md` | 9 篇 / ~60KB | create / edit / analyze / analyze-recipes / convert / finance / finance_lite / advanced / vba / edit-patterns |
| `skills/xlsx/engines/*.md` | 4 篇 / ~45KB | design(23.3KB) / chart / chart-templates / vba-templates(13.8KB) |
| `skills/xlsx/quality/pipeline.md` | 214 行 | 三角色质检流水线（Blueprint → Builder → Inspector） |
| `skills/xlsx/xlsx.py` | 1313 行 / 54.8KB | 7 个 QA/操作子命令 |
| `skills/xlsx/templates/base.py` | 750 行 | 设计令牌 + 字体解析 + 样式工厂 + auto-fit |
| `skills/xlsx/templates/palettes.py` | 17.1KB | 12 套色板 + 三档关键词匹配 |
| `skills/xlsx/env_setup/*` | 5 文件 | env_check.sh / setup_windows.ps1(10KB) / setup_mac_linux.sh / setup.md / font_list.txt |
| `skills/xlsx-viewer/SKILL.md` | 3.7KB | 侧栏看板技能 |

### 1.2 pdf 插件（`pdf-plugin`）

| 文件 | 体量 | 作用 |
|---|---|---|
| `.zcode-plugin/plugin.json` | 1.3KB | MCP server + UI surface `ui://pdf/panel.html` |
| `agents/visual-judge.md` | 5.4KB | 同上（两插件共用同一份） |
| `skills/pdf/SKILL.md` | 1004 行 / 74.5KB | 主入口：Triage → Briefing → 路由矩阵 → Preflight → 8 段质检清单 |
| `skills/pdf/briefs/*.md` | 9 篇 / ~290KB | report(73KB) / academic(47KB) / creative(48KB) / creative-fixed-canvas(50KB) / poster(34.7KB) / process(18KB) / process-advanced / resume(10.8KB) |
| `skills/pdf/typesetting/*.md` | 10 篇 / ~120KB | overflow(22.8KB) / fill-engine(20.6KB) / cover(14.2KB) / pagination(13.7KB) / charts(12.5KB) / cover-backgrounds(16.6KB) / palette / geometry / typography(317B) |
| `skills/pdf/scripts/pdf.py` | 3076 行 / 116.8KB | 30+ 子命令（env/extract/pages/meta/form/convert/font/toc/palette/sanitize） |
| `skills/pdf/scripts/design_engine.py` | 2816 行 / 125KB | 意图→色板→SVG→布局→HTML 编译器 |
| `skills/pdf/scripts/pdf_qa.py` | 899 行 / 31.6KB | 成品 PDF 质检（PyMuPDF） |
| `skills/pdf/scripts/toc_validate.py` | 2069 行 / 80KB | DOCX/PDF 目录校验 + 修复 |
| `skills/pdf/scripts/poster_validate.py` | 1336 行 / 59.7KB | HTML/PDF/TeX 三态校验器 |
| `skills/pdf/scripts/cover_validate.js` | 367 行 | Playwright DOM 量测：文字×装饰线重叠 |
| `skills/pdf/scripts/html2pdf-next.js` | 754 行 / 31.9KB | Playwright + pdf-lib 渲染器 |
| `skills/pdf/scripts/html2poster.js` | 288 行 | 海报渲染器（动态高度） |
| `skills/pdf/scripts/cover_render.py` | 487 行 / 26.7KB | ReportLab 封面引擎（9 模板） |
| `skills/pdf/configs/*` | 3 篇 / 22KB | visual_framework / components / fonts |
| `skills/pdf/references/*.tex` | 2 份 | AltaCV / Academic CV 模板 |

---

## 2. xlsx 侧解剖

### 2.1 用什么库

`$X/xlsx.py:19-37`：`argparse / json / os / platform / re / shutil / subprocess / sys / zipfile`，唯一三方依赖 **openpyxl**（`$X/xlsx.py:34-37`，导入失败直接 `sys.exit(1)`）；pivot 与样式分支里再局部导入 `openpyxl.styles/chart`（`$X/xlsx.py:812`、`$X/xlsx.py:962`、`$X/xlsx.py:979`、`$X/xlsx.py:1013`）。

工具链声明在 `$X/SKILL.md:205-218`：

| 工具 | 用途 |
|---|---|
| openpyxl | 公式、格式、图表、单元格级控制 |
| pandas | 数据分析、批量操作、CSV/TSV |
| `load_workbook(read_only=True)` | 大文件读 |
| `Workbook(write_only=True)` | 大文件写 |
| `templates/base.py` | 设计令牌 / 字体解析 / 样式工厂（单一真源） |
| `xlsx.py` | QA 命令 |
| LibreOffice | `recalc`、xlsx→PDF、csv→xlsx（`$X/SKILL.md:233-279` 强制安装条款） |

即：**没有自研 xlsx 引擎**，全是 Python 生态 + LibreOffice 外部进程。工作簿元数据固定 `wb.properties.creator = "Z.ai"`（`$X/SKILL.md:216`）。

### 2.2 SKILL.md 的决策树（三道闸门）

**闸门 1 — 意图闸门**（`$X/SKILL.md:40-50`）：先判断是否真需要表格。述职/调研报告 → docx；汇报/演示 → pptx；合同/证书/"PDF" → pdf；只要图不要表 → charts。确认 xlsx 后才进路由。

**闸门 2 — 场景路由器**（`$X/SKILL.md:102-133`，原文树）：

```
├─ 有现存文件?
│  ├─ 是 → 改内容/结构? → scenes/edit.md ；只读分析 → scenes/analyze.md
│  └─ 格式转换(CSV↔XLSX/JSON/PDF 表) → scenes/convert.md
├─ 从零创建?
│  ├─ 财务/预算/预测?
│  │  ├─ 复杂(DCF/LBO/三表联动/敏感性/IB model) → scenes/finance.md
│  │  └─ 简单(预算/费用/收支/项目成本/记账) → scenes/finance_lite.md
│  └─ 通用表/报表/模板 → scenes/create.md
├─ 批处理/大文件/保护/校验 → scenes/advanced.md
├─ VBA/宏/自动化 → scenes/vba.md + engines/vba-templates.md
├─ 需要图表 → append engines/chart.md
└─ 需要设计体系 → append engines/design.md
```

判定关键词在 `$X/SKILL.md:137-141`：finance（DCF/LBO/P&L/利润表/资产负债/valuation/IRR/敏感性/scenario）、finance_lite（预算/budget/费用/expense/收支/记账/报销/ROI）、VBA（宏/macro/VBA/自动化/.xlsm/按钮/auto-run）。

**闸门 3 — 强制全量加载**（`$X/SKILL.md:77-96`）："Always load ALL matched files. No shortcuts, no lazy loading." 8 条加载序：SKILL.md → 场景文件（**整篇读**）→ 图表则加 `engines/chart.md` → 有样式输出则加 `engines/design.md` → analyze 加 `analyze-recipes.md` → edit 加 `edit-patterns.md` → VBA 加 `vba-templates.md` → 永远跑 `quality/pipeline.md`。设计引擎默认加载（`$X/SKILL.md:96`）。

多意图示例见 `$X/SKILL.md:59-71`（如"做财务模型+图表+导出 PDF 摘要" = finance + chart + 移交 pdf skill），执行顺序固定为 **数据准备 → 分析 → 可视化 → 样式 → QA**（`$X/SKILL.md:73`）。

七条设计原则（`$X/SKILL.md:147-174`）里最硬的四条：
- 活公式保证：派生值用公式；但**若产物由 Python 校验**，TOTAL/SUM 行改写计算值（openpyxl 不求值，`data_only=True` 返回 `None`）——`$X/SKILL.md:150`。
- 零公式错误：所有除法包 `IFERROR` 或 `IF(denom=0,...)`，共享分母用绝对引用——`$X/SKILL.md:152-153`。
- 兼容优先：禁用动态数组函数（`FILTER/UNIQUE/XLOOKUP/SORT/SEQUENCE/LET/LAMBDA/RANDARRAY`）——`$X/SKILL.md:156`。
- 原文件只读，交付写新文件——`$X/SKILL.md:172-174`。

### 2.3 scenes 分工

| 场景 | 职责 | 关键证据 |
|---|---|---|
| `create.md` | 从零建表：PLAN→STYLE→BUILD→QA→PIVOT→VALIDATE 六步；画布 B2 起、行 2 标题/行 3 间隔/行 4 表头/行 5+ 数据/末行汇总；冻结窗格与打印设置（`page_setup`/`print_area`/`print_title_rows`） | `$X/scenes/create.md:10-17`、`24-32`、`93-105` |
| `edit.md` | 改现有文件：**Preserve First**；`.xlsm` 必须 `keep_vba=True` 否则静默毁宏；默认另存 `<stem>_updated.xlsx`，原地改前先 `<stem>_backup.xlsx`；1a 结构勘察 / 1b 语义采样（强制）/ 1c 跨表列语义映射（禁止按列位拷贝，`$X/scenes/edit.md:78-93`）/ 1d `normalize_cell_value` 规范化 | `$X/scenes/edit.md:11-17`、`21-31`、`57-93`、`95-105` |
| `analyze.md` | 数据→Excel：LOAD/EXPLORE/ANALYZE/DESIGN/BUILD/CHART/QA/PIVOT/VALIDATE；A–E 五阶段框架；多表报表布局（Dashboard/Detail/Charts/Raw Data）；数据溯源 Sources 表 | `$X/scenes/analyze.md:8-20`、`22-54`、`58-65`、`73-81`、`85-89` |
| `analyze-recipes.md` | 可复用代码：加载探索、聚合分组、时间序列、同比、统计、清洗、pandas→openpyxl 桥、KPI 卡、交叉校验 Review 表、pivot 工作流、键值交替列、FIFO 分摊公式 | `$X/scenes/analyze-recipes.md:7,25,43,54,66,79,96,113,132,150,181,200` |
| `convert.md` | 格式转换矩阵（CSV/TSV/JSON/XLSX/PDF 表/图片 OCR）+ 编码坑表（gbk/utf-8-sig/shift_jis）+ 转换后 6 条验收 | `$X/scenes/convert.md:8-15`、`116-133` |
| `finance.md` | 复杂财务模型：标准三表/假设层结构、公式构造规则（禁魔数、保护除法、期内一致）、假设表布局、IB 版式（分区头/缩进/汇总）、勾稽校验行、敏感性双变量表、P&L / 预算实际 / SaaS 指标 / 项目预算四模板 | `$X/scenes/finance.md:11,36,52,63,81,120,167,189,237` |
| `finance_lite.md` | 轻预算：何时用 lite vs 复杂、标准结构、差异/差异率/累计/毛利率/状态公式、数字格式、简版条件格式、月预算/报销单/项目成本三模板 | `$X/scenes/finance_lite.md:11,28,64,91,115,165` |
| `advanced.md` | 大文件（read_only/write_only/chunk）、批处理多文件、下拉校验、条件格式、工作表保护、命名区域、筛选排序、合并单元格、性能、VBA 模块检视 | `$X/scenes/advanced.md:8,54,89,130,150,172,188,204,225,238` |
| `vba.md` | 宏生成与保留：安全优先、`vba_archive` 读写、xlsm 规则、代码规范、常用模式、三种注入法（保留/模板/裸 `vbaProject.bin`）、安全检查表、性能、调试 | `$X/scenes/vba.md:9,17,34,45,106,187,235,251,284` |
| `edit-patterns.md` | 8 个编辑模式：块检测、空行预滤、排序+公式重写、分组归并、组内最大保并、序列填充、零显示为空、并排表识别 | `$X/scenes/edit-patterns.md:7,47,61,91,126,156,186,201` |

分层的实质：**场景=流程与业务语义，引擎=横切能力，quality=门禁**。图表/设计永远"追加"而不是独立场景（`$X/SKILL.md:129-132`）。

### 2.4 quality/pipeline.md 的质检流程（逐条）

三角色顺序协作（`$X/quality/pipeline.md:3`）：Blueprint Architect → Builder → Inspector。

**工具面**（`$X/quality/pipeline.md:11-19`）：
| 命令 | 目的 | 调用方 |
|---|---|---|
| `recalc <file>` | LibreOffice 重算 + 扫错误 | Builder 自检 |
| `audit <file>` | 公式错误 + 零值 + 隐式数组扫描 | Builder 自检 |
| `scan <file>` | 越界/含表头/小聚合/模式不一致 | Builder 自检 |
| `inspect <file> --pretty` | 结构 JSON | Architect |
| `pivot <in> <out> ...` | "PivotTable"（实为聚合表，见 §3.6-⑤） | Builder 最后一步 |
| `chart-verify <file>` | 图表是否有数据 | Builder 自检 |
| `validate <file>` | 结构校验（放行门） | Inspector |

**Role 1 Blueprint Architect**（`$X/quality/pipeline.md:39-48`）：拆解显式/隐式需求；逐表映射（表名、列结构、公式依赖、跨表引用）；识别数据流（源→派生→汇总）；歧义必须先问。产出是心智蓝图，不落文件。

**Role 2 Builder**（`$X/quality/pipeline.md:52-81`）：
- 单表纪律：写完一张表→存盘→自检链 `recalc → audit → scan (+chart-verify)` → 全绿才进下一张（`$X/quality/pipeline.md:56-75` 流程图）。
- 三条硬约束：**禁止批量后检**（早期错误会静默传播）、**禁止错误后移**（一张表有未解错误就阻塞后续）、**禁止静默交付**（没过自检的只是草稿）——`$X/quality/pipeline.md:77-81`。
- Pivot 特殊排序（`$X/quality/pipeline.md:83-104`）：永远是**最后一个数据操作**；示例命令含 `--values "Revenue:sum,Units:count" --rows --cols --filters --location --style finance --chart bar`；聚合支持 sum/count/avg/max/min；**"Never modify pivot output with openpyxl afterward — it corrupts the pivotCache"**（`$X/quality/pipeline.md:104`）。

**Role 3 Inspector**（`$X/quality/pipeline.md:108-139`）：
- 语义检查（改/转类任务）：行数是否符合预期、列合计是否与源一致、抽 2-3 行对照、公式是否可在 Python 求值（跨表/自引用则比计算值）。模板见 `$X/quality/pipeline.md:123-130`（`abs(src_total - out_total) < 0.01`）。
- 结构检查（放行门）：`validate`，**exit 0 = 可交付；非 0 = Builder 必须从头用修正后的代码重建**（`$X/quality/pipeline.md:132-139`）。

**已知陷阱表 14 条**（`$X/quality/pipeline.md:143-162`，逐条）：
1. `data_only=True` 后保存 → 公式被缓存值永久替换。
2. 列号算错（64 ≠ "BK"）→ 用 `get_column_letter()`。
3. 行列基准混淆（DataFrame 第 5 行 = Excel 第 6 行）。
4. NaN 漏进公式 → `=A1+nan` 成坏串；先 `pd.notna()`。
5. 跨表引用漏引号（`Sheet 1`）。
6. 除零 → `IFERROR`/`IF(denom=0,...)`。
7. 文本以 `=` 开头 → `#NAME?`，前缀 `'`。
8. 隐式数组公式 → `#N/A`；改用 `SUMPRODUCT`。
9. 图表空白 → 公式无缓存值，建图前先 `recalc`。
10. 隐藏行 → 图表跳过数据；`chart.plot_visible_only = False`。
11. 图表重叠 → 每图约 15 行 + 2 行间隙算锚点。
12. 新写公式用 `data_only=True` 验 → 得 `None`；改为 Python 算期望值比对。
13. 手工调换行排序 → 公式引用不跟着走；需按新行号重生成公式串。
14. NBSP `\xa0` / ZWSP `\u200b` 看着空白却 `is not None` → 统一规范化。

**交叉校验 Review 表**（`$X/quality/pipeline.md:166-197`）：分析型交付必须在工作簿内嵌 Review sheet（tab 色 `FFC000`），列为 Check/Expected/Actual/Status，Status 用活公式 `=IF(B2=C2,"✓ PASS","✗ FAIL")`；每个 Summary/Dashboard 指标都要有回溯源的交叉核对公式；Review 放在最后一页（Sources 之前）。

**Release 清单 10 条**（`$X/quality/pipeline.md:201-214`，逐条）：每表过自检链 ✓；语义检查过 ✓；`validate` 返回 0 ✓；**每个数据表都调过 `auto_fit_columns()` + `auto_fit_row_heights()`**（防文字裁切）✓；清理临时/重试产物 ✓；用户原文件未被改动（备份留下不算临时产物）✓；重试版本只留最新 ✓；输出目录只剩预期交付物 ✓；`.xlsm` 保留 VBA 模块 ✓；生成的 VBA 过 `scenes/vba.md` 安全检查 ✓。

### 2.5 xlsx.py 七命令逐条（含判定逻辑）

| 命令 | 行号 | 实际检查内容 | 退出码 |
|---|---|---|---|
| `recalc` | `$X/xlsx.py:338` | 先 `_setup_libreoffice_macro()` 写 `Module1.xba`（`$X/xlsx.py:172-214`，宏体就是 `RecalculateAndSave`：`calculateAll` + `store` + `close`），再 `soffice --headless ... vnd.sun.star.script:Standard.Module1.RecalculateAndSave`（`$X/xlsx.py:250-255`，非 Windows 再套 `timeout`/`gtimeout` 包装 `:272`），然后扫全表 7 类错误串 | 有错 1 |
| `audit` | `$X/xlsx.py:367-447` | 双开工作簿（`data_only=True` + 公式态），逐公式比：① 缓存值命中 `EXCEL_ERRORS`（`#VALUE!/#DIV/0!/#REF!/#NAME?/#NULL!/#NUM!/#N/A`，`$X/xlsx.py:60`）② 缓存值为 0 的公式（疑似引用错）③ 命中 `IMPLICIT_ARRAY_PATTERNS`（`$X/xlsx.py:63`）。输出 total/error/zero/implicit 四计数，错误列表截断 50 条 | `1 if errors else 0`（`$X/xlsx.py:447`）——**零值/隐式数组只报不拦** |
| `scan` | `$X/xlsx.py:454-596` | 四类：① `out_of_range`（跨表/同表区间超数据行 3 倍且 >100 行，`$X/xlsx.py:495-521`）② `header_included`（`SUM/AVERAGE/...(\s*[A-Z]{1,3}1:` 从第 1 行起算，`$X/xlsx.py:523-535`）③ `small_aggregate`（聚合跨度 ≤2 格，`$X/xlsx.py:537-551`）④ `inconsistent_pattern`（同列公式行号归一后模式不一致，且少数派 ≤2 行，`$X/xlsx.py:553-579`） | `1 if findings else 0`（`$X/xlsx.py:596`） |
| `inspect` | `$X/xlsx.py:603` | 结构→JSON（表名/维度/数据区/表头） | — |
| `pivot` | `$X/xlsx.py:702-1059` | 读源区表头→Python 分组聚合（`$X/xlsx.py:790-801`）→写新表并套 `finance`/`monochrome` 两套样式→可选柱/线/饼图。**不是 OOXML PivotTable，是静态汇总表** | — |
| `chart-verify` | `$X/xlsx.py:1107-1154` | 遍历 `ws._charts`，看每个 series 的 `numCache.ptCount` 或 `numLit` 是否有数据（`$X/xlsx.py:1083-1095`）；空图先自动 recalc 再复查（`$X/xlsx.py:1126-1133`） | 有空图 1 |
| `validate` | `$X/xlsx.py:1161-1270` | ① `FORBIDDEN_FUNCTIONS` 命中（`$X/xlsx.py:72`，Excel 2019 及更早不支持）② 文本被误存为公式（两条启发式：`=` 开头且无运算符；`data_type=='f'` 但体内无函数/引用，`$X/xlsx.py:1197-1221`）③ 公式引外部文件（`_EXT_FILE_REF_PATTERN`，`$X/xlsx.py:92`、`1224-1232`）④ **zip 内 `.rels` 出现 `Target="C:\` 绝对 Windows 路径或 `file:///`**（`$X/xlsx.py:1234-1253`） | `0 if not issues else 1`（`$X/xlsx.py:1270`） |

一句话评价：`recalc/audit/scan/validate` 是**规则密度最高、最适合 Go 化**的部分（纯 regex + 阈值 + zip 读），而 `pivot`/图表生成是 openpyxl 生态绑定且命名失实。

### 2.6 templates/base.py 提供什么抽象

`$X/templates/base.py` 750 行，7 个能力块：

1. **字体跨平台解析**（`$X/templates/base.py:27-67`）：`_resolve_font()` 按 `platform.system()` 给候选集打表（Darwin/Windows/Linux 各一组），命中即返回，否则回退链首项；`CJK_BODY_CHAIN = [Noto Sans SC, Noto Sans CJK SC, Source Han Sans SC, WenQuanYi Micro Hei]`（`:43-48`）；**`HEADER_BOLD = FONT_NAME not in _HEAVY_FONTS`**（`:63-67`）——重笔画字体不加粗，靠背景+白字拉层级。
2. **色板令牌 + 运行时切换**（`:74-107` 三色规则令牌：PRIMARY/PRIMARY_LIGHT/SECONDARY/ACCENT×3/NEUTRAL×6/HEADER_TEXT/CHART_COLORS/CF_*）；`use_palette(prompt)`（`:114-130`）与 `use_palette_explicit(style)`（`:133-144`）通过 `_apply()` 用 `global` 重写全部模块级颜色（`:152-186`）——**全局可变状态**，调用顺序敏感（必须先于建样式）。
3. **列宽/行高/数字格式字典**（`:193-203` COLUMN_WIDTHS 9 类型；`:368-375` ROW_HEIGHTS 6 类型；`:210-218` FORMATS 7 种）。
4. **样式工厂**（`:225-251` font_title/header/subheader/body/caption/kpi/kpi_label；`:327-345` fill_header/fill_total/fill_data_row/border_header/border_total；`:348-361` align_*）。
5. **图表标题跨端一致 hack**（`:254-324` `make_chart_title`）：把字体塞进 `<tx><rich><defRPr>/<rPr>`，Y 轴用 `bodyPr rot=-5400000`，**轴标题只用单 run + `\n` 不换段**（多段会被 WPS 渲染成互相叠的文本框）——这是踩坑换来的兼容知识。
6. **工作表装配**（`:378-397` `setup_sheet`：隐藏网格线、A 列宽 3、行 1/2/3 高、B2 合并标题；`:399-437` style_header_row / style_data_row / style_total_row）。
7. **自适应尺寸**（`:575-684` `auto_fit_row_heights`：按 `unicodedata.east_asian_width` 把 CJK 记 1.7 宽，估行数 → 22pt + 16pt/行；`:687-750` `auto_fit_columns`：按**数据**而非表头定宽，表头超出则 `wrap_text`）。

**缺口（重要）**：base.py **没有打印区域/页面设置抽象**。打印相关只有 `scenes/create.md:93-105` 的 4 行示例（`page_setup.orientation/fitToWidth/fitToHeight`、`print_area`、`print_title_rows`）。也就是说"表格→可直接打印/PDF"的能力在 xlsx 侧是很薄的，靠 xlsx→PDF 交给 LibreOffice。

### 2.7 engines：chart / chart-templates / vba-templates 的办公价值

- **`engines/design.md`（23.3KB，602 行）是设计规范本体**：Borderless-first（只为有数据的区域上样式，`$X/engines/design.md:9-15`）、Three-Color Discipline（≤3 色角色 + 中性灰，`:21-23`、`:31-38`）、12 套风格色板的触发词与定位表（`:70-88`）、三档匹配（显式关键词→场景推断→professional 兜底，`:89-93`）、财务场景 IB 颜色编码（蓝=手输/黑=公式/绿=跨表/红=外部，`:107-116`）、字号层级（`:129-141`）、B2 画布原点（`:231-247`）、行高/列宽标准（`:248-274`）、边框只允许 2 条（`:316-341`）、条件格式仅用 accent（`:421-471`）、图表色序（`:475-493`）、**12 条设计自检清单**（`:586-602`）。
- **`engines/chart.md`（87 行）= 选型规则**：原生图 vs matplotlib 图的分工（`$X/engines/chart.md:7-14`）、尺寸三档与"每图 15 行 + 2 行间隙"防重叠（`:18-28`）、9 种数据形态→图型映射与"避免什么"（`:32-47`）、按表头词自动推荐（日期/类别/占比/预算vs实际/收入成本利润/增长，`:49-58`）、6 条铁律（`tight_layout`、`titles_from_data=True`、建图前 recalc、隐藏行、等比缩放、中文字体，`:62-69`）。
- **`engines/chart-templates.md`（164 行）= 可抄代码**：柱/线/饼/组合双轴四段（`$X/engines/chart-templates.md:10-95`）、matplotlib 中文字体注册（Noto Sans SC + DejaVu 兜符号，`:99-112`）、标准模板与嵌图等比缩放（`:114-145`）、`recommend_chart()` 启发式（`:147-164`，含"合计≈100 → 饼图"这种业务判据）。
- **`engines/vba-templates.md`（13.8KB）= 6 个成品宏**：月度报表自动生成、多表批处理、数据校验清理、多文件合并、按钮触发自动化、保护工作表+可编辑区（`:9,114,178,281,373,395`）。

**办公价值判断**：chart.md 的"数据形态→图型 + 反例"表是**可以直接变成 gaea 图表推荐器规则**的高价值资产（比任何代码都值钱）；chart-templates 的代码是 openpyxl 专用，只能当参考；VBA 三件套对"帮用户做 Excel 自动化"确实有市场，但**平台风险极高**（见 §6 C4）。

---

## 3. PDF 侧解剖

### 3.1 管线全貌（四条）

`$P/SKILL.md:680-687` 的"每个 brief 的技术栈"表 + `$P/SKILL.md:55-87` 的路由树，可还原为：

| 管线 | 入口 | 链路 | 引擎 |
|---|---|---|---|
| **Report**（报告/提案/白皮书/合同/发票/试卷/ATS 简历） | `briefs/report.md` | 内容→`palette.cascade` 出色板→ReportLab 写 story→**`cover_render.py` 单独出封面 PDF**→`pypdf` 合并为第 1 页 | ReportLab + pypdf（必需依赖，`$P/SKILL.md:20-25`） |
| **Creative**（海报/邀请函/信息图/证书/名片/信封/PPT 式） | `briefs/creative.md`(+`poster.md`) | LLM 输出 **JSON blueprint** → `pdf.py convert.blueprint` → `design_engine.py compile` 出 HTML → `html2pdf-next.js`(Playwright) 出 PDF | Playwright/Chromium（可选、按需装） |
| **Academic**（论文/数学/IEEE/ACM/Beamer/学术简历） | `briefs/academic.md` | LaTeX（`ctex` + `amsmath` + `tikz`/`tabularx`）→ `convert.latex`（Tectonic）→ 同样用 `cover_render.py` 出封面（模板 06-09）→ pypdf 合并 | Tectonic（可选；内置二进制仅 macOS arm64，`$P/scripts/pdf.py:1924-1928`） |
| **Process**（已有 PDF：提取/合并/拆分/表单/Office→PDF/OCR/压缩/水印） | `briefs/process.md` | `pdf.py` 30+ 子命令：pikepdf / pdfplumber / PyMuPDF / **LibreOffice** | LibreOffice 强制（`$P/SKILL.md:698-715`） |

Creative 自动管线的实参在 `$P/scripts/pdf.py:2007-2102`：从 LLM 回复里用 `re.search(r'```(?:json)?\s*(\{.*?\})\s*```')` 抽 JSON（`:2023`）→ 递归扫 `U+FFFD`/孤立代理项并打印告警（`:2038-2067`）→ 落 `<stem>_pure_blueprint.json` → `subprocess` 调 `design_engine.py compile`（`:2074-2084`）→ 调 node `html2pdf.js`，**宽高写死 720×960**（`:2086-2097`）。

Report 的封面与正文解耦也写在 `$P/briefs/report.md:386-509`（body.pdf 不含封面；封面单独出再合并；`$P/briefs/report.md:509` 还有"封面隔离铁律"）。

> **注意**：`convert.html`（`:1876-1899`）与 `convert.blueprint`（`:2086-2097`）都调用 `html2pdf.js`，但仓库里只有 `html2pdf-next.js`（`$P/scripts/html2pdf-next.js`）。**这是一处前后不一致**：直接跑 `convert.html` 会报 `html2pdf.js not found`（`$P/scripts/pdf.py:1882-1884`）。移植时不要照抄这个引用。

### 3.2 design_engine.py 的职责边界

2816 行，做 6 件事，**边界很清楚：它只产 HTML/CSS 与色板数据，不产 PDF、不做版式合规判定**。

| 职责 | 入口 | 说明 |
|---|---|---|
| ① 意图推导 | `derive_intent()` `$P/scripts/design_engine.py:99` | 从标题/描述文本推 intent（serenity/elegance/minimalism/warmth/neutral/tension/energy 等） |
| ② 色板生成 | `generate_color_palette()` `:187`、`generate_cascade_palette()` `:362` | HSL 锁定 + 和谐关系（complementary/split/triadic/analogous/monochrome）；cascade 版按"角色分层（cover/body/charts/semantic）"输出，强制 **面积 ∝ 1/饱和度** |
| ③ 色板审计 | `audit_palette()` `:1221-1276`、`audit_cascade_palette()` `:671` | 按 mode 卡 S/L 上限（bg/mid 上限 0.15-0.42，accent 0.48-0.78，`:1236-1248`）；L 护栏（dark bg ≤0.10、minimal bg ≥0.90、jewel ≤0.28、pastel ≥0.83，`:1250-1261`）；**WCAG：text:bg ≥ 4.5:1、accent:bg ≥ 2.5:1**（`:1263-1274`） |
| ④ SVG 背景生成 | `generate_flow_svg/grid/noise/supergraphic/ordered_texture/generative/continuous_flow/unified` `:733-1119` | 程序化生成 flow/grid/noise/超图形等底纹，多页用 `generate_unified_svg` 跨页连续（`:1047`） |
| ⑤ 布局计算 | `calculate_layout()` `:1132-1216`、`_divide_vertical` `:1196` | offset/centered/overlap 三种风格的分区坐标 |
| ⑥ Blueprint→HTML 编译 | `compile_blueprint()` `:2559-2816` | 读 JSON：`art_direction`(intent/palette_mode/color_harmony/background_svg/canvas/grid_gap/bleed) → 生成 `:root` CSS 变量 + `@page { size: Wpx Hpx; margin:0 }` + `html,body{width;min-height}`（`:2631-2647`）→ 多页走"连续画布 + 绝对定位 page-section"（`:2666-2741`）→ 组件渲染走 `render_component()`（`:2367`）+ 自动栅格分配 `_auto_assign_grid_areas()`（`:2193`）+ 按字数决定安全边距（>500 字 `8% 10%`、>200 字 `10% 12%`，`:2698-2704`）。还带 `_prevent_orphan_chars()`（`:1919`）防标点落行首 |

**不做什么**（这是它最值得学的设计）：不碰分页、不做 overflow 检测、不算行高、不校验合规——那些全部外置到 `typesetting/*.md`（人/LLM 遵守）与 `poster_validate.py`（机器判定）。引擎只负责"把结构化意图变成确定性的 CSS/HTML"。

### 3.3 typesetting 六件套逐条摘（最强资产）

#### ① `overflow.md`（630 行）——"内容太多"侧

- 核心哲学 **"Measure first, draw second"**：禁止裸 `draw_text(x,y)`，一切内容先过约束系统（`:15-17`）。
- **Bounding Box 系统**（`:21-43`）：每块有 `Max_Width = page_width - 左右边距`；双栏 `/2 - col_gap`；嵌套单元格再减 padding；**铁律：任何块渲染宽度不得超过父级 Max_Width**。
- **水平居中铁律**（`:47-78`）：左右边距必须相等；左对齐封面文字起点 0.10–0.15W；ReportLab `leftMargin == rightMargin`；LaTeX `\geometry{left=X,right=X}`；HTML `margin:0 auto`。
- **文字溢出：字体度量而非字数**（`:82-118`）：`stringWidth()` 量宽；表格单元格一律 `Paragraph()`；**CJK 在 10pt 时约 12pt/字**；URL 长串要 `wordWrap='CJK'` 或手工切。
- **图片/图表等比缩放**（`:161-215`）：`fit_image()` 按 `min(ratio_w, ratio_h)` 缩放；图片高度上限 `A4 高 × 0.35`；**HTML 侧 `max-height: 45vh`** ——否则高图 + `break-inside: avoid` 会独占整页（`:215`）。
- **横排 flex 溢出**（`:219-283`，这是 LLM 生成 HTML 的高频崩点）：≥3 个子项必须 `flex-wrap`；子项要 `min-width` + `flex-shrink`;箭头等分隔符必须做伪元素（`:249-263`）；**一旦 `scrollWidth > clientWidth`，Playwright 会缩放整页，导致所有页面内容左移**（`:223`）；4-6 项横排、7+ 项转竖排/网格的阈值表（`:265-272`）。
- **表格列宽按权重分配**（`:287-359`）：`calculate_col_widths()` 先量每列最宽内容（+8pt padding），总宽不足则按比例加宽、超出则按比例压缩并保底 30pt；`build_safe_table()` 里**断言 `sum(colWidths) <= available_width + 0.5`**（`:354-355`）。
- **LaTeX 表格**（`:361-381`）：`tabularx{\columnwidth}` / `table*` 跨栏 / `resizebox` 兜底且缩放后字号 ≥6pt。
- **公式溢出**（`:383-417`，双栏第二大坑）：一行两式必溢出，改 `align`/`multline`/抽子式；**启发式：equation 体 >60 字符基本会溢出**（`:415`）。
- **算法块**（`:419-431`）：双栏必须 `\SetAlFnt{\small}`，长 Input/Output 手动折行。
- **`\columnwidth` vs `\textwidth`**（`:433-443`）：`table` 浮动体内用 `\columnwidth`（≈252pt），`table*` 才用 `\textwidth`；`check-tex` 会把单栏 float 里的 `\resizebox{\textwidth}` 判为 error。
- **降级阶梯**（`:447-475`，顺序固定）：① 换 `Paragraph` 折行 → ② 字号 -1pt（**下限单栏 14pt / 双栏 12pt**）→ ③ 减 padding（下限 4pt）→ ④ 转横版（须用户同意）→ ⑤ 拆元素 → ⑥ 记日志照渲。
- **纵向溢出 = 分页**（`:492-541`）：Y 游标模型；`KeepTogether([heading, first_para])`、`KeepTogether([chart, caption])` 强制；长表 `repeatRows=1`。
- **两遍渲染**（`:559-599`）：先建 Layout Tree 量尺寸 → `check_overflow()` 找越界 → 修 → 再渲；实践上映射到 ReportLab `doc.multiBuild()`。
- **三路线对照表**（`:603-613`）+ **10 条自检**（`:617-630`）。

#### ② `pagination.md`（367 行）——跨页完整性

- **末页留白控制**（`:11-35`）：末页填充率 ≥40% 通过 / 25-40% 可接受 / **<25% 必须回溯**；回溯手段按优先级：压段间距（2-4px）→ 行高 1.7→1.55（不低于 1.5）→ 字号 16→15px（不低于 14px）→ 删冗余 → 合并小节。
- **表格跨页完整性**（`:39-86`）：`break-inside: avoid`；必须拆时 `thead{display:table-header-group}`；行 ≤8 整表不拆、>8 允许拆但表头重复；卡片网格同理。
- **CJK 标点**（`:89-131`）：`line-break: strict`（Chromium 原生，禁行首 `，。、；：！？）】》…—`、禁行尾 `（【《`）；ReportLab 用 `wordWrap='CJK'` + `TA_JUSTIFY`。
- **大节 3/4 阈值**（`:135-196`，最有价值的"人味"规则）：H1 级标题若剩余空间 < 页高 25% 就强制换页，否则"标题挤在底部 25%"很难看；实现：ReportLab `CondPageBreak(available_height*0.25)`、CSS 后处理 JS 把 `relativeY > 0.75` 的 h1 加 `break-before`、LaTeX `\needspace{0.25\textheight}`。**只对 H1 生效**，H2/H3 只做"标题不落页底"。
- **其他防拆**（`:200-229`）：标题 `break-after: avoid`；图/卡 `break-inside: avoid`（必须配 `max-height`）；列表 `li{break-inside:avoid}` + 末项 `break-before: avoid`。
- **五区页码规范**（`:247-275`）：封面=逻辑第 1 页但**不显示**；前置（目录/摘要/致谢）用**小写罗马数字**从 i；正文阿拉伯数字**重置为 1**；附录与参考文献**继续正文编号不重置**；**禁止 "Page X of Y"（分母禁止）**（`:263`）；无前置则跳过第 2 区；无封面则正文直接 1。
- 偏差场景（`:362-367`）：单页文档（证书/信/海报）不编；≤3 页用连续阿拉伯；试卷每页都编（含第 1 页）。

#### ③ `fill-engine.md`（527 行）——"内容太少"侧（与 overflow 互为镜像）

- **安全网 1 可读性红线**（`:24-58`）：正文 ≥14pt（单栏）/≥12pt（双栏）；默认 15pt(CJK)/14pt(Latin)；**H1 ≥32pt、H2 ≥24pt、H3 ≥18pt**；overflow 的降级字号不得击穿此线——**"放不下就换页，不缩字"**（`:58`）。
- **安全网 2 填充率与段落膨胀**（`:62-163`）：虚拟渲染算 `fill_ratio = 内容高 / 可用高`；四档：≥80% 正常 / 65-80% 轻膨胀（只调行高+段距）/ 40-65% 全膨胀（+字号+组件）/ <40% 全膨胀 + 黄金比锚定（`:86-90`）。膨胀公式：行高线性插值至上限 **2.2**（`:97-112`）；段距注入剩余空间的 **40%**（`:125-141`）；字号最多 **+2pt**（`:151-161`）。
- **安全网 3 组件级膨胀**（`:167-226`）：表格 cell padding +10~20pt（`:174-186`）、引用块字 1.5×/上下各 40pt（`:195-214`）、列表间距 ×1.5（`:219-226`）。
- **安全网 4 Y 轴黄金比锚定**（`:230-257`）：膨胀后仍 <40% 时，`offset = 剩余空间 × 0.382`（上 38.2% / 下 61.8%），把"没填满"变成"刻意的留白"；备选绝对居中 `remaining/2`（`:261-266`）。
- **三路线实现**（`:286-442`）：ReportLab `build_page_with_fill_engine()` 主入口顺序；LaTeX 用 `\fontsize` 定底线 + `\vfill` 做黄金页；HTML/JS 的 `runFillEngine()` 用 `scrollHeight/clientHeight` 算比例后改 `--body-line-height`/padding。
- **执行顺序图**（`:446-499`）+ 10 条自检（`:514-527`）。

#### ④ `cover.md`（234 行）——封面引擎 V4.0

- **能不能不做**（`:24-34`）：正式报告/提案/白皮书/作品集 → 要；简历/菜单/卡片/邀请函 → 不要；实验报告/论文 → 视模板。**"封面是可选件，拿不准就跳过"**（`:14`）。
- **单一 PDF 输出**（`:16-20`）：封面单独渲染成 1 页 PDF，用 pypdf 合为第 1 页；**禁止交付独立封面文件**；封面绝不与正文/目录同页。
- **引擎架构 A0.0-A0.3**（`:63-102`）：坐标系 A4(595.28×841.89pt) 但按 794×1123px 设计稿（96dpi）表达，helpers `X()/LW()/VH()/YT()/FSpx()`；**绝对 Y 锚点栅格，禁止流式布局**（块只在自己盒内生长，绝不推挤邻居）；字重系统表（Kicker/Footer 16pt+3pt 字距、Summary 16-18pt/1.6、Meta 16-22pt、Hero 45-66pt Heavy）；**数据绑定铁律：Hero=机构/对象名，Kicker=报告类型，不许反**；**强制 Summary 块 2-4 行**（无则自动生成）；Z 层级 Layer0 背景→Layer1 装饰（需裁剪）→Layer2 结构线→Layer3 文字；**线到文字最小间距 = U = W×0.05**。
- **九套模板 01-09**（`:110-145`）：01 HUD 数据终端 / 02 企业社论（大年份水印）/ 03 Monolith（左侧硬对齐+右侧竖排水印）/ 04 博物馆极简（四角 crop marks）/ 05 实心侧栏 / 06 学术竖锚（深色）/ 07 学术对称（深色）/ 08 学术期刊（关键词块）/ 09 机构版（白底黑框，可 `style:"kai"` 楷体）。每套都给锚点百分比与字号。
- **选型矩阵**（`:152-165`）：Intent × Document Type 二维表，无全局默认；论文/开题/学位 → 09 默认。
- **字体策略**（`:174-195`）：按角色（sans/serif/kai/latin）在宿主机上探测**可嵌入**字体（SimHei→微软雅黑→Heiti SC→文泉驿…；宋体族；楷体）；**ReportLab 只能嵌 TrueType 轮廓，CFF/PostScript 轮廓（PingFang、Hiragino、Noto CJK OTC、Source Han）一律跳过**（`:186-188`）；缺 Heavy 用 faux-bold 多重偏移叠印，不许手描边（`:189-190`）。
- **颜色四则**（`:199-214`）：封面主色=正文主题色；次色=主色亮度 ±20%；背景白/浅灰（学术 06-08 深色例外）；禁大面积深色/渐变/高饱和/彩虹/纹理/超 2 字体/剪贴画。
- **安全**（`:218-225`）：Hero 最多 3 行、下探到 40pt 为止；硬宽度（可纵向长，不可横向越界）；水印必须 100% 在页内（缩小而不裁切）；竖线≈相邻文字块高、横线≥最宽文字；稀疏内容居中，底部死白不超 40%。

#### ⑤ `geometry.md`（142 行）——装饰锚点

- 只做"非功能性装饰元素"，越抽象越高级（`:9-15`）；6 种基本形状语义表（`:19-28`）；5 个构图范式：错位叠印（3 层 0.15/0.25/0.4 透明度，`:34-47`）、尺度对比（大空心 + 小实心，`:49-62`）、网格交点（`:64-80`）、弧线流动（贝塞尔 + 端点圆，`:82-93`）、几何拼贴（`:95-110`）。
- 放置指南（封面 120-200px 右上/左下等，`:114-121`）；**只用 1 色、靠透明度分层（0.1→0.5）、笔画优于填充（实心 ≤30%）、线宽 0.3-0.8px**（`:125-131`）；五条禁令：具象图标、多色、>8 元素、对称居中、粗线/阴影（`:135-142`）。

#### ⑥ `palette.md` / `charts.md` / `typography.md`

- **palette.md（7.3KB）**：单色系 ≤5 色、所有色可溯源到主色、同层级不用不同色相区分（用明度/透明/边框）、渐变端点色相差 <20°、禁高饱和大色块（`$P/SKILL.md:845-852` 引用其结论）。
- **charts.md（12.5KB）**：图表与文字最小 24pt 间距、图表间 30pt、legend 不得压数据、相邻数值标签错位/旋转/择隐、饼图默认甜甜圈（hole 60-70%、中心放总量）、<5% 用引导线、<3% 并入 Others、X 轴标签均值 >5 个汉字转横向条形、折线只标首末极值、坐标轴去上右框线、网格 0.5pt/20% 透明、柱顶 2-4px 圆角、legend 去框（`$P/SKILL.md:894-907`）。
- **typography.md 只有 317 字节**——几乎是空壳，字号体系实际写在 fill-engine 与 briefs 里（这是资产不均衡的一处）。

### 3.4 质检四脚本逐条（含行号与阈值）

#### ① `pdf_qa.py`（899 行，PyMuPDF）——成品 PDF 的"像素/结构"体检

`$P/scripts/pdf_qa.py:9-21` 自述 12 项检查，实际函数与判据：

| # | 函数 | 行号 | 判据 / 阈值 | 级别 |
|---|---|---|---|---|
| 1 | `check_metadata` | `:658-686` | title/author/creator 缺失、或 title 为 `untitled`/`(anonymous)` | WARN |
| 2 | `check_page_size_consistency` | `:236-257` | 各页 `(w,h)` 四舍五入到 0.1 不一致 | WARN |
| 3 | `check_blank_pages` | `:166-182` | 无文字 + 无图 + 无矢量图 | **ERROR** |
| 4 | `check_punctuation` | `:124-163` | 行首命中 `。、，；：！？）】〛〉」』""…—～·`（`:38-45`）、行尾命中 `（【《〈「''"`（`:48-52`） | WARN（最多报 10 条） |
| 5 | `check_colors` | `:185-233` | 仅统计文字色/绘图色，过滤近灰（`max_diff > 20` 才算彩色） | 纯 info（`MAX_COLORS` 已废弃，`:57-58`） |
| 6 | `check_font_embedding` | `:569-595` | `page.get_fonts()` 的 ext 为空 = 未嵌入 | WARN |
| 7 | `check_helvetica_in_cjk` | `:598-655` | 文档含 CJK 且可见文字用 Helvetica 渲染 → 说明裸字符串塞进 Table（`:645-655`） | WARN |
| 8 | `check_text_overflow` | `:260-281` | block bbox 越页面 ±2px 容差 | WARN |
| 9 | `check_content_fill_ratio` | `:284-349` | 跳过第 1 页（封面）；中间页 <40%、**末页 <25%** 报警（`:334-338`） | WARN |
| 10 | `check_cover_bleed` | `:352-442` | 封面（或 poster 模式全页）内容/图形 bbox 四边留白 >**5%** 即"未满版"（`:422-424`）；无任何背景图形也警（`:405-413`） | WARN |
| 11 | `check_margin_symmetry` | `:445-475` | 文字块 <3 则跳过；左右边距差 >**页宽 5%** 报警（`:464`） | WARN |
| 12 | `check_table_centering` | `:478-566` | 从 drawings 的线段/矩形聚类成表格（横竖各 ≥2 条，`:550-553`）；左右边距差 >5%（`:558`） | WARN |
| 13 | `check_toc_without_cover` | `:689-719` | 第 1 页前 300 字含 "table of contents/contents/目录/目 录" → 有目录无封面 | WARN |
| 14 | `check_formula_overflow` | `:722-754` | 需 `--formulas`；以右侧边中位数为内容右边，单行且宽于半页、或含数学符（`:724`）超 10pt 即警 | WARN |

**汇总与放行口径**：`format_report()`（`:801-849`）——`total_issues==0` → **PASS**；有 ERROR → **FAIL**；只有 WARN → **WARN**。`SKILL.md` 的映射是 PASS 交付 / WARN 评估后可不改 / **FAIL 必须修完重出**（`$P/SKILL.md:820-822`）。

**两个必须知道的实现缺陷**：
- `check_last_page_fill()`（`:83-121`）**从未被 `run_qa()` 调用**（运行清单 `:776-795` 里没有它），而它内部第 `:118` 行引用了被注释掉的 `LAST_PAGE_MIN_FILL`（`:54-55`）——**一旦启用就是 `NameError`**。末页留白实际由 `check_content_fill_ratio` 承担。
- 除 blank page 外**全部是 WARN**，所以 `pdf_qa.py` 的"FAIL"只在真出现全空页时触发，门禁强度名不副实，靠 LLM 自觉按 SKILL.md 处理 WARN。

#### ② `poster_validate.py`（1336 行）——**最该搬的校验器**（HTML/PDF/TeX 三态，纯文本规则）

**`check-html`**（`:253-830`，逐 code）：

| code | 行号 | 判据 | 级别 |
|---|---|---|---|
| `FONT_NO_FALLBACK` | `:290` | `font-family` 无通用兜底（serif/sans-serif/…，`:38-39`）；可 `--fix` 自动补（`:186-188`） | **error** |
| `OVERFLOW_HIDDEN_CONTAINER` | `:321,342` | `html/body/.page/.poster` 等容器上 `overflow:hidden` → 裁内容 + 触发渲染器自动修 | **error** |
| `FIXED_SIZE_NO_SCREEN_ADAPT` | `:393-399` | 定尺寸页无 `@media screen` 自适应 | warning |
| `SCREEN_ADAPT_NO_SCALE` | `:425-428` | 有 `@media screen` 但无 scale/transform/zoom | warning |
| `REMOTE_IMAGE` | `:437-452` | 引用远程图（打印可能丢） | warning |
| `ABSOLUTE_PATH` | `:461-464` | 本地绝对路径素材 | warning |
| `NO_PAGE_SIZE` | `:470-473` | 缺 `@page size` | warning |
| `MISSING_MARGIN_RESET` | `:484-488` | 缺 `html,body{margin:0;padding:0}` | warning |
| `PAGE_SIZE_CSS_VAR` | `:499-504` | `@page` 里用 CSS 变量（浏览器不解析） | **error** |
| `WHITE_BACKGROUND` | `:525-528` | 纯白底 | warning |
| `TINY_FONT` | `:543-549` | 字号 **< 10px** | warning |
| `COLOR_CONTRAST` | `:564-571` | 文字/背景对比 **< 3.0:1** | warning |
| `MISSING_PRINT_BG_NOTE` | `:576` | 无打印背景说明 | warning |
| `OVERFLOW_DECORATION` | `:592-595` | 负定位可能出黑边 | warning |
| `BG_COLOR_MISMATCH` | `:635-639` | body 背景 ≠ canvas/poster 背景 | warning |
| `SCREEN_BG_MISMATCH` | `:679-684` | screen 媒体下 html 背景 ≠ body | warning |
| `SCREEN_NO_BG` | `:692-697` | screen 块无 html 背景色 | warning |
| `MULTIPAGE_BODY_BG_MISSING` | `:750-755` | 多页深色页但 html/body 无背景 → 页缝露白 | warning |
| `HEIGHT_LOCKED` | `:790-794` | 高度被锁死（内容会溢出） | warning |

**`check-pdf`**（`:836-959`）：`FILE_TOO_SMALL`（`:853`）、`DEPENDENCY_MISSING`（`:863`，无 PyMuPDF）、`PDF_UNREADABLE`（`:875`）、`TEXT_MISSING`（`:907`，文字抽不出来 = 排版/字体崩）、`SOURCE_HTML_NOT_FOUND`（`:914`）、`DIMENSIONS_UNREASONABLE`（`:923-928`，宽或高 **<200px**）、`ORPHAN_PAGE`（`:939`，末页孤立；poster 模式抑制）。

**`check-tex`**（`:973-1256`，纯 regex 扫 `.tex`）：

| code | 行号 | 判据 | 级别 |
|---|---|---|---|
| `TWOCOLUMN_DETECTED` | `:1001` | 识别 `twocolumn`/`\documentclass[...two column]`（info） | info |
| `BARE_TABULAR_OVERFLOW` | `:1028-1033` | 双栏下 ≥5 列 `tabular` 未包 resizebox/adjustbox/table* | **error** |
| `TABULAR_OVERFLOW_RISK` | `:1037-1041` | 双栏 4 列 tabular 无宽度约束 | warning |
| `TABULAR_WIDE` | `:1045-1048` | 单栏 ≥7 列无约束 | warning |
| `TABULAR_NO_FLOAT` | `:1059-1062` | tabular 不在 table/table* 内 | warning |
| `TABULARX_NOT_LOADED` | `:1075-1078` | 有 tabular 未载 tabularx | warning |
| `IMAGE_NO_WIDTH` | `:1090-1093` | `\includegraphics` 无 width/height/scale | warning |
| `CJK_ASCII_QUOTES` | `:1129-1135` | CJK 旁出现 ASCII `"`（LaTeX 会渲成右引号），跳过 verbatim/\texttt/\url | **error** |
| `EQUATION_DUAL_ON_LINE` | `:1163-1168` | equation 内 2+ 式用 `\quad` 连排 | warning |
| `EQUATION_OVERFLOW_RISK` | `:1174-1178` | equation 体 >80 数学字符 | warning |
| `RESIZEBOX_TEXTWIDTH` | `:1195-1201` | 单栏 float 里用 `\resizebox{\textwidth}` | **error** |
| `ALGORITHM_NO_SMALL_FONT` / `ALGORITHM_LONG_IO` | per `$P/SKILL.md:644-645` | 双栏算法未 `\SetAlFnt{\small}` / IO 行 >120 字符 | warning |

输出契约统一为 `{pass, source, check_type, errors[], warnings[], info[]}`（`:13-14`、`:811-825`），`--fix` 时返回 `fixed_file`（`:822`）。

#### ③ `toc_validate.py`（2069 行）——DOCX/PDF 目录专项

定位（`:1-36`）：`check-docx` / `check-pdf` / `check-conversion` 三态，另有 `fix-docx`（`:1079`）与 `fix_docx_accurate_pages`（`:1772`，两遍编译按真实页码回填）。退出码 0 通过 / 1 失败 / 2 脚本错（`:24-28`）。

**check-docx（`:503-680`）**：
| 检查 | 行号 | 判据 | 级别 |
|---|---|---|---|
| `TOC_FIELD_MISSING` | `:568-575` | 标题数 ≥3 却无 TOC 域 | warning |
| `TOC_PLACEHOLDER_EMPTY` | `:577-585` | 有域但无占位条目 | **error** |
| `TOC_HEADING_STYLE` | `:587-605` | TOC 之后"加粗且大字"的段落却非标准 Heading 样式 → 目录抓不到 | **error** |
| `TOC_ENTRY_MISMATCH` | `:607-626` | 目录条目与真实标题模糊匹配率 **<0.5** | **error** |
| `TOC_NO_PAGEBREAK` | `:628-651` | 目录后 2 段内无分页符 | warning |
| `TOC_HINT_STYLE` | `:653-678` | 提示文字未做灰/小样式 | warning |

**check-pdf（`:686-859`）**：`TOC_NOT_FOUND`（`:730-744`，>5 页却前 5 页无目录）、`TOC_ON_FIRST_PAGE`（`:746-757`，目录在第 1 页 = 缺封面或未分页，**error**）、`TOC_NO_ENTRIES`（`:779-785`，条目 <2，**error**）、`TOC_PAGES_INVALID`（`:787-801`，页码 >总页数，**error**）、`TOC_ALL_SAME_PAGE`（`:803-813`，所有条目同页码 = 占位未更新，**error**）、`TOC_LINKS_MISSING`（`:815-856`，无注释/超链，warning）。

**check-conversion（`:865-996`）**：`CONV_TOC_LOST`（`:931-938`，源有目录、目标无且 >5 页，**error**）、`CONV_HINT_LEAKED`（`:940-954`，提示语漏进 PDF，**error**）、`CONV_HEADING_DRIFT`（`:956-993`，标题数与目录条目数偏差 **>30%**，warning）。

#### ④ `cover_validate.js`（367 行）——封面"文字×装饰线"重叠检测

- 定位：**只测封面单页**，明确写了"不要用在多页文档（用 html2pdf-next 的 pre-render）和海报（用 html2poster）"（`:21-23`）。
- 手法（`:112-279`）：Playwright 打开 → `document.fonts.ready` 等字体（`:330-333`）→ `page.evaluate` 注入检测脚本：把元素分成"装饰线"（`<hr>`、高≤5px 且宽>20、宽≤5px 且高>20、宽高比 >15、只有 border 无文字，`:131-147`）与"文字元素"（标题/段落/span/li/td 或有直接文本节点，`:159-173`）→ 按最小面积去重父节点（`:176-179`）→ 横线测垂直间隙、竖线测水平间隙（`:190-269`）。
- 阈值：**最小间距 = 1U = 页宽 5%（A4≈40px）**（`:299`、文件头 `:9`），`--min-gap` 可覆盖（`:75`）。
- 退出码（`:16-19`）：0 无重叠 / 1 有重叠（stderr 逐条给"Text/Line/建议移动 Xpx"）/ 2 脚本或浏览器错误。它自身承认"legacy，只在手工 HTML 封面时用"（`$P/SKILL.md:755`）。

### 3.5 闭环：生成 → 校验 → 回修、阈值、轮次

**闭环结构（源码可见的四段式）**

```
① 生成前（HTML 路线强制）：poster_validate.py check-html  → error 必修，--fix 可自动改
        （$P/SKILL.md:491-513，"MANDATORY for ALL HTML→PDF paths"）
② 生成时：html2pdf-next.js preRenderHooks（$P/scripts/html2pdf-next.js:164-490）
        —— 渲染前注入度量/修正 JS（等价于把 overflow/fill-engine 规则落到 DOM）
③ 生成后：pdf_qa.py（成品体检）+ poster_validate.py check-pdf（渲染完整性）
        + 封面专项 cover_validate.js + 目录专项 toc_validate.py
④ 回修：FAIL → 改源重渲；WARN → 评估
        （$P/SKILL.md:820-822"Result FAIL → must fix and regenerate"）
```

**阈值汇总（跨脚本口径，注意彼此不完全一致——移植时必须统一）**

| 量 | 阈值 | 出处 |
|---|---|---|
| 末页填充率 | ≥40% 通过 / 25-40% 警告 / <25% 必改 | `$P/typesetting/pagination.md:33-35`；脚本里 `check_content_fill_ratio` 中间页 40%、末页 25%（`pdf_qa.py:335`） |
| 单页填充率（海报） | ≥70% | `$P/SKILL.md:483` |
| 正文填充率（一般） | ≥60% | `$P/SKILL.md:482` |
| 大节换页阈值 | 剩余 <25% 页高 | `pagination.md:139-146` |
| 正文字号底线 | 单栏 14pt / 双栏 12pt | `fill-engine.md:30-33` |
| 海报字号底线 | 正文 24px、主标题 50px、Meta 16px | `$P/briefs/poster.md:113-120` |
| 简历字号底线 | 12px(9pt) 全篇硬地板 | `$P/briefs/resume.md:21` |
| 简历填充率 | ≥85% 页高 | `$P/briefs/resume.md:33` |
| 满版留白 | 四边 ≤5% | `pdf_qa.py:422` |
| 边距对称 | 左右差 ≤5% 页宽 | `pdf_qa.py:464` |
| 表格居中 | 左右差 ≤5% 页宽 | `pdf_qa.py:558` |
| 封面线-文字间距 | ≥1U = 5% 页宽 | `cover_validate.js:299` |
| HTML 字号下限 | 10px | `poster_validate.py:543` |
| HTML 对比度 | ≥3.0:1 | `poster_validate.py:564` |
| 色板对比度 | text:bg ≥4.5:1、accent:bg ≥2.5:1 | `design_engine.py:1269,1273` |
| 目录匹配率 | ≥0.5 | `toc_validate.py:620` |
| 目录漂移 | ≤30% | `toc_validate.py:987` |
| PDF 尺寸合理性 | ≥200px | `poster_validate.py:923` |

**轮次**：源码里**没有显式最大轮次**。唯一的"重试语义"是：
- `convert.latex` 有 `-r/--runs N`（默认 1，`$P/scripts/pdf.py:2111-2112`）——那是 LaTeX 交叉引用需要的多遍编译，不是质检轮次。
- `toc_validate.py fix_docx` + `fix_docx_accurate_pages` 是**两遍法**：先占位编译出真实页码，再回填（`:1772`）。
- 其余全部是"FAIL → 人为重做"，属 LLM 自觉循环。若 gaea 要工程化，必须自己定轮次（建议：结构类 error 最多 3 轮，视觉类 2 轮，超过则降级交付 + 报告缺陷）。

### 3.6 源码里必须知道的坑（移植前必读）

1. **`LAST_PAGE_MIN_FILL` 是死引用**：`$P/scripts/pdf_qa.py:54-55` 注释掉定义，`:118` 仍在使用 → 该函数一旦被调用即 `NameError`；而 `run_qa()`（`:776-795`）恰好没调用它，所以"看起来能跑"。
2. **`convert.html` / `convert.blueprint` 指向不存在的 `html2pdf.js`**：`$P/scripts/pdf.py:1882`、`:2088`，仓库里只有 `html2pdf-next.js`。
3. **Creative 画布写死 720×960**：`$P/scripts/pdf.py:2096`，与 `design_engine` 支持任意 canvas（`design_engine.py:2592-2593`）矛盾；海报等非 4:3 场景必须绕过 `convert.blueprint` 直接调 `html2poster.js`。
4. **Tectonic 内置二进制仅 macOS arm64**：`$P/scripts/pdf.py:1924-1928`，Windows/Linux 必须自行安装——gaea 主战场是 Windows，这条几乎等于"Academic 路线不可用"。
5. **`xlsx.py pivot` 名不副实**：`$P` 无关，但同理——`$X/xlsx.py:702-1059` 是 Python 聚合后写静态表，`$X/quality/pipeline.md:104` 却写"不要用 openpyxl 改 pivot 输出，会破坏 pivotCache"。**文档在保护一个并不存在的 pivotCache**。
6. **`pdf_qa.py` 门禁偏软**：除 `Blank pages` 外全是 WARN（`:70-77`），`FAIL` 实际只在全空页时出现。
7. **`typography.md` 只有 317 字节**：字号体系散落在 fill-engine / poster / resume，没有单一真源——搬的时候别指望这个文件。
8. **两次覆盖的 LibreOffice 宏**：`$X/xlsx.py:172-214` 与 gaea 现有 `.gaea/skills/xlsx/scripts/recalc.py`（`C:\AI\wubigrok\.gaea\skills\xlsx\scripts\recalc.py:29-37`）写的是同一个 `Module1.xba` / `RecalculateAndSave` —— **能力重复，不要重复移植**。
9. **brief 与 typesetting 漂移**：`$P/briefs/report.md:32` 写"7 cover layouts"，而 `$P/typesetting/cover.md:15` 已是 **01-09 共 9 套**（V4.0 重编号）。搬规则时必须先对齐一版，否则会按旧编号选错模板。
10. **Preflight 链条本身是可执行规格**：`$P/briefs/report.md:26` 把交付前步骤写成固定序 `code.sanitize → 执行 → meta.brand → font.check → toc.check → pages.clean → pdf_qa.py`（对应 `pdf.py:2971,988,2645,2683,801` 与 `pdf_qa.py` 主入口）。gaea 若做 PDF 交付门禁，这条链是最小可用顺序。

---

## 4. briefs：10 条最硬的摘录 + 映射到 gaea 的形态

> 映射目标分三类：**T=导出模板**（结构化版式）｜**P=prompt 规则**（生成期约束）｜**V=校验规则**（生成后判定）。

| # | 摘录（原文要点 + 证据） | 硬度 | 能变成 gaea 的什么 |
|---|---|---|---|
| 1 | **表格单元格一律 `Paragraph()`，无例外**；单位指数必须 `W/m<super>2</super>`；\|值\|≥10⁴ 或 ≤10⁻³ 用科学计数法（`$P/briefs/report.md:599-628`） | 极高（LLM 默认就写裸字符串，这是最高频崩点） | **V**：docx/PDF 表格后处理器检测裸字符串与 `m2`/`W/m2` 形态；**P**：生成约束一句 |
| 2 | **`sum(colWidths) ≤ available_width` 必须在代码里断言**，优先按比例 `[0.25,0.40,0.20,0.15]×available`（`$P/briefs/report.md:679-715`；同 `overflow.md:354-355`） | 极高 | **V**：文档渲染前自检；**T**：表格列宽策略 |
| 3 | **大节 3/4 阈值**：H1 前剩余 <25% 页高则强制换页（`$P/typesetting/pagination.md:139-196`） | 高（"看起来像意外"的排版病） | **P** + **V**（PDF 后检：H1 落在页面底部 25% 即报） |
| 4 | **页码五区制 + 禁止 "Page X of Y"**：封面逻辑 1 不显示、前置罗马、正文重置 1、附录续编（`pagination.md:247-275`，尤其 `:263`） | 高（公文/报告硬需求） | **T**：docx/pdf 页脚模板；**V**：末页/页脚格式检查 |
| 5 | **末页填充率 ≤25% 必须回溯**，回溯顺序：段距→行高(≥1.5)→字号(≥14px)→删冗余→并节（`pagination.md:17-35`） | 高 | **V**：已有 `pdf_qa` 类判据；**P**：回修策略 |
| 6 | **填充率四档 + 黄金比 0.382 锚定**（≥80/65-80/40-65/<40；行高上限 2.2、字号 +2pt 上限、段距注入 40%）（`fill-engine.md:86-90,97-161,230-257`） | 高（"半页空白"是自动化文档最常见丑相） | **P**（策略）+ **V**（阈值）；这是 gaea docx 导出可直接受益的规则 |
| 7 | **封面字重系统 + 数据绑定铁律 + 强制 Summary 2-4 行**：Hero=机构名 45-66pt Heavy，Kicker=类型 16pt +3pt 字距；反了就废；无摘要自动补（`$P/typesetting/cover.md:77-88`） | 高 | **T**：封面模板参数约束；**V**：字段映射检查 |
| 8 | **海报字号硬地板：正文 24px / 主标题 50px / 副标题 28px / Meta 16px / 数字 48px**（`$P/briefs/poster.md:111-122`） | 极高（海报=远读介质，报告字号直接搬会废） | **V**：HTML/PDF 字号扫描（已有 `TINY_FONT` 10px 太松，海报要 24px 档） |
| 9 | **反卡片化**：单页海报 `background+border` 容器 ≤3 个、`Glass_Canvas` ≤1 次、禁 2×2/2×3 卡片墙；层级靠字号/字重/透明度而非彩盒（`$P/briefs/poster.md:174-201`） | 极高（LLM 天然把一切装进卡片） | **V**：CSS 结构计数（容器带 bg+border 计数）；**P**：硬规则 |
| 10 | **海报行分配按字数比例，不许均分**：`rows_i = total_rows × chars_i/total_chars`，每块 ≥3 行，端点首尾相接覆盖 1→13（`$P/briefs/poster.md:72-95`） | 高（均分必溢出重叠） | **P**（生成时算）+ **V**（校验行区间首尾相接） |
| 11 | **简历：12px(9pt) 全篇硬地板 + ≥85% 页高填充**，不足则按序加段距(10→18pt)→行高(14→18pt)→字号(+0.5~1pt)→补摘要节→加边距(1.5→2cm)；溢出则反向（`$P/briefs/resume.md:21,33-41`） | 极高 | **T** + **V**（简历专项：字号地板 + 填充率） |
| 12 | **试卷六条 E1-E6**：题号与题干必须同段（禁空行，否则 `\parskip` 造出"空行"）→ `\parskip=0` → 优先 `enumitem` → `tasks` 多列 → 答题留白（填空 2cm/简答 2cm/计算 4cm/证明 5cm）→ 大题间可断、单题禁断（`$P/briefs/academic.md:776-848`） | 高（中文试卷刚需，且是"无人做对"的细分） | **T**：试卷模板；**P**：LaTeX/docx 生成规则；**V**：`\parskip` 与空行检测 |

（表里 12 条，前 10 条即"最硬的 10 条"，11-12 为附赠高价值项。）

---

## 5. gaea 现状（对照证据）

### 5.1 xlsx 侧：Go 原生编辑 + Plan→Apply（比 ZCode 更工程化）

| 能力 | 实现 | 证据 |
|---|---|---|
| 单元格编辑 | excelize 直写；`=` 开头走 `SetCellFormula`、空串清空、可解析为 float 则写数值 | `internal/app/gaea_xlsx_edit.go:230-248` |
| **Plan→Apply**（本次调研最该对标的设计） | `GaeaXlsxPlanEdit`：读上下文 → AI 出 ops → **临时副本试运行** → 与原文件 diff 出单元格级 `changes` 清单供用户批准（`:41-103`，`PlanOps` 在 `:88`）；批准后 `GaeaXlsxApplyEdit` 原样透传 ops 执行（`:107-165`） | `internal/app/gaea_xlsx_edit.go:30-36,41,88,105-165` |
| 操作前基线快照（回滚原料） | 应用前把整文件拷到 `.gaea/work/rollback/xlsx-<nano>.before` | `:131-140` |
| 证据链 | `appendXlsxEvidence` → `appendOfficeEvidence("xlsx_apply", ...)`，ops JSON 超限则不落（"宁漏勿误"） | `:167-184` |
| 行/列操作 | `InsertRows/RemoveRow/InsertCols/RemoveCol` + 同表公式/合并区由 excelize 平移 | `:287-424` |
| 公式重算 | **调 Python `recalc.py`**（LibreOffice 宏），`Recalc(path, root)`，找不到脚本则提示"跳过（未找到 recalc.py）" | `internal/office/xlsxedit/xlsxedit.go:485-516`；`gaea_xlsx_edit.go:148-155,275-279` |
| 原生图表 | excelize `AddChart`（Col/Line/Pie/Scatter），锚点自动放在数值列右侧两列，返回 mini 预览数据 | `internal/app/gaea_xlsx_chart.go:193-245`（图表构造 `:220-227`，锚点 `:228-232`） |
| 预览渲染 | `xlsxpreview.Render`（含条件格式解析 `internal/office/xlsxpreview/condfmt.go`） | `gaea_xlsx_edit.go:156-159` |
| 交付导出的 xlsx | **极简**：把 Markdown 里的表格抽成 sheet，无样式、无公式、无图表 | `internal/app/gaea_export.go:245-298` |
| 已有 office 校验 | `.gaea/skills/xlsx/scripts/office/validate.py`（**XSD 模式校验** + auto-repair），及 `recalc.py` | `.gaea/skills/xlsx/scripts/office/validate.py:1-19`；`.gaea/skills/xlsx/scripts/recalc.py:29-37` |

**结论**：gaea 的 xlsx **写入层比 ZCode 强**（Plan/Apply/证据/回滚），**样式层与质检层比 ZCode 弱**（无 design tokens、无 audit/scan、无 chart-verify、导出 xlsx 几乎裸奔）。ZCode 的 `design.md + base.py 令牌 + 12 色板` 正好补这一块。

### 5.2 PDF 侧：只有 LibreOffice 转换（能力真空）

| 能力 | 实现 | 证据 |
|---|---|---|
| 任意文档→PDF | `GaeaConvertToPdf`：md→docx→soffice；直转白名单 `.docx/.xlsx/.pptx/.odt/.html/.htm/.txt/.csv`，`.doc` 明确拒绝 | `internal/app/gaea_pdf.go:34-37,42-107,97-98` |
| soffice 调用 | 独立 `UserInstallation` profile 防锁冲突；`--headless --norestore --nolockcheck --nologo --convert-to pdf`；180s 超时；输出先落临时目录再搬 | `:152-183` |
| soffice 定位 | PATH 优先，再补 Windows `LOCALAPPDATA/Programs/LibreOffice/program/soffice.exe`、`ProgramFiles`、macOS、Linux 常见路径 | `:187-213` |
| 统一交付出口 | `GaeaExportDeliverable`：格式 docx/pptx/xlsx/md/pdf；模板仅 4 种（通用/公文/报告/合同）；**pdf = Markdown→docx→soffice** | `internal/app/gaea_export.go:21-31,45-120,102-107`；模板白名单 `:58-62` |
| PDF 生产/排版 | **无**。全仓 grep 无 `reportlab` / `pymupdf` / `fitz` / `pdfplumber` 依赖 | （grep `internal/**/*.go` 零命中） |
| 已有 .gaea pdf 技能 | 是**另一套**（表单填写/OCR/边界框），不是生产管线 | `.gaea/skills/pdf/SKILL.md`、`scripts/fill_fillable_fields.py`、`ocr_local.py` 等 |

**结论**：gaea 的 PDF = "Office 转出来什么样就是什么样"。没有任何版式能力、没有任何 PDF 质检。ZCode 的 `typesetting`（规则）+ `poster_validate`（文本级校验）能补规则，但**渲染引擎必须另选**（见 §6）。

---

## 6. 移植判定总表

> 成本口径：**S** ≤ 1 人日（拷贝/格式化）；**M** 2-5 人日（重写逻辑 + 测试）；**L** 1-3 周（新渲染栈/新依赖）；**XL** > 3 周（不建议）。

### 6.1 原样搬（copy，成本 S）

| # | 资产 | 搬到哪 | 成本 | 风险 | 备注 |
|---|---|---|---|---|---|
| A1 | `typesetting/overflow.md`、`pagination.md`、`fill-engine.md`、`geometry.md`、`cover.md`、`palette.md`、`charts.md`（≈100KB） | `.gaea/skills/pdf/typesetting/` + 摘要进 prompt | **S** | 低（纯文本）。**但必须加"适用引擎"标注**：文中的 ReportLab/Playwright/LaTeX 代码片段 gaea 都没有，只保留判据与阈值 | 这是全套资产里性价比最高的一搬 |
| A2 | `briefs` 的"铁律/自检清单"节选（§4 的 12 条 + `report.md:18-40` 工作流、`academic.md:489-510` preflight） | gaea prompt 资产（按模板分节检索，不要整篇 290KB 注入） | **S** | 低；风险在"全文注入会撑爆 context"，需切片 | 建议做成"规则卡"（每条 1-3 行 + 证据链接） |
| A3 | `templates/palettes.py` 的 12 套色板（结构化数据）+ `engines/design.md` 三色规则/字号层级/边框规则 | gaea 主题包（`shared/` 或 `design-system/` 下 JSON/YAML） | **S** | 低。但建议**只保留 6 套**（professional/warm/elegant/creative/bloomberg/original_blue），品牌向 6 套（muji/aesop/kinfolk/celine/bottega/chanel）与 gaea 设计系统冲突 | 色板是"数据"，不是"逻辑"，最易搬 |
| A4 | `quality/pipeline.md` 的 14 条陷阱表 + 10 条 Release 清单 + `base.py` 的 `normalize_cell_value`/`auto_fit_*` 语义 | gaea xlsx 技能的 prompt + 实现 checklist | **S** | 低 | 陷阱 12/14（openpyxl 缓存值、NBSP）在 Go 下表现不同，需改写措辞 |
| A5 | `poster_validate.py check-tex` 的 12 条 LaTeX 规则定义（纯文本描述） | 规则表（若 gaea 出 LaTeX 才启用） | **S** | 低 | 先存档，不投入实现 |
| A6 | §3.5 的阈值总表（末页 40/25%、大节 25%、字号地板 14/12pt、海报 24px、简历 12px、满版 5%、对称 5%、对比 3.0/4.5） | gaea 校验配置（一处定义，多处引用） | **S** | 低；风险是"照抄多套不一致的阈值"——必须统一成单表 | 现状阈值散在 5 个文件且互相打架 |

### 6.2 要改写（rewrite，成本 M/L）

| # | 资产 | 改写成什么 | 成本 | 风险 | 理由 |
|---|---|---|---|---|---|
| B1 | `xlsx.py audit + scan + validate`（`$X/xlsx.py:367-596,1161-1270`） | **Go 规则引擎**（excelize 读公式 + `archive/zip` 读 `.rels`）：错误串扫描、零值公式、隐式数组模式、越界/含表头/小聚合/公式模式不一致、禁用函数、文本伪公式、外部引用、绝对路径 | **M** | 中。① excelize 读不到"缓存值"，audit 的 error 检测需改为"重算后读值"（gaea 已有 Recalc，可串起来）② 共享公式/数组公式在 excelize 里的展开方式需实测 | 规则全是 regex+阈值，Go 化天然；且能直接挂到 `GaeaXlsxApplyEdit` 后置 |
| B2 | `xlsx.py recalc`（`$X/xlsx.py:172-338`） | **不重写，复用** gaea 现有 `xlsxedit.Recalc` + `recalc.py` | **0（复用）** | 低 | 两边宏体是同一个 `Module1.xba`/`RecalculateAndSave`，重复实现纯浪费 |
| B3 | `poster_validate.py check-html`（19 条，`$P/scripts/poster_validate.py:253-830`） | **Go 校验器**（`golang.org/x/net/html` + regex on CSS）：字体兜底、容器 overflow、@page、margin reset、对比度、字号、多页背景、固定高度 | **M** | 低-中。纯文本规则，无浏览器依赖；难点是 CSS 选择器/规则解析的完备性（可只做白名单规则 + 保守漏报） | 19 条规则全是"字符串/结构"判据，是**最值得 Go 化**的一套 |
| B4 | `design_engine.py` 的色板段（`:99-315` + `:362-592` + `:671-702` + `:1221-1276`） | **Go 色板服务**：intent 关键词→基色相；HSL 锁；cascade 角色分层（面积∝1/饱和度）；WCAG 对比度审计 | **M** | 低。纯算术（HSL↔RGB、相对亮度、对比度）；无外部依赖 | 一套色板可同时服务 png/docx/pptx/xlsx/pdf 五个出口；比 ZCode 现在的"每管线一套色板"更优 |
| B5 | `design_engine.py` 的 SVG 生成（`:715-1119`） | Go 版 SVG 生成器（flow/grid/noise/supergraphic） | **M** | 低-中。数学简单，但"好看"需调参；建议先只做 grid + flow 两种 | 与 B4 同栈，边际成本低 |
| B6 | `toc_validate.py check-docx`（`:503-680`） | Go 读 docx（`archive/zip` + `encoding/xml`）校验 TOC 域/占位/标题样式/匹配率/分页 | **M** | 中。`is_heading_style`/`fuzzy_match` 的阈值需重标；gaea 的 `create_docx.py`（python-docx）是否产出 TOC 域需先核对 | gaea 已能出 docx（`create_docx.py`）且有 `--toc` 开关（`gaea_export.go:138-140`），这个校验器正好补门禁 |
| B7 | `pdf_qa.py` 全套（899 行） | **两条路**：① Python 侧车（gaea 已有 `runPython` 通道 + `findSkillScript` 定位，`gaea_export.go:300-322`）② Go 重写（pdfcpu/unipdf，能力不足） | **S（侧车）/ L（Go 重写）** | 中-高。侧车风险：用户机需 Python + PyMuPDF（gaea 已有 python 依赖链：`runPython`、`recalc.py`，边际可接受）；Go 重写风险：PDF 内容 bbox/字体/颜色/矢量聚类在 Go 生态很弱 | **推荐侧车**：先把 `pdf_qa.py` 当外部校验器调用，JSON 解析进 gaea 的 Verifier；不要 Go 重写 |
| B8 | `cover_validate.js`（367 行） | 若要复用必须 Node+Playwright+Chromium | **M** | 高（见 B10） | 依赖 Chromium，收益仅在"HTML 封面"路线；gaea 若不做 HTML 渲染则整块跳过 |
| B9 | 封面引擎 `cover_render.py`（487 行 ReportLab，9 模板） | Go PDF 绘图（gofpdf/unipdf）重写坐标与字重系统 | **L** | 高。① ReportLab 只能嵌 TrueType 的约束在 Go 侧变成"字体度量+子集化"自研 ② faux-bold 叠印、`U=W*0.05` 栅格、水印不裁切都要重算 ③ 9 套模板 × 3 类 content schema 的视觉验收成本高 | 若 gaea 的封面可以走 docx（`create_docx.py --cover`，`gaea_export.go:135-137`），**就不该做这个**。仅当"PDF 封面必须独立于 Word 模板"时才值得 |
| B10 | Creative 全链（design_engine compile → html2pdf-next.js → check-html 闭环） | 打包 Chromium（Playwright 或 CDP）+ 保留 Node 脚本 | **L** | **高**。① Chromium 体积 100MB+，Windows 离线安装包会显著变大 ② `html2pdf-next.js` 依赖 playwright + pdf-lib 两个 npm 包 ③ 字体/中文渲染需额外配置 ④ 版本漂移（Playwright 与 Chromium 绑定） | 只有"海报/信息图/证书"是 gaea 真需求时才做；否则不做 |
| B11 | Academic 全链（Tectonic） | 打包 tectonic（Windows 版）或改走 MiKTeX/TeXLive 检测 | **L** | 高。① 内置二进制只有 macOS arm64（`$P/scripts/pdf.py:1924-1928`）② 首次运行要联网拉 TeX 包（离线场景废） ③ Windows 用户装 TeX 发行版的比例极低 | **不建议**。数学重文档可退化为"docx + OMML 公式"，或明确"不支持" |

### 6.3 不该搬（拒绝，理由要写进决策记录）

| # | 资产 | 为什么不该搬 |
|---|---|---|
| C1 | `.zcode-plugin/plugin.json`、MCP server `dist/server.mjs`、UI surface（`ui://.../panel.html`）、`skills/xlsx-viewer`、`skills/pdf-viewer` | **宿主耦合**。这是 ZCode 插件商店 + 侧栏面板机制（`clones/zcode/apps/zcode-cli/packages/pdf-plugin/.zcode-plugin/plugin.json:16-25` MCP 段、`:26-40` UI surface 段、`:36` `resourceUri`），gaea 是 Wails 单进程，没有 MCP 宿主与 `ZCODE_PLUGIN_ROOT`/`ZCODE_PROJECT_DIR` 注入（同文件 `:19-23`）。搬过来只会得到一堆死配置 |
| C2 | `agents/visual-judge.md` | ① 模型写死 `account:bigmodel-individual-coding-plan/GLM-5.3-Flash`、`thoughtLevel: max`（`:5-6`）② 绑定 ZCode 的任务协议（"dispatch grouping / delivery protocol" `:3`）③ **gaea 已有等价物**：`internal/app/gaea_verify.go`、`gaea_verify_diff.go`、`gaea_verify_channel_b_test.go` + `evidence` 包。再引入一套视觉 agent 会造成双门禁 |
| C3 | `env_setup/*`：`env_check.sh`、`setup_windows.ps1`（10KB）、`setup_mac_linux.sh`（13.7KB）、`setup.md`（16.3KB）、`font_list.txt` | ① gaea 已有 soffice 定位与 profile 隔离（`gaea_pdf.go:152-213`）、`recalc.py` 的 `office/soffice.py` ② `setup.md` 的主体是"LibreOffice 必须装、不许用 Excel/WPS"的**产品政策话术**（`$X/SKILL.md:233-279`、`$P/SKILL.md:698-715`），不是能力实现；gaea 的 UX 与合规口径不同，照搬会变成错误提示文案 |
| C4 | VBA 全套：`scenes/vba.md`（8.2KB）、`engines/vba-templates.md`（13.8KB）、`vbaProject.bin` 注入 | ① **安全/合规**：gaea 不提供宏执行沙箱，也不该鼓励用户开宏 ② **技术不可靠**：`$X/scenes/vba.md:227` 自己写"fragile — Method 1/2 preferred" ③ **平台不匹配**：excelize 对 `vbaProject.bin` 的支持与 openpyxl `keep_vba` 不同，Windows 下更易产生"Excel 提示修复"的坏文件 ④ 若真有需求，正确做法是生成"操作说明 + Office Script"，不是注入宏 |
| C5 | `xlsx.py pivot`（`$X/xlsx.py:702-1059`） | ① **命名欺诈**：产出是 Python 聚合后的静态表，不是 OOXML PivotTable；② **自相矛盾**：`$X/quality/pipeline.md:104` 说"绝不要用 openpyxl 改 pivot 输出（会毁 pivotCache）"，而这个命令本身就是用 openpyxl 写表；③ gaea 若要汇总，用 excelize 聚合 + 自己的样式层更可控；④ 真要做交互透视表，excelize 也不支持创建，需直接写 pivotCache XML（成本 XL，收益低） |
| C6 | 品牌向 6 套色板（muji/aesop/kinfolk/celine/bottega/chanel，`$X/engines/design.md:81-86`） | 与 gaea 现有设计系统（`design-system/`、`DESIGN.md` 口径）冲突；这些是"小红书风格"审美，不是办公交付需求。保留 6 套通用/金融向即可 |
| C7 | `check-html --fix` 的自动改写 HTML 功能（`$P/scripts/poster_validate.py:186-188,811-825`） | 对 LLM 生成的 HTML 做**正则级**自动修补（补 font fallback、删 overflow:hidden）风险不对称：可能改坏布局而无人察觉。gaea 若做校验，应"报错让人/模型改"，不做文本手术 |
| C8 | "Page X of Y 禁止"之外的 ZCode 引用规范（`::zcode-file-citation{...}`，`$X/SKILL.md:305-349`、`$P/SKILL.md:958-1004`） | 纯宿主协议（ZCode 的文件引用语法），gaea 有自己的引用/证据卡体系（`appendOfficeEvidence`）。**但"正文内联引用而非文末列表"这条交互原则可以借鉴** |

### 6.4 一张图看判定

```
ZCode xlsx ──┬─ 规则/令牌/色板/陷阱表 ────────────────► 原样搬 (S)   [A3,A4]
             ├─ audit/scan/validate 规则 ──────────────► Go 重写 (M)  [B1]
             ├─ recalc ───────────────────────────────► 复用现有 (0) [B2]
             ├─ design.md 版式语义 ───────────────────► 搬规则+Go 实现 (M) [B6 类]
             ├─ chart 选型表 ─────────────────────────► Go 规则 (S/M)
             └─ VBA / pivot / env_setup / plugin.json ─► 不搬        [C1,C3,C4,C5]

ZCode pdf  ──┬─ typesetting/*.md (100KB) ──────────────► 原样搬 (S)   [A1]
             ├─ briefs 铁律节选 ──────────────────────► prompt 卡 (S) [A2]
             ├─ poster_validate check-html/tex 规则 ───► Go 重写 (M)  [B3]
             ├─ design_engine 色板+cascade+WCAG ───────► Go 重写 (M)  [B4]
             ├─ pdf_qa.py (PyMuPDF) ──────────────────► Python 侧车 (S) [B7]
             ├─ toc_validate check-docx ──────────────► Go 重写 (M)  [B6]
             ├─ cover_render.py (ReportLab 9 模板) ────► 视需求 (L)   [B9]
             ├─ Creative 链 (Chromium) ───────────────► 视需求 (L)   [B10]
             ├─ Academic 链 (Tectonic) ───────────────► 不建议 (L)   [B11]
             └─ plugin.json / visual-judge / env_setup ► 不搬        [C1,C2,C3]
```

---

## 7. 建议落地顺序（按性价比）

1. **第 1 步（S，1 天）**：把 `typesetting/` 7 篇 + `design.md` 三色/字号/边框规则 + §3.5 阈值统一表落进 gaea（`.gaea/skills/pdf/typesetting/` + 一张 `thresholds.yaml`）。这一步不改一行 Go 代码，但立刻能改 prompt 质量。
2. **第 2 步（S，1 天）**：12 条 briefs 铁律做成"规则卡"（每条含判据、阈值、反例），进 gaea 导出/写作 prompt。
3. **第 3 步（M，3 天）**：Go 写 `xlsxqa`（B1），挂到 `GaeaXlsxApplyEdit` 之后 + `GaeaExportDeliverable(xlsx)` 之后；复用现有 `Recalc`。收益立刻可见：错误公式/文本伪公式/绝对路径在交付前被拦住。
4. **第 4 步（M，3 天）**：Go 写 `htmlcheck`（B3 的 19 条），先只服务 gaea 现有 HTML→PDF/deliverable 路径（若已在用 `soffice` 转 html，也有价值：渲染前先查字号/对比度/overflow）。
5. **第 5 步（M，3-5 天）**：Go 写色板服务（B4），替换"各处硬编码颜色"，让 docx/pptx/xlsx/pdf 共用一套主题。
6. **第 6 步（S，1 天）**：把 `pdf_qa.py` 作为 Python 侧车接进 Verifier（B7），只解析 JSON、只出报告，不改文件。
7. **暂不做**：封面引擎（B9）、Chromium 全链（B10）、Tectonic（B11）、VBA（C4）。这四项都是"看起来最炫、维护成本最高"的部分。

---

## 8. 方法与限制

- **只读**：本报告未修改 `clones/zcode` 与 gaea 任何代码；仅新增本文件。
- **未运行**：受 Python/Node/PyMuPDF/Chromium 环境与"不改代码"约束，所有脚本结论均来自静态阅读（含 `ast.parse` 语法确认），未实际执行 `pdf_qa.py`/`poster_validate.py`。§3.6 的两个缺陷（`LAST_PAGE_MIN_FILL` 死引用、`html2pdf.js` 缺文件）是通过"定义处 vs 引用处"的静态比对得出的，**未做运行时验证**。
- **证据基准**：行号基于当前 clone 检出（`clones/zcode` 工作树）。若上游更新，行号需重校。
- **未覆盖**：`briefs/creative*.md`（98KB）只做了路由与规则抽样，未逐条拆；`process.md`/`process-advanced.md` 只做了能力清单级阅读；`html2pdf-next.js` 的 `preRenderHooks`（326 行）只确认了存在与职责，未逐行分析其修正逻辑——若 gaea 决定走 HTML 路线，这部分需要单独一轮深挖。
