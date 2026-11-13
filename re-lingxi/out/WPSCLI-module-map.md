# wpscli：JS 片段 → Rust 模块家族（按模块路径标记划区）

- 源: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe`
- 模块路径标记 169 个 / 56 种；片段 129 条
- 归属：取片段 offset 最近的 `src/*.rs` 或 `jsapi/*.rs` 路径标记；距离列给可靠性

## 模块 × 片段数

| 模块 | 片段数 |
|---|---|
| `slice.rs` | 46 |
| `addressing.rs` | 34 |
| `pdf_scripts.rs` | 22 |
| `str.rs` | 9 |
| `de.rs` | 5 |
| `auth.rs` | 4 |
| `server.rs` | 3 |
| `service.rs` | 3 |
| `string.rs` | 1 |
| `main.rs` | 1 |
| `bytes_mut.rs` | 1 |

## 逐片段

| offset | 模块 | 距离 | 长度 | 开头 |
|---|---|---|---|---|
| `5721371` | `addressing.rs` | 1122 | 599 | `outlinemetadatatablesexpected numeric indexrange array must have 1 or 2 elements` |
| `5721993` | `addressing.rs` | 1744 | 66 | `- �< + 1}, (_, i) => Application.ActiveDocument.Paragraphs.Item(�` |
| `5722632` | `addressing.rs` | 2383 | 64 | `- �: + 1}, (_, i) => Application.ActiveDocument.Sections.Item(�` |
| `5723401` | `addressing.rs` | 3152 | 47 | `if (idx < 0) idx = pres.Slides.Count + idx + 1;` |
| `5723450` | `addressing.rs` | 3201 | 92 | `if (idx < 1 // idx > pres.Slides.Count) throw new Error('slide index out of rang` |
| `5723598` | `addressing.rs` | 3349 | 47 | `for (var j = 1; j <= slide.Shapes.Count; j++) {` |
| `5723960` | `addressing.rs` | 3711 | 62 | `return JSON.stringify({ok: true, slide: idx, shapes: shapes});` |
| `5724318` | `addressing.rs` | 4069 | 83 | `return JSON.stringify({ ok: true, format: "markdown", output: "__OUTPUT_PATH__" ` |
| `5724403` | `addressing.rs` | 4154 | 50 | `__OUTPUT_PATH__pres.SaveAs("__OUTPUT_PATH__", 32);` |
| `5724455` | `addressing.rs` | 4206 | 76 | `return JSON.stringify({ok: true, format: 'pdf', output: "__OUTPUT_PATH__"});` |
| `5724570` | `addressing.rs` | 4321 | 77 | `return JSON.stringify({ok: true, format: 'html', output: "__OUTPUT_PATH__"});` |
| `5724697` | `addressing.rs` | 4448 | 40 | `var slide = pres.Slides.Item(__SLIDE__);` |
| `5724886` | `addressing.rs` | 4637 | 72 | `return JSON.stringify({ok: true, action: 'text_set', slide: __SLIDE__});` |
| `5725011` | `addressing.rs` | 4762 | 46 | `for (var i = 1; i <= pres.Slides.Count; i++) {` |
| `5725119` | `addressing.rs` | 4870 | 51 | `for (var j = 1; j <= slide.Shapes.Count; j++) {` |
| `5725172` | `addressing.rs` | 4923 | 41 | `var shape = slide.Shapes.Item(j);` |
| `5725420` | `addressing.rs` | 5171 | 94 | `slides.push({index: i, name: slide.Name, shapeCount: slide.Shapes.Count, shapes:` |
| `5725519` | `addressing.rs` | 5270 | 81 | `return JSON.stringify({ok: true, slideCount: pres.Slides.Count, slides: slides})` |
| `5727353` | `addressing.rs` | 7104 | 43 | `var win = Application.ActiveWindow;` |
| `5728501` | `addressing.rs` | 8252 | 88 | `return JSON.stringify({ok: true, message: "success", data: {removed: removed}});` |
| `5728596` | `addressing.rs` | 8347 | 80 | `return JSON.stringify({ok: false, message: "母版中未找到水印", data: {removed: 0}});` |
| `5729026` | `addressing.rs` | 8777 | 85 | `return JSON.stringify({ok: false, message: "母版中已存在水印，请先移除后再添加", data: null});` |
| `5731738` | `addressing.rs` | 11489 | 41 | `if (s < 0) s = pres.Slides.Count + s + 1;` |
| `5731780` | `addressing.rs` | 11531 | 41 | `if (e < 0) e = pres.Slides.Count + e + 1;` |
| `5731904` | `addressing.rs` | 11655 | 103 | `throw new Error('slide range out of range: ' + s + ':' + e + ' (count=' + pres.S` |
| `5732973` | `addressing.rs` | 12724 | 98 | `if (slideIndex < 1 // slideIndex > pres.Slides.Count) throw new Error('slide ind` |
| `5733092` | `addressing.rs` | 12843 | 41 | `var slide = pres.Slides.Item(slideIndex);` |
| `5733282` | `addressing.rs` | 13033 | 64 | `slide.Shapes.AddPicture(image, 0, -1, left, top, width, height);` |
| `5733347` | `addressing.rs` | 13098 | 79 | `return JSON.stringify({ok: true, action: 'image_inserted', slide: slideIndex});` |
| `5733925` | `addressing.rs` | 13676 | 93 | `try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = "__TITLE__"; ` |
| `5734020` | `addressing.rs` | 13771 | 96 | `try { slide.Shapes.Placeholders.Item(2).TextFrame.TextRange.Text = "__SUBTITLE__` |
| `5734118` | `addressing.rs` | 13869 | 59 | `return JSON.stringify({ok: true, action: 'slide_created'});` |
| `5736339` | `addressing.rs` | 16090 | 53 | `for (var i = slideMaster.Shapes.Count; i >= 1; i--) {` |
| `5736624` | `addressing.rs` | 16375 | 42 | `var shape = slideMaster.Shapes.AddTextbox(` |
| `5759074` | `auth.rs` | 923 | 57 | `try { Application.DisplayAlerts = false; } catch (e) {}` |
| `5759631` | `auth.rs` | 1480 | 46 | `var active = Application.ActiveWorkbook;` |
| `5759765` | `auth.rs` | 1614 | 62 | `tried.push("Application.ActiveWorkbook: " + errText(e));` |
| `5759876` | `auth.rs` | 1725 | 47 | `wb = Application.Workbooks.Open(srcPath);` |
| `5786807` | `bytes_mut.rs` | 1964 | 42 | `var doc = Application.Documents.Add();` |
| `5672188` | `de.rs` | 6117 | 76 | `var doc = Application.Documents.Open(templatePath, false, false, false);` |
| `5672429` | `de.rs` | 5876 | 78 | `return typeof __result === "string" ? __result : JSON.stringify(__result);` |
| `5672525` | `de.rs` | 5780 | 70 | `return JSON.stringify({ok: false, error: e.message // String(e)});` |
| `5673294` | `de.rs` | 5011 | 47 | `var pres = Application.Presentations.Add();` |
| `5674206` | `de.rs` | 4099 | 41 | `var wb = Application.Workbooks.Add();` |
| `5764830` | `main.rs` | 636 | 339 | `either --script or --script-file requiredok--text and --from-file cannot be used` |
| `5811558` | `pdf_scripts.rs` | 6981 | 120 | `return JSON.stringify({ok: true, format: "png", pages: total, exported: files.le` |
| `5813453` | `pdf_scripts.rs` | 5086 | 56 | `for (var i = 1; i <= Application.Documents.Count; i++) {` |
| `5813511` | `pdf_scripts.rs` | 5028 | 57 | `before[docKey(Application.Documents.Item(i))] = true;` |
| `5813939` | `pdf_scripts.rs` | 4600 | 56 | `for (var j = 1; j <= Application.Documents.Count; j++) {` |
| `5813997` | `pdf_scripts.rs` | 4542 | 42 | `var d = Application.Documents.Item(j);` |
| `5815348` | `pdf_scripts.rs` | 3191 | 51 | `return JSON.stringify({ok: true, replaced: count});` |
| `5816030` | `pdf_scripts.rs` | 2509 | 80 | `return JSON.stringify({ok: true, action: 'image_inserted', paragraph: idx});` |
| `5816113` | `pdf_scripts.rs` | 2426 | 71 | `return JSON.stringify({ok: true, action: 'image_inserted', char: idx});` |
| `5816359` | `pdf_scripts.rs` | 2180 | 63 | `return JSON.stringify({ok: true, action: 'inserted', char: n});` |
| `5816689` | `pdf_scripts.rs` | 1850 | 53 | `return JSON.stringify({ok: true, paragraphs: texts});` |
| `5817122` | `pdf_scripts.rs` | 1417 | 77 | `if (!src) src = Application.Documents.Open(srcPath, false, false, false);` |
| `5817438` | `pdf_scripts.rs` | 1101 | 57 | `return JSON.stringify({ok: true, action: "created"});` |
| `5817933` | `pdf_scripts.rs` | 606 | 73 | `return JSON.stringify({ok: true, action: 'inserted', at_paragraph: idx});` |
| `5818625` | `pdf_scripts.rs` | 86 | 79 | `var pdfDoc = Application.Documents.Open(filePath, false, false, "", "", false);` |
| `5818836` | `pdf_scripts.rs` | 297 | 99 | `return JSON.stringify({ok: true, action: "merged", output: __OUTPUT_PATH__, file` |
| `5819021` | `pdf_scripts.rs` | 482 | 82 | `if (!keyword) { return JSON.stringify({ok: false, error: "empty search query"});` |
| `5819487` | `pdf_scripts.rs` | 948 | 128 | `return JSON.stringify({ok: true, query: keyword, count: results.length, truncate` |
| `5819687` | `pdf_scripts.rs` | 1148 | 117 | `return JSON.stringify({ok: true, action: "split", pages_per_file: __PAGES_PER_FI` |
| `5820564` | `pdf_scripts.rs` | 2025 | 107 | `return JSON.stringify({ok: true, action: "extracted", pages: "__PAGE_RANGE__", o` |
| `5820949` | `pdf_scripts.rs` | 2410 | 50 | `return JSON.stringify({ok: true, metadata: meta});` |
| `5822166` | `pdf_scripts.rs` | 3627 | 68 | `return JSON.stringify({ok: true, bookmarks: collectOutlines(root)});` |
| `5822877` | `pdf_scripts.rs` | 4338 | 75 | `return JSON.stringify({ok: true, format: "png", output: "__OUTPUT_DIR__"});` |
| `5656768` | `server.rs` | 2907 | 143 | `>(function(){ try { return JSON.stringify({ok:true, text: OCR('�N')}); } catch(e` |
| `5657152` | `server.rs` | 3291 | 4000 | `wps_searchCheck WPS Office installation, engine connectivity, file version (> 12` |
| `5688030` | `server.rs` | 1738 | 109 | `(function() { try { �W } catch(e) { return JSON.stringify({ok: false, error: e.m` |
| `5823235` | `service.rs` | 4085 | 72 | `return JSON.stringify({ok: true, count: fields.length, fields: fields});` |
| `5824077` | `service.rs` | 3243 | 48 | `return JSON.stringify({ok: true, pages: texts});` |
| `5824224` | `service.rs` | 3096 | 61 | `return JSON.stringify({ok: true, action: "watermark_added"});` |
| `5739226` | `slice.rs` | 16967 | 63 | `var slide = pres.Slides.Add(pres.Slides.Count + 1, layout);` |
| `5739291` | `slice.rs` | 16902 | 99 | `try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = title // ""; ` |
| `5739424` | `slice.rs` | 16769 | 75 | `var tr = slide.Shapes.Placeholders.Item(2).TextFrame.TextRange;` |
| `5741548` | `slice.rs` | 14645 | 54 | `var ph = slide.Shapes.Placeholders.Item(__PH_INDEX__);` |
| `5741647` | `slice.rs` | 14546 | 79 | `return JSON.stringify({ok: true, action: 'placeholder_set', slide: __SLIDE__});` |
| `5741728` | `slice.rs` | 14465 | 52 | `__PH_INDEX__var slide = pres.Slides.Item(__SLIDE__);` |
| `5741823` | `slice.rs` | 14370 | 94 | `return JSON.stringify({ok: true, format: 'png', slide: __SLIDE__, output: "__OUT` |
| `5742122` | `slice.rs` | 14071 | 109 | `throw new Error('slide notes range out of range: ' + s + ':' + e + ' (count=' + ` |
| `5742679` | `slice.rs` | 13514 | 69 | `notes = shapeText(notesPage.Shapes.Placeholders.Item(2));` |
| `5742798` | `slice.rs` | 13395 | 63 | `for (var i = 1; i <= notesPage.Shapes.Count; i++) {` |
| `5742863` | `slice.rs` | 13330 | 50 | `var sh = notesPage.Shapes.Item(i);` |
| `5743317` | `slice.rs` | 12876 | 75 | `slides.push({index: i, notes: readNotesForSlide(pres.Slides.Item(i))});` |
| `5743415` | `slice.rs` | 12778 | 72 | `return JSON.stringify({ok: true, slide: s, notes: slides[0].notes});` |
| `5743492` | `slice.rs` | 12701 | 93 | `return JSON.stringify({ok: true, from: s, to: e, slideCount: slides.length, slid` |
| `5743587` | `slice.rs` | 12606 | 131 | `"remove"__BODY__(function() { try { __BODY__ } catch(e) { return JSON.stringify(` |
| `5747155` | `slice.rs` | 9038 | 85 | `if (vals === null // vals === undefined) return JSON.stringify({ok: true, data: ` |
| `5747242` | `slice.rs` | 8951 | 76 | `if (!Array.isArray(vals)) return JSON.stringify({ok: true, data: [[vals]]});` |
| `5747320` | `slice.rs` | 8873 | 46 | `return JSON.stringify({ok: true, data: vals});` |
| `5749017` | `slice.rs` | 7176 | 88 | `return JSON.stringify({ok: true, type: "charts", count: charts.length, charts: c` |
| `5749297` | `slice.rs` | 6896 | 57 | `chartObj.Chart.SetSourceData(ws.Range("__DATA_RANGE__"));` |
| `5749420` | `slice.rs` | 6773 | 59 | `return JSON.stringify({ok: true, action: 'chart_created'});` |
| `5749537` | `slice.rs` | 6656 | 46 | `ws.Range("__RANGE__").Formula = "__FORMULA__";` |
| `5749585` | `slice.rs` | 6608 | 77 | `return JSON.stringify({ok: true, action: 'formula_set', range: "__RANGE__"});` |
| `5750298` | `slice.rs` | 5895 | 98 | `return JSON.stringify({ ok: true, message: "success", data: { path: path, format` |
| `5750516` | `slice.rs` | 5677 | 76 | `return JSON.stringify({ok: true, format: "csv", output: "__OUTPUT_PATH__"});` |
| `5750784` | `slice.rs` | 5409 | 49 | `return JSON.stringify({ok: true, sheets: names});` |
| `5750899` | `slice.rs` | 5294 | 81 | `return JSON.stringify({ok: true, data: [], address: null, rows: 0, cols: 0});` |
| `5751367` | `slice.rs` | 4826 | 90 | `return JSON.stringify({ok: true, data: [], address: address, rows: rows, cols: c` |
| `5751491` | `slice.rs` | 4702 | 90 | `return JSON.stringify({ok: true, data: [[vals]], address: address, rows: 1, cols` |
| `5751691` | `slice.rs` | 4502 | 114 | `return JSON.stringify({ok: true, data: [vals], address: address, rows: rows // 1` |
| `5751810` | `slice.rs` | 4383 | 88 | `return JSON.stringify({ok: true, data: vals, address: address, rows: rows, cols:` |
| `5752190` | `slice.rs` | 4003 | 77 | `return JSON.stringify({ok: true, action: 'data_written', rows: data.length});` |
| `5752397` | `slice.rs` | 3796 | 68 | `return JSON.stringify({ok: true, values: vals, formulas: formulas});` |
| `5752777` | `slice.rs` | 3416 | 96 | `return JSON.stringify({ok: true, format: "png", range: "__RANGE__", output: "__O` |
| `5752903` | `slice.rs` | 3290 | 43 | `ws.Range("__RANGE__").Value2 = "__VALUE__";` |
| `5752948` | `slice.rs` | 3245 | 72 | `return JSON.stringify({ok: true, action: 'edited', range: "__RANGE__"});` |
| `5753252` | `slice.rs` | 2941 | 60 | `for (var i = 1; i <= Application.Documents.Count; i++) {` |
| `5753314` | `slice.rs` | 2879 | 44 | `var d = Application.Documents.Item(i);` |
| `5753441` | `slice.rs` | 2752 | 78 | `if (!doc) doc = Application.Documents.Open(filePath, false, false, false);` |
| `5753981` | `slice.rs` | 2212 | 78 | `var pdfDoc = Application.Documents.Open(filePath,false,false,"","",false);` |
| `5754579` | `slice.rs` | 1614 | 64 | `for (var i = 1; i <= Application.Presentations.Count; i++) {` |
| `5754645` | `slice.rs` | 1548 | 48 | `var p = Application.Presentations.Item(i);` |
| `5754791` | `slice.rs` | 1402 | 84 | `if (!pres) pres = Application.Presentations.Open(filePath, false, false, false);` |
| `5755475` | `slice.rs` | 718 | 60 | `for (var i = 1; i <= Application.Workbooks.Count; i++) {` |
| `5755537` | `slice.rs` | 656 | 44 | `var w = Application.Workbooks.Item(i);` |
| `5755663` | `slice.rs` | 530 | 55 | `if (!wb) wb = Application.Workbooks.Open(filePath);` |
| `5804466` | `str.rs` | 587 | 54 | `return JSON.stringify({ok: true, action: 'appended'});` |
| `5807467` | `str.rs` | 3588 | 66 | `return JSON.stringify({ok: true, type: 'tables', tables: tables});` |
| `5808238` | `str.rs` | 4359 | 86 | `return JSON.stringify({ ok: true, message: "success", data: { savePath: savePath` |
| `5808828` | `str.rs` | 4949 | 68 | `return JSON.stringify({ok: true, type: 'outline', items: headings});` |
| `5808934` | `str.rs` | 5055 | 76 | `return JSON.stringify({ok: true, format: 'txt', output: "__OUTPUT_PATH__"});` |
| `5809581` | `str.rs` | 5702 | 49 | `return JSON.stringify({ok: true, filled: count});` |
| `5809853` | `str.rs` | 5974 | 68 | `return JSON.stringify({ok: true, type: 'metadata', metadata: meta});` |
| `5810228` | `str.rs` | 6349 | 72 | `return JSON.stringify({ok: true, issues: issues, total: issues.length});` |
| `5810747` | `str.rs` | 6868 | 53 | `return JSON.stringify({ok: true, action: 'created'});` |
| `5760564` | `string.rs` | 1521 | 74 | `return JSON.stringify({ ok: true, action: "created", from: srcPath });` |

## 工具家族 ↔ 模块（按命名对应，非反编译结论）

| 工具前缀 | 预期模块 | 依据 |
|---|---|---|
| `wps_doc_*` | `commands/doc.rs` | 工具名与模块名同族；描述里出现 `src\commands\doc.rs` |
| `wps_sheet_*` | `commands/sheet*.rs` | 同上 |
| `wps_ppt_*` | `commands/ppt*.rs` | 同上 |
| `wps_pdf_*` | `jsapi/pdf_scripts.rs` + `commands/pdf*.rs` | `pdf_scripts.rs` 已命中 |
| `wps_status` / `wps_search` / `wps_file_info` | `doctor.rs` / 无 | 无 JS 片段 |