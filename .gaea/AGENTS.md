# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 长期规划（权威，2026 定稿）

- **唯一权威路线图 = `docs/gaea-nextgen-roadmap-2026.md`**（11 个子代理调研合成）：
  8 板块竞品调研（办公/造价/AI 底座/编程/小说/绘梦/轻语/微信+语音）+ WorkBuddy×灵犀
  深度拆解（§12）+ **版本重定义"双空间"（§10）** + 四层落地（§13 后端/前端/UX/UI）+
  执行计划（§14 阶段 0 地基 → 阶段 1 双空间内核 → 阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板：工作与娱乐分开、互不干扰**——工位（办公/造价/编程/资料+工作记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区互不检索、模型策略各配各的、
  上下文永不跨界；跨空间仅用户显式发起（如"把乐园封面放进报告"）。
  ~~"陪伴×办公融合"（旧 v4.3）已删除~~。
- **本轮关键纠正**：灵犀 = 金山 WPS 独立 AI 办公 Agent（非阿里/通义系）；
  WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台（非 Kimi 系；Kimi Work 是月之暗面的）。
- **i18n 决策（2026 追加）**：采用审计 §405「诚实 zh-only」选项——**壳层 chrome +
  设置外壳三语**（已完成，消灭壳层混合语言根因）；**页面内容层保持 zh 单语**，
  不再逐页铺 en/zh-TW 字典（个人中文工具无国际化受众，~5000 字符边际价值≈0；
  未来需国际化时按 S2.3b WireShape 模式整页迁移）。
- **文档纪律**：docs/ 旧调研/已落地计划已归档至 `docs/archive/`（见其 README）；
  后续会话以本文件 + 长期规划 + `.gaea/progress.md` 为权威，勿引用 docs/archive 结论。
- **下一执行**：阶段 0 地基（S0.1 并发加固 → S0.6 edit_file）→ 阶段 1/2 双空间；
  审计结论见长期规划 §11（前 P0 = AgentRunner map 竞态、Registry 无锁、edit_file 未实现）。

## 版本状态

- **最新发布：v4.2.0（2026-08-29）「智慧」工位造价包**：
  git tag `v4.2.0`；CHANGELOG / releases/v4.2.0.md / README 索引同步；设计
  `docs/gaea-v42-cost-ai-design.md`。要点：
  - **AI 组价**：`cost.PriceBand` 价格带纯函数（P25-P75/离散度/离群/置信度/证据链，
    R-7 与 costref 同口径）+ `GaeaCostCompose`（关键词+语义+rerank 相似检索 →
    价格带 → LLM 人材机拆解 `routeSensitiveLocal("office")` 失败规则降级 →
    建议视图）+ `GaeaCostComposeApply` 回写；前端 ComposeModal（测算明细行入口）。
  - **询价飞轮**：`costinquiry` 四源归一数据点 + 到期预警 + 调差建议（标题归一化
    匹配 |差幅|>2%）；前端询价视图（CostLibraryView 第三视图）。
  - **五算对比**：`coststage` 估/概/预/结/决阶段值 + 对比（环比/累计差）+
    偏差三档（正常/关注/异常，阈值 5/15 导出）+ 规则诊断文案；前端 FiveCalcPanel。
  - **验证**：Go 全量绿；vitest **759/759**（142 文件）；tsc/eslint 0；
    绑定面 **517** 漂移 PASS（+11）；spaceBindings **229** 全覆盖；版本五处统一 4.2.0。
  - **下一执行**：v4.3 乐园做深（轻语关系记忆 + 情感语音；创作间一体化，§10.4）；
    组价 LLM 拆解质量依赖办公功能模型（未配置时规则降级可用）；
    复盘笔记 AI 偏差诊断（LLM 生成文案）列后续小步。

- **最新发布：v3.9.0（2026-08-29）「双空间壳 + 办公信任链」**：
  git tag `v3.9.0`；CHANGELOG / releases/v3.9.0.md / README 索引同步。要点：
  - **双空间壳（阶段 2，S2.1-S2.3/S2.3b）**：两视图+空间切换持久化（gaea.shell.space /
    gaea.shell.page.<space>）+ 删旧 10 板块导航 + 双首页；工作台 localStorage 空间分键 +
    keepAlive 跨空间卸载；**i18n 决策**（壳层三语 + 页面 zh-only）；页面迁入 P1（chat→play
    对话流）；bridge 三门面（spaceBindings 214 方法分类）+ types 全量迁移（WireShape）；
    151 hex token 化（VoiceSettingsPanel 浅色 bug 修复 + eslint no-raw-hex）。
  - **v4.1 办公信任链**：证据链（ChangeRecord 原文摘要/ChangeLedger/JournalStore，
    play 不落盘）+ Verifier 双通道复核 + 基线快照回滚（手工编辑冲突保护）+
    GB/T 9704 红头规范体检；前端证据入口（DeliverablesPanel 复核/回滚）。
  - **验证**：Go 全量绿；vitest **738/738**；tsc/eslint 0；vite build；绑定面 **506** 漂移
    PASS；spaceBindings 218 全覆盖；版本五处统一 3.9.0。
  - **下一执行**：v4.2 造价 AI 化（AI 组价 + 询价飞轮 + 五算对比，§10.4）；
    遗留小修（stubGate 竞态/Cancel flake/ProgrammingPage 负载 flake）。

- **最新发布：v3.8.0（2026-08-29）「双空间内核 · 工位/乐园 + 质量地基」**：
  git tag `v3.8.0`；CHANGELOG / releases/v3.8.0.md / README 索引同步。要点：
  - **双空间内核（阶段 1，S1.1-S1.5）**：SchemaV14（facts/tasks space_id 回填 work）+
    会话目录分区 `sessions/work|play` + 事件日志/checkpoint space + `space.mode` 开关；
    记忆写侧盖章（remember/dream 指纹含 space/审计加列）+ 读端隔离（GetInSpace/citations/
    UnifiedSearch scope）+ 前端 scope 切换；`[space_profiles]` 模型 + 工具装配期过滤
    （work 33/play 1/shared 13 + MCP spec 层滤）；任务 per-space 并发/优先级；权限策略按空间
    （play 不弹审批卡）+ play 内容护栏（5 处生成点钳制）。**用户拍板：编程板块保持独立
    DSH 窗口，不并入工位/不共享工具面（防工具膨胀）。**
  - **质量地基（阶段 0）**：S0.1 turnMu 并发加固（worktree 实证修复前必崩）｜S0.2 Registry
    锁+幽灵名｜S0.3 gate 原子化｜S0.4 retry_until 门控｜S0.6 edit_file 工具层（五工具）｜
    隔离岛（knowledge 缓存/office 原子写/secure AES/tasks LRU）｜前端虚拟化+轮询门控｜CI -race。
  - **验证**：Go 全量 **115 包** + vet；vitest 全绿；eslint 0/0；tsc 0；绑定面 **502 方法**
    漂移 PASS；版本五处统一 3.8.0；wails build + 冒烟 200。
  - **下一执行**：阶段 2 双空间壳——S2.1 壳层两视图+空间切换+双首页已收官
    （`docs/gaea-space-shell-design.md`），S2.2/S2.3/S2.3b 已收官；i18n 决策定稿
    （壳层三语 + 页面 zh-only）。**下一执行：阶段 3 v4.1 办公信任链**——设计已定稿
    （`docs/gaea-v41-evidence-chain-design.md`），v4.1a 证据链待实施；
    遗留小修（stubGate 竞态/Cancel flake/持久化套件统一）。

- **最新发布：v3.7.0（2026-08-29）「办公蒸馏 codex 收官 · 引用可追溯 + 审批决策族 + 输出事件化」**：
  git tag `v3.7.0`；CHANGELOG / releases/v3.7.0.md / README 索引同步；内容 = 第二/三刀
  全部 8 个 feat 提交（89cf962/445485f/01bc032/807dcf5/b38e79c/341e78e/9e161fe/cfcf72d）。
  要点：
  - **C2 记忆引用可追溯**：RecallBlock 引用键 `[MEM:name]` + 标注纪律 + 90 天时效提示；
    `memory/citations.go` 回传解析 Touch；前端 remarkMemCitations + MemCitationChip
    弹层（记忆详情/沉淀来源，零新增绑定面）。
  - **C4 审批决策族**：deny/abort 拒绝三分；`[agent] approval_timeout_secs` 超时
    （默认 0=等待）；persist_allow「始终允许」回写 `[permissions].allow`（hardAsk
    降级不回写）；`GaeaApprove` 决策串五值重构，审批卡快捷键 1-5。
  - **C9 任务输出事件化**：gaea-task 事件携带 `outputTail` 整尾回放（独立节流，
    终态兜底带完整尾），输出 dock 事件即推、2s 轮询兜底。
  - **C5 上下文占用状态行**：RunStatus 窗口占用百分比 + 75%/90% 两档压缩前预警。
  - **C6 项目说明文件**：单文件 32KB 注入预算（UTF-8 边界截断留标记）+
    `.gaea/AGENTS.md` 子目录发现（更具体者后注入）。
  - **C3 自动做梦 no-op**：dream 输入 sha256 指纹去重（成功后记录）；三层渐进
    披露确认已存在，LLM 滚动摘要层列观察项。
  - **验证**：Go 全量 **114/114 包** + vet；vitest **681/681（130 文件）**；eslint 0/0；
    tsc 0；绑定面 **499 方法**漂移 PASS；版本四处统一 3.7.0；wails build + 冒烟 200。
  - **下一阶段候选**：权限升级请求（独立一刀）；观察项 C11 压缩口径 / C7 碎片
    抽象 / C10 回合恢复 / 执行容量上限 / 记忆 LLM 滚动摘要层——按真实反馈排期。

- **最新发布：v3.6.0（2026-08-29）「办公文件编辑审阅制 · 本地优先 · 对话面减负」**：
  git tag `v3.6.0`；CHANGELOG / releases/v3.6.0.md / README 索引同步。要点：
  - **xlsx AI 编辑审阅制（Plan→Apply 两段式）**：`GaeaXlsxPlanEdit`（AI 操作集在原文件
    临时副本试运行 + 单元格级 diff 变更清单，不落盘；`xlsxedit.PlanOps/diffOps`）→
    用户批准 → `GaeaXlsxApplyEdit`（excelize 执行 + LibreOffice 重算）；新增 `set_style`
    （叠加样式不丢现有填充色，`mergeStyle`）/`merge_cells`/`unmerge_cells`/`set_col_width`；
    XlsxPreview 规划审阅卡（对标 Copilot Plan/Show Changes）。
  - **xlsx 原生图表**：excelize 原生图表对象嵌入工作簿（Excel/WPS 打开即可见可编辑，
    非图片截图），返回锚点+数据供前端迷你预览。
  - **PDF 统一出口**：`GaeaConvertToPdf`（soffice 无头转换 + 独立 UserInstallation
    profile 防锁冲突；md/markdown 经 docx 中转），顶栏「导出 PDF」同管线；缩略图/预览
    支持 pdf/docx 文本提取（UTF-8 字符边界截断）。
  - **办公本地优先路由**：`routeOfficeLocal`（与 sensitive-local 共用 `routeHerdsmanLocal`，
    source 区分）——办公功能级 AI 调用（Word/Excel 编辑、资料摘要、知识导入、记忆整理）
    默认走本地 Herdsman（数据不出本机、省 token），不可用回退常规路由；聊天主 agent
    不受影响；`GetOfficeLocal/SetOfficeLocal` 绑定 + 安全设置面板开关（配置 `office_local`，
    默认开）。
  - **运行中插话（GaeaSteer）**：对齐豆包「边跑边改」——运行中消息作为当前回合 guidance
    注入（不开新回合、不打断工具），未运行走 GaeaSend 排队兜底；`event.Steer` → notice 回显。
  - **对话面减负**：**回退 C1 方案模式 v1**（Controller mode/shouldPlan/planGate、
    Ask.Plan 计划卡、composer 模式切换器、GaeaAgentMode/GaeaSetAgentMode、Agent.AutoPlan
    配置与 auto_plan_score 字段——与既有 goal/todo 闸门职责重叠，方案模式连同计划流式
    事件列后续观察项）；**撤下任务目标/验收清单**（GoalCard + GaeaRequirement 系 8 绑定 +
    requirements 存取层）；用户消息 Codex 式无气泡 + Kimi Work 式超长折叠（240 字符 3 行）。
  - **修复**：whisper 关机排水丢任务（排队任务对 chain 令牌不可见 → 末轮抽取被整体跳过；
    pending WaitGroup 修复）；trajectory/contextview 绑定层早退路径 nil 切片序列化 null
    崩溃（EmptyTimeline/EmptyTrajectory/EmptyAgentNetwork 非 nil 保证 + 前端归一化）；
    TrajectoryView 测试负载 flaky 加固（显式 5s 超时——教训：全量套件高负载下 jsdom
    调度会饿死 RTL 默认 1s 超时）。
  - **验证**：Go 全量 **114/114 包** + vet；vitest **669/669（127 文件）**；eslint 0/0；
    tsc 0；绑定面 **503→499 方法**漂移 PASS；版本四处统一 3.6.0；wails build + 冒烟 200。
  - **后续（2026-08-29，办公蒸馏 codex 清单第二刀，随下版发布）**：
    - **C2 记忆引用可追溯（feat 89cf962）**：`RecallBlock` 注入行带引用键 `[MEM:name]`
      + 块头句末标注纪律（「记忆可能过时，以工作区文件为准」）+ 陈旧记忆（90 天未修订）
      时效提示；`memory/citations.go` `ExtractCitationNames/ResolveCitations`——回合结束
      （runTurnWithRaw/Run 两路径 `touchMemoryCitations`）解析最终回复并 Touch 命中记忆
      （未知键静默丢弃，正则 `(?i)\[MEM:([a-z0-9][a-z0-9-]*)\]` 与记忆 Name kebab-case
      同构）；前端 `remarkMemCitations`（复用 remarkFileLinks 架构，`mem:` 自定义协议走
      urlTransform 放行 + a 渲染器拦截）+ `MemCitationChip` 点击弹层展示记忆详情/
      沉淀来源（复用现有 `GaeaMemory` 绑定，零新增绑定面）；4 Go + 7 前端新测试，
      vitest 669→676。教训：JS 正则别用 `\[(?:MEM|mem):...\]` 局部交替忘加 `i` 旗标
      （Go 侧 `(?i)` 全局作用域踩过坑）；`vi.mock` 工厂被提升，mock 函数必须
      `vi.hoisted`。
    - **C4 审批决策语义（feat 445485f）**：拒绝三分——deny（拒绝但回合继续，原有）/
      allow / **abort「拒绝并停止本轮」**（对齐 codex `ReviewDecision::Abort`）；
      `approvalReply` +abort 字段，`requestApproval` 收到 abort 即 `c.Cancel()` 终止
      回合（闸门按拒绝返回不记会话放行）；`GaeaApprove(id, allow, session, abort)`
      签名扩展（绑定面仍 499，仅签名），审批卡第 4 按钮（快捷键 4）+ 三语 i18n；
      `TestApprovalAbort` 断言取消触发。遗留观察项：审批超时（TimedOut，无人值守
      自动判定）、权限升级请求、策略文件回写——按真实反馈排期。
    - **验证**：Go 全量 114/114 + vet；vitest **676/676（129 文件）**；eslint 0/0；
      tsc 0；绑定面 499 漂移 PASS。
  - **后续（2026-08-29，办公蒸馏 codex 清单第三刀，随下版发布）**：
    - **C9 任务输出事件化（feat 01bc032）**：`appendOutput` 经 `gaea-task` 事件推送
      输出尾部整尾回放——`Task` 加非持久化事件视图字段 `outputTail`/`outputTruncated`
      （omitempty，List/Get 不带），输出事件独立节流（与进度事件分计时，防紧邻
      Report 被挤掉节流窗口）；`markTerminal` 终态事件兜底带完整尾回放（覆盖节流
      窗口内被跳过的最后几行）；TaskCenter 输出 dock 事件即推，2s 轮询降级为兜底
      （对齐 Codex「事件为主、轮询兜底」）。测试教训：`markTerminal` 先落库后 emit，
      测试断言事件要轮询 collector 而非 `waitTerminal` 后直接读快照（竞态）。
    - **C5 上下文占用状态行（feat，纯前端）**：RunStatus 增窗口占用迷你进度条 +
      百分比；≥75%「接近自动压缩」/≥90%「即将强制压缩」两档预警（对齐 gaea 压缩
      触发线 80%/强制线 90%），窗口未知（window=0）隐藏；usage 事件实时驱动
      （state.context.used 本就逐事件更新）。
    - **C6 项目说明文件（feat，蒸馏 codex agents_md.rs）**：单文件 32KB 注入字节
      预算（`docMaxBytes`，对齐 project_doc_max_bytes 默认），超限 UTF-8 边界截断 +
      `<!-- truncated -->` 标记；每层目录在平铺记忆名之后追加发现 `.gaea/AGENTS.md`
      （更具体者后注入），自动进入记忆面板 DocEditor 可编辑（allowedDocPaths 已
      含已发现文档）。既有分层发现（root→cwd/@import/override）此前已具备，本刀
      只补差距。
    - **验证**：Go 全量 114/114 + vet；vitest **681/681（130 文件）**；eslint 0/0；tsc 0。
  - **后续（2026-08-29 续，C3 收尾 + C4 遗留第一项）**：
    - **C3 自动做梦 no-op（feat，对齐 codex memories/write phase2「无变化即 no-op
      成功」）**：`runDream` 输入 sha256 指纹（`dreamInputHash`），与上次**成功处理**
      的轮次内容一致时直接跳过 LLM 提炼（重试/重入/同内容连问省一次调用）；指纹仅
      在完整处理后记录（SaveDreamFacts 失败不记录，可重试）；仅内存态不跨重启。
      排查结论：三层渐进披露**已存在**——ProfileBlock 画像常载（600 rune）/
      「Saved memories」索引注册表（capMemoryIndex 4KB 截断 + memory_search 提示）/
      RecallBlock 逐轮注入 / memory_search+UnifiedSearch 按需，映射 codex
      summary/registry/rollout 完整；LLM 滚动摘要层（codex memory_summary.md 式）
      列观察项，不预先建管线。
    - **C4 审批等待超时（feat，TimedOut）**：`requestApproval` 配置
      `approvalTimeout>0` 时无人响应按拒绝处理（回合继续、工具结果记拒绝，不静默
      放行）并发 Notice；配置 `[agent] approval_timeout_secs`（默认 0=永久等待，
      交互场景兼容）经 boot 贯通 `control.Options.ApprovalTimeout`；TestApprovalTimeout
      双用例。C4 剩余：权限升级请求、策略文件回写（granted map 重启即失）。
    - **验证**：Go 全量 114/114；前端本刀零改动（上轮 681/681 门禁仍有效）。
  - **后续（2026-08-29 再续，C4 尾项·策略文件回写）**：
    - **Approve 决策串重构（feat）**：`GaeaApprove(id, allow, session, abort)` 4 bool
      → `GaeaApprove(id, decision)`，决策串 `allow_once / allow_session / persist_allow
      / deny / abort`（`control.ApproveDecision` 常量，对齐 codex ReviewDecision 语义族
      + 策略修订分离）；审批卡快捷键固定映射 1-5（hardAsk 隐藏的按钮按键不响应）。
    - **「始终允许」策略回写（feat，persist_allow）**：会话 granted + 经
      `Options.PersistAllowRule` 回调回写——boot 实现用 `config.Load → AddPermissionRule
      ("allow", rule)（幂等去重 + ParseRule 校验）→ Save()`（AtomicWrite + RenderTOML
      →Load 往返保证）；规则串 `"ToolName"` / `"ToolName(subject-glob)"`；hardAsk
      （alwaysPrompt）完全降级——不记 granted、不回写（「任何级别都不自动放行」硬纪律）；
      回调失败仅记日志不阻断批准。frontend 审批卡新增「始终允许」按钮（非 hardAsk，
      快捷键 4）+ 三语 i18n。3 Go 新测试（回写成功/失败不阻断/hardAsk 降级）。
    - **验证**：Go 全量 + vet；tsc 0；eslint 0；vitest **681/681（130 文件）**；
      绑定面 499 方法漂移 PASS（签名变更数量不变）。
  - **下一阶段候选**：权限升级请求（模型临时申请放开某目录/工具，codex
    request_permissions_for_environment——需新工具面 + 审批卡复用，独立一刀）；
    C11 压缩口径、C7 碎片抽象、C10 回合恢复、执行容量上限（观察项，按真实反馈排期）。

