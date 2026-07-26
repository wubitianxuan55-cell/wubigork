# AI 绘梦页面全面重做 — 设计文档

**日期**: 2026-07-03
**版本**: v1.0
**影响范围**: `ImageGenPage.tsx`（重写）、新增 4 个组件文件、`image_handler.go`（扩展）、`image_comfyui.go`（扩展负向 prompt）

---

## 一、目标

将 AI 绘梦页面从单列"输入→输出"流程升级为**双栏专业工作台**，新增负向 Prompt、种子控制、批量生成、进度计时、快速预设模板、会话历史画廊。

---

## 二、架构

```
ImageGenPage.tsx  (页面框架, 状态管理, 双栏布局)
├── PromptPanel.tsx    正/负向 prompt + 预设模板
├── GenControls.tsx    尺寸/模型/风格/种子/数量
├── GenButton.tsx      生成按钮 + 预计耗时
├── ResultGallery.tsx  当前批次结果网格
├── Lightbox.tsx       全屏单图预览
└── HistoryStrip.tsx   本次会话历史缩略图
```

- **桌面端**: 左栏 320px 控制面板 | 右栏 flex:1 结果区
- **移动端**: 单栏，控制面板全宽，结果区以底部抽屉/全屏切换展示

---

## 三、组件设计

### 3.1 ImageGenPage — 页面框架

**职责**: 状态管理、双栏布局、后端通信

| 状态 | 类型 | 说明 |
|------|------|------|
| `prompt` | string | 正向 prompt |
| `negative` | string | 负向 prompt |
| `size` | string | 尺寸 "1024x1024" |
| `style` | string | 风格预设值 |
| `model` | string | "flux" / "z-image-turbo" |
| `seed` | number | 0=随机, >0=固定 |
| `count` | number | 批量数量 1-4 |
| `backend` | string | "xai" / "comfyui" |
| `generating` | boolean | 生成中 |
| `results` | GenResult[] | 当前批次结果 |
| `history` | GenResult[] | 本次会话全部历史 |
| `lightboxIndex` | number | 灯箱当前索引，-1=关闭 |

```ts
interface GenResult {
  image: string        // base64 data URL
  seed: number         // 实际种子
  time: number         // 秒
  prompt: string
  negative: string
  model: string
  size: string
  style: string
}
```

**生成流程**:
1. 前端调用 `window.go.app.App.GenerateFreeImage(prompt, negative, size, style, model, seed, count)`
2. 显示生成中状态 + 进度文案
3. 返回 `{ images: GenResult[], error: string }`
4. 追加到 `results` 和 `history`（前端去重）
5. 首张图自动进入灯箱预览

### 3.2 PromptPanel — 输入区

**Props**: `prompt`, `negative`, `onPromptChange`, `onNegativeChange`, `onTemplateSelect`

**预设模板**: 5 大类 20 个模板，每个含正/负向 prompt：

