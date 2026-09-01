# 任务进度

> 最后更新: 2026-09-02（v4.30.0「办公 UI 化繁为简 · 第二刀」：产物置前/行级降噪/
> 命令面板视图重排/预览两档——绑定面 552 零变更）

## 当前状态

- **最新发布：v4.30.0（2026-09-02）「办公 UI 化繁为简 · 第二刀：产物置前/行级降噪/
  命令面板视图重排/预览两档」**：git tag v4.30.0；基线 v4.29.0；绑定面 **552 零变更**
  （纯前端呈现重组）。用户点名「继续优化完善 gaea」——从 v4.29.0 欠账清单收齐四项，
  红线不变（**简化≠删除功能**，被隐藏的信息全部有确定性寻回路径：title/aria/悬停/激活
  即见）。①**产物生成自动置前/角标**（Devin Auto-open 式）：App diff 会话内新产物路径
  （首现即新，会话切换重置基线——恢复会话不误标「新」）→ 产物 tab 角标（未查看数，激活
  即清零，与运行角标同语义）+ DeliverablesPanel 新 freshPaths prop（经 sidebarRegistry
  ctx 接线）→ 行「新」徽标（Sparkles）+ data-fresh 高亮；②**面板行级降噪**（Cowork
  一行式）：产物/变更/任务三列表次级信息（路径/相对路径/时间/重试计数）group-hover
  悬停次行显现（opacity 150ms），title 全保留，主行断言零改动；③**命令面板按当前视图
  重排**（Linear 式）：新 lib/paletteRank.ts 纯函数（当前激活右栏面板 cmd 置顶 >
  chatTab=overview 时概览置顶 > 其余面板命令 > 模板/会话保序，稳定排序），App paletteItems
  接线，CommandPalette 组件零改动；④**预览「半幅↔最大化」两档**（VS Code Toggle
  Maximized Panel）：icons 补 Maximize2/Minimize2（antd Fullscreen 系列），FilePreview
  头部按钮（不传 onToggleMaximize 不渲染向后兼容），App previewMaximized 状态（最大化=
  占满可用宽度 视口−侧栏−360 与拖拽上限同源，还原回半幅 ref 记忆，拖拽分割条自动退出
  最大化）。**验证**：Go build/vet 0 FAIL（零 Go 变更）；tsc/tsc -b/eslint 0；
  vitest **1157/1157**（+10：paletteRank 6+DeliverablesPanel 新徽标 2+FilePreview 两档 2，
  未删改旧锁）；drift PASS（552）；版本四处 4.30.0。**欠账**：产物自动弹 tab（激进版
  Auto-open）暂不做可加偏好；行级降噪仅悬停次行未做折叠重构；命令面板个性化排序远期；
  预览最大化不持久化；沿旧 v4.28 全部。详见 releases/v4.30.0.md。

- **上一发布：v4.29.0（2026-09-02）「办公 UI 化繁为简 · 顶栏收拢/自适应标签/
  预览降噪」**：git tag v4.29.0；基线 v4.28.0；绑定面 **552 零变更**（纯前端
  呈现重组）。用户点名主轴「UI 界面化繁为简，参考市场同类产品」，派刀即立
  红线 **「简化界面不是删除功能！」**。先跑模块制调研两线（原始稿
  docs/research-2026-09-01b/ + 合成 docs/market-research-2026-09-01b.md）：
  化繁为简=复杂度从「常驻视觉空间」迁「按需检索空间」+出现时机半自动化
  （Linear「只隐藏数据，不删除 issue」为官方先例）。三交付点（功能/出口全
  保留）：①顶栏导出收拢（新 ExportMenu：Markdown/Word/PDF 三文字钮→单钮
  「导出 ⌄」下拉，管线原样，操作钮 7→5）；②右栏 tab 窄栏自适应图标化
  （<420px 文字 CSS hidden 只显图标，aria-label/title/角标保留，6 tab 集合与
  数量锁不动，340px 基线宽拥挤根治；对标 Notion Icon only）；③预览头部降噪
  （FilePreview 打开/定位图标化+头部按钮去边框；编辑/保存/取消状态语义文字
  保留——编辑能力保留红线测试钉住）。**验证**：Go 110 包 0 FAIL；tsc/tsc -b/
  eslint 0；vitest **1147/1147**（+9：ExportMenu 5+WorkspaceTabs compact 3+
  FilePreview 1，未删改旧锁）；drift PASS（552）；版本四处 4.29.0。
  **欠账**：产物自动置前/角标；面板行级悬停次行化；palette 吸附右栏项+按视图
  重排；预览半幅↔最大化两档；沿旧 v4.28 全部。详见 releases/v4.29.0.md。

- **上一发布：v4.28.0（2026-09-01）「浏览器与版本 · 观察窗/版本时间线/pptx
  交互」**：git tag v4.28.0；基线 v4.27.4；绑定面 550→552（+GaeaPptxOutline/
  GaeaBrowserObserve）。规划「浏览器与版本」刀，三并行子代理分线（文件所有权
  互斥）+主代理集成：
  ①**A2 浏览器观察窗**（线 B）：browser.Manager.Observe()（CDP
  captureScreenshot jpeg ≤1280 缩放；未运行 Available=false 绝不拉起）+
  GaeaBrowserObserve 绑定（Default() 单例 seam 可测）；右栏第 6 tab
  BrowserPanel（URL/标题+截图 zoom+操作时间线 browser_* 倒序 20+权限静态行+
  自动弹出胶囊 gaea.browserAutoOpen；App 接线新 browser_* 工具自动切 tab；
  2.5s 可见门控轮询）。帧流/接管远期。
  ②**B1 版本时间线**（线 A，纯前端零 Go）：产物 vN 徽标可点→内联
  VersionTimeline（聚合/倒序/过滤无快照卡）+基线预览（GaeaPreview abs 直接
  预览 rollback 快照）+恢复（RollbackRecord=写回基线+追加新证据卡）；对标
  Notion 版本史/Artifacts rewind，预览即护栏。留白：单版本无入口、恢复不先
  快照当前态。
  ③**B2/C3 pptx**（线 C）：GaeaPptxOutline（python-pptx 逐页大纲，python
  3.13+python-pptx 1.0.2 真机可用）+ GaeaPreview .pptx 分支（soffice→PDF
  缓存 7 天 TTL+poppler 逐页 ≤60 页）+ 前端逐页预览+大纲侧栏+页锚点滚动+
  「针对第 N 页修改」指令插入；真机冒烟 3 页 deck 全通过。
  **集成**：gen_bindings 552+删线 B 临时手工 wrapper；bindingNames/bridge
  （PptxOutline camel+映射、GaeaBrowserObserve 同名直调）/spaceBindings
  work×2（分面锁 264）；App browser_* 自动弹出；补更组件级
  WorkspaceTabs.test 数量锁 5→6（线 B 漏项——**教训：新增 tab 要全查三层
  测试锁：lib 清单/注册表/组件按钮条**）。**验证**：Go 110 包 0 FAIL
  （stress flaky 沿旧）；tsc -b/eslint 0；vitest 1138/1138（+43）；drift
  PASS（552）；版本四处 4.28.0。**欠账**：A2 帧流/接管/动态权限卡；B1 单
  版本无入口+恢复不先快照当前；B2 真编辑远期；沿旧（子代理气泡恢复暂缺/
  中途进度不回投/WorkHeader 历史轮耗时/窄栏适配/tasks flaky）。**下一刀
  候选**：v4.29 从欠账池+调研剩余挑主轴。详见 releases/v4.28.0.md。

- **最新发布：v4.27.4（2026-09-01）「todo 持久化改名 · progress.md 撞名根治」**：
  git tag v4.27.4；基线 v4.27.3；绑定面 550 零变更。**勘误**：本文件四次被
  覆写并非「并行会话」（用户核实无其他会话）——真凶=gaea 自身 `todo_write`
  内置工具 V10.6 计划进度持久化：每次 todo_write 写 `<工作区根>/.gaea/
  progress.md`，办公代理以本仓库为工作区跑任务时逐次覆盖（四次内容全是任务
  todo 表，时间与任务节点吻合；backups/ 四份快照即 todo 表）。文件名撞车：
  ①宿主仓库项目记忆 ②代理运行时 todo 持久化，用 gaea 开发 gaea 必然相撞。
  **修复**：写入端改名 **todos.md**（todo.go saveProgressMarkdown）+
  compaction 读取端 readProgressFile 优先 todos.md/回退旧名（compact_util.go，
  存量工作区兼容）。测试 +2（TestSaveProgressMarkdownWritesTodosNotProgress
  =事故直接回归锁；TestReadProgressFilePrefersTodosAndFallsBack——walk-up
  设计使「均缺失」断言在真实机器不成立，顺带实锤主目录
  C:\Users\wubi\.gaea\progress.md 有 8 月底代理 todo 残留可清理）。Go 110 包
  0 FAIL、前端零改动。**教训**：运行时产物文件名不得与宿主仓库约定文件同名
  ——自举项目工作区常驻本仓库，任何 <cwd> 相对写入都要过撞名审查；归因要
  先证伪（「并行会话」结论未核实就写进了三份发布说明）。详见
  releases/v4.27.4.md。

