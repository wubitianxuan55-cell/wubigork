# gaea 项目版本历史存档（2026-09-09 自 .gaea/AGENTS.md「版本状态」段迁出）

> 迁入记录：2026-09-09 首迁（v4.49.0 及更早的旧「版本状态（历史存档）」段）；2026-09-10 二迁（v4.173.0–v4.146.0）——因 `.gaea/AGENTS.md` 曾达 104 KB 超工作区指令预算（65536 B），其版本段只留最近 14 版速览，尾部纪律/规划段一度被截断不可见。
> **覆盖范围（勿误读为全量磁带）**：本件含 v4.173–v4.146 与 v4.49 及更早（至 v3.0.6）。**v4.50–v4.145 不在本件**——该区间逐版一行账只在 `CHANGELOG.md`（220+ 版磁带）与 `releases/vX.Y.Z.md`；回填属可选工作。
> 用途：逐版演进全文检索。当前版本动态见 `.gaea/AGENTS.md` 顶部速览；发布物全文见 `releases/`。

## 版本状态

- **近期三版一行账**：v4.156.0 pptx 真编辑刀2+刀3（编辑面板+版本对比,Office 三件套编辑闭环,绑定 598）· v4.155.0 双工期欠账放开（cd 全搭接 SS/FF/SF,§3.2.5 重推零不动点）· v4.154.0 MPP 导入 Project 2013+ 变体（真机样本五处漂移修复）——观察池=mppdi 刀4 真机 Project/WPS/斑马走查、2013+ 资源/分配键位、E1 负数对称、list_projects 真机、**pptx 刀2 真机走查**；双工期欠账池剩=导入任务日历转 cd 迁移助手/斑马导入口径探测/等效跨度灰显双读数（按需）；pptx 欠账=刀4 修改队列泛化/docx_apply 证据链另立小刀。

- **最新发布：v4.178.0（2026-09-09）「造价·条目匹配键刀：定额编码贯通存储→导入→匹配→工具面→UI」**——造价域缺口清单 §1 首刀销账（docs/gaea-cost-domain-survey-2026-09.md）。**数据层**=SchemaV17 cost_entries/cost_estimate_items 增 code 列（TEXT DEFAULT ''，归一化存储）+idx_cost_code 索引，迁移测试覆盖新库全链 V1→V17+V16 升级（历史行默认空串）；**cost.NormalizeCode**=全角 ASCII→半角+去空白+大写（ａ１－１２/a1 12/A1-12 同码命中，空串=未录入）；Entry/Summary/Item 三结构 Save/Get/Search/List 贯通，搜索 haystack+BM25 均含 code。**导入链**=表头 fieldCode 映射（清单项目编码/定额编号/清单编码/项目编码/定额编码/定额号/清单号/编码，最长胜出，「编号」仍噪声）；**编码优先匹配**=带码命中→覆盖提示、带码未命中→直接新增（同标题不同编码=不同子目不标题兜底，杜绝跨子目误覆盖）、无码保持原语义；AI 解析 prompt 增 code 提取；vision 路径有意不动（报价单鲜带定额编码）。**匹配消费方**=costref entryIndexes 双索引→三索引、matchEntry 编码精确优先（matchedBy=code|entry_name|title 溯源，带码未命中宁漏勿误配与导入链同语义）；沉淀携带 item.Code+引用条目继承编码。**工具面/UI**=cost_save 增 code 参数（覆盖未传保留原编码）+cost_search 结果表编码列；EntryModal 编码表单项+LibraryView 编码 chip/行前缀+ImportModal 可编辑编码列+ProjectsView 明细编码输入+EntryPicker 引用带入；mock 层同批贯通。**附带收口=e-check 三处陈旧守卫修复**（E25 ChatSend/ChatImportTopic 接受 bridge 形态 app.ChatSend/app.ChatImportTopic=v4.174 双轨退役甩下；E24 MemoryHubPage→GraphView 接受 React.lazy 动态 import=v4.176 甩下；干净 HEAD 实测即红非本刀引入）。门禁=Go 全量 0 FAIL（五包 +9 测试）/vitest 2738/2738（2737+1）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/e-check OK/build strip+冒烟 200。SHA256=F83CDB9BED35C7B4F045CC835B59981B95B5B4B67E39EC1D5282EF8F2E199561。欠账=造价缺口 2-6（组价流式分步+多方案/五算快照贯通/询价异常扫描/检索主动维护+清单级测评集/行业含量区间）、编码存量回填等带码样本、壳内真机池/审计刀D 真机取证不变。


- **最新发布：v4.177.0（2026-09-09）「瘦身 P4 懒加载三刀收官（mermaid 动态化 H1 + locales 按需 H7 + SearchModal lazy H8）」**——瘦身执行层第十二版，P4 运行刀收官。**H1 mermaid 动态 import**（Markdown.tsx/mermaidPng.ts 删静态 import→ensureMermaid 异步 await+模块缓存 strict/loose 配置原样；MermaidBlock 迁 useEffect async IIFE 取消/失败语义不变；**全仓 mermaid 静态 import 清零、entry 对 mermaid.core 零引用、566.9KB 独立 async chunk 按需**；测试零改动）。**H7 locales 按需**（zh 静态保留首帧零异步+en/zh-TW 动态 import 显式分支独立 chunk；useT/t/setPref 签名零变动；Provider 先同步渲染回退→chunk 就绪 forceRender+alive）。**主代理关键修复=translate 回退链 DICTS[locale]→DICTS.zh→DICTS.en→key**——**zh 静态恒可用兜底**，非 React 调用方（lib/tools.ts 摘要 currentLocale 默认 en）在 en chunk 未加载时裸键永不外泄（首跑 13 例失败根因）；5 测试补 beforeAll(loadLocale('en'))+ModelSwitcher await。**H8 SearchModal lazy**（React.lazy+Suspense fallback null）。门禁=vitest 2737/2737（首跑 13 例 i18n 时序失败 zh 兜底+适配后全绿；1 例 FilePreviewModal 并发负载 flaky 单独复跑 15/15 绿在册先例）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=E98E49CD8134C2F0964A1E041F49714465A385AB678E6295A946F94971E7B5D0。**度量=entry 1193→755.48KB（−37%）**；en/zh-TW/SearchModal 独立 chunk；**exe strip 实战验证**（build.bat 固化 `-ldflags "-s -w" -trimpath`——46.2→46.15MB 减量甚微，Go 符号表在 Wails 应用占比小，-trimpath 安全收益为主；下轮起发布构建自动 strip）。**P4 收官**：Go >50KB 0；entry −37%；MemoryHubPage 页壳 15.95KB；mock/office 1.1KB 入口；wailsjsCompat 退役（v4.174）。欠账=壳内真机池/审计刀D 真机取证/观察池刀3（全真机绑定观察项）。


- **最新发布：v4.176.0（2026-09-09）「瘦身 P4 结构刀2 + entry 懒加载（mock/office 拆分 + MemoryHubPage 全 tab lazy + mock 异步 chunk）」**——瘦身执行层第十一版，P4 性能版第二刀。**结构刀2**：mock/office.ts 51.2→1.1KB 入口（6 文件：types/state/**schedule 10.5**/methods_xlsx 4.9/methods_office 32.7/build；**TS2632 教训**=let mockScheduleCurrent 被 Create/Delete 直接重赋值禁跨模块，schedule 状态与方法必须同文件与 weixin.ts 先例一致）。**运行刀 entry 懒加载**（调研修正 P0 两处：**cytoscape/cynefin 是 mermaid 传递依赖非 hub 相关、pageLazy 是 PDF 逐页懒挂载非页面 lazy**——页面级 lazy 早已 main.tsx registerPage 全就位）：**H6 mock 异步 chunk**（proxy.ts 删静态 import→startMockChunk 单例 `import("../mock")`+真机门控零加载+冷路径异步 thunk；events 走 mockEventSharedSync/waitMockReady；**index 1149→967.23KB −182KB/−15.8%**）；**H2-H5 MemoryHubPage 8 组件全页内 React.lazy**（**1402.5→15.95KB 页壳**；GraphView/three.js 1,354.97KB、KnowledgePanel 51.3/Markdown 212.9/katex 261.3/mermaid.core 609.8 全独立 chunk 按需——GraphView lazy 决策：home 默认 tab 首开仍拉一次与现状等价但其余入口不摊 three.js）。**H1 mermaid 动态 import 递下一轮**（mermaidPng.ts footprint 外静态可达 + memoryhub lazy 已拆共享 chunk + Markdown 测试面大）。**连带测试时序修复×2**（mock 异步化使 `window.__mockScheduleFile` 与 SearchModal RouteIntent/UnifiedSearch 演示规则同步断言未就绪——两测试补 `beforeAll(await waitMockReady())`、bridge.ts 入口加 waitMockReady re-export；首跑 7 例失败修复后全绿）。门禁=vitest 2737/2737 全绿/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=402A1B1A937CF6690E435C1F4E61AA6BB6D72487543A6B078E0F9F5F132DBD2F。度量=entry 967.23kB（基线 1193→−19%）。欠账=P4 续（**H1 mermaid 动态 import + mermaidPng.ts、H7 locales 按需 −160~200KB、H8 SearchModal lazy −20~40KB、exe strip wails -ldflags -s -w -trimpath 发布版 dev 保留符号**）、壳内真机池/审计刀D 不变。


- **最新发布：v4.175.0（2026-09-09）「瘦身 P4 结构刀1：三巨文件拆分（config.go/engine.go/store.ts，Go >50KB 2→0）」**——瘦身执行层第十版，P4 性能版结构前置刀。三线并发（足迹互斥）+ 主代理收口，行为零变化。**线A config.go 58.7→16.1KB**（7 文件同包：核心骨架+config_keys/config_types/config_features/config_prefs/config_realtime/config_save；逐段 Contains 断言 8 区逐字节命中）。**线B engine.go 56.5→13.3KB**（8 文件同包：核心+engine_keys/engine_custom/engine_crud/engine_connect/engine_models/engine_modelhub/engine_state；核实无 herdsman 专属函数；45 func 无重复无丢失）。**线C store.ts 54.3→0.4KB**（聚合入口 export *×3+store/controller+store/preview+store/commonts；18 导出面逐一同 63 消费方零改动；963/963 行逐字节对账；controller 相对 import 深度调整 4 行）。主代理收口：足迹 17 项零越界；Go >50KB 源文件 2→0；**mock/office.ts 51.2KB 浮现**（下轮目标）；store/controller.ts 50.2KB 裁定=单 store 自然边界保留（再拆需深拆 reducer 超行为零变化红线——50KB 红线是指导性目标，自然聚合边界可容忍略超）；entry chunk 1190.55→1176.62kB。门禁=Go 128/128 包 0 FAIL/vitest 2737/2737 首跑全绿/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/冒烟 200/SHA256=729C942B1A21B4FE0DB242565808B8CE86DB6F5B812810C836DDEBE4065F9DB8。欠账=P4 续（mock/office.ts 51.2 拆分、entry 懒加载 MemoryHubPage 1.4MB/GaeaPage 622KB/cynefin 690KB 按 tab 动态 import、exe strip -ldflags -s -w 对靶）、壳内真机池/审计刀D 不变。


- **最新发布：v4.174.0（2026-09-09）「瘦身 P3 版3终局：bridge 双轨退役收官（wailsjsCompat.ts shim 删除，全仓生产代码零引用）」**——瘦身执行层第九版，轨道四「双轨」渐进退役**收官刀**。**里程碑**：全仓生产 wailsjsCompat import 清零（python 实证）；**frontend/src/wailsjsCompat.ts 删除**；LegacySurfaceNames 292→**184**（累计转正 108 方法）；spaceBindings 423。**契约扩展批次三四线共 42 方法**：3a +18（VoiceBindings 9 语音/WhisperClearSession/TTS×2 + ImageBindings 2——**GetTTSSpeakers/GetVoicePipelineConfig 实挂 ImageB 非 VoiceB** + OfficeBindings 6 DataBackup* + Core 1 ListProjects）；3b +38（CharLibBindings 16 角色库族 + NovelBindings 22 章节族/叙事状态族/场景族/项目角色族，**GetCharacters 同名核对=charlib.ts 已有与 Go NovelB 同签名直接可用**，CancelCreateChapter→boolean，v4.7x 小说革命注释段更新 RunChapterGate 保留）；3c +3（QuickBrainstormBranches/CreateChapter 8 参/DeleteOutlineNode）；3d +4（SetActiveASRModel/SetActiveTTSModel/SetChatVoiceModel ImageB + VoiceHealth VoiceB）；spaceBindings 360→378→416→419→423。**业务迁移 8 线并发**（各线定向测试全绿）：组1 语音族（useVoiceChat 8 方法 **VoicePushAudio Array→base64 对齐 Go []byte 契约**、useVoiceState 5、VoiceSettingsPanel 4、ChatPage 5 直调——22/22）；组2 cast 族（**CreatePage/ChapterEditor 删 `App as unknown as {...}` cast 对象**=绑定再生成后 cast 编译期冗余走 NovelB 门面具名——9/9）；组3 角色库（api/characterlib 18 方法 6 处 as unknown as——44/44）；组4 章节/小说角色（ChapterPage 4 + novel/api/character 9——7/7）；组5 收尾（useBindState/DataPanel 9——9/9）；CreatePage 收尾（7 处+3c——9/9）；终局 3d（useVoiceState+VoiceHealth+ChatPanel 4 保留整体 try/catch——47/47）；主代理收尾（ChatPage.test/ChapterPage.test wailsjsCompat 别名改 bindingsBridge 直取、SettingsPage.test 死 vi.mock 改 bridge、**shim 删除**）。门禁=vitest **2737/2737 首跑全绿**（shim 删除后零回归）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=AD7C7CED1066F7BFF077B2416D41145BEE207619868D411F9A39FDF518737E2C。**收官意义**：S2-3 兼容层退役全部前端统一 gaea/lib/bridge 代理（?mock=1 dev mock 回退+BridgeError 错误归一），masterplan 轨道四「双轨」目标达成；CharacterList 分类遗留裁定=work（v4.48 微信触点先例）有意保留。后续=P4（config.go 58.7/engine.go 55.2/store.ts 54.3 拆分、entry 懒加载、exe strip）+ 壳内真机池/审计刀D 不变。

- **最新发布：v4.173.0（2026-09-09）「瘦身 P3 版3：bridge 双轨退役批次二（chat/novel/settings 三族 22 文件迁 bridge + 元组契约修正）」**——瘦身执行层第八版，轨道四「双轨」渐进退役第二刀（六线并发：1 契约扩展 + 1 契约修正 + 4 业务线，足迹互斥；wailsjsCompat 引用 45→35、LegacySurfaceNames 276→247）。**线1 契约扩展 29 新方法**：CoreBindings+5（GetConfig/SaveConfig/GetStats/ListSkills/**ExportAll 实测在 bindings_core.go:23 非 NovelB**）/ChatBindings+9（ChatTopicCreate/Delete/Rename/SetMode/Clear/ImportTopic/ChatSend 6 参/ChatStreamPlain 5 参/ExportMarkdown）/NovelBindings+12（**ChatWorldview 契约纠偏=bindings_chat.go:35 返回 map 非 string**，NovelSettingPage 消费处本按对象用仅需 unknown 收窄）/VoiceBindings+2（VoiceGetSettings/VoiceChatText）/ImageBindings+1（SetImageBackend 4 参）；已核实跳过 3（WhisperGetPersonalities/ListSessions/MemoryHubOverview 接口已有）；drift 摘除 29（误删 MigrateProjectToV4 立即恢复）、spaceBindings +29 锁 331→360、mock 补齐（novel.ts 重写保留 NovelSearch）。**线1b 元组契约错位修正（重要坑）**：bridge/chat.ts 原 `ChatTopicsList: Promise<[T[],unknown]>` 元组系 **T6-3.2 历史误读**（注释自认「读错返回 error」）——真实 Wails 对 Go ([]T,error) 成功返回数组失败 reject，正确契约=数组；修正 bridge+mock+contract-t63 断言（Array.isArray），**下游零消费者**（tsc 全绿证明），useChatTopics 迁移后 `|| []` 直接可用、免调研预测的元组解构。**教训：契约以真实 Wails 行为为权威勿信 mock 自造形态；发现 bridge 注释自认「读错」时先验证真实返回再定方案**。**chat 组**（22/22 绿）：useChatTopics 11 处+ChatTopicCreate 返回 `{id;title;mode;[k]:unknown}` 进 chat.Topic[] state 需 as unknown as 双断言；useChatStream Record any→unknown 触发 answered_by/emotion 两处最小 cast；chat/utils GaeaLogFrontendError→app.LogFrontendError（core.ts 短名+mappings:124）；ChatPage.test bindingsBridge 扩展 11 共享 vi.fn。**novel 组**（25/25 绿）：10 源+4 测试；**NovelSettingPage.test 关键决策=app 用 Proxy 回落真代理**（审计刀B b 壳内导入导出用例 SaveFileAs/PickFiles/ReadFileB64 必须走真实代理路由 window.go stub，纯对象 mock 打挂 3 例——stub 窗台方法必须真实代理）。**settings 组**：api/settings.ts 8 调用全命中迁移；ChatPanel/DataPanel/VoiceSettingsPanel/useVoiceChat **宁少勿多递批次三**。**ModuleLauncher**：4 处迁移无遗留。门禁=vitest **2737/2737 首跑全绿**（批次一曾 5 例连带失败本批零失败）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=C3DFA08F1A56C2389C2E4D5515A0E7F09D017B0DFB763DD0CDC459EBE66750CE。欠账=批次三（语音族 useVoiceChat+VoiceSettingsPanel 六方法转正+**直调同步 throw 语义重评**；ChatPage 五直调 TTSSpeakBase64WithParams/TTSSpeakBase64/ChatTopicClear/ChatTopicExportMarkdown 后两者契约已转正可先迁；WhisperClearSession/GetVoicePipelineConfig/GetTTSSpeakers 转正解锁 ChatPanel；DataPanel GaeaDataBackup×6；cast 族 CreatePage/ChapterEditor 先入类型；角色库族 api/characterlib+novel/api/character；ChapterPage 四直调；useBindState；**wailsjsCompat.ts 退役+悬空注释清理** api/engines.ts:362 等）。


- **最新发布：v4.172.0（2026-09-09）「瘦身 P3 版3：bridge 双轨退役批次一（wailsjsCompat 首批 12 文件迁 bridge）」**——瘦身执行层第七版，轨道四「双轨」渐进退役第一刀（出口判据=绑定零变更 drift PASS + 迁移面行为零变化/错误归一 BridgeError）。**调研先行**：子代理三分类——39 个非测试文件 = 2A（100% 命中零扩展）/24B（≤3 缺失）/13C（语音/角色库/cast 族暂缓）；权威「③ 不在 bridge」清单=drift.ts `LegacySurfaceNames` 292 项（扩 bridge 必经 `_CheckAppBindingsCoversAll` 编译期钉死，必须同步摘除）；排除项=api/engines.ts、api/image.ts 已走 `window.go?.app?.App ?? bridgeApp` 混合回退（迁移样板）、GitPanel 已用 bridge。**线1 契约扩展 16 新方法**：CoreBindings+4（GetAppInfo/GetBoardManifests/GetFeatureModel(feature)/GetFeatureModelEnabled(feature)）/ModelBindings+3（SetFeatureModel(feature,engineID,modelName) 3 参/SetFeatureModelEnabled/GetActiveModel）/ImageBindings+1（StartLocalTTSService）/VoiceBindings+6（VoiceApplySettings+Whisper 读写族 5——**Go 侧实挂 VoiceB 非 MemoryB**，任务书纠偏：WhisperUpdateFact 3 参）/NovelBindings+1（NovelSearch 单参）/OfficeBindings+1（SaveSettings+mappings SaveSettings:"GaeaSaveSettings"）；drift 摘除 16 名、spaceBindings +16 分类（9 shared+7 play）锁 315→331、mock 补齐（含新建 mock/voice.ts+mock/novel.ts）。**线2A 零依赖直迁**：PersonaPicker（CharacterList）/CharacterCard（WhisperAssistantSave——assistant.Assistant 结构可直接满足 Partial 零断言）。**线2B 10 文件 4 路并发**（足迹互斥）：CharacterMemoryModal/WhisperMemoryModal（as unknown as 类型适配）/manifests(+manifests.test+launcher.test bridge mock)/AboutPanel/useChatVoice（去 `?.`）/useFeatureModel/BindSection/useEngineState(+test)/novelSearchUtils/OfficePanel（gaeaApp.SaveSettings+Reload，AppModels.SettingsView→gaea SettingsView）。**主代理收口=连带测试修复×2**：ChatPage/ChapterPage（C 类本体未迁）消费迁移过的 useChatVoice（ChatAppendMessages）/novelSearchUtils（NovelSearch）→ 测试原 vi.mock 仅 wailsjsCompat 断言打空（首跑 5 例失败）——补 bridge vi.mock 与 wailsjsCompat mock **共享同一 vi.fn 引用**（bindingsBridge via vi.hoisted + Proxy 转发），22/22 转绿。**坑=vi.mock 相对路径必须与真实 import 绝对解析一致**：src/pages/ 下 bridge 是 `'../gaea/lib/bridge'` 非 `'../../gaea/lib/bridge'`（错路径解析到不存在模块、vi.mock 静默不生效、断言 calls=0 且 tsc 不报——诊断文件实锤后修正；新测试补 bridge mock 时先对照被 mock 文件自己的 import 路径写出相对路径）。门禁=vitest 2737/2737（313 文件）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿（本版零 Go 改动）/冒烟 200/SHA256=C9B020CE9B0257FDC21FE00D9C0417D0CCE4D47767BBE1BF2310B9AB1799AFE1。度量=wailsjsCompat 引用 57→45；LegacySurfaceNames 292→276。欠账=批次二（api/settings.ts 一次扩 GetConfig/SaveConfig/GetActiveModel/SetImageBackend/Voice* 解锁四面板、useChatTopics ChatTopic* 簇+**元组重构**、useChatStream ChatSend/ChatStreamPlain+mock 事件、NovelPage/NovelSettingPage/novel 面板族、ModuleLauncher/pages/chat/utils）+批次三（语音族 useVoiceChat/VoiceSettingsPanel/useVoiceState 依赖直调同步 throw 需重评、ChatPage 五直调、cast 族 CreatePage/ChapterEditor 需先入类型、角色库族 api/characterlib+novel/api/character、ChapterPage 四直调、useBindState/DataPanel）+ 全部完成后 wailsjsCompat.ts 退役（shim 删除）。壳内真机池/审计刀D 不变。


- **最新发布：v4.171.0（2026-09-09）「瘦身 P3 结构版2：office 抽核 + 次批巨文件拆分（>50KB 首拆池 9→0）」**——瘦身执行层第六版，P3 结构面第二刀（masterplan 轨道三；出口判据=绑定零变更 drift PASS + >50KB 首拆池 9→0）。**⚠接手会话收口**：工作树由前一会话（02:04–02:20）完成遗留未提交，本会话全量门禁核验+修 1 处 lint+补齐发布。**线A office 抽核**：`internal/office` 顶层 8 文件（executor/job_manager/job_routing/session_mode/desktop_session/audit_log/types+测试）下沉新包 `internal/core/`（fs/jobs/journal/modes/routing/sessionmode/types）；新 `internal/office/aliases.go` 类型别名 `ExecResult`/`AgentJobState`=绑定面生成签名（bindings_office.go 由 gen_bindings 生成受 TestBindingsCompleteness 保护，别名=同一类型 JSON/反射零变化）；app 层 office_handler.go/embed_check_test.go import 收口；**archive.go 保留**（whisper 事实归档不在抽核范围）。**线B 次批 5+1 拆**：KnowledgePanel 62.2→40.8（knowledge/ 8 文件：CenterView/constants/EditForm/EntryRow/HistoryModal/Inspector/MergeModal/utils）/DeliverablesPanel 57.4→23.2（deliverables/ 4：DeliverableRow/EvidenceSection/RegistrySection/shared）/XlsxPreview 56.4→35.5（xlsxpreview/ 4：FormulaBar/MiniChart/numFmt/SheetGrid）/ChapterPage 55.9→35.6（chapter/ 5：editChrome/errText/readingChrome/readingOverlays/readingPanel）/herdsmanTemplates 55.7→1.6（data/templates/ 13）/**SchedulePage 超额**（schedule/ 3：CalendarEditor/ExportDialog/ProjectsManagePanel）；classNames/data-testid 逐字节照搬。**收口修复**：ExportDialog react-refresh warning×2（EXPORT_KIND_LABEL/EXPORT_KIND_FILE/ExportKind 拆分前就是局部 const 仅内用却带 export→去 export 本地化）。**坑=vitest 首跑 4 文件 31 例失败**（与 Go test-all 同机并发负载假红，TrajectoryView「denied by policy」超时等），**复跑两次全绿 2737/2737**=在册「并发负载 flaky 复跑清同族」先例，勿把首跑红当回归。门禁=Go 128 包 0 FAIL/vitest 2737（313 文件）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/冒烟 200/SHA256=834DCDEF35DDFCD049C7E9CB7E54FB3DB55EA04793B8055384770C42C2BCEB38。欠账=P3 版3 候选（**bridge 双轨退役启动**：wailsjsCompat 单 shim 仍被 57 文件引用渐进迁移；Go 侧 config.go 58.7/engine.go 55.2+frontend store.ts 54.3 二次拆分；entry/页面级懒加载），壳内真机池/审计刀D 不变。


- **最新发布：v4.170.0（2026-09-09）「瘦身 P3 结构版1：巨文件首批 4 拆（App/bridge/types/GanttView，>50KB 9→5）」**——瘦身执行层第五版，P3 结构面第一刀（masterplan 轨道三；出口判据=绑定零变更 drift PASS + 拆分池 >50KB 9→5），四线并发子代理（足迹互斥）纯结构拆分=行为零变化/公开导出面逐一同。**线B App.tsx** 87.3→46.6KB：palette 构建/exportConversation/JSX 布局因 App.export.test/App.palette.test 的 src 级 readFileSync 断言**刻意留壳内**（拆走即红）+11 组逻辑迁新 `gaea/app/`（useSubagentTabs/useTaskCards/usePreviewAutoFront/useDeliverables/useWorkspaceLayout/usePreviewPanel/useSessionHandlers/useAppKeyboard/useTasksAutoOpen/sessionCleanup/layout），**deps 数组与执行顺序逐项原样**（35→16 组 effect，原 2 处 eslint-disable 保留；hook 参数化后 exhaustive-deps warn 按仓库风格加带理由 disable 注释）。**线C bridge.ts** 85.1→1.3KB：按 bindingNames 域切 `gaea/lib/bridge/` 16 文件（10 分域接口+`interface AppBindings extends …` 组合+mappings/proxy/events/http/drift），**Proxy/懒探测/mock 回退/事件时序原样**（PowerShell 按内容锚点切片非手抄，方法名 parity old=315=new 与 spaceBindings.test 锁一致），**bindingNames 602 零变更=drift 锁不动**，公开导出 17 名 re-export 逐一同。**线D types.ts** 66.4→1.2KB：纯类型按域拆 `gaea/lib/types/` 24 文件，**220 声明零增减零重名**（列 0 锚定 155+63 一致；缩进接口 CostSummary 成员/CostEntry/CostComponent 为 +2 真值），入口纯 type re-export，135 处消费方零改动。**线E GanttView.tsx** 64.8→25.2KB：`schedule/gantt/` 8 文件（ganttUtil+PredEditor+CustomFieldCell+GanttToolbar+GanttTable+GanttTimeline+GanttBars+GanttOverlay），**classNames/data-testid 逐字节照搬**；GanttView 保留 state/memo/handler 组合层（深拆需 context/reducer 超红线）。**坑=四线并发期间 tsc/vitest 会短暂报足迹外错（如 ganttUtil 的 CostResult 中间态），属并发线中间态、稳定后自愈，子代理按「只修足迹内」纪律忽略**。门禁=Go 127 包 0 FAIL/vitest 2737（313 文件，纯拆分零新增测试）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死。欠账=P3 版2（office 抽核 journal/evidence/文件读写→internal/core——office 树 29 处 import 集中 internal/AI 22 文件+根包 3 处已盘点；bridge 双轨退役启动=wailsjsCompat 单 shim 被 57 文件引用；次批 5 巨文件 KnowledgePanel/DeliverablesPanel/XlsxPreview/ChapterPage/herdsmanTemplates）、壳内真机池/审计刀D 不变。


- **最新发布：v4.169.0（2026-09-09）「瘦身 P2 主干：双空间并列落地 + home 空间感知化 + 走查待证项收口」**——瘦身执行层第四版，P2 形态主干收官（masterplan §轨道一·2/3；审计=docs/gaea-slim-p2-dualspace-2026-09.md），三线并发子代理（足迹互斥）+ 主代理收口。**线A rail 双空间分域+切换器**：CommandRail 新增 space/onSwitchSpace props + rail 顶部**工位/乐园切换器**（SHELL_SPACES 驱动、labelKey/titleKey 三语、aria-pressed 激活态）；rail 主体改 `getActiveMenuBoardsForSpace`（shared 恒在/independent 剔除/settings inMenu:false 不在 rail）——**基线 §3.1 rail code 双入口缺陷闭合**（independent 仅 foot 单列，CommandRail.test 断言全 rail 编程入口=1）；manifests.test +4 钉 work/play 派生序、新 CommandRail.test 5 例。**线B home 空间感知化**：Bento 按当前空间过滤（`deriveLauncherModules(…, space)`：shared 保留/编程不再上首页/settings 两空间皆在）+ 新 `LAUNCHER_FEATURED`（work=gaea 办公旗舰/play=chat 会客厅旗舰，bento 动态排除 featured 键防双卡）+ hero 空间 chip + **工位右舷「最近文档」面板**（loadRecentFiles localStorage 单源前 5、点击→办公、零新 binding）；launcher.test +3。**线C 走查项收口**：**knowledge 孤儿页处置**（main.tsx 移除注册+删 pages/KnowledgePage.tsx，全仓 grep 零引用实证；知识库=记忆中枢子面等价，后端 D7 保留导航侧过滤）；**刀2 G-2 基线数关闭**（GschedSummary 增 baselineCount+ScheduleFileCard「基线 N」StatChip，测试 +5）；**后端 nav 子项差异取证**（cost 后端 7 vs 前端静态 4、novel 后端 6 vs 5=有意的双源回退：normalizeManifests `nav: r.nav ? … : base?.nav` 后端优先，运行时以后端为准不改码）。**主代理收口**：三语字典契约键统一（shell.space.* 书房/庭院→**工位/乐园**、home.recentDocs×3、schedCard.baselineCount；progEntry 死键随编程瓦片移除清零，1448 键 0 死）；**坑=locale 死键连锁**（线B 移除 codeModule 宽瓦片后 shell.launcher.progEntry 变死键，主代理三语删）；**Go 测试加固**：TestGaeaGitNotARepo 在 test-all.ps1 的 TMP=.tmp（仓库内）配置下确定性假红——git 向上寻根命中外层仓库、「非仓库」前提被环境破坏；加固=沙箱选址时检测 gitTopLevelOf 命中即改系统用户缓存区另建（t.TempDir 语义不变），双配置全绿（与 v4.164 vitest 抗抖同类：门禁假红一次就教人无视红灯）。门禁=Go 127 包 0 FAIL/vitest 2720→**2737**（313 文件全绿，exit 0）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/vite+wails+冒烟 200。欠账=壳内真机走查（rail 切换器与分域渲染/home 最近文档面板/.gsched「基线 N」布局/knowledge 无残留入口）、审计刀D 真机取证（printSvg print/拖拽/粘贴）、观察池刀3（工作台内嵌办公）待一周评估（knife2 审计 §5 基线评估=中等改造维持）、P4 性能版（entry/MemoryHub 拆解+冷启动基线+exe strip）随阶段推进。


- **最新发布：v4.167.0（2026-09-09）「瘦身 P2 刀0（白名单解耦）+ P0 基线落档 + 审计刀D Filters + nanoid 升级」**——瘦身执行层第二版，三线并发子代理（足迹互斥）+ 主代理第四线 + 收口。**P2 刀0 白名单解耦**：`deriveNavigateWhitelist` filter `inMenu && !isHome` → `!isHome`——**注册即可导航**，inMenu 只控菜单（v4.164 前科根治：藏板块 NAVIGATE 不再被白名单静默丢；刀1 schedule inMenu=false 依赖本语义）；settings 进白名单（有隐式入口）、fixture weixin 进（角色库→青鸟 crossSpace 深链依赖），menuBoards 展示层零改动。**P0 基线落档**：`docs/gaea-slim-baseline-2026-09.md`（基线证据层）——§1 七面四表实测（dist 10.0MB/227、entry 1192.79kB、MemoryHub 1436.11kB 解剖=3d-force-graph 在页/cytoscape+cynefin 独立 chunk、mermaid/katex/CodeEditor 仅宿主视图、exe 46.22MB、npm 直依赖 29、>50KB 源文件 16）、§2 功能等价快照 13 板块×核心动作（纠正 knowledge 非一级导航=后端 D7 被过滤、KnowledgePage 孤儿页入 P2 待证）、§3 IA 走查（rail 12 板块平铺不分空间+foot code 区 **勘出 code 双入口疑似项**、默认空间=work、首页 Bento 无空间过滤、**双空间并列未表达实证成立**仅 3 处空间感知）。**审计刀D Filters 可选参数**（代码化完成）：`GaeaPickFiles(filters string)` 新增可选扩展名过滤（Go 转 Wails FileFilter 前置系统对话框）+ 前端 bridge/mock/pickFile 传参（接受逗号拼接），后置校验保留=fail-closed 双保险；pick_file_filter_test.go 7 例。**nanoid 升级**：审计实证受影响范围 **<3.3.18**（修正任务书 <3.3.8 过时口径），3.3.17→**3.3.18**（纯传递依赖 vite→postcss 免 overrides；npm 11 重写 lockfile 写坏→git restore+手动单条目+dry-run，diff 3 行）；`npm audit` total=0，**GHSA-2v37-7h3g-55p8 清零**。**downloadMarkdown 死代码清理**：export.ts 删除零调用函数，App.export.test 源级断言强化。门禁=Go 115 包 0 FAIL/vitest 2684→**2715**/tsc 0/eslint 0/drift PASS@602（Filters 签名变化方法名零变更）/vite+wails+冒烟 200。欠账=审计刀D 真机取证（printSvg print/拖拽/粘贴）、P2 走查待证项（rail code 双入口/后端 nav 子项差异/knowledge 孤儿页）、P2 刀1/刀2 待续、双空间并列落地随 P2 主干。


