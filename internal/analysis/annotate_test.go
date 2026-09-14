package analysis

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

const annotateContent = "林晚推开祠堂的门，铜匣静静躺在供桌下。她想起祖母的话：「月圆之夜，勿开铜匣。」\n\n夜风穿堂而过，烛火明明灭灭。铜匣的锁，竟在无风自转。"

// TestLocateKeyword 四段式定位（§8.4 A2）：精确/去标点反投影/前缀/未命中。
func TestLocateKeyword(t *testing.T) {
	// ① 精确匹配
	pos, length := LocateKeyword(annotateContent, "铜匣静静躺在供桌下")
	if pos < 0 || length != len([]rune("铜匣静静躺在供桌下")) {
		t.Fatalf("精确匹配失败: pos=%d len=%d", pos, length)
	}
	// ② 去标点匹配 + 反投影：keyword 省略逗号，pos 应回投到原文首字
	pos, length = LocateKeyword(annotateContent, "祠堂的门铜匣")
	if pos < 0 {
		t.Fatalf("去标点匹配失败")
	}
	if got := string([]rune(annotateContent)[pos]); got != "林" && got != "祠" {
		t.Fatalf("反投影起点错位: pos=%d rune=%q", pos, got)
	}
	if length <= 0 || length > len([]rune("祠堂的门，铜匣")) {
		t.Fatalf("反投影长度不合理: %d", length)
	}
	// ③ 长 keyword 前 15 rune 前缀
	long := "林晚推开祠堂的门，铜匣静静躺在供桌之下纹丝不动" // 尾部与原文不同
	pos, length = LocateKeyword(annotateContent, long)
	if pos < 0 || length != 15 {
		t.Fatalf("前缀匹配失败: pos=%d len=%d", pos, length)
	}
	// ④ 未命中
	if pos, _ := LocateKeyword(annotateContent, "毫不相干的句子"); pos != -1 {
		t.Fatalf("未命中应返回 -1: %d", pos)
	}
	// 空输入
	if pos, _ := LocateKeyword("", "x"); pos != -1 {
		t.Fatalf("空正文应 -1")
	}
}

// TestBuildAnnotations 标注构建：锚定类定位成功保留/失败丢弃、suggestion 留 -1。
func TestBuildAnnotations(t *testing.T) {
	v2 := &types.AnalysisResultV2{
		Hooks: []types.Hook{
			{Type: "悬念", Content: "铜匣的锁，竟在无风自转", Strength: 8,
				Keyword: types.Keyword{Text: "铜匣的锁，竟在无风自转"}},
		},
		Foreshadows: []types.ForeshadowHit{
			{Type: "planted", Title: "星门", Content: "月圆之夜星门开启", Strength: 7,
				Keyword: types.Keyword{Text: "月圆之夜，勿开铜匣"}}, // 与原文去标点可命中
		},
		PlotPoints: []types.PlotPoint{
			{Content: "无法定位的推进点", Importance: 0.9}, // 无 keyword → 丢弃
		},
		Suggestions: []string{"收紧中段节奏", "深化祖母形象"},
	}
	anns := BuildAnnotations(annotateContent, v2)

	var hook, foreshadow *types.Annotation
	suggestionCount := 0
	for i := range anns {
		switch anns[i].Type {
		case "hook":
			hook = &anns[i]
		case "foreshadow":
			foreshadow = &anns[i]
		case "suggestion":
			suggestionCount++
			if anns[i].Pos != -1 {
				t.Fatalf("suggestion 应 Pos=-1: %+v", anns[i])
			}
		case "plot_point":
			t.Fatalf("无 keyword 的推进点应被丢弃: %+v", anns[i])
		}
	}
	if hook == nil || hook.Pos < 0 {
		t.Fatalf("hook 应定位成功: %+v", hook)
	}
	if foreshadow == nil || foreshadow.Pos < 0 {
		t.Fatalf("foreshadow 应经去标点定位成功: %+v", foreshadow)
	}
	if suggestionCount != 2 {
		t.Fatalf("suggestion 应 2 条: %d", suggestionCount)
	}
	fr := []rune(annotateContent)
	if !strings.Contains(string(fr[foreshadow.Pos:foreshadow.Pos+foreshadow.Length]), "月圆之夜") {
		t.Fatalf("反投影区间应覆盖原文片段")
	}
}

// TestBuildAnnotations_Empty 空载荷安全。
func TestBuildAnnotations_Empty(t *testing.T) {
	if got := BuildAnnotations("正文", &types.AnalysisResultV2{}); len(got) != 0 {
		t.Fatalf("空载荷应 0 条: %+v", got)
	}
	if got := BuildAnnotations("正文", nil); got != nil {
		t.Fatalf("nil 载荷应返回 nil")
	}
}
