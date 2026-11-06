# gaea 绘梦·阶段二刀 B：蒙版局部重绘（canvas 涂选 → 只改圈选区）规格书

> 2026-09-23 立项。来源：用户指令「继续」（画室线既定刀序：T3 编辑力余项头号）。
> 上游：长期规划 `docs/gaea-image-domain-longterm-plan-2026.md` T3「蒙版局部重绘 +
> 扩图 + 抠图」；v4.392.0（阶段二刀 A 本地档）观察池。前置：指令编辑双档已通
> （云端 /images/edits v4.327 + ComfyUI 本地档 v4.392）。目标版本：v4.393.0。
> 本刀范围：**蒙版 + 局部重绘**；扩图/抠图留观察池（画布扩边交互/透明底导出是
> 另两把刀的 UX 面）。

## 1. 论点

「不满意→修」的最后一格是「只修我圈的地方」：全图编辑对已满意的区域有回归
风险（模型每次重采样全图），蒙版局部重绘是 NovelAI V5/即梦/可灵全系标配。
gaea 双编辑档（OpenAI 兼容/ComfyUI）都具备蒙版通道：OpenAI /images/edits 的
`mask` multipart 字段是标准暴露面；ComfyUI `SetLatentNoiseMask` 是标准 inpaint
节点（本机 ComfyUI 0.36 源码实读：samples+mask，noise_mask=1 区域重采样）。

## 2. 统一蒙版契约（关键裁决）

**gaea 全链统一口径：Mask = 灰度 PNG data URL，白(255)=重绘区、黑(0)=保留区。**

- 前端画布黑底白笔刷导出，直观可预览；
- ComfyUI 分支零转换：LoadImage(RGB)→ImageScale→ImageToMask(red) 白→1.0；
- OpenAI 分支一处转换：`grayMaskToOpenAIMask`——白区→alpha=0（OpenAI 语义
  透明=编辑区），黑区→不透明黑。标准库 image/png 实现，无需新依赖。

## 3. 裁决

| # | 裁决 |
|---|---|
| Q1 | `ImageGenerationRequest` +`Mask` 字段（data URL）；仅 `mode=edit` 消费 |
| Q2 | app 层 fail-closed：`Mask` 非空且 `mode!=edit` → 拒绝「蒙版仅支持指令编辑」；edit 透传 |
| Q3 | ComfyUI 蒙版链（尺寸对齐是难点）：`LoadImage(mask)` → `ImageScale(lanczos, targetW, targetH, disabled)` → `ImageToMask(red)` → `SetLatentNoiseMask(samples=VAEEncode)` → `KSampler.latent_image`。**targetW/H = 复刻 `PREFERRED_KONTEXT_RESOLUTIONS`（17 档）最近宽高比选择**（本机 ComfyUI nodes_flux.py:105 实读），与原图过 FluxKontextImageScale 的目标尺寸一致——蒙版与 latent 空间对齐；原图尺寸经 `image.DecodeConfig(InitImage)` 取得，解析失败诚实报错 |
| Q4 | TextEncodeQwenImageEditPlus 的 image1 保持**全图**（语义参考需要全图上下文；重绘区域由 noise_mask 限制）——社区验证的 edit-model inpaint 形态 |
| Q5 | 前端 `MaskBrushEditor`：strokes 状态（点列+笔刷半径）+ 黑底白笔刷导出 `renderMaskDataURL`；jsdom 无 2d ctx 时导出守卫返回 null（降级=提示先涂抹，不崩）；坐标按 `getBoundingClientRect` 比例映射到自然尺寸（CSS 缩放显示不影响涂抹精度） |
| Q6 | `InstructionEditModal` 加「范围：全图/局部」切换：局部=原图上叠可涂抹画布（半透明红笔刷）+ 笔刷大小滑杆 + 清除；局部未涂抹→warning 不触发生成 |
| Q7 | 零新绑定（714 不变）：走 GenerateMedia 既有 paramsJSON 加 `mask` 键 |
| Q8 | `img2img+mask`（无指令语义的纯局部重绘）留观察池——本刀只做「指令+蒙版」复合形态 |

## 4. 线 A：internal/ai

- `types.go`：`Mask string`（json mask,omitempty）。
- `image_openai.go`：editImage 在 Mask 非空时转换并附 `mask` 文件（与 image 同款
  文件名推断；转换失败诚实报错）。
- `image_comfyui.go`：edit 分支 Mask 非空 → uploadImage(mask) + DecodeConfig(原图)
  → buildQwenImageEditWorkflow 加蒙版链节点（"2" LoadImage(mask)/"161" ImageScale/
  "162" ImageToMask/"163" SetLatentNoiseMask；KSampler.latent_image 改接 "163"）；
  `kontextScaleSize(w,h)` 复刻 17 档表（0/负数兜底 1024²）。

## 5. 线 B：internal/app

- `mediaGenParams` +`Mask`（json mask）；GenerateMedia：Q2 的 fail-closed 分支 +
  imgReq.Mask 透传。

## 6. 线 C：前端

- `api/image.ts`：MediaParams +`mask?: string`。
- 新 `MaskBrushEditor.tsx`：props {src, onMaskChange}；内部 strokes/笔刷滑杆
  (12–96)/清除/「已涂 N 笔」状态；pointer 事件（down/move/up）收集点列；stroke
  结束或清除时导出并回调。
- `InstructionEditModal.tsx`：范围 Radio + 局部渲染编辑器 + run() 带 mask + 未涂
  warning。

## 7. 测试

- ai：①OpenAI edit+mask multipart 形状（灰度 PNG→转换后白区 alpha=0/黑区 255，
  httptest 捕获文件字节回读断言）②`grayMaskToOpenAIMask` 单元（白/黑像素 alpha）
③`kontextScaleSize` 单元（方图→1024²；2:1→1456×720；退化→1024²）④ComfyUI
  edit+mask 工作流形状（161/162/163 链+ImageScale 尺寸=kontextScaleSize+KSampler
  latent_image 接 163；无 mask 时零蒙版节点）。
- app：mask+img2img 拒绝；mask+edit 透传（fake 后端捕获 req.Mask）。
- 前端：MaskBrushEditor 交互状态（涂抹计数/清除回调 null）；Modal 局部未涂
  warning + mock 编辑器产 mask 透传断言。

## 8. 验收与门禁

- go build/vet、定向测试、tsc -b、eslint、vitest 全量、ci.ps1、drift PASS@714；
- 真机走查挂池：ComfyUI 档局部重绘端到端（用户闲置窗口；需 v4.392 的编辑权重）。

## 9. 观察池（本刀不做）

扩图（outpaint 画布扩边交互）；抠图/透明底 PNG 导出；img2img+mask 纯局部重绘；
蒙版反选/羽化半径；编辑历史变体簇（ParentID 链）；Lightning LoRA；多图编辑。
