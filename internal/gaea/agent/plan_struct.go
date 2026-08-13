package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
)

// ParsePlan 尽力把模型输出的计划解析为结构化 Plan。模型按 planSystemPrompt
// 输出严格 JSON；这里容错处理 Markdown 代码围栏与前后杂质。解析失败或内容
// 为空时返回 nil，调用方回退为纯文本计划卡。
func ParsePlan(raw string) *event.Plan {
	s := strings.TrimSpace(raw)
	s = stripPlanFences(s)
	var p event.Plan
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return nil
	}
	// 清洗：去掉空步骤，保证结构可用。
	steps := p.Steps[:0]
	for _, st := range p.Steps {
		st.Title = strings.TrimSpace(st.Title)
		st.Detail = strings.TrimSpace(st.Detail)
		st.Deliverable = strings.TrimSpace(st.Deliverable)
		if st.Title == "" {
			continue
		}
		st.Resources = nonEmpty(st.Resources)
		st.Tools = nonEmpty(st.Tools)
		steps = append(steps, st)
	}
	p.Steps = steps
	p.Questions = nonEmpty(p.Questions)
	p.Goal = strings.TrimSpace(p.Goal)
	if p.Goal == "" && len(p.Steps) == 0 {
		return nil
	}
	return &p
}

// RenderPlanMarkdown 把结构化计划渲染回可读 Markdown（Ask 卡片的纯文本兜底，
// 以及不支持结构化渲染的旧前端）。
func RenderPlanMarkdown(p *event.Plan) string {
	if p == nil {
		return ""
	}
	var b strings.Builder
	if p.Goal != "" {
		fmt.Fprintf(&b, "**任务理解**：%s\n", p.Goal)
	}
	for i, st := range p.Steps {
		fmt.Fprintf(&b, "\n%d. **%s**", i+1, st.Title)
		if strings.TrimSpace(st.Detail) != "" {
			fmt.Fprintf(&b, "：%s", st.Detail)
		}
		if len(st.Resources) > 0 {
			fmt.Fprintf(&b, "\n    - 将读资料：%s", strings.Join(st.Resources, "、"))
		}
		if len(st.Tools) > 0 {
			fmt.Fprintf(&b, "\n    - 将用工具：%s", strings.Join(st.Tools, "、"))
		}
		if strings.TrimSpace(st.Deliverable) != "" {
			fmt.Fprintf(&b, "\n    - 产出物：%s", st.Deliverable)
		}
	}
	if len(p.Questions) > 0 {
		fmt.Fprintf(&b, "\n\n**待确认**：")
		for _, q := range p.Questions {
			fmt.Fprintf(&b, "\n- %s", q)
		}
	}
	return strings.TrimSpace(b.String())
}

// stripPlanFences 去掉 ```json ... ``` 围栏（模型偶尔会带）。
func stripPlanFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func nonEmpty(in []string) []string {
	out := in[:0]
	for _, v := range in {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}
