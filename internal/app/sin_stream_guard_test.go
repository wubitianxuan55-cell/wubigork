package app

// 原罪流并发与收尾纪律测试（v4.422 批）：
//   - 同话题并发 SinStream 互斥（register 命中在途即拒绝，不再覆盖登记）；
//   - Delete/Clear 在途流中如实拒绝（防「复活」/FK 报错）；
//   - 零正文取消只落用户消息（不落空 assistant 行）；
//   - 中途流错误保留已流出正文（与取消同口径，不再整段丢弃）；
//   - 底稿块便签用原始下标（与工具 set/delete 的 index 参数对齐）；
//   - 附件截断按 rune 边界回退（不劈半个多字节字符）；
//   - SinNotesSave 丢弃空白便签。

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/modelengine"
)

// retargetSinEngine 把 sin 功能绑定的 herdsman 引擎指到给定 URL（装配复用
// newSinSlowStreamApp，仅换假 LLM 的落点）。
func retargetSinEngine(t *testing.T, a *App, url string) {
	t.Helper()
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Enabled: true, BaseURL: url,
		Models: []modelengine.ModelInfo{{ID: "qwen3-8b"}},
	}); err != nil {
		t.Fatalf("SaveEngine(%s): %v", url, err)
	}
}

// newSinHangingStreamApp 构造一个「收到请求后挂住直到客户端中止」的假 LLM：
// 全程不吐内容，供互斥/删除守卫/零正文取消测试稳定复现「在途流」。
func newSinHangingStreamApp(t *testing.T) *App {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(": ping\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done() // 客户端取消前一直挂着
	}))
	t.Cleanup(srv.Close)
	a := newSinSlowStreamApp(t)
	retargetSinEngine(t, a, srv.URL)
	return a
}

// newSinAbortAfterFirstChunkApp 构造「吐一段后连接中断」的假 LLM：
// 模拟中途流错误（读错误 → chunk.Error），验证已流出正文保留落库。
func newSinAbortAfterFirstChunkApp(t *testing.T) *App {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"第一段。\"}}]}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		panic(http.ErrAbortHandler) // 连接中断 → 客户端读错误
	}))
	t.Cleanup(srv.Close)
	a := newSinSlowStreamApp(t)
	retargetSinEngine(t, a, srv.URL)
	return a
}

func waitSinRunGone(t *testing.T, topicID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if takeSinRun(topicID) == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("等待在途流收尾超时")
}

func TestSinStreamRejectsConcurrentRun(t *testing.T) {
	a := newSinHangingStreamApp(t)
	story, err := a.SinTopicCreate("双发")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "第一轮"); err != nil {
		t.Fatalf("首轮 SinStream: %v", err)
	}
	// 同话题第二发：register 命中在途即拒绝（历史实现静默覆盖登记 → 双写者）
	if _, err := a.SinStream(story.ID, "第二轮"); err == nil {
		t.Fatal("并发 SinStream 应被拒绝")
	} else if !strings.Contains(err.Error(), "正在生成中") {
		t.Fatalf("拒绝语义应如实说明在途: %v", err)
	}
	// 其他话题不受影响（互斥按话题隔离）
	other, err := a.SinTopicCreate("另一篇")
	if err != nil {
		t.Fatalf("SinTopicCreate(other): %v", err)
	}
	if _, err := a.SinStream(other.ID, "并行写"); err != nil {
		t.Fatalf("其他话题应可并行: %v", err)
	}
	if err := a.SinCancel(other.ID); err != nil {
		t.Fatalf("cancel other: %v", err)
	}
	// 取消并等收尾后，同话题可再次发起
	if err := a.SinCancel(story.ID); err != nil {
		t.Fatalf("SinCancel: %v", err)
	}
	waitSinRunGone(t, story.ID)
	if _, err := a.SinStream(story.ID, "再来一轮"); err != nil {
		t.Fatalf("收尾后应可再次发起: %v", err)
	}
	if err := a.SinCancel(story.ID); err != nil {
		t.Fatalf("cleanup cancel: %v", err)
	}
	waitSinRunGone(t, story.ID)
}

func TestSinTopicDeleteClearRejectWhileRunning(t *testing.T) {
	a := newSinHangingStreamApp(t)
	story, err := a.SinTopicCreate("在途删除")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	if err := a.SinTopicDelete(story.ID); err == nil || !strings.Contains(err.Error(), "正在生成中") {
		t.Fatalf("在途中删除应如实拒绝, got %v", err)
	}
	if err := a.SinTopicClear(story.ID); err == nil || !strings.Contains(err.Error(), "正在生成中") {
		t.Fatalf("在途中清空应如实拒绝, got %v", err)
	}
	if err := a.SinCancel(story.ID); err != nil {
		t.Fatalf("SinCancel: %v", err)
	}
	waitSinRunGone(t, story.ID)
	if err := a.SinTopicDelete(story.ID); err != nil {
		t.Fatalf("收尾后删除应放行: %v", err)
	}
}

