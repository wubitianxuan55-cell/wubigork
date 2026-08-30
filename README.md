# 🌍 gaea — 多功能 AI 助手

> gaea（盖亚）——你的通用办公与日常 AI 伙伴：可靠、清晰、有温度。
> 对话、轻语、小说、绘梦、模型引擎、工程办公、微信助手，多模块共生，统一品牌 V1.0.0。

![gaea](frontend/public/favicon.svg)

## 功能

- 💬 **对话** — 通用 AI 对话，流式输出，话题管理，本地存储
- 🌸 **轻语** — 可定制人格的陪伴型 AI（29 人格模板 + 情绪融合 + LLM 记忆管线）
- 📚 **小说创作** — 世界观/角色/大纲 Agent，章节流式创作，一键导出 TXT/MD/EPUB
- 🎨 **绘梦** — 文生图（ComfyUI：Flux / Z-Image-Turbo / Krea2），模型后端可切换
- ⚙️ **模型引擎** — 多供应商模型中心（xAI / DeepSeek / MiMo），OAuth 一键登录
- 🏗️ **通用办公** — 轻量工具集（17 个内置核心工具 + 技能扩展）：文档转换/图表生成/文档拼装、
  记忆与知识库、方案编写；开工前计划卡片 + 资料一键引用、对话内文件链接点击即预览、
  交付物卡片与会话产物面板、会话结束后自动「做梦」整理记忆；工作区全文搜索（轻量 RAG）、
  常用资料固定（新会话自动带入）、任务模板库（欢迎页一键发起 + slash 命令）
- 🧮 **造价数据库** — 一级板块（zaojia-database 蒸馏）：综合单价一级/人材机二级组成、
  测算项目与版本沉淀闭环、造价参考分位数对标、复盘笔记、《市政成本测算手册》整本导入、
  历史数据自愈；「数据库就是数据库」，测算/对标由办公 agent 工具承担
- 💬 **微信助手**（beta，个人使用）— 扫码绑定，ClawBot 远程对话；凭证失效不自动恢复
- 🔑 **SuperGrok OAuth** — PKCE 一键登录，token 持久化

## 技术栈

| 层 | 技术 |
|---|------|
| 桌面框架 | Wails v2 |
| 后端 | Go 1.26 |
| 前端 | React + TypeScript + Ant Design + Vite |
| AI 引擎 | xAI Grok (SuperGrok OAuth PKCE) + 多模型中心 |
| 图像 | ComfyUI (Flux / Z-Image-Turbo / Krea2) |

## 快速开始

```bash
# 安装前端依赖
cd frontend && npm install

# 开发模式（热加载）
wails dev

# 网页版对齐桌面端（HTTP 调试桥接）
设置 `GAEA_HTTP_PORT=8080` 后启动（`wails dev` 或生产 exe），Go 内核会额外
暴露 HTTP 桥接：浏览器访问 `http://localhost:5173`（Vite 已把 `/api` 代理到
桥接）即可驱动同一个内核——模型中心/聊天/记忆/办公引擎全部走真实数据与本地
模型，而不是前端 mock。事件经 `/api/stream` SSE 实时推送。

# 命令行登录
gaea login

