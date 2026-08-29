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
- **用户拍板（2026 追加）**：**编程板块与办公板块不得混合**——保持现状 DSH 独立窗口
  （iframe），不并入工位、不共享工具面（防工具膨胀）；不做原生编程工作台/MCP 板块总线
  喂办公工具给编码 agent（长期规划 §4/§10.3/§13.1 已修订）。
- 本轮调研另一项关键纠正：**灵犀 = 金山 WPS 独立 AI 办公 Agent**（非阿里/通义）；
  **WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台**（非 Kimi 系，Kimi Work 是月之暗面的）。

## 下一步执行（按执行计划 §14）

### 阶段 0 · 地基修补 —— ✅ 全部完成（12 提交）
- [x] **S0.1** AgentRunner 回合级 map 并发加固（turnMu）+ 回归测试 —— `96644b9`（临时 worktree 实证修复前必崩）
- [x] **S0.2** tool.Registry RWMutex + suspended 幽灵名修复 —— `a77ce5e`（全包 -race 绿）
- [x] **S0.3** gate 改 atomic.Pointer[gateWrapper]（撕裂换闸）—— `8b661d2`
- [x] **S0.4** retry_until 走统 gated 派发（堵绕过审批 shell）—— `87bbf20`
- [x] **S0.5** CI 后端 job 加 `go test -race`（独立 ubuntu job）—— `afefa11`
- [x] **S0.6 edit_file 工具层**（grep/edit_file/multi_edit/edit_lines/move_file 五工具 + 名单全对齐 + 双路径失效特判）—— `7d560e6`
- [x] **S0.7/隔离岛** knowledge 索引缓存 `aadb7aa`；office 原子写 `5a82209`；secure 非 Win AES-GCM `fa1778f`；tasks LRU `2a3ca50`
- [x] **前端** 聊天 memo+尾部窗口 `9a41090`；keepAlive 轮询门控 `d373fba`
- [ ] 遗留：gate_test.go stubGate 计数器既有竞态（测试基建小修）；持久化套件统一（desktop_session/archive 原子写）

### 阶段 1 · 双空间内核（后端 space 维度）—— ✅ 全部完成（S1.1-S1.5）
- [x] **S1.1 空间维度落地**（设计文档 `docs/gaea-space-dimension-design.md`）：
  - [x] S1 列落库（SchemaV14）`5c24d16`｜S2 会话空间（目录分区+日志/checkpoint space+开关）`0722aeb`｜S3 子代理继承 `76f565e`｜S4 产物分区+绑定面（499→502）`5c9fc4e`
- [x] **S1.2 记忆空间隔离器**（设计 `docs/gaea-memory-isolation-design.md`）：后端 A+B `819d7ff`+`f0187a2`（写侧盖章/读端隔离/UnifiedSearch scope）；前端 C `53d621d`
- [x] **S1.3 模型 profile + 工具标签**（装配族 `f8612d0`）：`[space_profiles]` 段 + 工具装配期过滤（work 33/play 1/shared 13）+ MCP spec 层滤
- [x] **S1.4 任务分账**（`5ccaa7a`）：per-space 并发/优先级 pickNext + HasActiveInSpace + cron 显式 work + GaeaTaskList 变参
- [x] **S1.5 空间策略**：S1.5-A 权限按空间+hardAsk 参数化+persist_allow 分段（装配族 `f8612d0`）；S1.5-B play 护栏 5 处钳制（`21f8978`）
- 设计文档：`docs/gaea-space-dimension-design.md` / `gaea-memory-isolation-design.md` / `gaea-space-assembly-design.md`

### 阶段 2 · 双空间壳（前端两视图，S2.1-S2.3）—— 待启动
- S2.1 壳层重构（工位任务工作台 + 乐园沉浸视图 + 空间切换持久化；删除旧 10 板块导航；双首页）
- S2.2 页面迁入 + i18n 全铺 + 性能门控（并入 S0.7 项）+ store/key 空间前缀迁移 + 事件订阅按空间过滤 + Ctrl+K scope
- S2.3 bridge 分面（work/play/shared）+ types 生成化

## 纪律（沿用）

- 每 Step 独立提交可回退；旧数据只读兼容；回退演练纳入验收；不做新板块、不堆功能；
  docs/ 过时文档一律归档到 `docs/archive/`（权威 = 长期规划 + AGENTS.md + 本文件）。
