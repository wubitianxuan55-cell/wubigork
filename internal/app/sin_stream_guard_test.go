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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// ── v4.426 审计落地锚 ──

// TestSinStreamPlainErrorSavesUserMessage 首轮普通失败（错误与工具无关，如
// 断网/5xx）：不降级重试（历史实现会谎报「模型不支持工具」并静默关工具）、
// error 帧如实上报，用户指令照常落库（镜像零正文取消口径，刷新不丢话）。
func TestSinStreamPlainErrorSavesUserMessage(t *testing.T) {
	var mu sync.Mutex
	reqs := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		mu.Unlock()
		// 400（不可故障转移，与降级测试同通道）+ 与工具无关的错误文本：
		// 历史实现会把它误判成「模型不支持工具」并静默关工具重试。
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream connect error"}}`))
	})
	sinTestHome(t)
	a := newChatServiceTestAppWithHandler(t, handler)
	if err := a.core.SetFeatureModel("sin", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(sin): %v", err)
	}
	story, err := a.SinTopicCreate("普通失败")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	waitSinRunGone(t, story.ID)
	mu.Lock()
	got := reqs
	mu.Unlock()
	if got != 1 {
		t.Errorf("普通失败不应触发去工具重试，请求数 = %d, want 1", got)
	}
	msgs, err := a.SinMessages(story.ID)
	if err != nil {
		t.Fatalf("SinMessages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Role != "user" || msgs[0].Content != "写开场" {
		t.Fatalf("失败轮应只落用户指令: %+v", msgs)
	}
}

// TestSinAttachIllustrationRejectsForeignTopic 插图回写归属校验：messageID
// 不属于目标话题时 fail-closed 拒绝（前端状态错位不能把图片写进别的故事）。
func TestSinAttachIllustrationRejectsForeignTopic(t *testing.T) {
	sinTestHome(t)
	a := newSinTestApp(t)
	tA, err := a.SinTopicCreate("甲")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	tB, err := a.SinTopicCreate("乙")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if err := a.chatStore.AppendExchange(tA.ID, "画一张", "好的。", ""); err != nil {
		t.Fatalf("AppendExchange: %v", err)
	}
	msgs, err := a.SinMessages(tA.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("SinMessages = %+v, err=%v", msgs, err)
	}
	if err := a.sinAttachIllustration(tB.ID, msgs[1].ID, "0", "C:/art/x.png"); err == nil {
		t.Fatal("跨话题回写应被拒绝")
	}
	if err := a.sinAttachIllustration(tA.ID, msgs[1].ID, "0", "C:/art/x.png"); err != nil {
		t.Fatalf("本话题回写不应报错: %v", err)
	}
}

