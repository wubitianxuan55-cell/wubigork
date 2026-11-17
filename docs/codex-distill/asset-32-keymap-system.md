# 资产 32 · 键位与快捷键系统施工级：和弦解析、上下文分区、同源帮助浮层

> 源码（codex 0.165.0，`codex-rs/tui/src`）：`keymap.rs`、`keymap/{bindings.rs,chords.rs,leader.rs}`、`key_hint.rs`、
> `shortcut_help.rs`、`bottom_pane/{shortcut_overlay.rs,shortcut_hud.rs}`、`tui/{input_boundary.rs,windows_key_sequence.rs}`、
> `app/input.rs`、`keymap_setup.rs`、`chatwidget/permission_shortcuts.rs`、`text_formatting.rs`。
> 对照（gaea 只读）：`frontend/src/wenxue/app/useAppKeyboard.ts`、`frontend/src/gaea/app/useAppKeyboard.ts`、
> `layouts/MainLayout.tsx`、`schedule/commandRegistry.ts`、`pages/schedule/ScheduleCommandPalette.tsx`、
> `pages/SchedulePage.tsx`、`gaea/lib/{paletteRank.ts,spaceBindings.ts}`、`gaea/components/CommandPalette.tsx`。
> 目标：React 前端落地「表驱动 + 两键和弦 + 上下文位集 + 帮助浮层与表同源」。

---

## 一、机制事实

### 1.1 表的组织：上下文分组 → 动作 → 键集合

- `RuntimeKeymap`（`keymap.rs:81-95`）是唯一派发形状：按**焦点/模式**切成 12 个键位组
  （`app/chat/composer/editor/vim_normal/vim_operator/vim_search/vim_text_object/pager/list/agents/approval`），
  外加独立 `chords` 层；上下文枚举 13 个值（多出的 `Global` 作兜底，`keymap/bindings.rs:15-30`）。
- 动作身份 = `(context, action)` 二元组（`bindings.rs:99-103`），配置路径由它生成
  `tui.keymap.<context>.<action>`（`bindings.rs:125-127`）。一个动作可绑多个键，**顺序只选主提示、不作派发优先级**
  （`key_hint.rs:199-213`）。
- 解析优先级：上下文覆盖 → `global` 兜底 → 内置默认（`keymap.rs:68-79`）；**显式空列表是权威解绑**，不再兜底
  （`keymap.rs:2554-2575`）。和弦独立成层 `RuntimeChordKeymap{leader, bindings}`（`chords.rs:236-240`）：每条 =
  `(action, KeyChord{prefix, completion}, spec 原文)`，`spec` 保留用户原文供回显与反向写配置。

### 1.2 和弦解析：两键序列、超时、leader

- 只支持**两键**。`leader <key>` 是符号写法，解析时展开成「每个 leader 前缀 × 每个 completion」笛卡尔积，
  设 1024 条上限防二次方爆炸（`chords.rs:335-343`）。
- 状态机 `advance`（`chords.rs:548-610`）出口：`Pending`/`Completed`/`Cancelled`/`PassThrough`；`Ignored` 留给内部 token
  与非按下事件（`chords.rs:444-450`、`554-565`）。
- **超时 1 秒**（`KEY_CHORD_TIMEOUT`，`chords.rs:31`），过期即丢弃、**不回放首键**（`chords.rs:535-546` 只清 pending）——
  与 gaea 想要的「500ms 回落单键」相反，见 §四。
- leader 前缀**无 deadline**：`PendingChord.started_at: Option<Instant>`，`None` = 常驻菜单（`chords.rs:452-458`、`601-605`）；
  常驻判定 =「是 `leader` 写法，或与 Global 动作重叠」（`chords.rs:592-600`）。
- 取消：`Esc` 或再按同一 prefix（仅常驻态）→ `Cancelled`（`chords.rs:568-572`）；上下文集合变化亦取消（`chords.rs:535-540`）。
- 完成不直接调函数：动作映射成**内部 F 键 token**追加进该动作绑定表，让既有单键派发表继续当唯一出口（`chords.rs:1-8`、
  `613-650`）；token 遇 `advance` 被忽略（`554-556`、`662-667`），防回流。
