# 灵犀办公技能能力清单（蒸馏用）

来源：`%APPDATA%\WPS 灵犀\app_server_5.24.0\skills` 与 `builtin_skill`（License: MIT, Copyright (c) 2026 灵犀）
证据：argparse 静态解析（`add_parser` 行号）+ SKILL.md 正文。

---

## 1. docx — `skills/docx/scripts/docx_cli.py`（10 子命令）

| 子命令 | 关键参数 | 作用 |
|---|---|---|
| `create` | `-o --cwd --ref REMOTE=LOCAL --asset-manifest` | JS 代码 → 远程 office-transcoder → DOCX（`POST lingxi.wps.cn/api/transcoder/docx/create`） |
| `lint` | `--partial` | 生成前 JS 静态校验 |
| `inspect` | — | DOCX 完整性（refs/images/headings） |
| `source-status` | — | JS 源 revision 状态 |
| `source-patch` | `--ops-file`（sha256 CAS） | 定点原子补丁 |
| `source-undo` | — | 回滚上一补丁 |
| `edit` | `--replace OLD=NEW`（可重复）、`--regex`、`--ignore-case`、`--parts GLOB` | 本地文本替换 |
| `unpack` / `pack` | — | OOXML 部件解/打包 |
| `restore` | — | 沙箱内 unsupported |

配套：`docx_lint.py`、`inspect_docx.py`、`edit_docx.py`、`office/{unpack,pack,validate,_shared}.py`、`docx-helper`（远程，`h.p/h.h1/h.h2/h.toc/h.spacer/h.headerFooter/h.build`）。

## 2. xlsx — `xlsx_cli.py`(3) + `lbo_cli.py`(3)

| 命令 | 参数 | 作用 |
|---|---|---|
| `xlsx_cli build` | `<script.py>`（REMAINDER 透传） | 执行构建脚本 |
| `xlsx_cli recalc` | `--timeout 30`、`--dynamic-array encode\|materialize\|preserve` | `formulas` 纯 Python 重算 + 注入缓存值 + 全表错误扫描 |
| `xlsx_cli audit` | — | 公式结构审计 |
| `lbo_cli render` | `--outdir preview --options-file --scale --sheets --page-mode single\|chunk --chunk-rows --features --json-path --timeout 180` | 原生渲染 PNG（需先 recalc） |
| `lbo_cli capabilities` / `path` | — | 能力探测 / 解析二进制路径 |

模块：`recalc.py`、`audit.py`、`dynamic_array.py`、`pivot_ooxml.py`（自写 pivotCache/pivotTable）、`chart_labels.py`（`add_scatter_xy_series` / `finalize_scatter_chart`）。

## 3. pptx — `pptx_cli.py`（28 子命令）

**读取**：`query`、`query-shape`、`get-fonts --category chinese|latin`、`get-allowed-fonts`、`list-templates`
**编辑**：`edit-text --operations JSON`、`copy-shape`、`delete-shape`、`move-shape`、`resize-shape`、`slide-organize --operations JSON`、`export-image`、`replace-image`
**HTML 路线**：`check-layout --mode static|browser|full`、`html-to-pptx`、`screenshot`(`--viewport 1280x720 --visual-ready-timeout 8 --no-guide`)、`check-visual`、`cancel-task`
**模板复刻**：`flatten` → `shell-report --min-pages` → `make-skeleton --pages 1,5,5,27` → `apply-flat build` → `hybrid-manifest --ppt-dir` → `assemble -o` → `pack --check-only`
**其它**：`extract-style --slides`、`export-screenshots --dpi 96|150|300`、`image-grid`（≤9 图，3×3 上限）

模块：`ooxml_flatten.py`、`hybrid_assemble.py`、`theme_registry.py`、`font_intersection.py`、`pptx_merge/{merge,opc,validate}.py`、`transcoder_api.py`、`theme_request_builder.py`。

## 4. pdf — `pdf_cli.py`（17 子命令）

`info`、`page-count`、`extract-text --pages --fast`、`extract-table --page --format json|csv`、`to-md`（OCR）、`merge`、`split`、`delete-pages`、`rotate --angle 90|180|270`、`crop --bbox x0,y0,x1,y1`、`watermark --watermark 水印.pdf`、`compress`、`encrypt/decrypt --password`、`render --dpi 150`、`extract-images`、`batch --pattern "*.pdf" <子命令>`。

## 5. wps-docs — `wps_docs_cli.py`（domain: drive 17 / dbsheet 21 / ap 7）+ ET

