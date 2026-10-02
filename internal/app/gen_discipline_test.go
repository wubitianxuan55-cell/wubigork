package app

// 第二轮·线A：后台生成链纪律（审计 P0#4/AP1-01 收敛闭环、P0#5/AP1-02 逐场景生成）。
//
// 两条链都是「后台生成协程」，纪律 = registerChapterGen 登记（同章互斥 + 取消句柄）
// + chapterGenWG 记账（waitGensDone 以它为准）+ 协程退出时 unregisterChapterGen
// 独占清理。本文件只钉这两条链的记账与取消，不改生成语义。
//
// 受控慢速 mock：请求到达后在 gate 上挂起（测试放行），或请求 context 结束（取消/
// 切书）才返回——用「在飞窗口」做确定性断言，不靠 sleep 猜时序。

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// gatedSSEPayload mock 的 SSE 响应体（原始串字面量：本仓转义换行踩过坑，不写转义）。
const gatedSSEPayload = `data: {"choices":[{"delta":{"content":%s},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`

// newGatedLLMApp 构造带受控慢速 mock LLM 的生成 App：
//   - 每个请求到达即向 arrived 非阻塞发一次信号（容量 1）；
//   - 请求在 gate 关闭前挂起；gate 关闭或请求 context 结束（取消/切书）才产出正文。
func newGatedLLMApp(t *testing.T, gate <-chan struct{}, arrived chan<- struct{}, reply string) (*App, *project.Manager) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if arrived != nil {
			select {
			case arrived <- struct{}{}:
			default:
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		select {
		case <-gate:
		case <-r.Context().Done():
			return // 取消/切书：不产出正文，在飞窗口由测试控制
		}
		fmt.Fprintf(w, gatedSSEPayload, strconv.Quote(reply))
	}))
	t.Cleanup(srv.Close)

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

	pm := newChapterGateProject(t)
	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.ctx = context.Background()
	a.setPM(pm)
	waitGensBeforeTempDirRemove(t, a)
	return a, pm
}

// gateReleaser 一次性放行闸（用例 Fatal 退出时由 Cleanup 兜底放行，避免 srv.Close 挂死）。
func gateReleaser(t *testing.T, gate chan struct{}) func() {
	t.Helper()
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	t.Cleanup(release)
	return release
}

// wgIsZero 非阻塞探测 chapterGenWG 是否归零（waitGensDone 的最终判据）。
func wgIsZero(a *App, d time.Duration) bool {
	done := make(chan struct{})
	go func() {
		a.chapterGenWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

// TestSceneCardsGenerateWGCoversAllScenes ①逐场景生成计入 chapterGenWG：
// 首个场景仍在生成（在飞窗口）时 WG 必须非零——waitGensDone 不得提前返回；
// 放行后 waitGensDone 返回 == 全部场景已落盘（不是「首个协程已登记消失」）。
func TestSceneCardsGenerateWGCoversAllScenes(t *testing.T) {
	gate := make(chan struct{})
	release := gateReleaser(t, gate)
	arrived := make(chan struct{}, 1)

	a, pm := newGatedLLMApp(t, gate, arrived, "生成的正文内容。")
	mkCardScene(t, pm, 1, "第一场", true)
	mkCardScene(t, pm, 1, "第二场", true)

	if _, err := a.NovelChapterScenesGenerate(1, false); err != nil {
		t.Fatalf("逐场景生成应放行: %v", err)
	}
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		t.Fatal("5s 内未收到首个场景生成请求")
	}

	// P0#5 红点：协程未计 WG 时 Wait 立即返回，waitGensDone 在场景落盘前放行。
	if wgIsZero(a, 300*time.Millisecond) {
		t.Fatalf("逐场景生成未计入 chapterGenWG：waitGensDone 会在场景落盘前提前返回")
	}

	release()
	waitGensDone(t, a)

	// waitGensDone 等到的是「全部场景写完」。
	sm := pm.SceneManager(1)
	metas, err := sm.List()
	if err != nil || len(metas) != 2 {
		t.Fatalf("列场景: %v (%d)", err, len(metas))
	}
	for _, m := range metas {
		sc, rerr := sm.Read(m.ID)
		if rerr != nil || !strings.Contains(sc.Content, "生成的正文内容") {
			t.Fatalf("场景 %s 正文未落盘（waitGensDone 应等到全部场景写完）: %q err=%v", m.ID, sc.Content, rerr)
		}
	}
}

// TestConvergeGenRegistrationAndNoWriteAfterCancel ②收敛闭环纳入同一纪律：
// 运行期间进登记表（同章互斥生效、CancelCreateChapter 有句柄可取消），协程计 WG；
// 取消后不再续写旧 project.Manager（目标章字节不变）。
func TestConvergeGenRegistrationAndNoWriteAfterCancel(t *testing.T) {
	gate := make(chan struct{})
	gateReleaser(t, gate) // 用例 Fatal 退出时兜底放行，避免 srv.Close 挂死
	arrived := make(chan struct{}, 1)

	a, pm := newGatedLLMApp(t, gate, arrived, "他推门进来，把伞收在墙角。雨还没停。")
	original := strings.Repeat("他感到无比的震惊。此外，值得注意的是，这一切仿佛命运的安排。", 8)
	if err := pm.WriteChapter(1, original); err != nil {
		t.Fatal(err)
	}
	beforeBytes, _ := readChapterBytes(pm, 1)

	if _, err := a.NovelChapterConverge(1, 3, 1); err != nil {
		t.Fatalf("收敛启动应放行: %v", err)
	}
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		t.Fatal("5s 内未收到收敛修补请求")
	}

	// 登记（P0#4 红点：未登记 → 同章互斥与取消句柄都不存在）
	a.chapterGenMu.Lock()
	_, registered := a.chapterGenCancels[chapterGenKey(1, "")]
	a.chapterGenMu.Unlock()
	if !registered {
		t.Fatalf("收敛闭环未进取消登记表：同章互斥/取消对收敛链失效")
	}
	// 同章再生成必须被拒绝（沿用既有互斥机制，不自造）
	if _, err := a.NovelChapterConverge(1, 3, 1); err == nil {
		t.Fatalf("收敛进行中，同章第二次生成应被拒绝")
	} else if !strings.Contains(err.Error(), "正在") {
		t.Fatalf("拒绝错误应指明正在生成: %v", err)
	}
	// 取消句柄在册：CancelCreateChapter（切书/取消同款入口）必须命中收敛链
	if !a.CancelCreateChapter(1, "") {
		t.Fatalf("CancelCreateChapter 未命中收敛链（收敛链无取消登记）")
	}

	// WG 记账 + 取消后不再续写
	waitGensDone(t, a)
	afterBytes, _ := readChapterBytes(pm, 1)
	if string(afterBytes) != string(beforeBytes) {
		t.Fatalf("取消后不应再写旧 project.Manager（before=%dB after=%dB）", len(beforeBytes), len(afterBytes))
	}

	// 清理兜底：登记表必须已清空（取消 ≠ 提前摘表）
	a.chapterGenMu.Lock()
	left := len(a.chapterGenCancels)
	a.chapterGenMu.Unlock()
	if left != 0 {
		t.Fatalf("取消后登记表应清空，仍有 %d 项", left)
	}
}
