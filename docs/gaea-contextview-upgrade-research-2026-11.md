# 办公板块上下文界面升级换代调研——dsh-context 0.83.1 对账（2026-11-11）

> 调研批，不动刀。结论先行：**gaea 已有 dsh-context 的 Go 移植（Phase A，`internal/gaea/contextview`，
> types.go:2 自述），本次"复刻"实为已有实现的升级换代**——缺口集中在四块：跨会话洞察页（全缺）、
> 命令执行卡（后端无折叠）、Fleet 协作面（只有网络图）、浏览器/趋势的 0.83.x 增强交互（DNA/总览/
> 来源 chip/kind 筛选/Raw-Markdown）。

调研底料：npm pack `dsh-context@0.83.1`（Apache-2.0，作者 bowenliang123，162 版本，周下载 ~14.8 万）
解包至 `.tmp/dsh-context-research/`；README 全文 + `lib/index.d.ts`（554 行类型面，实现为压缩 bundle
index.js 182KB / client.js 942KB，逐行读无意义，以 README+d.ts 为准）。

---

## 1. dsh-context 0.83.1 功能全集（上游口径）

### 1.1 五入口

| 入口 | 内容 |
| --- | --- |
| **上下文洞察页**（一级侧栏面板） | KPI 栏（活跃会话/计费 Token/预估成本/缓存命中率/轮次，按 step 发起日归因）；7 天用量成对柱图（Token+成本/天）；8 周活动热力图（点击某日→过滤会话+三统计卡只看当日）；技能加载统计；工具调用统计；命令执行统计；Context Composition 环形图；会话卡片（按最近活动/Token/上下文大小排序、搜索、按工作区分组、点击跳转）；时间范围预设（今天/7天/30天/全部）点击写回偏好 |
| **Context 标签页**（单会话仪表板） | Context Stats（轮次/步骤/用户输入/实时工具调用+成本悬停单价）；Token Stats（七分类拆分）；Timing Stats；Current Context（六色堆叠条+压缩预留区）；Context Trend；Context Browser；Context Events；File Activity；命令执行卡 |
| **Fleet 标签页** | 摘要条（Agent 数/运行中/工作项/消息）；Agent 网络关系图（父/子各一卡，谱系连线悬停点亮整条链，运行中入边绿脉冲）；跨 Agent 协作泳道（每 Agent 一条道：消息+任务标记+委派同时间线，静默道可收起，任务编号跨道认同一件）；协作任务（状态筛选+blocked-by 依赖链点亮）；事件检查（家族会话日志合并流：消息/报告/委派，可筛选，含结算来源与完整收尾报告） |
| **/context 命令** | 居中弹窗：Current Composition + Context Browser + vs 上一轮差异徽标 |
| **Preferences 卡** | 面板入口（all/tab/sidebar/hide）、Fleet Tab 开关、洞察默认时间范围、趋势粒度/模式/delta 基准、工具/文件排序默认、趋势时长曲线开关、总览自动阈值（off/100/200/300）、**promptPrefix/promptSuffix（拼接到每个会话系统提示词前后，含子 Agent，部署级行为设定）** |

### 1.2 数据架构（三投影 + 缓存轻量化）

- 三投影：`contextTimeline`（主时间线）、`contextHeaders`（逐请求 system+工具定义快照）、
  `contextActivity`（逐日活动，洞察页数据源）。
- **缓存轻量化**：投影缓存只存精简摘要（组成/计数/成本耗时/模型信息）；重明细（surface 文本、
  请求拆分、文件操作、命令程序、span）在面板需要时从持久日志**重算**——长会话缓存不膨胀。
- 折叠上限（DEFAULT_BOUNDS）：步骤 1500 · 轮次 300 · 事件 400 · surface 节点 2000 · 存档 400 ·
  文件操作 400 · 命令 400。
- 估算 vs 实际：组成按固定密度启发式；趋势固定某步后并列显示服务商报告的 **Actual Prompt/Output/Cache**。

### 1.3 分类与计费口径

- **七分类**：system / tools / user / inject / **skill** / assistant / tool；assistant 内再拆
  `assistantReasoning` / `assistantAnswerTokens` / `assistantCallTokens` 三个子计数。
- 技能独立分类：可用技能目录、`/skill` 调用、`skill` 工具加载的内容单列，不藏在 inject/tool 里。
- 成本：models.dev 价格表映射（文档化规则），`CostPriceFace{rate(hit/miss/write/out), via,
  confidence: high|medium|low}` + `CostPriceMiss`（无价可查的诚实缺口）；DeepSeek 挂牌价为谷时价、
  峰时 ×2，`SessionCostUsage` 按 provider/model × peak/off 分桶；另有 `PlatformBalance`（钱包余额）。
- TimingTotals：wall/ttft/gen/reasoningMs/textMs/toolArgMs/toolBusyMs/idleMs/speedTokens + 每工具 {calls, ms}。
- CommandRecord：完整命令生命周期——status(running/completed/failed/killed)、exit/signal/timedOut/
  denied/background/sandbox/codeMode/outChars/stderr/derived；`programTally` 程序分布统计。

