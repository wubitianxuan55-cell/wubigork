# codex → gaea 前端蒸馏总清单（批次 F，只出文档、不改代码）

> **边界（本次硬约束）**：仓库 `C:\AI\wubigrok` 只读。本批次产物全部是 `docs/codex-distill/` 下的文档，
> **不修改任何 gaea 源码、配置、测试、依赖**。任何"落点/施工卡"都是计划，等拍板后另开批次执行。
>
> **证据根**
> - codex 0.165.0 上游源码（tag `rust-v0.165.0`，commit `e374836b`）：`C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs\tui\src`（1140 个 `.rs`，46.2 万行）
> - 二进制旁证：`C:\AI\re\codex-0.165.0\evidence\crate-map.txt`（`tui` crate = 371 个文件路径，是唯一前端 crate）
> - gaea 前端：`C:\AI\wubigrok\frontend\src`（1095 个 ts/tsx，24.2 万行，不含测试）

---

## 1. codex 前端分层 → gaea 前端映射

codex 的 TUI 前端是一个 crate 内分层（没有独立的 render crate），分层在模块边界上：

| codex 层 | 职责 | gaea 对应 | 差异要点 |
|---|---|---|---|
| `app.rs` / `app_event.rs` / `app/event_dispatch.rs` | 状态机 + 内部事件总线（widget 发事件、app 层执行动作） | `pages/ChatPage.tsx`（编排层）+ `stores/appStore.ts` + `gaea/lib/bridge` + Wails events | gaea 用 store + 事件桥，没有显式"AppEvent 总线"层 |
| `chatwidget.rs` + `chatwidget/*` | 会话 UI 聚合（渲染、流式、slash、工具请求、状态面） | `components/chat/*`（MessageList/ChatRow/ChatComposer/ChatInspector） | codex 是单结构体 + 子模块；gaea 是组件树 + memo 边界 |
| `bottom_pane/*` | 输入面板 + 视图栈 + 弹层（审批/选择/追问） | `components/chat/ChatComposer.tsx`、`gaea/components/{Composer,ApprovalModal,CommandPalette,ArgMenu,AskCard}.tsx` | gaea 弹层各自独立，缺统一视图栈 |
| `transcript_view.rs` + `transcript_view/*` | 已提交历史的视图（回折、搜索、折叠、选区、copy mode、follow） | `components/chat/MessageList.tsx`（尾部窗口 + content-visibility）、`useFollowScroll`、`SearchPanel` | gaea 的滚动宿主在 ChatPage（`ChatPage.tsx:211`） |
| `history_cell/*` | 提交后的单元格（消息/补丁/审批/搜索/警告…） | `components/chat/ChatRow.tsx`（memo 单行） | codex 一类一个文件；gaea 单组件多分支 |
| `streaming/*` + `markdown_stream.rs` | 流式双区（stable/tail）+ 提交动画 + 表格 holdback | `hooks/useThrottledStreamText.ts`、`gaea/lib/markdownCut.ts`（`findStableCut`）、ChatRow 流式分支 | **gaea 已有稳定切点，缺"提交队列/不变量/表格 holdback"** |
| `render/highlight.rs` | 语法高亮（流式增量 + 主题 revision） | `components/ChatCodeBlock.tsx`、`gaea/lib/codeHighlight.ts` | — |
| `markdown_render.rs` + `markdown_render/*` | markdown → 行（表格/数学/mermaid/链接/指令） | `components/{ChatMarkdown,MarkdownContent}.tsx` + remark/rehype 插件 + `gaea/lib/{mermaidRender,mathText,tableData}.ts` | gaea 走 react-markdown 生态，codex 是手写解析 |
| `diff_render.rs` | 补丁渲染（折叠/截断/行号） | `gaea/lib/{diff,diffRender,diffHighlight}.ts` + `ChangesDiff` | gaea 已有 xlsx/pptx/docx 专用 diff |
| `keymap*.rs` / `shortcut_*` | 键位表 + 和弦 + 帮助浮层 | `wenxue/app/useAppKeyboard.ts` + `gaea/app/useAppKeyboard.ts`（**两份手写 if 链**）、`layouts/MainLayout.tsx:257-347`（CommandRail 内联，无独立文件）、`schedule/commandRegistry.ts` | 无键位表/无上下文分区/无多键序列/无冲突检测/无帮助浮层同源；`gaea/lib/spaceBindings.ts` 是 Wails 绑定归属表，与键位无关（命名撞车） |
| `custom_terminal.rs` / `insert_history.rs` / `tui/*` | 终端输出层（scrollback 插入、帧请求、尺寸监听） | 浏览器/React（DOM 追加 + 滚动锚定 + rAF） | **整层不搬** |
| `status/*` | `/status` 字段卡（模型/目录/权限/token/限额） | `gaea/components/AppStatus.tsx`、`ContextPill`、模型中心 | gaea 已有"单一真相"纪律 |