- **最新发布：v4.27.3（2026-09-01）「markdown 包裹符 · 交付卡片路径修复」**：
  git tag v4.27.3；基线 v4.27.2；绑定面 550 零变更。用户报告「交付卡片点击
  无法打开、定位打开的不是文件位置」→ computer-use 真实会话现场实锤：模型
  用反引号包裹路径（\`安全文明手册/….docx），卡片提取路径带开头反引号 →
  预览「文件不存在」、定位错位。根因=fileLinks 路径字符集不排除 markdown
  包裹符 \` 与 *（Windows 文件名非法字符，应作路径边界；v4.26.1 全角括号
  盲区第二弹）。修复=PATH_BODY/FIRST_SEG 排除+PATH_BOUNDARY 纳入+BARE_FILE_RE
  允许包裹符前缀；存量消息渲染时实时重提取、重启即恢复。+5 测试。
  vitest 1095/1095、tsc -b/eslint 0。**教训**：路径正则字符集以「目标平台
  文件名合法字符集」为准——非法字符（Windows: \ / : * ? " < > |）必然是
  markdown/包裹符，作边界不作路径体。**progress.md 第四次被并行会话覆写
  （backups/ 留档），再次恢复。**详见 releases/v4.27.3.md。


## 当前状态

- **最新发布：v4.27.2（2026-09-01）「细节收口」**：git tag v4.27.2；基线
  v4.27.1；绑定面 550 零变更。①**subagent_message 端到端收口**（v4.26 回投
  特性此前实际未通：后端发 kind=subagent_message、前端无消费整条被丢）——
  wire 层转译 kind="message"+subagentRef（gaeaEventMap，磁盘日志仍按原始
  kind 落）→ 前端 reducer message case 既有 subagentRef 语义接管（「子代理」
  徽标气泡）；补拉折叠同步：GaeaResyncItem 加 subagentRef（恒全键契约测试
  同步扩键）、fold subagent_message → 独立 assistant 条目 + closePending
  （其后 text 不误续写）；恢复会话场景欠账（模型上下文投影 ProjectMessages
  未含，避免改变续跑模型语义——记远期）。②**轨迹面板子代理记录**（子代理
  线交付）：TrajectoryRecordKind 加 "subagent"，KindBadge/Bot 图标/折叠行
  （答复摘要+ref）/RecordInspector 全文展开/搜索命中，turns 与 betweenTurns
  双落点覆盖。③**sidebar_open 目录定位**（收 v4.25 欠账）：directory 分支
  → handleRevealInTree，FileTree 目录行补 data-path/flash（验证发现目录行
  原本不被 reveal 命中——父链能展开但无锚点无高亮）。**验证**：Go 110 包
  0 FAIL（TestCancelConcurrentStress 全量负载下偶发 flaky，单跑稳定，与本
  刀无关）；tsc -b/eslint 0；vitest 1090/1090（+5）；drift PASS（550）。
  **教训：FileTree reveal 只锚 data-path——新增行型（目录）必须同步锚点，
  否则定位静默失效。progress.md 第三次被并行会话覆写（backups/ 留档第 3
  份），再次从 git 恢复。**详见 releases/v4.27.2.md。


## 当前状态

- **最新发布：v4.27.1（2026-09-01）「seq 防线 omitempty 失配修复」热修**：
  git tag v4.27.1；基线 v4.27.0；绑定面 550 零变更。用户报告「运行中只显示
  一个思考读秒，没有交替出现过程卡/文本卡（只有轨迹面板有显示）」。根因=
  v4.26 seq 补拉防线前后端形状契约失配：Go GaeaResyncItem 全字段 omitempty，
  流式 assistant 条目空 reasoning、写类工具 readOnly:false 的键被序列化省略；
  前端 parseResyncItems 严格校验缺键即整快照判坏 → 补拉快照 100% 被拒、
  防线静默失效，Wails 吞件期间对话窗无物可渲染（WorkHeader 是 store tick
  驱动所以活着；轨迹面板读盘不受害）——正是用户看到的形态。修复：①Go 全
  字段去 omitempty（序列化恒全键，TestGaeaResyncItemWireAllKeys 锁契约）；
  ②前端 parseResyncItems 缺省键宽容（缺字符串/布尔键→零值，类型错仍拒、
  kind/id/status 枚举不变）。**真机验证（computer-use 驱动真实应用发只读
  任务）**：对话窗内 WorkHeader 完成态「已完成 · 用时 15s · 7 步」+ 阶段行
  （正在解析@引用/装配首轮上下文/检索记忆）+ 思考块 + ls 工具卡（参数+输出）
  + 两段正文交替出现（各卡 elapsed 3s→8s→14s 证明运行中逐个渲染）；v4.26.1
  的交付文件卡片同屏确认。vitest 1085/1085、Go 110 包 0 FAIL、tsc -b/eslint 0。
  **教训：跨语言 wire 契约（Go struct json tag ↔ 前端严格解析）必须有一条
  「真实序列化形态」往返测试；omitempty 是严格校验的天敌。注意：.gaea/
  progress.md 被并行会话再次覆写并随 v4.27.0 提交，已再次从 git 恢复。**
  详见 releases/v4.27.1.md。


## 当前状态

- **最新发布：v4.26.1（2026-09-01）「全角括号文件名 · 交付卡片失配修复」**：
  git tag v4.26.1；基线 v4.26.0；绑定面 550 零变更。用户报告「看不见完工交付
  卡片、无法点击查看文件」。定位过程：静态审计 v4.26 全部渲染/事件路径均
  无回归 → 主代理用 computer-use 观察运行中的真实会话，a11y 树实锤：正文有
  「交付文件：C:\AI\bangong\黄甲\开工筹备计划（修订）.docx」但卡片未渲染 →
  根因=lib/fileLinks.ts PATH_BODY 把全角括号（）当路径终止符，文件名含（）
  时两条匹配（绝对路径+关键词裸名）全落空。修复=PATH_BODY 排除集移除全角（）
  （扩展名锚定末尾不吞补语）；GaeaPreview 的 resolvePreviewPath 本就接受绝对
  路径，点击链路零改动。测试：fileLinks +5（真实括号路径四形态+deliverableMentions
  数据源）+ DeliverableCards.regress.test.tsx 组件守卫 3（真实事件流渲染/resync
  替换后仍可见）。vitest 1080/1080（170 文件）、tsc -b/eslint 0。**教训**：
  用户报告「与版本相关」的 bug 未必是版本回归——先用运行中的真实数据实证
  形态再定修法；中文办公文件名带（）是常态，路径类正则必须过全角括号用例。
  详见 releases/v4.26.1.md。**注意：并行会话正在做 v4.27 右栏加宽
  （App/WorkspacePanel/workspaceTabs 未提交改动在工作区），提交时严禁 git add -A。**

- **最新发布：v4.26.0（2026-09-01）「对话流式重造 · 对齐 Codex」（插刀）**：
  git tag v4.26.0；基线 v4.25.0；绑定面 549→550（+1 GaeaResyncEvents）。
  起点=用户报告「办公板块对话发送后窗口静默半天，轨迹面板却在工作」。主代理
  事件流逐环节探查+调研子代理（Codex/Claude Code/Cursor 流式工作态，落
  docs/research-2026-09-01/codex-streaming-ux.md）→ 根因六连（子代理 Text/
  Reasoning 有意不进主聊天、预处理窗零事件、Wails 吞件、Retrying 未映射、
  phase 空 seam、TTFT 静默）→ 三并行编码子代理线（Go 事件线/前端状态线/
  渲染线，文件所有权互斥）+ 主代理集成：
  ①**WorkHeader 工作态头部行**：turn 激活期常驻（spinner+阶段文本+已用时
  1s tick+步数，items 为空也渲染=发送那一帧起窗口不空），完成转「已完成 ·
  用时 · N 步」耗时行；StreamingIndicator 收敛兜底（连接中/仍在等待事件）。
  ②**后端 phase 事件接线**：预处理各阶段发射 phase（正在启动引擎/解析 @引用/
  装配首轮上下文/检索记忆/思考中）+ Retrying/compaction 转译 phase（磁盘日志
  格式不变，200ms 同文案节流）；phase 收编过程卡+头部防重复。
  ③**子代理活动回投主回合**（Codex 2026-08 同款）：新事件 subagent_message
  完成态回投子代理最终答复（ref/parentId），主区消息「子代理」徽标；task 卡
  running 实时 lastText/lastTool 预览（App 5s 轮询注入 taskActivity）+完成
  结果摘要；中途进度不回投防刷屏（实时进度走分工面板）。
  ④**事件序号防线**：gaea-event 全量带 seq（转发层原子递增，会话切换归零），
  跳号→新绑定 GaeaResyncEvents 从磁盘日志折叠对话项全量快照整体替换（5s 冷却/
  在途去重/坏快照保底/streaming 续接/running 不动）；golden 逐字节不变。
  ⑤重复工具折叠「已调用 X · N 次」（Claude Code 式）；顺带修
  weixin_reminder_test 时间炸弹（固定基准过期必炸→取真实时钟）。
  **集成**：绑定面四处（bindingNames +GaeaResyncEvents/bridge AppBindings.
  ResyncEvents+映射/spaceBindings work 分面锁 262/types GaeaResyncResult）+
  App.tsx（fetcher 挂载+task 卡轮询注入）。
  **验证**：Go build/vet/test 全量 0 FAIL；tsc -b/eslint 0；vitest 1072/1072
  （169 文件，净增 71：eventSync 26、store.resync 18、taskActivity 12、
  WorkHeader 8 等）；drift PASS（550）；版本四处 4.26.0。**教训**：根因探查
  先于动手（六个根因里三个半在后端，纯前端重造治不了）；「吞件」类问题用
  序号+读盘补拉根治而非重试；并行会话会覆写 .gaea/progress.md（本次被无关
  会话清掉 851 行，已从 git 恢复+对方内容备份 backups/）。
  **欠账**：子代理中途进度不回投；并行多子代理派发瞬间 task 卡预览可能短暂
  空 ref；历史轮无耗时数据源；TrajectoryView 未消费 kind="subagent" 记录。
  **下一刀 v4.27「浏览器与版本」**（原 v4.26 顺延）：A2 观察窗+B1 版本时间线
  +B2 pptx+C2/C3（调研弹药 docs/market-research-2026-09-01.md）。详见
  releases/v4.26.0.md。


## 当前状态

- **最新发布：v4.25.0（2026-09-01）「文件工作台 · 编辑器 tab 化/变更 diff/
  选区联动/模型主动打开」**：git tag v4.25.0；基线 v4.24.0；绑定面 549→549
  （零新增）。规划第三刀（docs/gaea-office-upgrade-plan-2026-09.md A3+B3）。
  本轮全程 3 并行编码子代理分线（文件所有权互斥）+ 3 调研子代理同期跑 +
  主代理集成收口：
  ①**A3 编辑器 tab 化（EditorTabs）**：文件树点开→右栏内多文件编辑器 tab
  （lib/editorTabs zustand 外部 store：上限 12 LRU/关闭激活相邻/localStorage
  坏值兜底/openEditorTab 命令式入口）；FilePreview embedded 模式（默认 false
  行为不变），docx 框选即改/xlsx 直编+Plan→Apply 原样随迁（换壳不换芯红线）；
  双入口保留（树行点击=右栏内 tab、右键「预览」=主区 pane）；产物行「树中
  定位」reveal→FileTree 展开父链+滚动+1.6s 闪烁（注册表 ctx 增
  revealRequest/onRevealInTree 两字段透传，面板本体零改动）。
  ②**A3 变更 tab diff 化**：文件行展开→行级红绿 diff（lib/planDiff 数据构造 +
  ChangesDiff 三态渲染；数据源诚实评估：edit_file/multi_edit 有 old/new→真
  diff，write_file/edit_lines 仅新内容→写入预览+原因，其余不伪造）+ 回滚接
  证据链 Journal 最近基线（GaeaJournalList+RollbackRecord，路径匹配排除
  rollback 记录）。
  ③**B3 选区联动**：xlsx 选中单元格→浮动「引用到对话」（双击直编时隐藏）；
  docx 框选工具栏补「引用到对话」；docx 渲染失败降级纯文本（lib/docxText
  提取 word/document.xml 正文段落+amber 提示条，能力边界如实）。
  ④**模型主动打开 sidebar_open**：Go 内置工具（work 空间/ReadOnly 直允许/
  防穿越 realPath+within/envelope data {kind,path_abs,path_rel}；+20 Go 用例）
  + lib/sidebarOpen.ts 解析器（坏 JSON/失败 code→null 不抛）+ App 按工具事件
  id 去重消费（file→openEditorTab 开右栏编辑器 tab，directory→亮文件 tab；
  命中自动亮右栏切「文件」tab，先收主区预览）。
  **验证**：Go build/vet/test 全量 0 FAIL；tsc -b/eslint 0；vitest 1001/1001
  （164 文件，净增 74：editorTabs 16、sidebarOpen 8、planDiff 12、ChangesDiff 4、
  DocxPreview 6、EditorTabs/WorkspacePanel/ChangesPanel 重写等）；drift PASS
  （549）；版本四处 4.25.0。**同期调研**：docs/market-research-2026-09-01.md
  合成版 + docs/research-2026-09-01/ 3 原始稿（v4.24.0 基线：浏览器观察窗/
  版本时间线/pptx 与工作台动向，喂 v4.26）。**教训**：并行子代理跨线只共享
  「接口契约（prop 名/签名）」并在派发时逐字写进两份 brief，中途再以消息补发
  约定（本轮 editorTabs 外部 store 补约定避免返工）；子代理自查 tsc 可能撞见
  并行线半成品错误，收口以主代理全量门禁为准。**欠账**：变更 diff 的
  write_file/edit_lines 旧内容缺失（待 B1 写前快照库）；回滚粒度=最近基线
  （逐版本回滚待 B1）；docx 降级仅正文段落；sidebar_open directory 不定位
  树根；EditorTabs 窄栏精细适配；AgentNetworkCard 旧卡不动（沿 v4.24 约定）。
  **下一刀 v4.26「浏览器与版本」**：A2 观察窗（截图步进流起版+操作时间线+
  权限卡内联）+ B1 版本时间线（写前内容寻址快照库与登记表同源+vN 徽标
  popover 双入口+恢复=新增版本）+ B2 pptx（结构化大纲卡先行+页级指令两通道）
  + C2/C3。详见 releases/v4.25.0.md。

- **最新发布：v4.24.0（2026-09-01）「子代理工作台 · 树拓扑/实时动态/
  产物登记表」**：git tag v4.24.0；基线 v4.23.0；绑定面 548→549（+1：
  GaeaDeliverableRegistry）。规划第二刀（docs/gaea-office-upgrade-plan-2026-09.md
  A1+C1）。上一轮会话完成主体编码，本轮接续完善收口：
  ①**A1 树形实时拓扑（AgentTree）**：GaeaAgentNetwork 嵌套 Children 全量渲染
  （此前只画两层）；root 折叠为「主 agent」行、一级恒可见、更深默认收起可
  展开/收起；新节点出现自动展开父链（首轮只记基线）；节点量化——状态色点/
  任务摘要/工具数/模型徽标/耗时（running 实时已用 1s tick）/token/错误数；
  下钻链：节点→详情卡→完整 transcript→**工具调用行点击定位结果消息**（收
  v4.21 欠账，data-located 高亮）。
  ②**合并活动流（Devin 式单列 feed）**：running 子代理 lastText/lastTool 按
  updatedAt 倒序合并、上限 20、空态收起；与树内行预览并存。
  ③**新子代理自动展开（可关，默认开）**：新 ref 出现→onSubagentStarted 回调
  →App 亮出右栏切「分工」tab（tab 停用时尊重停用态）；偏好键
  gaea.subagentAutoOpen（localStorage，损坏值回落默认开——交付前修 bug：
  原实现把垃圾值当关闭）。
  ④**C1 权威产物登记表**：trajectory.FoldDeliverables 纯函数从事件日志折叠
  写类 8+生成导出类 3 工具的落盘登记（路径/最近工具/来源轮次/时间/累计次数，
  按 path 去重、updatedAt 倒序、上限 200，Total 完整去重数）；登记口径
  evidence.IsDeliverableTool/ExtractDeliverablePaths（与证据链 extractPaths/
  前端 changes.ts 同源对齐，不收 source；bash/screen_capture 无结构化路径
  参数诚实不猜）；新绑定 GaeaDeliverableRegistry(sessionPath)（防穿越；
  无事件日志 Available=false 不报错）；DeliverablesPanel「权威产物登记」只读
  区（tool 徽标+路径+轮次+次数+时间，点击预览；total>200 提示最近 N 条），
  补启发式（正文扩展名白名单）漏登。
  **验证**：tsc/tsc -b/eslint 0；vitest 927/927（157 文件，净增 16：
  SubagentsPanel 重写 8、AgentTree 7、DeliverablesPanel 登记表 3、subagentPrefs
  3 等；旧扁平卡用例随新结构重写）；Go build/vet/test 全量 0 FAIL；drift PASS
  （549）；版本六处统一 4.24.0。**教训**：交付前必须跑 `tsc -b`（build.bat
  同配置）且检查新测试文件路径参数（vitest run 无参数时 tsconfig 的 include
  范围与手测不同）；vi.mock 全模块 mock 时注意副作用导入（onEvent 订阅器）。
  **欠账**：AgentNetworkCard（主区上下文 tab 旧卡）仍两层 SVG 本轮按约定不动；
  登记表为独立只读区未与启发式合并去重（v4.25 A3 变更 tab 统一快照时处理）；
  下一刀 v4.25「文件工作台」（A3 编辑器 tab 化+变更 diff+模型主动打开+reveal
  +B3 选区联动）；v4.26 浏览器观察窗+版本时间线+pptx（A2/B1/B2/C2/C3）。
  详见 releases/v4.24.0.md。

- **最新发布：v4.23.0（2026-08-31）「工作台框架 · 右栏工作台化第一刀」**：
  git tag v4.23.0；基线 v4.22.0；零新增绑定（548）。用户拍板：右面板重造为
  DSH-better-sidebar/Codex 式「运行工作台」（子代理/浏览器/文件编辑器等实时
  操作面），状态显示类迁主区轨迹/上下文旁边；规划稿
  docs/gaea-office-upgrade-plan-2026-09.md（v2，分期号顺延）。
  ①**Tab 注册表 lib/sidebarRegistry.ts**：元数据复用 workspaceTabs 清单 +
  render 接线单一数据源，右栏渲染/命令面板全派生；新增面板 = 清单 + RENDERERS
  各一条、面板组件本体零改动（框架/内容解耦，为 v4.25/4.26 浏览器观察窗与
  编辑器 tab 留平等挂载点）。
  ②**工作台外壳三件套（学 better-sidebar v0.18 交互形状，不抄代码）**：全局
  宽度键（左缘拖拽 280–720，最后一次拖拽胜出跨会话跟随）；声明式设置（齿轮
  →侧边卡片，每 tab 独立开关，停用即隐藏/整组停用隐藏主 Tab/至少保留一个/
  停用不进命令面板，启用集全局键 gaea.rightPanel.v1:tabsEnabled）；会话记录
  v2（{v,tab,enabled,width}，v1 裸 id 兼容可读，坏值逐项兜底，失效激活指针
  修正）。
  ③**主区「概览」tab（A4 统计迁移）**：ChatTabs 第 4 tab（对话/轨迹/上下文/
  概览）+ OverviewPanel 承载原 StatsPanel（本体零改动）；右栏统计下线
  （WorkspaceTabId union 全量移除，v4.22 旧 tab:"stats" 宽容收敛回「文件」
  并钉回归用例），右栏收敛 3 主 Tab×7 面板；命令面板新增「概览面板」项；
  chatTab 恢复白名单加 overview。
  **实现方式**：两个并行子代理分线（框架线/概览线，文件所有权互斥）+ 主代理
  集成收口。**教训**：tsc --noEmit 与 build.bat 的 tsc -b 配置不同——NodeList
  for...of 迭代错只在 tsc -b 暴露（已改 forEach），新测试代码过门前必须跑
  `npx tsc -b`。
  **验证**：tsc/tsc -b/eslint 0；vitest 911/911（净增 38）；Go build/vet/test
  0 FAIL；drift PASS（548）；版本四处 4.23.0；build.bat 构建成功 + 冒烟
  /api/health 200。**欠账**：Tab 拆分分栏/底部面板/自由窗口、设置二级齿轮
  弹窗、注册表懒加载 chunk 化；下一刀 v4.24「子代理工作台」（树形实时拓扑 +
  live 批量预览 + 节点耗时/进度 + transcript 工具级下钻 + 合并活动流 + 产物
  登记表）。详见 releases/v4.23.0.md。

- **v4.22.0（2026-08-31）「一次性收官 · 真虚拟化/transcript 定位/
  晨报预载 UI」**：用户要求「一次性做完」——办公板块剩余本地可做欠账一次
  清完并整理提交收尾，三件事一并落地：
  ①**轨迹真虚拟化（react-window v2 动态行高）**：扁平行流按视口窗口渲染
  （±overscan 12），超长会话 DOM 恒定，v4.21「首批 250 + 加载更多」分批机制
  退役；useDynamicRowHeight + ResizeObserver 实测展开行高自动重排（jsdom
  回落 defaultRowHeight 29）；概览跳转走 listRef.scrollToRow、搜索词变化
  回顶、收起全部/展开全部在虚拟流上照常生效；test/setup 补 ResizeObserver
  stub（react-window 内部无守卫使用）。
  ②**transcript 消息定位**：每条消息按原位置带序号 #N；搜索命中自动滚动到
  第一条命中（scrollIntoView），计数「命中/总数」。
  ③**晨报预载 UI 开关（+2 绑定 GaeaMorningPreload/GaeaSetMorningPreload）**：
  internal/config.Save 持久化 ~/.gaea_config.json + 内存更新 + 重建引擎即时
  生效；记忆面板头部「晨报预载 开/关」胶囊按钮（与记忆开关同款交互）；
  gen_bindings 重生成 548，bindingNames/mock/bridge/spaceBindings 同步。
  **验证**：Go 全量 0 FAIL（绑定面完整性 548）；tsc/eslint 0；vitest 873/873
  （+3：虚拟化 DOM 有界与滚动到末尾、transcript 序号、晨报预载开关）；
  drift PASS（548）；版本四处 4.22.0。**收尾**：v4.17.0-v4.22.0 六轮改动
  交织在同一工作区，作为一次合并发布提交并打 tag v4.22.0（历史 release
  notes 保留演进记录）。剩余欠账仅外部资源/官方数据项（Realtime 真机、
  自动路由、浏览器下载上传/headless UI/Windows UIA、iLink 真机窗口）——本地
  不可完成，不在「做完」范围。详见 releases/v4.22.0.md。

- **最新发布：v4.21.0（2026-08-31）「长会话与 transcript · 增量渲染/消息搜索」**：
  续 v4.20.0 剩余两条欠账（轨迹超长会话渲染量 + transcript 只读无搜索），
  纯前端零新增绑定：
  ①**轨迹增量渲染（DOM 有界）**：渲染从「逐轮整体」改为扁平行流（轮次头 +
  展开记录行 + Between-turns），按批渲染——首批 250 行，滚动到底自动续载
  或点「加载更多（剩余 N 条）」，搜索词变化回首批；概览跳转同步把目标轮
  之后的可见区扩到视口内（不再「跳过去了但没渲染」）；收起全部/展开全部在
  平行流上照常生效（折叠 = 记录行不进流）。
  ②**子代理 transcript 消息搜索**：查看器头部搜索框，按正文/推理/工具名/
  参数/结果过滤，计数「命中/总数」，无匹配空态。
  ③**注释清理**：ChatTabs「[轨迹]（暂占位）」更新为 v4.17-v4.21 实际能力。
  **验证**：tsc/eslint 0；vitest 872/872（+2：超长会话增量渲染与加载更多、
  transcript 搜索）；Go cached 全绿、drift PASS（546）；版本四处 4.21.0。
  欠账：增量渲染为分批 DOM 而非 react-window 真虚拟化（渲染量有界但已渲染
  部分仍为真实 DOM）；transcript 只读无跳转/引用定位。详见 releases/v4.21.0.md。

- **最新发布：v4.20.0（2026-08-31）「剩余收官 · 子代理 transcript/轨迹概览/
  旧会话趋势补齐」**：清掉 v4.17-v4.19 三刀之后的剩余欠账，三件事一并落地：
  ①**子代理完整 transcript 查看器（+1 绑定 GaeaSubagentTranscript）**：读取
  `<sessionDir>/subagents/<ref>.jsonl` 全量消息（role/content/reasoning/
  toolCalls/toolCallId），ref 校验 sa_ 前缀+安全字符（防穿越），读取失败返回
  错误（区分「没有」与「读不了」）；gen_bindings 重生成（546），bindingNames
  同步；前端 Agent 网络详情面板「查看完整 transcript」→ 消息流可滚动可收起。
  ②**轨迹 Overview 投影 + 轮次跳转 + 折叠控制**：轨迹标签顶部概览条（每轮
  一根柱，柱高∝记录密度，含工具调用高亮、报错标红，hover 明细，点击平滑
  跳转并展开目标轮）+「收起全部/展开全部」（长会话折叠成轮次索引，新回合
  默认展开不被旧折叠态吞掉）。
  ③**迁移/兜底会话趋势补齐（诚实估算）**：ToLogEntries 每回合合成
  request_header（system=真实 system 消息拼接，tools=该轮 assistant 实际
  工具名集合，schema 未知的最小诚实形状；顺序与运行期一致——user 先落、
  header 随后）；contextview 新增回合末估算关闭（turn_done 未见 usage 时用
  当前估算构成落 estimated 记录并刷新 brief），前端步骤详情显示「估算构成
  （无用量记录）」，不伪造 promptTokens 等用量数字。
  **验证**：Go 全量 0 FAIL（+2 用例）；tsc/eslint 0；vitest 870/870（+3）；
  drift PASS（546）；版本四处 4.20.0。欠账：轨迹虚拟滚动未做（以收起全部+
  概览跳转缓解渲染量）；子代理 transcript 只读无搜索。详见
  releases/v4.20.0.md。

- **最新发布：v4.19.0（2026-08-31）「看板收官 · 上下文浏览器//context 命令/
  子代理节点详情」**：续 v4.17.0+v4.18.0 的「继续完善」第三刀，上下文标签
  最后一个页脚占位收掉，三件事互不相交一并落地：
  ①**上下文浏览器**：contextview 折叠补全系统/工具节点（request_header 的
  system prompt 与工具集合只在构成变化时入 nodes——初版+变化版，每步重复
  不刷屏；文本=300 字预览，全文在日志）；前端 ContextBrowserCard（活跃/归档
  双页签 + 六分类过滤 chips + 节点行可展开，归档=被压缩移出带「已压缩」），
  页脚占位整行移除。
  ②**/context 命令**：GaeaCommands 内置 + i18n（zh/en CmdContext），斜杠
  菜单可发现；classifyComposerCommand 增 context 分类，App.handleSend 拦截
  → setChatTab("context")（不发给模型）；CLI 未拦截路径走控制器未知斜杠
  Notice。
  ③**Agent 网络节点点击 → 子代理详情**：AgentNetworkCard 增 sessionPath
  （App 从 currentSessionPath 注入），点击子代理节点 → SubagentRuns(sessionPath)
  按任务前缀匹配（与后端 enrichAgentNetwork 同口径）→ 固定详情面板（状态/
  模型/工具调用数/更新时间 + lastText/lastTool + 最后回答摘要），无匹配回退
  节点统计；悬停文案更新。
  **验证**：Go 全量 0 FAIL（+1 用例）；tsc/eslint 0；vitest 867/867（+4）；
  drift PASS（545）；版本四处 4.19.0。欠账：子代理完整 transcript 查看器；
  轨迹 Overview 投影与虚拟滚动；迁移会话系统/工具分类与趋势柱（诚实不造
  数）。详见 releases/v4.19.0.md。

- **最新发布：v4.18.0（2026-08-31）「看板补全 · 文件活动/增量模式/实时刷新」**：
  续 v4.17.0（事件日志默认开启，轨迹/上下文数据源接通）后的「继续完善」，
  收掉两个看板剩余的占位尾巴，三件事互不相交一并落地：
  ①**文件活动时间线**（上下文标签新卡）：contextview 折叠新增 FileActivity——
  工具参数确定性提取路径（path/rel/source/destination/image_path/output 键）
  + 工具→动作白名单（read/grep/vision/format_convert=读；write/edit/multi_edit/
  edit_lines/chart_gen/diagram_gen/screen_capture=写；move=移；ls=目录），
  screen_capture 从结果输出补记，bash 等无法确定性取路径者诚实不造数；同轮
  同步骤同路径合并、上限 200、空切片非 nil。前端文件活动卡（动作徽标+工具+
  路径+时间，倒序最近 40 条），页脚改为「上下文浏览器将在后续阶段接入」。
  ②**增量（Delta）模式启用**：趋势图「增量」按钮去灰置（Phase B 占位收口），
  切换展示每步相对上一步净变化（绿=净增·红=净减图例），柱色改全站语义色。
  ③**运行中实时刷新**：新 hook useLiveReload 订阅 gaea 事件流——运行中节流
  刷新（1200ms）+ turn_done 立即刷新 + 整轮完成刷新；轨迹/上下文/Agent 网络
  三处统一接入。**验证**：Go 全量 0 FAIL（+2 用例）；tsc/eslint 0；看板+
  mock-contract vitest 22/22（+2）；drift PASS（545）；版本四处 4.18.0。欠账：
  上下文浏览器（surface 节点浏览/归档）仍占位；Agent 节点点击跳子代理会话；
  轨迹 Overview 投影与虚拟滚动；/context 命令；迁移会话系统/工具分类与趋势
  柱（旧消息无 request_header/usage，诚实不造数）。详见 releases/v4.18.0.md。

- **最新发布：v4.17.0（2026-08-31）「轨迹上下文接通 · 事件日志默认开启」**：
  用户反馈办公板块「轨迹」「上下文」标签是空壳。排查结论：前端/绑定/折叠器
  均完整，根因在数据源——两看板依赖追加式事件日志，而 `session.log_format`
  缺省 legacy = sink 不接线 → 日志从未落盘 → 看板恒读空日志。交付四块：
  ①**事件日志缺省开启**：config.EffectiveLogFormat 缺省 "event"（显式 legacy
  可退回），gaea_handler 注入生效值、boot 同源创建 EventLogSink；②**旧会话读端
  兜底**：session.ReadEntriesFor 优先事件日志、缺失时从 legacy 会话投影折叠
  条目（纯读不落盘），轨迹/上下文/Agent 网络三看板共用；③**迁移产物带回合
  边界**：ToLogEntries 每条 user 消息前写 turn_started、流尾写 turn_done
  （ProjectMessages 忽略边界，恢复投影逐字节不变）+ 轨迹折叠兼容
  assistant_message（内嵌工具调用展开为 tool 记录并与结果合并）；④**资源
  释放**：EventLogSink.Close 挂进 Controller.Cleanup——缺省 event 后 Windows
  上会话目录可删除/迁移（文件句柄泄漏面一并修掉）。**验证**：Go 全量 0 FAIL
  （+6 用例）；tsc -b 0；看板组件 vitest 12/12；drift PASS（545）；版本四处
  4.17.0。欠账：迁移/兜底会话无 request_header/usage（系统/工具分类 0、趋势
  无柱，新会话完整）；Agent 网络对 legacy 会话仅 root；上下文增量模式/浏览器/
  File activity/SSE 增量刷新 = v3.5 既定欠账。详见 releases/v4.17.0.md。

- **最新发布：v4.16.0（2026-08-31）「四刀并行 · 离线收口/浏览器键盘与 iframe/复核
  可视化/晨报预装配」**：用户拍板「全部并行处理」——v4.15.0 欠账四个可离线方向由
  四个并行子代理同步落地（足迹隔离：刀①/②/④零绑定零前端，刀③独占绑定面相关
  文件），主控全绿门禁。**零新增绑定（545 不变）**。①**persona 侧离线裂缝收口**
  （真 bug）：gaea_whisper_causal/retell/whisper_handler 三处 featureModel("chat")
  → routeModel("chat")——全局离线过滤对轻语链路生效（此前绑云端照样发云端），
  用户功能绑定语义不变（同源）+2 离线回归。②**浏览器键盘级 Input + iframe**
  （v4.13/14 欠账）：新工具 browser_press（第 11 工具，Input.dispatchKeyEvent 键盘
  级输入：key 别名表/组合键/text 真实输入，Enter 补 \r 触发 keypress 真机踩坑
  修复）+ browser_read/click/type 加 frame 参数（getFrameTree→createIsolatedWorld
  →contextId，**iframe 内交互完整实现**，真 headless Edge 真机验证 Read/Click/Type
  全通）；snapshot 不下钻 iframe 诚实拒。③**Verifier 通道 B 结果进前端**（v4.14
  欠账）：Verdict 增 channelBRatio/channelBPages/channelBArtifacts（omitempty 旧卡
  兼容）+ 证据卡「视觉复核：像素差异率 x.x% · N 页」行 +「查看复核产物」按钮
  （打开产物目录 before/after PDF + 逐页 PNG）。④**晨报深度预装配**（v4.14 欠账）：
  memory.BuildMorningPreloadBlock 纯函数（复用 BuildMorningBrief 排序口径，≤600
  rune 确定性零 LLM）→ sysprompt 装配点注入「【工作记忆晨报】」块（门控
  Memory.Enabled && morning_preload && space==work，play/mode=off 不注入=双空间
  红线）；config 键 morning_preload（默认 true，仅配置文件可控）。**验证**：Go
  全量 0 FAIL（+20）；vitest **861/861**（+2）；tsc/eslint 0/0；drift PASS（545）；
  build.bat 冒烟 200；版本四处 4.16.0。欠账：Realtime 真机（需用户真 key+麦克风）；
  自动路由本体（待官方逐模型缓存/峰谷数字）；浏览器 snapshot 不下钻 iframe/
  下载上传/headless UI/Windows UIA；通道 B 逐页缩略图；晨报预载无 UI 开关。详见
  releases/v4.16.0.md。

- **最新发布：v4.15.0（2026-08-31）「聊天路由归位 · 由谁回答」**：v4.14.0 欠账
  「自动路由 v1」经**用户拍板收缩**为最小价值刀——砍成本档位机制/auto_route 开关/
  UI（缓存价/峰谷价无官方逐模型数字，按「未核实不入表」纪律诚实不做），只留两块
  真实价值。①**聊天路由归位（真 bug 修复）**：chat_service.go:68/:105 +
  chat_handler.go:9 三处 `featureModel("chat")` → `routeModel("chat")`——用户功能
  绑定语义逐字节不变（同源），新增收益=全局离线模式对 plain 聊天生效（修复「总闸
  不总」裂缝）+ 无绑定时全局/兜底与 persona 一致 + model.route 事件补齐。②**「由谁
  回答/为何/花了多少」回显**：modelengine 导出 EstimateCostCNY（本地/未知恒 0、
  USD 折算 CNY、非法汇率回退 7.2）；chat done 帧/ChatSend 返回加
  answered_by{engine,model,source,cost_cny}（流式按 chunk.Usage 实算，usage 不可达
  诚实记 0）；前端 AnsweredByLine 消息底部小字（费用 ≤0 隐藏段）+ useChatStream
  解析（旧事件静默跳过向后兼容）。**验证**：Go 全量 0 FAIL（+7）；vitest **859/859**
  （+7）；tsc/eslint 0/0；drift PASS（545 不变）；build.bat 冒烟 200；版本四处 4.15.0。
  欠账：自动路由本体未做（待官方逐模型缓存/峰谷数字后另刀）；persona 侧
  gaea_whisper_causal/retell 同类离线裂缝=观察项；按 source 拆分统计未做；plain
  费用口径 usage 不可达恒 0（诚实降级）。详见 releases/v4.15.0.md。

- **最新发布：v4.14.0（2026-08-31）「三箭并行 · 晨报预取 + 浏览器续刀 + 复核产品化」**：
  用户拍板「多刀并行」，三刀足迹隔离并行落地（探索子代理 ×3 代码地图 → 实现
  子代理 ×3 → 主控全绿门禁），绑定面 544→**545**（+1：GaeaMemoryMorningBrief）。
  ①**浏览器续刀**（v4.13.0 欠账）：空闲 TTL 自动关停（Options.IdleTTL 默认 10min
  + GAEA_BROWSER_IDLE_TTL env，Ensure 成功路径刷 lastActive，once 守护 watcher
  到期 teardownLocked 自动回收，browser_* 自动重拉闭环）+ 多标签页（Manager
  重构 tabs map + activePageID，/json/list 全量 target 真源；ListTabs/NewTab/
  SwitchTab/CloseTab，切 tab 置 epoch=0 旧 refs 诚实失效，关 active 切剩余、
  最后整体回收）+ 新工具 ×3（browser_tabs/browser_new_tab/browser_switch_tab）
  + browser_close 可选 tab_id（缺省保持现语义）；零新增绑定、前端零改动。②**做梦
  2.0 主动预取 MVP**（路线图 T0 欠账）：memory.BuildMorningBrief 纯函数（零 LLM/
  零 IO/确定性，max(UpdatedAt,LastUsedAt) 降序 top5 user/project 优先 +
  procedural/rule ≤3 + rune 边界截断 120 + 空输入非 nil 空数组）+ 新绑定
  GaeaMemoryMorningBrief() (string, error)（JSON 串对齐 GaeaCostGraph 先例，
  ListInSpace("work") 只读 + 近 24h dream 审计计数，零写库零落审计，play 红线
  安全）+ 首页 MorningBriefCard（仅 work 空间渲染，失败/空静默隐藏，全 token）
  + i18n home.morningBrief.* 三语 + gen_bindings 重生成（bindingNames 545）。
  ③**Verifier 产品化**（调研 ★★☆，纯前端零新增绑定）：证据卡三步展开——卡面
  （无 baselinePath 回滚禁用 + 「可复核明细」徽标）→ 声明↔实况 diff（opsJson ×
  GaeaPreview 现取，口径同后端数值容差 1e-9/去空白/公式归一，✓/✗/跳过 + 近似
  比对脚注）→ 操作回放时间线（applyOne 风格中文描述 + 批量 op 折叠，旧卡回退
  beforeSummary）；lib/verifyDiff.ts 纯函数 + types 补 baselinePath/opsJson/
  XlsxOpView/VerifyDiffRow + mock/office.ts 补证据域三绑定。**验证**：Go 全量 0
  FAIL（+25）；vitest **852/852**（150 文件，+27）；tsc/eslint 0；drift PASS
  （545）；版本四处统一 4.14.0；build.bat 冒烟 200。欠账：晨报深度预装配（进
  agent 上下文）列第二刀；浏览器 iframe/键盘级 Input/下载上传/headless UI/
  Windows UIA；Verifier 通道 B 结果未进前端、复核明细绑定留待真实需求；
  本地-云端自动路由 v1 顺延下一刀。详见 releases/v4.14.0.md。

- **最新发布：v4.13.0（2026-08-31）「自动操作·浏览器」**：四柱「自动操作」
  唯一空柱的第一块砖（刀序④）。internal/gaea/browser 新包——msedge 三段式
  定位 + 隔离临时 profile（绝不碰用户主 profile）+ Job Object 绑定 + 页面
  级 CDP WebSocket 会话（gorilla/websocket 零新增依赖，写串行/幂等关/事件
  旁路）+ Ensure 幂等与失联自愈 + URL 白名单 http/https；7 个 browser_*
  内置工具（navigate/read/snapshot/click/type/scroll/close，work 空间，ref
  机制 data-gaea-ref+代数守门、React 兼容 type、envelope 结构化返回）；权
  限门（read/snapshot 只读档恒放行、其余写档弹卡可记忆 + permission
  subjectKeys 追加 url 键可固化窄规则）；事件留痕全链对工具名零特判（会话
  JSONL→trajectory→前端过程卡自动展示）。**真机实测 PASS**（真 headless
  Edge 导航/读/snapshot/双路点击/type/file: 拒绝/旧 ref 失效，1.16s）；零
  新增绑定（544）、前端零改动；Go +25 测试全量 0 FAIL、tsc/eslint 0、
  vitest 825/825（首跑 2 例负载 flaky 复跑 0 failed=既有先例）、drift PASS、
  build.bat 冒烟 200。欠账：多标签页、空闲 TTL 关停、文件下载上传/iframe/
  键盘级 Input 事件、headless 开关 UI、Windows UIA（下一块砖）。详见
  releases/v4.13.0.md。

- **v4.12.0（2026-08-31）「成本透亮」**：调研后第一刀（成本层，
  审计 T0 缺口②③的计价前置）——①GLM 价格表补全（原仅 glm-4.7 一条，GLM
  用量实际未被计价；官方 docs.z.ai USD 价 + 既有 usd_cny_rate 折 CNY，免费
  档计 0，未核实者诚实不入表，glm-5-turbo 显式挡板防前缀误匹配）②编码套餐
  积分口径（/api/coding/ 端点调用 billing_mode=coding_points，EstimatedCost
  恒 0 不进 TotalCost，聚合 glm@coding 单列 Tokens 计入费用 0；summary 新
  增 engines 按引擎聚合，旧 stats.json 兼容）③模型别名注记（官方 coding-plan
  概览核实 4 条自动切换：glm-5.2/5.1→glm-5.3、glm-5-turbo/4.7→glm-5.3-flash；
  ModelInfo 加 alias_of 仅 coding 家族下发，std 不注记；前端模型卡「自动切
  换」标记；RecordCall 记账归一让 glm-5.2 用量落 glm-5.3 价格桶；请求名不改
  写，服务端自行切换）④GLM 目录数据驱动（22 模型迁 glm_catalog.json
  //go:embed 逐字锁定；config 新键 glm_catalog_path 覆盖文件 mtime 热重载 +
  坏 JSON 回退内嵌，照 usd_cny_rate 先例启动注入、非密钥无反射测试）。
  **零新增绑定（544→544，bindings_*.go/bindingNames/bridge/spaceBindings
  零改动）**；Go +5 组测试全量绿、vitest 825/825（+4）、tsc/eslint 0、drift
  PASS（544）、build.bat 冒烟 200。执行方式：探索子代理代码地图 → 后端/前
  端实现子代理串行 → 主控全绿门禁复核。欠账：本地-云端路由 v1 本体（下一
  刀，目标函数=本地优先→缓存命中→峰谷）、Context Compiler 前缀策略化、国
  内 CNY 原价表（bigmodel.cn JS 渲染未取得，现 z.ai USD+汇率）、覆盖文件
  UI 入口。详见 releases/v4.12.0.md。

- **模块制市场调研复扫（2026-08-31，调研非刀，v4.11.0 基线，绑定面 544 不变）**：
  8 子代理并行复扫（办公/造价/记忆/模型中心/编程/创作/轻语/触点），合成
  `docs/market-research-2026-08-31.md`（分模块原始稿 `docs/research-2026-08-31/`；已归档 docs/archive/，结论被 2026-09-01 复扫取代）。
  五个确定性：①「可审计」是最深护城河且行业正在逼近（竞品只有事前审批，
  gaea 事后复核链先发半步，Verifier 应产品化为 UI）②本地优先升级为合规红利
  （《AI 拟人化互动服务管理暂行办法》2026-07-15 施行，个人本地使用不在适用
  范围）③计费快变——GLM Coding Plan 已改积分制+旧模型名自动切换，coding
  端点静态适配有失真风险，静态模型目录需热更新；路由 v1 目标函数=本地优先→
  缓存命中→峰谷，不做语义缓存 ④实时语音换挡窗口已开（GLM-Realtime 0.18
  元/分、Qwen3.5-Omni-Realtime 语义打断与 TurnControl 同构，换引擎边际成本低）
  ⑤OpenClaw（38.8 万星）成开源个人助理事实标准，gaea 以「中文办公纵深+可审计
  +双空间隔离」差异化。编程板块 DSH 独立窗口拍板经调研验证仍成立（壳窗已官方
  协议化，独立壳窗商业脆弱）。**候选刀序（供拍板）**：★Realtime 真机收口
  （provider 白名单可顺手扩 "zhipu"）→★指令内核 v1+Ctrl+K 命令面板先行→★
  模型中心计费三件套（目录热更新/GLM 积分口径/成本仪表 v0）→★Verifier 产品化
  （diff 可视化/操作回放/一键回滚）→★做梦 2.0 主动预取 MVP。不做清单：语义
  缓存、算量赛道、人格包公开分享、原生编程工作台。

- **最新发布：v4.11.0（2026-08-30）「GLM 全模态纵深」**：基线 v4.10.0 + 1
  提交（29c23ee），绑定面 543→544；vitest 821/821、drift PASS（544）。
  详见 releases/v4.11.0.md 与下方刀明细。欠账：做梦 2.0 主动预取、
  Realtime 真机、本地-云端自动路由 v1、iLink 语音/视频、更深跳因果。

- **最新发布：v4.11.0（2026-08-30）「GLM 全模态纵深」**：基线 v4.10.0 + 1
  提交（29c23ee），绑定面 543→544；vitest 821/821、drift PASS（544）。
  详见 releases/v4.11.0.md 与下方刀明细。欠账：做梦 2.0 主动预取、
  Realtime 真机、本地-云端自动路由 v1、iLink 语音/视频、更深跳因果。

- **GLM 生图 + 官方双端点（2026-08-30，v4.10.0 后第一刀）**：①生图后端
  `ai.GLMImageBackend`（kind=glm）——官方 images/generations 端点只发
  model/prompt/size（response_format/n 等扩展字段官方 schema 不收，故不复用
  通用 OpenAI 后端），URL 统一下载转 data URL 复用前端落盘链路，官方错误体
  `{"error":{code,message}}` 原样透出，200 无图提示内容审核，img2img 诚实
  拒绝；app 三处接线（initImageBackend / SetImageBackend / 剧照
  buildPortraitClient），GLM Key 经 `Manager.GLMKey()` 与 chat 同源取用，
  size 参数保留（官方接受）。②官方双端点切换 `SetGlmEndpoint`（绑定面
  543→544）——std=/api/paas/v4（按量付费）coding=/api/coding/paas/v4（编码
  套餐额度，官方 coding-plan/quick-start 核实），后端只收两个官方常量
  （GLMBaseURLStd/GLMBaseURLCoding），GLM 卡片 Segmented 切换+落盘持久化；
  云端引擎不露地址框防线延伸，防 Key 粘错框类事故。③静态目录补生图四模型
  （glm-image/cogview-4-250304/cogview-4/cogview-3-flash，锚定官方图像生成
  API 枚举，18→22）+ 修 glm-5-turbo 误分类（通用 turbo 关键词把它判成生图
  ——GLM 引擎先按官方目录判型再落通用关键词表，回归测试锁死）。前端：
  ImageSection 加 GLM 选项、ImageGenPanel 引擎标签诚实化（此前云端引擎也标
  「本地引擎」）、classifyModel 补 cogview、glmEndpointFamily 判定工具。
  Go +15 测试、vitest 821/821、tsc/eslint 0、drift PASS（544）。

- **最新发布：v4.10.0（2026-08-30）「GLM 引擎 · 办公秘书人设」**：v4.9.0
  后 9 提交（GLM 引擎三轮真机打通 + 工作人设收口 + 审计欠账三刀 + Herdsman
  CLI 透明化），绑定面 540→543；vitest 818/818、drift PASS（543）、build.bat
  冒烟 200。用户已确认 GLM 可用。欠账清单见 releases/v4.10.0.md。
- **GLM 按官方文档重写（2026-08-30，两轮不可用后核对 docs.bigmodel.cn）**：
  根本误判=智谱无 /models 端点（文档仅 chat/completions 等），此前测试连接
  /刷新模型永远失败。重写：glmStaticModels 静态目录（锚定官方模型概览：
  glm-5.3 旗舰/4.7-flash 免费/tts·asr·embedding·rerank 全家桶）+
  TestConnection 走最小 chat ping 真实验证 Bearer Key（错误体官方形态原样
  透出）+ 默认模型 glm-5.3。Go +2 测试、绑定面 543 不变。教训入账：**接新
  云端引擎先读官方文档验证端点清单，禁止拿 OpenAI 习惯外推**。
- **GLM 地址防呆（2026-08-30，真机实测收口）**：用户把 Key 粘进 GLM 地址框
  （云端引擎误露地址框=UI 疏漏）→ base_url=Key 本体 → 请求报 unsupported
  protocol scheme "" 且原生错误回显 Key。修复：地址框仅本地引擎显示；
  SaveEngine 拒绝无 scheme 地址（不回显原值）；LoadState 忽略脏地址自愈
  （重启即恢复预置）；fetchModels 友好错误。+1 回归测试，绑定面 543 不变。
  用户侧：重启应用自愈；Key 已泄漏建议重新生成。
- **模型中心新增 GLM 引擎（2026-08-30）**：智谱 GLM 云端（OpenAI 兼容
  open.bigmodel.cn/api/paas/v4，端点已实测 401=存在）照 DeepSeek 模式全链
  路接入——EngineGLM 常量+预置卡（默认 glm-4.6，排序在 DeepSeek 后）+
  glm_api_key DPAPI 加密落盘/旧明文迁移 + SetGlmKey/GetGlmKeyStatus 绑定
  （541→543）+ GaeaSetProviderKey 映射 + 前端引擎卡 Key 输入。云端属性
  IsLocal=false，离线模式自动跳过、用量统计自动归类。Go +2 测试（预置数
  7→8 断言同步、GLM key/URL/云端属性）、vitest 818/818、tsc/eslint 0、
  drift PASS（543）。
- **Herdsman CLI 错误透明化（2026-08-30，真机诊断收口）**：模型中心「模型
  库」报「exit status 3」三路实证根因=Herdsman 桌面端以管理员运行，skill
  管道（Herdsman-skill-v1）DACL 拒绝普通权限 gaea（桌面端在跑但连不上）。
  runHerdsmanCLI 失败路径改捕获 stdout 透出 CLI 结构化错误（error 字符串/
  对象两态兼容）+ Access denied 定向提示「普通方式重启 Herdsman」；不再
  显示误导性的「请确认桌面端已启动」。Go +3 测试、前端零改动、绑定面 541
  不变。用户侧解法=普通方式重启 Herdsman（取消快捷方式的「以管理员身份
  运行」勾选）。
- **工作人设收口（2026-08-30，用户拍板：gaea=专业严谨的办公秘书，不是文艺
  女青年）**：微信/语音实测「[SPLIT] 裸漏+答非所问」三根因一并收口——
  ①节奏引擎 professional tag 豁免（PAD 标尺下 chatter 阈值形同虚设，gaea/
  secretary 永不拆分，乐园陪伴人格保持原设计）②[SPLIT] 出口净化（内部
  格式协议全仓库零消费方，WhisperChat 三出口共同上游归一为换行）③搜索
  触发收窄（删口语疑问词，保显式动词+硬时效词，宁漏勿误）。新增「办公
  秘书」人格（30→31，结论先行/要点分明/禁撒娇禁文艺腔）。Go +8 测试、
  前端零改动、绑定面 541 不变。人设方向：工作通道（语音/微信/任务对话）
  锁 professional；乐园 29 陪伴人格不动——双空间各归其位。
- **做梦 2.0 蒸馏真实合并（2026-08-30，路线图 T0 第一刀）**：自动做梦只增
  不减的重复记忆有了非破坏合并通道——memory.DistillMergeCandidates 纯函数
  检测（同空间内同名异写 / 异名同 type+kind 同描述；跨空间不成候选=双空间
  红线；封顶 8 条同输入同输出）+ control.DistillMerge 锁内重算校验执行
  （Store.Archive 归档较旧条可逆 + Touch 保留条 + dream 审计
  source=distill_merge，越权配对拒绝）+ 记忆面板「建议」合并卡区 +
  GaeaAcceptMergeSuggestion（绑定面 540→541）。Go +7 测试、vitest 818/818、
  tsc/eslint 0、drift PASS（541）。欠账延续：做梦 2.0 主动预取留后续刀。
- **T0 内核层盘点（2026-08-30，审计「待补充」项收口）**：grep 到实现实证——
  MCP host client（GaeaAddMCPServer/Remove/Retry 等绑定，tools 面）已实装；
  做梦管线（auto_dream + 审计）在库；agent 层 compaction（窗口/阈值/digest
  稳定前缀）+ 工具目录 cache 指纹 + budget 钱闸 + contextview 装配视图均
  已有。路线图 T0 四项真实缺口仅剩：做梦 2.0 主动预取、本地-云端自动路由
  v1+成本仪表增量、Context Compiler 的预算调度/前缀排序策略化（装配明细
  category 条已在 ContextView 呈现）。
- **Verifier 通道 A 引用级深化（2026-08-30，审计欠账收口）**：xlsx_apply 复核
  从「重算零错误」升级为 +「声明↔实况」引用级比对——ChangeRecord 增可选
  opsJson（落卡随卡携带，超限不落），复核逐条回读工作簿（set_value 浮点等值
  比值 / set_formula 剥 = 归一比公式 / replace 比落盘），批量/样式类诚实跳过
  计数，不符即 fail 并给预期/实际示例；旧卡无 opsJson 按不适用降级（宁漏勿
  误）。Go +9 测试、前端零改动（vitest 818/818 沿用，ProgrammingPage 1 例负载
  型超时=既有 flaky，单跑绿）、绑定面 540 不变。勘误：v4.9.0 欠账清单「锚点
  策略刻度对齐」已在 90ab160 交付，一并移除。
- **多跳因果链（2026-08-30，因果推断纵深）**：GaeaWhisperCausalExplain 证据
  从单跳升级 ≤2 跳因果链——buildCausalChains 在 KG「导致」边上有界 DFS（溯源
  +顺藤双向），链按因果序渲染「A → 导致 → B → 导致 → X」，防环双保险 + 链
  封顶 4 条 + 单跳去重；提示词补链语义。Go +4 测试、vitest 818/818 沿用
  （前端零改动）、tsc/eslint 0、绑定面 540 不变。说明：跳数=2 为诚实边界
  （substring 匹配下更深噪声放大），语义锚定多跳推理留后续；文档纠偏——
  成本知识图谱 v4.8.0 已交付（BuildGraph + CostGraphView），从欠账清单移除。
- **最新发布：v4.9.0（2026-08-30）「星枢首页·轻语记忆纵深」**：基线 v4.8.3
  + 15 提交（轻语记忆回放系列 + 图谱三维度 + 跨事实因果推断 + 首页重构
  「星枢指挥所」+ 两段式启动动画 + 工程化收尾），绑定面 535→540；vitest
  818/818、drift PASS（540）、build.bat 冒烟 200（顺带修 pwsh→powershell
  回退）。欠账清单见 releases/v4.9.0.md；下一执行 = Realtime 真机验证轮。
- **首页重构「星枢指挥所」+ 启动动画（2026-08-30，v4.9.0）**：启动默认从首页
  落地（MainLayout 不再恢复上次页面，跨空间恢复保留）；两段式启动动画
  （index.html 静态启动屏 → BootSplash 组件：光环 + 徽记 + 分步状态 + 进度条，
  rAF 节流 / reduced-motion 全降级）；首页 Hero 中轴 + 命令条（orb/打字/语音/
  ⌘K）+ 真实遥测细条 + 能力矩阵 Bento（manifest 驱动，旗舰卡双类提权）+
  门廊 + 信息条；i18n 三语 home.*/boot.* 共 17 键。vitest 818/818、tsc/eslint 0、
  wails build 冒烟正常。说明：语音晶核收进命令条 orb；规格同步
  design-system/gaea/pages/home.md（v3）。
- **跨事实因果推断「为什么」（2026-08-30，审计 §C 深度补口）**：
  GaeaWhisperCausalExplain（绑定面 540，play）——问「为什么<entity>」时确定性
  收集证据（KG「导致」三元组 + event_chain 关联，上限 8 条），当前人格口吻 LLM
  解释因果链（只用证据 / 不编造 / 证据不足诚实说明 / ≤200 字）；无证据零 LLM
  调用回退文案。图谱面板「解释因果」按钮 + 结果区。Go +5 测试、vitest +1
  （818/818）、tsc/eslint 0、drift PASS（540）。说明：证据 = 确定性规则，LLM
  只做人话化；多跳因果图推理留后续。
- **关联入图（2026-08-30，审计 §C「推理仅邻接遍历」补口）**：
  GaeaWhisperGraphSubgraph 并入 AssocIndex 关联边——事实 Subject → 实体节点，
  类型映射中文（event_chain→因果、temporal→时间、thematic→主题…），权重 =
  strength，只并入与子图连通者；前端因果边琥珀色虚线 + 图例。Go +2 测试、
  vitest +1（817/817）、tsc/eslint 0、绑定面 539 不变。说明：event_chain 数据源
  = LLM 整合 + 冷启动启发式（同子类近似因果）；LLM 深度跨事实因果推断留后续。
- **图谱因果维度（2026-08-30，审计 §C 欠账收口）**：extractCausalTriples 从事实
  摘要确定性提取 {因, 导致, 果}（因为/由于…所以…、X导致/引发/造成Y、X让我/使我Y，
  正则 + 测试锁定，直接式剥离连接词），情绪随事实落图；ingest 与文档导入双路径
  接线，图谱面板「导致」谓词天然渲染。Go +6 测试、vitest 816/816 沿用（前端零
  改动）、tsc/eslint 0、绑定面 539 不变。观察项：关联表 event_chain 未在图谱
  展示；LLM 跨事实因果推断留后续设计。
- **图谱情绪维度（2026-08-30，审计 §C 欠账收口）**：Triple 增情绪三字段
  （EmotionLabel/Intensity/Valence），事实提取 attachEmotion 落图（效价派生
  正面/负面/中性），新增 AddTriple 保留情绪（Add 兼容旧调用）；BASIC_PROFILE
  主语实体化（档案键替代硬编码「用户」）；hermes.db 迁移 V13→V14
  （knowledge_triples +3 列，存量安全）；前端图谱边按情绪着色 + 图例。
  Go +5 测试、vitest +1（816/816）、tsc/eslint 0、绑定面 539 不变。欠账延续：
  因果维度未做（需 LLM/事件链）、时空索引未建（暂以 CreatedAt 承担）。
- **记忆重述 + 锚点刻度对齐（2026-08-30，记忆回放收尾两刀）**：
  GaeaWhisperMemoryRetell（绑定面 539，play）——LLM 以当前人格口吻把情节/锚点
  记忆重述成故事（输入复用确定性回放：摘要+情绪+原文对话；第一人称/称「你」/
  ≤300 字），episode/anchor 双入口 + 前端「让 gaea 重述这段记忆」按钮（MemoryRetell
  组件三态）；锚点策略阈值刻度对齐（weight≥2/selfRelevance≥4 在 0-1 抽取标尺上
  不可达 → 0.9/0.8/0.9，偏离 ackem 已注释），补策略测试 2 组。Go 全量绿、
  vitest 815/815、tsc/eslint 0、drift PASS（539）。
- **时间锚点「重访那一天」（2026-08-30，记忆回放续刀）**：写路径接线——
  ShouldWriteTemporalAnchor/BuildTemporalAnchor 此前有定义无生产调用（同类骨架
  欠账），现经 MemoryWritePayload/IngestTurnArgs 的 TemporalAnchorSink 逐事实
  评估（IsNew=计数差）→ Orchestrator.AddTemporalAnchor（回合锁内）落
  State.TemporalAnchors 随 companion_state 持久化；读路径 GaeaWhisperAnchors +
  GaeaWhisperAnchorReplay（锚点→事实轮次→情节→原文回放，绑定面 538）；前端记忆库
  「纪念日」tab + 回放弹窗（ReplayDialogue 与情节回放共用）。Go 8 新测试、vitest
  +2（813/813）、tsc/eslint 0、drift PASS。观察项：锚点策略阈值（weight≥2/
  selfRelevance≥4）与 LLM 抽取标尺（0-1）刻度不一致，常命中周期纪念日分支，
  刻度对齐留后续决策；LLM 重述未做（当前为确定性原文回放）。
- **轻语记忆回放（2026-08-30，审计 §C 欠账收口）**：新绑定
  GaeaWhisperEpisodeReplay（绑定面 535→536，MemoryB/play）——按情节 ID 从
  hermes.db 读情节 + SourceSessionID/[StartTurn,EndTurn] 从 chat_history 确定性
  重建原始对话（不调 LLM），过旧情节 Replayable=false 回退仅摘要；前端情节详情
  弹窗新增「回放原始对话」气泡视图（用户/gaea + 轮次，加载/失败/不可回放三态）。
  Go 3 测试 + vitest 2 测试；验证：Go 全量绿（whisper 偶发挂起为环境 flake，
  单跑绿）、vitest 811/811、tsc/eslint 0、drift PASS（536）。欠账延续：时间锚点
  「重访雨夜」入口、图谱情感/因果维度。
- **构建冒烟自动化（2026-08-30，v4.8.3 教训收口）**：build.bat 真实退出码 +
  产物新鲜度守卫（构建前删旧 exe、被常驻实例锁定时显式报错）+ 默认自动冒烟
  （.tmp 临时副本 → scripts/smoke.ps1，18999 /api/health 200 + status=ok 响应
  体校验，失败即停并提示勿发布）；`build.bat skip-smoke` 跳过（快速迭代）；
  smoke.ps1 增 Start-Process 失败显式报错、响应体校验、finally 判空回收。
  验证：默认路径实跑绿（构建 44.5s + 冒烟通过 + 进程回收）；失败路径假
  wails 桩实测 [FAIL] + exit 1。
- **最新小步（2026-08-30，欠账收尾）**：①VoiceStart realtime 门小修——
  端到端回复走服务端 response 事件（事件泵不经 whisperChatFn），realtime
  在位时不再要求 whisper 对话回调，whisperChatFn=nil 也可启动；拼接管线
  双门（ASRReady + WhisperReady）逐字节保留，新增两回归测试 ②持久化套件
  统一——desktop_session SaveModes 走 fileutil.AtomicWrite（临时文件+
  rename，不留半截 JSON）、archive JSONL 单次 Write 落整行（消除双写撕裂
  窗），新增 7 回归测试 ③XlsxPreview 大表格行虚拟滚动（观察项收账）——
  300 行以上只渲染可见窗口 ±10 overscan + spacer 保滚动条总高，冻结行常
  驻；2000×100 预览不再整表 20 万 td，小表全量渲染逐字节不变；2 新
  vitest。验证：Go 全量绿、vitest **809/809**（148 文件）、tsc/eslint 0、
  绑定面 535 不变、drift 不受影响。
- **最新发布：v4.8.3（2026-08-30）**——微信图片双向真协议（v4.8.2 发布
  当日真机实测复盘五刀）：①出图回推 getuploadurl→AES-128-ECB→CDN 密文
  上传→image_item 卡片+caption 补发，真机 delivered（此前 SendFileCard
  是孤儿=真凶）②发图识别 type=2+aeskey 解密下载→魔数终审落盘，真机两
  连发通过 ③识图模型升级多模态 Qwen3.6-35B 优先（手写体强、零额外显存），
  PaddleOCR→MinerU→OvisOCR2 降兜底 ④身份类问题跳过联网搜索（乱字母根
  因）⑤关键坑锁死（type=2/aes_key=base64(hex)/上传域与扫码 baseurl 无
  关）。协议三方印证（本机抓包解密+hermes-agent+openilink SDK）。验证：
  Go 全量绿、绑定 535 不变、前端零改动。详见 releases/v4.8.3.md。
- **下一执行**：Realtime 真机验证轮（真 key 下端到端对话/打断体感/AEC，
  S2 骨架已就绪）；手写识别质量复测；iLink 语音/视频 item 未探明（静默
  跳过）=观察项；生命库可写化=观察项。
- 构建注意：wails build 走 `build.bat` 的 TMP/TEMP 重定向到 `.tmp`（规避 SAC
  策略拦截）。

## 长期规划（权威文档）

- **`docs/gaea-nextgen-roadmap-2026.md`** —— gaea 长期规划（2026+）：
  8 板块竞品调研 + WorkBuddy×灵犀 对标 + **版本重定义（双空间：工位/乐园）** +
  四层落地（后端/前端/UX/UI）+ 执行计划（阶段 0 地基 → 阶段 1 双空间内核 →
  阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板（重要）**：**工作与娱乐分开、互不干扰**——工位（办公/造价/编程/资料记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区、模型策略各配、上下文永不跨界；
  跨空间仅用户显式发起。~~"陪伴×办公融合"~~ 已删除。
- **用户拍板（2026 追加）**：**编程板块与办公板块不得混合**——保持现状 DSH 独立窗口
  （iframe），不并入工位、不共享工具面（防工具膨胀）；不做原生编程工作台/MCP 板块总线
  喂办公工具给编码 agent（长期规划 §4/§10.3/§13.1 已修订）。
- 本轮调研另一项关键纠正：**灵犀 = 金山 WPS 独立 AI 办公 Agent**（非阿里/通义）；
  **WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台**（非 Kimi 系，Kimi Work 是月之暗面的）。

## 下一步执行（按执行计划 §14）

### 阶段 0 · 地基修补 —— ✅ 全部完成（12 提交）
- [x] **S0.1** AgentRunner 回合级 map 并发加固（turnMu）+ 回归测试 —— `96644b9`（临时 worktree 实证修复前必崩）
- [x] **S0.2** tool.Registry RWMutex + suspended 幽灵名修复 —— `a77ce5e`（全包 -race 绿）
- [x] **S0.3** gate 改 atomic.Pointer[gateWrapper]（撕裂换闸）—— `8b661d2`
- [x] **S0.4** retry_until 走统 gated 派发（堵绕过审批 shell）—— `87bbf20`
- [x] **S0.5** CI 后端 job 加 `go test -race`（独立 ubuntu job）—— `afefa11`
- [x] **S0.6 edit_file 工具层**（grep/edit_file/multi_edit/edit_lines/move_file 五工具 + 名单全对齐 + 双路径失效特判）—— `7d560e6`
- [x] **S0.7/隔离岛** knowledge 索引缓存 `aadb7aa`；office 原子写 `5a82209`；secure 非 Win AES-GCM `fa1778f`；tasks LRU `2a3ca50`
- [x] **前端** 聊天 memo+尾部窗口 `9a41090`；keepAlive 轮询门控 `d373fba`
- [x] 遗留收尾：gate_test.go stubGate 计数器加锁（v4.8.2 已清账）；持久化
  套件统一（desktop_session SaveModes → fileutil.AtomicWrite；archive
  JSONL 单次 Write 落整行）——本轮收口

### 阶段 1 · 双空间内核（后端 space 维度）—— ✅ 全部完成（S1.1-S1.5）
- [x] **S1.1 空间维度落地**（设计文档 `docs/gaea-space-dimension-design.md`）：
  - [x] S1 列落库（SchemaV14）`5c24d16`｜S2 会话空间（目录分区+日志/checkpoint space+开关）`0722aeb`｜S3 子代理继承 `76f565e`｜S4 产物分区+绑定面（499→502）`5c9fc4e`
- [x] **S1.2 记忆空间隔离器**（设计 `docs/gaea-memory-isolation-design.md`）：后端 A+B `819d7ff`+`f0187a2`（写侧盖章/读端隔离/UnifiedSearch scope）；前端 C `53d621d`
- [x] **S1.3 模型 profile + 工具标签**（装配族 `f8612d0`）：`[space_profiles]` 段 + 工具装配期过滤（work 33/play 1/shared 13）+ MCP spec 层滤
- [x] **S1.4 任务分账**（`5ccaa7a`）：per-space 并发/优先级 pickNext + HasActiveInSpace + cron 显式 work + GaeaTaskList 变参
- [x] **S1.5 空间策略**：S1.5-A 权限按空间+hardAsk 参数化+persist_allow 分段（装配族 `f8612d0`）；S1.5-B play 护栏 5 处钳制（`21f8978`）
- 设计文档：`docs/gaea-space-dimension-design.md` / `gaea-memory-isolation-design.md` / `gaea-space-assembly-design.md`

### 阶段 2 · 双空间壳（前端两视图，S2.1-S2.3）—— 全部完成
- [x] **S2.1 壳层重构**（设计 `docs/gaea-space-shell-design.md`）：
  - [x] manifest 空间维度（Go + TS 同构）：work/play/shared/**independent**（编程 DSH 独立窗口，
    用户拍板不并入工位/乐园）——导航/启动器/快捷键/导航事件白名单全部由 manifest.space 派生
  - [x] 两视图壳层：CommandRail 空间切换（appStore.space 持久化 `gaea.shell.space`）+ 按空间菜单
    （shared + 当前空间）；独立窗口板块单独 rail 入口；每空间最后页面持久化 `gaea.shell.page.<space>`
  - [x] 双首页：ModuleLauncher 按空间装配（工位=任务工作台门面 / 乐园=会客厅创作间门面）
  - [x] 事件订阅空间过滤：events.ts `subscribeForSpace`（payload.spaceId/space 过滤）；
    TaskCenter/useRunningBadge 只收工位任务（TaskView.spaceId）
  - [x] Ctrl+K 搜索 scope：工位/乐园/全部三档，默认当前空间（工位=UnifiedSearch；乐园=项目搜索）
- [x] **S2.2 增量**：
  - [x] 工作台 localStorage 空间分键 + 迁移（`workbenchStorage`：rightTab/布局/chatTab/
    compact/focus → `gaea.work.*`，旧 key 只读回退）
  - [x] 性能门控：空间切换剪枝 keepAlive 保活页（跨空间页面卸载，后台轮询归零）
  - [x] i18n 全铺第一刀：LocaleProvider 提升根级 + S2.1 新壳 chrome 文案三语字典化
  - [x] **i18n 存量切片（壳层 chrome）**：MainLayout（strip/rail/telemetry 超载警告等）
    + ModuleLauncher（统计卡/语音/信息条/相对时间）+ SearchModal（分类标签/计数）
    全部三语字典化（~90 新 key）；fmtWords/fmtRel 接 translator
  - [x] **i18n 存量切片（SettingsPage 外壳）**：控制室/搜索/分类导航/帮助面板
    三语化（~40 新 key）；分类 label/desc 接 labelKey/descKey（keywords 为搜索数据保留 zh）；
    SettingsPage.test 包 LocaleProvider + 固定 zh
  - [x] **i18n 决策定稿（2026 追加）**：采用审计 §405「诚实 zh-only」选项——
    **壳层 chrome + 设置外壳三语（已完成，消灭壳层混合语言根因）；页面内容层
    保持 zh 单语**，不再逐页铺 en/zh-TW 字典（~5000 字符、边际价值≈0、个人
    中文工具无国际化受众）。若未来需要国际化，按 S2.3b WireShape 模式整页迁移。
  - [x] **页面迁入收口**（`docs/gaea-page-migration-design.md`）：处置矩阵按版本锚点
    落账——办公升格/造价领域包/编程独立/记忆侧栏形态/微信触点 v4.0 已达成；
    **P1 对话流**（chat shared→play：工位不再有独立聊天板块，乐园=会客厅）本轮落地；
    创作间合并=v4.3（§10.4 明示）、模型中心并入设置=v4.1+、微信任务入口=v4.4
- [x] **S2.3 bridge 分面 + types 生成化**：
  - [x] `spaceBindings.ts` 分类表：AppBindings 214 方法显式分类（work/play/shared/
    independent），satisfies + 双向断言编译期全覆盖；UnifiedSearch=shared（scope 隔离）
  - [x] bridge 三门面 workApp/playApp/sharedApp（类型级 + 运行时双保险）；
    TaskCenter/useRunningBadge 消费 workApp
  - [x] types 生成化第一刀：typesGenerationCheck.ts 编译期漂移校验 + 修 SessionMeta.spaceId
  - [x] **S2.3b types 全量迁移**：WireShape 剥生成类实例方法，55 个重叠类型改别名
    （1375→1065 行）；增强类型（TaskStatus/TaskView/SearchScope 等）保留手写；
    顺带修 FileSearchHit.modTime / UpdateInfo.version / FilePreview 契约漂移
  - [x] **S0.7 遗留：151 hex token 化**（审计点名的 VoiceSettingsPanel 浅色主题崩坏
    真 bug 已修——dark-zinc 硬编码 → 主题 token）：语义色（成功/警告/危险/文本/表面/
    边框/主色）全部 token 化；图表调色板/品牌识别色/图片覆盖层/主题预览样张/标注数据色
    显式 `// hex-exempt` 豁免（审计口径「图表豁免」）；新增 eslint 规则
    `local/no-raw-hex`（图表/数据文件白名单 + 行内豁免注释）防回归

