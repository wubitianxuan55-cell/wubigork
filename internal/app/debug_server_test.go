package app

import (
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"testing"
)

// TestDebugServerHealthAndStack 诊断端口可用：/healthz 存活探针 + /stack 协程栈。
func TestDebugServerHealthAndStack(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/stack", func(w http.ResponseWriter, _ *http.Request) {
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		w.Write(buf[:n])
	})
	srv := &http.Server{Addr: "127.0.0.1:0", Handler: mux}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	resp, err := http.Get("http://" + ln.Addr().String() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(b) != "ok" {
		t.Fatalf("healthz = %q", b)
	}

	resp2, err := http.Get("http://" + ln.Addr().String() + "/stack")
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !strings.Contains(string(b2), "goroutine") {
		t.Fatalf("stack 缺少 goroutine 信息")
	}
}
