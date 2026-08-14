package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/secure"
)

func newTestStore(t *testing.T) (*TokenStore, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_token.json")
	return NewTokenStore(path), path
}

func TestTokenSaveLoad_RoundTrip(t *testing.T) {
	store, path := newTestStore(t)
	token := &Token{
		AccessToken:  "test_access",
		RefreshToken: "test_refresh",
		TokenType:    "Bearer",
		ExpiresIn:    21600,
		Scope:        "openid profile",
		ObtainedAt:   time.Now(),
	}

	if err := store.Save(token); err != nil {
		t.Fatalf("Save 失败: %s", err)
	}

	// 验证文件确实存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Save 后文件不存在")
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load 失败: %s", err)
	}
	if loaded == nil {
		t.Fatal("Load 返回 nil")
	}
	if loaded.AccessToken != "test_access" {
		t.Errorf("AccessToken = %q, 期望 test_access", loaded.AccessToken)
	}
	if loaded.RefreshToken != "test_refresh" {
		t.Errorf("RefreshToken = %q, 期望 test_refresh", loaded.RefreshToken)
	}
	if loaded.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, 期望 Bearer", loaded.TokenType)
	}
	if loaded.Scope != "openid profile" {
		t.Errorf("Scope = %q, 期望 openid profile", loaded.Scope)
	}
	if loaded.ExpiresIn != 21600 {
		t.Errorf("ExpiresIn = %d, 期望 21600", loaded.ExpiresIn)
	}
}

func TestTokenLoad_NonExistentFile(t *testing.T) {
	store := NewTokenStore("/nonexistent/path/token.json")
	token, err := store.Load()
	if err != nil {
		t.Fatalf("Load 不存在的文件应返回 nil, nil, 得到 error: %s", err)
	}
	if token != nil {
		t.Fatal("Load 不存在的文件应返回 nil")
	}
}

func TestTokenDelete(t *testing.T) {
	store, path := newTestStore(t)
	token := &Token{AccessToken: "del_test", ObtainedAt: time.Now()}

	if err := store.Save(token); err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	if err := store.Delete(); err != nil {
		t.Fatalf("Delete 失败: %s", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Delete 后文件仍存在")
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Delete 后 Load 失败: %s", err)
	}
	if loaded != nil {
		t.Fatal("Delete 后 Load 应返回 nil")
	}
}

func TestTokenDelete_NonExistentFile(t *testing.T) {
	store := NewTokenStore("/nonexistent/path/token.json")
	if err := store.Delete(); err != nil {
		t.Fatalf("Delete 不存在的文件应返回 nil, 得到: %s", err)
	}
}

func TestTokenIsExpired_NotExpired(t *testing.T) {
	token := &Token{
		AccessToken: "fresh",
		ExpiresIn:   21600,
		ObtainedAt:  time.Now(),
	}
	if token.IsExpired() {
		t.Error("刚获取的 token 不应过期")
	}
}

func TestTokenIsExpired_Expired(t *testing.T) {
	token := &Token{
		AccessToken: "stale",
		ExpiresIn:   3600,
		ObtainedAt:  time.Now().Add(-2 * time.Hour),
	}
	if !token.IsExpired() {
		t.Error("2小时前获取、1小时过期的 token 应已过期（含1小时提前量）")
	}
}

func TestTokenIsExpired_NilToken(t *testing.T) {
	var token *Token
	if !token.IsExpired() {
		t.Error("nil token 应返回 true")
	}
}

func TestTokenIsExpired_EmptyAccessToken(t *testing.T) {
	token := &Token{AccessToken: "", ExpiresIn: 3600, ObtainedAt: time.Now()}
	if !token.IsExpired() {
		t.Error("空的 access_token 应视为过期")
	}
}

func TestTokenIsExpired_NoExpiryInfo(t *testing.T) {
	token := &Token{
		AccessToken: "no_expiry",
		ExpiresIn:   0,
		ObtainedAt:  time.Now(),
	}
	if token.IsExpired() {
		t.Error("ExpiresIn=0 时应假设未过期")
	}
}

func TestTokenValidate_Valid(t *testing.T) {
	token := &Token{AccessToken: "valid_token"}
	if err := token.Validate(); err != nil {
		t.Errorf("有效 token 应返回 nil, 得到: %s", err)
	}
}

func TestTokenValidate_Nil(t *testing.T) {
	var token *Token
	if err := token.Validate(); err == nil {
		t.Error("nil token 应返回 error")
	}
}

func TestTokenValidate_EmptyAccessToken(t *testing.T) {
	token := &Token{AccessToken: ""}
	if err := token.Validate(); err == nil {
		t.Error("空 access_token 应返回 error")
	}
}

func TestTokenStoreConcurrentAccess(t *testing.T) {
	store, _ := newTestStore(t)
	var wg sync.WaitGroup

	token := &Token{AccessToken: "concurrent", ObtainedAt: time.Now()}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.Save(token)
			_, _ = store.Load()
		}()
	}
	wg.Wait()

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("并发后 Load 失败: %s", err)
	}
	if loaded == nil || loaded.AccessToken != "concurrent" {
		t.Error("并发后 token 应完整")
	}
}

// realEncryptionAvailable 探测 secure 包在当前环境的实际行为：
// true = DPAPI 真实加密（密文不含明文）；false = 非 Windows 降级（"dpapi:" 前缀 + 原值）。
func realEncryptionAvailable(t *testing.T) bool {
	t.Helper()
	probe, err := secure.EncryptString("probe-secret")
	if err != nil {
		return false
	}
	return probe != "dpapi:probe-secret"
}