### 1.4 Context 标签页交互（0.83.x 增强面）

- **趋势图**：Step/Turn 粒度 × Total/Delta 模式；✂ 压缩/清理事件标柱；✨ 三行步骤摘要
  （User 回顾 / In 新增内容 / Response 回复+工具，点击行→浏览器对应消息）；悬停即时详情+点击固定
  （固定后并列估算与 Actual）；悬停柱→旁边浏览器实时预览该步上下文；**总览模式**（一键把整会话
  压到面板宽成连续堆叠带，四分位标记，悬停读数，点击固定步并在浏览器展开该分类）；Delta 模式
  增长堆基线上、压缩向下延伸。
- **浏览器**：7 类三层展开（类→项→内容）；工具定义行带**来源 chip**（`tool-*`/`dsh-*`/`mcp:<server>`）
  + 每类独立搜索框 + 按大小/名称排序；助手类 **Thinking/Tools/Answer kind 筛选 chips**（显示命中数）；
  工具结果展开=完整调用（名称/参数/OK-error 状态/**Raw|Markdown 切换**/行数）；附件按消息顺序成卡
  （图片缩略+尺寸+存储大小+官方 token 估算，点击开宿主预览）；**vs 上一轮差异徽标**（+N 项 / ±Nk Token）；
  压缩前旧步骤从存档重建；近似组成诚实标注。
- **DNA 模式**：把选定步骤的上下文按模型读取顺序平铺成色带+平铺列表（位置 `#12`/类别/引入步骤/token），
  工具定义与系统提示词作分组行收尾；卡片等高内滚动。
- **文件活动**：Read/Written/Searched/Images 标题栏即筛选器+路径搜索；±行数（单文件+汇总）；
  搜索命中对应实际文件行（含命中数）；点击行展开完整操作日志→跳浏览器对应工具结果；点击文件名
  开侧栏文件预览（复用宿主查看器）。
- **命令执行卡**：状态/墙钟/步骤定位；会话摘要（runs/failed/ms/bg）；**程序分布与 notable 常驻全局**，
  仅搜索与状态筛选收窄列表；点击程序回填过滤框。
- i18n：中/英/日/西/法/德/俄/葡八语言。

---

## 2. 现状盘点：gaea Phase A 已覆盖面

代码规模：Go `internal/gaea/contextview`（types 286 + fold 875 + fold_timing 180 + detail/estimate/
imgrefs/imgtoken ≈ 2861 行，黄金测试 1061 行）+ 绑定 `internal/app/gaea_ui_contextview.go`（338 行）
+ 前端 `ContextView.tsx` 817 + `inspector.tsx` 861 + `cards.tsx` 453 + `AgentRadial.tsx` 377 +
`ContextModal.tsx` 97 + `cats.ts` 42 ≈ 2650 行。

| dsh 能力 | gaea 现状 | 证据 |
| --- | --- | --- |
| 六分类组成+堆叠条+压缩预留 | ✅ CurrentContextCard 三层水位竖标 | ContextView.tsx:50,76 |
| 逐请求趋势（step/turn × total/delta） | ✅ | ContextView.tsx:324,367 |
| ✂ 事件标柱 | ✅ compactSeqs | ContextView.tsx:278,411 |
| ✨ 三行步骤摘要+点击跳浏览器 | ✅ brief 三行+BriefUserSeq/BriefRespSeq 锚 | types.go:44-51 |
| 悬停详情+图例联动 | ✅（v4.27/v4.69） | ContextView.tsx:62,235,338 |
| vs 上一轮差异徽标 | ✅ delta strip（含跨压缩 Approx 诚实标注） | ContextView.tsx:462, types.go:94-100 |
| 前缀稳定判定 | ✅ **gaea 增强，dsh 无**（PrefixVerdict） | types.go:73-80 |
| Context Browser 三层展开 | ✅ 六分类树+懒加载 NodeDetail | inspector.tsx:282, types.go:129 |
| 附件卡（缩略/尺寸/token 估算） | ✅ NodeImage 官方 patch 口径 | types.go:159-172 |
| 事件流五类+筛选 | ✅ inject/compact/prune/switch/mode | ContextView.tsx:536 |
| File Activity ±行数/命中数/合并 | ✅（上限 200 条） | fold.go:397,422-448 |
| Timing 卡 | ✅ Wall/TTFT/Gen/每工具排行 20（秒级日志诚实近似） | types.go:199-232 |
| 成本 | ◐ usage 自报费率快照 CostRate+CostEstimate+预算闸门+钱包余额 | types.go:262-273, agent/budget/gate.go, billing/balance.go |
| /context 弹窗 | ✅ ContextPill+ContextModal | ContextModal.tsx |
| 偏好卡 | ◐ localStorage contextPrefs+设置面板 ContextPrefsPanel | lib/contextPrefs.ts |
| Agent 网络图 | ✅ AgentRadial 径向树+AgentTree 实时拓扑 | AgentRadial.tsx |
| 子代理会话 | ✅ SubagentThread+GaeaSubagentContextView | gaea_ui_contextview.go:319 |
| 折叠纯函数+黄金测试 | ✅（比上游更严：测试 1061 行） | fold_test.go |