- 前缀歧义：同 prefix 多 completion **允许**并存，选择靠 `binding_for_completion` 的 `min_by_key` 上下文偏好
  （pager 关闭优先、warnings 页优先 list 而非 global），不是「先注册先得」（`chords.rs:365-387`）。
- 隐式默认 `g g`（vim `jump_top`）仅在「未配、且同上下文无动作占用 `g`」时注入（`chords.rs:299-333`）。

### 1.3 上下文/模式：位集 + 动作级例外

- 上下文是位集：`KeymapContextSet(u32)`（`chords.rs:46-57`）；焦点切换时算活跃集，见 `App::active_keymap_contexts`
  （`app/input.rs:224-273`）：overlay→Pager（搜索编辑→Editor）；transcript 搜索→Editor；activity→`List|Global`；
  browsing→`Pager|Global`；否则 ChatWidget 上下文 ∪ Global ∪ Chat（+Voice）。
- 判定统一走 `contains_action`（`chords.rs:144-170`），编码**动作级例外**：`open_presets` 在 List 聚焦失效、
  `toggle_voice` 需未被屏蔽、warnings 页里 `list.accept` 与多数 global 失效、只剩 `open_warnings`/`copy`。
- 两层重叠：`KeymapContext.overlaps`（`bindings.rs:57-95`：chat/global/voice/composer 共享主输入路径，vim 与 editor 互斥，
  `List↔Approval`、`List↔Agents` 重叠）+ `KeymapActionId::overlaps` 动作例外（`bindings.rs:107-123`）。
  冲突检测全靠这两层，不靠字符串比较。

### 1.4 冲突检测与覆盖优先级

五道闸，全部**加载期报错**（不是运行期静默让位）：

| 闸 | 位置 | 拒绝什么 |
|---|---|---|
| 同组去重 | `keymap.rs:1999-2020`（`validate_unique` 2422） | 同上下文两动作同键；chat 组占可打印键 |
| 保留键 | `keymap.rs:2478-2509`，表 `2511-2552` | 占 `ctrl-c/d/v`、`shift-tab`、`esc`、`alt-←→`、`/ ! @ $` |
| 跨组遮蔽 | `keymap.rs:2442`（`allowed_overlaps` 白名单） | app 组动作遮蔽 approval/list 同键 |
| 和弦三重 | `chords.rs:707-761` | ①leader 当 completion ②prefix 遮蔽活跃单键 ③同 chord 两动作 |
| 形状/保留 | `chords.rs:763-853` | 纯文本/AltGr 当 prefix（仅 vim 与 command center，`bindings.rs:50-55`）；`esc` 入 stroke；Unix `ctrl-z`；按上下文查保留表（`825-838`） |

leader 专属校验：默认 `ctrl-x`，若已被别处占用则让位不注入（`leader.rs:16-22`）；禁多键 leader（`:23-30`）；禁纯文本/AltGr
首键（`:42-54`）；Unix 禁 `ctrl-z`（`:55-61`）；禁撞保留表（`:62-70`）。运行期**覆盖优先级**只有一处：
`binding_for_completion` 排序（`chords.rs:378-386`）；其余同键冲突在加载期已拒。

### 1.5 平台差异

- 修饰键显示名按 OS 切表：macOS `⌃ ⇧ ⌥ ⌘`、Linux `^`、其余 `ctrl/shift/alt/super`（`key_hint.rs:26-41`）；和弦显示
  「前缀 空格 后缀」（`key_hint.rs:73-80`）。事件归一 `normalize_key_parts`（`key_hint.rs:166-187`）：`Tab+SHIFT→BackTab`；
  裸 C0（0x01–0x1A、0x1C–0x1F）还原成 `ctrl-a…z`/`ctrl-4…7`（`:189-197`）；大写字母补 SHIFT 并降小写——故 `shift-a`
  与终端直报 `A` 等价（`:43-48`）。
