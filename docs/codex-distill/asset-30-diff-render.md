# 资产 30 · diff 渲染施工级：行模型、折叠、退让与工具卡绑定

> 源码：`tui/src/diff_render.rs`（2745 行，主战场）、`tui/src/render/highlight.rs`（限流常量）、
> `tui/src/history_cell/patches.rs`、`tui/src/history_cell/activity_{group,details,preview}.rs`、
> `tui/src/bottom_pane/apply_patch_header.rs`、`tui/src/line_truncation.rs`、`tui/src/get_git_diff.rs`。
> 对照：gaea `frontend/src/gaea/lib/{diff,diffRender,diffHighlight,planDiff}.ts`、`…/components/{ChangesDiff,ChangesPanel}.tsx`。
> 结论先行：codex 的 diff 是**「单列行号 + 硬换行 + hunk 间 ⋮ 分隔」三段式**，没有「保留 N 行上下文折叠中段」这回事；
> 折叠发生在**卡片层**（预览只给 3 行、且预览里上下文行整体不画）。gaea 的 3/6 折叠是另一套模型，不要按 codex 改。

---

## 一、行模型与常量总表

`FileChange` 只有三态：`Add{content}` / `Delete{content}` / `Update{unified_diff}`（`diff_render.rs:424-430`）。
`Update` 走 `diffy::Patch::from_str`，行类型就是 diffy 的 `Insert / Delete / Context`（`diff_render.rs:612`、`620-624`）。
**没有** file-header / hunk-header 行类型——`@@` 只被当作 hunk 边界，头文本从不渲染（见 §二第 3 点）。

| 常量 | 值 | 位置 |
|---|---|---|
| `TAB_REPLACEMENT` | `"    "`（4 空格） | `diff_render.rs:53` |
| `TAB_WIDTH` | `4` | `diff_render.rs:55` |
| `DETAIL_PREVIEW_LINES` | `3` | `activity_preview.rs:7` |
| `MAX_HIGHLIGHT_BYTES` | `512 * 1024` = 524288 | `highlight.rs:640` |
| `MAX_HIGHLIGHT_LINES` | `10_000` | `highlight.rs:643` |
| `MAX_HIGHLIGHT_LINE_BYTES` | `4 * 1024` = 4096 | `highlight.rs:646` |
| `DIFF_COMMAND_TIMEOUT` | `30s`（git diff 采集） | `get_git_diff.rs:18` |
| 行号列宽 | `max_line.to_string().len()`，0 时取 1 | `diff_render.rs:1081-1087` |

配色双轨：真彩 `DARK_TC_ADD_LINE_BG_RGB=(33,58,43)`/`DARK_TC_DEL_LINE_BG_RGB=(74,34,29)`、
`LIGHT_TC_ADD_LINE_BG_RGB=(218,251,225)`/`LIGHT_TC_DEL_LINE_BG_RGB=(255,235,233)`（`64-67`）；
256 退让 `22/52/194/224`（`73-76`）。ANSI-16 时**整个背景色层被跳过**（`Ansi16 → None`，`169-175`）。

## 二、渲染管线：单列行号、硬换行、hunk 分隔

**1. 行号是单列，不是双列。** `Insert→new_ln`（`704`）、`Delete→old_ln`（`729`）、`Context→new_ln`（`759`）：
一端被选、另一端不画，「两侧错位」这个问题在模型层就不存在。

**2. 列宽靠一次预扫描定死，不抖动。** 进渲染前先遍历**全部 hunk**求 `max_line_number`（`613-643`），
再 `line_number_width(max_line_number)`（`654`），整个补丁共用同一宽度。行号右对齐 + 一个空格：`format!("{line_number:>gutter_width$} ")`（`962`）。
`gutter_width = line_number_width.max(1)`（`898`）。`Add/Delete` 分支没有 hunk，用整文件行数定宽（`551`、`582`）。
**这是 gaea 最该照搬的一条**：列宽必须由整块最大值决定，否则长文件尾部 4 位行号会让正文整体右移。

**3. hunk 之间只有一个 `⋮`。** 非首个 hunk 前推一行 `format!("{:width$} ", "")` 的 gutter 底 + `"⋮"`（`660-671`），
**不打印 `@@ -a,b +c,d @@`**。文件级头由 `render_changes_block` 合成：单文件时 `"• Edited path (+a -b)"`（`465-475`），
多文件时 `"• Edited N files (+a -b)"` 再逐文件 `"  └ path (+a -b)"`（`477-497`），每文件正文统一 4 空格缩进（`504-516`）。
`calculate_add_remove_from_diff` 解析失败时返回 `(0,0)`（`826-829`）——头里会显示 `(+0 -0)`。

