package app

// Guardian 评审装配（codex 蒸馏 T11 刀三）：把 [guardian] 配置折算成
// guardian.Runner——评审调用按所属板块走本地优先路由（与 dream/会话命名同
// 口径），一次 ChatSimple 调用，小 max_tokens（评审输出是一行 JSON）。

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/guardian"
)

// guardianCaller 是 guardian.Runner 的调用面：按 feature 走本地引擎优先，
// 失败即失败
// （guardian 侧分类回落用户审批卡，不在调用面重试）。
type guardianCaller struct {
	app     *App
	feature string
}

// Chat 完成一次评审调用。
func (g guardianCaller) Chat(ctx context.Context, system, user string) (string, error) {
	a := g.app
	if a == nil || a.clientRef() == nil {
		return "", errors.New("ai client unavailable")
	}
	feature := strings.TrimSpace(g.feature)
	if feature == "" {
		feature = "office"
	}
	eng, model, _ := a.routeOfficeLocal(feature)
	if strings.TrimSpace(model) == "" {
		return "", errors.New(feature + " chat model unavailable")
	}
	out, err := a.clientRef().ChatSimpleStreamWithOptions(ctx, model, system, user,
		ai.ChatSimpleOptions{EngineID: eng, Feature: feature, Temperature: 0.1, MaxTokens: 400})
	if err != nil {
		return "", err
	}
	return out, nil
}

// newGuardianReviewer 按 [guardian] 配置构造评审器；未启用/配置缺省返回
// nil（审批纯人工，现状逐字节）。
func newGuardianReviewer(a *App, cfg *gaeaConfig.Config, feature string) guardian.Reviewer {
	if cfg == nil || !cfg.Guardian.Enabled {
		return nil
	}
	if strings.TrimSpace(feature) == "" {
		feature = "office"
	}
	r := &guardian.Runner{Caller: guardianCaller{app: a, feature: feature}}
	if ms := cfg.Guardian.TimeoutMs; ms > 0 {
		r.Timeout = time.Duration(ms) * time.Millisecond
	}
	if b := cfg.Guardian.InputBudget; b > 0 {
		r.InputBudget = b
	}
	return r
}
