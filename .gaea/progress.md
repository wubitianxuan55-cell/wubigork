# 任务进度

> 最后更新 2026-08-13（v2.14.0 发布）

## 已发布：v2.14.0「办公板块会话化升级：项目分组 + 会话生命周期 + 任务目标 + 变更面板」（2026-08-13）

| 状态 | 任务 |
|------|------|
| ✅ | 侧边栏按项目聚合会话：当前工作区置顶，其余为最近打开且有会话的工作区；组内按当前→置顶→最近排序，>50 截断 |
| ✅ | 会话生命周期：活跃会话置顶（.pinned.json）、归档（移入 <sessions>/archive/）、「已归档」分组一键恢复 |
| ✅ | 任务目标（需求→验收）：会话首条用户消息自动锚定并随会话持久化，待办栏展示进行中/已验收，可一键切换 |
| ✅ | 文件变更面板：汇总写/改工具实际改动的文件及次数，覆盖 path/file_path/paths/edits/source+destination 等参数形态 |
| ✅ | 恢复会话保留工具过程：GaeaHistory 还原 tool/tool_result，恢复后过程卡与变更面板仍可见 |
| ✅ | 专注模式：Aim 按钮 / Ctrl+Shift+F 收起侧栏与右侧面板（状态持久化） |
| ✅ | 修复：会话删除三处注册表全量清理；记忆图谱三元组提前于成本条目入图；App 层 toast 静默失效；恢复历史待办收尾；重复快捷键与 WRITE_TOOL_NAMES 去重 |
| ✅ | 验证：go build/vet 通过；go test（app/config）全绿；前端 tsc+vite build 通过、vitest 138 例全过 |
| ✅ | 版本：v2.14.0（versioninfo.rc / wails.json）、releases/v2.14.0.md、CHANGELOG、README、releases/README |

## 下一阶段优化迭代规划（v2.15+）

> 目标：在「能用且好用」的基础上，把工程可维护性和关键交互可靠性补到下一档；
> 先做低成本高收益，再做结构性重构。每阶段可独立发布。

### P0 — 稳定性与缺陷收口（1 轮）

- [x] 会话恢复/重命名补失败提示：`handleResumeSession` / `handleRenameSession` 当前异常会静默
      （归档/置顶/恢复刚补过 toast），统一接入 App 层 toast。
- [x] 归档会话的「永久删除」加二次确认：与活跃会话删除交互保持一致，避免误删不可恢复数据。
- [x] `Welcome` 最近会话与侧边栏「项目」视图对齐：改为从 `projectGroups` 派生跨项目最近会话，
      而非沿用旧扁平 `sidebarSessions`（当前仅当前工作区、且被分页截断）。
- [x] 变更面板补“累计改动次数”与“最近改动”排序，让验收前核对范围更快。
- [x] 回归测试补：会话删除后三处注册表（标题/目标/置顶）全部清空的 Go 用例。

### P1 — 关键交互测试补强（1–2 轮）

- [ ] 为核心办公交互补 Vitest + Testing Library 用例：App 主流程（新建/恢复/发送）、
      Sidebar（项目分组/搜索/置顶/归档）、Composer（slash 命令/权限切换）、Transcript（过程卡折叠/滚动跟随）。
- [ ] 目标：前端用例从 138 提升到 ~200+，并覆盖“恢复历史后工具卡/待办收尾”等易回归路径。
      （当前进度：164，新增 store.test.ts 8 + ChangesPanel.test.tsx 2 + useSessionManager.test.ts 4 + Sidebar.test.tsx 3 +
        command.test.ts 3 + time.test.ts 6；同步把 App 斜杠命令分类与 Sidebar 相对时间抽成纯函数）
- [x] 给 `store.ts` 的 reducer/applyEvent 补纯函数表驱动测试（历史重建、turn 收尾、跨轮文本不覆盖）。

### P2 — 结构收敛与可维护性（1–2 轮）

- [ ] 拆分 800+ 行单文件：`App.tsx`（877）、`CostLibraryView.tsx`（840）、`CapabilitiesPanel.tsx`（818）、
      `Composer.tsx`（775）、`Sidebar.tsx`（720）按职责切为子组件/hook。