**结论**：codex 前端值得抄的是**四类"状态机 + 不变量"**（流式双区、转录视图交互、视图栈/弹层优先级、键位表），
而不是它的渲染代码——终端与 DOM 的渲染模型不可比。

---

## 2. 资产索引（本批 7 篇）

| 编号 | 文件 | 主题 | 主要判定 |
|---|---|---|---|
| 27 | `asset-27-streaming-render-two-region.md` | 流式渲染双区（stable/tail、提交动画、表格 holdback、围栏快路径） | **最高价值**，机制可直接照搬 |
| 28 | `asset-28-transcript-view-interaction.md` | 转录视图（reflow、搜索、折叠、选区/copy、follow、书签、分页浮层） | 高价值，多数为交互补齐 |
| 29 | `asset-29-markdown-render-and-copy.md` | Markdown（源表、键值表、行内指令、链接、列表间距、数学、mermaid、复制） | 中高，改造为插件 |
| 30 | `asset-30-diff-render.md` | Diff 渲染（行模型、折叠、截断、活动分组） | 中，gaea 已有大半 |
| 31 | `asset-31-bottom-pane-and-overlays.md` | 输入面板 + 视图栈 + 审批/追问/选择弹层 | 高，缺"弹层优先级裁决" |
| 32 | `asset-32-keymap-system.md` | 键位系统（表结构、和弦、冲突、帮助浮层同源） | 中，缺多键序列与同源帮助 |
| 33 | `asset-33-status-motion-clipboard.md` | 状态卡字段表 / 动效无障碍 / 剪贴板双表示 | 低（判定为主），3 项值得动 |

每篇结构统一：**机制事实（file:line）→ gaea 现状核对（file:line）→ 三档判定 → 施工卡（落点/验收/开关/回退）→ 风险 → 证据自查**。

---

## 2.5 各篇核心结论（一句话）

| 篇 | 核心结论 |
|---|---|
| 27 流式 | 本质是**一条不变量 + 四个守门人**（换行门控 / 表格三态 holdback / 围栏快路径 / 8192B 预览上限），不是"节流器"；gaea 已有 `findStableCut` 与节流，缺的是不变量、表格定形与高亮闸门 |
| 28 转录 | 阅读位置必须是**坐标系**（`Anchor{EntryKey, index, offset, row_bias}`）而不是像素；gaea 的缺口不在滚动跟随，而在「行级」这一层整个不存在 |
| 29 Markdown | 表格复制产出的不是 TSV 而是 **GFM 源表**（`\| a \| b \|` + 按 alignment 拼分隔行）；必须搬的是两类上限（高亮 512KB/10000 行/单行 4KiB、URL 8KB），不要搬终端制表符与键值表阈值 |
| 30 Diff | codex **没有中段折叠**（折叠只在卡片层 3 行），diff 正文走**硬换行不截断**、选区取逻辑源；gaea 的三处阈值互相架空（240 < 20000 < 600） |
| 31 输入面板 | 「谁能收键」是一条**显式阶梯**（warnings → questions → 栈顶 → banner → composer），提示与路由同源；审批**默认焦点 ≠ 默认执行**，队列是 LIFO 需倒序 drain 还原 FIFO |
| 32 键位 | 键位 = **上下文位集 × (context, action) 身份 × 单键表 + 独立和弦层**，冲突全在加载期报错；gaea 是两份手写 if 链，且 `CommandRail` 已内联进 `MainLayout`（无独立文件） |
| 33 状态/动效/剪贴板 | 状态卡是字段表（标签列宽 + 续行缩进）；动效偏好启动播种 + JS 显式降级；剪贴板一次写入 **text/plain + text/html 双表示** |