- **最新发布：v3.5.0（2026-08-28）「办公对话区标签页 · dsh-context Go 移植」**：
  git tag `v3.5.0`；CHANGELOG / releases/v3.5.0.md / README 索引同步；规划文档
  `docs/2026-08-27-dsh-context-go-port.md`。要点：
  - **对话窗口三标签**：办公聊天窗顶部 `[对话 | 轨迹 | 上下文]`（ChatTabs，localStorage
    持久化）；对话=Transcript 零改动，轨迹=事件账本，上下文=构成看板。
  - **request_header 事件**：`event.RequestHeader` + `request_header` 日志行——每次模型
    请求前记录实际 system prompt 与工具 schema（「模型可见必入日志」请求头落点，旧日志
    无此事件按估算降级）。
  - **上下文标签**：新包 `internal/gaea/contextview`（FoldTimeline 纯函数折叠：六分类
    组成/趋势/事件/节点归档，usage 实际 promptTokens 等比锚定与顶栏同源，`Referenced
    context:` 前缀拆 inject，压缩 gone+负 delta）+ 绑定 `GaeaContextView`（+1）；7 测试。
  - **轨迹标签**：新包 `internal/gaea/trajectory`（FoldTrajectory 对齐 DSH
    ui-trajectory 扁平事件账本：user/header/assistant/tool/compact/ask/approval 记录，
    header change 检测，tool parentId 嵌套与 running/error/截断/耗时，轮间压缩
    Between-turns 区段，turn-end 错误）+ 绑定 `GaeaTrajectory`（+1）+ TrajectoryView
    （chips/搜索/徽标/检查器）；9 折叠 + 5 vitest。
  - **Agent 网络**：`FoldAgentNetwork` 子代理树（拥有子记录的元工具调用，不写死工具名）
    + 绑定 `GaeaAgentNetwork`（+1，subagents meta 任务摘要匹配富化）+ AgentNetworkCard
    （SVG 树/节点 token 环/running 绿脉冲/悬停详情）；3+3 测试。
  - **随版并入**：记忆统一层第二刀前端收尾（GaeaMemoryUnarchiveBatch /
    GaeaMemorySetRetentionDays 的 bridge 类型、批量恢复/保留期 UI 与 mock、生命周期测试）。
  - **验证**：Go 全量 **114/114 包** + vet；vitest **668/668（127 文件）**；eslint 0/0；
    tsc 0；绑定面 **503 方法**漂移 PASS；版本四处统一 3.5.0；wails build + 冒烟 200。
  - **下一阶段（Phase D）**：增量（Delta）模式、上下文浏览器、`/context` 命令、
    File activity、SSE 增量刷新、轨迹时间条 Overview 投影与虚拟滚动、
    Agent 节点点击跳转子代理会话。

- **最新发布：v3.4.0（2026-08-27）「记忆统一层第一刀 · 统一检索收口 + 生命周期产品化」**：
  git tag `v3.4.0`；CHANGELOG / releases/v3.4.0.md / README 索引同步；计划
  `docs/superpowers/plans/2026-08-27-记忆统一层第一刀.md`。要点：
  - **统一检索后端收口**：`GaeaUnifiedSearch` 视图扩展四组——keyword（工作区全文）+
    semantic（跨库语义）+ **brain（三脑命中，新增；a.brain==nil 时空数组不报错）** +
    **files（文件语义，新增；复用 GaeaFileSemanticSearch 抽出的私有实现 fileSemanticHits）**；
    hub 搜索（MemoryHubPage.runSearch）由「4 绑定 Promise.all 前端拼装」（BrainSearch +
    WorkspaceSearch + SemanticSearch + FileSemanticSearch）收敛为「单次 app.UnifiedSearch」，
    四组映射回原 HubSearchHit 渲染（徽标/预览/@ 引用零变化）；WorkspaceSearchPanel 跨库
    模式零改动，其 kindChip 补 file kind（后端本就返回，前端类型漏声明）。
  - **归档 tab 永远空白（缺陷修复）**：前端归档 tab 读 `view.archives`，但后端 `GaeaMemory()`
    的 MemoryView 结构体**没有 archives 字段** → 列表永远空白；改 `GaeaMemoryArchivedList`
    分页加载（每页 50 + 加载更多 + total 展示）。
  - **恢复能力补齐（Unarchive）**：memory 包此前只有 Archive（软删）无恢复路径（注释声称
    「90 天可恢复」但实际不存在）；补双后端 `Store.Unarchive`（sqlite 置 archived=0 +
    updated_at；file 从 `.archive/<ts>-<name>.md` 移回主目录 + reindex；未归档/已硬删报错）+
    新绑定 `GaeaMemoryUnarchive`（绑定面 497→**498**）+ 归档 tab「恢复」按钮（Rollback 图标/
    恢复中态/成功后刷新提示）。
  - **保留期下发展示**：`MemoryArchivedPage` 增 `RetentionDays`（= ArchivedRetention 90 天），
    归档 tab 顶部「归档保留 N 天，超期可清理」，清理确认弹窗文案跟随真实保留期。
  - **修复漂移脚本单条差异静默放行（质量 bug）**：`check-bindings-drift.ps1` 判
    `$diff.Count -gt 0` 但 **PS 5.1 下单条差异 `$diff` 是单个 PSCustomObject（无 .Count
    属性）→ `$null -gt 0` 为 False 静默放行**（实测复现：新增 GaeaMemoryUnarchive 后脚本
    仍报 OK）；`@()` 强制数组化修复 + 负向验证（单条漂移现在 exit 1）。**教训：PS 脚本里
    对可能为单个对象的管道结果判 Count 前先 `@()` 包裹。**
  - **验证**：Go 全量 **112/112 包**（+6：Unarchive 双后端/app 绑定/RetentionDays/BrainNil）、
    eslint **0/0**、tsc 0 errors、vitest **654/654（124 文件，+2：归档 tab 交互 +
    UnifiedSearch 扩展契约）**、绑定面 **498 方法**漂移 PASS、版本四处统一 3.4.0、
    wails build + 冒烟 /api/health 200；v3.3.0 资产归档 releases/archive/。
  - **下一阶段**（记忆统一层后续 + v3.2.0 里程碑剩余）：生命周期产品化收尾（归档保留期
    可配置/批量恢复）；受控自主（goal gate 深化：目标验收自动追踪产品化、审批流收敛）；
    C9 分栏对照留待验证真实需求；造价数据库体验收口（手册二期/测算项目导入导出/分类树
    维护）；XlsxPreview 虚拟滚动待真实卡顿反馈。

- **最新发布：v3.3.0（2026-08-27）「质量收敛 · eslint 存量 warnings 清零 + flaky 治理」**：
  git tag `v3.3.0`；CHANGELOG / releases/v3.3.0.md / README 索引同步。要点：
  - **eslint 366 → 0（errors 0 / warnings 0）**，四层收敛：
    ① 配置显式化——`no-unused-vars` 加 `^_` 前缀 ignore patterns（下划线=显式「故意
    不用」，社区标准）、`no-empty` 开 `allowEmptyCatch`（空 catch 为降级吞错设计）、
    react-refresh 开 `allowConstantExport`；② 死代码清理 56 处（未用 import/const/
    函数/catch 参数/解构成员，跨 40 文件；含 mock chat.ts 只写不读 `cancelled` 及其
    两处死写入）；③ exhaustive-deps 40 处——稳定依赖补全 / 不稳定依赖显式 disable
    注释（含 GhostText/useVoiceChat 两处 TDZ 陷阱 useCallback 定义上移重排、
    GraphView/Composer 两处 disable 位置修正）/ 复杂表达式提取变量 / 每渲染重建数组
    wrap useMemo / ref cleanup 竞态局部变量化；④ react-refresh 25 处混合导出加文件级
    显式声明（14 文件）+ 移除 10 处冗余 `@ts-ignore`（wails.d.ts 类型早已生成，tsc 验证
    0 错误）。
  - **flaky 治理**：filewatch 测试超时 3s→5s（沙箱/CI 高负载下 fsnotify 投递延迟曾致
    首跑假红复跑绿）；CI 后端测试失败后整体重试一次（重试后仍失败正常红）；确认 CI
    已排除 internal/tts、test-all.ps1 已有 AV 锁重试。
  - **releases/README.md 乱码恢复**：v2.40.0 及更早 98 行 GBK 损坏（U+FFFD 不可逆）
    从 git 历史（v3.0.1 提交 7c53db8 干净版）逐行重建，0 残留。
  - **前端性能体检**：大组件 memo 复查（页面级组件收益有限不额外加）；唯一热点 =
    XlsxPreview Excel 网格全量渲染（maxRow×maxCol `<td>`），修复需虚拟滚动重构、
    收益/风险比低——按「先体检再决定」纪律记录待真实卡顿反馈。
  - **验证**：eslint **0/0**（366→0）、tsc 0 errors、vitest **652/652（124 文件）**
    零回归、Go 全量 **112/112 包**、filewatch 5 测试绿、绑定面 **497 方法**漂移 PASS
    （零新绑定）、版本四处统一 3.3.0、wails build + 冒烟 /api/health 200；v3.2.1 归档
    releases/archive/。
  - **下一阶段**（v3.2.0 里程碑剩余）：记忆统一层（路线图 V4）+ 受控自主（goal gate
    深化）；C9 分栏对照留待验证真实需求；造价数据库体验收口（手册二期/测算项目导入
    导出/分类树维护）。eslint 已清零——CI 门禁「存量 warn 随迭代清理」正式收官。

- **最新发布：v3.2.1（2026-08-26）「工作区内联编辑 · C5 文本文件直接编辑保存」**：
  git tag `v3.2.1`；CHANGELOG / releases/v3.2.1.md / README 索引同步。要点：
  - **C5（蒸馏候选清单 9 项全部收官）**：新绑定 `GaeaWriteFile(rel, content)`（绑定面
    497，+1）——四重校验：相对路径（拒绝对/..穿越）+ 写根（WriteRoots：工作区 +
    allow_write）+ 文本扩展名白名单 30 种 + ≤2MB + 仅已存在文件；原子写（临时文件 +
    fsync + rename，失败保留原文件）；用户显式保存=用户意图不走审批（agent 写仍受
    权限面约束）。FilePreview 编辑模式（markdown/text 且未截断才可编辑）：脏标记 /
    Ctrl+S / 保存状态机（失败可重试）/ 保存后自动重读预览 / 脏退出内联确认条。
  - **验证**：Go 全量测试绿（TestGaeaWriteFile：正常写回 + 五类拒绝 + 拒绝不改动原文件）；
    vitest **652/652（124 文件，+5：FilePreview 编辑模式）**、tsc/eslint 0 errors
    （366 存量 warnings）；绑定面 497 漂移 PASS；wails build + 冒烟 /api/health 200。
  - **下一阶段**（v3.2.0 里程碑剩余）：记忆统一层（路线图 V4）+ 受控自主（goal gate
    深化）；C9 分栏对照留待验证真实需求；质量收敛（eslint 366 warnings / flaky 治理 /
    前端性能复查）；造价数据库体验收口（手册二期/测算项目导入导出/分类树维护）。

