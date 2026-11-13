# wpscli.exe 内嵌 JSAPI 脚本（静态抽取）

- 二进制: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe`（sha256 `796966c1…`，见 REA Evidence `ev_bd7498ba…`）
- 命中 IIFE 块 25 个，去重后 **13 个**
- 抽取方式：二进制内 UTF-8 字符串做括号配对（跳过字符串字面量），未执行任何 WPS 进程
- 占位符契约（66）：`__ACTION__`、`__ACTIVATE_SHEET__`、`__BODY__`、`__BODY____OUTPUT_PATH__`、`__BODY____TEMPLATE_PATH____OUTPUT_PATH__`、`__CHART_TYPE__`、`__CHAR__`、`__COLLECT_CHARTS__`、`__COUNT__`、`__DATA_JSON__`、`__DATA_RANGE__`、`__DATA_RANGE____CHART_TYPE__`、`__END_PAGE__`、`__END__`、`__FILE_PATH__`、`__FILE_PATH____SAVE_AND_CLOSE__`、`__FIND__`、`__FIND____REPLACE____USE_REGEX__`、`__FIRST_FILE__`、`__FONT_COLOR__`、`__FONT_TRANSPARENCY__`、`__FORMULA__`、`__HEIGHT__`、`__IMAGE__`、`__IMAGE____LEFT____TOP____WIDTH____HEIGHT__`、`__INDEX__`、`__INSERTS__`、`__KIND__`、`__KIND____IMAGE__`、`__LEFT__`、`__OPERATIONS_JSON__`、`__OTHER_PATH__`、`__OUTLINE__`、`__OUTPUT_DIR__`、`__OUTPUT_DIR____PAGE__`、`__OUTPUT_PATH__`、`__PAGES_PER_FILE__`、`__PAGE_RANGE__`、`__PAGE__`、`__PARA_INDEX__`、`__PH_INDEX__`、`__QUERY__`、`__RANGE__`、`__REPLACE__`、`__SAVE_AND_CLOSE__`、`__SAVE_AS__`、`__SHAPE_ACCESS__`、`__SHEET_ACCESS__`、`__SHEET_ACCESS____RANGE__`、`__SLIDE__`、`__SLIDE____SHAPE_ACCESS____TEXT__`、`__SOURCE_PATH__`、`__SOURCE_PATH____OUTPUT_PATH__`、`__START_PAGE__`、`__START_PAGE____END_PAGE__`、`__START__`、`__START____END__`、`__SUBTITLE__`、`__TEMPLATE_PATH__`、`__TEXT__`、`__TITLE__`、`__TITLE____SUBTITLE__`、`__TOP__`、`__USE_REGEX__`、`__VALUE__`、`__WIDTH__`

## 脚本清单（按文件偏移）

### 1. offset `5656769` · 140 字符

上下文（op 名/错误串）：`        � @    h               �� @    �pW@            �       �pW@            �       �� @    �               �� @    �pW@                    �pW@            $       �� @    H               �� @    >`

```javascript
(function(){ try { return JSON.stringify({ok:true, text: OCR('�N')}); } catch(e) { return JSON.stringify({ok:false, error: e.message}); } })
```

### 2. offset `5672038` · 566 字符

上下文（op 名/错误串）：`           PresentationInputpngsrc\commands\doc.rs L�W@            �   "   L�W@            p   "   kwpsjsrunner    L�W@            �   "   L�W@            �   "   paragraphchar   L�W@                "`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var templatePath = "__TEMPLATE_PATH__";
    var outputPath = "__OUTPUT_PATH__";
    var doc = Application.Documents.Open(templatePath, false, false, false);
    if (!doc) throw new Error("failed to open template");
    var __result = (function(doc) { __BODY__ })(doc);
    doc.SaveAs2(outputPath);
    doc.Close(0);
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 3. offset `5673230` · 425 字符

上下文（op 名/错误串）：`           �   "   extract requires a non-empty --pages range      ��W@                "   annotate��W@            �   "   edit    ��W@            �   "   ��W@            �   "   ��W@            �   "`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var pres = Application.Presentations.Add();
    var __result = (function(pres) { __BODY__ })(pres);
    pres.SaveAs("__OUTPUT_PATH__");
    pres.Saved = true;
    pres.Close();
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 4. offset `5674142` · 391 字符

