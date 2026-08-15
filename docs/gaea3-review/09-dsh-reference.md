# 09 · DSH 参考机制调研报告

> 定位：gaea 3.0 架构改造（原则级向 DeepSeek Harness 靠拢）的机制级参考证据。
> 调研对象：C:\AI\deepseek-harness（只读，未改代码）。
> 结论形态：每个机制给出「原理 3-5 句 + 关键源码位置 + 对 gaea 3.0 的映射建议」；源码位置为仓库相对路径 + 行号。
> 配套导读：docs/architecture.md、docs/tool-execution-pipeline.md、docs/subsystems/{core,session,session-projection,tools,scope,persistence,web}.md（结论以源码为准）。

## 1. 概览

DSH 是一个“全部由插件构成”的系统：模型适配器、工具注册表、会话日志、agent 循环本身都是插件（docs/architecture.md:15-25「There is no privileged core to patch」）。对 gaea 3.0 最有价值的不是 Cordis 框架本身（gaea 不移植），而是四个**可脱离 Cordis 独立复制的机制形态**：

1. **事件日志即事实源**：SessionEventMap 可合并扩展事件字典；append-only 日志 + surface 投影 + 检查点；任何模型可见输入必须先入日志（docs/architecture.md:96「Model-visible means logged」）。→ 对应 gaea §5.1。
2. **工具注册与流水线**：注册（schema 进 prompt）+ 执行流水线事件（pre-execute/execute/post-execute/result），策略（如“读前必读”）在流水线**外**通过单决策槽事件挂接，工具代码不感知策略。→ 对应 gaea §5.3 之前的工具层改造。
3. **capability seam 三元组**：Service Definition（抽象类 + ctx 声明 + 事件词汇）/ Service Provider（具体实现，互斥注册）/ Consumer（工具、agent loop）。→ 对应 gaea §5.3 Provider Seam，可直接照搬形态。
4. **同一引擎 + 不同 agent 配置**：per-agent scope 注册空间 + agent-preset（目录 = 一个 agent.cordis.yml 插件行列表），会话日志记录 agent-preset/selected。→ 对应 gaea 二期编程板块“模式预设”。

装配（profile/bundle/patch 分层）与前端（slot + ConversationNodeDefinition）是另外两个独立可借鉴面。总体判断：**gaea 3.0 规划的四个 Step 在 DSH 中均有成熟同构物，且 DSH 比 gaea 规划更精细的地方集中在：崩溃恢复语义（torn tail）、投影缓存（ver/seq 检查点）、策略外挂的“单一决策槽”、per-agent 工具可见性**。

---

## 2. 会话事件日志机制（原理 + 源码 + 映射建议）

### 2.1 原理

1. **事件字典是可合并扩展的接口**：SessionEventMap 是接口，任何插件用 declare module 合并新事件类型（types.ts:236）；事件全集是运行时集合 KNOWN_SESSION_EVENT_TYPES（known-event-types.ts:19，由脚本生成），读路径遇到未知类型：带 ignorable: true 标记可跳过，否则拒绝重建——“宁可过拒，不可静默错读”。
2. **append-only 的硬不变量**：每个事件是 lossless JSON（拒绝 BigInt/函数/循环引用等），seq 恒等于当前 log 长度（index.ts:604 append，seq 连续性契约在 types.ts:404-437 SessionEvent 联合类型与构造/恢复校验），事件进入 log 即提交，发布回调同步但被隔离（观察者失败不影响日志）。
3. **投影派生而非改写**：只有三种事件类型（user/message、assistant/message、tool/result）能上“有序表面”（surface），各自带 surfaceOp: append | {op:replace,start,end} 与 sourceEventSeqs（types.ts:343-380）；SurfaceManager（surface.ts:398）维护节点序，deriveMessages()（index.ts:726）把每个节点经 deriveEventMessage（surface.ts:83）投影成 LLM 消息，带增量缓存（每个节点只投影一次；replace 代际失效）。压缩/回退不用改写中间事件，而是用 replace 节点“遮蔽”旧范围。
4. **持久化与检查点解耦**：内存 SessionStore（index.ts:792）不落盘；持久化是订阅 session/event 的插件。SessionPersistence 抽象（session-persistence/src/index.ts:84）定义 append（首事件 seq 必须等于存储的 next-seq）、load（恢复时把未完成的 turn 用合成事件“补平衡”）、inspect、readFrom（从 seq 续读）、list。协调器（coordinator.ts:588 PersistenceCoordinator）提供缓冲、串行化、崩溃修复；后端只需实现 6 个原语（coordinator.ts:127 PersistenceBackend）。
5. **崩溃恢复语义（最值得抄的部分）**：写路径是 write-behind 批量 + flush 屏障（write-behind.ts:22 SessionWriteBehind：固定延迟合并、失败保留、flush() 排空到静止点）；**检查点由策略插件负责**（session-checkpoint-policy/src/index.ts:63 apply）：在 llm/stream 前、顶层 tools/execute 前、agent/pre-step 前强制 sessions.flush()，即“请求入日志并落盘后才发模型/执行副作用”（fail-closed）。读路径：最后 turn/end 之后的不完整物理尾巴（torn tail）被截断，完整但未关的 turn 用合成 closers 补平衡（coordinator.ts 的 crash/torn 语义；SQLite 端 scanRows，schema.ts:232：最后一个 turn/end 之前有洞 = 已提交区损坏，拒绝；之后有洞 = torn tail，容忍截断）。
6. **投影是注册制的领域单位**：session-projection（index.ts:42 ProjectionDefinition：init/apply/view + stateVersion）要求“状态事件必须携带完整后状态（whole-value），绝不只写 delta”；框架驱动 apply 前进、维护 per-session watermark；session-projection-cache（index.ts:27）把每单位状态以 (sessionId, key, ver, seq, val) 落盘，ver 不匹配即丢弃——**检查点是折叠捷径，永不权威**。
7. **标题等派生物也走日志**：session/title 是日志事件（session-title/src/index.ts:94），foldSessionTitle（:191）从最新事件折叠，标题推导者（fallback / LLM provider / 用户改名）互斥注册。

### 2.2 关键源码位置

- 事件字典与不变量：packages/core/session/src/types.ts:236（SessionEventMap）、:343（SurfaceEventType）、:404（SessionEvent 联合）、:56（SESSION_FORMAT_VERSION）
- 事件全集：packages/core/session/src/known-event-types.ts:19
- 投影规则：packages/core/session/src/surface.ts:83（deriveEventMessage）、:398（SurfaceManager）
- Session 主体：packages/core/session/src/index.ts:425（class Session）、:604（append）、:726（deriveMessages）、:472（firstLiveSeq）、:792（SessionStore）
- 持久化契约：packages/session/session-persistence/src/index.ts:84
- 协调器与后端原语：packages/session/session-persistence/src/coordinator.ts:127（PersistenceBackend）、:588（PersistenceCoordinator）
- 批量写：packages/session/session-persistence/src/write-behind.ts:22
- JSONL 后端：packages/session/session-persistence-jsonl/src/format.ts:33（HeaderLine，首行 type:session）、index.ts:121
- SQLite 后端：packages/session/session-persistence-sqlite/src/schema.ts:20（SCHEMA_VERSION=15）、:232（scanRows torn-tail 切割）、index.ts:99
- 检查点策略：packages/session/session-checkpoint-policy/src/index.ts:63
- 投影：packages/session/session-projection/src/index.ts:42、packages/session/session-projection-cache/src/index.ts:27
- 标题：packages/session/session-title/src/index.ts:94

