# gaea 办公板块 CAD 图纸识别工具·总体规划（2026-10）

> 背景：市场调研已收官（`docs/gaea-cad-recognition-market-survey-2026-10.md`，含二轮 GitHub/社区盘点附录），用户拍板立项：办公板块增加 CAD 图纸识别工具（可选使用，dwg/PDF 等格式）。本文档是刀路准绳，逐期关账；后续刀路以本文为准。
>
> 定位一句话（调研定案）：**桌面本地的图纸信息提取与问答工具**——导入 DWG/DXF/PDF，看得见（查看器）、提得走（图层/文本/图框结构化）、问得出（复用识图管线）。**明确不做**看图算量、构件计数直出工程量、全构件建模翻模。

---

## 一、现状与可复用资产（摸底结论）

### 1.1 识图管线（问图纸的底座，零改动复用）

- `internal/gaea/vision`：Provider 注册表（OpenAI 兼容视觉端点）；运行时解析链=跟随聊天模型 → func_vision 功能绑定 → gaea.toml 静态注入（`internal/app/gaea_vision.go:31-61`）。**VisionRoute 只读不记账红线**：TouchEngine 记账只在真实识图调用路径（`visionLocalRuntime`）。
- 前端识图=Composer 附件动作模式（`gaea/hooks/useComposerAttachments.ts:185-203`）：`app.RecognizeImage(path, prompt)` → 结果拼 `【图片识图：文件名】` 作为用户消息发对话流渲染。**CAD 问图纸对称复用此模式，零新模型绑定**。

### 1.2 文件导入管线

- 前端单源 `gaea/lib/pickFile.ts`：`pickFileAsFile(accept)` = 扩展名白名单对话框 + 后置复核；**按扩展名判型（type 恒空的坑已在 v4.508 根修）**。
- 后端护栏 `internal/app/gaea_ui_extra.go`：`GaeaPickFiles` 登记 pickedFiles 白名单 → `GaeaReadFileB64` 强制 `isPickedFile` + 64MB 上限；`GaeaSaveAttachmentFile` 落 `<gaeaCwd()>/.gaea/uploads/`。
- 拖拽路径 `attachDroppedFiles`：>10MB 确认弹窗、分块 base64（8KB）。

### 1.3 PDF 通道（现成，直接复用）

无 pdf.js/three 依赖；PDF 是「后端渲染、前端贴图」：soffice→PDF→**pdftoppm 逐页 PNG**（`internal/app/gaea_preview.go`）→ `FilePreview.tsx:467-499` 逐页 `<img>` 懒加载。**PDF 图纸的光栅化不需要新方案**，复用此通道出页图喂识图。

### 1.4 功能卡机制（「可选使用」的落点）

右栏工作台视图卡注册表 `gaea/lib/workspaceTabs.ts` WORKSPACE_TABS（`{id,label,icon,keywords,defaultEnabled}`）+ `sidebarRegistry.ts` RENDERERS 接线；`defaultEnabled=false` 即可选启用（用户经设置卡开）。头注已写明扩展法：**新增面板 = WORKSPACE_TABS 追加一项 + RENDERERS 补一行，按钮条/设置卡/命令面板零改动**。这是 CAD 工具的落点。

### 1.5 新绑定六步（流程纪律，勿漏）

Go 方法（`Gaea*` 前缀默认进 office 门面，勿撞 `excludedBindings`）→ `go run ./scripts/gen_bindings`（-names/-legacy-ts/-signatures-ts 三产物 + bindings_office.go 委托再生）→ bridge 域接口 + `mappings.ts` 短名 → `spaceBindings.ts` GAEA_METHOD_FACETS 认领（**两空格缩进格式坑**）→ `mock/office/methods_office.ts` 补 mock → 组件消费。drift 锁与契约锁自动核验。

### 1.6 造价域联动点（二期用）

清单/定额三层（gf_* 表）；`MatchResourceByTitleSpec` 唯一命中=宁缺勿误管线；清单弹窗挂定额体系（gf_bill_quota_links）。二期只做**导出/只读联动**，不动组价写路径。

### 1.7 缺口（本工具要新建的）

无 DXF/DWG 解析能力；前端无矢量渲染依赖；无图纸光栅化（DWG→PNG）能力；无 CAD 领域知识技能。

---

## 二、产品形态与红线

