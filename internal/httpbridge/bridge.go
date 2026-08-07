// Package httpbridge exposes the desktop App's bound methods over HTTP so a
// plain browser (or phone) can drive the real Go kernel instead of a mock:
//
//	POST /api/rpc        {method, args} → same contract as window.go.app.App.*
//	GET  /api/stream?id=X  SSE stream of runtime events named X
//	GET  /api/health     liveness probe
//
// It is a thin, generic seam: methods are dispatched by reflection over the
// App instance, so any Wails binding works without a per-method adapter. Native
// capabilities that only exist inside the Wails shell (file dialogs, opening
// external URLs, the tray) still target the desktop window when the bridge runs
// inside the same process — acceptable degradation for web debugging.
package httpbridge

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"sync"
	"time"
)

// rpcRequest mirrors frontend/src/api/bridge.ts's RPC envelope.
type rpcRequest struct {
	Method string          `json:"method"`
	Args   json.RawMessage `json:"args"`
}

// rpcResponse mirrors the frontend's RPCResponse: result XOR error.
type rpcResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Bridge dispatches RPC calls to a single App instance and fans its events
// out to SSE subscribers.
type Bridge struct {
	app any
	hub *eventHub
}

// New wraps an App instance (any object whose exported methods match the
// Wails binding surface, e.g. *app.App).
func New(app any) *Bridge {
	// Share the package-level hub so core.emit → Publish reaches the streams
	// served by whichever Bridge instance is listening.
	return &Bridge{app: app, hub: globalHub}
}

// Handler returns the HTTP routes with CORS enabled for browser clients.
func (b *Bridge) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/rpc", b.handleRPC)
	mux.HandleFunc("/api/stream", b.handleStream)
	return cors(mux)
}

// Serve starts the bridge on addr (e.g. "127.0.0.1:8080"). Call in a
// goroutine; it blocks until the server is closed.
func Serve(addr string, app any) error {
	return http.ListenAndServe(addr, New(app).Handler())
}

// Publish routes a runtime event to every SSE subscriber of that name. Safe to
// call before/without a server running — a no-op when nobody subscribes.
func Publish(eventName string, data map[string]interface{}) {
	if eventName == "" {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	// Package-level hub so internal/app can emit without importing Bridge.
	globalHub.publish(eventName, payload)
}

func (b *Bridge) handleRPC(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, rpcResponse{Error: "bad request: " + err.Error()})
		return
	}
	if req.Method == "" {
		writeRPC(w, rpcResponse{Error: "method is required"})
		return
	}

	result, err := b.call(req.Method, req.Args)
	if err != nil {
		writeRPC(w, rpcResponse{Error: err.Error()})
		return
	}
	writeRPC(w, rpcResponse{Result: result})
}

// call invokes an exported method on the wrapped App by reflection, decoding
// args (a JSON array) into the method's parameter types. Missing trailing args
// become zero values; panics become errors.
func (b *Bridge) call(method string, rawArgs json.RawMessage) (result any, err error) {
	mv := reflect.ValueOf(b.app)
	if !mv.IsValid() {
		return nil, fmt.Errorf("app is nil")
	}
	m := mv.MethodByName(method)
	if !m.IsValid() {
		return nil, fmt.Errorf("unknown method %q", method)
	}

	mt := m.Type()
	var argv []json.RawMessage
	if len(rawArgs) > 0 && string(rawArgs) != "null" {
		if err := json.Unmarshal(rawArgs, &argv); err != nil {
			return nil, fmt.Errorf("args must be a JSON array: %w", err)
		}
	}

	in := make([]reflect.Value, 0, mt.NumIn())
	for i := 0; i < mt.NumIn(); i++ {
		pt := mt.In(i)
		v := reflect.New(pt)
		if i < len(argv) && len(argv[i]) > 0 && string(argv[i]) != "null" {
			if err := json.Unmarshal(argv[i], v.Interface()); err != nil {
				return nil, fmt.Errorf("arg %d for %s: %w", i, method, err)
			}
		}
		in = append(in, v.Elem())
	}

	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("httpbridge RPC panic", "method", method, "panic", rec)
			err = fmt.Errorf("method %s panicked: %v", method, rec)
		}
	}()
	outs := m.Call(in)
	if len(outs) == 0 {
		return nil, nil
	}
	// Convention used across the App: the last return is `error` when present.
	last := outs[len(outs)-1]
	if errType := reflect.TypeOf((*error)(nil)).Elem(); last.IsValid() && last.Type().Implements(errType) {
		if !last.IsNil() {
			return nil, last.Interface().(error)
		}
		if len(outs) == 1 {
			return nil, nil
		}
		return normalize(outs[0]), nil
	}
	return normalize(last), nil
}

// normalize converts a reflect.Value into a JSON-marshalable value, keeping
// nil/zero values as their natural zero (not typed nil pointers).
func normalize(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
		if v.IsNil() {
			return nil
		}
	}
	if v.CanInterface() {
		return v.Interface()
	}
	return nil
}

func writeRPC(w http.ResponseWriter, resp rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// cors allows browser clients on other origins (Vite dev server) to call us.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── SSE event hub ─────────────────────────────────────────────────────────

type eventHub struct {
	mu   sync.Mutex
	subs map[string]map[chan []byte]struct{}
}

var globalHub = newEventHub()

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[string]map[chan []byte]struct{})}
}

func (h *eventHub) subscribe(id string) chan []byte {
	ch := make(chan []byte, 64)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[id] == nil {
		h.subs[id] = make(map[chan []byte]struct{})
	}
	h.subs[id][ch] = struct{}{}
	return ch
}

func (h *eventHub) unsubscribe(id string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.subs[id]; ok {
		if _, exists := set[ch]; exists {
			delete(set, ch)
			close(ch)
		}
		if len(set) == 0 {
			delete(h.subs, id)
		}
	}
}

func (h *eventHub) publish(id string, payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[id] {
		select {
		case ch <- payload:
		default: // slow subscriber — drop rather than block the emitter
		}
	}
}

// handleStream serves one SSE connection per event name (?id=…). The frontend
// runtimePolyfill opens these for every event it subscribes to.
func (b *Bridge) handleStream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id query param is required", http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := b.hub.subscribe(id)
	defer b.hub.unsubscribe(id, ch)

	// Initial connection + periodic keep-alive so proxies don't time out.
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"id\":%q}\n\n", id)
	flusher.Flush()

	ctx := r.Context()
	keepAlive := make(chan struct{})
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				select {
				case keepAlive <- struct{}{}:
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepAlive:
			_, _ = fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		case payload, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}
