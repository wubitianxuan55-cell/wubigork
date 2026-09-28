package app

// 线0 · N1（P0）取消生成覆盖已完成章节 + N2（P1）取消提前摘互斥。
//
// N1：取消落盘旧实现无条件写 chapters/NNN.md——对已写完的章再点生成并中途
// 取消，整章正文被半截替换且无备份。修复后：目标章在本次生成开始前已有正文
// （或取消时复查已有正文，含 v4 场景承载正文）时残稿另存 NNN.partial-<ts>.md，
// 正稿字节原样保留，事件补 partialSaved/partialPath/notice。
//
// N2：取消只调 cancel，登记表条目置 nil 占位，清理由 unregisterChapterGen
// 独占；「已请求取消、协程未退出」窗口内 register 如实拒绝。

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/httpbridge"
)

// subscribeCreateChapterStream 经 httpbridge 全局 hub 订阅 create-chapter-stream
// SSE 面，返回事件快照函数（订阅 handler 与 hub 是包级单例，app.emit→Publish
// 即达）。消费 connected 帧确保订阅登记后才放行。
func subscribeCreateChapterStream(t *testing.T, channel string) func() []map[string]interface{} {
	t.Helper()
	srv := httptest.NewServer(httpbridge.New(nil).Handler())
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/stream?id="+channel, nil)
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		srv.Close()
		t.Fatalf("打开 %s SSE: %v", channel, err)
	}
	t.Cleanup(func() { cancel(); resp.Body.Close(); srv.Close() })

	br := bufio.NewReader(resp.Body)
	for i := 0; i < 3; i++ { // 消费 connected 帧 = 订阅已登记
		if _, err := br.ReadString('\n'); err != nil {
			t.Fatalf("读 connected 帧: %v", err)
		}
	}

	var mu sync.Mutex
	var events []map[string]interface{}
	go func() {
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var m map[string]interface{}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &m) == nil {
				mu.Lock()
				events = append(events, m)
				mu.Unlock()
			}
		}
	}()
	return func() []map[string]interface{} {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]interface{}(nil), events...)
	}
}

// waitCancelledEvent 轮询等待 cancelled 事件（订阅在 Publish 之前）。
func waitCancelledEvent(t *testing.T, snap func() []map[string]interface{}) map[string]interface{} {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, ev := range snap() {
			if ev["type"] == "cancelled" {
				return ev
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("未收到 cancelled 事件")
	return nil
}

// partialSidecars 列出某章目录下的残稿侧车文件（NNN.partial-*.md）。
func partialSidecars(t *testing.T, pmDir string, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(pmDir, "chapters"))
	if err != nil {
		t.Fatalf("读 chapters 目录: %v", err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix+chapterPartialFileSuffix) && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e.Name())
		}
	}
	return out
}

