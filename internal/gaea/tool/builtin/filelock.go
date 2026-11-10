package builtin

// 写锁 sidecar（D4，dsh-ops bind-before-publish 蒸馏，2026-11-10）。
//
// 双执行体同树（DSH+ZCode/双开 app）的丢失更新窗口：A3 revision CAS 防
// 「基于陈旧读的写」（opt-in），但「CAS 通过→rename 前」的跨进程 TOCTOU 无
// 锁可挡（fileutil 注释自认；internal/app/gaea_task_inbox.go:95-98 明文记载
// 「需要文件锁，属另一刀」——本文件即那另一刀）。
//
// 协议（上游 file_update_lock.rs 同构，如实缩小两处记档）：
//   - acquire=O_CREATE|O_EXCL 哨兵创建+写进程唯一 token（EEXIST→退避重试，
//     仍占=LOCK_CONTENTION 可重试失败）；不用 LockFileEx/flock——gaea 全树
//     零 OS 锁先例，O_EXCL 哨兵有先例（attachment/store.go:231）且免平台分文件；
//   - bind=锁定时 stat 目标存身份快照；publish 前 re-stat 比对 os.SameFile
//     （目标被替换=conflict）；release=回读 token 相符才删锁（防删别人重建的锁）；
//   - stale 回收=锁龄>10s 且 token 畸形视为死持有者回收（上游用 PID liveness，
//     年龄阈值是简单如实的取舍——锁是运行时哨兵不是数据，与「绝不按 mtime 猜
//     数据归属」的磁盘纪律不冲突）。
//
// sidecar 名 `~$<base>.gaea.lock`（目标同目录）：`~$` 前缀天然被 PublishGuard
// 剪枝（publishguard.go:208 跳 Office 锁件），短生命周期不污染正文。

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

const fileLockStaleAge = 10 * time.Second

// fileLockDelays 退避节奏（RenameWithRetry 同款 5/10/20ms 家族）。
var fileLockDelays = []time.Duration{5, 10, 20, 20, 40, 40, 80, 80}

// newFileLockToken 造持有者 token（进程启动时取一次：进程内写者由 agent 批
// 冲突键串行，同进程并发同文件不会发生；跨进程靠 token 区分持有者）。
func newFileLockToken() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

var fileLockToken = newFileLockToken()

// lockSidecarPath 目标文件的锁哨兵路径（同目录）。
func lockSidecarPath(target string) string {
	return filepath.Join(filepath.Dir(target), "~$"+filepath.Base(target)+".gaea.lock")
}

// acquireFileLock 拿哨兵（O_EXCL+token）。锁龄超阈且 token 畸形→回收重试。
func acquireFileLock(target string) error {
	sidecar := lockSidecarPath(target)
	for attempt, delay := range fileLockDelays {
		f, err := os.OpenFile(sidecar, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			if _, werr := f.WriteString(fileLockToken); werr != nil {
				f.Close()
				_ = os.Remove(sidecar)
				return fmt.Errorf("lock %s: %w", sidecar, werr)
			}
			return f.Close() // token 落盘即持有（重开校验防误删用内容比对）
		}
		if !os.IsExist(err) {
			return err
		}
		// stale 回收：高龄+token 畸形 → 死持有者，删了重试。
		if fi, serr := os.Stat(sidecar); serr == nil && time.Since(fi.ModTime()) > fileLockStaleAge {
			if raw, rerr := os.ReadFile(sidecar); rerr != nil || !validLockToken(string(raw)) {
				_ = os.Remove(sidecar)
				continue
			}
		}
		if attempt < len(fileLockDelays)-1 {
			time.Sleep(delay)
		}
	}
	return &tool.Failure{
		Code:        tool.CodeExecError,
		Subcode:     "LOCK_CONTENTION",
		Err:         fmt.Errorf("file lock contention on %s (另一执行体正在写)", target),
		Retryable:   true,
		EffectState: tool.EffectNone,
		NextAction:  "另一执行体持有该文件写锁：稍后重试，或先协调避免双开同时写",
	}
}

// validLockToken 粗验持有者 token 形状（24 hex 或 fallback- 前缀）。
func validLockToken(s string) bool {
	if len(s) >= 10 && s[:9] == "fallback-" {
		return true
	}
	if len(s) != 24 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// releaseFileLock 回读 token 相符才删（防删别人重建的锁）。
func releaseFileLock(target string) {
	sidecar := lockSidecarPath(target)
	raw, err := os.ReadFile(sidecar)
	if err != nil || string(raw) != fileLockToken {
		return // 不是我们的锁（极端窗口被别人回收重建）——留着
	}
	_ = os.Remove(sidecar)
}

// fileLockBind 目标身份快照（bind-before-publish 的 bind）。
type fileLockBind struct {
	existed bool
	stat    os.FileInfo
}

// bindFileLockTarget 锁定时刻的目标身份。
func bindFileLockTarget(target string) (fileLockBind, error) {
	fi, err := os.Stat(target)
	if err == nil {
		return fileLockBind{existed: true, stat: fi}, nil
	}
	if os.IsNotExist(err) {
		return fileLockBind{}, nil // absent 快照（新增文件场景）
	}
	return fileLockBind{}, err
}

// verifyBind publish 前重核：目标身份漂移=conflict。
func (b fileLockBind) verify(target string) error {
	fi, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			if !b.existed {
				return nil // 仍然 absent ✔
			}
			return &tool.Failure{
				Code:        tool.CodeConflict,
				Err:         fmt.Errorf("REVISION_CONFLICT: %s 在锁定期间被删除", target),
				Retryable:   true,
				EffectState: tool.EffectNone,
				NextAction:  "重读现状后再决定重建或换路径",
			}
		}
		return err
	}
	if !b.existed {
		return &tool.Failure{
			Code:        tool.CodeConflict,
			Err:         fmt.Errorf("REVISION_CONFLICT: %s 在锁定期间被外部创建", target),
			Retryable:   true,
			EffectState: tool.EffectNone,
			NextAction:  "目标已出现：重读后再决定覆盖或换路径",
		}
	}
	if !os.SameFile(b.stat, fi) {
		return &tool.Failure{
			Code:        tool.CodeConflict,
			Err:         fmt.Errorf("REVISION_CONFLICT: %s 在锁定期间被外部替换（inode 漂移）", target),
			Retryable:   true,
			EffectState: tool.EffectNone,
			NextAction:  "文件已被替换：重读拿最新内容后再改",
		}
	}
	return nil
}

// WithFileLock 包住「读→CAS→写」整段：acquire → bind → fn → release。
// fn 返回错误原样透传（锁保证释放）。bind 校验由调用方在 publish 前显式调
// bind.verify（读改写形态各异，不越权代劳）。
func WithFileLock(target string, fn func() error) error {
	if err := acquireFileLock(target); err != nil {
		return err
	}
	defer releaseFileLock(target)
	return fn()
}
