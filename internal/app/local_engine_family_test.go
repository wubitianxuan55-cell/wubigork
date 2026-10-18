package app

// 本地引擎名单族回归：strata/tianxuan 加入「本地引擎」族时，按引擎名硬编码的
// 名单漏改了两处 —— isLocalEngine（加载预警误把最吃资源的两个引擎当云端）与
// GaeaModelSwitchEstimate（切到 strata 直接报「引擎已就绪」，冷启动不预警）。
// 两处现均改走 modelengine.BuiltinEngineTypeOf(...).IsLocal() 单一真相口，
// 本文件钉住消费口径与 strata 的换模预估链路。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/whisper"
)

// TestIsLocalEngine_LocalFamily 本地族齐五、云端族五个全假。
func TestIsLocalEngine_LocalFamily(t *testing.T) {
	for _, id := range []string{"ollama", "herdsman", "cosyvoice", "strata", "tianxuan"} {
		if !isLocalEngine(id) {
			t.Errorf("%s 是本地引擎（占本机资源，应计入加载预警）", id)
		}
	}
	for _, id := range []string{"xai", "deepseek", "glm", "opencode-go", "opencode-zen", "custom-abc", ""} {
		if isLocalEngine(id) {
			t.Errorf("%s 不是本地引擎，不应计入加载预警", id)
		}
	}
}

// strataModelsJSON 真机形状：strata /v1/models 的 status 是非标对象
// （{"value":"loaded"}），且只列已加载集合。
const strataModelsJSON = `{"object":"list","data":[{"id":"qwen3.6-35b-a3b-uncensored-q4_k_p","object":"model","status":{"value":"loaded"},"meta":{"n_ctx":131072}}]}`

// newStrataScheduleApp 构造 App：strata 引擎指向给定测试服务器。
func newStrataScheduleApp(t *testing.T, baseURL string) *App {
	t.Helper()
	a := scheduleTestApp(&config.Config{})
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "strata", BaseURL: baseURL + "/v1", Enabled: true,
	}); err != nil {
		t.Fatalf("SaveEngine(strata): %v", err)
	}
	return a
}

// TestStrata_PlayGuardrailKeepsThinkingFlag play 护栏与思考三态的交互回归：
// applyWhisperGuardrails 持指针就地改 opts（温度/预算），极易顺手把新增字段
// 冲掉——这里钉住 DisableThinking 穿过护栏存活，且护栏的「只降不升」上限
// 确实落到 MaxTokens 上（留给 ai 装配层的优先序判定：开思考时守护再把上限
// 抬回 4096，关思考时尊重护栏上限，见 ai/client_chat.go strata 分支）。
func TestStrata_PlayGuardrailKeepsThinkingFlag(t *testing.T) {
	g := gaeaConfig.PlayGuardrails{PersonaLock: true, TemperatureMax: 0.5, MaxOutputTokens: 2048}

	opts := ai.ChatSimpleOptions{EngineID: "strata", DisableThinking: true}
	_ = applyWhisperGuardrails(&opts, "sys", whisper.PersonalityPreset{}, g)

	if !opts.DisableThinking {
		t.Error("applyWhisperGuardrails 冲掉了 DisableThinking（三态关侧丢失即思考关不掉）")
	}
	if opts.MaxTokens != 2048 {
		t.Errorf("护栏 max_output_tokens=2048 应钳制到 2048（基线 4096 只降不升），got %d", opts.MaxTokens)
	}
	if opts.Temperature != 0.5 {
		t.Errorf("护栏 TemperatureMax=0.5 应锁到 0.5，got %v", opts.Temperature)
	}
	if opts.EngineID != "strata" {
		t.Errorf("护栏不应改写 EngineID，got %q", opts.EngineID)
	}
}

