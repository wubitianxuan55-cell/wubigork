# gaea 后端壳层调研报告（3.0 架构改造）

> 调研对象：C:\AI\wubigrok\internal\app（203 文件 / 约 3.6 万行）
> 方法：只读代码调研。先 glob 建 203 文件清单，按文件前缀/职责分类，精读装配、绑定、
> 注册表、主脑、办公引擎 5 组关键文件，并全量 grep 事件发射点与 provider 硬编码 switch。
> 所有结论均带文件路径 + 行号；报告与《docs/2026-08-15-gaea3-architecture-design.md》的
> as-is 盘点交叉核对（下文以「设计文档 §x」标注引用）。

---

## 1. 概览（203 文件分类清单）

```internal/app 共 203 个 .go 文件，其中非测试 112、测试 91（含 bindings_completeness_test.go
生成物测试）```

按职责分组如下（括号为文件数）：

### 1.1 核心装配与状态（4）
- ```app.go``` — App/core/writingState/mediaState/whisperState/officeState 定义、New/Startup/Shutdown、emit
- ```writing_state.go``` — 写作域项目管理（getPM/setPM/closePM）、initAgents、restoreImageBackend
- ```whisper_state.go``` — 轻语域 assistantMgr/weixin 通道（initWeixin/startAssistantWx/stopAssistantWx）
- ```app_info.go``` — AppVersion 常量、GetAppInfo、CHANGELOG 解析

### 1.2 绑定面（11 非测试 + 1 测试）
- ```bindings_core.go```（48 方法）/ ```bindings_office.go```（137）/
  ```bindings_memory.go```（31）/ ```bindings_cost.go```（22）/
  ```bindings_model.go```（34）/ ```bindings_voice.go```（45）/
  ```bindings_chat.go```（25）/ ```bindings_novel.go```（67）/
  ```bindings_image.go```（38）/ ```bindings_charlib.go```（15）— 合计 **462 个委托方法**
- ```bindings_manifest.go``` — NewBindings 装配清单（生成物）
- ```bindings_completeness_test.go``` — 反射完备性测试（生成物）

### 1.3 模块注册表与主脑（8 非测试 + 8 测试）
- ```module_registry.go``` / ```module_bindings.go``` — 模块注册表 + 4 模块注册 + RunModule
- ```main_brain.go``` — 意图识别 + 主脑派发
- ```brain_store.go``` / ```brain_main.go``` / ```brain_left.go``` /
  ```brain_right.go``` / ```brain_link_store.go``` / ```brain_materials.go``` /
  ```brain_bindings.go``` — 三脑统一访问层

### 1.4 办公引擎接入（44 个 gaea_*.go + 2 个 office_*.go）
- 引擎装配：```gaea_handler.go```（gaeaRuntime/gaeaLoadConfig/GaeaInit/GaeaReload）
- UI 绑定：```gaea_ui.go```（会话/历史/文件树）、```gaea_ui_meta.go```（Meta/Context/Balance/Jobs）、
  ```gaea_ui_extra.go```（设置面板/Provider/Permissions/gaeaCwd）
- 工具：```gaea_tools.go```（imageGenTool 等 6 个 ExtraTools）、```gaea_specialist_tools.go```（ocr/vision 等）
- 能力：```gaea_dream.go```（自动做梦）、```gaea_preview.go```（文档预览）、
  ```gaea_pinned.go```（资料夹）、```gaea_templates.go```（任务模板）、
  ```gaea_tasks.go```（任务调度器+gaea-task 事件）、```gaea_schedule.go```（keep-warm/auto-preload）、
  ```gaea_file_index.go```、```gaea_price_sources.go```、```gaea_summarize.go```、
  ```gaea_translate.go```、```gaea_export.go```、```gaea_crosslink.go```、
  ```gaea_docx_edit.go```、```gaea_xlsx_edit.go```、```gaea_factbase.go```、
  ```gaea_diagram.go```、```gaea_ocr.go```、```gaea_vision.go```、
  ```gaea_data_backup.go```、```gaea_benchmark.go```、```gaea_stream_probe.go```、
  ```gaea_routine_llm.go```、```gaea_usage_overview.go```、```gaea_retrieval_eval.go```、
  ```gaea_unified_search.go```、```gaea_semantic_search.go```、
  ```gaea_workspace_search.go```、```gaea_cost_import(_vision).go```、```gaea_cost_rerank.go```、
  ```gaea_memory_lifecycle.go```、```gaea_memory_meta.go```、```gaea_memory_suggestions.go```、
  ```gaea_knowledge_meta.go```、```gaea_knowledge_import.go```、
  ```gaea_herdsman_health.go```、```gaea_herdsman_probe.go```、```gaea_herdsman_sec.go```
- 办公域：```office_handler.go```（Office* 桌面动作）、```office_skill_capture.go```（技能沉淀）

### 1.5 Herdsman 底座（6）
```herdsman_catalog.go``` / ```herdsman_operations.go``` / ```herdsman_digitallife.go``` /
```herdsman_lifecycle.go``` / ```herdsman_disk.go``` / ```herdsman_stats.go``` — 全部映射到 model 门面

### 1.6 语音 / TTS / OCR / 轻语（8）
```voice_handler.go```（voiceEmitter + initVoice）、```voice_model_handler.go```（ASR/TTS 模型切换）、
```tts_handler.go```、```tts_service.go```（本地 TTS 服务保活）、```ocr_model_handler.go```、
```whisper_handler.go```（29 方法，轻语主面）、```whisper_taskplan.go```、```whisper_write_observability.go```

### 1.7 模型路由 / 引擎（2）
```model_router.go```（routeModel/routeSensitiveLocal + model.route 事件）、```model_engine_handler.go```

### 1.8 聊天 / 记忆中枢（3）
```chat_service.go```（ChatSend/ChatStreamPlain/ChatTopicsList）、```chat_handler.go```（ChatGeneral）、
```memory_hub.go```（记忆中枢 8 库聚合 + hubOfficeStore）