func TestSinCancelWithZeroContentKeepsUserOnly(t *testing.T) {
	a := newSinHangingStreamApp(t)
	story, err := a.SinTopicCreate("零正文取消")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	if err := a.SinCancel(story.ID); err != nil {
		t.Fatalf("SinCancel: %v", err)
	}
	waitSinRunGone(t, story.ID)
	msgs, err := a.SinMessages(story.ID)
	if err != nil {
		t.Fatalf("SinMessages: %v", err)
	}
	// 零正文取消：只落用户消息，不落空 assistant 行（历史/导出/轨迹不被空行污染）
	if len(msgs) != 1 {
		t.Fatalf("应只落用户消息 1 条, got %d: %+v", len(msgs), msgs)
	}
	if msgs[0].Role != "user" || msgs[0].Content != "写开场" {
		t.Fatalf("保留的应是用户消息: %+v", msgs[0])
	}
}

func TestSinStreamErrorKeepsPartial(t *testing.T) {
	a := newSinAbortAfterFirstChunkApp(t)
	story, err := a.SinTopicCreate("中断保留")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	msgs := waitSinMessages(t, a, story.ID, 2, 3*time.Second)
	assistant := msgs[1]
	if !strings.Contains(assistant.Content, "第一段。") {
		t.Errorf("已流出的正文应保留落库: %q", assistant.Content)
	}
	if strings.Contains(assistant.Extra, `"cancelled"`) {
		t.Errorf("错误路径不是取消，不应标 cancelled: %q", assistant.Extra)
	}
}

// TestSinDraftBlockKeepsOriginalNoteIndices 底稿块便签序号 = doc.Notes 原始下标
// （工具 set/delete 的 index 参数口径）：空条目跳过显示但不错位重编号。
func TestSinDraftBlockKeepsOriginalNoteIndices(t *testing.T) {
	doc := sinNotesDoc{Notes: []string{"", "第一条：女主叫林晚", "第二条：顾城是线人", ""}}
	blk := sinDraftBlock(doc)
	if !strings.Contains(blk, "#1 第一条：女主叫林晚") {
		t.Errorf("应保留原始下标 #1: %q", blk)
	}
	if !strings.Contains(blk, "#2 第二条：顾城是线人") {
		t.Errorf("应保留原始下标 #2: %q", blk)
	}
	if !strings.Contains(blk, "共 4 条") {
		t.Errorf("总条数按 doc.Notes 口径（含空条目）: %q", blk)
	}
	if strings.Contains(blk, "#0") || strings.Contains(blk, "#3") {
		t.Errorf("空条目不应占用展示行: %q", blk)
	}
}

// TestSinReadFileRefTruncatesAtRuneBoundary 64KB 截断按 rune 边界回退：
// 切点落在多字节字符中间时整字回退，不产出半个字符。
func TestSinReadFileRefTruncatesAtRuneBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.txt")
	// "ab" 前缀让 64KB 切点正好落在「好」的续字节上（字节布局已验算）
	content := "ab" + strings.Repeat("好", 21845)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := sinReadFileRef(path)
	if err != nil {
		t.Fatalf("sinReadFileRef: %v", err)
	}
	if !strings.Contains(got, "[已截断") {
		t.Fatalf("超限文件应带截断标记: len=%d", len(got))
	}
	head := strings.SplitN(got, "\n\n", 2)[0]
	if strings.ContainsRune(head, '\uFFFD') {
		t.Errorf("截断不应劈出半个字符（U+FFFD）: tail=%q", head[len(head)-12:])
	}
	if !strings.HasSuffix(head, "好") {
		t.Errorf("切点应整字回退到「好」: tail=%q", head[len(head)-12:])
	}
}

// TestSinNotesSaveDropsBlankNotes 面板保存丢弃空白便签：doc.Notes 不留空串
// （工具 index 与底稿块下标保持对齐）。
func TestSinNotesSaveDropsBlankNotes(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	story, err := a.SinTopicCreate("空行清理")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	baseJSON := `{"notes":[],"outline":""}`
	v, err := a.SinNotesSave(story.ID, baseJSON, "", `["女主叫林晚", "   ", ""]`, false)
	if err != nil {
		t.Fatalf("SinNotesSave: %v", err)
	}
	if len(v.Notes) != 1 || v.Notes[0] != "女主叫林晚" {
		t.Fatalf("空白便签应被丢弃: %+v", v.Notes)
	}
	path := mustSinNotesPath(t, story.ID)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(raw), `" "`) || strings.Contains(string(raw), `"",`) {
		t.Errorf("落盘文件不应含空白条目: %s", raw)
	}
}
