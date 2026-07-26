# wubigork v4.0 — 划时代革新蓝图

> **代号**: 「织梦者」Dreamweaver
> **日期**: 2026-06-29
> **状态**: 蓝图阶段
> **对标**: Scrivener (结构) + Sudowrite (AI 工作流) + Notion+NovelAI (数据+上下文) + Obsidian (知识图谱) + Cursor/Windsurf (AI 写作 UX)

---

## 0. 核心洞察：为什么需要 v4.0

### 现状诊断

| 维度 | v3.x 现状 | 根本问题 |
|------|-----------|----------|
| **写作单元** | 章节 = 一个 Markdown blob | 无法场景级管理、无法分块 AI 操作 |
| **AI 交互** | 点击按钮 → 全章生成 | 缺乏渐进式、可撤销、可组合的 AI 写作 |
| **数据模型** | 平铺 JSON 文件、子串搜索 | 无语义关联、无双向链接、无智能检索 |
| **上下文** | 全量注入 Grok 1M 窗口 | 缺乏智能压缩、优先级、触发式注入 |
| **版本** | 直接覆盖式保存 | 无快照、无 diff、无分支历史 |
| **生态** | 2 个内置 Skill | 无插件系统、无可扩展性 |
| **模型** | 仅 xAI Grok | 无模型选择、无本地模型 |
| **可视化** | 基本的 3D 关系图、时间线画布 | 缺乏交互式叙事可视化 |

### 革新方向

```
v3.x "AI 辅助写作工具"  ──→  v4.0 "AI 原生叙事工作室"
```

核心转变：从「人操作 AI 写」→「人机共创的叙事操作系统」— 一个集结构化管理、AI 协写、知识图谱、可视化叙事于一体的专业创作平台。

---

## 1. 架构革命

### 1.1 数据模型重设计：场景优先 (Scene-First)

```
旧模型：Project → Chapters (blob)
新模型：Project → Acts → Chapters → Scenes (原子单元)
```

```
MyNovel/
├── project.json                    # 元信息
├── style-profile.json              # 🆕 风格档案（AI 学习作者语气）
├── story-rules.md                  # 🆕 创作约束规则
├── worldview/
│   └── sections/                   # 🆕 世界构建（独立文件）
│       ├── era.md
│       ├── geography.md
│       ├── factions.md
│       ├── rules.md
│       ├── culture.md
│       └── history.md
├── characters/
│   ├── database.json               # 🆕 角色数据库（支持查询）
│   ├── profiles/                   # 🆕 每个角色一个 Markdown
│   │   ├── elara.md
│   │   └── kael.md
│   └── portraits/                  # 🆕 AI 生成角色图
├── outline.json                    # 保持卷-章-节树
├── chapters/
│   └── 001/
│       ├── manuscript.md           # 🆕 场景拼接视图
│       ├── scenes/                 # 🆕 原子单元
│       │   ├── 001-opening.md
│       │   ├── 002-confrontation.md
│       │   └── 003-cliffhanger.md
│       ├── summary.json
│       ├── analysis.json
│       └── snapshots/              # 🆕 自动快照
├── lorebook.json                   # 保留，增强注入逻辑
├── foreshadows.json
├── canvas.json                     # 🆕 视觉画布数据
├── versions/                       # 🆕 Git-like 版本树
│   └── ...
└── .wubigork/                      # 🆕 项目级配置
    ├── rules.md
    └── templates/
```

### 1.2 后端架构升级

```
internal/
├── app/                  # Wails 绑定层（不变）
├── scene/                # 🆕 场景管理器 — 原子级读写
├── chapter/              # 保留，改为场景编排
├── memory/               # 🆕 语义记忆 — ChromaDB / LanceDB 嵌入式
├── context/              # 🆕 智能上下文引擎 — 优先级+触发式注入
├── model/                # 🆕 多模型路由器
│   ├── router.go         #   自动选最优模型
│   ├── grok.go           #   xAI Grok
│   ├── claude.go         #   Anthropic Claude
│   ├── openai.go         #   OpenAI GPT
│   └── local.go          #   Ollama / LM Studio 本地
├── snapshot/             # 🆕 版本快照引擎
├── plugin/               # 🆕 插件加载器 (Go 插件 / WASM)
├── canvas/               # 🆕 画布数据管理
├── ... (其余不变)
```

