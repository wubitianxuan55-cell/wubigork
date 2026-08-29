# gaea 长期规划：Herdsman 底座加固与工程门禁（2026-08-14 定稿）

> 依据：代码盘点（2026-08-13）、`docs/2026-08-office-cloud-local-architecture-review.md`（D1–D9）、
> `docs/2026-08-12-herdsman-models-evaluation-report.md`（120 组对照测评）、
> `docs/2026-08-13-herdsman-tts-ocr-comparison.md`、本机 `C:\Users\wubi\.herdsman` 实地盘点。
> 原则：先止血（herdsman 底座可靠性）→ 再门禁（工程自动化）→ 后纵深（安全/架构/成本）。
> 与用户 2026-08-10 既定决策对齐：尽量免费；出图先 XAI 再 Herdsman；不固化「常规→本地」强制路由。

## 背景：herdsman 是 gaea 本地优先战略的底座

本机事实：

- 已装 14 个模型约 110GB（35B 对话 ×2、zimage-turbo、Gemma4-12B、voxcpm2、mineru、
  embedding ×2、reranker ×2、Hy-MT 翻译、paddleocr、sherpa-onnx ASR）；
- `qwen3-tts-*` 三个 TTS 模型目录 0MB（未安装），`edge-tts` 服务端返回空音频，本地可用 TTS 实际只有 voxcpm2；
- gaea 的 8 条本地能力降级链（OCR/文档解析/embedding/rerank/视觉/ASR/TTS/生图/翻译/routine_llm）
  全部挂在 herdsman 一个服务（localhost:8080）上；
- gaea 依赖 herdsman 多个**非公开契约**：`herdsman.exe skill models ... --json`、
  `launch_records/*.json`、`model_stats/events.jsonl`、`skill-operations.json`、
  `thinking_enabled` / `reasoning_effort` 扩展字段；
- herdsman 当前 `lan_accessible: true` 且 API 无鉴权，benchmark API 可读本机路径与运行端口。

## 阶段划分

### 阶段 0 · Herdsman 底座加固（立即，H0-1 ~ H0-5）

| 编号 | 任务 | 内容 | 验收 |
|---|---|---|---|
| H0-1 | 环境探测与兼容契约 | 探测 herdsman 安装（HERDSMAN_EXE/默认路径）、CLI 可用、/v1/models 可达、关键数据文件（launch_records/model_stats/skill-operations/config.yaml）可解析；结构化 ProbeResult + 兼容告警 | Go 单测 + go test 全绿 |
| H0-2 | 服务健康检查 | 8080 端口占用、各能力端点可达性、已装模型按能力归类健康视图、一键拉起入口 | Go 单测 + 全绿 |
| H0-3 | TTS 默认值动态解析 | 默认不再硬编码 qwen3-tts-customvoice（本机未装）；运行时按已装模型解析（voxcpm2 → qwen3-tts-* → edge-tts） | Go 单测 + 全绿 |
| H0-4 | LAN 暴露检测与告警 | 解析 herdsman config.yaml 的 lan_accessible/port，暴露时返回结构化告警与处置指引（gaea 侧提示，不改 herdsman） | Go 单测 + 全绿 |
| H0-5 | 模型默认值对齐测评结论 | 日常聊天默认回 HauhauCS（65 tok/s、识图表格最准）；LynnStyle 标注「可审计推理」用途；思考模式 max_tokens ≥4096 传参守护 | 代码核对 + 提示文案 |

### 阶段 1 · 工程门禁（1–2 周）

| 编号 | 任务 | 内容 |
|---|---|---|
| E1-1 | 前端 CI 修复 | 清零预存 TS 编译错误；移除 `continue-on-error`；vitest 进 CI |
| E1-2 | 发布冒烟测试 | wails build 后自动启动 exe + 探活（进程/HTTP 桥接/日志无 panic） |
| E1-3 | 版本节奏 | 冻结一周功能开发；改为周版本（下一版本直接 v2.16.0 起） |
| E1-4 | 资源调度协同 | 模型中心批量操作队列化（herdsman `local_concurrency: 1` + 8s 换模）；显示「等待中」；一键恢复上次模型组合（复用 launch_records） |

### 阶段 2 · 安全与架构收敛（2–4 周）

| 编号 | 任务 | 内容 |
|---|---|---|
| S2-1 | herdsman LAN 风险处置 | 首启检测 + 醒目告警 + 指引关闭（S2-1 与 H0-4 前端联动） |
| S2-2 | gaea 自身安全开关 | WebView2 远程调试改 env 开关默认关；HTTP 桥接加一次性 token |
| S2-3 | App 绑定面拆分 | 294 个导出方法按板块拆多个绑定对象（只挪不改逻辑，测试兜底） |
| S2-4 | 敏感数据本地通道 | 成本/报价类 AI 操作默认本地（D8），可配置回云端 |

