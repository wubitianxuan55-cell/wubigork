package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/spill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// read_spill 取回泄洪全文（v4.379，dsh spill-policy 蒸馏的取回配套）：工具
// 结果超过内联预算时 executeOne 把原文存进 runner 泄洪库并在内联文本尾部附
// locator，本工具按 id 分页取回。无状态 builtin——泄洪库经 ctx 盖章注入
// （agent.WithSpillStore），缺库（未开启泄洪/测试）如实报不可用。
type readSpill struct{}

func init() { tool.RegisterBuiltin(readSpill{}) }

func (readSpill) Name() string { return "read_spill" }

func (readSpill) Description() string {
	return "Retrieve the full text of a spilled tool result. When a tool output is too large to inline, the agent stores it and shows a locator like \"[spilled] ... saved as spill sp-000042\" — call this tool with that id to read it. Pages with offset (bytes); returns total size and remaining bytes so you can fetch more. Results live for the current session only."
}

func (readSpill) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Spill id from the locator, e.g. sp-000042."},"offset":{"type":"integer","description":"Byte offset to start reading from (0 = start)."},"max_bytes":{"type":"integer","description":"Max bytes to return (default 16384, cap 49152)."}},"required":["id"]}`)
}

// ReadOnly：读会话内存泄洪库，无任何可观察副作用。
func (readSpill) ReadOnly() bool { return true }

func (readSpill) CompactDescription() string { return compactDesc["read_spill"] }
func (readSpill) CompactSchema() json.RawMessage {
	return compactSchema["read_spill"]
}

func (readSpill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		ID       string `json:"id"`
		Offset   int    `json:"offset"`
		MaxBytes int    `json:"max_bytes"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.ID == "" {
		return "", fmt.Errorf("id is required")
	}
	store, ok := spill.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("spill store is not available (tool spill is disabled for this session)")
	}
	chunk, total, remaining, err := store.Retrieve(p.ID, p.Offset, p.MaxBytes)
	if err != nil {
		return "", err
	}
	out := chunk
	if p.Offset == 0 {
		out = fmt.Sprintf("[spill %s | tool output | total %d bytes]\n%s", p.ID, total, chunk)
	}
	if remaining > 0 {
		out += fmt.Sprintf("\n[... %d more bytes — read_spill(id=%q, offset=%d) continues]", remaining, p.ID, p.Offset+len(chunk))
	}
	return out, nil
}