- [ ] `lib/mock.ts`（1344）按模块拆分，web 调试 mock 与生产代码隔离。
- [ ] 收敛 `bridge.ts` 动态/静态 import 混用产生的 chunk 告警。
- [ ] `WRITE_TOOL_NAMES` / 会话注册表等易漂移常量补单一来源 + 编译期校验。

### P3 — 办公能力增强（按需 1–2 轮）

- [ ] 变更面板接入 diff 预览（复用 `lib/diff.ts`），点文件直接看本次改动差异，而不只是路径列表。
- [ ] 成本条目补「地区 + 期数」字段（信息价三要素：规格 + 地区 + 时间），导入时映射到多级分类。
- [ ] 待办工具任务完成时自动置 completed（减少依赖模型回写状态，前端已有 turn_done 兜底）。
- [ ] 跨工作区会话全文搜索（当前侧栏搜索只覆盖当前分组内活跃会话）。

### 发布与产物（每次发布前执行）

- [ ] 发布版移除 WebView2 渲染调试端口（`main.go` 诊断配置，本地保留）。
- [ ] exe 瘦身：Wails 构建压缩 / 前端大 chunk（mermaid、MemoryHub 等）按需加载。
- [ ] 保持「CHANGELOG / README / wails.json / versioninfo.rc / releases / .gaea」六处版本与记录同步。

## 已发布：v2.13.22「修复整轮结束后大过程卡折叠 / 小过程卡合并误展开」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 定位：合并大卡复用首段小卡实例（key 相同）→ 大卡继承小卡折叠态；上一版反向撑开实例又造成「小卡展开成大卡」 |
| ✅ | 修复：整轮结束后的合并段用独立 key（done- 前缀）全新挂载，大卡默认展开；小卡实例保持折叠 |
| ✅ | 回归测试：小卡挂载折叠、大卡挂载展开（35 文件 129 例全过） |
| ✅ | 验证：tsc + 前端 129 例全过；wails build 通过 |
| ✅ | 发布：v2.13.22 版本号、releases/v2.13.22.md、CHANGELOG、gaea-v2.13.22.exe、git tag v2.13.22 |

## 已发布：v2.13.21「办公板块安全审计：封堵子代理绕过持久化写入审批」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 审计定位：默认 task 子代理继承全部工具但运行在 headless 审批通道，可静默写入成本库/记忆/知识库/技能 |
| ✅ | 修复：FilterRegistry 剔除持久化写入工具（cost_save/remember/forget/knowledge_add/promote_session_facts/install_skill） |
| ✅ | 主代理 hardAskTools 补 forget / install_skill |
| ✅ | 复查：Windows 控制台子进程全部隐藏窗口（含 MCP stdio）；知识库不注入上下文、技能索引 4000 上限、记忆索引 3000 上限 |
| ✅ | 验证：go build/agent/control/permission 测试通过；wails build 通过 |
| ✅ | 发布：v2.13.21 版本号、releases/v2.13.21.md、CHANGELOG、gaea-v2.13.21.exe、git tag v2.13.21 |

## 已发布：v2.13.20「记忆/知识库写入强制确认 + 记忆索引注入预算」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 定位：remember/knowledge_add/promote_session_facts 由 AI 直接落盘、无确认，杂乱记忆整体注入上下文 |
| ✅ | 修复：三个记忆写入工具纳入硬性逐条审批（任何权限级别都弹卡、批准仅本次生效），审批摘要含条目名/描述/分类/来源 |
| ✅ | 记忆索引注入预算：Saved memories 索引限 3000 runes，截断提示用 memory_search 查询 |
| ✅ | 前端：硬性审批工具统一隐藏「本会话允许」 |
| ✅ | 验证：go build/control/memory 测试通过；前端 tsc + 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.20 版本号、releases/v2.13.20.md、CHANGELOG、gaea-v2.13.20.exe、git tag v2.13.20 |

