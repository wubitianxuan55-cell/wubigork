# gaea 蒸馏 dsh-univer-office 长期规划：办公板块「契约照收、引擎不换」

> 状态：规划版（调研与规划，未改任何功能代码）
> 日期：2026-09-05 · gaea 基线 v4.96.0（绑定面 579）
> 上游：github.com/dream-num/dsh-univer-office（Apache-2.0）· v0.2.14 · shallow clone 快照（2026-09-05）
> 上游调研方式：只读克隆至临时目录逐文件通读（双语 README / 架构 ADR / turn 投影设计 / 14 个工具定义 /
> 8 个 SKILL.md / SQLite schema / license 常量 / 对比查看器包）；gaea 侧按既有设计文档与源码逐缝核对。

---

## 0. 摘要

dsh-univer-office 是 DreamNum（Univer 官方团队）给 DeepSeek Harness（DSH）做的**办公插件**：
Univer 引擎 + 隔离草稿 worktree + 14 个结构化工具 + 8 个技能 + 回合审阅卡/实时浮窗 +
「结构回读→布局 lint→截图取证」验证闭环。它证明了一件事：**桌面 AI 助手交付 Office 真编辑
已经不是空白**（修正 docs/market-research-2026-09-05.md 战略层结论），但其实现路线 =
中间格式 `.univer`（SQLite）+ Univer Pro 授权（水印/导入体积限制，上游自带 90 天轮换 dev
license）+ Node ≥22 + 无头 Chrome + libsql 重依赖。

gaea 的蒸馏总方针：**契约照收、引擎不换**。

1. 上游真正可搬运的资产是**行为契约**：草稿生命周期（draft→ready→merge/discard，用户审批）、
   verify 循环（「工具成功≠正确」，回读+lint+截图）、回合投影（生命周期/写入/读取三类操作
   独立归约）、浮窗拉起规则（写动作才弹、用户关闭优先）、`Error [CODE]` 错误路由、按 Unit
   分册的技能纪律——这些全部与引擎无关，蒸馏进 gaea 现有三件套栈（docx 修订制 / xlsx
   Plan→Apply / pptx 真编辑设计）；
2. **引擎不换**：gaea 的差异化恰恰是自研 OOXML 字节级手术（零中间格式、零授权运维、保真
   不重写）——对照上游后这条路线的价值更清晰；Univer OSS 交互面降级为试点拍板项（U4），
   Pro 依赖（导入导出/Slides/Base/Board/协作/语义 diff/打印）默认不做；
3. 对外口径修正：「全场空白」改为「第三方桌面助手空白已被 DSH 官方插件打破；gaea 是
   『编辑直接发生在原文件』的自研保真路线」。

排期上让位进行中队列（genui P2–P5 优先），本规划 U1/U2 为独立小中刀，U3 即 pptx 真编辑
既有设计（吸收上游纪律，不重复设计），U4/U5 为拍板后远期。

---

## 1. 上游能力面（dsh-univer-office v0.2.14 快照）

### 1.1 定位与进程模型

- 单 npm 包（Apache-2.0），DSH bundle 形态：Host（Node，可信）+ Gateway 子进程（协作域服务，
  默认端口 9080 起步逐个 +1）+ 一次性 Unit Content Worker（无头 Univer，执行完即退）+
  Render Machine（无头 Chrome/Chromium，截图/PDF/lint 文本度量）+ Viewer（浏览器端 Univer
  应用）+ Client（DSH 会话侧投影）。Node ≥22.19，依赖 libsql、puppeteer-core。
- 分层纪律（其架构 ADR 明文）：Client 不 fetch（HTTP 集中 api 层）、Client 不解析自由文本
  （只消费结构化工具事件）、Viewer URL 由 Host 按会话 cwd realpath 授权后下发的不透明值、
  Gateway 是提交与 revision 的唯一依据、merge/discard 模型不得自决。

### 1.2 领域模型

- **`.univer` 文件 = 多 Unit 容器**（SQLite/libsql 持久化，v0/v1/v2 格式）：一个文件可同时
  装 Sheet / Doc / Slide / 多维表格 Base / 画布 Board，跨 Unit 公式与内容引用。
