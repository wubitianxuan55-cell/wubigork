# gaea-bloom029-final-20261123-555eda3e

- 上游基准：`dsh-bloom-theme` `origin/main`
  `bd3e93f9a708d0d951156866a6f2274d0d33c824`（0.29.0）
- 交付提交：`555eda3e3064bb20d860ef7e2b58ed14418635df`
  （`fix(gaea): preserve Bloom meteor color streams`）
- 分支：`main`
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-555eda3e`，detached HEAD
- 前端：`npm run build`，Vite `27.41s`，6611 modules
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；
  `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，
  Wails `55.951s`
- 主树对照构建：Vite `28.37s`，Wails `34.434s`；
  `build\bin\gaea.exe` SHA256 `7A29D84C6678E530ADBBCC92EF821C07916AD86474FA3D4A78AF42F64F784221`
- 漂移：Wails 再次生成仅 CRLF/EOL 噪音，忽略 EOL 后内容差异文件数 `0`
- 冒烟：正式 payload `/api/health` 返回 `200`，PID `68940` 已回收

## 核心修复

- 星尘挂到 `MainLayout` 根壳的独立 stacking context，修掉业务背景盖住星河的问题
- 应用内“减弱动态”保留 270 颗静态可见星点，只隐藏流星
- 彩色流星 `.m2/.m3` 选择器提升到与基础规则同级的完整作用域，
  `colorful` 恢复主色 / 青 / 紫三束光谱，`solid` 继续统一为当前强调色
- 流星密度 20 / 40 / 60 与速度 `1.6 / 1 / 0.6` 在真实桌面进程中验证生效

## 对齐验证

- Vite 与 `build\bin\gaea.exe` 都在 `1264×761` 下逐项读取实际计算令牌
- 15 套主题 × 亮 / 暗共 30 组：主色、主色容器、画布、容器、高容器、
  三束 motion 色全部一致
- 3 种表面材质：`veil / matte / soft` 的填充、滤镜、sheen 全部一致
- 4 种暗色氛围：`corner / diagonal / galaxy / off` 的尺寸、遮罩、背景图与显示态全部一致
- 4 种亮色氛围：`stream / wash / clear / off` 的背景图、fog mask 与 glaze 全部一致
- 动效：减弱时星点静态可见且流星隐藏；完整时 270 星运动、标准密度显示 40 流星
- 流星：彩色为 `#7faee9 / #50b9df / #ac9fe5` 三束，纯色全为当前 accent；
  密度 `20 / 40 / 60`；速度倍率 `1.6 / 1 / 0.6`
- 定向回归：`AppearancePanel`、`appStore`、四层架构 3 文件 `30/30`
- 截图：
  `C:\AI\wubigrok\temp\gaea-desktop-final-aligned.png`
  `C:\AI\wubigrok\temp\gaea-vite-final-aligned.png`
  `C:\AI\wubigrok\temp\gaea-desktop-bloom029-settings-final.png`

## 差异说明

- Vite 读取当前源码并受 HMR 控制；桌面 exe 是构建时固化进二进制的快照。
- 本机曾出现 Vite 同秒编辑缓存未失效，实际 DOM 仍执行旧流星模块；已通过刷新 mtime
  并重新验证解决。
- 桌面 profile 独立于 Vite profile。此前桌面残留自定义强调色 `#1dd7bf`，
  而 Vite 使用青金主题色 `#7faee9`；已恢复为跟随主题。两者现已统一为
  青金 / 暗色 / 液态玻璃 / 漫天星辰 / 完整动效 / 彩色标准流星。
- 真实桌面后端与 Vite `?mock=1` 的会话内容、引擎状态和侧栏展开态仍会不同，
  不属于皮肤或冻结构建差异。

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `274BE4A611536E5810C76C47EF215735FBC06F36F6997274C19C99B9A0172036` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-bloom029-final-20261123-555eda3e.zip | `6C0CAF53CC8E5CB4E8F8D7DDF583903232010B6215BB0792E5D4E05F51A87B45` |
| gaea-bloom029-final-20261123-555eda3e-src.zip | `FC1B741E67FC951695279AADFBFAD33F21E8602500B258ECC55CE25FF984CFAD` |