## 已发布：v2.13.19「成本库写入强制逐条确认，AI 不再直接入库」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 定位：auto/yolo 权限级别自动放行全部工具，cost_save 直接把 AI 价格写入成本库 |
| ✅ | 修复：cost_save 硬性逐条审批（AlwaysAsk），任何权限级别都弹卡、不记忆会话放行；审批卡展示名称/单价/单位/规格/来源 |
| ✅ | 前端：cost_save 审批卡隐藏「本会话允许」并加逐条确认提示 |
| ✅ | 回归测试：yolo 下仍审批、会话放行不记忆、摘要格式 |
| ✅ | 验证：go build/control/permission 测试通过；前端 tsc + 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.19 版本号、releases/v2.13.19.md、CHANGELOG、gaea-v2.13.19.exe、git tag v2.13.19 |

## 已发布：v2.13.18「通用办公左侧面板重设计（参考 Codex）」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 侧栏按 Codex 风格重排：紧凑头部、搜索框前置放大镜、会话行标题+时间+预览、选中左色条、区块小字统一 |
| ✅ | 功能全保留：搜索/重命名/删除/历史、后台任务、事实底座、记忆/知识库/技能导航、折叠与拖拽 |
| ✅ | 验证：tsc + 前端 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.18 版本号、releases/v2.13.18.md、CHANGELOG、gaea-v2.13.18.exe、git tag v2.13.18 |

## 已发布：v2.13.17「修复运行中已完成的小过程卡不折叠」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 定位：v2.13.9 把 ProcessCard 初始状态改为默认展开，运行中已完成的分段小卡（不含文本）不再收起 |
| ✅ | 修复：分段小卡（small）默认折叠、段完成自动收起；含文本的合并大卡保持展开；流式最后一段保持展开 |
| ✅ | 验证：tsc + 前端 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.17 版本号、releases/v2.13.17.md、CHANGELOG、gaea-v2.13.17.exe、git tag v2.13.17 |

## 已发布：v2.13.16「办公板块铺满窗口 + 顶部标签删除 + 底栏只显示本地模型」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 删除办公板块顶部「办公」二级标签：MainLayout 直接挂 GaeaPage，删除 OfficeHubPage 与相关样式 |
| ✅ | 底栏已启动模型只显示本地模型（isLocal 过滤），超载报警维持本地统计，云端引擎不展示不计入 |
| ✅ | 通用办公铺满窗口：内容区对办公页去除 8/16px 内边距（与聊天页一致） |
| ✅ | 验证：tsc + 前端 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.16 版本号、releases/v2.13.16.md、CHANGELOG、gaea-v2.13.16.exe、git tag v2.13.16 |

## 已发布：v2.13.15「删除方案编写，办公板块收敛为单一入口」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 删除方案编写前端页面（OfficePage）与后端（internal/office/proposal、internal/office/db） |
| ✅ | 移除全部 Proposal* 绑定、office 模块注册、方案库设置卡片；办公二级导航删除，板块收敛为通用办公单一入口 |
| ✅ | 左脑记忆源改接办公记忆 facts（三脑检索/记忆中枢不受影响，测试通过） |
| ✅ | 验证：go build/定向测试通过；前端 tsc + 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.15 版本号、releases/v2.13.15.md、CHANGELOG、gaea-v2.13.15.exe、git tag v2.13.15 |

## 已发布：v2.13.14「通用办公工具/技能面板显示 Word·Excel·PDF 技能」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 定位：技能索引只扫当前工作区 .gaea/skills + ~/.gaea/skills，文档技能只装在仓库，其它工作区面板为空 |
| ✅ | 安装：docx/xlsx/pdf/pptx 复制到 ~/.gaea/skills 用户级全局目录（金具厂工作区实测可见） |
| ✅ | 工具面板新增「文档技能」分组（docx/xlsx/pdf/pptx） |
| ✅ | 验证：tsc + 前端 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.14 版本号、releases/v2.13.14.md、CHANGELOG、gaea-v2.13.14.exe、git tag v2.13.14 |

## 已发布：v2.13.13「聊天面板左侧栏可折叠 + 悬浮绑定模型卡随面板隐藏」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 聊天左侧会话栏新增折叠：窄栏 52px（保留展开/新建按钮），展开态头部加折叠按钮，状态持久化 |
| ✅ | 折叠时左下角「聊天」绑定模型卡随面板一起隐藏 |
| ✅ | 验证：tsc + 前端 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.13 版本号、releases/v2.13.13.md、CHANGELOG、gaea-v2.13.13.exe、git tag v2.13.13 |