### 2.1 形态

**右栏工作台可选视图卡「CAD 图纸」**（id 暂定 `cad`，`defaultEnabled=false`，用户在设置卡开启——满足「可选使用」拍板）。卡内三区：

1. **导入区**：选文件（.dwg/.dxf/.pdf，扩展名白名单）或从附件拖入；导入后常驻最近图纸列表（内存态，不落库——图纸不入持久化库，避免数据膨胀）。
2. **查看区**：DXF/DWG→查看器渲染（图层树/缩放平移）；PDF→pdftoppm 页图复用 FilePreview 模式。
3. **信息区**：结构化摘要（图框图号/图名、图层清单、文本实体清单）+「问图纸」按钮（光栅化当前视图/页 → 识图管线 → 结果拼 `【图纸问答：文件名】` 发对话流，对称识图卡模式）。

### 2.2 红线（逐期沿用）

1. **可选使用**：defaultEnabled=false；不进 OFFICE_CAPABILITIES 欢迎卡；不占 Composer 默认位。
2. **只读不记账**：新 Cad 绑定全部只读（解析/提取/光栅化）；「问图纸」走既有 `GaeaRecognizeImage`（其内部已有记账纪律），Cad 绑定面不 TouchEngine。
3. **AI 结果不直接信**：问图纸结果进对话流供人读；结构化提取展示**依据**（图层名/原始文本/坐标），数值字段不自动写入任何库——二期联动必须经用户确认（沿用记忆写入确认红线）。
4. **GPL 纪律**：不引入 LibreDWG 任何链接形态（含 WASM）；mlightcad 只用其 MIT 核心（DXF 解析+查看器），不引其 DWG 包。若启用 LibreDWG 兜底，仅独立子进程 + 随附 GPL 声明 + 法务过目。
5. **领域知识技能化**：CAD 读图知识（图例/平法/图框规范）进技能（internal/gaea/skill/builtins.go 体系，参照 office-gongwen），内核只留通用机制。
6. **简化≠删功能**：不动识图卡既有四消费点；PDF 通道共用不改行为。
7. 天正边界诚实：检测 ACAD_PROXY_ENTITY 代理实体→警告文案引导用户天正 TXDC 导出 T3 后再导入（无开源解析，调研定案）。

---

## 三、技术选型（调研落定，PoC 复核）

| 能力 | 选型 | 许可 | 兜底 |
|---|---|---|---|
| DWG/DXF 解析 | `unitedrhino/go-cad`（纯 Go，DWG R9-R2018 读、DXF R12-R2018、自带 PNG 渲染+图框识别） | Apache-2.0 | LibreDWG 子进程（仅 PoC 不过时启用）或 DWG 降级为引导转 DXF |
| 前端查看（一期） | **后端分幅 PNG 翻页**（go-cad RenderAllSheets，PoC 已证；零前端依赖零许可风险） | — | 交互式矢量查看器（@mlightcad/cad-simple-viewer MIT / vagran MPL-2.0）推二期——**PoC 实测 mlightcad npm 发布矩阵自相矛盾（2026-10-23），当前不可干净集成，挂上游修复** |
| PDF 光栅化 | 复用既有 soffice/pdftoppm 通道 | 既有 | — |
| 语义/问答 | 复用 vision 管线（本地 herdsman Qwen3.6 或识图功能绑定的云端引擎） | 既有 | — |
| 中间格式 | 后端统一产出规范化 DXF 文本 + 摘要 JSON + 光栅化 PNG | — | — |

风险点：go-cad 2026-09 才开源（59★，公司主体维护）——**第 0 期 PoC 是硬门禁**。

---

## 四、分期刀路

### 第 0 期：PoC 门禁（**2026-10-23 已收官：合成样本范围 PASS，报告见 `docs/gaea-cad-poc-report-2026-10.md`；真实 DWG 验证不阻塞一期，挂用户样本**）

