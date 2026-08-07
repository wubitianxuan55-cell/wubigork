package app

import (
	"strings"
	"testing"
)

func TestMainBrainChatDispatchWithMaterials(t *testing.T) {
	a := &App{}
	a.modules = NewModuleRegistry()
	_ = a.modules.Register(Module{
		ID: "office", Name: "方案", Intents: []string{"create"},
		Handle: func(input map[string]any) (map[string]any, error) {
			return map[string]any{"created": true, "title": input["title"]}, nil
		},
	})
	a.brain = &BrainStore{
		main:  &fakeMainBrain{rows: map[string]string{}},
		left:  &fakeLeftBrain{},
		right: &fakeRightBrain{rows: map[string]string{"甲方A|偏好": "保守报价"}},
	}
	out, err := a.MainBrainChat("报价")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"module": "office"`) || !strings.Contains(out, `"intent": "create"`) {
		t.Fatalf("out = %s", out)
	}
	// 材料注入：右脑命中应出现在结果里。
	if !strings.Contains(out, "保守报价") {
		t.Fatalf("materials 未注入: %s", out)
	}
}
