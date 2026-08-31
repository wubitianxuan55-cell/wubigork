package browser

// 真机门控测试：GAEA_LIVE_BROWSER_TEST 置值才跑（真启动 headless Edge）。
//   set GAEA_LIVE_BROWSER_TEST=1
//   go test ./internal/gaea/browser/ -run TestBrowserLive -v

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// livePage 测试页：静态文本 + 点击计数按钮 + input 联动回显 + 表单
// （Enter 提交走 onsubmit 不跳转）+ 全局 keydown 记录 + 同源 iframe。
const livePage = `<!doctype html><html><head><meta charset="utf-8"><title>gaea live</title></head><body>
<h1>gaea-live-ok</h1>
<p id="msg">初始状态</p>
<button id="btn" onclick="window.__n=(window.__n||0)+1;document.getElementById('msg').textContent='已点击 '+window.__n">点我</button>
<input id="name" placeholder="姓名" oninput="document.getElementById('echo').textContent='echo:'+this.value">
<span id="echo"></span>
<a id="lnk" href="#sec2">跳转链接</a>
<form id="f" onsubmit="document.getElementById('formmsg').textContent='submitted:'+document.getElementById('q').value;return false;">
  <input id="q" placeholder="搜索">
</form>
<p id="formmsg">form-initial</p>
<p id="keys">keys-initial</p>
<iframe id="inner" src="/frame"></iframe>
<script>
document.addEventListener('keydown',function(e){var k=document.getElementById('keys');if(k)k.textContent='key:'+e.key;});
</script>
</body></html>`

// liveFrame iframe 内页面：文本 + 点击计数按钮 + input 联动回显。
const liveFrame = `<!doctype html><html><head><meta charset="utf-8"><title>gaea frame</title></head><body>
<p id="finner">iframe-内容</p>
<button id="fbtn" onclick="window.__fn=(window.__fn||0)+1;document.getElementById('finner').textContent='iframe点击 '+window.__fn">iframe按钮</button>
<input id="fq" oninput="document.getElementById('fecho').textContent='fecho:'+this.value">
<span id="fecho">fecho-初始</span>
</body></html>`

