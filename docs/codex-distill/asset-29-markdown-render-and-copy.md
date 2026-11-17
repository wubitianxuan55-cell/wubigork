# 资产 29 · Markdown 渲染与表格/数学/链接/复制施工级

> 源码：`tui/src/markdown.rs`、`markdown_render.rs`（3267 行，Writer 与表格主体）、
> `markdown_render/{source_tables,table_key_value,math,math/render,math/render/structured,local_links,web_links,list_spacing,inline_directives,preferences,task_lists,mermaid,streaming}.rs`、
> `tui/src/markdown_copy.rs` + `markdown_copy/table.rs`、`table_detect.rs`、`render/highlight.rs`、
> `line_truncation.rs`、`terminal_hyperlinks.rs`、`config/src/tui_rendering.rs`。
> 对照：gaea `components/{ChatMarkdown,ChatCodeBlock,MarkdownContent}.tsx`、`components/chat/{ChatMermaid.tsx,remarkDisableCodeIndented.ts}`、
> `gaea/lib/{mathText,tableData,remarkFileLinks,remarkMemCitations,markdownCut,sanitize,codeHighlight}.ts`。只读源码，不改 gaea 代码。

---

## 一、渲染管线（缺一段就不成立）

| 阶段 | 落点 | 干什么 | 关掉的后果 |
|---|---|---|---|
| ① 解析选项 | `markdown_render.rs:424-432` | 开 `STRIKETHROUGH/TABLES`；`TASKLISTS` 跟 `preferences.lists` | GFM 表变代码 |
| ② 行内指令 | `streaming.rs:71` | `:codex-file-citation{}`/`:codex-followup[]` → link/label | 指令原文漏屏 |
| ③ 数学掩码 | `math.rs:27`、`markdown_render.rs:428` | TeX 先占位、后回填 Unicode 布局 | `\$` 被 Markdown 吃掉 |
| ④ 源表还原 | `source_tables.rs:14-36` | `tables=false` 时把表**还原成字面块** | 关了表仍被解析 |
| ⑤ 写行 | `Writer`（`markdown_render.rs:1410`） | 列宽分配、键值回退、copy 标注 | 无复制语义 |

**主轴**：不是一次性 `to_html`，而是**事件流 + 每行携带 copy 注解**——`HyperlinkLine` 同时装显示 span、链接元数据、`LogicalLineSource` copy 源（`terminal_hyperlinks.rs:44-48`），复制时反推 Markdown。

## 二、机制事实

### 2.1 `tables` 开关是两条反向的路（`tui_rendering.rs:11-30`，mermaid/math/tables/lists 默认全 `true`）

```rust
if preferences::current().tables { unwrap_markdown_fences(s) } else { Cow::Borrowed(s) }  // markdown.rs:256-262
```

