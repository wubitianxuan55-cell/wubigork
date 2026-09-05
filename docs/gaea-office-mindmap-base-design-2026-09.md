# gaea 办公板块新增能力设计：思维导图 + 多维表（gaea 原生）

> 状态：M1 / B1 已实施（2026-09-05，均零绑定面，vitest 1898 全绿）；M2 / B2 待拍板。
> 2026-09-05 用户拍板办公板块新增两能力；承接
> docs/gaea-dsh-univer-office-distill-plan-2026-09.md 方针「取道不取器」——dsh-univer-office
> 的 Base（多维表格）/Board（含思维导图分支）为 Univer Pro 承载，本文裁定其**能力本身**
> 以 gaea 原生形态实现，引擎/私有格式零引入。

---

## 0. 结论速览

- **思维导图：markdown 大纲 = 权威格式**。零新格式——agent 用既有 writefile/editfile 即可
  创建与修改，行级 diff/版本对比/导出全部白得；预览新增「思维导图视图」（自研轻量交互树，
  复用 AgentTree/AgentNetworkCard 树布局先例，零新依赖）；mermaid mindmap 继续承担正文
  内嵌静态小图。
- **多维表：xlsx 为数据底座 + `.gbase.json` 视图配置 sidecar**——「多维表 = 数据的视图，
  不是新格式」。分组/筛选/排序/着色（看板列 B2）叠在 xlsx 上，Excel/WPS 继续可开；
  XlsxPreview 虚拟滚动、Plan→Apply、原生图表、CrossEmbed、版本对比、Verifier 全部直接
  继承；agent 建表走既有 xlsx 管线、建视图走 writefile JSON，**零新绑定**。
- 刀序：M1（导图视图切换）→ B1（多维表 v1）→ M2（画布编辑）/ B2（看板+字段面板）；
  M1/B1 足迹互斥可并行，建议与 U 系列（蒸馏线）穿插。

---

## 1. 上游参照（只取体验口径）

- Base：表/字段/记录/多视图，筛选/排序/分组，公式字段，结构化校验，导出 xlsx/csv。
- Board：节点/连线画布（mind 分支=思维导图），结构校验，不可导出文件。
- 蒸馏两件事：**视图丰富度**（同一份数据多视图并存、分组/看板）、**结构校验纪律**
  （agent 写前校验、坏结构诚实报错）；不取：Pro 引擎、协作、`.univer` 容器。

---

## 2. 现状盘点（逐项已核）

| 基建 | 现状 | 对本设计的意义 |
|---|---|---|
| mermaid | ^11.4.1（含 mindmap 图型）；Markdown.tsx code 分派缝已有 | 正文内嵌静态导图零成本 |
| 树形 UI | AgentTree/AgentNetworkCard 自研 SVG 树（折叠/展开） | 导图布局与交互先例 |
| 表格 | XlsxPreview（虚拟滚动/直编/合并单元格/NumFmt）+ Plan→Apply + 原生图表 + xlsxCellDiff/VersionTimeline | 多维表底座全套现成 |
| 存储 | modernc.org/sqlite（knowledge/Hephaestus.db 先例，纯 Go 无 cgo） | 备选后端；本设计 v1 用工作区文件 |
| agent 工具 | writefile/editfile/readfile/format_convert/chart_gen；产物登记表；预览 kind 分派（FilePreview/FilePreviewModal） | 导图/多维表零新工具即可交付 |
| excelize | 数据验证（dropdown）原生支持 | select 字段映射通道 |

---

## 3. 思维导图（M 线）

### 3.1 格式裁定：markdown 大纲 = 权威

- 一棵导图 = 一个 `.md`：首个 H1 为根节点，H2/H3 与嵌套列表为层级（解析细则实施时定：
  推荐「嵌套列表优先、标题作次级根」，纯函数解析 + 单测钉死口径）。
- 不用 `.gmind` JSON 的理由：LLM 写 JSON 树错误率高、用户不可直读、diff/导出全要另做；
  markdown 三项全免。
- 视图态（折叠/缩放/布局方向）存 localStorage 不进文件；节点颜色/图标 v1 不做。

