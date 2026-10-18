package modelengine

import (
	"testing"
	"time"
)

// TestEngineActivity_TouchAndRead 记账读写闭环：Touch 后可读回；显式时间
// 原样落账；从未记账返回零值；空 engineID 忽略。
func TestEngineActivity_TouchAndRead(t *testing.T) {
	m := NewManager("", "")
	if !m.EngineLastActivity("tianxuan").IsZero() {
		t.Error("从未记账应返回零值")
	}

	m.TouchEngine("tianxuan")
	if m.EngineLastActivity("tianxuan").IsZero() {
		t.Error("TouchEngine 后应非零")
	}

	// 显式时间原样落账（测试回拨能力=收割器判定可测的前提）
	past := time.Now().Add(-11 * time.Minute)
	m.TouchEngineAt("strata", past)
	if got := m.EngineLastActivity("strata"); !got.Equal(past) {
		t.Errorf("EngineLastActivity(strata) = %v, want %v", got, past)
	}

	// 空引擎 id 忽略（不 panic 不落账）
	m.TouchEngine("")
	if !m.EngineLastActivity("").IsZero() {
		t.Error("空 id 不应落账")
	}
}

// TestEngineActivity_BuildChatURLTouches 每次成功解析 chat 端点 = 一次真实
// 使用（空闲自动退出的记账点）；引擎不存在/未启用的错误路径不记账。
func TestEngineActivity_BuildChatURLTouches(t *testing.T) {
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "tianxuan", BaseURL: "http://127.0.0.1:8094/v1", Enabled: true})
	m.SaveEngine(EngineConfig{ID: "offline-eng", BaseURL: "http://127.0.0.1:1/v1", Enabled: false})

	if _, _, err := m.BuildChatURL("tianxuan"); err != nil {
		t.Fatalf("BuildChatURL: %v", err)
	}
	if m.EngineLastActivity("tianxuan").IsZero() {
		t.Error("成功解析端点应记账")
	}

	// 错误路径不记账：不存在 / 未启用
	if _, _, err := m.BuildChatURL("no-such"); err == nil {
		t.Error("不存在引擎应报错")
	}
	if _, _, err := m.BuildChatURL("offline-eng"); err == nil {
		t.Error("未启用引擎应报错")
	}
	if !m.EngineLastActivity("no-such").IsZero() || !m.EngineLastActivity("offline-eng").IsZero() {
		t.Error("错误路径不应记账")
	}
}
