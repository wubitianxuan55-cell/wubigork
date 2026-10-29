//go:build windows

package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// TestConfineAcceptsShortNameRoot 回归锁（2026-10-30 远端 CI 实锤）：GH windows
// runner 的 TEMP 是 8.3 短名形态（RUNNER~1），构造方传短名 root、写目标解析成
// 长名时，confine 必须双侧归一后判定，不得误判 outside。真用户短名主目录同险。
func TestConfineAcceptsShortNameRoot(t *testing.T) {
	long := t.TempDir()
	p, err := windows.UTF16PtrFromString(long)
	if err != nil {
		t.Skip("路径无法转 UTF16:", err)
	}
	buf := make([]uint16, 260)
	n, err := windows.GetShortPathName(p, &buf[0], uint32(len(buf)))
	if err != nil || n == 0 {
		t.Skip("该卷未启用 8.3 短名（GetShortPathName 无输出）:", err)
	}
	short := windows.UTF16ToString(buf[:n])
	if short == long {
		t.Skip("短名与长名相同（8.3 未启用），本用例无对象")
	}
	// 短名 root 下放行：confine 双侧归一后应判定在界内
	if err := confine([]string{short}, filepath.Join(long, "out.json")); err != nil {
		t.Fatalf("短名 root 应与长名写目标归一匹配: %v", err)
	}
	// 短名 root 之外仍拒绝（护栏不因归一放松）
	outside := filepath.Dir(long) // TempDir 的上一级（workspace 之外）
	if err := confine([]string{short}, filepath.Join(outside, "escape.txt")); err == nil {
		t.Fatal("root 之外的目标必须拒绝")
	}
	// 短名目录本身可被 os 侧验证（确认拿到的是真 8.3 形态而非恒等）
	if _, err := os.Stat(short); err != nil {
		t.Fatalf("短名路径不可访问: %v", err)
	}
}
