# 附录 A · 机械扫描原始数据（2026-10-02）

> 本附录由 `docs/code-audit-2026-10-02/machine/scan.py` + `scan2.py` 生成，全部数据可由脚本复跑（工作目录 C:/AI/wubigrok）。
> 口径：只统计 `internal/` 与 `frontend/src/` 下的源码（排除 node_modules / clones / backups / dist / build / releases / whisper_data / .tmp / novels / wailsjs）。

## A0. 规模与门禁基线

| 项 | 文件数 | 非测试 LOC | 测试文件 | 测试 LOC |
|---|---|---|---|---|
| Go（internal/） | 1723 | 203872 | 849 | 153259 |
| TS/TSX（frontend/src） | 1127 | 148135 | 414 | 60033 |
| **合计** | **2850** | **352007** | **1263** | **213292** |

- `go vet ./...`：exit 0，0 issues（2026-10-02 本机实测）
- `golangci-lint run ./...`（v2.14.0，errcheck/govet/ineffassign/staticcheck/unused/misspell 全开）：`0 issues.`
- 结论：**静态检查层已经清零，剩下的全是结构层问题**（下面每一项都是 lint 抓不到的）。

## A1. 上帝文件

### A1.1 Go 非测试文件 LOC TOP 40（>600 行即为高发区）

| # | LOC | 文件 |
|---|---|---|
| 1 | 1430 | `internal/ai/client.go` |
| 2 | 1268 | `internal/voice/voice_manager.go` |
| 3 | 1265 | `internal/gaea/costimport/costimport.go` |
| 4 | 1245 | `internal/gaea/tasks/tasks.go` |
| 5 | 1189 | `internal/app/gaea_ui_extra.go` |
| 6 | 1135 | `internal/channels/weixin/clawbot.go` |
| 7 | 1111 | `internal/project/project.go` |
| 8 | 1096 | `internal/app/image_handler.go` |
| 9 | 1069 | `internal/gaea/browser/manager.go` |
| 10 | 1057 | `internal/gaea/control/controller.go` |
| 11 | 1052 | `internal/ai/image_comfyui.go` |
| 12 | 1042 | `internal/gaea/contextview/fold.go` |
| 13 | 1037 | `internal/novelcontext/novelcontext.go` |
| 14 | 1034 | `internal/whisper/orchestrator.go` |
| 15 | 1020 | `internal/app/whisper_handler.go` |
| 16 | 1019 | `internal/app/create_chapter_handler.go` |
| 17 | 1002 | `internal/gaea/agent/task.go` |
| 18 | 976 | `internal/gaea/agent/agent.go` |
| 19 | 956 | `internal/gaea/config/config.go` |
| 20 | 925 | `internal/schedule/mpp.go` |
| 21 | 922 | `internal/gaea/tool/builtin/websearch.go` |
| 22 | 909 | `internal/gaea/boot/boot.go` |
| 23 | 894 | `internal/app/characterlib_gen_handler.go` |
| 24 | 888 | `internal/app/app.go` |
| 25 | 885 | `internal/app/sin_handler.go` |
| 26 | 883 | `internal/gaea/cost/cost.go` |
| 27 | 881 | `internal/gaea/costref/graph.go` |
| 28 | 871 | `internal/app/consistency_deep_handler.go` |
| 29 | 849 | `internal/app/novel_plan_handler.go` |
| 30 | 842 | `internal/app/gaea_handler.go` |
| 31 | 828 | `internal/office/docxedit/docxedit.go` |
| 32 | 825 | `internal/character/character.go` |
| 33 | 812 | `internal/app/memory_hub.go` |
| 34 | 801 | `internal/app/gaea_ui.go` |
| 35 | 800 | `internal/gaea/plugin/plugin.go` |
| 36 | 793 | `internal/app/gaea_dag.go` |
| 37 | 784 | `internal/gaea/agent/compact.go` |
| 38 | 781 | `internal/gaea/agent/subagent_store.go` |
| 39 | 778 | `internal/office/xlsxedit/xlsxedit.go` |
| 40 | 772 | `internal/app/gaea_cost_import_vision.go` |

### A1.2 前端非测试文件 LOC TOP 40

| # | LOC | 文件 |
|---|---|---|
| 1 | 1680 | `frontend/src/gaea/locales/zh-TW.ts` |
| 2 | 1678 | `frontend/src/gaea/locales/en.ts` |
| 3 | 1676 | `frontend/src/gaea/locales/zh.ts` |
| 4 | 1501 | `frontend/src/pages/CreatePage.tsx` |
| 5 | 1188 | `frontend/src/pages/ChapterPage.tsx` |
| 6 | 1133 | `frontend/src/gaea/components/CostLibraryView.tsx` |
| 7 | 1098 | `frontend/src/genui/renderNode.tsx` |
| 8 | 1066 | `frontend/src/gaea/components/Sidebar.tsx` |
| 9 | 1041 | `frontend/src/gaea/App.tsx` |
| 10 | 1039 | `frontend/src/components/ModuleLauncher.tsx` |
| 11 | 1035 | `frontend/src/pages/WeixinPage.tsx` |
| 12 | 1030 | `frontend/src/gaea/lib/mock/office/methods_office.ts` |
| 13 | 1024 | `frontend/src/gaea/components/Transcript.tsx` |
| 14 | 1021 | `frontend/src/gaea/components/DocxPreview.tsx` |
| 15 | 1007 | `frontend/src/pages/CharacterPage.tsx` |
| 16 | 884 | `frontend/src/components/characterlib/CharacterLibEditor.tsx` |
| 17 | 875 | `frontend/src/gaea/lib/store/controller.ts` |
| 18 | 873 | `frontend/src/gaea/components/context/inspector.tsx` |
| 19 | 863 | `frontend/src/gaea/components/memoryhub/CostProjectsView.tsx` |
| 20 | 830 | `frontend/src/gaea/components/MemoryPanel.tsx` |
| 21 | 827 | `frontend/src/genui/guard.ts` |
| 22 | 822 | `frontend/src/components/novel/ChapterEditor.tsx` |
| 23 | 812 | `frontend/src/layouts/MainLayout.tsx` |
| 24 | 809 | `frontend/src/gaea/components/XlsxPreview.tsx` |
| 25 | 775 | `frontend/src/pages/SchedulePage.tsx` |
| 26 | 775 | `frontend/src/gaea/lib/spaceBindings.ts` |
| 27 | 772 | `frontend/src/gaea/components/ContextView.tsx` |
| 28 | 748 | `frontend/src/gaea/components/KnowledgePanel.tsx` |
| 29 | 747 | `frontend/src/gaea/lib/bindingNames.ts` |
| 30 | 746 | `frontend/src/schedule/store.ts` |
| 31 | 739 | `frontend/src/components/novel/ChapterPlanCard.tsx` |
| 32 | 735 | `frontend/src/gaea/lib/mock/cost.ts` |
| 33 | 700 | `frontend/src/gaea/lib/bridge/novel.ts` |
| 34 | 681 | `frontend/src/gaea/lib/mock/novel.ts` |
| 35 | 675 | `frontend/src/gaea/components/FilePreview.tsx` |
| 36 | 664 | `frontend/src/components/novel/BookSearchModal.tsx` |
| 37 | 659 | `frontend/src/gaea/components/DagPanel.tsx` |
| 38 | 656 | `frontend/src/pages/CostLibraryPage.tsx` |
| 39 | 652 | `frontend/src/pages/MemoryHubPage.tsx` |
| 40 | 646 | `frontend/src/components/imagegen/ControlPanel.tsx` |

## A2. 上帝函数（Go，AST 精确统计，单函数 ≥100 行，共 110 个；其中 ≥200 行 19 个）