### 1.3 前端架构升级

```
frontend/src/
├── pages/
│   ├── HomePage.tsx           # 保留，加书架 Dashboard 2.0
│   ├── WritePage.tsx          # 🆕 融合 ChapterPage — 场景级编辑
│   ├── CorkboardPage.tsx      # 🆕 Scrivener Corkboard — 卡片规划
│   ├── CanvasPage.tsx         # 重写 — Obsidian Canvas 风格
│   ├── GraphPage.tsx          # 🆕 故事知识图谱
│   ├── WorldviewPage.tsx      # 保留，增强
│   ├── CharacterPage.tsx      # 保留，数据库化
│   ├── OutlinePage.tsx        # 保留，增强
│   ├── ExportPage.tsx         # 保留，升级
│   └── SettingsPage.tsx       # 保留，增强
├── components/
│   ├── editor/                # 🆕 编辑器组件包
│   │   ├── SceneEditor.tsx    #   场景级编辑器
│   │   ├── DiffReview.tsx     #   AI 修改 diff 审查
│   │   ├── GhostText.tsx      #   Tab 式内联补全
│   │   ├── CommandBar.tsx     #   Cmd+K 命令栏
│   │   └── ContextBuilder.tsx #   @-mention 上下文构建器
│   ├── canvas/                # 🆕 画布组件包
│   │   ├── InfiniteCanvas.tsx #   无限画布
│   │   ├── CanvasCard.tsx     #   场景/角色/地点卡片
│   │   ├── TimelineRail.tsx   #   时间线轨道
│   │   └── CanvasEdge.tsx     #   连接线
│   ├── graph/                 # 🆕 知识图谱
│   ├── agent/                 # 🆕 AI Agent 面板
│   └── ... (其余组件)
```

---

## 2. 六大革新模块

### 模块 A：场景引擎 (Scene Engine) — 对标 Scrivener

**核心理念**: 章节由场景组合而成，场景是创作的最小原子单元。

```
A1. 场景级编辑
    - 每个场景是独立 .md 文件，有自己的元数据
    - 场景元数据: POV角色、地点、时间、情感基调、字数、状态、标签
    - "Scrivenings" 模式: 选中多个场景 → 拼接为一个连续编辑视图
    - 场景可拖拽重排，重排后自动更新拼接视图

A2. 软木板 (Corkboard)
    - 每个场景 = 一张卡片（synopsis + 字数 + 状态标签）
    - 自由排列模式: 卡片拖放到任意位置
    - 彩色泳道: 每条泳道 = 一条故事线/POV线
    - 排列即编排: 在 Corkboard 上拖拽重排 = 重建章节结构

A3. 快照系统
    - 每次 AI 改写前自动创建快照
    - 快照存储 diff（不是全量副本）节省空间
    - 时间线滑块: 浏览场景的所有历史版本
    - 任意版本一键恢复
    - 双栏 Diff 对比（对标 Cursor inline diff）

A4. 元数据标签 + 智能收藏
    - 为场景打标签: "高潮", "情感戏", "动作", "悬念", "转场"
    - 创建 Saved Query: "所有 POV=Elara 且 标签=高潮 且 状态=Draft"
    - 保存为动态收藏集，实时更新
    - 收藏集可导出为独立阅读顺序
```

### 模块 B：AI 协写系统 (Co-Writer) — 对标 Cursor + Sudowrite

**核心理念**: AI 不是"替代"作者写，而是像 Cursor 一样"伴随"作者写。

```
B1. 内联补全 (Ghost Text)
    - 写作时，AI 实时建议下一句/下一段
    - Ghost text 以灰色斜体出现在光标后
    - Tab 接受、继续打字忽略
    - 支持中文长句补全
    - 可配置: 补全长度、频率、风格匹配度

B2. 命令编辑 (Cmd+K)
    - 选中任意文本 → Cmd+K → 输入自然语言指令
    - "用更紧张的节奏重写" / "加入感官描写" / "改为第一人称"
    - AI 原地编辑，预览为 diff
    - ⌘Y 接受 / ⌘N 拒绝
    - 预设指令列表: "丰富描写", "精简", "改变语气", "修复POV"

B3. Beat-to-Prose 管道
    - 阶段1: 定义 Beats (3-8个简短动作描述)
    - 阶段2: AI 按 Beat 逐个生成场景
    - 阶段3: 重排 Beat = 重写过渡段落
    - 两栏视图: 左 Beat 卡片 | 右 生成的 Prose
    - 每个 Beat 可独立锁定，不被后续重排影响

B4. 感官扩展器
    - 选中一段文字 → "Add Sensory Detail"
    - AI 返回5个维度的扩展建议:
        视觉/听觉/嗅觉/触觉/味觉
    - 每个维度独立开关，可组合应用
    - 一键合并入选的维度到原文

B5. 多模型路由
    - 不同任务自动选择最优模型:
        创意生成 → Grok (高创造力)
        文笔润色 → Claude (文笔细腻)
        结构化分析 → GPT (逻辑强)
        快速补全 → 本地小模型 (低延迟)
    - 用户可手动覆盖
    - 统一的 API 适配层
```

