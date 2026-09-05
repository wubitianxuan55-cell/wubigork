package app

// MH4（蒸馏 unsloth 规划 §三/§四）Model Hub 常驻预热回归：
// 武装位 / 活跃引擎跟随 / 显存互斥让路 / 已加载跳过（幂等守护）/ 端到端预热收敛。

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// modelHubHubStub 假 Unsloth Studio：/v1/models 返回 loaded 集合，
// /api/inference/load 记录调用并可翻转 loaded。逐端点计数供断言。
type modelHubHubStub struct {
	mu        sync.Mutex
	loaded    map[string]bool
	modelHits int
	loadHits  int
	loadedReq string // 最近一次 load 的 model_path
}

func newModelHubHubStub(t *testing.T) (*modelHubHubStub, *httptest.Server) {
	t.Helper()
	st := &modelHubHubStub{loaded: map[string]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st.mu.Lock()
		defer st.mu.Unlock()
		switch {
		case r.URL.Path == "/v1/models":
			st.modelHits++
			data := []any{}
			for id, ok := range st.loaded {
				if ok {
					data = append(data, map[string]any{"id": id, "loaded": true})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": data})
		case r.URL.Path == "/api/inference/load":
			st.loadHits++
			body, _ := io.ReadAll(r.Body)
			var req struct {
				ModelPath string `json:"model_path"`
			}
			_ = json.Unmarshal(body, &req)
			st.loadedReq = req.ModelPath
			st.loaded[req.ModelPath] = true
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "loading"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return st, srv
}

// modelHubPreloadTestApp 组装武装到位的 App（活跃引擎/Key/默认模型按参数）。
func modelHubPreloadTestApp(t *testing.T, hubURL string, activeEngine string, withKey, withDefault bool) *App {
	t.Helper()
	cfg := &config.Config{AutoPreload: true}
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID: "modelhub", BaseURL: hubURL + "/v1", Enabled: true,
		DefaultModel: map[bool]string{true: "ollama-manifest:qwen:UD-Q4_K_XL", false: ""}[withDefault],
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if withKey {
		mgr.UpdateModelHubKey("sk-unsloth-test")
	}
	c := ai.NewClient(cfg)
	if activeEngine != "" {
		c.SetActiveEngine(activeEngine)
	}
	return &App{core: &core{cfg: cfg, engineMgr: mgr, client: c}, officeState: &officeState{}}
}

// armModelHubPreload 置位武装位并在测试结束后复位（包级位，避免泄漏到其他用例）。
func armModelHubPreload(t *testing.T) {
	t.Helper()
	modelHubPreloadArmed.Store(true)
	t.Cleanup(func() { modelHubPreloadArmed.Store(false) })
	// 测试用短轮询：收敛等待从 5s 缩到 5ms。
	old := modelHubPreloadPollInterval
	modelHubPreloadPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { modelHubPreloadPollInterval = old })
}

func TestModelHubPreloadUnarmedSkips(t *testing.T) {
	st, srv := newModelHubHubStub(t)
	a := modelHubPreloadTestApp(t, srv.URL, "modelhub", true, true)
	// 不调 armModelHubPreload：武装位保持复位
	a.runModelHubPreload()
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.modelHits != 0 || st.loadHits != 0 {
		t.Fatalf("未武装不应触达 Studio（models=%d load=%d）", st.modelHits, st.loadHits)
	}
}

func TestModelHubPreloadFollowsActiveEngine(t *testing.T) {
	st, srv := newModelHubHubStub(t)
	a := modelHubPreloadTestApp(t, srv.URL, "herdsman", true, true)
	armModelHubPreload(t)
	a.runModelHubPreload()
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.modelHits != 0 {
		t.Fatalf("活跃引擎非 modelhub 不应触达 Studio（models=%d）", st.modelHits)
	}
}

func TestModelHubPreloadRequiresKeyAndDefaultModel(t *testing.T) {
	st1, srv1 := newModelHubHubStub(t)
	a1 := modelHubPreloadTestApp(t, srv1.URL, "modelhub", false, true)
	armModelHubPreload(t)
	a1.runModelHubPreload()

	st2, srv2 := newModelHubHubStub(t)
	a2 := modelHubPreloadTestApp(t, srv2.URL, "modelhub", true, false)
	a2.runModelHubPreload()

	for i, st := range []*modelHubHubStub{st1, st2} {
		st.mu.Lock()
		hits := st.modelHits + st.loadHits
		st.mu.Unlock()
		if hits != 0 {
			t.Fatalf("case %d：缺 Key/缺默认模型不应触达 Studio（hits=%d）", i, hits)
		}
	}
}

func TestModelHubPreloadSkipsWhenAlreadyLoaded(t *testing.T) {
	st, srv := newModelHubHubStub(t)
	st.mu.Lock()
	st.loaded["ollama-manifest:qwen:UD-Q4_K_XL"] = true
	st.mu.Unlock()
	a := modelHubPreloadTestApp(t, srv.URL, "modelhub", true, true)
	armModelHubPreload(t)

	a.runModelHubPreload()

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.loadHits != 0 {
		t.Fatalf("已加载模型不应触发 load（幂等守护，§四-2），load=%d", st.loadHits)
	}
	if st.modelHits != 1 {
		t.Fatalf("应恰好探测一次已加载集合，models=%d", st.modelHits)
	}
}

func TestModelHubPreloadSkipsWhenHerdsmanPreloadWouldStart(t *testing.T) {
	st, srv := newModelHubHubStub(t)
	cfg := &config.Config{AutoPreload: true, FuncGaeaEngine: "herdsman", FuncGaeaModel: "gaea-m"}
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID: "modelhub", BaseURL: srv.URL + "/v1", Enabled: true, DefaultModel: "ollama-manifest:qwen:UD-Q4_K_XL",
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	mgr.UpdateModelHubKey("sk-unsloth-test")
	c := ai.NewClient(cfg)
	c.SetActiveEngine("modelhub")
	a := &App{core: &core{cfg: cfg, engineMgr: mgr, client: c}, officeState: &officeState{}}

	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		return []byte(catalogResult("gaea-m", true, false)), nil // 已安装未运行=herdsman 预载将启动
	}
	armModelHubPreload(t)

	a.runModelHubPreload()

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.modelHits != 0 || st.loadHits != 0 {
		t.Fatalf("herdsman 预载在途时应让路（models=%d load=%d）", st.modelHits, st.loadHits)
	}
}

func TestModelHubPreloadEndToEnd(t *testing.T) {
	st, srv := newModelHubHubStub(t)
	a := modelHubPreloadTestApp(t, srv.URL, "modelhub", true, true)
	armModelHubPreload(t)

	a.runModelHubPreload() // 同步返回（收敛轮询在内）

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.loadHits != 1 {
		t.Fatalf("应恰好发起一次加载，load=%d", st.loadHits)
	}
	if st.loadedReq != "ollama-manifest:qwen:UD-Q4_K_XL" {
		t.Fatalf("加载目标应为钉选默认模型，got %q", st.loadedReq)
	}
	if !st.loaded[st.loadedReq] {
		t.Fatal("收敛后模型应处于已加载状态")
	}
}
