package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
)

// newCreateChapterSuspendingApp 构造章节生成测试 App：herdsman 指向一个
// 会发出首个 chunk 后挂起（保持连接打开）的 mock LLM 服务，用于并发/取消测试。
// firstChunk 非空时 mock 先流式发送该片段再挂起；ready 收到首个 chunk 已
// 发出（服务端 flush）的信号。
func newCreateChapterSuspendingApp(t *testing.T, ready chan<- struct{}, firstChunk string) (*App, *project.Manager, string) {
	t.Helper()
	hang := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		if firstChunk != "" {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", strconv.Quote(firstChunk))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if ready != nil {
			select {
			case ready <- struct{}{}:
			default:
			}
		}
		// 挂起流：连接保持打开，直到客户端取消（请求 context 结束）或测试结束
		select {
		case <-r.Context().Done():
		case <-hang:
		}
	}))
	t.Cleanup(func() { close(hang); srv.Close() })

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"

	engMgr := modelengine.NewManager("", "")
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}

	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)

	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := project.Create(dir, "测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	t.Cleanup(func() { _ = pm.Close() })

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.ctx = context.Background()
	a.setPM(pm)
	return a, pm, dir
}

// waitFor 轮询等待条件满足，超时失败。
func waitFor(t *testing.T, timeout time.Duration, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("等待超时: %s", desc)
}

// waitGensDone 等待所有章节生成协程结束（登记表清空），避免测试结束时仍有
// 协程在写盘。
func waitGensDone(t *testing.T, a *App) {
	t.Helper()
	waitFor(t, 5*time.Second, "章节生成协程全部结束", func() bool {
		a.chapterGenMu.Lock()
		defer a.chapterGenMu.Unlock()
		return len(a.chapterGenCancels) == 0
	})
}

// TestCreateChapter_SameChapterConcurrentRejected 并发写同一章节仅一次成功：
// 同章节第二次 CreateChapter 被明确拒绝（不会同时写同一 NNN.md），
// 不同章节可并行。
func TestCreateChapter_SameChapterConcurrentRejected(t *testing.T) {
	ready := make(chan struct{})
	a, _, _ := newCreateChapterSuspendingApp(t, ready, "并发测试正文")

	if _, err := a.CreateChapter("设定", "", "剧情", 1, "", "", 3000, 0); err != nil {
		t.Fatalf("第一次 CreateChapter: %v", err)
	}
	<-ready // 第一个流已开始（挂起中，登记生效）

	// 同章节并发生成必须被拒绝，错误须明确
	if _, err := a.CreateChapter("设定", "", "剧情", 1, "", "", 3000, 0); err == nil {
		t.Fatalf("同章节并发生成应被拒绝")
	} else if !strings.Contains(err.Error(), "正在生成") {
		t.Fatalf("拒绝错误应指明正在生成, got: %v", err)
	}

	// 不同章节可并行
	if _, err := a.CreateChapter("设定", "", "剧情", 2, "", "", 3000, 0); err != nil {
		t.Fatalf("不同章节并行生成应成功: %v", err)
	}

	if !a.CancelCreateChapter(1, "") {
		t.Fatalf("CancelCreateChapter(1) 应返回 true")
	}
	if !a.CancelCreateChapter(2, "") {
		t.Fatalf("CancelCreateChapter(2) 应返回 true")
	}
	waitGensDone(t, a)
}

// TestCreateChapter_CancelPreservesPartial 取消后已生成部分被保留并落盘：
// 取消发生在部分正文已流式到达之后，章节文件应包含该片段（不含摘要标记）。
func TestCreateChapter_CancelPreservesPartial(t *testing.T) {
	const firstChunk = "第一章正文片段：林晚踏出山门。"
	ready := make(chan struct{})
	a, pm, _ := newCreateChapterSuspendingApp(t, ready, firstChunk)

	if _, err := a.CreateChapter("设定", "", "剧情", 1, "", "", 3000, 0); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}
	<-ready // mock 已发出首个 chunk

	// 给本机回环的流式消费留出余量，确保部分正文已进入生成循环后再取消
	time.Sleep(500 * time.Millisecond)

	if !a.CancelCreateChapter(1, "") {
		t.Fatalf("CancelCreateChapter(1) 应返回 true")
	}

	path := pm.ChapterPath(1)
	waitFor(t, 5*time.Second, "取消后部分正文落盘", func() bool {
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		return strings.Contains(string(data), firstChunk)
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取章节文件: %v", err)
	}
	if strings.Contains(string(data), "---CHAPTER_SUMMARY---") {
		t.Errorf("取消落盘不应包含摘要标记")
	}
	waitGensDone(t, a)
}

