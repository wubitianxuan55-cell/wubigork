# 资产 24 · 记忆引用与使用反馈：gaea 已领先，补三条定位精度

> 源码：`memories/read/src/{citations.rs 2.6 KB, usage.rs 2.4 KB, lib.rs}`。
> 对照：gaea `internal/gaea/memory/{citations.go 4.6 KB, injection_eval.go 6.8 KB}`。
> **关键发现**：gaea 的 `citations.go` 注释直接写着「蒸馏 codex memories/read 的 citation 闭环」——
> 这块 gaea 已经做过一轮，且在某些维度更完整。

---

## 一、codex 的做法：模型输出里带结构化引用块

模型在回答里输出两个块（不是事后检索记录，是**模型自己写的**）：

```
<citation_entries>
path/to/file:10-20|note=为什么这条重要
another/file.md:3-3|note=依据
</citation_entries>
<rollout_ids>
thread-id-1
thread-id-2
</rollout_ids>
```

解析规则（`citations.rs`）：
- `citation_entries` 每行 `路径:起始行-结束行|note=说明`，用 `rsplit_once('|')` 与 `rsplit_once(':')`
  逐层切——**从右往左切**，所以路径里带冒号（`C:\...`）也不会切错；
- `rollout_ids` 逐行去重（`HashSet`）后转 `ThreadId`；
- 两个块都为空则返回 `None`（不产出空引用）。

**用途**：引用能精确到「文件 + 行范围 + 一句为什么」，UI 可以直接跳转；
`rollout_ids` 则回答「这条记忆出自哪次会话」。

## 二、codex 的「使用」是推断出来的，不是模型自报

```rust
pub enum MemoriesUsageKind { MemoryMd, MemorySummary, RawMemories, RolloutSummaries, Skills }

pub fn memories_usage_from_command(command: &str) -> Vec<(MemoriesUsageKind, MemoryVersion)> {
    // ① 解析 shell 脚本成 ParsedCommand 序列
    // ② 只要出现一条 Unknown → 整体返回空（不猜！）
    // ③ Read{path} / Search{path} → 按路径归类；ListFiles / Unknown → 不算
}
```

路径归类：`memories/MEMORY.md` / `memory_summary.md` / `raw_memories.md` /
`rollout_summaries/` / `skills/`；版本看路径是否含 `memories_v2/`。

**两个值得抄的点**：

1. **用行为信号代替自报**：模型不必说「我用了记忆」，只要有读命令就算用过——
   比让模型自报可靠得多（模型经常忘记上报）。
2. **`Unknown` 就不猜**：脚本里有一条无法解析的命令，整条记录作废。
   宁可少统计，也不要错误归因。

## 三、gaea 现状（对照结论：已领先两处）

| 维度 | codex | gaea | 判断 |
|---|---|---|---|
| 引用触发 | 模型输出 `<citation_entries>` 块 | 模型在句末写 `[MEM:key]` | 各有取舍：codex 块更好解析，gaea 内联更省 token |
| 引用内容 | 路径 + 行范围 + note | 记忆键（映射到记忆记录） | **codex 更精确**（能跳到文件行） |
| 使用反馈 | shell 读命令推断 | 解析引用键 → `Touch`（更新 `last_used_at`） | gaea 更直接（一次解析两用） |
| 悬空引用 | 无显式处理 | `CitationResult{Resolved, Dangling}` 悬空键留痕不触达 | **gaea 更完整** |
| 注入质量 | 无对应 | `injection_eval.go` 五条不变量（预算×2 / 归档零泄漏 / 跨空间零泄漏 / 引用零悬空） | **gaea 明显更强** |
| 版本归属 | 路径 `memories_v2/` 判版本 | 未见同等机制 | codex 有一点可借 |

## 四、三条可借（小改，1–2 小时）

1. **引用带定位**：`[MEM:key]` 旁边允许可选定位后缀，例如
   `[MEM:cost-rule@会话id#消息id]` 或 `[MEM:cost-rule@文件:12-18]`；
   解析器兼容旧格式（无后缀即当前行为），UI 徽标有定位时给「跳到原文」按钮。
2. **悬空的展示**：gaea 已有 `Dangling`，建议在 UI 上做成「这条引用找不到依据」的黄色徽标，
   而不是静默忽略——这是记忆可信度的直接信号。
3. **版本绑定**：给记忆记一个 `schema_version`，引用与指标都带版本，
   将来记忆结构升级时能区分「旧记忆被引用了多少次」。

## 五、不建议照搬

- **用 shell 读命令推断使用**：gaea 的记忆在 SQLite 里，没有「读文件」这个动作；
  对应物是 `recall` 工具调用与 UI 打开事件——要用行为信号，也该用这两个，而不是文件读。
- **两个引用块（entries + rollout_ids）**：gaea 的记忆本身带 `Source`，不需要额外块；
  多一个块就多一处解析失败点。
- **`MemoriesUsageKind` 五类**：那是 codex 的文件布局（MEMORY.md / summary / raw / rollouts / skills），
  gaea 的记忆分类不同，硬套会失真。

## 六、一句话

引用这块 gaea 已经比 codex 完整（有悬空处理、有注入质量评测），
唯一值得补的是**定位精度**：让引用能指到「哪次会话、哪段原文」，而不是只指到记忆记录。