- **解包极保守**：info 必须是 `md`/`markdown`（`table_detect.rs:219`，大小写不敏感），且 body 内出现「上一行像表头 + 当前行是分隔行」相邻对——`is_table_header_line(previous) && !is_table_delimiter_line(previous) && is_table_delimiter_line(trimmed)`（`markdown.rs:374-378`）。
- 必须整块缓冲才判定；未闭合围栏按原样（含开围栏行）重发，流式半截降级成代码（`markdown.rs:272-276`）；无 ``` 时零拷贝借出（`:284`）；围栏缩进 **< 4 列**（空格 1、Tab 4，`:300-317`）。

### 2.2 表「识别」是纯结构判定（`table_detect.rs`）

`parse_table_segments`（:38-54）去首尾 `|`、按**未转义** `|` 切、段去空白，无外框且仅 1 段 → `None`；`split_unescaped_pipe`（:61-80）遇 `\` 跳 2 字节且**保留反斜杠**（结构检测不负责渲染）；`is_table_header_line` 要求至少一个非空段（:88-90）；`is_table_delimiter_segment` 允许 `:` 前后缀、中间 `-` **≥3 个**（:95-103），`is_table_delimiter_line` 要求**每段**合格（:108-111）。`FenceTracker`：>3 前导空格不算围栏（:157-165）、剥 blockquote、闭合须同字符且长度 ≥ 开围栏（:169-177），`FenceKind::{Outside,Markdown,Other}` 用来跳过 `sh`/`rust` 里的 `|`（:124-131）。

### 2.3 列宽：先满配，再按列种类让步

常量（`markdown_render.rs:108-111`）：`TABLE_COLUMN_GAP=2`、`TABLE_CELL_PADDING=1`、表头分隔 `━`、表体分隔 `─`；预留 = `prefix + (cols-1)*2 + cols*1*2`（`:1558-1567`）；`min_column_width = 3`（`:1589`）。

- 列种类（`:1672-1680`）：长 token（`display_width >= 20`，`:1651`）数 ≥ 其余词数 → `TokenHeavy`；否则 `avg_words>=4.0 || avg_cell_width>=28.0` → `Narrative`；余 `Compact`。
- 软地板（`:1699-1707`）：`Narrative/TokenHeavy` 固定 **16**；`Compact` 取 `max(表头 token 宽, min(表体 token 宽,16))`。
- 瘦身顺序（`:1716-1760`）：`TokenHeavy → Narrative → Compact`，同种内对 cap 二分、**先砍离地板最远者**；总宽连 `cols*3` 都塞不下 → `None`（`:1598-1601`）。
- 回退阶梯（`:1477-1516`）：有行 → 键值记录；无行 → 原始管道行（`:1922`）。
- 多列表里无管道的表体行会把后续段落吞成一格行，故识别为 **spillover** 在表后按纯文本补出（`:1431-1438`）。

### 2.4 键值表：把表竖过来，而不是压成窄条

常量（`table_key_value.rs:18-27`）：`FIELD_LEADING_PADDING=1`、`FIELD_GAP=2`、`MIN_VALUE_WIDTH=3`、`MIN_ALIGNED_COMPACT_VALUE_WIDTH=12`、`MIN_ALIGNED_EXPANSIVE_VALUE_WIDTH=24`、`MIN_SCANNABLE_NARRATIVE_WIDTH=12`、`CRAMPED_EXPANSIVE_CELL_LINES=4`、`CATASTROPHIC_NARRATIVE_CELL_LINES=7`、`STACKED_VALUE_INDENT=2`。

触发（`:31-71`）：受影响行数 ≥ `rows==1 ? 1 : max(2, ceil(len/3))`；受影响 = `Compact` 列出现宽于列宽的 token，或 `TokenHeavy` 列宽 <12 且同样现象。「憋屈」（`:73-96`）：≥2 个非 Compact 格折到 ≥4 行，或 `Narrative` 列宽 <12 却 ≥7 行。布局：`1 + label_w + 2 + min_value_w <= 可用宽` 才对齐（`:119-121`），否则堆叠；记录间插满宽 `─`（`:139-145`）。

### 2.5 行内指令：语法与四条安全边界（`inline_directives.rs`）

```
:codex-file-citation{path="src/x.rs:12"}  → 链接，标签=格式化目标（无作者标签，:236）
:codex-followup[点这里]                    → 只留 label 作行内文本（:194-215）
```

1. **只认扫描位**：逐 `:` 匹配，前是 `:` 或落在字面区间（code/html/链接/图片/引用定义）即跳过（`:67-99`）。
2. **掩码不改长**：命中把 `start+1..end-1` 换成 `x`（`:143-146`），偏移与 flanking 全不动。
3. **预算按输入比例**：`scan_budget = [len*4; 2]`，preferred/fallback 共享（`:89`）；citation 偏好字面引用、其余偏好反斜杠（`:102-109`）；`\:` 因反斜杠奇偶不认（`:122-128`）。
4. **空字段即拒**：citation `path` 空、followup 标签 trim 后空 → 跳过（`:129-142`）；path 先 `%`→`%25`、`#`→`%23`、`?`→`%3F`，尾部 `:line:col` 冒号转 `%3A`（`:228-235`）。

**路径分裂**：指令只在流式/历史重绘生效（`streaming.rs:71`），纯完成态入口不含（`markdown_render.rs:418-438`）；`/copy` 走 `followup_labels`（`transcript.rs:99`、`copy_mode.rs:43`）只换 followup、其余保留源 Markdown。

### 2.6 链接：本地「显示截短」与网页「目的地可藏」

