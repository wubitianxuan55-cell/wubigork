# 任务进度

> 最后更新: 2026-08-30（首页重构「星枢指挥所」+ 两段式启动动画；跨事实因果
> 推断收口审计 §C 深度补口）

## 当前状态

- **最新发布：v4.9.0（2026-08-30）「星枢首页·轻语记忆纵深」**：基线 v4.8.3
  + 15 提交（轻语记忆回放系列 + 图谱三维度 + 跨事实因果推断 + 首页重构
  「星枢指挥所」+ 两段式启动动画 + 工程化收尾），绑定面 535→540；vitest
  818/818、drift PASS（540）、build.bat 冒烟 200（顺带修 pwsh→powershell
  回退）。欠账清单见 releases/v4.9.0.md；下一执行 = Realtime 真机验证轮。
- **首页重构「星枢指挥所」+ 启动动画（2026-08-30，v4.9.0）**：启动默认从首页
  落地（MainLayout 不再恢复上次页面，跨空间恢复保留）；两段式启动动画
  （index.html 静态启动屏 → BootSplash 组件：光环 + 徽记 + 分步状态 + 进度条，
  rAF 节流 / reduced-motion 全降级）；首页 Hero 中轴 + 命令条（orb/打字/语音/
  ⌘K）+ 真实遥测细条 + 能力矩阵 Bento（manifest 驱动，旗舰卡双类提权）+
  门廊 + 信息条；i18n 三语 home.*/boot.* 共 17 键。vitest 818/818、tsc/eslint 0、
  wails build 冒烟正常。说明：语音晶核收进命令条 orb；规格同步
  design-system/gaea/pages/home.md（v3）。
- **跨事实因果推断「为什么」（2026-08-30，审计 §C 深度补口）**：
  GaeaWhisperCausalExplain（绑定面 540，play）——问「为什么<entity>」时确定性
  收集证据（KG「导致」三元组 + event_chain 关联，上限 8 条），当前人格口吻 LLM
  解释因果链（只用证据 / 不编造 / 证据不足诚实说明 / ≤200 字）；无证据零 LLM
  调用回退文案。图谱面板「解释因果」按钮 + 结果区。Go +5 测试、vitest +1
  （818/818）、tsc/eslint 0、drift PASS（540）。说明：证据 = 确定性规则，LLM
  只做人话化；多跳因果图推理留后续。
- **关联入图（2026-08-30，审计 §C「推理仅邻接遍历」补口）**：
  GaeaWhisperGraphSubgraph 并入 AssocIndex 关联边——事实 Subject → 实体节点，
  类型映射中文（event_chain→因果、temporal→时间、thematic→主题…），权重 =
  strength，只并入与子图连通者；前端因果边琥珀色虚线 + 图例。Go +2 测试、
  vitest +1（817/817）、tsc/eslint 0、绑定面 539 不变。说明：event_chain 数据源
  = LLM 整合 + 冷启动启发式（同子类近似因果）；LLM 深度跨事实因果推断留后续。
- **图谱因果维度（2026-08-30，审计 §C 欠账收口）**：extractCausalTriples 从事实
  摘要确定性提取 {因, 导致, 果}（因为/由于…所以…、X导致/引发/造成Y、X让我/使我Y，
  正则 + 测试锁定，直接式剥离连接词），情绪随事实落图；ingest 与文档导入双路径
  接线，图谱面板「导致」谓词天然渲染。Go +6 测试、vitest 816/816 沿用（前端零
  改动）、tsc/eslint 0、绑定面 539 不变。观察项：关联表 event_chain 未在图谱
  展示；LLM 跨事实因果推断留后续设计。
- **图谱情绪维度（2026-08-30，审计 §C 欠账收口）**：Triple 增情绪三字段
  （EmotionLabel/Intensity/Valence），事实提取 attachEmotion 落图（效价派生
  正面/负面/中性），新增 AddTriple 保留情绪（Add 兼容旧调用）；BASIC_PROFILE
  主语实体化（档案键替代硬编码「用户」）；hermes.db 迁移 V13→V14
  （knowledge_triples +3 列，存量安全）；前端图谱边按情绪着色 + 图例。
  Go +5 测试、vitest +1（816/816）、tsc/eslint 0、绑定面 539 不变。欠账延续：
  因果维度未做（需 LLM/事件链）、时空索引未建（暂以 CreatedAt 承担）。
