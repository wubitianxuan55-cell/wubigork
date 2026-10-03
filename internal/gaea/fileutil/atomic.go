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
// 需要断电级 durability（落盘后才 rename）的站点改用 AtomicWriteSync。
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	return atomicWrite(path, data, perm, false)
}

// AtomicWriteSync 是 AtomicWrite 的断电级变体（A7 fsync 决策落地，2026-10-03）：
// rename 前对临时件 fsync——崩溃/断电后目标要么是旧完整内容、要么是新完整
// 内容，绝不出现「rename 成功但数据仍在页缓存」的空窗。默认关闭（性能），
// 显式 Sync 的站点（报告/编辑器写盘）走本变体。
func AtomicWriteSync(path string, data []byte, perm os.FileMode) error {
	return atomicWrite(path, data, perm, true)
}

func atomicWrite(path string, data []byte, perm os.FileMode, sync bool) error {
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
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if sync {
		if err := tmp.Sync(); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("sync temp: %w", err)
		}
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	// perm 参数生效（审计 512 条·上报未动池收口）：CreateTemp 恒以 0600 创建，
	// 直接 rename 会把调用方传入的权限位静默失真为 0600（editfile 工具「保持
	// 原文件权限位」的文档承诺此前实际未兑现）。rename 前对仍在己手的临时件
	// chmod，不产生错权限的可见中间态；Unix 上精确生效（不经 umask——调用方
	// 全部传显式值，无 umask 依赖语义要保留），Windows 上 Chmod 仅切换只读位
	// （0600/0644 均可写，等效 no-op）。
	if err := os.Chmod(tmpPath, perm.Perm()); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod %s: %w", tmpPath, err)
	}
	if err := RenameWithRetry(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
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
