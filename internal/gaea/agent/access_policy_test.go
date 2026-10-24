package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/largefile"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func providerCall(name, args string) provider.ToolCall {
	return provider.ToolCall{ID: "test-" + name, Name: name, Arguments: args}
}

func TestAgentReadFileOutsideWorkspaceBlocked(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("TOP-SECRET-CONTENT"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tool.NewRegistry()
	read, ok := tool.LookupBuiltin("read_file")
	if !ok {
		t.Fatal("read_file builtin not registered")
	}
	reg.Add(read)
	a := New(nil, reg, NewSession(""), Options{
		AccessPolicy:  tool.NewPathAccessPolicy(workspace, nil, nil, nil),
		DisableVerify: true,
	}, event.Discard)

	out := a.executeOne(context.Background(), providerCall("read_file", `{"path":"`+filepath.ToSlash(secret)+`"}`))
	if !out.blocked {
		t.Fatalf("outside read should be blocked: %+v", out)
	}
	if strings.Contains(out.output, "TOP-SECRET-CONTENT") {
		t.Fatalf("access denial leaked file content: %q", out.output)
	}
}

func TestAgentEditPrecheckOutsideWorkspaceDoesNotReadContent(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("PRECHECK-SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tool.NewRegistry()
	edit, ok := tool.LookupBuiltin("edit_file")
	if !ok {
		t.Fatal("edit_file builtin not registered")
	}
	reg.Add(edit)
	a := New(nil, reg, NewSession(""), Options{
		AccessPolicy:  tool.NewPathAccessPolicy(workspace, nil, nil, nil),
		DisableVerify: true,
	}, event.Discard)

	args := `{"path":"` + filepath.ToSlash(secret) + `","old_string":"PRECHECK-SECRET","new_string":"x"}`
	out := a.executeOne(context.Background(), providerCall("edit_file", args))
	if !out.blocked {
		t.Fatalf("outside precheck should be blocked: %+v", out)
	}
	if strings.Contains(out.output, "PRECHECK-SECRET") {
		t.Fatalf("precheck denial leaked file content: %q", out.output)
	}
}

func TestAgentWithoutAccessPolicyFailsClosedForPathTools(t *testing.T) {
	workspace := t.TempDir()
	inside := filepath.Join(workspace, "inside.txt")
	if err := os.WriteFile(inside, []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tool.NewRegistry()
	read, _ := tool.LookupBuiltin("read_file")
	reg.Add(read)
	a := New(nil, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	out := a.executeOne(context.Background(), providerCall("read_file", `{"path":"`+filepath.ToSlash(inside)+`"}`))
	if !out.blocked {
		t.Fatalf("missing production policy must fail closed, got %+v", out)
	}
}

func TestAgentAllDeclaredPathToolsConfinement(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	reg := tool.NewRegistry()
	names := []string{
		"grep", "ls", "vision", "format_convert", "chart_gen", "diagram_gen",
		"sidebar_open", "schedule_get", "schedule_apply", "schedule_analyze",
	}
	for _, name := range names {
		tl, ok := tool.LookupBuiltin(name)
		if !ok {
			t.Fatalf("%s builtin not registered", name)
		}
		reg.Add(tl)
	}
	reg.Add(largefile.NewSummarizeToolAt(nil, workspace))
	a := New(nil, reg, NewSession(""), Options{
		AccessPolicy:  tool.NewPathAccessPolicy(workspace, nil, nil, nil),
		DisableVerify: true,
	}, event.Discard)

	outsideSlash := filepath.ToSlash(outside)
	insideSlash := filepath.ToSlash(filepath.Join(workspace, "doc.pdf"))
	cases := []struct {
		name string
		args string
	}{
		{"grep", `{"pattern":"secret","path":"` + outsideSlash + `"}`},
		{"ls", `{"path":"` + outsideSlash + `"}`},
		{"vision", `{"image_path":"` + outsideSlash + `/image.png"}`},
		{"format_convert", `{"path":"` + outsideSlash + `/doc.pdf"}`},
		{"format_convert", `{"path":"` + insideSlash + `","output":"` + outsideSlash + `/doc.md"}`},
		{"chart_gen", `{"labels":["a"],"values":[1],"output":"` + outsideSlash + `/chart.png"}`},
		{"diagram_gen", `{"nodes":[{"id":"a","label":"A"}],"output":"` + outsideSlash + `/diagram.png"}`},
		{"sidebar_open", `{"path":"` + outsideSlash + `/doc.md"}`},
		{"schedule_get", `{"path":"` + outsideSlash + `/plan.gsched.json"}`},
		{"schedule_apply", `{"path":"` + outsideSlash + `/plan.gsched.json","project":{}}`},
		{"schedule_analyze", `{"path":"` + outsideSlash + `/plan.gsched.json"}`},
		{"summarize_file", `{"paths":["` + outsideSlash + `/a.md","` + insideSlash + `"]}`},
	}
	for _, tc := range cases {
		out := a.executeOne(context.Background(), providerCall(tc.name, tc.args))
		if !out.blocked || !strings.Contains(out.output, "path access denied") {
			t.Errorf("%s outside workspace = %+v, want access denial", tc.name, out)
		}
	}
}
