# Z-Image-Turbo Integration for wubigork

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 wubigork 的文生图功能新增 Z-Image-Turbo 模型支持（ComfyUI 后端），与现有 Flux 工作流并存，用户可在 Flux 和 Z-Image-Turbo 之间切换。

**Architecture:** 通过 ComfyUI-ZImagePowerNodes 自定义节点调用 Z-Image-Turbo GGUF 量化模型。Go 后端在 `ComfyUIBackend` 中新增 `buildZImageWorkflow` 方法，根据配置选择工作流。前端新增模型选择下拉框。8GB VRAM 使用 Q5_K_M GGUF（~5.2GB）。

**Tech Stack:** Python (ComfyUI custom nodes), Go (wubigork backend), TypeScript/React (frontend), GGUF quantization

## Global Constraints

- 运行环境：Windows, RTX 4070 Laptop 8GB VRAM, PyTorch 2.7.1+cu118
- Git 可用（github.com git clone），但 HTTP raw 被墙
- ComfyUI 已安装于 `D:/AI/ComfyUI/`
- wubigork 已有 ComfyUI Flux 工作流（`internal/ai/image_comfyui.go`）
- 模型从 ModelScope（modelscope.cn）或 hf-mirror.com 下载
- 保持 Flux 工作流不变，新增 Z-Image-Turbo 路径
- 前端不引入新 npm 包

---

### Task 1: Install ComfyUI-ZImagePowerNodes

**Files:**
- Create: `D:/AI/ComfyUI/custom_nodes/ComfyUI-ZImagePowerNodes/`（git clone）

**Interfaces:**
- Consumes: ComfyUI at `D:/AI/ComfyUI/`
- Produces: Z-Image custom nodes registered in ComfyUI

- [ ] **Step 1: Clone the custom node repository**

```bash
cd D:/AI/ComfyUI/custom_nodes
git clone https://github.com/martin-rizzo/ComfyUI-ZImagePowerNodes.git
```

Expected: Repository cloned successfully. Verify with `ls D:/AI/ComfyUI/custom_nodes/ComfyUI-ZImagePowerNodes/`.

- [ ] **Step 2: Install Python dependencies for the nodes**

```bash
cd D:/AI/ComfyUI/custom_nodes/ComfyUI-ZImagePowerNodes
pip install -r requirements.txt 2>&1 || echo "No requirements.txt, checking pyproject.toml"
# If pyproject.toml exists:
pip install -e . 2>&1
```

Expected: Dependencies installed without error.

- [ ] **Step 3: Restart ComfyUI and verify nodes load**

Start ComfyUI and check the console output for `ZImagePowerNodes` or similar import messages. Alternatively, check with a quick Python import test:

```bash
python -c "import sys; sys.path.insert(0,'D:/AI/ComfyUI'); from custom_nodes.ComfyUI_ZImagePowerNodes import NODE_CLASS_MAPPINGS; print(list(NODE_CLASS_MAPPINGS.keys())[:5])" 2>&1
```

Expected: List of node class names (e.g., `ZSamplerTurbo`, `ZImageLoader`, etc.) printed.

---

### Task 2: Download Z-Image-Turbo GGUF Model from ModelScope

**Files:**
- Download: Z-Image-Turbo Q5_K_M GGUF → `D:/AI/ComfyUI/models/unet/`

**Interfaces:**
- Consumes: ModelScope API / hf-mirror.com
- Produces: `D:/AI/ComfyUI/models/unet/z-image-turbo-Q5_K_M.gguf`

- [ ] **Step 1: Install modelscope CLI**

```bash
pip install modelscope
```

- [ ] **Step 2: Search for Z-Image-Turbo GGUF on ModelScope**

```bash
python -c "from modelscope import snapshot_download; print('modelscope installed')"
```

Then search via: https://modelscope.cn/models?q=Z-Image-Turbo

Alternative if ModelScope doesn't have GGUF: use hf-mirror.com:

```bash
# Set HF mirror
export HF_ENDPOINT=https://hf-mirror.com
pip install huggingface_hub
```

