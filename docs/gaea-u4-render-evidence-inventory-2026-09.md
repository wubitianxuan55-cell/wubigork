# U4 渲染证据通道盘点（零绑定方案，2026-09）

规划依据：docs/gaea-dsh-univer-office-distill-plan-2026-09.md §4.3-①（agent 看得见渲染结果）
与 §5 落地表第 6 行「U4 渲染证据通道 = 先盘点零绑定方案（既有 screenshot/vision 工具接线），
缺口再立小绑定」。决策拍板项 6 同源。

- 版本基线：v4.99.0（绑定面 581 冻结）；本文盘点结论：**U4 渲染证据通道目标 0 新绑定可达成**。
- 「渲染证据」的两类消费方，口径分开评：
  - **agent 侧取证**（verify 循环：模型自己取渲染结果做断言/自查）；
  - **UI 侧证据**（用户在预览面看到的保真预览/缩略/页图）。
- 所有路径引用均为仓库实际代码位（2026-09-05 盘点）。

---

## 1. 通道清单

### ① soffice 无头渲染（agent 侧取证主通道；技能条款已固化）

| 维度 | 结论 |
|---|---|
| 落盘位置 | agent 指定路径（工作区内任意可写位置）；Go 内部 seam 产物另见②③ |
| 触发方式 | **既有、零改动**：agent 经 bash 工具跑 `soffice --headless --convert-to pdf` + `pdftoppm -png`。office-edit 内置技能（internal/gaea/skill/builtins.go「验证闭环（写 → 读回 → 看渲染）」分节）已写死该条款：「渲染证据（视觉相关改动）：bash 运行 soffice --headless --convert-to pdf 把文件转 PDF，再 pdftoppm -png 出页图自查；截图只补视觉判断，不替代结构回读」，pptx 分节另有「每改一页 soffice 渲染自查三件事（越界/溢出/重叠）」 |
| 证据消费 | 页图落盘后走既有 `vision` 工具（internal/gaea/tool/builtin/vision.go：image_path → 本地视觉模型，不耗主模型 token）读图断言——取证闭环 **bash + vision 两个既有工具即成立** |
| Go 内部同源 | internal/app/gaea_pdf.go `convertToPdfFile`（独立 profile 防与 recalc/用户 LibreOffice 抢锁，180s 超时，findSoffice PATH+常见安装位）；包级变量 `verifyConvertToPdf`（gaea_verify.go）是 Verifier 通道 B 与 pptx 预览共用的可注入 seam |
| 触发成本 | soffice 冷启动约 2~10s/次（进程级）；pdftoppm 100 DPI 全本约 1~3s。时延中 |
| 依赖 | LibreOffice 安装（缺 → 既有「视觉渲染降级 warn，仅结构复核」兜底）；poppler pdftoppm |
| 保真度 | LibreOffice 渲染口径——规划风险表拍板「渲染证据以 soffice 管线为准」，即这是**保真基准通道** |
| 绑定面需求 | **0** |
| 复用为 UI 证据 | 不直接（UI 不跑 bash）；UI 侧要 soffice 口径页图时走③的 pptx 管线或②的 Verifier 产物 |

### ② Verifier 逐页缩略图链（pdftoppm PNG 落盘 + 成对缩略，审计兜底）

| 维度 | 结论 |
|---|---|
| 落盘位置 | `<工作区>/.gaea/work/journal/verify/<id>/`：before.pdf / after.pdf + before/before-N.png、after/after-N.png（internal/app/gaea_verify.go `runVisualDiff`；对可视化文档复核时自动生成，产物绝对路径记入 verdict.channelBArtifacts） |
| 前端链 | lib/verifyArtifacts.ts（纯函数：路径相对化/页码解析/成对配对/降级分类）+ components/verify/VerifyArtifactsThumbs.tsx（点「查看缩略图」才列目录/取图：`GaeaListDir` + 逐页 `app.Preview` 取 image dataUrl，并发 ≤4，展开过的数据保留） |
| 触发方式 | 证据卡「复核」（GaeaVerifyRecord）时对二进制办公文档自动跑；非实时（只在复核时生成） |
| 触发成本 | soffice×2 + pdftoppm×2，一次复核约 4~20s；PNG 落盘持久（审计产物，可事后人工查差异页） |
| 依赖 | LibreOffice + poppler（缺 → warn 降级，仅结构复核，通道仍可用） |
| 保真度 | 100 DPI PNG（diff 用低分辨率）；**before/after 逐页成对 + 像素差异率是独有能力**（其它通道无改前改后对比） |
| 绑定面需求 | **0**（GaeaListDir / GaeaPreview 既有绑定） |
| 复用为 UI 证据 | 已在用：回合卡/证据卡的视觉复核行内缩略。定位=「写后证据的兜底展示面」，不是实时跟随通道 |

