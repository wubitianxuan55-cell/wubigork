package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D8（2026-11-20）：工程图件纪律必须能从项目技能根发现，且正文保留
// 数字溯源、示意/实测边界、确定性排版体检和 vision 语义边界四条核心纪律。
func TestEngineeringFiguresSkillWiring(t *testing.T) {
	st := New(Options{ProjectRoot: "../../..", HomeDir: t.TempDir()})
	sk, ok := st.Read("engineering-figures")
	if !ok {
		t.Fatal("engineering-figures skill not found")
	}
	if sk.Scope != ScopeProject || sk.RunAs != RunInline {
		t.Fatalf("scope/runAs = %s/%s, want project/inline", sk.Scope, sk.RunAs)
	}
	for _, anchor := range []string{
		"数字可追溯", "示意与实测", "layout", "chart_gen", "diagram_gen",
		"mpl_layout_check.py", "vision", "文本-文本", "18%",
	} {
		if !strings.Contains(sk.Body, anchor) {
			t.Errorf("body missing %q", anchor)
		}
	}
	script := filepath.Join(filepath.Dir(sk.Path), "scripts", "mpl_layout_check.py")
	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("layout checker script missing: %v", err)
	}
	for _, anchor := range []string{"check_figure_layout", "box_overflow", "not_checked"} {
		if !strings.Contains(string(data), anchor) {
			t.Errorf("script missing %q", anchor)
		}
	}
}