- [ ] **Step 3: Download the model (run interactively)**

```bash
# Option A: ModelScope (if available)
python -c "
from modelscope import snapshot_download
snapshot_download('Tongyi-MAI/Z-Image-Turbo', cache_dir='D:/AI/ComfyUI/models/unet/')
"

# Option B: HF Mirror for GGUF variant
# Look for GGUF variants at https://hf-mirror.com search "Z-Image-Turbo-GGUF"
# Common uploaders: city96, mradermacher, bartowski
python -c "
from huggingface_hub import snapshot_download
import os
os.environ['HF_ENDPOINT'] = 'https://hf-mirror.com'
snapshot_download('city96/Z-Image-Turbo-GGUF', local_dir='D:/AI/ComfyUI/models/unet/', allow_patterns='*Q5_K_M*')
"
```

- [ ] **Step 4: Download CLIP/VAE models if needed**

Z-Image-Turbo uses Qwen2.5 VL text encoder and a standard VAE. Check if ComfyUI-ZImagePowerNodes bundles them or if they need separate download.

```bash
ls D:/AI/ComfyUI/models/clip/ | grep -i qwen
ls D:/AI/ComfyUI/models/vae/
```

If missing:

```bash
# Text encoder from HF mirror
python -c "
from huggingface_hub import snapshot_download
import os
os.environ['HF_ENDPOINT'] = 'https://hf-mirror.com'
snapshot_download('Qwen/Qwen2.5-VL-7B-Instruct', local_dir='D:/AI/ComfyUI/models/text_encoders/qwen2.5-vl-7b/', allow_patterns='*.safetensors')
"
```

- [ ] **Step 5: Verify model files exist**

```bash
ls -la D:/AI/ComfyUI/models/unet/*Z-Image* D:/AI/ComfyUI/models/unet/*z-image* 2>/dev/null
ls -la D:/AI/ComfyUI/models/unet/*GGUF/ 2>/dev/null
```

Expected: At least one `.gguf` or `.safetensors` file for Z-Image-Turbo found.

---

### Task 3: Test Z-Image-Turbo in ComfyUI Manually

**Files:**
- Create: `D:/AI/ComfyUI/user/default/workflows/z-image-test.json`（测试工作流）

**Interfaces:**
- Consumes: ZImagePowerNodes, downloaded model
- Produces: Verified working Z-Image-Turbo generation

- [ ] **Step 1: Create a minimal test workflow JSON**

Identify the exact node types from ZImagePowerNodes by checking `__init__.py`:

```bash
grep -r "class " D:/AI/ComfyUI/custom_nodes/ComfyUI-ZImagePowerNodes/ | grep -v __pycache__ | head -20
```

- [ ] **Step 2: Build and queue a test workflow via ComfyUI API**

Use Python to submit a test workflow to ComfyUI:

```bash
python -c "
import json, requests, time

# Simple workflow using ZImagePowerNodes
# Node IDs will be determined after inspecting the node types
# This is a placeholder — adjust after Step 1
url = 'http://127.0.0.1:8188'

# First check if ComfyUI is running
try:
    r = requests.get(f'{url}/system_stats', timeout=5)
    print('ComfyUI running:', r.status_code)
except:
    print('ComfyUI not running! Start it first.')
    exit(1)

# Submit a test generation
# (exact workflow depends on node types from Step 1)
print('Ready for test workflow submission')
"
```

- [ ] **Step 3: Verify image output**

Check that an image was generated in `D:/AI/ComfyUI/output/` with `wubigork` prefix.

---

### Task 4: Add Z-Image-Turbo Workflow to wubigork Go Backend

**Files:**
- Modify: `D:/AI/wubigork/internal/ai/image_comfyui.go`

**Interfaces:**
- Consumes: `ImageGenerationRequest.Model` field (existing but unused by ComfyUIBackend)
- Produces: `buildZImageWorkflow(prompt string, width, height, seed, steps int) map[string]interface{}`

- [ ] **Step 1: Add `buildZImageWorkflow` method to `ComfyUIBackend`**

