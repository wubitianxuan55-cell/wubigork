# 资产 06 · 智能体角色（数据化）与生成拓扑（极简契约）

> 源码：`codex-rs/agent-roles/`（`agent_role_config.rs` 7 KB / `discovery.rs` 1.2 KB / `loader.rs` 12.3 KB）、
> `codex-rs/agent-graph-store/`（`store.rs` 2.6 KB / `types.rs` 1.4 KB / `local.rs` 12 KB）、
> `codex-rs/agent-identity/`（38 KB）。
> 对照对象：gaea `internal/pantheon`（60 文件，`manager.go` 121 KB / `prompts.go` 50 KB / `types.go` 33 KB）
> 与 `internal/gaea/agent/subagent_store.go`（35 KB）、`trajectory/agentnet.go`（5.4 KB）。

---

## 一、先说结论（省得白干）

| 部分 | codex 做法 | gaea 现状 | 判断 |
|---|---|---|---|
| **角色定义** | 角色是**文件**：`description`（必需，给 spawn 工具当说明）+ 可选独立 config 层 + 昵称候选 | 席位/角色写在 `pantheon` 代码里（`prompts.go` 50 KB），只有 `agents.json` 存注册信息 | **值得搬**：让用户定义席位不必改代码 |
| **生成拓扑** | 只存**线程 spawn 边**：父、子、状态（open/closed），子最多一个父 | `subagent_store` 有 `continue_from` 关系；`trajectory/agentnet.go` 已能从日志折叠出智能体网络 | **不建议搬**：gaea 已有更全的折叠视图 |
| 身份 | `agent-identity`（38 KB，能力/身份串） | 席位 ID + 认证台账（`attestation.go`） | 各有各的，无需互换 |

---

## 二、角色数据化（真正值得抄的部分）

```rust
pub struct AgentRoleConfig {
    /// spawn 工具引导里展示给模型看的角色说明（用户自定义角色必填）
    description: String,
    /// 该角色专属的 config 层路径（可覆盖模型、沙箱、审批等）
    config_layer: Option<PathBuf>,
    /// 用这个角色 spawn 出来的智能体可选昵称
    nickname_candidates: Vec<String>,
}
```

三个要点：

1. **`description` 是给模型看的**，不是给人看的——它直接进 spawn 工具的引导文本。
   所以它是**必需**字段（加载后归一化，空则拒绝）。
2. **`config_layer` 允许角色覆盖配置**：一个「审稿人」角色可以自带更严的审批策略、
   更小的上下文预算，而不影响其他角色。这是 gaea 现在做不到的（席位共用一套配置）。
3. **`nickname_candidates`**：同一角色多次 spawn 时取不同昵称，避免「五个审稿人」在日志里混淆。

加载链路：`discovery::collect_agent_role_files()` 扫角色文件 → `parse_agent_role_file_contents()`
解析（含废弃字段与新字段的优先级归并）→ 与 `ConfigToml` 的 `[agents.<name>]` 合并 → 得到可 spawn 角色表。

**对 gaea 的落点**：`internal/pantheon` 的席位定义从代码搬到文件（例如 `agents/<role>.md` + frontmatter，
或 `agents.toml`），保留 `agents.json` 作为运行期注册表。收益：
- 用户能自定义第五席位（现在只能改代码）；
- 每个席位可带自己的审批/预算（对应清单第 3、4 项的落地前提）；
- 席位说明与实现分离，改人设不用重新编译。

**验收**：新建一个角色文件即可在 UI 出现该席位；删除后不残留注册；角色 description 出现在 spawn 引导里；
角色 config 覆盖只影响该席位（用两个席位跑同一任务对比审批行为）。

---

## 三、生成拓扑：契约简单，但 gaea 不需要

`AgentGraphStore` 只有四个方法 + 四条不变式（注释里写得很清楚）：

```
upsert_thread_spawn_edge(parent, child, status)   // 同一个 child 重复 upsert = 更新父与状态
set_thread_spawn_edge_status(child, status)       // 缺失的 child 视为成功 no-op
list_children(parent, status_filter)              // 稳定排序，便于与内存态合并
```

不变式：
1. 一个 child **最多一个**持久化父；
2. 重复 upsert 要**同时**更新父与状态（不能只改一半）；
3. 对不存在的 child 设状态 **不算错误**（幂等）；
4. 列表要有**稳定顺序**，否则「持久化图 + 内存图」合并会出现不确定输出。

**为什么 gaea 不需要**：`trajectory/agentnet.go` 已经能从会话日志折叠出智能体网络
（`FoldAgentNetwork(entries, window)`，含节点聚合、运行态判断），比这张「只存 spawn 边」的表信息更多；
`subagent_store` 的 `continue_from` 也已表达父子延续关系。

**只在一种情况下再考虑**：如果将来需要「跨进程/重启后仍能查询某父线程的所有子线程状态」而日志折叠又太慢，
再引入这张表——此时按上面四条不变式实现即可。

---

## 四、不建议照搬

- **`agent-identity`（38 KB）**：它解决的是「身份凭据 + 能力声明」的跨进程信任问题，
  gaea 是单机单用户，席位的 `attestation.go` 台账已经够用；
- **`agent-graph-store` 的持久化层**：理由见上，日志折叠已覆盖；
- 角色文件的**废弃字段优先级归并**逻辑：gaea 从零设计角色格式，没有历史包袱，不必背兼容代码。

## 五、本轮给清单第 10 项的收口结论

第 10 项「多智能体消息板 + 图谱」拆成两块已有结论：
- **消息板**（asset-04）：值得借「预览/正文分离 + 字符偏移续读 + 拒绝错误预算化」，异步协作只在真实需求出现时做；
- **图谱/角色**（本篇）：角色数据化值得做（中等收益），图谱不值得搬（gaea 折叠视图更强）。
故第 10 项从「大优点」降级为「一条中等收益（角色数据化）+ 一条已覆盖（图谱）」。
