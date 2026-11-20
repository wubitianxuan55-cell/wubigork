# gaea-computeruse-runtime2-20261120-ee424218

电脑操控八刀（施工单 W0~W7）+ 运行监测二轮「可执行状态面」+ 模型中心归位——
同日两线收官包。打包提交 `ee424218`，干净工作树（`.tmp/gaea-pkg-ee424218`，
detached 于 `ee424218`）独立构建。

## 提交

- 基线：origin/main 最新推送点（设置中枢 12 提交列车之后，ahead 起点 `ef64047f`）
- 本包包含提交（origin/main..HEAD 共 **14 笔，未 push**）：
  - `a7ea2ac4`/`ef64047f` 设置板块整板 UI 二次重设计（此前已独立装机，exe `7302bee3…`）
  - `bd37782a` W0 多模态旁路（Message.Images + content parts + bridge 编码/预算）
  - `0844ed0d` permission linux 交叉编译桩（dangerous_other.go）
  - `0049ee30` W1 computer 包骨架（会话/版次/TTL 懒回收）
  - `71ef7280` W2 窗口枚举 + UIA 树（SDK 头钉 vtable + codex 文法渲染 + 真机实证）
  - `8cd8b80e` W3 只读三工具 + [computer_use] 配置段（默认关）+ 契约三表
  - `ea4b29bb` W4 按 app 授权（subjectKeys 钩子 + computer_key 硬问 + 审批卡）
  - `c216b8ab` W5 输入注入（双闸门 + DPI 失败即禁 + deny 表 + 动作后回树）
  - `fc51233b` W6 状态条 + 截图给人看（computer-status 事件 + uploads + shot_markdown）
  - `75891a56` check-docs 欠账补齐（day-log 索引 ×3 + releases 计数）
  - `8323e9d4` 运行监测二轮 + 模型中心归位（接手 codex 在途工作树，复验后收口）
  - `c4644c3f` day-log §13；`ee424218` 账本收口 + day-log §14

## 内容

1. **电脑操控**（全默认关，`[computer_use].enabled=true` 才注册）：
   - 工具面：`computer_windows` / `computer_state`（树文本+截图，截图走 W0 多模态
     旁路进主对话）/ `computer_launch` / `computer_click`（元素优先+坐标回落）/
     `computer_type` / `computer_key`（AlwaysAsk）。
   - 闸门链：绑定窗一致 → 白名单 → deny 表（Win 键族/终端/密码/认证/安全软件/
     ChatGPT·Codex 自身）→ 元素版次/坐标版次双闸 → SendInput（DPI 失败即禁坐标）。
   - 审批：window_id→`process:<exe>` 解析进 subject，「始终允许」按 app 窄粒度落盘。
   - 观察文法与 codex 逐字对齐（163 截断/32KB 行帽/LocalizedControlType）。
2. **运行监测「可执行状态面」**：当前判定关注清单（处置动作）+ 同步可信度
   （暂停恢复/防重入/轮询联动/同步中断快照）+ 下钻闭环（错误定位最近失败请求/
   四维搜索）+ progressbar·aria-live；页面规范 `design-system/gaea/pages/runtime.md`。
3. **模型中心归位**：ModelGroupNav 组级导航（语言/图像/语音/专业+计数）+ 引擎管理
   收口（顶栏 xAI 登录钮模型中心页隐藏）+ 设置模型组瘦身五页签（高频归模型中心）。

## 产物与校验

见同目录 `SHA256SUMS.txt`。要点：

| 文件 | SHA256（前 16） |
|---|---|
| payload/gaea.exe | `5739042b2dcfd702` |
| payload/onnxruntime.dll | `68d0d5f502be4509` |
| payload/sherpa-onnx-c-api.dll | `5dd36dbd3572846b` |
| payload/sherpa-onnx-cxx-api.dll | `4c56327796cdb980` |
| gaea-computeruse-runtime2-20261120-ee424218.zip | `40d6f51e45ec9cd1` |
| gaeaee424218-source.zip | `43ea874003a402f6` |

- 三 DLL 取主树 build/bin（与装机位同源同哈希）。
- payload exe 为干净工作树独立构建；与当日装机位 exe（`39C682D9…`，主树同提交
  构建链）同源不同哈希（构建时间戳），两者各自冒烟通过。

## 构建与验收

- 干净工作树：frontend `npm.cmd run build` 24.15s（node_modules 走主树 junction，
  规避无 lockfile 全新 install 的 esbuild Access denied 坑）→ ucrt64
  `CGO_ENABLED=1 wails build -s -ldflags "-s -w" -trimpath` 46.7s（bindings 未跳过）。
- 冒烟：`scripts\smoke.ps1 -ExePath <本包 payload>\gaea.exe` → `/api/health` 200
  （正式路径运行，非 .tmp）。
- 测试底账：computer 14 例+session 6 例 / permission+control / tool+builtin /
  boot / config / ai+provider / app 全包绿；前端定向 8 文件 71/71+useStatsState
  4/4+tsc -b 0；GOOS=linux 交叉编译全树绿。
- **source.zip 排除 `releases/`**（56.4MB）：历届包 zip 入库导致 git archive 层层
  套娃（office-r2 时 47MB → 本次若不排除 741MB），自本包起止血，后续包沿用。

## 装机

- 同日先行装机位（主树构建链）：exe `39C682D92A91…`，PID 26908，冒烟 200。
- 本包 payload 为归档产物，装机可复用同哈希 exe。

## 余项（不在本包）

- 电脑操控：设置页 computer_use UI 分组（需 gaea.toml 绑定，config 直编可用）；
  记事本端到端与 125% 缩放手验；`GAEA_LIVE_COMPUTER_TEST=1` / `GAEA_LIVE_VISION_TEST=1`
  用户实弹；p2 池（WGC 抓屏/多区域 shotID/Esc 钩子/drag）。
- 12 笔（含设置 2 笔共 14 笔）未 push——push 挂用户拍板。