> 口径：`go/parser` + `go/ast` 精确函数跨度（不是花括号计数），工具 `machine/godfunc`（源码 `.tmp/godfunc/main.go`），只算 `internal/` 非测试文件。

| # | 行数 | 位置 | 方法/函数 |
|---|---|---|---|
| 1 | 551 | `internal/gaea/boot/boot.go:108` | `Build` |
| 2 | 449 | `internal/config/config.go:27` | `Load` |
| 3 | 449 | `internal/gaea/agent/agent_run.go:23` | `AgentRunner.runDirect` |
| 4 | 434 | `internal/schedule/ops.go:106` | `applyOne` |
| 5 | 357 | `internal/whisper/orchestrator.go:129` | `Orchestrator.PreLLMTurn` |
| 6 | 338 | `internal/gaea/agent/execute_one.go:21` | `AgentRunner.executeOne` |
| 7 | 289 | `internal/gaea/tool/builtin/bash.go:86` | `bash.Execute` |
| 8 | 279 | `internal/app/create_chapter_handler.go:488` | `writingState.streamCreateChapter` |
| 9 | 268 | `internal/gaea/control/controller_submit.go:39` | `Controller.dispatchSlash` |
| 10 | 265 | `internal/app/sin_handler.go:268` | `App.runSinStream` |
| 11 | 256 | `internal/schedule/cpm.go:148` | `computeCpmFull` |
| 12 | 253 | `internal/office/xlsxedit/xlsxedit.go:330` | `applyOne` |
| 13 | 239 | `internal/gaea/skill/builtins.go:18` | `builtinSkills` |
| 14 | 239 | `internal/app/app.go:329` | `App.Startup` |
| 15 | 224 | `internal/app/shelf.go:122` | `App.SaveConfig` |
| 16 | 216 | `internal/schedule/xlsx.go:427` | `ImportXlsx` |
| 17 | 213 | `internal/gaea/config/render.go:13` | `RenderTOML` |
| 18 | 207 | `internal/app/create_chapter_handler.go:44` | `writingState.createChapter` |
| 19 | 206 | `internal/app/image_handler.go:442` | `mediaState.GenerateMedia` |
| 20 | 197 | `internal/ai/image_comfyui.go:204` | `ComfyUIBackend.GenerateImage` |
| 21 | 191 | `internal/app/consistency_deep_handler.go:670` | `deepCompareStateLine` |
| 22 | 181 | `internal/gaea/agent/agent_stream.go:29` | `AgentRunner.stream` |
| 23 | 179 | `internal/app/sin_insight.go:185` | `foldSinContext` |
| 24 | 176 | `internal/gaea/agent/compact_summary.go:14` | `BuildCompactSummary` |
| 25 | 173 | `internal/gaea/tool/builtin/readfile.go:62` | `readFile.Execute` |
| 26 | 163 | `internal/app/gaea_preview.go:125` | `App.GaeaPreview` |
| 27 | 160 | `internal/app/memory_hub.go:293` | `App.GaeaMemoryGraph` |
| 28 | 159 | `internal/app/story_spine_handler.go:69` | `writingState.NovelStoryHealth` |
| 29 | 158 | `internal/ai/client.go:789` | `Client.parseStreamEvents` |
| 30 | 156 | `internal/app/image_handler.go:261` | `mediaState.generateImageInternal` |
| 31 | 153 | `internal/booksource/rule.go:183` | `Rule.Validate` |
| 32 | 150 | `internal/app/whisper_state.go:65` | `whisperState.startAssistantWx` |
| 33 | 150 | `internal/outline/outline.go:129` | `Agent.Continue` |
| 34 | 149 | `internal/analysis/story_memory.go:43` | `ExtractStoryMemories` |
| 35 | 144 | `internal/app/whisper_handler.go:151` | `whisperState.whisperChat` |
| 36 | 141 | `internal/gaea/costinquiry/scan.go:41` | `Store.ScanAnomalies` |
| 37 | 140 | `internal/office/docxedit/docxedit.go:619` | `rebuildParagraph` |
| 38 | 140 | `internal/ai/image_comfyui.go:725` | `ComfyUIBackend.pollComfyProgress` |
| 39 | 137 | `internal/whisper/orchestrator.go:866` | `Orchestrator.buildTierBBlock` |
| 40 | 137 | `internal/gaea/agent/batch_executor.go:15` | `AgentRunner.executeBatch` |
| 41 | 136 | `internal/gaea/cost/repair.go:35` | `legacyCategoryTarget` |
| 42 | 136 | `internal/characterstate/characterstate.go:74` | `ApplyChapterDiff` |
| 43 | 135 | `internal/app/novel_bookcover.go:37` | `App.GaeaGenerateBookCover` |
| 44 | 135 | `internal/auth/oauth.go:59` | `DoLogin` |
| 45 | 134 | `internal/whisper/memory_graph.go:197` | `KnowledgeGraph.QuerySubgraph` |
| 46 | 134 | `internal/gaea/tool/builtin/schedule_tools.go:213` | `scheduleApply.Execute` |
| 47 | 134 | `internal/schedule/project.go:75` | `Validate` |
| 48 | 132 | `internal/gaea/agent/task.go:578` | `TaskTool.runSubSession` |
| 49 | 131 | `internal/gaea/boot/sysprompt.go:43` | `buildSystemPrompt` |
| 50 | 131 | `internal/app/sin_handler.go:554` | `App.SinIllustrate` |
| 51 | 129 | `internal/app/novel_rewrite_handler.go:155` | `writingState.novelChapterRewritePartial` |
| 52 | 128 | `internal/modelengine/engine_models.go:39` | `Manager.fetchModels` |
| 53 | 128 | `internal/app/herdsman_digitallife.go:120` | `loadHerdsmanDigitalLife` |
| 54 | 126 | `internal/ai/client.go:463` | `Client.chatOnce` |
| 55 | 126 | `internal/channels/weixin/clawbot.go:529` | `Server.handle` |
| 56 | 125 | `internal/docmd/office.go:338` | `xlsxToMarkdown` |
| 57 | 125 | `internal/app/gaea_resync.go:147` | `foldResyncItems` |
| 58 | 125 | `internal/gaea/control/controller_approval.go:233` | `Controller.promptApproval` |
| 59 | 124 | `internal/office/pptxedit/pptxedit.go:515` | `rebuildParagraph` |
| 60 | 124 | `internal/gaea/agent/render/sink.go:57` | `Sink.Emit` |

### A2.1 单文件最大函数 TOP 25（文件级上帝化程度）

