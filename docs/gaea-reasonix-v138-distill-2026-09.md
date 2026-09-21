# Reasonix (esengine/DeepSeek-Reasonix) v1.15→v1.38 真身蒸馏 —— 侦察与刀池（2026-09-21）

- 源：`clones/deepseek-reasonix`（GitHub **esengine/DeepSeek-Reasonix**，最新 v1.38.11，HEAD 5c4b343a4 @2026-09-21；Go 单二进制终端 agent，MIT）。gaea agent 内核注释里的「DeepSeek-Reasonix-V1.12/V1.15」即此仓的旧版——**上一版 V1.15 之后到 v1.38 共 5227 提交/107 tag 的增量是本轮蒸馏面**。
- 对象勘误史：v4.373 误把 `/c/AI/deepseek-harness`（TS monorepo dsh）当 Reasonix——已勘误更名 `docs/gaea-dsh-016-distill-2026-09.md`（五刀本身有效）；本档是真身。
- 纪律：蒸馏=取道不取器；GoalCard 系已撤下（reasonix 的 Goal 长任务运行时同属红线不评估）；MCP/平台化/跨设备在册排除。

## 侦察方法

CHANGELOG.md（行 7~220 Unreleased=v1.21+，221~455=v1.20.0 详版，456+=早期摘要）+ `git log v1.15.0..HEAD -- internal/<pkg>` 变更密度定位（run_loop.go 140 commits、execute_batch.go 44、compact.go 58、permission 35 为高频区）。三路并行：agent 内核与收尾、工具执行层、provider/安全/memory/skill。

## 本轮已落地（v4.374.0，七刀）

| 刀 | 源机制（上游文件） | gaea 落点 |
|---|---|---|
| 刀1 Host 白名单 | `internal/serve/hostguard.go`：loopback 服务被 DNS-rebinding 变同源后可无预检驱动 RPC→421 拒绝 | `httpbridge/hostguard.go`（通配绑定豁免；Handler 默认 loopback 名单；ServeWithToken 按监听地址派生）+ app 诊断端口 /healthz·/stack 同挂 |
| 刀2 只读表修正 | `shellsafe/effect.go`+`permission/bash_approval.go`：cargo check/doc 跑 build.rs 非只读；awk 内联程序 system() 执行 shell；env 包装任意命令 | `permission/bash_readonly.go`：移除 env/awk/sed（程序体是可执行 shell 的迷你语言，字符串层不可分类，fail-closed 交审批）；cargo 仅留 search；find 补 -ok/-okdir；git 补 --ext-diff/--textconv/cat-file --filters/grep --open-files-in-pager 守卫 |
| 刀3 clean-filter 中和 | `internal/gitcmd`：diff 是唯一在工作树内容上跑 clean filter 的子命令，.gitattributes 不可信 | `app/gaea_git.go gitRun`：枚举 repo 本地 filter.* 注入 `-c filter.<n>.clean=/-c process=/-c required=false`（恒等直通，diff 仍真实；非 repo fail-open） |
| 刀4 SSRF 共享件+booksource 补洞 | `internal/installsource/ssrf.go` RoundTripper 层 IP-literal 再验 | 判据提升 `netclient.BlockedSensitiveIP/BlockedInternalIP/GuardedClient`（webfetch 委托，loopback 放行语义不变）；booksource HTTPFetcher 默认 client 从裸 DefaultClient 换守卫版（书源规则可被诱导注入，不得摸内网/元数据） |
| 刀5 收尾判定 | `run_loop.go` Deterministic natural-turn completion（v1.37 删 completion-validator）：reasoning-only 干净 stop=最终回答；真零内容=冻结请求重放 | `agent_run.go`：reasoning-only stop 不再强制可见文本/注 nudge；零内容重试不注入合成 user 消息（原 emptyFinalRetryMessage 退役）——缓存前缀零污染 |
| 刀6 输出内存上限 | `shellrun/bounded.go`：10MiB 级运行中封顶，防失控命令打爆内存 | `tool/builtin/bash.go boundedOutput`：head 1MiB+滚动尾 64KiB+截断标记，Write 即封顶（与退出后截断本质不同） |
| 刀7 stuck 解锁 | `context_manager.go` stuckInputHash：卡死绑定输入形状，新输入自动解锁 | `compact.go`：compactStuck 记 stuckAtMessages，新消息=新折叠边界自动解锁重试（旧熔断是会话级永久）；midTurn 不再提前短路 |