### 2.3 对 gaea 3.0 的映射建议（对应设计 §5.1，基本可直接采纳）

- **日志格式**：<id>.gaea-log.jsonl 每行 {seq, ts, kind, payload} 的设计与 DSH 一致；建议补两条硬不变量——(a) seq 恒等于已写行数（写入器单点控制）；(b) 事件 payload 必须可无损 JSON 序列化（Go 侧写入前校验，拒绝非法值，杜绝“坏事件在恢复时才炸”）。
- **压缩协议**：DSH 的做法（不改写中间事件，checkpoint + 重放）与 gaea §5.1「压缩发生时写 checkpoint + 从 seq 后重放」完全同构；额外可借鉴：DSH 用 **surface replace 节点**遮蔽旧消息（压缩后模型只见压缩摘要、人类仍可读原始流）。gaea 若想要“压缩后旧内容仍可追溯”，可以模仿 replace 语义（compaction 事件 + 派生层遮蔽），若只要简单版，checkpoint 即可。
- **崩溃恢复**：直接抄 torn-tail 语义——最后 turn_done 之后的不完整行截断，完整但未关的 turn 补合成关闭事件；“发模型请求前必须已落盘”的检查点策略（在 gaea 是：模型调用前 flush 日志）成本低收益大。
- **投影层**：projectMessages(checkpoint, tail) 对应 DSH deriveMessages；建议把“哪些 kind 投影为消息”做成纯函数表（user/assistant/tool 三类），与 gaea 现有 Session.Messages 兼容层一致。标题/统计/成本全部实现为“从日志 fold”的独立派生函数，不落第二份真值——这是 gaea §5.1 已声明的原则 2。
- **whole-value 规则**：日志事件尽量携带完整后状态（如 usage 累加、todo 全量快照），派生方永不依赖跨事件差值，回放才可任意截断重放。


### 2.4 事件字典对照（gaea §5.1 事件种类 ↔ DSH SessionEventMap）

| gaea §5.1 设计事件 | DSH 对应事件（types.ts SessionEventMap 键） | 投影/表面语义 |
|---|---|---|
| user_message | user/message | surface：append；投影为 user 消息（source 区分人类提示/inject 上下文/目标轮次） |
| assistant_started | （无独立事件；由 turn/step 边界 + assistant/chunk 表达） | 建议 gaea 保留 started 作 UI 用，但标注为纯日志事件 |
| assistant_delta | assistant/chunk | **纯日志**（token 级重放保真）；不投影 |
| assistant_message | assistant/message | surface：append；投影为 assistant 消息；携带 usage（无独立 usage 事件） |
| tool_call | tool/call | 纯日志；arguments 是模型原文 JSON 字符串（未解析），callId 与 result 配对 |
| tool_result | tool/result | surface：append；投影为 tool 消息；可选 meta（工具私有展示载荷，必须 JSON 可序列化） |
| usage | （并入 assistant/message.usage） | 纯日志派生：从 assistant/message 累加 |
| notice | user/message（source=inject）+ agent/inbox 事件 | inject 类上下文走 user/message，是"模型可见必入日志"的落点 |
| compaction | compaction/start / compaction/end / compaction/summary + surface replace | 压缩不删事件，用 replace 遮蔽 |
| steer / retry | （无独立事件；靠 turn/end reason 与 agent/request 瀑布） | gaea 若保留 steer/retry 事件，建议归为"改写了模型视角"类 |
| turn_started / turn_done | turn/start / turn/end（reason: completed/aborted/blocked/error/max-tokens/interrupted） | 纯日志；turn/end 是崩溃恢复的平衡锚点 |

结论：gaea 事件集合与 DSH 交集约 80%，缺的是"compaction 遮蔽""usage 随消息携带"两个语义细节；事件名可保留 gaea 现有命名（snake_case），不强制对齐 DSH 的斜杠命名。

### 2.5 物理格式与恢复走读（JSONL 后端的关键实现细节）

- **文件布局**：每个会话一个文件，首行是 type:'session' 标记的 HeaderLine（format.ts:33），其后每行一个事件；目录按项目路径 + 会话 id 组织，会话 id 经 encodeSegment 转义（format.ts:70-96：遍历所有 UTF-16 码元，不安全字符转 ~XXXX，防路径穿越/防碰撞）。
- **可选 zstd 压缩**：.jsonl.zstd 物理编码，读端 zstd 解码后仍是逐行 JSONL（format.ts:22-31、zstd.ts）。
- **恢复算法（SQLite 后端示例，schema.ts:232 scanRows）**：按 seq 升序扫描；最后一个合法的 turn/end 之前若有解析失败或 seq 空洞 = **已提交区损坏，抛错**；之后 = **torn tail，容忍并截断**（记录 tornFrom 供物理删除）。JSONL 端同语义：可解析前缀保留，无法解析的最后一条记录丢弃，完整的中断 turn 用合成 closers 补平衡（commitRepair 截断 + 补事件）。
- **修订标识**：SessionPersistenceRevision（revision.ts）是后端所有权的模糊令牌，listSnapshots 用它做"日志是否变化"的廉价探测；SQLite 用 sessions 行内单调 revision 字段（schema.ts:44-55）。
- **preparations**（coordinator 的 PreparedSessionSource）：load 出的冷会话被缓存为"未发布 Session"，resume 时直接复用（避免重复解析）；持久化 revision 变化则重载。

### 2.6 gaea 落地骨架（Go 伪代码级建议）

```go
// internal/gaea/agent/sessionlog/log.go —— 建议新增
type Kind string // user_message | assistant_started | assistant_delta | assistant_message | tool_call | tool_result | usage | compaction | turn_started | turn_done | ...

type Event struct {
    Seq     int64           `json:"seq"` // 恒等于已写行数（写入器单点保证）
    TS      int64           `json:"ts"`
    Kind    Kind            `json:"kind"`
    Payload json.RawMessage `json:"payload"`
    Surface *SurfaceMeta    `json:"surface,omitempty"` // 仅 user/assistant/tool 三类携带
}

type SurfaceMeta struct {
    Op             string `json:"op"` // "append" | "replace"
    Start, End     int64  `json:"start,omitempty"`
    SourceEventSeqs []int64 `json:"sources,omitempty"`
}

// 追加：校验 payload 可无损 JSON 序列化（json.Valid + 拒绝非法值），seq=len(log)
func (l *Log) Append(kind Kind, payload any, surface *SurfaceMeta) (Event, error)

// 投影：纯函数表，只有三类事件投影为消息
func ProjectMessages(log []Event) []provider.Message

// 恢复：checkpoint + tail 重放
func Restore(checkpoint *Checkpoint, tail []Event) ([]provider.Message, error)
```