- **最新发布：v3.2.0（2026-08-26）「任务可见性 · C1 任务实时输出 + C2 子代理活动行」**：
  git tag `v3.2.0`；CHANGELOG / releases/v3.2.0.md / README 索引同步。要点：
  - **C1 任务实时输出**：tasks 包 `Progress.Output(line)` 输出环形缓冲（200 行/64KB 上限，
    超限截断标注、只回放不消费游标）；三个消费者（价格抓取/批量/语义索引）逐源逐批
    时间戳输出；新绑定 `GaeaTaskOutput(taskID) → {tail, truncated}`（绑定面 496，+1）；
    任务中心选中任务 → 底部输出 dock（pre 回放、运行中 2s 轮询 + 尾随滚动、截断标注、
    可关闭，终态仍可复核）。
  - **C1 结束态细分 `stopping`**：取消 running 先条件置 stopping（WHERE status='running'
    防覆盖终态竞态）再传播取消，handler 退出终态 cancelled；前端「停止中」琥珀色徽标；
    重启续跑把 stopping 一并恢复 queued。
  - **C2 子代理活动行**：`summarizeSubagentTranscript` 从 transcript 尾部派生 lastText
    （最后 assistant 文本 160 字）与 lastTool（工具名+结果首行 80 字），随 SubagentRunView
    下发；分工面板运行中卡片显示「正在：…」+「⚙ 工具」两行活动线。父子拓扑暂不做
    （meta 无父子记录，退化为活动行 + 扁平列表）。
  - **修复数据备份隔离缺陷**：`dataBackupPlan` 混用 DataRoot（尊重 GAEA_DATA_ROOT）与
    MemoryUserDir/UserConfigPath/SessionDir/ArchiveDir（永远真实用户目录）——测试隔离时
    zip 混入真实数据 + 相对路径穿越；统一从 DataRoot 派生全部条目（生产行为不变）。
  - **验证**：Go 全量测试绿（tasks +2 输出缓冲/stopping 竞态、app +1 活动行派生、备份
    修复回归）；vitest **647/647（123 文件，+4：TaskCenter 3 + SubagentsPanel 1）**、
    tsc/eslint 0 errors（360 存量 warnings）；绑定面 496 漂移 PASS；wails build + 冒烟
    /api/health 200。
  - **下一阶段**（v3.2.0 后续刀）：C5 工作区内联编辑（需 GaeaWriteFile）/ C9 分栏对照
    （候选清单建议验证真实需求后再启动）；记忆统一层 + 受控自主（路线图 3.2.0）；
    eslint 存量 warnings 收敛 + flaky 治理。

- **最新发布：v3.1.1（2026-08-26）「造价数据库闭环补齐 · 测算项目 UI + 造价参考 + 复盘笔记 + 选区转对话」**：
  git tag `v3.1.1`；CHANGELOG / releases/v3.1.1.md / README 索引同步。要点：
  - **测算项目 UI**（CostProjectsView，造价板块新增导航）：左列项目列表（状态徽标/条目数/合计/
    版本数）+ 右区详情——项目信息编辑、工程量清单行内编辑（失焦保存、金额=数量×单价实时算、
    「引用成本库单价」搜索下拉回填）、保存版本（不可变快照 + 备注 + 查看明细表 + 前端编排恢复）、
    **沉淀选中行回成本库**（UPSERT + 刷新库概览）。全部复用 v3.1.0 绑定，零后端改动。
  - **造价参考 UI**（CostIndicatorsView）：按科目/分类分组切换 + 分位数表格（P25/中位数/均值/P75），
    实时聚合不落表；空态引导「保存版本或沉淀后自动成为样本」。
  - **复盘笔记 UI**（CostNotesView）：搜索 + 状态过滤 + 编辑弹窗（结论/边界/风险/证据/可信度/
    复核状态/分类/类型/有效期）+ 引用次数；删除 Modal.confirm。
  - **板块导航同步**：board cost manifest Nav 增 测算项目/造价参考/复盘笔记（与页面 MODULES 对齐）。
  - **C4 选区转对话**（SelectionToComposer，纯前端）：办公板选中正文 → 浮动「转为提问」→ `> 引用`
    插入输入框；忽略输入框/弹窗内选区；portal 渲染。
  - **仓储卫生**：删根目录 `.go`/`.split.go`/旧 `gaea.exe`；releases/README.md 补 v3.0.8 行 +
    修 v3.0.1/v3.0.0 乱码（更深历史乱码保留）。
  - **验证**：绑定面 495 方法零新增漂移 PASS；vitest **643/643（122 文件，+13）**、tsc/eslint 0 errors
    （359 存量 warnings）；Go build/vet + board/bindings 测试绿；wails build + 冒烟 /api/health 200。
  - **下一阶段**（v3.2.0 候选）：蒸馏收尾 C1 任务实时输出 / C2 子代理活动行（需后端字段）/
    C5 工作区内联编辑（需 GaeaWriteFile）/ C9 分栏对照；记忆统一层 + 受控自主（路线图 3.2.0）；
    eslint 存量 warnings 收敛 + flaky 治理。

- **最新发布：v3.1.0（2026-08-26）「造价数据库 · 一级板块 + 办公蒸馏 + 死锁修复」**：
  git tag `v3.1.0`；CHANGELOG / releases/v3.1.0.md / README 索引同步。要点：
  - **一级板块「造价数据库」**（zaojia-database 蒸馏，2026-08-19 启动）：成本库从记忆中枢
    二级分类提升为独立板块（board cost，`CostLibraryPage`，MenuOrder 5，导航：概览/成本条目/
    价格源/价格仓库）；记忆中枢成本二级入口移除（记忆图谱节点保留琥珀色）。
  - **综合单价架构**（用户定调「数据库就是数据库」）：综合单价=一级、人材机=二级组成
    （SchemaV12/V13：cost_entries 增人工/材料/机械合计 + 管理/利润/垫资/税率仅展示；
    `cost_entry_components` 组成行表；Save 组成行整组替换）；价格三要素与溯源
    （SchemaV9：region/price_date/price_type/valid_until/source_row，history 同步）；
    默认分类树重构（综合单价→专业→分部）；《市政成本测算手册》整本导入实测 8 表 234 条
    全命中。调研文档 `docs/market-research-2026-08-cost-architecture-zonghe-danjia.md`。
  - **测算项目 + 造价参考**（新包 costproject/costref，SchemaV10 三表）：测算项目容器 +
    明细行（数量×单价自动算金额）+ 不可变版本快照 + 「沉淀」UPSERT 回成本库；
    costref 实时聚合分位数指标（不落表）+ 复盘笔记；`cost_indicators` 办公 agent 工具。
  - **数据自愈**（cost/repair.go）：201 条非法 category_path 规则引擎幂等映射回合法路径 +
    保守回填地区/期数，`Store.Open` 自动执行。
  - **办公蒸馏**（2026-08-20/26 两轮，全部随 v3.1.0 发布）：C3 会话级右侧面板持久化 /
    C6 运行域活动角标（useRunningBadge）/ C7 预览队列 chip 化；FileTree → 资源管理器
    （行悬浮 @引用/右键菜单/cwd 持久化/树内搜索）；删除完成轮大过程卡（Transcript 交替
    语义）；GoalCard/TodoCard 默认折叠紧凑化；Tailwind v4 `max-w-(--maxw)` 括号语法修复。
  - **成本库入口接线（用户决策 2026-08-26）**：办公右侧「文件」组新增「成本库」子 Tab
    （CostLibraryPanel），4 主 Tab 收敛不变。
  - **修复办公板块初始化死锁**：`GaeaInit` 持 `ga.mu` 时 `resumeLastSession →
    syncGoalForSession` 对同一把非重入锁二次加锁 → 永久卡死；`syncGoalForSession` 改显式
    接收控制器 + 两个回归测试；`persistWorkspaceLocked` 同步磁盘 workspace_root。
  - **绑定面 495 方法**（+15：CostB 测算项目/明细/版本/沉淀 + 造价参考/复盘笔记/指标），
    漂移 PASS；版本四处统一 3.1.0；验证：go 全量测试绿（filewatch 首跑环境抖动复跑绿）、
    tsc/eslint 0 errors、vitest **630/630（118 文件）**、wails build + 冒烟 /api/health 200。
  - **下一阶段候选**（见 docs/superpowers/plans/2026-08-26-…右侧面板.md 与蒸馏候选清单）：
    C1 任务实时输出 / C2 子代理活动行（需后端字段）/ C4 选区转对话 / C5 工作区内联编辑
    （需 GaeaWriteFile）/ C9 分栏对照；造价数据库体验收口（CostLibraryPanel 已是入口，
    待测：测算项目页 UI 打磨、造价参考应用、手册二期）。
  - **下一会话计划（2026-08-26 更新）**：按
    `docs/superpowers/plans/2026-08-26-gaea-下一阶段-优化迭代方向.md` 执行——主线 =
    v3.1.1「造价数据库闭环补齐」（测算项目 UI + 造价参考/复盘笔记 UI：后端 costproject/
    costref 与 15 个绑定已就绪，纯前端 + vitest）+ C4 选区转对话 + 仓储卫生；然后 v3.2.0
    （C1/C2/C5/C9 蒸馏收尾 + 记忆统一层 + 受控自主）。纪律不变：不做新板块、不堆功能、
    每 Step 独立提交可回退。

- **路线决策（2026-08-26）：V4.0 dsh化 验证失败，正式废弃；继续 V3 迭代。** 曾把 gaea 改造成
  DSH 插件体系（独立工作空间 `C:\AI\gaea-v4`，DSH 底座 + `packages/gaea/*` 插件，V4.19.0）——
  用户验证后判定失败。已删除**工作空间内** V4.0 相关文档：`docs/superpowers/specs/2026-06-29-wubigork-v4-blueprint.md`、
  `docs/superpowers/plans/2026-06-29-phase4-data-foundation.md`、`phase4.1-copilot.md`、
  CHANGELOG.md 的 v4.0.0「织梦者」章节。**纪律：`C:\AI\gaea-v4` 与 `~/.dsh*`（会话/记忆/编码记忆）为
  工作空间外文件，不删不动**；勿再引用或复活 V4.0 路线。开发主线 = 本仓库（gaea V3 桌面端，
  Wails + Go + React，当前已发布 v3.1.0）。

- **右侧面板改造（2026-08-26，蒸馏 dsh-better-sidebar，用户拍板不装 DSH 插件、直接改 gaea 代码）**：
  承接 2026-08-20 蒸馏候选清单（C1-C9），本轮落地三项纯前端（计划
  `docs/superpowers/plans/2026-08-26-办公蒸馏-better-sidebar-右侧面板.md`）：
  - **C3 会话级布局持久化**：`rightTab` 改为按会话 key（`gaea.rightPanel.v1:<sessionKey>`）读写，
    切会话/新建/恢复各自恢复面板关注点；无会话路径回退全局 key `gaea.workspace.rightTab`
    （旧值兼容）。`currentSessionPath`/`currentSessionKey` 上移到 App.tsx 顶部（rightTab 之前）。
  - **C6 运行域活动角标**：`useRunningBadge` hook（TaskList 基线 + gaea-task 事件增量，重算活跃任务数），
    WorkspaceTabs 新增 `badges` prop，运行组未激活时显示计数角标（99+ 封顶）。
  - **C7 预览队列 chip 化**：PreviewNavBar 从 ← index/total → 升级为文件 chip 条（basename、点击切换、
    × 关闭、中键关闭 VS Code 语义按下/弹起同目标才关）；`usePreviewStore` 增 `navTo(index)` 与
    `closePreviewAt(index)`（删当前项跳相邻、删唯一项清空）。
  - 验证：tsc 0 errors、vitest **629/629（118 文件）**全绿（新增 42 用例：workspaceTabs 会话 key 5、
    previewQueue closePreviewAt/navTo 5、PreviewNavBar chip 6、WorkspaceTabs badge 5 + 更新旧断言）。
  - 后续轮次：C1 任务实时输出（需后端 GaeaTaskOutput）/ C2 子代理活动行（需后端字段）/ C4 选区转对话 /
    C5 工作区内联编辑（需 GaeaWriteFile）/ C9 分栏对照。

- **删除输出显示的大过程卡（2026-08-26，用户决策）**：办公板块（`Transcript.tsx`）已完成轮不再把
  「思考 + 工具 + 中间正文」整轮合并成一张默认展开的大过程卡（`consolidatedSegments` 已删除）；
  所有轮次统一走 `alternatingSegments` 交替——正文（含中间正文）始终以独立消息显示、不进卡片，
  过程（思考/工具）单独成小卡。**纪律：交替过程卡逻辑 / `ProcessCard` / `TurnBlock` 组件零改动**
  （只删大卡合并函数、Transcript 渲染 key 去掉 `done-` 前缀防止完成后重挂载丢失手动折叠态、注释
  对齐）；`Transcript.test.ts` 断言重写为交替语义（3 用例）。验证：tsc 0 errors、vitest 629/629 全绿。

- **目标卡/待办卡紧凑化（2026-08-26，用户：太占地方）**：
  - `GoalCard`：由「始终展开不可折叠」改为**默认折叠**——折叠态只占一行（chevron + 标题 + 状态徽标 +
    进度 + 目标文本截断，整行可点击展开）；展开才显示验收清单 / 添加 / 操作 / 自动追踪开关（开关移入
    展开区操作行，头部不再常驻）。展开体条件渲染（折叠时不占 DOM，对齐 TodoCard 模式）；头部 button
    显式 `aria-label="任务目标"`。
  - `TodoCard`：头部收紧——展开按钮由文字改图标（ChevronRight/ChevronDown），padding 紧凑化，
    关闭按钮图标化（`aria-label="展开待办/收起待办/关闭待办"`）。
  - 验证：tsc 0 errors、vitest **630/630**（118 文件）全绿（GoalCard 测试改为「默认折叠 + 展开交互」
    9 用例，新增折叠态断言 1）。

- **卡片宽度与输入框对齐（2026-08-26，用户）**：发现并修复 **Tailwind v4 下 `max-w-[--maxw]` 任意值
  语法不生成 CSS**（计算样式 `max-width: none`）——GoalCard/TodoCard/PromptShelf 的宽度约束此前
  全部失效，宽窗口下会无限撑宽；输入框正常仅因 `.composer-glow`（styles.css 手写
  `max-width: var(--maxw)`）兜底。统一改为 Tailwind v4 括号语法 **`max-w-(--maxw)`**（四处：
  GoalCard / TodoCard / PromptShelf / Composer），实测计算样式 `max-width: min(1000px, 100%)`、
  卡片与输入框完全对齐。**教训：本项目 Tailwind v4 下 CSS 变量任意值一律用 `max-w-(--var)` 括号
  语法，`[--var]` 方括号旧语法不生效**。验证：tsc 0 errors、vitest 630/630 全绿。

- **V3.0.0 已发布（2026-08-15）**：git tag `v3.0.0`；星枢 Constellation OS UI 革命性重设计首发
  （详情 CHANGELOG.md + releases/v3.0.0.md）。3.0.0 首发目标（chat 会话可恢复 + 知识库试点）已随
  Step 0-3 架构改造 + UI 重设计达成；下一版本节奏 = 3.1.0 板块生态·记忆起步。

- **V3.0 总规划（2026-08-15 定稿，V1-V8 已全部确认）**：docs/2026-08-15-gaea3-vision-roadmap.md（个人 AI 智能体平台：一个内核 + 统一记忆 + 板块插件化 + 本地优先/分层智能 + 移动终端）；架构执行见 docs/2026-08-15-gaea3-architecture-design.md（Step 0-3）。已确认决策：V1 愿景=个人 AI 智能体平台；V2 3.0.0 首发=chat 会话可恢复+知识库试点；V3 版本节奏=3.0.0 地基→3.1.0 板块生态·记忆起步→3.2.0 受控自主→3.3.0+ 身份；V4 记忆统一层 3.2.0；V5 数字生命 3.3+ 再定；V6 启动用户试用；V7 分层智能模型策略（云端统筹+本地执行+能本地则本地）；V8 插件化边界（只做启动期声明式装配，不做运行期热替换；工具级 MCP 热增删保留例外）；D8 office 模块路由 GaeaSend；单窗口编排已废弃。~~编程板块搁置（用户指示）~~ → **已恢复（2026-08-16）：编程板块桌面内嵌 Harness Web 工作台落地**。

