# AI 绘梦页面重做 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 AI 绘梦页面从单列布局重做为主题栏工作台，新增负向 Prompt、种子控制、批量生成、进度计时、20个预设模板、会话历史画廊。

**Architecture:** 前端拆分为 6 个组件（PromptPanel / GenControls / GenButton / ResultGallery / Lightbox / HistoryStrip）+ 1 个数据文件（imageTemplates）。Go 后端扩展 GenerateFreeImage 为 7 参数，Flux 工作流支持负向 prompt，Z-Image-Turbo 忽略。

**Tech Stack:** React + TypeScript + Ant Design（零新 npm 包），Go + Wails bindings

## Global Constraints

- 零新 npm 依赖 — 纯 Ant Design 组件
- 桌面端双栏（左 320px 控制面板，右 flex:1 结果区）
- 移动端单栏 + 底部 Drawer 切换（useIsMobile）
- 历史仅在内存（useState），不持久化
- xAI 后端不支持负向 prompt/种子，仅 ComfyUI 支持
- 现有 SettingsPage、风格预设列表、后端切换逻辑不受影响

---

### Task 1: 新建图片模板数据文件

**Files:**
- Create: `frontend/src/data/imageTemplates.ts`

**Interfaces:**
- Produces: `TEMPLATES: Record<string, Template[]>` 和 `Template` interface

```ts
// Template interface
interface Template {
  label: string    // e.g. "📸 写实摄影"
  prompt: string
  negative: string
}

// Grouped by category
const TEMPLATES: Record<string, Template[]> = {
  '📕 创作类': [
    { label: '📕 小说封面', prompt: '精致小说封面设计，专业排版构图，电影级光影，浮雕标题质感，8K超高清', negative: '低质量, 模糊, 文字扭曲, 错别字, 非中文' },
    { label: '👤 角色肖像', prompt: '精致角色肖像，柔和棚拍光线，半身构图，精致的面部细节，8K超清', negative: '变形手指, 多余肢体, 低分辨率, 模糊' },
    { label: '🏔️ 场景概念', prompt: '史诗级场景概念艺术，广阔视野，戏剧性光影，氛围感强，丰富细节', negative: '模糊, 平面感, 简陋, 低质量' },
    { label: '🎨 插画内页', prompt: '精美插画风格，细腻线条，柔和色调，叙事性构图，艺术感', negative: '照片感, 3D渲染感, 低分辨率' },
    { label: '📖 章节插图', prompt: '小说章节插图，与文字搭配的叙事画面，适合排版，留白设计，精致', negative: '混乱, 模糊, 低质量, 过度复杂' },
  ],
  '📸 写实类': [
    { label: '📸 写实摄影', prompt: '写实摄影风格，自然光线，超高分辨率，逼真质感，细节丰富，8K', negative: '模糊, 低质量, 动漫风格, 绘画质感, 3D渲染感' },
    { label: '🎬 电影剧照', prompt: '电影剧照风格，宽银幕构图，戏剧性光影，胶片质感，电影级调色', negative: '模糊, 低质量, 平面感, 电视画质' },
    { label: '👗 时尚大片', prompt: '时尚杂志大片风格，精致妆造，高级灯光，完美构图，奢侈品质感', negative: '模糊, 低质量, 廉价感, 过时服装' },
    { label: '🔍 微距特写', prompt: '微距摄影特写，极致细节，浅景深，柔美散景，清晰焦点，8K超清', negative: '模糊, 大景深, 噪点, 低分辨率' },
    { label: '🚶 纪实街拍', prompt: '纪实街拍风格，自然抓拍感，真实光影，生活气息，人文情怀', negative: '摆拍感, 过度修饰, 滤镜感, 3D渲染' },
  ],
  '🎨 风格类': [
    { label: '🖌️ 中国水墨', prompt: '中国水墨画风格，写意笔触，留白构图，淡雅墨色，传统韵味', negative: '色彩鲜艳, 油画质感, 照片感, 西式构图' },
    { label: '🎪 日系动漫', prompt: '日系动漫风格，鲜艳色彩，精致角色，明亮光影，干净线条', negative: '写实感, 照片感, 暗黑风格, 3D渲染' },
    { label: '🌆 赛博朋克', prompt: '赛博朋克风格，霓虹灯光，雨夜都市，高科技感，蓝紫冷色调', negative: '自然风光, 暖色调, 古代风格, 简陋' },
    { label: '🧙 奇幻风格', prompt: '奇幻艺术风格，魔法氛围，史诗感，精细纹理，丰富想象力', negative: '现代元素, 科技感, 简约风格, 照片感' },
    { label: '🎨 数字油画', prompt: '数字油画风格，电影级光影，高细节，8K，丰富色彩层次', negative: '模糊, 低质量, 线稿, 扁平风格' },
  ],
  '📐 构图类': [
    { label: '🧍 半身肖像', prompt: '精致半身肖像构图，柔光，干净背景，人物突出，8K超清', negative: '全身, 多人, 混乱背景, 低分辨率' },
    { label: '🧎 全身立绘', prompt: '全身角色立绘，完整人物展示，清晰服装设计，白色背景，角色设计图', negative: '半身, 裁切, 模糊面部, 复杂背景' },
    { label: '🌄 广角场景', prompt: '广角场景构图，宏大视野，深远景深，丰富层次，电影级', negative: '特写, 浅景深, 扁平, 低分辨率' },
    { label: '🚁 俯瞰航拍', prompt: '俯瞰航拍视角，上帝视角，全景构图，丰富细节，震撼', negative: '仰视, 平视, 特写, 低清晰度' },
    { label: '👁️ 特写面部', prompt: '面部大特写，极致细节，柔光，眼神表达，8K超清', negative: '全身, 远景, 遮挡面部, 模糊' },
  ],
}
```

