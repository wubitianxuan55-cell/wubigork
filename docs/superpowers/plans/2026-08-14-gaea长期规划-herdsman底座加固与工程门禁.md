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

### 阶段 4 · 产品化收口（按需）

| 编号 | 任务 | 内容 |
|---|---|---|
| P4-1 | 模块收口 | 微信助手标注 beta 或冻结（gateway 现状：4 绑定中 2 凭证失效）；移动端冻结 |
| P4-2 | 发布体验 | 安装器（NSIS/Inno）+ 自动更新 + 代码签名（SAC/SmartScreen） |
| P4-3 | 数据可迁移 | 记忆/知识/成本库备份与导入导出 |
| P4-4 | 磁盘治理 | 模型库显示已装总量/可用磁盘；releases 目录清理（README 约定保留最近 5 版） |

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
  - ⏳ v2.16.1 构建产物（`cmd /c build.bat`）与冒烟（`scripts\smoke.ps1`）移交本机。
