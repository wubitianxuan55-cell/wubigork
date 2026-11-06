# gaea 绘梦·阶段二刀 E：抠图/透明底导出（T3 收官）规格书

> 2026-09-23 立项。来源：用户指令「继续」（画室线：T3 最后一件）。
> 上游：长期规划 T3「蒙版局部重绘 + 扩图 + 抠图（canvas 圈选/笔刷；透明底 PNG
> 导出，对齐 NovelAI V5 原生透明卖点）」，验收「透明底立绘可导出并回填角色库」。
> 前置：蒙版涂选组件（v4.393 MaskBrushEditor）与统一灰度蒙版契约已就位。
> 目标版本：v4.397.0。

## 1. 论点

抠图按规划原意走「涂选 + 透明底导出」：**零新模型零引擎调用**——蒙版即 alpha
通道（涂选保留主体 → 主体区不透明原像素、其余透明），纯 Go 图像合成。规划
验收的「回填角色库」拆两步：本刀交付导出+台账登记（素材库可见）；角色库参考图
管线对接挂观察池（characterlib 上传链是另一刀的 UX 面）。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | **抠图蒙版语义：白=保留主体、黑=背景（转透明）**——与编辑的「白=重绘」相反，两个原语各自契约文档化；前端涂选文案明示「涂抹要保留的区域」。MaskBrushEditor 复用（其导出即灰度蒙版，语义由消费方定义） |
| Q2 | `ComposeCutout(initImage, mask)`（internal/ai 新文件 image_cutout.go）：decode 原图+蒙版 → RGBA 输出（白区原像素 alpha=255、黑区 alpha=0）→ PNG data URL。蒙版与原图尺寸不一致→报错（前端同画布构造性成立）；边缘硬切（羽化半径挂观察池） |
| Q3 | **新绑定 `ImageCutout(initImage, mask)`**（ImageB 门面，绑定面 714→715，play）：ComposeCutout → 落盘 ImageSaveDir（复用 saveMediaToDisk）→ 台账登记（v4.396 修复后闸正常；params.mode="cutout"）→ 返回 `{path, assetId}`。非模型调用不设引擎门 |
| Q4 | 前端：ResultStage 结果卡新动作「抠图」（与「改图」「指令编辑」并列）→ CutoutModal（MaskBrushEditor 涂选 + 「导出透明 PNG」+ 成功显示路径）；未涂 warning。导出产物不进画布 results（透明图在结果流的展示/历史语义未定——挂观察池） |
| Q5 | spaceBindings：ImageCutout=play；数量锁 537→538；bindingNames 再生 715 |

## 3. 线 A：internal/ai（image_cutout.go 新文件）

`ComposeCutout` 纯函数+契约注释（白=保留）。标准库 image/draw，零新依赖。

## 4. 线 B：internal/app

- `mediaState.ImageCutout(initImage, maskData string) map[string]interface{}`：
  校验（空参/decode 失败如实报错）→ ComposeCutout → saveMediaToDisk →
  recordImageHubGeneratedFor（sourceBoard=imagegen，params 带 mode=cutout）→
  返回 {path, asset_id}（asset_id 由 byPath 回填，v4.395 链路复用）。
- ImageB 门面委托。

## 5. 线 C：前端

- `api/image.ts`：appFacade 加 ImageCutout 声明 + `imageCutout()` 封装。
- 新 `CutoutModal.tsx`：props {open, source, onClose}；MaskBrushEditor（文案
  「涂抹要保留的主体」）+ 导出按钮（未涂 warning）+ 成功态（路径+提示素材库
  可见）。
- `ResultStage.tsx`：+`onCutout?: (index) => void` 动作；`ImageGenPage.tsx` 接线。

## 6. 测试

- ai：ComposeCutout 单元（白区原像素+alpha255/黑区 alpha0/尺寸不一致报错/非
  data URL 报错）。
- app：ImageCutout 绑定（闸开：落盘+台账+返回 path/asset_id；闸关：落盘仍出
  path 但台账空=字段空；空 mask 报错）。
- 前端：CutoutModal（涂选→导出参数 mask 透传/未涂 warning/成功态路径显示）；
  ResultStage 抠图按钮派发。

## 7. 验收与门禁

定向+全量 ci.ps1、tsc/eslint/vitest、drift PASS@715、spaceBindings 538；
真机挂池（涂选导出透明 PNG 后在资源管理器验 alpha）。

## 8. 观察池（本刀不做）

羽化半径/边缘平滑；角色库参考图管线对接（回填立绘）；模型推理抠图
（BiRefNet/RMBG 类，需新权重）；透明产物进画布/历史的展示语义；抠图反选。
