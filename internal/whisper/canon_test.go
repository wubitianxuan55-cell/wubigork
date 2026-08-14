package whisper

import (
	"strings"
	"testing"
)

// ─── T6-5.1: canon.go / canon_creator.go / canon_origin.go / canon_father_reference.go ──

func TestBuildAckemCanonBlock_DefaultName(t *testing.T) {
	got := BuildAckemCanonBlock("温柔", "")
	if !strings.Contains(got, "你的名字叫gaea") {
		t.Errorf("空名字应回退默认名 gaea: %q", got)
	}
	if !strings.Contains(got, "温柔") || !strings.Contains(got, "AI 伙伴") {
		t.Errorf("应含人设与本质: %q", got)
	}
	if !strings.Contains(got, "Jason") {
		t.Errorf("应含创造者: %q", got)
	}
}

func TestBuildAckemCanonBlock_CustomName(t *testing.T) {
	got := BuildAckemCanonBlock("傲娇", "Luna")
	if !strings.Contains(got, "你的名字叫Luna") {
		t.Errorf("应使用自定义名字: %q", got)
	}
}

func TestBuildStrangerGuardBlock(t *testing.T) {
	got := BuildStrangerGuardBlock("温柔")
	if !strings.Contains(got, "初识防护") || !strings.Contains(got, "温柔") {
		t.Errorf("初识防护块错误: %q", got)
	}
}

func TestBuildMandatorySpecialDateBlock(t *testing.T) {
	got := BuildMandatorySpecialDateBlock("情人节")
	if !strings.Contains(got, "情人节") || !strings.Contains(got, "必须") {
		t.Errorf("特殊日期块错误: %q", got)
	}
}

func TestShouldInjectStrangerGuard(t *testing.T) {
	if !ShouldInjectStrangerGuard(StageStranger) {
		t.Error("陌生人阶段应注入初识防护")
	}
	if ShouldInjectStrangerGuard(StageFamiliar) || ShouldInjectStrangerGuard(StageIntimate) {
		t.Error("非陌生人阶段不应注入")
	}
}

func TestDefaultCreatorSeeds(t *testing.T) {
	seeds := DefaultCreatorSeeds()
	if len(seeds) != 3 {
		t.Fatalf("应有 3 条创造者种子, got %d", len(seeds))
	}
	for _, s := range seeds {
		if s.Weight <= 0 || s.Subject == "" || s.Summary == "" {
			t.Errorf("种子字段不完整: %+v", s)
		}
	}
}

func TestDetectFatherRef_Hit(t *testing.T) {
	for _, msg := range []string{"Jason是谁", "谁创造了你", "你的主人是谁", "jason 在吗"} {
		sig := DetectFatherRef(msg)
		if sig == nil || sig.Kind != "explicit" || sig.Score != 0.9 {
			t.Errorf("DetectFatherRef(%q) = %+v, want explicit 0.9", msg, sig)
		}
	}
}

func TestDetectFatherRef_Miss(t *testing.T) {
	if got := DetectFatherRef("今天天气不错"); got != nil {
		t.Errorf("无关消息应返回 nil, got %+v", got)
	}
}

func TestBuildCreatorMemoryBlock_Empty(t *testing.T) {
	if got := BuildCreatorMemoryBlock(nil); got != "" {
		t.Errorf("空种子应返回空串, got %q", got)
	}
}

func TestBuildCreatorMemoryBlock_NonEmpty(t *testing.T) {
	got := BuildCreatorMemoryBlock([]CreatorMemorySeed{{Subject: "创造者", Summary: "我的创造者是Jason"}})
	if !strings.Contains(got, "创造者") || !strings.Contains(got, "Jason") {
		t.Errorf("创造者记忆块错误: %q", got)
	}
	if !strings.Contains(got, "当前用户") {
		t.Errorf("应强调当前用户优先: %q", got)
	}
}

func TestBuildOriginGuardBlock(t *testing.T) {
	got := BuildOriginGuardBlock()
	if !strings.Contains(got, OriginGuardMarker) || !strings.Contains(got, "回归用户") {
		t.Errorf("OEG 块错误: %q", got)
	}
}

func TestShouldSuppressOriginProactive(t *testing.T) {
	if ShouldSuppressOriginProactive(nil) {
		t.Error("nil 不应抑制")
	}
	if ShouldSuppressOriginProactive(&OriginExposure{State: OriginDeep}) == false {
		t.Error("OriginDeep 应抑制")
	}
	if ShouldSuppressOriginProactive(&OriginExposure{State: OriginGuardCooldown}) == false {
		t.Error("GuardCooldown 应抑制")
	}
	if ShouldSuppressOriginProactive(&OriginExposure{State: OriginNormal}) {
		t.Error("OriginNormal 不应抑制")
	}
}