- 别名互通：`ctrl-]`↔`ctrl-5`（`key_hint.rs:143-148`）；`ctrl-/`↔`ctrl-7`（`chords.rs:669-676`、`696-704`）。
- AltGr：**仅 Windows** 把 `ctrl+alt` 视作 AltGr（`key_hint.rs:301-311`），故禁 `ctrl-alt-<char>` 当快捷键（`chords.rs:767-776`）。
- Windows 终端序列：Windows Terminal 用 `sendInput` 逐键发 Shift+Enter，Crossterm 收到 7 个裸键；
  `WindowsKeySequence` 只在凑齐 `ESC [ 1 3 ; 2 u` 时合成，**50ms 内凑不齐原样重放**（`windows_key_sequence.rs:1-7`、
  `26-35`、`66-117`、`131-149`）。纯文本与粘贴事件永不扫描。

### 1.6 输入边界：不吞键

- `discard_pending_terminal_input`（`tui/input_boundary.rs:25-64`）：接受安全敏感决策前排空已解码/未读输入；
  **未完成控制序列继续隔离**，1s 判不出边界返回 `TimedOut` 而非硬吞；Windows 用 `GetNumberOfConsoleInputEvents` 探队列（`:99-117`）。
- 状态机不缓冲普通输入：只存一个 `pending`（`chords.rs:460-464` 注释「without buffering ordinary input」），
  `PassThrough` 立即把原事件交回上层（`app/input.rs:163-168`）；pending 时 `schedule_frame_in(KEY_CHORD_TIMEOUT)`
  让 UI 在过期时刻重绘（`:173-179`）。
- 消费顺序（`app/input.rs:67-193`）：视图切换/配置编辑器先看物理键 → overlay/transcript 自有键 → chord → 普通 keymap。
- 反向保护：`is_dispatch_token_event` 让内部 token 永不被当用户键（`chords.rs:662-667`）；
  先查活跃绑定、再问 chord、全不命中才把 `Alt+字符` 还原成 `Esc+字符`（`app/input.rs:31-64`）——
  方向是「宁可漏兼容路径，不可吞正文输入」。

### 1.7 帮助浮层 / HUD：同一份表渲染

- `shortcut_overlay::lines`（`shortcut_overlay.rs:26-146`）不硬编码键位，取 `props.key_hints`（运行期 `ShortcutHint`）：
  头注释直说「Runtime hints remain authoritative for remapped, chorded, and disabled bindings」（`:3-5`）。
  只有不可配固定键（`/ @ !`、`ctrl-c`、`pgup/pgdn`）才写字面量。
- 关闭提示独立一行（`?`/`esc`，窄屏降级只留 `esc close`，`:148-161`）；页脚固定 `/keymap customize`（`:163-171`）；
  高度不足截断并追加 `… resize to see all`（`:173-187`）。
- 排版 `shortcut_help.rs`：按每组最长键标签算列宽对齐（`:38-52`），宽度不足三列→两列→一列（`:55-82`），列式渲染 `:84-100`。
- HUD 常驻提示 `ShortcutHud`（`shortcut_hud.rs:25-51`）是「`key · label`」被动网格：>5 条才开两列（`:53-62`），
  最小列宽 32、列距 2（`:20-23`），单格超宽省略号裁（`:126-148`），裁完末行换 `… resize to see all`（`:150-159`）。
  由面板构造（例 `ShortcutHud::new("Presets", …)`，`chatwidget/quick_dials.rs:77`），内容同源。标签裁字另有
  `text_formatting.rs:70`（`truncate_text`）。
- 唯一「写」侧是 `/keymap`（`keymap_setup.rs`）：动作清单来自同一 `RuntimeKeymap`（`active_binding_specs`，`:589-641`），
  捕获的 `KeyEvent` 反序列化成配置串（`:678-748`），冲突弹专页给原文错误（`:405-448`）；清自定义 = 槽位 `None` 恢复兜底
  （`:657-668`，「`None` vs 空列表」区别在 `:652-656`）。
- pending 发现性：`pending_hint_items`（`chords.rs:472-529`）用**同一 matcher** 推「首键 then 后缀→动作」，最多 6 项、
  末尾补 `esc cancel`，超宽先缩文案再逐项丢弃 —— 提示与派发共用一次解析，不会「提示能按、按了没反应」。

## 二、gaea 现状核对

