package app

import (
	"encoding/json"
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
	Ref       string    `json:"ref"`        // sa_YYYYMMDD_HHMMSS_... 稳定引用
	Status    string    `json:"status"`     // running | completed | failed
	Model     string    `json:"model,omitempty"`
	ToolScope []string  `json:"toolScope,omitempty"`
	Task      string    `json:"task"`       // transcript 首条 user 消息（任务摘要）
	Answer    string    `json:"answer"`     // 最后一条 assistant 回答（截断摘要）
	ToolCalls int       `json:"toolCalls"`  // transcript 中工具调用次数
	LastText  string    `json:"lastText,omitempty"` // C2 活动行：最后一段 assistant 文本（运行中实时更新）
	LastTool  string    `json:"lastTool,omitempty"` // C2 活动行：最后一次工具调用摘要（name + 结果头）
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
//   <sessionDir>/subagents/sa_*.meta.json（状态/模型/工具范围/时间）
//   <sessionDir>/subagents/sa_*.jsonl    （transcript：任务 = 首条 user 消息）
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
		Ref       string         `json:"ref"`
		CreatedAt time.Time      `json:"createdAt"`
		UpdatedAt time.Time      `json:"updatedAt"`
		Status    string         `json:"status"`
		ToolScope []string       `json:"toolScope,omitempty"`
		Model     string         `json:"model,omitempty"`
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
			Ref:       ref,
			Status:    m.Status,
			Model:     m.Model,
			ToolScope: m.ToolScope,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
		// transcript：任务摘要（首条 user 消息）+ 最后回答 + 工具调用计数 + 活动行
		task, answer, toolCalls, lastText, lastTool := summarizeSubagentTranscript(filepath.Join(dir, ref+".jsonl"))
		v.Task = task
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
