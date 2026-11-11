package app

// whirl_proc_test.go — WHIRL 引擎生命周期测试。只测零副作用面：状态形状、
// 路径解析、启动参数契约与 served-id 归一化（纯函数，不碰真实 8126、不拉模型）。

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

func TestWhirlEngineStatus_Shape(t *testing.T) {
	a := &App{}
	st := a.WhirlEngineStatus()
	for _, key := range []string{"running", "ready", "port", "profile", "profiles", "modelPath", "modelReady", "binPath", "binReady"} {
		if _, ok := st[key]; !ok {
			t.Fatalf("status missing %s key: %v", key, st)
		}
	}
	if st["port"] != whirlPort {
		t.Fatalf("unexpected port: %v", st["port"])
	}
	profiles, ok := st["profiles"].([]map[string]string)
	if !ok || len(profiles) != len(whirlProfiles) {
		t.Fatalf("profiles payload = %#v", st["profiles"])
	}
}

// TestWhirlEngineStartRequiresExplicitProfile 锁死「引擎启动不捆绑默认模型」：
// 空档位直接返回 model-required，不允许隐式拉起 qwen35_whirl。
func TestWhirlEngineStartRequiresExplicitProfile(t *testing.T) {
	a := &App{}
	got := a.WhirlEngineStart("")
	if started, _ := got["started"].(bool); started {
		t.Fatalf("空 profile 不应启动模型: %#v", got)
	}
	if got["reason"] != "model-required" {
		t.Fatalf("reason = %#v, want model-required", got["reason"])
	}
}

func TestWhirlEngineStart_RejectsUnknownProfile(t *testing.T) {
	a := &App{}
	got := a.WhirlEngineStart("turbo")
	if got["reason"] != "unknown-model" {
		t.Fatalf("reason = %#v, want unknown-model", got["reason"])
	}
}

func TestWhirlRootPath_DefaultAndConfigured(t *testing.T) {
	if got := (&App{}).whirlRootPath(); got != config.DefaultWhirlPath {
		t.Fatalf("nil config root = %q, want %q", got, config.DefaultWhirlPath)
	}
	a := &App{core: &core{cfg: &config.Config{WhirlPath: `D:\engines\whirl`}}}
	if got := a.whirlRootPath(); got != `D:\engines\whirl` {
		t.Fatalf("configured root = %q", got)
	}
}

// TestWhirlServeArgs_ExternalInstallContract 钉死启动命令契约：模型路径首位、
// 端口/balance/-np/alias 顺序无关断言；mmproj 仅在文件在位时附加（缺失降级）。
func TestWhirlServeArgs_ExternalInstallContract(t *testing.T) {
	p, ok := engineProfileByID(whirlProfiles, "qwen35_whirl")
	if !ok {
		t.Fatalf("qwen35_whirl profile missing: %#v", whirlProfiles)
	}
	args := whirlServeArgs(p, whirlPort)
	joined := strings.Join(args, " ")

	if args[0] != whirlProfileModelPath(p) {
		t.Fatalf("model path must be first arg: %v", args)
	}
	for _, want := range []string{
		"--port 8126",
		"--balance",
		"-np 1",
		"--alias qwen35_whirl",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q: %s", want, joined)
		}
	}

	mmproj := filepath.Join(voiceV2ModelsRoot, p.ModelDir, p.MMProj)
	if voiceV2FileExists(mmproj) {
		if !strings.Contains(joined, "--mmproj "+mmproj) {
			t.Fatalf("mmproj exists but not passed: %s", joined)
		}
	} else if strings.Contains(joined, "--mmproj") {
		t.Fatalf("mmproj missing but passed anyway: %s", joined)
	}
}

func TestWhirlProfileFromModels(t *testing.T) {
	cases := []struct{ served, want string }{
		{"qwen35_whirl", "qwen35_whirl"},
		{"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2-whirlfix2", "qwen35_whirl"},
	}
	for _, tc := range cases {
		if got := whirlProfileFromModels([]string{tc.served}); got != tc.want {
			t.Fatalf("served %q: profile = %q, want %q", tc.served, got, tc.want)
		}
	}
	for _, model := range []string{"qwen3.6-35b-a3b-ud-q4_k_m", "unknown-model", "qwen35"} {
		if got := whirlProfileFromModels([]string{model}); got != "" {
			t.Fatalf("foreign model %s: profile = %q, want empty (不得误认 llama 档位)", model, got)
		}
	}
}

// TestManagedEngineSpec_IncludesWhirl 目录接线对账：whirl 引擎返回档位目录
//（探活结果随环境，false 合法——引擎未运行不缺目录）。
func TestManagedEngineSpec_IncludesWhirl(t *testing.T) {
	profiles, _, _ := managedEngineSpec("whirl")
	if len(profiles) != len(whirlProfiles) {
		t.Fatalf("managedEngineSpec(whirl) = %#v", profiles)
	}
	if managedEngineProfileFromModels("whirl", []string{"qwen35_whirl"}) != "qwen35_whirl" {
		t.Fatal("managedEngineProfileFromModels(whirl) mismatch")
	}
}