| codex 机制 | gaea 现状 | 缺口 |
|---|---|---|
| 表驱动 | 两份手写 `if` 链：`wenxue/app/useAppKeyboard.ts:28-48` 与 `gaea/app/useAppKeyboard.ts:28-50`（后者多 `openKnowledge`） | **两份真相**，无键位表 |
| 上下文分区 | 只有「焦点在输入框就跳过」（`useAppKeyboard.ts:32-33`；`SchedulePage.tsx:741-742`） | 无 list/pager/modal 分区；模态开时全局键照旧生效 |
| 多键序列 | 无 | 全缺 |
| 前缀歧义等待 | 无 | 全缺 |
| 冲突检测 | 无：`commandForShortcut`（`commandRegistry.ts:202-206`）线性 `find` 首个命中即返回 | 同键两命令静默取先注册者 |
| 帮助浮层 | 仅命令面板列里展示 `shortcuts`（`ScheduleCommandPalette.tsx:87` 起），无总览 | 文案与分发可漂移（`formatShortcut` 只显示，`:176-190`） |
| 常驻 HUD | 无（rail 只有 `title` tooltip，`MainLayout.tsx:295`） | 全缺 |
| `/keymap` 自定义 | 无 | 用户不可改键 |
| 平台修饰键 | `ctrlKey \|\| metaKey` 合并（`useAppKeyboard.ts:31`、`commandRegistry.ts:169`、`MainLayout.tsx:454`） | mac 上 `Ctrl+K`/`Cmd+K` 不可分；无 `super` |
| 键名归一 | `normalizedKey` 只做 space/escape/小写（`commandRegistry.ts:161-165`） | 无 `event.code`、无 BackTab、无别名互通 |
| 主提示 | `SCHEDULE_COMMANDS` 数组序即提示序（`commandRegistry.ts:89`） | 无主提示概念 |
| 逐键发现性 | `spaceBindings.ts:750-771` 的 warn-once 去重集 | 模式可借给「重复和弦告警」 |

补充（直接影响落点判断）：
- 任务书要求的 `layouts/CommandRail.tsx` **不存在**：`CommandRail` 已内联在 `MainLayout.tsx:257-347`（`CommandRail.test.tsx:4`
  亦从 `./MainLayout` 导入）。改 MainLayout，不要新建同名文件。
- `gaea/lib/spaceBindings.ts` 与键位无关：它是 Wails `AppBindings` 的**空间归属**表（头注释 `:1-19`），命名撞车，勿混用。
- 已有两处全局监听互抢：`MainLayout.tsx:452-476`（`Ctrl+1~9`、`Ctrl+K`，`page !== 'gaea'` 才开搜索）与
  `useAppKeyboard.ts:39`（gaea 内 `Ctrl+K` 开自有面板）——正是 codex 用「上下文位集 + 单入口路由」消灭的分叉。
- 无交叉校验：`Ctrl+1` 在 `MainLayout.tsx:456-463` 按 `getActiveMenuBoards()` 序号派发，进度板块 `ctrl+p/z/f`
  走 `commandRegistry.ts:44-52` 自己的表，两表不同源。
- `paletteRank.ts:23-39`（按当前视图稳定重排命令）不是键位系统，但可作「视图 = 上下文来源」的先例。

## 三、浏览器里的硬约束

1. **保留键拦不住**：`Ctrl/Cmd+W`（关标签）、`Ctrl/Cmd+T`、`Ctrl/Cmd+N`、`Cmd+Q` 由宿主处理，`preventDefault()` 无效；
   `Ctrl+L`（地址栏）、`Ctrl+P`（打印）、`Ctrl+S`（保存）多半也先被浏览器/WebView2 吃掉。做法：这些键**不注册**，
   改用带修饰的非常用键；Wails 场景另在 Go 菜单层用加速键兜。
2. **`event.key` vs `event.code`**：`key` 随布局与 Shift 变（`Shift+1`→`"!"`，AZERTY 的 `Digit1` 出 `&`），`code` 是物理位。
   **匹配用 `code`（可配 fallback），显示用 `key`**；输入法激活时 `key` 可能是 `"Process"`。
