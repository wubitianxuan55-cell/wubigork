# 任务进度

> 本文件为**最近发布速览**：仅保留最近 7 条版本记录（v4.171.0 ~ v4.177.0）。
> v4.170.0 及之前的完整历史磁带（90 条记录 + 历史期章节）已于 2026-09-09 瘦身归档至
> `docs/archive/progress-history-2026-09.md`；历史详情请读归档文件。

## 最新发布：v4.177.0（2026-09-09）「瘦身 P4 懒加载三刀收官（mermaid 动态化 H1 + locales 按需 H7 + SearchModal lazy H8）」

- **H1 mermaid 动态 import**：Markdown.tsx/mermaidPng.ts 删静态 import → ensureMermaid 异步 await（模块缓存，strict/loose 配置原样）；MermaidBlock 迁 useEffect async IIFE（取消/失败语义不变）。**全仓 mermaid 静态 import 清零、entry 对 mermaid.core 零引用、566.9KB 独立 async chunk 按需**；测试零改动。
- **H7 locales 按需**：zh 静态保留（首帧零异步）+ en/zh-TW 动态 import 显式分支（Vite 静态解析独立 chunk）；useT/t/setPref 签名零变动；Provider 切换先同步渲染回退→chunk 就绪 forceRender+alive。
- **主代理关键修复**：translate 回退链 `DICTS[locale]→DICTS.zh→DICTS.en→key`——**zh 静态恒可用兜底**，非 React 调用方（tools.ts 摘要，currentLocale 默认 en）在 en chunk 未加载时裸键永不外泄（首跑 13 例失败根因）。5 测试补 beforeAll(loadLocale('en')) + ModelSwitcher en 用例 await。
- **H8 SearchModal lazy**：React.lazy + Suspense fallback null；测试不经 MainLayout 零改动。
- **门禁**：vitest 2737/2737（首跑 13 例 i18n 时序失败修复后全绿；1 例 FilePreviewModal 并发负载 flaky 单独复跑 15/15 绿在册先例）、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=E98E49CD8134C2F0964A1E041F49714465A385AB678E6295A946F94971E7B5D0。
- **度量**：**entry 1193→755.48KB（−37%）**；en=76.95/zh-TW=74.91/SearchModal 独立 chunk；**exe strip 实战验证**（build.bat 固化 `-ldflags "-s -w" -trimpath`，46.2→46.15MB 减量甚微——Go 符号表在 Wails 应用占比小，-trimpath 安全收益为主）。
- **P4 收官**：Go >50KB 0；entry −37%；MemoryHubPage 页壳 15.95KB；mock/office 1.1KB 入口；wailsjsCompat 退役（v4.174）。
- **欠账**：壳内真机池（rail 切换器/home 最近文档/.gsched）、审计刀D 真机取证（printSvg print/拖拽/粘贴）、观察池刀3（工作台内嵌办公）待评估——全部真机绑定的观察项不变。

## 最新发布：v4.176.0（2026-09-09）「瘦身 P4 结构刀2 + entry 懒加载（mock/office 拆分 + MemoryHubPage 全 tab lazy + mock 异步 chunk）」

- **结构刀2**：mock/office.ts 51.2→1.1KB 入口（6 文件：types/state/schedule 10.5/methods_xlsx 4.9/methods_office 32.7/build）。**TS2632 教训**：`let mockScheduleCurrent` 被 ScheduleProjectCreate/Delete 直接重赋值——跨模块重赋值导入绑定被 TS 禁止，schedule 状态与方法必须同文件（与 weixin.ts 域内私有状态先例一致）。
- **运行刀 entry 懒加载**（调研修正 P0 两处：cytoscape/cynefin 是 mermaid 传递依赖非 hub 相关、pageLazy 是 PDF 逐页懒挂载非页面 lazy——页面级 lazy 早全就位）：
  - **H6 mock 异步 chunk**：proxy.ts 删静态 import → startMockChunk 单例 + 真机门控零加载 + 冷路径异步 thunk；events 走 mockEventSharedSync/waitMockReady。**index 1149→967.23KB（−182KB/−15.8%）**。
  - **H2-H5 MemoryHubPage 页内 lazy**：8 组件全 React.lazy（**1402.5→15.95KB 页壳**；GraphView/three.js 1,354.97KB、KnowledgePanel/Markdown/katex/mermaid.core 全独立 chunk 按需）。
  - **H1 mermaid 动态 import 递下一轮**（mermaidPng.ts 在 footprint 外静态可达 + memoryhub lazy 已拆共享 chunk + Markdown 测试面大）。