### 3.2 交互视图（M1，小刀）

- FilePreview / FilePreviewModal 对 `.md` 增加「文档 / 导图」视图切换（头部 toggle，偏好记忆）；
  渲染 = 自研轻量交互树（right-facing 逻辑布局、折叠、缩放/平移、一键回中），
  复用 AgentTree 布局先例，零新依赖；markmap（MIT，带 d3）列备选——polish 高但引入 d3，
  若拍板走此路须懒加载 + 体积预算审计。
- 场景分工：正文里 ```mermaid mindmap```（静态小图）照旧；文件级导图（可交付、可迭代）
  走交互视图。

### 3.3 agent 侧（零新工具）

- office-edit 技能（U1）增加 mindmap 分节：大纲写法、层级纪律（≤4 层、每节点 ≤20 字、
  同层 ≥3 项）、大导图分文件、改图 = readfile+editfile 定点改行（宁拒不误改口径沿用）。
- 写盘即进产物登记/预览，无额外接线。

### 3.4 M2 画布编辑（拍板后）

- 画布上节点增删改/拖拽挂载 → 回写大纲文本（画布态→大纲文本 diff→保存，复用 md 内联
  编辑 Ctrl+S 状态机先例）；快捷键 Tab/Enter/Delete。
- 实施前拍板：自研回写 vs mind-elixir（MIT，全功能编辑器但自有数据格式需双向转换）。

---

## 4. 多维表（B 线）

### 4.1 架构裁定：xlsx 为底 + `.gbase.json` 视图 sidecar

- 数据永远在 `.xlsx`：字段=列、首行=表头；select 字段用 excelize 数据验证（下拉）落地；
  公式列就是 xlsx 公式（LibreOffice 重算通道已有）。
- 视图配置存同目录同名 `.gbase.json`：
  `{ "views": [ { "id", "name", "type": "grid"|"board", "filter", "sort", "groupBy",
  "colorRules", "cardFields" } ] }`
- 为什么不建原生 .gbase 数据格式：①用户数据保持通用格式（Excel/WPS 可开，文件是权威）；
  ②XlsxPreview/Plan→Apply/图表/版本对比/Verifier 全部白得；③agent 通道全现成；
  ④SQLite 后端把数据藏进 .db 违背「交付物在工作区可见」哲学。边界如实标注：
  字段类型系统弱于 Airtable（无关联字段/跨表引用）——v2 口：需求出现时另案评估原生格式。

### 4.2 视图能力（B1 v1，中刀）

- XlsxPreview 头部视图切换：**表格（现状）/ 分组视图**；筛选器（列+条件构建器）、排序、
  条件着色（值规则）。配置读写 `.gbase.json`（保存状态机复用 md/text 内联编辑先例）。
- 分组 = 按 groupBy 列值分块折叠（与虚拟滚动兼容：块内虚拟、块头固定）。
- 坏配置容错：JSON 坏/列名失配 → 降级表格视图 + 红横幅提示（genui 红横幅口径）。

### 4.3 agent 侧（零新绑定）

- 建表：xlsx 管线既有 + 技能条款（首行表头=字段名、select 列写数据验证、一表一主题）；
- 建视图：writefile `.gbase.json`（schema 进技能 Body，JSON 自检四步同 genui 口径；
  视图按列名引用，禁止按列序）；
- 改数据：走既有 xlsx Plan→Apply（审阅制天然继承）。

### 4.4 B2（拍板后）

- 看板视图（select 列=泳道、行=卡片；拖拽改值 = 单元格写，走 Plan→Apply 通道确认）；
- 字段类型面板（列↔类型映射 UI、着色规则构建器）；画廊视图；
- 可选 `base_view_validate` 只读工具（若 agent JSON 错误率实测偏高；新增按 pptx 刀1 先例）。

---

## 5. 共同底座与验收

- 版本对比：`.md` 行级 diff 已有；`.gbase.json` 走文本 diff；xlsx 全套已有——VersionTimeline
  零改动或仅注册扩展名。