| # | 验证项 | 通过标准 |
|---|---|---|
| 1 | go-cad 解析真实图纸：用户出 3~5 张（含 R2018 dwg、dxf、天正导出 T3） | 实体/图层/文本解析对照 LibreDWG dwg2dxf 与 CAD 快速看图目检，无错漏；天正图能识别代理实体并诚实报警告 |
| 2 | go-cad 光栅化 PNG（中文字形）与图框识别 | 目检可读；图框图号提取正确 |
| 3 | mlightcad cad-simple-viewer 嵌 React（vite 沙箱） | 渲染上述图纸的 DXF；>5MB 图纸交互不卡（首帧 <3s） |
| 4 | 本地 Qwen3.6-35B 图纸问答金标：5~10 张图纸 ×5 问（表格/图名/说明提取） | 表格/文字类目测可用即达标（调研基线 0.85+ 在云端旗舰；本地档预期略低，据实定默认路由：本地直连 vs 引导绑云端） |

产出 `docs/gaea-cad-poc-report-2026-10.md`；**PoC 不过 → 一期降级为「DXF+PDF only，DWG 显示引导转换文案」**，其余刀路不变。

### 一期（P0）：图纸查看 + 信息提取 + 问图纸（**2026-10-23 代码落地并二批收口：internal/gaea/cad 包（cad/render/pdf/count）+ GaeaCadOpen/Raster/Texts/Count 四只读绑定 + CadPanel 右栏可选卡（defaultEnabled=false，启用开关经 paneEnabled store 接线三入口）+ 最近图纸（内存态）+ 文本全文复制 + cad-takeoff 内置技能；Go/前端全量测试/tsc/drift 全绿；待真机走查+发版**）

**实现与规划的偏差（以本节为准）**：

- 绑定面四只读：`GaeaCadOpen`（判型+解析+摘要+全图框分幅 PNG）、`GaeaCadRaster`（单框取图/加宽重渲，喂问图纸）、`GaeaCadTexts`（矢量文本全量清单）、`GaeaCadCount`（通用符号统计：块引用计数精确+邻近文字众数自动命名+图框归属+图层线长）。原规划的 `GaeaCadDxf`（DXF 文本出产物）随 PoC §8.5 WriteDXF 证伪**退役**（查看走后端分幅 PNG，前端零 DXF 传输面）；`GaeaCadExtractText` 更名为 `GaeaCadTexts`。
- `GaeaCadCount` 为用户拍板的方向补充：**gaea 只做识图工具（数据供给），算量表交 AI 模型**——新增内置技能 `cad-takeoff`（待确认清单先行/数量只来自工具数据/系数明示/组价宁缺勿误），前端「交 AI 算量」把识别数据打包发对话流。
- 产物落盘：`<gaeaCwd()>/.gaea/cad/<内容键>/`（内容键=路径|大小|mtime 的 SHA1 前 16 hex，图纸变更即换键重渲；LRU 保留 8 张图纸，gitignore 已收）。
- 前端单组件 `CadPanel.tsx`（规划中的 CadViewer/CadInfoPanel 并入一体：图框 chips 质量闸排序+分幅图+问图纸+文本清单+符号统计），`cadRecent.ts` 最近图纸内存态（不落库，重启即清）。
- 「可选使用」开关：`lib/paneEnabled.ts`（zustand 单源包 workspaceTabs 既有启用集 API）——欢迎网格只出启用卡、欢迎页「管理」Popover 逐行开关、＋下拉/命令面板未启用项显式标注「（未启用）」点击即启用。

**Go 侧**：新包 `internal/gaea/cad`（纯函数内核，无状态，绑定层薄壳）：

```
internal/gaea/cad/
  cad.go        // Open：判型（DWG 魔数兜底）→ go-cad 解析 → 图框识别 + 文本收集
                // （\U+ 转义解码 DecodeUnicodeEscape）+ 天正边界警告；RenderSheets/RenderSheet
  pdf.go        // OpenPDF：poppler pdftoppm 页图（与文件预览同源），产出同构 OpenResult
  count.go      // Count：通用符号统计（块计数+众数命名+图框归属）+ 图层线长
```

**绑定面**（4 个，全 `Gaea*` office 门面、空间 "work"、只读）：

| 绑定 | 签名 | 职责 |
|---|---|---|
| GaeaCadOpen | (path string) → JSON | 判型+解析+摘要（图框图号/图层/警告/file 元信息）+ 全图框分幅 PNG 落盘（DWG/DXF=go-cad；PDF=pdftoppm 页图） |
| GaeaCadRaster | (path string, sheet, width int) → JSON | 取第 sheet 图框/页 PNG `{pngPath,dataUrl}`；width>0 加宽重渲（问图纸红线 6000），pngPath 直喂 RecognizeImage |
| GaeaCadTexts | (path string) → JSON | 矢量文本全量清单（图层/坐标/字高/图框归属；截断上限 2000）；PDF 报错引导问图纸 |
| GaeaCadCount | (path string) → JSON | 通用符号统计 `{items,layers,warnings}`；算量表数据源（交 AI 算量） |

