# 资产 28 · 转录视图交互施工级：条目模型、三分坐标、搜索与跟随

> 源码：`codex-rs/tui/src/transcript_view.rs`（707 行）+ 同目录模块（`text.rs` 711 / `search.rs` 743 / `selection.rs` 832 / `copy_mode.rs` 381 / `disclosure.rs` 414 / `layout.rs` 406 / `follow_control.rs` 111 / `footer.rs` 272），外加 `transcript_reflow.rs` 331、`insert_history.rs` 1184、`pager_overlay*`、`wrapping.rs` 2096、`text_selection.rs` 78、`rendered_selection.rs` 193。
> 对照：gaea `gaea/components/Transcript.tsx`（1167）、`components/chat/MessageList.tsx`（197）、`utils/followScroll.ts`（66）。
> 本篇只讲一件事：**把「阅读位置」做成坐标系**——条目身份、逻辑行、显示行、源偏移四层各自稳定，回折/搜索/选区/跟随/跳转全部退化为坐标换算。

---

## 一、条目模型：HistoryCell 与 consolidation

流式期输出临时 `AgentMessageCell` 序列；答案结束，app 把整段尾部 run 换成**一个**持有原始 markdown 的 `AgentMarkdownCell`，它成为回折与复制的唯一真源：

```rust
// app/agent_message_consolidation.rs:47-74
let start = trailing_run_start::<AgentMessageCell>(&self.transcript_cells);          // :52
let consolidated: Arc<dyn HistoryCell> = Arc::new(source);                           // :57
self.native_history.consolidate(&self.transcript_cells[start..end], &consolidated);  // :58-59
self.transcript_cells.splice(start..end, std::iter::once(consolidated.clone()));     // :73-74
```

三条纪律：①视图先保住「已显示的那一版」再替换（`:8-9`），否则在读段落跳行；②终端只许打印一次——未打印的 run 整段替换，已打印一部分的只改 owner 不重发（`app/native_history.rs:51-79`，原话 *never replay the prefix already owned by the terminal*）；③`DynamicToolCallCell` 未结束时整队延后（`native_history.rs:22-26`, `:35-42`）。

身份是**指针**不是下标（`transcript_view.rs:53-63`，`EntryKey::Cell(Arc::as_ptr(cell) as usize)`）：`resolve()` 先按下标比指针、失败全表搜指针、再失败才退化 `index.min(len-1)`（`:672-688`），故「N 条换 1 条」不让锚点错位。冻结层 `ViewSnapshot`（`snapshot.rs:9-15`）只在选区或阅读期存在，pin 住的是 layout 本身。注意视图**不持有** `cells`：每次都从参数拿（`transcript_view.rs:185`），app 侧才是真源（`app/native_history.rs:12-15`）。

## 二、三分坐标：逻辑行 / 物理行 / 制表符展开

| 字段 | 含义 | 证据 |
|---|---|---|
| `logical: Vec<LogicalLine>` | 源行（缩进、换行策略、源区间） | `text.rs:46`、`text_logical.rs:14-22` |
| `text: String` | 逻辑行以 `\n` 拼成的**唯一源串**（选区/搜索/复制都读它） | `text.rs:43-54`, `:525-545` |
| `rows: Vec<TextRow>` | 物理行：`source` 区间、`line_end`、`first_column`、`tabs` | `text.rs:62-71` |

逻辑行合并判据是「同一源串 + 区间连续」，不是「看着像一段」：

```rust
// text_logical.rs:31-34
Arc::ptr_eq(&p.origin.text, &source.text) && p.origin.range.end <= source.range.start
```

`from_logical` 逐行写 `text` 并回填 `row.line_end`（`text.rs:529-545`）：软回折行 `None`（`:65-66`）、硬换行有值，硬/软在数据上可分。制表符是**第三种展开**——源字节不变，每个 `\t` 在实际显示行（含合成缩进）推进到下一个 8 列制表位（`text_tabs.rs:1-6`），实现按 `\t` 切 run 后逐段回折（`:91-100`），稀疏表反查列（`:35-47`）。

