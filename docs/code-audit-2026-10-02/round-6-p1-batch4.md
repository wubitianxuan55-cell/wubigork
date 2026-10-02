# 第六轮修复记录（P1 批次四：AP8-01 主刀）· 2026-10-02

> **承接**：批次一~三（12aeaf6d / 9986bb9d / 532f5444）。
> **验证**：`go build` 全绿、`go vet ./...` 0、`internal/app` 全包 + weixin 包 `go test -count=1` 全绿、gofmt 干净。

## 已落地（2 条）

### AP8-01 · a.client 锁收敛（127 处读写点）

- `core` 新增 `clientMu sync.RWMutex` + `clientRef()`（读锁快照）/ `setClient()`（整体换入唯一写入口）。
- 生产码 47 文件 **127 处** `a.client` 读写全部改道 `clientRef()`（5 个写点=Login/重建链走 `setClient`；含 nil 检查、传参、局部捕获各形态）。测试构造器直填字段不受影响（同包、无并发）。
- 语义保持：写点全部赋非 nil 新实例，故「nil 检查→使用」两段式读不存在中途变 nil 的窗口；并发换入时读侧拿到的是完整的新或旧实例。
- 验收：全量替换后 `a\.client\b` 生产码残留 0；app 全包测试绿（含 AI 生成链、登录、绑定面全量用例）。

### AP6-08 · crypto/rand 失败降级留痕

`randomHex`（clawbot.go，client_id/fileKey 源）原 `rand.Read(b)` 吞错——失败产出**全零 hex**：client_id 冲突会话互踢、fileKey 撞车覆盖。现失败时 slog.Error + 降级「时间+进程内序号」（`clientIDFallbackSeq`）——client_id 第一要求是唯一而非保密，唯一性不受影响且绝不出全零。`media_upload` 的 aesKey 路径本就正确上抛，未动。

## 留池（同前）

AP7-08（core.cfg 快照锁，读面更宽需单独刀）、AP6-03（PreLLMTurn oplog）、AP5-04（待证上游是否堵住）、FE2-08/FE7-11（前端批）。
