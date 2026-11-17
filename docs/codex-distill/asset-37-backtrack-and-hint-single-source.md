# 资产 37 · 回退语义与提示同源：Esc 谁先收、编辑回退凭什么算"可回退"、提示怎么不切开

> 证据根（只读）：`codex-rs/tui/src/`（`app_backtrack.rs`、`footer_hint.rs`、`shortcut_help.rs`、
> `key_hint.rs`、`bottom_pane/popup_consts.rs`、`bottom_pane/footer_state.rs`）。
> 对照根（只读）：`frontend/src/gaea/app/useAppKeyboard.ts`、`frontend/src/components/chat/ChatRow.tsx`、
> `frontend/src/gaea/components/ApprovalModal.tsx`（F31 已核出的两处缺陷）、`frontend/src/gaea/components/Sidebar.tsx`。
>
> 本篇只出机制与落点；不改 gaea 源码。

---

## 一、Esc 在 codex 是**阶梯**，且回退前面还有一道守卫

阶梯本体在 `bottom_pane/mod.rs:858-997`（F31 已逐条列出：迁移键 → Release 广播 → warnings → questions → 视图栈顶 → banner → composer）。
本篇补的是它在 App 层的**入口守卫**与**两段式回退**。

`app_backtrack.rs:110-122` `handle_backtrack_esc_key`：

```rust
if !self.chat_widget.composer_is_empty() { return; }   // ① 草稿非空 → 这一下 Esc 不归回退
if !self.backtrack.primed { remember_browsing_origin(); prime_backtrack(); }  // ② 第一次：进"预备态"
else if self.overlay.is_none() { self.open_backtrack_preview(); }             // ③ 第二次：开预览
```

三条设计意图可以从代码直接读出来：
- **①草稿优先**：用户正在打字时，Esc 不该把人送回上一条消息——先把"清空/取消"留给输入面；
- **②两段式**：单次 Esc 只"预备"，必须再按一次才真的进回退浏览（防误触把会话拖回历史）；
- **③幂等**：已经在预览里时不重复开。

## 二、回退的判据是 **turn 身份**，不是"第几条消息"

`app_backtrack.rs:64-108` 的状态与选择：

| 字段 | 语义 | 注释里的理由 |
|---|---|---|
| `primed: bool` | Esc 已预备 | — |
| `base_id: Option<ThreadId>` | 被浏览的基线线程 | 「线程变了，回退选择即刻失效，必须忽略」 |
| `nth_user_message: usize` | 高亮的是「自上次会话启动以来的用户消息视图」里的第几条 | **明确不是 `transcript_cells` 的下标**；`usize::MAX` = 无选择 |
| `overlay_preview_active: bool` | 预览在开 | — |
| `origin` | 浏览起点（关预览时回到哪儿） | — |

`BacktrackSelection{thread_id, nth_user_message, prompt}`（`app_backtrack.rs:83-91`）——**选择里带线程 id**，
所以 `apply_backtrack_selection`（`:124-146`）第一步就是"侧聊态直接报错退出"，第二步 `thread_id` 不匹配直接 `return`，
第三步才 `nth_user_position(cells, nth)` 定位到 **cell** 发出 `RevertSessionForPromptEdit{thread_id, selected_cell, prompt}`。

`nth_user_position` / `user_count`（`:653-731`）不数 `transcript_cells.len()`，而是数"用户消息 cell"，
并且**忽略 context-compacted 标记与隐藏的 review prompt**（测试名直说：
`agent_group_count_ignores_context_compacted_marker` / `backtrack_revert_before_turn_id_skips_hidden_review_prompts` /
`…_skips_hidden_nested_review_prompts`）。序号口径错了，回退就会把用户送错地方——所以它单独做函数、单独测。

## 三、"可回退"的四条硬判据（都在测试名里）

`backtrack_revert_before_turn_id`（`:509+`）是真正的 revert 入口，它的拒绝条件被测试逐条钉死：

1. **拒绝回合中途的 steer**：`backtrack_revert_before_turn_id_rejects_mid_turn_steers`
   ——一轮里追加的用户插话不是"独立回合起点"，不能作为回退目标；
2. **部分历史不让 steer 变可编辑**：`backtrack_revert_partial_history_does_not_make_a_steer_editable`
   ——历史被截断/只有半截时，仍然不许把 steer 当回合起点（宁可不给回退，也不给错的回退）；
3. **拒绝在飞与缺失**：`…_rejects_in_progress_and_missing_prompts`
   ——回合还在跑、或 prompt 已不存在，直接拒绝（不是"尽力而为"）；
4. **隐藏提示不计入**：review 类隐藏 prompt 不占序号（见 §二）。

失败路径也写死：`restore_backtrack_prompt_after_revert_error`（`:148-156`）
——**把 prompt 放回 composer** 再报错「Failed to edit the selected prompt: …」。
即"回退失败不等于用户输入丢失"。侧聊里则给专门文案
`SIDE_EDIT_PREVIOUS_UNAVAILABLE_MESSAGE = "Editing previous prompts is unavailable in side conversations."`（`:59-60`）。

