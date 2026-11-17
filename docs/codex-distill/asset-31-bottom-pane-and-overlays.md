# 资产 31 · 底部输入面板（bottom pane）+ 视图栈 + 审批覆盖层

> 源码根：`codex-rs/tui/src/bottom_pane/`（`mod.rs` 4343 行 / `view_stack.rs` 130 / `chat_composer.rs` 12750 /
> `approval_overlay.rs` 2591 / `list_selection_view.rs` 3515 / `request_user_input/mod.rs` 3970）。
> 下文行号均相对该目录。对照根：gaea `frontend/src/`（`gaea/components/*`、`gaea/hooks/*`、
> `components/chat/*`、`lib/composerDrafts.ts`）。

---

## 一、视图栈：唯一键盘主人

`view_stack.rs:1-2` 把纪律写在文件头：**只有栈顶视图收输入；画背景不改焦点**。

- 渲染（`view_stack.rs:33-43`）：栈顶 `Inline` 直接画；`Centered` 先画下一层当背板再盖居中对话框。
- 压栈（`mod.rs:753-758`）：置 `key_chord_reset_requested`（作废跨视图前缀和弦）→ 入栈 → 排帧。
- 弹栈（`mod.rs:760-781`）：`Accepted` 会**级联弹出**所有 `dismiss_after_child_accept()` 的父视图，
  `Cancelled` 只清父视图标记（`mod.rs:1582` 替换视图时也继承该标记）。
- 替换（`mod.rs:1592-1605`）：按 `view_id` 自顶向下 `rposition` 替换，不动上层；
  `mod.rs:1613-1639` 连续 pop 再压栈 = 把 N 个旧视图折成一个列表页。
- 焦点不变量：`cursor_pos/cursor_style` 只问栈顶（`view_stack.rs:58-69`）；
  对话框几何 `dialog_area`：宽 `width-4` 夹在 `8..=72`，高=自报高度截视口（`view_stack.rs:17-30`）。

## 二、键盘路由是一条显式阶梯（最该照搬）

`mod.rs:858-997` 是唯一入口，顺序即裁决，**不存在两个弹层抢键**：

| 序 | 持有者 | 证据 |
|---|---|---|
| 0 | 视图迁移键直接吞 | `mod.rs:864-866` |
| 1 | Release 广播全栈；被遮视图只"观察" | `mod.rs:867-880` |
| 2 | `warnings_view` 独占 | `mod.rs:892-908` |
| 3 | 展开的 `questions` 独占 | `mod.rs:909-923` |
| 4 | `view_stack` 栈顶（Esc 可先走 `on_ctrl_c`） | `mod.rs:924-968` |
| 5 | inline banner → 中断键 | `mod.rs:969-984` |
| 6 | composer（内部再分弹层/历史/vim） | `mod.rs:985-996` |

配套 `mod.rs:1000-1013` `keymap_contexts()` 用同一顺序回答"下一个键落到哪个 context"，
让提示、宏、HUD 与真实路由同源——gaea 缺的正是这层抽象。`mod.rs:1024-1061` `on_ctrl_c`
与 `mod.rs:1063-1100` `handle_paste` 是同构阶梯：**粘贴永远只进栈顶**。

## 三、高度分配与 footer 优先级

`mod.rs:2418-2571` 用 `FlexRenderable` 按固定顺序 push（顺序即纵向顺序）：
banner（仅非运行）→ status → hook_status → unified_exec_footer → pending_thread_approvals
→ 空行 → pending_input_preview → question_summary → composer。
两条隐藏规则：`mod.rs:2482-2501` 有 preview 才补空行；`mod.rs:2525-2532` 历史覆盖层
**不得**清掉上方 status/preview，于是把 `Overlay` 降级成 `AboveComposer`。
composer 内三段布局（`composer_layout.rs:77-103`）：shortcuts 在上 / popup 在上 / popup 覆盖，
一律 `Min(3 + voice_rows)` 保底 3 行草稿；状态行被 `area.height - footer - 3` 夹住
（`composer_layout.rs:44-49`），**永不吃输入行**。总高公式 `composer_layout.rs:170-209`；
textarea 高度只数换行（`textarea.rs:541-546`）。

## 四、composer 输入状态机

弹层单例：`popup_state.rs:105-131` `ActivePopup::{None,Command,File,Skill,MentionV2}`，
高度 `required_height = popup + footer`。被 Esc 关掉的 token 记 `DismissedToken{query,token,occurrence}`
（`popup_state.rs:12-45`），用**「同文本第几次出现」**而非字节偏移定身份——前插文字不会让已关弹层复活。

