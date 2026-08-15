package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
)

// TestImageGenToolMeta 校验生图工具的元信息与 Schema。
func TestImageGenToolMeta(t *testing.T) {
	tool := imageGenTool{}
	if tool.Name() != "image_gen" || strings.TrimSpace(tool.Description()) == "" {
		t.Fatalf("工具元信息异常: %s", tool.Name())
	}
	if !json.Valid(tool.Schema()) || !json.Valid(tool.CompactSchema()) {
		t.Fatal("Schema 非法")
	}
	if tool.ReadOnly() {
		t.Error("image_gen 不应为只读工具")
	}
}

// TestSaveGenImage 校验 data URL 落盘（png/jpg/纯 base64）。
func TestSaveGenImage(t *testing.T) {
	t.Chdir(t.TempDir())
	cases := []struct {
		dataURL string
		ext     string
	}{
		{"data:image/png;base64,aGVsbG8=", ".png"},
		{"data:image/jpeg;base64,aGVsbG8=", ".jpg"},
		{"aGVsbG8=", ".png"}, // 无 data: 前缀的裸 base64
	}
	for _, c := range cases {
		rel, err := saveGenImage(".", c.dataURL)
		if err != nil {
			t.Fatalf("saveGenImage(%s): %v", c.dataURL[:20], err)
		}
		if !strings.HasSuffix(rel, c.ext) {
			t.Errorf("扩展名 = %s, want %s", rel, c.ext)
		}
		if _, err := os.Stat(filepath.FromSlash(rel)); err != nil {
			t.Errorf("文件不存在: %v", err)
		}
	}
}

// TestSaveGenImage_BadData 非法 base64 应报错。
func TestSaveGenImage_BadData(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := saveGenImage(".", "data:image/png;base64,!!!"); err == nil {
		t.Fatal("期望解码失败，实际成功")
	}
}

// capturingImageBackend 捕获 GenerateImage 请求，供 image_gen 工具测试断言。
type capturingImageBackend struct {
	req *ai.ImageGenerationRequest
}

func (b *capturingImageBackend) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	b.req = req
	return &ai.ImageGenerationResponse{Data: []ai.ImageData{{B64JSON: "data:image/png;base64,aGVsbG8="}}}, nil
}

// TestImageGenTool_SizeByBackendKind 固化 3.0 Step 3d #5：size 参数只传给
// ComfyUI 后端（kind=ai.ImageBackendKindComfyUI），其余后端一律剥离（xAI 会 400）。
// 判定引用注册表 kind 常量，后端选择由 cfg.ImageBackend 配置驱动。
func TestImageGenTool_SizeByBackendKind(t *testing.T) {
	cases := []struct {
		backend    string
		expectSize bool
	}{
		{ai.ImageBackendKindComfyUI, true},
		{ai.ImageBackendKindOpenAI, false},
		{"xai", false},
		{"herdsman", false},
		{"ollama", false},
	}
	for _, tt := range cases {
		t.Run(tt.backend, func(t *testing.T) {
			t.Chdir(t.TempDir())
			fake := &capturingImageBackend{}
			c := &ai.Client{}
			c.SetImageBackend(fake, tt.backend)
			a := &App{core: &core{
				cfg:    &config.Config{ImageBackend: tt.backend, ImageModel: "test-model"},
				client: c,
			}}
			tool := imageGenTool{a: a}
			_, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"cat","size":"1024x1024"}`))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			hasSize := fake.req != nil && fake.req.Size != ""
			if hasSize != tt.expectSize {
				t.Errorf("backend=%q size 传递 = %v, want %v（req=%+v）", tt.backend, hasSize, tt.expectSize, fake.req)
			}
			if fake.req != nil && fake.req.Prompt != "cat" {
				t.Errorf("prompt = %q, want cat", fake.req.Prompt)
			}
		})
	}
}