### ③ XlsxPreview / DocxPreview / pptx 逐页直读渲染（UI 保真预览主通道）

全部经 **`app.Preview`（GaeaPreview，internal/app/gaea_preview.go）单绑定口**消费，FilePreview /
FilePreviewModal / FileThumb / DeliverablesPanel 共用：

| 格式 | 后端 | 前端 | 保真度与时延 |
|---|---|---|---|
| docx | `.docx` 分支：整文件 base64 dataUrl + docmd 轻量 markdown 缩略文本（gaea_preview.go docx 分支） | DocxPreview：docx-preview `renderAsync` 浏览器内渲染——版式/表格/页眉页脚/修订/批注均保留（「框选即改」修订制同源）；失败降级 docxText 纯文本 | 保真**高**（修订可见=审阅语义）；局限：WebView 渲染口径与 Word/soffice 有差异；整本 base64 传输（MB 级）。时延：本地读盘毫秒级 + 前端渲染 0.1~1s |
| xlsx | `.xlsx` 分支：internal/office/xlsxpreview（excelize 解析值/样式/列宽/合并，公式无缓存值时先 LibreOffice `Recalc` 重算）→ 结构化 JSON | XlsxPreview：NumFmt 近似渲染 + 直编/Plan→Apply/gbase 视图层 | 保真**中**（「预览口径」如实标注：条件格式、原生图表不渲染——规划 §5 U4 行「按实测保真度分步，不承诺引擎级」）；时延低（毫秒级，需 recalc 时 + 一次 soffice） |
| pptx | `.pptx` 分支 → previewPptx（gaea_pptx.go）：soffice→PDF 落 `.gaea/cache/pptx-preview/<hash>.pdf` → poppler 64 DPI 逐页 PNG 缓存（`<hash>-pages/`）→ Pages dataUrl 随 PreviewResult 回传；pdftoppm 缺失回退整本 PDF dataUrl；Hint=outline 叠 GaeaPptxOutline 大纲卡 | FilePreview kind=pdf 逐页铺 + PptxOutline 页锚点滚动（v4.28 B2 / v4.33 懒加载） | 保真**高**（soffice 口径真渲染）；缓存键 = sha256(路径+size+mtime 纳秒)——**文件一变立即失配重转，写后重预览天然拿到新版本**（规划 §4.3-2「pptx 缓存键含 ModTime 已天然支持」）；首次 2~10s，命中缓存毫秒级；7 天 TTL 清扫 |
| pdf/doc/xls | docmd ConvertLimitProgress（OCR 逐页进度事件回传） | markdown 渲染 + 截断提示 | 文本口径非版式 |

### ④ GaeaPreview 探测（前端万能取图口 + 结构化目录错误码）

- `app.Preview(rel)` 对 image 扩展名直接回 dataUrl——VerifyArtifactsThumbs 逐页取图、FileThumb
  卡片缩略（首 4 行 × 4 列迷你表格 / markdown 头部）、DeliverablesPanel 全部复用同一口；
- 目录探测语义结构化：kind=error + `Error [GAEADIR_*]` 码（verifyArtifacts.parseErrorCode 路由）；
- 时延低、依赖 0、绑定面 **0**（已是既有绑定）。

### ⑤ 文本直读 GaeaReadFile（辅助，非渲染证据）

- internal/app/gaea_ui_extra.go:577，契约 `{path, markdown, size}`（无 kind）；XlsxPreview 的
  .gbase.json sidecar 视图配置、纯文本/笔记读侧用。跟随时若只刷主文件不刷 sidecar，
  视图层会滞后——本轮 B 已把 sidecar 重读挂进刷新路径（见 §4 注）。

### ⑥（参考，排除项）screen_capture + vision 与 CDP Edge

- `screen_capture`（builtin/screenshot.go）：OS 级全屏/区域截图 → .gaea/uploads/ → vision。
  截的是屏幕不是文档，**对 office 文件取证无用**（仅用户手动开着 WPS/Word 对比时间接有用）。
- CDP Edge（internal/gaea/browser）：驱动 Edge 截网页/取页面（BrowserPane）——U3 pptx lint 拟用
  它做文本度量；非 office 文档渲染证据通道。
- 规划 §4.4 已排除 puppeteer/Node 无头链，不另设。

---

## 2. 通道对比总表