- **trunk（主线）+ worktree（隔离草稿）**：所有内容写入只能落显式 draft worktree；
  生命周期 `draft → ready →（merge | discard）`，ready 后拒绝写入需 `reopen`；
  merge/discard 终态不可复用，且**必须用户明确请求并经宿主审批**。
- 工具结果携带 file/worktree/unit 结构化标识 → Client 据此恢复预览目标，不猜。

### 1.3 工具面（14 个，全部结构化 schema + 稳定错误码 + 纯 presentation）

| 工具 | 作用 | gaea 对应物 |
|---|---|---|
| univer_new | 创建空 `.univer` 容器（不覆盖） | 不适用（gaea 以原 OOXML 文件为权威） |
| univer_status | 发现 Unit 与 worktree 状态（一切工具的入口） | 🟡 分散在 docx/xlsx/pptx 读接口 |
| univer_worktree | create/ready/reopen/merge/discard | ❌ 草稿会话概念（证据链只兜底安全） |
| univer_unit | draft 内增删 Unit | 不适用 |
| univer_import | xlsx/csv/tsv/docx/pptx → Unit（Pro exchange） | 反向不适用（gaea 直接编辑原文件） |
| univer_inspect | 结构化读取范围/结构/概览/元素详情 | 🟡 xlsx opsJSON、docx/pptx 大纲（读侧已齐） |
| univer_execute | 跑版本匹配的 Facade JS 写 draft | ❌ 不可照搬：gaea agent 在 Go，无 JS 运行时；gaea=Go 工具直写 |
| univer_export | Unit → Office 文件（Pro exchange） | 反向不适用 |
| univer_lint | Slide 三规则布局检查（越页/越容器/文本重叠，真实字形度量） | ❌（pptx 设计刀3 只做文本 diff） |
| univer_compile_svg | SVG 按浏览器文本度量编译成 Slide 页（生成主路径） | 🟡 理念可借鉴进 pptx 生成技能 |
| univer_screenshot | 渲染 PNG 证据回传模型（≤30 页、像素上限 16,777,216） | 🟡 Verifier 通道 B 像素 diff（审阅侧）；模型侧 vision 已有 |
| univer_print_pdf | Sheet/Doc/Slide/Board → workspace PDF | 🟡 soffice→PDF 通道已有 |
| univer_api | Facade API 关键词查找 + 精确引用（不许猜签名） | 对应理念：gaea 工具参数纪律 |
| univer_resources | 内置图标/Logo/Emoji/插画 registry 查找导出 | ❌（低优先，不立项） |

### 1.4 技能面（8 个：univer 总纲 + sheet/doc/slide/base/board 分册 + embed + 跨 Unit 公式）

可直接迁移的工作流纪律（按 gaea 工具名改写后进内置技能）：

- **入口纪律**：先 status（发现 ID 与状态）再动；内容写入前必须加载对应 Unit 分册；
  每次写入要给出完整三元组地址。
- **「工具成功≠正确」**：写入后必须结构化回读（inspect）并用任务断言核验；
  截图只补视觉证据，不替代结构回读。
- **Slide 生成主路径 = SVG**：先写 `spec.md`（版式/结构类型/单页核心信息/逐字文案/常量
  一次定死），逐页闭环（本页过 lint 才进下一页），改页用 replace 不用 add 叠加；
  每个变更页 lint + 最终逐页截图（≤5 页一批）过 8 项检查清单（越界/溢出/重叠/遮挡/对比度/
  缺内容/箭头断连/跨页一致性）；缺陷按模式全 deck 排查。
- **状态机纪律**：ready 拒写、终态不复用、继续修改先 reopen；merge/discard 只在用户明说时。
- **错误路由**：失败按 `Error [CODE]` 路由恢复（GATEWAY_UNAVAILABLE 重试一次读 /
  权限类换路径不重试 / 状态类先刷 status），不解析自然语言。

### 1.5 验证闭环（上游产品能力的核心）

```
写入(execute/compile_svg/import) → inspect 结构回读 → lint 布局三规则
→ screenshot 逐页 PNG 回传模型人工核 → (导出/打印只在核验后) → ready 交用户审
```

- lint 三规则保守判定：文本出页面（必修）、文本逃出不透明容器、字形带重叠（查证后再定）。
- 截图与 lint 与结构回读三者互不替代；「声称已视觉验证」仅限当次实际看过的图。