| # | 最大函数行数 | 文件 | 该文件函数数 | 位置 |
|---|---|---|---|---|
| 1 | 551 | `internal/gaea/boot/boot.go` | 15 | `internal/gaea/boot/boot.go:108` |
| 2 | 449 | `internal/gaea/agent/agent_run.go` | 6 | `internal/gaea/agent/agent_run.go:23` |
| 3 | 449 | `internal/config/config.go` | 5 | `internal/config/config.go:27` |
| 4 | 434 | `internal/schedule/ops.go` | 11 | `internal/schedule/ops.go:106` |
| 5 | 357 | `internal/whisper/orchestrator.go` | 18 | `internal/whisper/orchestrator.go:129` |
| 6 | 338 | `internal/gaea/agent/execute_one.go` | 12 | `internal/gaea/agent/execute_one.go:21` |
| 7 | 289 | `internal/gaea/tool/builtin/bash.go` | 18 | `internal/gaea/tool/builtin/bash.go:86` |
| 8 | 279 | `internal/app/create_chapter_handler.go` | 20 | `internal/app/create_chapter_handler.go:488` |
| 9 | 268 | `internal/gaea/control/controller_submit.go` | 2 | `internal/gaea/control/controller_submit.go:39` |
| 10 | 265 | `internal/app/sin_handler.go` | 23 | `internal/app/sin_handler.go:268` |
| 11 | 256 | `internal/schedule/cpm.go` | 14 | `internal/schedule/cpm.go:148` |
| 12 | 253 | `internal/office/xlsxedit/xlsxedit.go` | 15 | `internal/office/xlsxedit/xlsxedit.go:330` |
| 13 | 239 | `internal/app/app.go` | 12 | `internal/app/app.go:329` |
| 14 | 239 | `internal/gaea/skill/builtins.go` | 1 | `internal/gaea/skill/builtins.go:18` |
| 15 | 224 | `internal/app/shelf.go` | 6 | `internal/app/shelf.go:122` |
| 16 | 216 | `internal/schedule/xlsx.go` | 17 | `internal/schedule/xlsx.go:427` |
| 17 | 213 | `internal/gaea/config/render.go` | 7 | `internal/gaea/config/render.go:13` |
| 18 | 206 | `internal/app/image_handler.go` | 35 | `internal/app/image_handler.go:442` |
| 19 | 197 | `internal/ai/image_comfyui.go` | 24 | `internal/ai/image_comfyui.go:204` |
| 20 | 191 | `internal/app/consistency_deep_handler.go` | 25 | `internal/app/consistency_deep_handler.go:670` |
| 21 | 181 | `internal/gaea/agent/agent_stream.go` | 6 | `internal/gaea/agent/agent_stream.go:29` |
| 22 | 179 | `internal/app/sin_insight.go` | 15 | `internal/app/sin_insight.go:185` |
| 23 | 176 | `internal/gaea/agent/compact_summary.go` | 3 | `internal/gaea/agent/compact_summary.go:14` |
| 24 | 173 | `internal/gaea/tool/builtin/readfile.go` | 16 | `internal/gaea/tool/builtin/readfile.go:62` |
| 25 | 163 | `internal/app/gaea_preview.go` | 8 | `internal/app/gaea_preview.go:125` |

## A3. 坏味道标记词频（非测试源码）

