package session

// S2 会话空间（work/play）：目录分区写新读兼容的会话包层测试。
// 覆盖：日志 space 自描述往返、恢复空间校验（fail-closed 防穿越）、
// 旧平铺会话恒按 work 兼容、checkpoint/分支 meta 空间对账、
// 列表三目录枚举（平铺兜底 + 两空间分区）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

func TestSpaceForPath(t *testing.T) {
	cases := map[string]string{
		filepath.Join(`C:\ws`, ".gaea", "sessions", "s.jsonl"):                     spaces.SpaceWork, // 平铺 = work 兼容
		filepath.Join(`C:\ws`, ".gaea", "sessions", "work", "s.jsonl"):             spaces.SpaceWork,
		filepath.Join(`C:\ws`, ".gaea", "sessions", "play", "s.jsonl"):             spaces.SpacePlay,
		filepath.Join(`C:\ws`, ".gaea", "sessions", "archive", "s.jsonl"):          spaces.SpaceWork,
		filepath.Join(`C:\ws`, ".gaea", "sessions", "play", "archive", "s.jsonl"):  spaces.SpacePlay,
		filepath.Join(`C:\ws`, ".gaea", "sessions", "work", "archive", "s.jsonl"):  spaces.SpaceWork,
		"":                                                                         spaces.SpaceWork, // 空路径防御
		filepath.Join(t.TempDir(), "loose.jsonl"):                                  spaces.SpaceWork, // 非会话目录不误判
	}
	for path, want := range cases {
		if got := SpaceForPath(path); got != want {
			t.Errorf("SpaceForPath(%q) = %q, want %q", path, got, want)
		}
	}
}