- [ ] **Step 1: 创建模板数据文件**

```bash
mkdir -p D:/AI/wubigork/frontend/src/data
```

Write the complete file with the content above.

- [ ] **Step 2: 验证编译**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit src/data/imageTemplates.ts 2>&1
```

Expected: No errors.

---

### Task 2: 扩展 Go 后端 — image_handler.go

**Files:**
- Modify: `internal/app/image_handler.go`

**Interfaces:**
- Produces: `GenerateFreeImage(prompt, negative, size, style, model string, seed int, n int) (map[string]interface{}, error)`
- Returns: `{ "images": [{ "image": "...", "seed": 42, "time": 48.5, "prompt": "...", "model": "z-image-turbo", "size": "1024x1024" }], "error": "" }`

```go
func (a *App) GenerateFreeImage(prompt string, negative string, size string, style string, model string, seed int, n int) (map[string]interface{}, error) {
	if a.client == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}

	fullPrompt := prompt
	if style != "" {
		fullPrompt = prompt + "。风格: " + style
	}
	if size == "" {
		size = "1024x1024"
	}
	if n < 1 || n > 4 {
		n = 1
	}

	type imageItem struct {
		Image  string  `json:"image"`
		Seed   int     `json:"seed"`
		Time   float64 `json:"time"`
		Prompt string  `json:"prompt"`
		Model  string  `json:"model"`
		Size   string  `json:"size"`
	}

	images := make([]imageItem, 0, n)

	for i := 0; i < n; i++ {
		genSeed := seed
		if genSeed == 0 {
			genSeed = int(time.Now().UnixNano()%1000000) + i*777 // 不用 math/rand，避免 seed=0 时重复
		}

		imgModel := a.cfg.ImageModel
		if model != "" {
			imgModel = model
		}

		imgReq := &ai.ImageGenerationRequest{
			Model:    imgModel,
			Prompt:   fullPrompt,
			Negative: negative,
			N:        1,
			Size:     size,
			Seed:     genSeed,
		}

		start := time.Now()
		resp, err := a.client.GenerateImage(a.ctx, imgReq)
		elapsed := time.Since(start).Seconds()

		if err != nil {
			slog.Warn("图片生成失败", "attempt", i+1, "error", err)
			continue
		}
		if len(resp.Data) == 0 {
			continue
		}

		imageData := resp.Data[0].URL
		if imageData == "" {
			imageData = resp.Data[0].B64JSON
		}

		images = append(images, imageItem{
			Image:  imageData,
			Seed:   genSeed,
			Time:   math.Round(elapsed*10) / 10,
			Prompt: fullPrompt,
			Model:  imgModel,
			Size:   size,
		})

		if a.cfg.ImageSaveDir != "" && imageData != "" {
			a.saveImageToDisk(imageData, fullPrompt)
		}
	}

	if len(images) == 0 {
		return map[string]interface{}{"error": "所有生成尝试均失败"}, nil
	}

	return map[string]interface{}{
		"images": images,
	}, nil
}
```

- [ ] **Step 1: 添加 import**

在 import 区增加：
```go
"math"
```

- [ ] **Step 2: 替换 GenerateFreeImage 函数**

将原函数（~60行）整体替换为上述新版本。

- [ ] **Step 3: 验证编译**

```bash
cd D:/AI/wubigork && go build ./... 2>&1
```

Expected: Build succeeds.

---

### Task 3: 扩展 Go 后端 — 负向 Prompt + 种子 + 类型

**Files:**
- Modify: `internal/ai/image_comfyui.go` — Flux 工作流负向 prompt
- Modify: `internal/ai/types.go` — 添加 Negative/Seed 字段

**Step 1: 扩展 ImageGenerationRequest**

在 `internal/ai/types.go` 的 `ImageGenerationRequest` struct（第 73-80 行）中添加字段：

```go
type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Negative       string `json:"negative,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Seed           int    `json:"seed,omitempty"`
}
```