### 模块 C：叙事知识图谱 (Story Graph) — 对标 Obsidian + Notion

**核心理念**: 小说不是线性文本，而是一个互联的知识网络。

```
C1. 双向链接 [[wiki-style]]
    - 编辑器内 `[[角色名]]` 创建链接
    - 角色/地点/事件/概念都是可链接实体
    - 反向链接面板: 点击任意实体 → 看到所有提及它的场景
    - "未链接提及"检测: AI 扫描全文，自动发现应该链接的实体

C2. 知识图谱可视化
    - 节点类型: 场景、角色、地点、事件、物品、概念
    - 边类型: 出场、提及、影响、因果、拥有、位于
    - 局部图: "当前场景连接了什么"
    - 全局图: "整部小说的关系网络"
    - 聚类着色: 按章节/故事线/角色社群自动分组
    - 时间维度: 拖拽时间轴看图谱演化

C3. 实体数据库
    - 角色/地点/物品/事件 各自是结构化数据库
    - 多视图: Table / Kanban(状态) / Gallery(头像) / Timeline
    - 关联字段: 场景 ↔ 角色(出场) / 地点 ↔ 场景(发生地)
    - Rollup 聚合: "Elara 出场场景数"、"青云宗 涉及章节"

C4. 一致性守护
    - AI 自动检测: "Elara 第3章蓝眼睛，第7章绿眼睛"
    - 角色状态追踪: 每章后自动更新角色当前状态
    - 时间线校验: 事件是否按正确时间顺序发生
    - 伏笔追踪网络: 伏笔-揭示的关联图
```

### 模块 D：上下文智能引擎 (Context Engine) — 对标 NovelAI + Cursor

**核心理念**: 不再全量注入，而是智能裁剪、优先级分层、触发式注入。

```
D1. Lorebook 2.0 — 触发式上下文注入
    - 一个 Lorebook 条目 = {触发词, 内容, 优先级, 最近性权重}
    - 当触发词在上下文中出现 → 自动注入条目内容
    - 支持: 正则触发、层级触发（上级触发连带子级）
    - Token 预算管理: 可视化展示使用了多少 token
        ┌─────────────────────────────────┐
        │ Context Budget: 128K / 1M       │
        │ ████████████░░░░░░░░░░░░ 13%    │
        │                                 │
        │ System Prompt    ████ 12K       │
        │ Story Rules      ██ 2K          │
        │ Current Scene    ██████ 18K     │
        │ Prev Scenes      ████ 12K       │
        │ Character Cards  ████████ 24K   │
        │ Lorebook Inject. ███ 8K         │
        │ Story Memory     ████ 10K       │
        │ Semantic Results ██████ 16K     │
        │ Free Space       ░░░░░░░░░ 900K │
        └─────────────────────────────────┘

D2. 语义记忆检索
    - 嵌入式 ChromaDB / LanceDB（无外部依赖）
    - 每章生成后自动向量化关键片段
    - 写作时自动检索"与当前内容最相关的历史记忆"
    - 检索结果按相关性排序注入上下文
    - 支持: 按章节/角色/情感过滤检索

D3. 自适应压缩
    - 超过 token 预算时，自动压缩低优先级内容:
        P0 (必须保留): 当前大纲、当前场景、关键角色
        P1 (摘要): 前文 → 自动生成渐进式摘要
        P2 (压缩): 历史记忆 → 保留标题+关键词
    - 压缩度可配置（激进/平衡/保守）

D4. @-mention 上下文构建器
    - 输入 @ → 弹出实体选择器
    - @Elara @青云宗 @第三章 @research/武侠设定
    - 每个 mention 显示 token 消耗
    - 拖拽排序优先级
    - 保存为"上下文预设"可复用
```