### 1.6 UI 投影（回合卡 / 浮窗 / compare，纯 reducer 设计）

- **Turn 投影**：按 callId 配对 tool/call+result，三类操作独立归约——生命周期操作按序推进
  结论（outcome ∈ trunk/draft/ready/merged/discarded/unchanged）；写入只标 changedContent
  与优先 Unit，**不覆盖已成功的终态结论**；读取（status/inspect/lint/export）不产生转换，
  最后一个显式读取 scope 只决定默认查看目标。失败操作记录但不提交转换。
- **统一回合尾卡片**：每 `Turn×file` 一张卡，header（文件名+worktree 名+路径+状态+全屏/折叠）
  + body（变更 Unit chips + 完整 Viewer）；**卡片不复制 Viewer 内置的提交/丢弃/合入按钮**；
  历史卡同布局默认折叠；文件被确认删除则不渲染；加载失败只用同布局的不可用态，不回退旧卡。
- **实时浮窗**：仅 `new`/worktree create/reopen/ready/写入类操作主动拉起；纯读取永不拉起
  已关闭浮窗；用户手动关闭优先并清除打开意图；跨回合保留「用户保持打开」意图，查询失败
  不当终态；同文件同回合最多一个浮窗；终态清除意图。
- **compare**：有界固定比较会话（创建时钉死两侧 revision head），右侧=worktree、左侧=trunk
  或另一活跃 worktree；任一侧 live head 变化只标 stale，用户显式刷新才建新会话；
  语义 diff 走 Pro History，返回带分页/筛选/稳定实体身份的 JSON。

### 1.7 存储与依赖（重）

`.univer`=SQLite（collaboration_units/changesets/snapshots + worktrees 系列 + history 派生表，
v0/v1/v2 格式探测）；Gateway 与 Worker 用精确版本 insiders SDK 构建；Render Machine 需本机
Chrome（可 `UNIVER_RENDER_BROWSER` 指定）；发布包按平台装原生依赖（rust 公式绑定、Office
转换器、SQLite），不能跨平台复制 node_modules。

### 1.8 授权实质（关键风险证据）

- 插件依赖 `@univerjs-pro/*` 1.0.0-**insiders** 锁版本（exchange 导入导出、协作、History、
  Slides、Base、Board、图表、打印）+ `@univerjs-pro/license`。
- 源码内置常量：**应用自有 90 天开发版 license**（2026-08-14 签发，域名仅 localhost，
  到期 et≈2026-11-12），注释明写「must be replaced on the 90-day rotation」——连上游官方
  都背着授权轮换运维。
- 官方政策：无 license 可试用但有**水印 + 导入体积 + 协作数量**限制（数值不公开）；
  30 天试用 license；正式版联系报价。OSS 核心（sheets/docs 引擎、公式、筛选、排序、numfmt、
  条件格式、数据验证、绘图）为 Apache-2.0；**OOXML 高保真导入导出明确是 Pro**（docx 导入
  官方文档建议用开源解析库自建）。

---

## 2. 社区调研（同步完成，2026-09-05 快照）

### 2.1 引擎生态

| 候选 | License | 形态 | 评估 |
|---|---|---|---|
| **Univer**（dream-num） | 核心 Apache-2.0；Pro 商业授权 | 可嵌入 SDK（sheets/docs/slides/base/board），自定位 "The Office Harness for AI Agents" | 社区主流选择（Luckysheet 官方后继）；Pro 门槛见 §1.8 |
| OnlyOffice DocumentServer | **AGPL-3.0** | 全套件+服务端 | 排除：AGPL 传染违背 gaea 零 AGPL 纪律；服务端重 |
| Collabora Online | MPL-2.0（LibreOffice 内核） | 服务端 | 排除：需常驻服务端；gaea 已用 soffice 做转换/重算，无需整套 |
| FortuneSheet / x-spreadsheet / jspreadsheet-ce | MIT 系 | 轻表格 | 排除：仅 sheet、无 doc/ppt、维护弱 |
| Handsontable | 商业 | 表格组件 | 排除：商业授权 |
| SheetJS CE | Apache-2.0 | 读 xlsx 库 | 备忘：若走 U4 自建桥，读侧原料之一（Go 侧已有 excelize 更顺） |

### 2.2 AI 助手格局修正