| 通道 | 消费方 | 触发成本 | 时延 | 保真度 | 依赖 | 绑定面需求 |
|---|---|---|---|---|---|---|
| ① bash soffice+pdftoppm + vision | agent | 2 次 bash + 1 次 vision/文件 | 3~15s | soffice 口径（基准） | LibreOffice+poppler（缺则降级） | **0** |
| ② Verifier 通道 B 缩略 | UI（证据卡）+审计 | 复核动作自动 | 4~20s/次 | 100 DPI PNG，成对 diff | 同上 | **0** |
| ③ GaeaPreview 直读（docx/xlsx/pptx） | UI（预览面） | 打开/刷新预览即触发 | docx/xlsx 毫秒~1s；pptx 首次 2~10s、命中缓存毫秒 | docx 高 / xlsx 中（预览口径）/ pptx 高（soffice） | 无外部依赖（xlsx recalc 需 LibreOffice，缺则显公式） | **0** |
| ④ GaeaPreview image 探测 | UI（缩略/逐页取图） | 按需逐页 | 毫秒/页 | 原图 | 无 | **0** |
| ⑤ GaeaReadFile 文本 | UI（sidecar/文本） | 按需 | 毫秒 | 纯文本 | 无 | **0** |

## 3. 结论：推荐组合方案

**「写后反馈」证据首选 = ③+④（GaeaPreview 直读刷新）**：
写类工具落盘 → 前端重新拉一次 `app.Preview(已打开文件)` 即拿到全部三种格式的最新渲染
（docx=新 dataUrl 重渲染；xlsx=新结构化 JSON；pptx=缓存键含 mtime 自动失配重转出新的
soffice 逐页页图）。**零绑定、零后端改动、时延最低**，且这正是本轮 U4-B「写后预览实时
跟随」的落点：前端信号派生 + 800ms 防抖合并 + FilePreview 静默重载（不弹窗、不重挂、
滚动位保持），pptx 侧 soffice 重转的代价由既有 mtime 缓存键与逐页懒加载摊薄。

**兜底 = ②（Verifier 通道 B 成对缩略）+ ①（bash soffice+vision 技能条款）**：
- 用户要「改前改后视觉差异 / Office 级渲染口径」→ 回合卡证据卡的视觉复核行
  （VerifyArtifactsThumbs）就地看成对页图，产物落盘可审计；
- agent 自查 → office-edit 技能验证闭环条款（U1 已落地）用 bash+vision，模型按需取证据，
  gaea 不为其新建立绑定。

组合纪律（对齐规划风险表）：UI 预览面以「预览口径」呈现（docx-preview/xlsxpreview），
渲染保真口径以 soffice 管线为准；两套口径在 UI 上如实标注、不混充。

## 4. 缺口清单（仅列缺口，不立绑定；「若立绑定建议形态」仅备忘）

1. **UI 侧无 docx/xlsx 的 soffice 页图通道**：预览面（③）是浏览器/前端口径，用户无法在
   UI 内直接看 soffice 渲染的 docx/xlsx 页图（pptx 已有）。现状兜底：②复核产物 + 外部打开。
   *若立绑定建议形态*：把 previewPptx 的 `.gaea/cache/render-preview/<hash>`（hash=path+size+
   mtime）模式推广到 docx/xlsx，`GaeaRenderPages(rel, dpi?) → {pages:[{page,dataUrl}]}`，
   纯 Go 内部扩展、绑定 +1；建议待 pptx 刀2（逐页截图回读）验证该管线后再评估。
2. **xlsx 条件格式/原生图表预览不渲染**（xlsxpreview 不解析 dxf/chart XML）：规划已拍板
   分步、不承诺引擎级。*若增强建议形态*：Go 侧 excelize 读条件格式规则并入既有 JSON
   （**零绑定**，前端 XlsxPreview 加渲染分支）；图表待实测排期。
3. **大 docx 整本 base64 刷新成本**：每次跟随刷新重传 MB 级 dataUrl（本地回环，实测可感知
   但可接受）。*若立绑定建议形态*：`GaeaDocxPages(rel)` 分页 dataUrl（soffice+pdftoppm +
   mtime 缓存，同 pptx 模式）；现状先不动，观察真机大文档走查再定。
4. **agent 取证要自己拼 bash 命令**（soffice 探测/DPI 约定散在技能文案）：技能条款已可用，
   成本=每文件 2 次 bash + 1 次 vision。*若立绑定建议形态*：`GaeaRenderEvidence(rel)` 一键
   soffice→PDF→PNG 落 `.gaea/work/render/<名>/` 返回页图路径（绑定 +1）；建议先做 U4 真模型
   走查（规划 §6-4）看失败率再决定是否值得。
5. **sidecar（.gbase.json）视图跟随**：本轮已在 XlsxPreview 把 sidecar 重读挂进刷新路径
   （body 变化即重读），无需绑定；遗留=sidecar 与 xlsx 本体分离写入的中间态短暂可见
   （诚实呈现，不加机制）。
6. **多写并发的 UI 节流**：前端 800ms 防抖合并已覆盖（本轮 B）；后端无需变更、无缺口。

> 本盘点为研究产出，不改任何绑定/Go 代码；U4-B「写后预览实时跟随」按 §3 推荐组合落地于
> frontend（officeTurnProjection.ts / App.tsx / paneTabs.ts / FilePreview / DocxPreview /
> XlsxPreview / WorkspacePane），绑定面 0 变更。