## 2.6 跨篇收敛：真正的缺口只有 6 个

把七篇的"缺口"去重后，落到 6 条（其余都是它们的派生）：

| # | 缺口 | 出处 | 性质 |
|---|---|---|---|
| 1 | 审批模态没有键盘陷阱；硬确认卡「3=终止」按键被守卫吞掉 | F31（已复核，见 §6.5） | **正确性 bug** |
| 2 | 阅读位置用像素不用坐标系（`scrollTop` + 250ms `setTimeout` 还原） | F28 | 结构性，是 F28 全部缺口的根因 |
| 3 | 稳定切点**可回退**（围栏内回退到围栏前），使"稳定段"字符串仍会变 | F27 | 正确性+性能 |
| 4 | 行级能力缺失：行级搜索、块级/字符级复制、折叠态跨 consolidation 迁移 | F28 | 功能缺失 |
| 5 | 复制能力缺失：表格→GFM 源表、剪贴板富文本双表示 | F29 + F33 | 功能缺失 |
| 6 | 高亮与渲染**无统一闸门**：无 512KB/10000 行/单行 4KiB 上限、无块级缓存、三处阈值互相架空 | F27 + F29 + F30 | 性能/一致性 |

**建议优先级**（按"错得离谱"而非"改得容易"排）：① F31 两个按键 bug → ② F28 坐标系 → ③ F27 不变量与表格 holdback → ④ F29+F33 复制链 → ⑤ F30 统一闸门 → ⑥ F32 键位表 → ⑦ F33 其余（状态字段表/动效订阅）。

---


```
波 0（无依赖，纯函数/纯组件）
  F27 流式双区 ──┐
  F29 markdown ──┤
  F33 状态/剪贴板 ┘     （F27 的 stable/tail 是 F28 搜索与 F31 提交排队的地基）

波 1（依赖 F27 的稳定切点契约）
  F28 转录视图（搜索/折叠/选区/copy）
  F31 输入面板视图栈（弹层优先级 + 排队输入）

波 2（独立，但改动面最大）
  F30 diff（gaea 已有 diff 库，属"补退让与折叠口径"）
  F32 键位（跨页面，需先定"键位表单一来源"再动入口）
```

**排序理由**：F27/F29 是"纯算法 + 组件内部"，改动不外溢；F28/F31 会动滚动宿主与全局键盘，必须先有稳定切点与弹层栈的契约；F32 最后，因为它要求所有入口先收敛到同名动作。

---

## 4. 与既有批次的去重（重要）

| 已有 | 本次相邻项 | 处理 |
|---|---|---|
| 第一批 T1–T5（后端：三档行匹配/环境注入/拒绝理由/Never 语义/schema 预算） | 本批全部前端 | **零重叠**，但 T5 的"字节预算"思想与 F29 的超长行/高亮上限同源，可共用常量命名 |
| 第二批 T6 碎片契约 / T7 章节化 / T9 压缩 | F27 的"稳定区"概念 | 不同层：T6/T7 是**上下文**装配，F27 是**渲染**分区；命名需区分（`stable region` vs `context section`） |
| T13 执行语义（双缓冲/双闸门） | F31 排队输入 + F27 提交队列 | 概念同构（pending vs transcript），实现独立，不要互引代码 |
| `docs/gaea-deepseek-chatdemo-distill-2026-11.md`（前端 §2.4 等） | F29 markdown | 已在 gaea 落地（`remarkDisableCodeIndented`），本批只补"源表/复制/指令" |
| `docs/gaea-contextview-upgrade-research-2026-11.md` | F28 转录视图 | 一个是上下文视图，一个是消息流，别混 |