### 阶段 3 · 领域包纵深（v4.1 → v4.4）—— v4.1 设计定稿
- [x] **v4.1 办公信任链设计**（`docs/gaea-v41-evidence-chain-design.md`）：
  evidence.Record（原文摘要/来源/模型/时间戳/回滚信息）+ Apply→Verify→Journal
  三段式 + Verifier 双通道（结构/引用完整性 + PDF 视觉 diff）+ 基线快照回滚
  （用户手工编辑冲突保护）+ GB/T 9704 红头 lint 第一刀；Step 拆 v4.1a/b/c
- [x] **v4.1a 证据链核心**：ChangeRecord（原文摘要 8KB）+ ChangeLedger（ctx 盖章）
  + JournalStore（JSONL 追加 + turn markdown 投影）；AgentRunner 回合收尾 flush
  （play 不落盘红线）；edit_file/write_file/move_file 接入上报
- [x] **v4.1a2 证据链补齐**：multi_edit/edit_lines 逐条摘要；xlsx_apply 接入
  （App 层写 Journal，SessionID 取自控制器，play 红线过滤）；新绑定
  GaeaJournalList（gen_bindings 重生成 bindingNames 503）+ 前端「证据」入口
  （DeliverablesPanel 证据链折叠区，展示最近证据卡）
