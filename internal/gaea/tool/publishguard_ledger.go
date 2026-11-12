package tool

// D6 清理债务账本（dsh-ops 蒸馏 2026-11-10 第二波）：「先记账后放句柄」——
// Quarantine 移动文件**之前**先向 <quarantine>/ledger.jsonl 追加 pending 记录
// （stamp/时间/根/条目/来源），移动后补结果记录；清扫只删账本背书的过期
// stamp，账本外名字=整 stamp 跳过（绝不按名字/mtime 猜测归属）；失败推迟
// 不遗忘（sweep_failed 留账下次重试）。存量无账本 stamp 不自动删（手动处理）。
//
// 上游纪律（distill §8.2）：清理债务账本=「首个句柄发出之前先落持久记录，
// 清理失败被推迟而非遗忘」；「绝不按名字模式、mtime 或死 PID 删除」。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// quarantineLedgerFile 账本文件名（隔离根下，与 stamp 目录平级）。
const quarantineLedgerFile = "ledger.jsonl"

// 隔离 stamp 保留期：账本创建时间起算（不看目录 mtime——copy 兜底不保 mtime）。
const quarantineRetention = 30 * 24 * time.Hour

var quarantineLedgerMu sync.Mutex // 前台调用与后台 job 完成回调可能并发隔离

// quarantineEntry 一条越界件的意图记录。
type quarantineEntry struct {
	RelPath string `json:"rel"`
	Kind    string `json:"kind"`
	Size    int64  `json:"size"`
}

// quarantineRecord 账本记录（JSONL 每行一条；同 stamp 两条：pending→终态）。
type quarantineRecord struct {
	Version int               `json:"version"`
	Stamp   string            `json:"stamp"`
	Time    time.Time         `json:"time"`
	Roots   []string          `json:"roots"`
	Entries []quarantineEntry `json:"entries"`
	Source  string            `json:"source"`            // "foreground" / "background"
	Command string            `json:"command,omitempty"` // 命令预览（截断）
	Status  string            `json:"status"`            // pending/quarantined/partial/swept/sweep_failed
	Moved   []string          `json:"moved,omitempty"`
	Errors  []string          `json:"errors,omitempty"`
	SweptAt time.Time         `json:"swept_at,omitempty"`
}

