// Package secure 提供本机级密钥保护：Windows 使用当前用户 DPAPI 加密，
// 其他平台回退为原样存储以保持可用性。
package secure

import "strings"

// 加密值前缀，用于区分 DPAPI 密文与旧版明文（迁移兼容）。
const prefix = "dpapi:"

// EncryptString 加密字符串：Windows 返回 "dpapi:" + base64(DPAPI blob)；
// 非 Windows 或加密失败时返回原值并携带错误，调用方按需处理。
func EncryptString(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	enc, err := encryptPlatform(s)
	if err != nil {
		return s, err
	}
	return prefix + enc, nil
}

// DecryptString 解密字符串：无 "dpapi:" 前缀（旧版明文）原样返回，
// 保证历史配置无缝迁移；解密失败返回错误。
func DecryptString(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if !strings.HasPrefix(s, prefix) {
		return s, nil
	}
	return decryptPlatform(strings.TrimPrefix(s, prefix))
}
