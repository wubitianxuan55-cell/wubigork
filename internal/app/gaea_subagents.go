package app

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// SubagentRunView 是「多智能体分工」面板的单条子代理视图（P2，对标
// WorkSwarm 蜂群 / QClaw V2 多 Agent：让用户看到「谁在干什么」）。
type SubagentRunView struct {
	Ref    string `json:"ref"`    // sa_YYYYMMDD_HHMMSS_... 稳定引用
	Status string `json:"status"` // running | completed | failed
	// Kind 区分两类运行：subagent（task/run_skill 派生的真子代理）与
	// model_tool（vision/summarize_file 等本地模型工具的单轮调用）。旧数据
	// 缺省补 subagent。
	Kind string `json:"kind"`
	// Tool 仅 model_tool 填写：触发记录的工具名（vision / summarize_file …）。
	Tool      string   `json:"tool,omitempty"`
	Model     string   `json:"model,omitempty"`
	ToolScope []string `json:"toolScope,omitempty"`
	Task      string   `json:"task"`               // transcript 首条 user 消息（任务摘要）
	Answer    string   `json:"answer"`             // 最后一条 assistant 回答（截断摘要）
	ToolCalls int      `json:"toolCalls"`          // transcript 中工具调用次数
	LastText  string   `json:"lastText,omitempty"` // C2 活动行：最后一段 assistant 文本（运行中实时更新）
	LastTool  string   `json:"lastTool,omitempty"` // C2 活动行：最后一次工具调用摘要（name + 结果头）
	// FollowUpError 是最近一次追问的后台失败原因摘要（v4.66，meta 透传）：
	// 非空 = 该运行最近一次追问失败，会话 tab 轮询据此把乐观气泡转失败态。
	FollowUpError string    `json:"followUpError,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// SubagentRunsView 是当前会话的子代理分工总览。
type SubagentRunsView struct {
	Available bool              `json:"available"` // false = 会话无 subagents 目录（无派发）
	Runs      []SubagentRunView `json:"runs"`
	Total     int               `json:"total"` // 当前 running 数量
	Running   int               `json:"running"`
}

// GaeaSubagentRuns 读取当前会话目录下 subagents/ 的全部子代理 meta + transcript，
// 返回分工列表（按创建时间倒序）。路径经 sessionDirForPath 校验（防穿越）；
// 无 subagents 目录返回 Available=false（前端显示空状态，不报错）。
// 数据源 = agent.SubagentStore 落盘的两件套：
//
//	<sessionDir>/subagents/sa_*.meta.json（状态/模型/工具范围/时间）
//	<sessionDir>/subagents/sa_*.jsonl    （transcript：任务 = 首条 user 消息）
func (a *App) GaeaSubagentRuns(sessionPath string) SubagentRunsView {
	if sessionPath == "" || sessionDirForPath(sessionPath) == "" {
		return SubagentRunsView{}
	}
	dir := filepath.Join(filepath.Dir(filepath.Clean(sessionPath)), "subagents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return SubagentRunsView{Available: false}
		}
		return SubagentRunsView{}
	}

	type metaSide struct {
		Ref       string    `json:"ref"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
		Status    string    `json:"status"`
		Kind      string    `json:"kind,omitempty"`
		Title     string    `json:"title,omitempty"`
		Tool      string    `json:"tool,omitempty"`
		ToolScope []string  `json:"toolScope,omitempty"`
		Model     string    `json:"model,omitempty"`
		// FollowUpError：最近一次追问后台失败摘要（v4.66），空 = 无失败。
		FollowUpError string `json:"followUpError,omitempty"`
	}
	runs := []SubagentRunView{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		ref := strings.TrimSuffix(e.Name(), ".meta.json")
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var m metaSide
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		v := SubagentRunView{
			Ref:           ref,
			Status:        m.Status,
			Kind:          runKind(m.Kind, ref),
			Tool:          m.Tool,
			Model:         m.Model,
			ToolScope:     m.ToolScope,
			FollowUpError: m.FollowUpError,
			CreatedAt:     m.CreatedAt,
			UpdatedAt:     m.UpdatedAt,
		}
		// transcript：任务摘要（首条 user 消息）+ 最后回答 + 工具调用计数 + 活动行
		task, answer, toolCalls, lastText, lastTool := summarizeSubagentTranscript(filepath.Join(dir, ref+".jsonl"))
		v.Task = firstNonEmpty(m.Title, task)
		v.Answer = answer
		v.ToolCalls = toolCalls
		v.LastText = lastText
		v.LastTool = lastTool
		runs = append(runs, v)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt.After(runs[j].CreatedAt) })

	running := 0
	for _, r := range runs {
		if r.Status == "running" {
			running++
		}
	}
	return SubagentRunsView{
		Available: true,
		Runs:      runs,
		Total:     len(runs),
		Running:   running,
	}
}

// runKind 归一记录家族：meta 显式 Kind 优先，旧数据按 ref 前缀补缺省
// （sa_ → subagent；mt_ → model_tool；未知 → subagent）。
func runKind(kind, ref string) string {
	if kind != "" {
		return kind
	}
	if strings.HasPrefix(ref, "mt_") {
		return "model_tool"
	}
	return "subagent"
}

