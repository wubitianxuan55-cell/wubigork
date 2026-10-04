# 前端审计 · 测试资产 + CSS

> 范围：`frontend/src/**/*.test.ts(x)`（435 个）+ `frontend/src/test/` + 根级测试（`App.tokens.test.ts`、`contrast-hex.test.ts`、`perf-guards.test.ts`、`novel-workspace.test.ts`、`events.test.ts`、`events-discipline.test.ts`）+ `frontend/src/events.ts` + 全部 `frontend/src/**/*.css`（26 个）+ `internal/**/*_test.go`（仅统计）。
> 排除：`frontend/node_modules/`、`frontend/dist/`、`frontend/wailsjs/`、`frontend/.tmp/`、`clones/`。
> 方法：AST 级正则解析 + 跨文件规范化窗口比对 + 类名**整词（whole-token）**匹配 + 动态串（模板字面量 / 字符串拼接 / `clsx()` 参数）上下文判定。所有分析脚本写在 `%TEMP%\gaea-audit\`，**未修改仓库任何文件**。
> 未跑全量 vitest（遵守基线约定）；未起 dev server；未 `pnpm install`；未上网。

---

## 测试统计（文件数、总 LOC、最大 15 个）

| 指标 | 数值 |
|---|---|
| 测试文件数 | **435**（`.test.ts` / `.test.tsx`，与基线 435 一致） |
| 测试总行数（含空行） | **63,214** |
| 测试总行数（非空行） | 56,012 |
| 共享测试基建 | `frontend/src/test/setup.ts`（81 行，唯一文件） |
| 平均文件体量 | 145 行 |
| >800 行文件 | **1** |
| ≥600 行文件 | 8 |
| 单 `it` >120 行 | 0 |
| describe 嵌套 ≥4 层 | **0** |
| 全仓 CSS | 26 文件 / 18,929 行 / 2,783 规则块 / 2,041 个「文件×类名」 |

**最大 15 个测试文件（按总行数）**

| # | LOC | 文件 |
|---:|---:|---|
| 1 | 950 | `frontend/src/pages/CreatePage.test.tsx` |
| 2 | 788 | `frontend/src/gaea/components/DeliverablesPanel.test.tsx` |
| 3 | 759 | `frontend/src/pages/ChapterPage.test.tsx` |
| 4 | 653 | `frontend/src/gaea/components/ContextView.test.tsx` |
| 5 | 651 | `frontend/src/schedule/GanttView.test.tsx` |
| 6 | 616 | `frontend/src/gaea/components/VersionTimeline.test.tsx` |
| 7 | 616 | `frontend/src/pages/NovelSettingPage.test.tsx` |
| 8 | 602 | `frontend/src/components/characterlib/CharacterLibEditor.test.tsx` |
| 9 | 567 | `frontend/src/gaea/lib/officeTurnProjection.test.ts` |
| 10 | 563 | `frontend/src/gaea/components/FilePreview.test.tsx` |
| 11 | 516 | `frontend/src/pages/ChatPage.test.tsx` |
| 12 | 511 | `frontend/src/components/novel/ForeshadowPanel.test.tsx` |
| 13 | 449 | `frontend/src/pages/WeixinPage.test.tsx` |
| 14 | 445 | `frontend/src/pages/sin/SinSidePanel.test.tsx` |
| 15 | 431 | `frontend/src/gaea/components/DagPanel.test.tsx` |

---

## 死/跳过测试清单（文件:行 | 类型 | 证据 | 建议）

### 结论摘要

| 类别 | 数量 | 结论 |
|---|---:|---|
| `it.skip` / `describe.skip` / `test.todo` / `xit` / `xdescribe` / `fit` / `.only` | **0** | 全仓零命中（435 文件） |
| 测试 import 了**不存在的模块路径** | **0** | 全量相对路径解析（`.ts/.tsx/.js/.jsx/.json/.css/index.*`）全部命中磁盘 |
| 测试 import 了**不存在的具名导出** | **0** | 逐个目标模块解析 export 面（含 `export *` 保守跳过）后为 0 |
| `vi.mock()` 指向**已不存在的模块** | **1** | 见下 |
| 无断言的 `it` | **2** | 见下 |
| 永真断言 `expect(true).toBe(true)` 族 | **0** | 零命中 |
| 注释掉的大块测试（>10 行） | **0** | 15 个候选经人工核对**全部是文件头 JSDoc 设计说明**，不是注释掉的测试 |
| `frontend/known-flaky.txt` | 空（仅注释） | 与「零注册 flaky」现状一致；但注释里写「2799/2799」已过期 |

### 明细

| 文件:行 | 类型 | 证据 | 建议 |
|---|---|---|---|
| `frontend/src/pages/ChatPage.test.tsx:90` | **死 mock（D1，最高价值）** | `vi.mock('../../src/wailsjsCompat', () => ({ ChatTopicsList: ..., }))` → 路径解析到 `frontend/src/wailsjsCompat(.ts/.tsx/index.*)` **磁盘上不存在**；全仓 grep `wailsjsCompat` 仅剩注释，`frontend/src/gaea/lib/bridge.variadic.test.ts:4` 明写「v4.174 退役 wailsjsCompat shim」 | **直接删除该 `vi.mock` 块（第 90-约 120 行）**。模块已被 tsc/vitest 判为不可解析；当前之所以不红，是因为源码侧已无 import，vitest 惰性解析未触发。这是全仓唯一一处指向幽灵模块的 mock。confidence: high |
| `frontend/src/components/imagegen/VisionTrial.test.tsx:57-60` | 无断言 `it`（D2） | `it('选图 → SavePastedImage 落盘 → 展示预览与落盘路径', async () => { const { container } = renderTrial(); await pickImage(container) })` — 块内 0 个 `expect`；断言全在被调 helper `pickImage` 里 | 把 helper 内断言上提 1 条到用例，或改名标注为 smoke。confidence: high |
| `frontend/src/pages/ChatPage.test.tsx:459-471` | 无断言 `it`（D2） | `it('卸载后继续推进计时器：无异常、无遗留更新（循环已中止）')` — 仅 `advanceTimersByTime`，无 `expect`，靠「不抛错」隐式通过 | 补 `expect(() => vi.advanceTimersByTime(...)).not.toThrow()` 或断言无 console.error。confidence: high |
| `frontend/known-flaky.txt:7-10` | 过期注释（低危） | 注释称「连续 3 轮全量 2799/2799 通过」，但当前基线是 **435 文件 / 3760 用例** | 更新数字，避免误导。confidence: high |

**已核实为误报、不需要动的「疑似」项**（列出以免后续重复排查）：
- 13 处 `RETURN-BEFORE-EXPECT`（`schedule/store.sync.test.ts`、`verify/VerifyArtifactsThumbs.test.tsx`、`novel/create/EditorPanel.test.tsx` 等）：逐个人工核对，`return` 均在**嵌套回调**内（`save: async (...) => { ...; return ... }`、`mockImplementation(async (rel) => { ... return ... })`），其后有正常 `expect`。**非遮蔽断言**。
- 15 处「注释代码块」：均为文件头 JSDoc（如 `AppearancePanel.test.tsx:1-25`、`toolArgs.test.ts:1-9`、`appStore.contrast.test.ts:1-18`），是设计说明而非注释掉的测试。
- 32 处 `expect(x).toBeTruthy(); // 中文说明`：`toBeTruthy` 作用在 `screen.getByText(...)` 上，是有效的存在性断言（`getBy*` 本身会抛），仅风格冗余。

---

## 【专项】gaea/styles.css 死选择器独立复核

### 复核结论

另一路审计称「53 个死选择器块 / 约 323 行（`context-menu` 全族、`ds-*`、`drawer__*`、`btn-primary/secondary`、`msg--*`）」。