- [x] **v4.1b Verifier + 回滚**：基线快照（写盘工具 + xlsx 应用前）；GaeaVerifyRecord
  双通道复核（A 结构/引用完整性 + B 基线 PDF 视觉健全性）+ verdict 落库；
  GaeaRollbackRecord 基线回滚 + 手工编辑冲突保护；前端证据卡复核/回滚操作
- [x] **v4.1c 规范包**：internal/office/standard GB/T 9704 红头 7 要素 lint +
  GaeaDocumentLint 绑定 + OfficePanel「规范体检」入口

### v4.2 造价 AI 化 —— ✅ 全部完成（v4.2.0 发布）
- [x] **v4.2 设计定稿**（`docs/gaea-v42-cost-ai-design.md`）：三支柱契约 +
  SchemaV15 数据模型（cost_inquiry_records / cost_stage_values）+ 绑定面 + Step 拆分
- [x] **v4.2a 组价底座**：`cost.PriceBand` 价格带推荐纯函数（P25/P50/P75/离散度/
  离群/置信度/证据链 BandSource，R-7 分位数与 costref 同口径）+ `RecommendPrice`
  五档；22 测试
- [x] **v4.2b 询价飞轮核心包**：`costinquiry` 四源归一数据点（信息价/OCR报价/供应商
  比价/手动询价）+ 到期预警（valid_until ≤ today+days）+ 调差建议（标题归一化匹配
  + |差幅|>2%）；7 测试
