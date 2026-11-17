# 资产 12 · 上下文碎片契约施工级：四个组成 + 归属七类

> 源码：`context-fragments/src/{fragment.rs,annotated_content.rs}`、
> `protocol/src/models/item_metadata.rs`（归属契约，含向后兼容反序列化）。
> 这份契约是已完成的第 1（章节化）、第 3（Guardian）、第 6（压缩重注入）三项的**共同地基**。

---

## 一、四个组成（缺一个就不成立）

```rust
pub trait ContextualUserFragment {
    fn role(&self) -> &'static str;                       // ① 消息角色
    fn content_kind(&self) -> ContentItemKind;            // ② 分类（稳定字符串）
    fn markers(&self) -> (&'static str, &'static str);    // ③ 标记对
    fn body(&self) -> ContentItem;                        //    正文
    fn content_metadata(&self) -> ContentItemMetadata;    // ④ 归属（默认 harness）
    fn type_markers() -> (&'static str, &'static str);    //    类型级标记（用于识别）
    fn matches_text(text: &str) -> bool;                  //    用 type_markers 判定
}
```

**① role 与 ④ 归属是两件事**。契约文件头一句话点题：
*"Per-part context sources, independent of message role, visibility, and trust."*
——归属与角色、可见性、信任都独立。这是整份契约最重要的设计。

## 二、归属七类（`ContentItemProvenance`）

```rust
pub enum ContentItemProvenance {
    Unknown,                                   // 历史数据或来源不明
    DeveloperInstructions { from_additional_requirements: bool },
    Skills,                                    // 技能目录/用法/选中技能内容
    AgentsMd,                                  // AGENTS.md 载入的指令
    Harness { hook: bool },                    // 框架托管上下文（hook 输出单列）
    Tool { namespace: Option<ContentItemNamespace> },  // 工具结果或工具上下文
    User,                                      // 真实用户输入
}
```

三条注释里写死的纪律，值得原样搬：

1. **"Tool-related context remains Tool even when the harness writes it"** ——
   工具相关上下文**即使由框架代写**，归属仍是 `Tool`，不能因为「是我们自己写的」就洗成 `Harness`。
2. **`User` 专指「普通用户输入」，不是「带 user 角色的注入片段」** ——
   这条把「用户真的说了」和「我们塞了一段 user 角色的指令」分开，是提示注入防御的判定基础。
3. **`Harness{hook}` 把 hook 输出与内建上下文分开** —— hook 是用户配置的脚本，可信度与内建不同。

`ContentItemNamespace` 是枚举 + `Other(String)` 开放集：内建命名空间（`clock` / `functions` /
`tool_search` / `collaboration` / `multi_agent_v1`）走强类型，MCP 与自定义走 `Other` 保留原名。

## 三、渲染规则

```rust
fn render(&self) -> ContentItem {
    let (start, end) = self.markers();
    let body = self.body();
    if start.is_empty() && end.is_empty() { return body; }   // 无标记 → 直出
    match body {
        ContentItem::InputText { text } => input_text(format!("{start}{text}{end}")),
        content => content,                                   // 非文本内容不包标记
    }
}
```

**只对文本包标记**——图片/音频这类结构化内容包标记反而会破坏解析。
`type_markers()` 与 `markers()` 分离：`markers()` 是本实例的（可能带实例 id），
`type_markers()` 是类型级的常量，`matches_text` 用它判定「这段是不是我这类」。

## 四、向后兼容（老记录怎么映射）

```rust
// 反序列化时：缺 provenance 字段的老记录按下面规则还原
(Some(namespace), _)      => Tool { namespace: Some(namespace) }
(None, Some(true))        => Harness { hook: false }     // 老字段 harness_injected
(None, Some(false)|None)  => Unknown
// 另有 serde alias: "user_configured" → Unknown
```

注释直说：老的 `user_configured` 与裸 `harness_injected:false` **都不足以精确判定来源**，
所以一律归 `Unknown`，不猜。

---

## 五、gaea 施工方案（Go 契约）

```go
// Provenance 是内容出处，独立于消息角色。
type Provenance int

const (
    ProvUnknown Provenance = iota
    ProvDeveloperInstructions
    ProvSkills
    ProvAgentsMD
    ProvHarness          // 框架托管（内建上下文）
    ProvHarnessHook      // hook 输出，单独一档
    ProvTool             // 工具结果/工具上下文（即使框架代写）
    ProvUser             // 真实用户输入
)

// Fragment 是所有注入上下文的统一形状。
type Fragment struct {
    Role    string            // system / user / developer / tool
    Kind    string            // "<feature>.<name>"，稳定
    Start   string            // 标记对；都为空则直出不包
    End     string
    Body    string
    Prov    Provenance
    NS      string            // 工具命名空间（ProvTool 时）
}

func (f Fragment) Render() string            // 文本包标记
func MatchesText(kind, text string) bool     // 类型级标记判定
```

### 三条实现纪律（照抄 codex，不要"优化"）

1. **归属由生产者声明**，绝不从渲染文本反推（`AttributedFragment` 就是为这个存在的）；
2. **工具上下文即使由框架写入，`Prov` 仍为 `ProvTool`**；
3. **只有文本才包标记**，结构化内容直出。

### 接线点（与已完成的三项对接）

| 对接项 | 用契约的哪部分 | 具体做法 |
|---|---|---|
| 第 1 项 章节化 | `Kind` + `Start/End` | 每个 section 产出的片段带自己的标记，压缩/重注入按 `Kind` 精确替换 |
| 第 3 项 Guardian | `Prov` | 只有 `ProvUser` / `ProvDeveloperInstructions` / `ProvAgentsMD` 能建立授权；`ProvTool` / `ProvHarnessHook` 一律不可信 |
| 第 6 项 压缩 | `Kind` + `Prov` | 新窗口重建时，按 `Kind` 决定「必须重注入」还是「可丢弃」；`ProvUser` 的消息优先保留 |

## 六、验收（可断言）

1. 注入片段被渲染后，用 `MatchesText(kind, rendered)` 能稳定识别（同一 kind 连续三轮都命中）；
2. 工具输出片段即使由框架代写，`Prov == ProvTool`（写一条单测钉死）；
3. 带 user 角色但 `Prov == ProvHarness` 的片段**不能**作为授权证据（与 Guardian 验收联动）；
4. 图片类内容不包标记，文本类必包；
5. 老会话（无归属字段）载入后 `Prov == ProvUnknown`，不误判成 `ProvUser`。

## 七、不建议照搬

- **`ContentItemNamespace` 的强类型枚举**：gaea 的工具命名空间还没稳定，用字符串 + 白名单校验即可，
  过早枚举每次加命名空间都要改类型；
- **`AnnotatedContent::from_parts` 的三元组合**（内容 + 分类 + 归属）：go 里用结构体字段更直接；
- **完整的反序列化兼容层**：gaea 没有历史包袱，只留「缺字段 → Unknown」一句即可。

## 八、一句话

这份契约的价值不在「能包标记」，而在**把「谁写的这段」变成结构化字段**——
有了它，Guardian 才有判定依据、压缩才能精确替换、章节化才不会重复注入。
它是第 1/3/6 三项的接口总线，应该最先落（半天的活）。