| 模式 | 命中数 |
|---|---|
| TODO/FIXME/HACK/XXX | 4 |
| 临时/兜底/兼容/废弃/占位(中文注释) | 1244 |
| nolint | 8 |
| panic( | 41 |
| recover() | 78 |
| time.Sleep | 13 |
| _, _ = / 丢错 | 489 |
| interface{}/any 泛滥(go) | 1462 |
| goroutine 起协程 | 96 |

### 样例：TODO/FIXME/HACK/XXX

- `internal/gaea/i18n/messages_zh.go:161` — `gaea run "把 main.go 里的 TODO 实现掉"`
- `internal/gaea/outputstyle/outputstyle.go:43` — `Description: "Collaborate and leave TODO(human) stubs for the user to complete",`
- `internal/gaea/outputstyle/outputstyle.go:49` — `"`TODO(human)` stub with a one-line description for the user to implement themselves.",`
- `internal/style/profile.go:102` — `context.TODO(),`

### 样例：临时/兜底/兼容/废弃/占位(中文注释)

- `internal/ai/client.go:142` — `// 响应头超时兜底「连接 + 首字节等待」：代理或远端黑洞时避免无限挂起`
- `internal/ai/client.go:807` — `var streamUsage *ChatUsage // 流结束块携带的用量（OpenAI 兼容 API 在最后一块带 usage）`
- `internal/ai/client.go:1067` — `// prompt_tokens_details.cached_tokens。命中取 CacheHitTokens()（两者兼容）；`
- `internal/ai/client.go:1196` — `// prepareStreamRequest 装配一轮流式请求：模型名兜底、default 采样、modelhub 让位、`
- `internal/ai/client.go:1221` — `// 维持 0.7 兜底。`
- `internal/ai/client.go:1355` — `// 走注册表 openai 兼容后端（kind = ImageBackendKindOpenAI），保留 xAI 特有参数清理；`
- `internal/ai/client.go:1369` — `// xAI API 只接受 model/prompt/n/response_format，清空不兼容字段`
- `internal/ai/image_backend.go:17` — `// ImageBackendKindOpenAI OpenAI 兼容图片后端（/v1/images/generations）：`

### 样例：nolint

- `internal/app/image_comfyui_proc.go:98` — `if strings.Contains(pythonExe, "python\\python.exe") \|\| strings.Contains(pythonExe, "python_embeded") \|\| strings.Contain`
- `internal/app/image_comfyui_proc.go:186` — `filepath.Join(comfyUIPath, "python_embeded", "python.exe"),       //nolint:misspell // python_embeded 系上游真实目录名`
- `internal/app/novel_booksource_handler.go:556` — `defer pm.Close() //nolint:errcheck // 元信息时间戳尽力而为`
- `internal/booksource/fetch.go:101` — `httpReq.Header.Set("User-Agent", userAgents[rand.IntN(len(userAgents))]) //nolint:gosec // UA 池随机，非安全用途`
- `internal/booksource/fetch.go:168` — `opt.Rand = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1)) //nolint:gosec`
- `internal/gaea/tasks/tasks.go:226` — `lastSeq   int   //nolint:unused // 预留：将来可按 seq 增量拉取（当前整尾回放）`
- `internal/gaea/tool/builtin/webfetch.go:237` — `return ne.Timeout() \|\| ne.Temporary() //nolint:staticcheck // SA1019: Temporary 已弃用但旧错误路径仍依赖，替换候选单独评估`
- `internal/whisper/paced_stream.go:306` — `return //nolint:staticcheck // SA4004: 泵循环各分支显式收束，保留 for-select 骨架`

### 样例：panic(

- `internal/ai/image_backend.go:44` — `panic("ai: image backend kind must not be empty")`
- `internal/ai/image_backend.go:47` — `panic("ai: duplicate image backend kind " + kind)`
- `internal/app/sin_tools.go:63` — `panic("app: sin tool kind/factory must not be empty")`
- `internal/app/sin_tools.go:66` — `panic("app: duplicate sin tool kind " + kind)`
- `internal/asr/provider.go:43` — `panic("asr: provider kind must not be empty")`
- `internal/asr/provider.go:46` — `panic("asr: duplicate provider kind " + kind)`
- `internal/channels/weixin/media_crypt.go:33` — `panic("crypto/cipher: input not full blocks (ECB)")`
- `internal/channels/weixin/media_download.go:41` — `panic(err)`

### 样例：recover()

- `internal/ai/client.go:688` — `if r := recover(); r != nil {`
- `internal/ai/copilot.go:59` — `if r := recover(); r != nil {`
- `internal/ai/copilot.go:332` — `if r := recover(); r != nil {`
- `internal/app/app.go:477` — `if r := recover(); r != nil {`
- `internal/app/app.go:551` — `if r := recover(); r != nil {`
- `internal/app/app.go:591` — `if r := recover(); r != nil {`
- `internal/app/auth_handler.go:56` — `if r := recover(); r != nil {`
- `internal/app/auth_handler.go:86` — `if r := recover(); r != nil {`

### 样例：time.Sleep

- `internal/app/gaea_schedule.go:243` — `time.Sleep(preloadDelay)`
- `internal/app/image_comfyui_proc.go:237` — `time.Sleep(2 * time.Second)`
- `internal/app/image_comfyui_proc.go:243` — `time.Sleep(3 * time.Second)`
- `internal/app/image_comfyui_proc.go:274` — `time.Sleep(3 * time.Second)`
- `internal/app/model_engine_handler.go:42` — `time.Sleep(2 * time.Second)`
- `internal/app/tts_service.go:103` — `time.Sleep(delay)`
- `internal/app/tts_service.go:150` — `time.Sleep(3 * time.Second)`
- `internal/docmd/ocr.go:87` — `time.Sleep(time.Second)`

### 样例：_, _ = / 丢错

- `internal/ai/image_comfyui.go:89` — `body, _ := io.ReadAll(resp.Body)`
- `internal/ai/image_comfyui.go:915` — `if statusStr, _ := status["status_str"].(string); statusStr == "error" {`
- `internal/ai/image_comfyui.go:921` — `if msgType, _ := msgArr[0].(string); msgType == "execution_error" {`
- `internal/ai/image_comfyui.go:923` — `if em, _ := details["exception_message"].(string); em != "" {`
- `internal/ai/image_comfyui.go:957` — `fn, _ := itemMap["filename"].(string)`
- `internal/ai/image_comfyui.go:1003` — `s, _ := v.(string)`
- `internal/ai/image_cutout.go:52` — `r, _, _, _ := m.At(mb.Min.X+x, mb.Min.Y+y).RGBA()`
- `internal/ai/image_openai.go:320` — `r, _, _, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()`

### 样例：interface{}/any 泛滥(go)

- `internal/ai/client.go:46` — `OnEvent func(eventType string, data map[string]interface{})`
- `internal/ai/client.go:1130` — `c.emit("response", map[string]interface{}{`
- `internal/ai/client.go:1158` — `c.emit("request", map[string]interface{}{`
- `internal/ai/client.go:1248` — `req.ChatTemplateKwargs = map[string]any{"enable_thinking": opts.EnableThinking}`
- `internal/ai/client.go:1259` — `req.ChatTemplateKwargs = map[string]any{"enable_thinking": true}`
- `internal/ai/client.go:1412` — `func (c *Client) emit(eventType string, data map[string]interface{}) {`
- `internal/ai/image_comfyui.go:308` — `var workflow map[string]interface{}`
- `internal/ai/image_comfyui.go:378` — `if nm, ok := n.(map[string]interface{}); ok {`

### 样例：goroutine 起协程

- `internal/ai/client.go:686` — `go func() {`
- `internal/ai/copilot.go:57` — `go func() {`
- `internal/ai/copilot.go:330` — `go func() {`
- `internal/ai/image_comfyui.go:642` — `go b.pollComfyProgress(pollCtx, promptID, nodeClasses, func(status string, elapsed int, percent int, node string) {`
- `internal/ai/image_comfyui.go:749` — `go func() {`
- `internal/app/app.go:475` — `go func() {`
- `internal/app/app.go:540` — `go a.startFileWatch()`
- `internal/app/app.go:549` — `go func(id string) {`

## A4. 死代码候选（Go 导出函数/方法，除定义文件外全仓零引用，共 235 个）

> 启发式：导出名（首字母大写）在其它任何 .go 文件（含测试）的词元集合中都不出现。绑定面经 Wails 反射/生成代码调用者会自动落此列，判定时需人工确认（尤其 `internal/app/gaea_*` 与 `bindings_*`）。

| # | 位置 | 接收者 | 名称 |
|---|---|---|---|
| 1 | `internal/ai/client.go:1308` | `c *Client` | `ListModels` |
| 2 | `internal/ai/copilot.go:245` | `c *Client` | `GenerateBeats` |
| 3 | `internal/ai/copilot.go:292` | `c *Client` | `GenerateProseFromBeat` |
| 4 | `internal/analysis/evolution.go:53` | `a *Agent` | `EvolveAfterChapter` |
| 5 | `internal/app/app.go:760` | `` | `CLILogin` |
| 6 | `internal/app/brain_left.go:102` | `s *officeFactLeftSource` | `ListFactsInSpace` |
| 7 | `internal/app/gaea_cost_stage.go:14` | `` | `SetCostStageStoreForTest` |
| 8 | `internal/app/gaea_cost_stage.go:20` | `` | `ResetCostStageStoreForTest` |
| 9 | `internal/asr/herdsman_asr.go:63` | `h *HerdsmanASR` | `SetModel` |
| 10 | `internal/bookimport/reconstruct.go:212` | `` | `JSONHint` |
| 11 | `internal/bookimport/reconstruct.go:250` | `` | `CheckExpected` |
| 12 | `internal/booksource/rule.go:361` | `` | `LoadFile` |
| 13 | `internal/channels/weixin/clawbot.go:210` | `s *Server` | `HasILink` |
| 14 | `internal/channels/weixin/clawbot.go:346` | `it *imageItem` | `UnmarshalJSON` |
| 15 | `internal/channels/weixin/clawbot.go:398` | `it *fileItem` | `UnmarshalJSON` |
| 16 | `internal/channels/weixin/media_crypt.go:31` | `m ecbBlockMode` | `CryptBlocks` |
| 17 | `internal/character/character.go:517` | `a *Agent` | `BatchGenerate` |
| 18 | `internal/characterlib/portrait.go:278` | `s *Store` | `DataDir` |
| 19 | `internal/characterlib/portrait.go:286` | `s *Store` | `PortraitsDir` |
| 20 | `internal/characterlib/portrait.go:294` | `s *Store` | `PortraitFilePath` |
| 21 | `internal/config/config.go:518` | `` | `ResolveResourceDirForTest` |
| 22 | `internal/config/config_prefs.go:43` | `c *Config` | `SetReadScreenSummary` |
| 23 | `internal/config/config_prefs.go:57` | `c *Config` | `SetReadScreenKeepLast` |
| 24 | `internal/config/config_prefs.go:115` | `c *Config` | `SetIntentsLLMTimeoutMS` |
| 25 | `internal/config/walkthrough.go:24` | `` | `IsWalkthrough` |
| 26 | `internal/docmd/ocr.go:336` | `p *ovisOCRProvider` | `ExtractImage` |
| 27 | `internal/docmd/ocr.go:385` | `p *tesseractOCRProvider` | `ExtractImage` |
| 28 | `internal/gaea/agent/agent.go:542` | `a *AgentRunner` | `SetMemoryQueue` |
| 29 | `internal/gaea/agent/agent.go:572` | `a *AgentRunner` | `SetSink` |
| 30 | `internal/gaea/agent/agent.go:670` | `a *AgentRunner` | `CacheBreakCount` |
| 31 | `internal/gaea/agent/agent.go:892` | `a *AgentRunner` | `ProvName` |
| 32 | `internal/gaea/agent/canonical_todo.go:59` | `a *AgentRunner` | `CanonicalTodoState` |
| 33 | `internal/gaea/agent/compact.go:353` | `a *AgentRunner` | `SummarizeFrom` |
| 34 | `internal/gaea/agent/compact.go:380` | `a *AgentRunner` | `SummarizeUpTo` |
| 35 | `internal/gaea/agent/reasoning_language.go:14` | `` | `NormalizeReasoningLanguage` |
| 36 | `internal/gaea/agent/reasoning_language.go:27` | `` | `NormalizeResponseLanguage` |
| 37 | `internal/gaea/agent/reasoning_language.go:42` | `` | `ResponseLanguageBlock` |
| 38 | `internal/gaea/agent/reasoning_language.go:56` | `` | `ReasoningLanguageBlock` |
| 39 | `internal/gaea/agent/reasoning_language.go:69` | `` | `WithResponseLanguage` |
| 40 | `internal/gaea/agent/reasoning_language.go:79` | `` | `WithReasoningLanguage` |
| 41 | `internal/gaea/agent/reasoning_language.go:134` | `` | `WithResponseLanguagePreference` |
| 42 | `internal/gaea/agent/reasoning_language.go:142` | `` | `ResponseLanguageFromContext` |
| 43 | `internal/gaea/agent/reasoning_language.go:155` | `` | `WithReasoningLanguagePreference` |
| 44 | `internal/gaea/agent/reasoning_language.go:163` | `` | `ReasoningLanguageFromContext` |
| 45 | `internal/gaea/agent/render/sink.go:286` | `` | `DimText` |
| 46 | `internal/gaea/agent/subagent_store.go:140` | `` | `EphemeralSubagentRun` |
| 47 | `internal/gaea/agent/subagent_store.go:705` | `s *SubagentStore` | `CleanupStaleRunning` |
| 48 | `internal/gaea/agent/task.go:126` | `` | `IsSubagentMetaTool` |
| 49 | `internal/gaea/agent/task.go:401` | `t *TaskTool` | `SetTemplatePrefix` |
| 50 | `internal/gaea/agent/task.go:403` | `t *TaskTool` | `SubUsage` |
| 51 | `internal/gaea/agent/testutil/mock_provider.go:130` | `p *MockProvider` | `LastRequest` |
| 52 | `internal/gaea/agent/testutil/mock_provider.go:148` | `p *MockProvider` | `SetScript` |
| 53 | `internal/gaea/agent/testutil/mock_provider.go:180` | `` | `UsageTurn` |
| 54 | `internal/gaea/agent/testutil/mock_provider.go:193` | `` | `ErrorTurn` |
| 55 | `internal/gaea/agent/textutils/text.go:163` | `` | `HasSignalKeyword` |
| 56 | `internal/gaea/agent/textutils/text.go:254` | `` | `VisibleWidth` |
| 57 | `internal/gaea/backup/backup.go:106` | `` | `ValidateManifest` |
| 58 | `internal/gaea/cache/compiler.go:53` | `c *Compiler` | `WithInstructions` |
| 59 | `internal/gaea/cache/runtime.go:181` | `rc *RuntimeLayer` | `DetectConflict` |
| 60 | `internal/gaea/cache/runtime.go:200` | `rc *RuntimeLayer` | `MergeChildEdits` |
| 61 | `internal/gaea/cache/skill.go:50` | `l *SkillLayer` | `LockVersion` |
| 62 | `internal/gaea/cache/skill.go:145` | `l *SkillLayer` | `PromoteVersion` |
| 63 | `internal/gaea/cache/skill.go:151` | `l *SkillLayer` | `RecordOutcomeWithReason` |
| 64 | `internal/gaea/cache/skill.go:154` | `l *SkillLayer` | `DemoteVersion` |
| 65 | `internal/gaea/config/config.go:633` | `e *ProviderEntry` | `ModelList` |
| 66 | `internal/gaea/config/config.go:713` | `e PluginEntry` | `ShouldAutoStart` |
| 67 | `internal/gaea/config/edit.go:43` | `c *Config` | `SetSubagentModel` |
| 68 | `internal/gaea/config/edit.go:59` | `c *Config` | `SetSubagentModelForSkill` |
| 69 | `internal/gaea/config/recent.go:16` | `` | `RecentWorkspacesPath` |
| 70 | `internal/gaea/config/recent.go:43` | `` | `SaveRecentWorkspaces` |
| 71 | `internal/gaea/config/walkthrough.go:10` | `` | `WalkthroughWorkspaceDir` |
| 72 | `internal/gaea/context/manager.go:137` | `cm *ContextManager` | `ActiveTools` |
| 73 | `internal/gaea/context/provider_adapter.go:42` | `` | `DefaultProviderHint` |
| 74 | `internal/gaea/control/controller.go:871` | `c *Controller` | `TCCAStats` |
| 75 | `internal/gaea/control/controller.go:899` | `c *Controller` | `TCCAReport` |
| 76 | `internal/gaea/control/controller.go:1018` | `c *Controller` | `HookRunner` |
| 77 | `internal/gaea/control/controller_mcp.go:82` | `c *Controller` | `ConfiguredMCPNames` |
| 78 | `internal/gaea/control/controller_mcp.go:96` | `c *Controller` | `DisconnectedMCPNames` |
| 79 | `internal/gaea/control/controller_memory.go:306` | `c *Controller` | `SessionRemember` |
| 80 | `internal/gaea/control/controller_memory.go:470` | `c *Controller` | `SessionFacts` |

（仅列前 80 条，完整清单见 `machine/scan.json` 的 `dead_export_candidates`）

## A5. 复制粘贴证据

### A5.1 长行原样复制（≥80 字符的同一行出现在 ≥2 个文件，共 287 组）

| # | 涉及文件数 | 文件（最多 5） | 复制的行（截断） |
|---|---|---|---|
| 1 | 9 | `frontend/src/components/imagegen/AssetStudio.tsx`, `frontend/src/components/novel/ChapterAnalysisPanel.tsx`, `frontend/src/components/novel/PromptWorkshopPanel.tsx`, `frontend/src/components/novel/WorldviewSectionsEditor.tsx`, `frontend/src/pages/CharacterPage.tsx` | `import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'` |
| 2 | 7 | `frontend/src/gaea/components/AgentTree.tsx`, `frontend/src/gaea/components/BrowserPanel.tsx`, `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/SubagentsPanel.tsx`, `frontend/src/gaea/components/VersionTimeline.tsx` | `"flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-co` |
| 3 | 6 | `frontend/src/gaea/components/memoryhub/ComposeModal.tsx`, `frontend/src/gaea/components/memoryhub/CostGraphView.tsx`, `frontend/src/gaea/components/memoryhub/CostIndicatorsView.tsx`, `frontend/src/gaea/components/memoryhub/CostNotesView.tsx`, `frontend/src/gaea/components/memoryhub/CostProjectsView.tsx` | `"inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-` |
| 4 | 5 | `frontend/src/components/imagegen/AssetStudio.tsx`, `frontend/src/components/novel/PartialRewriteModal.tsx`, `frontend/src/components/novel/PromptWorkshopPanel.tsx`, `frontend/src/components/settings/AboutPanel.tsx`, `frontend/src/pages/modelcenter/EngineSection.tsx` | `<div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>` |
| 5 | 5 | `internal/analysis/analysis.go`, `internal/chapter/chapter.go`, `internal/character/character.go`, `internal/outline/outline.go`, `internal/worldview/worldview.go` | `func New(client ai.LLMClient, pm *project.Manager, cfg *config.Config, eng *prompt.Engine) *Agent {` |
| 6 | 4 | `internal/chapter/chapter.go`, `internal/character/character.go`, `internal/outline/outline.go`, `internal/whisper/memory_ingest.go` | `if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {` |
| 7 | 4 | `frontend/src/components/imagegen/ModelDirectory.tsx`, `frontend/src/components/imagegen/VisionTrial.tsx`, `frontend/src/components/novel/ChapterReviewPanel.tsx`, `frontend/src/components/novel/ForeshadowPanel.tsx` | `<div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>` |
| 8 | 4 | `frontend/src/components/imagegen/ResultStage.tsx`, `frontend/src/components/novel/BookHealthPanel.tsx`, `frontend/src/components/novel/ChapterAnalysisPanel.tsx`, `frontend/src/components/settings/AppearancePanel.tsx` | `<div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>` |
| 9 | 4 | `internal/app/gaea_docx_edit.go`, `internal/app/gaea_pptx_edit.go`, `internal/app/gaea_verify.go`, `internal/app/gaea_xlsx_edit.go` | `st, err := evidence.OpenJournal(filepath.Join(gaeaCwd(), ".gaea", "work", "journal"))` |
| 10 | 4 | `internal/channels/weixin/media_download.go`, `internal/gaea/plugin/ssrf.go`, `internal/gaea/tool/builtin/webfetch.go`, `internal/netclient/ssrf.go` | `return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))` |
| 11 | 4 | `frontend/src/components/imagegen/AssetLibrary.tsx`, `frontend/src/components/imagegen/AssetStudio.tsx`, `frontend/src/components/imagegen/ModelDirectory.tsx`, `frontend/src/components/imagegen/VisionTrial.tsx` | `style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', padding` |
| 12 | 4 | `frontend/src/gaea/components/BrowserPanel.tsx`, `frontend/src/gaea/components/DeliverablesPanel.tsx`, `frontend/src/gaea/components/SubagentsPanel.tsx`, `frontend/src/gaea/components/TaskCenter.tsx` | `<div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secon` |
| 13 | 4 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/SubagentsPanel.tsx`, `frontend/src/gaea/components/VersionTimeline.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` | `border: "1px solid color-mix(in srgb, var(--md-sys-color-warning) 32%, transparent)",` |
| 14 | 4 | `frontend/src/gaea/components/memoryhub/ComposeModal.tsx`, `frontend/src/gaea/components/memoryhub/CostInquiryPanel.tsx`, `frontend/src/gaea/components/memoryhub/CostNotesView.tsx`, `frontend/src/gaea/components/memoryhub/CostProjectsView.tsx` | `"w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none fo` |
| 15 | 4 | `frontend/src/gaea/components/memoryhub/ComposeModal.tsx`, `frontend/src/gaea/components/memoryhub/CostNotesView.tsx`, `frontend/src/gaea/components/memoryhub/CostProjectsView.tsx`, `frontend/src/gaea/components/memoryhub/FiveCalcPanel.tsx` | `"inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:o` |
| 16 | 4 | `frontend/src/gaea/lib/diff.ts`, `frontend/src/gaea/lib/diffRender.ts`, `frontend/src/gaea/lib/docxTextDiff.ts`, `frontend/src/gaea/lib/pptxTextDiff.ts` | `const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));` |
| 17 | 3 | `frontend/src/gaea/components/memoryhub/OfficeMemoryLibrary.tsx`, `frontend/src/gaea/components/memoryhub/ProfileLibrary.tsx`, `frontend/src/gaea/components/memoryhub/WhisperMemoryLibrary.tsx` | `className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint h` |
| 18 | 3 | `frontend/src/components/VoiceSettingsPanel.tsx`, `frontend/src/components/WhisperEmotionPanel.tsx`, `frontend/src/components/settings/ChatPanel.tsx` | `<div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>` |
| 19 | 3 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/VersionTimeline.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` | `border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 32%, transparent)",` |
| 20 | 3 | `frontend/src/gaea/components/DocxOutline.tsx`, `frontend/src/gaea/components/DocxPreview.tsx`, `frontend/src/gaea/components/FilePreview.tsx` | `className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-poi` |
| 21 | 3 | `frontend/src/gaea/components/memoryhub/DigitalLifeLibrary.tsx`, `frontend/src/gaea/components/memoryhub/PriceSourcesPanel.tsx`, `frontend/src/gaea/components/memoryhub/PriceSourcesRepository.tsx` | `className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-poi` |
| 22 | 3 | `internal/app/create_chapter_handler.go`, `internal/app/novel_booksource_handler.go`, `internal/project/project.go` | `if sc, rerr := sm.Read(meta.ID); rerr == nil && strings.TrimSpace(sc.Content) != "" {` |
| 23 | 3 | `internal/app/ghost_suggest.go`, `internal/app/novel_reading_handler.go`, `internal/app/scene_cards_handler.go` | `reply, err := a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{` |
| 24 | 3 | `internal/channels/weixin/media_download.go`, `internal/gaea/plugin/ssrf.go`, `internal/netclient/ssrf.go` | `DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {` |
| 25 | 3 | `internal/character/character.go`, `internal/outline/outline.go`, `internal/worldview/worldview.go` | `func (a *Agent) chat(ctx context.Context, system, user string) (string, error) {` |
| 26 | 3 | `internal/character/character.go`, `internal/outline/outline.go`, `internal/worldview/worldview.go` | `return a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{EngineID:` |
| 27 | 3 | `frontend/src/components/WelcomePage.tsx`, `frontend/src/components/settings/ImageGenPanel.tsx`, `frontend/src/components/settings/WorkspacePanel.tsx` | `background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)',` |
| 28 | 3 | `frontend/src/components/chat/SuggestionCard.tsx`, `frontend/src/components/novel/character/CharacterCard.tsx`, `frontend/src/components/novel/character/OrganizationCard.tsx` | `onKeyDown={(e) => { if (e.key === 'Enter' \|\| e.key === ' ') { e.preventDefault(); onClick() } }}` |
| 29 | 3 | `frontend/src/components/imagegen/AssetLibrary.tsx`, `frontend/src/components/imagegen/AssetStudio.tsx`, `frontend/src/components/imagegen/VisionTrial.tsx` | `borderRadius: 10, background: 'color-mix(in srgb, var(--color-text) 3%, transparent)',` |
| 30 | 3 | `frontend/src/components/imagegen/AssetLibrary.tsx`, `frontend/src/components/imagegen/AssetStudio.tsx`, `frontend/src/components/imagegen/VisionTrial.tsx` | `style={{ width: '100%', aspectRatio: '1 / 1', objectFit: 'cover', borderRadius: 7, display: 'block' ` |