### 模块 E：视觉叙事工作台 (Visual Storytelling) — 对标 Obsidian Canvas + Scrivener Corkboard

**核心理念**: 用空间化、可视化的方式规划和审视你的故事。

```
E1. 无限画布 (Infinite Canvas)
    - 基于 JSON Canvas 开放格式
    - 拖入: 场景卡片、角色卡片、地点卡片、参考图片、PDF、网页
    - 连线: 带标签的彩色连线（因果、POV转移、伏笔呼应）
    - 分组框: 用彩色包围框代表幕/章
    - 嵌套: 画布内可嵌入子画布

E2. 故事时间线
    - 水平滚动时间线
    - 轨道: 主线 / 子线A / 子线B / 角色弧光
    - 每个场景卡片在对应位置
    - 卡片颜色 = 情感基调（红=紧张, 蓝=平静, 绿=温馨）
    - 卡片高度 = 场景重要性/字数
    - 缩放: 鸟瞰全局 / 聚焦单章

E3. 章节情绪曲线
    - 自动分析每章的情绪变化
    - 可视化折线图: X=章节进度 / Y=紧张度
    - 叠加多章对比
    - 标注关键事件点
    - AI 建议: "第5-7章情绪过于平缓，建议增加冲突"

E4. 角色弧光热力图
    - 矩阵图: 行=角色 / 列=章节
    - 颜色深度 = 角色在该章的重要性
    - 点击单元格 = 看到角色在本章的状态
    - AI 分析: 角色出现频率、POV 分布是否合理
```

### 模块 F：平台化生态 (Platform) — 对标 Obsidian 插件 + Notion 集成

**核心理念**: wubigork 不是封闭工具，而是开放创作平台。

```
F1. 插件系统
    - 插件格式: WASM (安全沙箱) 或 JavaScript (受限)
    - 插件能力:
        - 添加新页面/面板
        - 注册 AI 动作（新的 Cmd+K 命令）
        - 注册导出格式
        - 读写项目文件（权限控制）
        - 注册新的数据视图
    - 插件市场: 在线浏览/安装/评价
    - 内置首批插件:
        - 写作马拉松计时器
        - 敏感内容检测
        - 投稿格式检查

F2. Skill 市场
    - 社区共享 Skill (写作风格指导)
    - 在线浏览: 按标签/类型/评分排序
    - 一键安装到本地
    - 评分和评论系统

F3. 多格式导入
    - Scrivener 项目 → wubigork
    - Markdown 文件夹 → wubigork 项目
    - Word (.docx) → 按标题自动分章
    - TXT → 智能分章

F4. 导出 2.0
    - 新增: PDF (专业排版)、DOCX (Word兼容)、HTML (网页阅读)
    - Compile 模板系统: 定义排版规则 → 一键应用
    - 预设模板: 网文投稿 / 出版投稿 / 自用阅读
    - 自定义模板: 字体/字号/行距/页眉页脚/目录

F5. 风格档案 (Style Profile)
    - AI 分析已有章节，学习作者的语气、节奏、用词习惯
    - 生成 "Style Profile" 配置
    - 所有 AI 生成都参考此档案
    - 多档案: 不同 POV 角色可有不同风格
    - 导出/导入: 分享你的风格档案

F6. 写作仪表盘 2.0
    - 实时统计: 今日字数 / 章节进度 / 全本进度
    - 目标设置: 日目标/章目标/截止日期
    - 写作日历: GitHub 贡献图风格
    - 趋势分析: 写作速度变化 / 创作高峰期
    - 里程碑成就系统
```

---

## 3. 实施路线图

### Phase 4.0 — 数据基础 (预计 2 周)

```
目标: 重构数据层，为所有革新提供地基

Task 4.0.1: 场景引擎后端
  - 新增 internal/scene/ 包
  - 场景 CRUD、元数据、文件读写
  - 章节 ↔ 场景映射层（向后兼容现有章节数据）
  - 场景拼接视图（Scrivenings 模式）

Task 4.0.2: 项目目录升级
  - 自动迁移 v3.x 项目到 v4 目录结构
  - 迁移脚本: 单章 blob → 单场景
  - 向后兼容: v4 能打开 v3 项目

Task 4.0.3: 快照引擎
  - 新增 internal/snapshot/ 包
  - 基于 diff 的高效存储
  - 快照时间线 API
```