**4. 换行是硬换行（Hard），续行不带符号与行号。** `wrap_policy = LineWrapPolicy::Hard`（`942`）；可用列
`available = width.saturating_sub(gutter_width + 2).max(1)`（`952`）——减的是 gutter 加「行号后空格 + 符号」两列。
续行前缀 `format!("{:gutter_width$}  ", "")`（`947-951`）：gutter 个空格 + 2 空格，**无 `+/-`、无数字**，接 `source.continuation_indent`（`951`）；
制表符拼 span 时换成 4 空格（`938`）。

**5. 折叠：只在卡片层，不在行层。** `create_diff_summary_with_links`（`373-380`）`preview_lines=None` → **上下文全画**（`750-779`）；
`create_diff_preview_with_links`（`383-390`）`preview_lines=Some(3)` → `Context` 分支**直接 `continue`**（`751-755`），
顶部按 `out.len() >= limit` 早 break（`657-659`、`693-695`），末尾 `lines.truncate(limit)` 兜一次（`513-515`）：
卡面 3 行 = 只显 +/‑，上下文零成本丢弃，**不是「收起中段」**。
所以 codex **没有「保留 N 行上下文、中段折叠」的阈值**；`⋮` 是 hunk 边界标记，不是折叠按钮（无交互态）。

## 三、超长行与超限补丁：两套闸门，别混

**闸门 A（行级：硬换行，不截断）。** diff 正文**没有任何字节/字符截断**：长行被 `wrap_styled_spans` 切段后完整渲染（`953`、`996-1079`）。
截断 + 省略号只出现在**活动预览行**：`truncate_line_with_ellipsis_if_overflow`（`line_truncation.rs:76-101`，先测宽、溢出才截、
在 `max_width-1` 处补 `"…"`，`88-95`），调用点只有 `activity_preview::clipped_line`（`activity_preview.rs:18-20`，宽度=面板宽）
与补丁失败卡紧凑态（`patches.rs:99-103`）。`activity_preview.rs:17` 点题：**「Clip a preview row without teaching selection to copy text that is currently hidden.」**
——裁剪只作用于视觉行，选区按 `HyperlinkLine.source`（`968-980`，含 `prefix_bytes`+`range`）还原逻辑行，于是「看到被截断、复制到全文」。这条必须照搬。

**闸门 B（块级：超限整体退让高亮）。** 统计全补丁 `total_diff_bytes / total_diff_lines`（`625-626`），超限则 `diff_lang = None`（`648-652`）：
**整个补丁不做语法着色**（照画，只是无色）。阈值 `exceeds_highlight_limits`：`bytes > 512KiB || lines > 10000`（`highlight.rs:653-654`）；
单行超 4KiB 也在 `highlight.rs:678-685` 被拒（三闸一起返回 `None`）。
`Add/Delete` 分支是「整篇一次高亮」（`550`、`581`），没有块级预闸门，靠 `highlight.rs:678-685` 内部兜底。

## 四、二进制与「不识别的文件」怎么显示

- **补丁侧直接拒**：apply_patch 不允许打到二进制目标（`*** Add File: binary.dat` / `*** Move to: binary.dat` 均断言失败，
  `codex-rs/apply-patch/src/lib.rs:1357-1363`），所以二进制文件**进不了 diff 渲染器**。
- **图片走另一条车道**：`ViewImageHistoryCell::image_path()` / `ImageGenerationHistoryCell::saved_path`（`patches.rs:172-174`、`259-261`），
  终端里不画像素，只出 `Viewed image <name>`（`176-187`）或 `Generated Image:` + `Saved to:`（`218-242`）。
- **解析失败静默空体**：`Update` 分支是 `if let Ok(patch) = diffy::Patch::from_str(unified_diff)`（`612`），**没有 else**。
  解析不了时正文一行不出，头里因 `(0,0)` 显示 `(+0 -0)`（`826-829`）。用户看到的是「文件被编辑了，但没有内容」。
  这是 codex 的**真实空洞**：没有「非文本文件」「格式不认识」「补丁被拒」三档文案区分。gaea 必须补三档，别学这里。