// readWithRetry 重试读取（iframe 加载可能晚于主文档 load，最多 ~5s）。
func readWithRetry(ctx context.Context, m *Manager, selector, frame string) (string, error) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		res, err := m.Read(ctx, selector, 6000, frame)
		if err == nil {
			return res.Text, nil
		}
		if time.Now().After(deadline) {
			return "", err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func TestBrowserLive(t *testing.T) {
	if os.Getenv("GAEA_LIVE_BROWSER_TEST") == "" {
		t.Skip("set GAEA_LIVE_BROWSER_TEST=1 to run")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/frame" {
			_, _ = io.WriteString(w, liveFrame)
			return
		}
		_, _ = io.WriteString(w, livePage)
	}))
	defer srv.Close()

	m := NewManager(Options{Headless: true, ProbeTimeout: 20 * time.Second})
	defer m.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 导航：真 Edge headless 打开本地页面。
	nav, err := m.Navigate(ctx, srv.URL, 30)
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	if !strings.Contains(nav.Title, "gaea live") {
		t.Errorf("title = %q, want 含 %q", nav.Title, "gaea live")
	}

	// Read：断言页面文本。
	read, err := m.Read(ctx, "", 6000, "")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(read.Text, "gaea-live-ok") || !strings.Contains(read.Text, "初始状态") {
		t.Errorf("read.Text = %q", read.Text)
	}

	// Snapshot：交互元素应带 ref（按钮/输入框/链接）。
	snap, err := m.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	var btnRef, inputRef int
	for _, it := range snap.Items {
		switch {
		case it.Tag == "button" && strings.Contains(it.Text, "点我"):
			btnRef = it.Ref
		case it.Tag == "input" && strings.Contains(it.Text, "姓名"):
			inputRef = it.Ref // 页面有两个 input，按 placeholder 区分 #name
		}
	}
	if btnRef == 0 || inputRef == 0 {
		t.Fatalf("snapshot 未标记到按钮/输入框: %+v", snap.Items)
	}

	// Click：按钮计数状态变化。
	if _, err := m.Click(ctx, btnRef, "", ""); err != nil {
		t.Fatalf("Click(ref=%d): %v", btnRef, err)
	}
	read2, err := m.Read(ctx, "", 6000, "")
	if err != nil {
		t.Fatalf("Read after click: %v", err)
	}
	if !strings.Contains(read2.Text, "已点击 1") {
		t.Errorf("点击后文本 = %q, want 含 已点击 1", read2.Text)
	}

	// Type：input 联动回显（React 兼容 setter 路径 + oninput 事件）。
	if _, err := m.Type(ctx, inputRef, "", "gaea测试", false, ""); err != nil {
		t.Fatalf("Type(ref=%d): %v", inputRef, err)
	}
	read3, err := m.Read(ctx, "", 6000, "")
	if err != nil {
		t.Fatalf("Read after type: %v", err)
	}
	if !strings.Contains(read3.Text, "echo:gaea测试") {
		t.Errorf("输入后文本 = %q, want 含 echo:gaea测试", read3.Text)
	}

	// Click(selector) 兜底路径：再点一次计数 +1。
	if _, err := m.Click(ctx, 0, "#btn", ""); err != nil {
		t.Fatalf("Click(selector): %v", err)
	}
	read4, err := m.Read(ctx, "", 6000, "")
	if err != nil {
		t.Fatalf("Read after 2nd click: %v", err)
	}
	if !strings.Contains(read4.Text, "已点击 2") {
		t.Errorf("二次点击后文本 = %q, want 含 已点击 2", read4.Text)
	}

	// ── 键盘级 Input（browser_press）──
	// Enter 提交表单：聚焦 #q 输入 hello → Press Enter → onsubmit 回显。
	if _, err := m.Type(ctx, 0, "#q", "hello", false, ""); err != nil {
		t.Fatalf("Type(#q): %v", err)
	}
	if _, err := m.Press(ctx, "enter", nil, ""); err != nil {
		t.Fatalf("Press(enter): %v", err)
	}
	formmsg, err := m.Read(ctx, "#formmsg", 6000, "")
	if err != nil {
		t.Fatalf("Read(#formmsg): %v", err)
	}
	if !strings.Contains(formmsg.Text, "submitted:hello") {
		t.Errorf("Enter 提交后 formmsg = %q, want 含 submitted:hello", formmsg.Text)
	}
	// 字符键与 Escape：全局 keydown 记录 e.key。
	if _, err := m.Press(ctx, "z", nil, ""); err != nil {
		t.Fatalf("Press(z): %v", err)
	}
	if _, err := m.Press(ctx, "esc", nil, ""); err != nil {
		t.Fatalf("Press(esc): %v", err)
	}
	keys, err := m.Read(ctx, "#keys", 6000, "")
	if err != nil {
		t.Fatalf("Read(#keys): %v", err)
	}
	if !strings.Contains(keys.Text, "key:Escape") {
		t.Errorf("Escape 按键后 keys = %q, want 含 key:Escape", keys.Text)
	}

	// ── iframe 内交互（frame 参数）──
	// 读取：frame URL 子串定位 iframe → 隔离世界内读文本。
	frameText, err := readWithRetry(ctx, m, "#finner", "/frame")
	if err != nil {
		t.Fatalf("Read(frame): %v", err)
	}
	if !strings.Contains(frameText, "iframe-内容") {
		t.Errorf("frame read = %q, want 含 iframe-内容", frameText)
	}
	// 点击：iframe 内按钮计数。
	if _, err := m.Click(ctx, 0, "#fbtn", "/frame"); err != nil {
		t.Fatalf("Click(frame): %v", err)
	}
	frameText, err = readWithRetry(ctx, m, "#finner", "/frame")
	if err != nil {
		t.Fatalf("Read(frame after click): %v", err)
	}
	if !strings.Contains(frameText, "iframe点击 1") {
		t.Errorf("iframe 点击后 = %q, want 含 iframe点击 1", frameText)
	}
	// 输入：iframe 内 input 联动回显。
	if _, err := m.Type(ctx, 0, "#fq", "hi", false, "/frame"); err != nil {
		t.Fatalf("Type(frame): %v", err)
	}
	frameText, err = readWithRetry(ctx, m, "#fecho", "/frame")
	if err != nil {
		t.Fatalf("Read(frame after type): %v", err)
	}
	if !strings.Contains(frameText, "fecho:hi") {
		t.Errorf("iframe 输入后 = %q, want 含 fecho:hi", frameText)
	}

	// URL 白名单：file: 拒绝。
	if _, err := m.Navigate(ctx, "file:///C:/Windows/win.ini", 5); err == nil {
		t.Error("file: URL 应被拒绝")
	}

	// Snapshot ref 失效守门：重新导航后旧 ref 必须报错。
	if _, err := m.Navigate(ctx, srv.URL, 30); err != nil {
		t.Fatalf("re-Navigate: %v", err)
	}
	if _, err := m.Click(ctx, btnRef, "", ""); err == nil {
		t.Error("跳转后沿用旧 ref 应报 refs 失效")
	}
}