在 `image_comfyui.go` 的 `buildFluxWorkflow` 方法之后（第 168 行之后），添加新方法。具体工作流节点取决于 Task 3 中确认的 ZImagePowerNodes 节点类型。以下为预期结构（使用 ZSamplerTurbo + ZImageLoader）：

```go
// buildZImageWorkflow 构建 Z-Image-Turbo GGUF 工作流 JSON
// 使用 ComfyUI-ZImagePowerNodes 自定义节点
// Z-Image-Turbo: 8 步, CFG=0, S3-DiT 单流架构
func (b *ComfyUIBackend) buildZImageWorkflow(prompt string, width, height, seed, steps int) map[string]interface{} {
	// 确保 steps 为 8（Turbo 蒸馏模型标准步数）
	if steps <= 0 || steps > 50 {
		steps = 8
	}

	return map[string]interface{}{
		// ZImageLoader — 加载 Z-Image-Turbo UNet
		"4": map[string]interface{}{
			"class_type": "ZImageLoader",
			"inputs": map[string]interface{}{
				"unet_name": "z-image-turbo-Q5_K_M.gguf",
			},
		},
		// CLIPLoader — 加载 Qwen2.5 VL text encoder
		"5": map[string]interface{}{
			"class_type": "CLIPLoader",
			"inputs": map[string]interface{}{
				"clip_name": "qwen2.5-vl-7b-instruct.safetensors",
				"type":      "qwen_image",
			},
		},
		// VAELoader
		"6": map[string]interface{}{
			"class_type": "VAELoader",
			"inputs": map[string]interface{}{
				"vae_name": "ae.safetensors",
			},
		},
		// CLIPTextEncode — positive
		"7": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs": map[string]interface{}{
				"text": prompt,
				"clip": []interface{}{"5", 0},
			},
		},
		// CLIPTextEncode — negative (Z-Image Turbo: CFG=0, 空字符串)
		"8": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs": map[string]interface{}{
				"text": "",
				"clip": []interface{}{"5", 0},
			},
		},
		// EmptyLatentImage
		"9": map[string]interface{}{
			"class_type": "EmptyLatentImage",
			"inputs": map[string]interface{}{
				"width":      width,
				"height":     height,
				"batch_size": 1,
			},
		},
		// ZSamplerTurbo — Z-Image 专用采样器（替代 KSampler）
		"10": map[string]interface{}{
			"class_type": "ZSamplerTurbo",
			"inputs": map[string]interface{}{
				"seed":         seed,
				"steps":        steps,
				"cfg":          0.0, // Z-Image Turbo: CFG=0
				"sampler_name": "euler",
				"scheduler":    "simple",
				"denoise":      1.0,
				"model":        []interface{}{"4", 0},
				"positive":     []interface{}{"7", 0},
				"negative":     []interface{}{"8", 0},
				"latent_image": []interface{}{"9", 0},
			},
		},
		// VAEDecode
		"11": map[string]interface{}{
			"class_type": "VAEDecode",
			"inputs": map[string]interface{}{
				"samples": []interface{}{"10", 0},
				"vae":     []interface{}{"6", 0},
			},
		},
		// SaveImage — 输出节点
		"12": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs": map[string]interface{}{
				"filename_prefix": "wubigork",
				"images":          []interface{}{"11", 0},
			},
		},
	}
}
```

**注意：** 上述节点类型名称（`ZImageLoader`, `ZSamplerTurbo`）是预期值。Task 3 完成后，如果实际节点名不同，需据此调整。

- [ ] **Step 2: Modify `GenerateImage` to dispatch based on `req.Model`**

修改 `image_comfyui.go` 第 46 行，根据 `req.Model` 选择工作流：

```go
// 第 44-46 行，替换现有的:
	steps := 20

	workflow := b.buildFluxWorkflow(req.Prompt, width, height, seed, steps)

// 改为:
	var workflow map[string]interface{}
	switch req.Model {
	case "z-image-turbo":
		steps := 8
		workflow = b.buildZImageWorkflow(req.Prompt, width, height, seed, steps)
	default: // "flux" 或空
		steps := 20
		workflow = b.buildFluxWorkflow(req.Prompt, width, height, seed, steps)
	}
```