```ts
const TEMPLATES = {
  创作类: [
    { label: '📕 小说封面', prompt: '精美小说封面设计，专业排版构图，电影级光影，浮雕标题，8K超高清', negative: '低质量, 模糊, 文字扭曲, 错别字' },
    { label: '👤 角色肖像', prompt: '精致角色肖像，柔和棚拍光线，半身构图，精致的面部细节，8K超清', negative: '变形手指, 多余肢体, 低分辨率, 模糊' },
    { label: '🏔️ 场景概念', prompt: '史诗级场景概念艺术，广阔视野，戏剧性光影，氛围感强，丰富细节', negative: '模糊, 平面感, 简陋, 低质量' },
    { label: '🎨 插画内页', prompt: '精美插画风格，细腻线条，柔和色调，叙事性构图，艺术感', negative: '照片感, 3D渲染感, 低分辨率' },
    { label: '📖 章节插图', prompt: '小说章节插图，与文字搭配的叙事画面，适合排版，留白设计，精致', negative: '混乱, 模糊, 低质量, 过度复杂' },
  ],
  写实类: [
    { label: '📸 写实摄影', prompt: '写实摄影风格，自然光线，超高分辨率，逼真质感，细节丰富，8K', negative: '模糊, 低质量, 动漫风格, 绘画质感, 3D渲染感' },
    { label: '🎬 电影剧照', prompt: '电影剧照风格，宽银幕构图，戏剧性光影，胶片质感，电影级调色', negative: '模糊, 低质量, 平面感, 电视画质' },
    { label: '👗 时尚大片', prompt: '时尚杂志大片风格，精致妆造，高级灯光，完美构图，奢侈品质感', negative: '模糊, 低质量, 廉价感, 过时服装' },
    { label: '🔍 微距特写', prompt: '微距摄影特写，极致细节，浅景深，柔美散景，清晰焦点，8K超清', negative: '模糊, 大景深, 噪点, 低分辨率' },
    { label: '🚶 纪实街拍', prompt: '纪实街拍风格，自然抓拍感，真实光影，生活气息，人文情怀', negative: '摆拍感, 过度修饰, 滤镜感, 3D渲染' },
  ],
  风格类: [
    { label: '🖌️ 中国水墨', prompt: '中国水墨画风格，写意笔触，留白构图，淡雅墨色，传统韵味', negative: '色彩鲜艳, 油画质感, 照片感, 西式构图' },
    { label: '🎪 日系动漫', prompt: '日系动漫风格，鲜艳色彩，精致角色，明亮光影，干净线条', negative: '写实感, 照片感, 暗黑风格, 3D渲染' },
    { label: '🌆 赛博朋克', prompt: '赛博朋克风格，霓虹灯光，雨夜都市，高科技感，蓝紫冷色调', negative: '自然风光, 暖色调, 古代风格, 简陋' },
    { label: '🧙 奇幻风格', prompt: '奇幻艺术风格，魔法氛围，史诗感，精细纹理，丰富想象力', negative: '现代元素, 科技感, 简约风格, 照片感' },
    { label: '🎨 数字油画', prompt: '数字油画风格，电影级光影，高细节，8K，丰富色彩层次', negative: '模糊, 低质量, 线稿, 扁平风格' },
  ],
  构图类: [
    { label: '🧍 半身肖像', prompt: '精致半身肖像构图，柔光，干净背景，人物突出，8K超清', negative: '全身, 多人, 混乱背景, 低分辨率' },
    { label: '🧎 全身立绘', prompt: '全身角色立绘，完整人物展示，清晰服装设计，白色背景，角色设计图', negative: '半身, 裁切, 模糊面部, 复杂背景' },
    { label: '🌄 广角场景', prompt: '广角场景构图，宏大视野，深远景深，丰富层次，电影级', negative: '特写, 浅景深, 扁平, 低分辨率' },
    { label: '🚁 俯瞰航拍', prompt: '俯瞰航拍视角，上帝视角，全景构图，丰富细节，震撼', negative: '仰视, 平视, 特写, 低清晰度' },
    { label: '👁️ 特写面部', prompt: '面部大特写，极致细节，柔光，眼神表达，8K超清', negative: '全身, 远景, 遮挡面部, 模糊' },
  ],
}
```

点击模板时：
- 如果 prompt 为空 → 直接填入
- 如果已有内容 → 追加到末尾（用逗号分隔）
- 负向 prompt 同样追加

### 3.3 GenControls — 参数区

**Props**: `size`, `model`, `style`, `seed`, `count`, `backend`, callbacks

| 控件 | 类型 | 值 |
|------|------|-----|
| 尺寸 | Radio.Button | 方形(1024), 宽屏(1024×576), 竖屏(576×1024) |
| 模型 | Radio.Button | Flux / Z-Image-Turbo（仅 ComfyUI 显示） |
| 风格 | Select | 与现有 STYLE_PRESETS 一致 |
| 种子 | InputNumber + 🎲按钮 | 默认留空=随机；填数字=固定；🎲清空 |
| 数量 | Select | 1/2/3/4 |

### 3.4 GenButton — 生成按钮

**Props**: `generating`, `count`, `backend`, `model`, `onGenerate`

```
┌─ 状态 ─────────────────────────────┐
│ 空闲:  [⚡ 生成 N 张]   预计 ~90s   │
│ 生成中: [⏳ 生成中...]  已耗时 48s  │
│ 完成:  [🔄 再生成 N 张] 上次 48s    │
└────────────────────────────────────┘
```

预计时间估算：
- Z-Image-Turbo: count × ~20s/张
- Flux: count × ~60s/张
- xAI: count × ~5s/张