视图内路由（`chat_composer.rs:1974-2008`）：vim 搜索 → 历史搜索 → vim 历史 → Ctrl+R →
粘贴 Tab → 按 popup 类型分派 → `sync_popups()`。`popup_active()` 定义见 `chat_composer.rs:2012-2016`。
- 斜杠：Esc 只关弹层**不改草稿**（`slash_input.rs:245-252`）；Tab 补全、部分命令直接派发（`slash_input.rs:296-304`）；
  过滤器取首行首 token（`command_popup.rs:110-129`），上限 `MAX_POPUP_ROWS=8`（`popup_consts.rs:14`）。
- 文件搜索：异步结果只在"光标 token 仍以该 query 开头"时应用（`chat_composer.rs:1895-1918`），只留第一页
  （`file_search_popup.rs:72`；测试 `set_matches_keeps_only_the_first_page_of_results`，`:295`）。
- 排队提交：queue 键在"运行中 或 `queue_submissions` 或 非 shell"时走 `handle_submission(true)`
  （`chat_composer.rs:3531-3539`）；`InputResult::Queued{text,text_elements,action,pending_pastes}`
  （`chat_composer.rs:467-494`）——**排队条目带粘贴占位符**，出队才展开。

## 五、paste burst：终端补丁（阈值可直接当夹具）

`paste_burst.rs:157-170`：`MIN_CHARS=3`、`ENTER_SUPPRESS=120ms`、`CHAR_INTERVAL=8ms`、
`ACTIVE_IDLE=8ms`（Windows `60ms`）。判定链（`223-253`）：活动期→`BufferAppend`；
挂起首字符且间隔 ≤8ms→`BeginBufferFromPending`；`consecutive>=3`→`BeginBuffer{retro_chars:n-1}`；
否则 `RetainFirstChar`。计数 `278-287`。flush 用 `>` 不走 `>=`（`298-312`），调用方须多等 1ms。
Enter 抑制 = 活动期 或 120ms 窗口内（`348-351`）。追捕仅在"抓回片段含空白或 ≥16 字符"时回删前缀（`404-424`）。
另有独立大粘贴阈值 `LARGE_PASTE_CHAR_THRESHOLD=1000`（`chat_composer.rs:457`，占位折叠
`paste_input.rs:145,185`）与开关 `set_disable_paste_burst`（`paste_input.rs:100-107`，关前先 flush）。

## 六、历史搜索 = 快照 + 可撤销

`HistorySearchSession`（`history_search.rs:61-74`）存五份状态：`original_draft`、`preview_draft`、
`original_vim_history`、`original_vim_state`、`query`。`begin_history_search`（`127-156`）顺序是纪律：
flush burst → 清 file 查询 → 清 popup → 清图片选中 → 快照 → 重置游标。`update_history_search_query`
（`290-313`）先恢复 preview 再重搜，query 空则回 Idle 并恢复原草稿；`cancel_history_search`
（`321-333`）draft/vim history/vim state 三还原。状态 `Idle/Searching/Match/NoMatch`（`93-99`）。
footer 优先级（`footer_state.rs:134-166`）：搜索/`:q` → 强制 HistorySearch；否则基础态上依次允许
EscHint、ShortcutOverlay、QuitReminder 覆盖。

## 七、审批覆盖层：选项、焦点、队列

结构（`approval_overlay.rs:185-196`）：`current_request + queue: Vec<ApprovalRequest> + list: ListSelectionView`；
`build_options`（`259-338`）产「标题 + header + items + footer_hint」，四张选项表：

| 类型 | 选项顺序 | 证据 |
|---|---|---|
| Exec | 由 `available_decisions` 定：Accept / Execpolicy 前缀 / Session / 网络策略 / Decline / Cancel | `852-938` |
| ApplyPatch | proceed / don't ask again for these files / No, tell Codex | `1035-1053` |
| Permissions | this turn / +strict auto review(`r`) / this session / deny | `1055-1087` |
| MCP elicitation | Accept / Decline / Cancel，**Esc 硬绑 Cancel** 且从 decline 去重 | `1096-1120` |