func TestAdvanceOriginStreak_Progression(t *testing.T) {
	// 连续 5 次检测 → 到达 GuardCooldown
	ex := DefaultOriginExposure()
	ex = AdvanceOriginStreak(ex, true, 1)
	if ex.State != OriginEntry || ex.Streak != 1 {
		t.Errorf("第 1 次应 Entry, got %+v", ex)
	}
	ex = AdvanceOriginStreak(ex, true, 2)
	if ex.State != OriginExplore {
		t.Errorf("第 2 次应 Explore, got %+v", ex)
	}
	ex = AdvanceOriginStreak(ex, true, 3)
	if ex.State != OriginExplore || ex.Streak != 3 {
		t.Errorf("第 3 次仍 Explore, got %+v", ex)
	}
	ex = AdvanceOriginStreak(ex, true, 4)
	if ex.State != OriginDeep {
		t.Errorf("第 4 次应 Deep, got %+v", ex)
	}
	ex = AdvanceOriginStreak(ex, true, 5)
	if ex.State != OriginGuardCooldown || ex.CooldownUntilTurn != 5+int(OriginCooldownTurns) {
		t.Errorf("第 5 次应 GuardCooldown 并设 cooldown, got %+v", ex)
	}
}

func TestAdvanceOriginStreak_Decay(t *testing.T) {
	ex := &OriginExposure{State: OriginDeep, Streak: 4, CooldownUntilTurn: 0}
	ex = AdvanceOriginStreak(ex, false, 1)
	if ex.Streak != 3 || ex.State != OriginExplore {
		t.Errorf("无检测应衰减到 3/Explore, got %+v", ex)
	}
}

func TestAdvanceOriginStreak_CooldownBlocks(t *testing.T) {
	ex := &OriginExposure{State: OriginNormal, Streak: 0, CooldownUntilTurn: 10}
	got := AdvanceOriginStreak(ex, true, 5)
	if got.Streak != 0 || got.State != OriginNormal {
		t.Errorf("cooldown 期间不应推进, got %+v", got)
	}
}

func TestAdvanceOriginStreak_NilExposure(t *testing.T) {
	got := AdvanceOriginStreak(nil, true, 1)
	if got == nil || got.Streak != 1 || got.State != OriginEntry {
		t.Errorf("nil 应初始化并推进, got %+v", got)
	}
}

func TestClassifyFatherRef(t *testing.T) {
	cases := []struct {
		msg  string
		want FatherRefKind
	}{
		{"谁创造了你", FatherRefAckemCreator},
		{"Jason 和你的关系是什么", FatherRefAckemCreator},
		{"讲讲你的出身故事", FatherRefAckemCreator},
		{"我爸今天催我回家", FatherRefUserFamily},
		{"我和我爸爸吵架了", FatherRefUserFamily},
		{"我妈让我回去吃饭", FatherRefUserFamily},
		{"今天天气不错", FatherRefNone},
		{"晚安", FatherRefNone},
	}
	for _, c := range cases {
		if got := ClassifyFatherRef(c.msg); got != c.want {
			t.Errorf("ClassifyFatherRef(%q) = %q, want %q", c.msg, got, c.want)
		}
	}
}

func TestClassifyFatherRefStrict(t *testing.T) {
	// 回归用例全集：每类至少命中一个
	creatorHit, familyHit, noneHit := false, false, false
	for _, c := range FatherRefRegressionCases {
		if got := ClassifyFatherRefStrict(c.Query); got != c.Kind {
			t.Errorf("回归用例 %q 应判为 %q, got %q", c.Query, c.Kind, got)
		}
		switch c.Kind {
		case FatherRefAckemCreator:
			creatorHit = true
		case FatherRefUserFamily:
			familyHit = true
		default:
			noneHit = true
		}
	}
	if !creatorHit || !familyHit || !noneHit {
		t.Errorf("回归用例应覆盖三类, creator=%v family=%v none=%v", creatorHit, familyHit, noneHit)
	}
	// 严格模式兜底到宽松
	if got := ClassifyFatherRefStrict("随便聊聊"); got != FatherRefNone {
		t.Errorf("兜底失败: %q", got)
	}
}
