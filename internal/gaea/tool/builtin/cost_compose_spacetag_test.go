package builtin

import "testing"

// TestCostComposeSpaceTag 隔离红线：组价开口=工地话题，钉 work 不进会客厅。
func TestCostComposeSpaceTag(t *testing.T) {
	if got := SpaceTagOf(costCompose{}); got != "work" {
		t.Fatalf("SpaceTag = %q, want work", got)
	}
}
