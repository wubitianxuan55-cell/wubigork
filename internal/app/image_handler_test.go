package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// fakeImageBackend 实现 ai.ImageBackend + Interrupt/ResetCancel，用于取消/保存路径测试。
type fakeImageBackend struct {
	interrupts int
	resets     int
	result     *ai.ImageGenerationResponse
	err        error
}

func (f *fakeImageBackend) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeImageBackend) Interrupt(ctx context.Context) error { f.interrupts++; return nil }
func (f *fakeImageBackend) ResetCancel()                        { f.resets++ }

func pngDataURLApp(s string) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(s))
}

func TestGenerateFreeImage_sizeCleanup(t *testing.T) {
	tests := []struct {
		backendType string
		expectSize  bool
	}{
		{"comfyui", true},
		{"xai", false},
		{"herdsman", true},
		{"ollama", false},
		{"glm", true},
	}

	for _, tt := range tests {
		t.Run(tt.backendType, func(t *testing.T) {
			req := &ai.ImageGenerationRequest{
				Model:    "test-model",
				Prompt:   "test prompt",
				Negative: "bad quality",
				N:        1,
				Size:     "1024x1024",
				Seed:     42,
			}

			// 模拟 image_handler.go 中的清理逻辑
			if tt.backendType != "comfyui" && tt.backendType != "herdsman" && tt.backendType != "glm" {
				req.Size = ""
			}

			body, _ := json.Marshal(req)
			jsonStr := string(body)

			if tt.expectSize && !strings.Contains(jsonStr, `"size"`) {
				t.Errorf("%s 应包含 size，实际: %s", tt.backendType, jsonStr)
			}
			if !tt.expectSize && strings.Contains(jsonStr, `"size"`) {
				t.Errorf("%s 不应包含 size，实际: %s", tt.backendType, jsonStr)
			}
		})
	}
}

// TestFindPython_StandaloneEnv 验证 standalone-env 优先于系统 python（ROCm PyTorch 必需）
func TestFindPython_StandaloneEnv(t *testing.T) {
	root := t.TempDir()
	comfyPath := filepath.Join(root, "ComfyUI")
	os.MkdirAll(filepath.Join(comfyPath), 0o755)
	os.MkdirAll(filepath.Join(root, "standalone-env"), 0o755)

	// 模拟 standalone-env python 存在
	os.WriteFile(filepath.Join(root, "standalone-env", "python.exe"), []byte("x"), 0o755)

	// 配置路径为空 → 应自动找到 standalone-env
	got := findPython(comfyPath, "")
	want := filepath.Join(root, "standalone-env", "python.exe")
	if got != want {
		t.Errorf("findPython = %q, want %q（standalone-env 应优先，系统 Python 是 CPU-only）", got, want)
	}

	// 显式配置优先于自动查找
	explicit := filepath.Join(root, "my-python", "python.exe")
	os.MkdirAll(filepath.Join(root, "my-python"), 0o755)
	os.WriteFile(explicit, []byte("x"), 0o755)
	if got := findPython(comfyPath, explicit); got != explicit {
		t.Errorf("显式配置优先 = %q, want %q", got, explicit)
	}
}

// TestGetImageBackendInfo_FullConfig 验证模型中心恢复表单所需的完整配置字段。
func TestGetImageBackendInfo_FullConfig(t *testing.T) {
	ms := &mediaState{
		core: &core{
			cfg: &config.Config{
				ImageBackend:      "comfyui",
				ImageModel:        "z-image-turbo",
				ComfyUIURL:        "http://127.0.0.1:8188",
				ImageSaveDir:      `D:\pics`,
				ComfyUIPath:       `C:\ComfyUI`,
				ComfyUIPythonPath: `C:\ComfyUI\python.exe`,
			},
		},
	}
	got := ms.GetImageBackendInfo()
	if got["model"] != "z-image-turbo" || got["image_model"] != "z-image-turbo" {
		t.Fatalf("model/image_model = %q/%q, want z-image-turbo", got["model"], got["image_model"])
	}
	if got["comfyui_url"] != "http://127.0.0.1:8188" {
		t.Fatalf("comfyui_url = %q", got["comfyui_url"])
	}
	if got["image_save_dir"] != `D:\pics` {
		t.Fatalf("image_save_dir = %q", got["image_save_dir"])
	}
	if got["comfyui_path"] != `C:\ComfyUI` {
		t.Fatalf("comfyui_path = %q", got["comfyui_path"])
	}
	if got["comfyui_python_path"] != `C:\ComfyUI\python.exe` {
		t.Fatalf("comfyui_python_path = %q", got["comfyui_python_path"])
	}
}

// ── T6-4.1 取消真实生效 ────────────────────────────────────────

// TestCancelImageGeneration_Idempotent 重复取消不报错、第二次返回 false（幂等）。
func TestCancelImageGeneration_Idempotent(t *testing.T) {
	ms := &mediaState{core: &core{cfg: &config.Config{}}}
	ms.beginImageGen(context.Background())

	if !ms.CancelImageGeneration() {
		t.Fatal("首次取消应返回 true")
	}
	if ms.CancelImageGeneration() {
		t.Fatal("重复取消应返回 false（幂等）")
	}
	if ms.imageGenRunning {
		t.Fatal("取消后 imageGenRunning 应为 false")
	}
}

