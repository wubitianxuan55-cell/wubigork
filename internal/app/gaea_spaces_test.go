package app

// S2 会话空间（work/play）：app 层三守卫放行、双空间列表 + 平铺兜底、
// mode=off 回退平铺的集成测试。
// 三守卫（漏改=静默拒绝，比崩溃更阴险）：sessionDirForPath / GaeaArchiveSession /
// GaeaPinSession 对 sessions/<space>/ 形态逐条放行验证。

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// writeSpaceSession 在指定目录写一个有效会话（1 个用户回合，turns>0 才被收录），
// 返回会话路径。
func writeSpaceSession(t *testing.T, dir, name, prompt string) string {
	t.Helper()
	writeProjectSession(t, dir, name, prompt, time.Now().Add(-time.Hour))
	return filepath.Join(dir, name+".jsonl")
}

// TestSessionDirForPathAcceptsSpaces 守卫一：sessionDirForPath 接受
// sessions/<space>/ 与 sessions/<space>/archive/ 形态，拒绝逃逸路径。
func TestSessionDirForPathAcceptsSpaces(t *testing.T) {
	root := t.TempDir()
	for _, space := range []string{spaces.SpaceWork, spaces.SpacePlay} {
		inSpace := filepath.Join(root, ".gaea", "sessions", space, "a.jsonl")
		if got := sessionDirForPath(inSpace); got != filepath.Dir(inSpace) {
			t.Errorf("sessionDirForPath(%q) = %q, want 放行空间分区", inSpace, got)
		}
		archived := filepath.Join(root, ".gaea", "sessions", space, "archive", "a.jsonl")
		if got := sessionDirForPath(archived); got != filepath.Dir(archived) {
			t.Errorf("sessionDirForPath(%q) = %q, want 放行空间归档", archived, got)
		}
	}
	// 逃逸形态仍然拒绝：.gaea/work（非 sessions 子目录）、sessions 下自定义子目录
	for _, bad := range []string{
		filepath.Join(root, ".gaea", "work", "a.jsonl"),
		filepath.Join(root, ".gaea", "sessions", "sub", "a.jsonl"),
		filepath.Join(root, ".gaea", "sessions", "work", "sub", "a.jsonl"),
		filepath.Join(root, ".gaea", "sessions", "archive", "work", "a.jsonl"),
	} {
		if got := sessionDirForPath(bad); got != "" {
			t.Errorf("sessionDirForPath(%q) = %q, want 空（非法路径应拒绝）", bad, got)
		}
	}
}

// TestGaeaSpaceGuardsArchivePin 守卫二/三：归档与置顶对 sessions/play/
// 会话放行（此前 Base(dir)!="sessions" 静默拒绝），归档落空间目录自己的
// archive/，恢复回到原空间目录。
func TestGaeaSpaceGuardsArchivePin(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	ga.cfg, ga.ctrl = nil, nil
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	playDir := gaeaConfig.WorkspaceSessionDir(ws, spaces.SpacePlay)
	path := writeSpaceSession(t, playDir, "p1", "play 空间会话")

	a := &App{}
	// 守卫三：置顶放行两段路径，注册表落 play 目录
	if err := a.GaeaPinSession(path, true); err != nil {
		t.Fatalf("GaeaPinSession(play 路径): %v（守卫应放行 sessions/<space>/）", err)
	}
	if !loadPinned(playDir)[filepath.Base(path)] {
		t.Fatal("置顶注册表应写入空间分区目录")
	}
	if err := a.GaeaPinSession(path, false); err != nil {
		t.Fatalf("取消置顶: %v", err)
	}

	// 守卫二：归档放行两段路径，归档落在 play 目录自己的 archive/
	if err := a.GaeaArchiveSession(path); err != nil {
		t.Fatalf("GaeaArchiveSession(play 路径): %v（守卫应放行 sessions/<space>/）", err)
	}
	archivedPath := filepath.Join(playDir, "archive", filepath.Base(path))
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("归档应落在空间分区 archive/: %v", err)
	}
	// 恢复回 play 目录
	got, err := a.GaeaUnarchiveSession(archivedPath)
	if err != nil {
		t.Fatalf("GaeaUnarchiveSession: %v", err)
	}
	if got != path {
		t.Errorf("恢复路径 = %q, want 原空间目录路径 %q", got, path)
	}
}