- **连带测试时序修复×2**：mock 异步化使 `window.__mockScheduleFile`（schedule hook）与 SearchModal RouteIntent/UnifiedSearch 演示规则在同步断言时未就绪——两测试补 `beforeAll(await waitMockReady())`，bridge.ts 入口加 waitMockReady re-export。首跑 7 例失败修复后全绿。
- **门禁**：vitest 2737/2737 全绿、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=402A1B1A937CF6690E435C1F4E61AA6BB6D72487543A6B078E0F9F5F132DBD2F。
- **欠账=P4 续**：H1 mermaid 动态 import（+mermaidPng.ts）、H7 locales 按需（−160~200KB）、H8 SearchModal lazy（−20~40KB）、**exe strip**（wails build -ldflags "-s -w" -trimpath 发布版，dev 保留符号）；壳内真机池/审计刀D 不变。

## 最新发布：v4.175.0（2026-09-09）「瘦身 P4 结构刀1：三巨文件拆分（config.go/engine.go/store.ts，Go >50KB 2→0）」

- **三线并发**（config/engine/store，足迹互斥）+ 主代理收口；行为零变化（导出面/签名/JSON tag/默认值逐字节保留）。
- **线A config.go 58.7→16.1KB**（7 文件同包）：config.go 核心骨架 + config_keys 8.6/config_types 14.7/config_features 2.7/config_prefs 4.9/config_realtime 0.9/config_save 12.5；逐段 Contains 断言 8 区逐字节命中。
- **线B engine.go 56.5→13.3KB**（8 文件同包）：engine.go 核心 + engine_keys 3.0/engine_custom 7.0/engine_crud 3.7/engine_connect 3.8/engine_models 12.6/engine_modelhub 10.9/engine_state 4.0；核实无 herdsman 专属函数；45 func 无重复无丢失。
- **线C store.ts 54.3→0.4KB**（聚合入口 export *×3）：store/controller 50.2 + store/preview 3.3 + store/commonts 1.5；18 导出面逐一同 63 消费方零改动；963/963 行逐字节对账；controller 相对 import 深度调整（./bridge→../bridge 等 4 行）。
- **主代理收口**：足迹 17 项零越界；Go >50KB 源文件 2→0；**mock/office.ts 51.2KB 浮现**（下轮目标）；store/controller.ts 50.2KB 裁定=单 store 自然边界保留（再拆需深拆 reducer 超红线）；entry chunk 1190.55→1176.62kB。
- **门禁**：Go 128/128 包 0 FAIL、vitest 2737/2737 首跑全绿、tsc/eslint 0、drift PASS@602（零绑定）、locale 0 死、冒烟 200、SHA256=729C942B1A21B4FE0DB242565808B8CE86DB6F5B812810C836DDEBE4065F9DB8。
- **欠账=P4 续**：mock/office.ts 51.2 拆分；entry 懒加载（MemoryHubPage 1.4MB/GaeaPage 622KB/cynefin 690KB 按 tab 动态 import——P0 基线 §1 解剖建议）；exe strip（-ldflags "-s -w" 实测对靶 46.22MB）；壳内真机池/审计刀D 不变。

## 最新发布：v4.174.0（2026-09-09）「瘦身 P3 版3终局：bridge 双轨退役收官（wailsjsCompat.ts shim 删除，全仓生产代码零引用）」

