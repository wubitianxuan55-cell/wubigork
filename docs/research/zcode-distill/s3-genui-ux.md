# S3 · ZCode Agent 交互界面与 GenUI 蒸馏

> 调研对象：`C:\AI\wubigrok\clones\zcode`（只读源码）+ gaea 前端 `frontend/src/gaea`（React + AntD，玄墨 v11 视觉语言）
> 目标：找出能提升「办公对话流」体验的具体设计，给出证据、办公价值、gaea 落点、成本与风险。
> 纪律：只写读到过的文件。所有行号来自本次实际读取；未读到的文件不引用。
> 日期：2026-10-24

---

## 0. TL;DR

**最值得抄的 5 条**（详见 §7 打分表）

1. **上下文占用条**：工具栏常驻触发 + hover 分段面板，分项只给百分比、无数据即整块隐藏 —— gaea 现在只有「上下文」整页，对话时看不见预算。
2. **审批内联 dock**（不是 modal）：钉在对话流里、编号 + 数字键、第 N 行是「拒绝并附意见」、`allowAlways` 直接展示将要放行的命令前缀、子代理发起时加来源徽标。
3. **GenUI 引用三态 + 尺寸契约**：`pending / invalid / ui` 三态渲染、完成门槛（不完整引用不挂载）、guest 侧测高 + 10000px 上限 + 隐藏根滚动条 + 禁止 vh 反馈环。gaea 已有 JSON-spec GenUI，缺的正是这三件边界工程。
4. **每轮文件摘要 + 交付物「作为文件打开」**：轮尾 `N 个文件 +A −D` 折叠摘要（展开只到文件行，不再套 inline diff）+ 分段按钮「默认 App / 打开方式（已装编辑器）/ 在工作区显示 / 复制绝对路径」。
5. **计划卡 + 右栏「计划」tab**：时间线只放裁切预览（`max-h-64` + 底部渐隐），全文进侧栏稳定 tab；轮尾「复制」按轮聚合正文 + 计划全文。

**不该抄的 3 条**

1. **hover-only 控件与五段式工具摘要行**（`opacity-0 → 100` 的箭头、`kindLabel + primary + secondary + diffstat + sourceLabel`）。办公用户不 hover、不学暗号。
2. **双 delivery profile + CLI 权威 queue 全语义**（`desktop-continuous` / `web-remote-replayable`、`autoDrain`、`guide`、`addRules`、owner/lease）。只抄用户可见的两个面：暂停横幅、立即发送。
3. **HTML 沙箱 GenUI 整套**（visualize 技能 + webview 沙箱 + 输出根目录校验 + 符号链接/跨会话读取规则）。gaea 已有 `frontend/src/genui`（JSON spec），再引一套 HTML 沙箱是三条成本线（安全/主题投影/手机降级）。

**判断**：gaea 的对话流**骨架已经 Codex 化**（`ProcessCard` 自动收起、`ToolGroup` 同名折叠、`WorkHeader` 首 token 前不死寂、`TodoCard` 进度、`DeliverableCards` 轮尾交付、`ComposerQueueList` 排队）。所以 ZCode 的增量不在「工具卡长得像不像」，而在四件 gaea 目前为空的事：**①预算可见性 ②审批不打断阅读 ③交付物「打开」的最后一公里 ④过程/计划与正文的分层**。

---

## 1. 调研范围与方法

| 范围 | 读到的入口 |
| --- | --- |
| GenUI 规格 | `docs/specs/gen-ui-{file-summary,inline-sizing,output-directory,styles,external-links}.md` |
| GenUI 实现 | `packages/ui/src/gen-ui/{domain/references.ts,components/GenUiCard.tsx,components/GenUiMessageResponse.tsx,contract.ts,adapters/page.ts}`、`packages/ui/src/hooks/useGenUi.ts` |
| 对话流 | `ToolCallBlocks/{ToolSummaryRow,ToolCallProgressIndicator}.tsx`、`ToolCallBlocks/renderers/{todo,plan-guidance}.tsx`、`TodoPanel.tsx`、`components/ai-elements/{plan,queue}.tsx`、`PermissionDialog.tsx`、`v4/ConversationQueuePanel.tsx`、`chat-input-toolbar/contextUsage.tsx`、`app-shell/SidePaneTabTrigger.tsx` + `sidePaneTabPresentation.ts` |
| 失败与恢复 | `docs/chat/error-banner-restore-dismissal.md`、`docs/ui/chat-user-message-retry.md`、`chat-input-toolbar/display.tsx`（`ChatApiRetryStatus`）、`docs/chat/interaction-behavior.md`、`docs/specs/gen-ui-*.md` 中的 replayable 段落 |
| 产出物 | `docs/ui/task-file-change-summary.md`、`app-shell/workflow-artifacts/{artifactPresentation.tsx,ArtifactOpenAsFileButton.tsx}`、`docs/ui/plan-detail-side-pane.md` |
| 设计语言 | `packages/ui/src/gen-ui` 注释、`docs/specs/gen-ui-styles.md`、`DESIGN.md` 引用（仅按文档内声明引用，未通读全文） |

方法：先读规格文档拿「契约」，再到实现文件核对「真的这么做」，最后 grep gaea 确认落点组件存在并读该文件。

---

## 2. ZCode UI 的前提假设（必读）

ZCode 的交互形态是三条假设的产物。办公场景三条全部不成立，这是「哪些要简化」的根。

### 假设 A：用户是程序员

**证据**
- 工具摘要行的信息构成是给读代码的人设计的：类别 + 主体 + 行数统计 + 来源标签，五段并列（`ToolSummaryRow.tsx:53-162`）。
- 允许原始数据直出：`mcp`/`fallback` 分流会把整包 toolCall JSON 摊开的历史被当成 bug 修（`PermissionDialog.tsx:300-303`、`:317-323` 的注释：「以前走 fallback 会把 raw JSON 摊开，并把工具名显示两次」），说明「默认摊原始 JSON」曾是常态。
- 名称用工具名原文：`todo.tsx:59` 会回退到 `toolCall.title`；`kindLabel` 用 `mcp__` 前缀识别（`PermissionDialog.tsx:264-275`）。
- 键盘优先：审批窗数字键 1/2/3 直接应答（`PermissionDialog.tsx:710-726`），`Enter/Space` 在摘要行上是展开（`ToolSummaryRow.tsx:199-208`）。
- 有「显示思考过程」这种开发者开关（`settingsPageHelpers.tsx:877-887`，`settings.messageStreamShowReasoning`）。

**办公含义**：办公用户不读 JSON、不认识 `bash`/`mcp__*`、不用键盘选 1/2/3，也不会去设置里关「显示思考过程」。

### 假设 B：长时间挂着看

**证据**
- 处处计时：`WorkHeader` 每轮「已完成 · 用时 · N 步」（`WorkHeader.tsx:103-126`）、工具卡 live 行已用时（`ToolCard.tsx:101-128`）、思考块 `Thought · N 秒`（`reasoning.tsx:344-360`）。
- 降噪纪律写进注释：只读工具「安静降噪」（`ToolCard.tsx:175-176` `quiet = item.readOnly && ...`），「完成后整卡摊开是对话流垂直噪音的主源」（`Transcript.tsx:481-484`）。
- 延迟升级：只有 5 秒无事件才出等待条（`StreamingIndicator.tsx:21`、`:53-55`），正常回合不叠第二条反馈。
- 完成即消失：todo 全部完成整块隐藏（`TodoPanel.tsx:64-69`），纯问答轮不挂「已完成」行（`WorkHeader.tsx:93-95`）。

**办公含义**：办公用户是「发一条、去干别的、回来看结果」。他们**需要**回来时一眼看到「跑到哪了、卡在哪」，但**不需要**每秒刷新的计时器和渐变扫光（`TodoPanel.tsx:35` 的 `animated-gradient-text`、`TodoCard` 的 GSAP 动画）。计时是给「盯着看的人」用的，办公用户需要的是「一句话状态 + 一个可点的下一步」。

### 假设 C：窗口宽大、可以横向铺

**证据**
- 侧栏 tab 默认 156px 等宽、可缩到 60px、超出后外层滚动（`SidePaneTabTrigger.tsx:43-45`、`:146` `flex-[1_1_9.75rem] min-w-15 max-w-39`）。
- 侧栏 tab 类型有 20 种（`sidePaneTabPresentation.ts:4-26` 的 `labels` 就列了 20 个标题键，`:28-81` 的 `getSidePaneTabSearchHint` 逐个分支）。
- hover 重活：GenUI 工具栏靠 `group-hover:opacity-100` 出现（`GenUiCard.tsx:128`），`snapshot.hovered` 还是快照字段（`contract.ts:14`）。
- 窄屏是事后补丁：`@max-[360px]/conversation:hidden`（`ToolSummaryRow.tsx:83`、`:95`）——即「只有在 360px 以下才丢信息」，360–1024px 之间照旧全量。

**办公含义**：办公窗口经常只有半边屏（左邮件右对话、或 1366 笔记本）。hover 在触屏/远程桌面下不可靠。

### 办公用户画像与「要简化」清单

| 维度 | ZCode 假设 | 办公用户实际 | 该简化什么 |
| --- | --- | --- | --- |
| 频率 | 挂 2 小时、几十轮 | 5–15 分钟、3–8 轮，被打断 | 去掉「每轮计时/步数」的仪式感，只留「进行中/已完成/需要你」三态；排队与审批必须「离开再回来能读懂」 |
| 目标 | 改对代码 | 拿到能用的文件 | **交付物是主角**：diff、JSON、命令是配角，默认不出现 |
| 技能 | 读得懂工具名 | 不知道 `bash` 是什么 | 工具摘要压到两段：「做了什么 + 结果如何」；`kindLabel`/`sourceLabel`/`diffstat` 收进展开 |
| 输入设备 | 键鼠、宽屏 | 触屏/小窗、单鼠标 | hover-only 一律常驻；`opacity-0` 箭头改常显；面板宽 `w-80`（`contextUsage.tsx:399`）在窄屏要能贴右 |
| 打断 | 自己控制 | IM/电话随时打断 | 审批窗**不能是 modal**（挡住正文）；排队要显示「为什么没发出去」 |
| 复习 | 会滚回去看 | 基本不回看 | 回看成本要低：文件行点击直达预览，而不是展开 diff |

---

## 3. GenUI：协议与宿主渲染

### 3.1 引用协议：三态 + 完成门槛（★ 值得抄）

**设计要点**
- Agent 在正文里输出 `::visualize{...}`（另兼容 Unicode 哨兵 `` `…` `` 与 JSON 参数三种写法），宿主侧把正文切成 `text / ui / pending / invalid` 四类片段（`references.ts:8-11` 的 `GenUiPart`，`:23-81` 的 `projectGenUiReferences`）。
- **代码区排除**：`findMarkdownCodeRanges` 命中范围内的引用不算（`references.ts:53`），所以模型解释语法时贴的例子不会变成卡。
- **完成门槛**：流式未完成时一律 `pending`，只有 `completed=true` 且 schema 通过才产出 `ui`（`references.ts:56-65`）；不合规产出 `invalid`。
- **半截引用不吞正文**：末尾可能是 JSON 前缀/哨兵前缀时把尾部截住，避免「缺失右括号后出现的普通正文一起被隐藏」（`references.ts:17-22`、`:67-80`）。
- 三态在消息层各有落点：`pending` → 「等待视图」(streaming) / 「引用不完整」，`invalid` → 「引用无效」（`GenUiMessageResponse.tsx:52-65`）。
- 卡片身份用 `${messageKey}:${offset}`（`GenUiMessageResponse.tsx:27`、`:47`），保证同一条消息里两张卡不串、重挂载不重建。

**证据**：`packages/ui/src/gen-ui/domain/references.ts:8-81`；`components/GenUiMessageResponse.tsx:40-66`。

