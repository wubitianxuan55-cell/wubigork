# 第十一轮修复记录（P1 批次九：真 bug 与静默失效）· 2026-10-02

> **承接**：批次一~八（b6f092f7 → afc39177）。
> **验证**：`go build`/`go vet` 全绿；boot/tool-builtin/schedule/whisper/app 五包测试全绿（app 全包 111s）；`tsc -b` 0、eslint（触达文件）0、context+bridge+ContextView 族 8 文件 86 测试全绿。
> **本批主题**：从余量 172 条 P1 里挑「真 bug / 静默失效 / 用户可见」类——全部为行为修复或契约钉死，非结构搬移。

## 已落地（8 条）

### GA4-01 · boot Effort 覆盖晚于 NewProvider——静默失效
`boot.go` 里 `entry.Effort = cfg.Agent.Effort` 写在 `NewProvider(entry)` **之后**：NewProvider 构造即把 `entry.Effort` 逐值拷进 `provider.Config.Extra`，构造后再赋值纯空转——设置中心写入的 effort 对主执行 provider 从未生效。改为先覆盖再构造（与同函数子代理路径 :300-309 的既有正确形状同序）。**金线测试** `TestBuildAppliesAgentEffortBeforeProviderConstruction`：注册探针 kind 构造时快照 `Extra["effort"]`，旧序实收 `""` 必红、新序实收 `"high"`。RED→GREEN 双向已证。
**深挖附记**：`Extra["effort"]` 当前仓内无消费方（bridge provider 不读；boot 注释所称 anthropic provider 不在本树）——本刀修复的是装配序契约（对任何读 Extra 的 provider kind 生效）；让 effort 贯通到请求体是独立功能线（`provider.Request` 尚无该字段），不属审计修复范围。

### AP1-04 · 弧线水位把循环下标当章号
`storySpineSection` 水位点选取 `arc.Beats[j].Chapter > last`——`last` 是**下标**却拿**章号**去比。水位点不保证按章序存储（AI 提案/手编合法乱序），高章水位被错降成低章（实测乱序 [9,2,6] 注入「第6章」而非「第9章」）。改比对 `arc.Beats[last].Chapter`。回归锁 `TestStorySpineArcWatermarkChapterOrder`（乱序选取+未来章排除双断言），RED→GREEN 已证（旧实现实弹产出第 6 章）。

### AP6-07 · whisper 日志预览字节切片切中文
`userMsg[:min(60, len(userMsg))]` 字节切片把多字节中文切成 mojibake 进日志。改同包既有 `truncateRunes`（rune 安全+省略号）。

### IN3-05 · xlsx 日期解析 time.Local 混 UTC 口径
`tryParse` 用 `time.Local`，而 schedule 模块日期语义是 UTC 午夜（calendar.go 同款）——Local 午夜在 UTC+8 折成前一日 16:00Z，与日历函数比较时整体偏移一天。改 `time.UTC`（解析/Format 同位往返，日期文本输出不变，属口径统一）。回归锁 `TestTryParseUTCConvention`：UTC 落点+改 Local 结果不变（确定性口径）。

### FE1-04 · FileActivityTree 未接自定义详情源，点开详情串数据源
`useNodeDetails` 本就支持自定义源注入（v4.412 原罪复用），ContextBrowserTree 有接、**FileActivityTree 漏接**——原罪页（OriginalSinPage → ContextView 传 `fetchNodeDetail`）里文件活动面板的操作详情仍按缺省会话日志回读，同一页面两套数据源。修法：FileActivityTree 加 `fetchNodeDetail` prop 透传 useNodeDetails + ContextView :733 下传。回归锁：inspector.test.tsx 新用例（展开操作行点详情→断言调注入源且内容来自注入源）。**测试坑**：排序初值读 localStorage 偏好，同文件「排序三胶囊」用例写入「按路径」会让本用例首行漂移——用例内先清 `gaea.context.prefs`。

### FE3-02 · 错误上报通道自身同步抛错击穿 invoke 归一
`logFrontendError` 运行在 invoke 的 catch 回调里；`Promise.resolve(lfe(message))` 的**参数求值**先于 Promise 构造——`lfe` 同步抛错直接炸穿 catch 回调，调用方拿到日志通道异常（顶掉原始绑定错误、且非 BridgeError）。修：try/catch 包裹 + `.then(() => lfe(message))` 惰性调用。回归锁 `proxy.errorchannel.test.ts` 双用例（原始 Error / 字符串拒绝值均不被顶掉），RED→GREEN 已证（旧码 2 用例全红）。

### IN4-08 · normalizePath 注释撒谎→诚实化+防护属性钉死
注释称「禁止 .. 逃逸」实无校验（Clean 只做词法消解）、称「展开 %ENV%」实不展开（os.ExpandEnv 只认 `$var`，既有测试已自证）。定性：cwd 圈定会破坏「绝对路径是桌面操作合法输入」的契约，**越界防护本就由 evaluatePathPolicy 对解析后路径施加**。修法=注释如实描述 + `TestNormalizePathEscapeStillGuarded` 钉死真实防护属性（`..` 逃逸进 System32 的写入、相对形式直达策略层，均硬阻断）。

### GA2-04 · spaceTags 漏 17 个已注册工具，fail-open 混过分类面
实测缺口与审计精确吻合：browser_*×11 + schedule_analyze/apply/get + genui_validate + read_spill + sidebar_open。全部显式入表为 shared（表内 shared 与表外缺省走同一 AllowsSpace 路径，**零行为变更**；work/play 重归类属产品决策，不在审计修复范围）。守卫 `TestSpaceTagsCoversAllBuiltins`：每个 RegisterBuiltin 注册的工具必须显式入表（新工具漏归类即红）。**守卫边界**：反向（表⊆注册面）不可在本包查——表还覆盖 app/agent 层构造的工具名（image_gen/ask 等非 RegisterBuiltin 注册），全量注册面在 boot 装配层。

## 本批坑

1. **heredoc 反斜杠在册第四踩**：Go 字符串字面量 `..\` 进 bash heredoc 触发 python unicodeescape SyntaxError——按复训一律 Edit 手写，仍侥幸试了一次 python 源码，同样降级。
2. `tsc -b` 经管道时 `$?` 反映的是 tail 的退出码——判断类型检查结果必须重跑取原exit code（本次实跑发现 2 处测试类型错，`npm run -s build` 的静默 0 是假象）。
3. FE 测试排序偏好跨用例污染（localStorage 持久化）——依赖隐式全局状态的用例要么自清要么显式设初值。

## 留池（不变）

需设计拍板：AP1-10（归一化超集）、AP5-01（截断 14 份）、AP6-05（回合收尾四套）、AP2-03（在途登记六套）、FE2-05/06/08、FE7-11、AP4-01/AP4-04（缺省面）、#25（三重嵌入）。结构类（god-file/god-func/大区收敛）按批次路线另行开刀。