「问图纸」**零新绑定**：前端取 `pngPath` → `app.RecognizeImage(pngPath, 图纸问答提示词)`（加宽重渲红线，PoC §五）。

产物落盘：分幅 PNG 落 `<gaeaCwd()>/.gaea/cad/<内容键>/`（内容键=路径|大小|mtime SHA1 前 16 hex；LRU 保留 8 张图纸）；图纸本体与提取结果**不入 SQLite**。

**前端**：组件族 `gaea/components/cad/`：

- `CadPanel.tsx`：主面板一体（导入区/最近图纸 chips/图框列表按 passRate 质量闸+自动选最优/分幅图预览+加宽重渲/问图纸+发对话/文本清单+全文复制/符号统计+交 AI 算量），WORKSPACE_TABS + RENDERERS 注册（id `cad`，defaultEnabled=false）。
- `cadRecent.ts`：最近图纸内存态单源（不落库，重启即清，最多 5 条）。
- 交互式矢量查看器（mlightcad/vagran）推二期视上游修复再上（PoC §8.6：一期=后端分幅 PNG）。
- pickFile：`app.PickFiles("dwg,dxf,pdf")` 扩展名白名单+后置复核；Composer 附件轻入口未做（一期减面）。

**接线清单**（漏一即 drift/tsc 报）：三份词典加键（zh 值=界面中文原样）；`methods_office.ts` 四 mock；`spaceBindings.ts` GAEA_METHOD_FACETS 四行认领；`bridge/cad.ts` 接口 + `mappings.ts` 短名映射（CadOpen/CadRaster/CadTexts/CadCount）。

**验收**：① 真实 dwg/dxf/pdf 三格式进→看→提→问全链走通；② 关闭卡后零残留（不注册监听/不预载）；③ 天正图诚实警告；④ 可选卡默认关：欢迎网格不出卡，管理 Popover/＋下拉/命令面板可开启；⑤ 既有识图四消费点回归零变更；⑥ 组件端到端金样（mock Proxy 模式）+ 契约锁 + drift PASS；⑦ 走查：办公空间开关卡→导入→问图纸进对话流，console 零错。（原 ④「>16MB DXF 拒绝提示」随 GaeaCadDxf 退役删除。）

### 二期（P1）：表格结构化 → 造价联动

- `GaeaCadExtractTables`：门窗表/钢筋表/做法表→结构化 JSON（VLM + 矢量文本坐标对账降幻觉——矢量文本做锚，VLM 做结构）。
- 联动面：提取表→确认弹窗（用户逐表确认，记忆写入确认红线）→导出 xlsx/复制；进阶挂清单弹窗（只读参照 MatchResourceByTitleSpec 宁缺勿误，不自动写库）。
- CAD 读图技能：inline skill `cad-drawing`（图例/平法/图框常识，参照 office-gongwen 纪律分工）。
- 验收：金标图纸集上表格字段准确率报告；确认制全链；造价域零未确认写入。

### 三期（P2）：观察池（不立项，挂条件）

符号 spotting（FloorPlanCAD 底座+本地引擎训练）；批量归档（图框识别批量出 PDF/改名）；扫描件理解（走光栅化问图纸天然覆盖，矢量化不做）。立项须用户拍板。

---

## 五、测试与验收策略

- **Go**：cad 包纯函数单测 + golden（解析摘要 JSON 快照）；fixtures 纪律——测试图纸用最小样本（几 KB~几十 KB 自造/裁剪，**勿把用户真实图纸整图入库**；大样本走 `GAEA_WALKTHROUGH` 沙箱外置）。
- **前端**：组件端到端金样（vi.mock bridge Proxy 模式）覆盖开关卡/导入/拒绝上限/问图纸消息拼装；i18n zh 值与界面中文逐字一致（零视觉变化复绿范式）。
- **契约**：bindingSignatures 再生 + spaceBindings 认领 + drift 锁（`check-bindings-drift.ps1`）。
- **真机走查**：办公空间（ml-space-*）开卡→三格式导入→问图纸→关闭卡；沙箱纪律沿用（隔离 UserHomeDir、token 不落）。

