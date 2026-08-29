# 市场调研：办公板块文件交互体验（文件引用 / 预览 / 附件 / 产物），2026-08-16

> 目的：为 gaea 通用办公「文件交互体验」下一轮优化提供设计依据。承接此前
> 《对话内成果交付与文件交互》《中期编辑·后期输出与预览》《大型文件与上下文》
> 三份调研，本轮聚焦**人与文件之间的所有触点**：输入侧（@ 引用、拖拽、附件、
> 发送队列）、对话内（行内文件 chip、多文件预览）、输出侧（产物卡片、版本、
> 对比、下载）。来源为 2026 年公开资料、GitHub release/PR 与实测报道。

## 0. 一句话结论

2026 年头部办公 agent 已把「文件」从**附件/路径文本**升级为**一等公民交互对象**：
输入侧是彩色 @ 引用 chip + 多文件发送队列，对话内是行内文件 chip 点击即预览、
预览不离开 Chat（Unified Files Workspace），输出侧是带版本步进/对比/下载的产物卡片。
gaea 已具备链路底座（mdast 文件链接化、@ 菜单、保真预览、附件状态机），缺的是
**四件小事**：① Composer 非图片附件没有 chip 只有裸路径文本；② 预览停留单文件、
无最近引用/会话产物快捷区；③ 无产物版本/迭代时间线；④ 无多文件批量选择与发送队列。

## 1. 竞品矩阵：文件交互体验

| 产品/版本 | 输入侧（引用/附件） | 对话内（展示/预览） | 输出侧（产物/版本） | 可蒸馏优点 |
|---|---|---|---|---|
| QwenPaw v2.1.0 | 多文件预览在 composer 内换行，rich messages 进发送队列 | **Unified Files Workspace**：不离开 Chat 即可浏览/预览/编辑/对比/上传/下载 | 最近编辑/文件对比；快照恢复保留选定工作区文件 | ① 文件操作全收敛到 Chat 一侧；② 多文件发送队列；③ 文件对比 |
| Proma v0.7.3 | 输入框 Mention：`/` skill、`#` MCP、`@` 文件；选中显示彩色 icon chip（skill 紫/MCP 绿/文件蓝）；发送后同步渲染彩色 badge 并注入 `<mentioned_tools>` 结构化指令 | Agent 消息里的行内文件路径**自动渲染为可点击 chip**，点击弹预览（图片/视频/MD/JSON/XML/HTML/PDF/DOCX；不支持类型自动调系统默认程序） | 工作区共享目录 workspace-files/ 跨会话常驻；文件浏览器 = 会话文件（上）+ 工作区文件（底） | ④ 行内路径→chip 自动渲染；⑤ Mention 彩色 chip + 结构化注入；⑥ 跨会话共享工作区文件 |
| Hermes Desktop artifacts | — | 大产物从 transcript **提升为卡片**：版本步进器（‹ v2 of 3 › / Latest）、PREVIEW/SOURCE 切换、下载 | 产物卡片带版本时间线 | ⑦ 版本步进器；⑧ 预览/源码双视图切换 |
| Claude Code 生态 | `@` 引用单文件 = 完整内容进上下文；引用目录 = 目录清单不进全部内容 | 桌面端拖 PDF/DOCX 只有图片被支持（社区 bug 讨论）→ 说明「拖任意文档进 prompt」是强需求 | — | ⑨ 引用语义分层：单文件进内容、目录只进清单 |
| WPS 灵犀（原生型） | AI 对话 + 编辑区**同屏**，就地修改 | 编辑区即预览，保留批注/修订 | 产物留在 WPS 生态 | ⑩ 预览=编辑，就地修改不跳转 |
| WorkBuddy / Kimi Work / TRAE Work（产物型） | 装完第一件事是**授权文件目录**，任务都在授权目录内读写 | Agent 生成文件 → 预览器 → 下载/继续对话迭代 | 会话产物可下载 Zip | ⑪ 目录授权=文件访问权限心智；⑫ 会话产物集中打包 |
| ChatGPT Work vs Claude Cowork | — | Work 偏可编辑办公文件；Cowork 返回 **Live Artifact**（数据驱动仪表盘实时更新） | — | ⑬ 「活产物」概念：文件不再是静态交付物 |
| DeepSeek Harness | 左侧边栏 = 新建会话/工作区/各工作区会话 | 工作区选择 + 权限是标准布局 | — | ⑭ 工作区=会话的组织单元，文件随工作区走 |