- **路径降级**：同仓才相对化，跨仓退 `~`，否则绝对路径（`792-813`）；移动用 ` → dst`（`453-456`），高亮语言按**目标**扩展名取（`499-502`）。

## 五、补丁头与「编辑动作摘要」的关系

`*** Begin Patch` 语法在 `codex-rs/apply-patch`（`invocation.rs:419` 起的 `wrap`/解析测试即语法样本：
`*** Add File:`/`*** Update File:`/`*** Move to:`/`*** Delete File:`/`@@`/`+`/`-`）。TUI 侧**不解析补丁文本**，
只拿结构化 `HashMap<PathBuf, FileChange>`；补丁头只服务审批卡：

| 审批卡元素 | 取值 | 位置 |
|---|---|---|
| `Thread: <label>` | 可选，有则先出一行 + 空行 | `apply_patch_header.rs:13-22` |
| `Description: <reason>` | 空/缺省回落 `"Apply proposed file edits"`，斜体 | `24-32` |
| `Destination: <path>` | 由 `changes` 的 path 与 `Update{move_path}` 的 dst 合并，排序去重；空则 `"unavailable"` | `34-71` |

**「编辑动作摘要」= 谁的计数**：卡面头部的 `(+a -b)` 来自 `line_counts`（`424-430`），`Add` 取 `content.lines().count()`，
`Update` 取 `calculate_add_remove_from_diff`。`DiffSummary::From` 是另一套呈现（审批/汇总用）：按 path 排序、空行分隔、
`path (+a -b)` + `InsetRenderable` 左缩进 2（`336-361`）。

## 六、活动分组与工具卡绑定/缓存键

- `ActivityGroup<T> { calls: Vec<T>, details: ActivityDetails }`（`activity_group.rs:11-14`）；`details` 用**「在第几个 call 之后」的位置键**存内联明细：
  `push(cell)` 记 `self.calls.len()`（`24-26`），`prepend(older)` 把旧位置整体 `+= older_calls` 再前插（`28-31`，`activity_details.rs:23-29`）——压缩/翻页时位置不漂移的关键。
- **编辑动作不参与活动分组。** `ActivityGroup` 只有两个使用者：`ExecCall`（`exec_cell/model.rs:80`）与 MCP 工具调用（`computer_activity.rs:11`）。
  补丁是**一格一 cell**：`"patch:{uuid}"`（`patches.rs:21`、`72`）、失败卡 `"patch-failure:{uuid}"`（`80`）。
  所以「N 次编辑聚成一条 + 计数」在 codex 里**不存在**；计数只发生在**一次 patch 内的文件维度**（`Edited 3 files`，`diff_render.rs:477-479`）。
- **绑定**：`HistoryCell` 三个渲染口都吃 `width` 现算，不缓存行：`compact_hyperlink_lines` → 预览 3 行（`patches.rs:31-38`）、
  `display_hyperlink_lines` → 全量（`44-50`）、`raw_lines` → 纯文本固定宽 `RAW_DIFF_SUMMARY_WIDTH`（`56-62`）。
- **缓存键**：diff 行**无缓存**。唯一「每帧一次」是 `DiffRenderStyleContext`（`197-202`），在 `render_change_with_preview` 顶上取一次（`546`），
  注释明确「每帧取一次、不要每行取，避免主题切换串色」（`215-223`）；`render` 与 `desired_height` 各调一次 `render_change`（`322-333`）
  ——同一帧算两遍，是终端排版代价，Web 不要复制。

## 七、颜色与语法高亮的交互（重算边界）

- 高亮的**单位是 hunk，不是行**：hunk 全部行拼成 `hunk_text`，一次 `highlight_code_to_styled_spans`（`676-686`），
  校验 `syntax_lines.len() == h.lines().len()` 才对位，否则整 hunk 无色（`687`）；注释点题：保留 syntect 跨行 parser 状态（`674-675`）。
- 着色不改 diff 底色，只调前景可读性：`readable_color_on(fg, line_style.bg)` + `remove_modifier(DIM)`（`917-924`）。
- 背景/符号/行号三层分离：`style_line_bg_for`（`912`）、`style_sign_add/del`（`1283-1315`）、`style_gutter_for`（`1258-1282`）；
  整行无底色时 gutter 自身底色也清掉（`944-946`）。