- 门禁：vitest（大纲解析纯函数 / .gbase.json 读写与失配降级 / 分组视图状态机 / 导图交互树）+
  tsc -b + eslint 0 + Go 全量 + drift PASS（M1/B1 目标绑定面 579 不变）+ ?mock=1 走查。
- 技能索引预算：mindmap/base 条款并入 office-edit（与 U1 同刀或紧随，拍板项 3）。

---

## 6. 刀序

| 刀 | 内容 | 绑定面 | 档位 |
|---|---|---|---|
| **M1** | `.md` 导图视图切换 + 自研交互树（解析纯函数+单测先行） | 0 | 小刀（✅ 已实施 lib/mindmap.ts + MindMapView，双入口 FilePreview/FilePreviewModal） |
| **B1** | `.gbase.json` schema + XlsxPreview 视图层（分组/筛选/排序/着色）+ agent 技能条款 | 0 | 中刀（✅ 已实施 lib/gbase.ts + GbaseGroupedView；**agent 技能条款随 U1 office-edit 补**） |
| **M2** | 画布编辑回写大纲（拍板后） | 0 | 中刀 |
| **B2** | 看板拖拽 + 字段面板 + 可选 validate 工具（拍板后） | 0~1 | 中刀 |

M1/B1 足迹互斥可并行；与 U 系列穿插排期，M1 最小有感可先行。

---

## 7. 拍板项

| # | 决策 | 推荐 | 备选 |
|---|---|---|---|
| 1 | 导图 v1 渲染 | 自研轻量交互树（零依赖，AgentTree 先例） | markmap（MIT+d3 懒加载） |
| 2 | 多维表架构 | xlsx + 视图 sidecar | 原生 `.gbase` 独立格式（列 v2 口） |
| 3 | 技能承载 | 并入 office-edit 三分节+mindmap/base 两分节（单技能） | 独立 mindmap/base 技能 |
| 4 | M2/B2 首轮 | 不随首轮，v1 观察后定 | 随首轮交付 |
| 5 | 大纲解析口径 | 列表优先、标题作次级根（单测钉死） | 纯标题树 |

---

## 8. 风险与对策

| 风险 | 对策 |
|---|---|
| 大纲解析歧义（列表/标题混排、缩进脏数据） | 解析纯函数 + 单测钉死口径；技能约束写法；解析失败红横幅降级文本视图 |
| xlsx 承载多维表的类型上限（无关联/引用字段） | 如实定位「视图层」；v2 口另案；不宣传 Airtable 全能 |
| `.gbase.json` 与 xlsx 漂移（改列名/删列） | 视图按列名引用；渲染端失配降级+提示；技能条款「改结构须同步改视图」 |
| 视图配置被 agent 写坏 | JSON 自检四步 + 渲染端容错 + B2 可选 validate 工具 |
| markmap/d3 若拍板引入的体积负担 | 懒加载纪律 + 体积审计；默认自研线不触发 |
| 与 XlsxPreview 既有直编状态冲突（切换视图丢编辑态） | 视图切换不动表格数据态；看板拖拽写入走 Plan→Apply 审阅 |

---

## 9. 明确不做

- 不引入 Univer Base/Board/协作引擎；不做多人协同/实时共享；
- v1 不做节点颜色/图标、关联字段、跨表引用、公式列 UI（公式 = xlsx 原生既有能力）；
- 不做 `.gmind`/`.gbase` 私有数据格式（视图 sidecar 除外——它不是数据）；
- 思维导图不做自由画布（Board 式任意连线/形状）——那是 diagram_gen/白板另案范畴。

---

## 10. 参考

- 上游口径：docs/gaea-dsh-univer-office-distill-plan-2026-09.md §1/§4（Base/Board 体验与裁定）
- 既有先例：AgentTree/AgentNetworkCard（树布局）、XlsxPreview + gaea_xlsx_edit（Plan→Apply）、
  Markdown.tsx mermaid 缝、genui 红横幅降级口径、md 内联编辑 Ctrl+S 状态机
- 相关设计：docs/gaea-office-upgrade-plan-2026-09.md（右栏工作台/预览双入口）