| 判定 | 规则 | 行号 |
|---|---|---|
| 本地像路径 | `file://` `/` `~/` `./` `../` `\\` `X:/` `X:\` | `local_links.rs:29-41` |
| `#L..` 片段 | `^L\d+(C\d+)?(-L\d+(C\d+)?)?$` | `local_links.rs:23-27` |
| `:line[:col]` | `:\d+(?::\d+)?(?:[-–]\d+(?::\d+)?)?$`，必须贴结尾 | `local_links.rs:15-21`、`:117-122` |
| 显示 | 分隔符转 `/`、UNC→`//srv/sh`、绝对路径按会话 `cwd` 剥前缀，剥不动就保持绝对 | `local_links.rs:130-138`、`:191-206` |
| 网页目的地 | 多路复用器/`screen`/`tmux`/`dumb`/Terminal.app/Warp **保留**；Ghostty/iTerm2/WezTerm/Kitty/VSCode/Alacritty/WindowsTerminal/Konsole/GNOME/VTE **只显标签** | `web_links.rs:64-95` |
| 非终端/`STY` | 一律保留目的地（标签不得藏掉唯一可用 URL） | `web_links.rs:102-109` |

裸 URL 也当链接（`web_links.rs:23-55`），但 `MAX_HYPERLINK_DESTINATION_BYTES = 8 * 1024`（`terminal_hyperlinks.rs:41`），超限**不建链接**退纯文本（`:627-632`）、剥控制字符、只认 `http/https` 且有 host（`:612-619`）；尾部标点按括号配平裁掉（`,` `.` `;` `!` `'` `"`、不配对闭括号、`)`/`]`/`}`/`>` 后跟 `?`，`:560-609`）。

### 2.7 列表间距与任务列表

`ListSpacing` 三档（`list_spacing.rs:7-15`）：`AfterMultiline`（终端回滚默认）/`Compact`（流式不加分隔）/`Uniform`（有一项多行则全加）。`UniformList::finish` 只删本列表插入的临时分隔行且保留链接元数据（`:25-43`）——间距是布局策略，不是内容。任务列表（`task_lists.rs:8-38`）把 `• ` 换成 `☑`/`☐` 并按显示宽重排前缀，同时记 `copy_marker = "[x] "/"[ ] "`（`:26`），复制时还原成 GFM 勾选。

### 2.8 数学：为什么必须结构化

上限 `MAX_MATH_BYTES = 4096`（`math.rs:17`）、布局 `MAX_ROWS=16`/`MAX_COLUMNS=256`（`math/render.rs:10-11`）。识别（`math.rs:81-95`）：`$$`/`\[` 为 display，`\(`/`$` 为 inline；转义跳过（`:98`）；`$` 后接空白或 `(`/`{` 放弃（`:102-106`）；`$5`/`$USD` 类假阳性被拒（`:210-224`）；inline 含换行不认（`:207-209`）。display 须独占一行，否则只在前 4 KB 找 closer（`:107-152`），行内有文字前缀的 `$$` 标 `rejected_display` 并把配对限制在窗口内，避免 shell `$$` 把整段响应钉成可变块（`:158-182`）。掩码为等长 `$`（`:247-252`），完成后按 `replacements` 逐事件回填（`:257-296`）。

**结构化的理由**（`structured.rs:1-4`）：分数、上下标、`\sum` 上下限、`\begin{aligned}` 在等宽终端只有多行对齐才可读；`display_operator`（`:51-83`）先定宽再居中上下限，aligned 保留列对齐（`:85-90`）。多行布局仅在 `preferences.math`、未被容器包住、每行宽 ≤ `width-4` 时采用，否则**逐字回退源文本**（`math.rs:230-245`）。

### 2.9 Mermaid：替换语义 + 回退

只画**已完成**围栏，且要求真有闭合围栏（CommonMark 到 EOF 也发 `End`）——`has_closing_fence` 判定闭合串 marker 数 ≥ 开围栏数（`mermaid.rs:19-37`）；默认宽 120（`:39-40`）；三类失败文案 `Unsupported`/`TooWide`/`Limit`（`:44-54`）。**回退**：失败先打一行 dim 提示，再把围栏源码按 `mermaid` 语法高亮贴出（`:55-60`）——降级成代码块而非消失；成功路径只用主题取色的 `Node/Edge/Text` span，**绝不产出终端控制序列**（`:1-5`）。

### 2.10 复制：从渲染行反推 Markdown（**表格不是 TSV**）

```rust
struct CopyLine { prefix, continuation, item_prefix, code, code_source, quote_source,
                  table: Option<table::TableLine>, rule, heading, hard_break, omit, runs, literal_ranges }
// markdown_copy.rs:97-116；MAX_COPY_DEPTH = 64（:38）限定内联栈与容器前缀深度
```

