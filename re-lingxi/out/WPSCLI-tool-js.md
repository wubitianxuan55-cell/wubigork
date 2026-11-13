# wpscli 内嵌 JS 片段按文档类型分组

- 源: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe`
- 片段 **129** 条，分组: doc=13、image=3、other=83、pdf=3、ppt=19、sheet=8

## doc（13）

| offset | 长度 | 片段开头 |
|---|---|---|
| `5672188` | 76 | `var doc = Application.Documents.Open(templatePath, false, false, false);` |
| `5721371` | 599 | `outlinemetadatatablesexpected numeric indexrange array must have 1 or 2 elementsinvalid ra` |
| `5721993` | 66 | `- �< + 1}, (_, i) => Application.ActiveDocument.Paragraphs.Item(�` |
| `5722632` | 64 | `- �: + 1}, (_, i) => Application.ActiveDocument.Sections.Item(�` |
| `5753252` | 60 | `for (var i = 1; i <= Application.Documents.Count; i++) {` |
| `5753314` | 44 | `var d = Application.Documents.Item(i);` |
| `5753441` | 78 | `if (!doc) doc = Application.Documents.Open(filePath, false, false, false);` |
| `5786807` | 42 | `var doc = Application.Documents.Add();` |
| `5813453` | 56 | `for (var i = 1; i <= Application.Documents.Count; i++) {` |
| `5813511` | 57 | `before[docKey(Application.Documents.Item(i))] = true;` |
| `5813939` | 56 | `for (var j = 1; j <= Application.Documents.Count; j++) {` |
| `5813997` | 42 | `var d = Application.Documents.Item(j);` |
| `5817122` | 77 | `if (!src) src = Application.Documents.Open(srcPath, false, false, false);` |

## sheet（8）

| offset | 长度 | 片段开头 |
|---|---|---|
| `5674206` | 41 | `var wb = Application.Workbooks.Add();` |
| `5749297` | 57 | `chartObj.Chart.SetSourceData(ws.Range("__DATA_RANGE__"));` |
| `5749537` | 46 | `ws.Range("__RANGE__").Formula = "__FORMULA__";` |
| `5752903` | 43 | `ws.Range("__RANGE__").Value2 = "__VALUE__";` |
| `5755475` | 60 | `for (var i = 1; i <= Application.Workbooks.Count; i++) {` |
| `5755537` | 44 | `var w = Application.Workbooks.Item(i);` |
| `5755663` | 55 | `if (!wb) wb = Application.Workbooks.Open(filePath);` |
| `5759876` | 47 | `wb = Application.Workbooks.Open(srcPath);` |

## ppt（19）

| offset | 长度 | 片段开头 |
|---|---|---|
| `5657152` | 4000 | `wps_searchCheck WPS Office installation, engine connectivity, file version (> 12.1.0.27880` |
| `5673294` | 47 | `var pres = Application.Presentations.Add();` |
| `5723401` | 47 | `if (idx < 0) idx = pres.Slides.Count + idx + 1;` |
| `5723450` | 92 | `if (idx < 1 // idx > pres.Slides.Count) throw new Error('slide index out of range: ' + idx` |
| `5724697` | 40 | `var slide = pres.Slides.Item(__SLIDE__);` |
| `5725011` | 46 | `for (var i = 1; i <= pres.Slides.Count; i++) {` |
| `5725519` | 81 | `return JSON.stringify({ok: true, slideCount: pres.Slides.Count, slides: slides});` |
| `5731738` | 41 | `if (s < 0) s = pres.Slides.Count + s + 1;` |
| `5731780` | 41 | `if (e < 0) e = pres.Slides.Count + e + 1;` |
| `5731904` | 103 | `throw new Error('slide range out of range: ' + s + ':' + e + ' (count=' + pres.Slides.Coun` |
| `5732973` | 98 | `if (slideIndex < 1 // slideIndex > pres.Slides.Count) throw new Error('slide index out of ` |
| `5733092` | 41 | `var slide = pres.Slides.Item(slideIndex);` |
| `5739226` | 63 | `var slide = pres.Slides.Add(pres.Slides.Count + 1, layout);` |
| `5741728` | 52 | `__PH_INDEX__var slide = pres.Slides.Item(__SLIDE__);` |
| `5742122` | 109 | `throw new Error('slide notes range out of range: ' + s + ':' + e + ' (count=' + pres.Slide` |
| `5743317` | 75 | `slides.push({index: i, notes: readNotesForSlide(pres.Slides.Item(i))});` |
| `5754579` | 64 | `for (var i = 1; i <= Application.Presentations.Count; i++) {` |
| `5754645` | 48 | `var p = Application.Presentations.Item(i);` |
| `5754791` | 84 | `if (!pres) pres = Application.Presentations.Open(filePath, false, false, false);` |

## pdf（3）

| offset | 长度 | 片段开头 |
|---|---|---|
| `5724455` | 76 | `return JSON.stringify({ok: true, format: 'pdf', output: "__OUTPUT_PATH__"});` |
| `5753981` | 78 | `var pdfDoc = Application.Documents.Open(filePath,false,false,"","",false);` |
| `5818625` | 79 | `var pdfDoc = Application.Documents.Open(filePath, false, false, "", "", false);` |

## image（3）

| offset | 长度 | 片段开头 |
|---|---|---|
| `5656768` | 143 | `>(function(){ try { return JSON.stringify({ok:true, text: OCR('�N')}); } catch(e) { return` |
| `5733282` | 64 | `slide.Shapes.AddPicture(image, 0, -1, left, top, width, height);` |
| `5764830` | 339 | `either --script or --script-file requiredok--text and --from-file cannot be used togetherm` |

## other（83）

| offset | 长度 | 片段开头 |
|---|---|---|
| `5672429` | 78 | `return typeof __result === "string" ? __result : JSON.stringify(__result);` |
| `5672525` | 70 | `return JSON.stringify({ok: false, error: e.message // String(e)});` |
| `5688030` | 109 | `(function() { try { �W } catch(e) { return JSON.stringify({ok: false, error: e.message // ` |
| `5723598` | 47 | `for (var j = 1; j <= slide.Shapes.Count; j++) {` |
| `5723960` | 62 | `return JSON.stringify({ok: true, slide: idx, shapes: shapes});` |
| `5724318` | 83 | `return JSON.stringify({ ok: true, format: "markdown", output: "__OUTPUT_PATH__" });` |
| `5724403` | 50 | `__OUTPUT_PATH__pres.SaveAs("__OUTPUT_PATH__", 32);` |
| `5724570` | 77 | `return JSON.stringify({ok: true, format: 'html', output: "__OUTPUT_PATH__"});` |
| `5724886` | 72 | `return JSON.stringify({ok: true, action: 'text_set', slide: __SLIDE__});` |
| `5725119` | 51 | `for (var j = 1; j <= slide.Shapes.Count; j++) {` |
| `5725172` | 41 | `var shape = slide.Shapes.Item(j);` |
| `5725420` | 94 | `slides.push({index: i, name: slide.Name, shapeCount: slide.Shapes.Count, shapes: shapes});` |
| `5727353` | 43 | `var win = Application.ActiveWindow;` |
| `5728501` | 88 | `return JSON.stringify({ok: true, message: "success", data: {removed: removed}});` |
| `5728596` | 80 | `return JSON.stringify({ok: false, message: "母版中未找到水印", data: {removed: 0}});` |
| `5729026` | 85 | `return JSON.stringify({ok: false, message: "母版中已存在水印，请先移除后再添加", data: null});` |
| `5733347` | 79 | `return JSON.stringify({ok: true, action: 'image_inserted', slide: slideIndex});` |
| `5733925` | 93 | `try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = "__TITLE__"; } catch(e)` |
| `5734020` | 96 | `try { slide.Shapes.Placeholders.Item(2).TextFrame.TextRange.Text = "__SUBTITLE__"; } catch` |
| `5734118` | 59 | `return JSON.stringify({ok: true, action: 'slide_created'});` |
| `5736339` | 53 | `for (var i = slideMaster.Shapes.Count; i >= 1; i--) {` |
| `5736624` | 42 | `var shape = slideMaster.Shapes.AddTextbox(` |
| `5739291` | 99 | `try { slide.Shapes.Placeholders.Item(1).TextFrame.TextRange.Text = title // ""; } catch (e` |
| `5739424` | 75 | `var tr = slide.Shapes.Placeholders.Item(2).TextFrame.TextRange;` |
| `5741548` | 54 | `var ph = slide.Shapes.Placeholders.Item(__PH_INDEX__);` |
| `5741647` | 79 | `return JSON.stringify({ok: true, action: 'placeholder_set', slide: __SLIDE__});` |
| `5741823` | 94 | `return JSON.stringify({ok: true, format: 'png', slide: __SLIDE__, output: "__OUTPUT_PATH__` |
| `5742679` | 69 | `notes = shapeText(notesPage.Shapes.Placeholders.Item(2));` |
| `5742798` | 63 | `for (var i = 1; i <= notesPage.Shapes.Count; i++) {` |
| `5742863` | 50 | `var sh = notesPage.Shapes.Item(i);` |
| `5743415` | 72 | `return JSON.stringify({ok: true, slide: s, notes: slides[0].notes});` |
| `5743492` | 93 | `return JSON.stringify({ok: true, from: s, to: e, slideCount: slides.length, slides: slides` |
| `5743587` | 131 | `"remove"__BODY__(function() { try { __BODY__ } catch(e) { return JSON.stringify({ok: false` |
| `5747155` | 85 | `if (vals === null // vals === undefined) return JSON.stringify({ok: true, data: []});` |
| `5747242` | 76 | `if (!Array.isArray(vals)) return JSON.stringify({ok: true, data: [[vals]]});` |
| `5747320` | 46 | `return JSON.stringify({ok: true, data: vals});` |
| `5749017` | 88 | `return JSON.stringify({ok: true, type: "charts", count: charts.length, charts: charts});` |
| `5749420` | 59 | `return JSON.stringify({ok: true, action: 'chart_created'});` |
| `5749585` | 77 | `return JSON.stringify({ok: true, action: 'formula_set', range: "__RANGE__"});` |
| `5750298` | 98 | `return JSON.stringify({ ok: true, message: "success", data: { path: path, format: "markdow` |
| `5750516` | 76 | `return JSON.stringify({ok: true, format: "csv", output: "__OUTPUT_PATH__"});` |
| `5750784` | 49 | `return JSON.stringify({ok: true, sheets: names});` |
| `5750899` | 81 | `return JSON.stringify({ok: true, data: [], address: null, rows: 0, cols: 0});` |
| `5751367` | 90 | `return JSON.stringify({ok: true, data: [], address: address, rows: rows, cols: cols});` |
| `5751491` | 90 | `return JSON.stringify({ok: true, data: [[vals]], address: address, rows: 1, cols: 1});` |
| `5751691` | 114 | `return JSON.stringify({ok: true, data: [vals], address: address, rows: rows // 1, cols: co` |
| `5751810` | 88 | `return JSON.stringify({ok: true, data: vals, address: address, rows: rows, cols: cols});` |
| `5752190` | 77 | `return JSON.stringify({ok: true, action: 'data_written', rows: data.length});` |
| `5752397` | 68 | `return JSON.stringify({ok: true, values: vals, formulas: formulas});` |
| `5752777` | 96 | `return JSON.stringify({ok: true, format: "png", range: "__RANGE__", output: "__OUTPUT_PATH` |
| `5752948` | 72 | `return JSON.stringify({ok: true, action: 'edited', range: "__RANGE__"});` |
| `5759074` | 57 | `try { Application.DisplayAlerts = false; } catch (e) {}` |
| `5759631` | 46 | `var active = Application.ActiveWorkbook;` |
| `5759765` | 62 | `tried.push("Application.ActiveWorkbook: " + errText(e));` |
| `5760564` | 74 | `return JSON.stringify({ ok: true, action: "created", from: srcPath });` |
| `5804466` | 54 | `return JSON.stringify({ok: true, action: 'appended'});` |
| `5807467` | 66 | `return JSON.stringify({ok: true, type: 'tables', tables: tables});` |
| `5808238` | 86 | `return JSON.stringify({ ok: true, message: "success", data: { savePath: savePath } });` |
| `5808828` | 68 | `return JSON.stringify({ok: true, type: 'outline', items: headings});` |
| `5808934` | 76 | `return JSON.stringify({ok: true, format: 'txt', output: "__OUTPUT_PATH__"});` |
| `5809581` | 49 | `return JSON.stringify({ok: true, filled: count});` |
| `5809853` | 68 | `return JSON.stringify({ok: true, type: 'metadata', metadata: meta});` |
| `5810228` | 72 | `return JSON.stringify({ok: true, issues: issues, total: issues.length});` |
| `5810747` | 53 | `return JSON.stringify({ok: true, action: 'created'});` |
| `5811558` | 120 | `return JSON.stringify({ok: true, format: "png", pages: total, exported: files.length, outp` |
| `5815348` | 51 | `return JSON.stringify({ok: true, replaced: count});` |
| `5816030` | 80 | `return JSON.stringify({ok: true, action: 'image_inserted', paragraph: idx});` |
| `5816113` | 71 | `return JSON.stringify({ok: true, action: 'image_inserted', char: idx});` |
| `5816359` | 63 | `return JSON.stringify({ok: true, action: 'inserted', char: n});` |
| `5816689` | 53 | `return JSON.stringify({ok: true, paragraphs: texts});` |
| `5817438` | 57 | `return JSON.stringify({ok: true, action: "created"});` |
| `5817933` | 73 | `return JSON.stringify({ok: true, action: 'inserted', at_paragraph: idx});` |
| `5818836` | 99 | `return JSON.stringify({ok: true, action: "merged", output: __OUTPUT_PATH__, fileCount: __C` |
| `5819021` | 82 | `if (!keyword) { return JSON.stringify({ok: false, error: "empty search query"}); }` |
| `5819487` | 128 | `return JSON.stringify({ok: true, query: keyword, count: results.length, truncated: results` |
| `5819687` | 117 | `return JSON.stringify({ok: true, action: "split", pages_per_file: __PAGES_PER_FILE__, outp` |
| `5820564` | 107 | `return JSON.stringify({ok: true, action: "extracted", pages: "__PAGE_RANGE__", output: "__` |
| `5820949` | 50 | `return JSON.stringify({ok: true, metadata: meta});` |
| `5822166` | 68 | `return JSON.stringify({ok: true, bookmarks: collectOutlines(root)});` |
| `5822877` | 75 | `return JSON.stringify({ok: true, format: "png", output: "__OUTPUT_DIR__"});` |
| `5823235` | 72 | `return JSON.stringify({ok: true, count: fields.length, fields: fields});` |
| `5824077` | 48 | `return JSON.stringify({ok: true, pages: texts});` |
| `5824224` | 61 | `return JSON.stringify({ok: true, action: "watermark_added"});` |
