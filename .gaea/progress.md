# 任务进度

> 最后更新: 2026-08-10 04:10:00

## 当前版本

- **v2.10.1（正式发布，2026-08-10）**：开工前（计划卡片/@ 引用增强/资料概览）+
  交付收尾（文件链接点击预览/交付卡片/会话产物面板/编辑后刷新/粘贴表格即数据）+
  记忆自进化（自动做梦）。产物 releases/gaea-v2.10.1.exe（40.8MB，SHA256 已归档），
  桌面端同步；go test 全量 + vitest 89 例全过，wails build 通过，git tag v2.10.1。

## 发布动作（v2.10.1）

- 版本号对齐：app_info.go / wails.json / frontend/package.json / versioninfo.rc → 2.10.1
- CHANGELOG.md 新增 v2.10.1 条目；README 通用办公能力更新；releases/v2.10.1.md 发布说明；
  releases/README.md 版本表补 v2.10.0 + v2.10.1 两行（v2.10.0 当时漏记）
- .gitignore 补充运行/测试产物（.tmp* / cpu.out / xlsx.out / *.test.exe / .gaea/exports|sessions）

## 开发中（v2.10.1 方向）：开工前 P0 落地

| 状态 | 任务 |
|------|------|
| ✅ | @ 文件引用增强：Composer @ 菜单改为「目录内浏览 / 最近使用 / 工作区跨目录搜索」三合一（统一 AtEntry），文件行显示扩展名徽标（docx/xlsx/pdf/md/png…），选过的文件进 localStorage 最近使用（≤20） |
| ✅ | 后端 GaeaFileSearch：工作区文件名搜索（大小写不敏感、深度 ≤6、限 30 条、跳过 .git/node_modules/dist/build/.tmp* 等噪音目录、中文命中），与 @ 菜单打通（bridge FileSearch → GaeaFileSearch，含 mock） |
| ✅ | 测试：后端 GaeaFileSearch（中文命中/噪音目录跳过/limit/深目录不 panic）通过；前端 88 例全绿（新增 FileMenu 徽标用例），tsc + vite build 通过 |
| ✅ | 工作区资料概览（P0-③）：右侧新增「资料」Tab（MaterialsPanel）——docx/xlsx/pptx/pdf/md/txt/csv 按 文档/表格/演示/PDF 分组、最新在前；行内「预览 / 一键 @ 引用 / 外部打开」；引用通过 useComposerInsertStore 插入输入框（面板→Composer 打通） |
| ✅ | 后端 GaeaMaterials：资料文件按修改时间倒序（限 100、深度 ≤5、跳过噪音目录、只收 office/文本扩展名） |
| ✅ | 测试：后端 GaeaMaterials（类型过滤/噪音跳过/最新在前/路径无前导斜杠）通过；前端 89 例全绿（新增 MaterialsPanel 分组+引用+预览用例），tsc + vite build 通过 |
| ✅ | 开工前计划卡片（P0-①）：AutoPlan 门控打通——AgentRunner.Plan 用同一 provider 一次性生成开工计划；Controller 在回合前（非简单查询时）以 ask 卡片询问「确认执行 / 先调整」，确认才执行，取消则通知并结束；计划生成失败自动回退为直接执行 |
| ✅ | 接线：boot.Options.AutoPlan ← config auto_plan（ask/on 开启）；办公板块 gaeaLoadConfig 默认 ask；前端 AskCard 提示改为 Markdown 渲染（计划以列表呈现） |
| ✅ | 测试：AgentRunner.Plan（成功/失败）、Controller askPlanApproval（确认/调整/空执行器回退）通过；agent/control/boot/app 全量测试通过，go vet 干净；前端 89 例全绿，tsc + vite build 通过 |

### 开工前 P0 全部落地（③ @ 引用增强 + 资料概览 + 计划卡片）

## 调研（2026-08-10）：开工前阶段 — 第五轮竞品优点蒸馏

> 产出 docs/market-research-2026-08-office-prework-context.md
> 主题：任务发出后、AI 动手前的文件交互与调研启动

### 蒸馏出的可落地优点（P0）
1. 开工前计划卡片：auto-plan 结果渲染为「计划卡片」（步骤/待读资料/将用工具/待确认），
   用户确认后再开干——对标 WorkBuddy Plan 模式、千问办公任务步骤清单
2. @ 文件引用增强：@ 菜单支持跨目录搜索 + 最近使用 + 按扩展名分组——对标 WorkBuddy @
   文件列表、Claude @ 按需加载
3. 工作区资料概览：切换/首次进入工作区时列出 docx/xlsx/pdf/md/csv 资料视图，一键
   @引用为上下文——对标千问办公选文件夹授权、aily 关联知识

### 蒸馏出的可落地优点（P1）
4. 工作区全文搜索（轻量 RAG，SQLite FTS5 索引文本文件）
5. 项目常用资料装配（pin 常用文件 → 新会话自动进上下文）
6. 任务模板库（欢迎页能力卡 + slash 升级为预置任务模板）

