package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// memoryGetTool reads one saved memory's full body. The file backend's facts
// are plain .md files readable with read_file; the SQLite backend (Hephaestus.db)
// has no readable file, so the model uses memory_get instead.
type memoryGetTool struct{ store Store }

// NewMemoryGetTool returns the `memory_get` tool bound to store.
func NewMemoryGetTool(store Store) tool.Tool { return memoryGetTool{store: store} }

func (memoryGetTool) Name() string { return "memory_get" }

func (memoryGetTool) Description() string {
	return "Read the full content of one saved memory by name (use the name from the memory index). " +
		"When a memory index shows \"[Title](name.md)\", the .md link is a logical reference — " +
		"call memory_get with that name to see the full fact. Do not try to read_file a name.md path."
}

func (memoryGetTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type": "object",
"properties": {
  "name": {"type": "string", "description": "The memory's slug name from the index, e.g. \"prefers-tabs\"."}
},
"required": ["name"]
}`)
}

func (memoryGetTool) ReadOnly() bool { return true }

func (t memoryGetTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(p.Name) == "" {
		return "", fmt.Errorf("name is required")
	}
	// S1.2 B 读端隔离器：读取与触达都限定调用 ctx 的会话空间（缺省 work），
	// 跨空间键不命中——工位会话读不到乐园记忆，反之亦然（验收红线）。
	space := SpaceFromContext(ctx)
	m, ok := t.store.GetInSpace(p.Name, space)
	if !ok {
		return "", fmt.Errorf("memory %q not found", p.Name)
	}
	// 读取即使用：更新 last_used_at，供「关键词+时间+高频」注入排序。
	_ = t.store.TouchInSpace(p.Name, space)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", displayTitle(m.Title, m.Name))
	fmt.Fprintf(&b, "- name: %s\n- type: %s\n- kind: %s\n", m.Name, m.Type, m.Kind)
	if len(m.Tags) > 0 {
		fmt.Fprintf(&b, "- tags: %s\n", strings.Join(m.Tags, ", "))
	}
	b.WriteString("\n")
	b.WriteString(m.Body)
	return b.String(), nil
}
