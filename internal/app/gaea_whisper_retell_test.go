package app

import (
	"strings"
	"testing"
)

func TestGaeaWhisperMemoryRetell_Episode(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedReplayEpisode(t, a.whisperDataRoot, "whisper_retellA", "epRetellA")

	got, err := a.whisperState.GaeaWhisperMemoryRetell("episode", "epRetellA", "p1")
	if err != nil {
		t.Fatalf("GaeaWhisperMemoryRetell: %v", err)
	}
	if !strings.Contains(got, "你好呀") {
		t.Errorf("重述应返回模型回复, got %q", got)
	}
}

func TestGaeaWhisperMemoryRetell_Anchor(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedAnchorScenario(t, a)

	got, err := a.whisperState.GaeaWhisperMemoryRetell("anchor", "ancBirthday", "p1")
	if err != nil {
		t.Fatalf("GaeaWhisperMemoryRetell(anchor): %v", err)
	}
	if !strings.Contains(got, "你好呀") {
		t.Errorf("锚点重述应返回模型回复, got %q", got)
	}
}

func TestGaeaWhisperMemoryRetell_UnknownKindAndMissing(t *testing.T) {
	a := newChatServiceTestApp(t)
	if _, err := a.whisperState.GaeaWhisperMemoryRetell("story", "x", "p1"); err == nil ||
		!strings.Contains(err.Error(), "未知记忆类型") {
		t.Fatalf("未知 kind 应报错, got %v", err)
	}
	if _, err := a.whisperState.GaeaWhisperMemoryRetell("episode", "nope", "p1"); err == nil ||
		!strings.Contains(err.Error(), "情节不存在") {
		t.Fatalf("不存在的情节应报错, got %v", err)
	}
}

func TestBuildMemoryRetellContext(t *testing.T) {
	ctx := buildMemoryRetellContext("深夜一起改 bug", "兴奋", []WhisperReplayLine{
		{TurnIndex: 3, Role: "user", Text: "这个 bug 搞不定了"},
		{TurnIndex: 3, Role: "assistant", Text: "把报错贴给我看看"},
	})
	for _, want := range []string{"深夜一起改 bug", "兴奋", "对方：这个 bug 搞不定了", "gaea：把报错贴给我看看"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("重述上下文应包含 %q, got:\n%s", want, ctx)
		}
	}
	if strings.Contains(ctx, "用户") {
		t.Errorf("重述上下文不应出现「用户」字样（应替换为对方/gaea）:\n%s", ctx)
	}
}