- 主题/色深探测是**环境读**：`diff_theme()` 探终端背景（`1100-1102`），`diff_color_level()` 读 `supports-color`/`WT_SESSION`/`FORCE_COLOR`
  （`1112-1120`），Windows Terminal 被特判升到真彩（`137-142` 注释）。

---

## 八、gaea 现状核对（真实缺口，带 file:line）

gaea 侧资产：`lib/diff.ts`（72 行，全量矩阵 LCS）、`diffRender.ts`（152，改蓝配对+字符高亮+折叠）、
`diffHighlight.ts`（80，Lezer 单行着色）、`planDiff.ts`（181，写工具→DiffRow / git unified 解析）、
`ChangesDiff.tsx`（227，唯一查看器）、`ChangesPanel.tsx`（474，三层折叠+调用列表）。消费者 4 处、渲染器却 3 套：
`ChangesPanel.tsx:334`、`GitPanel.tsx:265`（传 path 有高亮）、`CommitGraphView.tsx:298`（**path=undefined，静默丢高亮**）、
`ScheduleDiffCard.tsx:87`（手写渲染，`ROW_CAP = 60` 于 `24`）。

| # | 缺口 | 证据 | codex 对应 |
|---|---|---|---|
| G1 | **完全没有行号列**，只有符号列 `w-4` | `ChangesDiff.tsx:131-136` | `diff_render.rs:962`（单列行号 + 预扫描定宽） |
| G2 | marker 列固定 `w-9`、无 truncate/title，长 ref（`Sheet1!AA100`）溢出或挤压正文 | `ChangesDiff.tsx:137-144`、`86-93` | 无对应（codex 用整块最大宽） |
| G3 | 折叠阈值 `FOLD_KEEP=3`、`run > keep*2`（即 >6）收中段 | `diffRender.ts:117`、`141-148` | codex 无行层折叠；预览=3 行且丢弃全部上下文（`diff_render.rs:751-755`）——**两套模型，别对齐数值** |
| G4 | **双重省略口径不一**：先折中段（`⋯ 已折叠 N 行`，`ChangesDiff.tsx:78`），再 `MAX_ROWS=300` 截尾（`28`、`179-183`），用户看到两个不同数字 | 同左 | codex 只有 `⋮` 一种省略（`670`），无双计数 |
| G5 | 超长行按 `slice(0, 2000)` 切片进 DOM，**折叠态下截断部分不可选中复制**（口径收紧：`expandedLong` 可展开到 20000，`ChangesDiff.tsx:31-32` 定常量、`:172` 有明示文案，因此真实死角只有「>20000 字符的行永久不可复制」+「跨选区一次性复制时拿到的是截断文本」两处） | `ChangesDiff.tsx:102-104`、`157-173` | `diff_render.rs:968-980` + `activity_preview.rs:17` 注释：视觉裁剪、选区取逻辑源 |
| G6 | 三处阈值互不相干且**互相架空**：`MAX_CHAR_DIFF=240`（`diffRender.ts:70`）＜ `pairCanHighlight` 的 20000（`ChangesDiff.tsx:105-107`）＜ `MAX_HIGHLIGHT_LEN=600`（`diffHighlight.ts:49`） | 同左 | 单一闸门 `exceeds_highlight_limits`（`highlight.rs:653-654`） |
| G7 | 高亮**逐行 parse、结果不缓存**，且写在 render 的 `map` 内；一次「展开长行」setState 触发整块重算 | `diffHighlight.ts:55-79`、注释 `6`；`ChangesDiff.tsx:151`、`162-167` | 每 hunk 一次 + 超限整块退让（`diff_render.rs:676-687`、`648-652`） |
| G8 | `unfolded` / `expandedLong` 用**数组下标**做键，同文件二次写入后数据变、下标语义漂移，展开态错位 | `ChangesDiff.tsx:61-62`（`useState<Set<number>>`） | cell 实例化 + `patch:{uuid}` 身份（`patches.rs:21`、`72`），无下标依赖 |
| G9 | 二进制退让只在 git 文本层做了（`planDiff.ts:147-149`），渲染层 `kind="none"` 只有一行 note，**「非文本」「解析失败」「补丁被拒」不分档** | `planDiff.ts:179`、`ChangesDiff.tsx:222-226` | 同样薄弱（`diff_render.rs:612` 无 else），但 gaea 是 Web，应补三档 |
| G10 | `MAX_XLSX_SHEET_CELLS=200` 等各模块自定上限，无统一「补丁规模」闸门 | `xlsxCellDiff.ts:56`、`ChangesDiff.tsx:28-32`、`ScheduleDiffCard.tsx:24` | 块级统一闸门（`highlight.rs:640-646`） |

