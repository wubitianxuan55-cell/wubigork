# 调研与规划:dsh-context Go 化移植到办公板块

> 日期:2026-08-27 | 状态:调研完成,待评审后开工
> 目标:① 对话窗口上方增加标签页([对话] [上下文]);② 把 [dsh-context](https://github.com/bowenliang123/dsh-context)
> 的上下文洞察/管理功能用 Go 重写后端、移植进办公板块;③ 本次只做调研与规划,不动代码。

## 1. 调研结论

### 1.1 dsh-context 是什么

dsh-context(bowenliang123,Apache-2.0,v0.34.0,约 482 star)是 DeepSeek Harness 生态里最完整的
**上下文洞察与管理插件**:把"模型每次请求到底带了什么、上下文怎么组成、怎么演进"摊开给用户看。

架构上是典型的 DSH 双半结构:

- **Host 半(对应我们 Go 侧)**:两个 session-projection 投影单元,把会话**持久化事件日志**折叠成
  全量快照,经投影管线推给浏览器,无自定义 RPC:
  - `contextTimeline`(热数据):当前组成、逐请求历史、上下文事件、模型可见 surface 节点 + 被压缩归档的节点。
  - `contextHeaders`(冷数据):system prompt 与工具 schema 的内容纪元,供浏览器展开查看。
- **Client 半(对应我们 React 侧)**:在 `conversation.view` 槽注册"上下文"标签(order 20,排在
  对话 0 / 轨迹 10 之后),在输入区注册 `/context` 命令弹窗,加一个设置卡。用 `useProjection`
  读宿主推来的快照,**零轮询零 RPC**。

功能面(按 README 整理):

1. **Context stats**:轮次/步骤/注入/压缩/剪枝计数。
2. **当前组成(六分类堆叠条)**:system prompt / 工具 schema / 用户消息 / 注入上下文 /
  助手回复 / 工具结果,按模型上下文窗口缩放显示余量;附最贵 top-5 工具 schema。
3. **Context 趋势**:每次模型请求一根堆叠条;Step/Turn 粒度、Total/Delta 模式;
  悬停即看该步组成、点击钉住;压缩点用 ✂ 标记,条高回落看得见。
4. **Step brief**:每根条下方三行自然语言——User(开启该轮的消息)/ In(新进上下文的东西)/
  Response(模型回复或调用的工具),点击直达浏览器对应消息。
5. **上下文事件流**:注入/压缩/剪枝/模型切换/模式切换,带 token 增量、来源、轮次归属、时间。
6. **File activity**:每个被触碰文件的读/写/搜索次数、行数增减、失败标记,点击展开逐操作日志。
7. **Agent network**:当前 agent 的父子/子孙树,每个节点一圈自己会话的上下文占比环,点击跳转。
8. **Context browser**:任一步请求的实际组装内容,六个可折叠分区、逐元素 token 价、全文展开、
  与上一轮 diff、压缩前的归档重建(标注"近似")、多模态图片。

### 1.2 gaea 现有可复用基础(实测盘点)

我们不是从零开始,以下能力已经在仓库里:

- **会话事件日志**:`internal/gaea/agent/session/log.go` —— append-only JSONL,seq=行数,
  payload 无损 JSON 校验,torn-tail 修复(`ReadLogRepaired`/`BalanceEntries`),checkpoint/save/rewind
  齐全。事件种类:turn_started / reasoning / text / message / tool_call / tool_result /
  usage / notice / phase / approval_request / ask_request / turn_done / compaction_started /
  compaction_done / retrying / steer / user_message / system_message。
- **上下文内核**:`internal/gaea/context/` TCCA 四层(identity/runtime/skill/flow)+
  `CacheMetrics`(保存量、缓存命中、压缩计数、父子层级聚合)。
- **现有 UI**:聊天顶栏已有薄 `ContextBar`(used/window 进度条,来自 `GaeaContext` 绑定);
  右侧面板已有 `WorkspaceTabs`(文件/成果/运行/分析);已有子代理面板(扁平列表,无父子拓扑)。
- **事件推送**:办公板块已有 SSE/事件流(`/api/stream` + `useBridgeWatch`),前端免轮询。

### 1.3 DSH → gaea 映射表

| dsh-context 概念 | gaea 对应/落点 | 改动量 |
|---|---|---|
| session-projection 注册表 + 折叠 | 新增 `internal/gaea/contextview`(从会话日志折叠 timeline) | 新包 |
| `contextTimeline` 快照 | 新绑定 `GaeaContextView(sessionID) → ContextTimeline` | +1 绑定 |
| `contextHeaders`(prompt/schema 内容) | 折叠时携带 system 内容与工具 schema 文本 | 同包 |
| contextPressure/token-meter | provider usage 实际 promptTokens 锚定 + 六分类拆分估算 | 估算器 |
| `conversation.view` 标签 | **对话窗口上方新增 tab bar:[对话] [上下文]** | 前端 |
| `/context` 命令 | 复用办公 slash 体系(/compact /dream 同路径) | 前端+命令 |
| inject/compact/prune/switch 事件 | 现有 compaction/usage/tool_result 已覆盖大半;inject/prune/switch 需补日志写点 | 小 |
| Context browser | 快照 + 日志逐条读取 | 新前端组件 |
| File activity | tool_call/tool_result payload 折叠(路径/截断/输出) | 折叠函数 |
| Agent network | 子代理拓扑:需要补 parentId 或先做扁平(与 v3.2.0 决策一致) | 中 |
| 设置卡 | 前端 localStorage 偏好(粒度/模式/排序) | 小 |

### 1.4 UI 效果图解析(用户 2026-08-27 提供)

用户提供了一张 dsh-context 实机效果图(本机 gaea 真实数据:AGENTS.md 指令注入、
tool-jobs 通知、OfficeMemoryLibrary 文件活动等),这是移植的 UI 目标,要点如下:

**对话窗口顶部标签条**:`[ 对话 ] [ 轨迹 ] [ 上下文 ]`,选中态蓝色下划线,当前选中"上下文"。
→ 不是两标签,而是 dsh 的 conversation.view 三槽语义:对话(现 Transcript)、轨迹(dsh 的
trajectory 视图,gaea 可先以工具调用/步骤时间线占位)、上下文(本移植)。

**上下文页 = 左右两栏看板**:

左栏(主数据区):
1. **上下文统计卡**:轮次 / 步数 / 注入 / 压缩 / 剪枝 / 工具调用 / 图片 / 缓存命中% /
   预估费用,每项独立数值块。
2. **当前上下文卡**:模型名(deepseek-v4-flash · deepseek-official)、`351.3k / 1.0M tokens`、
   35% 分段进度条、六分类图例(系统提示词蓝 / 工具定义橙 / 用户消息绿 / 注入内容紫 /
   助手消息深蓝 / 工具结果青),各带 ≈tokens 与占比。
3. **上下文趋势**:堆叠柱状图(0~350.6k),按钮 `步数 | 轮次 | 全局 | 增量`(全局=Total、
   增量=Delta),提示"× 表示压缩/剪枝";点击某根柱联动下方详情与右侧浏览器。
4. **步骤详情卡**(如"第2轮·第60步"):时间戳、实际 prompt / 输出 / 缓存命中率、
   输入(grep …)→ 回复(read …)、该步六分类构成。
5. **上下文事件流**:筛选按钮 `注入 | 压缩 | 剪枝 | 切换 | 模式`;条目形如
   `+ 注入 指令注入 · .gaea\AGENTS.md → 第1轮·第172步 +10.5k 22:47:54`
   (类型 · 来源 · 轮步归属 · token 增量 · 时间)。

右栏:
6. **上下文浏览器**:副标题"对比上轮末步的变动 | 当前(下一次请求)";顶部估算合计
   (≈241.8k)分段条;六个分类可展开,每类显示 `N项 ≈tokens (占比)` 与相对上轮
   增量徽标(`+1 → +33`);展开后看每条实际内容。
7. **文件活动**:统计 chips `全部 | 读取 | 写入 | 搜索 | 图片` + 路径过滤;每文件行 =
   操作次数、行数差(`+370 -150`)、最后操作时间;点击展开逐操作日志。

右上角插件信息卡是 dsh 特有(版本/更新/仓库链接),gaea 移植时**不照搬**,改为帮助入口
或省略。

视觉要求:浅色卡片式、信息密度高;六分类颜色固定并复用 gaea 设计令牌;增量绿 `+` /
减少红 `-`;趋势图原生 SVG 不引重型图表库。

## 2. 方案设计

### 2.1 产品形态

办公板块聊天窗顶部(Transcript 上方、顶栏下方)新增一行标签条(对齐效果图):

```
[ 对话 ]  [ 轨迹 ]  [ 上下文 ]   ← 新 tab bar(对话窗口上方,选中态蓝色下划线)
─────────────────────────
对话:现有 Transcript
轨迹:工具调用/步骤时间线(Phase A 先占位,后续可接事件日志 timeline)
上下文:左右两栏看板(本移植主体)
─────────────────────────
GoalCard / TodoCard / Composer(不变)
```

“上下文”标签页布局(照效果图):

- 左栏:上下文统计卡 → 当前上下文卡(六分类分段条)→ 上下文趋势(步数|轮次|全局|增量)
  → 步骤详情卡 → 上下文事件流(注入|压缩|剪枝|切换|模式筛选)。
- 右栏:上下文浏览器(当前/对比上轮,六分类展开 + 增量徽标)→ 文件活动(读取|写入|搜索|图片
  chips + 路径过滤 + 行数差)。
- `/context` 命令在对话内直接弹同样的看板 Modal。

### 2.2 后端(Go)设计

新增包 `internal/gaea/contextview/`(纯折叠,无 I/O 依赖,可单测):

```go
// timeline.go —— 从 session 日志折叠出 dsh-context 同构快照
type Category struct {
    System, Tools, User, Inject, Assistant, Tool int64 // tokens
}

type RequestRecord struct {
    Seq       int64    `json:"seq"`              // 日志锚点(浏览器/排序/去重用)
    Turn      int      `json:"turn"`
    Step      int      `json:"step"`
    Category  Category `json:"category"`
    BriefUser string   `json:"briefUser,omitempty"`
    BriefIn   []string `json:"briefIn,omitempty"`
    BriefResp string   `json:"briefResp,omitempty"`
    Model     string   `json:"model,omitempty"`
    Usage     *Usage   `json:"usage,omitempty"`  // provider 实际值
}

type ContextEvent struct {
    Kind   string `json:"kind"`   // inject | compact | prune | switch | mode
    Seq    int64  `json:"seq"`
    Delta  int64  `json:"delta"`  // 净回收/新增 tokens
    Source string `json:"source"` // 文件/插件/技能名
    Turn   int    `json:"turn"`
    Step   int    `json:"step"`
}

type SurfaceNode struct {
    Seq    int64  `json:"seq"`
    Cat    string `json:"cat"`              // user|inject|assistant|tool|system|tools
    Tokens int64  `json:"tokens"`
    Text   string `json:"text,omitempty"`   // 截断预览,浏览器端展开拉全文
    Gone   *int64 `json:"gone,omitempty"`   // 被压缩/剪枝取代的 seq
}

type ContextTimeline struct {
    Current      Category       `json:"current"`
    Window       int64          `json:"window"`
    Stats        Stats          `json:"stats"`       // turns/steps/injects/compacts/prunes
    Requests     []RequestRecord `json:"requests"`   // 保留最近 N(默认 200)
    Events       []ContextEvent  `json:"events"`     // 全量(分页)
    Nodes        []SurfaceNode   `json:"nodes"`      // live surface(带 archive)
    Archive      []SurfaceNode   `json:"archive"`
    Dropped      int             `json:"droppedNodes"`
    ArchiveFloor *int64          `json:"archiveFloor,omitempty"`
}

func FoldTimeline(entries []session.LogEntry, window int64, retention int) (ContextTimeline, error)
```

要点:

- `FoldTimeline` 是**纯函数**:输入日志条目 + 窗口,输出快照;与 DSH 投影 unit 同构,可黄金测试。
- **token 估算**:六分类先用统一估算器(如 `len(utf8)/4` 或按 provider 口径);当 usage 事件带
  promptTokens 时,用实际值锚定整条,再按分类比例拆分(与 DSH "官方 meter + fold 比例"同思路,
  保证与顶栏 ContextBar 口径可对齐)。
- **archive 重建**:compaction 事件携带被移除消息的摘要(现有 `CompactionDone` payload 已含
  Messages/Summary/Archive),折叠时给对应节点打 `Gone` 标记,浏览器端可重建压缩前任意步。
- **补日志写点**(小改动,需审计):① 注入上下文(ResolveRefs/attachments/memory 注入点)追加
  `inject` 事件;② tool_result 截断(已有 Truncated 标志)折叠为 prune 事件,可不补写点;
  ③ 模型切换/模式切换追加 `switch`/`mode` 事件(前端已有信息,从 UI 调用点补)。

绑定:新增 `GaeaContextView(sessionID string) (ContextTimeline, error)`(绑定面 +1,漂移检查同步),
事件推送走现有 `/api/stream`(turn/usage/compaction 到达时前端主动拉一次,或后端推送 `context` 事件)。

### 2.3 前端(React)设计

新增组件(全部挂在 gaea 办公板块):

- `ChatTabs`(新):Transcript 上方的 `[对话] [轨迹] [上下文]` 切换条,状态持久化;
  轨迹页 Phase A 先渲染现有工具调用/步骤卡片流占位。
- `ContextView`(新):左右两栏看板容器,复用现有 `ContextBar` 的视觉语言(令牌/堆叠条样式)
  与效果图六分类配色(映射到设计令牌)。
  - `StatsBoard`:轮次/步骤/注入/压缩/剪枝计数。
  - `CurrentComposition`:六分类堆叠条 + 窗口余量灰轨 + top-5 贵工具 schema。
  - `TrendChart`:每请求一根堆叠条;`步数|轮次|全局|增量` 四钮、悬停 tooltip、
    压缩/剪枝 × 标记、点击柱联动步骤详情与浏览器;原生 SVG,**不引入图表库**。
  - `StepDetail`:选中柱的"第N轮·第M步"卡(时间戳/实际 prompt/输出/缓存率/输入→回复/构成)。
  - `EventsList`:注入/压缩/剪枝/切换/模式事件流,分类筛选 chips。
  - `FileActivity`:每文件读/写/搜索/图片计数、行数差(`+370 -150`)、失败红点、路径过滤。
  - `AgentNetwork`:子代理树 + 上下文占比环(Phase C)。
  - `ContextBrowser`:六个可折叠分区、`N项 ≈tokens`、相对上轮增量徽标、展开看全文、
    压缩前近似标注、"当前(下一次请求)"与"对比上轮末步"双模式。
- `/context` 命令:复用办公 slash 命令路径,弹 `ContextModal`。

数据流:进入上下文页时 `GaeaContextView` 拉全量快照;运行中通过现有 SSE 事件订阅增量刷新
(turn_done/usage/compaction 后重拉或增量合并)。**不做轮询**。

### 2.4 阶段拆分(建议按周版本节奏)

| 阶段 | 内容 | 工作量 |
|---|---|---|
| **A 核心闭环** | 后端 `contextview` 包(折叠+估算+事件)+ `GaeaContextView` 绑定 + 对话区 tab bar + 组成卡片 + 趋势图(Total) | 1-1.5 天 |
| **B 洞察** | Context browser(展开/逐元素 token/压缩归档重建)+ `/context` 命令 + 事件流列表 + Delta 模式 | 1 天 |
| **C 增值** | File activity + Agent network(先扁平,parentId 补全后升级为树)+ 补 inject/switch/mode 日志写点 | 1 天 |
| **D 打磨** | 设置持久化、多模态图片、性能(retention/分页)、i18n | 0.5-1 天 |

### 2.5 风险与裁剪(诚实清单)

1. **token 口径**:分类估算 vs 顶栏 ContextBar 的 used 可能不一致。对策:以 provider 实际
   promptTokens 为锚,分类为拆分;两者永远同源。
2. **日志缺 inject/switch/mode 事件**:Phase B/C 需在注入点、模型切换点补写;审计确认改动面小,
   且旧日志缺失时快照标注"部分事件不可追溯",不硬报错。
3. **子代理父子拓扑**:v3.2.0 已记录 meta 无父子关系。Phase C 先扁平列表,后续给 subagent
   transcript/task 补 parentId 再升级为树(与 dsh-context agentGraph 对齐但按需)。
4. **大会话性能**:折叠是 O(n) 读日志。对策:retention 上限(最近 200 requests)+ events 分页 +
   快照缓存(ver/seq 检查点,复用 DSH 投影缓存思想)。
5. **前端体积**:趋势图/堆叠条用原生 SVG,不引入 echarts/mermaid 等重库;ContextView 整体
   `React.lazy` 挂到 tab,避免拖大办公主块。
6. **与现有 UI 的关系**:右侧 `WorkspaceTabs`(文件/成果/运行/分析)保留不动;新 tab bar 只负责
   对话区视图切换,避免两套标签语义打架。

### 2.6 验收标准

- Go:`FoldTimeline` 黄金测试(构造会话日志 → 期望快照)、token 估算器测试、事件折叠测试;
  `go test ./...` 全绿;绑定面漂移 PASS(497+1)。
- 前端:tab 切换、趋势图 Total/Delta、浏览器展开、`/context` 命令;vitest 新增用例;
  eslint 0/0、tsc 0 errors;构建产物对比(办公主块体积不显著增长)。
- UI 对齐:布局与效果图一致([对话|轨迹|上下文] 三标签、左右两栏、六分类配色、
  统计卡/趋势/步骤详情/事件流/浏览器/文件活动逐项对照),走查通过。
- 端到端:真实跑一轮办公任务(读文件→工具调用→压缩),上下文页趋势/事件/文件活动与 Transcript
  所见一致;压缩后 ✂ 标记与回落实测可见。

## 3. 一句话总结

gaea 的会话日志 + TCCA 上下文内核 + 事件流已经构成 dsh-context 的 80% 底座;缺的是
**从日志到"逐请求上下文快照"的折叠层**和**对话区上方的标签页 UI**。按 Phase A→D 推进,
预计 4-5 个工作日可达到 dsh-context 核心功能水平,且完全复用现有质量门禁。

---

## 附:Phase A 实施记录(2026-08-27,未发布)

### 已落地

- **`request_header` 事件**:`event.go` 新增 `RequestHeader` kind;`session/log.go` 映射
  `request_header` 日志;`agent_stream.go` 在每次模型请求前发出(系统 prompt + 工具 schema,
  「模型可见必入日志」在请求头层的落点)。旧日志无此事件,折叠按估算降级。
- **`internal/gaea/contextview` 新包**:`FoldTimeline(entries, window, retention)` 纯函数折叠,
  输出六分类当前组成 / 逐请求趋势 / 上下文事件 / surface 节点与归档;token 估算复用仓库口径
  (chars × 0.25),有 usage 时按实际 promptTokens 等比锚定;`Referenced context:` 前缀的用户
  消息拆分为 inject + user;compaction 节点标记 gone 并计负 delta;tool_result 截断记 prune。
- **绑定**:`GaeaContextView() (ContextTimeline, error)`(active session),绑定面 497→**501**
  (含另一线程新增 3 个),漂移检查 PASS。
- **前端**:对话窗口上方新增 `[对话] [轨迹] [上下文]` 标签条(`ChatTabs`,localStorage 持久化);
  `ContextView` 看板 = 统计卡 + 当前组成(六分类分段条/图例)+ 趋势图(原生 SVG,步数|轮次|
  全局|增量四钮,增量暂禁用 Phase B)+ 步骤详情 + 事件流(注入|压缩|剪枝筛选);
  轨迹页 Phase A 占位;浏览器 mock 同步。

### 验证(全绿)

- Go:112/112 包 + `go vet` 通过;`contextview` 7 个折叠黄金测试。
- 前端:vitest **660/660(125 文件,+8:ContextView 4 + 另一线程 4)**;eslint 0/0;tsc 0 errors。
- 绑定面:501 方法漂移 PASS。
- `wails build` 成功(contextview 类型已进 Wails KnownStructs)+ 冒烟 `/api/health 200`。

### 轨迹标签(Phase B,2026-08-27 与用户 UI 截图 + DSH 源码对齐后完成)

- **对齐 DSH `packages/client/ui-trajectory` 的事件账本语义**(用户指出后重读源码):
  轨迹 = 按轮次组织的**扁平事件记录表**,不是分组步骤卡。记录类型:
  user / request-header / assistant / tool(含 parentId 嵌套子工具)/ compact /
  ask / approval;每条记录带 ts、durationMs、step 归属;轮间压缩独立成
  Between-turns 区段;turn-end 带错误。
- 后端:重写 `internal/gaea/trajectory`(types + fold + 9 个黄金测试):
  header change 检测(initial/system/tools/system-and-tools)、tool dispatch+result
  按 ID 合并、parentId 保留、running/error 状态、轮间/轮内压缩分区、ask/approval 记录。
- 前端:重写 `TrajectoryView`:统计 chips(Duration/Turns/Calls)+ 搜索框 +
  类型徽标(ASSISTANT 紫 / TOOL 橙 / 提问 深蓝 / REQUEST HEADER / COMPACTION)+
  点击展开检查器(工具参数/结果、请求头、用量、耗时)+ Between-turns 区段;
  新增 `User`/`Scissors` 图标到 icons.ts。5 个 vitest。
- 验证:Go 全量 + vet;vitest **665/665**;eslint 0/0;tsc 0;绑定面 502 漂移 PASS;
  wails build + 冒烟 200。

### 待办(Phase C/D)

- 增量(Delta)模式、上下文浏览器(展开原始内容/逐元素 token/压缩前近似重建)、`/context` 命令。
- 注入/模型切换/模式切换事件写点(现在 inject 靠前缀启发式,AGENTS.md 归入 system 分类)。
- File activity、SSE 增量刷新(现为回合结束重拉)。
- 轨迹:虚拟滚动/分页(大会话)、时间条 Overview 投影、流式跟随尾部。

### Agent 网络(Phase C 第一刀,2026-08-28)

- **后端**:`trajectory.FoldAgentNetwork(entries, window)`——主 agent 为根,子代理 =
  日志里拥有子工具记录(parentId 指向它)的 task/run_skill/explore 等元工具调用;
  每节点聚合子树工具调用数/错误数/估算 token/首末时间,状态 running/error/completed;
  `GaeaAgentNetwork` 绑定用 subagents/ meta(task 摘要前缀匹配)富化状态/模型。3 个黄金测试。
- **前端**:`AgentNetworkCard` 挂进上下文页底部——SVG 树(root + 子代理),节点环 =
  token 占比,中心 = 工具调用数,running 绿脉冲,一级子树各一色相,悬停详情条
  (任务/状态/模型/工具数/错误/token/时间)。3 个 vitest。
- **验证**:Go 全量 + vet;vitest **668/668**;eslint 0/0;tsc 0;绑定面 503 漂移 PASS;
  wails build + 冒烟 200。
- 遗留:节点点击跳转子代理会话(Phase D)、嵌套子代理(gaea 当前一层派发,无需树深)。
