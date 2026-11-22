# gaea day-log 2026-11-22

## 编程板块·DSH 隔离栈全量升级（core+插件+REA+dsh-wubi 适配，gaea 仓代码零改动）

执行体：ZCode（wubi 会话）。资产根 `C:\AI\dsh`（隔离根，红线=不触 `C:\Users\wubi\.dsh` 与官方桌面安装）。

### 升级面

| 项 | 前 → 后 |
|---|---|
| DSH core | 0.2.2-rc.1 → **0.2.2-rc.5**（`npm i --no-save --prefix C:\AI\dsh @deepseek-ai/dsh@0.2.2-rc.5`，640 包变更） |
| dshmarket | 1.73.15+patch → **1.77.1**（**1.73.15.patch 退役**：windowsHide 已进上游 ≥1.73.19，1.77.1 原生×5 实锤；patch 文件移 `profiles/web/patch-work/retired-20261122/`） |
| 其余插件 | genui 0.17.0 / dsh-ops 0.5.5 / skills-manager 1.5.0 / better-sidebar 0.32.1 / dsh-context 0.87.0 / dsh-unlag 1.17.10 / skill-explorer 0.4.21 |
| REA | 6.8.0 → **6.10.1**（`C:\AI\dsh\rea` npm install；入口 `scripts/rea.mjs mcp` 契约不变；技能模板 v49/229→**v63/263** 已同步 `home/agents/skills`） |
| dsh-wubi | v2.1.0 基线上适配两笔：**289838c**（rc.5 needle 复验）+ **c5c76d2**（回退压缩态手术） |

### dsh-wubi 补丁适配细账（终账 65/82 applied、0 pending）

- **#13/#35**（escalation 两刀）：rc.5 把 `new Error` 换成 `SandboxEscalationError`+错误码、cancelled 分支拆 `throwIfAborted`——各追加 rc.5 形态变体。
- **#18**（repeat-tool-reminder 禁用）：结构化工具错误改造删掉 `downstream.kind==="block"` 分支——追加 rc.5 块变体。
- **#56/#77（回退两刀）一波三折**：先按「client.js 压缩 bundle 化+逻辑上移 session fold」记档退役（289838c）；**真机走查证伪退役**（见下），c5c76d2 复活为 ModuleLoader 压缩态变体：
  - #56 变体 4 = `replaceWindow(entries,hasMore){` 首行注入自包含 IIFE 过滤（marker seq+op range+sourceEventSeqs+整轮扩张），不依赖被 mangle 的函数名；
  - #77 变体 3 = `append(record){` 命中 `rewind-` 标记即 `replaceWindow([...inputs,record],hasMore)` 重建。
- `check-dsh-022.mjs` 版本断言 rc.1→rc.5。rc.5 升级后 skipped≠全部版本正确，**逐刀人工定性是纪律**（#14 marker 误报 applied：rc.5 原生已带 ternary guard，marker 恰好命中=无操作但意图原生满足）。

### junction 断链坑（本批逮住根修）

profile `package.json` 的 `link:` spec 一直指向已迁走的 `clones/dsh-wubi`（11-08 迁移时只手动重建了 junction、没改 spec）；11-22 早一次 pnpm 同步按 spec 重建符号链接→断链（运行中实例靠已加载模块活着，无感知）。本批改 spec=`link:C:/AI/wubigrok/dsh-wubi`+`pnpm install` 重建。**教训：手动重建 junction 后必须同批改 package.json spec，否则下次 pnpm 同步打回原形。**

### 真机走查（rc.5 web 实例+IAB 浏览器实弹，deepseek-flash 3 轮）

- 标准会话两轮：回复正常、上下文注入三项（AGENTS.md/runtime-context/skill-catalog）、**零插值错误**（{{cwd}} 修复类在 rc.5 仍稳）、缓存命中 8%→54%→69%（提示词组合跨轮稳定）。
- 预设注册：preset-declare **registered 20 插件 broken=null 首试即绿**（worker 14 同绿）；技能 23/23；工具箱 15/15；REA MCP 直连 `tools/list` = **263 工具**实数。
- 回退行为（#56/#77 手术验证）：rc.5 原生只兑现「放回输入框」不兑现「清掉这一轮」——被撤轮 live 与 reload 后都残留（会话日志 fold 数据正确，纯客户端视图缺口）。手术后：**reload 重放路径全净**（两轮被撤气泡+回复都拿掉，含嵌套回撤旧标记；轮 1 完好）；**实时路径**回复当场消失+composer 放回，用户气泡残留至 reload（与 rc.1 patched 的 replaceWindow 重建口径一致，留档不追）。
- 走查姿势坑：reload 后 composer 草稿持久化（回退还句的文本在 reload 后仍在输入框）——快照文本匹配会误判「气泡残留」，须 DOM 级探针（气泡=`_bubble_`/`userRow` 类名，composer=`uV2eYG_input`）分清；composer 在 reload 后 fill+Enter 不触发 React 提交（11-18 坑册同款复现），改页内 dispatch 或直点发送钮。