// TestCreateChapter_CancelNoContentNoWrite 取消时尚未生成任何内容：只发
// cancelled 事件，不写空章节文件（避免覆盖既有章节）。
func TestCreateChapter_CancelNoContentNoWrite(t *testing.T) {
	ready := make(chan struct{})
	// firstChunk 为空：mock 只挂起，从不发送正文
	a, pm, _ := newCreateChapterSuspendingApp(t, ready, "")

	if _, err := a.CreateChapter("设定", "", "剧情", 1, "", "", 3000, 0); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}
	<-ready

	if !a.CancelCreateChapter(1, "") {
		t.Fatalf("CancelCreateChapter(1) 应返回 true")
	}
	waitGensDone(t, a)

	if _, err := os.Stat(pm.ChapterPath(1)); !os.IsNotExist(err) {
		t.Fatalf("未生成任何内容时取消不应写章节文件 (err=%v)", err)
	}
}

// TestCancelCreateChapter_Idempotent CancelCreateChapter 幂等：首次取消返回
// true，重复取消与对未开始章节的取消返回 false；主章节取消不影响分支。
func TestCancelCreateChapter_Idempotent(t *testing.T) {
	ready := make(chan struct{})
	a, _, _ := newCreateChapterSuspendingApp(t, ready, "幂等测试正文")

	if a.CancelCreateChapter(3, "") {
		t.Fatalf("未开始生成的章节取消应返回 false")
	}
	if a.CancelCreateChapter(1, "a") {
		t.Fatalf("未开始的分支生成取消应返回 false")
	}

	if _, err := a.CreateChapter("设定", "", "剧情", 1, "", "", 3000, 0); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}
	<-ready

	if !a.CancelCreateChapter(1, "") {
		t.Fatalf("首次取消应返回 true")
	}
	if a.CancelCreateChapter(1, "") {
		t.Fatalf("重复取消应返回 false（幂等）")
	}
	waitGensDone(t, a)
}

// TestSubstituteWordCount_Precise 模板含两处 {word_count} 占位符时仅字数位被
// 替换，其他 "5000" 字样（如历史版本说明）不被误伤。
func TestSubstituteWordCount_Precise(t *testing.T) {
	tmpl := "不少于{word_count}字。\n输出要求：正文不少于{word_count}字。\n参考历史版本：5000 字标准。"
	got := substituteWordCount(tmpl, 3000)
	if strings.Contains(got, "{word_count}") {
		t.Errorf("占位符未全部替换: %s", got)
	}
	if !strings.Contains(got, "不少于3000字") || !strings.Contains(got, "正文不少于3000字") {
		t.Errorf("字数位未替换为实际字数: %s", got)
	}
	if !strings.Contains(got, "5000 字标准") {
		t.Errorf("无关的 5000 字样被误伤: %s", got)
	}
}

// TestCreateChapterTemplateUsesWordCountPlaceholder 真实 create-chapter 模板
// 以 {word_count} 声明目标字数，替换后不残留占位符、不出现字面量 5000。
func TestCreateChapterTemplateUsesWordCountPlaceholder(t *testing.T) {
	eng := prompt.NewEngine("../../prompts")
	tmpl := eng.Get("create-chapter")
	if tmpl == nil {
		t.Fatalf("缺少 create-chapter 模板")
	}
	sys := tmpl.BuildSystemPrompt("")
	if !strings.Contains(sys, wordCountPlaceholder) {
		t.Fatalf("模板应使用 {word_count} 占位符声明目标字数: %s", sys)
	}
	replaced := substituteWordCount(sys, 6666)
	if strings.Contains(replaced, "{word_count}") {
		t.Errorf("替换后不应残留占位符: %s", replaced)
	}
	if !strings.Contains(replaced, "6666") {
		t.Errorf("字数占位应替换为实际字数: %s", replaced)
	}
	if strings.Contains(replaced, "5000") {
		t.Errorf("模板不应再出现字面量 5000: %s", replaced)
	}
}