### 1.9 handler 域（30 个 *_handler.go；设计文档 §3.1 层 5 统计为 29 域，多出的
```handler_search_export.go``` 属核心 Search/ExportAll 而非独立域）
analysis / auth / chapter / character / characterlib / characterlib_gen / chat / context /
copilot / create_chapter / feature_model / gaea / graph / image / lorebook / model_engine /
ocr_model / office / outline / platform / plot_branch / project / stats / tts / visual /
voice / voice_model / whisper / worldview + handler_search_export

### 1.10 杂项（1）
```shelf.go``` — 书架（GetNovelsDir/ListProjects，实际映射 core 门面）

**五层不一致（与设计文档 §3.1 结论一致，本次复核确认）**：导航 8 板块 ≠ 注册表 4 模块 ≠
README 7 模块 ≠ 门面 10 个 ≠ 服务域 29 个。门面是「Wails 绑定面分组」，不是板块。

---

## 2. App 装配与生命周期（证据）

### 2.1 结构体构成

- **core**（```internal/app/app.go:44-64```）：ctx / cfg / client / engineMgr / activeOCREngine /
  activeOCRModel / chatStore / charLib / distFS。注释明言「所有子服务共享的基础依赖，指针嵌入」。
- **writingState**（app.go:67-96）：内嵌 core + app 反向引用 + chapterGenMu/Cancels + pm +
  eng(prompt.Engine) + 5 个子代理（worldview/character/outline/chapter/analysis）+ skillLoader。
- **mediaState**（app.go:99-134）：内嵌 core + app 反向引用 + TTS/ASR 引擎选择 +
  voiceManager + ComfyUI 进程句柄 + 图像任务取消句柄。
- **whisperState**（app.go:137-155）：内嵌 core + app 反向引用 + whisperDataRoot +
  assistantMgr + weixinServers + writeErrors。
- **officeState**（app.go:158-196）：内嵌 core + app 反向引用 + priceCronStop/fileIndexStop/
  keepWarmStop（三个 chan + Once）+ tasks.Manager + filewatch.Watcher + keepAliveAt + preloadOnce。
- **App**（app.go:200-210）：聚合 core + 四个子状态 + ```brain *BrainStore``` + ```modules *ModuleRegistry```。
  注释明言「Wails 要求绑定单对象：所有子服务方法通过 Go 嵌入自动提升」——这是 10 门面拆分
  （S2-3）之前的遗留说明，现由 NewBindings 取代。

### 2.2 New()（app.go:229-248）

顺序：```config.Load()``` → ```ai.NewClient(cfg)``` → 构造 core → 依次构造四个子状态：
writingState（含 ```prompt.NewEngine(filepath.Join(cfg.ResourceDir,"prompts"))```）、
mediaState、whisperState（whisperDataRoot = ```config.DataRoot()/whisper_data```）、officeState。
**此阶段仅建对象，不初始化任何服务**。

### 2.3 Startup(ctx)（app.go:249-374）完整流程

1. ```a.ctx = ctx```（L250）
2. ```startDebugServer()``` — 独立 HTTP 诊断端口 127.0.0.1:18123（/healthz、/stack）（L254, L385-407）
3. ```migrateLegacyDataRoot()``` — ResourceDir/whisper_data → DataRoot 迁移（L259）
4. ```applyPendingRestore()``` — 恢复前备份（L262）
5. slog 落盘 gaea.log（L265-270）；重建 ```ai.NewClient```（L272，与 New 重复创建一次）
6. 密钥 DPAPI 迁移 + 解密注入（L275-280）
7. ```modelengine.NewManager("", deepseekKey)``` + UpdateOpencodeKey/ZenKey + LoadState(engines.json) +
   SetStatsPath + SetUsdCnyRate + EnsureModel("xai","grok-tts") / EnsureModel("cosyvoice","CosyVoice2-0.5B") +
   xAI token 恢复（L283-299）
8. ```configureClient()```（L300；定义 auth_handler.go:18）、```initImageBackend()```（L301；定义 app.go:410-436）
9. 从 cfg 恢复语音/OCR 选择（L304-313）
10. ```initVoice()```（L315；定义 voice_handler.go:83）→ ```initWeixin()```（L316；定义 whisper_state.go:14-48）
11. charLib：```characterlib.NewStore(whisperDataRoot/characterlib)``` + 剧照迁移 + EnsureBuiltins +
    EnsureAssistants（L319-330）
12. chatStore：```chat.NewStore(whisperDataRoot/chat)```（L333）
13. ```initBrain()```（L334；定义 brain_bindings.go:14-23）→ ```initModules()```（L335；定义 module_bindings.go:9）
14. ```ensureLocalTTSService("cosyvoice")```（L338）
15. 阶段 5 定时器装配（L341-352）：```startTaskScheduler()```（gaea_tasks.go:38）→
    ```startKeepWarm()```（gaea_schedule.go:74）→ ```startAutoPreload()```（gaea_schedule.go:228）→
    ```startPriceCron()```（gaea_price_sources.go:25）→ ```startFileIndexCron()```（gaea_file_index.go:43）→
    ```startFileWatch()```（gaea_tasks.go:297）
16. 后台刷新 4 个引擎模型列表：硬编码 ```[]string{"xai","herdsman","ollama","deepseek"}```（L355-373），
    完成后 ```applyASRClient()```

**依赖链**：client/engineMgr → 语音/微信/角色库/聊天存储 → 三脑/模块 → 本地服务保活 →
定时任务 → 模型列表刷新。```initBrain``` 依赖 whisperDataRoot 与 hubOfficeStore（memory_hub.go:87，
读 Hephaestus.db facts）；```initModules``` 只做纯注册不依赖其他服务。

### 2.4 Shutdown(ctx)（app.go:439-485）

顺序：voiceManager.Stop → weixinServers 逐个 Stop → closePM（writing_state.go:34-43）→
chat.CloseDatabase → characterlib.CloseDatabase → priceCronStop/fileIndexStop/keepWarmStop
（Once 幂等 close chan）→ tasks.Close → fileWatch.Close。