- **里程碑**：全仓生产代码 wailsjsCompat import **清零**；LegacySurfaceNames 292→**184**（累计转正 108 方法）；spaceBindings 423。
- **契约扩展批次三（四线共 42 方法）**：3a +18（VoiceBindings 9 + ImageBindings 2（GetTTSSpeakers/GetVoicePipelineConfig 实挂 ImageB）+ OfficeBindings 6 DataBackup* + Core 1 ListProjects）；3b +38（CharLibBindings 16 角色库族 + NovelBindings 22 章节/叙事状态/场景/项目角色族）；3c +3（QuickBrainstormBranches/CreateChapter 8 参/DeleteOutlineNode）；3d +4（SetActiveASR/TTSModel/SetChatVoiceModel + VoiceHealth）。
- **业务迁移 8 线并发**：组1 语音族（useVoiceChat 8 方法 **VoicePushAudio Array→base64 对齐 Go []byte 契约** + VoiceSettingsPanel + ChatPage 5 直调——22/22）；组2 cast 族（CreatePage/ChapterEditor **删 App as unknown as cast 对象**——绑定再生成后 cast 编译期冗余，走 NovelB 门面具名——9/9）；组3 角色库（api/characterlib 18 方法——44/44）；组4 章节/小说角色（ChapterPage 4 + novel/api/character 9——7/7）；组5 收尾（useBindState + DataPanel 9）；CreatePage 收尾（7 处+3c——9/9）；终局 3d（useVoiceState 5 + VoiceHealth + ChatPanel 4 保留整体 try/catch——47/47）。
- **主代理收尾**：ChatPage/ChapterPage.test 的 wailsjsCompat 别名引用改 bindingsBridge 直取；SettingsPage.test 死 vi.mock 改 bridge；**wailsjsCompat.ts shim 删除**。
- **GetCharacters 同名裁定**：charlib.ts 已有与 Go NovelB 同签名（角色页 via bridge，项目角色 via NovelB），直接可用零重复。
- **CharacterList 分类遗留裁定**：role 库族归 play 但 CharacterList=work（v4.48 微信触点先例）有意保留不追改。
- **门禁**：vitest 2737/2737 首跑全绿（shim 删除后零回归）、tsc 0、eslint 0、drift PASS@602（零绑定）、locale 0 死、Go 回归绿、冒烟 200、SHA256=AD7C7CED1066F7BFF077B2416D41145BEE207619868D411F9A39FDF518737E2C。
- **收官意义**：S2-3 兼容层退役，全部前端调用统一 gaea/lib/bridge 代理（?mock=1 dev mock 回退 + BridgeError 错误归一）；masterplan 轨道四「双轨」目标达成。
- **后续**：P4（config.go 58.7/engine.go 55.2/store.ts 54.3 拆分、entry 懒加载、exe strip）；壳内真机池（rail 切换器/home 最近文档/.gsched）、审计刀D 真机取证不变。

## 最新发布：v4.173.0（2026-09-09）「瘦身 P3 版3：bridge 双轨退役批次二（chat/novel/settings 三族 22 文件迁 bridge + 元组契约修正）」