小结：**算法层弱在规模（全量矩阵 LCS，`diff.ts:27`），机制层弱在「视觉/逻辑双轨」与「统一闸门」。**
codex 快不是渲染聪明，而是 ①单列行号省一列 ②超限直接放弃着色 ③预览层直接丢上下文 ④行层从不截断（换行代替）。

## 九、可迁移机制三档

**A. 直接照搬（改配置即可，0 算法风险）**
1. 单列行号（`Insert/Context` 新号、`Delete` 旧号，`704/729/759`）——比双列省 4-6 字符，窄面板救命；
2. 列宽预扫描（`613-643`+`654`+`962`）——一次求 max 定宽，消掉「行号列宽抖动」；
3. hunk 间 `⋮` 分隔（`660-671`）——`@@` 收进 `title` 提示，不占正文行；
4. tab→4 空格（`53`、`938`）；`DiffRenderStyleContext` 每渲染一次、不每行取（`197-223`、`546`）；
5. 块级超限闸门（`highlight.rs:640-654`）换成 Web 阈值（见施工卡）。

**B. 改造适配（语义照搬、实现重写）**
1. 「视觉裁剪 + 选区取逻辑源」（`968-980`、`activity_preview.rs:17`）→ Web 侧 `truncated` 行同时渲染可见片段与
   `<span aria-hidden data-full>`（或复制事件读 `full`），满足「截断但原文可复制」；
2. 「每 hunk 一次高亮」→ 保留 hunk 粒度，加 `WeakMap<hunkText, TokenSeg[][]>` 缓存（key=path+hunkText hash），避免展开/收起整块重算（对照 G7）；
3. 硬换行（`942`、`952`）→ **不搬**，Web 用 `whitespace-pre-wrap` + 横向滚动（见 C），但**续行不重复符号**要留：多行视觉行只在第一行画 `+/-`，续行留 2 空格缩进；
4. 卡面 3 行（`DETAIL_PREVIEW_LINES=3`、`751-755`）→ 改「首屏 3 行 + 点击展开全量」，展开时恢复上下文行（复用现有 fold 机制）。

**C. 不搬 + 理由**
1. **终端硬换行 / 无横向滚动**：Web 有 `<pre>` + `overflow-auto`，硬换行会让长代码行断裂错位，不如横向滚；
2. **ANSI-16 退让层**（`169-175`）：浏览器恒真彩，`RichDiffColorLevel` 整层是终端债；
3. **ratatui `Line/Span/HyperlinkLine`**：`prefix_bytes/range` 是终端选区模型，Web 用 Range API，照搬让实现翻倍；
4. **`desired_height` 双算**（`322-333`）：终端要预量高度，Web 不需要；
5. **活动分组结构**（`ActivityGroup`/`ActivityDetails`）：gaea 的 `ChangesPanel` 三态折叠（写入/编辑/读取）更贴合产品，不要为对齐而合并。

## 十、施工卡

**落点（只改这 4 个文件，不动数据源）**

| 文件 | 改动 |
|---|---|
| `frontend/src/gaea/lib/diffRender.ts` | 新增 `gutterWidth()`、`toPresented()`（视觉/逻辑双轨），保留 `pairModifications`/`foldContext` 语义 |
| `frontend/src/gaea/components/ChangesDiff.tsx` | 加行号列、统一闸门、截断可复制、下标键 → 稳定内容键 |
| `frontend/src/gaea/lib/diffHighlight.ts` | 加 hunk 级批量 + 缓存；`MAX_HIGHLIGHT_LEN` 与统一闸门对齐 |
| `frontend/src/gaea/components/ChangesPanel.tsx` | 消费者侧加 `memo`，避免父层 setState 触发整块重渲 |

**关键签名**

