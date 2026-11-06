# gaea 绘梦·阶段二刀 C：扩图（outpaint）——画布加边，模型补全新区域规格书

> 2026-09-23 立项。来源：用户指令「继续」（画室线既定刀序：T3 编辑力第三件）。
> 上游：长期规划 `docs/gaea-image-domain-longterm-plan-2026.md` T3「蒙版局部重绘 +
> 扩图 + 抠图」；v4.393.0（阶段二刀 B 蒙版）观察池「扩图（outpaint 画布扩边交互）」
> 销号。目标版本：v4.394.0。

## 1. 论点

扩图 = 把画布向指定方向扩大，新区域由模型按指令补全、原图区域保持不变。
市场标配（即梦「扩图」/Nano Banana Pro「outpaint」/PS 生成式扩展）。技术上门是
开着的：**扩图 ≡ 合成大画布（原图贴锚点+新区域填中性灰）+ 灰度蒙版（新区域白）**
——正是 v4.393 已全链打通的 edit+mask 请求形态，两个后端（OpenAI 兼容/ComfyUI）
零引擎改动即可吃。新代码集中在**合成器**（Go 纯函数）与前端**扩边交互**。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 合成器放 `internal/ai`（图像域领域逻辑，sin/portrait 等入口未来可复用）：`ComposeOutpaint(initImage, expand)` → (画布 dataURL, 蒙版 dataURL)。画布=原图锚定 + 四边按百分比扩展；新区域填中性灰 128（对噪声初始化最中性）；蒙版=原图区黑、扩边环白 |
| Q2 | 展开量：四边各自百分比（0–200，相对原图对应边长）；四边全 0 → 报错「扩图需要至少一边扩展量大于 0」；合成画布 >16.7MP（4096²）→ 报错「扩图后画布过大」（防 base64 传输爆炸） |
| Q3 | 前端契约：`mode:"outpaint"` + `expand:{left,top,right,bottom}` + initImage；app 层把 outpaint **转换为 edit 请求**（合成→ imgReq.Mode="edit"+合成画布+合成蒙版）下发——ai 层 fail-closed（mask 仅 edit）天然满足；结果 mode 回显 outpaint（用户视角诚实）；comfyui 元数据 qwen-image-edit 沿用 edit 分支 |
| Q4 | 前端：InstructionEditModal 范围 Radio 三态「全图 / 局部（涂选）/ 扩图」；扩图态=预设 chips（左 50%/右 50%/上 50%/下 50%/四边 25%/补正方）+ 四边数字输入（0–200）+ 虚线扩边框预览（按比例 padding）；至少一边>0 才可生成 |
| Q5 | 与蒙版互斥：outpaint 不带用户 mask（蒙版由合成器生成）；「补正方」=长边补短边（左右或上下均分差额），目标 1:1 |
| Q6 | 零新绑定（714 不变）：GenerateMedia paramsJSON 加 `expand` 键与 mode 值 |

## 3. 线 A：internal/ai/image_outpaint.go（新文件）

```go
type OutpaintExpand struct{ Left, Top, Right, Bottom int } // 百分比
func ComposeOutpaint(initImage string, e OutpaintExpand) (canvas, mask string, err error)
```

- decode 原图（decodeDataURLBytes 复用；非 data URL 报错）；
- 画布尺寸 `origW×(1+(L+R)/100)`（floor），锚点 `(origW×L/100, origH×T/100)`；
- 画布 RGBA：先整幅填灰 128 → draw.Draw 原图到锚点；蒙版 Gray：黑底 → 扩边环白
  （四个矩形：上条/下条/左条/右条，与锚定原图区互补）；
- 双 PNG encode → data URL 返回。

## 4. 线 B：internal/app GenerateMedia

- `mediaGenParams` +`Expand *outpaintExpandJSON`（{left,top,right,bottom}）；
- mode=outpaint 分支：InitImage 必填（复用 edit 校验文案「指令编辑需要原图」
  → 独立文案「扩图需要原图」）；ComposeOutpaint → imgReq{Mode:"edit",
  InitImage:画布, Mask:蒙版}；结果 mode 回显 outpaint；P.Mask 用户键在 outpaint
  下非空 → 拒绝（fail-closed：蒙版与扩图是两条通道不混用）。

## 5. 线 C：前端

- `api/image.ts`：mode 联合 +`'outpaint'`；+`expand?: {left,top,right,bottom}`。
- `InstructionEditModal`：Radio 三态；扩图态渲染 OutpaintPanel（内联）：
  chips 预设（点击写入四边值）+ 四个 InputNumber + 预览虚线框；run() 带
  mode/expand/initImage；四边全 0 → warning。

## 6. 测试

- ai：①ComposeOutpaint 形状（2×1 原图+左 100/右 100 → 6×1 画布、原图像素锚点
  x=2、蒙版白区=左右扩边各 2px、黑区=中间 2px；四边 0 报错；超 16.7MP 报错；
  非 data URL 报错）②补正方数学（3×1 补方 → 3×3，上下各 1）。
- app：outpaint 透传转换（fake lastReq.Mode=edit + lastReq.InitImage 可 decode
  断言尺寸 + lastReq.Mask 非空；mode 回显 outpaint）；outpaint+用户 mask 拒绝；
  outpaint 缺图拒绝。
- 前端：Modal 扩图态（切 Radio→chips 点击→params 带 expand/mode=outpaint；
  全 0 warning；预览框在位）。

## 7. 验收与门禁

go build/vet、定向+全量 ci.ps1、tsc/eslint/vitest 全量、drift PASS@714；
真机走查挂池（与编辑/蒙版同班：ComfyUI 权重就位后一并验）。

## 8. 观察池（本刀不做）

抠图/透明底 PNG 导出（T3 最后一件）；扩边羽化；非矩形扩区（自由形状=涂选已覆盖）；
img2img+mask；变体簇；多图编辑；Lightning LoRA。
