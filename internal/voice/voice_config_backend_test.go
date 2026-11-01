// voice_config_backend_test.go — Voice V2 档位缺省与校验（M4 产品接入刀）。
package voice

import "testing"

func TestVoiceBackendDefaultAndValidate(t *testing.T) {
	c := DefaultVoiceConfig()
	if c.VoiceBackend != "v1" {
		t.Fatalf("缺省应 v1，got %q", c.VoiceBackend)
	}
	c.VoiceBackend = "v2"
	c.Validate()
	if c.VoiceBackend != "v2" {
		t.Fatalf("v2 应保留，got %q", c.VoiceBackend)
	}
	c.VoiceBackend = "" // 空 → 回 v1（flag 缺省安全）
	c.Validate()
	if c.VoiceBackend != "v1" {
		t.Fatalf("空值应回 v1，got %q", c.VoiceBackend)
	}
	c.VoiceBackend = "v3" // 未知 → 回 v1
	c.Validate()
	if c.VoiceBackend != "v1" {
		t.Fatalf("未知值应回 v1，got %q", c.VoiceBackend)
	}
}