- **DSH 官方插件已交付 Office 真编辑**且进入插件市场推荐位（知乎「8 个插件」/掘金「15 款
  插件」均列 dsh-univer-office；DSH 桌面封装项目 13k+ star 量级），社区认知正在形成。
- M365 Copilot / Gemini in Workspace 属 Office 宿主内能力；ChatGPT Canvas / Claude
  Artifacts 是自研 web 画布，非 OOXML 保真交付。
- 结论修正（回填 market-research-2026-09-05.md 战略层）：「六家（lobe-chat 等 web 系助手）
  均无」仍然成立，但**「全场空白」不再成立**——harness 型竞品的官方插件已破局，且
  dream-num 正以 "Office Harness for AI Agents" 定位直接进攻 agent 市场，预计更多竞品
  跟进嵌入 Univer。gaea 的应对不是「也嵌 Univer」（同质化+授权受制），而是放大自研保真
  手术路线的差异化（见 §4.5）。

---

## 3. gaea 现状映射（逐缝核对）

### 3.1 三件套技术栈现状（承 docs/gaea-pptx-edit-design-2026-09.md §1，代码已核）

| 通道 | 底座 | 编辑范式 | 证据链 | 预览 |
|---|---|---|---|---|
| docx | Go 自研字节级手术（internal/office/docxedit） | 框选即改 + Word 修订制 | ❌ 不落（既有欠账） | docx-preview DOM 可框选 |
| xlsx | excelize v2.11 直编 + LibreOffice 重算 | Plan→Apply 审阅制 | ✅ Journal + 快照 + 回滚 | XlsxPreview 结构化 + 虚拟滚动 |
| pptx | 读：python-pptx + soffice/poppler；写：从零生成 | 编辑=0（刀1–刀4 已设计待拍板） | 待刀1 | 逐页 PNG + 大纲卡 |

### 3.2 上游能力 ↔ gaea 对应物总表

| 上游能力 | gaea 现状 | 裁定 |
|---|---|---|
| worktree 草稿生命周期 + 用户审批 | 🟡 证据链快照 + GaeaRollbackRecord 零覆盖回滚兜底安全；无草稿会话 UX | U2 轻量版（§4.2-2） |
| verify 循环（回读+lint+截图） | 🟡 Verifier 通道 B 像素 diff（审阅侧）；工具侧无纪律化 | U1 必收 |
| Turn 投影 + 统一回合卡 | 🟡 DeliverableCards/VersionTimeline 按文件聚合；v4.26 seq+resync 事件底座在 | U2 必收 |
| 浮窗拉起语义 | 🟡 v4.30 自动置前；缺「用户关闭优先/读取不弹/意图跨回合」 | U2 必收 |
| compare 固定会话语义 diff | 🟡 versionCompare 三 kind（pptx 刀3 补第四种）；无钉死双侧会话概念 | 借鉴语义，不引入会话实体 |
| `Error [CODE]` 错误路由 | ❌ 工具错误为自由文本 | U1 必收（限 office 工具） |
| 分册技能纪律 | 🟡 builtin 4 个（genui/format-convert/chart-builder/doc-assemble），无 office 编辑纪律 | U1 必收 |
| univer_execute Facade 编程 | ❌ 无 JS 运行时（gaea agent 在 Go） | 不做（架构差异，非缺陷） |
| 多 Unit `.univer` 容器 | ❌ gaea 以用户原文件为权威 | 不做 |
| 资源库 / Base / Board / 协作 / 打印 | ❌ | 不做/远期（§4.4） |
| CDP/无头浏览器 | gaea 已有 CDP 控制 Edge 通道 | pptx lint 文本度量候选执行器（不引 puppeteer） |

### 3.3 关键架构差异（决定「引擎不换」的硬约束）

1. gaea agent 在 **Go**，无 Node 运行时——上游「无头 Univer worker 跑 Facade JS」整条
   agent 写入通道不可照搬；gaea 的写入权威是 Go 工具（docxedit/xlsxedit/pptxedit）。
2. gaea 前端是 **Wails 系统 webview**——Univer OSS 可作为前端依赖嵌入（用户交互面），
   但 bundle 体积、webview 兼容、快照保真都要试点验证。
