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
| ✅ | 阶段 4 个人使用收口 v2.20.0（2026-08-14，用户决策「不商用」）：阶段 4 重新定标（删安装器/自动更新/代码签名等商用项）。P4-3 数据可迁移：internal/gaea/backup（Plan/Create/Extract/WritePending/ApplyPending，SQLite VACUUM INTO 一致性快照 + zip-slip 防穿越）+ App 绑定（GaeaDataBackupInfo/Create/Restore/Pending/Cancel/RestoreResult + Startup applyPendingRestore 钩子，恢复前自动备份当前数据）+ 设置「数据」分类 DataPanel（备份清单/一键备份/从备份恢复/待应用告警/恢复结果提示）。P4-1 模块收口：微信通道 beta 标注、移动端冻结标注。P4-4 磁盘治理：releases 清理（删 24 个旧 exe，保留 5 版）+ README 约定。验证：Go backup 6 测试 + App 3 测试 + internal/app 全量 ok（29.97s）、vet 干净、gen_bindings 442 方法、tsc/eslint 0 errors、vitest 251/251、真实 E2E（备份 zip 无 wal/shm、恢复→重启自动应用→before 目录生成）。发布产物 gaea-v2.20.0.exe（SHA256=49B72348...） |
| ✅ | 独立子代理审查 v2.20.1（2026-08-14）：data_backup_review.md 发现 3 高危+4 中危+9 低危，全部修复——#1 恢复重试幂等（两阶段）、#2 home 配置恢复（HomeConfigRel 点前缀根因）、#3 快照 busy_timeout+checkpoint 回退+Warnings、#4 失败写 result、#5 已有 pending 拒绝+随机后缀+孤儿清理、#6 dirSize 缓存、#7 Rollback 回滚、#8~#17 低危。验证：backup 9 测试+App 5 测试、go 全量 ok、vitest 251/251、真实 E2E（二次恢复拒绝/重启应用/home 配置）。发布 gaea-v2.20.1.exe（SHA256=F7347E70...） |