- **记忆重述 + 锚点刻度对齐（2026-08-30，记忆回放收尾两刀）**：
  GaeaWhisperMemoryRetell（绑定面 539，play）——LLM 以当前人格口吻把情节/锚点
  记忆重述成故事（输入复用确定性回放：摘要+情绪+原文对话；第一人称/称「你」/
  ≤300 字），episode/anchor 双入口 + 前端「让 gaea 重述这段记忆」按钮（MemoryRetell
  组件三态）；锚点策略阈值刻度对齐（weight≥2/selfRelevance≥4 在 0-1 抽取标尺上
  不可达 → 0.9/0.8/0.9，偏离 ackem 已注释），补策略测试 2 组。Go 全量绿、
  vitest 815/815、tsc/eslint 0、drift PASS（539）。
- **时间锚点「重访那一天」（2026-08-30，记忆回放续刀）**：写路径接线——
  ShouldWriteTemporalAnchor/BuildTemporalAnchor 此前有定义无生产调用（同类骨架
  欠账），现经 MemoryWritePayload/IngestTurnArgs 的 TemporalAnchorSink 逐事实
  评估（IsNew=计数差）→ Orchestrator.AddTemporalAnchor（回合锁内）落
  State.TemporalAnchors 随 companion_state 持久化；读路径 GaeaWhisperAnchors +
  GaeaWhisperAnchorReplay（锚点→事实轮次→情节→原文回放，绑定面 538）；前端记忆库
  「纪念日」tab + 回放弹窗（ReplayDialogue 与情节回放共用）。Go 8 新测试、vitest
  +2（813/813）、tsc/eslint 0、drift PASS。观察项：锚点策略阈值（weight≥2/
  selfRelevance≥4）与 LLM 抽取标尺（0-1）刻度不一致，常命中周期纪念日分支，
  刻度对齐留后续决策；LLM 重述未做（当前为确定性原文回放）。
- **轻语记忆回放（2026-08-30，审计 §C 欠账收口）**：新绑定
  GaeaWhisperEpisodeReplay（绑定面 535→536，MemoryB/play）——按情节 ID 从
  hermes.db 读情节 + SourceSessionID/[StartTurn,EndTurn] 从 chat_history 确定性
  重建原始对话（不调 LLM），过旧情节 Replayable=false 回退仅摘要；前端情节详情
  弹窗新增「回放原始对话」气泡视图（用户/gaea + 轮次，加载/失败/不可回放三态）。
  Go 3 测试 + vitest 2 测试；验证：Go 全量绿（whisper 偶发挂起为环境 flake，
  单跑绿）、vitest 811/811、tsc/eslint 0、drift PASS（536）。欠账延续：时间锚点
  「重访雨夜」入口、图谱情感/因果维度。
- **构建冒烟自动化（2026-08-30，v4.8.3 教训收口）**：build.bat 真实退出码 +
  产物新鲜度守卫（构建前删旧 exe、被常驻实例锁定时显式报错）+ 默认自动冒烟
  （.tmp 临时副本 → scripts/smoke.ps1，18999 /api/health 200 + status=ok 响应
  体校验，失败即停并提示勿发布）；`build.bat skip-smoke` 跳过（快速迭代）；
  smoke.ps1 增 Start-Process 失败显式报错、响应体校验、finally 判空回收。
  验证：默认路径实跑绿（构建 44.5s + 冒烟通过 + 进程回收）；失败路径假
  wails 桩实测 [FAIL] + exit 1。
