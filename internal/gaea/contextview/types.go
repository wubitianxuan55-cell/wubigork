// Package contextview 从会话事件日志折叠出「逐请求上下文构成」快照——
// dsh-context 的 Go 移植（Phase A）：当前六分类组成、逐请求趋势、上下文
// 事件、模型可见 surface 节点。折叠是纯函数（无 I/O），可黄金测试。
package contextview

// DefaultRetention 是快照中保留的最近请求数（与 dsh-context retention 同义）。
const DefaultRetention = 200

// Category 是一次模型请求的六分类 token 构成。
type Category struct {
	System    int64 `json:"system"`
	Tools     int64 `json:"tools"`
	User      int64 `json:"user"`
	Inject    int64 `json:"inject"`
	Assistant int64 `json:"assistant"`
	Tool      int64 `json:"tool"`
}

// Total 返回六分类合计。
func (c Category) Total() int64 {
	return c.System + c.Tools + c.User + c.Inject + c.Assistant + c.Tool
}

// Stats 是会话级上下文统计（效果图「上下文统计」卡）。
type Stats struct {
	Turns           int     `json:"turns"`
	Steps           int     `json:"steps"`
	Injects         int     `json:"injects"`
	Compacts        int     `json:"compacts"`
	Prunes          int     `json:"prunes"`
	ToolCalls       int     `json:"toolCalls"`
	Images          int     `json:"images"`
	CacheHitPercent float64 `json:"cacheHitPercent,omitempty"`
	CostEstimate    float64 `json:"costEstimate,omitempty"`
}

// RequestRecord 是日志中的一次模型请求（一根趋势柱）。
type RequestRecord struct {
	Seq            int64    `json:"seq"`
	Ts             int64    `json:"ts"`
	Turn           int      `json:"turn"`
	Step           int      `json:"step"`
	Category       Category `json:"category"`
	BriefUser      string   `json:"briefUser,omitempty"`
	BriefIn        []string `json:"briefIn,omitempty"`
	BriefResp      string   `json:"briefResp,omitempty"`
	PromptTokens   int64    `json:"promptTokens,omitempty"`
	OutputTokens   int64    `json:"outputTokens,omitempty"`
	CacheHitTokens int64    `json:"cacheHitTokens,omitempty"`
	CacheMissTokens int64   `json:"cacheMissTokens,omitempty"`
}

// ContextEvent 是一次上下文变化（注入/压缩/剪枝/切换/模式）。
type ContextEvent struct {
	Kind   string `json:"kind"` // inject | compact | prune | switch | mode
	Seq    int64  `json:"seq"`
	Delta  int64  `json:"delta"`
	Source string `json:"source,omitempty"`
	Turn   int    `json:"turn"`
	Step   int    `json:"step"`
	Ts     int64  `json:"ts"`
}

// SurfaceNode 是一条模型可见的上下文元素（浏览器/归档重建的基础）。
type SurfaceNode struct {
	Seq    int64  `json:"seq"`
	Cat    string `json:"cat"` // system|tools|user|inject|assistant|tool
	Tokens int64  `json:"tokens"`
	Text   string `json:"text,omitempty"` // 截断预览
	Gone   *int64 `json:"gone,omitempty"` // 被压缩/剪枝取代的事件 seq
}

// ContextTimeline 是 context-view 的整值快照（GaeaContextView 返回值）。
type ContextTimeline struct {
	Ok       bool            `json:"ok"`
	Window   int64           `json:"window"`
	Current  Category        `json:"current"`
	Stats    Stats           `json:"stats"`
	Requests []RequestRecord `json:"requests"`
	Events   []ContextEvent  `json:"events"`
	Nodes    []SurfaceNode   `json:"nodes"`
	Archive  []SurfaceNode   `json:"archive"`
}

// EmptyTimeline 返回全空快照（会话/日志不存在时绑定层的早退返回值）。
// 切片一律非 nil：Go 的 nil 切片会序列化成 JSON null，前端按数组消费会崩。
func EmptyTimeline() ContextTimeline {
	return ContextTimeline{
		Ok:       true,
		Requests: []RequestRecord{},
		Events:   []ContextEvent{},
		Nodes:    []SurfaceNode{},
		Archive:  []SurfaceNode{},
	}
}
