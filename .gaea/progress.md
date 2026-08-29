# 任务进度

> 最后更新: 2026（长期规划定稿，v3.7.0 基线）

## 当前状态

- **最新发布：v3.7.0（2026-08-29）**——办公蒸馏 codex 收官（C2 记忆引用可追溯 / C4 审批决策族 /
  C9 任务输出事件化 / C5 上下文占用状态行 / C6 项目说明预算 / C3 做梦 no-op）。
  验证：Go 114/114 + vet；vitest 681/681；eslint 0/0；tsc 0；绑定面 499 漂移 PASS。

## 长期规划（权威文档）

- **`docs/gaea-nextgen-roadmap-2026.md`** —— gaea 长期规划（2026+）：
  8 板块竞品调研 + WorkBuddy×灵犀 对标 + **版本重定义（双空间：工位/乐园）** +
  四层落地（后端/前端/UX/UI）+ 执行计划（阶段 0 地基 → 阶段 1 双空间内核 →
  阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板（重要）**：**工作与娱乐分开、互不干扰**——工位（办公/造价/编程/资料记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区、模型策略各配、上下文永不跨界；
  跨空间仅用户显式发起。~~"陪伴×办公融合"~~ 已删除。
- 本轮调研另一项关键纠正：**灵犀 = 金山 WPS 独立 AI 办公 Agent**（非阿里/通义）；
  **WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台**（非 Kimi 系，Kimi Work 是月之暗面的）。

## 下一步执行（按执行计划 §14）

- [ ] **S0.1** AgentRunner 回合级 map 并发加固（batch_executor/execute_one）+ 回归测试
- [ ] **S0.2** tool.Registry RWMutex + suspended 幽灵名修复
- [ ] **S0.3** SetGate/SetPermLevel → atomic.Pointer
- [ ] **S0.4** retry_until 走统一 gated 派发
- [ ] **S0.5** CI 后端 job 加 `go test -race`
- [ ] **S0.6** edit_file 工具层实现（最小编辑工具集）
- [ ] **S0.7** 前端：keepAlive 轮询门控 + 聊天虚拟化 + hex token 化 + i18n 决策
- [ ] S1.1–S1.4 双空间内核（后端 space 维度/记忆隔离/模型 profile/任务分账）
- [ ] S2.1–S2.3 双空间壳（前端两视图/页面迁入/bridge 分面）

## 纪律（沿用）

- 每 Step 独立提交可回退；旧数据只读兼容；回退演练纳入验收；不做新板块、不堆功能；
  docs/ 过时文档一律归档到 `docs/archive/`（权威 = 长期规划 + AGENTS.md + 本文件）。
