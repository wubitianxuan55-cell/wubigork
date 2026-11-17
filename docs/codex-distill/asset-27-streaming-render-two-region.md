# 资产 27 · 流式渲染施工级：稳定区/尾区双区模型 + 提交动画队列

> 源码（tag `rust-v0.165.0`）：`codex-rs/tui/src/streaming/{controller.rs,commit_tick.rs,chunking.rs,prose_preview.rs,render.rs,code_fence.rs,table_holdback.rs,mod.rs}`、
> `markdown_stream.rs`、`render/highlight_streaming.rs`、`markdown_render/streaming.rs`、`table_detect.rs`、
> `history_cell/markdown_render_cache.rs`。
> 对照（只读）：gaea `frontend/src`（`hooks/useThrottledStreamText.ts`、`hooks/useChatStream.ts`、
> `gaea/lib/markdownCut.ts`、`components/chat/{MessageList,ChatRow}.tsx`、`components/{MarkdownContent,ChatCodeBlock}.tsx`、`gaea/components/HlCode.tsx`）。
> 常量证据：`COMMIT_ANIMATION_TICK = 8_333_334 ns`（≈8.33 ms）、`MAX_PREVIEW_BYTES = 8192`、
> `MAX_HIGHLIGHT_LINE_BYTES = 4*1024`、`MAX_HIGHLIGHT_BYTES = 512*1024`、`MAX_HIGHLIGHT_LINES = 10_000`。

---

## 一、双区模型：一条不变量定生死

```rust
/// - `emitted_stable_len <= enqueued_stable_len <= render.lines.len()`.
/// - committed source is append-only until `reset()`; never modified mid-stream.
/// - The committed-source tail starts at `enqueued_stable_len`.
```
（`streaming/controller.rs:38`、`controller.rs:40`）

- **稳定区**＝已提交进 scrollback（`emitted_stable_len`）；**尾区**＝活跃 cell 里可变的 `render.lines[enqueued_stable_len..]` + 预览行（`controller.rs:272-277`）。
- 三层计数语义分开：`enqueued` 是"已排队待动画"，`emitted` 是"已真的进 scrollback"（`controller.rs:94-97`）。**尾区必须从 `enqueued` 起算而不是 `emitted`**——否则"已排队未发射"的行会同时出现在活跃 cell 与队列里，屏幕上重复（`controller.rs:266-270`）。
- 只有**换行终止的源**才允许进稳定渲染；未终止的半行既不进队也不进尾，由预览兜（`controller.rs:144-149`）。门控在收集器里：`commit_complete_source()` 取 `rfind('\n')+1`（`markdown_stream.rs:87-96`）。
- 每个 delta 后：`render.append(committed_source)` → `holdback_scanner.scan(...)` → `sync_stable_queue()`；delta 含 `\n` 时先 `preview = ProsePreview::default()`（`controller.rs:150-179`）。
- 目标稳定长度 = `render.lines.len() - tail_budget`，且**不小于** `emitted_stable_len`（`controller.rs:450-457`）。`sync_stable_queue` 里若目标**回退**（结构重写），会从 `emitted..target` 重新入队（`controller.rs:461-483`）。

## 二、提交动画队列：tick 常量与两档齿轮

| 事实 | 值 | 证据 |
|---|---|---|
| tick 周期 | `8_333_334 ns`（120 fps 上限，非 16.6 ms） | `tui/frame_rate_limiter.rs:13`、`app.rs:460` |
| tick 调度 | 外部 interval，不是本模块自己排程 | `app/event_dispatch.rs:987-991`、`commit_tick.rs:7-9` |
| Smooth | 每 tick 走 **1 行**（`DrainPlan::Single`） | `chunking.rs:201`、`commit_tick.rs:185` |
| CatchUp | `Batch(queued_lines.max(1))` 全清 | `chunking.rs:202` |

队列压力阈值（全部**两信号之一即触发进档**，退档要**两者同时低于**）：

- 进档：`queued_lines >= 8` 行 **或** `oldest_age >= 120 ms`（`chunking.rs:85`、`chunking.rs:90`、`chunking.rs:267-272`）
- 退档起点：`queued_lines <= 2` 行 **且** `oldest_age <= 40 ms`（`chunking.rs:95`、`chunking.rs:100`）
- 退档保持：`EXIT_HOLD = 250 ms`（`chunking.rs:103`、`chunking.rs:240-243`）
- 退档后冷却：`REENTER_CATCH_UP_HOLD = 250 ms`（`chunking.rs:108`、`chunking.rs:258-261`）
- 严重积压绕过冷却：`queued_lines >= 64` 行 **或** `oldest_age >= 300 ms`（`chunking.rs:113`、`chunking.rs:116`）
- 队列为空 → 直接回 Smooth（`chunking.rs:181-189`）