### 2.5 emit 统一事件发射（app.go:214-226）

`````````go
func (c *core) emit(eventName string, data map[string]interface{}) {
    httpbridge.Publish(eventName, data)   // 本地 HTTP 桥接（无 Wails 也发布）
    if c.ctx == nil { return }
    if c.ctx.Value("events") == nil { return }  // 非 Wails 上下文（测试）跳过
    runtime.EventsEmit(c.ctx, eventName, data)
}
`````````
这是全壳层唯一 Wails 事件出口（见 §5 事件清单）；Step 1 事件日志接入点即此函数。

---

## 3. 绑定面体系（10 门面 + 生成器）

### 3.1 生成器 ```scripts/gen_bindings/main.go```（526 行）

- **接收者白名单**（L28-31）：App / core / writingState / mediaState / whisperState / officeState。
- **门面顺序**（L34）：core → office → memory → cost → model → voice → chat → novel → image → charlib。
- **mapMethod 映射规则**（L46-82），优先级从高到低：
  1. ```explicitOverrides``` 显式覆盖表（L104-157，约 40 项）；
  2. ```Gaea``` 前缀 → ```mapGaea```（L85-101）：GaeaCost/GaeaPrice→cost；GaeaKnowledge/GaeaMemory/
     GaeaProfile/GaeaSemantic/GaeaWhisper→memory；GaeaModels/GaeaSetModel/GaeaModel/GaeaEngines/
     GaeaSetEngine→model；GaeaCharacter→charlib；其余→office（默认办公）；
  3. ```Herdsman```→model；```TTS/Voice/Whisper/ASR```→voice；```Chat/MainBrainChat/Brain```/RunModule→chat；
     ```Character```→charlib；```GenerateFreeImage/CancelImageGeneration/GetComfyUI/GetImageBackend/
     SetImageBackend/GetPortraitConfig/SetPortraitConfig```→image；
  4. 接收者默认：writingState→novel、mediaState→image、whisperState→voice、officeState→office；
  5. 兜底 →core。
- **collectMethods**（L229-338）：AST 解析 internal/app 全部非测试 .go（跳过 ```bindings_*``` 生成物），
  收集导出方法签名 + import 映射；去重规则「同名方法保留 App 直接声明的（shadow 嵌入），否则保留
  首次出现」（L318-335）。
- **```-names``` 模式**（L176-186）：仅输出全部导出方法名（字典序），供前端 bindingNames.ts 与 CI
  漂移检查对照。
- **产出**（L204-224）：10 个 ```bindings_<facade>.go``` + ```bindings_manifest.go``` +
  ```bindings_completeness_test.go```；未映射方法直接报错退出（L192-195，防遗漏）。

### 3.2 门面清单与职责（方法数经生成物统计）

| 门面 | 文件 | 方法数 | 职责 |
|---|---|---|---|
| CoreB | bindings_core.go | 48 | 认证/项目/书架/设置/引擎凭据（Startup/Shutdown/Login/GetConfig/SaveEngine…） |
| OfficeB | bindings_office.go | 137 | 办公引擎全量 UI（Gaea* 主体）+ Office* 桌面动作 + LocalTranslate |
| MemoryB | bindings_memory.go | 31 | 记忆中枢 8 库（GaeaMemory*/GaeaKnowledge*/GaeaProfile*/GaeaSemantic*/GaeaWhisper*） |
| CostB | bindings_cost.go | 22 | 成本库与价格源（GaeaCost*/GaeaPrice*） |
| ModelB | bindings_model.go | 34 | 模型中心 + Herdsman 底座 + 功能级模型绑定（SetFeatureModel 等） |
| VoiceB | bindings_voice.go | 45 | 语音管道 + TTS + 轻语（Whisper* 29 + Voice*/TTS* + GetEngineList） |
| ChatB | bindings_chat.go | 25 | 聊天话题/消息 + 三脑绑定（Brain*）+ 主脑（MainBrainChat/RunModule/ChatGeneral） |
| NovelB | bindings_novel.go | 67 | 小说写作域（章节/大纲/角色/世界观/图谱/批注…） |
| ImageB | bindings_image.go | 38 | 绘梦/媒体（生图/ComfyUI/画布/肖像/TTS 音频端点） |
| CharlibB | bindings_charlib.go | 15 | 全局角色库（Character* 前缀） |

合计 **462 个导出方法**。注意跨域特例：```GenerateCharacterPortrait/SetCharacterPortrait```
显式归 image（generator L123-124）；```GetSystemStats/GetTTSConfig/OpenImageSaveDir``` 等归 image
（接收者 mediaState）；```WhisperChat``` 归 voice 而非 chat（前缀 Whisper 优先于接收者）。

### 3.3 NewBindings 与 main.go

- ```bindings_manifest.go:7-23```：NewBindings(a *App) 返回 10 个门面指针（```&CoreB{a: a}...```）。
- ```main.go:58-76```：wails.Run 的 ```Bind: app.NewBindings(application)```（L75），OnStartup/OnShutdown
  挂 application.Startup/Shutdown（L70-71）；注释说明「不再绑定单一 App 对象，前端经
  gaea/lib/bridge.ts 单点路由」。
- ```main.go:29-30```：```app.New()``` + ```SetDistFS(assets)```（embed dist）。
- ```main.go:38-48```：GAEA_HTTP_PORT 可选 httpbridge 调试桥（/api/rpc + /api/stream，一次性 token）。

### 3.4 bindings_completeness_test 兜底机制

```bindings_completeness_test.go:13-31```（生成物）：反射收集 ```*App{}``` 完整方法集（含嵌入提升）与
NewBindings 全部门面方法集，排序后逐位比对，长度或顺序不一致即 Fatalf。```collectExported```
（L34-40）用 ```t.NumMethod()``` + ```m.PkgPath==""``` 判导出。**这是绑定面与 App 方法集一致性的机器兜底**
——任何新增/改名方法若未重跑生成器，测试立刻失败（设计文档 §5.2 要求 manifest 化后继续保留）。