**办公价值**：办公场景最常见的坏体验是「模型说了『下面是为您生成的成本对比图』，然后什么都没有」。三态把「还没有/坏了/在这」显式化。
例：让 agent 为 `成本测算.xlsx` 生成一张对比图，流式过程中正文出现「正在生成视图…」，落盘后原地变成图；如果模型写错了路径，用户看到的是「引用无效」而不是一条静默消失的正文。

**gaea 落点**：`frontend/src/genui/parse.ts`（围栏解析）、`extractFence.test.ts`、`frontend/src/gaea/components/GenuiPanel.tsx`（已被 `genui.walkthrough.test.tsx:28-47` 覆盖「围栏→渲染→action」链路）。三态需要新增在 `parse.ts` 的返回类型与 `MemoMarkdown` 的围栏渲染分支。

**成本**：S（解析层加一个 `status` 字段 + 两个占位块；不动 blocks）。

**风险**：
- gaea GenUI 是 ```` ```genui ```` 围栏（JSON spec），ZCode 是正文内联指令。若照搬「正文内联」会与现有围栏语法并存成两套；建议只借**三态语义**，语法仍用围栏。
- 现有 `interaction.test.ts` / `guard.test.ts` 对围栏非法时的降级有断言，新增 `invalid` 占位要同步改测试契约。

### 3.2 流式增量渲染与尺寸约束（★ 值得抄）

**设计要点**
- **测高所有权在 guest**：内容尺寸/根子节点/视口宽度变化 → guest 合并到下一动画帧读几何 → 走 `ui/notifications/size-changed` → 宿主更新内联高度（`gen-ui-inline-sizing.md:13-18`）。
- **只读几何、不写高度**：旧实现会把 `html` 高度临时设成 `max-content`，测量本身引起布局变化；新测量「只读内容几何」（`gen-ui-inline-sizing.md:34-35`、回归表 `:53`）。
- **断反馈环**：相同宽度下「内容增量完全等于视口高度增量」时报（`gen-ui-inline-sizing.md:25-26`），否则 `vh`/百分比高度会随宿主调整无限增高。
- **隐藏根滚动条**：只作用于 `html`/`body`，不动作者组件内滚动区（`gen-ui-inline-sizing.md:5-6`），且样式由沙箱装配注入，作者片段不用改（`:27-28`）。
- **硬上限 10000px**：宿主侧 `Math.max(0, Math.min(10000, Math.ceil(height)))`（`adapters/page.ts:128-135`）；超出后「保持原有内容访问能力，不新增提示或展示模式」（`gen-ui-inline-sizing.md:7-8`）。
- **初始占位高度**：`EMPTY_GEN_UI_SNAPSHOT.height = 240`（`contract.ts:18-27`），避免首帧 0 高度抖动。

**证据**：`docs/specs/gen-ui-inline-sizing.md:5-35`、`adapters/page.ts:128-135`、`contract.ts:7-27`。

**办公价值**：办公 GenUI 经常是「一张表 / 一张图 / 一个看板」，高度变化剧烈。若由宿主猜高度，会出现「图被截一半」「卡片里冒出第二条滚动条」「填表时页面往下跳」。
例：把 `季度费用.xlsx` 汇总成看板，用户切「按部门/按月份」tab 时高度从 180px 变 620px；正确行为是内容撑开、外层对话滚动条接管，鼠标滚轮滚整个对话而不是卡在看板里。

**gaea 落点**：`frontend/src/genui/GenuiBlock.tsx`（React 渲染，天然自测高，**不需要** iframe 协议）、`renderNode.tsx`（36KB，节点渲染）、`limits.ts`（节点/嵌套上限，见 `genui` 技能）。
→ 结论：**ZCode 的测高协议对 gaea 不适用**（gaea 不是 iframe），但**10000px 上限思路可迁移为 `maxHeight + 内部滚动` 的单块上限**，以及「切 tab 不抖」的 `layout` 稳定性。

**成本**：S（gaea 侧只需给超大块加 `max-height + overflow` 与 `keepMounted`）。

**风险**：`GenuiBlock.test.tsx`（6KB）已断言渲染结构；加高度容器会改变 DOM 层级，可能打破 `querySelector` 选择器。

### 3.3 卡片宿主的边界态（值得抄「状态清单」，不值得抄「控件密度」）

**设计要点**
- 一个卡片有 6 种可见态：`!supported`（桌面独占）→「仅桌面可用」；`!online` →「离线」；`phase==="error" | acquisitionError` →「加载失败」+ 错误文案 + **重试按钮**；其余 →「正在加载…」+ spinner（`GenUiCard.tsx:112-120`、`:195-210`）。
- 只有 `phase === "ready"` 才出工具栏（`GenUiCard.tsx:212`）。
- 工具栏三件：复制为图片（失败走 `role="alert"` 行内错误，不用 toast）、Tweak 面板（仅当 `snapshot.groups.length > 0`）、展开（`GenUiCard.tsx:141-186`）。
- 展开态：`85vh × min(1100px, 94vw)`，`modal={false}` + 自绘 backdrop（点击关闭），关闭后焦点回展开按钮（`GenUiCard.tsx:220-247`）。
- `escapeSequence` 自增即收起（`GenUiCard.tsx:109-111`，`contract.ts:16`）——genui 页面里按 Esc 能让卡片退出展开，不用宿主全局监听。
- 页面可反向请求宿主弹「输入确认」（`confirm`），宿主用 Dialog + Textarea 提交（`GenUiCard.tsx:83-96`、`:248-273`）。
- 生命周期挂 task 删除：`watchTaskRemoved` 订阅 `task_deleted` 后 release（`useGenUi.ts:44-57`）。
- 重建成本控制：`acquireGenUiPage` 以 `JSON.stringify(target)` 为 key，target 不变不重新获取（`useGenUi.ts:30-31`、`:73`）。

**证据**：`components/GenUiCard.tsx`、`hooks/useGenUi.ts`、`gen-ui/contract.ts`。

**办公价值**：办公用户看到空白卡时的第一反应是「坏了」。显式状态 + 一个重试按钮，把「模型/宿主故障」变成「一个可点的动作」。
例：`预算汇总看板` 的 html 源文件被清理掉了 → 卡片显示「加载失败 + 重试」，用户点重试或直接在聊天里说「重出一次」，而不是盯着空白。

**gaea 落点**：`frontend/src/genui/GenuiBlock.tsx`、`frontend/src/gaea/components/MemoMarkdown.tsx`（围栏宿主）、`frontend/src/gaea/components/GenuiPanel.tsx`（已有「清空」二次确认、空态引导文案，`:45-53`）。

**成本**：M（要新增「genui 资源获取失败」这一层——gaea 的 spec 是内联的，没有「获取资源」这一步，所以只剩「spec 非法/超限」两种失败 + 重试入口）。

**风险**：gaea 的 `guard.ts`（25KB）与 `limits.ts` 已有降级策略，新增状态层容易与既有降级文案重复。

### 3.4 样式投影契约（抄「token 纪律」，不抄「HTML 控件体系」）

**设计要点**（`docs/specs/gen-ui-styles.md:12-31`）
- 字号阶梯：h1/h2/h3–h6 = 正文 +4/+2/+0，辅助文字 −2，tooltip −1，行内代码 −2。
- 圆角是**独立 token** 而非比例推导：sm/md/lg/xl = 4/6/8/12px；card 从 xl 起，嵌套逐级降（xl→lg→md→sm）；控件按所在容器选级（xl 或无圆角父容器用 lg，lg 里用 md，md/sm 里用 sm）。
- 按钮默认 outline；最小高 28px，xs/sm/lg = 20/24/32px；`disabled opacity 0.5`；**不许用固定高度裁掉放大后的文字**。
- input/select 高 36px，紧凑 28/24；textarea 最小 64px；checkbox 16px 且支持 indeterminate；switch 32×18、thumb 16。
- 焦点必须在 hover 之外可辨：独立导出缺 `focused` token 时用 ring 边框（`:28`）。
- 单一来源：`App styles.css → Gen UI 专用变量映射 → HostContext → guest 根 CSS 变量 → visualize.css`（`:36-41`），只用 MCP 标准变量 + `zcode/genUiStyles` 扩展，不往 SDK 标准变量对象塞自定义键（`:43-45`）。

**办公价值**：把 GenUI 和宿主拉到同一套半径/字号上，卡片才不会看起来像「网页嵌进文档」。
例：`合同台账.docx` 旁边插一张 GenUI 表格，表格圆角 12、按钮高 28，与正文卡片一致；若各写一套，视觉上会「抖」。

**gaea 落点**：`frontend/src/genui/styles.css`（17KB，已经是令牌化样式表）。

**成本**：S（只做一次 token 对齐审计）。

**风险**：玄墨 v11 纪律是「只消费 `--md-sys-*` / `--gaea-*` / `--color-*` / `--v3-*` 与旧栈令牌，零硬编码色值」（`frontend/src/gaea/components/memoryhub/hub.css:12-14`）。GenUI `styles.css` 若存在字面色值，需先清账再加新色，否则违反 v11。**这条是「审计项」不是「新功能项」**。

### 3.5 外部链接（抄「默认安全 + 不做下载」）

**设计要点**（`docs/specs/gen-ui-external-links.md:24-30`）
- 只放行绝对 `http(s)`；相对、`file:`、`javascript:`、`data:` 一律不通过宿主流转。
- 必须真实用户手势：合成点击、加载期定时调用被拒（`:26`）；宿主消费手势后**再校验一次**页面实例与可见性（`:26`、时序图 `:8-19`）。
- 已被页面 `preventDefault()` 的点击不重复处理；空 href 与 `#片段` 保留页内行为（`:25`）。
- `download` 链接**不**转成外部打开，「本次不提供下载能力」（`:25`）——这是刻意的边界。
- 打开动作发生在**查看页面的客户端**（`IPlatformService.openExternal`），远程工作区不在远端执行（`:29`）。

**办公价值**：GenUI 里放「查看源表」链接是高频需求；但办公场景更怕「点一下卡里就把浏览器打开了 5 个窗口」。手势校验 + 只 http(s) 是正确的收紧。
例：`报价单_v3.docx` 里嵌的 GenUI 卡片带「打开客户官网」链接，点击一次只开一次主流浏览器；卡片里的内嵌 iframe 加载不会自己弹窗。

**gaea 落点**：`frontend/src/genui/guard.ts`（URL/协议白名单所在层，已有 guard 测试 `guard.test.ts`）、`frontend/src/gaea/lib/bridge`（`openExternal`）。

**成本**：S。

**风险**：gaea 走 Electron bridge，`openExternal` 已是统一入口；风险在于 GenUI 里的 `<a href>` 是否会绕过它——需实测 `renderNode.tsx` 对 `a` 标签的处理。

### 3.6 固定输出目录（抄「契约思路」，不抄「目录实现」）

**设计要点**（`docs/specs/gen-ui-output-directory.md:5-19`）
- 生成文件落在「执行端 Host 提供的固定输出根目录」（默认应用数据目录下 `visualizations/`），**Agent 不回退到当前工作目录**（`:5-7`）。
- 子目录由 `workspaceIdentity || workspacePath` + `sessionId` 派生（`:6-7`）。
- **关键澄清**：「固定目录」是**输出位置约定，不是工作区包含关系限制**（`:9-11`）；原实现把「不在当前目录生成」误解成「必须位于工作区之外」，导致 Home 工作区初始化直接抛错（`:29-31`）。
- 会话子目录**不是读取授权边界**：fork 继承父会话 HTML 路径后可直接加载，不用复制文件或改写历史（`:13-15`）。
- 仍然拒绝：根目录之外、符号链接、非普通文件、非法 HTML 片段、超体积；根目录本身可解析符号链接，解析后仍执行同一检查（`:17-19`）。
- 远端读取必须走执行端 `gen-ui` RPC，**失败原样返回、不回落到本机读取**（`:37-40`）。

**办公价值**：办公用户最容易踩「产物在哪」。固定根目录 + 会话子目录，使「打开产物目录」成为一个稳定动作；也避免 agent 把看板写进用户的项目目录弄脏工作区。
例：agent 为 `供应商比价.xlsx` 生成三张对比图，存放在 `…/visualizations/<ws>/<session>/` 下；交付卡上「在工作区显示」不适用（不在工作区），但「打开所在目录」可用。

