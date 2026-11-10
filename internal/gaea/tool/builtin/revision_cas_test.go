package builtin

// A3 revision CAS + B1 凭据围栏 + A1' grep 续页游标测试（dsh-ops 蒸馏落地，
// 2026-11-10）。全部 hermetic（temp 工作区，无外部依赖）。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func casWorkspace(t *testing.T) (string, writeFile, editFile) {
	t.Helper()
	ws := t.TempDir()
	return ws, writeFile{workDir: ws, roots: []string{ws}}, editFile{workDir: ws, roots: []string{ws}}
}

func casArgs(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRevisionCASWriteFile(t *testing.T) {
	ws, w, _ := casWorkspace(t)
	// create-only 哨兵：新文件成功
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "a.txt", "content": "v1", "expected_revision": "absent",
	})); err != nil {
		t.Fatalf("absent+新文件应成功: %v", err)
	}
	// create-only 冲突：文件已存在
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "a.txt", "content": "v2", "expected_revision": "absent",
	})); err == nil || !strings.Contains(err.Error(), "REVISION_CONFLICT") {
		t.Fatalf("absent+已存在应 REVISION_CONFLICT: %v", err)
	}
	// 陈旧修订拒绝：错误 sha
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "a.txt", "content": "v2", "expected_revision": strings.Repeat("0", 64),
	})); err == nil || !strings.Contains(err.Error(), "REVISION_CONFLICT") || !strings.Contains(err.Error(), "重读") {
		t.Fatalf("陈旧 sha 应 REVISION_CONFLICT+自愈指引: %v", err)
	}
	// 正确 sha 成功
	rev, err := RevisionSHA256(filepath.Join(ws, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "a.txt", "content": "v2", "expected_revision": rev,
	})); err != nil {
		t.Fatalf("正确 sha 应成功: %v", err)
	}
	// 不传参=零行为变化（可覆盖）
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "a.txt", "content": "v3",
	})); err != nil {
		t.Fatalf("不传 expected_revision 应旧行为: %v", err)
	}
	// 非法长度
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "a.txt", "content": "v4", "expected_revision": "abc",
	})); err == nil || !strings.Contains(err.Error(), "expected_revision 非法") {
		t.Fatalf("非法长度应报错: %v", err)
	}
	_ = ws
}

func TestRevisionCASEditFamily(t *testing.T) {
	ws, w, e := casWorkspace(t)
	me := multiEdit{workDir: ws, roots: []string{ws}}
	el := editLines{workDir: ws, roots: []string{ws}}
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "b.txt", "content": "alpha\nbeta\ngamma",
	})); err != nil {
		t.Fatal(err)
	}
	rev, _ := RevisionSHA256(filepath.Join(ws, "b.txt"))
	stale := strings.Repeat("f", 64)
	// edit_file：陈旧 sha 拒绝且文件零变化
	if _, err := e.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "b.txt", "old_string": "alpha", "new_string": "ALPHA", "expected_revision": stale,
	})); err == nil || !strings.Contains(err.Error(), "REVISION_CONFLICT") {
		t.Fatalf("edit_file 陈旧 sha 应拒绝: %v", err)
	}
	if raw, _ := os.ReadFile(filepath.Join(ws, "b.txt")); string(raw) != "alpha\nbeta\ngamma" {
		t.Fatalf("拒绝后文件应零变化: %q", string(raw))
	}
	// edit_file：正确 sha 成功
	if _, err := e.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "b.txt", "old_string": "alpha", "new_string": "ALPHA", "expected_revision": rev,
	})); err != nil {
		t.Fatalf("edit_file 正确 sha 应成功: %v", err)
	}
	// multi_edit CAS
	rev2, _ := RevisionSHA256(filepath.Join(ws, "b.txt"))
	if _, err := me.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "b.txt", "expected_revision": rev2,
		"edits": []map[string]any{{"old_string": "beta", "new_string": "BETA"}},
	})); err != nil {
		t.Fatalf("multi_edit 正确 sha 应成功: %v", err)
	}
	if _, err := me.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "b.txt", "expected_revision": stale,
		"edits": []map[string]any{{"old_string": "BETA", "new_string": "x"}},
	})); err == nil || !strings.Contains(err.Error(), "REVISION_CONFLICT") {
		t.Fatalf("multi_edit 陈旧 sha 应拒绝: %v", err)
	}
	// edit_lines CAS
	rev3, _ := RevisionSHA256(filepath.Join(ws, "b.txt"))
	if _, err := el.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "b.txt", "start_line": 3, "end_line": 3, "new_content": "GAMMA", "expected_revision": rev3,
	})); err != nil {
		t.Fatalf("edit_lines 正确 sha 应成功: %v", err)
	}
	if _, err := el.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "b.txt", "start_line": 1, "end_line": 1, "new_content": "x", "expected_revision": stale,
	})); err == nil || !strings.Contains(err.Error(), "REVISION_CONFLICT") {
		t.Fatalf("edit_lines 陈旧 sha 应拒绝: %v", err)
	}
}