---

## 4. 模块注册表与主脑（含已知缺陷复核）

### 4.1 ModuleRegistry（module_registry.go）

- ```Module{ID, Name, Intents []string, Handle func(input map[string]any) (map[string]any, error)}```
  （L6-11）——注意 Handle 是**闭包**而非方法指针，注册时直接捕获 App 方法。
- ```Register``` 拒重复 ID（L22-31）；```Dispatch(moduleID, intent, input)``` 先查模块再查意图（L33-44）；
  ```Has```（L46-48）。未知模块/意图均显式报错——协议与 gaea2/module-protocol.md §1 一致。

### 4.2 initModules 注册（module_bindings.go:9-58）——仅 4 个模块

| ID | 意图 | 底层方法 | 行号 |
|---|---|---|---|
| gaea | chat | a.ChatGeneral(msg) | L11-22 |
| whisper | chat | a.WhisperChat(msg, pid, false) | L23-34 |
| novel | create_chapter | a.CreateChapter(setting,"",plotReq,num,"","",0,0) | L35-44 |
| imagegen | generate | a.GenerateFreeImage(prompt,negative,size,style,model,seed,n,"") | L45-57 |

```RunModule(moduleID, intent, inputJSON)```（L71-87）是唯一 Wails 绑定入口（映射 chat 门面）。

### 4.3 主脑意图识别与派发（main_brain.go）

- ```classifyMainBrainIntent(msg)```（L10-24）：关键词规则——
  ```标书/招标/方案/报价/proposal/tender``` → ("office","create")；
  ```章节/小说/大纲/角色/章/chapter/novel``` → ("novel","create_chapter")；
  ```轻语/聊天/陪/whisper``` → ("whisper","chat")；
  ```画/图/生图/绘梦/image/generate``` → ("imagegen","generate")；
  默认 ("gaea","chat")。
- ```MainBrainChat(message)```（L36-63）流程：classify → ```a.brain.Search(message)``` 取三脑材料（截断 5 条，
  L39-44）→ 若 ```a.modules.Has(moduleID)``` 则 Dispatch（L48-59），**否则静默跳过**——result 只有
  module/intent/materials，无 output/reply，也无任何日志 → JSON 返回。

### 4.4 已知缺陷复核（设计文档 §1 缺陷 2）

**复核结论：缺陷实锤。**
1. classifyMainBrainIntent 返回 ("office","create")（main_brain.go:13-14），
   initModules 未注册 office 模块（module_bindings.go:11-57 只有 4 个）。
2. 主脑收到「写一份标书」→ Has("office")=false → main_brain.go:48 的 if 不进入 →
   **静默跳过，无日志、无 output**。
3. gaea2/module-protocol.md:18 协议表写明 office.create→ProposalCreate，但
   **ProposalCreate 在全部 .go 文件零命中**（仅 CHANGELOG.md、docs、旧 plans 提及）——
   协议与实现双脱节。
4. main_brain_test.go:5-26 只断言 classify 输出（含 "帮我把这个标书写了"→office/create），
   **不验证 MainBrainChat 的派发结果**，因此缺陷在测试下不可见。
5. module_registry_test.go:10-18 在**测试内**注册了一个 office 模块（仅用于测 Dispatch），
   从侧面证明 office 模块在真实装配中缺失。

### 4.5 三脑机制（brain_*.go）

- 命名空间（brain_store.go:6-10）：brain.main / brain.left / brain.right。
- ```BrainStore```（brain_store.go:41-47）：main/left/right 三个 ```brainAdapter```（Read/Write/Search 最小
  接口，L35-39）+ LinkStore；Search 空 brains = 三脑全搜（L66-84）；adapter 按命名空间分发（L100-110）。
- **主脑**（brain_main.go:9-12）：profile(memory.ProfileStore, Hephaestus.db) + kb(knowledge.Store)，
  Search 为朴素子串匹配（L38-54）。
- **左脑**（brain_left.go:14-16）：只读办公记忆 facts（```officeFactLeftSource```，L65-71 →
  memory_hub.go:87 hubOfficeStore 读 Hephaestus.db facts）；Write 直接返回 nil（L29-31）——
  左脑为只读业务域。
- **右脑**（brain_right.go:9-11）：轻语 hermes.db 记忆事实（whisperdb.LoadFactsFromDB/InsertFact）。
- **跨脑关联**（brain_link_store.go:10-32）：LinkStore，db 非 nil 落 Hephaestus.db ```brain_links``` 表
  （CREATE TABLE IF NOT EXISTS），nil 退化为内存模式（测试）。
- **材料注入**（brain_materials.go:18-42）：buildBrainMaterials 跨脑检索关键词，去重最多 3 条，
  格式「【跨脑记忆·右脑】实体：文本」。
- **装配**（brain_bindings.go:14-23 initBrain）：gaeadb.GetDatabase(MemoryUserDir) →
  gaekb.Global().Store() → 组装 BrainStore。绑定：BrainWrite/BrainSearch/BrainCrossRefs
  （brain_bindings.go:26-67，映射 chat 门面）。
- 注：module-protocol.md §4 说「BrainSearch 取两脑材料」，实际 BrainStore.Search 已三脑全搜
  （brain_store.go:66-84），协议文档措辞略滞后。

---

## 5. 办公引擎接入与事件发射

### 5.1 gaeaRuntime 单例（gaea_handler.go:26-34）

`````````go
var ga = &gaeaRuntime{}   // 包级全局单例，非 App 成员！
type gaeaRuntime struct {
    mu   sync.Mutex
    ctrl *control.Controller
    cfg  *gaeaConfig.Config
}
`````````
**关键事实**：办公引擎实例是包级全局单例，不挂在 App/officeState 上（与 office_handler.go:8-9 的
```jm = office.NewJobManager(nil)```、```sm = office.NewSessionModeStore()``` 同类——壳层存在多处
「包级全局可变状态」，生命周期逃逸 App）。

