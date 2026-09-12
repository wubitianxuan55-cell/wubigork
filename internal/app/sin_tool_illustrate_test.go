package app

// sin_illustrate 工具测试（v4.270 入册，源自 .tmp/sin-next-knife 留存件；真机走查
// 拍板放行——单链路委托既有 SinIllustrate，产物走 Artifacts 轨迹+落库回写）。

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestSinIllustrateToolArtifactAndText sin_illustrate：真的出图 → 产物记进
// Artifacts（前端过程卡据此渲染缩略图）→ 返回文本把路径交给模型 → 写类工具。
func TestSinIllustrateToolArtifactAndText(t *testing.T) {
	fake := &sinRefBackend{}
	a := newSinRefTestApp(t, "xai", "grok-image", fake)
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}

	tool := sinToolByName(a.sinToolSet(story.ID), sinToolIllustrate)
	if tool == nil {
		t.Fatal("sin_illustrate 未注册")
	}
	if tool.ReadOnly() {
		t.Error("sin_illustrate 会落盘出图，ReadOnly 必须为 false")
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("Schema 不是合法 JSON: %v", err)
	}
	if req, _ := schema["required"].([]interface{}); len(req) != 1 || req[0] != "prompt" {
		t.Errorf("Schema required = %v, want [prompt]", schema["required"])
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(
		`{"prompt":"雨夜里的短发女人，半身近景","caption":"雨夜"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("应调用一次图像生成，got %d", len(fake.requests))
	}
	if !strings.Contains(fake.requests[0].Prompt, "雨夜里的短发女人") {
		t.Errorf("画面描述未传给图像模型: %q", fake.requests[0].Prompt)
	}

	p, ok := tool.(sinToolArtifactProvider)
	if !ok {
		t.Fatal("sin_illustrate 必须实现 sinToolArtifactProvider")
	}
	arts := p.Artifacts()
	if len(arts) != 1 {
		t.Fatalf("产物条数 = %d, want 1", len(arts))
	}
	if arts[0].Kind != sinToolArtifactKindImage {
		t.Errorf("产物 kind = %q, want %q", arts[0].Kind, sinToolArtifactKindImage)
	}
	if arts[0].Caption != "雨夜" {
		t.Errorf("产物 caption = %q, want 雨夜", arts[0].Caption)
	}
	if !strings.HasPrefix(arts[0].Path, sinArtDir()) {
		t.Errorf("产物必须落在原罪自有目录 %s, got %s", sinArtDir(), arts[0].Path)
	}
	if _, err := os.Stat(arts[0].Path); err != nil {
		t.Errorf("产物文件不存在: %v", err)
	}
	if !strings.Contains(out, arts[0].Path) {
		t.Errorf("返回文本应把路径讲给模型: %q", out)
	}
	if !strings.Contains(out, "不必再为它写插图标记") {
		t.Errorf("返回文本应提醒别再写标记（避免同一张图生成两次）: %q", out)
	}
}

// TestSinIllustrateToolRejectsBadArgs 缺 prompt / 参数不是合法 JSON：如实报错，
// 不产生产物（模型可据此自纠重调）。
func TestSinIllustrateToolRejectsBadArgs(t *testing.T) {
	a := newSinRefTestApp(t, "xai", "grok-image", &sinRefBackend{})
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	tool := sinToolByName(a.sinToolSet(story.ID), sinToolIllustrate)
	if tool == nil {
		t.Fatal("sin_illustrate 未注册")
	}
	for _, raw := range []string{`{}`, `{"prompt":"   "}`, `{"prompt":`} {
		if _, err := tool.Execute(context.Background(), json.RawMessage(raw)); err == nil {
			t.Errorf("args=%s 应报错", raw)
		}
	}
	if arts := tool.(sinToolArtifactProvider).Artifacts(); len(arts) != 0 {
		t.Errorf("报错路径不该留下产物: %+v", arts)
	}
}


// TestSinIllustrateInToolSet sin_illustrate 在工具集（outline 之后、export 之前）。
func TestSinIllustrateInToolSet(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	tools := a.sinToolSet(story.ID)
	names := make([]string, 0, len(tools))
	posIll, posExp := -1, -1
	for i, tl := range tools {
		names = append(names, tl.Name())
		if tl.Name() == sinToolIllustrate {
			posIll = i
		}
		if tl.Name() == sinToolExport {
			posExp = i
		}
	}
	if posIll < 0 || posExp < 0 || posIll > posExp {
		t.Fatalf("工具顺序异常: %v", names)
	}
}

// TestSinPersistToolArtifacts 落库回写：轨迹产物按 tool0..toolN 并入
// extra.illustrations（cue 前缀防与正文标记数字键互踩）；messageID<=0 跳过。
func TestSinPersistToolArtifacts(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	extra := `{"tools":[{"id":"c1","name":"sin_illustrate","artifacts":[{"kind":"image","path":"C:/art/x.png","caption":"雨夜"}]}],"reasoning":"r"}`
	if err := a.appendChatExchange(story.ID, "画一张", "好的。", extra); err != nil {
		t.Fatalf("appendChatExchange: %v", err)
	}
	msgs, err := a.chatStore.ListMessages(story.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	var mid int64
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "assistant" {
			mid = msgs[i].ID
			break
		}
	}
	var trace []sinToolTrace
	if err := json.Unmarshal([]byte(`[{"id":"c1","name":"sin_illustrate","artifacts":[{"kind":"image","path":"C:/art/x.png","caption":"雨夜"}]}]`), &trace); err != nil {
		t.Fatalf("trace unmarshal: %v", err)
	}
	a.sinPersistToolArtifacts(mid, trace)
	msg, err := a.chatStore.GetMessage(mid)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	var ex map[string]any
	_ = json.Unmarshal([]byte(msg.Extra), &ex)
	ills, _ := ex["illustrations"].(map[string]any)
	if ills == nil || ills["tool0"] != "C:/art/x.png" {
		t.Fatalf("回写后 extra.illustrations = %v", ex["illustrations"])
	}

	// messageID<=0：不落库不报错
	a.sinPersistToolArtifacts(0, trace)
}