要点：Log.Append 单点写文件（append-only、O_APPEND + 原子行写）；ProjectMessages 是纯函数（无状态、可重放）；Restore = 从 checkpoint.seq 之后重放 tail。这三件套实现后，标题/统计/成本/多前端同步全部变成"读日志折叠"。

---

## 3. 工具注册与流水线

### 3.1 原理

1. **工具 = schema + 输出契约 + 执行体 + 可选展示钩子**：ToolDefinition（core/tools/src/index.ts:222）含 name/description/parameters（JSON Schema）+ output（schema/render/presentationMeta）+ execute + presentCall/presentResult（纯函数、可重放的 UI 渲染意图）+ timeoutMs/isConcurrencySafe。defineTool（schema.ts:545）从 TS 类型推断 JSON Schema，并在 execute 前校验参数；展示钩子对历史参数“软校验、失败回退通用渲染”（绝不 throw）。
2. **schema 进 prompt 是组装器的事，不是工具的事**：system-prompt 服务（core/system-prompt/src/index.ts:53 PromptSection、:358 section()、:430 tools()、:467 assemble()）维护有序 prompt 片段 + 工具 schema 提供者；ToolRuntime 构造时注册 ctx.systemPrompt.tools(context => this.wireSchemas(context.scope))（index.ts:832），每个工具插件同时注册 100-199 号段的引导段落（tool-fs/read.ts 的 tool:read section，order 100）。模型请求时 assemble() 把 sections + tools + variables 合成系统提示。
3. **注册是作用域化的**：ToolRuntime（index.ts:787）用 ScopedLayers（core/scope/src/store.ts:159）——全局层 + 每 agent 覆盖层；register（index.ts:1037）插入所在 ctx 的层并返回可逆 disposer；restrict（:1071）按 agent 做 allow/deny 掩码（会话可用工具集 = 全局 ∩ 该 agent 掩码 ∪ 该 agent 注册）。presentAs（:946）让一个 agent 把全部工具折叠成 run_code 一种呈现（Code Mode），同一进程里 code 与 native agent 并存。
4. **执行流水线 = 三个 waterfall 事件 + 两个通知事件**：tools/pre-execute（允/拒/问）→ 单调 guards → tools/execute（around：超时/重试/度量）→ 工具体 → tools/post-execute（改/换/堵结果）→ 注册表归一化 → finalizeContent → tools/result（只读最终快照）。全程事件化（docs/tool-execution-pipeline.md:8-30），工具实现不感知任何策略。
5. **策略在流水线外挂 = 单决策槽（single-slot waterfall）**：以“读前必读”为例——fs seam 声明三个事件（fs/fs/src/index.ts:58 fs/write-intent、fs/edit-intent、fs/observed）；write 工具执行时调 ctx.waterfall(fs/write-intent)（tool-fs/src/write.ts:111）并 ctx.emit(fs/observed)（:122）；fs-observation-policy 插件（index.ts:21 ObservedStateGate、:119 占用决策槽、**不调 next()**）用 WeakMap<owner, Map<path, version>> 记录每个 agent 会话的观察状态，据此产出 createIfAbsent/replaceIfVersion 意图或 FS_NOT_OBSERVED 拒绝。没有该插件时 write 就是无条件原子覆盖——**策略缺席 = 宽松默认，工具代码零改动**。

### 3.2 关键源码位置

- 工具定义：packages/core/tools/src/index.ts:222（ToolDefinition）、:1037（register）、:1071（restrict）、:946（presentAs）、:980（wireSchemas）、:832（systemPrompt.tools 挂接）
- defineTool：packages/core/tools/src/schema.ts:545
- prompt 组装：packages/core/system-prompt/src/index.ts:53/358/430/467
- 执行流水线图：docs/tool-execution-pipeline.md:8-30
- 策略外挂实例：packages/fs/tool-fs/src/write.ts:111/122、packages/fs/fs-observation-policy/src/index.ts:21/119、packages/fs/fs/src/index.ts:58（事件词汇声明）
- 工具插件注册样板：packages/fs/tool-fs/src/index.ts（apply 统一注册 read/write/edit/read_image）

### 3.3 对 gaea 3.0 的映射建议

- gaea 的 tool.Registry 已有注册/自注册/执行；**缺的是「schema → prompt」的组装器和「执行流水线事件」**。Go 侧轻量实现：给工具定义加 PromptSection(order int, text func(agentCtx) string) 字段；每次请求时按 order 排序拼 system prompt——一行 assemble() 替代现在各板块各自拼 prompt。
- **策略外挂的 Go 形态**：不要在每个工具里 if 判断策略，而是“单决策槽”——工具调 fs.WriteIntent(target, actor)，默认返回无条件意图；策略插件（如 read-before-write）在装配期注册覆盖该函数。工具与策略解耦的收益：新增策略（如“写前必须审批”）不动工具代码。
- **读前必读**：在 gaea 是明显的刚需（现有 write 工具无版本概念）。实现 = per-session map[path]version 观察表（read 时记录、write/edit 时校验 CAS），完全照抄 DSH 的 ObservedStateGate 形态，成本半天。
- 工具可见性 per-agent（restrict 掩码）与“模式预设”配套（见 §5），是编程板块试点前就可以给办公引擎加的能力。


### 3.4 执行流水线阶段表（DSH 事件 ↔ gaea 对应物）

| DSH 阶段 | DSH 事件 | 语义 | gaea 3.0 对应建议 |
|---|---|---|---|
| 前置闸 | tools/pre-execute（waterfall） | 允/拒/问；审批策略在这层 | 工具执行前 hook 切片（allow/deny/ask），审批系统挂这里 |
| 单调守卫 | （注册守卫，不可重排） | deny 或 abstain | 可裁剪：gaea 无重排需求，守卫并入 pre-execute |
| 环绕 | tools/execute（waterfall） | 超时/重试/度量包裹工具体 | gaea 现有时限/重试在工具内 → 上移为环绕层 |
| 工具体 | ToolDefinition.execute | 返回 lossless JSON 规范值 | 现 tool.Registry 执行体，补 output schema 校验 |
| 后置 | tools/post-execute（waterfall） | 接受/替换/阻塞/追加上下文 | 可裁剪；需要"结果改写"的场景再用 |
| 归一化 | 注册表归一化 | 管线/结果快照抛错 → isError 结果 | 统一错误 → isError 结构（现有 remediate 思路保留） |
| 终化 | finalizeContent | 最后一个内容不变式 | 可裁剪 |
| 观察 | tools/result（emit） | 冻结的最终结果通知 | 事件 Sink 已具备：工具结果作为事件发出 |

### 3.5 gaea 工具注册 Go 骨架（对应 DSH 注册 + schema 进 prompt）

```go
// internal/gaea/tool/definition.go —— 建议扩展
type Tool struct {
    Name        string
    Description string
    Parameters  map[string]any // JSON Schema
    Output      OutputDecl     // schema + render（新增：规范化输出契约）
    Execute     func(args map[string]any, exec ExecCtx) (any, error)
    PromptSection *Section     // 新增：{Order int; Text func(agent AgentCtx) string}
    PresentCall, PresentResult func(...) (View, error) // 可选：UI 渲染意图（纯函数）
}

// 注册进 per-agent 层还是全局层，取决于调用 ctx 的 scope（等价 ScopedLayers）
func (r *Registry) Register(t Tool) (dispose func())

// 组装：按 Order 排序 PromptSection + 所有可见工具的 Parameters → system prompt
func (r *Registry) Assemble(scope ScopeKey) (system string, schemas []any)

// 单决策槽（策略外挂）：默认无条件；read-before-write 策略在装配期覆盖
type WriteIntentFn func(target string, actor ActorID) (Intent, error)
func (r *Registry) SetWriteIntent(fn WriteIntentFn) // 策略插件调用
```

