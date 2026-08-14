package assistant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSave_EncryptsWxTokenOnDisk 落盘 JSON 不含明文 wxToken、带 dpapi: 前缀；
// Load 后内存恢复明文（List/调用方回显明文不受影响）。
func TestSave_EncryptsWxTokenOnDisk(t *testing.T) {
	dir := t.TempDir()
	m := NewEmpty(dir)
	ast := Assistant{ID: "a1", Name: "助手A", WxToken: "plain-secret-token", WxUserID: "u1", Enabled: true}
	if err := m.Add(ast); err != nil {
		t.Fatalf("Add: %v", err)
	}

	path := filepath.Join(dir, "assistants.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取落盘文件: %v", err)
	}
	raw := string(data)
	if strings.Contains(raw, "plain-secret-token") {
		t.Fatal("落盘 JSON 不应包含明文 wxToken")
	}
	if !strings.Contains(raw, "dpapi:") {
		t.Fatal("落盘 wxToken 应带 dpapi: 前缀")
	}

	// 结构仍可解析，且 wxToken 字段为密文
	var disk []Assistant
	if err := json.Unmarshal(data, &disk); err != nil {
		t.Fatalf("解析落盘 JSON: %v", err)
	}
	if len(disk) != 1 || !strings.HasPrefix(disk[0].WxToken, "dpapi:") {
		t.Fatalf("落盘 wxToken 应为 dpapi: 前缀密文, got %q", disk[0].WxToken)
	}

	// Load 后内存明文，token 仍可读
	m2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := m2.Get("a1")
	if got == nil || got.WxToken != "plain-secret-token" {
		t.Fatalf("Load 后 token 应恢复明文, got %q", got.WxToken)
	}
}

// TestLoad_LegacyPlaintextMigrates 旧版明文文件 Load 后自动重写为加密，token 仍可读。
func TestLoad_LegacyPlaintextMigrates(t *testing.T) {
	dir := t.TempDir()
	legacy := `[
		{"id":"a1","name":"旧助手","wxToken":"legacy-plain-token","wxUserId":"u1","enabled":true}
	]`
	path := filepath.Join(dir, "assistants.json")
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("写旧版明文文件: %v", err)
	}

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load 旧版明文: %v", err)
	}
	got := m.Get("a1")
	if got == nil || got.WxToken != "legacy-plain-token" {
		t.Fatalf("旧明文 token 应可读, got %q", got.WxToken)
	}

	// 一次性迁移：文件已被重写为加密，不再含明文
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取迁移后文件: %v", err)
	}
	raw := string(data)
	if strings.Contains(raw, "legacy-plain-token") {
		t.Fatal("迁移后文件不应再含明文 wxToken")
	}
	if !strings.Contains(raw, "dpapi:") {
		t.Fatal("迁移后 wxToken 应带 dpapi: 前缀")
	}

	// 再次 Load 仍可读（加密→解密闭环）
	m2, err := Load(dir)
	if err != nil {
		t.Fatalf("二次 Load: %v", err)
	}
	if got := m2.Get("a1"); got == nil || got.WxToken != "legacy-plain-token" {
		t.Fatalf("二次 Load 后 token 应仍可读, got %q", got.WxToken)
	}
}
