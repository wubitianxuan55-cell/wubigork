# 🌍 gaea — 多功能 AI 助手

> gaea（盖亚，大地女神）——一个不断完善的本地 AI 助手。
> 对话、轻语、小说、绘梦、模型引擎、工程办公、微信助手，多模块共生，统一品牌 V1.0.0。

![gaea](frontend/public/favicon.svg)

## 功能

- 💬 **对话** — 通用 AI 对话，流式输出，话题管理，本地存储
- 🌸 **轻语** — 可定制人格的陪伴型 AI（29 人格模板 + 情绪融合 + LLM 记忆管线）
- 📚 **小说创作** — 世界观/角色/大纲 Agent，章节流式创作，一键导出 TXT/MD/EPUB
- 🎨 **绘梦** — 文生图（ComfyUI：Flux / Z-Image-Turbo / Krea2），模型后端可切换
- ⚙️ **模型引擎** — 多供应商模型中心（xAI / DeepSeek / MiMo），OAuth 一键登录
- 🏗️ **工程办公** — 土壤修复投标方案：47 个工程工具 + Hermes/Hephaestus 双模型 agent + 6 个工程技能
- 💬 **微信助手** — 扫码绑定，ClawBot 远程对话
- 🔑 **SuperGrok OAuth** — PKCE 一键登录，token 持久化

## 技术栈

| 层 | 技术 |
|---|------|
| 桌面框架 | Wails v2 |
| 后端 | Go 1.26 |
| 前端 | React + TypeScript + Ant Design + Vite |
| AI 引擎 | xAI Grok (SuperGrok OAuth PKCE) + 多模型中心 |
| 图像 | ComfyUI (Flux / Z-Image-Turbo / Krea2) |

## 快速开始

```bash
# 安装前端依赖
cd frontend && npm install

# 开发模式（热加载）
wails dev

# 命令行登录
gaea login

# 生产构建（产物 build/bin/gaea.exe）
wails build
```

## 项目结构

```
gaea/
├── main.go                    # Wails 入口
├── wails.json                 # Wails 配置
├── internal/
│   ├── app/                   # App 绑定 + 各模块 handler
│   ├── ai/                    # AI 调用层（SSE 流式 + 图片）
│   ├── auth/                  # OAuth PKCE + token
│   ├── gaea/                  # 办公引擎（agent/tool/control/skill/boot/knowledge）
│   ├── whisper/               # 轻语模块（人格/记忆/分发）
│   ├── modelengine/           # 模型中心
│   ├── weixin/                # 微信助手
│   ├── project/               # 项目管理
│   ├── worldview/ character/ outline/ chapter/   # 小说 Agents
│   ├── export/                # TXT/MD/EPUB 导出
│   └── config/                # 全局配置（~/.gaea_config.json）
├── frontend/                  # React 前端
│   └── src/
│       ├── pages/             # 对话/轻语/小说/绘梦/模型中心/办公/知识库
│       ├── gaea/              # 办公板块（gaeaW 原生 UI）
│       └── stores/            # Zustand 状态
├── prompts/                   # RTCO Prompt 模板
├── skills/                    # 内置 Skill
└── docs/                      # 设计文档
```

## 版本

| 版本 | 日期 | 说明 |
|------|------|------|
| **v1.0.0** | 2026-08-01 | 品牌重塑：wubigrok 正式更名 gaea，全量替换品牌名与 logo，版本重新起算 |
