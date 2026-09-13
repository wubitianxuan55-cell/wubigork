// Package fileutil provides file system utilities shared across the kernel.
package fileutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AtomicWrite writes data to path atomically: it writes to a temporary
// sibling file and renames it over the target. Readers never see a partial
// write, and a crash mid-write can't leave a truncated file that would fail
// to reload. The parent directory is created if missing.
//
// This is the single shared implementation for the many places that need
// crash-safe persistence (session files, config, memory store, …).
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := RenameWithRetry(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}

// renameRetryBackoff rename 瞬时失败的重试退避序列（毫秒）。
var renameRetryBackoff = []time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 20 * time.Millisecond}

// RenameWithRetry os.Rename with transient-failure retry (v4.293 flaky 根修):
// Windows 上 AV/索引器/备份进程会短暂持有目标文件，覆盖式 rename 报
// "Access is denied"（os.ErrPermission）且绝大多数是瞬时的——本会话 ci 的
// config/schedule 假红均属此类。对权限类错误按退避序列重试 3 次（共 4 次
// 尝试）；其他错误原样快速失败。
func RenameWithRetry(src, dst string) error {
	var err error
	for attempt := 0; attempt <= len(renameRetryBackoff); attempt++ {
		if attempt > 0 {
			time.Sleep(renameRetryBackoff[attempt-1])
		}
		if err = os.Rename(src, dst); err == nil || !errors.Is(err, os.ErrPermission) {
			return err
		}
	}
	return err
}
