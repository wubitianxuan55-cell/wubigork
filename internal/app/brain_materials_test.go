package app

import (
	"strings"
	"testing"
)

func TestBuildBrainMaterials(t *testing.T) {
	bs := &BrainStore{
		main:  &fakeMainBrain{rows: map[string]string{}},
		left:  &fakeLeftBrain{},
		right: &fakeRightBrain{rows: map[string]string{"甲方A|偏好": "保守报价"}},
	}
	out := buildBrainMaterials(bs, "土壤修复标书", "报价")
	if len(out) == 0 || !strings.Contains(out[0], "保守报价") {
		t.Fatalf("materials = %+v", out)
	}
	if !strings.Contains(out[0], "右脑") {
		t.Fatalf("应标注右脑来源: %q", out[0])
	}
	if got := buildBrainMaterials(nil, "x"); got != nil {
		t.Fatalf("nil brain 应返回空: %+v", got)
	}
}
