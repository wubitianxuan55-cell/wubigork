# 资产 35 · 输入弹层：候选模型、匹配语义、与草稿的契约

> 证据根（只读）：`codex-rs/tui/src/bottom_pane/`（`mentions_v2/*`、`command_popup.rs`、
> `file_search_popup.rs`、`skill_popup.rs`、`popup_consts.rs`、`chat_composer/popup_state.rs`、
> `chat_composer/slash_input.rs`）。
> 对照根（只读）：`frontend/src/gaea/{hooks/useComposerMenus.ts,lib/composer.ts,components/{SlashMenu,ArgMenu,MenuContainer}.tsx,components/Composer.tsx}`。
>
> 本篇只出机制与落点；不改 gaea 源码。

---

## 一、codex 只有**一个** @ 弹层，三种检索模式

`mentions_v2/mod.rs:1-10` 点明它是「默认的统一 mention 弹层」，`mentions_v2` 开关只作回滚路径。
`mentions_v2/popup.rs:18-25` 的持有物是一份 `Popup{query, file_search, candidates, rows, search_mode, state}`——
**同一次按键序列里，文件、技能、插件、任务共用一套查询、一套选区、一套视口**。

模式是三态环（`mentions_v2/search_mode.rs:4-42`）：

| 模式 | 收哪些类型 | 标签 | 循环 |
|---|---|---|---|
| `Results` | 全部 | `All Results` | ← → 依次 |
| `FilesystemOnly` | File / Directory | `Filesystem Only` | 同上 |
| `Tools` | Plugin / Skill | `Plugins` | 同上 |

标签文本就是 UI 上的切换说明（`search_mode.rs:38-41`），**过滤语义与标签同源**，不存在"标签叫 Plugins 却还混文件"。
footer 的四组提示也是从同一张表渲染的（`mentions_v2/footer.rs:23-31`：`Enter/Tab insert · Esc close · Up/Down select · Left/Right filter`），
宽度不够时走 `truncate_line_with_ellipsis_if_overflow`。

## 二、候选的身份：五个类型 + 一个可插入值

`mentions_v2/candidate.rs:8-14` `Selection = File(PathBuf) | Tool{insert_text, path}`——**"显示的东西"与"插入的东西"分开**；
`candidate.rs:16-41` `MentionType = Plugin | Skill | Task | File | Directory`，每类有固定三字母标签列（`Plugin/Skill/Task/File/Dir`）与固定配色；
`candidate.rs:56-64` `SearchResult{display_name, description, mention_type, selection, match_indices, score}`——
`match_indices` 只服务高亮，`score` 只服务排序。

> 对照 gaea：`AtEntry{path,name,isDir,size}`（`useComposerMenus.ts:72-95`），只有"文件/目录"二维，
> 插件/技能/任务没有进 @ 面；`SlashMenu` 的 `kind`（`SlashMenu.tsx:19-25` project/mcp/skill）是斜杠侧的标签，@ 侧没有对应物。

## 三、排序：先按类型，再按分数；同类 Task 之间保持原序

`mentions_v2/filter.rs:62-79` `sort_rows`：

```
type_order: Plugin 0 < Skill 1 < Task 2 < File|Directory 3
a,b 同为 Task  -> Ordering::Equal（不重排，保持进入顺序）
否则 type_order 比 -> compare_within_rank -> display_name 兜底
```

`compare_within_rank`（`filter.rs:81-100`）再分两种口径：
- 文件之间：**纯 `score` 降序**（文件搜索本身已带质量分）；
- 其余：`match_indices.is_none()` 升序（命中名字的排前面）→ `score` 升序（越小越靠前，fuzzy 分数是距离）→ 名字。

匹配本身（`filter.rs:48-60` `best_tool_match`）：先 `fuzzy_match(display_name)`；不中则在 `search_terms` 里逐个 fuzzy，**取最小分**（最像的那个别名代表候选）。

