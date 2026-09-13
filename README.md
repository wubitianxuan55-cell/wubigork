# gaea — 多功能 AI 助手

> 盖亚：本机上的通用办公与日常伙伴。对话、轻语、小说、绘梦、模型、办公、微信，共用一套引擎。

![gaea](frontend/public/favicon.svg)

## 功能

- **对话** — 流式输出、话题管理、本地存储；回答可内嵌 GenUI（卡片/表/图/表单/测验）
- **轻语** — 可定制人格的陪伴（模板 + 情绪 + 记忆）
- **小说** — 设定 / 角色 / 大纲 / 章节流式创作，导出 TXT、MD、EPUB
- **绘梦** — 文生图（ComfyUI：Flux / Z-Image-Turbo / Krea2）
- **模型中心** — 多供应商（xAI / DeepSeek / 智谱等），OAuth 登录，功能域绑定
- **通用办公** — 文档转换与编辑、知识库、计划卡片、产物面板、工作区搜索、做梦整理记忆
- **造价数据库** — 综合单价 / 人材机、测算版本、询价与组价；测算走办公 agent，库本身只存数
- **微信助手**（beta）— 扫码绑定，远程对话；凭证失效不自动恢复

## 技术栈

| 层 | 技术 |
|---|------|
| 桌面 | Wails v2 |
| 后端 | Go |
| 前端 | React + TypeScript + Ant Design + Vite |
| 模型 | 云端多供应商 + 本机（Herdsman / Ollama 等） |
| 图像 | ComfyUI |

## 快速开始

```bash
cd frontend && npm install
wails dev                 # 热加载
wails build               # 产物 build/bin/gaea.exe
```

网页对齐桌面：设 `GAEA_HTTP_PORT=8080` 后启动，浏览器打开 Vite（`http://localhost:5173`），`/api` 代理到同一内核，事件走 `/api/stream` SSE。命令行登录：`gaea login`。

## 项目结构

```
gaea/
├── main.go / wails.json
├── internal/app          # 绑定与各模块 handler
├── internal/gaea         # 办公引擎（含 cost / costproject / costref）
├── internal/whisper      # 轻语
├── internal/modelengine  # 模型中心
├── frontend/src/pages    # 各板块页
├── frontend/src/gaea     # 办公 UI
├── prompts/  skills/  docs/
└── CHANGELOG.md          # 完整版本磁带
```

## 版本

当前 **v4.282.0**。

逐条演进只写一份：[CHANGELOG.md](./CHANGELOG.md)。单版说明在 [releases/](./releases/)。这里不再抄 patch 年表。

发版约定：产品号只在一批用户能感到的能力或修复落地时动。文档整理、令牌对照、单点样式等小改记入 CHANGELOG，**不单独抬版本**。

| 代 | 大致范围 |
|---|----------|
| v4 | 书斋/闲庭、小说场景制、造价组价、办公管家、瘦身与绑定面 |
| v3 | 星枢壳层、小说阅读、角色库闭环 |
| v2 | 通用办公闭环、统一角色库、HTTP 调试桥 |
| v1.0.0 | 品牌更名为 gaea |

## 许可

私有软件，保留所有权利（见 [LICENSE](./LICENSE)）。定位为个人工具（2026-09-08 拍板：路线 A，产品化不作承诺）。
