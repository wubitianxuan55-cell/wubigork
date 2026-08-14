# 任务进度

> 最后更新: 2026-08-14

| 状态 | 任务 |
|------|------|
| ✅ | 长期规划定稿（docs/superpowers/plans/2026-08-14-gaea长期规划-herdsman底座加固与工程门禁.md） |
| ✅ | H0-1 环境探测与兼容契约（internal/herdsman/probe.go + App.HerdsmanProbe） |
| ✅ | H0-2 服务健康检查（internal/herdsman/health.go + App.HerdsmanHealth） |
| ✅ | H0-3 TTS 默认模型动态解析（voice.ResolveHerdsmanTTSModel + voice_handler 接入） |
| ✅ | H0-4 LAN 暴露检测与告警（internal/herdsman/lancheck.go + App.HerdsmanSecurityCheck） |
| ✅ | H0-5 模型用途建议（模型库卡片 Hint）+ 思考模式 max_tokens 守护（bridge/ai 两路径） |
| ✅ | E1-1 前端 CI 修复（eslint 配置/插件、28 硬错误清零、CI 加 vitest、ComfyUI 路径文案） |
| ✅ | E1-2 发布冒烟脚本 scripts/smoke.ps1 |
| ✅ | E1-3 版本节奏转周更：v2.16.0 |
| ✅ | 发布文档：CHANGELOG / README / releases/v2.16.0.md / wails.json productVersion=2.16.0 |
| ✅ | 变更面逐包验证全绿（herdsman/app/bridge/ai/voice/tts；tsc；eslint 0 errors；模型库卡片 vitest 5/5） |
| ✅ | E1-4 模型中心资源协同 + 磁盘治理：生命周期操作串行化（herdsmanOpMu）+ 模型库磁盘 KPI（installed_bytes/disk_total/disk_free）+ fmtSize TB 档；全量 vitest 243/243 通过 |
| ✅ | v2.16.0 提交 692c959；v2.16.1 提交 660b5aa |
| ✅ | 环境问题解决：go telemetry off（持久）；danger-full-access 后构建缓存/遥测写入解禁；vite 构建 24s 成功（独立）；wails build -s 8.9s 产出 32MB exe |
| ✅ | 发布产物完成：releases/gaea-v2.16.1.exe + SHA256SUMS-v2.16.1.txt + smoke.ps1 冒烟通过（/api/health 200） |
| ✅ | 发布收尾：versioninfo.rc 2.15.0→2.16.1；build/windows/info.json 补 product_version（ProductVersion 0.0.0.0 修正为 2.16.1.0）；重打包 + 冒烟通过；SHA256=BD208094... |
| ✅ | 记忆更新：AGENTS.md 清理混合编码→干净 UTF-8（版本状态→v2.16.1、发布流程补版本资源步骤、新增沙箱环境备忘）；环境备忘 docs/2026-08-14-sandbox-environment-notes.md |
| ✅ | 阶段 2 安全收敛 v2.17.0（2026-08-14）：S2-1 LAN 暴露全局告警横幅 + 设置「安全」面板；S2-2 WebView2 远程调试 GAEA_WEBVIEW_DEBUG 开关默认关 + HTTP 桥接一次性 token（GAEA_HTTP_TOKEN/自动生成，Bearer/X-Gaea-Token/?token=）；S2-3 绑定面拆分（429 方法 → CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB，scripts/gen_bindings 生成 + TestBindingsCompleteness 兜底；前端 bridge.ts 单点路由 + wailsjsCompat 兼容层）；S2-4 敏感域本地化开关（sensitive_local 默认开，GaeaCostImportAIParse 走 routeSensitiveLocal 强制本地 Herdsman）。验证：go build/vet 全绿、internal/app 全量 19.8s ok、tsc/eslint 0 errors、vitest 243/243、vite build、冒烟 + token 端到端（401/200）。发布产物 gaea-v2.17.0.exe（SHA256=ADBFD953...） |
| ✅ | 阶段 3 首轮 v2.18.0（2026-08-14）：D3-1 跨库统一语义检索并入「资料」（kind=file，复用文件索引持久化向量）+ GaeaSemanticIndexStatus/Counts；D3-2 分流统计 GaeaUsageOverview（gaea 调用记录 + herdsman events.jsonl 打通，本地=events 全量+其他本地引擎不重复计；云端实际混合单价折算节省，无云端回退 deepseek-v4-flash ¥1.5/MTok）+ 模型中心「本地 vs 云端」对比卡；D3-3 测评产品化（herdsman /api/benchmarks：GaeaBenchmarkList/Start/Detail/Export + 模型中心「受控测评」分类，10 项任务预设、上下文 4K~32K、并发 1/2/4、Markdown 报告导出；真实端到端发起 202→succeeded）。验证：go 全量 ok、tsc/eslint 0 errors、vitest 246/246、冒烟通过。发布产物 gaea-v2.18.0.exe（SHA256=5AB1DC35...） |
| ✅ | 阶段 3 第二刀 v2.19.0（2026-08-14，D3-4 补测评缺口）：报告专项分析（每模型对比/长上下文 TTFT vs ctx/缓存复用 first vs second+prefill 加速比/显存参数 effective_launch_params/并发说明）；压力专项任务预设 3 项（长上下文/长输出/显存）；快速流式探针 GaeaBenchmarkStreamProbe（SSE 直连 herdsman /v1/chat/completions，观测 TTFT/分块数/max_gap 间隔/[DONE] 完整性）+ 模型中心「快速流式探针」区。验证：go 全量 ok（26.5s）、tsc/eslint 0 errors、vitest 247/247、真实端到端探测成功（冷启动 TTFT 15.2s、60 块、max_gap 83ms）。发布产物 gaea-v2.19.0.exe（SHA256=CB338574...）。注：同路径 exe 二开会被 WebView2 数据目录占用（8007139f）——E2E 用副本路径 |
