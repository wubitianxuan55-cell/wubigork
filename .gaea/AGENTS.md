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

- 最新发布：v2.5.0（2026-08-08）「全界面科幻视觉重设计 · 绘梦图生图/文生视频 · 角色库模型绑定」
- 里程碑：设置/首页/聊天欢迎屏/小说/绘梦五板块统一玻璃 HUD 设计语言；
  绘梦新增图生图（ComfyUI 低 denoise）与文生视频（LTX-Video，GenerateMedia 绑定）；
  模型中心功能绑定实时同步 + 办公/聊天去重合并；角色库新增 LLM 绑定（func_characterlib_*）
  与剧照独立后端/模型（portrait_backend / portrait_model）；vision 技能切 Qwen3.6-35B-A3B
  并修复 PS 5.1 UTF-8 编码（BOM + .NET HttpClient）
- 已知注意：角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定；
  文生视频依赖本地 ComfyUI 安装 LTX-Video 模型