### 阶段 3 · 数据与成本纵深（1–2 月）

| 编号 | 任务 | 内容 |
|---|---|---|
| D3-1 | 持久化向量索引 | bge-m3 向量持久化 + 跨库统一语义检索（记忆/知识/成本/资料） |
| D3-2 | 分流统计面板 | 复用 `model_stats/events.jsonl`（TTFT/TPS/token），打通 gaea 侧调用记录，出「本地 vs 云端节省对比」 |
| D3-3 | 测评产品化 | 复用 herdsman `/api/benchmarks` 异步测评 + 报告导出，把 120 组对照方法学做成模型中心「一键受控测评」 |
| D3-4 | 补测评缺口 | 长上下文（>4K）、并发、显存压力、流式中断恢复专项 |

### 阶段 4 · 个人使用收口（2026-08-14 重新定标）

> 用户 2026-08-14 决策：**gaea 个人使用、不商用**。阶段 4 不再做产品化分发
> （安装器/自动更新/代码签名/SmartScreen 等商用项全部删除），聚焦个人使用最需要的
> 三项：**数据可迁移**（换机/重装/防丢）、**模块收口**（降低维护面）、**磁盘治理**。
> 发布形态保持「exe + SHA256SUMS + 冒烟 + 升级文档」即可，升级 = 替换 exe（数据在用户目录，不动）。

| 编号 | 任务 | 内容 | 验收 |
|---|---|---|---|
| P4-1 | 模块收口 | 微信助手标注 beta/冻结（gateway 现状：4 绑定中 2 凭证失效）；移动端访问标注冻结 | 前端标注 + 文案 |
| P4-2 | 发布形态简化 | 删除商用项（安装器/自动更新/代码签名）；升级文档写「替换 exe + 数据备份先行」 | 文档 + 发布流程核对 |
| P4-3 | 数据可迁移 | 一键备份/恢复：Hephaestus.db（记忆/知识/成本/语义向量）+ whisper_data（hermes/office/角色库/聊天）+ 配置 + sessions，zip + manifest，恢复前先备份、需重启生效 | Go 单测 + 全绿 + 真实 zip 往返 |
| P4-4 | 磁盘治理 | 模型库已装总量/磁盘余量 KPI（v2.16.1 已交付）+ releases 保留最近 5 版约定 | README 约定 + 清理 |

## 明确不做（战略聚焦）

- 不加新功能板块（8 个板块已超单作者带宽）；
- 不做三脑记忆/引擎大重写（收益不确定、风险高）；
- 不固化「常规→本地」强制路由（尊重 2026-08-10 用户决策，保持云端统筹按需调度）；
- 不追新模型热点（维持「先受控测评、再接入」纪律）。

## 执行记录

