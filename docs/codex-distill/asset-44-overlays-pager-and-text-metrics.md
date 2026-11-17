# 资产 44 · 浮层与文本度量：一个翻页器三种内容、重放上限、按显示宽度截断

> 证据根（只读）：`pager_overlay.rs`（591）、`transcript_reflow.rs`（13.6KB）、`resize_reflow_cap.rs`（6.3KB）、
> `line_truncation.rs`、`rendered_selection.rs`、`vim_search.rs`、`wrapping.rs`（72.8KB，只作对比）、`width.rs`。
> 对照根（只读）：`frontend/src/components/chat/MessageList.tsx`（10.6KB，含"为何不用 react-window"的注释）、
> `frontend/src/gaea/components/Transcript.tsx`（53KB）、`components/chat/SearchPanel.tsx`、`pages/ChatPage.tsx:211`（滚动宿主）。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、一个翻页器承载三种内容

`pager_overlay.rs:44-48`：

```rust
enum Overlay { Transcript(TranscriptOverlay), Static(StaticOverlay), Analytics(Box<AnalyticsView>) }
```

- `new_transcript(cells, keymap, copy_on_select, mouse_scroll_speed)`（`:51-62`）把**单元格列表**交给翻页器；
- `new_static_with_lines/new_static_with_renderables`（`:71-79`）承载任意静态长文（帮助、diff、日志）；
- 分析面板直接是一个 `Overlay::Analytics` —— 统计面复用同一个"全屏可滚"容器。

**三种内容的共同点只有一个：它们都是"可滚的行源"**。于是键位、滚动、提示条、关闭语义只实现一次。

`PagerView`（`:172-207`）：

| 字段 | 用途 |
|---|---|
| `renderables: Vec<Box<dyn Renderable>>` | 行源（可多段拼） |
| `scroll_offset: usize` | 滚动位置（行索引，不是像素） |
| `last_content_height: Option<usize>` | 上一帧内容高度（用于翻页/回顶判断） |
| `title` + `keymap` | 标题与键位（同一份运行期键位表） |

`content_height(width)`（`:196-201`）= 各 renderable 的 `desired_height(width)` 之和——
**度量与渲染同源**（同一个 `desired_height`），不是估一个值。
`render` 开头先 `Clear.render(area)`（`:203-204`），理由与 F34 §二相同：重排后不许留残影。
底部提示条固定高度 `HINTS_HEIGHT = 3`（`:423`），由 `render_key_hints` 绘制——提示与键位同源。

`TranscriptHistoryState`（`:398`）把"转录历史处于什么状态"显式化（加载中/已到顶/…），
而不是靠 `scroll_offset == 0` 去猜。

## 二、重放上限：按终端给不同 cap，且说明"为什么要 cap"

`resize_reflow_cap.rs:1-35` 文件头把道理讲完了：

> 「这些 cap 是刻意保守的：**Codex 是在重建普通终端 scrollback，不是内部虚拟转录**。
> 重放比终端保留量更多的行是浪费，还可能让交互 resize 更卡，却不会给用户更多可用历史。」

| 终端 | cap 行数 | 证据 |
|---|---|---|
| VS Code 终端 | 1,000 | `:19` |
| Windows Terminal | 9,001 | `:20` |
| WezTerm | 3,500 | `:21` |
| Alacritty | 10,000 | `:22` |
| 其它 | 配置的回退值 | `:11` |

`resize_reflow_max_rows(config)`（`:29-35`）在 `Auto` 时用终端探测 + VS Code 环境探针（因为 VS Code 里的 shell 可能自报成宿主 shell）；
**`max_rows = 0` 表示用户显式关闭行数限制**（返回 `None`）。

> 对 Web 的映射不是照抄数字，而是照抄**决策逻辑**：
> "渲染超过宿主能保留的历史是纯浪费"。gaea 的 `MessageList` 用尾部窗口 + `content-visibility` 正是这条决策的另一种实现，
> 但**没有把 cap 写成可配置的显式上限**（注释里只有原因，没有数值边界）。

## 三、按**显示宽度**截断，且保留样式

`line_truncation.rs:7-40`：

- `line_width(line)` = 各 span 的 `display_width` 之和（不是字符数）；
- `truncate_line_to_width(line, max_width)`：逐 span 累计宽度，**整 span 放得下才放**，
  超限即停（不切半个 span 的样式）；`max_width == 0` 直接返回空行（而不是原样返回）；
- 另一个入口 `truncate_line_with_ellipsis_if_overflow` 在溢出时补省略号（被 F35 的 mention footer、F38 的列布局共用）。