- **最新发布：v3.0.8（2026-08-17）「办公板块表格可交付 + 会话产物打包 + 多智能体分工 + 界面收敛」**：
  - **市场调研**：docs/market-research-2026-08-office-table-agent-and-package.md（表格 Agent
    「可交付」+ 会话产物打包 + 补充调研：多 Agent 团队化/AI PPT 全流程/政务场景）。
  - **P0-1 会话产物一键打包 Zip**：`GaeaZipDeliverables`（只接受工作区相对路径，拒绝绝对路径
    与 `..` 穿越、缺失/目录静默跳过、zip 内保留相对路径结构防同名覆盖）+ 会话产物面板
    「打包下载」按钮（zipping 态 + toast + 文件管理器定位）。对标 Kimi 工作空间/WorkBuddy。
  - **P0-2 表格「选中区域 → 一键图表」**：`GaeaXlsxChart`（选中区域 `A1:B6`/单单元格 `B2`=表头到
    选中行/空=自动前两列数据行 → 标签列+数值列 → matplotlib PNG 落 .gaea/exports → 预览队列，
    复用 crosslink.GenerateChartPNG）+ XlsxPreview「图表 ▾」下拉（柱状/折线/饼图 PNG + 图表→Word/
    →PPT）。对标千问表格 Agent/ChatExcel/GLM in Excel。
  - **P1 产物缩略图增强**：FileThumb 升级为内容缩略图（xlsx 前 3×3 迷你表格/md 文本摘要/图片
    dataURL，失败静默回退类型图标），接入 DeliverableCards 与会话产物面板，**零新后端绑定**
    （复用 GaeaPreview 结构化数据）。
  - **P2-1 多智能体分工可见**：`GaeaSubagentRuns`（读 `<sessionDir>/subagents/` 的 meta +
    transcript 派生：状态/任务摘要=首条 user 消息/最后回答/工具调用次数，路径经 sessionDirForPath
    防穿越，无目录返回 available=false）+ 右侧「运行」组「分工」子面板（SubagentsPanel：
    状态徽标/任务摘要/模型/工具范围/耗时/展开回答，运行中 5 秒轮询）。对标 WorkSwarm 蜂群/
    QClaw V2 多 Agent/飞书 aily 同事。
  - **右侧面板 Tab 收敛为 4 个主标签（用户决策：不堆功能）**：7 个子面板按「文件域/成果域/
    运行域/分析域」归入 4 个主 Tab——文件（文件/资料）、成果（产物/变更）、运行（任务/分工）、
    分析（统计）；workspaceTabs 重构为 WORKSPACE_GROUPS 分组清单 + groupOfTab/defaultTabOfGroup
    映射 + 扁平兼容导出，App.tsx 渲染分支与命令面板零改动；WorkspaceTabs 两级渲染（第一级主
    Tab + 第二级组内子 Tab，data-grouptab/data-subtab 测试锚点）。
  - **Excel 编辑器工具栏收敛（用户决策：不做 PPT、聚焦 Word/Excel、优化现有）**：① 顶部 10 个
    常驻按钮 → 行操作（选中才显示）+ 重算公式 + 图表 ▾ 下拉；② 选中单元格布局重排为「公式栏
    （写值/公式）在上、AI 编辑（自然语言指令）在下」——两个输入框逻辑分层，AI 编辑从两行大块
    收成单行紧凑（预设回填指令有激活态 + 输入框 + 执行 + 关闭）。原则：**入口按上下文收敛，
    不按功能铺开**。Word（DocxPreview）检查后确认已收敛未改动（不为了对称而改）。
  - **验证**：Go `internal/app` 全量 ok（12 个新测试）；绑定面 **480 方法**漂移 PASS（+3：
    GaeaZipDeliverables/GaeaXlsxChart/GaeaSubagentRuns）；前端 tsc 0 errors、vitest **605 通过**
    （新增 18 用例）、vite build 通过；wails build 发布版（35.3MB）+ 冒烟 /api/health 200。
  - 版本四处统一 3.0.8（sync-version 三处 + package.json）；README 版本索引新增 v3.0.8 行；
    v3.0.7 资产归档 releases/archive/（exe/sha/md）；git tag `v3.0.8`，commit `219d019`。
  - **决策记录**：PPT 分阶段工作流取消（用户：PPT 有专门软件，平时用得不多，聚焦 Word/Excel）；
    办公迭代原则 = 先体检现有再决定改什么、入口按上下文收敛、不堆功能。

- **最新发布：v3.0.7（2026-08-17）「办公板块文件交互体验 + 内置 prompt 模板兜底」**：
  - **文件交互 P0-P2 全落地**（调研 docs/2026-08-16-office-file-interaction-research.md）：
    P0-1 非图片附件 chip 化（docx/xlsx 拖入不再注入裸 @路径，进附件栏渲染为
    「图标+文件名+扩展名 badge」chip，点击预览/移除，提交仍统一注入 @路径零变化）；
    P0-2 行内文件 chip 视觉统一（FileChip 组件 + lib/fileBadge 扩展名单源，FileLinkText/
    Markdown/流式尾部全部升级）；P0-3 最近文件快捷区（lib/recentFiles localStorage
    单源，@ 引用与预览共用去重置顶 20 条，文件面板顶部 RecentFilesBar）；
    P1-1 多文件预览队列（preview store 扩展 previewList/index/navPreview 单源，App
    预览状态全部改由 store 驱动消除双写，预览底部 ←/→ 导航条 PreviewNavBar）；
    P1-2 产物版本时间线（sessionDeliverables 记录 versions，产物面板 vN 徽标）；
    P2-2 大工具输出有界预览（boundedOutput >60 行折叠 + ToolCard 展开全部开关）；
    P2-4 附件上下文占用透明化（附件 chip 显示 KB 占用）。
  - **内置 prompt 模板兜底（SetPromptFS）**：main.go go:embed 内置 prompts/ 模板，
    exe 单文件分发（旁边无 prompts/ 目录）时兜底，磁盘 prompts/ 仍优先（开发期
    直接改模板生效）；prompt 引擎 6 新测试。
  - **验证**：前端 tsc/eslint 0 errors、vitest **587 通过**（新增办公文件交互 40+
    用例）、vite build 通过；Go prompt/app/main 全绿；绑定面 **477 方法**漂移 PASS；
    wails build 发布版 35.2MB；冒烟 /api/health 200。
  - 版本四处统一 3.0.7（sync-version 三处 + package.json/package-lock）；README
    版本索引新增 v3.0.7 行（含补齐 v3.0.6 缺行）；v3.0.2 归档；git tag `v3.0.7`。

- **最新发布：v3.0.6（2026-08-16）「编程板块工作台 + 办公会话回退分叉 + 顶栏工具栏迁移」**：
  - 编程板块：DeepSeek Harness Web 以 iframe 内嵌桌面窗口（http://127.0.0.1:3080，
    未运行显示启动引导）；启动引导 = 真实前置条件逐项检查（GetProgrammingWebPreflight）+
    日志尾部查看（ProgrammingWebLogTail）+ 启动动画视图（失败自动展开日志）；后端全链路
    probe 探针注入，programming_web_test.go 16 用例；绑定面 470 方法漂移 PASS。
  - **顶栏工具栏迁移决策**：运行中工具栏（Harness Web 徽标/URL/刷新/浏览器打开/停止）
    经 portal 进 MainLayout v3-strip 的 `v3-prog-host` 宿主（与聊天模式条 v3-chatmode-host
    同款模式），仅编程板块激活时显示、其他板块隐藏；iframe 独占工作区全高；宿主缺失
    兜底保持原布局。新增板块级工具栏一律走 portal 宿主模式，不在页面内再放横条。
  - 办公板块：会话回退/分叉/回退点后端落地（兑现前端 rewind 链路，此前为空实现）+
    GaeaSessionStats 恢复回填 + 右侧 Tab 清单化（workspaceTabs 单源）+ 死绑定清理 +
    mock 场景补全（?mock= 优先）。
  - 版本统一 3.0.6（sync-version 三处 + package.json/package-lock）；README 版本索引补齐
    v3.0.2–v3.0.5 缺行；v3.0.1 归档；git tag `v3.0.6` 已推送 origin/main。
- **下一会话计划（2026-08-15 定）**：开始写代码。顺序 = 阶段 7（v2.34-2.37 正确性纵深，既有计划照旧）→ Step 0（office 模块补注册路由 GaeaSend + MainBrainChat 全链路测试 + 版本源三处同步脚本化，0.5 天，可搭阶段 7 任一刀发布）→ Step 1 事件日志。**回退保障为硬要求**（用户强调）：每 Step 独立提交可 revert、旧数据只读兼容、二进制保留 5 版、运行时开关可切旧机制、每 Step 验收含回退演练——四层保障详见架构文档 §6.6。
- **架构方向备忘（2026-08-15）**：3.0 方向 = 向 DSH 靠拢的插件化地基（事件日志事实源 / 板块 Manifest / Provider Seam，见 docs/2026-08-15-gaea3-architecture-design.md）。**单窗口编排方案（原 docs/gaea2/module-protocol.md §5）已废弃并删除该文档，勿再引用或复活。**

- **3.0 架构改造设计（2026-08-15 定稿，待评审后开工）**：权威文档 docs/2026-08-15-gaea3-architecture-design.md（事件日志事实源 / 板块 Manifest / Provider Seam，四步实施计划）；调研证据存档在 docs/gaea3-review/（只读参考，权威性以设计文档为准）。阶段 7（v2.34-2.37 正确性纵深）先行，3.0 Step 0-3 在其后启动。

- 最新发布：**v2.40.0（2026-08-15）「3.0 架构主线 · Wave 4：Step 3 收官」（2 子代理并行 + 父代理集成）**：
  - semantic_search 工具注册（89d9fae）：决策纳入——实现完整且有 E2E 测试，能力面板「本地专业模型」
    分组声明了它；gaeaSpecialistTools 集中注册 ocr + semantic_search（原仅 ocr），ExtraTools 装配展开，
    TestSpecialistTools_Registered 断言两者防回归。
  - BalanceKind 贯通（89d9fae）：ProviderEntry 新增 `balance_kind`（可选，空=历史默认 deepseek 形状）；
    boot.NewProvider 透传 entry.BalanceKind → control.Options.BalanceKind → controller.Balance 改走
    billing.FetchByKind(kind,url,key)——切换余额后端只改配置、未知 kind fail-closed，补齐 Step 3d #8
    消费端贯通；render.go 渲染 balance_kind；config/control/boot 三层 7 测试（TOML 往返 / 自定义 kind
    路由 / 空 kind 默认 deepseek / 无 url (nil,nil) / 未知 kind fail-closed / boot 全链路）。
  - ModuleLauncher 清单化（4a6033a）：新增 boards/launcher.ts 纯函数（deriveLauncherModules +
    LAUNCHER_DESC）；ModuleLauncher 改 useSyncExternalStore(subscribeBoards, getActiveBoards) 订阅活动
    清单，删除静态 canonicalBoards 引用——后端合并清单（含 knowledge）变化后首页启动器自动跟随；
    launcher.test.ts 7 用例 + manifests.test.ts 36（43/43 过）。
  - 验证：go build/vet 干净 + test-all.ps1 **110/110 包** + 前端 tsc/eslint 0 errors + vite build 17s +
    vitest 427 过（27 jsdom localStorage 基线失败零回归）+ TestBindingsCompleteness PASS（464）+ 冒烟
    /api/health 200。发布 gaea-v2.40.0.exe（33.1MB，SHA256=f7c5fd1bc1859b025a742dcb78b26065a5718d8aa4374ef5c8cd90d7aaaff317）。
  - **Step 0-3 全部落地**：3.0.0 发布条件仅剩统一回归与版本发布；pickHerdsmanModel 能力标签挑选保留。
- 最新发布：**v2.39.0（2026-08-15）「3.0 架构主线 · Wave 3」（Step 3b LLM + Step 3c OCR/ASR/TTS + Step 3d 分类统一与 8 处注册表化 + 前端 GetBoardManifests 接线，4 子代理实现 + 父代理集成）**：
  - Step 3b LLM Seam（d183af7）：LLMProvider{Provider;Chat} + ChatFromStream 聚合 + streamChatAdapter 自动适配；
    bridge 互斥自注册（LLMKindWubigrok，DefaultLLMKind=wubigrok 空 kind 缺省=现状）；boot.NewProvider 经 NewLLM
    （gaea.toml providers[].kind 驱动 + fail-closed）；agent 聊天/子代理/Plan/judge/compact 只依赖 seam 接口；
    herdsman/ollama 思考模式分支测试冻结；19 新测试。
  - Step 3c OCR/ASR/TTS Seam（078ce1d）：OCRProvider（ovis 常驻探测冷却/tesseract，GAEA_OCR_ENGINE auto=ovis→
    tesseract 显式单引擎 fail-closed）+ TTSProvider（edge/sapi/herdsman 工厂四模式/xai 自注册，TTSSpeakBase64
    四级回退与 TTSSpeakStreaming 合成器链注册表化 TTSChain 首个成功即赢）+ ASRProvider（herdsman，voice.Manager
    SetASRProvider 接口注入弃回调注入）；isSTTModel 委托 modelengine.ClassifyModelByName，isTTSModelID 因 edge
    关键词口径差异保守保留；cosyvoice ensure 惰性保持。
  - Step 3d 分类统一 + 8 处注册表化（9a535c4）：modelengine 导出 ClassifyModelKind/ClassifyModelByName（六桶单源
    关键词表零变化）；websearch 6 引擎 SearchEngine 注册表（[search] engine_order 可配序）/embed·rerank
    EmbeddingProvider·RerankProvider（弃 HERDSMAN_BASE_URL）/vision Provider（删 GAEA_VISION_* 读 env 未知 kind
    fail-closed）/image_gen comfyui 常量/OCR 工具补注册 ExtraTools/markitdown MarkdownConverter（cli 两级回退）/
    billing FetchByKind（deepseek 形状默认）；6 注册表测试文件。
  - 前端接线（b1cc2fd）：loadBoardManifests 改调 wailsjs CoreB.GetBoardManifests（v2.38.0 wails build 生成），
    成功→normalizeManifests 合并替换（knowledge D7 菜单第 9 项 BookOutlined 补注册表/home 壳层 isHome 首位/
    weixin page 空为准/重叠后端优先），失败→fail-closed 回退静态；MainLayout subscribeBoards+useReducer；
    main.tsx 补注册 KnowledgePage；manifests.test.ts +16 用例（45/45）。
  - 父集成（048768c）：gaea.toml 新增 [retrieval]/[vision]/[markdown_converter] 段 + [search] engine_order
    （config.go 新结构体，零值=全默认本地端点）；boot/sysprompt.go 装配 SetSearchEngineOrder/SetRetrievalRuntime/
    SetVisionRuntime/SetMarkdownConverterRuntime；app 层 5 处 provider.New→provider.NewLLM。
  - 验证：go build/vet 干净 + test-all.ps1 **110/110 包** + 前端 tsc/eslint 0 errors + vite build 42.9s +
    vitest 420 过（27 基线失败零回归）+ TestBindingsCompleteness PASS（464 无新绑定）+ 冒烟 /api/health 200。
    发布 gaea-v2.39.0.exe（34.7MB，SHA256=fac19c50accc8310210bd768be51f1f519ba28cbce80d2a32a96ada271fb31fb）。
    提交：d183af7(3b)/078ce1d(3c)/9a535c4(3d)/b1cc2fd(前端)/048768c(集成)。
  - 遗留（Wave 4 候选）：semantic_search 工具定义未注册（#6 只含 OCR）；billing kind 未从 ProviderEntry 贯通；
    ModuleLauncher 仍静态 canonicalBoards；pickHerdsmanModel 能力标签挑选保留。回退保障硬要求不变。
- 最新发布：**v2.38.0（2026-08-15）「3.0 架构主线 · Wave 2」（Step 1 app 接线 + Step 2 板块 Manifest + Step 3a Image Seam，4 子代理核验 + 父代理收尾）**：
  - Step 1 app 层接线（事件日志「日志即真相」运行时闭环）：Resume→Restore（DetectLegacy 迁移→checkpoint+log tail
    重放）、Save→日志（saveEventMode 双写）、模型调用前 flush 检查点（FlushCheckpointFailClosed fail-closed，
    失败中止回合）、压缩→checkpoint（回合后 Snapshot 刷新压缩投影 + 已消费 seq）；session.log_format=legacy|event
    回退开关，legacy 零行为变化；新增 event_mode_test/controller_eventlog_test（含压缩后 checkpoint 可恢复）。
  - Step 2 板块 Manifest：board 包（Board 接口 + Manifest 16 字段对齐 §5.2 + Validate 缺陷 2 防复发）+ 10 板块
    canonical（9 业务 + knowledge D7）；module_registry FillFromManifests manifest 驱动 + Startup 装配失败显式
    slog.Error；GetBoardManifests 挂 App+CoreB（gen_bindings explicitOverrides，绑定面 464 方法 → 10 门面）；
    前端 PageRegistry（main.tsx registerPage 8 页）+ MainLayout 附 B #1~#12 全部清单化 + ModuleLauncher 清单驱动
    （补 memoryhub/characterlib 入口）+ events.ts（21 后端 + 4 前端常量 + subscribe 封装）+ bindingNames.ts 464
    重生成 + bridge.ts 双向守卫补 GetBoardManifests/CheckModuleIntegrity。**label 单一来源=菜单文案（用户决策）**：
    chat/imagegen/modelcenter 用 聊天/绘梦/模型中心，面包屑跟随菜单。
  - Step 3a Image Seam：RegisterImageBackend/NewImageBackend/ImageBackendKinds 注册表（openai 兼容 + comfyui 各自
    init 自注册；互斥注册 panic、未知 kind fail-closed）；generateImageXAI 走注册表（kind=openai）+ 401 刷新 token
    单次重试守卫（retried 防无限递归）+ imagine:content-moderated 友好提示；GenerateImage 分发不变（imageBackend
    实例优先，config 驱动选择零代码切换）。
  - 验证：go build/vet 干净 + test-all.ps1 **110/110 包**（首跑 6 包失败为并发抖动 + 状态文件残留，单独重跑全绿）+
    前端 tsc/eslint 0 errors（350 存量 warnings）+ vite build 15.7s + vitest 404 过（27 个 jsdom localStorage 基线
    失败与 v2.37.0 一致零回归）+ TestBindingsCompleteness PASS（464）+ check-bindings-drift OK + 冒烟 /api/health 200。
    发布 gaea-v2.38.0.exe（34.6MB，SHA256=a9eeb837462109f8aa599213558ce6b3bd798eafbd25783fefa2edb6ca0b29fa）。
    提交：f5ddf62(Step2后端)/a9254c8(Step1接线)/4b5af82(Step2前端)/2ba821b(label对齐)/9d9716d(Step3a)。
  - 遗留（Wave 3）：Step 3b LLM / 3c OCR-ASR-TTS / 3d 分类统一 8 处 + classifyModelKind 4 处；前端
    GetBoardManifests 绑定接线（wails build 后 wailsjs 生成后替换 loadBoardManifests 静态 fallback +
    normalize 板块差集 home/weixin/knowledge）；观察项 boot.go ctrlOpts 未传 LogFormat（生产消费者
    gaeaBuildController 已注入闭环，CLI/子代理宿主需自行注入）。回退保障硬要求不变。