// TestCancelImageGeneration_InterruptsComfyUI 取消时应对 ComfyUI 后端调用 /interrupt。
func TestCancelImageGeneration_InterruptsComfyUI(t *testing.T) {
	fake := &fakeImageBackend{}
	c := &ai.Client{}
	c.SetImageBackend(fake, "comfyui")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui"}, client: c}}
	ms.beginImageGen(context.Background())

	if !ms.CancelImageGeneration() {
		t.Fatal("取消应返回 true")
	}
	if fake.interrupts != 1 {
		t.Fatalf("Interrupt 调用次数 = %d, want 1", fake.interrupts)
	}
}

// TestCancelImageGeneration_NoClient 客户端未初始化时取消不 panic（守护空指针）。
func TestCancelImageGeneration_NoClient(t *testing.T) {
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui"}}}
	ms.beginImageGen(context.Background())
	if !ms.CancelImageGeneration() {
		t.Fatal("取消应返回 true（即使无客户端也应安全）")
	}
}

// ── T6-4.3 历史图片可恢复（FilePath 落历史元数据）───────────────

// TestGenerateMedia_PersistsFilePath 生成流程把保存路径写入 imageItem.FilePath。
func TestGenerateMedia_PersistsFilePath(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-media"), Kind: "image"}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, "comfyui")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui", ImageSaveDir: dir}, client: c}}

	res, err := ms.GenerateMedia("{\"prompt\":\"测试\",\"mode\":\"txt2img\",\"count\":1}")
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if errMsg, _ := res["error"].(string); errMsg != "" {
		t.Fatalf("GenerateMedia 返回错误: %s", errMsg)
	}
	results := res["results"].([]imageItem)
	if len(results) != 1 {
		t.Fatalf("结果数 = %d, want 1", len(results))
	}
	if results[0].FilePath == "" {
		t.Fatal("FilePath 未写入历史元数据（T6-4.3）")
	}
	if _, err := os.Stat(results[0].FilePath); err != nil {
		t.Fatalf("FilePath 指向的文件不存在: %v", err)
	}
	if fake.resets != 1 {
		t.Fatalf("ResetCancel 调用 = %d, want 1（新一轮生成清除取消标记）", fake.resets)
	}
}

// TestGenerateFreeImage_PersistsFilePath GenerateFreeImage 同样写入 FilePath。
func TestGenerateFreeImage_PersistsFilePath(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-free")}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, "comfyui")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui", ImageSaveDir: dir}, client: c}}

	res, err := ms.GenerateFreeImage("测试 prompt", "", "512x512", "", "krea2", 42, 1, "")
	if err != nil {
		t.Fatalf("GenerateFreeImage: %v", err)
	}
	if errMsg, _ := res["error"].(string); errMsg != "" {
		t.Fatalf("GenerateFreeImage 返回错误: %s", errMsg)
	}
	images := res["images"].([]imageItem)
	if len(images) != 1 {
		t.Fatalf("结果数 = %d, want 1", len(images))
	}
	if images[0].FilePath == "" {
		t.Fatal("FilePath 未写入（T6-4.3）")
	}
	if _, err := os.Stat(images[0].FilePath); err != nil {
		t.Fatalf("FilePath 指向的文件不存在: %v", err)
	}
}

// TestImageItem_FilePathJSON imageItem 序列化包含 file_path 字段（前端历史据此恢复）。
func TestImageItem_FilePathJSON(t *testing.T) {
	item := imageItem{Image: pngDataURLApp("x"), FilePath: "C:\\pics\\gaea_1.png"}
	b, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["file_path"] != "C:\\pics\\gaea_1.png" {
		t.Fatalf("file_path = %v, want C:\\pics\\gaea_1.png", m["file_path"])
	}
}

// TestSaveImageToDisk_ReturnsPath saveImageToDisk 返回真实可读路径。
func TestSaveImageToDisk_ReturnsPath(t *testing.T) {
	dir := t.TempDir()
	ms := &mediaState{core: &core{cfg: &config.Config{ImageSaveDir: dir}}}
	path := ms.saveImageToDisk(pngDataURLApp("fake-save"), "测试")
	if path == "" {
		t.Fatal("saveImageToDisk 返回空路径")
	}
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("路径 %q 不在保存目录 %q 下", path, dir)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取保存文件失败: %v", err)
	}
	if string(data) != "fake-save" {
		t.Fatalf("文件内容 = %q, want fake-save", string(data))
	}
}

// ── T6-4.5 端口注入修复 ─────────────────────────────────────────

