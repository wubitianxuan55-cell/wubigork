# gaea 绘梦·阶段二刀 A：指令编辑本地档（ComfyUI · Qwen-Image-Edit 2511）规格书

> 2026-09-22 立项。来源：用户指令「开始优化」（会话主线：乐园画室欠账盘点 → 画室三件套
> 头号 = 编辑力补全）。上游规格 `进度计划/gaea-instruct-edit-20260917.md`（阶段一刀 C
> 云端先行）§7 观察池第 2 项「ComfyUI 本地档」即本刀；长期规划
> `docs/gaea-image-domain-longterm-plan-2026.md` T3「本地档（B 计划）」。
> 前置核对：用户本机全局生图后端 = comfyui（`~/.gaea_config.json` image_backend），
> 即**编辑在日用后端上此前恒被拒**（「请使用云端 OpenAI 兼容引擎」）——本刀让编辑
> 在本地闭环。目标版本：v4.392.0。

## 1. 论点

调研共识（长期规划 §6 / 竞品格局）：编辑能力是「从重抽到修图」的范式分水岭。刀 C
已落云端（OpenAI 兼容 `/images/edits`），但 gaea 用户主力是本地优先（模型中心 comfyui
档）。ComfyUI 官方模板库（comfyui-workflow-templates 0.11.62，本机
`C:\AI\ComfyUI\ComfyUI\.venv\...\templates`）提供 `image_qwen_image_edit_2511.json`
官方编辑工作流——蒸馏纪律「取道不取器」：按官方节点图在 gaea 侧以 API-format 工作流
构建器实现（与 krea2/z-image 既有构建器同范式），不搬模板文件不依赖 subgraph。

## 2. 蒸馏源（官方模板还原，2026-09-22 实读）

`image_qwen_image_edit_2511.json`（subgraph 展开后的 API 图）：

- `UNETLoader`：`qwen_image_edit_2511_fp8mixed.safetensors`（bf16 同源可替换）
  → `ModelSamplingAuraFlow(shift=3.1)` → `CFGNorm(strength=1, pre_cfg=false)` → KSampler
- `CLIPLoader`：`qwen_2.5_vl_7b_fp8_scaled.safetensors`，type=`qwen_image`
- `VAELoader`：`qwen_image_vae.safetensors`（**与 krea2 共用，本机已有**）
- `LoadImage` → `FluxKontextImageScale`（按原图宽高比重标，编辑不套 1024×1024）
  → ① `TextEncodeQwenImageEditPlus`(正/负，image1=缩放图, vae) ×2
  → ② `VAEEncode`(pixels=缩放图) → `KSampler.latent_image`
- `KSampler`：steps 20 / cfg 4.0 / euler / simple / **denoise 1.0**（官方 Note
  「Comfy」列：20 步 CFG 4.0；Qwen 原生 40 步）
- `VAEDecode` → `SaveImage`
- `FluxKontextMultiReferenceLatentMethod` 两节点：官方 Note 明示「用 Comfy 官方权重
  不需要」——不接。
- Lightning 4 步 LoRA：官方模板可选加速件——v1 不接（观察池）。

## 3. 裁决

| # | 裁决 |
|---|---|
| Q1 | ComfyUI `mode="edit"` 从「诚实拒绝」改为真实现：单编辑族 = Qwen-Image-Edit 2511（中文编辑最强 + VAE 与 krea2 共用）。req.Model 在 edit 分支被忽略（请求里的 model 是生图模型名，不代表编辑引擎） |
| Q2 | **缺权重不预检不拦截**：gaea 从不自动下载模型；缺文件时 ComfyUI 回 `value_not_in_list`，edit 分支在提交错误上追加可操作中文提示（三件文件名 + HF 链接 + 存放目录）。长期规划「按显存档位默认关闭」在此简化为「手动放权重即启用」——无自动拉起即无默认拖垮风险 |
| Q3 | app 层 GenerateMedia：`edit + comfyui` 时 `imgModel` 如实改写为 `qwen-image-edit`（结果卡/台账元数据诚实——实际跑的就是编辑引擎，不是 krea2） |
| Q4 | 尺寸：edit 不消费 `width/height`（FluxKontextImageScale 按原图宽高比重标）；app 层默认 size 照传不生效，无害 |
| Q5 | denoise 固定 1.0（语义编辑，非整幅重绘），忽略请求 Denoise；LoRA 链不进 edit（Lightning 留观察池） |
| Q6 | 零新绑定（绑定面 714 不变）：edit 走既有 GenerateMedia mode 通道 |
| Q7 | 前端仅改 InstructionEditModal 底部说明文案（本地档已支持 + 缺权重行为），零交互变更 |

## 4. 线 A：internal/ai/image_comfyui.go

- edit 分支：InitImage 必填（缺图报「指令编辑需要原图」）→ `uploadImage` →
  `buildQwenImageEditWorkflow(prompt, seed, imageName)` → 既有 queuePrompt/waitForResult
  链（进度回调/取消/轮询全继承）。
- 提交失败且 edit 模式 → 错误追加 `qwenEditMissingModelHint`（匹配 value_not_in_list
  / 模型名时给出三件文件 + HF 链接 + 目录；不匹配返回空串零影响）。
- 常量集中：`qwenEditUNET/qwenEditCLIP/qwenEditVAE` 单点维护（后续可配置化的落点）。

## 5. 线 B：internal/app/image_handler.go

- GenerateMedia 循环内：`mode=="edit" && a.cfg.ImageBackend=="comfyui"` →
  `imgModel = "qwen-image-edit"`（仅元数据如实化；ai 层本就不读该字段）。

## 6. 线 C：前端 InstructionEditModal.tsx

- 底部说明改口：「云端走 OpenAI 兼容 /images/edits；本地 ComfyUI 走 Qwen-Image-Edit
  2511（需在 ComfyUI models 目录放置模型文件，缺失时错误会列出所需文件与下载地址）；
  GLM 暂不支持」。

## 7. 测试

- ai 层：① edit 工作流形状（httptest：/upload/image → /prompt 捕获工作流 →
  /history → /view；断言 TextEncodeQwenImageEditPlus×2 吃 FluxKontextImageScale 图、
  VAEEncode 同源、KSampler 20/4.0/1.0、UNET/CLIP/VAE 文件名与 type、CFGNorm+AuraFlow
  链）；② 缺原图报错；③ 缺模型提示（value_not_in_list → 提示含文件名+HF 链接；
  普通错误不追加）。
- app 层：edit+comfyui 模型改写断言（既有 GenerateMedia 测试面扩展）。
- 前端：InstructionEditModal 说明文案断言更新（如既有测试触及）。

## 8. 验收与门禁

- go build/vet、ai+app 测试绿、tsc -b、eslint、vitest 全量、ci.ps1、drift PASS@714；
- 真机走查挂池（用户闲置窗口）：comfyui 档编辑弹窗 → 缺权重错误列出三件文件
  （本机无 qwen_image_edit 权重，预期即提示路径）。

## 9. 观察池（本刀不做）

Lightning 4 步加速 LoRA；image2/image3 多图编辑（刀 C 观察池同项）；蒙版局部重绘/
扩图/抠图；编辑历史变体簇（ParentID 链）；模型文件名可配置化（config 键）；FLUX.2
Klein 编辑族第二档；编辑产物自动回填角色库。