## 六、风险与坑清单

| 风险/坑 | 对策 |
|---|---|
| go-cad 太新（59★） | 第 0 期硬门禁 + LibreDWG 子进程/降级 DXF 双兜底 |
| Wails 大字符串传输 | DXF 走 base64 且 16MB 上限；更大图纸提示拆分 |
| mlightcad 大图性能 | PoC #3 实测；不达标换 vagran/dxf-viewer 或后端页图模式 |
| 天正代理实体 | 诚实警告+TXDC 引导，不假装解析 |
| GPL 传染 | 选型表已全部排雷；新增依赖必须先过许可审查（列入 PR 检查项） |
| 新绑定漏接线 | 六步清单逐项勾销；提交前 `git status` 对照 ?? 勾销（既有坑） |
| 本地模型图纸问答准确率未知 | PoC #4 金标定路由，不达标默认引导云端识图绑定 |
| 版本三处被外部同秒改写 | 发版走 release.ps1 既有流程，DSH 并发时轮询不抢跑 |

## 七、发版与关账

- 一期一刀发版（预计 v4.524+ 档位，随当时版本节奏），release.ps1 台账自述行同步；CHANGELOG 条目+releases/README 三处同步（既有纪律）。
- 每期关账更新本文件状态行；PoC 报告单独成档。


## 追记（2026-10-28）：agent 工具 cad_ask——对话侧读图通道

**实弹翻车**：用户早间在办公对话里让 AI 读 PDF 版 CAD 图纸，AI 读不了。取证：当日
全局主日志零 CAD/识图调用（无 CadOpen/CadError/RecognizeImage；CAD 链路失败有
LogFrontendError 留痕纪律，成功不留痕）。定性三重断点：①附件识图只认图片
（PDF 归 file 型不走视觉）；②agent 无 CAD 工具可自己调（vision 工具只吃图片路径、
doc_render 走 LibreOffice/poppler——吃不了 DWG/DXF 原生格式，PDF 渲染 DPI 也不受控）；
③CAD 识图能力只活在工具面板 UI 里，对话侧完全够不着。

**根修**：`internal/gaea/tool/builtin/cad_ask.go` 注册 agent 工具 `cad_ask(path,
question, sheet?)`——把本计划工具面板同款管线一步带给模型：
- DWG/DXF：原生解析 → 全图框默认宽渲图（passRate 质量闸回填）→ 缺省选质量最好
  图框 → 6000px 加宽渲图（识图喂图红线）→ vision.RecognizeImage；
- PDF：按页渲染，ask 档 DPI=200（面板默认档 110 的近两倍——对话问图纸就是为了
  读字）；sheet 参数显式选页；
- 多图框返回图框清单（名称+清晰度），模型可换 sheet 再问；ReadOnly+ModelBacked
  （mt_ 运行记录面，同 vision 工具）；渲染走每次调用临时目录 defer 清理（UI 侧
  cadToolCache 缓存不适用于 agent 单次问答形态）。
- 注册面三表同批：compactDesc/compactSchema（CompactDescriptor 无条件替换模型面）
  + spaceTags（SpaceWork，GA2-04 守卫 TestSpaceTagsCoversAllBuiltins 当场逮住漏表
  ——守卫体系第 N 次立功）；config 无 tools.enabled 白名单时 boot 全注册路径自动
  生效（用户 config 实测无该名单）。
- **面板问图纸 PDF 分支同刀收口（同日）**：cad 包新增 `RenderPDFPage`（单页按
  DPI 渲染，复用 docmd.RenderPDFPageRange 的 1..1 页区间）；`GaeaCadRaster` PDF
  分支 width>0（问图纸高清档）改走单页 200dpi 重渲（`cadAskRasterDPI`，落缓存
  目录 ask 前缀不覆盖默认页图），与对话侧 cad_ask 同口径——遗留观察池销号。
  守卫=TestRenderPDFPage（DPI 传播实证：36dpi→306px / 144dpi→1224px；越界页
  如实报错）。坑：pdftoppm 单页输出无零填充，同前缀两渲同名覆写——测试量宽度
  必须渲完立刻量。