- [x] **v4.2b 五算对比核心包**：`coststage` 估/概/预/结/决阶段值（UPSERT）+ 对比
  计算（环比/累计差/除零保护）+ 偏差特征（正常<5/关注 5-15/异常>15 + 规则文案）；
  17 测试
- [x] **v4.2c AI 组价绑定**：`GaeaCostCompose`（关键词+语义召回+rerank 精排 →
  PriceBand → LLM 人材机拆解 `routeSensitiveLocal("office")` 失败规则降级 →
  建议视图）+ `GaeaCostComposeApply` 回写
- [x] **v4.2d 前端面板**：ComposeModal（测算明细行「AI 组价」：价格带卡/证据链 8 列
  表/人材机行编辑 → 应用为明细行）；询价视图（CostLibraryView 第三视图：数据点
  CRUD/四源徽标/预警横幅/调差一键更新）；FiveCalcPanel（项目详情五算区：输入保存/
  对比表着色/偏差卡）
- [x] **后端集成**：SchemaV15 正式迁移收编 + CostB 门面 11 新绑定（gen_bindings
  重生成，绑定面 506→517）+ bindingNames/spaceBindings(229)/bridge/types 接线

**v4.2 造价 AI 化已收官**（bindingNames 517 / spaceBindings 229 / vitest 759
+ go 全量绿）。剩余领域包：v4.3 乐园 / v4.4 触点（§10.4）。

