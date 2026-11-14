# Obscura × ego(lite) 浏览器项目蒸馏调研（2026-11-14）

调研性质：可研，未动刀。事实源=本档；证据锚 GitHub 仓库快照（2026-11-14）与第三方评测。

## 一、两项目一句话定位

- **Obscura**（[h4ckf0r0day/obscura](https://github.com/h4ckf0r0day/obscura)，[官网](https://www.obscura.sh)）：Rust 写的**无 Chromium** 无头浏览器引擎，内置 V8 跑真 JS、自研渲染引擎（截图/PDF/录屏），CDP 子集兼容，Apache-2.0 **全开源（含引擎）**，~70MB 单二进制。面向 AI agent 与爬虫。v0.2.7（2026-11-13）双周发版，35.3k★/2.8k fork/1,724 commits。Cloudflare Kitesurf 原型即以它为灵感。
- **ego(lite)**（[citrolabs/ego-lite](https://github.com/citrolabs/ego-lite)）：**让 agent 用你已登录的真实浏览器**。Spaces=浏览器内隔离并行工作区，首启迁移 Chrome 登录态/cookie/扩展。仓库 MIT 但**只含 skills + ego-browser 连接层；浏览器本体是 CDN 下发的闭源免费软件**。22.9k★/560 commits，macOS+Windows，Linux 在路线图。

## 二、第三方实测对 Obscura 宣传的修正（[wavect 评测](https://wavect.io/blog/obscura-rust-browser-ai-agents)）

- 「30MB 内存/85ms 加载」只在**无渲染小 fixture** 成立；真实渲染场景 98 页 live corpus 中位 **5.2s（Chrome 2.1s）**、峰值 RSS 中位 64.2MB。
- Web Platform Tests：核心 83.3%、全量 60.0%——长尾 CSS/部分 Web API/media/compositor 与 Chromium 有差距，官方 README 自认。
- Playwright/Puppeteer 可连但只是 CDP 子集，Playwright 官方称 CDP 接入「lower fidelity」。
- 全部性能数据为项目自测，无独立复跑。评测结论=**pilot，勿 wholesale**；建议钉版本+可逆任务类灰度+Chrome 兜底。
- 对比项：Lightpanda（DOM-first）是 AGPL-3.0——**gaea AGPL 不搬红线，直接排除**。

## 三、gaea 现状盘底（2026-11-14 代码实证）

已有（`internal/gaea/browser/` + `internal/gaea/tool/builtin/`）：

1. **手写 CDP 栈零第三方依赖**：`cdp.go:64` Dial（gorilla/websocket）、`targets.go` DevTools HTTP、`input.go` 键盘注入、`manager_evaluate.go` Runtime.evaluate、`screenshot.go` 截图；11 个 `browser_*` 工具（navigate/read/snapshot/click/type/press/scroll/tabs/new_tab/switch_tab/close）。
2. 受管 Edge 干净临时 profile（`edge.go:24` FindEdge 三段式），IdleTTL 10min 看门狗，JobObject 收割。
3. `web_fetch` 纯 HTTP+SSRF 门+**低保真 htmlToText**（剥标签式，无 readability 主提取）；`web_search` 6 引擎静态 SERP。
4. 受管引擎 seam 空着浏览器位：`managed_engine_idle.go:79` 目标表、`local_engine_proc.go` setWithJob。

缺口（Explore 实证）：引擎硬绑 Edge；fetch↔browser 无升级路由；HTML→MD 保真低；无资源抓取；**BrowserPane iframe 用户视图与 agent CDP 会话互相不感知**。

## 四、逐维对比

| 维度 | gaea 现状 | Obscura | ego(lite) |
|---|---|---|---|
| 浏览器本体 | 系统 Edge（必在） | 自研引擎+V8，70MB，**全开源** | **闭源二进制**（CDN 分发） |
| License | — | Apache-2.0 ✓ | MIT 仅外围；本体不可审计 ✗ |
| 会话模型 | 单实例多 tab 干净 profile | 每 agent 隔离会话毫秒级拉起；`serve` 常驻 + worker 并行 | Spaces 共享**用户真实登录态** |
| 渲染保真 | Chromium 全保真 | 60~83%，长尾缺 | Chromium fork（推测），全保真 |
| agent 接口 | 自写 CDP+11 工具多轮往返 | CDP 子集（Target/Page/Runtime/DOM/Network/Fetch/IO/Storage/Input）+MCP 14 工具 | **in-page JS 一次执行**（snapshot/fill/click/wait/navigate/capture 编成一段 JS） |
| 亮点 | 工程底座齐（JobObject/idle seam/InjectHTTPBase 测试缝） | LP.getMarkdown DOM→MD、stealth 指纹、SSRF 门、worker 并行 | 登录态共享、一次执行省 token 往返、Space 可视化接管 |
| Windows | ✓ | ✓ x64（VC runtime 打包，render 变体才有截图） | ✓ x64 |
| 成熟度 | — | v0.2.x 双周版、生态初成（2026-08 首版） | 新、无公开 release 页、PR 仅协作者可开 |

CDP 覆盖核对：gaea 现用域（Target/Page/Runtime/DOM/Input/截图）**全部落在 Obscura 子集内**，`cdp.go` 理论上可直接拨 `obscura serve --port 9222`。

## 五、蒸馏判定

### Obscura——可蒸馏（Apache-2.0 全开源）

1. **sidecar 刀**：obscura.exe 作为可选第二浏览器引擎。seam 就两处——`edge.go` FindEdge 泛化 + `browser.Options` 启动参数；挂 `managed_engine_idle.go:79` 目标表得空闲回收。价值场景=Edge 缺席兜底/并行多会话扩容。**注意**：gaea 用户机器 Edge 必在，30MB vs 200MB 内存差在单用户桌面场景非痛点，立项须有真实需求面（并行 agent 抓取/无 Edge 环境）。
2. **思路蒸馏刀（不引本体）**：LP.getMarkdown 式高保真 DOM→Markdown——gaea 用已有 goquery 依赖（booksource 在用）在 web_fetch 写 readability 正文主提取即可，零新依赖。
3. **暂不动**：stealth 指纹（gaea 非爬虫对抗场景）、worker 批量抓取（无需求面）——挂观察池。

### ego(lite)——本体不可蒸馏，仅范式可蒸馏

1. **范式刀·最实**：in-page JS 一次执行。gaea 已有 `manager_evaluate.go` Runtime.evaluate seam——把 11 个 browser_* 的多轮工具往返升级为「一段 JS 批量 snapshot+动作」，省 token 省延迟，纯增量不动现有工具。
2. **范式刀·需拍板**：共享登录态 Spaces。与 gaea「干净临时 profile」设计哲学相反，把用户 cookie 交给 agent 触及信任红线（同族=办公记忆写入确认红线）。若做：默认关+逐次确认闸+只读起步。且目前无「agent 操作用户已登录 web 应用」的实证需求，挂拍板池。
3. **不建议**：引入闭源浏览器本体当依赖——不可审计、CDN 分发、与自管开源栈冲突。

### 共通红旗

- 两项目皆 2026 年新项目，生态波动大，接入必须钉版本。
- Obscura 性能数字有水分（见 §二），立项不得按宣传数字算账。
- 渲染保真缺口决定 Obscura 定位=**补充引擎，非替换 Edge**。

## 六、建议刀路（进拍板池，等点名再动）

1. **P1** browser_run_js 一次执行范式（ego-lite 思路，gaea 现成 seam，最小刀最实收益）
2. **P2** web_fetch 正文提取升级：goquery readability（借 Obscura LP.getMarkdown 思路，零新依赖）
3. **P2** obscura.exe 可选引擎接 FindEdge 泛化 seam+managed idle 记账（Edge 缺席/并行需求实证后才立）
4. **P3** 共享登录态+BrowserPane↔agent 通道打通（触及信任面，需求实证+逐次确认闸后才立）

## 附：证据源

- https://github.com/h4ckf0r0day/obscura （仓库/README/发布）
- https://github.com/h4ckf0r0day/obscura/releases （v0.2.7，2026-11-13，四变体×22 资产）
- https://www.obscura.sh / https://obscura.dev
- https://wavect.io/blog/obscura-rust-browser-ai-agents （第三方实测修正）
- https://github.com/citrolabs/ego-lite （仓库树核实：仅 skills+连接层，本体闭源）
- https://lite.ego.app/blog/hermes-with-wings-what-ego-lite-unlocks