## 3. 差距清单（按档）

**A 档·整块缺失**
1. **跨会话洞察页**：KPI 栏/7 天柱图/8 周热力图/技能统计/工具统计/命令统计/Composition 环形图/
   会话卡片——全无。需 Go 新增跨会话 activity 投影（对齐上游 `contextActivity`：逐日
   tokens/requests/cost/turns/toolCalls/toolsMs/wallMs + skills/tools/cmds 三 tally）+ 汇总缓存 +
   前端新页。挂载点待拍板（办公板块一级 tab / 运行监测板块 / 设置面板）。
2. **命令执行卡**：fold.go 无命令折叠（tool_dispatch/tool_result 数据已在日志，shell 类工具
   可推导 status/exit 近似 + programTally；`derived:true` 语义上游已有先例）。
3. **Fleet 四件缺三**：编队摘要条/跨 Agent 协作泳道/协作任务面板/事件检查合并流均无；
   数据源 GaeaAgentNetwork（gaea_ui_contextview.go:213）+ SubagentRuns 富化（:272）已有。

**B 档·分类与口径升级**
4. **技能注入第七分类**：gaea 技能经 run_skill 管线执行、内容以工具结果入上下文
   （skill/tools.go:67,104）——fold 层可按工具名推导 skill 分类（跨会话技能 tally 同源），
   无需内核加事件；若要「可用技能目录注入」单独可见则需内核 RequestHeader 扩展。
5. **assistant 三拆**（reasoning/answer/call 子计数）：日志已有 reasoning/text/message 三事件，
   fold 可拆；SurfaceNode 需加 reasoning/answerTokens/callTokens 字段（对齐上游 d.ts）。
6. **成本价格表映射**：现为「usage 自报费率、无则不显示」；上游是内置价格表+置信度+缺口诚实标注
   （CostPriceMiss）。gaea 形态=内置常用模型价格快照（含更新日期）+ via/confidence 字段 +
   无价时 miss 列表；DeepSeek 峰谷 ×2 分桶可作可选项。
7. **保留量拉齐**：retention 200 vs 上游 1500 步；events/nodes/files 上限同步放宽
   （配合按需重算不膨胀桥载荷）。

**C 档·交互增强（前端为主）**
8. 浏览器：工具**来源 chip**+类内搜索/排序、助手 **kind 筛选 chips**、工具结果 **Raw|Markdown**、
   OK/error 状态（Err 字段已有）。
9. 趋势：**总览模式**（整会话压缩到面板宽）、悬停柱→浏览器实时预览联动、固定步并列 Actual。
10. **DNA 模式**（按模型读取顺序平铺色带+位置编号列表）。
11. 命令执行卡前端（依赖 A2 后端）。

**D 档·偏好与行为（独立小刀）**
12. 偏好扩展：placement/fleetTab/洞察默认范围回写/总览阈值/durationCurve。
13. **promptPrefix/promptSuffix**：系统提示词拼接（含子代理）——内核行为级功能，影响所有会话，
   须单独立刀+拍板（非纯 UI）。

## 4. 升级换代切刀草案（拍板池，未拍板不动）

- **刀 1 后端折叠扩展**：assistant 三拆 + skill 分类推导 + 命令执行折叠 + bounds 拉齐 +
  价格表映射（B4-B7+A2）。纯函数纪律延续，黄金测试同步扩。
- **刀 2 跨会话 activity 投影**（Go）：跨 workspace 会话日志扫描汇总+轻量缓存（对齐上游
  缓存轻量化：摘要常驻、明细按需重算）。
- **刀 3 洞察页前端**：KPI/柱图/热力图/三统计行/Composition/会话卡片（A1）。挂载点先拍板。
- **刀 4 Context tab 前端升级**：浏览器增强+总览/DNA+悬停联动（C8-C10）。
- **刀 5 Fleet 页**：摘要条+泳道+协作任务+事件检查（A3）。
- **刀 6 偏好与入口**（D12）；**promptPrefix/Suffix 单独拍板**（D13）。

## 5. 纪律与约束（动刀时适用）

- 图表延续自绘 SVG（仓内无 echarts/recharts/d3 全量库；3d-force/d3-force/gsap/react-window 已有）。
- i18n 单源 en.ts（DictKey），zh/zh-TW 同步；contextview.* 键已有 174 个，新增沿用。
- 语义色走 `--md-sys-color-*`/`--color-*` 变量；cats.ts 色板 hex-exempt 先例（第七分类加色沿用）。
- mock 契约：lib/mock 形参须过 bindingSignatures 契约测试；bindings_completeness_test 白名单。
- 诚实纪律：估算/近似/缺失一律标注不造数（Estimated/Approx/CostRate nil 先例全仓贯彻）。
- 上游方言对齐：新字段名尽量贴 d.ts（surface/requests/events/fileOps/commands/spans），
  便于后续对照上游演化。
- 遵循「调研→拍板→动刀」三段分离；本文件只交拍板池。