- [ ] **Step 3: Update the `ComfyUIBackend` comment**

将第 17 行注释改为：

```go
// ComfyUIBackend 通过 ComfyUI REST API 调用本地 Flux / Z-Image-Turbo 模型
```

- [ ] **Step 4: Verify compilation**

```bash
cd D:/AI/wubigork
go build ./internal/ai/...
```

Expected: `go build` succeeds with no errors.

---

### Task 5: Update wubigork Config & Image Handler for Model Selection

**Files:**
- Modify: `D:/AI/wubigork/internal/config/config.go`
- Modify: `D:/AI/wubigork/internal/app/image_handler.go`
- Modify: `D:/AI/wubigork/internal/app/app.go`

**Interfaces:**
- Produces: `Config.ImageModel` field ("flux" | "z-image-turbo")
- Consumes in handler: passes model to `ImageGenerationRequest.Model`

- [ ] **Step 1: Add `ImageModel` to config**

在 `config.go` 的 `configFile` struct（第 27 行之后）添加新字段：

```go
	ImageModel         string  `json:"image_model,omitempty"`   // "flux" (默认) | "z-image-turbo"
```

在 `Config` struct（第 77 行之后）添加：

```go
	ImageModel string // ComfyUI 模型选择: "flux" (默认) | "z-image-turbo"
```

在 `Load()` 函数默认值区域（第 117 行之后）添加：

```go
		ImageModel: "flux",
```

在 config 文件覆盖区域（第 241 行附近）添加读取：

```go
			if cf.ImageModel != "" {
				cfg.ImageModel = cf.ImageModel
			}
```

在 `Save()` 函数 switch（第 338 行附近）添加 case：

```go
	case "image_model":
		cf.ImageModel = value
```

- [ ] **Step 2: Update `GenerateFreeImage` in image_handler.go**

修改 `image_handler.go` 第 31-36 行，将硬编码的 `"flux"` 替换为配置值：

```go
	imgReq := &ai.ImageGenerationRequest{
		Model:  a.cfg.ImageModel, // 来自配置，flux 或 z-image-turbo
		Prompt: fullPrompt,
		N:      1,
		Size:   size,
	}
```

- [ ] **Step 3: Update `SetImageBackend` to also accept model parameter**

修改 `image_handler.go` 的 `SetImageBackend` 函数签名（第 112 行），增加 model 参数：

```go
func (a *App) SetImageBackend(backend string, comfyUIURL string, imageModel string) error {
	if a.client == nil {
		return fmt.Errorf("AI 客户端未初始化")
	}
	switch backend {
	case "comfyui":
		a.cfg.ImageBackend = "comfyui"
		if comfyUIURL != "" {
			a.cfg.ComfyUIURL = comfyUIURL
		}
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
		a.client.SetImageBackend(ai.NewComfyUIBackend(a.cfg.ComfyUIURL))
	case "xai":
		a.cfg.ImageBackend = "xai"
		a.client.SetImageBackend(nil)
	default:
		return fmt.Errorf("不支持的后端: %s（支持 xai / comfyui）", backend)
	}
	return nil
}
```

- [ ] **Step 4: Update `GetImageBackend` to also return model**

修改 `image_handler.go` 第 103-108 行：

```go
// GetImageBackendInfo 获取当前图片后端类型和模型（供前端显示）
func (a *App) GetImageBackendInfo() map[string]string {
	result := map[string]string{
		"backend": "xai",
		"model":   "flux",
	}
	if a.client != nil {
		result["backend"] = a.client.GetImageBackendType()
	}
	result["model"] = a.cfg.ImageModel
	return result
}
```

保留旧 `GetImageBackend()` 方法作为兼容（或直接在下面声明新的委托给它）：

```go
func (a *App) GetImageBackend() string {
	return a.GetImageBackendInfo()["backend"]
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd D:/AI/wubigork
go build ./...
```

Expected: Build succeeds.