**gaea 落点**：gaea 的 GenUI 无落盘步骤（spec 内联在消息里），此条**不迁移**。可迁移的只有一句话：**「产出物的存放位置要有唯一约定，并且这个约定不参与读取授权」**——落到 gaea 即交付物登记表（`frontend/src/gaea/lib/deliverablesTurn`、`useTurnDeliverables`）的「登记口径 vs 存在性判定」分离。

**成本**：N/A（不迁移）。

**风险**：若强行引入「固定输出目录」，会与 gaea 现有「写在工作区、打开走 `openPaneFileOrPreview`」的模型冲突（`DeliverableCards.tsx:164`），并让「在工作区显示」失效。

### 3.7 源文件不进变更摘要（★ 可抄，且能直接变成办公价值）

**设计要点**（`docs/specs/gen-ui-file-summary.md:5-13`）
- 被 visualize 引用的 HTML 是「内联可视化的源文件」，**轮尾不再为同一文件显示 HTML 网站卡片，也不计入文件变更摘要的文件数/增删行/展开列表**（`:5-7`）。
- 过滤后**没有普通文件时，整块变更摘要（含撤销按钮）不显示**；混合修改时保留普通文件（`:7-8`）。
- 识别复用 GenUI 引用解析器，只匹配「当前轮正文中完整且通过 schema 校验的绝对路径」；代码示例、无效/未闭合引用不生效（`:9-10`）。
- 路径按**完整路径**匹配，兼容 Windows 分隔符；**不按 basename、扩展名或名为 visualizations 的目录批量隐藏**（`:11-12`）。
- 折叠态也读 fileChanges 详情，**读取完成前暂不显示摘要**，避免「先显示含源文件的统计再消失」（`:24-26`）。
- 读取失败保留原摘要 + 重试入口，避免误隐藏普通文件（`:27`）。

**证据**：上述 spec；`docs/ui/task-file-change-summary.md:97-99` 的摘要口径。

**办公价值**：这是「交付物可信度」问题。办公用户看到「本轮 4 个文件变更」时会数：只想要那 1 个 docx，另外 3 个是渲染中间产物——于是不信这个计数器。
例：让 agent 生成「Q3 经营分析.docx」+ 一张内嵌图，轮尾摘要应显示「1 个文件 +86 −0」，点开只有 docx；中间生成的 html 不进清单。

**gaea 落点**：`frontend/src/gaea/components/DeliverableCards.tsx`（两路合并：正文启发式 + 权威登记表，`:57-95`）、`frontend/src/gaea/lib/deliverablesTurn.ts`（`mergeDeliverableCards`）、`frontend/src/gaea/lib/fileLinks.ts`（`deliverableMentions`）。
→ 具体做法：给 `deliverableMentions` 增加「排除被同轮 genui 围栏引用的路径」；并在合并后为空时整块不渲染（现在 `cards.length === 0` 已经 `return null`，`:95`，只需保证过滤发生在 `useTurnDeliverables` 之后）。

**成本**：S。

**风险**：`DeliverableCards.test.tsx` 断言了「正文卡在前、登记卡在后」「计数徽标文案」（`:101-110`）；加过滤会改变计数，需同步更新这些断言。

---

## 4. 对话流

### 4.1 工具调用呈现

**设计要点**
- **摘要行三模**（`ToolSummaryRow.tsx:164-234`）：
  - 有 `action` → `role="button"` 直接执行（如「打开会话」），不展开；
  - 可 `canToggle` → `CollapsibleTrigger`，箭头 `opacity-0 → 100` 只在 hover/展开/`forceOpen` 时可见（`:213-220`）；
  - 静态 → 不可交互行（`:226-234`）。
- 摘要行由 5 段组成：`icon | kindLabel | kindDetail | sourceLabel | primaryText/secondaryText/diffCount/statusNode`（`:53-162`），`prioritizePrimaryText` 时窄屏（<360px）优先丢 `kindLabel`/`sourceLabel`（`:83`、`:95`）。
- 摘要内容有「翻滚」动画：`QueuedSummaryContent` + `contentKey`/`contentRefreshVersion`，展开时停（`ToolSummaryRow.tsx:150-158`）。
- **行内细进度条**：有 `fraction` 用 determinate，否则 indeterminate 扫光；后跟截断的 `progress.message`；颜色只用 brand/surface token（`ToolCallProgressIndicator.tsx:17-46`）。
- **失败态**：`statusLabel` + `statusTooltip`（错误文本）+ `showFailureStatus`（`todo.tsx:146-148`），即失败不只变色，还给文字。
- 分组：连续同名工具折成一行（gaea 侧同名实现见 `ToolGroup.tsx:86-115`；ZCode 侧见 `ProcessCard` 内的 `scanGroups`）。

**证据**：`ToolSummaryRow.tsx`、`ToolCallProgressIndicator.tsx`、`docs/ui/tool-display-rendering.md:24-29`（可见性过滤只作用渲染层，协议投影保留原始行）。

**办公价值**：办公场景工具调用的**种类极少**（读文件、写 docx、跑 python 算 xlsx、联网查一次），高频的是「同一个工具反复调用」。
例：agent 为 `12 个月报销明细.xlsx` 逐个 sheet 跑脚本 → 应折成「python × 12」一行，展开看每次的 sheet 名，而不是 12 张卡。

**gaea 落点**：`ToolCard.tsx`（已有 `TaskLiveRow` 实时活动行 + `diffStat` 芯片 + 可恢复错误降级，`:175-176`、`:254-266`、`:338-347`）、`ToolGroup.tsx`、`Transcript.tsx` 的 `ProcessCard`（`:465-628`）。

**成本**：S–M。真正缺的只有两件：**行内细进度条**（`ToolCard` 现在只有 spinner，无 fraction）与 **失败的文字标签**（现在失败只靠图标色 + 展开区文本，`:45-50`）。

**风险**：
- `ToolSummaryRow` 的五段结构直接搬会与 `ToolCard` 的单行结构打架（gaea 是 `chevron | icon | name | subject | diffstat/summary | status`）。
- `ToolCard.test` / `ProcessCard.test` / `ToolGroup.test` 均断言 DOM 文本（如 `toolGroup.called`），加段落会打破。
- 玄墨纪律：进度条用 `bg-brand`（ZCode）vs gaea 的 `--gaea-glow`（`TodoCard.tsx:109`），需统一。

### 4.2 思考过程

**设计要点**（`docs/chat/reasoning-streaming-auto-collapse.md:7-23`）
- **流式默认收起**，触发器显示 `Thinking · <当前行>`：按换行边界派生摘要快照，同一行 token 原位更新，出现新行时旧行上滚、新行下方进（`:10-11`）。
- 摘要宽度随内容增长，撑满后单行 + 每次更新滚到末尾（`:12`）；横向溢出时左右各 16px mask，未溢出**不挂** mask（`:13`）。
- 运行中展开 → 头部变 `icon + Thought + · + duration`，每秒更新（`:15`）；完成态 `Thought · 持续了 N 秒`（`:16-17`）。
- 展开内容默认吸底；用户上滚离开底部则暂停吸底，回底恢复（`:22`）；只有实际产生纵向滚动才显示边缘 mask（`:21`）。
- 实现上有 `shouldAutoCollapseReasoning`、`getReasoningBottomDistance`、`resolveReasoningStreamingSummary` 等纯函数（`components/ai-elements/reasoning.tsx:66-90`、`:255-271`）。

**办公价值**：办公用户会盯着「Thinking」看 20 秒然后以为卡了。单行流式摘要把「模型在读哪个文件」直接显示出来，比 spinner 有用得多。
例：让 agent 核对 `采购合同_终版.docx` 与 `附则.docx` 的条款冲突，收起态显示「Thinking · 正在比对第 7 条与附则第 2 款」，用户就知道它在干活而不是在发呆。

**gaea 落点**：`Transcript.tsx` 的 `InlineReasoning`（`:401-433`）——已有「默认折叠 + 运行中小圆点 + `reasoning.label` / `thinkingRunning`」，**但没有单行流式摘要**，运行中只显示「思考中」。

**成本**：M（要新增「最后一行摘要 + mask + 翻滚」的纯函数与 DOM；`displayReasoningText` 已有截断基础）。

**风险**：
- 翻滚动画与 `StreamingIndicator`/`WorkHeader` 的动画节奏叠加会「抖」；`reasoning.tsx:373-374` 明确说「thinking 会在长流式回复里持续存在，旋转 loader 会长期占用渲染资源」。
- gaea 现有 `MergeConsecutiveReasoning`（`Transcript.tsx:47`）会合并连续纯推理消息，摘要取哪一条要定义。

### 4.3 todo / 任务清单

**设计要点**
- **全完成即隐藏**：`completedCount === plan.length` 直接 `return null`，理由写在注释里——provider 可能先全标 completed 再推 `task_complete`，中间间隙继续显示会让用户以为还有事（`TodoPanel.tsx:64-69`）。
- **只显示未完成项**：`visiblePlan = plan.filter(status !== "completed")`，理由是完成项加删除线只会让面板越来越长（`:71-73`）。
- **折叠态 = 当前任务**：`activeStep` 取 `in_progress` → 首个未完成 → 最后一项（`:74-77`）；折叠时只显示 `activeStep.title + completed/total`（`:127-140`）。
- **加载视觉只在真运行期**：`showLoadingState = isStreaming && !isAllCompleted`；非运行期即使残留 `in_progress` 也降级为普通文本，避免「一直转圈」（`:78-83`、`:41-45` 的 idle 类表）。
- 状态色 + 图标 + 文字三重（`:27-45`；同款纪律见 `TaskCenter.tsx:32-33`）。
- 工具卡内的 todo 摘要：`activeStep.title` + `completed/total`，展开才是列表（`renderers/todo.tsx:48-97`）。

**办公价值**：todo 在办公场景是「这次要做哪几步」。它应该是**常驻的一张小便签**，不是一条会被忽略的卡。
例：让 agent 把 5 份 Excel 合成一张总表，todo 显示「合并表头（2/5）」，完成后整块消失，用户不会误以为还剩活。

**gaea 落点**：`TodoCard.tsx`（`:31-45` 进度/摘要/全部完成判定、`:50-95` 单行头部、`:98-147` 展开体）。**gaea 已明显领先**：有进度条、未完成在前/已完成收尾分组、`level` 子任务缩进、展开自动滚到当前项（`:40-44`）。

**成本**：S。gaea 唯一可借的是「**全部完成即隐藏**」这一条（现在 `TodoCard` 显示「全部完成」文案 + 关闭按钮，`:80-90`，需要用户手动关）。

**风险**：`TodoCard.test.tsx` 断言了「全部完成」文案与关闭按钮；改成自动隐藏要改测试，并且要考虑「用户正在看的时候突然消失」——建议加 3–5 秒延迟或只在下一轮开始前隐藏。

### 4.4 计划（plan mode）呈现（★ 值得抄）

**设计要点**（`docs/ui/plan-detail-side-pane.md`）
- **时间线只放紧凑卡，全文进侧栏**：卡片显示「计划」+ 复制 + 裁切预览，底部悬浮胶囊「查看完整计划 →」；「主时间线不再原地展开，也不保留底部『展开计划』按钮」（`:5`、`:13-15`）。
- 裁切预览：`max-h-64` + 溢出裁切 + 底部 mask，**前 30% 完全可见、后 70% 渐隐**，渐隐按可见高度计算（`:103-106`）。
- **稳定 tab 身份**：每个 `(workspaceKey, parentSessionId, toolCallId)` 一个 tab，同一计划复用，不同计划并存（`:16`、`:83-85`）；切换父任务按 `parentSessionId` 隐藏/恢复（`:73-75`）。
- **位置不搬家**：计划卡严格停在 `ExitPlanMode` 的 tool row 位置，**不提升到轮尾**；若在正文之前，随「已工作」历史一起折叠（`:19`、`:117`）。
- 流式从「首个可解析 `plan` 字段」开始显示，**不展示行数或流式计数动画**（`:20`、`:112`）。
- 轮尾「复制」按轮聚合：先全部 assistant 正文段，再追加同轮所有计划 Markdown，**不读卡片裁切 DOM**（`:22`、`:116`）。
- 右上角摘要新增「计划」分区：只列已进入 `success/error/cancelled` 终态且有正文的计划，`rowId` 从新到旧（`:23-24`）。
- `EnterPlanMode` 是内部边界，**不向用户渲染 tool row**（`docs/ui/tool-display-rendering.md:26`）；但它的 guidance 内容用 Markdown 块展示（`renderers/plan-guidance.tsx:82-97`）。
- 无障碍：整卡可聚焦、`Enter/Space` 打开详情，但**忽略来自内部交互元素**的 Enter/Space，避免底部按钮与整卡重复开同一个 tab（`:96-100`）。

