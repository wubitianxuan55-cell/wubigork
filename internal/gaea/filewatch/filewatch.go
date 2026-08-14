// Package filewatch 工作区实时文件监听（阶段 5 T5-2）。
//
// 基于 fsnotify（Windows 上为 ReadDirectoryChangesW）：监听工作区目录树，
// 事件去抖合并后输出批次（changed/removed/full），供调用方做增量语义索引
// （复用 semantic.Store 的内容感知 Ensure + Remove）。目录级变更与事件风暴
// 标记 Full（建议全量重建）；监听不可用时 WatchErr 非空，调用方回退轮询。
package filewatch

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultSkipDirs 扫描/监听时跳过的内部与产物目录（与 fileindex 保持一致）。
var DefaultSkipDirs = []string{
	".git", "node_modules", "dist", "build", ".venv", "vendor",
	".gaea", "releases", ".tmp", "backups",
}

// Event 是一次去抖合并后的事件批次。
type Event struct {
	// Changed 相对路径列表（新增/修改/重命名新名，/ 分隔）。
	Changed []string
	// Removed 相对路径列表（删除/重命名旧名，/ 分隔）。
	Removed []string
	// Full 目录级变更或事件风暴：建议全量重建而非逐文件增量。
	Full bool
}

// Watcher 工作区目录监听器。
type Watcher struct {
	root     string
	skipDirs map[string]bool
	debounce time.Duration
	stormN   int // 超过该数量的单文件事件合并为 Full

	fs        *fsnotify.Watcher
	out       chan Event
	done      chan struct{}
	closeOnce sync.Once

	mu       sync.Mutex
	watchErr error // 监听过程中出现的错误（目录不可加等），调用方据此回退
}

// New 创建监听器（未开始监听；调用 Start）。
func New(root string, skipDirs []string, debounce time.Duration) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(skipDirs))
	for _, d := range skipDirs {
		m[d] = true
	}
	if debounce <= 0 {
		debounce = 2 * time.Second
	}
	return &Watcher{
		root:     root,
		skipDirs: m,
		debounce: debounce,
		stormN:   50,
		fs:       fs,
		out:      make(chan Event, 8),
		done:     make(chan struct{}),
	}, nil
}

// Healthy 报告监听是否健康（无致命错误）。轮询兜底据此判断是否跳过。
func (w *Watcher) Healthy() bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.watchErr == nil
}

// WatchErr 返回监听错误（nil=健康）。
func (w *Watcher) WatchErr() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.watchErr
}

// Events 返回去抖事件批次通道（Start 后可用）。
func (w *Watcher) Events() <-chan Event { return w.out }

// Start 开始监听：递归添加目录（跳过 skipDirs），启动事件循环。
// 目录添加失败不致命（记录 WatchErr，仍监听已成功的子树）。
func (w *Watcher) Start() error {
	if err := w.addTree(w.root); err != nil {
		w.setErr(err)
	}
	go w.loop()
	return nil
}

// Close 停止监听并关闭事件通道。
func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		err = w.fs.Close()
	})
	return err
}

// ─── 内部 ─────────────────────────────────────────────────

func (w *Watcher) setErr(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watchErr == nil {
		w.watchErr = err
	}
	slog.Warn("filewatch: 监听异常", "error", err)
}

// addTree 递归添加目录到监听。
func (w *Watcher) addTree(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != dir && w.skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if err := w.fs.Add(path); err != nil {
			slog.Warn("filewatch: 目录监听添加失败", "dir", path, "error", err)
			w.setErr(err)
		}
		return nil
	})
}

func (w *Watcher) loop() {
	changed := map[string]bool{}
	removed := map[string]bool{}
	full := false
	var timer *time.Timer
	var timerC <-chan time.Time

	flush := func() {
		if len(changed) == 0 && len(removed) == 0 && !full {
			return
		}
		ev := Event{Full: full}
		for p := range changed {
			ev.Changed = append(ev.Changed, p)
		}
		for p := range removed {
			ev.Removed = append(ev.Removed, p)
		}
		sort.Strings(ev.Changed)
		sort.Strings(ev.Removed)
		// 输出通道满（消费方积压）：升级为 Full 并丢弃明细，避免无限阻塞
		select {
		case w.out <- ev:
		default:
			fullEv := Event{Full: true}
			select {
			case w.out <- fullEv:
			default:
			}
		}
		changed = map[string]bool{}
		removed = map[string]bool{}
		full = false
	}

	for {
		select {
		case <-w.done:
			flush()
			close(w.out)
			return
		case <-timerC:
			flush()
			timerC = nil
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			w.setErr(err)
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			rel, relErr := filepath.Rel(w.root, ev.Name)
			if relErr != nil || strings.HasPrefix(rel, "..") {
				continue
			}
			rel = filepath.ToSlash(rel)
			// 跳过内部目录下的文件事件（skipDirs 子树）
			if skipRel(w.skipDirs, rel) {
				continue
			}
			isDir := false
			if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
				isDir = true
			}
			switch {
			case ev.Op&fsnotify.Remove != 0:
				if isDir {
					full = true
					_ = w.fs.Remove(ev.Name)
				} else {
					removed[rel] = true
				}
			case ev.Op&fsnotify.Rename != 0:
				if isDir {
					full = true
					_ = w.fs.Remove(ev.Name)
				} else {
					removed[rel] = true
					changed[rel] = true // 新名未知：保守重嵌（Ensure 内容感知会跳过未变）
				}
			case ev.Op&fsnotify.Create != 0:
				if isDir {
					// 新目录：加入监听（递归），目录内容视为全量变更
					_ = w.addTree(ev.Name)
					full = true
				} else {
					changed[rel] = true
				}
			case ev.Op&fsnotify.Write != 0:
				changed[rel] = true
			}
			if len(changed)+len(removed) > w.stormN {
				full = true
			}
			if timerC == nil {
				timer = time.NewTimer(w.debounce)
				timerC = timer.C
			}
		}
	}
}

func skipRel(skip map[string]bool, rel string) bool {
	parts := strings.Split(rel, "/")
	if len(parts) == 0 {
		return false
	}
	return skip[parts[0]]
}