- 注释明确不复现完整响应源文：rich selection 保留选中词与**成对**标记（`:1-6`），全文另有 `/copy`。
- 整段字面（无任何语义标记）→ `CopyFormat::PlainText` 原样（`:347-355`、`:405-415`）。
- 表格（`markdown_copy/table.rs`）：按 `(row, column)` 收格子（`:111`、`:221-226`），**行内多视觉片段以单空格 `join(" ")`**（`:198`），行输出 `| a | b |`（`:232`，分隔串 `" | "`），表头后紧跟按原 alignment 拼的分隔行（`:234-238`）。**结论：复制结果是 GFM 源表 + 对齐行，没有 `\t`，不是 TSV。**
- 单格 + 整选中 + 字面 → 裸文本 + `PlainText`（`:165-178`）。
- 转义：代码格内 `|`→`\|`（`markdown_copy.rs:193-197`）；普通文本走 `escape()`，转义集 `` \ ` * _ [ ] < > ~ # | & ! `` 加行首 `=`/`-`/`+` 与「数字后接 `.`/`)`」（`:604-625`）；行内代码用 `fence()` 定反引号数并补 padding（`:198-212`、`:584-593`）；链接目的地再编码 `\`→`\\`、`&`→`&amp;`、`<`→`%3C`、`>`→`%3E`、`|`→`%7C`、`\n`→`%0A`、`\r`→`%0D`，输出 `[label](<dest>)`（`:226-236`）。
- 输出前剥控制字符只留 `\n`/`\t`（`:568`；PlainText 分支 `:468-471`）；硬换行在 Markdown 输出里补两空格（`:302-315`），PlainText 再去掉（`:324-330`）。

### 2.11 高亮与行截断：两套上限

| 项 | 值 | 行号 |
|---|---|---|
| 总字节 | `MAX_HIGHLIGHT_BYTES = 512 * 1024` | `render/highlight.rs:640` |
| 总行数 | `MAX_HIGHLIGHT_LINES = 10_000` | `render/highlight.rs:643` |
| **单行** | `MAX_HIGHLIGHT_LINE_BYTES = 4 * 1024` | `render/highlight.rs:646` |
| 聚合预检 | `exceeds_highlight_limits(bytes, lines)` | `render/highlight.rs:653` |
| 主题版本 | `THEME_REVISION: AtomicU64` + `syntax_theme_revision()` | `render/highlight.rs:68`、`:282` |

任一越限 → `None` → 回退纯文本，一行一 Line，且用 `lines()` 而非 `split('\n')` 避免尾部幽灵空行（`render/highlight.rs:678-685`、`:740-753`）。行截断按 grapheme 累加显示宽，溢出预留 1 列给省略号（`line_truncation.rs:47-58`、`:76-88`），`max_width==0` 返回空行（`:14-16`、`:80-82`）。

---

## 三、gaea 现状核对（真实缺口）

| 能力 | codex | gaea 现状（file:line） | 缺口 |
|---|---|---|---|
| GFM 表 | 原生 + 列宽/键值回退 | `ChatMarkdown.tsx:68-82` 纯 `<table>` + `overflowX:auto`；`MarkdownContent.tsx:90-105` | 无列宽策略、无键值回退 |
| **表格复制** | GFM 源表（含对齐行），`\|` 转义 | `ChatCodeBlock.tsx:26` 只复制**代码块**，`clipboard.ts:11` 是唯一出口 | **缺**：无表格复制、无 `escape()`/对齐行 |
| 表格数据方向 | — | `tableData.ts:22-61` 是**反向**（CSV/TSV → Markdown），`:56` 已转义 `\|` | 方向互补，口径可复用 |
| md 围栏解包 | `md/markdown` + 表头/分隔配对才扒 | 无；`remarkDisableCodeIndented.ts:12-15` 是另一件事（禁四空格缩进码块） | **缺**：`` ```md `` 包表仍渲染成代码 |
| 行内指令 | `:codex-file-citation{}`/`:codex-followup[]` | 无；`remarkFileLinks.ts:33-61`、`remarkMemCitations.ts:32-60` 只包**已有文本** | **缺**：指令语法、掩码、编码、`\:` 奇偶 |
| 本地链接截短 | 按会话 cwd 剥前缀 + `#L/:line` 归一 | `Markdown.tsx:585` `mdUrlTransform` 只做协议白名单；`fileLinks.ts` 管识别 | **缺**：显示层 cwd 短路径与 `#L12C3` 归一 |
| 网页目的地 | 终端启发式 | `ChatMarkdown.tsx:63-67` 恒显标签 + `target=_blank` | 不适用（浏览器恒可点），见风险 3 |
| 高亮上限 | 512 KB / 10 000 行 / 单行 4 KB | `codeHighlight.ts:76-90` 全文 `hljs.highlight`；`HlCode.tsx:14` 全文传入 | **缺**：单行 4 KB 与总量门限 |
| 数学 | 4 KB 上限 + ASCII 回退 | `mathText.ts:17-19`、`:23-36`；`rehype-katex` 真排版 | 前端更优，**缺** `MAX_MATH_BYTES` 与 KaTeX 失败回退 |
| Mermaid | 闭合围栏判定 + 降级源码 | `ChatMermaid.tsx:44` 直接 `render`；`:60-76` 只画完整行前缀；`:104-110` 失败提示 | **缺** `hasClosingFence`（EOF 半截被当完成态判失败） |
| 任务列表 | `☑/☐` + copy `[x]` | `remark-gfm` 原生 checkbox；`MarkdownContent.tsx:113` | 复制得到 HTML 复选框，非 GFM 文本 |
| 流式切点 | 表格 holdback | `markdownCut.ts:18-46`：`\n\n` 切 + 行首 ``` 翻转，落在未闭合 fence 内回退 | **缺**：未闭合**表格**（表头出、分隔行未到）会被切在中间 |
| sanitize 面 | 无 raw HTML 渲染 | `sanitize.ts:91-131` 只挂 `Markdown.tsx:628`；`ChatMarkdown.tsx:29` **无** `rehypeRaw` | 面窄但一致，见风险 1 |

---

## 四、可迁移三档

| 档 | 机制 | 落法 |
|---|---|---|
| **直接照搬** | 单行 4 KB / 512 KB / 10 000 行门限；`MAX_COPY_DEPTH=64`；`escape()` 转义集与 `fence()`；`\|` 单元格转义；`[x] `/`[ ] ` 复制标记；8 KB URL 超限退纯文本 | 纯函数，无框架耦合 |
| **改造成插件** | ① `md/markdown` 围栏解包 → **rehype** 转换（`pre>code.language-md` 且 body 含 header+delimiter 对 → 重解析）；② 行内指令 → **remark** 插件（先收字面区间、再按 `:` 扫、掩码、生成 `link`/`text`）；③ 表格复制 → 组件层从 DOM `<table>` 取 `(row,col)` + `align`，按 codex 口径序列化 | 与 `remarkFileLinks`/`remarkMemCitations` 同架构（`visit` + `SKIP_TYPES`） |
| **不搬** | 终端特性：`web_links` 身份启发式、`osc8`、`━/─`、键值表的「憋屈/灾难」阈值、`ListSpacing` 三档、`math/render` 多行布局、grapheme 截断 | 浏览器有 DOM/KaTeX/CSS 溢出；硬搬会把自适应退化成「固定 16 列地板」 |

---

## 五、施工卡

**落点**：`gaea/lib/copyMarkdown.ts`（DOM→Markdown 单源）、`gaea/lib/remarkTableFences.ts`（先于 `remarkGfm`）、`gaea/lib/remarkInlineDirectives.ts`、`gaea/lib/codeHighlight.ts`（门限）、`components/ChatCodeBlock.tsx`（`TableToolbar`：复制/切 TSV）、`gaea/lib/markdownCut.ts`（表格 holdback）。

```ts
// copyMarkdown.ts
export type TableSnapshot = { rows: string[][]; align: ("---"|":--"|"--:"|":-:")[] };
export function snapshotTable(el: HTMLTableElement): TableSnapshot;
export function tableToMarkdown(t: TableSnapshot): string;      // `| a | b |` + `|---|:--|`
export function escapeMarkdownCell(s: string, inCode: boolean): string;
export function findStableCut2(text: string): [string, string]; // 表格 holdback 版