3. **IME 组合态**：组合中 `keydown` 的 `isComposing === true`（旧 Safari 只有 `keyCode === 229`），字符属输入法，
   **任何 `key.length === 1` 判定必须先 return**。gaea 只有 Composer 判断（`utils/chatComposer.ts:7-9`、
   `components/Composer.tsx:400`），全局监听完全没有（`useAppKeyboard.ts:28-48`）：中文输入时按 `n/k/b` 会误触全局命令。
4. **裸字母前缀在中国输入法下几乎不可用**：`g g`、`j k` 会被输入法先吃，前缀应限定「带修饰键的键 + 功能键」或仅在焦点
   非输入框且 `!isComposing` 时参与匹配；并用 `compositionstart/compositionend` 维护 ~200ms 冷却窗，避免组合收尾那一下
   被当成序列首键。

## 四、可迁移机制三档

**直接照搬**
- 动作身份 `(context, action)` + 配置路径生成（`bindings.rs:99-127`）；备选键顺序 = 主提示、不涉派发（`key_hint.rs:199-213`）。
- 位集上下文 + `contains_action` 例外表（`chords.rs:46-197`）+ 两层 `overlaps`（`bindings.rs:57-123`）+ 保留键表
  （`keymap.rs:2478-2552`）。
- 帮助浮层 / HUD / pending 提示三者从运行期 hint 派生（`shortcut_overlay.rs:26-146`、`shortcut_hud.rs:44-51`、`chords.rs:472-529`）；
  `/keymap` 的「活绑定 → 配置串」双向序列化（`keymap_setup.rs:589-748`）。

**改造适配**
- 两键和弦 + 超时：照搬状态机，超时改 **500ms**，并加「首键有单键绑定则回落派发一次」（codex 是纯丢弃，`chords.rs:535-546`）。
- leader：默认不设，保留抽象但要求前缀含修饰键（§三.4）。
- 内部 F 键 token（`chords.rs:613-650`）→ 换 `Symbol`/常量事件，浏览器没有 F13+。
- `discard_pending_terminal_input` → 「切页/开模态时清 chord pending + 清一次性按键队列」；`WindowsKeySequence` 不搬，
  但保留「失败前缀原样重放」（`windows_key_sequence.rs:66-117`）：序列不匹配时首键必须按单键逻辑重走。
- `/keymap`：先做只读总览（同表渲染），写入侧二期。

**不搬 + 理由**
- 终端 C0 归一、`ctrl-5`↔`ctrl-]`、`ctrl-/`↔`ctrl-7`、AltGr（`key_hint.rs:143-197`、`301-311`）：浏览器给不出这些形态，
  照搬只增加不可测分支。
- CSI-u 序列解码（`windows_key_sequence.rs` 全文件）：仅「Windows Terminal → Crossterm 逐键」链路成立。
- vim 系列上下文（`bindings.rs:22-25`）：gaea 编辑器非模态；将来引入时 `allows_plain_chord_prefix`（`bindings.rs:50-55`）可原样用。
- 1024 条 leader 上限 + 笛卡尔积（`chords.rs:335-343`）：前端改「注册总数 ≤ 256」的简单闸。

## 五、施工卡

**落点**：① `lib/keymap/types.ts`（类型 + 纯函数校验）② `lib/keymap/table.ts`（唯一动作表，派生派发/浮层/HUD）
③ `lib/keymap/matcher.ts`（`advance/expire/cancel`）④ `lib/keymap/useKeymap.ts`（一个 `document` keydown + 一个上下文来源，
替换两份手写链，`useAppKeyboard.ts` 留导出名做薄壳）⑤ `gaea/components/{ShortcutOverlay,ShortcutHud}.tsx`（禁字面量键名）
⑥ 迁移序：`Ctrl+1~9`/`Ctrl+K`（`MainLayout.tsx:452-476`）→ gaea 全局链 → `schedule`（`commandRegistry.ts:202-206`）。

