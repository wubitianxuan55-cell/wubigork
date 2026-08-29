package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(multiEdit{}) }

// multiEdit applies several exact replacements to one file in memory, serially
// (each edit sees the result of the previous one), and writes the file back
// once, atomically — only when every edit succeeded. A failed edit leaves the
// file untouched on disk, so the model never has to reason about a half-applied
// batch. Shares the match kernel with edit_file (replaceInContent).
type multiEdit struct {
	roots   []string
	workDir string
}

func (multiEdit) Name() string { return "multi_edit" }

func (multiEdit) Description() string {
	return "Apply multiple exact replacements to one file in a single atomic write. Edits are applied in order (later edits see earlier results); if any edit fails, nothing is written. Same matching rules as edit_file (old_string must be non-empty and unique unless replace_all)."
}

func (multiEdit) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"File path"},
  "edits":{"type":"array","description":"Replacements applied in order","items":{"type":"object","properties":{
    "old_string":{"type":"string","description":"Exact text to replace (must be non-empty)"},
    "new_string":{"type":"string","description":"Replacement text (empty string deletes)"},
    "replace_all":{"type":"boolean","description":"Replace every occurrence of this old_string (default false)"}
  },"required":["old_string","new_string"]}}
},
"required":["path","edits"]
}`)
}

func (multiEdit) ReadOnly() bool { return false }

func (multiEdit) CompactDescription() string     { return compactDesc["multi_edit"] }
func (multiEdit) CompactSchema() json.RawMessage { return compactSchema["multi_edit"] }

func (m multiEdit) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path  string `json:"path"`
		Edits []struct {
			OldString  string `json:"old_string"`
			NewString  string `json:"new_string"`
			ReplaceAll bool   `json:"replace_all"`
		} `json:"edits"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if len(p.Edits) == 0 {
		return "", fmt.Errorf("edits must contain at least one edit")
	}
	p.Path = resolveIn(m.workDir, p.Path)
	if err := confine(m.roots, p.Path); err != nil {
		return "", err
	}
	content, enc, err := readFileEncoded(p.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}
	cur := content
	for i, e := range p.Edits {
		if e.OldString == "" {
			return "", fmt.Errorf("multi_edit[%d]: old_string is required and must be non-empty "+
				"(pass new_string:\"\" to delete text, or use write_file to create a file)", i)
		}
		next, _, err := replaceInContent(cur, e.OldString, e.NewString, e.ReplaceAll)
		if err != nil {
			// Nothing has been written — the on-disk file still has the
			// original content, so the whole batch is safely retryable.
			return "", fmt.Errorf("multi_edit[%d] failed, no changes written: %w", i, err)
		}
		cur = next
	}
	if err := atomicWriteEncoded(p.Path, cur, enc); err != nil {
		return "", fmt.Errorf("write %s: %w", p.Path, err)
	}
	out := fmt.Sprintf("%s: applied %d/%d edits", p.Path, len(p.Edits), len(p.Edits))
	for i := range p.Edits {
		out += fmt.Sprintf("\n#%d ok", i)
	}
	return out, nil
}