// appendQuarantineRecord 追加一条账本记录（fsync 防崩溃丢账）。
func appendQuarantineRecord(quarantineRoot string, rec quarantineRecord) error {
	quarantineLedgerMu.Lock()
	defer quarantineLedgerMu.Unlock()
	if err := os.MkdirAll(quarantineRoot, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(quarantineRoot, quarantineLedgerFile),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	_ = f.Close()
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// ledged 挂在 PublishGuard 上的账本上下文（Quarantine 一次生命周期内使用）。
type quarantineLedger struct {
	root    string
	rec     quarantineRecord
	pending bool // pending 是否成功落账（失败=该 stamp 永不自动删）
}

// beginQuarantineRecord 先记账（在任何 MkdirAll/Rename 之前）。
func beginQuarantineRecord(quarantineRoot, source, command string, vs []PublishViolation) *quarantineLedger {
	led := &quarantineLedger{root: quarantineRoot}
	led.rec = quarantineRecord{
		Version: 1,
		Stamp:   time.Now().Format("20060102-150405.000000000"),
		Time:    time.Now(),
		Roots:   []string{},
		Source:  source,
		Command: truncateForLedger(command, 200),
		Status:  "pending",
	}
	for _, v := range vs {
		led.rec.Entries = append(led.rec.Entries, quarantineEntry{RelPath: v.RelPath, Kind: v.Kind, Size: v.Size})
		if v.Root != "" {
			led.rec.Roots = appendUnique(led.rec.Roots, v.Root)
		}
	}
	// 账本写失败不阻断隔离（P1b 硬边界优先），该 stamp 视为无主永不自动删。
	if err := appendQuarantineRecord(quarantineRoot, led.rec); err != nil {
		return led // pending=false
	}
	led.pending = true
	return led
}

// finish 移动循环后补终态记录（只有 pending 已落账的才补——无主 stamp 不追踪）。
func (led *quarantineLedger) finish(moved []string, errs []string) {
	if led == nil || !led.pending {
		return
	}
	status := "quarantined"
	if len(errs) > 0 {
		status = "partial"
	}
	led.rec.Status = status
	led.rec.Moved = moved
	led.rec.Errors = errs
	_ = appendQuarantineRecord(led.root, led.rec) // 终态失败留待下次清扫发现 pending 悬挂（保守不删）
}

func truncateForLedger(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func appendUnique(ss []string, v string) []string {
	for _, x := range ss {
		if x == v {
			return ss
		}
	}
	return append(ss, v)
}

// SweepQuarantine 清扫过期隔离 stamp（启动时每工作区异步跑一次）。
// retention==0 取默认 30 天；负数=立即到期（测试/强制清扫用）。只删账本
// 终态 quarantined/partial 且 age>retention 的 stamp；stamp 目录内出现账本
// 外文件=整 stamp 跳过；逐文件删+自底向上删空目录（不用 RemoveAll——并发
// 写者让删除失败而不是丢数据）；失败记 sweep_failed 下次重试。
// 返回删除的 stamp 数与文件数（记档用）。
func SweepQuarantine(quarantineRoot string, retention time.Duration) (stamps, files int, err error) {
	if retention == 0 {
		retention = quarantineRetention
	}
	raw, rerr := os.ReadFile(filepath.Join(quarantineRoot, quarantineLedgerFile))
	if rerr != nil {
		if os.IsNotExist(rerr) {
			return 0, 0, nil // 无账本=无主 stamp，一律不动
		}
		return 0, 0, rerr
	}
	// 同 stamp 取最后一条记录（状态机：pending→quarantined/partial→swept）。
	latest := map[string]quarantineRecord{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec quarantineRecord
		if json.Unmarshal([]byte(line), &rec) == nil {
			latest[rec.Stamp] = rec
		}
	}
	var names []string
	for stamp := range latest {
		names = append(names, stamp)
	}
	sort.Strings(names) // 确定性序
	for _, stamp := range names {
		rec := latest[stamp]
		if rec.Status != "quarantined" && rec.Status != "partial" {
			continue // pending 悬挂（终态没落上）/已清扫/无主：不动
		}
		if time.Since(rec.Time) < retention {
			continue
		}
		n, serr := sweepStamp(quarantineRoot, stamp, rec)
		if serr != nil {
			fin := rec
			fin.Status = "sweep_failed"
			fin.Errors = append(fin.Errors, serr.Error())
			_ = appendQuarantineRecord(quarantineRoot, fin)
			continue
		}
		stamps++
		files += n
		fin := rec
		fin.Status = "swept"
		fin.SweptAt = time.Now()
		_ = appendQuarantineRecord(quarantineRoot, fin)
	}
	return stamps, files, nil
}

// sweepStamp 删一个 stamp 目录（账本背书+名字全核验+自底向上）。
func sweepStamp(quarantineRoot, stamp string, rec quarantineRecord) (int, error) {
	stampDir := filepath.Join(quarantineRoot, stamp)
	expected := map[string]bool{}
	for _, e := range rec.Entries {
		expected[e.RelPath] = true
	}
	var files []string
	err := filepath.WalkDir(stampDir, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			if os.IsNotExist(werr) {
				return nil
			}
			return werr
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(stampDir, path)
		if rerr != nil {
			return rerr
		}
		if !expected[filepath.ToSlash(rel)] {
			return fmt.Errorf("账本外文件 %s（stamp 跳过）", rel)
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return 0, err
	}
	// 自底向上：先删文件，再从深到浅删空目录。
	for _, f := range files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return 0, fmt.Errorf("删 %s: %w", f, err)
		}
	}
	var dirs []string
	_ = filepath.WalkDir(stampDir, func(path string, d os.DirEntry, _ error) error {
		if d != nil && d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(dirs))) // 深的先删
	for _, dir := range dirs {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			// 非空（并发写者）或占用：留着，下次重试。
			continue
		}
	}
	return len(files), nil
}
