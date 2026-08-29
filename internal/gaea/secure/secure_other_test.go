//go:build !windows

package secure

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain 将密钥文件重定向到临时目录：测试绝不读写真实用户配置目录，
// 且密钥路径可注入（keyPathOverride）而非写死。
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gaea-secure-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, "创建临时目录失败:", err)
		os.Exit(1)
	}
	keyPathOverride = filepath.Join(dir, "secure.key")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// TestAESRoundTrip 加密→解密往返；密文带 dpapi:aes: 前缀且不含明文。
func TestAESRoundTrip(t *testing.T) {
	orig := "sk-aes-roundtrip-9876543210"
	enc, err := EncryptString(orig)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if !strings.HasPrefix(enc, prefix+aesMarker) {
		t.Fatalf("密文应带 %q 外层前缀与 %q 密文标记, got %q", prefix, aesMarker, enc)
	}
	if strings.Contains(enc, orig) {
		t.Fatalf("密文不应包含明文: %q", enc)
	}
	dec, err := DecryptString(enc)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if dec != orig {
		t.Fatalf("round trip = %q, want %q", dec, orig)
	}
}

// TestKeyFilePerm0600 首次加密自动生成密钥文件，权限必须为 0600。
func TestKeyFilePerm0600(t *testing.T) {
	if _, err := EncryptString("perm-probe"); err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	fi, err := os.Stat(keyPathOverride)
	if err != nil {
		t.Fatalf("密钥文件不存在: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0600 {
		t.Fatalf("密钥文件权限 = %v, want -rw------- (0600)", got)
	}
}

// TestWrongKeyFails 密钥文件被更换后，旧密文必须解密失败（显式报错，不解出垃圾）。
func TestWrongKeyFails(t *testing.T) {
	enc, err := EncryptString("sk-wrong-key-probe")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	other := make([]byte, keySize)
	if _, err := rand.Read(other); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	if err := os.WriteFile(keyPathOverride, other, 0600); err != nil {
		t.Fatalf("覆盖密钥文件: %v", err)
	}
	if _, err := DecryptString(enc); err == nil {
		t.Fatal("更换密钥后解密旧密文应报错")
	}
}

// TestTamperedCiphertextFails 篡改密文（认证标签）后解密必须报错。
func TestTamperedCiphertextFails(t *testing.T) {
	enc, err := EncryptString("sk-tamper-probe")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	b64 := strings.TrimPrefix(enc, prefix)[len(aesMarker):]
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 解码: %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // 篡改认证标签末字节
	tampered := prefix + aesMarker + base64.StdEncoding.EncodeToString(raw)
	if _, err := DecryptString(tampered); err == nil {
		t.Fatal("篡改密文解密应报错")
	}
}

// TestLegacyDpapiPlaintextCompat 历史"恒等实现"落盘的 "dpapi:" + 明文值，
// 必须按明文兼容读出（迁移兼容），不报错。
func TestLegacyDpapiPlaintextCompat(t *testing.T) {
	dec, err := DecryptString("dpapi:sk-legacy-plain-secret")
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if dec != "sk-legacy-plain-secret" {
		t.Fatalf("dec = %q, want %q", dec, "sk-legacy-plain-secret")
	}
}
