# 记忆「自动做梦」写入审批决策（T6-8.1）

> 定稿：2026-08-14（阶段 6 第八刀 v2.31.0）
> **修订：2026-09-21（v4.377 建议制）——见 §5**
> 关联代码：internal/app/gaea_dream.go（决策注释）、internal/app/gaea_dream_pending.go（待确认队列）、
> internal/gaea/control/controller_memory.go（SaveDreamFacts 审计 + 清污）、
> internal/gaea/control/controller_approval.go（hardAskTools）

## 1. 问题

代码审查（G8）发现：自动做梦（dream）写入事实直接调用 SaveDreamFacts，
不经过 hardAskTools 逐条审批——与 remember / forget / knowledge_add /
cost_save 等持久化写入的审批纪律不一致，存在「绕过审批」的表面风险。

## 2. 旧决策（v4.376 及以前）：默认放行，全程审计（不纳入 hardAskTools）

**dream 写入不纳入 hardAskTools 逐条审批，以审计日志作为补偿。** 理由：

1. **后台流程无法等待人工确认**。「自动做梦」在会话轮次成功后由后台
   goroutine 异步触发（90s 超时、单飞），此时用户可能已离开会话；若改走
   审批，整理任务会阻塞等待超时或被丢弃，自动记忆整理能力失效。
2. **显式路径本身即用户触发**。记忆建议「接受」按钮、/dream extract 命令
   都是用户主动操作，再弹审批属于重复打扰。
3. **写入内容低风险**。dream 只提炼「值得长期记住」的稳定事实（Kimi 二问
   过滤），写入按 name 去重、最多 5 条；不涉及删除/覆盖用户文件。
4. **删除/修改仍受既有纪律约束**。ForgetMemory、UpdateFact、remember 工具
   等原审批路径不变；用户随时可在记忆面板删除任何自动沉淀的事实。

## 3. 补偿机制：写入审计日志

每次 SaveDreamFacts 实际写入（saved > 0）后，追加一行审计到
`<userDir>/dream-audit.jsonl`（JSONL，与 gaea.db 同目录，追加式、损坏行跳过）：

```json
{"ts":"2026-08-14T12:00:00+08:00","source":"auto_dream","saved":2,"names":["user-unit","project-budget"],"space":"work"}
```

- `source`：`auto_dream`（轮次后自动整理）| `explicit`（用户接受记忆建议 / /dream extract）
  | `dream_purge`（v4.377 清污：批量删除 auto_dream 写入的事实，Names=实际删除名单）。
- `space`（S1.2 记忆空间隔离器新增）：本批事实落库空间（写侧 Normalize 兜底
  后的生效值 `work`/`play`）；旧行无该字段（读端零值 ""），JSONL 追加列向后兼容。
- 审计写入尽力而为：失败仅记 slog.Warn，不阻断记忆写入主流程。
- 读取入口：`control.DreamAuditEntries(userDir, max)`（倒序最近 max 条）。

## 4. 验收口径（T6-8.1）

- 决策文档化：本文档 + gaea_dream.go / SaveDreamFacts 代码注释引用。
- 审计日志测试 ≥2：自动/显式两路径各断言 1 条审计行（source/saved/names）。
- 全量 go test ok + vet 干净。

## 5. 修订（v4.377，2026-09-21）：默认建议制——未经用户确认不写记忆

**实测推翻旧决策的前提假设 3**：dream 提炼的「低风险」不成立——每轮对话
结束后无条件提炼，把大量错误/噪音当事实直写长期记忆与项目记忆文档
（AGENTS.md 的 QuickAdd 追加更是直改用户文档），用户无从拦截，数据被污染
（用户实测反馈）。异步无法弹审批卡的结论仍成立，但解法不是放行直写，
而是**把确认从「写入时」移到「接受时」**：

1. **默认 suggest（建议制）**：提炼结果进待确认队列
   （`<userDir>/dream-pending.json`，FIFO 上限 30 条），记忆面板「建议」
   标签页展示；用户逐条「接受」才入库（走 SaveDreamFacts，
   source=explicit 落审计）、「忽略」即丢弃。不询问永不写入。
2. **auto（旧行为）**：显式配置 `[dream] mode = "auto"` 才恢复直写
   （审计留痕不变）——回退开关，不再是默认。
3. **off**：`[dream] mode = "off"` 完全关闭轮次后自动整理（手动
   /dream extract 仍可用）。
4. **记忆总开关收口**：`[memory] enabled = false` 时任何模式都不整理
   （旧实现只关注入、管不住 dream 的缺口一并修复）。
5. **存量清污**：旧直写已污染的数据按审计日志批量清理——
   `GaeaDreamPurgePreview()` 返回 auto_dream 审计记录过且当前仍存的
   事实名（用户显式接受的 explicit 与手工沉淀不在内），
   `GaeaDreamPurge(names)` 仅允许删该交集（绑定面不可删任意记忆），
   记忆面板「建议」页提供两步确认入口。
6. 配套改口：默认 system prompt 记忆段改为「只在用户明确要求记住时调用
   remember」；cost_save/cost_compose/cost_search 工具描述与返回文案
   去掉「测完就沉淀」的劝写指示，改为「用户确认后才保存」。

验收：suggest/auto/off 三态 + 队列上限/出队/空间过滤 + accept/dismiss +
清污交集（含交集外拒绝）测试全绿；全量 ci 绿。