- 最新发布：**v2.37.0（2026-08-15）「正确性纵深 · 收官」（阶段 7 第二~四刀 T7-2/T7-3/T7-4 + 3.0 Step 0/1，5 子代理并行）**：
  - T7-2 可见性收口（v2.35.0）：qrlogin/chatWebSearch/SaveConfig/LocalTranslate 吞错清零；成本进料截断
    6000 字 + 整批事务；测评参数钳制 + 基地址 engineMgr；token 明文清理 + 剧照 ID 哈希防穿越；41 测试。
  - T7-3 名实相符（v2.36.0）：PDF FlateDecode 压缩流还原 + OCR 单页容错 + OvisOCR2 4096/截断检测；语义检索
    按需 + search 上限 + BM25 缓存 + dashboard mtime 真实聚合 + WatchErr 回退轮询；约 33 测试。
  - T7-4 前端性能收尾（v2.37.0）：写路径静默清零 + 三态错误重试 + Transcript/MarkdownContent memo +
    Toast role + reconcileFinalAnswer 完整文本比较；41 用例。
  - Step 0 修债：office 模块注册 GaeaSend + MainBrainChat 全链路测试 8/8 + 版本源同步脚本（搭车）。
  - Step 1 会话事件日志（3.0 地基，机制层）：append-only 日志 + 投影 + checkpoint + 迁移 + 派生 API +
    GaeaHistory 黄金测试逐字节一致；session 67 测试；session.log_format 回退开关。运行时「日志即真相」
    app 层接线（Resume→Restore/Save→日志/压缩→checkpoint）留待 gen_bindings 阶段。
  - 验证：go build/vet 干净 + 逐包测试全绿（internal/app 仅 1 个既有 flaky whisper 测试 + docmd GBK 环境
    失败为基线）+ 前端 tsc/eslint 0 errors + vite build 通过 + 冒烟 /api/health 200。发布
    gaea-v2.37.0.exe（34.5MB，SHA256=37A56F54DF653E3D9E8A5751EA282CEB34BF5BBCA2672D26439BF7BAEBA7A62B）。
    提交：4dbba0c(Step0)/72fae6c(Step1)/0a9fb6f(T7-2)/d7934eb(T7-3)/5364281(T7-4)。
- 下一会话计划（2026-08-15 收官后）：Step 1 app 层接线（运行时「日志即真相」：Resume→Restore、Save→
  日志、压缩→checkpoint、模型调用前 flush，配合 gen_bindings）→ Step 2 板块 Manifest（board 包 + 9 板块 +
  MainLayout 清单化 + PageRegistry）→ Step 3 Provider Seam（Image/LLM/OCR/TTS）。回退保障硬要求不变。
- 最新发布：**v2.34.0（2026-08-15）「正确性纵深 · 并发正确性」（阶段 7 第一刀 T7-1，4 子代理并行：whisper/tasks/metrics/ai）**：
  - T7-1.1 轻语会话并发安全（internal/whisper + whisper_handler + app.go Shutdown）：三入口串行化
    （Orchestrator per-instance Mutex + LockTurn/UnlockTurn）；**CloneFullState 深拷贝**（修复浅拷贝快照
    与下一轮原地修改的 -race 实证竞态）；WorkingMemory/AssociationIndex/HabitsStore/ActiveRecall 加
    RWMutex；**forSession 读路径惰性写 map 改只读**（-race 实证写-写竞态）；persistStateSync（回合锁内
    快照 + persistMu 落库）+ drainAndPersistAll 挂 Shutdown（末轮先 drain 再 persist）；rhythm 包级
    计数器移入 Orchestrator 实例（Reset 只清自己的）；12 新测试（whisper_concurrency_test.go 8 +
    whisper_persist_concurrency_test.go 4）。
  - T7-1.2 任务调度器竞态修复（internal/gaea/tasks）：markTerminal 进度语义（set100 仅 succeeded，SQL
    CASE WHEN，fail/cancel 保留实际进度）；取消优先于 succeeded（handler 返回 nil 也判 cancelled）；
    Cancel 与出队原子化（WHERE status='queued' 条件 UPDATE + runNext 出队前先注册取消，消除「已出队未
    注册」窗口）；callHandler defer recover（handler panic→failed 不重试，worker 存活）；10 新测试，
    22/22 -race 全绿（跑两遍无抖动）。
  - T7-1.3 TCCA 指标聚合收敛（internal/gaea/context/metrics.go）：MergeChild 与 Report 同字段集（补
    CacheHitTokens/CacheMissTokens/BreakCount/CompactionCount 四条漏项）；merged 标记 check-and-set
    移入 child.mu 临界区 + 数据快照走 child.Report()（锁序 父→子→孙 无死锁）；ForkCount +1 每 child
    恰好一次（children 移除 + merged 标记防重）；6 新测试。
  - T7-1.4 AI 客户端状态与重试（internal/ai/client.go）：非流式 Chat 复用流式退避（连接/5xx 重试 1s/2s；
    401 仅 xAI 同函数内刷新重发一次 attempt=-1，不递归不占双槽）；activeEngineID/imageBackend/
    imageBackendType/token 加 RWMutex + GetToken single-flight（快路径 RLock + 写锁二次判空）；
    修 vet 错误（Sprintf %w→%v）；7 新测试（client_retry_test.go，含 httptest OIDC 桩）。
  - 验证：go build/vet 干净、scripts/test-all.ps1 **109/109 包 ok**；并发门禁 C：whisper/tasks/context/ai
    -race 全绿（-race 需 cgo：CGO_ENABLED=1 + CC=C:/msys64/ucrt64/bin/gcc.exe + PATH 前置 ucrt64/bin，
    否则 gcc cc1 静默失败）；前端零改动（tsc 0 errors、eslint 0 errors、359 存量 warnings 与基线一致）；
    TestBindingsCompleteness 兜底（无新绑定）。冒烟通过（/api/health 200）。发布 gaea-v2.34.0.exe
    （34.4MB，SHA256=beebccf7b9d2ab1703a704db64060b77cb7dad12d515b1af3f0c8102eb8a7a07）；
    releases 清理至 5 版（删 v2.29.0）。
  - 遗留（后续刀）：H6（LLM 失败状态回滚）另开任务；whisper 包级全局竞态（traceRing/affHistoryWindow/
    涌现追踪）建议按 ④ 同法迁移；L3（WhisperGetState/GetFacts/SetEngine 无锁）未动。
- 下一会话计划（2026-08-15 更新）：阶段 7 第二刀 **T7-2 可见性收口**（v2.35.0：qrlogin/chatWebSearch/
  SaveConfig/gaea_translate 吞错清零、成本进料与凭据（上限钳制/基地址/keep-warm Enabled/明文迁移删除/
  剧照路径清洗）、批量事务）→ T7-3 名实相符（v2.36.0）→ T7-4 前端性能收尾（v2.37.0）→ Step 0
  （office 模块补注册路由 GaeaSend + MainBrainChat 全链路测试 + 版本源同步脚本化）→ Step 1 事件日志。
  回退保障硬要求不变（每 Step 独立提交/旧数据只读兼容/二进制保留 5 版/运行时开关/验收含回退演练）。
- 最新发布：**v2.33.0（2026-08-14）「质量收敛 · 前端收敛」（阶段 6 第十刀 T6-10，贯穿收官）**：
  - T6-10.1 巨型文件拆分（8 个巨型文件全部收敛）：ChatPage.tsx 1022→370 行（pages/chat/{constants,types,
    utils}.ts + components/chat/{ChatComposer,ChatModeBar,ChatPersonaBar,MessageList,SuggestionCard,
    WelcomeScreen}.tsx + hooks/useChatStream/useChatTopics/useChatVoice/useCustomTemplates.ts）；
    ImageGenPage.tsx 911→310（hooks/useImageGenConfig/useImageGenHistory/useImageGenQueue +
    components/imagegen/meta.ts）；CapabilitiesPanel.tsx 803→178（capabilities/{ServersSection,
    SkillsSection,ToolsSection}.tsx + hooks/useCapabilitiesData.ts）；Composer.tsx 786→406（composer/
    7 组件 + hooks/useComposer{Attachments,Menus,Workspace}.ts）；mock.ts 1563→50（lib/mock/
    {chat,core,cost,memory,model,office,retrieval,settings,shared,state}.ts 10 文件，11 个 no-op 落实并注释）。
  - T6-10.2 any 清零：eslint.config.js no-explicit-any warn→error 进 CI（315→0，新增 any 即 lint 失败）；
    历史 as any 逃生口由类型化兼容层替代。
  - T6-10.3 绑定漂移检查恢复（双向）：scripts/gen_bindings 新增 -names 模式（只输出方法名稳定排序不写文件）；
    frontend/src/gaea/lib/bindingNames.ts（462 方法清单）；bridge.ts 类型级双向守卫
    _CheckAppBindingsHasNoStray + _CheckAppBindingsCoversAll（任一方向漂移 tsc 即红）；
    scripts/check-bindings-drift.ps1（对照 Go 面不一致 exit 1）+ CI backend job 新增步骤。
  - T6-10.4 mock 契约对齐：mock-contract-e5.test.ts RetrievalEvalRun 改契约校验（total=12 真实查询集、
    threshold=0.8、passed 与 recallAt10 自洽、expected/topHits 为 kind:name、首条锚点「打桩设备 台班价」），
    弃虚构 0.85；CostImportVisionPreview/CostCompare/UnifiedSearch 补结构断言。
  - T6-10.5 虚拟化与性能：Sidebar 会话列表改 react-window List 虚拟滚动（新依赖 react-window ^2.3.0，
    rowComponent + useMemo 行装配）+ 过滤防抖；CostLibraryView memo/useCallback/useMemo 化；新增
    hooks/useDebouncedValue（空串/空值即时同步、卸载清理）+ 6 测试，Composer 外 2 处消费。
  - T6-10.6 桥接归一：删除 frontend/src/api/bridge.ts（123 行旧代理）；initBridge 并入 gaea/lib/bridge.ts
    （+478）；wails.d.ts 同步收敛；新增绑定单处注册。
  - T6-10.7 测试补强：Sidebar.test +40/CostLibraryView.test +56/GraphView.test +47/ChatPage.test +26/
    CharacterLibEditor.test +8/BindSection.test +8/useDebouncedValue.test 新 6——vitest 354→361（80 文件）。
  - 验证：go build/vet 干净、go test ./... **109/109 包 ok**（test-all.ps1）、TestBindingsCompleteness PASS
    （462 方法，无绑定变更）、check-bindings-drift OK；tsc 0 errors、eslint 0 errors（359 存量 warnings，
    no-explicit-any=error 0 违反）、vitest **361/361**（80 文件）、vite build 14.46s；冒烟通过
    （/api/health 200）。发布 gaea-v2.33.0.exe（32.8MB，SHA256=8FADBB7385D794DB69842171D4F95E678849FA8481686F646A4BBA6E94F4E92F）；
    releases 清理至 5 版（删 v2.28.0）。注：exe 无版本资源为历史已知状态（windres 缺失）。
- v2.32.0（2026-08-14）「质量收敛 · 辅助合集·名实相符」（阶段 6 第九刀 T6-9，4 子代理并行：微信/OCR/配置+TTS/token）**：
  - T6-9.1 微信生命周期（channels/weixin/clawbot.go）：Stop 幂等（stopMu+stopCh 关闭即置 nil）、
    Start 重启（running.Swap 幂等+通道重建+sessionExpired 重置）；会话过期（errcode=-14）触发
    OnSessionExpired 回调后退出轮询（删 5 分钟空转），app 层注入 emit notice「微信助手 X 会话过期，
    请重新扫码绑定」；getUpdatesFn/notifyStartFn/notifyStopFn 测试注入点；3 测试。
  - T6-9.2 凭据与表治理：wxToken DPAPI（assistant/manager.go save() 落盘加密 dpapi: 前缀、Load 解密+
    旧明文一次性迁移、解密失败返回含 ID 明确错误；内存保持明文 List 回显不变）；weixin_* 4 死表
    SchemaV13 DROP（migrations 追加 V13）+ ClearStructuredData 移除；3 测试。
  - T6-9.3 OCR（office/docmd/ocr.go）：startOvisServer 改 proc.StartTracked（Job Object），超时
    KillTracked 杀树+同步 Wait（零孤儿）；OCRImageText 单图降级 tesseract；single_prompt.go:43 删
    「Windows 原生 OCR」（RapidOCR 真实存在保留）；ovisStartWait/ovisHealthy/ovisBuildCmd/
    tesseractLookPath/tesseractImage 注入点；4 测试。
  - T6-9.4 配置原子写（internal/config）：saveConfigFile（CreateTemp 同目录→Write→Sync→Chmod→
    Rename，失败清理保留原文件，renameFile 注入）；Load 损坏备份 .gaea_config.json.corrupt-<ts>；4 测试。
  - T6-9.5 CosyVoice 可配置（cosyvoice_dir/cosyvoice_port 键，默认 C:\AI\cosyvoice/8010，端口校验）；
    tts_service.go 写死常量全删（ttsCosyVoiceCmdFor/ttsURL 推导，ttsReady/startTTSService 方法化）；
    启动失败 1s/2s/4s 退避重试；4 测试。
  - T6-9.6 token 改 header：httpbridge tokenOK 删 ?token=（仅 Bearer/X-Gaea-Token）；前端
    runtimePolyfill 弃 EventSource 改 fetch 流式 SSE（parseSSEFrame/parseSSEStream 纯函数）；
    httpToken.ts 删 URL query；Go 3+前端 6 测试。
  - 验证：改动包全绿+vet 干净+TestBindingsCompleteness PASS（462 方法，无绑定变更）；tsc 0 errors、
    vitest **354/354**（79 文件）、eslint 0 errors（72 存量 warnings）、vite build 15.17s；
    全量 go test：hook/skill/tts 3 包 AV 锁 test.exe（单独重跑全绿，环境抖动先例）、docmd GBK 类失败
    （c426d3f 基线 worktree 复现同款，非回归）。发布 gaea-v2.32.0.exe（32.8MB，
    SHA256=CC8EEF7AF693B934BBDA629853F0BA3BFD9B5566B08DBD881FBD4B0FD01761B0）；
    releases 清理至 5 版（删 v2.27.0）。
