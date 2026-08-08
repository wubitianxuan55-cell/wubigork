package vision

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeTestPNG 生成一张 2x2 红色 PNG 测试图。
func writeTestPNG(t *testing.T, dir string) string {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, "test.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	img.Pix[3], img.Pix[7] = 0, 255 // 红色
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

// TestRecognizeImageAt 验证请求体包含 base64 图片并正确解析返回文本。
func TestRecognizeImageAt(t *testing.T) {
	path := writeTestPNG(t, t.TempDir())
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s, want /chat/completions", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"图中是一个红色圆点"}}]}`))
	}))
	defer srv.Close()

	text, err := RecognizeImageAt(context.Background(), srv.URL, "qwen-test", path, "这是什么", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if text != "图中是一个红色圆点" {
		t.Errorf("text = %q", text)
	}
	messages := gotBody["messages"].([]interface{})
	content := messages[0].(map[string]interface{})["content"].([]interface{})
	img := content[1].(map[string]interface{})["image_url"].(map[string]interface{})["url"].(string)
	if !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Errorf("image_url = %.40s..., want data:image/png", img)
	}
	if _, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(img, "data:image/png;base64,")); err != nil {
		t.Errorf("base64 解码失败: %v", err)
	}
}

// TestRecognizeImageAt_Error 服务端错误应返回明确错误。
func TestRecognizeImageAt_Error(t *testing.T) {
	path := writeTestPNG(t, t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer srv.Close()
	if _, err := RecognizeImageAt(context.Background(), srv.URL, "m", path, "p", 5*time.Second); err == nil {
		t.Fatal("期望错误，实际成功")
	}
}
