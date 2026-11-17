# 资产 38 · 多智能体面：四组状态、分组键、列表刷新不丢选区

> 证据根（只读）：`codex-rs/tui/src/app/agents_overview*.rs`（主文件 52.9KB / view 30.5KB）、
> `app/agent_center/*`（6 文件 35KB）、`multi_agents.rs`（35.6KB）、`agent_picker.rs`、`agent_navigation.rs`、
> `agent_status_feed.rs`、`thread_transcript/*`（8 文件 47KB）、`collaboration_modes.rs`。
> 对照根（只读）：`frontend/src/pages/{PantheonPage.tsx,pantheon/*}`、`frontend/src/gaea/components/{SubagentThread,DagPanel,TaskInboxPanel}.tsx`、
> `frontend/src/gaea/lib/taskActivity*`、`frontend/src/pages/chat/*`。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、状态只有四个组，映射是**函数**不是散落的 if

`app/agents_overview_view.rs:69-100`：

| 组 | 判据（`for_status`，`:77-90`） | 标签（`label`，`:92-99`） |
|---|---|---|
| `NeedsYou` | `Active && (WaitingOnApproval \|\| WaitingOnUserInput)`，或 `SystemError` | `Needs input` |
| `Working` | `Active`（其余 active flag） | `Working` |
| `Ready` | `Idle` | `Ready` |
| `Finished` | `NotLoaded` | `Inactive` |

**注意 `SystemError` 归到 `NeedsYou`**——错误是"要你处理"，不是"结束了"。这条归类决定了排序与徽标。

徽标是第二层派生（`agents_overview_view.rs:428-438`）：

| 组 + 特例 | 文案 | 符号 |
|---|---|---|
| NeedsYou 且 `status == SystemError` | `Error` | `!` 红 |
| NeedsYou | `Needs input` | `●` 红 |
| Working | `Working` | `●` 绿 |
| Ready | `Ready` | `○` 青 |
| Finished | `Inactive` | `○` 暗 |

过滤条与计数同源（`app/agent_center/mod.rs:14-20`）：
`All / Needs you / Working / Ready / Inactive` 五项，**每项直接绑定组枚举**，不另写匹配串。

## 二、分组键 = Git 身份（common_dir + relative_cwd），不是目录字符串

`AgentsOverviewProjectGroup::for_thread`（`agents_overview_view.rs:123-141`）：
- worktrees 开启且能取到 `repository_identity(cwd)` → `key = (common_dir, relative_cwd)`，`heading = primary_root/relative_cwd`；
- 否则 `key = (cwd, "")`，`heading = cwd`。

`directory_group_key`（`:144-149`）：worktrees 开启时先 `dunce::canonicalize(cwd)`（Windows 长路径/UNC 归一），失败才用原路径。
即：**工作树里的多个会话会落到同一个项目组下**，而不会因为路径不同被拆成两组。

## 三、Row 只带六件东西

`AgentsOverviewRow`（`:103-110`）：`details`（详情投影）、`thread`、`thread_id`、`group`、`is_current`、`has_voice`。
标题口径（`:112-115`）：`name` 优先，否则 `preview`，取**首行**，空则 `Untitled task`。

> 这六件是"列表渲染所需的最小集合"——所有耗时字段（详情/用量）都在 `details` 里按需取，列表不背大对象。

## 四、刷新/删除时必须保住选区（两处小算法，值得照抄）

1. **刷新后**：`reconcile_command_center_selection`（`app/agent_center/mod.rs:31-38`）——
   可见集合里若不含当前选中项，就回落到首个可选项，没有则 `usize::MAX`（"加载更多"行）。
   同时 `show_more_selected = false`，避免刷新后焦点停在假行上。
2. **删除后**：`selection_after_removal`（`agents_overview_view.rs:316-327`）——
   先取当前可见位置，然后**从该位置向后找第一个未被删的行，找不到再向前回卷**（`chain(..position.rev())`），
   全部被删才返回 `None`。行为等价于"光标落到最近的一条幸存项"。

