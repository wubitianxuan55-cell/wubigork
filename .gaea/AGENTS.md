# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态

- **V3.0 总规划（2026-08-15 定稿，V1-V8 已全部确认）**：docs/2026-08-15-gaea3-vision-roadmap.md（个人 AI 智能体平台：一个内核 + 统一记忆 + 板块插件化 + 本地优先/分层智能 + 移动终端）；架构执行见 docs/2026-08-15-gaea3-architecture-design.md（Step 0-3）。已确认决策：V1 愿景=个人 AI 智能体平台；V2 3.0.0 首发=chat 会话可恢复+知识库试点；V3 版本节奏=3.0.0 地基→3.1.0 板块生态·记忆起步→3.2.0 受控自主→3.3.0+ 身份；V4 记忆统一层 3.2.0；V5 数字生命 3.3+ 再定；V6 启动用户试用；V7 分层智能模型策略（云端统筹+本地执行+能本地则本地）；V8 插件化边界（只做启动期声明式装配，不做运行期热替换；工具级 MCP 热增删保留例外）；D8 office 模块路由 GaeaSend；单窗口编排已废弃。编程板块搁置（用户指示）。
- **下一会话计划（2026-08-15 定）**：开始写代码。顺序 = 阶段 7（v2.34-2.37 正确性纵深，既有计划照旧）→ Step 0（office 模块补注册路由 GaeaSend + MainBrainChat 全链路测试 + 版本源三处同步脚本化，0.5 天，可搭阶段 7 任一刀发布）→ Step 1 事件日志。**回退保障为硬要求**（用户强调）：每 Step 独立提交可 revert、旧数据只读兼容、二进制保留 5 版、运行时开关可切旧机制、每 Step 验收含回退演练——四层保障详见架构文档 §6.6。
- **架构方向备忘（2026-08-15）**：3.0 方向 = 向 DSH 靠拢的插件化地基（事件日志事实源 / 板块 Manifest / Provider Seam，见 docs/2026-08-15-gaea3-architecture-design.md）。**单窗口编排方案（原 docs/gaea2/module-protocol.md §5）已废弃并删除该文档，勿再引用或复活。**

- **3.0 架构改造设计（2026-08-15 定稿，待评审后开工）**：权威文档 docs/2026-08-15-gaea3-architecture-design.md（事件日志事实源 / 板块 Manifest / Provider Seam，四步实施计划）；调研证据存档在 docs/gaea3-review/（只读参考，权威性以设计文档为准）。阶段 7（v2.34-2.37 正确性纵深）先行，3.0 Step 0-3 在其后启动。

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