关键取舍：gaea 不需要 water fall 链的完整泛化，**先实现"pre-execute 钩子 + 单决策槽 + schema 组装"三件套**即可覆盖办公引擎 90% 的场景；审批/读前必读/超时统一改为钩子，而不是散落在各工具内部。

---

## 4. capability seam 范式（定义 / 提供者 / 消费者）

### 4.1 原理

1. **三元组的代码形态**：每个 seam 是三个独立的包族——**定义包**（抽象类 extends Service + declare module 声明 ctx.xxx 服务键 + **事件词汇**声明）、**提供者包**（-local/-sandbox/-windows-acl 等，实现抽象类、作为插件装载、一个 context 只允许一个实现，重复装载报错）、**消费者包**（tool-* 工具、agent loop、其他 seam）。docs/architecture.md:98-105「Seams are why one provider swap changes the whole product」。
2. **fs seam**：FileSystem 抽象（fs/fs/src/index.ts:86）定义 resolve/stat/readText/writeText/editText 等 + 三个 fs/* 事件；提供者 fs-local（宿主文件系统，realpath 目标身份 + 原子写）、fs-sandbox（沙箱文件系统）；消费者 tool-fs。
3. **shell seam**：ShellExecutor（shell/shell/src/index.ts:65）定义 resolve/run/start；提供者 bash-local、pwsh-local 共享同一 settings 命名空间 SHELL_SETTINGS_NAMESPACE（win32 主机二选一）；**提供者叠提供者**——bash-local 通过 ctx.subprocess 派生进程（bash-local/src/index.ts 注释：“over the subprocess capability seam”）。shell-env 是配套注册表：插件可注册受管理的 DSH_* 环境变量（shell-env/src/index.ts）。
4. **sandbox seam**：SandboxProvider.confine(argv, policy)（sandbox/sandbox/src/index.ts:158）返回“被包裹的 argv + 强制完整度 + 拒绝方言签名”；sandbox-local 按平台选 runner 链（bwrap→landlock / seatbelt / Windows ACL 受限令牌），**不可用时 fail-closed 拒绝**；sandbox-policy（sandbox-policy/src/index.ts）是**会话级策略解析器**（effectiveSandboxMode 从 sandbox/mode 日志事件折叠），提供者保持无会话状态；sandbox-windows-acl 是 Windows 特化（workspace SID 常驻授权 + 每会话随机临时目录）。
5. **subprocess seam**：SubprocessRuntime（subprocess/subprocess/src/index.ts:102）定义 spawn/collect/terminal；subprocess-local 实现进程树管理 + 凭据擦除 + SIGTERM→grace→SIGKILL。
6. **策略与 seam 的边界**：策略（观察、sandbox 模式、审批）不进入 seam 定义；策略是独立包，通过事件或独立服务（ctx.sandboxPolicy）挂接。**seam 声明事件词汇，工具是消费者，策略是旁路**。

### 4.2 关键源码位置

- 定义：packages/fs/fs/src/index.ts:86（FileSystem）、packages/shell/shell/src/index.ts:65（ShellExecutor）、packages/sandbox/sandbox/src/index.ts:158（SandboxProvider）、packages/subprocess/subprocess/src/index.ts:102（SubprocessRuntime）
- 提供者：packages/fs/fs-local、packages/fs/fs-sandbox、packages/shell/bash-local、packages/shell/pwsh-local、packages/sandbox/sandbox-local、packages/sandbox/sandbox-windows-acl、packages/subprocess/subprocess-local
- 配套：packages/shell/shell-env/src/index.ts、packages/sandbox/sandbox-policy/src/index.ts（session-mode.ts 的 effectiveSandboxMode）
- 导读：docs/capability-seams.md（服务图）、docs/architecture.md:98

### 4.3 对 gaea 3.0 的映射建议（对应设计 §5.3，直接采纳形态）

- gaea §5.3 的 LLM/Image/OCR/Voice 四 seam 表格**与 DSH 三元组完全同构**，照做即可；补一个 DSH 特有的要点：**定义包必须同时声明“能力事件词汇”**（如 fs 的 fs/* 事件），因为策略外挂依赖事件而非接口方法。gaea 的 seam 定义接口时，顺带定义 image/event、ocr/event 等事件面。
- 提供者互斥注册（重复装载启动报错）与“不可用即 fail-closed”（sandbox 无 runner 就拒绝执行而非裸奔）——gaea 的 image/ocr 后端切换可用同一语义，避免“配置了不存在的后端却静默回退”。
- **会话级策略**（sandbox-policy 从日志事件折叠模式）比 gaea 现状的全局配置更接近“会话可回放”原则：gaea 的 sandbox 模式/审批策略可做成会话事件，恢复会话时自动还原。
- 提供者叠提供者（bash over subprocess）在 gaea 的对应物：命令执行 seam 之上再叠一层（如“沙箱命令执行”= 命令 seam 的提供者），保持调用方只见最上层接口。


### 4.4 seam 清单（DSH 四 seam 的方法面 → gaea 五 seam 对照）

| DSH seam | 定义包（接口方法） | 提供者 | 消费者 | gaea 3.0 对应 |
|---|---|---|---|---|
| fs | fs/fs：resolve/stat/readText/streamText/writeText/editText + fs/* 事件 | fs-local / fs-sandbox | tool-fs | 新增（文件能力目前散在办公工具里） |
| shell | shell/shell：resolve/run/start | bash-local / pwsh-local / bash-sandbox / pwsh-sandbox | tool-bash / tool-pwsh | gaea command 包 → 定义 + local 提供者 |
| sandbox | sandbox/sandbox：confine(argv, policy) | sandbox-local（bwrap/landlock/seatbelt/win-acl） | shell 执行器、fs 后端 | gaea sandbox 包 → 定义 + 平台提供者 |
| subprocess | subprocess/subprocess：spawn/collect/terminal | subprocess-local | bash-local、LSP host | gaea 命令执行底层 → 定义 + 提供者 |
| （配套）shell-env | shell-env：DSH_* 环境注册表 | 各插件贡献 | 工具 | gaea 可裁剪（环境变量注入需求弱） |
| （配套）sandbox-policy | sandbox-policy：会话级模式解析 | session-mode.ts | 所有执行消费者 | **可采纳**：sandbox 模式做成会话事件 |

gaea §5.3 已有四 seam（LLM/Image/OCR/Voice），对照后建议**新增一个 fs seam**（办公工具的文件操作集中化，读前必读策略才有挂点）。

### 4.5 gaea 提供者注册 Go 骨架（seam 三元组在 Go 的形态）

```go
// 1) 定义包（internal/providers/image/image.go）
type Provider interface {
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)
}
type Registry struct{ m map[string]Provider }
func (r *Registry) Register(kind string, p Provider) error { // 重复注册报错
    if _, ok := r.m[kind]; ok { return fmt.Errorf("image provider %q already registered", kind) }
    r.m[kind] = p; return nil
}
func (r *Registry) MustGet(kind string) (Provider, error) { // 未注册 fail-closed
    p, ok := r.m[kind]; if !ok { return nil, fmt.Errorf("image provider %q not registered", kind) }
    return p, nil
}

// 2) 提供者包（internal/providers/image/comfyui.go / openai.go）：实现 Provider，init 时 Register
// 3) 消费者（绘梦板块、image_gen 工具）：只依赖 Provider 接口，kind 来自配置 image.backend
// 4) 事件词汇（seam 定义包一并声明）：image/started、image/progress、image/result —— 供策略/UI 外挂
```

与现状差异：gaea 的 image 后端已是按文件分（image_openai.go/image_comfyui.go）但没有接口与注册表；补接口 + 互斥注册 + 配置驱动 + 事件词汇四项即为完整 seam。特别注意 **fail-closed**：配置了不存在的后端应启动报错或调用时报错，绝不静默回退默认后端（现 engine switch 的行为恰恰是静默回退）。

---

## 5. agent 装配与作用域（模式预设）

### 5.1 原理

1. **per-agent 注册空间是 scope**：createScope(ctx, key)（core/scope/src/index.ts:137）从宿主 ctx 派生一个打了 scope 标记的子 context；在其上做的所有注册（tools.register、systemPrompt.section、事件监听）都归属该 scope，随 scope dispose 整体撤销。scopeTarget（:170）构造**路由载体**：事件按 scope 过滤派发，且**祖先 scope 能收到后代事件**（stand-up 装配可观察每个子 agent），bindScopeParent/scopeChainOf（:72）维护父子链。
2. **作用域注册表 = 全局层 + 覆盖层**：ScopedLayers（core/scope/src/store.ts:159）——全局层 + 每 scope 层；读取按父链合并、最近者胜（chainLayers/merge），注册返回可逆 disposer，层空即回收。工具可见性、prompt 片段、sandbox 模式都走这一套。
3. **同一引擎 + 不同 agent 配置**：ReactLoopAgent（core/agent-loop/src/agent.ts）构造时 this.scope = createScope(loopCtx, this)（:94），之后该 agent 的所有 prompt/tool 注册都落在自己的 scope 层；驱动（AgentLoop，core/agent-loop/src/index.ts:296）对所有 agent 完全同一套（turn/step 循环、deriveMessages 取历史、request/header 折叠、checkpoint 策略），**差异 100% 来自 scope 内的注册内容**。
4. **preset = 目录里的插件行列表**：agent-presets（packages/preset/agent-presets）——目录名是 preset id，目录内含 agent.cordis.yml（组合行列表，discovery.ts:26）+ preset.yml（名称/描述/排序）；用户预设根 ~/.dsh/.agent-presets（:41）。装配时 PresetTree extends Include（mount.ts:57）把组合行**挂到该 agent 的 scope context 下**（每会话私有 realm，服务行必须带 isolate）。仓库自带示例：apps/cli/config/agent-presets/{standard,minimal,code,cordis}/agent.cordis.yml——standard（全功能）、minimal（两工具极简）、code（Code Mode 呈现）、cordis（自指：可写自己运行时的预设）。
5. **会话记录“用了哪个预设”**：header 记 agentPreset（types.ts:98），运行中切换记 agent-preset/selected 日志事件（agent-presets/src/session.ts:26），恢复/派生时 resolveSessionPreset 从日志折叠（最新者胜）——**组合是会话可回放状态的一部分**。
6. **plan-mode 是同型机制**：plan/mode 是日志事件（plan-mode/src/index.ts:53），折叠出“当前是否计划模式”，决定 prompt 里是否含 plan 指引段 + exit_plan_mode 工具是否可用；模式切换只改 prompt 段，不改工具目录。

### 5.2 关键源码位置

- scope：packages/core/scope/src/index.ts:137（createScope）、:170（scopeTarget）、:72（bindScopeParent）；packages/core/scope/src/store.ts:159（ScopedLayers）
- agent 装配：packages/core/agent-loop/src/agent.ts:94（scope 创建）、:246（turn 循环）、:341（deriveMessages 取历史）、:407（buildRequest + request/header）；packages/core/agent-loop/src/index.ts:296（AgentLoop）
- preset：packages/preset/agent-presets/src/discovery.ts:26/41、mount.ts:57、session.ts:26、preset.ts:21；示例组合 apps/cli/config/agent-presets/*/agent.cordis.yml
- plan-mode：packages/plan/plan-mode/src/index.ts:53

