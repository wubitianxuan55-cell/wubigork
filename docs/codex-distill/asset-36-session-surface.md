# 资产 36 · 会话面：分页检索、预览先于打开、名字解析不猜

> 证据根（只读）：`codex-rs/tui/src/resume_picker.rs`（264KB，本篇只引结构与状态机）、
> `resume_picker/{archive,page_loading,layout}.rs`、`named_session_lookup.rs`、
> `session_archive_commands.rs`、`session_queue_commands.rs`、`unarchive_prompt.rs`。
> 对照根（只读）：`frontend/src/gaea/{hooks/useSessionManager.ts,lib/projectGroups.ts,components/Sidebar.tsx}`。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、列表的全部状态都在一个 `PickerState` 里，分页是**显式游标**

`resume_picker.rs:96-108` 先把布局常量钉死（分页与排版同一文件，便于一起调）：

| 常量 | 值 | 用途 |
|---|---|---|
| `PAGE_SIZE` | 25 | 每页行数 |
| `LOAD_NEAR_THRESHOLD` | 5 | 距尾 5 行就预取下一页 |
| `SESSION_META_*` | date 12 / gap 2 / indent 2 / cwd 30–72 | 元信息列宽（cwd 可伸缩但要夹住） |
| `FOOTER_COMPACT_BREAKPOINT` | 120 | footer 换紧凑版的宽度阈值 |

分页结构：`PageCursor`（`:349-352`）+ `PickerPage{rows, history_modes, next_cursor, num_scanned_files, reached_scan_cap}`（`:353-360`）。
`ingest_page`（`:1666-1696`）落页时干三件事：**回填分页状态**（`complete_page(next_cursor, num_scanned_files, reached_scan_cap)`）→
**按 `seen_key` 去重**（`self.seen_rows.insert(key)` 为真才 push）→ `apply_filter()`。
`reached_scan_cap` 是一等公民：扫到上限时 UI 必须能说"没扫完"，而不是假装到底了。

## 二、搜索必须"边搜边续页"，且带 token 防串台

`set_query`（`:1768-1800`）是这篇最值得抄的一段：

```
query 未变 -> return
query = q; selected = 0; apply_filter()          // 先本地过滤（已加载的页）
if q 为空 -> SearchState::Idle; return
if 有本地命中 -> SearchState::Idle; return        // 命中就够了
if 扫到上限 或 没有 next_cursor -> SearchState::Idle; return   // 没得再搜
token = allocate_search_token(); SearchState::Active{token};
load_more_if_needed(LoadTrigger::Search{token})   // 继续拉页，结果为同一 token 才认
```

配套 `SearchState`（`:943-947`）：`Idle | Active{token}`；`LoadTrigger`（`:973-976`）：`Scroll | Search{token}`。
即"为用户正在进行的搜索"单独开一条加载理由，**滚动触发的加载与搜索触发的加载不混账**。
`row_matches_query`（`Row::matches_query`，`:1051-1082`）搜五个字段：`preview`、`thread_name`、`thread_id`、
`git_branch`、`cwd`，全部 `to_lowercase().contains(q)`。

两条配套细节：
- `clear_query_preserving_selection`（`:1799+`）：清搜索前先取当前行的 `seen_key`，清完把它找回并选中——
  Esc 退出搜索不丢当前位置；
- `pending_page_down_target`（`:1697-1717` `complete_pending_page_down`）：加载中按 PageDown 时**记住目标索引**，
  页到了再落位（目标超出末页且有下一页就继续拉），而不是"按键丢失"或"停在旧处"。

## 三、先看再开：转录预览的三态

`TranscriptPreviewState = Loading | Loaded(Vec<TranscriptPreviewLine>) | Failed`（`:949-954`）；
`SessionTranscriptState = Loading | Loaded(TranscriptCells) | Failed`（`:955-961`）——
**预览（摘要若干行）与完整转录是两个状态机**，各有自己的失败态。
`TranscriptPreviewLine{speaker: User|Assistant, text}`（`:962-971`）只有"谁说的 + 文本"，不做富渲染。

加载与帧的配合（`:1183-1230`）：
`is_transcript_loading` / `note_transcript_loading_frame_drawn`（"这一帧已经画过 loading 了"）/
`open_pending_transcript_if_ready`（数据到了且**画过 loading 帧**才切过去）/ `begin_transcript_loading(thread_id)`。
目的很直白：**避免"空帧闪烁"**——先让 loading 上屏，再让内容顶替。

