# 资产 43 · 工作树与环境：owner 四态、目标落文件不塞 prompt、可选字段不是失败

> 证据根（只读）：`worktree_browser.rs`（168）、`managed_worktree_tools.rs`（338）、`workspace_command.rs`（209）、
> `additional_dirs.rs`（149）、`projectless.rs`（110）、`oss_selection.rs`（462）、
> `goal_display.rs`（111）、`goal_files.rs`（245）、`branch_summary.rs`（755）。
> 对照根（只读）：`frontend/src/gaea/components/{GitPanel.tsx,FileTree.tsx,DagPanel.tsx}`、
> `frontend/src/gaea/lib/{gitStatusStore.ts,branches.ts}`、`frontend/src/pages/chat/branches.ts`。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、工作树浏览：`Owner` 四态决定能做什么

`worktree_browser.rs:20-47`：

```rust
struct Entry { root: PathBuf, cwd: PathBuf, owner: Owner }
enum Owner { None, Unavailable(ThreadId), Archived(ThreadSummary), Resumable(ThreadSummary) }
enum Action { Resume(ThreadId), Copy(PathBuf), Remove(PathBuf) }
```

- `ThreadSummary{id, title, updated_at}`——工作树关联的会话只带三件；
- `Unavailable(id)` 表示"知道有这么个会话，但元信息没取到"，`fetch`（`:49-…`）会为它补一次 `ThreadRead`，
  并跨 `codex_home/archived_sessions` 找归档记录；
- 动作与 owner 绑定：可续（`Resumable`）才 `Resume`，归档的只能 `Copy`/`Remove`，无属主则不给 Resume。

> 这条比"工作树列表 + 一个删除按钮"多出来的东西是：**每个工作树属于哪个会话、还能不能接着干**，
> 并且"元信息缺失"是一个显式状态而不是空字段。

## 二、工作树操作被工具化（模型可用，不只人点）

`managed_worktree_tools.rs:33-70`：`ManagedWorktreeTools` + `Operation` + 三个参数结构
`CreateArguments` / `StatusArguments` / `ListArguments`。

即：创建/查询状态/列工作树是**三个工具动作**，各自有 schema。
好处是"人和模型走同一条路径"，不会出现"UI 能建、模型不能建"的两套实现。

## 三、工作区命令有明确的输出/错误结构

`workspace_command.rs:37-147`：`WorkspaceCommand{…}`、`WorkspaceCommandOutput{…}`、`WorkspaceCommandError{…}`、
`AppServerWorkspaceCommandRunner`。
命令执行不是"跑一个 shell 返回字符串"，而是**请求/输出/错误三段 + 一个 runner 抽象**（可换实现、可测）。

`additional_dirs.rs`（149 行）：额外目录（工作区外的可读路径）是独立概念，不是往 cwd 里塞。

`projectless.rs`（110 行）：无项目（不在任何仓库里）的会话也有独立路径处理——
"没有 git 仓库"是一等场景，不是异常分支。

## 四、目标（goal）落成文件 + 一句引用，不塞进 prompt

`goal_files.rs:17-38`：

```
GOAL_ATTACHMENT_DIR = "attachments"
GOAL_FILE_NAME      = "goal-objective.md"
GOAL_FILE_PREFIX    = "Read the Codex goal objective file at "
GOAL_FILE_SUFFIX    = " before continuing."
```

`materialize_goal_draft`（`:33-…`）的处理顺序：
1. 目标文本必须非空（含展开粘贴占位符后的再判空，`:39-52`）；
2. 大粘贴**各自落文件** `pasted-text-{n}.txt`，占位符替换成
   `"pasted text file: {path}. Read this file before continuing."`（`:69-77`）；
3. 本地图片逐张处理，占位符替换为文件引用行（`:80-…`）。

> 机制要点：**长内容不占上下文，落盘 + 引用**；引用句是**可执行指令**（"继续之前先读这个文件"），
> 所以模型必须真的去读，而不是"prompt 里有一段很长的目标"。

`goal_display.rs:5`：`GOAL_USAGE = "Usage: /goal [<objective>|clear|edit|pause|resume]"`——
目标的五个操作（设定/清除/编辑/暂停/恢复）是**一条 usage 字符串**定义的，帮助与解析同源。

## 五、状态行里的 Git 摘要：缺字段是"省略"，不是失败

`branch_summary.rs:24-57`：

- `GitBranchDiffStats{additions, deletions}`（HEAD 与比较基准之间）；
- `StatusLineGitSummary{pull_request?, branch_change_stats?}`，注释写死：
  「两者可能只有其一对，**渲染方应把缺字段当作省略的可选 UI，而不是硬查询失败**」；
- `StatusLinePullRequest{number, url}`，注释：**只收 open 的 PR**，closed/merged 在本模块内过滤；
- `DefaultBranch` 用 merge-base 比较，且基准可能是 `refs/remotes/origin/main` 这类远端跟踪引用。