**独立复核结论：方向正确，但数字偏保守，且漏判与误判各有若干。**

| 项 | 另一路 | 我的独立计数 |
|---|---:|---:|
| 死**类名**数 | —（按「块」计） | **35** |
| 死**规则块**数 | 53 | **55**（全部为「整块可删」，无与活类名共享的块） |
| 可删行数 | ~323 | **388**（并集，含块间空行/注释头） |
| 未识别 | — | `card`、`tooltip`、`md-code`、`empty-state__icon`、`pwa-titlebar`、`drawer`（基类）、`drawer--wide`、`mem-doc`、`mem-suggestion` 共 9 个额外死类 |

**必须澄清的一个高风险误判**：如果「`ds-*`」被理解成 `--ds-*` **CSS 自定义属性**，那是**严重误判** —— `--ds-bg-app` / `--ds-overlay` / `--ds-font-mono` / `--ds-font-display` / `--ds-shadow-card(-hover)` / `--ds-shadow-panel` / `--ds-shadow-dropdown` 定义在 `gaea/styles.css:74-88`，被 **18 处以上**代码消费（`ApprovalModal.tsx:99-101`、`Composer.tsx:477`、`ExportMenu.tsx:69`、`ResizableDrawer.tsx:181`、`Toast.tsx:53`、`TodoCard.tsx:57`、`XlsxPreview.tsx:514`、`MenuContainer.tsx:17`、`CommandPalette.tsx:140`、`Welcome.tsx:226,249`、`SkillCaptureModal.tsx:123`、`AskCard.tsx:155` …）。**可删的只是 `.ds-*` 类选择器，绝不是 `--ds-*` 变量。**

### 判定方法（防误报的关键）

1. 类名**整词**匹配：正则 `(?:^|[^A-Za-z0-9_-])<cls>(?:$|[^A-Za-z0-9_-])`。这一步排除了三类高频误报：`mem-card` 不匹配 `card`、`ant-btn-primary` 不匹配 `btn-primary`、`plan-words-input` 不匹配 `ds-input`。
2. **动态拼接探测**：对每个候选类名生成前缀探针（BEM 分隔符 `--` / `__`、逐级剥尾段、去尾部数字），要求命中点落在**模板字面量内**、**字符串拼接旁**或 **`clsx()/classNames()/cn()` 实参**中才算「活」。这一步否决了如 `.context-menu` 因探针 `context` 命中 `` data-testid={`agent-view-context-${...}`} `` 而被误判为活的情况。
3. **库生成类豁免**：`hljs-*`（highlight.js ^11.12 真实依赖，`gaea/lib/codeHighlight.ts` 产 span）、`ant-*`/`anticon`（antd 运行时注入）、`tok-*`（`@lezer/highlight` 的 `classHighlighter`，见 `gaea/components/changesdiff-tok.css:1-14` 自述）、`function_`/`class_`（hljs 复合作用域名 `hljs-title.function_`）——**一律不判死**。
4. 反例自检：`.sched-group-c0..c5` 由 `` `sched-group-c${d.synSeq % 6}` ``（`schedule/gantt/GanttBars.tsx:69`）生成，**正确判为活**；`gui-text-${size}`、`gui-tone-${tone}`、`gui-btn-${tone}`（`genui/renderNode.tsx:60,135,581`）同样正确处理。

### 详细清单（gaea/styles.css）

| 类名 | 规则块行号 | grep 证据（frontend 代码，整词） | 可删/不可删 | 置信度 |
|---|---|---|---|---|
| `.btn-primary` | 759-775, 775-778, 778-781, 781-787 | 整词命中 0；仅 `ant-btn-primary`（antd）与 `gui-btn-primary`（genui） | **可删** | high |
| `.btn-secondary` | 787-802, 802-805, 805-809 | 整词命中 0 | **可删** | high |
| `.card` | 809-817 | 整词命中 264 处，但 `className` 上下文里全是 `card.title` / `card.genre` 等**属性访问**；`className="card"` 零命中 | **可删** | high |
| `.tooltip` | 1319-1325, 1325-1345, 1345-1348 | `className="tooltip"` 零命中；唯一 `tooltip` 出现在 `TaskInboxPanel.tsx:110` 的 antd `ellipsis={{ tooltip: ... }}` 选项对象；`data-tooltip` 全仓 0 命中（`.tooltip::after{content:attr(data-tooltip)}` 永不触发） | **可删** | high |
| `.context-menu` | 1348-1358 | `context-menu` / `contextmenu` 全仓 0；仅 `onContextMenu` React 处理器（原生右键，与 CSS 无关） | **可删** | high |
| `.context-menu__sep` | 1358-1363 | 同上 | **可删** | high |
| `.context-menu__item` | 1363-1377, 1377-1383, 1383-1387 | 同上 | **可删** | high |
| `.context-menu__item--danger` | 1387-1390, 1390-1394 | 同上 | **可删** | high |
| `.context-menu__item--section` | 1394-1402 | 同上 | **可删** | high |
| `.context-menu__icon` | 1402-1409 | 同上 | **可删** | high |
| `.context-menu__label` | 1409-1412 | 同上 | **可删** | high |
| `.drawer` | 1256-1266 | 整词命中 21 处**全部是英文注释**（`useSessionHandlers.ts:53` "History drawer:"、`CapabilitiesPanel.tsx:6`、`DrawerHeader.tsx:4`）；`className` 上下文 0 | **可删** | high |
| `.drawer--wide` | 1266-1269 | 整词命中 0 | **可删** | high |
| `.drawer__head` | 1269-1277 | 整词命中 0（注释里的 `drawer` 不构成 `__head` 证据） | **可删** | high |
| `.drawer__title` | 1277-1282 | 整词命中 0 | **可删** | high |
| `.drawer__summary` | 1282-1287 | 整词命中 0 | **可删** | high |
| `.drawer__close` | 1287-1301, 1301-1305 | 整词命中 0 | **可删** | high |
| `.ds-card` | 725-732 | `ds-card` 整词 0 | **可删** | high |
| `.ds-input` | 732-743, 743-747, 747-750 | `ds-input` 整词 0（唯一的子串命中是 `ChapterPlanCard.tsx:641` 的 `data-testid="plan-words-input"`，`wor**ds-input**` 巧合） | **可删** | high |
| `.ds-kbd` | 750-759 | 整词 0 | **可删** | high |
| `.ds-code-inline` | 817-827 | 整词 0 | **可删** | high |
| `.ds-code-block` | 827-837 | 整词 0 | **可删** | high |
| `.ds-modal-overlay` | 837-848 | 整词 0 | **可删** | high |
| `.ds-modal` | 848-857 | 整词 0 | **可删** | high |
| `.ds-button-pill` | 918-934, 934-940, 940-943 | 整词 0 | **可删** | high |
| `.empty-state__icon` | 969-973 | 整词 0；`.empty-state`（957-969）**是活的**（`CapabilitiesPanel.tsx:112` `className="empty-state"`），但 `__icon` 无任何使用 | **可删** | high |
| `.md-code` | 464-472 | 整词 0。注意 `.md`（383-464, 127 行）**是活的**（`Markdown.tsx:558` 与 `MemoMarkdown.tsx:101` 均 `<div className="md ...">`），**不要连带删** | **可删** | high |
| `.mem-doc` | 1242-1245 | 整词 0 | **可删** | high |
| `.mem-suggestion` | 1236-1239, 1239-1242 | 整词 0 | **可删** | high |
| `.msg` | 589-599 | `className="msg"` 零命中；整词 351 处全是 `err.message` / 变量 `msg` / i18n key `msg.thinking`。`gaea/components/Message.tsx` 已全量 Tailwind 化（42 处 className 无一个是 `msg*`） | **可删** | high |
| `.msg--user` | 599-606, 606-613, 613-619, 1460-1465 | `msg--` / `msg__` 全仓 **0 命中**（含模板串） | **可删** | high |
| `.msg--assistant` | 586-589, 619-627, 627-632, 1465-1469 | 同上 | **可删** | high |
| `.msg__body` | 606-613, 627-632, 1460-1465, 1465-1469 | 同上 | **可删** | high |
| `.msg__text` | 613-619 | 同上 | **可删** | high |
| `.pwa-titlebar` | 1427-1430 | 全仓（含 `.html`/`.go`/`.json`）仅 styles.css 自身 1 处 | **可删** | high |
| `.ant-layout` | 1431-1437, 1437-1440 | 代码里 0 字面量，但 **antd `<Layout>` 运行时注入 `ant-layout`** | **不可删** | high |

