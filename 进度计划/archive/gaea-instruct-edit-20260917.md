# gaea 绘梦·阶段一刀 C：指令编辑「改图」MVP（云端先行）规格书

> 2026-09-17 立项。来源：用户指令「继续」。规格 docs/gaea-dream-studio-nextgen-
> 2026-09.md §4 阶段一刀 C；阶段七规划 §4 在册（一致性选型对齐调研共识）。
> 前置核对：刀 A 画室资产面板已落（AssetStudio 三槽）；刀 B 一致性参考槽已落
> （RefImages/RefMethod，img2img 可用 ipadapter/pulid 诚实拒绝）；刀 E 模型目录
> 分层已落（ModelDirectory 能力族）。**刀 C 指令编辑是真缺口**——现状「改图」
> 按钮=把结果填回图生图参考图（整幅重绘），没有「原图+人话指令」的语义编辑。
> 目标版本：v4.327.0。

## 1. 论点

调研共识（§1.1/§1.2）：编辑能力是旗舰标配、角色一致性入场券。gaea 后端
已有 OpenAI 兼容图片面（/images/generations + /images/img2img），但缺
**/images/edits**（原图+指令 → 局部语义编辑）——这正是 Qwen-Image-Edit 系
云端网关的标准暴露面。本刀补齐：模式 `"edit"` 全链（ai 后端 → GenerateMedia
→ 结果卡「指令编辑」弹窗）。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 复用 `ImageGenerationRequest`：`Mode="edit"` + `InitImage`（原图 data URL）+ `Prompt`（人话指令）——**零新请求字段** |
| Q2 | OpenAI 兼容后端实现 edit：multipart POST `{baseURL}/images/edits`（model/prompt/n/size + image 文件），响应解析复用既有 b64/url→dataURL 链 |
| Q3 | GLM：官方端点无 edits → 诚实报错；ComfyUI：本地档未接 → 诚实报错（B 计划）；**app 层不设后端门**（透传，按后端诚实报错——img2img 的门是历史裁决不动） |
| Q4 | **零新绑定**：GenerateMedia 既有 paramsJSON 加 mode="edit"；绑定面 705 不变 |
| Q5 | 前端 V1=InstructionEditModal：原图预览 + 指令输入 + 生成 + 原图/新图对照 + 「用到画布」（并入 results/history，走既有保存/台账链）；编辑产物已由后端落盘登记（ImageHub sourceBoard=imagegen） |
| Q6 | 保留区域/mask 不做（观察池）；成本显示沿用结果卡 model/time 元数据（编辑结果同构） |

## 3. 线 A：internal/ai

- `image_openai.go`：
  - 抽 `parseImageResponse(ctx, respBody)`（既有 JSON 解析 + 相对/绝对 URL →
    data URL 下载逻辑原样提取，edit/generation 两分支共用）；
  - `GenerateImage` 加 `req.Mode == "edit"` 分支：InitImage 必须 data URL
    （否则诚实报错）→ multipart（model/prompt/n/size 条件携带 + image 文件
    名按 MIME 推断）→ POST edits → parseImageResponse。
- `image_glm.go`：edit 模式并入既有拒绝（错误文案补「指令编辑」）。
- `image_comfyui.go`：`comfyResolveRefMode` 前置 edit 拒绝（「指令编辑暂未
  接入本地档（规划中），请使用云端 OpenAI 兼容引擎」）。
- 测试：edit multipart 形状（httptest 断言路径/字段/文件名/内容）、缺原图
  报错、GLM/ComfyUI 拒绝文案。

## 4. 线 B：internal/app GenerateMedia

- `mode == "edit"`：InitImage 必填校验（「指令编辑需要原图」）+ prompt 既有
  校验；**不加后端门**（Q3）。t2v/img2img 既有门零变化。
- 测试：edit 缺图报错；edit 请求透传（mock 后端不必——ai 层已测，app 层只
  钉校验分支）。

## 5. 线 C：前端

- `api/image.ts`：MediaParams.mode 联合类型 + `'edit'`。
- 新 `InstructionEditModal.tsx`：props {open, source: GenResult, onClose,
  onApply(r: GenResult)}——原图预览 + 指令 TextArea + 生成（generateMedia
  {mode:'edit', initImage, prompt:指令, model 沿用源图}）+ 对照区（原图|
  新图并排）+ 用到画布（onApply）/失败 error 展示。
- `ResultStage.tsx`：+`onInstructEdit?: (index) => void`，结果卡与灯箱
  Action「指令编辑」（与既有「改图」并列，icon 区分）。
- `ImageGenPage.tsx`：editSource 状态 + 弹窗接线；onApply=setResults 前插 +
  setHistory 前插（queue.ts 107-108 同款镜像）。
- 测试：Modal（指令生成/对照/应用回调/错误态）+ ResultStage 按钮渲染派发。

## 6. 验收与门禁

- ai 三后端分支测试 + app 校验测试 + 前端组件测试全绿；
- go build/vet、tsc -b、eslint、vitest 全量、ci.ps1；
- 出口：结果卡「指令编辑」→ 指令 → 对照 → 用到画布全链可达（真机走查挂池）。

## 7. 观察池（本刀不做）

mask/保留区域；ComfyUI 本地档（Qwen-Image-Edit/FLUX Kontext 工作流）；多图
编辑输入；编辑历史谱系（edit-of-edit 链）；成本单价表（刀 E 余项）。
