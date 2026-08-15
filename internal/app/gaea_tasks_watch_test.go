package app

// gaea_tasks_watch_test.go — T7-3：fileWatchLoop 周期检查 WatchErr 非空 →
// 触发全量重建（兑现「实时监听异常回退轮询」承诺）；回退轮询周期提交索引任务。
// 用 fake 监听器（fileWatchSource 接口）注入，不依赖真实 fsnotify。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/filewatch"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

// fakeWatchSource 实现 fileWatchSource 接口（Events + WatchErr）。
type fakeWatchSource struct {
	err    error
	events chan filewatch.Event
}

func (f *fakeWatchSource) Events() <-chan filewatch.Event { return f.events }
func (f *fakeWatchSource) WatchErr() error                { return f.err }

// registerNoopIndex 注册一个立即成功的 file_index handler（避免任务执行失败噪声）。
func registerNoopIndex(t *testing.T, a *App) {
	t.Helper()
	a.officeState.tasks.Register(tasks.KindFileIndex, func(ctx context.Context, tk *tasks.Task, p *tasks.Progress) error {
		return nil
	})
}

// waitForTaskPayload 轮询任务列表直到出现 Kind=file_index 且 Payload 含子串的任务。
func waitForTaskPayload(t *testing.T, a *App, substr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		list, err := a.taskMgr().List(100)
		if err == nil {
			for _, tk := range list {
				if tk.Kind == string(tasks.KindFileIndex) && strings.Contains(tk.Payload, substr) {
					return
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待任务 payload 含 %q 超时", substr)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestFileWatchLoopWatchErrTriggersFullRebuild WatchErr 非空时，周期健康检查
// 触发全量重建任务（reason=watch-error），兑现「监听异常回退」承诺。
func TestFileWatchLoopWatchErrTriggersFullRebuild(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	registerNoopIndex(t, a)

	fake := &fakeWatchSource{err: errors.New("fsnotify broke"), events: make(chan filewatch.Event)}
	go a.fileWatchLoopWith(fake, 10*time.Millisecond)
	t.Cleanup(func() { close(fake.events) })

	waitForTaskPayload(t, a, "watch-error")
}

// TestFileWatchLoopHealthyNoRebuild 监听健康（WatchErr=nil）时不触发任何重建。
func TestFileWatchLoopHealthyNoRebuild(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	registerNoopIndex(t, a)

	fake := &fakeWatchSource{err: nil, events: make(chan filewatch.Event)}
	go a.fileWatchLoopWith(fake, 10*time.Millisecond)
	t.Cleanup(func() { close(fake.events) })

	// 给几个健康周期的时间
	time.Sleep(80 * time.Millisecond)
	list, _ := a.taskMgr().List(100)
	for _, tk := range list {
		if tk.Kind == string(tasks.KindFileIndex) {
			t.Fatalf("健康监听不应触发重建任务: %+v", tk)
		}
	}
}

// TestFileWatchLoopFullEventStillTriggersRebuild 事件通道照常工作：Full 事件
// 仍然触发 watch-full 全量重建。
func TestFileWatchLoopFullEventStillTriggersRebuild(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	registerNoopIndex(t, a)

	fake := &fakeWatchSource{err: nil, events: make(chan filewatch.Event)}
	go a.fileWatchLoopWith(fake, time.Hour) // 健康检查周期极长，避免干扰
	t.Cleanup(func() { close(fake.events) })

	fake.events <- filewatch.Event{Full: true}
	waitForTaskPayload(t, a, "watch-full")
}

// TestWatchPollingFallbackSubmitsTasks 回退轮询：周期提交全量索引任务
// （reason=watch-poll-*），兑现「实时监听不可用回退轮询」承诺。
func TestWatchPollingFallbackSubmitsTasks(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	registerNoopIndex(t, a)

	old := watchPollInterval
	watchPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { watchPollInterval = old })

	stop := a.startWatchPollingFallback("test")
	t.Cleanup(stop)

	waitForTaskPayload(t, a, "watch-poll-test")
}
