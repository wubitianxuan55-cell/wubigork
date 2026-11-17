# 第二批蒸馏执行计划（v2 · 对齐第一批落地现状）

> 第一批（T1–T5）正在落地中。本计划**先核对现状，再排第二批**，
> 避免与在改文件撞车、避免重复劳动。
> 依据：`git status` / `git diff --stat` 实测（2026-11-17 当前工作树）。

---

## 0. 第一批现状核对（实测）

| 项 | 状态 | 证据（工作树） |
|---|---|---|
| T1 三档行匹配 | **已落地** | 新增 `tool/seek.go` + `seek_test.go`；`tool/builtin/editfile.go` +152 行、`multiedit.go` +15 行；新增 `editfile_seek_test.go` |
| T2 执行环境注入 | **已落地** | 新增 `proc/env.go` + `env_test.go` |
| T3 审批拒绝理由 / 超时 | **进行中** | 新增 `control/controller_denial_test.go`；`controller_approval.go` 有改动 |
| T4 `Never` 语义 + Windows 危险命令 | **已落地** | 新增 `permission/approval_mode.go` + 测试、`permission/dangerous_windows.go`；`permission.go` +40/-7 |
| T5 工具 schema 字节预算 | **已开工** | 新增 `tool/budget.go`（尚无测试文件） |

**合计改动**：13 个文件、+420/−105 行。

**结论**：第一批 5 项里 3 项已落、1 项在改、1 项开了一半。第二批必须按此重排。

---

## 1. 冲突面（第二批哪些项会撞车）

第一批已占用这些包/文件：`tool/{seek,budget}.go`、`tool/builtin/{editfile,multiedit,bash}.go`、
`proc/env.go`、`permission/{permission,approval_mode,dangerous_windows}.go`、
`control/{controller,controller_approval}.go`、`control/controller_denial_test.go`。

| 第二项 | 会碰的包 | 冲突风险 |
|---|---|---|
| T10 审批七态 | `permission` + `controller_approval.go` | **高**（第一批 T3/T4 正在改同两个文件） |
| T8 工具面预算 | `tool/budget.go`（已存在）+ `tool/catalog.go`（新） | **中**（T5 已建 budget.go，需在其上扩展而非新建） |
| T13 执行语义 | `proc/env.go`（已存在） | **低**（新文件 `session.go`/`headtail.go`，不动 env.go） |
| T6 碎片契约 | `prompt` / `contextview` | **无** |
| T7 章节化 | `contextview` / `prompt` | **无** |
| T9 压缩 | `agent/compact.go`、`hook` | **无** |
| T11 Guardian | 新增 `internal/gaea/guardian` | **无**（但依赖 T10 完成） |
| T12 记忆护栏 | `memory` | **无** |
| T14 技能 RRF | `skill` | **无** |
| T15 角色数据化 | `pantheon` | **无** |

---

## 2. 第二批执行顺序（三波，按冲突与依赖排）

```
第一波（零冲突，可立刻并行）           第二波（依赖第一波 T6）        第三波（依赖 T10）
  T6 碎片契约 ──┐                        T7 章节化                      T11 Guardian
  T12 记忆护栏  ├─ 三者互不相干            T8 工具面预算（并入 T5）        T15 角色数据化
  T14 技能 RRF  │                        T9 压缩三形态
  T13 执行语义 ─┘                        T13 续（双缓冲/双闸门）
                                         
  等第一批合入后：T10 审批七态（T3/T4 已铺好路）→ 再上 T11
```

**关键判断**：`T10` 虽然重要，但**必须排在第一批合入之后**——现在动 `permission.go` / `controller_approval.go`
必然与 T3/T4 的在改内容冲突。而 T6（碎片契约）零冲突且是 T7/T9/T11 的地基，应当**立刻开工**。

---

## 3. 施工卡（第二批，精简版）

### 第一波

**T6 · 上下文碎片契约**（0.5 天，零冲突）
- 落点：新增 `internal/gaea/context/fragment.go`（`Role/Kind/Markers/Body/Provenance` + `Render` + `MatchesText`）；
  注入入口在 `prompt`/`contextview` 侧**逐步接入**，先只加字段不改行为。
- 验收：① 同 Kind 片段连续三轮可被 `MatchesText` 命中；② 工具输出片段即使框架代写 `Prov=Tool`；
  ③ 带 user 角色但 `Prov=Harness` 的片段不能作为授权证据。
- 开关：字段缺省 `Unknown`，无行为变化（无需开关）。

**T12 · 记忆四护栏**（2 天，零冲突）
- 落点：`memory/sanitize.go`（四条正则）、`memory/gate.go`（背压）、`memory/baseline.go`（差分）。
- 验收：① 12 条假密钥写入后按原文 grep 全表零命中；② 5 条含 `key`/`token` 字样的普通文本不误伤；
  ③ 限流中整理被跳过且事件 `skipped: rate_limited`；④ 差分超阈值（删除>10% / 修改>30%）被拦下且原记忆未改。
- 开关：`memory.sanitize.enabled`（建议常开）、`memory.baseline.enabled`（默认关）。

**T14 · 技能 RRF**（1–1.5 天，零冲突）
- 落点：新增 `skill/{lexical,ngram,fuse,select}.go`。
- 验收：① 同查询 100 次结果逐元素相同；② 覆盖 3 个词的文档排在覆盖 1 词之前；③ 8 KiB 查询不报错且 `query_truncated=true`。
- 开关：`skill.rrf.enabled` 默认关。

