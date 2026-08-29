// Package secure 提供本机级密钥保护：Windows 使用当前用户 DPAPI 加密，
// 其他平台使用 AES-256-GCM + 本机密钥文件（0600）加密，两者格式兼容互通
// （统一 "dpapi:" 外层前缀，平台差异封装在 encryptPlatform/decryptPlatform 内）。
package secure

import "strings"

// 加密值前缀：所有平台统一对外使用，调用方（auth/token.go、assistant/manager.go
// 等）据此识别密文并与旧版明文（迁移兼容）区分。
// Windows 密文为 base64(DPAPI blob)；非 Windows 密文为 "aes:" + base64（见
// secure_other.go）。旧恒等实现写入的 "dpapi:" + 明文值按明文兼容读。
const prefix = "dpapi:"

// EncryptString 加密字符串：返回 "dpapi:" + 平台密文（Windows DPAPI blob /
// 非 Windows AES-256-GCM）；加密失败时返回原值并携带错误，调用方按需处理。
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