快照是两个控制器**深度求和、年龄取 max**（`commit_tick.rs:105-117`）；`CommitTickScope::CatchUpOnly` 只在 CatchUp 档才真的发射，否则返回空 output（`commit_tick.rs:31-36`、`commit_tick.rs:82-84`）。模式切换有 `tracing::trace!` 打点（`commit_tick.rs:131-140`）。

## 三、表格 holdback：三态 + 保守扫描

```rust
enum TableHoldbackState { None, PendingHeader { header_start }, Confirmed { table_start } }
```
（`table_holdback.rs:24-33`）

- **为什么必须 holdback**：表格非增量，加一行会改每列宽度并重塑所有已渲染行（`controller.rs:14-18`）。`Confirmed` 时"从表头起"整块当尾；`PendingHeader` 只把"疑似表头那行起"当尾，前面正文继续流（`controller.rs:502-509`）。
- **保守扫描**只说"尾从哪开始"，不校验整表、不预测布局（`table_holdback.rs:7-11`）。一行是表头＝有未转义 `|` 切出的段且至少一格非空（`table_detect.rs:88-90`）；分隔行＝**每段**都是 `:?-{3,}:?`（至少 3 个 `-`）（`table_detect.rs:95-111`）；无外层 `|` 时必须切出 >1 段才算（`table_detect.rs:48-50`）。
- 确认条件＝**上一行是表头 + 本行是分隔行**，且两侧 `FenceKind` 都不是 `Other`（非 markdown 围栏内的 `|` 是代码）（`table_holdback.rs:155-167`）。表格渲染被关掉时**不进 Confirmed**（`table_holdback.rs:161-164`）。
- 退档/重扫：`source_offset < completed_source_len` 或 holdback 起点已落到完成区之前 → 整机 `reset()` 并从新完成长度重扫（`table_holdback.rs:112-118`）。空行结束 `in_table`（`table_holdback.rs:152-154`）。
- 字节偏移 → 渲染行数的**唯一桥**是 `tail_budget_from_source_start`（`controller.rs:528-540`）；前缀行数按 `(source_start, width)` 缓存，避免密集表格每行重渲前缀（`controller.rs:547-584`）。
- 尾区边界是四个来源取 **min**：表起点、`mutable_fence_start`、`pending_math_start`、裸列表标记行（`controller.rs:417-447`）。

## 四、代码围栏快路径：OpenCodeFence 何时复活 highlighter