3. 绑定面纪律：新增绑定按 pptx 刀1 先例（579→580，gen_bindings 再生），不为蒸馏铺新面。
4. 零商业依赖、零 AGPL 红线：Pro 授权（轮换/水印/导入限制）与 OnlyOffice（AGPL）均撞线。

---

## 4. 蒸馏范围裁定

### 4.1 核心判断

1. 上游的**行为契约层**与其 Univer 引擎解耦良好，可整组搬运到 gaea 既有栈；
2. gaea 与上游的真正分野在引擎路线（自研 OOXML 手术 vs Univer 中间格式+Pro 引擎），
   这条分野是**卖点不是欠账**；
3. 引擎层只留一个低成本试点口（Univer Sheet 交互面，U4），默认不启动。

### 4.2 必收（行为契约，引擎无关）

| # | 上游契约 | gaea 落法 | 刀 |
|---|---|---|---|
| 1 | verify 循环纪律（成功≠正确；回读→lint→截图） | 内置 inline 技能 `office-edit`（三件套分节，gaea 工具名改写）；Verifier/工具验收面引用同清单 | U1 |
| 2 | 草稿生命周期语义 | **轻量版**：不引入持久草稿实体——以「交付物 draft 状态」呈现（产物卡标 draft/ready，来源=证据链 Journal 首次写盘→ready），merge/discard 对应「保留/回滚」既有按钮；docx/xlsx 逐操作范式**不动** | U2 |
| 3 | Turn 投影（三类操作独立归约） | 前端纯函数 reducer（tool_dispatch/result 按 callId 配对），生命周期/写入/读取分离，失败不提交转换；驱动统一 Office 审阅卡 | U2 |
| 4 | 统一回合尾卡片（不复制操作 footer、历史折叠、同布局不回退） | DeliverableCards/VersionTimeline 收敛为 Office 回合卡（沿用 v4.30 产物置前），pptx 刀2 的 PptxEditPanel 挂进同一卡 | U2 |
| 5 | 浮窗拉起规则（写弹读不弹、关闭优先、意图跨回合、终态清理） | 预览 pane 自动置前语义补齐（纯前端状态机） | U2 |
| 6 | `Error [CODE]` 错误码 | office 写类工具（docx/xlsx/pptx 应用）错误统一前缀 + 技能内恢复路由表 | U1 |
| 7 | Slide 生成纪律（spec 先行/逐页闭环/replace 不叠加/8 项清单） | 进 `office-edit` 技能 pptx 分节 + 未来 pptx 生成技能；lint 落地在 U3 | U1/U3 |

### 4.3 选收（引擎试点，默认不启动，拍板后立 U4）

- **Univer Sheet 只读/交互试点**：Univer OSS 前端嵌入；数据来源走 **excelize→Univer snapshot
  只读桥**（Go 侧读 xlsx→IWorkbookData JSON→前端渲染），公式重算/筛选/排序用 OSS 引擎；
  **写回**（snapshot→excelize）列二期，且只允许经 Plan→Apply 通道落盘。
- 如实标注失真清单：图表（Pro）、图片锚定、sparkline、部分 numfmt——桥的覆盖面先行实测。
- 红线：XlsxPreview 全量保留（换壳不换芯），Univer 面是**并存的新入口**，由设置卡开关。

### 4.4 拒绝清单（本规划明确不做）

- Univer Pro 全家（OOXML 导入导出、Slides、Base、Board、协作、History 语义 diff、打印、
  图表）——授权轮换+水印+导入限制+insiders 锁版本，运维与合规双输；
- `.univer` 中间格式作为 gaea 文件形态（用户文件权威性不可让渡）；
- Node sidecar / puppeteer 无头链（gaea 无 Node；截图/文本度量走既有 soffice+CDP Edge）；
- OnlyOffice / Collabora / 商业表格组件（§2.1）；
- 语义 diff 自研替代已定：versionCompare/pptxTextDiff/ChangesDiff（v4.87 管线），不引 Pro History。

### 4.5 差异化口径（对外叙事，回填 marketing 语境）

- 竞品（Univer 路线）：编辑发生在「导入的渲染副本」，交付=导出，导入导出受授权与体积限制。
- gaea 口径：**「AI 直接改你的原文件」**——编辑发生在原 OOXML 字节层（未触碰部分零扰动）、
  修订制/Plan→Apply 审阅、证据链快照回滚、零中间格式零授权依赖。招牌场景不变
  （框选→改→落盘回写），新增强调「保真」与「原文件」。

