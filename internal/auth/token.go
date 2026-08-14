package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/secure"
)

// securePrefix 是 secure 包加密值的前缀（Windows DPAPI 密文 / 非 Windows 降级值均带此前缀）。
// 与 internal/gaea/secure 的 prefix 保持一致，用于识别旧版明文（迁移兼容）。
// 注：internal/app/encryptSecretIfLegacy 也直接使用该字面量，改动需同步。
const securePrefix = "dpapi:"

// Token 表示 OAuth token 对
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	ObtainedAt   time.Time `json:"obtained_at"`
}

// Validate 检查 token 是否有效
func (t *Token) Validate() error {
	if t == nil {
		return fmt.Errorf("token 为空")
	}
	if t.AccessToken == "" {
		return fmt.Errorf("access_token 为空")
	}
	// RefreshToken 不存在时仍可使用（只是不能刷新）
	return nil
}

// IsExpired 判断 access token 是否已过期
//
// 对齐 hermes-agent 的 XAI_ACCESS_TOKEN_REFRESH_SKEW_SECONDS（3600 秒 = 1 小时）：
// xAI OAuth access token 约 6 小时有效，提前 1 小时刷新可以在
// gateway/cron 等间歇使用场景下避免 credential 过期。
func (t *Token) IsExpired() bool {
	if t == nil || t.AccessToken == "" {
		return true
	}
	if t.ExpiresIn <= 0 {
		return false // 没有过期信息则假设未过期
	}
	expiryTime := t.ObtainedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
	// 提前 1 小时刷新（与 hermes-agent 一致）
	return time.Now().Add(1 * time.Hour).After(expiryTime)
}

// TokenStore token 文件存储
type TokenStore struct {
	mu   sync.RWMutex
	path string
	// legacyPath 旧品牌 token 文件（.wubigork_token.json），主路径不存在时回退读取
	legacyPath string
}

// NewTokenStore 创建 token 存储
func NewTokenStore(path string) *TokenStore {
	// 兼容旧品牌：.gaea_token.json 不存在时回退 .wubigork_token.json（老用户免重新登录）
	legacy := ""
	if filepath.Base(path) == ".gaea_token.json" {
		legacy = filepath.Join(filepath.Dir(path), ".wubigork_token.json")
	}
	return &TokenStore{path: path, legacyPath: legacy}
}

// Save 保存 token 到文件：敏感字段（access_token、refresh_token）经 secure 加密后落盘，
// 非敏感字段（token_type/expires_in/scope/obtained_at）保持明文，JSON 结构不变。
// 加密失败返回错误（不写入明文兜底，杜绝静默降级）。
func (s *TokenStore) Save(token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token == nil {
		return fmt.Errorf("token 为空")
	}
	return s.writeLocked(token)
}

// Load 从文件加载 token：带 "dpapi:" 前缀的字段解密还原；无前缀的旧版明文
// 读取成功并触发一次自动重写为加密（迁移）。解密失败返回明确错误，
// 绝不静默返回 nil token（否则用户会被误判为未登录）。
func (s *TokenStore) Load() (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil && os.IsNotExist(err) && s.legacyPath != "" {
		// 回退旧品牌 token 文件（老用户免重新登录）
		data, err = os.ReadFile(s.legacyPath)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取 token 文件失败: %w", err)
	}
	var disk Token
	if err := json.Unmarshal(data, &disk); err != nil {
		return nil, fmt.Errorf("解析 token 文件失败: %w", err)
	}
	token, migrated, err := decryptToken(&disk)
	if err != nil {
		return nil, err
	}
	if migrated {
		// 旧版明文 → 自动重写为加密存储（一次性迁移；从 legacyPath 读入时同时落到主路径）
		if err := s.writeLocked(token); err != nil {
			return nil, fmt.Errorf("迁移旧 token 为加密存储失败: %w", err)
		}
	}
	return token, nil
}

// Delete 删除 token 文件
func (s *TokenStore) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 token 文件失败: %w", err)
	}
	return nil
}

// writeLocked 将 token（敏感字段加密）以 0600 权限写入 s.path。调用方须持有锁。
func (s *TokenStore) writeLocked(token *Token) error {
	enc, err := encryptToken(token)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(enc, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 token 失败: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("写入 token 文件失败: %w", err)
	}
	return nil
}

// encryptToken 返回敏感字段经 secure.EncryptString 加密后的副本；空字段保持为空。
// 非 Windows 平台 secure 包降级为 "dpapi:" 前缀 + 原值，此处不感知平台差异。
func encryptToken(token *Token) (*Token, error) {
	enc := *token
	var err error
	if enc.AccessToken, err = secure.EncryptString(token.AccessToken); err != nil {
		return nil, fmt.Errorf("加密 access_token 失败: %w", err)
	}
	if enc.RefreshToken, err = secure.EncryptString(token.RefreshToken); err != nil {
		return nil, fmt.Errorf("加密 refresh_token 失败: %w", err)
	}
	return &enc, nil
}

// decryptToken 解密敏感字段并返回还原后的 token 与是否发现旧版明文（需迁移）。
// 解密失败返回明确错误（含字段名），不静默吞错。
func decryptToken(disk *Token) (*Token, bool, error) {
	out := *disk
	migrated := false
	decryptField := func(v, name string, dst *string) error {
		if v == "" {
			return nil
		}
		if !strings.HasPrefix(v, securePrefix) {
			// 旧版明文（无前缀）：读取成功，标记迁移
			*dst = v
			migrated = true
			return nil
		}
		dec, err := secure.DecryptString(v)
		if err != nil {
			return fmt.Errorf("解密 %s 失败: %w", name, err)
		}
		*dst = dec
		return nil
	}
	if err := decryptField(out.AccessToken, "access_token", &out.AccessToken); err != nil {
		return nil, false, err
	}
	if err := decryptField(out.RefreshToken, "refresh_token", &out.RefreshToken); err != nil {
		return nil, false, err
	}
	return &out, migrated, nil
}
