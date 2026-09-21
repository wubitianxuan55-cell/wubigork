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

## 留池（按价值排序，未做）

1. **执行序批次**（`agent/execute_batch.go`）：保持 provider 顺序、连续只读段并行、writer 串行；每调用落库屏障；取消 15s 残尾宽限记「效应未知」advisory。gaea batch_executor 是 v1.15 形态（只读并行/writer 串行已有，无屏障/残尾/对应性校验）。
2. **Live file observations**（`fileops/observation.go`，~370 行自包含）：host-owned 版本观察 `{Target,Version(元数据哈希)}`，外部修改 FS_STALE_VERSION、未读 FS_NOT_OBSERVED。gaea stale anchor 只是轮内 bool map。数据正确性一等公民。
3. **采样恢复状态机+预算学习**（`sampling_recovery.go`/`output_budget.go`）：frozen request 统一重试流（中断/溢出/超限/thinking-400），学习模型真实 output budget（24h TTL）。
4. **压缩救援阶梯**（`fold_ladder.go`/`truncate.go`）：PromptTokens 校准 replan→slim 摘要→投影截断终级+`maximumSafeSummaryPrefixEnd` 二分。gaea 溢出自愈止于剪枝+强制压缩。
5. **Persistent bash PTY**（`persistentshell/`）：cd/env/函数跨调用存活；marker 协议+PTY+分帧是全套工程，Windows 需 conpty。廉价近似=每会话缓存 cwd+env 前缀注入。
6. **重复调用降级**：reasonix 把硬阻断全退役改 3/5/8 纯 advisory（实测硬阻断误伤多于收益）。gaea repeatedSuccessBlock（同签名写工具 ≥2 阻）与上游方向相反——观察一个版本再定。
7. **memory 事实生命周期**：subject keys 冲突更新/freshness 三档/expiry 硬过期/pinned-relevant 二维/auto_recall 免责前缀+本地路径抹除（最后两件近零成本）。
8. **skill catalog 预算化渲染**（二分压缩描述行）+引用按需分页+watcher 热重载。
9. 杂项：read_tasks 续读游标/steer 持久化/会话私有临时目录 env 重定向/压缩状态跨重启保留/jobs 路径段校验/websearch 有界编码循环/git 硬化基线（fsmonitor=false 等）。

## 否定结论（不跟进）

- **SessionWriteAuthority 世代/lease 全栈**（v1.24.2）：解决桌面多窗口双写者竞态——gaea 单运行时，整套栈是纯负担。
- **Schema-2 DAG 会话日志+recovery 谱系目录**：单机单写无此竞态；fork/rewind 语义 gaea 会话模型不需要。
- **Goal 长任务运行时**：红线（GoalCard 撤下）；且 reasonix 自己也刚把数值暂停全拆了，等它沉淀。
- **turn_phase 计费/turn 持久化有序化/Follow v2**：遥测与多消费者语义，gaea 单 UI 无靶子。
- **MCP 全部**（在册排除）；netclient（gaea 已是完全体）；模型 reasoning 契约（接 R 类推理模型时再看）。
