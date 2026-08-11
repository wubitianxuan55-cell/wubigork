package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// TestCapCharactersStripsHugePortraits 巨型 base64 剧照不随角色响应返回，
// 且不修改 agent 缓存里的原始数据（防止 Wails IPC 被撑爆卡死）。
func TestCapCharactersStripsHugePortraits(t *testing.T) {
	chars := []types.Character{
		{Name: "remote", PortraitURL: "https://imgen.x.ai/xai-image/abc"},
		{Name: "small", PortraitURL: "data:image/png;base64," + strings.Repeat("a", 100)},
		{Name: "huge", PortraitURL: "data:image/png;base64," + strings.Repeat("b", 400*1024)},
	}
	out := capCharacters(chars)
	if out[0].PortraitURL == "" {
		t.Fatal("远程 URL 不应被截断")
	}
	if out[1].PortraitURL == "" {
		t.Fatal("小内联头像不应被截断")
	}
	if out[2].PortraitURL != "" {
		t.Fatalf("超大内联头像应置空, got len=%d", len(out[2].PortraitURL))
	}
	if chars[2].PortraitURL == "" {
		t.Fatal("不应改动原始切片（agent 缓存）")
	}
}
