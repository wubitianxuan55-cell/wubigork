# ✍️ wubigork

> 基于 xAI Grok 的桌面端小说创作 Agent

## 功能

- 🔑 **SuperGrok OAuth** — PKCE 一键登录，token 持久化
- 🌍 **世界观 Agent** — AI 对话式构建和迭代世界观
- 👤 **角色 Agent** — 批量生成主角/配角/反派，自动维护关系网
- 📋 **大纲 Agent** — AI 规划章节树、续写、展开子节点
- ✍️ **章节创作** — SSE 流式生成，实时逐字显示，右侧终端面板监控
- 📊 **自动分析** — 9 维度情节分析 + 伏笔追踪 + 角色状态同步
- 🎯 **Skill 系统** — 可插拔写作指导（长篇网文 / 去 AI 味）
- 📖 **一键导出** — TXT + Markdown + EPUB
- 🚀 **快速引导** — 导入参考素材，AI 自动生成世界观+角色+大纲

## 技术栈

| 层 | 技术 |
|---|------|
| 桌面框架 | Wails v2 |
| 后端 | Go 1.26 |
| 前端 | React 18 + TypeScript + Ant Design 5 + Vite |
| AI 引擎 | xAI Grok (SuperGrok OAuth PKCE) |

## 快速开始

```bash
# 安装依赖
cd frontend && npm install

# 开发模式（热加载）
wails dev

# 命令行登录
wubigork login

# 生产构建
wails build
```

## 项目结构

```
wubigork/
├── main.go                    # Wails 入口
├── wails.json                 # Wails 配置
├── internal/
│   ├── app/
│   │   ├── app.go             # 结构体 + 生命周期 + 并发访问器
│   │   ├── auth_handler.go    # OAuth 登录绑定
│   │   ├── project_handler.go # 项目 CRUD 绑定
│   │   ├── worldview_handler.go   # 世界观绑定
│   │   ├── character_handler.go   # 角色绑定
│   │   ├── outline_handler.go     # 大纲绑定
│   │   ├── chapter_handler.go     # 章节绑定
│   │   ├── analysis_handler.go    # 分析绑定
│   │   ├── export_handler.go      # 导出绑定
│   │   ├── search_handler.go      # 搜索绑定
│   │   ├── stats_handler.go       # Skill/统计/配置绑定
│   │   └── shelf.go           # 书架绑定
│   ├── ai/                    # AI 调用层 (SSE 流式)
│   ├── auth/                  # OAuth PKCE + token
│   ├── project/               # 项目管理 CRUD
│   ├── worldview/             # 世界观 Agent
│   ├── character/             # 角色 Agent
│   ├── outline/               # 大纲 Agent
│   ├── chapter/               # 章节创作 Agent
│   ├── analysis/              # 情节分析 + 伏笔
│   ├── export/                # TXT/MD/EPUB 导出
│   ├── search/                # 全文搜索 + 版本备份
│   ├── stats/                 # 统计仪表盘
│   ├── skill/                 # Skill 加载引擎
│   ├── prompt/                # RTCO 模板引擎
│   ├── types/                 # 数据模型
│   └── config/                # 全局配置
├── frontend/                  # React 前端
│   └── src/
│       ├── layouts/           # 三栏布局 + 终端面板
│       ├── pages/             # 7 个页面
│       ├── components/        # ChatPanel 等
│       └── stores/            # Zustand 状态
├── prompts/                   # RTCO Prompt 模板 (16 个)
├── skills/                    # 内置 Skill (2 个)
└── docs/superpowers/          # 设计文档 + 实现计划
```

## 版本

| 版本 | 日期 | 说明 |
|------|------|------|
| **v4.0.0** | 2026-06-29 | 「织梦者」AI原生叙事工作室：场景引擎+AI协写+知识图谱+上下文智能+视觉叙事+生态平台 |
| v3.2.0 | 2026-06-26 | — |
| v3.1.0 | 2026-06-21 | Layered Glass 视觉重设计 + TTS 语音朗读 |
