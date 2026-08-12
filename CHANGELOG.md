# gaea · 多功能 AI 助手

## v2.13.19「成本库写入强制逐条确认，AI 不再直接入库」（2026-08-12）
> 修复成本库被 AI 生成的虚高价格污染：此前权限级别为 auto/yolo 时所有工具
> 自动放行，cost_save 直接把数据写进成本库。现在 cost_save 成为硬性审批项——
> 任何权限级别（含 yolo）都必须弹审批卡，逐条确认条目名称/单价/单位/规格/
> 来源后才写入；不提供「本会话允许」，批准仅本次生效。
> 详见 releases/v2.13.19.md。
- permission.Gate：新增 AlwaysAsk 硬门，cost_save 无视权限级别/放行规则/会话记忆
- control：gateApprover 对 cost_save 强制 requestApproval（alwaysPrompt），审批摘要含条目名称/单价/单位/规格/来源
- SetPermLevel：auto/yolo 均保留 cost_save 硬门（yolo 改用空策略 gate 替代 nil gate）
- ApprovalModal：cost_save 隐藏「本会话允许」，显示逐条确认提示
- 回归测试：yolo 下仍触发审批、会话放行不被记忆、审批摘要格式
- 验证：go build/control/permission 测试通过；前端 tsc + 128 例全过；wails build 通过

## v2.13.18「通用办公左侧面板重设计（参考 Codex）」（2026-08-12）
> 左侧面板按 Codex 会话栏风格重排：紧凑头部（logo + 名称 + 新建/折叠图标按钮）、
> 搜索框前置放大镜、会话行改为「标题 + 时间 + 预览」结构（悬停操作、左色条选中态）、
> 区块头统一安静小字，功能与折叠/拖拽行为全部保留。
> 详见 releases/v2.13.18.md。
- Sidebar.tsx：头部精简、会话行重构、搜索框带图标、分组头/区块头统一，删除大号胶囊新建按钮
- 保留：会话搜索/重命名/删除/历史、后台任务、事实底座（导出/沉淀/复制/清空）、记忆/知识库/技能导航、折叠与拖拽调整宽度
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.18.exe

## v2.13.17「修复运行中已完成的小过程卡不折叠」（2026-08-12）
> 修复 v2.13.9 引入的回归：当时为让「大过程卡（整轮合并、包含文本卡的卡）」
> 保持展开，把 ProcessCard 初始状态改成了默认展开，导致运行中不含文本的
> 分段「小过程卡」在段完成后也不再收起。现在小过程卡默认折叠、段完成后
> 收起；含文本的大过程卡保持展开；正在流式的最后一段保持展开。
> 详见 releases/v2.13.17.md。
- Transcript.ProcessCard：新增 small 标记（运行中的分段小卡）；初始折叠、完成/历史段自动收起
- 大过程卡（consolidated，含中间文本）默认展开逻辑不变
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.17.exe

## v2.13.16「办公板块铺满窗口 + 顶部标签删除 + 底栏只显示本地模型」（2026-08-12）
> 通用办公三个显示细节优化：① 删除办公板块顶部「办公」二级标签，模块直接由
> 通用办公工作台承载；② 底部状态栏的已启动模型列表/超载报警只统计本地模型，
> 云端引擎（xai/deepseek/opencode）不再展示与计入；③ 全屏/最大化时通用办公
> 铺满整个窗口（此前内容区对办公页保留了 8/16px 内边距）。
> 详见 releases/v2.13.16.md。
- MainLayout：办公页路由直接挂 GaeaPage，删除 OfficeHubPage 与 office-hub/office-page 样式；内容区对办公页去内边距、铺满
- StatusBar：已启用模型列表按 isLocal 过滤，只显示本地模型；超载报警维持本地统计
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.16.exe

## v2.13.15「删除方案编写，办公板块收敛为单一入口」（2026-08-12）
> 办公板块下线「方案编写」二级分支：删除整个方案编写模块（前端页面、
> proposal/db 后端包、Proposal* 绑定、模块注册、方案库），去除办公二级导航，
> 办公板块直接承载通用办公工作台。左脑记忆源从方案库改为办公记忆 facts，
> 三脑检索与记忆中枢不受影响。
> 详见 releases/v2.13.15.md。
- 前端：删除 OfficePage，OfficeHubPage 去除二级导航（单一通用办公入口）；模块启动器与设置面板清理「方案编写」文案
- 后端：删除 internal/office/proposal 与 internal/office/db，移除全部 Proposal* 绑定与 office 模块注册
- 左脑：办公记忆源改为 Hephaestus facts（三脑检索/记忆图谱继续可用）
- 验证：go build/定向测试通过；前端 tsc + 128 例全过；wails build 通过；产物 gaea-v2.13.15.exe

## v2.13.14「通用办公工具/技能面板显示 Word·Excel·PDF 技能」（2026-08-12）
> 通用办公的「技能面板」看不到 docx/xlsx/pdf/pptx：技能只装在仓库 .gaea/skills，
> 而技能索引只扫「当前工作区 .gaea/skills + 用户级 ~/.gaea/skills」，其它工作区
> 均为空。已把四个 ModelScope 文档技能安装到用户级全局目录（任意工作区可见），
> 并在「工具面板」新增「文档技能」分组展示这四个技能。
> 详见 releases/v2.13.14.md。
- 安装：docx / xlsx / pdf / pptx 复制到 ~/.gaea/skills（全局技能，所有工作区可见）
- CapabilitiesPanel：工具面板新增「文档技能」分组（docx/xlsx/pdf/pptx 卡片）
- 验证：全局技能在任意工作区可见（金具厂实测通过）；tsc + 前端 128 例全过；wails build 通过

## v2.13.13「聊天面板左侧栏可折叠 + 悬浮绑定模型卡随面板隐藏」（2026-08-12）
> 聊天页面左侧会话栏原本固定 264px 宽、无法折叠；左下角「聊天」绑定模型卡
> 悬浮在其上方也不随面板变化。新增折叠窄栏（52px，保留展开/新建按钮），
> 折叠态持久化；折叠时悬浮绑定模型卡一并隐藏。
> 详见 releases/v2.13.13.md。
- ChatTopicSidebar：新增 collapsed/onToggle，折叠为窄栏，展开态头部增加折叠按钮
- ChatPage：折叠状态存 localStorage；折叠时隐藏左下角绑定模型卡
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.13.exe

## v2.13.12「通用办公布局优化：精简右侧边栏 + 绑定模型卡随面板折叠」（2026-08-12）
> 通用办公右侧工作区面板删除「成本库」「搜索」两个标签页；左下角「绑定模型卡」
> 原为绝对定位悬浮，会盖住左侧栏底部的「知识库」「MCP 与技能」按钮，且不随
> 面板折叠。改为放入左侧栏底部（与后台任务/事实底座同规则），折叠时随面板隐藏。
> 详见 releases/v2.13.12.md。
- App.tsx：右侧边栏删除成本库/搜索标签与对应面板，同步移除命令面板的搜索入口
- Sidebar.tsx：绑定模型卡移入左侧栏底部，折叠时隐藏，不再遮挡底部导航
- GaeaPage.tsx：移除原来的绝对定位悬浮层
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.12.exe

## v2.13.11「修复记忆中枢·用户画像打不开」（2026-08-12）
> 记忆中枢「用户画像」页打开即白屏：后端无冲突时返回 nil 切片，序列化成
> JSON null，前端直接读 `conflicts.length` 抛 TypeError，被 ErrorBoundary
> 拦截。后端统一保证空数组，前端对 conflicts/tags 做 null 兜底，并加固
> 图谱与聊天记忆页同型风险。
> 详见 releases/v2.13.11.md。
- GaeaProfileConflicts：无冲突时返回 [] 而非 null（DetectConflicts 改 []string{}）
- ProfileLibrary：conflicts/tags 加 ?? [] 兜底（崩溃根因）
- 顺带加固：GaeaWhisperEpisodes 错误路径、GaeaMemoryGraph 空图、GraphView/WhisperMemoryLibrary null 防护
- 回归测试：Go DetectConflicts 非 nil 空切片 + 前端 ProfileLibrary null 用例
- 验证：go build/测试通过；前端 128 例全过；wails build 通过；产物 gaea-v2.13.11.exe

## v2.13.10「修复办公文档处理反复弹 cmd 黑窗」（2026-08-12）
> 通用办公后台做 docx/xlsx 转换、公式重算、报告导出、绘图、桌面自动化时，
> 子进程没加隐藏窗口参数，每执行一次就闪一个 cmd/conhost 黑窗；批量转换时
> 会"不停弹窗"。统一补上 CREATE_NO_WINDOW，全部静默执行。
> 详见 releases/v2.13.10.md。
- docmd.markitdown：`python -m markitdown` 补 HideWindow（弹窗主因，已实测抓到）
- xlsxedit.Recalc、gaea_export.runPython、diagram_gen、whisper 桌面操作（powershell）一并补上
- 验证：go build 通过；前端 127 例全过；wails build 通过；产物 gaea-v2.13.10.exe

## v2.13.9「每一轮的大过程卡默认展开 · 内部卡默认折叠」（2026-08-12）
> 调整通用办公输出显示：输出完成后每一轮的外层大过程卡都默认保持展开，
> 不再只保留最新回合；大过程卡内部的思考卡改为默认折叠，工具卡/工具组
> 保持默认折叠，只留过程文本直接可见。
> 详见 releases/v2.13.9.md。
- ProcessCard：移除 isLatest 限制，每轮大过程卡完成后均保持展开（用户手动折叠过仍尊重）
- InlineReasoning：思考卡默认折叠，点开才看推理内容；工具卡/工具组保持默认折叠
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.9.exe

## v2.13.8「仅最新回合的大过程卡默认展开」（2026-08-12）
> 修正过程卡展开策略：只有最新回合的外层大过程卡默认展开，旧回合自动折叠，
> 避免“所有过程卡全部摊开”；手动展开过的旧卡仍保留。
> 详见 releases/v2.13.8.md。
- ProcessCard 增加 isLatest：完成时仅最新回合保持展开，新回合开始后旧卡折叠
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.8.exe

## v2.13.7「办公上下文窗口修正 + 大上下文处理提示」（2026-08-12）
> 修复“长任务只有执行中、迟迟不出流式”的根因：办公 provider 上下文窗口此前
> 写死 1M，自动压缩阈值高达 80 万 token，会话膨胀到十几万也从不压缩，模型
> 首字要等几分钟，看起来像没输出。窗口调至 256k 让压缩生效，并加处理提示。
> （v2.13.6 的最终回答兜底补渲染一并包含在本版）
> 详见 releases/v2.13.7.md。
- gaea 办公 provider ContextWindow 1_000_000 → 256_000（80% 阈值≈204k，超限自动压缩）
- 前端 RunStatus：运行 ≥20s 且上下文 ≥4 万 token 时显示「处理大上下文中 · N」，不再误判卡死
- 最终回答兜底：turn_done / 看门狗恢复时校验 History 补渲染缺失的最终回答
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.7.exe

## v2.13.5「运行中强制跟随底部，修复“卡住没输出”假象」（2026-08-12）
> 长任务其实一直在输出，但智能滚动一旦被布局变化/折叠动画置为非跟随状态，
> 视图就冻在旧位置，看起来像卡死。运行中改为强制跟随底部，结束恢复智能滚动。
> 详见 releases/v2.13.5.md。
- Transcript：running 时内容变化强制滚到底部；运行结束后保持原智能滚动
  （用户上翻浏览时不再强拉）
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.5.exe

## v2.13.4「外层过程卡完成后默认展开」（2026-08-12）
> 修正 v2.13.3 的方向：要展开的是包裹「过程文本 + 过程卡」的最外层大过程卡
> （ProcessCard），输出完成后默认保持展开；内部工具卡仍默认折叠。
> 详见 releases/v2.13.4.md。
- ProcessCard：running → 完成后不再 setOpen(false)，外层大过程卡默认保持展开
  （用户手动折叠过仍尊重）；v2.13.3 误加的内部卡自动展开已撤回
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.4.exe

## v2.13.3「过程卡完成后默认展开」（2026-08-12）
> 输出完成后的大过程卡（bash/python 等长输出卡片）默认展开，用户手动折叠后
> 不再干预；连续同名工具组含大卡时组也默认展开。
> 详见 releases/v2.13.3.md。
- ToolCard：输出 ≥10 行或输出/参数 ≥2000 字符的大卡，完成（done/error）后自动展开
- ToolGroup：组内任一成员为大卡且完成时，组默认展开
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.3.exe

## v2.13.2「办公过程文件落盘规范：统一 .gaea/work/」（2026-08-12）
> 办公 agent 执行纪律新增「文件落盘规范」：过程/中间文件（提取文本、OCR 页图、
> 脚本、临时图表等）统一写入 .gaea/work/<任务名>/，交付物进 .gaea/exports/，
> 不再与源文件混在工作空间根目录；启动时自动创建这两个目录。
> 详见 releases/v2.13.2.md。
- 办公 agent 提示词新增「文件落盘规范」章节（single_prompt.go）
- GaeaInit 自动创建 .gaea/work/ 与 .gaea/exports/；.gitignore 增加 .gaea/work/
- 验证：go build/vet 干净；wails build 通过；产物 gaea-v2.13.2.exe

## v2.13.1「修复 @PDF 引用注入二进制导致办公输出不可见」（2026-08-12）
> 补丁版：修复通用办公里 @引用 PDF 时，文本提取失败会把 %PDF-1.4 原始二进制
> 塞进模型上下文，导致任务只剩读秒、看不到任何输出。
> 详见 releases/v2.13.1.md。
- @引用 office 文档（PDF/docx）转换失败时注入占位提示（format_convert / summarize_file），
  绝不回退成原始字节；二进制检测扩展到整个读取窗口（此前只查前 8KB，PDF 头无 NUL 会漏检）
- 验证：go build/vet 干净；wails build 通过；产物 gaea-v2.13.1.exe

## v2.13.0「通用办公打磨 + 模型中心优化 + 本地模型实测」（2026-08-12）
> 承接 v2.12.0，本轮聚焦通用办公稳定性与方案写作质量：修复方案分节 100 字
> 截断（流式与批量两处）、docx 读取乱码、前端“运行中”假卡死；配套模型中心
> 优化与 Herdsman/Ollama 本地模型双模式实测。
> 详见 releases/v2.13.0.md。
- 通用办公：方案「生成该节」与批量生成按章节字数目标续写（WordTarget，缺省 800），
  修复原硬编码 100 字截断；docx 读取统一走 ConvertToMarkdown（MarkItDown / 内置
  转换 / 扫描件提示），修复直接读 zip 字节乱码；前端 turn 结束事件丢失兜底
  （停止按钮无条件复位 + 30s GaeaRunning 看门狗），任务完成不再卡「执行中」；
  待办从未推进时 turn_done 自动收尾，不再残留 0/N
- 工具：format_convert 输出父目录自动创建（与 write_file 对齐）+ 回归测试；
  docmd 新增 MarkItDown 通道（docx/pptx），PDF 页数上限与扫描件 OCR 回退保持
- 模型中心：功能绑定/板块细节优化；本地引擎思考参数（enable_thinking /
  chat_template_kwargs）与路由增强
- 语音/角色库：whisper web_search、角色画像文件化等累积修复
- 实测：Herdsman 三模型 20 任务 × 思考/非思考双模式 + Ollama GLM 实测，
  脚本 scripts/test_herdsman_models.go 可复现；详见
  docs/2026-08-12-herdsman-models-evaluation-report.md
- 验证：go build/vet 干净，方案包等测试全绿；前端 127 例全过，tsc + vite build 通过

## v2.12.0「稳定工程 + 成本库多级分类重设计」（2026-08-11）
> 系统根治 WebView2 rAF 冻结引发的「界面卡死 / 点不了 / 角色卡看不见」，
> 成本库升级为按分类分级保存 + 列表/表格双视图，剧照文件化防 IPC 撑爆。
> 详见 releases/v2.12.0.md。
- 稳定性：全应用 30+ Modal 关闭即卸载 + 禁用过渡动画（弹窗关闭后不再残留遮罩卡死）；
  rAF 持续探测 + 8s 冻结看门狗（运行中晚发节流也能降级）；角色卡/启动器/聊天气泡
  入场动画纳入降级名单（修复角色库卡片全看不见）；前端错误/长任务/心跳诊断进 gaea.log
- 成本库：cost_categories 多级分类树 + 条目 category_path 多级保存；默认按信息价体系
  分类（材料细分三级）；分类增删改、改名自动重写子树路径、按路径含子树过滤；
  面板重设计为分类树 + 列表/表格双视图（排序/批量/价格历史）；cost_search/cost_save
  支持分类路径
- 剧照文件化：base64 剧照落盘存路径，>300KB 内联头像截断，根治巨型 base64 撑爆 IPC
- 记忆中枢/办公：成本导入（预览确认 + AI 归一化）、价格源仓库、价格历史与跳幅识别、
  知识导入/版本历史/查重合并、画像库、办公记忆查重合并、记忆图谱增强；后端导入链路
  加固与批量测试覆盖
- 验证：go build/vet 干净，go test 相关包全绿；前端 127 例全过，tsc + vite build 通过

## v2.11.0「通用办公大优化：四大库能力闭环 + 本地语义检索栈」（2026-08-10）
> 承接 v2.10.1 开工前 P0，落地调研蒸馏出的 P1 三项：工作区全文搜索、
> 常用资料固定装配、任务模板库——把「开工前」从「能搜到文件」升级为
> 「能搜到内容、能钉住常用资料、能一键发起常见任务」。
- 工作区全文搜索（轻量 RAG）：右侧新增「搜索」标签页 + 命令面板入口，在
  docx/xlsx/pdf/md/txt/csv 等正文里定位关键词（中文 bigram 分词 + TF-IDF 打分），
  返回命中片段，可预览或一键 @ 引用；正文按 (路径, mtime, size) 缓存，重复搜索
  不重复解析；.gaea/sessions|archive|cache 等噪音跳过，.gaea/exports 交付产物可索引
- 常用资料固定装配（P1-②）：资料面板每行图钉「固定为常用资料」，清单持久化到
  <工作区>/.gaea/pinned.json（去重、限长、防路径逃逸）；新会话启动时自动把固定
  清单装进系统提示词——文本类附正文摘要、办公文档列名按需读取（装配而非灌输）
