package control

// S2 会话空间（work/play）：控制器传播链（config → ctrl → session）与
// 空间自描述写入语义的测试。
// 覆盖：New/Resume/SetSessionPath 四点传播、SessionSpace（事件日志 sink 的
// 空间来源）、space.mode=off 平铺+日志不写 space、Fork 落同空间（天然继承）、
// 恢复空间穿越拒绝（fail-closed）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// newSpaceController 装配轻量控制器（事件日志模式 + 指定空间配置）。
func newSpaceController(t *testing.T, dir, space string) (*Controller, *agent.Agent) {
	t.Helper()
	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	c := New(Options{Runner: &fakeStateRunner{}, Executor: exec, Sink: event.Discard, SessionDir: dir,
		LogFormat: "event", Space: space})
	return c, exec
}

// TestSpacePropagationSetSessionPath 验证 SetSessionPath 注入空间自描述：
// work 配置 + 平铺/work 路径 → "work"；play 配置 + play 分区路径 → "play"。
func TestSpacePropagationSetSessionPath(t *testing.T) {
	dir := t.TempDir()
	c, exec := newSpaceController(t, dir, spaces.SpaceWork)
	path := filepath.Join(dir, "s.jsonl") // 平铺目录：按 work 兼容
	c.SetSessionPath(path)
	if got := exec.Session().Space(); got != spaces.SpaceWork {
		t.Fatalf("平铺路径会话 space = %q, want work", got)
	}
	if got := c.SessionSpace(); got != spaces.SpaceWork {
		t.Fatalf("SessionSpace = %q, want work（事件日志 sink 的空间来源）", got)
	}

	// play 分区路径
	playDir := filepath.Join(dir, "sessions", "play")
	playPath := filepath.Join(playDir, "p.jsonl")
	c.SetSessionPath(playPath)
	if got := exec.Session().Space(); got != spaces.SpacePlay {
		t.Fatalf("play 分区路径会话 space = %q, want play（路径归属优先）", got)
	}
	if got := c.SessionSpace(); got != spaces.SpacePlay {
		t.Fatalf("SessionSpace(play) = %q, want play", got)
	}
}