# 生产构建（产物 build/bin/gaea.exe）
wails build
```

## 项目结构

```
gaea/
├── main.go                    # Wails 入口
├── wails.json                 # Wails 配置
├── internal/
│   ├── app/                   # App 绑定 + 各模块 handler
│   ├── ai/                    # AI 调用层（SSE 流式 + 图片）
│   ├── auth/                  # OAuth PKCE + token
│   ├── gaea/                  # 办公引擎（agent/tool/control/skill/boot/knowledge）
│   │   ├── cost/              # 成本库（条目/组成行/自愈）
│   │   ├── costproject/       # 测算项目（明细/版本/沉淀闭环）
│   │   └── costref/           # 造价参考（分位数指标/复盘笔记）
│   ├── whisper/               # 轻语模块（人格/记忆/分发）
│   ├── modelengine/           # 模型中心
│   ├── weixin/                # 微信助手
│   ├── project/               # 项目管理
│   ├── worldview/ character/ outline/ chapter/   # 小说 Agents
│   ├── export/                # TXT/MD/EPUB 导出
│   └── config/                # 全局配置（~/.gaea_config.json）
├── frontend/                  # React 前端
│   └── src/
│       ├── pages/             # 对话/轻语/小说/绘梦/模型中心/办公/知识库
│       ├── gaea/              # 办公板块（gaeaW 原生 UI）
│       └── stores/            # Zustand 状态
├── prompts/                   # RTCO Prompt 模板
├── skills/                    # 内置 Skill
└── docs/                      # 设计文档
```

## 版本

| 版本 | 日期 | 说明 |
|------|------|------|
| **v4.11.0** | 2026-08-30 | GLM 全模态纵深：生图后端接通（官方 images/generations，只发官方 schema 字段，URL 转 data URL，错误体原样透出）+ 官方双端点切换（std=按量/coding=编码套餐，SetGlmEndpoint 只收官方常量，绑定面 543→544）+ 生图模型目录补全（glm-image/cogview 系 18→22）+ glm-5-turbo 误分类修复。Go +15 测试、vitest 821/821、drift PASS（544）。详见 releases/v4.11.0.md |
| **v4.10.0** | 2026-08-30 | GLM 引擎上线（智谱 OpenAI 兼容 paas/v4：Bearer Key 全链路、chat ping 验证、官方文档锚定静态模型目录 glm-5.3 旗舰等 18 模型、地址防呆三防线）+ 做梦 2.0 蒸馏真实合并（确定性重复检测+可逆归档合并，路线图 T0 第一刀）+ 工作人设收口（professional 节奏豁免+办公秘书人格+[SPLIT] 出口净化+搜索触发收窄）+ 多跳因果链（≤2 跳「导致」链）+ Verifier 通道 A 引用级深化（opsJson 随卡+声明↔实况比对）+ Herdsman CLI 错误透明化。绑定面 540→543；vitest 818/818；drift PASS。详见 releases/v4.10.0.md |
| **v4.9.0** | 2026-08-30 | 星枢首页·轻语记忆纵深（基线 v4.8.3 + 15 提交，绑定面 535→540）：轻语记忆回放系列（GaeaWhisperEpisodeReplay 按情节重建原始对话 + GaeaWhisperMemoryRetell LLM 人格重述 + 时间锚点「重访那一天」写路径与纪念日回放）+ 图谱三维度（情绪着色/hermes.db V14、确定性因果三元组入图、event_chain 关联边琥珀虚线）+ 跨事实因果推断 GaeaWhisperCausalExplain「为什么」（确定性证据 + 人格口吻 LLM 人话化，无证据零 LLM 诚实回退）+ 首页重构「星枢指挥所」（启动默认首页 + 两段式启动动画 BootSplash + 命令条/遥测细条/Bento 能力矩阵，i18n 三语 17 键）+ 工程化（build.bat 冒烟自动化、持久化原子写、XlsxPreview 行虚拟滚动、VoiceStart realtime 门、微信识图两修）。Go 全量绿、vitest 818/818、tsc/eslint 0、drift PASS（540）。详见 releases/v4.9.0.md |
| **v4.8.3** | 2026-08-30 | 微信图片双向（真协议实装，三方印证：本机抓包解密+hermes-agent+openilink SDK）：出图回推 getuploadurl→AES-128-ECB→CDN 密文上传→image_item 卡片+caption 补发（真机 delivered）+ 发图识别 type=2/aeskey 解密下载→魔数终审落盘（真机两连发通过）+ 识图模型升级多模态 Qwen3.6-35B 优先（手写体强，与聊天同体零额外显存；PaddleOCR→MinerU→OvisOCR2 降兜底）+ 身份类问题跳过联网搜索（乱字母根因）+ 关键坑锁死（type=2/aes_key=base64(hex)/上传域与扫码 baseurl 无关）。Go 全量绿、零新增绑定（535）、前端零改动。详见 releases/v4.8.3.md |
| **v4.8.2** | 2026-08-30 | 欠账收尾：权限升级请求（request_permission 工具 + 硬纪律三闸——deny 先行/hardAsk 拒升级/批准只写 glob 规则表不绕闸门 + 审批卡 request 形态 reason 原文块 + 会话 glob 规则表补全，零新增绑定）+ 竞态/flake 全治理（Cancel 被 succeeded 吞掉的收尾窗生产竞态 ×10 压力绿、stubGate 测试桩加锁、filewatch 风暴测试时序根治、ProgrammingPage 显式 5s 超时）+ Realtime S2 事件环骨架（Resample16kTo24k 纯函数 + 事件常量 +7 + TurnControl 可选接口 + voice_manager 事件泵/barge-in 三联/24k WAV 冲洗 + 前端 PCM 推送死门打通 + 五重降级护栏，真机欠账如实记账）。Go 全量绿、vitest 807/807、tsc/eslint 0、drift PASS。详见 releases/v4.8.2.md |
| **v4.8.1** | 2026-08-30 | 欠账清尾：全局离线模式设置 UI（SecurityPanel 总闸段 + GaeaGet/SetOfflineMode 绑定 533→535 + shelf 内存同步；与敏感域/办公本地优先的叠加关系文案化）+ Realtime S1（realtime 三键落盘——provider 仅 openai、Key 走 DPAPI 密文仅本机可解；initVoice 注入接线位置测试守护；VoiceApplySettings/GetSettings 扩三键——明文 Key 永不出后端，测试断言无泄漏；VoiceSettingsPanel「实时语音（实验）」入口段：供应商/模型/密码框 Key/保存回读 hasKey）。Go 全量绿、vitest 803/803、tsc/eslint 0、drift PASS。详见 releases/v4.8.1.md |
| **v4.8.0** | 2026-08-30 | 全面铺开·触点纵深（六线并行调研→七刀，多子代理分工）：读屏纵深（多显示器 Monitors/CaptureArea + 「第N屏/主屏/副屏」序数解析 + OCR 本地摘要朗读 + 截图留档）+ intent LLM 兜底分类器（默认关：白名单 navigate/status/read_screen + 0.75 置信门 + 2s 硬超时，dryRun 恒不调用，宁漏勿误）+ 生图产物 CardPath 接通（微信回推数据源）+ iLink 微信通道离线收敛（限频/4KB 截断/多媒体上限/SSRF+魔数下载防线/图片→vision 识别管线/防御解析矩阵/SendFileCard seam）+ 全局离线模式总开关（EngineType.IsLocal + routeModel 三步云过滤，跨版欠账清账）+ 成本知识图谱可视化（costref.BuildGraph 树聚合/条目展开双视角 + CostGraphView 零依赖 SVG，成本库第 8 模块，绑定面 532→533）+ 实时语音 Realtime S0 铺底（internal/realtime seam + VoiceHealth realtimeReady，S1/S2 留欠账）。Go 全量绿、vitest 800/800（146 文件）、tsc/eslint 0。详见 releases/v4.8.0.md |
| **v4.7.0** | 2026-08-30 | 命令面板接内核·读屏（S4.6 完整收口）：GaeaRouteIntent(text,dryRun) 绑定（531→532；dry-run 预览-确认制——搜索词不是整句指令入口，命中出卡点执行才真跑，宁漏勿误）+ SearchModal 指令预览卡（动作标签+预览语+回执内联，导航类复用 INTENT_NAVIGATE 切板块收面板）+ 真·Ctrl+K（MainLayout 全局快捷键，办公板块让位工作台自有 CommandPalette）+ 屏幕感知能力「读一下屏幕」（read_screen 窄规则 + 截屏→临时 PNG→既有 OCR 链→300 字截断回传，语音 TTS 朗读/面板内联/微信回推三入口免费受益，失败诚实回执）。Go 全量绿、vitest 796/796（145 文件）、tsc/eslint 0。详见 releases/v4.7.0.md |
| **v4.6.1** | 2026-08-30 | 微信统一路由·规范包机制·归因对标：微信消息接统一意图路由（routeIntentWithResult，提醒之外生图/状态/导航全命中）+ iLink 图片消息协议第一刀（image_item 防御解析 + 发图提示行）；规范包机制化（Checker 注册表 + GB/T 9704 红头 + 造价工程表式双检查器，OfficePanel 按规范包分组）；成本归因对标（项目明细 vs 参考指标 P25/P75 带宽，差幅等级/贡献金额/主因 TopDrivers，参考池排除自身防自对标；绑定面 530→531，FiveCalcPanel 归因区）。Go 全量绿、vitest 791/791、tsc/eslint 0。详见 releases/v4.6.1.md |
| **v4.6.0** | 2026-08-30 | 执行审计后第一轮补课·双空间收尾+纵深：红线三条全接线（记忆注入 InSpace 视图收窄——work 只注入 work/play 只注入 play；[tasks] 配置段启用按空间分账 {work=1, play=1}+价格抓取优先；onTaskEvent 订阅层空间过滤推广）；前端治理收尾（keepAlive 裸轮询 8 处门控 + CSS 真硬编码 token 化）；纵深三件（Mood→TTS 连续韵律闭环——「听得出她今天低落」；Verifier 通道 B 真视觉 diff（soffice→pdftoppm→像素差异率）+ 失败回 Plan（xlsx 一键重新规划）；询价异常检测分级+线性回归价格预测+OCR 报价单自动幂等入询价库）。绑定面不变（变参兼容）；Go 全量绿、vitest 791/791、tsc/eslint 0。详见 releases/v4.6.0.md |
| **v4.5.0** | 2026-08-30 | 规划修订(§10.4a)后第一刀·指令中枢：统一「意图→能力→结果回传」路由内核落地触点层「同内核多入口」——intent 纯函数解析包(导航/生图/状态/提醒，宁漏勿误纪律) + 能力执行层(navigate 事件/generate_image/status/reminder 复用离线代办) + 语音指令通路(JARVIS 一档：voice 回调分流，命中即能力执行经 TTS 播报，voice 包零改动) + 前端 INTENT_NAVIGATE → navigateBoard 自动切空间；绑定面 530 零新增；Go 108/108、vitest 789/789。详见 releases/v4.5.0.md |
| **v4.4.0** | 2026-08-30 | v4.4「触点」一期·微信遥控器：离线代办——微信对助手说「提醒我 30分钟后 喝水 / 明天早上9点 开会」→ 中文时间解析（相对/日期+段词/裸时刻，中文数字进位）建提醒 → 20s ticker 到点经微信 Push 回推（失败重试 ≤5 次，JSON 持久化重启恢复）；weixin.Server 主动推送通路（最近活跃会话记忆）；WeixinPage 书房板块页落地（扫码绑定流/通道状态/提醒面板，板块 inMenu=true 进 rail 与首页左翼）；绑定面 525→530（WhisperWeixin*/WhisperAssistant* 转正 + WeixinReminder* 新增），spaceBindings 235→247 全归 work；vitest 789/789、Go 107/107。详见 releases/v4.4.0.md |
| **v4.3.2** | 2026-08-30 | 首页「双翼·中庭」重构 + 空间导航收敛：中庭语音+打字一体对话条（VoiceChatText 共用管道 + 放大 orb 磁吸核心，hero 让位细眉）；左翼「书房」2×2 格（办公/造价/记忆/模型）；右翼「庭院」纵向列表（聊天/小说/绘梦/角色）；门廊编程独立窗口入口；命名 工位→书房、乐园→庭院；移除 rail 空间切换按钮，navigateBoard 按板块自动切空间；搜索 scope 文案同步。绑定面不变、vitest 789/789、Go 全量绿。详见 releases/v4.3.2.md |
| **v4.3.1** | 2026-08-30 | v4.3 后续小步：主动关心定时推送频控（ticker 四信号：频控/作息尊重/生日祝福/门控合成 → gaea-whisper-proactive 事件 + GaeaWhisperProactiveConfig 配置绑定）+ 创作间世界模型面板（设定页维度化编辑器 6 维度卡片 + 伏笔登记表状态流转/回收率 + 一致性检查三类告警）+ 角色参考图（SchemaV2 reference/gallery 两列 + CharacterGeneratePortraitWithRef img2img 参考槽 + 前端参考图管理）+ 朗读情绪 UI（9 情绪选择器 + 会话情绪跟随 + TTSSpeakBase64WithParams）。绑定面 525、spaceBindings 235、vitest 789/789、Go 118/118。详见 releases/v4.3.1.md |
| **v4.3.0** | 2026-08-29 | v4.3 乐园做深（阶段 3+ 第二发）：会客厅关系记忆（三表持久化闭环 + ReseedAssociationGraph + QuerySubgraph 多跳子图召回 + WhisperGraphPanel SVG 图谱）+ 主动关心（GaeaWhisperProactiveNow 评估 + 前端「轻语先开口」）+ 情感语音（TTS SynthesizeWithParams 参数扩展 + cosyvoice 修复 + 情绪→参数映射 + 长期心境维 Mood）+ 创作间图文联动（章节配图复活死绑定 + GaeaGenerateBookCover 3:4 书封落 play exports）。绑定面 522、spaceBindings 233、vitest 769/769。详见 releases/v4.3.0.md |
| **v4.2.0** | 2026-08-29 | v4.2 造价 AI 化（阶段 3+ 领域包第一发）：AI 组价（PriceBand 价格带 P25-P75+置信度+证据链 → GaeaCostCompose 相似检索+LLM 人材机拆解 → 一键回写，前端 ComposeModal）+ 询价飞轮（costinquiry 四源归一数据点 + 到期预警 + 调差建议 + 前端询价视图）+ 五算对比（coststage 估/概/预/结/决阶段值 + 对比/偏差三档 + 前端 FiveCalcPanel）。绑定面 517、spaceBindings 229、vitest 759/759。详见 releases/v4.2.0.md |
| **v3.9.0** | 2026-08-29 | 双空间壳（阶段 2 收官：两视图+空间切换持久化+双首页+删旧 10 板块导航+bridge 三门面+types 全量迁移+151 hex token 化+i18n 决策+页面迁入 P1 对话流）+ v4.1 办公信任链（证据链 ChangeRecord/Journal + Verifier 双通道复核 + 基线回滚冲突保护 + GB/T 9704 红头规范体检）。绑定面 506、spaceBindings 218、vitest 738/738。详见 releases/v3.9.0.md |
| **v3.8.0** | 2026-08-29 | 双空间内核（工位/乐园隔离：会话/记忆/任务/产物/模型/工具/权限/护栏全按空间装配，space.mode 可回退）+ 质量地基（并发加固/Registry 锁/gate 原子化/retry_until 门控/edit_file 工具层/前端虚拟化与轮询门控/CI -race）+ 长期规划定稿（编程板块保持独立 DSH 窗口）。绑定面 502、Go 115 包、vitest 全绿。详见 releases/v3.8.0.md |
| **v3.7.0** | 2026-08-29 | 办公蒸馏 codex 清单收官：C2 记忆引用可追溯（注入行引用键 [MEM:name] + 回传解析 Touch + 前端引用徽标弹层展示记忆详情/沉淀来源）；C4 审批决策族（deny/abort 拒绝三分 + approval_timeout_secs 超时 + persist_allow「始终允许」回写 [permissions].allow 策略文件，hardAsk 降级不回写，GaeaApprove 重构决策串五值）；C9 任务输出事件化（gaea-task 事件推输出尾回放，dock 事件即推、轮询兜底）；C5 运行状态行窗口占用百分比 + 压缩前预警（75%/90% 两档）；C6 项目说明文件 32KB 预算 + .gaea/AGENTS.md 发现；C3 自动做梦 no-op 指纹去重。绑定面 499、vitest 681/681、Go 114/114。详见 releases/v3.7.0.md |
| **v3.6.0** | 2026-08-29 | 办公文件编辑审阅制 + 本地优先 + 对话面减负：xlsx AI 编辑两段式（GaeaXlsxPlanEdit 临时副本试运行+单元格级 diff → 批准 → GaeaXlsxApplyEdit，新增 set_style 叠加/合并/列宽）；xlsx 原生图表嵌入工作簿（非截图）；PDF 统一出口（GaeaConvertToPdf，LibreOffice 无头 + md 经 docx 中转 + 顶栏导出）；办公功能级 AI 本地优先（routeOfficeLocal + 安全设置开关，主 agent 不受影响）；运行中插话（GaeaSteer + event.Steer）；回退方案模式 v1、撤下任务目标/验收清单（GoalCard + GaeaRequirement 系）、用户消息 Codex 式收敛 + 超长折叠；修复 whisper 关机排水丢任务 / 空切片 null 崩溃。绑定面 499、vitest 669/669、Go 114/114。详见 releases/v3.6.0.md |
| **v3.5.0** | 2026-08-28 | 办公对话区标签页 + dsh-context Go 移植：对话窗口上方 [对话\|轨迹\|上下文] 三标签；request_header 事件（模型可见必入日志的请求头落点）；上下文标签（contextview 包：六分类组成 + 原生 SVG 趋势图 + 事件流 + 步骤详情，usage 锚定与顶栏同源）；轨迹标签（trajectory 包对齐 DSH ui-trajectory 扁平事件账本：header change/工具 parentId 嵌套/轮间压缩/ask+approval + 检查器/搜索/统计）；Agent 网络（子代理树 + 节点 token 环 + subagents meta 富化）。绑定面 503、vitest 668/668、Go 114/114。详见 releases/v3.5.0.md |
| **v3.4.0** | 2026-08-27 | 记忆统一层第一刀：统一检索后端收口（GaeaUnifiedSearch 增三脑/文件语义两组，hub 搜索 4 绑定前端拼装→1 绑定）+ 生命周期产品化（归档 tab 分页修复 + Unarchive 恢复 + retentionDays 展示）+ 修复漂移脚本单条差异静默放行 bug。eslint 0/0、tsc 0、vitest 654、Go 112/112、绑定面 498。详见 releases/v3.4.0.md |
| **v3.3.0** | 2026-08-27 | 质量收敛：eslint 存量 warnings 366→0（配置显式化 `^_` 前缀/空 catch/常量导出 + 死代码清理 56 处 + exhaustive-deps 40 处含两处 TDZ 重排 + 混合导出显式声明 + 冗余 @ts-ignore 移除）；flaky 治理（filewatch 超时 3s→5s + CI 测试失败重试）；releases/README.md 历史乱码重建；前端性能体检。eslint 0/0、tsc 0、vitest 652、Go 112/112、绑定面 497。详见 releases/v3.3.0.md |
| **v3.2.1** | 2026-08-26 | 工作区内联编辑（C5，蒸馏清单收官）：GaeaWriteFile（相对路径/写根/文本白名单/大小四重校验 + 原子写）+ 文件预览编辑模式（脏标记/Ctrl+S/保存状态机/保存后重读）。vitest 652、绑定面 497 方法。详见 releases/v3.2.1.md |
| **v3.2.0** | 2026-08-26 | 任务可见性：C1 任务实时输出（Progress.Output 环形缓冲 + GaeaTaskOutput + 任务中心输出 dock 2s 轮询尾随滚动 + stopping 结束态细分）+ C2 子代理活动行（lastText/lastTool「正在…」实时线）+ 数据备份计划路径统一修复。vitest 647、绑定面 496 方法。详见 releases/v3.2.0.md |
| **v3.1.1** | 2026-08-26 | 造价数据库闭环补齐：测算项目 UI（明细编辑/版本快照/恢复/沉淀回库）+ 造价参考分位数对标 + 复盘笔记（后端 v3.1.0 已就绪，纯前端 + 导航同步）+ C4 选区转对话（选中正文→转为提问插入输入框）+ 仓储卫生（删根目录临时脚本/旧 exe、releases 索引补全）。vitest 643、绑定面 495 方法零新增。详见 releases/v3.1.1.md |
| **v3.1.0** | 2026-08-26 | 造价数据库一级板块（综合单价架构 + 测算项目/造价参考/复盘笔记 + 手册整本导入 + 数据自愈）+ 办公蒸馏（右侧面板会话持久化/活动角标/预览队列 chip、文件工作台资源管理器）+ 办公初始化死锁修复；绑定面 495 方法、vitest 630。详见 releases/v3.1.0.md |
| **v3.0.8** | 2026-08-17 | 办公板块「表格可交付 + 会话产物打包 + 多智能体分工」+ 界面收敛：表格选中区域→一键图表（柱/线/饼，图表 ▾ 菜单）+ 会话产物一键打包 Zip + 产物缩略图增强（xlsx 迷你表格/md 文本摘要）+ 子代理「分工」面板（状态/任务摘要/回答）；右侧面板 Tab 收敛为 4 主标签（文件/成果/运行/分析）+ Excel 工具栏按上下文收敛（用户决策：不堆功能、聚焦 Word/Excel）。vitest 605 通过、绑定面 480 方法。详见 releases/v3.0.8.md |
| **v3.0.7** | 2026-08-17 | 办公板块文件交互体验（调研 P0-P2 落地）：非图片附件 chip 化 + 行内文件 chip 视觉统一 + 最近文件快捷区 + 多文件预览队列（←/→ 导航）+ 产物版本时间线（vN 徽标）+ 大工具输出有界预览 + 附件上下文占用透明化；内置 prompt 模板兜底（SetPromptFS，exe 单文件分发）。vitest 587 通过。详见 releases/v3.0.7.md |
| **v3.0.6** | 2026-08-16 | 编程板块桌面内嵌工作台（iframe 内嵌 DeepSeek Harness Web + 启动引导：前置条件检查/日志/启动动画）+ 运行中工具栏移入顶栏（仅编程板块显示）；办公板块会话回退/分叉/回退点 + 右侧 Tab 清单化 + 会话统计回填 + mock 场景补全。详见 releases/v3.0.6.md |
| **v3.0.5** | 2026-08-16 | 首页任务指挥中心改版（参照 DeepSeek 首页风格）：Hero 左文右卡 + AI 状态细条 + 「全部模块」办公大卡 + 8 卡 4×2 网格；修复设置卡被遮挡、PageRegistry 补注册编程板块；语音晶核取消粒子星云恢复发光球。详见 releases/v3.0.5.md |
| **v3.0.4** | 2026-08-16 | 办公板块能力加强（任务目标→验收清单 + 目标/待办卡拆分 + 自动追踪）+ 小说阅读体验重构（排版/书签/划线/AI 伴读/朗读同步/全文搜索/导入成品小说）+ 角色库闭环 + 导出合并进阅读面板。详见 releases/v3.0.4.md |
| **v3.0.3** | 2026-08-16 | 小说（设定页「应用到设定」/创作面板字号/默认 story-deslop/剧情构思后台化）+ 模型中心（统计与资源/AMD 显存识别/失败可见+重试）+ 绘梦模板库 19 类 231 个 + ComfyUI 实时进度 + 角色剧照远程 URL 本地化。详见 releases/v3.0.3.md |
| **v3.0.2** | 2026-08-16 | 移除运行态看门狗（反复误杀办公板块回合，用户决策）；模型流空闲超时与回合 panic 恢复保留兜底。详见 releases/v3.0.2.md |
| **v3.0.1** | 2026-08-15 | 小说板块 UX/UI 重构：书架卡片化（封面渐变条/阅读进度条/搜索排序/继续阅读联动）、阅读页单条 chrome + 沉浸阅读模式（居中限宽衬线排版）、AI 控制台 v3 玻璃面板化、创作页令牌化与生成发光统一、全板块零硬编码色值；修复 app_info.go 版本漂移（2.40.0→3.0.1 三处统一）。详见 releases/v3.0.1.md |
| **v3.0.0** | 2026-08-15 | 星枢 Constellation OS · UI 革命性重设计（V3.0 首发）：壳层革命（左侧指挥轨道 + 顶部轨道条 + 底部遥测轨道）；10 板块统一 3 分区工作台；Luminous Glass 2.0 令牌体系；发布前打磨（聊天模式条移入顶栏、输入框两级重设计、首页全屏适配）+ 7 个 CSS 文件注释 `*/` 根因修复（面板「只有上半截」）。详见 releases/v3.0.0.md |
| **v2.40.0** | 2026-08-15 | 3.0 架构主线 Wave 4「Step 3 收官」：semantic_search 工具注册（死代码恢复为办公 agent 可用工具）+ BalanceKind 从 ProviderEntry 贯通（`balance_kind` 配置项，切换余额后端只改配置，未知 kind fail-closed）+ ModuleLauncher 清单化（subscribeBoards 订阅活动清单，后端并入 knowledge 后首页启动器自动多出「知识库」卡） |
| **v2.39.0** | 2026-08-15 | 3.0 架构主线 Wave 3：Step 3b LLM Seam（LLMProvider + NewLLM 配置驱动）+ Step 3c OCR/ASR/TTS Seam（三类注册表 + GAEA_OCR_ENGINE 驱动）+ Step 3d 分类单源化与 8 处硬编码注册表化（websearch/embed/rerank/vision/markitdown/billing）+ 前端 GetBoardManifests 接线（normalize 差集 + KnowledgePage）+ gaea.toml 新配置段 |
| **v2.38.0** | 2026-08-15 | 3.0 架构主线 Wave 2：Step 1 app 层接线（会话事件日志「日志即真相」运行时闭环：Resume→Restore / Save→日志 / 模型调用前 fail-closed 检查点）+ Step 2 板块 Manifest（board 包 10 板块 + module_registry manifest 驱动 + GetBoardManifests + MainLayout 附 B 12 硬编码点清单化 + PageRegistry + events 常量表 + label 单一来源）+ Step 3a Image Seam（图片后端注册表化：openai/comfyui 自注册 + 401 单次重试守卫）。绑定面 464 方法 |
| **v2.37.0** | 2026-08-15 | 正确性纵深收官：T7-2 可见性收口 / T7-3 名实相符 / T7-4 前端性能收尾 + 3.0 Step 0 修债 + Step 1 会话事件日志（append-only 日志 + 投影 + checkpoint + 迁移 + GaeaHistory 黄金测试） |
| **v2.34.0** | 2026-08-15 | 正确性纵深第一刀：轻语会话并发安全（深拷贝/-race 实证修复）+ 任务调度器竞态 + TCCA 指标聚合收敛 + AI 客户端状态与重试 |
| **v2.33.0** | 2026-08-14 | 阶段 6「质量收敛」第十刀·前端收敛：巨型文件拆分（ChatPage 1022→370 / ImageGenPage 911→310 / CapabilitiesPanel 803→178 / Composer 786→406 / mock.ts 1563→50 按域拆 10 文件，11 no-op 落实）；any 清零（no-explicit-any 升 error 进 CI，315→0）；绑定漂移检查恢复（gen_bindings -names + bindingNames.ts + bridge.ts 双向类型守卫 + CI 步骤）；mock 契约对齐（RetrievalEvalRun 12 条真实查询集）；Sidebar react-window 虚拟滚动 + CostLibraryView memo 化 + useDebouncedValue；删除 api/bridge.ts 双桥接合流；测试 354→361 |
| **v2.32.0** | 2026-08-14 | 阶段 6「质量收敛」第九刀·辅助合集名实相符：微信生命周期（Stop 幂等/Start 重启/过期钩子退出空转）+ wxToken DPAPI 加密迁移 + 4 张死表 V13 删除；OCR 兜底名实相符（超时杀进程树 + 单图 tesseract 降级 + 删「Windows 原生 OCR」文案）；配置原子写（fsync+rename）+ 损坏备份恢复；CosyVoice 路径/端口可配置 + 启动退避重试；token 改 header（服务端去 query 兜底 + 前端 fetch 流式 SSE） |
| **v2.31.0** | 2026-08-14 | 阶段 6「质量收敛」第八刀·记忆生命周期与审计：dream 写入审计（决策成文 DREAM_WRITE_POLICY.md，SaveDreamFacts 带 source 落 dream-audit.jsonl）；facts 归档生命周期（90 天保留硬删 CleanupArchived + ListArchivedPaged 分页 + 新绑定 GaeaMemoryCleanupArchived/GaeaMemoryArchivedList + 前端清理按钮 + purge-audit 溯源审计）；索引截断按边界（4096 字节口径统一 + 行边界 + markdown 链接保护，6 测试）；GraphView/WhisperMemoryLibrary 补测 13 用例 |
| **v2.30.0** | 2026-08-14 | 阶段 6「质量收敛」第七刀·小说导出与原子性：export 整改（失败计数/作者元数据/统一分段器/世界观对齐/保留名消毒，13 测试）；生成中断（CancelCreateChapter 绑定 + 章节互斥 + 取消保留部分正文）；writeFileAtomic 落盘原子化；模板 {word_count} 占位符；CreatePage 791→288 行 + 停止按钮 + 判别联合（18 vitest） |
| **v2.29.0** | 2026-08-14 | 阶段 6「质量收敛」第六刀·模型中心密钥与 UI：refresh_token DPAPI 加密（旧明文自动迁移）；汇率配置化（usd_cny_rate 键 + 新绑定 + ModelPanel 输入）；probe 告警文案修复；ModelCenterPage 顶层 useState 42→3（5 hooks 下沉）+ XAI_VOICES 单源；refreshLocalModels 竞态守卫 + 定时器随分类重置 |
| **v2.28.0** | 2026-08-14 | 阶段 6「质量收敛」第五刀·轻语测试与可观测：补测试 146 用例（emotion_fusion 100% 覆盖 + 记忆管线 ≥92%）；异步记忆写错误可观测（WriteErrors 计数 + 四类 phase）；成人模式决策成文（ADULT_MODE.md + 删除死接口 WhisperSetAdultMode）；GetDatabase 签名收敛 (db, err) + PRAGMA 单源 + FTS 失败日志；陈旧占位清理 4 文件 |
| **v2.27.0** | 2026-08-14 | 阶段 6「质量收敛」第四刀·绘梦链路真实生效：取消真实中断 ComfyUI（/interrupt + 本地取消标记）；flux 真实工作流（映射表 + 静默降级消除）；历史图片 file_path 恢复（分级存储 + 回填）；尺寸严格校验（64-2048）；端口命令注入修复（netstat 参数数组）；核心链路 httptest 31 用例 |
| **v2.26.0** | 2026-08-14 | 阶段 6「质量收敛」第三刀·对话流可靠：流订阅竞态修复（订阅先行 + 30s 无帧超时 + sending 五路复位）；落库错误透传（ChatTopicsList/ChatMessagesList 返回 error + 流式 error 终态）；语音消息持久化（新绑定 ChatAppendMessages 单事务）；旧数据迁移一次性化；导出 Markdown 转义 + 文件名消毒；导入事务化 |
| **v2.25.0** | 2026-08-14 | 阶段 6「质量收敛」第二刀·办公引擎正确性：PDF 页数统计/分页过滤修复（/Type /Pages 干扰 + OCR 绝对页码）；TurnResult 语义收紧（blocked/suppressed 计入）；落地后端看门狗（墙钟 10min/停滞 30s，工具执行与审批等待豁免）；运行中 Send 限长队列（8 条 FIFO 排空）；子代理禁写清单注册表化（PersistWrite 标记推导）；TCCA 97%/evidence 91% 覆盖率；docmd.go 1521 行拆 4 文件 |
| **v2.24.0** | 2026-08-14 | 阶段 6「质量收敛」第一刀·基础层可靠性：SSE 流式加固（64KB 行上限消除/连接与 5xx 退避重试/60s 空闲超时）；AI 流量接入系统代理（localhost 强制直连）；前端错误可见性（BridgeError 归一化 + 14 处静默 catch 改记录）；后端吞错清理 + 桥接 token 日志脱敏 |
| **v2.23.0** | 2026-08-14 | 阶段 5「运行纵深」第三刀·进料与质量：成本库 PDF/图片报价单本地识别入表（OCR+本地视觉归一化，sensitive_local 不出机）+ 供应商比价卡；统一检索入口（关键词+跨库语义单框两组）+ 检索质量受控测评（Recall@10≥0.8 门槛，模型中心「检索质量」区） |
| **v2.22.0** | 2026-08-14 | 阶段 5「运行纵深」第二刀·速度与韧性：本地模型调度纵深（保活 keep-warm 轻量探针防卸载、启动自动预载、换模预计等待提示、KV 缓存命中率并入调用统计）；中断续跑（会话「未完成」徽标 + 恢复注入中断摘要、轻语任务计划持久化与恢复入口） |
| **v2.21.0** | 2026-08-14 | 阶段 5「运行纵深」第一刀·调度与异步化：通用任务调度器（持久任务表/进度事件/取消/自动重试/手动重试/重启续跑，价格抓取与文件索引全异步化 + 办公右栏「任务中心」Tab）；fsnotify 实时文件监听（新文件秒级进语义索引，10 分钟轮询降级兜底） |
| **v2.20.1** | 2026-08-14 | 数据可迁移独立审查修复（3 高危+4 中危）：恢复重试幂等、home 配置恢复、SQLite 快照 busy_timeout + checkpoint 回退、失败可见、已有 pending 拒绝、dirSize 缓存、Rollback 回滚 |
| **v2.20.0** | 2026-08-14 | 个人使用收口（不商用）：设置「数据」一键备份/恢复（Hephaestus.db + whisper_data + 配置 + sessions → zip + manifest；SQLite VACUUM INTO 一致性快照；两阶段恢复重启生效、恢复前自动备份当前数据）；微信通道 beta 标注、移动端冻结；发布形态简化（去安装器/签名/自动更新） |
| **v2.19.0** | 2026-08-14 | 阶段 3 第二刀（D3-4 补测评缺口）：测评报告专项分析（每模型对比/长上下文/缓存复用/显存参数/并发）；压力专项任务预设；快速流式探针（SSE 断流/卡顿观察） |
| **v2.18.0** | 2026-08-14 | 长期规划阶段 3 首轮：跨库统一语义检索补齐「资料」+ 索引状态；模型中心「本地 vs 云端」分流统计与节省对比；Herdsman 受控测评产品化（一键发起/逐 case 明细/Markdown 报告导出） |
| **v2.17.0** | 2026-08-14 | 长期规划阶段 2·安全与架构收敛：LAN 暴露全局告警横幅 + 设置「安全」面板；WebView2 远程调试默认关闭（GAEA_WEBVIEW_DEBUG 开关）；HTTP 桥接一次性 token 鉴权；敏感域本地化开关（成本/报价 AI 默认本地 Herdsman）；429 个导出方法按板块拆 10 个绑定门面（脚本生成 + 完备性测试兜底） |
| **v2.16.1** | 2026-08-14 | E1-4 模型中心资源协同：生命周期操作串行化（对齐 herdsman local_concurrency=1）+ 模型库磁盘 KPI（已装占用/卷余量）+ fmtSize TB 档 |
| **v2.16.0** | 2026-08-14 | 长期规划首轮：Herdsman 底座加固（环境探测/健康检查/TTS 默认动态解析/LAN 暴露告警/模型用途提示/思考模式 token 守护）+ 工程门禁（前端 CI 修复、发布冒烟脚本、周版本节奏） |
| **v2.15.7** | 2026-08-13 | 通用办公 P0·开工前计划卡片结构化：计划改出严格 JSON，后端解析为「任务理解/步骤（资料·工具·产出物）/待确认」，Ask 卡片渲染专属计划卡片，解析失败自动回退纯文本；测试 241→243 |
| **v2.15.6** | 2026-08-13 | Herdsman 深挖 P5·数字生命联动：记忆中枢新增「数字生命」库（角色/关系/记忆摘要/时间线/世界事件，只读）；最近 Herdsman 操作列表；路线图六阶段收官；测试 239→241 |
| **v2.15.5** | 2026-08-13 | Herdsman 深挖 P4·检索升级 + 调用统计：语义检索动态切 qwen3-embedding-4b / qwen3-reranker-4b（未装回退 bge）；模型中心新增 Herdsman 本地调用统计面板（model_stats 聚合）；两个模型已下载启动 |
| **v2.15.4** | 2026-08-13 | Herdsman 深挖 P3·本地翻译：优先 Hunyuan-MT / Hy-MT 翻译模型（未安装回退办公模型）；新增 translate_text 办公专业工具；能力面板入口；Go +6 用例 |
| **v2.15.3** | 2026-08-13 | Herdsman 深挖 P2·模型生命周期：模型库卡片直接启动/停止/下载/卸载 Herdsman 模型；读 launch_records 生成「启动预设」徽标；测试 238→239 |
| **v2.15.2** | 2026-08-13 | Herdsman 深挖 P1·模型库：模型中心新增「模型库」分类，接入 Herdsman 完整 90 模型目录（能力/安装/运行/量化/变体/大小），可搜索过滤；后端 HerdsmanModelCatalog（CLI RPC）+ 测试 234→238 |
| **v2.15.1** | 2026-08-13 | 通用办公·产物与资料体验收口：会话产物面板图片缩略图 + 一键复制全部路径；Ctrl+K 新增资料/产物/变更跳转；欢迎页任务模板内置兜底（离线/空库不再空白）；前端测试 229→234 |
| **v2.15.0** | 2026-08-13 | 模型中心 P0/P1/P2 + UI 重设计（引擎状态联动/连接诊断/功能绑定回退态、搜索收藏置顶/资源占用可视化/选择器统一、批量启停/统计抽屉；浅色深色双主题） |
| **v2.14.12** | 2026-08-13 | 绘梦 UI 重构落地（可折叠分区/底部生成栏/任务中心三 Tab/玻璃 HUD）+ 选项改下拉并修复 WebView2 弹层；herdsman 生图修复（size 契约/URL 转 data/图生图接入） |
| **v2.14.11** | 2026-08-13 | 小说板块后端（章节流式/状态/书架/世界观/统计/导出）+ 绘梦生成链路闭环（队列/取消、历史元数据持久化、ComfyUI 进度、模板/绘照、LoRA 重试）；绘梦 UI 重构已立项 |
| **v2.14.10** | 2026-08-13 | 修复办公模型改绑不生效：运行时改绑「办公」后同步重注入 bridge 并重建引擎，不再沿用旧模型 |
| **v2.14.9** | 2026-08-13 | 聊天板块后端：用户/助手消息原子落库，AppendMessage 用 RETURNING 收敛 seq 分配 |
| **v2.14.8** | 2026-08-13 | 聊天板块后端：会话列表预览 N+1 收敛为单条子查询，新增 GetTopic 供创建/导入/导出按 ID 读取 |
| **v2.14.7** | 2026-08-13 | 聊天板块交互收尾：清空对话二次确认、切换话题自动聚焦输入框、标题生成纯函数化；前端测试 194→196 |
| **v2.14.6** | 2026-08-13 | 聊天板块会话导出为 Markdown（落盘用户数据目录，前端一键导出并复制路径） |
| **v2.14.5** | 2026-08-13 | 聊天板块普通对话真实流式输出（后端逐块下发 delta/reasoning/done/error），替换前端模拟打字流 |
| **v2.14.4** | 2026-08-13 | 聊天板块收口：联网搜索不再污染用户历史、上翻「回到底部」、侧栏预览随发送/清空同步 |
| **v2.14.3** | 2026-08-13 | 聊天板块补强：中文输入法 Enter 防误发、快速切话题竞态修复、模式栏收敛；前端测试 190→194 |
| **v2.14.2** | 2026-08-13 | 聊天板块优化：会话最近活跃排序 + 相对时间、侧栏会话搜索、生成中智能滚动（上翻不吸底）、重开聊天正确载入历史消息；前端测试 182→190 |
| **v2.14.1** | 2026-08-13 | 办公板块缺陷收口（会话失败提示/归档删除确认/跨项目最近会话/变更面板汇总排序/归档搜索）+ 前端测试 138→179 + 首轮结构收敛 |
| **v2.14.0** | 2026-08-13 | 办公板块会话化升级：项目分组 + 会话置顶/归档/恢复 + 任务目标（需求→验收）+ 文件变更面板 + 专注模式；修复记忆图谱三元组被成本节点挤掉、会话删除注册表清理不完整、App 层 toast 静默失效 |
| **v2.13.22** | 2026-08-12 | 修复整轮结束后大过程卡折叠：合并大卡独立 key 全新挂载（默认展开），小过程卡实例始终折叠不误展开 |
| **v2.13.21** | 2026-08-12 | 办公板块安全审计：子代理不再继承持久化写入工具（封堵 headless 通道绕过审批），forget/install_skill 纳入硬性确认 |
| **v2.13.20** | 2026-08-12 | 记忆/知识库写入强制确认（remember/knowledge_add/promote_session_facts 与 cost_save 同规则）；记忆索引注入限 3000 runes，控制上下文占用 |
| **v2.13.19** | 2026-08-12 | 成本库写入强制逐条确认：cost_save 任何权限级别（含 yolo）都弹审批卡，显示条目/单价/规格/来源，批准仅本次生效，杜绝 AI 直接入库 |
| **v2.13.18** | 2026-08-12 | 通用办公左侧面板重设计（参考 Codex 会话栏）：紧凑头部、搜索前置、会话行标题+时间+预览、统一区块小字 |
| **v2.13.17** | 2026-08-12 | 修复运行中已完成的小过程卡不折叠：分段小卡（不含文本）默认折叠、段完成收起；含文本的大过程卡保持展开 |
| **v2.13.16** | 2026-08-12 | 办公板块铺满窗口；删除顶部「办公」二级标签；底栏只显示本地模型、云端引擎不再计入超载报警 |
| **v2.13.15** | 2026-08-12 | 删除方案编写模块与办公二级导航，办公板块收敛为单一「通用办公」入口；左脑记忆改接办公事实，三脑检索不受影响 |
| **v2.13.14** | 2026-08-12 | 通用办公工具/技能面板显示文档技能：docx/xlsx/pdf/pptx 安装到用户级全局技能目录（任意工作区可见），工具面板新增「文档技能」分组 |
| **v2.13.13** | 2026-08-12 | 聊天面板左侧栏可折叠（窄栏 + 状态持久化）；折叠时左下角悬浮绑定模型卡一并隐藏 |
| **v2.13.12** | 2026-08-12 | 通用办公布局优化：右侧边栏删除成本库/搜索标签；绑定模型卡移入左侧栏底部，随面板折叠、不再遮挡导航按钮 |
| **v2.13.11** | 2026-08-12 | 修复记忆中枢·用户画像打不开：后端空结果返回 null 导致前端崩溃，统一改为空数组 + 前端 null 兜底 |
| **v2.13.10** | 2026-08-12 | 修复办公文档处理反复弹 cmd 黑窗：markitdown 转换 / Excel 重算 / 报告导出 / 绘图 / 桌面自动化统一隐藏子进程窗口 |
| **v2.13.9** | 2026-08-12 | 每一轮的大过程卡完成后默认展开（不再只保留最新回合）；大过程卡内部思考卡默认折叠，工具卡/工具组保持折叠 |
| **v2.13.8** | 2026-08-12 | 仅最新回合的大过程卡默认展开，旧回合自动折叠，不再全部摊开 |
| **v2.13.7** | 2026-08-12 | 办公上下文窗口修正（1M→256k，超限自动压缩）+ 大上下文处理提示 + 最终回答兜底 |
| **v2.13.5** | 2026-08-12 | 运行中强制跟随底部：长任务实时显示最新输出，修复“卡住没输出”假象 |
| **v2.13.4** | 2026-08-12 | 外层过程卡（ProcessCard）完成后默认展开，过程文本与过程卡直接可见；内部工具卡默认折叠 |
| **v2.13.3** | 2026-08-12 | 过程卡完成后默认展开：大输出卡片（≥10 行或 ≥2000 字符）完成自动展开，用户折叠后不再干预 |
| **v2.13.2** | 2026-08-12 | 办公过程文件落盘规范：中间文件统一 .gaea/work/、交付物 .gaea/exports/，不再与源文件混放 |
| **v2.13.1** | 2026-08-12 | 补丁：修复 @引用 PDF 时二进制注入上下文导致办公输出不可见 |
| **v2.13.0** | 2026-08-12 | 通用办公打磨：方案分节按字数目标续写（修复 100 字截断）、docx 读取乱码修复、运行态看门狗与停止按钮复位、待办自动收尾、模型中心优化、Herdsman/Ollama 本地模型双模式实测 |
| **v2.12.0** | 2026-08-11 | 稳定工程：WebView2 rAF 冻结看门狗、成本库多级分类重设计、剧照文件化、记忆中枢/办公四库增强 |
| **v2.11.0** | 2026-08-10 | 通用办公大优化：工作区全文搜索、常用资料固定装配、任务模板库、记忆生命周期与自进化 |
| **v2.10.0** | 2026-08-09 | 通用办公三阶段闭环正式发布：前期解析（OCR 四级管线 + 事实底座）、中期编辑（Word 框选即改/修订、Excel 单元格级编辑 + 插行插列 + 公式重算）、后期输出（统一交付出口 + 成本测算模板）、Codex 式文件预览布局（v2.7.0–v2.9.4 累积） |
| **v2.7.9** | 2026-08-09 | 通用办公：粘贴图片附件一键「提取文字」（OvisOCR2 常驻服务），识图/提字双入口 |
| **v2.7.8** | 2026-08-09 | 扫描件 OCR 常驻服务（llama-server 共享端口、按需拉起，多页 2.4s）+ format_convert 直连 Ovis，修复扫描件误判文本 |
| **v2.7.7** | 2026-08-09 | 扫描件 OCR 上 OvisOCR2（0.8B 文档解析 GGUF+Vulkan，Markdown/表格/公式），auto 链路 Ovis→Rapid→WinRT→VLM |
| **v2.7.6** | 2026-08-09 | 性能专项：docx→MD 提速约 12x（修复 O(n²) 标签扫描）、PDF 约 4x、xlsx 微调，新增转换基准 |
| **v2.7.5** | 2026-08-09 | 方案校验增强：整改建议 + 原文定位（章节/摘录/一键跳转），AI 覆盖建议透传 |
| **v2.7.4** | 2026-08-09 | 事实底座一键沉淀长期记忆（去重/分类映射），后续对话自动加载 |
| **v2.7.3** | 2026-08-09 | 事实底座一稿多用：fact_add/list/clear 工具 + 侧栏事实底座面板，交付物基于同一底座生成 |
| **v2.6.9** | 2026-08-09 | 移除 VoxCPM2（实测不达标：耗时长、音色男女混乱），本地 TTS 保留 CosyVoice2 |
| **v2.6.8** | 2026-08-09 | 模型中心一键启动本地 TTS 服务：gaea 启动自动保活 CosyVoice2/VoxCPM2，模型卡片「启动」/测试连接/合成前兜底均自动拉起 |
| **v2.6.7** | 2026-08-09 | VoxCPM2 Vulkan GGUF 加速（克隆 RTF 0.65–0.84 / 语音设计 0.60）+ 本地 TTS 4 音色替换（火山引擎样本）；含 v2.6.5/2.6.6（CosyVoice2 GGUF+Vulkan 提速 8–10×、VoxCPM2 本地引擎接入） |
| **v2.4.5** | 2026-08-08 | 通用办公欢迎页重设计：6 大核心能力卡 + 内置技能 chips，替换土壤修复旧内容 |
| **v2.4.4** | 2026-08-07 | 文件预览重设计：对话内点击文件打开大尺寸预览（docx/xlsx/pdf 内联转 Markdown），右侧面板删消息/报告并默认折叠 |
| **v2.4.3** | 2026-08-07 | 精简内置工具集：38→17 个核心工具，文档专项工具移交 ModelScope 技能，清理前端死列表 |
| **v2.4.1** | 2026-08-07 | 设置面板重设计：按功能板块左侧导航 + 全局搜索，清除死代码与重复面板 |
| **v2.4.0** | 2026-08-07 | 网页调试桥接（GAEA_HTTP_PORT：RPC+SSE 驱动同一内核）+ 办公引擎热加载（技能/工具/插件免重启生效） |
| **v2.3.0** | 2026-08-07 | 界面焕新与办公整合：角色库/首页/智能办公/方案编写重设计，随机生成完善（含人格） |
| **v2.2.0** | 2026-08-07 | 统一角色库：全局角色资产（小说×聊天同一模型）、小说抽卡单向引用、聊天内选角色、角色记忆隔离、状态/记忆/追踪归集角色库、取消轻语称谓 |
| **v2.1.0** | 2026-08-07 | 二代完善：模型中心持久化/启停语义、Cmd+K 引擎路由、OAuth 回归、前端 E 系列守卫、小说剧照与模型链路审计 |
| **v2.0.1** | 2026-08-07 | 三脑底座：模型路由降级链、三脑记忆、主脑可选编排、基线加固 |
| **v1.0.0** | 2026-08-01 | 品牌重塑：wubigrok 正式更名 gaea，全量替换品牌名与 logo，版本重新起算 |
