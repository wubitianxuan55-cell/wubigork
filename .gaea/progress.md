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

### 阶段 0 · 地基修补 —— ✅ 全部完成（12 提交）
- [x] **S0.1** AgentRunner 回合级 map 并发加固（turnMu）+ 回归测试 —— `96644b9`（临时 worktree 实证修复前必崩）
- [x] **S0.2** tool.Registry RWMutex + suspended 幽灵名修复 —— `a77ce5e`（全包 -race 绿）
- [x] **S0.3** gate 改 atomic.Pointer[gateWrapper]（撕裂换闸）—— `8b661d2`
- [x] **S0.4** retry_until 走统 gated 派发（堵绕过审批 shell）—— `87bbf20`
- [x] **S0.5** CI 后端 job 加 `go test -race`（独立 ubuntu job）—— `afefa11`
- [x] **S0.6/隔离岛** knowledge TF-IDF 索引缓存 `aadb7aa`；office 原子写 `5a82209`；secure 非 Win AES-GCM `fa1778f`；tasks 输出 LRU `2a3ca50`
- [x] **S0.7/前端** 聊天列表 memo+尾部窗口 `9a41090`；keepAlive 轮询门控+LoRA 修复 `d373fba`
- [ ] **S0.6 edit_file 工具层**（审计 P1 功能 bug：edit_file/multi_edit/edit_lines/move_file/grep 被 ~40 处引用但无实现，模型被指示用却收 "unknown tool"）——产品文件编辑脊柱，**未做，独立一刀**
- [ ] 遗留观察：持久化套件统一（desktop_session.go/archive.go 原子写，office 子代理排查）

### 阶段 1 · 双空间内核（后端 space 维度，S1.1-S1.5）
- [ ] S1.1 space 维度（session/tasks/事件日志字段 + DB 迁移 + 旧数据回填 work + 子代理空间继承 + 产物路径分区）
- [ ] S1.2 记忆空间隔离器（work/play 互不检索 + dream 空间化 + [MEM:] 限定 + 统一检索 scope）
- [ ] S1.3 模型 profile 按空间 + 工具空间标签装配
- [ ] S1.4 任务/资源按空间分账
- [ ] S1.5 空间策略（work 审阅制 / play 内容护栏）

### 阶段 2 · 双空间壳（前端两视图，S2.1-S2.3）

## 纪律（沿用）

- 每 Step 独立提交可回退；旧数据只读兼容；回退演练纳入验收；不做新板块、不堆功能；
  docs/ 过时文档一律归档到 `docs/archive/`（权威 = 长期规划 + AGENTS.md + 本文件）。