含换行的 execpolicy 前缀会被丢弃（`874-876`）——不可读的"别再问"规则干脆不给。
默认焦点：`picker()` 不设 `initial_selected_idx`（`picker_presets.rs:22-31`、`list_selection_view.rs:315`），
`apply_filter` 收尾落 `first_enabled_visible_idx()` 再退 0（`list_selection_view.rs:720-728`）——
**高亮第一项但未生效**，只有 Enter/点击才 `take_last_selected_index → apply_selection`（`601-609`）。
快捷键通道在列表之前（`557-592`）。`cancel_current_request`（`515-551`）清空整队再收摊。
队列是 `Vec+pop()`（`507-513`，LIFO），故 BottomPane 从尾部倒序 drain 还原 FIFO（`mod.rs:817-820`）。
`APPROVAL_PROMPT_TYPING_IDLE_DELAY=1s`（`mod.rs:243`）：**打字时审批不抢屏**，idle 1s 才上（`mod.rs:790-820`）。

## 八、request_user_input / 列表 / pending

- 多问题（`request_user_input/mod.rs:129-162`）：`queue: VecDeque` + 每题 `AnswerState{options_state,
  draft, answer_committed, notes_visible}` + `current_idx` + `Focus{Options,Notes}`；进度
  `Question i/N (k unanswered)`（`326-340`）；notes 复用 `ChatComposer::plain_text()`（`183-227`，
  配置见 `chat_composer.rs:600-610`）；非 blocking 请求 60s 静默 + 60s 倒计时后空提交（`73-74,270-290`）。
- 列表（`list_selection_view.rs:218,289`）：`max_visible_rows` 默认 `MAX_POPUP_ROWS=8`；可见行
  `min(max, 已渲染)`（`654-662`）。tab：`SelectionTab{id,label,header,items}`（`selection_tabs.rs:17-22`），
  单行高（`24-26`），`‹/›` 仅在宽 `≥5/≥7` 出现（`71-72`），左右键切 tab（`1325-1336`）；
  带 tab 且无初始项才主动选首行（`551-553`）。翻页 `ScrollState`：上下 wrap、翻页 clamp
  （`scroll_state.rs:31-50,94-117`）。可搜索列表里普通字符归查询不导航（`1289-1296`）。
- pending（`pending_input_preview.rs:23-38`）：`pending_steers`/`rejected_steers`/`queued_messages`
  三段各限 `PREVIEW_LINE_LIMIT=3` 行（`38,62-72`），顺序 steer→rejected→queued（`98-140`）。
  多线程待审批只列前 3 + `...` + `/subagents`（`pending_thread_approvals.rs:40-70`）。

## 九、冲突裁决总表

| 优先级 | 持有者 | 同时存在时 |
|---|---|---|
| 1 | warnings view | 吸收一切（`mod.rs:892-908`） |
| 2 | 展开 questions | 独占；栈非空时反而让位（`909-923`） |
| 3 | 栈顶视图（审批/选择器/表单） | Esc 可先被 `on_ctrl_c` 截（`924-968`） |
| 4 | banner / 中断键 | `969-984` |
| 5 | composer 弹层（slash/file/@） | 单例；Esc 关层不改稿（`slash_input.rs:245-252`） |
| 6 | 历史搜索 / vim 查询 | 早于弹层（`chat_composer.rs:1975-1992`） |
| 7 | 普通编辑 | 最后 |
| 8 | 粘贴 | 只进栈顶，否则 composer（`mod.rs:1063-1100`） |

**审批弹窗 + 斜杠弹窗并存**：栈非空 ⇒ composer 收不到键（`mod.rs:925`），弹层随 composer 冻结，
且渲染时被降级为 `AboveComposer` 以免遮状态行（`2525-2532`）。

## 十、gaea 现状核对（逐项，带 file:line）

