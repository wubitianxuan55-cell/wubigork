# wpscli：JS 片段 → 工具归属（物理邻近映射）

- 源: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe`
- 语义工具 **37** 个（含 legacy），片段 **129** 条，IIFE 外壳 **25** 个
- 归属规则：逐片段找**最近的工具名**（不设窗口），距离列给出可靠性；距离越大越可疑

## 工具 × 片段数

| 工具 | 片段数 | 外壳数 | 用到占位符 |
|---|---|---|---|
| `wps_presentation` | 122 | 21 | __BODY__, __COUNT__, __DATA_RANGE__, __FORMULA__, __OUTPUT_DIR__, __OUTPUT_PATH__… |
| `wps_writer` | 5 | 3 |  |
| `wps_search` | 2 | 1 |  |

## 逐片段明细

| offset | 工具 | 距离 | 长度 | 占位符 | 开头 |
|---|---|---|---|---|---|
| `5688030` | `wps_presentation` | 12926 | 109 | — | `(function() { try { �W } catch(e) { return JSON.stringify({ok: false, ` |
| `5721371` | `wps_presentation` | 46267 | 599 | — | `outlinemetadatatablesexpected numeric indexrange array must have 1 or ` |
| `5721993` | `wps_presentation` | 46889 | 66 | — | `- �< + 1}, (_, i) => Application.ActiveDocument.Paragraphs.Item(�` |
| `5722632` | `wps_presentation` | 47528 | 64 | — | `- �: + 1}, (_, i) => Application.ActiveDocument.Sections.Item(�` |
| `5723401` | `wps_presentation` | 48297 | 47 | — | `if (idx < 0) idx = pres.Slides.Count + idx + 1;` |
| `5723450` | `wps_presentation` | 48346 | 92 | — | `if (idx < 1 // idx > pres.Slides.Count) throw new Error('slide index o` |
| `5723598` | `wps_presentation` | 48494 | 47 | — | `for (var j = 1; j <= slide.Shapes.Count; j++) {` |
| `5723960` | `wps_presentation` | 48856 | 62 | — | `return JSON.stringify({ok: true, slide: idx, shapes: shapes});` |
| `5724318` | `wps_presentation` | 49214 | 83 | __OUTPUT_PATH__ | `return JSON.stringify({ ok: true, format: "markdown", output: "__OUTPU` |
| `5724403` | `wps_presentation` | 49299 | 50 | __OUTPUT_PATH__ | `__OUTPUT_PATH__pres.SaveAs("__OUTPUT_PATH__", 32);` |
| `5724455` | `wps_presentation` | 49351 | 76 | __OUTPUT_PATH__ | `return JSON.stringify({ok: true, format: 'pdf', output: "__OUTPUT_PATH` |
| `5724570` | `wps_presentation` | 49466 | 77 | __OUTPUT_PATH__ | `return JSON.stringify({ok: true, format: 'html', output: "__OUTPUT_PAT` |
| `5724697` | `wps_presentation` | 49593 | 40 | __SLIDE__ | `var slide = pres.Slides.Item(__SLIDE__);` |
| `5724886` | `wps_presentation` | 49782 | 72 | __SLIDE__ | `return JSON.stringify({ok: true, action: 'text_set', slide: __SLIDE__}` |
| `5725011` | `wps_presentation` | 49907 | 46 | — | `for (var i = 1; i <= pres.Slides.Count; i++) {` |
| `5725119` | `wps_presentation` | 50015 | 51 | — | `for (var j = 1; j <= slide.Shapes.Count; j++) {` |
| `5725172` | `wps_presentation` | 50068 | 41 | — | `var shape = slide.Shapes.Item(j);` |
| `5725420` | `wps_presentation` | 50316 | 94 | — | `slides.push({index: i, name: slide.Name, shapeCount: slide.Shapes.Coun` |
| `5725519` | `wps_presentation` | 50415 | 81 | — | `return JSON.stringify({ok: true, slideCount: pres.Slides.Count, slides` |
| `5727353` | `wps_presentation` | 52249 | 43 | — | `var win = Application.ActiveWindow;` |
| `5728501` | `wps_presentation` | 53397 | 88 | — | `return JSON.stringify({ok: true, message: "success", data: {removed: r` |
| `5728596` | `wps_presentation` | 53492 | 80 | — | `return JSON.stringify({ok: false, message: "母版中未找到水印", data: {removed:` |
| `5729026` | `wps_presentation` | 53922 | 85 | — | `return JSON.stringify({ok: false, message: "母版中已存在水印，请先移除后再添加", data: ` |
| `5731738` | `wps_presentation` | 56634 | 41 | — | `if (s < 0) s = pres.Slides.Count + s + 1;` |
| `5731780` | `wps_presentation` | 56676 | 41 | — | `if (e < 0) e = pres.Slides.Count + e + 1;` |
| `5731904` | `wps_presentation` | 56800 | 103 | — | `throw new Error('slide range out of range: ' + s + ':' + e + ' (count=` |
| `5732973` | `wps_presentation` | 57869 | 98 | — | `if (slideIndex < 1 // slideIndex > pres.Slides.Count) throw new Error(` |
| `5733092` | `wps_presentation` | 57988 | 41 | — | `var slide = pres.Slides.Item(slideIndex);` |
| `5733282` | `wps_presentation` | 58178 | 64 | — | `slide.Shapes.AddPicture(image, 0, -1, left, top, width, height);` |
| `5733347` | `wps_presentation` | 58243 | 79 | — | `return JSON.stringify({ok: true, action: 'image_inserted', slide: slid` |
| `5733925` | `wps_presentation` | 58821 | 93 | __TITLE__ | `try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = "__` |
| `5734020` | `wps_presentation` | 58916 | 96 | __SUBTITLE__ | `try { slide.Shapes.Placeholders.Item(2).TextFrame.TextRange.Text = "__` |
| `5734118` | `wps_presentation` | 59014 | 59 | — | `return JSON.stringify({ok: true, action: 'slide_created'});` |
| `5736339` | `wps_presentation` | 61235 | 53 | — | `for (var i = slideMaster.Shapes.Count; i >= 1; i--) {` |
| `5736624` | `wps_presentation` | 61520 | 42 | — | `var shape = slideMaster.Shapes.AddTextbox(` |
| `5739226` | `wps_presentation` | 64122 | 63 | — | `var slide = pres.Slides.Add(pres.Slides.Count + 1, layout);` |
| `5739291` | `wps_presentation` | 64187 | 99 | — | `try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = tit` |
| `5739424` | `wps_presentation` | 64320 | 75 | — | `var tr = slide.Shapes.Placeholders.Item(2).TextFrame.TextRange;` |
| `5741548` | `wps_presentation` | 66444 | 54 | __PH_INDEX__ | `var ph = slide.Shapes.Placeholders.Item(__PH_INDEX__);` |
| `5741647` | `wps_presentation` | 66543 | 79 | __SLIDE__ | `return JSON.stringify({ok: true, action: 'placeholder_set', slide: __S` |
| `5741728` | `wps_presentation` | 66624 | 52 | __PH_INDEX__, __SLIDE__ | `__PH_INDEX__var slide = pres.Slides.Item(__SLIDE__);` |
| `5741823` | `wps_presentation` | 66719 | 94 | __OUTPUT_PATH__, __SLIDE__ | `return JSON.stringify({ok: true, format: 'png', slide: __SLIDE__, outp` |
| `5742122` | `wps_presentation` | 67018 | 109 | — | `throw new Error('slide notes range out of range: ' + s + ':' + e + ' (` |
| `5742679` | `wps_presentation` | 67575 | 69 | — | `notes = shapeText(notesPage.Shapes.Placeholders.Item(2));` |
| `5742798` | `wps_presentation` | 67694 | 63 | — | `for (var i = 1; i <= notesPage.Shapes.Count; i++) {` |
| `5742863` | `wps_presentation` | 67759 | 50 | — | `var sh = notesPage.Shapes.Item(i);` |
| `5743317` | `wps_presentation` | 68213 | 75 | — | `slides.push({index: i, notes: readNotesForSlide(pres.Slides.Item(i))})` |
| `5743415` | `wps_presentation` | 68311 | 72 | — | `return JSON.stringify({ok: true, slide: s, notes: slides[0].notes});` |
| `5743492` | `wps_presentation` | 68388 | 93 | — | `return JSON.stringify({ok: true, from: s, to: e, slideCount: slides.le` |
| `5743587` | `wps_presentation` | 68483 | 131 | __BODY__ | `"remove"__BODY__(function() { try { __BODY__ } catch(e) { return JSON.` |
| `5747155` | `wps_presentation` | 72051 | 85 | — | `if (vals === null // vals === undefined) return JSON.stringify({ok: tr` |
| `5747242` | `wps_presentation` | 72138 | 76 | — | `if (!Array.isArray(vals)) return JSON.stringify({ok: true, data: [[val` |
| `5747320` | `wps_presentation` | 72216 | 46 | — | `return JSON.stringify({ok: true, data: vals});` |
| `5749017` | `wps_presentation` | 73913 | 88 | — | `return JSON.stringify({ok: true, type: "charts", count: charts.length,` |
| `5749297` | `wps_presentation` | 74193 | 57 | __DATA_RANGE__ | `chartObj.Chart.SetSourceData(ws.Range("__DATA_RANGE__"));` |
| `5749420` | `wps_presentation` | 74316 | 59 | — | `return JSON.stringify({ok: true, action: 'chart_created'});` |
| `5749537` | `wps_presentation` | 74433 | 46 | __FORMULA__, __RANGE__ | `ws.Range("__RANGE__").Formula = "__FORMULA__";` |
| `5749585` | `wps_presentation` | 74481 | 77 | __RANGE__ | `return JSON.stringify({ok: true, action: 'formula_set', range: "__RANG` |
| `5750298` | `wps_presentation` | 75194 | 98 | — | `return JSON.stringify({ ok: true, message: "success", data: { path: pa` |
| `5750516` | `wps_presentation` | 75412 | 76 | __OUTPUT_PATH__ | `return JSON.stringify({ok: true, format: "csv", output: "__OUTPUT_PATH` |
| `5750784` | `wps_presentation` | 75680 | 49 | — | `return JSON.stringify({ok: true, sheets: names});` |
| `5750899` | `wps_presentation` | 75795 | 81 | — | `return JSON.stringify({ok: true, data: [], address: null, rows: 0, col` |
| `5751367` | `wps_presentation` | 76263 | 90 | — | `return JSON.stringify({ok: true, data: [], address: address, rows: row` |
| `5751491` | `wps_presentation` | 76387 | 90 | — | `return JSON.stringify({ok: true, data: [[vals]], address: address, row` |
| `5751691` | `wps_presentation` | 76587 | 114 | — | `return JSON.stringify({ok: true, data: [vals], address: address, rows:` |
| `5751810` | `wps_presentation` | 76706 | 88 | — | `return JSON.stringify({ok: true, data: vals, address: address, rows: r` |
| `5752190` | `wps_presentation` | 77086 | 77 | — | `return JSON.stringify({ok: true, action: 'data_written', rows: data.le` |
| `5752397` | `wps_presentation` | 77293 | 68 | — | `return JSON.stringify({ok: true, values: vals, formulas: formulas});` |
| `5752777` | `wps_presentation` | 77673 | 96 | __OUTPUT_PATH__, __RANGE__ | `return JSON.stringify({ok: true, format: "png", range: "__RANGE__", ou` |
| `5752903` | `wps_presentation` | 77799 | 43 | __RANGE__, __VALUE__ | `ws.Range("__RANGE__").Value2 = "__VALUE__";` |
| `5752948` | `wps_presentation` | 77844 | 72 | __RANGE__ | `return JSON.stringify({ok: true, action: 'edited', range: "__RANGE__"}` |
| `5753252` | `wps_presentation` | 78148 | 60 | — | `for (var i = 1; i <= Application.Documents.Count; i++) {` |
| `5753314` | `wps_presentation` | 78210 | 44 | — | `var d = Application.Documents.Item(i);` |
| `5753441` | `wps_presentation` | 78337 | 78 | — | `if (!doc) doc = Application.Documents.Open(filePath, false, false, fal` |
| `5753981` | `wps_presentation` | 78877 | 78 | — | `var pdfDoc = Application.Documents.Open(filePath,false,false,"","",fal` |
| `5754579` | `wps_presentation` | 79475 | 64 | — | `for (var i = 1; i <= Application.Presentations.Count; i++) {` |
| `5754645` | `wps_presentation` | 79541 | 48 | — | `var p = Application.Presentations.Item(i);` |
| `5754791` | `wps_presentation` | 79687 | 84 | — | `if (!pres) pres = Application.Presentations.Open(filePath, false, fals` |
| `5755475` | `wps_presentation` | 80371 | 60 | — | `for (var i = 1; i <= Application.Workbooks.Count; i++) {` |
| `5755537` | `wps_presentation` | 80433 | 44 | — | `var w = Application.Workbooks.Item(i);` |
| `5755663` | `wps_presentation` | 80559 | 55 | — | `if (!wb) wb = Application.Workbooks.Open(filePath);` |
| `5759074` | `wps_presentation` | 83970 | 57 | — | `try { Application.DisplayAlerts = false; } catch (e) {}` |
| `5759631` | `wps_presentation` | 84527 | 46 | — | `var active = Application.ActiveWorkbook;` |
| `5759765` | `wps_presentation` | 84661 | 62 | — | `tried.push("Application.ActiveWorkbook: " + errText(e));` |
| `5759876` | `wps_presentation` | 84772 | 47 | — | `wb = Application.Workbooks.Open(srcPath);` |
| `5760564` | `wps_presentation` | 85460 | 74 | — | `return JSON.stringify({ ok: true, action: "created", from: srcPath });` |
| `5764830` | `wps_presentation` | 89726 | 339 | — | `either --script or --script-file requiredok--text and --from-file cann` |
| `5786807` | `wps_presentation` | 111703 | 42 | — | `var doc = Application.Documents.Add();` |
| `5804466` | `wps_presentation` | 129362 | 54 | — | `return JSON.stringify({ok: true, action: 'appended'});` |
| `5807467` | `wps_presentation` | 132363 | 66 | — | `return JSON.stringify({ok: true, type: 'tables', tables: tables});` |
| `5808238` | `wps_presentation` | 133134 | 86 | — | `return JSON.stringify({ ok: true, message: "success", data: { savePath` |
| `5808828` | `wps_presentation` | 133724 | 68 | — | `return JSON.stringify({ok: true, type: 'outline', items: headings});` |
| `5808934` | `wps_presentation` | 133830 | 76 | __OUTPUT_PATH__ | `return JSON.stringify({ok: true, format: 'txt', output: "__OUTPUT_PATH` |
| `5809581` | `wps_presentation` | 134477 | 49 | — | `return JSON.stringify({ok: true, filled: count});` |
| `5809853` | `wps_presentation` | 134749 | 68 | — | `return JSON.stringify({ok: true, type: 'metadata', metadata: meta});` |
| `5810228` | `wps_presentation` | 135124 | 72 | — | `return JSON.stringify({ok: true, issues: issues, total: issues.length}` |
| `5810747` | `wps_presentation` | 135643 | 53 | — | `return JSON.stringify({ok: true, action: 'created'});` |
| `5811558` | `wps_presentation` | 136454 | 120 | — | `return JSON.stringify({ok: true, format: "png", pages: total, exported` |
| `5813453` | `wps_presentation` | 138349 | 56 | — | `for (var i = 1; i <= Application.Documents.Count; i++) {` |
| `5813511` | `wps_presentation` | 138407 | 57 | — | `before[docKey(Application.Documents.Item(i))] = true;` |
| `5813939` | `wps_presentation` | 138835 | 56 | — | `for (var j = 1; j <= Application.Documents.Count; j++) {` |
| `5813997` | `wps_presentation` | 138893 | 42 | — | `var d = Application.Documents.Item(j);` |
| `5815348` | `wps_presentation` | 140244 | 51 | — | `return JSON.stringify({ok: true, replaced: count});` |
| `5816030` | `wps_presentation` | 140926 | 80 | — | `return JSON.stringify({ok: true, action: 'image_inserted', paragraph: ` |
| `5816113` | `wps_presentation` | 141009 | 71 | — | `return JSON.stringify({ok: true, action: 'image_inserted', char: idx})` |
| `5816359` | `wps_presentation` | 141255 | 63 | — | `return JSON.stringify({ok: true, action: 'inserted', char: n});` |
| `5816689` | `wps_presentation` | 141585 | 53 | — | `return JSON.stringify({ok: true, paragraphs: texts});` |
| `5817122` | `wps_presentation` | 142018 | 77 | — | `if (!src) src = Application.Documents.Open(srcPath, false, false, fals` |
| `5817438` | `wps_presentation` | 142334 | 57 | — | `return JSON.stringify({ok: true, action: "created"});` |
| `5817933` | `wps_presentation` | 142829 | 73 | — | `return JSON.stringify({ok: true, action: 'inserted', at_paragraph: idx` |
| `5818625` | `wps_presentation` | 143521 | 79 | — | `var pdfDoc = Application.Documents.Open(filePath, false, false, "", ""` |
| `5818836` | `wps_presentation` | 143732 | 99 | __COUNT__, __OUTPUT_PATH__ | `return JSON.stringify({ok: true, action: "merged", output: __OUTPUT_PA` |
| `5819021` | `wps_presentation` | 143917 | 82 | — | `if (!keyword) { return JSON.stringify({ok: false, error: "empty search` |
| `5819487` | `wps_presentation` | 144383 | 128 | — | `return JSON.stringify({ok: true, query: keyword, count: results.length` |
| `5819687` | `wps_presentation` | 144583 | 117 | __OUTPUT_DIR__, __PAGES_PER_FILE__ | `return JSON.stringify({ok: true, action: "split", pages_per_file: __PA` |
| `5820564` | `wps_presentation` | 145460 | 107 | __OUTPUT_PATH__, __PAGE_RANGE__ | `return JSON.stringify({ok: true, action: "extracted", pages: "__PAGE_R` |
| `5820949` | `wps_presentation` | 145845 | 50 | — | `return JSON.stringify({ok: true, metadata: meta});` |
| `5822166` | `wps_presentation` | 147062 | 68 | — | `return JSON.stringify({ok: true, bookmarks: collectOutlines(root)});` |
| `5822877` | `wps_presentation` | 147773 | 75 | __OUTPUT_DIR__ | `return JSON.stringify({ok: true, format: "png", output: "__OUTPUT_DIR_` |
| `5823235` | `wps_presentation` | 148131 | 72 | — | `return JSON.stringify({ok: true, count: fields.length, fields: fields}` |
| `5824077` | `wps_presentation` | 148973 | 48 | — | `return JSON.stringify({ok: true, pages: texts});` |
| `5824224` | `wps_presentation` | 149120 | 61 | — | `return JSON.stringify({ok: true, action: "watermark_added"});` |
| `5656768` | `wps_search` | 384 | 143 | — | `>(function(){ try { return JSON.stringify({ok:true, text: OCR('�N')});` |
| `5657152` | `wps_search` | 0 | 4000 | — | `wps_searchCheck WPS Office installation, engine connectivity, file ver` |
| `5672188` | `wps_writer` | 2887 | 76 | — | `var doc = Application.Documents.Open(templatePath, false, false, false` |
| `5672429` | `wps_writer` | 2646 | 78 | — | `return typeof __result === "string" ? __result : JSON.stringify(__resu` |
| `5672525` | `wps_writer` | 2550 | 70 | — | `return JSON.stringify({ok: false, error: e.message // String(e)});` |
| `5673294` | `wps_writer` | 1781 | 47 | — | `var pres = Application.Presentations.Add();` |
| `5674206` | `wps_writer` | 869 | 41 | — | `var wb = Application.Workbooks.Add();` |

## 没有 JS 片段的工具（= 纯 Rust 实现，不起 WPS 进程）

- `wps_browser`
- `wps_doc_check`
- `wps_doc_compare`
- `wps_doc_create`
- `wps_doc_edit`
- `wps_doc_export`
- `wps_doc_fill`
- `wps_doc_read`
- `wps_doc_script`
- `wps_file_info`
- `wps_image_extract`
- `wps_pdf_annotate`
- `wps_pdf_edit`
- `wps_pdf_export`
- `wps_pdf_extract`
- `wps_pdf_form`
- `wps_pdf_read`
- `wps_pdf_script`
- `wps_pdf_search`
- `wps_ppt_create`
- `wps_ppt_edit`
- `wps_ppt_export`
- `wps_ppt_read`
- `wps_ppt_script`
- `wps_ppt_watermark`
- `wps_sheet_chart`
- `wps_sheet_clean`
- `wps_sheet_create`
- `wps_sheet_edit`
- `wps_sheet_export`
- `wps_sheet_read`
- `wps_sheet_script`
- `wps_spreadsheet`
- `wps_status`

## 外壳归属

| 外壳 offset | 长度 | 归属工具 | 距离 |
|---|---|---|---|
| `5686765` | 568 | `wps_presentation` | 11661 |
| `5688030` | 109 | `wps_presentation` | 12926 |
| `5688261` | 427 | `wps_presentation` | 13157 |
| `5689237` | 393 | `wps_presentation` | 14133 |
| `5743603` | 115 | `wps_presentation` | 68499 |
| `5753069` | 771 | `wps_presentation` | 77965 |
| `5753920` | 473 | `wps_presentation` | 78816 |
| `5754395` | 823 | `wps_presentation` | 79291 |
| `5755293` | 743 | `wps_presentation` | 80189 |
| `5758445` | 2359 | `wps_presentation` | 83341 |
| `5765027` | 142 | `wps_presentation` | 89923 |
| `5786743` | 396 | `wps_presentation` | 111639 |
| `5816760` | 830 | `wps_presentation` | 141656 |
| `5836349` | 568 | `wps_presentation` | 161245 |
| `5837614` | 109 | `wps_presentation` | 162510 |
| `5837845` | 427 | `wps_presentation` | 162741 |
| `5838821` | 393 | `wps_presentation` | 163717 |
| `5847840` | 568 | `wps_presentation` | 172736 |
| `5849105` | 109 | `wps_presentation` | 174001 |
| `5849336` | 427 | `wps_presentation` | 174232 |
| `5850312` | 393 | `wps_presentation` | 175208 |
| `5656769` | 142 | `wps_search` | 383 |
| `5672038` | 568 | `wps_writer` | 3037 |
| `5673230` | 427 | `wps_writer` | 1845 |
| `5674142` | 393 | `wps_writer` | 933 |