> 这套排序的关键性质：**类型优先是硬约束，分数只在同类里说话**。
> 所以用户永远不会看到"一个高分文件压倒所有技能"。gaea 的 @ 合并（`useComposerMenus.ts:72-95`）是
> `recent → atHits → entries` 三段顺序 push + `out.length < 12` 截断，**"最近使用"会吃掉全部名额**，
> 工作区搜索命中常常进不了列表。

## 四、异步结果的三元守卫：`pending` / `display` / `waiting`

`mentions_v2/popup.rs:131-171` `FileSearch`：

| 字段 | 语义 | 证据 |
|---|---|---|
| `pending_query` | 用户**最新**打的串 | `popup.rs:132` |
| `display_query` | 当前 `matches` **对应的**串 | `popup.rs:133` |
| `waiting` | 还在等这一串的结果 | `popup.rs:134` |

- `set_query`（`popup.rs:139-149`）：串变了就 `pending_query = query; waiting = true`（**不清旧 matches**，界面继续显示上一批）；
- `set_matches`（`popup.rs:151-159`）：`query != pending_query` 直接 `return`（过期结果丢弃），否则 `display_query = query`，`matches = matches.take(MAX_POPUP_ROWS)`；
- 空查询（`popup.rs:140-144`）：整个清空，进入"空提示"态（`file_search_popup.rs:61-71` `set_empty_prompt`：`display/pending/matches` 全清 + `state.reset()`）；
- 空态文案分三种（`popup.rs:165-171`）：`waiting` → `loading...`；否则 `no matches`；`should_show_matches` 为假时**不渲染文件段**。

`file_search_popup.rs:22-30` 是同一套字段的另一个实现（旧路径），注释写得比代码更清楚：
"`display_query` = 当前 match 对应的查询；`pending_query` = 用户最新键入，**搜索在飞时两者可以不同**"。

> gaea 的对应物是 `useComposerMenus.ts:63-70`：`live` 布尔守卫 + 直接 `setAtHits`。
> 它防住了"组件卸载后 setState"，**没防"同一组件内先后两次查询乱序返回"**——
> 慢盘 + 快速输入时，旧查询的后到结果会覆盖新查询。也没有 loading 态，慢盘表现为"没搜到"。

## 五、Esc 的身份是「第几次出现的那个 token」

`chat_composer/popup_state.rs:11-54`：

```rust
struct DismissedToken { query: String, token: String, occurrence: usize }
matches(t, range, query) =
      self.query == query
   && text[range] == self.token
   && occurrences_before(text, self.token, range.start) == self.occurrence
```

注释三条纪律值得抄（`popup_state.rs:16-40`）：
1. `token` **带 sigil**（`@xxx` / `/xxx` 整体），`query` **不带**——两者分开记，因为补全只改 token 不改 query；
2. `occurrence` 是「同一 token 文本在草稿中第几次出现」，因此 **`@a @a` 里 Esc 掉第一个，第二个照样弹**；
3. 身份判定不用字节偏移——"偏移会因 offset-only 编辑而漂移，token 文本 + 序号才稳"。

`slash_input.rs:245-252` 的 Esc 动作只有两行：记下当前 token 身份 → `active = None`；**draft 一个字节都不改**。
复位点唯一：token 变了（`chat_composer.rs:3971-3975`，`dismissed_command_token != 当前 token` 则清空）。

> gaea 是**一个布尔** `dismissed`（`useComposerMenus.ts:18`），复位条件是 `[slashQuery, atRaw]` 任一变化（`:103`）。
> 三个后果：① 同一草稿里两个 `@` 无法分别压掉；② 斜杠菜单与 @ 菜单共用同一面旗（Esc 掉 @ 后，下一个 `/` 菜单也会被压一次）；
> ③ 参数菜单（slasharg）没有独立身份，Esc 一次等于关掉整条链。