- **六线并发**（1 契约扩展 + 1 契约修正 + 4 业务线，足迹互斥）+ 主代理收口；wailsjsCompat 引用 45→35、LegacySurfaceNames 276→247。
- **线1 契约扩展 29 新方法**：CoreBindings+5（GetConfig/SaveConfig/GetStats/ListSkills/ExportAll——**ExportAll 实测在 bindings_core.go:23 非 NovelB**）/ChatBindings+9（ChatTopicCreate/Delete/Rename/SetMode/Clear/ImportTopic/Send 6 参/StreamPlain 5 参/ExportMarkdown）/NovelBindings+12（含 **ChatWorldview 契约纠偏=bindings_chat.go:35 返回 map 非 string**）/VoiceBindings+2/ImageBindings+1（SetImageBackend 4 参）；已核实跳过 3（WhisperGetPersonalities/ListSessions/MemoryHubOverview）；drift 摘除 29（误删 MigrateProjectToV4 恢复）、spaceBindings +29 锁 331→360、mock 补齐。
- **线1b 元组契约错位修正（重要发现）**：bridge/chat.ts 原声明 `[T[],unknown]` 元组系 **T6-3.2 历史误读**（注释自认「读错返回 error」）——真实 Wails 对 Go ([]T,error) 成功返回数组失败 reject；修正为数组契约+mock 返回 []+mock-contract-t63 断言 Array.isArray；**下游零消费者**（tsc 全绿）——useChatTopics 迁移免去调研预测的元组解构。**教训：契约以真实 Wails 行为为权威，勿信 mock 自造形态；发现 bridge 注释自认「读错」时先验证真实返回再定迁移方案**。
- **chat 组**（22/22 绿）：useChatTopics 11 处 + ChatTopicCreate 返回 `{id;title;mode;[k]:unknown}` 进 `chat.Topic[]` state 需 as unknown as 双断言；useChatStream Record any→unknown 触发 answered_by/emotion 两处最小 cast；chat/utils `App.GaeaLogFrontendError`→`app.LogFrontendError`（core.ts 短名+mappings）；ChatPage.test bindingsBridge 扩展 11 共享 vi.fn。
- **novel 组**（25/25 绿）：10 源+4 测试；**NovelSettingPage.test 关键决策=app 用 Proxy 回落真代理**（审计刀B b 壳内导入导出用例的 SaveFileAs/PickFiles/ReadFileB64 必须走真实代理路由 window.go stub，纯对象 mock 打挂 3 例）；ChatWorldview 返回值 `unknown` 收窄 `typeof result?.reply === 'string'`。
- **settings 组**：api/settings.ts 8 调用全命中迁移；ChatPanel/DataPanel/VoiceSettingsPanel/useVoiceChat **宁少勿多递批次三**（WhisperClearSession/GetVoicePipelineConfig/GetTTSSpeakers/GaeaDataBackup×6/语音族 6 方法未转正）。
- **ModuleLauncher**：4 处迁移无遗留。
- **门禁**：vitest **2737/2737 首跑全绿**（批次一曾 5 例连带失败本次零失败）、tsc 0、eslint 0、drift PASS@602（零绑定）、locale 0 死、Go 回归绿、冒烟 200、SHA256=C3DFA08F1A56C2389C2E4D5515A0E7F09D017B0DFB763DD0CDC459EBE66750CE。
- **欠账=批次三**：语音族（useVoiceChat 8 直调+VoiceSettingsPanel，VoiceStart/VoiceStop/VoicePlaybackDone/VoiceCancelTTS/VoicePushAudio/VoiceSetPTTActive 6 方法转正+**直调同步 throw 语义重评**）、ChatPage 五直调（TTSSpeakBase64WithParams/TTSSpeakBase64/ChatTopicClear/ChatTopicExportMarkdown 后两者契约已转正可先迁）、WhisperClearSession/GetVoicePipelineConfig/GetTTSSpeakers 转正解锁 ChatPanel/VoiceSettingsPanel、DataPanel GaeaDataBackup×6、cast 族 CreatePage/ChapterEditor、角色库族 api/characterlib+novel/api/character、ChapterPage 四直调、useBindState、**wailsjsCompat.ts 退役+悬空注释清理**（api/engines.ts:362 等）。

## 最新发布：v4.172.0（2026-09-09）「瘦身 P3 版3：bridge 双轨退役批次一（wailsjsCompat 首批 12 文件迁 bridge）」

