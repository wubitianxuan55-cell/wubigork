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

### 阶段 2 · 双空间壳（前端两视图，S2.1-S2.3）—— 全部完成
- [x] **S2.1 壳层重构**（设计 `docs/gaea-space-shell-design.md`）：
  - [x] manifest 空间维度（Go + TS 同构）：work/play/shared/**independent**（编程 DSH 独立窗口，
    用户拍板不并入工位/乐园）——导航/启动器/快捷键/导航事件白名单全部由 manifest.space 派生
  - [x] 两视图壳层：CommandRail 空间切换（appStore.space 持久化 `gaea.shell.space`）+ 按空间菜单
    （shared + 当前空间）；独立窗口板块单独 rail 入口；每空间最后页面持久化 `gaea.shell.page.<space>`
  - [x] 双首页：ModuleLauncher 按空间装配（工位=任务工作台门面 / 乐园=会客厅创作间门面）
  - [x] 事件订阅空间过滤：events.ts `subscribeForSpace`（payload.spaceId/space 过滤）；
    TaskCenter/useRunningBadge 只收工位任务（TaskView.spaceId）
  - [x] Ctrl+K 搜索 scope：工位/乐园/全部三档，默认当前空间（工位=UnifiedSearch；乐园=项目搜索）
- [x] **S2.2 增量**：
  - [x] 工作台 localStorage 空间分键 + 迁移（`workbenchStorage`：rightTab/布局/chatTab/
    compact/focus → `gaea.work.*`，旧 key 只读回退）
  - [x] 性能门控：空间切换剪枝 keepAlive 保活页（跨空间页面卸载，后台轮询归零）
  - [x] i18n 全铺第一刀：LocaleProvider 提升根级 + S2.1 新壳 chrome 文案三语字典化
  - [x] **i18n 存量切片（壳层 chrome）**：MainLayout（strip/rail/telemetry 超载警告等）
    + ModuleLauncher（统计卡/语音/信息条/相对时间）+ SearchModal（分类标签/计数）
    全部三语字典化（~90 新 key）；fmtWords/fmtRel 接 translator
  - [x] **i18n 存量切片（SettingsPage 外壳）**：控制室/搜索/分类导航/帮助面板
    三语化（~40 新 key）；分类 label/desc 接 labelKey/descKey（keywords 为搜索数据保留 zh）；
    SettingsPage.test 包 LocaleProvider + 固定 zh
  - [x] **i18n 决策定稿（2026 追加）**：采用审计 §405「诚实 zh-only」选项——
    **壳层 chrome + 设置外壳三语（已完成，消灭壳层混合语言根因）；页面内容层
    保持 zh 单语**，不再逐页铺 en/zh-TW 字典（~5000 字符、边际价值≈0、个人
    中文工具无国际化受众）。若未来需要国际化，按 S2.3b WireShape 模式整页迁移。
  - [x] **页面迁入收口**（`docs/gaea-page-migration-design.md`）：处置矩阵按版本锚点
    落账——办公升格/造价领域包/编程独立/记忆侧栏形态/微信触点 v4.0 已达成；
    **P1 对话流**（chat shared→play：工位不再有独立聊天板块，乐园=会客厅）本轮落地；
    创作间合并=v4.3（§10.4 明示）、模型中心并入设置=v4.1+、微信任务入口=v4.4
- [x] **S2.3 bridge 分面 + types 生成化**：
  - [x] `spaceBindings.ts` 分类表：AppBindings 214 方法显式分类（work/play/shared/
    independent），satisfies + 双向断言编译期全覆盖；UnifiedSearch=shared（scope 隔离）
  - [x] bridge 三门面 workApp/playApp/sharedApp（类型级 + 运行时双保险）；
    TaskCenter/useRunningBadge 消费 workApp
  - [x] types 生成化第一刀：typesGenerationCheck.ts 编译期漂移校验 + 修 SessionMeta.spaceId
  - [x] **S2.3b types 全量迁移**：WireShape 剥生成类实例方法，55 个重叠类型改别名
    （1375→1065 行）；增强类型（TaskStatus/TaskView/SearchScope 等）保留手写；
    顺带修 FileSearchHit.modTime / UpdateInfo.version / FilePreview 契约漂移
  - [x] **S0.7 遗留：151 hex token 化**（审计点名的 VoiceSettingsPanel 浅色主题崩坏
    真 bug 已修——dark-zinc 硬编码 → 主题 token）：语义色（成功/警告/危险/文本/表面/
    边框/主色）全部 token 化；图表调色板/品牌识别色/图片覆盖层/主题预览样张/标注数据色
    显式 `// hex-exempt` 豁免（审计口径「图表豁免」）；新增 eslint 规则
    `local/no-raw-hex`（图表/数据文件白名单 + 行内豁免注释）防回归

### 阶段 3 · 领域包纵深（v4.1 → v4.4）—— v4.1 设计定稿
- [x] **v4.1 办公信任链设计**（`docs/gaea-v41-evidence-chain-design.md`）：
  evidence.Record（原文摘要/来源/模型/时间戳/回滚信息）+ Apply→Verify→Journal
  三段式 + Verifier 双通道（结构/引用完整性 + PDF 视觉 diff）+ 基线快照回滚
  （用户手工编辑冲突保护）+ GB/T 9704 红头 lint 第一刀；Step 拆 v4.1a/b/c
- [x] **v4.1a 证据链核心**：ChangeRecord（原文摘要 8KB）+ ChangeLedger（ctx 盖章）
  + JournalStore（JSONL 追加 + turn markdown 投影）；AgentRunner 回合收尾 flush
  （play 不落盘红线）；edit_file/write_file/move_file 接入上报
- [ ] v4.1a2（multi_edit/edit_lines 摘要 + xlsx_apply 接入 + 前端「证据」入口）
- [ ] v4.1b Verifier（通道 A 公式重算/引用 + 通道 B PDF 视觉 diff + verdict 状态机）
- [ ] v4.1c 规范包（GB/T 9704 红头 lint + 体检报告）

## 纪律（沿用）

- 每 Step 独立提交可回退；旧数据只读兼容；回退演练纳入验收；不做新板块、不堆功能；
  docs/ 过时文档一律归档到 `docs/archive/`（权威 = 长期规划 + AGENTS.md + 本文件）。
