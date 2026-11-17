package app

import (
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/guardian"
)

// T11 双板块审计：Guardian 评审调用必须跟随所属 feature，不能固定写 office。
func TestNewGuardianReviewerFeatureRouting(t *testing.T) {
	cfg := &gaeaConfig.Config{Guardian: gaeaConfig.GuardianConfig{Enabled: true}}
	rev := newGuardianReviewer(&App{}, cfg, "wenxue")
	runner, ok := rev.(*guardian.Runner)
	if !ok {
		t.Fatalf("reviewer type = %T, want *guardian.Runner", rev)
	}
	caller, ok := runner.Caller.(guardianCaller)
	if !ok {
		t.Fatalf("caller type = %T, want guardianCaller", runner.Caller)
	}
	if caller.feature != "wenxue" {
		t.Fatalf("guardian feature = %q, want wenxue", caller.feature)
	}
	if got := newGuardianReviewer(&App{}, &gaeaConfig.Config{}, "wenxue"); got != nil {
		t.Fatalf("未启用时应返回 nil，got %T", got)
	}
}
