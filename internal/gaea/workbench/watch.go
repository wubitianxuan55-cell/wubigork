package workbench

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gaea/gaea/internal/gaea/filefilter"
)

const watchDebounce = 150 * time.Millisecond

// DirWatch watches exactly the directories the file tree has expanded. Notices
// are debounced and report workspace-relative directory paths, which lets the
// renderer re-list only the affected levels.
type DirWatch struct {
	root string
	fs   *fsnotify.Watcher
	stop chan struct{}
	once sync.Once
}

// NewDirWatch starts watching paths (workspace-relative; "" is the root).
// onChange receives a sorted, de-duplicated set of changed directory paths.
func NewDirWatch(root string, paths, exclude []string, onChange func([]string)) (*DirWatch, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watched := map[string]string{}
	matcher := filefilter.Compile(exclude)
	for _, rel := range paths {
		abs, cleanRel, err := resolveWithin(rootAbs, rel)
		if err != nil {
			_ = fw.Close()
			return nil, err
		}
		cleanSlash := slashRel(cleanRel)
		if cleanSlash != "" && matcher.Match(cleanSlash, filepath.Base(cleanRel)) {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			_ = fw.Close()
			return nil, fmt.Errorf("监听目录不可用: %s（%w）", slashRel(cleanRel), err)
		}
		if !info.IsDir() {
			_ = fw.Close()
			return nil, fmt.Errorf("监听目标不是目录: %s", slashRel(cleanRel))
		}
		if err := fw.Add(abs); err != nil {
			_ = fw.Close()
			return nil, fmt.Errorf("添加目录监听失败: %s（%w）", slashRel(cleanRel), err)
		}
		watched[filepath.Clean(abs)] = slashRel(cleanRel)
	}
	w := &DirWatch{root: rootAbs, fs: fw, stop: make(chan struct{})}
	go w.loop(watched, matcher, onChange)
	return w, nil
}

func (w *DirWatch) Close() error {
	if w == nil {
		return nil
	}
	w.once.Do(func() { close(w.stop) })
	return w.fs.Close()
}

func (w *DirWatch) loop(watched map[string]string, matcher *filefilter.Matcher, onChange func([]string)) {
	if onChange == nil {
		return
	}
	pending := map[string]struct{}{}
	var timer *time.Timer
	var timerC <-chan time.Time
	flush := func() {
		if len(pending) == 0 {
			return
		}
		paths := make([]string, 0, len(pending))
		for p := range pending {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		pending = map[string]struct{}{}
		onChange(paths)
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			eventRel, relErr := filepath.Rel(w.root, ev.Name)
			if relErr != nil || matcher.Match(slashRel(eventRel), filepath.Base(ev.Name)) {
				continue
			}
			dir := filepath.Clean(filepath.Dir(ev.Name))
			rel, found := watched[dir]
			if !found {
				continue
			}
			pending[rel] = struct{}{}
			if timer == nil {
				timer = time.NewTimer(watchDebounce)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(watchDebounce)
			}
			timerC = timer.C
		case <-timerC:
			timerC = nil
			flush()
		}
	}
}
