# wubigork 设计文档

> 基于 xAI Grok 的桌面端小说创作 Agent

**版本**: v1.0  
**日期**: 2026-06-19  
**状态**: 设计阶段

---

## 1. 产品概述

wubigork 是一款桌面端小说创作工具，由一个主 Agent 和四个专用子代理协同工作，通过 xAI Grok 模型辅助作者完成从世界观构建到全本导出的完整创作流程。

## 2. 技术栈

| 层 | 技术 |
|---|------|
| 桌面框架 | Wails v3 |
| 后端 | Go 1.26+ |
| 前端 | React 18 + TypeScript + Ant Design 5 + Vite |
| AI 引擎 | xAI Grok（SuperGrok OAuth PKCE） |
| 向量存储 | ChromaDB（Go 客户端或嵌入式） |
| 存储 | 一部小说一个文件夹，JSON + Markdown |
| 导出 | Markdown / TXT / EPUB |

## 3. 项目结构

### 3.1 代码仓库

```
wubigork/
├── main.go                     ← Wails 入口
├── go.mod
├── wails.json                  ← Wails 配置
├── internal/                   ← Go 后端
│   ├── app/
│   │   └── app.go              ← Wails 生命周期 + 依赖注入
│   ├── project/
│   │   └── project.go          ← 项目管理 CRUD
│   ├── worldview/
│   │   └── worldview.go        ← 世界观 Agent
│   ├── character/
│   │   └── character.go        ← 角色 Agent
│   ├── outline/
│   │   └── outline.go          ← 大纲 Agent
│   ├── chapter/
│   │   └── chapter.go          ← 章节创作 Agent
│   ├── analysis/
│   │   └── analysis.go         ← 情节分析 Agent
│   ├── skill/
│   │   └── skill.go            ← Skill 加载/缓存/注入
│   ├── prompt/
│   │   └── prompt.go           ← RTCO Prompt 模板引擎
│   ├── memory/
│   │   └── memory.go           ← 语义记忆（ChromaDB）
│   ├── ai/
│   │   ├── client.go           ← HTTP 客户端 + SSE 流式
│   │   ├── provider.go         ← 消息构建 + tool_call
│   │   └── types.go            ← 通用类型
│   ├── auth/                   ← 已有
│   │   ├── discovery.go        ← OIDC Discovery
│   │   ├── oauth.go            ← PKCE loopback
│   │   └── token.go            ← Token 持久化
│   └── config/
│       └── config.go           ← 全局配置
├── frontend/                   ← React 前端
│   ├── src/
│   │   ├── App.tsx
│   │   ├── layouts/
│   │   │   └── MainLayout.tsx  ← 全局布局（侧栏+面板）
│   │   ├── pages/
│   │   │   ├── HomePage.tsx    ← 项目列表/新建
│   │   │   ├── WorldviewPage.tsx
│   │   │   ├── CharacterPage.tsx
│   │   │   ├── OutlinePage.tsx
│   │   │   ├── ChapterPage.tsx ← 章节创作（主界面）
│   │   │   ├── ExportPage.tsx
│   │   │   └── SettingsPage.tsx
│   │   ├── components/
│   │   │   ├── ChatPanel.tsx   ← AI 对话面板（复用）
│   │   │   ├── StreamingText.tsx
│   │   │   ├── OutlineTree.tsx
│   │   │   ├── CharacterCard.tsx
│   │   │   └── ...
│   │   ├── stores/            ← Zustand 状态管理
│   │   └── wails/             ← Wails 生成的 Go 绑定
│   └── package.json
├── skills/                     ← 全局内置 Skill
│   ├── story-long-write/
│   │   └── SKILL.md
│   └── story-deslop/
│       └── SKILL.md
├── prompts/                    ← 全局默认 Prompt 模板
│   ├── worldview.json
│   ├── character-batch.json
│   ├── outline-create.json
│   ├── chapter-generate.json
│   └── analysis.json
└── docs/
    └── superpowers/
        ├── specs/
        └── plans/
```

### 3.2 小说项目目录

```
MyNovel/                       ← 一部小说 = 一个文件夹
├── project.json               ← 元信息
├── worldview.md               ← 世界观（自由文本，Agent 对话迭代）
├── careers.json               ← 职业体系
├── characters.json            ← 角色 & 组织 & 关系网
├── outline.json               ← 大纲树（章节计划）
├── chapters/                  ← 章节正文
│   ├── 001.md
│   ├── 001-analysis.json      ← 章节分析结果
│   ├── 002.md
│   └── ...
├── foreshadows.json           ← 伏笔追踪
├── prompts/                   ← 项目自定义 prompt
├── skills/                    ← 本项目启用的 skill
└── memories/                  ← ChromaDB 向量库
```

## 4. Agent 架构

### 4.1 主 Agent