`oss_selection.rs`：provider 选择（本地 OSS 模型）有 `ProviderStatus`、`MOVE_LEFT/RIGHT_KEYS` 两组键常量（`:85-94`），
左右键是"显式常量数组"，不是散在 match 里。

## 六、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `internal` 侧 `GaeaGitWorktrees` + `gaea/components/GitPanel.tsx`（29.4KB） | 工作树**只读展示**（G1.6），Git 四/五镜头（状态/差异/提交图/详情） | 无 `Owner` 概念：不知道某个工作树属于哪个会话、能否续跑 |
| `gaea/lib/gitStatusStore.ts` | 统一 git 状态（30s/focus/in-flight 去重） | 可复用为 owner 探测的数据源之一 |
| `FileTree.tsx`（54.1KB） | 文件树 + git 着色 + 排除规则 | 与工作树浏览是两件事，已分离（好） |
| `pages/chat/branches.ts`（`btOf/dbIdOf`） | 会话分支/版本树 | 是**消息分支**，不是 git 分支；两者命名需持续区分 |
| 全仓 | 无目标（goal）机制 | 缺"长期目标落文件 + 引用"这套上下文瘦身法 |
| 全仓 | 无"额外目录/无项目"概念 | gaea 的 cwd 是单值，工作区外路径需显式配置（现在靠 excludes/prefs） |

## 七、三档判定

**档一（照搬语义）**

1. **owner 四态**（None/Unavailable/Archived/Resumable）：工作树列表每行标"属于哪个会话、能否续跑"。
2. **动作跟 owner 走**：不可续的不给 Resume 按钮（不是按了才报错）。
3. **缺字段即省略**：Git 摘要类信息（PR/增删行）缺失时省略该段，不整块报错。
4. **PR 只收 open**：过滤在数据层，渲染层不再判状态。

**档二（改造搬）**

5. **目标落文件**：长目标/大粘贴/图片落 `attachments/`，prompt 里只留一句"读这个文件再继续"。
6. **工作树操作工具化**：create/status/list 三个动作同时给人（按钮）与模型（工具）。
7. **`/goal` 五操作 + usage 单串**：帮助文本与命令解析同源。

**档三（不搬）**

8. `AppServerWorkspaceCommandRunner` 的具体实现（gaea 有自己的 exec 通道）；
9. `oss_selection` 的 OSS provider 选择（gaea 的本地引擎走 ModelCenter，语义不同）。

## 八、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F43-A owner 标注** | `GitPanel` 工作树镜头 + `lib/worktreeOwners.ts` | `ownerFor(worktree, sessions)` 纯函数；行内 chip（可续/归档/未知） | 单测：会话在工作树内 ⇒ Resumable；仅归档 ⇒ Archived；无匹配 ⇒ None |
| **F43-B 动作守卫** | 同上 | Resume 仅对 Resumable 渲染 | 单测：Archived 行无 Resume；Unavailable 行显示"信息获取中" |
| **F43-C 目标落文件** | 新 `lib/goalFiles.ts` + ChatPage 发送前钩子 | 超阈值文本/图片落 attachments，正文替换为引用句 | 单测：>8KB 文本 ⇒ 落文件 + 引用句；引用句含绝对路径 |
| **F43-D 可选字段省略** | `GitPanel`/状态条 | PR/增删行缺则整段省略 | 单测：两个字段都缺 ⇒ 不渲染 Git 段且不报错 |

**开关**：A/B 只读聚合；C 触及发送链，`FEATURE_GOAL_FILES` 单闸并保留旧路径；D 为渲染层。

## 九、风险

1. **`owner` 需要"会话 ↔ 工作树"的映射**：gaea 会话存的是 cwd 字符串（`SessionMeta.path`），
   工作树切换后 cwd 不同 → 需要按 git 身份（`common_dir + relative_cwd`，见 38 §二）归一，否则永远匹配不上。
2. **目标落文件与工作区边界**：`attachments/` 必须落在工作区内（gaea 有工作区边界单点收口），否则工具读写被拒。
3. **`/goal` 不能与既有"任务清单"混**：gaea 已有 `todo_write` 待办与任务管理页，goal 是"长期目标"，需在 UI 上分层。

## 十、证据自查

- 工作树浏览：`worktree_browser.rs:20-64`；工具化：`managed_worktree_tools.rs:33-70`。
- 命令/目录/无项目：`workspace_command.rs:37-147`、`additional_dirs.rs`、`projectless.rs`。
- 目标落文件：`goal_files.rs:17-77`；usage 串：`goal_display.rs:5`。
- Git 摘要与 PR：`branch_summary.rs:24-64`；OSS 选择键：`oss_selection.rs:85-94`。
