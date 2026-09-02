package app

import (
	"testing"

	"github.com/gaea/gaea/internal/assistant"
)

// TestWhisperAssistantSave_EmptyTokenPreservesCredentials 「启停切换只传 id+enabled」
// 式部分保存不得清空扫码凭据：incoming WxToken/WxUserID 为空串时保留 existing 现值。
// 种子助手直接经 assistantMgr.Add 落库（不走 WhisperAssistantSave，避免 Add 路径
// 拉起微信轮询）；更新用 Enabled=false，保证 Update 后也不会 startAssistantWx
//（Server.Start 对非空 token 会起 pollLoop 打 ilinkai 外网，测试必须离线）。
func TestWhisperAssistantSave_EmptyTokenPreservesCredentials(t *testing.T) {
	a := newChatServiceTestApp(t)
	if err := a.assistantMgr.Add(assistant.Assistant{
		ID: "wx-a1", Name: "峨嵋", PersonalityID: "gaea",
		WxToken: "scan-token-abc", WxUserID: "wxid-123", Enabled: true,
	}); err != nil {
		t.Fatalf("seed Add: %v", err)
	}

	// 部分保存：仅 id + enabled，凭据字段为空串
	if err := a.WhisperAssistantSave(assistant.Assistant{ID: "wx-a1", Enabled: false}); err != nil {
		t.Fatalf("WhisperAssistantSave: %v", err)
	}

	got := a.assistantMgr.Get("wx-a1")
	if got == nil {
		t.Fatal("助手 wx-a1 应存在")
	}
	if got.WxToken != "scan-token-abc" {
		t.Errorf("空 token 保存应保留旧凭据, got %q", got.WxToken)
	}
	if got.WxUserID != "wxid-123" {
		t.Errorf("空 wxUserId 保存应保留旧绑定, got %q", got.WxUserID)
	}
	if got.Enabled {
		t.Error("enabled 应照写为 false")
	}

	// 非空 token 照写（显式换绑场景不受保留逻辑影响）
	if err := a.WhisperAssistantSave(assistant.Assistant{
		ID: "wx-a1", Enabled: true, WxToken: "scan-token-new", WxUserID: "wxid-456",
	}); err != nil {
		t.Fatalf("WhisperAssistantSave(显式新凭据): %v", err)
	}
	got = a.assistantMgr.Get("wx-a1")
	if got == nil || got.WxToken != "scan-token-new" || got.WxUserID != "wxid-456" {
		t.Fatalf("非空凭据应照写, got %+v", got)
	}
}

// TestWhisperChatAsAssistant_InjectsAssistantNameInLock 助手名注入在持锁窗口内：
// whisperChatAsAssistant 回合结束后 orch.AssistantName = 助手名；空助手名不覆盖
// 现值（同人格多助手各回合注入自己的名字，锁外直写的互相覆盖/数据竞争已消除）。
// DataRoot 置空让整个回合全内存化（orch.DataRoot="" 时状态恢复/落库均短路）：
// 正常回合结尾的 persistStateAsync 异步协程会在 TempDir 清理后重开 hermes.db，
// Windows 下文件句柄未释放导致 RemoveAll 失败——本测试只验证锁内名字注入，
// 与持久化无关，不必引入该文件生命周期竞争。
func TestWhisperChatAsAssistant_InjectsAssistantNameInLock(t *testing.T) {
	a := newChatServiceTestApp(t)
	a.whisperState.whisperDataRoot = ""
	pers := testKind("asst-name")
	cleanupWhisperSession(t, pers)

	if _, err := a.whisperChatAsAssistant("你好呀", pers, "峨嵋", false); err != nil {
		t.Fatalf("whisperChatAsAssistant: %v", err)
	}
	orch := a.getOrCreateOrch(pers)
	orch.LockTurn()
	got := orch.AssistantName
	orch.UnlockTurn()
	if got != "峨嵋" {
		t.Fatalf("AssistantName 应注入为峨嵋, got %q", got)
	}

	// 空助手名：不碰字段（保留现值），与 whisper_state 锁外直写的旧行为一致
	if _, err := a.whisperChatAsAssistant("在吗", pers, "", false); err != nil {
		t.Fatalf("whisperChatAsAssistant(空名): %v", err)
	}
	orch.LockTurn()
	got = orch.AssistantName
	orch.UnlockTurn()
	if got != "峨嵋" {
		t.Fatalf("空助手名不应覆盖现值, got %q", got)
	}
}