---

### Task 6: Update Frontend — Model Selector in ImageGenPage

**Files:**
- Modify: `frontend/src/pages/ImageGenPage.tsx`

**Interfaces:**
- Consumes: `window.go.app.App.GetImageBackendInfo()`, `window.go.app.App.GenerateFreeImage()`
- Produces: Model selector dropdown visible when ComfyUI backend is active

- [ ] **Step 1: Add model state and load from backend**

In `ImageGenPage.tsx`, after line 37 (`const [backend, setBackend] = useState('xai')`), add:

```tsx
  const [imageModel, setImageModel] = useState('flux')
```

In the `useEffect` (lines 42-53), update to also load model:

```tsx
  React.useEffect(() => {
    (async () => {
      try {
        // @ts-ignore
        const info = await window.go.app.App.GetImageBackendInfo()
        if (info?.backend) setBackend(info.backend)
        if (info?.model) setImageModel(info.model)
      } catch (_) {}
    })()
  }, [])
```

- [ ] **Step 2: Add model selector UI**

After the size selector (around line 209, before the style selector), add:

```tsx
              {backend === 'comfyui' && (
                <div style={{ width: isMobile ? '100%' : undefined }}>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>
                    模型
                  </Typography.Text>
                  <Select
                    value={imageModel}
                    onChange={setImageModel}
                    style={{ width: isMobile ? '100%' : 160 }}
                    options={[
                      { label: '🌊 Flux Dev (20步)', value: 'flux' },
                      { label: '⚡ Z-Image-Turbo (8步)', value: 'z-image-turbo' },
                    ]}
                  />
                </div>
              )}
```

- [ ] **Step 3: Update generate call to pass model**

The `GenerateFreeImage` currently takes `(prompt, size, style)`. We need to also pass the model. Since the Wails binding signature is defined in Go, we need to either:
- Add `model` parameter to `GenerateFreeImage` Go function, or
- Use a separate `SetImageModel` call + read from config

The simpler approach: add a new Wails binding `SetImageModel(model string)` and use `GetImageBackendInfo` to read it. But for a cleaner UX, update `GenerateFreeImage` to accept a 4th parameter.

**Option A (推荐): Update GenerateFreeImage signature** — 修改 Go 函数签名增加 model 参数。

**Option B: Separate SetImageModel call** — 生成前先调用设置。

选择 **Option A**:

在 `handleGenerate` (line 55-79), update the call:

```tsx
      // @ts-ignore
      const res = await window.go.app.App.GenerateFreeImage(prompt.trim(), size, style, imageModel)
```

- [ ] **Step 4: Update backend label display**

Line 108-110, update Tag display:

```tsx
          <Tag color={backend === 'comfyui' ? 'green' : 'blue'} style={{ borderRadius: 'var(--radius-md)' }}>
            {backend === 'comfyui' ? <><HomeOutlined /> 本地 {imageModel === 'z-image-turbo' ? 'Z-Image-Turbo' : 'Flux'}</> : <><CloudOutlined /> xAI 云端</>}
          </Tag>
```

- [ ] **Step 5: Update waiting text**

Line 279:

```tsx
                {backend === 'comfyui'
                  ? (imageModel === 'z-image-turbo'
                    ? '本地 Z-Image-Turbo 正在绘制中，通常需要 15-40 秒...'
                    : '本地 Flux 正在绘制中，通常需要 40-90 秒...')
                  : 'xAI 正在生成图片...'}
```

- [ ] **Step 6: Update Go handler to accept model parameter**

Modify `image_handler.go` `GenerateFreeImage` signature:

```go
func (a *App) GenerateFreeImage(prompt string, size string, style string, model string) (map[string]interface{}, error) {
```

And use `model` parameter when building the request (override config default if model is provided):

```go
	imgModel := a.cfg.ImageModel
	if model != "" {
		imgModel = model
	}
	imgReq := &ai.ImageGenerationRequest{
		Model:  imgModel,
		Prompt: fullPrompt,
		N:      1,
		Size:   size,
	}
```

---

