# gaea 绘梦·阶段三刀 A：参考槽 Qwen-Edit 通道（T2 一致性首刀）规格书

> 2026-09-23 立项。来源：用户指令「继续」（画室线转 T2 一致性）。
> 上游：长期规划 T2「一致性参考槽：参考图入生图请求；ComfyUI 工作流按模型族
> 注入（krea2/z-image 先用 img2img 近似；flux 上 IP-Adapter/PuLID）」。
> **路线修订（本刀核心判断）**：gaea 模型族（krea2/z-image，Qwen-Image 架构）
> 没有 IP-Adapter/PuLID 生态——但 **Qwen-Image-Edit 2511 原生支持 3 图参考槽**
> （TextEncodeQwenImageEditPlus 的 image1/2/3，v4.392 工作流只用 image1）。
> 一致性正路=「参考图+指令→编辑引擎」而非装 IP-Adapter。目标版本：v4.398.0。

## 1. 论点

现状参考槽是 v0 近似：**img2img 整幅重绘**（denoise 0.65，带参考图构图跑）——
「角色一致的新场景」做不对（构图被参考图锁死）。且队列在 txt2img 时丢参考
（useImageGenQueue 241：refImages 只在 img2img 模式带）——「描述新场景+角色参考」
根本无通道。Qwen-Image-Edit 的多图参考正是为此训练（官方模板：image1=沙发
image2=皮草+指令=换材质）；复用 v4.392 工作流加 image2/3 槽=零新权重零新引擎。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | `comfyResolveRefMode` 新增 `refMethod="qedit"`：txt2img+参考 → 新 mode `qedit`；`img2img` 模式照旧（本身有图）。ipadapter/pulid 拒绝文案改口（指向 qedit：Qwen 架构族无 IP-Adapter 生态，正路是编辑参考） |
| Q2 | `buildQwenImageEditWorkflow` 签名改为 `images []string`（images[0]=原图/参考1，[1]/[2] 可选进 image2/3）——正/负 TextEncodeQwenImageEditPlus 都接全部参考槽（官方模板口径）；edit 场景传 [原图]；qedit 传参考 ≤3 张（超出如实报错「Qwen 参考最多 3 张」） |
| Q3 | qedit 的 latent/尺寸：FluxKontextImageScale(参考1) → VAEEncode（与 edit 同款）——输出尺寸跟随参考图；KSampler 同 edit 参数（20/4.0/1.0）。蒙版不与 qedit 组合（fail-closed 报「参考编辑不支持蒙版」） |
| Q4 | app 层：GenerateMedia 在 txt2img 也透传 refImages/refMethod（解除「仅 img2img 带」的现状限制——零新字段）；comfyui 元数据如实化扩展：qedit+comfyui → qwen-image-edit（mode override 条件加 refMethod=="qedit"） |
| Q5 | 前端：ControlPanel「人设参考」区加一致性方法选择（Radio：**图生图近似**=现状默认/**Qwen 参考编辑**=新）；qedit 时提示「prompt 描述新场景，人物形象与参考保持一致」且不强制切 img2img；queue hook：task.refMethod 透传+txt2img 带参考（仅 qedit 时——img2img 近似保持既有切换行为） |
| Q6 | sin 侧不动（sinRefPlan 走 img2img 判定现状保留——升级候观察池）；GLM/OpenAI 后端 qedit：OpenAI 兼容 edits 端点吃单图+指令——qedit 在 OpenAI 后端走 edits（InitImage=ref1）？**不做**——本刀 ComfyUI 专供，OpenAI 后端 qedit 诚实报「Qwen 参考编辑当前仅 ComfyUI 本地档」（观察池：云端多图编辑端点成熟后接） |
| Q7 | 零新绑定（715 不变） |

## 3. 线 A：internal/ai

- `comfyResolveRefMode`：qedit 分支（mode=="qedit" 或 refMethod=="qedit" 且
  mode 为 txt2img → "qedit"；mode=="img2img" 优先返回 img2img）。
- `GenerateImage`：edit 分支重构为 edit/qedit 共用——qedit：InitImage 可空
  （RefImages 必填）、上传 refs、Mask 必空；switch case "qedit" →
  buildQwenImageEditWorkflow(prompt, seed, uploadedRefs, "", 0, 0)。
- builder 签名改造+两处调用点（edit/qedit）适配。

## 4. 线 B：internal/app

- GenerateMedia：refMethod=="qedit" 时 comfyui 元数据 qwen-image-edit；
  txt2img 透传 refImages/refMethod（ai 层自行路由）。

## 5. 线 C：前端

- `useImageGenQueue`：task 加 refMethod 透传；txt2img+refMethod==qedit 时带
  refImages+refMethod。
- `ControlPanel`：人设参考区+一致性方法 Radio（img2img/qedit）；qedit 时不切
  img2img；文案更新（v0 近似说明→两法并列说明）。
- `ImageGenPage`：refSlot 状态带 refMethod 传 queue。

## 6. 测试

- ai：①comfyResolveRefMode qedit 矩阵（txt2img+qedit→qedit/img2img 模式优先
  img2img/ipadapter 改口文案）②qedit 工作流形状（httptest：image1/2/3 进正负
  TextEncode、latent=VAEEncode(ref1 缩放)、无蒙版节点；>3 参考报错；qedit+mask
  拒绝）③edit 回归（单图槽不受改造影响——既有形状测试钉）。
- app：txt2img+refs+qedit 透传（fake lastReq）；qedit+comfyui 元数据。
- 前端：ControlPanel 方法选择渲染+qedit 不切模式；queue 透传断言。

## 7. 验收与门禁

定向+全量 ci.ps1、tsc/eslint/vitest、drift PASS@715；真机挂池（角色参考+
新场景指令出一图，人物一致性目检）。

## 8. 观察池（本刀不做）

sin 参考槽升级 qedit；云端多图编辑（OpenAI 端点成熟后）；角色资产 v2（三视图/
一致性评分）；LoRA 训练向导；导演 Agent；参考权重（per-image weight——2511
模板未暴露，等上游）。
