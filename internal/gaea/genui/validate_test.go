package genui

import (
	"strings"
	"testing"
)

func TestValidateSpec_OK(t *testing.T) {
	raw := `{"title":"看板","items":[
		{"type":"stat","label":"营收","value":"¥128k"},
		{"type":"chart","kind":"bars","data":[{"label":"1月","value":98}]}
	]}`
	r := ValidateSpec(raw)
	if !r.OK {
		t.Fatalf("want OK, got %v", r.Errors)
	}
	if r.Nodes != 2 {
		t.Fatalf("nodes = %d, want 2", r.Nodes)
	}
}

func TestValidateSpec_SingleComponentRoot(t *testing.T) {
	r := ValidateSpec(`{"type":"callout","content":"hi","tone":"info"}`)
	if !r.OK {
		t.Fatalf("want OK for single component root, got %v", r.Errors)
	}
}

func TestValidateSpec_UnknownTypeAndMissingField(t *testing.T) {
	r := ValidateSpec(`{"items":[
		{"type":"evil"},
		{"type":"stat","label":"x"}
	]}`)
	if r.OK {
		t.Fatal("want not OK")
	}
	joined := strings.Join(r.Errors, "\n")
	if !strings.Contains(joined, `未知组件 type "evil"`) {
		t.Fatalf("missing unknown-type error: %s", joined)
	}
	if !strings.Contains(joined, "stat 缺少必填字段 value") {
		t.Fatalf("missing required-field error: %s", joined)
	}
}

func TestValidateSpec_SyntaxError(t *testing.T) {
	r := ValidateSpec(`{oops`)
	if r.OK || len(r.Errors) == 0 {
		t.Fatalf("want syntax error, got %+v", r)
	}
}

func TestValidateSpec_DepthBudget(t *testing.T) {
	var raw strings.Builder
	raw.WriteString(`{"items":[]}`)
	r := ValidateSpec(raw.String())
	if r.OK {
		t.Fatal("empty items should fail")
	}
}