### Task 7: Update SettingsPage for Model Configuration

**Files:**
- Modify: `frontend/src/pages/SettingsPage.tsx`

**Interfaces:**
- Consumes: `window.go.app.App.SetImageBackend(backend, url, model)`, `GetImageBackendInfo()`
- Produces: Model dropdown in settings

- [ ] **Step 1: Add imageModel state**

Near the other image states (around line 270-280 in SettingsPage), add:

```tsx
  const [imageModel, setImageModel] = useState('flux')
```

- [ ] **Step 2: Load imageModel on mount**

In the settings page's load effect, add:

```tsx
        // @ts-ignore
        const info = await window.go.app.App.GetImageBackendInfo()
        if (info?.model) setImageModel(info.model)
```

- [ ] **Step 3: Add model selector in image backend card**

After the ComfyUI service address input (around line 371) and before the storage dir input, when `imageBackend === 'comfyui'`:

```tsx
          {imageBackend === 'comfyui' && (
            <>
              <div>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
                  ComfyUI 服务地址
                </Typography.Text>
                <Input
                  placeholder='http://127.0.0.1:8188'
                  value={comfyUIURL}
                  onChange={(e) => setComfyUIURL(e.target.value)}
                  style={{ background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)' }}
                />
              </div>
              <div>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
                  生成模型
                </Typography.Text>
                <Select
                  value={imageModel}
                  onChange={(val: string) => setImageModel(val)}
                  style={{ width: 200 }}
                  options={[
                    { label: '🌊 Flux Dev (20步，高质量)', value: 'flux' },
                    { label: '⚡ Z-Image-Turbo (8步，极速)', value: 'z-image-turbo' },
                  ]}
                />
              </div>
            </>
          )}
```

- [ ] **Step 4: Update handleSaveImageBackend to pass model**

Update the save handler to call with the model parameter:

```tsx
      // @ts-ignore
      await window.go.app.App.SetImageBackend(imageBackend, comfyUIURL, imageModel)
```

And update the existing `handleSaveImageBackend` function call to also save the model config:

```tsx
      await window.go.app.App.SetConfig('image_model', imageModel)
```

- [ ] **Step 5: Update backend dropdown labels**

Line 354-357 options:

```tsx
              options={[
                { label: '☁️ xAI 云端 (Grok Imagine)', value: 'xai' },
                { label: '🏠 ComfyUI 本地 (Flux / Z-Image)', value: 'comfyui' },
              ]}
```

---

### Task 8: Build & Integration Test

**Files:**
- No code changes — verification only

**Interfaces:**
- Consumes: All previous tasks
- Produces: Working end-to-end generation

- [ ] **Step 1: Build wubigork**

```bash
cd D:/AI/wubigork
go build -o build/bin/wubigork.exe .
```

Expected: Build succeeds with no errors.

- [ ] **Step 2: Build frontend**

```bash
cd D:/AI/wubigork/frontend
npm run build
```

Expected: Frontend builds without errors.

- [ ] **Step 3: Start ComfyUI and test end-to-end**

```bash
# Terminal 1: Start ComfyUI
cd D:/AI/ComfyUI
python main.py

# Terminal 2: Start wubigork
D:/AI/wubigork/build/bin/wubigork.exe
```

- [ ] **Step 4: Manual test checklist**

1. 打开 wubigork → 设置 → 选择 "ComfyUI 本地 (Flux / Z-Image)" → 模型选择 "Z-Image-Turbo" → 点击切换后端
2. 进入 AI 绘梦 → 确认模型标签显示 "本地 Z-Image-Turbo"
3. 输入 prompt → 选择尺寸 → 确认模型选择器显示 "Z-Image-Turbo (8步)"
4. 点击生成 → 等待 15-40 秒 → 确认图片生成成功
5. 切换回 Flux 模型 → 再次生成 → 确认 Flux 仍然正常工作

- [ ] **Step 5: Commit**

```bash
cd D:/AI/wubigork
git add -A
git commit -m "feat: add Z-Image-Turbo support alongside Flux in ComfyUI backend"
```