- 2026-08-14：规划定稿；启动阶段 0（H0-1 ~ H0-4 并行实施）。
- 2026-08-14（本轮收尾）：
  - ✅ H0-1 环境探测与兼容契约：`internal/herdsman/probe.go` + `internal/app/gaea_herdsman_probe.go`（config/CLI/API/四数据契约 + 中文告警，HERDSMAN_PROBE_LIVE 控制真实探测）；
  - ✅ H0-2 服务健康检查：`internal/herdsman/health.go` + `internal/app/gaea_herdsman_health.go`（端口/API/十能力归类/Summary）；
  - ✅ H0-3 TTS 默认动态解析：`internal/voice/tts_model_resolve.go`（voxcpm2 优先）+ voice_handler 接入（回退标记透传前端）；
  - ✅ H0-4 LAN 暴露检测与告警：`internal/herdsman/lancheck.go` + `internal/app/gaea_herdsman_sec.go`（零依赖 YAML 解析 + 处置指引）；
  - ✅ H0-5 模型用途提示（后端 Hint + 前端卡片渲染）+ 思考模式 max_tokens 守护（bridge/ai 两路径 + 单测）；
  - ✅ E1-1 前端 CI 修复：eslint 插件/配置/script 修复、28 硬错误清零（含 Lightbox 12 处条件 Hook 隐患）、
    ci.yml 去 continue-on-error + vitest 步骤、ComfyUI 路径文案修复；
  - ✅ E1-2 发布冒烟脚本 `scripts/smoke.ps1`；
  - ✅ E1-3 周版本节奏：v2.16.0；发布文档（CHANGELOG/README/releases/v2.16.0.md/releases/README.md/wails.json/.gaea/progress.md）已更新；
  - 验证：变更面逐包全绿（internal/herdsman、internal/app 22s 全量、bridge/ai/voice/tts；internal/chat 经手动编译二进制 PASS 证实沙箱 exec 拒绝为环境因素）；
    tsc -b 通过、eslint 0 errors、模型库卡片 vitest 5/5。本会话沙箱对长进程树（go test 全量/vitest 全量/wails build）有限制，
    全量门禁交由 GitHub Actions CI 与本地 `scripts/ci.ps1` / `scripts/smoke.ps1` 执行。
  - ⏳ wails build → gaea-v2.16.0.exe + SHA256SUMS + 冒烟：本会话沙箱下 vite 前端编译挂起（子进程边界），
    已移交本机执行（`cmd /c build.bat` → 复制到 releases/gaea-v2.16.0.exe → 生成 SHA256SUMS-v2.16.0.txt →
    `pwsh -File scripts\smoke.ps1` 冒烟）。
  - ✅ 收尾补丁（子代理复盘发现，与探测契约对齐）：`herdsmanExePath` 补 PATH 回退（原来只有硬编码默认路径）；
    probe 默认根目录支持 HERDSMAN_DATA_DIR（与 lifecycle 口径统一）；`parseHerdsmanModelStats` 剥 BOM
    （原来首条统计记录会被静默跳过）。各附回归测试：internal/herdsman 30 测试 PASS（手动二进制验证）、
    internal/app 全量 ok、vet/gofmt 干净。
  - 📋 留待后续（记录在案）：HerdsmanModelStats/Operations 双通道报错收敛为单通道；
    health.go DefaultPort 与 lancheck.go defaultPort 常量语义合并；GetAppInfo 接收者风格统一。
- 2026-08-14（阶段 1 E1-4）：
  - ✅ 生命周期操作串行化：`herdsmanOpMu` 串行 Start/Stop/Download/Uninstall
    （对齐 herdsman local_concurrency=1；并发验证最大在飞=1）；
  - ✅ 磁盘治理：`herdsmanDiskInfo`（GetDiskFreeSpaceEx，可注入替身）+ HerdsmanCatalog
    installed_bytes/disk_total/disk_free/disk_error + 模型库「已装空间/磁盘余量」KPI + fmtSize TB 档；
  - ✅ 全量前端 vitest 243/243（59 文件）在本会话首次完整跑通；v2.16.1 文档就绪。
  - ✅ 环境问题解决（2026-08-14）：
    1. Go 遥测噪音 → `go telemetry off`（持久生效，不再报 upload.token 错误）；
    2. 构建缓存/遥测写入拒绝 → 随 danger-full-access 策略解除（默认 GOCACHE 可用，无需覆盖）；
    3. wails build 前端编译挂起 → 根因是 wails 捕获前端输出的管道边界；独立 `npm run build`
       （24s）+ `wails build -s`（8.9s）可完整构建；
    4. `go test ./...` 全量 81 ok；个别包测试二进制偶发 fork/exec Access is denied——
       经 `go test -c` 手动编译运行全部 PASS，确认纯环境因素非代码问题；
       `scripts/test-all.ps1`（逐包 + 重试 + 状态续跑）作为沙箱内全量验证工具。
  - ✅ v2.16.1 发布产物：releases/gaea-v2.16.1.exe（32MB）+ SHA256SUMS-v2.16.1.txt +
    `scripts/smoke.ps1` 冒烟通过（/api/health 200 {"status":"ok"}）。
