//go:build !windows

package secure

// encryptPlatform 非 Windows 平台暂不加密（原样返回，保持跨平台可用）。
func encryptPlatform(s string) (string, error) { return s, nil }

// decryptPlatform 非 Windows 平台原样返回。
func decryptPlatform(s string) (string, error) { return s, nil }