---

## 5. 分期刀序（长期；版本号以发布节奏为准，genui P2–P5 队列优先）

| 刀 | 主题 | 内容 | 绑定面 | 档位 |
|---|---|---|---|---|
| **U1** | 工作流纪律蒸馏 | ① 内置 inline 技能 `office-edit`（docx/xlsx/pptx 三分节：入口纪律、verify 循环、状态机、错误路由表、Slide 生成纪律；全部用 gaea 工具名与既有管线表述）② office 写类工具错误码规范化（`Error [OFFICE_XXX]` 常量 + 单测）③ 技能索引/提示词不超限校验 | 0 | 小刀 |
| **U2** | 回合投影与审阅收口 | ① officeTurnProjection 前端纯函数（三类操作归约，vitest 全覆盖）② 统一 Office 回合卡（DeliverableCards/VersionTimeline 收敛，draft/ready 状态徽标，不复制操作 footer）③ 预览浮窗语义状态机（写弹读不弹/关闭优先/意图跨回合/终态清理）④ 草稿轻量版（Journal 首写→draft、Plan→Apply 批准→ready 映射） | 0 | 中刀 |
| **U3** | pptx 真编辑（=既有设计刀1–刀4，吸收上游纪律） | 按 docs/gaea-pptx-edit-design-2026-09.md 执行；增量吸收：刀2 验收加「逐页截图回读+8 项清单」、刀3 增加 **pptx lint**（越页/越容器/文本重叠——几何取自 pptxedit 解析数据 + 文本度量经 CDP Edge 或 Go 字体度量，不引 puppeteer） | 刀1 +1 | 中刀×4 |
| **U4**（拍板后） | Univer Sheet 试点 | excelize→snapshot 只读桥 + XlsxUniver 面（设置卡开关、双入口并存）+ 失真清单实测文档；写回二期且只走 Plan→Apply | +1~2 | 中刀 |
| **U5**（远期拍板） | 引擎扩展与授权评估 | 视 U4 反馈与 Univer 商务条款再定：Doc/Slide 面、Pro license 商务评估（若评估，须先过「授权轮换不进发布纪律」这条红线） | — | — |

并行线：U1 内部技能文案线与错误码线足迹互斥可并行；U2 的 reducer 纯函数线与卡片 UI 线
可拆；U3 按既有刀序；U4 前端面线与 Go 桥线可拆。U1/U2 与 genui P2–P5 互不相交，可穿插。

---

## 6. 验收与门禁（每刀沿用仓库纪律）

1. Go：`scripts/test-all.ps1` 全量 0 FAIL；U1 错误码/技能索引单测、U3 pptxedit 包单测。
2. 前端：vitest 计数续增（U2 投影 reducer/卡片/浮窗状态机为主战场）；tsc/eslint 0。
3. 契约：`scripts/check-types-drift.mjs` PASS（U1/U2 绑定面 579 不变；U3/U4 按新增再生）。
4. 真模型走查：U1 后让 agent 实际执行「改表→回读核验→截图确认」一轮并对照技能条款；
   U2 走「多步编辑→回合卡状态机→关闭浮窗不被读取复活」场景。
5. 文档：releases/vN.0.md 如实列欠账；本规划状态行随刀推进更新；.gaea/progress.md 回写。

---

## 7. 决策拍板清单

| # | 决策 | 推荐 | 影响 |
|---|---|---|---|
| 1 | 引擎路线 | 契约照收、引擎不换；U4 仅 Sheet 只读试点且默认不启动 | 资源投向纪律线，避免授权受制 |
| 2 | 草稿概念形态 | 轻量版（产物级 draft/ready 状态映射证据链），不建持久 worktree 实体 | 零绑定面；docx/xlsx 逐操作范式不动 |
| 3 | office-edit 技能形态 | 单技能三分节 inline（对齐 genui 技能先例），非三技能 | 索引 ≤4000 字预算内可控 |
| 4 | 错误码范围 | 仅 office 写类工具（docx/xlsx/pptx 应用+导出），不全局推广 | 小切口可回退 |
| 5 | pptx lint 执行器 | 优先 CDP Edge（既有通道），Go 字体度量备选；不引 puppeteer | 零新重依赖 |
| 6 | U4 是否立项 | 默认否；用户拍板后才进排期（需先实测失真清单） | 避免与 XlsxPreview 双面维护冲突 |
| 7 | 对外口径 | 「AI 直接改你的原文件 / 保真自研路线」；修正「全场空白」表述 | README/营销语境同步 |
| 8 | 竞品跟踪 | dsh-univer-office 与 Univer agents 动向纳入市场调研例行（每季） | docs/market-research-* |

