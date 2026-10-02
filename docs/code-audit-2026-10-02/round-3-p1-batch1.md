# 第三轮修复记录（P1 批次一：并发 + 吞错）· 2026-10-02

> **承接**：P0 两轮清账（b6f092f7）后的 P1 开刀批。选刀口径=「真咬人优先」：concurrency 23 条 + error-swallow 9 条里，足迹互斥、修法明确、可用小刀落地的 10 条。
> **验证**：`go build` 全绿、`go vet ./...` 0、character/whisper/memory/netclient/plugin/schedule/app 七包 `go test -count=1` 全绿。

## 已落地（8 条修复 + 1 条并发回归锁）

### AP6-06 · whisper Set* 写字段持写锁
`WhisperSetEngine/SetModel/SetImageModel` 三处 RLock 下改写 orchestrator 字段改 Lock（并发语音回合读到撕裂值、race 必红）。

### AP8-05 · ListSkills 懒构造 sync.Once
`core.skillLoaderOnce sync.Once`；绑定方法体不再裸写共享字段（并发首调双重构造互相覆盖指针）。附带 nil cfg 守卫。

### AP4-10 · DAG 懒清扫纳入 dagMu
`GaeaDagList/GaeaDagGet` 的清扫「读档→改→回存」移入 `ga.dagMu`，与 `dagMarkNode` 同一临界区——锁外回存会把并发收跑刚写的节点状态用旧快照盖回去。

### AP4-08 · 索引落盘唯一临时名 + 自愈写回 best-effort
`SaveScheduleIndex` 改 `os.CreateTemp` + `RenameWithRetry`（固定 `index.json.tmp` 会让并发写者互抢，Windows 读列表撞占用报错）；`adoptIndex` 读路径自愈写回失败降为 slog.Warn 不影响返回。回归锁：`TestSaveScheduleIndexConcurrentNoTmpClash`（16 并发保存全成功、零 .tmp 残留）。

### IN2-04 · SSRF 守卫逐个已验 IP 拨号
`netclient.GuardedClient` 与 webfetch `directDialContext` 同口径改法：校验全量解析 IP 后**逐个已验 IP 尝试建连**——原实现固定拨 ips[0]，单一目标失败即整体失败（无回退）；两处拨号目标仍严格取自已校验集合。

### IN2-10 · SSE 解析 panic 上抛错误块
`ChatStream` 解析协程的 close(chunks) 责任上移到包装层：panic 时 recover 分支**先补发 `SSEChunk{Error}` 再 close**——此前 recover 只写日志、通道已被被调方 defer 关闭，消费方按正常收帧拿到「空成功」。用量记账的 Success=false 语义不变（parseStreamEvents 自身 defer 已覆盖）。直调 parseStreamEvents 的测试改为包装 goroutine 内 close（保持原生命周期）。

### IN4-07 · 安全审计写盘失败必须留痕
desktop_router 五处 `_ = AppendDesktopAgentAudit(...)` 收敛进 `appendAuditOrFail`：写盘失败 slog.Error（被拦截的危险操作不能因落盘失败而无迹可查）。用户可见 Summary 文案不变。

### IN1-10 · 角色更新写盘失败上抛
`applyUpdates` 返回 error；`ChatWithAutoSave`/`ChatCharacterDetail` 不再「写失败仍报成功」。附带堵住更险的一支：读角色失败不再退化为「空文件+增量」写回（那会清空既有角色），改为 fail-closed 报错。

### GA3-06 · cite 留痕失败 slog 留痕
`AppendCiteEvents` 统计失败数并 slog.Warn（尽力而为语义不变——不阻断回合，但「这轮引用过什么」的留痕缺口不再无声）。

## 复核结论（不修，留档）

- **AP1-03（收敛轮无同章互斥）**：已随 P0#4 修复闭环——收敛协程现以 `chapterGenKey(chapterNum, "")` 进登记表，同章生成/收敛互斥由 registerChapterGen 统一拒绝。
- **GA4-05（MCP 连接超时撤下他者）证伪**：审计描述的「Remove 插在 append 之前」不成立——`h.addConnected` 返回与 `h.Remove(s.Name)` 在同一 goroutine 顺序执行，append（h.mu 内）先于 return 先于 Remove，Remove 必然命中；超时边缘「注册了死 ctx 客户端」的情形也已被既有 `fired→Remove` 路径清理。`Stop()=false` 不保证 cancel 完成这一点属实，但其后果（客户端刚入表即被撤）正是现有语义（超时即拒收）。

## 本批未动（留池）

AP7-08（core.cfg 无锁读写，需配置快照方法）、AP7-09（拉起等待接 ctx）、AP8-01（client 锁收敛）、AP5-03/04（追问单飞，审计自标「是否被上游堵住待证」）、AP6-03/04/09、IN4-02、IN1-07、FE2-08、FE7-11、GA1-05/06/08、GA4-10、IN2-xx 余项——下一批按同口径继续。
