# gaea 办公板块：模型与工具调用链（2026-08-10 核对）

> 目的：把「转换 / 解析 / 生图 / 制表 / 编辑」等能力的前端入口 → Go 处理 → 模型/工具/
> Python 技能调用链一次性核对清楚，作为开发和排障的依据。均为代码实锤（文件+行号级）。

## 0. 两条模型入口

1. **聊天 Agent（office 对话）**：controller（`gaeaBoot.Build`，`gaeaBuildController`）→
   bridge provider（kind=gaea，`cfg.DefaultModel="gaea"`）→ 模型中心引擎
   （`ai.Client` → `modelengine.Manager.BuildChatURL`，OpenAI 兼容 `/chat/completions`）。
   模型通过工具注册表（47+ 内置工具 + 应用注入工具 + 技能）自主干活。
2. **功能级直连（前端按钮/面板）**：`bridge.ts` 短名 → `Gaea*` handler →
   `a.client.*`（`routeModel("office")` 路由到模型中心）或本地工具/Python 脚本。

模型路由统一走 `routeModel(feature)`（model_router.go）：功能绑定 → 全局活跃引擎 →
首个可用引擎；`ai.Client.resolveChatEndpoint` 决定 URL/API Key（xAI 走 OAuth，其余走
engineMgr）。

## 1. 文件转换（docx/xlsx/pdf → Markdown）

| 入口 | 链路 | 模型? |
|---|---|---|
| 聊天 `format_convert` | builtin → `docmd.Convert`（纯 Go） | 无 |
| 预览 `.doc/.xls/.pdf` | `GaeaPreview` → `docmd.Convert` | 无 |

`internal/office/docmd`：docx 用 zip + XML 解析（标题/表格）；xlsx 用 excelize/XML 提取；
pdf 文本提取 + 扫描件 OCR（OvisOCR2 常驻 llama-server 优先 → 退回 pdftoppm + tesseract）。
转换本身不调 LLM，也不依赖 Python（OCR 服务除外）。

## 2. 解析

| 能力 | 链路 | 模型? |
|---|---|---|
| 图片 OCR（提取文字） | `GaeaOCRText` → `docmd.OCRImageText` → OvisOCR2 常驻（本地 llama-server，OpenAI 兼容） | 本地小模型 |
| 识图（截图/图片理解） | `GaeaRecognizeImage` → `vision.RecognizeImage` → 本地视觉模型 `http://127.0.0.1:8080/v1` | 本地视觉模型 |
| 招标文件解析（方案编写） | `proposal.Service.parseBidFile`：docmd 转 Markdown → **分块 12000 字调 AI**（`s.ai.ChatSimpleStream`，parseSystemPrompt 提取资格/评分/红线/格式/暗标+原文 quote）→ 落库 | **是**（office 路由） |
| 粘贴表格即数据 | 纯前端 `tableData.ts`（识别→Markdown 表格） | 无 |

## 3. 生图 / 图表 / 绘图

| 能力 | 链路 | 模型? |
|---|---|---|
| 聊天 `image_gen` | app 注入工具 `imageGenTool` → `ai.Client.GenerateImage` → 图片后端（xAI / Herdsman / Ollama / ComfyUI，按 `cfg.ImageBackend/ImageModel`）→ 存 `.gaea/uploads` | **是**（图片模型） |
| 聊天 `chart_gen`（统计图） | builtin → 内联 Python matplotlib 脚本 | 无 |
| 聊天 `diagram_gen`（Mermaid） | app 注入 `diagramTool` → `ChatSimpleStreamWithOptions`（提示词约束节点数/类型）→ 存 `.mmd` → 前端 mermaid 渲染 | **是**（文本模型） |
| 跨应用联动 `CrossEmbed` | xlsx → `crosslink.ExtractChartData`（excelize）→ `GenerateChartPNG`（matplotlib）→ 嵌入 docx/pptx（python-docx/pptx） | 无 |

