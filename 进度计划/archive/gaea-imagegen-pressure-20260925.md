# 图像生成内存压力预检（iGPU 冷载预期管理）

> 2026-09-25 立项。来源：观察池「iGPU 压力预检」（v4.404.1 起四版在册）。
> Go 1 文件+3 挂点+前端 1 util+App 全局订阅，绑定面 717 零变更。
> 目标版本：v4.409.0。

## 1. 背景

- v4.404.1 分诊实录：iGPU 共享内存架构下权重被换出后冷载 20.5 分钟，客户端
  15 分钟上限先到误报「失败」——等待可见性与取消已收（v4.405~408），但用户
  提交时对「这单可能很慢」零预期。
- 预检=提交时读系统物理内存（iGPU 共享内存架构下 RAM 就是模型权重的实际
  载体），可用偏低即如实提示；只提示不拦截。

## 2. 落地

1. **internal/app/imagegen_pressure.go**：
   - `imageGenPressureNote(availMB, totalMB)` 纯函数：可用 < 总量 15% 或
     绝对值 < 4GB 命中；文案「系统可用内存偏低（X GB / 共 Y GB）——本地模型
     可能需冷载，本次生成等待会明显变长」。
   - `readSystemMemoryMB` 复用 image_handler 既有 GlobalMemoryStatusEx 原生
     采集（getMemoryStats，弃 wmic 教训在案）——不新增 API 面。
   - `noteImageGenMemoryPressure()`：命中→WARN 落日志（分诊对齐）+emit
     `imagegen:pressure` {note}；读不到静默跳过（预检是增益非硬依赖）。
2. **挂点**（仅 comfyui 后端，云端无冷载语义）：绘梦 generateImageInternal /
   GenerateMedia 两口 + 角色库 attachComfyProgress（剧照/设定卡共用）。
3. **前端**：utils/imagegenPressure.ts 节流 handler（2 分钟，空 note 忽略）；
   App.tsx 全局订阅一次（subscribeWailsEvent 唯一入口纪律）——绘梦/角色库/
   sin 全部提交口覆盖，连续提交不刷屏。

## 3. 测试

- Go TestImageGenPressureNote 六态表（正常/占比命中/绝对命中/贴线两侧/
  totalMB=0）；TestReadSystemMemoryMBWindows 真机读数合理性；TestNote
  ImageGenMemoryPressure 三态（命中恰一帧含文案/不命中零帧/读不到零帧，
  seam 注入+httpbridge SSE 捕获）。
- 前端 imagegenPressure.test 3 例（首提示/节流窗口+重置/note 缺失空白忽略）。
- 全量 ci 绿 + tsc 0 + eslint 0。

## 4. 门禁

全量 `scripts/ci.ps1` 绿 EXIT=0；版本漂移闸 OK@4.409.0；绑定面 717 零变更。

## 5. 观察池

评分 LLM 流式；sin 评分入口。