### A5.2 逐字重复代码块（归一化后连续 10 行完全相同）

- Go：174 组；TS：134 组

| # | 语言 | 涉及文件数 | 文件 |
|---|---|---|---|
| 1 | Go | 4 | `internal/app/gaea_prompt_store.go`, `internal/app/gaea_route_suggestions.go`, `internal/app/gaea_skill_stats.go`, `internal/app/gaea_task_inbox.go` |
| 2 | Go | 4 | `internal/app/gaea_prompt_store.go`, `internal/app/gaea_route_suggestions.go`, `internal/app/gaea_skill_stats.go`, `internal/app/gaea_task_inbox.go` |
| 3 | Go | 4 | `internal/app/gaea_prompt_store.go`, `internal/app/gaea_route_suggestions.go`, `internal/app/gaea_skill_stats.go`, `internal/app/gaea_task_inbox.go` |
| 4 | Go | 4 | `internal/app/gaea_prompt_store.go`, `internal/app/gaea_route_suggestions.go`, `internal/app/gaea_skill_stats.go`, `internal/app/gaea_task_inbox.go` |
| 5 | Go | 4 | `internal/app/gaea_prompt_store.go`, `internal/app/gaea_route_suggestions.go`, `internal/app/gaea_skill_stats.go`, `internal/app/gaea_task_inbox.go` |
| 6 | Go | 3 | `internal/chapter/chapter.go`, `internal/character/character.go`, `internal/outline/outline.go` |
| 7 | Go | 3 | `internal/chapter/chapter.go`, `internal/character/character.go`, `internal/outline/outline.go` |
| 8 | Go | 2 | `internal/gaea/costimport/costimport.go`, `internal/gaea/knowledgeimport/knowledgeimport.go` |
| 9 | Go | 2 | `internal/gaea/costimport/costimport.go`, `internal/gaea/knowledgeimport/knowledgeimport.go` |
| 10 | Go | 2 | `internal/analysis/analysis.go`, `internal/chapter/chapter.go` |
| 11 | Go | 2 | `internal/analysis/analysis.go`, `internal/chapter/chapter.go` |
| 12 | Go | 2 | `internal/app/characterlib_gen_handler.go`, `internal/character/character.go` |
| 13 | Go | 2 | `internal/app/gaea_cost_compose.go`, `internal/app/gaea_cost_import.go` |
| 14 | Go | 2 | `internal/app/gaea_cost_compose.go`, `internal/app/gaea_cost_import.go` |
| 15 | Go | 2 | `internal/app/gaea_cost_import.go`, `internal/gaea/cost/cost.go` |
| 16 | Go | 2 | `internal/app/gaea_cost_import.go`, `internal/gaea/cost/cost.go` |
| 17 | Go | 2 | `internal/app/gaea_cost_rerank.go`, `internal/gaea/tool/builtin/cost_tools.go` |
| 18 | Go | 2 | `internal/app/gaea_cost_rerank.go`, `internal/gaea/tool/builtin/cost_tools.go` |
| 19 | Go | 2 | `internal/app/gaea_cost_rerank.go`, `internal/gaea/tool/builtin/cost_tools.go` |
| 20 | Go | 2 | `internal/app/gaea_cost_rerank.go`, `internal/gaea/tool/builtin/cost_tools.go` |
| 1 | TS | 4 | `frontend/src/gaea/app/usePreviewPanel.ts`, `frontend/src/gaea/app/useWorkspaceLayout.ts`, `frontend/src/gaea/components/ResizableDrawer.tsx`, `frontend/src/gaea/hooks/useSidebar.ts` |
| 2 | TS | 3 | `frontend/src/gaea/components/BrowserPanel.tsx`, `frontend/src/gaea/components/DeliverablesPanel.tsx`, `frontend/src/gaea/components/SubagentsPanel.tsx` |
| 3 | TS | 3 | `frontend/src/gaea/components/BrowserPanel.tsx`, `frontend/src/gaea/components/DeliverablesPanel.tsx`, `frontend/src/gaea/components/SubagentsPanel.tsx` |
| 4 | TS | 2 | `frontend/src/components/ModuleLauncher.tsx`, `frontend/src/layouts/MainLayout.tsx` |
| 5 | TS | 2 | `frontend/src/components/characterlib/CharacterLibEditor.tsx`, `frontend/src/pages/sin/SinCastPanel.tsx` |
| 6 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 7 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 8 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 9 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 10 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 11 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 12 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 13 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 14 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 15 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 16 | TS | 2 | `frontend/src/gaea/components/DeliverableCards.tsx`, `frontend/src/gaea/components/deliverables/DeliverableRow.tsx` |
| 17 | TS | 2 | `frontend/src/gaea/components/FilePreview.tsx`, `frontend/src/gaea/components/FilePreviewModal.tsx` |
| 18 | TS | 2 | `frontend/src/gaea/components/FilePreview.tsx`, `frontend/src/gaea/components/FilePreviewModal.tsx` |
| 19 | TS | 2 | `frontend/src/gaea/components/FilePreview.tsx`, `frontend/src/gaea/components/FilePreviewModal.tsx` |
| 20 | TS | 2 | `frontend/src/gaea/components/FilePreview.tsx`, `frontend/src/gaea/components/FilePreviewModal.tsx` |

