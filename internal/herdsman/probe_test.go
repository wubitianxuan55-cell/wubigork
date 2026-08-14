// H0-1 探测逻辑单元测试：全部隔离在临时目录，零外部网络依赖
// （真实 HTTP 探测目标一律指向进程内 httptest 服务或必定拒绝的端口）。
package herdsman

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile 在临时目录内写入文件并创建父目录（测试辅助）。
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("创建父目录失败: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入 %s 失败: %v", rel, err)
	}
}

// hasWarning 判断告警列表是否含指定子串。
func hasWarning(warnings []string, substr string) bool {
	for _, w := range warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}

// newIsolatedProbe 构造隔离探测器：rootDir 注入临时目录，baseURL 指向
// 进程内 httptest 服务（返回 OpenAI 兼容 /models 空列表），保证测试
// 不触碰真实 herdsman 服务。
func newIsolatedProbe(t *testing.T, rootDir string) *Probe {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	t.Cleanup(srv.Close)
	return NewProbe(rootDir, srv.URL)
}

// TestProbeDefaultSkipsAPI 覆盖默认参数分支：rootDir/baseURL 均为默认值
// 且未设置 HERDSMAN_PROBE_LIVE=1 时跳过真实 HTTP 探测。
func TestProbeDefaultSkipsAPI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HERDSMAN_PROBE_LIVE", "")
	t.Setenv("HERDSMAN_EXE", "")

	r := NewProbe("", "").Run()
	if r.APIReachable {
		t.Errorf("默认参数下 APIReachable 应为 false")
	}
	if r.APIError != "skipped" {
		t.Errorf("默认参数下 APIError 应为 skipped，实际 %q", r.APIError)
	}
	want := filepath.Join(home, ".herdsman")
	if r.HomeDir != want {
		t.Errorf("HomeDir = %q, want %q", r.HomeDir, want)
	}
	if r.ConfigPath != filepath.Join(want, "config.yaml") {
		t.Errorf("ConfigPath = %q, want %q", r.ConfigPath, filepath.Join(want, "config.yaml"))
	}
	if len(r.DataFiles) != 4 {
		t.Errorf("DataFiles 应有 4 项（launch_records/events.jsonl/skill-operations.json/models），实际 %d", len(r.DataFiles))
	}

	// 结果必须可 JSON 序列化，且字段名遵循 json tag（snake_case）。
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("探测结果 JSON 序列化失败: %v", err)
	}
	for _, k := range []string{"home_dir", "config_path", "config_ok", "cli",
		"api_reachable", "api_error", "data_files", "warnings"} {
		if !strings.Contains(string(b), k) {
			t.Errorf("JSON 输出缺少字段 %q", k)
		}
	}
}

// TestProbeDefaultDataDirEnv 验证 HERDSMAN_DATA_DIR 优先于 home/.herdsman
// （与 internal/app 的 herdsmanDataDir 口径一致）。
func TestProbeDefaultDataDirEnv(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HERDSMAN_DATA_DIR", dir)
	t.Setenv("HERDSMAN_PROBE_LIVE", "")
	t.Setenv("HERDSMAN_EXE", "")

	r := NewProbe("", "").Run()
	if r.HomeDir != dir {
		t.Errorf("HomeDir = %q, want HERDSMAN_DATA_DIR %q", r.HomeDir, dir)
	}
	if r.ConfigPath != filepath.Join(dir, "config.yaml") {
		t.Errorf("ConfigPath = %q, want %q", r.ConfigPath, filepath.Join(dir, "config.yaml"))
	}
}

// TestProbeConfigYAML 覆盖 config.yaml 存在且 lan_accessible=true/port 提取。
func TestProbeConfigYAML(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "config.yaml", `general:
    data_dir: C:/tmp
api:
    lan_accessible: true
    port: 9090
log:
    level: info
`)
	r := newIsolatedProbe(t, root).Run()
	if !r.ConfigOK {
		t.Fatalf("ConfigOK 应为 true，ConfigError=%q", r.ConfigError)
	}
	if r.ConfigError != "" {
		t.Errorf("ConfigError 应为空，实际 %q", r.ConfigError)
	}
	if r.Config.LANAccessible == nil || !*r.Config.LANAccessible {
		t.Errorf("LANAccessible 应解析为 true，实际 %v", r.Config.LANAccessible)
	}
	if r.Config.Port == nil || *r.Config.Port != 9090 {
		t.Errorf("Port 应解析为 9090，实际 %v", r.Config.Port)
	}
	if !hasWarning(r.Warnings, "lan_accessible=true") {
		t.Errorf("lan_accessible=true 应产生暴露告警，实际 %v", r.Warnings)
	}
}

