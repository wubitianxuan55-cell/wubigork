package ocr

import (
	"os"
	"strings"
	"testing"
)

// TestLiveMinerU 真实调用本机 Herdsman /v1/documents/parse。
// 仅在 HERDSMAN_LIVE=1 时运行。
func TestLiveMinerU(t *testing.T) {
	if os.Getenv("HERDSMAN_LIVE") != "1" {
		t.Skip("HERDSMAN_LIVE=1 时运行真实 Herdsman 文档解析验证")
	}
	img := os.Getenv("HERDSMAN_OCR_IMAGE")
	if img == "" {
		img = `C:\AI\wubigrok\.tmp\herdsman_ocr_test.png`
	}
	client := New("http://localhost:8080/v1", DefaultOCRModel)
	res, err := client.ParseDocument(ParseOptions{
		Model:  DefaultParseModel,
		Path:   img,
		Mode:   "pipeline",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if !strings.Contains(res.Text, "Herdsman OCR") {
		t.Errorf("解析文本未包含预期内容: %q", res.Text)
	}
	t.Logf("MinerU text=%q elapsed_ms=%d", res.Text, res.Metadata.ElapsedMS)
}