### A5.3 同区域高相似文件对（≥150 行文件的词元 Jaccard ≥0.30，平行实现候选）

| # | Jaccard | A（LOC） | B（LOC） |
|---|---|---|---|
| 1 | 0.462 | `frontend/src/gaea/components/memoryhub/CostImportModal.tsx`（361） | `frontend/src/gaea/components/memoryhub/KnowledgeImportModal.tsx`（278） |
| 2 | 0.402 | `internal/office/docxedit/docxedit.go`（828） | `internal/office/pptxedit/pptxedit.go`（690） |

### A5.4 同名文件散布（同名基线出现在 ≥2 条路径，共 102 组，取最多的 20 组）

- `types`（14 处）：`internal/ai/types.go`, `internal/core/types.go`, `internal/gaea/contextview/types.go`, `internal/gaea/trajectory/types.go`, `internal/schedule/types.go`, `internal/types/types.go`
- `store`（8 处）：`internal/characterlib/store.go`, `internal/chat/store.go`, `internal/gaea/context/store.go`, `internal/gaea/knowledge/store.go`, `internal/gaea/memory/store.go`, `internal/gaea/pricefeed/store.go`
- `search`（6 处）：`internal/booksource/search.go`, `internal/gaea/knowledge/search.go`, `internal/gaea/memory/search.go`, `internal/gaea/search/search.go`, `internal/search/search.go`, `frontend/src/gaea/lib/types/search.ts`
- `cost`（6 处）：`internal/gaea/cost/cost.go`, `internal/schedule/cost.go`, `frontend/src/gaea/lib/bridge/cost.ts`, `frontend/src/gaea/lib/mock/cost.ts`, `frontend/src/gaea/lib/types/cost.ts`, `frontend/src/schedule/cost.ts`
- `memory`（5 处）：`internal/gaea/memory/memory.go`, `internal/memory/memory.go`, `frontend/src/gaea/lib/bridge/memory.ts`, `frontend/src/gaea/lib/mock/memory.ts`, `frontend/src/gaea/lib/types/memory.ts`
- `provider`（4 处）：`internal/asr/provider.go`, `internal/gaea/provider/provider.go`, `internal/realtime/provider.go`, `internal/tts/provider.go`
- `office`（4 处）：`internal/docmd/office.go`, `frontend/src/gaea/lib/bridge/office.ts`, `frontend/src/gaea/lib/mock/office.ts`, `frontend/src/gaea/lib/types/office.ts`
- `profile`（4 处）：`internal/gaea/agent/budget/profile.go`, `internal/gaea/cache/profile.go`, `internal/gaea/memory/profile.go`, `internal/style/profile.go`
- `state`（4 处）：`internal/gaea/agent/session/state.go`, `frontend/src/gaea/lib/mock/office/state.ts`, `frontend/src/gaea/lib/mock/state.ts`, `frontend/src/genui/blocks/state.ts`
- `index`（4 处）：`internal/gaea/skill/index.go`, `internal/schedule/index.go`, `frontend/src/genui/index.ts`, `frontend/src/types/index.ts`
- `util`（4 处）：`internal/novelreview/util.go`, `internal/tts/util.go`, `internal/util/util.go`, `internal/whisper/util.go`
- `memory_graph`（3 处）：`internal/app/memory_graph.go`, `internal/whisper/db/repos/memory_graph.go`, `internal/whisper/memory_graph.go`
- `manager`（3 处）：`internal/assistant/manager.go`, `internal/gaea/browser/manager.go`, `internal/gaea/context/manager.go`
- `model`（3 处）：`internal/characterlib/model.go`, `frontend/src/gaea/lib/bridge/model.ts`, `frontend/src/gaea/lib/mock/model.ts`
- `engine`（3 处）：`internal/context/engine.go`, `internal/modelengine/engine.go`, `internal/rewrite/engine.go`
- `fingerprint`（3 处）：`internal/gaea/agent/cache/fingerprint.go`, `internal/novelstyle/fingerprint.go`, `frontend/src/genui/fingerprint.ts`
- `session`（3 处）：`internal/gaea/agent/session/session.go`, `frontend/src/gaea/lib/session.ts`, `frontend/src/gaea/lib/types/session.ts`
- `skill`（3 处）：`internal/gaea/cache/skill.go`, `internal/gaea/skill/skill.go`, `internal/skill/skill.go`
- `tool`（3 处）：`internal/gaea/largefile/tool.go`, `internal/gaea/sessionquery/tool.go`, `internal/gaea/tool/tool.go`
- `bridge`（3 处）：`internal/gaea/provider/bridge/bridge.go`, `internal/httpbridge/bridge.go`, `frontend/src/gaea/lib/bridge.ts`