// remarkTableFences.ts / remarkInlineDirectives.ts
export function remarkTableFences(o?: { langs?: string[] }): (tree: MdNode) => void;
export function remarkInlineDirectives(): (tree: MdNode) => void;

// codeHighlight.ts
export const MAX_HIGHLIGHT_BYTES = 512 * 1024;
export const MAX_HIGHLIGHT_LINES = 10_000;
export const MAX_HIGHLIGHT_LINE_BYTES = 4 * 1024;
export function exceedsLimits(bytes: number, lines: number, longest: number): boolean;
```

**开关与默认**（对齐 `tui_rendering.rs:22-30` 全 `true`）

```ts
export const markdownPrefs = {
  tables: true, math: true, mermaid: true, lists: true,
  tableCopyFormat: "markdown" as "markdown" | "tsv",  // 默认 markdown = codex 口径
  inlineDirectives: false,                            // 默认关：指令是 codex 私有协议
};
```

**可自动断言验收**

1. **表格复制 = GFM 源表**：`| 名称 | 命令 |` + `| a\|b | x |` → 第 1 行 `| 名称 | 命令 |`、第 2 行 `|---|---|`、含 `|` 的格输出 `a\|b`，且 `!out.includes("\t")`（**不是 TSV**）。
2. **对齐行**：`align` 全 `":--"` 时第 2 行严格为 `|:--|:--|`。
3. **2000 字符长行不触发高亮且原样显示**：`exceedsLimits(2000, 1, 2000) === false`；5000 字符单行 → `true`，`HlCode` 快照无 `hljs-` 类、正文逐字相等。
4. **围栏解包**：`` ```md `` + `| a | b |`/`|---|---|`/`| 1 | 2 |` → `<table>`；去掉分隔行 → 仍是 `<pre><code>`。
5. **指令**：`:codex-followup[点这里]` → 文本 `点这里`；`\:codex-followup[x]`、`:codex-file-citation{path=""}`、行内 code 内的指令 → 全部原样。
6. **流式切点**：`"段\n\n| a | b |\n---"` 的稳定段不含表头行（切点落在表头之前）。
7. **复制兜底诚实**：`copyToClipboard` 返回 `false` 时不翻转「已复制」（沿用 `ChatCodeBlock.tsx:26-31`）。

