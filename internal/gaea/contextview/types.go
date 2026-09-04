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
	// Estimated 标记该请求在回合结束时未见 usage 事件，按当前估算分类关闭
	// （旧日志/无 usage 提供方；诚实标注「估算」，不伪造用量数字）。
	Estimated bool `json:"estimated,omitempty"`

	// Delta 是「对比上一步」（dsh-context 同名能力的 Go 折叠版）：该请求相对
	// 上一次请求的模型可见 surface 净变化；首个请求 First=true（基线=空）。
	// 快照点与 Category 同拍（request_header 组装时；旧日志退化在 usage 关闭
	// 时），保证 delta 与分类构成永远自洽。
	Delta *RequestDelta `json:"delta,omitempty"`
}

// CatDelta 是单个分类在「对比上一步」中的净变化（有变化才进 ByCat）。
type CatDelta struct {
	Cat    string `json:"cat"`
	Items  int64  `json:"items,omitempty"`
	Tokens int64  `json:"tokens"`
}

// RequestDelta 是请求间 surface 差分。Signed：+=新增/膨胀，−=移除/瘦身。
// 口径：system/tools 取最新 request_header 的整体估算（节点只在变化时入列，
// 旧头不回收，逐条聚合会重计历史头）；user/inject/assistant/tool 取活节点
// 逐条聚合。两次快照之间发生过压缩（compact）时 Approx=true——差分基线被
// 结构性改写，前端标注「近似」（对齐 dsh 压缩后步骤的诚实标注）。
type RequestDelta struct {
	Items  int64      `json:"items"`            // 净项数变化（+N/−N 条）
	Tokens int64      `json:"tokens"`           // 净 token 变化
	ByCat  []CatDelta `json:"byCat,omitempty"`  // 有变化的分类，按 |tokens| 降序（并列按名称稳定排序）
	Approx bool       `json:"approx,omitempty"` // 跨压缩：基线结构性变化，近似
	First  bool       `json:"first,omitempty"`  // 首个请求：基线=空，差值即全量构成
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

// FileActivity 是一次工具调用对工作区文件的接触（「文件活动」时间线的最小
// 单位：读/写/移动/列目录）。Path 取工具参数里的路径键，纯函数确定性提取；
// 提取不到（如 bash 里的路径）诚实不造数。
type FileActivity struct {
	Seq    int64  `json:"seq"`
	Ts     int64  `json:"ts"`
	Turn   int    `json:"turn"`
	Step   int    `json:"step"`
	Tool   string `json:"tool"`
	Action string `json:"action"` // read | write | move | dir
	Path   string `json:"path"`
}

// ContextTiming 是会话级耗时统计（对齐 dsh-context TimelineState.timing 的
// 诚实近似版）。日志时间戳为秒级（time.Now().Unix()），所有 ms 值是秒粒度
// 近似（差值 ×1000）；日志无法支撑的指标保持零值并整体省略（omitempty），
// 不伪造数字。口径：
//   - WallMs：各轮次（turn_started→turn_done）活跃时长合计，末轮未闭合时
//     以最后一条日志收尾；
//   - TTFTMs：模型等待——步骤起点（request_header，旧日志退化为前一事件
//     ts）→ 该步首条 reasoning/text 增量；无增量的纯工具步以 message 充当
//     首 token（此时 gen 为 0）；
//   - GenMs：生成——首 token → assistant 消息（message/assistant_message）收尾；
//   - Calls：模型调用次数 = message/assistant_message 条数（子代理文本增量
//     wire-only 不落盘，故天然只计主回合调用）；
//   - ToolsMs/ToolCalls：tool_dispatch→tool_result 按 id 配对，ms=收发差值，
//     并行调用重复计（与 dsh 同口径）；子代理工具调用经转发同样入账（与其
//     父 task 工具的挂钟时长重复计，同一并行口径）；未配对（中断轮）不计数；
//   - Tools：每工具名 {calls, ms} 排行——数组按 ms 降序（并列按名称稳定
//     排序）截断 20；wire 用有序数组而非 map：Go map 序列化按键名字典序，
//     无法承载「按 ms 降序」的排行语义。
type ContextTiming struct {
	WallMs    int64        `json:"wallMs,omitempty"`
	TTFTMs    int64        `json:"ttftMs,omitempty"`
	GenMs     int64        `json:"genMs,omitempty"`
	Calls     int          `json:"calls,omitempty"`
	ToolsMs   int64        `json:"toolsMs,omitempty"`
	ToolCalls int          `json:"toolCalls,omitempty"`
	Tools     []ToolTiming `json:"tools,omitempty"`
}

// ToolTiming 是单个工具名的调用次数与执行时长合计（Tools 排行条目）。
type ToolTiming struct {
	Name  string `json:"name"`
	Calls int    `json:"calls"`
	Ms    int64  `json:"ms"`
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
	Files    []FileActivity  `json:"files"`
	// Timing 是耗时统计卡（会话无任何可测时长时为 nil，整体省略——例如
	// 迁移日志全部条目同一秒，差值处处为零，诚实留空）。
	Timing *ContextTiming `json:"timing,omitempty"`
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
		Files:    []FileActivity{},
	}
}