## A6. 测试卫生

- Go `t.Skip(` 命中 60 处（前 20）：
  - `internal/app/cost_preview_test.go:17` — `t.Skip("未设置 GAEA_SMOKE_COST")`
  - `internal/app/gaea_crosslink_test.go:55` — `t.Skip("未设置 GAEA_SMOKE_CROSS")`
  - `internal/app/gaea_deliverable_zip_test.go:91` — `t.Skip("设置 GAEA_SMOKE_CHART=1 运行真实图表 smoke")`
  - `internal/app/gaea_export_test.go:116` — `t.Skip("未设置 GAEA_SMOKE_EXPORT")`
  - `internal/app/gaea_git_test.go:17` — `t.Skip("环境无 git CLI，跳过 Git 面板测试")`
  - `internal/app/gaea_git_test.go:151` — `t.Skip("环境无 git CLI")`
  - `internal/app/gaea_git_test.go:167` — `t.Skip("TMP 落入 git 仓库内且无法选址仓库外沙箱：非仓库断言前提不可满足")`
  - `internal/app/gaea_pdf_test.go:72` — `t.Skip("设置 GAEA_SMOKE_PDF=1 运行真实 PDF 转换 smoke")`
  - `internal/app/gaea_pdf_test.go:75` — `t.Skip("未找到 soffice")`
  - `internal/app/gaea_pipeline_test.go:76` — `t.Skip("未设置 GAEA_SMOKE_PIPELINE")`
  - `internal/app/gaea_pptx_test.go:235` — `t.Skip("未设置 GAEA_SMOKE_EXPORT")`
  - `internal/app/gaea_selfgen_test.go:28` — `t.Skip("未设置 GAEA_SELFGEN（真实模型自生成测试）")`
  - `internal/app/gaea_workspace_test.go:40` — `t.Skip("APPDATA 未设置（非 Windows），跳过路径前缀断言")`
  - `internal/app/herdsman_catalog_livecheck_test.go:13` — `t.Skip("缺少实时 fixture")`
  - `internal/app/herdsman_cli_error_test.go:17` — `t.Skip("Windows 专用（.cmd 假 CLI）")`
  - `internal/app/image_domain_test.go:324` — `t.Skip("本测试要求 cfg 快照为 nil（纯绘梦会话形态）")`
  - `internal/app/imagegen_pressure_test.go:53` — `t.Skip("仅 Windows")`
  - `internal/auth/token_test.go:325` — `t.Skip("DPAPI 不可用（降级模式原样返回），无法构造解密失败场景")`
  - `internal/docmd/markitdown_test.go:14` — `t.Skip("markitdown 不可用")`
  - `internal/docmd/ovis_test.go:81` — `t.Skip("设置 GAEA_TEST_SCAN_PDF 指向扫描件 PDF 后运行端到端 OCR 测试")`
