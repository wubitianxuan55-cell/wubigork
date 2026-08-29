//go:build !windows

package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// aesMarker 非 Windows 平台真实加密的密文标记：AES-256-GCM 密文 payload 以
// "aes:" 开头（完整落盘值为 "dpapi:aes:" + base64(nonce|密文|认证标签)）。
// 外层 "dpapi:" 前缀由 secure.go 统一添加——auth/token.go、assistant/manager.go、
// app 等调用方据此识别密文并做旧明文迁移，本包不得更换外层前缀。
// 历史"恒等实现"写入的 "dpapi:" 值内容为明文且不含 "aes:" 标记，据此区分新旧格式。
const aesMarker = "aes:"

const (
	keyFileName = "secure.key" // 密钥文件名（位于用户配置目录 gaea/ 下）
	keySize     = 32           // AES-256 密钥长度（字节）
)

// keyPathOverride 测试注入的密钥文件路径（非空时优先于默认位置），避免测试写死
// 或触碰真实用户配置目录。
var keyPathOverride string

// keyPath 返回密钥文件路径：测试注入路径优先；默认与 internal/gaea/config 的目录
// 约定一致（os.UserConfigDir()/gaea/ 下），不引入对 config 包的依赖。
func keyPath() (string, error) {
	if keyPathOverride != "" {
		return keyPathOverride, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("解析用户配置目录失败: %w", err)
	}
	return filepath.Join(dir, "gaea", keyFileName), nil
}

// loadKey 读取（不存在则生成）本机 AES-256 密钥，密钥文件权限 0600。
// 每次加解密都重新读取：密钥文件丢失或被更换后，旧密文立即显式报错而不是解出垃圾。
func loadKey() ([]byte, error) {
	p, err := keyPath()
	if err != nil {
		return nil, err
	}
	if k, err := os.ReadFile(p); err == nil {
		if len(k) != keySize {
			return nil, fmt.Errorf("密钥文件 %s 长度 %d，期望 %d（密钥损坏）", p, len(k), keySize)
		}
		return k, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取密钥文件失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return nil, fmt.Errorf("创建密钥目录失败: %w", err)
	}
	k := make([]byte, keySize)
	if _, err := rand.Read(k); err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}
	// O_EXCL：并发/多进程下绝不覆盖已有密钥（覆盖会使旧密文永久不可解）。
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			// 另一进程刚创建了密钥：直接使用它。
			if k2, rerr := os.ReadFile(p); rerr == nil && len(k2) == keySize {
				return k2, nil
			}
		}
		return nil, fmt.Errorf("写入密钥文件失败: %w", err)
	}
	if _, err := f.Write(k); err != nil {
		f.Close()
		return nil, fmt.Errorf("写入密钥文件失败: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("写入密钥文件失败: %w", err)
	}
	return k, nil
}

// newGCM 装载本机密钥并构建 AES-256-GCM AEAD。
func newGCM() (cipher.AEAD, error) {
	key, err := loadKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("初始化 AES 失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化 GCM 失败: %w", err)
	}
	return gcm, nil
}

// encryptPlatform 使用本机密钥文件 AES-256-GCM 加密，
// 返回 "aes:" + base64(nonce|密文|认证标签)，由 secure.go 统一加 "dpapi:" 外层前缀。
func encryptPlatform(plain string) (string, error) {
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	ct := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return aesMarker + base64.StdEncoding.EncodeToString(ct), nil
}

// decryptPlatform 解密带 "aes:" 标记的 AES-256-GCM 密文（认证失败/密钥不匹配/
// 格式损坏一律显式报错）；无 "aes:" 标记的 payload 为旧恒等实现写入的明文，
// 原样返回（迁移兼容——下次加密时由调用方按常规落盘写回 "aes:" 密文）。
func decryptPlatform(payload string) (string, error) {
	if !strings.HasPrefix(payload, aesMarker) {
		return payload, nil
	}
	raw, err := base64.StdEncoding.DecodeString(payload[len(aesMarker):])
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %w", err)
	}
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("密文过短（%d 字节）", len(raw))
	}
	ns := gcm.NonceSize()
	pt, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("解密失败（密钥不匹配或密文被篡改）: %w", err)
	}
	return string(pt), nil
}