### 明确不做
- 云端知识库 / 团队共享上下文（个人工具定位）
- 整个文件夹全量塞进 prompt（坚持 @ 按需加载 + 检索装配）

## 开发中（v2.10.1 方向）：记忆自进化 P0 落地

| 状态 | 任务 |
|------|------|
| ✅ | 自动做梦（空闲自整理）v1：gaeaBuildController sink 在 TurnDone（成功）后触发后台整理——取最后一轮对话 → office 模型提炼（Kimi 二问纪律：只记稳定事实、≤5 事实/≤3 笔记、宁缺毋滥）→ 写入长期记忆 |
| ✅ | 写入路径：Controller.SaveDreamFacts（与 promote 同一 Store.Save 路径，按 name slug 去重、更新即修订），笔记走 QuickAdd（user/project/local 作用域） |
| ✅ | 防御：单飞（同时只跑一个）、实质内容门槛（≥2 条消息且助手输出 ≥100 字符，寒暄不整理）、90s 超时、失败静默跳过 |
| ✅ | 用户反馈：整理成功后发 Notice 事件「已自动整理记忆：新增 N 条事实、M 条笔记」，前端聊天内可见 |
| ✅ | 测试：parseDreamOutput（围栏/前缀/坏 JSON/空名过滤）、dreamTurnMessages（只取最后一轮 user/assistant）、dreamWorthwhile（寒暄不触发）、输入截断；Controller.SaveDreamFacts 去重/更新/空值跳过；internal/app 与 control 全量测试通过，go vet 干净 |

### 说明（自动回忆① 与生命周期③ 现状）
- 自动回忆注入（①）大部分已存在：memory.Compose 把文档索引 + ProfileBlock（user 语义事实自动聚合画像）+ ProceduralBlock（过程性规则每轮注入）折进系统提示词；剩余差距是「高频/近期排序与来源时间戳」，留作细化
- 记忆生命周期（③）部分已存在：sqlite facts 表有 created_at/updated_at、Archive 软删除；剩余差距是 last_used_at 高频排序与冲突修订链

## 调研（2026-08-10）：记忆与自进化 — 第四轮竞品优点蒸馏

> 产出 docs/market-research-2026-08-office-memory-evolution.md

### 蒸馏出的可落地优点（P0）
1. 自动回忆注入：会话开始时把用户画像 + 高频/近期 facts + 项目记忆压缩成「记忆上下文」
   注入提示词（带来源与时间戳）——对标 Claude MEMORY.md 前 200 行 / WPS 记忆自动带入
2. 空闲自整理（自动做梦）：会话结束（turn_done）后台归纳对话 → 新事实/偏好/踩坑/技能
   候选入库，复用 compact 提取 + fact 稳定 slug 去重，写入频率受控（Kimi 二问思路）
3. 记忆生命周期：facts 增加 created_at/last_used_at/source_message；同 slug 冲突修订链、
   使用计数参与注入排序、过期候选仅提示不自动删——对标 WPS 夜间整理 / ChatGPT Dreaming

### 蒸馏出的可落地优点（P1）
4. 记忆可控与溯源：MemoryPanel 显示来源会话/消息可跳转 + 「不再记住」+ 记忆开关
5. 轻量检索与压缩注入：facts/画像按「关键词+时间+高频」排序，注入预算 500-800 token
6. 方法论自动候选：多次同类任务后自动提示可沉淀技能（千问办公组织级 Skill 的个人版）

### 明确不做
- 团队共享记忆 / 组织级 Skill 权限（个人工具定位）
- 云端记忆存储与跨设备同步
- 自动删除用户记忆（只提示候选，删除须用户确认）

## 开发中（v2.10.1 方向）：P1 深化落地

| 状态 | 任务 |
|------|------|
| ✅ | 产物溯源：会话产物面板每条记录生成它的轮次（turn），悬停「跳转到生成它的消息」（MessageSquare）→ 收起面板并滚动到对应轮次（对标 AutoGPT Open chat / Claude Provenance） |
| ✅ | 编辑后自动回写刷新：useUpdatedFilesStore 变化 → 工作区文件树自动刷新（替代手动刷新按钮） |
| ✅ | 粘贴表格即数据：tableData.ts 识别 CSV/TSV 表格块（≥2 行、列数一致、≥2 列），Composer 显示「已识别表格数据 N×M」提示条 + 「发送时转为 Markdown 表格」开关（默认开），普通发送与 Shift+Enter 纠正发送均生效 |
| ✅ | 测试：前端 87 例全绿（新增 tableData 5 例、产物溯源 1 例），tsc + vite build 通过 |

## 开发中（v2.10.1 方向）：P0 成果交付收尾体验落地

