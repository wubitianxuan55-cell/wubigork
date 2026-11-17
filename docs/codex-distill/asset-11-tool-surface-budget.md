# 资产 11 · 工具面预算施工级：三层结构（预算 / 目录差分 / 检索展开）

> 源码：`core/src/context/world_state/{tools.rs,tools_budget.rs,top_level_tools.rs}`、
> `core/src/tools/handlers/tool_search.rs`（24.9 KB）、
> `tools/src/{tool_search.rs,tool_discovery.rs,responses_api.rs}`（`coalesce_loadable_tool_specs`）。
> 对照：gaea `internal/gaea/tool/{tool.go,registrymap.go}`（已有 `Hide/HideUnlessOnly/RemovePrefix/FilteredSchemas`
> + `CompactDescriptor` 两套 schema）。

---

## 一、三层结构（缺一层都不成立）

| 层 | 文件 | 干什么 | 缺了会怎样 |
|---|---|---|---|
| ① 预算 | `tools_budget.rs` | 把「顶层面」的 schema 文本压进字节预算 | 工具一多，首轮就爆 |
| ② 目录差分 | `top_level_tools.rs` | `name → hash` 快照，只发 `added/removed` | 每轮重发全部 schema，缓存前缀失效 |
| ③ 检索展开 | `tool_search` | BM25 检索延迟工具，**只把选中的**落成 schema | 藏起来但模型找不到 |

**关键认知**：省 token 靠的是 ②③，①只是兜底。只做①（截断描述）会让工具「看得见但看不懂」；
只做②（差分）在工具集突变时仍会一次性灌入大量 schema。三层配套才既省又可用。

## 二、目录差分：`name → hash` 快照

```rust
type Snapshot = BTreeMap<String, WorldStateHash>;   // 名字 → 定义 JSON 的哈希
const ACTION_ADDED: &str = "added";
const ACTION_REMOVED: &str = "removed";
```

三条纪律（文件头注释）：
1. **一个增量批次只加一条 developer 通知**——不是每个工具一条；
2. **`catalog hashes are unchanged`**：定义没变就不重发，即使内部重建；
3. **整命名空间移除时吞并其成员**——删一个命名空间不产生 N 条删除通知。

**这就是前缀缓存友好的实现方式**：模型看到的工具列表是「累积状态」，每次只追加变化，
哈希相同 = 零输出。

## 三、检索展开：只物化被选中的

```rust
pub const TOOL_SEARCH_DEFAULT_LIMIT: usize = 8;
pub const TOOL_SEARCH_TOOL_NAME: &str = "tool_search";

pub struct ToolSearchEntry { pub search_text: String, spec: Arc<ToolSpec> }
impl ToolSearchEntry {
    /// 迭代候选身份，但不物化 schema
    pub fn tool_names(&self) -> impl Iterator<Item = ToolName> + '_ { .. }
    /// 只对选中的结果物化；输出的 schema 在被丢弃前共享
    pub fn to_loadable_spec(&self) -> LoadableToolSpec { .. }
}
```

四个设计要点：

1. **索引里只放 `search_text` + `Arc<ToolSpec>`**：内存里一份 schema，检索条目只是引用，
   不会因为「工具多」而把内存翻倍；
2. **`tool_names()` 可以只列身份不建 schema**：想告诉模型「搜到 5 个工具叫 X/Y/Z」时，成本极低；
3. **`to_loadable_spec()` 才物化**：默认 limit = **8**，一次最多落 8 个；
4. **`ToolSpec::ToolSearch / WebSearch` 不进索引**（`from_shared_spec` 直接过滤）——
   避免「搜索工具自己出现在搜索结果里」。

检索实现用 **BM25**（`bm25::Embedder / Scorer / Language`），字段就是 `search_text`；
命中后由 `coalesce_loadable_tool_specs` 合并进本轮的 responses-API 工具清单。

## 四、和「延迟」相关的两个特性开关（默认关，别搞混）

- `deferred_tool_world_state`：世界状态里的工具面延迟渲染；
- `incremental_tools`：增量下发工具面。

两者都在 `features/src/lib.rs`，**默认关闭**（有测试专门断言 deferred 默认关）。
换句话说：**默认路径仍是「全量下发」**，这套机制是给大工具集的优化档。

---

## 五、gaea 施工方案（三步，可分别验收）

### 步一：目录差分（最省事、收益最直接，0.5–1 天）

1. 在 `tool/registrymap.go` 旁新增 `tool/catalog.go`：
   ```go
   type Catalog struct { hashes map[string]string }   // name → sha256(定义 JSON)
   func (c *Catalog) Diff(next map[string]json.RawMessage) (added, removed []string, changed bool)
   ```
2. 每轮 assemble 时算一次差分，**只有变化才把这些工具的 schema 放进上下文**，其余用一句
   「工具目录无变化」占位；一个批次只加一条通知；
3. 整命名空间移除时只报命名空间。

**验收**：连续三轮工具集不变 → 后两轮的上下文里工具部分**逐字节相同**（0 新增）；
新增一个工具 → 只有该工具的 schema 出现 + 一条通知；删除整个命名空间 → 一条通知，不出现 N 条。

### 步二：检索展开（1–1.5 天）

4. 新增 `tool/search_index.go`：索引项只存 `{name, namespace, searchText, specRef}`，
   `specRef` 指向注册表里的同一份 schema（不要复制）；
5. 检索用现有 `bm25`（gaea 已有）：`ToolSearch(query, limit=8)` 返回身份列表 + 得分；
6. 命中后**只把这 ≤8 个的 schema 落进本轮上下文**，并标 `loadable`；
7. 加一个工具 `tool_search`（名字与语义对齐 codex），参数只留 `query` 与可选 `limit`。

**验收**：注册 40 个工具、首轮只下发 12 个，模型调 `tool_search("导出 xlsx")` 后
本轮出现目标工具的 schema 并成功调用；同一 query 两次返回顺序一致（确定序）。

### 步三：字节预算（兜底，1 天）

8. 按 `hotspot-06` 的算法实现 `tool/budget.go`：名字优先全额、描述轮次均分、`...` 截断、
   omission 预留，UTF-8 字符边界；
9. 默认关闭（`tools.schema_budget.enabled`），关时与现状逐字节等价。

**验收**：预算收紧到原来 50% 时，工具**名字全部保留**，只有描述被截；同输入两次输出逐字节相同。

## 六、不建议照搬

- **`Responses API` 的 namespace / `coalesce_loadable_tool_specs`**：那是 OpenAI responses 协议的形状，
  gaea 的 provider 抽象不一样，抄语义即可，不必抄数据结构；
- **`indirect_description_prefixes` / `functions_namespace_function_prefixes`**：为长命名空间做的前缀压缩，
  工具数没到几十个别上；
- **`list_available_plugins_to_install` / `request_plugin_install`**：插件市场能力，gaea 无对应。

## 七、一句话

工具面省 token 的正确姿势不是「少给工具」，而是**目录只发差分、检索才给全量、描述按预算裁**。
三层里最划算的是第②层目录差分——半天工作量，换来「工具不变就零输出」。