| 能力 | codex | gaea 现状 | 缺口 |
|---|---|---|---|
| 视图栈/单焦点 | `view_stack.rs:33-43`+`mod.rs:924-968` | 无栈；三处各挂不同层级监听（`ApprovalModal.tsx:93` document、`CommandPalette.tsx:159` input、`Composer.tsx:548` textarea） | **硬缺口**：无"谁拥有键盘"单一真源 |
| 审批期间打字 | 被栈顶吃掉 | `ApprovalModal.tsx:78-81` target 是 INPUT/TEXTAREA 时**主动让路** | 审批卡开着仍可输入、回车仍可提交 |
| 默认焦点/可达性 | 首项高亮 + 快捷键双通道（`list_selection_view.rs:720-728`、`approval_overlay.rs:582-591`） | `ApprovalModal.tsx:76` 聚焦 `tabIndex=-1` 卡片容器，按钮无 roving focus | 只靠背数字；Tab 进不去按钮序 |
| hardConfirm 数字键 | 选项与快捷键同源生成 | `:172` abort 显示 `hardConfirm?3:5`，但 `KEY_DECISIONS["3"]="allow_session"`（`:40-46`）且 `:89` 守卫拦下 | **真 bug**：硬确认卡"3=终止"无效，只剩 Esc→deny(`:82-85`) |
| 弹层互斥 | `popup_state.rs:105-131` 单例 | `useComposerMenus.ts:97-101` 三选一有优先级，但不知审批是否打开 | 审批/AskCard 在场时斜杠照弹 |
| 粘贴突发 | `paste_burst.rs:157-170` 有毫秒维度 | `PasteManager.tsx:11-21` 只有 ≥2000 字符 / ≥20 行 | 浏览器 paste 是单事件 ⇒ **不搬**；缺的是 Enter 抑制 |
| 大粘贴占位 | 阈值 1000 + 提交展开（`paste_input.rs:152-215`） | 阈值 2000（`PasteManager.tsx:11`）+ `expandBlocks`（`62-68`） | 语义等价，仅阈值差异 |
| 历史搜索 | `history_search.rs:61-74,321-333` | 只有 ↑/↓（`Composer.tsx:408-410`）+ sessionStorage（`:257-264`） | 无 Ctrl+R 前缀搜索、无取消恢复 |
| 排队提交 | `chat_composer.rs:3531-3539` | 已有 `Composer.tsx:283-290,170-197`（队列卡/插话/撤回） | ✅ 已有；差异是 codex 随队带 `pending_pastes` |
| 草稿持久化 | 队列不跨重启重放 | `lib/composerDrafts.ts:1-3,63-66`（队列恢复为 pending 且不自动派发） | ✅ 一致 |
| 多问题表单 | 每题状态 + 进度 + 自动消解（`request_user_input/mod.rs:129-162`） | `AskCard.tsx:22-23` 两个字典一次性全渲染，`aria-modal="false"`（`:159`） | 无逐题焦点/进度/自动消解 |
| 列表分页/tab | `MAX_POPUP_ROWS=8`、`ScrollState`、tab 条 | `MenuContainer.tsx:16` 固定 `max-h-[280px]` 滚动 | 无分页/横向 tab |
| 键提示栏 | `shortcut_hud.rs:53-160` 响应式两列 | `MenuContainer.tsx:21-33` 单行固定提示 | 可保留，两列不必搬 |
| 技能补全 | `ActivePopup::Skill` 内联 | 独立 `components/SkillModal.tsx:24-48` | 不走输入内联（标注差异即可） |
| footer 优先级 | `footer_state.rs:134-166` 六态瀑布 | `Composer.tsx:449-455` placeholder 四分支 | 无搜索/退出提醒覆盖态 |
| 命令面板重复 | — | `wenxue/components/CommandPalette.tsx:1-16` 与 gaea 版 **revision 相同** | 一份文件两份拷贝，改一处须改两处 |

## 十一、可迁移机制三档

**A. 直接照搬**
1. 渲染顺序即优先级（`mod.rs:2440-2570`）：用 flex 顺序表达，别用 z-index 猜。
2. 对话框几何（`view_stack.rs:17-30`）：`max-w-[72ch]` + 视口高上限 + 居中。
3. 审批四张选项表 + `Esc=cancel` + 按类型去重快捷键（`approval_overlay.rs:852-1120`）。
4. token 的**出现序号**身份判定（`popup_state.rs:12-45`），替换字符串比较。
5. 队列条目带待展开文本（`InputResult::Queued.pending_pastes`）。

**B. 改造成 React 状态机**
1. 视图栈 → `useViewStack{stack,push,pop(r),replaceIf}`；导出 `owner = warnings ?? expandedQuestions ?? last(stack) ?? banner ?? 'composer'`。
2. 键盘阶梯 → 纯函数 `resolveKeyOwner(event)` + 顶层 `onKeyDownCapture`，禁止组件自挂 document。
3. footer → `resolveFooterMode()` 六态瀑布（`footer_state.rs:134-166`）。
4. 历史搜索 → `useHistorySearch{snapshot,previewDraft,query,status,cancel}`。
5. 列表 → `useScrollState(len,visibleRows)`：上/下 wrap、翻页 clamp、`ensure_visible`。
6. 多问题 → `Record<qid,{selected,notes,committed,focus}>`。

**C. 不搬（终端独有）**
1. `paste_burst` 全套时间阈值：浏览器 `paste` 是单个合成事件，无逐字符洪流；只借「粘贴后 120ms 内 Enter 不提交」。
2. vim 模式与键位序列（`textarea/vim.rs:7-15` Normal/Insert/Replace + `VimPending` + `mod.rs:754` 和弦重置）：Web 有原生撤销栈，收益<成本。
3. `cursor_pos/cursor_style`（`view_stack.rs:58-69`）：DOM 管光标。
4. `CellDiffOption::AlwaysUpdate` 全表面重绘（`view_stack.rs:79-85`）：终端缓存残留问题。
5. `pending_thread_approvals` 多线程审批：gaea 单会话下为空集。

