# 资产 18 · 跨进程身份（agent-identity）：判定不做

> 源码：`agent-identity/src/lib.rs`（单文件 38 KB）。
> 对照：gaea 无对应需求（单机单用户，席位的 `attestation.go` 台账已足够）。

---

## 一、它是什么

一套**云端身份与任务注册**体系，一行不落地依赖 OpenAI 后端：

```
Ed25519 密钥对：由 64 字节种子 + 上下文串 b"codex-agent-identity-ed25519-v1" 派生
JWT：audience = "codex-app-server"，issuer = https://chatgpt.com/codex-backend/agent-identity
JWKS：从远端拉公钥集校验签名（超时 10s）
注册：POST 到 auth.openai.com/api/accounts（超时 15s；任务注册 30s）
响应：Curve25519 box 解密的 task_id（crypto_box）
超时/重试：is_retryable_registration_error() 区分可重试错误
环境：ChatGptEnvironment::{Prod, Staging} 两套 base URL
```

核心用途是「**证明这个 agent 实例是后端认证过的**」——为云端计费、配额、多租户隔离服务。

## 二、为什么 gaea 不做

| 维度 | codex 的依赖 | gaea 的现实 |
|---|---|---|
| 信任根 | chatgpt.com 的 JWKS + auth.openai.com 注册 | 单机应用，没有服务端可注册 |
| 密钥 | 从服务端下发的种子派生 | gaea 没有「发放身份」的一方 |
| 用途 | 云端计费/配额/多租户 | 用户只有自己 |
| 跨进程对手方 | app-server ↔ 客户端 ↔ 后端三方 | 进程内 + 本地 MCP，无第三方 |

强行引入会变成「自签自验」——自己发密钥、自己验签名，安全上零收益，维护上多一套密钥生命周期。

## 三、唯一值得记下的三条小实践

1. **密钥派生带上下文串**（domain separation）：同一个种子派生不同用途的密钥时，
   用 `b"codex-agent-identity-ed25519-v1"` 这类固定串参与派生，避免跨用途密钥混用。
   —— 如果 gaea 将来要给本地 MCP 连接签短期令牌，这条用得上。
2. **JWT 校验必须钉住 audience 与 issuer**：代码里那两个常量是硬编码的，
   不是从 token 里读出来再比——否则等于没校验。
3. **重试要分类**：`is_retryable_registration_error()` 把「网络抖动」与「签名/参数错误」分开，
   后者重试一万次也不会成功。

## 四、结论

**不做。** 这一块 38 KB 全部服务于云端身份，对本地单用户应用没有可移植的内核；
三条小实践记在 §三，将来若给本地 MCP 通道加短期令牌时再取用，不必现在建 `identity` 包。