`selectable_indices`（`app/agent_center/mod.rs:23-30`）把"加载更多"折成 `usize::MAX` 塞进可选项列表——
**键盘导航不需要为假行写特例**，走到 `usize::MAX` 就是触发加载。

## 五、空态是五句话，不是一句"暂无"

`app/agent_center/rows.rs:60-75` 的判定顺序：

```
connection_notice 存在 -> "Reconnecting…"
loading && rows 空     -> "Loading tasks…"
refresh_failed         -> "Could not load tasks"
rows 空                -> "No tasks yet"
否则                    -> "No matching tasks"
```

顺序即优先级：**断线 > 加载中 > 刷新失败 > 真没有 > 只是没匹配**。
"刷新失败"与"没有任务"分开说，是这一组文案的全部价值。

## 六、列表列宽是"够宽才给元信息"

`rows.rs:33-58` `columns(area)`：`width >= 56` 时 metadata 24 列（status 11 + updated 9 + 间隙），否则 0；
gutter `min(width,4)`；title 吃掉剩余宽度。即**窄容器降级为"标题 + 状态点"**，而不是横向滚动。

## 七、子代理卡片与 agent picker

`multi_agents.rs`：

| 机制 | 证据 | 说明 |
|---|---|---|
| 三个预览上限 | `:30-32` | prompt 160 / error 160 / response 240 **grapheme**（不是字节、不是字符数） |
| 活动卡投影 | `:49-54 SubAgentActivityDisplay`、`:287-320` `sub_agent_activity_display` / `sub_agent_activity_history_cell` | 同一活动既能进主聊天，也能成历史 cell |
| 生成请求摘要 | `:71-74 SpawnRequestSummary`、`:189+ spawn_request_summary` | 从 `ThreadItem` 抽出"派了什么活" |
| 上一个/下一个 agent | `:106-134` `previous_agent_shortcut` / `next_agent_shortcut` / `*_matches` | 键位是**函数返回**，与键位表同一来源 |
| 词移动回退 | `:135-188` | 键与"按词移动"冲突时不抢键（`*_word_motion_fallback`） |

`thread_transcript/*` 把转录拆成四种页（`tools.rs` / `other_items.rs` / `computer_groups.rs` / `exploration_groups.rs` / `activity_pages.rs`），
外加 `PendingActivity`（`thread_transcript.rs:46`）与 `RawReasoningVisibility`（`:63`）两个可见性开关。

`collaboration_modes.rs`：`default_mask` / `plan_mask` / `mask_for_kind` / `next_mask`（`:14-65`）——
协作模式是**由模型目录过滤出来的 mask 列表**，`next` 是纯轮换，不硬编码顺序。

## 八、gaea 现状核对

| gaea | 现状 | 与 codex 的差 |
|---|---|---|
| `pages/PantheonPage.tsx`（48.9KB）+ `pages/pantheon/PantheonRecord.tsx`（140.5KB） | 议事厅/纪要（席位、发言、决议） | 那是"**一场讨论**"，不是"**在跑的任务总览**"：codex 的 AgentsOverview 是"所有会话/子代理一屏" |
| `gaea/components/SubagentThread.tsx`（33.4KB，两壳各一份） | 子代理会话转录（工具卡配对、正文/思考） | 有单线程视图，无"一屏多代理 + 分组 + 过滤" |
| `gaea/components/DagPanel.tsx`（30.7KB） | 有向图（任务依赖） | 图 ≠ 列表；子代理的"状态/是否需要我"没有一屏视图 |
| `gaea/components/TaskInboxPanel.tsx`（17.2KB）+ `lib/taskActivity` | 任务收件箱（后台任务） | 与"会话状态"是两套数据源，没有 codex 那种"Thread 即任务"的统一 |
| `pages/chat/*` + 侧栏 | 会话切换、项目分组 | 有项目分组（`projectGroups.ts`），但**分组键是路径字符串**，不是 git 身份；工作树场景会拆组 |
| 全仓 | 无"Needs input / Working / Ready / Inactive"四组状态机 | 子代理/任务状态散落在各自组件 |

