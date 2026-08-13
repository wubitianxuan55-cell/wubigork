package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseHerdsmanOpResult(t *testing.T) {
	r, err := parseHerdsmanOpResult([]byte(`{"ok":true,"result":{"status":"completed"}}`))
	if err != nil || !r.OK || r.Status != "completed" {
		t.Fatalf("completed 解析失败: r=%+v err=%v", r, err)
	}
	// 无 result.status 且 ok=true → 视为 completed（stop/uninstall 等短命令）。
	r, err = parseHerdsmanOpResult([]byte(`{"ok":true}`))
	if err != nil || r.Status != "completed" {
		t.Fatalf("ok=true 空状态解析失败: r=%+v err=%v", r, err)
	}
	// 失败路径：ok:false 带错误。
	r, err = parseHerdsmanOpResult([]byte(`{"ok":false,"result":{"status":"failed","error":"模型启动失败"}}`))
	if err == nil || !strings.Contains(err.Error(), "模型启动失败") {
		t.Fatalf("失败路径应返回错误: r=%+v err=%v", r, err)
	}
	// 非法 JSON。
	if _, err := parseHerdsmanOpResult([]byte(`nope`)); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

func TestParseHerdsmanLaunchPresets(t *testing.T) {
	dir := t.TempDir()
	f1 := `[
	  {"model_name":"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2","inference_engine":"llama.cpp","start_time":"2026-08-13T10:00:00+08:00","port":8896,"status":"success","options":{"context_size":262144,"gpu_layers":99,"cache_type_k":"f16","no_mmap":true}},
	  {"model_name":"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2","inference_engine":"llama.cpp","start_time":"2026-08-13T09:00:00+08:00","port":24331,"status":"success","options":{"context_size":4096}},
	  {"model_name":"old-model","inference_engine":"llama.cpp","start_time":"2026-08-12T08:00:00+08:00","port":1,"status":"failed","options":{}}
	]`
	f2 := `[{"model_name":"Gemma4:12B-IT","inference_engine":"llama.cpp","start_time":"2026-08-13T11:58:31+08:00","port":48503,"status":"success","options":{"context_size":262144,"mmproj":"mmproj-BF16.gguf"}}]`
	if err := os.WriteFile(filepath.Join(dir, "qwen.json"), []byte(f1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gemma.json"), []byte(f2), 0o644); err != nil {
		t.Fatal(err)
	}
	presets, err := parseHerdsmanLaunchPresets(dir)
	if err != nil {
		t.Fatalf("parseHerdsmanLaunchPresets: %v", err)
	}
	if len(presets) != 2 {
		t.Fatalf("len(presets) = %d, want 2", len(presets))
	}
	// 按模型名排序：Gemma4:12B-IT 在前。
	if presets[0].Model != "Gemma4:12B-IT" || presets[1].Model != "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2" {
		t.Fatalf("排序错误: %+v", presets)
	}
	// 每个模型取最近一次成功启动。
	if got := presets[1].Options["context_size"]; got != float64(262144) {
		t.Errorf("应取最近成功记录 context_size=%v", got)
	}
	if presets[1].Port != 8896 {
		t.Errorf("Port = %d, want 8896", presets[1].Port)
	}
}

func TestHerdsmanLaunchPresets_EmptyDir(t *testing.T) {
	presets, err := parseHerdsmanLaunchPresets(t.TempDir())
	if err != nil || len(presets) != 0 {
		t.Fatalf("空目录应返回空: presets=%v err=%v", presets, err)
	}
}

func TestHerdsmanDataDir(t *testing.T) {
	t.Setenv("HERDSMAN_DATA_DIR", `D:\herdsman-data`)
	if got := herdsmanDataDir(); got != `D:\herdsman-data` {
		t.Fatalf("herdsmanDataDir() = %q", got)
	}
}

func TestHerdsmanLifecycleHandlers(t *testing.T) {
	origList, origLong := herdsmanCLI, herdsmanCLIWithTimeout
	defer func() { herdsmanCLI, herdsmanCLIWithTimeout = origList, origLong }()

	herdsmanCLIWithTimeout = func(_ time.Duration, args ...string) ([]byte, error) {
		return []byte(`{"ok":true,"result":{"status":"completed"}}`), nil
	}
	a := &App{}
	for name, fn := range map[string]func(string) (HerdsmanOpResult, error){
		"start":     a.HerdsmanModelStart,
		"stop":      a.HerdsmanModelStop,
		"download":  a.HerdsmanModelDownload,
		"uninstall": a.HerdsmanModelUninstall,
	} {
		r, err := fn("bge-m3")
		if err != nil || !r.OK || r.Status != "completed" {
			t.Fatalf("%s 成功路径失败: r=%+v err=%v", name, r, err)
		}
		if _, err := fn(""); err == nil {
			t.Fatalf("%s 空模型名应报错", name)
		}
	}

	// CLI 失败透出。
	herdsmanCLIWithTimeout = func(_ time.Duration, args ...string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	if _, err := a.HerdsmanModelStop("bge-m3"); err == nil {
		t.Fatal("CLI 失败应透出")
	}
}