- [ ] **Step 2: Flux 工作流使用负向 prompt**

在 `image_comfyui.go` 的 `buildFluxWorkflow` 中，修改 node 8（负向 CLIPTextEncode）。当前 node 8 的 `text` 固定为空字符串。改为接受一个参数。由于 `buildFluxWorkflow` 目前不接收 negative，需要修改签名。

在 `GenerateImage` 的 dispatch 中（第 46-53 行），传递 negative：

```go
switch req.Model {
case "z-image-turbo":
    steps := 8
    workflow = b.buildZImageWorkflow(req.Prompt, width, height, seed, steps)
default:
    steps := 20
    workflow = b.buildFluxWorkflow(req.Prompt, req.Negative, width, height, seed, steps)
}
```

修改 `buildFluxWorkflow` 签名和 node 8：

```go
func (b *ComfyUIBackend) buildFluxWorkflow(prompt string, negative string, width, height, seed, steps int) map[string]interface{} {
```

Node 8 改为：
```go
"8": map[string]interface{}{
    "class_type": "CLIPTextEncode",
    "inputs": map[string]interface{}{
        "text": negative,
        "clip": []interface{}{"5", 0},
    },
},
```

- [ ] **Step 3: 验证编译**

```bash
cd D:/AI/wubigork && go build ./... 2>&1
```

Expected: Build succeeds.

---

### Task 4: 新建 PromptPanel 组件

**Files:**
- Create: `frontend/src/components/PromptPanel.tsx`

**Interfaces:**
- Props: `{ prompt, negative, onPromptChange, onNegativeChange, onTemplateSelect }`

组件 JSX 结构：

```tsx
import React from 'react'
import { Typography, Input, Tag, Space } from 'antd'
import { C } from '../utils/theme'
import { TEMPLATES, type Template } from '../data/imageTemplates'

const { TextArea } = Input

interface Props {
  prompt: string
  negative: string
  onPromptChange: (v: string) => void
  onNegativeChange: (v: string) => void
  onTemplateSelect: (t: Template) => void
}

const PromptPanel: React.FC<Props> = ({ prompt, negative, onPromptChange, onNegativeChange, onTemplateSelect }) => {
  const [showNegative, setShowNegative] = React.useState(false)

  return (
    <div>
      {/* 正向 Prompt */}
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>
        描述你想要的画面
      </Typography.Text>
      <TextArea
        placeholder="例如：一座悬浮在云端的东方仙侠城市，琉璃瓦宫殿，瀑布倾泻..."
        value={prompt}
        onChange={(e) => onPromptChange(e.target.value)}
        rows={4}
        autoSize={{ minRows: 3, maxRows: 6 }}
        style={{
          background: 'rgba(255,255,255,0.05)',
          border: '1px solid var(--border-subtle)',
          borderRadius: 'var(--radius-md)',
          color: 'var(--color-text)',
          resize: 'none',
          marginBottom: 8,
        }}
      />

      {/* 负向 Prompt — 可折叠 */}
      <Typography.Link
        onClick={() => setShowNegative(!showNegative)}
        style={{ fontSize: 11, marginBottom: showNegative ? 4 : 12, display: 'block' }}
      >
        {showNegative ? '🚫 收起不想出现的内容' : '🚫 添加不想出现的内容...'}
      </Typography.Link>
      {showNegative && (
        <TextArea
          placeholder="模糊, 低质量, 畸形手指, 多余肢体..."
          value={negative}
          onChange={(e) => onNegativeChange(e.target.value)}
          rows={2}
          autoSize={{ minRows: 1, maxRows: 3 }}
          style={{
            background: 'rgba(255,255,255,0.05)',
            border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--color-text)',
            resize: 'none',
            marginBottom: 12,
          }}
        />
      )}

      {/* 快速模板 */}
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 6 }}>
        📐 快速模板
      </Typography.Text>
      {Object.entries(TEMPLATES).map(([category, templates]) => (
        <div key={category} style={{ marginBottom: 6 }}>
          <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary'), marginRight: 4 }}>
            {category}
          </Typography.Text>
          <Space wrap size={[4, 4]}>
            {templates.map((t) => (
              <Tag
                key={t.label}
                style={{ cursor: 'pointer', borderRadius: 'var(--radius-sm)', fontSize: 11 }}
                onClick={() => onTemplateSelect(t)}
              >
                {t.label}
              </Tag>
            ))}
          </Space>
        </div>
      ))}
    </div>
  )
}

export default PromptPanel
```

