package memory

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// ─── subject keys：同空间同键单值，撞键拒绝并点名持有者（v4.378）─────────────

// SQLite 后端：撞键拒绝点名持有者；同名重存（修订）放行；跨空间同键放行；
// 空键不参与约束；Get/List 往返回填 SubjectKey。
func TestSubjectKeyConflictSQLite(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")

	if _, err := s.Save(Memory{
		Name: "pkg-manager", Space: "work", SubjectKey: "Project.PackageManager",
		Description: "包管理器", Type: TypeProject, Body: "本仓库用 pnpm。",
	}); err != nil {
		t.Fatalf("save holder: %v", err)
	}

	// 撞键（大小写/空白归一后同键）：拒绝且错误点名持有者。
	_, err := s.Save(Memory{
		Name: "pm-contradiction", Space: "work", SubjectKey: "  Project.PackageManager ",
		Description: "矛盾事实", Type: TypeProject, Body: "本仓库用 npm。",
	})
	if err == nil || !strings.Contains(err.Error(), "pkg-manager") || !strings.Contains(err.Error(), "subject key") {
		t.Fatalf("held subject must be rejected naming the holder, got %v", err)
	}
	if n := len(s.List()); n != 1 {
		t.Fatalf("rejected save must not write, got %d memories", n)
	}

	// 同名重存 = 修订，放行且值更新。
	if _, err := s.Save(Memory{
		Name: "pkg-manager", Space: "work", SubjectKey: "project.package_manager",
		Description: "包管理器（修订）", Type: TypeProject, Body: "本仓库改用 npm。",
	}); err != nil {
		t.Fatalf("same-name revision must pass: %v", err)
	}
	if m, ok := s.Get("pkg-manager"); !ok || m.Body != "本仓库改用 npm。" || m.SubjectKey != "project.package_manager" {
		t.Fatalf("revision must land with subject key round-tripped, got %+v ok=%v", m, ok)
	}

	// 跨空间同键：互不冲突（单值约束按空间分域）。
	if _, err := s.Save(Memory{
		Name: "play-pkg", Space: "play", SubjectKey: "project.package_manager",
		Description: "乐园包管理器", Type: TypeProject, Body: "play 项目用 yarn。",
	}); err != nil {
		t.Fatalf("same key in another space must pass: %v", err)
	}

	// 清除声明（subject_key 置空重存）：键释放，他人可声明。
	if _, err := s.Save(Memory{
		Name: "pkg-manager", Space: "work",
		Description: "叙事化", Type: TypeProject, Body: "包管理史：先 pnpm 后 npm。",
	}); err != nil {
		t.Fatalf("clearing subject key must pass: %v", err)
	}
	if _, err := s.Save(Memory{
		Name: "new-holder", Space: "work", SubjectKey: "project.package_manager",
		Description: "新持有者", Type: TypeProject, Body: "现在用 bun。",
	}); err != nil {
		t.Fatalf("released subject must be claimable: %v", err)
	}

	// 未声明键的保存不受检查（叙事型事实）。
	if _, err := s.Save(Memory{
		Name: "narrative", Space: "work",
		Description: "叙事", Type: TypeProject, Body: "昨天聊了很久的架构取舍。",
	}); err != nil {
		t.Fatalf("narrative save must pass: %v", err)
	}
}

// 文件后端：render→loadMemory 往返保 subject_key；无键旧文件读回空串。
func TestSubjectKeyFileBackendRoundTrip(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if _, err := s.Save(Memory{
		Name: "release-branch", SubjectKey: "project.release_branch",
		Description: "发布分支", Type: TypeProject, Body: "release 分支为 main。",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if m, ok := s.Get("release-branch"); !ok || m.SubjectKey != "project.release_branch" {
		t.Fatalf("subject key must round-trip through frontmatter, got %+v ok=%v", m, ok)
	}
	// 磁盘形状：metadata.subject_key 嵌套键（frontmatter 扁平化读回）。
	raw, err := readFileString(s.Path("release-branch"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "subject_key: project.release_branch") {
		t.Fatalf("frontmatter must carry subject_key, got:\n%s", raw)
	}

	// 旧文件（无 subject_key）读回空串，不参与冲突。
	if _, err := s.Save(Memory{
		Name: "old-fact", Description: "旧事实", Type: TypeProject, Body: "无键。",
	}); err != nil {
		t.Fatal(err)
	}
	if m, ok := s.Get("old-fact"); !ok || m.SubjectKey != "" {
		t.Fatalf("legacy file must read back with empty subject key, got %+v", m)
	}
}

// remember 工具端到端：subject_key 参数落库 + 撞键错误如实回传模型。
func TestRememberToolSubjectKey(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	tl := NewRememberTool(store, nil)

	if _, err := tl.Execute(context.Background(), []byte(
		`{"name":"style","subject_key":"user.response_style","description":"回复风格","body":"要精炼。"}`)); err != nil {
		t.Fatalf("first save: %v", err)
	}
	_, err := tl.Execute(context.Background(), []byte(
		`{"name":"style-contradiction","subject_key":"user.response_style","description":"矛盾","body":"要长篇。"}`))
	if err == nil || !strings.Contains(err.Error(), "style") {
		t.Fatalf("tool must surface the holder on conflict, got %v", err)
	}
	// 修订（同名）放行。
	if _, err := tl.Execute(context.Background(), []byte(
		`{"name":"style","subject_key":"user.response_style","description":"回复风格","body":"精炼，两段内。"}`)); err != nil {
		t.Fatalf("revision via same name must pass: %v", err)
	}
}

// NormalizeSubjectKey：trim、ASCII 小写、内部空白折叠为 '-'；中文原样保留。
func TestNormalizeSubjectKey(t *testing.T) {
	cases := map[string]string{
		"  Project.PackageManager ": "project.packagemanager",
		"user  response style":      "user-response-style",
		"包管理器":                      "包管理器",
		"":                          "",
		"   ":                       "",
	}
	for in, want := range cases {
		if got := NormalizeSubjectKey(in); got != want {
			t.Fatalf("NormalizeSubjectKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func readFileString(p string) (string, error) {
	b, err := os.ReadFile(p)
	return string(b), err
}