## 九、三档判定

**档一（照搬语义）**

1. **四组状态机 + `SystemError → NeedsYou`**：所有"多代理/多任务"视图共用同一个 `groupForStatus()` 纯函数。
2. **徽标派生**（Error/Needs input/Working/Ready/Inactive 五态符号与色），与状态同源。
3. **选区不丢**：刷新回落 + 删除环形回卷 + "加载更多"折成 `usize::MAX` 的可行。
4. **空态五分支**（断线/加载/失败/空/未匹配），顺序固定。

**档二（改造搬）**

5. **分组键升级为 git 身份**：`projectGroups` 的 key 从 path 字符串改成 `(common_dir, relative_cwd)`，工作树归组自然正确（gaea 已有 `GaeaGitWorktrees` 与 `gitStatusStore` 可复用）。
6. **一屏代理总览**：新页面/面板聚合"主会话 + 子代理 + 后台任务"，四组过滤 + 右栏详情。
7. **预览上限用 grapheme**：`Intl.Segmenter` 计数，替代 `slice(0,60)`。

**档三（不搬）**

8. `agent_picker` 的上/下一个键位与词移动回退（终端键位冲突专属，Web 用 Tab 循环）；
9. `collaboration_modes` 的 mask 轮换（gaea 的模式是产品语义，不是模型目录投影）。

## 十、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F38-A 状态机纯函数** | 新 `gaea/lib/agentGroups.ts` | `groupForStatus(status, flags)` + `statusBadge(group, isError)` | 单测：`SystemError ⇒ NeedsYou/Error`；`WaitingOnApproval ⇒ NeedsYou`；`Idle ⇒ Ready` |
| **F38-B 选区算法** | 同上 `nextSelectionAfterRemoval(visible, selected, removed)` + `reconcileSelection(visible, selected)` | 环形回卷 + `usize.MAX` 假行 | 单测：删中间项 ⇒ 落下一项；删末尾 ⇒ 回卷上一项；全删 ⇒ null |
| **F38-C 空态五分支** | `TaskInboxPanel`/新总览面板 | 顺序固定 | 单测：失败态优先于空态文案 |
| **F38-D 分组键 git 化** | `gaea/lib/projectGroups.ts` | key 改 `{commonDir, relCwd}`，heading = primaryRoot/relCwd | 单测：两个 worktree 路径 ⇒ 同一组 |

**开关**：A/B/C 纯函数先行；D 触及侧栏分组，需 `FEATURE_GIT_GROUP_KEY` 单闸并保留旧键一版。

## 十一、风险

1. **`SubagentThread` 与 `ToolCard` 同类问题**：两壳双份（`gaea/` 与 `wenxue/` 各 33.4KB），F38 落地前先定单源方向（见 34 §十 F34-E）。
2. **数据源分裂**：会话、子代理会话、后台任务当前是三套 store；总览页要么做适配层（只读聚合），要么先统一 id。
   建议**只读聚合**：不改现有 store，新增 selector。
3. **`taskActivity` 已有节流**：新增总览订阅必须走 `perf-guards`，禁止在列表里对每行做时间格式化（`formatRelativeTime` 应 memo）。

## 十二、证据自查

- 四组与标签：`app/agents_overview_view.rs:69-100`；徽标：`:428-438`；过滤五项：`app/agent_center/mod.rs:14-20`。
- 分组键：`agents_overview_view.rs:117-149`。
- 选区：`:312-327`、`app/agent_center/mod.rs:23-38`。
- 空态：`app/agent_center/rows.rs:60-75`；列宽：`:33-58`。
- 子代理卡与键位：`multi_agents.rs:30-32,49-74,106-134,135-188,189+,287-320`。
- 转录与模式：`thread_transcript.rs:46,63`、`thread_transcript/*`、`collaboration_modes.rs:14-65`。
