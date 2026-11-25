package memory

import (
	"context"
)

// SessionFactPromoter is the interface for promoting session facts to permanent
// storage. The controller implements this; boot stamps it on the agent context.
type SessionFactPromoter interface {
	PromoteSessionFacts() (int, error)
}

type promoterKey struct{}

// WithPromoter stamps p onto ctx so `remember`（action=promote）can read it.
// promote_session_facts 工具已于 2026-11-25 并入 remember（办公工具梳理批）。
func WithPromoter(ctx context.Context, p SessionFactPromoter) context.Context {
	return context.WithValue(ctx, promoterKey{}, p)
}

// PromoterFromContext returns the promoter, if any.
func PromoterFromContext(ctx context.Context) (SessionFactPromoter, bool) {
	p, ok := ctx.Value(promoterKey{}).(SessionFactPromoter)
	return p, ok && p != nil
}
