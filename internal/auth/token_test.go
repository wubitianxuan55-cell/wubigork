package auth

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
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