```ts
export type KeymapContext = "global" | "chat" | "composer" | "app" | "list" | "pager" | "schedule" | "modal";
export interface KeySpec { code: string; ctrl?: boolean; alt?: boolean; shift?: boolean; meta?: boolean }
export interface ActionEntry {
  context: KeymapContext;
  action: string;                      // 稳定 id，如 "palette.open"
  keys: KeySpec[];                     // 备选；[0] 为主提示
  chords?: Array<[KeySpec, KeySpec]>;  // 两键序列
  label: string; group: string;        // 帮助浮层用
  enabled?: () => boolean;             // 运行期禁用：灰显但保留提示
  run: () => void | Promise<void>;
}
export type MatchResult =
  | { kind: "pass" } | { kind: "pending"; prefix: KeySpec; actions: ActionEntry[] }
  | { kind: "complete"; action: ActionEntry } | { kind: "cancel" } | { kind: "ignored" };
```

**可自动断言验收**（vitest + 可控时钟）
1. 同一和弦注册两次 → **后注册者覆盖**，开发期 `console.warn` 恰好一次（借 `spaceBindings.ts:750-771` 的 warn-once 写法）。
2. 序列首键后 **500ms 无后续键** → 回落该首键的单键动作（若有），否则 `pass`；断言 `pending` 已清空。
3. 首键 + 非 completion 次键 → 首键动作**原样重放**（等价 codex `flush`，`windows_key_sequence.rs:59-64`），且不吞次键。
4. 同上下文两动作同 `KeySpec` → `buildTable()` 抛错，错误串含两个 `context.action`（对齐 `chords.rs:744-757`）。
5. 和弦 prefix 与活跃单键相同 → 抛错（对齐 `chords.rs:727-742`）。
6. `Escape` 作任一 stroke → 抛错；pending 时 `Escape` 取消并消费该键（对齐 `chords.rs:568-572`、`800-805`）。
7. 上下文切换 → `expire()` 返回 true，且不再派发旧上下文动作（对齐 `chords.rs:535-540`）。
8. 浮层行集合 === `table.ts` 里 `enabled()` 为 true 的动作集合（删一个动作，浮层少一行）。
9. `isComposing === true` 的 keydown → `pass` 且不产生 `pending`。
10. 保留键（`Ctrl+W` 等）命中 → 表构建期拒绝注册，断言从不调用 `preventDefault`。

**开关与默认**：`settings.keymap.enabled`（默认 true，关掉回落两份手写链，零行为变化）；`chordTimeoutMs`（默认 500）；
`legacyCtrlMetaMerge`（默认 true，见风险 2）；主提示取 `keys[0]`，显示复用 `formatShortcut`（`commandRegistry.ts:176-190`）。

**回退**：薄壳（`useAppKeyboard.ts` 内部换实现，一文件一 flag）；浮层/HUD 是新增组件，删即回退；
灰度 `schedule` → `gaea` → 壳层 `MainLayout`，每步独立可退。

## 六、风险

1. **`code` 与布局/输入法错配**：`code` 稳定但用户记字符，德语布局 `Ctrl+Z` 落在 `KeyY`。缓解：默认 `code`，每动作可配 `matchBy: "code" | "key"`，文本类（`/ @ !`）用 `key`。
2. **`ctrlKey || metaKey` 合并的迁移陷阱**：`commandRegistry.ts:169` 与 `MainLayout.tsx:454` 都合并了；一旦拆开，
   mac 用户原 `Ctrl+K` 失效。缓解：默认开 `legacyCtrlMetaMerge` 别名，显式配置后才拆。
3. **全局监听数量**：现至少 4 处（`useAppKeyboard.ts:45`、`MainLayout.tsx:474`、`SchedulePage.tsx:748`、`:140` 捕获阶段）。
   收敛时漏迁会出现「某板块快捷键整体失灵」；用验收 8 的「表 = 浮层」不变式兜住，并保留 `defaultPrevented` 早退（`:740`）。
4. **HUD 空间成本**：浏览器常驻 HUD 挤正文与移动端视口。缓解：仅 hover/`?` 展开，宽度 < 768px 只显示 3 条高频，
   沿用「… resize to see all」截断（`shortcut_hud.rs:150-159`）而非硬换行。

证据自查：`keymap/chords.rs:800-805`（`esc` 入 stroke 在加载期报错）、`bottom_pane/shortcut_hud.rs:150-159`（末行替换为截断提示）、`frontend/src/schedule/commandRegistry.ts:169`（`ctrlKey || metaKey` 合并为一档）。