### 5.3 对 gaea 3.0 的映射建议（对应二期编程板块“模式预设”）

- gaea §9 二期试点「编程 = 办公引擎 + 模式预设 + manifest 新页面」与 DSH agent-presets 是**同一答案**。Go 侧轻量实现不需要 Cordis：给会话装配加一个 AgentPreset 结构（PromptTemplate + ToolNames []string + Params），会话创建时把预设实例化到该会话的注册空间（办公引擎的 tool.Registry 改为**每会话可覆盖**：全局注册表 + 会话层覆盖，等价于 ScopedLayers 的 global + scoped）。
- **“用了哪个预设”入日志**：建议直接照抄——会话创建头记 preset id，切换记事件；恢复会话时从日志折叠出预设再装配。这条成本极低，但保证“回放出的会话与模型当时看到的工具/prompt 一致”。
- scope 的**多级继承**（预设→agent→子代理链）gaea 可裁剪为单层（会话即叶子）；但“事件按会话过滤 + 全局监听可收所有会话事件”值得保留，这是多前端/审计面板的前提。
- plan-mode 形态可直接映射：gaea 若做“草案模式/审查模式”，就是“日志事件折叠模式 → 决定 prompt 段 + 工具可见性”。

### 5.4 standard 预设逐行解读（DSH 组合文件长什么样）

apps/cli/config/agent-presets/standard/agent.cordis.yml 的实际结构（这是“模式预设”的权威样例）：

- **身份层**：persona（人设 prompt，支持 {{model}}/{{cwd}} 变量解析）+ agent-instructions（读 AGENTS.md 注入，maxBytes 限制）——对应 gaea 的 system prompt 预设（现有 GaeaSetAgentParams 的机制化）。
- **能力层（工具行）**：tool-bash / tool-pwsh（平台互斥，disabled: !!js process.platform === win32）、tool-fs（read/write/edit）、tool-fs-search、tool-web、tool-todo、tool-goal、tool-skill、tool-subagent、tool-workflow……每行只是“往注册表注册工具”，服务本身在 HOST 组合里。
- **realm 规则**：组合里任何“提供服务”的行必须包在带 isolate 的 group 里（每会话私有实例），否则进程全局碰撞——gaea 不需要（Go 无 DI 服务注册），但“会话级实例 vs 全局实例”的边界概念保留。
- **注释即文档**：组合文件头部注释说明“这是 AGENT-PLANE 组合，host 组合拥有注册表本身”——两平面（host=注册表/持久化/沙箱/审批；agent=每个会话贡献的工具/prompt）的划分对 gaea 的 manifest 设计有直接参考：**板块 manifest 声明“能力贡献”，共享设施（引擎、注册表、存储）不在 manifest 内**。