**T13 · 执行语义（续）**（2 天，低冲突，复用已落的 `proc/env.go`）
- 落点：新增 `proc/session.go`（`Session{Drain,Transcript}`）、`proc/headtail.go`；`batch_executor` 接双闸门。
- 验收：① 连续三次 `Drain` 不重叠且拼接等于 `Transcript()` 前缀；② 只有一个省略标记且字节数正确；
  ③ `yieldMS=100` 在 ≈100ms 返回且进程仍活；④ 退出后 100ms 内拿到收尾。
- 开关：`proc.session.enabled` 默认关。

### 第二波（T6 之后）

**T7 · 章节化 + 差分**（1–1.5 天）
- 落点：`contextview` 加章节表与 `RenderDiff`；`prompt` 按章节装配。
- 验收：① 三轮无变化 → 后两轮零片段；② 有变空 → 一条移除通知；③ `Unknown` 且非空 → 替换通知。
- 开关：`context.sections.diff` 默认关。

**T8 · 工具面预算（并入 T5）**（1.5 天）
- 现状：`tool/budget.go` 已存在（T5 开工）→ **在其上加目录差分与检索，不新建第二套预算**。
- 落点：`tool/catalog.go`（name→hash 差分）、`tool/search_index.go`（BM25，limit 8）。
- 验收：① 三轮工具集不变 → 工具部分上下文逐字节相同；② 40 工具首轮只发 ≤12 个，`tool_search` 后目标 schema 出现并可调用。
- 开关：`tools.catalog.diff.enabled`、`tools.search.enabled` 默认关。

**T9 · 压缩三形态**（1–1.5 天）
- 落点：`agent/compact.go` 加 `newWindowCompact(reason)`；`hook` 加 `PostCompact`。
- 验收：① 无摘要占位但有 `ContextCompaction` 起止事件；② `input_goal_ids` 前后一致；
  ③ `window_number` 单调递增；④ 切小窗口模型时先压缩再请求。
- 开关：`compact.new_window.enabled` 默认关。

### 第三波（第一批合入后）

**T10 · 审批七态**（2 天，承接 T3/T4）
- 前提：第一批 T3（拒绝理由）与 T4（`Never`）已合入，避免同文件冲突。
- 落点：`permission/argv_rule.go`（argv 前缀规则，替代对 bash 的 glob）、`controller_approval.go`（七态 + 快照 + 追认）。
- 验收：① `git status` 放行但 `git status; rm -rf /tmp/x` 仍要审批；② `Session` 放行 `/new` 后失效、`Persistent` 仍生效；
  ③ 超时 → `TimedOut` 且不产生 deny 账本；④ 追认后同一动作执行一次并记账。
- 开关：`permission.argv_rules.enabled` 默认关。

**T11 · Guardian**（4–5 天，依赖 T10 + T6）
- 落点：新增 `internal/gaea/guardian`（contract/budget/decision）+ 接 `internal/ai`。
- 验收：① 6 条样例文本解析符合预期（含散文包裹/缺字段/空/非 JSON）；② 超时不产生拒绝账本；
  ③ `Prov` 不是 User/DeveloperInstructions/AgentsMd 的片段不能支撑 allow。
- 开关：`guardian.enabled` 默认关。

**T15 · 角色数据化**（1.5–2 天，可选）
- 落点：`pantheon` 加载层 + `agents/<role>.md`。
- 验收：新建角色文件即出现在 UI；删除不残留；`description` 进 spawn 引导；角色级配置覆盖只影响该席位。

---

## 4. 去重与合并（必须遵守）

| 第一批 | 第二批对应 | 处理 |
|---|---|---|
| T1 三档行匹配 | — | 已完成，不再重复 |
| T2 环境注入 | T13 | **env 部分已做**，T13 只做双缓冲 + 双闸门，不要重写 env |
| T3 拒绝理由 | T10 | T10 在其上补七态/超时/追认，**不要回退 T3 的改动** |
| T4 `Never` + 危险命令 | T10 | T10 的 argv 规则与 T4 的 `approval_mode.go` 并存，`NormalizeAsk` 保持不变 |
| T5 schema 预算 | T8 | **`tool/budget.go` 是同一个文件**，T8 只加目录差分与检索 |

**硬规则**：任何第二次触碰同一文件的任务，开工前先 `git log -1 --stat` 看清上一批怎么改的，
并在实现上**叠加而非替换**。

---

## 5. 门禁与回报

```powershell
go build ./...
go vet ./internal/gaea/...
go test ./internal/gaea/{context,prompt,memory,skill,proc,tool,agent,control,permission,guardian,pantheon}/...
```

回报格式（每项）：
```
T<n> <任务名>
基线：<包> N 例 / 既有失败 M 例
改动：<文件:行> 一句话
门禁：build / vet / test 结果
开关：默认 <关/开>，关闭时行为等价：是/否
冲突：是否触碰第一批已改文件（是则说明如何叠加）
```

---

## 6. 风险

1. **同文件冲突**（最高风险）：`permission.go` / `controller_approval.go` / `tool/budget.go` 三个文件会被两批同时改。
   对策：T10 排到第三波；T8 明确「在 budget.go 上叠加」。
2. **开关蔓延**：本批新开关共 8 个，建议统一前缀（`tools.*` / `context.*` / `compact.*` / `proc.*` / `skill.*` / `memory.*` / `permission.*` / `guardian.*`），
   并在设置页给一个「实验特性」总开关，避免散落。
3. **验收不可执行**：所有验收都写成了断言式（可 grep / 可计数），执行时若发现某条无法自动断言，应在回报里说明而不是放宽标准。