## 2. 行业趋势（从调研提炼）

1. **文件交互从「上传附件」演进到「工作区即上下文」**：授权目录（WorkBuddy）→
   工作区文件树（DeepSeek Harness）→ `@` 引用（Claude Code）→ 工作区共享目录常驻
   （Proma workspace-files/）。用户的心智是「我在哪个工作区、哪些文件可见、权限如何」，
   而不是「我上传了什么」。
2. **产物体验是办公 agent 的核心竞争力**：行内文件 chip（Proma）、产物卡片
   （Hermes）、版本步进器、对比、下载——对话里给的是「可继续使用的成果」，
   而不是一段路径文本。产物型（WorkBuddy/Kimi）与原生型（WPS 灵犀）两条路线
   最终都要提供：产物预览、版本、审阅、权限。
3. **@ 引用系统化**：从「文本里塞路径」升级为「彩色 chip + 结构化注入 +
   会话内持久化」。Proma 在发送后历史中同步渲染彩色 badge 并注入
   `<mentioned_tools>` 结构化指令——引用不仅是视觉装饰，还影响模型侧的工具上下文。
4. **预览与编辑不离开 Chat**：QwenPaw Unified Files Workspace 把浏览/预览/编辑/
   对比/上传/下载全部收敛进对话侧；WPS 灵犀把编辑区与对话同屏。「跳外部程序打开」
   被当作兜底而非主路径。
5. **多文件是常态**：QwenPaw rich messages 在 composer 内多文件预览换行、发送队列；
   豆包/WorkBuddy 一次拖入图片/PDF/Word/Excel。单文件模式正在被多文件批量模式取代。

## 3. gaea 现状盘点（基于代码，2026-08-16）

### 已具备
- 文件链接化：`remarkFileLinks`（mdast 插件）+ `findFileMentions`，聊天正文、流式
  尾部、工具输出（`FileLinkText`）均可点击预览；代码块/已有链接/公式天然跳过。
- @ 引用菜单：`FileMenu`（目录浏览 / 工作区搜索 / 最近使用文件，本地持久化 20 条，
  扩展名 badge，键盘导航，Tab 进子目录）。
- 附件状态机：`useComposerAttachments`（拖拽 / 粘贴 / 截图 / PickFiles），图片走
  attachment 预览，非图片 base64 落盘后注入 `@路径` 文本；10MB 大文件异步确认；
  OCR 进度反馈。
- 文件树：`WorkspacePanel` + `FileTree`（懒加载目录、加载/错误三态 + 重试，点击文件
  收起面板 → 主区预览）。
- 保真预览：`FilePreview` / `FilePreviewModal`（docx/xlsx/markdown/text/image/error
  分 kind，标题栏「定位 / 外部打开」，OCR 进度，预览尺寸可拖）。
- 面板：MaterialsPanel（资料固定）、DeliverablesPanel（缩略图/复制路径）、
  ChangesPanel（变更）、WorkspaceSearchPanel（搜索）。

### 差距（本轮调研确认的候选落点）
1. **Composer 非图片附件 = 裸路径文本**：拖入 docx/xlsx 只往输入框注入 `@路径`，
   无 chip、无图标、不可点预览（对照 Proma 彩色 chip、QwenPaw 多文件预览换行）。
2. **预览停留单文件**：preview store 只有 `previewFile: string | null`，无预览历史/
   队列/前后切换；无「最近引用 / 会话产物」快捷区（对照 Hermes 版本步进器、
   QwenPaw 文件对比）。
3. **无产物版本/迭代时间线**：DeliverablesPanel 展示当前产物，但改完一版没有
   v1/v2/v3 时间线（对照 Hermes ‹ v2 of 3 ›）。
4. **发送无队列、多文件选择弱**：Composer 有排队列表（ComposerQueueList）但无
   rich messages 式多文件卡片预览；批量选择仅靠 PickFiles 多选后逐个注入文本。