| 换算入口（一律走 `TextLayout`） | 方向 | 证据 |
|---|---|---|
| `position_at(row,col)` | 显示 → 源偏移（grapheme 夹紧，padding 归行尾） | `text.rs:342-356` |
| `row_for_offset(off)` | 源偏移 → 显示行（跳过 disclosure 合成行） | `text.rs:406-423` |
| `column_for_offset_on_row` / `cursor_position_at` | 源偏移 → 列（含 `\t`）/ 键盘 caret | `text.rs:430-443`, `:359-380` |

回折走 URL 感知路径（标准 `textwrap` 会切开 URL，`wrapping.rs:1-26`）：`adaptive_wrap_line_with_source`（`:726-756`），源区间由 `word_wrap_line_with_source`（`:913`）回填。

## 三、resize 语义：以 cells 为真源重排

```rust
// transcript_view.rs:195-202
let resized = self.area.width != area.width || self.area.bottom() != area.bottom(); // :196
self.prepare_width(area.width);                                                     // :197
if resized { self.reveal_copy_context(cells); }                                     // :200-202
```

`prepare_width`（`search.rs:329-343`）= 快照 layout 全部 `rewrap` + 当前命中 `rewrap` + **重启搜索**（源偏移只对某个源布局有效）；`TextLayout::rewrap` 复用 `logical`，只重建显示行并按源偏移重插 disclosure 控件（`text.rs:136-151`）。终端侧另有 reflow 调度，关键是**三个宽度分开记**（观察/已重建/待重建）：

```rust
const TRANSCRIPT_REFLOW_DEBOUNCE: Duration = Duration::from_millis(75); // transcript_reflow.rs:18
fn note_width(&mut self, w: u16) -> TranscriptWidthChange              // :62-71 首帧只初始化
fn reflow_needed_for_width(&self, w: u16) -> bool                      // :79-81 比"已重建"
fn schedule_debounced(&mut self, target: Option<u16>) -> bool          // :89-96
fn schedule_immediate(&mut self)                                       // :102-105 consolidation 后
```

| 事件 | 变化 | 结果 |
|---|---|---|
| 首帧 / 高度变大 | `last_reflow=W` / `schedule_debounced(None)` | 首帧不重建（`:64-66`）；高度变大需重建但不占宽度名额（`:86-88`） |
| resize W→W' | `pending=W'`，截止 `now+75ms`，每次事件推后 | 中间宽度全跳过，只按最终宽度重建 |
| 重建期仍在流式 | `ran_during_stream` | 流结束强制再跑（`:9-13`） |

## 四、折叠披露：以「保留的活动身份」为键的集合状态机

```rust
// disclosure.rs:17-22
struct Disclosure { expanded: HashSet<String>, focused: Option<Vec<String>>,
                    live_ids: Vec<String>, keymap: RuntimeKeymap }