> 这一节是整篇最有迁移价值的：**回退 = 一次带事务语义的历史截断**，
> 判据必须落在"回合边界 + 完整性 + 在飞状态"三件上，而不是"列表里第 N 条"。

## 四、提示的三条纪律：不切开、按宽度换档、度量与渲染同源

`footer_hint.rs:1-2` 文件头就是纪律：「把 footer 提示装进行里，**不把快捷键与它的标签拆开**」。

- `shortcut(keys, label)`（`:7-11`）：`key_hint::key_label_spans(keys)` + `" {label}"`，
  注释：「**同一个样式内容**同时用于度量与渲染」——先算宽度再画，不会出现"量的是 A、画的是 B"。
- `first_fitting_line(candidates, width)`（`:14-22`）：从**若干个完整变体**里挑第一个装得下的，
  「窄面永不显示半个快捷键」。空则给空行，不截断。
- `wrap_hint_rows(hints, width, separator_width, hint_width)`（`:24-62`）：贪心装行，
  每个 hint 宽度 `min(hint_width, width)`（单条比整行还宽时不撑爆），行内累加分隔宽；
  **超宽只换行、不切断**。

`shortcut_help.rs`（帮助浮层）是同一条纪律的排版版：
- 每组内 `key_width = max(条目键宽)`，按键右补空格对齐（`Group::lines`，`:38-52`）；
- 三列 → 两列 → 一列 的**显式塌缩**（`group_lines`，`:55-82`）：`sum(widths)+2*GAP ≤ width` 就三列；
  否则"第一列 + 右侧两列堆叠"两列；再不行就整列纵向，组间空行。`COLUMN_GAP = 4`（`:10`）。

键位标签本身只有一个出口：`key_hint::ShortcutHint::display_label()`。
`popup_consts.rs:33-46` 的 `picker_hint_line_for_keymap(&ListKeymap)` 直接把 footer 提示**从键位表派生**，
注释写明「只用配置的 list 动作」——改键位不会留下过时提示。

`bottom_pane/footer_state.rs:134-166` 再规定**优先级**：搜索态 / `:q` → 强制 HistorySearch；
否则在基础态上依次允许 `EscHint`、`ShortcutOverlay`、`QuitReminder` 覆盖。
即"提示也有一条阶梯"，与 §一 的按键阶梯同构。

## 五、gaea 现状核对（只读实测）

| 位置 | 现状 | 差在哪 |
|---|---|---|
| `gaea/app/useAppKeyboard.ts:34` | `if (ke.key === "Escape" && !inInput && !state.running)` 才进门 | **运行中就完全没有 Esc 语义**（Stop 只能点按钮）；输入框内 Esc 也没人接（除编辑态） |
| 同文件 `:35-37` | 阶梯只有两层：`previewFile` → `closeTopmost()` | 无 warnings/questions/banner 层；`closeTopmost` 是唯一裁决点，弹层互斥靠它（F31 已记 `ApprovalModal` 的键盘陷阱缺陷） |
| `components/chat/ChatRow.tsx:119` | 编辑态 Esc → `setEditing(false)` | 只有"退出编辑"，没有"回到上一条消息"（codex 的 backtrack 全缺） |
| `components/chat/ChatRow.tsx:83-93` | 编辑提交走 `onEditSubmit(msg.key, text)` | 是否走"回合级截断"取决于 `pages/chat/branches` 的实现；**没有"回合中途 steer 不可编辑"这类判据**，也没有失败回填 composer 的显式路径 |
| `gaea/components/Sidebar.tsx:247-250` | 改名：Enter 提交 + `onBlur` 也提交 | 两条提交路径并存；Esc 只 `setRenameTarget(null)`。**取消/提交的先后顺序需真机实测**（unmount 是否触发 blur 取决于浏览器与 React 版本），至少是脆弱点 |
| 全局 | 没有 hint bar 组件（搜索 `*Hint*/*Kbd*/*Shortcut*` 零命中） | 提示散落在 `Tooltip`/`title`/内联文案里；`CommandRail` 的键位提示内联在 `layouts/MainLayout.tsx:257-347`（F32 已记） |
| `ApprovalModal.tsx:78-81` | 模态期间 `INPUT/TEXTAREA/contentEditable` 直接 return | F31 缺陷①：审批开着时键盘仍在编辑器里生效 |

**结论**：gaea 缺的不是"某个快捷键没绑"，而是**两层缺失**——
① 没有显式 Esc 阶梯（谁先收），② 没有"回合级回退"这个动作（只有"编辑某条消息"）。
提示侧则缺"不切开 + 换档 + 同源"三条纪律（当前是零散文案）。

## 六、三档判定

**档一（照搬语义）**

1. **Esc 阶梯显式化**：把 `useAppKeyboard.ts:35-37` 的隐式顺序改写成一张显式表
   （warnings → questions → 弹层栈顶 → banner → 输入面 → 空草稿回退），并让"提示文案"从同一张表派生。