### 5.5 gaea AgentPreset Go 骨架（模式预设的轻量形态）

```go
// internal/gaea/preset/preset.go —— 建议新增（二期编程板块试点时启用）
type AgentPreset struct {
    ID          string            `json:"id"`          // 目录名/配置 id
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Prompt      string            `json:"prompt"`      // 支持 {{model}}/{{cwd}} 变量
    Tools       []string          `json:"tools"`       // 允许的工具名集合（restrict 掩码语义）
    Config      map[string]any    `json:"config"`      // 各工具的会话级配置
}

// 装配：创建会话时把 preset 实例化到该会话的注册空间
func ComposeSession(global *tool.Registry, p AgentPreset) *tool.ScopedRegistry {
    // 等价 ScopedLayers：global 层 + 会话覆盖层；restrict 掩码决定可见工具集
}

// 日志记录：会话创建头记 AgentPresetID；切换时追加 agent-preset/selected 事件
// 恢复：RestoreSession 先从日志折叠 preset id，再按 id 装配
```

要点：ComposeSession 是纯函数（global + preset → 会话注册空间），会话与预设的绑定**入日志**，恢复时重演同一组合——与 DSH 的 resolveSessionPreset 同构。编程板块试点只需：一个 standard 预设（办公引擎现有工具集）+ 一个 code 预设（办公工具 + grep/workspace/todo），manifest 新页面，零引擎改动。

---

## 6. 装配与捆绑（profiles / bundles / patch）

### 6.1 原理

1. **profile = 一个目录**：$DSH_HOME/profiles/<name>/，内含 package.json（dsh.profile.bundles 有序 bundle 列表，boot/app-boot/src/profile.ts:48）+ cordis.patch.yml（用户自己的补丁层，:39）；web/headless 是两个内置模板（:114）。
2. **bundle = 一个 npm 包**：package.json 声明 dsh.bundle.patch: "./cordis.patch.yml"（:24-32 DshBundleManifest）；dsh-base（packages/bundle/base/cordis.patch.yml:15 一个巨型 insert）是每个 profile 的第一层：模型适配器、工具、持久化、sandbox、审批、设置、凭据、遥测。
3. **装配 = 空 entry list 上按序叠 patch 层**：bundle 层（按 dsh.profile.bundles 顺序）→ profile 自身 patch → home 级用户层 → --patch 覆盖层（profile-compose.ts:126 composeProfile；热重载时 composeLivePatches :168 重排同样的栈）。patch 语义：按 id 定位行，**整行替换 config**（不合并）或 insert 新行，后写者胜；**行顺序无装载语义，激活由服务可用性驱动**（base/cordis.patch.yml 注释）。
4. **命令行入口也是装配的一部分**：web-startup 把 --host/--port/--trusted-host 解析成服务 WEB_STARTUP_SERVICE（bundle/web-app/src/startup.ts:20），其他行注入读取——“启动参数以服务形式进入运行树”。
5. **layers apply in order 是可审计事实**：docs/architecture.md:15-25；dsh --profile web --dump-config 输出实际启动的行树，任何行可被用户 patch 覆盖。

### 6.2 关键源码位置

- profile：packages/boot/app-boot/src/profile.ts:39/48/114
- 组合：packages/boot/app-boot/src/profile-compose.ts:126/168
- bundle 补丁：packages/bundle/base/cordis.patch.yml:15、packages/bundle/web-app/cordis.patch.yml、packages/bundle/headless/cordis.patch.yml
- 启动参数服务化：packages/bundle/web-app/src/startup.ts:20
- 导读：docs/architecture.md:15-25

### 6.3 对 gaea 3.0 的映射建议（对应设计 §5.2/§5.4）

- gaea 的 Board Manifest 可以从 DSH 借三个概念：(a) **层间覆盖**——板块 manifest 提供“默认 config”，用户/部署层可整段覆盖（等价 patch 层），manifest 合并规则 = 后层整块替换，简单可解释；(b) **启动自检**——装配后对每个 manifest 断言“intent→handler、page→组件、bindings→门面”完整，缺即启动报错（对应 gaea §5.2 验收与缺陷 2 的机器保证）；(c) **清单 dump**——提供 GetBoardManifests() + 一份“实际装配结果”输出用于审计，对应 DSH 的 dump-config。
- “行顺序无装载语义，激活由依赖可用性驱动” gaea **裁剪**：保持显式初始化顺序（app.Startup 注释化），不引入依赖图推理。
- 命令行参数服务化（web-startup 模式）可借鉴：gaea 的 httpbridge 端口/调试开关等启动参数以单一 struct 注入各板块，而不是各板块各自读 env。
- 热重载（HMR、composeLivePatches）gaea **明确不适用**（设计 §2.2 非目标），但“同栈重排”的纯函数形态（给定 layers 输出 entry list）值得保留，便于测试与审计。

### 6.4 profile/bundle 装配数据形态（给 gaea manifest 的参考 schema）

DSH 的装配事实可压缩为三句话：**profile 是清单（bundles 列表 + 自己的补丁），bundle 是补丁包（patch 文件），装配 = 按序叠补丁**。

```yaml
# $DSH_HOME/profiles/web/package.json（profile 清单）
{"dsh": {"profile": {"bundles": ["@deepseek-ai/dsh-base", "@deepseek-ai/dsh-web-app"]}}}

# @deepseek-ai/dsh-base/package.json（bundle 声明）
{"dsh": {"bundle": {"patch": "./cordis.patch.yml"}}}

# cordis.patch.yml（补丁内容：insert 新行，或按 id 覆盖 config）
- insert:
    - id: session-persistence-jsonl
      name: "@deepseek-ai/dsh-session-persistence-jsonl"
      config: {root: !!js dshHomePath("sessions")}

# 用户覆盖（profile 自己的 cordis.patch.yml）：按 id 整行替换
- id: session-persistence-jsonl
  config: {root: "D:/custom/sessions"}
```

### 6.5 gaea 的对应：manifest 合并规则与装配审计

gaea 不引入 YAML/补丁引擎，但**层间覆盖**可以直接映射为 Go 结构：

```go
// internal/app/board/manifest.go —— 建议形态
type BoardManifest struct {
    ID       string          `json:"id"`
    Name     string          `json:"name"`
    Icon     string          `json:"icon"`
    Page     string          `json:"page"`             // 前端 lazy 组件 key
    Nav      NavSpec         `json:"nav"`              // order/shortcut
    Bindings []string        `json:"bindings"`         // 门面名（gen_bindings 校验用）
    Intents  []IntentDecl    `json:"intents"`          // {id, handler}
    Tools    []string        `json:"tools"`            // 参与会话的工具名
    Config   json.RawMessage `json:"configSchema"`     // 默认配置
}

// 层间覆盖：defaults 内嵌于代码/文件，userOverrides 来自用户配置（map[id]json.RawMessage）
// 合并规则 = 整块替换（不 merge），与 DSH patch 一致
func MergeLayer(base []BoardManifest, overrides map[string]json.RawMessage) ([]BoardManifest, error)

// 装配后自检：每个 intent 的 handler 必须存在；每个 page 必须在 PageRegistry；
// 每个 binding 方法必须在绑定面清单里 —— 缺任一即启动报错（缺陷 2 的机器保证）
func AssertManifests(manifests []BoardManifest) error
```