- **最新发布：v4.166.0（2026-09-09）「瘦身 P1 快赢：轮子四刀全收 + 审计刀B/C + locale 死键 + 依赖验活」**——瘦身执行层首版（规划=docs/gaea-slim-masterplan-2026-09.md，P1=低风险机械刀热身）。**W1 字节+双门收口**：新中立层 `gaea/lib/bytes.ts`（b64ToBytes，唯一解码规范）+ `gaea/lib/saveFile.ts`（downloadBlob/dataUrlToBlob/saveExportBlob 从 schedule/exportArtifact 迁入，schedule 侧 re-export 零改动）；全仓 b64 手搓解码 ~20 处收口（DocxPreview/docxOutline/docxText/pptxTextDiff/pickFile/schedule api/exportArtifact svgToPdf/TTSPlayer/ChatPage/SchedulePage/useVoiceChat/useImageGenHistory/ImageGenPage…）；**inShell 判定合一**=pickFile.inShellEnv 全仓唯一规范，exportArtifact.inShell 薄委托。**W2 slug×5 收敛**：新 `internal/gaea/strutil/title_slug.go`（TitleSlug=小写/保留 Unicode 字母数字/其余折叠'-'/60 rune 截断/entry 兜底），cost.SlugName/knowledgeimport.slugName/app.slugFromTitle/modelengine.customSlug 并入（customSlug 保留 ASCII 白名单前置、cost 保留 "cost" 兜底语义），controller_memory.slugifyName 为界面文案并入；**legacy golden matrix 测试钉死五个旧实现逐项对照**。**W3 novel diff 收口**：DiffReview 手搓弱贪心行 diff 迁 `lib/diff.ts` LCS（前向 3 行窗口漂移用例锁死）+3。**审计刀B（下载类×4）**：ImageGen 两处（useImageGenHistory/ImageGenPage）壳内 dataURL→Blob 走 saveExportBlob、浏览器回退 a.download；NovelSetting 导入（pickFileAsFile）导出（saveExportBlob）双门；**办公 md 分支**并入 exportConversation 统一交付管线（App.export.test 源级回归锁）。**审计刀C（上传类×3）**：ControlPanel/VisionTrial/SkillModal 壳内走 GaeaPickFiles 系统对话框+扩展名后置校验 fail-closed，浏览器回退原 input（各 +3 用例）。**locale 死键清零**：scripts/slim-locale-deadkeys.mjs（dry-run 默认/--write 落盘；1445 键 0 死，静态 1438+动态 7，en/zh/zh-TW 同步删）。**依赖验活**：26 个运行时直依赖逐个 import 计数——存疑名单 gsap(2)/docx-preview(1)/unist-util-visit(2) 全部洗清；**codemirror 顶包死重移除**（源码零引用、lockfile 零依赖者），CodeEditor 直接使用的 @codemirror/state|view|commands|language 四子包显式进 dependencies（npm 11 再次写坏 lockfile 缺 wasi-threads/tslib 条目——**先例复现：npm uninstall 后必须 git restore lockfile + 手动删条目 + npm ci --dry-run 验证，勿信 npm 11 自动重写**）。门禁=Go 115 包全量绿（复跑清 flaky 同族）/vitest 2684→**2712**（+28，311 文件）/tsc 0/eslint 0/drift PASS@602（零绑定）/vite build 过。欠账=审计刀D（真机取证+Filters 可选参数）、P0 基线快照表入档、nanoid 升级刀、lib/export downloadMarkdown 死代码下轮清。


- **最新发布：v4.165.0（2026-09-08）「拍板落档（路线A+私有许可）+ 壳内残留审计刀A：P0×2 修复」**——§0 拍板落档=路线 A 终极个人工具 + LICENSE 私有 All Rights Reserved（AGENTS 长期规划已回写）。刀A：①新中立层 `gaea/lib/pickFile.ts`（**只 import ./bridge 防域倒挂**；GaeaPickFiles 无过滤器→扩展名后置校验 fail-closed 抛错，调用方 toast）；②BaselinesPanel 签证台账 CSV 改 saveExportBlob（**教训：v4.162 只修了 SchedulePage 主链路，同域新出口 BaselinesPanel 漏网——同组件域新增导出/导入出口必须过统一收口**）；③角色库「添加参考图」壳内接 pickImageAsDataUrl（此前点击无反应且无替代，img2img 立绘链被卡死），浏览器回退原生 input、hidden input 与 jsdom 用例口径原样保留。红→绿取证（修复前 GaeaPickFiles 0 次调用）。vitest 2678→**2684**（全量**首跑全绿**，v4.164 抗抖当场见效）、Go 0 FAIL、tsc/eslint 0、drift PASS@602、build 45.4s+冒烟 200。欠账=审计刀 B/C/D、nanoid 升级刀。


- **最新发布：v4.164.0（2026-09-08）「收敛计划 W1 卫生刀：供应链可复现+测试抗抖+过期导航 id 修复+壳内残留审计」**——收敛计划立项（docs/gaea-convergence-plan-2026-09.md）首轮，三线并发子代理足迹互斥。①package-lock.json 入库+CI npm ci（**坑：npm 11.6.2 --package-lock-only 产物缺条目，须 npm 10 生成或 dry-run 验证**）；②vitest testTimeout 15s+CI 前端 flaky retry（动因=Go/vitest 同机满并发 32 例 5s 超时假红、隔离复跑全绿——**门禁假红一次就会教人无视红灯**）；③修 DeliverablesPanel「回办公面板重新规划」失效：NAVIGATE 载荷沿用历史模块名 `page:"office"` 而 manifest 板块 id=gaea，MainLayout 白名单静默丢弃（**规约：NAVIGATE.page 是壳层板块 id，发送方必须用 manifest id**）；④WebView2 残留审计落档（P0×2：BaselinesPanel CSV/CharacterLibEditor 参考图；刀序 A-D）。零绑定 drift PASS@602；Go 116 包绿、vitest 2678 复跑全绿、tsc/eslint 0、build+冒烟过。欠账=nanoid high 升级刀、审计刀 A-D、LICENSE/性质拍板。


- **最新发布：v4.163.0（2026-09-08）「gaea 长期日志机制」**——此前=单文件 whisper_data/gaea.log（目录误植/无轮转/无上限）。机制（internal/app/logging.go）：**按日分文件 `<DataRoot>/logs/gaea-YYYYMMDD.log`**（当日 O_APPEND），启动清理过期（**保留 365 天**）+总量兜底（>1GB 从最旧删），旧 gaea.log 一次性迁入 logs/gaea-legacy.log；启动头写 version+Go 版本，Shutdown 记运行时长；前端诊断（GaeaLogFrontendError→slog）自动落同一文件；新绑定 **GaeaOpenLogsDir**+设置→数据「打开日志目录」按钮。**⚠附带修 v4.162 缺陷：ReadFileB64/SaveFileAs 漏 facade 委托行致运行时不可达——Wails 只暴露 Bind 里门面结构体的方法，App 直挂方法必须同文件在任一门面（bindings_*.go）加委托行；gen_bindings -names 只扫方法名探测不到漏委托，接手后新绑定必须壳内实测一次调用可达。****坑：①壳内页面调试=WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333 + CDP 受信任点击（合成 el.click() 在 rc-trigger 行为不同）；②原生对话框探测 UIA #32770 漏报，须 P/Invoke EnumWindows 按 pid 枚举；③壳内首页入口是 [role=button] div 非 button 标签，冷启动 >10s 要轮询。**绑定 601→603、vitest 2677、tsc/eslint 0、drift PASS@603、版本三处 4.163.0、build+冒烟过。欠账：日志查看 UI 按需另立小刀。

- **最新发布：v4.162.0（2026-09-08）「进度计划导入导出壳内修复：原生对话框链路」**——用户实测「导入导出点击没有反应,包括菜单栏这些也是」。**根因=Wails WebView2 壳内 `<input type=file>.click()` 不弹文件对话框 + `<a download>` 下载不落盘**（浏览器 dev/prod 全正常,故测试全绿不可见;菜单/下拉本身正常）。修复:新绑定 **GaeaReadFileB64**(读所选文件,64MB 上限)+**GaeaSaveFileAs**(系统另存为+写盘) 599→601;导入四路壳内走 GaeaPickFiles+ReadFileB64 还原 File 喂原解析器(浏览器回退 input.click,hidden input 保留 jsdom 口径);导出五路统一 saveExportBlob(壳内另存为/浏览器 download)。**诊断配方（复用价值）：①分层排除=浏览器 dev/prod ✓ 壳内 ✗ → 壳特有;②壳内 DOM 调试=WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333 启动 + Node25 内置 WebSocket CDP,受信任点击必须 Input.dispatchMouseEvent（el.click() 合成事件在 rc-trigger 行为不同）;③原生对话框探测=UIA #32770 RootElement 搜索漏报,须 P/Invoke EnumWindows 按 pid 枚举;④壳内首页入口是 [role=button] div 非 button 标签,冷启动渲染 >10s 要轮询。**绑定 599→601、vitest 2671→2677、tsc/eslint 0、drift PASS@601、版本三处 4.162.0、build+冒烟过+壳内 CDP 实测对话框弹出 ✓。欠账：window.print 壳内行为未走查;全 app `<input type=file>`/`<a download>` 残留扫尾（其余入口多为绑定通路）。

- **最新发布：v4.161.0（2026-09-08）「双代号对齐标杆二期：工程标尺 + 图面语言」**——继续对齐重庆干休所标杆件。①**时间坐标框架进视图**：新纯函数 `aoaRuler.ts`（顶部工程日刻度/月/日+底部星期/工程周,列=工作日/步进按总工期 5-10-20/月标签落本月首个工作日/工程周每 7 列一档），月界竖线贯通图面；标尺与图面同一 scale 容器随缩放对齐；手动布局不画（x 与时间解耦口径）。②**波形线独立着色**：`segsToPathSplit` 拆独立 path 着绿 #22c55e（标杆图例语言）,`segsToPath` 保留兼容包装（d+' '+wave 逐字符等价旧输出,PdmView/既有用例零改动）,导出件同步。③**图面语言**：关键事件红圈(es=ls)/名称工期蓝色系/非关键线加深 #475569,图例同步。④**修 v4.160 带内打包阶梯化 bug**（目检实锤：按节点序打包把合并链拆成阶梯①②③逐级下降）——改按**全局重心行道 gRow** 压缩打包（带内行号=该带升序 gRow 序号）,链保持一条水平线。**坑：①v4.160 的行距「恰按公式」断言与打包模型耦合——布局断言写不变量（等距/放大/封顶）;②esbuild 目检 harness 必须放 frontend/ 内（node_modules 解析）,--bundle 对 css import 自动产同名 .css 手写占位会被覆盖;③Edge 无头截图（--window-size）是 SVG/组件目检的最保真轻量路径,MuPDF 直渲 SVG 黑白反转只核结构。**目检方式=esbuild 隔离挂载 AoaView+Edge 截图（releases/v4.161.0.md 有配方）。零绑定 599、vitest 2671→2677、tsc/eslint 0、版本三处 4.161.0、build 过。欠账：汇总总线双线样式观察池;标尺真机观感走查挂池。

- **最新发布：v4.160.0（2026-09-08）「双代号分级横幅：一级/二级分区布局」**——用户以重庆干休所上报件 PDF 为标杆（PyMuPDF 渲染逐象限目检提炼画法）：**每个分部工程一条横幅**——分部汇总线=横幅顶部通长线（分部名标线上，summarySegs 三段正交：竖起→横贯→竖落）、二级工序只在横幅内排布、跨分部一律虚工作竖向衔接、交替底纹分带。落地：①buildAoa band 归属（一级分组行=横幅,事件归成员最小 band=跨组共用界点归前组;组前未分组并头部合成横幅,组后并最后分组横幅;无分组=单一横幅退化旧行为）+band 内保序打包（沿用全局重心序为带内相对序,带间 1 行空档+首行半行帽）,行距自适应改按打包总行数;②**跨横幅不合并事件**（FS/SS 合并限同带+非首横幅无前置自设开始事件+START 虚工作补入）——否则后学分部挂前学分部事件→任务边跨带穿行标注压字（目检实锤）;0 工期虚工作不改时间参数/关键判定。**坑：①造样例数据任务 id 必须唯一,重名→byId 后者覆盖→链接解析成环假报「循环依赖」;②MuPDF 直渲 SVG 黑白反转文字糊只可核结构,保真目检用 Edge 无头截图（--force-device-scale-factor + window-size 控制幅面）;③v4.159 行距「恰按公式」断言在打包后失效——布局断言写不变量（等距/放大/封顶）勿复刻公式。**视图/导出同口径接底纹（sched-aoa-band-N testid）。零绑定 599、vitest 2665→2671、tsc/eslint 0、版本三处 4.160.0、build 过。欠账：用户真实计划横幅观感真机走查挂池;工程标尺四行视图侧未加（导出侧已有,按需另立小刀）。

- **最新发布：v4.159.0（2026-09-08）「进度计划画布手感：滚轮缩放 + 双代号行距自适应」**——用户实测反馈两处（附截图）。①**三块画布补滚轮缩放**：新共用钩子 `schedule/wheelZoom.ts`（普通滚轮=光标为锚缩放,Shift+滚轮=原生横移;**锚定=记光标下图面坐标、缩放提交后 useLayoutEffect 回写 scrollLeft/Top,同步回写会被旧内容尺寸钳制;监听器必须原生挂 {passive:false},React 合成 onWheel 是 passive preventDefault 无效;applyZoom 走 ref 中转免监听器反复拆挂**）;横道=Ctrl+滚轮缩日宽（MS Project 口径,**普通滚轮保持滚动行**,dayG 锚定+dayWRef 镜像）。②**双代号布局松绑**：AOA_ROW_H 74→112（旧 74px 行距扣掉节点上下时间标注 56px 只剩 18px 空隙=「全挤在一起」根因）+新 AOA_ASPECT_MAX=7 行距自适应（时标宽被工期锁定,低并行长计划天然 20:1 扁条——宽高比超 7 放大行距,clamp [ROW_H,2×ROW_H],y 与时间无耦合波形语义不变）+通道层距 12→16（层距须容「名称上/工期下」两标注,12px 叠压=截图红字互压根因）;PdmView 补 clientWidth≤0 适配防护（v4.143 jsdom 0 宽纪律同款）。**坑：el.dispatchEvent(WheelEvent) 不经 React act,断言读不到 setState 渲染——滚轮用例须 fireEvent.wheel;snapPt 网格随常量变 55×56,历史 pin 绝对坐标不受影响。**零绑定 599、vitest 2658→2665、tsc/eslint 0、版本三处 4.159.0、build 过（Go 仅版本常量,go build+app 包绿）。欠账：滚轮手感/自适应行距真机走查挂池。

- **最新发布：v4.158.0（2026-09-09）「AI 组价复核闭环：确认留痕 + 拆解合理性校验」**——造价板块接续（两路并行调研子代理：域代码盘点 file:line/需求材料核对）。**重要校正：AI 组价 v4.2 已在产**（组价→检索→价格带→LLM 拆解→确认回写全链路,gaea_cost_compose.go/priceband.go/ComposeModal）,roadmap §15「无嵌入索引/无异常检测」等欠账描述已过时——现状基线落档 **docs/gaea-cost-domain-survey-2026-09.md**（含刀路池:条目匹配键/流式分步/五算贯通/询价库级扫描/索引主动维护/含量对照基线）。本刀销两真缺口:①**SchemaV16 cost_compose_records**（契约写 V11 但实际迁移链已到 V15,子代理正确顺延——**接手契约先核对迁移链实际编号**）确认即留痕完整视图快照,无确认不落库红线不变,留痕尽力而为不阻断回写;GaeaCostComposeRecords(entryName 空=全库最近 20)回看,坏 snapshot 行如实 nil;CostEntryModal「组价依据(N 次)」只读折叠区,证据链表抽共享 ComposeEvidenceTable(ComposeModal 零视觉变化迁用);SQL 落点=app 层直写+cost.Store.DB() 一行句柄委托(**hubCostStore 测试注入路径与生产一致是关键约束,勿绕 store 直取 db.GetDatabase 打真实用户库**)。②**cost.CheckComposeComponents** 纯函数(R1 金额一致性/R2 非正值(不进 R1R3 防双重误报)/R3 合计×1.05 warn ×0.5 info,恰阈值不触发)进 View.checks,ComposeModal 行级徽标(同行归并 warn 主导)+全局提示,校验随留痕入档=回看时见「当时提示过什么」。**Go 全量绿(+18)、vitest 2650→2658、tsc/eslint 0、drift PASS@599、版本三处 4.158.0、build+冒烟过;flaky=app 包 TempDir 清理竞态+GitPanel 全量并发 28 败(均单跑/复跑绿,在册同族)。欠账:组价依据区 mock 下恒空真机走查挂池;刀路池见 survey 文档。**

- **最新发布：v4.157.0（2026-09-08）「Office 编辑链一致性小刀：docx 证据链补齐 + pptx 面板字符级对比」**——两个小项收口 v4.156 三件套编辑线（零新绑定 598）。①**docx_apply 补证据链**（pptx 设计 §6 拍板项 4 推荐项）：GaeaDocxApplyEdit 落盘前快照（.gaea/work/rollback/docx-<ts>.before,尽力而为不阻断）+Journal（Tool=docx_apply,摘要 truncateStr 120,BaselinePath 如实）——**ApplyTrackedReplace 编辑语义零改动**,docx 编辑从此可经 GaeaRollbackRecord 回滚（VersionTimeline kind:"docx" 已支持）;GaeaDocxAcceptChanges 仅 accept==true（清除修订标记=不可逆整理）加同款,accept==false 不加（拒绝=恢复原文天然回滚）。②**PptxEditPanel 对比区升级 ChangesDiff**（刀3 层3 在面板侧接通）：docxTextDiff 句级 LCS 输出归一 DiffRow——changed=相邻 del+add 对交 pairModifications 改蓝配对+字符级高亮,未变句给 ctx 行 foldContext 折叠不伪造全量;删「句级对比（无字符级高亮）」降级标注,「不支持修订标记可恢复」常驻标注保留;状态机/应用逻辑零改动。**坑：gaeaEffectiveSpace() 单测缺省返回 ""≠"work",守卫注入先例=直赋 ga.cfg=&gaeaConfig.Config{}（零值→mode on+space work,gaea_pptx_test.go:248 先例）,Workspace 留空使 gaeaCwd() 回退 os.Getwd()=t.Chdir 目录。**Go app 包全绿（+3:apply 快照+Journal 字段逐项/120 截断口径/accept 两分支对称）、vitest 2650 全绿、tsc/eslint 0、drift PASS@598、版本三处 4.157.0、build+冒烟过。**欠账:pptx 刀2 真机走查仍挂池;刀4 修改队列待反馈;Verifier 通道 B 对 docx_apply 的复核口径未动。**