**办公价值**：办公任务（写 30 页报告、整理 200 行台账）的「计划」是**用户最想审的东西**——方向错了白干。时间线里占 40 行会淹掉正文；塞进 tab 又怕找不到。ZCode 的解法是「时间线给裁切预览 + 一个明确的『查看完整计划』胶囊，tab 常驻」，兼顾两者。
例：`Q4 经营分析报告.docx` 的计划有 9 个章节，时间线里只显示前 8 行 + 渐隐 + 「查看完整计划」；用户点开右栏边看计划边在输入框里写「第 4 章换成同比」，不用滚回计划。

**gaea 落点**：**目前为空**。
- plan mode 只有一个开关：`Composer.tsx:55`、`:541` 的 `planActive` / `onTogglePlan`，接 `/plan on|off`（`App.tsx:885-886`）；`ComposerToolbar.tsx:209-217` 是按钮的 `aria-pressed`。
- 审批走通用 `ApprovalModal.tsx`（5 钮 + 数字键 + Esc=拒绝，`:40-46`、`:75-92`），没有「计划」语义。
- 侧栏 tab 已有承载：`ChatTabs.tsx`（对话/轨迹/上下文/记忆 + 动态会话 tab）。

→ 落点建议：新增「计划」为 `ChatTabs` 的一个 tab（或 `ChatTabs` 的 `extraTabs`），数据源接 plan 文本；时间线里加一张裁切卡。

**成本**：M–L。M 若只做「计划卡 + 复制 + 打开右栏 tab（复用现有 tab 壳）」；L 若要完整复刻稳定 tab 身份 / 目录 / 流式首解析。

**风险**：
- gaea 的 tab 是**视图 tab**（单例），不是「每个计划一个 tab」。要复刻 ZCode 的稳定 tab 身份，需要给 `ChatTabs` 加「多实例 tab」能力——与现有 `extraTabs`（子代理会话）语义接近，可复用但会扩大 tab 状态面。
- `ApprovalModal` 是 modal（`role="dialog" aria-modal="true"`），计划审批若沿用会很重；建议先改 §4.5 的 dock，再谈计划审批。
- 测试：`ChatTabs.test.tsx` 断言了四个 tab 与键盘导航（`ChatTabs.tsx:47-62`），新增 tab 要同步。

### 4.5 审批弹窗（permission dialog）（★ 最值得抄的一条）

**设计要点**
- **内联 dock 而非 modal**：根节点 `w-full shrink-0 relative z-1`，本体 `rounded-2xl border border-border bg-popover`（`PermissionDialog.tsx:825-835`）——它躺在对话流里，用户能向上滚看上下文，不会被遮罩挡住。
- **限高但只有一层滚动**：保存类确认窗 `max-h-[min(72dvh,42rem)]`，且注释明确记录了教训——「上一版问句钉顶、中段滚动、段内三块各自滚动、窗口再兜底滚动，最多五层滚动容器叠在一起，一次滚轮手势的去向取决于指针恰好在哪个框上」（`:816-823`）。
- **选项列表 = listbox**：每行 `1.` 编号 + 标签 + 描述（`role="option"`、`aria-selected`、`tabIndex` roving，`:899-1004`）。
- **数字键直接应答**：`1/2/3` 立即执行该选项（`:715-726`）；`Esc` 在反馈框里是 blur 而非关闭（`:700-703`）。
- **第 N 行永远是「拒绝并附意见」**：`feedbackIndex = orderedOptions.length`（`:584`），草稿按 `requestId` 归零避免串味（`:588-592`）；提交时 `trim` 后作为拒绝理由（`:661-668`）。
- **`allowAlways` 直接展示将放行的命令前缀**：从 `permissionUpdates[].rules` 里取 `bash` 且以 `:*` 结尾的 `ruleContent`，去尾后单行省略到 160 字符（`:96-121`、`:141-161`）——用户看到的是「将要放行什么」，不是一个抽象的「始终允许」。
- **审批窗内禁止折叠**：`canToggle: false` + `forceOpen: true`，注释写明「避免用户把关键内容藏起来」；只有 workflow 脚本折叠是 spec 记录的例外（`:331-354`）。
- **申请理由必须可见**：`displayReason` 缺省时不留空，且有「非用户面理由」过滤表（`NON_USER_FACING_PERMISSION_REASONS`），避免把协议诊断文案（`"High risk tools require explicit approval"`）当成人话展示（`:91-94`、`:243-262`）。
- **来源徽标**：子代理发起的请求在标题旁加 `InteractionRequestOriginBadge`，文案点名哪个子代理，超长截断进 tooltip（`PermissionDialog.tsx:850-855`、`InteractionRequestOriginBadge.tsx:12-46`）。
- **权限预览用真实 renderer**：按 blockKind 分流到 `edit/execute/mcp/web-fetch/search/skill/workflow/fallback` 的真实工具块（`:795-808`），并显式传 `isRunning: false` + `isPermissionPreview: true`——注释记录了 bug：「Bash 权限会显示『执行中』并带 loading，遮住真正的申请原因」（`:405-409`）。
- **失败应答就地显示**：`responseError` 用 `role="alert"` 放在按钮上方（`:1073-1077`），不是 toast。
- 手机兼容：iOS 输入聚焦缩放用 16px token（`:500-505`、`:1066`）。

**办公价值**（这是办公场景最高频的摩擦点）
1. 办公用户审批时**最想向上滚看上一轮的数字/表格**。modal 直接剥夺这个动作，于是只能「先拒绝，回去看，再发一次」。
2. 「始终允许」在办公场景是危险的：用户不知道自己在放行什么。ZCode 把 `bash` 前缀摊在按钮右边，等价到办公就是——「始终允许写入 `报价单_*.docx`」应该显示这个通配符。
3. 「拒绝并说明」比「拒绝」信息量高一个量级：模型可以直接按意见改，省一轮往返。
4. 子代理来源徽标：办公里常见「总控 agent 派了 3 个子 agent 各改一个文件」，审批时不知道谁在问就会懵。