// TestGaeaListSessionsSpaces 双空间列表：work/play 分区目录各自列出 +
// 平铺目录兜底（平铺内容标记 space=work），spaceId 随条目返回。
func TestGaeaListSessionsSpaces(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	ga.cfg, ga.ctrl = nil, nil
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	base := gaeaConfig.WorkspaceSessionDir(ws, "") // 平铺基目录
	flatPath := writeSpaceSession(t, base, "flat", "旧平铺会话")
	workPath := writeSpaceSession(t, gaeaConfig.WorkspaceSessionDir(ws, spaces.SpaceWork), "inwork", "work 会话")
	playPath := writeSpaceSession(t, gaeaConfig.WorkspaceSessionDir(ws, spaces.SpacePlay), "inplay", "play 会话")

	a := &App{}
	got := a.GaeaListSessions()
	if len(got) != 3 {
		t.Fatalf("GaeaListSessions = %d 项, want 3（平铺兜底 + 两空间各自列出）: %+v", len(got), got)
	}
	byPath := map[string]string{}
	for _, m := range got {
		byPath[m.Path] = m.SpaceID
	}
	if byPath[flatPath] != spaces.SpaceWork {
		t.Errorf("平铺会话 spaceId = %q, want work（旧平铺恒按 work 兼容）", byPath[flatPath])
	}
	if byPath[workPath] != spaces.SpaceWork {
		t.Errorf("work 分区会话 spaceId = %q, want work", byPath[workPath])
	}
	if byPath[playPath] != spaces.SpacePlay {
		t.Errorf("play 分区会话 spaceId = %q, want play", byPath[playPath])
	}

	// 项目分组视图同样覆盖三个目录
	groups := a.GaeaListProjectSessions()
	if len(groups) != 1 || len(groups[0].Sessions) != 3 {
		t.Fatalf("GaeaListProjectSessions = %+v, want 1 组 3 会话（含平铺兜底）", groups)
	}
}

// TestSpaceModeOffFlatFallback mode=off 回退平铺：EffectiveSessionSpace 为空
// → 写路径解析回平铺目录；旧分区数据仍可读（读端三目录枚举与 mode 无关）。
func TestSpaceModeOffFlatFallback(t *testing.T) {
	ws := t.TempDir()

	cfg := gaeaConfig.Default()
	if got := cfg.EffectiveSessionSpace(); got != spaces.SpaceWork {
		t.Fatalf("缺省生效空间 = %q, want work", got)
	}
	if got, want := gaeaConfig.WorkspaceSessionDir(ws, cfg.EffectiveSessionSpace()),
		filepath.Join(ws, ".gaea", "sessions", "work"); got != want {
		t.Fatalf("mode=on 会话目录 = %q, want work 分区 %q", got, want)
	}

	// 配置回退：space.mode=off → "" → 平铺目录（写路径回退点）
	cfg.Space.Mode = "off"
	if got := cfg.EffectiveSessionSpace(); got != "" {
		t.Fatalf("mode=off 生效空间 = %q, want 空", got)
	}
	if got, want := gaeaConfig.WorkspaceSessionDir(ws, cfg.EffectiveSessionSpace()),
		filepath.Join(ws, ".gaea", "sessions"); got != want {
		t.Fatalf("mode=off 会话目录 = %q, want 平铺 %q（整体回退点）", got, want)
	}

	// 回退后旧分区数据仍可读：play 分区会话仍出现在列表（读端不过滤）
	playDir := gaeaConfig.WorkspaceSessionDir(ws, spaces.SpacePlay)
	writeSpaceSession(t, playDir, "legacy-play", "off 时代的 play 会话")
	infos, err := agent.ListSessions(filepath.Join(ws, ".gaea", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, in := range infos {
		if filepath.Base(in.Path) == "legacy-play.jsonl" && in.Space == spaces.SpacePlay {
			found = true
		}
	}
	if !found {
		t.Fatalf("mode=off 下 play 分区存量会话应仍可读: %+v", infos)
	}
}