- **最新发布：v4.156.0（2026-09-08）「pptx 真编辑刀2+刀3：编辑面板 + 版本对比（Office 三件套编辑闭环）」**——docs/gaea-pptx-edit-design-2026-09.md 刀2/刀3 三线并行交付（Go 绑定线/前端刀2 线/前端刀3 线,足迹互斥;刀1 数据层早于 v4.109.0 落地,设计文档状态行曾过时——**接手先核对设计文档状态 vs git log**）。**绑定 597→598**（+`GaeaPptxSlideText`=pptxedit.LocateText 段落全文,纯 Go 零 python）：动机=大纲 texts 是 200 rune 截断预览,直接当 apply 的 target 必匹配失败（ApplyTextReplace 单段落精确匹配、宁拒不误改）——段落粒度即 target 粒度。①刀2 编辑面：PptxOutline 两级导航（页→文本框,页锚点/composer 插入/降级全保留,未接线宿主编辑入口不渲染）;新组件 **PptxEditPanel**（P1=xlsx 同款 Plan→Apply:载入段落全文失败诚实降级→段落单选→预设四动作+自定义指令（附加约束「短句化、长度相近防溢出」拼 instruction,GaeaOfficeEditText 零改动复用）→双栏对比（docxTextDiff LCS 句级整块着色,诚实标注无字符级）→应用（后端错误原样透出,成功用返回 PreviewResult 刷宿主+静默重拉）→常驻「PPT 不支持修订标记,应用即生效可回滚」）;FilePreview pdf 分支挂面板（DocxQueuePanel 宿主先例）。②刀3 对比：lib/pptxTextDiff.ts（JSZip 解 slides 文件名自然序=显示层近似（后端 sldIdLst 放映序,注释明示）;页签名 LCS 对齐+changed 页复用 diffDocxParagraphs 段落级;坏 zip 结构化 {ok:false} 降级）;versionCompare kind:"pptx" 字段对齐 XlsxDiff;**取数通道偏离 docx=GaeaAttachmentDataURL**（GaeaPreview 对 pptx 渲染成 PDF 逐页缩略且基线快照 .before 扩展名不走 Preview 分派——docx 的 Preview-dataUrl 通道两侧都拿不到字节,照抄即死代码）;VersionTimeline PptxCompareBody 页摘要+差异页 hunk（改蓝配对+字符高亮白得）。**坑：①gaea 域无 lib/api.ts——组件直调 app.*（DocxPreview 先例）,契约偏差如实记录;②mock Preview 无 .pptx 分支（v4.28 起即如此）→?mock 下编辑链路不可达,真机走查挂真机池（组件状态机 23 测试覆盖）;③bindings_office.go 头部标注 generated——手加委托后 gen_bindings 再生成一致（drift PASS@598 实证）;④TestCreateChapter_SameChapterConcurrentRejected 全量跑偶发 TempDir 清理竞态（Windows unlinkat not empty）,单跑×3+整包复跑绿=在册 flaky 同类。**Go 全量绿（+3）、vitest 2617→**2650**（+33）、tsc -b/eslint 0、drift PASS@598、版本三处 4.156.0、build+冒烟过。**欠账：真机走查（mock 不可达）;刀4 修改队列泛化待刀2 反馈拍板;docx_apply 不落证据链（设计 §6 推荐另立小刀）;面板内字符级高亮未接（句级诚实降级）。**

- **最新发布：v4.155.0（2026-09-08）「双工期欠账放开：SS/FF/SF×cd 全搭接（§3.2.5 重推，零不动点）」**——双工期设计 §6 欠账池首项销账：v4.150 刀1 曾 fail-closed 收窄「cd 任务仅允许 FS 搭接」，本刀按 §3.2.5 要求实施前重推语义矩阵——**原矩阵四格「⚠ 需不动点」均系 wd 代数形误判，实际零迭代可解，八格全放开**。①新原语 `cdEarliestStart`/`CdEarliestStart`（单调逆查：最小 s 使 cdToEf(s)≥T；估锚 max(0,T−cd)+有界双侧校正，与 cdToEf/cdLatestStart 互为镜像三件套）解 FF/SF-to——fwd 关于 es 单调不减有平段，逆像确定存在；②SS/SF-from **直接 ls 下界**（SS/SF 约束本就落在开始上 `ls≤to.ls−lag`，不经 lf+durFrom 换算——换算才是原矩阵担心的不动点源）+后继 **effLf**=cd 后继的 cdToEf(ls)（FF/SF 对前驱传播读实际最迟完成才诚实；v1 全 FS 组合永不触及，旧 pin 零风险）；③逆推 from=cd 拆双上界：lf 收 FS/FF、lsBound 收 SS/SF，`ls=min(cdLatestStart(lf), lsBound)`、lf 保留 raw（兼容「fwd(ls)≤lf」pin）；④wd 路径逐位不变（FS 项无条件读 to.ls 与旧式恒等，快路径用例零回归）。⑤拆闸三处：Go Validate「仅 FS」/引擎防御拒绝/analyze 防御 finding，cd 余四闸不动；文案四处同步（scheduleApply 描述/技能条款（锚点锁 37 不变）/types 注释/甘特 chip title）。**测试：锚点表×6（cdToEf/cdEarliestStart/cdLatestStart×cd=5/7，平台 0-2→5、5-7→10）+用例 A-E 两侧镜像断言 es/ef/ls/lf/tf/critical/duration；golden 62→65（cd+SS(lag2)/SF(lag1)/upsert cd+FF 成功例）。坑：①契约用例 D 原稿把 SS 正推期望误写成 FS 式数字（es=5 只有 from.ef+lag 能得出）——两侧子代理独立发现按定稿语义收敛同组修正值，镜像无损；写引擎测试契约时正推/逆推期望必须按同一搭接型自洽；②既有「cd 非 FS fail-closed」用例物理上不可能零改动，原位翻转为 ok=true 是正路。**Go 全量绿（+2 函数/5 子用例）、vitest 2601→**2617**（+13 TS+3 golden 双跑）、tsc -b/eslint 0、drift PASS@597、版本三处 4.155.0、build+冒烟过。设计文档 §3.2.5/§6 已回写重推结论；欠账池剩=导入任务日历转 cd 迁移助手/斑马导入口径探测/等效跨度灰显双读数（按需）。

- **最新发布：v4.154.0（2026-09-08）「MPP 导入支持 Project 2013+ 变体（真机样本驱动修复）」**——用户真实工程文件（重庆干休所总进度计划，Project 2013+ 保存）此前被 ParseMpp 拒收（「缺 CompObj」），强拆后系统性错乱。**字节级取证钉死 2013+ 变体五处漂移**：①新版无 CompObj 流→工程目录编号判版回退（`   114`=MPP14；无 CompObj ⇒ appVer≥15=lag@14 口径）；②任务工期 i32@42→**@84**（@88 恒同值；全行偏移扫描唯一全行命中且全为整天）；③Var2Data 回链键 uid@0→**id@4**（uid 是陈旧键与名字表大面积错位——27 空名根因）；④parentUID@36 恒 0 失效→**大纲层级 i16@172**（0 项目/1 总账/2 分部/3 分项与 WBS 逐行吻合）+层级栈重建父链；⑤FixedMeta byte8 恒 0xe3 失效→里程碑=零工期语义。搭接 pred/succ 确认同为 ID 键空间（45 行全命中），lag@14 数值合理（14d/27d）。**样本现已完整导入：38 任务/8 分组/42 搭接/总工期 136 天/任务名零缺失，CPM 自洽（192 自然日≈136 工作日×7/5）。**诚实降级：2013+ 资源/分配行键位未钉死（解析出垃圾资源），宁缺勿错暂不解析留观察池；日历例外日仍不解析（v4.140 口径，原文件 150 天 vs 引擎 136 天差异主因）。**坑：①Fixed2Data/Fixed2Meta 不是任务第二表（是字段定义/陈旧快照，勿当主数据源）；②全行偏移扫描=定位漂移字段的最快取证法（候选偏移取「全行命中+业务数值合理」交集）；③gated 样本落 clones/mpp2013/（git-ignored）。**TestMppRealSamples2013 门控落档（38/8/42/136 钉死），旧三样本（MPP9/12/14-2010）零回归；Go 全量绿、版本三处 4.154.0、drift PASS@597、build+冒烟过。

- **最新发布：v4.153.0（2026-09-08）「双工期口径刀4：mspdi 互通」**——双工期四刀**全部收官**（纯前端 597 零变更，Go 零改动）。①导出：cd 叶任务 `<DurationFormat>8</DurationFormat>`（elapsed 日历天）+ `PT{自然日×24}H0M0S`（28 自然日=PT672H）；wd 恒 7+PT{d×8}H 逐位不变；分组行恒工作日口径；Start/Finish 本就 wdToDate 日历日期自动吻合。②导入：DurationFormat=8 → `durationUnit:'cd'`（分钟÷1440）；其余/未知格式维持 ÷480 工作日折算宽松容错；里程碑/汇总不标 cd（normalize+引擎 fail-closed 双保险）；`parseDurationDays`→`parseDurationMinutes` 按格式分流（FORMAT_*/MINUTES_PER_ELAPSED_DAY 常量显式化）。③**真机池（本机未装 MS Project，沿用「等真机」先例）：Project 打开导出文件对 8 的显示与重算、`PT×24` 序列化假说钉死（XSD xsd:duration 语义允许，若有出入仅影响 Export 端常数——Import 端 ÷1440 与分钟解析两口径兼容）、WPS/斑马行为；真机走查通过前 Export 端对 Project 实际兼容性视为待验证，Import 端是失真修复方向性收益确定。**vitest 2596→**2601**（+5：导出口径/roundtrip/Format8 导入/未知格式回落/elapsed 里程碑不标 cd）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。**双工期口径设计（durationUnit wd|cd）四刀全落地：模型引擎/agent 通道/板块 UI/mspdi 互通；拍板池三项（diff 确认卡/多工程/双工期）全部收官。**

- **最新发布：v4.152.0（2026-09-08）「双工期口径刀3：板块 UI」**——纯前端 597 零变更，Go 零改动。①工期列「日历」chip：cd 行常显（primary ghost 点击切回工作日）、**wd 行 hover 行时浮现**（CSS 渐进披露=「wd 行不显后缀」条款落地）、分组行/里程碑无入口；点击循环 wd↔cd **不换算数值**（拍板项 7，28 wd→28 自然日）；工期列宽 48→84。②拖拽缩放：drag.ts 新 **`resizeToNaturalDuration`**（右缘=新 ef 工作日吸附不变，回写**自然日差**=offsets[newEf]−offsets[es]，引擎复算 ceil 吸附不动=所见即所得）；**右缘起点 `dayNo(origEs+origDur)`→`dayNo(row.ef)` 顺带修复 cd 条形画过宽**（自然日数曾被当工作日序号加——条宽改按 dayNo(ef)−dayNo(es) 真实日历跨度，wd 两值逐位相等零回归）；拖拽浮标「日历 X → Y 天（自然日）」按自然日直绘；move 语义不变（manualStart 恒工作日）。③口径透明：条形悬停「日历天 28（等效 20 工作日，日期区间）」等效只读（不做双值列拍板项 7）；前锋线 frontierWd 的 dur 换源 ef−es；检查器工期行「28 天（日历，等效 20 工作日）」；状态栏「总工期 N 天（工作日）」；单代号节点/双代号箭线 cd 标「(日历)」+两图图例注记（**AOA 网格恒工作日刻度，cd 箭线长度仍自然日数=计算不经视图诚实标注**）。**坑：①交互测试必须 setState store project 与渲染 prop 同源（拖拽/点击写 store，渲染用 prop——不同源则断言打到旧工程）；②全量 vitest 并发 flaky 28 败（在册先例），单跑+复跑两轮全绿再发布。**vitest 2589→**2596**（+7）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。**刀4 mspdi 互通（DurationFormat 8 导出导入，绑真机池——不真机不发布）待续。**

- **最新发布：v4.151.0（2026-09-08）「双工期口径刀2：agent 通道」**——刀1 引擎能算 cd 但 agent 说不了这个话，本刀打通对话链路（设计 §3.5，零新绑定 597）。①ops 通道：patchTask.DurationUnit 指针三态（缺 key=不动、**空串 no-op 同 Mode 先例**、非法枚举报错原文）；**单位分支先于工期分支**——「工期 28→30」词尾随新单位（cd 显「日历天」），cd 态工期超 3650 当场拒；回执口径词「工期口径→日历天（自然日定时）」/「工期口径→工作日」；upsert_task 的 Task 直带 durationUnit（零值缺省=wd，校验同 Validate 五闸，新增回执「工期 28 日历天」）。②**Go/TS 对拍 golden +12 例（49→61）**：单位切换/显式 wd/空串 no-op/分组行/里程碑/超上限/非法枚举 + 「cd 全链 upsert+FS+基线」（cd 进 SnapshotBaseline 的等效跨度口径由对拍钉死），opsSim.golden.test.ts 63/63 过——语义漂移即炸对端。③三工具：schedule_get taskView 带 durationUnit（omitempty，wd 不出=旧客户端零噪音）；schedule_analyze 增 **cdTasks 清单**（id/name/duration 自然日数/span 等效工作日跨度/anchor 锚点日期=es 所在工作日）+ qualityChecks 增 cd 非 FS 防御 finding（闸在 Validate/apply，疑旧口径残留提示改 FS）；schedule_apply 回执「写入后总工期 X 天（工作日）」补词。④schedule-edit 技能：工期分节增 cd 纪律（养护/干燥/成活期自然日定时才标 cd/搭接仅 FS/成本按等效跨度 ef−es=元/工日不因日历天放大/定额合同工期禁误标）；汇报口径补「总工期 X→Y 天（工作日）；混排计划先讲 cdTasks」；**锚点锁 35→37**。compact 三工具同步。**Go 全量绿（+3：ops 口径词/引擎 cdTasks/工具链路 TestScheduleDualDurationChain）、vitest 2576→**2589**、tsc -b 0、drift PASS@597、build+冒烟过。**刀3 板块 UI（工期列单位切换+chip+悬停等效提示+拖拽 resize 换算）→刀4 mspdi 互通（绑真机池，不真机不发布）待续。**

- **最新发布：v4.150.0（2026-09-08）「双工期口径刀1：任务级工期单位（模型+引擎）」**——拍板池最后一项欠账启动实施（docs/gaea-schedule-dual-duration-design-2026-09.md 推荐项全采纳）：`task.durationUnit?: 'wd'|'cd'` 缺省 wd=旧文件零迁移零行为变化；cd=日历天（养护/干燥类自然日定时，对齐 mspdi DurationFormat 8），28 个自然日误录 28 工作日的 +8 工作日失真消除。①换算纯函数对 `calendar.ts cdToEf`（正推边界=es 所在工作日+cd 自然日，**ceil 吸附**到首个 ≥边界的工作日序号；26/27/28cd 整周工作制下同 ef=20 吸附折叠）/`cdLatestStart`（逆推最大 s 使 fwd(s)≤lf：镜像回退+有界双侧校正，性质 fwd(ls)≤lf<fwd(ls+1) 钉死；平段取最大 s；负时差极端截 0）。②cpm 增可选 ctx `{calendar?, startDate?}`（Go 先例式双入口 `ComputeCpmCal`/`ComputeCpmPlanCal` 委托同一 `computeCpmFull`）：**快路径铁律**=无 cd 任务分支全短路逐位一致（既有用例零改动全绿+逐位等价用例）；有 cd 而**缺开工日期 → ok=false**（日历缺省回落周一~五=对设计 §3.2.2 的有意识细化，与全部 wdToDate 读点同口径）；cd 涉非 FS 引擎层同样拒（Validate 之外防直接调用）；逆推 FS 上界统一 `to.ls−lag`（wd 后继与旧式逐位相等，cd 后继必须用已换算的 ls）。③成本/基线换源：work 分配与基线快照/漂移 dur 改用 **cpm 行等效工作日跨度 ef−es**（wd 两值恒等有 pin；养护期周末不记工日，28cd×100 元/工日=2000 非 2800；日历变化改 cd 等效跨度时 durDrift 如实呈现=诚实口径）。④校验：Go Validate 五闸（枚举 wd|cd/分组行禁 cd/里程碑禁 cd/数值上限 3650/搭接仅 FS）+TS normalizeProject 容错回落（违规丢字段回 wd）。⑤接线零行为变化：TS 六处 computeCpm 调用点传 ctx（SchedulePage/store/applyDiff/gschedSummary/mock/opsSim——opsSim 与 Go set_baseline→SnapshotBaseline 镜像对齐），Go 四处（Save/Analyze/SnapshotBaseline/schedule_tools Before 摘要）。**坑：①ComputeCosts 收 Project 值不收指针（测试样板函数别返回 *Project）；②grep 中文偶发 GBK 显示乱码——file/python 复核字节，非真损坏勿慌；③发布索引四处（releases/README/README 表/CHANGELOG 头插/AGENTS 速览+一行账）+版本三处（sync-version.ps1 一把梭）。**刀2 agent 通道（patch_task.durationUnit+cdTasks 清单+技能条款）→刀3 板块 UI→刀4 mspdi 互通（绑真机池）按序待续。Go 全量绿（+10）、vitest 2550→**2576**（+26）、tsc -b 0、drift PASS@597、版本三处 4.150.0。

- **最新发布：v4.149.0（2026-09-08）「diff 确认卡刀D：ops 通道投影模拟器 + Go/TS 对拍收官」**——diff 确认闭环设计**四刀全部收官**（A 回滚/B diff 卡/C 引擎强制/D 投影对拍）。①`schedule/opsSim.ts` `simulateOps(before, ops)`=Go internal/schedule/ops.go ApplyOps 的逐语义 TS 镜像：深拷贝顺序应用 fail-closed（单条失败即中止、半途状态不外泄）；指针三态（缺 key=不动、显式 0/空串=改、mode 空串 no-op）、upsert_task 零值语义（缺 level=0 分组/缺 progress=0）、remove_task 扁平数组 level 子孙级联+搭接清理、set_links 整体替换缺省 FS+自前置/悬空拒绝、set_meta 日期 10 字口径+deadline 空串清除/null 不动、auto_chain 手动跳过+无可补报错、set_baseline 缺 savedAt 拒绝+无 planFinish CPM 口径、资源级联/重复对/外来 taskId/坏金额全对齐。②**Go/TS 对拍主阵地**：internal/schedule/ops_golden_test.go 生成并校验 `frontend/src/schedule/ops_golden.fixture.json`（49 例=21 成功+28 失败路径，覆盖 12 op 全守卫分支；`go test -run TestApplyOpsGolden -update-golden` 重生成），opsSim.golden.test.ts 同文件双跑逐例比对「是否失败+after 状态」——**任何一侧语义漂移都炸对端测试**；canonical 比对消化 Go omitempty 与 TS undefined 表达差（空数组双侧同丢）。③卡体：ops 通道缺省路径即 simulateOps 投影→diffProjects 同管线真 diff（「ops 投影」来源标注），三级诚实降级（非缺省路径/读取失败/模拟失败→意图清单+原因）；落盘权威永远在 Go。**坑：①jsdom 里 import.meta.url 非 file 协议读不了 fixture——改 resolveJsonModule（tsconfig.app.json）+直接 JSON 导入；②canonical 不丢空数组则 Go omitempty 空切片与 TS [] 必炸；③control 包同包测试的 strPtr 等指针助手已存在，新增要换名（gStrPtr）；④if 语句头复合字面量加括号 (gateApprover{c}).Approve。**Go 全量绿、vitest 2499→**2550**（+51）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。

- **最新发布：v4.148.0（2026-09-08）「diff 确认卡刀C：project 整量替换引擎强制逐条确认」**——schedule_apply project 通道（args.project 非空=整计划替换/生成）在 `gateApprover.Approve` 增分支：hardAskSet 判定之后、auto 快捷判定之前 → `requestApproval(alwaysPrompt=true)`——任何权限级别（含 auto/yolo）前置逐条弹卡、不读不写会话放行（拍板项 2 禁会话记忆自然落地）；ops 增量通道维持现状闸门（拍板项 3：auto/yolo 豁免，后置回执+回滚兜底）。`approvalSubjectFor` 增 project 分支 `scheduleApplyProjectSubject`：解析 args.project→`schedule.Project.Analyze()`（同一引擎口径）→「整计划替换（目标文件）：N 项工作，总工期 Y 天，关键 K 项」；CPM 不过标注「批准也将被引擎拒绝」；before 侧对比不进 subject（闸门无工作区上下文，由前端 diff 卡承担）。前端 ApprovalModal：`scheduleApplyIsProjectChannel`（applyDiff.ts 镜像判定）→ project 卡按 hardAsk 三钮形态渲染（hardConfirm 统一分支），ops 五钮不变。schedule-edit 技能 Body 增 **「确认与回滚」** 分节（project 逐条批准/被拒=未写入且禁止原样重发/超时=拒绝如实告知/以回执为准/回滚入口=工具卡「回滚本次」+变更 tab），锚点锁 **31→35**；scheduleApply.Description 尾部+compact.go compactDesc 同步确认机制。**坑：①if 语句头用复合字面量 gateApprover{c} 必须加括号 (gateApprover{c}).Approve（vet 抓）；②control 包 import internal/schedule 无环（引擎是叶包）可放心；③不动 event.Approval 的设计红线——前端 hardAsk 形态用 args 判定镜像，强制权威在 Go。**刀D（simulateOps TS 镜像+Go/TS 对拍）待续。Go 全量绿（+5 闸门用例）、skill 锚点过、tsc -b/eslint 0、vitest 2499、drift PASS@597、build+冒烟过。

- **最新发布：v4.147.0（2026-09-08）「diff 确认卡刀B：schedule_apply 审批卡升级为结构化 diff 预览」**——diff 确认闭环设计刀B（§2.2）。①纯函数层 `schedule/applyDiff.ts`：`diffProjects(before, after)` 两侧 normalizeProject 归一后纯比较——任务 id 键控增删改（字段级 from→to 中文 label：任务名/工期/进度/层级/里程碑/模式/手动开始/固定成本）、搭接按 to 分组入边比对（对齐 set_links 整体替换语义）、资源删除标级联分配数、分配 (taskId,resourceId) 键控增删改、meta 四项；汇总行=总工期/关键/总成本双方 computeCpm/computeCosts+倒排校核+基线漂移预览；**after CPM 不过如实标注「引擎将拒绝」（预览层复刻 fail-closed）**；empty 判定。`scheduleApplyArgsOf(items)`=宿主从 Transcript 取 running 工具卡 args（§1.3 ToolDispatch 先于闸门）。②`ScheduleDiffCard` 卡体 + ApprovalModal 新 prop `scheduleApplyArgs`（approval.tool==="schedule_apply" 时替换通用参数原文区，决策/快捷键/hardAsk 不动；双宿主各传各的 items）。降级三级：非缺省路径（loadScheduleFile 只服务缺省）/读取失败/ops 通道（simulateOps=刀D）→ 意图清单+「无现状对比」诚实标注；卡头「预览数字，落盘以回执为准」；60 行截断+展开全部。**坑：①i18n schedDiff.* 键位有意识偏离不做了——卡体是 schedule 域内容层（§405 zh-only），未新增 shell 键；②组件文件只许导出组件（react-refresh），scheduleApplyArgsOf 放 applyDiff.ts 纯函数层；③全角空格进 JSX 文本踩 no-irregular-whitespace；④CpmResult 从 types 导入非 cpm.ts；SchedCalendar 字段名 holidays 非 exceptions（TS 会抓）。**刀C（Go approvalSubjectFor project 分支=整计划替换 hardAsk 化+锚点锁 31→N）→刀D（simulateOps+Go/TS 对拍）待续。vitest 2483→**2499**（+16）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。

- **最新发布：v4.146.0（2026-09-08）「对话改计划回滚闭环（diff 确认卡刀A）」**——v4.125 落档的 diff 确认闭环设计（docs/gaea-schedule-diff-confirm-design-2026-09.md）按刀序启动，刀A=后置回滚补全（设计原文：即使确认卡不做也独立成立）。①变更 tab：WRITE_TOOL_NAMES/WRITE_ONLY_TOOL_NAMES +schedule_apply（三态完备性锁同步满足）；`extractChangedPaths(args, tool?)` 增可选 tool 参——schedule_apply 缺省 path 回填「进度计划/当前计划.gsched.json」（与 Journal target 同口径，缺省路径整计划替换不漏记）；buildChangeDiff 显式降级说明不伪造行级 diff。②新组件 `ScheduleApplyRollback`：schedule_apply 回执卡 done 后常驻「回滚本次」行（ToolCard 内 TaskLiveRow 同位、不进折叠区）——Journal 按 target 匹配**最新一条带基线快照**的记录（ChangesPanel 同款语义），点击调既有 GaeaRollbackRecord（「已被手工修改」守卫在 Go 侧）；无匹配整行不渲染；多次 apply 同路径不逐卡区分（Journal 无调用级关联键，标题带 turn，设计 §5 欠账另案）。**坑：①新刀插入文档头又吃掉上版标题（第三次！）——「old_string=上版标题行」的插入方式必须把标题原样带回 new_string 尾部，已两次靠事后核对 head 挽救；②缺省路径常量在 changes/组件本地镜像不 import gschedSummary（防拉起 schedule store 图谱，DEFAULT_SCHEDULE_PATH 先例）；③旧测试锁「WRITE⊆EDIT∪WRITE_ONLY」，加白名单必须同步选边（schedule_apply→WRITE_ONLY）。**刀B（diffProjects+ScheduleDiffCard+ApprovalModal 变体）→刀C（Go 审批闸门 project 分支+锚点锁）→刀D（ops 投影+Go/TS 对拍）按设计刀序待续。vitest 2477→**2483**（+6）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。