## 六、Tab 的完整契约（这部分 gaea 目前是缺的）

`slash_input.rs:282-330`，Tab 一次要过六道：

1. **先 resync**：拿最新 `text + cursor` 重算 filter 并 `popup.on_composer_text_change(filter_text)`
   （注释："确保补全前过滤/选区反映最新草稿"）——防"上帧的过滤结果"。
2. **立即派发型命令**（`slash_input.rs:292-304` + `532-534` `selected_command_dispatches_immediately_on_tab`）：
   这类命令 Tab 直接执行并清空草稿，不插入文本。
3. **保尾部补全**（`complete_selected_slash_command_preserving_existing_draft_tail_as_inline_args`，`slash_input.rs:426+`）：
   `/re` + 尾部已有 `view` 时，补成 `/review view` 而不是丢掉 `view`。
4. **反对"把在写前缀当完成"**（`slash_input.rs:142-157` `is_editing_command_name`）：
   注释原句"草稿尾部可能让在写的前缀看起来完整（`/re` + `view`）——光标还在命令名里时保持可编辑"。
5. **有可见项但无选中项** → Tab 被吞（`slash_input.rs:328-330`）：不提交、不补全。
6. 都不成立且 `is_task_running` → 走排队提交（`slash_input.rs:331-333`）。

> 第 5 条是纯正确性：菜单里全是"当前不可用"的命令时，Tab 不该把用户没写完的输入发出去。
> gaea 的 `pickCommand`（`useComposerMenus.ts:105`）是 `"/"+name+" "` + `setTextCaretEnd`——
> **光标强制到末尾**、尾部参数直接丢弃，也没有"立即派发"这一类命令的表达。

## 七、斜杠命令：精确 → 前缀两段，别名隐藏，不可用项保留但降位

`command_popup.rs:110-140` `on_composer_text_change`：
- 只看**首行**；`strip_prefix('/')`；取 `trim_start()` 后的**第一个空白 token**；
- 过滤器变了才 `state.reset()`（选区归零），然后 `clamp_selection` + `ensure_visible(len, min(8, len))`；
- 注释：「`/clear something` 仍显示 `/clear` 的帮助」——**打了参数菜单不关**。

`filtered()`（`command_popup.rs:152-215`）：
- 空过滤器：全量命令 − `ALIAS_COMMANDS`（`Quit` 是 `exit` 的别名、`Btw` 是 `side` 的别名，`command_popup.rs:16-20` 注释"每个动作只出现一次"）− `is_unavailable`；
- 非空：**两段桶**，`exact`（display 或 name 全等）先出，`prefix` 后出，各带 `indices_for(offset)` 高亮区间；同段内**保持原展示序**；
- `side_conversation_active` 时把不可用项排到最后（`filtered()` 尾段）——**保留可见性但降位**，不是直接删。

可用性来自 `CommandPopupFlags`（`command_popup.rs:44-60`，12 个能力位）→ `commands_for_input`。
即：**列表渲染与按键派发读同一份能力位**。`command_popup.rs:86-97` 还在构造期过滤掉 `debug*` 与 `Apps`。

统一常量与同源提示：`popup_consts.rs:14` `MAX_POPUP_ROWS = 8`（注释"所有 popup 一致，手感统一"）；
`popup_consts.rs:33-46` `picker_hint_line_for_keymap(&ListKeymap)` 把 footer 提示**从键位表派生**——
"提示跟着键位表走，改键位不会留下过时提示"。

> gaea：`lib/composer.ts:2-4` 的 `slashQueryOf` 只要出现**任意空白**就返回 `null`，
> 于是输入 `/clear ` 之后菜单直接消失，参数菜单才接上（`useComposerMenus.ts:28`）。
> codex 是"命令菜单 + 参数补全同时在场"。匹配是 `includes`（`useComposerMenus.ts:24`）：
> 输入 `vi` 会命中 `review`，且**没有精确优先**，也没有 unavailable 概念。

