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
