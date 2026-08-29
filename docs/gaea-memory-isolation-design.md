# gaea 双空间 S1.2 记忆空间隔离器 · 设计

> 依据：只读勘察报告（2026）。S1.1 已完成（facts/tasks space_id、ListInSpace、session.Space、
> space.mode、spaces 包）。本文件是 S1.2 实现权威；三步入：A 写侧+dream 空间化 → B 读端隔离器 → C 前端 scope。

## 检索面地图（全部已定位）
| 组 | 位置 | 过滤方式 |
|---|---|---|
| keyword | `internal/app/gaea_unified_search.go:74`（wssearch 全工作区） | 工作区单根目录=共享面，**不过滤**；但 wssearch.go:54-61 噪音规则需加 `.gaea/play/exports`（现只跳 sessions/archive/cache，play 产物会漏进 work 检索） |
| brain（三脑） | `brain_bindings.go:14` 适配器 | facts 走 ListInSpace；**whisper 右脑=play 专属**：work scope 下滤掉 brain.right（brain.left/mid 属 work） |
| semantic | `gaea_semantic_search.go:66`（共享 semantic_vectors 按 kind） | **不能过滤索引源**（Ensure/Stale :96-142 会经 Stale(keep) 删另一空间向量）→ 只对最终 hits 后过滤 |
| files | `gaea_file_index.go:96` | 文件语义检索按会话/工作区共享面处理（同 keyword） |

前端：hub 搜索 `MemoryHubPage.tsx:140 runSearch → app.UnifiedSearch`；`WorkspaceSearchPanel.tsx:94,114,131` 三个调用点；当前空间来源 `SpaceChip.tsx:21`（S4 已有 SpaceActiveView）。

## 写侧漏洞（A 步修）
1. **dream 管线无空间**：TurnDone 事件槽（gaea_handler.go:77）→ maybeDreamAfterTurn → runDream（gaea_dream.go:105）→ SaveDreamFacts（controller_memory.go:280）不带 Space → sqlite 缺省 work（sqlite.go:46）。指纹 dreamInputHash（gaea_dream.go:184）不含 space → 同内容跨空间会误判 no-op。
2. **remember 工具泄漏**：remember.go:87 不盖 Space → play 会话记忆默认落 work。
3. **dream notes 走 QuickAdd 到共享 AGENTS.md**（gaea_dream.go:164）绕过 facts 表——空间化时注明（AGENTS.md 属 work 项目说明，play dream 不写它或经空间目录分支，按 A 步方案定）。
4. citations：ResolveCitations（citations.go:44）→ Store.Get/Touch（sqlite.go:151,326）均无 space 谓词；listInSpace SELECT 未回填 space_id（sqlite.go:173）→ B 步补回填（供前端展示/调试）。

## 方案（三步，可独立提交/回退）
- **A 写侧+dream 空间化**：ctx 携带 space（S3 已有 WithSpace/SpaceFromContext）→ remember 工具 Save 带 space；dream 链路 session space → SaveDreamFacts 带 space + dreamInputHash 指纹键含 space（防跨空间 no-op 误判）+ dream-audit.jsonl 加 Space 列（审计可追溯）。
- **B 读端隔离器**：memory 包内 `spaceList` 单点 helper（space=""=全部/旧行为，非空=ListInSpace）；`Store.Load` 加 Space 选项、新增 GetInSpace/TouchInSpace；`GaeaUnifiedSearch` 加 scope 参数（""=全部/旧行为，"work"/"play"）——**签名变更，绑定面计数不变**；四组过滤按上表（semantic 只滤最终 hits）；brain 组 work scope 滤 right。
- **C 前端 scope 切换**：hub 搜索 + WorkspaceSearchPanel 传 scope（默认当前空间，提供「工位/乐园/全部」显式切换）；复用 SpaceChip 的当前空间来源。

## 验收红线（§15）
工位搜索/记忆检索永远搜不到乐园记忆，反之亦然；play 会话 dream 不写入 work 记忆；scope=""=全部仅显式选择时使用。

## 风险
- semantic Stale 陷阱（见上）——实现时严禁按空间过滤索引源。
- GaeaUnifiedSearch 签名变更会触碰前端 hub/面板调用点与 mock（mock/core.ts 需同步）。
- 绑定面计数不变（签名变更），但 bridge.ts 类型需同步。

## 勘误与关键锚点（完整勘察报告补录）
- remember 工具在 **`internal/gaea/memory/remember.go:87-95`**（非 agent 包）；ctx 空间管线仿 `memory.WithQueue/WithSessionSaver`（memory/queue.go:15-41，注入点 execute_one.go:147-150）。
- dream：gaea_handler.go:77-79 事件槽不带上下文 → 取 ctrl.SessionSpace()（回退 gaeaEffectiveSpace()，mode=off=""）→ gaea_dream.go:82,105 runDream(space) → :184 指纹 = sha256(space+"\x00"+input) → :147 SaveDreamFacts（controller_memory.go:295 空 Space 统一 Normalize 兜底）→ :164 notes QuickAdd：**play dream notes 丢弃+slog.Debug**；审计 DreamAuditEntry（controller_memory.go:20-25）加 Space 列（JSONL 追加列向后兼容）；显式路径 gaea_memory_suggestions.go:199 同点盖章。
- citations：ResolveCitations（citations.go:44-65）内部 Get/Touch → GetInSpace/TouchInSpace（space="" 走旧方法）；touchMemoryCitations（control/controller.go:488-493）传 c.SessionSpace()；memory_get 工具（memory/get.go:50-55）同步换。
- semantic Stale 陷阱：gaea_semantic_search.go:96-101,138-142 Ensure+Stale 会物理删另一空间向量——只对最终 hits 后过滤（kind=="office" 按 ListInSpace(scope) 映射；cost/knowledge/file 不过滤）。
- brain：左脑 brain_left.go:69-71 List→ListInSpace(scope)；右脑（whisper=play 整域）work scope 整体丢弃；profile+knowledge 共享不过滤。
- **UNIQUE(project,name) 跨空间同名冲突**：ON CONFLICT DO UPDATE 会覆盖并翻转 space_id——本刀不改约束，盖章路径在「目标名已存在且 space 不同」时打 audit 警告（可溯源）。
- **listInSpace SELECT 补取 space_id 回填 m.Space**（sqlite.go:173；store.go:70 注释随之更新）。
- **mode=off 三态回退**：scope=""/SessionSpace=""/EffectiveSessionSpace="" 必须严格等价旧行为（不过滤/不写 space/纯内容哈希指纹）——任何一处把 "" 归一成 work 都会让 off 模式旧数据不可见。
- 未接线项：ProceduralBlock/EpisodicMatches/EpisodicBlock 无活调用方（接线时必须走 spaceList）；hub 库列表前端消费面未清点（留 S2.1）；dream 触发仅 app 事件槽一处（实现时全库 grep TurnDone 复核）。
