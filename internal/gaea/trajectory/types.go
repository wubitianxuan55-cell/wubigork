// Package trajectory 从会话事件日志折叠出「轨迹」视图——对齐 DSH
// ui-trajectory 的事件账本语义（packages/client/ui-trajectory）：
// 按轮次组织的扁平记录表（user/header/assistant/tool/compact/turn-end），
// 每条记录带时序（ts/duration）与步骤归属；工具记录保留 parentId 嵌套
// 子工具树；轮间压缩独立成 Between-turns 区段。折叠是纯函数。
package trajectory

// Trajectory 是整份轨迹快照（GaeaTrajectory 返回值）。
type Trajectory struct {
	Ok          bool     `json:"ok"`
	Turns       []Turn   `json:"turns"`
	BetweenTurns []Record `json:"betweenTurns,omitempty"` // 轮次之间的独立压缩等
}

// Turn 是一轮用户回合：粗分割线边界 + 扁平记录列表。
type Turn struct {
	Turn      int      `json:"turn"`
	StartedAt int64    `json:"startedAt,omitempty"`
	// DurationMs 是轮级耗时（ms）= turn_done.Ts − turn_started.Ts 再 ×1000，
	// 对齐 Record.DurationMs 的换算与命名（日志 Ts 为 unix 秒）。仅 turn_done
	// 到达且 Ts > StartedAt 时计算；悬挂轮/时钟异常保持 0，omitempty 省略——
	// v4.26 WorkHeader「历史轮无耗时数据源」欠账：历史轮读盘折叠自带耗时。
	DurationMs int64    `json:"durationMs,omitempty"`
	End        *TurnEnd `json:"end,omitempty"`
	Records    []Record `json:"records"`
}

// EmptyTrajectory 返回全空轨迹（会话/日志不存在时绑定层的早退返回值）。
// Turns 非 nil：Go 的 nil 切片会序列化成 JSON null，前端按数组消费会崩。
func EmptyTrajectory() Trajectory {
	return Trajectory{Ok: true, Turns: []Turn{}}
}

// EmptyAgentNetwork 返回仅含 root 的空网络（早退返回值），Children 非 nil。
func EmptyAgentNetwork() AgentNetwork {
	return AgentNetwork{
		Ok:   true,
		Root: AgentNode{ID: "root", Name: "主 agent", Kind: "root", Status: "completed", Children: []AgentNode{}},
	}
}

// TurnEnd 是 turn-end 记录（doneAt/错误）。
type TurnEnd struct {
	Seq int64  `json:"seq"`
	Ts  int64  `json:"ts"`
	Err string `json:"err,omitempty"`
}

// Record 是轨迹中的一条事件记录（kind 决定哪个子结构生效）。
type Record struct {
	Seq        int64         `json:"seq"`
	Kind       string        `json:"kind"` // user | header | assistant | tool | compact | ask | approval | subagent
	Ts         int64         `json:"ts"`
	DurationMs int64         `json:"durationMs,omitempty"`
	Step       int           `json:"step,omitempty"` // 所属步骤（0=轮级/轮间）
	User       *UserRec      `json:"user,omitempty"`
	Header     *HeaderRec    `json:"header,omitempty"`
	Assistant  *AssistantRec `json:"assistant,omitempty"`
	Tool       *ToolRec      `json:"tool,omitempty"`
	Compact    *CompactRec   `json:"compact,omitempty"`
	Ask        *AskRec       `json:"ask,omitempty"`
	Approval   *ApprovalRec  `json:"approval,omitempty"`
	Subagent   *SubagentRec  `json:"subagent,omitempty"`
}

// UserRec 是一条用户输入记录。
type UserRec struct {
	Text string `json:"text"`
}

// HeaderRec 是一次请求头记录（system prompt + 工具快照 + 相对上一请求的变化）。
type HeaderRec struct {
	System    string `json:"system,omitempty"` // 预览
	ToolCount int    `json:"toolCount"`
	Tokens    int64  `json:"tokens"`
	Change    string `json:"change,omitempty"` // initial | system | tools | system-and-tools
}

// AssistantRec 是一次助手回复记录（文本/推理/用量）。
type AssistantRec struct {
	Text      string `json:"text,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
	Usage     *Usage `json:"usage,omitempty"`
}

// ToolRec 是一次工具调用记录（dispatch+result 合并；parentId 用于嵌套子树）。
type ToolRec struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Args      string `json:"args,omitempty"`
	Output    string `json:"output,omitempty"`
	Err       string `json:"err,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
	Status    string `json:"status"` // ok | error | running
	ParentID  string `json:"parentId,omitempty"`
}

// CompactRec 是一次压缩记录。
type CompactRec struct {
	Trigger string `json:"trigger,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// AskRec 是一次结构化提问记录（ask 工具）。
type AskRec struct {
	Question string `json:"question,omitempty"`
}

// ApprovalRec 是一次工具调用审批记录。
type ApprovalRec struct {
	Tool    string `json:"tool,omitempty"`
	Subject string `json:"subject,omitempty"`
}

// SubagentRec 是一条子代理完成回投记录（v4.26 对话流式重造）：子代理
// （task 等元工具派生）完成时其最终答复文本回投父回合。Ref 为子代理
// transcript 引用（临时子代理为空）；ParentID 是父 task 调用 ID。
type SubagentRec struct {
	Ref      string `json:"ref,omitempty"`
	Text     string `json:"text,omitempty"`
	ParentID string `json:"parentId,omitempty"`
}

// Usage 是一次请求的用量（来自 usage 事件）。
type Usage struct {
	PromptTokens     int64 `json:"promptTokens,omitempty"`
	CompletionTokens int64 `json:"completionTokens,omitempty"`
	CacheHitTokens   int64 `json:"cacheHitTokens,omitempty"`
	CacheMissTokens  int64 `json:"cacheMissTokens,omitempty"`
	ReasoningTokens  int64 `json:"reasoningTokens,omitempty"`
}

// AgentNetwork 是「Agent 网络」视图（dsh-context Agent network 卡）：
// 主 agent 为根，子代理 = 日志里带有子工具记录（parentId 指向它）的
// task/run_skill/explore 等元工具调用；每节点聚合子树工具数/错误/估算
// token/状态。
type AgentNetwork struct {
	Ok     bool      `json:"ok"`
	Window int64     `json:"window"`
	Root   AgentNode `json:"root"`
}

// AgentNode 是一个 agent 节点。
type AgentNode struct {
	ID        string      `json:"id"`   // "root" | 工具调用 ID
	Name      string      `json:"name"` // 主 agent | 子代理工具名
	Kind      string      `json:"kind"` // root | subagent
	Status    string      `json:"status"` // running | completed | error
	Model     string      `json:"model,omitempty"`
	Task      string      `json:"task,omitempty"` // 子代理任务摘要（prompt/description 预览）
	ToolCalls int         `json:"toolCalls"`      // 子树内工具调用数
	Errors    int         `json:"errors"`
	Tokens    int64       `json:"tokens"` // 子树 args+output 估算
	FirstTs   int64       `json:"firstTs,omitempty"`
	LastTs    int64       `json:"lastTs,omitempty"`
	Children  []AgentNode `json:"children,omitempty"`
}