- [ ] **Step 1: 写组件文件**

- [ ] **Step 2: 验证编译**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit src/components/PromptPanel.tsx 2>&1
```

Expected: No errors.

---

### Task 5: 新建 GenControls + GenButton 组件

**Files:**
- Create: `frontend/src/components/GenControls.tsx`
- Create: `frontend/src/components/GenButton.tsx`

**GenControls Props:**
```ts
{ size, model, style, seed, count, backend, showModel, onSizeChange, onModelChange, onStyleChange, onSeedChange, onCountChange }
```

**GenButton Props:**
```ts
{ generating, count, lastTime, backend, model, onGenerate }
```

- [ ] **Step 1: 写 GenControls.tsx**

完整代码见设计文档 3.3 节。使用 Ant Design Select、InputNumber、Radio.Button。

```tsx
import React from 'react'
import { Typography, Select, InputNumber, Button, Space } from 'antd'
import { ShakeOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

interface Props {
  size: string
  model: string
  style: string
  seed: number
  count: number
  backend: string
  showModel: boolean
  onSizeChange: (v: string) => void
  onModelChange: (v: string) => void
  onStyleChange: (v: string) => void
  onSeedChange: (v: number) => void
  onCountChange: (v: number) => void
}

const SIZE_OPTIONS = [
  { label: '🟦 方形 1024×1024', value: '1024x1024' },
  { label: '🎬 宽屏 1024×576', value: '1024x576' },
  { label: '📱 竖屏 576×1024', value: '576x1024' },
]

const STYLE_PRESETS = [
  { label: '🎨 数字油画', value: '数字油画风格，电影级光影，高细节，8K' },
  { label: '📸 写实摄影', value: '写实摄影风格，自然光，超高分辨率，逼真质感' },
  { label: '🖊️ 线稿插画', value: '精致线稿风格，干净利落的线条，扁平色彩，插画风' },
  { label: '🌌 概念艺术', value: '概念艺术风格，史诗级场景，戏剧性光影，氛围感强' },
  { label: '🎭 中国水墨', value: '中国水墨画风格，写意，留白，淡雅色调，传统笔触' },
  { label: '🎪 动漫风格', value: '日系动漫风格，鲜艳色彩，精致角色，明亮光影' },
  { label: '无', value: '' },
]

const GenControls: React.FC<Props> = (props) => {
  return (
    <div>
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8 }}>尺寸</Typography.Text>
      <Select value={props.size} onChange={props.onSizeChange} options={SIZE_OPTIONS} style={{ width: '100%', marginBottom: 12 }} />

      {props.showModel && (
        <>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8 }}>模型</Typography.Text>
          <Select value={props.model} onChange={props.onModelChange} style={{ width: '100%', marginBottom: 12 }}
            options={[
              { label: '🌊 Flux Dev (20步)', value: 'flux' },
              { label: '⚡ Z-Image-Turbo (8步)', value: 'z-image-turbo' },
            ]}
          />
        </>
      )}

      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8 }}>风格</Typography.Text>
      <Select value={props.style} onChange={props.onStyleChange} options={STYLE_PRESETS} style={{ width: '100%', marginBottom: 12 }} />

      <Space style={{ width: '100%' }} size={12}>
        <div style={{ flex: 1 }}>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>
            🎲 种子
          </Typography.Text>
          <InputNumber
            value={props.seed || undefined}
            onChange={(v) => props.onSeedChange(v || 0)}
            placeholder="随机"
            min={1}
            max={2147483647}
            style={{ width: '100%' }}
            addonAfter={
              <Button type="text" size="small" icon={<ShakeOutlined />}
                onClick={() => props.onSeedChange(0)}
                style={{ padding: 0, height: 20 }} />
            }
          />
        </div>
        <div>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>
            数量
          </Typography.Text>
          <Select value={props.count} onChange={props.onCountChange} style={{ width: 70 }}
            options={[{ label: '1', value: 1 }, { label: '2', value: 2 }, { label: '3', value: 3 }, { label: '4', value: 4 }]} />
        </div>
      </Space>
    </div>
  )
}