```

* id 来自 cell 的 `activity_ids()`（`disclosure.rs:81-86`），判定是组内任一命中 `expanded`（`:36-38`）。
* 控件**只在该组获焦时绘制**（`:101-130`），作为零宽合成行插在源偏移处、不进 `text`——不参与复制与搜索（`text.rs:160-203`，注释 `:6`）；搜索的临时展开不写回集合（`search_presentation.rs:7-11`、`snapshot.rs:64-72`）。
* 活尾被提交时锚点从 `EntryKey::Live` 改挂新 cell，`expanded` 集合原样迁移（`disclosure.rs:51-79`）；标签按宽度与状态切换（`layout.rs:38-54`：`+ N lines (ctrl+o to expand)` / `− Show less` 等）。

```
无 ids ─► NoControls；ids 非空且 !detailed && Rich ─► Collapsed（控件隐藏，行零宽）
Collapsed ──获焦/展开键──► Focused ──► Expanded(ids∈expanded)；收起键回 Collapsed
Expanded ──活尾 consolidation──► 集合按 id 迁移到历史 cell（disclosure.rs:51-79）
任意态 ──搜索命中──► 仅该 match 的 layout 临时展开，集合不变
```

## 五、搜索：增量扫描、单命中、页脚即状态机

搜索**不读可见文本**，读全量源文本（`search.rs:1-6`：*independent of collapsed display*）。

```rust
const QUERY_BYTES: usize = 4096;      // search.rs:23
const SCAN_BYTES: usize = 16 * 1024;  // search.rs:24
const ENTRIES_PER_FRAME: usize = 8;   // search.rs:25
```

`SearchMode = Closed|Editing|Reading`（`:56-61`），`Progress = Idle|Restart|Scanning|AwaitingHistory|Found|Exhausted`（`:40-48`），只保留一个当前命中 + 一个扫描游标（`:50-54`）。`advance_search`（`:346-406`）返回「本帧能否推进」：①`Restart`→回原位清块、从 Older 重扫（`:353-363`）；②`AwaitingHistory` 且历史已 `Complete/Idle` → 直接 `Exhausted`（`:364-371`），不许无限等；③最多 8 条，命中即 `install_search_match` 返回 false（`:373-396`）；④累计过 `SCAN_BYTES` 提前返回（`:399-403`）。

查询超限**回滚本次输入**并置 `query_truncated`（`:189-203`），粘贴按 grapheme 截断（`:266-288`），页脚缀 `· query limited to 4 KiB`（`search_footer.rs:78-82`）。命中安装（`search_presentation.rs:14-52`）：命中行上方留**三分之一视口**（`:36-40`），用 `move_rows` 负向移动让上下文可落在前一条目，同时保留命中 revision 的 layout（`:24-33`）——「跳转不跑偏」全在这。高亮只染源 grapheme（`text.rs:446-471`），行尾硬换行补一格（`:474-501`），选区/搜索激活时让位（`transcript_view.rs:277-280`）。越界向 app 要页：`needs_history` = 搜索需要 || 视口近顶（`:547-550`），`near_start` 以视口高度为阈值累加前序行数（`:552-572`）；取消必须还原呈现（`search.rs:232-264`）。

| Progress | 页脚首行 |
|---|---|
| `Idle` 空 / 非空 | `Type to find` / `enter accept · ctrl+p older · ctrl+n newer` |
| `Scanning` / `AwaitingHistory` | `Searching…` / `Searching earlier history…`（失败：`History unavailable · ctrl+p retry`） |
| `Found` / `Exhausted`+命中 / `Exhausted` | `Find · …· esc latest` / `No more matches · …` / `No matches` |

## 六、选区与 copy mode：字符级 / 块级 / 整条回答

`Anchor{key,index,offset,row_bias}`（`transcript_view.rs:65-73`）：正 bias = 源之前（分隔行），负 bias = 源之后（披露控件）——同一源偏移上的合成行由此区分。选区一生成就冻结 `ViewSnapshot`（`selection.rs:25-41`），只 pin 可见与选中的 layout（`:1-2`、`snapshot.rs:37-82`），故边拖边流式不会跑位；单位字符/词/行，行单位按**逻辑行**（含硬换行）而非显示行（`text_selection.rs:12-17`, `:38-46`），连击 400ms 内 `%3+1` 循环（`:51-66`）。

块级候选由源元数据成段给出——同一 `copy_source` 指针才算一块，相邻等值载荷不并块（`text.rs:74-128`, `:97-107`）。揭示目标只滚到最近边界并留 1/4 视口留白（`copy_mode.rs:242-277`, `:267-273`），超长目标从上文起显（`:240-241`）；键位 `↑/↓/j/k`、`g/G`、`Enter` 复制并回跟随（`:62-89`），动作以 `ViewAction` 出栈（`input.rs:21-29`），`is_copy_key` 兼容 Cmd+C / Ctrl+Shift+C / Ctrl+Insert（`text_selection.rs:68-78`）。

| 档 | 端点 | 复制内容 |
|---|---|---|
| 字符级拖选 | `Anchor{offset}`+`row_bias`，跨折叠仍连续 | 源串切片 |
| 块级 | `copy_block_ranges` 源区间 | 整块原文 + 语言/来源元数据 |
| 整条回答 | `CopyMode{whole:true}` | `copy_source()` 或 markdown 原文（`copy_mode.rs:34-47`） |

## 七、跟随、未读、书签

跟随是三个与条件的合取：`select_mode.is_none() && selection.is_none() && position == Latest`（`transcript_view.rs:430-432`）。「回底」不是滚动命令而是状态归一：向下滚到 `bottom_start` 就 `jump_to_latest`（`:487-495`），后者清选区/选择模式/copy mode/未读/披露焦点（`:447-457`）；底部起点由「末条行数 + recap 间距」反推，不记上次 scrollTop（`:605-621`，recap 判定 `:608-615`）。

未读两处写入：尾部变化时只统计**有行**的新条目（`mutations.rs:9-26`）、活尾 revision 变且未跟随（`transcript_view.rs:387-389`）；清除条件是末行真被画出，判据 = key + layout 文本 + `row+1==row_count` 三者同时成立（`activity.rs:11-28`）。回底控件是视口内一行，隐藏时绝不拦指针（`follow_control.rs:1-3`），标签按宽度降级且随未读换文案（`:52-81`），命中矩形渲染期写入（`:21-50`），按下抬起都须在矩形内（`:83-106`）。

书签存位置三元组而非滚动量（`position`/`saved_position`/`suppressed_prompt_header`/`detailed+mode`/快照），恢复先 `jump_to_latest` 再 `set_presentation` 再按当前宽度 `rewrap_snapshot`（`bookmark.rs:6-41`）。

## 八、prompt header、页脚、分页浮层

**prompt header** 是表现层一行、永不进内容（`prompt_header.rs:1-3`）：仅当首条可见条目不是用户提示且宽度 ≥16 时向上找最近有内容的 `UserHistoryCell`（`:24-41`），text sanitize 后截 512 字符（`:43-47`），空则 `[attachments]`（`:48-52`）；占位时正文收缩一行并**重算起点**（跟随可能越过 turn 边界，`transcript_view.rs:222-245`）；在边界让位则冻结 `SuppressedHeader{viewport,key,row}` 防闪（`prompt_header.rs:10-16`、`transcript_view.rs:217-221`）。

**页脚**一次只出一个，优先级：copy mode → 搜索行 → 选区提示 → 搜索读数 → 活动披露 → 加载/未读/可回底（`footer.rs:17-151`）；加载时 shimmer 且**先缩次要文本再裁快捷键**（`:100-133`），`navigation_line` 只给已知动作前缀上键帽、绝不作用于正文与查询串（`:229-268`）。

**分页浮层**复用同一 `TranscriptView`（`pager_overlay.rs:51-62`），只换 chrome 高度（浏览 2 行 / 交互 5 行，`pager_overlay/transcript.rs:70-80`）；静态分页用 `render_scrolled`，不支持则回落「高缓冲再拷贝」（`pager_overlay/scrolling.rs:8-40`）；键盘临时接管靠总开关 `has_active_interaction()` = 选择模式 || 选区 || 搜索激活 || 活动聚焦（`transcript_view.rs:419-424`）。

## 九、gaea 现状核对：已有与真缺

| 能力 | 现状（file:line） | 真实缺口 |
|---|---|---|
| 尾部窗口 | `MessageList.tsx:66-70` 常量、`:132-138` 切片、`:144-161` 顶部 120px 扩载 | 只按**条数**，无行概念；扩载靠 Chromium 锚定补偿（头注释 `:9-10`），无自有锚点 |
| 绘制虚拟化 | `chat-board.css:197-200`（`content-visibility:auto` + `contain-intrinsic-size:auto 120px`） | 离屏行是估高，跳转/resize 后 bbox 漂移；缺「实测行高→锚点回算」 |
| 吸底跟随 + 回底按钮 | `followScroll.ts:24-39`（贴底恢复、滚轮上滚才取消）、`ChatPage.tsx:211`、`Transcript.tsx:784`/`:836-838` | 判据只有 `atBottom`（`scroll.ts:3` 阈值 80px）；缺 `tailVisible`/`unseen`/末行已绘制（对照 `activity.rs:11-28`）与未读文案阶梯（`follow_control.rs:52-81`） |
| 折叠披露 | `Transcript.tsx:417-454` 的 `useState(false)` | 组件内布尔、无 id、不持久化，无法在 consolidation 时迁移（对照 `disclosure.rs:17-22`） |
| 行级搜索 + 高亮 | 全仓无 in-transcript find：`SearchPanel.tsx:33-51` 是联网来源面板，`SearchModal.tsx:67-90` 是全库检索，`:44-57` 只标首个命中 | 缺「查询→命中→游标→跳转」整链与行内全命中分级（对照 `search.rs:40-48`, `:346-406`） |
| 复制语义 | `ChatRow.tsx:141-143`、`:234-236` 整条 `msg.content` | 缺字符级拖选与块级两档（对照 `text.rs:74-128`） |
| 跳转定位 | `Transcript.tsx:884-887` `scrollIntoView({behavior:"smooth"})` | 平滑滚动+估高漂移；无 1/3 视口上文与锚点校准（对照 `search_presentation.rs:36-40`） |
| 折叠保位 | `Transcript.tsx:897-904` `setTimeout(250)` 硬还原 `scrollTop` | 与动画时长耦合且是像素还原；应还原锚点再 rewrap（对照 `bookmark.rs:31-41`） |
| 分页/上下文头 | `ChatPage.tsx:211` 仅 `atBottom`；`MessageList.tsx:151-153` 触发 `onLoadOlder`；`:177` 只有 `newGroup` 名牌 | 无 `LoadingBeginning/Partial/Failed` 态与页脚投影（`pager_overlay.rs:397-412`）；长回答滚过提问后丢上下文（`prompt_header.rs:18-68`） |

## 十、可迁移机制三档

**A 直接照搬**：①四层坐标 + `Anchor{key,index,offset,row_bias}`（`transcript_view.rs:65-73`）；②`expanded: Set<string>` + 组内任一命中（`disclosure.rs:17-38`）；③`SearchMode × Progress` 状态机与每帧预算（`search.rs:23-25`, `:40-48`）；④「贴底恢复、手势取消」+ 末行已绘制的未读判据（`followScroll.ts:24-39` + `activity.rs:11-28`）；⑤prompt header 的表现行语义（`prompt_header.rs:10-52`）；⑥书签三元组（`bookmark.rs:6-41`）。

**B 改造适配**：①reflow：保留「源为真源 + 逻辑行复用」，把 75ms 尾随去抖换成 `ResizeObserver → useLayoutEffect 两阶段测量`（先测受影响行，再落锚点）；②copy mode：不用 `markdown_copy::SelectedLine`，改在渲染前 markdown AST 切块，端点仍是「块 id + 字符偏移」；③搜索：`SCAN_BYTES` 换 `requestIdleCallback` 分片 + `AbortSignal`，保留 4 KiB 上限与 grapheme 截断；④页脚：按 `footer.rs:17-151` 的优先级阶梯接入现有底部槽位；⑤历史分页：`TranscriptHistoryState` 六态直接映射 `hasOlder/loadingOlder/loadError`。

**C 不搬（终端独有）**：

| 机制 | 理由 |
|---|---|
| `insert_history.rs:1-5`, `:45-59` 用转义序列写历史 | 浏览器没有「应用拥有的 scrollback」，DOM 即 scrollback |
| `transcript_reflow.rs` 「清空自有历史再重发」 | 只解决「终端已写出的行归终端所有」（`:1-7`）；重排由浏览器负责 |
| Sixel/图片协议与 `terminal_cell_size`（`text.rs:307-316`）、tmux/Zellij 键位特判（`input.rs:65-86`） | 无对应概念；`<img>` + 实测尺寸即可，终端复用器键位在 Web 无收益 |
| `rendered_selection.rs:46-78` 截屏采样逐格复写；`Buffer` diff（`:74-77`） | 只读区域安全选区与 ratatui 管线专属；DOM 用 `Range` 原生能力 |

## 十一、施工卡

| 落点（新增） | 职责与签名 |
|---|---|
| `gaea/transcript/anchor.ts` | `type Anchor = { entryId: string; lineIdx: number; offset: number; bias: number }`；`anchorToPx(root, a, rowH: RowHeightMap): number` |
| `gaea/transcript/measure.ts` | `interface RowHeightMap { get(entryId, lineIdx, width): number; invalidate(entryId?): void }`；`measureEntry(el: HTMLElement, width: number): RowHeightMap` |
| `gaea/transcript/useFollowControl.ts` | `useFollowControl(scrollerRef, { bottomThresholdPx?: number }) => { atBottom, tailVisible, unseen, follow(), jumpToLatest(), onScroll(), onWheel(e) }` |
| `gaea/transcript/useTranscriptFind.ts` | `useTranscriptFind({ entries: () => FindEntry[], onJump(a: Anchor, ctxPx: number) }) => { open, close, setQuery, next(dir: 'older'\|'newer'), state: FindState }` |
| `gaea/transcript/disclosureStore.ts` | `useDisclosure() => { isExpanded(ids), toggle(ids), focus(ids \| null), expanded: ReadonlySet<string> }` |
| `gaea/components/Transcript.tsx` | 只接线：`stick.current` 换 `useFollowControl`，折叠 `useState` 换 `useDisclosure`；不动 `buildSegments`/`TurnBlock` |

```ts
type TranscriptState = {
  position: { kind: 'latest' } | { kind: 'reading'; anchor: Anchor };
  savedPosition: TranscriptState['position'] | null;                  // ← saved_position
  unseen: boolean; tailVisible: boolean;                              // ← unseen_activity / tail_visible
  disclosure: { expanded: Set<string>; focused: string[] | null };
  find: { mode: 'closed'|'editing'|'reading'; query: string; current: Anchor | null;
          progress: 'idle'|'restart'|'scanning'|'awaitingHistory'|'found'|'exhausted'; truncated: boolean };
};
```

**可自动断言验收**：①尾部在视口外时追加一条消息，已渲染第 k 行 `getBoundingClientRect().top` 变化 ≤0.5px；②上滚 1px 后 3 个 rAF tick 内 `scrollTop` 未被改写（spy setter 计数 0）；③滚回底部 1px 内，下一帧 `scrollTop === scrollHeight - clientHeight`；④折叠展开 250ms 后锚点行 top 差 ≤1px；⑤搜索命中第 3 条第 2 段，命中行距视口顶 ≈`height/3` 且 `state.find.current.entryId` 正确；⑥`next('older')` 连按 5 次命中单调前移，耗尽时页脚命中 `/No more matches/`；⑦全选复制文本 === 逻辑行按 `\n` 拼接且不含 `+ N lines` 控件文案；⑧容器 600→900px 后同一 `Anchor` 的 `offsetTop` 重算且 `entryId` 不变。

**开关与默认**：`transcript.strictAnchor`（on）、`transcript.followUnseen`（on）、`transcript.disclosurePersist`（on）、`transcript.find`（off，灰度达标转 on）。**回退**：三开关全关即等价旧行为；新逻辑全在新增文件，删除引用即回退；`Transcript.tsx` 旧 `stick`/`showScrollDown` 分支保留一个版本周期。

## 十二、风险

1. **估高与锚点互斥**：`content-visibility:auto` 下离屏行只有 120px 估值（`chat-board.css:199`），`anchorToPx` 必须「先 scrollIntoView、再实测、再修正一次 scrollTop」，一次不收敛会抖。
2. **并发渲染**：`useLayoutEffect` 落底可能与 transition 打架，需 `flushSync` 或双 rAF，否则流式期出现「先落后被旧布局覆盖」的假冻结（`followScroll.ts:5-8` 记过同类病灶）。
3. **分片预算不可控**：`requestIdleCallback` 节流使长会话首查十余帧，必须保留「历史已 Complete 即判 exhausted」的出口（`search.rs:364-371`），否则页脚永停 `Searching earlier history…`。
4. **披露 id 稳定性**：`expanded` 按 id 持久化，重新生成导致 id 变化会静默丢展开态；codex 依赖「保留工具身份」（`disclosure.rs:1`, `:51-79`），React 侧需约定 `ids = toolCallId ?? hash(turn,index,kind)`。

证据自查：`transcript_view.rs:487`（向下滚到 `bottom_start` 即 `jump_to_latest`）、`search.rs:24`（`SCAN_BYTES = 16 * 1024` 每帧扫描预算）、`disclosure.rs:18`（`expanded: HashSet<String>` 披露集合）。
