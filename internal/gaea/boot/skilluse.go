package boot

import (
	"context"
	"encoding/json"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// skillUseCountTool 包装 run_skill（7.2-2 判据②）：执行后按工具级成败汇报
// 一次调用。args.name 解析失败不计——宁少勿扰。只包 Execute，其余成员原样
// 转发；CompactDescriptor 的回退语义与注册表缺省回退一致（Description/Schema）。
type skillUseCountTool struct {
	tool.Tool
	onUse func(name string, ok bool)
}

var _ tool.Tool = (*skillUseCountTool)(nil)

func (t *skillUseCountTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Name string `json:"name"`
	}
	name := ""
	if err := json.Unmarshal(args, &in); err == nil {
		name = in.Name
	}
	out, err := t.Tool.Execute(ctx, args)
	if name != "" {
		t.onUse(name, err == nil)
	}
	return out, err
}

func (t *skillUseCountTool) CompactDescription() string {
	if c, ok := t.Tool.(tool.CompactDescriptor); ok {
		return c.CompactDescription()
	}
	return t.Description()
}

func (t *skillUseCountTool) CompactSchema() json.RawMessage {
	if c, ok := t.Tool.(tool.CompactDescriptor); ok {
		return c.CompactSchema()
	}
	return t.Schema()
}
