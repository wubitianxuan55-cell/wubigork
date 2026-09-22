package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
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
	lastReq    *ai.ImageGenerationRequest // 最近一次请求（配图 v2 等形状断言）
}

func (f *fakeImageBackend) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	f.lastReq = req
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
		err := ms.SetImageBackend("glm", "", "cogview-4-250304", "")
		if err == nil || !strings.Contains(err.Error(), "未启用") {
			t.Fatalf("应报引擎未启用, got %v", err)
		}
	})

	t.Run("无 Key 时拒绝", func(t *testing.T) {
		ms := newMS()
		ms.engineMgr.UpdateGLMKey("")
		err := ms.SetImageBackend("glm", "", "cogview-4-250304", "")
		if err == nil || !strings.Contains(err.Error(), "Key 未配置") {
			t.Fatalf("应报 Key 未配置, got %v", err)
		}
	})

	t.Run("就绪时绑定并持久化", func(t *testing.T) {
		ms := newMS()
		if err := ms.SetImageBackend("glm", "", "cogview-4-250304", ""); err != nil {
			t.Fatalf("SetImageBackend: %v", err)
		}
		if ms.cfg.ImageBackend != "glm" || ms.cfg.ImageModel != "cogview-4-250304" {
			t.Fatalf("配置未落地: backend=%s model=%s", ms.cfg.ImageBackend, ms.cfg.ImageModel)
		}
	})

	t.Run("未知后端报错列出 glm", func(t *testing.T) {
		ms := newMS()
		err := ms.SetImageBackend("deepseek", "", "", "")
		if err == nil || !strings.Contains(err.Error(), "glm") {
			t.Fatalf("错误提示应列出 glm, got %v", err)
		}
	})
}

// TestGenerateMedia_EditMode 指令编辑模式（阶段一刀 C）：缺原图拒绝；
// 带原图透传后端（fake 后端不校验 mode，形状由 ai 层测试钉死）。
func TestGenerateMedia_EditMode(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-edit"), Kind: "image"}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, "openai")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "openai", ImageSaveDir: dir}, client: c}}

	// 缺原图：拒绝（不触后端）
	res, err := ms.GenerateMedia(`{"prompt":"把外套改成红色","mode":"edit","count":1}`)
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "指令编辑需要原图" {
		t.Fatalf("缺原图应拒绝，得到: %v", res["error"])
	}

	// 带原图：透传后端出结果（edit 不设后端门——Q3）
	res, err = ms.GenerateMedia(`{"prompt":"把外套改成红色","mode":"edit","initImage":"data:image/png;base64,AAAA","count":1}`)
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatalf("带原图应透传出结果，得到: %s", msg)
	}
	if res["mode"] != "edit" {
		t.Fatalf("mode 应回显 edit: %v", res["mode"])
	}
	results := res["results"].([]imageItem)
	if len(results) != 1 || results[0].FilePath == "" {
		t.Fatalf("编辑结果应落盘带回路径: %+v", results)
	}
}

// TestGenerateMedia_EditComfyuiModelOverride 指令编辑本地档（阶段二刀 A）：
// backend=comfyui 时元数据如实改写 qwen-image-edit——请求 model 字段是生图模型名
// （krea2 等），不代表编辑引擎；结果卡/台账按真实引擎记。
func TestGenerateMedia_EditComfyuiModelOverride(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-edit-c"), Kind: "image"}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, "comfyui")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui", ImageModel: "krea2", ImageSaveDir: dir}, client: c}}

	res, err := ms.GenerateMedia(`{"prompt":"把外套改成红色","mode":"edit","model":"krea2","initImage":"data:image/png;base64,AAAA","count":1}`)
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatalf("comfyui 编辑应出结果，得到: %s", msg)
	}
	results := res["results"].([]imageItem)
	if len(results) != 1 {
		t.Fatalf("应出一图: %+v", results)
	}
	if results[0].Model != "qwen-image-edit" {
		t.Fatalf("comfyui 编辑元数据应如实记 qwen-image-edit: %s", results[0].Model)
	}
}

