# gaea-bloom029-glass-all-20261123-326d6eef

- 上游基准：`dsh-bloom-theme` `origin/main`
  `bd3e93f9a708d0d951156866a6f2274d0d33c824`（0.29.0）
- 功能提交：`b917ed6d`（全板块路由表面接入统一玻璃材质）
- 收口提交：`326d6eef`（补齐运行监测页头、KPI、图表与统计容器）
- 分支：`main`
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-326d6eef`，detached HEAD
- 前端：`npm run build`，Vite `37.19s`，6611 modules
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；
  `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，
  Wails `47.817s`
- 主树对照构建：Vite `33.48s`，Wails `28.69s`
- 漂移：Wails 再生成仅 CRLF/EOL 噪音，忽略 EOL 后内容差异文件数 `0`
- 冒烟：正式 payload `/api/health` 返回 `200`，PID `12336` 已回收

## 修复内容

- 一级板块路由根层不再绘制独立整页实底，统一透出 shell 背景
- `chat / office / wenxue / imagegen / cost / pantheon / atlas /
  modelcenter / characterlib / weixin / schedule / runtime`
  的主工作区统一消费 `--gaea-material-fill`、`--gaea-shell-sheen`
  与 `--gaea-material-filter`
- `v3-panel / v3-card` 在玻璃模式下同步接入统一材质契约
- 运行监测页头、KPI 区、图表卡、质量区和资源卡补齐玻璃材质
- 实底模式不加覆盖，继续保留各板块原有实底布局
- 减少透明度与强制对比模式继续通过材质 token 自动降级为实底

## 验证记录

- Vite 实际切换 12 个一级板块，根层均为透明；玻璃模式下不存在
  面积超过 10 万像素、仍在 70% 以上不透明度的残留主容器
- 三种材质实机计算值：
  - 毛玻璃：58% + `blur(24px) saturate(1.15)`
  - 磨砂玻璃：48% + grain + `blur(12px) saturate(1.02)`
  - 液态玻璃：25% + sheen + `blur(3px) url(#gaea-liquid-lens) saturate(1.2)`
- 亮色与暗色均验证路由根透明、主容器读取当前材质 token
- 包内 exe 实机切换文学、进度计划、运行监测等板块通过
- 定向回归：`AppearancePanel`、`appStore`、四层架构 3 文件 `31/31`
- 改动面 ESLint：0
- 截图：`C:\AI\wubigrok\temp\gaea-payload-glass-all-final.png`

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `AF66B6ED67C5B7242335D1C30FC2DD6441BFE1348EDDFFF8C561F0E21CD07EF1` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-bloom029-glass-all-20261123-326d6eef.zip | `F8B6B08DB993C7CAF62532131974AD95139FAC114AB60E1E336D1FACDDFB436D` |
| gaea-bloom029-glass-all-20261123-326d6eef-src.zip | `42741D98498D21EE1E0F4A3F5D090AB012998461786464288D042D071B97A5A4` |