## 四、按名字找会话：**同名必报歧义，跨页不确定就拒绝猜**

`named_session_lookup.rs:23-38`：

```rust
enum AmbiguousSessionName {
  Multiple{name,first_id,second_id}  // "Multiple sessions match '{name}' (including …); use a session UUID to disambiguate."
  Paginated(String)                  // "Cannot verify a unique session label across server pages; matching session UUID: {…}. Use it only if this is the session you want."
}
```

- 显示名 = `thread.name` 去空白非空，否则 `thread.preview.trim()`（`:40-46`）；
- 遍历 active/archived 两个集合 × 多个 source_kind，逐页 `limit: 100`；
- **排序键固定 `CreatedAt`**，注释写死理由：「活动不得把未读线程顶到游标前面」（`:79-83`）——
  否则一页之间顺序会漂，分页去重与"唯一性判定"同时失效；
- 分页游标不合法（不含 `|` 或解析失败）→ `unverified_pagination = true`，最终如果命中唯一但分页不可验证，
  仍然抛 `Paginated` 而不是"就这么用"。

> 这套"名字 → id"的规矩是**宁缺勿错**的极端版：宁可让用户贴 UUID，也不替他选一个可能错的会话。

## 五、归档是一个有状态的动作用，且要看键位有没有被占

`resume_picker/archive.rs:22-31` `ArchiveState = Idle | Pending{thread_id} | Restoring{thread_id}`——
归档/取消归档各自是一个"进行中"状态，界面据此禁用重复触发。
`archive_shortcut_available()`（`:34-50`）在给动作绑定 `Ctrl+A` 之前先查：
**键位表里 `Ctrl+A` 是否已有 list 动作或被和弦前缀占用**（`keymap.list.action_for()` + `chords.bindings` 前缀检查）。
即"动作可用性"包含"这个键还没被别处用掉"——不会出现两处抢同一个键。

配套还有一个独立入口 `unarchive_prompt.rs`（7.5KB）：归档集合为空时给"要不要看归档"的提示，
而不是让用户对着空列表猜东西去哪了。

## 六、gaea 现状核对（只读实测）

| 位置 | 现状 | 差在哪 |
|---|---|---|
| `gaea/hooks/useSessionManager.ts:5` | `PAGE_SIZE = 10` | 与 codex 的 25 只是数值差；真正差别在下一行 |
| 同文件 `:39-45` `refreshSessions` | `listSessions()` **一次拉全量** → 缓存进 `allSessionsRef` → 只展示前 10 | 会话多了每次刷新都全量拉；后端没有游标概念，前端"分页"是切片 |
| 同文件 `:47-52` `loadMore` | 纯客户端 `slice` | 无"预取阈值"、无 `reached_scan_cap`、无"扫没扫完"的可信表达 |
| `gaea/lib/projectGroups.ts:5-20` `filterProjectGroups` | 搜 3 字段：`title/preview/path` | codex 搜 5 字段（多 `thread_id`、`git_branch`）；且 gaea 无"边搜边续页"（因为全量已在内存，暂无必要） |
| `components/Sidebar.tsx:286-288` | 行内 `s.preview` 摘要 | 只有一行预览；**打开前看不到转录**（codex 有独立预览态） |
| `Sidebar.tsx:187-199` | 每项目一个"已归档"折叠桶 + 计数 | 比 codex 的全局 archived 集合更细；保留 |
| `Sidebar.tsx:241-267` | 行内改名（Enter/Esc/blur） | 见 F37 §八：提交路径三条，需真机验证顺序 |
| `SessionMeta`（`lib/types`） | 有 `path/title/preview/current` | 无 `thread_id`/`git_branch`/`updated_at` 排序键 → 名字解析与分支检索都做不了 |
| 全局 | 没有"按名字打开会话"的入口 | codex 的 `AmbiguousSessionName` 机制无对应物（gaea 里同题会话靠 title 区分，靠人眼） |

## 七、三档判定

**档一（照搬语义）**

1. **`seen_key` 去重**：`SessionMeta` 增稳定键（`path` 或后端 id），刷新/合并/追加一律按它去重——
   现在靠"重新拉全量覆盖"，一旦接后端增量就会重复行。