- **最新发布：v4.49.0（2026-09-02）「青鸟助手生命周期补全」**：git tag
  `v4.49.0`；基线 v4.48.0；绑定面 559 零变更。「继续」双线并行子代理：
  ①编辑已有助手（PersonaPickerPanel 抽出复用，自定义人格原值保留，gaea 禁改）；
  ②角色卡一键创建未绑定助手（custom 卡专属）；③镜像守卫——EnsureAssistants
  跳过 custom 命中行（v4.48 隐患：助手 personalityId 指向 custom 角色会在
  启动时被镜像冲掉名字/kind）。详见 releases/v4.49.0.md。

- **最新发布：v4.48.0（2026-09-02）「青鸟：板块更名 + 人格选择器」**：
  git tag `v4.48.0`；基线 v4.47.0；绑定面 559 零变更。用户拍板两件：微信助手
  更名「青鸟」（前后端 label/意图别名/页内文案，board id 不动）；新增助手
  人格选择器（轻语预设+角色库可聊天角色双栏选择、详情预览、立绘回显、18+
  过滤）。CharacterList/WhisperGetPersonalities 转正 AppBindings（work 分面）。
  详见 releases/v4.48.0.md。

- **最新发布：v4.47.0（2026-09-02）「微信助手星枢化：通讯枢纽工作台」**：
  git tag `v4.47.0`；基线 v4.46.0；绑定面 559 零变更。UI 重构刀（功能零删减）：
  三卡堆叠 → 星枢三分区（玻璃细条+左通道轨道+主区三视图：通道详情/离线提醒/
  使用指南）；通道状态三重传达（状态点四态+状态字+键值卡）；扫码流三步指示；
  dev mock 补微信域（mock/weixin.ts）；设计蓝图 design-system/gaea/pages/weixin.md。
  详见 releases/v4.47.0.md。

- **最新发布：v4.46.0（2026-09-02）「小说板块第二轮：伏笔闭环 · 导出扩展 · 伴读多轮 ·
  一致性 AI 深检」**：git tag `v4.46.0`；基线 v4.45.0；**绑定面 557→559**（+SaveForeshadows、
  +CheckConsistencyDeep）。①伏笔登记表闭环（写入口+syncForeshadows 按 ID 合并+面板
  登记/流转/编辑/删除）；②导出扩展（EPUB 真封面/onlyMainline/DOCX via gooxml）；
  ③伴读问书多轮化（historyJSON+划线窗口+会话式弹窗）；④一致性 AI 深检 v0（状态卡
  提取+本地比对五类矛盾+source 徽标+诚实降级）。**小说侧剩余欠账：ChapterPage 拆分、
  branches.json 历史版本、搜索高亮落划线**。详见 releases/v4.46.0.md。

- **最新发布：v4.45.0（2026-09-02）「百炼全量下线：引擎 + 配置 + UI + 微信改图链」**：
  git tag `v4.45.0`；基线 v4.44.0；绑定面 557 零变更。**用户拍板「把百炼删除干净」**：
  引擎（image_dashscope 退出注册表）/配置（DashScopeAPIKey 字段+迁移+initImageBackend
  分支全删，SetImageBackend 第 5 参彻底移除）/UI（引擎枚举去 dashscope，ResultStage 新增
  「改图」动作用 ComfyUI/Herdsman 走 img2img）/微信改图链（v4.9 起：ActionEditImage 意图
  解析、execEditImage+seam、入站图缓存 wx_image_cache、wx_agent edit_image 工具 7→6、
  editImageFromCard）全量退役。基建：gen_bindings 空白参数名合成占位名。详见
  releases/v4.45.0.md。

- **最新发布：v4.44.0（2026-09-02）「绘梦专项三刀：百炼模型残留修复 · 引擎
  枚举单源化 · 模板画幅落地」**：git tag `v4.44.0`；基线 v4.43.0；绑定面 557
  零变更。绘梦板块专项摸底后收三条互不相交线（无并行子代理，主代理直改 +
  测试收口）：①**百炼模型残留修复（真 bug）**——dashscope 后端
  GetImageBackendInfo/SetImageBackend 空或残留模型（grok-imagine-*/krea2）归位
  qwen-image-edit-plus（手填官方编辑系保留；GetImageBackendConfig 同口径）；
  前端 useImageGenConfig modelOptions 对 dashscope 固定三档官方编辑模型 +
  切后端 defaultModel 归位 DASHSCOPE_DEFAULT_MODEL；queue 提交前
  backendSupportsMode 拦截引擎固有模式残留（百炼仅改图 / GLM 仅文生图），不再
  点击后才被后端拒收。②**引擎枚举单源化 + 补 GLM**——meta.ts 收敛唯一
  BACKEND_OPTIONS（能力位 img2imgOnly/txt2imgOnly）+ backendLabel/
  isLocalBackend/backendSupportsMode 辅助；ControlPanel 下拉、ImageGenPage 顶条
  状态 pill、useImageGenConfig 启动消息、GenerationBar 门禁统一走单源（修复
  「☁️ xAI 云端 云端」式拼接重复与 dashscope/glm 无 label 回退裸 id）；引擎下拉
  新增 GLM（txt2imgOnly，非文生图模式禁用 + Tooltip「GLM 仅支持文生图」+ 残留
  专属警告）。③**模板推荐画幅落地**——新纯函数 templateSizeToPreset(ratio,mode)
  把模板 size 比例标签映射到实际画幅（2:3 立绘无预置档→自定义 768×1152，仅
  txt2img 生效）；applyTemplate 应用模板时同步画幅；TemplatePickerModal
  handleSelect 不再丢弃 size 字段（此前只传 label/prompt/negative）。
  **验证**：Go 全量 test exit 0（+8 用例：GetImageBackendInfo dashscope 归位
  ×4/SetImageBackend 残留归位 ×4）；tsc -b/eslint 0；vitest 1285/1285（+11：
  meta.test 7 + ControlPanel GLM 4）；drift PASS（557）；版本四处 4.44.0。
  **新欠账**：ControlPanel GLM 未启用/无 Key 的切换前置引导（现由后端报错兜底）；
  dashscope 图生图 UI 内「参考图+指令」一键改图入口仍缺（v4.40 沿旧）；百炼
  txt2img（qwen-image-2.0 系）后端未放行。详见 releases/v4.44.0.md。

- **最新发布：v4.43.0（2026-09-02）「小说板块优化四刀：生成上下文 · 分支收账 ·
  搜索定位 · 一致性修复」**：git tag `v4.43.0`；基线 v4.42.0；绑定面 557 零变更。
  小说板块自 v4.3.1 后首次专项投入（四条互不相交线并行子代理，零 UI 结构变更）：
  ①章节生成上下文增强（prompt 追加未回收伏笔/世界观要点区段+角色卡放宽补
  身份/目标/关系+摘要 200/续写尾 1500 rune，4000 rune 总预算双层截断，读失败
  静默跳过）；②分支链路收账（branches.json 持久化，ApplyBranch 读存储主路径
  零 AI 重调，syncCharactersFromOutline no-op 变真同步）；③全文搜索升级（每章
  全部命中+段级位置，前端共 N 处·M 章+跳章滚段临时高亮）；④一致性纯规则两
  bug（分支章节纳入扫描带 Branch 标记、断档不停扫）；死代码清理（未挂载分支
  组件 4 件+孤儿 api/search.ts）。**新欠账：伏笔登记表写入口；Continuity Linter
  AI 化；EPUB 封面/导出格式扩展；ChapterPage 拆分**。详见 releases/v4.43.0.md。

- **最新发布：v4.42.0（2026-09-02）「微信智能体 v1：LLM 工具调用派发」**：
  git tag `v4.42.0`；基线 v4.41.2；绑定面 557 零变更。**定位用户拍板**：微信消息
  由模型自己派发任务给各板块（关键词路由太落后），v4.41.x 两轮正则补丁即其账。
  实现：wx_agent.go 7 工具（导航/生图/改图/提醒/产物推送/状态/读屏）零改动复用
  exec 函数 + ChatStream tools 透传（ai 管道既有）+ 4 轮/60s 循环 + 坏参数喂回
  自纠 + 人格记忆同锁语义（失败 CloneFullState 回滚防双计数）+ 能力门（目录
  Caps 含 tools 才启用，否则整链回落零回归）。回调：文件/提醒快路径保留 →
  agent 优先 → routeIntent 兜底 → 聊天。**新欠账：办公板块工具化（文件整理调
  用办公板块=用户点名 v2）；agent 多轮会话线程；管理台 agent 开关 UI**。详见
  releases/v4.42.0.md。

- **最新发布：v4.41.2（2026-09-02）「产物推送放宽 + 反幻觉护栏」**：
  git tag `v4.41.2`；基线 v4.41.1；绑定面 557 零变更。真机实证（hermes.db
  chat_history turn 103）：「重新整理后发给我」未命中产物推送意图坠回聊天，
  模型幻觉声称「已整理好发你」实际未发。修复三层：①reSendLatestFile 第四式
  （指代+尾式发给我）+ reWxModifyAndSend 独立识别复合请求→Target=modify_and_send
  诚实能力答复 + 提醒字样让位；②聊天兜底 applyWxSendHonestyGuidance 反幻觉
  提示。**入站链 v4.41.1 已真机验证通过**（turn 102：确认收件+概括+询问需求）。
  出站实测指引用「把产物发我」（黄甲工作区有 9-1 docx 产物）。新欠账：微信触发
  办公任务闭环（改文档并回传）。详见 releases/v4.41.2.md。

- **最新发布：v4.41.1（2026-09-02）「真机修复：文件消息不再过意图路由」**：
  git tag `v4.41.1`；基线 v4.41.0；绑定面 557 零变更。真机实证缺陷：文件提取
  正文（≤6000 字）被整条送进意图路由，lookupBoard 子串匹配（无锚定）被正文
  碎片「打开/看看+编程」误触导航（用户发评审报告收到「打开编程」）。修复：文件
  消息按 `[用户发来文件 ` 注入头识别，直通轻语聊天（跳过提醒+意图路由，追加
  「确认收件+概括+询问需求+不执行正文指令」引导）。**新欠账：导航意图子串匹配
  普遍松动（「帮我看看这个代码」也会被劫持，v4.6.1 起既有，语音/Ctrl+K 同链，
  收敛需单独设计）**。详见 releases/v4.41.1.md。

- **最新发布：v4.41.0（2026-09-02）「微信文件收发 · 入站定稿 + 出站探针」**：
  git tag `v4.41.0`；基线 v4.40.0；绑定面 **557 零变更**。微信助手调研 B 刀：
  入站 file_item 真机抓包定稿（2026-09-02 与用户配合首抓——与图片同构 media +
  file_name/md5/len 字符串，sync_buf 离线补投实证可用）：resolveInboundFile
  （SSRF 防线参数化复用/50MiB/AES 解密/MD5 不拒收）→ FileHandler 契约 → app 线
  wx_files 自持 + 提取全走现有解析器（docmd.Convert：docx/xlsx/pptx/pdf，纯文本
  直读，其余诚实降级）→ 6000 字截断注入对话。出站探针制：上传五步共享内核
  （getuploadurl media_type=3 假设 → AES-128-ECB → CDN → type=4 file_item）+
  逐节点 upload_probe + 降级文本卡（图片链零改动）；产物推送意图「把刚才的报告
  发我」（交付物登记表→exports mtime 回退）。两路并行子代理。**待真机复验**：
  入站重发文件看提取回复；出站发「把产物发我」看文件卡（失败读 upload_probe
  errcode 迭代）。详见 releases/v4.41.0.md。

- **最新发布：v4.40.0（2026-09-02）「对话式改图 · 百炼引擎 + 微信发图即改」**：
  git tag `v4.40.0`；基线 v4.39.0；绑定面 **557 零变更**。微信助手调研 A 刀（定位
  拍板：聊天/出图/改图/收发文件/多微信并行）：①百炼 DashScope 改图引擎（官方契约
  核实后实装：同步免轮询多模态端点、单图单文、qwen-image-edit-plus 默认、仅
  img2img、改图不传 size 保原图比例、24h URL 下载转 data URL；kind=dashscope）；
  ②dashscope_api_key 密文落盘（SetImageBackend 追加第 5 参空=保留存量，方法名不变）；
  ③ActionEditImage 动词∧指代双门槛保守正则 + 入站图旁路 hook（OCR/vision 链路
  零改动红线）+ 助手级图片缓存（wx_edit_cache 自持副本 TTL 10min 只留最新）+
  routeIntentForAssistant 内部变体（未命中/面板/语音不接管回落聊天）+ 产物 CardPath
  图片卡回推；④绘梦「百炼改图」选项 + img2img 门禁三处白名单 + 设置页 Key 框。
  实施三路并行子代理（引擎/接线/前端），收口把 EditImageFromCard 改未导出保 557
  （教训：门面完整性测试会逼导出方法进绑定面——内部链路方法直接小写开头）。
  **下一刀**：B 文件收发（file_item 抓包硬前置，可先抓包）。详见 releases/v4.40.0.md。

- **最新发布：v4.39.0（2026-09-02）「微信助手管理台 · 多微信并行 + 并发正确性两修」**：
  git tag `v4.39.0`；基线 v4.38.0；绑定面 **557 零变更**。微信助手专项调研
  （docs/market-research-2026-09-02b.md）C 刀。**定位用户拍板纠偏**：微信助手=通过
  微信与 gaea 对话进行各项工作（聊天/出图/改图/收发文件/多微信并行）——本刀收多微信
  并行：①WeixinPage 管理台（助手卡 Avatar/人格/状态徽标 + 启停 + 删除 + 逐助手扫码
  绑定/重绑修硬编码 gaea + 新增助手表单 wx_ 动态 id；gaea 禁删禁停）②manager.Update
  补回写 WxBotID/PortraitURL/VoiceGuide/Gender/Tags/Dims（空值保留防清空）
  ③WhisperAssistantSave 空 token/userId 保留旧凭据 ④同人格多助手 AssistantName
  锁外直写竞争修复（内部 whisperChat 链透传 name，注入入 LockTurn 窗口，绑定签名
  零变更）。两路并行子代理实施（Go 后端线/前端线所有权互斥），主代理定契约收口。
  已知边界：同人格多号仍共享会话上下文（多号建议独立人格）。**下一刀序**：A 对话式
  改图（Qwen-Image-Edit 云端引擎+改图意图+InitImage 接线）→ B 文件收发（file_item
  抓包前置）。详见 releases/v4.39.0.md。

- **最新发布：v4.38.0（2026-09-02）「目录通用化 · DeepSeek/xAI/Zen 官方元数据入册」**：
  git tag `v4.38.0`；基线 v4.37.1；绑定面 **557 零变更**。用户指出 v4.36.0 目录只覆盖
  GLM——通用化：model_catalog.json v1（deepseek 3/xai 7/opencode-zen 15 条目，官方页
  多页互证）+ estimatePrice 目录优先层扩展（deepseek-v4-flash 估算 3.0→12.672 CNY 修正、
  grok-4.5/4.6 129.6→57.6、zen 8 模型新增计价；claude/gpt/gemini/kimi 旧条目回归锁逐位
  一致）+ fetchModels 动态列表 enrich（id 归一化匹配只填空不覆盖，徽标自动点亮前端零改动）。
  **拍板**：opencode-go 订阅制无按量售价不入目录（防误导），引擎级「订阅制」徽标留欠账。
  内置表过时暴露：deepseek-chat/reasoner 官方 2026-07-24 停用、grok-4/3 系已下架。
  详见 releases/v4.38.0.md。

- **最新发布：v4.37.1（2026-09-02）「模型中心四刀：自定义引擎/目录 v2/巡检转移/D 收口」**：
  同日四刀（模型中心专项调研 docs/market-research-2026-09-02.md → 用户拍板 ABCD）。
  **v4.35.0 自定义引擎**（552→555）：EngineCustom OpenAI 兼容任意服务商（添加/更新/删除），
  Key 存 config 加密 map custom_engine_keys，BaseURL 校验防 Key 粘错框（v4.9.1 防线延伸），
  聊天路径 custom 分支真可聊天。**v4.36.0 GLM 目录 v2**（555 零变更）：glm_catalog.json
  44 条目 schema v2（上下文/价格/免费档/能力 caps/coding 积分系数，官方查不到的绝对价
  不编数沿用内置 z.ai USD 估算口径）+ 远程热更新 v0（glm_catalog_url 默认禁用，仅影响
  展示与估算不碰路由）+ estimatePrice 目录优先单源化（估算值零回归锁）+ 前端能力/价格徽标。
  **v4.37.0 健康巡检+故障转移 v0**（555→557）：10 分钟周期探已启用非本地引擎（Error 永不含
  Key）+ engine_failover_enabled 开关（默认关，网络类/5xx 才转移、用候选默认模型重试一次、
  流式仅首字节前）+ 前端开关卡/双事件订阅。**v4.37.1 D 收口**（557 零变更）：卸载确认带
  释放大小；磁盘展示与 /health 透出经核实已存在剔除（伪欠账再验证）。**教训**：
  ①gen_bindings 的 explicitOverrides 是显式门面归属清单——新增绑定方法必须先登记再跑
  生成器，否则被前缀规则误归（C 刀实测：手写 ModelB 委托被生成器覆盖进 OfficeB）；
  ②build.bat 同链 git commit 中文可能乱码，用 -F UTF-8 文件 amend。**验证**（累计）：
  Go 全量 test exit 0、tsc -b/eslint 0、vitest 1243/1243、drift PASS（557）。
  **欠账**：A2 帧流/接管、B2 pptx 真编辑、降噪折叠、palette 个性化、task 卡空 ref、-race、
  model-failover 文案 engineLabel 化、dev mock 补 failover 开关、自定义引擎用户价目（目录 v3）。
  详见 releases/v4.35.0.md ~ v4.37.1.md。

- **最新发布：v4.34.0（2026-09-02）「子代理气泡恢复 · 恢复会话不再丢失子代理答复」**：
  git tag `v4.34.0`；基线 v4.33.0；绑定面 **552 零变更**。收 v4.26 沿旧欠账。**根因**：
  恢复链投影 session.ProjectMessages 无 subagent_message case 整条忽略；**模型面投影
  不可动**（投影直接喂模型，恢复后模型上下文须与实时语义一致）→ **UI 侧并行投影**：
  session 新导出 `ProjectSubagentAnchors`（与 ProjectMessages 逐 case 同拍游标镜像，
  subagent_message 记「插在第 K 条消息后」锚点；projection.go 零改动）+ GaeaHistory 读
  磁盘事件日志 `mergeSubagentAnchors` 合并子代理气泡（**logOffset 校正**：检查点 Snapshot
  含日志从不投影的 system 提示，provider History 系统性多于投影，不校正锚点会提前一档；
  越界锚点宁漏勿误丢弃）；HistoryMessage 加 `subagentRef`（golden 不变）；前端
  rebuildHistoryItems 透传复用实时徽标渲染。**教训**：①「UI 视图 ≠ 模型上下文」——同一条
  日志流两套投影，UI 侧补投影绝不能顺手改模型侧；②子代理抓出 brief 假设漏洞（provider
  History ≠ ProjectMessages(entries)，差 system 提示一档）——派 brief 里的核心假设要
  标注「待验证」。**验证**：Go 全量 test exit 0（投影/Restore/golden 回归全绿）、tsc -b/
  eslint 0、vitest 1210/1210（+3）、drift PASS（552）、wails generate module 已刷新。
  **欠账**：沿旧 A2 帧流/接管、B2 pptx 真编辑（独立刀体量）、降噪折叠、palette 个性化、
  -race。详见 releases/v4.34.0.md。

- **最新发布：v4.33.0（2026-09-02）「细节收口 · 第三刀：回滚守卫统一/pdf 占位比精确化/
  主区预览懒加载对齐」**：git tag `v4.33.0`；基线 v4.32.0；绑定面 **552 零变更**。三并行
  子代理+主代理集成。①**回滚守卫统一 + write_file >8KB 恒误报修复（真 bug）**：rollback
  卡接入「恢复后已被手工修改」守卫（撤销恢复前校验防覆盖编辑）；write_file 守卫精确比较
  vs 落库截断 SummaryLimit(8KB) 必然误拒 >8KB 未手改文件，改 `evidence.ClampSummary`
  同口径截断比较（导出单点化，RecordChange/app 复用；字节口径含切进 UTF-8 中间字节的
  历史行为）；已知边界=8KB 摘要窗口外手改不可检（宁漏勿误）。②**pdf 占位比按实测精确
  化**：pageLazy 新 `nextPageAspect`/`placeholderAspect`（img onLoad 读 naturalWidth/
  Height，首个有效测量为整档比例不被推翻，无测量回落 A4），弹窗占位→真身交换不再跳高。
  ③**主区预览 pdf 懒加载对齐弹窗**：FilePreview pdf 分支接入同款 IO 单向懒加载（初始
  4 页/800px/不卸载/大纲跳转强制渲染/ref 回调登记即补 observe），主代理追加测量比例
  接线。**核实**：v4.26「TrajectoryView 未消费 subagent」欠账已过期剔除；**子代理气泡
  恢复暂缺**=GaeaHistory（provider History 无来源字段）×事件日志（ResyncEvents 已折叠
  subagentRef）跨源对齐，留独立刀先出方案。**验证**：Go 全量 test exit 0（线A -count=2）、
  tsc -b/eslint 0、vitest 1207/1207（+11）、drift PASS（552）、版本四处 4.33.0。**欠账**：
  rollback 守卫 8KB 窗口外边界；独立刀候选=子代理气泡恢复；沿旧 A2/B2/降噪折叠/palette
  个性化/-race。详见 releases/v4.33.0.md。

- **最新发布：v4.32.0（2026-09-02）「细节收口 · 第二刀：回滚可撤销/产物自动弹出/弹窗
  pdf 懒加载/预览最大化持久化」**：git tag `v4.32.0`；基线 v4.31.1；绑定面 **552 零变更**。
  用户点名「继续优化完善 gaea」——欠账池四条互不相交线，三并行子代理+主代理 App.tsx
  接线。①**回滚先快照当前态**（收 v4.28 B1 欠账）：GaeaRollbackRecord 恢复前快照目标
  当前内容（evidence 新导出 `StageBaselineTo`，命名逻辑单点化），rollback 记录升级完整
  证据卡——**恢复动作本身成为时间线里可再恢复的版本**；目标缺失/快照失败降级不阻断；
  取舍=rollback 卡不接「手工修改」守卫（>8KB 基线截断后精确匹配恒误报会阻断合法恢复），
  数据已就位待后续方案。②**产物自动弹出+偏好**（收 v4.30 欠账）：`gaea.deliverableAutoOpen`
  **默认关** opt-in（新 lib/deliverablePrefs 对齐 browserPrefs）+ DeliverablesPanel 头部
  胶囊 + App 新产物 diff 自动切「产物」tab（尊重 tab 停用态、激活即清零角标、不动
  FilePreview）；单版本徽标 title 细化「有 N 个历史快照」。③**弹窗 pdf 逐页懒加载**（收
  v4.31 欠账）：新 lib/pageLazy 纯函数 + IO 单向懒加载（初始 4 页/800px 预挂/已挂载不
  卸载），大纲跳转目标页强制渲染，无 IO 全量降级；顺带修 preview/loading 两次异步提交
  致 IO 观察集为空、懒加载永不触发的真 bug（ref 回调登记即补 observe）。④**预览最大化
  持久化**（收 v4.30 欠账）：`gaea.previewMaximized` 独立简单键（writePrefs 只落 sizes
  数字 map）+ App 三处落盘。**集成教训**：tsc -b 抓到子代理线 ReadonlySet/Set 类型口径
  三处不一致（返回类型诚实放宽修净）。**验证**：Go build/vet/全量 test exit 0（线A
  -count=2）、tsc -b/eslint 0、vitest 1196/1196（+30）、drift PASS（552）、版本四处
  4.32.0。**欠账**：rollback 卡守卫未接（见①取舍）；弹窗 pdf 占位高为 A4 估计值；沿旧
  v4.28 全部 + v4.30 降噪折叠/palette 个性化 + -race 无 gcc。详见 releases/v4.32.0.md。

- **最新发布：v4.31.1（2026-09-02）「-count>1 全量绿化 · 测试全局态 -count 不兼容根治 +
  whisper 末气泡真 bug 修复」**：git tag `v4.31.1`；基线 v4.31.0；绑定面 **552 零变更**
  （纯测试治理 + 1 处生产修复）。v4.31.0 线 D 收尾延伸：全量 `go test -count=2 ./...`
  从 FAIL → 全绿（exit 0），含生产修复按先例独立成版。**根因（统一）**：测试用固定 /
  `t.Name()` kind 或固定会话 ID 写入**进程级全局状态**（provider/billing/boot/app 注册表、
  app `whisperSessions` 会话缓存），`-count` 多次运行不兼容（非产品缺陷，whisper 除外）。
  **修法**：①注册 kind 改 `testKind(prefix)`（进程级 atomic 单调计数后缀，任意 -count
  唯一；19 注册点：provider/billing/boot/app 各加 testkind_test.go）；billing 无菌态断言
  「==[deepseek]」改「含 deepseek」（语义词保留）；②app whisper 会话隔离改唯一会话 ID +
  `cleanupWhisperSession`（t.Cleanup 删进程级 whisperSessions 缓存，与 proactive/
  persist_concurrency 模式一致，12 调用点）；③**whisper 真 bug（唯一生产改动 +3 行）**：
  `PacedStreamEmitter.pump` 在 streamDone 且文本已全部送出时不收尾末气泡 → `MarkDone`
  排空等待因 bubbleOpen 恒真永久挂起（全量负载实测 10m 超时；**生产表现为最后一个气泡的
  OnBubbleEnd 永不触发**）——streamDone 分支收尾末气泡（仅 bubbleOpen 时收尾，主代理验收
  无副作用）。**验证**：五包 -count=2/-count=5 全绿（app 逐层 3 类问题修净）、发射器
  -count=300 全绿、tasks -count=20 仍全绿、**全量 `go test -count=2 ./...` 与 -count=5
  ./... 双绿 exit 0**（主代理复跑确认）、tasks -shuffle=on -count=10 无顺序依赖、go vet
  全绿；前端零改动（vitest 1166 沿用）；drift PASS（552）；版本四处 4.31.1。
  **欠账**：-count>1 既有问题清零；-race 仍不可用（无 gcc）；沿旧 v4.28 全部 + v4.30/v4.31
  欠账。详见 releases/v4.31.1.md。