// TestSinFileRefQuotedToken 含空格路径走 @"..." 引号 token（Composer 对含空格
// 路径注入引号形式）：内容照常注入；裸 token 在空格处截断后不是引用，原样保留。
func TestSinFileRefQuotedToken(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "space dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "设定集.txt")
	if err := os.WriteFile(p, []byte("雨夜设定内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	block, errs := sinFileRefBlock(context.Background(), `先看 @"`+p+`" 再写`)
	if len(errs) != 0 {
		t.Fatalf("引号 token 不应报错: %v", errs)
	}
	if !strings.Contains(block, "雨夜设定内容") {
		t.Fatalf("含空格路径应照常注入: %q", block)
	}
	// 裸 token（历史形态）在空格处截断后 stat 不中 → 不是引用，零误伤
	block2, errs2 := sinFileRefBlock(context.Background(), `先看 @`+p+` 再写`)
	if len(errs2) != 0 || strings.Contains(block2, "雨夜设定内容") {
		t.Fatalf("裸 token 空格截断应按普通文本保留: block=%q errs=%v", block2, errs2)
	}
}

// TestSinReadFileRefEmptyFile 空文件 = 空内容（io.EOF 不是读取失败）。
func TestSinReadFileRefEmptyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := sinReadFileRef(p)
	if err != nil {
		t.Fatalf("空文件不应报错: %v", err)
	}
	if got != "" {
		t.Fatalf("空文件内容应为空: %q", got)
	}
}

// TestSinExportFileNameReservedDeviceName Windows 保留设备名避让：stem 命中
// 保留名表加「_」前缀（CON.md 打到 DOS 设备而非磁盘）。
func TestSinExportFileNameReservedDeviceName(t *testing.T) {
	now := time.Now()
	for raw, want := range map[string]string{
		"CON":    "_CON",
		"aux.md": "_aux",
		// stem 含空格不再是保留名（Windows 设备名只认纯 stem）
		"con 记事": "con 记事",
		"普通标题":   "普通标题",
	} {
		got, err := sinExportFileName(raw, "sin_1_1", now)
		if err != nil {
			t.Fatalf("sinExportFileName(%q): %v", raw, err)
		}
		if got != want {
			t.Errorf("sinExportFileName(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestSinToolArtifactsInMarkdownExport 工具产物进导出：sin_illustrate 的图不在
// 正文标记里，Markdown 导出按调用序附录（caption 作替代文本），不再整批消失。
func TestSinToolArtifactsInMarkdownExport(t *testing.T) {
	sinTestHome(t)
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("工具图")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	extra := `{"tools":[{"id":"c1","name":"sin_illustrate","artifacts":[{"kind":"image","path":"C:/art/x.png","caption":"雨夜站台"}]}]}`
	if err := a.chatStore.AppendExchange(topic.ID, "画一张", "好的。", extra); err != nil {
		t.Fatalf("AppendExchange: %v", err)
	}
	md, err := a.SinExportMarkdown(topic.ID)
	if err != nil {
		t.Fatalf("SinExportMarkdown: %v", err)
	}
	if !strings.Contains(md, "![雨夜站台](C:/art/x.png)") {
		t.Fatalf("工具产物应进导出附录: %s", md)
	}
}

// TestSinExportEpubSkipsLegacyEmptyRows 历史遗留零正文行：不占回数、不消耗
// 指令引块；连续多条用户指令逐条入引块（与 Markdown 导出同口径）。
func TestSinExportEpubSkipsLegacyEmptyRows(t *testing.T) {
	sinTestHome(t)
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("历史行")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if err := a.chatStore.AppendExchange(topic.ID, "指令一", "第一回正文。", ""); err != nil {
		t.Fatalf("AppendExchange: %v", err)
	}
	// 失败轮遗留：用户指令 + 零正文 assistant 行（v4.423 前的历史数据形态）
	if err := a.chatStore.AppendExchange(topic.ID, "指令二", "", ""); err != nil {
		t.Fatalf("AppendExchange(空行): %v", err)
	}
	if err := a.chatStore.AppendExchange(topic.ID, "指令三", "第二回正文。", ""); err != nil {
		t.Fatalf("AppendExchange(2): %v", err)
	}
	path, err := a.SinExportEpub(topic.ID)
	if err != nil {
		t.Fatalf("SinExportEpub: %v", err)
	}
	defer func() { _ = os.Remove(path) }()
	text, _ := readSinEpubText(t, path)
	if !strings.Contains(text, "第一回正文") || !strings.Contains(text, "第二回正文") {
		t.Fatalf("两回正文都应入书: %s", text)
	}
	// 指令二（失败轮的）与指令三都应挂在第二回引块里，不再被零正文行吃掉
	if !strings.Contains(text, "我：指令二") || !strings.Contains(text, "我：指令三") {
		t.Fatalf("连续用户指令应逐条入引块: %s", text)
	}
	if strings.Contains(text, "第三回") {
		t.Fatalf("零正文行不应占回数: %s", text)
	}
}

// TestSinFileRefVisionBoundedConcurrency 图片附件识图按 sinVisionConcurrency
// 有界并发（v4.427：串行时 N 张图 = N×90s 首帧前干等），且块序保持出现序。
func TestSinFileRefVisionBoundedConcurrency(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i := 0; i < 3; i++ {
		p := filepath.Join(dir, fmt.Sprintf("img%d.png", i))
		if err := os.WriteFile(p, []byte{0x89, 'P'}, 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	var (
		mu       sync.Mutex
		cur, max int
	)
	orig := sinVisionRecognize
	sinVisionRecognize = func(ctx context.Context, path, hint string) (string, error) {
		mu.Lock()
		cur++
		if cur > max {
			max = cur
		}
		mu.Unlock()
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		cur--
		mu.Unlock()
		return "图" + filepath.Base(path), nil
	}
	t.Cleanup(func() { sinVisionRecognize = orig })

	line := "看这三张 @" + paths[0] + " @" + paths[1] + " @" + paths[2]
	block, errs := sinFileRefBlock(context.Background(), line)
	if len(errs) != 0 {
		t.Fatalf("识图不应报错: %v", errs)
	}
	if max > sinVisionConcurrency {
		t.Fatalf("识图并发 %d 超上限 %d", max, sinVisionConcurrency)
	}
	if max < 2 {
		t.Fatalf("并发未生效（max=%d，应≥2——否则还是串行）", max)
	}
	// 块序 = token 出现序（并发不乱序）
	i0 := strings.Index(block, "图img0.png")
	i1 := strings.Index(block, "图img1.png")
	i2 := strings.Index(block, "图img2.png")
	if i0 < 0 || i1 < 0 || i2 < 0 || !(i0 < i1 && i1 < i2) {
		t.Fatalf("识图块应按出现序拼装: i0=%d i1=%d i2=%d", i0, i1, i2)
	}
}

// TestSinContextNodeDetailRejectsForeignTopic 节点详情按 seq 反解消息 id 后
// 校验归属（v4.427 单条查询路径）：他话题的消息 seq 与不存在 seq 同样报错，
// 与旧全量扫描行为一致。
func TestSinContextNodeDetailRejectsForeignTopic(t *testing.T) {
	sinTestHome(t)
	a := newSinTestApp(t)
	tA, err := a.SinTopicCreate("甲")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	tB, err := a.SinTopicCreate("乙")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if err := a.chatStore.AppendExchange(tA.ID, "问", "答。", ""); err != nil {
		t.Fatalf("AppendExchange: %v", err)
	}
	msgs, err := a.SinMessages(tA.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("SinMessages = %+v err=%v", msgs, err)
	}
	// 甲的消息 seq：乙的看板拿去展开 → 拒绝（不能跨话题读正文）
	if _, err := a.SinContextNodeDetail(tB.ID, sinNodeSeq(msgs[0].ID, 0)); err == nil {
		t.Fatal("跨话题节点详情应被拒绝")
	}
	if _, err := a.SinContextNodeDetail(tA.ID, sinNodeSeq(msgs[0].ID, 0)); err != nil {
		t.Fatalf("本话题 user 节点应可展开: %v", err)
	}
}