## 已发布：v2.13.12「通用办公布局优化：精简右侧边栏 + 绑定模型卡随面板折叠」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 右侧边栏删除「成本库」「搜索」标签页（含命令面板搜索入口） |
| ✅ | 左下角绑定模型卡移入左侧栏底部，折叠随面板隐藏，不再盖住「知识库/MCP 与技能」按钮 |
| ✅ | 验证：tsc + 前端 128 例全过；wails build 通过 |
| ✅ | 发布：v2.13.12 版本号、releases/v2.13.12.md、CHANGELOG、gaea-v2.13.12.exe、git tag v2.13.12 |

## 已发布：v2.13.11「修复记忆中枢·用户画像打不开」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 定位：GaeaProfileConflicts 无冲突时返回 nil → JSON null，前端 conflicts.length 抛 TypeError（ErrorBoundary 日志实证） |
| ✅ | 修复：DetectConflicts 返回 []；ProfileLibrary conflicts/tags null 兜底；图谱/聊天记忆页同步加固 |
| ✅ | 回归测试：Go DetectConflicts 非 nil + 前端 ProfileLibrary null 用例（128 例全过） |
| ✅ | 发布：v2.13.11 版本号、releases/v2.13.11.md、CHANGELOG、gaea-v2.13.11.exe、git tag v2.13.11 |

## 已发布：v2.13.10「修复办公文档处理反复弹 cmd 黑窗」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 定位：gaea 后台批量转换 docx 时 `python -m markitdown` 未隐藏窗口，每次 spawn 闪 cmd/conhost（150 秒实测抓到 8 次） |
| ✅ | 修复：markitdown / xlsx 重算 / 报告导出 / 绘图 / whisper 桌面操作共 5 处补 CREATE_NO_WINDOW |
| ✅ | 发布：v2.13.10 版本号、releases/v2.13.10.md、CHANGELOG、gaea-v2.13.10.exe、git tag v2.13.10 |

## 已发布：v2.13.9「每一轮的大过程卡默认展开 · 内部卡默认折叠」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 撤回 v2.13.8 的「仅最新回合」限制：每一轮的大过程卡输出完成后都默认保持展开，旧回合不再自动折叠 |
| ✅ | 大过程卡内部思考卡默认折叠；工具卡/工具组保持默认折叠；用户手动操作过的大过程卡仍尊重其选择 |
| ✅ | 发布：v2.13.9 版本号、releases/v2.13.9.md、CHANGELOG、gaea-v2.13.9.exe、git tag v2.13.9 |

## 已发布：v2.13.8「仅最新回合的大过程卡默认展开」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 仅最新回合的外层大过程卡默认展开；旧回合新回合开始后自动折叠；手动展开过的旧卡保留 |
| ✅ | 发布：v2.13.8 版本号、releases/v2.13.8.md、CHANGELOG、gaea-v2.13.8.exe、git tag v2.13.8 |

## 已发布：v2.13.7「办公上下文窗口修正 + 大上下文处理提示」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 根因定位：办公 ContextWindow 写死 1M，压缩阈值 80 万，会话 18.5 万 token 也不压缩，模型首字分钟级 |
| ✅ | ContextWindow 1M → 256k，超限自动压缩 |
| ✅ | 前端「处理大上下文中 · N」提示；最终回答兜底补渲染（v2.13.6 并入） |
| ✅ | 发布：v2.13.7 版本号、releases/v2.13.7.md、CHANGELOG、gaea-v2.13.7.exe、git tag v2.13.7 |

## 已发布：v2.13.5「运行中强制跟随底部」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | Transcript 运行中强制滚动到底部（智能滚动标记失效导致视图冻住的“卡死假象”已修复） |
| ✅ | 运行结束后恢复智能滚动，上翻浏览不被打断 |
| ✅ | 发布：v2.13.5 版本号、releases/v2.13.5.md、CHANGELOG、gaea-v2.13.5.exe、git tag v2.13.5 |

