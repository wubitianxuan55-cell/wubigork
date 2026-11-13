# wpscli：真 JS 语句片段（收紧判定）

- 源: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe`
- 候选 300 → 去重 **213** 条（要求含 `var/return/function` + 括号 + 分号，且中文占比 < 12%）

## 按文档类型

| 类型 | 片段数 | 对象模型入口 | 方法调用数 | 属性赋值数 |
|---|---|---|---|---|
| other | 173 | 2 | 29 | 2 |
| doc | 13 | 9 | 9 | 0 |
| ppt | 11 | 4 | 7 | 0 |
| sheet | 7 | 5 | 4 | 0 |
| pdf | 7 | 2 | 7 | 0 |
| image | 2 | 0 | 0 | 0 |

## other（173）

方法：`Item`、`Close`、`Save`、`AddTextbox`、`ChartObjects`、`Add`、`GetWorkbookEx`、`Address`、`Cell`、`GetDocumentEx`、`Children`

属性赋值：`Saved`、`DisplayAlerts`


## doc（13）

方法：`Item`、`Open`、`Paragraphs`、`Add`


## ppt（11）

方法：`Item`、`Add`、`Open`


## sheet（7）

方法：`Open`、`Add`、`Item`


## pdf（7）

方法：`Close`、`Open`、`Save`、`PageText`、`ExportToMarkdown`


## image（2）


## 片段清单

| offset | 类型 | 开头 |
|---|---|---|
| `5656768` | image | `>(function(){ try { return JSON.stringify({ok:true, text: OCR('�N')}); } catch(e) { return JSON.stri` |
| `5657152` | ppt | `wps_searchCheck WPS Office installation, engine connectivity, file version (> 12.1.0.27880), and API` |
| `5672188` | doc | `var doc = Application.Documents.Open(templatePath, false, false, false);` |
| `5672325` | other | `var __result = (function(doc) { __BODY__ })(doc);` |
| `5672429` | other | `return typeof __result === "string" ? __result : JSON.stringify(__result);` |
| `5672525` | other | `return JSON.stringify({ok: false, error: e.message // String(e)});` |
| `5673294` | ppt | `var pres = Application.Presentations.Add();` |
| `5673343` | other | `var __result = (function(pres) { __BODY__ })(pres);` |
| `5674206` | sheet | `var wb = Application.Workbooks.Add();` |
| `5674249` | other | `var __result = (function(wb) { __BODY__ })(wb);` |
| `5688030` | other | `(function() { try { �W } catch(e) { return JSON.stringify({ok: false, error: e.message // String(e)}` |
| `5723960` | other | `return JSON.stringify({ok: true, slide: idx, shapes: shapes});` |
| `5724318` | other | `return JSON.stringify({ ok: true, format: "markdown", output: "__OUTPUT_PATH__" });` |
| `5724455` | other | `return JSON.stringify({ok: true, format: 'pdf', output: "__OUTPUT_PATH__"});` |
| `5724570` | other | `return JSON.stringify({ok: true, format: 'html', output: "__OUTPUT_PATH__"});` |
| `5724697` | ppt | `var slide = pres.Slides.Item(__SLIDE__);` |
| `5724886` | other | `return JSON.stringify({ok: true, action: 'text_set', slide: __SLIDE__});` |
| `5725011` | ppt | `for (var i = 1; i <= pres.Slides.Count; i++) {` |
| `5725119` | other | `for (var j = 1; j <= slide.Shapes.Count; j++) {` |
| `5725172` | other | `var shape = slide.Shapes.Item(j);` |
| `5725519` | ppt | `return JSON.stringify({ok: true, slideCount: pres.Slides.Count, slides: slides});` |
| `5726061` | other | `if (color === undefined // color === null) return WATERMARK_FONT_COLOR;` |
| `5726137` | other | `if (typeof color === "number") return color;` |
| `5726186` | other | `var hex = ("" + color).replace(/^#/, "");` |
| `5726333` | other | `var r = parseInt(hex.substring(0, 2), 16);` |
| `5726380` | other | `var g = parseInt(hex.substring(2, 4), 16);` |
| `5726427` | other | `var b = parseInt(hex.substring(4, 6), 16);` |
| `5726562` | other | `if (transparency === undefined // transparency === null) return WATERMARK_FONT_TRANSPARENCY;` |
| `5726659` | other | `var value = parseFloat(transparency);` |
| `5726701` | other | `if (isNaN(value)) return WATERMARK_FONT_TRANSPARENCY;` |
| `5727149` | other | `if (!name // name.indexOf(WATERMARK_TAG_PREFIX) !== 0) return false;` |
| `5727222` | other | `return /^\d+$/.test(name.slice(WATERMARK_TAG_PREFIX.length));` |
| `5727657` | other | `var fontColor = (typeof _fontColorRaw === "number") ? resolveFontColor(_fontColorRaw) : WATERMARK_FO` |
| `5727767` | other | `var fontTransparency = (typeof _fontTransparencyRaw === "number") ? resolveFontTransparency(_fontTra` |
| `5728270` | other | `for (var i = shapes.Count; i >= 1; i--) {` |
| `5728501` | other | `return JSON.stringify({ok: true, message: "success", data: {removed: removed}});` |
| `5728596` | other | `return JSON.stringify({ok: false, message: "母版中未找到水印", data: {removed: 0}});` |
| `5728921` | other | `for (var i = 1; i <= shapes.Count; i++) {` |
| `5729163` | other | `var rotationRad = Math.abs(rotationDeg) * Math.PI / 180;` |
| `5729272` | other | `var tmpShape = shapes.AddTextbox(1, 0, 0, 500, 100);` |
| `5729790` | other | `var bboxWidth = Math.ceil(textWidth * cosA + textHeight * sinA);` |
| `5729855` | other | `var bboxHeight = Math.ceil(textWidth * sinA + textHeight * cosA);` |
| `5729922` | other | `var cols = Math.floor(slideWidth / bboxWidth);` |
| `5729969` | other | `var rows = Math.floor(slideHeight / bboxHeight);` |
| `5730067` | other | `var gapX = (slideWidth - cols * bboxWidth) / (cols + 1);` |
| `5730124` | other | `var gapY = (slideHeight - rows * bboxHeight) / (rows + 1);` |
| `5730238` | other | `for (var col = 0; col < cols; col++) {` |
| `5730281` | other | `var cellLeft = gapX + col * (bboxWidth + gapX);` |
| `5730337` | other | `var cellTop = gapY + row * (bboxHeight + gapY);` |
| `5730394` | other | `var shape = shapes.AddTextbox(1, 0, 0, 500, 100);` |
| `5731822` | other | `if (s > e) { var tmp = s; s = e; e = tmp; }` |
| `5733092` | ppt | `var slide = pres.Slides.Item(slideIndex);` |
| `5733347` | other | `return JSON.stringify({ok: true, action: 'image_inserted', slide: slideIndex});` |
| `5734118` | other | `return JSON.stringify({ok: true, action: 'slide_created'});` |
| `5736339` | other | `for (var i = slideMaster.Shapes.Count; i >= 1; i--) {` |
| `5736538` | other | `var left = (slideWidth - shapeWidth) / 2;` |
| `5736580` | other | `var top = (slideHeight - shapeHeight) / 2;` |
| `5738003` | other | `for (var b = 0; b < item.bullets.length; b++) {` |
| `5738290` | other | `return { text: parts.join("\r"), asBullets: true };` |
| `5738750` | doc | `var para = textRange.Paragraphs(i);` |
| `5738799` | other | `var t = String(para.Text // "").replace(/^\s+/\s+$/g, "").replace(/\r/g, "");` |
| `5739226` | ppt | `var slide = pres.Slides.Add(pres.Slides.Count + 1, layout);` |
| `5739424` | other | `var tr = slide.Shapes.Placeholders.Item(2).TextFrame.TextRange;` |
| `5739841` | other | `for (var i = 0; i < items.length; i++) {` |
| `5740032` | other | `var title = item && item.title != null ? String(item.title) : "";` |
| `5740275` | other | `var lines = String(text).split("\n");` |
| `5740569` | other | `for (var i = 0; i < lines.length; i++) {` |
| `5740615` | other | `var line = lines[i].replace(/\r/g, "").trim();` |
| `5740701` | other | `var heading = line.match(/^(#{1,6})\s+(.*)/);` |
| `5741049` | other | `var trimmed = String(raw).replace(/^\s+/\s+$/g, "");` |
| `5741548` | other | `var ph = slide.Shapes.Placeholders.Item(__PH_INDEX__);` |
| `5741647` | other | `return JSON.stringify({ok: true, action: 'placeholder_set', slide: __SLIDE__});` |
| `5741728` | ppt | `__PH_INDEX__var slide = pres.Slides.Item(__SLIDE__);` |
| `5741823` | other | `return JSON.stringify({ok: true, format: 'png', slide: __SLIDE__, output: "__OUTPUT_PATH__"});` |
| `5742340` | other | `return String(shape.TextFrame.TextRange.Text // '');` |
| `5742798` | other | `for (var i = 1; i <= notesPage.Shapes.Count; i++) {` |
| `5742863` | other | `var sh = notesPage.Shapes.Item(i);` |
| `5743415` | other | `return JSON.stringify({ok: true, slide: s, notes: slides[0].notes});` |
| `5743492` | other | `return JSON.stringify({ok: true, from: s, to: e, slideCount: slides.length, slides: slides});` |
| `5743587` | other | `"remove"__BODY__(function() { try { __BODY__ } catch(e) { return JSON.stringify({ok: false, error: e` |
| `5743720` | other | `var ops = JSON.parse("__OPERATIONS_JSON__");` |
| `5743892` | other | `if (v === null // v === undefined) return true;` |
| `5743945` | other | `if (typeof v === "string" && String(v).replace(/^\s+/\s+$/g, "") === "") return true;` |
| `5744466` | other | `var v = ws.Cells.Item(row, firstCol + c).Value2;` |
| `5744524` | other | `if (!isBlankValue(v)) return false;` |
| `5744998` | other | `if (typeof op === "string") return op;` |
| `5745042` | other | `if (op && typeof op.type === "string") return op.type;` |
| `5745281` | other | `for (var r = b.firstRow; r <= b.lastRow; r++) {` |
| `5745338` | other | `for (var c = b.firstCol; c <= b.lastCol; c++) {` |
| `5745399` | other | `var cell = ws.Cells.Item(r, c);` |
| `5745504` | other | `var t = cell.Value2.replace(/^\s+/\s+$/g, "");` |
| `5745923` | other | `for (var rr = b2.lastRow; rr >= b2.firstRow; rr--) {` |
| `5746574` | other | `var key = rowKey(rd, b3.firstCol, b3.cols);` |
| `5747155` | other | `if (vals === null // vals === undefined) return JSON.stringify({ok: true, data: []});` |
| `5747242` | other | `if (!Array.isArray(vals)) return JSON.stringify({ok: true, data: [[vals]]});` |
| `5747320` | other | `return JSON.stringify({ok: true, data: vals});` |
| `5747394` | sheet | `for (var s = 1; s <= wb.Worksheets.Count; s++) {` |
| `5747972` | other | `return (t != null && map[String(t)]) ? map[String(t)] : null;` |
| `5749017` | other | `return JSON.stringify({ok: true, type: "charts", count: charts.length, charts: charts});` |
| `5749240` | other | `var chartObj = ws.ChartObjects().Add(10, 10, 400, 300);` |
| `5749420` | other | `return JSON.stringify({ok: true, action: 'chart_created'});` |
| `5749585` | other | `return JSON.stringify({ok: true, action: 'formula_set', range: "__RANGE__"});` |
| `5749836` | other | `var result = exportSheetData(wb, { path: path, format: "markdown" });` |
| `5750061` | other | `var bookEx = wb.GetWorkbookEx && wb.GetWorkbookEx();` |
| `5750298` | other | `return JSON.stringify({ ok: true, message: "success", data: { path: path, format: "markdown" } });` |
| `5750516` | other | `return JSON.stringify({ok: true, format: "csv", output: "__OUTPUT_PATH__"});` |
| `5750629` | sheet | `for (var i = 1; i <= wb.Worksheets.Count; i++) {` |
| `5750784` | other | `return JSON.stringify({ok: true, sheets: names});` |
| `5750899` | other | `return JSON.stringify({ok: true, data: [], address: null, rows: 0, cols: 0});` |
| `5751061` | other | `if (typeof a === "function") a = ur.Address();` |
| `5751367` | other | `return JSON.stringify({ok: true, data: [], address: address, rows: rows, cols: cols});` |
| `5751491` | other | `return JSON.stringify({ok: true, data: [[vals]], address: address, rows: 1, cols: 1});` |
| `5751691` | other | `return JSON.stringify({ok: true, data: [vals], address: address, rows: rows // 1, cols: cols // vals` |
| `5751810` | other | `return JSON.stringify({ok: true, data: vals, address: address, rows: rows, cols: cols});` |
| `5752063` | other | `for (var c = 0; c < row.length; c++) {` |
| `5752190` | other | `return JSON.stringify({ok: true, action: 'data_written', rows: data.length});` |
| `5752397` | other | `return JSON.stringify({ok: true, values: vals, formulas: formulas});` |
| `5752553` | other | `var chartObj = ws.ChartObjects().Add(0, 0, rng.Width, rng.Height);` |
| `5752777` | other | `return JSON.stringify({ok: true, format: "png", range: "__RANGE__", output: "__OUTPUT_PATH__"});` |
| `5752948` | other | `return JSON.stringify({ok: true, action: 'edited', range: "__RANGE__"});` |
| `5753022` | other | `__VALUE__doc.Close(0);doc.Save(); doc.Close(0);(function() {` |
| `5753170` | other | `var want = filePath.replace(/\//g, "\\").toLowerCase();` |
| `5753252` | doc | `for (var i = 1; i <= Application.Documents.Count; i++) {` |
| `5753314` | doc | `var d = Application.Documents.Item(i);` |
| `5753441` | doc | `if (!doc) doc = Application.Documents.Open(filePath, false, false, false);` |
| `5753842` | pdf | `__FILE_PATH____SAVE_AND_CLOSE__pdfDoc.Close(0);pdfDoc.Save(); pdfDoc.Close(0);(function() {` |
| `5753981` | pdf | `var pdfDoc = Application.Documents.Open(filePath,false,false,"","",false);` |
| `5754131` | pdf | `var __result = (function(pdfDoc) { __BODY__ })(pdfDoc);` |
| `5754579` | ppt | `for (var i = 1; i <= Application.Presentations.Count; i++) {` |
| `5754645` | ppt | `var p = Application.Presentations.Item(i);` |
| `5754791` | ppt | `if (!pres) pres = Application.Presentations.Open(filePath, false, false, false);` |
| `5755220` | other | `pres.Saved = true;pres.Save();wb.Close(false);wb.Save(); wb.Close(false);(function() {` |
| `5755475` | sheet | `for (var i = 1; i <= Application.Workbooks.Count; i++) {` |
| `5755537` | sheet | `var w = Application.Workbooks.Item(i);` |
| `5755663` | sheet | `if (!wb) wb = Application.Workbooks.Open(filePath);` |
| `5758879` | other | `function normPath(p) { return String(p // "").replace(/\\/g, "/").replace(/^file:\/\//, "").toLowerC` |
| `5758990` | other | `function errText(e) { return String((e && (e.message // e.description)) // e); }` |
| `5759074` | other | `try { Application.DisplayAlerts = false; } catch (e) {}` |
| `5759765` | other | `tried.push("Application.ActiveWorkbook: " + errText(e));` |
| `5759876` | sheet | `wb = Application.Workbooks.Open(srcPath);` |
| `5760564` | other | `return JSON.stringify({ ok: true, action: "created", from: srcPath });` |
| `5764830` | image | `either --script or --script-file requiredok--text and --from-file cannot be used togethermessagesugg` |
| `5786807` | doc | `var doc = Application.Documents.Add();` |
| `5804466` | other | `return JSON.stringify({ok: true, action: 'appended'});` |
| `5804554` | other | `return String(s).replace(/^\s+/\s+$/g, '');` |
| `5804636` | other | `return String(s).replace(/\r/\x07/g, '');` |
| `5804840` | other | `for (var n = 1; n <= cell.Tables.Count; n++) {` |
| `5804892` | other | `var nr = cell.Tables.Item(n).Range;` |
| `5804937` | other | `if (pStart >= nr.Start && pStart < nr.End) return true;` |
| `5805335` | other | `try { return trimText(cleanCellRaw(cell.Range.Text)); }` |
| `5805517` | other | `for (var i = 1; i <= paras.Count; i++) {` |
| `5805658` | other | `var t = trimText(cleanCellRaw(p.Range.Text));` |
| `5806019` | other | `for (var c = 1; c <= cols; c++) {` |
| `5806081` | other | `var cell = tbl.Cell(r, c);` |
| `5806274` | other | `for (var n = 1; n <= nc; n++) {` |
| `5806323` | other | `var child = readTableMatrix(cell.Tables.Item(n));` |
| `5806972` | other | `for (var t = 1; t <= doc.Tables.Count; t++) {` |
| `5807467` | other | `return JSON.stringify({ok: true, type: 'tables', tables: tables});` |
| `5807709` | other | `var result = exportDocumentAsMarkdown({ savePath: savePath }, doc);` |
| `5807941` | other | `var docEx = doc.GetDocumentEx && doc.GetDocumentEx();` |
| `5808238` | other | `return JSON.stringify({ ok: true, message: "success", data: { savePath: savePath } });` |
| `5808361` | doc | `for (var i = 1; i <= doc.Paragraphs.Count; i++) {` |
| `5808481` | other | `var name = (typeof style === 'object' && style.NameLocal) ? style.NameLocal : String(style);` |
| `5808828` | other | `return JSON.stringify({ok: true, type: 'outline', items: headings});` |
| `5808934` | other | `return JSON.stringify({ok: true, format: 'txt', output: "__OUTPUT_PATH__"});` |
| `5809581` | other | `return JSON.stringify({ok: true, filled: count});` |
| `5809705` | other | `for (var i = 1; i <= props.Count; i++) {` |
| `5809853` | other | `return JSON.stringify({ok: true, type: 'metadata', metadata: meta});` |
| `5810029` | other | `var text = p.Range.Text.replace(/\r/g, '');` |
| `5810228` | other | `return JSON.stringify({ok: true, issues: issues, total: issues.length});` |
| `5810661` | doc | `return doc.Paragraphs.Item(idx).Range.Text;` |
| `5810747` | other | `return JSON.stringify({ok: true, action: 'created'});` |
| `5810969` | other | `var pages = doc.Windows.Item(1).Panes.Item(1).Pages;` |
| `5811311` | other | `var sep = /[\\\/]$/.test(outDir) ? "" : (outDir.indexOf("\\") >= 0 ? "\\" : "/");` |
| `5811558` | other | `return JSON.stringify({ok: true, format: "png", pages: total, exported: files.length, outputDir: out` |
| `5812356` | other | `return String(d.Name // "") + "/" + full;` |
| `5812433` | other | `return String(s // "").replace(/\r/\x07/g, "").replace(/^\s+/\s+$/g, "");` |
| `5812594` | other | `try { count = target.Revisions.Count; } catch (e) { return { count: 0, revisions: [] }; }` |
| `5812784` | other | `var r = target.Revisions.Item(i);` |
| `5813511` | doc | `before[docKey(Application.Documents.Item(i))] = true;` |
| `5813939` | doc | `for (var j = 1; j <= Application.Documents.Count; j++) {` |
| `5813997` | doc | `var d = Application.Documents.Item(j);` |
| `5814603` | other | `for (var k = 0; k < created.length; k++) {` |
| `5815348` | other | `return JSON.stringify({ok: true, replaced: count});` |
| `5816030` | other | `return JSON.stringify({ok: true, action: 'image_inserted', paragraph: idx});` |
| `5816113` | other | `return JSON.stringify({ok: true, action: 'image_inserted', char: idx});` |
| `5816359` | other | `return JSON.stringify({ok: true, action: 'inserted', char: n});` |
| `5816573` | doc | `for (var i = s; i <= e && i <= doc.Paragraphs.Count; i++) {` |
| `5816689` | other | `return JSON.stringify({ok: true, paragraphs: texts});` |
| `5816858` | other | `var want = srcPath.replace(/\//g, "\\").toLowerCase();` |
| `5817122` | doc | `if (!src) src = Application.Documents.Open(srcPath, false, false, false);` |
| `5817438` | other | `return JSON.stringify({ok: true, action: "created"});` |
| `5817933` | other | `return JSON.stringify({ok: true, action: 'inserted', at_paragraph: idx});` |
| `5818625` | pdf | `var pdfDoc = Application.Documents.Open(filePath, false, false, "", "", false);` |
| `5818836` | other | `return JSON.stringify({ok: true, action: "merged", output: __OUTPUT_PATH__, fileCount: __COUNT__});` |
| `5819021` | other | `if (!keyword) { return JSON.stringify({ok: false, error: "empty search query"}); }` |
| `5819124` | pdf | `for (var p = 0; p < pdfDoc.PageCount; p++) {` |
| `5819170` | pdf | `var text = pdfDoc.PageText(p) // "";` |
| `5819270` | other | `var idx = text.indexOf(keyword, from);` |
| `5819487` | other | `return JSON.stringify({ok: true, query: keyword, count: results.length, truncated: results.length >=` |
| `5819687` | other | `return JSON.stringify({ok: true, action: "split", pages_per_file: __PAGES_PER_FILE__, output_dir: "_` |
| `5820220` | pdf | `var ok = pdfDoc.Exports.ExportToMarkdown(outputPath, startPage, endPage);` |
| `5820564` | other | `return JSON.stringify({ok: true, action: "extracted", pages: "__PAGE_RANGE__", output: "__OUTPUT_PAT` |
| `5820949` | other | `return JSON.stringify({ok: true, metadata: meta});` |
| `5821076` | other | `var pageRange = pageCount > 0 ? ("1-" + pageCount) : "";` |
| `5821591` | other | `var child = outline.Children(i);` |
| `5822166` | other | `return JSON.stringify({ok: true, bookmarks: collectOutlines(root)});` |
| `5822877` | other | `return JSON.stringify({ok: true, format: "png", output: "__OUTPUT_DIR__"});` |
| `5823235` | other | `return JSON.stringify({ok: true, count: fields.length, fields: fields});` |
| `5823649` | other | `var startIndex = toPageIndex(start1based);` |
| `5823992` | other | `for (var i = startIndex; i <= endIndex; i++) {` |
| `5824077` | other | `return JSON.stringify({ok: true, pages: texts});` |
| `5824224` | other | `return JSON.stringify({ok: true, action: "watermark_added"});` |