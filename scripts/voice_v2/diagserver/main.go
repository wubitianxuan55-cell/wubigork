// main.go — diag_tap 诊断端点自测服务端(独立 main,ws_probe 同款形态)。
//
// 起 realtime.Server 于 :8799(生产 8791 不碰),仅用于真浏览器 fetch→
// CORS 预检→落盘的集成自测。用法:go run ./scripts/voice_v2/diagserver
package main

import (
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/voice/realtime"
)

func main() {
	srv := &realtime.Server{Addr: "127.0.0.1:8799"}
	if err := srv.Start(); err != nil {
		panic(err)
	}
	fmt.Println("diag selftest server on 127.0.0.1:8799")
	for {
		time.Sleep(time.Hour)
	}
}
