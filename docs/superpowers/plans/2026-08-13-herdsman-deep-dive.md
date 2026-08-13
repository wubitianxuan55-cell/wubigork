# Herdsman 深挖路线图（gaea × Herdsman 深度集成）

> 创建：2026-08-13。背景：gaea 已大量调用 Herdsman（对话/图片/OCR/文档/语音/检索），
> 但仅停留在「HTTP 网关 + 少量硬编码模型名」的浅层使用。盘点 `C:\Users\wubi\.herdsman`
> 后确认 Herdsman 实际是一整套本地 AI 基础设施：90 模型目录、模型生命周期 CLI/RPC、
> 启动参数记录、逐请求性能统计、数字生命记忆库。本规划分阶段把「使用」升级为「深挖」。

## 阶段划分

| 阶段 | 内容 | 状态 |
|------|------|------|
| P0 | 资产盘点：目录结构、CLI 命令、模型目录、launch_records、model_stats、digital-life、环境健康 | ✅ 已完成 |
| P1 | 模型中心 × Herdsman 模型库联动（只读）：90 模型目录 + 能力字段 + 安装/运行状态进模型中心 | ✅ 已完成（v2.15.2） |
| P2 | 模型生命周期管理：下载/启动/停止/卸载 + 本机启动参数预设（launch_records → UI 预设） | ✅ 已完成（v2.15.3） |
| P3 | 本地翻译工具（Hunyuan-MT / Hy-MT，chat/completions；`/v1/translations` 为语音翻译不适用） | ✅ 已完成（v2.15.4） |
| P4 | 检索升级（qwen3-embedding-4b / qwen3-reranker-4b，动态发现+bge 回退）+ 调用统计合并（model_stats/events.jsonl） | ✅ 已完成（v2.15.5） |
| P5 | 数字生命记忆联动（digital-life schema 只读对接：角色/关系/摘要/时间线/世界事件）+ 最近操作可见性（skill-operations.json） | ✅ 已完成（v2.15.6） |

## 关键事实（P0 盘点结论）

- Herdsman exe：`C:\Program Files\starwave\Herdsman\herdsman.exe`（v0.5.3），API 网关 8080（LAN 已开）。
- CLI：`herdsman.exe skill models/env/audio/image/monitor/screenshot/speech --json`，另有 `aicache`（Phison AICache）、`envfix`。
- 模型目录：`skill models list` 返回 90 个（12 装 / 78 未装），含 name/type/capabilities/installed/running/quantization/变体/大小；HTTP `/v1/models` 只有 id+status，目录必须走 CLI/RPC。
- 启动参数：`launch_records/*.json` 记录每台机器实测启动参数（context/cache/gpu_layers/no_mmap/cache_ram/mmproj…），可直接做成 UI 预设。
- 统计：`model_stats/events.jsonl` 逐请求 TTFT/TPS/tokens/变体。
- 记忆：`digital-life/life.sqlite3`（18GB）含 life_timeline_events 22 万条、life_state_commits 21 万条、world_events、characters、relationships、memory_summaries；`persona-os/state.json` 管理 personas/agents（含微信网关绑定）。

## P1 实施清单（本轮）

1. 后端 `App.HerdsmanModelCatalog()`：
   - `herdsmanExePath()`：`HERDSMAN_EXE` 环境变量 → `%ProgramFiles%\starwave\Herdsman\herdsman.exe` 默认路径。
   - 执行 `herdsman.exe skill models list --json`（25s 超时），解析 `result` 数组。
   - 结构化字段：name / 显示名（zh）/ type / capabilities / installed / running / status / run_status / quantization / parameter_count / file_size / llama_cpp_variants。
   - 汇总计数：total / installed / running；CLI 缺失或 Herdsman 未运行时返回可读错误（不阻塞模型中心）。
   - 纯解析函数 + 单测（内联 fixture 覆盖 已安装/运行中/未安装/多模态/翻译能力/变体）。
2. 前端模型中心新增「模型库」分类：
   - `api/engines.ts` 增加 `getHerdsmanCatalog()`（走 `window.go.app.App` 动态绑定，无需手改 wailsjs）。
   - `HerdsmanCatalogSection.tsx`：KPI（总数/已安装/运行中）+ 搜索 + 状态过滤 + 类型分组 + 模型卡片（名称/能力/状态/量化/大小/变体），复用 ui.tsx 组件与 mc-* 样式。
   - `Category` 增加 `'catalog'`，TABS 增加「模型库」。
3. 验证：`go test ./internal/app`（含新用例）、`tsc -b`、vitest（新增 section 用例）、`wails build`。

## 风险与注意

- CLI 依赖 Herdsman 桌面进程运行：进程未启动时 `skill models list` 返回错误，前端展示空态+引导提示，不报错崩溃。
- `herdsman.exe skill models list` 每次约 1–2s（RPC），不做启动期同步调用，仅在用户进入「模型库」分类时加载 + 手动刷新。
- P2 的下载/启动属于会改变本机状态的操作，需用户显式点击且以异步 operation 展示进度，不自动执行。
