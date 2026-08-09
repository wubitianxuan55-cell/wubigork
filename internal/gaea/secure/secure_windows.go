//go:build windows

package secure

import (
	"encoding/base64"
	"fmt"
	"syscall"
	"unsafe"
)

// CryptProtectData 的 CRYPTPROTECT_UI_FORBIDDEN 标志：禁止弹出任何 UI。
const cryptProtectUIForbidden = 0x1

var (
	crypt32            = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtect   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotect = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

// encryptPlatform 使用当前 Windows 用户凭据加密数据（无需口令，仅本用户/本机可解）。
func encryptPlatform(plain string) (string, error) {
	in := []byte(plain)
	inBlob := dataBlob{cbData: uint32(len(in))}
	if len(in) > 0 {
		inBlob.pbData = &in[0]
	}
	var outBlob dataBlob
	r1, _, err := procCryptProtect.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, // szDataDescr
		0, // pOptionalEntropy
		0, // pvReserved
		0, // pPromptStruct
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r1 == 0 {
		return "", fmt.Errorf("CryptProtectData: %v", err)
	}
	defer syscall.LocalFree((syscall.Handle)(unsafe.Pointer(outBlob.pbData)))
	return base64.StdEncoding.EncodeToString(unsafe.Slice(outBlob.pbData, outBlob.cbData)), nil
}

// decryptPlatform 解密 DPAPI 密文（base64 编码的 blob）。
func decryptPlatform(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %w", err)
	}
	inBlob := dataBlob{cbData: uint32(len(raw))}
	if len(raw) > 0 {
		inBlob.pbData = &raw[0]
	}
	var outBlob dataBlob
	r1, _, err := procCryptUnprotect.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, // szDataDescr
		0, // pOptionalEntropy
		0, // pvReserved
		0, // pPromptStruct
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r1 == 0 {
		return "", fmt.Errorf("CryptUnprotectData: %v", err)
	}
	defer syscall.LocalFree((syscall.Handle)(unsafe.Pointer(outBlob.pbData)))
	return string(unsafe.Slice(outBlob.pbData, outBlob.cbData)), nil
}