- 2026-08-14（阶段 2 安全与架构收敛 v2.17.0）：
  - ✅ S2-1 herdsman LAN 风险处置前端联动：全局安全横幅 `components/SecurityBanner.tsx`
    （启动调 App.HerdsmanSecurityCheck，暴露时醒目告警 + 指引 + 重新检测/本次忽略）；
    设置页新增「安全」分类 `components/settings/SecurityPanel.tsx`（LAN 检测 + 敏感域开关 + 调试说明）；
  - ✅ S2-2 gaea 自身安全开关：main.go WebView2 远程调试改 `GAEA_WEBVIEW_DEBUG=1` 才开启
    （此前 WebviewDisableRendererCodeIntegrity=true 连带开 9333，Wails v2.13 源码行为）；
    HTTP 桥接一次性 token（httpbridge.SessionToken/ServeWithToken：GAEA_HTTP_TOKEN 或自动生成
    32 位 hex；/api/rpc、/api/stream 须 Bearer/X-Gaea-Token/?token=，/api/health 开放；CORS 放行
    Authorization；前端 httpToken.ts + bridge.ts/runtimePolyfill.ts 透传）；单测 5 场景 + 端到端验证；
  - ✅ S2-3 App 绑定面拆分：429 个导出方法（App + core/writingState/mediaState/whisperState/
    officeState）→ 10 个板块门面（CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/
    CharlibB，internal/app/bindings_*.go），方法体零改动纯委托；`scripts/gen_bindings`（go/ast）
    生成 + `TestBindingsCompleteness` 反射完备性测试兜底（App 方法集与门面并集全等）；
    main.go 绑定 `app.NewBindings(application)`；前端单点适配（gaea/lib/bridge.ts realApp 按方法名
    路由门面、api/bridge.ts 补 window.go.app.App 兼容代理、27 处旧 wailsjs 导入改指
    src/wailsjsCompat 重导出）；wailsjs 重新生成；
  - ✅ S2-4 敏感数据本地通道：config `sensitive_local`（默认开，~/.gaea_config.json 持久化）+
    `routeSensitiveLocal`（成本/报价 AI 强制本地 Herdsman，不可用回退常规）+ GaeaCostImportAIParse
    接入 + App.GetSensitiveLocal/SetSensitiveLocal + 设置页开关；单测 4 组（强制/停用回退/关闭/默认）；
  - ✅ 验证：go build/vet 全绿；internal/app 全量 19.8s ok（含绑定完备性）；config/httpbridge
    新增用例 ok；tsc -b、eslint 0 errors、vitest 243/243、vite build；冒烟通过 +
    HTTP 桥接 token 端到端（无 token 401 / Bearer 200）；
  - ✅ v2.17.0 发布产物：releases/gaea-v2.17.0.exe（33,725,952 字节，SHA256=ADBFD953904C359EFFB125585B4B4C2D8E52B3D33C4DBC39E4F64AD8CD4DD2A9）
    + SHA256SUMS-v2.17.0.txt；发布文档（CHANGELOG/README/releases/v2.17.0.md/releases/README.md/
    wails.json/versioninfo.rc）已更新。
- 2026-08-14（阶段 3 数据与成本纵深 v2.18.0，D3-1 ~ D3-3 首轮）：
  - ✅ D3-1 持久化向量索引：GaeaSemanticSearch 跨库统一语义检索并入工作区资料（kind=file，
    复用文件索引 cron 维护的持久化向量，检索不扫描）；新增 GaeaSemanticIndexStatus（各 kind
    向量条数）+ semantic.Store.Counts()；
  - ✅ D3-2 分流统计面板：GaeaUsageOverview 打通 gaea 侧调用记录（云端引擎费用估算）与
    herdsman events.jsonl 本地遥测——本地口径 = events 全量 + 其他本地引擎（herdsman 不重复计）；
    节省 = 本地 token × 云端实际混合单价（无云端回退 deepseek-v4-flash ¥1.5/MTok）；
    模型中心「调用统计」新增「本地 vs 云端·节省对比」卡；
  - ✅ D3-3 测评产品化：复用 herdsman /api/benchmarks——GaeaBenchmarkList/Start/Detail/Export
    （POST 发起契约对齐 runs.json config；明细含逐 case TTFT avg/p95、token、cached_tokens）；
    模型中心新增「受控测评」分类（10 项任务预设蒸馏自 120 组对照测评方法学、上下文 4K~32K
    覆盖长上下文、并发 1/2/4、Markdown 报告导出）；真实 herdsman 端到端验证
    （POST 202 → 40s 完成 succeeded，TTFT 119ms）；
  - ✅ 验证：go 全量 ok（internal/app 29s，含分流/测评/semantic Counts 新用例）、vet 干净、
    tsc/eslint 0 errors、vitest 246/246（新增 BenchmarkSection 3/3）、vite build、冒烟通过；
  - ✅ v2.18.0 发布产物：releases/gaea-v2.18.0.exe（33,776,640 字节，SHA256=5AB1DC3578A4C868758D7D7992C1027BB052242AB0F450A2D02AB09175E9C4FC）
    + SHA256SUMS-v2.18.0.txt；发布文档已更新。
  - 📋 留待后续：D3-3 显存压力/流式中断专项报告模板（上下文与并发入口已就绪）；D3-4 专项测评。