- **调研先行**（子代理）：39 个非测试文件三分类（2A 全命中零扩展 / 24B ≤3 缺失 / 13C 语音·角色库·cast 族暂缓）；权威「③ 不在 bridge」=drift.ts LegacySurfaceNames 292 项（扩 bridge 必经 _CheckAppBindingsCoversAll 编译期钉死+同步摘除）；排除项=api/engines.ts、api/image.ts 已走 `window.go?.app?.App ?? bridgeApp` 混合回退（迁移样板）、GitPanel 已用 bridge。
- **线1 契约扩展 16 新方法**：CoreBindings+4（GetAppInfo/GetBoardManifests/GetFeatureModel(feature)/GetFeatureModelEnabled(feature)）/ModelBindings+3（SetFeatureModel 3 参/SetFeatureModelEnabled/GetActiveModel）/ImageBindings+1（StartLocalTTSService）/VoiceBindings+6（VoiceApplySettings+Whisper 读写族 5——**Go 侧实挂 VoiceB 非 MemoryB**，WhisperUpdateFact 3 参）/NovelBindings+1（NovelSearch 单参）/OfficeBindings+1（SaveSettings+mappings）；drift 摘除 16 名、spaceBindings +16 分类（9 shared+7 play）锁 315→331、mock 补齐（新建 mock/voice.ts+mock/novel.ts）。
- **线2A 零依赖直迁**：PersonaPicker（CharacterList）/CharacterCard（WhisperAssistantSave——assistant.Assistant 结构直接满足 Partial 零断言）。
- **线2B 10 文件 4 路并发**（足迹互斥）：CharacterMemoryModal/WhisperMemoryModal（as unknown as 类型适配）/manifests(+2 tests)/AboutPanel/useChatVoice（去 `?.`）/useFeatureModel/BindSection/useEngineState(+test)/novelSearchUtils/OfficePanel（gaeaApp.SaveSettings+Reload）。
- **主代理收口=连带测试修复×2**：ChatPage/ChapterPage（C 类本体未迁）消费迁移过的 hook/util → 测试原 vi.mock 仅 wailsjsCompat 断言打空（首跑 5 例失败）→ 补 bridge vi.mock 与 wailsjsCompat mock **共享同一 vi.fn 引用**（bindingsBridge via vi.hoisted + Proxy 转发），22/22 转绿。
- **坑=vi.mock 相对路径必须与真实 import 绝对解析一致**：src/pages/ 下 bridge 是 `'../gaea/lib/bridge'` 非 `'../../gaea/lib/bridge'`（错路径解析到不存在模块、vi.mock 静默不生效、断言 calls=0 且 tsc 不报——诊断文件实锤后修正）。
- **门禁**：vitest 2737/2737（313 文件）、tsc/eslint 0、drift PASS@602（零绑定）、locale 0 死、Go 回归绿（零 Go 改动）、冒烟 200、SHA256=C9B020CE9B0257FDC21FE00D9C0417D0CCE4D47767BBE1BF2310B9AB1799AFE1。度量=wailsjsCompat 引用 57→45；LegacySurfaceNames 292→276。
- **欠账**：批次二（api/settings.ts 一次扩 GetConfig/SaveConfig/GetActiveModel/SetImageBackend/Voice* 解锁 DataPanel/ChatPanel/VoiceSettingsPanel/OfficePanel 四面板、useChatTopics ChatTopic* 簇+**元组重构**、useChatStream ChatSend/ChatStreamPlain+mock 事件发射、NovelPage/NovelSettingPage/novel 面板族、ModuleLauncher/pages/chat/utils）+批次三（语音族 useVoiceChat/VoiceSettingsPanel/useVoiceState 依赖直调同步 throw 需重评、ChatPage 五直调、cast 族 CreatePage/ChapterEditor 需先入 AppBindings 类型、角色库族 api/characterlib+novel/api/character、ChapterPage 四直调、useBindState/DataPanel）+ 全部完成后 wailsjsCompat.ts 退役（shim 删除）；壳内真机池/审计刀D 不变。

## 最新发布：v4.171.0（2026-09-09）「瘦身 P3 结构版2：office 抽核 + 次批巨文件拆分（>50KB 首拆池 9→0）」

- **接手会话收口**：工作树由前一会话（02:04–02:20）完成遗留未提交（office 抽核 + 次批拆分），本会话全量门禁核验+修 1 处 lint（ExportDialog react-refresh）+补齐发布流程。
- **线A office 抽核**：internal/office 顶层 8 文件（executor/job_manager/job_routing/session_mode/desktop_session/audit_log/types+测试）下沉新包 internal/core（fs/jobs/journal/modes/routing/sessionmode/types）；新 aliases.go 类型别名 ExecResult/AgentJobState 保绑定面生成签名（gen_bindings 产物受 TestBindingsCompleteness 保护，别名=同一类型）；app 层 office_handler/embed_check_test import 收口；archive.go 保留（whisper 事实归档不在范围）；**绑定面 602 零变更 drift PASS**。
- **线B 次批 5+1 拆**：KnowledgePanel 62.2→40.8（knowledge/ 8）/DeliverablesPanel 57.4→23.2（deliverables/ 4）/XlsxPreview 56.4→35.5（xlsxpreview/ 4）/ChapterPage 55.9→35.6（chapter/ 5）/herdsmanTemplates 55.7→1.6（data/templates/ 13）/SchedulePage 超额（schedule/ 3：CalendarEditor/ExportDialog/ProjectsManagePanel）；classNames/data-testid 逐字节照搬。
- **收口修复**：ExportDialog scoped warning×2（EXPORT_KIND_* 拆分前即局部 const 仅内用→去 export），eslint 0 error 0 warning 恢复。
- **坑=vitest 首跑 4 文件 31 例失败**（与 Go test-all 同机并发负载假红，复跑两次全绿 2737/2737=在册「并发负载 flaky 复跑清同族」先例）——发布门禁以复跑全绿为准，勿把首跑红当回归。
- **门禁**：Go 128/128 包 0 FAIL、vitest 2737（313 文件）、tsc/eslint 0、drift PASS@602（零绑定）、locale 0 死、build 34.7s+13.8s、冒烟 200、SHA256=834DCDEF35DDFCD049C7E9CB7E54FB3DB55EA04793B8055384770C42C2BCEB38。
- **欠账**：P3 版3 候选=bridge 双轨退役启动（wailsjsCompat 单 shim 仍被 57 文件引用渐进迁移）+ Go 侧 config.go 58.7/engine.go 55.2 + frontend store.ts 54.3 二次拆分 + entry/页面级懒加载；壳内真机池/审计刀D 不变。