- `OpenCodeFence` 只认"一个开着的、顶层的、带语言标签的围栏"，且源必须与 pulldown-cmark 的 code text 完全一致（`code_fence.rs:12-22`）。
- `detect` 的拒绝条件（任一命中即回 CommonMark 规范路径）：marker 不是 `` ` ``/`~`、marker 长度 `< 3`、源未以 `\n` 结尾、主题 revision 变了、含 `\r`/`\0`、info 串含 `&`/`\`（或反引号围栏里含反引号）、语言 token 长度 `> MAX_HIGHLIGHT_LINE_BYTES`、以及**任何可能关闭围栏的行**（`code_fence.rs:33-76`）。
- **highlighter 是懒复活**：`highlighter: None`，注释写明"Initialized only if another chunk arrives, avoiding duplicate work on one-shot fences"（`code_fence.rs:20-21`、`code_fence.rs:95-102`）。即一次性围栏永不建状态。
- `append` 的守门：`source_len + committed.len == raw.len`（严格追加）、新块是 raw 的后缀、无 `\r`/`\0`、无潜在关闭行、主题 revision 未变；任一不满足返回 `None` **并消费掉保留态**，要求整围栏重渲（`code_fence.rs:87-94`）。
- 关闭行判定故意**假阳性**（`trim_start` 后连续 marker ≥ 开启长度即算），因为"宁可多回落"也不复制 CommonMark 关闭语法（`code_fence.rs:128-136`）。
- 增量高亮本体：`new()` 用 `HighlightLines` 重放已渲染前缀取回 `(HighlightState, ParseState)`；`append()` 从状态续跑并累计字节/行数，跨限即返回 `None`（要求整围栏重渲、同时清掉旧着色）（`highlight_streaming.rs:36-67`、`highlight_streaming.rs:74-114`）。
- 限额：`MAX_HIGHLIGHT_BYTES = 512*1024`、`MAX_HIGHLIGHT_LINES = 10_000`、`MAX_HIGHLIGHT_LINE_BYTES = 4*1024`，判定为 `>` 严格大于（`render/highlight.rs:640`、`render/highlight.rs:643`、`render/highlight.rs:646`、`render/highlight.rs:653-654`）。超限或语言未知 → `state: None` 的**永久纯文本回退**（`highlight_streaming.rs:16-19`、`highlight_streaming.rs:52-54`）。

## 五、prose preview：有界、可丢弃、不推进稳定边界

- `MAX_PREVIEW_BYTES = 8192`；窗口从末尾向前取，用 `ceil_char_boundary` 保证 UTF-8 边界（`prose_preview.rs:17`、`prose_preview.rs:82`）；裁剪过就在首行插 `…`（`prose_preview.rs:117-119`）。
- 只扫**新增字节**（`source[self.scanned_len..]`），所以超长单行不重复扫描（`prose_preview.rs:44-45`）。
- 停止推进的条件：见过 `|`、行首是空格/tab/`>`、行首是三反引号/三波浪线、或整行就是 1~2 个反引号/波浪线——即"可能是嵌套块结构"时保留原文换行门控不改（`prose_preview.rs:48-63`）。数学模式例外：`PreviewMode::Math` 总是推进（`prose_preview.rs:48`）。
- Rich 模式额外扣留未闭合链接目标与 `:codex-followup` 指令：`rich_safe_len` 取各起点最小值（`prose_preview.rs:65-75`），滚动窗口**不得**从指令提示语内部开始（`prose_preview.rs:98-103`）。
- 出结果与旧行 `lines_with_sources_eq` 相等就返回 `false`（不触发重排）（`prose_preview.rs:120-124`）。预览行永不进 scrollback（`prose_preview.rs:1-2`）。

## 六、chunking 与 resize 重渲染策略

- 渲染器只在**最后一个顶层块**可变：`stable_source_len`/`stable_rendered_len` 双边界，其余块渲染一次即留（`render.rs:20-41`）。追加时 `truncate(stable_rendered_len)` 再拼尾（`render.rs:233-244`）。
- 整源重渲（`recompute`）只在三种情形：width/render_mode 变化、出现引用式链接定义、出现 inline visualization 指令（`render.rs:71-128`、`render.rs:153-174`）；重渲后两个 stable 边界归零（`render.rs:126-128`）。
- 顶层块起点由 `last_top_level_block_start` 给出，且被两道过滤掉（落在未闭合数学/display math 区间内则不给）（`markdown_render/streaming.rs:26`、`markdown_render/streaming.rs:95-103`）。
- resize（`set_width`）：同宽直接返回；否则重渲 → `refresh_preview` → 若有挂起队列且已发射到底，**故意留 1 行不发射**，防止新宽度下换行压缩把待发内容永久跳过（`controller.rs:311-319`）→ `clear_queue()` → 若"无挂起队列且无活跃尾"，直接把 `emitted=enqueued=lines.len()` 防重放（`controller.rs:321-328`）→ 否则 `rebuild_stable_queue_from_render()`。
- 重渲还要**按前缀行数比例对齐 emitted**：`emitted = prefix_len(new) + (emitted - prefix_len(old))`，不足则取 `min`（`controller.rs:394-414`）。这就是 codex 明说的"不做字节级 remap，只保留 split"（`controller.rs:290-293`）。

## 七、markdown_render_cache 失效条件

缓存键是六元组：`width(u16)`、`list_spacing`、`syntax_theme_revision`、`terminal_fg`、`terminal_bg`、`color_level`；**只保留最近一条**，未命中就整体替换（`history_cell/markdown_render_cache.rs:13-21`、`markdown_render_cache.rs:24-51`）。锁中毒走 `PoisonError::into_inner` 继续用（`markdown_render_cache.rs:42`）。

---

## 八、gaea 现状核对：已有等价物与真实缺口

| codex 机制 | gaea 等价物（file:line） | 差距 |
|---|---|---|
| 换行门控（`commit_complete_source`） | rAF 合帧缓冲（`hooks/useChatStream.ts:185-192`、`:296-308`） | 只有**时间**门控，无"半行不进稳定区" |
| 稳定/尾切点 | `findStableCut`（`gaea/lib/markdownCut.ts:18-48`） | 只认 `\n\n` + 围栏回退；无表格/数学/列表标记 |
| 提交动画两档齿轮 | `useThrottledStreamText`（`hooks/useThrottledStreamText.ts:12-14`、`:22-25`） | 是**倍增退避**（`min(2000, 2×上次间隔)`），不是队列压力模型 |
| 尾区窗口 + 绘制成本 | `MessageList` 尾部窗口 `50/80/60/120px`（`components/chat/MessageList.tsx:66-72`）、`content-visibility:auto`（`chat-board.css:197-200`） | 约束的是**历史行**，与流式双区无关 |
| 稳定区不可变（memo 边界） | `React.memo(ChatRow)`（`MessageList.tsx:75`）、`MarkdownContent` memo（`components/MarkdownContent.tsx:37`） | 键是**整串 text**，切点一动全前缀失效 |
| 围栏增量高亮 | `ChatCodeBlock` → `HlCode`（`components/ChatCodeBlock.tsx:77-95`、`gaea/components/HlCode.tsx:9-20`） | 无增量状态，text 变即整段重高亮 |
| 渲染缓存键 | 仅 `source/className/components` 引用 | 缺 width、主题 revision、color level |

**真实缺口（带证据）**

1. **切点会回退，稳定段并不稳定**：围栏内切点回退到围栏开启行之前（`markdownCut.ts:43-46`）→ `stablePart` 字符串变化 → `MarkdownContent` 的 memo（`MarkdownContent.tsx:37`）整个前缀重解析。codex 的对应做法是稳定前缀**只增不减**，回退走显式重渲预算（`controller.rs:38`、`controller.rs:461-483`）。
2. **表格没有定形判定**：gaea 靠"表内无空行"被动把表格留在 `pendingPart`（`markdownCut.ts:19-20`），没有 `PendingHeader/Confirmed`。后果：表头行可能先随 `\n\n` 进 `stablePart`，分隔行到达后前缀被改写——正是 `controller.rs:14-18` 要避免的重塑。
3. **围栏高亮重复解析**：流式尾段直接进 `MarkdownContent → ChatCodeBlock → HlCode`（`ChatRow.tsx:222`、`ChatCodeBlock.tsx:88-92`），`HlCode` 的 effect 依赖 `[text, lang]`，每帧整段 `highlightCode` 并 `dangerouslySetInnerHTML` 重建（`HlCode.tsx:11-25`）。长围栏＝O(n²)，且没有任何 4 KB/512 KB/10 000 行限额（对照 `highlight.rs:640-654`）。
4. **尾区无字节上限**：`pendingPart` 无 8192 字节上限（对照 `prose_preview.rs:17`），超长单行/长表格时每帧全量 re-parse。
5. **缓存键缺维**：主题切换、容器宽度变化不进 memo 键（对照 `markdown_render_cache.rs:13-21`）；只能靠 CSS 变量侥幸正确，`HlCode` 侧则完全无缓存。
6. **resize 语义空白**：codex 有 `set_width` 的"留 1 行 + 前缀比例对齐"（`controller.rs:311-328`、`:394-414`）；gaea 靠浏览器 reflow，但 `contain-intrinsic-size: auto 120px`（`chat-board.css:197-200`）在宽度变化后需重新测量，尾区没有一次显式重排触发点。

## 九、可迁移机制三档

**① 直接照搬（算法/常量/不变量）**
- 三层计数不变量 `emitted <= enqueued <= rendered`（`controller.rs:38`）；尾区从 `enqueued` 起算（`controller.rs:272-277`）。
- 表格三态扫描器：状态定义、`prev_line.is_header + is_delimiter` 确认、空行收尾、`reset` 重扫条件（`table_holdback.rs:24-33`、`:155-167`、`:112-118`）。
- `MAX_PREVIEW_BYTES = 8192` + `ceil_char_boundary` + `…` 前缀 + "相等则不重排"（`prose_preview.rs:17`、`:82`、`:117-124`）。
- 高亮限额三件套 `4*1024 / 512*1024 / 10_000`（`highlight.rs:640-654`），跨限即整段回退纯文本。
- chunking 八常量（`chunking.rs:85-116`），若采用"按行发射"节奏可直接用原值。

**② 改造适配（终端语义 → DOM/React 语义，为什么不能一比一）**
- `commit tick` → **rAF 帧预算**：终端一 tick 一行是"打字节奏"，DOM 里每帧全量 diff 已经合并，照搬会额外引入 120 次/秒 setState。做法：保留 `Smooth=1 行/帧` 作为**可选**节奏，默认 `Batch(剩余)`，阈值降到 `queue>=8` 才切（沿用 `chunking.rs:85`）。
- `render.lines` 行边界 → **顶层块边界**：DOM 无法按"第 N 渲染行"切片（换行由 CSS 决定），所以稳定/尾的边界只能用**源字节偏移**（`markdown_render/streaming.rs:26` 的 `last_top_level_block_start` 语义可精确移植）。
- `OpenCodeFence` 增量高亮 → **行块缓存**：hljs 无 `ParseState` 可续跑（对照 `highlight_streaming.rs:36-67` 的可序列化状态），只能按 64 行一块缓存 HTML，尾块重高亮；`append` 的"严格追加否则整块重渲"守门（`code_fence.rs:87-94`）可以直接搬成"分块前缀必须严格前缀"。
- `MarkdownRenderCacheKey` → React memo 键扩展：`source + width + themeRevision + listSpacing`（`markdown_render_cache.rs:13-21` 去掉 fg/bg/color_level，改 CSS 变量）。
- `set_width` 前缀比例对齐 → **ResizeObserver 触发一次尾区重排**，不搬行映射（`controller.rs:290-293` 自己都说那是终端妥协）。

**③ 明确不搬**
- **scrollback 概念本身**（`emitted_stable_len` 的"已进终端回滚缓冲、不可再改"）：DOM 没有不可变区，只有"这次不再改的字符串"。等价的工程约束是 **stable 段的字符串在流结束前只允许在顶层块边界增长**，而不是"行数计数"。
- `CommitTickScope::CatchUpOnly` 机会式补帧（`commit_tick.rs:82-84`）：那是给终端"用户按键时顺带推进动画"用的，浏览器有独立渲染循环。
- `PlanStreamController` 双控制器（`commit_tick.rs:109-112`）：gaea 只有一条消息流。
- `finalize_remaining` 全量重渲 + `split_off(emitted)`（`controller.rs:212-224`）：DOM 里直接 `setStable(全文)` 即可，零成本。
- mermaid 关闭围栏时"用图替换源 + resize 回退源"（`controller.rs:23-26`）：gaea 已有 `ChatMermaid` 图表/源码双态（`ChatCodeBlock.tsx:84-92`），不需要这条。

## 十、施工卡

**新文件（计划路径，不实现）**

| 路径 | 关键签名 |
|---|---|
| `frontend/src/gaea/lib/streaming/twoRegionCut.ts` | `cutTwoRegion(text: string, opts:{tables:boolean}): { stableEnd:number; tailStart:number; reason:'paragraph'\|'table'\|'fence'\|'math'\|'listMarker' }` |
| `frontend/src/gaea/lib/streaming/tableHoldback.ts` | `class HoldbackScanner { scan(text:string, completedEnd:number):void; state():HoldbackState; allowsProsePreview():boolean }`；`type HoldbackState = {kind:'none'}\|{kind:'pendingHeader';headerStart:number}\|{kind:'confirmed';tableStart:number}` |
| `frontend/src/gaea/lib/streaming/highlightChunks.ts` | `highlightChunks(text:string, lang:string\|undefined, chunkLines=64): Promise<{html:string; hits:number}>`；块键 `lang\|themeRevision\|chunkIndex\|hash(chunkText)` |
| `frontend/src/hooks/useTwoRegionStream.ts` | `useTwoRegionStream(text:string, streaming:boolean, opts:{maxPreviewBytes:number; tickBudgetLines:number}): {stable:string; tail:string; tailReason:string; stableEpoch:number}` |

**改动文件（计划）**：`components/chat/ChatRow.tsx:99,220-228`（改用双区 + 拆 `<StableRegion>/<TailRegion>`）、`components/MarkdownContent.tsx:37`（memo 键加 `width/themeRevision`）、`components/ChatCodeBlock.tsx:88-92` + `gaea/components/HlCode.tsx:11-20`（改分块高亮）、`pages/chat/constants.ts`（新增阈值常量）。

**状态机（尾区）**

```
none --(行匹配表头)--> pendingHeader{headerStart}
pendingHeader --(下一行是分隔行 且 tables 开)--> confirmed{tableStart}
pendingHeader --(下一行非分隔/空行)--> none
confirmed --(空行 / re-entering Outside fence)--> none
confirmed | pendingHeader --(completedEnd 越过 holdbackStart)--> reset()+从新 completedEnd 重扫
```
（对齐 `table_holdback.rs:139-188`、`:112-118`）

**前端不变量**：`stableEpoch` 单调增；`tailStart` 单调不减，唯一例外是 `reason==='fence'` 回退，且回退必须 `stableEpoch++`（允许一次全量重渲）；`stable` 字节数 ≤ `text` 字节数且 `stable` 是 `text` 的前缀。

**开关与默认值**：`streamTwoRegion`，localStorage `gaea.stream.twoRegion`，**默认 `true`**；`0`/`false` 关闭。**关闭等价保证**：走 `findStableCut + useThrottledStreamText` 原分支，不改任何既有函数签名与导出（纯加分支），三个 fixture 快照逐字节一致。**回退方式**：运行时置 `gaea.stream.twoRegion=0`（无需发版）；代码回退为单个 commit revert。

**可自动断言验收**

1. 连续 5 个 delta（各追加 `"line\n"`）后：`tailStart` 单调不减、`stable` 的 sha1 不变、`stable.length` 增量等于新增完整行字节数（不重排）。
2. 送 `| a | b |\n` → `state.kind==='pendingHeader'` 且 `stable` 中**不含** `"| a |"`；再送 `| --- | --- |\n| 1 | 2 |\n` → `state.kind==='confirmed'` 且 `tableStart === headerStart`；断言前 N 行（表前正文）`getBoundingClientRect().height` 与首帧一致。
3. 200 行代码围栏、60 帧增量：`hits <= ceil(200/64)+1 = 5`，且最终 `html === 一次性全量高亮结果`。
4. 单行 100 KB 无换行：`tail` 渲染文本 UTF-8 字节 ≤ `8192` 且首字符 `…`。
5. 容器宽 900→620→900 px，静止 300 ms 后每行高度数组与一次性全量渲染逐项相等（`|Δ| <= 1px`）。
6. 开关关闭：渲染树 + 文本节点序列与 `v4.x` 现状快照完全一致（3 fixture）。

## 十一、风险

1. **性能：切点扫描 O(n)/帧**——`findStableCut` 已在每次 `display` 变化时全文扫（`markdownCut.ts:31-41`）。缓解：`HoldbackScanner` 保留 `sourceOffset`，只扫新增区（对齐 `table_holdback.rs:118-128`），并用断言 1 钉住。
2. **正确性：GFM 语义漂移**——blockquote 内表格、`~~~` 围栏、缩进代码块、表格后紧跟列表都会误判。缓解：一律**保守留尾**（拿不准就整块当尾，对齐 `table_holdback.rs:7-11` 的哲学），并把 `blockquote` 前缀剥离照搬（`table_holdback.rs:191-199`）。
3. **可观测性：切点回退造成"闪一下"**——回退触发整前缀重解析。缓解：打点 `stream.cut.fallback{reason}`，每流上限 1 次，超限降级为"不切"（全文当尾）并打 `stream.cut.disabled`。
4. **DOM 结构：拆两段丢跨段上下文**——列表续行、表格跨段时两棵子树各自解析会掉缩进/边框。缓解：只在顶层块边界切，尾段容器加 `data-seg="tail"` 抑制续行缩进；断言 2 的几何不变式覆盖表格场景。

## 十二、一句话

codex 的流式渲染不是"节流器"，是**一条不变量加四个守门人**：不变量管已提交内容永不回改，四个守门人（换行门控、表格三态、围栏快路径、预览上限）保证**尾区永远只包含"下一行可能改写它"的东西**——gaea 缺的正是这条不变量，所以表格与围栏在 DOM 里被反复重塑。

证据自查：`streaming/controller.rs:38`（`emitted_stable_len <= enqueued_stable_len <= render.lines.len()`，原文一致）、`streaming/chunking.rs:85`（`const ENTER_QUEUE_DEPTH_LINES: usize = 8;`，原文一致）、`streaming/prose_preview.rs:17`（`const MAX_PREVIEW_BYTES: usize = 8192;`，原文一致）