### 5.2 配置加载 gaeaLoadConfig（gaea_handler.go:38-74）

```gaeaConfig.Default()``` → 读用户持久化 TOML（UserConfigPath）→ 强制 ```cfg.DefaultModel = "gaea"``` →
AutoPlan 默认 "ask" → 注入 bridge provider（```Kind:"wubigrok"```，ContextWindow=256k，L55-63）→
Tools.Enabled=nil（全开 47 工具）→ ```Sandbox.Bash = "off"``` → WorkspaceRoot 跟随工作区。

### 5.3 控制器构建与重建

- ```gaeaBuildController()```（L77-117）：event.FuncSink 把办公引擎事件流转发为 ```a.emit("gaea-event",
  gaeaEventMap(e))```，TurnDone 成功后触发 ```maybeDreamAfterTurn()```（gaea_dream.go:75）→
  ```gaeaBoot.Build(ctx, Options{Model:"gaea", SessionDir: WorkspaceSessionDir(gaeaCwd()), Cwd: gaeaCwd(),
  ExtraTools: [imageGenTool, diagramTool, routineLLMTool, translateTextTool, factAddTool,
  factListTool, factClearTool]})```（L90-109）→ ```EnableInteractiveApproval()```（L115）。
- ```gaeaRebuildLocked()```（L121-132）：构建新控制器替换旧实例并 Close（调用方须持 ga.mu）。

### 5.4 生命周期方法

- ```GaeaInit()```（L135-194，幂等）：bridge.SetClient(a.client) → bridge.SetFeature（功能级模型绑定
  func_gaea_engine/model，L147-150）→ gaeaLoadConfig → TouchRecentWorkspace → 任务模板落盘 →
  .gaea/work + .gaea/exports 目录 → gaeaConfig.SetLoader（闭包读 ga.cfg，L175-180）→ 构建控制器 →
  ```resumeLastSession```（gaea_ui.go:570）→ ```a.emit("gaea-ready", {"kind":"ready"})```（L192）。
- ```GaeaReload()```（L250-280）：未初始化先 Init → 重读配置 → 替换 ga.cfg → gaeaRebuildLocked →
  ```emit("gaea-ready",{"kind":"reloaded"})```（L278）→ 返回 Tools/Skills 数。
- ```GaeaSend```（L283-294，异步）/ ```GaeaCancel```（L297-303）/ ```GaeaRunning```（L306-310）/
  ```GaeaNewSession```（L313-320）/ ```GaeaModel```（L323-330）/ ```GaeaEngines```（L333-338）/
  ```GaeaSetEngine```（L341-343）/ ```GaeaTools```（L346-356）/ ```GaeaSkills```（L359-373）/
  ```GaeaCallTool```（L376-382，前端 UI 双通道直接调内置工具）。
- **功能级模型重绑**：```SetFeatureModel```（L199-207）/```SetFeatureModelEnabled```（L211-224）在
  core 方法之上覆盖——feature=="gaea" 时重新注入 bridge 并重建控制器（applyOfficeFeatureModel
  L228-237）。这是「模型绑定 → 引擎重建」的显式耦合点。

### 5.5 事件转换（gaea_eventMap）

- ```gaea_eventMap```（L387-488）：把 internal/gaea/event.Event 转为 gaeaW WireEvent 兼容 map——
  覆盖 Text/Reasoning/Message/ToolDispatch/ToolResult/Notice/Phase/TurnDone/Usage/
  ApprovalRequest/AskRequest（含 Plan）/CompactionStarted/CompactionDone。
- ```gaeaKindName```（L491-504）：turn_started/reasoning/text/message/tool_dispatch/tool_result/
  usage/notice/phase/approval_request/ask_request/turn_done/compaction_started/compaction_done。
- ```gaeaLevelName```（L509-516）：warn/info。

### 5.6 全量事件发射点清单（66 处 a.emit，按事件名归组）

| 事件名 | 发射点（文件:行） | 语义 |
|---|---|---|
| xai-output | auth_handler.go:21；create_chapter_handler.go:278, 420 | xAI 登录输出 / 章节生成过程输出 |
| xai-login-failed | auth_handler.go:49, 55, 63 | 登录失败 |
| xai-login-success | auth_handler.go:87 | 登录成功 |
| character-fill-progress | characterlib_gen_handler.go:86 | 角色填充进度 |
| chat-stream:{runID} | chat_service.go:101, 109, 118, 126, 130, 146, 152 | 聊天流式（error/delta/reasoning/done） |
| create-chapter-stream | create_chapter_handler.go:224, 234, 244, 269, 295, 318, 337, 342, 349, 403, 408, 413, 427 | 章节流式 |
| new-characters-discovered | create_chapter_handler.go:616 | 章节生成发现新角色 |
| feature-model-changed | feature_model_handler.go:109, 128 | 功能级模型绑定变更 |
| gaea-event | gaea_handler.go:80（事件流）、285（init 错误）；gaea_dream.go:151（自动做梦 notice）；gaea_preview.go:193（OCR 预览进度 preview_progress）；whisper_state.go:82（微信会话过期 notice） | 办公引擎事件流（kind 见 5.5） |
| gaea-ready | gaea_handler.go:192（ready）、278（reloaded） | 办公引擎就绪/重载 |
| keep-warm-status | gaea_schedule.go:87 | 本地模型保活状态 |
| gaea-task | gaea_tasks.go:70（emitTaskEvent） | 任务调度器进度 |
| model-changed | model_engine_handler.go:72, 107 | 活跃引擎/模型变更 |
| model.route | model_router.go:56 | 功能域模型路由决策（feature/global/fallback/sensitive-local） |
| ocr-model-changed | ocr_model_handler.go:23, 57 | OCR 引擎/模型变更 |
| tts-stream | tts_handler.go:184, 201, 219, 234, 245 | TTS 音频流（chunk/done） |
| tts-service-status | tts_service.go:135, 142, 148 | 本地 TTS 服务启停状态 |
| voice:state / voice:transcript / voice:reply / voice:tts-audio / voice:tts-speak-text / voice:tts-speak-cancel / voice:listening / voice:thinking / voice:error | voice_handler.go:45, 49, 53, 57, 61, 65, 69, 73, 77（常量定义 L27-35） | 语音管道事件（voiceEmitter） |
| voice-model-changed | voice_model_handler.go:61, 100, 144, 176 | ASR/TTS/聊天 TTS 模型变更 |