### Phase 4.1 — AI 协写革命 (预计 3 周)

```
目标: 实现 Cursor 式的 AI 写作体验

Task 4.1.1: 内联补全 (Ghost Text)
  - 前端编辑器集成 ghost text 组件
  - 后端 SSE 流式补全 API
  - Tab 接受 / 继续输入忽略

Task 4.1.2: Cmd+K 命令编辑
  - CommandBar 组件
  - 选择文本 → 指令 → diff 预览 → 接受/拒绝
  - 预设指令库

Task 4.1.3: Inline Diff 审查
  - DiffReview 组件（绿色新增/红色删除）
  - 逐块接受/拒绝
  - 键盘快捷键

Task 4.1.4: Beat-to-Prose 管道
  - 两栏布局组件
  - Beat CRUD + 拖拽重排
  - 按 Beat 生成 prose
```

### Phase 4.2 — 知识图谱 (预计 2 周)

```
目标: 小说变为互联的知识网络

Task 4.2.1: 双向链接引擎
  - [[wiki-link]] 解析器
  - 反向链接面板
  - 未链接提及检测

Task 4.2.2: 实体数据库
  - 角色/地点/物品/事件 结构化存储
  - 关联字段
  - 多视图 (Table/Kanban/Gallery/Timeline)

Task 4.2.3: 知识图谱可视化
  - 2D force-directed 图（比3D更实用）
  - 节点/边类型系统
  - 局部图 + 全局图

Task 4.2.4: 一致性守护
  - 角色属性冲突检测
  - 时间线校验
```

### Phase 4.3 — 上下文智能 (预计 2 周)

```
目标: 聪明地管理 AI 上下文窗口

Task 4.3.1: 语义记忆 (ChromaDB / LanceDB)
  - 嵌入式向量数据库集成
  - 自动向量化 + 检索 API

Task 4.3.2: Lorebook 2.0
  - 触发式注入引擎
  - Token 预算可视化
  - 优先级 + 最近性权重

Task 4.3.3: @-mention 上下文构建器
  - 实体选择器 UI
  - Token 计数器
  - 上下文预设

Task 4.3.4: 多模型路由
  - 统一适配层 (Grok/Claude/GPT/Ollama)
  - 任务-模型自动匹配
```

### Phase 4.4 — 视觉叙事 (预计 2 周)

```
目标: 空间化、可视化的故事理解

Task 4.4.1: 无限画布
  - JSON Canvas 格式
  - 拖拽、连线、分组
  - 嵌入多种媒体

Task 4.4.2: 故事时间线
  - 水平滚动轨道
  - 场景卡片渲染
  - 鸟瞰 + 聚焦

Task 4.4.3: 情绪曲线 + 角色热力图
  - 自动分析 → 可视化
  - AI 建议
```

### Phase 4.5 — 生态平台 (预计 2 周)

```
目标: 从工具到平台

Task 4.5.1: 插件系统
  - WASM 沙箱运行时
  - 插件 API 定义
  - 首批内置插件

Task 4.5.2: 导出 2.0 + 导入
  - PDF/DOCX/HTML 导出
  - Compile 模板
  - Scrivener/Word 导入

Task 4.5.3: Skill 市场 + 风格档案
  - 在线 Skill 仓库
  - 风格分析 + 学习引擎
  - 风格档案导入/导出

Task 4.5.4: 写作仪表盘 2.0
  - 统计 / 目标 / 日历
  - 成就系统
```

### Phase 4.6 — 打磨与发布 (预计 1 周)

```
Task 4.6.1: 性能优化
Task 4.6.2: 全功能回归测试
Task 4.6.3: 迁移工具完善（v3 → v4）
Task 4.6.4: v4.0.0 发布 + 文档
```

**总预计**: 14 周 (约 3.5 个月)

---

## 4. 关键设计决策

### 决策 1: 为什么场景优先？

章节作为写作单元太粗粒度。场景是：
- 单一时间/地点/事件 → AI 理解更精准
- 可独立重写/重排/标注 → 编辑更灵活
- 元数据自然附着 → 检索/统计/可视化更精确