// TestTokenSave_EncryptsSensitiveFields 落盘文件不得包含明文 refresh_token/access_token
// （DPAPI 可用时真实加密断言；不可用时按 secure 包降级行为断言：带前缀且 Load 可还原）。
func TestTokenSave_EncryptsSensitiveFields(t *testing.T) {
	store, path := newTestStore(t)
	token := &Token{
		AccessToken:  "secret_access_token",
		RefreshToken: "secret_refresh_token",
		TokenType:    "Bearer",
		ExpiresIn:    21600,
		Scope:        "openid profile",
		ObtainedAt:   time.Now(),
	}
	if err := store.Save(token); err != nil {
		t.Fatalf("Save 失败: %s", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取落盘文件失败: %s", err)
	}
	raw := string(data)

	// 敏感字段必须带加密前缀（secure.EncryptString 在所有平台都加 "dpapi:" 前缀）
	if !strings.Contains(raw, securePrefix) {
		t.Fatalf("落盘文件未包含加密前缀 %q, 内容: %s", securePrefix, raw)
	}
	if realEncryptionAvailable(t) {
		// DPAPI 可用：密文中不得出现明文
		if strings.Contains(raw, "secret_access_token") || strings.Contains(raw, "secret_refresh_token") {
			t.Fatalf("落盘文件包含明文敏感字段: %s", raw)
		}
	} else {
		// 降级路径：值为 "dpapi:" + 明文，Load 必须能完整还原（不静默失败）
		loaded, err := store.Load()
		if err != nil {
			t.Fatalf("降级模式下 Load 失败: %s", err)
		}
		if loaded == nil || loaded.AccessToken != "secret_access_token" || loaded.RefreshToken != "secret_refresh_token" {
			t.Fatalf("降级模式 round-trip 失败: %+v", loaded)
		}
	}

	// 非敏感字段保持明文可读（JSON 结构不变）
	if !strings.Contains(raw, `"token_type": "Bearer"`) || !strings.Contains(raw, `"scope": "openid profile"`) {
		t.Fatalf("非敏感字段应保持明文 JSON: %s", raw)
	}
}

// TestTokenLoad_MigratesLegacyPlaintext 旧版明文 token 文件读取成功，并自动重写为加密。
func TestTokenLoad_MigratesLegacyPlaintext(t *testing.T) {
	store, path := newTestStore(t)

	// 构造旧版明文 token 文件（无 "dpapi:" 前缀）
	legacy := &Token{
		AccessToken:  "legacy_access",
		RefreshToken: "legacy_refresh",
		TokenType:    "Bearer",
		ExpiresIn:    21600,
		Scope:        "openid",
		ObtainedAt:   time.Now(),
	}
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("构造旧版明文文件失败: %s", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("写旧版明文文件失败: %s", err)
	}

	// 旧明文必须读取成功（兼容旧版，不静默返回 nil）
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("旧明文 Load 应成功, 得到: %s", err)
	}
	if loaded == nil {
		t.Fatal("旧明文 Load 返回 nil")
	}
	if loaded.AccessToken != "legacy_access" || loaded.RefreshToken != "legacy_refresh" {
		t.Fatalf("旧明文值读取错误: access=%q refresh=%q", loaded.AccessToken, loaded.RefreshToken)
	}

	// 自动迁移：落盘文件已被重写为加密（敏感字段带 "dpapi:" 前缀）
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取迁移后文件失败: %s", err)
	}
	rawStr := string(raw)
	if !strings.Contains(rawStr, securePrefix) {
		t.Fatalf("迁移后文件未包含加密前缀 %q: %s", securePrefix, rawStr)
	}
	if realEncryptionAvailable(t) && strings.Contains(rawStr, "legacy_refresh") {
		t.Fatalf("迁移后文件仍含明文 refresh_token: %s", rawStr)
	}

	// 迁移幂等：再次 Load 不再触发重写（文件内容稳定）
	again, err := store.Load()
	if err != nil {
		t.Fatalf("迁移后再次 Load 失败: %s", err)
	}
	if again == nil || again.AccessToken != "legacy_access" || again.RefreshToken != "legacy_refresh" {
		t.Fatalf("迁移后再次 Load 值错误: %+v", again)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取二次 Load 后文件失败: %s", err)
	}
	if string(after) != rawStr {
		t.Fatal("二次 Load 不应再次重写文件")
	}
}

// TestTokenLoad_DecryptFailureReturnsError 解密失败必须返回明确错误（含字段名），
// 绝不静默返回 nil token。非 Windows 降级模式无法构造解密失败，跳过。
func TestTokenLoad_DecryptFailureReturnsError(t *testing.T) {
	if !realEncryptionAvailable(t) {
		t.Skip("DPAPI 不可用（降级模式原样返回），无法构造解密失败场景")
	}
	store, path := newTestStore(t)
	bad := &Token{AccessToken: "dpapi:!!!not-base64!!!", RefreshToken: "x", ObtainedAt: time.Now()}
	data, err := json.MarshalIndent(bad, "", "  ")
	if err != nil {
		t.Fatalf("构造坏文件失败: %s", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("写坏文件失败: %s", err)
	}

	tok, err := store.Load()
	if err == nil {
		t.Fatalf("解密失败应返回错误, 得到 token: %+v", tok)
	}
	if tok != nil {
		t.Fatal("解密失败不应返回 token")
	}
	if !strings.Contains(err.Error(), "access_token") {
		t.Fatalf("错误信息应指明失败字段, 得到: %s", err)
	}
}
