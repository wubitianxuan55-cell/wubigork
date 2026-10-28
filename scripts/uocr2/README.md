# scripts/uocr2 — Unlimited-OCR-2 胶水代码存档

`internal/docmd/ocr_uocr2.go`（uocr2 OCR 引擎）依赖的常驻 python 服务与 CLI 的
**唯一版本控制副本**。运行实例装在 `C:\AI\unlimited-ocr2`（模型 weights 6.7GB +
venv，**不入仓库**，布局与补丁说明见该目录 README.md）。

- `server.py` — 常驻服务（gaea 按需拉起，默认端口 8138）：
  `GET /health` + `POST /ocr`（单页）+ `POST /ocr_batch`（多页批识别）；
  模型加载完才监听=应答即就绪；422=截断/页数不齐 fail-closed。
- `ocr.py` — 手动 CLI：`python ocr.py 图.png [图2...] | --pdf 文档.pdf [--dpi 300]`；
  crop_mode=True 逐页 gundam 切片 + 5 页分窗 + 批退化双守卫（页数对账+近空页
  48 rune）→ 逐页回退；`eval_mode=True`（缺省 stream=True 会流式重印全文）。

**同步纪律**：改运行实例（C:\AI\unlimited-ocr2）后必须回拷本目录一并提交；
反向（从本目录部署到新机器）按实例 README 的 venv/补丁步骤装配。
