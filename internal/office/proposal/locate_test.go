package proposal

import "testing"

func TestLocateQuote_Exact(t *testing.T) {
	md := "第一章 项目概况\n本项目位于华东地区。"
	start, end, ok := LocateQuote(md, "本项目位于华东地区")
	if !ok {
		t.Fatal("exact 未命中")
	}
	if md[start:end] != "本项目位于华东地区" {
		t.Errorf("偏移错误: %d-%d => %q", start, end, md[start:end])
	}
}

func TestLocateQuote_WhitespaceFlexible(t *testing.T) {
	md := "施工方案（20 分）\n要求：\n- 完整\n- 合理"
	start, end, ok := LocateQuote(md, "施工方案（20分）")
	if !ok {
		t.Fatal("归一化未命中")
	}
	if got := md[start:end]; got != "施工方案（20 分）" {
		t.Errorf("偏移错误: %d-%d => %q", start, end, got)
	}
}

func TestLocateQuote_NotFound(t *testing.T) {
	if _, _, ok := LocateQuote("完全无关的内容", "废标条款"); ok {
		t.Error("不应命中")
	}
}

func TestLocatePage(t *testing.T) {
	pages := []PageText{
		{Page: 1, Text: "第一章 项目概况"},
		{Page: 2, Text: "评分标准：施工方案 20 分"},
		{Page: 3, Text: "废标条款"},
	}
	if got := LocatePage(pages, "施工方案"); got != 2 {
		t.Errorf("LocatePage = %d, want 2", got)
	}
	if got := LocatePage(pages, "不存在的词"); got != 0 {
		t.Errorf("LocatePage 未命中应返回 0，实际 %d", got)
	}
}