## 已发布：v2.13.4「外层过程卡完成后默认展开」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 外层大过程卡（ProcessCard）运行结束后不再折叠，默认保持展开；内部工具卡/工具组恢复默认折叠（撤回 v2.13.3 误改） |
| ✅ | 发布：v2.13.4 版本号、releases/v2.13.4.md、CHANGELOG、gaea-v2.13.4.exe、git tag v2.13.4 |

## 已发布：v2.13.3「过程卡完成后默认展开」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | ToolCard 大过程卡（≥10 行或 ≥2000 字符）完成（done/error）后默认展开，用户折叠后不再干预 |
| ✅ | ToolGroup 组内含大卡且完成时默认展开；小卡保持折叠 |
| ✅ | 发布：v2.13.3 版本号、releases/v2.13.3.md、CHANGELOG、gaea-v2.13.3.exe、git tag v2.13.3 |

## 已发布：v2.13.2「办公过程文件落盘规范」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 办公 agent 执行纪律新增「文件落盘规范」：过程/中间文件统一 .gaea/work/<任务名>/，交付物 .gaea/exports/，源文件原位不动 |
| ✅ | 启动自动创建 .gaea/work/ 与 .gaea/exports/；.gitignore 忽略 .gaea/work/ |
| ✅ | 发布：v2.13.2 版本号、releases/v2.13.2.md、CHANGELOG、gaea-v2.13.2.exe、git tag v2.13.2 |

## 已发布：v2.13.1「修复 @PDF 引用注入二进制」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | @引用 office 文档转换失败时注入占位提示，不再把 %PDF-1.4 原始字节塞进模型上下文 |
| ✅ | 二进制检测扩展到整个读取窗口（此前只查前 8KB，PDF 头无 NUL 会漏检） |
| ✅ | 发布：v2.13.1 版本号、releases/v2.13.1.md、CHANGELOG、gaea-v2.13.1.exe、git tag v2.13.1 |

## 已发布：v2.13.0「通用办公打磨 + 模型中心优化 + 本地模型实测」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 方案分节字数修复：流式「生成该节」+ 批量生成按 WordTarget 续写（原硬编码 100 字截断） |
| ✅ | docx 读取乱码修复：ReadTextFile 统一走 ConvertToMarkdown |
| ✅ | 前端运行态兜底：停止按钮无条件复位 + 30s GaeaRunning 看门狗（turn_done 丢失不再卡「执行中」） |
| ✅ | 待办自动收尾：从未推进的列表 turn_done 后显示已完成 |
| ✅ | format_convert 输出父目录自动创建 + 回归测试 |
| ✅ | docmd MarkItDown 通道（docx/pptx）+ PDF 页数上限保持 |
| ✅ | 模型中心功能绑定/板块细节优化；本地引擎思考参数透传（enable_thinking / chat_template_kwargs） |
| ✅ | whisper web_search、角色画像文件化等累积修复 |
| ✅ | 本地模型实测：Herdsman 三模型 20 任务×思考/非思考双模式 + Ollama GLM；脚本 scripts/test_herdsman_models.go 可复现 |
| ✅ | 发布：v2.13.0 版本号、releases/v2.13.0.md、CHANGELOG、gaea-v2.13.0.exe、git tag v2.13.0 |

## 验证

- go build / go vet 干净；方案包（proposal）测试全绿
- 前端 127 例全过，tsc + vite build 通过
- 产物 gaea-v2.13.0.exe（约 39.6MB），桌面端同步

## 后续候选

- 成本库导入行分类映射到多级路径（当前导入落到一级节点，可在编辑时细分）
- 成本条目补充「地区 + 期数」字段（信息价三要素：规格 + 地区 + 时间）
- 发布版移除 WebView2 渲染调试端口（main.go 诊断配置，本地保留即可）
- exe 瘦身：Wails 构建压缩 / 前端大 chunk 按需加载
- 待办工具：任务完成时自动置 completed（当前靠模型回写状态，前端 turn_done 已兜底）
- tesseract 缺失时 OCR 错误提示更友好（当前靠本地视觉模型兜底）
