# 任务进度

> 最后更新: 2026-08-30（v4.3.2 首页重构发布完成）

## 当前状态

- **最新发布：v4.3.2（2026-08-30）**——首页「双翼·中庭」重构 + 空间导航收敛：
  中庭语音+打字一体对话条（VoiceChatText 共用管道 + 放大 orb 磁吸核心）、左翼
  书房 2×2 格、右翼庭院纵向列表、门廊编程独立入口；命名 工位→书房、乐园→庭院；
  移除 rail 空间切换，navigateBoard 按板块自动切空间。验证：Go 全量绿；
  vitest 789/789；tsc -b / eslint 0；版本五处统一 4.3.2。
- **桌面端产物（2026-08-30）**：`releases/gaea-v4.3.2.exe`（36,630,016 字节 ≈ 33MB，
  SHA256 `6a0486db4d8b03b221867d401df7e08a045b99694a9f628d52f7bdeacb348dbb`，
  见 releases/SHA256SUMS-v4.3.2.txt）；wails v2.13.0 构建（1m1s）+ 冒烟
  /api/health 200（GAEA_HTTP_PORT=18999 隔离）。
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

## 纪律（沿用）

- 每 Step 独立提交可回退；旧数据只读兼容；回退演练纳入验收；不做新板块、不堆功能；
  docs/ 过时文档一律归档到 `docs/archive/`（权威 = 长期规划 + AGENTS.md + 本文件）。
