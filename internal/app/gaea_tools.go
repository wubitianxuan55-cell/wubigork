package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// imageGenTool 生图工具：让 gaea 智能体像 Codex 一样自行生成图片。
// 复用模型中心的图片后端（xAI / Herdsman / ComfyUI），
// 结果保存到工作区 .gaea/uploads/ 并返回文件路径。
type imageGenTool struct {
	a *App
	// mediaRoot 媒体根注入缝（2026-11-15 众神殿首刀媒体分档，pan_1794754879290）：
	// nil=gaeaCwd() 办公现状逐字节；图文引擎（wenxueBaseTools）注入 wenxueWorkspace
	// ——图文会话生成图落小说目录 .gaea/uploads/，不再写错书。
	mediaRoot func() string
}

// root 落盘根解析（nil 缝=办公 gaeaCwd 现状）。
func (t imageGenTool) root() string {
	if t.mediaRoot != nil {
		return t.mediaRoot()
	}
	return gaeaCwd()
}

func (t imageGenTool) Name() string { return "image_gen" }

func (t imageGenTool) Description() string {
	return "生成图片：根据文字描述生成一张或多张图片，保存为 PNG/JPG 到工作区 .gaea/uploads/ 并返回文件路径。支持指定尺寸、数量、随机种子和 LoRA。云端生成通常几十秒到数分钟。"
}

func (t imageGenTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "prompt":{"type":"string","description":"图片内容描述（英文效果通常更好）"},
  "negative":{"type":"string","description":"可选：负面提示词，描述不希望出现的内容"},
  "size":{"type":"string","description":"可选：尺寸，如 1024x1024 / 1024x1792（仅 ComfyUI 后端支持）"},
  "model":{"type":"string","description":"可选：图片模型名，默认使用模型中心配置"},
  "n":{"type":"integer","description":"可选：生成数量（默认1）","minimum":1,"maximum":4},
  "seed":{"type":"integer","description":"可选：随机种子，相同种子+提示词可复现结果"},
  "lora":{"type":"string","description":"可选：LoRA 文件名（逗号分隔多个）"}
},
"required":["prompt"]
}`)
}

func (t imageGenTool) ReadOnly() bool { return false }

func (t imageGenTool) CompactDescription() string {
	return "生成图片并保存到工作区.gaea/uploads/,返回文件路径"
}

func (t imageGenTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string"},"negative":{"type":"string"},"size":{"type":"string"},"model":{"type":"string"},"n":{"type":"integer"},"seed":{"type":"integer"}},"required":["prompt"]}`)
}

func (t imageGenTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Prompt   string `json:"prompt"`
		Negative string `json:"negative"`
		Size     string `json:"size"`
		Model    string `json:"model"`
		N        int    `json:"n"`
		Seed     int    `json:"seed"`
		Lora     string `json:"lora"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return "", fmt.Errorf("prompt 不能为空")
	}
	if p.N <= 0 {
		p.N = 1
	}
	if p.N > 4 {
		p.N = 4
	}
	if p.Seed == 0 {
		p.Seed = int(time.Now().UnixNano() % 1000000)
	}
	model := t.a.cfg.ImageModel
	if p.Model != "" {
		model = p.Model
	}
	req := &ai.ImageGenerationRequest{
		Model: model,
		// S1.5-B play 内容护栏：image_safe_mode 提交前注入提示词安全段
		// （后端无 NSFW 透传字段，按后端能力缺省关；未配置 = 提示词原样）。
		Prompt:   applyImageSafeMode(p.Prompt, playGuardrails().ImageSafeMode),
		Negative: p.Negative,
		N:        p.N,
		Size:     p.Size,
		Seed:     p.Seed,
		Lora:     p.Lora,
	}
	// 非 ComfyUI 后端不接受 size 参数（xAI 返回 400）。
	// 3.0 Step 3d #5：后端判定引用注册表 kind 常量（ai.ImageBackendKindComfyUI），
	// 不再出现裸字符串硬编码；cfg.ImageBackend 仍由配置驱动。
	if t.a.cfg.ImageBackend != ai.ImageBackendKindComfyUI {
		req.Size = ""
	}
	// 生图卡（2026-11-20 用户拍板：办公/文学板块生图+前端状态卡）：wire-only
	// 事件驱动（与 OCR 大卡同通道 a.emit，不进账本不消费 seq），前端按 id
	// upsert 一张卡：running（数十秒到分钟级）→ done（文件清单）/failed。
	genID := fmt.Sprintf("imggen-%d", time.Now().UnixNano())
	emitGen := func(stage string, files []string, errMsg string) {
		promptClip := p.Prompt
		if r := []rune(promptClip); len(r) > 120 {
			promptClip = string(r[:120]) + "…"
		}
		t.a.emit("gaea-event", map[string]interface{}{
			"kind": "imagegen_progress",
			"imagegen": map[string]interface{}{
				"id": genID, "prompt": promptClip, "model": model, "n": p.N,
				"stage": stage, "files": files, "err": errMsg,
			},
		})
	}
	emitGen("running", nil, "")
	resp, err := t.a.clientRef().GenerateImage(ctx, req)
	if err != nil {
		emitGen("failed", nil, err.Error())
		return "", fmt.Errorf("生图失败: %w", err)
	}
	if len(resp.Data) == 0 {
		emitGen("failed", nil, "API 返回空结果")
		return "", fmt.Errorf("生图失败：API 返回空结果")
	}

	var out []string
	var files []string
	for i, d := range resp.Data {
		dataURL := d.URL
		if dataURL == "" {
			dataURL = d.B64JSON
		}
		if dataURL == "" {
			out = append(out, fmt.Sprintf("[%d] 无图片数据", i+1))
			continue
		}
		rel, err := saveGenImage(t.root(), dataURL)
		if err != nil {
			out = append(out, fmt.Sprintf("[%d] 保存失败: %v", i+1, err))
			continue
		}
		files = append(files, rel)
		extra := ""
		if d.RevisedPrompt != "" {
			extra = "（模型润色提示词）"
		}
		out = append(out, fmt.Sprintf("[%d] [%s](%s)%s", i+1, filepath.Base(rel), rel, extra))
	}
	emitGen("done", files, "")
	return "生图完成，文件：\n" + strings.Join(out, "\n"), nil
}

// saveGenImage 把图片 data URL 保存到工作区 .gaea/uploads/，
// 返回相对工作区的路径（如 .gaea/uploads/gen-xxx.png）。
func saveGenImage(baseDir, dataURL string) (string, error) {
	mime := "image/png"
	payload := dataURL
	if strings.HasPrefix(dataURL, "data:") {
		comma := strings.Index(dataURL, ",")
		if comma < 0 {
			return "", fmt.Errorf("无效的 data URL")
		}
		header := dataURL[:comma]
		mime = strings.TrimSuffix(strings.TrimPrefix(header, "data:"), ";base64")
		payload = dataURL[comma+1:]
	}
	ext := ".png"
	switch {
	case strings.Contains(mime, "jpeg"), strings.Contains(mime, "jpg"):
		ext = ".jpg"
	case strings.Contains(mime, "webp"):
		ext = ".webp"
	case strings.Contains(mime, "gif"):
		ext = ".gif"
	}
	b, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("图片 base64 解码失败: %w", err)
	}
	dir := filepath.Join(baseDir, ".gaea", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("gen-%d%s", time.Now().UnixNano(), ext)
	rel := filepath.ToSlash(filepath.Join(".gaea", "uploads", name))
	if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

var _ tool.Tool = imageGenTool{}
