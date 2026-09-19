package app

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sinTestPNG 最小 1x1 PNG（EPUB 内嵌用；go-epub 只搬运字节不解码）。
var sinTestPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

// seedSinEpubStory 造一个两回合故事：第一回合带已生成插图（真 PNG 落盘）+
// 未生成标记；第二回合带 webp 附件路径（go-epub 不支持，应保留占位）。
func seedSinEpubStory(t *testing.T) (*App, string) {
	t.Helper()
	t.Setenv("USERPROFILE", t.TempDir())
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("雨夜站台")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if err := a.chatStore.AppendExchange(topic.ID,
		"写一个雨夜开头",
		"夜雨落在站台上。\n"+sinIllustrationCueOpen+"站台灯下的女人"+sinIllustrationCueClose+"\n她没有回头。",
		""); err != nil {
		t.Fatalf("AppendExchange: %v", err)
	}
	msgs, err := a.SinMessages(topic.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("SinMessages = %+v, err=%v", msgs, err)
	}
	artPath := filepath.Join(sinArtDir(), "art-1.png")
	if err := os.MkdirAll(sinArtDir(), 0o755); err != nil {
		t.Fatalf("MkdirAll(art): %v", err)
	}
	if err := os.WriteFile(artPath, sinTestPNG, 0o644); err != nil {
		t.Fatalf("WriteFile(art): %v", err)
	}
	if err := a.sinAttachIllustration(msgs[1].ID, "0", artPath); err != nil {
		t.Fatalf("sinAttachIllustration: %v", err)
	}
	if err := a.chatStore.AppendExchange(topic.ID,
		"继续，给一个特写",
		"近景：她的睫毛上挂着水。\n"+sinIllustrationCueOpen+"睫毛特写"+sinIllustrationCueClose,
		""); err != nil {
		t.Fatalf("AppendExchange(2): %v", err)
	}
	return a, topic.ID
}

// readSinEpubText 读出 EPUB（zip）内全部 xhtml 文本与条目名。
func readSinEpubText(t *testing.T, path string) (string, []string) {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("打开 EPUB（zip）失败: %v", err)
	}
	defer zr.Close()
	var names []string
	var text bytes.Buffer
	for _, f := range zr.File {
		names = append(names, f.Name)
		if !strings.HasSuffix(f.Name, ".xhtml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("打开 %s: %v", f.Name, err)
		}
		_, _ = io.Copy(&text, rc)
		_ = rc.Close()
		text.WriteString("\n")
	}
	return text.String(), names
}

// TestSinExportEpubFullChain EPUB 导出主链：文件落 exports、zip 结构成立、
// 用户指令引块在位、已生成插图内嵌（图片条目 + img 标签 + 封面）、
// 未生成/不支持的标记保留画面描述占位、正文与标记零残留。
func TestSinExportEpubFullChain(t *testing.T) {
	a, topicID := seedSinEpubStory(t)

	path, err := a.SinExportEpub(topicID)
	if err != nil {
		t.Fatalf("SinExportEpub: %v", err)
	}
	if filepath.Dir(path) != sinExportsDir() {
		t.Errorf("产物应落导出目录: %s", path)
	}
	if !strings.Contains(filepath.Base(path), "雨夜站台") {
		t.Errorf("文件名应取自故事标题: %s", path)
	}
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatalf("产物文件异常: %v, %v", info, err)
	}

	text, names := readSinEpubText(t, path)
	for _, want := range []string{
		"夜雨落在站台上。",           // 正文
		"<blockquote><p>我：写一个雨夜开头", // 用户指令引块
		`<img src="`,                     // 插图内嵌
		"（插图未生成：睫毛特写）",        // 未生成占位（cue 次序键无映射）
		"她没有回头。",              // 标记后的正文不丢
	} {
		if !strings.Contains(text, want) {
			t.Errorf("EPUB 缺少 %q", want)
		}
	}
	if strings.Contains(text, sinIllustrationCueOpen) {
		t.Errorf("EPUB 不应残留插图标记")
	}
	var imgs int
	for _, n := range names {
		if strings.HasPrefix(filepath.Base(n), "illu-") {
			imgs++
		}
	}
	if imgs != 1 {
		t.Errorf("应恰好内嵌 1 张插图，got %d（entries=%v）", imgs, names)
	}
	hasCover := false
	for _, n := range names {
		if strings.HasSuffix(n, "cover.xhtml") {
			hasCover = true
		}
	}
	if !hasCover {
		t.Errorf("首张内嵌插图应设为封面（缺 cover.xhtml）: %v", names)
	}

	// 同名不覆盖：第二次导出落 -2 新文件，原产物不动。
	path2, err := a.SinExportEpub(topicID)
	if err != nil {
		t.Fatalf("SinExportEpub(2): %v", err)
	}
	if path2 == path {
		t.Fatalf("重复导出不应覆盖: %s", path2)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("首次产物不应被顶掉: %v", err)
	}
}

// TestSinExportEpubGuards 空故事拒绝导出；非 sin 话题拒绝（守卫复用）。
func TestSinExportEpubGuards(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinExportEpub(topic.ID); err == nil {
		t.Error("空故事应拒绝导出")
	}
	chatTopic, err := a.ChatTopicCreate("闲聊", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if _, err := a.SinExportEpub(chatTopic.ID); err == nil {
		t.Error("聊天话题应被守卫拒绝")
	}
}
