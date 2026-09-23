package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestPollComfyProgressReadable 载入阶段进度可读化（v4.405）：假 ComfyUI WS
// 服务推送脚本化事件序列，断言回调——
//   - status（queue_remaining>1）→ queued + node=queue（排队可见）
//   - executing（无百分比）→ 节点名经 nodeClasses 映射透出（载入可见）
//   - progress（有百分比）→ percent 透传
//   - progress_state running 但 max=0 → 不再整段丢弃，节点名透出
//   - 外来 prompt 的事件被忽略
func TestPollComfyProgressReadable(t *testing.T) {
	up := websocket.Upgrader{}
	msgs := []string{
		// 外来 prompt：应被忽略
		`{"type":"executing","data":{"node":"4","prompt_id":"other"}}`,
		// 排队：前面还有任务
		`{"type":"status","data":{"status":{"exec_info":{"queue_remaining":2}}}}`,
		// 载入模型：节点开始执行，无百分比
		`{"type":"executing","data":{"node":"4","prompt_id":"p1"}}`,
		// 采样：真实进度
		`{"type":"progress","data":{"value":3,"max":8,"node":"10","prompt_id":"p1"}}`,
		// 新版协议：VAE 载入 running 但 max=0（此前被整段丢弃）
		`{"type":"progress_state","data":{"prompt_id":"p1","nodes":{"6":{"value":0,"max":0,"state":"running"}}}}`,
		// 整单结束信号：node=null 不应产生回调
		`{"type":"executing","data":{"node":null,"prompt_id":"p1"}}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for _, m := range msgs {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
				return
			}
			time.Sleep(30 * time.Millisecond)
		}
		// 保持连接一小段，等客户端读完
		time.Sleep(800 * time.Millisecond)
	}))
	defer srv.Close()

	b := NewComfyUIBackend(srv.URL)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	_ = wsURL

	nodeClasses := map[string]string{"4": "UNETLoader", "10": "KSampler", "6": "VAELoader"}

	type cbRec struct {
		status  string
		elapsed int
		percent int
		node    string
	}
	var mu sync.Mutex
	var got []cbRec
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go b.pollComfyProgress(ctx, "p1", nodeClasses, func(status string, elapsed int, percent int, node string) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, cbRec{status, elapsed, percent, node})
	})

	// 等到全部 4 条预期回调（外来 prompt 与 node=null 不产生）
	deadline := time.Now().Add(2500 * time.Millisecond)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 4 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) < 4 {
		t.Fatalf("应收到 4 条回调，得到 %d: %+v", len(got), got)
	}
	// 1. 排队
	if got[0].status != "queued" || got[0].node != "queue" || got[0].percent != -1 {
		t.Fatalf("排队回调应 queued/queue/-1: %+v", got[0])
	}
	// 2. 载入模型可见（无百分比）
	if got[1].status != "running" || got[1].node != "UNETLoader" || got[1].percent != -1 {
		t.Fatalf("executing 应透出 UNETLoader 无百分比: %+v", got[1])
	}
	// 3. 采样进度
	if got[2].node != "KSampler" || got[2].percent != 37 {
		t.Fatalf("progress 应 KSampler 37%%: %+v", got[2])
	}
	// 4. progress_state max=0 不再丢弃
	if got[3].node != "VAELoader" || got[3].percent != -1 {
		t.Fatalf("progress_state max=0 应透出 VAELoader: %+v", got[3])
	}
	// 5. node=null 与外来 prompt 均未产生回调（总数恰 4）
	if len(got) != 4 {
		t.Fatalf("不应有额外回调（外来 prompt/node=null）: %+v", got[4:])
	}
}
