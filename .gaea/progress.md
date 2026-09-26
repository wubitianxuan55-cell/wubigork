## 非版本刀（2026-09-26）真机走查池全清账：plan-mode 审批卡可视确认（零代码）

- **环境**：xAI 重新登录后，v4.418.0 壳 + GAEA_WALKTHROUGH=1 沙箱（首刀实战）+ CDP。全程零触碰用户工作区（黄甲 mmin 复核零新文件）。
- **plan-mode 全链实弹**：/plan on → 政策注入 → 任务「查工作区+建 hello.txt」→ 模型只读研究（list 1 工具）→ **真调 exit_plan_mode → ask 审批卡渲染（计划审批/批准/继续计划 双选项）** → **用户本人在屏幕上点击批准** → 闸翻转 off + 叙事「计划已批准」落历史 → write_file 执行 → hello.txt 落沙箱 → 回合完成。三重传达/键盘/产物全数在位。
- **池清账**：①「审批卡路由漂移下未可视确认」✅ 关账——不仅可视确认，且获得真实用户点击交互（比脚本点击强得多的证据）。②「pre-turn 动词 Notice 不可见」✅ 实锤为真实缺陷（Notice 渲染在「过程」折叠卡内默认不可见），**候小刀=动词回执提为可见 chip**，维持挂池。
- **沙箱开关实战评价**：GAEA_WALKTHROUGH=1 全程无感隔离——写文件落沙箱、会话落沙箱、清场删目录。真机走查从此默认带开关。
- **坑**：AskCard 审批卡由用户真人点击（最初误判为自动批准/超时——Controller.Ask 无超时、AskCard 无预选，人类反应时间戳是最终判据；诊断交互类问题先确认屏幕前有没有真人）。

## 最新发布：v4.418.0（2026-09-26）「走查沙箱开关：GAEA_WALKTHROUGH=1 强制隔离工作区」

- **刀型**：真机走查险情根因封堵（候产品刀当轮立项落地）。Go 1 文件改+1 新测试，绑定面 720 零变更，前端零改动。
- **背景**：真机走查「先建专用会话再发消息」无效——装配前 NewSession 必然诚实报错，首条消息走自动装配+自动恢复，workspace 解析到哪回合落到哪（当日实证接进用户真实会话，模型调用即败未持久化）。
- **落地**：GAEA_WALKTHROUGH=1 → config.Load() 强制 workspace=%TEMP%\gaea-walkthrough\workspace（自动创建），覆盖用户/项目配置；未设零变化；激活 slog.Warn 可审计；删沙箱目录即清场。
- **测试**：walkthrough_test.go 两例（开=恒沙箱/关=零触碰）；config+app 包回归绿；全量 ci 绿（OriginalSinPage.tabs 负载 flaky 一轮隔离复跑绿，v4.414.0 同款）。
- **真机验证实录**：沙箱模式启动→GaeaInit 装配→ListDir 探针=沙箱（14 真实会话零出现）；沙箱内 /plan on 实弹确认「pre-turn Notice 不可见」=真实缺陷（藏在过程折叠卡，候可见 chip 小刀）；/plan off 正常；清场=删沙箱目录。**诊断坑三连**：①壳侧 gaeaLoadConfig 不走 config.Load，开关双入口必须同调 ApplyWalkthroughOverride ②装配前 ga.cfg=nil 探针全假 ③CDP 探针可打在僵尸旧实例（gaea-v4.418.0.exe≠gaea.exe 按名漏杀）——跨实例诊断先对 PID。
- **产物**：见 SHA256SUMS-v4.418.0.txt（冒烟 200 过）；保留策略删 v4.414.0.exe。
- **文档**：releases/v4.418.0.md+CHANGELOG/README+releases/README（442→443+裁 v4.387）+AGENTS 走查配方增补+progress。

## 非版本刀（2026-09-26）真机走查尝试+污染预防实录（零代码；v4.417.0 壳）

- **目的**：真机走查池三项（审批卡可视确认/pre-turn Notice 可见性/whisper 实数据观感）。
- **实况**：v4.417.0 壳+CDP 9333 附着。GaeaNewSession 诚实报错实弹复验通过（引擎未装配→「先发送一条消息完成引擎装配」，v4.414.1 根修二在产验证）。首条消息触发引擎自动装配——**工作区解析到用户真实工作区 C:AIangong黄甲（14 真实会话），自动恢复语义把回合接到用户最新真实会话 20260924-111332 上**（v4.414.1 事故同路径：诚实报错拦不住首条消息自动装配）。所幸模型调用即失败（xAI OIDC token 过期：获取 OIDC 端点失败），回合未持久化。
- **清场（按 RewindLog 同口径）**：杀壳→引擎日志截回 seq≤469（移除 8 行，与备份前缀比对通过；备份 .tmp/backup-walk417/）→三重复核：主 jsonl 26 行与走查前一致/checkpoint 26 条全为用户真实内容（写在 user_message 之前）/state 语义不变（仅 updatedAt 刷新）。**用户可见残留=0**。
- **实得**：①whisper 实数据观感 ✅（165 条事实+12 情节，v4.416 情绪色点实数据渲染正常：时间线节点色点+标签、未知情绪键回退 glow 无布局破损）②v4.414.1 根修二（诚实报错）真机在产复验 ✅③**审批卡可视确认/pre-turn Notice 可见性=阻断于 xAI 登录过期（需用户重新登录），维持挂池**。
- **坑入档（高优先）**：真机走查前必须先完成引擎装配并核 GaeaListSessions 基线（基线在装配后取才有意义——装配前 workspace 未解析恒 0）；「GaeaNewSession 建专用会话再发消息」的隔离方案对首条消息无效（引擎未装配时 NewSession 必然诚实报错，而首条消息走自动恢复）。**候产品刀**：诊断/走查用 workspace 覆盖开关（如 GAEA_WALKTHROUGH=1 强制空隔离工作区），让真机走查与用户数据彻底解耦。

## 非版本刀（2026-09-26）UI 走查收官：全板块×明暗补走查（零代码，零缺陷）

- v4.415~v4.417 三批发版后的验证轮：vite mock+无头 Edge 隔离目检把首轮未覆盖的板块补齐——schedule（mock 实载示例工程 15 任务，斑马分组/前锋线/关键红亮暗两态均可读）、sin 原罪（mock 故事+占位插图+创作画板）、code 编程（服务管理+前置条件清单）三板块×明暗首走，其余板块亮色复走。
- **结论：零新增缺陷**。schedule 亮色下 v4.415 的 --sched-* 令牌（斑马分组六带/deadline/front）实证生效；sin/code 亮色无对比度问题；原罪故事卡的 v4.415 aria-label 删除按钮在位。mock 插图占位画与登录门属预期。
- 至此「优化完善前端UI」线程收官：静态审计可捞的用户可见缺陷已见底；剩余候选（间距栅格收敛/var 回退式清理/圆角随密度化）均为低收益高搅动，重启条件已在案（v4.417.0 release note）。
- 真机走查池（需引擎/数据窗口，与 mock 目检互补）：审批卡路由漂移可视确认、pre-turn 动词 Notice 可见性、whisper 数据面情绪色点实数据观感。

## 最新发布：v4.417.0（2026-09-26）「前端 UI 完善批三：硬编码色值收口 + 可访问性补尾」

- **刀型**：接 v4.415/v4.416 观察池。前端 6 文件，绑定面 720 零变更，Go 零改动，零功能删除。
- **色值收口**：ContextView 趋势图例裸 hex（亮态绿白底 2:1）→语义令牌；WhisperTracePanel MiniBar 四色→primary/success/warning/--whisper-accent；WhisperEmotionPanel 六维条→--whisper-accent。
- **可访问性补尾**：ChapterEditor 右键菜单键盘可达（role=menu/menuitem+聚焦+Esc+↑↓）；MemoryHub 计数「…」加载占位三态齐整；CharacterLibEditor Modal aria-label。
- **证伪降级**：圆角令牌化专项前提不成立（--md-sys-radius-* 明暗恒 8/12/16/28px，迁移零收益）；RelationGraph/GraphView/TisorRadar hex=SVG/canvas 字面量域豁免在案。
- **门禁**：定向 4 套件 80 例绿+tsc 0+eslint 0；全量 ci 绿+漂移闸 OK@4.417.0。
- **产物**：见 SHA256SUMS-v4.417.0.txt（冒烟 200 过）；保留策略删 v4.413.0.exe。
- **文档**：releases/v4.417.0.md+CHANGELOG/README+releases/README（441→442+裁 v4.386）+AGENTS 迁 1 插 1（一百三十五迁）。

## 最新发布：v4.416.0（2026-09-26）「前端 UI 完善批二：emoji 图标红线清零 + schedule 交互反馈补漏」

- **刀型**：接 v4.415.0 观察池头名。前端 19 文件，绑定面 720 零变更，Go 零改动，零功能删除。
- **emoji 清零（19 site）**：whisper 四文件（DesirePanel 分类 emoji→antd 组件；MemoryLibrary 25 情绪 emoji→效价色点 EMOTION_TONES+EmotionDot，全语义令牌；TracePanel 🤫✍️🔧📊→AudioMuted/CheckCircle/Tool/LineChart；MemoryModal 🧠⭐💫📊→Database/文本/删/BarChart；VALENCE_LABEL 按「内容语义」先例保留）；零散 11 文件（🎤→AudioOutlined、🛠️→ToolOutlined、StatsSection ⚠→WarningOutlined，其余装饰字符删除/「工具×N」文本化）。
- **schedule 补漏**：XML 导入 xmlBusy 合并按钮 loading；示例工程 Popconfirm 防呆（与「清空」同款）。
- **门禁**：定向 7 套件绿+tsc 0+eslint 0；全量 ci 绿+漂移闸 OK@4.416.0。
- **产物**：见 SHA256SUMS-v4.416.0.txt（冒烟 200 过）；保留策略删 v4.412.0.exe。
- **文档**：releases/v4.416.0.md+CHANGELOG/README+releases/README（440→441+裁 v4.385）+AGENTS 迁 1 插 1（一百三十四迁）。

## 最新发布：v4.415.0（2026-09-26）「前端 UI 完善批：绘梦整页崩根修 + 走查实证六项修复」

- **刀型**：「优化完善系统的前端UI」整批：vite mock+无头 Edge 隔离目检（9 板块×明暗截图）+静态审计（Explore 全量扫描）双证据定刀。前端 15 文件+2 新测试文件，绑定面 720 零变更，Go 零改动，零功能删除。
- **根修（P0）**：绘梦整页崩（`Cannot read properties of undefined (reading 'toFixed')`）——`GetSystemStats` 契约是 `Partial<SystemStats>`（facade 类型即如此）而 api 层零归一化直透，ControlPanel `memUsed.toFixed()` 字段缺失即抛 TypeError 连坐整页。根修=api 缝合处 statNum 归一化（缺字段补 0/非 string gpuName 收空串/null→null），消费方零改动。测试 `api/image.test.ts` 五例。
- **走查实证修复**：①chat 欢迎屏建议卡被悬浮输入区工具行遮挡→`.chat-welcome` 底部预留 176px（明暗双主题截图复验；`.chat-empty` 是载入态溅屏勿动）②CharacterPage 读取失败静默仅 console→诚实错误态+重试（HomePage v4.349 口径，+2 测试）③schedule 亮色拖拽虚线/前锋线点白系不可见→on-surface 令牌+缩放手柄白芯深边双保险+竣工/前锋线收进 `--sched-deadline/--sched-front` 令牌（暗档原值零变化/亮档加深）④MemoryHub 检索命中标签 tailwind pastel 亮主题近不可见→hub.css `.hub-hit-tag--*` 领域色两档锁定+错误态 `--md-sys-color-destructive`⑤角色卡「白墨」浅海报白名对比不足→`.ccard-id` 黑纱中段 0.42→0.55⑥aria 包：ToolbarButton 组件级 ariaLabel 缺省回落 title（五处消费点自动达标）+灯箱上一张/下一张、sin 删除故事、chat 发送、角色卡九处+ModelCenter aria-selected→aria-pressed。
- **审计证伪关账**：`--radius-*` 「未定义」不成立（App.tsx 早已内联注入，live DOM getComputedStyle 实测 `--radius-md`=12px 与 `--md-sys-radius-md` 同值）；modelcenter FEATURES emoji 字段=零渲染点死数据（BindSection 已走 FEATURE_ICONS antd）删字段了账；sin 页头 `.sin-mark` 本就是 17px 细条非重复大标题。
- **观察池新挂**：whisper 子系统 emoji 图标群（WhisperDesirePanel/WhisperMemoryLibrary/WhisperTracePanel/WhisperMemoryModal 4 文件专项）；内联 borderRadius 222 处+CSS px 圆角 291 处令牌化（批替换专项）；间距 6/10/14 非栅格收敛；ChapterEditor 右键菜单键盘可达（可点 div 无 role/tabIndex）；SchedulePage 示例工程覆盖无确认（Popconfirm 候选）；CharacterLibEditor 大 Modal 可访问名复核。
- **门禁**：定向 10 套件全绿（新增 7 例）+tsc 0+eslint 0；全量 ci 绿+漂移闸 OK@4.415.0；走查隔离目检不动用户壳与会话（v4.414.1 教训执行），console 0 新增错误。
- **产物**：见 SHA256SUMS-v4.415.0.txt（仅本地；冒烟 200 过）；保留策略删 v4.411.0.exe。
- **文档**：releases/v4.415.0.md+CHANGELOG/README+releases/README（439→440+34 席插 v4.415 裁 v4.384+清历史孤儿碎片行）+AGENTS 迁 1 插 1（一百三十三迁：v4.411.0 入 archive，头部补记 130~132 迁流水）。

## 最新发布：v4.414.1（2026-09-26）「桌面 Send 斜杠派发根修 + GaeaNewSession 诚实报错（真机走查实录）」

- **刀型**：v4.414 plan-mode 真机走查拽出双缺陷根修（v4.405.1 先例）。Go 3 文件改+1 新测试，绑定面 720 零变更。
- **根修一（长期缺口）**：桌面 Send 绕过全部斜杠动词——桥接 `app.Submit→GaeaSend→Controller.Send`，而动词面（/plan /compact /dream /memories /goal /perm /distill /new + `#<note>` 快记 + /mcp__ prompt + 自定义命令 + unknown 告知）只活在 `Controller.Submit`（HTTP 路径）；桌面主界面打任何斜杠动词都被当文本开回合发给模型。根修=动词面抽 `dispatchSlash(trimmed) bool`，Submit/SendWithRaw 共用。测试 TestSendDispatchesSlashVerbs 四段断言。
- **根修二**：GaeaNewSession 引擎未初始化静默 no-op（假成功）→诚实报错；前端 catch 有可见提示（"新建失败不重置界面"）。
- **plan-mode 实弹全通**：/plan on 派发（Notice 上 wire）→任务轮（模型只读研究，缓存命中 20096 tok）→模型真调 exit_plan_mode→ask 事件（计划审批 payload 正确）→桥接批准→批准叙事落历史→/plan status=关。console 全程 0。
- **用户会话污染事故与复原（透明记录）**：走查回合写进用户真实会话（黄甲 0924）——根因=走查脚本在引擎初始化前调 GaeaNewSession（静默 no-op）+旧 exe 把 /plan on 当文本开回合。复原=杀壳离线手术按 RewindLog 同口径（事件日志截 seq≤469+镜像截 4 user 前+删检查点+state 摘要修正），重启验证 39 条历史零走查痕迹、末条=用户真实定稿、console 0；四源文件备份 .tmp/backup-*。全程只读（计划模式纪律生效）。教训=走查会话隔离必须落盘双确认，静默 no-op 会骗人。
- **产物**：exe 见 SHA256SUMS-v4.414.1.txt；保留策略留 v4.411~v4.414.1 删 v4.410.0.exe。
- **观察池新项**：审批卡渲染路由漂移下未可视确认（ask payload 正确，渲染留复走）；pre-turn 动词 Notice 上 wire 但办公流不可见。
- **坑**：①Windows 文件锁——RewindLog 的 AtomicWrite rename 对活壳自身句柄退避×3 仍败，离线手术须先杀壳②taskkill /F 后 rmSync 有句柄释放竞态（检查点复活假象），删除后必须 existsSync 复核③node -e 内联脚本反斜杠路径会被吃——走查脚本路径用正斜杠。

## 最新发布：v4.414.0（2026-09-26）「Plan mode 计划审批门（dsh plan-mode 蒸馏）」

- **刀型**：DSH 留池⑦（实质头名：①子代理控制面等上游稳定）。蒸馏=取道不取器：471 行上游 → 内核 1 新文件（agent/plan_mode.go）+2 挂点（executeOne 盖章/boot 注册）+1 斜杠动词（/plan）。绑定面 720 零变更，前端零改动。规格 进度计划/gaea-plan-mode-20260926.md。
- **落地**：①exit_plan_mode 恒注册（工具目录跨模式切换稳定）：计划模式下 markdown 计划（# 标题强制）提交 ask 通道审批——批准=闸翻转 off+叙事落历史/继续计划=错误带回反馈/headless 无 Asker=诚实拒绝（执行闸必须人来开，刻意不仿 ask 自选降级）②PlanGate 窄接口经 callContext 盖章（asker 同位同纪律）+SubagentMetaTools 排除子代理③/plan on|off|status（on 注入 `[plan mode]` 政策 user 消息/off 退出叙事/status 回落，running 拒绝）④与上游两偏离：**系统前缀冻结（V10.36）→政策走 user 消息注入**（mid-session 改 system 段必破缓存，技能目录热重载同判先例）；**状态 v1=runner 内存态**（事件日志 kind=plan_mode+恢复折叠留二刀，最敏感面零碰）；沙箱/权限独立不做硬工具门（与上游同判）。
- **测试**：Go plan_mode_test 六例（inactive 错误/批准翻转+叙事/继续计划语义/headless 拒绝/标题校验含六级合法七级拒/真闸直测含前缀不动+两条注入断言）；agent/control/boot 三包回归绿；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.414.0.txt；保留策略留 v4.410~v4.414 删 v4.409.0.exe。
- **文档**：规格+releases/v4.414.0.md+CHANGELOG/README+releases/README（437→438+裁 v4.382）+AGENTS 迁 1 插 1（一百三十一迁：v4.409.0 入 archive）+todos（dsh 留池⑦清账，余①子代理控制面=等上游稳定）。
- **坑**：①工具契约=Execute(ctx, json.RawMessage)+ReadOnly() bool（跟 string 签名走会炸 reg.Add）②包内测试桩撞名先查（stubGate 已被 gate_test.go 权限桩占用）③双引号 Go 串里的 \n 是真换行——拼 JSON 断言夹具须字面 \\n。

## 非版本刀：sin 书源一条龙实弹+常驻内存采集（2026-09-26，零代码）

- **走查面**：书源一条龙观察池收官——真机 v4.413.0 壳+假站 walk_server.py(18099)+临时规则 zz-walkthrough-fake.json，桥接 SinB 全链（Search/Toc/Download/BooksList/BookExportEpub/BookDelete）+UI 书源卡+底稿直注+工具链+导出，全程 console 零新增错误。
- **SSRF 护栏实弹实证（头条发现）**：书源通道 netclient.GuardedClient 对假站 127.0.0.1 报 `refusing to fetch internal address`——v4.374 设计注释明说「书源=内网全禁，防提示注入的规则摸到本机服务」；**假站成功链路真机结构性不可行=护栏按设计工作**，成功路径由 Go 测试注入 Fetcher 覆盖（fetch.go Options.Fetcher）。实测三态全诚实：聚合搜索 total=0 静默+UI 卡片逐源显示失败原因（「书源「zz-walkthrough-fake」搜索失败: …refusing to fetch…」）+Toc 直接抛同错误；下载任务全章失败→成书清单诚实空（「还没有成书」）。
- **底稿直注实写过**：SinNotesSave(baseline 底稿)→SinStream 轮（本地 herdsman Qwen3.6-35B）→上下文逐轮 `inject=45` token 注入段非零（system 919+inject 45+assistant 50），助手文案带注入特征（灰蓝长衫/短须）——v4.412 上下文口径真机自洽。
- **工具链实弹过**：第二轮明示「用 book_search 工具搜索」→本地模型真调工具（轨迹第 3 轮「3 条记录 1 工具调用」UI 在位）+诚实回复「没有搜到候选…另外 1 条来源失败被略过」（即护栏拒假站）。
- **导出双形态过**：SinExportMarkdown 全文（user/assistant 对+落款头）+SinExportEpub 落 `%APPDATA%\gaea\sin\exports\*.epub`；轨迹页签 Duration 77s/Turn 3/Call 1 渲染正常。
- **清场零残留**：临时故事/角色（remaining=0）/规则文件/导出 EPUB 全删，书架空，rules 目录复原（模板+engines），壳+webview+假站杀净。**零碰 novelsDir**（书源卡下载自包含 sin 书架，与小说导入是两条链——本次链路勘误入档）。
- **常驻内存首采**：v4.413.0 壳启动闲置态 WS=223MB/Private=235MB（slim-masterplan §3 已回填，P5 表「未采」全清）。
- **观察池新项**：reload 后偶现 `Callback 'app.ModelB.GetModelMonitor-…' not registered!!!` console 告警×2——Wails 回调重注册竞态（轮询先于注册），无功能损害，候小刀（重注册前静默或幂等重放）。
- **坑**：①SinStream 是异步流——返回流 ID（`ss_…`）即返回，消息经事件落库，勿当同步结果断言 ②SinMessages 空话题返回 null（非空数组），判空再取 length ③书源规则目录=$APPDATA/gaea/booksource/rules（模板+引擎+用户规则混装，搜索时逐文件装载）。

## 最新发布：v4.413.0（2026-09-26）「良性取消日志降级（前端错误观测通道降噪）」

- **刀型**：观察池项落地（v4.412 sin 实弹走查收官时新挂）。Go 1 文件+1 测试，绑定面 720 零变更，版本三处 4.413.0（sync-version.ps1）。
- **落地**：GaeaLogFrontendError 对 message 含 `context canceled` 降级 slog.Info，其余照旧 slog.Error——良性取消不再污染 Error 级观测通道，审计痕迹保留（INFO 在册）；剧照/设定卡/绘梦三条取消链路同时降噪；前端零改动（proxy.ts invoke 统一入口语义不变：仍上报、仍原拒绝语义抛出）。
- **测试**：Go TestGaeaLogFrontendError_CancelDowngrade（slog 捕获两例：取消链 INFO/真错 ERROR，复用 captureLogs 基建）；全量 ci.ps1 绿 EXIT=0+绑定漂移闸 OK（720 方法）+版本漂移闸 OK@4.413.0。
- **产物**：exe 51339264B SHA256=5d9b4b6ba2ce43c32e0d1e5d9f0a5dc473af1960b5c7eed6f7096f61cb7c2d24（SHA256SUMS-v4.413.0.txt，仅本地；build.bat 冒烟 200 过+桌面副本）；保留策略留 v4.409~v4.413 删 v4.408.0.exe。
- **文档**：releases/v4.413.0.md+CHANGELOG/README+releases/README（436→437+34 席插 v4.413 裁 v4.381）+AGENTS 迁 1 插 1（一百三十迁：v4.408.0 入 archive）+todos（观察池「取消降级」清账）。
- **顺带 P5 冷启动补采**：Go 同步链 43–110ms（十二样本中位 ~62ms，charlib+stores 40–47ms 恒大头）+前端 gaea:interactive=497ms（v4.413.0 壳 CDP 实测：gaea:boot 91ms→interactive 588ms，DCL 139ms）；常驻内存仍挂真机窗口——slim-masterplan §3 已回填。
- **坑**：build.bat 是批处理——`powershell -File build.bat` 会 EXIT=127（旧 exe 原样未动），须 `cmd //c build.bat`。

## 非版本刀：sin 实弹走查收官+v4.412 三页签真机（2026-09-25/26，ComfyUI 在跑窗口）

- **窗口捕获**：ComfyUI 8188 探活 200+队列空而壳未运行→挂池最贵的「真机实弹走查」当场解锁。CDP 9333 附着 v4.412 壳（WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS 配方），临时角色 lib_1790352026650（64×64 canvas dataURL 参考图）+临时故事 sin_1790352026653_2+SinCastSet 挂 cast——用户真实故事/角色/配置全程零触碰，走查后临时数据全删（CharacterList 查 w413=0、架子只剩用户故事）。
- **通过①（v4.412 三页签真机）**：中栏「故事/轨迹/上下文」真机渲染——轨迹诚实空态（暂无轨迹记录+Duration/Turn/Call 0）+上下文 hero「当前上下文 0/0 token —」六段估算+趋势+Token 统计全在位；console.error=0/pageError=0（页签切换全程）。坑=ChatTabs 页签钮不带 role=tab，选择器走 `.sin-tabs button`；SinCastSet 第二参直传数组（JSON.stringify 会踩 json: cannot unmarshal string into []string）。
- **通过②（v4.408 未验项清账）**：chip「生成 走查临时角色-w413 的设定卡」实弹提交→ComfyUI 队列 running:1 实达；进度行实生成态「加载模型 · 已用时 2s+取消钮」真机在位（截图 walk-v4413-progress.png）；点取消→行消失+「已取消生成」info 语义 toast（非错误样式，walk-v4413-after-cancel.png）+ComfyUI 队列秒清（interrupt 双达实证）+壳日志 context canceled 全链传播；临时角色 referenceImages 保持 1 张=取消先于完成零写回。
- **诚实闸验证（意外收获）**：portrait_backend=xai 时设定卡点击→「设定卡生成（Qwen 参考编辑）当前仅 ComfyUI 本地档（当前后端：xai）」fail-fast，console 仍零污染（v4.410 结论真机复现）。
- **观察池新项**：①取消路径全局前端错误钩记 ERROR 级（[frontend] [CharacterGenerateSheetError]…context canceled）——UI 已语义化但错误日志通道被良性取消污染，候小刀（按 context canceled 降级）②走查期发现 bridge 平面 API 形状坑（NovelB.SaveCharacter 是项目角色须先开项目；全局库=CharacterSave 返回带 id 全量）已沉淀记忆。
- **配置零漂移**：comfyui 本就是用户全局生图后端（09-24 启动日志「图片后端: ComfyUI」实证），走查中 image_backend 重写=原值；portrait_backend xai→comfyui→xai 精确还原；运行时配置真身在 ~/.gaea_config.json（config.toml 是 agent 配置，别看错文件）。壳杀净（gaea.exe+webview 过滤），ComfyUI 保持用户原状。
- **顺带 P5 季度复测落档**：entry 755→970KB(+28%)/dist 10.0→11.1MB/exe 46.15→51.44MB——回涨属功能比例不立项刀；Go >50KB 0→4 破防挂观察池（详见 slim-masterplan §3）。

## 源码包推送：wubigork-v4.411.0（2026-09-25）

- wubigork-v4.411.0-source.tar.gz @20b7204f 排除旧源码包，12951822 B / 4393 files / SHA256=248fd3bc…502ccb（SHA256SUMS-v4.411.0.txt 登记）。覆盖 v4.400~v4.411 十二版；随包推送 24 commit + v4.400.0~v4.411.0 14 tag 上 remote（origin/main 对齐，余 0 领先）。
## 最新发布：v4.411.0（2026-09-25）「一致性评分写回元数据」

- **v4.412.0** 上下文页 UI 优化（hero 提位/窗口未知「—」/趋势 hover）+ 原罪复用轨迹/上下文页签（sin_insight 折叠三绑定 717→720，TrajectoryView/ContextView 原组件自定义数据源）
- **刀型**：编辑器评分历史池①。绑定面 717 零变更，characterlib 迁移补 reference_scores 列。
- **落地**：reference_scores 与参考图索引对齐（读写两路维护+读侧补 0）；编辑器评分写 form 保存持久化+缩略图分数徽标（≥80 绿/60-79 橙/<60 红）；移除同下标裁剪。
- **测试**：Go 往返/对齐/短表三例；前端徽标/持久化/移除 35 例；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.411.0.txt；保留策略留 v4.407.0~v4.411 删 v4.406.0.exe。
- **观察池**：图vs图评分（视觉运行时验证门）；自动重生成（GPU 开销拍板）。
## 非版本刀：v4.408~v4.410 sin 新面真机走查（2026-09-25）

- CDP 9333 附着 v4.410 壳（WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS 配方）。**通过**：chip「评分 gaea 的参考图」（v4.410）与「生成 gaea 的设定卡」真机在位；评分空态诚实指路（暂无参考图→不用视觉模型）；设定卡非 comfyui 后端闸诚实报错（当前后端 xai→拒绝，fail-fast）；进度行/取消钮空闲态不渲染（闸正确）；单会话 console.error=0/pageError=0。**未验项**：进度行实生成态+取消交互（ComfyUI 已关，等其在跑窗口）。临时故事桥接清场，用户真实故事零触碰。截图 .tmp/walk-v4410.png。
## 非版本刀：whisper 域死代码刀4（2026-09-25）

- 整文件死 88 文件 9,920 行净删（基线 590→196），deadcode -test 方法论三教训（type 依赖/app 反射盲区/依赖闭包剥壳）落档；12f77401。
## 非版本刀：whisper 域死代码调研档（2026-09-25，拍板池）

- deadcode -test 基线：590 不可达 func/147 文件/估 6-8k 行，ackem 蒸馏标记 267 处；五簇分解与三方案（整域/只删 A 簇/维持保留）落 docs/gaea-whisper-deadcode-survey-20260925.md，待拍板。
## 最新发布：v4.410.0（2026-09-25）「sin 评分入口 v1（快路径）+ 评分流式辨伪关账」

- **刀型**：观察池末两项收官——等待体验/一致性线程观察池全部关账。纯前端，绑定面 717 零变更。规格 进度计划/gaea-sin-score-entry-20260925.md。
- **落地**：chip「评分」钮评角色最新参考图（getCharacter→末张→scoreCharacterConsistency→编辑器同款 Modal，<60 补参考提示）；scoreId 与 genId 互斥；评分流式辨伪缓办（JSON 契约无增益+churn 大，重开条件=图vs图 V2）。
- **测试**：SinCastPanel 9→12 例（互斥双向在案）；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.410.0.txt；保留策略留 v4.406.0~v4.410 删 v4.405.1.exe。
- **观察池**：（线程关账）编辑器侧历史池（图vs图/写回元数据/自动重生成）维持挂起待拍板。
## 最新发布：v4.409.0（2026-09-25）「图像生成内存压力预检（iGPU 冷载预期管理）」

- **刀型**：观察池在册项（v4.404.1 起四版）。Go 1 文件+3 挂点+前端 1 util+App 全局订阅，绑定面 717 零变更。规格 进度计划/gaea-imagegen-pressure-20260925.md。
- **落地**：comfyui 提交口读系统内存（复用既有 getMemoryStats 原生采集），可用 < 15% 或 < 4GB 命中→WARN+imagegen:pressure 事件；前端 App 全局订阅+2 分钟节流提示；只提示不拦截。
- **测试**：Go 阈值六态+读数合理性+接线三态（seam+httpbridge SSE 捕获）；前端 3 例；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.409.0.txt；保留策略留 v4.405.1~v4.409 删 v4.404.1.exe。
- **观察池**：评分 LLM 流式；sin 评分入口。
## 最新发布：v4.408.0（2026-09-25）「sin 设定卡 chip 进度行+取消钮」

- **刀型**：图像等待体验线程 sin 侧收尾（v4.407 观察池头名「sin 面板取消钮」+v4.406「sin 进度行」）。纯前端，绑定面 717 零变更。规格 进度计划/gaea-sin-cast-progress-cancel-20260925.md。
- **落地**：SinCastPanel 生成期 1s 轮询同源快照，chips 下方进度行（节点中文+用时/排队中，复用 COMFY_NODE_LABELS）+行尾取消钮（设定卡链 v4.407 起全局可取消，sin 复用自动获得）；handleCastSheet 的 context canceled 语义化「已取消生成」。
- **测试**：SinCastPanel 3→6 例（进度行出现+取消调用+结束消失/排队文案/读取失败不阻断+既有回归）；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.408.0.txt；保留策略留 v4.404.1~v4.408 删 v4.401.0.exe。
- **观察池**：评分 LLM 流式；iGPU 压力预检；sin 评分入口。
## 最新发布：v4.407.0（2026-09-24）「角色库生成取消（剧照/设定卡可中止）」

- **刀型**：图像等待体验线程观察池头名。纯接线，绑定面 717 零变更。规格 进度计划/gaea-charlib-cancel-20260924.md。
- **落地**：剧照/设定卡两链接入 beginImageGen/endImageGen（ctx+/interrupt 双达，修 v4.406 无条件 clear 缺陷）+resetComfyCancel；前端进度行取消钮+「已取消生成」语义化。
- **测试**：TestCharacterGenerateSheetCancel（挂起模拟冷载→取消/interrupt 恰一次/可再提交）+全链回归+前端两态；全量 ci 绿 399 文件 3377 例。
- **产物**：exe 见 SHA256SUMS-v4.407.0.txt；保留策略留 v4.404.1~v4.407 删 v4.405.0.exe（桌面副本待壳关闭补拷）。
- **观察池**：sin 取消钮；评分流式；iGPU 压力预检。
## 最新发布：v4.406.0（2026-09-23）「角色库生成进度接线（剧照/设定卡载入可见）」

- **刀型**：图像等待体验线程收尾。纯接线，绑定面 717 零变更。规格 进度计划/gaea-charlib-progress-20260923.md。
- **落地**：attachComfyProgress（comfyui 预置 queued+挂回调，剧照/设定卡同链+defer 清理）；编辑器忙碌期 1s 轮询快照+hero 进度行（复用节点标签表）；相对路径写入源清点关账（@mention 相对=设计）。
- **测试**：TestAttachComfyProgress 三态+全链回归+前端两态；全量 ci 绿 399 文件 3375 例。
- **产物**：exe 见 SHA256SUMS-v4.406.0.txt；保留策略留 v4.404.1~v4.406 删 v4.404.0.exe。
- **观察池**：角色库生成取消（接 beginImageGen）；sin 进度行；评分流式。
## 最新发布：v4.405.1（2026-09-23）「AttachmentDataURL 相对路径根修（真机日志实录）」

- **分诊**：真机日志 18:49 实录——可读根检查只认绝对路径，工作区相对形态被整类误拒，文件真实存在。
- **落地**：resolveAgainstWorkspace（相对按工作区根 Join，同 GaeaListDir 惯例）+读侧解析后 ReadFile+写侧同款；穿越仍拒。纯 Go 1 文件，绑定面 717 零变更。
- **测试**：TestAttachmentDataURLRelativePath（放行+端到端+穿越拒+写侧）；app 全包+全量 ci 绿 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.405.1.txt；保留策略留 v4.401~v4.405.1 删 v4.403.0.exe。
- **观察池**：相对路径来源面清点——写入侧统一绝对化才是根治。
## 最新发布：v4.405.0（2026-09-23）「WS 载入阶段进度可读化（排队/载入可见）」

- **刀型**：v4.404.1 分诊后续（20 分钟冷载界面零反馈）。纯 Go 1 文件+前端 1 标签，绑定面 717 零变更。规格 进度计划/gaea-ws-load-progress-20260923.md。
- **落地**：WS 补 executing（节点开始→「当前节点: 加载模型」+不定态）/status（排队可见）/progress_state max=0 兜底三类事件；中文由前端既有节点标签表承担。
- **测试**：gorilla 假 WS 脚本序列回调断言（含两负例）；ai 全包+imagegen 128 例；全量 ci 绿 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.405.0.txt；保留策略留 v4.401~v4.405 删 v4.402.0.exe。
- **观察池**：ComfyUI 载入百分比真实可得性；iGPU 压力预检；取消显性化。
## 最新发布：v4.404.1（2026-09-23）「ComfyUI 生成等待根修：上限 15→30 分钟 + 超时收割（真机实录）」

- **分诊**：用户真机「图生图失败」=客户端白等误报——服务端 20.5 分钟冷载后成功，15 分钟上限先到；产物实际已生成。根因=iGPU 共享内存权重被换出后重载慢（同 GPU 语音管道刚停）。
- **落地**：genTimeout 15→30 分钟+harvestOnTimeout 超时收割（服务端已完成则下载返回）。纯 Go 1 文件，绑定面 717 零变更。
- **测试**：TestHarvestOnTimeout 两态；全量 ci 绿 399 文件 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.404.1.txt；保留策略留 v4.401~v4.404.1 删 v4.400.0.exe（桌面副本待壳关闭补拷）。
- **观察池**：WS 载入阶段进度可读性；取消显性化；iGPU 压力预检。
## 最新发布：v4.404.0（2026-09-23）「角色一致性评分 v1（文字锚点，T2 一致性收口刀）」

- **刀型**：用户拍板解锁挂池项。绑定面 716→717（play）、spaceBindings 540。规格 进度计划/gaea-consistency-score-20260923.md。
- **落地**：视觉 LLM 图 vs 文字设定评分（0-100+总评+issues，复用 RecognizeImage 单图链）；data URL 临时文件链+解析钳位+坏回复带原文报错；编辑器参考图「评分」钮+Modal+<60 补参考提示+单飞。
- **测试**：Go 纯函数+解析七态+seam 全链；前端两态；全量 ci 绿 399 文件 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.404.0.txt；保留策略留 v4.400~v4.404 删 v4.399.0.exe。
- **文档**：releases/v4.404.0.md+CHANGELOG/README+releases/README（425→426+裁 v4.370）+AGENTS 迁 1 插 1（一百一十九迁：v4.399 入 archive）+todos。
- **观察池**：图 vs 参考图评分（多图验证）；sin 评分入口；评分写回元数据；自动重生成。
## 最新发布：v4.403.0（2026-09-23）「sin 侧设定卡入口（T2 角色资产 v2 收尾刀）」

- **刀型**：观察池在册项。纯前端编排，绑定面 716 零变更、spaceBindings 539。规格 进度计划/gaea-sin-sheet-entry-20260923.md。
- **落地**：原罪右栏角色 chip 设定卡钮（单击 triptych 快路径）→getCharacter 全量→qedit→追加 referenceImages→saveCharacter 自动存回→reloadLibrary；chip 级单飞；面板吞错页面层 message；prop 可选向后兼容。
- **测试**：SinCastPanel 3 例+SidePanel 夹具补 prop；全量 ci 绿 399 文件 3372 例（首轮 ProgrammingPage 负载 flaky 隔离绿+复跑全绿在案）。
- **产物**：exe 见 SHA256SUMS-v4.403.0.txt；保留策略留 v4.399~v4.403 删 v4.398.0.exe。
- **文档**：releases/v4.403.0.md+CHANGELOG/README+releases/README（424→425+裁 v4.369）+AGENTS 迁 1 插 1（一百一十八迁：v4.398 入 archive）+todos。
- **观察池**：一致性评分（vision 待拍板）；chip 模板菜单；分张并发档；评分驱动补参考。
## 最新发布：v4.402.0（2026-09-23）「设定卡分张连发（T2 角色资产 v2 第三刀）」

- **刀型**：v4.401 观察池头名。纯前端编排，绑定面 716 零变更、spaceBindings 539。规格 进度计划/gaea-char-sheet-split3-20260923.md。
- **落地**：菜单「三视图分张（连发）」——正/侧/背串行排队逐张追加进参考图（即到即见+N/3 进度）；失败语义=首张失败即中止（系统性故障）/中途失败继续+warning 点名单张重试；sheetGuard 收敛共用。
- **测试**：前端 3 例（全成功 variant 序/首张中止只调 1 次/中途失败 3 张都发起+点名）；全量 ci 绿 398 文件 3369 例。
- **产物**：exe 见 SHA256SUMS-v4.402.0.txt；保留策略留 v4.398~v4.402 删 v4.397.0.exe。
- **文档**：releases/v4.402.0.md+CHANGELOG/README+releases/README（423→424+裁 v4.368）+AGENTS 迁 1 插 1（一百一十七迁：v4.397 入 archive）+todos。
- **观察池**：一致性评分（vision 待拍板）；sin 侧设定卡入口；分张并发档；评分驱动补参考。
## 最新发布：v4.401.0（2026-09-23）「设定卡模板矩阵 + 多参考锚定（T2 角色资产 v2 第二刀）」

- **刀型**：v4.400 观察池头名（多姿势/分张）+qedit 三图槽自然消费。签名扩展 (chJSON, variant)，绑定面 716 零变更、spaceBindings 539。规格 进度计划/gaea-char-sheet-v2-20260923.md。
- **落地**：charSheetVariantBody 六模板（triptych 默认/front·side·back 分张/sitting/action）+多参考锚定（≤3 填满 qedit 三图槽，剧照兜底，单张失败跳过）+前端 Dropdown.Button 模板菜单+**顺手根修 v4.400 剧照/设定卡按钮同点位重叠**（.cd-hero-gens 堆叠容器）。
- **测试**：Go 纯函数矩阵+假 ComfyUI（2 可用 1 缺失→上传恰 2 次/variant 透传/非法报错/两态回归）+前端主按钮默认 triptych+菜单坐姿两态；全量 ci 绿 398 文件 3367 例。
- **坑**：NodeList 解构 tsc -b 报 TS2488 须 Array.from/Dropdown.Button 的 testid 挂 wrapper 须 querySelector 取真按钮/新按钮抄旧 class 先核对是否承担布局定位。
- **产物**：exe 见 SHA256SUMS-v4.401.0.txt；保留策略留 v4.397~v4.401 删 v4.396.0.exe。
- **文档**：releases/v4.401.0.md+CHANGELOG/README+releases/README（422→423+裁 v4.367）+AGENTS 迁 1 插 1（一百一十六迁：v4.396 入 archive；头部记录行补 105~115 漏登案）+todos。
- **观察池**：一致性评分（vision 文本锚点法待拍板）；分张一键连发（三张排队+单张失败重试）；sin 侧设定卡入口；评分驱动补参考。
## 最新发布：v4.400.0（2026-09-23）「角色设定卡生成（三视图，qedit 通道 · T2 角色资产 v2 首刀）」

- **刀型**：T2 角色资产 v2 首刀；复用 qedit+参考图画廊零 schema。绑定面 715→716（play）。规格 进度计划/gaea-char-sheet-20260923.md。
- **落地**：buildCharacterSheetPrompt+CharacterGenerateSheet（qedit 首参考锚定→三视图设定卡）+Editor 按钮（产物追加参考图列表）。
- **测试**：Go 纯函数+假 ComfyUI 全链+两态报错；前端按钮两态；全量 ci 绿。
- **坑**：buildPortraitClient 独立建客户端注入不生效须假 ComfyUI/renderEditor 包装对象传裸静默不命中/prompt 文案断言用词对齐。
- **产物**：exe 见 SHA256SUMS-v4.400.0.txt；保留策略留 v4.396~v4.400 删 v4.395.0.exe。
- **文档**：releases/v4.400.0.md+CHANGELOG/README+releases/README（421→422+裁 v4.366）+AGENTS 迁 1 插 1（一百一十五迁：v4.395 入 archive）+todos。
- **观察池**：一致性评分（vision 文本锚点法）；多姿势模板；设定卡分张；sin 侧入口；评分驱动补参考。

## 最新发布：v4.399.0（2026-09-23）「原罪插图参考升级 qedit（T2 第二消费方）」

- **刀型**：v4.398 观察池头名小刀。纯 Go 2 文件，绑定面 715 零变更。
- **落地**：sinRefPlan comfyui→(txt2img,qedit)（不判生图模型）；herdsman 维持 img2img；refFallback 链兜底权重缺失。
- **测试**：矩阵+透传+TestSin 回归绿；全量 398 文件 3365 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci 绿 EXIT=0+漂移闸 OK@4.399.0。
- **产物**：exe 见 SHA256SUMS-v4.399.0.txt；保留策略留 v4.395~v4.399 删 v4.394.0.exe。
- **文档**：releases/v4.399.0.md+CHANGELOG/README+releases/README（420→421+裁 v4.365）+AGENTS 迁 1 插 1（一百一十四迁：v4.394 入 archive）+todos。
- **观察池**：sin 参考方法可配；两级回退；徽标区分方法；qedit 语义 prompt 模板。

## 最新发布：v4.398.0（2026-09-23）「绘梦·参考槽 Qwen-Edit 通道（T2 一致性首刀）」

- **刀型**：T2 一致性首刀；路线修订（Qwen 架构走编辑参考不走 IP-Adapter）。Go 3+前端 5 文件，绑定面 715 零变更。规格 进度计划/gaea-qedit-ref-20260923.md。
- **病根**：参考槽 v0=img2img 整幅重绘（构图被锁死）+队列 txt2img 丢参考（新场景+角色一致无通道）。
- **落地**：qedit 路由+builder 三图槽（正负 TextEncode image1/2/3）+元数据如实化+ControlPanel 方法 Radio+applyRefCharacter 分流+queue 打通。
- **测试**：ai +2+回归绿+app +1+前端 +2；全量 398 文件 3365 例绿+tsc 0+eslint 0。
- **坑**：代码块搬家块尾注释找边界/regex 中插截断泛型/缺省行为是契约新方法并列露出。
- **产物**：exe 见 SHA256SUMS-v4.398.0.txt；保留策略留 v4.394~v4.398 删 v4.393.0.exe。
- **文档**：releases/v4.398.0.md+CHANGELOG/README+releases/README（419→420+裁 v4.364）+AGENTS 迁 1 插 1（一百一十三迁：v4.393 入 archive）+todos。
- **观察池**：sin 参考槽升级；云端多图编辑；角色资产 v2（三视图/一致性评分）；LoRA 向导；导演 Agent；per-image 权重。

## 最新发布：v4.397.0（2026-09-23）「绘梦·抠图/透明底导出（T3 编辑力收官）」

- **刀型**：T3 最后一件。零新模型：蒙版即 alpha。Go 3 文件+前端 5 文件，绑定面 714→715（+ImageCutout，play）。
- **落地**：ComposeCutout（白=保留主体）→ImageCutout 绑定（落盘+台账+{path,asset_id}）→CutoutModal+ResultStage「抠图」动作。
- **测试**：ai +2+app +1+前端 +3；全量 398 文件 3363 例绿+tsc 0+eslint 0。
- **坑**：bindingNames 两份（真实消费 gaea/lib 那份）/bindings_image.go 单行风格勿 gofmt -w 整文件/JSX 锚点替换嵌条件块插完必 tsc。
- **产物**：exe 见 SHA256SUMS-v4.397.0.txt；保留策略留 v4.393~v4.397 删 v4.392.0.exe。
- **文档**：releases/v4.397.0.md+CHANGELOG/README+releases/README（418→419+裁 v4.363）+AGENTS 迁 1 插 1（一百一十二迁：v4.392 入 archive）+todos。
- **观察池**：羽化；角色库参考图管线对接；模型推理抠图；透明产物展示语义；**画室线转 T2 一致性**（ipadapter/pulid、角色资产 v2、LoRA 向导、导演 Agent）。

## 最新发布：v4.396.0（2026-09-23）「纯绘梦会话台账登记根修（真机走查抓出）+ 编辑四件套真机验收班」

- **刀型**：真机验收班抓出的真缺陷根修。Go 1 文件+1 测试，绑定面 714 零变更。
- **病根**：台账登记闸=armed&&gaeaCfgSnapshot()!=nil；ga.cfg 只在办公引擎 GaeaInit 赋值——纯绘梦会话恒 nil→误判非运行态→登记静默跳过（v4.98 起：素材库恒空/变体簇断）。
- **落地**：闸只留 armed 位；回归测试钉纯绘梦会话形态。
- **验收班**：编辑四件套真机全过（详见 releases/v4.396.0.md）；本地档真图编辑 ~400s（20B 首载）。
- **坑**：国内大模型下载首选 ModelScope（33MB/s vs HF/镜像 2-5MB/s）；分段下载 curl -C - 与 -r 互斥、重试追加超长需单段重下。
- **产物**：exe 见 SHA256SUMS-v4.396.0.txt；保留策略留 v4.392~v4.396 删 v4.391.0.exe。
- **文档**：releases/v4.396.0.md+CHANGELOG/README+releases/README（417→418+裁 v4.362）+AGENTS 迁 1 插 1（一百一十一迁：v4.391 入 archive）+todos。
- **走查班挂池**：变体徽标重验、蒙版/扩图真图、走查产物清场。

## 最新发布：v4.395.0（2026-09-23）「绘梦·编辑历史变体簇：ParentID 链 + 溯源，回到任一变体继续改」

- **刀型**：长期规划 T3「编辑历史形成变体簇」；T0 契约 ParentID 预留位激活。Go 7 文件+前端 5 文件，绑定面 714 零变更。规格 进度计划/gaea-variant-chain-20260923.md。
- **病根**：编辑链 A→A'→A'' 天然成链但不可见——台账无关系字段、前端无变体标识，改三轮无法回到第 2 版分叉。
- **落地**：①sourcePath→imageHubAssetIDByPath→ParentID（查不到=诚实断链）②登记链路 +parentID（四调用点传空）③imageItem 回传 asset_id/parent_id④变体徽标+溯源按钮⑤VariantChainModal 建链+用到画布回祖先。
- **测试**：Go +2+前端 +4+2；全量 397 文件 3360 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.395.0。
- **坑**：批量 replace 命中同名 meta 构造块先数命中数/测试按包依赖挪文件/gofmt 只 -w 触碰文件。
- **产物**：exe 见 SHA256SUMS-v4.395.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.391~v4.395 删 v4.390.0.exe。
- **文档**：releases/v4.395.0.md+CHANGELOG/README+releases/README（416→417+34 席插 v4.395 裁 v4.361）+AGENTS 迁 1 插 1（一百一十迁：v4.390 入 archive）+todos。
- **观察池**：HistoryRail 簇折叠；跨空间溯源；sin 编辑链；变体簇导出；按簇删除；抠图/透明底（T3 最后一件）。

## 最新发布：v4.394.0（2026-09-23）「绘梦·扩图（outpaint）：画布加边，模型补全新区域」

- **刀型**：图像域长期规划 T3「扩图」；v4.393 观察池销号。经济路线：扩图=合成大画布+蒙版环=v4.393 的 edit+mask 形态，两后端零引擎改动。Go 3 文件+前端 3 文件，绑定面 714 零变更。规格 进度计划/gaea-outpaint-20260923.md。
- **病根**：扩图是市场标配（即梦/Nano Banana Pro/PS 生成式扩展），gaea 编辑三件已落两件但扩图缺位——横补竖构图、特写拉远景无处落。
- **落地**：①ComposeOutpaint 纯函数（四边百分比/锚点按扩展量/中性灰 128/蒙版黑白环/>4096² 拒）+OutpaintToSquare②app mode=outpaint 转换 edit 请求（三门 fail-closed/mode 回显/清空 size）③前端 Modal 三态+6 预设 chips+四边输入+虚线预览+零扩展 warning。
- **测试**：ai +3+app +1+前端 +1；全量 396 文件 3354 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.394.0。
- **坑**：夹具 RGBA 忘 A:255=透明色贴图断言必挂/补正方百分比基数是短边原长/gate 顺序互斥门在通用 mask 门之前/测试期望先口算。
- **产物**：exe 见 SHA256SUMS-v4.394.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.390~v4.394 删 v4.389.0.exe。
- **文档**：releases/v4.394.0.md+CHANGELOG/README+releases/README（415→416+34 席插 v4.394 裁 v4.360）+AGENTS 迁 1 插 1（一百零九迁：v4.389 入 archive）+todos。
- **观察池**：抠图/透明底导出（T3 最后一件）；扩边羽化；非矩形扩区（涂选已覆盖）；img2img+mask；变体簇；多图编辑；Lightning LoRA。

## 最新发布：v4.393.0（2026-09-23）「绘梦·蒙版局部重绘：涂选要改的区域，其余保持原样」

- **刀型**：图像域长期规划 T3「蒙版局部重绘」；v4.392 观察池销号第 2 项。Go 4 文件+前端 4 文件，绑定面 714 零变更。规格 进度计划/gaea-mask-inpaint-20260923.md。
- **病根**：双编辑档都有蒙版通道但从未接线——前端没有涂选交互、后端请求没有 Mask 字段；全图编辑对已满意区域有回归风险（蒙版局部重绘是 NovelAI V5/即梦/可灵全系标配）。
- **落地**：①统一蒙版契约（灰度 PNG 白=重绘黑=保留；ComfyUI ImageToMask(red) 零转换，OpenAI grayMaskToOpenAIMask 白→透明一处转换）②ComfyUI 蒙版链 LoadImage→ImageScale(lanczos,targetW/H)→ImageToMask(red)→SetLatentNoiseMask→KSampler——targetW/H 复刻 PREFERRED_KONTEXT_RESOLUTIONS 17 档（nodes_flux.py:105 实读）对齐缩放图；TextEncode image1 保持全图③app fail-closed（mask 非 edit 拒绝）④MaskBrushEditor（strokes 状态+红笔刷显示/黑底白笔刷导出同坐标+比例映射+jsdom 守卫）+Modal 范围切换+未涂 warning。
- **测试**：Go +4 组+app +1+前端 +4+2；全量 396 文件 3353 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.393.0。
- **坑**：蒙版黑白数据语义要 hex-exempt/SetLatentNoiseMask 裁剪不 resize 须显式对齐/JSON 往返数字变 float64/vi.mock 工厂 hoisting 不能引顶层标识/antd Slider 不转发 testid。
- **产物**：exe 见 SHA256SUMS-v4.393.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.389~v4.393 删 v4.388.0.exe。
- **文档**：releases/v4.393.0.md+CHANGELOG/README+releases/README（414→415+34 席插 v4.393 裁 v4.359）+AGENTS 迁 1 插 1（一百零八迁：v4.388 入 archive）+todos。
- **观察池**：扩图（outpaint 画布扩边）；抠图/透明底导出；img2img+mask 纯局部重绘；蒙版反选/羽化；编辑历史变体簇；Lightning LoRA；多图编辑。

## 最新发布：v4.392.0（2026-09-22）「绘梦·指令编辑本地档：ComfyUI 接入 Qwen-Image-Edit 2511 官方工作流」

- **刀型**：图像域长期规划 T3「本地档（B 计划）」第一刀；阶段一刀 C（v4.327 指令编辑云端先行）观察池第 2 项销号。会话主线=乐园画室欠账盘点→「画室三件套」头号=编辑力补全。Go 2 文件+前端 1 文件，绑定面 714 零变更。规格 进度计划/gaea-comfyui-edit-20260922.md。
- **病根**：指令编辑只有云端档（OpenAI 兼容 /images/edits），ComfyUI 诚实拒绝——而用户本机全局生图后端=comfyui，编辑在日用后端上不可用，与本地优先基本盘直接冲突。
- **落地**：①buildQwenImageEditWorkflow 蒸馏官方 comfyui-workflow-templates image_qwen_image_edit_2511（取道不取器，API-format 构建器与 krea2/z-image 同范式）：LoadImage→FluxKontextImageScale（按原图宽高比重标，编辑不套 1024×1024）→缩放图同时进 TextEncodeQwenImageEditPlus×2（image1+vae）与 VAEEncode→KSampler.latent_image；UNET→ModelSamplingAuraFlow(3.1)→CFGNorm(1)→KSampler（20 步/CFG 4.0/euler/simple/denoise 恒 1.0=语义编辑非整幅重绘）②缺权重可操作提示——gaea 从不自动下载模型，value_not_in_list 时错误追加三件文件名+HF 链接+目录（VAE 与 krea2 共用），错误即安装指引；普通错误不追加③GenerateMedia edit+comfyui 时 imgModel 如实改写 qwen-image-edit（结果卡/台账记真实引擎）④InstructionEditModal 说明改口。编辑链全继承既有基建：进度回调/取消/ComfyUI 自动拉起（v4.387）/落盘+台账登记零新增路径。FluxKontextMultiReferenceLatentMethod（官方明示官方权重不需要）与 Lightning 4 步 LoRA（可选加速）均不接。
- **测试**：ai 层改写 ComfyUI edit 拒绝测试为三件（全链 httptest 工作流形状/缺原图/缺权重提示含 HF 链接且普通 500 不追加）+app 层 +1（元数据改写）+前端 +1（说明文案）；全量 395 文件 3347 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.392.0。
- **坑**：官方模板新 subgraph 格式——节点图要从 definitions.subgraphs 展开+interface 槽位对射，顶层只有 4 节点/CLIPLoader 的 clip_name 查 text_encoders 目录非 clip/编辑不套请求 size——FluxKontextImageScale 按原图重标/提示锚 value_not_in_list+模型名双匹配防误挂。
- **产物**：exe 见 SHA256SUMS-v4.392.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.388~v4.392 删 v4.387.0.exe。
- **文档**：releases/v4.392.0.md（含升级说明：ComfyUI 档编辑需先放模型文件，错误即指引）+CHANGELOG/README+releases/README（413→414+34 席插 v4.392 裁 v4.358）+AGENTS 迁 1 插 1（一百零七迁：v4.387 入 archive）+todos。
- **观察池**：Lightning 4 步加速 LoRA；image2/3 多图编辑；蒙版局部重绘/扩图/抠图；编辑历史变体簇（ParentID 链）；模型文件名 config 化；FLUX.2 Klein 编辑族第二档；编辑产物回填角色库。

## 最新发布：v4.391.0（2026-09-22）「附件 URL 缓存 blob 化：base64 大字符串迁出 JS 堆 + selbar z 防御修」

- **刀型**：性能池⑦「base64→blob URL 会话资源管理」收口（v4.365 审计留池最后一项免拍板性能项）+selbar z 防御修+todos 考古。纯前端 3 文件+2 测试（改 1 新 1），绑定面 714 零变更。
- **病根**：usePortraitUrl 的 dataUrlCache=全局无界 Map，值是 MB 级 base64 data URL 字符串——消息内联图/缩略图/inspector/角色头像/剧照全部经此通道（v4.365 已消重复桥调用，但字符串驻留 JS 堆的问题在）。
- **落地**：①缓存值 data URL→blob: object URL（atob→Uint8Array→Blob→createObjectURL——字节迁浏览器侧 blob 存储脱离 JS 堆，原字符串即被 GC；转换在既有 then 异步回调；消费方零改动）②LRU 96+命中刷新+淘汰延迟 120s revoke③无 createObjectURL 回退 data URL④selbar z 1080→990。
- **测试**：+1 文件 5 例+PortraitImg 断言按新行为收口；全量 395 文件 3346 例绿（一轮 1 例无足迹 flaky 复跑全绿）+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.391.0。
- **坑**：jsdom 30 原生 createObjectURL（钉 data: 断言真挂）/LRU 淘汰即 revoke 裂在屏图（条目被组件 state 持有）/data URL→blob 别用 fetch(dataURL)（jsdom 不支持）/后台测试命令只留 tail 丢失败详情。
- **产物**：exe 见 SHA256SUMS-v4.391.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.387~v4.391 删 v4.386.exe。
- **文档**：性能池⑦销号回填+releases/v4.391.0.md+CHANGELOG/README+releases/README（412→413+34 席插 v4.391 裁 v4.357）+AGENTS 迁 1 插 1（一百零六迁：v4.388 入 archive）+todos。

## 最新发布：v4.390.0（2026-09-22）「canvas 调色板集中化：resolveThemeColor 收口三份手写解析器 + 主题切换跟随缺陷根修」

- **刀型**：视觉池③+⑤专项（v4.361 审计留池大工程候立项中可代码级收口的两项；①②④仍需逐板块截图对照拍板）。纯前端 7 文件+3 新测试，绑定面 714 零变更。
- **侦察发现**：三份手写 CSS 变量解析器各缺一块（RelationGraph 模块级求值不随主题刷新/GraphView 只解析一个变量且 once/CompanionAvatar 字符串手拆不支持 color-mix）；RelationGraph 真缺陷=角色色/组织色/关系色/画布底模块级顶层求值，v4.350 只修组件内 canvasColors 漏网模块常量。
- **落地**：①resolveThemeColor/RGB（utils/theme.ts）span 探针法——浏览器自己算，支持 color-mix/嵌套 var/fallback 链；RGB 元组版拼 rgba()②RelationGraph palette=useMemo([darkMode]) 全量收口+graphData/decoratedNodes/edges 依赖补 palette+图例改 var() 活解析③GraphView 背景跟随（探针+MutationObserver 监听 :root 内联 style，不引老栈 store）④CompanionAvatar 删手拆改探针+withAlpha（hex 后缀拼接遇 rgb() 串拼非法色被 canvas 静默忽略）⑤AoaView/PdmView critical 兜底 #dc2626→#e02020。
- **测试**：+3 文件 9 例（theme 5+RelationGraph 2〔darkMode 切换后 fillStyle 先暗后亮=重解析链路钉死〕+CompanionAvatar 2）+目检（esbuild 挂具+Edge 无头暗/亮两页：两态节点+连线+图例全渲染，亮=浅底深字走重解析路径，改造前停留暗色）；全量 394 文件 3341 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.390.0。
- **坑**：①canvas 测试 jsdom 三连坑——offsetParent 恒 null 触发 v4.365 空转挂起/首绘 rAF 等帧 flush/ctx 桩 Proxy 兜底任意方法②hex 后缀拼接只兼容 6 位 hex——非法色被 canvas 静默忽略不报错③模块级求值=主题切换隐形死角④getPropertyValue 读 custom property 只到引用替换层⑤Edge --screenshot 单帧不跑 rAF——canvas rAF 组件截图全空须 --virtual-time-budget⑥gaea 侧主题跟随不引老栈 store。
- **产物**：exe 51181568B SHA256=0ad223995fc7816bbaf810f1cff82d41942390bcdc8958464a85f79f05335404（SHA256SUMS-v4.390.0.txt，仅本地；冒烟 /api/health 200 过；桌面副本同哈希实测一致；字节数与 v4.389 巧合相同哈希不同）；保留策略 5 版留 v4.386~v4.390 删 v4.385.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.390.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+计数 411→412+34 席插 v4.390 裁 v4.356）+AGENTS 迁 1 插 1（一百零五迁：v4.387 入 archive）+todos。

## 最新发布：v4.389.0（2026-09-22）「模型中心统计重设计：右栏检查器遥测读出式 + 详细统计抽屉信息分层」

- **动机**：用户指令「重新设计模型中心右侧面板的统计和详细统计」。
- **诊断**：右栏=4 张 KPI 卡 2×2 平铺（300px 窄柱又厚又挤、hint 常被省略号截断）+ 两张 720 宽 viewBox 全尺寸趋势图压进 ~276px 渲染（轴字缩到 ~4px 完全不可读）；抽屉=块内标题与抽屉标题重复、5 张 KPI 图标全是同一个闪电、「本地 vs 云端」6 张卡两行平铺。
- **落地**（纯前端 8 文件含 eslint 配置 1，绑定面 714 零变更）：①右栏遥测读出式——三格主指标（总调用·成败/成功率阈值着色/估算费用）+ Token 紧凑读数行（k/M）+ 窄柱专用迷你图 RequestsSpark/TokenMiniBars（260 视口 1:1 设计字标不缩放，保留逐点 tooltip）+ 底部「查看详细统计」入口（context 新增 openStatsDrawer，右栏直开抽屉）②抽屉信息分层——工具条（统计自/价格目录芯片+排序刷新清空）+「云端/本地分流」单卡叙事（Token 占比条+云端费用/本地用量/已节省三列+KV 命中率虚线紧凑行）+ KPI 五格语义图标+Token hint 紧凑格式防截断+图例类化③全尺寸两图 viewBox 720→540+轴字 10→11（抽屉双列 ~380px 栏宽有效字号 5.3→7.7px）④eslint globalIgnores 补 .tmp（挂具 bundle 的 legal comments 触发指令错的门禁欠账）。
- **测试**：前端 +2 文件 8 例（InspectorPanel 三态+入口联动/StatsSection 工具条芯片+KPI 着色+hint 紧凑+分流带占比口算）；modelcenter 17 套 121 例+全量 391 文件 3332 例绿+tsc 0+eslint 0。目检两轮（esbuild 挂具+Edge 截图）：检查器六项全过，抽屉唯一缺陷（Token hint 截断）修复后复检确认。坑：窄柱放全尺寸图轴字不可读——图表按目标渲染宽度设计 viewBox；antd 两字中文按钮自动插空格+图标 textContent 前导空格——断言按存在性/trim；挂具视口 <1180px 触发检查器隐藏媒体查询（截图全黑≠渲染失败）；fmtCompact 归 utils.tsx 不进组件文件（react-refresh 告警）。
- **门禁**：全量 ci.ps1 绿 EXIT=0 + 版本漂移闸 OK@4.389.0。
- **产物**：exe 51181568B SHA256=ba43b3a1cecf79431de4739bc1deb918e621a1b5b98025f4d3c471bc0857693a（SHA256SUMS-v4.389.0.txt，仅本地；冒烟 200 过；桌面副本同哈希实测一致）；保留策略 5 版留 v4.385~v4.389 删 v4.384.0.exe（SUMS 全保留）。
- **文档**：releases/v4.389.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+计数 410→411+34 席插 v4.389 裁 v4.355）+AGENTS 迁 1 插 1（一百零四迁：v4.386 入 archive）+todos。

## 最新发布：v4.388.0（2026-09-22）「原罪插图独立生图绑定：功能绑定卡新增『原罪插图』后端/模型选择」

- **动机**：用户反馈「原罪的功能绑定卡片内增加生图模型选择，这么多模型为什么没有候选，只能使用 comfyui？」。
- **诊断**：原罪生图此前无独立绑定，跟随全局生图设置（本机全局后端=comfyui，所以看起来「只能 comfyui」）；模型中心「功能绑定」原罪卡只绑故事文本模型；跨引擎生图候选其实一直在（剧照卡同款 imageModelOptionsFor），原罪没有入口。
- **落地**（镜像「角色库剧照」先例，Go 6 文件+前端 9 文件，绑定面 712→714）：①GetSinImageConfig/SetSinImageConfig（sin_image_backend/model，空=跟随全局，任一项可单独回退）+shelf 热更新②生成路由——buildImageClientFor 从 buildPortraitClient 通用化（comfyui/herdsman/ollama/glm/xai 五后端独立客户端，不动绘梦全局）；generateImageInternal 加 clientOverride/backendOverride 通道（进度回调/size 裁剪/ComfyUI 自动恢复与拉起分支统一按生效后端判定）；SinIllustrate 级联 sinImageBinding（绑定>全局）+sinRefPlan 用生效值+绑定后端≠全局时独立客户端；sin_illustrate 工具同入口受益③模型中心新增「原罪插图」卡（六后端下拉+跨引擎模型候选），原罪故事卡注释补指引。
- **测试**：Go +4 组（级联四态/Get 回读/buildImageClientFor 分支/override 通道 override=1 全局零触碰）+前端 +1（卡渲染+保存+已绑定态）。坑：config.Save 直写真实 ~/.gaea_config.json 无隔离钩子——Set roundtrip 不能真调，测读映射+纯函数级联；种子引擎目录恒 Enabled 须 SaveEngine 禁用再测未启用分支；override 后端语义要跟到底（三处判定原读全局配置）。
- **门禁**：全量 ci.ps1 绿 EXIT=0 + 版本漂移闸 OK@4.388.0 + spaceBindings 锁 535→537 + bindingNames 714。
- **产物**：exe 51171328B SHA256=48f1a288eaadb80a6a8632a81ac6d0dbff0fd110b87318947c4682a37bb141e1（SHA256SUMS-v4.388.0.txt，仅本地；冒烟 200 过；桌面副本同哈希实测一致——用户已关 gaea 连跳两版直更 v4.388）；保留策略 5 版留 v4.384~v4.388 删 v4.383.0.exe（SUMS 全保留）。
- **文档**：releases/v4.388.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+计数 409→410+34 席插 v4.388 裁 v4.354）+AGENTS 迁 1 插 1（一百零三迁：v4.385 入 archive）+todos。

## 最新发布：v4.387.0（2026-09-22）「ComfyUI 未运行自动拉起：原罪插图/绘梦生成连接被拒自愈」

- **动机**：用户报障「原罪生图失败，插图失败：ComfyUI 提交失败…connectex: No connection could be made because the target machine actively refused it」。
- **诊断（两层根因）**：①环境层——8188 无监听、ComfyUI 进程不在；手动拉起又暴露 standalone-env 落后于仓库 requirements（comfy-aimdo 0.4.13<0.5.5 缺 storage、comfy-kitchen 0.2.28<0.2.35 缺 int8_attention 等，一族过期，起都起不来）②产品层——gaea 生成链只对「孤儿实例 [Errno 22]」自动恢复，服务压根没跑直接 connectex 原始错误失败；原罪插图页无恢复入口（绘梦页才有 StartComfyUI 按钮）。
- **环境修复（用户本机随刀处置）**：按 requirements 钉版精确补 comfy-aimdo==0.5.5/kitchen 0.2.35/embedded-docs 0.5.11/frontend-package 1.52.7/workflow-templates 0.11.62（**刻意不 pip install -r**——torch 无钉版行会重解析，ROCm 特制 torch 可能被 PyPI 通用版顶掉）；修后以 gaea 同配方（standalone-env python+--windows-standalone-build，日志同 ~/.gaea/logs/comfyui.log）拉起 ~30s 就绪（ComfyUI 0.36.0，/system_stats 200）——**用户当前会话重试插图即恢复**。
- **落地（gaea 根修，Go 1 文件+测试，绑定面 712 零变更）**：ensureComfyUIRunning 助手（未运行→StartComfyUI→/system_stats 就绪轮询有界 120s；未配置路径/启动失败/超时=返回 false，**原始错误如实上抛不吞错**）接线 generateImageInternal 重试分支——错误含「连接 ComfyUI 失败」（dial 层文案，服务不可达才出现，服务在但卡死不进防误杀）+backend=comfyui 时拉起成功后重试一次（comfyBooted 闸）；并发竞态由 StartComfyUI 端口占用守卫兜底（「已被占用」转就绪等待）；原罪插图与绘梦共用本链，一处接线双板块受益。
- **测试**：Go +3（image_boot_test.go 零真进程：连接被拒→假就绪端点→重试恰一次〔计数=2〕/未配置路径不重试且原始错误上抛/ensure 三分支契约——已运行即真·未配置即假·无 main.py 启动失败即假快速返回）；internal/app 全量绿。坑=测试成功路径必设 ImageSaveDir 指临时目录（否则 saveToNovelImages 撞装配态 nil）。
- **门禁**：全量 ci.ps1 绿 EXIT=0 + 版本漂移闸 OK@4.387.0。
- **产物**：exe 51163136B SHA256=f7ec8eb3c30260104a180beb81ba6354399aecc6bece6962c70bf9ba85a2e17b（SHA256SUMS-v4.387.0.txt，仅本地；冒烟 /api/health 200 过；桌面副本因用户会话在跑未覆盖——关闭 gaea 后替换，当前会话靠环境修复已可用）；保留策略 5 版留 v4.383~v4.387 删 v4.382.0.exe（SUMS 全保留）。
- **文档**：releases/v4.387.0.md（含升级说明：首次冷启动生图多等一段属预期）+CHANGELOG/README+releases/README（保留策略行+计数 408→409+34 席插 v4.387 裁 v4.353）+AGENTS 迁 1 插 1（一百零二迁：v4.384 入 archive）+todos。

## 最新发布：v4.386.0（2026-09-22）「造价库分页绑定：成本条目列表/表格分页 + 检索下拉载荷降载」

- **动机**：用户口径「继续优化迭代」——todos 扫一遍，可自主推进的工程项=前端性能池⑥余项（造价库全表留池；其余多为等真机/等拍板/等上游的等条件型）。成本库「成本条目」每次搜索/筛选全表拉回全量渲染，用户真实库 1553 条时每次 250ms 防抖击键=全量桥载荷+1553 行 DOM。Go 3 文件+前端 10 文件，绑定面 711→712（+1）。
- **落地**：①**GaeaCostSearchPage 分页绑定**（App 层实现，cost.Store 零改动）——检索管线（SQL→Go 关键词过滤→语义召回→本地精排）提为 costSearchAll 助手两绑定共用，分页与非分页口径一致、工具面 cost_search 零影响；排序进服务端 title/price/updatedAt（空=管线序）**tie-break 恒定 name**（全序=跨页不漂移前提）；limit 钳 [1,200]（≤0→100）、offset 负归零、Total=过滤后总数；未知 sortKey 不排序②**CostLibraryView 分页改造**——首屏 100+「加载更多（余 N 条）」100/批、计数如实「已载 X / 共 Y 条」、分类树计数用 total；**客户端排序删除**（分页时代只排已载子集是静默错误答案）改服务端排序重拉第 1 页；过期响应 reqSeq 丢弃+追加页跨页 name 去重③**EntryPicker 切分页**（top8 载荷降载）；loadStats 总览仍全量（聚合需要，开页一次非热路径）；mock 同口径。
- **测试**：Go +5（切片连续+total/全序 tie-break+降序镜像/钳制四态/零命中空页/未知键管线序/过滤×排序组合〔updated_at RFC3339 秒级落盘→SQL 直铺确定时间序〕）+前端 +1（150 条分页补齐）+迁移断言（7 参形态/排序异步等待）。全量 vitest 389 文件 3323 例绿+tsc 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.386.0+bindingNames 再生 712+spaceBindings 锁 534→535。
- **事故与修复（本轮最大坑）**：Go 测试零命中查询触发**真实语义管线**——NewManager 种子目录 herdsman 恒 Enabled+localhost:8080，本机 embedding 服务在跑时 SQL 召回<3 即走语义补召回：Stale("cost",keep) 按测试条目集清掉真实库 cost 向量行+Ensure 写假向量。**已修复**：假行 DELETE 清污+走应用自身 GaeaSemanticIndexBackfill 全量重建 1553/1553=24s（复核实测）；测试侧 SaveEngine 禁用 herdsman 断通道。教训：hubCostStore/hubSemanticStore override 只护一半，resolveHerdsmanSearchModel 引擎目录在测试环境是活的；App 嵌 *core 裸 &App{} 字段访问即 panic（须 &App{core:&core{…}}）。
- **产物**：exe 51160576B SHA256=4201619c…458cd（全文见 SHA256SUMS-v4.386.0.txt，仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.382~v4.386 删 v4.381.0.exe（SUMS 全保留）。
- **文档**：性能池⑥销号回填+releases/v4.386.0.md（含升级说明：标题排序改字节序、列表默认 100 条）+CHANGELOG/README+releases/README（保留策略行+计数 407→408+34 席插 v4.386 裁 v4.352）+AGENTS 迁 1 插 1（一百零一迁：v4.383 入 archive）+todos。
- **下刀候选**：性能池⑦base64→blob 会话资源管理；视觉大工程（字号阶梯/玻璃收敛等，需专项+截图对照）；其余等条件型不动。

## 非版本刀（2026-09-22）projection-cache 收益评估：辨伪销号（dsh ⑪）

- **动机**：dsh 留池③「session-projection-cache 冷启动加速（收益待评估）」——评估债清偿，用基准实测代替拍脑袋。
- **结论**：**辨伪销号，不立项**。gaea checkpoint 机制（每轮模型调用前 flush 消息投影+seq，fail-closed）已是投影缓存的等价物——Restore 只投影 checkpoint 之后的尾差；基准实测（合成日志 user/assistant 交替 ~1.5KB/条）：2MB≈20ms / 10MB≈64ms / 50MB≈267ms（有 checkpoint；无 checkpoint 也仅 +15~45%），真实会话多在 1-5MB=几十 ms。成本主导项是全文件 read+parse（不换文件格式无法省，而换格式牵动 append-only 不变量），投影缓存的增量收益不可感知。
- **产物**：基准在案 internal/gaea/agent/session/restore_bench_test.go（go test -bench BenchmarkRestoreSynthetic，2/10/50MB × 有无 checkpoint 六组，可复跑）。
- **文档**：dsh 蒸馏文档⑪辨伪销号回填（留池余 2=子代理控制面等上游稳定+Plan mode 需前端配套，全为等条件型）+todos 头部行更新。

## 最新发布：v4.385.0（2026-09-22）「工程健康轮 #2：excelize 防线扩面 + 死代码清欠 + 双扫描报告」

- **动机**：用户口径「继续」——距 v4.370 工程健康轮 15 版，按惯例复扫三面。Go 2 文件+前端 4 文件，绑定面 711 零变更。
- **扫描报告**：govulncheck 1 项在案（GO-2026-6452 excelize，上游 N/A）/pnpm audit 0/knip 1 未用导出+3 未用类型+3 重复+5 提示。
- **落地**：①excelize 防线扩面（v4.370 只防 xlsxpreview；本轮符号追踪新可达面 schedule.ImportXlsx GetRows+GaeaXlsxRowOps/ColOps InsertRows/Cols/RemoveRow/Col——同款 named-return+defer recover 转可读错误）②knip 清欠 4 项（useModelCenter 兼容口零消费+桥接死类型 ForeshadowUrgencyPayload/NovelRewriteRequest/SinBookSourceDownloadResult；tsc+grep 双验证；stale 注释同步更新；重复导出/配置提示 KEEP 不动）。
- **测试**：app/schedule 全量绿（纯增量 defer 零行为变化）+tsc 0+modelcenter 112 例绿。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.385.0。
- **坑**：①「已防御」≠「全防御」——符号追踪随代码演化增长，扫描要周期性复跑②Fixed in N/A 常驻项——新增 excelize 消费面必须同步配 recover 入口③knip 未用导出区分死代码与契约哨兵（重复导出 KEEP 不跟风删）。
- **产物**：exe 51153920 B SHA256=9d8de0811f3aa6de298ed4b9ee5b78d3159a735bebbf4a48189580d2f6b5a72e（releases/gaea-v4.385.0.exe+SHA256SUMS-v4.385.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.381~v4.385 删 v4.380.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.385.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 406→407+34 席插 v4.385 裁 v4.351）+AGENTS 迁 1 插 1（一百迁：v4.382 入 archive）+progress/todos。

## 最新发布：v4.384.0（2026-09-22）「dsh 蒸馏第五弹：session_query 会话检索（跨会话记忆首刀）」

- **动机**：用户口径「继续」——dsh 余 4 中唯一可独立推进的大工程项⑧ session-query 首刀落地，留池余 3。全 Go，绑定面 711 零变更。
- **落地**：①索引器（新包 internal/gaea/sessionquery：索引面=装配空间会话目录 *.gaea-log.jsonl 的 user_message/assistant_message 正文，system/工具结果/流式 delta 噪声面不入索引；增量按 (size,mtime) 新鲜度指纹惰性刷新零 boot 成本；重索引=DELETE 旧行再全量插；空间隔离由目录构造，双空间红线同构）②session_search 只读工具（query+limit 默认 8 上限 32，newest-first 带 session id/role/时间/snippet；中文降级=FTS5 unicode61 不切 CJK 子串 MATCH 零命中回退 LIKE 子串+手工裁窗，whisper 先例；MATCH 短语引号化）。
- **测试**：Go +5（FTS 命中+噪声面排除+ts 倒序/CJK 子串降级/增量新鲜度/目录隔离/工具面格式化+空查询+零命中诚实）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.384.0。
- **坑**：①ReadLogRepaired 的 torn-tail 修复截断「未以换行收尾」的最后一行——手工构造日志必须换行收尾（真实 LogWriter 恒换行收尾）②FTS5 标准表（非 contentless）才支持普通 DELETE——UNINDEXED 元数据列建虚表免联表可重索引③LIKE 路径 snippet 手工裁窗——两路径 Scan 形状不同，共用 Scan 助手静默拿空原文（首写即犯测试当场抓出）。
- **产物**：exe 51150848 B SHA256=221d26d35d9bd7eeaf5a4fcb15afa647cb30cfaf0ff1a498c4ae68fea5967d25（releases/gaea-v4.384.0.exe+SHA256SUMS-v4.384.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.380~v4.384 删 v4.379.0.exe（SUMS 身份档案全保留）。
- **文档**：dsh 蒸馏文档⑧首刀销号回填（余 3）+releases/v4.384.0.md（含升级说明：首检索全量索引一次）+CHANGELOG/README+releases/README（计数 405→406+34 席插 v4.384 裁 v4.350）+AGENTS 迁 1 插 1（九十九迁：v4.381 入 archive）+progress/todos。

## 最新发布：v4.383.0（2026-09-22）「聊天虚拟化第二层：绘制层 content-visibility + 挂池考古销号一批」

- **动机**：用户口径「继续」——v4.365 池②「聊天消息真虚拟化」收口（两形态分层），顺带一批过时挂池条目考古销号（零代码）。前端 2 文件+测试 +2，绑定面 711 零变更。
- **落地**：①绘制层虚拟化（超阈值会话根节点挂 chat-flow-cv，行级 content-visibility:auto 离屏跳 layout/paint；contain-intrinsic-size:auto 120px 记住行实际高度；≤阈值不挂类零影响；流式吸底/antd portal 菜单不受影响；react-window 被 v4.370 侦察否决有案=滚动宿主/动态行高/流式吸底三冲突）②挂池考古销号（零代码）=v4.355②file:// 头像（v4.361 已清账）/v4.355③longtask 阈值（v4.361 已提 200ms）/v4.349⑤initRuntimePolyfill（辨伪同步成本≈0）+vendor manualChunks（无新证据）。
- **测试**：前端 +2（>阈值挂 chat-flow-cv/≤阈值不挂——类挂载即完整契约）；全量 vitest 389 文件 3322 例绿+tsc 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.383.0。
- **坑**：①content-visibility 的行内 fixed/portal 子元素语义要先查——containment 裁剪 fixed 后代，聊天行菜单 portal 到 body 不受影响，novel selbar 不在作用面②「真虚拟化」不等于 react-window——三冲突场景下渲染层窗口+绘制层 CV 分层零结构风险达成同目标，池条目按目标销号不按实现手段。
- **产物**：exe 51128320 B SHA256=4453235ab91e5a39187e4e4112e363df89937e66e96731f43f6f7bd0a310e588（releases/gaea-v4.383.0.exe+SHA256SUMS-v4.383.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.379~v4.383 删 v4.378.0.exe（SUMS 身份档案全保留）。
- **文档**：池②收口+三条考古销号回填+releases/v4.383.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 404→405+34 席插 v4.383 裁 v4.349）+AGENTS 迁 1 插 1（九十八迁：v4.380 入 archive）+progress/todos。

## 最新发布：v4.382.0（2026-09-22）「前端挂池清欠：整店订阅清零 + 交付物缓存失效接线」

- **动机**：用户口径「继续」——dsh 余 4 项全是「等条件」型，转前端挂池清欠（v4.349 池④+v4.354 池④）。前端 4 文件+测试，绑定面 711 零变更。
- **落地**：①无 selector 整店订阅清零（全仓扫描四家 store 仅剩 WorkspacePanel/FilePreviewModal 2 处 selector 化；审计对账=MainLayout/HomePage/AppearancePanel 已被中间轮次 selector 化行号过时，本轮后全仓归零；gaea useController 的 useShallow 整面订阅保留=App.tsx 消费近乎全量字段，热字段 items 流式本就逐 chunk 变化）②交付物缓存失效接线（invalidateTurnCaches 接到 turn_done 事件消费点+loadSessionData 会话切换；新轮交付物/新建文件不再被 TTL 内旧目录探测误标缺失；纯内存清空零桥接成本）。
- **测试**：前端 +2（turn_done 后 ensureTurnRegistry 重拉打桥的调用计数断言/会话装载同语义）；全量 vitest 389 文件 3320 例绿+tsc 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.382.0。
- **坑**：①挂池审计行号会过时——清欠前先全仓重扫，销号写清哪些本轮修哪些中间轮已修②恰好一次的事件绑定是模块级单例——从可观测面（bridge 调用计数）断言行为不 mock 内部③缓存失效时机在事件边界——reducer 保持纯函数，副作用接既存非纯区（turn_done 块本就有 ContextUsage/Balance 副作用）。
- **产物**：exe 51128320 B SHA256=a44212dc84ddcdad14c54c7a1c0bd91783858d9907efa377e0362c72f0b0139f（releases/gaea-v4.382.0.exe+SHA256SUMS-v4.382.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.378~v4.382 删 v4.377.0.exe（SUMS 身份档案全保留）。
- **文档**：挂池④/dirListingsCache 销号回填+releases/v4.382.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 403→404+34 席插 v4.382 裁 v4.348）+AGENTS 迁 1 插 1（九十七迁：v4.379 入 archive）+progress/todos。

## 最新发布：v4.381.0（2026-09-22）「dsh 蒸馏第四弹：锚定式 token 计量（压缩触发时机去漂移）」

- **动机**：用户口径「继续」——按上轮承诺对 dsh ②锚定式 token 计量单独一版精修（compression 域最后一块精度拼图），留池余 4。全 Go，绑定面 711 零变更，单刀版。
- **落地**：agent/tokenmeter.go 锚定压力表——压力=上次真实 usage 锚点+表面有符号差值；per-message 估价 memo（FNV-64 与 msgChars 同记账面，ReasoningContent 不入账）让差值只统计新增/删除/改写，未变消息锚点估价逐项相消（剪枝/压缩重写后不漂移）；落锚点=maybeCompact 收到真实 usage 时（表面恰=刚被 answered 的请求、assistant/tool 结果未入账，无系统偏差；LastPrompt 回退不落锚）；估压有锚=锚点+差值（负差值钳非负），无锚=裸字符估算旧口径不变；恒装配无配置面，EstimateContextTokens 委托计量表（midTurn/compactIfOver/force 检查同口径）。
- **测试**：Go +5（比率源漂移下未变消息不重算+新增移除精确移动+重复读取零漂移/压缩重写精确移动/负差值钳/无锚回退/maybeCompact 落锚而回退不落）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.381.0。
- **坑**：①锚点时点对齐「刚被 answered 的请求面」——maybeCompact 在 session.Add(assistant) 前调用才无系统偏差，挪后差一整轮②新消息按当前比率估、老消息按锚点时点比率消是刻意的（memo 键只含内容不含比率，比率演化不影响一致性）③sync.Mutex 不可重入——持锁调内部无锁变体，别再走公开方法。
- **产物**：exe 51128320 B SHA256=42b574026869c84f0d9ae31203b3cdd6d7036fc4907d4c04a313e078dc316647（releases/gaea-v4.381.0.exe+SHA256SUMS-v4.381.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.377~v4.381 删 v4.376.0.exe（SUMS 身份档案全保留）。
- **文档**：dsh 蒸馏文档②销号回填（余 4）+releases/v4.381.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 402→403+34 席插 v4.381 裁 v4.347）+AGENTS 迁 1 插 1（九十六迁：v4.378 入 archive）+progress/todos。

## 最新发布：v4.380.0（2026-09-22）「dsh 蒸馏第三弹·子代理编排：Fork 型子代理 + 结构化输出补完 + V10.36 对齐复活」

- **动机**：用户口径「继续」——dsh 留池两把可安全落地的编排刀凑一组（委派上下文继承 + 结构化结果回收），留池余 5。全 Go，绑定面 711 零变更。
- **落地**：①Fork 型子代理（task 新增 fork；TaskTool 实现 ContextualTool 取父历史做种子；种子自带父 system 字节同源、不注模板提示=子请求前缀与父逐字节一致→KV 缓存继承第一轮近乎免费；组合禁忌 background/continue_from/retry_until；runSubSession 会话持有上移）②结构化输出补完（schema 注入子代理 prompt+[OUTPUT FORMAT] 指令；terminal guard 非 JSON 同会话纠偏重入有界 2 次，热前缀缓存成本极低，耗尽诚实降级诊断前缀+原样；stripJSONFences 剥围栏）③根修 V10.36 对齐复活（P0-1 每轮重置 activeSchemas 无条件清 nil 把构造期父对齐 schema 第一轮抹掉=死代码；New 存 baseSchemas 基线、重置恢复基线，主代理基线 nil 行为不变；V6.0 两测试钉死形态改写为「请求目录父全量缓存对齐+执行面过滤 buildSubReg」双断言）。
- **测试**：Go +6（fork 种子字节同源+缓存形状+模板提示不注入/组合禁忌四态+无父上下文/schema 注入可见/guard 同会话恢复含剥围栏/guard 耗尽诚实降级/stripJSONFences 三态）+改写 2（V6.0 双断言重构）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.380.0。
- **坑**：①「每轮重置」类清理会把构造期注入的配置一起抹掉——重置要回基线不是零值，死代码靠端到端断言请求形状辨识（schema 数 1 vs 2 当场暴露）②V6.0/V10.36 两代测试钉矛盾行为各活各的（对齐是死的两边都绿）——修复一处另一处断言爆开，改断言不是改实现③RunSubAgent 把会话藏在内部——需同会话二次进入/会话预置的功能都要把会话持有上移调用方④executeOne 把工具结果包信封 JSON——测试透信封断言别假设裸文本。
- **产物**：exe 51117568 B SHA256=1a3a237214d1805c337530bf53481f1d516003d6f2de8c2cb883e61b0b878b27（releases/gaea-v4.380.0.exe+SHA256SUMS-v4.380.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.376~v4.380 删 v4.375.0.exe（SUMS 身份档案全保留）。
- **文档**：dsh 蒸馏文档 fork/结构化输出销号回填（余 5）+releases/v4.380.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 401→402+34 席插 v4.380 裁 v4.346）+AGENTS 迁 1 插 1（九十五迁：v4.377 入 archive）+progress/todos。

## 最新发布：v4.379.0（2026-09-21）「dsh 蒸馏第二弹：spill 泄洪 + 中断流结构化块保全」

- **动机**：用户口径「继续」——dsh 留池可安全落地的两把内核刀（spill 泄洪 + 中断流结构化块保全），留池余 7。全 Go，绑定面 711 零变更（read_spill 是 agent 工具目录新增非 App 绑定）。
- **落地**：①spill 主动泄洪（新叶子包 internal/gaea/spill 内存库：单条 4MB/总量 64MB FIFO 驱逐；executeOne 在 SmartCompress **之前**捕获原文——gaea 压缩管线会先销毁中段；内联文本**前置** locator；新工具 read_spill 分页取回；豁免 read_file/read_spill/task；best-effort 纪律；[agent] tool_spill 默认开，子代理 taskTool.SetSpill 随父）②中断流结构化块保全（终态流错误/预算阻断路径 assistant 含 calls 落库 + persistInterruptedCalls 成对补结果行：预执行只读用真实结果、其余 WrapError 信封合成占位；ToolResult 事件照发收口前端卡片；恢复路径刻意不落 calls=悬空非法形态辨析）。
- **测试**：Go +12（spill 包 4：分页/剩余字节/FIFO 驱逐/超限拒收/ctx 往返；agent 5：泄洪全文锚点/豁免与开关/截断生存性/read_spill 端到端跨回合保序/未知 id 报错；中断保全 3：成对落库+悬空检查/预执行真实结果/恢复不落 calls）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.379.0。
- **产物**：exe 51111424 B SHA256=affb81cd23963555085479c6bd47adbf716abd3762b10e7ce92fcd7b661e3385（releases/gaea-v4.379.0.exe+SHA256SUMS-v4.379.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.375~v4.379 删 v4.374.0.exe（SUMS 身份档案全保留）。
- **坑**：①信封是单行紧凑 JSON——截断按「行」选择按「字节」裁切，行数不超限时全行保留后首行被**头部**裁切：要活的内容（locator 指引）必须放头部，尾缀版被测试当场抓出假绿②混合批执行序非调用序（v4.375 分区并行不同资源键共存）——「先泄洪再取回」端到端测试同批会竞态假红，跨回合才有全序③agent 测试引 builtin + builtin 引 agent=测试期 import cycle（生产编译不报 go test 才炸）——SpillStore 下沉叶子包，策略留 agent④preOutcomes 存已包信封 output——合成占位同信封约定，否则回放两种形态混存。
- **文档**：dsh 蒸馏文档①③销号回填（余 7）+releases/v4.379.0.md（含升级说明：tool_spill 关闭回旧行为/中断续聊占位语义）+CHANGELOG/README+releases/README（计数 400→401+34 席插 v4.379 裁 v4.345）+AGENTS 迁 1 插 1（九十四迁：v4.376 入 archive）+progress/todos。
## 最新发布：v4.378.0（2026-09-21）「Reasonix 真身蒸馏第四弹·留池收官：bash 会话状态锚定 + memory activation 二维 + subject keys」

- **动机**：用户口径「继续优化完善 gaea」——Reasonix 真身蒸馏第四弹，清掉最后 2 项留池（Persistent bash PTY + memory 事实生命周期）收官整条蒸馏线；同日对账 dsh 留池销 3（12→9）。全 Go，绑定面 711 零变更。
- **落地**：①bash 会话状态锚定（cd/export 含导出函数跨调用存活；取道不取器=进程不持久状态持久；EXIT trap 观测式采集：cwd 经 cygpath -w 转 host 形+env 用 export -p 全量转储前置 eval 重放；显式 exit 照采退出码原样传播；被杀/覆盖/exec 换身不采纳；并行批次独立探针完成序采纳；目录消失自愈；豁免=PowerShell/WSL enforce/后台；会话切换经新接口 tool.SessionStateResetter 由 controller NewSession/Resume 重置）②memory activation 二维（pinned=正文随会话快照预算化注入稳定前缀 1200/单条 400、Name 升序保 touch 不翻前缀；relevant=仅检索可达；门控仅记忆总开关，空集合零注入）③memory subject keys（remember 可选 subject_key 单值问题键；Store.Save 撞键拒绝点名持有者，同名=修订放行，置空=释放；SchemaV23+文件后端 metadata.subject_key 往返）。
- **辨伪销号**：PTY 进程持久残留（与 Job Object 灭树/boundedOutput/超时硬杀相克+conpty 不值残余价值）/freshness 三档+expiry（早已在册 decay.go 三态+CleanupArchived）/dsh 三项（timeout-policy=工具内错误串无外层 deadline 通道、skill digest 重发布=缓存稳定前缀同判、contextBreakdown=ContextView 已有）。
- **测试**：Go +15（shellstate 9：端到端存活/显式 exit/保守四态/合法采纳/自愈/reset/nil 锚/Resetter 契约；activation 3；subject keys 4）+装配 1（pinned 正文双空间各见各的+未固化不进+开关关零注入）；Git Bash 真机验证 trap 采集+eval 重放。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.378.0。
- **坑**：①bash printf 的 %s 是 Go Sprintf 动词——wrapper 模板写 %%s 否则 vet 拒编②MSYS 路径形态——探针必须 cygpath -w 转 host 形否则下次 cmd.Dir chdir 失败③事件日志列与 Event 结构一一对应——subject_key 真相归 facts 表（与 body 同待遇），不为审计单独开列④NormalizeSubjectKey 空白折叠与下划线不是一回事——键等价只走 trim+ASCII 小写+空白折叠'-'一条路。
- **产物**：exe 51089920 B SHA256=3f65aafb6c69ae807f394c3f34a0272fcbf39f7326a2d6789423376f60e48622（releases/gaea-v4.378.0.exe+SHA256SUMS-v4.378.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.374~v4.378 删 v4.373.0.exe（SUMS 身份档案全保留）。
- **文档**：docs/gaea-reasonix-v138-distill-2026-09.md 第四弹三刀+留池⑤⑦收官回填（池清空）+docs/gaea-dsh-016-distill-2026-09.md 销号 3 项（12→9）+releases/v4.378.0.md+CHANGELOG/README+releases/README（保留策略行+发布说明计数 399→400+34 席索引插 v4.378 裁 v4.344）+AGENTS 迁 1 插 1（九十三迁：v4.375 入 archive；补记九十二迁漏登）+progress/todos（reasonix 行翻✅收官）。

## 最新发布：v4.377.0（2026-09-21）「办公记忆/成本库写入纠偏：自动做梦建议制 + 存量清污 + 成本劝写改口」

- **动机**：用户反馈「不管什么都自己记入记忆和成本库，根本不询问，大量错误污染数据」——主源=auto-dream 每轮直写（设计上豁免审批且记忆开关管不住），次源=成本工具文案劝写。
- **落地**：①自动做梦建议制（[dream] mode：suggest 默认/auto 旧直写/off；待确认队列 FIFO 30+接受才入库 source=explicit/忽略丢弃；DREAM_WRITE_POLICY §5 修订；记忆开关收口）②存量清污（GaeaDreamPurgePreview/GaeaDreamPurge 按审计交集删，explicit 与手工沉淀不动）③成本劝写改口（cost_save 描述+cost_search/compose 文案 4 处+compact 2 行）④默认 system prompt 记忆段改口（仅用户明确要求才 remember；自定义提示词需自查）⑤面板 UI（自动做梦三态按钮+建议忽略+清污两步确认+i18n 三语+mock 4 绑定）。
- **测试**：Go +8（模式归一化/队列 FIFO 上限出队三态空间隔离/入队分流/视图转换/接受幂等+忽略不写库/清污交集审计留痕）+前端 MemoryPanel +3（三态循环/忽略/清污两步）。绑定面 707→711。
- **门禁**：go build/vet 0+全量 ci.ps1 绿+版本漂移闸 OK@4.377.0。
- **坑**：①bindingNames.ts 重生成必须保 as const（丢则 drift 报 never 不指向真凶）②jsdom LocaleProvider 默认 en——i18n 断言双语正则③出队失败不得静默落旧写入路径④清污按钮不能只在建议非空分支渲染⑤Proxy mock 方法必须 mockResolvedValue（裸 vi.fn() 返回 undefined 接 .then 崩，单文件假绿全量才炸）。
- **文档**：DREAM_WRITE_POLICY §5+releases/v4.377.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 399）+AGENTS 迁 1 插 1（九十二迁：v4.374 入 archive）+todos。
## 最新发布：v4.376.0（2026-09-21）「Reasonix 真身蒸馏第三弹：压缩救援阶梯 + 重复调用裁决 + 技能目录预算化」

- **动机**：续 v4.375 真身留池第三弹——压缩/恢复域两把主刀+技能目录一把小刀，辨伪销号一批收口留池。全 Go，绑定面 707 零变更。
- **落地**：①压缩救援阶梯（1a slim 摘要档=估算超 window−预留−5% 边际即逐消息截头换有界转录，provider 真实拒绝后重试也降半预算 slim，fits 逐字重放不变；1b 投影截断终级 truncateRescue=压缩无进展或压缩后仍超窗时保护区收窄 target/4→抹大工具结果→整单元丢最老装 marker，先归档 fail-closed）②重复调用硬阻断降级 advisory（repeatedSuccessBlock 退役，第 3 次同签名成功起批后一次性合成提醒，真实结果不再被 blocked 替换）③skill 目录预算化渲染（超 4000 字符二分收缩描述宽度保全员可见，压缩态尾注计入预算，纯名行仍超才退硬截断）。
- **辨伪销号一批**：采样恢复状态机+预算学习（主链路不发 max_tokens、reasoning 不回放，无靶子）/skill 引用按需分页+watcher 热重载/read_tasks 续读游标/会话私有临时目录 env 重定向（Git Bash TMP 注入牵动 /tmp 映射）/压缩状态跨重启保留（重启丢熔断=天然解锁）。余池=Persistent bash PTY+memory 事实生命周期。
- **测试**：Go +13（slim 截头/参数封顶/请求收缩/fits 逐字/拒绝后降档/rescue 抹结果丢单装 marker/KeepErrors 豁免/三态无靶/溢出落到截断终级/advisory 越线一次性+改钉两条/目录压缩保全员/fitting 零变更）+2 处适配（summarize 加 forceSlim 形参）。
- **门禁**：go build/vet 0+全量 ci.ps1 绿 EXIT=0 首跑即绿+版本三处 4.376.0。
- **坑**：①救援机制的可达带先算清再写测试——可达带=compact 尾钳与 rescue 尾钳间盲区，prune 够不到、compact 尾保护又留下的大块 assistant 正文才是构造素材，否则前置清场测试假绿②token 估算器 max(bytes/4, runes/2)——纯 ASCII 按 2 字符/token 计，边界按此口径算③破坏性截断必须 fail-closed 于归档——三级都先归档再动手，archive 坏=整体拒绝。
- **产物**：exe 51030528 B SHA256=8c0febb68f936ed247a317baa6bb86d714474e9f96b9a2c50a3724f7d59b4aef（releases+SUMS 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略删 v4.371.0.exe。
- **文档**：蒸馏文档第三弹三刀+留池销号回填+releases/v4.376.0.md+CHANGELOG/README+releases/README（计数 397→398+34 席插 v4.376 裁 v4.342）+AGENTS 迁 1 插 1（九十一迁：v4.373 入 archive）+progress/todos。

## 最新发布：v4.375.0（2026-09-21）「Reasonix 真身蒸馏第二弹：执行序批次 + 文件观察 + 杂项加固」

- **动机**：续 v4.374 真身留池第二弹——执行序与数据正确性两把主刀+两把小刀。全 Go，绑定面 707 零变更。
- **落地**：①执行序批次修正（read:/file: 统一 file:<path> 资源键同路径拆批保序，跨路径共存并行保留；call/result 对应性守卫）②Live file observations（会话级版本指纹：读后记录/写前比对/自身写后刷新；外部修改 blocked:[stale version]；与 stale-anchor 并存）③git 硬化基线（-c fsmonitor/maintenance 关闭+三个 env+diff 强制 --no-ext-diff --no-textconv）④memory_search 低权威免责前置。
- **测试**：Go +5（保序/共存回归锁成对/观察三态）+键名适配。
- **门禁**：go build/vet 0+全量 ci 绿 EXIT=0+版本三处 4.375.0。
- **坑**：①批次分区改动必须带共存收益回归锁（只测拆批不测共存=延迟优化静默回退）②观察类机制防「假装观察过」——观察点绑真实 I/O 成功路径。
- **产物**：exe 51011072 B SHA256=88499e001577a49b184f8e754a1f1a879e38c13e3affe86dab9572e48c3e09a6（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.370.exe。
- **文档**：蒸馏文档留池四项销号/辨伪回填+releases/v4.375.0.md+CHANGELOG/README+releases/README（计数 396→397+34 席插 v4.375 裁 v4.341）+AGENTS 迁 1 插 1（九十迁：v4.372 入 archive）+progress/todos。

## 最新发布：v4.374.0（2026-09-21）「Reasonix 真身蒸馏：安全四刀 + 内核三刀」

- **动机**：用户纠偏 v4.373 对象认定（dsh 非 reasonix）并指路 GitHub 搜索——真身=esengine/DeepSeek-Reasonix（Go 单二进制终端 agent，V1.12/V1.15 端口出处），本版蒸馏 v1.15→v1.38.11 增量（5227 提交，源码入 clones/deepseek-reasonix）；v4.373 dsh 档勘误更名保留。
- **落地**：安全四刀=①Host 白名单 421（httpbridge+诊断端口，挡 DNS-rebinding）②只读表修正（env/awk/sed 移出只读、cargo 仅留 search、find -ok 族、git 四类执行旗标）③git clean-filter 中和（repo 本地 filter.* 置空三件套）④SSRF 共享件（netclient 三导出）+booksource 默认 client 守卫化。内核三刀=⑤收尾判定（reasoning-only=final；零内容冻结重放不注合成消息）⑥bash 输出运行中封顶（head 1MiB+滚动尾 64KiB）⑦compactStuck 新消息解锁（stuckAtMessages）。
- **测试**：Go +10（HostGuard 10 子例/BlockedInternalIP+GuardedClient/cleanFilterOverrides 真仓/间接执行 20 例/收尾两例/stuck 两态/bounded 三例）；行为变更适配 3 处。
- **门禁**：go build/vet 0+全量 ci 绿 EXIT=0+版本三处 4.374.0。
- **坑**：①蒸馏判据带上「上游机制的前提本地是否成立」（awk -f 例外依赖上游已记忆规则模型，gaea 自动放行无此前提）②安全刀落地先 grep 假站/httptest 用例 client 来源。
- **产物**：exe 51004928 B SHA256=85149421d0098b19dc28fbf66e4a0c7a5e52281fdf86b1e97b8ef35d9aed0a90（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.369.exe。
- **文档**：docs/gaea-reasonix-v138-distill-2026-09.md 新增+gaea-dsh-016 勘误更名+releases/v4.374.0.md+CHANGELOG/README+releases/README（计数 395→396+34 席插 v4.374 裁 v4.340）+AGENTS 迁 1 插 1（八十九迁：v4.371 入 archive）+progress/todos。

## 最新发布：v4.373.0（2026-09-21）「Reasonix (dsh) 0.1.6 内核蒸馏：溢出自愈 + 缓存对齐摘要等五刀」

- **动机**：用户口径「继续优化迭代 gaea，本次优化进行蒸馏最新版的 reasonix」——对 /c/AI/deepseek-harness（dsh v0.1.6-alpha.2，上游同步 2026-09-19）三域全量侦察后落地五刀，全 Go 内核，绑定面 707 零变更。
- **落地**：①上下文溢出自愈（IsContextOverflow 七措辞分类器+剪枝+强制压缩+rewrite version 验持久进展+有界重试，连续上限 1）②缓存对齐摘要（摘要请求=会话 system+被折叠区间逐字重放+指令作末条 user 消息，命中热 KV 缓存；renderTranscript 退役）③shrink 硬校验（摘要必须小于被折叠区间否则退机械摘要）④force 多轮压缩（压缩后重测仍超再压一轮，空转不计入 compactStuck）⑤repeat 分级提醒（[3,5,8] 三档递进+deny 计数+插话重置）。
- **测试**：Go +7（IsContextOverflow 9 子例/溢出自愈四路/摘要请求前缀逐字稳定/shrink 两路/force 有界/repeat 三档）；summaryInstruction 快照断言适配。
- **门禁**：go build/vet 0+agent/provider 全包绿+全量 ci.ps1 绿 EXIT=0+drift OK@707+版本三处 4.373.0。
- **坑**：①折叠区含旧 digest（kept）时重放到第一个 kept 消息即与上一请求分歧——system+tools 最贵前缀仍全命中，部分对齐可接受②无进展判定必须看 session.RewriteVersion 而非 compact 返回值（compact 对无可折叠区返回 nil，按返回值判会把空转当成功）。
- **产物**：exe 50995712 B SHA256=363b40e7450e95a6a9089bfe7d3900c9edbb791fb50a2fc163b0098c0f29ac30（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.368.exe。
- **文档**：docs/gaea-dsh-016-distill-2026-09.md（侦察+五刀+留池 12 项）+releases/v4.373.0.md+CHANGELOG/README+releases/README（计数 394→395+34 席插 v4.373 裁 v4.339）+AGENTS 迁 1 插 1（八十八迁：v4.370 入 archive）+progress/todos。

## 最新发布：v4.372.0（2026-09-21）「Gantt 缩放性能：Ctrl+滚轮 rAF 合并」

- **动机**：用户口径「继续」——性能留池「Gantt 行组件化」的可安全子集落地：Ctrl+滚轮缩放的 wheel 事件 rAF 合并（一帧最多一次 setDayW，终态一致）。schedule 1 源文件+1 测试适配，绑定面 707 不动。
- **落地**：GanttView 的 Ctrl+滚轮缩放路径——高分辨率触控板每秒 60~120 次 wheel 事件，原实现每次事件立即 setDayW→三窗格（横道/表格/时间轴）全量重渲染同频发生。改 rAF 合并——帧内累积缩放因子（pendingW），每帧最多提交一次 setDayWAt；锚点（缩放时光标处对应工程日）以累积后的目标日宽换算，滚动位置恢复与逐事件处理终态一致。清理路径补 cancelAnimationFrame。「GanttBars 行组件化+dayW 走 CSS transform」的结构性改造仍留池：条形定位从 dayW 像素改为 scale 变换涉及拖拽坐标换算（鼠标位置→日期）重写，属核心交互高风险区，需专门的拖拽真机回归轮。
- **测试**：GanttView 测试适配——缩放断言等待 rAF 帧提交（jsdom 下 rAF 为 16ms 模拟 setTimeout），33 例全绿；drag.test 7 例绿。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例；首跑 BookHealthPanel 1 例 flaky 与本轮足迹零交叠，单跑 7/7 复绿后全量复跑 OK）+drift OK@707+版本三处 4.372.0。
- **坑**：①rAF 合并后同步断言失效——jsdom 下 rAF 为 16ms 模拟 setTimeout，fireEvent.wheel 后同步断言拿到的还是旧值，测试需 await act 等待帧提交；②it 回调补 async 时易引连环括号错误——手工补 async( 后遗漏原闭合括号层级，esbuild 语法错误逐层暴露；脚本化改写测试文件时优先整段重写而非点状替换。
- **产物**：exe 50,988,032B SHA256=cf94c096112e9c177f6e46f040fe315acf82e5ebc3fb5b4a765ee78057bab7b8（releases/gaea-v4.372.0.exe+SHA256SUMS-v4.372.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.368~v4.372 删 v4.367.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.372.0.md+CHANGELOG/README+releases/README（计数 393→394+34 席插 v4.372 裁 v4.338）+AGENTS 迁 1 插 1（八十七迁：v4.369 入 archive）+progress/todos（Gantt 缩放 rAF 合并销号；行组件化+transform 结构改造留池）。

## 最新发布：v4.371.0（2026-09-21）「聊天历史分页绑定（游标式，绑定面 706→707）」

- **动机**：用户口径「继续」——性能留池「聊天历史分页绑定」设计并落地：大话题切会话不再全量拉取，首屏最新 200 条+向上翻页游标续拉。Go 3 源文件+1 测试+前端 6 源文件。
- **契约**：store 层 ListMessagesPage(topicID, limit, beforeSeq)——beforeSeq<=0 从最新一条向前取 limit 条，否则取 seq<beforeSeq 的最新 limit 条；返回升序消息+hasMore（取 limit+1 条探测后丢弃多余的）。App 层 ChatMessagesPage 绑定返回 {messages, hasMore, oldestSeq}（oldestSeq=本批最早 seq，前端下一次游标）；ChatB 门面+gen_bindings 再生（合计 707 方法→11 门面）+bindingNames/spaceBindings(shared) 手工同步+spaceBindings.test 数量 529→530。
- **前端**：useChatTopics.loadTopic 首屏改拉 ChatMessagesPage(id, 200, 0)，记录 hasMore/oldestSeq；新增 loadOlder（每次 120 条 prepend 到消息头部）；MessageList 接 hasOlder/loadingOlder/onLoadOlder/prepend 四 props——滚动宿主接近顶部且窗口已覆盖全部已加载消息时触发 loadOlder。**关键交互**：头部追加会改变 messages[0].key，沿用既有 anchor 重置逻辑会把用户弹回尾部——新增 prepend 通知（{id 递增, count}）让 MessageList 区分「分页头部追加」（窗口随新增条数扩大）与「切话题整体替换」（重置窗口）。全量路径 ChatMessagesList 保留不动（导出 Markdown 等场景）。
- **测试**：定向 +1（Go TestListMessagesPage——首屏截断/hasMore/游标续拉/拉到底四场景）+前端 ChatPage.test mock 接线（ChatMessagesPage 返回 page 形状镜像消息，15 例全绿）。
- **门禁**：go build/vet 0+internal/chat 全量绿+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例）+bindings drift OK@707+版本三处 4.371.0。
- **坑**：①头部追加 vs 整体替换的窗口语义——MessageList 的 anchor 重置逻辑原本不区分「切话题」与「分页头部追加」，直接接分页会把用户弹回尾部；②mock 契约必须跟随签名演进——ChatMessagesList 返回数组而 ChatMessagesPage 返回 page 对象，测试 mock 若只镜像数组形态，loadTopic 的 page?.messages 提取会静默得到空。
- **产物**：exe 50,987,520B SHA256=59cfc300918fd55521a58ad8619884ecf0441795cda15a81fd84b607da19a200（releases/gaea-v4.371.0.exe+SHA256SUMS-v4.371.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.367~v4.371 删 v4.366.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.371.0.md+CHANGELOG/README+releases/README（计数 392→393+34 席插 v4.371 裁 v4.337）+AGENTS 迁 1 插 1（八十六迁：v4.368 入 archive）+progress/todos（性能八项消一项：分页绑定收官——余项均需更深的契约设计）。

## 最新发布：v4.370.0（2026-09-21）「工程健康轮：依赖漏洞清零 + Go 工具链升级 + 构建管线修正」

- **动机**：用户口径「继续」——十轮快跑后工程健康整固：govulncheck+npm audit 双扫描按发现修复。Go 1 源文件+依赖升级+构建脚本修正，绑定面 706 不动。
- **扫描与处置**：①Go 标准库 5 处（net/url 二次复杂度/crypto-tls 握手/net-http ReadHeaderTimeout/encoding-xml·asn1 递归）→工具链 1.26.5→1.26.6（go.mod toolchain 指令，构建自动拉取，exe 实测 go1.26.6）；②golang.org/x/image v0.40→v0.46（webp/bmp 解码 panic、VP8L 越界内存分配 4 处——图片解码是用户可控输入面，连带 x/sys/x/text）；③excelize GO-2026-6452 恶意 xlsx panic（上游 Fixed in: N/A）→应用层防线：xlsxpreview Render/NeedsRecalc 入口 defer recover 转错误「xlsx 预览失败（文档结构异常）」（上游修复后可移除），govulncheck 复扫 11→1 仅剩此条；④npm @vitest/mocker 2 moderate（路径遍历，dev-only）→vitest ^4.1.10→4.1.11，audit 归零；⑤**新增 pnpm-lock.yaml**（前端依赖历史上从未锁定，vitest 4.1.11 锁定，CI 实测兼容）。
- **构建管线修正（在册坑落地）**：build.bat 的 wails 命令此前未带 -s（跳过前端），wails 内部隐式 npm install 在 lockfile 缺失+peer 依赖冲突（eslint-plugin-prettier ^4.2.5）时直接失败——本轮 vitest 升级触碰 package.json 即暴露。按在册坑配方修正 build.bat：先 `npm.cmd run build` 显式构建前端，再 `wails build -s`（wails 捕获前端输出走管道会挂起+隐式 install 环境敏感，AGENTS 在册坑原文配方）。
- **门禁**：govulncheck 复扫 11→1+npm audit 归零+go build/vet 0+office 包全量绿+全量 ci.ps1 绿 EXIT=0（vitest 388 文件 3315 例，vitest 4.1.11 兼容）+drift OK@706+版本三处 4.370.0。
- **坑**：①pnpm 项目里 npm install 会生成 package-lock.json 形成双轨——发现后立即删除并 pnpm install 恢复单轨；②lockfile 缺失是环境敏感炸弹——任何依赖树变动触发重新解析，peer 冲突在 wails 隐式 install 中爆炸，lockfile 入库后不再复现；③wails 隐式 npm install 依赖环境解析成功——显式前端构建+-s 是在册坑配方，build.bat 此前未遵守本轮落地。
- **产物**：exe 50,979,328B SHA256=785a740a681d410dfff9c8744d09cb37698ca6d756e0808f31b42edbae8de95f（releases/gaea-v4.370.0.exe+SHA256SUMS-v4.370.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过；go1.26.6 构建）；保留策略 5 版留 v4.366~v4.370 删 v4.365.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.370.0.md+CHANGELOG/README+releases/README（计数 391→392+34 席插 v4.370 裁 v4.336）+AGENTS 迁 1 插 1（八十五迁：v4.367 入 archive）+progress/todos（excelize 上游未修留观）。

## 最新发布：v4.369.0（2026-09-21）「真机走查班第二班：流式分段渲染真机验收 + --w-ink-3 重指向」

- **动机**：用户口径「继续」——走查班第二班：v4.366~v4.368 三轮（聊天组件 memo 化/编辑输入性能/流式分段渲染）真机验收；顺手落地别名重指向小刀 --w-ink-3。前端 1 css 文件，绑定面 706 不动。
- **走查结论**：①流式分段渲染链路真机验证通过——聊天页新建会话→发送短消息→模型正常回复（「我是一个热心且博学的AI助手。」）→回复正常落库与渲染，终态无错误、无 file:// 残留、无 console 告警；分段切分契约由 findStableCut 6 例单测锁定。②--w-ink-3 重指向目检通过——launcher 局部令牌实测 #5b6472 = --md-sys-color-text-tertiary（亮态），别名重指向两目收官。③环境事实：走查观察到的「请求超时」历史消息系用户环境模型服务当时不可用所致，非本轮回归。
- **落地**：module-launcher.css `--w-ink-3` 从 `color-mix(in srgb, var(--color-text) 46%, transparent)` 重指向 `var(--md-sys-color-text-tertiary)`（实色随主题令牌走），`.ml-avatar-user` 二次混色底 12%→14% 微调对齐观感——「次级文本六别名收敛」的最后一目收官。
- **清场**：走查用的测试消息对（user+assistant）经 chat.db 按 id 精准删除（删前 SELECT 确认、按主键删，会话恢复原有 6 条消息）；走查脚本/截图清理；杀壳。用户既有会话与真实工程零触碰。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例）+drift OK@706+版本三处 4.369.0。
- **坑**：①CDP 目检读局部令牌要在定义作用域元素上读——--w-ink-3 定义在 .ml 选择器下（板块局部令牌层），从 documentElement 读恒为空串；②走查发消息会写用户真实库——聊天无消息级删除绑定，测试消息只能按 id 从 chat.db 精准删除；后续走查若需验证流式，先建专用测试会话再删除整个会话（ChatTopicDelete），避免污染用户会话。
- **产物**：exe 50,951,680B SHA256=26f0293b4b08e03f70d9220093872c3eff335ed72642116d341ec6029fbdfc50（releases/gaea-v4.369.0.exe+SHA256SUMS-v4.369.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.365~v4.369 删 v4.364.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.369.0.md+CHANGELOG/README+releases/README（计数 390→391+34 席插 v4.369 裁 v4.335）+AGENTS 迁 1 插 1（八十四迁：v4.366 入 archive）+progress/todos（别名重指向两目收官销号）。

## 最新发布：v4.368.0（2026-09-21）「性能轮深水区：聊天流式分段渲染 O(n²) 根修」

- **动机**：用户口径「继续」——性能留池最大项落地。前端 3 源文件+1 新测试，绑定面 706 不动。
- **问题**：聊天流式回复时每个 delta chunk 都对全量累积文本重跑 react-markdown（remarkGfm+remarkMath+rehypeKatex），O(n²)——5KB 尾段单次 parse 5~20ms × 30+ chunk/s = 主线程持续高占用与 GC 压力。
- **方案**：三选一定方案 A（ChatRow 流式分支分段，影响面锁死一处）。方案 B（MarkdownContent 内部分段）会冲击 t74 测试精确的 ReactMarkdown 调用计数断言且波及非流式消费方；限频只降频率不降单次 parse 规模。
- **落地**：①findStableCut（gaea/components/MemoMarkdown.tsx）导出+盲区修复——原只扫切点后 suffix 判断 fence 状态，fence body 含空行（代码极常见）时计数偶数误判可切、把代码块拦腰切断（稳定段含悬挂 fence+尾段代码行被当正文的瞬时错排）；改全量行扫描候选切点前 fence 状态，切点在未闭合 fence 内回退到打开行前（切分更保守，稳定段永不含悬挂 fence）；既有 MemoMarkdown 测试零改动通过。②ChatRow 流式分支 stable+pending 两段渲染——stable 段字符串不变时 react-markdown 子树被 memo 整体跳过，pending 段（通常 KB 级）每帧重解析；终态回既有路径零变化。
- **契约与兼容**：genui 零破坏（fence 当普通 fence，闭合落稳定段；流式分支本就不传 overrides——流式中 genui 显示原始 JSON 面板现状不变）；已知接受折衷=松散列表流式序号短暂重排终态自愈+streaming 翻转 DOM 重建一次（与 gaea 同款）；相比 gaea 原设计改进=尾段走完整 react-markdown 管线，未闭合 fence 流式中显示为持续增长代码块更接近终态。
- **门禁**：定向 +6（findStableCut.test.ts：无空行全不稳定/多段落切分/切点后未闭合 fence 留尾/fence body 含空行不拦腰〔盲区修复〕/闭合 fence 入稳定段/空文本）+MemoMarkdown 既有 2 例零改动通过+ChatMarkdown.genui/聊天组件/genui.walkthrough 回归全绿+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例）+drift OK@706+版本三处 4.368.0。
- **坑**：①正则整段替换函数体非贪婪匹配会吞闭合括号——替换串忘了带 `}`，tsc 当场暴露；②性能改造的行为等价是「终态等价」——流式中间态 DOM 从单块变两块（stable+pending），验收口径=终态一致+中间态不劣化，不能拿中间态 DOM 逐字节比。
- **产物**：exe 50,951,680B SHA256=e1bc4358584f6c0c70dacb2c9ac18cd545dd831f8dc6157a9b3d88b7cf2849a0（releases/gaea-v4.368.0.exe+SHA256SUMS-v4.368.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.364~v4.368 删 v4.363.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.368.0.md+CHANGELOG/README+releases/README（计数 389→390+34 席插 v4.368 裁 v4.334）+AGENTS 迁 1 插 1（八十三迁：v4.365 入 archive）+progress/todos（性能八项消一项：流式分段渲染收官——余七项）。

## 最新发布：v4.367.0（2026-09-21）「编辑输入性能：章节编辑 memo 化 + 设定页预览/统计防抖」

- **动机**：用户口径「继续」——性能轮延续：编辑输入链路每击键全页重渲染与全文重解析治理，全部行为等价。前端 4 源文件+1 新 hook，绑定面 706 不动。
- **①ChapterEditor memo+updateTab useCallback**：章节编辑每击键 onUpdate('scenes')+'saved' 两次 setTabs 使 959 行 ChapterPage 整页重渲染，未 memo 的编辑区全场景框陪跑。修=updateTab 改 useCallback（依赖 activeKey）稳定 onUpdate 引用；ChapterEditor 包 React.memo——父页无关 state（focus/ghost/其它 tab）不再重渲染编辑区。
- **②ChapterPage 上报防抖**：novel:chapter-active 事件 effect 依赖 activeTab（每击键引用变），每次全文 join+countTextChars 并派发事件拖动壳层 inspector 重渲染。修=250ms 防抖（cleanup clearTimeout），输入期间不付全文遍历，停顿后上报一次最终值。
- **③NovelSettingPage 预览/字数防抖**：分屏/渲染模式每击键对全文跑 react-markdown+KaTeX 重解析（10k 字约 10~30ms/键，输入明显卡顿）；wordCount 每键 trim+countTextChars 全文扫描。修=新增 useDebouncedValue 通用 hook（300ms），MarkdownContent 与字数统计消费防抖镜像值；编辑器保持逐键受控（输入零延迟）。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3309 例；ChapterPage/NovelSettingPage/novel 组件 187 例定向全绿）+drift OK@706+版本三处 4.367.0。
- **坑**：①跨层字符串转义地狱——python 补丁多层传递中 \n 被吃成真实换行写进源码（TS Unterminated string literal），构造含反斜杠字面量用 chr(92) 拼接彻底避开转义层级；②防抖的正确姿势是「镜像值」而非「延迟源」——编辑器保持逐键受控（输入零延迟），只有下游重计算（Markdown 解析/字数）消费防抖镜像；防抖放 onChange 上游=输入回显延迟，那是 bug 不是优化。
- **产物**：exe 50,951,680B SHA256=177331f52125c531ba4096e7e2e121cc1e653083d77912ebce6d935d2be8ac38（releases/gaea-v4.367.0.exe+SHA256SUMS-v4.367.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.363~v4.367 删 v4.362.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.367.0.md+CHANGELOG/README+releases/README（计数 388→389+34 席插 v4.367 裁 v4.333）+AGENTS 迁 1 插 1（八十二迁：v4.364 入 archive）+progress/todos（性能八项再消两目——ChapterEditor 受控下沉被 memo+防抖等效覆盖大半，下沉本体留池）。

## 最新发布：v4.366.0（2026-09-21）「性能轮收官：聊天组件 memo 化专项 + StoryStream 行级 memo」

- **动机**：用户口径「继续」——性能八项留池中「聊天组件 memo 化」的 props 链梳理完成，落地收官；StoryStream 行级 memo 同步根修。前端 7 源文件+1 守卫适配，绑定面 706 不动。
- **刀1 聊天组件 memo 化**：ChatPage 五个内联箭头回调 useCallback 化（handleToggleSearch/handleToggleForceSearch/handleToggleThinking/handleOpenVoiceSettings/handleSwitchPersona）；`quickReplies={mode !== 'plain' ? QUICK_REPLIES : []}` 的内联 [] 提模块级 EMPTY_REPLIES 常量（内联数组是 memo 恒失效典型来源）；四组件（ChatModeBar/ChatPersonaBar/ChatComposer/ChatInspector）React.memo 包裹（XInner 声明+文件尾 memo 导出）；ChatInspector 统计三遍全文遍历合并单趟 useMemo。memo 语义安全：props 梳理不完整只损失收益无正确性风险。
- **刀2 StoryStream 行级 memo**：抽 SinAssistantRow memo 行组件（storyId/m/两插图回调为 props），parseStorySegments 下沉行内。前提核实：useSinStory 流式 patch（prev.map 只换流式行引用）+showNotice/setIllustration 均 useCallback 稳定。效果：流式期间每 delta 只重渲染流式中的那一行（原 O(消息数×分段)/token 且 keepAlive 后台照跑）。
- **守卫适配**：E16（frontend-e-check.mjs）QUICK_REPLIES 作用域断言按 memo 化声明形态适配（ChatComposerInner 或 memo 导出取 max 位置）——守卫意图（常量在组件之前）不变。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3309 例+E16 PASS）+drift OK@706+版本三处 4.366.0。
- **坑**：①改组件导出形态先查源断言守卫——E16 按 `export const ChatComposer:` 文本定位组件声明，memo 化改名后守卫失明；守卫与被守代码同 commit 演进但断言意图不变；②memo 化顺序纪律——先稳定 props（useCallback/常量）再包 memo，顺序反了 memo 全部空转且无报错只有性能损失。
- **产物**：exe 50,951,168B SHA256=a6d7be9a917339e4d010ceb17fa3b10c594f4225f845a0f9e3b2fbaa222b10c1（releases/gaea-v4.366.0.exe+SHA256SUMS-v4.366.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.362~v4.366 删 v4.361.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.366.0.md+CHANGELOG/README+releases/README（计数 387→388+34 席插 v4.366 裁 v4.332）+AGENTS 迁 1 插 1（八十一迁：v4.363 入 archive）+progress/todos（性能八项消两项：聊天组件 memo 化+StoryStream 行级 memo）。

## 最新发布：v4.365.0（2026-09-21）「前端性能轮：流式节流 + 后台空转治理 + 缓存统一」

- **动机**：用户口径「继续」——前端换新镜头性能审计（三路：React 渲染性能/长列表与流式渲染/桥接与数据面），全落行为等价低风险刀。前端 22 源文件+1 测试，绑定面 706 不动。
- **线1 流式节流**：useChatStream delta 改 buffer+rAF flush——原每 chunk 一次 setState 使 ChatPage 以 30~120/s 全页重渲染+吸底强制 reflow；四条终态路径（done/error/超时/启动失败）flush 前 cancelPendingDelta 防挂起帧污染已清空文本。角色模式打字机 setTimeout(14ms)≈71 次/秒改 rAF 时间驱动（每 14ms 推进 step 字的节奏不变）。ChatPanel 打字机 12ms/字符且每步 O(n) slice 改 rAF 每帧 3 字符。
- **线2 后台空转治理**：keepAlive 13 页 display:none 常驻——五个 canvas rAF 循环（ParticleFlow/SoundWaveOverlay/CompanionAvatar/VoiceChatOrb/RelationGraph）加 `document.hidden || offsetParent===null` 挂起守卫（保留 rAF 链恢复自动续绘）；DagPanel/AgentTree/useComfyTaskProgress/useImageGenQueue/SinIllustration 轮询秒表加 hidden 跳过（时间值基于 start 计算恢复自动正确）；TaskCenter onTaskEvent 接 gate；AIConsole 隐藏丢事件。
- **线3 渲染链**：MainLayout（根壳）/HomePage/ModelCenterPage/AppearancePanel(8 处) 整 store 订阅改逐字段选择器——任意 store set() 不再触发根壳与全部 keepAlive 页元素重渲染；MemoryHubPage 检索 hits key 稳定化。
- **线4 缓存统一**：usePortraitUrl 导出 getCachedAttachmentDataURL（共享成功缓存+方法缺失防御）；Message/FileThumb/inspector 三处直调接线；粘贴/截图落盘后复用内存 dataUrl 作预览（原三重 base64 往返）。
- **线5 AOA 几何**：nodeById/anchorById/taskNameById 查找表 useMemo（taskName 原每边线性扫 tasks O(E×T)/帧）；箭线几何管线整段提升组件级 useMemo（缩放/选中变化不再全量重算，zoom 走外层 scale 与几何无关）；hooks 全部移 early return 前。
- **顺手修真 bug**：AIConsole 违反 v4.62.2 EventsOff 事故纪律（直接摸 window.runtime 且卸载 EventsOff 全清通道）改 subscribeWailsEvent 唯一入口；TaskCenter 事件通道 gate 漏接接上。
- **门禁**：定向 +4 源断言（perf-guards）+tsc/eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3309 例）+drift OK@706+版本三处 4.365.0。
- **留池**（行为级/需设计，审计已给证据）：流式分段渲染/聊天真虚拟化/StoryStream 行级 memo/ChapterEditor 受控下沉/Gantt 行组件化/聊天历史与造价库分页绑定/base64→blob URL/聊天组件 memo 化（props 链需梳理）。
- **坑**：①组件 memo 非免费午餐——内联 props 使 memo 恒失效，须先梳 props 链；②rules-of-hooks 硬约束——提升 useMemo 连同 manual/shown 计算移 early return 前（eslint 当场拦）；③测试 mock 缺方法时可选链与直接调语义差——共享工具要有方法缺失防御（ContextView 空渲染 5s 超时教训）；④整 store 订阅代价被 keepAlive 放大——根壳粗订阅优先治理。
- **产物**：exe 50,951,168B SHA256=c2d09a937841630a46052517ea750cd16ac975d26e242d026c06f26296653948（releases/gaea-v4.365.0.exe+SHA256SUMS-v4.365.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.361~v4.365 删 v4.360.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.365.0.md+CHANGELOG/README+releases/README（计数 386→387+34 席插 v4.365 裁 v4.331）+AGENTS 迁 1 插 1（八十迁：v4.362 入 archive）+progress/todos（性能大工程八项挂池）。

## 最新发布：v4.364.0（2026-09-21）「真机走查班：v4.361~v4.363 前端三轮验收 + withinReadRoots 绘梦图误伤修复」

- **动机**：用户口径「继续」（授权闲置窗口）——按 v4.355 先例前端优化三轮后开真机走查班（CDP 9333 附着 v4.363.0 壳，只读纪律），走查拽出 v4.358 读侧收口真机才现形的误伤，修复发版。Go 1 源文件+1 测试，绑定面 706 不动。
- **走查结论**：双空间 14 板块（工作 6+闲庭 7+设置/进度计划 navigate 直达）错误边界 0/空白 0/console.error 0/exception 0/`img[src^="file://"]` 残留 0。深验=①聊天 PersonaPicker popover「31 个可聊天角色」26 头像全渲染零裂图零 file://（v4.355 报 27 条告警同场景，v4.361 头像收口确证）②schedule `.sched-row-no` 行号实测 rgb(74,64,96)=Violet 亮态 onSurfaceVariant #4a4060（v4.361 收口生效）③tertiary 双态 #5b6472↔#8b93a0 明暗切换正确④longtask 走查全程仅 1 条 800ms 真长任务（阈值 200ms 后 69~165ms 噪音归零）。
- **修复 withinReadRoots 遗漏图片保存目录**：走查日志 6 条 `[AttachmentDataURLError]`（Pictures\gaea 绘梦历史图被拒）——v4.358 论断「绘梦资产均落两根内」不成立：绘梦图实际落 cfg.ImageSaveDir（默认 %USERPROFILE%\Pictures\gaea）。修=withinReadRoots 改 App 方法补第三根（ImageSaveDir 非空时+默认兜底，与 image_handler 落盘口径一致）。真机复验同一张被拒图成功读回 2.89MB data URL。定向 +1=TestWithinReadRootsCoversImageSaveDir（配置根命中+兜底根纯前缀断言不写用户真实目录）。顺手修=方法化后 &App{} 裸构造既有测试 TestGaeaAttachmentRoundTrip panic（nil core 解引用）加 a.core!=nil 守卫回归绿。
- **门禁**：定向 +1+internal/app 全量绿（102s）+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3305 例）+drift OK@706+版本三处 4.364.0。
- **坑**：①mock/单测测不出路径语义——v4.358 白名单论断没经真机验证就写进注释，读侧白名单变更必须真机过一遍真实数据路径；②走查拽错要看壳日志不只看页面——AttachmentDataURL 失败前端有占位兜底页面全绿，问题只在壳日志里；③包级函数改方法先 grep 裸构造测试（&App{} 是合法测试惯用形，方法化后嵌入指针解引用即 panic）；④CDP 探针传 Windows 路径的转义陷阱（多层引号反斜杠被 JS 转义吃掉、 成 formfeed），用 join(String.fromCharCode(92)) 或正斜杠（Windows API 皆收）。
- **产物**：exe 50,948,096B SHA256=e026857b8c74d1667b9d5606267d93bffd94d03a3678008073deca4d4f76b976（releases/gaea-v4.364.0.exe+SHA256SUMS-v4.364.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.360~v4.364 删 v4.359.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.364.0.md+CHANGELOG/README+releases/README（计数 385→386+34 席插 v4.364 裁 v4.330）+AGENTS 迁 1 插 1（七十九迁：v4.361 入 archive）+progress/todos（走查班清账+误伤修复销号）。

## 最新发布：v4.363.0（2026-09-21）「后端留池终审：ConvertToPdf 读侧收口落地 + prompt 双构造辨伪关闭」

- **动机**：用户口径「继续」——后端挂池仅剩 2 项（均标「需设计」）逐项终审：一项设计落地、一项证据辨伪关闭，**后端挂池清零**。Go 1 源文件+1 测试+前端 1 小刀，绑定面 706 不动。
- **刀1 ConvertToPdf 读侧收口**（挂池设计项落地）：取证确认前端生产调用点仅 FilePreviewModal exportPdf 一处，三类路径来源（工作区相对/uploads 绝对/对话框工作区外素材）全部被现有机制覆盖。收口=①相对路径 Clean 拒 .. 穿越（原 `filepath.Join(gaeaCwd(), rel)` Clean 掉 .. 可逃逸工作区，对齐 GaeaReadFile 口径）②统一门 `!withinReadRoots(path) && !isPickedFile(path)` fail-closed（对话框素材经 GaeaPickFiles 登记时已入表，isPickedFile 现成覆盖——唯一真·合法绝对路径场景无需新设计）；写侧 exports 目录/文件名全服务端拼装无越界。三类合法调用方零误伤；前端 exportPdf 已有 toast 错误呈现零改动；内存登记表重启即清=重启后历史会话工作区外附件导出被拒，fail-closed 可接受。定向 +1=TestConvertToPdfPathGuard（相对穿越拒绝/未登记绝对路径拒绝/登记路径过守卫/工作区路径过守卫，四场景零外部进程）。
- **刀2 prompt 双构造辨伪关闭**（不修，证据留档）：复核「双构造」实为 New() 构造后立即被 SetPromptFS 换新（main.go:32→34 同步单线程、Startup/httpbridge 全部之前）——第一实例存活期零读者，无正确性影响；换新是 embed FS 注入的唯一无锁安全手段（prompt.Engine 显式无锁契约 override 只写一次）；收益=省一次 20 文件/92KB 解析≈个位数毫秒一次性 vs 成本=改 New 签名（波及测试/CLI）或 Engine 加锁（违反契约）+绑定生成面三处登记——风险>收益辨伪关闭。日后低风险形态=New() 可选 embed-FS 参数+SetPromptFS 幂等短路，排池尾不单独开刀。
- **刀3 前端小刀**：MemoryHubPage 总览读取失败 rail 计数显「—」占位（title 提示，恢复自动清除；原 catch 静默空缺）。
- **门禁**：定向 +1+internal/app 全量绿（99.9s）+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3305 例）+drift OK@706+版本三处 4.363.0。
- **坑**：①收口测试的「放行」断言不能借真实转换——.docx 在装了 LibreOffice 的测试机会真转成功（err==nil）断言失效，借「.doc 显式拒绝」这类转换前确定性分支验证过门，四场景零外部进程；②辨伪关闭也要证据链写进留档（风险侧+收益侧），下次取证不必重查。
- **产物**：exe 50,947,584B SHA256=5a2bdafa10858b913df40604d9428be5adb3969dfeae8205eecf274a5b68b0fa（releases/gaea-v4.363.0.exe+SHA256SUMS-v4.363.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.359~v4.363 删 v4.358.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.363.0.md+CHANGELOG/README+releases/README（计数 384→385+34 席插 v4.363 裁 v4.329）+AGENTS 迁 1 插 1（七十八迁：v4.360 入 archive）+progress/todos（后端挂池清零销号）。

## 最新发布：v4.362.0（2026-09-21）「前端优化轮第二弹：吞错可见化收尾 + tertiary 文本令牌收敛」

- **动机**：用户口径「继续」——v4.361 前端优化轮余量再取证：镜头B 剩余吞错项全量复核 + 次级文本别名收敛影响面，两路并行只读后定刀。纯前端 17 源文件+2 测试+3 语言文件，绑定面 706 不动。
- **刀线一 吞错可见化收尾（12 刀）**：①**OfficePanel 读失败防覆盖**（置顶数据覆盖风险——gaeaSettings 失败被吞后草稿留全默认值 permMode 'ask'/sandboxBash 'enforce'，用户点保存就把默认配置覆盖真实引擎 config.toml，NovelSettingPage 同族）→loadFailed+Alert 重试+保存禁用+handleSave 入口拦截，定向 +2；②**「删除假成功」家族 5 刀**（PriceSourcesRepository/PriceSourcesPanel〔加载失败空面板→错误态+重试〕/CostProjectsView 删明细行/CostProjectsView restoreVersion 旧行删除失败中止恢复〔原吞错继续插新行=删半截+重复行〕/CostNotesView）——失败 message/toast 且不推进「已删」语义；③OfficeMemoryLibrary doMergeAll 串行计数分报，失败对保留可重试；④useChatTopics 初始化失败一次性 message.error（原只进后端日志，侧栏空白像历史全丢）；⑤ProgrammingPage 状态轮询首败 setError 复用 prog-error 横幅成功即清（原永久「检测中…」假加载）；⑥ModelSwitcher 失败态「读取失败点击重试」与「未配置任何模型」分流；⑦TaskInboxPanel 状态变更失败提示；⑧小刀 3 处（ChatPanel 人格清单/AboutPanel 版本「v—」→「版本读取失败」/useSinStory 自动命名 setNotice）。低优挂账=MemoryHubPage 左轨计数「—」占位（空缺非假成功）。
- **刀线二 tertiary 文本令牌收敛**：ThemeTokens 新增 colorTextTertiary 正式字段（6 暗工厂 #8b93a0 原值平移/6 亮工厂 #5b6472——原 App.tsx 三别名硬编码 #6b7280 不在 lightFn，Violet 亮 surface 实测 4.41:1 不达 AA；#5b6472 全 surface ≥5.45 连 Violet 容器高层 4.65 过线；不取 TERTIARY 表——暖容器配对色不当正文中性灰）；App.tsx 三别名（--md-sys-color-text-tertiary/--v3-fg-soft/--color-text-tertiary）改由 effTokens 下发**变量名全保留**=25 消费点+softTextStyle 7 文件零改动；机检扩面=contrast BODY_TOKENS +tertiary（12 主题×2 背景 24 条新断言）+明暗锁值+App.tsx 防回流源断言。别名层裁决：--whisper-ink-muted/--fg-faint 纯转发保留；--fg-faint 千级消费重指向 tertiary 是独立视觉拍板项；--w-ink-3（color-mix 非纯转发）重指向需目检挂池。
- **门禁**：定向 +4+tsc 0+eslint 0 error 0 warn（新增 4 处 exhaustive-deps 全数补齐）+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3305 例）+bindings drift OK@706+版本三处 4.362.0。
- **坑**：①LocaleProvider 默认跟随 localStorage（gaea-lang），beforeAll loadLocale('zh') 不稳固——组件重渲染（失败态→重试成功）后 t() 翻回 en 中文断言飘，测试须 beforeEach setItem；②antd disabled Button 的 accessible name 在 jsdom 不稳定（getByRole 正则失配），按钮定位走 testid/textContent 直查；③zh-TW.ts 必须补齐全部 DictKey（类型从 en.ts 推导全键校验，tsc 当场拦）；④恢复类操作「先删后插」的吞错=删半截+重复行，失败要中止而非吞掉继续；⑤对比度选值要喂最差场景背景（Violet 亮 surface 4.41，不是白底 4.83）。
- **产物**：exe 50,946,048B SHA256=c26432852cdcb9211edf0c3e4bb6d6831a6be8a9dd57e09331276783aa3cd274（releases/gaea-v4.362.0.exe+SHA256SUMS-v4.362.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.358~v4.362 删 v4.357.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.362.0.md+CHANGELOG/README+releases/README（计数 383→384+34 席插 v4.362 裁 v4.328）+AGENTS 迁 1 插 1（七十七迁：v4.359 入 archive）+progress/todos。

## 最新发布：v4.361.0（2026-09-20）「前端优化轮：失败可见化 + 观察池清账 + 令牌卫生」

- **动机**：用户口径「优化gaea前端UI」——后端优化五连发后换前端镜头，三路并行只读审计（令牌一致性/交互三态覆盖/观察池取证）后定刀落地。纯前端 34 源文件+4 测试+2 新工具文件，绑定面 706 不动（零 Go 改动），零删功能（UI 简化红线）。
- **刀线一 失败可见化（7 刀）**：审计撞出 6 处「失败被伪装」缺陷——NovelSettingPage 读失败吞成空编辑器可覆盖保存真实设定（唯一真丢数据风险→loadFailed+Alert 重试+禁保存+Ctrl+S 拦截）；SearchModal 三路 `.catch(()=>null)` 后端全挂显示「未找到」（v4.350 漏网面→参与线全败才报错、单路降级；顺补存任务失败提示）；ConsistencyPanel 检查失败渲染绿色「全部通过」假阴性（→ruleError 横幅+「结果不可信」）；CostLibraryView batchStatus 吞错后无条件报成功（batchDelete v4.350 同函数漏网兄弟→计数口径+busy 闸）；CostLibraryPage 概览失败伪装「空库新手引导」（→全失败错误态+重试）；WeixinPage 提醒静默+首拉闪空态（→v4.351 范式+loadedOnce）；批量吞错 6 处（审核无反馈/开关静默回弹/清空失败消息复活/语音发送石沉大海/任务输出留白/语音设置假保存）。
- **刀线二 观察池清账（3 项全清）**：①file:// 头像——v4.355 走查 27 条 console 告警清零：PortraitImg 抽 usePortraitUrl hook（独立文件 react-refresh 合规+成功缓存防 popover 数百次绑定调用），PersonaPicker/AssetStudio×2/WeixinPage×4 残留直连全部接线（Avatar 场景失败自然回退首字）；②longtask 日志阈值 50→200ms（69~165ms 常规抖动不再刷屏，10s 窗口判定前置 continue）；③schedule 弱化文字——对比度普查遗留「视觉拍板项」落地：12 处把 outline（0.25 alpha 边框令牌）当文字色全换 on-surface-variant（亮态 1.3~1.8:1→≥6:1，有 v4.127 先例；border 用途合法保留），回归锁进 contrast-hex。
- **刀线三 令牌卫生**：softTextStyle 7 文件逐字重复→utils/uiStyles.ts 单源；情绪色 map 2 处重复→utils/emotionColors.ts（no-raw-hex 豁免名单随文件迁移登记）；novel-workspace #fff 反向兜底→currentColor；tailwind --transition 注明被 App.tsx 注入覆盖。
- **门禁**：定向 +6+eslint 0 error 0 warn+全量 ci.ps1 绿（vitest 386 文件 3301 例+E 系列守卫+卫生守卫）+drift OK@706+版本三处 4.361.0。
- **坑**：①「参与者全失败」合取式里不参与的路贡献 true（vacuous truth）——写反两次被定向测试当场抓出；②源断言行首锚定+令牌名后带标点（border-color/outline-variant 前缀误伤）；③no-raw-hex 豁免按文件名，色板迁新文件要同步登记；④hook 别与组件同文件（react-refresh 0 warn 门禁）；⑤select 测试按 option 文本定位、value 必须是 STATUSES 合法值否则 jsdom 静默回落空串短路 onChange。
- **产物**：exe 50,941,440B SHA256=a2c695b3b6b375c309fa6ecb3dac0942b31e8a49e00cea6c2c7e1484b0f9b5da（releases/gaea-v4.361.0.exe+SHA256SUMS-v4.361.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.357~v4.361 删 v4.356.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/v4.361.0.md+CHANGELOG/README+releases/README（计数 382→383+34 席插 v4.361 裁 v4.327）+AGENTS 迁 1 插 1（七十六迁：v4.358 入 archive，顺带补记七十二~七十五迁案行漏记）+progress/todos。

## 最新发布：v4.360.0（2026-09-20）「后端优化轮第五弹：挂池再清 3 刀——.tmp 卫生守卫 / 导入事务化 / 版本号原子化」

- **动机**：用户口径「继续」——v4.359 清完 v4.358 池后余量再取证：本轮落地 3 刀（含 v4.359 发现的 .tmp 病根本身入闸），绑定面 706 不动，前端零改动。
- **刀1 .tmp 卫生守卫**：scripts/clean-tmp.ps1 新增+ci.ps1 在设置 TMP/TEMP 重定向前接入——.tmp 超 512MB 阈值时只清「已知安全」瞬态模式（Test*/*.log/edge-*·walk-*·ui-sweep*·dsh-context* 旧诊断 profile/smoke-*.exe/go-build*），其余不动。**坑=.ps1 含非 ASCII 必须带 UTF-8 BOM**（check-docs 在册守卫④）——无 BOM 时 powershell.exe 按 GBK 解析，中文注释乱码破坏 param 行，阈值默认值静默失效变成「每次都清」（151MB 就触发），加 BOM 后语义恢复。
- **刀2 ImportProjectCharacters 整体事务化**（characterlib/store.go）：原每角色 Get+Upsert+Associate 三次独立 autocommit，百级导入=三百多次写，中途失败=部分导入。修法=抽 execer 接口（*sql.DB/*sql.Tx 公共 SQL 子集）+ getOn/prepareUpsert（校验+时间戳+剧照本地化，磁盘 IO 不入事务）/upsertOn（序列化+UPSERT）/associateOn 四助手拆分；Get/Upsert/Associate 改薄包装签名不变（绑定面零影响），导入循环 Begin→tx 循环→Commit，任一步失败 Rollback。既有 4 个 Import 测试零改动全绿=事务化等价性证明。
- **刀3 SaveVersion 版本号原子化+SchemaV22**（costproject/gaea db）：原「先 MAX 后 INSERT」两步无事务无约束，并发保存同项目落重复版本号。①INSERT 内 SELECT COALESCE(MAX(version),0)+1 自算——单语句在 SQLite 写锁下天然原子，读改写窗口消除；回读实际落库版本填返回值（与预估值并发错位以库为准）②SchemaV22 先清历史重复（bug 产物保留每组最新 rowid）再建 idx_cost_versions_proj_ver 唯一索引硬约束背书。定向 +1=TestSaveVersionSequenceAndUniqueIndex（序列 1,2 连续+直插重复被拒）。
- **留池仅剩 2 项**（均需设计）：prompt 引擎双构造（agent 构造捕获 eng 指针、SetPromptFS 必须先行的时序风险）/GaeaConvertToPdf 绝对路径（preview→convert 链有合法绝对路径用途，需 Pick/白名单统一设计）。
- **门禁**：go build/vet 0+受影响 4 包测试绿+全量 ci.ps1 绿（clean-tmp 步首次随 CI 运行；首跑因 Invoke-Native 插在函数定义前即插即炸，移到版本漂移闸门后修复）+bindings drift OK@706+版本三处 4.360.0。
- **坑**：①.ps1 无 BOM 的乱码是语义问题非显示问题——症状是「守卫每次都触发」而非报错②execer 接口（Exec+QueryRow 公共子集）是 Go 事务化标准姿势，磁盘 IO 留事务外③「单语句天然原子」优先于「包事务」——INSERT...SELECT MAX+1 零锁成本，事务留给多语句组合④ci.ps1 的 Invoke-Native 调用必须在其函数定义之后（插早了 CommandNotFoundException）。
- **产物**：exe 50936832B SHA256=3c42f42929ee734a49e4c3921800b9065719e14c2a56f3e661f4661f7f3abd58（时间戳 2026-09-20 22:13:30 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.356~v4.360 删 v4.355.exe。

## 最新发布：v4.359.0（2026-09-20）「后端优化轮第四弹：挂池清账——v4.358 审计辨伪项落地 7 刀」

- **动机**：用户口径「继续继续」——v4.358 三路审计留下的挂池逐项再取证定刀：可落地 7 刀全清，需设计/需重构/需复现的 4 项如实留池。纯 Go 7 源文件，绑定面 706 不动，前端零改动。
- **刀A SQL**：①SelfHeal 启动期 N+1 点查（repair.go）——分类树一次捞内存 map，pathResolves 逐条目逐段 QueryRow 改纯查表+补 entries rows.Err；②migrateLegacyDB（whisper db）——只拷主文件+wal（-shm 自动重建不拷）、.migrating 临时文件+rename 原子落位+残留 log.Printf 收口 slog（vet 参数错位同修）；③FTS 降级 LIKE 多列长 OR 链（fts.go）——在册 modernc 空集坑高危形状且是 FTS 失败的兜底（兜底自己失明），抽 likeSearch 助手按模式分轮单列查询+Go 合并去重+凑满 limit 提前收工+rows.Err。
- **刀B 启动链**：④Startup 幂等三处（app.go/whisper_state.go/gaea_tasks.go）——logClose 先关旧再换新、initWeixin 先按 Shutdown 同款清扫 weixinServers 再重建、startFileWatch 先 Close 旧 watcher；⑤ai.Client 双构造+桥接早启窗口（app.go）——Startup 复用 New() 实例（nil 才新建），同一 client 贯穿进程生命周期，configureClient 统一接线，Login 重建路径不变；⑥filewatch 同步 Walk——**addTree 异步化试错证伪回退**（测试 5s 超时=监听就绪前事件真丢，核实「全量索引兜底」不成立：正常启动成功路径不做全量索引），改为 app 层 `go a.startFileWatch()` 整体移出 Startup 临界路径，filewatch 保持「Start 返回=监听就绪」同步契约（测试零改动零丢事件）。
- **刀C 校验**：⑦GaeaListDir 相对路径拒 .. 穿越（gaea_listdir.go）——前端四调用方逐一核对全工作区相对路径零误伤；IsAbs 分支保留（v4.98 冻结面拍板+resolvePreviewPath 口径对齐）。
- **留池 4 项**：ImportProjectCharacters 事务化（需三助手 tx 化中型重构）/prompt 引擎双构造（毫秒级+agent 持 eng 指针时序风险）/SaveVersion 并发窗口（待复现）/GaeaConvertToPdf 绝对路径（需统一设计）——todos 已给原因。
- **门禁**：go build/vet 0+受影响 6 包测试绿（app 96s 全量/cost/whisper db+repos/filewatch）+全量 ci.ps1 绿+bindings drift OK@706+版本三处 4.359.0。
 - **坑**：⓪vitest 连续两轮 CI 失败病根=.tmp 堆积 2.4GB（go test 残留+旧浏览器 profile），清到 157MB 后稳过；「.tmp 定期清理进卫生守卫」挂池①审计论断「事件丢失有全量索引兜底」要核实兜底是否真存在——本例正常启动成功路径不做全量索引，照抄审计建议就是真回归；filewatch 测试 5s 超时是免费的反证②挂池再取证≠照单全收——留池 4 项与落地 7 刀同等重要，留池注明原因让下次取证不必重查。
- **产物**：exe 50931200B SHA256=acd0851d4cbdcbfcac9801ffb59411418ad2530a32a0a8f8f8ba90e18c5fa89f（时间戳 2026-09-20 20:21:39 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.355~v4.359 删 v4.354.exe。

## 最新发布：v4.358.0（2026-09-20）「后端优化轮第三弹：三路审计（SQL 存储层/启动链路/输入校验）——分类改名 P0 根修+读侧收口+FTS 增量」

- **动机**：用户口径「继续优化后端」——v4.356~v4.357 已扫 panic 防线/并发锁/资源句柄/错误吞噬四条轴，本轮换新镜头开三路并行只读审计（线1 SQL/存储层、线2 启动链路、线3 输入校验），按证据定刀。纯 Go 36 源文件+3 新测试文件，绑定面 706 不动，前端零改动。
- **线1 SQL**：①P0 分类改名 substr 字节/字符错位（cost.go SaveCategory，Go len() 喂 SQLite 按字符计数的 substr——中文路径子树条目路径整段截断重挂）→rune 计数偏移+精确行 category 叶子名同步+两写包事务+换父也重写+环检测+名称含 / 拒绝（同函数五问题一次收口）②rows.Err 缺失 17 处批量补（记忆/三元组/情节恢复主路径+成本检索语料+ListItems 直通不可变版本快照）③单条写全量重建 FTS O(N²)→接上闲置增量助手（失败降级全量重建兜底）+两个 Rebuild 事务化（原 DELETE+逐条重插 autocommit=索引空窗）④schema_v15 补 facts/episodes 会话索引+向量写事务化+COUNT 收错×5+GetSource 点查。
- **线2 启动链**：⑤ASR provider 四引擎刷新 goroutine 并发写竞争→SetASRProvider 持锁+读侧快照（对齐 SetRealtimeSession 先例）⑥恢复/迁移结论先于 setupLogging GUI 全盲→restoreSummary 字段日志就绪后回放⑦三 db 打开失败 log.Printf→slog.Error⑧Shutdown 补关 Hephaestus/whisper db+微信 Stop 通知异步化（对齐 notifyStart 先例）+回退轮询 stop 接线+reminderStop Once（复用已有 reminderOnce 字段）。
- **线3 读侧收口**（审计结论：写删侧护栏完备、读侧系统性短板）：⑨GaeaReadFile 对齐写端口径（拒穿越+withinWriteRoots+2MB 截断）⑩AttachmentDataURL withinReadRoots 根白名单+32MB⑪ReadFileB64 契约强制化=Pick 登记机制（fail-closed，pickFile.ts 链路零破坏）⑫Preview 相对路径拒穿越+image/docx 32MB 封顶⑬快照/场景 sceneID 白名单（唯一写越界点）⑭上下文看板 sessionPath 补 sessionDirForPath⑮CreateProject 书架 containment。
- **辨伪挂池**：FTS 降级长 OR 链疑似空集坑/版本号并发窗口/legacy WAL 拷贝/deferred BUSY/SelfHeal N+1/导入无事务/Startup 幂等/filewatch Walk/prompt·aiClient 双构造/ListDir 全盘枚举（需查前端调用源）/ConvertToPdf 绝对路径——todos 新挂池行已给 file:line。
- **测试**：定向 +3（TestSaveCategoryRenameChineseSubtree+Guards/TestReadFileB64RequiresPick/TestGaeaReadFileTraversalGuard/TestSnapshotSceneIDGuard）+既有测试修正 2 处（contextview 伪造路径改真实会话目录形态）+V15 迁移断言。门禁=build/vet 0+受影响 10 包绿+全量 ci 绿+drift OK@706+版本三处 4.358.0。
- **坑**：①Go/SQLite 长度语义差是中文环境必炸点——len() 喂按字符计数函数 ASCII 全绿中文必坏，跨语言偏移一律 rune 或 Go 侧完成②增量助手存在却没人用——接闲置能力补失败降级兜底③同函数多问题一次想全④读侧防线不能照抄写侧——根白名单+Pick 登记+尺寸封顶组合，逐绑定核前端调用链（AttachmentDataURL 只限 portraits 会误伤绘梦资产）。
- **产物**：exe 50928640B SHA256=2e1bfdf827f3aa70cc2f8f84a21d3259d2776c6bec337e5f33b553c21e2641bf（时间戳 2026-09-20 07:27:12 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.354~v4.358 删 v4.353.exe。

## 最新发布：v4.357.0（2026-09-19）「后端优化轮第二弹：挂池三组清账——错误吞噬可见化 4 处 / 并发观察 4 刀根修 / 死代码 2 文件」

- **动机**：用户口径「继续优化后端」——v4.356 挂池三组（错误吞噬卫生 P2×5/并发观察 6 项/AuditLogger·DecisionLogger 死代码）逐项取证定刀。纯 Go 8 源文件+3 测试文件（含 1 处签名适配），绑定面 706 不动（零签名变更 drift OK），前端零改动。
- **刀A 失败可见化 4 处**：①characterlib ListChatEnabled 吞错→签名 ([]Character, error) 上抛（WhisperGetPersonalities 失败 warn 回退内置人格、app.go 剧照同步 warn 跳过——「没有可聊天角色」与「库读失败」不再混淆）②DAG 懒清扫 `_ = Save` ×2→warn（sweep 幂等下次重扫重存，失败可见即可）③turn markdown 投影 `_, _ =`→warn（JSONL 证据链仍在，投影静默缺失不可见）④cost_projects 状态标签 Save ×2→warn（版本/沉淀本体不受影响，列表页状态不再永远旧档）。辨伪：AppendTurnTraceToDB 失败仅 log 不升级——whisper 链无 Notice 事件总线（WhisperChat 同步绑定返回），接线成本>收益，日志 Error 级已可见。
- **刀B 并发加固 4 刀**：⑤PrepareContinue TOCTOU 根修（subagent_store.go）——meta.Status 检查与 MarkRunning 写回之间窗口，双路续跑同 ref 交错写转录；绑定层 followUpClaims 只盖 UI 路径，TaskTool continue_from 无守卫。修法=**接通 store 半成品**（SubagentRun.release 字段+Release() 调用+注释三件齐备却无人赋值恒 no-op）：continueClaims sync.Map 占 ref 单飞，第二路报 already being continued；claim 随 defer run.Release()（调用点本就有）或 SaveCompleted/SaveFailed 终态写兜底释放；转录 Load 失败不占坑；零调用方改动。⑥cost BM25 语料/版本戳原子化——rankerFor 内部再 Load 与调用方捞语料非同一时点，写路径在两步间推进版本时旧语料挂新版本 key 用到下次写；调用方捞语料前快照 corpusVer 传入（rankerFor 加 version 参数不再自取）。⑦weixin Stop→Start 双轮询根修（clawbot.go）——旧 pollLoop 睡在轮询间隔/长轮询 HTTP 里，Stop 后立刻 Start 时 running 被 Swap(true) 复位，旧 loop 醒来条件复活与新 loop 并行；修法=生命周期代际 gen atomic.Int64，每次 Start 递增，pollLoop 捕获启动代 alive()=running&&gen==启动代；panic 兜底的 running 复位加代际守卫（防旧 loop panic 打掉新 loop）。⑧realtime Dial 双拨号泄首连（openai.go）——检查（conn==nil）与赋值之间窗口，输家覆盖 s.conn=赢家连接被遗忘无人 Close（fd 泄漏+双 readLoop）；修法=CAS 落位锁内重检 closed/conn，输家关掉自己拨到的连接再报 already connected/session closed。
- **刀C 死代码**：删 internal/gaea/control/audit.go+decisions.go（~230 行）——New*Logger/SummarizeAuditLog 全仓零调用（v3.2/v3.3 期设计未接线），控制包测试零引用；实际审计走 core/journal.go AppendAudit 与 whisper/desktop_audit_log.go 两条活链。挂池③销账（「直接删」）。
- **挂池辨伪不修 2 项**：memory file_backend 双写非原子（事实文件+MEMORY.md 索引两步）——设计内自愈语义已在（损坏行跳过/traceable and recoverable/索引可重建），事务化两阶段写成本>收益；QuickAdd 锁内 AppendDoc——锁刻意串行化 ReadFile+WriteFile 读改写（移出锁=并发丢更新），低频小文件阻塞窗口毫秒级。
- **测试**：定向 +2——TestPrepareContinueSingleFlight（双路同 ref 第二路受理处即拒/Release 后可再续/终态写兜底释放后可再续）+TestOpenAISession_DialConcurrentLoserCloses（并发双 Dial 恰一胜一负，输家报 already connected，赢家连接保留）；既有测试适配 1 处（store_test 双返回值）。race 检测本机无 gcc 未跑（CI race job 兜底）。
- **门禁**：go build/vet 0+受影响 6 包测试绿（agent/realtime/cost/weixin/characterlib/app）+全量 ci.ps1 绿（exit 0）+bindings drift OK@706+版本三处 4.357.0。
- **坑**：①半成品机制比没有更危险——release 三件齐备无人赋值，读代码误以为守卫存在；新并发守卫落地先 grep 赋值点验证接线完整性②可见化≠fail-closed——幂等写失败 warn 即可不阻断主流程，判据=失败是否改变主操作结果③代际修复要同时护 panic 兜底复位路径④吞错修复优先改签名上抛而非原地打日志（编译器强制全调用点审查）。
- **产物**：exe 50900480B SHA256=c3877db06fd398b74df4a6b981b0f599ba104e7f81062ecd886b0ff51872a413（时间戳 2026-09-19 23:03:48 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.353~v4.357 删 v4.352.exe（SUMS 身份档案全保留）。

## 最新发布：v4.356.0（2026-09-19）「后端优化轮：panic 防线收口 12 处 / 确定性自死锁根修 / cfg.Model 并发收口 / 快照失败可见化」

- **动机**：用户口径「继续优化迭代 gaea，本次会话优化后端」——2026-09-12 后端性能普查（两遍扫描，v4.245~v4.253 刀A~E）已全清，换镜头开**三路并行只读审计**（线1 panic 防线与错误吞噬/线2 并发安全与回调发射序/线3 资源句柄生命周期），按证据定刀。纯 Go 30 源文件+2 测试扩展，绑定面 706 不动（零签名变更 drift OK），前端零改动。
- **审计总貌**：线1=Wails dispatcher 只保同步绑定调用，goroutine 裸奔 panic=exe 闪退；65 处 go func+约 30 处 go xxx() 交叉比对 58 处 recover，撞出 12 处漏网。线2=整体纪律水准之上（jobs 锁序/Registry leaf-lock/approval 双查/tasks 条件 UPDATE 堪称范本），真问题=绑定层「调用方自觉」缺口（自死锁/裸字段/无运行闸），非内核锁错。线3=**零 P0/P1**（body 52 文件/os 79 处/rows 93 处/ticker 22 处/子进程收尸/fsnotify 全配对）——句柄轴线纪律极好，余卫生项转刀B。
- **刀A1 防线 12 处**：微信 clawbot pollLoop+handle（唯一常驻外部通道内同步跑完整 AI 回合；同文件三处先例证明漏网；降级文案 fallbackReplyText 常量化）/DAG 三处（节点 panic 转节点 failed 与 err 同形+dagStart 编排兜底+改向——与 GaeaSubagentFollowUp 逐行同构却没抄防线）/语音回合链三处（handleSpeechEnd/runReply/runRealtimePump，全文件 0 recover）/MCP StartAvailable 连接（外部协议数据 panic 转 RecordFailure）/书源五处（导入/追加/下载 panic 转 error 事件不挂进度；fetch worker 与搜索单源转槽位错误项宁漏勿误）/dream/tts/SSE parseStreamEvents（自身 defer close(chunks) panic 路径照常执行，消费方不悬挂）。
- **刀A2/A3 并发守卫**：①GaeaMemorySetRetentionDays 持 ga.mu 调 GaeaInit（内部首行再取锁）——sync.Mutex 不可重入，**引擎未初始化时调用即确定性死锁**：绑定 goroutine 永久卡死持锁，GaeaSend/Cancel/Skills 全部消费方连锁冻结仅重启可解→锁外探测 needInit 仅未初始化才前置 Init（gaeaApplyCfg 同序；已初始化路径行为不变，现有测试零改动）②cfg.Model「UI 三写点×请求热路径四读点」裸字段竞态（string ptr+len 双字撕裂）→internal/config/config_model.go 新增 modelMu+SetModelMem/GetModelMem（Config 是值语义大结构加锁字段毁复制性，访问器是最小侵入解；启动期 Load 单线程装配不受限）③GaeaNewSession/GaeaResumeSession 补 Running() 运行闸——run loop「only swaps session while idle」是注释约定无执行强制，回合中换会话=SetSession 与无锁读交错+后续消息落旧 session 转录分裂；照 GaeaSubagentFollowUp 先例绑定自守防 httpbridge 绕过前端闸。
- **刀B 失败可见性+卫生**：①回合收尾快照失败 slog.Warn-only→Warn 级 Notice（前端 Transcript 按 level==="warn" 渲染可关闭错误卡）——会话文件被 OneDrive/杀毒/备份短暂锁定或磁盘满时 UI 全绿重启回退数轮无提示；对照模型调用前 checkpoint fail-closed 纪律，收尾快照漏了同款 ②chat 流式 runID 毫秒时间戳拼 atomic 序号（同毫秒双路碰撞=两 goroutine 向同一事件通道交错发帧）③netclient 新增 DrainAndClose（非 2xx 早退不读尽 body=连接被丢弃不复用，健康探测每次重新 TCP/TLS 握手；64KB 上限；统一替换 herdsman health/modelengine 三文件/plugin transport_http 9 处 defer Close，成功路径 no-op+Close 安全）④裸 rename 迁 fileutil.RenameWithRetry 21 处/18 文件（app 13+config_save/backup/dag×2/narrative/office docxedit·pptxedit/project；Windows AV/索引器瞬时占用保存失败韧性；跳过跨盘回退触发/跨目录搬迁/恢复流程等特殊语义 12 处）。
- **测试**：定向 +2——TestGaeaMemorySetRetentionDaysUninitialized（死锁回归：未初始化路径带 30s 超时守卫+os.Exit 防 defer 二次挂起，照 gaea_init_deadlock_test 先例；断言锁外 Init 拉起成功+保留期生效）+TestDagNodePanicMarkedFailed（runner panic→节点 failed 带 panic 证据+下游 skipped）。
- **门禁**：go build/vet 0+受影响 10 包测试绿+全量 ci.ps1 绿（go test 全量+lint+build+vitest 386 文件 3295 例——ContextView 2 例在册 flaky 复跑 28/28 绿）+check-docs OK（AGENTS 38231B 预算内）+bindings drift OK@706+版本三处 4.356.0。
- **坑**：①goroutine 裸奔是绑定层系统性缺口而非个例——防线判据=「同文件/同型先例是否已立」（clawbot 三先例/followup 同构/plugin 双标），漏网处全是新码没抄先例②sync.Mutex 不可重入死锁是确定性而非竞态（cfg==nil 分支必现）——「锁外探测+幂等重入」修法保已初始化路径零变化③cfg.Model 收口走包级访问器不改结构（加 sync.Mutex 字段会触发 vet copying locks）④vitest flaky 先复跑失败文件再定与本轮无关（足迹零交叠）⑤后台复合命令 `taskkill…; cmd /c build.bat` 在 harness 下静默假成功（exit 0 只出 banner）——**构建产物必须核对时间戳/哈希/大小三件**，本次靠 v4.355 旧哈希识破重跑。
- **产物**：exe 50893824B SHA256=9C570C94C07A1500282DAABDAE7FCD7BDC2E26ACA1BD305BDBF66EDC25709205（releases/gaea-v4.356.0.exe+SHA256SUMS-v4.356.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.352~v4.356 删 v4.351.exe（顺带补清上版漏删实物的 v4.350.exe；SUMS 身份档案全保留）。
- **挂池新增（观察池）**：错误吞噬卫生 P2（characterlib ListChatEnabled 吞错返回空列表/cost_projects 双 Save 仅状态标签/dag sweep `_ = Save`/AppendTurnTraceToDB 失败仅 log/turn markdown 导出吞错）；并发观察 6 项（PrepareContinue TOCTOU 双路续跑同 ref——claims 只盖 UI 路径/cost BM25 语料与版本戳非原子/weixin Server Stop→Start 旧 pollLoop 未退可双轮询——当前绑定面不可达/realtime Dial 双拨号泄首连——单会话不可达/memory file_backend 双写非原子 legacy/QuickAdd 锁内 AppendDoc 文件 IO）；AuditLogger/DecisionLogger 死代码（Close 无人调，接线时须挂生命周期或直接删）；15 处「SetLoader 形式竞态」类已注释自证的良性单字指针读维持观察。

## 最新发布：v4.355.0（2026-09-19）「真机走查班：UnifiedSearch 参数序错位根修」

- **动机**：用户口径「继续」——前端优化六连发后按惯例开真机走查班（v4.343 非版本刀先例），CDP 9333 附着 v4.354.0 壳（WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333 直启 exe）。
- **巡检基线**：双空间 13 页（书斋 6+乐园 7）错误边界 0 / 空白 0 / exception 0（.tmp/walk-v4354*.mjs 三版迭代）。
- **真发现：UnifiedSearch 三调用方参数序错位（月级老 bug）**：Go GaeaUnifiedSearch(query, topN int, scope ...string)（v2.23.0 起）vs 前端 bridge 声明与调用方 (query, scope, topN)——scope 落 int 形参每次 unmarshal 失败；记忆中枢三脑检索/工作区搜索面板/Ctrl+K 真壳自 v2.23 一直坏。捂因=mock 按前端期望签名写（契约全绿）+bindings_completeness_test 只收方法名+.catch(()=>null) 静默——v4.350 失败可见化（「检索失败，请重试」）一次走查拽出。修复=bridge 签名 (query, topN?, scope?)+MemoryHubPage/WorkspaceSearchPanel/SearchModal 换序+mock 同序+mock-contract-e5/MemoryHubPage/WorkspaceSearchPanel 三测试断言换序；零 Go 改动绑定面 706 不动。
- **修复确证（新壳复验）**：错误从 cannot unmarshal → embedding 请求失败（localhost:8080 语义服务超时——本机 embedding 未部署属环境事实挂观察池）；UI 三态如实呈现。语义检索完全可用需先起 embedding 服务，届时补一条龙走查。
- **其余发现**：角色库页 console 27 条 file:// 头像 WebView2 拒载（既有非本轮回归，挂池：AttachmentDataURL 或 asset:// 通道）；[longtask] 69~165ms 误报阈值过敏感挂池；ResizableDrawer Esc 真机抽验因导航成本收尾（vitest 4 例已锁）。
- **坑**：①mock 契约必须从 Go 绑定复制签名含参数序——按「前端怎么调」写，签名错位时 mock 全绿真壳全坏②静默吞错捂 bug 时长与可见性成反比，失败可见化是最便宜的测试③走查探针按钮定位优先类名/testid，textContent 匹配撞到别的按钮得假阴性（「检索中…」也是按钮文案）。
- **门禁**：tsc 0（--force）+eslint 0/0+全量 ci.ps1 绿（vitest 386 文件 3295 例）+bindings drift OK@706+版本三处 4.355.0；产物见 releases/v4.355.0.md。**文档**=releases/v4.355.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（七十迁：v4.352 入 archive）+progress/todos。**未做（下刀池）**=embedding 服务就绪后语义检索一条龙走查/file:// 头像通道治理/longtask 阈值/dirListingsCache 接线/海报墙 hero 内容槽（内容决策）。

## 最新发布：v4.354.0（2026-09-19）「前端优化第六轮：P0 数据形状崩溃×2 / 异步竞态 11 处 / 资源生命周期」

- **动机**：用户口径「继续」——五连发后新现场再开三路并行只读审计（异步竞态与并发 / 运行时数据形状防御 / 资源生命周期第二层），按证据定刀；纯前端 17 源码 + 1 测试扩展，零 Go 改动、零绑定变更（706 不动）。
- **P0 数据形状崩溃×2**（均为「后端产出前端未声明形状 + mock 不可复现」同型）：①伏笔面板 STATUS_META 3 态 vs Go 6 态并集透传，「清理→项目重置」把手动条目置 pending 重拉即整页崩（一键自毁式交互）→类型 6 态并集+STATUS_META 补全+statusMetaOf 防御未知值+流转按钮 6 态文案语义，测试+2；②记忆面板 suggestions.skills=null（Go suggestSkillsFromMemories 规则类记忆<2 条 return nil，nil slice 无 omitempty 序列化 JSON null；mock 恒 []）→setSuggestions 入口统一收窄归一（skills/memories/merges ??= []），消费侧五处不再逐个防御。
- **异步竞态 11 处**（统一项目已有 seq/loadToken 范式补齐，辨伪 18+）：PromptWorkshopPanel 模板切换 seq（A 正文写进 B 表单保存即数据损坏）；BookSearchModal pickSeq+backToSearch 失效（B 书源 URL+A 章节范围错误导入）；useChatTopics.createTopic/ChatPage.handleClearMessages 失效在途（新增 invalidateLoads 暴露，空话题视图被旧话题填满后清错对象）；useSinStory loadSeqRef+createStory 失效（切故事串台）；ChapterPage.handleSave 保存快照+setTabs 函数式比对（保存期间继续打字不再被强制 saved=true，未保存保护不丢，提示「请再次保存」）；SearchModal 移除 onPressEnter 双发+searchSeqRef（旧意图卡不再可被执行）；MemoryHubPage runSearch searchSeqRef（Enter 路径守卫）；useImageGenConfig backendSwitching 重入闸；outlineStore 模块级 loadSeq（错书章节读写风险）；WeixinPage togglingId busy 闸+Switch loading（连点两条相反 Save）；TTSPlayer 卸载 cleanupAudio（泄 blob URL 且音频继续外放）。
- **资源生命周期第二层**：blob URL/Observer/AudioContext/MediaStream 全量辨伪通过（saveFile 同步 revoke、useVoiceChat 四路回收、9 处 Observer 均 disconnect）；dirListingsCache 无界+invalidateTurnCaches 零调用方挂池（结构性欠账需设计接线时机）。
- **坑**：①Go nil slice→JSON null 且 mock 手写 [] 永不复现——前端收窄层统一归一优于消费点逐个防御②saved 标志语义=「缓冲区==磁盘」，异步完成侧不能无条件置位，快照比对是通用解③antd Input.Search 回车已触发 onSearch，再挂 onPressEnter=双发，删除而非守卫④竞态修复统一走已有 seq 范式不发明新机制。
- **门禁**：tsc 0（--force）+eslint 0/0+全量 ci.ps1 绿（vitest 386 文件 3295 例）+bindings drift OK@706+版本三处 4.354.0；产物见 releases/v4.354.0.md。**文档**=releases/v4.354.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十九迁：v4.351 入 archive）+progress/todos。**未做（下刀池）**=dirListingsCache 失效接线/CharacterPage 双击重复写（幂等低危）/海报墙 hero 内容槽（内容决策）。

## 最新发布：v4.353.0（2026-09-19）「前端优化第五轮：ResizableDrawer 弹层可访问性收口 / 可点 div 键盘化余量」

- **动机**：用户口径「继续」——可访问性余池两条线收口；纯前端 9 源码 + 1 测试（新建），零 Go 改动、零绑定变更（706 不动）。
- **线1 ResizableDrawer 三缺补齐**（对照 ApprovalModal 先例）：dialog 语义（role=dialog+aria-modal+aria-label 新 prop，两调用方传 i18n 标题）+ Esc 关闭（可编辑目标不劫持）+ 焦点管理（开记录 activeElement+聚焦容器 tabIndex=-1，卸载还原 try 兜底）；ResizableDrawer.test 新建 4 例（Esc 经退出动画/普通键不关/dialog 三件套/开聚焦+卸载还原）。
- **线2 可点 div 键盘化余量**（v4.349 池逐点核对净剩 4 处+顺带 5 钮）：ChapterTreePanel 章节树节点/OutlinePanel 大纲行/StatsPanel 会话·本轮两折叠→role=button+tabIndex+Enter/Space+aria-label（FileTree 范式）；ChapterEditor 加/删场景+ChapterTreePanel 重新生成/删除/添加五 icon Button 补 aria-label（title 不参与可访问名）；FiveCalcPanel 版本带入遮罩补 Esc。辨伪=RailItem/FactCard 已是 button。
- **坑**：①effect 引用 useCallback 声明顺序——Esc effect 插在 handleClose 声明前，依赖数组渲染期即时求值 TDZ ReferenceError；effect 一律放依赖声明后②React 合成 KeyboardEvent 无 isContentEditable（DOM HTMLElement 属性），div onKeyDown 判可编辑目标要 e.target 断言③审计清单是快照不是现状，逐点核对再动手。
- **门禁**：tsc 0（--force）+eslint 0/0+全量 ci.ps1 绿（vitest 386 文件 3293 例）+bindings drift OK@706+版本三处 4.353.0；产物见 releases/v4.353.0.md。**文档**=releases/v4.353.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十八迁：v4.350 入 archive）+progress/todos。**未做（下刀池）**=reducer 级整店订阅 13 处（v4.350 审计辨伪「appStore 写入面低频不构成热点」维持不做）/selbar z-1080 阶梯治理（防御性）/海报墙 hero 内容槽（内容决策）/持久全量 overlay（真机窗口）。

## 最新发布：v4.352.0（2026-09-19）「前端优化第四轮：ForeshadowPanel 行 memo / ModelCenter ctx 两通道拆分 / 图标钮可访问名」

- **动机**：用户口径「继续」——v4.351 下刀池延续；纯前端 16 源码 + 5 测试（扩 5），零 Go 改动、零绑定变更（706 不动）。
- **线1 ForeshadowPanel 行 memo**：抽 ForeshadowRow memo 组件 + flow/remove/saveEdit/handleEditChange/cancelEdit/startEdit 六回调 useCallback 化 + 编辑态布尔/文本双 prop 下发——行内编辑/登记表单每键只重渲染受控行（原全列表重渲染，数百行可达每行 Tag/Tooltip/Popconfirm）；测试 +2（编辑保存全量写回含 3 条/取消不写回其余行不受影响）。
- **线2 ModelCenter ctx 两通道拆分**（v4.350 辨伪项的正解）：ModelCenterContext 拆 StateContext（状态字段，内容变化才触发消费者重渲染）/ ActionsContext（setter/handler 引用波动单独承载）/ 兼容口（两通道合并，useModelCenter 存量不破坏）；页面 stateCtx/actionsCtx 各自 useMemo，9 个 section 迁 useModelCenterState + 按需 useModelCenterActions（多行解构逐字段分类），settingGlmEndpoint 归 state——键入 API Key 不再全 section 重渲染；4 个测试文件兼容 Provider 补三连挂载（value 同对象）。
- **线3 图标钮可访问名**：ChatRow 四处（用户/助手复制动态态+朗读）+ ChatModeBar 四处（联网开关动态态/角色库管理/切换角色/语音设置）补 aria-label——Tooltip 不参与可访问名；SecurityPanel 排除（按钮带文字标签非图标钮）。
- **坑**：①多行解构迁移必须先从源码抄全字段清单再分类（凭接口记忆分类漏 trendData/settingGlmEndpoint/setXxx——UNCLASSIFIED 断言救一半，tsc 逮另一半）②测试文件改完必须 tsc -b --force（上轮沉淀本轮再验证）③兼容口三 Provider 嵌套顺序 state→actions→兼容，测试挂载三连 value 同对象语义不变。
- **门禁**：tsc 0（--force 全量复核）+eslint 0/0+全量 ci.ps1 绿（vitest 385 文件 3289 例）+bindings drift OK@706+版本三处 4.352.0；产物见 releases/v4.352.0.md。**文档**=releases/v4.352.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十七迁：v4.349 入 archive）+progress/todos。**未做（下刀池）**=其余可点 div 键盘化 ~8 处（v4.349 池）/ResizableDrawer 焦点管理/reducer 级整店订阅 13 处/selbar z-1080 阶梯治理/海报墙 hero 内容槽（内容决策）。

## 最新发布：v4.351.0（2026-09-19）「前端优化第三轮：启动反闪根修 / 绘梦历史有界化 / 亮主题叠加层失效群 / color-scheme / 反馈确认补遗」

- **动机**：用户口径「继续」——v4.350 下刀池延续，六线并收；纯前端 29 源码 + 4 测试（1 新 3 扩），零 Go 改动、零绑定变更（706 不动）。
- **线1 启动反闪根修**：亮色用户每次启动先看 ~2s 深空屏（index.html 静态启动屏硬编码深底，主题变量 App useEffect 首帧后才注入）→ head 内联脚本（bundle 前）预读 gaea-display-mode，判定顺序**逐字对齐 appStore.loadMode**（light→亮 / system→matchMedia / 旧布尔键 gaea-dark・wubigork-dark '0'→亮 / 缺失→dark 缺省），亮则 html[data-bs-light]；静态屏与 body 浅色分支 + boot-splash.css 首帧兜底——App 注入真令牌后 var() 接管，两段同色无跳变。
- **线2 绘梦历史有界化**（无界三处收口，素材库 120+12/页先例对齐补漏）：持久化 HISTORY_META_MAX=500 读/写同截（原 path-only 条目可积数千条）；渲染窗口 60+「加载更多(N)」增量展开（原数百 img/video DOM 一次性构建）；回填 restoreHistoryImages 加 limit 参数按 needsFileRestore 过滤取前 48（内联小图不占名额；原全量 dataURL 常驻=数百次生成数百 MB），窗口外缩略图落既有占位、选中/下载/复用 resolveResultImage 按需读文件——记录仍在，零功能删除。
- **线3 亮主题中性叠加层失效群 30 处**：rgba(255,255,255,0.02~0.08)（暗底白提亮惯用法）在浅底白叠白不可见（hover 消失/卡片只剩边框/进度轨道消失/shimmer 不动）→统一 color-mix(var(--color-text) N%, transparent) 明暗自动正确；辨伪保留 ChatCodeBlock 2 处（hex-exempt 暗色代码面板 chrome 刻意不随主题）；MarkdownContent pre 底 rgba(0,0,0,0.3)→surface-container 令牌（亮主题中灰脏块）。
- **线4 color-scheme 声明**：App.tsx 注入 root.style.colorScheme（原全仓无声明，暗主题原生滚动条轨道角/表单控件残留亮色渲染）。
- **线5 反馈确认补遗**（上轮 B 系列遗留 4 处）：角色记忆弹窗三路读取全吞错→失败 Alert（原打开即空白与「无记忆」不可分）；青鸟助手两路全失败原伪装「暂无助手」假空态→载入失败条 role=status（单路失败仍兜底不误报，轮询自动重试）；图片下载两处失败 message.error+成功提示（另存为抛错原落 unhandled rejection，取消不算错）；自定义生图模板删除补 Popconfirm。
- **测试**：HistoryRail.test.tsx 新建 3 例（超窗渲染 60/点击增量展开到底按钮消失/不足窗口全量不回归）+historyMeta +1（limit 按需恢复项计数）+meta +2（读截/写截）+WeixinPage +1（双失败提示+单路兜底不误报）。
- **坑**：①静态期主题预读判定序必须逐字对齐 store.loadMode，多一个 matchMedia 兜底就分叉首帧跳变②limit 语义按需恢复项计数非位置（slice 让内联小图占名额）③heredoc 写测试三层转义（python→文件→JS），\ 才是源码单反斜杠，写完看原始字节（eslint no-useless-escape 逮住）④rgba(255,255,255,x) 不全是债，先排除 hex-exempt 名单再批量替换⑤GenResult 无 id 字段——tsc 增量缓存漏检测试文件改动，强制 tsc -b --force 复核。
- **门禁**：tsc 0（强制全量复核）+eslint 0/0+全量 ci.ps1 绿+bindings drift OK@706+版本三处 4.351.0；产物见 releases/v4.351.0.md。**文档**=releases/v4.351.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十六迁：v4.348 入 archive）+progress/todos。**未做（下刀池）**=ForeshadowPanel 行 memo/ModelCenter ctx 拆分/其余可点 div 键盘化（v4.349 池）/ResizableDrawer 焦点管理/reducer 级整店订阅 13 处/selbar z-1080 阶梯治理。

## 最新发布：v4.350.0（2026-09-19）「前端优化第二轮：事件总线连环炸根修 / P0 幽灵令牌 / 破坏操作确认 / 静默失败 / 流式渲染热点 / 主题破绽」

- **动机**：用户口径「优化 gaea 前端」（v4.349 五线后第二轮）。同范式：四路并行**只读审计**（事件与定时器泄漏 / 交互反馈三态 / 渲染热点 / 视觉一致性，各带 file:line 与量级推理）再按证据定刀；纯前端 32 源码 + 6 测试（3 新 3 适配），零 Go 逻辑改动、零绑定变更（706 不动）。
- **线1 事件总线连环炸根修**：❗`onTaskEvent` 清理走 `EventsOff(channel)` 全清（wails 语义=注销通道全部监听者；v4.61 事故红线 v4.62.2 只修了 onEvent/onSubagentText，本条漏网）——gaea-task 有 5 个并发订阅点（运行角标/任务自动激活/任务面板/价格源/索引任务），任一卸载（关一次任务面板即触发）=角标冻结+自动打开失灵+下载卡「进行中」，keepAlive 下直至应用重启→一律 `subscribeWailsEvent` 按监听者退订；onUpdaterProgress/onReady 同纪律收口。polyfill（网页模式）双修：EventsOn 返回 void→消费方 deps 每变往 eventBus **叠加** handler（ModelCenter 四通道单事件 N 次重复后端拉取）→改返回「只摘自己」退订对齐 wails v2.13 桌面语义；EventsOff(带 callback) 无条件 sse.close()→共享通道他人推送被掐断→只摘自己、通道空才关 SSE。
- **线2 P0 幽灵令牌**（12 主题功能性坏点）：`--md-sys-color-error` 全仓零定义而 gaea 四组件 26 处无 fallback 消费（CSS invalid：失败态红点/红字/红框全消失）→注入 colorDestructive 语义别名；`--md-sys-color-surface-container-low` 零定义→追问发送按钮箭头 12 主题恒隐形+输入区透明+VersionTimeline 对比底丢失→color-mix 插值；`--color-info`/`--md-sys-color-info` 补注入（#38bdf8/#0369a1 明暗分档）；幽灵四连 `--v3-fg-soft`(17 处)/`--v3-line-soft`/`--color-text-tertiary`/`--md-sys-color-text-tertiary` 明暗分档定义（暗主题 12px 次要文字 3.9:1<AA）；modelcenter.css 13 处 gaea 作用域 token 换主令牌直连（--fg 系只在 gaea/styles.css 定义且该页不加载→亮主题 #ddd 白底 1.3~2.5:1）。
- **线3 破坏操作二次确认 9 处**：高危批量/清空 4=知识库批量删除（双分支）/成本库批量删除（**连带假成功根修**：吞错后无条件「已删除 N 条」→诚实成功 N 失败 M）/绘梦清空历史（prompt/seed 即时持久化）/原罪清空故事消息（ToolbarButton 无 forwardRef→Modal.confirm 命令式）；单条 5=任务收件箱删除（**连带吞错可见化**+i18n 三语新键）/青鸟提醒/组织/角色关系/划线想法列表（span 挡冒泡防触发跳转）。
- **线4 静默失败反馈**：三脑检索失败与 0 命中三态化（命中/换关键词/失败重试，role=status）；排程导出 XML 补成败提示（SaveFileAs 抛错原落 unhandled rejection）；章节载入失败白板→message.error 带原因（原只 console.error 且 saved:true 误导正文丢失）。
- **线5 渲染性能**：Transcript 提及扫描按文本内容缓存（原每 chunk 对全会话 assistant 正文跑双全局正则+O(m²) 重叠；完成后文本不可变→缓存命中只付流式段成本，上限 512）；ChatPage topicList memo+ChatTopicSidebar React.memo+过滤 useMemo+行 hover 删 hoveredId 改纯 CSS；ToolCard summary/prettyArgs/outputLines 三处 memo（write_file args 内联整文件数百 KB）；`useNow(active)` 条件订阅（已完成消息/过程卡/RunStatus 无条件订阅 1s 时钟=O(N)/秒常驻重渲染清零；**伴生修复**「思考 Xs」完成后持续增长→完成边沿 Date.now() 定格）。
- **线6 视觉**：RelationGraph 画布 #ddd/#555/#333/#444 写死（6 亮主题节点名不可见）→resolveCSSColor 主题令牌解析+darkMode 入 deps；mermaid 跟应用 data-hl 而非 OS prefers-color-scheme。
- **辨伪**：ModelCenter ctx（100+ 字段）不 memo——五 hook 返回对象字面量每渲染新引用 deps 无法建立，强行 memo=陈旧 ctx 真 bug（挂观察池）；TTSPlayer/AIConsole 单实例全清无害；z-index 1080 阶梯脆弱但无叠加路径；boot 亮色反闪挂池。
- **测试**：events.test 新建 4 例（双订阅退订隔离+EventsOff 永不调用/space 过滤不变/updater+ready 同纪律）+runtimePolyfill +3 例+useNow 3 例；TaskInboxPanel/WeixinPage/KnowledgePanel 删除用例改两步确认。
- **坑**：①EventsOff 红线是「禁全清」非「禁用」——polyfill 带 callback 只摘自己+通道空才关 SSE 才是完整语义②Popconfirm clone 注入 onClick 覆盖子元素（v4.348 Dropdown 坑家族），子按钮原 onClick 必须移除；无 forwardRef 按钮包不上改命令式③确认钮定位一律 testid（okButtonProps 透传）防「删 除」空格名④go test 全量负载 flaky（足迹零交叠复跑绿）；ci 日志勿 tail 截断否则 FAIL 无包名。
- **门禁**：tsc 0+eslint 0/0（新色值走 // hex-exempt）+全量 ci.ps1 绿（vitest 384 文件 3280 例）+drift OK@706+版本三处 4.350.0；产物见 releases/v4.350.0.md。**文档**=releases/v4.350.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十五迁：v4.347 入 archive）+progress/todos。**未做（下刀池）**=boot 亮色主题启动反闪（index.html 内联预读主题，动 splash 链路）；MarkdownContent pre 底 rgba(0,0,0,0.3) 亮主题脏块；白色半透明 hover 底亮主题失效群（PersonaPicker/imagegen）；--color-scheme 声明；ModelCenter ctx context 拆分（state/actions 两通道）；绘梦历史无上限渲染+dataURL 全量回填（挂载内存膨胀）；ForeshadowPanel 行 memo。

## 最新发布：v4.349.0（2026-09-19）「前端优化五线：可访问性动线 / 亮态可读性 / 运行开销 / 布局 / 稳健性」

- **动机**：用户口径「优化迭代 gaea，进行前端优化」。先四路并行**只读审计**（主题可读性/交互可访问性/渲染性能/布局响应式，各带 file:line 与实测数字），再按证据定刀；审计判错的项当轮辨伪不改。纯前端 18 源码+9 测试，**零 Go 逻辑改动、零绑定变更**。
- **线1 可访问性动线**：记忆中枢全局检索命中行（主入口动线）裸 div → `role=button`+`tabIndex=0`+Enter/Space；轻语记忆面板标题/摘要键盘化 + 确认/取消/编辑/删除四图标钮补 `aria-label`（此前只挂 Tooltip，**不参与可访问名**）；阅读「删除高亮」补 `Popconfirm`（误点即删不可恢复）；Lightbox 关闭钮补名；批准弹窗 `aria-modal` false→true。
- **线2 亮态可读性**：①**真缺陷**——`changesdiff-tok.css` 写死 One Dark 七色而 `ChangesDiff` 容器不设背景（继承面板=亮态浅底），实测 `.tok-typeName #e5c07b` **1.38:1**、`.tok-string` 1.61:1（文件注释自称「浅色下同样可辨识」被证伪）→ 接共享调色板 `--hl-*`（真源 `hljs-theme.css` 已备浅色深阶组并随 data-hl 切；ChangesDiff 显式 import 真源防懒加载链 var 未定义），文件零字面色值；②`lightFn` 下沉：warning `#d97706`(2.90~3.13:1)→`#92400e`、success `#059669`→`#047857`（Moss lime 阶保留）、primary Jade/Rose/Amber→`#0f766e`/`#be123c`/`#b45309`（accentRgb/glow 跟随）；**Violet/Moss/Slate 原达标一律不动**；③新增 `appStore.contrast.test.ts` 12 主题×5 令牌×2 背景矩阵机检 + 深阶锁值 + hljs 浅色组逐令牌 ≥4.5 + tok 文件零字面色值。
- **线3 运行开销**：①`main.tsx` 诊断层 500ms 轮询只写一个时间戳（≈17.3 万次/天纯空转）→ 心跳改由既有 rAF 探测循环每帧维护（跨 IIFE 共享 `mainThreadBeatAt`），诊断层只做 8s 低频判读；**降级路径补 500ms 兜底续跳**（否则误报卡死）；②`getThemeTokens()` 每次返回新对象且留在渲染体 + `ConfigProvider theme={{…}}` 内联字面量 → 每次渲染重跑 ≈45 个 setProperty 与 antd 主题算法 → 双 memo 化；③`SelectionToComposer` 的 selectionchange 先判 `isCollapsed` 短路（原上来就 `toString()` 物化整页选区文本）；④`WhisperMemoryList` 过滤/分组 useMemo + 查询词预小写（原每次渲染 ≈7N 谓词）。
- **线4 布局**：摘要 chip 补 `whitespace-nowrap + shrink-0`（窄列曾标签内折行、「基线 N」整排参差）；基线槽位行 `flexWrap:'wrap'` + 工期/时间戳固定段 `nowrap/flexShrink:0`（原固定部分 326px>minWidth 300 致行内换行）；海报描述补 line-clamp（`overflow:hidden` 定高卡原先静默裁切无省略号）。
- **线5 稳健性**：①书架 `loadProjects` 只 `console.error` → 首页把「读失败」渲染成「书架空空如也」**假空态** → store 增 `projectsError`（只表态不抛出，调用方 await 序列零变化）+ 首页错误态含原因与重试；②「最近文档」仅挂载时读一次而 v4.346 起页面常驻保活 → `lib/recentFiles` 增订阅通道（写入广播 + 解析缓存保快照引用稳定，满足 `useSyncExternalStore`），首页账页与办公「最近文件」条同改订阅（空态挂载后首次写入也出现）。
- **审计辨伪（记录在案，当轮不改）**：①印泥章 `.w-seal #d06055` 不改——20px/**700 粗体**属 WCAG large text（阈值 **3:1**），实测白底 3.83/近白 3.49~3.67 **达标**，按 4.5 判死是误报，改深会漂移品牌记号色（阈值已写入测试注释）；②reduced-motion **无缺口**（`index.css:461` 全局 `*` 兜底经 main.tsx 确认已加载 + GSAP 三 hook 均真实调用 `prefersReducedMotion()`）；③暗态 6 主题×27 配对 **0 失败**（最差 4.69）；亮暗两态 `colorText/colorTextSecondary` 本就 ≥4.63——亮态问题**只**集中在 primary/warning/success 被当文字用；④硬编码 `rgb(255 255 255)` 多落在固定深色 scrim（character-detail.css:141/488）上 = 正确豁免。
- **测试**：新增 4 文件 44 例（对比度矩阵 18 / 记忆面板 5 / 阅读浮层 3 / 列表 memo 5 / 结构守卫 13）+ 扩展 5 文件 8 例（命中行键盘 2 / 首页失败重试 1 / 最近文件广播 2 / 挂载后写入 2 / 选区短路 1）。
- **门禁**：tsc 0 + eslint **0 error 0 warn**（全仓）+ **全量 ci.ps1 绿**（vitest **382 文件 3270 例**）+ drift OK@706（零绑定）+ 版本三处 4.349.0；**首轮 ci 被 frontend build 拦下**（新增测试文件后未重跑 tsc：`[...NodeList]` 需 DOM.Iterable、ReadingAnnotation 缺 nodeId/createdAt）→ 修复后复跑全绿。
- **目检**（vite dev ?mock=1 + 无头 Edge CDP 9347，只读，收工已回收进程与临时 profile）：三页错误边界 0/空白 0；检索「振动锤」5 条命中**全部 `role=button`+`tabindex=0`**；亮态实测 `--md-sys-color-primary #0f766e` / warning `#92400e` / success `#047857` 与锁值一致；`.p-poster-desc` 实测 `webkitLineClamp:4`+`overflow:hidden`；`.p-poster.is-hero` 中段空白实测 **273px**（卡高 420，与审计 276px 吻合=本轮有意不动的拍板项）；console 仅 antd Spin tip 警告 + 自诊断 longtask 66ms，**无 exception**；截图 `.tmp/uiwalk-v4349/*.png`+report.json。
- **产物**：exe **50,865,152 B（48.5 MB）** SHA256=**615319a09773dc693c2be6d943904bae226dd686d797291c905d605b5174ecea**（releases+SUMS，格式 `hash *file` 与历版一致；桌面副本同哈希）；冒烟 health 200 过；保留策略执行留 v4.345~v4.349 删 v4.344；顺带纠回 `releases/README.md` 的「最近 5 版」行与「34 席」索引（此前落后三版）与发布说明计数（254→371）。
- **坑**：①审计阈值必须复核——large text 3:1 被按 4.5 判死会白改品牌色②`--hl-*` 是懒加载 CSS 变量，接色板必须同时 import 真源③source-guard 里 `indexOf('.hl-scope-dark')` 会命中文档头注释，取色块须 `lastIndexOf`④antd 两字按钮可访问名带空格（「删 除」「重 试」），测试按 `/^删除$/` 找不到⑤心跳改 rAF 驱动必须处理降级路径，否则每 8s 误报卡死⑥**新加测试文件后必须重跑 tsc**（tsconfig.app 含 src）。
- **未做（下刀）**：海报墙 hero 中段留白 276px（须加内容槽=内容决策，改动需同时验 1440/1180/1100/760 四断点）；其余可点 div 键盘化（章节树/大纲/统计分类 ~8 处）与 `ResizableDrawer` 手写弹层焦点管理按动线权重分刀；进度计划页 `.gsched` 工具栏 spacer 换行与基线 Popover **本轮未实测**（mock 书斋 rail 无「进度计划」入口，代码级证据在案待真机）；跨窗口最近文件同步（storage 事件）本仓单实例无场景。

## 最新发布：v4.348.0（2026-09-19）「原罪板块优化双刀：故事底稿直注前情 + EPUB 电子书导出」

- **动机**：原罪板块优化完善班。①故事越长前情窗口（12 条×1500 rune）越丢早期设定，模型自记的便签/大纲底稿不在提示词里——要花工具轮 read 自己记的设定，忘 read 就自相矛盾；②故事成品只有 Markdown（插图本地路径引用），阅读器/传设备缺内嵌插图的可分发格式（go-epub 已在依赖，书源线有先例）。
- **刀1 底稿直注**：buildSinUserPrompt 增底稿块（角色块后、前情前）——大纲防御性再截 4000 rune + 便签合计预算 sinPromptNotesBudgetRunes=4000 从头带，超限「其余 N 条未展示」如实报数；空稿=提示词逐字零漂移；回合中途 write 下一轮生效；sinDraftForPrompt fail-open+锁内读。工具描述/系统提示词同步改口「底稿已随前情附上」，通常无需 read（省工具轮），核对单条全文才 read。
- **刀2 EPUB 导出**：SinExportEpub（705→706，play）——每条助手回合一节（第N回中文序数）+用户指令引块置节首+插图内嵌（go-epub AddImage；首图兼封面；webp 不在支持集→「插图未能内嵌」占位）+未生成占位（cue 次序键优先/描述键兜底同 MD）；落 sin/exports 同名不覆盖-2/-3、半截产物即删；前端顶栏导出改下拉（Markdown 另存为/EPUB 后端落盘告知路径，mock 如实抛）。
- **测试**：sin_draft_test 5 用例（空稿逐字一致/注入排序/预算截断+未展示计数/大纲便签独立判空/fail-open）+ sin_export_epub_test 2 用例（zip 全链：正文/引块/img/占位/illu-1 恰 1 张/cover.xhtml/同名-2；守卫：空故事/聊天话题拒绝）；既有 buildSinUserPrompt 三处调用点补 sinNotesDoc{} 断言零改动。
- **门禁**：go build/vet 0+internal/app 全量绿（102s）+tsc 0+eslint 0+vitest 定向 94/94+drift OK@706+spaceBindings 锁 528→529+bindingNames 全量再生+版本三处 4.348.0；全量 ci.ps1 绿；产物=exe 50863104B SHA256=f84e9bd9164f2bd55b1b12220f6963c6251f39247abad8dba1bb8a6a0249ed00（releases+SUMS；桌面副本同哈希；冒烟 health 200 过）；本地 exe 保留策略执行留 v4.344~v4.348 删 v4.343；
- **坑**：①Dropdown clone 子元素注入 onClick 会覆盖 ToolbarButton 的 prop（无 forwardRef 且 onClick 必填）——占位空函数即净，不为原罪改办公组件②单条恰=预算的便签 runes>room 为假应整条收下——截断语义「超了才截」，测试前提咬一次③go-epub AddImage 收本地路径返回值直接作 img src，webp 宁占位不硬塞。
- **未做（下刀）**：sin 真机一条龙走查（书源卡+导出下拉+底稿直注实写）挂闲置窗口池；插图变体历史候立项；章节化结构候拍板。

# 任务进度

> 本文件为**最近发布速览**。完整历史磁带见 `docs/archive/progress-history-2026-09.md`
> 与 `releases/`。

## 最新发布：v4.347.0（2026-09-19）「GLM-5.3-FlashX 新模型接入目录 + Flash 补官方绝对价」

- **来源**：智谱 9 月中旬新发 GLM-5.3-FlashX（Flash 加速版，320B 总参/18B 激活，200 tokens/s）；官方四页交叉核实（模型概览/详情/定价/coding 活动）后接入模型中心 GLM 静态目录。纯数据刀，零新绑定 705。
- **落地**：`glm_catalog.json` +1 条目（1M/128K、caps 同 Flash、国内绝对价 2/7 元/M、缓存命中 0.57 记 price_note）；coding 套餐官方明示暂未开放 FlashX→不配积分系数不进别名；Flash 补绝对价 0.8/2.8 元/M（「仅相对价回退 USD」旧口径作废，估算 0.65 USD→3.6 CNY /M+M）。
- **测试**：锚定清单+计数 45→46 全对齐 ×8；FlashX 元数据/价格锁值；积分守卫+估算锁值翻新；modelengine 包全绿。
- **验收**：drift OK@705；全量 ci 绿；产物见 releases/v4.347.0.md。
- **未做（下刀）**：coding 套餐若开放 FlashX 补 points/别名（覆盖文件可先行）。
## 最新发布：v4.346.0（2026-09-19）「UI 健壮性双修：错误边界页级隔离（keepAlive 连坐根修）+ TisorRadar 缺档崩页」

- **来源**：UI 优化班——隔离目检（vite dev ?mock=1 + 无头 Edge CDP）双空间全页×明暗两态；纯前端四文件，零新绑定 705。
- **根修**：①MainLayout 错误边界下沉 keepAlive 每页内部（原包整组外面=任何一页崩卸载全部页且后续导航全停错误态，keepAlive 状态俱失）②TisorRadar dims 守卫+CharacterCard 调用方守卫（缺档不渲染不出假图）③mock 三角色补 dims 对齐真契约。
- **测试**：TisorRadar.test +3 +CharacterCard.test +1（按 circle 计数断言）定向 19/19；tsc 0。
- **验收**：修复前角色库明暗皆崩；修复后 3 卡×30 circle 雷达在位、双空间明暗全绿；全量 ci 绿；drift OK@705；产物见 releases/v4.346.0.md。
- **未做（下刀）**：亮态 accent 对比度打磨（视觉拍板维持观察池）；动效手感待上手定论；真机走查清池班；技能核数 ≈09-30。
## 最新发布：v4.345.0（2026-09-19）「filewatch 关闭竞态根修：fs 通道关闭路径漏 close(out) 致消费方挂起」

- **来源**：全量 ci 偶发时间型 flaky 深挖——预算拉满 10s 仍挂=真缺陷非调度慢。
- **根因**：loop 三退出路径中 fs 通道关闭两条裸 return 漏 close(out)，与 done 分支 select 竞速，输则消费方 range Events() 永久挂起（~1/20）。
- **修复**：outOnce+closeOut() 三路径幂等关闭；测试 2s 硬超时改终态轮询 10s 预算。
- **验收**：filewatch 20 连跑全绿（修前 1/20 挂）；全量 ci 绿；产物见 releases/v4.345.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.343.0（2026-09-19）「t7 收尾「编辑器标注高亮」轻量版：定位单条镜像 mark + 编辑即失效」

- **来源**：观察池最后一项（v4.325 有意裁剪）的轻量落地；纯前端 EditorPanel+一条 CSS，零新绑定 705。
- **落地**：locate 时镜像背景层（同排版/文本透明仅 mark 显色/textarea 底透出）；滚动条占宽补偿+滚动同步；content/activeNode 变化即失效（偏移漂移结构性规避）。
- **测试**：EditorPanel 域 11/11（+1）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.343.0.md。
- **未做（下刀）**：真机走查清池班（镜像对齐需真机目检后再议持久全量）；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.341.0（2026-09-19）「t7 观察池「情感曲线图形化」：全书体检面板情感弧线折线」

- **来源**：观察池条目（v4.325 记录）；纯前端单文件 BookHealthPanel，零新绑定 705、零新依赖。
- **落地**：体检面板情感曲线区——按需逐章拉分析 V2 情感弧线强度（0-10，缺档跳过计数），纯 SVG 折线（参考线+tooltip+间隙如实呈现），<2 章诚实不足态。
- **测试**：BookHealthPanel 域 7/7（+4）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.341.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 余项候拍板。
## 最新发布：v4.344.0（2026-09-19）「t7 最终形态：编辑器持久标注高亮（overlay 常驻 + 编辑退场）」

- **来源**：v4.343 轻量版完全体（真机镜像对齐已过）；纯前端三文件，零新绑定 705、零新依赖。
- **落地**：annotationMarks.ts 分段工具收敛（面板/编辑器同一实现）；EditorPanel annotations 驱动全量 mark 常驻 + dirty 编辑退场 + gutter 实测；CreatePage 随激活章加载标注（alive 守卫）；locate 契约不变。
- **测试**：EditorPanel 11/11（重写+2）+ Panel 11/11（重构等价）+ CreatePage 15/15（补漏桩）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.344.0.md。
- **未做（下刀）**：真机走查清池班（持久 overlay 目检随下一班）；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.342.0（2026-09-19）「t7 观察池「多章对比」最小形态：分析面板章际对比」

- **来源**：观察池条目最小可用形态；纯前端两文件，零新绑定 705。
- **落地**：分析面板「章际对比」——选对比章拉其 V2，四维+情感强度差值表（Δ 上绿下红、情感中性）；未分析行内提示；chapterOptions 缺省整体隐藏。
- **测试**：Panel 域 11/11（+3）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.342.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 仅剩 overlay 高亮候拍板。
## 最新发布：v4.340.0（2026-09-19）「t2 观察池「反推任务取消绑定」：AI 反推大纲轮询期可取消」

- **来源**：观察池条目（t2 拆书线 v4.291 体验余项）；纯前端单文件 CreatePage，零新绑定 705。
- **落地**：轮询期「取消反推」钮（taskId 到手门控）→ 停止等待 + GaeaTaskCancel 尽力协作取消（完成竞态被吞，任务中心权威）；同步回落路径不出入口。
- **测试**：CreatePage 域 15/15（+2）；补 NovelOutlineReconstruct/TaskCancel 两处漏桩。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.340.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.339.0（2026-09-19）「t7 观察池「标注定位编辑器光标」：分析面板→正文编辑器定位闭环」

- **来源**：观察池条目（v4.325 记录）按既定习惯开刀；零新绑定 705，纯前端三组件。
- **落地**：EditorPanel forwardRef+locate 句柄（rune→code-unit 换算+选区+聚焦滚动）；分析面板标注行「编辑器定位」钮（stopPropagation）；CreatePage 接线（gate 跳他章时入口按章一致条件隐藏）。
- **测试**：EditorPanel +3 + Panel +3（18/18）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.339.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 观察池余项候拍板。
## 最新发布：v4.338.0（2026-09-19）「无参绑定会话语义审计收官 + 死链清理（绑定面 707→705）」

- **来源**：todos 挂账（v4.181.0 起池）收官；审计档 docs/gaea-session-binding-audit-2026-09.md。
- **审计结论**：需修 0 项——会话模型「看历史必经 ResumeSession 切内核」，主流水线无参读内核构造性自洽；v4.181 旁路看板族无同构残余；复审口径入档。
- **顺手清账**：删 GaeaCheckpoints 死链（UI 回退实走 GaeaRewind）+GaeaTCCAReport 孤儿链（state.tcca 零消费）；绑定面 707→705。
- **拍板池**：GaeaSkillDraftFromSession 参数化（从历史会话蒸馏技能）候拍板。
- **验收**：go build/vet 0+internal/app ok；tsc/eslint 0；store 域 38/38；drift OK@705；全量 ci 绿；产物见 releases/v4.338.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；TaskCenter 会话关联结构刀。
## 最新发布：v4.337.0（2026-09-18）「办公对话流美化三轮收官：对齐 Codex web（降噪+动线+语义归位）」

- **来源**：办公对话流美化线程收口发版（在途三轮合一版）；纯前端零新绑定 707，零 Go 改动。
- **落地**：用户消息气泡化；裸思考行 bare 模式（行数 meta 三语+英文单复数）；过程卡完成即收起；WorkHeader 三改（doneSuppressed/纯问答 0 步不渲染/运行态对齐过程条）；尾随工具段合并（过程卡归位正文前）；轮尾登记卡补挂+omitPaths 跨段去重；交付卡统一边框容器；消息操作 hover 淡入；顶条只留 waiting 档；ToolGroup「bash × 3」三语。
- **测试**：ProcessCard 展开态测试重写钉新行为（旧测试钉已废行为，全量首跑 1 红）；vitest 3199/3199；tsc/eslint 零；三语 +4 键同轴。
- **目检**：vite dev 9344 mock=demo+独立 Edge 9345 CDP 两轮五态截图全过（bare「思考 1 行」实测）。
- **验收**：全量 ci.ps1 绿；drift OK@707；产物见 releases/v4.337.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。
## 最新发布：v4.336.0（2026-09-18）「chapter-gate 通知跳转 + oh-story T5 落库接线」

- **来源**：用户指令「继续」；规格书 进度计划/gaea-gate-jump-t5-wiring-20260918.md。
- **甲件**：自动门通知可点击——onOpen 回调+CreatePage 覆盖态+跨章拉正文锚定；无回调零变化（v4.331 观察池头名收口）。
- **乙件**：T5 七角色卡升格可派发——SKILL.md runAs=subagent+派发契约；spawn 写作域模板 subagent_writing+boot 映射；验收钉 Go +3。oh-story 只剩写前契约硬闸（等上游）。
- **验收**：全量 ci 绿；drift OK@707；产物见 releases/v4.336.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。

## 最新发布：v4.335.0（2026-09-17）「办公输入动线对齐 DSH/Codex：运行中 Enter=排队，插话改显式」

- **来源**：用户报告「办公板块消息输入的排序/插话/撤回不见了，直接插入正在跑的对话中」；拍板「按 DSH/Codex 方式处理」。
- **归因（非回归）**：队列本体（v4.200/201）一直在，v3.6.0 起 Enter=插话、排队退到 Clock 小按钮——队列卡从未被 Enter 动线触达=体感丢失。
- **落地**：Enter=排队（队列卡回主动线，回合结束 FIFO 派发）；Alt+Enter=直插（显式插话）；Shift+Enter 纠正/Esc 停止不变；三语文案+title 三态更新；steerTitle 键删（0 死键）；组装逻辑抽 helper。零绑定 707。
- **验收**：Composer.queue +3/I18n 对齐；composer 域 22/22；tsc/eslint/ci.ps1 全绿；产物见 releases/v4.335.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。

## 最新发布：v4.334.0（2026-09-17）「revolution 刀8 收官：跨模块验收+回退演练+版本三处漂移根治——**revolution 刀 1–8 全线收官**」

- **来源**：用户指令「继续优化迭代 gaea」；三线并发子代理+主代理收口；规格书 进度计划/gaea-rev-knife8-acceptance-20260917.md。
- **跨模块验收 Go +7（零生产代码改动）**：线A=AI 味性能钉（3.3~5ms<1s）+去味保真（分降/rune 漂移 0/实体句数不变）；线B=生成主轴端到端史上首测（双路桩贯通自动门成功路径+chapter-gate 事件 httpbridge 捕获）+50 章书级体检；线C=审批链 app 级（approved=false 零入库/journal 回放一致/账本只增）+快照往返+真 v3 迁移非破坏。
- **版本漂移根治**：app_info.go/versioninfo.rc 停 4.323.0（十版未跑 sync-version）——三处同步 4.334.0+ci.ps1 新增 version drift check 段；顺手 eslint 5→0（EXTRACT_OPTIONS 归位 novelOptions.ts）。
- **验收**：go vet/test 全量绿；vitest 3185/3185；tsc 零错；drift OK@707（零新绑定）；产物=exe 50832384B SHA256=4d2a01c5…81bc（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：RetryJSON 桩回包须包 ```json 围栏；rebuildScenesFromBlob 无场景章 no-op；project.Create 直写 v4 标记（旧 v3 夹具假设过时）。
- **未做（下刀）**：真机走查清池班（v4.319~v4.334 挂池，须闲置窗口）；技能核数 ≈09-30；绘梦阶段二与缺省形态翻转候拍板。

## 最新发布：v4.333.0（2026-09-17）「阶段七出口对账 + 7.3-1 会话级回源收口」

- **来源**：用户指令「继续」；规格书 进度计划/gaea-stage7-audit-resume-20260917.md。
- **会话级回源**：pendingSessionResume 双态通道（keepAlive 事件/冷启 pending）+gaea App 消费（ref 防过期闭包）+收件箱精确回源（V1 板块粒度回退零变化）。零新绑定 707。
- **对账**：7.1/7.2/7.3 已落、7.4 已清；挂账=本地建议样本（真机）/复跑抽样+≥5 次（≈09-30）/总判据⑤走查班。
- **验收**：前端 +5；go 全量复跑绿+vitest 3185/3185；ci 第三跑 OK（首两撞环境竞争先例+2）；产物=exe 50832384B SHA256=89d1de3d…0ba2（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **未做（下刀）**：真机走查清池班（须闲置窗口）；技能核数 ≈09-30；刀8 余项/绘梦阶段二候拍板。

## 最新发布：v4.332.0（2026-09-17）「7.3-2 板块降级为任务视图（阶段七『层跃升』刀）」

- **来源**：用户「开始吧」提前拍板解锁（原窗口 ≈09-23）；规格书 进度计划/gaea-tasks-first-home-20260917.md。**本版=阶段七层跃升版本**。
- **论点**：首页从「用哪个工具」翻转为「什么在等我」。裁决=homeLayout 开关缺省 classic（渐进零变化）；TaskInboxBoard 内联抽取（面板零破坏）；回退开关双位；零新绑定 707；manifest 零改动。
- **落地**：appStore homeLayout+守卫；TasksFirstHome（收件箱大区/最近会话/能力 chips 全量直达/切回经典）；ModuleLauncher 分支+SpaceSwitch 快捷钮；设置页 HomeLayoutPanel；三语 12 键。
- **验收**：前端 +10；零回归实证（TaskInboxPanel 9/9+ModuleLauncher 13/13 零改动）；vitest 3180/3180 一次全绿；ci.ps1 全绿；产物=exe 50831872B SHA256=80dacda4…d7eb（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：测试尾追加吞 describe 闭合；LocaleProvider 包裹；ESM 无 require。
- **未做（下刀）**：阶段七出口判据复核对账（三门收口）；刀8 余项；绘梦阶段二提案候拍板。

## 最新发布：v4.331.0（2026-09-17）「自动门可见化 + CI race 扩面（刀8 race 项收口）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-gate-notice-race-20260917.md。
- **论点**：自动门零感知（事件无消费）+race 门禁只盖办公包。落地=useChapterGateNotice 轻通知（四维拼装/防叠/分支标记/无通道静默）挂 CreatePage；CI race job 扩面六包（app 注释不进候评估）。零新绑定 707。
- **验收**：前端 +3；vitest 3172/3172 一次全绿；六包本地全绿；ci.ps1 全绿；产物=exe 50824704B SHA256=a5e239f2…3198（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：vi.stubGlobal('window') 整窗替换炸 react-dom——只换 window.runtime 属性。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；刀8 余项（验收演练）候排。

## 最新发布：v4.330.0（2026-09-17）「刀7续收官：POV 视图（场景圣经消费面）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-pov-view-20260917.md。前置核对：场景生成/脑图/指纹已落，刀7续仅剩 POV 视图；revolution 勾选对账后只剩刀8。
- **论点**：POV 掩码只在生成链内部消费，视角纪律不可审视。裁决=+1 绑定（707）双路编译（场景/整章合成）；视图 camelCase 恒非 nil；空区段如实空。
- **落地**：novel_scene_bible.go wire 投影；SceneBibleDrawer（已知绿/不知情红对照+角色卡状态回灌）；ChapterEditor「视角」按钮。锁 529→530；drift OK@707。
- **验收**：Go +3+前端 +3；ChapterEditor 7/7 零破坏；vitest 3169/3169 一次全绿；ci.ps1 全绿；产物=exe 50823680B SHA256=4ea48652…4eeb（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；刀8 race 门禁（CI 侧）。

## 最新发布：v4.329.0（2026-09-17）「绘梦阶段一刀 E：画室消耗（月度聚合+折叠面板）——绘梦阶段一全清」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-studio-usage-20260917.md。**绘梦阶段一（A 资产面板/B 参考槽/C 指令编辑/D 配图 v2/E 画室消耗）全清**。
- **论点**：台账有 Cost+CreatedAt 无聚合视图。裁决=+1 绑定（706）；当月过滤×模型分组；记录单价为事实源（缺价回落目录）；可解析才估算、未定价诚实计数；全免费显示 0；不用积分话术。
- **落地**：imagehub_usage.go 聚合（cwd 参数化供测试）；StudioUsagePanel 折叠面板挂 ImageGenPage；bridge/mock/锁 529/bindingNames 706。
- **验收**：Go +4+前端 +4；vitest 3166/3166 一次全绿；ci.ps1 全绿；产物=exe 50809856B SHA256=90e9ec56…ad78（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：裸 "0" 单价解析、gaeaCwd 全局态、mock 尾锚点。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；绘梦阶段二规划拍板。

## 最新发布：v4.328.0（2026-09-17）「绘梦阶段一刀 D 核心片：章节配图管线 v2（角色参考图+风格槽）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-scene-illustration-v2-20260917.md。
- **论点**：配图纯文生图（一致性靠文字、风格写死）。裁决=签名扩展空串零变化（绑定面 705 不变）；参考路由按后端能力附 img2img（SceneRefDenoise=0.6）或 refNote 诚实降级；风格槽替换风格描述。
- **落地**：chapter.Agent +V2 方法；handler opts+参考路由+readImageFileAsDataURL；前端风格输入+角色勾选+refNote 行；修复挂载效应自动重跑 bug（didMountRun 守卫）。
- **验收**：Go +7+前端 3 改 5；vitest 3162/3162 一次全绿；ci.ps1 全绿；产物=exe 50796032B SHA256=070f39f8…6c39（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **观察池**：书封参考；素材库变体/设书封；灯箱共用；denoise 可调。
- **未做（下刀）**：刀 E 画室消耗轻量面板或 7.3-2（≈09-23）。

## 最新发布：v4.327.0（2026-09-17）「绘梦阶段一刀 C：指令编辑『改图』MVP（云端先行）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-instruct-edit-20260917.md。前置核对：刀 A/B/E 已落，刀 C 是真缺口（改图=整幅重绘无语义编辑）。
- **论点**：gaea OpenAI 兼容图片面缺 /images/edits（Qwen-Image-Edit 系云端网关标准面）。裁决=复用请求字段（Mode=edit）；三后端诚实口径（openai 实现/glm·comfyui 拒绝）；零新绑定 705 不动。
- **落地**：image_openai.go editImage multipart 分支+parseImageResponse 共用提取；GenerateMedia edit 原图必填；InstructionEditModal（对照+用到画布）+ResultStage「指令编辑」Action+ImageGenPage 接线。
- **验收**：Go +5+前端 +4；ResultStage 5/5 零破坏；ci.ps1 全绿（FilePreviewModal 1 例 flaky 隔离绿，两日两现候选常驻）；产物=exe 50785280B SHA256=855b56f1…9ae0（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **观察池**：mask/保留区域；ComfyUI 本地档；多图输入；edit-of-edit 谱系；刀 D/刀 E 轻量收尾。
- **未做（下刀）**：绘梦阶段一余项（刀 D 章节配图 v2/刀 E 画室消耗）或 7.3-2（≈09-23）。

## 最新发布：v4.326.0（2026-09-16）「GenerationGate 闭环收口：生成后自动分析门 + 全书体检」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-gen-gate-closure-20260916.md；阶段七 §4 在册项（revolution「GenerationGate 完整闭环」未勾项收口）。
- **论点**：生成 done 只挂去味+AI 味分+角色提取，分析路（V2 落盘/伏笔同步/记忆回填）从不自动发生——三条已建成管道水源靠手动。裁决=自动门异步默认武装；只跑确定性三路+分析路一次 LLM（review/consistency 留手动）；分支章跳分析路；chapter-gate 事件零消费（t7 面板天然消费面）；全书体检纯确定性零 LLM。
- **落地**：novel_book_health.go（buildAutoGateReport 同步可测+runAutoGateAfterGeneration go 位+RunBookHealthCheck 逐章三路+伏笔 Lint 复用+V2 覆盖）；create_chapter_handler done 后挂接；NovelB +1（704→705）；BookHealthPanel（聚合卡+最差 AI 味告警+逐章表红标+伏笔 findings）；锁 527→528；drift OK@705。
- **验收**：Go +5（自动门三例+体检两例）+前端 +5；CreatePage 既有测试零破坏；tsc/eslint 零错；产物=exe 50774528B SHA256=0581c0be3ea4a61f33989eecf4ebd666a287cd648a10ec2a21e5da304aa5f83b（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：analysis.New(nil client) 在 Analyze 内 panic——测试用无引擎真 client；JSX 泛型不认索引访问抽别名；无大纲节点=无契约可违不计。
- **出口**：生成后 V2/伏笔同步/记忆回填自动发生；全书体检聚合报告零 LLM；revolution 未勾项勾销。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；闲庭在册清欠沿列车。

## 最新发布：v4.325.0（2026-09-16）「t7 前端统一接线收官：分析 V2 消费面板 + 标注高亮视图」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-analysis-v2-panel-t7-20260916.md。**本刀合入=小说·MuMu 蒸馏六域前端接线全清（t7 收官）**。
- **论点**：分析 V2（九维落盘）与标注层（keyword→rune 偏移）后端在位 UI 零消费；AnalyzeChapter 困在 Legacy 面零入口。裁决=types 直连（不复制九维子类型树）；缺章诚实报错接分析按钮闭环；高亮 V1=面板只读视图不动编辑器；rune→code-unit 换算+交叠裁剪。
- **落地**：analysis_handler.go +NovelChapterAnalysisV2（703→704）；AnalyzeChapter 出 Legacy 转正（AppBindings+mock）；ChapterAnalysisPanel（头部评分+chips+meta、九维分节空节隐藏、标注区 列表|高亮视图 双模式+锚点定位）；CreatePage rail「章节分析」。锁 525→527；drift OK@704。
- **验收**：Go +3+前端 +8（含交叠裁剪断言）；CreatePage 13/13 零破坏；vitest 全量 3151 例（1 例环境 flaky 隔离绿）；tsc/eslint 零错；产物=exe 50756096B SHA256=6604f8e06b3f27cd586053794bfe22d8b881851e72210a7e7b0e2aeb22d89528（releases/gaea-v4.325.0.exe+SHA256SUMS-v4.325.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）
- **出口**：分析按钮→V2 落盘→九维可见；标注列表+只读高亮+锚点；AnalyzeChapter 出 Legacy；既有面零回归。
- **观察池**：编辑器 overlay 持久高亮；标注定位编辑器光标；多章对比；情感曲线图形化。
- **未做（下刀）**：7.3-2 板块降视图（等 v4.318 零功能周≈09-23）；闲庭在册清欠沿既有列车。

## 最新发布：v4.324.0（2026-09-16）「t6-C2 提示词工坊第二刀：模板包导入导出（content_hash 三态）」

- **来源**：用户指令「继续优化迭代 gaea」。小刀单线主代理直做；规格书 进度计划/gaea-prompt-bundle-t6c2-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md §6.4+§11.4。
- **论点**：覆盖表只有一份本地状态——换机/重装无搬运手段、内置升级无对账工具。补全量快照 JSON 包+三态导入（kept/converted/created_or_updated + gaea 三闸 invalid/unknown/duplicate）；对 MuMu 升级=导入真校验（error 跳行不阻断整包）+去重；裁剪=未知键不建行。
- **落地**：internal/promptstore/bundle.go（ContentHash/SameTemplate canonical/BuildBundle/ImportBundle 零 IO 纯函数）；gaea_prompt_store.go +2 绑定（Export 回 JSON 字符串/Import 三态落盘）；面板顶栏导出（saveExportBlob 双门）/导入（pickFileAsFile 双门+结果 Modal）；mock/契约同步。绑定面 701→703（NovelB +2 显式覆盖表归域——v4.323 同款坑复现即修；play 锁 523→525）。
- **验收**：Go +4 测试函数（三态矩阵 10 分支+round-trip 内核+回魂 e2e）+前端 +4；tsc -b 零错；vitest 全量 3143 例（1 例环境 flaky 复跑绿）；drift OK@703；产物=exe 50734592B SHA256=5c1b3916518cd268f84ec9dfc6b61a3b3746c7d12f778151c1076736380d0abc（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **出口**：干净环境导入覆盖逐字节回魂；内置升级三态分支钉死；坏模板行不落盘整包其余正常；壳内导出/导入双门。
- **观察池**：项目级 scope；自建模板键；升级合并交互；按书切换配方；triggers；模板包签名/加密。
- **未做（下刀）**：t7 前端统一接线（小说线最后一域）；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.323.0（2026-09-16）「t6 提示词工坊首刀：模板可编辑覆盖层（{{name}} 渲染+三级解析+工坊面板）」

- **来源**：用户指令「继续优化迭代 gaea，记得使用子代理」。三线并行子代理（A 引擎纯函数/B handler/C 前端）→ 主代理收口；规格书 进度计划/gaea-prompt-workshop-t6-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md。
- **论点**：20 个 prompts/*.json 只读两层用户改不了；占位符单点硬编码；保存零校验（MuMu §9.3 同型）。裁决={{name}} 统一（旧语法兼容）；V1 只全局覆盖；裁剪云端工坊三表；列表零正文。
- **落地**：internal/prompt +四元数据字段+RenderPlaceholders+SetOverride 三级解析+Names；新包 promptstore 六规则校验+Upsert 自增；prompts ×20 元数据+2 文件占位符迁移；gaea_prompt_store.go 状态文件+五 App 方法+两装配点挂钩+substituteWordCount 双语法；PromptWorkshopPanel 三区+api/prompt.ts+CreatePage 入口+mock 五档。绑定面 696→701（NovelB +5 显式覆盖表归域；play 锁 518→523）。
- **验收**：Go +22 测试函数+前端 +12（CreatePage 13/13 零破坏）；tsc -b 零错；ci.ps1 全绿 exit 0（go 130 包+vitest 367 文件 3138 例）；drift OK@701；产物=exe 50754048B SHA256=1a712949e53414c308ae8a14f85de914d95e021b47fc9711f3ae3e880fdcf4d6（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **出口**：20 模板可看可改可恢复；覆盖即时生效（eng.Get 三级解析全链共享）；保存真校验；{{name}} 统一。
- **观察池**：模板包导入导出（content_hash 三态，t6-C2）；项目级 scope；自建模板键；triggers；升级合并交互；风格注入与 novelstyle 打通。
- **未做（下刀）**：t6-C2 模板包导入导出 / t7 前端统一接线（小说线最后两域）；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.322.0（2026-09-16）「t4-C3 收官：场景工程整章重写（单场景替换）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-scene-rewrite-20260916.md。
- **产品裁决**：单场景替换（应用/Restore 对称）；LLM 分场景回写与场景粒度重写入观察池。
- **落地**：移除 whole v4 拒绝+stitch 读；Apply/Restore +rebuildScenesFromBlob（复用既有原语零新机制）；partial v4 守卫；EditChrome 双按钮+ChapterPage 挂两弹窗。零新绑定 696。
- **验收**：Go +2+前端 +2（11/11）；ci.ps1 全绿 exit 0（go 129 包+vitest 365 文件）；drift OK@696；产物=exe 50638848B SHA256=5b1b105ad81bff7b5dd391cc445aeda8cb77aed575a516c996e10441c38ea29e（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **出口**：t4-C3 余项全清（三通道×两存储矩阵仅剩 partial×场景=守卫+观察池）。
- **未做（下刀）**：t6 提示词工坊/t7 前端统一接线；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.321.0（2026-09-16）「t4-C3 余项：partial 选段局部重写（版本库升级版）」

- **来源**：用户指令「可继续」。三线并行子代理（A/B/C 足迹互斥全部一次绿）→ 主代理收口；规格书 进度计划/gaea-partial-rewrite-20260916.md。
- **论点**：整章重写对「改一段」太重；partial=选区+指令+长度模式只重写选中片段。对 MuMu 升级=走版本库（全文快照可回滚）；全链 rune 口径；零新绑定（696 不动）。
- **落地**：rewrite/partial.go 四纯函数+NormalizeRequest 五规则（whole 零字节变化）+novelChapterRewritePartial（拼接钉死前后文零变化+版本字段）+前端 EditorPanel 选区 rune 换算/PartialRewriteModal 三态/CreatePage 挂载。
- **验收**：rewrite 9 函数+app +3+前端新增 6（合并定向 34 全绿）；ci.ps1 全绿 exit 0（go 129 包+vitest 365 文件）；drift OK@696；产物=exe 50638336B SHA256=c0f1abd770b4b4a2253c17e9c92d912f06f7992fbcbc62fe155e1e456a97fa7f（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **观察池**：选段高亮回写编辑器；流式输出；deslop 入口（类型已留）。
- **未做（下刀）**：t4-C3 余项剩场景工程整章重写；t6 提示词工坊/t7 前端统一接线；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.320.0（2026-09-16）「t4-C3 余项：重写版本历史面板（版本库前端消费）」

- **来源**：用户指令「继续」。小刀单线主代理直做（纯前端后端零改动）；契约先行（规格书 进度计划/gaea-rewrite-history-panel-20260916.md）。
- **论点**：v4.304 版本库只有「刚重写完那一刻」的 RewriteModal 能操作，List/Get 两绑定零 UI 消费——关弹窗/换会话后历史版本不可看不可用（v4.305「用户到不了的功能等于没有」同型）。
- **落地**：RewriteHistoryPanel（多行展开+详情懒拉缓存+动作镜像后端状态机门控：completed→应用+放弃/discarded→应用/applied→恢复原文/其余只读；Popconfirm 显式 okText）+CreatePage rail「重写历史」入口（打开才拉）+mock 两样本。零后端：绑定面 696/锁数不动、drift OK@696。
- **验收**：面板 5 用例+CreatePage 既有 13/13 零破坏；tsc/eslint 零错；ci.ps1 全绿 exit 0（go 129 包+vitest 364 文件）；产物=exe 50613248B SHA256=4bc1d56d44a5f97a92fde68d1fb8b18026dec5d0c4a8c6fec19731fca8dca837（releases/gaea-v4.320.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：双栏 diff 视图；partial 入口随 partial 刀；按模式/状态过滤。
- **未做（下刀）**：t4-C3 余项剩 partial 局部重写+场景工程；t6 提示词工坊/t7 前端接线；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## 最新发布：v4.319.0（2026-09-16）「7.2-2 判据②收口：结晶技能调用计数（skill_stats）」

- **来源**：用户指令「继续」。小刀单线主代理直做（体量一轮未拆子代理）；契约先行（规格书 进度计划/gaea-skill-usage-stats-7-2-2-20260916.md）。
- **论点**：v4.317 出口判据②欠账「结晶技能调用 ≥5 次且成功率可见」：现状核实=技能计数是前端会话级（useToolStats 只算当前会话 run_skill，不持久、无成功率、漏 read_skill 主消费通道——boot 按需加载器才是结晶技能被「翻开来用」的时刻）；本刀补**工具级调用计数**：read_skill+run_skill 双钩子 → <DataRoot>/skill_stats.json 持久化 → GaeaSkillStats 绑定 → 能力面板技能行「N 次 · 成功率 X%」。诚实口径（V1）=成功率工具级（read_skill 交付正文/run_skill 管线无错=ok；读不到名也计 fail 不静默）；回合级留观察池；「≥5 次」使用事实门槛不做硬闸。
- **落地**：①新纯包 internal/skillstats（零 IO 表驱动，先例 routesuggest/taskinbox）：Record（空名忽略/MaxSkills=500 新名拒收防漂移堆积/零值 File 自动建表）+View（calls 降序→name 升序稳定）+JSON camelCase 钉死。②boot 双钩子：Options.OnSkillUse（缺省 nil 零行为变化，EmitSubagentText 先例）+sysprompt resolver 包装（read_skill 失败也计）+run_skill 装饰器 skillUseCountTool（args.name 解析失败不计宁少勿扰；CompactDescriptor 回退语义同注册表缺省）。③App：skill_stats.json route_suggestions 同款容错读（缺失/损坏/version≠1 回空表）+原子写+skillStatsMu 串行；recordSkillUse 包级回调；绑定 GaeaSkillStats（OfficeB +1，**695→696**）只读现算。④前端：CapabilitiesPanel open 态拉 app.SkillStats 一次零轮询（reload 随引擎热加载刷新）→SkillsSection 可选 usage prop→SkillRow 徽标行灰字「N 次 · 成功率 X%」（skill-usage-stat；缺省/零调用不渲染，会话 chip 语义不变）+SkillStatView（types/memory）+三语 +2 键+mock 桩+spaceBindings work+1（锁 517→518）。
- **验收**：skillstats 5 函数+app 3 例+前端 2 例；tsc -b 零错、eslint 零告警；ci.ps1 全绿 exit 0 一次过（go 129 包+vitest 363 文件全过）、drift OK@696、版本三处 4.319.0；产物=exe 50606592B SHA256=43739c69ad424a8356a0994525d3d4d93c4dc367e6037e00802bf249b0641de1（releases/gaea-v4.319.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：回合级成功率（点赞点踩/重生成信号另刀）；蒸馏建议 tab 内联已结晶技能计数；两周后真机清池核「≥5 次」。
- **未做（下刀）**：7.3-2 板块降视图（等 v4.318 稳定一个零功能周）；闲庭在册清欠沿既有列车。

## 最新发布：v4.318.0（2026-09-16）「阶段七第五刀：多入口统一任务收件箱（任务持久实体+四入口存为任务+状态机追踪，7.3-1）」

- **来源**：用户指令「继续推进」（沿「继续 并行使用子代理，优化迭代 gaea」习惯）。契约先行（规格书 进度计划/gaea-task-inbox-7-3-1-20260915.md：A/B/C 三线足迹互斥+出口对照）→ A/B/C 三线并行 → 主代理收口。三线全部一次绿，**无卡死接管**（v4.316/v4.317 连续两刀接管后首回全并行一次过）。
- **论点**：四入口（意图中枢/微信/语音/Ctrl+K）v4.5 已汇同一内核（intent.Parse→routeIntent），本刀补「任务」为**持久实体**+统一收件箱：指令除即时执行外可「存为任务」→ 状态机（待处理/进行中/已完成/已放弃）→ 跳回来源板块。**收件箱是清单不是调度器**（零轮询零定时器零后台事件，读写均用户动作触发=「关闭即停」昼夜运转拍板）；space 必带 work/play 隔离（跨空间仅显式，不做移动/复制）；不加新板块（挂既有双空间首页）。
- **线A**：新纯包 internal/taskinbox（零 IO 表驱动，先例 skilldistill）：CanTransition 四条合法迁移（pending→doing→done、pending|doing→abandoned；终态拒迁同值恒 false）+NormalizeTitle（哨兵 ErrEmptyTitle、120 rune 截断）+ValidSource 五来源（ctrlk|palette|voice|weixin|inbox）+ParseTaskID（ti-+12hex 残渣防御）+FilterBySpace（''=全部，GaeaTaskList 变参先例）+Sort（状态组序→组内 UpdatedAt 降序→ID 稳定=行动优先）。9 测试函数 ~104 断言（JSON camelCase 形状逐字钉死）。
- **线B**：intent 新动作 save_task（存为任务X/记个任务X/帮我添加一个任务X/任务：X/待办：X 首尾锚定两式，排提醒后让位——「提醒我存个任务明天开会」归提醒；空标题不命中坠回聊天宁漏勿误；+9 表用例）+状态文件 <DataRoot>/task_inbox.json（{version:1,tasks:[]} camelCase，route_suggestions 同款容错读+temp+rename 原子写）+四绑定（OfficeB +4）：List（FilterBySpace+Sort 损坏回空表）/Save（id 空=新建 pending+ti-+12hex crypto/rand+超 500 拒；id 非空只改 title/note——**Source/Action/Target/Session/CreatedAt 不可变=来源审计链**；source 缺省兜底 inbox）/SetStatus（同值短路零落盘）/Delete；执行层 execSaveTask：语音/微信走 routeIntent→space=gaeaEffectiveSpace() 归一回退 work、assistantID 判 weixin/voice、reply「已存入任务收件箱：<title>」；**Ctrl+K/面板不走后端执行**——dry-run 照常命中，前端直调 Save（shell 空间更准，来源可区分）。app 9 测试函数（camelCase 钉死禁 created_at 泄漏/"WORK" 大写不泄漏/dry-run 零落盘）。
- **线C**：TaskInboxPanel（antd Modal 四档 tab 计数、CanTransition 前端镜像非法组合按钮不渲染、action=navigate→去板块/session→回会话 V1 板块粒度、V3Empty 两分；9 用例）+双空间首页挂点（书斋 w-vitals 第五节 desk-task-inbox；闲庭 p-foot 第五节 garden-task-inbox 内联 gridColumn '1 / -1' 跨全列——固定四列网格零 CSS 改动）+四入口（Ctrl+K 指令卡〔存为任务〕次按钮+save_task「执行」短路直调 Save；面板 query 非空动态项「存为任务『query』」**捕获期 input 监听**镜像输入——CommandPalette 契约不动，观察池列清退方案；语音/微信零前端改动）+lib/types 本地重述 TaskInboxView（herdsman 防环先例）+三语 +28 键（zh=en=zh-TW=1518 逐键相等）。
- **主代理收口**：gen_bindings+bindingNames 再生 695（drift OK@695；bindings_novel 多行委托规整单行 +101/−101 零语义——v4.313/4.317 重排噪音同款甄别）+bridge/core +4（同名前缀无需映射，GetProgrammingWebStatus 先例）+mock/chat 四桩（两空间样本内存态）+spaceBindings 四名 shared（隔离由 space 参数承担，UnifiedSearch 口径）+收口修 5 处 tsc（bindingNames 再生丢 as const→drift 串味；mock vi.fn 空数组推断 never[]→补 Promise<TaskInboxView[]> 完整样本；antd Text 无 rows 配置→Paragraph）。
- **绑定面**：691→695（OfficeB +4）；锁数 513→517（shared +4）；drift OK@695。
- **出口对账**：①四入口来源可区分 ✅（五 source 落库，测试钉死）③空间隔离零泄漏 ✅（space 必带+严格过滤+"WORK" 大写不泄漏）④零常驻 ✅（无轮询/定时器/事件订阅）；②板块级 ✅（Target 精确导航）、**会话级欠账**——按路径恢复 seam 存在（palette sessionItems→onResumeSession(path)）但活在 gaea App 树内首页拿不到，Session 字段已落库数据就绪后续刀接。
- **验收**：taskinbox 9 函数+app 9 函数+intent +9；前端 33/33（Panel 9+ModuleLauncher 12+SearchModal 8+锁数 4）+App palette/export 源级锁 6/6；tsc -b 零错、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（一次过）、drift OK@695、版本四处 4.318.0；产物=exe 50592256B SHA256=63be1263d9077006a5b312f552abb2984787e9f4d3553ecaeb31bfdf3aaeb311（releases/gaea-v4.318.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：命令面板 query DOM 捕获→CommandPalette 可选 prop 清退（约 6 行）；精确回源会话接 seam（面板挂工作台或 resumeSession 提全局事件）；任务模板联动（GaeaTaskTemplates 从模板建任务）；LLM 兜底分类器对 save_task 覆盖；releases exe 留版数待用户拍板（沿 v4.315 观察项）。
- **未做（下刀）**：7.3-2 板块降视图（依赖本刀稳定一个零功能周）或 7.2-2 判据②调用计数小刀（statsFile 先例）；闲庭在册清欠沿既有列车。

## 最新发布：v4.317.0（2026-09-15）「阶段七第四刀：journal 历史蒸馏（重复模式挖掘+结晶审阅+审计链，7.2-2）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 进度计划/gaea-journal-distill-7-2-2-20260915.md：契约+足迹互斥表+出口对照）→ A/B/C 三线并行 → 主代理收口。**坑**=A 线纯函数包子代理跑满上下文**零落盘**——主代理接管代写（卡死判据先例再+1：v4.316 B 线同款「长时间零落盘」）；B/C 两线正常交付（B 10/10、C 12/12）。
- **论点**：journal 证据链（.gaea/work/journal/*.jsonl，每卡=一次已应用变更）是书斋审批链既有沉淀，把它变成程序性记忆矿床——确定性挖掘跨会话重复 ≥3 次的工具应用模式 → 建议卡只提示不打扰 → 点「结晶为技能」→ LLM 从多次执行回放蒸馏草稿 → 复用 7.2-1 审阅弹窗编辑落盘 → 决定与结晶全程审计。**7.2 技能结晶双入口齐装**（演示式录制 v4.313+历史蒸馏本刀）。
- **线A（主代理代写）**：新纯包 internal/skilldistill（零 IO 表驱动，先例 routesuggest）：MineFlows 会话聚合确定性排序+Distill（连续同 (Tool,Shape) 折叠→会话级序列精确匹配分组→Repeat=不同会话数≥3→步数∈[2,12]→ignored 静默→截 3 宁少勿扰）+BuildPatternID（jd-+sha256 前 8hex 幂等）+ParsePatternID 残渣防御。V1 裁决=滑窗/子序列不进（观察池）。
- **线B**：internal/app/gaea_skill_distill.go 三绑定（OfficeB +3）——Candidates（journal 现算只读零 LLM，ignored∪crystallized 双静默）/Draft（buildJournalReplay ≤3 最近会话分段+AfterSummary 200 rune+整段 600 rune 防护→prompts/skill-from-journal.json（RTCO：多次执行共性进步骤、差异进 cautions、文件名通用化占位）→RetryJSON 2→复用 SkillRecordResult 只回不落盘）/Decide（ignore|crystallized；crystallized 必带技能名，审计条目=sessions+skill+decidedAt）；状态 <DataRoot>/skill_distill_state.json route_suggestions 同款容错读+原子写。
- **线C**：SkillDistillSection（PatternLine 步骤序+重复徽章+证据折叠+结晶/不再提示同卡互斥 loading+空态两分）挂 MemoryPanel 建议 tab「流程蒸馏」分区（props 全可选缺省隐藏，既有调用方零改动）；SkillRecordModal 加 preload/onSaved 可选（preload 直接回填不调内部蒸馏，缺省与 7.2-1 逐字节一致；7.2-1 Composer 入口不动，MemoryPanel 本地托管第二实例零复制）；三语 memory.distill.* 11 键三语键集相等。
- **主代理收口**：gen_bindings+bindingNames 再生 691（bindings_novel 368 行重排噪音甄别还原，v4.313 坑再证）+bridge/core +3（SkillDistillView 局部重述 herdsman 防环先例）+mappings+mock/chat 三桩（走查样本）+spaceBindings work 510→513+App.tsx 接线（useSessionHandlers 六值解构+MemoryPanel 六 props）+建议 tab 首开自动拉候选（C 线 distillView 初值 null 使 undefined 判据失效——主代理改 ref 守卫；fetch 期显示 loading 而非「不可用」误判）+补 SkillRecordModal preload 2 例。
- **绑定面**：688→691（OfficeB +3）；work 锁 510→513；drift OK@691。
- **出口对账**：①只提示不打扰、拒绝即静默 ✅（Decide ignore→状态→复算跳过，测试钉死）；③审计链完整（哪次历史/谁确认/哪版）✅；②结晶技能调用 ≥5 次且成功率可见=**欠账如实入档**（需技能调用计数机制 statsFile 先例另刀）。
- **验收**：skilldistill 15 例+app 10 例+前端 18 例（含收口补 2 例）；tsc -b 零错（双向漂移防线过）、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（复跑：首跑 exit 1 无 FAIL——flaky 先例再+1）；版本四处 4.317.0；产物=exe 50576384B SHA256=e5577402da81afbf0b0135631db066494aa7e415455da67eee2dbe9b0d5e3d2d（releases/gaea-v4.317.0.exe+SUMS；桌面副本同哈希；冒烟 200 过）。
- **观察池**：滑窗/子序列挖掘 V2 等真实 journal 数据分布；journal 回放纳入 verdicts（复核结论）过滤；真机走查（同型任务×3→建议卡→结晶→新会话命中技能一条龙）挂闲置窗口；releases/README V4_RECENT 实存条数与「最近 34 版」头注不符（v4.315 前已存在，非本刀引入）。
- **未做（下刀）**：7.3-1 任务收件箱（收窄版任务空间制首刀）或 7.2-2 判据②调用计数小刀；闲庭在册清欠沿既有列车。

## 最新发布：v4.316.0（2026-09-15）「阶段七第三刀：路由学习（任务级账目+成本归因+建议制改绑，7.1-2）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 进度计划/gaea-route-learning-7-1-2-20260915.md：三线调研结论+契约+足迹互斥表）→ A/B/C 三线并行 → 主代理收口。**坑**=B 线原实现子代理 40 分钟零落盘、两次检查点未应——中断接管主代理代写（子代理卡死判据：长时间零落盘+检查点静默；先例入 AGENTS 执行纪律候选）。
- **判据①任务级账目全键覆盖**：ChatRequest/ChatSimpleOptions 加 Feature 带外字段（json:"-"）→ ai 包 recordUsage/parseStreamEvents/prepareStreamRequest 透传 → statsKey 三维 feature|engine|model（whisper→chat 归一）→ statsFile v2→v3 兼容读 → 24 handler 文件 ~40 调用点一行式透传七键全覆盖（grep 复核零遗漏）；copilot 三入口获批扩迹；bridge 工厂 Provider 打标签（boot→gaea/显式引擎→office）。残留如实报：platform_handler 风格分析（internal/style 足迹外，落未标记桶）、gaea_diagram 两处全局引擎。
- **判据②成本归因视图**：模型中心「成本归因」tab（AttributionSection：KPI×4+功能分组表三维桶行+未标记桶最后；useAttributionState；api/engines.ts +4 函数）。
- **判据③建议制改绑**：新纯包 internal/routesuggest（score=成功率×成本因子域内归一；ScoreGapThreshold 0.25+MinSamples 20 双门槛宁少勿扰；单域至多 1 条；ID 确定性重算幂等+解析残渣防御）；App 三绑定 Suggestions/Apply/Ignore（Apply 走 SetFeatureModel 既有链路绝不自动改绑；状态原子写 <DataRoot>/route_suggestions.json）。
- **判据④本地引擎同池**：候选=全部启用引擎 llm 模型（Kind 回退 ClassifyModelKind）本地云端同池；mock 契约内置 herdsman 建议卡样本（走查锚点；真机采纳样本挂闲置窗口）。
- **绑定面**：684→688（ModelB +4，gen_bindings 覆盖表→model 门面）；work 锁 506→510（shared）；bridge/model.ts 局部重述类型+mock 四桩+新契约测试 mock-contract-route。
- **验收**：routesuggest 11 例+stats 3 例+ai 3 例+app 5 例+前端 9 例+锁数；tsc -b 零错、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（一次过）、drift OK@688、版本四处 4.316.0；产物=exe 50515456B SHA256=82c0e4537d28dfb53314770054672ea094e636aac2645509227d76de88113213（桌面副本同哈希；冒烟 200 过）。
- **观察池**：按任务实例切片留 7.3 联动；bridge 启发式边界（显式引擎 provider=office 域现状 6 处）；route_suggestions.json 无 UI 清单；真机走查（聊天→归因 tab 见桶行+建议卡）。
- **未做（下刀）**：7.2-2 journal 历史蒸馏或 7.3-1 任务收件箱；闲庭在册清欠沿既有列车。

## 最新发布：v4.315.0（2026-09-15）「书斋首页 v9『案头』重设计+外观模块下拉化（零功能删除）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。工作树发现上会话中断的在飞刀（两份规格书随刀入库：进度计划/gaea-desk-home-redesign-v9-202609.md+gaea-appearance-module-redesign-202609.md；实现与工具链验证上会话已完成）——本轮接管收口：独立审查子代理七点对账+CDP 走查取证+全量门禁+发版。
- **v9 诊断**（v8 遗留）：砍刊头后无排印锚点（首屏最大字号 19px）/清单海 ~80% hairline 行+等宽点 10 处/命令条 54px 主角不立/右舷五件纵排无图形锚点。
- **落地**：①masthead 身份区（w-seal 印章 44px 印泥朱 #d06055 唯一 hex 豁免+台名 clamp(30,3vw,42px)+lede+chip/pill 迁入；与闲庭月洞门身份对仗）②kicker 行退役+命令条 54→60px③w-index→w-plaques 双列案牌④w-rail-panel→w-vitals（写作升首节、ml-ring 72→80px 作用域隔离）⑤外观面板 ChoiceCards/ThemeCard 退役→Select 泛型下拉（hover 预览「预览中」徽章）+SegmentedRow 三面板⑥WhisperTracePanel hex→--md-sys-* 令牌（亮色可读）。
- **契约**：五钩子 testid 全保；三语键集 1479 零增删（1 键文案随交互更新）；闲庭逐行不动；零绑定面 drift OK@684。
- **验收**：CDP 9333 DOM 断言书斋 17 项+闲庭 5 项全绿+亮色档令牌实测（surface=#f0fdf9）+三截图留档（.tmp/v9-desk-1440.png 等）；ModuleLauncher 8/8+AppearancePanel 6/6+审查对账（className 双向孤儿=0）。
- **门禁**：ci.ps1 全绿 exit 0（第三跑——首跑 netclient flaky 定向复跑绿、二跑 internal/app TempDir 清理竞争，环境 flaky 先例再+2；三跑清走 Vite/无头 Edge 负载后全绿）、版本四处 4.315.0；产物=exe 50464768B SHA256=356b7c6e8ac9ed4ecfa4f0fb5b1f9fa6c52f61b1f1dcdcc1bd13b620e3bc2a1d（桌面副本同哈希；冒烟 200 过）。
- **文档**：home.md 补 v9 节+releases README V4_RECENT 补齐 v4.308~v4.315 九版滞后+规格书入库+AGENTS 三十迁。
- **观察池**：releases 实存 33 版 exe 与「留 5 版」拍板不符（不动文件待用户定夺）；v3-rise 入场序号两对同拍；pnpm 污染两枚已清+frontend/NUL 坏文件已删。
- **未做（下刀）**：7.1-2 路由学习（三线调研已成：R1 遥测=RecordCall 全量收口+ChatRequest 缺 feature 字段；R2 绑定=featureModelKeys 8 键+SetFeatureModel 链路+routesuggest 纯包落点+route_suggestions.json 存储；R3 前端=模型中心「成本归因」tab 落点推荐+命名避让 GaeaCostAttribution。契约待主代理定稿→后端账目/接线+建议生成+前端三线并行）。

## 最新发布：v4.314.0（2026-09-15）「书斋首页 v8 重设计『驾驶舱台面』（技能库驱动，零功能删除）」

- **来源**：用户指令「使用技能重新设计书斋首页」——ui-ux-pro-max 技能库三路检索定方向（Swiss 数据密集+星枢令牌 MASTER 权威）+ Edge 无头 CDP 截图 + analyze_image 诊断。
- **诊断**：v7 刊头大标题占 ~10% 视高/左栏长卷目录挤出首屏/账页×目录同构列表纵向堆叠/右栏五节等权无主次。
- **落地**：①刊头压缩为 kicker 行（w-mast→w-deck）②命令台升格第一主角件（玻璃抬升面收命令条/气泡流/语音状态）③账页×目录双列 7fr/5fr+旗舰横带压缩+目录单列④右栏主从：晨报置顶⑤间距校准。
- **契约不变**：testid 全保留/i18n 零键增删/功能零删除；既有测试 9/9 零改动。
- **门禁**：ci.ps1 全绿 exit 0、drift OK@684、版本四处 4.314.0；产物见 `releases/SHA256SUMS-v4.314.0.txt`。
- **观察池**：晨报有数据时右栏高度回看；写作环空态密度优化候选。
- **未做（下刀）**：7.1-2 路由学习或 7.2-2 journal 历史蒸馏。

## 最新发布：v4.313.0（2026-09-15）「阶段七第二刀：会话录制技能（演示式录制，7.2-1）」

- **来源**：阶段七 7.2-1（平台调研验证形态：Anthropic Record a Skill / OpenAI Record & Replay 均已发布；SKILL.md 成事实标准）。已有单轮沉淀+记忆聚类候选两通道，缺「会话级多轮回放→结构化草稿→审阅入库」。
- **落地**：①skill-from-session.json 模板（蒸馏纪律+JSON 草稿契约）②office_skill_record.go（buildSkillReplay 末段 60 条回放/parseSkillDraft 归一断拒/renderSkillDraftBody）③GaeaSkillDraftFromSession（本地优先路由+RetryJSON，只回不落盘）+GaeaSkillDraftSave（审阅后落盘）④saveSkillFile 共享通道重构（单轮/会话录制同一落点）⑤SkillRecordModal+Composer 工具栏入口；绑定面 682→684（work 锁 506）。
- **纪律**：LLM 只产草稿，落盘必经用户审阅。
- **测试**：Go 6 例+vitest 4 例；既有零改动。坑=heredoc 转义破坏 TS 字符串/锁数量忘同步/gen_bindings 噪音再证。
- **门禁**：ci.ps1 全绿 exit 0、drift OK@684、版本四处 4.313.0；产物见 `releases/SHA256SUMS-v4.313.0.txt`。
- **未做（下刀）**：7.1-2 路由学习或 7.2-2 journal 历史蒸馏；闲庭在册清欠沿既有列车。

## 最新发布：v4.312.0（2026-09-15）「阶段七首刀：记忆质量受控评测（三脑四域零依赖基线+经验习得题集，7.1-1）」

- **来源**：阶段七规划（docs/gaea-stage7-plan-2026-09.md，2026-09-15 拍板启动）首刀；闭合在册开放增量「记忆质量可评测」（对齐物升级为 BEAM/LongMemEval-V2）。三脑记忆域检索质量此前零量化——记忆不可信则 7.2 技能结晶复用成功率不可测。
- **落地**：①新包 internal/memoryeval（纯函数+种子语料零外部依赖）：Evaluate（recall@10 门槛 0.8 沿 T5-6 口径+precision 均值）+ParseSet/ResolveSetPath；②三域 runner 全走生产路径：story=纯 Go BM25/work=gaea bm25 Ranker/persona=whisper SQLite FTS 含中文 2-gram LIKE 降级；③经验习得题集 22 题（≥20 达标，LongMemEval-V2「习得经验」方向，查询=未来任务口吻）——7.2 评测基建预置；④种子语料 needle-in-haystack（12+40/12+28/10+20/22），题集与种子 ID 漂移由测试硬断言兜住。
- **测试**：memoryeval 4 例（解析/数学/结构硬断言/基线软告警——低于门槛只 WARN 不阻断）；既有零改动。
- **基线**：四域 recall@10 全 1.000（precision 均值 0.433/0.478/0.661/0.727）落档 docs/memory-eval-set.md；评测产出真实发现=persona 泛主语查询淹没（「用户」二字组 LIKE 命中全部事实）——生产弱点记录在档不动引擎。
- **门禁**：ci.ps1 全绿（vitest 首跑 ContextView 类 flaky 28 例、全量复跑 356 文件/3064 全过）、零绑定面 drift OK@682、版本四处 4.312.0；产物见 `releases/SHA256SUMS-v4.312.0.txt`。
- **未做（下刀）**：7.1-2 路由学习（任务级成本归因+建议制改绑）或 7.2-1 演示式录制（无依赖可先行）；闲庭在册清欠（t4-C3 余项/t6/t7/GenerationGate 收口）沿既有列车。

## 最新发布：v4.311.0（2026-09-15）「t5 收官刀：关系图谱升级（三类节点·四层边·分类筛选·详情浮层，§7.5）」

- **来源**：MuMu 蒸馏线 t5 收官刀（§6.1/§6.2/§7.5）。改造前组织节点是孤岛、职业不可见——状态机数据图谱可视化是收官题眼。
- **落地**：①纯函数模块 graphData.ts（buildGraphData）：节点三类+分组虚拟节点、边四层（组织成员 0.5/分组 0.35/主职 0.3/副职 0.2/人际 0.02——MuMu layoutWeight 映射 d3 strength，不引 dagre 不引新依赖）、人际 pair 去重、无结构边时人际 0.3 兜底、active 过滤+悬空防御；②组件：五类筛选 checkbox/点击详情浮层/职业方形节点/虚线边/选中描边；③零新依赖，调用点零改动。
- **测试**：vitest graphData +4；tsc 全绿；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面、版本四处 4.311.0；产物见 `releases/SHA256SUMS-v4.311.0.txt`。
- **t5 域全清**：差分器→回灌→组织差分→职业 UI→图谱，五刀收官。
- **未做（下刀）**：t4-C3 余项；t6/t7。

## 最新发布：v4.310.0（2026-09-15）「t5 第四刀：职业管理页（角色-职业绑定 UI，§7.6 断线补齐）」

- **来源**：MuMu 蒸馏线 t5 第四刀（§6.4/§7.6/§7.7-#9）。MuMu 职业接口后端完备前端断线；gaea 四刀全通后同样「用户到不了」——本刀补 UI 通路。
- **落地**：①NovelB +SetCharacterCareer/RemoveCharacterCareer（主职业允许替换〔手工意图 vs 差分器防 LLM 误改，两处语义有意不同〕/副职上限 2/同名幂等/阶段钳 ≥1/手工不写水位；存储=项目 characters.json 与差分器同库即设即生成可见）②CharacterPage Drawer 增「职业体系」区块（主/副管理+即时保存+上限禁用+操作后同步快照）+types v2 字段（GetCharacters 早已返回前端未消费）③契约面：bridge/mock/bindingNames 682/spaceBindings 锁 504。
- **测试**：character +1（绑定矩阵）+vitest 3 例（渲染/逐参/上限+Popconfirm 确认）。
- **维护**：AGENTS 速览整段分流——留 3 版迁 10 条入 archive，水位 61671→38151B 根治。
- **门禁**：ci.ps1 全绿、drift OK@682、版本四处 4.310.0；产物见 `releases/SHA256SUMS-v4.310.0.txt`。
- **未做（下刀）**：t5 余项唯一=关系图谱升级 §7.5；t4-C3 余项；t6/t7。

## 最新发布：v4.309.0（2026-09-15）「t5 第三刀：组织状态顶层差分接线（分析链三路输入全通）」

- **来源**：MuMu 蒸馏线 t5 第三刀（§3.7/§3.8/§7.7-#5）。差分器有组织管道但调用侧传 nil、模板无契约、载荷无字段——有管道无水源。
- **落地**：①V2 wire 契约 OrganizationStateChangeV2/OrgMemberChangeV2 名称式引用（与差分器 ID 式输入分层，对齐 character_states 先例）+AnalysisResultV2 +organization_states；②模板 output 加 organization_states JSON 形状+must 稀疏差分纪律（名称精确一致否则省略，不猜测）；③syncCharacterStatesV2 名称→ID 匹配（失败 WARN 跳过）传第 4 参；④**修首刀缺陷：覆灭短路**（destroyed=true 清零势力值+跳过同条 power 与成员变更；destroyed=false 不复活）。
- **范围裁决**：PowerValue 用 0-100 绝对值而非 MuMu 相对增量——幂等可重放，免疫 MuMu §8 坑 2（org 更新无水位守卫）。
- **测试**：analysis +2（端到端：晋升/加入/缺省忠诚/ID 匹配/不存在跳过/覆灭忽略 power/空载荷零写盘）+characterstate +1（覆灭短路纯函数钉死+destroyed=false 不复活）；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.309.0；产物见 `releases/SHA256SUMS-v4.309.0.txt`。
- **未做（下刀）**：t5 余项（关系图谱升级 §7.5/职业管理页 §7.6）；t4-C3 余项；t6/t7。

## 最新发布：v4.308.0（2026-09-15）「t5 第二刀：状态回灌（SceneBible 实体属性 + 章节生成 character_states 槽位，降噪）」

- **来源**：MuMu 蒸馏线 t5 角色状态机域第二刀（§3.9/§7.4）。v4.307 差分器写盘后生成侧看不见——状态机一半价值在写回生成侧。MuMu 两通道格式不一致（历史债），gaea 统一多行式。
- **落地**：①SceneBible 实体属性回灌（零新增管道）：entityFromCharacter +current_state/career_main/career_sub，sceneCharFromEntity 读回+角色文件兜底，formatSceneChar 渲染（存量老角色零噪声）；②create-chapter 模板 +character_states P1 槽位，注入器 buildCharacterStatesSection 多行式（存活 emoji/心理截150/职业「剑修·3阶」/组织+忠诚度/关系+亲密度），接线 CreateChapter 主链；③降噪（照搬 MuMu :794/821）：intimacy==50 与 loyalty==50（0 视为未记录）不注入、past 关系跳过、非 active 成员不显示、关系前5/组织前3、区段预算 1500 rune、存量项目全员无 v2 字段返回空串零噪声；④types.CareerMainLabel/CareerSubLabels（无最大阶——方案 C 死解析不做已裁决；CareerName 快照优先）。
- **范围裁决**：chapter-generate.json 是 embedded 死资产（无消费方），槽位打真实链路 create-chapter，死资产不养死槽位。
- **测试**：types +1 + novelcontext +3（roundtrip/兜底/零噪声）+ app +4（降噪矩阵/模板联通）；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.308.0（含 versioninfo.rc，sync-version.ps1 同步）；产物见 `releases/SHA256SUMS-v4.308.0.txt`。
- **未做（下刀）**：t5 余项（组织顶层差分 UI/关系图谱升级 §7.5/职业管理页 §7.6）；t4-C3 余项；t6/t7。

## 最新发布：v4.307.0（2026-09-15）「t5 首刀：角色状态机差分更新器（水位守卫+存活级联+关系差分）」

- **来源**：MuMu 蒸馏线 t5 角色状态机域首刀（§3.3~§3.6/§7.2/§7.3）。契约 v4.278 已落，差分器实现是缺口。
- **落地**：①internal/characterstate 差分更新器（三阶段严格顺序+七条不变量全实现：双水位单调/存活短路/**存活级联三件套**/职业钳制/MaxSubCareers=2/副职业水位守卫）；②亲密度算法最长匹配优先禁子串累加（修「不信任」-5 失真）+LLM DeltaHint 优先；③关系差分双向无向/新建基线 50/时间线追加；④模板扩 survival_status+relationship_changes+稀疏差分纪律；⑤syncCharacterStatesV2 委托差分器。
- **测试**：characterstate +5（级联全链/守卫/关系双向/词典/职业组织矩阵）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本三处 4.307.0；产物见 `releases/SHA256SUMS-v4.307.0.txt`。
- **未做（下刀）**：t5 余项（回灌 SceneBible/chapter-generate 槽位/组织差分 UI/图谱升级）；t4-C3 余项；t6/t7。

## 最新发布：v4.306.0（2026-09-15）「t4-C4 分析标注层：keyword→正文偏移（坐标反投影）+ 读取绑定」

- **来源**：MuMu 蒸馏线 t4 情节分析域第三刀（§3.10/§8.4）。C1 keyword 锚点坐标从未回投正文，本刀补标注层数据源。
- **落地**：①定位四段式（精确/去标点反投影 indexMap 修 MuMu 坐标缺陷/前缀 15/未命中）；②构建规则（锚定失败丢弃不留 -1；hook Content 兜底锚点；suggestion 留 -1 仅列条目；伏笔带 tags）；③落盘 analysis/annotations/<MMM>.json+Analyze 接线；④NovelChapterAnnotations 绑定（缺档有分析按需重建）。
- **测试**：analysis +3（定位四段式/构建矩阵/空载荷）。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@680（锁 501→502）、版本三处 4.306.0；产物见 `releases/SHA256SUMS-v4.306.0.txt`。
- **未做（下刀）**：前端内联高亮随 t7；t4-C3 余项；t5~t7 未开工。

## 最新发布：v4.305.0（2026-09-15）「t4-C3 收尾：整章重写前端消费（对比确认 UI）+ 建议读取绑定」

- **来源**：t4-C3 收尾刀——v4.304 后端六绑定零前端消费。
- **落地**：①NovelChapterSuggestions（678→679，读 analysis-v2 建议，无分析空数组正常态）；②RewriteModal 三态（表单：建议勾选/自定义要求/重点方向/保留元素/字数，source 自动判；结果：统计+新全文+应用/放弃；应用后恢复入口）；③CreatePage rail 按钮+onApplied 刷新编辑器；④mock +7 诚实空态。
- **测试**：vitest RewriteModal 5 例；既有零改动。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@679（锁 500→501）、版本三处 4.305.0；产物见 `releases/SHA256SUMS-v4.305.0.txt`。
- **未做（下刀）**：partial 局部重写/场景工程/版本历史面板；t4-C4 锚点标注。

## 最新发布：v4.304.0（2026-09-15）「t4-C3 首刀：驱动式整章重写 + 版本库（应用/丢弃/恢复）」

- **来源**：MuMu 蒸馏线 t4 情节分析域第二刀（§5.2/§8.3 C3）。C1 建议产出后补重写消费链；gaea 强制增量=版本库自带恢复（MuMu 无 restore 路由）。
- **落地**：①internal/rewrite 引擎（指令四段/输出清理/diff 统计/请求归一，纯规则）；②版本库（索引无全文+按需读，契约路径）；③六绑定：重写（建议来自 analysis-v2，温度 0.7，不自动落章）/列表/读全文/应用/丢弃/恢复（原文快照写回+审计）。
- **测试**：rewrite +4 / project +1 / app +4（桩 LLM e2e 全链含恢复审计）。
- **门禁**：ci.ps1 全绿、绑定面 +6 drift OK@678（spaceBindings 锁 494→500）、版本三处 4.304.0；产物见 `releases/SHA256SUMS-v4.304.0.txt`。
- **未做（下刀）**：partial 局部重写/场景工程/前端对比 UI；t4-C4 锚点标注。

## 最新发布：v4.303.0（2026-09-15）「t3-P2 记忆召回消费：语义检索进 SceneBible 记忆区段（长程一致性全闭环）」

- **来源**：MuMu 蒸馏线 t3 长程一致性域第三刀（§3.3/§3.4/§12.1/§12.4）。分析自动回填（P1）+语义召回注入（P2）全通；检索复用 semantic.Store+本地 embedding，不引 ChromaDB。
- **落地**：①结构化 query（人物[:8]/关键事件[:6]/叙事目标/情绪/本章要求[:150]，整体[:800]）；②四段流水线（阈值 0.35 归一化余弦/兜底 3/topK 8，三常量唯一声明）；③SceneBible.Memories+「相关记忆（按相关度）」区段（预算 400）；④Ensure 增量+Stale 清死向量+kind=项目目录隔离+当前章排除；⑤CreateChapter 全链容错接线。
- **测试**：app +4 / novelcontext +2；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.303.0；产物见 `releases/SHA256SUMS-v4.303.0.txt`。
- **未做（观察池）**：t3 三刀收官；真机检索质量评估；候选 2 记忆评测开放；t4-C3/C4、t5~t7 未开工。

## 最新发布：v4.302.0（2026-09-15）「t3-P1 记忆生产者：规则表抽取 StoryMemory 按章落盘（自动回填闭环）」

- **来源**：MuMu 蒸馏线 t3 长程一致性域第二刀（§2.3/§12.1/§12.2/§12.4-4）。t4-C1 解锁 V2 载荷，本刀补「分析→结构化记忆」抽取——自动回填闭环咬合（市场差异化点，调研档 §9）。
- **落地**：①types.StoryMemory+确定性 ID `<MMM>-<type>-<ordinal>`+五类常量+is_foreshadow 三值（无 Position 死字段 D11）；②提取规则表纯函数（chapter_summary 必有三级回退固定 0.6/hook≥6/foreshadow 全部/plot_point≥0.6/character_event 0.7/conflict≥7→plot_point，门槛常量唯一声明）；③memories/MMM-<n>-memory.json 按章整文件替换写（幂等规避 MuMu D1，空=删文件）；④Analyze 接线 persistStoryMemories（容错）。
- **测试**：analysis +3（规则表/确定性 ID/回退链）+project +1（回环/空写删档）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.302.0；产物见 `releases/SHA256SUMS-v4.302.0.txt`。
- **未做（下刀）**：t3-P2 召回消费（semantic kind=story_memory+SceneBible 记忆区段+三常量+结构化 query）——接上即全闭环；t4-C3/C4。

## 最新发布：v4.301.0（2026-09-14）「t4-C1 分析代理 V2 化：9 维结构化 + 三维评分联动 + analysis-v2.json 落盘」

- **来源**：MuMu 蒸馏线 t4 情节分析域首刀（§8.3 C1）。市场调研增量轮（调研档 §9）抬升优先级：自动设定库回填是行业痛点（Novelcrafter 手填摩擦/Sudowrite 书长极限），V2 载荷是回填数据底座。
- **落地**：①模板 V2 九维化原地升级（伏笔追踪约束原样保留，ForeshadowHit 契约零改动）；②服务端权威归一（三维钳位/overall 重算/建议联动只裁上限）；③analysis-v2.json 按章号 upsert 落盘（容错注入）；④旧 wire 零破坏派生（AnalyzeChapter 返回键零变化）；syncCharacterStates 升 V2 差分。
- **测试**：analysis +4 + project 落盘回环；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.301.0；产物见 `releases/SHA256SUMS-v4.301.0.txt`。
- **未做（下刀）**：t3-P1 记忆生产者（已解锁，下一刀闭环自动回填）；t4-C3 驱动式重写；t4-C4 锚点标注。

## 最新发布：v4.300.0（2026-09-14）「t3 首刀：章节前文摘要窗口预算化 + 回退链 + 反重复约束」

- **来源**：MuMu 蒸馏线 t3 长程一致性域首刀。§11.3 缺口：prevSummary 全前章 200 rune 无界拼接（200 章≈40k rune 前缀），不参与预算体系。
- **落地**：①最近 10 章窗口（buildPrevSummaryWindow 纯函数，单章 180 rune，章号升序确定性，空返回跳槽位）；②窗口声明头（部分视图语义显式化）；③摘要回退链：大纲 Summary→章节摘要文件→跳过；④create-chapter 模板反重复 must（衔接不复述+窗口外既定剧情不得矛盾）。
- **测试**：app +4（窗口/截断/回退链/乱序空态钳位）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.300.0；产物见 `releases/SHA256SUMS-v4.300.0.txt`。
- **未做（t3 下刀）**：P1 记忆生产者（StoryMemory+规则表+落盘，依赖 t4 产出 AnalysisResultV2）；P2 语义召回消费（semantic kind=story_memory+记忆区段+三常量）。

## 最新发布：v4.299.0（2026-09-14）「伏笔面板接调度可视面：清理入口/统计/紧急度 Badge/同步结果（t1-P4）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第四刀（收官刀）。P1~P3 全在后端，前端面板仍是旧三态视图。
- **落地**：①SyncResult 上绑定面（v4.297 欠账）：GetLastForeshadowSync（面 671→672），同步计数+skippedReasons 全量（D3 可见），未分析显式报错；②GetForeshadows 附带 urgency 运行时投影（共享入口算，A5 不落库，回收/废弃不带键）；③面板四件：后端统计行（分状态+超期红 Tag，降级前端计数）/紧急度 Badge（红橙金+tooltip）/清理折叠区（三入口全确认+自动重载）/上次同步常显卡（计数+前 2 条跳过原因）；④A5 护栏=写回 stripForeshadowUrgency 剥离；⑤契约面+mock+5+load() 增量绑定降级。
- **测试**：Go +2（投影矩阵/lastSync；SyncForeshadows 按规格签名转正导出）+ vitest 面板 14/14（+6）。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@672（spaceBindings 锁 493→494）、版本三处 4.299.0；产物见 `releases/SHA256SUMS-v4.299.0.txt`。
- **未做（观察池）**：真机走查闭环一条龙（等闲置窗口）；SyncResult 持久化不做（会话内诊断面）。

## 最新发布：v4.298.0（2026-09-14）「伏笔生命周期清理与统计 + Lint 扩两码（t1-P3）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第三刀（spec §8.1/§5.1/P3.4）。P2 打通分析自动回收后，重分析/重新生成场景缺「干净重来」入口。
- **落地**：①三个清理入口（新 `internal/app/foreshadow_cleanup_handler.go`，语义严格区分）：DeleteChapterForeshadows（删埋入∨回收本章，默认只删分析来源）/CleanChapterAnalysisForeshadows（重分析前：删「分析∧埋入本章」+回退本章回收→planted 清回收痕迹）/ClearProjectForeshadowsForReset（删全部来源；手动**重置不删除**→pending 清章节关联时间戳）；关键不变量=source_type 唯一判据、手动条目永不批量删除；②统计 GetForeshadowStats：分状态+longTermCount+overdueCount（ClassifyResolve 唯一入口，currentChapter≤0 自动），resolved 别名归一并入；③Lint 扩两码 5→7：overdue（medium）+unplanned（low，长线豁免，与 stale「该收了」vs「缺计划字段」并存）。
- **测试**：app +6（三入口 e2e/统计口径/Lint 新码矩阵/camelCase 形状锁）。
- **门禁**：ci.ps1 全绿、绑定面 +4 drift OK@671（spaceBindings 锁 489→493）、版本三处 4.298.0；产物见 `releases/SHA256SUMS-v4.298.0.txt`。
- **未做（下刀）**：t1-P4 前端（面板接清理/统计/新体检码+Badge，mock 随消费面补）；真机走查（分析→回收→清理闭环，等闲置窗口）。

## 最新发布：v4.297.0（2026-09-14）「伏笔分析驱动自动回收：三级匹配闭环 + 候选清单三层渲染（t1-P2）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第二刀（spec P2 闭环核心）。此前分析侧回收只做「描述精确相等」，候选清单是整包 JSON 直塞 prompt——模型记不住该回填哪个 ID。本刀按 `docs/distill/01-foreshadow-spec.md` §3.3~§3.5/§6.1 把分析侧接上 v2 契约，埋入→回收闭环咬合，零绑定面。
- **落地**：①纯函数层（新 `internal/types/foreshadow_match.go`）：WordOverlap（rune 级 2/3-gram Jaccard 加权）+ MatchForeshadowByContent 六策略加权（标题族取最大/关键词/内容相似/引用章/分类/角色 Jaccard 累加，采纳≥0.5 同分取先=最早埋入）；②同步层重写（新 `internal/analysis/foreshadow_sync.go`，消费 `types.ForeshadowHit` 替代旧 ForeshadowAction）：回收三级匹配=精确 ID（查不到禁止回落内容匹配）→内容兜底→跳过不新建；已回收不重置、pending 拒收、hinted/partial 可回收（D15）；埋入两道防重+每章新建≤5+Importance=min(Strength/10,1) 唯一派生+计划章缺失不猜值；SyncResult 完整含 SkippedReasons 六类 Kind（修 MuMu D3 静默跳过）；③候选清单三层渲染（新 `internal/analysis/foreshadow_prompt.go`）：L1 必须回收逐条带「⚠️ 回收时 reference_stable_id 填写」紧邻指令/L2 超期≤5/L3 其他≤10+溢出注记，分层走 ClassifyResolve 唯一入口（D9）；④`prompts/analysis-chapter.json`：foreshadows 换 v2 字段+候选槽位升 P1+伏笔追踪任务指令+三条强约束+ID 追踪自检（spec §6.1）；写入口径仍 revealed，前端零消费零破坏。
- **测试**：types +4 / analysis 同步 15 例（4 迁移+11 新增，含无效引用不回落关键回归）/ 渲染 4 例（三层/排除口径/折叠上限/空态 D14 截断）。
- **门禁**：ci.ps1 全绿、drift OK@667（零绑定面）、版本三处 4.297.0；产物见 `releases/SHA256SUMS-v4.297.0.txt`。
- **未做（下刀）**：t1-P3 清理入口/Lint 扩展 overdue·unplanned/统计口径；t1-P4 前端（表格/Badge 用后端 urgency，SyncResult 上绑定面随 P4）；真机走查（分析闭环+书源线一条龙，等闲置窗口）。

## 最新发布：v4.293.0（2026-09-14）「伏笔分层注入：按计划回收章调度生成上下文（t1-P1）」

- **来源**：MuMuAINovel 蒸馏线 t1 伏笔域消费方第一刀。契约（6 态并集+计划回收章+注入控制）v4.278.0 已落库但零生成侧消费方；本刀按 `docs/distill/01-foreshadow-spec.md` §4 落分层注入（spec P1），零绑定面。
- **落地**：①纯函数层（新 `internal/types/foreshadow_urgency.go`）：UrgencyLevel（0-3 运行时不落库；D15 修正 partial/hinted 有回收压力；缺计划章不猜值不假超期）+ClassifyResolve 四值+ForeshadowLayerOf 分层唯一入口（L1 必须回收/L2 超期/L3 近期参考/L4 本章计划埋入/L5 无计划兜底〔gaea 扩展：存量无 target_resolve_in 条目保底注入防回归〕）+调度常量唯一声明；②include_in_context=false 全层排除、auto_remind=false 只抑制 L3（D8：L1/L2 硬约束不消隐）、远期不注入；③四层渲染（新 `internal/app/foreshadow_context.go`）替代旧「未回收伏笔（创作约束）」单层：模板对齐 spec §4.2（D14 省略号按长度判断）、L2≤3/L3≤5/总量≤15/整区含标题≤1600 rune、消耗顺序 L1→L2→L3→L4→L5、空层不输出标题、区段头带调度规则约束行；④`resolveTargetChapterNum`（显式/分支父节点/顺延，与 ensureChapterNode 同源复用）；⑤novelcontext 场景圣经分层优先采样（硬约束层带层标注先于参考层；分层逻辑全仓唯 types 一处=D9 消除）；⑥删 docs/mumu-distill 重复副本。
- **测试**：types +4 / app +7 / novelcontext +1（抓出超期章数负号 bug）/ 既有 context 测试迁移分层口径。
- **门禁**：ci.ps1 全绿、drift OK@660（零绑定面）、版本三处 4.293.0；产物见 `releases/SHA256SUMS-v4.293.0.txt`。
- **未做（下刀）**：t1-P2 分析驱动自动回收（MatchByContent 六策略+SyncForeshadows 三级匹配+分析 Prompt 候选清单）；t1-P3 清理/Lint 扩展；t1-P4 前端。

## v4.283.0 ~ v4.292.0 段落补登（2026-09-14；逐版全文在 CHANGELOG/releases，速览补账）

- **v4.292.0** tail×反推串联：tail 导入成功→确认→novel:goto-tab+novel:auto-reconstruct 两事件→CreatePage 任务化反推链自动开跑。
- **v4.291.0** 反推任务化：tasks.KindOutlineReconstruct+Start/TaskGet（NovelB 660），长书后台态页面关闭不丢。
- **v4.290.0** 角色名→角色库 ID 匹配：matchCharacterIDs 合并进反推 Apply（不冲手工）。
- **v4.289.0** oh-story T6 书级文风档案 style.md 注入（T1~T6 收官）；**v4.288.0** T4 篇幅路由三档骨架（bookimport/skeleton.go）。
- **v4.287.0** tail 出口 ImportNovelBookEx；**v4.286.0** oh-story T2 内核消费（patterns.json+书级白名单）。
- **v4.285.0** 搜索历史+引擎规则编辑器（NovelB 657）；**v4.284.0** 失败章重试补下（NovelB 655）；**v4.283.0** 在线搜书入口；v4.283.1 nil-Fetcher panic 根修。
- 非版本刀：真机走查（假站 /unfail 配方）、flaky 治理（ProgrammingPage 15s/ContextView 40s）、rename 根修（fileutil.RenameWithRetry）。

## 最新发布：v4.282.0（2026-09-13）「oh-story 蒸馏首刀接线：平台质量评审（rubric 引擎 + 创作间面板）+ 生成门确定性两路直显」

- **来源**：用户「把 oh-story-claudecode（MIT 网文写作插件）蒸馏给小说板块」——把已在库的三份资产（T1 评审 rubric / T2 去 AI 味门禁资产 / T3 生成门资产）**接成用户可用的一条链**（规格 `docs/gaea-novel-ohstory-distill-2026-09.md`）。
- **① 平台质量评审引擎**（新 `internal/novelreview`，零 LLM/零网络/纯函数）：15 个可机械判定维度（字数区间 / 开篇钩子 / 章尾钩子 / 预告式收尾 / 情绪节点密度 / 爽点密度 / 段落节奏 / 对话占比 / 标点节奏 / 破折号 / 格式合规 / 人称视角 / 主角存在感 / 金手指提及 / 字数表述核对），每维 `PASS|WARN|FAIL|SKIP` + S1~S4 + **原文证据（rune 区间 + 段落号）** + 改法；结论 `APPROVE|CONCERNS|REJECT` 与 rubric 同门槛；**语义维度不下结论**，缺外部数据显式 SKIP 并说明原因，**不静默给 PASS**。阈值/词表/档位是数据资产 `internal/novelreview/rubric.json`（四档：通用/番茄/起点/知乎盐言；可被 `<工作区>/.gaea/skills/novel-review/rubric.json` 整体替换；fail-closed 校验）。
- **② 接线**：NovelB +2（`NovelChapterReview`/`NovelReviewPlatforms`，648→650，drift OK）；创作间 rail「平台评审」→ 面板（档位选择 + 结论 + 逐维按 S1→S4 排序 + 证据摘录 + 黄金三问）。主角名取自角色库 `role_type=protagonist`。
- **③ T3 两路直显**：`NovelInspector`「章节体检」区新增「写前契约：N 项待补 / 齐备」与「写后硬信号：N 项 / 未命中」+ 严重度排序条目（与 AI 四路并列；缺省诚实降级）。
- **④ 口径对齐**：`.gaea/skills/novel-review/rubrics/generic.json` 18 维 → **26 维**（+8 引擎扩展，标 `measurable/engine`）+ `deterministicEngine` 段；SKILL.md 补「确定性引擎」一节（15 维映射 / 覆盖路径 / `plot_loop` 代理判定口径）——**资产=评审协议、引擎=可机械判定子集，共用同一套维度 id 与 S1~S4**。
- **测试**：Go +16（`internal/novelreview` 12 / `internal/app` 4）+vitest +10（面板 6 / 检查器 3 / CreatePage 1）+spaceBindings 锁 470→472。
- **门禁**：ci.ps1 全绿、drift OK@650、版本三处 4.282.0；产物=exe 49,597,440B SHA256=83F5BFD603DB5EC1B091E006F8CACB85B37C3283FF653CCED669F44A9628EAD9（releases 归档 + 桌面副本同哈希 + 冒烟 200）。
- **未做（下刀）**：T2 内核消费（`gates.json` 模式级门禁进 novelstyle 打分/去味）/ T4 导入结构映射与篇幅路由 / T5 子代理角色卡资产 / T6 风格档案协议；写前契约「拒绝生成」硬闸与书级白名单。

## 非版本刀（2026-09-13）：小说·生成门补确定性两路——写前大纲契约 + 写后质量体检（oh-story 蒸馏 T3）

- **来源**：oh-story-claudecode（MIT）蒸馏第三刀（规格 docs/gaea-novel-ohstory-distill-2026-09.md §2 真增量第 1 条：gaea 缺「写前结构契约闸 + 写后确定性质量闸」）。
- **落地**：①新包 `internal/novelgate`（零 LLM 纯函数）——`OutlineContractIssues`（标题/计划/要点/情感四项目齐备性，S2~S4）+ `ChapterQualityIssues`（空正文 S1 / 超长段落 S3 带行号 / 电报体 S2〔短句占比 >40% 且均句长 <10 字〕/ 平均句长偏短 S3 / 标点堆砌 S3 / 省略号滥用 S3 / 通篇句号化 S3）；Issue 与评审 rubric S1-S4 同轴、必带证据；阈值常量集中、rune 口径。②接线 `RunChapterGate` 新增确定性两路 `outlineContract` + `deterministic`，与 AI 四路并列（必出、零成本、不阻断）。
- **测试**：Go +7（契约齐备与分级/空正文 S1/正常文本零误报/电报体/超长段落/标点与省略号/句号化/证据随行）。
- **门禁**：本刀自身证据（go build/vet exit 0 + novelgate 7 例 + 生成门定向用例绿）；全量 ci.ps1 未取全绿——并行线在制品 internal/app/novel_review_handler_test.go:117 当前为红（与本刀无关）；drift OK@648；**未抬版本**（下次发版随 exe 出）。
- **未做（下刀）**：T4 导入结构映射与篇幅路由；写前契约硬闸（需先有「一键补大纲」）；书级白名单与 T2 `gates.json` 的内核消费。

## 最新发布：v4.281.0（2026-09-13）「拆书导入 P1 接线：AI 反推大纲（预览载荷 + 幂等落库 + 创作间入口）」

- **来源**：续 v4.280.0（反推引擎），把 t2 P1 接成用户可用的一条链（规格 §8.2/§8.3 首刀）。
- **后端**（新 `internal/app/novel_import_ai.go`，绑定 646→648）：`NovelOutlineReconstruct` 读当前工程章节（标题取大纲节点、正文 `ReadChapterAsStitch`，上限 200 章）→ Stage1 立项反推（模板 book-import-project，采样 3 章 ×2000 rune）→ 分批章节大纲（batchSize=5，逐批独立降级）→ 预览载荷零落库；`NovelOutlineReconstructApply` 按**章号**命中节点写 summary/scene_ideas/key_points/emotion，**幂等**、不新建不删除、不碰正文、角色名只进预览；空载荷/全不匹配**显式报错**。
- **降级诚实**：无模型或解析失败逐级回落规则兜底并把原因写进 warnings（aiUsed=false），单批失败只影响该批。
- **前端**：CreatePage rail「AI 反推大纲」→ 反推 → 确认弹窗（题材/视角/目标字数 + 前两条告警 + 「不覆盖正文、可重复执行」）→ 应用 → 刷新大纲 + rail 消息。
- **接线**：NovelB +2、bindingNames 648、spaceBindings 归 play（锁 468→470）、wailsjs 由 `wails generate module` 再生。
- **测试**：Go +2（无模型全规则兜底且逐章完整 / 幂等与作用域 + 空载荷与全不匹配报错）+vitest spaceBindings 4 例 + CreatePage 9 例。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@648、版本三处 4.281.0；产物=exe 49,455,616B SHA256=7FB09E36…2A693（releases/gaea-v4.281.0.exe + SHA256SUMS-v4.281.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：导入向导的逐章预览 UI；长书后台化任务态（`tasks` 表 `KindBookImport`）；角色名→角色库 ID 匹配；tail 模式出口。

## 最新发布：v4.280.0（2026-09-13）「拆书导入 P1 引擎：大纲反推编排 + Prompt 模板移植 + 输入节稳定序」

- **来源**：续 v4.279.0（t2 P0），本刀落 t2 P1「AI 反推」**引擎层**（规格 `docs/distill/02-book-import.md` §8.2/§3.7/§3.8）；真模型调用编排与预览 UI 下刀接线（本刀零 LLM 依赖、全部可单测）。
- **落地**：①反推编排引擎（`internal/bookimport/reconstruct.go`）——具名契约 + 逐字段归一化（rune 截断 / characters type 二元 / **title 与 chapter_number 强制用输入值**）+ **位置对齐**批量归一化（AI 少返/乱序不错章）+ 数量不符**整批回退规则结构**的断言式防线 + 规则兜底逐字对齐规格 + 视角 11 别名归一 + target_words <1000 回落/>3e6 夹取；②`CallJSON` 负反馈重试（期望类型显式 object|array、失败原文截 200 字注入、传输错误不重试、ctx 取消零调用）+ `ExtractJSON` 容忍 markdown 包裹；③新增两个 RTCO 模板 `prompts/book-import-project.json` / `book-import-outline.json`；④**补掉规格点名的引擎缺陷**：`BuildUserPrompt` 原按 Go map 遍历输入节（同模板两次渲染字节不同、长文本无法稳定置尾、前缀缓存无法命中）→ `InputDef` 增 `order`，按 Order 升序 → key 字典序稳定排序，长文本固定置尾。
- **测试**：`internal/bookimport` +9 例（视角矩阵/字段回落与夹取/位置对齐/截断与 type 归一/兜底取首句/重试提示要素/类型不符报错/传输错误与 ctx 取消/JSON 容错与采样）+`internal/prompt` +2 例（**同模板 20 次渲染字节一致**/order 覆盖 key 序）。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646（零新绑定）、版本三处 4.280.0；产物=exe 49,405,440B SHA256=0F9BF89A…8EF4B（releases/gaea-v4.280.0.exe + SHA256SUMS-v4.280.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：app 绑定 `NovelImportReconstruct*`（真模型编排 + 预览载荷）/ 导入向导 UI（预览→应用，含 tail 模式出口）/ 幂等落库与三件套（§8.3，`tasks` 表 `KindBookImport`）。

## 最新发布：v4.279.0（2026-09-13）「拆书导入 P0：三级分章 + 5 级编码链（MuMu 蒸馏 t2 首刀）」

- **来源**：续 v4.278.0（t1 共享契约），本刀落 t2「拆书/导入反推」**P0 阶段**（规格 `docs/distill/02-book-import.md` §8.1，纯规则零 AI、可独立验收）。
- **改前短板**：导入只有强标题正则，识别不到即退化成单章「全文」；编码链缺 utf-8-sig 与 Big5，且 GB18030 解码几乎不报错 + `utf8.Valid` 二次校验**静默放过误判**。
- **落地**：①新包 `internal/bookimport`（纯规则零 IO）——`Decode` 五级链 + UTF-16 BOM + **候选结果含替换符即判误判续探**（实测 GB18030/GBK 解 Big5 不报错只出乱码）；`Clean` 六步顺序敏感清洗；`Split` 三级切分（强标题存在不叠加弱标题、弱标题需 ≥2 候选、无标题 >5000 字走 3000~5000 窗口取最靠后句读边界）；`tail` 裁剪（5 倍数取整、>50 降 full）；四类告警。②app 接线（委托 + `NovelImportResult` 增 `encoding`/`split_strategy`/`warnings`）。③前端成功文案直显「编码 · 切分策略」+ 告警弹窗。
- **有意偏离 MuMu**：首标题前 <200 字并入首章（不丢书名/作者）；阈值一律 rune 计数（MuMu 用字节）。
- **测试**：`internal/bookimport` 17 例（Big5 不被 GBK 遮蔽回归/BOM/UTF-16/兜底不 panic/清洗顺序/三级切分/窗口上界/三类告警/tail 取整与降级）+`internal/app` 导入 5 例（BOM/GB18030/无标题窗口/弱标题/e2e 报告字段）。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.279.0；产物=exe 49,397,760B SHA256=D8947306…7659E（releases/gaea-v4.279.0.exe + SHA256SUMS-v4.279.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：AI 反推编排（字段级归一化 + JSON 负反馈重试）/幂等落库与任务态（`tasks` 表 `KindBookImport`，禁用内存 dict）/导入向导 UI（tail 引擎已就绪，当前仅 full 可达）。

## 最新发布：v4.278.0（2026-09-13）「小说域 v2 共享契约落库 + 伏笔一致性体检（含 wire 口径纠偏）」

- **来源**：两条线汇合——①「优化完善gaea」并行池小说线续刀=伏笔一致性 Linter（文风指纹 v4.277.0 已落）；②MuMuAINovel 蒸馏实施线（规格 docs/distill/）由外部团队先落 t1 共享契约后停摆，用户确认后由 Codex 侧接管收口。
- **①伏笔一致性体检**：新 `internal/app/novel_foreshadow_lint_handler.go`（纯函数 5 类确定性检查：ordering / status-mismatch〔partial 豁免〕/ dangling / stale≥10 章〔长线豁免〕/ duplicate）+ 报告 totalChapters/items/planted/hinted/revealed/longTerm/findings；前端 ForeshadowPanel「一致性体检」（概要行 + severity Tag 轻/中/重 + 说明 + 原文条目 + 章节引用，Go 侧 omitempty/null 全防御）；绑定 645→646。
- **②t1 共享契约落库**（internal/types/）：伏笔并集 6 态 + 计划回收章 + 来源/评分/关联/注入控制 + urgency 显式 must_resolve（MuMu 静默丢弃字段的教训）+ 引用失效禁止回落内容匹配；角色三态章节水位守卫 + 方案 C 职业引用 + 派生 member_count + 亲密/delta_hint 钳制；9 维分析 + 三维分档评分 + 建议数硬联动；ChapterPlan / RewriteVersion+Index / Annotation；模板解析-覆盖-渲染-校验与上下文组装接口 + Lint 接口 + ChapterNumOf；Character/Organization/Relationship v2 可选字段（全 omitempty 零迁移）+ compat_test 兼容矩阵。
- **③P0 wire 口径纠偏**：在制品曾把「已回收」**写入口径**定为 `resolved`（`revealed` 降读取别名）——与 handoff §2-3/§4-7 相反，会让统计回收率/lint 计数/章节注入/SaveForeshadows 白名单等 7 类消费方**静默读空**；纠回 `revealed` 写入口径 + 新增 `IsResolvedStatus` 覆盖两套 wire 值并接进 stats / 伏笔 lint（检查·标签·计数）/ create_chapter 注入闸 / analysis 聚合；SaveForeshadows 白名单 3 态→并集 6 态（别名先归一化）。
- **④规格入库**：`docs/distill/`（01~07 六域 + 09-impl-handoff）入库并在 docs/README 登记；docs/mumu-distill/ 为重复副本未入库。
- **测试**：Go 全量绿（types 兼容矩阵 + 两套 wire 值新用例 / app 伏笔 Lint 5 类 / stats / analysis）+ vitest 体检 3 例 + spaceBindings 锁 467→468。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.278.0；产物=exe 49,379,328B SHA256=E6E17B1B…C9FE1（releases/gaea-v4.278.0.exe + SHA256SUMS-v4.278.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **欠账**：六域实施 t2~t7 未开工（规格在库、契约就位）；t1 契约除伏笔体检外暂无消费方；**走查未覆盖创作面壳内真机**（mock 无工程夹具）；观察项=ChapterNumOf 双实现 / stats·narrative 无 owner / docs/mumu-distill 重复副本待删 / **novel api 直取 `window.go.app.App`（`components/novel/api/outlines.ts` 等，mock 模式 console 报 Wails runtime unavailable；生产壳不受影响）**。

## 最新发布：v4.277.0（2026-09-13）「小说·文风指纹落地面：参考档构建 + 对照体检」

- **来源**：「优化完善gaea」→ 拍板池三项均用户决策项，按既定习惯从板块并行池挑小说线既定余项（novel-revolution 刀3 内核指纹算子齐备但 app 层零消费）。
- **落地**：参考档 fingerprint.json（Manager 三方法原子写）+ NovelFingerprintBuild（全部已写章节 ComputeFingerprint 落盘，门槛 ≥3 章 ≥3000 字诚实拒绝）+ NovelFingerprintScore（有参考档 ScoreText+Delta 对照 / 无参考档通用阈值，两路可用）+ NovelFingerprintStatus + CreatePage「文风指纹」面板（空态引导/摘要格子/体检大数字+Δ 基线距离+issues 摘录）。绑定 642→645。v1 口径=作者成稿即样本，「只学作者手改」留后续刀。
- **测试**：Go+6+形状锁（camelCase 断言+delta=0 指针零值不吞）+vitest+5+spaceBindings 锁 464→467；tsc/eslint 0；drift OK@645。
- **真机走查**：夹具工程（project.Create 造 3 章约 4000 字）壳内 CDP 全流程全通（空态→构建→体检→重建覆盖）+console 零错误+清场。坑复训=海报墙须 scrollIntoView 再点/无 outline 夹具致体检钮 disabled/delete-pending 须 PowerShell 复删。
- **门禁**：ci.ps1 run1 抓 v4.276 存量缺陷当场修（ChatPage.test 语音断言未随 camelCase 发送对齐，改齐后单文件 15/15）run2 全绿/版本三处 4.277.0/桌面副本同哈希冒烟 200。产物 exe 49,352,704B SHA256=2CB99312…802CD。

## 最新发布：v4.276.0（2026-09-13）「绑定面 JSON 形状普查：造价参考/复盘笔记/会话组价价格带断链修复」

- **普查方法**：把 v4.269 走查抓到的「Go struct 缺 json 标签→Wails 线上 PascalCase→前端 camelCase 读空」bug 类升级为全仓 AST 闭包普查（绑定面签名类型递归展开×缺标签求交）：327 缺标签 struct 中绑定面可达仅 7。
- **三真断链当场修**：cost.PriceBand/BandSource（v4.191 起会话组价价格带卡全空+证据表离群判定失效）/costref.Note（复盘笔记 validUntil/refCount 读空）/costref.Indicator（造价参考视图读空）；四输入方向防御补齐。连带抓出 useChatVoice.ts 三处 PascalCase 发送点改 camelCase。
- **结构性守卫**：TestWireShapeGuard 把普查逻辑测试化进 ci（缺标签即 FAIL，负向验证过）——v4.269 修 7 结构体没防住本轮同款，守卫是根治。
- **真机走查**：CDP 桥验 GaeaCostNoteList 返回 camelCase+UI 断言笔记卡片全字段渲染+console 零错误+清场干净；坑=整段式走查脚本 UI 段卡死，拆探针分段+看门狗全链过。
- **门禁**：ci.ps1 全绿（run1 负载 flaky 两例隔离复跑绿）/ drift OK@642 / 版本三处 4.276.0；产物=exe 49,294,336B SHA256=1A12B1CA…DA869（SHA256SUMS-v4.276.0.txt，冒烟 200 过）。

## 最新发布：v4.275.0（2026-09-13）「价格带数据源调研结案：信息价『除税价』列适配」

- **调研**（docs/gaea-priceband-datasource-research-2026-09.md）：价格带数据面基础设施已完整在产（四要素字段+统计+导入链+询价飞轮），外部自动接入四路评估后建议候选4 结案（抓取脆弱+合规灰、商业 API 同「询比价不做」口径、LLM 查价不作基线）。
- **真机验证抓到实锤当场修**：信息价样本 CSV 走 ImportPreview（零落库）——「除税价（元）」不在 fieldPrice 字典→12 行全 skip；当场修字典增「除税价/含税价」+Go 回归测试，复测 12/12 全识别。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.275.0；产物见 SHA256SUMS-v4.275.0.txt。**候选4 结案待用户确认标记**。

## AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）：恢复 14 版水位

- CI 仓库卫生守卫 WARN（.gaea/AGENTS.md 59926 B 逼近 65536 B 预算）触发既定分流：迁 v4.261.0/v4.260.0/v4.259.0/v4.257.1/v4.256.0/v4.255.0 六条入 `docs/archive/agents-version-history-2026-09.md`（八迁记录已更新），主文件恢复「最近 14 版」。水位 59926→47518 B。
- **坑**：迁移动作用 node 脚本而非 sed/正则手改——archive 是 CRLF 行尾，`indexOf('## 版本状态\n')` 匹配不上；锚点一律不带行尾、写入前先验 includes。迁移后三项完整性断言（主文件条目数 14/六条入档/迁入记录更新）全过。

## 最新发布：v4.274.0（2026-09-13）「UX 线第六刀：自定义强调色亮态自动深化」

- **根因**（对比度普查观察池第一项挖到底）：非主题令牌缺陷（lightFn glow 全是深化值），是**自定义强调色不分明暗覆盖 glow/primary**——暗色调亮的 #1dd7bf 切亮态压浅底对比仅 ~1.5，全站 accent 文字看不清。
- **修复**：`ensureLightContrast` 纯函数（lib/accent.ts，保色相压亮度至 WCAG 4.5，单测 7 例含真实案例）+ App.tsx effTokens 亮态深化暗态原样（用户存储色不变）。
- **复验**：重建 exe 复跑对比度扫描——light 告警 63→53 accent 类清零（亮态 glow 实测 #128475），dark 15 不变；剩余为禁用态/误报/设计弱化。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.274.0；产物见 SHA256SUMS-v4.274.0.txt。

## 每版度量看板补账（2026-09-13，维持轨）：v4.273.0 六行落账，滑步注记

- **发现**：slim-baseline §7「自 v4.267 起恢复逐版落账」实际只落了 v4.267 一行，v4.268~272 五版滑步（CHANGELOG v4.252/253 漏记同型教训再现）。
- **补采 v4.273.0 六行**：板块 5/5（manifests 未动）；entry chunk gz **243.6KB**（v4.267=243.0，+0.6KB≈UX 三刀纯 CSS）；冷启动 **1.39~1.70s 六轮**（bind→UI 405~424ms，同 v4.267 区间）；内存 139.1MB 工作集/152.9 私有（采样时开着用户真实工程，环境态口径注）；exe 47.0MB（49,292,288B）；**>50KB 源文件 6 个**（同 v4.267，零变化）。**全部指标零回归**。
- 表格补 v4.273.0 行+滑步注记行，自本行起恢复逐版落账。

## 对比度普查收账（2026-09-13，非版本刀，零代码）：无正文级硬伤，不改

- **方法**：CDP 13 页×明暗两态程序化扫描——枚举可见文本宿主，沿祖先合成有效背景色，算 WCAG 对比度（正文 4.5 / 大字 3.0），脚本 .tmp/walk-v4274-contrast.mjs，数据 .tmp/ui-contrast/report.json。
- **结论**：dark 15 项 / light 63 项告警，逐类甄别后**大头为脚本误报**（渐变/图片背景无法从 backgroundColor 合成——角色卡白字压图片、首页「闲庭」chip 压渐变底、accent 按钮白字，目检实际清晰可读）；真实低对比仅剩**亮态 accent 弱化文本**（徽标/链接 chip ~10 处 1.5~1.8）与 schedule 甘特行号（1.3，设计弱化）+ weixin antd purple tag（3.39）——**全部为装饰性/弱化文本，非正文，无 AA 硬伤，按「收益趋零不做」纪律不动**。
- **观察池新增（打磨候选待拍板）**：亮态 accent 弱化文本（徽标/链接类 chip）对比 1.5~1.8——若要修需动主题 lightFn 令牌（glow/accent 亮态深化），影响所有 accent 消费面，属视觉拍板项。
- **坑**：对比度自动扫描的背景合成对 background-image/渐变必然误报——告警必须逐类目检甄别后才能定刀。

## 留池清账（2026-09-13，非版本刀，零代码）：v4.270 sin_illustrate live 端到端补验通过

- **结论**：live 模型调 `sin_illustrate` 端到端全通——模型真调工具、图片真实落盘（sin/art/…「雨夜回眸」.png 1.8MB）、轨迹 artifacts 在位、`extra.illustrations['tool0']` 回写（画廊可见）、过程卡「✓思考过程·319 字·生成插图」元数据在位；截图 .tmp/walk-v4270-illustrate.png。脚本 .tmp/walk-v4270-illustrate.mjs（含空间切换修正）+ .tmp/retry-illustrate.cjs。
- **历史失败归因翻案**：前两次「模型回合未落消息」实为**上游 grok-4.6 一次性空返回**（只有 reasoning 354 字、无正文无工具）——后端兜底如实报「模型没有返回内容，请重试」（sin_handler.go:339 空正文不落库），重试一轮即成功。发送/流式/错误呈现链路全部正常，**非 gaea 缺陷**。
- **清场**：走查故事已删；产物图删除时被壳句柄锁住（Windows 删除挂起，rmSync 静默），杀壳后 PowerShell 删除成功。
- **观察池新增**：sin/art/ 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34 三张「雨夜站台」走查 prompt 产物，v4.270 失败轮次超时未清场遗留），待人工确认后删除。
- **坑**：①node rmSync 对被占用文件不抛错但删除不生效（Windows delete-pending），删完必须 existsSync 复核；②v4.270 脚本超时 throw 路径没走清场（清场代码在正常尾部）——走查脚本清场应放 finally。

## 最新发布：v4.273.0（2026-09-13）「UX 线第五刀：键盘焦点环全站恢复」

- **来源**：「继续」续 UX 线。reduced-motion 普查=基建已齐（双全局兜底）不动；CDP Tab 实测抓到 v4.255 全局焦点环**全站性失效**：Tailwind v4.3.3 产物注入源码不存在的 `:focus-visible{outline:none}`，同特异性按文档序吃掉普通规则。
- **修复**（index.css 单文件）：全局环 `!important` 必胜 + antd 输入类豁免同步 !important + **ant-btn 移出豁免**（Button 聚焦零视觉，豁免=键盘用户找不到焦点）。
- **复验**：重建 exe 六页 Tab 遍历，四页零无环，余 2 站为有意豁免的 ant-input（probe 采样时序误报）；antd 按钮聚焦实拍 2px glow 环。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.273.0；产物见 SHA256SUMS-v4.273.0.txt。

## 最新发布：v4.272.0（2026-09-13）「UX 线第四刀：窄窗响应式收口」

- **来源**：「继续」续 UX 线，换镜头 CDP Emulation 900×600 复跑 13 页——布局自适应面健康，抓到两处硬伤。
- **修复**：①底部遥测条右缘截断（全站六页）：引擎 pod max-width 240 可截断（.v3-pod-text 出省略号）+tele-key/value shrink-0，挤压顺序 pod→工程名→遥测数值完整保；②sin composer 工具行按钮竖排折字：ComposerToolbar 外层 flex-wrap+按钮组 shrink-0+按钮 nowrap（办公共享组件两处受益）。
- **测试**：tsc 0/eslint 0/composer 26+layouts 5 用例绿；重建 exe 复跑 13 页窄窗全 ok（修复前六页越界），目检两处修复实证。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.272.0；产物见 SHA256SUMS-v4.272.0.txt。

## 最新发布：v4.271.0（2026-09-13）「UX 线第三刀：滚动条全站单一真源」

- **来源**：用户「继续优化迭代前端 UI」接 v4.255/v4.260 UX 线。普查=CDP 起壳 26 页次（13 页×两态）零硬伤；视觉层抓到四套滚动条并存，v1.3.0 霓虹遗产（glow 渐变 thumb）喧宾夺主。
- **落地**（纯 CSS 零 TS 零 Go）：index.css 8px 中性胶囊单一真源（module-launcher 形态升全局）；删霓虹化覆写+清 ::selection 死规则（v4.255 拍板版一直被旧规则盖着从未生效）；删 styles.css/redesign.css/module-launcher 三处重复定义。
- **走查**：重建 exe 复跑 26 页次两态——亮青粗条全变中性胶囊，novel/sin 零变化，零回归。观察池新增=weixin 通道详情留白（待拍板）/办公右栏窄窗卡片横滚兜底/novel 技能下拉裸 id（全站惯例不动）。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.271.0；产物见 SHA256SUMS-v4.271.0.txt（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.270.0（2026-09-13）「sin_illustrate 工具入册（拍板放行）」

- **来源**：拍板池清单后用户连说「继续」=按序放行；推翻 §13.2「刻意不做」，前提（产物链）本刀补齐。
- **落地**：工具集 6→7；单链路委托 SinIllustrate；Artifacts 轨迹链（trace 字段+循环收集+result 帧）+落库回写 tool0..toolN；过程卡缩略图+元数据「生成插图」。
- **测试**：Go+4+计数锁 6→7+vitest +3；零新绑定。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.270.0。走查=部分完成如实记录（live 模型调工具留池补验，详见 releases/v4.270.0.md）；产物=exe 49,293,312B SHA256=523EAB82…ECF5（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.269.0（2026-09-13）「造价域线上断链修复：json 标签+五算带入」

- **来源**：维持轨「五算带入与含量对照端到端」走查清账时抓到两个线上真 bug。
- **Bug1**：costproject/coststage 结构体无 json 标签→线上 PascalCase，前端读 camelCase→列表卡空名/¥NaN、五算面板同断（v4.204 起）；修复=7 结构体补 camelCase 标签（存量数据兼容）。
- **Bug2**：五算带入/手动保存/询价录入 payload 带空串时间字段→Wails time.Time 解析必炸（v4.194/4.2 起）；修复=载荷省略+类型改可选。
- **测试**：Go+2（TestWireShapeCamelCase×2 形状回归锁）+vitest 2977 全绿；零新绑定。
- **门禁**：ci.ps1 全绿（run3）/ drift OK@642 / 版本三处 4.269.0。真机走查=一次性项目端到端全通（列表卡正确渲染+带入桥验落库+溯源备注+compose 诚实回退），零前端 JS 错误；产物=exe 49,281,024B SHA256=88864D3C…C2A3（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.268.0（2026-09-13）「knip unused exports 甄别清账」

- **动机**：瘦身 P5 既定池项（§100「169 甄别」）；实跑仅 11+6+8（存量数字陈旧）。
- **甄别**：全死删（含 bridge.ts 五个死 re-export）/双出口裁未用侧/契约哨兵 @public 保留/downloadBlob 误报注明（test spy 委托）；knip.json ignore 补 bindingNames.ts+drift.ts。
- **测试**：tsc 0/eslint 0/knip unused 清零/vitest 2977 例全绿；零行为变化零 Go。
- **门禁**：ci.ps1 全绿（run2）/ drift OK@642 / 版本三处 4.268.0。真机走查=CDP 四页导航扫描渲染全过零前端错误；产物=exe 49,281,024B SHA256=10A2566A…F15D5（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.267.0（2026-09-13）「原罪工具集 v1.1：sin_export 图文导出入册」

- **动机**：v4.262 留存合约清账（13.2「下刀接完产物链路再入册」）；sin_illustrate 维持刻意不做待拍板。
- **落地**：sin_export 入册（工具集 5→6，order 末位）——委托既有 SinExportMarkdown、落 sin/exports 同名不覆盖、文件名净化 fail-closed、原子写；过程卡「图文导出」+Download 图标。零新 App 绑定。
- **测试**：Go+3+计数锁 5→6+vitest+1。
- **门禁**：ci.ps1 全绿（run1+run2）/ drift OK@642 / 版本三处 4.267.0。真机走查=临时故事两回合端到端：模型真调工具、导出件存在且含正文、过程卡渲染 ✓；走查抓到「同回合写完就导=本回合正文未落库只导头部」缺口，边界写进工具 Description 并复验过，零 JS 错误；产物=exe 49,281,024B SHA256=BD2AEE86…6D827（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.266.0（2026-09-13）「原罪底稿面板内编辑：大纲/设定集」

- **动机**：v4.263 下刀候选清账（「需冲突合并设计」）；此前设定集/大纲只读。
- **设计（§14.7）**：冲突合并=回合互斥+锁内基线比对确认，不做字段级 merge——sending 锁编辑；SinNotesSave 锁内比对编辑起始快照，不符拒（「底稿冲突：」前缀）→前端覆盖确认→force；限长与工具侧同口径。
- **落地**：新绑定 SinNotesSave（641→642，play）；大纲/设定页签编辑器（编辑/写大纲/记便签入口、逐条增删改、rune 计数）；空底稿也给入口。
- **测试**：Go+1（SaveBinding 矩阵）+vitest+8。
- **门禁**：ci.ps1 全绿（run4）/drift OK@642/版本三处 4.266.0。真机走查=临时故事端到端不打模型：编辑保存桥验落盘+冲突弹窗 force 覆盖全通，零 JS 错误；产物=exe 49,269,248B SHA256=6D40A0D9…CC3C（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.265.0（2026-09-13）「原罪画廊重新生成入口」

- **动机**：v4.263 下刀候选清账——流内插图有「重新生成」，画廊只能看。
- **落地**：零 Go 零绑定（SinIllustrate 全链路既有，`arts[cue]=path` 覆盖回写）。画廊缩略图悬浮钮+大图 Modal footer 钮；与流内同一 illustrationQueue 串行；成功后 reloadMessages 全链刷新；SinIllustration 外部换图采纳（extPathRef，不在生成中才采纳）防画廊/流内不同步。失败如实 notice 原图保留；persisted:false 也提示。
- **测试**：vitest +8（定位回调/busy/sending 禁用/空 prompt/Modal+预览跟新/collect 字段/外部换图×2/reloadMessages）。
- **门禁**：ci.ps1 全绿（vitest 342 文件 2969 例）/ 版本三处 4.265.0 / drift OK@641。真机走查=临时故事端到端（结束即删，用户故事未动）：重生成 busy 态→回写换新路径→画廊/流内同步换图，零前端错误；产物=exe 49,257,984B SHA256=868668BD…65AB（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.264.0（2026-09-12）「原罪右栏面板宽度可拖拽」

- **动机**：用户口径「右侧面板宽度应该是可以自由拉伸的」（v4.263.0 下刀候选清账）。
- **交互**：左缘拖拽手柄（骑边框 8px 命中带 + accent 显色），指针拖拽与办公 useWorkspaceLayout 同源——实时跟手/向左拖=变宽/松手持久化/pointercancel 兜底；双击复位默认 268。
- **钳制**：240~640 硬档 + 视口收敛（innerWidth-520 保故事架与故事流可读）；记忆键 gaea.sin.panelWidth 自有键续用。
- **收益**：画廊列数 auto-fill minmax(104px,1fr) 随宽自适应（默认两列不变，拖宽三/四列）。
- **测试**：vitest +5（跟手+钳制+持久化 / 收窄下限 / 双击复位 / 记忆恢复 / clamp 矩阵）；零 Go 零绑定。
- **门禁**：ci.ps1 全绿（代码树 run3+终态树 run4）/ 版本三处 4.264.0 / drift OK@641。真机走查=CDP 受信任鼠标拖拽 268→448 实时跟手（中途采样 358）+落盘+画廊列数 2→3+双击复位 268，两轮零前端错误；产物=exe 49,255,424B SHA256=4A1E96E0…0102F（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.263.0（2026-09-12）「原罪右栏创作面板：角色 / 大纲 / 设定 / 插图（对标办公标签页面板）」

- **动机**：用户口径「继续优化原罪板块，增加类似办公的右侧面板，可以看角色、大纲、插图、设定」。改前右栏是静态说明栏，且 v4.262 工具写进 notes/<故事id>.json 的设定集/大纲**前端没有读取通道**。
- **绑定**：SinNotesGet（640→641，play 空间）只读便签文件返回 {notes, outline}；topicGuard 守卫 + 与工具侧同一把 sinNotesMu；缺失/损坏=空清单空大纲（同口径容错）。
- **面板**：SinSidePanel 四页签（角色=SinCastPanel 原样 / 大纲=只读 pre-wrap / 设定=逐条 #序号 / 插图=全故事画廊 + Modal 大图），页签计数徽标（**纯文字页签：真机走查实证 268px 下图标+两字折行**）；玩法/插图协议/边界收进头部问号气泡（内容逐字保留）。
- **开合**：顶栏 PanelRight 钮，记忆键 gaea.sin.panelOpen/panelTab（自有命名空间）；宽窗默认开窄窗默认收，替代 1180px 强制隐藏媒体查询；每回合结束自动重读底稿。
- **测试**：Go +1（绑定三态）+ vitest +14（面板 8 / collectIllustrations 2 / useSinNotes 4）。
- **真机走查**：用户真实故事四页签全通——大纲/设定空态轨道环（无底稿不编造）、插图 3 张缩略图全转 data URL、Modal 大图预览、开合与页签记忆；抓到并修掉页签折行缺陷；零前端错误。
- **门禁**：ci.ps1 全绿 ×2 / drift OK@641 / spaceBindings 分类锁 462→463 / 版本三处 4.263.0 / exe 49,253,888B SHA256=6E281D6A…0A880（桌面副本同源，冒烟 200）。
- **下刀候选**：大纲/便签面板内编辑（需设计与 AI 写入的冲突合并）、面板宽度拖拽、画廊「重新生成」入口。

## 最新发布：v4.262.0（2026-09-12）「原罪工具集 v1 + 过程卡：联网搜索 / 网页抓取 / 角色卡 / 故事便签 / 故事大纲」

- **动机**：用户问「原罪现在能用哪些工具？看不见调用工具的过程卡」。改前原罪**模型零工具**（单轮 chat 流，无 tools 字段），过程卡无从显示。
- **工具集**：`web_search` / `web_fetch`（委托办公 builtin，schema 转发零漂移）+ `sin_cast`（只读角色卡）+ `sin_notes`（便签 list/read/write/set/delete）+ `sin_outline`（大纲 read/write）；注册表 `sin_tools.go`（init 自注册/重复 panic/顺序固定）。
- **循环**：有界多轮（上限 4，末轮不带 tools 强制收尾）；新 ai 入口 `ChatStreamMessages`（与 `ChatStreamChunks` 共用装配）；首轮不支持 tools ⇒ 去工具重试 + notice；SinCancel 起手即登记、覆盖流式与工具。
- **过程卡**：新事件 `tool_dispatch`/`tool_result`/`notice`；`done.tools` 与 `extra.tools` 同形态（重开还原）；前端 `SinProcessCard`（思考折叠 + 工具行：标签/摘要/状态/用时/展开明细）。
- **数据面**：便签/大纲落 `%APPDATA%\gaea\sin\notes\<故事id>.json`（原子写 / id 白名单 / 损坏当空），不碰办公工作区与记忆。
- **未做**：`sin_illustrate` / `sin_export`（越契约实现、产物回写链路未接完，已挪 `.tmp/sin-next-knife/`，下刀接）。
- **教训**：并发子代理要显式禁改契约文件；产物类工具的跨线依赖必须在契约里点名责任人。
- **测试**：Go +14 / vitest +13（含预算拦截、收尾兜底、静默计时重置三例回归锁）；**门禁**：ci.ps1 全绿 / 版本三处 4.262.0 / 零新绑定；exe 49,242,624B SHA256=470F86EF…C6E68（桌面副本同哈希，冒烟 200）。
- **真机走查**（抓到并修 3 个单测覆盖不到的问题）：①模型一轮连搜 12 次 → 加调用预算（同名 ≤3/整轮 ≤8）；②收尾轮仍要工具 → 整单无正文且不落库 → 加收束令 + 兜底收尾轮；③前端静默超时是一次性 90s → 改每帧重置。最终=走查故事一轮出正文 1185 字（落库 1340 字）+ 9 条工具轨迹（含预算回绝），零渲染错误（走查故事已删）。

## 最新发布：v4.261.1（2026-09-12）「修复：原罪插图报『marshal image request: 图生图需要提供参考图』」

- **根因**：v4.259.0 参考槽门控只看「后端×模型能不能用」，没看「这一轮有没有取到图」——没选角色 / 只有远端剧照 / 本地文件缺失时 `refImages=0` 仍以 `mode=img2img` 发请求，Herdsman 图生图端点整单拒绝；回退条件 `refImages>0` 不成立，报错直出。
- **真机取证**：`%APPDATA%\gaea\logs\gaea-20260912.log` 20:26 两连发（图片后端 Herdsman）+ 用户 `sin/cast.json` 不存在（没选角色）。
- **修复**：新纯函数 `sinResolveRefImages`（图/名/台账归属 id）+ 解析为空即清 `mode/refMethod` 走纯文本；顺带修台账归属错位（多角色只有第二人有图时记错人）。
- **测试**：Go +3（红→绿：改前 3 子例全红；纯函数矩阵 4 面 + 端到端 3 子例）。
- **门禁**：ci.ps1 全绿 / 版本三处 4.261.1 / 零新绑定；exe 49,164,288B SHA256=CCE99E91…D5CAB（桌面副本同哈希，冒烟 200）。
- **真机走查**：v4.261.1 exe 打开原罪故事 → 两个插图位 20s 内自动出图（产物落 `%APPDATA%\gaea\sin\art\`、回写 extra.illustrations），日志零失败/零异常；该故事没选角色=与用户同一分支。
- **文档**：docs/gaea-sin-board-design-2026-09.md §12.5、releases/v4.261.1.md。

## 最新发布：v4.259.0（2026-09-12）「原罪插图：角色一致性锚定（文本锚点 + 图像参考槽）」

- **链路**：角色库 → 故事角色（v4.257）→ 插图人物（本刀）：文本锚点对所有后端生效；图像参考槽走绘梦 T2（ComfyUI krea2*/z-image-turbo · Herdsman 图生图，denoise 0.65）。
- **防互串**：提示词点名优先；单角色兜底；多角色未点名不锚定。
- **诚实**：能力门控（不支持就跳过并给原因，不硬塞）+ 参考槽失败自动退回纯文本重试（标 ref_fallback）+ caption 徽标可见。
- **未做**：ipadapter/pulid（后端未实现）。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.259.0；走查徽标实测（零异常）。
- **文档**：docs/gaea-sin-board-design-2026-09.md §12、releases/v4.259.0.md。

## 最新发布：v4.258.0（2026-09-12）「原罪插图：生成进度动画 + 串行队列 + 真停止」

- **进度**：抽共享进度组件 GenerationProgress（自绘梦 GenerationBar 抽出，类名/阶段名表单点维护）+ 轮询 hook useComfyTaskProgress；ComfyUI 走确定态（百分比+当前节点+已用时），其余后端走不定态光带 + 本地秒表（不编造百分比）。
- **串行队列**：同板块一次一张（图像后端单任务），排队显示「前面还有 N 张」，失败不阻断。
- **真停止**：新绑定 SinCancel（绑定 639→640）中止底层流并保留已生成部分落库（extra.cancelled）；插图「取消」接 CancelImageGeneration。
- **顺带修真缺陷**：sending 因 deadRef+StrictMode 卡死（输入框锁死）、Composer「停止」是空实现。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.258.0；走查：两处进度条 + 取消一张另一张继续 + 完成后停止按钮消失（零错误零异常）。
- **文档**：docs/gaea-sin-board-design-2026-09.md §11、releases/v4.258.0.md。

## 最新发布：v4.257.1（2026-09-12）「修复：真机插图报 AttachmentDataURL is not a function」

- **根因**：S2-3 兼容代理（window.go.app.App）只认字面名、不做 gaeaToGaea 短名映射 → 短名 `AttachmentDataURL` 在真机（Go 方法名 GaeaAttachmentDataURL）查找落空。
- **修复**：兼容代理补映射回退（字面名优先→映射名→仍未命中才 undefined）+ `api/image.readFileAsDataURL` 改走规范 bridge 路径 + HTTP 门面表加 SinB。
- **波及面**：同根因的 4 个方法（AttachmentDataURL/SavePastedImage/RecognizeImage/OCRText）一并修好。
- **测试/门禁**：vitest +3（真机形态假门面回归）；tsc 0 / eslint 0 error / vitest 全绿 / drift OK@639 / 版本三处 4.257.1。
## 最新发布：v4.257.0（2026-09-12）「原罪三刀：与办公硬隔离 + 模型中心独立卡片 + 角色库接入」

- **①硬隔离**：插图产物落 `<用户配置目录>/gaea/sin/art`（绘梦生成链 provenance 参数化，绘梦行为不变）→ 不写办公工作区、不依赖办公 ImageSaveDir；角色配置落 sin/cast.json；无记忆面接入；模型路由 routeModel("sin") 独立；工位关键词检索改为**整个 `.gaea/play` 判噪音**（改前只跳 play/exports，画室台账等会漏进办公检索）；顺带修「原罪插图被登记两次」缺陷。
- **②模型中心卡片**：FEATURES 增「原罪」+ 图标/说明；走查确认在「功能绑定」区在位（未绑定→跟随全局）。
- **③角色库接入**：右栏角色卡 + 多选弹层 → SinCastGet/SinCastSet（悬空过滤/去重保序/封顶 8）→ SinStream 注入角色块（外观/性格/背景/动机/口吻/说话样例 + 不得改设定）；只引用不改库。
- **未做**：角色参考图当图像参考槽（人物一致性锚定）留拍板。
- **门禁**：Go 全绿 / drift OK@639 / tsc 0 / eslint 0 error / vite build 成功 / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.257.0；走查零错误零异常。
- **文档**：docs/gaea-sin-board-design-2026-09.md §8/§9、releases/v4.257.0.md。

## 最新发布：v4.256.0（2026-09-12）「闲庭新板块『原罪』：对话式图文混杂故事创作（复用办公组件）」

- **定位**：用户点单的闲庭新板块（原话见 CHANGELOG）；闲庭第七个一级板块，形态=「说一句→写一段→你接着指方向」的对话式故事创作 + 就地出图。
- **落地**：前端复用办公组件（Composer/Markdown/ToolbarButton/EmptyState/三件套样式，零新建聊天组件、零改办公件）；后端新门面 SinB（9 方法）=统一聊天存储 mode=sin（与聊天板块双向隔离）+功能级路由 sin（模型中心可单挂模型）+绘梦图像链内联插图+图文 Markdown 导出。
- **协议**：模型另起一行输出 `@@插图|画面描述@@`，前端按出现次序（0 起）就地生成并嵌进正文流；正文原样落库，重开按 `extra.illustrations` 渲染不重生成。
- **边界**：成人内容口径沿用 docs/ADULT_MODE.md（零新立），四条硬边界写进提示词并有测试锁定。
- **门禁**：Go 全绿 / drift OK@637 / tsc 0 / eslint 0 error / vite build 成功 / vitest 334 文件全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.256.0。
- **文档**：docs/gaea-sin-board-design-2026-09.md（设计基线+复用清单+已知边界）、releases/v4.256.0.md。

## 最新发布：v4.218.0（2026-09-11）「阶段六 6.6 首刀：跨域 EVM——进度×造价挣值」

- **定位**：阶段六 6.6 首刀（施工债级可穿插；gaea 独有跨域）。**6.6 判据「三数+两指数出且口径注全」满足**。
- **落地**：computeEvm TS 纯函数（PV 基线分摊/EV 完成率×预算/AC 手录+SPI/CPI/SV/CV/BAC/evTop，口径注直出）+ 进度页「挣值分析」视图七档（AC 录入面）。
- **门禁**：vitest +6 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.218.0。
- **阶段六状态**：6.1/6.2/6.4/6.5/6.6 判据全满足；余 6.3 多文件 DAG（委托式深水区，动手前先立设计文档）。

## 最新发布：v4.217.0（2026-09-11）「阶段六 6.5 首刀：进度·蒙特卡洛工期带」

- **定位**：阶段六 6.5 首刀（进度两翼之二，nPlan 对位）。**6.5 判据「三档+稳定度本地可跑」满足**。
- **落地**：monteCarlo TS 纯函数（种子化 RNG+三角分布采样+逐次重算 CPM；P25/中位/P75 三档+直方图+关键路径稳定度/换线风险）+ SchedulePage「工期模拟」视图（不确定度滑杆）。
- **门禁**：vitest +8 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.217.0。
- **下一步**：6.3 多文件 DAG（委托式深水区）；6.6 跨域 EVM（施工债级可穿插）。

## 最新发布：v4.216.0（2026-09-11）「阶段六 6.4 首刀：进度·DCMA 14 点计划质量体检」

- **定位**：阶段六 6.4 首刀（进度两翼之一，对标 Acumen Fuse）。**6.4 判据「十四点纯函数全测」满足**。
- **落地**：dcma.ts TS 纯函数（14 点逐项+阈值+超标点名+n/a 诚实口径+CPM 断裂 score=null）+ SchedulePage「质量体检」视图（DcmaView 纯展示）。
- **门禁**：vitest +11 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.216.0。
- **下一步**：6.5 蒙特卡洛工期带（同工作日口径哲学）；6.3 多文件 DAG；6.6 EVM 可穿插。

## 最新发布：v4.215.0（2026-09-11）「阶段六 6.2 首刀：记忆驱动项目本体——项目本体注入」

- **定位**：阶段六 6.2 首刀（依赖 5.1 图谱底座）。**6.2 判据满足**（决策引用+可关闭）。
- **落地**：BuildProjectBrief 纯函数（固化优先+[MEM:] 引用键+决策来源归因+600 rune 预算）入 work 空间装配前缀；config project_brief 键+GaeaMemoryBrief/GaeaSetMemoryBrief 绑定（611→613）+MemoryPanel「项目本体」开关。
- **门禁**：Go 全量 0 FAIL（+2）/drift PASS@613/vitest MemoryPanel 2/2/版本三处 4.215.0。
- **下一步**：6.3 办公·多文件 DAG（委托式深水区）；6.4/6.5 进度两翼；6.6 EVM 施工债级可穿插。

## 最新发布：v4.214.0（2026-09-11）「阶段六 6.1 首刀：可审计默认化——文件预览默认版本条」

- **定位**：阶段旗交阶段六（书斋纵深·办公主角），6.1 首刀；纯前端、零新绑定、零 Go 改动。**6.1 判据满足**。
- **落地**：FileVersionStrip 默认可见版本条入文件预览（版本数/最近改动/状态/可恢复；展开=该文件 VersionTimeline 逐版本预览/恢复；pptx 页码归因来自证据卡 p3 摘要）。
- **门禁**：vitest +4（FileVersionStrip）/FilePreview 27 例全绿/tsc -b 0/绑定 611 零变更/版本三处 4.214.0。
- **下一步**：6.2 办公·记忆驱动项目本体（依赖 5.1 图谱已就位）或 6.3 多文件 DAG；6.6 EVM 施工债级可穿插。

## 最新发布：v4.213.0（2026-09-11）「5.3 首刀：记忆生命周期三态」

- **定位**：阶段五 5.3 首刀；「90 天一刀切」替代。**5.3 出口判据全满足**（蒸馏合并/预取开关此前已落地）。
- **落地**：固化（SchemaV19 pinned：豁免清理+排序加权+Pin/Unpin 事件留痕）+ 衰减（DecayScore 纯函数+LifecycleOf 分类）+ 三态可查（GaeaMemoryLifecycle/GaeaMemoryPin 绑定 609→611；OfficeMemoryLibrary 统计行+FactCard 固化锁）。
- **门禁**：Go 全量 0 FAIL（+5 用例）/ drift PASS@611 / vitest 6/6（OfficeMemoryLibrary）/ 版本三处 4.213.0。
- **阶段五状态**：5.1/5.2 证据链/5.3 判据全满足；余=5.2 主动调度与 5.1/5.2 UI 深化按需另刀，阶段五可评估出口。

## 最新发布：v4.212.0（2026-09-11）「5.2 首刀：上下文编译 dump/diff + 前缀稳定证明」

- **定位**：阶段五 5.2 首刀；纯 Go、零绑定、前端零改动。证据链出口判据落地。
- **落地**：agent 消息级编译摘要（逐消息 SHA256 链+长度前缀防拼接歧义）+ DiffMessages 相邻请求前缀稳定判定；判定随 RequestHeader 落会话日志（dump/diff/回放一体）；contextview 趋势柱携带 Prefix 与 CacheHitTokens 配对。
- **门禁**：Go 全量 0 FAIL（+4 用例）/ 版本三处 4.212.0 / vitest 以 v4.210 基线为准。
- **下一步**：5.2 深化（意图分类×预算调度的主动调度）或 5.3 记忆生命周期三态（touch 事件已留痕），按规划推进。

## 最新发布：v4.211.0（2026-09-11）「5.1 收口：回复发出前剥离悬空引用」

- **定位**：v4.210.0 的 5.1 余项收口刀；纯 Go、零绑定、前端零改动。**5.1 出口判据全满足**。
- **落地**：agent stream() 收尾定稿闸（FinalizeText 钩子：Message 事件前改写+同进 session/摘要）+ boot 记忆开关注入剥离闭包（StripDanglingCitations 纯函数，不 Touch；剥离键落 dangling cite 事件）。
- **门禁**：Go 全量 0 FAIL（+5 用例）/ 版本三处 4.211.0 / vitest 以 v4.210 基线为准（纯 Go 刀先例）。
- **下一步**：阶段五 5.2 上下文编译（意图分类→预算调度→前缀稳定排序；embedding 向量列启用）。

## 最新发布：v4.210.0（2026-09-11）「记忆语义图谱：事件日志投影出图」

- **定位**：阶段五「记忆OS+上下文编译」首刀（规划大纲 §2 主轴旗，底座#1）。设计基线=docs/gaea-memory-graph-51-design-2026-09.md。
- **落地**：memory_events 事件日志（SchemaV18 追加式，五条写路径挂钩，Touch 逐条留痕）→ ProjectEvents 纯函数投影（实体/事件/来源三向边，[MEM:name] 图寻址，悬空拒写留 Dangling）→ mem_graph_* 物化（embedding 占位列；读前水位对账懒重建=删库可重建）。
- **接线**：绑定 608→609（GaeaMemorySemanticGraph）；MemoryHub 图谱页「关联图/事件图谱」切换；回合收尾引用解析落 cite 事件（命中+悬空）。
- **门禁**：Go +9 用例 / vitest +1 / tsc 0 / drift@609 / 版本三处 4.210.0。
- **欠账**：5.1 余项=真·发送前悬空引用剥离（流式层另刀）；UI 遍历/手动重建按钮按需另刀（Go API GraphNeighbors/RebuildGraph 已备）。

## 最新发布：v4.209.0（2026-09-10）「组价含量对照：拆解含量 vs 同类条目分位带」

- **造价刀路池 §6 收官，survey 六项缺口全清**——docs/gaea-cost-domain-survey-2026-09.md 已逐项回写收口状态（§3→v4.194 / §4→v4.195 / §5→v4.196 / §6→本刀），该文档不再充当欠账池。
- **缺口**：AI 拆解人材机含量只有金额自洽校验（v4.158），含量本身离谱（如水泥 420kg 对同类 300kg）无人提示。
- **落地**：`cost.CheckContentBaseline` 纯函数——拆解组件 vs 相似条目池同键含量 P25/P75 分位带（R-7 同口径），带外 warn 带内静默；匹配键=标题归一（全角→半角/去空白/小写）+单位归一精确相等（含双方都空），**一方缺失一律不比（宁缺勿误）**；同值退化带 ±5% 容差；同键样本 <3 不比对。接进 `GaeaCostCompose` `composeChecks`（金额自洽+含量对照两层；对照池=相似条目组件明细，与价格带同池；池空该层静默），Checks 随视图展示+Apply 留痕自动承载，**零新结构零新绑定前端零改动**。边界：外部「行业含量区间」数据源明确不引入——基线源=用户自己的库（私人记忆哲学）。
- **门禁**：Go 全量 exit 0（116 包 0 FAIL，+5 用例：contentband 四例+composeChecks 接线一例）/ drift PASS@608（零绑定）/ 纯 Go 刀 vitest 以 v4.208 全绿基线为准 / 版本三处 sync 4.209.0。
- **产物**：build.bat（1m02.7s）→ `build\bin\gaea.exe` **48,580,096 B** + 冒烟 `/api/health` 200 + Desktop\gaea.exe 同哈希；SHA256=**637c6d835d6355b62edfffbed68602bda35fec1e78a7a8d1f7b948607e7f9537**（releases/SHA256SUMS-v4.209.0.txt）；未打 tag（随 v4.200+ 现状）。
- **欠账**：组价含量对照端到端待真机（观察池）；造价域新缺口另立调研。

## 品牌资产补齐（2026-09-10，非版本刀）

- **缺口**：2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核）；**两个二进制件没跟上**——`build/appicon.png`（1024²）与 `build/windows/icon.ico` 仍停在 **2026-08-05** 的旧视觉（翡翠球体 + 破土嫩芽 + 星芒，深蓝底板 `#0F172A`）。后果：exe 内嵌图标、任务栏、资源管理器、桌面快捷方式**全显示旧 logo**，只有窗口内 UI 是新 logo——「换了一半」。
- **落地**：① 无头 Edge 渲染 `build/appicon.svg` → 1024×1024 RGBA PNG（关键参数 `--default-background-color=00000000` 保透明，圆角外 alpha=0 与旧资产同规范）；② Pillow 由同一母图生成 **7 档 ICO**（16/24/32/48/64/128/256——旧资产只有 6 档、缺 24，本次补齐 Windows 标准全集）；③ 重建产物 `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**，`-s` 跳过前端编译用现成 dist）。
- **验证（三条独立证据）**：① **几何精度**——核心圆 r=50@viewBox512 实测 1024 图中直径 **200px**、圆心 (539.5, 511.5) 对理论 (540, 512)；环 `stroke-width=48` 实测线宽 **96px**；圆角 `rx=104` 对角线首个不透明像素 d=**61** 对理论 60.9——渲染零缩放零偏移。② **exe 内嵌图标提取**（`ExtractAssociatedIcon`）——底板 `#0B1210`、米色核 (243,230,198)、翡翠环 (103,235,195)=新「地核 G」；旧版是 `#0F172A` + 绿球。③ **dist bundle**——新 logo 独有标记 `gaea-logo-ring`×3 / `gaea-logo-light-ring`×3 / `gaea-logo-core`×2 / `0B1210`×2 在册，旧独占色 `#6ee7b7`/`#047857` **零命中**；另按旧 SHA256 全仓比对，**零旧 logo 二进制残留**。
- **门禁**：wails build exit 0 / 冒烟 `/api/health` **200** `{"status":"ok"}` / 纯资产刀——**零前端源码改动、零 Go 源码改动、绑定面不变**。
- **产物**：`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希）。中间态 903F9759…C36EC068 系「只换 appicon.png + 主图 ICO」那一版，小尺寸优化后重建覆盖；原始 59C2B518BDB6165CCBAC07BBC565759A2D7571F254E287E27AA00B2BCCD97A16 系 v4.208.0 归档值（releases 档案自洽，未动）。
- **坑（四条，按价值排序）**：
  1. **无头 Edge 截图默认白底**——不传 `--default-background-color=00000000` 时 SVG 圆角外会合成为**不透明白**（CSS 里写 `background:transparent` 无效，截图不继承页面背景），透明通道静默丢失；旧资产角像素 alpha=0 可作规范基线对照。
  2. **颜色特征校验必须先验证该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变（`stop-opacity=0.12`）与**旧** logo 主色，拿它判新旧会把新资产误报成旧的；判据只能用**独有**标记（SVG id 名 / 独占色 `#6ee7b7`、`#047857`）。
  3. **`pwsh` 不在 PATH**——手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（非 app 故障，易误判为冒烟失败）；`build.bat` 内已有 `where pwsh` fallback 到 Windows PowerShell，手工执行要自己判。
  4. **运行中的 exe 可被 `Copy-Item` 直接覆盖**（Windows 允许替换映像路径，进程持旧映射），桌面副本无需先关 app；但**已加载实例不会换图标**，需重启才是新 logo。
- **小尺寸可辨识优化（同日追加）**：原方案对大图统一降采样，**16×16 环宽仅 1.5px、核 3.1px**，G 字形糊成一坨深色块。新增派生资产 **`build/appicon-small.svg`**（环 48→64、核 r50→62、去 halo），ICO 按尺寸分流源图：**16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**，字重过渡平滑（不像 24→32 那样跳）。**坑：Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 每帧 ICONDIRENTRY 16B + PNG 帧；**256 档的宽/高字段写 0**，写 256 会溢出单字节）。对比目检 `.tmp/logo-render/small-size-compare.png`（16/24/32 × 浅底/深底 × 原版/变体）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。

## 工作空间与文档整理（2026-09-10，非版本刀）

- **目录**：根目录 48 份散落的 `SHA256SUMS-v4.*.txt` 归位 `releases/`（该目录现 218 份，归档位置统一）；`nul`（Windows 重定向残留）删除；`.tmp` 清掉可再生的构建/测试缓存与目检 profile（`gocache*`/`gotmp*`/`node-compile-cache`/`edge-home`/`edge-v7`/`smoke-gaea.exe` 等），**释放 ~2.98 GB**（3354 MB → 395 MB）。
- **文档索引**：`docs/README.md` 复核重建——45 份顶层文档 + 2 子目录**全覆盖**（补回 7 份漏登记件：角色域盘点/30 分钟上手/office 枢纽审计/瘦身基线+刀2+P2 三份证据/壳内残留审计），状态行按 git log 对齐，并立「未登记=孤儿」维护规则。
- **归档索引**：`docs/archive/README.md` 顶层 53 项**逐条登记**（原表格只用通配描述，实际漏登 14 项），磁带类大件标明覆盖范围与缺口。
- **AGENTS.md 瘦身（本刀最大发现）**：该文件曾 **104.9 KB**，超工作区指令预算 **65536 B**——尾部「执行纪律 / 交互纪律（不许用等待换时间）/ 工装 / 长期规划 / 项目定位 / 技术栈 / 发布流程 / 沙箱备忘 / 本地 TTS」**整段对后续会话不可见**。二次分流（v4.173–v4.146 逐版迁入 `docs/archive/agents-version-history-2026-09.md`）后 **45.7 KB**，全文可见；归档件同步标明覆盖范围（v4.173–v4.146 + v4.49 及更早）与缺口（v4.50–v4.145 只在 CHANGELOG/releases）。
- **引用完整性**：修掉 12 处「文档已移入 `docs/archive/` 但引用未跟」的悬空路径（含 AGENTS 三处：执行审计 / VoxCPM2 / CosyVoice 记录）。
- **防复发门禁**：新增 `scripts/check-docs.mjs`（① docs 顶层孤儿登记 ② `docs/` 悬空引用（带外部路径白名单）③ AGENTS.md 字节预算水位）接入 `scripts/ci.ps1`——同类漂移下次直接红。
- **产物保留策略落地（2026-09-10 用户拍板：保留最近 5 个版本）**：`releases/` 下 51 个 exe / 1996.6 MB → 保留 **v4.186.0 / v4.98.0 / v4.97.0 / v4.96.0 / v4.95.0**（226.6 MB），删掉其余 46 个 / **释放 1770 MB**。每个被删版本的**身份仍在** `releases/SHA256SUMS-vX.Y.Z.txt`（218 份校验和覆盖到 v4.99.0 及更早）；策略已写入 `releases/README.md`「本地产物保留策略」与 `.gaea/AGENTS.md` 发布流程第 4 步（下次发版删第 6 新的一版即可）。
- **未动（体量大且属用户数据，等拍板）**：`clones/` 参考仓（mpp2013/projectlibre/unsloth，**338.9 MB**——含隐藏 .git；首轮统计未计隐藏文件故报 212 MB）、`backups/` 120.4 MB、`whisper_data/` 运行时数据 49.5 MB、`.tmp/codex` 67 MB。

## 最新发布：v4.208.0（2026-09-10）「首页 v7 重设计：书斋「文书台」/ 闲庭「游园画廊」」

- **缺口**：v6 首页仍被判廉价——根因是「等大圆角卡片 + 描边」一种形态重复到底，层级只靠 13/14px 微差。
- **落地**：结构收敛为三形态（仪表条 / 账页 / 海报墙）+ 四档排印 + 三级色调面 + 版式记号（序号水印 / 月洞门细环），删极光斑；信息零删除，契约 testid 与 `.garden-banner` 保留。修复重写时遗失的 `.ml-avatar-ai`/`.ml-avatar-user`（气泡左右曾退化为同底色）、清掉空挂 `.p-foot-wide`、删除误入的 pnpm 锁/工作区桩文件。
- **附带门禁可信化**：CI 由「无条件重试一次」改为「只在册 flaky 隔离复跑」（`known-flaky.txt` 当前为空 + 分类器自测）、`maxWorkers` 跟物理核（原按逻辑核开 ~31 worker 导致重组件超时假红）、`testTimeout` 30s、RTL `asyncUtilTimeout` 5s。
- **门禁**：tsc/eslint 0、e-check OK、drift PASS@608（零绑定）、Go 全量、vitest 全量（324 文件 2799 例）、版本三处 sync 4.208.0。
- **产物**：主树 `wails build -ldflags "-s -w" -trimpath`（1m08.5s）→ `build\bin\gaea.exe` 48,570,880 B / 46.32 MB；冒烟 `/api/health` 200 通过；Desktop/build\bin 双副本落位；SHA256=59c2b518bdb6165ccbac07bbc565759a2d7571f254e287e27aa00b2bccd97a16（`releases/SHA256SUMS-v4.208.0.txt`）。构建后工作树仍干净（`npm install` 未改写 lockfile）。
- **目检**：Vite dev（`?mock=1`）+ 无头 Edge（CDP 9333），两空间 × 1440/1440 高/1100/880 共 10 张；无横向溢出，880 档书斋单列、闲庭海报墙 2 列塌缩正常。
- **欠账**：海报墙首张（小说）大样中段留白偏多（待用户观感定夺）；浅色主题两首页未目检；壳内真机走查池不变。

## v4.207.0 / v4.206.0 / v4.205.0（2026-09-10）主题令牌收口 + 小说书房工坊接线

- **v4.207**：深色主题硬编码深灰——RelationGraph 缩放/提示/图例写死 `#555/#ddd`、进度里程碑菱形/标签写死深灰，改走 `--color-text` / on-surface。
- **v4.206**：33 处 `bg-accent text-white` → `text-accent-fg`（= on-primary：暗色深字、亮色白字），App 补注入 `--color-on-primary` / `--color-surface`。
- **v4.205**：小说书房工坊接线——书架画廊头 + 正在编辑横条 + 书脊/角标；阅读页属性检查器由 `display:none` 改为默认可折叠（章节体检入口回来）。
- 三刀均纯前端、绑定 608 零变更、Go 零改动。

## 最新发布：v4.204.0（2026-09-10）「组价多方案对照：P25/中位/P75 当场切」

- **缺口**：survey §2 多方案对照——三档数字在，推荐价钉死中位数。
- **落地**：ComposeModal 点选 P25/中位/P75，应用价随档；cost_compose `mode` + 三档对照行。不拆逐步盖章。608 零变更。
- **门禁**：ComposeModal 11/11（+1）、Go TestCostCompose 绿、tsc/eslint 0、版本三处 4.204.0。
- **欠账**：流式打字机/费率政策对照不做；含量基线等样本。

## 最新发布：v4.203.0（2026-09-10）「空间策略其余功能域键：总闸当场切」

- **缺口**：v4.190 七键写路径已通，UI 只放 gaea 主控；其余功能域覆写仅非空 chip，空键看不见写不了。
- **落地**：纯前端 608 零变更。每张空间卡「其余功能域」六键常显可编（对话/轻语/小说/办公文档/角色库/例行），闭环与 gaea 同款；空串清除。
- **门禁**：vitest StrategySection 8/8（+2）、tsc/eslint 0、版本三处 sync 4.203.0。
- **欠账**：功能域×空间交叉矩阵按需另刀；壳内真机走查池不变。

## 最新发布：v4.182.0（2026-09-09）「双空间首页分版：书斋/闲庭定名 + 切换器迁首页顶栏」

- **定名**：书斋（work，Study）/闲庭（play，Lounge）三语（shell.space.*/shell.search.scope.*+space.ts 兜底）。
- **切换器迁移**：首页顶栏 SpaceSwitch（胶囊组 aria-pressed+模型徽标，直连 switchSpace）；rail 顶部改竖排空间指示徽标（非交互），分域导航不变。
- **两首页分版**：书斋「文书台」三栏效率台（Hero 命令条+最近文档流水主角面板+能力矩阵+右舷状态栏）；闲庭「游园画廊」全幅画廊（旗舰横幅+大卡两列+园底信息带：进度/继续话题 chips/记忆/遥测细条）——信息零删除形态分化（ui-ux-pro-max 定 dial）。
- **门禁**：vitest 2745（+3 ModuleLauncher.test；flaky 族单独复跑绿）、tsc/eslint 0、e-check OK、locale 0 死键、零绑定 drift PASS@602、零 Go 改动、build strip+冒烟 200、SHA256=F8B238863236F53803AB07822A1136366EEC92E97403BC5B592C1C04214E1337。
- **欠账**：壳内真机走查两首页观感/切换手感（真机池）；改名再调=纯 locale 三文件。

## 最新发布：v4.181.0（2026-09-09）「上下文/轨迹按会话读取 + GLM Coding Plan 429 分诊」

- **根因**：Agent 网络不按当前会话=GaeaTrajectory/GaeaAgentNetwork 无参绑定恒读内核 ga.ctrl 单例会话，UI 切历史会话后不跟（「UI 会话≠内核会话」根因家族，其余无参绑定待审计）。
- **修复**：两绑定变参 sessionPath（显式优先/缺省回落内核兼容）+resolveGaeaSessionPath；bridge 契约数组形态+mock+agentNetworkStore 路径声明（跨会话清快照宁空勿错）+六消费方接线；BrowserPanel 保持内核兜底。
- **附带 GLM 429 分诊**：Coding Plan 资源包挂 coding 端点计费域，标准端点 429=端点未切（非 Key 坏）；glmPing 429 时提示切换「编码套餐」端点。
- **门禁**：Go 全量 0 FAIL（+1 显式路径用例）、vitest 2742（+3；flaky 族单独复跑绿）、tsc/eslint 0、drift PASS@602（零绑定名）、build strip+冒烟 200、SHA256=D3BD8386862D5F74EC2BC63F8CA5B35394FA812588376DA306ECAC078AA62D55。
- **欠账**：TaskCenter session 维度结构刀；无参绑定「内核会话」语义逐个审计。

## 最新发布：v4.180.0（2026-09-09）「办公任务管理会话关联刀：会话待办区入任务管理页」

- **定位**：用户反馈任务管理「没与当前会话关联、全是固定内容」——页面主体 TaskCenter 数据源 `GaeaTaskList`=全局后台任务表（cron 周期任务持续在列，天然会话无关），而会话真任务（todo_write 待办）无入口；子代理/本地模型工具两区本就按会话建册。
- **修复**：TasksWorkbench 首区新增「会话待办」——直订全局 controller store items（零 props 穿层，会话切换自然更替，ContextModal 弹层同源正确）+ 复用 useTodoExtractor 提取最新 todo_write；三态条目+进度 n/m，无待办不占位。页面四区=会话待办/子代理/本地模型工具（会话）+任务中心（全局）。
- **门禁**：vitest **2739/2739 首跑全绿**（+1）、tsc 0、eslint 0、e-check OK、drift **PASS@602**（零绑定）、零 Go 改动、build strip+冒烟 200、SHA256=69C89001048CB8BFD99BCE7122C69AB92A8C01A85CB4E34D23BAA45A1A71C0DD。
- **欠账**：TaskCenter 会话关联需 Go 任务表加 session 维度（结构刀另立版本）；「会话后台任务」过滤视图等该维度就位再评估。

## 最新发布：v4.179.0（2026-09-09）「瘦身清点刀：knip 死代码清除 + 依赖显式化 + 冷启动基线打点」

- **knip 首扫三发现三处置**：①死文件 13 全删（App/AppBar/MobileSheet/SettingsMobile/SettingsUpdates/自制 Tooltip/PromptShelf/office 死链对/memoryhub 两件/typesGenerationCheck/m3-palette，全仓零 import 甄别含 scripts 层；连带清 zIndex.ts 过时注释）；②幽灵依赖 13 包显式化（jszip×8 处/katex/dayjs/unified/hast-util-sanitize/@lezer×8 共 20 处 import 靠传递依赖侥幸工作，按 lock 现版本钉死）；③顺带清死传递依赖 @codemirror/search@6.7.2（零引用）。三存疑依赖验活结案：gsap/docx-preview/unist-util-visit 全部在用保留。
- **冷启动基线打点（轨道五先测量后优化）**：Go app.New 一条+Startup 七段（migrate/logging/engine/voice+weixin/charlib+stores/tts-kickoff/schedulers）+total 落长期日志——日常启动自动积累真机基线；前端 gaea:boot mark→两 rAF gaea:interactive measure（CDP 可读；有意不落 gaea.log 避免污染 Error 级日志通道）。
- **初始化链审计结案**：TTS/价格/索引/模型刷新/预载/巡检全部已异步化，同步链只剩轻量磁盘活——懒初始化改造无需立项，轨道五收敛为真机采集（与壳内真机池同窗口）。
- **门禁**：vitest 2738/2738（5 例并发 flaky 单独复跑全绿）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、e-check OK、Go 116 包 0 FAIL、build strip+冒烟 200、SHA256=3FCB187F496AF2B1A89884D860F675D1E56A92B6B50AA932E0655A65521D320F。
- **度量**：exe 46.16MB 持平（死文件本就不进 bundle，收益在仓库卫生与依赖正确性）；knip 复扫死文件/幽灵依赖双清零。
- **欠账**：knip Unused exports 169 项挂下版（测试专用/预留面/真死三态甄别）；壳内真机池/审计刀D 真机取证不变。

## 最新发布：v4.178.0（2026-09-09）「造价·条目匹配键刀：定额编码贯通存储→导入→匹配→工具面→UI」

- **造价域缺口 §1 首刀**（docs/gaea-cost-domain-survey-2026-09.md）：条目匹配全靠标题字符串 → 编码体系贯通。
- **数据层**：SchemaV17 `cost_entries`/`cost_estimate_items` 增 `code`（归一化存储）+ `idx_cost_code`；迁移测试「新库全链 V1→V17 + V16 升级」双覆盖；`cost.NormalizeCode`（全角→半角/去空白/大写，空串=未录入）。
- **导入链**：表头 `fieldCode` 映射（定额编号/清单编码/项目编码/编码等 8 关键词最长胜出，「编号」仍噪声）；**编码优先匹配**——带码命中→覆盖提示、带码未命中→直接新增（同标题不同编码不标题兜底，杜绝跨子目误覆盖）、无码原语义；AI 解析 prompt 增 code 提取；vision 有意不动。
- **匹配消费方**：costref `entryIndexes` 三索引 + `matchEntry` 编码精确优先（matchedBy=code|entry_name|title）；带码未命中宁漏勿误配（测试钉死）。沉淀携带 item.Code + 引用条目继承编码。
- **工具面/UI**：cost_save 增 code 参数（覆盖未传保留原编码）+ cost_search 表格编码列；EntryModal 编码表单项 / LibraryView 编码 chip+行前缀 / ImportModal 可编辑编码列 / ProjectsView 明细编码输入+EntryPicker 带入；mock 同批贯通。
- **附带收口：e-check 三处陈旧守卫修复**（干净 HEAD 实测即红，非本刀引入）——E25 ChatSend/ChatImportTopic 接受 bridge 形态（v4.174 双轨退役甩下）；E24 MemoryHubPage→GraphView 接受 React.lazy 动态 import（v4.176 甩下）。
- **门禁**：Go 全量 0 FAIL（五包 +9 测试）、vitest **2738/2738**（2737+1）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、locale 0 死、e-check OK、build strip + 冒烟 200、SHA256=F83CDB9BED35C7B4F045CC835B59981B95B5B4B67E39EC1D5282EF8F2E199561。
- **欠账**：造价缺口 2-6 不变；编码存量回填等带码样本导入自然补全（不做标题猜测回填）；壳内真机池/审计刀D 真机取证不变。
## 最新发布：v4.177.0（2026-09-09）「瘦身 P4 懒加载三刀收官（mermaid 动态化 H1 + locales 按需 H7 + SearchModal lazy H8）」

- **H1 mermaid 动态 import**：Markdown.tsx/mermaidPng.ts 删静态 import → ensureMermaid 异步 await（模块缓存，strict/loose 配置原样）；MermaidBlock 迁 useEffect async IIFE（取消/失败语义不变）。**全仓 mermaid 静态 import 清零、entry 对 mermaid.core 零引用、566.9KB 独立 async chunk 按需**；测试零改动。
- **H7 locales 按需**：zh 静态保留（首帧零异步）+ en/zh-TW 动态 import 显式分支（Vite 静态解析独立 chunk）；useT/t/setPref 签名零变动；Provider 切换先同步渲染回退→chunk 就绪 forceRender+alive。
- **主代理关键修复**：translate 回退链 `DICTS[locale]→DICTS.zh→DICTS.en→key`——**zh 静态恒可用兜底**，非 React 调用方（tools.ts 摘要，currentLocale 默认 en）在 en chunk 未加载时裸键永不外泄（首跑 13 例失败根因）。5 测试补 beforeAll(loadLocale('en')) + ModelSwitcher en 用例 await。
- **H8 SearchModal lazy**：React.lazy + Suspense fallback null；测试不经 MainLayout 零改动。
- **门禁**：vitest 2737/2737（首跑 13 例 i18n 时序失败修复后全绿；1 例 FilePreviewModal 并发负载 flaky 单独复跑 15/15 绿在册先例）、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=E98E49CD8134C2F0964A1E041F49714465A385AB678E6295A946F94971E7B5D0。
- **度量**：**entry 1193→755.48KB（−37%）**；en=76.95/zh-TW=74.91/SearchModal 独立 chunk；**exe strip 实战验证**（build.bat 固化 `-ldflags "-s -w" -trimpath`，46.2→46.15MB 减量甚微——Go 符号表在 Wails 应用占比小，-trimpath 安全收益为主）。
- **P4 收官**：Go >50KB 0；entry −37%；MemoryHubPage 页壳 15.95KB；mock/office 1.1KB 入口；wailsjsCompat 退役（v4.174）。
- **欠账**：壳内真机池（rail 切换器/home 最近文档/.gsched）、审计刀D 真机取证（printSvg print/拖拽/粘贴）、观察池刀3（工作台内嵌办公）待评估——全部真机绑定的观察项不变。

## 最新发布：v4.176.0（2026-09-09）「瘦身 P4 结构刀2 + entry 懒加载（mock/office 拆分 + MemoryHubPage 全 tab lazy + mock 异步 chunk）」

- **结构刀2**：mock/office.ts 51.2→1.1KB 入口（6 文件：types/state/schedule 10.5/methods_xlsx 4.9/methods_office 32.7/build）。**TS2632 教训**：`let mockScheduleCurrent` 被 ScheduleProjectCreate/Delete 直接重赋值——跨模块重赋值导入绑定被 TS 禁止，schedule 状态与方法必须同文件（与 weixin.ts 域内私有状态先例一致）。
- **运行刀 entry 懒加载**（调研修正 P0 两处：cytoscape/cynefin 是 mermaid 传递依赖非 hub 相关、pageLazy 是 PDF 逐页懒挂载非页面 lazy——页面级 lazy 早全就位）：
  - **H6 mock 异步 chunk**：proxy.ts 删静态 import → startMockChunk 单例 + 真机门控零加载 + 冷路径异步 thunk；events 走 mockEventSharedSync/waitMockReady。**index 1149→967.23KB（−182KB/−15.8%）**。
  - **H2-H5 MemoryHubPage 页内 lazy**：8 组件全 React.lazy（**1402.5→15.95KB 页壳**；GraphView/three.js 1,354.97KB、KnowledgePanel/Markdown/katex/mermaid.core 全独立 chunk 按需）。
  - **H1 mermaid 动态 import 递下一轮**（mermaidPng.ts 在 footprint 外静态可达 + memoryhub lazy 已拆共享 chunk + Markdown 测试面大）。
- **连带测试时序修复×2**：mock 异步化使 `window.__mockScheduleFile`（schedule hook）与 SearchModal RouteIntent/UnifiedSearch 演示规则在同步断言时未就绪——两测试补 `beforeAll(await waitMockReady())`，bridge.ts 入口加 waitMockReady re-export。首跑 7 例失败修复后全绿。
- **门禁**：vitest 2737/2737 全绿、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=402A1B1A937CF6690E435C1F4E61AA6BB6D72487543A6B078E0F9F5F132DBD2F。
- **欠账=P4 续**：H1 mermaid 动态 import（+mermaidPng.ts）、H7 locales 按需（−160~200KB）、H8 SearchModal lazy（−20~40KB）、**exe strip**（wails build -ldflags "-s -w" -trimpath 发布版，dev 保留符号）；壳内真机池/审计刀D 不变。

## 历史发布（v4.176.0 及之前，已归档）

- **v4.175** 瘦身 P4 结构刀1：三巨文件拆分（config.go/engine.go/store.ts，Go >50KB 2→0）
- **v4.174** 瘦身 P3 版3终局：bridge 双轨退役收官（wailsjsCompat shim 删除，LegacySurfaceNames 292→184）
- **v4.173** 瘦身 P3 版3 批次二：chat/novel/settings 三族 22 文件迁 bridge（契约扩展 29 方法+元组契约错位修正）
- **v4.172** 瘦身 P3 版3 批次一：wailsjsCompat 首批 12 文件迁 bridge（契约扩展 16 方法+2A/2B 迁移；LegacySurfaceNames 292→276）

- **v4.170** 瘦身 P3 结构版1：巨文件首批 4 拆（App/bridge/types/GanttView，>50KB 9→5；逐字节/绑定零变更纪律）
- **v4.169** 瘦身 P2 主干：双空间并列落地 + home 空间感知化 + 走查待证项收口
- **v4.168 / v4.167** 瘦身 P2 刀1（schedule 并入办公文档面）+ 刀0（白名单解耦、P0 基线落档、审计刀D Filters、nanoid 升级）
- **v4.166 / v4.165 / v4.164** 瘦身 P1 快赢（轮子四刀、审计刀B/C、locale 死键、依赖验活）+ 拍板落档（路线A+私有许可）+ 收敛计划 W1 卫生刀
- **v4.163 / v4.162** gaea 长期日志机制（按日分文件+365 天保留）/ 进度计划导入导出壳内修复（原生对话框链路）
- **v4.161 ~ v4.159** 进度计划双代号线：对齐标杆二期（工程标尺+图面语言）、一级/二级分级横幅、画布手感（滚轮缩放+行距自适应）
- **v4.158 ~ v4.156** AI 组价复核闭环 / docx 证据链补齐 / pptx 真编辑刀2+刀3（编辑面板+版本对比，Office 三件套编辑闭环）
- **v4.155 ~ v4.150** 双工期口径四刀：SS/FF/SF×cd 全搭接重推、MPP Project 2013+ 变体导入、mspdi 互通、板块 UI、agent 通道、任务级工期单位（wd/cd）
- **v4.149 ~ v4.144** diff 确认卡四刀（回滚/结构化 diff 预览/引擎逐条确认/ops 投影对拍收官）+ 工程复制 + 导入为新工程安全化
- **v4.111 / v4.110** 进度计划板块起始：CPM 引擎+三视图自由切换、Project/斑马对齐（v4.112~v4.143 期记录以「最后更新」大段落内联保留：多工程/资源成本/AOA 手动布局/基线对比等）
- **v4.109 ~ v4.100** pptx 真编辑刀1 数据层、导图画布/多维表、Model Hub 蒸馏 MH1-4 + ComfyUI 预热、Hub 落库+绑定转正、办公搜索对齐、GenUI 围栏审计
- **v4.99 ~ v4.90** 三线并行收口：图像域契约、GenUI 蒸馏（回答即 UI）、Verifier 逐页缩略、子代理网络会话/恢复入口、CodeMirror 高亮、Mermaid strict、diff 渲染升级
- **v4.89 ~ v4.82** 上下文页与工具链：成本费率 hover、/context 弹层、Git 面板最小集、文件三态折叠、HTML 沙箱预览、工具结果缩略卡、上下文趋势跳转
- **v4.81 ~ v4.60** 早期面板期：上下文/文件/任务线（活动行级增量、任务页、记忆界面、子代理会话/并行/流式/追问闭环、办公板块交付验收 A2、better-sidebar pane 化）
