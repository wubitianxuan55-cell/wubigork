package types

import (
	"fmt"
	"math"
	"testing"
)

// TestWordOverlap 字符级 n-gram 相似度（spec §3.5）：完全相同=1.0、
// 空串安全、中文按 rune 不按 byte、3-gram 权重更高。
func TestWordOverlap(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want float64
	}{
		{"完全相同", "神秘铜匣的钥匙藏在祠堂", "神秘铜匣的钥匙藏在祠堂", 1.0},
		{"空白归一化后相同", "神秘铜匣的钥匙", "神秘铜匣的 钥匙\n", 1.0},
		{"大小写归一化", "The Silver Key", "the silver key", 1.0},
		{"完全不同", "青铜古灯", "左臂旧伤", 0},
		{"空串对非空", "", "神秘铜匣", 0},
		{"双空串", "", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WordOverlap(tc.a, tc.b)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("WordOverlap(%q,%q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}

	// rune 安全：中文按字符切 n-gram。前缀包含关系应有明显大于 0 的相似度，
	// 且 byte 切分的错误实现会得到不同（错位）结果——锁住 rune 口径。
	prefix := WordOverlap("神秘铜匣的钥匙", "神秘铜匣的钥匙藏在祠堂地砖之下")
	if prefix <= 0.2 || prefix > 1.0 {
		t.Fatalf("前缀包含相似度应在 (0.2,1.0]，实际 %v", prefix)
	}
}

// TestStripResolveTitleSuffix 回收标题后缀剥离（只剥一次）。
func TestStripResolveTitleSuffix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"铜匣之谜·回收", "铜匣之谜·"},
		{"铜匣之谜回收", "铜匣之谜"},
		{"铜匣之谜揭示", "铜匣之谜"},
		{"铜匣之谜解答", "铜匣之谜"},
		{"铜匣之谜兑现", "铜匣之谜"},
		{"回收真相", "回收真相"}, // 后缀不在词表，不剥
		{"无后缀", "无后缀"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := StripResolveTitleSuffix(tc.in); got != tc.want {
			t.Fatalf("Strip(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestMatchForeshadowByContent 六策略加权评分（spec §3.4 / P2.1 验收）：
// 完全相同标题=1.0、后缀剥离命中=0.95、min_similarity=0.5 边界、同分取先出现者。
func TestMatchForeshadowByContent(t *testing.T) {
	planted := []Foreshadow{
		{ID: "f1", Title: "神秘铜匣", Description: "祠堂地砖下的铜匣，钥匙下落不明", Category: "item", PlantedIn: "001.md"},
		{ID: "f2", Title: "左臂旧伤", Description: "主角左臂有旧伤，阴雨隐隐作痛", Category: "mystery", PlantedIn: "002.md"},
		{ID: "f3", Title: "铜匣之谜", Description: "铜匣上的纹路指向族谱秘辛", Category: "item", PlantedIn: "003.md"},
	}

	t.Run("标题完全相同=1.0", func(t *testing.T) {
		got, score := MatchForeshadowByContent(ForeshadowHit{Title: "神秘铜匣"}, planted, 0.5)
		if got == nil || got.ID != "f1" || score != 1.0 {
			t.Fatalf("应命中 f1 满分，实际 %+v score=%v", got, score)
		}
	})

	t.Run("后缀剥离命中", func(t *testing.T) {
		// 「铜匣之谜回收」剥后缀 = f3 标题 → 0.95 路径
		got, score := MatchForeshadowByContent(ForeshadowHit{Title: "铜匣之谜回收"}, planted, 0.5)
		if got == nil || got.ID != "f3" || score != 0.95 {
			t.Fatalf("后缀剥离应命中 f3@0.95，实际 %+v score=%v", got, score)
		}
	})

	t.Run("包含关系0.8与同分取先出现者", func(t *testing.T) {
		// 两个候选同为「包含关系」0.8（无完全相等者）→ 严格大于保证取先出现者
		both := []Foreshadow{
			{ID: "early", Title: "古钟", Description: "村口古钟每晚自鸣", PlantedIn: "001.md"},
			{ID: "late", Title: "村口古钟之谜上", Description: "古钟内藏半张地图", PlantedIn: "002.md"},
		}
		got, score := MatchForeshadowByContent(ForeshadowHit{Title: "村口古钟之谜"}, both, 0.5)
		if got == nil || got.ID != "early" || score != 0.8 {
			t.Fatalf("包含关系应 0.8 且同分取先出现者（early），实际 %+v score=%v", got, score)
		}
	})

	t.Run("min_similarity_0.5_边界", func(t *testing.T) {
		// 仅策略5 分类一致 = 0.1：低于阈值不采纳
		got, score := MatchForeshadowByContent(ForeshadowHit{Category: "item", Content: "风马牛不相及的内容"}, planted, 0.5)
		if got != nil {
			t.Fatalf("低于阈值不应采纳，实际 %+v score=%v", got, score)
		}
		// 策略3 内容相似 0.6 ≥ 0.5：恰好采纳
		same := []Foreshadow{{ID: "c1", Description: "月圆之夜星门开启"}}
		got, score = MatchForeshadowByContent(ForeshadowHit{Content: "月圆之夜星门开启"}, same, 0.5)
		if got == nil || got.ID != "c1" || score < 0.5 {
			t.Fatalf("内容完全相同（0.6 权重）应过阈值命中，实际 %+v score=%v", got, score)
		}
		// 阈值提高后同样分数被拒
		got, _ = MatchForeshadowByContent(ForeshadowHit{Content: "月圆之夜星门开启"}, same, 0.7)
		if got != nil {
			t.Fatalf("阈值 0.7 不应采纳，实际 %+v", got)
		}
	})

	t.Run("关键词命中与累加策略", func(t *testing.T) {
		// 策略2 关键词命中 0.75 + 策略5 分类一致 0.1 = 0.85
		got, score := MatchForeshadowByContent(ForeshadowHit{
			Title: "完全不同的标题", Keyword: Keyword{Text: "铜匣"}, Category: "item",
		}, planted, 0.5)
		if got == nil || got.ID != "f1" || math.Abs(score-0.85) > 1e-9 {
			t.Fatalf("关键词+分类应命中 f1@0.85，实际 %+v score=%v", got, score)
		}
	})

	t.Run("引用章号与角色累加", func(t *testing.T) {
		got, score := MatchForeshadowByContent(ForeshadowHit{
			Category: "item", ReferenceChapter: 1, RelatedChars: []string{"林晚", "陈铮"},
		}, planted, 0.5)
		// f1: 策略5(0.1) + 策略4(0.15) + 策略6(交2/并2=1.0*0.1) = 0.35 —— 仍低于阈值
		if got != nil {
			t.Fatalf("纯累加不足 0.5 不应命中，实际 %+v score=%v", got, score)
		}
		_ = score
	})

	t.Run("空候选与全低分", func(t *testing.T) {
		if got, _ := MatchForeshadowByContent(ForeshadowHit{Title: "任意"}, nil, 0.5); got != nil {
			t.Fatalf("空候选应返回 nil")
		}
	})
}

// TestMatchForeshadowByContent_TieUsesFirst 同分取先出现者的确定性（planted
// 列表按埋入章升序传入时 = 取最早埋入的，MuMu :1648-1650）。
func TestMatchForeshadowByContent_TieUsesFirst(t *testing.T) {
	mk := func(id string) Foreshadow {
		return Foreshadow{ID: id, Title: fmt.Sprintf("信物%d", len(id)), Description: "一枚成对的黑玉扳指"}
	}
	// 两个候选除 ID 外完全相同 → 同分，严格大于保证只取先出现者
	tie := []Foreshadow{mk("first"), mk("second")}
	got, _ := MatchForeshadowByContent(ForeshadowHit{Title: "信物5", Content: "一枚成对的黑玉扳指"}, tie, 0.5)
	if got == nil || got.ID != "first" {
		t.Fatalf("同分应取先出现者 first，实际 %+v", got)
	}
}