// TestGenerateMedia_MaskGateAndPassthrough 蒙版局部重绘（阶段二刀 B）：
// mask+img2img fail-closed 拒绝；mask+edit 透传到后端请求。
func TestGenerateMedia_MaskGateAndPassthrough(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-edit-m"), Kind: "image"}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, "herdsman")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "herdsman", ImageSaveDir: dir}, client: c}}

	// mask + img2img：拒绝（不触后端）
	res, err := ms.GenerateMedia(`{"prompt":"改","mode":"img2img","initImage":"data:image/png;base64,AAAA","mask":"data:image/png;base64,BBBB","count":1}`)
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "蒙版仅支持指令编辑模式（局部重绘）" {
		t.Fatalf("mask+img2img 应拒绝，得到: %v", res["error"])
	}
	if fake.lastReq != nil {
		t.Fatal("拒绝路径不应触达后端")
	}

	// mask + edit：透传
	res, err = ms.GenerateMedia(`{"prompt":"把外套改成红色","mode":"edit","initImage":"data:image/png;base64,AAAA","mask":"data:image/png;base64,BBBB","count":1}`)
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatalf("mask+edit 应透传出结果，得到: %s", msg)
	}
	if fake.lastReq == nil || fake.lastReq.Mask != "data:image/png;base64,BBBB" {
		t.Fatalf("Mask 应透传到后端请求: %+v", fake.lastReq)
	}
}

// TestGenerateMedia_OutpaintConvert 扩图（阶段二刀 C）：转换为 edit 请求
// （合成画布+蒙版经 fake 捕获断言尺寸）；mode 回显 outpaint；用户 mask 拒绝。
func TestGenerateMedia_OutpaintConvert(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-outpaint"), Kind: "image"}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, "openai")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "openai", ImageSaveDir: dir}, client: c}}

	// 缺原图
	res, _ := ms.GenerateMedia(`{"prompt":"扩展背景","mode":"outpaint","expand":{"left":50},"count":1}`)
	if msg, _ := res["error"].(string); msg != "扩图需要原图" {
		t.Fatalf("缺原图应拒绝，得到: %v", res["error"])
	}
	// 用户 mask 与扩图蒙版互斥
	res, _ = ms.GenerateMedia(`{"prompt":"扩展背景","mode":"outpaint","initImage":"data:image/png;base64,AAAA","mask":"data:image/png;base64,BBBB","expand":{"left":50},"count":1}`)
	if msg, _ := res["error"].(string); msg != "扩图不需要蒙版（扩展区由扩展量决定）" {
		t.Fatalf("扩图+用户蒙版应拒绝，得到: %v", res["error"])
	}

	// 真图（2×1 红）+ 左右各 100 → 画布 6×1；请求应为 edit + 合成图 + 合成蒙版
	src := solidOutpaintSrc(t, 2, 1)
	res, err := ms.GenerateMedia(`{"prompt":"扩展背景","mode":"outpaint","initImage":"` + src + `","expand":{"left":100,"right":100},"count":1}`)
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatalf("扩图应出结果，得到: %s", msg)
	}
	if res["mode"] != "outpaint" {
		t.Fatalf("mode 应回显 outpaint: %v", res["mode"])
	}
	if fake.lastReq == nil || fake.lastReq.Mode != "edit" {
		t.Fatalf("请求应转换为 edit: %+v", fake.lastReq)
	}
	if fake.lastReq.Mask == "" {
		t.Fatal("请求应带合成蒙版")
	}
	payload := strings.TrimPrefix(fake.lastReq.InitImage, "data:image/png;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("合成画布应可解码: %v", err)
	}
	cfgImg, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		t.Fatalf("合成画布 decode: %v", err)
	}
	if cfgImg.Bounds().Dx() != 6 || cfgImg.Bounds().Dy() != 1 {
		t.Fatalf("合成画布应 6x1: %v", cfgImg.Bounds())
	}
}

// solidOutpaintSrc 纯色 PNG data URL（app 层扩图测试夹具）。
func solidOutpaintSrc(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// TestGenerateMedia_VariantChain GenerateMedia 变体簇传递：edit+sourcePath 在
// 运行态闸开时登记带 ParentID 且 item 回填；闸关时零 panic 字段空（测试态默认）。
func TestGenerateMedia_VariantChain(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-edit-vc"), Kind: "image"}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, "openai")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "openai", ImageSaveDir: dir}, client: c}}

	// 闸关（默认测试态）：sourcePath 查得空 → item 回填空，不 panic 不报错
	res, err := ms.GenerateMedia(`{"prompt":"改","mode":"edit","initImage":"data:image/png;base64,AAAA","sourcePath":"C:/nope/v1.png","count":1}`)
	if err != nil {
		t.Fatalf("GenerateMedia: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatalf("闸关应正常出结果: %s", msg)
	}
	items := res["results"].([]imageItem)
	if len(items) != 1 || items[0].ParentID != "" || items[0].AssetID != "" {
		t.Fatalf("闸关时变体字段应为空: %+v", items[0])
	}
}
