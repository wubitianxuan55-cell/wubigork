//go:build windows

package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// openDenyingWrite 以「只共享读」打开 path——WPS/Excel 打开文档时的共享模式，
// 别的进程既写不了也删不了它。
func openDenyingWrite(t *testing.T, path string) {
	t.Helper()
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatalf("路径转 UTF-16 失败 %s: %v", path, err)
	}
	h, cerr := syscall.CreateFile(ptr, syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if cerr != nil {
		t.Fatalf("CreateFile(只共享读) %s: %v", path, cerr)
	}
	t.Cleanup(func() { _ = syscall.CloseHandle(h) })
}

// 2026-11-26 造价反馈（旺平 #5）：用户开着交付表时 move_file 落盘失败，
// 只透传 OS 原话。两种占用形态都要点名原因并给出路，且不留半截副作用。
func TestMoveFile_OccupiedDestinationNamesTheCause(t *testing.T) {
	cases := []struct {
		name string
		hold func(t *testing.T, dst string)
	}{
		// 动手之前就被预检拦住（Office 类应用拒绝写共享）。
		{"拒绝写共享", openDenyingWrite},
		// 预检放行、删目标时才撞上（句柄没给 FILE_SHARE_DELETE）。
		{"不给删共享", func(t *testing.T, dst string) {
			f, err := os.Open(dst)
			if err != nil {
				t.Fatalf("open %s: %v", dst, err)
			}
			t.Cleanup(func() { _ = f.Close() })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src, dst := filepath.Join(dir, "new.xlsx"), filepath.Join(dir, "out.xlsx")
			writeTemp(t, dir, "new.xlsx", "NEW")
			writeTemp(t, dir, "out.xlsx", "OLD")
			tc.hold(t, dst)

			m := moveFile{roots: []string{dir}, workDir: dir}
			_, err := m.Execute(context.Background(), json.RawMessage(
				`{"source":"new.xlsx","destination":"out.xlsx","overwrite":true}`))
			if err == nil {
				t.Fatal("目标被占用时必须报错")
			}
			if !strings.Contains(err.Error(), "正被占用") || !strings.Contains(err.Error(), "office_edit") {
				t.Fatalf("应点名占用并指引 office_edit，实得: %v", err)
			}
			if got, _, _ := readFileEncoded(dst); got != "OLD" {
				t.Errorf("占用失败后目标应原样未动，实得 %q", got)
			}
			if _, err := os.Stat(src); err != nil {
				t.Errorf("占用失败后源应仍在: %v", err)
			}
		})
	}
}

// 非占用类失败不得被误贴占用提示（error 种类分不清会把模型引去关 WPS）。
func TestMoveFile_OccupiedHintNotAttachedToOtherErrors(t *testing.T) {
	if occupiedHintFor(nil, os.ErrNotExist) != "" {
		t.Fatal("非占用错误不应带占用提示")
	}
}