2. **回退前置守卫**：草稿非空时 Esc 不进回退（`composer_is_empty()` 的等价物），改为清空/取消输入。
3. **两段式回退**：第一次 Esc 只"预备"并给可见提示（"再按一次编辑上一条"），第二次才进预览。
4. **提示不切开**：任何"键 + 标签"渲染走同一个 helper（`firstFittingLine`），窄容器换完整变体而不是 `text-overflow` 切半。

**档二（改造搬）**

5. **回合级回退**：新增 `canBacktrack(msgKey)` 纯函数，判据四条（回合边界 / 历史完整 / 非在飞 / 非隐藏项）；
   回退动作 = 截断到该回合之前 + 把 prompt 放回 composer（失败时**必须**放回 + 报错）。
6. **提示换档**：帮助浮层按宽度 3→2→1 列塌缩；键列按最长键宽对齐（`shortcut_help.rs` 的算法是 30 行纯函数，可直接移植）。
7. **提示优先级**：footer 提示也走阶梯（搜索态 > 强制态 > 基础态 > 常驻提醒），避免"两个提示抢同一行"。

**档三（不搬）**

8. `Overlay::new_transcript` 的 alt-screen、`copy_on_select`、`mouse_scroll_speed`（终端专属）；
9. `key_hint` 的 span 渲染与 `Span::width()` 度量（Web 用 `ch`/`pill` 组件 + `ResizeObserver`）。

## 七、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F37-A Esc 阶梯表** | 新 `gaea/lib/escLadder.ts`（纯函数：`(ctxState) => owner`）＋ `useAppKeyboard.ts:34-38` 接 | 返回 `"warnings"\|"questions"\|"modal"\|"banner"\|"composer"\|"backtrack"\|null`；运行中也要能给出 `"stop"` | 单测：运行中 + 弹层开 ⇒ `modal`；输入框聚焦 ⇒ `composer`；空草稿无弹层 ⇒ `backtrack` |
| **F37-B 回退判据** | 新 `gaea/lib/backtrack.ts` `canBacktrack()` + `pages/chat/branches` 接 | 四条判据（回合边界/完整/非在飞/非隐藏） | 单测：steer 不可回退；半截历史不可回退；在飞回合不可回退；正常回合可回退 |
| **F37-C 回退失败回填** | 回退调用点 | 失败 ⇒ `setText(prompt)` + 错误条 | 单测：revert 抛错时草稿等于原 prompt，且错误可见 |
| **F37-D 提示不切开** | 新 `components/ui/HintRow.tsx`（`firstFittingLine` + `wrapHintRows` 移植） | 提示条目为 `{keys, label, variantList}` | 单测：宽 120 ⇒ 完整变体；宽 48 ⇒ 短变体；永不出现半截键名 |
| **F37-E 帮助浮层塌缩** | 复用 `CommandPalette`/`shortcut` 面 | 3→2→1 列纯函数（`layoutShortcutGroups`） | 单测：三组宽度和 + 8 ≤ 容器 ⇒ 3 列；否则 2 列；再否则 1 列 |

**开关**：F37-A/B 是纯函数 + 接线，建议 `FEATURE_ESC_LADDER` 单闸；F37-D/E 为新增组件，旧文案保留一版。
**回退**：A/B 只改"裁决点"，不改既有命令实现；关闸时 `useAppKeyboard` 行为等价于现状三行。

## 八、风险

1. **Esc 是全局监听**：`useAppKeyboard` 是 `document.addEventListener("keydown")`（`:49`），
   新增阶梯会改变所有页面的 Esc 行为；`frontend/src/perf-guards.test.ts` 对"新增全局监听"敏感，本卡不得新增监听，只改既有回调内部分支。
2. **`closeTopmost` 是现有唯一裁决点**（`:36`）：F37-A 若绕过它会与弹层栈脱钩，
   正确做法是把阶梯**放在 `closeTopmost` 之前**做类型判断，仍然只调它一次。
3. **回退与分支树耦合**：`pages/chat/branches`（`btOf/dbIdOf`，`ChatRow.tsx:25,82`）已承载版本切换语义，
   回退必须与"截断"用同一套 dbId 口径，否则切版本后回退指向错消息。
4. **改名提交路径**：`Sidebar.tsx:247-250` 的 Enter/blur 双路径在 F37 之外，但同一类"取消≠取消"问题；
   改动前先真机验证 unmount 是否触发 blur，别按猜测动手。

## 九、证据自查

- Esc 守卫与两段式：`app_backtrack.rs:110-122`；选择与状态：`:64-108`。
- revert 判据与失败回填：`app_backtrack.rs:124-156,509+`；序号口径：`:653-731`；
  拒绝条件测试名：`:820,842,863,912,960,1030,1093`。
- 提示纪律：`footer_hint.rs:1-62`；帮助排版：`shortcut_help.rs:10,38-93`；
  提示优先级：`bottom_pane/footer_state.rs:134-166`；同源提示：`bottom_pane/popup_consts.rs:33-46`。
- gaea 侧：`gaea/app/useAppKeyboard.ts:34-38,49`、`components/chat/ChatRow.tsx:25,83-93,119`、
  `gaea/components/Sidebar.tsx:247-250`、`gaea/components/ApprovalModal.tsx:78-81`。