## 八、gaea 现状核对（只读实测）

| 位置 | 现状 | 差在哪 |
|---|---|---|
| `gaea/lib/composer.ts:2-4` | 含空白即无斜杠查询 | `/cmd arg` 时命令菜单消失（codex 只在"首 token 之后"仍保留帮助） |
| `useComposerMenus.ts:24` | `name.includes(query)` + `slice(0,8)` | 无 exact→prefix 两段；无 unavailable 降位；无高亮区间 |
| `useComposerMenus.ts:18,103` | 单一 `dismissed` 布尔 | 无 token 三元组身份（见 §五三条后果） |
| `useComposerMenus.ts:28-38` | `SlashArgs` 每字符请求一次，仅 `live` 守卫 | 无 pending/display 分区；乱序返回会覆盖；无 loading 文案 |
| `useComposerMenus.ts:63-70` | `FileSearch` 同款 `live` 守卫 | 同上 |
| `useComposerMenus.ts:72-95` | `recent → hits → entries` 三段拼接，`<12` 截断 | 最近文件可吃满名额；无类型分区/模式切换 |
| `useComposerMenus.ts:105-119` | `pickCommand`/`pickArg` 都 `setTextCaretEnd`，尾部被替换 | 补全丢尾部；光标强制到末尾 |
| `SlashMenu.tsx:19-25,36-40` | kind 标签（project/mcp/skill）贴在行尾 | 与 codex 的"类型优先排序"配合缺失（标签有，序无） |
| `MenuContainer`（`SlashMenu.tsx:5` 引入） | 滚动容器 + `useMenuScroll(activeIndex)` | 视口能力齐，缺"行数上限单一常量"（8 写死在 `slice(0,8)`） |

## 九、三档判定

**档一（照搬语义，收益/成本比最高）**

1. **匹配两段化**：`exact` 先、`prefix` 后、段内保持注册序；`includes` 只作兜底（或彻底去掉）。
2. **Esc 身份三元组**：`dismissed` 布尔 → `{query, token, occurrence}`；三个菜单各自一份。
3. **空查询给提示不给列表**（`set_empty_prompt` 的等价物）：现在 `@` 直接灌 12 条，用户还以为搜到了。
4. **Tab 六道**：补全前 resync；补全保尾部；`has_visible && !selectable` 吞 Tab（**这条是正确性**）。
5. **参数在场时命令菜单不消失**（`slashQueryOf` 改取首行首 token）。

**档二（改造搬）**

6. **异步三元守卫**：把 `live` 布尔升级成 `pendingQuery/displayQuery/waiting`，慢盘给 `loading`；
   `FileSearch` 与 `SlashArgs` 两处同改（一个纯 hook：`useAsyncQuery<T>`）。
7. **类型优先排序**：`AtEntry` 增 `kind`（file/dir/recent/hit/skill/task），排序改为"类型 → 命中质量 → 名字"；
   顺带解决"最近文件吃满 12 条"。
8. **模式切换**：@ 面加三档（全部 / 仅文件 / 技能与任务），键位沿用 `←/→`；斜杠面暂不合并（合并成本过高）。

**档三（不搬）**

9. `MAX_POPUP_ROWS=8`、`COMMAND_COLUMN_WIDTH=AutoAllRows`、`key_hint` 的 span 渲染——终端排版专属，Web 用 CSS 行高与 `max-h`。
10. `complete_token_occurrences_before` 的 element-aware 遍历（`popup_state.rs:57-…`）——gaea 没有原子 element，纯文本 `match_indices` 即可。