## 历史发布（v4.170.0 及之前，已归档）

- **v4.170** 瘦身 P3 结构版1：巨文件首批 4 拆（App/bridge/types/GanttView，>50KB 9→5；逐字节/绑定零变更纪律）
- **v4.169** 瘦身 P2 主干：双空间并列落地 + home 空间感知化 + 走查待证项收口
- **v4.168 / v4.167** 瘦身 P2 刀1（schedule 并入办公文档面）+ 刀0（白名单解耦、P0 基线落档、审计刀D Filters、nanoid 升级）
- **v4.166 / v4.165 / v4.164** 瘦身 P1 快赢（轮子四刀、审计刀B/C、locale 死键、依赖验活）+ 拍板落档（路线A+私有许可）+ 收敛计划 W1 卫生刀
- **v4.163 / v4.162** gaea 长期日志机制（按日分文件+365 天保留）/ 进度计划导入导出壳内修复（原生对话框链路）
- **v4.161 ~ v4.159** 进度计划双代号线：对齐标杆二期（工程标尺+图面语言）、一级/二级分级横幅、画布手感（滚轮缩放+行距自适应）
- **v4.158 ~ v4.156** AI 组价复核闭环 / docx 证据链补齐 / pptx 真编辑刀2+刀3（编辑面板+版本对比，Office 三件套编辑闭环）
- **v4.155 ~ v4.150** 双工期口径四刀：SS/FF/SF×cd 全搭接重推、MPP Project 2013+ 变体导入、mspdi 互通、板块 UI、agent 通道、任务级工期单位（wd/cd）
- **v4.149 ~ v4.144** diff 确认卡四刀（回滚/结构化 diff 预览/引擎逐条确认/ops 投影对拍收官）+ 工程复制 + 导入为新工程安全化
- **v4.111 / v4.110** 进度计划板块起始：CPM 引擎+三视图自由切换、Project/斑马对齐（v4.112~v4.143 期记录以「最后更新」大段落内联保留：多工程/资源成本/AOA 手动布局/基线对比等）
- **v4.109 ~ v4.100** pptx 真编辑刀1 数据层、导图画布/多维表、Model Hub 蒸馏 MH1-4 + ComfyUI 预热、Hub 落库+绑定转正、办公搜索对齐、GenUI 围栏审计
- **v4.99 ~ v4.90** 三线并行收口：图像域契约、GenUI 蒸馏（回答即 UI）、Verifier 逐页缩略、子代理网络会话/恢复入口、CodeMirror 高亮、Mermaid strict、diff 渲染升级
- **v4.89 ~ v4.82** 上下文页与工具链：成本费率 hover、/context 弹层、Git 面板最小集、文件三态折叠、HTML 沙箱预览、工具结果缩略卡、上下文趋势跳转
- **v4.81 ~ v4.60** 早期面板期：上下文/文件/任务线（活动行级增量、任务页、记忆界面、子代理会话/并行/流式/追问闭环、办公板块交付验收 A2、better-sidebar pane 化）