### v4.3 乐园做深 —— ✅ 全部完成（v4.3.0 发布）
- [x] **v4.3 设计定稿**（`docs/gaea-v43-play-deepen-design.md`）：4 份只读调研
  （轻语关系记忆/情感语音 TTS/创作间世界模型/角色资产库）结论入账——后端骨架约
  70% 已存在，本版以「接线 + 参数扩展 + 持久化闭环」为主
- [x] **v4.3a 轻语记忆持久化闭环**：memory_associations/user_habits/temporal_anchors
  三表补 repo（此前有 schema 无 repos）+ 装配进持久化链 + ReseedAssociationGraph
  打通 + hermes.db 外键延迟检查实证修复
- [x] **v4.3b 关系图谱子图召回**：KnowledgeGraph.QuerySubgraph 多跳邻接（BFS+去重+
  权重，10 测试）+ GaeaWhisperGraphSubgraph 绑定（play）
- [x] **v4.3c 主动关心**：GaeaWhisperProactiveNow 评估绑定（门控+合成器复用）；
  前端「轻语先开口」；定时推送留后续小步
- [x] **v4.3d 情感语音**：TTSProvider.SynthesizeWithParams 参数扩展（cosyvoice 工厂
  修复不丢 voiceDescription、edge SSML 参数化）+ GetEmotionVoiceParams 情绪映射 +
  EmotionState.Mood 长期心境维（EWMA α=0.01 持久化）+ TTSSpeakBase64WithParams
