// proxy.errorchannel.test.ts — FE3-02 回归锁：错误上报通道自身故障不得击穿
// invoke 的错误归一。logFrontendError 运行在 invoke 的 catch 回调里，若它
// 同步抛错，调用方拿到的将是日志通道的异常（顶掉原始绑定错误、且不是
// BridgeError）。真实形态：GaeaLogFrontendError 绑定异常/桥接层同步失败。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { app, BridgeError } from './proxy'

function installFailingChannel(backendError: unknown) {
  ;(window as unknown as { go?: unknown }).go = {
    app: {
      CoreB: {
        // 探针绑定：任意非映射名走字面回退命中
        GaeaAuditProbeFail: vi.fn(async () => {
          throw backendError
        }),
        GaeaLogFrontendError: vi.fn(() => {
          throw new Error('日志通道同步炸了')
        }),
      },
    },
  }
}

beforeEach(() => {
  delete (window as unknown as { go?: unknown }).go
})

describe('错误上报通道自身故障隔离（FE3-02）', () => {
  // 探针绑定走字面名回退（非 gaeaToGaea 映射名），类型面无此键属预期。
  const probe = () => (app as unknown as { GaeaAuditProbeFail: () => Promise<unknown> }).GaeaAuditProbeFail();

  it('日志通道同步抛错时，调用方仍拿到原始错误的 BridgeError', async () => {
    installFailingChannel(new Error('原始绑定错误'))
    await expect(probe()).rejects.toMatchObject({
      code: 'GaeaAuditProbeFailError',
      message: '原始绑定错误',
    })
  })

  it('日志通道同步抛错时，非 Error 拒绝值同样不被顶掉', async () => {
    installFailingChannel('字符串拒绝值')
    const err = await probe().catch((e: unknown) => e)
    expect(err).toBeInstanceOf(BridgeError)
    expect((err as BridgeError).message).toBe('字符串拒绝值')
  })
})