`ChapterAgent` — 章节创作。读所有上下文（世界观、角色、大纲节点、伏笔、记忆），流式生成正文。

### 4.2 四个子代理

每个子代理是独立的 AI 对话会话，维护自己的 context window，可被用户随时唤醒：

| Agent | 文件 | 职责 |
|-------|------|------|
| `WorldviewAgent` | `worldview.md` | 世界观迭代构建、规则一致性校验、冲突检测 |
| `CharacterAgent` | `characters.json` | 角色/组织 CRUD、关系网维护、角色弧光追踪、状态演进 |
| `OutlineAgent` | `outline.json` | 大纲树规划、续写、展开/合并、伏笔位置建议 |
| `AnalysisAgent` | `chapters/*-analysis.json` | 9 维度情节分析、质量评分、改进建议 |

### 4.3 跨 Agent 通信

所有 Agent 共享 `ProjectContext`：

```go
type ProjectContext struct {
    Project     ProjectMeta      // project.json
    Worldview   string           // worldview.md 全文
    Characters  []Character      // characters.json
    Outlines    []OutlineNode    // outline.json
    Foreshadows []Foreshadow     // foreshadows.json
    Memories    []MemoryResult   // 语义检索结果
    RecentChapters []ChapterSummary // 最近 N 章摘要
}
```

子代理修改文件后，主 Agent 下次调用自动读最新版本。

## 5. Prompt 体系（RTCO 框架）

所有 prompt 模板统一结构：

```json
{
  "name": "chapter-generate",
  "system": "你是一位专业小说作家...",
  "task": "根据以下大纲节点创作一章内容",
  "input_sections": {
    "worldview": { "priority": "P1", "label": "世界观" },
    "characters": { "priority": "P1", "label": "角色" },
    "outline_node": { "priority": "P0", "label": "当前大纲" },
    "prev_chapter": { "priority": "P0", "label": "上一章正文+摘要" },
    "foreshadows": { "priority": "P1", "label": "伏笔状态" },
    "memories": { "priority": "P2", "label": "相关历史" }
  },
  "output": { "format": "markdown", "description": "..." },
  "constraints": {
    "must": ["保持角色性格一致", "推进大纲情节点"],
    "forbidden": ["总之", "综上所述", "此外", "此章到此"]
  }
}
```

## 6. Skill 系统

- **格式**: `SKILL.md`（YAML frontmatter + Markdown body）
- **位置**: 全局 `skills/` + 项目 `skills/`
- **注入**: 章节生成时，选中 Skill 的 body 追加到 system prompt
- **管理**: 前端提供 Skill 列表、启用/停用、编辑

## 7. 数据流

```
用户指令 → Wails Bridge → Go Handler → Agent.Run()
    │
    ├── 1. 加载 ProjectContext（读文件）
    ├── 2. 构建 Prompt（RTCO 模板 + Skill 注入）
    ├── 3. 调用 xAI API（SSE 流式）
    │        ├── 流式 chunk → Wails Event → 前端 StreamingText
    │        └── 完整响应 → 写入章节文件
    ├── 4. 自动触发分析（可选）
    └── 5. 更新伏笔/角色状态
```

## 8. 创作完整工作流

```
📂 新建/打开项目
 │
 ├─ 🌍 世界观 Agent
 │     AI 对话式构建 → 编辑 → 一致性校验
 │
 ├─ 👤 角色 Agent
 │     AI 批量生成 + 单角色微调 → 关系网
 │
 ├─ 📋 大纲 Agent
 │     规划章节树 → 展开 → 伏笔建议
 │
 ├─ ✍️ 章节创作（主界面）
 │     ├─ 选大纲节点
 │     ├─ 选 Skill
 │     ├─ AI 流式生成
 │     ├─ 📊 自动分析
 │     └─ 确认 / 编辑 / 重写
 │
 └─ 📖 导出
```

## 9. 前端布局（三栏）

```
┌──────────┬───────────────────────┬──────────┐
│          │                       │          │
│   大纲树  │     章节正文（编辑）    │  AI 对话  │
│          │                       │  (Chat    │
│  章节列表 │     Markdown 渲染     │  Panel)  │
│          │                       │          │
│  切换节点 │     流式实时更新       │  子代理   │
│          │                       │  切换入口 │
├──────────┴───────────────────────┴──────────┤
│  状态栏：当前项目 / Agent / Token用法         │
└──────────────────────────────────────────────┘
```

## 10. 关键设计原则

1. **文件即真相** — 所有数据源是文件，不搞缓存与文件不同步
2. **Agent 独立性** — 每个子代理有自己的对话历史，互不污染 context
3. **流式优先** — 所有 AI 调用走 SSE，用户实时看到生成过程
4. **去 AI 味** — 内置 anti-AI 约束，Skill 可叠加
5. **单二进制分发** — Wails 编译，用户只需一个 `.exe`
