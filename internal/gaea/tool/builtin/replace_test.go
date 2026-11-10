package builtin

// replace 工具测试（D4，dsh-ops ops_replace 蒸馏，2026-11-10）+三表在册检查。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func replaceWorkspace(t *testing.T) (string, replaceTool) {
	t.Helper()
	ws := t.TempDir()
	return ws, replaceTool{workDir: ws, roots: []string{ws}}
}

func rArgs(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReplaceBasicsAndChangedCountsOccurrences(t *testing.T) {
	ws, r := replaceWorkspace(t)
	for _, n := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(ws, n+".txt"), []byte("foo one\nfoo two\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 零命中文件不入计划。
	if err := os.WriteFile(filepath.Join(ws, "z.txt"), []byte("nothing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := r.Execute(context.Background(), rArgs(t, map[string]any{
		"pattern": "foo", "replacement": "bar",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "files=3") || !strings.Contains(out, "changed=6") {
		t.Fatalf("changed 应计替换次数（3 文件×2 处）: %q", out)
	}
	for _, n := range []string{"a", "b", "c"} {
		raw, _ := os.ReadFile(filepath.Join(ws, n+".txt"))
		if string(raw) != "bar one\nbar two\n" {
			t.Fatalf("%s 替换结果错: %q", n, raw)
		}
	}
}

func TestReplaceDryRunZeroChanges(t *testing.T) {
	ws, r := replaceWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, "a.txt"), []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := r.Execute(context.Background(), rArgs(t, map[string]any{
		"pattern": "foo", "replacement": "bar", "dry_run": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dry_run") {
		t.Fatalf("dry_run 回执应带预览口径: %q", out)
	}
	raw, _ := os.ReadFile(filepath.Join(ws, "a.txt"))
	if string(raw) != "foo\n" {
		t.Fatalf("dry_run 不得写盘: %q", raw)
	}
}

func TestReplaceMaxBudgetPending(t *testing.T) {
	ws, r := replaceWorkspace(t)
	for _, n := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(ws, n+".txt"), []byte("foo x\nfoo y\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := r.Execute(context.Background(), rArgs(t, map[string]any{
		"pattern": "foo", "replacement": "bar", "max_replacements": 3,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "changed=3") || !strings.Contains(out, "pending") {
		t.Fatalf("预算帽+pending 记账缺失: %q", out)
	}
	// a/b 全替换（4 处帽下取 3：a 全 2+b 第 1 处？遍历序 a→b→c——a 取 2，
	// b 取 1（部分替换该文件前 1 处），c pending。
	rawB, _ := os.ReadFile(filepath.Join(ws, "b.txt"))
	if string(rawB) != "bar x\nfoo y\n" {
		t.Fatalf("b 应只替换第 1 处（文件内部分替换）: %q", rawB)
	}
}

func TestReplacePlanSizeLimitZeroChanges(t *testing.T) {
	ws, r := replaceWorkspace(t)
	// 造一个替换后会超 32MiB 的计划：大文件（1 处命中）+replacement 巨长。
	var b strings.Builder
	b.WriteString("head foo tail\n")
	b.WriteString(strings.Repeat("x", 1024*1024))
	if err := os.WriteFile(filepath.Join(ws, "big.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	// 另一个正常文件：预检失败必须整次零变更。
	if err := os.WriteFile(filepath.Join(ws, "ok.txt"), []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	huge := strings.Repeat("y", 33*1024*1024)
	_, err := r.Execute(context.Background(), rArgs(t, map[string]any{
		"pattern": "foo", "replacement": huge,
	}))
	if err == nil {
		t.Fatal("计划超限应整次拒绝")
	}
	f, ok := tool.AsFailure(err)
	if !ok || f.Subcode != "PLAN_SIZE_LIMIT" || f.EffectState != tool.EffectNone {
		t.Fatalf("超限语义错: %+v", f)
	}
	raw, _ := os.ReadFile(filepath.Join(ws, "ok.txt"))
	if string(raw) != "foo\n" {
		t.Fatal("预检失败必须零变更（正常文件也不得写）")
	}
}

func TestReplaceSkipsFenceNoiseAndBinary(t *testing.T) {
	ws, r := replaceWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, ".env"), []byte("foo=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "bin.dat"), []byte("foo\x00bin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, ".gaea", "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".gaea", "sessions", "s.jsonl"), []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "main.go"), []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := r.Execute(context.Background(), rArgs(t, map[string]any{
		"pattern": "foo", "replacement": "bar",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "credential-fence skipped 1") || !strings.Contains(out, "binary skipped 1") {
		t.Fatalf("围栏/二进制记账缺失: %q", out)
	}
	env, _ := os.ReadFile(filepath.Join(ws, ".env"))
	if string(env) != "foo=1\n" {
		t.Fatal("凭据文件不得被写")
	}
	sess, _ := os.ReadFile(filepath.Join(ws, ".gaea", "sessions", "s.jsonl"))
	if string(sess) != "foo\n" {
		t.Fatal(".gaea 宿主子树不得被写")
	}
	main, _ := os.ReadFile(filepath.Join(ws, "main.go"))
	if string(main) != "bar\n" {
		t.Fatal("正常文件应被替换")
	}
}

func TestReplaceStalePlanHashConflict(t *testing.T) {
	ws, r := replaceWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, "a.txt"), []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 造「计划过期」：计划哈希锚旧内容，磁盘已被外部改——publishItem 必须
	// conflict 跳过且零写入（阶段一→二的毫秒竞态窗口的确定性单测）。
	stale := replaceFilePlan{
		path: filepath.Join(ws, "a.txt"), rel: "a.txt", count: 1,
		content:  "foo\n",
		revision: RevisionOfBytes([]byte("foo\n")),
	}
	if err := os.WriteFile(filepath.Join(ws, "a.txt"), []byte("foo EXTERNAL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := r.publishItem(context.Background(), stale, "foo", "bar")
	if err == nil {
		t.Fatal("计划哈希不匹配应 conflict")
	}
	f, ok := tool.AsFailure(err)
	if !ok || f.Code != tool.CodeConflict || f.EffectState != tool.EffectNone {
		t.Fatalf("过期计划语义错: %+v", f)
	}
	raw, _ := os.ReadFile(filepath.Join(ws, "a.txt"))
	if string(raw) != "foo EXTERNAL\n" {
		t.Fatal("冲突文件不得被写")
	}
	// 新鲜计划（哈希锚现状）正常发布。
	fresh := replaceFilePlan{
		path: filepath.Join(ws, "a.txt"), rel: "a.txt", count: 1,
		revision: RevisionOfBytes([]byte("foo EXTERNAL\n")),
	}
	if err := r.publishItem(context.Background(), fresh, "foo", "bar"); err != nil {
		t.Fatalf("新鲜计划应发布: %v", err)
	}
	raw2, _ := os.ReadFile(filepath.Join(ws, "a.txt"))
	if string(raw2) != "bar EXTERNAL\n" {
		t.Fatalf("发布结果错: %q", raw2)
	}
}

// TestReplaceThreeTablesInPlace 三表在册（照 cad_ask_test 配方：注册面+
// compactDesc+compactSchema）。
func TestReplaceThreeTablesInPlace(t *testing.T) {
	if _, ok := tool.LookupBuiltin("replace"); !ok {
		t.Fatal("replace 未注册")
	}
	if compactDesc["replace"] == "" {
		t.Fatal("compactDesc 缺 replace")
	}
	if len(compactSchema["replace"]) == 0 {
		t.Fatal("compactSchema 缺 replace")
	}
}