// TestLogSpaceRoundTrip 验证写入器 space 自描述随行落盘、读端原样读回；
// 空 space（mode=off 形态）不写 space 字段（与旧行为逐字节一致）。
func TestLogSpaceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	playLog := filepath.Join(dir, "play.gaea-log.jsonl")
	w, err := OpenLog(playLog, "", spaces.SpacePlay)
	if err != nil {
		t.Fatalf("OpenLog(play): %v", err)
	}
	if _, err := w.Append(KindUserMessage, userLogPayload{Content: "hi"}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	entries, err := ReadLog(playLog)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Space != spaces.SpacePlay {
		t.Fatalf("entries = %+v, want space=play", entries)
	}

	// 空 space：行内无 space 键（byte 级旧形态）
	plainLog := filepath.Join(dir, "plain.gaea-log.jsonl")
	w, err = OpenLog(plainLog, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(KindUserMessage, userLogPayload{Content: "hi"}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	data, err := os.ReadFile(plainLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"space"`) {
		t.Fatalf("空 space 日志不应包含 space 字段: %s", data)
	}
	entries, err = ReadLog(plainLog)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Space != "" {
		t.Fatalf("plain entries = %+v, want space 为空", entries)
	}
}

// TestRestoreFlatLegacyCompatible 旧平铺会话（无 space 字段的日志 + 检查点）
// 恢复兼容：平铺目录与 work 分区目录下均按 work 降级可读。
func TestRestoreFlatLegacyCompatible(t *testing.T) {
	for _, dir := range []string{
		t.TempDir(), // 平铺
		filepath.Join(t.TempDir(), "sessions", "work"), // work 分区
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		logPath := filepath.Join(dir, "s.gaea-log.jsonl")
		w, err := OpenLog(logPath, "", "") // 旧行为：无 space 字段
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Append(KindUserMessage, userLogPayload{Content: "u1"}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Append(KindAssistantMessage, assistantLogPayload{Text: "a1"}); err != nil {
			t.Fatal(err)
		}
		w.Close()
		msgs, last, err := Restore(filepath.Join(dir, "nope.json"), logPath)
		if err != nil {
			t.Fatalf("Restore(%s): %v", dir, err)
		}
		if len(msgs) != 2 || msgs[0].Content != "u1" || last != 2 {
			t.Fatalf("恢复 = %+v last=%d（旧平铺会话应按 work 兼容可读）", msgs, last)
		}
	}
}

// TestRestoreRejectsForeignSpace 空间穿越守卫（fail-closed）：play 日志落进
// 平铺/work 目录 → 拒绝恢复；放回 play 目录 → 正常恢复。
func TestRestoreRejectsForeignSpace(t *testing.T) {
	// play 分区日志
	playDir := filepath.Join(t.TempDir(), "sessions", "play")
	if err := os.MkdirAll(playDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playLog := filepath.Join(playDir, "s.gaea-log.jsonl")
	w, err := OpenLog(playLog, "", spaces.SpacePlay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(KindUserMessage, userLogPayload{Content: "u1"}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	// 原地恢复：通过
	if msgs, _, err := Restore(filepath.Join(playDir, "nope.json"), playLog); err != nil || len(msgs) != 1 {
		t.Fatalf("play 目录内恢复应通过: msgs=%v err=%v", msgs, err)
	}

	// 穿越：复制到平铺目录 → 拒绝
	flatDir := t.TempDir()
	foreign := filepath.Join(flatDir, "s.gaea-log.jsonl")
	data, err := os.ReadFile(playLog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Restore(filepath.Join(flatDir, "nope.json"), foreign); err == nil {
		t.Fatal("play 日志落进平铺目录应拒绝恢复（fail-closed）")
	} else if !strings.Contains(err.Error(), "不一致") {
		t.Errorf("错误信息应说明空间不一致: %v", err)
	}

	// 穿越：复制到 work 分区目录 → 拒绝
	workDir := filepath.Join(t.TempDir(), "sessions", "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign2 := filepath.Join(workDir, "s.gaea-log.jsonl")
	if err := os.WriteFile(foreign2, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Restore(filepath.Join(workDir, "nope.json"), foreign2); err == nil {
		t.Fatal("play 日志落进 work 分区应拒绝恢复（fail-closed）")
	}
}

// TestRestoreIgnoresForeignCheckpoint 检查点空间对账：与目录归属不一致的
// 检查点按「损坏」处理（丢弃，不阻塞恢复），日志全量重放。
func TestRestoreIgnoresForeignCheckpoint(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, err := OpenLog(logPath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(KindUserMessage, userLogPayload{Content: "u1"}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	// 伪造一个 play 空间的检查点放进平铺目录（对账失败）
	cpPath := filepath.Join(dir, "s.gaea-checkpoint.json")
	cp := Checkpoint{Seq: 1, Space: spaces.SpacePlay,
		Messages: []provider.Message{{Role: provider.RoleUser, Content: "外来投影"}}}
	b, _ := json.Marshal(cp)
	if err := os.WriteFile(cpPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	msgs, last, err := Restore(cpPath, logPath)
	if err != nil {
		t.Fatalf("对账失败的检查点不应阻塞恢复: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "u1" || last != 1 {
		t.Fatalf("应丢弃外来检查点并全量重放日志: msgs=%+v last=%d", msgs, last)
	}

	// 一致的检查点（无 space 字段 → 降级 work）正常参与恢复：
	// Seq=0 不覆盖任何条目 → 消息 = checkpoint(sys) + 日志重放(u1)。
	cp2 := Checkpoint{Seq: 0, Messages: []provider.Message{{Role: provider.RoleSystem, Content: "sys"}}}
	b2, _ := json.Marshal(cp2)
	if err := os.WriteFile(cpPath, b2, 0o644); err != nil {
		t.Fatal(err)
	}
	msgs, _, err = Restore(cpPath, logPath)
	if err != nil || len(msgs) != 2 || msgs[0].Content != "sys" || msgs[1].Content != "u1" {
		t.Fatalf("一致检查点应参与恢复: msgs=%+v err=%v", msgs, err)
	}
}

// TestCheckpointSpaceRoundTrip 检查点 space 字段往返。
func TestCheckpointSpaceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "s.gaea-checkpoint.json")
	if err := WriteCheckpoint(cpPath, 3, []provider.Message{{Role: provider.RoleUser, Content: "u"}}, spaces.SpacePlay); err != nil {
		t.Fatal(err)
	}
	cp, err := ReadCheckpoint(cpPath)
	if err != nil || cp == nil {
		t.Fatalf("ReadCheckpoint: %v", err)
	}
	if cp.Space != spaces.SpacePlay {
		t.Fatalf("checkpoint space = %q, want play", cp.Space)
	}
	data, _ := os.ReadFile(cpPath)
	if !strings.Contains(string(data), `"space":"play"`) {
		t.Fatalf("checkpoint 落盘应包含 space 字段: %s", data)
	}
}

// TestMigrateLegacyToLogCarriesSpace 迁移条目携带空间自描述。
func TestMigrateLegacyToLogCarriesSpace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions", "play")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(dir, "s.jsonl")
	s := New("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	if err := s.Save(sessionPath); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateLegacyToLog(LogPathFor(sessionPath), sessionPath, spaces.SpacePlay); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadLog(LogPathFor(sessionPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Space != spaces.SpacePlay {
			t.Fatalf("迁移条目 space = %q, want play", e.Space)
		}
	}
	// 迁移后的日志必须能通过恢复校验（同目录归属一致）
	msgs, _, err := Restore(filepath.Join(dir, "nope.json"), LogPathFor(sessionPath))
	if err != nil {
		t.Fatalf("迁移后恢复校验应通过: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("恢复 = %+v", msgs)
	}
}

// TestBranchMetaSpace 分支 meta 空间自描述：play 目录新建 meta 带 play，
// 平铺目录按 work 兼容。
func TestBranchMetaSpace(t *testing.T) {
	playDir := filepath.Join(t.TempDir(), "sessions", "play")
	if err := os.MkdirAll(playDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playPath := filepath.Join(playDir, "s.jsonl")
	m, err := EnsureBranchMeta(playPath)
	if err != nil {
		t.Fatal(err)
	}
	if m.Space != spaces.SpacePlay {
		t.Fatalf("play 分支 meta space = %q, want play", m.Space)
	}
	// 往返
	m2, ok, err := LoadBranchMeta(playPath)
	if err != nil || !ok {
		t.Fatalf("LoadBranchMeta: %v", err)
	}
	if m2.Space != spaces.SpacePlay {
		t.Fatalf("分支 meta 落盘 space = %q, want play", m2.Space)
	}

	flatPath := filepath.Join(t.TempDir(), "s.jsonl")
	if m3, err := EnsureBranchMeta(flatPath); err != nil || m3.Space != spaces.SpaceWork {
		t.Fatalf("平铺分支 meta space = %q err=%v, want work", m3.Space, err)
	}
}

// TestListDirEnumeratesSpaces 列表三目录枚举：平铺兜底（标记 work）+
// work/play 分区各自列出，结果按时间新→旧合并。
func TestListDirEnumeratesSpaces(t *testing.T) {
	base := t.TempDir()
	writeSession := func(dir, name string, mod int64) {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		s := New("sys")
		s.Add(provider.Message{Role: provider.RoleUser, Content: name})
		p := filepath.Join(dir, name+".jsonl")
		if err := s.Save(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, time.Unix(mod, 0), time.Unix(mod, 0)); err != nil {
			t.Fatal(err)
		}
	}
	writeSession(base, "flat", 3000)                                         // 平铺兜底
	writeSession(filepath.Join(base, "work"), "inwork", 2000)                // work 分区
	writeSession(filepath.Join(base, "play"), "inplay", 1000)                // play 分区

	infos, err := List(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 3 {
		t.Fatalf("List = %d 项, want 3（平铺兜底 + 两空间分区）: %+v", len(infos), infos)
	}
	// 新→旧
	want := []struct{ name, space string }{
		{"flat", spaces.SpaceWork},
		{"inwork", spaces.SpaceWork},
		{"inplay", spaces.SpacePlay},
	}
	for i, w := range want {
		if filepath.Base(infos[i].Path) != w.name+".jsonl" || infos[i].Space != w.space {
			t.Errorf("infos[%d] = %s (space=%s), want %s (space=%s)",
				i, filepath.Base(infos[i].Path), infos[i].Space, w.name+".jsonl", w.space)
		}
	}

	// 空目录族：不报错、返回空
	if got, err := List(filepath.Join(t.TempDir(), "missing")); err != nil || len(got) != 0 {
		t.Fatalf("缺失目录 List = %+v err=%v, want 空,nil", got, err)
	}
}

// TestListArchivedEnumeratesSpaces 归档按空间分区：平铺 archive 兜底 +
// 各空间目录自己的 archive/。
func TestListArchivedEnumeratesSpaces(t *testing.T) {
	base := t.TempDir()
	writeArchived := func(dir, name string) {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		s := New("sys")
		s.Add(provider.Message{Role: provider.RoleUser, Content: name})
		if err := s.Save(filepath.Join(dir, name+".jsonl")); err != nil {
			t.Fatal(err)
		}
	}
	writeArchived(filepath.Join(base, "archive"), "flatarch")
	writeArchived(filepath.Join(base, "work", "archive"), "workarch")
	writeArchived(filepath.Join(base, "play", "archive"), "playarch")

	infos, err := ListArchived(base)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, in := range infos {
		got[strings.TrimSuffix(filepath.Base(in.Path), ".jsonl")] = in.Space
	}
	if len(got) != 3 {
		t.Fatalf("ListArchived = %d 项, want 3: %+v", len(infos), infos)
	}
	if got["flatarch"] != spaces.SpaceWork || got["workarch"] != spaces.SpaceWork || got["playarch"] != spaces.SpacePlay {
		t.Fatalf("归档空间标记错误: %+v", got)
	}
}

// TestSaveEventModeCarriesSpace 会话 Save 首次迁移日志时携带会话空间。
func TestSaveEventModeCarriesSpace(t *testing.T) {
	playDir := filepath.Join(t.TempDir(), "sessions", "play")
	if err := os.MkdirAll(playDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(playDir, "s.jsonl")
	s := New("sys")
	s.SetLogFormat("event")
	s.SetSpace(spaces.SpacePlay)
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadLog(LogPathFor(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("Save 应迁移产生日志")
	}
	for _, e := range entries {
		if e.Space != spaces.SpacePlay {
			t.Fatalf("迁移日志 space = %q, want play", e.Space)
		}
	}
}
