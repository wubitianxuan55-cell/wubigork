package agent

// S4 产物路径分区：SingleModelPromptFor 空间参数化的契约测试。
// 兼容红线：work 缺省输出必须与历史 const SingleModelPrompt 逐字节一致
// （boot 缺省链路 + 前缀缓存 + 既有文本锚定都依赖逐字稳定）；只有 play
// 空间才把落盘规范的 exports/work 根段替换为分区路径。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// TestSingleModelPromptForDefaultVerbatim 缺省（work/空/非法值）输出逐字不变。
func TestSingleModelPromptForDefaultVerbatim(t *testing.T) {
	for _, space := range []string{"", spaces.SpaceWork, "Work", "PLAY", "bogus"} {
		if got := SingleModelPromptFor(space); got != SingleModelPrompt {
			t.Errorf("SingleModelPromptFor(%q) 与 const 逐字不一致（兼容红线）", space)
		}
	}
}

// TestSingleModelPromptForPlayRoots play 空间：根段替换为 .gaea/play/{exports,work}，
// 且不再出现现状根段；其余文本（角色/工作流/禁止事项）保持不变。
func TestSingleModelPromptForPlayRoots(t *testing.T) {
	play := SingleModelPromptFor(spaces.SpacePlay)
	if !strings.Contains(play, ".gaea/play/exports/") {
		t.Error("play 文本缺少 .gaea/play/exports/ 根段")
	}
	if !strings.Contains(play, ".gaea/play/work/") {
		t.Error("play 文本缺少 .gaea/play/work/ 根段")
	}
	if strings.Contains(play, ".gaea/exports/") {
		t.Error("play 文本仍含现状根段 .gaea/exports/（应全部替换）")
	}
	if strings.Contains(play, ".gaea/work/") {
		t.Error("play 文本仍含现状根段 .gaea/work/（应全部替换）")
	}

	// 替换只发生在路径根段：去掉根段差异后与 const 同构。
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, ".gaea/play/exports/", "ROOT")
		s = strings.ReplaceAll(s, ".gaea/play/work/", "ROOT2")
		return s
	}
	want := strings.ReplaceAll(strings.ReplaceAll(SingleModelPrompt, ".gaea/exports/", "ROOT"), ".gaea/work/", "ROOT2")
	if got := normalize(play); got != want {
		t.Error("play 渲染除根段外不应改动任何文本")
	}

	// 根段出现次数逐一对账（const 内 .gaea/exports/ 两处、.gaea/work/ 一处）。
	if n := strings.Count(SingleModelPrompt, promptExportsRoot); n != 2 {
		t.Errorf("const 内 %q 出现 %d 次, want 2（新增根段需同步本测试）", promptExportsRoot, n)
	}
	if n := strings.Count(SingleModelPrompt, promptWorkRoot); n != 1 {
		t.Errorf("const 内 %q 出现 %d 次, want 1", promptWorkRoot, n)
	}
}
