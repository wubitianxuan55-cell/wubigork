# P1 Bug 修复设计

> 修复 wubigork v1.2.0 中 4 个确认 Bug，涉及 SSE 流结束检测、章节编号、前后端一致性、错误处理。

**目标:** 消除直接影响用户使用的 4 个 Bug，每个独立可测，一次 commit 交付。

**涉及文件:**
- `internal/ai/client.go` — Bug 1A (finish_reason), Bug 1D (marshal/read 错误)
- `internal/app/app.go` — Bug 1B (chapterNum 硬编码)
- `internal/outline/outline.go` — Bug 1C (OrderIndex 下限)
- `frontend/src/pages/ChapterPage.tsx` — Bug 1C (前端适配)

---

## Bug 1A: SSE 流从不检查 `finish_reason`，可能导致 goroutine 永久阻塞

**位置:** `internal/ai/client.go:214`

**根因:** 流式读取的 `scanner.Scan()` 循环仅在收到 `[DONE]` 行时退出。xAI/OpenAI 兼容 API 可能在最后一个 chunk 的 `choices[0].delta` 中携带 `finish_reason`（值为 `"stop"`），而不是发送额外的 `[DONE]` 行。此时 `scanner` 等待下一行但永远等不到，goroutine 阻塞。

**修复:** 在解析 `data:` 行后，检查 `choices[0].finish_reason`，若非空则视为流结束。

```go
// 在解析 choice 之后，检查 finish_reason
if len(choice.Choices) > 0 {
    if choice.Choices[0].FinishReason != "" {
        resultCh <- StreamResult{Done: true}
        return
    }
}
```

**涉及类型补充:** 需要在 `StreamChoiceDelta` 或相应的响应结构体中增加 `FinishReason string` 字段。

---

## Bug 1B: `GenerateChapter` goroutine 中 chapterNum 被硬编码为 1

**位置:** `internal/app/app.go:631`

**根因:** goroutine 闭包内 `chapterNum := 1` 从未从实际 outline node 更新。`ReadChapterSummary(chapterNum)` 始终读第 1 章的摘要。生成第 5 章时前端收到第 1 章的摘要。

**修复:** 在闭包外计算正确的 chapterNum（从 `projCtx.CurrentOutline.OrderIndex` 获取），再被闭包捕获。

```go
// 闭包前计算
chapterNum := projCtx.CurrentOutline.OrderIndex
if chapterNum < 1 {
    chapterNum = 1
}

// goroutine 闭包中直接使用 chapterNum（不再声明）
```

---

## Bug 1C: 章节编号 0 导致前端无法保存

**位置:** `internal/outline/outline.go` (后端), `frontend/src/pages/ChapterPage.tsx:96,127` (前端)

**根因:** 
- 后端 `CreateNode`/`UpdateNode` 允许 `OrderIndex = 0`
- 前端 `chapterNum = node.order_index || 0` → `0`
- `handleSave` 条件 `if (!chapterNum || !content) return` → 拒绝保存

**修复:** 
- **后端 (outline.go):** 在创建/更新节点时强制 `OrderIndex >= 1`
- **前端:** `handleSave` 条件放宽，允许 `>= 1`（已经是了，但要做防御）

```go
// outline.go — CreateNode / UpdateNode 中
if node.OrderIndex < 1 {
    node.OrderIndex = 1
}
```

```tsx
// ChapterPage.tsx — handleSave
if (chapterNum < 1 || !content) return;
```

---

## Bug 1D: `json.Marshal` / `io.ReadAll` 错误被静默吞掉

**位置:** `internal/ai/client.go:88, 100, 139, 164, 286`

**根因:** 使用 `_` 丢弃错误返回值：
- `body, _ := json.Marshal(req)` — 若序列化失败，body 为 nil，HTTP 请求体为空
- `respBody, _ := io.ReadAll(resp.Body)` — 若读取失败，后续解析得到截断或空数据

**修复:** 6 处全部改为显式错误处理：

```go
body, err := json.Marshal(req)
if err != nil {
    return "", fmt.Errorf("marshal request: %w", err)
}

respBody, err := io.ReadAll(resp.Body)
if err != nil {
    return "", fmt.Errorf("read response body: %w", err)
}
```

---

## 测试验证

每个修复的验证方式：

| Bug | 验证方法 |
|-----|---------|
| 1A | 生成短章节观察流式输出完整结束（前端控制台无挂起），或构造只发 `finish_reason` 的 mock |
| 1B | 生成非第 1 章（如第 3 章），检查控制台摘要是否正确 |
| 1C | 创建 OrderIndex=0 的节点，在前端编辑后确认可以保存 |
| 1D | 正常生成章节不报错（marshal/read 在正常路径不触发），代码审查通过 |

**构建验证:** `wails build` 后运行 exe，执行上述手动测试。