// TestGaeaModelSwitchEstimate_Strata 换模预估：strata 走本地引擎分支（不再恒
// 报 hot），已加载→hot、未加载→cold、不可达→unknown。
func TestGaeaModelSwitchEstimate_Strata(t *testing.T) {
	newSrv := func(body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("已加载→hot", func(t *testing.T) {
		a := newStrataScheduleApp(t, newSrv(strataModelsJSON).URL)
		e := a.GaeaModelSwitchEstimate("strata", "qwen3.6-35b-a3b-uncensored-q4_k_p")
		if e.Status != "hot" || e.WaitSeconds != 1 {
			t.Errorf("Status/Wait = (%q,%d), want (hot,1)；note=%q", e.Status, e.WaitSeconds, e.Note)
		}
		if e.Model != "qwen3.6-35b-a3b-uncensored-q4_k_p" {
			t.Errorf("Model = %q, want qwen3.6-35b-a3b-uncensored-q4_k_p", e.Model)
		}
	})

	t.Run("未加载→cold", func(t *testing.T) {
		a := newStrataScheduleApp(t, newSrv(strataModelsJSON).URL)
		e := a.GaeaModelSwitchEstimate("strata", "another-model")
		if e.Status != "cold" || e.WaitSeconds != 0 {
			t.Errorf("Status/Wait = (%q,%d), want (cold,0)——大模型加载时长不假报秒数", e.Status, e.WaitSeconds)
		}
	})

	t.Run("空模型回退引擎默认模型", func(t *testing.T) {
		a := newStrataScheduleApp(t, newSrv(strataModelsJSON).URL)
		if err := a.engineMgr.SetDefaultModel("strata", "qwen3.6-35b-a3b-uncensored-q4_k_p"); err != nil {
			t.Fatalf("SetDefaultModel: %v", err)
		}
		for _, arg := range []string{"", "(默认)"} {
			e := a.GaeaModelSwitchEstimate("strata", arg)
			if e.Status != "hot" || e.Model != "qwen3.6-35b-a3b-uncensored-q4_k_p" {
				t.Errorf("arg=%q 应回退默认模型/hot，got %q/%q", arg, e.Status, e.Model)
			}
		}
	})

	t.Run("服务不可达→unknown", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close() // 关掉：端口必然不可达
		a := newStrataScheduleApp(t, url)
		e := a.GaeaModelSwitchEstimate("strata", "qwen3.6-35b-a3b-uncensored-q4_k_p")
		if e.Status != "unknown" {
			t.Errorf("不可达应如实 unknown（宁未知勿假 hot），got %q", e.Status)
		}
	})

	t.Run("非 200→unknown", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)
		a := newStrataScheduleApp(t, srv.URL)
		e := a.GaeaModelSwitchEstimate("strata", "m")
		if e.Status != "unknown" {
			t.Errorf("HTTP 500 应如实 unknown，got %q", e.Status)
		}
	})
}

// TestStrataModelLoaded_NonStandardStatus 真机非标 status 对象的宽容解码回归：
// status 是 {"value":"loaded"} 而非裸字符串，裸 string 解码会整表炸解析
// （v4.480 前 Strata 引擎即因此被判定不可用）。显式给出非就绪状态则不算加载。
func TestStrataModelLoaded_NonStandardStatus(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		model string
		want  bool
	}{
		{"对象形态 loaded", strataModelsJSON, "qwen3.6-35b-a3b-uncensored-q4_k_p", true},
		{"裸字符串 loaded", `{"data":[{"id":"m1","status":"loaded"}]}`, "m1", true},
		{"状态缺失", `{"data":[{"id":"m1"}]}`, "m1", true},
		{"对象形态 loading", `{"data":[{"id":"m1","status":{"value":"loading"}}]}`, "m1", true},
		{"对象形态 error", `{"data":[{"id":"m1","status":{"value":"error"}}]}`, "m1", false},
		{"目标不在已加载集合", strataModelsJSON, "another-model", false},
		{"空列表", `{"data":[]}`, "m1", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(c.body))
			}))
			defer srv.Close()
			a := newStrataScheduleApp(t, srv.URL)
			got, err := a.engineMgr.StrataModelLoaded(t.Context(), c.model)
			if err != nil {
				t.Fatalf("StrataModelLoaded: %v", err)
			}
			if got != c.want {
				t.Errorf("loaded = %v, want %v（model=%s body=%s）", got, c.want, c.model, c.body)
			}
		})
	}
}