上下文（op 名/错误串）：`    �W@            �   "    �W@            �   "    �W@            �   "   !unsupported sheet export format: �      �W@            �   "   sheet PNG export requires --range        �W@            �   "`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var wb = Application.Workbooks.Add();
    var __result = (function(wb) { __BODY__ })(wb);
    wb.SaveAs("__OUTPUT_PATH__");
    wb.Close(true);
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 5. offset `5688030` · 107 字符

上下文（op 名/错误串）：`               "   annotate �W@            �   "    �W@            7   "    �W@            +   "    �W@                a   edit     �W@            �   "    �W@            �   "    �W@            �   "`

```javascript
(function() { try { �W } catch(e) { return JSON.stringify({ok: false, error: e.message || String(e)}); } })
```

### 6. offset `5743603` · 113 字符

上下文（op 名/错误串）：`s === e) {      return JSON.stringify({ok: true, slide: s, notes: slides[0].notes});  }  return JSON.stringify({ok: true, from: s, to: e, slideCount: slides.length, slides: slides});  "remove"__BODY__`

```javascript
(function() { try { __BODY__ } catch(e) { return JSON.stringify({ok: false, error: e.message || String(e)}); } })
```

### 7. offset `5753069` · 769 字符

上下文（op 名/错误串）：`"});  var ws = __SHEET_ACCESS__;  ws.Range("__RANGE__").Value2 = "__VALUE__";  return JSON.stringify({ok: true, action: 'edited', range: "__RANGE__"});  __VALUE__doc.Close(0);doc.Save(); doc.Close(0);`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var filePath = "__FILE_PATH__";
    var want = filePath.replace(/\//g, "\\").toLowerCase();
    var doc = null;
    for (var i = 1; i <= Application.Documents.Count; i++) {
      var d = Application.Documents.Item(i);
      if ((d.FullName || "").toLowerCase() === want) { doc = d; break; }
    }
    if (!doc) doc = Application.Documents.Open(filePath, false, false, false);
    if (!doc) throw new Error("failed to open: " + filePath);
    var __result = (function(doc) { __BODY__ })(doc);
    __SAVE_AND_CLOSE__
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 8. offset `5753920` · 471 字符

上下文（op 名/错误串）：`.stringify(__result);    } catch(e) {      return JSON.stringify({ok: false, error: e.message || String(e)});    }  })()  __FILE_PATH____SAVE_AND_CLOSE__pdfDoc.Close(0);pdfDoc.Save(); pdfDoc.Close(0);`

```javascript
(function() {
  try {
    var filePath = "__FILE_PATH__";
    var pdfDoc = Application.Documents.Open(filePath,false,false,"","",false);
    if (!pdfDoc) throw new Error("failed to open PDF: " + filePath);
    var __result = (function(pdfDoc) { __BODY__ })(pdfDoc);
    __SAVE_AND_CLOSE__
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 9. offset `5754395` · 821 字符

上下文（op 名/错误串）：`   __SAVE_AND_CLOSE__      return typeof __result === "string" ? __result : JSON.stringify(__result);    } catch(e) {      return JSON.stringify({ok: false, error: e.message || String(e)});    }  })()`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var filePath = "__FILE_PATH__";
    var want = filePath.replace(/\//g, "\\").toLowerCase();
    var pres = null;
    for (var i = 1; i <= Application.Presentations.Count; i++) {
      var p = Application.Presentations.Item(i);
      if (p.FullName && String(p.FullName).toLowerCase() === want) { pres = p; break; }
    }
    if (!pres) pres = Application.Presentations.Open(filePath, false, false, false);
    if (!pres) throw new Error("failed to open: " + filePath);
    var __result = (function(pres) { __BODY__ })(pres);
    __SAVE_AND_CLOSE__
    pres.Close();
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 10. offset `5755293` · 741 字符

上下文（op 名/错误串）：` JSON.stringify(__result);    } catch(e) {      return JSON.stringify({ok: false, error: e.message || String(e)});    }  })()  pres.Saved = true;pres.Save();wb.Close(false);wb.Save(); wb.Close(false);`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var filePath = "__FILE_PATH__";
    var want = filePath.replace(/\//g, "\\").toLowerCase();
    var wb = null;
    for (var i = 1; i <= Application.Workbooks.Count; i++) {
      var w = Application.Workbooks.Item(i);
      if ((w.FullName || "").toLowerCase() === want) { wb = w; break; }
    }
    if (!wb) wb = Application.Workbooks.Open(filePath);
    if (!wb) throw new Error("failed to open: " + filePath);
    var __result = (function(wb) { __BODY__ })(wb);
    __SAVE_AND_CLOSE__
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 11. offset `5758445` · 763 字符

上下文（op 名/错误串）：`re child stdin    f @                    �5T@     f @                    P�S@    �5T@    � Y@    �� @    0� @     � @    �� @    �� @    failed to capture child stdoutspecify --from-json or --from-csv`

```javascript
(function() {
  // 从 CSV 新建表格：**CSV 交给 WPS 侧打开**（所有平台同一条路，CLI 不读这个文件）。
  //
  // OHOS 上这条路是唯一选择：wpscli 进程跑在调用方应用沙箱里，对用户目录的 .csv open()
  // 直接 EPERM（真机 2026-09-24：`sheet create -o out.xlsx --from-csv <桌面 csv>` 在 WPS
  // 还没启动时就返回 `io error: Operation not permitted (os error 1)`）。桌面则是因为
  // ET 本来就能把 .csv 当工作簿打开，打开 → 另存为就是「从 CSV 新建表格」，解析（引号、
  // 字段内逗号、首行 BOM）也比 CLI 侧朴素 split(',') 更准——两个平台共用这套更好。
  var srcPath = "__SOURCE_PATH__";
  function normPath(p) { return String(p || "").replace(/\\/g, "/").replace(/^file:\/\//, "").toLowerCase(); }
  function errText(e) { return String((e && (e.message || e.description)) || e); }

  try { Application.DisplayAlerts = false; } catch (e) {}

  var want = normPath(srcPath);
  var tried = [];
  var wb = null;

  // 1)
```

### 12. offset `5786743` · 394 字符

上下文（op 名/错误串）：` target disagreepng--page must be >= 1 (all pages when omitted)--page is only supported for png exportinternal error: entered unreachable code: TextFile 自带完整脚本src\commands\doc.rs �}Y@                )`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var doc = Application.Documents.Add();
    var __result = (function(doc) { __BODY__ })(doc);
    doc.SaveAs2("__OUTPUT_PATH__");
    doc.Close(0);
    return typeof __result === "string" ? __result : JSON.stringify(__result);
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```

### 13. offset `5816760` · 828 字符

上下文（op 名/错误串）：`texts = [];  for (var i = s; i <= e && i <= doc.Paragraphs.Count; i++) {      texts.push(doc.Paragraphs.Item(i).Range.Text);  }  return JSON.stringify({ok: true, paragraphs: texts});  __START____END__`

```javascript
(function() {
  try {
    Application.DisplayAlerts = false;
    var srcPath = "__SOURCE_PATH__";
    var want = srcPath.replace(/\//g, "\\").toLowerCase();
    var src = null;
    for (var i = 1; i <= Application.Documents.Count; i++) {
      var d = Application.Documents.Item(i);
      if ((d.FullName || "").toLowerCase() === want) { src = d; break; }
    }
    if (!src) src = Application.Documents.Open(srcPath, false, false, false);
    if (!src) throw new Error("failed to open: " + srcPath);
    var text = src.Content.Text;
    src.Close(0);
    var doc = Application.Documents.Add();
    doc.Content.Text = text;
    doc.SaveAs2("__OUTPUT_PATH__");
    doc.Close(0);
    return JSON.stringify({ok: true, action: "created"});
  } catch(e) {
    return JSON.stringify({ok: false, error: e.message || String(e)});
  }
})
```