---

## 5. 门禁（拍板落地时才跑；本次不跑）

```powershell
cd frontend
npm.cmd run build      # tsc -b + vite build
npm.cmd run lint       # eslint .
npm.cmd test           # vitest run
```

前端已有性能守卫 `frontend/src/perf-guards.test.ts`：任何"新增订阅/新增 rAF/新增全局监听"的改动必须在该文件加一条断言，否则视为回归。

---

## 6. 全局风险（跨篇）

1. **两套壳**：主聊天走 `pages/ChatPage.tsx`；工作台走 `gaea/App.tsx`（`pages/GaeaPage.tsx:1` 包装）。同一机制若两处消费，**纯函数放 `frontend/src/gaea/lib/` 单源**，两壳只做接线，否则必然漂移（`gaea/` 与 `wenxue/` 已出现 `gsapAnimations.ts`、`sanitize.ts` 双份拷贝，`wenxue/lib/` 是既有前例）。
2. **CSS 令牌与 JS 动画双轨**：gaea 已有 CSS 变量体系（`--dur-fast/base/slow`）+ GSAP 常量镜像（`gsapAnimations.ts:5-12`）。新增动效必须两边同步，否则 reduced-motion 下行为不一致。
3. **流式路径最敏感**：MessageList 的尾部窗口 + content-visibility + 吸底跟随是历史踩坑区（`MessageList.tsx:1-30` 注释记录了为何不用 react-window）。F27 的任何改动都要先在流式行上做 60fps 录屏比对。
4. **诚实反馈纪律**：`gaea/lib/clipboard.ts:1-9` 记录了 v4.513"假已复制"事故。F33 的富文本复制必须保持"只在真正写入后翻转成功态"。
5. **不要照搬终端常量**：`80/120 列`、`8_333_334ns`、`50ms 帧` 这些是终端语义；Web 侧只保留"上限/不变量"的语义，数值按浏览器实测重定。

---

## 6.5 核对中发现的 gaea 现存缺陷（只记录，本次不改代码）

对话式核对（asset-31）时逐行读 `gaea/components/ApprovalModal.tsx` 发现两处**与 codex 语义相反**的行为，已在本次复核确认：

| 缺陷 | 证据 | 后果 |
|---|---|---|
| 模态期间没有键盘陷阱 | `ApprovalModal.tsx:78-81`：`event.target` 是 `INPUT`/`TEXTAREA`/`contentEditable` 时**直接 return**，整张卡的快捷键与 Esc 全部失效 | 审批卡开着时点回编辑器打字/回车，仍能继续输入并提交——审批形同建议 |
| 硬确认卡的「3 = 终止」按键无效 | `ApprovalModal.tsx:172` 按 `hardConfirm ? 3 : 5` 显示序号；但 `KEY_DECISIONS["3"]` 是 `allow_session`（`:40-46`），而 `:89` 在 `hardConfirm` 下拦截 `allow_session`/`persist_allow` | 按钮写着「3」，按 3 什么也不发生；abort 只剩 `Esc → deny` 一条路，且 `deny ≠ abort`（deny 继续、abort 终止本轮） |

两条都是「提示与行为不一致」，属于比性能更该先修的类型；修法在 asset-31 的施工卡里（`resolveKeyOwner()` + 显式 `hardConfirm` 键位派生），**本次批次不动实现**。

其余缺口（弹层互斥、历史搜索快照、列表分页/tab、`CommandPalette` 双份拷贝）见 asset-31 的「gaea 现状核对」一节，行号在那里给全。

---

## 7. 回报格式（落地批次用）

```
F<n> <任务名>
基线：<测试文件> N 例 / 既有失败 M 例
改动：<file:line> 一句话
门禁：build / lint / test 结果
开关：默认 <关/开>；关闭时与现状逐字节等价：是/否
性能：流式行 60fps 录屏比对（改前/改后）
风险：<是否触碰 ChatPage 滚动宿主 / 是否新增全局监听>
```