测试：Go +10 文件级（HostGuard 10 子例/桥 421/BlockedInternalIP+GuardedClient 拒 loopback/cleanFilterOverrides 两路/间接执行 20 例/reasoning-only final/零内容冻结重放/stuck 解锁两态/boundedOutput 三例）；行为变更适配两处（TestHTTPFetcherZeroValueLocalServer 改钉守卫拒绝语义；output_continue_test 裁剪至 length-nudge 段）。

## 后续蒸馏

- **第二弹（v4.375.0）**：留池 1 执行序批次 + 2 Live file observations + 9 之 git 硬化基线 + 7 之 auto_recall 免责，四刀销号，详见 v4.375.0 发布说明。
- **第三弹（v4.376.0，三刀）**：
  | 刀 | 源机制 | gaea 落点 |
  |---|---|---|
  | 刀1 压缩救援阶梯 | `fold_ladder.go`/`truncate.go`/`rescueOrFail`：slim 摘要档→投影截断终级（`truncateProtectShare=4`） | `agent/fold_ladder.go`：①slim 档=摘要请求估算超 window−预留−5% 边际时逐消息截头（ToolCall 参数截 160）换有界转录，全量档被 provider 真实拒绝后重试也降半预算 slim——溢出场景「摘要请求自己 400→机械裸计数」的命运换成真摘要；fits 时逐字重放不变（缓存对齐仍是首选档）②`truncateRescue`=压缩无进展**或压缩后估算仍在窗口之上**（上游 at-or-above-ceiling 语义）时：保护区收窄到 target/4（planCompaction 的 ≥2000 token 尾钳在「窗口太小压不完」困境里会把可救内容全划成圣地）→抹大工具结果（老→新，KeepErrors/KeepProtected 豁免）→整单元丢最老+装 marker；改动前归档（与 prune 同纪律，archive 坏=整体拒绝 fail-closed） |
  | 刀2 重复调用裁决 | reasonix 把硬阻断全退役改纯 advisory（实测误伤多于收益）；gaea 观察期 v4.375→v4.376 满 | `execute_one.go`：repeatedSuccessBlock 硬阻断退役，计数照记；`advisoryRepeatSuccess` 在批后注入一次性合成 user 提醒（每签名每轮至多一条，确定性序），真实工具结果不再被 blocked 替换（turnToolErrors/Success 语义更真实） |
  | 刀3 skill 目录预算化 | skill catalog 二分压缩描述行 | `skill/index.go`：目录超 IndexMaxChars 先二分收缩描述宽度保**全员可见**（尾部技能被硬截断=模型永远用不到），压缩态尾注说明+计入预算；纯名行仍超的极端才退硬截断（旧行为） |

  测试：Go +13（slim 截头/参数封顶/oversized 请求收缩/fits 逐字不变/provider 拒绝后降档重试/rescue 抹结果/丢单元装 marker/KeepErrors 豁免/无窗无区无靶三态/溢出恢复落到截断终级/advisory 越线注入一次性两条改钉/目录压缩保全员/fitting 目录零变更）+2 处适配（summarize 加 forceSlim 形参）。**坑**=①rescue 的可达带=「compact 尾钳与 rescue 尾钳之间的盲区」——大块 assistant 正文是 prune 永远够不到（只抹 RoleTool）、compact 尾保护又留下的内容，E2E 场景必须用这类内容构造②estimator 是 max(bytes/4, runes/2)——纯 ASCII 按 2 字符/token 计，构造 fits/over 边界时按这个口径算。

## 留池（按价值排序，未做）