```ts
// 列宽预扫描：整块统一，不抖动（照搬 diff_render.rs:613-654）
export function gutterWidth(rows: DiffRow[]): number;            // 返回 Math.max(1, String(maxNo).length)
// 视觉/逻辑双轨：visual 进 DOM，full 供复制（照搬 968-980 + activity_preview.rs:17）
export interface Presented { sign: "+" | "-" | " "; no: number | null; visual: string; full: string; truncated: boolean }
export function toPresented(rows: DiffRow[], maxBytes: number): Presented[];
// 统一闸门：超限返回 null = 整块不着色（照搬 highlight.rs:653-654）
export function highlightHunk(path: string, hunkText: string): TokenSeg[][] | null;
// 稳定键：内容指纹取代下标（修 G8）
export function rowKey(p: Presented): string;                    // `${p.no}:${p.sign}:${hash(p.full)}`
```

**开关与默认**
- `gaea.diff.renderer = "v2"`（默认 v2；`"v1"` 回退旧 `PresentRows`，开关读 localStorage，缺省 v2）；
- `gaea.diff.gutter = true`（默认开；关闭即回到现网无行号态）；
- `gaea.diff.highlightMaxBytes = 262144`（256 KiB）、`highlightMaxLines = 5000`、`highlightMaxLineBytes = 4096`
  ——三个值**必须**从同一处 `exceedsHighlightLimits()` 出，禁止再出现 G6 那种三处各写一个常量；
- `gaea.diff.visualClamp = 2000`（视觉截断字符数，仅影响 DOM，不影响复制）。

**可自动断言的验收（5 条，全部可在 vitest/testing-library 里写死）**
1. **规模**：构造 5000 行补丁，`render` 后断言 `container.querySelectorAll('[data-diff-row]').length <= 300`
   且 `performance.now()` 差值 `< 60`（首帧预算；超限时 `highlightHunk` 被 spy 到 **0 次调用**）；
2. **截断但可复制**：构造 3000 字符单行，断言存在 `[data-truncated="true"]`、DOM 文本长度 ≤ 2000，
   且 `copy` 事件读到的文本长度 === 3000（证明逻辑源未丢）；
3. **列宽不抖动**：同一补丁跨 3 个 hunk、行号 9 / 99 / 9999 混排，断言所有行号元素的 `offsetWidth` 全等（差 0px）；
4. **折叠口径唯一**：折叠块文案里的 `N` 与「共有多少行被省略」一致；断言页面上**不存在**第二个「已截断」计数（修 G4）；
5. **展开态稳定**：对同一文件写入两次（第二次行数变化），断言已展开的 fold 仍指向同一内容行（`rowKey` 命中，修 G8）。

**回退**：`renderer=v1` 直接走旧路径，不改 `DiffRow`/`ChangeDiff` 结构与 `planDiff` 输出——数据层零风险，只回退表现层。
灰度顺序建议：GitPanel（`GitPanel.tsx:265`，有 path 有高亮）→ ChangesPanel（`ChangesPanel.tsx:334`）→
CommitGraphView（`CommitGraphView.tsx:298`，顺带补 path）→ ScheduleDiffCard 最后（独立渲染器，收益低）。

## 十一、风险

1. **LCS 是 O(n·m) 全量矩阵**（`diff.ts:27`，注释自承不可滚动数组化）：5000×5000 = 25M 个 number，单次就可能几十 MB / 秒级卡顿
   （codex 用 `diffy` 线性空间绕开）。**折叠/虚拟化只遮住渲染，遮不住构造期 DP**——大规模补丁必须先换算法或加行数上限拒绝。
2. **`useState<Set<number>>` 下标键**（`ChangesDiff.tsx:61-62`）：同文件再写 → `call.diff` 变 → 下标语义漂移，表现为「展开的是别处的行」。
   这是正确性问题，优先级应高于样式。
3. **行号 + marker + 符号三列并存**在窄面板（< 320px）吃掉 8-10 字符，正文不可读。必须容器查询断点降级（窄屏丢行号、保 marker），
   不要沿用 `w-4`/`w-9` 硬编码。
4. **「截断可复制」若实现成把全文塞 `data-full`**，等于没省成本。正确姿势：复制时按 key 从 `ChangeDiff` 原对象取，DOM 只留可见片段——
   否则 G5 修好了、首帧预算又破了。

---

证据自查：`tui/src/render/highlight.rs:640`（`MAX_HIGHLIGHT_BYTES = 512 * 1024`）、`tui/src/diff_render.rs:952`（`available = width.saturating_sub(gutter_width + 2).max(1)`）、`frontend/src/gaea/components/ChangesDiff.tsx:102`（`const longLine = r.text.length > MAX_DIFF_LINE_COLLAPSED;`）