func TestParseNetstatPID(t *testing.T) {
	out := "  TCP    0.0.0.0:8188           0.0.0.0:0              LISTENING       12345\r\n" +
		"  TCP    127.0.0.1:8188         127.0.0.1:0            LISTENING       9999\r\n" +
		"  TCP    0.0.0.0:8080           0.0.0.0:0              LISTENING       7777\r\n" +
		"  UDP    0.0.0.0:5353           *:*                                    1000\r\n"
	if got := parseNetstatPID(out, "8188"); got != 12345 {
		t.Errorf("parseNetstatPID(8188) = %d, want 12345", got)
	}
	if got := parseNetstatPID(out, "8080"); got != 7777 {
		t.Errorf("parseNetstatPID(8080) = %d, want 7777", got)
	}
	if got := parseNetstatPID(out, "8189"); got != 0 {
		t.Errorf("parseNetstatPID(8189) = %d, want 0（无匹配端口）", got)
	}

	// 无 LISTENING 状态 → 0
	noListen := "  TCP    0.0.0.0:8188           0.0.0.0:0              TIME_WAIT       55\r\n"
	if got := parseNetstatPID(noListen, "8188"); got != 0 {
		t.Errorf("TIME_WAIT 不应匹配: got %d", got)
	}
}

func TestIsValidPort(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"8188", true}, {"1", true}, {"65535", true},
		{"", false}, {"0", false}, {"65536", false},
		{"abc", false}, {"81a8", false}, {"-1", false},
		{" 8188", false}, {"8188 ", false}, {"8188;rm", false},
		{"00008188", false}, // 长度超 5（注入防护不因前导零放行）
	}
	for _, tc := range cases {
		if got := isValidPort(tc.in); got != tc.want {
			t.Errorf("isValidPort(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestGetImageBackendInfo_Defaults 验证空模型/空保存目录的兜底值。
func TestGetImageBackendInfo_Defaults(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\test`)

	t.Run("xai 空模型默认高质量", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "xai"}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != "grok-imagine-image-quality" {
			t.Fatalf("image_model = %q", got)
		}
	})

	t.Run("comfyui 空模型默认 krea2", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui"}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != "krea2" {
			t.Fatalf("image_model = %q, want krea2", got)
		}
	})

	t.Run("glm 空模型归位官方默认生图模型", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "glm"}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != ai.GLMDefaultImageModel {
			t.Fatalf("image_model = %q, want %s", got, ai.GLMDefaultImageModel)
		}
	})

	t.Run("glm 残留 xAI 模型归位官方默认生图模型", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "glm", ImageModel: "grok-imagine-image-quality"}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != ai.GLMDefaultImageModel {
			t.Fatalf("image_model = %q, want %s（残留模型不应带到 GLM）", got, ai.GLMDefaultImageModel)
		}
	})

	t.Run("空保存目录回退默认路径", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "xai"}}}
		want := filepath.Join(`C:\Users\test`, "Pictures", "gaea")
		if got := ms.GetImageBackendInfo()["image_save_dir"]; got != want {
			t.Fatalf("image_save_dir = %q, want %q", got, want)
		}
	})
}

// TestSetImageBackend_GLM GLM 生图后端接线：未启用/无 Key 诚实报错，
// 就绪时绑定 GLM 后端并落配置（config.Save 走临时 USERPROFILE 隔离）。
func TestSetImageBackend_GLM(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())

	newMS := func() *mediaState {
		mgr := modelengine.NewManager("", "")
		mgr.UpdateGLMKey("zk-key")
		return &mediaState{core: &core{
			cfg:       &config.Config{ImageSaveDir: t.TempDir()},
			client:    &ai.Client{},
			engineMgr: mgr,
		}}
	}

	t.Run("引擎禁用时拒绝", func(t *testing.T) {
		ms := newMS()
		mgr := modelengine.NewManager("", "")
		mgr.SaveEngine(modelengine.EngineConfig{ID: "glm", Enabled: false})
		ms.engineMgr = mgr
		err := ms.SetImageBackend("glm", "", "cogview-4-250304", "", "")
		if err == nil || !strings.Contains(err.Error(), "未启用") {
			t.Fatalf("应报引擎未启用, got %v", err)
		}
	})

	t.Run("无 Key 时拒绝", func(t *testing.T) {
		ms := newMS()
		ms.engineMgr.UpdateGLMKey("")
		err := ms.SetImageBackend("glm", "", "cogview-4-250304", "", "")
		if err == nil || !strings.Contains(err.Error(), "Key 未配置") {
			t.Fatalf("应报 Key 未配置, got %v", err)
		}
	})

	t.Run("就绪时绑定并持久化", func(t *testing.T) {
		ms := newMS()
		if err := ms.SetImageBackend("glm", "", "cogview-4-250304", "", ""); err != nil {
			t.Fatalf("SetImageBackend: %v", err)
		}
		if ms.cfg.ImageBackend != "glm" || ms.cfg.ImageModel != "cogview-4-250304" {
			t.Fatalf("配置未落地: backend=%s model=%s", ms.cfg.ImageBackend, ms.cfg.ImageModel)
		}
	})

	t.Run("未知后端报错列出 glm", func(t *testing.T) {
		ms := newMS()
		err := ms.SetImageBackend("deepseek", "", "", "", "")
		if err == nil || !strings.Contains(err.Error(), "glm") {
			t.Fatalf("错误提示应列出 glm, got %v", err)
		}
	})
}