- [x] **v4.3g 创作间图文联动**：章节配图复活死绑定（ChapterPage 按钮 + 弹窗）；
  GaeaGenerateBookCover 3:4 书封落 play exports（修 Windows 盘符卷 bug）
- [x] **绑定集成**：gen_bindings 522（+5，显式覆盖归位 voice/novel）+ bindingNames/
  spaceBindings(233)/bridge/types 接线

### v4.3 后续小步 —— ✅ 全部完成（v4.3.1 发布）
- [x] **主动关心定时推送频控（v4.3c 补完）**：app 层 ticker 四信号评估（AttentionManager
  频控 ≤3 条/小时 → MatchHabits dnd 作息尊重 → DetectSpecialDatesV2 生日祝福（每天首条、
  人格感知提示词）→ 门控+合成器）→ `gaea-whisper-proactive` 事件推前端（space=play）；
  新绑定 GaeaWhisperProactiveConfig/SetProactiveConfig（开关/上限/间隔/免打扰时窗）；
  前端 WhisperGraphPanel 订阅显示推送气泡（birthday 徽标）；16 Go 测试
- [x] **设定页维度化编辑器 + 伏笔登记表 + 一致性面板（v4.3e/f 落地）**：NovelSettingPage
  「维度化」模式（6 维度卡片就地编辑整存，复用已有绑定）+ ForeshadowPanel（状态流转/
  回收率）+ ConsistencyPanel（三类告警/重新检查）；7 vitest