- 任务模板库（P1-③）：欢迎页新增「任务模板」区（周报/会议纪要/成本测算/方案大纲/
  数据分析/文档转换/报告拼装/演示文稿），一键填入结构化指令；同源模板同时落盘为
  .gaea/commands/*.md（幂等、不覆盖用户文件），/ 菜单与 Submit 直接可用
- 修复：办公引擎命令发现改为基于工作区根（CommandDirsAt），切换工作空间后
  模板与自定义命令跟随新工作区，与技能/记忆的发现口径一致
- 记忆中枢关联（本轮打通）：记忆中枢首页「三脑检索」并入工作区全文搜索——
  文件命中以「文件」徽标同框展示，可预览 / 一键 @ 引用（回到办公板块自动插入
  输入框）；首页新增「项目资料」卡片（固定常用文件计数）与对应库面板（已固定/
  可固定资料管理、预览、固定/取消，与办公面板共用 .gaea/pinned.json）；固定
  资料作为 material 节点进入记忆 3D 图谱（天蓝色，图谱过滤器可单独开关）
- 记忆生命周期与可控（P1-④/⑤）：facts 新增 last_used_at / source_session /
  source_message（SchemaV3 迁移，修订不重置使用时间）；memory_get 读取即 Touch，
  逐轮记忆注入改为「关键词 + 时间 + 高频」排序 + 预算压缩（RecallBlock 默认
  800 rune，画像 600 rune），procedural 常驻、episodic 按标签触发、相关事实按
  命中带入；记忆面板顶部新增「记忆开关」（持久化配置 + 引擎重建立即生效），
  事实卡片展示来源会话与最近使用时间
- 方法论自动候选（P1-⑥）：记忆面板「建议」标签接入真实后端——从 procedural
  记忆按主题词聚类（≥2 条同主题 → 提议 workflow-<主题> 技能，附证据与正文），
  接受后经既有技能沉淀通道固化并热加载；记忆候选保持为空（自动做梦已直接入库，
  宁缺毋滥）
- 验证：后端新增 wssearch/pins 包测试 + boot 集成测试（工作区命令发现），
  以及记忆中枢关联测试（固定数统计 + 图谱 material 节点）、记忆生命周期/压缩
  注入/技能聚类测试；前端新增 WorkspaceSearchPanel 2 例 + 资料固定 1 例 +
  项目资料库 1 例，vitest 93 例全过，tsc 与 vite build 通过

### 本轮追加：成本库/知识库/办公记忆/工作区文件大优化
- **成本库打通与功能深化**：cost_search/cost_save 内置工具（读/写 Hephaestus.db
  cost_entries，同名 UPSERT）；cost-estimate 模板「先查后沉淀」；办公右侧成本库 Tab
  （浏览/搜索/一键结构化引用）；产物面板「沉淀到成本库」；导入 xlsx/csv + AI 归一化
  （预览确认，无确认不落库）；编辑/删除/批量操作；价格源订阅 + 手动/定时抓取
  （四川造价信息网实测 24 行解析）+ 价格历史 + 单期跳幅 ±20% 异常识别
- **知识库大优化**：md/txt/docx/pdf/xlsx/csv 导入 + AI 多主题拆分（预览确认）；
  批量管理 + 审核流（草稿→现行、审核人留档）+ 版本历史（内容变化自动留档）+ 批量
  导出 Markdown；查重（CJK 二元组 Dice）与相似条目一键合并
- **办公记忆**：两两查重 + 一键合并（textsim 共享相似度），自动做梦产生的同名变体
  可一键归并
- **工作区文件语义索引**：fileindex 扫描 md/txt/csv/docx/xlsx/pdf/pptx 提取正文
  （pptx 零依赖解 zip 读 slide XML），bge-m3 向量化入库（semantic_vectors kind=file，
  内容感知增量重建 + 10 分钟自动维护），搜索面板「语义」模式 + 记忆中枢统一搜索
  并入「语义·文件」命中
- **本地语义检索栈（四库统一）**：子串召回 →（不足时）bge-m3 语义召回 → BM25 排序 →
  （>8 条）bge-reranker-v2-m3 精排，任一层失败自动降级；跨库统一语义检索
  （GaeaSemanticSearch，成本/知识/办公记忆）；Herdsman bge-m3 / bge-reranker-v2-m3
  实测端到端通过，全部本地零 token
- 验证：go test 全量 + go vet 干净；前端 111 例全过，tsc 与 vite build 通过；
  真实 Herdsman 语义召回/精排与四川价格源抓取实测通过；产物 gaea-v2.11.0.exe

## v2.10.1「开工前 · 交付收尾 · 记忆自进化」（2026-08-10）
> 把办公体验从「下完指令等结果」补成完整闭环：开工前先出计划、资料一键引用；
> 对话里所有文件引用可点击预览，交付物收尾成卡片；会话结束后自动「做梦」整理记忆。
- 输出文件全量可点击预览：mdast（remark 插件）识别绝对/相对/裸文件名（参考
  openclaw/llama.cpp 的 AST 插件架构），聊天正文、流式尾部、工具输出、ask 提问均可点击，
  代码块/已有链接/公式天然跳过；后端支持裸文件名自动解析到 exports 等常见输出目录
- 交付收尾：助手消息尾部「交付文件」卡片（缩略图/图标 + 文件名 + 扩展名，预览/复制路径/
  外部打开/定位）；右侧「会话产物」面板（本次会话交付文件清单 + 溯源跳转到生成消息）；
  预览内编辑后自动标「已更新」并刷新文件树；粘贴表格即数据（CSV/TSV 识别 → Markdown 表格）
- 记忆自进化（自动做梦）：轮次成功后后台归纳对话 → 提炼稳定事实/偏好/笔记写入长期记忆
  （按 name 去重、Kimi 二问纪律、单飞、失败静默）；整理结果在聊天内通知
- 开工前：@ 文件引用增强（工作区跨目录搜索 + 最近使用 + 扩展名徽标）；右侧「资料」面板
  （docx/xlsx/pptx/pdf/md/csv 按类型分组、最新在前、一键 @ 引用进输入框）；开工前计划卡片
  （非简单任务先生成计划，用户「确认执行 / 先调整」，auto_plan 可配置，默认开启）
- 三轮竞品优点蒸馏文档：交付收尾 UX、记忆与自进化、开工前上下文；调用链核对文档
  （docs/gaea2/office-model-tool-call-chains.md）
- 发布说明：产物 gaea-v2.10.1.exe（Windows x64），桌面端同步；go test（agent/control/boot/
  app 全量）+ vitest 89 例全过，tsc 与 wails build 通过

## v2.10.0「正式发布 · 通用办公三阶段闭环」（2026-08-09）
> 自 v2.6.9 之后最大一次发布：把「办公前期的文件解析、中期的 Word/Excel
> 编辑、后期的文件输出与预览」串成一条完整闭环，并完成桌面端体验重构与
> 真实模型端到端验收。本版本包含 v2.7.0–v2.9.4 的全部累积特性。
- 前期解析：docx/xlsx/pdf → Markdown 提速（docx 约 12x / PDF 约 4x）；
  扫描件 OCR 四级管线（OvisOCR2 常驻服务 → RapidOCR → WinRT → 本地视觉模型），
  粘贴图片「提取文字/识图」双入口；事实底座（fact_add/list/clear + 侧栏面板 +
  一键沉淀长期记忆，后续对话自动加载）
- 中期编辑：Word 框选即改 + 修订模式逐条接受/拒绝 + diff 回看；Excel 单元格级
  预览/编辑（双击改值/写公式、fx 栏、AI 指令编辑、插入/删除行与列、LibreOffice
  公式重算、数字格式/冻结表头/合并单元格）；跨应用联动（Excel 数据 → 图表 →
  嵌入 Word/PPT 并随数据同步刷新）
- 后期输出：统一交付出口（对话成果一键导出 docx/pptx/xlsx/md）、模板与样式体系、
  成本测算模板（市政道路改造工程，公式全联动 + 原生图表，真实 DeepSeek 绑定
  自主生成端到端验收通过）
- 桌面端体验：Codex 式文件预览布局（右侧文件树、主区域可拖宽预览、Esc/按钮
  快捷收起）、文件就地编辑不跳出对话
- 发布说明：产物 gaea-v2.10.0.exe（Windows x64，约 40MB），桌面端同步；
  go test / vitest 59 例全过，tsc 类型检查与 wails build 通过

## v2.9.4「表格列操作：插入/删除列」（2026-08-09）
> 按用户要求：排序/筛选/条件格式不在 gaea 内做预览层实现（不写回文件），
> 后期交给 Excel/WPS 专业软件处理；本次仅落地会真实写文件的列操作。
- 插入/删除列：点击列字母选中整列，工具栏出现「← 插列 / → 插列 / 删除列」
  （删除二次确认），后端 excelize InsertCols/RemoveCol + 重算刷新，改动落盘
- 新增后端 GaeaXlsxColOps；列选择与单元格选择互斥
- 清理：移除本轮试验性的排序宏（LibreOffice）、AutoFilter 写入与条件格式渲染
- 验证：Go 单测（插列/删列错位、非法操作/非法引用）+ 前端列操作用例；
  go test / vitest 59 例全过；桌面 gaea.exe 已重建同步

## v2.9.3「表格行操作：插入/删除行」（2026-08-09）
> 按用户要求补齐高频表格操作——行级插入与删除。
- 预览头部新增「↑ 插行 / ↓ 插行 / 删除行」：基于选中单元格所在行，
  插入空行或删除整行；删除需二次确认（点击后变为「确认删除」）
- 后端新增 GaeaXlsxRowOps（excelize InsertRows/RemoveRow 平移同表公式与合并区，
  随后 LibreOffice 重算刷新结果并重渲染预览）
- 验证：Go 单测（插入错位/删除上移/非法操作/非法引用）+ 前端插行用例；
  go test / vitest 58 例全过；桌面 gaea.exe 已重建同步

## v2.9.2「公式结果显示 + 表格功能补强」（2026-08-09）
> 修复公式格只显示 fx 无结果的问题，并补上两类高频表格能力。
- 公式结果：根因是 openpyxl 生成的文件公式没有缓存值（未重算），预览渲染时
  只剩 fx 标记。现在 GaeaPreview 自动检测「无缓存值公式」→ LibreOffice 重算后
  再渲染；预览头部新增「重算公式」按钮可手动刷新；新增 GaeaXlsxRecalc 接口
- 数字格式：NumFmt 已提取但前端未应用，现按格式显示千分位/百分比/货币/小数位
  （内置编号 1-11、44-47 及常见自定义格式），公式结果也按单元格格式呈现
- 冻结表头：提取 sheet 冻结窗格（GetPanes），预览中冻结行 sticky 固定，
  滚动数据时表头不跑
- 验证：新增 NeedsRecalc 单测 + 冒烟回归（真实成本表 51 个公式全部有结果，
  预览自动重算 1.5s）+ 前端数字格式用例；go test / vitest 57 例全过；
  桌面 gaea.exe 已重建同步

## v2.9.1「预览拖拽修复 + 表格就地编辑」（2026-08-09）
> 修复 Codex 式预览的两个实测问题：拖拽分割条失效、表格只能看不能改。
- 修复拖拽：预览宽度上下限计算写反导致宽度卡死（恒为最大），改回
  320–1100px 自由拖拽；分割条命中区 6px→10px，悬停/拖拽高亮
- 表格 Excel 式就地编辑：双击单元格直接输入（Enter 保存 / Esc 取消），
  上方 fx 公式栏也可输入值或 `=公式` 回车写回；纯数字按数值写入保持可计算，
  等号开头写公式，写回后 LibreOffice 重算并即时刷新预览
- 新增后端 GaeaXlsxSetCell 直写接口（excelize 写值/公式 + 重算 + 预览）
- 验证：新增 Go 单测（写值/写公式/非法引用）+ 前端双击编辑用例；
  go test / tsc / vitest 56 例全过；桌面 gaea.exe 已重建同步

## v2.9.0「Codex 式文件预览布局」（2026-08-09）
> 参照 Codex 桌面端交互改造办公文件查看：右侧面板收敛为「增强文件树」，
> 点击文件后原右侧面板隐藏，预览在聊天区右侧主区域展开（全高、宽度可拖），
> 聊天与预览互不遮挡，可边对话边看文件。
- 布局：点文件 → 树收起 → 主区域出现预览面板，聊天区保留在左侧；
  预览与聊天之间加 6px 拖拽分割条，宽度 320–1100px 自由调整并持久化（localStorage）
- 预览头部：新增「文件」返回按钮（回到文件树）、文件名/大小、定位/外部打开/关闭；
  Esc 或工具栏面板按钮均可收起预览回到树
- 文件树增强：docx 蓝 / xlsx 绿 / pptx 橙 / pdf 红 / 图片紫 按扩展名着色，
  文件夹优先 + 自然排序；新增手动刷新按钮（强制重载目录）
- 清理：移除已被树内嵌预览取代的「最大化面板」逻辑与重复的“查看文件变更”按钮
- 验证：tsc --noEmit 通过；vitest 55 例全过；wails build 成功，桌面 gaea.exe 已同步

## v2.8.9「自主生成验收 · gaea 自己生成成本测算表」（2026-08-09）
> 让 gaea 智能体用真实模型（办公绑定 deepseek/deepseek-v4-flash）自主生成
> 成本测算表：无头驱动 Controller.Run + bridge provider + 完整工具链，验证
> 产物含公式与原生图表——桌面端已构建，办公功能绑定已生效。
- 新增 TestGaeaSelfGenerateCost（GAEA_SELFGEN 门控）：读取 ~/.gaea_config.json
  办公功能绑定路由引擎/模型；照桌面端接线 engineMgr + ai.Client + bridge；
  提示词要求 openpyxl 建表、公式联动、原生饼图/柱状图、LibreOffice 重算
- 实测结果（900s）：智能体自主完成环境检查→写脚本→生成→重算→自检；
  产物 3 工作表（成本测算/费用汇总/编制说明）、35 个公式、2 个原生图表，
  openpyxl 断言全过；路由日志「来源 feature」确认走了用户绑定
- 顺手修复既有测试全局污染：TestGaeaBootBuild 注入 gaeaConfig.SetLoader
  未恢复，会影响同包后续 boot.Build（真实模型跑测试时被 agent 诊断发现）；
  已 defer 恢复；自生成测试产物自动复制到 .gaea/exports/
- 验证：go vet/test ./internal/app 全绿；桌面端 gaea.exe 已重新构建并
  同步桌面（含全部办公能力与绑定）

## v2.8.8「成本测算模板 · 真实样例交付」（2026-08-09）
> 新增可复用成本测算生成脚本 cost_estimate.py（openpyxl）：市政道路改造工程
> 成本测算工作簿——按费用构成（人工/材料/机械/企管/规费/利润/税金）建表，
> 公式全联动（数量×单价=合价、小计、占比、含税总价、综合单价），原生饼图+
> 柱状图，LibreOffice 重算后带缓存值。
- 交付物：.gaea/exports/市政道路改造工程成本测算.xlsx（含税总价
  9,992,806.15 元，综合单价 462.63 元/m²，53 个公式 0 错误，2 个原生图表）
- 验证：openpyxl 数值/公式/图表断言、PDF 渲染文本核验（表/图表标题/编制说明
  完整）、gaea GaeaPreview 单元格预览管线读取正常（GAEA_SMOKE_COST 门控）

## v2.8.7「P2 跨应用联动 · Excel 数据 → 图表 → 嵌入 Word/PPT」（2026-08-09）
> 联动闭环：xlsx 数据一键生成 matplotlib 图表并嵌入 docx（报告模板：标题+
> 图表+数据表）或 pptx（图表页+数据明细页）；数据源更新后重新导出即刷新图表，
> 实现「图表与数据保持同步」（避免脆弱的 OLE 活链接）。
- 新增 internal/office/crosslink：xlsx 数据提取（自动/显式区域 A1:B6，表头+
  数值列解析、千分位清洗）、matplotlib 图表生成（bar/line/pie/scatter，中文
  字体、无窗口执行）、docx/pptx 嵌入 spec 构建（docx 含数据表，pptx 图表页+
  数据明细页）
- create_pptx.py 新增每页 image 支持（图表页居中嵌入）；脚本镜像同步
- 新增 GaeaCrossEmbed 绑定：格式/图表类型校验、产物与图表落到 .gaea/exports/
- 前端 XlsxPreview 新增「图表→Word」「图表→PPT」按钮：取当前工作表数据一键
  联动，成功后自动定位产物
- 验证：crosslink 单测（数据提取/区域裁剪/spec 构建）+ 真实 matplotlib 图表
  冒烟（23KB PNG，0.5s）+ docx/pptx 嵌入冒烟（产物合法 zip 且含 media 图片）；
  go test ./... 全绿、vet 干净；tsc + vite build 通过；vitest 55 例全过

## v2.8.6「打磨联调 · 边界加固 + 端到端走查」（2026-08-09）
> 把前几轮的能力串成一条全流程验证：解析 → 修订式编辑 → 接受修订 →
> 提取成果 → 模板化多形态交付，并补齐三处常见边界。
- 边界加固：
  - docxedit：表格单元格（w:tbl>w:tr>w:tc>w:p）内修订与接受全链路（合同场景）；
  - xlsxedit：transform 公式保留 $ 绝对引用、跳过函数名（LOG10/ROUND）误判，
    行引用调整专项用例；
  - 交付出口：标题含 Windows 非法字符（< > : " / ? 等）时清洗且保留中文
- 端到端：
  - 新增 TestOfficeFullPipeline（纯 Go）：编辑→接受→提取→导出 xlsx/md；
  - 新增真实文档全链路走查（GAEA_SMOKE_PIPELINE 门控）：26MB 方案文档
    编辑→接受→提取→报告模板导出 docx，全程 4s 完成、产物内容正确
- 验证：go test ./... 全绿、vet 干净；tsc + vite build 通过；vitest 54 例全过

## v2.8.5「后期输出 · 模板与样式体系」（2026-08-09）
> P1 模板库落地：统一交付出口支持 通用/公文/报告/合同 四套版式预设——
> 字体、标题色/标题字体、表头底色按模板切换，事实底座一键出对应样式的 Word。
- create_docx.py 新增 --template 预设：通用（宋体+深蓝标题）、公文（仿宋正文+
  黑体标题）、报告（微软雅黑+蓝标题）、合同（宋体+黑色标题）；封面标题色、
  标题 run、表头底色全部参数化贯穿
- GaeaExportDeliverable 新增 Template 字段（默认通用，非法值明确报错），
  docx 出口透传 --template
- 事实底座侧栏「导出报告」升级为「模板选择 + 导出」：报告（封面+目录）/
  公文 / 合同 / 通用，导出后自动定位文件
- 验证：四套模板实测生成（颜色/字体抽查：通用 1F3864/宋体、报告 2E74B5/
  微软雅黑、公文 000000/仿宋）；docx 导出冒烟带模板通过；go test ./... 全绿；
  tsc + vite build 通过；vitest 54 例全过；技能镜像 ~/.codex/skills 已同步

## v2.8.4「中期编辑 · 修订接受/拒绝 + diff 回看」（2026-08-09）
> P1 信任机制落地：框选即改的修订不再只能去 Word 里处理——预览里直接
> 「接受修订 / 拒绝修订」一键扁平化（按作者 gaea AI，不动他人修订），
> 修订样式本身即 diff 回看（原文划除 + 新文插入）。
- docxedit 新增 AcceptChanges / RejectChanges：字节级扁平化 w:del/w:ins
  （接受=删 w:del 留 w:ins；拒绝=删 w:ins 还原 delText→t 并保留 run 格式），
  只处理指定作者、跳过嵌套/重叠、无修订时明确报错
- 新增 GaeaDocxAcceptChanges 绑定（accept=true/false），返回更新预览
- 前端 DocxPreview：渲染后自动检测 ins/del 修订，标题栏出现「接受修订 /
  拒绝修订」按钮（加载中禁用），操作后重渲染并提示
- 验证：docxedit 接受/拒绝/无修订/他人修订不动四类单测 + 真实 26MB 文档
  「修订→接受」冒烟（1.7s 完成、XML 合法、标记清空）；go test ./... 全绿；
  tsc + vite build 通过；vitest 54 例全过

## v2.8.3「后期输出 · 事实底座 → 多形态交付统一出口」（2026-08-09）
> P0-④ 落地：统一交付管线「受控 Markdown → docx / pptx / xlsx / md」——
> 事实底座与对话成果一稿多用，多形态基于同一底座生成、彼此一致。
- 新增 GaeaExportDeliverable 绑定：格式校验、标题自动推导、时间戳防重名、
  非法字符清洗（保留中文）、交付到 .gaea/exports/ 并返回相对路径
- docx：复用 create_docx.py（python-docx，封面/目录/页眉页脚/页码参数化）；
  pptx：Markdown → slides spec（# 标题开新页、要点/表格行转要点）→ create_pptx.py；
  xlsx：Go 侧 excelize 直接提取 Markdown 表格为工作表（无表格时正文入 Sheet1）；
  md：直写；python 子进程带超时与 stderr 捕获
- 前端两个入口：会话工具栏新增「导出 Word」（对话成果一键交付，成功后自动定位）；
  事实底座侧栏新增「导出报告」（封面+目录，基于同一底座）
- 验证：md/xlsx/校验单测（纯 Go）+ docx/pptx 真实脚本冒烟（GAEA_SMOKE_EXPORT，
  实测通过、产物为合法 zip）；go test ./... 全绿；tsc + vite build 通过；
  vitest 54 例全过

## v2.8.2「中期编辑 · Excel 单元格级操作」（2026-08-09）
> P0-③ 闭环：在单元格预览上「选中单元格 → 指令（求和/平均/拆分列/清洗/替换/
> 自定义）→ AI 规划操作 → excelize 执行 → LibreOffice 重算公式 → 重渲染」，
> 公式可校验、结果可检查。
- 新增 internal/office/xlsxedit：AI 操作集（set_formula/set_value/fill_range/
  transform/replace/split_column/clean）校验与执行；transform 逐行公式自动调整
  相对行引用（跳过 $ 绝对引用与函数名）；操作数量/单元格数上限防滥用
- 新增 ai.XlsxEditOps：表格上下文（工作表/表头/抽样数据）+ 指令 → 严格 JSON 操作集；
  GaeaXlsxEdit 绑定走 office 功能绑定路由，闭环后返回更新预览与逐条摘要
- 公式重算：复用技能环境 recalc.py（LibreOffice 宏，自动探测 soffice），
  best-effort——重算不可用时编辑仍生效并提示；真实重算冒烟 1.75s 通过
- 前端 XlsxPreview 新增选中编辑工具栏：四预设（求和/平均值/拆分列/清洗）+ 自定义
  指令，执行后重渲染并展示应用摘要
- 验证：xlsxedit 单测（公式/逐行变换/填充/替换/拆分/清洗/错误分支）+ app 校验
  单测 + 真实 LibreOffice 重算冒烟；go test ./... 全绿；tsc + vite build 通过；
  vitest 53 例全过

## v2.8.1「中期编辑 · Excel 单元格级预览」（2026-08-09）
> P0-③ 第一步：xlsx 预览从「转 Markdown 弱表格」升级为「单元格级保真视图」——
> sheet 切换、公式标识与公式栏、样式近似还原、合并单元格与列宽，为后续
> 「选中区域 → 指令（公式/清洗/透视）→ 写回」的单元格编辑打底。
- 新增 internal/office/xlsxpreview：excelize v2.11.0（已修复 2026 年三个 CVE）
  提取结构化单元格 JSON（值/公式/类型/样式/合并/列宽）；超大表格截断到
  2000 行 × 100 列并标记；图表/宏表自动跳过；.xls 保持原 markdown 兜底
- GaeaPreview 对 .xlsx 返回 kind=xlsx（body 为 JSON），wire 契约复用原字段
- 前端新增 XlsxPreview 组件：sheet 标签切换、公式栏（点击 fx 单元格显示公式）、
  表头行列冻结、样式（加粗/填充/对齐/边框/颜色）、合并单元格 colspan/rowspan、
  列宽近似；侧栏与弹层两个预览入口接入
- 验证：xlsxpreview 单测（样式/公式/合并/列宽/多 sheet/截断）+ app 层
  GaeaPreview 端到端；go test ./... 全绿；tsc + vite build 通过；vitest 52 例全过

## v2.8.0「中期编辑 · Word 框选即改 / 修订模式」（2026-08-09）
> P0「中期编辑 · Word 框选即改」落地：在保真预览底座上，选中文字 → 指令
> （AI 四预设 + 自定义）→ AI 生成替换 → 以 Word 修订模式（w:del + w:ins）
> 就地写入并重渲染，其余内容与版式零扰动。
- 新增 internal/office/docxedit：字节级 XML 手术（不重建 OOXML）定位选中文本并执行
  修订式替换；保留 run rPr 格式、应用实体转义、空白折叠匹配兜底、优先修订非 hyperlink
  段落（TOC 目录项不被 docx-preview 映射 ins/del）；选区包含图片/制表符/换行等特殊
  格式时明确拒绝
- 新增 GaeaOfficeEditText 绑定（办公向改写提示词：关键信息不变、纯文本输出，走 office
  功能绑定路由）+ GaeaDocxApplyEdit（修订写入并返回更新预览，标记作者「gaea AI」）
- 前端 DocxPreview 新增「框选即改」工具栏：选中文字后出现，四类快捷指令
  （润色/精简/翻译中文/扩写）+ 自定义指令输入，diff 预览（原文 → AI 替换）后
  「应用修订 / 放弃」；应用后修订样式重渲染
- 验证：docxedit 单测（单 run 拆分/跨 run 切分/XML 转义/空白折叠兜底/超链接优先/
  特殊格式拒绝）+ 真实 26MB 方案文档冒烟（0.9s 修订、合法回写、docx-preview 映射
  <ins>/<del>）；go test ./... 全绿；tsc + vite build 通过；vitest 49 例全过

## v2.7.9「通用办公 · 粘贴图片一键提取文字」（2026-08-09）

> 便捷入口闭环：截图/扫描件贴进对话后，点图片附件的「提取文字」即可用本地 OvisOCR2
> 常驻服务抽出图中文字，作为用户消息发给助手——不用再靠模型读图，离线、秒级。
- 新增 GaeaOCRText 绑定（docmd.OCRImageText）：复用常驻 llama-server（自动拉起/共享端口），
  图片不存在或服务不可用时给出明确错误
- Composer 图片附件新增「提取文字」按钮（FileText 图标，位于「识图」旁），识别中显示 spinner，
  完成后以【图片文字提取：文件名】发送给助手；与既有「识图」（本地视觉模型）互补——
  识图给描述、提取文字给原文
- 验证：docmd 新增 OCRImageText 端到端测试（GAEA_TEST_OCR_IMAGE 门控，实测 0.76s 返回
  「项目周报…营收 120 万元…」）；go test ./... 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.8「扫描件 OCR · 常驻服务 + format_convert 闭环」（2026-08-09）

> 多页扫描提速 + 内置转换器闭环：llama-server 常驻（按需拉起、共享 8137 端口），
> 多页 PDF 不再每页重载模型（单页约 5s → 整页 2.4s）；format_convert/文件预览的扫描件
> 兜底从「提示装 tesseract」改为直连本地 OvisOCR2 服务，tesseract 降为兜底。
- 常驻服务：ocr_local.py 与 docmd 共用同一端口（GAEA_OCR_URL/GAEA_OCR_PORT 可覆盖），
  健康检查发现未运行即按需拉起 llama-server（Vulkan，隐藏窗口），多页逐页复用；
  路径用 GAEA_OCR_DIR / GAEA_OCR_LLAMA / GAEA_OCR_MODEL / GAEA_OCR_MMPROJ 覆盖
- format_convert 闭环：docmd.ocrPDF 先试 OvisOCR2 服务（按页 POST），不可用时退回
  tesseract；findPdftoppm 探测真实 poppler exe（本机 PATH 里的 .cmd 包装器指向
  不存在路径，直接执行会失败，回退 codex 运行时自带的 pdftoppm.exe）
- 修复扫描件被误当文本：extractRawText 曾把 PDF 对象字典/trailer/ICC/图像流的可打印
  垃圾当正文返回，OCR 永远不触发——新增 stripNonTextStreams（关键字感知流扫描，只保留
  含 BT/ET/Tj 的文本流）+ 内容质量门槛（按空白分词、拒绝高熵符号串），扫描件正确落入 OCR
- 文本提取增强：extractPDFText 支持 TJ 数组的十六进制字符串（UTF-16BE，Word/LibreOffice
  中文 PDF 常见），带 BOM/无 BOM/ASCII 自动判别
- 验证：docmd 新增 httptest 客户端、hex 解码、流剥离、端到端扫描件（env 门控）测试；
  go test ./... 全绿；pdf 技能已同步 ~/.codex/skills 镜像

## v2.7.7「扫描件 OCR · OvisOCR2 本地文档解析」（2026-08-09）

> 按用户建议安装专用本地 OCR 模型：OvisOCR2（0.8B 端到端文档解析，Qwen3.5-0.8B 后训练，
> OmniDocBench 96.58，GGUF + llama.cpp Vulkan，Apache-2.0）成为扫描件 OCR 首选通道，
> 直接输出 Markdown（含 LaTeX 公式与表格），中文印刷体/表格识别显著强于 WinRT。
- 安装：llama.cpp b10333（Vulkan，适配 Radeon 8060S 核显）+ OvisOCR2-Q5_K_M（578MB）+
  mmproj-F16（205MB），落位 C:\AI\gaea-ocr（约 850MB）
- ocr_local.py 新增 --mode ovis 与自动链路：auto = OvisOCR2 → RapidOCR → WinRT → 本地视觉模型，
  文本过短/为空时逐级降级；路径可用 GAEA_OCR_DIR / GAEA_OCR_LLAMA / GAEA_OCR_MODEL /
  GAEA_OCR_MMPROJ 环境变量覆盖
- 顺带补齐 RapidOCR（PP-OCR 转 ONNX，隔离 venv C:\AI\gaea-ocr-env）作为 Ovis 缺失时的离线兜底
- 实测：真实扫描 PDF（微软雅黑渲染后嵌图）→ Ovis 识别「项目周报：本周完成 8 项需求 /
  营收 120 万元，同比增长 18% / 修复目标：砷 ≤ 60 mg/kg（GB 36600-2018）」，
  公式转 LaTeX、单页约 5s（含模型加载）；WinRT / RapidOCR 通道同步验证
- 验证：pdf 技能与 ~/.codex/skills 镜像同步；go build/test 不受影响

## v2.7.6「性能专项 · 大文件转换提速」（2026-08-09）

> P1「性能专项」落地：先用基准量化 format_convert 热点再针对性优化——
> docx→Markdown 百页级文件提速约 12 倍（69ms→5.8ms），PDF 提速约 4 倍（5.0ms→1.2ms）。
- 根因：docxToMarkdown 主循环每次迭代都对剩余全文搜索 `<w:p>` 与 `<w:p ` 两种形式，
  带属性形式在多数文档中不存在，导致 O(n²) 全文扫描（CPU profile 显示 findOpenTag 占 83%）
- 修复：新增 nextOpenTag 单趟扫描（按 `<` 推进 + 标签名边界判断），一次定位最早的段落/表格标签；
  PDF 页数统计从逐 rune 转 string 的热循环改为 strings.Count("/Type /Page")；
  xlsx 工作表按名建索引替代逐 sheet 全文件扫描，sharedStrings 索引解析改用 strconv.Atoi
- 基准（新增 docmd bench_test.go：gooxml 构造百页 docx / 千行 xlsx / 1500 页 PDF）：
  docx 300 段 + 20 表 69ms→5.8ms（约 12x）；PDF 1500 页 5.0ms→1.2ms（约 4x）；xlsx 1000x10 微调
- 验证：docmd 既有用例全部通过（表格 round-trip / 属性段落），go test ./... 全绿

## v2.7.5「方案校验 · 原文定位 + 整改建议」（2026-08-09）

> P1「方案校验规则引擎」增强：全面检查从「报问题」升级为「指出在哪、怎么改」——
> 每条发现带整改建议与原文定位（章节 + 上下文摘录 + 一键跳转）。
- CheckItem 新增 Suggestion（整改建议）与 Locations（原文定位：sectionId/excerpt/offset）：
  废标条款响应、数据一致性（缺失事实/工期冲突/暗标单位）、跨章节重复、暗标格式（加粗/删除线/emoji）、
  规范引用（未引用/编号存疑）逐条补齐可照做的整改建议
- 原文定位：locateNeedle 忽略空白差异在章节内定位命中，excerptAround 按 rune 截取上下文摘录
  （不切坏 UTF-8，前后省略号）；重复检测给出主章节与重复章节两处定位；AI 评分覆盖检查的
  suggestion 同步透传到整改建议
- 前端：校验报告新增「整改建议」行与原文定位 chips（摘录 + 定位按钮，点击跳转对应章节）
- 验证：新增 check_advice 测试（建议/定位/双章节/覆盖建议透传/摘录 UTF-8 安全）；
  go test 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.4「事实底座 · 长期记忆沉淀」（2026-08-09）

> P0「记忆与自进化」落地：事实底座不再止步于会话内——一键把交付前沉淀的事实写入长期记忆，
> 后续对话自动加载，越用越懂你、不用反复交代。
- 新增 GaeaFactBasePromote 绑定 + 侧栏「沉淀为长期记忆」按钮：把当前会话事实底座逐条写入
  memory 存储（kind=semantic），按稳定 ASCII slug 去重（中文 key 走 hash 兜底），
  重复沉淀同 key 原位更新不产生重复条目；preference 分类映射为用户画像（type=user），其余为项目事实
- 面板按钮点击后 toast 反馈沉淀条数；事实底座本身保留，可继续编辑/清空
- 验证：新增 promote 单测（写入/去重更新/中文 slug 稳定性/分类映射/单行摘要）；go test 全绿、
  tsc + vite build 通过、vitest 47 例全过

## v2.7.3「事实底座 · 一稿多用」（2026-08-09）

> P0「多形态成果交付」核心闭环：通用办公引入 任务 → 事实底座 → 多形态交付 的默认工作流，
> 交付物（docx/pptx/xlsx/图表）基于同一事实底座生成，跨交付物保持一致、可回看、可复制。
- 新增 fact_add / fact_list / fact_clear 三个会话级事实底座工具（ExtraTool）：
  fact_add 按 key 沉淀/修正事实（含来源与分类，空值即删除），fact_list 输出 Markdown 表格，
  fact_clear 开启新任务时清空；事实按会话持久化到 .gaea/sessions/<session>-facts.json
- 侧栏新增「事实底座」面板：事实列表（key/值/来源）+ 复制 Markdown + 清空，随 fact_* 工具结果
  与回合结束实时刷新；删除会话时事实底座同步清理
- 办公提示词新增「事实底座（一稿多用）」纪律：交付任务先沉淀后交付，交付物基于同一底座生成，
  事实有更新先 fact_add 修正再重新生成，新任务先 fact_clear 隔离上一任务事实
- 验证：新增 factbase 包单测（upsert/清空/Markdown 转义/round-trip/路径）+ app 侧工具与绑定测试；
  go build/test 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.1「办公三件套核心闭环修复」（2026-08-09）

> 触及核心：修复 Word/Excel/PDF 三个工具的真实断点，端到端实测打通。
> 此前通用办公的 docx 技能依赖未安装的 node docx-js 与 pandoc，Excel 公式重算在 Windows
> 上因 soffice 不在 PATH 而失败——这三个断点全部修复并补回归测试。
- format_convert（docmd）：修复 docx→Markdown 表格解析把 `<w:tcPr>` 等 XML 当文本的 bug；
  `findWt` 偏移错位导致段落/单元格文本错乱；兼容带属性段落（w14:paraId）与 `<w:tabs>` 内嵌标签
  （新增 TestConvertDocxTableRoundTrip / TestConvertDocxAttrParagraphs 回归测试）
- docx 技能：新增 scripts/create_docx.py（python-docx，环境已装）——Markdown/JSON spec → 排版 Word
  （A4/Letter、封面、目录域、标题层级、表格、列表、页眉页脚、页码、图片），替代未装的 docx-js 成为主路径；
  读取兜底改为 gaea 内置 format_convert；SKILL.md 依赖清单同步更新
- xlsx 技能：scripts/office/soffice.py 增加 Windows 安装路径自动探测（Program Files / (x86) / LOCALAPPDATA），
  recalc.py 不再因 soffice 不在 PATH 失败；SKILL.md 补批量合并/统一格式流程
- docx 技能 soffice.py 同步路径探测（docx→PDF 转换同样受益）；pdf 技能补 PDF→Word 与表格→xlsx 闭环流程
- 实测：Markdown→docx→format_convert 表格 round-trip 正确；openpyxl 写公式→LibreOffice 重算 4 公式 0 错误
  （SUM/差额/利润率读回 220/130/90/40.9%）；reportlab PDF→pdfplumber 表格提取→xlsx 结构完整
- 验证：go test ./... 全绿（docmd 新增 2 例）；技能修改已同步镜像 ~/.codex/skills

## v2.7.2「本地模型优势落地 · 扫描件本地 OCR」（2026-08-09）

> 善用 gaea 本地模型资产：扫描件 PDF 的 OCR 不再依赖未安装的 pytesseract，
> 改走本地双通道——Windows 原生 OCR（WinRT，离线零成本）+ 本地视觉模型（herdsman Qwen3.6）兜底。
- pdf 技能新增 scripts/ocr_local.py：PyMuPDF 渲染 PDF 页（300 DPI）→ windows-ocr.ps1（WinRT zh-Hans）
  → 文本过短/为空时降级本地视觉模型（GAEA_VISION_BASE_URL/GAEA_VISION_MODEL 可覆盖）；
  输出 UTF-8 文本或 JSON（每页 + 工具来源），支持 --mode auto|winrt|local
- pdf 技能自带 scripts/windows-ocr.ps1（WinRT OcrEngine 离线调用，与 ds-vision-skill 同源）
- SKILL.md：扫描件 OCR 节改为本地 OCR 主路径，pytesseract 降为可选外部方案
- 办公提示词新增「本地能力优先」纪律：扫描件/图片文字提取优先本地 OCR 或 vision，敏感文档本地处理不出机
- 实测：中文 docx→PDF→本地 OCR 识别出「项目周报 / 本周完成 8 项需求 / 营收 120 万元，同比增长 18%」；
  表格 PDF→OCR 数字与表头全部识别；英文/中文双链路均离线跑通
- 验证：技能修改已同步镜像 ~/.codex/skills

## v2.7.0「通用办公强化 · PPT 交付 + 技能沉淀 + 便捷入口 + 后台任务」（2026-08-09）

> 按《市场调研：通用办公 AI 智能体》P0 结论落地：新增 pptx 演示文稿技能（python-pptx 一键成稿）、
> 欢迎页新增「演示文稿」入口、一次成功对话一键沉淀为可复用技能、粘贴剪贴板图片即转图片附件上下文、
> 侧栏后台任务面板（运行中任务 + 完成 toast）、办公提示词增强（PPT 交付纪律 + 偏好主动记忆沉淀）。
- pptx 技能：新增 `.gaea/skills/pptx` 并镜像 `~/.codex/skills/pptx`（SKILL.md + scripts/create_pptx.py，
  16:9 封面/章节要点/演讲备注，缺 python-pptx 时自动 pip 安装，生成后回读校验页数）
- 前端入口：欢迎页核心能力新增「演示文稿」卡 + 内置技能新增 pptx chip；icons 兼容层新增 FilePpt
- 提示词：SingleModelPrompt 执行纪律加入 pptx 技能与演示大纲流程；新增「记忆沉淀」检查点
  （用户偏好/项目事实/踩坑经验主动 remember，避免重复交代）
- 便捷入口：Composer 粘贴剪贴板图片（截图/网页图）不再静默丢弃，转为图片附件上下文（复用 SavePastedImage）
- 后台任务：侧栏新增「后台作业」面板（运行中 bash/task 列表 + 状态点 + 相对时间，仅展开态显示）；
  JobDoneNotifier 在任务从运行列表消失时自动 toast「后台任务已完成」
- 技能沉淀：助手消息新增「沉淀为技能」操作，弹窗预填本次任务/回答，确认后写入 .gaea/skills/<name>/SKILL.md
  并镜像 ~/.codex/skills，热加载后立即以 /技能名 调用（GaeaCaptureSkill + skill.RenderSkillFile，含校验/覆盖/测试）
- 验证：go build/test 全绿（含 skill capture 新用例）、tsc + vite build 通过、vitest 47 例全过、
  pptx --demo 端到端生成 3 页演示文稿

## v2.6.9「搜索修复 · 人格重设计 · 移除 VoxCPM2」（2026-08-09）

> 通用办公搜索修复（新增 Bing/DDG 兜底 + 代理接入，不再报「所有搜索引擎失败」）；
> 聊天板块联网搜索接通（ChatSend searchEnabled + 开关生效）；gaea 人格与提示词整体重设计
> （统一 gaea 身份，清除 Ackem/Hermes/大地女神旧文案，清空 config.toml 土壤修复旧覆盖）；
> 首页语音固定 gaea；按用户要求移除实测不达标的 VoxCPM2，本地 TTS 保留 CosyVoice2。
- web_search：Bing/DuckDuckGo Lite 无 key 兜底、代理复用 web_fetch 链路、修复 cancel bug 与 429 重试、UA 换浏览器标识
- 聊天搜索：ChatSend(searchEnabled) 普通+角色对话自动联网注入结果；前端开关生效（普通模式也显示）；whisper WebSearch Bing 优先约 0.3s 带标题/链接
- 人格：personality/canon/product_identity/main_chat 提示词重写，gaea 预设去 goddess 标签；20+ 处 Hermes/Ackem → gaea；AboutPanel/首页语音标签同步
- 首页语音：固定 gaea，删除聊天人格自动同步语音的副作用
- 移除 VoxCPM2：引擎/前端 6 文件删除，本地 C:\AI\voxcpm、C:\AI\llama-omni 清理，释放约 14GB
- 验证：go build/test 全绿、tsc+vite build 通过、wails build 成功（releases/gaea-v2.6.9.exe + SHA256SUMS）

## v2.6.8「模型中心一键启动本地 TTS 服务」（2026-08-09）

> 本地 TTS 不再需要手动运行启动脚本：gaea 启动时自动保活 CosyVoice2 / VoxCPM2；
> 模型中心点击模型的「启动」按钮也会即时拉起对应服务；「测试连接」与语音合成前兜底同样自动拉起。
- 新增 internal/app/tts_service.go：core.ensureLocalTTSService（幂等）+ mediaState.StartLocalTTSService（Wails 绑定）
- 探测：CosyVoice2 http://127.0.0.1:8010/v1/models；VoxCPM2 http://127.0.0.1:8020/v1/status
- 拉起（隐藏窗口 CREATE_NO_WINDOW，不阻塞 UI）；异步轮询就绪（cosyvoice ≤10s / voxcpm ≤180s），就绪/失败通过 tts-service-status 事件通知前端
- 验证：go build/test 全绿、tsc + vite build 通过、wails build 成功（releases/gaea-v2.6.8.exe + SHA256SUMS）

## v2.6.7「VoxCPM2 Vulkan GGUF 加速 · 4 音色替换」（2026-08-09）

> 实测 VoxCPM2 三层架构落地：8030 llama-tts-server（llama.cpp-omni + Vulkan，Q8_0+F16 GGUF）为主后端；
> 8021 ROCm PyTorch 备胎；8020 adapter.py 统一入口（gaea 零改动）。
- 关键修复：SSLServer 空证书导致 bind 失败（改普通 httplib::Server）；AudioVAE 参考特征 frame-major 布局对齐 Python（克隆从近静音恢复）
- 性能：短句克隆 RTF 0.65–0.84（6 步/CFG 1.5），语音设计 0.57–0.60；对比 ROCm 5 步 RTF ≈0.06–0.12，整体快 1.5–8 倍
- 音色：CosyVoice / VoxCPM2 统一替换为火山引擎 4 音色（中文女/男、英文女/男）；参考音频 ≤3s/16kHz，适配器自动音量归一
- 验证：go build/test 全绿、tsc + vite build 通过、四音色端到端实测通过；wails build 成功（releases/gaea-v2.6.7.exe + SHA256SUMS）
## v2.6.6「VoxCPM2 本地语音引擎接入」（2026-08-09）

> 模型中心新增「VoxCPM2 (本地)」引擎：本地 OpenAI 兼容 TTS 服务（127.0.0.1:8020），
> 2B 扩散式多语种 TTS、48kHz 高保真输出，内置 7 音色 + 参考音频零样本克隆 + 声音设计；
> ROCm 驱动 Radeon 8060S 核显 + TunableOp 调优，实测 RTF ≈ 2.0。
- 引擎接入：modelengine 新增 `voxcpm` 类型与内置引擎，启动自动补齐 `VoxCPM2` 模型；
  「语音模型」页与「功能绑定 → 聊天语音」均可选择
- 服务端（`C:\AI\voxcpm\server.py`）：OpenAI 兼容 `/v1/audio/speech`、
  `/v1/audio/info`、`/v1/models`、`/v1/voices`（参考音频注册克隆音色，持久化）；
  Python 3.12 venv + torch 2.9.1+rocm7.2.1（AMD ROCm Windows）+ voxcpm 2.0.3
- 性能：ROCm 核显推理 + TunableOp GEMM 调优缓存（4~5 倍）；CFG 1.5 避免长文本跑飞重试；
  实测 7.7s 中文约 16s（RTF ≈ 2.0），短句 RTF ≈ 1.35；启动预热后首个请求即达稳态
- 音质：48kHz 输出；内置 7 个与 CosyVoice2 同源参考音色（中文女/男、英文女/男、日语男、粤语女、韩语女）；
  支持声音设计（自然语言描述音色）与参考音频克隆
- 验证：go build/test 全绿、tsc 通过；服务端直连合成 / 克隆 / 音色列表实测通过
  （详细记录：`docs/2026-08-09-voxcpm2-integration.md`）

## v2.6.5「CosyVoice2 LLM 核显加速」(2026-08-09)

> CosyVoice2 LLM 环节从 PyTorch CPU 切换到 llama.cpp GGUF + Vulkan（Radeon 8060S 核显），
> 整条合成管线提速约 8–10 倍：短句 6.5s→~1.5s，长句 24.5s→~2.8s。
> 默认使用 f16 GGUF（音质最接近原始权重），q8 为更快备选。
- 调研：Tinysoft/Cosyvoice2-0.5B-GGUF + llama.cpp PR #14711（Qwen2 bias 支持）
- 引擎：`cosyvoice/llm/gguf_engine.py`（token 直喂、KV cache、复刻 ras_sampling），
  `Qwen2LM.load_gguf()` 接入，实测词表布局 logits 相关性 0.9999
- 服务：server.py 默认加载 `gguf\cosyvoice_f16.gguf`（环境变量 `COSYVOICE_LLM_GGUF` 可切），
  启动预热 shader；失败自动回退 torch
- 修复：`mask_to_bias` fp16 下 -1e10 溢出 -inf 导致 ONNX Softmax NaN（fp16 改用 -3e4）
- 否决：fp16 flow estimator（误差累积，波形相关性 0.016）、LLM 单步 ONNX（链式发散）、
  bf16（EOS 失效）、flow 4 步（无稳定收益）
- 验证：直连 `/v1/audio/speech` 短句 ~1.4–1.8s、长句 ~2.8s；7 音色/中英日正常
  （详细记录：`docs/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`）
- 发布：`go build/vet/test` + `tsc/vite build` + E 系列回归全绿；`wails build` 成功，
  `releases/gaea-v2.6.5.exe` + `SHA256SUMS-v2.6.5.txt` + `v2.6.5.md`，桌面 `gaea.exe` 已同步

# gaea · 多功能 AI 助手

## v2.6.4「CosyVoice2 提速 + 模型中心音色选择」(2026-08-09)

> CosyVoice2 合成提速约 35%（短句 6.5s → 4.1s）：flow 解码器切 ONNX + DirectML（AMD 核显），
> 采样步数 10→5，音质与 torch 路径相关性 1.0000；
> 「功能绑定 → 聊天语音」卡片新增音色选择（xAI / CosyVoice2 / Herdsman）。
- CosyVoice2 提速：`flow.decoder.estimator.fp32.onnx` + `DmlExecutionProvider` 替换 torch estimator，
  flow 步数 5；服务端 server.py 内置，重启服务即生效
- LLM 优化探索：Qwen2 单步 ONNX 导出成功（fp32 CPU 28.8ms/步、DML 10ms/步），
  但链式推理数值发散导致音频失真，暂不启用；bf16 破坏 EOS 采样，弃用
- 模型中心：聊天语音绑定卡新增「音色」下拉（xAI 26 音色 / CosyVoice2 7 音色 / Herdsman 服务端列表），
  选择即写 `tts_voice` 持久化
- 验证：直连 4.1s/0.92s wav；go test 全绿；tsc + vite build 通过
  （releases/gaea-v2.6.4.exe，SHA256SUMS-v2.6.4.txt）

## v2.6.3「CosyVoice2-0.5B 本地音色克隆接入」(2026-08-08)

> 模型中心新增「CosyVoice2 (本地)」引擎：本地 OpenAI 兼容 TTS 服务（127.0.0.1:8010），
> 聊天语音可绑定 CosyVoice2-0.5B，内置 7 个音色并支持参考音频零样本克隆。
- 引擎接入：modelengine 新增 `cosyvoice` 类型与内置引擎，启动/刷新自动补齐 `CosyVoice2-0.5B` 模型；
  「语音模型」页与「功能绑定 → 聊天语音」均可选择
- 音色支持：`/v1/audio/info` 拉取音色列表（中文女/男、英文女/男、日语男、粤语女、韩语女），
  设置面板/聊天设置新增 CosyVoice 选择器；无效音色回退「中文女」，
  `defaultVoiceForModel`/`ttsVoiceForModel` 补齐 cosyvoice 默认音色
- 服务端：`POST /v1/audio/speech`（OpenAI 兼容）+ `POST /v1/voices`（参考音频注册新音色，持久化到 spk2info.pt）
- 实测：直连合成 6.5s/0.92s 音频（RTF 6.8x，PyTorch CPU）；gaea 客户端端到端 38KB wav；
  go test 全绿、tsc + vite build 通过
  （releases/gaea-v2.6.3.exe，SHA256SUMS-v2.6.3.txt）

## v2.6.2「xAI Grok TTS 接入」(2026-08-08)

> xAI 引擎内置 `grok-tts` 云端语音模型：模型中心「功能绑定 → 聊天语音」可绑定 xAI，
> 语音对话/朗读优先走 Grok 音色（Eve/Ara/Rex/Sal/Leo + 旗舰 21 个），
> 实测合成约 1.4~2.4 秒，快于本地 qwen3-tts（声码器 CPU）的 2.7~3.3 秒。
- xAI TTS 路由：语音管道新增 `tryEngineTTS`，xAI 走 `POST /v1/tts`（复用 OAuth token），
  聊天语音绑定 → 全局 TTS → 自动路由均生效；流式朗读选中 xAI 时同样优先云端
- 模型中心：xAI 引擎保活内置 `grok-tts`（启动/刷新均自动补齐），
  功能绑定与「语音模型」页可直接选择；设置面板/聊天设置新增 xAI 音色选择器，
  无效音色自动回退 `eve`，`tts_voice` 持久化
- xAI TTS 客户端完善：`SynthesizeWithMime` 返回真实 MIME，音色列表静态校验 + 单测
- 验证：`go build/test ./internal/{tts,modelengine,app}/...` 全绿；`tsc` + `vite build` 通过；
  直连 `api.x.ai/v1/tts` 实测 HTTP 200 / audio/mpeg；`wails build` 成功
  （releases/gaea-v2.6.2.exe，SHA256SUMS-v2.6.2.txt）

## v2.6.1「语音交互打通 · 音色可配置 · 首页语音角色与聊天一致」(2026-08-08)

> 按 Herdsman 新版接口重新打通语音对话：AI 回复后 TTS 朗读恢复正常，不再卡在“正在聆听”；
> 语音音色可在设置面板直接选择并持久化；首页语音角色与聊天板块保持一致。
- 语音交互打通：TTS `audio_url` 支持 data URI / 相对路径 / 绝对 URL 三种形态并返回真实 MIME；
  合成前动态查询 `/v1/audio/info` 音色列表，`Cherry` 等已移除音色自动回退可用音色；
  修复 ASR `/v1/v1/audio/transcriptions` 双重前缀 404；TTS 播放期间保持“AI 回复中”状态
- 音色设置：聊天面板「语音设置」新增 Herdsman 音色选择器（实时拉取服务端支持列表），
  写入 `~/.gaea_config.json` 的 `tts_voice` 持久化；设置页聊天设置同步
- 首页语音角色：不再写死 gaea，读取聊天板块保存的同一角色键并动态显示标签；
  后端新增 `voice_personality` 持久化
- 普通对话联动：聊天板块切换「普通对话」时语音回复使用中性助手口吻，
  首页语音同步为「普通对话」；切换人格时首页语音跟随该人格
- 模型中心「功能绑定」新增聊天语音模型：单独绑定聊天语音的引擎+模型（持久化），
  未绑定回退全局 TTS；语音模型列表随引擎自动刷新，便于后续扩展
- 验证：go build/vet 全绿；tsc + vite build OK；前端 E 系列回归守卫 OK；
  直连 Herdsman 端到端验证合成 122KB 真实语音；wails build 成功
  （releases/gaea-v2.6.1.exe，SHA256SUMS-v2.6.1.txt）

## v2.6.0「小说创作工作台重设计 · 角色卡补齐/剧照/合并」(2026-08-08)

> 小说创作页改为三栏写作工作台（章节树 / 编辑器 / 创作设置），写作技能与生成参数
> 常驻可调；章节捕获的角色卡补齐 AI 补档、剧照生成与同名合并；修复设定未注入、
> 办公目录被误判为代码工程等问题。
- 小说创作界面：三栏写作工作台——左章节树（未写/生成中/已写入状态点）、中纸面化
  编辑器（标题/状态/字数/重写/保存/空状态引导）、右创作设置面板（设定摘要/技能/
  生成参数/剧情方向/统计）；技能选择常驻并显示说明与适用场景
- 生成设置真实生效：目标字数（不足自动续写）与生成温度由界面传入后端
  （CreateChapter 新增参数，默认 5000 字/服务端默认温度）
- 正文标题约束：提示词禁止 AI 自拟章节标题或编号行，章节编号由系统管理，
  杜绝「第七章…这是第一章」错乱
- 小说设定注入修复：创作页在挂载/切书/开向导/生成前均重读最新设定，提示词必注入
- 章节角色捕获：新角色只写本书 characters.json，需手动「一次性迁移」进全局库；
  角色页未入库标记 + 刷新联动
- 角色卡能力：未入库角色支持 AI 补齐空缺字段、生成剧照（自动补全关键描述）、
  合并同一人的不同称呼（引用与关系重定向去重）
- 通用办公：Profile.Scan 语言改为按工程清单真实检测，办公目录不再被标成 Go 工程；
  项目画像/技能/记忆基于工作区扫描；开发 mock 的 coding agent 提示词改为办公助手
- 技能加载：修复 SKILL.md 多行 applies_to 解析为空的问题
- 验证：go build/vet/test 全绿、tsc + vite build OK、vitest 47 例全过、
  wails build 成功（releases/gaea-v2.6.0.exe，SHA256SUMS-v2.6.0.txt）

## v2.5.0「全界面科幻视觉重设计 · 绘梦图生图/文生视频 · 角色库模型绑定」(2026-08-08)

> 设置、首页、聊天欢迎屏、小说、绘梦五大板块统一为「玻璃 HUD + 单一强调色」设计语言；
> 绘梦新增图生图与文生视频（ComfyUI 后端）；模型中心功能绑定实时同步并新增角色库绑定；
> 角色库剧照支持独立后端/模型；本地视觉识别管线修复 UTF-8 编码并切换到 Qwen 视觉模型。
- 设置面板：左侧导航改为顶部平铺分类磁贴（吸顶可切换），统一 HUD 角标/聚焦/按压反馈；
  修复 flex + margin auto 导致容器收缩为内容宽度的问题（宽度 641px → 1080px）
- 首页：语音中枢升级为视觉主角——HUD 角标框、雷达脉冲环、声谱均衡条、等宽遥测读数，
  聆听/回复时扫描光带掠过面板；新增 380px 语音球与实时音量展示
- 聊天欢迎屏：普通模式用 VoiceChatOrb、角色模式用 CompanionAvatar（随语音状态变色），
  配 HUD 角标、脉冲环、遥测行；建议卡升级为键盘可达（role/tabIndex/focus-visible）；
  角色模式空状态隐藏重复人格条，修复 orb 被 flex 压缩的问题
- 小说板块：默认 Tabs 改为顶部二级导航平铺（书架/设定/角色/创作/阅读/导出），子页保持挂载；
  新增统一设计层 novel-workspace.css——玻璃面板、HUD 面板头、章节节点树、纸面化衬线编辑面、
  antd Tabs 玻璃化；大纲面板硬编码紫色改为 var(--gaea-glow)；设定页改为左右双栏
  （左设定编辑器 + 右设定 Agent 常驻，修复 flex 宽度收缩）
- 绘梦：顶部三模式导航（文生图 / 图生图 / 文生视频）；图生图支持上传/拖拽参考图 + 重绘幅度；
  文生视频支持分辨率/时长帧率预设；后端新增 ComfyUI 图生图工作流（LoadImage+VAEEncode+低 denoise）
  与 LTX-Video 文生视频工作流，输出解析扩展到 images/gifs/videos，结果舞台/历史胶片/灯箱支持视频；
  新增 GenerateMedia 绑定；修复引擎列表为空时页面崩溃
- 模型中心：功能绑定面板监听 feature-model-changed 事件实时同步（其他页面改绑定即时刷新）；
  聊天合并轻语、办公合并方案（office+gaea 双写），删除重复行；新增角色库 LLM 绑定
  （func_characterlib_*，角色生成/补全走独立绑定）
- 角色库剧照：新增独立图片后端/模型绑定（portrait_backend/portrait_model，空=跟随绘梦），
  模型中心新增「角色库剧照」卡；删除小说卡上的剧照模型标签
- 视觉识别管线：vision 技能默认模型切换为 Qwen3.6-35B-A3B-Uncensored（布局识别准确率明显提升）；
  修复 PowerShell 5.1 下 UTF-8 编码问题（脚本转 BOM + 改用 .NET HttpClient），中文输出不再乱码
- 验证：go build/vet/test 全绿、tsc + vite build OK、vitest 37 例全过、
  Edge headless 各界面实测 + 视觉模型复核

## v2.4.5「通用办公欢迎页重设计」(2026-08-08)

> 通用办公对话窗口的欢迎界面从土壤修复时代的旧版，重设计为贴合当前定位的通用办公入口。

- 内容更新：移除场地调查/投标标书/修复方案/污染风险/成本测算等土壤修复专项卡片，
  改为 6 个通用办公核心能力（文档撰写/表格处理/格式转换/图表生成/报告拼装/知识沉淀）
- 新增内置技能区：format-convert / chart-builder / doc-assemble / docx / xlsx / pdf 六个技能 chips，
  点击即填入对应任务提示词
- 视觉升级：主视觉改为「GAEA OFFICE + 今天想做什么？」大标题层级，logo 带光晕角标，
  能力卡 hover 顶部高光线 + 箭头提示，新增渐次入场动画（尊重 prefers-reduced-motion）
- 保留工作区/模型 pill、最近会话区并同步优化样式
- 文案清理：welcome.tagline 与加载页 skeleton 描述从「土壤修复引擎」改为通用办公，
  删除 locale 中无人引用的土壤修复快捷卡片条目
- 验证：tsc + vite build OK；vitest 37 例全过；Edge headless 实拍确认布局正常

## v2.4.4「文件预览与右侧面板精简」(2026-08-07)

> 右侧面板去掉「消息/报告」标签并默认折叠；对话内可直接点击文件路径打开全新的预览阅览面板。

- 右侧面板：删除「消息」与「报告」标签页（MessageNavigator / ReportPreviewPanel），保留「文件」「统计」；
  面板改为默认折叠，点工具栏按钮展开
- 对话内文件预览：Markdown 渲染中的本地文件路径（.md/.docx/.xlsx/.pdf/.png 等）渲染为可点击的文件 chip，
  用户消息里的 @ 附件同样可点击；点击打开居中大尺寸预览弹层（Esc/遮罩关闭，支持定位与外部打开）
- 预览面板重设计：后端新增 GaeaPreview 统一预览接口——图片返回 dataURL、Markdown/文本原文渲染、
  docx/xlsx/pdf 经 docmd 转 Markdown 内联预览（含 OCR 回退），不支持格式给出外部打开入口；
  工作区「文件」面板的预览同步升级为 Markdown 渲染
- 转换引擎抽取：format_convert 的 docx/xlsx/pdf→Markdown 逻辑迁至 internal/office/docmd 包，
  工具与预览面板共用一份实现；修复 openpyxl 内联字符串单元格（inlineStr）丢表头的问题
- 测试：新增 FilePreviewModal 3 例 + Markdown 本地文件链接 2 例，前端 37 例全过；go test 全绿

## v2.4.3「精简内置工具集」(2026-08-07)

> 按市场调研结论（同类产品 10~20 个核心工具、文档/表格用技能扩展）把内置工具从 38 个精简到 17 个核心工具。

- 删除 21 个冗余内置工具：计算器（calc_math/calc_stats/calc_unit）、压缩（archive）、
  电脑操作（computer_use）、甘特图（gantt_gen）、项目初始化（project_init）、工具链模板（run/save_template）、
  方案 agent 工具（proposal_list/write/export），以及被 ModelScope docx/pdf/xlsx 技能覆盖的文档专项工具
  （docx_read/docx_write/pdf_create/pdf_extract/pptx_create/xlsx_read/xlsx_write/doc_merge/csv_parse）
- 删除与 run_skill 重叠的 parallel_skills（保留 RunDAG 管道基础设施）
- 保留 17 个核心工具：文件与命令（read_file/write_file/ls/bash/bash_output/kill_shell/wait）、
  网络（web_search/web_fetch）、任务（todo_write/complete_step）、记忆与知识（memory_search/knowledge_add/knowledge_search）、
  技能（read_skill）、办公引擎（format_convert/chart_gen）
- 内置技能去工具依赖：chart-builder 改为 bash + python 读表、doc-assemble 改为 format_convert + bash 拼装，
  不再依赖已删除的 xlsx_read/csv_parse/doc_merge/docx_write
- 系统提示词与单模型执行纪律改写为「文档创建/编辑交给 docx/xlsx/pdf 技能，格式转换用 format_convert」
- 修复 chart_gen 在 Windows 被 python3 商店别名劫持的问题，并抑制 matplotlib 字体告警污染 JSON 输出
- 前端能力抽屉工具列表重建为实际 7 组 29 个工具（compact 模式对模型隐藏 kill_shell/wait），
  清理 git/notebook/edit 等死条目与报告来源旧映射
- 验证：go build + go test ./... 全绿；tsc + vite build OK；vitest 32 例全过；format_convert 与 chart_gen 端到端验证通过

## v2.4.2「通用办公改造 · ModelScope 文档技能」(2026-08-07)

> 「智能办公」精简为「通用办公」：删除土壤修复专项工具与技能，安装 ModelScope 的 docx/pdf/xlsx 文档技能。

- 办公定位：DefaultSystemPrompt 与单模型执行纪律改写为通用办公助手（文档/表格/图表/演示/检索/方案报告/任务跟踪）
- 删除 6 个土壤修复专项工具：survey_report/bid_proposal/imple_plan/spec_query/spec_judge/material_query，
  并从 compact 描述/模式表中清理
- 内置子代理技能收敛为 3 个通用技能：format-convert / chart-builder / doc-assemble（含 wrapper 工具与测试）
- 删除项目内 5 个土壤办公技能（site-survey/risk-assessment/remed-design/bid-package/data-report），保留 skill-creator
- 安装 ModelScope 技能：docx / pdf / xlsx（SKILL.md + 完整 scripts/schemas/templates），
  同时写入 ~/.codex/skills 与 .gaea/skills，供 Codex 与 gaea 通用办公使用
- 前端文案统一：智能办公→通用办公，报告类型映射改为通用办公类型，移除 Hephaestus 残留
- 验证：go vet + go test ./... 全绿；tsc + vite build OK；vitest 32 例全过

## v2.4.1「设置面板重设计」(2026-08-07)

> 按当前功能板块重构设置中心：左侧模块导航 + 全局搜索，清除死代码与重复面板。

- 布局重设计：顶部 Tab 改为左侧功能分组导航（通用 / 聊天 / 小说 / 绘梦 / 办公 / 模型 / 关于），
  窄屏自动转横向滚动 chips；保留全局搜索并可过滤分组
- 聊天：合并「AI 伴侣 + 默认人格 + 语音核心项」（语音对话/朗读回复/合成音色），
  移除无人读取的「主动搭话」开关与重复的「清除全部会话」；完整语音面板仍在聊天板块
- 办公：并入「方案编写模型」绑定（原方案 Tab 唯一有效内容），删除纯文档的「方案生成」说明
- 模型（新增）：找回此前丢失入口的全局「推理强度」，展示当前模型并提供「前往模型中心」跳转
- 关于：版本卡片 + 系统信息（配置路径）+ 可折叠更新日志，替换原系统 Tab
- 清理：删除死代码 EnginePanel、重复面板 VoicePanel/ProposalPanel、遗留 WhisperPanel/SystemPanel
- 验证：新增 SettingsPage 4 例回归测试（分组渲染/默认分组/搜索过滤/切换），
  `npm run test` 32 例全过，`tsc` + `vite build` OK，`scripts/ci.ps1` CI OK

## v2.4.0「网页调试桥接 · 办公引擎热加载」(2026-08-07)

> 新增 `GAEA_HTTP_PORT` 网页调试桥接：浏览器/手机直接驱动同一个 Go 内核（RPC + SSE），
> 办公引擎支持从磁盘热加载，技能/工具/插件变更无需重启桌面端即可生效。

- HTTP 调试桥接（`internal/httpbridge`）：`POST /api/rpc` 反射分发全部 Wails 绑定方法、
  `GET /api/stream?id=` SSE 事件推送（15s keep-alive）、`/api/health` 存活探针；
  `core.emit` 统一发布到桥接订阅者（无 Wails 上下文也推送）
- 前端 HTTP 模式：`runtimePolyfill` 补齐 EventsOn/EventsOff/EventsOnMultiple/EventsEmit，
  所有事件经 SSE 对齐桌面端——修复并发订阅只连首个事件的竞态，探测失败后桥接就绪可自动重连；
  `bridge.ts` 将 `window.go.app.App.*` 代理到 `/api/rpc`；Vite 将 `/api` 代理到桥接
- 办公引擎热加载：`GaeaReload` 重新读取磁盘持久化配置并重建 controller
  （Agent 参数/权限/沙箱/技能路径/插件），成功后广播 gaea-ready 令前端 store 重新拉取；
  失败时保持旧引擎继续运行，不替换任何状态
- 前端入口：能力抽屉（MCP/工具/技能）新增「热加载」按钮并展示重建后的工具/技能数量；
  设置→办公新增「从磁盘热加载」；三语 i18n 同步
- 验证：新增 httpbridge RPC/SSE 2 例 + GaeaReload 热加载 1 例；`go vet` + `go test ./...` 全绿、
  `tsc` + `vite build` OK、`scripts/ci.ps1` CI OK

## v2.3.0「界面焕新 · 办公整合」(2026-08-07)

> 在 v2.2.0 基础上迭代：角色库/首页/办公板块全面重设计，办公双模块合并为独立二级导航，随机生成能力补全（含人格）。
- 角色库界面重设计：档案墙（立绘主导 + 身份覆盖 + 档案眉），加载骨架/空状态/深浅色适配，去掉多色硬编码统一主题强调色
- 角色命名：28 个内置角色配中文人名（白霜/温言/苏晚晴/林小满/顾清霜/陆寒川/姜棠/谢临川/叶绵绵/许一诺/秦挽月/阮慈/周栗/季如烟/陈恪/沈墨白/程暖阳/席晚棠/霍承渊/虞栀/江野/晏观澜/裴聿修/夏知微/乐桃/顾衍/卫昭），gaea 保留原名
- 角色相卡面板重设计 + 随机生成完善：新增 `CharacterGenerateRandom`（all 全量随机/按字段随机），字段旁 ↻ 骰子单独随机、五维本地随机、性别/定位/状态/年龄即时随机
- 首页 AI 中枢启动器重设计：单一主题强调色、玻璃档案化卡片、加载/焦点/降级动效处理
- 移除窗口顶部流光扫描条（scanline-top / scanSweep）
- 办公板块整合：合并「办公 + 方案编写」为独立二级导航（智能办公/方案编写），`OFFICE_MODULES` 注册表可扩展，移除顶部重复「方案编写」入口
- 智能办公整层重设计：对话区铺满面板（修复全屏缩成一团）、正文行宽约束、清爽玻璃面层（侧栏/顶栏/消息/输入框/滚动条）
- 方案编写整层重设计：胶囊式标签导航、左侧项目导航唯一入口（移除重复「方案列表」标签页）、隐藏重复步骤条，消除层层标题
- 验证：28 项回归测试通过；`go build` + `tsc` + `vite build` OK；`wails build` 成功（build/bin/gaea.exe ~38MB）

## v2.2.0「统一角色库」(2026-08-07)

> 在 v2.1.0 基础上迭代六轮：全局统一角色库、小说单向引用+抽卡、聊天内选角色、角色记忆隔离、状态/记忆/追踪归集角色库、取消「轻语」称谓。

- 角色库架构重构：新增 `characterlib.db` 全局统一角色存储（小说字段 + 聊天字段同一行），内置人格种子化进库；项目只引用角色并携带项目内弧线状态；导入只增不改，杜绝小说反向污染角色
- 小说角色面板改为「抽卡」：从角色库随机抽取（性别/标签/可聊天过滤），不再自行生成角色；面板只编辑项目内定位/弧线/状态，全局设定去角色库
- 聊天只选角色：聊天内 PersonaPicker 选择器（搜索/头像/类型标签），角色管理/编辑/状态/记忆/追踪全部归集角色库
- 角色记忆隔离：事实/情节/图谱按会话恢复与合并写回，A 角色记忆不串 B 角色；追踪新增会话归属列并按角色持久化
- 取消「轻语」称谓，统称聊天；首页启动器、设置、模型中心、记忆中枢文案全面统一
- 验证：新增 30+ 回归测试；`scripts/ci.ps1` CI OK；`wails build` 成功（build/bin/gaea.exe 38MB）

## v2.1.0「二代完善」(2026-08-07)

> 在 v2.0.1 基础上迭代六轮：模型中心完善、功能级模型启停、Cmd+K 引擎路由、OAuth 回归验证、前端 E 系列核对、小说链路审计。

- 模型中心：引擎状态持久化（`whisper_data/engines.json`）+ 稳定顺序 + 连接状态缓存；修复 `active_engine_id` 只存不读；DeepSeek 脱敏与真实活跃模型展示
- 模型路由：FeatureModelBar 启停改为功能级开关（`func_*_enabled`），不再误关全局引擎；Cmd+K 编辑走 novel 绑定；5 个 agent 未绑定回退按引擎解析默认模型
- OAuth：discovery 配置化（`OIDCDiscoveryURL` 生效）+ 换 token 超时客户端；E04/E13 回归
- 前端：E16/E22/E23/E24 核对 + 静态回归守卫 `scripts/frontend-e-check.mjs` 接入 CI
- 小说：角色注入剥离剧照 base64（`types.PromptView`）；E02/E03 回归
- 验证：go build/vet/test + tsc + vite build + 前端守卫全绿（`scripts/ci.ps1` CI OK）

## v2.0.1「三脑底座」(2026-08-07)

> 在 v1.21.0 基础上迭代：模型路由、三脑记忆、主脑可选编排、基线加固。

- 基线：frontend/package.json 纳入版本控制；scripts/ci.ps1 闸门；E01-E24 评估集 + 首批回归
- 模型：routeModel 降级链（功能绑定→全局→首个可用）；novel/whisper 调用收敛；模型中心"当前生效"
- 记忆：BrainStore 三脑统一访问 + brain_links 跨脑关联；记忆中枢三脑检索
- 编排：ModuleRegistry + RunModule + MainBrainChat（可选入口，不经由模块直达路径）
- 互联：方案生成自动注入跨脑记忆；3.0 模块协议文档预留
- 验证：go build/vet/test 全绿 + tsc + vite build + wails build（见 releases/v2.0.1.md）

## v1.21.0「UI 工作台」(2026-08-05)

> 方案编写板块重构为三栏工作台：左侧项目/方案树（检查分数徽章）＋ 中间文档工作区 ＋ 右侧 AI 上下文面板（招标要点/检查摘要/单人复核清单）；去除 Office 页 emoji 标签与 whisper-theme 依赖。

- 后端：Proposal 增加 CheckSummary（failed/warn/total）与 ReviewChecklist（复核项 Done），SchemaV7 落库；CheckAll 自动写入检查摘要并补默认复核清单（废标逐条/工期一致/评分覆盖/暗标合规/规范引用/签字盖章）；toMap/fromMap 透传
- 前端三栏：左栏项目与方案树（点击过滤/选中，方案卡显示模板、检查分数红/橙徽章）；中栏保留流程步骤条 + 6 个 Tab；右栏 AI 上下文 Drawer（招标要点、检查摘要、复核清单勾选保存、来源定位提示）
- 视觉：移除 Office 页全部 emoji 标签与 `whisper-theme.css` 引用，统一走主题令牌
- 验证：新增 1 测试（摘要与清单持久化）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.20.0「工作流编排」(2026-08-05)

> 方案编写板块引入四阶段流水线（招标解读→投标生成→投标检查→排版导出）与阶段闸门；一键流水线；办公 agent（Hephaestus）新增方案读写/导出工具。

- 阶段模型：Proposal.Stage（parse/generate/check/format，SchemaV6 proposals.stage 列）；解析成功→parse、大纲→generate、全面检查→check、导出→format 自动推进
- 阶段闸门：有招标文件但未解析时，生成大纲/章节/批量生成返回明确提示（空白方案不拦截）
- 一键流水线：RunPipeline 按序执行 解析→大纲→批量生成→全面检查，进度事件 proposal-pipeline-progress
- 办公 agent 打通：proposal 全局服务单例（测试可注入）+ 内置工具 proposal_list / proposal_write / proposal_export，Hephaestus 可直接读写方案需求/章节并导出
- 规范索引抽离共享包 specdata（builtin 工具与 proposal 模块共用，解除导入环）
- 前端：顶部 5 步步骤条（解析/大纲/编制/检查/导出）+ 一键流水线按钮 + 下一步引导（按阶段跳 Tab）
- 验证：新增 6 测试（闸门/阶段推进/流水线/办公工具）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.19.0「排版导出」(2026-08-05)

> 方案编写板块 docx 导出升级为可配置排版引擎：封面/目录/标题层级/页眉页脚页码/Markdown 表格与代码块；按章节导出；暗标规则库（内置土壤修复通用规则，导出自动清理）。

- docx 排版引擎：A4 页边距、页眉（方案标题）、页脚页码（PAGE 字段）、封面页、静态目录、章节三级标题（22/16/14 磅）、正文 12 磅宋体；Markdown 块解析（段落/标题/列表/表格→docx 表格/代码块等宽字体）
- 按章节导出：ExportSectionDocx 单章（含子章节）渲染为独立 docx；按章节 MD 已有
- 暗标规则库：office.db SchemaV5 dark_rules 表；内置“土壤修复通用暗标规则”（无加粗/斜体/下划线/彩色/emoji/特殊符号/压缩空行）；规则可增删改；导出选项选择规则后自动清理内容且标题不加粗
- 绑定：ProposalExportDocxWithOptions（封面/目录/暗标规则）、ProposalExportSectionDocx、ProposalDarkRulesList/Save/Delete
- 前端：导出设置 Modal（封面/目录开关 + 暗标规则选择）、单章导出（章节下拉）、暗标规则库管理 Modal（列表/编辑/选项 Checkbox/删除）
- 验证：新增 8 测试（渲染封面目录表格、单章导出隔离、暗标 seed/清理/CRUD、绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.18.0「校验引擎」(2026-08-05)

> 方案编写板块检查能力升级：可插拔 CheckRule 引擎 + 结构化规则（废标响应/数据一致性/跨章重复/暗标格式/规范引用）+ AI 语义覆盖规则，统一检查报告前端逐项处理。

- 校验引擎框架：CheckRule 接口 + ruleFunc 适配器 + RunChecks 运行器（规则错误转 error item）
- 结构化规则（确定性，无需 LLM）：废标条款响应（未明确回应 warn）、项目事实一致性（未体现 warn/工期冲突 fail/暗标单位名 fail）、跨章重复（20 字 n-gram 交并比，>50% fail / >30% warn）、暗标格式（加粗/删除线/emoji）、规范引用（无引用 warn/编号不在知识库 warn）
- AI 语义覆盖规则：对照招标评分标准 full/partial/none 检查（LLM）
- CheckAll 聚合全部规则输出统一检查报告；既有 CheckCoverage 内部复用并保持绑定兼容
- 绑定 ProposalCheckAll；前端导出 Tab「全面检查」报告面板（规则/状态着色/证据/章节定位跳转）
- 验证：新增 15+ 测试（框架/规则/聚合/绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

### v1.18.0 补充「转换诊断与扫描件 OCR」(2026-08-05)

> 修复桌面实测问题：招标解析失败根因多为第一步转换未成功且不可见。新增扫描件 OCR、转换结果阅览、AI 工作台。

- 根因修复：convertPdfToMD 无页面文字时不再返回“仅页眉”的伪成功（此前扫描件被误判已转换）→ 返回明确错误并触发 OCR；ParseBidFile 全文件转换失败时错误信息包含每个文件名与原因
- 扫描件 OCR：OCRProvider 可插拔接口 + Python 管线（PyMuPDF 渲染 300dpi 页面 → rapidocr_onnxruntime 识别，中文支持）；自动检测，无 OCR 引擎时给出安装指引；FileDoc 增加 error/ocrStatus 字段，OCR 结果标记“OCR 转换”
- 转换结果阅览：文件列表状态细化（已转换/OCR 转换/转换失败+原因 tooltip/待转换）+「查看」抽屉预览转换后 Markdown 或失败原因
- AI 工作台：转换与 AI 分析全过程事件（proposal-ai-progress：start/parse-file/parse-request/parse-reply/done/error），前端阶段动画（Spin+阶段文案）与右侧日志面板（自动滚动）；AI 分析按钮在“任一文件转换成功”时可用（不再被失败文件卡死）
- 验证：新增 6 测试（OCR 调用/OCR 不可用报错/全文件失败明确错误/进度事件）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.17.0「记忆中枢知识资产」(2026-08-05)

> 方案编写板块知识资产统一集成 gaea 记忆中枢：规范条文/素材/历史方案全部入库 Hephaestus.db knowledge 表，spec_query 改读知识库，方案可归档回写并提供同类型参考；模板库落库。

- 规范知识入库：内置土壤修复规范索引（GB 36600/15618、HJ 25.x、HJ 682、HJ 1185 等 15+ 条文）与「土壤修复常用技术」幂等写入记忆中枢（Category=规范标准/经验总结），SearchSpecs 按查询令牌重排（标题 3 分/正文 1 分）
- 素材库：业绩/人员/设备/常用段落以 Category=素材库 入库（AddAsset/ListAssets/SearchAssets/RemoveAsset），名称纳秒级唯一
- 历史方案：ArchiveProposal 将装配全文归档（Category=设计方案 + tag legacy-proposal），SearchLegacyProposals 检索；SectionContext 自动注入同类型历史方案参考摘要（≤600 字，注明不得抄袭）
- spec_query 改读记忆中枢：优先检索知识库规范条文（Top 5），知识库不可用时回退内置索引
- 模板库落库：DefaultTemplates 幂等 seed 到 office.db templates 表，ListTemplates 优先读库（失败回退默认）
- 前端：导出 Tab「归档到记忆中枢」按钮；右上角「素材库」弹窗（搜索/新增/删除）
- 验证：新增 15+ 测试（规范入库/检索重排、素材 CRUD、归档与参考注入、模板 seed、spec_query 知识库优先、绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.16.0「大纲与撰写引擎」(2026-08-05)

> 方案编写板块长篇生成落地：目录策略 + 字数预算分解（以招标文件要求为准）+ 大纲重排/目录导入 + 项目事实基线 + 统一章节上下文 + 批量生成队列（断点续写/合并装配）+ 工艺流程图/组织架构图预设。

- 大纲策略与字数预算：三种目录策略（严格按评标办法/严格按格式要求/参考两者）；解析层提取招标 totalWords，预算按招标要求分配（未要求兜底 10 万、用户可改），递归分配到章→节→小节（叶子合计严格等于总数）
- 大纲重排与目录导入：同级上移/下移自动重编号；Markdown 标题（#/##/###）解析导入替换大纲
- 项目事实基线：office.db SchemaV3 project_facts（工期/业主/修复目标/人员等跨方案共享），SchemaV4 sections 增加 word_target/words 列
- 章节上下文 v2：统一 SectionContext（大纲/评分/废标/格式/暗标/项目事实/字数目标/前章摘要/后节锚点），单章生成与流式生成共用
- 批量生成与合并装配：RunBatch 按叶子单元顺序生成、跳过已完成（断点续写）、失败继续、进度回调（proposal-batch-progress）；Assemble 自动编号（第N章/N.M/N.M.K）并用于导出
- 前端：大纲 Tab 策略/总字数/导入目录/上移下移/字数目标；文本编制 Tab 批量生成/停止/进度条；方案列表右上角项目事实编辑；图表 Tab 工艺流程图/组织架构图/横道图
- 验证：新增 20+ 测试（预算分配/策略提示词/重排/导入/事实往返/上下文注入/批量/装配）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.15.0「招标解析管线」(2026-08-05)

> 方案编写板块招标解析升级：结构化字段提取（概况/工期/资质/评分/废标/格式/暗标）+ 原文来源定位 + parse_results 落库 + 前端结构化卡片与原文抽屉。OCR 仅定义可插拔接口，文字型 PDF 优先。

- 文档分页文本提取（doctext.go）：PDF 按页返回真实页码，DOCX/TXT 单页（Page=0），供来源定位使用
- office.db SchemaV2：新增 parse_results 表（字段/页码/Markdown 偏移/摘录/置信度），Store 提供 SaveParseResults/ListParseResults；AddFile 返回文件 ID
- 来源定位器（locate.go）：AI 摘录 quote 先精确匹配，失败后做忽略空白匹配，映射回原文偏移与页码
- 招标解析管线 v2（parse.go）：逐文件/分块 LLM 提取字段+原文摘录 → 后端确定性定位 → 结果写入 parse_results 与 BidSummary（qualification/format/darkRules/redLineItems/parseStatus，旧字段全兼容）
- 绑定序列化：btm/bsf 改为 JSON 往返，新字段自动透传前端
- 前端：招标解析 Tab 由 JSON 文本框改为结构化字段卡（可编辑保存）+ 来源 chip + 原文预览抽屉（滚动高亮摘录）
- 验证：新增 20+ 测试（分页提取/定位器/解析结果 CRUD/解析管线/序列化往返）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.14.0「方案数据底座」(2026-08-05)

> 方案编写板块数据底座重建：JSON 文件存储迁移 SQLite（office.db），引入项目（标段）层级、版本快照与旧数据无损迁移；方案列表按项目分组。

- office.db 网关（internal/office/db）：SchemaV1 六表（projects/proposals/sections/files/versions/templates）+ schema_meta 迁移链，纯 Go SQLite 驱动，与主脑库同模式
- proposal.Store 迁移 SQLite：项目 CRUD、方案/章节树持久化（含子章节递归归一化）、版本快照（每次更新 +1）、级联删除、附件登记
- Service 接线：启动自动建库 + 确保「未归档项目」+ 旧 JSON 无损迁移（幂等、只读不删原文件）；导出/上传目录迁至 office/exports、office/files；应用退出关闭 office.db
- 后端绑定：新增 ProposalProjectList/Create/Delete，ProposalCreate 支持 projectId；方案/章节数据带 projectId/version/sources
- 前端项目化：方案列表按项目分组（含未分组）、新建方案可选已有项目或新建项目
- 验证：新增 30+ 测试（网关/Store CRUD/树形持久化/版本/迁移/绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.13.0「记忆检索升级」(2026-08-04)

> 轻语记忆检索体系升级：检索双轨收敛为单轨（buildTierBBlock）+ FTS 全文检索接线（触发词之外的摘要词召回）+ 语音链路测试补齐（voice/tts）。
> tag v1.13.0

- 记忆检索双轨收敛为单轨：删除零调用孤岛 PrepareTurnContext → MemoryRetriever.Retrieve（重复实现整套检索但从未被消费），统一到 orchestrator.buildTierBBlock 单一主流程；精简 memory_retrieve/types，删除 RelevanceHint/RetrievalResult 类型与 7 个 TestRetrieve_* 死路径测试（净删 353 行）
- FTS5 全文索引修复（此前建而不用）：V2 外部内容表列名与主表不匹配（fact_id vs id）导致 rebuild 必失败 → V11 迁移独立表；rebuild 改为显式全量同步（修复 MaxOpenConns(1) 下 rows 未关时 Exec 死锁）；SearchFactIDsFTS 修复 MATCH 成功但空结果不降级 bug
- 中文全文检索：LIKE 降级升级为 2-gram 多模式（整句 + 相邻两字）——用户说「咖啡」能命中摘要「她喜欢喝美式咖啡」，解决触发词之外摘要词无法召回的问题
- FTS 全文检索接入 TierB 记忆上下文：Orchestrator 新增 FTSSearch 回调（app 层注入 repos 实现，避免循环依赖）；buildTierBBlock 把 FTS 命中事实补入候选（×1.3 加权）；persist 写回后自动重建索引（RebuildFactsFTS/RebuildEpisodesFTS 从零调用变为活跃）
- 语音链路测试补齐：voice 包 0% → 54.2%（31 测试，情绪→TTS 映射/配置校验/VAD 状态机/打断检测，核心状态机 100% 覆盖）；tts 包 0% → 20.9%（26 测试，分句/SSML 转义/RFC6455 握手向量/引擎回退链）
- 验证：新增 60+ 测试（FTS 重建/中文降级/2-gram 模式/引擎回退/语音状态机）；go vet clean + go build ./... 全过 + go test ./... 60 包全绿 + frontend tsc 0 错误

## v1.12.0「轻语记忆贯通」(2026-08-03)

> 轻语记忆系统与 hermes.db 全链路贯通（事实/情节/知识图谱三表持久化）+ TierB 记忆上下文补全（情节/触发词/关联扩散/记忆回声）+ 记忆中枢展示（情节 Tab + 三元组入图）。
> tag v1.12.0，构建 37,743,616 字节。

- 轻语记忆贯通 hermes.db（右脑落地的关键缺口）：Orchestrator 新增 EpisodicStore 运行时实例（原 handler 硬编码 nil，情节从未生成）；restoreWhisperState 从 hermes.db 恢复事实库/情节库/图谱（重启不丢记忆）；persistWhisperState 写回——事实合并写回（本会话以内存为准含退役态，保留其他会话事实），情节/图谱全量写回
- 修复 restoreWhisperState 早期返回缺陷：companion_state/chat_history 无行时提前 return，阻断记忆恢复（首次使用或清空历史后记忆永不加载）
- 修复数据竞争：KnowledgeGraph/EpisodicStore 无锁——extractTriples/情节生成在异步 goroutine 写 + PreLLMTurn 主流程读，Go map 并发写读会 fatal / slice append 实测丢数据 → 加 sync.RWMutex
- 修复 KnowledgeGraph.Query 评分 bug：原遍历全局 entityIdx 给所有三元组加分（图谱越大误命中越多）→ 改为匹配三元组自身 subject/object；单字实体可命中
- TierB 记忆上下文补全：情节检索（EpisodicStore.Search → 【相关记忆片段】）+ 触发词命中事实 boost 1.5x + 关联扩散（Top5 事实经 AssocIndex → 【关联记忆】）
- 记忆回声接线：buildTierBBlock 用检索事实 EmotionalContext 聚合记忆回声（Aff/Sec/Aro/Dom），PreLLMTurn 叠加到状态情绪（ApplyMemoryEcho/ComputeMemoryEchoFacts 从零调用孤岛变为生效，clamp ±100）
- 记忆中枢展示：轻语库新增「事实/情节」Tab（情节时间线流：情绪 emoji 角标 + 强度渐变条 + 关键词 chips + 轮次范围，按 AI 伴侣记忆库调研）；记忆图谱新增轻语三元组（实体节点 t: 复用 whisper 色 + predicate 关系边，共享实体去重）
- FactStore 新增 Restore（保留原 ID/退役态）/ListAll；KnowledgeGraph 新增 Restore/ListAll；影响面：记忆中枢轻语库/总览/归档/图谱首次读到真实数据
- 验证：新增 12+ 测试（FactStore Restore/ListAll、KG 并发/Restore/Query、EpisodicStore 并发、TierB 情节注入/记忆回声/关联扩散、集成往返、图谱三元组）；go test ./... 全绿 + go vet clean + go build 全过 + tsc 0 错误 + vite build 成功

## v1.11.0「界面体验深化 · 全站重设计」(2026-08-02)

> 设置中心外观细化升华（实时预览/三态显示/字体/密度/动效/强调色）+ 聊天 Markdown 消息体验 + 轻语面板 UI 重设计（角色状态头/气泡/情绪回复）+ 虚拟助手面板与角色卡详情重设计 + 轻语测试深化（21.8%）+ P3 archiveExporter 记忆归档导出。
> tag v1.11.0，构建 37,676,544 字节。

- 轻语测试深化：新增 33 个测试覆盖 memory_ingest 管线（LLM 抽取/自动退役/三元组/情节生成/隐私透传）、association_cold_start（文本重叠批量建边/孤儿链接/边去重）、paced_stream（气泡分隔/流排空）、memory 检索路径（触发词 boost/隐私过滤/budget 截断/情节检索/关联扩散/记忆回声）；覆盖率 16.8% → 21.8%
- 修复 paced_stream 真实 bug：流结束发完末段不 finishBubble → MarkDone 等待循环死锁（中文无断点文本必现）；FirstDisplayUnitLen 返回 rune 索引但调用方按字节切片 → 中文句号处切乱码（现转字节偏移）
- P3 archiveExporter：记忆归档导出（对齐 ackem archiveExporter）——README 索引（事实/核心/情节统计 + 领域分布表）+ 每个领域/子类一个 Markdown 文件；退役事实不入档；whisper.WriteArchive 写盘；绑定 GaeaWhisperExportArchive（hermes.db 数据源）+ GaeaPickDirectory 目录选择；记忆中枢轻语库「导出归档」按钮
- 模型中心：LLM 主卡片补三态状态徽章（● 运行中 / ○ 就绪 / ○ 已停止），对齐行业「模型状态三态」基准
- 设置中心：全局搜索（tab 关键词索引 + 实时过滤 + 自动切换到匹配分组 + 匹配计数）+ 即时生效统一徽章（SettingsSection instant prop，5 面板统一）
- 外观设置细化升华：外观实时预览区（主题+模式组合微缩界面，hover 主题卡即时联动「👆 预览中」）+ 主题色系大预览卡（氛围渐变 + 霓虹光晕 + 选中发光对勾）+ 显示模式三卡（暗色/亮色/跟随系统，matchMedia 实时派生，darkMode 保持 boolean 兼容现有消费者零改动）
- 外观设置扩展非颜色维度：字体设置（5 预置字体 + 字号 12-20 带预览）、界面密度（标准/紧凑，ConfigProvider token + .ui-compact CSS）、动效强度（完整/减弱，.ui-reduced-motion 全局禁用动画对齐 macOS 减弱动态）、强调色自定义（取色器覆盖 --gaea-glow/primary 令牌链，留空跟随主题）
- 聊天板块消息体验升级：Markdown 渲染（ChatMarkdown：代码块深色底+复制按钮/表格/列表，完成态渲染）+ 消息分组（同角色连续紧凑 + 组首头像）+ hover 操作栏（复制/朗读/重新生成显隐）+ 重新生成 + 建议卡点击即发送 + 头部元信息（你/gaea AI）+ 错误态红色气泡 + 玻璃内高光
- 轻语面板 UI 重设计：角色状态头（头像 + 关系阶段徽章 + L2 情绪 emoji 徽章 + 信任霓虹进度条 + 对话轮数）+ 消息气泡化（AI 琥珀玻璃气泡 / 用户粉紫渐变）+ 等待期 typing dots 修复（原空白光标）+ 快捷情绪回复 chips + 虚拟助手管理中心按钮卡片化（暖色玻璃入口卡）
- 虚拟助手面板 + 角色卡详情重设计：列表卡视觉区 TisorRadar → CompanionAvatar 粒子光球 + 性格标签 chips + 迷你五维条；详情卡大视觉区粒子球 + 五维区「左 TisorRadar 大雷达 + 右条形列表」并排
- 验证：go vet clean + go build ./... 全过 + go test whisper/app 全绿 + frontend tsc 0 错误（全流程每步验证）

## v1.10.0「科幻记忆中枢 · 架构归拢」(2026-08-02)

> 记忆中枢科幻感首页（中央 3D 图谱 + 霓虹玻璃卡片）+ 3D 图谱白屏修复（three-forcegraph → 3d-force-graph）+ AI 控制台归小说专用 + 小说专属代码归拢 components/novel/。
> tag v1.10.0，构建 37,617,152 字节。

- 记忆中枢科幻首页：中央 3D 图谱 + 四周霓虹玻璃模块卡片（hub.css 玻璃拟态 + 扫描线 + stagger），点击切库面板
- 3D 图谱白屏修复：误装底层库 three-forcegraph（class 需 new）→ 换 3d-force-graph（Kapsule 可调用）；图谱渲染时序修复（数据到达即构图）
- AI 控制台：默认关闭 + 记忆中枢隐藏（面板/按钮双层排除）+ 从 MainLayout 抽出为 components/novel/AIConsole.tsx 仅小说页挂载
- 小说代码归拢：16 组件 + hooks/api 全部移入 components/novel/（git 识别 18 rename），MainLayout 删 ~380 行小说控制台代码
- 界面配色跟随系统主题：hub.css 硬编码深色 → gaea 令牌（--bg/--fg/--accent），图谱背景/label 走令牌
- 验证：go test 57 包 0 失败（后端零改动）+ tsc -b + vite + wails build 全绿

「记忆中枢」(2026-08-02)

> 记忆体系三脑架构落地（命名 Hephaestus/Hermes + 主脑 Hephaestus.db + 左脑办公 SQLite + 右脑 hermes.db + 调度路由 + 知识库 RAG）+ 记忆中枢板块（七库统一管理）+ 3D 记忆图谱 + 成本库。
> tag v1.9.0，构建 36,552,704 字节。

- 命名体系：办公 agent → Hephaestus（火神），轻语 agent → Hermes（信使），gaea 之子女；AIgaea 产品类型统一
- 三脑架构：新建 Hephaestus.db（facts/profile/knowledge 三表 + 迁移链）；memory.Store 后端抽象（文件/SQLite 双后端，调用方零改动）
- 左脑接通：办公记忆 Markdown → Hephaestus.db 幂等迁移 + memory_get 工具 + boot/controller 默认 SQLite
- 右脑更名：whisper.db → hermes.db（首次打开自动迁移，保留备份）
- 调度路由：remember type=user → 主脑画像（profile 跨 agent 共享）+ DetectConflicts 冲突检测
- 知识库 RAG：迁入 Hephaestus.db + 共享向量层 internal/gaea/search（TF-IDF + 中文 bigram 余弦）混合排序
- 记忆中枢板块：知识库入口升级为多库面板（知识/成本/画像/办公三 tab/轻语只读/图谱），12+ 新绑定
- 3D 记忆图谱：three-forcegraph（type 着色/按库过滤/hover/点击详情），后端预计算 nodes+links
- 成本库：cost_entries 表（schema V2）+ cost 包 + CostLibrary（基础版）
- 画像冲突一键裁决（以画像为准 / 以 facts 为准）+ 轻语详情弹窗
- 验证：go test 全量 57 包 0 失败 + tsc + vite + wails build 全绿

「单模型架构 · 知识库板块」(2026-08-02)

> 办公板块删除双模型架构（Hermes/Hephaestus → 单模型）+ 知识库整合为独立板块（统一服务层 + 全文检索）+ AI 聊天双会话面板修复。
> tag v1.8.0，构建 36,084,224 字节。

- 办公板块：删除 Hermes/Hephaestus 双模型 agent（hermes.go 645 行 + 测试），runner 直接用 executor 单 Agent；
  config 删 planner_model/temperature/effort；前端删 Planner 配置/统计列/RunStatus 简化
- 单模型工作流梳理：系统提示词分层（DefaultSystemPrompt=领域知识 / SingleModelPrompt=执行纪律），
  boot 拼接保执行纪律不丢；删除 PlanCard 计划确认死链路（AskQuestion.Plan 字段全链清理）
- 知识库独立板块：knowledge.Service 进程级单例（工具/UI 统一走 ~/.gaea/knowledge），
  新增 GaeaKnowledgeSearch 全文检索（含正文）；KnowledgePage 板块页 + 导航五处注册；
  与记忆系统明确区分（显式知识 vs 隐式事实）
- AI 聊天：删除 ChatTopicSidebar 重复渲染（双会话面板修复）
- 验证：go test 全量 0 失败 + tsc + vite + wails build 全绿

## v1.7.0「设置中心重构 · 模型统一」(2026-08-02)

> 设置中心全面重构（小说/方案/办公/轻语/更新信息）+ 模型绑定统一左下角卡片 + 代码审查修复 9 项并发与绑定 bug。
> tag v1.7.0，构建 36,140,032 字节。

- 审查修复：轻语引擎 per-call 覆盖（删全局切换竞态）、小说 9 处绑定模型生效、ASR 校验、config.Save 加锁、
  manager 副本防 race、轮询进程合并、删 VoiceSetChatTarget 死绑定、finalTranscript 修复
- ComfyUI：findPython 加 standalone-env 兜底（ROCm）+ windows-standalone-build 标志 + 工作流结构测试
- 绑定模型卡：5 板块统一左下角浮动卡片（三态 + 启停），聊天页补渲染；预警只统计本地模型
- 模型中心：功能模型绑定独立标签页（2 列卡片网格）
- 设置中心 6 标签：小说（目录 C:\AI\xiaoshuo）/ 方案（新）/ 办公（完整可编辑）/ 轻语（设置合并）/
  系统（更新信息 + 删迁移）/ 语音
- 轻语界面移除设置弹窗，左栏 240 防卡片遮挡；删空态多余标题
- 验证：go test 全量 + tsc + vite + wails build 全绿

## v1.6.4「设置瘦身」(2026-08-02)

> 设置页删除与模型中心重复的「模型引擎」配置 tab，引擎/模型管理统一收敛到模型中心。
> tag v1.6.4，构建 36,106,240 字节。

- 移除设置页 EnginePanel tab（import/图标/副标题文案同步清理）
- 设置页保留：外观 / 工作空间 / 语音 / 绘梦 / 办公 / 系统
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.3「剧照模型可选」(2026-08-02)

> 补丁：小说/轻语角色剧照生成可选手模型（含 ComfyUI 本地 krea2/z-image-turbo/flux）。
> tag v1.6.3，构建 36,110,848 字节。

- 后端 GetImageBackendConfig：availableModels 恒含 ComfyUI 本地模型（无论全局后端），
  小说角色剧照弹窗即可选本地模型
- 轻语角色详情：生成剧照按钮旁加「出图模型」Select（自动加载可用模型，默认当前全局模型），
  handleGeneratePortrait 用所选模型
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.2「方案模型条 + 底栏常驻」(2026-08-02)

> 补丁：方案编写模型条不可见修复 + 底栏常驻（无项目时也显示模型监控与资源）。
> tag v1.6.2，构建 36,109,824 字节。

- OfficePage 顶部插入 FeatureModelBar（feature="office"）
- MainLayout 底栏 Footer 去掉 projectOpen 条件 → 常驻显示已启用模型 + CPU/内存/GPU
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.1「小说统一模型」(2026-08-02)

> 补丁：小说板块章节/角色/世界观 agent 统一接入 func_novel，整部小说用一个 LLM 模型（v1.6.0 已知限制 1 修复）。
> tag v1.6.1，构建 36,108,288 字节。

- worldview / character / chapter / analysis / outline 5 个 agent 全部加 featureModel + chat（带 func_novel 引擎覆盖），
  替换全部 `ChatSimpleStream(a.cfg.Model)` 调用 → `chat(ctx, ...)` + `ChatSimpleStreamWithOptions(EngineID: FuncNovelEngine)`
- 运行中切换小说模型即时生效（各 agent 动态读 cfg.FuncNovelEngine/Model）
- 验证：go test ./... 53 包全绿 + tsc -b + vite + wails build

## v1.6.0「语音交互 · 角色中心 · 功能模型」(2026-08-02)

> 语言交互全面落地：首页语音对话 + 核心 AI 助手 gaea（大地女神）+ 角色中心（助手即角色）+ 各功能板块独立模型绑定与资源监控。
> 23 提交（v1.5.0 后），tag v1.6.0，构建 36,108,288 字节。详见 releases/v1.6.0.md

### 首页语言交互（全新）
- 深空虚空首页：400px 大语言粒子球（连线网络+双环绕轨道+音量驱动）悬浮，8 透明玻璃卡片浮游，虚空微尘背景
- 「进入语音对话」本页直启麦克风：轻语 voiceManager 管道 → gaea 对话 → 识别/回复气泡 + TTS 朗读
- 布局响应式（粒子球随视口缩放）+ 语音卡片边框透明

### 核心 AI 助手 gaea = 大地女神盖亚
- 新增人格预设 gaea（首位全局默认）：大地之母，温厚沉稳包容滋养（五维 T85/I55/S20/O80/R50，goddess 标签）
- 启动确保 gaea 始终存在（修复旧数据缺失）；唯一「AI 助手」，其余标「角色」

### 角色中心（助手即角色）
- 管理中心改铺满主界面角色中心（角色卡网格 + 详情弹窗）；AI 角色剧照；小说↔轻语互传（参数统一）
- gaea 排第一 > 当前对话 > 启用 > 禁用

### 功能级模型绑定
- ai.Client per-call 引擎覆盖；config 10 键（chat/whisper/novel/office/gaea 引擎+模型）持久化
- 接入：聊天/轻语（含语音）/小说大纲；模型中心「功能模型绑定」UI
- 各窗口 FeatureModelBar（三态状态+一键启停）；底栏资源监控（CPU/内存/GPU+已启用模型）；超载弹窗警告

### 模型中心完善
- 恢复 ComfyUI 引擎（启停/状态/本机路径写死）；ComfyUI 图片模型（krea2/z-image-turbo）入墙
- 语音模型 STT/TTS 三段选择 + 持久化

### 已知限制（范围控制）
- 小说 agent 绑定仅大纲；办公 agent 绑定仅 UI；语音端到端依赖本地 herdsman 未 GUI 自动化实测

## v1.5.0「微信接通」(2026-08-02)

> 微信 ClawBot 通道全链路打通（微信发消息 → AI 自动回复），办公板块 v1.4.0 回归修复。
> 11 提交（v1.4.0 后），tag v1.5.0，构建 36,011,520 字节。
> 详见 releases/v1.5.0.md

### 微信 ClawBot 通道接通（核心）
- 认证修复：`Authorization: Bearer <token>` + `iLink-App-Id` + `iLink-App-ClientVersion`（数字编码 132099）
  + getUpdates 端点全小写 → 消除"会话过期"（errcode -14）
- 消息字段对齐**腾讯官方 openclaw-weixin**：`item_list[].type`（type=1 文本）+ client_id `{prefix}:{ts}-{hex}`
  （此前对齐社区 Rust SDK 误用 `item_type`，消息被静默丢弃 → 微信无回复的最终根因）
- 扫码绑定流程补全：need_verifycode 配对码二次查询（新绑定 WhisperWeixinQRStatusWithCode）、
  scaned_but_redirect/verify_code_blocked 处理；confirmed 返回完整 token + botId
- 会话过期状态透出：UI 显示"微信会话过期 · 需重新绑定"替代虚假"在线"
- 防脱敏 Token：前端含 `*` 用原值、后端含 `*` 拒绝
- 助手名字注入：BuildAckemCanonBlock 名字参数化 + Orchestrator.AssistantName，微信 AI 自称助手名
  （如"峨嵋"）而非默认"轻语"

### 办公板块修复（v1.4.0 回归）
- 右侧面板 Drawer 内联渲染（getContainer=false）+ 改为布局内 grid 列（340px）——不再跨页面残留/遮挡导航栏/凸出遮挡对话区
- 恢复对话区样式（修复 eb7d5e6 误删 chat-pane/transcript/markdown 样式族，输出框消失）
- 移除办公设置 + 明暗配色按钮（与系统重复，由主应用设置中心接管）

### 验证
- go test ./... 53 包全绿；tsc + vite build；wails build 产物 36,011,520 字节
- 微信全链路用户实测通过（收到消息 → AI 回复 → 发送成功）

## v1.4.0「架构统一」(2026-08-02)

> 后端四类重复收敛 + 前端两套 UI 体系统一。净删 6874 行（+555/-7429），10 提交，tag v1.4.0。
> 详见 releases/v1.4.0.md

### 后端架构重构（-526 行）
- HTTP client 统一：netclient 提升共享层，22 处裸 `http.Client` 收敛到 `NewSimpleClient`
- 工具函数收敛：office 死代码 + asr/whisper 手写字符串函数替换标准库
- 配置系统整合：删除 gaea config 零消费者死域（ComfyUI/Statusline/Notify/Serve/LSP）
- AI 通道唯一化：删除 provider/stream_client.go，通道=前端→agent→bridge→ai.Client→modelengine

### 前端两套 UI 体系统一（-6300+ 行）
- 死代码清理：老栈 29 废弃组件 + gaea 7 死代码 + 4 死 hook/util + highlight.js（-5800 行）
- 令牌层统一：gaea 颜色引用老栈 M3 令牌，删独立主题系统，暗亮双向联动
- 图标体系统一：66 个 lucide → @ant-design/icons，lucide 依赖移除
- 布局壳 antd 化：Layout/Drawer/Tooltip 用 antd，功能面板按分层原则保留自绘
- 死样式清理：.app/workspace-panel 样式族删除（-690 行）

### 验证
- go test 77 包全绿；tsc + vite build 通过；wails build 产物 gaea.exe

## v1.3.0「未来感 UI」(2026-08-02)

> 未来感 AI 多功能助手平台：深空星云 × 玻璃拟态 × 霓虹光效全链路统一 + 设置中心整合全部参数。
> 五批 UI 改造（+718/-268 前端），tag v1.3.0，构建 36.1MB。

### 设计系统
- 12 套主题新增 glow/glassBg/auroraBg 三令牌（App.tsx 注入 CSS 变量）
- 未来感 CSS 层：赛博网格 + 星点 + 漂浮光球背景、玻璃拟态、霓虹描边卡片、发光状态点、流光扫描线

### 框架与界面
- 顶栏玻璃化 + logo 光晕 + 菜单霓虹选中态；AI 中枢首页（真实模型状态 + 霓虹卡片墙）
- 模态框/弹层全局玻璃化；主题色块发光胶囊；页面切换过渡 + 霓虹加载态
- 聊天界面：玻璃气泡 + 渐变用户气泡 + 霓虹输入栏；侧栏玻璃化
- 绘梦面板玻璃化 + 画廊 hover 发光；底栏霓虹状态条

### 设置中心（整合全部设置参数）
- SettingsPage 重写为 7 分组 Tabs（外观/工作空间/模型引擎/语音/绘梦/办公/系统）
- 新增 settings/ 8 组件：主题发光选择器、图片保存目录、推理强度、引擎编辑、语音健康检测与阈值、绘梦后端、办公摘要、v4 迁移
- api/settings.ts 补 5 封装；设置页隐藏 AI 控制台

### 验证
- go test ./... 全绿（54 包）；go build / go vet 通过
- npm run build + wails build 产物 gaea.exe（36.1MB）
## v1.2.0「架构瘦身」(2026-08-01)

> 后端臃肿治理：绑定层按域拆分（197 方法迁移）+ 死代码清理 + 办公板块设置写路径接通
> + staticcheck U1000 全仓库清零 + 原子写实现去重。5 批提交，净删 ~600 行。

### 绑定层拆分（App 聚合 + 嵌入提升）
- App 结构体从 20+ 平铺字段收敛为 core + 4 域 State 嵌入引用（writing/media/whisper/office）
- 197 个方法按域归位；Go 嵌入提升保证 Wails 绑定不变，前端零改动
- 子服务持 app 反向指针协调跨域调用

### 办公板块设置写路径接通（20 个存根）
- 新增配置持久化（~/.config/gaea/config.toml 原子写）+ 改配置 → 保存 → 重建 controller
- Agent 参数 / 权限 / 沙箱 / 模型设置真实生效；Provider 增删改转发模型中心
- 回退 / 更新类改为明确错误语义，前端隐藏回退菜单

### 死代码清理（staticcheck U1000 全仓库清零）
- gaea 引擎：notify 包（2 文件）+ filteredSchemas + incompleteCanonicalTodos + runTurn
- whisper：candidate_collector.go 整文件（179 行孤岛）+ emotion_fusion 7 函数 + 3 常量
- ai：buildFluxWorkflow（已被 Z-Image/Krea 取代）

### 原子写去重
- 重建 fileutil.AtomicWrite，5 处手写重复实现收敛，净删 41 行

### 验证
- go build / go vet / 53 包测试全绿；staticcheck U1000 清零；前端 tsc 通过
- 新增测试：配置持久化往返 + 嵌入提升编译期断言

## v1.1.0「质量工程」(2026-08-01)
## v1.1.0「质量工程」(2026-08-01)

> 稳定性加固 + 架构瘦身 + 测试防线建立：21 处 goroutine recover、SSE 阻塞修复、
> 死代码清理 902 行、四个核心模块测试覆盖大幅提升（modelengine 97.1% / office 35.6% / ai 24% / whisper 16.8%）。

### 稳定性（3 批提交）
- 21 处 goroutine 无 recover 防护：桌面应用 panic 崩溃问题根治，最严重为
  controller.runGuarded turn 执行 panic 导致前端永久"生成中"，修复后复位状态 + Emit TurnDone
- SSE 流式发送 select+ctx.Done 保护：取消后 goroutine + HTTP 连接阻塞泄漏根治（hang 类问题根因）

### 架构整理（5 批提交，净删 902 行）
- gaea 模块：管理命令组 / 模糊编辑移植 / 宪法文件 / 7 处未使用字段别名
- 其他模块：剧照旧实现 / 世界观上下文 / 章节迁移等
- staticcheck U1000 84→25 处（剩余为 whisper 对齐 ackem + ComfyUI 迭代相关）

### 测试防线（5 批提交，+1900 行测试）
- modelengine 0%→97.1%、office/proposal 0%→35.6%、ai 7%→24%、whisper 13.1%→16.8%
- 修复 3 处测试暴露缺陷：GenerateSection 漏设 completed、UpdateSection/RemoveRawFile 静默失败

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`，完整说明见 `releases/v1.1.0.md`

## v1.0.0「品牌重塑 · 盖亚」(2026-08-01)

> wubigrok 正式更名为 **gaea**（盖亚，大地女神）——从「小说创作 Agent」升级为「多功能 AI 助手」。
> 全新品牌视觉：翡翠球体 + 破土嫩芽 + 灵感星芒，favicon / appicon 全量替换。

### 品牌重塑
- 全产品品牌替换：窗口标题「gaea · 多功能 AI 助手」、应用名/产物（gaea.exe）、UI 显示、文档、导出署名、图片下载前缀、日志文件
- Go module 重命名 `github.com/wubigork/wubigork` → `github.com/gaea/gaea`（233 个文件 import 同步）
- 新 logo 三件套：`frontend/public/favicon.svg` + `build/appicon.svg` + `build/appicon.png`（1024x1024）
- 版本号从 v5.x 重新起算为 **V1.0.0**（versioninfo.rc / wails.json / CHANGELOG）

### 数据兼容（老用户零丢失）
- 配置文件 `~/.gaea_config.json`（回退读取 `~/.wubigork_config.json`）
- 登录 token 回退读取 `.wubigork_token.json`（免重新登录）
- 项目标记目录 `.gaea/`（识别旧 `.wubigork/` 项目，v4 检测双向兼容）
- localStorage 键 `gaea_*`（回退读取旧键：聊天记录/人格/主题/绘梦模板保留）
- 内部 provider 注册名 `wubigrok` 保留（bridge provider 引擎兼容），UI 显示名全部为 gaea

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`

## v5.76.0「工程办公」(2026-07-31)

> gaeaW（土壤修复工程办公 AI 助手）完整移植：47 个工程工具 + Hermes/Hephaestus 双模型 agent + 6 个工程技能 + gaeaW 原生 UI，模型统一走 gaea 模型中心。

### 新板块：办公
- 后端移植 `internal/gaea/` 30 包（agent/tool/control/skill/command/plugin/knowledge/memory/boot/config），模型经 `provider/bridge` 接入模型中心（空模型动态跟随引擎切换）
- ai 包扩展工具调用支持（OpenAI 兼容 + SSE tool_calls 分片拼装，向后兼容）
- gaeaW 原生 UI 完整移植至 `frontend/src/gaea/`（App + 70 组件 + Tailwind v4），GaeaPage 渲染 gaeaW App
- 适配层：bridge.ts 90+ 方法名映射（Submit→GaeaSend 等），wubigrok 补齐 80+ Gaea* 绑定方法，事件格式精确对齐 WireEvent
- `.gaea/skills/` 6 个工程技能（场地调查/风险评估/修复设计/投标/数据报告）

### 发布
- 完整说明见 `releases/v5.76.0.md`

## v5.73.1「方案编写完善」(2026-07-31)

### 关键修复
- 嵌套章节（AI 大纲的 2/3 级）后端查找/更新全部改为递归展平，子章节可正常撰写/润色/改图/重命名
- 流式撰写内容真正落盘（原为副本指针，刷新即丢）
- 前端自动保存与图表插入支持子章节（updTree 递归更新）
- Word/MD 导出、覆盖/规范检查包含全部层级章节
- 浏览器上传招标文件改为 base64 落盘后转换（原路径为空必然失败）

### 新能力
- 大纲手工编辑：新增子章节/重命名/删除（含确认与编号重排），三个章节树均可用
- 流式撰写上下文补齐：需求、评分标准、废标条款、完整大纲、前一章节
- 失败时向前端发送 error 事件，不再卡死「生成中」

### 发布
- 完整说明见 `releases/v5.73.1.md`

## v5.20.0「精炼」(2025-07-26)

> 提示词全面重设计 + 死代码大清理 + 编译修复。净删 ~3000 行，prompts 23→15。

### 提示词重设计（4 个核心 prompt）
- **create-chapter**：硬编码字符串 → 模板化，「正在写这本书的作者」
- **chapter-generate**：「出版级」→「作者」，去 AI 味交由技能注入
- **plot-branch-browser**：「剧情策划人」→「story breaker」，固定 3 分支
- **worldview-agent**：「设定顾问」→「设定编辑」，代码块输出替换原文

### 死代码大清理（16 文件删除 + 后端精简）
- **删除 9 个废弃 prompt JSON**：brainstorm-ideas, chapter-review, outline-generate-detail, story-thread-chat, story-thread-generate, worldview-chat-section, worldview-check-consistency, worldview-generate-all, bootstrap-reference-summarize
- **删除 6 个前端组件**：AIAssistSheet, BrainstormModal, DialogueModal, StoryBibleModal, StoryBibleSteps, BeatToProse
- **删除 brainstorm_handler.go**（59 行）
- **Go handler 死代码清理**：chapter_handler (-178), copilot_handler (-237), create_chapter_handler (-82), outline_handler (-108), plot_branch_handler (-80), project_handler (-107), worldview_handler (-42)
- **核心模块精简**：chapter.go (-301), outline.go (-203), worldview.go (-215)
- **前端页面清理**：MainLayout (-60), CreatePage (-46)，其他页面移除死引用

### 编译修复
- chapter_handler.go 缺失函数闭合 `}` 修复
- copilot_handler.go 6 个未使用 import 清理
- chapter_handler.go 未使用 "strings" import 清理

## v5.17.0「重塑」(2025-07-26)

> 从 v5.7.1 分支重建。移除移动端代码，新增「创作」面板，章节节点树 + 分支系统。

### 移除
- 删除 `mobile/` 独立移动端应用
- 删除 `internal/mobile/` Go HTTP 服务
- 删除 MobileTabBar、MobileDrawer、useIsMobile 等全部移动端组件

### 创作面板（全新）
- **三栏布局**：左侧节点树 | 中间编辑器 | 右侧 AI 控制台（全局复用）
- **CreateChapter API**：一步到位，跳过对话和大纲阶段，直接生成正文
- **三分支构思**：AI 读取设定 + 前文摘要 → 生成 3 个剧情方向 → 用户选择 → 生成正文
- **节点树系统**：每章永久节点（ID、摘要、parent_id），支持分支/覆盖/删除/重生成
- **摘要注入**：每章生成后自动提取摘要存入节点树，前文注入改为节点摘要拼接

### AI 控制台增强
- 展开 REQ 查看完整 SYSTEM/USER prompt
- 展开 OK 查看 AI 完整响应（修复 response 事件缺失 content 字段）
- 固定宽度 380px + alignSelf: stretch

### 后端
- `internal/app/create_chapter_handler.go` — CreateChapter 一步生成+保存
- 正文末尾 `---CHAPTER_SUMMARY---` 标记自动分离摘要
- `GenerateOutlineWithDialogue` prompt 限制章节数，避免浪费 tokens

## v5.7.0「凝练」(2026-07-06)

> 7 轮组件化拆分迭代：42 文件变更，+3,244 / -1,946，(window as any) 和 @ts-ignore 全面清零

### ⚛️ 组件化重构（7 轮）

#### R3 — 去重 & 组件化
- **BranchSelectorPanel**: 提取 PlotBranchModal + NextChapterModal 共享分支选择 UI（187 行）
- **StoryBibleSteps**: StoryBibleModal 5 步子组件拆分，主组件 489→288 行
- **净效果**: +701/-448，两 Modal 各减少 ~90 行重复代码

#### R4 — 书架模块
- **提取 4 子组件**: WelcomePage / BrainstormModal / CreateNovelModal / ProjectCardItem
- **HomePage**: 640→280 行 (-56%)，Ctrl+N 快捷键 + 骨架屏
- **Utility**: 提取 formatRelativeTime + delay 到 utils/time.ts
- **类型化**: 新增 BrainstormIdea 接口，消除 any
- **移动端**: 统一 btnBase 样式，移除无效 CSS animation

#### R5 — 世界观模块
- **API 抽象层**: api/worldview.ts，消除 6 处 @ts-ignore
- **SectionNav**: 共享维度导航（桌面固定侧栏 / 移动端下拉面板），消除 ~90 行重复
- **ConsistencyReport / MapFullscreen**: 提取行内子组件
- **WorldviewPage**: 490→328 行 (-33%)

#### R6 — 角色模块
- **API 抽象层**: api/character.ts，消除 11 处 @ts-ignore
- **RelationshipModal / OrganizationEditModal / PortraitLightbox**: 提取行内 Modal
- **OrgField**: 移入 CharacterFormHelpers.tsx
- **renderCharEditor()**: 消除桌面/移动端 ~40 行重复
- **CharacterPage**: 489→396 行 (-19%)

#### R7 — 写作模块
- **Wails 类型导入**: 消除 7 处 (window as any)
- **ReviewResult**: 审稿组件（评分圆环 + 优势/不足/改进建议）
- **OutlinePanel**: 大纲面板独立组件（折叠/展开 + 卷/章树形渲染）
- **超长行拆分**: 最长行 620→314 字符，提取 createTabData / handleFinalize / handleNextChapterGenerate

#### R8 — 绘梦模块
- **API 抽象层**: api/image.ts，消除 12 处 (window as any)
- **ResultGallery 增强**: 添加 onDelete 支持
- **PromptBar / CustomTemplateModal**: 提取底部输入栏和模板编辑弹窗
- **ImageGenPage**: 665→521 行 (-22%)

#### R9 — 设置面板
- **API 抽象层**: api/settings.ts，消除 ~15 处 (window as any) + @ts-ignore
- **SettingsCard**: 统一卡片组件，消除 7 处重复 cardStyle
- **SettingField**: 统一「标签 + Input/Select」模式
- **handleToggleMobile / handleMigrate**: 提取命名函数
- **SettingsPage**: 475→356 行 (-25%)

### 🧹 全局清理
- **(window as any) 清零**: 前端所有页面全部替换为 wailsjs/go/app/App 类型导入
- **@ts-ignore 清零**: 通过 API 抽象层消除所有类型跳过
- **catch(_) 清零**: 全部替换为 catch(err) + console.error
- **mountedRef 移除**: React 18 自动批处理已解决

### 📦 构建
```
wails build -o build/bin/wubigork-v5.7.0.exe
```
- 新增 28 文件，修改 14 文件

## v5.6.0「移动端远程操控」(2026-07-03)

> 移动端完整功能实现：RPC 桥接、SSE 流式、静态文件服务、CORS、QR 码、IP 检测

### 📱 移动端功能
- **HTTP RPC 调度器** (`internal/mobile/rpc.go`) — 泛型反射调用 App 方法，黑名单过滤生命周期方法
- **SSE 流式推送** (`internal/mobile/sse.go`) — StreamHub 事件广播 + EventSource 频道
- **前端桥接层** (`frontend/src/api/bridge.ts`) — 非 Wails 环境自动创建 `window.go.app.App` HTTP 代理
- **Runtime 多填充** (`frontend/src/api/runtimePolyfill.ts`) — `EventsOn/Off/Emit` + SSE EventSource 多填充
- **静态文件服务** — embed.FS 生产模式 + `dist/` 前缀兼容 + SPA fallback
- **CORS 中间件** — 手机浏览器跨域访问支持
- **IP 检测** — 虚拟网卡过滤（Docker/WSL/Hyper-V 等），优先 WiFi 真实 IP
- **端口预检** — `net.Listen` 预检端口可用性，占用时立即报错

### 🐛 修复
- **移动端页面白屏** — `serveStaticOrSPA` 路径前导斜杠导致 embed.FS 拒绝读取
- **事件名不一致** — 前端 `prose-stream` → `beat-prose-stream`
- **render 崩溃** — HomePage `card.title` null 守卫
- **IP 错误** — UDP 拨号法回退到 Docker IP，接口名过滤法替代

### 📦 构建
```
wails build -o build/bin/wubigork-v5.6.0.exe
```
- 新增 4 文件，修改 12 文件

## v5.5.0「精炼」— 全量代码优化 + 工程质量加固 (2026-07-03)

> 两轮迭代：后端去重 + 前端巨型组件拆分集成 + 死代码清理 + 类型安全加固

### 🏗 后端重构（12项）
- **EstimateTokens 去重**: context+memory→util，修复中文检测用 unicode.Is
- **ExtractJSON 去重**: style→util
- **401 重试合并**: ai/client.go 三合一 → refreshAndRetry()
- **config.Save 重构**: 18-case switch→map 注册模式
- **marker 解析抽象**: ParseMarkedSections()，worldview/character 共用
- **for i:=1;;i++→ForEachChapter**: 8处死循环统一，缺失章节 continue 而非 break
- **chapter.Generate 拆分**: 200行→3函数(buildGenerateContext/streamAndRetry/postProcess)
- **ChatStream 拆分**: 请求构建+SSE 解析分离
- **mobile 死代码清理**: 删除 SPAHandler，修复 _ = path
- **app.go 类型安全**: comfyUICmd 删除，distFS interface{}→fs.FS
- **skill YAML 加固**: Windows \r\n 兼容，修正 scan() 注释
- **slog.Warn 统一**: 35+处→Agent.warnRead()

### ⚛️ 前端重构（11项）
- **ChapterPage**: 1170→227 行 (-80%)，集成 ChapterEditor/AIAssistSheet/useChapterStream
- **OutlinePage**: 1188→350 行 (-70%)，集成 OutlinePanel/ThreadPanel/DialogueModal/useOutlines
- **StoryBibleModal**: 488行→5步子组件+路由壳
- **PlotBranchModal+NextChapterModal**: 共享 usePlotBranch hook
- **4 新 hooks**: useChapterStream/useOutlines/useWailsEvent/usePlotBranch
- **errorHandler**: handleError+wrapAsync 统一错误处理
- **OutlineNode 类型冲突修复**: api/outlines.ts 删除重复定义
- **StepCreate.tsx**: onChange 死代码修复
- **HomePage Creating**: mount guard 防 unmount 后 setState
- **LorebookModal/SkillModal**: AbortController cleanup
- **TTSPlayer**: onStatusChange ref 冻结

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.5.0.exe -ldflags="-s -w" .
wails build -o build/bin/wubigork-v5.5.0.exe
```
- 新增 16 文件，修改 ~25 文件，删除 ~900 行
- 零新依赖

## v5.4.0「锻造」— 稳定性修复 + 本地生图增强 + 功能打磨 (2026-07-03)

> 修复 13 项 Bug，新增 ComfyUI 一键启停、系统状态监控、角色 Agent、图片管理增强

### 🐛 Bug 修复（13项）
- **xAI 默认模型名**: `"flux"` → `"grok-imagine-image-quality"`，修复云端生图失败
- **xAI API size 参数**: API 不再接受 `size`，请求中清除该字段
- **ComfyUI 编码崩溃**: Python 3.13 + GBK 控制台无法编码 emoji → 补丁 logger.py + prestartup_script.py
- **Login() 后 ComfyUI 丢失**: 重新登录后自动恢复后端配置
- **小说切换数据不同步**: 5 页面 (世界观/角色/大纲/章节/画布) 监听 projectPath 重新加载
- **Token 刷新无超时**: `RefreshAccessToken` 用 `http.DefaultClient` → 15s 超时，防止永久卡死
- **生成失败消息**: 返回具体原因而非笼统的「所有生成尝试均失败」
- **ComfyUI 执行错误**: 错误解析索引 msgArr[2]→msgArr[1]，正确提取异常信息
- **右侧面板隐藏**: 绘梦右侧栏改为始终显示（文件夹按钮在无历史时也可见）
- **CMD 弹窗**: `getCPUUsage` 调用 wmic 时隐藏窗口
- **防重复提交**: 绘梦生成按钮加 useRef 锁

### ✨ 新功能
- **ComfyUI 一键启停**: 绘梦顶部 🟢/⚫ 状态 + 启动/停止按钮，设置页配置安装路径
- **系统状态监控**: 绘梦右侧栏显示 CPU% + GPU 显存占用，3s 刷新
- **角色 Agent**: 角色页底部 ChatPanel，AI 对话自动创建/编辑角色
- **图片单张删除**: 画廊和历史缩略图支持逐张删除
- **图片文件夹快捷打开**: 右侧栏 📁小说图片 / 🖼生成图片 目录
- **世界观地图双击全屏**: AI 地图和 3D 势力图支持双击放大
- **图片自动保存**: 未配置专用目录时自动存到 `<小说>/images/`

### 🎨 UI/UX
- 导航 `项目` → `书架`，其余文案 `项目` → `小说`
- 大纲页卷结构面板拉长填满窗口
- 清理冗余文件回收 ~79MB

### 📦 构建
- `build/bin/wubigork-v5.4.0.exe` — 16MB

---
## v5.3.0「暗夜」— 全局配色重设计 (2026-07-03)

> 暗夜系列 6 套主题 — 手工调色，表面色与强调色分离，完全替换 M3 Tonal Palette

### 🎨 主题系统重做
- **6 套暗夜主题**: 暗夜青（默认）/ 暗夜紫 / 暗夜玫 / 暗夜金 / 暗夜苔 / 暗夜墨
- **表面色与强调色分离**: 不再从单一 seed 派生所有颜色，每套有独立的表面色温
- **手工调色**: 替换 M3 `generateTonalPalette` 自动生成，消除同色系单调问题
- **亮色模式**: 6 套暗夜主题各自对应一套亮色变体
- **零破坏**: 旧 localStorage 中的主题名自动回退到默认暗夜青

### 🔧 累积修复 (v5.2.1 → v5.3.0)
- Z-Image sampler 节点引用 14→13
- 大纲 prompt 矛盾约束（恰好5卷 vs 保留不修改）
- Lightbox z-index 1000→1050（剧照全屏被遮挡）
- AI 绘梦模板系统：类别下拉 + 自定义 + 去重风格

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.3.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **3 文件变更**: +70/-188 行
- **零新依赖**

---
## v5.2.0「凝形」— Bug修复+工程质量+性能优化 (2026-07-03)

> 基于 v5.1.0 审计，修复 24 项问题：10 Bug修复 + 8 工程质量 + 6 性能与体验

### 🐛 Bug 修复（10项）
- **Z-Image 尺寸参数**: width/height 正确映射到 Z-Image ratio（1:1/3:2/4:3/16:9/2:1），不再被忽略
- **Context cancel 泄漏**: `copilot.go` `_ = cancel` → `defer cancel()`，防止 goroutine 泄漏
- **io.ReadAll 错误忽略**: ComfyUI `queuePrompt`/`checkHistory` 两处错误现在正确返回
- **os.MkdirAll 错误忽略**: `export.go` 目录创建失败不再静默继续
- **NovelsDir 硬编码**: `D:\AI\xiaoshuo` → `~/wubigork-novels`，跨系统兼容
- **CancelGhost 空实现**: 真正取消进行中的 Ghost 补全 goroutine，前端收到 `cancelled` 事件
- **事件监听器清理**: `EventsOn('xai-output')` 添加 useEffect cleanup 防止内存泄漏
- **静默吞异常**: 全站 `catch (_) {}` → 带标签的 `console.error`
- **loadOutlines 重复**: ChapterPage/OutlinePage 重复实现 → 提取到 `api/outlines.ts`

### 🏗 工程质量（8项）
- **API 抽象层**: 新建 `frontend/src/api/outlines.ts`，封装后端调用
- **Wails 类型声明**: 新建 `frontend/src/types/wails.d.ts`，`AppAPI` 接口声明 100+ 方法，消除 `@ts-ignore`
- **TTS 配置持久化**: `config.go` 新增 `tts_port`/`tts_backend`/`tts_speed` 的 Load/Save
- **Save() 常量化**: 定义 19 个 `Key*` 常量替代硬编码字符串，防止拼写错误
- **SafeGo 工具**: `util/util.go` 新增 `SafeGo(fn)` — goroutine + panic recover + 日志
- **requirePM() 辅助**: `app.go` 新增读锁获取项目的方法，消除 handler 重复的 nil 检查
- **废弃文件清理**: 删除 `internal/ai/context.go`

### ⚡ 性能与体验
- **移动端 API 对接**: `mobile/handlers.go` `ProjectsProvider` 回调，替换硬编码 JSON 占位
- **统一日志**: 新建 `frontend/src/utils/logger.ts`，四级日志（debug/info/warn/error），生产环境可关闭
- **页面保活**: `MainLayout` `visitedPages` Set 机制，切 tab 不再销毁组件丢失状态
- **大纲五阶段固定卷**: `ContinueOutline(5)` 全量替换（起承转合终），简化 AI 生成流程
- **Z-Image NSFW LoRA**: 工作流新增 `LoraLoaderModelOnly` 节点 (strength 0.7)
- **世界地图图片**: `SaveWorldMapImage`/`GetWorldMapImage` Wails API，base64 存取

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.2.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **27 文件变更**: +1095/-346 行
- **新增 3 文件**: api/outlines.ts / types/wails.d.ts / utils/logger.ts
- **删除 1 文件**: internal/ai/context.go
- **零新依赖**

---
## v5.1.0「绘梦师」— Z-Image-Turbo 极速生图 + AI 绘梦工作台 + 角色剧照打通 (2026-07-03)

> 本地 Z-Image-Turbo 8 步生图（48s）、AI 绘梦双栏工作台重做、角色剧照与 AI 绘梦打通、角色详卡三段式布局

### ⚡ Z-Image-Turbo 本地生图集成
- **新模型支持**: ComfyUI-ZImagePowerNodes + GGUF Q5_K_M (~5.2GB) + Qwen3-4B Q4_K_M (~2.5GB) text encoder
- **8 步极速**: ZSamplerTurbo2Simple，48 秒出图（Flux 需 60-90 秒 20 步）
- **双模型切换**: Flux Dev / Z-Image-Turbo，前/后端完整支持，配置持久化
- **VRAM 优化**: UNet partial offload ~4GB 加载 + 1.4GB 卸载，8GB 显存刚好够用
- **v5.0.0 补全**: 下载脚本 + 工作流测试验证 + 节点参数自动探测

### 🎨 AI 绘梦工作台重做
- **双栏布局**: 左栏 320px 控制面板 / 右栏自适应结果区，桌面端专业工作台体验
- **负向 Prompt**: 可折叠输入区，Flux 工作流节点 8 注入
- **种子控制**: InputNumber + 🎲 随机，精准复现同一张图
- **批量生成**: 一次 1-4 张，循环提交 + 每张独立计时
- **20 个预设模板**: 创作/写实/风格/构图 4 大类，点击自动填入正/负向 prompt
- **全屏灯箱**: 键盘导航（← → Esc），显示种子/模型/尺寸/耗时，一键下载/重用参数
- **会话历史**: 底部横向缩略图画廊，点击回溯，支持清空
- **进度预估**: 按钮下方显示「预计 ~90s」/「上次 48s」，根据模型和数量动态计算
- **6 个新组件**: PromptPanel / GenControls / GenButton / ResultGallery / Lightbox / HistoryStrip
- **移动端适配**: 单栏控制面板 + 底部 Drawer 结果抽屉

### 👤 角色剧照与 AI 绘梦打通
- **剧照后端统一**: GeneratePortrait 改用 `cfg.ImageModel`（支持本地 Z-Image-Turbo/Flux），中文 prompt
- **AI 绘梦→角色剧照**: Lightbox 新增「设为剧照」下拉选择器，选角色自动保存到 `portraits/<id>.png` 并写回 `characters.json`
- **角色卡片缩略图**: 卡片列表圆形头像显示 `portrait_url`（有图时），无图时回退图标
- **SetCharacterPortrait API**: Go `character.Agent.SetPortrait()` + Wails 绑定

### 📋 角色详卡三段式重做
- **概览区**: 剧照 160×160 左侧主视觉 + 姓名/类型/状态/性格标签 + 操作按钮
- **Tab 分组**: 「📋 档案」（基本信息+性格+外貌+身材+动机+弧光+背景）/「🔗 关系」（组织+人物关系+出场章节）
- **剧照三态**: 有图悬浮重生成 / 生成中 Spin / 无图虚线框 CTA
- **删除按钮**: 移至右上角，减小误触

### 🔧 修复
- **config.go**: `comfyui_url` Save case 空白无赋值（严重bug）→ 正确赋值 `ComfyUIURL`
- **config.go**: `image_save_dir` Save case 错误写入了 `ComfyUIURL` → 修正为写入 `ImageSaveDir`
- **config.go**: `ImageModel` 配置字段新增 + Load/Save 支持

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.1.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **新增 7 文件**: 6 个前端组件 + 模板数据
- **修改 9 文件**: Go 后端 5 + 前端 4
- **零新 npm 依赖**

---

## v5.0.0「织梦者·移动」— 移动端远程操控 + M3 设计语言 (2026-07-02)

> 手机远程操控桌面端、Material Design 3 设计迁移、ComfyUI LoRA 生图链、20 commits

### 🎨 Material Design 3 全站迁移
- **M3 Tonal Palette**: 轻量内联实现，5 套主题从种子色自动生成 13 级色调调色板
- **Ant Design Token 驱动**: ConfigProvider 模拟 M3 视觉（surface/surfaceContainer/elevation/outline）
- **CSS 变量标准化**: `--md-sys-color-*` / `--md-sys-elevation-*` M3 命名体系
- **Layered Glass → M3 Surface**: `.glass-panel` → `.md-surface-container`，移除全站 backdrop-filter
- **M3 交互**: CSS-only ripple 涟漪效果、`.md-card` elevation hover lift、触控 44px 最小区域
- **25 变量兼容填充**: 旧 CSS 变量名 → M3 token 映射，零破坏迁移

### 📱 移动端远程操控
- **HTTP 服务**: Go `internal/mobile/` 包 — LAN IP 检测 + 二维码生成 + SPA fallback
- **响应式双导航**: 桌面端 Sidebar / 移动端 Bottom TabBar + Drawer + AppBar
- **Container Query 布局**: `useMediaQuery` hook + 3 级断点（compact/medium/expanded）
- **通用移动组件**: MobileSheet（底部滑出面板）、LongPressable（长按 500ms + 震动反馈）
- **全页面适配**: 9 个页面 + 5 个组件 — 3D 关系图→2D SVG 降级、Canvas→Grid、编辑器→全屏+Sheet
- **设置页**: Switch 开关 + 二维码面板，手机扫码即连

### 🎨 ComfyUI 集成增强
- **内嵌 Web UI**: ImageGenPage 一键切换 iframe 控制台，无需另开浏览器
- **LoRA 链**: Flux.1 Dev → Realism (0.8) → NSFW (0.6)，LoraLoaderModelOnly 级联
- **URL 持久化**: `GetConfig('comfyui_url')` 自动加载

### 🔧 工程质量
- **零新依赖**: 前端无新 npm 包，Go 纯 `net/http` 标准库
- **Wails 绑定**: 3 个新方法 `StartMobileServer` / `StopMobileServer` / `GetMobileServerStatus`
- **20 commits**: 每个 Task 独立审查（Spec ✅ + Quality Approved），2 个 fix round
- **前后端双编译**: `tsc -b && vite build` + `go build` 零错误

### 📦 构建
```bash
wails build                              # 生产包 (build/bin/wubigork.exe, 16MB)
go build -o build/bin/wubigork-v5.0.0.exe .  # 开发包 (不含嵌入 dist)
```

---

## v4.0.0「织梦者」— AI 原生叙事工作室 (2026-06-29)

> 对标 Scrivener + Sudowrite + Cursor + Obsidian + Notion，从「AI 辅助写作工具」跃升为「人机共创的叙事操作系统」

### 🏗 数据基础: 场景优先架构
- **场景引擎**: 章节由场景组合，场景级 CRUD + 元数据（POV/地点/情感/标签），`Stitch()` 拼接编辑
- **快照系统**: 行级 diff 存储，自动快照 + 时间线恢复 + 双栏对比
- **v4 项目结构**: `chapters/NNN/scenes/` 目录，非破坏性 v3→v4 迁移（原件备份到 `_v3_backup/`）

### ✍️ AI 协写系统 (对标 Cursor + Sudowrite)
- **Ghost Text**: 内联 AI 补全，800ms 防抖，Tab 接受/Esc 取消，光标追踪
- **Cmd+K**: 6 个预设指令 + 自定义指令，选中文本 → AI 原地编辑 → diff 预览
- **Diff Review**: 行级 diff（绿色新增/红色删除），⌘Y 接受/⌘N 拒绝
- **Beat-to-Prose**: 两栏布局，AI 生成叙事节拍 → 逐个展开为正文，完成自动存为 v4 场景

### 🕸 叙事知识图谱 (对标 Obsidian + Notion)
- **[[wiki-link]]**: 双向链接解析 + 反向链接索引 + 未链接提及检测
- **EntityDB**: 6 种实体类型（角色/地点/物品/事件/概念/组织），关联查询
- **一致性守护**: 跨章属性冲突检测（眼睛/发色）+ 角色状态异常（Dead→出场）+ 时间线校验
- **StoryGraph**: Canvas 力导向 2D 图谱 + 悬停高亮 + 6 色图例

### 🧠 上下文智能引擎 (对标 NovelAI + Cursor)
- **语义记忆**: BM25 检索 + 中文 2-gram 分词，零外部依赖
- **Lorebook 2.0**: 触发式关键词注入 + Token 预算可视化（7 分区堆叠条形图）
- **多模型路由**: 8 种任务类型 → 最优参数自动映射（温度/Token/推理深度）
- **@-mention**: 弹出实体选择器，@角色/@地点/@概念 精确控制 AI 上下文

### 📊 视觉叙事工作台 (对标 Scrivener Corkboard)
- **StoryTimeline**: 水平滚动时间线 + 彩色情绪卡片 + 字数比例条 + 悬停详情
- **EmotionCurve**: Canvas 贝塞尔曲线 + tension/valence 切换 + 渐变填充
- **CharacterHeatmap**: 角色×章节矩阵 + POV 红色标记 + 颜色强度
- **CanvasCards**: SVG 连线软木板 + 30%-200% 缩放

### 🌐 生态平台
- **导出 2.0**: HTML 导出 + 3 个 Compile 模板（网文阅读/出版审阅/极简纯净）+ 暗色模式
- **写作仪表盘**: 8 个写作成就 + 统计总览 + 章节柱状图 + 目标进度
- **风格档案**: AI 分析 3 章样本 → 8 维风格特征 → 自动注入 prompt

### 🔧 工程质量
- **Go**: 6 个新包 (scene/snapshot/graph/memory/context/visual) + 34 项测试
- **React**: 14 个新组件 + TypeScript 零错误
- **API**: 60+ Wails 绑定方法
- **兼容**: v3 项目完全兼容，迁移可手动触发

---

## v3.1.0 — Layered Glass 视觉重设计 (2026-06-21)

### 🎨 UI 全面升级 — "Layered Glass"
- **设计系统**: 17 个新 CSS 设计令牌 (accent-rgb, shadow-sm/md/lg/glow, border-subtle, bg-deep/base/elevated/glass, radius-sm/md/lg/xl, transition-fast/normal/slow)，4 套主题统一渐变+辉光
- **玻璃态全站**: 所有面板/卡片 `backdrop-filter: blur(8px)` + 半透明背景 + 柔和边框替代 thin 1px 实线
- **导航重设计**: 顶栏 sticky glass + 菜单去下划线/pill 选中态 + 底栏玻璃态
- **XAI 控制台**: 从等宽字体调试面板 → 圆角玻璃浮层 + 系统字体 + 彩色左边线
- **首页**: 项目卡片 glass-card 悬停上浮 + 空状态品牌水印 + 按钮辉光
- **写作页**: 玻璃侧边栏 + pill 标签页 + 编辑器内凹 inset 阴影 + 工具栏统一玻璃按钮
- **对话框**: Ant Design Modal 全局玻璃覆盖 + 弹簧入场动画 + 遮罩 blur
- **骨架屏**: 三处 loading 从裸 Spin 升级为 Ant Design Skeleton 骨架屏
- **无障碍**: `prefers-reduced-motion` 全局尊重

### ✨ 功能增强
- **TTS 语音朗读**: VoxCPM 本地神经网络合成 + Edge TTS 在线自然语音 + Windows SAPI 零延迟
- **写作页标签页优化**: 可横向滚动 + 关闭未保存确认弹窗
- **设置页**: 工作空间目录可配置

### 🔧 代码质量
- **DRY 清理**: 消除 9 处 `C()` 重复定义 → 统一 `import { C } from utils/theme`；消除 RelationGraph 颜色映射重复；提取 `sortNodes` 到 `utils/outline.ts`
- **错误处理**: 消除 15+ 处 `json.Marshal`/`io.ReadAll` 静默吞错误
- **死代码清理**: 删除 `internal/xai/` + `pkg/novel/` 重复函数 ~350 行

### 🛠 修复
- Edge TTS WebSocket 重写 + 引擎链降级 (SAPI → Edge → VoxCPM)
- VoxCPM 编译为静态链接，消除 DLL 依赖
- exec.Command 隐藏控制台窗口，消除朗读时弹 cmd
- NovelsDir 默认值不再依赖配置文件

---

## v3.0.0 — 革命性跃升 (2026-06-20)

### 🚀 核心创新
- **剧情分支选择器**: AI 推理 3-5 个下一章方向，用户选用或手工录入，自动写入大纲并同步角色/世界观
- **全书发展编辑**: AI 审读全书生成 1500 字编辑信 + 5 维评分 + 角色弧光诊断，对标 $3500 人工编辑
- **自我演化引擎**: 每章生成后自动分析 → 建议 Lorebook 词条 → 追加世界观空维度 → 记录伏笔变化
- **跨模块协作**: 大纲角色点击跳转角色页 + 角色出场章节查询 + 世界观一致性一键编辑

### 🎯 v2.0.0 功能 (2026-06-20)
- Story Bible 引导式创建（5 步向导：灵感 → 一键生成 → 角色优化 → 大纲优化 → 开写）
- 情节画布（水平时间线可视化全文章节 + 角色色条 + 品质情绪标注）
- Skill 管理面板（浏览/导入/创建自定义写作风格 Markdown）

### 💡 v1.5.0 功能 (2026-06-20)
- Lorebook 词条系统（定义概念 → AI 写作时自动注入相关上下文）
- 写作统计仪表盘（每章字数柱状图 + 品质趋势进度条）
- AI 审稿（5 维评分环形图 + 改进建议高亮卡片）
- Brainstorm 脑暴面板（输入题材 → AI 生成 6 个核心点子 → 选用创建）

### 🔧 v1.4.0 功能 (2026-06-20)
- 右键 AI 操作（选中正文 → 丰富描写/扩展场景/重写此段）
- 多章节标签页（同时编辑多个章节，独立状态）
- 大纲拖拽排序（Ant Design Tree draggable + 批量保存）
- 专注写作模式（全屏沉浸，隐藏侧栏和 Agent）

### 🛠 v1.3.2 修复 (2026-06-20)
- 大素材归纳压缩（先 AI 归纳关键信息再注入 prompt，防止上下文溢出）

---

**技术栈**: Go + Wails v2 + React 19 + Ant Design 6 + Three.js + d3-force-3d  
**构建**: `go build -o build/bin/wubigork-v4.0.0.exe .`  
**许可证**: MIT