| 状态 | 任务 |
|------|------|
| ✅ | 会话产物面板：右侧新增「产物」Tab（DeliverablesPanel），从会话消息提取交付文件（去重、最新在前），点击预览，悬停打开/定位/复制路径；空状态引导（对标 Kimi 工作空间 / 千问办公产物面板） |
| ✅ | 交付卡片增强：图片缩略图（AttachmentDataURL 前端加载，无新依赖）、复制路径动作、预览内编辑后「已更新」徽标 |
| ✅ | 已更新徽标链路：useUpdatedFilesStore + DocxPreview（应用修订/接受拒绝）/ XlsxPreview（AI 编辑/写单元格/重算/行列操作）成功后 markUpdated |
| ✅ | 测试：前端 81 例全绿（新增 DeliverablesPanel 3 例、DeliverableCards 已更新用例），tsc + vite build 通过 |
| ⏸️ | Office（xlsx/docx/pptx/pdf）首屏缩略图：需 LibreOffice→PDF→PNG 管线（pdftoppm/PyMuPDF 打包或依赖决策），暂缓，避免引入环境依赖 |

## 调研（2026-08-10）：对话内成果交付与文件交互 — 第三轮竞品优点蒸馏

> 产出 docs/market-research-2026-08-office-deliverable-ux.md

### 蒸馏出的可落地优点（P0）
1. 会话产物面板：右侧文件树之上新增「本次会话产物」视图（复用 findFileMentions +
   交付扩展名过滤，按时间倒序，点击预览/悬停打开/定位/复制路径）——对标 Kimi 工作空间、
   千问办公右侧产物面板、Claude artifact 网格
2. 交付卡片增强：图片缩略图（AttachmentDataURL）、xlsx/pptx 首屏预览图、复制路径动作、
   预览内编辑后卡片自动标「已更新」——对标豆包缩略图卡片、千问办公改完即交付

### 蒸馏出的可落地优点（P1）
3. 产物溯源：文件 ↔ 生成它的消息/任务跳转（对标 AutoGPT Open chat / Claude Provenance）
4. 编辑后自动回写刷新：DocxApplyEdit/XlsxEdit 成功后文件树/预览/卡片自动刷新
5. 粘贴即数据：Composer 粘贴 CSV/表格文本自动识别为结构化上下文

## 开发中（v2.10.1 方向）：输出文本文件引用全量可点击预览

| 状态 | 任务 |
|------|------|
| ✅ | 参考 openclaw / llama.cpp 同类实现，把"文件路径识别"从字符串正则替换改为 mdast（remark 插件）AST 层处理：代码块/行内代码/已有链接/公式天然跳过，无占位符保护 hack |
| ✅ | 聊天正文（含流式未稳定尾部）、工具输出、ask 提问中的文件引用均可点击打开内置预览：覆盖绝对路径（C:\…、/…）、相对路径（exports/…、.gaea/exports/…）、关键词引导裸文件名（输出文件：xxx.docx） |
| ✅ | 后端 GaeaPreview 支持裸文件名在常见输出目录（exports / .gaea/exports / outputs / docs / uploads / attachments / templates 等）中自动解析 |
| ✅ | gaea 输出约定：交付/生成/保存文件时正文必须给出 [文件名](路径) 可点击链接，路径不进代码块 |
| ✅ | 交付物附件卡片（对齐千问办公/Kimi）：回复尾部把正文中的文件引用去重渲染成"图标+文件名+扩展名"卡片，整卡点击预览，悬停提供外部打开/定位（DeliverableCards，接入 AssistantMessage） |
| ✅ | 测试：前端 77 例全绿（Markdown 渲染 / mdast 插件 / 流式尾部 / FileLinkText / 交付卡片 / 识别逻辑），tsc + vite build 通过，后端 GaeaPreview 裸文件名解析用例通过 |

## 当前版本

- **v2.10.0（正式发布）**：通用办公三阶段闭环（前期解析 / 中期编辑 / 后期输出）
  全部落地，桌面端 gaea.exe 已打包发布（Windows x64，约 40MB）。

## 本阶段已完成

| 状态 | 任务 |
|------|------|
| ✅ | 办公前期解析：docx/xlsx/pdf → Markdown 提速 + 扫描件 OCR 四级管线 + 事实底座 |
| ✅ | 办公中期编辑：Word 框选即改/修订、Excel 单元格级编辑（插行插列、公式重算） |
| ✅ | 办公后期输出：统一交付出口（docx/pptx/xlsx/md）+ 模板 + 成本测算模板 |
| ✅ | 跨应用联动：Excel 数据 → 图表 → 嵌入 Word/PPT 并随数据同步 |
| ✅ | Codex 式文件预览布局：右侧文件树 + 主区域可拖宽预览 |
| ✅ | 真实模型验收：DeepSeek 绑定自主生成成本测算表（公式 + 原生图表） |

## 已按用户决定收窄的范围

- 排序 / 筛选 / 条件格式不在 gaea 内做预览层实现（不写回文件），后期交给
  Excel/WPS 专业软件处理；gaea 保留真实落盘的编辑能力。

## 备忘

- 发布动作：版本号对齐 v2.10.0（app_info.go / wails.json / package.json），
  产物 releases/gaea-v2.10.0.exe，桌面端同步，git 打 tag v2.10.0。
