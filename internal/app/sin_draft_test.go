package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/chat"
)

// TestBuildSinUserPromptEmptyDraftIsIdentical 空底稿 = 与无底稿行为逐字一致
//（底稿直注是增量，不是格式变更；老故事/没写过底稿的会话提示词零漂移）。
func TestBuildSinUserPromptEmptyDraftIsIdentical(t *testing.T) {
	history := []chat.Message{
		{Role: "user", Content: "写个开头"},
		{Role: "assistant", Content: "夜雨落在站台上。"},
	}
	withEmpty := buildSinUserPrompt(history, "继续", nil, sinNotesDoc{})
	withoutParam := buildSinUserPrompt(history, "继续", nil, sinNotesDoc{Version: sinNotesVersion})
	if withEmpty != withoutParam {
		t.Fatalf("空底稿两次装配应逐字一致:\n--- A ---\n%s\n--- B ---\n%s", withEmpty, withoutParam)
	}
	if strings.Contains(withEmpty, "故事底稿") {
		t.Errorf("空底稿不应出现底稿块:\n%s", withEmpty)
	}
}

// TestBuildSinUserPromptInjectsDraft 非空底稿直注：大纲与便签进提示、
// 排在本次指令之前、便签带序号。
func TestBuildSinUserPromptInjectsDraft(t *testing.T) {
	draft := sinNotesDoc{
		Version: sinNotesVersion,
		Notes:   []string{"女主叫林晚，右眉有疤", "故事发生在 1998 年的重庆"},
		Outline: "第一章 雨夜相遇；第二章 旧照片；第三章 天台对峙",
	}
	prompt := buildSinUserPrompt(nil, "继续写", nil, draft)
	for _, want := range []string{
		"【故事底稿",
		"第一章 雨夜相遇",
		"#0 女主叫林晚，右眉有疤",
		"#1 故事发生在 1998 年的重庆",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("前情缺底稿内容 %q:\n%s", want, prompt)
		}
	}
	if idx := strings.Index(prompt, "故事底稿"); idx > strings.Index(prompt, "【本次指令】") {
		t.Fatalf("底稿块应排在本次指令之前:\n%s", prompt)
	}
}

// TestSinDraftBlockBudget 便签合计预算：超预算从头截断、末条带省略号、
// 未收录条数如实报出（不静默丢）；带不下的部分引导用 sin_notes read 查看。
func TestSinDraftBlockBudget(t *testing.T) {
	long := strings.Repeat("设", sinPromptNotesBudgetRunes+1) // 单条超预算 → 截断收录
	doc := sinNotesDoc{
		Version: sinNotesVersion,
		Notes:   []string{long, "第二条进不来", "第三条也进不来"},
	}
	block := sinDraftBlock(doc)
	if !strings.Contains(block, "共 3 条") {
		t.Errorf("便签总条数应如实报 3:\n%s", block)
	}
	if !strings.Contains(block, "其余 2 条未展示") {
		t.Errorf("未收录条数应如实报出:\n%s", block)
	}
	if !strings.Contains(block, "…") {
		t.Errorf("被截断的末条应带省略号:\n%s", block)
	}
	if strings.Contains(block, "第二条进不来") {
		t.Errorf("预算外的便签不应进入底稿块:\n%s", block)
	}
	// 逐条可收下时不报「未展示」。
	fit := sinDraftBlock(sinNotesDoc{Notes: []string{"a", "b"}})
	if strings.Contains(fit, "未展示") {
		t.Errorf("预算内便签不应报未展示:\n%s", fit)
	}
}

// TestSinDraftBlockEmptyParts 大纲与便签独立判空：只有大纲/只有便签都成块，
// 全空（含空白串）= 空串。
func TestSinDraftBlockEmptyParts(t *testing.T) {
	if got := sinDraftBlock(sinNotesDoc{Outline: "  \n "}); got != "" {
		t.Errorf("空白大纲 = 空串, got %q", got)
	}
	outlineOnly := sinDraftBlock(sinNotesDoc{Outline: "三幕结构"})
	if !strings.Contains(outlineOnly, "大纲") || strings.Contains(outlineOnly, "设定便签") {
		t.Errorf("只有大纲时不应出现便签段:\n%s", outlineOnly)
	}
	notesOnly := sinDraftBlock(sinNotesDoc{Notes: []string{"伏笔：怀表"}})
	if !strings.Contains(notesOnly, "设定便签") || strings.Contains(notesOnly, "大纲：") {
		t.Errorf("只有便签时不应出现大纲段:\n%s", notesOnly)
	}
}

// TestSinDraftForPrompt 直注取稿 fail-open：文件缺失 = 空文档；非法 id =
// 空文档不报错（底稿是辅助数据，绝不阻断故事创作）。
func TestSinDraftForPrompt(t *testing.T) {
	a := newSinTestApp(t)
	if d := a.sinDraftForPrompt("sin_不存在_9"); len(d.Notes) != 0 || d.Outline != "" {
		t.Errorf("缺失文件应返回空文档: %+v", d)
	}
	if d := a.sinDraftForPrompt("bad/id"); len(d.Notes) != 0 || d.Outline != "" {
		t.Errorf("非法 id 应返回空文档: %+v", d)
	}
}