- **最新小步（2026-08-30，欠账收尾）**：①VoiceStart realtime 门小修——
  端到端回复走服务端 response 事件（事件泵不经 whisperChatFn），realtime
  在位时不再要求 whisper 对话回调，whisperChatFn=nil 也可启动；拼接管线
  双门（ASRReady + WhisperReady）逐字节保留，新增两回归测试 ②持久化套件
  统一——desktop_session SaveModes 走 fileutil.AtomicWrite（临时文件+
  rename，不留半截 JSON）、archive JSONL 单次 Write 落整行（消除双写撕裂
  窗），新增 7 回归测试 ③XlsxPreview 大表格行虚拟滚动（观察项收账）——
  300 行以上只渲染可见窗口 ±10 overscan + spacer 保滚动条总高，冻结行常
  驻；2000×100 预览不再整表 20 万 td，小表全量渲染逐字节不变；2 新
  vitest。验证：Go 全量绿、vitest **809/809**（148 文件）、tsc/eslint 0、
  绑定面 535 不变、drift 不受影响。
- **最新发布：v4.8.3（2026-08-30）**——微信图片双向真协议（v4.8.2 发布
  当日真机实测复盘五刀）：①出图回推 getuploadurl→AES-128-ECB→CDN 密文
  上传→image_item 卡片+caption 补发，真机 delivered（此前 SendFileCard
  是孤儿=真凶）②发图识别 type=2+aeskey 解密下载→魔数终审落盘，真机两
  连发通过 ③识图模型升级多模态 Qwen3.6-35B 优先（手写体强、零额外显存），
  PaddleOCR→MinerU→OvisOCR2 降兜底 ④身份类问题跳过联网搜索（乱字母根
  因）⑤关键坑锁死（type=2/aes_key=base64(hex)/上传域与扫码 baseurl 无
  关）。协议三方印证（本机抓包解密+hermes-agent+openilink SDK）。验证：
  Go 全量绿、绑定 535 不变、前端零改动。详见 releases/v4.8.3.md。
- **下一执行**：Realtime 真机验证轮（真 key 下端到端对话/打断体感/AEC，
  S2 骨架已就绪）；手写识别质量复测；iLink 语音/视频 item 未探明（静默
  跳过）=观察项；生命库可写化=观察项。
- 构建注意：wails build 走 `build.bat` 的 TMP/TEMP 重定向到 `.tmp`（规避 SAC
  策略拦截）。

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
- [x] 遗留收尾：gate_test.go stubGate 计数器加锁（v4.8.2 已清账）；持久化
  套件统一（desktop_session SaveModes → fileutil.AtomicWrite；archive
  JSONL 单次 Write 落整行）——本轮收口

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
- [x] **v4.1a2 证据链补齐**：multi_edit/edit_lines 逐条摘要；xlsx_apply 接入
  （App 层写 Journal，SessionID 取自控制器，play 红线过滤）；新绑定
  GaeaJournalList（gen_bindings 重生成 bindingNames 503）+ 前端「证据」入口
  （DeliverablesPanel 证据链折叠区，展示最近证据卡）
- [x] **v4.1b Verifier + 回滚**：基线快照（写盘工具 + xlsx 应用前）；GaeaVerifyRecord
  双通道复核（A 结构/引用完整性 + B 基线 PDF 视觉健全性）+ verdict 落库；
  GaeaRollbackRecord 基线回滚 + 手工编辑冲突保护；前端证据卡复核/回滚操作
- [x] **v4.1c 规范包**：internal/office/standard GB/T 9704 红头 7 要素 lint +
  GaeaDocumentLint 绑定 + OfficePanel「规范体检」入口

### v4.2 造价 AI 化 —— ✅ 全部完成（v4.2.0 发布）
- [x] **v4.2 设计定稿**（`docs/gaea-v42-cost-ai-design.md`）：三支柱契约 +
  SchemaV15 数据模型（cost_inquiry_records / cost_stage_values）+ 绑定面 + Step 拆分
- [x] **v4.2a 组价底座**：`cost.PriceBand` 价格带推荐纯函数（P25/P50/P75/离散度/
  离群/置信度/证据链 BandSource，R-7 分位数与 costref 同口径）+ `RecommendPrice`
  五档；22 测试
- [x] **v4.2b 询价飞轮核心包**：`costinquiry` 四源归一数据点（信息价/OCR报价/供应商
  比价/手动询价）+ 到期预警（valid_until ≤ today+days）+ 调差建议（标题归一化匹配
  + |差幅|>2%）；7 测试
- [x] **v4.2b 五算对比核心包**：`coststage` 估/概/预/结/决阶段值（UPSERT）+ 对比
  计算（环比/累计差/除零保护）+ 偏差特征（正常<5/关注 5-15/异常>15 + 规则文案）；
  17 测试
