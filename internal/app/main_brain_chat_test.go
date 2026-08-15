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

// TestMainBrainChatOfficeFullChain 缺陷 2 修复回归：initModules 已注册 office，
// 输入「写一份标书」必须命中 office/create 并派发执行（不再静默跳过），
// 且 office Handle（D8：路由 GaeaSend）返回非空结构、无 error。
func TestMainBrainChatOfficeFullChain(t *testing.T) {
	a := &App{}
	a.initModules()
	if !a.modules.Has("office") {
		t.Fatal("initModules 未注册 office 模块（缺陷 2 复发）")
	}
	out, err := a.MainBrainChat("写一份标书")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"\"module\": \"office\"", "\"intent\": \"create\"", "\"status\": \"submitted\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("out 缺少 %s: %s", want, out)
		}
	}
	if strings.Contains(out, "\"error\"") {
		t.Fatalf("office 派发不应产生 error: %s", out)
	}
}

// TestMainBrainChatUnknownModuleLogsAndNoPanic 未注册模块不再静默：必须记录
// slog.Warn 告警，且返回结构保持兼容（module/intent 保留、无 output、不 panic）。
func TestMainBrainChatUnknownModuleLogsAndNoPanic(t *testing.T) {
	a := &App{}
	a.modules = NewModuleRegistry() // 空注册表：office 未注册
	records := captureLogs(t, func() {
		out, err := a.MainBrainChat("写一份标书")
		if err != nil {
			t.Fatalf("MainBrainChat 不应返回 error: %v", err)
		}
		if !strings.Contains(out, "\"module\": \"office\"") || !strings.Contains(out, "\"intent\": \"create\"") {
			t.Fatalf("out = %s", out)
		}
		if strings.Contains(out, "\"output\"") {
			t.Fatalf("未注册模块不应有 output: %s", out)
		}
	})
	if !logContainsMsg(records, "模块未注册") {
		t.Fatalf("未记录模块未注册告警, got %v", logMsgs(records))
	}
}

// TestMainBrainChatUnknownIntentNoPanic 已注册模块遇到未知 intent：Dispatch
// 返回错误（不 panic），MainBrainChat 把错误写入结果结构。
func TestMainBrainChatUnknownIntentNoPanic(t *testing.T) {
	a := &App{}
	a.modules = NewModuleRegistry()
	_ = a.modules.Register(Module{
		ID: "office", Name: "方案", Intents: []string{"other"}, // 无 create
		Handle: func(input map[string]any) (map[string]any, error) {
			return map[string]any{"x": 1}, nil
		},
	})
	out, err := a.MainBrainChat("写一份标书")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\"module\": \"office\"") {
		t.Fatalf("out = %s", out)
	}
	if !strings.Contains(out, "\"error\"") || !strings.Contains(out, "unknown intent") {
		t.Fatalf("未知 intent 应写入 error, out = %s", out)
	}
}

// TestModuleRegistryDispatchUnknownModuleLogs Dispatch 未知模块必须记录 slog.Error
// 并返回 error（不 panic）。
func TestModuleRegistryDispatchUnknownModuleLogs(t *testing.T) {
	reg := NewModuleRegistry()
	records := captureLogs(t, func() {
		out, err := reg.Dispatch("ghost", "chat", nil)
		if err == nil || !strings.Contains(err.Error(), "unknown module") {
			t.Fatalf("unknown module err = %v, out = %+v", err, out)
		}
	})
	if !logContainsMsg(records, "派发未知模块") {
		t.Fatalf("未记录派发未知模块日志, got %v", logMsgs(records))
	}
}