- v2.31.0（2026-08-14）「质量收敛 · 记忆·生命周期与审计」（阶段 6 第八刀 T6-8，父代理 Go 实现 + 3 子代理并行：前端接线/索引截断/组件补测）**：
  - T6-8.1 dream 审计：决策成文 docs/DREAM_WRITE_POLICY.md（dream 不纳入 hardAskTools 逐条审批，
    后台异步 90s 无法等确认 + 显式路径即用户触发；补偿=全程审计）；SaveDreamFacts 签名改
    (source string, facts)（source=auto_dream|explicit），每次实际写入落 <userDir>/dream-audit.jsonl
    （ts/source/saved/names）+ DreamAuditEntries 读取入口；3 测试。
  - T6-8.2 facts 生命周期：归档超 90 天（memory.ArchivedRetention）硬删；sqliteBackend/fileBackend
    双后端 CleanupArchived（返回被删行含溯源字段）+ ListArchivedPaged(limit,offset)（钳制 [1,200]/默认 50）；
    新绑定 GaeaMemoryCleanupArchived/GaeaMemoryArchivedList（gen_bindings 462 方法 + 完备性 PASS）；
    清理逐条 slog + 溯源审计 purge-audit.jsonl（GAEA_DATA_ROOT 隔离测试）；前端归档 tab 清理按钮；
    memory 5 + app 2 测试。
  - T6-8.3 索引截断：memoryIndexBudget（3000 runes）→ memoryIndexBudgetBytes=4096（与 Block()
    4096 字节阈值同口径）；truncateIndexByLines 行边界截断 + markdown 链接保护（未闭合 [ 或
    ](url 回退整行舍弃）；只在 '\n' 处切 UTF-8 安全；6 测试函数。
  - T6-8.4 前端补测：GraphView 5 + WhisperMemoryLibrary 8 = 13 vitest（3d-force-graph 链式 stub
    vi.hoisted；vi.mock("../../lib/bridge") 注入数据；组件实现 0 改动）。
  - 验证：go build/vet 干净、go test ./...（app 15.5s ok；hook/skill 被 AV 锁 test.exe
    Access denied——历史绿+本刀未触碰，环境抖动）、TestBindingsCompleteness PASS；tsc 0 errors、
    vitest **348/348**（78 文件）、eslint 0 errors（38 存量 warnings）、vite build 23.84s；
    冒烟通过。发布 gaea-v2.31.0.exe（32.7MB，SHA256=35A8445738A6A1D7991F32338C39060F44B34948020467225029CF049940AA71）；
    releases 清理至 5 版（删 v2.26.0）。
- v2.30.0（2026-08-14）「质量收敛 · 小说·导出与原子性」（阶段 6 第七刀 T6-7，后端三任务并行 + 前端单任务）**：
  - T6-7.1 export 整改（internal/export）：读取失败 slog.Warn+FailedChapters 计数；作者 ProjectMeta.Author
    回退 gaea；EPUB/HTML 统一 markdownToHTML（删 chapterToHTML）；TXT/MD 世界观对齐；写前 MkdirAll；
    sanitizeFilename Windows 保留名（CON/PRN/AUX/NUL/COM1-9/LPT1-9 含 CON.txt）+尾点；AddSection 错误不丢弃；
    13 测试（失败分支测试沙箱无 SeCreateSymbolicLinkPrivilege 以 Skip 降级，逻辑保留）。
  - T6-7.2 生成中断与互斥（create_chapter_handler.go）：请求级 context.WithCancel（取消传播 ChatStream，
    双路径落盘部分正文）；新绑定 CancelCreateChapter(chapterNum, branch) bool 幂等（gen_bindings 460 方法）；
    按章节互斥（同章节拒绝、异章节并行）；未生成内容不写空文件；6 测试。
  - T6-7.3 落盘原子化（internal/project）：writeFileAtomic（CreateTemp→fsync→Rename）覆盖 writeJSON 全部
    JSON + WriteChapter/WriteChapterBranch/WriteWorldview；5 测试。
  - T6-7.4 模板占位符：create-chapter.json "5000"→{word_count}，substituteWordCount 精确替换；2 测试。
  - T6-7.5 前端：CreatePage 791→288 行（拆 chapterStreamTypes 判别联合/useChapterStream/5 组件/
    characterStatus 枚举单源）；停止生成按钮（CancelCreateChapter 接线，false 本地兜底）；cancelled 事件
    三路收尾保留部分正文；18 vitest。前端 novel 流式经 wailsjsCompat 直调 NovelB（不经 gaea bridge/mock），
    CancelCreateChapter 类型由 wails build 再生成（CreatePage 局部桥接待删）。
  - 验证：export/project/types/app 包 ok（app 24.3s，首跑瞬态 FAIL 复跑通过）、vet 干净、
    TestBindingsCompleteness PASS；tsc 0 errors、vitest **334/334**（76 文件）、eslint 0 errors（存量 warnings）；
    冒烟通过。发布 gaea-v2.30.0.exe（32.6MB，SHA256=232E661F3A7E20799EBAFE9251EB21D41D24E3FA1DFABCEFF8766C395738E38F）；
    releases 清理至 5 版（删 v2.25.0）。
- v2.29.0（2026-08-14）「质量收敛 · 模型中心·密钥与 UI」（阶段 6 第六刀 T6-6，后端三任务并行 + 前端单任务）**：
  - T6-6.1 refresh_token DPAPI：token.go Save 敏感字段经 secure.EncryptString 加密落盘（dpapi: 前缀）；
    Load 解密失败返回含字段名明确错误（不静默 nil）；旧明文自动一次性重写迁移（幂等）；非 Windows 降级
    round-trip 完整；3 测试。
  - T6-6.2 汇率配置化：usd_cny_rate 配置键（默认 7.2，saveSetters 拒绝 <=0/NaN/Inf）；新绑定
    GaeaGet/SetUsdCnyRate（gen_bindings 459 方法 + 完备性 PASS）；注入式缓存（启动注入 statsRecorder、
    双写即时生效、零 IO）；ModelPanel 汇率输入 + engines.ts 包装；4 测试。
  - T6-6.3 probe 告警：目录缺失改打印真实目录 filepath.Join(rootDir, name)；1 测试。
  - T6-6.4 UI 拆分：ModelCenterPage 顶层 useState 42→3（useEngine/Stats/Image/Voice/BindState 5 hooks
    下沉）；XAI_VOICES 单源（utils.tsx:163 仅 1 处定义）。
  - T6-6.5 竞态守卫：refreshLocalModels 请求序号 refreshSeq 过期丢弃 + 5s 定时器随 category 重置；2 测试。
  - 验证：auth/modelengine/config/herdsman/app 5 包 ok、vet 干净、TestBindingsCompleteness PASS；
    tsc 0 errors、vitest **316/316**（72 文件）、eslint 0 errors（762 存量 warnings）；冒烟通过。
    发布 gaea-v2.29.0.exe（32.6MB，SHA256=E30EEB105778B0DACDAA4283F022E408019E3C5FAFF28BD47F48A509E64273A6）；
    releases 清理至 5 版（删 v2.24.0）。
- v2.28.0（2026-08-14）「质量收敛 · 轻语·测试与可观测」（阶段 6 第五刀 T6-5，批 1 并行 + 批 2 串行）**：
  - T6-5.1 补测试：仅新增 9 测试文件 146 用例（emotion_fusion 18/18 函数 100%、memory_consolidator 98.5%/
    contradiction 99.3%/self_editor 98.5%/vector_store 96.5%/dispatch_router 96.3%/agent_loop_runner 92.3%/
    canon 全 100%/desktop 96.8%+）；暴露 3 缺陷记录未改：normalizePath %ENV% 展开不生效（os.ExpandEnv
    不支持 %VAR%）、mergeProhibitions 道歉过滤漏带引号"对不起"、self_editor log 裁剪后可重新增长至 200。
  - T6-5.3 异步写可观测：whisperWriteErrors 计数器（count/摘要/时间）+ MemoryWriteErrorSink 透传
    （llm_extract/json_parse/panic/persist 四 phase）；persist 系列函数改返回 error（errors.Join）；
    5 测试。读取走内部方法 whisperWriteErrorStats()，不新增绑定。
  - T6-5.4 成人模式决策成文：docs/ADULT_MODE.md（六节）；orchestrator.go:91 注释引用；**删除
    WhisperSetAdultMode 死接口**（前端零引用 + 静默忽略参数；gen_bindings 457 方法重生成 + 完备性 PASS）。
  - T6-5.5 db 收敛：GetDatabase → (*sql.DB, error)（12 repos 文件 58 调用点）；PRAGMA 单源（DSN 唯一）；
    V11 FTS 失败 slog.Error；4 测试。
  - T6-5.6 占位清理：删 4 文件（plan_document_intent/paper_card_companion/desktop_mode_policy/desktop_opening）。
  - 验证：internal/whisper 3 包 + internal/app 23.9s ok、vet 干净、TestBindingsCompleteness PASS；
    tsc 0 errors、vitest **312/312**、eslint 0 errors（存量 warnings）；冒烟通过。
    发布 gaea-v2.28.0.exe（32.6MB，SHA256=8109B9B8C28C5CF1B11B6A202B32BEF702819AD015203A3A046ADE3B26AD6734）；
    releases 清理至 5 版（删 v2.23.0）。
- v2.27.0（2026-08-14）「质量收敛 · 绘梦·链路真实生效」（阶段 6 第四刀 T6-4，后端+前端两阶段串行）**：
  - T6-4.1 取消真实生效：CancelImageGeneration 调 POST /interrupt 中断 ComfyUI 当前任务 + 本地取消标记
    拒绝排队提交（ComfyUI 无删除排队 API，/queue 仅查询——注释说明）；checkHistory ctx 化；取消幂等。
  - T6-4.2 flux 名实相符：txt2imgWorkflows 显式映射表（krea2/z-image-turbo/flux），未知模型中文错误
    （静默降级消除）；flux 真实工作流（UNETLoader flux1-schnell + DualCLIPLoader type=flux + ae + 4 步 KSampler）；
    img2img 白名单。
  - T6-4.3 历史图片可恢复：imageItem 增 file_path；前端分级存储（>200k base64 只存 path）、挂载时经
    GaeaAttachmentDataURL 回填、下载/剧照优先 file_path；预览按 mediaIsVideo 选元素（t2v webp 修复）。
  - T6-4.4 parseSize 严格解析（弃 Sscanf）钳制 64-2048；T6-4.5 findProcessByPort 改 exec.Command netstat
    参数数组 + 端口白名单 1-65535；T6-4.6 httpClient/pollInterval 可注入 + httptest 五链路（Go 31 用例）
    + 前端 media/historyMeta/queue 纯逻辑模块（19 用例）。
  - 验证：internal/ai 4.2s + internal/app 23s ok、vet 干净、TestBindingsCompleteness PASS（无绑定签名变化）；
    tsc 0 errors、vitest **312/312**（70 文件）、eslint 0 errors（761 存量 warnings）；冒烟通过。
    发布 gaea-v2.27.0.exe（32.6MB，SHA256=B00C9A9B327D34370116583FC5D48A64500C7575924CC7B1225D61BCC21E90C1）；
    releases 清理至 5 版（删 v2.22.0）。注意：flux 需 ComfyUI 装 flux1-schnell/t5xxl_fp8/clip_l/ae 模型。
- v2.26.0（2026-08-14）「质量收敛 · 对话·流可靠」（阶段 6 第三刀 T6-3，后端+前端两阶段串行）**：
  - T6-3.1 流订阅竞态与超时（ChatPage.tsx）：runID 一到即同一微任务注册 EventsOn（零异步间隙首帧不丢）；
    STREAM_SILENCE_TIMEOUT_MS=30s 无帧超时 → sending 复位+错误展示+finally；finish 幂等五路收尾
    （done/error/超时/启动拒绝/卸载）；12 组件测试（fake timers）。Wails 事件按事件名精确匹配，
    严格"先订阅后启动"不可实现——采用等价形态"订阅后立即注册+超时兜底"。
  - T6-3.2 落库错误透传（chat_service.go）：appendChatExchange 返回 error；流式落库失败 emit error
    终态而非 done；**ChatTopicsList/ChatMessagesList 签名改 ([]T, error)**（绑定签名变更，
    gen_bindings 458 方法 → 10 门面）；前端全部调用点 try/catch + LogFrontendError（不再静默 || []）。
  - T6-3.3 语音持久化：新绑定 ChatAppendMessages + chat.Store.AppendMessagesTx（单事务批量）；
    前端语音识别/回复落库（不走 ChatSend 无重复）；朗读 URL revoke（onended/onerror/play 失败/卸载）；
    打字循环取消标志（切话题/卸载中止）。
  - T6-3.4 迁移一次性：migrateLegacyTopics 持久化标记 gaea_chat_migration_v1（成功才写、失败记日志
    保留旧键、会话内 initRef 仅一次）；ChatImportTopic 改 ImportTopicTx 单事务（全成或全回滚）。
  - T6-3.5 导出转义：ChatTopicExportMarkdown 转义 Markdown 敏感字符（行首井号/反引号/尖括号/竖线）；
    sanitizeChatFilename 加固（CON/PRN/AUX/NUL/COM1-9/LPT1-9 加前缀、尾点剔除、截断 40、空→chat）；
    ChatGeneral 不删除（module_bindings.go:16 主脑派发依赖，补注释）。
  - 验证：internal/app 22.2s ok + internal/chat ok + vet 干净；tsc 0 errors、vitest **290/290**（67 文件）、
    eslint 0 errors（762 存量 warnings）；冒烟通过。注：test-all.ps1 本刀多次被环境中断（AV 锁，done=30），
    以改动面+vet+前端门禁+v2.25 基线 109/109 叠加为准。发布 gaea-v2.26.0.exe（32.6MB，
    SHA256=9CBF876AC4C35E33A8C31FEAD73DB97FBFBA43E45FD893A5B96F1A1D73E40E73）；releases 清理至 5 版（删 v2.21.0）。
- v2.25.0（2026-08-14）「质量收敛 · 办公引擎·正确性」（阶段 6 第二刀 T6-2，4+2 子代理分批并行）**：
  - T6-2.1 PDF 页数/分页修复（docmd：countPDFPages 精确匹配排除 /Type /Pages；BT-ET 按页对象归类；
    ocrPDFRange 绝对页码 first+i 修复错位一页；pdf_pages_test.go 7 测试含构造 PDF fixture）
  - T6-2.2 TurnResult 语义（agent_run.go：blocked/precheck blocked/suppressed/tool panic 计入 Errors、
    Success=len(Errors)==0；终止流错误先写部分文本入会话；step-- 下限 0；10 测试；上层兼容确认——
    controller 丢弃 TurnResult、frontend 无引用）
  - T6-2.3 TCCA/evidence 补测（仅新增 6 测试文件 58 函数：context 覆盖 97.0%、evidence 91.1%；
    观察记录未改：MergeChild ForkCount +1 语义存疑、CacheReport 不聚合子项 CacheHitTokens 等）
  - T6-2.4 后端看门狗（watchdog.go：墙钟 10min/停滞 30s 默认、Options.Watchdog 可配（==0 默认/<0 禁用）、
    触发走该回合 cancel→Emit TurnDone(Err)+Notice、watchdogSink 推进观察（工具在途 ToolDispatch→ToolResult
    与审批/提问等待豁免停滞）；8+3 测试；v2.13.0 声称此前未落地已注释对齐；headless Run 路径不做）
  - T6-2.5 Send 队列 + 禁写注册表化（controller：running 期 Send 限长队列 8 条 FIFO 排空、队满拒绝 notice；
    tool：PersistWriteTool 接口 + Registry.PersistWriteNames()，task.go 删除手写 subagentForbiddenWrites，
    6 工具各加标记，集合与 hardAskTools 一致测试断言；8 新增 + 2 更新测试）
  - T6-2.6 docmd.go 拆分（1521→56 行 → office.go/pdf.go/ocr.go/pagespec.go 纯搬迁字节级验证，
    23 测试全绿、覆盖 39.2% 与拆分前一致——OCR/外部工具路径无环境跳过为既有状态）
  - 验证：Go 全量 **109/109 包 ok**、vet 干净；tsc 0 errors；eslint/vitest 与 v2.24.0 一致。
    发布 gaea-v2.25.0.exe（32.6MB，SHA256=72806071E71471EA594F9109140474CCB42C8E0F7E9BC4C9DFCC6A9E3FFF833C）；
    冒烟通过（/api/health 200）；releases 清理至 5 版（删 v2.20.1）。
- v2.24.0（2026-08-14）「质量收敛 · 基础层·可靠性」（阶段 6 第一刀 T6-1，3 子代理并行）**：
  - **T6-1.1 SSE 流式加固**（internal/ai/client.go）：bufio.Scanner 64KB 行上限改 bufio.Reader 任意长行
    （1.2MB 单行实测不断流）；doStreamRequest 连接错误/5xx 指数退避重试（默认 2 次 1s/2s，200 流开始后
    不重试，401 刷新、include_usage 400 降级整体重试）；空闲超时 60s（idleTimeoutBody 每次读重置计时，
    cancel 作用于 streamCtx 解除阻塞读——坑：超时后解析协程 send 守卫必须用调用方 ctx，否则丢错误块）；
    代理接入：httpClient 改 netclient.NewHTTPClient（与 web_fetch/web_search 同源 gaea 配置
    NetworkProxySpec），proxySpec() 强制 localhost/127.0.0.1/::1 回环直连（herdsman/ComfyUI 不走代理）；
    新增 client_reliability_test.go 8 测试（长行/连接失败重试/5xx 重试/耗尽/空闲超时/本地直连/云端代理/spec）。
  - **T6-1.2 前端错误可见性**（frontend/src/gaea/lib/bridge.ts、store.ts）：BridgeError（code/message 可枚举，
    继承 Error 兼容 17 处消费点）+ normalizeError + invoke 统一入口 + logFrontendError；app proxy 全部方法包
    invoke 层（LogFrontendError 防递归）；store.ts 8 集群 14 处静默 catch 改 logBridgeError（状态逻辑零改动）；
    bridge.test.ts 6 用例 + store.test.ts +3。useController 返回块 ~30 处"预期降级"catch 未逐一替换
    （bridge 层已统一记录，全量收敛留 T6-10）。
  - **T6-1.3 后端吞错清理 + 日志脱敏**：ChatTopicsList/ChatMessagesList 读错 slog.Error（绑定签名未改，
    变更留 T6-3）；whisper_handler 两处 _= 记日志；memory_ingest/memory_consolidator LLM/解析失败 slog.Error；
    config.Load 坏 JSON slog.Error（签名未改，原子写留 T6-9.4）；main.go 桥接 token 日志脱敏 maskToken
    （尾 4 位）；全库 _= 扫描补 task_plan_store/characterlib_handler/gaea_ui 三组；新增 13 测试
    （app 6/whisper 5/config 1/main 1，slog 捕获 handler 断言）。
  - 验证：Go 全量 **109/109 包 ok**、vet 干净；前端 tsc 0 errors、eslint 0 errors（749 存量 warnings）、
    vitest **274/274**（65 文件）；冒烟通过（/api/health 200）。发布 gaea-v2.24.0.exe（32.6MB，
    SHA256=5371DE13979BE2B2CC4EB1A6D471D436207D24C1E7587861F7FCD431C243A540）；releases 清理至 5 版
    （删 v2.20.0）。
  - 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md（十刀十版本）。
- v2.23.0（2026-08-14）「运行纵深 · 进料与质量」（阶段 5 第三刀 T5-5+T5-6，5 子代理并行）**：
  - **T5-5 成本库进料闭环**：GaeaCostImportVisionPreview（internal/app/gaea_cost_import_vision.go，
    新文件）——PDF 走 docmd.ConvertLimit 文本提取（扫描件自动转 docmd.OCRImageText 本地 OCR：
    OvisOCR2→Windows OCR 兜底）；图片（png/jpg/jpeg/webp/bmp）直接本地 OCR；文本→表格线启发式
    解析（表头含名称/规格/单位/价格；TSV/竖线/多空格对齐），回退整行解析（名称=行首+价格=行内
    首数字，含序号剥离/页码噪音过滤）；AI 字段归一化走 routeSensitiveLocal 本地通道（provider
    wubigrok），不可用静默降级规则解析并 message 注明；app 层 CostImportPreview 增 Source 字段
    （json:source，pdf_text/pdf_scan/image）；复用 costimport.MatchRows 匹配 + GaeaCostImportApply
    落地（无确认不落库）；GaeaCostCompare 三源比价（cost_entries 现价 kind=current /
    price_fetch 候选 kind=fetch / cost_price_history 历史 kind=history，diffPct 复用
    DetectAnomalies 单期跳幅算法，按 fetchedAt 倒序，无匹配空数组）；前端 CostImportModal 扩展名
    路由（.pdf/.png/.jpg/.jpeg/.webp → CostImportVisionPreview）+ 识别来源中文标注 + CostCompareModal
    比价弹层（跳幅着色 ≥20% 红/>5% 琥珀，空态「暂无其他来源」）。
  - **T5-6 检索统一与质量回归**：GaeaUnifiedSearch（internal/app/gaea_unified_search.go 新文件）
    一次返回 keyword（workspaceSearchHits，抽取自 GaeaWorkspaceSearch）+ semantic（semanticSearchHits，
    抽取自 GaeaSemanticSearch，已跨 cost/knowledge/office/file）两组；原两绑定收敛为一行委托
    （单一实现来源，行为零变化）；前端 WorkspaceSearchPanel「跨库」开关两段展示；GaeaRetrievalEvalRun
    （internal/app/gaea_retrieval_eval.go 新文件）——查询集 docs/retrieval-eval-set.md（12 条
    造价/工程域查询 + 19 个预期命中，```json 块解析，GAEA_RETRIEVAL_EVAL_SET 环境变量可覆盖
    路径），逐条 GaeaSemanticSearch 取前 10 → recall（kind:name 精确或子串）→ 平均 recall@10、
    threshold=0.8、passed；模型中心「检索质量」Tab（RetrievalEvalSection.tsx，KPI+逐查询明细表）。
  - 验证：检索测评 4 + 统一检索 3 + 识别/比价多组 + 既有检索回归 8/8；Go 全量 **90/90 包 ok**、
    vet 干净；gen_bindings **457 方法** → 10 门面 + 完备性；前端 tsc 0 errors、eslint 0 errors
    （752 存量 warnings）、vitest **265/265**（64 文件）；冒烟通过；发布 gaea-v2.23.0.exe
    （SHA256=2A1F1505...）；releases 清理至 5 版。
  - **本刀并行要点**：5 个子代理（E1 识别比价后端/E2 导入与比价组件/E3 统一检索/E4 测评/E5
    契约层+搜索入口+测评 UI）；契约层仍 E5 独占；eslint 曾 1 error（mock.ts 无用转义 [\/]→[/]）；
    E3 建议的两方法收敛委托已采纳；MemoryHubPage 搜索为四路合并超集，未接入 UnifiedSearch（不重复）。
- v2.22.0（2026-08-14）「运行纵深 · 速度与韧性」（阶段 5 第二刀 T5-3+T5-4）：
  - **T5-3 本地模型调度纵深**（internal/app/gaea_schedule.go，新文件）：
    ① 保活 keep-warm：每 5 分钟对 HerdsmanModelCatalog 中 Running=true 的模型发轻量 SSE 探针
    （POST /v1/chat/completions，max_tokens=8，15s 超时，复用 herdsmanBenchHTTP/v1Join）；探针失败
    记日志降级跳过该模型直至下轮 catalog 重新 Running；local_concurrency=1 下绝不主动 start；
    开关 `keep_warm_enabled`（~/.gaea_config.json，internal/config KeyKeepWarm，默认 true）；
    core.GetKeepWarm/SetKeepWarm 绑定；模型中心「本地调度」设置区（SchedulingSection.tsx）；
    ② 启动自动预载：Startup 后延迟 10s，按功能绑定（gaea→office→chat 优先级）取第一个
    engine==herdsman 的模型，catalog 判定 Installed 且 !Running 时 HerdsmanModelStart（--wait，
    后台 goroutine 不阻塞启动）；只预载一个（互踢）；开关 `auto_preload`（KeyAutoPreload 默认 true）；
    core.GetPreloadPlan/SetPreloadPlan；
    ③ 换模预计等待：GaeaModelSwitchEstimate(engineID)——非 herdsman hot/1s；herdsman 按
    catalog（Installed/Running）→ hot/cold（wait 20s，实测冷启动 15.2s）/download/unknown；
    estimateModelSwitch(installed, running) 纯函数；前端 ModelSwitcher 对 herdsman 弹确认；
    ④ KV 缓存命中率 KPI：modelengine.ModelCallUsage/ModelUsageStats/ModelStatsSummary 增
    CacheHitTokens/CacheMissTokens；ai/client 两处 RecordCall 上报（cacheSplitForUsage 归一：
    DeepSeek prompt_cache_hit_tokens 优先、OpenAI prompt_tokens_details.cached_tokens 次之；
    miss 优先服务端显式值否则 prompt-hit 推算；服务端完全未上报时返回 0/0 不污染命中率）；
    UsageOverview 增 cacheHitTokens/cacheMissTokens/cacheHitRate（**最终落地为 snake_case
    json tag：cache_hit_tokens/cache_miss_tokens/cache_hit_rate**——与 engines.ts 统计类型
    （ModelStatsSummary/UsageSide/UsageOverview）的 snake_case 风格一致；StatsSection 内联
    断言也用 snake_case；曾误写 camelCase 已纠正，运行时三层命名必须一致）；StatsSection
    「本地 vs 云端」卡新增全局/云端/本地命中率（KpiTile）。
  - **T5-4 中断续跑**：session sidecar（internal/gaea/agent/session/state.go：
    <session>.state.json，SessionState{Running,Summary,UpdatedAt}，AtomicWrite；StatePath/
    LoadState/SaveState/ClearState）；controller.runTurnWithRaw 回合开始写 Running=true、
    defer 结束写 Running=false+最后助手文本摘要（240 字截断）；进程被杀残留 running=true；
    SessionMeta.Interrupted + Sidebar「未完成」琥珀徽标（D3 子代理）；GaeaResumeSession 恢复时
    注入「上次会话中断于 <摘要>，请先总结进度再继续」system 消息并 ClearState（含
    resumeLastSession 启动自动恢复路径）；轻语任务计划持久化（internal/whisper/task_plan_store.go：
    dataRoot + NewTaskPlanStoreWithDataRoot + Save 原子落盘 whisper_data/task_plan.json +
    Clear + ReloadFromDisk + ActivePlan/Resume；internal/app/whisper_taskplan.go：进程级惰性装配 +
    WhisperTaskPlanStatus/Resume 绑定）。
  - 验证：45 个新 Go 用例；全量 90/90 包 ok、vet 干净；gen_bindings 453 方法 → 10 门面
    （D4/D6 共 7 个新绑定统一由父代理重生成）；前端 tsc 0 errors、eslint 0 errors（751 存量
    warnings）、vitest 255/255；冒烟通过；发布 gaea-v2.22.0.exe（SHA256=301E3CF2...）；
    releases 清理至 5 版。
  - **并行实施教训（本刀）**：7 个子代理并行，契约层文件（types.ts/bridge.ts/mock.ts）必须
    指定单一负责人避免并发写覆盖；gen_bindings 必须由父代理在所有后端代理完成后统一执行；
    Go json tag 风格先确认目标文件惯例（同一项目内 snake_case 与 camelCase 并存）；
    前端内联类型断言过渡要标记「契约落盘后可移除」。
- v2.21.0（2026-08-14）「运行纵深 · 调度与异步化」（阶段 5 第一刀 T5-1+T5-2）：
  - **T5-1 通用任务调度器**：internal/gaea/tasks（Hephaestus.db SchemaV8 tasks 表；状态机
    queued→running→succeeded|failed|cancelled、进度 0-100+消息、取消（context 传播，running 中断/
    queued 直接取消）、自动重试（指数退避默认 2 次）、手动重试（Retry 清零重排队）、**重启续跑**
    （Manager.Start 恢复 running→queued）；进度事件经 **gaea-task** 事件通道实时推送（节流 400ms，
    终态必达））；App 绑定 GaeaTaskList/GaeaTaskCancel/GaeaTaskRetry（446 方法→10 门面，gen_bindings
    重新生成 + TestBindingsCompleteness）；
  - **消费者**：价格抓取全异步化（GaeaPriceFetch/GaeaPriceFetchAll 提交任务立即返回；30 分钟 cron
    走任务队列 + 到期过滤 + 活动任务去重；同源抓取去重；失败明细在任务结果/消息；修复 SaveFetch
    按值拷贝致返回记录 ID 恒为空的历史缺陷——handler 预生成 fetch-<ns> ID）；文件索引重建异步化
    （分批 Ensure 报告进度、末批 Stale；手动/轮询/监听共用队列去重）；
  - **T5-2 实时文件监听**：internal/gaea/filewatch（fsnotify 监听工作区，2s 去抖合并输出 changed/
    removed 批次；目录级变更与事件风暴>50 标记 Full→全量重建任务；WatchErr 记录、调用方回退轮询）；
    增量索引（删除→semantic.Store.Remove 直接清向量；新增/修改→Extract+Ensure 内容感知重嵌；
    失败自愈全量重建）；10 分钟轮询降级兜底（watch.Healthy 时跳过）；
  - **前端**：办公右栏新增「任务」Tab（TaskCenter.tsx：活动/历史分组、进度条、取消/重试、失败原因、
    重试次数，onTaskEvent 实时增量）；PriceSourcesPanel/WorkspaceSearchPanel 异步化（提交任务→
    事件终态结算）；bridge.ts AppBindings + gaeaToGaea + onTaskEvent；mock.ts taskView/mockTaskSubscribe；
  - **验证**：tasks 包 13 测试、filewatch 5 测试、App 层 6 测试（单源任务流/同源去重/一键抓取/
    List-Cancel-Retry 链路/定时到期跳过/cron 去重）；Go 全量 **90/90 包 ok**、vet 干净；
    前端 tsc 0 errors、eslint 0 errors（749 存量 warnings）、vitest **253/253**（61 文件）；
    冒烟通过（/api/health 200）；发布 gaea-v2.21.0.exe（SHA256=15B599EA...）；releases 清理至 5 版。
  - 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段5-运行纵深.md；发布说明 releases/v2.21.0.md。
  - **沙箱注意**：npx/npm 的 .ps1 被执行策略拦截——一律用 `& 'C:\Program Files\nodejs\npx.cmd'` /
    `npm.cmd`（AGENTS 旧备忘写 npx 用 npx.cmd 已核实）；tsc 直接 `node node_modules/typescript/bin/tsc -b`。
- v2.20.1（2026-08-14）「数据可迁移·独立审查修复」：
  - 对 v2.20.0 变更面做独立子代理代码审查（data_backup_review.md），修复 3 高危 + 4 中危 + 9 低危：
    #1 ApplyPending 两阶段幂等（部分失败重试可成功）；#2 home 配置恢复（HomeConfigRel 需带 . 点前缀，
    否则恢复到错误文件名）；#3 SQLite 快照加 _busy_timeout=5000 + 重试，回退改 checkpoint 后复制，
    manifest 增 Warnings；#4 失败路径也写 .restore-result.json；#5 已有 pending 拒绝再次 Restore +
    staging 随机后缀 + Cancel 清理孤儿；#6 dirSize 缓存（mtime+TTL）；#7 GaeaDataBackupRollback 回滚；
    #8 盘符路径拒绝；#9 恢复二次确认 + 只收 .zip；#10 备份文件名毫秒+随机；#11 before 保留 2 份；
    #12 WritePending 原子写；#13 Extract 两阶段；#15 shouldSkip 精确化；#16 pending 错误透出；#17 entries 数组校验；
  - 验证：backup 包 9 测试（含重试幂等/home 配置/盘符拒绝）、App 层 5 测试（含已有 pending 拒绝/Rollback）、
    go 全量 ok、tsc/eslint 0 errors、vitest 251/251、真实 E2E（二次恢复拒绝、重启自动应用、home 配置恢复）。
  发布产物 gaea-v2.20.1.exe（SHA256=F7347E7049C9A7A30AC4484F402A4566C3B995EF0FC893B637145E59CB8AC9BA）。
- v2.20.0（2026-08-14）「个人使用收口·数据可迁移」（长期规划阶段 4，P4-1~P4-4）：
  - **产品定位（2026-08-14 用户决策）**：gaea 个人使用、不商用。阶段 4 不再做产品化分发
    （安装器/自动更新/代码签名/SmartScreen 等商用项全部删除）；发布形态 = exe + SHA256SUMS +
    冒烟 + 升级文档；升级 = 替换 exe（数据在用户目录，不动）。
  - P4-3 数据可迁移：internal/gaea/backup/backup.go（Plan/Create/Extract/ReadManifest/
    WritePending/ApplyPending/ClearPending；SQLite 用 VACUUM INTO 一致性快照，运行中备份安全，
    WAL 自动合并；zip-slip 防穿越；Skip 规则 = 段相等/前缀/后缀，-wal/-shm/gaea.log 自动排除）；
    internal/app/gaea_data_backup.go（GaeaDataBackupInfo/Create/Restore/Pending/Cancel/
    RestoreResult，恢复两阶段：staging + pending 标记 -> 重启后 Startup applyPendingRestore 应用，
    应用前自动备份当前数据到 .restore-before-<时间>）；设置页新增「数据」分类
    components/settings/DataPanel.tsx；
  - P4-1 模块收口：微信通道标注 beta（CharacterLibEditor）、移动端标注「已冻结」（SettingsMobile）；
  - P4-4 磁盘治理：releases 保留最近 5 版 exe 约定（releases/README.md），旧 exe 已清理；
  - 验证：Go backup 6 测试 + App 3 测试 + internal/app 全量 ok（29.97s）、vet 干净、
    gen_bindings 重新生成（442 方法 -> 10 门面）、tsc/eslint 0 errors、vitest 251/251、
    真实 E2E（备份 zip 无 wal/shm、恢复->重启自动应用->before 目录生成、restore-result 可读）。
  发布产物 gaea-v2.20.0.exe（SHA256=49B7234886A8CCA56080C1D6812D5C73AB6BAE05F9CAFAE330D594CFED2FBF92）。
  详见 releases/v2.20.0.md。
- v2.19.0（2026-08-14）「数据与成本纵深·补测评缺口」（长期规划阶段 3，D3-4）：
  - D3-4 报告专项分析：renderBenchmarkReport 新增每模型对比/长上下文专项（TTFT vs
    context_size）/缓存复用专项（first vs second TTFT + prefill 加速比）/显存相关启动参数
    （effective_launch_params：gpu_layers/no_kv_offload/batch/ubatch/cache_type 等）/
    并发专项说明；
  - D3-4 压力预设：受控测评任务集新增 压力·长上下文 / 压力·长输出 / 压力·显存 3 项
    （配合上下文 4K~32K、并发 1/2/4、max_tokens 组合成压力矩阵）；
  - D3-4 流式探针：`GaeaBenchmarkStreamProbe(model)` 直连 herdsman /v1/chat/completions
    SSE，观测 TTFT/分块数/max_gap（卡顿）/是否 [DONE] 收尾（断流）；模型中心「受控测评」
    新增「快速流式探针」区。真实端到端验证过（冷启动 TTFT 15.2s、60 块、max_gap 83ms）。
  验证：go 全量 ok（26.5s）、vet 干净、tsc/eslint 0 errors、vitest 247/247、冒烟通过。
  发布产物 gaea-v2.19.0.exe（SHA256=CB338574...）。详见 releases/v2.19.0.md。
  **沙箱教训**：同一 exe 路径的第二个实例会因 WebView2 用户数据目录占用启动失败
  （8007139f）——本机常驻 gaea 时，E2E/冒烟请用 releases 副本路径或临时副本。
- v2.18.0（2026-08-14）「数据与成本纵深·首轮」（长期规划阶段 3，D3-1 ~ D3-3）：
  - D3-1 持久化向量索引：GaeaSemanticSearch 跨库统一检索并入工作区资料（kind=file，复用
    文件索引 cron 维护的持久化向量，检索不扫描）；GaeaSemanticIndexStatus（各 kind 向量条数，
    semantic.Store.Counts()）；
  - D3-2 分流统计：GaeaUsageOverview 打通 gaea 调用记录（云端引擎费用估算）与 herdsman
    events.jsonl 本地遥测（本地=events 全量 + 其他本地引擎，herdsman 不重复计；节省按云端
    实际混合单价折算，无云端回退 deepseek-v4-flash ¥1.5/MTok）；模型中心「调用统计」新增
    「本地 vs 云端·节省对比」卡；
  - D3-3 测评产品化：复用 herdsman /api/benchmarks（GET 列表 / POST 发起，config 契约见
    model_benchmark/runs.json；明细含逐 case TTFT avg/p95、token、cached_tokens）——
    GaeaBenchmarkList/Start/Detail/Export + 模型中心「受控测评」分类（10 项任务预设蒸馏自
    120 组对照测评方法学、上下文 4K~32K、并发 1/2/4、Markdown 报告导出）。真实端到端验证过
    （POST 202 → 40s 完成 succeeded）。注意：herdsman 测评接口的 POST body 中文必须 UTF-8
    （PowerShell 测试曾因 GBK 转 ?，Go 客户端无此问题）。
  验证：go 全量 ok（internal/app 29s）、vet 干净、tsc/eslint 0 errors、vitest 246/246、
  冒烟通过。发布产物 gaea-v2.18.0.exe（SHA256=5AB1DC35...）。详见 releases/v2.18.0.md。
- v2.17.0（2026-08-14）「安全与架构收敛」（长期规划阶段 2，S2-1 ~ S2-4）：
  - S2-1 LAN 暴露全局告警横幅（SecurityBanner，启动检测 App.HerdsmanSecurityCheck）+ 设置页「安全」面板；
  - S2-2 WebView2 远程调试改 `GAEA_WEBVIEW_DEBUG=1` 才开启（默认关，此前 WebviewDisableRendererCodeIntegrity
    会连带开 9333）；HTTP 调试桥接加一次性 token（GAEA_HTTP_TOKEN 或自动生成打日志；/api/rpc 与
    /api/stream 须 Bearer/X-Gaea-Token/?token=，/api/health 开放；CORS 放行 Authorization）；
  - S2-3 App 绑定面拆分：429 个导出方法 → 10 个板块门面（CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/
    ChatB/NovelB/ImageB/CharlibB，internal/app/bindings_*.go，纯委托零逻辑改动；`scripts/gen_bindings`
    生成器 + `TestBindingsCompleteness` 反射完备性测试兜底；`app.NewBindings(a)` 供 main.go 绑定；
    前端 gaea/lib/bridge.ts realApp 按方法名路由门面、api/bridge.ts 补 window.go.app.App 兼容代理、
    旧 wailsjs 导入改指 src/wailsjsCompat 重导出）；
  - S2-4 敏感域本地化开关（`sensitive_local`，默认开，~/.gaea_config.json 持久化；成本/报价 AI 操作
    `GaeaCostImportAIParse` 走 routeSensitiveLocal 强制本地 Herdsman，不可用自动回退常规路由；
    App.GetSensitiveLocal/SetSensitiveLocal）。
  验证：go build/vet 全绿、internal/app 全量 19.8s ok、tsc -b/eslint 0 errors、vitest 243/243、
  vite build、冒烟 + token 端到端（401/200）。发布产物 gaea-v2.17.0.exe（32MB，
  SHA256=ADBFD953...）。详见 releases/v2.17.0.md。
- v2.16.1（2026-08-14）「E1-4 模型中心资源协同 + 磁盘治理」：
  生命周期操作串行化（herdsmanOpMu，对齐 herdsman local_concurrency=1）+ 模型库磁盘 KPI
  （installed_bytes/disk_total/disk_free，GetDiskFreeSpaceEx）+ fmtSize TB 档。
  全量 vitest 243/243；发布产物 gaea-v2.16.1.exe（32MB）冒烟通过。详见 releases/v2.16.1.md。
- v2.16.0（2026-08-14）「长期规划首轮：Herdsman 底座加固 + 工程门禁」：
  H0-1 环境探测（internal/herdsman/probe.go + App.HerdsmanProbe）、H0-2 健康检查（health.go + App.HerdsmanHealth）、
  H0-3 TTS 默认动态解析（voice.ResolveHerdsmanTTSModel，voxcpm2 优先）、H0-4 LAN 暴露告警（lancheck.go +
  App.HerdsmanSecurityCheck）、H0-5 模型用途提示上卡片 + 思考模式 max_tokens≥4096 守护；
  E1-1 前端 CI 修复（eslint 配置/插件、28 硬错误清零含 Lightbox 条件 Hook 隐患、CI 加 vitest）、
  E1-2 发布冒烟脚本 scripts/smoke.ps1、E1-3 周版本节奏。详见 releases/v2.16.0.md 与
  docs/superpowers/plans/2026-08-14-gaea长期规划-herdsman底座加固与工程门禁.md。
- 更早版本历史见 CHANGELOG.md / releases/README.md（版本表）。

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf → Markdown）、图表生成、报告拼装、
知识库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- **绑定面（v2.17.0 起）**：App 不再直接绑定 Wails；429 个导出方法拆 10 个板块门面
  （internal/app/bindings_*.go：CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB，
  纯委托零逻辑改动）。改绑定面方法后用 `go run ./scripts/gen_bindings` 重新生成 +
  `TestBindingsCompleteness` 兜底；前端调用经 gaea/lib/bridge.ts（按方法名路由门面）或
  api/bridge.ts 的 window.go.app.App 兼容代理；旧 wailsjs 导入走 src/wailsjsCompat 重导出；
  wails build 会重生成 wailsjs/go/app/<门面>.js
- 单模型架构：一个 executor 完成规划与执行，无独立规划器；任务/技能子代理走 `task` / `run_skill`
- 内置工具精简为 17 个核心工具（v2.4.3 起）：文件/命令、网络、任务、记忆/知识、技能、format_convert、chart_gen
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/office/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）
- 本地 AI 底座：**Herdsman**（localhost:8080/v1，~110GB 模型：35B 对话 ×2、zimage-turbo、voxcpm2、
  mineru、embedding/reranker、paddleocr、sherpa-onnx 等）；gaea 的聊天/视觉/检索/OCR/解析/ASR/TTS/生图/翻译
  全部依赖它，herdsman 升级可能破坏契约——用 App.HerdsmanProbe 启动探测

## 发布流程（2026-08-14 修订：补版本资源步骤）

1. 更新 CHANGELOG.md / README.md（版本表）/ wails.json（productVersion）/ releases/README.md（版本表）
2. **同步版本资源**：`build/windows/info.json` 是 Wails 生成版本信息的模板（fixed 段必须含
   `product_version`，否则 exe 的 ProductVersion 为 0.0.0.0）；根目录 `versioninfo.rc` 是遗留物，一并更新以免误导
3. 构建（本沙箱：`cd frontend; npm run build` → `wails build -s`；本机：`cmd /c build.bat`），
   产物 build/bin/gaea.exe（同时复制到桌面）
4. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`
5. 写 `releases/v<版本>.md` 发布说明（含 SHA256 与冒烟结果），更新 releases/README.md 版本表
6. 冒烟：`scripts/smoke.ps1 -ExePath releases\gaea-v<版本>.exe`（/api/health 200 即通过）
7. 更新 `.gaea/progress.md` 进度记忆与本文件（版本状态）

## 沙箱环境备忘（2026-08-14 整理，详细版见 docs/2026-08-14-sandbox-environment-notes.md）

**防止重蹈覆辙的四条铁律**：
1. `go telemetry off` 已持久生效；构建缓存写入问题随 danger-full-access 策略解除，无需再覆盖 GOCACHE
2. **wails build 前端编译会挂起**（wails 捕获前端输出走管道）——必须 `cd frontend && npm run build`
   再 `wails build -s`（-s = 跳过前端编译，9s 完成）
3. `go test ./...` 单进程树会被 harness 终止、个别测试二进制偶发 `fork/exec Access is denied`——
   逐包验证 + `scripts/test-all.ps1`（逐包/重试/状态续跑）；exec 拒绝用 `go test -c` 手动运行证明代码无恙
4. .ps1 脚本必须 UTF-8 带 BOM（否则 powershell.exe 按 GBK 解析报错）；npx 用 `& 'C:\Program Files\nodejs\npx.cmd'`

## 本地 TTS 引擎（重要记忆，勿遗忘；2026-08-09 整理）

> ⚠️ **VoxCPM2 已于 v2.6.9 移除**：实测不达标（耗时长、音色男女混乱、克隆不稳定）。
> 下方 VoxCPM/Vulkan 相关方法保留为「已废弃教训」，勿重新安装；当前本地 TTS 为 CosyVoice2。
> 注：herdsman 侧实测 voxcpm2 可用（冷启动约 50s，不支持预设音色），qwen3-tts-* 未安装。

本机（Radeon 8060S 核显 / 64GB 统一内存 / Windows）本地 TTS 有两条引擎线，gaea 只连 OpenAI 兼容 8020/8010。

### ~~VoxCPM2~~（已移除 v2.6.9，以下为废弃记录）

- `8030`：主后端 `C:\AI\llama-omni\build\bin\llama-tts-server.exe`（llama.cpp-omni，C++/ggml + Vulkan）
  - 模型：`C:\AI\llama-omni\models\VoxCPM2-BaseLM-Q8_0.gguf`（1.65GB）+ `VoxCPM2-Acoustic-F16.gguf`（1.74GB）
  - 8060S 识别 `KHR_coopmat + bf16`，全量 29 层 offload Vulkan0，加载约 2s
- `8021`：备胎 ROCm PyTorch（`C:\AI\voxcpm\server.py` + `VOXCPM_PORT=8021`）
- `8020`：适配层 `C:\AI\voxcpm\adapter.py`（FastAPI，gaea 入口，契约不变）
- 一键启动：`C:\AI\voxcpm\start_voxcpm_stack.ps1`（8030/8021/8020）

### CosyVoice2（端口 8010）

- `C:\AI\cosyvoice\server.py`：LLM 段 GGUF + Vulkan（`gguf\cosyvoice_f16.gguf`），flow 段 ONNX + DirectML（5 步）
- 启动：`C:\AI\cosyvoice\start_cosyvoice.bat`；约 14s 加载+预热，短句 ~1.5s

### 音色（两引擎统一 4 个，火山引擎 Speech-AI-Forge-spks 录音室样本）

- 中文女 `zh_female.wav`（f0≈221Hz）、中文男 `zh_male.wav`（f0≈133Hz）
- 英文女 `en_female.wav`（f0≈191Hz）、英文男 `en_male.wav`（f0≈109Hz）
- 参考音频 ~7s / 16kHz；转写在 `C:\AI\voxcpm\voices\_meta.json`

### 本次优化方法（AMD 核显提速教训，勿重蹈覆辙）

1. 不要再用纯 ROCm PyTorch 追赶速度：iGPU 共享内存架构下 ROCm 与 CPU 基本相同（RTF ≈1.06~1.12）；
   Vulkan + ggml 的 GEMM/coopmat 才是突破口（克隆 RTF 0.65~0.84）
2. 构建：MSYS2 UCRT64，`cmake -B build -DGGML_VULKAN=ON -DGGML_NATIVE=ON`
3. 坑 1（端口绑不上）：server-voxcpm2 会构造 SSLServer，空证书导致 is_valid_=false 任何端口 bind 失败；
   本地回环不需 TLS，改普通 httplib::Server
4. 坑 2（克隆近静音）：AudioVAE 参数特征必须 frame-major（`ggml_cont(latent)`），
   不能 `cont(transpose(latent))`（dim-major）
5. 坑 3：llama.cpp-omni 的 CLI `-r` 克隆偶发偏静音，HTTP server 路径正常；生产走 server
6. 坑 4：VoxCPM Python 长文本 CFG 2.0 会「静音+整段重试」（RTF 4.8~7.8），CFG 1.5 稳定；
   C++ server 端用 max_steps 限制解码上限
7. 网络：HuggingFace LFS 直连/hf-mirror 都不通，ModelScope 直链快（8.6MB/s）
8. 实测：短句克隆 RTF 0.65~0.84（6 步）、语音设计 0.57~0.60；同 seed 输出确定

### 详细记录

- `docs/2026-08-09-voxcpm2-integration.md`（VoxCPM2 全部历程）
- `docs/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`（CosyVoice GGUF 提速）

### 自动启动（当前仅 CosyVoice2）

- gaea 启动时后端 ensure cosyvoice；模型中心 TTS 模型卡片「启动」按钮 → `App.StartLocalTTSService(engineId)`；
  引擎连接测试兜底 ensure（等约 8s）；TTS 合成前兜底 ensure
- 实现：`internal/app/tts_service.go`（core.ensureLocalTTSService 幂等 + 异步轮询，emit `tts-service-status`；
  CosyVoice 直接 python server.py，隐藏窗口 CREATE_NO_WINDOW）
- 端口探测：CosyVoice2 `8010/v1/models`

## 已知注意

- 角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定
- 文生视频依赖本地 ComfyUI 安装 LTX-Video 模型
- 里程碑：2026-08-12 完成通用办公全面打磨（显示/布局/安全三线：成本库/记忆/知识库/技能写入全部硬性确认
  含 yolo、子代理路径；子代理不再继承持久化写入工具）