**传播路径**：所有事件统一走 ```core.emit```（app.go:214-226）→ httpbridge.Publish（调试桥）+
Wails runtime.EventsEmit → 前端 window.runtime.EventsOn（bridge.ts 集中订阅，设计文档 §3.1）。
**Step 1 事件日志只需在这一个函数加钩子即可捕获全部**（但 chat-stream/create-chapter-stream/
tts-stream 的流式事件高频，需按 DSH 事件种类降采样/归类）。

---

## 6. 域→门面→页面映射表

页面来源：设计文档 §3.1 层 1（9 导航页 + KnowledgePage 孤儿）。门面归属依据 §3.1 的 mapMethod
规则逐方法判定（文件级取多数）。

| handler 域（文件） | 板块 | 门面 | 前端页面 | 说明 |
|---|---|---|---|---|
| analysis_handler.go | novel | NovelB | NovelPage | 风格/剧情分析 |
| auth_handler.go | core | CoreB | SettingsPage/HomePage | xAI OAuth |
| chapter_handler.go | novel | NovelB | NovelPage/ChapterPage | 章节读写 |
| character_handler.go | novel（主体） | NovelB（+ChatB 2、ImageB 2） | NovelPage | 写作角色管理（ChatCharacter 等在 ChatB） |
| characterlib_handler.go | charlib | CharlibB | CharacterLibraryPage | 全局角色库 |
| characterlib_gen_handler.go | charlib | CharlibB | CharacterLibraryPage | 角色生成/填充 |
| chat_handler.go | chat | ChatB | ChatPage | ChatGeneral |
| chat_service.go | chat | ChatB | ChatPage | 聊天话题/流式 |
| context_handler.go | novel | NovelB | NovelPage | 上下文预算 |
| copilot_handler.go | novel | NovelB | NovelPage | 写作 Copilot |
| create_chapter_handler.go | novel | NovelB | ChapterPage | 章节生成 |
| feature_model_handler.go | model/core | ModelB（+CoreB） | ModelCenterPage | 功能级模型绑定 |
| gaea_handler.go | office | OfficeB（+ModelB 5、CoreB 3） | GaeaPage | 办公引擎 |
| graph_handler.go | novel | NovelB | NovelPage | 实体图谱 |
| image_handler.go | image | ImageB | ImageGenPage | 绘梦主面（29 方法） |
| lorebook_handler.go | novel | NovelB | NovelPage | 设定集 |
| model_engine_handler.go | model/core | ModelB（+CoreB） | ModelCenterPage | 引擎管理 |
| model_router.go | model | ModelB | ModelCenterPage | 模型路由 |
| ocr_model_handler.go | core | CoreB | SettingsPage | OCR 模型选择 |
| office_handler.go | office | OfficeB | GaeaPage | Office* 桌面动作 |
| office_skill_capture.go | office | OfficeB | GaeaPage | 技能沉淀 |
| outline_handler.go | novel | NovelB（+ChatB 2） | NovelPage | 大纲 |
| platform_handler.go | core | CoreB | SettingsPage | 平台/系统 |
| plot_branch_handler.go | novel | NovelB（+ChatB 1） | NovelPage | 剧情分支 |
| project_handler.go | core | CoreB | HomePage | 项目打开（initAgents 在此触发） |
| stats_handler.go | core | CoreB | HomePage | 统计 |
| tts_handler.go | voice/image | VoiceB（+ImageB） | ChatPage/设置 | TTS 端点 |
| tts_service.go | voice | VoiceB | ChatPage | 本地 TTS 保活 |
| visual_handler.go | image | ImageB | ImageGenPage | 画布/视觉 |
| voice_handler.go | voice | VoiceB（+ImageB 9） | ChatPage 语音 | 语音管道（voiceEmitter） |
| voice_model_handler.go | voice/image | VoiceB（+ImageB） | ModelCenterPage | ASR/TTS 模型切换 |
| whisper_handler.go | voice | VoiceB（+ChatB 1） | ChatPage（轻语） | 轻语主面（29 方法） |
| whisper_state.go | voice | VoiceB | ChatPage | 轻语状态/微信 |
| whisper_taskplan.go | voice | VoiceB | ChatPage | 任务计划 |
| worldview_handler.go | novel | NovelB（+ChatB 1） | NovelPage | 世界观 |
| handler_search_export.go | core | CoreB | HomePage/ExportPage | Search/ExportAll |
| memory_hub.go | memory | MemoryB | MemoryHubPage | 记忆中枢 8 库 |
| herdsman_*.go（6） | model | ModelB | ModelCenterPage | Herdsman 底座 |
| gaea_*.go（44） | office/memory/cost/model | OfficeB/MemoryB/CostB/ModelB | GaeaPage + MemoryHubPage + ModelCenterPage | 办公引擎全家 |
| shelf.go | core | CoreB | HomePage | 书架 |

**关键观察**：
- 门面 ≠ 板块：MemoryB/CostB 服务记忆中枢与办公两个页面；VoiceB 承载轻语 + 语音 + 部分聊天；
- 「聊天」板块页面 ChatPage 实际消费 ChatB + VoiceB（轻语）+ ImageB（TTS 音频）跨 3 个门面；
- KnowledgePage 是孤儿页面（实现存在、未挂导航），其能力已被 MemoryHubPage 的 knowledge 库取代
  （设计文档 §3.1 层 1 确认）。

---