- **最新发布：v4.31.0（2026-09-02）「细节收口 · 四线并行：单版本入口/弹窗 pdf 预览/历史轮
  耗时/tasks 竞态根治」**：git tag `v4.31.0`；基线 v4.30.0；绑定面 **552 零变更**（结构字段
  级，零新增绑定）。用户点名「开始，并行使用子代理」——从欠账池挑四条互不相交的线（文件
  足迹互斥），四并行子代理分线实现 + 主代理集成收口（v4.25/28 同款工作流），全量门禁：
  ①**产物版本时间线单版本入口**（线 A，收 v4.28 B1 欠账）：徽标条件 `{rev && …}` 放宽为
  `{(rev || journalEntry) && …}`——versions>1 按现状 vN 徽标（旧锁不破），versions≤1 但有
  journal baselinePath 快照的产物渲染「版本」入口徽标（title 区分「更新 N 次」/「有版本
  历史」），无快照保持空态；VersionTimeline 本体零改动。测试 30→33。
  ②**FilePreviewModal pdf/pptx 逐页预览**（线 B，收 v4.28 欠账）：方案 b 弹窗内内联（
  **FilePreview.tsx 本体零改动**）——kind=pdf 分支逐页缩略（data-pptx-page 锚点）+dataUrl
  整本回退+诚实空态+PptxOutline 大纲卡（页锚点滚动/「针对第 N 页修改」composer 插入）；
  非 pdf 分支逐字节未动。测试 7→10。
  ③**轨迹历史轮耗时**（线 C，收 v4.26 欠账）：后端 Turn.DurationMs（fold turn_done 分支
  Ts>StartedAt 时 =(Ts−StartedAt)×1000，omitempty 向后兼容，golden 逐字节不变）+前端
  TrajectoryTurn.durationMs+TrajectoryView 轮次头「用时 Ns」（复用 formatElapsed，仅
  turn.end&&durationMs 时显示）；零新增绑定（GaeaTrajectory 返回 struct 字段级）。测试
  Go+2、前端+3。WorkHeader 消费历史轮耗时不可行（只吃实时 store 无 trajectory 数据源），
  已跳过记欠账。
  ④**TestCancelConcurrentStress flaky 根治（线 D，实现层真竞态）**：根因=pickNext 不做任务
  级预留→同一 queued 任务多 worker 同时 execute；claim 落选者无条件 unregisterCancel 删
  cancels+cancelReq（用户取消意图）→Cancel 已成功返回的任务终态被 succeeded 吞掉（v4.8.2
  回归锁违背，探针轨迹恒 [queued running stopping succeeded] 实锤）。修复=tasks.go 新
  clearStaleCancel（只清残留预注册、**绝不删 cancelReq**，m.mu 下查状态仍 running/stopping
  归胜者不动）+claim 成功后胜者重登记 cancel；测试改事件驱动等待（50 终态事件到齐），断言
  不削弱（Cancel==nil ⇒ cancelled + 至少一个 Cancel 成功防空转）。验证 -count=20/100 全绿。
  **验证**：Go build/vet/test 全量 0 FAIL；tsc/tsc -b/eslint 0；vitest **1166/1166**（+9：
  A3+B3+C3，未删改旧锁）；drift PASS（552）；版本四处 4.31.0。**欠账**：WorkHeader 历史轮
  耗时未消费；弹窗 pdf 不虚拟化；单版本「版本」徽标静态文案；tasks -count>1 既有全局注册表
  duplicate kind 与 whisper 超时沿旧；沿旧 v4.28（A2 帧流/接管、B2 pptx 真编辑、子代理气泡
  恢复暂缺/中途进度不回投）+v4.30（预览最大化持久化、产物自动弹 tab）。详见
  releases/v4.31.0.md。

- **最新发布：v4.30.0（2026-09-02）「办公 UI 化繁为简 · 第二刀：产物置前/行级降噪/
  命令面板视图重排/预览两档」**：git tag `v4.30.0`；基线 v4.29.0；绑定面 **552 零变更**
  （纯前端呈现重组）。用户点名「继续优化完善 gaea」——从 v4.29.0 欠账清单收齐四项，
  红线不变（简化≠删除功能，被隐藏的信息全部有确定性寻回路径：title/aria/悬停/激活即见）：
  ①**产物生成自动置前/角标**（Devin Auto-open 式）：App diff 会话内新产物路径（首现即新，
  会话切换重置基线——恢复会话不误标「新」）→ 产物 tab 角标（未查看数，激活即清零，与
  运行角标同语义）+ DeliverablesPanel 新 `freshPaths` prop（经 sidebarRegistry ctx 接线）
  → 行「新」徽标（Sparkles）+ data-fresh 高亮；②**面板行级降噪**（Cowork 一行式）：产物/
  变更/任务三列表次级信息（路径/相对路径/时间/重试计数）group-hover 悬停次行显现
  （opacity 150ms），title 全保留，主行断言零改动；③**命令面板按当前视图重排**（Linear
  式）：新 lib/paletteRank.ts 纯函数 rankPaletteItems(items,{chatTab,rightTab})——当前激活
  右栏面板 cmd 置顶 > chatTab=overview 时概览置顶 > 其余面板命令 > 模板/会话保序（稳定
  排序）；App paletteItems 接线，CommandPalette 组件零改动；④**预览「半幅↔最大化」两档**
  （VS Code Toggle Maximized Panel）：icons 补 Maximize2/Minimize2（antd Fullscreen 系列），
  FilePreview 头部按钮（不传 onToggleMaximize 不渲染，向后兼容），App previewMaximized 状态
  （最大化=占满可用宽度 视口−侧栏−360 与拖拽上限同源，还原回半幅 ref 记忆，**拖拽分割条
  自动退出最大化**）。**验证**：Go build/vet 0 FAIL（零 Go 变更）；tsc/tsc -b/eslint 0；
  vitest **1157/1157**（+10：paletteRank 6+DeliverablesPanel 新徽标 2+FilePreview 两档 2，
  未删改旧锁）；drift PASS（552）；版本四处 4.30.0。**欠账**：产物自动弹 tab（激进版
  Auto-open）暂不做可加偏好；行级降噪仅悬停次行未做折叠重构；命令面板个性化排序远期；
  预览最大化不持久化；沿旧 v4.28 全部。详见 releases/v4.30.0.md。

- **最新发布：v4.29.0（2026-09-02）「办公 UI 化繁为简 · 顶栏收拢/自适应标签/
  预览降噪」**：git tag `v4.29.0`；基线 v4.28.0；绑定面 **552 零变更**（纯前端
  呈现重组）。用户点名主轴「UI 界面化繁为简，参考市场同类产品」，派刀即立红线
  **「简化界面不是删除功能！」**。按纪律先跑模块制调研两线（AI 代理工作台简化
  模式 / 办公文档工具与生产力软件）：原始稿 `docs/research-2026-09-01b/` +
  合成 `docs/market-research-2026-09-01b.md`——共同结论：化繁为简=把复杂度从
  「常驻视觉空间」迁到「按需检索空间」（菜单/面板/悬停/折叠）+ 出现时机半自动
  化，Linear「只隐藏数据，不删除 issue」即官方先例。三个交付点（功能与出口
  全保留）：①**顶栏导出收拢**——新 ExportMenu，导出 Markdown/Word/PDF 三
  常驻文字钮收进单钮「导出 ⌄」下拉（对标 Devin/Linear 只进菜单不加按钮 + VS
  Code 单点溢出），三出口管线原样接线，顶栏常驻操作钮 7→5；②**右栏 tab 窄栏
  自适应图标化**——容器 <420px（WORKSPACE_TAB_COMPACT_WIDTH）文字 CSS hidden
  只显图标（aria-label/title/角标保留；ResizeObserver 缺失回退文字；compact
  受控 prop 可测），宽栏恢复文字，**6 tab 集合与数量锁不动**，340px 基线宽
  拥挤根治（对标 Notion 视图 tab Icon only/Text only）；③**预览头部降噪**——
  FilePreview「打开/定位」图标化（title/aria-label 保留）+头部按钮统一去边框
  （HEAD_BTN），「编辑/保存/取消」状态语义文字保留（编辑能力保留红线，测试
  钉住）。**验证**：Go 110 包 0 FAIL（stress flaky 沿旧）；tsc/tsc -b/eslint 0；
  vitest **1147/1147**（+9：ExportMenu 5+WorkspaceTabs compact 3+FilePreview
  1；未删改任何旧锁）；drift PASS（552）；版本四处 4.29.0。**欠账**：产物
  自动置前/角标；面板行级次级信息悬停次行化；palette 吸附右栏面板项+按视图
  重排；预览「半幅↔最大化」两档；沿旧 v4.28 全部。详见 releases/v4.29.0.md。

- **上一发布：v4.28.0（2026-09-01）「浏览器与版本 · 观察窗/版本时间线/pptx
  交互」**：git tag `v4.28.0`；基线 v4.27.4；绑定面 550→552（+2：
  GaeaPptxOutline / GaeaBrowserObserve）。规划「浏览器与版本」刀（A2+B1+
  B2/C3），三并行子代理分线（文件所有权互斥）+主代理集成：
  ①**A2 浏览器观察窗**：browser 包 Manager.Observe()（CDP captureScreenshot
  jpeg ≤1280 等比缩、未运行 Available=false 绝不拉起）+ 绑定
  GaeaBrowserObserve（browser.Default() 同源、seam 可测）；右栏 running 组
  第 6 tab「浏览器」（BrowserPanel：URL/标题+截图 zoom+帧龄+操作时间线
  browser_* 倒序 20+权限静态行+自动弹出胶囊 gaea.browserAutoOpen；App 接线
  新 browser_* 工具自动切 tab；2.5s 可见门控轮询）。帧流/接管远期。
  ②**B1 版本时间线**（纯前端零 Go）：产物 vN 徽标可点→内联
  VersionTimeline（groupVersionsByPath 聚合倒序过滤无快照卡）+ 基线预览
  （GaeaPreview abs）+ 恢复（RollbackRecord=写回基线+追加新证据卡）；完全
  长在证据链上（对标 Notion 版本史/Artifacts rewind，预览即护栏）。留白：
  单版本无入口、RollbackRecord 不先快照当前态（待后端补）。
  ③**B2/C3 pptx**：绑定 GaeaPptxOutline（python-pptx 逐页大纲，失败结构化）
  + GaeaPreview .pptx 分支（soffice→PDF 缓存 .gaea/cache/pptx-preview 7 天
  TTL + poppler 逐页 PNG ≤60 页 → kind=pdf 复用前端渲染）+ 前端逐页预览+
  PptxOutline 大纲侧栏+页锚点滚动+「针对第 N 页修改」composer 指令插入；
  python-pptx 缺失诚实降级。真机冒烟通过（3 页 deck+缓存全命中）。
  **集成**：gen_bindings 552+删线 B 临时 wrapper；bindingNames/bridge
  （PptxOutline camel+映射、GaeaBrowserObserve 同名直调，类型自
  types.ts/BrowserPanel.tsx）/spaceBindings work×2（分面锁 264）；App
  browser_* 自动弹出（去重+偏好+停用态尊重）；补更组件级
  WorkspaceTabs.test 数量锁 5→6（线 B 漏项）。**验证**：Go 110 包 0 FAIL
  （stress flaky 沿旧）；tsc -b/eslint 0；vitest 1138/1138（+43）；drift
  PASS（552）。**欠账**：A2 帧流/接管/动态权限卡远期；FilePreviewModal pdf
  仅标签；B1 单版本无入口+恢复不先快照当前态；B2 真编辑 pptx 远期；沿旧
  欠账（子代理气泡恢复暂缺/中途进度不回投/WorkHeader 历史轮耗时/窄栏适配/
  tasks flaky）。**下一刀候选**：v4.29 从欠账池+调研剩余（C2 对话内文件
  链接→右栏定位、C3 弹窗对齐、编辑器窄栏适配）挑主轴。详见
  releases/v4.28.0.md。

- **最新发布：v4.27.4（2026-09-01）「todo 持久化改名 · progress.md 撞名根治」**：
  git tag `v4.27.4`；基线 v4.27.3；绑定面 550 零变更。**勘误**：progress.md
  四次被覆写并非「并行会话」——真凶是 gaea 自身 `todo_write` 内置工具的
  V10.6 计划进度持久化：每次 todo_write 写 `<工作区根>/.gaea/progress.md`，
  办公代理以本仓库为工作区跑任务（开工筹备/安全文明手册等）时逐次覆盖同名
  发布进度文件（四次内容全是任务 todo 表，时间与任务节点吻合，backups/ 四份
  快照即 todo 表）。文件名撞车：①宿主仓库项目记忆 ②代理运行时 todo 持久化，
  用 gaea 开发 gaea 必然相撞。**修复**：写入端改名 **todos.md**（todo.go）+
  compaction 读取端 readProgressFile 优先 todos.md/回退旧名（存量工作区兼容，
  compact_util.go）；compact.go 注释同步。测试 +2（写 todos.md 不碰
  progress.md；读取优先/回退——walk-up 设计使「均缺失」断言在真实机器不成
  立，测试过程顺带实锤主目录 C:\Users\wubi\.gaea\progress.md 有 8 月底代理
  todo 残留，可手动清理）。Go 110 包 0 FAIL、前端零改动。**教训**：运行时
  产物文件名不得与宿主仓库约定文件同名——自举项目（用 gaea 开发 gaea）的
  工作区常驻本仓库，任何 <cwd> 相对写入都要过一遍撞名审查。详见
  releases/v4.27.4.md。