`width.rs` 与 `wrapping.rs` 是同一套度量的地基：`display_width` 处理 CJK 宽字符/组合字符，
`wrapping.rs` 用 grapheme 级切分 + sentinel 字节保证"软换行不改源文本、光标位置是源偏移"。

## 四、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `components/chat/MessageList.tsx`（10.6KB） | 尾部窗口 + `content-visibility`，注释记录"为何不用 react-window" | 有决策与理由，**没有显式 cap 数值**与"宿主保留量"的概念 |
| `pages/ChatPage.tsx:211` | 滚动宿主在页面层 | 与 codex 的 PagerView 结构不同（页面滚 vs 容器滚），但都要求"度量与渲染同源" |
| `gaea/components/Transcript.tsx`（53KB） | 会话转录（多类行：消息/工具/过程/交付） | 无"一个容器承载转录 + 静态长文 + 分析面板"的复用；三处各有各的滚动实现 |
| `components/chat/SearchPanel.tsx` | 搜索面板（会话内搜索） | 与转录分离；codex 的 pager 搜索是"在同一个行源上扫" |
| 全仓 | 无按显示宽度截断的统一 helper | 多处 `truncate`/`slice` 按字符切，CJK/emoji 会切坏（与 F34 的 `TOOL_OUTPUT_LINE_CAP=160` 同源问题） |

## 五、三档判定

**档一（照搬语义）**

1. **一个可滚容器三种内容**：转录 / 静态长文（帮助、日志、diff）/ 分析面板共用一个外壳（滚动、键位、提示条、关闭）。
2. **行索引滚动**（`scroll_offset: usize`）而不是像素偏移——与 F28 的"坐标系而非像素"同一条纪律。
3. **顶部/底部提示条固定高度**，由键位表派生。
4. **`max_width == 0` 返空**：截断函数对 0 宽度有确定语义（避免调用点各自特判）。

**档二（改造搬）**

5. **显式 cap + 理由**：把 `MessageList` 的隐式窗口变成显式常量（如 `MAX_RENDERED_ROWS`），并在注释里写清"超过宿主保留量是浪费"。
6. **统一 `truncateToWidth`/`lineWidth`**：按显示宽度（`Intl.Segmenter` + East-Asian 宽度）截断，替换散落的字符切。
7. **转录状态显式化**（`TranscriptHistoryState`）：加载中/到顶/未读，不靠偏移量猜。

**档三（不搬）**

8. 每终端一个 cap 的数字（1000/9001/3500/10000）、`wrapping.rs` 的 sentinel 字节与 grapheme 换行（浏览器负责）；
9. `vim_search.rs` / `rendered_selection.rs` 的终端选区与 vim 搜索。

## 六、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F44-A 显式渲染上限** | `components/chat/MessageList.tsx` | `MAX_RENDERED_ROWS` 常量 + 注释写明理由；超过时保留首尾窗口 | 性能：3000 条夹具 60fps；DOM 节点数有上限 |
| **F44-B 统一按宽截断** | 新 `lib/textMetrics.ts`（`lineWidth`/`truncateToWidth`/`withEllipsis`） | 替换 `truncate`/`slice` 用法（先工具卡、侧栏、状态条三处） | 单测：CJK 宽度=2；emoji 不切半；`maxWidth=0` ⇒ 空 |
| **F44-C 转录状态** | `Transcript.tsx` | `historyState: idle/loading/top/none` 显式 | 单测：到顶 ⇒ `top`；加载中 ⇒ `loading` |
| **F44-D 统一浮层外壳** | 新 `components/ui/PagerOverlay.tsx` | 行源 + 标题 + 键位 + 提示条 + 关闭 | 手测：帮助/日志/分析三处共用，键位一致 |

**开关**：A 为常量提取（行为不变）；B 逐处替换、每处带单测；D 新增件，`FEATURE_PAGER` 单闸。

## 七、风险

1. **`MessageList` 是历史踩坑区**（注释记录了 react-window 的失败）：A 卡只允许"提取常量 + 注释"，禁止改窗口策略。
2. **行索引滚动与 DOM 滚动**：Web 的 `scrollTop` 是像素，F28 已判定要引入坐标系；F44-C 的状态判断若依赖像素会与 F28 冲突，需共用同一坐标系实现。
3. **`Intl.Segmenter` 可用性**：Chromium/WebView2 已支持；仍需 fallback（按码点）。

## 八、证据自查

- 三态 Overlay 与构造：`pager_overlay.rs:44-79`；PagerView：`:172-207`；提示条：`:423`；历史态：`:398`。
- 重放上限与理由：`resize_reflow_cap.rs:1-35`。
- 按宽截断：`line_truncation.rs:7-40`；度量地基：`width.rs`、`wrapping.rs`（对比用）。
