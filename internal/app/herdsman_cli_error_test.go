package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// v4.9.1 真机复盘：Herdsman 桌面端以管理员运行时，skill 管道对普通权限的
// gaea 拒绝访问，CLI 把结构化错误写 stdout 后 exit 3——旧 runHerdsmanCLI
// 在失败路径丢弃 stdout，模型中心只见「exit status 3」+ 误导性的
// 「请确认桌面端已启动」。本测试锁死：失败路径必须透出 CLI 真实错误。
func TestRunHerdsmanCLI_SurfacesCLIErrorOnNonZeroExit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows 专用（.cmd 假 CLI）")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-herdsman.cmd")
	// 模拟真实 CLI 行为：结构化错误写 stdout，进程退出码 3。
	// 注意：message 不含反斜杠——cmd echo 会折叠 \\，导致 JSON 转义失效
	//（真实 CLI 无此问题；此处只验证「非零退出→透出 stdout 结构化错误」）。
	content := "@echo off\r\n" +
		`echo {"version":1,"ok":false,"error":{"code":"unavailable","message":"herdsman desktop process is not running: open Herdsman-skill-pipe: Access is denied."}}` + "\r\n" +
		"exit /b 3\r\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	t.Setenv("HERDSMAN_EXE", script)

	_, err := runHerdsmanCLI(10_000_000_000, "skill", "models", "list", "--json")
	if err == nil {
		t.Fatal("exit 3 应返回错误")
	}
	msg := err.Error()
	if strings.Contains(msg, "exit status") {
		t.Errorf("不应只回显裸退出码, got %q", msg)
	}
	if !strings.Contains(msg, "Access is denied") {
		t.Errorf("应透出 CLI 真实错误文案, got %q", msg)
	}
	if !strings.Contains(msg, "普通方式重启 Herdsman") {
		t.Errorf("Access denied 应附定向修复提示, got %q", msg)
	}
}

func TestHerdsmanEnvelopeError_TwoShapes(t *testing.T) {
	// 对象态（v4.9.1 真机实测形态）
	obj := []byte(`{"version":1,"ok":false,"error":{"code":"unavailable","message":"pipe denied"}}`)
	if got := herdsmanEnvelopeError(obj); got != "pipe denied" {
		t.Errorf("对象态 error 应提取 message, got %q", got)
	}
	// 字符串态（老版本/操作命令形态）
	str := []byte(`{"ok":false,"error":"模型启动失败"}`)
	if got := herdsmanEnvelopeError(str); got != "模型启动失败" {
		t.Errorf("字符串态 error 应提取, got %q", got)
	}
	// ok=true / 非 JSON
	if got := herdsmanEnvelopeError([]byte(`{"ok":true}`)); got != "" {
		t.Errorf("ok=true 不应有错误, got %q", got)
	}
	if got := herdsmanEnvelopeError([]byte(`nope`)); got != "" {
		t.Errorf("非 JSON 应安全返回空, got %q", got)
	}
	// BOM 兼容（PowerShell 管道产物）
	bom := append([]byte("\xef\xbb\xbf"), []byte(`{"ok":false,"error":"带BOM"}`)...)
	if got := herdsmanEnvelopeError(bom); got != "带BOM" {
		t.Errorf("BOM 应被剥离, got %q", got)
	}
}

func TestHerdsmanErrorHint(t *testing.T) {
	got := herdsmanErrorHint(`open \\.\pipe\Herdsman-skill-v1: Access is denied.`)
	if !strings.Contains(got, "管理员") || !strings.Contains(got, "普通方式重启") {
		t.Errorf("Access denied 应给定向提示, got %q", got)
	}
	other := herdsmanErrorHint("some other failure")
	if other != "some other failure" {
		t.Errorf("其它错误应原样透传, got %q", other)
	}
}