- TS `it.skip/only/todo` 命中 0 处

### 最大的测试文件 TOP 25

| # | LOC | 文件 |
|---|---|---|
| 1 | 1218 | `internal/gaea/tasks/tasks_test.go` |
| 2 | 1128 | `internal/modelengine/engine_test.go` |
| 3 | 949 | `frontend/src/pages/CreatePage.test.tsx` |
| 4 | 937 | `internal/gaea/browser/manager_test.go` |
| 5 | 928 | `internal/config/config_test.go` |
| 6 | 874 | `internal/booksource/booksource_test.go` |
| 7 | 857 | `internal/app/novel_plan_handler_test.go` |
| 8 | 842 | `internal/channels/weixin/capture_test.go` |
| 9 | 838 | `internal/app/image_handler_test.go` |
| 10 | 794 | `internal/types/compat_test.go` |
| 11 | 788 | `frontend/src/gaea/components/DeliverablesPanel.test.tsx` |
| 12 | 759 | `frontend/src/pages/ChapterPage.test.tsx` |
| 13 | 750 | `internal/whisper/whisper_test.go` |
| 14 | 677 | `internal/app/chat_service_test.go` |
| 15 | 669 | `internal/gaea/contextview/fold_test.go` |
| 16 | 657 | `internal/modelengine/glm_catalog_test.go` |
| 17 | 653 | `frontend/src/gaea/components/ContextView.test.tsx` |
| 18 | 651 | `frontend/src/schedule/GanttView.test.tsx` |
| 19 | 639 | `internal/app/sin_stream_guard_test.go` |
| 20 | 638 | `internal/app/gaea_dag_test.go` |
| 21 | 631 | `internal/channels/weixin/clawbot_test.go` |
| 22 | 627 | `internal/netclient/netclient_test.go` |
| 23 | 616 | `internal/app/create_chapter_plan_gate_test.go` |
| 24 | 616 | `frontend/src/pages/NovelSettingPage.test.tsx` |
| 25 | 616 | `frontend/src/gaea/components/VersionTimeline.test.tsx` |

## A7. 上帝包（LOC TOP 30 的 Go 包目录）

| # | LOC | 文件数 | 包目录 |
|---|---|---|---|
| 1 | 13703 | 67 | `internal/gaea/agent` |
| 1 | 9391 | 43 | `internal/gaea/tool` |
| 1 | 5262 | 28 | `internal/gaea/memory` |
| 1 | 4455 | 12 | `internal/gaea/control` |
| 1 | 2641 | 28 | `internal/whisper/db` |
| 1 | 2264 | 7 | `internal/channels/weixin` |
| 1 | 2137 | 10 | `internal/gaea/config` |
| 1 | 2012 | 6 | `internal/gaea/browser` |
| 1 | 1865 | 6 | `internal/gaea/cost` |
| 1 | 1731 | 6 | `internal/gaea/contextview` |
| 1 | 1566 | 6 | `internal/gaea/skill` |
| 1 | 1547 | 9 | `internal/gaea/plugin` |
| 1 | 1432 | 7 | `internal/gaea/cache` |
| 1 | 1430 | 1 | `internal/ai/client.go` |
| 1 | 1342 | 2 | `internal/gaea/costref` |
| 1 | 1268 | 1 | `internal/voice/voice_manager.go` |
| 1 | 1265 | 1 | `internal/gaea/costimport` |
| 1 | 1245 | 1 | `internal/gaea/tasks` |
| 1 | 1193 | 4 | `internal/gaea/boot` |
| 1 | 1189 | 1 | `internal/app/gaea_ui_extra.go` |
| 1 | 1111 | 1 | `internal/project/project.go` |
| 1 | 1096 | 1 | `internal/app/image_handler.go` |
| 1 | 1065 | 6 | `internal/gaea/knowledge` |
| 1 | 1065 | 5 | `internal/gaea/provider` |
| 1 | 1052 | 1 | `internal/ai/image_comfyui.go` |
| 1 | 1037 | 1 | `internal/novelcontext/novelcontext.go` |
| 1 | 1034 | 1 | `internal/whisper/orchestrator.go` |
| 1 | 1020 | 1 | `internal/app/whisper_handler.go` |
| 1 | 1019 | 1 | `internal/app/create_chapter_handler.go` |
| 1 | 1008 | 4 | `internal/gaea/trajectory` |

## A8. 仓库体积（MB）与文档水位

| 目录 | MB |
|---|---|
| `frontend/node_modules` | 645.1 |
| `clones` | 623.4 |
| `releases` | 596.7 |
| `.tmp` | 210.6 |
| `backups` | 120.4 |
| `whisper_data` | 49.5 |
| `internal` | 11.9 |
| `dist` | 10.7 |
| `frontend/src` | 9.7 |
| `docs` | 7.2 |
| `node_modules` | 0.1 |

- `CHANGELOG.md` = 1279289 字节（1.22 MB，单文件）