// TestSaveCancelledPartial_ThreeStates 取消落盘三态（N1 必补单测）：
//  1. 目标不存在 → 保持现行为，残稿写入正稿；
//  2. 目标已存在（生成开始时即存在）→ 正稿字节不变 + 残稿文件存在 + 事件如实提示；
//  3. v4 场景存在 → 场景与 blob 都不被覆盖，残稿另存；
//  4. 生成开始时不存在但取消时已有正稿（其他写者补上）→ 仍不覆盖（复查兜底）。
func TestSaveCancelledPartial_ThreeStates(t *testing.T) {
	t.Run("目标不存在_写正稿", func(t *testing.T) {
		snap := subscribeCreateChapterStream(t, "create-chapter-stream")
		a := newGateEmptyApp()
		pm := newGateProject(t)
		a.setPM(pm)

		a.saveCancelledPartial(pm, "新章残稿片段。", "", 0, false, 1, "n1", "", false)

		got, err := pm.ReadChapter(1)
		if err != nil {
			t.Fatalf("读正稿: %v", err)
		}
		if got != "新章残稿片段。" {
			t.Fatalf("目标章不存在时残稿应写入正稿，得到 %q", got)
		}
		if n := len(partialSidecars(t, pm.Dir, "001")); n != 0 {
			t.Fatalf("目标章不存在时不应产生残稿侧车，得到 %d 个", n)
		}
		ev := waitCancelledEvent(t, snap)
		if ev["content"] != "新章残稿片段。" {
			t.Fatalf("事件应带 content（现行为保持），得到 %v", ev["content"])
		}
		if _, ok := ev["partialSaved"]; ok {
			t.Fatalf("未另存时不应有 partialSaved，得到 %v", ev["partialSaved"])
		}
	})

	t.Run("目标已存在_正稿不变_残稿另存", func(t *testing.T) {
		snap := subscribeCreateChapterStream(t, "create-chapter-stream")
		a := newGateEmptyApp()
		pm := newGateProject(t)
		a.setPM(pm)

		const original = "已完成的正稿全文，绝不能被半截残稿覆盖。"
		if err := pm.WriteChapter(1, original); err != nil {
			t.Fatalf("预置正稿: %v", err)
		}
		before, err := os.ReadFile(pm.ChapterPath(1))
		if err != nil {
			t.Fatalf("读正稿字节: %v", err)
		}

		a.saveCancelledPartial(pm, "半截残稿。", "", 0, false, 1, "n1", "", true)

		after, err := os.ReadFile(pm.ChapterPath(1))
		if err != nil {
			t.Fatalf("读正稿字节: %v", err)
		}
		if string(after) != string(before) {
			t.Fatalf("正稿字节被改动：before=%q after=%q", before, after)
		}
		sidecars := partialSidecars(t, pm.Dir, "001")
		if len(sidecars) != 1 {
			t.Fatalf("应另存 1 个残稿侧车，得到 %v", sidecars)
		}
		sideData, err := os.ReadFile(filepath.Join(pm.Dir, "chapters", sidecars[0]))
		if err != nil {
			t.Fatalf("读残稿: %v", err)
		}
		if string(sideData) != "半截残稿。" {
			t.Fatalf("残稿内容不对: %q", sideData)
		}

		ev := waitCancelledEvent(t, snap)
		if ev["partialSaved"] != true {
			t.Fatalf("事件应带 partialSaved=true，得到 %v", ev["partialSaved"])
		}
		path, _ := ev["partialPath"].(string)
		if !strings.Contains(path, sidecars[0]) {
			t.Fatalf("partialPath 应指向残稿落点 %q，得到 %q", sidecars[0], path)
		}
		notice, _ := ev["notice"].(string)
		if !strings.Contains(notice, "正文已存在") || !strings.Contains(notice, sidecars[0]) {
			t.Fatalf("notice 应如实说明残稿另存落点，得到 %q", notice)
		}
	})

	t.Run("v4场景存在_不覆盖场景与blob", func(t *testing.T) {
		snap := subscribeCreateChapterStream(t, "create-chapter-stream")
		a := newGateEmptyApp()
		pm := newGateProject(t)
		if err := pm.MigrateV3ToV4(); err != nil {
			t.Fatalf("迁移 v4: %v", err)
		}
		a.setPM(pm)

		const blob = "旧章正文（blob）。"
		const sceneText = "旧场景正文，改写前的内容。"
		if err := pm.WriteChapter(1, blob); err != nil {
			t.Fatalf("写 blob: %v", err)
		}
		sm := pm.SceneManager(1)
		sc, err := sm.Create("opening", "开场")
		if err != nil {
			t.Fatalf("建场景: %v", err)
		}
		sc.Content = sceneText
		if err := sm.Write(sc); err != nil {
			t.Fatalf("写场景: %v", err)
		}

		a.saveCancelledPartial(pm, "半截残稿。", "", 0, false, 1, "n1", "", true)

		// 场景未被覆盖
		gotScene, err := sm.Read(sc.Meta.ID)
		if err != nil {
			t.Fatalf("读场景: %v", err)
		}
		if gotScene.Content != sceneText {
			t.Fatalf("v4 场景正文被残稿覆盖: %q", gotScene.Content)
		}
		// blob 未被覆盖（阅读页/检索/导出读的就是它）
		gotBlob, err := pm.ReadChapter(1)
		if err != nil {
			t.Fatalf("读 blob: %v", err)
		}
		if gotBlob != blob {
			t.Fatalf("v4 blob 被残稿覆盖: %q", gotBlob)
		}
		if n := len(partialSidecars(t, pm.Dir, "001")); n != 1 {
			t.Fatalf("v4 残稿应另存 1 个侧车，得到 %d", n)
		}
		ev := waitCancelledEvent(t, snap)
		if ev["partialSaved"] != true {
			t.Fatalf("v4 取消事件应带 partialSaved=true，得到 %v", ev["partialSaved"])
		}
	})

	t.Run("开始时不存在但取消时已有正稿_仍不覆盖", func(t *testing.T) {
		a := newGateEmptyApp()
		pm := newGateProject(t)
		a.setPM(pm)
		// 生成开始时无正文（existedBefore=false），但取消前已有其他写者落下正稿。
		if err := pm.WriteChapter(1, "其他写者落下的正稿。"); err != nil {
			t.Fatalf("预置正稿: %v", err)
		}

		a.saveCancelledPartial(pm, "半截残稿。", "", 0, false, 1, "n1", "", false)

		got, err := pm.ReadChapter(1)
		if err != nil {
			t.Fatalf("读正稿: %v", err)
		}
		if got != "其他写者落下的正稿。" {
			t.Fatalf("取消时已有正稿仍须保护，得到 %q", got)
		}
		if n := len(partialSidecars(t, pm.Dir, "001")); n != 1 {
			t.Fatalf("应另存 1 个残稿侧车，得到 %d", n)
		}
	})
}

