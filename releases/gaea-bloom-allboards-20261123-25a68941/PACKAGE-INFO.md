# gaea-bloom-allboards-20261123-25a68941

- 交付提交：`25a6894162368fb19381142d2aa6a93083f7960f`
  （`fix(gaea): apply shell glass to every wenxue composer`）
- 基础提交：`bdc03895146540cc49a53cf4835bb4948c1544c8`
  （`fix(gaea): preserve starlight and glass across boards`）
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-25a68941`，detached HEAD
- 前端：`npm.cmd run build`，Vite `31.99s`，6614 modules
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；
  `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，
  Wails `48.596s`
- 漂移：Wails 再生仅 CRLF/EOL 噪音，`git diff --ignore-cr-at-eol` 内容差 `0`
- 冒烟：正式 payload `/api/health` 返回 **200**

## 交付内容

- 办公活动页 `.gaea-app-layout` 从路由材质 `:is()` 列表移除，避免整页
  fill/sheen/blur 压在星河上方
- 非实底模式新增高特异性 `.page-enter .gaea-app-layout.ant-layout`
  透明通道，覆盖文学等板块懒加载 CSS 后置写入的 aurora 实底
- 星尘继续位于内容背后（本地隔离壳内 `z-index:0`，对应上游 `-1`），
  没有新增冻结流星或假动画
- 文学所有会话状态的 `.office-composer-card`、输入行、工具条与聚焦态
  统一消费 shell composer 材质
- 聊天 `.chat-composer.gaea-glass-shell` 统一消费同一组 shell composer
  材质；保留聊天自身 760px 输入岛与 18px 圆角
- 架构测试新增 app layout 透明通道与文学/聊天 composer 材质守卫

## 验证

- 定向 `architecture/AppearancePanel/appStore`：**3 文件 32/32**
- `tsc -b`：0；改动面 ESLint：0
- 真实用户配置启动最终 payload：`prefers-reduced-motion=no-preference`，
  40 条流星正常显示
- 最终 payload 已覆盖安装，`INSTALL OK`，PID `16048`，启动时间
  `14:17:10` > 构建时间 `14:14:54`，装机 exe 与 payload 同哈希
- 文学活动 `.gaea-app-layout`：背景、背景图、滤镜全部透明
- 办公、文学（含已选会话）、聊天输入面逐项计算样式相等：
  `background-color`、`background-image`、`backdrop-filter`、
  `box-shadow`、`border`
- 当前 veil 材质实测三板块均为 58% 填充 +
  `blur(24px) saturate(1.15)`
- `bdc03895` 预备包在“文学已选会话”状态仍为实底，已作废，不作交付

## 包布局

- `payload/gaea.exe`
- `payload/onnxruntime.dll`
- `payload/sherpa-onnx-c-api.dll`
- `payload/sherpa-onnx-cxx-api.dll`
- `gaea-bloom-allboards-20261123-25a68941.zip`：上述四件运行文件
- `gaea-bloom-allboards-20261123-25a68941-src.zip`：
  `git archive` 自交付提交的源码包

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `3F8CE4D9F63A4468E0B706A0ECBC9E3A30C9450F7BB8A86FC1D49BE4F9F7C809` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-bloom-allboards-20261123-25a68941.zip | `504451BC7112D07E9B3BE62473234535745857E9BD2035B4B6572ECE5C444826` |
| gaea-bloom-allboards-20261123-25a68941-src.zip | `25B409A6423433742ED283CB2E6DE91E6BF0DCDFF5C3411D4CF3AECAC45D61F9` |

## 边界

- 未 push。
- 日常预览包，不抬产品版本。
