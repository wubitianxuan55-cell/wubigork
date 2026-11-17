package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/docmd"
)

// TestOCRProgressPayloadShape 钩子转发形状（OCR 大卡前端契约）：
// docmd OCRProgressEvent → payload 单对象带 path/name/engine/stage/done/
// total/chars；name=basename 免前端再算。
func TestOCRProgressPayloadShape(t *testing.T) {
	payload := ocrProgressPayload(docmd.OCRProgressEvent{
		Path: "C:/x/扫描件.pdf", Engine: "uocr2", Stage: docmd.OCRStageRunning,
		Done: 3, Total: 20,
	})
	if payload["kind"] != "ocr_progress" {
		t.Fatalf("kind = %v, want ocr_progress", payload["kind"])
	}
	ocr, ok := payload["ocr"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload 缺 ocr 对象: %v", payload)
	}
	if ocr["path"] != "C:/x/扫描件.pdf" || ocr["stage"] != "running" {
		t.Fatalf("ocr 字段错: %v", ocr)
	}
	if ocr["name"] != "扫描件.pdf" {
		t.Fatalf("name 应为 basename，got %v", ocr["name"])
	}
	if ocr["done"] != 3 || ocr["total"] != 20 {
		t.Fatalf("页进度错: %v", ocr)
	}
	// JSON 序列化守卫：前端契约按 json 消费，字段必须可序列化且键语义稳定
	b, err := json.Marshal(ocr)
	if err != nil || !strings.Contains(string(b), `"stage":"running"`) {
		t.Fatalf("ocr payload 不可序列化或键缺失: %s %v", string(b), err)
	}
}
