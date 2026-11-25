package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// main_sandbox_test.go — app 包测试进程的主目录沙箱（2026-11-26）。
//
// 起因：有不带隔离的用例经 config.Save 把夹具值写进了**真实的**
// ~/.gaea_config.json——wenxue_workspace / image_save_dir 被指向测试临时目录
// （早已删除），voice_persona_presets 被写成「傲娇音 → v1」。手工重定向 APPDATA
// 管不到它：这个文件按家目录（Windows 的 USERPROFILE）定位，不走 APPDATA。
// 任何执行体只要不带隔离跑一遍 app 包，用户配置就被改一次（实证：
// TestWenxueChatSessionBindingSmoke 经 WenxueCreateNovel 自动切换工作空间并落盘）。
//
// 做法：进程级把家目录与 APPDATA 指到临时目录，没有全局逃生口——要读真实
// 环境的诊断另写脚本，别借测试包。单个用例仍可 t.Setenv 自行覆盖（既有写法
// 不变，用例结束自动还原到沙箱值）。

func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "gaea-app-test-home-")
	if err != nil {
		// 建不出沙箱不能悄悄当成已隔离：如实报出来，由跑测试的人决定。
		fmt.Fprintf(os.Stderr, "app 测试沙箱：建临时主目录失败，本轮测试会触及真实家目录: %v\n", err)
	} else {
		data := filepath.Join(home, "AppData", "Roaming")
		if err := os.MkdirAll(data, 0o755); err == nil {
			os.Setenv("APPDATA", data)
		}
		os.Setenv("USERPROFILE", home)
		os.Setenv("HOME", home)
	}
	code := m.Run()
	os.Exit(code)
}

// TestMainSandboxIsolatesUserHome 钉住沙箱生效：家目录与 APPDATA 都落在
// gaea-app-test-home-* 里。
func TestMainSandboxIsolatesUserHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	if !strings.Contains(filepath.Base(home), "gaea-app-test-home-") {
		t.Fatalf("测试进程的家目录不是沙箱：%s——app 包测试会写进真实的 ~/.gaea_config.json", home)
	}
	data, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	if !strings.HasPrefix(filepath.Clean(data), filepath.Clean(home)+string(filepath.Separator)) {
		t.Fatalf("APPDATA 没落在沙箱家目录下：%s（home=%s）", data, home)
	}
}