- **最新发布：v4.27.3（2026-09-01）「markdown 包裹符 · 交付卡片路径修复」**：
  git tag `v4.27.3`；基线 v4.27.2；绑定面 550 零变更。用户报告「交付卡片点击
  无法打开、定位打开的不是文件位置」→ computer-use 真实会话现场实锤：模型用
  反引号包裹路径，卡片提取的路径带开头反引号（\`安全文明手册/….docx）→ 预览
  「文件不存在」、explorer /select 错位。**根因**：fileLinks 路径字符集不排除
  markdown 包裹符 \` 与 *——两者是 Windows 文件名非法字符，应作路径边界
  （v4.26.1 全角括号盲区第二弹，其余非法字符本就在排除集）。**修复**：
  PATH_BODY/FIRST_SEG 排除 \`*、PATH_BOUNDARY 纳入、BARE_FILE_RE 分隔符后允许
  包裹符前缀；下划线合法字符不受影响；存量消息渲染时实时重提取、重启即恢复
  可点。+5 测试（真实会话文件名四形态+下划线守卫）。vitest 1095/1095、
  tsc -b/eslint 0。**教训**：路径正则字符集以「目标平台文件名合法字符集」为
  准——非法字符必然是 markdown/包裹符，作边界不作路径体。详见
  releases/v4.27.3.md。

- **最新发布：v4.27.2（2026-09-01）「细节收口」**：git tag `v4.27.2`；基线
  v4.27.1；绑定面 550 零变更。①**subagent_message 端到端收口**（v4.26 回投
  特性此前实际未通：后端发 kind=subagent_message、前端无消费整条被丢）——
  gaeaEventMap wire 层转译 kind="message"+subagentRef（磁盘日志仍按原始 kind
  落；前端 reducer message case 既有 subagentRef 语义接管，「子代理」徽标气泡
  真实生效）；补拉折叠同步：GaeaResyncItem.subagentRef（恒全键契约测试扩键）、
  fold subagent_message→独立 assistant 条目（ID=sa<seq>）+closePending 防其
  后 text 误续写。留白（远期）：恢复会话的模型上下文投影 ProjectMessages 未
  含 subagent_message（避免改变续跑模型语义），恢复后对话视图该气泡暂缺。
  ②**轨迹面板子代理记录**（子代理线交付）：TrajectoryRecordKind 加
  "subagent"+KindBadge/Bot 图标/折叠行（摘要+ref）/RecordInspector 全文/搜索
  命中，turns 与 betweenTurns 双落点。③**sidebar_open 目录定位**（收 v4.25
  欠账）：directory→handleRevealInTree；顺带修 FileTree 目录行无 data-path
  锚点致 reveal 静默失效的暗坑。**验证**：Go 110 包 0 FAIL
  （TestCancelConcurrentStress 全量负载偶发 flaky、单跑稳定，与本刀无关）；
  tsc -b/eslint 0；vitest 1090/1090（+5）；drift PASS（550）。**教训**：
  FileTree reveal 只锚 data-path——新增行型必须同步锚点否则定位静默失效；
  progress.md 第三次被并行会话覆写（backups/ 留档），再次从 git 恢复。详见
  releases/v4.27.2.md。

- **最新发布：v4.27.1（2026-09-01）「seq 防线 omitempty 失配修复」热修**：
  git tag `v4.27.1`；基线 v4.27.0；绑定面 550 零变更。用户报告「运行中只显示
  一个思考读秒，没有交替出现过程卡/文本卡（只有轨迹面板有显示）」。**根因**=
  v4.26 seq 补拉防线前后端形状契约失配：Go GaeaResyncItem 全字段 omitempty
  （流式 assistant 空 reasoning、写类工具 readOnly:false 的键被序列化省略），
  前端 parseResyncItems 严格校验缺键即整快照判坏 → 补拉快照 100% 被拒、防线
  静默失效——Wails 吞件期间对话窗无物可渲染（WorkHeader 是 store tick 驱动
  所以活着，轨迹面板读盘不受害）。**修复**：①Go 全字段去 omitempty +
  TestGaeaResyncItemWireAllKeys 锁「序列化恒全键」契约；②前端缺省键宽容
  （缺键→零值，类型错/kind/id/status 校验不变）。**真机验证**（computer-use
  驱动真实应用发只读任务）：对话窗 WorkHeader「已完成 · 用时 15s · 7 步」+
  阶段行（解析@引用/装配上下文/检索记忆）+ 思考块 + ls 工具卡 + 正文交替
  （elapsed 3s→8s→14s 运行中逐个渲染）；v4.26.1 交付卡片同屏可点。Go 110 包
  0 FAIL、tsc -b/eslint 0、vitest 1085/1085（+5）。**教训**：跨语言 wire 契约
  必须有一条「真实序列化形态」往返测试，omitempty 是严格校验的天敌；协作
  纪要：.gaea/progress.md 被并行会话二次覆写并随 v4.27.0 入库，已再次从 git
  恢复。详见 releases/v4.27.1.md。

- **最新发布：v4.27.0（2026-09-01）「右侧面板对齐 Codex · 子代理对话实时下钻 /
  对话与上下文完善」**：git tag `v4.27.0`；基线 v4.26.1；绑定面 550 零变更
  （纯前端，Go 零改动）。延续 v4.26「对齐 Codex」四面打磨：①右栏文件工作台——
  点文件后预览占满右栏（原顶部 3/5 小窗+底部文件树挤压），文件树收敛为「文件」
  按钮切换的 260px 侧栏（打开首个文件自动收起）；宽度上限 720→1600 + 视口感知
  钳制（视口−侧栏−400 对话区，聊天区永不消失），首次打开文件自动抬升 560（写
  全局键）；编辑器 tab 文件类型图标（lib/fileIcon 共享单源）；树内高亮当前编辑
  文件；②标签扁平化——删「资料/成本库」（组件+测试移除），取消二级标签 →
  文件/产物/变更/任务/分工一级平铺，运行角标按任务/分工下发，旧存储值自动收敛；
  ③对话输出——用户消息去气泡（实现回归注释本意）、第 2 轮起「第 N 轮」回合
  分隔线、助手消息复制按钮、编辑类工具 +N−N diffstat 芯片（diffStatFor）；
  ④子代理实时下钻——点击子代理 → 全面板对话 SubagentThread（替代树内 10px
  窄卡），运行中 3s 轮询（不可见门控）+事件驱动（turn_done 立即）实时刷新、
  自动跟随底部，状态由分工轮询实时派生；⑤上下文标签——总览头部水位分色
  （≥70% 琥珀/≥90% 红）+缓存/费用/刷新、空态引导、文件活动行点击打开预览、
  步骤详情「占窗口 %」、趋势图悬停六分类构成详情。**验证**：vitest 1082/1082
  （169 文件，净增 2：删面板测试 −7、新增 +17）；tsc -b/eslint 0；Go 零改动
  （go build 0 FAIL）；drift PASS（550）；版本四处 4.27.0。**教训**：用户在
  迭代中连续给出同类方向（「对齐 Codex」「继续完善」）时，按面批量收口比单点
  打补丁体验好——右栏/对话/子代理/上下文一次成型；扁平化删面板要同步清理组件+
  测试+持久化收敛（isWorkspaceTabId 兜底）。**说明**：规划 v4.27「浏览器与
  版本」（A2 观察窗+B1 版本时间线+B2 pptx+C2/C3）顺延。欠账：子代理对话视图
  内嵌于分工面板（非独立右栏标签）；diffstat 仅 edit_file/multi_edit；沿用
  v4.26 欠账（子代理中途进度不回投/并行派发 task 卡短暂空 ref/历史轮无耗时/
  TrajectoryView 未消费 kind=subagent）。详见 releases/v4.27.0.md。

- **最新发布：v4.26.1（2026-09-01）「全角括号文件名 · 交付卡片失配修复」**：
  git tag `v4.26.1`；基线 v4.26.0；绑定面 550 零变更。用户报告「看不见完工
  交付卡片、无法点击查看文件」→ 主代理在运行中的真实会话实证（computer-use
  观察 a11y 树）：正文有「交付文件：C:\AI\bangong\黄甲\开工筹备计划（修订）.docx」
  但 DeliverableCards 未渲染。**根因**：lib/fileLinks.ts 的 PATH_BODY 字符类把
  全角括号（）当路径终止符——文件名含（）（中文办公常态）时正则截断、扩展名
  拼不上，绝对路径匹配与「输出文件/交付文件：」关键词匹配双双落空。**修复**：
  PATH_BODY 排除集移除全角（）（扩展名仍锚定匹配末尾，「报告.docx（三份）」
  不吞补语；ASCII ) 维持排除）；点击链路无需改（GaeaPreview 的
  resolvePreviewPath 本就接受工作区内绝对路径）。+5 匹配用例 + 组件级回归
  守卫 DeliverableCards.regress.test.tsx（真实事件流→卡片可见+resync 后仍可见）。
  vitest 1080/1080、tsc -b/eslint 0。**教训**：「与版本相关的用户报告」未必是
  版本回归——用运行中的真实会话实证数据形态（本次是文件名踩中正则盲区，
  v4.26 恰好同日发布造成「现在坏了」错觉）。详见 releases/v4.26.1.md。

- **最新发布：v4.26.0（2026-09-01）「对话流式重造 · 对齐 Codex」（插刀）**：
  git tag `v4.26.0`；基线 v4.25.0；绑定面 549→550（+1：GaeaResyncEvents）。
  根因（用户报告「发送后对话窗静默而轨迹在动」）六连：①task subSink 有意丢弃
  子代理 Text/Reasoning（主窗只挂 task 运行卡）②TurnStarted 前预处理窗零事件
  ③Wails 事件流吞件（轨迹读盘不受害=直接解释）④Retrying 转译表无映射
  ⑤phase 全链就绪但零发射点 ⑥TTFT 静默（turn_started 不产 item、过程卡需
  processItems>0）。交付：**WorkHeader 工作态头部行**（turn 激活期常驻
  spinner+阶段+用时 1s tick+步数，items 为空也渲染；完成转「已完成 · 用时 ·
  N 步」Codex 式耗时行；StreamingIndicator 收敛兜底）；**后端 phase 事件接线**
  （正在启动引擎/解析 @引用/装配首轮上下文/检索记忆/思考中 + Retrying/compaction
  转译 phase，磁盘日志格式不变，200ms 节流，phase 收编过程卡+头部）；**子代理
  活动回投主回合**（新事件 subagent_message 完成态回投最终答复+ref/parentId，
  主区消息「子代理」徽标，task 卡 running 实时 lastText/lastTool 预览=App 5s
  轮询 GaeaSubagentRuns 注入 taskActivity，空 ref 回退唯一 running 分工）；
  **事件序号防线**（gaea-event 全量带 seq 转发层原子递增/会话切换归零，跳号→
  GaeaResyncEvents 从磁盘日志折叠全量快照整体替换：5s 冷却/在途去重/坏快照
  保底/streaming 续接/running 不动；golden 逐字节不变）；重复工具折叠「已调用
  X · N 次」；顺带修 weixin_reminder_test 时间炸弹。**验证**：Go 全量 0 FAIL；
  tsc -b/eslint 0；vitest 1072/1072（净增 71）；drift PASS（550）；版本四处
  4.26.0。调研 docs/research-2026-09-01/codex-streaming-ux.md。**欠账**：子代理
  中途进度不回投（实时进度走分工面板）；并行多子代理派发瞬间 task 卡预览可能
  短暂空 ref；历史轮无耗时数据源；TrajectoryView 未消费新 kind="subagent"
  记录。**下一刀 v4.27「浏览器与版本」**（原 v4.26 顺延）：A2 观察窗+B1 版本
  时间线+B2 pptx+C2/C3。详见 releases/v4.26.0.md。

- **最新发布：v4.25.0（2026-09-01）「文件工作台 · 编辑器 tab 化/变更 diff/
  选区联动/模型主动打开」**：git tag `v4.25.0`；基线 v4.24.0；绑定面 549→549
  （零新增：sidebar_open 为内置工具，走 ToolDispatch/ToolResult 事件管线）。
  规划第三刀（docs/gaea-office-upgrade-plan-2026-09.md A3+B3）。三并行子代理
  分线（文件所有权互斥）+ 主代理集成：
  ①**A3 编辑器 tab 化**：文件树点开→右栏内多文件编辑器 tab（lib/editorTabs
  zustand 外部 store：open/close/activate、上限 12 LRU、关闭激活 tab 激活相邻、
  localStorage gaea.rightPanel.editorTabs.v1 坏值兜底；openEditorTab 命令式
  入口）；FilePreview 新增 embedded 模式（默认 false 行为不变），docx 框选即改/
  xlsx 直编+Plan→Apply 等能力原样随迁（换壳不换芯红线）；双入口保留（树行
  点击=右栏内 tab、右键「预览」=主区 pane）；产物行「树中定位」reveal→
  FileTree 展开父链+滚动+1.6s 闪烁（注册表 ctx 增 revealRequest/onRevealInTree）。
  ②**A3 变更 tab diff 化**：文件行展开→行级红绿 diff（lib/planDiff 三态：
  edit_file/multi_edit 真 before/after；write_file/edit_lines 写入内容预览+
  原因；其余诚实不伪造——后端 StageBaseline 不经事件流下发前端）+ 回滚接
  证据链 Journal 最近基线（GaeaJournalList+RollbackRecord，无基线诚实标注）。
  ③**B3 选区联动**：xlsx 选中单元格→浮动「引用到对话」；docx 框选工具栏补
  「引用到对话」；docx 渲染失败降级纯文本（lib/docxText 提取 word/document.xml
  段落+amber 提示条）。
  ④**模型主动打开 sidebar_open**：Go 内置工具（internal/gaea/tool/builtin/
  sidebar_open.go，work 空间/ReadOnly 直允许不弹卡/防穿越 realPath+within/
  envelope data {kind,path_abs,path_rel}）+ lib/sidebarOpen.ts 解析器 + App 按
  工具事件 id 去重接线（file→openEditorTab，directory→亮文件 tab；命中自动亮
  右栏切「文件」tab）。
  **验证**：Go build/vet/test 全量 0 FAIL（+20 用例）；tsc -b/eslint 0；
  vitest 1001/1001（164 文件，净增 74）；drift PASS（549）；版本四处 4.25.0。
  同期调研：docs/market-research-2026-09-01.md 合成版 + docs/research-2026-09-01/
  3 分模块原始稿（v4.24.0 基线，喂 v4.26）。**欠账**：变更 diff 的 write_file/
  edit_lines 旧内容缺失（待 v4.26 B1 写前快照库）；回滚粒度=最近基线；docx 降级
  仅正文段落；sidebar_open directory 不定位树根；EditorTabs 窄栏精细适配；
  AgentNetworkCard 旧卡不动。**下一刀 v4.26「浏览器与版本」**：A2 观察窗
  （截图步进流起版+操作时间线+权限卡内联）+ B1 版本时间线（写前内容寻址快照库
  与登记表同源+vN 徽标 popover 双入口）+ B2 pptx（结构化大纲卡先行+页级指令
  两通道）+ C2/C3。详见 releases/v4.25.0.md。

- **最新发布：v4.24.0（2026-09-01）「子代理工作台 · 树拓扑/实时动态/
  产物登记表」**：git tag `v4.24.0`；基线 v4.23.0；绑定面 548→549（+1：
  GaeaDeliverableRegistry）。规划第二刀（docs/gaea-office-upgrade-plan-2026-09.md
  A1+C1）。上一轮会话完成主体编码，本轮接续完善收口：
  ①**A1 树形实时拓扑（AgentTree）**：GaeaAgentNetwork 嵌套 Children 全量渲染
  （此前只画两层）；root 折叠为「主 agent」行、一级恒可见、更深默认收起可
  展开/收起；新节点出现自动展开父链（首轮只记基线）；节点量化——状态色点/
  任务摘要/工具数/模型徽标/耗时（running 实时已用 1s tick）/token/错误数；
  下钻链：节点→详情卡→完整 transcript→**工具调用行点击定位结果消息**（收
  v4.21 欠账，data-located 高亮）。
  ②**合并活动流（Devin 式单列 feed）**：running 子代理 lastText/lastTool 按
  updatedAt 倒序合并、上限 20、空态收起；与树内行预览并存。
  ③**新子代理自动展开（可关，默认开）**：新 ref 出现→onSubagentStarted 回调
  →App 亮出右栏切「分工」tab（tab 停用时尊重停用态）；偏好键
  gaea.subagentAutoOpen（localStorage，损坏值回落默认开——交付前修 bug：
  原实现把垃圾值当关闭）。
  ④**C1 权威产物登记表**：trajectory.FoldDeliverables 纯函数从事件日志折叠
  写类 8+生成导出类 3 工具的落盘登记（路径/最近工具/来源轮次/时间/累计次数，
  按 path 去重、updatedAt 倒序、上限 200，Total 完整去重数）；登记口径
  evidence.IsDeliverableTool/ExtractDeliverablePaths（与证据链 extractPaths/
  前端 changes.ts 同源对齐，不收 source；bash/screen_capture 无结构化路径
  参数诚实不猜）；新绑定 GaeaDeliverableRegistry(sessionPath)（防穿越；
  无事件日志 Available=false 不报错）；DeliverablesPanel「权威产物登记」只读
  区（tool 徽标+路径+轮次+次数+时间，点击预览；total>200 提示最近 N 条），
  补启发式（正文扩展名白名单）漏登。
  **验证**：tsc/tsc -b/eslint 0；vitest 927/927（157 文件，净增 16：
  SubagentsPanel 重写 8、AgentTree 7、DeliverablesPanel 登记表 3、subagentPrefs
  3 等；旧扁平卡用例随新结构重写）；Go build/vet/test 全量 0 FAIL；drift PASS
  （549）；版本六处统一 4.24.0。**教训**：交付前必须跑 `tsc -b`（build.bat
  同配置）且检查新测试文件路径参数（vitest run 无参数时 tsconfig 的 include
  范围与手测不同）；vi.mock 全模块 mock 时注意副作用导入（onEvent 订阅器）。
  **欠账**：AgentNetworkCard（主区上下文 tab 旧卡）仍两层 SVG 本轮按约定不动；
  登记表为独立只读区未与启发式合并去重（v4.25 A3 变更 tab 统一快照时处理）；
  下一刀 v4.25「文件工作台」（A3 编辑器 tab 化+变更 diff+模型主动打开+reveal
  +B3 选区联动）；v4.26 浏览器观察窗+版本时间线+pptx（A2/B1/B2/C2/C3）。
  详见 releases/v4.24.0.md。

- **最新发布：v4.23.0（2026-08-31）「工作台框架 · 右栏工作台化第一刀」**：
  git tag `v4.23.0`；基线 v4.22.0；绑定面 548→548（零新增）。用户拍板：右
  面板重造为 DSH-better-sidebar/Codex 式「运行工作台」（子代理/浏览器/文件
  编辑器等实时操作面），状态显示类迁主区轨迹/上下文旁边；规划稿
  docs/gaea-office-upgrade-plan-2026-09.md（v2，分期号已顺延）。
  ①**Tab 注册表 lib/sidebarRegistry.ts**：元数据复用清单 + render 接线单一
  数据源，右栏渲染/命令面板全派生；新增面板 = 清单 + RENDERERS 各一条，
  面板组件本体零改动（框架/内容解耦，为 A2 浏览器观察窗/A3 编辑器 tab 留
  平等挂载点）。
  ②**工作台外壳三件套（学 better-sidebar v0.18 交互形状，不抄代码）**：全局
  宽度键（左缘拖拽 280–720，最后一次拖拽胜出跨会话跟随，layoutPreferences
  blob workspacePanelWidth）；声明式设置（齿轮→侧边卡片，每 tab 独立开关，
  停用即隐藏/整组停用隐藏主 Tab/至少保留一个/停用不进命令面板，启用集全局键
  gaea.rightPanel.v1:tabsEnabled，booleanMapOf 式净化）；会话记录 v2
  （{v,tab,enabled,width} JSON，v1 裸 id 兼容可读，坏值逐项兜底，失效激活
  指针修正）。
  ③**主区「概览」tab（A4 统计迁移）**：ChatTabs 第 4 tab + OverviewPanel
  承载原 StatsPanel（本体零改动）；右栏统计下线（WorkspaceTabId union 全量
  移除，v4.22 旧 tab:"stats" 宽容收敛回「文件」并钉回归用例），右栏收敛
  3 主 Tab×7 面板；命令面板新增「概览面板」项；chatTab 恢复白名单加 overview。
  **实现方式**：两个并行子代理分线（框架线/概览线，文件所有权互斥）+ 主代理
  集成收口；build.bat 的 tsc -b 曾暴露 NodeList for...of 迭代错（tsc --noEmit
  配置差异，已改 forEach）——教训：新测试代码过门前必须跑 `tsc -b`。
  验证：tsc/tsc -b/eslint 0；vitest 911/911（净增 38）；Go build/vet/test
  0 FAIL；drift PASS（548）；版本四处 4.23.0；build.bat 构建成功冒烟 200。
  欠账：Tab 拆分分栏/底部面板/自由窗口、设置二级齿轮弹窗、注册表懒加载
  chunk 化；下一刀 v4.24「子代理工作台」（树拓扑/live/下钻链/活动流+产物
  登记表）。详见 releases/v4.23.0.md。

- **v4.22.0（2026-08-31）「一次性收官 · 真虚拟化/transcript 定位/
  晨报预载 UI」**：git tag `v4.22.0`；基线 v4.21.0（同工作区，未打 tag）；
  绑定面 546→548（+2：GaeaMorningPreload/GaeaSetMorningPreload）。用户要求
  「一次性做完」，办公板块剩余本地可做欠账一次清完并整理提交收尾：
  ①**轨迹真虚拟化（react-window v2 动态行高）**：扁平行流按视口窗口渲染
  （±overscan 12），超长会话 DOM 恒定，v4.21「首批+加载更多」分批机制退役；
  useDynamicRowHeight + ResizeObserver 实测展开行高自动重排（jsdom 回落
  defaultRowHeight）；概览跳转走 listRef.scrollToRow、搜索回顶、收起/展开
  照常；test/setup 补 ResizeObserver stub。
  ②**transcript 消息定位**：消息序号 #N（按原位置）+ 搜索命中自动滚动到
  第一条命中。
  ③**晨报预载 UI 开关（+2 绑定）**：GaeaMorningPreload/GaeaSetMorningPreload
  ——internal/config.Save 持久化 + 内存更新 + gaeaRebuildLocked 即时生效；
  记忆面板「晨报预载 开/关」胶囊按钮；gen_bindings 重生成 548，
  bindingNames/mock/bridge/spaceBindings 同步。
  验证：Go 全量 0 FAIL（绑定面完整性 548 PASS）；tsc/eslint 0；vitest 873/873
  （+3）；drift PASS（548）；版本四处统一 4.22.0。**收尾**：v4.17.0-v4.22.0
  六轮改动交织在同一工作区（无法按版本拆分文件），作为一次合并发布提交并
  打 tag v4.22.0；releases/v4.17.0.md…v4.22.0.md 保留完整演进记录。剩余欠账
  仅外部资源/官方数据项：Realtime 真机（真 key+麦克风）、自动路由本体（待
  官方逐模型缓存/峰谷数字）、浏览器下载上传/headless UI/Windows UIA、iLink
  真机窗口——本地不可完成。详见 releases/v4.22.0.md。

- **最新发布：v4.21.0（2026-08-31）「长会话与 transcript · 增量渲染/消息搜索」**：
  git tag `v4.21.0`；基线 v4.20.0（同工作区，未打 tag）；绑定面 546→546
  （零新增绑定，纯前端）。续 v4.20.0 剩余两条欠账：
  ①**轨迹增量渲染（DOM 有界）**：渲染从「逐轮整体」改为扁平行流（轮次头 +
  展开记录行 + Between-turns），按批渲染——首批 250 行，滚动到底自动续载
  或「加载更多（剩余 N 条）」，搜索词变化回首批；概览跳转同步把目标轮之后
  的可见区扩进视口（不再「跳过去了但没渲染」）；收起全部/展开全部在平行流
  上照常生效（折叠 = 记录行不进流）。
  ②**子代理 transcript 消息搜索**：查看器头部搜索框，按正文/推理/工具名/
  参数/结果过滤，计数「命中/总数」，无匹配空态；搜索词与展开状态互不干扰。
  ③**注释清理**：ChatTabs「[轨迹] …（暂占位）」更新为 v4.17-v4.21 实际
  能力（事件账本：概览/搜索/折叠/增量渲染）。
  验证：tsc/eslint 0；vitest 872/872（+2）；Go 全量 cached 绿（绑定面 546
  不变）、drift PASS（546）；版本四处统一 4.21.0。欠账：增量渲染为分批 DOM
  而非 react-window 真虚拟化（渲染量有界但已渲染部分仍为真实 DOM）；
  transcript 只读无跳转/引用定位。详见 releases/v4.21.0.md。

- **最新发布：v4.20.0（2026-08-31）「剩余收官 · 子代理 transcript/轨迹概览/
  旧会话趋势补齐」**：git tag `v4.20.0`；基线 v4.19.0（同工作区，未打 tag）；
  绑定面 545→546（+1：GaeaSubagentTranscript）。清掉 v4.17-v4.19 之后的剩余
  欠账，三件事一并落地：
  ①**子代理完整 transcript 查看器**：新绑定 `GaeaSubagentTranscript(sessionPath,
  ref)` 读取 `<sessionDir>/subagents/<ref>.jsonl` 全量消息（role/name/content/
  reasoning/toolCalls/toolCallId）；ref 校验 sa_ 前缀 + 仅安全字符（防路径
  穿越）+ 长度上限；读取失败返回错误（查看器区分「没有」与「读不了」）。
  gen_bindings 重生成（546），bindingNames.ts 同步，drift PASS。前端 Agent
  网络详情面板增「查看完整 transcript」按钮 → 消息流（角色徽标 + 推理块 +
  工具调用 + 结果，可滚动、可收起）。
  ②**轨迹 Overview 投影 + 轮次跳转 + 折叠控制**：轨迹标签顶部「轨迹概览」
  投影条（每轮一根柱，柱高 ∝ 记录密度，含工具调用高亮、报错轮标红，hover
  显示「第 N 轮 · X 条记录 · Y 工具调用」，点击平滑滚动到该轮并展开目标）；
  「收起全部 / 展开全部」控制长会话折叠成轮次索引，新回合（实时刷新）默认
  展开不被旧折叠态吞掉。
  ③**迁移/兜底会话趋势补齐（诚实估算）**：ToLogEntries 每回合合成
  request_header——system = 真实 system 消息拼接（joinPromptPart），tools =
  该轮 assistant 实际用到的工具名集合（schema 未知的最小诚实形状）；顺序与
  运行期一致（user 先落、header 随后）。contextview 新增回合末估算关闭
  （turn_done 未见 usage 时用当前估算构成落 estimated 记录并刷新 brief），
  前端步骤详情显示「估算构成（无用量记录）」，不伪造 promptTokens 等用量
  数字。
  验证：Go 全量 0 FAIL（+2 用例：回合末估算关闭、合成 header system/工具名
  断言）；绑定面完整性 546 PASS；drift PASS（546）；tsc/eslint 0；vitest
  870/870（+3：轨迹概览、收起/展开全部、子代理 transcript 渲染与收起）；
  版本四处统一 4.20.0。欠账：轨迹虚拟滚动未做（以收起全部+概览跳转缓解）；
  子代理 transcript 只读无搜索。详见 releases/v4.20.0.md。

- **最新发布：v4.19.0（2026-08-31）「看板收官 · 上下文浏览器//context 命令/
  子代理节点详情」**：git tag `v4.19.0`；基线 v4.18.0（同工作区，未打 tag）；
  绑定面 545→545（零新增绑定）。续 v4.17.0+v4.18.0 的「继续完善」第三刀，
  上下文标签最后一个页脚占位收掉，三件事互不相交一并落地：
  ①**上下文浏览器**：contextview 折叠补全系统/工具节点——request_header
  的 system prompt 与工具集合只在构成变化时入 nodes（初版+变化版各一条，
  每步重复不刷屏），节点文本=预览（300），全文在日志 request_header 行；
  nodes 覆盖全部六分类，与「模型可见节点」文档语义对齐。前端
  ContextBrowserCard：活跃/归档双页签（归档=被压缩移出节点，带「已压缩」
  标记）+ 六分类过滤 chips + 节点行（分类色点+≈tokens+预览，超长可展开/
  收起，展示最近 60 条）；页脚占位整行移除。
  ②**/context 命令**：GaeaCommands() 内置 + i18n（zh/en CmdContext），斜杠
  菜单可发现可补全；classifyComposerCommand 增 context 分类，App.handleSend
  拦截 `/context` → setChatTab("context")（不发给模型）；CLI 未拦截路径走
  控制器未知斜杠 Notice，诚实不猜。
  ③**Agent 网络节点点击 → 子代理详情**：AgentNetworkCard 增 sessionPath
  （App 从 currentSessionPath 注入），点击子代理节点 → SubagentRuns 按任务
  摘要前缀匹配（与后端 enrichAgentNetwork 同口径）→ 固定详情面板（状态/
  模型/工具调用数/更新时间 + lastText/lastTool + 最后回答摘要）；无会话路径
  或匹配不到回退节点统计；悬停文案更新为「悬停查看节点详情 · 点击节点固定
  子代理详情」。
  验证：Go 全量 0 FAIL（+1 用例：系统/工具节点只在构成变化时新增、文本含
  工具名）；tsc/eslint 0；vitest 867/867（+4：/context 分类、浏览器活跃节点
  展开/收起、归档页+占位移除、子代理节点点击详情）；drift PASS（545）；版本
  四处统一 4.19.0。欠账：子代理完整 transcript 查看器（详情面板非全文）；
  轨迹 Overview 投影与虚拟滚动；迁移/兜底会话系统/工具分类与趋势柱（旧消息
  无 request_header/usage，诚实不造数）。详见 releases/v4.19.0.md。

- **最新发布：v4.18.0（2026-08-31）「看板补全 · 文件活动/增量模式/实时刷新」**：
  git tag `v4.18.0`；基线 v4.17.0（同工作区，未打 tag）；绑定面 545→545
  （零新增绑定）。续 v4.17.0 的「继续完善」，三件事互不相交一并落地：
  ①**文件活动时间线**：contextview 折叠新增 FileActivity——工具参数确定性
  提取路径（path/rel/source/destination/image_path/output 键优先级）+ 工具→
  动作白名单（read_file/grep/vision/format_convert=read；write_file/edit_file/
  multi_edit/edit_lines/chart_gen/diagram_gen/screen_capture=write；move_file=
  move；ls=dir）；screen_capture 从结果输出补记；bash 等无法确定性取路径者
  诚实不造数；同轮同步骤同路径合并刷新、上限 200、空切片非 nil。前端
  FileActivityCard（动作徽标+工具+路径+时间，倒序最近 40 条），页脚改为
  「上下文浏览器将在后续阶段接入」。
  ②**增量（Delta）模式启用**：趋势图「增量」按钮去灰置（Phase B 占位收口），
  切换展示每步相对上一步净变化（绿=净增·红=净减，图例随模式出现）；柱色改
  全站一致的可视化语义色（hex-exempt）。
  ③**运行中实时刷新**：新 hook useLiveReload 订阅 gaea 事件流——运行中节流
  刷新（1200ms）、turn_done 立即刷新、running true→false 整轮刷新；轨迹/
  上下文/Agent 网络三处统一接入（替换「仅回合结束刷新」effect）。
  验证：Go 全量 0 FAIL（+2 用例：文件活动折叠 + 空快照 files:[] 序列化回归）；
  tsc/eslint 0；看板+mock-contract vitest 22/22（+2：文件活动卡渲染、增量
  图例）；drift PASS（545）；版本四处统一 4.18.0。欠账：上下文浏览器（surface
  节点浏览/归档重建）仍占位；Agent 节点点击跳子代理会话；轨迹 Overview 投影
  与虚拟滚动；/context 命令；迁移/兜底会话的系统/工具分类与趋势柱（旧消息无
  request_header/usage，诚实不造数）。详见 releases/v4.18.0.md。

- **最新发布：v4.17.0（2026-08-31）「轨迹上下文接通 · 事件日志默认开启」**：
  git tag `v4.17.0`；基线 v4.16.0 + 1 提交；绑定面 545→545（零新增绑定）。
  用户反馈办公板块「轨迹」「上下文」标签空壳 → 单刀接通数据链路：
  ①**事件日志缺省开启（根因修复）**：`config.EffectiveLogFormat()` 缺省
  "event"（仅显式 "legacy" 退回），`LogFormatIsEvent` 改用生效值；gaea_handler
  注入 `ctrl.SetLogFormat(EffectiveLogFormat())`，boot 同源创建 EventLogSink——
  两看板 + Agent 网络从下一轮对话起即有真实数据。
  ②**旧会话读端兜底**：`session.ReadEntriesFor(path)` 优先事件日志、缺失时
  把旧 `<id>.jsonl` 投影为折叠条目（含回合边界，纯读不迁移不落盘）；
  GaeaTrajectory/GaeaContextView/GaeaAgentNetwork 统一改用。
  ③**迁移产物带回合边界 + 折叠兼容**：ToLogEntries 每条 user 消息前写
  turn_started、流尾写 turn_done（ProjectMessages 忽略边界，恢复投影逐字节
  不变，golden round-trip 保持）；trajectory 折叠兼容 assistant_message
  （正文/推理并入 assistant、内嵌工具调用展开为 tool 记录并与 tool_result
  按 ID 合并）。
  ④**写入器随控制器释放**：boot 把 EventLogSink.Close 组合进 Controller.Cleanup
  （幂等）——缺省 event 后 Windows 会话目录可删除/迁移。
  验证：Go 全量 0 FAIL（+6 用例：config 缺省/legacy、ToLogEntries 边界与投影
  往返、迁移计数、ReadEntriesFor 优先/回退/双缺失、轨迹折叠迁移产物、boot
  缺省 event/显式 legacy）；tsc -b 0；TrajectoryView/ContextView/AgentNetworkCard
  vitest 12/12；drift PASS（545）；版本四处统一 4.17.0。欠账：迁移/兜底会话
  无 request_header/usage（系统/工具分类 0、趋势无柱——旧消息无法回填真实
  用量，诚实不造数）；Agent 网络对 legacy 会话仅 root；增量（Delta）模式仍
  Phase B、上下文浏览器/File activity/SSE 增量刷新/轨迹 Overview 投影与虚拟
  滚动 = v3.5 既定欠账未动。详见 releases/v4.17.0.md。

- **最新发布：v4.16.0（2026-08-31）「四刀并行 · 离线收口/浏览器键盘与 iframe/
  复核可视化/晨报预装配」**：git tag `v4.16.0`；基线 v4.15.0 + 1 提交；绑定面
  545→545（零新增绑定）。用户拍板「全部并行处理」——四个并行子代理落地（足迹
  隔离，主控全绿门禁）：
  ①**persona 侧离线裂缝收口（真 bug）**：gaea_whisper_causal/retell/whisper_handler
  三处 `featureModel("chat")` → `routeModel("chat")`——全局离线过滤对轻语链路生效
  （此前绑云端照样发云端），用户功能绑定语义不变（同源）+2 离线回归。
  ②**浏览器键盘级 Input + iframe（v4.13/14 欠账）**：新工具 `browser_press`（第 11
  工具：Input.dispatchKeyEvent 键盘级输入，key 别名表/组合键/text 真实输入，Enter
  补 `\r` 触发 keypress 真机踩坑修复）+ browser_read/click/type 加 `frame` 参数
  （getFrameTree→createIsolatedWorld→contextId，**iframe 内交互完整实现**，真
  headless Edge 真机验证 Read/Click/Type 全通）；snapshot 不下钻 iframe 诚实拒。
  ③**Verifier 通道 B 结果进前端（v4.14 欠账）**：Verdict 增 channelBRatio/
  channelBPages/channelBArtifacts（omitempty 旧卡兼容）+ 证据卡「视觉复核：像素
  差异率 x.x% · N 页」行 +「查看复核产物」按钮（打开产物目录）。
  ④**晨报深度预装配（v4.14 欠账）**：memory.BuildMorningPreloadBlock 纯函数（复用
  BuildMorningBrief 排序口径，≤600 rune 确定性零 LLM）→ sysprompt 装配点注入
  「【工作记忆晨报】」块（门控 Memory.Enabled && morning_preload && space==work，
  play/mode=off 不注入=双空间红线）；config 键 morning_preload（默认 true，仅配置
  文件可控）。
  验证：Go 全量 0 FAIL（+20）；**vitest 861/861**（+2）；tsc/eslint 0/0；drift PASS
  （545）；build.bat 冒烟 200；版本四处统一 4.16.0。欠账：Realtime 真机（需用户
  真 key+麦克风）；自动路由本体（待官方逐模型缓存/峰谷数字）；浏览器 snapshot
  不下钻 iframe/下载上传/headless UI/Windows UIA；通道 B 逐页缩略图；晨报预载无
  UI 开关。详见 releases/v4.16.0.md。
- **最新发布：v4.15.0（2026-08-31）「聊天路由归位 · 由谁回答」**：
  git tag `v4.15.0`；基线 v4.14.0 + 1 提交；绑定面 545→545（零新增绑定）。
  自动路由 v1 经**用户拍板收缩**为最小价值刀（砍成本档位/开关/UI——缓存价/峰谷
  价无官方逐模型数字，按「未核实不入表」纪律诚实不做）。要点：
  ①**聊天路由归位（真 bug 修复）**：chat_service.go:68/:105 + chat_handler.go:9
  三处 `featureModel("chat")` → `routeModel("chat")`——用户功能绑定语义逐字节不变
  （routeModel 步骤 1 与 featureModel 同源）；修复「总闸不总」裂缝（全局离线模式
  对 plain 聊天生效，此前绑云端照样发云端、persona 却被滤）+ 无绑定时全局/兜底
  与 persona 一致 + model.route 事件补齐。featureModel 保留（展示用），routeModel
  零改动。
  ②**「由谁回答/为何/花了多少」回显**：modelengine 导出 `EstimateCostCNY`（本地/
  未知恒 0、USD 按汇率折算 CNY、非法汇率回退 7.2）；chat done 帧/ChatSend 返回
  加 `answered_by{engine,model,source,cost_cny}`（流式按 chunk.Usage 实算，usage
  不可达诚实记 0）；前端 `AnsweredByLine` 消息底部小字（费用 ≤0 隐藏段）+ 
  `useChatStream` 解析（旧事件静默跳过向后兼容）。
  验证：Go 全量 0 FAIL（+7）；**vitest 859/859**（+7）；tsc/eslint 0/0；drift PASS
  （545）；build.bat 冒烟 200；版本四处统一 4.15.0。欠账：自动路由本体未做（待
  官方逐模型缓存/峰谷数字后另刀）；persona 侧 gaea_whisper_causal/retell 同类
  离线裂缝=观察项；按 source 拆分统计未做；plain 费用口径 usage 不可达恒 0
  （诚实降级）。详见 releases/v4.15.0.md。
- **最新发布：v4.14.0（2026-08-31）「三箭并行 · 晨报预取 + 浏览器续刀 + 复核产品化」**：
  git tag `v4.14.0`；基线 v4.13.0 + 1 提交；绑定面 544→545（+1：
  GaeaMemoryMorningBrief）。用户拍板「多刀并行」——三个互不相交小刀由三个
  并行子代理落地（探索 → 实现 → 主控全绿门禁）：
  ①**浏览器续刀**（v4.13.0 欠账）：空闲 TTL 自动关停（Options.IdleTTL 默认
  10min + GAEA_BROWSER_IDLE_TTL env 覆盖；Ensure 成功路径刷新 lastActive，
  once 守护 watcher 到期 teardownLocked 自动回收；到期后 browser_* 自动重拉
  闭环）+ 多标签页（Manager 重构 tabs map + activePageID，/json/list 全量
  target 为真源；ListTabs/NewTab/SwitchTab/CloseTab；切/建 tab 置 epoch=0 旧
  refs 诚实失效；关 active 切剩余、最后一个整体回收）+ 新工具 ×3（browser_tabs
  只读 / browser_new_tab / browser_switch_tab）+ browser_close 可选 tab_id
  （缺省保持现语义）。零新增绑定、前端零改动。
  ②**做梦 2.0 主动预取 MVP**（路线图 T0 欠账）：memory.BuildMorningBrief 纯函数
  （零 LLM/零 IO/确定性：max(UpdatedAt,LastUsedAt) 降序 top5 user/project 优先
  + procedural/rule ≤3 条 + rune 边界截断 120 + 空输入非 nil 空数组）+ 新绑定
  GaeaMemoryMorningBrief() (string, error)（JSON 串对齐 GaeaCostGraph 先例；
  ListInSpace("work") 只读 + 近 24h dream 审计计数；零写库零落审计，play 红线
  安全）+ 首页 MorningBriefCard（仅 work 空间渲染，失败/空静默隐藏，全 token）
  + i18n home.morningBrief.* 三语 + gen_bindings 重生成（bindingNames 545）。
  ③**Verifier 产品化**（调研 ★★☆，纯前端零新增绑定）：证据卡三步展开——卡面
  （无 baselinePath 回滚禁用 + 「可复核明细」徽标）→ 声明↔实况 diff（opsJson ×
  GaeaPreview 现取实况，口径同后端 1e-9/去空白/公式归一，✓/✗/跳过 + 近似比对
  脚注）→ 操作回放时间线（applyOne 风格描述 + 批量 op 折叠，旧卡回退
  beforeSummary）；lib/verifyDiff.ts 纯函数 + types 补 baselinePath/opsJson/
  XlsxOpView/VerifyDiffRow + mock/office.ts 补证据域三绑定。
  验证：Go 全量 0 FAIL（+25）；**vitest 852/852**（150 文件，+27）；tsc/eslint 0；
  drift PASS（545）；版本四处统一 4.14.0；build.bat 冒烟 200。欠账：晨报深度
  预装配（进 agent 上下文）列第二刀；浏览器 iframe/键盘级 Input/下载上传/
  headless UI/Windows UIA；Verifier 通道 B 结果未进前端、复核明细绑定留待真实
  需求；本地-云端自动路由 v1 顺延下一刀。详见 releases/v4.14.0.md。
- **最新发布：v4.11.0（2026-08-30）「GLM 全模态纵深」**：git tag `v4.11.0`；
  基线 v4.10.0 + 1 提交（29c23ee）；绑定面 543→544（+1：SetGlmEndpoint）。
  要点：GLM 生图后端（官方 images/generations 只发官方 schema 字段，URL 转
  data URL，错误体原样透出，img2img 诚实拒绝）+ App 三处接线（Key 经
  Manager.GLMKey() 同源）+ 官方双端点切换（std=按量/coding=编码套餐，只收
  GLMBaseURLStd/GLMBaseURLCoding 两常量，GLM 卡 Segmented）+ 生图模型目录
  补全（18→22）+ glm-5-turbo 误分类修复 + 设置页绘梦引擎标签诚实化。
  **vision 识图链路用户拍板不动**。Go +15 测试、vitest 821/821、drift PASS
  （544）。欠账见 releases/v4.11.0.md（做梦 2.0 主动预取 / Realtime 真机 /
  本地-云端自动路由 v1 / iLink 语音视频 / 更深跳因果）。
- **最新发布：v4.10.0（2026-08-30）「GLM 引擎 · 办公秘书人设」**：
  git tag `v4.10.0`；基线 v4.9.0 + 9 提交；绑定面 540→543（+3：SetGlmKey/
  GetGlmKeyStatus/AcceptMergeSuggestion）。要点：
  - **GLM 引擎上线（三轮真机打通）**：智谱 OpenAI 兼容 paas/v4 全链路；
    官方文档纠偏——无 /models 端点，改静态目录（18 模型，glm-5.3 旗舰默认）
    + chat ping 验证 Key；地址防呆三防线（云端隐藏地址框/SaveEngine scheme
    校验不回显原值/LoadState 脏数据自愈）；saveSetters 覆盖测试绝育此类错。
  - **工作人设收口（用户拍板）**：professional tag 豁免节奏引擎（PAD 标尺
    下 chatter 阈值形同虚设）；新增办公秘书人格（30→31）；[SPLIT] 三出口
    上游归一；搜索触发词收窄（宁漏勿误）。
  - **审计欠账三刀**：多跳因果链（≤2 跳「导致」链）；Verifier 通道 A 引用
    级深化（opsJson+声明↔实况）；做梦 2.0 蒸馏真实合并（确定性检测+可逆
    归档，T0 第一刀）。
  - **Herdsman CLI 错误透明化**：exit 3 根因=桌面端提权运行管道拒普通权限；
    失败路径透出结构化错误+定向提示。
  - **验证**：Go 全量绿（+20 回归）；vitest 818/818（148 文件）；tsc/eslint 0；
    drift PASS（543）；版本四处 4.10.0；build.bat 冒烟 200。
  - **欠账清单**：Realtime 真机验证轮；GLM 视觉/生图路由与 Coding 端点；
    做梦 2.0 主动预取；更深跳语义锚定因果推理；iLink 语音/视频 item。
- **上一发布：v4.9.0（2026-08-30）「星枢首页·轻语记忆纵深」**：
  git tag `v4.9.0`；基线 v4.8.3 + 15 提交；绑定面 535→540（+5：EpisodeReplay /
  MemoryRetell / AnchorReplay / GraphSubgraph / CausalExplain）。要点：
  - **轻语记忆回放系列（审计 §C 收口）**：GaeaWhisperEpisodeReplay 按情节从
    chat_history 确定性重建原始对话；GaeaWhisperMemoryRetell LLM 人格口吻重述；
    时间锚点「重访那一天」写路径接线（策略从定义变生产）+ 纪念日回放绑定与 UI。
  - **图谱三维度**：情绪（Triple 情绪三字段 + hermes.db V13→V14 + 前端按情绪
    着色）、因果（extractCausalTriples 确定性因果三元组，ingest/文档导入双路径
    入图）、关联（GraphSubgraph 并入 event_chain 等记忆关联边 + 前端因果琥珀
    虚线）。
  - **跨事实因果推断**：GaeaWhisperCausalExplain「为什么」——确定性收集证据
    （KG「导致」三元组 + event_chain 关联，上限 8 条）+ 当前人格口吻 LLM 人话化
    （只用证据/不编造/证据不足诚实说明/≤200 字）；无证据零 LLM 调用回退文案；
    图谱面板「解释因果」按钮。
  - **首页重构「星枢指挥所」+ 两段式启动动画**：启动默认从首页落地（跨空间
    恢复保留）；index.html 静态启动屏 → BootSplash（旋转光环+徽记+分步状态+
    进度条，rAF 节流/reduced-motion 全降级）；Hero 命令条（orb/打字/语音/⌘K）+
    真实遥测细条 + manifest 驱动 Bento 能力矩阵；i18n 三语 home.*/boot.* 17 键。
  - **工程化与修复**：build.bat 构建冒烟自动化（真实退出码+产物新鲜度守卫+
    自动冒烟）；desktop_session/archive 持久化原子写统一；XlsxPreview 大表行
    虚拟滚动；VoiceStart realtime 门不依赖 whisper chat（端到端走服务端
    response 事件）；微信识图提示词去 OCR 窄化 + 短问题豁免长度镜像。
  - **验证**：Go 全量绿（vet 0）；vitest 818/818（148 文件）；tsc/vite build/
    eslint 0；drift PASS（540）；版本四处统一 4.9.0；wails build + 冒烟 200。
  - **欠账清单**（v4.9.0 后对账更新）：Realtime 真机验证轮（真 key/麦克风/
    打断体感/AEC）；更深跳语义锚定因果推理（≤2 跳链已上线「多跳因果链」刀）；
    iLink 语音/视频 item 探明（静默跳过）。已收账：锚点策略刻度对齐
    （90ab160，发布时误列）；成本知识图谱（v4.8.0 已交付，发布时误列）；
    Verifier 通道 A 引用级深化（opsJson 随卡 + 声明↔实况比对）。
- **上一发布：v4.8.3（2026-08-30）「微信图片双向」**：
  git tag `v4.8.3`；CHANGELOG / releases/v4.8.3.md / README 索引同步。
  v4.8.2 发布当日真机实测复盘五刀（1bfd41d/2cca12a/b1921e9/cda6522/
  72e11e3），协议三方印证（本机抓包解密 + hermes-agent weixin.py +
  openilink SDK）。要点：
  - **出图回推（真机 delivered）**：getuploadurl（filekey/aeskey/md5/PKCS7
    filesize）→ CDN 密文直传（x-encrypted-param 票据）→ sendmessage
    image_item 卡片 + caption 独立补发；media_crypt.go 手写 AES-128-ECB
    + PKCS7（Go 无 ECB）；任何失败降级文本卡片逐字节不变；接线修复=
    SendFileCard 曾是孤儿（CardPath 拼文本从不调上传——真凶）+ handle
    空回复守卫。
  - **发图识别（真机两连发通过）**：入站 type=2 + image_item{aeskey,
    media{full_url, encrypt_query_param, aes_key}} 防御解析；
    DownloadImageEncrypted（dial-time SSRF/20MiB → 解密 → 魔数终审才
    落盘）；file:// 分支限 TempDir+魔数。
  - **识图模型升级**：多模态 Qwen3.6-35B 主模型优先（真机探针实测视觉
    完好；手写体强；与聊天同体零额外显存；OCR 式提示词），PaddleOCR →
    MinerU → OvisOCR2 三级链降兜底。
  - **识图排障**：身份类问题（你是谁/你会什么…）跳过联网搜索——「你是
    谁」误触搜索注入英文网页致回复夹乱字母；日志预览 rune 化（\xe6\x81
    伪影）。
  - **关键坑**：type=2 非 3；aes_key 必须 base64(hex字符串)（base64 原始
    字节=灰框）；上传域与扫码 baseurl 无关（无需重扫，v4.8.2 媒体域假设
    作废）。
  - **验证**：Go 全量绿、零新增绑定（535）、前端零改动（vitest 807/807
    沿用）、版本五处统一 4.8.3、gaea-v4.8.3.exe + 冒烟 200。
  - **欠账**：手写识别质量待复测 / chat 路由绑 35B Q4 慢（配置项）/
    iLink 语音视频 item 未探明 / Realtime 真机验证轮延续。

- **最新发布：v4.8.2（2026-08-30）「欠账收尾」**：
  git tag `v4.8.2`；CHANGELOG / releases/v4.8.2.md / README 索引同步。要点：
  - **权限升级请求**（v3.7.0 挂账清账，零新增绑定）：request_permission
    工具（reason 必填/headless Never-Ask 降级/六形态结果文本）+
    PermissionRequester ctx 盖章（仿 Asker）；control.RequestPermission
    硬纪律三闸——deny 规则硬拒先行、hardAsk 拒绝升级（yolo 同拒）、批准
    只写 grantedRules 会话 glob 规则表（真实调用仍走 Gate.Check 不绕闸门，
    TestRequestPermissionGrantFeedsNormalGate 钉死）；五决策接线
    （persist_allow 走 PersistAllowRule 持久化）；审批卡 request 形态
    （规则串+reason 原文块，普通卡逐字节不变，三语 +5 键）；granted map
    精确 key 局限由 glob 规则表补全（bash(go build*) 类规则可匹配）。
  - **竞态/flake 全治理**：Cancel 被 succeeded 吞掉的收尾窗竞态（真生产
    bug——worker 读 ctx.Err() 与 Cancel 的 cancel() 之间窄窗，修复=
    userCancel 单独即取消，×10 压力绿）；stubGate 测试桩加锁（v3.8.0
    挂账）；filewatch 风暴测试时序根治（合批窗 1s+条件等待，全量实战
    零 FAIL）；ProgrammingPage 显式 5s 超时（v3.9.0 挂账）。
  - **Realtime S2 事件环骨架**（方案 a，设计 docs/gaea-v482-realtime-s2-design.md）：
    Resample16kTo24k 定点插值纯函数（16k/24k 协议硬伤根治，8 例保真）+
    事件常量 +7（解析骨架零改动）+ TurnControl 可选接口（fail-closed）+
    voice_manager 事件泵（旁路本地 VAD/RMS 双源冲突、barge-in 三联、
    delta 聚合→done 冲洗 24k WAV 复用前端播放环、PTT→Commit）+ 前端
    browserASRAvailable 死门旁路（WebView2 PCM 从未进后端）+ 五重降级
    护栏（未注入=逐字节老路双层守护）。31 新测试 + 既有 41 用例全绿。
    **真机欠账**：延迟/AEC 误打断/打断手感/格式怪癖/gpt-realtime
    instructions 落位。
  - **验证**：Go 全量绿（风暴修复实战零 FAIL）；vitest **807/807**；
    tsc/eslint 0；绑定面 535、drift PASS；版本五处统一 4.8.2；桌面端
    gaea-v4.8.2.exe（SHA256 见 releases/SHA256SUMS-v4.8.2.txt）+ 冒烟 200。
  - **欠账**：Realtime 真机验证轮 / iLink 真机窗口 / 生命库可写化=观察项
    （VoiceStart WhisperReady 门小修、XlsxPreview 虚拟滚动已由后续小步
    收账——见上方「欠账收尾小步」）。

- **最新发布：v4.8.1（2026-08-30）「欠账清尾」**：
  git tag `v4.8.1`；CHANGELOG / releases/v4.8.1.md / README 索引同步。要点：
  - **全局离线模式设置 UI**（绑定面 533→**535**）：GaeaGet/SetOfflineMode
    绑定（ModelB 门面 + gen_bindings 归类）+ shelf.SaveConfig 内存同步；
    SecurityPanel「全局离线模式」总闸段（回填/切换即存/失败回滚不静默），
    文案点明与敏感域/办公本地优先的叠加关系。
  - **Realtime S1（兑现 S0 注释）**：realtime 三键六处注册（provider 非空
    仅 openai、Key 存储口径=DPAPI 密文）+ realtimeRuntimeCfg 解密助手 +
    initVoice 在 NewManager **之前**注入（接线位置测试守护——曾放后面只改
    局部副本）+ VoiceHealth.realtimeReady=配置且构造成功；未配置零变化。
    VoiceApplySettings/GetSettings 扩三键：Key 明文进内存+密文落盘、凭据
    保存失败返回错误（静默丢 key=「已配置」假象）、**明文 Key 永不出后端**
    （hasKey 布尔回读，测试断言无泄漏）；VoiceSettingsPanel「实时语音
    （实验）」段（供应商/模型/密码框 Key 不回显/保存回读）。
  - **验证**：Go 全量绿（120 包；filewatch 风暴测试已知抖动隔离复跑绿）；
    vitest **803/803**；tsc/eslint 0；绑定面 535、drift PASS；版本五处统一
    4.8.1；桌面端 gaea-v4.8.1.exe（SHA256 见 releases/SHA256SUMS-v4.8.1.txt）
    + 冒烟 200。
  - **欠账**：Realtime S2（端到端接管+打断联动，真 key 真机）/ iLink 真机
    窗口 / 权限升级请求+stubGate 竞态 / XlsxPreview 虚拟滚动、生命库可写化
    =观察项。

- **最新发布：v4.8.0（2026-08-30）「全面铺开 · 触点纵深」**：
  git tag `v4.8.0`；CHANGELOG / releases/v4.8.0.md / README 索引同步。要点：
  - **七刀并行落地**（多子代理分工、文件足迹不相交）：
    ① 读屏纵深——screen.Monitors()/CaptureArea 多显示器（EnumDisplayMonitors
    枚举，Capture 薄封装零行为变化）+ intent 序数解析「第N(块)屏/主屏/副屏」
    （动词锚定窄规则，越界诚实报错）+ OCR 本地摘要朗读（>300 字→本地
    Herdsman-only 压 200 字，失败退 300 字截断）+ 截图留档（默认关）；
    ② intent LLM 兜底（默认关）——ParseFallback 白名单 navigate/status/
    read_screen + 0.75 置信门 + 围栏容错；classifyIntentWithLLM routine 目标
    + intents_llm_timeout_ms 硬超时 2s + manifest 校验；dryRun 恒不调用；
    ③ 生图 CardPath 接通——勘误「异步」实为同步阻塞，首图 FilePath 即
    CardPath，微信回推数据源打通；
    ④ iLink 离线收敛——per-peer 限频 20 条/分 + 4KB 截断 + 多媒体上限 5 +
    DownloadImage 三重防线（SSRF/20MiB/魔数）+ OCRMediaRecognizer 注入接线 +
    imageItem/fileItem 防御 UnmarshalJSON + SendFileCard seam + 协议文档；
    ⑤ 全局离线模式 offline_mode（默认关）——EngineType.IsLocal() +
    routeModel 三步云过滤 + LLM 兜底联动，跨版欠账清账；
    ⑥ 成本知识图谱——costref.BuildGraph 纯函数组图器（7 节点/6 边、tree
    聚合/entry 展开双视角、EntryName 精确优先、截断/悬挂边/去重防护）+
    GaeaCostGraph 绑定（532→533）+ CostGraphView 零依赖 SVG + 成本库第 8 模块；
    ⑦ Realtime S0——internal/realtime seam（RealtimeSession/Event 9 常量/
    kind 注册表 fail-closed/openai 实现）+ VoiceHealth realtimeReady + 优雅
    降级，14 离线测试，S1/S2 留欠账。
  - **验证**：Go 全量绿（120 包，零 FAIL）；vitest **800/800**（146 文件）；
    tsc/eslint 0；绑定面 533、spaceBindings 250、drift PASS；版本五处统一
    4.8.0。
  - **欠账**：Realtime S1/S2（DPAPI/key UI/端到端接管/打断联动，需真 key
    真机）/ iLink 真机窗口（原始 JSON/上传域/sendmessage 端点）/ 离线模式
    设置 UI（v4.8.1）/ 权限升级请求+stubGate 竞态 / XlsxPreview 虚拟滚动、
    生命库可写化=观察项。

- **最新发布：v4.7.0（2026-08-30）「命令面板接内核 · 读屏」**：
  git tag `v4.7.0`；CHANGELOG / releases/v4.7.0.md / README 索引同步。要点：
  - **S4.6 命令面板接统一路由（完整收口）**：GaeaRouteIntent(text, dryRun)
    绑定（531→**532**，S4.6 显式豁免旧「零新增绑定」纪律并头注记录）——
    dryRun=true 零副作用预览（校验口径与执行层一致），false 真执行；
    SearchModal 指令预览卡（命中出卡、点「执行」才真跑——搜索词不是整句指令
    入口，宁漏勿误；回执内联，导航类 emit gaea-intent-navigate 切板块收面板）；
    **真·Ctrl+K**（MainLayout 全局快捷键，tooltip 名副其实；gaea 工作台内让位
    自有 CommandPalette 防双面板）。
  - **屏幕感知能力「读一下屏幕」**：intent.ActionReadScreen 窄规则（读/念/看/
    识别+屏幕、屏幕上有什么、读屏；不含裸读/看）+ execReadScreen（screen.Capture
    → 临时 PNG 即用即删 → GaeaOCRText 既有 OCR 链 → 300 字截断回传）；
    语音 TTS 朗读/面板内联/微信回推三入口免费受益；失败诚实回执不坠聊天。
  - **前端配套**：IntentResultView WireShape + typesGenerationCheck 断言；
    spaceBindings RouteIntent=shared（248→249）；mock RouteIntent；三语 i18n
    8 key；SearchModal.test 5 用例。
  - **验证**：Go 全量绿（121 包）；vitest **796/796**（145 文件）；tsc/eslint 0；
    绑定面 532、spaceBindings 249、drift PASS；版本五处统一 4.7.0；桌面端
    gaea-v4.7.0.exe（35MB，SHA256 e83934db）+ 冒烟 200。
  - **欠账**：iLink 真机收敛 / 成本知识图谱可视化 / 读屏纵深（OCR 摘要再朗读、
    多显示器）/ 实时语音排期 / 生命库可写化=不盲写。

- **最新发布：v4.6.1（2026-08-30）「微信统一路由 · 规范包机制 · 归因对标」**：
  git tag `v4.6.1`；CHANGELOG / releases/v4.6.1.md / README 索引同步。要点：
  - **S4.5 微信接统一路由**：`routeIntentWithResult`（产物感知；routeIntent
    包装不变）+ 微信回调提醒之后先过统一路由（navigate/生图/状态/提醒全命中），
    未命中才走聊天；iLink image_item/file_item 防御解析 + 非文本转模型提示行。
  - **规范包机制化**：standard.Checker 注册表 + LintDocument 聚合；红头要素 +
    造价工程表式（工程名称/编制依据/单位造价/人材机/合计/说明 六要素）双检查器；
    OfficePanel 按规范包分组展示。
  - **成本归因对标**：ComputeAttribution 纯函数（P25/P75 带宽、差幅等级、贡献
    金额、TopDrivers）+ 新绑定 GaeaCostAttribution（531，参考池排除本项目防自
    对标）+ FiveCalcPanel 归因区。
  - **验证**：Go 全量绿；vitest 791/791；tsc/eslint 0；绑定面 531、spaceBindings
    248、drift PASS；欠账清单如实列示。

- **最新发布：v4.6.0（2026-08-30）「双空间收尾 · 纵深补课」**：
  git tag `v4.6.0`；CHANGELOG / releases/v4.6.0.md / README 索引同步。要点：
  - **审计补课背景**：v4.x 执行审计裁决「最小版执行」——红线缺口三条 +
    C 类纵深欠账（docs/audit-2026-08-30-v4-execution-review.md §B/§C）。本版
    按 §E 刀序第一轮收账。
  - **红线 ①记忆注入按空间收窄**：boot sysprompt 索引 + controller refresh 两
    个生产调用点传 `Options.Space` → `InSpace` 读端视图；work 只注入 work、
    play 只注入 play；mode=off 旧行为零变化。
  - **红线 ②任务分账生产启用**：`[tasks]` 段（max_concurrent/per_space/
    priority），默认 {work=1, play=1} + 价格抓取优先；显式空表关闭分账。
  - **红线 ③事件过滤推广**：`onTaskEvent(cb, space?)` 订阅层过滤 + 五个工位
    消费点 + MainLayout subscribeForSpace。
  - **治理收尾**：keepAlive 裸轮询 8 处全门控；CSS 真硬编码 token 化
    （novel 阅读高亮/批注色板 + chat-board 混白）。
  - **纵深**：Mood→TTS 连续韵律闭环；Verifier 通道 B 真视觉 diff（像素差异率
    + 页数联合判定，审计产物落 journal/verify）+ 失败回 Plan（xlsx 重新规划）；
    询价异常分级 + 线性回归价格预测 + OCR 报价单自动幂等入询价库。
  - **验证**：Go 全量绿；vitest **791/791**；tsc/eslint 0；绑定面不变（变参
    向后兼容）；欠账清单如实列示于 releases/v4.6.0.md。

- **最新发布：v4.5.0（2026-08-30）「指令中枢」· 统一意图路由内核 + 语音指令**：
  git tag `v4.5.0`；CHANGELOG / releases/v4.5.0.md / README 索引同步。要点：
  - **规划修订背景（§10.4a）**：v4.0–v4.4 执行复盘发现 §8 革命性跃升未实质落地，
    重定主轴——本版落地触点层「同内核多入口」架构承诺（意图→能力→结果回传
    统一路由，语音/微信/命令面板共用）。
  - **intent 解析包（S4.1）**：纯函数规则引擎（导航/生图/状态/提醒 + 板块别名表
    贪婪匹配）；宁漏勿误纪律（闲聊「画得不错」绝不触发生图）；LLM 兜底留位。
  - **能力执行层（S4.2）**：`App.routeIntent` 统一入口——navigate（manifest 校验 +
    `gaea-intent-navigate` 事件）/ generate_image / status / reminder（复用离线
    代办）；**零新增 Wails 绑定**（530 不变，结果走事件 + TTS 回传）。
  - **语音指令通路（S4.3，JARVIS 一档）**：voice 对话回调先过路由，命中即能力
    执行经同一 TTS 流程播报（含 barge-in），未命中透传轻语聊天；voice 包零改动。
  - **前端导航（S4.4）**：MainLayout 订阅 INTENT_NAVIGATE → navigateBoard
    （语音「打开绘梦」自动切空间）。
  - **验证**：Go **108/108 包**；vitest **789/789**；tsc/eslint 0；版本五处统一
    4.5.0；桌面端 gaea-v4.5.0.exe（35MB，SHA256 027c726d）+ 冒烟 200。
  - **下一执行**：v4.5.1 微信接统一路由 + iLink 图片/文件卡片协议；v4.5.2
    Ctrl+K 命令面板接内核；v4.6 端到端实时语音（云端 Realtime 档）+ 离线总开关。

- **最新发布：v4.4.0（2026-08-30）「触点」一期·微信遥控器：离线代办**：
  git tag `v4.4.0`；CHANGELOG / releases/v4.4.0.md / README 索引同步。要点：
  - **定位**：路线图 §10.4 v4.4 第一刀——微信从「能聊天」升级为「能接活的
    遥控器」。离线代办 = 官方元宝做不了的桌面端差异化主打（桌面常驻 +
    微信回推）。
  - **主动推送通路**：weixin.Server 记录最近活跃会话（fromUser/contextToken），
    新增 Push 主动回推文本；httptest 校验目标/item。
  - **离线代办提醒域**（weixin_reminder.go）：中文时间解析（相对时长 / 日期
    前缀+段词「明天早上9点/明晚8点半/后天十点」/ 裸时刻；中文数字含「十」
    进位；无段词字面解释——确认文案带完整时间供纠正，不做玄学歧义）→
    wxReminder JSON 持久化（重启恢复）→ tryWxReminder 微信消息任务路由
    （提醒类接管，解析失败回格式提示）→ 20s ticker 到点回推（失败重试
    ≤5 次标 failed）。配置 remindersEnabled（weixin_task.json）。
  - **WeixinPage 落地（书房板块）**：扫码绑定流（QR 轮询/配对码/confirmed 落
    WhisperAssistantSave 重拉通道）+ 通道状态徽标 + 提醒列表（手动新建/删除/
    开关）+ 指令说明；weixin 板块 inMenu=true 进 rail 与首页左翼书房格
    （双翼 manifest 派生自动生效）。
  - **绑定面 525→530**：WhisperWeixin* 4 + WhisperAssistant* 3 自
    LegacySurfaceNames 转正，WeixinReminder* 5 新增（voice 门面）；
    spaceBindings 235→247 全归 work。
  - **验证**：Go **107/107 包**；vitest **789/789**（锁数量断言 +12 同步）；
    tsc -b / eslint 0；drift 530 PASS；版本五处统一 4.4.0；桌面端
    gaea-v4.4.0.exe（35MB，SHA256 ee9b45c2）+ 冒烟 200。
  - **下一执行**：v4.4.1 触发已有能力（生图/办公任务路由 + iLink 图片/文件
    卡片协议探明）+ 语音双通路（work 指令 / play 闲聊人格分叉，§10.4）；
    观察项（全局离线模式总开关、中庭对话条桌面端体验、主动关心配置面板、
    角色 gallery 管理、IP-Adapter 节点级参考槽）。

- **最新发布：v4.3.2（2026-08-30）「双翼·中庭」首页重构 + 空间导航收敛**：
  git tag `v4.3.2`；CHANGELOG / releases/v4.3.2.md / README 索引同步。要点：
  - **首页「双翼·中庭」**：中庭 = 语音 + 打字一体对话条（输入框打字走
    `VoiceChatText` 复用语音对话管道，回复经 voice:reply 回传；orb 放大
    148px 呼吸环磁吸锚点；hero 让位顶部细眉——遵循 design-taste-frontend：
    不对称组织、避免等宽栏与居中 hero 俗套）；左翼「书房」2×2 紧凑格
    （办公/造价/记忆/模型）；右翼「庭院」纵向列表（聊天/小说/绘梦/角色）；
    门廊 = 编程（独立窗口徽标）+ 设置。
  - **命名**：工位→**书房**、乐园→**庭院**（与「会客厅/创作间」宅邸体系
    一脉相承；三语字典 zh/zh-TW/en 同步；搜索 scope 文案同步）。
  - **空间导航收敛**：移除一级导航（rail）顶部空间切换按钮；`navigateBoard`
    按板块 manifest.space 自动切空间（书房板块→work / 庭院板块→play /
    编程→independent / 设置→shared）；rail 展示全部板块。
  - **验证**：Go 全量绿；vitest **789/789**；tsc -b / eslint 0；版本五处
    统一 4.3.2；桌面端 gaea-v4.3.2.exe（33MB，SHA256 6a0486db）+ 冒烟 200。
  - **下一执行**：v4.4 触点（微信任务入口 + 语音双通路 + 本地离线模式，§10.4）；
    观察项（中庭对话条桌面端体验、主动关心配置面板、角色 gallery 管理、
    IP-Adapter 节点级参考槽）。

- **最新发布：v4.3.1（2026-08-30）「乐园」后续小步**：
  git tag `v4.3.1`；CHANGELOG / releases/v4.3.1.md / README 索引同步。要点：
  - **主动关心定时推送频控（v4.3c 补完）**：app 层 ticker 四信号评估（AttentionManager
    频控 ≤3 条/小时 → MatchHabits dnd 作息尊重 → DetectSpecialDatesV2 生日祝福
    （每天首条、人格感知提示词）→ 门控+合成器）→ `gaea-whisper-proactive` 事件推
    前端（space=play）；新绑定 GaeaWhisperProactiveConfig/SetProactiveConfig；
    前端 WhisperGraphPanel 订阅显示推送气泡（birthday 徽标）。play 红线零落盘。
  - **创作间世界模型面板（v4.3e/f 落地）**：设定页「维度化」模式（6 维度卡片就地
    编辑整存）+ 伏笔登记表面板（状态流转/回收率）+ 一致性检查面板（三类告警）。
  - **角色参考图 + 生图参考槽（v4.3g 补完）**：characterlib SchemaV2 迁移（reference/
    gallery_images 两列幂等）+ CharacterGeneratePortraitWithRef（img2img 参考槽
    denoise 0.55 + 模型门禁）+ 前端参考图管理。
  - **文本朗读情绪 UI（v4.3d 收尾）**：EmotionSpeakSelector（9 标签对齐
    EmotionVoiceMap）+ 会话情绪自动跟随 + TTSSpeakBase64WithParams 携带情绪。
  - **验证**：Go 全量 **118/118 包**；vitest **789/789**（144 文件，+20）；
    tsc -b / eslint 0；绑定面 **525** 漂移 PASS（+3）；spaceBindings **235** 全覆盖；
    版本五处统一 4.3.1。
  - **下一执行**：v4.4 触点（微信任务入口 + 语音双通路 + 本地离线模式，§10.4）；
    观察项（主动关心配置前端面板、角色 gallery 前端管理、IP-Adapter 节点级参考槽）。

- **最新发布：v4.3.0（2026-08-29）「乐园」娱乐做深**：
  git tag `v4.3.0`；CHANGELOG / releases/v4.3.0.md / README 索引同步；设计
  `docs/gaea-v43-play-deepen-design.md`（4 份只读调研结论入账）。要点：
  - **会客厅关系记忆**：三表（associations/habits/temporal_anchors）repo 持久化
    闭环 + ReseedAssociationGraph 打通 + hermes.db 外键延迟检查；QuerySubgraph
    多跳子图召回 + GaeaWhisperGraphSubgraph（play）+ WhisperGraphPanel（SVG 图谱）。
  - **主动关心**：GaeaWhisperProactiveNow 评估绑定 + 前端「轻语先开口」；
    定时推送/频控列后续小步。
  - **情感语音**：TTS SynthesizeWithParams 参数扩展（cosyvoice 工厂修复/edge SSML
    参数化）+ 情绪→参数映射 + 长期心境维 Mood（EWMA 持久化）+ TTSSpeakBase64WithParams。
  - **创作间图文联动**：章节配图复活死绑定 + GaeaGenerateBookCover 3:4 书封
    （play exports）。
  - **验证**：Go 全量绿；vitest **769/769**（144 文件）；tsc -b / eslint 0；
    绑定面 **522** 漂移 PASS（+5）；spaceBindings **233** 全覆盖；版本五处统一 4.3.0。
  - **桌面端**：gaea-v4.3.0.exe（34.9MB，SHA256 7b8d3cf4…，releases/SHA256SUMS-v4.3.0.txt），
    wails v2.13.0 构建 + 冒烟 /api/health 200；v3.0.8/v3.1.0 归档（保留 5 版）。
  - **下一执行**：v4.4 触点（微信任务入口 + 语音双通路 + 本地离线模式，§10.4）；
    v4.3 后续小步（主动关心定时推送、文本朗读情绪 UI、设定页维度化编辑器/
    伏笔登记表/一致性面板、角色参考图 IP-Adapter）。

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

## 2026-09-11 四迁腾预算（自 .gaea/AGENTS.md 迁入：v4.183~v4.179 五条）

- **最新发布：v4.183.0（2026-09-09）「双空间首页重排版：书斋文书台/闲庭游园画廊 编辑部级排印」**——用户反馈 v4.182 版式「太 low」，定位=同质卡片海（8 等大 Bento 瓦片+右舷五连盒）是模板感根源；重排版信息零删除只动形态，ui-ux-pro-max 双查询定方向（书斋=Premium refined，闲庭=Bold gallery variance8/density3）。**书斋「文书台」**=编辑部排印：竖排空间名书脊 masthead（vertical-rl，≤900 收起）+30px 大字渐变标题（background-clip:text）+命令条（focus-within 辉光）+最近文档卷宗流水（hairline 报表行+等宽序号 hover 点亮）+旗舰横带（accent 竖条+细网格纹）+**目录式索引行**（等宽序号 01-05 两列报表 hover 辉光，替代等大瓦片）+右栏单一仪表纵栏（遥测/写作/会话/记忆/晨报 hairline 分节，去五连盒；≤1180 落底双列）。**闲庭「游园画廊」**=画廊排印：居中大字 clamp42+宽字距+空间名小签两侧渐隐线+会客厅旗舰横幅（**月洞门圆环母题**+84px 圆徽记+CTA 胶囊）+**竖版海报大卡**（62px 圆徽记+底部环形箭头 hover 滑入；设置=行形态）+园底单条信息带（进度环/继续话题 chips/记忆/遥测 hairline 四节，≤1100 折 2×2）。**结构修复（v4.182 遗留缺陷）**=M3 tertiary 角色全仓从未定义（切换器暖色恒走 fallback，闲庭与书斋实际同色）——appStore ThemeTokens 增 tertiaryContainer/onTertiaryContainer+TERTIARY 暖伴侣色表（6 预设×明暗）+getThemeTokens 合并+App.tsx 注入两 CSS 变量，契约测试锁定（12 套 tertiary 对存在且不等）。**坑**：bridge 浏览器 dev mock 优先于 window.go.main.App 桩（隔离目检须打 window.go.app+Gaea* 映射键名才走 realApp 真机路径）；晨报绑定回 JSON 字符串非对象。locale 零新增键；testid/aria 全保留（ModuleLauncher.test 3 例原样绿）。**门禁**=vitest **2746**（+1 tertiary 契约）首跑全绿/tsc 0/eslint 0/e-check OK/locale 0 死键/drift PASS@602/零 Go 改动（版本三处 sync 4.183.0 外）/build strip+冒烟 200/SHA256=a5f54c93af96605c89cec94f22cc31835f684d124552e82e360355d0ead93a09（exe 46.2MB）。目检=esbuild 隔离挂载+Edge 无头截图（书斋满数据/闲庭满数据/书斋 1000px 窄档三张）。欠账=壳内真机走查两首页观感/切换手感（真机池，含 v4.182 欠账）；浅色主题首页观感未目检。


- **v4.182.0（2026-09-09）「双空间首页分版：书斋/闲庭定名 + 切换器迁首页顶栏」**——用户拍板切换按钮迁首页顶栏+两首页各有特色不要长得一样。**定名**=书斋（work，zh-TW 書齋，en Study）/闲庭（play，zh-TW 閒庭，en Lounge），三语覆盖 shell.space.*/shell.search.scope.*+space.ts 兜底 label；组件无运行时硬编码（仅注释）。**切换器迁移**=首页顶栏 SpaceSwitch（SHELL_SPACES 驱动胶囊组 aria-pressed+当前空间点击防抖+闲庭激活 tertiary 暖色令牌+右侧模型徽标，直连 MainLayout.switchSpace）；rail 顶部改竖排空间指示徽标（非交互 div+title，空间感知保留切换动作归一首页），rail 分域导航不变。**书斋「文书台」**=三栏效率台（顶栏+Hero 命令条 AI 直启+最近文档流水主角面板+能力矩阵 Bento+右舷遥测/写作/会话/记忆/晨报；density7/Flat）。**闲庭「游园画廊」**=全幅无右舷无命令条（居中标题+会客厅旗舰横幅全宽渐变大卡+板块大卡两列画廊+园底信息带；density4/Showcase）——信息零删除形态分化。ModuleLauncher 重构=useLauncherData 顶层一次拉取+DeskHome/GardenHome 两变体分发。测试=ModuleLauncher.test 3 例+CommandRail.test 更新+space.test 同步；vitest 2745/tsc 0/eslint 0/drift PASS@602/SHA256=F8B238863236F53803AB07822A1136366EEC92E97403BC5B592C1C04214E1337。欠账=壳内真机走查两首页（v4.183 重排版后合并走查）。


- **最新发布：v4.181.0（2026-09-09）「上下文/轨迹按会话读取 + GLM Coding Plan 429 分诊」**——用户反馈 Agent 网络不按当前会话。**根因**=GaeaTrajectory/GaeaAgentNetwork **无参绑定恒读内核 `gaeaCtrl().SessionPath()`**（ga.ctrl 单例恒指向最近活跃会话），UI 切历史会话后子代理 runs（有参）正确切换而网络树/轨迹仍是内核会话内容——「UI 会话 ≠ 内核会话」是这族 bug 的共同根因（GaeaHistory/ContextView 等其余无参绑定待逐个审计）。**修复**：①两绑定变参 `sessionPath ...string`（显式路径优先/缺省回落内核会话兼容旧调用，GaeaTaskList 先例；resolveGaeaSessionPath 归一+gaeaContextWindow 拆出），门面两行透传，**绑定名 602 零变更 drift PASS**；②bridge 契约 JS 数组形态（Go 变参→JS 数组，TaskList 先例；**proxy 是裸转发勿声明单 string**）+mock 同步签名；③agentNetworkStore 路径声明=poller 增 path，subscribe(cb,{path})/reload(path)，**跨会话切换清空快照置 loading 立即重拉（旧会话树=误导数据宁空勿错）/同会话重试宁旧勿断不变**；④六消费方接线=ContextView（订阅+load reload+依赖补齐）、SubagentsPanel/TasksWorkbench（订阅+prevPath 切换+refresh+retry 全带 path）、TrajectoryView（新 sessionPath prop，App 传 currentSessionPath）、AgentNetworkCard（load 带参）；BrowserPanel 保持内核兜底（无会话上下文的注入式组件，fetchTrajectory 注入点留给消费方）。**附带 GLM 429 分诊**：用户实测 GLM 引擎 429「无可用资源包」——编码套餐资源包挂 coding 端点（/api/coding/paas/v4）计费域，标准端点（/api/paas/v4）查不到即 429，**不是 Key 坏是端点没切**（模型中心 GLM 卡片 std/coding 切换已有）；glmPing 标准端点 429 时错误信息追加分诊提示。**测试坑×2**：纯 Node store 测试无 act（用与原用例一致的 `vi.advanceTimersByTimeAsync(0)` flush，非 fake timers describe 用 `setTimeout(0)` macrotask）；测试间 poller 单例残留连累后续用例（用例内自清理所有订阅）。门禁=Go 全量 0 FAIL（+TestGaeaTrajectoryAndNetworkExplicitSessionPath 磁盘日志+ctrl=nil 下显式路径生效）/vitest 2742（+3 路径声明用例；全量两轮收敛至在册 ContextView/MemoryPanel/TrajectoryView 负载 flaky 族单独复跑 43/43 绿）/tsc 0/eslint 0/e-check OK/drift PASS@602/build strip+冒烟 200/SHA256=D3BD8386862D5F74EC2BC63F8CA5B35394FA812588376DA306ECAC078AA62D55。欠账=TaskCenter 会话归属（任务表 session 维度）挂结构刀池；GaeaHistory/ContextView 等无参绑定「内核会话」语义逐个审计。


- **最新发布：v4.180.0（2026-09-09）「办公任务管理会话关联刀：会话待办区入任务管理页」**——用户反馈任务管理「没与当前会话关联、全是固定内容」。**定位**=任务管理页（TasksWorkbench，GaeaPage→ContextView→AgentRadial 三层下）三区中子代理/本地模型工具本就按 sessionPath 建册，但页面主体是 ② TaskCenter——数据源 `GaeaTaskList` 是**全局后台任务表**（价格抓取/语义索引等系统 cron 周期任务持续在列），天然会话无关=用户看到的固定内容；会话真任务（todo_write 待办）反而无入口。**修复**=TasksWorkbench 首区新增「会话待办」：直订全局 controller store `useStore((s) => s.items)`（当前会话活动流随会话切换，零 props 穿层，ContextModal 弹层实例同源自动正确）+ 复用 `useTodoExtractor` 提取最新 todo_write（倒序取无 parentId 最新一条=与聊天流待办卡同一规则单一来源）；渲染=区块头「会话待办 n/m」+三态条目（in_progress 蓝点脉冲/pending 灰点/completed 绿点划线置灰，level 子步骤缩进），**无待办不占位**。页面四区语义=会话待办/子代理/本地模型工具（均会话）+任务中心（全局 work 空间）。**坑=测试 vi.mock("../lib/bridge") 工厂仅给 onEvent，TasksWorkbench 新 import lib/store（controller.ts import app/onReady）后 vitest 对缺失 named export 宽松处理未炸——新组件引入真 store 时 mock 工厂补齐 named exports 更稳**。门禁=vitest **2739/2739 首跑全绿**（+1：待办区无→出现→清单更新三段用例）/tsc 0/eslint 0/e-check OK/drift PASS@602（零绑定）/零 Go 改动/build strip+冒烟 200/SHA256=69C89001048CB8BFD99BCE7122C69AB92A8C01A85CB4E34D23BAA45A1A71C0DD。欠账=TaskCenter 会话关联需 Go 任务表加 session 维度（Schema+提交链路=结构刀另立版本，当前以区块语义区隔过渡观察反馈）；观察池=「会话后台任务」过滤视图等 session 维度就位。


- **最新发布：v4.179.0（2026-09-09）「瘦身清点刀：knip 死代码清除 + 依赖显式化 + 冷启动基线打点」**——瘦身总规划 P0-P4 收官后清点刀（轨道二尾欠 knip + 轨道五运行面测量）。**knip 首扫三发现三处置**：①死文件 13 全删（App.css/AppBar/MobileSheet/SettingsMobile/SettingsUpdates/自制 Tooltip=各处用的均 antd/PromptShelf/office ParseSummaryCards+SourcePreviewDrawer 互为唯一引用死链对/memoryhub ComingSoon+ModuleCard/typesGenerationCheck/m3-palette——全部零 import 甄别含 scripts 层；连带清 zIndex.ts 过时注释档位值不动）；②幽灵依赖 13 包显式化=20 处 import 靠传递依赖侥幸工作升级即断（jszip docx/pptx 链 8 处、dayjs WeixinPage/GanttToolbar、katex Markdown 直引 CSS、unified+hast-util-sanitize sanitize.ts、@lezer 八子包 diffHighlight.ts——按 lock 现解析版本钉死零漂移）；③npm install 顺带清死传递依赖 @codemirror/search@6.7.2（全仓零引用）；**三存疑依赖验活结案**（v4.166 名单）：gsap=2 hook×6 组件消费/docx-preview=DocxPreview/unist-util-visit=remark 两插件全部在用保留。**冷启动基线打点（轨道五先测量后优化）**：Go 侧 app.New 构造链一条+Startup 七段（migrate/logging/engine/voice+weixin/charlib+stores/tts-kickoff/schedulers）+total 汇总 slog——日常启动自动落 logs/gaea-YYYYMMDD.log 真机基线零成本积累；前端 index.html 解析即 performance.mark('gaea:boot')→React 挂载后第二个 rAF gaea:interactive+measure（CDP getEntriesByName 可读+console.info；**有意不落 gaea.log——现有前端日志通道 GaeaLogFrontendError 是 slog.Error 级，不污染错误信号**）。**初始化链审计结案=懒初始化改造无需立项**：TTS 拉起幂等异步/价格源+文件索引首轮=任务队列异步提交/四引擎模型刷新 goroutine/预载延迟 10s/巡检延迟 1min——同步 Startup 链只剩配置/嵌入 FS/DB 打开/角色库迁移轻量磁盘活，轨道五收敛为真机冷启动/常驻内存采集（与壳内真机池同窗口）。门禁=vitest 2738/2738（5 例 ContextView/MemoryPanel/TrajectoryView 并发负载 flaky 单独复跑全绿在册先例）/tsc 0/eslint 0/drift PASS@602（零绑定）/e-check OK/Go 116 包 0 FAIL/build strip+冒烟 200。SHA256=3FCB187F496AF2B1A89884D860F675D1E56A92B6B50AA932E0655A65521D320F。度量=exe 46.16MB 持平（死文件本就不进 bundle，本刀收益在仓库卫生与依赖正确性）；knip 复扫 Unused files 0/Unlisted dependencies 0。欠账=knip Unused exports 169 项挂下版（甄别三态：测试专用/预留面/真死，可配「新导出须有消费方」门禁）+壳内真机池/审计刀D 不变。

## 2026-09-11 五迁腾预算（自 .gaea/AGENTS.md 迁入：v4.208/v4.207-205/v4.204/v4.203 四条）

- **最新发布：v4.208.0（2026-09-10）「首页 v7 重设计：书斋「文书台」/ 闲庭「游园画廊」」**——v6 版式廉价感定位=「等大圆角卡片+描边」一种形态重复到底。**v7 结构三形态**（仪表条 hairline 行 / 账页等宽序号行 / 海报墙真大小跨格）+ **四档排印**（display clamp 负字距 / lede / body / label 宽字距，数据等宽）+ **三级色调面**（surface→container→container-high，描边退 hairline）+ 版式记号（序号水印 / 月洞门细环）取代极光斑；信息零删除，契约 testid（ml-space-* / desk-recent-docs / garden-*）全保留。**坑：CSS 重写会静默吃掉「同名 class 的规则」**——`.ml-avatar-ai` / `.ml-avatar-user` 在 HEAD 有规则、v7 新版没有（TSX 仍在用），两个头像退化为同底色；**检查配方=脚本双向比对「TSX className 集合 vs CSS 选择器集合」**，本刀跑出 3 处 TSX 无规则（2 真丢 + 1 空挂 `.p-foot-wide`）后修正。附带**门禁可信化**：CI「无条件重试一次」→「只在册 flaky 隔离复跑」（`known-flaky.txt` 现为空 + 分类器自测）、`maxWorkers` 按物理核夹 [4,8]（原按逻辑核开 ~31 worker→重组件首个 `await import` 撞超时假红）、testTimeout 30s、RTL `asyncUtilTimeout` 5s；**教训=afterEach 清 body 会打挂 antd 模块级 message 容器（实测一次 14 例假红）**，残留改由并发上限从源头消除。门禁=tsc/eslint 0 / e-check OK / drift PASS@608（零绑定）/ Go 全量 / vitest 324 文件 2799 例 / 版本三处 4.208.0 / 产物=主树 wails build -ldflags "-s -w" -trimpath（1m08.5s）→ build\bin\gaea.exe 48,570,880 B（46.32MB）+ 冒烟 /api/health 200 + Desktop/build\bin 双副本；SHA256=59c2b518bdb6165ccbac07bbc565759a2d7571f254e287e27aa00b2bccd97a16（releases/SHA256SUMS-v4.208.0.txt）——构建后工作树仍干净（wails 的 npm install 未改写 lockfile）/ 目检=dev(`?mock=1`)+无头 Edge CDP 两空间 ×4 档共 10 张（无横向溢出）+ **壳内实测**（构建产物 `build\bin\gaea.exe` + `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333`：书斋/闲庭**浅色主题+真实数据**渲染正常——内核 6 引擎/真实最近文档 5 条/记忆 1706 条/晨报 9 条、越界元素 0、**真点击切换器原地切换** aria-pressed 与 localStorage 同步、无 reload 绕过）。欠账=海报墙首张（小说）大样中段留白偏多（观感待定）；动效手感（入场分阶 / hover 位移节奏）待上手定论（浅色观感已见实机）；真机池余项=.gsched「基线 N」chips / 最近文档 localStorage 写路径 / 冷启动基线采集。
- **v4.207.0 / v4.206.0 / v4.205.0（2026-09-10）主题令牌收口 + 小说书房工坊接线**——v4.207 深色主题硬编码深灰（RelationGraph 图例/提示、进度里程碑菱形与标签）改走 --color-text / on-surface；v4.206 33 处 `bg-accent text-white`→`text-accent-fg`（= on-primary）+ App 注入 --color-on-primary / --color-surface；v4.205 小说书架画廊接线（画廊头 + 正在编辑横条 + 书脊/角标）+ 阅读页属性检查器由 display:none 改默认可折叠（章节体检入口回来）。三刀均纯前端、绑定 608 零变更、Go 零改动。
- **最新发布：v4.204.0（2026-09-10）「组价多方案对照：P25/中位/P75 当场切」**——造价刀路池 §2 半刀。价格带三档数字早就在，推荐价钉死中位数。本刀不拆逐步盖章：ComposeModal 三档格可点（aria-pressed，应用价随档，默认中位数既有用例不动）；cost_compose 增 mode 参数（未知回落中位数）+输出「三档对照」一行。绑定 608 零变更。门禁=ComposeModal 11/11（+1）/Go TestCostCompose 绿/tsc 0/eslint 0/版本三处 4.204.0。欠账=流式打字机/费率政策对照不做；含量基线等样本；壳内真机走查池不变。
- **最新发布：v4.203.0（2026-09-10）「空间策略其余功能域键：总闸当场切」**——v4.190 欠账收刀：七键写路径早已通，编辑 UI 却只放 gaea 主控。本刀纯前端（绑定 608 零变更）把 chat/whisper/novel/office/characterlib/routine 拉进同一张空间策略卡：六键常显（空键灰字「未配置（维持现状）」），行级编辑闭环与 gaea 同款（带出当前值 → GaeaSpaceProfileSet(space, key, ref) → 视图随返回刷新；空串=清除；后端校验失败原样 message.error）。gaea 主控行与既有 testid 不动。阶段二总闸画面出口补齐「书斋和闲庭可以各绑各的」。门禁=StrategySection 8/8（+2）/tsc 0/eslint 0/版本三处 4.203.0。欠账=功能域×空间交叉矩阵按需另刀；壳内真机走查池不变。