5. **AI 消息内文件路径可点击但形态朴素**：markdown 里的路径渲染为内联按钮，无
   图标/扩展名 badge/大小信息，与 @ 菜单视觉不统一。
6. **引用语义不区分**：`@` 单文件与目录行为一致（对照 Claude Code：单文件进内容、
   目录只进清单）；`<mentioned_tools>` 结构化注入缺失。

## 4. 落地建议（优先级）

### P0（工作量小、体验提升大，建议下一轮直接做）
- **P0-1 非图片附件 chip 化**：拖入/选择的非图片文件不再注入裸文本，而是在
  Composer 附件栏渲染为 chip（图标 + 文件名 + 扩展名 badge + 移除），点击可预览；
  提交时仍按现有 `@路径` 逻辑注入上下文，行为零变化。
- **P0-2 行内文件 chip 视觉统一**：把 `FileLinkText` / markdown 文件链接渲染升级为
  「图标 + 文件名 + 扩展名 badge」，与 @ 菜单、附件 chip 共用一套视觉；点击预览不变。
- **P0-3 最近引用 / 会话产物快捷区**：在预览侧栏或面板顶部加「最近文件」快捷条
  （复用 useComposerMenus 的最近 20 条），一键回到刚看过的文件。

### P1（中等工作量，拉开体验差距）
- **P1-1 多文件预览 + 队列**：preview store 扩展为 `{ list: string[], index }`，
  支持 ←/→ 切换、列表导航；Composer 内多附件预览换行（对照 QwenPaw rich messages）。
- **P1-2 产物版本时间线**：DeliverablesPanel 对同一文件记录修改版本（v1/v2/v3），
  版本步进器 + 下载（对照 Hermes artifacts）。
- **P1-3 @ 引用语义分层 + 结构化注入**：目录引用只进目录清单；发送时同步注入
  `@路径` 的结构化引用说明（对照 Proma `<mentioned_tools>`）。

### P2（后续演进）
- **P2-1 文件对比**：同一文件修改前后 diff/并排对比（对照 QwenPaw）。
- **P2-2 大工具输出有界预览**：超长工具输出折叠为卡片、可展开（对照 QwenPaw
  #6637/#6677）。
- **P2-3 授权目录心智**：首次使用时引导选择/确认可访问目录，文件树顶部显示当前
  工作区与权限状态（对照 WorkBuddy / DeepSeek Harness）。
- **P2-4 上下文用量透明化**：图片附件不把 Base64 计入 context-usage 环，或明确
  展示占用（对照 QwenPaw #6968）。

## 5. 来源

- 腾讯云开发者社区《Agentic AI 与 WebOffice 融合》：https://developer.cloud.tencent.com/article/2718838
- QwenPaw v2.1.0 release（Unified Files Workspace / rich messages / 对比 / Unicode
  PDF / SVG / context-usage）：https://github.com/agentscope-ai/QwenPaw/releases/tag/v2.1.0
- Proma v0.7.3 release（行内文件 chip / Mention 系统 / workspace-files / 20 项
  搜索 / 压缩上下文）：https://newreleases.io/project/github/ErlichLiu/Proma/release/v0.7.3
- Hermes Desktop artifacts PR（大产物卡片 / 版本步进器 / PREVIEW-SOURCE）：
  https://github.com/NousResearch/hermes-agent/pull/72345
- Claude Code 生态 @ 引用（单文件进内容 / 目录进清单 / 拖拽 PDF-DOCX 需求）：
  https://jerry.blog.csdn.net/article/details/162636704
- ChatGPT Work vs Claude Cowork（Live Artifact / 可编辑办公文件）：
  https://www.datacamp.com/zh/blog/chatgpt-work-vs-claude-cowork
- DeepSeek Harness 首发体验（工作区/会话侧栏）：https://www.donews.com/news/detail/1/6670742.html

> 注：yun88.com（403）、jerry.blog.csdn.net（521）、hermes PR（FETCH_TIMEOUT）等源
> 未能直接抓取正文，结论基于搜索摘要与既有调研交叉验证。