- [x] **v4.2c AI 组价绑定**：`GaeaCostCompose`（关键词+语义召回+rerank 精排 →
  PriceBand → LLM 人材机拆解 `routeSensitiveLocal("office")` 失败规则降级 →
  建议视图）+ `GaeaCostComposeApply` 回写
- [x] **v4.2d 前端面板**：ComposeModal（测算明细行「AI 组价」：价格带卡/证据链 8 列
  表/人材机行编辑 → 应用为明细行）；询价视图（CostLibraryView 第三视图：数据点
  CRUD/四源徽标/预警横幅/调差一键更新）；FiveCalcPanel（项目详情五算区：输入保存/
  对比表着色/偏差卡）
- [x] **后端集成**：SchemaV15 正式迁移收编 + CostB 门面 11 新绑定（gen_bindings
  重生成，绑定面 506→517）+ bindingNames/spaceBindings(229)/bridge/types 接线

**v4.2 造价 AI 化已收官**（bindingNames 517 / spaceBindings 229 / vitest 759
+ go 全量绿）。剩余领域包：v4.3 乐园 / v4.4 触点（§10.4）。

### v4.3 乐园做深 —— ✅ 全部完成（v4.3.0 发布）
- [x] **v4.3 设计定稿**（`docs/gaea-v43-play-deepen-design.md`）：4 份只读调研
  （轻语关系记忆/情感语音 TTS/创作间世界模型/角色资产库）结论入账——后端骨架约
  70% 已存在，本版以「接线 + 参数扩展 + 持久化闭环」为主
- [x] **v4.3a 轻语记忆持久化闭环**：memory_associations/user_habits/temporal_anchors
  三表补 repo（此前有 schema 无 repos）+ 装配进持久化链 + ReseedAssociationGraph
  打通 + hermes.db 外键延迟检查实证修复
- [x] **v4.3b 关系图谱子图召回**：KnowledgeGraph.QuerySubgraph 多跳邻接（BFS+去重+
  权重，10 测试）+ GaeaWhisperGraphSubgraph 绑定（play）
- [x] **v4.3c 主动关心**：GaeaWhisperProactiveNow 评估绑定（门控+合成器复用）；
  前端「轻语先开口」；定时推送留后续小步
- [x] **v4.3d 情感语音**：TTSProvider.SynthesizeWithParams 参数扩展（cosyvoice 工厂
  修复不丢 voiceDescription、edge SSML 参数化）+ GetEmotionVoiceParams 情绪映射 +
  EmotionState.Mood 长期心境维（EWMA α=0.01 持久化）+ TTSSpeakBase64WithParams
- [x] **v4.3g 创作间图文联动**：章节配图复活死绑定（ChapterPage 按钮 + 弹窗）；
  GaeaGenerateBookCover 3:4 书封落 play exports（修 Windows 盘符卷 bug）
- [x] **绑定集成**：gen_bindings 522（+5，显式覆盖归位 voice/novel）+ bindingNames/
  spaceBindings(233)/bridge/types 接线

### v4.3 后续小步 —— ✅ 全部完成（v4.3.1 发布）
- [x] **主动关心定时推送频控（v4.3c 补完）**：app 层 ticker 四信号评估（AttentionManager
  频控 ≤3 条/小时 → MatchHabits dnd 作息尊重 → DetectSpecialDatesV2 生日祝福（每天首条、
  人格感知提示词）→ 门控+合成器）→ `gaea-whisper-proactive` 事件推前端（space=play）；
  新绑定 GaeaWhisperProactiveConfig/SetProactiveConfig（开关/上限/间隔/免打扰时窗）；
  前端 WhisperGraphPanel 订阅显示推送气泡（birthday 徽标）；16 Go 测试
- [x] **设定页维度化编辑器 + 伏笔登记表 + 一致性面板（v4.3e/f 落地）**：NovelSettingPage
  「维度化」模式（6 维度卡片就地编辑整存，复用已有绑定）+ ForeshadowPanel（状态流转/
  回收率）+ ConsistencyPanel（三类告警/重新检查）；7 vitest
