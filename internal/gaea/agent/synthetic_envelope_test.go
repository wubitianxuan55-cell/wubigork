package agent

import (
	"strings"
	"testing"
)

// Z3/Z4 蒸馏刀：合成消息来源分面——父代理直穿与用户插话异面，防同意/
// 防洗白条款只挂父面；B5 白名单须认得新前缀（回扫真实轮起点不误判）。
func TestParentSteerEnvelope(t *testing.T) {
	msg := midTurnParentSteerMessage("先核对文件")
	if !strings.HasPrefix(msg, ParentSteerPrefix) || !strings.Contains(msg, "先核对文件") {
		t.Fatalf("父面信封内容不符: %q", msg)
	}
	if !isSyntheticUserTurn(msg) {
		t.Fatal("父面 steer 必须在合成轮白名单内（否则 A4 回扫误判真实轮起点）")
	}
	if !isSyntheticUserTurn(midTurnSteerMessage("用户的话")) {
		t.Fatal("用户面 steer 白名单回归")
	}
}

// 防同意尾注：恢复/门注入类合成消息带「host 生成非用户输入」尾注，且
// 前缀保持不变（B5 白名单按前缀匹配，尾注只追加不得改头）。
func TestSyntheticNoConsentNote(t *testing.T) {
	for _, m := range []string{streamRecoveryMessage(true), streamRecoveryMessage(false), finalReadinessRetryMessage("missing")} {
		if !strings.Contains(m, syntheticNoConsentNote) {
			t.Fatalf("合成消息缺防同意尾注: %.80s", m)
		}
	}
	if strings.HasPrefix(streamRecoveryMessage(true), syntheticNoConsentNote) {
		t.Fatal("尾注不得改写消息头部（白名单前缀锚定）")
	}
}

// 来源分面队列：用户与父代理插话同队列异来源，FIFO 语义不变。
func TestSteerOriginsSeparate(t *testing.T) {
	a := newSteerRunner(t)
	a.Steer("用户直发")
	a.SteerParent("父代理直穿")
	first, ok := a.consumeSteer()
	if !ok || first.origin != steerFromUser || first.text != "用户直发" {
		t.Fatalf("第一条应为用户面: %+v", first)
	}
	second, ok := a.consumeSteer()
	if !ok || second.origin != steerFromParent || second.text != "父代理直穿" {
		t.Fatalf("第二条应为父面: %+v", second)
	}
}