// TestProbeConfigFalseAndPort 覆盖 lan_accessible=false：不产生暴露告警。
func TestProbeConfigFalseAndPort(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "config.yaml", `api:
    lan_accessible: false
    port: 8080
`)
	r := newIsolatedProbe(t, root).Run()
	if !r.ConfigOK {
		t.Fatalf("ConfigOK 应为 true，ConfigError=%q", r.ConfigError)
	}
	if r.Config.LANAccessible == nil || *r.Config.LANAccessible {
		t.Errorf("LANAccessible 应解析为 false，实际 %v", r.Config.LANAccessible)
	}
	if r.Config.Port == nil || *r.Config.Port != 8080 {
		t.Errorf("Port 应解析为 8080，实际 %v", r.Config.Port)
	}
	if hasWarning(r.Warnings, "lan_accessible=true") {
		t.Errorf("lan_accessible=false 不应产生暴露告警，实际 %v", r.Warnings)
	}
}

// TestProbeConfigMissing 覆盖 config.yaml 缺失：ConfigOK=false + 告警。
func TestProbeConfigMissing(t *testing.T) {
	r := newIsolatedProbe(t, t.TempDir()).Run()
	if r.ConfigOK {
		t.Errorf("无 config.yaml 时 ConfigOK 应为 false")
	}
	if r.ConfigError == "" {
		t.Errorf("ConfigError 应说明原因，实际为空")
	}
	if !hasWarning(r.Warnings, "config.yaml") {
		t.Errorf("应含 config.yaml 缺失告警，实际 %v", r.Warnings)
	}
}

// TestProbeConfigEmpty 覆盖 config.yaml 存在但为空：视为未通过。
func TestProbeConfigEmpty(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "config.yaml", "\n   \n")
	r := newIsolatedProbe(t, root).Run()
	if r.ConfigOK {
		t.Errorf("空 config.yaml 时 ConfigOK 应为 false")
	}
	if r.ConfigError == "" {
		t.Errorf("ConfigError 应说明文件为空，实际为空")
	}
}

// TestProbePortDriftWarning 覆盖端口漂移告警：默认 baseURL 下配置端口
// 非 8080 时应提示默认连接可能失效（此场景同时验证默认参数跳过 API）。
func TestProbePortDriftWarning(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	root := filepath.Join(home, ".herdsman")
	writeFile(t, root, "config.yaml", "api:\n    port: 9090\n")
	r := NewProbe("", "").Run()
	if r.APIError != "skipped" {
		t.Errorf("默认参数下 APIError 应为 skipped，实际 %q", r.APIError)
	}
	if !hasWarning(r.Warnings, "api.port=9090") {
		t.Errorf("应含端口漂移告警（api.port=9090 与默认 8080 不一致），实际 %v", r.Warnings)
	}
}

// TestProbeDataFiles 覆盖四个数据契约齐备：均可解析，无缺失告警。
func TestProbeDataFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "config.yaml", "api:\n    lan_accessible: false\n    port: 8080\n")
	writeFile(t, root, "launch_records/qwen.json",
		`[{"model_name":"qwen","status":"success","start_time":"2026-01-01T00:00:00Z"}]`)
	// 第二行非法不应影响抽查（只查第一行非空行）。
	writeFile(t, root, "model_stats/events.jsonl",
		`{"model":"bge-m3","status":"succeeded","duration_ms":12}`+"\n"+"this-line-is-invalid-json")
	writeFile(t, root, "skill-operations.json",
		`[{"id":"1","kind":"tts","status":"completed"}]`)
	writeFile(t, root, "models/placeholder.txt", "x")

	r := newIsolatedProbe(t, root).Run()
	for _, name := range []string{"launch_records", "models"} {
		st := r.DataFiles[name]
		if !st.Exists || !st.Parseable {
			t.Errorf("%s 应为存在且可解析: %+v", name, st)
		}
	}
	if st := r.DataFiles["model_stats/events.jsonl"]; !st.Exists || !st.Parseable {
		t.Errorf("events.jsonl 应为存在且可解析（第二行非法不影响）: %+v", st)
	}
	if st := r.DataFiles["skill-operations.json"]; !st.Exists || !st.Parseable {
		t.Errorf("skill-operations.json 应为存在且可解析: %+v", st)
	}
	if hasWarning(r.Warnings, "数据文件缺失") || hasWarning(r.Warnings, "数据目录缺失") {
		t.Errorf("契约齐备时不应有数据文件缺失告警，实际 %v", r.Warnings)
	}
}

// TestProbeDataFilesInvalid 覆盖 JSON/JSONL 不可解析：Parseable=false + 告警。
func TestProbeDataFilesInvalid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "skill-operations.json", "not-json")
	writeFile(t, root, "model_stats/events.jsonl", "not-json-line-one\n")
	r := newIsolatedProbe(t, root).Run()

	st := r.DataFiles["skill-operations.json"]
	if !st.Exists || st.Parseable || st.Error == "" {
		t.Errorf("skill-operations.json 应存在但不可解析: %+v", st)
	}
	st = r.DataFiles["model_stats/events.jsonl"]
	if !st.Exists || st.Parseable || st.Error == "" {
		t.Errorf("events.jsonl 首行非法应不可解析: %+v", st)
	}
	if !hasWarning(r.Warnings, "不可解析") {
		t.Errorf("应含不可解析告警，实际 %v", r.Warnings)
	}
}

