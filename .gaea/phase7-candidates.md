# 阶段 7 候选证据草稿（父代理侦察，2026-08-14）

> 待 5 份独立子代理审查报告到达后合并定稿。

## A. v2.24.0/v2.25.0 明确遗留（可复现证据）

1. **非流式调用无重试**：internal/ai/client.go:326-351 Chat 非流式——连接错误/5xx 直接失败返回，
   仅 xAI 401 刷新重试（:340-345）。流式已加固（T6-1.1），非流式未做。
2. **image backend 无代理**：internal/ai/image_comfyui.go:41 用 netclient.NewSimpleClient(30min)，
   未接入 NetworkProxySpec（流式已接入，image 未做）。
3. **CacheReport 子项命中统计不聚合**：internal/gaea/context/metrics.go:149-158 Report() 聚合 children
   时漏 CacheHitTokens/CacheMissTokens/BreakCount（只聚合 Saved*/ForkCount 等旧字段）；metrics_test.go
   无对应断言。
4. **看门狗阈值未 UI 化**：internal/gaea/control/watchdog.go 有 WatchdogConfig（墙钟 10min/停滞 30s），
   Options.Watchdog 可配，但 app 层无绑定、前端无设置入口。

## B. 前端收敛遗留（T6-10 未覆盖）

5. **静默 catch 残留**：store.ts 32 处 .catch(() => {})（含 catch(() => []) 等降级）；全前端 146 处
   catch(() => {}) 模式（useVoiceChat 20/KnowledgePanel 9/CostLibraryView 6 等）。T6-1.2 只处理了
   store.ts 8 集群 14 处。
6. **eslint 359 存量 warnings**：主体为 no-empty（空块）+ @typescript-eslint/no-unused-vars（_ 参数），
   集中在降级 catch 与 mock；0 errors。
7. **巨型文件超过 400 行约定**：bridge.ts 1221、types.ts 1066、CostLibraryView 998、App.tsx 920、
   Sidebar 886、XlsxPreview 813、CharacterPage 783、Transcript 665、KnowledgePanel 567、
   imageTemplates 563、ChatPanel 552、StatsPanel 548、useVoiceChat 543、ControlPanel 543、
   store.ts 537、CharacterLibEditor 537、PriceSourcesPanel 497、engines.ts 493、MemoryPanel 482、
   Markdown 478、MainLayout 470、WorkspaceSearchPanel 448、Composer 438、RelationGraph 417、
   DocxPreview 407、ChatPage 407。
8. **后端巨型文件**（>800 行）：image_handler.go 1181（37 funcs）、ai/client.go 1128、config/config.go 1120、
   control/controller.go 1003、whisper/orchestrator.go 990、app/gaea_ui_extra.go 932、
   app/whisper_handler.go 879、voice/voice_manager.go 860（35 funcs）、ai/image_comfyui.go 856、
   office/docxedit/docxedit.go 825。

## C. 无测试包/死绑定

9. **6 个 Go 包零测试**：asr（2 文件 324 行，StreamASR websocket 链路）、chapter（175 行）、
   search（131 行）、stats（242 行）、style（166 行）、visual（309 行）。
10. **死绑定**：GaeaAgentMode（gaea_ui_extra.go:393 return ""）、GaeaWorkspaceChanges
    （gaea_ui_extra.go:617 return 空数组）——前端 bridge.ts 仅声明无 UI 调用；bindingNames.ts 含名。
11. **前端 hooks 缺测试**：useBridgeWatch/useCapabilitiesData/useComposerAttachments/useComposerMenus/
    useComposerWorkspace/useLayoutSizes/useModeManager/usePreviewProgress/useSidebar/useTodoExtractor/
    useToolStats（11 个无测试）。

## D. 既有候选（知识栈总结 2026-08-14 未交付项）

12. 轻语聊天记忆查重（只读库，需先开放写路径或只提示不合并）。
13. 票据批量入表（多文件队列化识别，T5-5 单文件已闭环）。
