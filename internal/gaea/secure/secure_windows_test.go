//go:build windows

package secure

import "testing"

func TestInvalidBlob(t *testing.T) {
	if _, err := DecryptString("dpapi:not-base64-!!!"); err == nil {
		t.Fatal("期望无效密文报错")
	}
}