// TestProbeDataDirMissingWarning 覆盖目录缺失告警文案：应打印被检查的
// 真实目录路径（rootDir/name），而非配置文件路径 config.yaml。
func TestProbeDataDirMissingWarning(t *testing.T) {
	root := t.TempDir()
	// config.yaml 存在且无暴露/端口漂移配置，排除其他告警干扰。
	writeFile(t, root, "config.yaml", "api:\n    lan_accessible: false\n    port: 8080\n")
	// 不创建 launch_records/ 与 models/，触发目录缺失告警。
	r := newIsolatedProbe(t, root).Run()

	for _, name := range []string{"launch_records", "models"} {
		if !hasWarning(r.Warnings, "数据目录缺失："+name) {
			t.Errorf("应含 %s 目录缺失告警，实际 %v", name, r.Warnings)
		}
		want := filepath.Join(root, name)
		if !hasWarning(r.Warnings, want) {
			t.Errorf("目录缺失告警应含真实目录路径 %q，实际 %v", want, r.Warnings)
		}
	}
	// 目录缺失告警不得把配置文件路径当目录路径打印。
	for _, w := range r.Warnings {
		if strings.Contains(w, "数据目录缺失") && strings.Contains(w, "config.yaml") {
			t.Errorf("目录缺失告警不应含 config.yaml 字样：%q", w)
		}
	}
}

// TestProbeCLI 覆盖 herdsman.exe 探测的四个分支。
func TestProbeCLI(t *testing.T) {
	t.Run("HERDSMAN_EXE有效", func(t *testing.T) {
		root := t.TempDir()
		fake := filepath.Join(root, "herdsman.exe")
		if err := os.WriteFile(fake, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HERDSMAN_EXE", fake)
		t.Setenv("PATH", t.TempDir()) // 排除 PATH 干扰
		r := newIsolatedProbe(t, root).Run()
		if !r.CLI.Found || r.CLI.Source != "env" || r.CLI.Path != fake {
			t.Errorf("CLI = %+v, want Found=true/Source=env/Path=%q", r.CLI, fake)
		}
	})

	t.Run("HERDSMAN_EXE不存在", func(t *testing.T) {
		root := t.TempDir()
		missing := filepath.Join(root, "missing.exe")
		t.Setenv("HERDSMAN_EXE", missing)
		t.Setenv("PATH", t.TempDir())
		r := newIsolatedProbe(t, root).Run()
		if r.CLI.Found {
			t.Errorf("不存在的 HERDSMAN_EXE 应判定为未找到: %+v", r.CLI)
		}
		if r.CLI.Source != "env" {
			t.Errorf("应记录来源 env，实际 %q", r.CLI.Source)
		}
	})

	t.Run("PATH命中", func(t *testing.T) {
		bin := t.TempDir()
		if err := os.WriteFile(filepath.Join(bin, "herdsman.exe"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HERDSMAN_EXE", "")
		t.Setenv("PATH", bin)
		r := newIsolatedProbe(t, t.TempDir()).Run()
		if !r.CLI.Found || r.CLI.Source != "path" {
			t.Errorf("PATH 命中应 Found=true/Source=path，实际 %+v", r.CLI)
		}
	})

	t.Run("PATH未命中", func(t *testing.T) {
		t.Setenv("HERDSMAN_EXE", "")
		t.Setenv("PATH", t.TempDir()) // 空目录
		r := newIsolatedProbe(t, t.TempDir()).Run()
		if r.CLI.Found {
			t.Errorf("PATH 未命中应 Found=false，实际 %+v", r.CLI)
		}
		if r.CLI.Source != "path" {
			t.Errorf("应记录来源 path，实际 %q", r.CLI.Source)
		}
	})
}

// TestProbeAPILive 覆盖 httptest 成功分支：真实 HTTP 探测可达。
func TestProbeAPILive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("探测请求路径应为 /models，实际 %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer srv.Close()

	r := NewProbe(t.TempDir(), srv.URL).Run()
	if !r.APIReachable {
		t.Errorf("httptest 服务下 APIReachable 应为 true，APIError=%q", r.APIError)
	}
	if r.APIError != "" {
		t.Errorf("APIError 应为空，实际 %q", r.APIError)
	}
}

// TestProbeAPIUnreachable 覆盖真实探测失败分支：无服务监听端口应不可达。
func TestProbeAPIUnreachable(t *testing.T) {
	// 127.0.0.1:1 无服务监听，连接应立即被拒绝（不会真正发出到外部）。
	r := NewProbe(t.TempDir(), "http://127.0.0.1:1").Run()
	if r.APIReachable {
		t.Errorf("无服务端口应判定为不可达")
	}
	if r.APIError == "" || r.APIError == "skipped" {
		t.Errorf("APIError 应为连接错误，实际 %q", r.APIError)
	}
	if !hasWarning(r.Warnings, "不可达") {
		t.Errorf("应含 API 不可达告警，实际 %v", r.Warnings)
	}
}
