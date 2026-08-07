# gaea 项目记忆

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf ↔ Markdown）、图表生成、报告拼装、
知识库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- 单模型架构：一个 executor 完成规划与执行，无独立规划者；任务/技能子代理走 `task` / `run_skill`
- 内置工具精简为 17 个核心工具（v2.4.3 起）：文件/命令、网络、任务、记忆/知识、技能、format_convert、chart_gen
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/office/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）

## 发布流程

1. 更新 CHANGELOG.md / README.md / wails.json（productVersion）
2. `cmd /c build.bat`（wails build，产物 build/bin/gaea.exe，同时复制到桌面）
3. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`
4. 写 `releases/v<版本>.md` 发布说明，更新 `releases/README.md` 版本表
5. 更新 `.gaea/progress.md` 进度记忆与本文件

## 版本状态

- 最新发布：v2.4.5（2026-08-08）「通用办公 · 文件预览 · 欢迎页重设计」，合并 v2.4.2~v2.4.5
- 里程碑：v2.4.2 通用办公改造 + ModelScope 文档技能；v2.4.3 内置工具 38→17；
  v2.4.4 文件预览重设计 + 右侧面板精简；v2.4.5 欢迎页重设计