**drive**：`upload-mindmap`、`get-file-info`、`download-file`、`upload-file`、`move-file`、`create-folder`、`find-folder-by-path`、`list-folder-files(--order-by ctime|mtime|dtime|fname|fsize)`、`list-latest-items`、`search-files(--type file_name|content|all)`、`rename-file`、`copy-file`、`save-as-file`、`list-my-files`、`list-deleted-files`、`restore-deleted-file`、`delete-file`
**dbsheet**：`create-file`、`get-schema`、`create/update/delete-fields`、`create/update/delete-records(+batch)`、`create/update/delete-sheet`、`create/update/delete-view`、`list-records(--criterion 字段:Op:值 --filter-mode AND|OR)`、`get-record`、`search-records`、`get/update-form-meta`、`get-view`
**ap**（智能文档 .otl）：`download-otl`、`block-query`、`upload-image`、`create-doc`、`block-insert(--index --block-id)`、`block-delete(--start-index --end-index)`、`block-update(--ops-file 或扁平 flag)`
**ET**（在线表格 .xlsx/.ksheet）：不走 CLI，走 `python_cell_exec` → `et/scripts/et.py` + `agent_mode/{range,sheet,formatting,hyperlink,pivot_table,workbook,summary,skills_registry}.py`

## 6. wps-collab — `wps_collab.py`（Python SDK 形态）

模块：聊天（`im_recent_chats` / `im_get_history` / 发消息）、日程、会议（含纪要）、邮件、身份、知识库。
统一返回 `{success, data, message}`，时间参数 ISO8601。

## 7. 其它技能

| 技能 | 脚本/资产 |
|---|---|
| `browser` | `browser.py`、`cdp_http.py`、`core.py`、`actions.js`、`extensions/snapshot.py`、3 个测试脚本 |
| `deep-research` | `assets/{evidence-bank,kg,og}-template.md` + `references/{deliverable-routing,final-gate,gap-detection,html-report,source-priority}.md` |
| `paper-writer` | `academic_search.py`、`scientific_visualization.py`、中英文论文参考 + `docx-api.md` |
| `image-canvas` | `cli.py`、`image_edit_client.py`、`image_prepare.py` |
| `stock-analysis` | SKILL.md（无脚本） |
| `builtin/skill-creator` | `init_skill.py`、`package_skill.py`、`quick_validate.py` |
| `builtin/speech-to-text` | `asr_client.py`、`cli.py` |
| `builtin/visualizer-design` | 3 模式（ECharts / HTML-SVG / 图片叠图）+ `visualization-plan.schema.json` + 金标准示例 |
| `builtin/web-builder` | `vite_build.mjs`、`staticize-react.mjs`、`web_builder_cli.py`、`references/local-service.md` |
| `builtin/wps-cowrite` | `wps_docx.py`、`kdocs_cli.py`、`canvas_edit_hook.py`、`doc_save_hook.py`、`skill_directive.py` |

## 8. Python 运行时依赖（`service/requirements.txt`）

文件操作 `send2trash`；HTTP `requests/aiohttp/aiohttp-socks`；
Office `python-docx / python-pptx / openpyxl / lxml / pillow / cairosvg / matplotlib / plotly / kaleido / fonttools / beautifulsoup4 / Jinja2 / html5lib`；
浏览器 `websockets / httpx / playwright==1.58.0 / nodriver==0.48.1 / cdp-use==1.4.5`；
表格 `formulas / xlrd / numpy / pandas / scikit-learn`；
PDF `pypdf / pdfplumber / reportlab / pypdfium2 / pytesseract / PyPDF2`；可选 `rdkit`。

## 9. 关键工程约束（蒸馏时优先保留）

1. **公式优先**：结果必须是表格公式，Python 只做写入；否则源数据变化无法重算。
2. **零公式错误**：交付前必须 `recalc` + 扫描 5 类错误值。
3. **动态数组**：尺寸可变的结果用 FILTER/UNIQUE/SORT/GROUPBY/XLOOKUP，溢出区保持空。
4. **视觉验收**：生成后必须渲染 PNG 人眼/模型复核（docx/xlsx/pptx 都有 render 指南）。
5. **字体白名单 ∩ 系统字体**，空交集回退微软雅黑。
6. **入口互斥**：新建 / 模板复刻 / 局部编辑三条路线不得混用。
7. **禁止浏览器操作云文档**，只走 API；权限错误即终止，不做变通。
8. **失败即停**：`success:false` 或 exit≠0 必须终止后续写操作。
9. **目录纪律**：`<task>/scripts/build`、`<task>/scripts/test`、产物在任务目录根部。
10. **按需读取指南**：禁止任务开始前通读全部 reference。