- [x] **角色参考图 + 生图参考槽（v4.3g 补完）**：characterlib SchemaV2 迁移（reference/
  gallery_images 两列幂等）+ CharacterGeneratePortraitWithRef（img2img 参考槽 denoise
  0.55 + 模型门禁）+ 前端参考图管理（以参考图生成剧照）；8 vitest + 5 Go 测试
- [x] **文本朗读情绪 UI（v4.3d 收尾）**：EmotionSpeakSelector（9 标签对齐 EmotionVoiceMap）+
  会话情绪自动跟随 + TTSSpeakBase64WithParams 携带情绪（无情绪回退原路径）；3 vitest
- [x] **绑定收口**：gen_bindings 525（+3）+ bindingNames/spaceBindings(235)/bridge/types

**v4.3.1 后续小步已收官**（bindingNames 525 / spaceBindings 235 / vitest 789
+ go 118/118 包）。剩余：v4.4.1 触发已有能力（生图/办公任务路由 + 图片/文件卡片协议）+ 语音双通路（work 指令/play 闲聊分叉）+ 全局离线模式总开关（§10.4）；观察项（主动关心配置面板、角色 gallery 管理、IP-Adapter 节点级参考槽）。

### v4.3.2 首页重构（双翼·中庭 + 空间导航收敛）—— ✅ 已完成（v4.3.2 发布）
- [x] **首页「双翼·中庭」**：中庭 = 语音 + 打字一体对话条（输入框打字走
  `VoiceChatText` 复用语音对话管道，回复经 voice:reply 回传；orb 放大 148px
  呼吸环磁吸锚点；hero 让位顶部细眉——遵循 design-taste-frontend：不对称组织、
  避免等宽栏与居中 hero 俗套）；左翼「书房」2×2 紧凑格（办公/造价/记忆/模型）；
  右翼「庭院」纵向列表（聊天/小说/绘梦/角色）；门廊 = 编程（独立窗口徽标）+
  设置。命名 工位→**书房**、乐园→**庭院**（三语字典同步）。
- [x] **空间导航收敛**：移除一级导航（rail）顶部空间切换按钮（此前位置隐蔽）；
  `navigateBoard` 统一导航按板块 manifest.space 自动切空间（书房板块→work /
  庭院板块→play / 编程→independent / 设置→shared）；rail 展示全部板块；
  搜索 scope 文案同步 书房/庭院（useSpaceScope）。
- [x] **验证**：Go 全量绿；vitest 789/789（scope 断言同步）；tsc/eslint 0；
  版本五处统一 4.3.2；桌面端 gaea-v4.3.2.exe（33MB，SHA256 6a0486db）+
  冒烟 /api/health 200。提交：9789d7e / 71b0f2b / 766a9f6 / 0f21650。
- 遗留：中庭对话条桌面端语音+打字双通道实际体验待验证；双翼板块卡数量由
  manifest 派生（新板块自动入对应栏）。

### v4.4.0 触点一期（微信遥控器·离线代办）—— ✅ 已完成（v4.4.0 发布）
- [x] **主动推送通路**：weixin.Server 最近活跃会话记忆（handle 时记录
  fromUser/contextToken）+ Push 主动回推文本（无会话报错）；httptest
  校验 sendmessage 目标与文本 item（3 测试）
- [x] **离线代办提醒域**（weixin_reminder.go）：中文时间解析（相对时长/
  日期前缀+段词/裸时刻 20 用例表驱动，中文数字「十」进位，明早/明晚
  预处理拆词，无段词字面解释）→ wxReminder JSON 持久化（重启恢复，
  done 不重推）→ tryWxReminder 微信消息任务路由（提醒类接管/解析失败
  回格式提示/其余走聊天）→ 20s ticker 到点回推（失败重试 ≤5 标 failed）
  → remindersEnabled 开关持久化
- [x] **WeixinPage 落地（书房板块）**：扫码绑定流（QR 轮询/need_verifycode
  配对码/confirmed 落 WhisperAssistantSave 自动重拉通道）+ 通道状态徽标
  （运行/过期/未绑定）+ 提醒列表（手动新建/删除/回推开关）+ 指令说明；
  weixin 板块 Page=""→WeixinPage、inMenu=true（rail + 首页左翼书房格
  manifest 派生自动生效）