例：agent 要在 `\\share\报价\2026Q3\` 下覆盖 `报价单_v3.docx` → 审批卡就在正文流里，标题「需要你确认」，理由行是模型写的「按新成本表重算单价并覆盖 v3」，选项 2 右边灰字显示「本会话允许写入 `报价单_*.docx`」，用户上滚能看到上一轮的单价表再决定。

**gaea 落点**：`frontend/src/gaea/components/ApprovalModal.tsx`（`:48-171`）。
**已有**：5 钮编号 + 数字键 + Esc=拒绝（`:40-46`、`:75-92`）、`hardAsk` 白名单隐藏「本会话/始终」（`:62-71`）、`schedule_apply` 专用 diff 卡体（`:135-145`）、理由缺失显式标注（`:129-131`）。
**缺**：①是 modal 而非 dock（`:95-104` `role="dialog" aria-modal="true"`，外层 `.plan-approval-dock` 只是定位容器）；②没有「拒绝并附意见」输入行；③「始终允许」没有把将放行的范围摊开；④没有来源徽标。

**成本**：M（改 `ApprovalModal` 为 dock：去掉 `aria-modal`、改 `role="group"` 或 `role="region"`、加反馈行、加 scope 摊开、加 origin badge；diff 卡体已有）。

**风险**：
- `ApprovalModal.test.tsx`（3.2KB）大概率断言 `role="dialog"`/`aria-modal`；改 dock 会破。
- 键盘 1–5 与 `Esc` 是全局 `document` 监听（`:90`），改为内联后可能与 Composer 的快捷键冲突（`Composer.tsx:391` 的 Alt+Enter 等）。
- 视觉：`bg-popover`/`bg-bg-elev`（`:98`）+ 玄墨「描边退场、近实底」（`hub.css:5-11`）——dock 若用粗描边 + 深色底会显得像「页面里的另一个页面」。建议 dock 用 `bg-surface` + 左侧 3px 语义色条（与 `ErrorCard.tsx:18` 的 `border-l-[3px]` 同语言）。

### 4.6 排队输入（queue / busy）

**设计要点**
- **权威归属**：queue 的唯一事实源是 CLI runtime，UI 只消费 `ProductProjection.queue`；「renderer 不维护 accepted queue，也不存在桌面 renderer queue 与手机 host queue 两套产品顺序」（`docs/ui/chat-message-queue.md:3`）。
- **操作表**（同文档 `:19-27`）：删除 / 撤回编辑 / 立即发送 / 排序 / 自动续跑，各自对应一条 V4 命令。
- **撤回编辑是 ACK 后才恢复草稿**：先 `deleteQueueItem(baseRevision)`，只有 accepted/duplicate 才在**发起端**恢复 `kind + text + AttachmentRef[]`；stale/noop/rejected/failed 一律不恢复并提示重试；composer 已有草稿时不发命令（`:31-51`）。
- **暂停横幅**：`autoDrain=false` 时列表顶部一条横幅，按 `pauseReason` 分 `stopped|error|generic` 三种文案 + 「继续」按钮（`ConversationQueuePanel.tsx:368-398`）。
- **立即发送**：每行一个 `sendNow`（`ArrowUpFromLine`），语义是「stop 当前 + 消费该项」（`:244-269`）。
- **行锁按派发状态**：`dispatchLocked = item.dispatch.state !== "queued"`，锁定后整行禁用（`:137-139`）。
- **错误终态不丢队列**：普通 turn 进入 `error` 只结束当前 turn，queue 原样保留并切 `autoDrain=false`，暂停原因标 error，用户点「继续」恢复队首 FIFO；错误路径禁止发 `TurnSteerDiscarded`、禁止自动 drain（`docs/conversation-protocol-declaration.md:312`）。
- **双语义分轨：queue vs guide**（`docs/chat/interaction-behavior.md`）：
  - `queue` = 未来 turn（默认）；`guide` = 在下一次合法 model-step 边界注入当前 turn（`:5-13`）。
  - guide 待生效时**立即在当前对话流末尾显示用户消息**，气泡下方「等待引导当前任务…」；收到 `TurnSteerDrained` 后由正式 `guided=true` 的用户行接管；若被降级为 queue，临时行消失、同一权威项回到队列面板（`:62-79`）。
  - `Enter` 与 `Cmd/Ctrl+Enter` **按当前偏好取反**：queue 模式下 `Cmd/Ctrl+Enter` = 立即发送（原子 `startAt`，不创建 QueueItem）；guide 模式下 = 加入队列（`:81-95`）。按住修饰键时发送按钮 Tooltip **主动弹出并预告相反动作**（`:97-101`）。
  - guide 的消费纪律极严：必须等当前 model step 的**全部 sibling tool results** 提交后才最多消费一条（`:54-60`）。

**办公价值**：办公用户的典型行为是「发了一条，突然想起要补一句」。
- 现在 gaea 的做法（运行中 Enter=排队）是对的，但**缺「为什么没发出去」的解释**：用户看到一条躺在框里的排队项，不知道是在等什么。
- ZCode 的暂停横幅 + `autoDrain` 语义正好补这个：出错停下后队列不会自己跑，用户明确点「继续」。
- 「拒绝后不丢队列」是办公最在意的一条：一次 429/超时不该把排好的 3 条全废掉。

例：用户排队「再补一列同比」「图表换成柱状」「导出 pdf」，第二句执行时报 API 429 失败 → 队列保留、横幅显示「上一轮出错，队列已暂停」+「继续」；用户点继续，第 2、3 句照原顺序跑。

**gaea 落点**：`Composer.tsx`（`queueRef`/`syncQueue`/`enqueueSubmit`/`cancelQueueItem`/`editQueueItem`/`steerQueueItem`/`reorderQueue`，`:107-320`）、`composer/ComposerQueueList.tsx`（`:20-164`，含发送中钉住、逐条/全部取消、逐条/全部 Steer、拖拽排序、`isQueueBusy` 整列锁定）、`composer/composerQueue.ts`。
**gaea 已有且不弱于 ZCode**：编辑撤回（`:114-120` 点击回填）、Steer（Alt+Enter 直插，`Composer.tsx:391`）、拖拽排序、逐条/全部取消。
**缺**：①`autoDrain`/暂停横幅（gaea 的 queue 是 renderer-local `useState`，`:109`，刷新即丢）；②「立即发送」逐条按钮（只有全部 Steer）。

**成本**：S–M。S 若只做「暂停横幅 + 逐条立即发送」（纯前端，在 `ComposerQueueList` 加一个 `error` 状态横幅即可）；M 若要 queue 持久化/权威化（需后端契约，跨层）。

**风险**：
- gaea 的 queue 是本地态，**照抄「ACK 后才恢复草稿」会引入一个不存在的失败轴**（本地撤没有 ACK）。
- `Composer.queue.test.tsx`（6.5KB）+ `ComposerQueueList.test.tsx`（6.6KB）已覆盖排队行为，加横幅与按钮要同步；`composerQueue.test.ts` 断言 `isQueueBusy` 语义。
- ZCode 的 `autoDrain` guard 与 goal/compact 耦合（`interaction-behavior.md:23-38`），gaea 没有 goal 状态机，**不要**照搬这套 guard 链。

### 4.7 上下文占用条（★ 最值得抄，且 gaea 完全空白）

**设计要点**（`chat-input-toolbar/contextUsage.tsx`）
- 入口在**输入工具栏**，不是独立页：`Context` + `ContextTrigger`，`ContextContent side="top"`、`!w-80`（`:995-998`、`:399`）。
- 触发器本身只有一个百分比/用量标签（`chat.contextUsage` 的 `{used}/{total}`，`:858-870`），**不在触发器上堆 breakdown**。
- 面板结构（`:1003-1075`）：
  1. 标题行 + 右对齐等宽 `used/total (pct)`（`:1004-1012`）；
  2. **分段进度条**：`<Progress segments={progressSegments} value={usagePercent*100}>`（`:1013-1019`）；
  3. **分项列表**：色块 + 名称 + 右对齐百分比（`:1031-1052`）。
- 分项只有 7 类且顺序固定：`messages / system_prompt / meta_user_context / skills / tool_prompt / system_tool_schemas / mcp_tool_schemas`（`:150-168`）；按 chars 降序排（`:192-204`）。
- **分项只给百分比，不给 token 数**——注释写明原因：「breakdown 行只展示占比，分项 token 数会和顶部总量口径混在一起造成误读」（`:1047`）。
- 5 档色阶由 `--color-usage-chart-1` 与 `--color-surface` **混合**生成（`:75-81`），不是 5 个独立色 token——保证深色主题下自动跟随。
- 缓存命中率只在 ≥78% 时展示（`CACHE_HIT_RATE_DISPLAY_THRESHOLD`，`:83`、`:117-139`），注释：「生产面板只露出明显缓存收益，避免低命中率分散对上下文容量的注意力」；dev 环境绕过阈值。
- **无数据即整块隐藏**：`!renderableTaskUsage && !hasCodingPlanUsageRemaining && !hasStartPlanBalance → return null`（`:897-903`）；`used<=0 || size<=0` 也视为不可渲染，避免「误导性的 0%」（`:216-235`）。
- 触摸兜底：`@media (hover:none)` 时在 `pointerdown` 阶段主动打开，因为 HoverCard 会拦掉 click（`:977-992`）。

**办公价值**：这是 gaea 目前**最大的可见性空洞**。办公会话是「长文档 + 多文件」型，token 消耗比代码会话更快；用户不知道还剩多少预算，直到某一轮突然报「上下文超限」。
- 好处 1：**提前决定是否 `/compact`**，而不是被动失败。
- 好处 2：**解释「为什么忘了前面说的」**——分项里 `messages` 掉下去、`tool_prompt` 涨上来，用户能理解模型被工具说明挤掉了。
- 好处 3：**降低焦虑**：一个常驻的「used/total」比任何解释都有用。

例：用户让 agent 通读 12 个 sheet 的 `2026 全年费用.xlsx` + 5 份 docx 合同，跑到第 18 轮时占用 78%。hover 一看 `messages 71% / tool_prompt 12%`，于是先 `/compact` 保留结论再继续，避免了中途「模型开始重复前面的话」。

**gaea 落点**：**工具栏为空**（`composer/ComposerToolbar.tsx` 无任何 context/token 匹配）；现有能力在整页视图 `components/ContextView.tsx`（`CurrentContextCard`，`:45-198`；`ContextTrendChart`，`:221-424`；`EventsList`，`:542-617`）与 `components/context/cards.tsx`（`StatsCard/TokenCard/TimingCard/SessionInfoCard/SummaryBar`）、`components/context/cats.ts`（六分类键与语义色）、`components/context/context-view.css`。
→ 落点建议：在 `ComposerToolbar.tsx` 加一个触发器（紧凑百分比），Popover 复用 `context/cards.tsx` 的 token 数据与 `cats.ts` 的分类，**不要**新起一套分类。

**成本**：M（数据结构已有；缺的是「紧凑触发器 + 分段条 + 分项列表」三件，以及一个 `used/size` 的精简投影）。

**风险**：
- **分类口径冲突**：gaea 是六分类（`cats.ts` 的 `CONTEXT_CATS`），ZCode 是七类（含 `skills`/`mcp_tool_schemas`）。直接抄 7 类会与 `ContextView` 的分项对不上，用户看到两个页面说法不同。
- 玄墨纪律：`--color-usage-chart-1` 这类「图表色」gaea 未必有；需先确认令牌存在，否则按 v11 只能用 `--gaea-glow`/`--accent` 派生。
- `ContextView.test.tsx`（36KB）体量巨大，若复用其数据层要小心不要改动它断言的分类合计。
- 阈值 78% 这类「产品味道」的参数在办公场景可能要调高（办公用户更该看全部命中率，因为他不知道缓存是什么——**这条反而是不该抄的**，见 §8）。

### 4.8 side pane tabs

**设计要点**
- **tab 就是视图类型**：20 种 tab 类型各有一个描述符（`sidePaneTabPresentation.ts:4-26` 的 labels 表、`:114-141` 的 `getSidePaneTabTypeLabel`）。
- **标题权威**：`getSidePaneTabTitle` 按类型分派（`SidePaneTabTrigger.tsx:489-580`）；**实例序号必须留在标题里**——注释：「同一车道族的实例共用脚本里那一个名字，少了序号 tab 条上就是两个无法区分的『reviewer』」（`:503-508`）；selection-side-chat 同理拼序号（`:519-521`）。
- **图标必须与来源一致**：`plan-detail` 用 `NotepadTextIcon` 而非 `ListChecksIcon`，「那会与状态面板 Todo section 撞图标」（`:273-278`）；`workflow-artifact` **固定一枚「交付物」图标而不按 kind 变**，理由：「tab 条上的图标要在打开前就稳定（tab 从内存恢复时元数据还没读回来）」（`:300-304`）。
- **等宽与收缩**：默认 156px 等宽，空间不足平均缩到 60px，之后由外层滚动承接；「避免标题长度改变每个 tab 的宽度」（`:43-45`）。
- **标题不撑宽 tab**：`[mask-image:linear-gradient(...)]` 做右端渐隐，关闭按钮出现时改 `--tab-fade-offset`（`:243-247`）。
- **中键关闭**：`auxclick` 阶段 `preventDefault` 并关闭目标 tab，「保持原 active tab 不变」（`:134-143`）；中键 `click` 也要在 `click` 阶段先拦，避免先把 inactive tab 激活（`:117-133`）。
- **不用 button 套 button**：`TabsTrigger asChild` 把真实 DOM 换成 `div`，关闭按钮仍是原生 `button`；注释记录了否则「生产包里浏览器会修正 DOM，导致 tab/close/drag 事件错位」（`:110-116`）。
- 右键菜单：关闭 / 关闭其他 / 关闭全部（`:194-201`）；tab 支持拖拽排序并钳制 `scaleX/scaleY=1`（`:76-93`）。
- **搜索提示按类型拼**：`getSidePaneTabSearchHint` 把 tab 身份的所有可用字段拼成可搜文本（含 sessionId、artifactId 等），理由：「排查时手里往往只有它」（`sidePaneTabPresentation.ts:28-81`）。

**办公价值**：「打开过的文件」是办公的核心导航。要点是三条：①同一文件重开要复用 tab（稳定 key）；②tab 标题要能区分同名文件（版本/序号）；③tab 图标在打开前就稳定（否则恢复会话时图标跳变）。
例：一天里打开过 `报价单_v2.docx`、`报价单_v3.docx`、`成本测算.xlsx`，tab 条上三个都叫「报价单」是不可接受的——需要版本后缀或序号。

**gaea 落点**：`ChatTabs.tsx`（视图 tab：对话/轨迹/上下文/记忆 + `extraTabs` 动态会话 tab，带状态点与关闭按钮，`:91-149`；键盘 roving tabindex，`:47-62`）；文件预览 tab 在 `components/FilePreview.tsx` / `FilePreviewModal.tsx` / `PreviewNavBar.tsx` / `TabButton.tsx` / `ResizableDrawer.tsx`。

**成本**：S（`ChatTabs` 的 `extraTabs` 已支持动态 tab + 状态点 + 关闭，只需把「文件 tab」接进同一套语义）。

**风险**：
- gaea 的 `ChatTabs` 是**单一 tab 条**混放「视图」与「会话」；ZCode 的语义是「视图 tab 条 + 侧栏 tab 条」两层。混在一起会在 3 个文件后就挤爆。
- `ChatTabs.test.tsx` 断言四个视图 tab；改结构要同步。

---

## 5. 失败与恢复

### 5.1 错误呈现

**设计要点**
- 错误横幅的关闭状态是 **pane-local UI 状态**，不进 CLI projection、不写 workspace 状态桶（`docs/chat/error-banner-restore-dismissal.md:5`）。
- **指纹去重**：`session + error code + 时间 + message` 组合成指纹，当前 pane 只保留最近约 20 个 dismissed 指纹，防止同一错误反复弹（`:6`）。
- 切/重建 pane 后**允许再次展示仍然存在的权威错误**；关闭 banner 不清除 CLI error 状态（`:7-8`）。
- 失败工具卡双档：`recoverable` 的错误文字降到 `text-fg-faint/60` 并加 `line-through`（`ToolCard.tsx:47`、`:241`、`:343-347`），不可恢复才是 `text-err`；行级 `data-tone="danger|warning|info"`（`:211`）。
- 权限应答失败**就地** `role="alert"`，不用 toast（`PermissionDialog.tsx:1073-1077`）；genui 复制失败也走 `role="alert"`（`GenUiCard.tsx:215-219`）。

**办公价值**：办公用户不看日志，只看「有没有红」。可恢复错误用「弱化 + 删除线」表达，比红色警告更准确——它在说「这条读文件失败了，但 agent 已经换路子了，你不用管」。
例：agent 读 `旧版台账.xlsx` 失败（文件被占用），自动改读副本 → 卡上那行应淡出 + 删除线，而不是整张卡变红让用户以为任务失败。

**gaea 落点**：`ErrorCard.tsx`（`:9-30`，`border-l-[3px]` 红条 + 关闭）、`ToolCard.tsx:338-347`（已有 `recoverable` 双档）、`Toast.tsx`。

**成本**：S（gaea 已有 recoverable 区分，缺「指纹去重」与「切会话后可重现」两条规则）。

**风险**：`ErrorCard` 的 dismiss 目前是 `onDismiss(id)` 直接移除 item；改成指纹去重需要把「dismissed 集合」提到 pane 级，否则刷新会话后错误消失（这反而是用户想要的？需产品确认）。

### 5.2 重试

**设计要点**
- 「重试本轮」是一条**命令**而不是 UI 组合：`retryTurn { target: { rowId, entityId } }` + `baseRevision/baseLogEpoch`，CLI 统一 resolver 做 CAS 裁决并生成新 turn；**UI 不执行旧 rewind+resend 组合**（`docs/ui/chat-user-message-retry.md:3-5`）。
- 用户消息行本身的 `edit` 是另一条命令 `editUserQuery`，「与 retry 是不同命令和产品语义」（`:7`）。
- **API 自动重试是环境状态，不是错误**：`ChatApiRetryStatus` 在输入工具栏位置显示「第 N/M 次重试」的渐变文字，`title` 附带 `HTTP <status>`（`display.tsx:135-183`）。
- **刻意不做倒计时**：注释写明「当前 ZCode Agent 只会推送某一刻的 `retryDelayMs` 快照，不会每秒递减。继续把这个值渲染成『X 秒后继续』会给用户造成倒计时在卡住的错觉」（`display.tsx:149-151`）。
- 视觉上重试状态用**次级**渐变峰值（`animated-gradient-text-subtle`），与 ToolCall/Thinking 同节奏但更弱（`display.tsx:176-180`）。

**办公价值**：办公长任务里 429/网络抖动很常见。两个要点：
1. **不要把自动重试渲染成错误**——用户会去点「重试」按钮，导致重复请求。
2. **不要假装有倒计时**——比没有倒计时更伤信任。
例：agent 在写 40 页报告的第 3 章时遇到 429，工具栏显示「第 2/5 次重试」，用户知道在等，不用干预；等 8 秒后继续。

**gaea 落点**：`Transcript.tsx` 已有 `phase` item（`:571-575` 渲染 `.phase` 行）与 `WorkHeader.latestTurnPhaseText`（`WorkHeader.tsx:53-60`）显示「正在重试 (n/m)」类文案；`ErrorCard.tsx` 是终态错误。**缺**：一个「环境级重试」与「终态错误」的明确分层（现在都走 phase 文本）。

**成本**：S。

**风险**：`WorkHeader` 的 phase 文本语义由后端决定（注释：`正在重试 (n/m)`），前端加专门组件会与后端文案重复。

### 5.3 中断

**设计要点**
- Stop 只停当前执行；queue 是否 held / 续跑由 CLI projection 表达（`docs/ui/chat-message-queue.md:63`）。
- 停止后 todo 面板立刻退出 loading：`showLoadingState = isStreaming && !isAllCompleted`，注释记录了 bug——「任务停止/结束后 plan 可能还留着 in_progress 项，如果继续套 animated-gradient-text，面板就会一直显示 loading 态」（`TodoPanel.tsx:39-45`、`:78-83`）。
- 中断的工具卡保持 `status: "stopped"` 并整卡 `opacity-60`（`ToolCard.tsx:48`、`:210`）。

**办公价值**：停止后最容易出的错是「界面还在转圈」。办公用户会以为没停下来而反复点停止。
例：用户发现 agent 开始改错文件，点停止 → todo 立刻从「转圈 + 渐变」变为静态文字，工具卡淡出，用户知道「真停了」。

**gaea 落点**：`ToolCard.tsx:48`（`status === "stopped"` → `Ban` 图标 + `opacity-60`）、`TodoCard.tsx:9-18`（`in_progress` 恒 `animate-spin`，**没有** isStreaming 门控）。

**成本**：S（给 `TodoCard` 加 `running` prop，非运行期把 spinner 降级为静态箭头——ZCode 的做法 `todo.tsx:40-44`：「用静态箭头表达当前项，避免和加载动画语义混在一起」）。

**风险**：`TodoCard.test.tsx` 可能断言 spinner 存在。

### 5.4 断线重连：`desktop-continuous` vs `web-remote-replayable`

**设计要点**
- 桌面窗口（含 SSH/WSL/Docker）固定 `desktop-continuous`（连续 live 流）；手机 `/remote` 与 Web remote 固定 `web-remote-replayable`（snapshot + gap 恢复）（`docs/conversation-product-protocol.md:207`、`docs/zcode-protocol.md:10`）。
- 两者**消费同一个 CLI ProductProjection**，但**恢复语义不能互相扩散**（`docs/product-capability-map.md:155`、`docs/zcode-protocol.md:10`）。
- 每个 attachment 有独立的 host 分配 `connectionId` 与独立 subscription registry；UI-facing subscribe 不能覆盖 profile（`docs/conversation-product-protocol.md:207`）。
- 具体差异举例：
  - 手机**不接收** live `tool_input_delta`，只通过最终 `row.upserted` / snapshot 拿到完整 `input` 与 `inputText`（`docs/conversation-session-case-catalog.md` STIP03 行）。
  - 手机收到的是**权威 deadline**：`ask_user_question` 的 5 分钟自动继续在断线重连后恢复同一绝对 deadline，**不重启计时**（`docs/ask-user-question-auto-resolution.md:109`）。
  - 手机 snapshot 恢复出的 `pendingElicitations` **只恢复 UI 状态，不补发系统通知**，避免断线重连把旧阻塞请求重复通知（`docs/chat/human-intervention-notifications.md:27`）。
  - GenUI 需要桌面沙箱：手机无沙箱时显示既有「桌面专用」提示，**不新增手机渲染通路**（`docs/specs/gen-ui-output-directory.md:50-52`、`gen-ui-styles.md:46-47`）。
  - 错误终态不撤销已 accepted 的后续输入：queue 保留 + `autoDrain=false`，**同时适用于 desktop continuous 与手机 replayable 的共享 CLI admission**（`docs/conversation-protocol-declaration.md:312`）。

**办公价值**：办公用户的主力端是**手机**（在外面看结果）和**桌面**（在家改）。所以「断线回来能不能接着看」比「流是否丝滑」重要得多。
- 关键设计是「**恢复的是事实，不是动画**」：回来时看到的是终态 + 明确的 pending 请求，不需要补齐中间过程。
- 「不重复通知」这条直接决定办公体验：在外面收到 5 条「需要你确认」推送，点进去发现是同一条，是致命的。

例：用户在手机上排队 3 条指令后切到地铁无信号，20 分钟后重连：应看到 3 条中 1 条已完成、2 条仍在队列（不是「队列消失」），且待审批的那条仍在，deadline 按绝对时间（不是重新给 5 分钟）。

**gaea 落点**：gaea 是「本地优先 + Electron bridge」，**没有这两种 profile**。相关落点：`frontend/src/gaea/lib/bridge`（事件订阅）、`StreamingIndicator.tsx`（连接态兜底）、`SubagentThread.tsx:26-29`（运行中 3s 轮询 + 事件驱动刷新，页面不可见门控空转）。

**成本**：N/A（不迁移协议）；可迁移的只有一条**UI 规则**：「重连后的第一帧必须是终态，不能重放中间动画」。gaea 的 `Transcript` 已有 `scrollVersion`/`TurnBlock` memo 优化（`Transcript.tsx:31-45`、`:304-308`），可在此基础上加「恢复态不做入场动画」。

**风险**：若在 gaea 引入「快照恢复」概念，会与现有 `useLiveReload`/轮询模型冲突，且没有后端契约支撑。**明确不做**。

---

## 6. 产出物（交付物）

### 6.1 每轮文件摘要（★ 值得抄）

**设计要点**（`docs/ui/task-file-change-summary.md`）
- 位置：**该 turn 的 assistant 工作区尾部**，格式 `N files changed +A -D`；移动端同等信息但按钮/文件行更紧凑（`:97`）。
- 展开后逐个文件给「相对路径 + `+/-` 行数」，**不展示写入/修改次数**（`:98`）。
- **不再支持单个文件行内继续展开 inline diff**：文件行点击/Enter/Space 与行内 `Review` 按钮**同一行为**——打开右侧 code viewer 的 diff 预览（`:105-107`）。
- `Review` 保留文字入口、不用图标，避免与整行点击形成重复强调（`:107`）。
- 撤销：先 preview dialog，区分 `safe / unsafe / ignored`，**unsafe 原因必须可见**；apply 不做部分写回（`:100`、`:81-82`）。
- 数据口径：同一轮同一文件多次改动，摘要按「该轮最初 before → 最终 after」计算，中间 checkpoint 只用于安全撤销顺序（`:75`）。
- 失败也保留：`task_error` 同样持久化本轮 `fileChanges`，失败任务也能显示汇总（`:113-114`）。
- 投影边界：`turnHeader.fileChanges` **只承载轻量摘要**（文件数、总增删行、撤销状态），**不把完整 patch 或 before/after content 放进 replayable snapshot**（`:67-68`）。

**办公价值**：办公用户对「改了哪些文件」极度敏感（覆盖了谁的文件是事故）。这一条把「文件计数」变成可信的账本。
例：一轮里 agent 改了 `报价单_v3.docx`、`成本表.xlsx`、`封面.png`，摘要「3 个文件 +128 −12」，展开三行；点 `成本表.xlsx` 直接开右栏 diff，看它是「改了 3 个单元格」而不是重写整表。

**gaea 落点**：`DeliverableCards.tsx`（轮尾交付列表）、`DeliverablesPanel.tsx`（右栏会话产物 + 版本时间线 `VersionTimeline` + 验收徽标）、`deliverables/DeliverableRow.tsx`、`deliverables/RegistrySection.tsx`、`deliverables/EvidenceSection.tsx`、`components/ChangesPanel.tsx` + `ChangesDiff.tsx` + `FileVersionStrip.tsx`。
→ **gaea 已经有 `ChangesPanel` 承担 diff，`DeliverablesPanel` 承担产物**。缺的是 ZCode 那条纪律：**「文件行点击 = 打开 diff 预览」，不要在行内再嵌套一层 diff**（`DeliverableRow` 现在是 `onToggleTimeline` 展开版本列表 `:15-16`，与 diff 是两条路径）。

**成本**：S（纪律对齐，不是新功能）。

**风险**：`DeliverablesPanel.test.tsx`（38KB！）+ `DeliverableCards.test.tsx` + `DeliverablesPanel.thumbs.test.tsx` 是 gaea 最大的测试面，任何行内行为改动都要过这三个。

### 6.2 artifact / file summary 卡片

**设计要点**（`app-shell/workflow-artifacts/artifactPresentation.tsx`）
- **六种 kind**：`file / markdown / chart / table / metrics / board`，其中后四种是「预置看板」，**没有字节与版本**（`:21-26`）。
- **同一产物在四处必须长得一样**：run 侧板瓦片、完成卡瓦片、`workflow-artifact` tab、通知行药丸——注释写明抽出来的唯一理由：「同一个产物在四处必须长得一样。图标或 kind 词各写一遍，通知行的 chip 与它点开的 tab 迟早会用两枚不同的图标指同一件东西」（`:10-19`）。
- 图标按 kind 固定一张（`:34-58`），尺寸由**调用方**给（卡片 `size-4`、chip `size-3.5`、tab 头部 `size-4`）——「尺寸是各表面的密度决定的，图形本身不是」（`:42-45`）。
- 明细行：文件类 `PDF · 4.0 KB`（扩展名徽字优先，无扩展名退 MIME 子类型），看板类只有条数（`:108-144`、`:87-106`）。
- **交付物（primary）规则**：有 `primary` 旗子就是它；没旗子且只有一件，那件就是；两件以上无旗子就没有交付物；**「单件规则只在 UI 上成立，协议与引擎从不推断旗子」**（`:169-180`）。
- 排序：primary 带头，其余保持发布顺序（`:186-192`）。
- 「能不能在本机定位」有独立判据：必须有 `sourcePath`、**且不是远程 workspace**（`workspaceIdentity` 为空）、**且不是手机远控**（无 `remoteSessionId`）（`:231-248`）。
- 「作为文件打开」的副本文件名按**这一版**的 `sourcePath` 基名取——「扩展名决定系统用哪个 App 打开，而同一个 id 的不同版本可以来自不同路径」（`:250-270`）。
- chip 标题截断 24 字符 + 最多摆 3 枚，其余「+N」（`:200-209`）。

**办公价值**：办公产物天然分「给人看的」（图表、看板）和「给人用的」（docx/xlsx）。同一套 kind 语言 + 明确的「哪一件是交付物」，能回答用户最简单的那个问题：「这次给我的是哪个文件？」
例：一次「做一份 Q3 复盘」产出 `Q3复盘.pptx`（primary）+ `数据表.xlsx` + 两张图 + 一个看板 → 卡片上 pptx 排第一并标「交付物」，其余按发布顺序；用户不需要问「哪个是成品」。

**gaea 落点**：`DeliverableCards.tsx`（`:130-265` 每行：缩略图 + 文件名 + 路径 + 徽标 + 三个操作）、`FileThumb.tsx`、`FileChip.tsx`、`FileLinkText.tsx`、`RecentFilesBar.tsx`。

**成本**：S（gaea 已有「已更新 / 未生成 / 草稿 / 就绪」四个徽标，`DeliverableCards.tsx:172-225`，比 ZCode 的 kind 语言更细）。真正可借的只有 **primary「交付物」** 概念——gaea 现在所有产物平权。

**风险**：`officeTurnProjection.deliverablePhaseOf` 已经占用了「阶段徽标」位置（草稿/就绪），再加「交付物」徽标会挤在一行；`DeliverableCards.regress.test.tsx` 可能断言徽标数量。

### 6.3 「作为文件打开」的最后一公里（★ 值得抄）

**设计要点**（`ArtifactOpenAsFileButton.tsx`）
- **分段按钮**：左「作为文件打开」（主动作 = 系统默认 App），右 chevron 打开菜单（`:156-186`）。
- 菜单三段：
  1. **已装编辑器列表**（带图标和名字，`persistLastSelectedEditorId` 记住上次选择）——「打开方式」(`:187-201`、`:125-131`)；
  2. 分隔线；
  3. 位置类动作：「在工作区显示」（有本机可达出处时才出现）+「复制绝对路径」（`:203-213`）。
- **宿主能力不够时整个按钮缺席**（`useCanOpenArtifactAsFile`，`:55-57`），调用方据此把「在工作区显示」放回头部（`:28-29`）——即「不能做的事情不显示一个禁用按钮」。
- **按版本缓存本地副本**：`Map<version-filename, Promise<localPath>>`，连点两下只落一次；失败即出缓存，下次点击重试（`:77-91`）。
- 落副本后打开**本机副本**，远程工作区也不带 `remoteTarget`，「否则会被当成远端路径去开」（`:127-130`）。
- 失败走 toast + `logger.warn`（`:93-105`）。

**办公价值**：这是办公交付物的**真实终点**。办公用户不会满足于「在预览器里看」，他要「用 Excel 打开它、改完发邮件」。
- gaea 现在有 `app.OpenWorkspacePath`（系统默认）与 `RevealWorkspacePath`（定位），**缺「用哪个 App 打开」**——而办公用户经常装了 WPS 和 Office 两套，默认 App 可能不是他要的那个。
- **版本化**：`报价单_v3.docx` 打开后发现要改，改的是工作区原件还是副本？ZCode 明确「打开的是这一版的副本」，语义清楚。

例：产物 `供应商比价.xlsx`，用户点「作为文件打开 ▾」，选「WPS 表格」；另一次产物是旧版本 v2，他要对比，可以在菜单里「复制绝对路径」粘到别的工具里。

**gaea 落点**：`DeliverableCards.tsx:244-261`（现有三按钮：复制路径 / `app.OpenWorkspacePath` 外部打开 / `app.RevealWorkspacePath` 定位）、`deliverables/DeliverableRow.tsx:203-220`（同款三件）、`ExportMenu.tsx`、`FileMenu.tsx`（可能已有「打开方式」语义，需核对）。
→ 落点明确：把 `DeliverableCards` 的「外部打开」按钮升级为**分段按钮 + 打开方式菜单**，复用 `app` bridge 上可能已有的编辑器枚举；若 bridge 无此能力，需后端加一个 `GetInstalledEditors` 类接口（成本上升）。

**成本**：M（前端 S + 可能的后端编辑器枚举接口）。

**风险**：
- gaea 的 `app` bridge 是否有「按 App 打开」能力未知（我未读到 `lib/bridge` 的实现细节）。若无，这条从 S 变 M/L。
- `DeliverableCards.test.tsx:70` 断言 `container.querySelectorAll("button")).toHaveLength(0)`（某个场景下按钮数为 0）——加按钮会破。
- Windows 下「打开方式」的正确实现是交给系统 shell，而不是自己枚举（枚举会漏 UWP/未注册应用）。建议：主按钮直接 shell open，菜单只列「工作区显示 / 复制路径 / 用默认程序打开」，**不自建编辑器列表**（ZCode 能自建是因为它有 `getInstalledEditors` 平台服务）。

---

## 7. 打分表（15 条候选）

成本口径：**S** ≈ 0.5 天（单组件、无契约变化）；**M** ≈ 1–3 天（跨 2–3 组件或需新数据投影）；**L** ≈ 1 周+（跨层/协议/新状态机）。

| # | 候选 | 交互设计要点（一句） | 证据 | 办公价值（docx/xlsx 例） | gaea 落点 | 成本 | 主要风险 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | **上下文占用条** | 工具栏常驻触发器 + hover 面板：分段进度条 + 分项只给百分比 + 无数据即隐 | `contextUsage.tsx:995-1019`、`:150-168`、`:1047`、`:897-903` | 18 轮后占用 78%，看 `messages 71%` 决定先 compact 再继续读 5 份合同 | `ComposerToolbar.tsx`（空）+ 复用 `ContextView.tsx`/`context/cards.tsx`/`context/cats.ts` | M | 六分类 vs 七分类口径冲突；`ContextView.test.tsx` 36KB |
| 2 | **审批内联 dock** | 钉在对话流、编号+数字键、第 N 行「拒绝并附意见」、`allowAlways` 摊出将放行前缀、子代理来源徽标 | `PermissionDialog.tsx:825-835`、`:584`、`:715-726`、`:96-121`、`:331-354`、`:850-855` | 覆盖 `报价单_v3.docx` 前能上滚看单价表；能写「第 3 条单价算错了」直接拒 | `ApprovalModal.tsx:48-171` | M | `aria-modal` 断言；全局 1-5/Esc 监听与 Composer 冲突 |
| 3 | **GenUI 引用三态 + 尺寸契约** | `pending/invalid/ui` 三态、完成门槛、不吞半截正文、块高上限、切 tab 不抖 | `references.ts:23-81`、`GenUiMessageResponse.tsx:40-66`、`gen-ui-inline-sizing.md:5-35`、`page.ts:128-135` | 成本对比图流式显示「正在生成视图…」，落盘原地替换；切「按部门/按月」不跳页 | `genui/parse.ts`、`GenuiBlock.tsx`、`MemoMarkdown.tsx` | S | 与 `interaction.test.ts`/`guard.test.ts` 契约；围栏语法 vs 内联指令 |
| 4 | **每轮文件摘要 + 行内点击=Review** | 轮尾 `N files +A −D`→展开只到文件行→点行=开 diff，行内不再嵌 diff | `task-file-change-summary.md:97-107`、`:67-68` | 「3 个文件 +128 −12」，点 xlsx 行直开右栏 diff 看改了 3 个单元格 | `DeliverableCards.tsx`、`DeliverablesPanel.tsx`、`ChangesPanel.tsx` | S | 三大测试面（38KB+8KB+3KB） |
| 5 | **交付物「作为文件打开」** | 分段按钮：默认 App / 打开方式 / 在工作区显示 / 复制绝对路径；能力不足则不显示 | `ArtifactOpenAsFileButton.tsx:156-213`、`:55-57`、`:77-91` | 比价.xlsx 用 WPS 打开；v2/v3 同名文件靠版本化的副本路径区分 | `DeliverableCards.tsx:244-261`、`DeliverableRow.tsx:203-220` | M | bridge 是否支持按 App 打开；Windows 打开方式应交系统 |
| 6 | **计划卡 + 右栏「计划」tab** | 时间线只放裁切预览（`max-h-64`+渐隐）+「查看完整计划」；全文进稳定 tab；轮尾复制聚合 | `plan-detail-side-pane.md:5-24`、`:103-106`、`:16` | 30 页可研的 9 章计划：时间线 8 行，点开右栏边看边改 | `ChatTabs.tsx`（新增 tab）+ 新计划卡 | M–L | `ChatTabs` 是视图 tab 非实例 tab；`ApprovalModal` 是 modal |
| 7 | **排队暂停横幅 + 立即发送** | `autoDrain=false` 横幅（stopped/error/generic）+「继续」；每行「立即发送」= stop 当前+消费该项 | `ConversationQueuePanel.tsx:368-398`、`:244-269`、`:137-139` | 3 条排队，第 2 条 429 失败 → 队列保留、横幅「已暂停」+继续，不丢第 3 条 | `ComposerQueueList.tsx:20-164`、`Composer.tsx:107-320` | S–M | gaea queue 是本地态；无 ACK 轴；两套 queue 测试 |
| 8 | **排队/引导分轨的对话流可见性** | 待生效的 guide 立刻在流末尾显示用户气泡 +「等待引导当前任务…」，被降级则回队列面板 | `interaction-behavior.md:62-79`、`:81-101` | 用户插话「这一章先别写」→ 立刻出现在流里并标「等待引导」，不会像发丢 | `Composer.tsx`（steer）+ `Transcript.tsx` | M | 需后端区分 admitted 与 queued；gaea 无 `TurnSteerDrained` 事件 |
| 9 | **思考单行流式摘要** | 收起态 `Thinking · <当前行>`，按换行翻滚、左右 mask、溢出才挂 | `reasoning-streaming-auto-collapse.md:7-23`、`reasoning.tsx:255-271` | 显示「正在比对第 7 条与附则第 2 款」，用户知道在读合同不是卡死 | `Transcript.tsx:401-433`（`InlineReasoning`） | M | 与 `WorkHeader`/`StreamingIndicator` 动画叠加；合并推理消息取哪条 |
| 10 | **todo 全完成即隐藏 + 非运行期去 loading** | 全完成 `return null`；`showLoadingState = isStreaming && !allDone` | `TodoPanel.tsx:64-69`、`:78-83`、`renderers/todo.tsx:40-44` | 5 份 Excel 合并完，便签消失，用户不会以为还剩活 | `TodoCard.tsx:31-45`、`:9-18` | S | 「正在看时突然消失」；`TodoCard.test.tsx` |
| 11 | **可恢复错误降级 + 错误指纹去重** | recoverable 错误淡出+删除线；banner dismiss 按 `session+code+time+msg` 指纹，pane 内留 20 条 | `ToolCard.tsx:47`、`:343-347`；`error-banner-restore-dismissal.md:5-8` | 「旧版台账.xlsx 被占用」淡出删除线，不整卡变红 | `ToolCard.tsx`、`ErrorCard.tsx` | S | dismiss 集合要提升到 pane 级 |
| 12 | **环境级重试与终态错误分层** | 自动重试显示「第 N/M 次重试」渐变（不假装倒计时，title 带 HTTP 码） | `display.tsx:135-183`、`:149-151` | 写第 3 章遇 429，工具栏「第 2/5 次重试」，用户不干预也不误点重试 | `WorkHeader.tsx:53-60`（phase 文本）+ `Transcript.tsx:571-575` | S | 与后端 phase 文案重复 |
| 13 | **GenUI 源文件不进变更摘要** | 被 genui 引用的路径不计入文件数/增删行/展开列表；全被过滤则整块隐藏 | `gen-ui-file-summary.md:5-13`、`:24-27` | 「Q3 分析.docx + 内嵌图」→ 摘要只显示 1 个文件 | `DeliverableCards.tsx:57-95`、`lib/deliverablesTurn.ts`、`lib/fileLinks.ts` | S | 计数徽标断言 |
| 14 | **tab 标题/图标权威 + 中键关闭** | 标题按类型分派、实例序号必留、图标与来源一致且固定；中键关闭不改 active | `SidePaneTabTrigger.tsx:489-580`、`:300-304`、`:134-143` | 三个「报价单」靠 v2/v3 序号区分；中键关掉不跳 tab | `ChatTabs.tsx:91-149` | S | 视图 tab 与文件 tab 混一条会挤爆 |
| 15 | **工具行内细进度条** | 有 `fraction` 用 determinate，否则 indeterminate 扫光 + 截断 message | `ToolCallProgressIndicator.tsx:17-46` | 跑 12 个 sheet 的 python 显示 7/12 而不是一个转圈 | `ToolCard.tsx:237-269` | S | 需后端给 progress 字段；品牌色 vs `--gaea-glow` |

**优先级建议**
- **P0（本周可做，S 且高价值）**：#13 → #10 → #4 → #15 → #3。
- **P1（需 1 天设计对齐）**：#1（先定分类口径）→ #2（先定 dock 形态）→ #11/#12。
- **P2（需产品拍板）**：#5（bridge 能力）→ #7（queue 是否持久化）→ #6（tab 模型）→ #8（需后端事件）。

---

## 8. 不该抄的 3 条（含理由与被抄后的具体后果）

### ① hover-only 控件 + 五段式工具摘要

**是什么**：箭头 `opacity-0` 只在 hover/展开时可见（`ToolSummaryRow.tsx:213-220`）；GenUI 工具栏 `group-hover:opacity-100`（`GenUiCard.tsx:128`）；摘要行五段并列（`kindLabel | kindDetail | sourceLabel | primary | secondary/diffstat`，`ToolSummaryRow.tsx:53-162`）；窄屏到 360px 才丢信息（`:83`、`:95`）。

**为什么不该抄**：ZCode 假设「用户会 hover、会记住图标含义、在 1440px 窗口工作」。办公用户的三条现实是：a) 窗口常是半边屏；b) 触屏/远程桌面下 hover 不可靠；c) 不认 `kindLabel`（`bash`/`mcp__`）。

**抄了会怎样**：办公用户永远找不到「展开看结果」的入口（因为箭头不 hover 就不存在），于是在对话里问「文件在哪」——把 UI 缺陷变成 token 消耗。

**替代方案**：箭头常显（低对比即可）；摘要固定两段（「做了什么」+「结果」）；`kind/source/diffstat` 收进展开区；横排信息超过 2 段就用第二行。

### ② 双 delivery profile + CLI 权威 queue 全语义

**是什么**：`desktop-continuous` / `web-remote-replayable` 两套恢复语义（`zcode-protocol.md:10`）；CLI 权威 queue + `autoDrain` + `guide` + `pendingInputs` + owner/lease + reservation + Stop barrier（`interaction-behavior.md:40-123`）；`deleteQueueItem(baseRevision)` 的 stale/noop/rejected 三分支（`chat-message-queue.md:31-51`）。

**为什么不该抄**：这套是**为解决多端并发 + 权威排序**建的，代价是 5 个状态机 + 3 份契约 + 一份 spec 矩阵。gaea 是单机 Electron 前端，queue 是本地 `useState`（`Composer.tsx:109`）；照抄等于引入一个「没有对手的对手方」。

**抄了会怎样**：`Composer.queue.test.tsx` 的 6 个用例全部要重写；开发成本 >= 2 周，而用户可见收益只有两条（暂停横幅、立即发送）。

**替代方案**：只抄用户面两条（§7 #7）：「暂停横幅 + 继续」与「逐条立即发送」。语义简化为：本地队列 + 一个 `blockedReason`。**不做** ACK 三分支。

### ③ HTML 沙箱 GenUI 整套

**是什么**：visualize 技能产出 HTML 文件 → 正文内联引用 → 宿主按固定输出根目录读取 → webview/iframe 沙箱挂载 → guest 测高回传 → 展开/复制图片/Tweak（`gen-ui-output-directory.md`、`gen-ui-inline-sizing.md`、`GenUiCard.tsx`）。附带：符号链接/非普通文件/超体积校验、跨 session 读取、远端 `gen-ui` RPC 转发（`gen-ui-output-directory.md:17-19`、`:37-40`）。

**为什么不该抄**：gaea 已有**另一套** GenUI：JSON spec 组件树（`frontend/src/genui/{parse,renderNode,guard,spec,limits}.ts(x)`，`guard.ts` 25KB、`styles.css` 17KB），并已有 panel 投递、指纹去重、历史重放幂等（`gaea/lib/genuiPanel.ts:1-60`、`genui.walkthrough.test.tsx`）。再引 HTML 沙箱意味着：a) 两套 GenUI 并存，模型不知道该用哪套；b) 第二套主题投影要维护（玄墨 v11 + 沙箱 CSS 变量）；c) 手机/窄屏降级要重做一遍。

**抄了会怎样**：`genui.walkthrough.test.tsx` 的三条验收走查（围栏→渲染→action→panel→重放）失去意义；两套体系的 `action` 回传通道会打两遍补丁。

**替代方案**：只取**边界语义**（§7 #3 的三态 + 高度上限）与 **§3.7 的「源文件不计入交付物」**。渲染仍走 `GenuiBlock`。

---

## 9. 落地路线（建议）

### 第 1 批：零契约成本的可见性修复（S，1–2 天）

1. **上下文占用触发器**（#1 的触发器部分）：`ComposerToolbar.tsx` 加一个紧凑百分比；先只接 `used/size`，面板复用 `context/cards.tsx` 的 `TokenCard` 数据。分项列表作为第二步（需先定分类口径）。
2. **交付物过滤**（#13）：`deliverableMentions` 排除同轮 genui 围栏引用路径；为空则整块不渲染。
3. **todo 完成即隐藏**（#10）：加 3 秒延迟，避免「正在看时消失」。
4. **可恢复错误淡出**（#11 的一半）：gaea 已有 `recoverable`，只需确认视觉到位。

**验收**：`DeliverableCards.test.tsx`、`TodoCard.test.tsx`、`Composer.queue.test.tsx` 全绿；手工验一条「docx + 内嵌图」对话，摘要计数为 1。

### 第 2 批：审批与计划（M，3–5 天）

5. **`ApprovalModal` → dock**（#2）：去 modal、加反馈行、加 scope 摊开、加 origin badge。**先只做前两项**，scope 摊开需后端给 `permissionUpdates.rules`。
6. **计划卡**（#6 的时间线部分）：裁切预览 + 「查看完整计划」胶囊 + 复用 `ChatTabs` 打开右栏。不做实例 tab。

### 第 3 批：边界工程（M，按需）

7. GenUI 三态（#3）。
8. 交付物「打开方式」（#5，先确认 bridge 能力）。
9. 排队暂停横幅 + 立即发送（#7）。

---

## 10. 证据索引

**ZCode 文档**
- `docs/specs/gen-ui-file-summary.md`（源文件过滤）
- `docs/specs/gen-ui-inline-sizing.md`（测高/滚动/10000px）
- `docs/specs/gen-ui-output-directory.md`（固定输出根目录）
- `docs/specs/gen-ui-styles.md`（主题投影/token）
- `docs/specs/gen-ui-external-links.md`（外链手势）
- `docs/ui/task-file-change-summary.md`（每轮文件摘要/撤销）
- `docs/ui/tool-display-rendering.md`（renderer 分流/可见性）
- `docs/ui/plan-detail-side-pane.md`（计划卡 + 侧栏）
- `docs/ui/chat-message-queue.md`（queue 权威与操作）
- `docs/ui/chat-user-message-retry.md`（retryTurn）
- `docs/chat/reasoning-streaming-auto-collapse.md`（思考摘要）
- `docs/chat/interaction-behavior.md`（queue vs guide）
- `docs/chat/error-banner-restore-dismissal.md`（错误指纹）
- `docs/chat/human-intervention-notifications.md`（重连不重复通知）
- `docs/conversation-protocol-declaration.md:312`（错误终态不丢队列）
- `docs/conversation-product-protocol.md:207`（两种 profile）
- `docs/product-capability-map.md:155`（恢复语义不互相扩散）
- `docs/ask-user-question-auto-resolution.md:109`（手机恢复同一 deadline）
- `docs/zcode-protocol.md:10`（token 与 profile）

**ZCode 实现**
- `packages/ui/src/gen-ui/domain/references.ts`
- `packages/ui/src/gen-ui/components/GenUiCard.tsx`
- `packages/ui/src/gen-ui/components/GenUiMessageResponse.tsx`
- `packages/ui/src/gen-ui/contract.ts`
- `packages/ui/src/gen-ui/adapters/page.ts:128-135`
- `packages/ui/src/hooks/useGenUi.ts`
- `packages/ui/src/ToolCallBlocks/ToolSummaryRow.tsx`
- `packages/ui/src/ToolCallBlocks/ToolCallProgressIndicator.tsx`
- `packages/ui/src/ToolCallBlocks/renderers/todo.tsx`
- `packages/ui/src/ToolCallBlocks/renderers/plan-guidance.tsx`
- `packages/ui/src/TodoPanel.tsx`
- `packages/ui/src/components/ai-elements/{plan,queue,reasoning}.tsx`
- `packages/ui/src/PermissionDialog.tsx`
- `packages/ui/src/InteractionRequestOriginBadge.tsx`
- `packages/ui/src/v4/ConversationQueuePanel.tsx`
- `packages/ui/src/chat-input-toolbar/contextUsage.tsx`
- `packages/ui/src/chat-input-toolbar/display.tsx:135-183`
- `packages/ui/src/app-shell/SidePaneTabTrigger.tsx`
- `packages/ui/src/app-shell/sidePaneTabPresentation.ts`
- `packages/ui/src/app-shell/workflow-artifacts/artifactPresentation.tsx`
- `packages/ui/src/app-shell/workflow-artifacts/ArtifactOpenAsFileButton.tsx`

**gaea（均已实际读取）**
- `frontend/src/gaea/components/Transcript.tsx`（`InlineReasoning` `:401-433`、`ProcessCard` `:465-628`）
- `ToolCard.tsx`、`ToolGroup.tsx`、`TodoCard.tsx`、`WorkHeader.tsx`、`StreamingIndicator.tsx`
- `ChatTabs.tsx`、`ApprovalModal.tsx`、`ErrorCard.tsx`
- `DeliverableCards.tsx`、`DeliverablesPanel.tsx`、`deliverables/DeliverableRow.tsx`
- `Composer.tsx`、`composer/ComposerQueueList.tsx`、`composer/ComposerToolbar.tsx`
- `ContextView.tsx`、`components/context/cards.tsx`、`components/context/cats.ts`
- `TaskCenter.tsx`、`SubagentThread.tsx`、`Message.tsx`
- `GenuiPanel.tsx`、`genui.walkthrough.test.tsx`、`frontend/src/genui/*`（目录与关键文件）
- `components/memoryhub/hub.css`（玄墨 v11 令牌纪律）

**未覆盖 / 需后续补**
- `packages/ui/src/app-shell/SidePaneTabOverview.tsx`、`sidePaneLayout.ts`、`AnimatedSidePanePanel.tsx`（tab 条布局细节）
- `packages/ui/src/v4/ConversationRowView.tsx`（reasoning 行接入点，仅 grep 到行号）
- gaea 的 `frontend/src/gaea/lib/bridge`（是否有编辑器枚举/按 App 打开能力）—— 影响 §7 #5 的成本判定
- gaea 的 `FileMenu.tsx` / `ExportMenu.tsx`（是否已有「打开方式」语义）
- `docs/ui/side-pane-tabs.md` 正文（仅取实现证据，未通读）
