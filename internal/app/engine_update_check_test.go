package app

// engine_update_check_test.go — 引擎官方版本检查测试。纯函数（版本抓取/比较）
// 与注入探针的编排逻辑；绝不真 exec 引擎、绝不真联网（app 测试隔离铁律）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

func TestParseFirstVersion(t *testing.T) {
	cases := []struct{ name, out, want string }{
		{"whirl", "whirl-server 0.2.5.1 (WHIRL - Windows HIP Inference for RDNA LLMs)", "0.2.5.1"},
		{"llama-dev", "version: 0.8.0-dev (build 1, commit f7c01c5)", "0.8.0"},
		{"llama-bnum", "version: 4559 (3efa77d)\nbuilt with MSVC 19.44 for x64", "4559"},
		{"strata", "strata 0.1.56", "0.1.56"},
		{"v-prefix", "v1.2.3 release", "1.2.3"},
		{"garbage", "no version here", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		if got := parseFirstVersion(tc.out); got != tc.want {
			t.Fatalf("%s: parse = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.2.5.1", "0.2.5.1", 0},
		{"0.2.5", "0.2.5.0", 0},   // 段数补零
		{"0.1.56", "0.2.13", -1},  // 段内数值比较（56>13 但 0.1<0.2）
		{"0.8.0", "0.9.0", -1},
		{"0.10.0", "0.9.0", 1},    // 数值比较非字典序
		{"1.0", "1.0.0", 0},
	}
	for _, tc := range cases {
		if got := compareVersions(tc.a, tc.b); got != tc.want {
			t.Fatalf("compare(%q,%q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestEngineUpdateCheck_InjectedProbes 注入本地/官方两侧探针，钉死编排契约：
// 缓存命中、has_update 判定、官方失败仍回本地版本、bin 缺失即错。
func TestEngineUpdateCheck_InjectedProbes(t *testing.T) {
	ResetEngineUpdateCacheForTests()
	defer ResetEngineUpdateCacheForTests()

	origProbe, origFetch := engineVersionProbe, engineUpdateFetchLatest
	defer func() { engineVersionProbe, engineUpdateFetchLatest = origProbe, origFetch }()

	// 全 hermetic：whirl 指向不存在的目录（bin-missing 用例），strata 指向
	// 临时目录的哑 bin 文件（编排用例）。
	whirlRoot := t.TempDir()
	a := &App{core: &core{cfg: &config.Config{WhirlPath: whirlRoot, StrataPath: t.TempDir()}}}

	// bin 不存在（注入探针也不该被走到——stat 先拦）
	probeCalls := 0
	engineVersionProbe = func(bin string) (string, error) {
		probeCalls++
		return "", fmt.Errorf("should not run")
	}
	res := a.EngineUpdateCheck("whirl")
	if res["error"] == "" {
		t.Fatalf("bin 缺失应报错: %#v", res)
	}
	if probeCalls != 0 {
		t.Fatalf("bin 缺失不应触发探针")
	}

	// 正常路径：本地落后于官方
	engineVersionProbe = func(string) (string, error) {
		return "strata 0.1.56\n", nil
	}
	engineUpdateFetchLatest = func(repo string) (string, string, string, error) {
		if repo != "Niko1221/Strata" {
			return "", "", "", fmt.Errorf("unexpected repo %q", repo)
		}
		return "v0.2.13", "Strata 0.2.13", "https://github.com/Niko1221/Strata/releases/tag/v0.2.13", nil
	}
	if err := os.MkdirAll(filepath.Join(a.cfg.StrataPath, "engine"), 0o755); err != nil {
		t.Fatal(err)
	}
	dummy := filepath.Join(a.cfg.StrataPath, "engine", "strata.exe")
	if err := os.WriteFile(dummy, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}
	res = a.EngineUpdateCheck("strata")
	if got := res["local_version"]; got != "0.1.56" {
		t.Fatalf("local_version = %v", got)
	}
	if got := res["latest_version"]; got != "0.2.13" {
		t.Fatalf("latest_version = %v", got)
	}
	if res["has_update"] != true {
		t.Fatalf("has_update = %v, want true", res["has_update"])
	}

	// 缓存命中：fetch 计数不再增长
	fetchCalls := 0
	engineUpdateFetchLatest = func(string) (string, string, string, error) {
		fetchCalls++
		return "v0.2.13", "", "", nil
	}
	_ = a.EngineUpdateCheck("strata")
	_ = a.EngineUpdateCheck("strata")
	if fetchCalls != 0 {
		t.Fatalf("缓存 TTL 内不应再打官方接口")
	}

	// 官方侧失败：本地版本照样返回，error 非空
	ResetEngineUpdateCacheForTests()
	engineUpdateFetchLatest = func(string) (string, string, string, error) {
		return "", "", "", fmt.Errorf("HTTP 403")
	}
	res = a.EngineUpdateCheck("strata")
	if res["local_version"] != "0.1.56" || res["error"] == "" {
		t.Fatalf("官方失败应保留本地版本并带 error: %#v", res)
	}
}

func TestEngineUpdateCheck_UnknownEngine(t *testing.T) {
	a := &App{}
	res := a.EngineUpdateCheck("herdsman")
	if !strings.Contains(fmt.Sprint(res["error"]), "不支持") {
		t.Fatalf("未知引擎应拒绝: %#v", res)
	}
}