### 会话文件解压配方（rc.5 多帧 zstd）

`session.v4.jsonl.zstd` = **多帧 zstd 追加**（每 turn 追加一帧）：`zstdDecompressSync` 只解第一帧（只剩 session header）；流式 `createZstdDecompress` 管道在第二帧报 `Unknown frame descriptor`。可用配方=**按 zstd magic（28 b5 2f fd）切帧逐帧 `zstdDecompressSync` 再拼接**（本批 13 帧全解成功）。

### 处置与遗留

- 升级用的孤儿 web 实例（11-16 起，父进程已亡）与两轮验证实例均已按 owned 树处置；**3091 已清场，用户在 gaea 编程板块点「启动」即由 gaea 持币**（adopt 机制按 `.gaea-web-session.json`）。
- gaea 侧零改动：entry `node_modules\@deepseek-ai\dsh\lib\bin.js` 路径不变；REA preflight 查红队预设 `id: rea-mcp` 行不变。
- 遗留挂池：①实时回退用户气泡残留（reload 自愈，与 rc.1 口径一致，观感反馈再议）；②红队会话 REA 工具面 229→263 的角色自检卡读数变化待实弹；③rc.2 的子代理命名多级协作/AgentTeam/搜索 API 迁移等新能力未走查。
- dsh-wubi 仓 ahead 3（6c5f7ea+5ae0ed8+本批两笔）未 push，push 留用户。

### 补段：双面工具定义读数（上下文诊断面板，同日续）

- **标准会话**（模式菜单切标准+首轮实弹）：总 20.7k/首轮，**工具定义 ≈10.5k**、系统提示词
  ≈8.4k、注入内容 ≈1.1k、技能注入 ≈739——优于 11-08 基线（当年标准面 ~90 项 ≈20k/总 21.6K）。
  回复「好」零插值错，133 tok/s。走查会话一发现：`新建会话` 会**继承当前选中会话的预设**
  （session header agentPreset=redteam 实锤）——要标准面必须模式菜单显式切「标准模式」。
- **红队会话**：**工具定义 ≈328.4k**（REA 263 v6.10.1+红队 54 席全家桶）——前批三轮实弹
  （「正常」会话）实为红队预设路径，等于把红队会话链也验了：零插值错、缓存 8%→69%、
  委派/轨迹/编队渲染全正常。
- 走查姿势追加：composer 注入用 `document.execCommand('insertText')` + 直点 aria-label=
  「发送消息」按钮（fill+Enter 在 reload 后不触发提交的同坑新配方）。

### 补段：关联软件官方最新版更新批（11-22 续）