- [x] **绑定面 525→530**：WhisperWeixin* 4 + WhisperAssistant* 3 自
  LegacySurfaceNames 转正；WeixinReminder* 5 新增（voice 门面）；
  spaceBindings 235→247 全归 work；bindingNames.ts 同步
- [x] **验证**：Go 107/107 包；vitest 789/789（manifests/launcher/
  spaceBindings/board_manifest 锁数量断言 +12 同步）；tsc/eslint 0；
  drift 530 PASS；版本五处统一 4.4.0；gaea-v4.4.0.exe（35MB，
  SHA256 ee9b45c2）+ 冒烟 200。提交：v4.4a 后端 / v4.4b 前端 / release
- 遗留：回推目标=最近活跃会话（多联系人场景由多助手隔离）；iLink 图片/
  文件卡片协议未探明（v4.4.1 触发生图/办公任务回推时处理）；LLM 意图
  路由（通用任务化）留后续刀

### v4.5.0 指令中枢（统一意图路由 + 语音指令）—— ✅ 已完成（v4.5.0 发布）
- [x] **规划修订**（docs/gaea-nextgen-roadmap-2026.md §10.4a + 阶段 4）：审查
  结论=§8 革命性跃升未实质落地，插入 v4.5 指令中枢主轴（同内核多入口），
  v4.6 端到端实时语音接替排期
- [x] **S4.1 intent 解析包**：导航/生图/状态/提醒四类意图 + 板块别名表；
  宁漏勿误纪律（21 命中 + 8 误判防线表驱动）
- [x] **S4.2 能力执行层**：App.routeIntent——navigate(manifest 校验+事件)/
  generate_image/status/reminder(复用离线代办 source=voice)；零新增绑定
- [x] **S4.3 语音通路**：setWhisperChatFn 闭包分流——命中即能力执行经 TTS
  播报（含打断），未命中透传聊天；voice 包零改动
- [x] **S4.4 前端**：events.ts INTENT_NAVIGATE（22→23）+ MainLayout 订阅 →
  navigateBoard 自动切空间
- [x] **验证**：Go 108/108；vitest 789/789（events 断言同步）；tsc/eslint 0；
  版本五处 4.5.0；gaea-v4.5.0.exe（35MB，SHA256 027c726d）+ 冒烟 200
- 遗留：能力面仅四类（导航/生图/状态/提醒）；生图结果无语音播报进度；
  LLM 兜底分类器未接（规则引擎对未覆盖意图一律走聊天）

### v4.6.0 双空间收尾·纵深补课（审计后第一轮收账）—— ✅ 已完成（v4.6.0）
- [x] **红线 ①记忆注入按空间收窄**：`boot/sysprompt.go` 系统提示词索引 +
  `controller_memory.go` refreshMemoryLocked 两个生产调用点传 `Options.Space`
  → InSpace 读端视图（work 只注入 work / play 只注入 play / mode=off 旧行为）；
  回归 TestRefreshMemorySpaceIsolation + TestBuildSystemPromptMemorySpaceScoped
- [x] **红线 ②任务分账生产启用**：`[tasks]` 配置段（max_concurrent/per_space/
  priority）→ startTaskScheduler 默认 {work=1, play=1} + 价格抓取优先
  （price_fetch=20/file_index=10）；显式 `per_space = {}` 关分账回退全局 sem；
  回归 TestTasksConfigTOML + TestTaskSchedulerOptionsDefaults
- [x] **红线 ③事件过滤推广**：onTaskEvent(cb, space?) 订阅层过滤 + TaskCenter/
  useRunningBadge/PriceSourcesPanel/WorkspaceSearchPanel 按 work + MainLayout
  subscribeForSpace；回归 onTaskEvent 空间过滤测试
- [x] **治理收尾**：keepAlive 裸轮询 8 处（TaskCenter/SubagentsPanel/
  FeatureModelBar/ProgrammingPage/BenchmarkSection/useStatsState/
  useImageGenQueue/useBridgeWatch）全接 usePollingGate；CSS 真硬编码 token 化
  （novel 阅读高亮/批注色板 --novel-read-*/--ann-*、chat-board 混白）