export default GenControls
```

- [ ] **Step 2: 写 GenButton.tsx**

```tsx
import React from 'react'
import { Button } from 'antd'
import { ThunderboltOutlined, LoadingOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

interface Props {
  generating: boolean
  count: number
  lastTime: number
  backend: string
  model: string
  onGenerate: () => void
}

const estimateTime = (backend: string, model: string, count: number) => {
  if (backend === 'xai') return count * 5
  if (model === 'z-image-turbo') return count * 20
  return count * 60
}

const GenButton: React.FC<Props> = ({ generating, count, lastTime, backend, model, onGenerate }) => {
  const est = estimateTime(backend, model, count)

  return (
    <div style={{ marginTop: 12 }}>
      <Button
        type="primary"
        block
        size="large"
        icon={generating ? <LoadingOutlined /> : <ThunderboltOutlined />}
        onClick={onGenerate}
        loading={generating}
        style={{ boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)', height: 44 }}
      >
        {generating ? '生成中...' : `生成 ${count} 张`}
      </Button>
      <div style={{ textAlign: 'center', marginTop: 4 }}>
        <span style={{ color: C('color-text-secondary'), fontSize: 10 }}>
          {generating ? '⏳ 等待中...' : lastTime > 0 ? `上次 ${lastTime}s · 预计 ${est}s` : `预计 ~${est}s`}
        </span>
      </div>
    </div>
  )
}

export default GenButton
```

- [ ] **Step 3: 验证编译**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit src/components/GenControls.tsx src/components/GenButton.tsx 2>&1
```

Expected: No errors.

---

### Task 6: 新建 ResultGallery + Lightbox + HistoryStrip 组件

**Files:**
- Create: `frontend/src/components/ResultGallery.tsx`
- Create: `frontend/src/components/Lightbox.tsx`
- Create: `frontend/src/components/HistoryStrip.tsx`

**共用类型**（在 ResultGallery.tsx 中定义并 export）:

```ts
export interface GenResult {
  image: string
  seed: number
  time: number
  prompt: string
  negative?: string
  model: string
  size: string
  style?: string
}
```

- [ ] **Step 1: 写 ResultGallery.tsx**

```tsx
import React from 'react'
import { Typography, Spin, Empty } from 'antd'
import { ExpandOutlined, DownloadOutlined, SyncOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import type { GenResult } from './ResultGallery' // self-ref for types

export interface GenResult { image: string; seed: number; time: number; prompt: string; negative?: string; model: string; size: string; style?: string }

interface Props {
  results: GenResult[]
  generating: boolean
  onPreview: (index: number) => void
  onDownload: (index: number) => void
  onReuse: (index: number) => void
}

const ResultGallery: React.FC<Props> = ({ results, generating, onPreview, onDownload, onReuse }) => {
  if (generating && results.length === 0) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 300, flexDirection: 'column', gap: 16 }}>
        <Spin size="large" />
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13 }}>AI 正在绘制中...</Typography.Text>
      </div>
    )
  }

  if (results.length === 0) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 300 }}>
        <Empty description={<span style={{ color: C('color-text-secondary') }}>输入描述，点击生成</span>} />
      </div>
    )
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 12 }}>
      {results.map((r, i) => (
        <div key={i} style={{
          position: 'relative',
          borderRadius: 'var(--radius-md)',
          overflow: 'hidden',
          border: '1px solid var(--border-subtle)',
          background: 'var(--bg-elevated)',
          cursor: 'pointer',
        }}
          onClick={() => onPreview(i)}
        >
          <img src={r.image} alt="" style={{ width: '100%', display: 'block', aspectRatio: r.size === '576x1024' ? '9/16' : r.size === '1024x576' ? '16/9' : '1/1', objectFit: 'cover' }} />
          <div style={{ position: 'absolute', top: 6, right: 6, background: 'rgba(0,0,0,0.6)', borderRadius: 'var(--radius-sm)', padding: '1px 6px', fontSize: 10, color: '#fff' }}>
            {r.time}s
          </div>
          <div style={{
            position: 'absolute', bottom: 0, left: 0, right: 0,
            background: 'rgba(0,0,0,0.5)', padding: '4px 8px',
            display: 'flex', gap: 8, justifyContent: 'center', opacity: 0.85,
          }}>
            <ExpandOutlined style={{ color: '#fff', fontSize: 12 }} onClick={(e) => { e.stopPropagation(); onPreview(i) }} />
            <DownloadOutlined style={{ color: '#fff', fontSize: 12 }} onClick={(e) => { e.stopPropagation(); onDownload(i) }} />
            <SyncOutlined style={{ color: '#fff', fontSize: 12 }} onClick={(e) => { e.stopPropagation(); onReuse(i) }} />
          </div>
        </div>
      ))}
    </div>
  )
}

export default ResultGallery
```

- [ ] **Step 2: 写 Lightbox.tsx**

```tsx
import React, { useEffect } from 'react'
import { Typography, Button, Space, Tag } from 'antd'
import { DownloadOutlined, SyncOutlined, CloseOutlined, LeftOutlined, RightOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import type { GenResult } from './ResultGallery'

interface Props {
  results: GenResult[]
  index: number
  onClose: () => void
  onIndexChange: (i: number) => void
  onDownload: (i: number) => void
  onReuse: (i: number) => void
}

const Lightbox: React.FC<Props> = ({ results, index, onClose, onIndexChange, onDownload, onReuse }) => {
  const r = results[index]
  if (!r) return null

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft' && index > 0) onIndexChange(index - 1)
      if (e.key === 'ArrowRight' && index < results.length - 1) onIndexChange(index + 1)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [index, results.length])

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 1000,
      background: 'rgba(0,0,0,0.85)', display: 'flex',
      flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
    }} onClick={onClose}>
      <div style={{ position: 'absolute', top: 16, right: 16 }}>
        <Button type="text" icon={<CloseOutlined />} onClick={onClose} style={{ color: '#fff', fontSize: 20 }} />
      </div>

      {index > 0 && (
        <Button type="text" icon={<LeftOutlined />} onClick={(e) => { e.stopPropagation(); onIndexChange(index - 1) }}
          style={{ position: 'absolute', left: 16, top: '50%', color: '#fff', fontSize: 24 }} />
      )}
      {index < results.length - 1 && (
        <Button type="text" icon={<RightOutlined />} onClick={(e) => { e.stopPropagation(); onIndexChange(index + 1) }}
          style={{ position: 'absolute', right: 16, top: '50%', color: '#fff', fontSize: 24 }} />
      )}

      <div onClick={(e) => e.stopPropagation()} style={{ maxWidth: '90vw', maxHeight: '75vh' }}>
        <img src={r.image} alt="" style={{ maxWidth: '100%', maxHeight: '75vh', borderRadius: 8, objectFit: 'contain' }} />
      </div>

      <div onClick={(e) => e.stopPropagation()} style={{ marginTop: 16, maxWidth: '90vw', textAlign: 'center' }}>
        <Typography.Text style={{ color: '#ccc', fontSize: 12, display: 'block', marginBottom: 8, maxHeight: 40, overflow: 'hidden' }}>
          {r.prompt}
        </Typography.Text>
        <Space size={8}>
          <Tag color="blue">🎲 种子: {r.seed}</Tag>
          <Tag color="green">{r.model}</Tag>
          <Tag>{r.size}</Tag>
          <Tag>⏱ {r.time}s</Tag>
        </Space>
        <div style={{ marginTop: 12 }}>
          <Space>
            <Button icon={<DownloadOutlined />} onClick={() => onDownload(index)} ghost>下载</Button>
            <Button icon={<SyncOutlined />} onClick={() => onReuse(index)} ghost>重用参数</Button>
          </Space>
        </div>
      </div>
    </div>
  )
}

export default Lightbox
```

- [ ] **Step 3: 写 HistoryStrip.tsx**

```tsx
import React from 'react'
import { Typography, Button } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import type { GenResult } from './ResultGallery'

interface Props {
  history: GenResult[]
  onSelect: (index: number) => void
  onClear: () => void
}

const HistoryStrip: React.FC<Props> = ({ history, onSelect, onClear }) => {
  if (history.length === 0) return null

  return (
    <div style={{ marginTop: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
          📜 本次会话历史 ({history.length})
        </Typography.Text>
        <Button type="text" size="small" icon={<DeleteOutlined />} onClick={onClear}
          style={{ color: C('color-text-secondary'), fontSize: 11 }}>清空</Button>
      </div>
      <div style={{ display: 'flex', gap: 8, overflowX: 'auto', paddingBottom: 4 }}>
        {history.map((h, i) => (
          <div key={i} onClick={() => onSelect(i)} style={{
            width: 72, height: 72, flexShrink: 0, cursor: 'pointer',
            borderRadius: 'var(--radius-sm)', overflow: 'hidden',
            border: '1px solid var(--border-subtle)',
          }}>
            <img src={h.image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
          </div>
        ))}
      </div>
    </div>
  )
}

export default HistoryStrip
```

- [ ] **Step 4: 验证编译**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit src/components/ResultGallery.tsx src/components/Lightbox.tsx src/components/HistoryStrip.tsx 2>&1
```

Expected: No errors.

---

### Task 7: 重写 ImageGenPage.tsx

**Files:**
- Modify: `frontend/src/pages/ImageGenPage.tsx`（完全重写）

**Interfaces:**
- Consumes: PromptPanel, GenControls, GenButton, ResultGallery, Lightbox, HistoryStrip, GenResult 类型
- Uses: `useIsMobile()`, `window.go.app.App.GenerateFreeImage(...)`, `window.go.app.App.GetImageBackendInfo()`

完整重写 ImageGenPage.tsx，双栏布局，整合所有子组件。代码 ~200 行。

详见设计文档 3.1 节的状态定义和生成流程。核心逻辑：

```tsx
import React, { useState, useCallback } from 'react'
import { Typography, Tag, Space, Button, Drawer } from 'antd'
import { PictureOutlined, CloudOutlined, HomeOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import { useIsMobile } from '../hooks/useMediaQuery'
import PromptPanel from '../components/PromptPanel'
import GenControls from '../components/GenControls'
import GenButton from '../components/GenButton'
import ResultGallery, { type GenResult } from '../components/ResultGallery'
import Lightbox from '../components/Lightbox'
import HistoryStrip from '../components/HistoryStrip'

const ImageGenPage: React.FC = () => {
  const isMobile = useIsMobile()

  // 参数状态
  const [prompt, setPrompt] = useState('')
  const [negative, setNegative] = useState('')
  const [size, setSize] = useState('1024x1024')
  const [style, setStyle] = useState('')
  const [model, setModel] = useState('flux')
  const [seed, setSeed] = useState(0)
  const [count, setCount] = useState(1)

  // 生成状态
  const [generating, setGenerating] = useState(false)
  const [backend, setBackend] = useState('xai')
  const [showModel, setShowModel] = useState(false)
  const [lastTime, setLastTime] = useState(0)

  // 结果 & 历史
  const [results, setResults] = useState<GenResult[]>([])
  const [history, setHistory] = useState<GenResult[]>([])
  const [lightboxIndex, setLightboxIndex] = useState(-1)
  const [showResult, setShowResult] = useState(false) // 移动端

  // 加载后端信息
  React.useEffect(() => {
    (async () => {
      try {
        // @ts-ignore
        const info = await window.go.app.App.GetImageBackendInfo()
        if (info?.backend) { setBackend(info.backend); setShowModel(info.backend === 'comfyui') }
        if (info?.model) setModel(info.model)
      } catch (_) {}
    })()
  }, [])

  // 生成
  const handleGenerate = useCallback(async () => {
    if (!prompt.trim()) return
    setGenerating(true)
    setResults([])
    setLightboxIndex(-1)
    try {
      // @ts-ignore
      const res = await window.go.app.App.GenerateFreeImage(prompt.trim(), negative.trim(), size, style, model, seed, count)
      if (res?.error) {
        message.error(res.error)
      } else if (res?.images?.length) {
        const genResults: GenResult[] = res.images
        setResults(genResults)
        setHistory((prev) => [...genResults, ...prev])
        setLastTime(Math.max(...genResults.map((r: GenResult) => r.time)))
        setLightboxIndex(0)
        setShowResult(true)
        message.success(`✨ 已生成 ${genResults.length} 张图片`)
      }
    } catch (err: any) {
      message.error(err?.message || '生成失败')
    } finally {
      setGenerating(false)
    }
  }, [prompt, negative, size, style, model, seed, count])

  // 模板点击
  const handleTemplate = useCallback((t: { prompt: string; negative: string }) => {
    if (prompt) setPrompt((p) => p + '，' + t.prompt)
    else setPrompt(t.prompt)
    if (negative) setNegative((n) => n + ', ' + t.negative)
    else setNegative(t.negative)
  }, [prompt, negative])

  // 重用参数
  const handleReuse = useCallback((index: number) => {
    const r = history[index]
    if (!r) return
    setPrompt(r.prompt)
    setSize(r.size)
    setModel(r.model)
    setSeed(r.seed)
    setLightboxIndex(-1)
  }, [history])

  // 下载
  const handleDownload = useCallback((index: number) => {
    const r = history[index] || results[index]
    if (!r?.image) return
    const a = document.createElement('a')
    a.href = r.image
    a.download = `wubigork-${Date.now()}-seed${r.seed}.png`
    a.click()
  }, [history, results])

  const controls = (
    <>
      <PromptPanel {...{ prompt, negative, onPromptChange: setPrompt, onNegativeChange: setNegative, onTemplateSelect: handleTemplate }} />
      <div style={{ height: 1, background: 'var(--border-subtle)', margin: '16px 0' }} />
      <GenControls {...{ size, model, style, seed, count, backend, showModel, onSizeChange: setSize, onModelChange: setModel, onStyleChange: setStyle, onSeedChange: setSeed, onCountChange: setCount }} />
      <GenButton {...{ generating, count, lastTime, backend, model, onGenerate: handleGenerate }} />
    </>
  )

  const resultArea = (
    <ResultGallery {...{ results, generating, onPreview: setLightboxIndex, onDownload: handleDownload, onReuse: (i: number) => handleReuse(history.length - results.length + i) }} />
  )

  return (
    <div style={{ height: 'calc(100vh - 120px)', display: 'flex', flexDirection: 'column' }}>
      {/* 顶部栏 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16, flexShrink: 0 }}>
        <Typography.Title level={4} style={{ color: C('color-text'), margin: 0 }}>
          <PictureOutlined style={{ marginRight: 10 }} />AI 绘梦
        </Typography.Title>
        <Tag color={backend === 'comfyui' ? 'green' : 'blue'} style={{ borderRadius: 'var(--radius-md)' }}>
          {backend === 'comfyui' ? <><HomeOutlined /> 本地 {model === 'z-image-turbo' ? 'Z-Image-Turbo' : 'Flux'}</> : <><CloudOutlined /> xAI 云端</>}
        </Tag>
      </div>

      {/* 主工作区 */}
      {isMobile ? (
        <>
          <div style={{ flex: 1, overflow: 'auto' }}>{controls}</div>
          <Drawer open={showResult} onClose={() => setShowResult(false)} placement="bottom" height="80%" title="生成结果">
            {resultArea}
          </Drawer>
        </>
      ) : (
        <div style={{ flex: 1, display: 'flex', gap: 20, overflow: 'hidden' }}>
          <div style={{ width: 320, flexShrink: 0, overflow: 'auto', paddingRight: 8 }}>
            {controls}
          </div>
          <div style={{ flex: 1, overflow: 'auto' }}>
            {resultArea}
            <HistoryStrip {...{ history, onSelect: setLightboxIndex, onClear: () => setHistory([]) }} />
          </div>
        </div>
      )}

      {/* 灯箱 */}
      {lightboxIndex >= 0 && (
        <Lightbox {...{ results: history, index: lightboxIndex, onClose: () => setLightboxIndex(-1), onIndexChange: setLightboxIndex, onDownload: handleDownload, onReuse: handleReuse }} />
      )}
    </div>
  )
}

export default ImageGenPage
```

- [ ] **Step 1: 写入重写后的 ImageGenPage.tsx**

完整内容见上方代码块。

- [ ] **Step 2: 验证编译**

```bash
cd D:/AI/wubigork/frontend && npm run build 2>&1
```

Expected: Build succeeds.

---

### Task 8: 构建 & 端到端测试

- [ ] **Step 1: 构建 Go 后端**

```bash
cd D:/AI/wubigork && go build -o build/bin/wubigork.exe . 2>&1
```

- [ ] **Step 2: 构建前端**

```bash
cd D:/AI/wubigork/frontend && npm run build 2>&1
```

- [ ] **Step 3: 功能测试清单**

1. 启动 ComfyUI + wubigork
2. 设置页切换到 ComfyUI + Z-Image-Turbo
3. AI 绘梦页：桌面端确认双栏布局，左栏 320px 控制面板
4. 输入 prompt → 点击模板标签 → 确认 prompt 被填充
5. 展开负向 prompt → 输入内容 → 生成
6. 设置种子为 42 → 批量 2 张 → 点击生成 → 确认 2 张图出现
7. 点击第一张 → 灯箱打开 → 左右切换 → 确认种子/耗时标签正确
8. 灯箱中"重用参数" → 确认 prompt/seed 填回控制面板
9. 下载图片 → 确认文件名含种子
10. 历史画廊 → 确认所有图片出现 → 清空
11. 缩小窗口到移动端 → 确认单栏 + 底部抽屉
