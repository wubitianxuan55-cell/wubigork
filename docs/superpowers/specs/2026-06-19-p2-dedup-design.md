# P2 重复代码消除 设计

> 消除 internal 包中 12 处重复函数，统一至 `internal/util`；删除死代码 `internal/xai/`。

**目标:** DRY — 5份 `extractJSON`、3份 `truncate`、2份 `max`、2份 `min` 各归其位。删除无人引用的 `internal/xai/` 包。

**涉及文件:**
- 新建: `internal/util/util.go`
- 修改: `analysis/analysis.go`, `chapter/chapter.go`, `character/character.go`, `outline/outline.go`, `worldview/worldview.go`, `app/app.go`, `ai/context.go`
- 删除: `internal/xai/`（整个目录）

---

## 新建 `internal/util/util.go`

```go
package util

// ExtractJSON 从 AI 回复中提取第一个完整 JSON 对象
func ExtractJSON(s string) string {
    start, end := -1, -1
    for i, ch := range s {
        if ch == '{' && start == -1 {
            start = i
        }
        if ch == '}' {
            end = i
        }
    }
    if start >= 0 && end > start {
        return s[start : end+1]
    }
    return s
}

// Truncate 按 rune 截断字符串
func Truncate(s string, maxLen int) string {
    runes := []rune(s)
    if len(runes) <= maxLen {
        return s
    }
    return string(runes[:maxLen]) + "..."
}

// Max 返回两个 int 中较大值
func Max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

// Min 返回两个 int 中较小值
func Min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

---

## 修改清单

### 1. `internal/analysis/analysis.go`
- 删除本地 `extractJSON`（line 191）和 `truncate`（line 183）
- 添加 `import "github.com/wubigork/wubigork/internal/util"`
- `extractJSON(...)` → `util.ExtractJSON(...)`
- `truncate(...)` → `util.Truncate(...)`

### 2. `internal/chapter/chapter.go`
- 删除本地 `extractJSON`（line 292）、`truncate`（line 284）、`max`（line 279）
- 添加 util import
- 三处调用替换为 `util.*`

### 3. `internal/character/character.go`
- 删除本地 `extractJSON`（line 430）、`min`（line 447）
- 添加 util import
- 两处调用替换

### 4. `internal/outline/outline.go`
- 删除本地 `extractJSON`（line 385）
- 添加 util import
- 调用替换

### 5. `internal/worldview/worldview.go`
- 删除本地 `extractJSON`（line 442）、`min`（line 254）
- 添加 util import
- 两处调用替换

### 6. `internal/app/app.go`
- 删除本地 `max`（line 823）
- 添加 util import
- `max(...)` → `util.Max(...)`

### 7. `internal/ai/context.go`
- 删除本地 `truncate`（line 182）
- 添加 util import
- `truncate(...)` → `util.Truncate(...)`

---

## 删除 `internal/xai/`

删除整个目录 `internal/xai/`（含 `client.go` 和 `types.go`）。确认无人引用（仅 `pkg/novel/agent.go` 引用，而 `pkg/novel` 本身也是死代码，保留不动）。

---

## 验证

- `wails build` 全量编译通过
- 所有 `extractJSON` / `truncate` / `max` / `min` 行为不变（函数体一字不差）