- 2026-08-14（阶段 3 第二刀 v2.19.0，D3-4 补测评缺口）：
  - ✅ D3-4 报告模板化：renderBenchmarkReport 新增 5 个专项段落——每模型对比（同任务横向
    可比）、长上下文专项（TTFT vs context_size）、缓存复用专项（first vs second TTFT +
    prefill 加速比/省时）、显存相关启动参数（effective_launch_params 关键字段：
    gpu_layers/no_kv_offload/batch/ubatch/cache_type）、并发专项说明；
  - ✅ D3-4 压力预设：受控测评任务集新增 压力·长上下文 / 压力·长输出 / 压力·显存（长上下文
    +长输出）3 项，配合上下文 4K~32K、并发 1/2/4、max_tokens 组合成压力矩阵；
  - ✅ D3-4 流式探针：`GaeaBenchmarkStreamProbe(model)` 直连 herdsman /v1/chat/completions
    SSE，观测 TTFT/分块数/max_gap（卡顿指示）/是否 [DONE] 收尾（断流检测）；模型中心
    「受控测评」新增「快速流式探针」区（每已安装模型一键探测，结果内联）；
  - ✅ 验证：go 全量 ok（26.5s，新增探针 3 组 + 报告分析 1 组）、vet 干净、tsc/eslint
    0 errors、vitest 247/247（BenchmarkSection 4/4）、真实端到端探测成功（冷启动
    TTFT 15.2s、60 块、max_gap 83ms、completed）；
  - ✅ v2.19.0 发布产物：releases/gaea-v2.19.0.exe（33,808,384 字节，SHA256=CB338574993322F6C74A8916ABA46ED0990B65A7D43047E69D38DF3B0D4E9EE1）
    + SHA256SUMS-v2.19.0.txt；发布文档已更新。
  - 📋 留待后续：D3-4 显存压力专项报告的实测模板校验（工具已齐）。
- 2026-08-14（阶段 4 个人使用收口 v2.20.0，用户决策「不商用」）：
  - ✅ 阶段 4 重新定标：删除安装器/自动更新/代码签名（SAC/SmartScreen）等商用项；聚焦
    数据可迁移（P4-3）、模块收口（P4-1）、磁盘治理（P4-4）；
  - ✅ P4-3 数据可迁移：`internal/gaea/backup/backup.go`（Plan/Manifest/Create/Extract/
    WritePending/ApplyPending/ClearPending，SQLite VACUUM INTO 一致性快照，zip-slip 防穿越）
    + `internal/app/gaea_data_backup.go`（GaeaDataBackupInfo/Create/Restore/Pending/Cancel/
    RestoreResult + Startup applyPendingRestore 钩子，恢复前先自动备份当前数据）；
    设置页新增「数据」分类 `components/settings/DataPanel.tsx`（备份清单/一键备份/从备份恢复/
    待应用告警与取消/恢复结果提示）；
  - ✅ P4-1 模块收口：微信通道标注 beta（CharacterLibEditor）、移动端标注「已冻结」（SettingsMobile）；
  - ✅ P4-4 磁盘治理：releases 清理（删除 24 个旧 exe，保留最近 5 版）+ README 约定；
  - ✅ 验证：Go backup 包 6 测试 + App 层 3 测试 + internal/app 全量 ok（29.97s）、vet 干净、
    gen_bindings 重新生成（442 方法 → 10 门面）+ 完备性测试、tsc/eslint 0 errors、
    vitest 251/251（61 文件，DataPanel 4 组 + SettingsPage 更新）；
  - 🔍 独立子代理代码审查（data_backup_review.md）：发现 3 高危 + 4 中危 + 9 低危；
  - ✅ v2.20.1 审查修复（2026-08-14）：#1 ApplyPending 两阶段幂等（重试可成功）、#2 home 配置恢复
    （修复 HomeConfigRel 缺 . 前缀根因）、#3 快照 busy_timeout + checkpoint 回退 + manifest Warnings、
    #4 失败写 result、#5 已有 pending 拒绝 + staging 随机后缀 + 孤儿清理、#6 dirSize 缓存、
    #7 GaeaDataBackupRollback + 前端回滚按钮、#8~#17 低危（盘符拒绝/二次确认/文件名防覆盖/before 保留 2 份/
    原子写/两阶段解压/shouldSkip 精确化/错误透出/entries 校验）；
    backup 包 9 测试 + App 层 5 测试、go 全量 ok、tsc/eslint 0 errors、vitest 251/251、
    真实 E2E（第二次恢复拒绝、重启自动应用、home 配置恢复）；发布 gaea-v2.20.1.exe
    （SHA256=F7347E70...）。