- [x] **角色参考图 + 生图参考槽（v4.3g 补完）**：characterlib SchemaV2 迁移（reference/
  gallery_images 两列幂等）+ CharacterGeneratePortraitWithRef（img2img 参考槽 denoise
  0.55 + 模型门禁）+ 前端参考图管理（以参考图生成剧照）；8 vitest + 5 Go 测试
- [x] **文本朗读情绪 UI（v4.3d 收尾）**：EmotionSpeakSelector（9 标签对齐 EmotionVoiceMap）+
  会话情绪自动跟随 + TTSSpeakBase64WithParams 携带情绪（无情绪回退原路径）；3 vitest
- [x] **绑定收口**：gen_bindings 525（+3）+ bindingNames/spaceBindings(235)/bridge/types

**v4.3.1 后续小步已收官**（bindingNames 525 / spaceBindings 235 / vitest 789
+ go 118/118 包）。剩余：v4.4.1 触发已有能力（生图/办公任务路由 + 图片/文件卡片协议）+ 语音双通路（work 指令/play 闲聊分叉）+ 全局离线模式总开关（§10.4）；观察项（主动关心配置面板、角色 gallery 管理、IP-Adapter 节点级参考槽）。

### v4.3.2 首页重构（双翼·中庭 + 空间导航收敛）—— ✅ 已完成（v4.3.2 发布）
- [x] **首页「双翼·中庭」**：中庭 = 语音 + 打字一体对话条（输入框打字走
  `VoiceChatText` 复用语音对话管道，回复经 voice:reply 回传；orb 放大 148px
  呼吸环磁吸锚点；hero 让位顶部细眉——遵循 design-taste-frontend：不对称组织、
  避免等宽栏与居中 hero 俗套）；左翼「书房」2×2 紧凑格（办公/造价/记忆/模型）；
  右翼「庭院」纵向列表（聊天/小说/绘梦/角色）；门廊 = 编程（独立窗口徽标）+
  设置。命名 工位→**书房**、乐园→**庭院**（三语字典同步）。
- [x] **空间导航收敛**：移除一级导航（rail）顶部空间切换按钮（此前位置隐蔽）；
  `navigateBoard` 统一导航按板块 manifest.space 自动切空间（书房板块→work /
  庭院板块→play / 编程→independent / 设置→shared）；rail 展示全部板块；
  搜索 scope 文案同步 书房/庭院（useSpaceScope）。
- [x] **验证**：Go 全量绿；vitest 789/789（scope 断言同步）；tsc/eslint 0；
  版本五处统一 4.3.2；桌面端 gaea-v4.3.2.exe（33MB，SHA256 6a0486db）+
  冒烟 /api/health 200。提交：9789d7e / 71b0f2b / 766a9f6 / 0f21650。
- 遗留：中庭对话条桌面端语音+打字双通道实际体验待验证；双翼板块卡数量由
  manifest 派生（新板块自动入对应栏）。

### v4.4.0 触点一期（微信遥控器·离线代办）—— ✅ 已完成（v4.4.0 发布）
- [x] **主动推送通路**：weixin.Server 最近活跃会话记忆（handle 时记录
  fromUser/contextToken）+ Push 主动回推文本（无会话报错）；httptest
  校验 sendmessage 目标与文本 item（3 测试）
- [x] **离线代办提醒域**（weixin_reminder.go）：中文时间解析（相对时长/
  日期前缀+段词/裸时刻 20 用例表驱动，中文数字「十」进位，明早/明晚
  预处理拆词，无段词字面解释）→ wxReminder JSON 持久化（重启恢复，
  done 不重推）→ tryWxReminder 微信消息任务路由（提醒类接管/解析失败
  回格式提示/其余走聊天）→ 20s ticker 到点回推（失败重试 ≤5 标 failed）
  → remindersEnabled 开关持久化
- [x] **WeixinPage 落地（书房板块）**：扫码绑定流（QR 轮询/need_verifycode
  配对码/confirmed 落 WhisperAssistantSave 自动重拉通道）+ 通道状态徽标
  （运行/过期/未绑定）+ 提醒列表（手动新建/删除/回推开关）+ 指令说明；
  weixin 板块 Page=""→WeixinPage、inMenu=true（rail + 首页左翼书房格
  manifest 派生自动生效）
- [x] **绑定面 525→530**：WhisperWeixin* 4 + WhisperAssistant* 3 自
  LegacySurfaceNames 转正；WeixinReminder* 5 新增（voice 门面）；
  spaceBindings 235→247 全归 work；bindingNames.ts 同步
- [x] **验证**：Go 107/107 包；vitest 789/789（manifests/launcher/
  spaceBindings/board_manifest 锁数量断言 +12 同步）；tsc/eslint 0；
  drift 530 PASS；版本五处统一 4.4.0；gaea-v4.4.0.exe（35MB，
  SHA256 ee9b45c2）+ 冒烟 200。提交：v4.4a 后端 / v4.4b 前端 / release
- 遗留：回推目标=最近活跃会话（多联系人场景由多助手隔离）；iLink 图片/
  文件卡片协议未探明（v4.4.1 触发生图/办公任务回推时处理）；LLM 意图
  路由（通用任务化）留后续刀

### v4.5.0 指令中枢（统一意图路由 + 语音指令）—— ✅ 已完成（v4.5.0 发布）
- [x] **规划修订**（docs/gaea-nextgen-roadmap-2026.md §10.4a + 阶段 4）：审查
  结论=§8 革命性跃升未实质落地，插入 v4.5 指令中枢主轴（同内核多入口），
  v4.6 端到端实时语音接替排期
- [x] **S4.1 intent 解析包**：导航/生图/状态/提醒四类意图 + 板块别名表；
  宁漏勿误纪律（21 命中 + 8 误判防线表驱动）
- [x] **S4.2 能力执行层**：App.routeIntent——navigate(manifest 校验+事件)/
  generate_image/status/reminder(复用离线代办 source=voice)；零新增绑定
- [x] **S4.3 语音通路**：setWhisperChatFn 闭包分流——命中即能力执行经 TTS
  播报（含打断），未命中透传聊天；voice 包零改动
- [x] **S4.4 前端**：events.ts INTENT_NAVIGATE（22→23）+ MainLayout 订阅 →
  navigateBoard 自动切空间
- [x] **验证**：Go 108/108；vitest 789/789（events 断言同步）；tsc/eslint 0；
  版本五处 4.5.0；gaea-v4.5.0.exe（35MB，SHA256 027c726d）+ 冒烟 200
- 遗留：能力面仅四类（导航/生图/状态/提醒）；生图结果无语音播报进度；
  LLM 兜底分类器未接（规则引擎对未覆盖意图一律走聊天）

### v4.6.0 双空间收尾·纵深补课（审计后第一轮收账）—— ✅ 已完成（v4.6.0）
- [x] **红线 ①记忆注入按空间收窄**：`boot/sysprompt.go` 系统提示词索引 +
  `controller_memory.go` refreshMemoryLocked 两个生产调用点传 `Options.Space`
  → InSpace 读端视图（work 只注入 work / play 只注入 play / mode=off 旧行为）；
  回归 TestRefreshMemorySpaceIsolation + TestBuildSystemPromptMemorySpaceScoped
- [x] **红线 ②任务分账生产启用**：`[tasks]` 配置段（max_concurrent/per_space/
  priority）→ startTaskScheduler 默认 {work=1, play=1} + 价格抓取优先
  （price_fetch=20/file_index=10）；显式 `per_space = {}` 关分账回退全局 sem；
  回归 TestTasksConfigTOML + TestTaskSchedulerOptionsDefaults
- [x] **红线 ③事件过滤推广**：onTaskEvent(cb, space?) 订阅层过滤 + TaskCenter/
  useRunningBadge/PriceSourcesPanel/WorkspaceSearchPanel 按 work + MainLayout
  subscribeForSpace；回归 onTaskEvent 空间过滤测试
- [x] **治理收尾**：keepAlive 裸轮询 8 处（TaskCenter/SubagentsPanel/
  FeatureModelBar/ProgrammingPage/BenchmarkSection/useStatsState/
  useImageGenQueue/useBridgeWatch）全接 usePollingGate；CSS 真硬编码 token 化
  （novel 阅读高亮/批注色板 --novel-read-*/--ann-*、chat-board 混白）
- [x] **纵深 Mood→TTS**：MoodToVoiceDescription 连续韵律 + WhisperChatFn 透传
  mood + 中性轮次心境主导；回归 TestMoodToVoiceDescription +
  TestManager_MoodDrivesNeutralTurnProsody
- [x] **纵深 Verifier 通道 B**：soffice→pdftoppm→纯 Go 像素差异率（页数联合
  判定 pass/warn/fail，渲染降级 warn），审计产物落 journal/verify/<id>；覆盖
  全部有基线的写盘工具；失败回 Plan（证据卡内联结论 + xlsx 重新规划按钮）；
  回归 TestPixelDiffRatio + TestRunVisualDiffVerdicts
- [x] **纵深 询价飞轮**：调差异常分级（正常/关注/异常）+ PredictNext 线性回归
  预测 + GaeaCostImportApply 变参 inquirySource → OCR 报价单幂等入询价库；
  回归 4 个新测试（含飞轮接线）
- [x] **验证**：Go 全量绿（无 FAIL）；vitest 791/791（144 文件）；tsc/eslint 0；
  绑定面不变（变参向后兼容）；版本统一 4.6.0
- 欠账（如实列示于 releases/v4.6.0.md）：规范包机制化 / 成本知识图谱+归因 /
  生命库可写化（评估=不盲写 Herdsman 库）/ Verifier 通道 A 引用级深化 /
  Mood 不进前端手动朗读（设计如此）

### v4.6.1 微信统一路由·规范包机制·归因对标（审计补课续刀）—— ✅ 已完成（v4.6.1）
- [x] **S4.5 微信接统一路由**：routeIntentWithResult（产物感知，routeIntent
  包装不变）+ 微信回调提醒之后先过统一路由（navigate/生图/状态/提醒全命中，
  未命中才走聊天）；iLink image_item/file_item 防御解析 + 非文本消息转模型
  提示行（发图即识别入口），未知空项宁漏勿误；回归 3 测试
- [x] **规范包机制化**：standard.Checker 注册表 + LintDocument 聚合；红头要素
  （既有）+ 造价工程表式（工程名称/编制依据/单位造价/人材机/合计/说明）双检查
  器；GaeaDocumentLint 切 LintDocument；OfficePanel 按规范包分组；回归 2 测试
- [x] **成本归因对标**：costref.ComputeAttribution 纯函数（P25/P75 带宽 + 中位
  基准，差幅等级/贡献金额/主因 TopDrivers/带宽退化 ±10% 兜底/无参考另计）；
  新绑定 GaeaCostAttribution（531，参考池排除本项目防自对标）+ FiveCalcPanel
  归因区；回归 3 测试
- [x] **验证**：Go 全量绿；vitest 791/791（144 文件）；tsc/eslint 0；绑定面 531、
  spaceBindings 248、drift PASS；版本统一 4.6.1
- 欠账（如实列示于 releases/v4.6.1.md）：iLink 文件卡片/字节级图片识别待真机
  收敛；成本知识图谱可视化形态；生命库可写化维持评估结论；微信通用能力触发
  依赖卡片协议

## v4.x 执行审计（2026-08-30）

- 全量「承诺 vs 代码」对照（三路探查，逐项到文件行号）已记录
  `docs/audit-2026-08-30-v4-execution-review.md`：阶段 0 地基五项与双空间内核
  骨架为真；红线缺口三条（记忆注入主链路跨空间未接线 / 任务空间分账未启用 /
  事件订阅过滤仅 1 处）+ 领域包纵深欠账表（Verifier 通道 B=页数对比、规范包
  =单 lint、询价无异常检测/预测、OCR→询价飞轮反向未接、成本图谱零实现、
  Mood 只存不用、生命库只读）+ 补课刀序（§E）。纪律修正：每刀验收加
  「纵深检查」，发布说明列欠账清单。

## 纪律（沿用）

- 每 Step 独立提交可回退；旧数据只读兼容；回退演练纳入验收；不做新板块、不堆功能；
  docs/ 过时文档一律归档到 `docs/archive/`（权威 = 长期规划 + AGENTS.md + 本文件）。
