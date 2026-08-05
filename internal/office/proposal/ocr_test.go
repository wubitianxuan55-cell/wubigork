package proposal

import (
	"context"
	"testing"
)

// fakeOCR 测试用 OCR 提供者
type fakeOCR struct {
	called bool
	text   string
	err    error
}

func (f *fakeOCR) OCR(ctx context.Context, filePath string) (string, error) {
	f.called = true
	return f.text, f.err
}

func TestConvertFiles_UsesOCRForScannedPDF(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	ocr := &fakeOCR{text: "扫描件识别出的招标内容"}
	s.SetOCRProviderForTest(ocr)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{{
		Name: "scan.pdf", Path: "testdata/empty.pdf",
	}}}
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	got, err := s.ConvertFiles(context.Background(), p.ID, nil)
	if err != nil {
		t.Fatalf("ConvertFiles: %v", err)
	}
	if !ocr.called {
		t.Fatal("OCR 未被调用")
	}
	f := got.BidSummary.RawFiles[0]
	if f.Markdown == "" || f.OCRStatus != "ocr" || f.Error != "" {
		t.Fatalf("OCR 转换结果异常: %+v", f)
	}
	if !containsAny(f.Markdown, "扫描件识别出的招标内容") {
		t.Fatalf("OCR 文本未写入: %q", f.Markdown)
	}
}

func TestConvertFiles_OCRUnavailableReportsError(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{{
		Name: "scan.pdf", Path: "testdata/empty.pdf",
	}}}
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	got, err := s.ConvertFiles(context.Background(), p.ID, nil)
	if err != nil {
		t.Fatalf("ConvertFiles: %v", err)
	}
	f := got.BidSummary.RawFiles[0]
	if f.Error == "" || !containsAny(f.Error, "扫描件", "OCR") {
		t.Fatalf("缺少可读的失败原因: %+v", f)
	}
}

func TestParseBidFile_AllFilesFailedClearError(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{
		{Name: "a.pdf", Error: "PDF 无可提取文本（可能是扫描件）"},
		{Name: "b.pdf", Error: "文件损坏"},
	}}
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	_, err := s.ParseBidFile(context.Background(), p.ID)
	if err == nil {
		t.Fatal("应返回明确错误")
	}
	if !containsAny(err.Error(), "a.pdf", "b.pdf", "未成功转换") {
		t.Fatalf("错误未透出文件名与原因: %v", err)
	}
}

func TestParseBidFileWithProgress_EmitsStages(t *testing.T) {
	ai := &mockAI{replies: map[string]string{
		"请解析以下招标文件": `{
  "overview": "项目概况", "overviewQuote": "项目概况",
  "techScoring": [{"name":"施工方案","maxScore":"20","requirement":"完整","quote":"施工方案"}]
}`,
	}}
	s := newServiceAt(t, t.TempDir(), ai)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{{
		Name: "t.txt", Markdown: "项目概况\n施工方案 20 分",
	}}}
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	var stages []string
	_, err := s.ParseBidFileWithProgress(context.Background(), p.ID, func(stage, detail string) {
		stages = append(stages, stage)
	})
	if err != nil {
		t.Fatalf("ParseBidFileWithProgress: %v", err)
	}
	if len(stages) == 0 {
		t.Fatal("未产生进度事件")
	}
}
