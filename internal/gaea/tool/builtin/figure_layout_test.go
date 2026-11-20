package builtin

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestFigureLayoutCheckFindsHardDefects runs the same embedded Python checker
// against a synthetic figure with guaranteed text-text overlap and box overflow.
func TestFigureLayoutCheckFindsHardDefects(t *testing.T) {
	requireDiagramPython(t)

	script := figureLayoutPython + `
import json
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import Rectangle

fig, ax = plt.subplots(figsize=(5, 3))
ax.axis("off")
fig.text(0.30, 0.55, "OVERLAP", fontsize=10)
fig.text(0.30, 0.55, "OVERLAP", fontsize=10)
box = Rectangle((0.10, 0.10), 0.15, 0.08, transform=ax.transAxes)
ax.add_patch(box)
text = ax.text(0.10, 0.10, "LABEL TOO LONG FOR NODE BOX", fontsize=10,
               transform=ax.transAxes)
ax.text(1.02, 0.45, "CLIPPED AT AXES EDGE", fontsize=10,
        transform=ax.transAxes, clip_on=True)
report = check_figure_layout(fig, [{"name": "node-a", "text": text, "box": box}])
print(json.dumps(report, ensure_ascii=False))
`
	python, err := lookPathFirst([]string{"python", "python3"})
	if err != nil {
		t.Skip("python unavailable")
	}
	cmd := exec.Command(python, "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python layout check: %v\n%s", err, out)
	}
	var report figureLayoutReport
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("parse report: %v\n%s", err, out)
	}
	if report.Status != "warn" || report.IssueCount < 3 {
		t.Fatalf("report should warn with hard defects: %+v\n%s", report, out)
	}
	types := map[string]bool{}
	for _, issue := range report.Issues {
		types[issue.Type] = true
	}
	if !types["text_overlap"] || !types["box_overflow"] || !types["text_out_of_bounds"] {
		t.Fatalf("missing deterministic checks, got %+v", report.Issues)
	}
}

func TestFormatFigureLayoutKeepsPassAndWarnings(t *testing.T) {
	pass := formatFigureLayout(figureLayoutReport{Status: "ok", CheckedTexts: 9}, "")
	if !strings.Contains(pass, "通过") || !strings.Contains(pass, "9") {
		t.Fatalf("pass receipt = %q", pass)
	}
	warn := formatFigureLayout(figureLayoutReport{
		Status:       "warn",
		CheckedTexts: 12,
		IssueCount:   2,
		Issues: []figureLayoutIssue{
			{Type: "text_overlap", Target: "甲 ↔ 乙", Detail: "overlap=100%"},
			{Type: "font_too_small", Target: "注释", Detail: "5pt"},
		},
	}, "字体缺少 bold")
	for _, want := range []string{"⚠", "文本重叠", "字号过小", "字体缺少 bold"} {
		if !strings.Contains(warn, want) {
			t.Fatalf("warning receipt missing %q: %s", want, warn)
		}
	}
}