## 十、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F35-A 匹配两段化** | `useComposerMenus.ts:24` | 抽纯函数 `rankSlashMatches(commands, query)`（exact→prefix→includes），返回 `{item, indices}` | 单测：`iew`（子串非前缀）⇒ 无结果；`vi` ⇒ `view` 命中且高亮前两位；精确输入 `review` ⇒ 恒第 1 位 |
| **F35-B Esc 身份** | `useComposerMenus.ts:18,103` + `Composer.tsx` 键位 | `dismissed: {query,token,occurrence}\|null`，三菜单各一份 | 单测：`@a @a` 里 Esc 第一个后，第二个条目仍在列表；在别处插字不复活；改字符复活 |
| **F35-C Tab 契约** | `Composer.tsx` 的 keydown → 抽 `useComposerCompletion` | resync → 立即派发名单 → 保尾部补全 → 空选区吞键 | 单测：`/re` + 光标在 `re` 内 ⇒ 不补全；尾部 `view` 保留；全 unavailable ⇒ 不提交 |
| **F35-D 异步三元** | 新 `gaea/lib/useAsyncQuery.ts`；`useComposerMenus.ts:28-70` 两处接 | `{pending,display,waiting}` + 过期丢弃 + `loading` 文案 | 单测：先发 `a` 后发 `ab`，`a` 的结果后到不得覆盖；`waiting` 时列表保留旧值 |
| **F35-E 类型排序** | `useComposerMenus.ts:72-95` + `MenuContainer` 行高 | `AtEntry.kind`；排序纯函数；12 条上限提为常量 | 单测：最近 10 条 + 1 命中 ⇒ 命中前排；目录与文件分段 |

**开关**：F35-A/C/D 为纯函数 + 单测先行；UI 层 `FEATURE_COMPOSER_MENU_V2` 单闸，关闸走原 `includes`/布尔路径。
**回退**：五卡都可只改 hook 内部，`Composer.tsx` 的 props 形状不变（`slashMatches/argRes/atItems/menuMode/menuCount` 保持）。

## 十一、风险

1. **`setTextCaretEnd` 是全局约定**：`Composer.tsx:118` 起的菜单 hook 全部通过它写草稿；改"保尾部补全"要动这个约定，
   牵连粘贴/队列/纠正模式（`Composer.tsx:156` `correctionRef`）。建议先在 hook 内用 `{text, caret}` 形态另开通道，验证后再迁移。
2. **`@` 面接后端**：`FileSearch` 已在 `GaeaListDirBatch`/`FileSearch` 两个绑定上（`useComposerMenus.ts:63-70`），
   加 `kind` 排序不需要新绑定；但技能/任务要进 @ 面就得先有数据源（`Commands()` 只覆盖斜杠侧）。
3. **性能守卫**：菜单每次按键都会重算候选，`frontend/src/perf-guards.test.ts` 对新增订阅敏感；
   `rankSlashMatches` 必须 memo 到 `(commands, query)` 二元组。
4. **两壳**：`gaea/` 与 `wenxue/` 各有 Composer/SlashMenu 一套，先确认 wenxue 是否复用 `useComposerMenus`；不复用则本批不碰 wenxue。

## 十二、证据自查

- 统一弹层与模式：`mentions_v2/mod.rs:1-10`、`popup.rs:18-25`、`search_mode.rs:4-42`、`footer.rs:23-31`。
- 候选与排序：`candidate.rs:8-41,56-64`、`filter.rs:48-100`。
- 异步三元：`popup.rs:131-171`、`file_search_popup.rs:20-72`。
- Esc 身份：`chat_composer/popup_state.rs:11-54`、`slash_input.rs:245-252`、`chat_composer.rs:3971-3975`。
- Tab 六道：`slash_input.rs:282-333,426+,532-534`、`is_editing_command_name` `142-157`。
- 斜杠匹配/可用性：`command_popup.rs:16-20,110-140,152-215`、`popup_consts.rs:14,33-46`。
- gaea 侧：`lib/composer.ts:2-4`、`useComposerMenus.ts:18,24,28-38,63-95,105-119`、`SlashMenu.tsx:19-40`、`Composer.tsx:118`。