// TestChapterBodyExists_ScenesOnlyBody v4「只有场景、无 blob」也算已有正文
// （否则取消会拿残稿去覆盖章节文件，进而在阅读页显出半截稿）。
func TestChapterBodyExists_ScenesOnlyBody(t *testing.T) {
	pm := newGateProject(t)
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("迁移 v4: %v", err)
	}
	if chapterBodyExists(pm, 2, "") {
		t.Fatal("空章不应算已有正文")
	}
	sm := pm.SceneManager(2)
	sc, err := sm.Create("opening", "开场")
	if err != nil {
		t.Fatalf("建场景: %v", err)
	}
	sc.Content = "场景承载的正文。"
	if err := sm.Write(sc); err != nil {
		t.Fatalf("写场景: %v", err)
	}
	if !chapterBodyExists(pm, 2, "") {
		t.Fatal("只有场景正文的 v4 章应算已有正文")
	}
}

// TestRegisterChapterGen_CancelPendingExclusive 取消语义（N2）：
// 取消只调 cancel 并把登记置 nil 占位，清理由 unregisterChapterGen 独占——
// 「已请求取消、协程未退出」窗口内同章再 register 必须被拒（不得并发出两个
// 写者），协程退出后可再 register；重复取消保持幂等 false。
func TestRegisterChapterGen_CancelPendingExclusive(t *testing.T) {
	a := newGateEmptyApp()

	if _, _, err := a.registerChapterGen(chapterGenKey(1, ""), 1, ""); err != nil {
		t.Fatalf("首次 register: %v", err)
	}
	if !a.CancelCreateChapter(1, "") {
		t.Fatalf("首次取消应返回 true")
	}
	if a.CancelCreateChapter(1, "") {
		t.Fatalf("重复取消应返回 false（幂等）")
	}

	// 协程尚未退出（登记值为 nil 占位）：同章再生成必须被拒，错误须说明取消中。
	_, _, err := a.registerChapterGen(chapterGenKey(1, ""), 1, "")
	if err == nil {
		t.Fatalf("取消未退出期间同章再 register 应被拒绝")
	}
	if !strings.Contains(err.Error(), "正在取消中") {
		t.Fatalf("拒绝错误应说明正在取消中, got: %v", err)
	}

	// 协程退出（unregisterChapterGen）后才放行。
	a.unregisterChapterGen(chapterGenKey(1, ""), func() {})
	if _, _, err := a.registerChapterGen(chapterGenKey(1, ""), 1, ""); err != nil {
		t.Fatalf("协程退出后应可再 register: %v", err)
	}
	// 清理：登记表清空，避免影响同包其他用例的 waitGensDone。
	a.chapterGenMu.Lock()
	delete(a.chapterGenCancels, chapterGenKey(1, ""))
	a.chapterGenMu.Unlock()
}
