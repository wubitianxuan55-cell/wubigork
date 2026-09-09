# 任务进度

> 本文件为**最近发布速览**：仅保留最近 7 条版本记录（v4.174.0 ~ v4.180.0）。
> v4.173.0 及之前的完整历史磁带（90 条记录 + 历史期章节）已于 2026-09-09 瘦身归档至
> `docs/archive/progress-history-2026-09.md`；历史详情请读归档文件。

## 最新发布：v4.180.0（2026-09-09）「办公任务管理会话关联刀：会话待办区入任务管理页」

- **定位**：用户反馈任务管理「没与当前会话关联、全是固定内容」——页面主体 TaskCenter 数据源 `GaeaTaskList`=全局后台任务表（cron 周期任务持续在列，天然会话无关），而会话真任务（todo_write 待办）无入口；子代理/本地模型工具两区本就按会话建册。
- **修复**：TasksWorkbench 首区新增「会话待办」——直订全局 controller store items（零 props 穿层，会话切换自然更替，ContextModal 弹层同源正确）+ 复用 useTodoExtractor 提取最新 todo_write；三态条目+进度 n/m，无待办不占位。页面四区=会话待办/子代理/本地模型工具（会话）+任务中心（全局）。
- **门禁**：vitest **2739/2739 首跑全绿**（+1）、tsc 0、eslint 0、e-check OK、drift **PASS@602**（零绑定）、零 Go 改动、build strip+冒烟 200、SHA256=69C89001048CB8BFD99BCE7122C69AB92A8C01A85CB4E34D23BAA45A1A71C0DD。
- **欠账**：TaskCenter 会话关联需 Go 任务表加 session 维度（结构刀另立版本）；「会话后台任务」过滤视图等该维度就位再评估。

## 最新发布：v4.179.0（2026-09-09）「瘦身清点刀：knip 死代码清除 + 依赖显式化 + 冷启动基线打点」

- **knip 首扫三发现三处置**：①死文件 13 全删（App/AppBar/MobileSheet/SettingsMobile/SettingsUpdates/自制 Tooltip/PromptShelf/office 死链对/memoryhub 两件/typesGenerationCheck/m3-palette，全仓零 import 甄别含 scripts 层；连带清 zIndex.ts 过时注释）；②幽灵依赖 13 包显式化（jszip×8 处/katex/dayjs/unified/hast-util-sanitize/@lezer×8 共 20 处 import 靠传递依赖侥幸工作，按 lock 现版本钉死）；③顺带清死传递依赖 @codemirror/search@6.7.2（零引用）。三存疑依赖验活结案：gsap/docx-preview/unist-util-visit 全部在用保留。
- **冷启动基线打点（轨道五先测量后优化）**：Go app.New 一条+Startup 七段（migrate/logging/engine/voice+weixin/charlib+stores/tts-kickoff/schedulers）+total 落长期日志——日常启动自动积累真机基线；前端 gaea:boot mark→两 rAF gaea:interactive measure（CDP 可读；有意不落 gaea.log 避免污染 Error 级日志通道）。
- **初始化链审计结案**：TTS/价格/索引/模型刷新/预载/巡检全部已异步化，同步链只剩轻量磁盘活——懒初始化改造无需立项，轨道五收敛为真机采集（与壳内真机池同窗口）。
- **门禁**：vitest 2738/2738（5 例并发 flaky 单独复跑全绿）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、e-check OK、Go 116 包 0 FAIL、build strip+冒烟 200、SHA256=3FCB187F496AF2B1A89884D860F675D1E56A92B6B50AA932E0655A65521D320F。
- **度量**：exe 46.16MB 持平（死文件本就不进 bundle，收益在仓库卫生与依赖正确性）；knip 复扫死文件/幽灵依赖双清零。
- **欠账**：knip Unused exports 169 项挂下版（测试专用/预留面/真死三态甄别）；壳内真机池/审计刀D 真机取证不变。

## 最新发布：v4.178.0（2026-09-09）「造价·条目匹配键刀：定额编码贯通存储→导入→匹配→工具面→UI」

- **造价域缺口 §1 首刀**（docs/gaea-cost-domain-survey-2026-09.md）：条目匹配全靠标题字符串 → 编码体系贯通。
- **数据层**：SchemaV17 `cost_entries`/`cost_estimate_items` 增 `code`（归一化存储）+ `idx_cost_code`；迁移测试「新库全链 V1→V17 + V16 升级」双覆盖；`cost.NormalizeCode`（全角→半角/去空白/大写，空串=未录入）。
- **导入链**：表头 `fieldCode` 映射（定额编号/清单编码/项目编码/编码等 8 关键词最长胜出，「编号」仍噪声）；**编码优先匹配**——带码命中→覆盖提示、带码未命中→直接新增（同标题不同编码不标题兜底，杜绝跨子目误覆盖）、无码原语义；AI 解析 prompt 增 code 提取；vision 有意不动。
- **匹配消费方**：costref `entryIndexes` 三索引 + `matchEntry` 编码精确优先（matchedBy=code|entry_name|title）；带码未命中宁漏勿误配（测试钉死）。沉淀携带 item.Code + 引用条目继承编码。
- **工具面/UI**：cost_save 增 code 参数（覆盖未传保留原编码）+ cost_search 表格编码列；EntryModal 编码表单项 / LibraryView 编码 chip+行前缀 / ImportModal 可编辑编码列 / ProjectsView 明细编码输入+EntryPicker 带入；mock 同批贯通。
- **附带收口：e-check 三处陈旧守卫修复**（干净 HEAD 实测即红，非本刀引入）——E25 ChatSend/ChatImportTopic 接受 bridge 形态（v4.174 双轨退役甩下）；E24 MemoryHubPage→GraphView 接受 React.lazy 动态 import（v4.176 甩下）。
- **门禁**：Go 全量 0 FAIL（五包 +9 测试）、vitest **2738/2738**（2737+1）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、locale 0 死、e-check OK、build strip + 冒烟 200、SHA256=F83CDB9BED35C7B4F045CC835B59981B95B5B4B67E39EC1D5282EF8F2E199561。
- **欠账**：造价缺口 2-6 不变；编码存量回填等带码样本导入自然补全（不做标题猜测回填）；壳内真机池/审计刀D 真机取证不变。
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

## 历史发布（v4.173.0 及之前，已归档）

- **v4.173** 瘦身 P3 版3 批次二：chat/novel/settings 三族 22 文件迁 bridge（契约扩展 29 方法+元组契约错位修正）
- **v4.172** 瘦身 P3 版3 批次一：wailsjsCompat 首批 12 文件迁 bridge（契约扩展 16 方法+2A/2B 迁移；LegacySurfaceNames 292→276）

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
