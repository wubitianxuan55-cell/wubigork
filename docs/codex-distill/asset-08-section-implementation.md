# 资产 08 · 章节化施工细节：一个 section 到底怎么写

> 源码：`core/src/context/world_state/`。
> 本篇不重复 asset-02 的「为什么章节化」，只给**落地时真正要写的那几十行**：
> section 的五个成员、render_diff 的三种返回、替换/移除通知的纪律、`tools` 与 `top_level_tools` 的分工。

---

## 一、最简单的完整 section：`context_window_guidance`（2.7 KB，全文可读）

它的状态就一个字符串，但把「差分」的三种情形写全了：

```rust
const REPLACEMENT_NOTICE: &str =
    "This context-window guidance replaces all previously provided context-window guidance.";
const REMOVAL_NOTICE: &str = "The previously provided context-window guidance no longer applies.";

impl WorldStateSection for ContextWindowGuidanceState {
    const ID: &'static str = "context_window_guidance";
    type Snapshot = String;

    fn matches_legacy_fragment(role: &str, text: &str) -> bool {
        role == "developer" && ContextWindowGuidance::matches_text(text)   // 用标记判定
    }

    fn render_diff(&self, previous: PreviousSectionState<'_, String>)
        -> SectionTransition<String>
    {
        // ① 与上一轮完全相同 → 什么都不发
        if matches!(previous, PreviousSectionState::Known(p) if p == &self.message) {
            return (None, Vec::new());
        }
        // ② 上一轮有没有内容，决定这次要发「替换」还是「移除」
        let previous_may_contain_guidance = match previous {
            PreviousSectionState::Known(p) => !p.is_empty(),
            PreviousSectionState::Unknown => true,     // 不确定就当有，保守
            PreviousSectionState::Absent => false,
        };
        let message = if self.message.is_empty() {
            if !previous_may_contain_guidance { return (Some(self.message.clone()), Vec::new()); }
            REMOVAL_NOTICE.to_string()                  // 现在没有 → 告诉模型「之前那条作废」
        } else if previous_may_contain_guidance {
            format!("{REPLACEMENT_NOTICE}\n\n{}", self.message)   // 有新的 → 明确「替换」+
        } else {
            self.message.clone()
        };
        (Some(self.message.clone()), vec![WorldStateUpdate::fragment(ContextWindowGuidance::new(&message))])
    }
}
```

### 三个必须抄的细节

1. **相同就不发**：`Known(p) if p == current` → `(None, [])`。这是整章节化省 token 的唯一来源，
   如果这条写不出来，章节化只剩形式。
2. **`Unknown` 当成「有」**：历史里查不到上一轮状态时按「可能有旧内容」处理，于是发**替换**而不是直接发新内容——
   宁可多一句通知，也不让模型以为「现在有两条指引」。
3. **空值要发「移除通知」**：以前有、现在没有，必须显式说「之前那条不再适用」，
   否则模型会继续遵守一条已撤销的指引。

## 二、`SectionTransition` 的三种返回

| 返回 | 含义 | 何时用 |
|---|---|---|
| `(None, [])` | 本轮**什么都不说** | 状态与上一轮完全一致 |
| `(Some(snapshot), [..])` | 记录新快照 + 发片段 | 新增/替换/移除 |
| `(Some(snapshot), [])` | 只更新快照、不发片段 | 状态变了但模型不需要知道（例如内部计数） |

快照**只放比较所需的数据**（这里是那个字符串），且不能序列化成 `null`（历史里 null 表示删除）。

## 三、`tools` 与 `top_level_tools` 的分工（别混）

两章节都跟工具面有关，但职责不同，文件头注释写得很清楚：

| 章节 | 文件头注释原意 | 职责 |
|---|---|---|
| `tools` | 「有界的**延迟命名空间摘要**：名字优先于描述预留；渲染过程中的分配不改变已持久化的命名空间快照」 | 给**延迟展开**的工具做摘要，受 `tools_budget` 字节预算约束 |
| `top_level_tools` | 「对**单层响应式目录**做差：可调用工具的命名空间 + 内建；缺失状态视为全新目录，**不迁移历史**；每个增量批次前加一条 developer 通知；目录哈希不变；整命名空间移除时通知**吞并**其成员」 | 给**顶级工具目录**做增量差分，带批次通知与哈希 |

两条纪律值得注意：
- **「整命名空间移除的子条目，不再单独通知」** —— 避免「删一个命名空间」产生十几条删除通知；
- **「目录哈希不变」** —— 工具的 schema 没变时，即使内部结构重建也不产生 diff。

## 四、给 gaea 的最小实现模板（Go）

```go
// Section 是回合内可差分的上下文片段。ID 必须稳定且随会话持久化。
type Section interface {
    ID() string
    Snapshot() any                    // 只放比较所需数据，序列化不得为 null
    RenderDiff(prev Previous) (next any, out []Fragment)
}

type Previous struct {          // 三态，别用 *T 两态
    Kind  PrevKind                // Known / Unknown / Absent
    Value any
}

// 最小实现示例
func (s GuidanceSection) RenderDiff(prev Previous) (any, []Fragment) {
    cur := s.Message
    if prev.Kind == PrevKnown {
        if p, ok := prev.Value.(string); ok && p == cur {
            return cur, nil                       // ① 相同 → 不发
        }
    }
    hadBefore := prev.Kind == PrevKnown && prev.Value != ""
    if prev.Kind == PrevUnknown { hadBefore = true }   // ② 不确定 → 当有
    switch {
    case cur == "" && !hadBefore:
        return cur, nil
    case cur == "":
        return cur, []Fragment{Notice("之前的上下文窗口指引已失效。")}   // ③ 移除通知
    case hadBefore:
        return cur, []Fragment{Notice("以下指引替换之前的全部指引。"), Text(cur)}
    default:
        return cur, []Fragment{Text(cur)}
    }
}
```

**写模板时的三个坑**（都在 codex 注释里踩过）：
- 用**三态**（Known/Unknown/Absent）而不是 `*T`，否则「历史里没有」和「历史里是空」分不开；
- 快照类型要用**可比较**的（字符串/小结构体），别塞整个对象——否则永远判不等，等于没差分；
- 通知文案要**固定**（替换/移除各一句），不要每轮动态生成，否则 diff 永远不收敛。

## 五、验收（可直接写成单测）

1. 同一 section 连续三轮快照不变 → 只有第一轮产生片段，后两轮 `len(out)==0`；
2. 快照从「有」变「空」→ 恰好一条移除通知；
3. `Previous.Kind == Unknown` 且当前非空 → 产生「替换」通知 + 内容；
4. 快照值改一个字符 → 恰好一条替换通知，且内容完整；
5. 五个 section 同时变更 → 每个 section 各出各的片段，顺序稳定（按 ID 字典序或注册序）。

## 六、配套：不是所有章节都需要「迁移历史」

`top_level_tools` 注释明确「**缺失状态视为全新目录，不迁移历史**」，而 `context_window_guidance` 提供了
`matches_legacy_fragment` 去识别旧片段。**这两种策略并存是刻意的**：
- 状态类的（指引、权限）→ 要能识别并替换旧片段，否则会叠加；
- 目录类的（工具目录）→ 重建即可，迁移成本高于收益。

施工时按章节性质二选一，不要一刀切。