**styles.css 小计：可删 35 个类名 / 55 个规则块 / 388 行（行号并集）**
可删行段（并集，已合并相邻）：`464-472`、`586-632`、`725-857`、`918-943`、`969-973`、`1236-1245`、`1256-1305`、`1319-1412`、`1427-1430`、`1460-1469`。

**特别提醒（不要在同一个 PR 里连坐删掉）**：`.md`(383-464)、`.layout`(197-210)、`.reasoning*`(632-725)、`.transcript`(344-353)、`.notice`(494-508)、`.phase`(508-516)、`.badge*`(881-918)、`.palette*`(1019-1231)、`.ds-chip*`(857-881)、`.drawer-backdrop`(1245-1256) **全部是活的**）。其中 `.layout` 的用法很容易被漏掉：`gaea/App.tsx:626-636` 用**数组字面量**拼串（`["layout", sidebarCollapsed ? "layout--sidebar-collapsed" : "", ...]`），任何基于 `className="layout"` 的朴素 grep 都会误判为死。

**附带反向发现**：`ResizableDrawer.tsx:175` 会加 `drawer--resizing`，但 styles.css **没有任何 `.drawer--resizing` 规则** —— 活类名配空样式，属遗留。

---

## 其它 CSS 疑似死类清单

同样方法（整词匹配 + 动态探针 + 库豁免）复核 26 个 CSS 文件。全仓原始候选 145 个，其中 83 个是库生成类（`ant-*` 40、`hljs-*` 29、`tok-*` 13、`function_`/`class_` 2）**不可删**；`anticon` 另 6 处亦为 antd 生成（`<span class="anticon">`），**不可删**。扣掉后得：

### index.css（可删 17 类 / 31 块 / 178 行）

| 类名 | 行号 | 证据 | 结论 | 置信度 |
|---|---|---|---|---|
| `.neon-card` | 270-273, 602-614, 614-633, 633-638, 638-639, 639-643 | `neon-card` 整词 0；`neon-` 全仓仅 1 处 `WelcomePage.tsx:31` 的 `neon-glow-text` | 可删 | high |
| `.void-card` | 238-248, 248-258, 258-263 | 整词 0（`void` 命中全是 `Promise<void>`） | 可删 | high |
| `.md-card` | 103-114, 114-118, 118-122, 124-125 | 整词 0 | 可删 | high |
| `.md-ripple` | 126-132, 132-143, 143-146 | 整词 0 | 可删 | high |
| `.md-surface` | 74-84 | 整词 0。注意 `--md-sys-color-*` **变量**是活的，命名空间不同 | 可删 | high |
| `.md-surface-container` / `-high` / `-highest` | 84-88 / 88-92 / 92-96 | 整词各 0 | 可删 | high |
| `.md-elevation-1..5` | 96-99 / 99-100 / 100-101 / 101-102 / 102-103 | 整词各 0 | 可删 | high |
| `.bento-grid` | 438-446, 447-447, 448-448, 449-449 | 整词 0；`bento` 3 处命中均为 `boards/launcher.ts:23` 的 **数据字段** `bento: [...]`，非类名 | 可删 | high |
| `.gp-panel` | 571-581 | 整词 0 | 可删 | high |
| `.gaea-brand-text` | 663-669 | 整词 0 | 可删 | high |
| `.page-transition-enter` | 357-360 | 整词 0 | 可删 | high |

### chat-board.css（可删 5 类 / 5 块 / 53 行）

`.chat-drawer`(773-784, 1261-1268)、`.chat-drawer-toggle`(784-792)、`.chat-trust-fill`(193-201, 1261-1268)、`.chat-trust-track`(177-185)、`.chat-trust-bar`(185-193) —— `chat-drawer` / `chat-trust` 前缀全仓 **0 命中**（含模板串），整词 0。**可删**，confidence: high。

### gaea/components/memoryhub/hub.css（可删 4 类 / 9 块 / 56 行）

`.hub-card`(540-553, 553-562, 562-569, 569-570, 607-608, 611-612)、`.hub-card-icon`(570-575)、`.hub-badge`(575-582)、`.hub-enter`(582-589, 606-607, 610-611)。已枚举代码里全部 47 个 `hub-*` token（`hub-bg`/`hub-body`/`hub-detail*`/`hub-grid`/`hub-rail*`/`hub-inspector*`/`hub-search-*`/`hub-strip*`/`hub-workspace`/`hub-empty*`/`hub-hit*`/`hub-scanline`/`hub-scope-*`/`hub-library-zone`/`hub-main`），**不含**上述 4 个。**可删**，confidence: high。

### 其它单点（可删 7 行，confidence: medium）