---

## 8. 风险与对策

| 风险 | 对策 |
|---|---|
| 技能纪律与真实工具行为漂移（上游技能描述的是 Univer API，gaea 是 Go 工具） | office-edit 全部用 gaea 实有工具与管线表述；每刀真模型走查校验条款可执行性 |
| U2 回合卡收敛动到 DeliverableCards 既有行为 | 红线：登记表/证据链入口/置前行为不变；卡片只做呈现层收敛；vitest 回归锁定 |
| draft/ready 徽标语义与 xlsx Plan→Apply「批准」撞车 | 拍板项 2 的映射口径写死（首次写盘=draft、批准=ready）；文案如实标注 |
| pptx lint 误报泛滥（上游也只敢做三保守规则） | 只做三规则+逐条「必修/查证后可保留」口径；findings 必须在验收单销账或显式豁免 |
| Univer 试点引发「双预览面」维护负担 | U4 默认不启动；立项时先出失真清单实测报告再拍板；设置卡可整体关闭 |
| 竞品跟进速度（更多助手嵌 Univer） | 每季竞品例行调研；差异化口径落 README/发布说明，不在功能面跟随 |
| 上游 Apache-2.0 引用边界 | 只蒸馏行为契约与纪律文案（改写不照抄技能全文）；如需引用原文段落保留 LICENSE 声明 |

---

## 9. 明确不做（反模式清单）

- 不把 gaea 办公栈重写为 Univer 前端（推倒换壳 = 删除既有编辑能力，撞 UI 简化红线）；
- 不引入 `.univer` 文件格式、协作服务、Gateway 常驻进程、Node/puppeteer 依赖链；
- 不做 Base（多维表格）/Board（画布）/资源库——gaea 已有 genui 面板、chart、表格面，
  边际价值低且均为 Pro；
- 不照抄上游技能全文（工具名、API、错误码全不同；只搬纪律骨架）；
- 不因「竞品有」而扩 binding 面（每处新增绑定须对应 pptx 刀1/U4 的实有功能）；
- 不在对外文案宣称「首家 Office 真编辑」（DSH 插件已在先）。

---

## 10. 参考资料

- 上游：github.com/dream-num/dsh-univer-office（README.zh-CN / docs/architecture.md /
  docs/univer-turn-projection-and-surfaces.md / skills/*/SKILL.md / src/host/tools/definitions /
  src/workers/unit-content/license.ts / packages/unit-comparison-viewer）
- 授权：docs.univer.ai/zh-CN/guides/pro/license（水印/导入体积/协作限制、30 天试用、商用报价）；
  docs.univer.ai/guides/sheets/features/import-export（导入导出=Pro；docx 导入建议开源库自建）
- 生态：github.com/dream-num/Luckysheet（停维护→Univer 后继公告）；univer.ai
  （"The Office Harness for AI Agents" 定位）；github.com/ONLYOFFICE/DocumentServer（AGPL-3.0）
- 社区信号：知乎 DeepSeek Harness 插件推荐文（dsh-univer-office 列入）、掘金 15 款插件文、
  DSH 桌面封装项目（13k+ star）
- 内部：docs/market-research-2026-09-05.md（战略层，本文修正其「全场空白」结论）、
  docs/gaea-pptx-edit-design-2026-09.md（U3 即其刀1–刀4）、
  docs/gaea-office-upgrade-plan-2026-09.md（右栏工作台主线）、
  docs/gaea-dsh-genui-distill-plan-2026-09.md（并行队列）、
  internal/office/docxedit · xlsxedit、internal/app/gaea_pptx.go、
  frontend/src/gaea/components/{DeliverableCards,VersionTimeline,XlsxPreview,DocxPreview,PptxOutline}.tsx