## 7. 与 3.0 目标相关的关键发现（装配点 / 事件点 / switch 清单）

### 7.1 装配点清单（Step 2 Manifest 化要动的点）

1. **main.go:75** — ```Bind: app.NewBindings(application)```：门面清单唯一装配入口（manifest 驱动点）。
2. **bindings_manifest.go:11-22** — NewBindings 硬编码 10 门面顺序（manifest 化后由清单生成）。
3. **app.go:229-248 New()** — 4 个子状态硬构造（manifest 化后仍可保留为内部实现，但板块初始化
   应改为 board.Init(a *App) 阶段）。
4. **app.go:249-374 Startup()** — 33 步顺序装配：initBrain(L334)/initModules(L335) 与
   charLib/chatStore 硬顺序；设计文档 §5.4 要求「装配顺序写入 Startup 注释，可审计」。
5. **module_bindings.go:9-58 initModules** — 4 个模块闭包注册（manifest 化后由 manifest 填充）。
6. **main_brain.go:36-63 MainBrainChat** — 关键词意图表（manifest 化后 intents 来自 manifest）。
7. **gaea_handler.go:26** — ```var ga = &gaeaRuntime{}``` 包级单例（应纳入 App 生命周期/合成根）。
8. **office_handler.go:8-9** — ```jm```/```sm``` 包级单例（同上）。
9. **frontend MainLayout/ModuleLauncher** — 静态导航 6 处编译期耦合（设计文档 §1 缺陷 1，属前端）。
10. **gen_bindings/main.go:28-31, 34** — receiverTypes/facadeOrder 白名单（manifest 化后从 manifest
    派生门面清单与完整性测试）。

### 7.2 事件发射点清单（Step 1 事件日志要接的点）

- **核心钩子唯一**：```core.emit```（app.go:214-226）——全部 66 处发射的必经之路（§5.6）。Step 1
  可在 emit 内按 eventName 白名单写事件日志（对齐 DSH「模型可见输入必须先入日志」不变量）。
- 需特别注意的高频流式事件：```chat-stream:{runID}```（chat_service.go）、```create-chapter-stream```
  （create_chapter_handler.go）、```tts-stream```（tts_handler.go）——日志应合并为
  assistant_delta 语义或仅记首尾/usage。
- 办公引擎会话本身的 Step 1 主战场在 ```internal/gaea/agent/session```（Save 整文件重写 JSONL，
  设计文档 §1 缺陷 3），壳层侧只需让 ```gaea-event```（gaea_handler.go:80）与 ```gaea-ready``` 事件参与日志。
- 会话派生态（标题/统计/成本）的事件来源：```model.route```（model_router.go:56）+ ```gaea-event``` 的
  usage 子事件（gaea_eventMap L426-440 已带 usage 明细）。

### 7.3 provider 硬编码 switch 清单（Step 3 Provider Seam 目标）

| 位置 | 形态 | 目标 Seam |
|---|---|---|
| app.go:410-436 initImageBackend | ```switch a.cfg.ImageBackend { comfyui / herdsman / ollama / default xai }``` | Image |
| writing_state.go:59-83 restoreImageBackend | ```switch cfg.ImageBackend```（comfyui 内嵌 herdsman 兜底、ollama、xai 注释） | Image |
| image_handler.go:572-600, 683-700 | ```switch backend```（comfyui/xai/herdsman/ollama 双处） | Image |
| characterlib_gen_handler.go:405-411 | ```switch backend```（comfyui/herdsman/ollama） | Image |
| app.go:355 | 引擎刷新硬编码 ```[]string{"xai","herdsman","ollama","deepseek"}``` | LLM 注册表 |
| model_router.go:80, 90 | ```routeSensitiveLocal``` 硬编码 ```GetEngine("herdsman")``` | LLM |
| tts_service.go:71-72 | ```switch engineID { case "cosyvoice": }``` | Voice(TTS) |
| tts_handler.go:225-227 | ```switch engine { case "edge": case "sapi": }``` | Voice(TTS) |
| feature_model_handler.go:181 | ```case "herdsman","ollama","cosyvoice"```（功能模型能力表） | LLM/Voice |
| gaea_cost_import_vision.go:758 | ```switch kind```（视觉导入引擎选择） | OCR/Image |
| gaea_herdsman_health.go:47 | ```switch kind```（健康检查类别） | 诊断 |
| herdsman_operations.go:96 | ```switch strings.ToLower(kind)```（仅显示名，非路由） | — |

另：办公引擎侧 bridge provider 已实现 ```Kind:"wubigrok"``` 的 seam 雏形（gaea_handler.go:55-63 +
bridge.SetClient/SetFeature L143-150），是 Step 3 LLM seam 的现成底图（设计文档引用
docs/gaea2/office-model-tool-call-chains.md）。

---

## 8. 缺陷与风险

### 8.1 活跃缺陷（可直接复现）

1. **主脑 office 意图静默跳过**（§4.4）：classify 返回 ("office","create") 但模块未注册，
   MainBrainChat 无日志、无 output、无错误。协议文档（module-protocol.md:18）指向不存在的
   ProposalCreate。**设计文档 Step 0 已立项修复**，D8 三个方案待决策。
2. **main_brain_test.go 盲区**：只测 classify，不测 Dispatch 结果，缺陷被测试掩盖（main_brain_test.go:5-26）。
3. **知识库孤儿页面**：KnowledgePage 无导航挂载（前端层，设计文档 §3.1 层 1），能力被 memoryhub 取代。

### 8.2 结构性风险

4. **包级全局可变单例**：```ga```（gaea_handler.go:26）、```jm```/```sm```（office_handler.go:8-9）、
   ```gaeaDreamState```（gaea_dream.go:34-38）——生命周期逃逸 App，测试隔离困难、多实例互踩。
5. **Startup 33 步顺序敏感**：硬编码顺序无阶段抽象（app.go:249-374），除 4 个模型刷新
   goroutine 有 recover（L356-361）外，任一步 panic 无恢复。
