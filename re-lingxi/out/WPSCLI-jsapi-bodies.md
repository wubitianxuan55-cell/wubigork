# wpscli.exe 内嵌 JSAPI 算子片段（静态抽取）

- 二进制: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe`
- 命中含 WPS 对象模型的 JS 文本片段 **129** 条（去重后）
- 用途：看清每个 op 真正怎么驱动 WPS JSAPI（选区/范围/形状/导出），供 gaea COM 侧对齐语义

### 1. offset `5656768` · 143 字符

```javascript
>(function(){ try { return JSON.stringify({ok:true, text: OCR('�N')}); } catch(e) { return JSON.stringify({ok:false, error: e.message}); } })()
```

### 2. offset `5657152` · 4000 字符

```javascript
wps_searchCheck WPS Office installation, engine connectivity, file version (> 12.1.0.27880), and API authorization. Same report as CLI doctor.wps_status[Legacy] Run raw JSAPI on document. Prefer wps_doc_* tools instead. Provide 'script' or 'script_file'.wps_writerBrowser automation via Chrome DevTools Protocol (CDP). Commands include 'screenshot', 'navigate', 'evaluate'. Requires a running browser with remote debugging.wps_browserEdit Word document (.docx). Operations: find/replace (find+replace), append, insert text or a local image at target.char / target.paragraph. Do not pass target.range for insert; that is a selection. Changes are saved automatically.wps_doc_editFill a Word document template (.docx) with JSON data. Template should contain {{placeholder}} markers that match keys in the JSON data.wps_doc_fillRead Word document (.docx) content. Set 'what' to: 'full', 'outline', 'metadata', 'tables', 'paragraph'. Use target.find to search all occurrences and get character range targets for later insert.wps_doc_readPDF edit operations: 'merge' (path to other PDF), 'split' (pages per output file, integer > 0, requires output directory via 'output'), 'watermark' (overlay text), 'bookmark'+'bookmark_page' (add bookmark). Set 'output' for result file or split directory.wps_pdf_editRead or fill PDF form fields. Omit 'field' to list all form fields. Set 'field'+'value' to fill a specific field.wps_pdf_formRead PDF content. Set 'what' to: 'full' (all text), 'page' (specific page, set target), 'metadata' (author/dates/pages), 'bookmarks' (table of contents), 'fields' (form fields).wps_pdf_readEdit presentation (.pptx). Set slide+shape+text or slide+placeholder+text to change text, or slide+image+pos [left,top,width,height] in cm to insert a local picture. pos is required for images.wps_ppt_editRead presentation (.pptx) content. Use 'target' to specify: {"slide": 1} (single slide), {"slide": 1, "shapes": true} (shapes), {"slide": 1, "notes": true} (speaker notes). Omit target for all slides.wps_ppt_readProofread and check a Word document (.docx) for spelling errors, formatting issues, and document integrity.wps_doc_checkGet file metadata: size, type, extension, creation/modification dates. Works with any file format.wps_file_infoCreate a new Word document (.docx). Provide content via 'from_text' (plain text) or 'data' (JSON for template). Exactly one source required.wps_doc_createExport Word document (.docx) to another format. Set 'format' to: 'txt' (plain text), 'html' (web page), 'md'/'markdown' (via advanced JSAPI exportDocumentAsMarkdown), or 'png'/'image' (one PNG per page, native Pages.SaveAsPNG). For png, 'output' is a directory (defaults to a folder named after the document) and the optional 'page' field selects a single 1-based page; omit 'page' to export every page.wps_doc_exportRun a custom WPS Writer JSAPI script on a document. Provide code via 'script' (inline) or 'script_file' (path to .js). Set 'ui' to true for headed mode.wps_doc_scriptExport a PDF. Set 'format' to 'png' (default: each page to PNG in output directory), 'txt' (plain text via Exports.ExportToText), or 'md'/'markdown' (via Exports.ExportToMarkdown). For markdown, optional start_page/end_page are 1-based inclusive (omit or 0/0 for all pages).wps_pdf_exportRun a custom WPS PDF JSAPI script. Provide code via 'script' (inline) or 'script_file' (path to .js). Set 'ui' to true for headed mode.wps_pdf_scriptSearch for text in a PDF document. Returns count and results with page (1-based), start/end character offsets, and matched text.wps_pdf_searchCreate a new presentation (.pptx). Set 'title' and 'subtitle' for a title slide. For multi-slide creation, use doc create with outline data.wps_ppt_createExport presentation (.pptx) to another format. Set 'format' to: 'pdf', 'html', 'png' (single slide image, requires 'slide' number), or 'md'/'markdown' (via Presentation.ConvertToMarkdown).wps_ppt_exportRun a custom WPS Presentation JSAPI script. Provide code via 'scrip
```

### 3. offset `5672188` · 76 字符

```javascript
var doc = Application.Documents.Open(templatePath, false, false, false);
```

### 4. offset `5672429` · 78 字符

```javascript
return typeof __result === "string" ? __result : JSON.stringify(__result);
```

### 5. offset `5672525` · 70 字符

```javascript
return JSON.stringify({ok: false, error: e.message || String(e)});
```

### 6. offset `5673294` · 47 字符

```javascript
var pres = Application.Presentations.Add();
```

### 7. offset `5674206` · 41 字符

```javascript
var wb = Application.Workbooks.Add();
```

### 8. offset `5688030` · 109 字符

```javascript
(function() { try { �W } catch(e) { return JSON.stringify({ok: false, error: e.message || String(e)}); } })()
```

### 9. offset `5721371` · 599 字符

```javascript
outlinemetadatatablesexpected numeric indexrange array must have 1 or 2 elementsinvalid range/index valueparagraph range is not an insertion point; use a single paragraph indexrange is a character selection, not an insertion point; use {"char": N} or --insert-at cNedit insertion requires {"char": N} or {"paragraph": N}range must be [start, end] with two different character offsetsinvalid character rangeApplication.ActiveDocument.Content.TextApplication.ActiveDocumentApplication.ActiveDocument.BuiltInDocumentPropertiesApplication.ActiveDocument.TablesApplication.ActiveDocument.Paragraphs.Count
```

### 10. offset `5721993` · 66 字符

```javascript
- �< + 1}, (_, i) => Application.ActiveDocument.Paragraphs.Item(�
```

### 11. offset `5722632` · 64 字符

```javascript
- �: + 1}, (_, i) => Application.ActiveDocument.Sections.Item(�
```

### 12. offset `5723401` · 47 字符

```javascript
if (idx < 0) idx = pres.Slides.Count + idx + 1;
```

### 13. offset `5723450` · 92 字符

```javascript
if (idx < 1 || idx > pres.Slides.Count) throw new Error('slide index out of range: ' + idx);
```

### 14. offset `5723598` · 47 字符

```javascript
for (var j = 1; j <= slide.Shapes.Count; j++) {
```

### 15. offset `5723960` · 62 字符

```javascript
return JSON.stringify({ok: true, slide: idx, shapes: shapes});
```

### 16. offset `5724318` · 83 字符

```javascript
return JSON.stringify({ ok: true, format: "markdown", output: "__OUTPUT_PATH__" });
```

### 17. offset `5724403` · 50 字符

```javascript
__OUTPUT_PATH__pres.SaveAs("__OUTPUT_PATH__", 32);
```

### 18. offset `5724455` · 76 字符

```javascript
return JSON.stringify({ok: true, format: 'pdf', output: "__OUTPUT_PATH__"});
```

### 19. offset `5724570` · 77 字符

```javascript
return JSON.stringify({ok: true, format: 'html', output: "__OUTPUT_PATH__"});
```

### 20. offset `5724697` · 40 字符

```javascript
var slide = pres.Slides.Item(__SLIDE__);
```

### 21. offset `5724886` · 72 字符

```javascript
return JSON.stringify({ok: true, action: 'text_set', slide: __SLIDE__});
```

### 22. offset `5725011` · 46 字符

```javascript
for (var i = 1; i <= pres.Slides.Count; i++) {
```

### 23. offset `5725119` · 51 字符

```javascript
for (var j = 1; j <= slide.Shapes.Count; j++) {
```

### 24. offset `5725172` · 41 字符

```javascript
var shape = slide.Shapes.Item(j);
```

### 25. offset `5725420` · 94 字符

```javascript
slides.push({index: i, name: slide.Name, shapeCount: slide.Shapes.Count, shapes: shapes});
```

### 26. offset `5725519` · 81 字符

```javascript
return JSON.stringify({ok: true, slideCount: pres.Slides.Count, slides: slides});
```

### 27. offset `5727353` · 43 字符

```javascript
var win = Application.ActiveWindow;
```

### 28. offset `5728501` · 88 字符

```javascript
return JSON.stringify({ok: true, message: "success", data: {removed: removed}});
```

### 29. offset `5728596` · 80 字符

```javascript
return JSON.stringify({ok: false, message: "母版中未找到水印", data: {removed: 0}});
```

### 30. offset `5729026` · 85 字符

```javascript
return JSON.stringify({ok: false, message: "母版中已存在水印，请先移除后再添加", data: null});
```

### 31. offset `5731738` · 41 字符

```javascript
if (s < 0) s = pres.Slides.Count + s + 1;
```

### 32. offset `5731780` · 41 字符

```javascript
if (e < 0) e = pres.Slides.Count + e + 1;
```

### 33. offset `5731904` · 103 字符

```javascript
throw new Error('slide range out of range: ' + s + ':' + e + ' (count=' + pres.Slides.Count + ')');
```

### 34. offset `5732973` · 98 字符

```javascript
if (slideIndex < 1 || slideIndex > pres.Slides.Count) throw new Error('slide index out of range');
```

### 35. offset `5733092` · 41 字符

```javascript
var slide = pres.Slides.Item(slideIndex);
```

### 36. offset `5733282` · 64 字符

```javascript
slide.Shapes.AddPicture(image, 0, -1, left, top, width, height);
```

### 37. offset `5733347` · 79 字符

```javascript
return JSON.stringify({ok: true, action: 'image_inserted', slide: slideIndex});
```

### 38. offset `5733925` · 93 字符

```javascript
try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = "__TITLE__"; } catch(e) {}
```

### 39. offset `5734020` · 96 字符

```javascript
try { slide.Shapes.Placeholders.Item(2).TextFrame.TextRange.Text = "__SUBTITLE__"; } catch(e) {}
```

### 40. offset `5734118` · 59 字符

```javascript
return JSON.stringify({ok: true, action: 'slide_created'});
```

### 41. offset `5736339` · 53 字符

```javascript
for (var i = slideMaster.Shapes.Count; i >= 1; i--) {
```

### 42. offset `5736624` · 42 字符

```javascript
var shape = slideMaster.Shapes.AddTextbox(
```

### 43. offset `5739226` · 63 字符

```javascript
var slide = pres.Slides.Add(pres.Slides.Count + 1, layout);
```

### 44. offset `5739291` · 99 字符

```javascript
try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = title || ""; } catch (e1) {}
```

### 45. offset `5739424` · 75 字符

```javascript
var tr = slide.Shapes.Placeholders.Item(2).TextFrame.TextRange;
```

### 46. offset `5741548` · 54 字符

```javascript
var ph = slide.Shapes.Placeholders.Item(__PH_INDEX__);
```

### 47. offset `5741647` · 79 字符

```javascript
return JSON.stringify({ok: true, action: 'placeholder_set', slide: __SLIDE__});
```

### 48. offset `5741728` · 52 字符

```javascript
__PH_INDEX__var slide = pres.Slides.Item(__SLIDE__);
```

### 49. offset `5741823` · 94 字符

```javascript
return JSON.stringify({ok: true, format: 'png', slide: __SLIDE__, output: "__OUTPUT_PATH__"});
```

### 50. offset `5742122` · 109 字符

```javascript
throw new Error('slide notes range out of range: ' + s + ':' + e + ' (count=' + pres.Slides.Count + ')');
```

### 51. offset `5742679` · 69 字符

```javascript
notes = shapeText(notesPage.Shapes.Placeholders.Item(2));
```

### 52. offset `5742798` · 63 字符

```javascript
for (var i = 1; i <= notesPage.Shapes.Count; i++) {
```

### 53. offset `5742863` · 50 字符

```javascript
var sh = notesPage.Shapes.Item(i);
```

### 54. offset `5743317` · 75 字符

```javascript
slides.push({index: i, notes: readNotesForSlide(pres.Slides.Item(i))});
```

### 55. offset `5743415` · 72 字符

```javascript
return JSON.stringify({ok: true, slide: s, notes: slides[0].notes});
```

### 56. offset `5743492` · 93 字符

```javascript
return JSON.stringify({ok: true, from: s, to: e, slideCount: slides.length, slides: slides});
```

### 57. offset `5743587` · 131 字符

```javascript
"remove"__BODY__(function() { try { __BODY__ } catch(e) { return JSON.stringify({ok: false, error: e.message || String(e)}); } })()
```

### 58. offset `5747155` · 85 字符

```javascript
if (vals === null || vals === undefined) return JSON.stringify({ok: true, data: []});
```

### 59. offset `5747242` · 76 字符

```javascript
if (!Array.isArray(vals)) return JSON.stringify({ok: true, data: [[vals]]});
```

### 60. offset `5747320` · 46 字符

```javascript
return JSON.stringify({ok: true, data: vals});
```

### 61. offset `5749017` · 88 字符

```javascript
return JSON.stringify({ok: true, type: "charts", count: charts.length, charts: charts});
```

### 62. offset `5749297` · 57 字符

```javascript
chartObj.Chart.SetSourceData(ws.Range("__DATA_RANGE__"));
```

### 63. offset `5749420` · 59 字符

```javascript
return JSON.stringify({ok: true, action: 'chart_created'});
```

### 64. offset `5749537` · 46 字符

```javascript
ws.Range("__RANGE__").Formula = "__FORMULA__";
```

### 65. offset `5749585` · 77 字符

```javascript
return JSON.stringify({ok: true, action: 'formula_set', range: "__RANGE__"});
```

### 66. offset `5750298` · 98 字符

```javascript
return JSON.stringify({ ok: true, message: "success", data: { path: path, format: "markdown" } });
```

### 67. offset `5750516` · 76 字符

```javascript
return JSON.stringify({ok: true, format: "csv", output: "__OUTPUT_PATH__"});
```

### 68. offset `5750784` · 49 字符

```javascript
return JSON.stringify({ok: true, sheets: names});
```

### 69. offset `5750899` · 81 字符

```javascript
return JSON.stringify({ok: true, data: [], address: null, rows: 0, cols: 0});
```

### 70. offset `5751367` · 90 字符

```javascript
return JSON.stringify({ok: true, data: [], address: address, rows: rows, cols: cols});
```

### 71. offset `5751491` · 90 字符

```javascript
return JSON.stringify({ok: true, data: [[vals]], address: address, rows: 1, cols: 1});
```

### 72. offset `5751691` · 114 字符

```javascript
return JSON.stringify({ok: true, data: [vals], address: address, rows: rows || 1, cols: cols || vals.length});
```

### 73. offset `5751810` · 88 字符

```javascript
return JSON.stringify({ok: true, data: vals, address: address, rows: rows, cols: cols});
```

### 74. offset `5752190` · 77 字符

```javascript
return JSON.stringify({ok: true, action: 'data_written', rows: data.length});
```

### 75. offset `5752397` · 68 字符

```javascript
return JSON.stringify({ok: true, values: vals, formulas: formulas});
```

### 76. offset `5752777` · 96 字符

```javascript
return JSON.stringify({ok: true, format: "png", range: "__RANGE__", output: "__OUTPUT_PATH__"});
```

### 77. offset `5752903` · 43 字符

```javascript
ws.Range("__RANGE__").Value2 = "__VALUE__";
```

### 78. offset `5752948` · 72 字符

```javascript
return JSON.stringify({ok: true, action: 'edited', range: "__RANGE__"});
```

### 79. offset `5753252` · 60 字符

```javascript
for (var i = 1; i <= Application.Documents.Count; i++) {
```

### 80. offset `5753314` · 44 字符

```javascript
var d = Application.Documents.Item(i);
```

### 81. offset `5753441` · 78 字符

```javascript
if (!doc) doc = Application.Documents.Open(filePath, false, false, false);
```

### 82. offset `5753981` · 78 字符

```javascript
var pdfDoc = Application.Documents.Open(filePath,false,false,"","",false);
```

### 83. offset `5754579` · 64 字符

```javascript
for (var i = 1; i <= Application.Presentations.Count; i++) {
```

### 84. offset `5754645` · 48 字符

```javascript
var p = Application.Presentations.Item(i);
```

### 85. offset `5754791` · 84 字符

```javascript
if (!pres) pres = Application.Presentations.Open(filePath, false, false, false);
```

### 86. offset `5755475` · 60 字符

```javascript
for (var i = 1; i <= Application.Workbooks.Count; i++) {
```

### 87. offset `5755537` · 44 字符

```javascript
var w = Application.Workbooks.Item(i);
```

### 88. offset `5755663` · 55 字符

```javascript
if (!wb) wb = Application.Workbooks.Open(filePath);
```

### 89. offset `5759074` · 57 字符

```javascript
try { Application.DisplayAlerts = false; } catch (e) {}
```

### 90. offset `5759631` · 46 字符

```javascript
var active = Application.ActiveWorkbook;
```

### 91. offset `5759765` · 62 字符

```javascript
tried.push("Application.ActiveWorkbook: " + errText(e));
```

### 92. offset `5759876` · 47 字符

```javascript
wb = Application.Workbooks.Open(srcPath);
```

### 93. offset `5760564` · 74 字符

```javascript
return JSON.stringify({ ok: true, action: "created", from: srcPath });
```

### 94. offset `5764830` · 339 字符

```javascript
either --script or --script-file requiredok--text and --from-file cannot be used togethermessagesuggestionexit_codestdoutlogsactioncommandsnamedescriptionerrorparametersoutput_schemaidempotentdata>(function(){ try { return JSON.stringify({ok:true, text: OCR('�N')}); } catch(e) { return JSON.stringify({ok:false, error: e.message}); } })()
```

### 95. offset `5786807` · 42 字符

```javascript
var doc = Application.Documents.Add();
```

### 96. offset `5804466` · 54 字符

```javascript
return JSON.stringify({ok: true, action: 'appended'});
```

### 97. offset `5807467` · 66 字符

```javascript
return JSON.stringify({ok: true, type: 'tables', tables: tables});
```

### 98. offset `5808238` · 86 字符

```javascript
return JSON.stringify({ ok: true, message: "success", data: { savePath: savePath } });
```

### 99. offset `5808828` · 68 字符

```javascript
return JSON.stringify({ok: true, type: 'outline', items: headings});
```

### 100. offset `5808934` · 76 字符

```javascript
return JSON.stringify({ok: true, format: 'txt', output: "__OUTPUT_PATH__"});
```

### 101. offset `5809581` · 49 字符

```javascript
return JSON.stringify({ok: true, filled: count});
```

### 102. offset `5809853` · 68 字符

```javascript
return JSON.stringify({ok: true, type: 'metadata', metadata: meta});
```

### 103. offset `5810228` · 72 字符

```javascript
return JSON.stringify({ok: true, issues: issues, total: issues.length});
```

### 104. offset `5810747` · 53 字符

```javascript
return JSON.stringify({ok: true, action: 'created'});
```

### 105. offset `5811558` · 120 字符

```javascript
return JSON.stringify({ok: true, format: "png", pages: total, exported: files.length, outputDir: outDir, files: files});
```

### 106. offset `5813453` · 56 字符

```javascript
for (var i = 1; i <= Application.Documents.Count; i++) {
```

### 107. offset `5813511` · 57 字符

```javascript
before[docKey(Application.Documents.Item(i))] = true;
```

### 108. offset `5813939` · 56 字符

```javascript
for (var j = 1; j <= Application.Documents.Count; j++) {
```

### 109. offset `5813997` · 42 字符

```javascript
var d = Application.Documents.Item(j);
```

### 110. offset `5815348` · 51 字符

```javascript
return JSON.stringify({ok: true, replaced: count});
```

### 111. offset `5816030` · 80 字符

```javascript
return JSON.stringify({ok: true, action: 'image_inserted', paragraph: idx});
```

### 112. offset `5816113` · 71 字符

```javascript
return JSON.stringify({ok: true, action: 'image_inserted', char: idx});
```

### 113. offset `5816359` · 63 字符

```javascript
return JSON.stringify({ok: true, action: 'inserted', char: n});
```

### 114. offset `5816689` · 53 字符

```javascript
return JSON.stringify({ok: true, paragraphs: texts});
```

### 115. offset `5817122` · 77 字符

```javascript
if (!src) src = Application.Documents.Open(srcPath, false, false, false);
```

### 116. offset `5817438` · 57 字符

```javascript
return JSON.stringify({ok: true, action: "created"});
```

### 117. offset `5817933` · 73 字符

```javascript
return JSON.stringify({ok: true, action: 'inserted', at_paragraph: idx});
```

### 118. offset `5818625` · 79 字符

```javascript
var pdfDoc = Application.Documents.Open(filePath, false, false, "", "", false);
```

### 119. offset `5818836` · 99 字符

```javascript
return JSON.stringify({ok: true, action: "merged", output: __OUTPUT_PATH__, fileCount: __COUNT__});
```

### 120. offset `5819021` · 82 字符

```javascript
if (!keyword) { return JSON.stringify({ok: false, error: "empty search query"}); }
```

### 121. offset `5819487` · 128 字符

```javascript
return JSON.stringify({ok: true, query: keyword, count: results.length, truncated: results.length >= 100000, results: results});
```

### 122. offset `5819687` · 117 字符

```javascript
return JSON.stringify({ok: true, action: "split", pages_per_file: __PAGES_PER_FILE__, output_dir: "__OUTPUT_DIR__"});
```

### 123. offset `5820564` · 107 字符

```javascript
return JSON.stringify({ok: true, action: "extracted", pages: "__PAGE_RANGE__", output: "__OUTPUT_PATH__"});
```

### 124. offset `5820949` · 50 字符

```javascript
return JSON.stringify({ok: true, metadata: meta});
```

### 125. offset `5822166` · 68 字符

```javascript
return JSON.stringify({ok: true, bookmarks: collectOutlines(root)});
```

### 126. offset `5822877` · 75 字符

```javascript
return JSON.stringify({ok: true, format: "png", output: "__OUTPUT_DIR__"});
```

### 127. offset `5823235` · 72 字符

```javascript
return JSON.stringify({ok: true, count: fields.length, fields: fields});
```

### 128. offset `5824077` · 48 字符

```javascript
return JSON.stringify({ok: true, pages: texts});
```

### 129. offset `5824224` · 61 字符

```javascript
return JSON.stringify({ok: true, action: "watermark_added"});
```
