package control

import (
	"encoding/json"
	"strings"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func TestListProbe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "20261014-deploy.jsonl")
	var legacy strings.Builder
	for _, l := range []struct{ role, content string }{
		{"user", "帮我把部署脚本改成幂等"},
		{"assistant", "ok"},
	} {
		raw, _ := json.Marshal(map[string]string{"role": l.role, "content": l.content})
		legacy.Write(raw)
		legacy.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(legacy.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	infos, err := session.List(dir)
	t.Logf("List err=%v n=%d", err, len(infos))
	for _, i := range infos {
		t.Logf("info: %s turns=%d preview=%q", i.Path, i.Turns, i.Preview)
	}
	if len(infos) == 0 {
		entries, err := os.ReadDir(dir)
		t.Logf("dir entries err=%v:", err)
		for _, e := range entries {
			t.Logf("  %s", e.Name())
		}
	}
}
