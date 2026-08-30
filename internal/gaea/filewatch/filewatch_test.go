package filewatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitEvent 在超时内等待一个批次事件。
// 注意：调用方统一传 5s（历史 flaky：沙箱/CI 高负载下 fsnotify 事件投递 + 100ms
// debounce 可能延迟超过 3s，导致首跑假红复跑绿；2026-08-27 质量收敛放宽至 5s）。
func waitEvent(t *testing.T, w *Watcher, timeout time.Duration) Event {
	t.Helper()
	select {
	case ev := <-w.Events():
		return ev
	case <-time.After(timeout):
		t.Fatal("等待事件超时")
		return Event{}
	}
}

func TestWatchFileCreateModifyRemove(t *testing.T) {
	root := t.TempDir()
	w, err := New(root, []string{".git"}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer w.Close()
	if err := w.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !w.Healthy() {
		t.Fatalf("应为健康状态: %v", w.WatchErr())
	}

	// 新增文件
	f1 := filepath.Join(root, "a.md")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	ev := waitEvent(t, w, 5*time.Second)
	if !contains(ev.Changed, "a.md") {
		t.Fatalf("新增事件应含 a.md: %+v", ev)
	}

	// 修改文件
	if err := os.WriteFile(f1, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	ev = waitEvent(t, w, 5*time.Second)
	if !contains(ev.Changed, "a.md") {
		t.Fatalf("修改事件应含 a.md: %+v", ev)
	}

	// 删除文件
	if err := os.Remove(f1); err != nil {
		t.Fatal(err)
	}
	ev = waitEvent(t, w, 5*time.Second)
	if !contains(ev.Removed, "a.md") {
		t.Fatalf("删除事件应含 a.md: %+v", ev)
	}
}

func TestSkipDirsFiltered(t *testing.T) {
	root := t.TempDir()
	// 预创建被跳过目录，验证其事件被过滤
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	w, err := New(root, []string{".git"}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer w.Close()
	if err := w.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	// 被跳过目录内的事件不应产生批次
	if err := os.WriteFile(filepath.Join(root, ".git", "x"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-w.Events():
		t.Fatalf("跳过目录事件不应输出: %+v", ev)
	case <-time.After(600 * time.Millisecond):
	}
}

func TestDirCreateTriggersFull(t *testing.T) {
	root := t.TempDir()
	w, err := New(root, nil, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer w.Close()
	if err := w.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	ev := waitEvent(t, w, 5*time.Second)
	if !ev.Full {
		t.Fatalf("新建目录应触发 Full: %+v", ev)
	}
	// 子目录内文件也应被监听到（addTree 递归加入）
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	ev = waitEvent(t, w, 5*time.Second)
	if !contains(ev.Changed, "sub/b.txt") {
		t.Fatalf("子目录文件事件应含 sub/b.txt: %+v", ev)
	}
}

// TestStormMergesToFull 锁定风暴语义：单个合批窗口内累积超过 stormN 个文件
// 事件 → 该批次标记 Full（建议全量重建）。
//
// flake 治理（2026-08-30，v3.3.0 后二次治理）：旧版 debounce=100ms 且只读
// 第一个事件就断言 Full。Full 是「单批次」语义（filewatch.go loop：timer 自
// 首个事件读起，flush 后计数清零），全量门禁高负载下 fsnotify 投递散布可跨
// 过 100ms 窗口——风暴被劈成 ≤stormN 的多批（按语义合法地不置 Full）或首批
// 非 Full 被立即判死 → 全量 FAIL、隔离 PASS。两处修复（不动生产语义）：
//  ① 合批窗口 100ms→1s：60 次写本身毫秒级完成，只有投递散布能劈窗，1s 是
//     对旧窗口 10 倍余量；等待上限 8s（对窗口 8:1，v3.3.0「显式超时放宽」先例）。
//  ② 断言改「排空到 Full 或 deadline」的条件等待：不因首个非 Full 批次立即
//     判死（散布劈窗时后续批次仍可能 >stormN 置 Full），deadline 内无 Full 才
//     失败——终态条件等待，非固定 sleep，断言不放宽。
func TestStormMergesToFull(t *testing.T) {
	root := t.TempDir()
	w, err := New(root, nil, 1*time.Second)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer w.Close()
	if err := w.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	// 一次性创建超过 stormN 的文件（同批次）
	for i := 0; i < 60; i++ {
		if err := os.WriteFile(filepath.Join(root, fmtFile(i)), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// 条件等待：收到 Full 即通过；非 Full 批次视为投递散布劈出的局部批，继续等
	deadline := time.After(8 * time.Second)
	for {
		select {
		case ev, ok := <-w.Events():
			if !ok {
				t.Fatal("事件通道在收到 Full 前被关闭")
			}
			if ev.Full {
				return // 风暴合并为 Full，语义成立
			}
		case <-deadline:
			t.Fatalf("8s 内未收到 Full 批次——事件风暴应合并为 Full（疑似投递散布持续劈窗）")
		}
	}
}

func TestCloseStopsEvents(t *testing.T) {
	root := t.TempDir()
	w, err := New(root, nil, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := w.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// 关闭后通道应被关闭
	select {
	case _, ok := <-w.Events():
		if ok {
			t.Fatal("关闭后不应再收到事件")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("关闭后事件通道应关闭")
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func fmtFile(i int) string {
	return "f" + string(rune('0'+i%10)) + fmtInt(i) + ".txt"
}

func fmtInt(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return fmtInt(i/10) + string(rune('0'+i%10))
}
