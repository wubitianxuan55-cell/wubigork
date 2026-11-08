# 当日总账（2026-11-08）：编程板块回流 + 记忆图谱真 3D 脑网格 + 语音星团随声互动

> 批末=本笔（release：wails.json 4.535.0+CHANGELOG+day-log+README 登记）。桌面 exe=
> build/bin/gaea.exe 61,556,736B，SHA256 前缀 **20D0B620**（C:\AI\gaea-app 桌面副本同
> 哈希 20d0b6200cd14c2e 已核，壳 PID 28968 装机即起）。今日三笔线程刀：
> **0c9a234a**（编程板块，35 文件）、**c97239eb**（记忆图谱真 3D 脑网格，10 文件）、
> **c3cc7d98**（语音星团随声互动，11 文件）。均未 push（本地 ahead 20），随下版列车。
> 产线分工注记：编程板块主体（DSH Web 隔离+REA+插件迁移+GPU/comfyui 六处 HideWindow
> 统一）为 DSH 执行体当日完成并已按 4.534.0 装机；本批 4.535.0 在其上补编程链路实弹
> 漏网（tasklist 探针）+两前端线。

## 提交索引（3+1 笔）

| 提交 | 线 | 一句话 |
|---|---|---|
| 0c9a234a | 编程板块 | 板块回流独立一级（14 板块 id=code，space=independent=foot 单列）+隔离 DSH Web 运行时五绑定（token 捕获/preflight 八查/JobObject 定点杀）+cmd 黑窗统一根修（裸 SysProcAttr 先显窗再隐→proc.HideWindow 六处+tasklist 探针补漏+回归钉）+专项 docs+.gaea 记录 |
| c97239eb | 记忆图谱 | 中央星图换真 3D 脑网格：NIH 3D 3DPX-021161（CC BY 4.0）18,511 顶点 X3D 离线解析，three.js WebGL Fresnel 脑壳+纤维，360° 旋转/缩放/Raycaster 点星，rAF≈57fps，three 独立懒加载 chunk |
| c3cc7d98 | 语音台 | 星团随声互动：采集 worklet 50Hz RMS（原 4Hz 台阶）+播放 worklet 20Hz 声卡实际外放统计（原 delta 时刻提前起跳），Orb 包络平滑+峰值衰减，speaking 取播放/麦克风较大值；VoiceMemoryBrainLayer 声核复用记忆图谱脑网格 |
| 本笔 | release | wails.json productVersion 4.534.0→**4.535.0**（版本真源）+CHANGELOG 补编程板块条目+本 day-log+README 登记 |

## 编程板块要点（0c9a234a）

- 隔离边界：`C:\AI\dsh` 根，home/agentsHome 必须within 根（preflight isolated 查）；
  3091 端口被非 gaea 实例占用时拒绝启动/拒绝误杀。REA 6.8.0 MCP 229 工具、
  桌面 9 插件迁 Web profile 8 bundle 启用；dshmarket@1.73.15 Windows restart
  helper 漏 windowsHide 已 profile 内补丁（上游跟踪挂 todos）。
- 版本漂移注记：`app_info.go` AppVersion=4.531.0 属恒空遗留（GetVersionInfo 不消费），
  版本真源=wails.json productVersion（13b3493f 记档），勿再「修」AppVersion。

## cmd 黑窗根修要点（全树口径）

- 机理：GUI 进程 spawn 控制台子系统 exe（tasklist/netstat/powershell/reg）必分配
  可见控制台；`SysProcAttr{HideWindow:true}` 单挂仍先显窗再隐，必须叠加
  CREATE_NO_WINDOW——统一走 `internal/gaea/proc.HideWindow`（它两者都设）。
- 漏网形态=「活探针」：启动路径挂了 HideWindow 不够，**状态轮询路径的存活探针
  也要挂**（本例 tasklist 每 3.5s 一闪，启动后才开始=与板块启动时序强相关，
  容易误判成启动器残留）。回归钉=TestProgrammingWebPIDProbeHidesWindow。
- 教训落点：新增 exec.Command 一律过 proc.HideWindow；grep 全树裸
  `SysProcAttr{HideWindow}` 应为 0（gpu_perf/image_comfyui 六处已清）。

## 记忆图谱 / 语音台要点

- 脑网格数据=brainMeshData.ts 内嵌二进制（base64→解码器 brainMeshDecoder.ts），
  License=CC BY 4.0 必须随署名（显示设置内+文件头）。
- 语音电平面新契约：micLevelRef 50Hz / playLevelRef 20Hz，画布逐帧读 ref 不走
  React state（state 仍 60ms 节流供非画布消费）；liveLevel=播放与麦克风较大值。
- 前端 chunk 纪律：three.js 网格经 React.lazy 独立 chunk（MemoryAtlasStage
  1,022KB），首屏壳不背 3D；GraphView 3d-force-graph 仍独立。

## 验证矩阵

- Go：`go build ./...` 全仓 + `internal/app/board` 包 + `internal/app` 定向
  （TestProgrammingWeb*/TestExtractProgrammingWebToken/TestBoardManifest*/
  TestBoardIdSingleSource）CGO 真跑（ucrt64 gcc+build/bin DLL 前置 PATH）
  全 PASS；TestProgrammingWebLiveStart=env 门控 SKIP 非红。
- 前端：`tsc -b` 0；vitest 253 例（boards/spaceBindings/CommandRail/
  ProgrammingPage 65 + worklets/useVoiceChatV2/MemoryHubPage/memoryhub 组件 188）
  全绿；`vite build` 过（MemoryAtlasStage 独立 chunk 1,022KB 符合设计）。
- 构建：无孤儿进程核过（gaea/wails/go 零）→pnpm build 21.2s→wails build -s
  43.3s→gaea_install.ps1 -Launch：PID 28968 即起稳态（126MB WS），桌面副本
  哈希与 build/bin 一致。

## 余池（挂后续）

- push 列车：本地 ahead 20（含 11-07 众神殿裁决 4 笔+今日 3+1 笔），随下版 push。
- 编程板块真机走查：REA 229 工具实弹、插件实弹、Ghidra/IDA/JADX/Kuna 原生引擎
  按需安装（todos 在册）。
- dshmarket 上游 windowsHide 合入跟踪。
- 既有欠账不动：check-docs FAIL 基线（12 旧孤儿/35 旧悬空，含 11-04 day-log
  未登记）、app_info.go AppVersion 恒空遗留。

## 环境坑（当日实录）

- `internal/app` CGO 测试二进制启动 0xc0000135（STATUS_DLL_NOT_FOUND）：缺
  sherpa-onnx-c-api.dll——测试 PATH 除 ucrt64 外还须前置 `build/bin`（DLL 在
  那里），否则测试二进制起不来、报错只有一行 exit status 无栈。
- PowerShell 误重定向产物：`rg … > $null` 在某些引号形态下会创建名为 `$null`
  的真实文件（内容=rg stderr）；入库前 `git status` 见 `$null` 即垃圾产物，
  直接删（本日已清）。
