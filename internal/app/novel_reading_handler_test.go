package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
)

// TestNovelReadingAsk_Guards 覆盖 AI 伴读的守卫分支（不触发真实 LLM）。
func TestNovelReadingAsk_Guards(t *testing.T) {
	a := newCharacterLibTestApp(t) // client 为 nil 的干净 App

	if _, err := a.NovelReadingAsk("summary", "第1章", "正文", "", ""); err == nil || !strings.Contains(err.Error(), "AI 客户端未初始化") {
		t.Fatalf("client 为空时应报未初始化: %v", err)
	}

	// 提供 client 骨架后走内容守卫（守卫分支在调用 LLM 之前返回）
	a.client = &ai.Client{}
	if _, err := a.NovelReadingAsk("summary", "第1章", "   ", "", ""); err == nil || !strings.Contains(err.Error(), "本章暂无内容") {
		t.Fatalf("空正文应报错: %v", err)
	}
	if _, err := a.NovelReadingAsk("ask", "第1章", "正文", "摘选", ""); err == nil || !strings.Contains(err.Error(), "请输入问题") {
		t.Fatalf("空问题应报错: %v", err)
	}
	if _, err := a.NovelReadingAsk("ask", "第1章", "正文", "", "问题"); err == nil || !strings.Contains(err.Error(), "请先摘选原文") {
		t.Fatalf("空摘选应报错: %v", err)
	}
	if _, err := a.NovelReadingAsk("unknown", "第1章", "正文", "", ""); err == nil || !strings.Contains(err.Error(), "未知的伴读类型") {
		t.Fatalf("未知类型应报错: %v", err)
	}
}
