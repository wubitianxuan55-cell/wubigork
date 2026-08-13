package app

// H0-1 App 层冒烟测试：HerdsmanProbe 默认参数下不发真实 HTTP 请求，
// 全部隔离在临时目录（USERPROFILE 重定向），不依赖本机 herdsman 安装。

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestHerdsmanProbe 验证 App.HerdsmanProbe：
//   - 默认参数下 APIError=skipped（不发真实 HTTP 请求）；
//   - 根目录解析为 USERPROFILE/.herdsman（测试中已重定向到临时目录）；
//   - 结果可 JSON 序列化，供前端直接展示。
func TestHerdsmanProbe(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HERDSMAN_PROBE_LIVE", "")
	// 固定 HERDSMAN_EXE 指向不存在的文件，避免命中本机真实安装。
	t.Setenv("HERDSMAN_EXE", filepath.Join(home, "missing-herdsman.exe"))

	a := &App{}
	r := a.HerdsmanProbe()

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
	if r.ConfigOK {
		t.Errorf("临时目录下不应存在 config.yaml，ConfigOK 应为 false")
	}
	if len(r.DataFiles) != 4 {
		t.Errorf("DataFiles 应有 4 项，实际 %d", len(r.DataFiles))
	}
	if _, err := json.Marshal(r); err != nil {
		t.Fatalf("探测结果应可 JSON 序列化: %v", err)
	}
}
