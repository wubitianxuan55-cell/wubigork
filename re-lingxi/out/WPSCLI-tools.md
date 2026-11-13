# wpscli.exe 工具面（静态抽取 · 续批）

- 二进制: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe`（8.0 MB，Rust + rmcp）
- 抽取方式: 二进制内串联字符串 `[描述][工具名]` 成对切段（静态，未执行目标）
- 动态校验: `wpscli.exe --help` → `错误：不支持非金山应用调用本程序。`（`src\parent_verify.rs`，exit=1） → **tools/list 拿不到，schema 只能静态还原**
- 语义工具 **34** 个（首批按 30 记，本批实测 34）；legacy 逃生舱 3 个；分组名 0 个

## 工具清单与描述原文

| 工具 | 类别 | 描述（原文节选） |
|---|---|---|
| `wps_browser` | 通用/其它 | Browser automation via Chrome DevTools Protocol (CDP). Commands include 'screenshot', 'navigate', 'evaluate'. Requires a running browser with remote debugging. |
| `wps_doc_check` | Writer/docx | Proofread and check a Word document (.docx) for spelling errors, formatting issues, and document integrity. |
| `wps_doc_compare` | Writer/docx | Compare two Word documents (.docx) and return their differences (insertions, deletions, formatting changes). |
| `wps_doc_create` | Writer/docx | Create a new Word document (.docx). Provide content via 'from_text' (plain text) or 'data' (JSON for template). Exactly one source required. |
| `wps_doc_edit` | Writer/docx | Edit Word document (.docx). Operations: find/replace (find+replace), append, insert text or a local image at target.char / target.paragraph. Do not pass target.range for insert; that is a selection. Changes are saved automatically. |
| `wps_doc_export` | Writer/docx | Export Word document (.docx) to another format. Set 'format' to: 'txt' (plain text), 'html' (web page), 'md'/'markdown' (via advanced JSAPI exportDocumentAsMarkdown), or 'png'/'image' (one PNG per page, native Pages.SaveAsPNG). For png, 'output' is a directory (defaults to a folder named after the document) and the opt… |
| `wps_doc_fill` | Writer/docx | Fill a Word document template (.docx) with JSON data. Template should contain {{placeholder}} markers that match keys in the JSON data. |
| `wps_doc_read` | Writer/docx | Read Word document (.docx) content. Set 'what' to: 'full', 'outline', 'metadata', 'tables', 'paragraph'. Use target.find to search all occurrences and get character range targets for later insert. |
| `wps_doc_script` | Writer/docx | Run a custom WPS Writer JSAPI script on a document. Provide code via 'script' (inline) or 'script_file' (path to .js). Set 'ui' to true for headed mode. |
| `wps_file_info` | 通用/其它 | Get file metadata: size, type, extension, creation/modification dates. Works with any file format. |
| `wps_image_extract` | 通用/其它 | Extract text from an image file using OCR (optical character recognition). Supports png, jpg, bmp formats. |
| `wps_pdf_annotate` | PDF | Add annotations to a PDF. Highlight a character range with 'page'+'start'+'end' (offsets from pdf search results), or add a FreeText note with 'text'+'x'+'y'+'page' (PDF coords: origin at bottom-left). |
| `wps_pdf_edit` | PDF | PDF edit operations: 'merge' (path to other PDF), 'split' (pages per output file, integer > 0, requires output directory via 'output'), 'watermark' (overlay text), 'bookmark'+'bookmark_page' (add bookmark). Set 'output' for result file or split directory. |
| `wps_pdf_export` | PDF | Export a PDF. Set 'format' to 'png' (default: each page to PNG in output directory), 'txt' (plain text via Exports.ExportToText), or 'md'/'markdown' (via Exports.ExportToMarkdown). For markdown, optional start_page/end_page are 1-based inclusive (omit or 0/0 for all pages). |
| `wps_pdf_extract` | PDF | Extract page ranges from a PDF into a new PDF file. Set 'pages' (e.g. "1-3,5") and 'output' (destination .pdf path). |
| `wps_pdf_form` | PDF | Read or fill PDF form fields. Omit 'field' to list all form fields. Set 'field'+'value' to fill a specific field. |
| `wps_pdf_read` | PDF | Read PDF content. Set 'what' to: 'full' (all text), 'page' (specific page, set target), 'metadata' (author/dates/pages), 'bookmarks' (table of contents), 'fields' (form fields). |
| `wps_pdf_script` | PDF | Run a custom WPS PDF JSAPI script. Provide code via 'script' (inline) or 'script_file' (path to .js). Set 'ui' to true for headed mode. |
| `wps_pdf_search` | PDF | Search for text in a PDF document. Returns count and results with page (1-based), start/end character offsets, and matched text. |
| `wps_ppt_create` | Presentation/pptx | Create a new presentation (.pptx). Set 'title' and 'subtitle' for a title slide. For multi-slide creation, use doc create with outline data. |
| `wps_ppt_edit` | Presentation/pptx | Edit presentation (.pptx). Set slide+shape+text or slide+placeholder+text to change text, or slide+image+pos [left,top,width,height] in cm to insert a local picture. pos is required for images. |
| `wps_ppt_export` | Presentation/pptx | Export presentation (.pptx) to another format. Set 'format' to: 'pdf', 'html', 'png' (single slide image, requires 'slide' number), or 'md'/'markdown' (via Presentation.ConvertToMarkdown). |
| `wps_ppt_read` | Presentation/pptx | Read presentation (.pptx) content. Use 'target' to specify: {"slide": 1} (single slide), {"slide": 1, "shapes": true} (shapes), {"slide": 1, "notes": true} (speaker notes). Omit target for all slides. |
| `wps_ppt_script` | Presentation/pptx | Run a custom WPS Presentation JSAPI script. Provide code via 'script' (inline) or 'script_file' (path to .js). Set 'ui' to true for headed mode. |
| `wps_ppt_watermark` | Presentation/pptx | Add or remove text watermark on presentation (.pptx) slide master. 'layout': 'tiled' (default, grid of rotated text), 'centered' (single centered box), or 'remove' (delete existing watermark). 'text': watermark content (default: 'åé¨æä»¶ è¯·å¿å¤ä¼ ', ignored when layout='remove'). 'color': hex color (e.g. '#FF00… |
| `wps_search` | 通用/其它 | Search for files on disk by name or content keyword. Returns matching file paths up to 'max' results. @ |
| `wps_sheet_chart` | Spreadsheet/xlsx | Create a chart in spreadsheet (.xlsx). Specify 'range' for data source (e.g. "A1:B10") and 'chart_type' (e.g. "xlColumnClustered", "xlLine", "xlPie"). |
| `wps_sheet_clean` | Spreadsheet/xlsx | Clean and normalize spreadsheet (.xlsx) data. Set 'operations' as JSON array: ["trim"] (whitespace), ["dedup"] (duplicates), ["remove_empty"] (blank rows). |
| `wps_sheet_create` | Spreadsheet/xlsx | Create a new spreadsheet (.xlsx) from data. Provide 'from_json' (JSON array of arrays, e.g. [["Name","Age"],["Alice",30]]) or 'from_csv' (path to CSV file). |
| `wps_sheet_edit` | Spreadsheet/xlsx | Edit spreadsheet (.xlsx) cells. Set 'range' (e.g. "A1"), then 'value' for data or 'formula' (e.g. "=SUM(A1:A10)"). Optional 'sheet' to target a specific sheet. |
| `wps_sheet_export` | Spreadsheet/xlsx | Export spreadsheet (.xlsx) to another format. Set 'format' to: 'csv' (comma-separated values), 'png' (image of sheet range), or 'md'/'markdown' (via advanced JSAPI exportSheetData). |
| `wps_sheet_read` | Spreadsheet/xlsx | Read spreadsheet (.xlsx) data. Use 'target' to specify: {"range": "A1:C10"}, {"what": "used_range"}, {"what": "charts"}, {"what": "all_sheets"}, {"what": "formula", "range": "A1"}. |
| `wps_sheet_script` | Spreadsheet/xlsx | Run a custom WPS Spreadsheet JSAPI script. Provide code via 'script' (inline) or 'script_file' (path to .js). Set 'ui' to true for headed mode. |
| `wps_status` | 通用/其它 | Check WPS Office installation, engine connectivity, file version (> 12.1.0.27880), and API authorization. Same report as CLI doctor. |

## legacy 逃生舱（任意 JSAPI 脚本）

| 工具 | 描述原文 |
|---|---|
| `wps_presentation` | [Legacy] Run raw JSAPI on presentation. Prefer wps_ppt_* tools instead. Provide 'script' or 'script_file'. |
| `wps_spreadsheet` | [Legacy] Run raw JSAPI on spreadsheet. Prefer wps_sheet_* tools instead. Provide 'script' or 'script_file'. |
| `wps_writer` | ÐpW@ K either 'script' (inline) or 'script_file' (path to .js file) must be provided failed to read script file 'À ': À xlColumnClustered ÐpW@ % descriptionPath to the document file to operate onOutput file path (optional)Inline JSAPI script code to executePath to a .js script file to executeRun with WPS UI visible (he |

## 参数词汇（工具间复用，来自 schema 字段串）

`file` 目标文档绝对路径 · `output` 输出文件/目录 · `script`/`script_file` JSAPI 逃生舱 · `ui` headed 真界面（默认 headless） · `format` 导出格式 · `what` 读取粒度（full/page/metadata/bookmarks/fields/tables/outline） · `target` 定位（find/paragraph/range/slide/shape/placeholder） · `data`/`from_text`/`from_json`/`from_csv` 创建与填充 · `range`/`chart_type`/`operations` 表格。

## 与 gaea 对照

- 覆盖面 ≈ gaea `office_*`（word 21 / excel 24 / ppt 19 类 op）。灵犀**只多**：`wps_doc_check`（校对）、`wps_file_info`、`wps_search`、`wps_image_extract`（OCR）。
- 差异不在 op 数量，在**通道形态**：MCP 协议 + `script/script_file` 逃生舱 + `ui` 开关（headless↔headed 同一工具双模）。
- 可照抄的最小形态：任何高层工具都配 `script` + `script_file` 双入参与 `ui` 布尔，模型不必因缺 op 而放弃。