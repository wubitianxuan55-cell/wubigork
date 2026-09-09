# gaea 30 分钟上手（2026-09 定稿）

> 写给新接手者：从零到「能安全改代码并发布」的 30 分钟路线。权威细节看 AGENTS.md 与
> docs/gaea-slim-masterplan-2026-09.md；本文只讲**每天要用的动作**。

## 0. 这是什么

gaea 是 Windows 桌面「通用办公」AI 助手：文档撰写/表格处理/格式转换/图表生成/知识库，
Wails v2（Go 1.26 后端 + React 19/TS/Vite 前端）。前端全部经 `gaea/lib/bridge` 代理调
后端绑定（旧 `wailsjsCompat.ts` shim 已于 v4.174 退役，全项目零引用）。

```
┌────────────────────┐   ┌──────────────────────────────┐
│ frontend/src        │   │ internal/                    │
│  React + Vite + TS  │──▶│  app(绑定面10门面)/core/...   │
│  gaea/lib/bridge    │   │  office/config/modelengine   │
└────────────────────┘   └──────────────────────────────┘
        ?mock=1 浏览器 dev 回退到 gaea/lib/mock（无 Go 也可开发 UI）
```

## 1. 五分钟：跑起来

前置：Go 1.26、Node 20+、npm 11、LibreOffice（文档转换）、Python 3.13（办公脚本）、
本地 herdsman（http://localhost:8080，重 AI 功能才需要）。

```bash
# 首次
cd frontend && npm ci          # package-lock 已入库，勿改依赖版本裸跑
cd .. && go mod download

# 开发：浏览器 mock 模式（不需要 Go 后端，最快看 UI）
cd frontend && npm run dev     # 打开 http://localhost:5173?mock=1

# 桌面调试（热加载前后端）
cd frontend && npm run dev     # 终端A 保持
# 终端B：go run . 或 wails dev  （本机建议 build.bat 前先看下方「构建」节）
```

## 2. 十分钟：跑测试与门禁（发布前必过）

```bash
# 前端全量
cd frontend && npx tsc -b && npx eslint . && npx vitest run
# 后端全量（逐包，避免单进程树被终止）
.\scripts\test-all.ps1
# 契约与快捷键
.\scripts\check-bindings-drift.ps1   # 绑定面 602 与前端清单一致性
node .\scripts\slim-locale-deadkeys.mjs   # 三语字典死键检查
```

**门禁清单（每刀必全绿）**：Go 128 包 / vitest 2737（313 文件）/ tsc 0 / eslint 0 /
drift PASS@602 / locale 0 死 / build + smoke 200。

**已知 flaky（勿把首跑红当回归）**：vitest 全量并发偶发负载假红（TrajectoryView 等超时）
——单文件复跑全绿即 OK；Go 偶发 TempDir 清理竞态。

## 3. 十五分钟：改代码的工作流（仓库纪律）

**默认并发子代理**（用户强化习惯）：≥2 条独立线就拆线并发，主代理定契约+收口。

1. **拆线**：列出「线 × 文件足迹」，线间文件互斥；契约类文件（bridge/types/mock/三语字典）
   单一负责人，生成动作主代理统一执行。
2. **改**：小步、行为零变化优先；新下载出口走 `saveExportBlob`、新选取走 `pickFile`、
   新 diff 用 `lib/diff`、新 slug 用 `strutil.TitleSlug`、新 b64 解码用 `b64ToBytes`。
3. **测**：改前端跑相关 vitest；改 Go 跑相关包测试；新功能补用例。
4. **全量门禁**（上方第 2 节）+ 构建冒烟。
5. **发布**（严格流程，见下方第 4 节）。

**测试 mock 关键**（异步 bridge 时代）：mock 是异步 chunk（v4.176 起）——测试若同步断言
window 钩子或 en 译文，先 `await waitMockReady()` 或 `await loadLocale('en')`。

## 4. 发布流程（20 分钟，不可跳步）

```bash
# 1) 版本号三处
.\scripts\sync-version.ps1 -Version 4.X.Y.0   # app_info.go/wails.json/versioninfo.rc

# 2) 构建 + 冒烟（build.bat 已含 strip 与冒烟；发布不得 skip-smoke）
#    沙箱环境：先前端再 wails（wails 捕获前端输出会挂起）
cd frontend && npm run build && cd .. && wails build -s

# 3) 复制产物 + SHA256
Copy-Item build/bin/gaea.exe releases/gaea-v4.X.Y.0.exe
(Get-FileHash releases/gaea-v4.X.Y.0.exe -Algorithm SHA256).Hash | `
  Set-Content releases/SHA256SUMS-v4.X.Y.0.txt

# 4) 冒烟
.\scripts\smoke.ps1 -ExePath releases\gaea-v4.X.Y.0.exe

# 5) 文档：releases/v4.X.Y.0.md + CHANGELOG（末尾追加）+ README/releases 版本表
# 6) git commit + tag v4.X.Y.0；回写 .gaea/{AGENTS,progress,todos}.md
```

发布说明模板：标题「瘦身 P…」/ 各线改动 / 门禁数字 / SHA256 / 欠账清单（每版必列）。

## 5. 常见任务速查

| 想做什么 | 去哪 |
|---|---|
| 看版本线 | releases/ + README 版本表 + CHANGELOG |
| 改办公文档（docx/xlsx/pptx） | internal/office/*（docxedit/xlsxedit/pptxedit）+ frontend 预览面板 |
| 改进度计划（CPM/甘特/双代号） | internal/schedule + frontend/src/schedule |
| 改记忆/知识库 | internal/app + gaea/components/memoryhub、knowledge/ |
| 加后端绑定 | internal/app/bindings_*.go → `go run ./scripts/gen_bindings` → bridge 分域接口 + mock |
| 加前端页面 | main.tsx registerPage(key, lazy(...)) + boards/manifests 注册 |
| 改本地模型链路 | internal/modelengine（engine_*.go 拆分域）+ modelcenter 页 |
| 看架构决策 | docs/gaea-nextgen-roadmap-2026.md、docs/gaea-slim-masterplan-2026-09.md |
| 查历史版本详情 | releases/v<版本>.md + .gaea/progress.md（v4.170 前：docs/archive/progress-history-2026-09.md） |

## 6. 三句铁律

1. **行为零变化**是结构类改动的红线（拆分/迁移/重构都逐字节对账）。
2. **门禁假红一次就教人无视红灯**——flaky 要单跑复跑确认，真红必须根因。
3. **发布前 必过 冒烟**，发布后 必回写 `.gaea/` 记忆。