## 十二、施工卡

**落点**：新增 `gaea/hooks/useViewStack.ts` + `gaea/components/composer/ViewStack.tsx`（唯一栈+owner）；
`Composer.tsx:399-441` 的 `onKeyDown` 降级为"owner==='composer' 才处理"，菜单分支（`:401-406`）并入 popup 层；
`ApprovalModal.tsx:78-95` 删 document 监听改由栈顶接键、`:76` 改为聚焦首个选项并补 `role="listbox"`
+ `aria-activedescendant`；`useComposerMenus.ts:97-101` 增 `ownerActive` 入参（为真则 `menuMode=null`）；
`AskCard.tsx:22-23` 收为 per-question 状态 + 进度脚注。

```ts
type KeyOwner = 'warnings'|'questions'|'view'|'banner'|'popup'|'composer';
interface ViewStackState { stack: ViewRef[]; owner: KeyOwner;            // owner 派生，不存两份
  push(v: ViewRef): void;                                                // 置 chordReset=true
  pop(r: 'accepted'|'cancelled'|'none'): void }                          // accepted 级联
interface ApprovalState { current: ApprovalRequest|null;
  queue: ApprovalRequest[];        // FIFO：入队 push、出队 shift（别学 Vec::pop）
  delayedUntil: number|null;       // now+1000ms 无输入才上屏
  focusIndex: number }             // 默认 0 = 首个 enabled 项
```

**可自动断言验收**（RTL + fake timers）
1. 「审批时打字不进 composer」：ApprovalModal+Composer 同挂，`keyDown(textarea,'a')` → value 不变、`onSend` 未调。**当前会失败**（`ApprovalModal.tsx:81` 让路）。
2. 「粘贴 800 字符/间隔 3ms 只触发一次 onChange」：`fireEvent.paste` 800 字符 → `onChange` 次数===1、undo 栈只 +1；若开 burst 兼容层，800/3ms 合成 `beforeinput` 流只产生 1 次 `setText`。
3. 「队列」：`running=true` 时 Enter → `queue.length+1`、`text===''`、`onSend` 未调（`Composer.tsx:294` 已满足）。
4. 「历史搜索」：Ctrl+R → 输入 query → Esc → `textarea.value` 恢复打开前草稿。
5. 「审批焦点」：挂载后 `activeElement` = 第 1 个决策按钮；`ArrowDown` 移动 `focusIndex`；Enter 触发该 decision；`hardConfirm` 下按 `3` 必须 abort。
6. 「弹层互斥」：owner==='view' 时输入 `/` → 不渲染 SlashMenu。

**开关与默认**：`features.viewStack=true`（新旧路径并存一版）、`approvalTypingIdleDelayMs=1000`、
`pasteEnterSuppressMs=120`（默认关，仅在"粘贴误回车"回归时开）、`menuBlocksWhenModal=true`。
**回退**：`viewStack=false` 时 `resolveKeyOwner()` 恒返回 `'composer'`，各组件恢复自挂监听；
`ApprovalModal` 的 document 监听保留在 `if(!features.viewStack)` 分支，一版后删。

## 十三、风险

1. **焦点丢失**：栈 pop 后焦点落 `body`，输入法候选框会飘。pop 后隔一帧 `composerRef.focus()`
   （对应 codex `schedule_active_view_frame`，`mod.rs:1134`）。
2. **键盘陷阱**：`CommandPalette.tsx:110-127` 按键全挂 input，焦点跑到列表按钮 Esc 就失效——现成反例；
   改栈顶统一接键 + 容器内焦点循环。
3. **竞态（审批抢屏 vs 打字）**：codex 用 1s idle（`mod.rs:243,790-820`）；React 侧要 `lastInputAt`+定时器，
   **每次输入都重排**，否则"打完一句审批才冒出来"。另 `AskCard.tsx:141-147` 自动聚焦与
   `Composer.tsx:116,214` 的 `requestAnimationFrame` 聚焦互抢，需统一为"最后挂载者赢"。
4. **级联弹出误伤**：`dismiss_after_child_accept` 会让一次确认收掉多层（`mod.rs:760-781`）；
   替换父视图时忘继承（`mod.rs:1582`）就退化成"确认了还留在原地"。

证据自查：`view_stack.rs:33`、`mod.rs:925`、`approval_overlay.rs:507`