### 3.5 ResultGallery — 结果网格

**Props**: `results: GenResult[]`, `onPreview(index)`, `generating`

- 网格布局，自适应 2-3 列
- 每张卡片：缩略图 + 耗时标签 + 悬停操作（放大/下载/重用参数）
- 生成中显示骨架屏动画

### 3.6 Lightbox — 全屏灯箱

**Props**: `results: GenResult[]`, `index`, `onClose`, `onIndexChange`, `onReuse`

```
┌────────────────────────────────────┐
│  ← 上一张    1/2    下一张 →    ✕  │
│                                    │
│          🖼️ 大图（自适应）          │
│                                    │
│  prompt 文本（可滚动）              │
│  负向: xxx                         │
│  🎲 种子: 42 | 模型: Z-Image      │
│  ⏱ 耗时: 48s | 尺寸: 1024×1024   │
│                                    │
│  [⬇ 下载] [🔄 重用参数]           │
└────────────────────────────────────┘
```

- 键盘：← → 切换，Esc 关闭
- "重用参数"：把该图的所有参数填回控制面板
- 背景半透明黑色遮罩

### 3.7 HistoryStrip — 历史画廊

**Props**: `history: GenResult[]`, `onSelect(index)`

- 横向滚动缩略图条，在页面底部或右栏底部
- 每张缩略图 80×80，点击打开对应灯箱
- 支持清空历史按钮
- 自动去重（同 seed + 同 prompt 视为重复）

---

## 四、Go 后端改动

### 4.1 扩展 GenerateFreeImage

**文件**: `internal/app/image_handler.go`

```go
func (a *App) GenerateFreeImage(
    prompt, negative, size, style, model string,
    seed int, n int,
) (map[string]interface{}, error)
```

**逻辑**：
1. 构建 prompt（style 追加）
2. 如果 `seed == 0`，每次生成用随机种子
3. 循环 n 次，并发/串行提交任务（ComfyUI 串行避免 OOM，xAI 并发）
4. 每次计时 `time.Since(start)`
5. 返回 `{ "images": [...], "error": "" }`

### 4.2 负向 Prompt 支持

**文件**: `internal/ai/image_comfyui.go`

Flux 工作流（已有 node 8 负向 CLIPTextEncode）：
```go
// node 8 原来是空字符串，改为传入 negative
"8": map[string]interface{}{
    "class_type": "CLIPTextEncode",
    "inputs": map[string]interface{}{
        "text": negative,  // ← 之前是 ""
        "clip": []interface{}{"5", 0},
    },
},
```

Z-Image-Turbo：CFG=0 不支持负向 prompt，不传。

### 4.3 种子控制

`ImageGenerationRequest` 已无 `Seed` 字段，通过工作流参数直接传入。ComfyUI 工作流中 `seed` 从请求参数取值（0 时用 `rand.Intn(1<<31)`）。

---

## 五、移动端适配

| 桌面端 | 移动端 |
|--------|--------|
| 左栏 320px | 全宽控制面板 |
| 右栏结果 | 底部 Drawer / 全屏结果页 |
| 灯箱居中 | 全屏覆盖 |
| 历史横条 | 水平滚动缩略图 |

使用现有 `useIsMobile()` hook。

---

## 六、不变更的部分

- SettingsPage 保持现状（后端配置入口）
- ComfyUI 内嵌 iframe 移到设置页（在 AI 绘梦里太占空间，不常用）
- xAI 后端逻辑不变
- 风格预设数组保持不变
- 零新 npm 依赖（纯 Ant Design 组件）

---

## 七、文件清单

| 操作 | 文件 |
|------|------|
| 重写 | `frontend/src/pages/ImageGenPage.tsx` |
| 新建 | `frontend/src/components/PromptPanel.tsx` |
| 新建 | `frontend/src/components/GenControls.tsx` |
| 新建 | `frontend/src/components/ResultGallery.tsx` |
| 新建 | `frontend/src/components/Lightbox.tsx` |
| 新建 | `frontend/src/components/HistoryStrip.tsx` |
| 新建 | `frontend/src/data/imageTemplates.ts` |
| 修改 | `internal/app/image_handler.go` |
| 修改 | `internal/ai/image_comfyui.go` |