// tianxuanModelsJSON 真机形状：天璇引擎（gufo）/v1/models 是标准形状
// （无 strata 的 flexStatus 异形），只列已加载集合。
const tianxuanModelsJSON = `{"object":"list","data":[{"id":"Qwen3.8-27B-UD-Q4_K_M","object":"model","created":1792236665,"owned_by":"gufo"}]}`

// newTianxuanScheduleApp 构造 App：tianxuan 引擎指向给定测试服务器。
func newTianxuanScheduleApp(t *testing.T, baseURL string) *App {
	t.Helper()
	a := scheduleTestApp(&config.Config{})
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "tianxuan", BaseURL: baseURL + "/v1", Enabled: true,
	}); err != nil {
		t.Fatalf("SaveEngine(tianxuan): %v", err)
	}
	return a
}

// TestGaeaModelSwitchEstimate_Tianxuan 换模预估：tianxuan 走本地引擎分支
// （不再恒报 hot），已加载→hot、未加载→cold、不可达→unknown。
func TestGaeaModelSwitchEstimate_Tianxuan(t *testing.T) {
	newSrv := func(body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("已加载→hot", func(t *testing.T) {
		a := newTianxuanScheduleApp(t, newSrv(tianxuanModelsJSON).URL)
		e := a.GaeaModelSwitchEstimate("tianxuan", "Qwen3.8-27B-UD-Q4_K_M")
		if e.Status != "hot" || e.WaitSeconds != 1 {
			t.Errorf("Status/Wait = (%q,%d), want (hot,1)；note=%q", e.Status, e.WaitSeconds, e.Note)
		}
	})

	t.Run("未加载→cold", func(t *testing.T) {
		a := newTianxuanScheduleApp(t, newSrv(tianxuanModelsJSON).URL)
		e := a.GaeaModelSwitchEstimate("tianxuan", "another-model")
		if e.Status != "cold" || e.WaitSeconds != 0 {
			t.Errorf("Status/Wait = (%q,%d), want (cold,0)——模型加载不假报秒数", e.Status, e.WaitSeconds)
		}
		// 天璇启动即固定模型（gufo 对非本进程模型名直接 404），note 必须说重启，
		// 不能承诺「首次对话自动加载」。
		if !strings.Contains(e.Note, "重启") {
			t.Errorf("cold note 应说明需重启 serve 引擎，got %q", e.Note)
		}
		if !strings.Contains(e.Note, "Qwen3.8-27B-UD-Q4_K_M") {
			t.Errorf("cold note 应点名当前服务的模型，got %q", e.Note)
		}
	})

	t.Run("服务不可达→unknown", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		a := newTianxuanScheduleApp(t, url)
		e := a.GaeaModelSwitchEstimate("tianxuan", "Qwen3.8-27B-UD-Q4_K_M")
		if e.Status != "unknown" {
			t.Errorf("不可达应如实 unknown（宁未知勿假 hot），got %q", e.Status)
		}
	})
}

// TestTianxuanModelLoaded_StandardShape 天璇 /v1/models 标准形状的加载判定：
// 命中即已加载；目标不在已加载集合→false。
func TestTianxuanModelLoaded_StandardShape(t *testing.T) {
	t.Run("已加载", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(tianxuanModelsJSON))
		}))
		defer srv.Close()
		a := newTianxuanScheduleApp(t, srv.URL)
		got, err := a.engineMgr.TianxuanModelLoaded(t.Context(), "Qwen3.8-27B-UD-Q4_K_M")
		if err != nil {
			t.Fatalf("TianxuanModelLoaded: %v", err)
		}
		if !got {
			t.Error("loaded = false, want true")
		}
	})
	t.Run("目标不在集合", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(tianxuanModelsJSON))
		}))
		defer srv.Close()
		a := newTianxuanScheduleApp(t, srv.URL)
		got, err := a.engineMgr.TianxuanModelLoaded(t.Context(), "another-model")
		if err != nil {
			t.Fatalf("TianxuanModelLoaded: %v", err)
		}
		if got {
			t.Error("loaded = true, want false")
		}
	})
}
