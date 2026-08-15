# 审查报告：模型中心与成本板块（review-model-cost）

## 高危
- H1 微信扫码轮询 PollQRStatusWithCode 吞 HTTP/解析错误（qrlogin.go:99-118，与 PollQRStatus:71-96 不一致；whisper_handler.go:607-625 前端无限轮询无提示）
- H2 成本库 AI 进料无上限全文进提示词（gaea_cost_import.go:98-111 textFallback 无截断，vision 路径 :523-527 有 6000 字；docmd.DefaultMaxPDFPages=500 页）
- H3 旧版明文 token 迁移后永不清除（auth/token.go:93-123 只写加密主路径；legacyPath .wubigork_token.json 从不删除；assistant/manager.go:89-97 同）

## 中危
- M1 受控测评参数无上限（gaea_benchmark.go:156-215 仅非空校验；64 并发×上千重复可跑数小时）
- M2 测评基地址硬编码 127.0.0.1:8080（gaea_benchmark.go:128，与配置脱节）
- M3 events.jsonl 每次全量读取（herdsman_stats.go:182-186；GaeaUsageOverview 每次打开读）
- M4 定价表覆盖面不足且费用不落盘（stats.go:94-126 未知模型费用恒 0；summary 重算历史费用）
- M5 keep-warm/auto-preload 不检查 herdsman Enabled（gaea_schedule.go:104-146/:245-267）
- M6 ClawBot 消息处理串行阻塞长轮询（clawbot.go:240-242 同步 handle 等 AI 30-120s）
- M7 角色剧照落盘用用户可控 ID 拼路径（characterlib/portrait.go:43，id 无清洗可越界）
- M8 统计落盘节流无退出冲刷（stats.go:207-210/:391-396 无 Flush；Shutdown 不调用）
- M9 成本条目批量导入逐条写库无事务（gaea_cost_import.go:200-230）

## 低危
- L1 stats.go:511-513 UsdCnyRate 无锁读；L2 gaea_usage_overview.go:125 硬编码 ref=1.5 与定价表重复；
  L3 herdsman_stats.go:88-92 canceled 算失败；L4 characterlib 删除不清理剧照文件/0644 权限；
  L5 benchmark 导出非原子；L6 qrlogin.go 多处忽略错误；L7 engine.go:331 TestConnection 返回 status,nil；
  L8 gaea_benchmark.go:303 用户提示词破坏 Markdown 表格

## 亮点
- 凭据 DPAPI 全链路加密+0600；前端永不接触密钥；OAuth PKCE+state+nonce 扎实；
  keep-warm 只探不启纪律好；统计器原子写+节流；成本敏感域本地化+vision 截断；
  characterlib 300KB 内联上限+迁移防死锁；weixin 生命周期自愈

## 优先级建议
1. H1+H2（确定 bug 路径、改动小）；2. H3（一次性迁移）；3. M2/M5（开关配置不一致）；4. M1/M6
