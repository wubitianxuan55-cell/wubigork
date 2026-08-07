# gaea · 多功能 AI 助手

# gaea · 多功能 AI 助手

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