6. **client 双创建**：New()（app.go:231）与 Startup（app.go:272）各 ```ai.NewClient``` 一次，属冗余。
7. **生成器与手工双轨**：bindings 由生成器维护，但 bridge.ts 的 AppBindings 接口是手工同步
   （bridge.ts:89-94 注释自认「Keep in sync by hand」）——前端类型漂移只能靠编译期断言兜底。
8. **门面数量与板块错位**：OfficeB 137 方法 + MemoryB/CostB 全 Gaea* 前缀，办公引擎一个板块
   占约 190 个绑定方法，门面粒度与业务板块不一致（设计文档 §3.1 关键事实）。
9. **AppVersion 三处版本源**：app_info.go:11（2.20.1）、wails.json、versioninfo.rc 需同步
   （设计文档 §8）。
10. **测试面 91/203（45%）**：```_test.go``` 占比高是好事，但绑定门面 462 方法中只有完备性测试
    覆盖「存在性」，无「行为」覆盖——门面是纯委托，行为风险主要在 handler 层。

---

## 9. 改造建议

### 9.1 对齐 Step 0（修债，0.5 天）
- office 模块按 D8 决策补注册：建议 (b)「office.create 路由到办公 agent 入口」——GaeaSend 已是
  现成入口（gaea_handler.go:283），比实现不存在的 ProposalCreate 成本低；同时把
  module_bindings.go:11-57 的闭包改造成注册表驱动。
- main_brain_test.go 增加 MainBrainChat 全链路断言（输入标书 → 返回 office 模块 output）。
- 版本三处同步脚本化（app_info.go:11 / wails.json / versioninfo.rc）。

### 9.2 对齐 Step 1（事件日志）
- 在 ```core.emit```（app.go:214-226）加日志钩子最省：单点拦截全部 66 处发射；对
  chat-stream:/create-chapter-stream:/tts-stream 等高频事件做事件种类归并（assistant_delta/usage）。
- 办公会话侧主战场在 internal/gaea/agent/session（整文件重写），壳层不重复造轮子，只负责
  gaea-event/gaea-ready 的事件化与投影消费。
- 派生 API 优先从 model.route + usage 子事件实现（标题/统计/成本）。

### 9.3 对齐 Step 2（板块 Manifest）
- 新增 internal/app/board/ 包：Board{ID,Name,Icon,Page,Bindings,Intents,Tools,Init}，builtins.go
  注册 canonical 9 板块（chat/novel/imagegen/gaea/memoryhub/modelcenter/characterlib/settings +
  weixin 服务板块待决）。
- ```NewBindings```（bindings_manifest.go:7-23）与 ```initModules```（module_bindings.go:9-58）改为由
  manifest 驱动；bindings_completeness_test.go 反射兜底保留。
- Startup 装配顺序按 DSH「layers apply in order」写入注释（app.go:249 起），并把 ga/jm/sm
  三个包级单例并入 App 生命周期。
- 前端 MainLayout/ModuleLauncher 数据驱动 + GetBoardManifests 绑定（挂 CoreB）。

### 9.4 对齐 Step 3（Provider Seam）
- 以 bridge provider（gaea_handler.go:55-63）为模板，建立 providers.Register(kind, Provider) 注册表：
  Image（xai/herdsman/ollama/comfyui，合并 app.go:410-436 / writing_state.go:59-83 /
  image_handler.go:572-600 的三处 switch）、LLM（routeModel 降级链 model_router.go:14-49 保留，
  引擎名改注册表键）、Voice/TTS（tts_service.go:71 / tts_handler.go:225）、OCR（docmd 侧）。
- 敏感域本地化（model_router.go:80,90 硬编码 herdsman）改为配置键 + 注册表查询。
- 验收：切换生图/OCR/TTS 后端仅改配置，代码零改动（设计文档 §5.3 验收标准）。

### 9.5 长期
- 将 mapMethod（gen_bindings/main.go:46-82）的映射规则文档化进 manifest schema，消除
  explicitOverrides 手写表（L104-157）与规则（前缀）双轨漂移风险。
- 孤儿 KnowledgePage 与 weixin 板块归属按设计文档 D7 决策收敛。

---

## 附：证据索引（核心文件 → 行号速查）

- App 结构/生命周期：internal/app/app.go:44-64, 67-96, 99-134, 137-155, 158-196, 200-210, 214-226, 229-248, 249-374, 439-485
- emit 唯一出口：internal/app/app.go:214-226
- 生成器映射：scripts/gen_bindings/main.go:28-31, 34, 46-82, 85-101, 104-157, 229-338, 516-552
- 门面清单：internal/app/bindings_manifest.go:7-23
- 完备性测试：internal/app/bindings_completeness_test.go:13-31
- 模块注册表：internal/app/module_registry.go:6-48；module_bindings.go:9-58, 71-87
- 主脑：internal/app/main_brain.go:10-24, 36-63
- 三脑：internal/app/brain_bindings.go:14-23；brain_store.go:6-10, 41-47, 66-84；brain_main.go:9-12；brain_left.go:14-16, 29-31；brain_right.go:9-11；brain_link_store.go:10-32
- 办公引擎：internal/app/gaea_handler.go:26-34, 38-74, 77-117, 121-132, 135-194, 199-237, 250-280, 283-382, 387-516
- 事件发射 66 处：见 §5.6 表（文件:行齐全）
- provider switch：app.go:410-436, 355；writing_state.go:59-83；image_handler.go:572-600, 683-700；characterlib_gen_handler.go:405-411；tts_service.go:71-72；tts_handler.go:225-227；feature_model_handler.go:181；model_router.go:80, 90
- 版本：internal/app/app_info.go:11
- 3.0 目标文档：docs/2026-08-15-gaea3-architecture-design.md（§1 缺陷 2/3、§3.1 五层清单、§5.1-5.3 机制、§10 D8）
- 模块协议：docs/gaea2/module-protocol.md（office→ProposalCreate，L18）