1. ~~**执行序批次**~~ → **已落地 v4.375.0（刀1 同路径资源键统一+对应性守卫）**：`read:/file:` 双键统一为 `file:<path>` 资源键——同批「读 A→改 A」拆批保序，跨路径共存并行的延迟收益保留；call/result 对应性守卫兜底双重 recover 间的逃逸路径。上游的全序严格执行（writer 一律屏障）未全取——gaea 冲突键模型已编码资源隔离，统一键即消真竞态且不退 v4.63 并行子代理特性。
2. ~~**Live file observations**~~ → **已落地 v4.375.0（刀2）**：会话级版本观察（size+mtime 指纹）——read_file 真实读后记录、写前比对、自身写后刷新；外部修改/删除 `blocked: [stale version]`；从未观察不拦；缓存命中不假装观察。与 V10.28 stale-anchor 规则并存（锚点新鲜度 vs 外部篡改）。
3. ~~**采样恢复状态机+预算学习**~~ → **辨伪销号（v4.376）**：四路重试逐对账——中断（stream recovery 在册）、溢出（v4.373 刀1 在册）已有；超限无靶子（gaea 主链路**从不发送 max_tokens**，provider 默认输出预算自己管，请求侧裁剪+共享窗 admission 无锚点）；thinking-400 无靶子（reasoning 不回放——openai provider 装配时丢弃 reasoning_content，没有「重放被 API 拒」的形态）；输出预算学习依附于请求侧 max_tokens，同无锚点。
4. ~~**压缩救援阶梯**~~ → **已落地 v4.376.0（第三弹刀1）**：slim 摘要档+投影截断终级+target/4 保护区收窄。未取：`maximumSafeSummaryPrefixEnd` 安全前缀二分（gaea 摘要请求非 live 前缀，降 slim 档即达同等效果且更简单）与 chunked fragment 多请求路径（gaea 无片段形态）——取道不取器。
5. **Persistent bash PTY**（`persistentshell/`）：cd/env/函数跨调用存活；marker 协议+PTY+分帧是全套工程，Windows 需 conpty。廉价近似=每会话缓存 cwd+env 前缀注入（缓存与真实 shell 状态漂移风险高，须专门设计——留池）。
6. ~~**重复调用降级**~~ → **已裁决 v4.376.0（第三弹刀2）**：硬阻断退役改批后 advisory（每签名每轮一条），与上游实测结论对齐。
7. **memory 事实生命周期**：subject keys 冲突更新/freshness 三档/expiry 硬过期/pinned-relevant 二维。~~auto_recall 免责前缀~~ → 已落地 v4.375.0（刀4：memory_search 结果前置低权威免责声明）；本地路径抹除仍留池（gaea 记忆内容不含机器路径，收益待证）。
8. ~~**skill catalog 预算化渲染**（二分压缩描述行）+引用按需分页+watcher 热重载~~ → **渲染已落地 v4.376.0（第三弹刀3）**；引用按需分页辨伪销号（read_skill 即按需加载，无预灌形态）；watcher 热重载辨伪销号（目录在 system prompt 缓存稳定前缀内，热重载必破前缀——新会话即得新目录，单用户桌面够用）。
9. 杂项：~~read_tasks 续读游标~~（辨伪：gaea read_file 已有 offset/limit 分页，task 工具无长列表形态）/steer 持久化（辨伪：gaea consumeSteer 即 session.Add=持久，上游缺口在其事件账本架构，gaea 无靶子）/~~会话私有临时目录 env 重定向~~（辨伪 v4.376：Git Bash 下 TMP/TEMP 注入牵动 /tmp 映射与 MSYS 语义，单用户桌面并行会话临时互踩稀薄——风险收益比不成立）/~~压缩状态跨重启保留~~（辨伪 v4.376：重启丢 compactStuck 熔断=天然解锁救援，持久化反而把这条路堵死）/jobs 路径段校验（gaea jobs 无 artifact 落盘路径拼接，无靶子）/websearch 有界编码循环（gaea truncateToolOutput 事后截断已兜，编码中顶破上限形态不存在）/~~git 硬化基线~~ → 已落地 v4.375.0（刀3：GIT_TERMINAL_PROMPT=0/GIT_OPTIONAL_LOCKS=0/GIT_CONFIG_NOSYSTEM=1 + -c fsmonitor/maintenance 关闭 + diff 强制 --no-ext-diff --no-textconv）。

## 否定结论（不跟进）

- **SessionWriteAuthority 世代/lease 全栈**（v1.24.2）：解决桌面多窗口双写者竞态——gaea 单运行时，整套栈是纯负担。
- **Schema-2 DAG 会话日志+recovery 谱系目录**：单机单写无此竞态；fork/rewind 语义 gaea 会话模型不需要。
- **Goal 长任务运行时**：红线（GoalCard 撤下）；且 reasonix 自己也刚把数值暂停全拆了，等它沉淀。
- **turn_phase 计费/turn 持久化有序化/Follow v2**：遥测与多消费者语义，gaea 单 UI 无靶子。
- **MCP 全部**（在册排除）；netclient（gaea 已是完全体）；模型 reasoning 契约（接 R 类推理模型时再看）。
