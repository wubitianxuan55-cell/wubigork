package app

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/httpbridge"
)

func TestImageGenPressureNote(t *testing.T) {
	cases := []struct {
		name     string
		availMB  uint64
		totalMB  uint64
		wantHit  bool
		wantSubs []string
	}{
		{"正常内存不命中", 16384, 32768, false, nil},
		{"占比命中 12.5%", 4096, 32768, true, []string{"4.0 GB", "32.0 GB"}},
		{"绝对值命中但占比安全", 3000, 8192, true, []string{"2.9 GB"}},
		{"贴线不命中（略高于 15%）", 4916, 32768, false, nil},
		{"贴线命中（略低于 15%）", 4915, 32768, true, nil},
		{"totalMB 为零读数不可信", 2048, 0, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := imageGenPressureNote(tc.availMB, tc.totalMB)
			if tc.wantHit && got == "" {
				t.Fatalf("期望命中，得到空串 (avail=%d total=%d)", tc.availMB, tc.totalMB)
			}
			if !tc.wantHit && got != "" {
				t.Fatalf("期望不命中，得到 %q", got)
			}
			for _, sub := range tc.wantSubs {
				if !strings.Contains(got, sub) {
					t.Fatalf("文案 %q 缺少子串 %q", got, sub)
				}
			}
		})
	}
}

// TestReadSystemMemoryMBWindows 真机读数合理性（仅 Windows 有实现；其余平台
// 生产路径读不到即静默跳过，无需测）。
func TestReadSystemMemoryMBWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows")
	}
	avail, total, ok := readSystemMemoryMB()
	if !ok {
		t.Fatalf("Windows 读内存失败")
	}
	if total == 0 || avail > total {
		t.Fatalf("读数不合理：avail=%d total=%d", avail, total)
	}
}

// TestNoteImageGenMemoryPressure 预检接线两态：命中→恰一帧 imagegen:pressure
//（含文案）；不命中/读不到→零帧。经 httpbridge SSE 捕获 emit。
func TestNoteImageGenMemoryPressure(t *testing.T) {
	orig := systemMemoryReader
	t.Cleanup(func() { systemMemoryReader = orig })

	openStream := func(t *testing.T, a *App) (chan string, func()) {
		t.Helper()
		srv := httptest.NewServer(httpbridge.New(a).Handler())
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/stream?id=imagegen:pressure", nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("打开 SSE: %v", err)
		}
		lines := make(chan string, 64)
		done := make(chan struct{})
		go func() {
			defer close(done)
			br := bufio.NewReader(resp.Body)
			for {
				line, err := br.ReadString('\n')
				if err != nil {
					close(lines)
					return
				}
				select {
				case lines <- line:
				case <-ctx.Done():
					return
				}
			}
		}()
		cleanup := func() {
			cancel()
			resp.Body.Close()
			srv.Close()
			<-done
		}
		return lines, cleanup
	}

	// SSE 帧：首行 data: {"id":"imagegen:pressure"}，payload 在后续 data: 行——
	// 累积读窗口内全部行再断言。
	waitEvent := func(t *testing.T, lines chan string) string {
		t.Helper()
		deadline := time.After(3 * time.Second)
		var acc strings.Builder
		for {
			select {
			case line, ok := <-lines:
				if !ok {
					t.Fatalf("SSE 提前关闭")
				}
				acc.WriteString(line)
				if strings.Contains(acc.String(), "已用时") || strings.Contains(acc.String(), "GB") {
					return acc.String()
				}
			case <-deadline:
				t.Fatalf("3s 内未收到 imagegen:pressure 事件帧（累计 %q）", acc.String())
			}
		}
	}

	assertNoEvent := func(t *testing.T, lines chan string) {
		t.Helper()
		select {
		case line, ok := <-lines:
			if ok && strings.Contains(line, "imagegen:pressure") {
				t.Fatalf("不应发事件，收到 %q", line)
			}
		case <-time.After(700 * time.Millisecond):
		}
	}

	t.Run("命中发事件", func(t *testing.T) {
		systemMemoryReader = func() (uint64, uint64, bool) { return 2048, 32768, true }
		a := &App{core: &core{}}
		lines, cleanup := openStream(t, a)
		defer cleanup()
		a.noteImageGenMemoryPressure()
		frame := waitEvent(t, lines)
		if !strings.Contains(frame, "imagegen:pressure") || !strings.Contains(frame, "2.0 GB") {
			t.Fatalf("事件帧缺 id 或文案： %q", frame)
		}
	})

	t.Run("不命中零帧", func(t *testing.T) {
		systemMemoryReader = func() (uint64, uint64, bool) { return 16384, 32768, true }
		a := &App{core: &core{}}
		lines, cleanup := openStream(t, a)
		defer cleanup()
		a.noteImageGenMemoryPressure()
		assertNoEvent(t, lines)
	})

	t.Run("读不到静默跳过", func(t *testing.T) {
		systemMemoryReader = func() (uint64, uint64, bool) { return 0, 0, false }
		a := &App{core: &core{}}
		lines, cleanup := openStream(t, a)
		defer cleanup()
		a.noteImageGenMemoryPressure()
		assertNoEvent(t, lines)
	})
}
