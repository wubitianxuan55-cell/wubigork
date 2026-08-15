# 审查报告：后端核心与可靠性（review-backend-core）

## 高危
- H1 看门狗墙钟 10 分钟硬上限无视推进状态（watchdog.go:154-157 无条件 fire；注释承诺的豁免仅停滞维度；
  boot.go:393-417 默认装配零值→DefaultWatchdog；正常多轮办公任务 >10min 被强杀，最终回答不产出）
- H3 useController 写路径全部静默吞错（store.ts:454 send / :457-465 cancel/approve/answerQuestion /
  :466-520 其余全部空 catch；approve 先 clearApproval 再调用，失败则审批卡消失后端挂起；
  T6-1.2 只修了读路径，测试也只覆盖读路径 store.test.ts:113-168）

## 中危
- M1 非流式 Client.Chat 无重试（client.go:283-370 单次 Do，仅 401 递归；v2.24.0 留待项未落地；
  当前生产调用方已流式化故不触发，但公开 API 面有回归隐患）
- M2 Chat 401 递归额外占信号量槽（client.go:340-346 递归期间持 2 槽）
- M3 Client 内部状态无并发保护（client.go:139-153 activeEngine / :963-969 imageBackend / :212-238 token）
- M4 tasks handler panic 无 recover（tasks.go:437 直接调用；worker 永久死亡+任务卡 running，进程不崩无日志）
- M5 看门狗墙钟在 userWait 期间继续计时（watchdog.go:154-157 不豁免，停滞才豁免 :158-161）
- M6 promptMu 跨阻塞持有（controller.go:612-634，契约脆弱）
- M7 预执行只读工具取消后仍写缓存（agent_stream.go:80-95，无实际损坏）
- M8 SaveConfig 内存同步只覆盖 4 键（shelf.go:113-142，其余 ~40 键落盘成功但运行时不变、UI 不提示）
- M9 herdsman health/probe 默认形态不一致（health.go:24-29 /models vs probe.go:26-30 /v1/models；
  127.0.0.1 vs localhost）
- M10 聊天搜索失败静默降级（chat_service.go:39-46，用户不知搜索未生效）
- M11 gaea_translate 用 context.Background()+3min 硬超时（gaea_translate.go:158-163，长文档截断丢部分结果）
- M12 reconcileFinalAnswer 120 字符前缀探针（store.ts:370-389，重复模板漏补/变换误补）

## 低危
- L1 代理无效回退 NewSimpleClient(0) 总超时丢失（client.go:118-121）；L2 ChatStream 401 重试重置 usage 计时；
  L3 turnLoop 排队回合 TurnDone 时序；L4 Snapshot 失败只记日志；L5 agent_stream.go:24 无锁读 session.Messages；
  L6 流恢复计数偏保守；L7 health.go:183 用 http.DefaultClient；L8 config.Load 注释只调一次实际多处调用；
  L9 httpbridge 反射面无白名单（回环+token 可控）；L10 SSE 丢帧无水位提示；
  L11 ChatStreamPlain 在 app 级 ctx 上运行；L12 bindings 生成代码约束

## 亮点
- SSE 三层重试+空闲超时+超长行（client.go 459-784）；看门狗原子状态+豁免模型+6 场景测试；
  工具执行防循环三重 recover（batch_executor.go）；tasks 持久化状态机+续跑+退避；
  netclient 四模式代理库级质量；httpbridge 回环+token+常量时间比较；后端不静默（chat_service.go:144-151）

## 建议不做
- 看门狗可视化面板（给 config 键即可）；httpbridge 白名单/SSE 重放（本机调试足够）；
  image backend 完整代理矩阵（默认后端全本地）；herdsman YAML 依赖；Client.Chat 全面重试高优投入；
  tasks per-task worker；reconcileFinalAnswer 全文指纹

## 优先级
H1 看门狗推进感知 > H3 写路径错误可见 > 中危 3-8 同批