**向后兼容方案**: 导入 v3 项目时，一个章节自动拆为 1 个场景（保持数据完整）。

### 决策 2: 为什么嵌入式向量数据库而不是 ChromaDB 服务器？

- 用户不需要额外安装/启动服务
- LanceDB (Rust, Go binding) 或 sqlite-vec 更适合桌面端
- 零配置、零维护

### 决策 3: 为什么保留 Go + Wails 而不是 Electron？

- 单二进制分发（~15MB）是核心优势
- Go 并发模型非常适合多 Agent + SSE 流式
- 性能优势（启动速度、内存）对桌面端体验至关重要

### 决策 4: 为什么 WASM 插件而不是 JavaScript 插件？

- 安全沙箱（不能随意读写文件系统）
- 语言无关（Rust/Go/C/C++ → WASM）
- 性能可控

---

## 5. UX 设计原则

1. **渐进式 AI**: AI 是伴随的，不是替代的。默认行为是建议，不是覆盖。
2. **可撤销性**: 所有 AI 操作必须可撤销（快照 + diff）。
3. **上下文透明**: 用户始终能看到 AI「知道」什么（Context Budget 面板）。
4. **键盘优先**: 核心操作都有键盘快捷键。
5. **空间化思维**: 提供多种可视化方式理解故事结构。
6. **数据主权**: 所有数据存本地文件，云端同步是可选插件。
7. **向后兼容**: v4 能打开 v3 项目并自动迁移。

---

## 6. 风险与缓解

| 风险 | 概率 | 缓解 |
|------|------|------|
| 功能过多导致开发周期失控 | 中 | 严格按 Phase 排期，每 Phase 有独立可交付价值 |
| 内联补全延迟影响写作流畅度 | 高 | 本地小模型做快速补全，云端大模型做深度生成 |
| 向量数据库嵌入复杂 | 中 | 优先选用 sqlite-vec (SQLite 扩展)，与现有文件基础一致 |
| 场景迁移破坏 v3 数据 | 低 | 迁移脚本只读 v3、写 v4，原创数据不碰 |
| 插件系统安全风险 | 中 | WASM 沙箱 + 权限声明 + 审核机制 |

---

## 附录 A: 参考软件精华提炼

| 软件 | 借鉴的核心机制 | 应用到 wubigork |
|------|---------------|-----------------|
| **Scrivener** | Corkboard、Snapshots、Scrivenings、Collections、Compile | 场景卡片规划、自动快照、拼接编辑、动态收藏、模板导出 |
| **Sudowrite** | Story Engine 分阶段、Describe 感官、Beat 系统、Brainstorm Canvas | 引导式创作向导、感官扩展器、Beat-to-Prose、脑暴画布 |
| **Notion** | 数据库多视图、Relation+Rollup、Backlinks、Synced Blocks、AI Autofill | 实体数据库、关联聚合、双向链接、角色模板同步、AI 自动标注 |
| **NovelAI** | Lorebook 触发注入、Memory 系统、Token Prob UI、Custom Modules | Lorebook 2.0、自适应记忆、置信度着色、Style Profile |
| **Obsidian** | Graph View、Canvas、Bidirectional Links、Dataview、Templater | 知识图谱、无限画布、[[wiki-link]]、统计查询、模板引擎 |
| **Cursor** | Tab 补全、Cmd+K、Agent 模式、Inline Diff、@-mention、.cursorrules | Ghost Text、命令编辑、Writer's Agent、Diff 审查、上下文构建器、Story Rules |

## 附录 B: v3 → v4 迁移兼容性

```
v3 项目目录:
MyNovel/
├── project.json          → project.json (不变)
├── worldview.md          → worldview/sections/ (拆分为6个文件)
├── characters.json       → characters/database.json + profiles/*.md
├── outline.json          → outline.json (不变，增加场景引用)
├── chapters/
│   ├── 001.md            → chapters/001/scenes/001.md (单场景)
│   └── 001-summary.json  → chapters/001/summary.json (不变)
├── foreshadows.json      → foreshadows.json (不变)
└── lorebook.json         → lorebook.json (不变，增强注入)
```

迁移是**单向 + 非破坏性**的：
- v3 原始文件保留在 `_v3_backup/` 子目录
- 迁移后 v4 工作在新结构中
- 用户可随时回退到 v3 打开备份
