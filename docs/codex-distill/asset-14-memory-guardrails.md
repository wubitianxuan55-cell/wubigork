# 资产 14 · 记忆四护栏施工级：脱敏正则、三道门、基线差分、产物校验

> 源码：`secrets/src/sanitizer.rs`（3 KB，全文已读）、`memories/write/src/{guard.rs,start.rs,workspace.rs,extensions/prune.rs}`。
> 对照：gaea `internal/gaea/memory/`（脱敏 0 处命中、基线 0 处命中、背压 0 处命中——三处都是空白）。

---

## 一、脱敏：四条正则，可以直接抄

```rust
static OPENAI_KEY_REGEX:  r"sk-[A-Za-z0-9]{20,}"
static AWS_ACCESS_KEY_ID: r"\bAKIA[0-9A-Z]{16}\b"
static BEARER_TOKEN:      r"(?i:\bBearer)[ \t]+[A-Za-z0-9._~+/-]{16,}=*"
static SECRET_ASSIGNMENT: r#"(?i)\b(api[_-]?key|token|secret|password)\b(\s*[:=]\s*)(["']?)[^\s"']{8,}"#
```

**替换顺序**（先具体的、后宽泛的，避免后者把前者吃掉）：

```
1) Bearer  → "Bearer [REDACTED_SECRET]"
2) sk-     → "[REDACTED_SECRET]"
3) AKIA    → "[REDACTED_SECRET]"
4) key=val → "$1$2$3[REDACTED_SECRET]"      // 保留键名、分隔符、引号，只换值
```

**第 4 条是关键**：保留 `api_key = ` 这段（含分隔符与引号），只把值替换掉。
这样记忆里仍能看出「这里原本有一个 api_key 字段」，但拿不到值——比整段抹掉更有信息量。

**Go 版**（`regexp` 直接支持，注意 Go 的 `\b` 与 RE2 语法差异）：

```go
var (
    reBearer = regexp.MustCompile(`(?i)\bBearer[ \t]+[A-Za-z0-9._~+/-]{16,}=*`)
    reOpenAI = regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`)
    reAWS    = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
    reAssign = regexp.MustCompile(`(?i)\b(api[_-]?key|token|secret|password)\b(\s*[:=]\s*)(["']?)[^\s"']{8,}`)
)

func Sanitize(s string) (string, int) {
    n := 0
    for _, re := range []*regexp.Regexp{reBearer, reOpenAI, reAWS} {
        s = re.ReplaceAllStringFunc(s, func(string) string { n++; return "[REDACTED_SECRET]" })
    }
    s = reAssign.ReplaceAllStringFunc(s, func(m string) string {
        n++
        sub := reAssign.FindStringSubmatch(m)
        return sub[1] + sub[2] + sub[3] + "[REDACTED_SECRET]"
    })
    return s, n
}
```

**落点**：`memory/distill.go` 与 `memory/remember.go` 的统一写入入口（如有多处入口，抽一个 `sanitize()` 强制走它）。

## 二、三道前置门

```rust
// start.rs
// · 临时会话（ephemeral）、子智能体会话 → 不跑
// · feature flag 关 → 不跑
// · guard.rs::rate_limits_ok() == false → 不跑（限流时不烧配额）
```

`rate_limits_ok` 的实现细节值得注意：**查询失败时返回 `true`**（`unwrap_or(true)`）——
即「拿不到限流状态」时按「可以跑」处理，不因为限流接口抽风就永久停摆。这是个务实的默认。

**gaea 落点**：`memory/morning.go` 与 `memory/distill.go` 的调度入口，加一个 `gate()`：
会话类型判断 → 配置开关 → 限流状态（拿不到就当通过）→ 三条都过才跑，跳过时记事件
`skipped: <reason>`。

## 三、基线差分（这块 gaea 完全没有）

```rust
prepare_memory_workspace(root)            // 准备基线
memory_workspace_diff(root) -> GitBaselineDiff   // 与基线比
write_workspace_diff(root, &diff)         // 差分落盘（可审）
validate_consolidation_artifacts(root)    // 校验产物
remove_memory_symlinks(root)              // 清理软链（防越界写入）
reset_memory_workspace_baseline(root)     // 重置基线
```

差分渲染是**有界的**：`append_bounded_diff(rendered, diff)` 配合 `previous_char_boundary(value, max_bytes)`
按**字符边界**截断，避免把中文切成半个字。

**产物校验示例**（v2 摘要的校验函数，可直接借鉴形态）：

```rust
pub fn is_valid_v2_summary(summary: &str) -> bool {
    summary.lines().next() == Some("v1")          // 版本行
        && summary.len() < 10_000                 // 长度上限
        && ["## User Profile", "## User preferences", "## General Tips", "## What's in Memory"]
            .iter().all(|h| summary.lines().any(|l| l.trim() == *h))   // 四个必备小节
}
```

**三个可抄的点**：① 版本行必须在第一行；② 硬长度上限；③ **必备小节集合**——
这比「非空即合法」强得多，能挡住模型只输出一句话就交差。

## 四、保留期清理

```rust
RETENTION_DAYS = 14   // 扩展资源保留 14 天，超期由 prune_old_extension_resources 清掉
```

gaea 没有「扩展资源」概念，但**会话附件/临时产物**有同样的膨胀问题，可以借这条：
`<memory_root>/extensions/<日期>/…`，保留 14 天，超期清理并记事件。

## 五、施工建议（三步，全部在 `memory` 包内，风险低）

**步一（半天）**：`memory/sanitize.go` + 接入写入入口
- 四条正则 + 计数；写入路径（distill / remember / 自动沉淀）统一过一遍；
- 计数进事件（`redacted: 3 (openai=2, assignment=1)`），**不记录原文**。

**步二（半天）**：`memory/gate.go` 背压门
- 会话类型 → 配置开关 → 限流状态（拿不到当通过）；
- 拒绝时记 `skipped: rate_limited` / `skipped: subagent` 等。

**步三（1–1.5 天）**：`memory/baseline.go` 基线差分
- 整理前快照（id → 内容哈希）；
- 整理后算 `added / modified / removed` 三类计数；
- **阈值保护**：`removed > 10%` 或 `modified > 30%` → **不应用**，落一条「待确认」事件；
- 差分与结果一并入库，供 UI 展示「这次整理改了什么」。

## 六、验收（可断言）

1. 12 条真实形态假密钥（sk- / AKIA / Bearer / `api_key=…` / 私钥块）写入后，按原文 grep 全表**零命中**；
2. 含 `key`/`token` 字样但非密钥的普通文本（如「小说里的 token 概念」）**不被误伤**（准备 5 条）；
3. 限流中触发整理 → 不执行且事件为 `skipped: rate_limited`；限流解除后下一次正常执行；
4. 一次整理产生差分（新增 N / 修改 M / 删除 K），超阈值时结果被拦下并标「待确认」，**原记忆未被改动**；
5. 差分渲染超过上限时按字符边界截断，中文不出现半个字（断言 UTF-8 合法）。

## 七、不建议照搬

- **git 作为基线实现**（`GitBaselineDiff`）：gaea 用 SQLite，直接存「id → 内容哈希」的表即可，不必起 git；
- **v1/v2 版本化产物校验**：gaea 没有历史包袱，只要「四个必备小节 + 长度上限 + 首行版本」三条即可；
- **`remove_memory_symlinks`**：gaea 记忆目录没有软链，除非将来允许用户软链挂载，否则跳过。