// TestSpaceModeOffPlainLogs space.mode=off（Space=""）：SessionSpace 为空
// （日志/检查点不写 space 字段），恢复侧空值降级 work 语义一致。
func TestSpaceModeOffPlainLogs(t *testing.T) {
	dir := t.TempDir()
	c, exec := newSpaceController(t, dir, "") // "" = space.mode=off
	path := filepath.Join(dir, "s.jsonl")
	c.SetSessionPath(path)
	if got := c.SessionSpace(); got != "" {
		t.Fatalf("mode=off SessionSpace = %q, want 空（日志不写 space 字段）", got)
	}
	if got := exec.Session().Space(); got != "" {
		t.Fatalf("mode=off 会话 space = %q, want 空", got)
	}

	// 落盘形状验证：用户消息行不含 space 键（旧字节形态）
	if err := c.logUserMessage("你好"); err != nil {
		t.Fatalf("logUserMessage: %v", err)
	}
	data, err := os.ReadFile(session.LogPathFor(path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"space"`) {
		t.Fatalf("mode=off 日志不应包含 space 字段: %s", data)
	}
}

// TestSpaceModeOffPlayDirSelfDescribes mode=off 下 play 分区目录仍恒定自
// 描述 play（防止混合空间日志被恢复校验拒绝）。
func TestSpaceModeOffPlayDirSelfDescribes(t *testing.T) {
	dir := t.TempDir()
	c, exec := newSpaceController(t, dir, "") // "" = space.mode=off
	playPath := filepath.Join(dir, "sessions", "play", "p.jsonl")
	c.SetSessionPath(playPath)
	if got := c.SessionSpace(); got != spaces.SpacePlay {
		t.Fatalf("mode=off play 分区 SessionSpace = %q, want play（恒自描述）", got)
	}
	if got := exec.Session().Space(); got != spaces.SpacePlay {
		t.Fatalf("mode=off play 分区会话 space = %q, want play", got)
	}
}

// TestResumeFromDiskSpaceByPath 验证恢复链的空间传播：play 分区会话在
// work 配置下恢复 → 会话空间仍按路径归属为 play（目录是唯一真相源）；
// 旧平铺会话（无 space 字段日志）恢复兼容。
func TestResumeFromDiskSpaceByPath(t *testing.T) {
	dir := t.TempDir()

	// 旧平铺会话（plain 日志）→ 恢复兼容，空间按 work 兼容
	flatPath := filepath.Join(dir, "legacy.jsonl")
	w, err := session.OpenLog(session.LogPathFor(flatPath), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	c, exec := newSpaceController(t, dir, spaces.SpaceWork)
	if _, err := c.ResumeFromDisk(flatPath); err != nil {
		t.Fatalf("旧平铺会话恢复应兼容: %v", err)
	}
	if got := exec.Session().Space(); got != spaces.SpaceWork {
		t.Fatalf("平铺恢复会话 space = %q, want work", got)
	}
	c.Close()

	// play 分区会话在 work 配置下恢复 → 按路径归属写 play
	playDir := filepath.Join(dir, "sessions", "play")
	if err := os.MkdirAll(playDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playPath := filepath.Join(playDir, "p.jsonl")
	w2, err := session.OpenLog(session.LogPathFor(playPath), "", spaces.SpacePlay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w2.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	w2.Close()

	c2, exec2 := newSpaceController(t, dir, spaces.SpaceWork) // 配置 = work
	defer c2.Close()
	if _, err := c2.ResumeFromDisk(playPath); err != nil {
		t.Fatalf("play 会话恢复应通过（目录归属一致）: %v", err)
	}
	if got := exec2.Session().Space(); got != spaces.SpacePlay {
		t.Fatalf("play 恢复会话 space = %q, want play（路径归属优先于配置）", got)
	}
}

// TestResumeFromDiskRejectsSpaceCrossing 空间穿越守卫贯穿控制器恢复链：
// play 日志落进平铺目录 → ResumeFromDisk 拒绝（fail-closed）。
func TestResumeFromDiskRejectsSpaceCrossing(t *testing.T) {
	dir := t.TempDir()
	// 构造 play 空间日志，放到平铺目录（模拟穿越落点）
	w, err := session.OpenLog(filepath.Join(dir, "x.gaea-log.jsonl"), "", spaces.SpacePlay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	c, _ := newSpaceController(t, dir, spaces.SpaceWork)
	defer c.Close()
	if _, err := c.ResumeFromDisk(filepath.Join(dir, "x.jsonl")); err == nil {
		t.Fatal("play 日志落进平铺目录应拒绝恢复（fail-closed 防穿越）")
	}
}

// TestForkInheritsSpace 验证 Fork 天然继承：分支文件落在父会话同目录
//（同空间），分支会话与分支 meta 的空间自描述与父一致。
func TestForkInheritsSpace(t *testing.T) {
	playDir := filepath.Join(t.TempDir(), "sessions", "play")
	if err := os.MkdirAll(playDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 父会话：play 分区，事件日志两轮
	path := filepath.Join(playDir, "parent.jsonl")
	w, err := session.OpenLog(session.LogPathFor(path), "", spaces.SpacePlay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindAssistantMessage, map[string]string{"text": "a1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append("turn_done", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindAssistantMessage, map[string]string{"text": "a2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append("turn_done", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	// 镜像（投影消息）供 Fork 截取
	entries, err := session.ReadLog(session.LogPathFor(path))
	if err != nil {
		t.Fatal(err)
	}
	parent := agent.NewSession("sys")
	parent.SetLogFormat("event")
	parent.SetSpace(spaces.SpacePlay)
	parent.Replace(session.ProjectMessages(entries))

	c, exec := newSpaceController(t, playDir, spaces.SpacePlay)
	defer c.Close()
	c.Resume(parent, path)

	forkPath, err := c.Fork(1)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	// 分支落在父会话同目录（同空间）
	if filepath.Dir(forkPath) != playDir {
		t.Fatalf("分支目录 = %q, want 父目录 %q（Fork 天然继承空间分区）", filepath.Dir(forkPath), playDir)
	}
	// 分支会话空间 = play
	if got := exec.Session().Space(); got != spaces.SpacePlay {
		t.Fatalf("分支会话 space = %q, want play", got)
	}
	// 分支日志条目空间 = play（首次迁移即携带）
	forkEntries, err := session.ReadLog(session.LogPathFor(forkPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(forkEntries) == 0 {
		t.Fatal("分支日志应为空是错误（应随 Save 迁移生成）")
	}
	for _, e := range forkEntries {
		if e.Space != spaces.SpacePlay {
			t.Fatalf("分支日志条目 space = %q, want play", e.Space)
		}
	}
	// 分支 meta 空间 = play
	meta, ok, err := session.LoadBranchMeta(forkPath)
	if err != nil || !ok {
		t.Fatalf("LoadBranchMeta: %v", err)
	}
	if meta.Space != spaces.SpacePlay {
		t.Fatalf("分支 meta space = %q, want play", meta.Space)
	}
	if meta.ParentID != session.BranchID(path) {
		t.Fatalf("分支 ParentID = %q, want %q", meta.ParentID, session.BranchID(path))
	}
	// 分支日志能通过恢复校验（同目录归属一致）
	if _, _, err := session.Restore(session.CheckpointPathFor(forkPath), session.LogPathFor(forkPath)); err != nil {
		t.Fatalf("分支日志恢复校验应通过: %v", err)
	}
}