2. **搜索结果与列表同源**：搜索走"过滤已加载 + 需要时继续拉"的两段，而不是"全量在内存里过滤"（为将来分页铺路）。
3. **`reached_scan_cap` 式诚实**：列表到底了才说"没有更多"；没扫完说"还有未加载"。
4. **加载中按键不丢**：`pending_page_down_target` 的等价物（当前列表是客户端的，成本很低）。

**档二（改造搬）**

5. **预览先于打开**：选中会话时拉 3–5 条摘要（复用一个轻绑定即可），三态 `loading/loaded/failed`；
   失败要能说失败（而不是空面板）。
6. **名字解析不猜**：新增"按名字打开"入口时，同名 ≥2 必须列候选并给稳定 id，不许取第一条。
7. **动作状态化**：归档/取消归档用 `Pending/Restoring` 三态，且动作可用性检查键位占用（F32 落地后才有意义）。

**档三（不搬）**

8. `AltScreenGuard`（`:872-887`）的全屏进出恢复、`FOOTER_COMPACT_BREAKPOINT=120` 的终端断点；
9. `PickerPage.history_modes` / `num_scanned_files` 这类后端扫描统计（gaea 的存储层不同）。

## 八、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F36-A 稳定键去重** | `gaea/lib/types`（`SessionMeta`）+ `useSessionManager.ts:32-85` | 增 `key`（path 归一）；`refresh/loadMore/删除` 三处按 key 去重合并 | 单测：同一 path 两次出现只留一行；删除后乐观更新不复活 |
| **F36-B 搜索两段** | 新 `gaea/lib/sessionSearch.ts` `searchSessions(sessions, q)`（5 字段） | 纯函数 + 命中计数；`filterProjectGroups` 改为调用它 | 单测：命中 `path`/`title` 各一条；`git_branch` 缺字段不炸 |
| **F36-C 预览态** | `Sidebar.tsx` 行选中 → 新 `SessionPreview` 组件 | 三态 + 失败文案；不阻塞主列表 | 单测：loading → loaded；失败显示"预览不可用"且可重试 |
| **F36-D 名字唯一性** | 新 `gaea/lib/sessionLookup.ts` | `resolveByName(name) -> {ok, id} \| {ambiguous, ids[]} \| {unknown}` | 单测：同名 2 条 ⇒ ambiguous 且返回两个 id；0 条 ⇒ unknown |
| **F36-E 动作状态** | `Sidebar.tsx` 归档按钮 | `idle/pending/restoring` 三态禁用 | 单测：pending 期间再点不重复调用 |

**开关**：全部为纯函数 + 组件内部状态，无新绑定前不触碰 store；`FEATURE_SESSION_PREVIEW` 单闸控预览。
**回退**：F36-A/B 只改 hook 内部与工具函数，`Sidebar` props 形状不变。

## 九、风险

1. **`refreshSessions` 是多方调用的热点**（新建/恢复/删除/重命名后都调）：加去重与回填时不要改变"刷新后清空搜索"的既有语义
   （`startNewSession` 里 `setSidebarQuery("")`，`useSessionManager.ts:49-56`）。
2. **预览会引入新请求**：`frontend/src/perf-guards.test.ts` 对新增订阅敏感；预览必须在"选中且列表稳定 200ms 后"才发。
3. **`SessionMeta` 扩字段要过 Wails 序列化**：gaea 有过"Go struct 缺 json 标签 → 前端读空"的旧账（progress v4.269），
   加字段必须给 camelCase tag 并在 mock 侧同步，否则预览/分支字段永远 undefined。
4. **两壳**：`wenxue/components/Sidebar.tsx`（50KB）是另一份更大的实现，本批不碰；单源化留作独立拍板项。

## 十、证据自查

- 分页常量与页结构：`resume_picker.rs:96-108,349-360,1666-1696`。
- 搜索续页与 token：`:943-976,1768-1800,1051-1082,1799+`。
- 预览三态与帧配合：`:949-971,1183-1230`。
- 名字解析与歧义：`named_session_lookup.rs:23-46,52-100`。
- 归档状态与键位占用检查：`resume_picker/archive.rs:22-50`。
- gaea 侧：`useSessionManager.ts:5,32-85`、`lib/projectGroups.ts:5-20`、`components/Sidebar.tsx:187-199,241-267,286-288`。
