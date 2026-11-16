package app

// memory_hub_wenxue_test.go — 记忆中枢图文库只读浏览单测（2026-11-16）。
// 钉死：browse 如实回图文脑 facts（跨书全览）；无 MemoryUserDir 时诚实
// Available=false；总览 WenxueCount 与 browse 计数一致。图文脑 db 目录走
// os.TempDir 自管清理（同 wenxue_character_import_test 先例——t.TempDir 对
// db 文件的严格 RemoveAll 在句柄延迟释放下必误报）。

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/db"
	gaeamemory "github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/modelengine"
	wdb "github.com/gaea/gaea/internal/whisper/db"
)

func newWenxueHubTestApp(t *testing.T) *App {
	t.Helper()
	// 图文脑目录走 override 缝指向自管临时根（os.UserConfigDir 在 Windows
	// 读 %AppData%，不受 USERPROFILE 重定向影响——不 override 会误写真实脑，
	// 2026-11-16 实测教训）。db 目录自管清理（t.TempDir 对 db 文件严格
	// RemoveAll 在句柄延迟释放下必误报，同 wenxue_character_import_test 先例）。
	root := filepath.Join(os.TempDir(), "gaea-wenxue-hub-test-"+t.Name())
	_ = os.RemoveAll(root)
	brainDir := filepath.Join(root, "wenxue-brain")
	prev := wenxueBrainDirOverride
	wenxueBrainDirOverride = brainDir
	t.Cleanup(func() {
		wenxueBrainDirOverride = prev
		_ = wdb.CloseDatabase(brainDir)
		for i := 0; i < 10; i++ {
			if os.RemoveAll(root) == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	})
	c := &core{
		cfg:       &config.Config{},
		engineMgr: modelengine.NewManager("", ""),
		chatStore: chat.NewStore(filepath.Join(root, "chat")),
		charLib:   characterlib.NewStore(filepath.Join(root, "characterlib")),
	}
	t.Cleanup(func() { _ = c.chatStore.Close() })
	t.Cleanup(func() { _ = c.charLib.Close() })
	t.Cleanup(func() { _ = wdb.CloseDatabase(root) })
	a := &App{core: c}
	a.mediaState = &mediaState{core: c, app: a}
	a.projectState = &projectState{core: c, app: a, mu: sync.RWMutex{}}
	a.whisperState = &whisperState{core: c, app: a, whisperDataRoot: filepath.Join(root, "whisper_data")}
	a.assistantMgr = assistant.NewEmpty(root)
	return a
}

func TestGaeaWenxueMemoryBrowse(t *testing.T) {
	a := newWenxueHubTestApp(t)
	dir := wenxueBrainDir()
	store := gaeamemory.SQLiteStoreFor(db.GetDatabase(dir), dir, "")
	if _, err := store.Save(gaeamemory.Memory{
		Name: "linwan-fears-thunder", Title: "林晚怕打雷", Type: gaeamemory.TypeProject,
		Body: "雷雨夜会失眠，次日状态下滑。", Space: "work",
	}); err != nil {
		t.Fatalf("seed fact: %v", err)
	}

	view := a.GaeaWenxueMemoryBrowse()
	if !view.Available {
		t.Fatal("有图文脑时 Available 应为 true")
	}
	if view.Count != 1 || len(view.Facts) != 1 {
		t.Fatalf("count = %d facts = %d, want 1/1", view.Count, len(view.Facts))
	}
	f := view.Facts[0]
	if f.Name != "linwan-fears-thunder" || f.Body != "雷雨夜会失眠，次日状态下滑。" ||
		f.Type != "project" || f.Space != "work" {
		t.Fatalf("fact 回读不符: %+v", f)
	}
	if _, err := time.Parse(time.RFC3339, f.UpdatedAt); err != nil {
		t.Fatalf("updatedAt 应为 RFC3339: %q (%v)", f.UpdatedAt, err)
	}
}

func TestGaeaWenxueMemoryBrowse_EmptyBrainAvailable(t *testing.T) {
	a := newWenxueHubTestApp(t)
	view := a.GaeaWenxueMemoryBrowse()
	if !view.Available {
		t.Fatal("MemoryUserDir 可得时（脑未落任何事实）Available 仍应为 true")
	}
	if view.Count != 0 || len(view.Facts) != 0 {
		t.Fatalf("count = %d, want 0", view.Count)
	}
}

func TestGaeaMemoryHubOverview_WenxueCount(t *testing.T) {
	a := newWenxueHubTestApp(t)
	ov := a.GaeaMemoryHubOverview()
	if ov.WenxueCount != 0 {
		t.Fatalf("空脑 WenxueCount = %d, want 0", ov.WenxueCount)
	}
	dir := wenxueBrainDir()
	store := gaeamemory.SQLiteStoreFor(db.GetDatabase(dir), dir, "")
	for _, name := range []string{"fact-one", "fact-two"} {
		if _, err := store.Save(gaeamemory.Memory{Name: name, Title: name, Type: gaeamemory.TypeProject, Body: "正文。"}); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	if ov := a.GaeaMemoryHubOverview(); ov.WenxueCount != 2 {
		t.Fatalf("WenxueCount = %d, want 2", ov.WenxueCount)
	}
}

// TestWenxueMemory_FactsFromWenxueBrain 面板恢复批回归：WenxueMemory 的
// facts 必须挂图文脑（此前读 hubOfficeStore=混源遗留），不得再掺办公脑数据。
func TestWenxueMemory_FactsFromWenxueBrain(t *testing.T) {
	a := newWenxueHubTestApp(t)
	dir := wenxueBrainDir()
	store := gaeamemory.SQLiteStoreFor(db.GetDatabase(dir), dir, "")
	if _, err := store.Save(gaeamemory.Memory{
		Name: "wx-panel-fact", Title: "面板事实", Type: gaeamemory.TypeProject, Body: "图文脑唯一事实。",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	view := a.WenxueMemory()
	if !view.Available {
		t.Fatal("有图文脑时 Available 应为 true")
	}
	if len(view.Facts) != 1 || view.Facts[0].Name != "wx-panel-fact" {
		t.Fatalf("facts = %+v, want 仅图文脑 1 条", view.Facts)
	}
}