## 4. 制表

| 入口 | 链路 | 模型? |
|---|---|---|
| 聊天内建表/数据处理 | 模型自主：`bash` + Python（openpyxl/pandas）或 `run_skill`(xlsx 技能 → 按 SKILL.md 用 bash 跑脚本) | 规划用模型，执行无 |
| 统一交付 `ExportDeliverable` format=xlsx | `exportXlsx`：Go excelize 直接写 | 无 |
| xlsx 预览 | `GaeaPreview` → `xlsxpreview.Render`（excelize 读单元格/公式/样式）；公式无缓存值先 `xlsxedit.Recalc`（recalc.py → LibreOffice 宏） | 无 |

## 5. 编辑

| 能力 | 链路 | 模型? |
|---|---|---|
| Word 框选即改（生成替换文本） | `GaeaOfficeEditText` → `ai.Client.OfficeEditText` | **是**（office 路由） |
| Word 写入修订 | `GaeaDocxApplyEdit` → `docxedit.ApplyTrackedReplace`（Go 写 w:del+w:ins，不动其他内容） | 无 |
| Word 接受/拒绝修订 | `GaeaDocxAcceptChanges` → `docxedit`（纯 Go） | 无 |
| Excel AI 编辑（单元格级指令） | `GaeaXlsxEdit`：`xlsxedit.BuildContext`（excelize 读上下文）→ **`ai.Client.XlsxEditOps`（模型规划操作 JSON）** → `xlsxedit.ApplyOps`（excelize 执行）→ `Recalc`（LibreOffice）→ `xlsxpreview.Render` | **是**（规划）+ 无（执行） |
| Excel 直接写单元格/行列/重算 | `GaeaXlsxSetCell / RowOps / ColOps / Recalc` → excelize + LibreOffice | 无 |
| docx 交付导出 | `ExportDeliverable` format=docx → `create_docx.py`（python-docx，`runPython`） | 无 |
| pptx 交付导出 | `ExportDeliverable` format=pptx → `markdownToSlides` → `create_pptx.py`（python-pptx） | 无 |

## 6. 工具与技能的执行方式

- **内置工具**：`tool.RegisterBuiltin`（format_convert / chart_gen / read_file / write_file /
  ls / bash / bash_output / kill_shell / wait / web_search / web_fetch / todo_write /
  complete_step / memory_search / knowledge_add / knowledge_search / read_skill +
  remember / forget / screenshot / vision / OCR 等）。聊天内模型按 schema 调用。
- **应用注入工具**（ExtraTools，可访问 App/模型中心）：`imageGenTool`（image_gen）、
  `diagramTool`（diagram_gen）、`factAddTool / factListTool / factClearTool`（事实底座）。
- **技能（run_skill）**：inline → SKILL.md 正文折回给模型，模型按说明用 bash 跑 Python
  脚本（docx=create_docx.py，xlsx=openpyxl+recalc.py，pptx=create_pptx.py，pdf=本地 OCR
  管线）；subagent / pipeline 走独立子循环。
- **本地环境依赖**：Python（openpyxl / pandas / matplotlib / python-docx / python-pptx）、
  LibreOffice（recalc.py 宏，soffice 路径由脚本探测）、OvisOCR2（本地 llama-server OCR）、
  poppler（pdftoppm）、tesseract（OCR 兜底）。

## 7. 一句话总结

**需要 LLM 的只有四处**：office 聊天 agent 本身、招标解析（分块提取）、Word/Excel 的
AI 编辑与 Excel 操作规划（生成替换文本/操作 JSON）、Mermaid 绘图；**图片生成走独立图片
后端**（xAI/本地/ComfyUI）。其余（转换、预览、制表导出、修订写入、OCR/识图、图表渲染）
全部是本地 Go / Python / LibreOffice / 本地小模型，不依赖外部 LLM——这正好是 gaea
「本地优先、数据不出机」的底座。