func TestReadFileRevisionMarker(t *testing.T) {
	ws, w, _ := casWorkspace(t)
	r := readFile{workDir: ws}
	content := "hello\nworld\n"
	if err := os.WriteFile(filepath.Join(ws, "c.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := r.Execute(context.Background(), casArgs(t, map[string]any{"path": "c.txt"}))
	if err != nil {
		t.Fatal(err)
	}
	rev := RevisionOfBytes([]byte(content))
	if !strings.Contains(out, "[revision sha256:"+rev+"]") {
		t.Fatalf("尾标 revision 缺失或不符: %q", out)
	}
	// 尾标与 CAS 闭环：拿尾标值写 expected_revision 成功
	m := strings.TrimPrefix(strings.TrimSuffix(strings.SplitN(out, "[revision sha256:", 2)[1], "]\n"), "")
	if _, err := w.Execute(context.Background(), casArgs(t, map[string]any{
		"path": "c.txt", "content": "changed", "expected_revision": m,
	})); err != nil {
		t.Fatalf("尾标值应能通过 CAS: %v", err)
	}
}

func TestCredentialFenceRules(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{".env", true}, {".env.local", true}, {"config/.env.production", true},
		{".netrc", true}, {".git-credentials", true},
		{".ssh/known_hosts", true}, {"home/user/.ssh/id_rsa", true},
		{".aws/credentials", true}, {".kube/config", true}, {".gnupg/pubring.kbx", true},
		{".docker/config.json", true},
		{"keys/id_rsa", true}, {"backup/server.pem", true}, {"cert.pfx", true},
		{"client.p12", true}, {"host.key", true}, {"tunnel.ppk", true}, {"app.keystore", true},
		// 不该拦的
		{"main.go", false}, {"notes.txt", false}, {"docs/office.docx", false},
		{"env.txt", false}, {"branch/main.go", false}, {"报表.xlsx", false},
		{"keys/说明.md", false},
	}
	for _, c := range cases {
		if _, got := credentialFenceRule(c.path); got != c.want {
			t.Errorf("credentialFenceRule(%q) = %v, want %v", c.path, got, c.want)
		}
	}
	// Windows 反斜杠形态
	if _, got := credentialFenceRule(`home\.ssh\id_rsa`); !got {
		t.Error("反斜杠路径应命中围栏")
	}
}

func TestCredentialFenceReadFile(t *testing.T) {
	ws, _, _ := casWorkspace(t)
	r := readFile{workDir: ws}
	for _, name := range []string{".env", "config/.aws/credentials", "keys/id_rsa"} {
		full := filepath.Join(ws, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("SECRET"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Execute(context.Background(), casArgs(t, map[string]any{"path": name})); err == nil ||
			!strings.Contains(err.Error(), "CREDENTIAL_DENIED") {
			t.Fatalf("read_file %s 应 CREDENTIAL_DENIED: %v", name, err)
		}
	}
	// 普通文件不受影响
	if err := os.WriteFile(filepath.Join(ws, "普通.md"), []byte("正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := r.Execute(context.Background(), casArgs(t, map[string]any{"path": "普通.md"})); err != nil || !strings.Contains(out, "正文") {
		t.Fatalf("普通文件应正常读: %q %v", out, err)
	}
}

func TestGrepOffsetCursorAndFence(t *testing.T) {
	ws, _, _ := casWorkspace(t)
	g := grepTool{workDir: ws}
	// 4 个文件各 2 条命中 → 共 8 条
	for i := 1; i <= 4; i++ {
		name := filepath.Join(ws, "f"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(name, []byte("needle one\nmiddle\nneedle two\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 基线页：默认 max_results=200 → 全 8 条，无游标头（未截断无 offset）
	out, err := g.Execute(context.Background(), casArgs(t, map[string]any{"pattern": "needle"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[grep page]") && !strings.Contains(out, "credential-fence") {
		t.Fatalf("未分页未围栏不应出头行: %q", out)
	}
	// 游标页：offset=3 → 后 5 条，头行带续页指引
	out2, err := g.Execute(context.Background(), casArgs(t, map[string]any{"pattern": "needle", "offset": 3}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "[grep page] offset=3 shown=5") {
		t.Fatalf("游标头行缺失: %q", out2)
	}
	if strings.Count(out2, "needle") < 5 {
		t.Fatalf("游标页应含后 5 条: %q", out2)
	}
	// 游标跨界（文件中间耗尽 skip）：offset=3 时 f2 的第 2 条 needle 应在第 3 页内
	out3, err := g.Execute(context.Background(), casArgs(t, map[string]any{"pattern": "needle", "offset": 3, "max_results": 2}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out3, "shown=2") || !strings.Contains(out3, "offset=5") {
		t.Fatalf("预算截断页头行应带下一游标: %q", out3)
	}
	// 围栏跳过：.env 内也有 needle → 不入结果，头行记账
	if err := os.WriteFile(filepath.Join(ws, ".env"), []byte("needle=secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out4, err := g.Execute(context.Background(), casArgs(t, map[string]any{"pattern": "needle", "max_results": 3}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out4, ".env") {
		t.Fatalf("围栏文件不得入结果: %q", out4)
	}
	if !strings.Contains(out4, "credential-fence skipped 1") {
		t.Fatalf("围栏记账缺失: %q", out4)
	}
}