**回退**：插件异常 → 原文直出（同 `source_tables::preserve` 思路）；表格快照失败 → `textContent` 逐行 `\n` + `PlainText`（对应 `table.rs:208-219`）；高亮越限 → 纯文本 `<pre>`（`highlight.rs:743-753`）；Mermaid 失败 → 源码块 + 一行提示（`mermaid.rs:44-60`）。

---

## 六、风险

1. **XSS 面被插件抬高**：新增「指令→链接」「围栏→表格重解析」两条改写 AST 的路径，一旦把不可信串塞进 `url` 就绕开 `mdUrlTransform`（`Markdown.tsx:585-588`）。铁律：指令路径先做 `%`/`#`/`?` 编码（`inline_directives.rs:228-235`），URL 一律过 `mdUrlTransform`；`ChatMarkdown.tsx:29` 目前不挂 `rehypeRaw`，别为解包顺手挂上。
2. **性能**：列宽瘦身是「逐列扫 token + 对 cap 二分」（`markdown_render.rs:1651`、`:1738-1760`），搬进 React 必须 `useMemo` 到 `(text,width)`；高亮无门限时（`codeHighlight.ts:76-90`）单个 200 KB 单行会卡主线程，先落 4 KB 单行门限再谈懒加载。
3. **一致性**：同一文本在三条链（`ChatMarkdown.tsx:29`、`Markdown.tsx:627-628`、`MarkdownContent.tsx:50`）插件集不同，复制结果与所见会分叉；表格复制格式必须与展示口径同源，否则「所见即所复制」不成立。

## 七、一句话

这套东西的价值不在「把 Markdown 画出来」，而在**每行渲染结果都带可反推的源坐标与语义标记**：表格能还原成带对齐行的 GFM（不是 TSV），指令能掩码替换而不动偏移，超限（4 KB 单行 / 512 KB 总量 / 8 KB URL / 4 KB 公式）一律**降级不清空**。gaea 要补的不是渲染器，是**复制语义 + 上限纪律**这两条横切面。

---

证据自查：`table_detect.rs:102`（分隔段 `-` ≥3）、`markdown_copy/table.rs:232`（表格复制输出 `| … |` 管道行，非 TSV）、`render/highlight.rs:646`（`MAX_HIGHLIGHT_LINE_BYTES = 4 * 1024`）
