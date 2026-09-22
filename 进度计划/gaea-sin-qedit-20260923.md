# gaea 绘梦·阶段三刀 B：原罪插图参考升级 qedit（T2 第二消费方）规格书

> 2026-09-23 立项。来源：用户指令「继续」（T2 余项按复利优先：v4.398 观察池
> 头名）。上游：v4.398 qedit 通道（Qwen-Image-Edit 三图参考槽）。目标版本：
> v4.399.0。**小刀**：sinRefPlan 单点升级+矩阵更新，Go 2 文件。

## 1. 论点

原罪插图的角色一致性（v4.258）走 img2img 整幅重绘近似——构图被参考图锁死，
「按场景描述出图+人物一致」做不对。v4.398 的 qedit 通道正是为此而生（参考图
进编辑引擎 image1..3，prompt 即场景描述）；原罪插图是天然的第二个消费方
（插图 prompt 本来就是场景描述+点名角色），一次单点升级让在用板块吃到 T2 代差。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | `sinRefPlan` comfyui 分支：`return "txt2img","qedit",true,""`——qedit 走编辑引擎（req.Model 不消费），**不再按 krea2/z-image 判型**（flux 等生图模型下插图参考也可用）；herdsman 维持 img2img 近似（无编辑引擎）；文案更新 |
| Q2 | 编辑权重缺失由 **sin_handler 既有 refFallback 链兜底**（生成失败→退纯文本重试+如实标注 ref_fallback）——无权重环境插图不断流，只是丢参考降级，行为可接受不加新门 |
| Q3 | denoise 参数照传无害（qedit 分支固定 1.0 不读）；文本锚点/角色选择/多角色互串防护（v4.258 纪律）全部不动；caption 徽标「角色参考:名字」语义不变不改动 |
| Q4 | 零新绑定（715 不变）；纯 Go 2 文件 |

## 3. 测试

- 矩阵更新：comfyui 全模型 → (txt2img, qedit)（含 flux-dev——编辑引擎不受生图
  模型限制）；refMethod 断言补强（backendWants）；透传测试期望更新
  （mode=txt2img+method=qedit）；TestSin 全量回归绿。

## 4. 门禁

定向+全量 ci.ps1、drift PASS@715。

## 5. 观察池

sin 参考方法可配置（qedit/img2img 二选一 UI）；qedit 失败回退 img2img 近似
（两级回退）；caption 徽标区分方法；sin_illustrate 工具透传场景描述优化
（配合 qedit 语义的 prompt 模板）。