额外借用：`GetBoardManifests()` 之外的“实际装配结果 dump”（含覆盖后 config）供审计与前端调试；gen_bindings 从 manifest 生成门面清单 + 完整性测试（设计 §5.2 已有）。

---

## 7. 前端 UI 装配（slot 机制 + ConversationNodeDefinition）

### 7.1 原理

1. **slot = 声明式插槽注册表**：SlotMap 接口经 TS 声明合并扩展（client/ui-slots/src/index.ts:24）；register 贡献一个组件 + 可选子槽声明 + store 席位 + locale 命名空间；槽种类 single | list | keyed | chain（:88）、作用域 root | session-maybe | session。运行时层（client/runtime/src/client/slots.ts:41）把 root 作为唯一单槽由外壳渲染，ui-layout 的 AppFrame 占据它并声明 sidebar/conversation/details/shell.overlay 子座；store 实例按 handle × scope key 创建（每会话独立实例、会话消亡清理持久化键），注册经 ctx.effect 随 fiber 卸载可逆。
2. **会话 UI = 事件族 → 节点定义**：ConversationNodeDefinition（docs/cookbook/adding-a-conversation-node.md:25 起）——match(event) 提取业务 id 与生命周期角色（start/update），引擎按 (kind, id) 定位 Context，start 一次 / update 每次，publication 控制渲染节流（immediate/animation-frame/none），buildViewNode 产出 keyed 渲染数据，buildLocationData 可把数据发布到 Turn/Step 位置供其他节点消费。**三条摄取路径**：replace（整窗重放）、prepend（补旧页，只重放受影响的 Context）、append（每事件 O(D) 匹配 + 常量时间 Context 查找，禁止扫窗）。
3. **事件族先设计，UI 后注册**：事件契约（review/start|progress|end，稳定业务 id）由宿主业务方在 SessionEventMap 合并声明；客户端插件只 import 类型做投影。UI 从不扫描会话窗口或别的渲染节点——一切经 State + 引擎。
4. **React 绑定很薄**：web-react 的 bind/scoped-slots 把 store snapshot 接成 hooks；web 的 boot/seed/app-shell 只负责装载。业务 UI 包（ui-conversation、ui-sidebar、ui-trajectory 等）都是“注册 + 组件”两件事。

### 7.2 关键源码位置

- slot 核心：packages/client/ui-slots/src/index.ts:24（SlotMap）、:88（SlotKind）、store.ts:44（StoreSpec）；packages/client/runtime/src/client/slots.ts:41（root 槽）
- 会话节点：packages/client/runtime/src/client/conversation/definition-registry.ts:4、conversation-assembler.ts（引擎）；docs/cookbook/adding-a-conversation-node.md:25-100（定义样板与三条摄取路径）
- 壳层：packages/client/ui-layout/src/index.ts、packages/client/web-react/bind.ts、packages/client/web/src/{boot,seed,AppRoot}.tsx

### 7.3 对 gaea 3.0 的映射建议（gaea 前端为 React 静态路由，取轻量形式）

- **不引入 slot 框架**，只借两个概念：(a) **事件族→节点定义表**——前端维护 Record<eventType, {idOf, start, update, render}>，后端事件到达时按 id 增量更新单条 UI 节点（对应 DSH append 路径的“按 key 更新、不重扫窗口”）；这是 gaea 报告 01 建议 6（ChatPage 从日志投影订阅）的具体实现形态。(b) **插槽注册表轻量版**——Record<slotName, Component[]> + 板块 manifest 声明它挂到哪些插槽（如 sidebar.tools、conversation.node），MainLayout 只渲染插槽；成本是一张表 + 一个渲染函数，收益是“加板块不改壳”（对应 gaea 目标 G1）。
- **事件先设计、UI 后注册**：gaea 办公引擎的新事件族（如 docmd/start|progress|end）先定稳定业务 id 契约，前端按 id 增量渲染，不做全量列表重渲染。
- keyed 渲染 + anchorSeq 概念可借鉴：每个 UI 节点带“来源事件 seq”作为排序/去重锚，配合日志分页可做“回滚窗口”。

### 7.4 gaea 轻量形式 TS 骨架（事件族 → 节点定义表）

```ts
// src/events/nodes.ts —— 建议新增（办公板块先行）
// 事件族契约：先由后端业务方定义稳定 id，前端只做投影
type NodeDef<S> = {
  idOf(event: SessionEvent): string | null,   // 业务 id（start/update 共用）
  roleOf(event: SessionEvent): "start" | "update" | null,
  start(event: SessionEvent): S,
  update(state: S, event: SessionEvent): S,
  render(state: S): ReactNode,                  // keyed 渲染：key = id
  anchorSeq(event: SessionEvent): number,       // 来源事件 seq，排序/去重锚
};

// 例：docmd 文档任务节点（对应 DSH review-job 样板）
const docmdDef: NodeDef<DocmdState> = {
  idOf: e => (e.kind === "docmd/start" || e.kind === "docmd/progress" || e.kind === "docmd/end") ? e.payload.taskId : null,
  roleOf: e => e.kind === "docmd/start" ? "start" : e.kind === "docmd/end" ? "update" : e.kind === "docmd/progress" ? "update" : null,
  start: e => ({ id: e.payload.taskId, stage: e.payload.stage, progress: 0 }),
  update: (s, e) => e.kind === "docmd/progress" ? { ...s, progress: e.payload.progress } : { ...s, done: true, summary: e.payload.summary },
  render: s => <DocmdCard state={s} />,
  anchorSeq: e => e.seq,
};

// 增量更新引擎：按 (kind,id) 维护 Map<id, S>，事件到达只更新一条，不重扫列表
class NodeEngine<S> { upsert(def: NodeDef<S>, event: SessionEvent): void }

// 插槽注册表（轻量版 slot）：板块 manifest 声明挂载点
type SlotName = "sidebar.tools" | "conversation.node" | "workspace.panel";
const slots: Record<SlotName, { key: string; render: () => ReactNode }[]> = { /* 由 manifest 注册 */ };
```

收益：ChatPage/办公工作台从“拉全量消息数组渲染”改为“订阅事件流 + 按 id 增量更新”，配合后端日志 readFrom 续读可实现分页补旧；新增板块只需注册定义 + 组件，不动 MainLayout（对应 gaea 目标 G1/G2）。

---

## 8. 外围机制速览（session-query / memory / goal）