- [x] **纵深 Mood→TTS**：MoodToVoiceDescription 连续韵律 + WhisperChatFn 透传
  mood + 中性轮次心境主导；回归 TestMoodToVoiceDescription +
  TestManager_MoodDrivesNeutralTurnProsody
- [x] **纵深 Verifier 通道 B**：soffice→pdftoppm→纯 Go 像素差异率（页数联合
  判定 pass/warn/fail，渲染降级 warn），审计产物落 journal/verify/<id>；覆盖
  全部有基线的写盘工具；失败回 Plan（证据卡内联结论 + xlsx 重新规划按钮）；
  回归 TestPixelDiffRatio + TestRunVisualDiffVerdicts
- [x] **纵深 询价飞轮**：调差异常分级（正常/关注/异常）+ PredictNext 线性回归
  预测 + GaeaCostImportApply 变参 inquirySource → OCR 报价单幂等入询价库；
  回归 4 个新测试（含飞轮接线）
- [x] **验证**：Go 全量绿（无 FAIL）；vitest 791/791（144 文件）；tsc/eslint 0；
  绑定面不变（变参向后兼容）；版本统一 4.6.0
- 欠账（如实列示于 releases/v4.6.0.md）：规范包机制化 / 成本知识图谱+归因 /
  生命库可写化（评估=不盲写 Herdsman 库）/ Verifier 通道 A 引用级深化 /
  Mood 不进前端手动朗读（设计如此）

### v4.6.1 微信统一路由·规范包机制·归因对标（审计补课续刀）—— ✅ 已完成（v4.6.1）
- [x] **S4.5 微信接统一路由**：routeIntentWithResult（产物感知，routeIntent
  包装不变）+ 微信回调提醒之后先过统一路由（navigate/生图/状态/提醒全命中，
  未命中才走聊天）；iLink image_item/file_item 防御解析 + 非文本消息转模型
  提示行（发图即识别入口），未知空项宁漏勿误；回归 3 测试
- [x] **规范包机制化**：standard.Checker 注册表 + LintDocument 聚合；红头要素
  （既有）+ 造价工程表式（工程名称/编制依据/单位造价/人材机/合计/说明）双检查
  器；GaeaDocumentLint 切 LintDocument；OfficePanel 按规范包分组；回归 2 测试
- [x] **成本归因对标**：costref.ComputeAttribution 纯函数（P25/P75 带宽 + 中位
  基准，差幅等级/贡献金额/主因 TopDrivers/带宽退化 ±10% 兜底/无参考另计）；
  新绑定 GaeaCostAttribution（531，参考池排除本项目防自对标）+ FiveCalcPanel
  归因区；回归 3 测试
- [x] **验证**：Go 全量绿；vitest 791/791（144 文件）；tsc/eslint 0；绑定面 531、
  spaceBindings 248、drift PASS；版本统一 4.6.1
- 欠账（如实列示于 releases/v4.6.1.md）：iLink 文件卡片/字节级图片识别待真机
  收敛；成本知识图谱可视化形态；生命库可写化维持评估结论；微信通用能力触发
  依赖卡片协议

## v4.x 执行审计（2026-08-30）

- 全量「承诺 vs 代码」对照（三路探查，逐项到文件行号）已记录
  `docs/audit-2026-08-30-v4-execution-review.md`：阶段 0 地基五项与双空间内核
  骨架为真；红线缺口三条（记忆注入主链路跨空间未接线 / 任务空间分账未启用 /
  事件订阅过滤仅 1 处）+ 领域包纵深欠账表（Verifier 通道 B=页数对比、规范包
  =单 lint、询价无异常检测/预测、OCR→询价飞轮反向未接、成本图谱零实现、
  Mood 只存不用、生命库只读）+ 补课刀序（§E）。纪律修正：每刀验收加
  「纵深检查」，发布说明列欠账清单。

## 纪律（沿用）

- 每 Step 独立提交可回退；旧数据只读兼容；回退演练纳入验收；不做新板块、不堆功能；
  docs/ 过时文档一律归档到 `docs/archive/`（权威 = 长期规划 + AGENTS.md + 本文件）。