`gaea/redesign.css:143-160` 的 `.msg--assistant` / `.msg--user` / `.msg__body`（18 行，与 styles.css 同族，Duplicate 详见下节）；`pages/modelcenter/modelcenter.css:1009-1014` `.is-accent`；`pages/sin/sin.css:714,729,740` `.is-failed`；`pages/weixin-page.css:226-227` `.is-expired`；`novel-workspace.css:511` `.is-abandoned`。
`is-failed` / `is-expired` / `is-abandoned` / `is-accent` 已专项检查动态构造（grep `'is-'`、`"is-"`、`` `is- ``、`is-${` → **全部 0**），故中等置信度判死；但这类「状态修饰符」最容易被未来的 `` `is-${status}` `` 复活，**建议只删不常用的三个（`is-expired`/`is-abandoned`/`is-accent`），`is-failed` 先保留观察一轮**。

### 汇总

| 文件 | 可删类名 | 规则块 | 可删行（并集） |
|---|---:|---:|---:|
| `gaea/styles.css` | 35 | 55 | 388 |
| `index.css` | 17 | 31 | 178 |
| `chat-board.css` | 5 | 5 | 53 |
| `gaea/components/memoryhub/hub.css` | 4 | 9 | 56 |
| `gaea/redesign.css` | 3 | 3 | 18 |
| `pages/modelcenter/modelcenter.css` | 1 | 1 | 6 |
| `pages/sin/sin.css` | 1 | 3 | 3 |
| `pages/weixin-page.css` | 1 | 1 | 2 |
| `novel-workspace.css` | 1 | 1 | 1 |
| **合计** | **68** | **109** | **705** |

**不可删（库生成 / 动态构造）必须保留的族**：`hljs-*`（highlight.js，29 类）、`tok-*`（Lezer classHighlighter，13 类）、`ant-*` + `anticon`（antd，46 类）、`gui-*`（GenUI 动态拼）、`sched-group-c0..c5`、`sched-base-*`、`novel-cover-tone-*`、`v3-rise-*`、`badge--*`、`is-active`。

---

## 重复 CSS 变量/规则清单

### 重复自定义属性：28 个同名多次定义，其中 25 个值不同

**绝大多数是合法的主题覆盖，不是缺陷**（务必不要「统一」它们）：

| 变量 | 次数 | 性质 | 判定 |
|---|---|---|---|
| `--hl-*`（`--hl-builtin`/`--hl-comment`/`--hl-func`/`--hl-keyword`/`--hl-meta`/`--hl-number`/`--hl-string`/`--hl-type`） | 各 3 | `hljs-theme.css:32-61` 的 dark / light / gaea 三套作用域值 | **合法**（主题切换机制） |
| `--read-bg` / `--read-fg` / `--read-sub` | 各 3 | `novel-workspace.css:1394-1406` 三套阅读主题 | **合法** |
| `--sched-normal/link/group/deadline/critical/front` | 各 2 | `schedule.css:6-11` 与 `18-23` = 明/暗两套 | **合法** |
| `--wails-draggable` | 6 | `styles.css:269,306,1097,1137` + `tailwind.css:133,137` = `drag`/`no-drag` 工具值 | **合法**（按元素取值） |
| `--border` / `--border-soft` | 各 2 | `redesign.css:14-15` **作用域是 `.gaea-app-layout`**（非 `:root`），`styles.css:24-25` 是 `:root` | **合法**（作用域覆盖，非重复定义） |
| `--ml-card-line` / `--ml-card-line-soft` | 各 2 | `module-launcher.css:50-51` 与 `84-85` | **合法**（猜测为预设作用域覆盖；需人工确认是否同选择器） |
| `--sidebar-width` | 2 | `styles.css:200`（展开）/ `219`（收起态） | **合法** |

**真正的缺陷（2 条，均可删/可收敛）**：

| 变量 | 定义 A | 定义 B | 问题 | 置信度 |
|---|---|---|---|---|
| `--font-size-base` | `gaea/tailwind.css:75` → **13px**（Tailwind v4 `@theme` 块，编译后落 `:root`） | `index.css:8` → **14px**（`:root`） | 两个全局 `:root` 级定义、值不同，**结果取决于页面 CSS 加载顺序**：`GaeaPage.tsx:3-5` 先 `index.css`（`main.tsx`）后 `tailwind.css` → tailwind 赢（13px）；不含 tailwind.css 的页面 → 14px。同仓两套字号真源。**该变量目前 0 消费者**（grep 仅定义处） | high |
| `--font-size-lg` | `gaea/tailwind.css:76` → **14px** | `index.css:9` → **16px** | 同上顺序依赖；且 `index.css:167` 有实际消费者 `font-size: var(--font-size-lg) !important` → 在 gaea 工作台内会解析成 14px 而非 16px，**与作者意图（16px）不符** | high |

建议：把 `--font-size-base` 从 `tailwind.css:75` 删除（无消费者）；`--font-size-lg` 二者取一并只留一处（倾向保留 `tailwind.css` 并由 `index.css` 改引用，或反之）。

### 重复规则块：70 组、约 499 行可回收 —— **但不建议提取**

方法：按「声明集合规范化后完全相同且 ≥3 条声明」分组。命中 70 组，最大 8 处相同。典型：

| 重复声明体 | 出现次数 | 位置 |
|---|---:|---|
| `display:flex;flex-direction:column;gap:8px` | 8 | `chat-board.css:1033`、`character-library.css:107,621,733`、`genui/styles.css:464`、`novel-workspace.css:655`、`modelcenter.css:736`、`sin.css:385` |
| `display:flex;flex-direction:column;gap:10px` | 7 | `character-detail.css:499`、`module-launcher.css:252`、`hub.css:426`、`novel-workspace.css:1673,1754`、`modelcenter.css:966`、`sin.css:784` |
| `display:flex;flex-wrap:wrap;gap:6px` | 7 | `chat-board.css:1211`、`character-library.css:137,550,609`、`modelcenter.css:935,1206`、`settings-page.css:390` |
| `display:flex;flex-direction:column;gap:4px` | 6 | `chat-board.css:1121`、`genui/styles.css:700`、`novel-workspace.css:1844`、`modelcenter.css:419`、`sin.css:811,814` |
| `body: background:var(--v3-edge);content:'';height:1px;left:0;position:absolute;right:0;top:0` | 3 | `chat-board.css:65`、`imagegen.css:310`、`modelcenter.css:631` |
| `align-items:center;display:flex;flex-wrap:wrap;gap:6px` | 4 | `chat-board.css:156`、`novel-workspace.css:1008`、`modelcenter.css:603,1199` |

**判定：不值得动。** 这些是 3-5 条 Flex 布局惯用式，把它们抽成共享类会造成**无关组件之间的耦合**（改一处影响 8 个板块的布局），可回收的 499 行里绝大多数是这类。唯一值得考虑的是 `--v3-edge` 顶部 1px 高光线那组（3 处完全一致），可抽成 `.v3-hairline` 工具类 —— **但它属于 `v3/foundation.css` 的设计语言，建议由样式负责人决定，不在本次清理范围。**

### CSS 加载顺序隐患（顺带发现）

同一批 `--*` 令牌在多页面以**不同顺序**加载（`GaeaPage.tsx:3-5` = styles→tailwind→redesign；`SchedulePage.tsx:60-63` = schedule→styles→redesign→tailwind；`CostLibraryPage.tsx:18-20` = styles→tailwind→hub）。目前因为各文件 `:root` 定义值大体一致（除上面 2 条 `--font-size-*`）而未爆雷，但**顺序敏感**是既成事实。confidence: medium。

---

## 重复 mock/helper/fixture 清单（含提取建议与每处行数）

### 已知样例核实结果

| # | 声称 | 核实结论 | 每处行数 | 可回收 |
|---|---|---|---|---|
| 1 | `schedule/{AoaView,baseline,BaselinesPanel,deadline,GanttView,TemplatesPanel}.test.tsx` 有 6 份逐字相同的任务 fixture | **成立**。`function chainProject(): SchedProject` 6 份**逐字相同**（含 3 tasks + 3 links + `startDate:'2026-09-07'`）：`AoaView.test.tsx:16-30`、`baseline.test.ts:13-27`、`BaselinesPanel.test.tsx:39-53`、`deadline.test.ts:12-26`、`GanttView.test.tsx:23-37`、`TemplatesPanel.test.tsx:21-35` | 15 行/份 | **75 行** |
| 2 | `schedule/{aoa,aoaLayout,cpm,pathDriver}.test.ts` 有 4 份 `t()`/`l()` 构建器 | **成立**。`function t(id,duration,extra?): SchedTask` + `function l(from,to,type='FS',lag=0): SchedLink`：`aoa.test.ts:5,8`、`aoaLayout.test.ts:7,10`、`cpm.test.ts:5,8`、`pathDriver.test.ts:13,16` | 约 6-7 行（t+l） | **约 26 行** |
| 3 | `pages/sin/*.test.tsx` 有 4 份相同 `vi.mock('../../gaea/lib/bridge')` | **成立**。8 行/份、含 `async (importOriginal)` + `new Proxy`：`SinIllustration.progress.test.tsx:25-32`、`useSinCast.test.tsx:12-19`、`useSinNotes.test.tsx:10-17`、`useSinStory.test.tsx:23-30` | 8 行/份 | **24 行** |
| 4 | `components/novel/create/*.test.tsx` 有 3 份 `stubRuntime()` | **成立但更广**：`NewCharactersModal.test.tsx:26-46`（21 行）、`useChapterGateNotice.test.ts:13-34`（22 行）、`useChapterStream.test.ts:14-35`（22 行）。**另有 2 份同名不同体**：`components/novel/BookSearchModal.test.tsx:39`、`pages/sin/SinBookSourcePanel.test.tsx:42` → 实际 **5 份 `stubRuntime`** | 20-22 行/份 | **约 65 行** |

### 扩展发现（新组）

| # | 组名 | 文件:行 | 每处行数 | 可回收 | 提取建议 |
|---|---|---|---:|---:|---|
| 5 | `new Proxy` 假 bridge（`app` 命名空间代理直通） | `components/novel/ConvergeModal.test.tsx:12`、`StorySpinePanel.test.tsx:15`、`StyleFingerprintPanel.test.tsx:17` | 约 14 | 约 42 | 与 #3 合并为 `frontend/src/test/helpers/bridgeMock.ts`，导出 `vi.mockBridgePassthrough()` |
| 6 | pptx 文件条目 fixture（`name:"汇报.pptx"`, `path:"exports/汇报.pptx"`, `size:4096`, `kind:"pdf"`） | `gaea/components/FilePreview.test.tsx`、`FilePreviewModal.test.tsx`、`PptxEditPanel.test.tsx` | 约 8 | 约 16 | `test/helpers/fixtures.ts` 导出 `makeFileEntry(over)` |
| 7 | Pptx/FilePreview 的 `kind` 值集与 `entry()` 工厂 | `gaea/components/PptxEditPanel.test.tsx`、`verify/VerifyArtifactsThumbs.test.tsx` | — | — | 同上 |
| 8 | 局部 `render<Something>` 包装函数 | **63 个测试文件**各自声明 | — | — | 不建议全量抽：包装差异大；但可抽一个 `test/renderWithProviders(ui, opts)` 覆盖最常用的 i18n + antd 组合 |
| 9 | 局部 `make*/mk*/build*/create*/stub*/fake*/mock*` 工厂 | **45 个测试文件**各自声明；重复名仅 `stubRuntime` x5 | — | — | 见 #4/#5 |

**总计可回收（确定项 #1-#6）：约 248 行**，且都是**逐字或近逐字**复制，提取风险低。建议落点：
- `frontend/src/test/helpers/scheduleFixtures.ts` → `chainProject()` + `t()`/`l()`（覆盖 #1、#2）
- `frontend/src/test/helpers/bridgeMock.ts` → bridge `vi.mock` 工厂（覆盖 #3、#5）
- `frontend/src/test/helpers/stubRuntime.ts` → 统一 `stubRuntime()`（覆盖 #4，5 份统一为 1 份）
- `frontend/src/test/helpers/fixtures.ts` → 文件条目工厂（覆盖 #6）

> 注：`vi.mock()` **必须**在模块顶层且路径解析为调用方相对路径，因此 bridge 工厂不能直接 `export function mockBridge()` 让各测试调用 —— 需要用 `vi.doMock` + `beforeEach`，或在每个测试文件保留一行 `vi.mock('...', () => import('.../bridgeMock').then(m => m.factory()))`。**这是本清单里唯一有实现难度的项**，其余 3 组是纯函数/fixture，零风险。

---

## Top 20 发现

| # | 优先级 | 路径:行 | 符号 | D 类型 | LOC | confidence | 证据 / 建议 |
|---|---|---|---|---|---|---|---|
| 1 | **P0** | `frontend/src/pages/ChatPage.test.tsx:90` | `vi.mock('../../src/wailsjsCompat')` | D1 死 mock | ~30 | high | 该模块 v4.174 已退役，全仓仅剩注释；路径在磁盘上不存在。**直接删该 mock 块**。全仓唯一指向幽灵模块的引用 |
| 2 | **P0** | `frontend/src/gaea/styles.css:1348-1412` | `.context-menu` 全族（7 类） | CSS 死类 | 74 | high | `context-menu`/`contextmenu` 全仓 0；仅 `onContextMenu` React 处理器。**可删** |
| 3 | **P0** | `frontend/src/gaea/styles.css:589-632 + 1460-1469` | `.msg` / `.msg--user` / `.msg--assistant` / `.msg__body` / `.msg__text` | CSS 死类 | 96 | high | `msg--`/`msg__` 全仓 0（含模板串）；`Message.tsx` 42 处 className 已全 Tailwind 化。**可删**（连带 `redesign.css:143-160`） |
| 4 | **P0** | `frontend/src/gaea/styles.css:725-857, 918-943` | `.ds-card`/`.ds-input`/`.ds-kbd`/`.ds-code-*`/`.ds-modal*`/`.ds-button-pill`（8 类） | CSS 死类 | 121 | high | 整词 0。**可删**；⚠️ **不可连带删 `--ds-*` 变量**（18+ 处消费） |
| 5 | **P0** | `frontend/src/gaea/styles.css:1256-1305` | `.drawer` / `.drawer--wide` / `.drawer__head/title/summary/close` | CSS 死类 | 68 | high | 整词命中全是英文注释；`ResizableDrawer.tsx` 已 Tailwind 化。**可删** |
| 6 | **P1** | `frontend/src/gaea/styles.css:759-809` | `.btn-primary` / `.btn-secondary` | CSS 死类 | 57 | high | 整词 0（只有 `ant-btn-primary`/`gui-btn-primary`）。**可删** |
| 7 | **P1** | `frontend/src/index.css:74-146, 238-263, 270-273, 438-449, 571-581, 602-643, 663-669` | `.md-*`(11 类)、`.neon-card`、`.void-card`、`.bento-grid`、`.gp-panel`、`.gaea-brand-text`、`.page-transition-enter` | CSS 死类 | 178 | high | 整词 0 + 动态探针 0。`bento` 命中是 `launcher.ts:23` 的数据字段。**可删** |
| 8 | **P1** | `frontend/src/gaea/tailwind.css:75-76` vs `frontend/src/index.css:8-9` | `--font-size-base`(13 vs 14px) / `--font-size-lg`(14 vs 16px) | CSS 重复变量 | 4 | high | 两个 `:root` 级定义值不同 → 结果**依赖页面加载顺序**；`index.css:167` 消费者在 gaea 页内拿到 14px 而非作者意图 16px。`--font-size-base` 目前 0 消费者，**先删它** |
| 9 | **P1** | `frontend/src/gaea/styles.css:1319-1348` | `.tooltip`（+`::after`/`:hover::after`） | CSS 死类 | 30 | high | `className="tooltip"` 0；`data-tooltip` 全仓 0 → `attr(data-tooltip)` 永不触发。**可删** |
| 10 | **P1** | `frontend/src/gaea/styles.css:809-817` | `.card` | CSS 死类 | 9 | high | 264 处整词命中全是 `card.title` 等属性访问；`className="card"` 0。**可删**（整词匹配的价值示范） |
| 11 | **P1** | `frontend/src/schedule/{AoaView,baseline,BaselinesPanel,deadline,GanttView,TemplatesPanel}.test.tsx` | `chainProject()` fixture ×6 | D4 重复 | 90 | high | 逐字相同 15 行 ×6。抽 `test/helpers/scheduleFixtures.ts`，**可回收 75 行**，零风险 |
| 12 | **P1** | `frontend/src/components/novel/create/{NewCharactersModal,useChapterGateNotice,useChapterStream}` + `BookSearchModal` + `sin/SinBookSourcePanel` | `stubRuntime()` ×5 | D4 重复 | 106 | high | 声称 3 份，实为 5 份。统一到 `test/helpers/stubRuntime.ts`，**可回收约 65 行** |
| 13 | **P1** | `frontend/src/pages/sin/{SinIllustration.progress,useSinCast,useSinNotes,useSinStory}.test.tsx` | `vi.mock('../../gaea/lib/bridge')` ×4 | D4 重复 | 32 | high | 逐字相同 8 行 ×4。**可回收 24 行**；需 `vi.doMock` 或保留 1 行 `vi.mock` 转发（有实现难度） |
| 14 | **P1** | `frontend/src/components/novel/{ConvergeModal,StorySpinePanel,StyleFingerprintPanel}.test.tsx` | `new Proxy` 假 bridge ×3 | D4 重复 | 42 | high | 新发现组，与 #13 同族，合并提取 |
| 15 | **P1** | `frontend/src/gaea/components/memoryhub/hub.css:540-612` | `.hub-card`/`.hub-card-icon`/`.hub-badge`/`.hub-enter` | CSS 死类 | 56 | high | 已枚举全仓 47 个 `hub-*` token，不含这 4 个。**可删** |
| 16 | **P1** | `frontend/src/chat-board.css:177-201, 773-792, 1261-1268` | `.chat-drawer`/`.chat-drawer-toggle`/`.chat-trust-*`（5 类） | CSS 死类 | 53 | high | `chat-drawer`/`chat-trust` 前缀全仓 0（含模板串）。**可删** |
| 17 | **P2** | `frontend/src/components/imagegen/VisionTrial.test.tsx:57` | `it('选图 → SavePastedImage 落盘 → …')` | D2 无断言 | 4 | high | 块内 0 `expect`，断言藏在 helper。上提 1 条或标注 smoke |
| 18 | **P2** | `frontend/src/pages/ChatPage.test.tsx:459` | `it('卸载后继续推进计时器…')` | D2 无断言 | 13 | high | 靠「不抛错」隐式通过。补 `not.toThrow()` 或断言无 `console.error` |
| 19 | **P2** | `frontend/src/gaea/redesign.css:143-160` | `.msg--assistant`/`.msg--user`/`.msg__body` | CSS 死类（跨文件重复） | 18 | high | 与 #3 同族，重复定义在第二个文件。**同 PR 一起删** |
| 20 | **P2** | `frontend/src/gaea/styles.css:1427-1430` / `:464-472` / `:969-973` / `:1236-1245` | `.pwa-titlebar` / `.md-code` / `.empty-state__icon` / `.mem-doc`+`.mem-suggestion` | CSS 死类 | 32 | high | 4 处零散死类，易被漏。**一并删**。⚠️ `.empty-state`(957-969) 与 `.md`(383-464) **是活的**，勿连坐 |

---

## 全量发现表

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---:|---|---|---|
| `frontend/src/pages/ChatPage.test.tsx:90` | `vi.mock('../../src/wailsjsCompat')` | D1 死 mock | ~30 | high | 磁盘无该模块；v4.174 已退役 | 删除 |
| `frontend/src/components/imagegen/VisionTrial.test.tsx:57` | `it('选图 → …')` | D2 无断言 | 4 | high | 块内 0 expect | 上提断言 |
| `frontend/src/pages/ChatPage.test.tsx:459` | `it('卸载后继续推进计时器…')` | D2 无断言 | 13 | high | 块内 0 expect | 补 not.toThrow |
| `frontend/src/gaea/styles.css:759-809` | `.btn-primary`, `.btn-secondary` | CSS 死类 | 57 | high | 整词 0（仅 ant-/gui- 前缀） | 可删 |
| `frontend/src/gaea/styles.css:809-817` | `.card` | CSS 死类 | 9 | high | `className="card"` 0 | 可删 |
| `frontend/src/gaea/styles.css:1319-1348` | `.tooltip` | CSS 死类 | 30 | high | `data-tooltip` 全仓 0 | 可删 |
| `frontend/src/gaea/styles.css:1348-1412` | `.context-menu` 族 ×7 | CSS 死类 | 74 | high | `context-menu` 全仓 0 | 可删 |
| `frontend/src/gaea/styles.css:1256-1305` | `.drawer` 族 ×6 | CSS 死类 | 68 | high | 整词命中全是英文注释 | 可删 |
| `frontend/src/gaea/styles.css:725-857` | `.ds-card/.ds-input/.ds-kbd/.ds-code-*/.ds-modal*` | CSS 死类 | 93 | high | 整词 0 | 可删（勿删 `--ds-*`） |
| `frontend/src/gaea/styles.css:918-943` | `.ds-button-pill` | CSS 死类 | 28 | high | 整词 0 | 可删 |
| `frontend/src/gaea/styles.css:589-632` | `.msg` 族 ×5 | CSS 死类 | 70 | high | `msg--`/`msg__` 全仓 0 | 可删 |
| `frontend/src/gaea/styles.css:1460-1469` | `.msg--user .msg__body` 等（媒体查询内） | CSS 死类 | 10 | high | 同上 | 可删 |
| `frontend/src/gaea/styles.css:969-973` | `.empty-state__icon` | CSS 死类 | 5 | high | 整词 0（`.empty-state` 活） | 可删 |
| `frontend/src/gaea/styles.css:464-472` | `.md-code` | CSS 死类 | 9 | high | 整词 0（`.md` 活） | 可删 |
| `frontend/src/gaea/styles.css:1236-1245` | `.mem-suggestion`, `.mem-doc` | CSS 死类 | 12 | high | 整词 0 | 可删 |
| `frontend/src/gaea/styles.css:1427-1430` | `.pwa-titlebar` | CSS 死类 | 4 | high | 全仓仅此 1 处 | 可删 |
| `frontend/src/gaea/redesign.css:143-160` | `.msg--assistant/.msg--user/.msg__body` | CSS 死类（跨文件重复） | 18 | high | 同 styles.css 族 | 可删 |
| `frontend/src/index.css:74-146` | `.md-surface*`, `.md-elevation-1..5`, `.md-card`, `.md-ripple` | CSS 死类 | 77 | high | 整词 0；`--md-sys-*` 变量另说 | 可删 |
| `frontend/src/index.css:238-263` | `.void-card` | CSS 死类 | 28 | high | 整词 0 | 可删 |
| `frontend/src/index.css:270-273,602-643` | `.neon-card` | CSS 死类 | 50 | high | `neon-` 仅 `neon-glow-text` | 可删 |
| `frontend/src/index.css:438-449` | `.bento-grid` | CSS 死类 | 12 | high | `bento` 命中是数据字段 | 可删 |
| `frontend/src/index.css:571-581` | `.gp-panel` | CSS 死类 | 11 | high | 整词 0 | 可删 |
| `frontend/src/index.css:663-669` | `.gaea-brand-text` | CSS 死类 | 7 | high | 整词 0 | 可删 |
| `frontend/src/index.css:357-360` | `.page-transition-enter` | CSS 死类 | 4 | high | 整词 0 | 可删 |
| `frontend/src/chat-board.css:177-201` | `.chat-trust-bar/.chat-trust-track/.chat-trust-fill` | CSS 死类 | 25 | high | `chat-trust` 前缀全仓 0 | 可删 |
| `frontend/src/chat-board.css:773-792,1261-1268` | `.chat-drawer`, `.chat-drawer-toggle` | CSS 死类 | 29 | high | `chat-drawer` 前缀全仓 0 | 可删 |
| `frontend/src/gaea/components/memoryhub/hub.css:540-612` | `.hub-card`, `.hub-card-icon`, `.hub-badge`, `.hub-enter` | CSS 死类 | 56 | high | 47 个 `hub-*` token 全集不含 | 可删 |
| `frontend/src/gaea/tailwind.css:75` | `--font-size-base: 13px` | CSS 重复变量 | 1 | high | 与 `index.css:8` 14px 冲突、0 消费者 | 删 |
| `frontend/src/gaea/tailwind.css:76` | `--font-size-lg: 14px` | CSS 重复变量 | 1 | high | 与 `index.css:9` 16px 冲突，`index.css:167` 消费者 | 二者取一 |
| `frontend/src/schedule/AoaView.test.tsx:16-30` | `chainProject()` | D4 重复 | 15 | high | 6 份逐字相同 | 抽 helper（-75 行） |
| `frontend/src/schedule/baseline.test.ts:13-27` | `chainProject()` | D4 重复 | 15 | high | 同上 | 同上 |
| `frontend/src/schedule/BaselinesPanel.test.tsx:39-53` | `chainProject()` | D4 重复 | 15 | high | 同上 | 同上 |
| `frontend/src/schedule/deadline.test.ts:12-26` | `chainProject()` | D4 重复 | 15 | high | 同上 | 同上 |
| `frontend/src/schedule/GanttView.test.tsx:23-37` | `chainProject()` | D4 重复 | 15 | high | 同上 | 同上 |
| `frontend/src/schedule/TemplatesPanel.test.tsx:21-35` | `chainProject()` | D4 重复 | 15 | high | 同上 | 同上 |
| `frontend/src/schedule/aoa.test.ts:5,8` | `t()` / `l()` | D4 重复 | 7 | high | 4 份相同 | 抽 `scheduleFixtures` |
| `frontend/src/schedule/aoaLayout.test.ts:7,10` | `t()` / `l()` | D4 重复 | 7 | high | 同上 | 同上 |
| `frontend/src/schedule/cpm.test.ts:5,8` | `t()` / `l()` | D4 重复 | 7 | high | 同上 | 同上 |
| `frontend/src/schedule/pathDriver.test.ts:13,16` | `t()` / `l()` | D4 重复 | 7 | high | 同上 | 同上 |
| `frontend/src/pages/sin/SinIllustration.progress.test.tsx:25-32` | `vi.mock('../../gaea/lib/bridge')` | D4 重复 | 8 | high | 4 份逐字相同 | 抽 bridgeMock |
| `frontend/src/pages/sin/useSinCast.test.tsx:12-19` | 同上 | D4 重复 | 8 | high | 同上 | 同上 |
| `frontend/src/pages/sin/useSinNotes.test.tsx:10-17` | 同上 | D4 重复 | 8 | high | 同上 | 同上 |
| `frontend/src/pages/sin/useSinStory.test.tsx:23-30` | 同上 | D4 重复 | 8 | high | 同上 | 同上 |
| `frontend/src/components/novel/create/NewCharactersModal.test.tsx:26-46` | `stubRuntime()` | D4 重复 | 21 | high | 5 份同族 | 抽 helper（-65 行） |
| `frontend/src/components/novel/create/useChapterGateNotice.test.ts:13-34` | `stubRuntime()` | D4 重复 | 22 | high | 同上 | 同上 |
| `frontend/src/components/novel/create/useChapterStream.test.ts:14-35` | `stubRuntime()` | D4 重复 | 22 | high | 同上 | 同上 |
| `frontend/src/components/novel/BookSearchModal.test.tsx:39` | `stubRuntime()` | D4 重复 | ~20 | medium | 同名，体可能不同 | 同上 |
| `frontend/src/pages/sin/SinBookSourcePanel.test.tsx:42` | `stubRuntime()` | D4 重复 | ~20 | medium | 同名，体可能不同 | 同上 |
| `frontend/src/components/novel/ConvergeModal.test.tsx:12` | `new Proxy` 假 bridge | D4 重复 | 14 | high | 3 份相同 | 抽 helper |
| `frontend/src/components/novel/StorySpinePanel.test.tsx:15` | 同上 | D4 重复 | 14 | high | 同上 | 同上 |
| `frontend/src/components/novel/StyleFingerprintPanel.test.tsx:17` | 同上 | D4 重复 | 14 | high | 同上 | 同上 |
| `frontend/src/gaea/components/FilePreview.test.tsx` | pptx 文件条目 fixture | D4 重复 | 8 | medium | 3 文件同值 | 抽 `fixtures.ts` |
| `frontend/src/gaea/components/FilePreviewModal.test.tsx` | 同上 | D4 重复 | 8 | medium | 同上 | 同上 |
| `frontend/src/gaea/components/PptxEditPanel.test.tsx` | 同上 | D4 重复 | 8 | medium | 同上 | 同上 |
| `frontend/src/components/chat/ChatComposer.voiceDegraded.test.tsx:35` | `afterEach(cleanup)` | D5 冗余 teardown | 1 | high | `test/setup.ts:73-81` 已全局 `afterEach(cleanup)` | 删 |
| `frontend/src/gaea/components/composer/ComposerI18n.test.tsx:42,156,186,198` | `afterEach(cleanup)` ×4 | D5 冗余 teardown | 4 | high | 同上（同文件内重复 4 次） | 删 |
| `frontend/src/gaea/components/Composer.queue.test.tsx:39` | `afterEach(() => { cleanup(); vi.useRealTimers() })` | D5 冗余 teardown | 1 | high | `cleanup` 冗余，保留 `useRealTimers` | 去掉 cleanup |
| `frontend/src/gaea/hooks/useComposerAttachments.fail.test.tsx:43` | `afterEach(() => { cleanup(); vi.clearAllMocks() })` | D5 冗余 teardown | 1 | high | 同上 | 去掉 cleanup |
| `frontend/src/pages/CreatePage.test.tsx` | 950 行单文件 | D6 屎山 | 950 | high | 全仓唯一 >800 | 拆为 `CreatePage.{rail,editor,history}.test.tsx` |
| `frontend/src/gaea/components/DeliverablesPanel.test.tsx` | 788 行 | D6 屎山 | 788 | medium | 接近阈值 | 可选拆分 |
| `frontend/src/gaea/components/FileMenu.test.tsx:6` | 本地 `scrollIntoView` polyfill | D5 与 setup 重复 | 1 | medium | `test/setup.ts:69-71` 已全局 stub | 可删（需确认该行确是 polyfill） |
| `frontend/src/novel-workspace.css:511` | `.is-abandoned` | CSS 死类 | 1 | medium | 整词 0 + `is-${` 0 | 观察一轮再删 |
| `frontend/src/pages/weixin-page.css:226-227` | `.is-expired` | CSS 死类 | 2 | medium | 同上 | 观察一轮再删 |
| `frontend/src/pages/sin/sin.css:714,729,740` | `.is-failed` | CSS 死类 | 3 | medium | 同上，但状态名易复活 | 保留 |
| `frontend/src/gaea/styles.css:1319-1348` | 活类名无规则：`drawer--resizing` | 反向发现 | — | medium | `ResizableDrawer.tsx:175` 用，CSS 无对应 | 补规则或删用法 |
| `frontend/known-flaky.txt:7-10` | 过期用例数注释 | 文档漂移 | 4 | high | 写 2799/2799，现基线 3760 | 更新 |

---

## Go 测试样板重复度总结（统计性）

| 指标 | 数值 |
|---|---|
| `internal/**/*_test.go` 文件数 | **917** |
| Go 测试总行数 | **145,722** |
| 平均 | 159 行/文件 |
| `t.Skip` / `t.Skipf` 调用 | **67 处**，分布在约 40 个文件 |
| `t.TempDir()` 使用 | **1,451 处** |
| `func newTest*(t *testing.T)` 声明 | 24 处（跨包） |
| `func setup*(t *testing.T)` 声明 | 2 处 |
| `func assert*(t *testing.T, …)` 声明 | 10 处 |
| `func mustWrite*(t *testing.T, …)` 声明 | 6 处 |

**最大 5 个 Go 测试文件**：`internal/gaea/tasks/tasks_test.go`(1118)、`internal/modelengine/engine_test.go`(1020)、`internal/gaea/browser/manager_test.go`(819)、`internal/booksource/booksource_test.go`(770)、`internal/config/config_test.go`(769)。

**样板重复度评价：低～中，不建议大规模重构。**

理由与典型例子：

1. **`newTestStore` 名字出现 8 次，但不是复制粘贴。** 分布在 `internal/characterlib`、`internal/chat`、`internal/gaea/costimport`、`internal/gaea/costinquiry`、`internal/gaea/knowledgeimport`、`internal/gaea/pricefeed`、`internal/gaea/semantic`、`internal/auth` —— 每个返回**各自包内不同的 `*Store` 类型**，只是命名收敛。抽共享 helper 会引入跨包依赖，**得不偿失**。
2. **`t.TempDir()` 1,451 次是 Go 标准库正确用法**（自动清理），不是样板膨胀。
3. **`t.Skip` 67 处全部是有理由的环境门禁**，不是僵尸测试。分类：外部工具缺失（`soffice`/`pdftoppm`/`matplotlib`/`git`/`bash`/`sandbox-exec`）约 60%；平台专属（`仅 Windows` / `darwin-only` / `symlink 需特权`）约 30%；显式 opt-in 环境变量（`GAEA_SMOKE_*`、`*_LIVE=1`、`GAEA_SELFGEN`、`HERDSMAN_LIVE`）约 10%。**没有一处是「跑了也不可信」的逃逸**，反而每处都写了为什么不能直接修。
4. **真正可提取的样板集中在同包内**，3 个典型例子：
   - `internal/app/` 下 `mustWriteChapter`(`novel_fingerprint_handler_test.go:26`)、`mustWriteAiTasteChapter`(:230)、`mustWritePartialChapter`(`novel_rewrite_handler_test.go:198`)、`mustWriteOutlines`(`novel_search_handler_test.go:28`)、`mustWriteCharacterWithPortrait`(`scene_illustration_v2_test.go:37`) —— **5 个同包 `mustWrite*` fixture 写入器**，形态一致（取 `*App` + 参数 + 建目录 + 写文件），可合并为一个 `mustWriteFixture(t, a, kind, args...)`。可回收约 40-60 行。
   - `internal/app/` 下 `newTestApp`/`newTestCore`/`newTestTaskApp`/`newTestAppWithProject`(`feature_model_handler_test.go:13`、`gaea_tasks_test.go:31`、`novel_bookcover_test.go:43`) —— **App 构造样板**，建议统一为 `newTestApp(t, opts...)`。
   - `internal/gaea/{command,memory}` 各有 `func mustWrite(t, path, body string)`（`command/symlink_test.go:47`、`memory/memory_test.go:135`）—— 跨包同名同体，可下沉到 `internal/testutil`。
5. **`internal/gaea/sandbox/seatbelt_darwin_test.go` 有 8 处 `t.Skip*`**（176,180,184,189,232,235,239,243）—— 该文件在非 darwin 平台整体无效，建议文件级 `t.Skip` 或加 `//go:build darwin` 约束，比 8 处逐用例 skip 更清晰。

**结论**：Go 测试样板重复度属于**健康区间**，唯二值得做的是「同包 `mustWrite*`/`newTestApp*` 收敛」和「seatbelt 文件级构建约束」，均为 P2，且**本次审计不逐符号报**（按要求）。

---

## 不建议动的地方

1. **`frontend/src/gaea/hljs-theme.css` 全部 `hljs-*` 规则（29 个类）** —— highlight.js v11 运行时按语法生成这些 span class，代码里搜不到是**正常的**。删了会静默丢失代码高亮配色，且 `stores/appStore.contrast.test.ts:121` 有对比度机检会红。CSS 里 `function_` / `class_` 同理（是 `hljs-title.function_` / `hljs-title.class_` 复合选择器的第二段）。
2. **`frontend/src/gaea/components/changesdiff-tok.css` 全部 `tok-*` 规则（13 个类）** —— 由 `@lezer/highlight` 的 `classHighlighter` 生成，文件头注释已自述；`appStore.contrast.test.ts:103` 有「diff 与 hljs 共用一套色」的机检。
3. **`frontend/src/index.css` / `character-page.css` / `novel-workspace.css` / `character-library.css` / `imagegen.css` 里的 `ant-*` 与 `anticon`（46 个类）** —— antd 运行时注入，删了 UI 立刻崩样式。
4. **`frontend/src/schedule/schedule.css:739-775` 的 `.sched-group-c0..c5`** —— 由 `` `sched-group-c${d.synSeq % 6}` ``（`gantt/GanttBars.tsx:69`）生成。**这是本次审计里最经典的动态拼接反例**，任何「类名字面 grep」的方法都会误判它死。
5. **`frontend/src/genui/styles.css` 的 `gui-text-*` / `gui-tone-*` / `gui-btn-*`** —— `genui/renderNode.tsx:60,135,581` 用模板串拼，同理不可判死。
6. **`frontend/src/gaea/styles.css` 的 `.md`(383-464, 127 行)、`.layout`(197-210)、`.reasoning*`(632-725)、`.transcript`、`.notice`、`.phase`、`.badge*`、`.palette*`、`.ds-chip*`、`.drawer-backdrop`** —— 全部有真实 className 用法。特别是 `.layout`：`gaea/App.tsx:626-636` 用**数组字面量**拼串，朴素 grep 极易误判；`.md` 在 `Markdown.tsx:558` / `MemoMarkdown.tsx:101`。
7. **`--ds-*` / `--md-sys-*` / `--hl-*` / `--read-*` / `--sched-*` / `--v3-*` 等自定义属性的「重复定义」** —— 除 `--font-size-base`/`--font-size-lg` 两条外，其余 26 条都是**明暗主题 / 预设作用域覆盖**，是设计机制而非重复。统一它们会破坏主题切换。
8. **CSS 70 组重复声明块** —— 是 3-5 条 Flex 惯用式，抽类会造成跨板块耦合，**收益为负**。
9. **`internal/**` Go 测试里的 `newTestStore`(×8) / `t.TempDir()`(×1451) / `t.Skip`(×67)** —— 命名收敛与标准用法，不是坏味道。
10. **32 处 `expect(screen.getByText(x)).toBeTruthy()`** —— `getBy*` 本身失败即抛，`toBeTruthy` 冗余但**无害且有效**，不值得开 PR。
11. **`frontend/src/gaea/components/{SubagentsPanel,MorningBriefCard}.tsx` 及其测试、`TrendChart.tsx`** —— 已知死代码，由另一路审计负责，本次不复核、不动。
12. **`frontend/src/test/setup.ts`** —— 5 个 polyfill 都写了「为什么不能删」的理由（`localStorage.clear` 缺失、antd `matchMedia`、`ResizeObserver`、`scrollIntoView`、**以及「不要在 afterEach 里清 body，否则 14 例 toast 假红」**）。这份注释质量很高，是本仓测试基建里最该保留的部分。

---

### 附：复核用的可复现命令（只读）

```powershell
# 跳过/聚焦用例（应为 0）
Get-ChildItem frontend\src -Recurse -Include *.test.ts,*.test.tsx |
  Select-String -Pattern '(describe|it|test)\.(skip|todo|only)\b|\bxit\(|\bxdescribe\(|\bfit\('

# gaea/styles.css 关键死类（应全部 0）
Get-ChildItem frontend\src -Recurse -Include *.ts,*.tsx,*.html |
  Select-String -Pattern 'context-menu|msg--|msg__|ds-button-pill|ds-modal|drawer__|className="card"|data-tooltip'

# 动态拼接反例自检（应命中 schedule/gantt/GanttBars.tsx:69）
Get-ChildItem frontend\src -Recurse -Include *.tsx | Select-String -Pattern 'sched-group-c\$\{'

# 唯一死 mock
Get-ChildItem frontend\src -Recurse -Include *.test.ts,*.test.tsx |
  Select-String -Pattern "vi\.mock\('\.\./\.\./src/wailsjsCompat'"
```