| 机制 | 形态 | 与 gaea 3.0 相关的最小可借鉴点 |
|---|---|---|
| session-query（packages/session-query） | SessionQueryEngine（session-query/src/index.ts:81）：读窗（window/readFrom 续读）、事件/会话过滤、全文检索（SQLite 后端，session-query-sqlite）、追踪（trace）、行内文本抽取；全部读自持久化日志，不依赖内存态 | 「从 seq 续读」原语（readFrom）→ gaea 的多前端同步/变更面板可基于“游标 + 续读”实现；检索可后置，不作 3.0 必需 |
| goal（packages/goal/goal） | 会话内目标域：goal/change 日志事件折叠出目标状态（domain.ts:14、fold.ts），goal-round-driver 驱动持续轮次；工具（tool-goal）经权威检查写日志 | 最小点：**目标/任务状态也是“日志折叠”而非独立存储**；gaea 的 todo/任务状态若进日志，恢复与多前端同步自动获得 |
| memory（packages/memory/memory） | 本 checkout 仅有编译产物（lib），无 src；从产物可见：tracks（user/agent）× scopes（global/workspace）、字符预算、SQLite 存储 | 概念点：记忆按“会话无关的全局/工作区”分层 + 预算管控；gaea 记忆中枢的 8 库可对照 tracks/scopes 分层，但实现各做各的，无需照抄 |
| session-title（packages/session/session-title） | 日志折叠标题（session/title 事件，index.ts:94），fallback/LLM/用户三来源互斥 | gaea §5.1 的标题派生可直接采用“最新事件折叠 + fallback 纯函数” |
| session-projection + cache | 注册制投影单位 + ver/seq 检查点缓存（见 §2.1.6） | gaea 的统计/成本投影若需跨重启复用，抄 (key, ver, seq, val) 检查点行即可（ver 不匹配即丢弃） |

共性结论：DSH 的外围机制全部建立在一个底座上——**任何派生状态都是「日志折叠 + 可选检查点」**，没有独立事实源。gaea 3.0 Step 1 只要把这个底座做对，标题/统计/成本/多前端/变更面板全是免费派生品。

---

## 9. 对 gaea 3.0 设计文档的直接输入（逐条：可采纳 / 需裁剪 / 不适用）

> 依据：docs/2026-08-15-gaea3-architecture-design.md（v0.2）的 §5.1-5.4、§9、附对照表。

### 9.1 会话事件日志（§5.1）—— 可采纳（主体照抄，两处增强）

1. 日志格式 {seq, ts, kind, payload} + append-only + 原子追加：**可采纳**；补硬不变量「seq = 已写行数」「payload 无损 JSON」。
2. 检查点 + 压缩协议：**可采纳**；DSH 额外提供 replace/遮蔽语义，gaea 简版（checkpoint + tail 重放）足够，遮蔽语义留待需要“压缩后旧内容仍可追溯”时再加。
3. 事件种类清单：**可采纳并微调**——DSH 的教训是“模型可见输入必须已入日志”是不变量而非约定；建议把 steer/retry/notice 这类“改写了模型视角”的事件明确标为 surface 相关（会影响投影输出），与纯日志事件（usage/统计）分开。
4. **新增建议（DSH 有、gaea 规划无）**：torn-tail 崩溃恢复语义（最后 turn_done 后不完整行截断 + 完整中断 turn 补合成关闭）+ 模型调用前 flush 检查点。成本半天，换来“断电/崩溃后日志仍可回放”。
5. 投影层 projectMessages：**可采纳**；实现为纯函数 + 按 kind 的投影表，接口兼容现有 Session.Messages。
6. 迁移（旧 jsonl 读取兼容、首次保存转新格式）：**可采纳**，与 DSH 的 SESSION_FORMAT_VERSION 拒绝策略一致（版本不匹配宁可拒绝不可错读）。

### 9.2 板块 Manifest（§5.2/§5.4）—— 可采纳（借用 patch 层间覆盖概念）

7. Manifest JSON + GetBoardManifests() + 启动自检：**可采纳**；DSH 佐证：装配即“按序叠层 + 启动时校验”，无特权核心。
8. **借用「层间覆盖」**：manifest 提供默认 config，用户/部署层按 id 整块覆盖（不合并）；比 gaea 现规划的“configSchema 校验”多一层“用户偏好层”，实现上就是一个 map[id]json.RawMessage 覆盖表。
9. 板块初始化顺序：**需裁剪**——DSH 的“行顺序无装载语义、依赖可用性驱动”不适用于 gaea 的静态装配；保持显式顺序 + 启动注释即可。
10. 热重载/运行期动态装配：**不适用**（gaea §2.2 已明确非目标，DSH 的 HMR/composeLivePatches 是 Cordis 特权，无需借鉴）。

### 9.3 Provider Seam（§5.3）—— 可采纳（形态直接照抄）

11. seam 三元组（定义/提供者/消费者）+ 注册表 + 配置驱动：**可采纳**，DSH 的 fs/shell/sandbox/subprocess 四个 seam 就是标准答案；照抄三点：定义包含**事件词汇**、提供者**互斥注册**、**不可用即 fail-closed**。
12. **新增建议**：会话级策略（sandbox 模式/审批策略作为日志事件折叠）——比全局配置更符合 gaea 的“会话可回放”原则，且与 §5.1 无缝衔接。
13. 提供者叠提供者：**可采纳**（命令 seam 上叠沙箱层），但 gaea 当前只有一层需求，可作为 seam 设计的扩展点而非必做。

### 9.4 模式预设（二期编程板块 §9）—— 可采纳（DSH agent-presets 的直接映射）

14. “编程 = 办公引擎 + 模式预设”：**可采纳且被 DSH 验证**——DSH 的 standard/minimal/code/cordis 四预设正是“同一 AgentLoop + 不同 scope 注册”；gaea 的 AgentPreset{prompt, tools, params} + 会话层注册覆盖即可。
15. **预设选择入日志**（header + agent-preset/selected 事件）：**可采纳**，恢复会话时从日志折叠预设再装配；这是“回放即真相”原则在组合层的延伸。
16. scope 多级继承/作用域事件过滤：**需裁剪**（gaea 单层会话足够）；但“全局监听收所有会话事件”保留（审计/多前端）。

### 9.5 前端装配 —— 需裁剪（取轻量形式）

17. slot 框架/ConversationNodeDefinition 引擎：**需裁剪**（gaea 静态路由 + 板块 manifest，不需要声明合并与 slot 运行时）；只取两个概念——事件族→按 id 增量更新的节点定义表、插槽注册表（Record<slotName, Component[]> + manifest 声明挂载点）。
18. “事件先设计、UI 后注册”与 anchorSeq（来源事件 seq 作渲染锚）：**可采纳**，为将来“回滚窗口/分页补旧”留路。

### 9.6 明确不适用清单

19. Cordis 本体（服务定位/依赖注入/可逆注册的实现机制）：不移植（gaea §2.2 非目标），但其**概念**（注册返回 disposer、单决策槽、事件即扩展点）已散入上述各条。
20. HMR/热重载、profile/bundle 的 npm 分发形态、TS 声明合并、session-query 全文检索、goal-round-driver、记忆预算机制：均不适用或后置，不进入 3.0 范围。

### 9.7 一句话总结

gaea 3.0 的四个 Step 在 DSH 中均有同构实现，且 DSH 的**崩溃恢复语义、whole-value 事件规则、单决策槽策略外挂、per-agent 工具可见性、预设入日志**五点是 gaea 规划中缺失但成本低收益高的增量，建议并入设计 v1.0。