- **sherpa-onnx**：go 绑定 k2-fsa/sherpa-onnx-go-windows/v2 v2.0.1→**v2.0.2**（官方 11-19 发），build/bin 与装机位 sherpa 两 DLL 换 v2.0.2、ORT 1.31.1 保留（k2-fsa 捆的是 ORT 1.17.1，不能跟走）。yalue/onnxruntime_go 试 bump v1.38.0 失败退回 **v1.28.0**：v1.38 绑定请求 ORT API 38，需 ORT DLL≥1.38；且 **microsoft/onnxruntime GitHub 仓 2026-11 中旬消失**（疑似安全事件，官方未公告迁移，NuGet/PyPI 未受影响）——不从野镜像拉 DLL，**ORT 钉 1.31.1**（2026-10-23 官方构建）。
- **System32 新坑（全机级）**：2026-11-11 起 Windows 系统组件在 `System32\onnxruntime.dll` 放 ORT 1.17.1（germanium）。go test 临时 exe / wails 的 gen_bindings 子进程无同目录 DLL 时按搜索序被劫持：语音 cgo 测试族崩 0xc0000005（"requested API version [28] not available"），wails build 报 0xc0000135。修=**`scripts/cgo_test_dll_prep.cmd`**（`go test -exec` 包装器，DLL 三件套复制到测试 exe 旁，`GAEA_TEST_DLL_DIR` 可覆盖）+ 构建时 PATH 前置 `build/bin`。构建指南坑表已补行。
- **llama.cpp**：官方 v1.0.0 系纯源码发布，二进制走 nightly **b12359 win-vulkan-x64**（Clang 20.1.8，36MB）→ 落 `C:\AI\gaea-VOICE\voice-v2\bin\`（bundle 位=`voiceV2LLMBinPath` 解析第一优先，qwen35/gemma4 两档自管 llama 从此吃官方最新）。实弹：qwen35 35B 25s 加载、MTP draft 上下文创建、mmproj 装载、补全回 `OK-35B` 全通；`--spec-type draft-mtp`/`-md`/`--spec-draft-ngl`/`--chat-template-file` 等旗面 b12359 全兼容。engine_update_check「llama」本地=官方 1.0.0，更新提示消失。
- **纠正 11-20 旧账**：`uocr2-llama\bin` 与 `bin-ocr-0.8.0-dev` 逐字节同一份构建（都是 b11980+unlimited-ocr2.patch，llama.dll 实含 arch；bin-mainline-broken-20261111 归档目录就在 bin/ 内）——「bin/ 恢复 mainline 归 qwen35」实际未落，qwen35 此前跑的一直是 OCR 补丁构建。本批 bundle 位补上真 mainline 后，OCR（钉 bin-ocr）与 qwen35（bundle bin）通道才算真正分离。OCR 钉住位复查 10s 加载/health 200 ✓。
- **不动项**：whirl（本地 0.2.5.1 领先官方 tag v020-rc11）、Strata（本地仓 0.2.13-rc 领先发布 v0.2.9）、ovis（自建 llama 0.4.0-dev 专 arch）、breeze-tts2（权重冻结）。**ComfyUI**：AMD standalone pack v0.20.1-env1 仍为最新 pack；核心 v0.20.1 vs 上游 v0.46.2——pack 内原地升级涉 ROCm 钉死 env，挂拍板。unlimited python venv（OCR 慢速腿）pip 已坏且被 llama 通道替代 10~14x，挂清理拍板。
- 验证链：voice native ok 5.7s + internal/app 全绿 177.5s（均走 wrapper）→ npm 31.3s → wails 24.78s → 冒烟 /api/health 200（PID 36032 回收）→ INSTALL OK（exe 5ECEC355…，15:46:01，三 DLL 同批）→ 装机拉起 PID 13396>构建时刻。

### 补段：编程板块接入 Nebula Memory（yb-zhai/agent-memory）+ 记忆中枢对齐蒸馏

- **上游识别**：Nebula Memory v1.3.8（dsh-memory-nebula），零依赖 Node 本地记忆（4 臂召回+JSONL+FSRS 访问动力学+consolidate），MIT，自带 DSH 插件安装器；Hindsight 语义层未开源（纯本地不自动积累）。克隆 `C:\AI\re\agent-memory` @ eeb378a（B 区第 15 个，README 已登记）。
- **安装**：install.ps1 -Profile web -DshHome C:\AI\dsh\home（事务式：备份 SHA256→暂存换装→健康检查→失败回滚）→ 插件落 home\plugins + profile manifest bundles/dependencies + memory-nebula.json（data=C:\AI\dsh\home\memory-nebula-data）。
- **会话侧**：home cordis.patch.yml（原空 []）挂 `dsh-mcp-client` 行（serverName: agent-memory，stdio，env.NEBULA_DATA_DIR 同库同源，4 工具 ≈1k tokens 全局挂载）；备份 .bak-20261122。
- **实弹链**：MCP stdio 探针（initialize→4 工具→remember→recall 0.1519→stats）✓；web 起站侧边栏「记忆星云」入口 ✓；红队会话端到端（6 步 770K tok——红队面工具重+模型先查配置）真实写入 nodes.jsonl mmwrjw5txc81ff8f4 ✓；3091/8898 已清场恢复 gaea 点启动态。
- **发现**：①面板与 MCP 同库双实例各持内存视图——MCP 写入后 /healthz units 仍 1（无文件 watch）；②MCP 写入 source="unknown"；③新会话继承预设坑复现（验收会话落红队面）。
- **蒸馏**：报告=docs/gaea-agent-memory-nebula-distill-2026-11.md（gaea 记忆中枢四件可借：四臂召回+arm 归因/FSRS 动力学/consolidate 事实化/层次化；重叠面不抄；P1~P5 提案交拍板池）；逆向说明=C:\AI\re\agent-memory\docs\蒸馏-gaea记忆中枢对齐-2026-11-22.md（file:line 锚全录）。gaea 记忆中枢代码零改动。