// firstNonEmpty 返回第一个非空串（meta.Title 优先于 transcript 推导任务）。
func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// summarizeSubagentTranscript 从子代理 transcript JSONL 提取：
// 任务摘要（首条 user 消息，截断 120 字）、最后一条 assistant 回答（截断 200 字）、
// 工具调用次数，以及 C2 活动行——最后一段 assistant 文本（截断 160 字）与
// 最后一次工具调用摘要（name + 结果头部，截断 80 字）。
// 错误静默返回空（meta 存在但 transcript 缺失/损坏时面板仍能展示状态）。
func summarizeSubagentTranscript(path string) (task, answer string, toolCalls int, lastText, lastTool string) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", 0, "", ""
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		var m provider.Message
		if err := dec.Decode(&m); err != nil {
			break
		}
		switch m.Role {
		case provider.RoleUser:
			if task == "" && strings.TrimSpace(m.Content) != "" {
				task = truncateRunes(strings.TrimSpace(m.Content), 120)
			}
		case provider.RoleAssistant:
			if strings.TrimSpace(m.Content) != "" {
				answer = truncateRunes(strings.TrimSpace(m.Content), 200)
				lastText = truncateRunes(strings.TrimSpace(m.Content), 160)
			}
		case provider.RoleTool:
			toolCalls++
			name := strings.TrimSpace(m.Name)
			if name == "" {
				name = "tool"
			}
			head := oneLineHead(m.Content, 60)
			if head != "" {
				lastTool = truncateRunes(name+": "+head, 80)
			} else {
				lastTool = truncateRunes(name, 80)
			}
		}
	}
	return task, answer, toolCalls, lastText, lastTool
}

// SubagentTranscriptMessage 是子代理 transcript 中的一条消息（查看器渲染用）。
type SubagentTranscriptMessage struct {
	Role       string              `json:"role"` // system | user | assistant | tool
	Name       string              `json:"name,omitempty"`
	Content    string              `json:"content,omitempty"`
	Reasoning  string              `json:"reasoning,omitempty"`
	ToolCalls  []provider.ToolCall `json:"toolCalls,omitempty"`
	ToolCallID string              `json:"toolCallId,omitempty"`
}

// SubagentTranscriptView 是子代理完整 transcript 视图（Agent 网络节点 →
// 「查看完整 transcript」的数据源；也是会话 tab 追问轮询的通道——meta 侧车的
// 最近追问失败摘要随快照带出，前端凭它把乐观气泡转失败态，v4.66）。
type SubagentTranscriptView struct {
	Ref  string `json:"ref"`
	Task string `json:"task"`
	// FollowUpError 透传 meta 的最近一次追问后台失败原因摘要（omitempty，
	// 空 = 无失败）。
	FollowUpError string                      `json:"followUpError,omitempty"`
	Messages      []SubagentTranscriptMessage `json:"messages"`
}

// GaeaSubagentTranscript 读取当前会话派发的某个子代理的完整 transcript
// （<sessionDir>/subagents/<ref>.jsonl）。ref 必须匹配 sa_ 前缀且仅含安全
// 字符（防路径穿越）；读取失败返回错误——查看器需要区分「没有」与「读不了」
// （与摘要接口的静默降级不同）。
func (a *App) GaeaSubagentTranscript(sessionPath, ref string) (SubagentTranscriptView, error) {
	if sessionPath == "" || !validSubagentRef(ref) || sessionDirForPath(sessionPath) == "" {
		return SubagentTranscriptView{}, errors.New("invalid subagent transcript reference")
	}
	dir := filepath.Join(filepath.Dir(filepath.Clean(sessionPath)), "subagents")
	// meta.Title 优先作任务标题（skill/model_tool 记录的 transcript 首条
	// user 消息可能是技能正文/工具参数，不适合做 UI 标题）；FollowUpError
	// 随 meta 侧车透传（追问失败可感知，v4.66）。
	var metaTitle, metaFollowUpErr string
	if b, err := os.ReadFile(filepath.Join(dir, ref+".meta.json")); err == nil {
		var m struct {
			Title         string `json:"title"`
			FollowUpError string `json:"followUpError"`
		}
		_ = json.Unmarshal(b, &m)
		metaTitle = m.Title
		metaFollowUpErr = m.FollowUpError
	}
	f, err := os.Open(filepath.Join(dir, ref+".jsonl"))
	if err != nil {
		return SubagentTranscriptView{}, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	view := SubagentTranscriptView{Ref: ref, FollowUpError: metaFollowUpErr, Messages: []SubagentTranscriptMessage{}}
	for {
		var m provider.Message
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return SubagentTranscriptView{}, err
		}
		if view.Task == "" && m.Role == provider.RoleUser && strings.TrimSpace(m.Content) != "" {
			view.Task = firstNonEmpty(metaTitle, truncateRunes(strings.TrimSpace(m.Content), 120))
		}
		view.Messages = append(view.Messages, SubagentTranscriptMessage{
			Role:       string(m.Role),
			Name:       m.Name,
			Content:    m.Content,
			Reasoning:  m.ReasoningContent,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		})
	}
	return view, nil
}

// validSubagentRef 校验运行引用：sa_（子代理）或 mt_（本地模型工具）前缀 +
// 仅字母数字/下划线/连字符/点，
// 且长度受限（防路径穿越与超长注入）。
func validSubagentRef(ref string) bool {
	if len(ref) == 0 || len(ref) > 80 ||
		(!strings.HasPrefix(ref, "sa_") && !strings.HasPrefix(ref, "mt_")) {
		return false
	}
	for _, r := range ref {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '_' || r == '-' || r == '.'
		if !ok {
			return false
		}
	}
	return true
}

// oneLineHead 取多行文本的首行且截断（工具结果摘要用，避免把整段结果塞进活动行）。
func oneLineHead(s string, n int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// truncateRunes 按 rune 截断字符串（中文字符按字符计）。
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
