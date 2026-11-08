/**
 * AudioWorklet 设备采样率兼容回归：16k 强制失败时不能在 44.1/48k 上
 * 把 320 帧冒充 20ms。这里用 VM 直接跑 public 产物源码，钉住真实 worklet。
 */
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'
import { describe, expect, it } from 'vitest'

function loadProcessor(file: string, rate: number) {
  let Processor: new () => {
    port: { onmessage: ((event: { data: unknown }) => void) | null; postMessage: (data: unknown) => void; close?: () => void }
    process: (inputs: Float32Array[][], outputs: Float32Array[][]) => boolean
    [key: string]: unknown
  }
  const posted: unknown[] = []
  class AudioWorkletProcessorStub {
    port = { onmessage: null, postMessage: (data: unknown) => posted.push(data), close: () => undefined }
  }
  const source = fs.readFileSync(path.join(process.cwd(), 'public', 'voice-worklets', file), 'utf8')
  vm.runInNewContext(source, {
    AudioWorkletProcessor: AudioWorkletProcessorStub,
    registerProcessor: (_name: string, ctor: typeof Processor) => { Processor = ctor },
    sampleRate: rate,
    Int16Array,
    ArrayBuffer,
    Float32Array,
    Math,
  })
  return { Processor: Processor!, posted }
}

describe('voice AudioWorklet 采样率兼容', () => {
  it('capture 在 48k 输入下落成 20ms/320@16k', () => {
    const { Processor, posted } = loadProcessor('captureProcessor.js', 48000)
    const p = new Processor()
    p.process([[new Float32Array(960).fill(0.25)]], [])
    const audio = posted.filter((d): d is ArrayBuffer => d instanceof ArrayBuffer)
    expect(audio).toHaveLength(1)
    expect(audio[0].byteLength).toBe(320 * 2)
  })

  it('capture 在原生 16k 输入上仍逐样本成帧', () => {
    const { Processor, posted } = loadProcessor('captureProcessor.js', 16000)
    const p = new Processor()
    p.process([[new Float32Array(160).fill(0.1)]], [])
    p.process([[new Float32Array(160).fill(0.2)]], [])
    const audio = posted.filter((d): d is ArrayBuffer => d instanceof ArrayBuffer)
    expect(audio).toHaveLength(1)
    expect(audio[0].byteLength).toBe(640)
  })

  it('capture 每帧回报 RMS 电平（20ms/50Hz），供星团逐帧驱动', () => {
    const { Processor, posted } = loadProcessor('captureProcessor.js', 16000)
    const p = new Processor()
    p.process([[new Float32Array(160).fill(0.25)]], [])
    p.process([[new Float32Array(160).fill(0.25)]], [])
    const levels = posted.filter(
      (d) => d && typeof d === 'object' && '__gaeaLevel' in (d as Record<string, unknown>),
    ) as { __gaeaLevel: number }[]
    expect(levels).toHaveLength(1)
    expect(levels[0].__gaeaLevel).toBeGreaterThan(0.5)
  })

  it('sink 在 48k 设备上把 16k PCM 扩为约 3 倍样本，flush 清 ring', () => {
    const { Processor, posted } = loadProcessor('sinkProcessor.js', 48000)
    const p = new Processor()
    const pcm = new Int16Array(320)
    pcm.fill(16384)
    p.port.onmessage?.({ data: { type: 'pcm', buffer: pcm.buffer } })
    expect(Number(p.size)).toBeGreaterThan(900)

    p.port.onmessage?.({ data: { type: 'flush' } })
    expect(Number(p.size)).toBe(0)
    const flushAck = posted.find((d) => d && typeof d === 'object' && (d as {flushed?: boolean}).flushed)
    expect(flushAck).toMatchObject({ flushed: true, buffered: 0 })
  })

  it('sink 20Hz 回报声卡实际外放电平（含欠载静音）', () => {
    const { Processor, posted } = loadProcessor('sinkProcessor.js', 16000)
    const p = new Processor() as unknown as {
      port: { onmessage: ((event: { data: unknown }) => void) | null }
      process: (i: Float32Array[][], o: Float32Array[][]) => boolean
    }
    const pcm = new Int16Array(320).fill(16384)
    p.port.onmessage?.({ data: { type: 'pcm', buffer: pcm.buffer } })
    for (let f = 0; f < 6; f++) p.process([], [[new Float32Array(128)]])
    const levels = posted.filter(
      (d) => d && typeof d === 'object' && '__gaeaPlayLevel' in (d as Record<string, unknown>),
    ) as { __gaeaPlayLevel: number }[]
    expect(levels.length).toBeGreaterThan(0)
    expect(levels[0].__gaeaPlayLevel).toBeGreaterThan(0.4)
    // flush 后外放静音，电平必须归零（不能停在上一句的峰值）
    p.port.onmessage?.({ data: { type: 'flush' } })
    for (let f = 0; f < 12; f++) p.process([], [[new Float32Array(128)]])
    const after = posted.filter(
      (d) => d && typeof d === 'object' && '__gaeaPlayLevel' in (d as Record<string, unknown>),
    ) as { __gaeaPlayLevel: number }[]
    expect(after[after.length - 1].__gaeaPlayLevel).toBe(0)
  })

  it('sink 诊断 tap：process 实际外放样本按 ≈1s 攒批回传（2026-11-07 重叠线复开）', () => {
    const { Processor, posted } = loadProcessor('sinkProcessor.js', 16000)
    const p = new Processor() as unknown as {
      port: { onmessage: ((event: { data: unknown }) => void) | null }
      process: (i: Float32Array[][], o: Float32Array[][]) => boolean
      tapOn: boolean
      tapBatchSamples: number
    }
    // 默认开启；攒满一批（16000 样本）须回传一条 tap 消息，值与外放一致
    expect(p.tapOn).toBe(true)
    expect(p.tapBatchSamples).toBe(16000)
    const tapCount = () =>
      posted.filter((d) => d && typeof d === 'object' && (d as { type?: string }).type === 'tap').length
    // 先灌 ring（16001 个 0.5 样本，首样本被插值基准吞掉→恰放出 16000），再放 125 帧
    const pcm = new Int16Array(16001).fill(16384)
    p.port.onmessage?.({ data: { type: 'pcm', buffer: pcm.buffer } })
    // 128 样本/帧 × 125 帧 = 16000（fill 报告消息独立计数不干扰 tap 断言）
    for (let f = 0; f < 125; f++) {
      p.process([], [[new Float32Array(128)]])
    }
    expect(tapCount()).toBe(1)
    const taps = posted.filter(
      (d) => d && typeof d === 'object' && (d as { type?: string }).type === 'tap',
    ) as { rate: number; buffer: ArrayBuffer }[]
    expect(taps[0].rate).toBe(16000)
    const samples = new Int16Array(taps[0].buffer)
    expect(samples.length).toBe(16000)
    expect(samples[0]).toBe(16383)
    expect(samples[15999]).toBe(16383)
    // tap 关闭后不再回传（fill 报告照常，不计入 tap 数）
    p.port.onmessage?.({ data: { type: 'tap', on: false } })
    expect(p.tapOn).toBe(false)
    p.process([], [[new Float32Array(128).fill(0.5)]])
    expect(tapCount()).toBe(1)
  })

  it('sink ring 容量 30s + 环满整块拒绝（2026-11-07 修复 B/C：逐样本顶替=时间轴压缩+毛刺=「加速+杂音」签名，废）', () => {
    const { Processor, posted } = loadProcessor('sinkProcessor.js', 16000)
    const p = new Processor() as unknown as {
      port: { onmessage: ((event: { data: unknown }) => void) | null }
      cap: number
      size: number
      process: (i: Float32Array[][], o: Float32Array[][]) => boolean
    }
    // 修复 C：容量 30s@16k = 480000（不再 4s=64000）
    expect(p.cap).toBe(480000)
    // 灌到拒绝边界：299 块（块1 吞 1 基准样本 → 478399），第 300 块预检
    // （478399+1602 > 480000）整块拒——预检即修复 B 的第一道闸
    for (let b = 0; b < 299; b++) {
      p.port.onmessage?.({ data: { type: 'pcm', buffer: new Int16Array(1600).fill(100).buffer } })
    }
    expect(p.size).toBe(478399)
    posted.length = 0
    // 再灌一块：整块拒绝——size 不变、rejected 上报、无逐样本顶替（size 恒定）
    const extra = new Int16Array(1600).fill(500)
    p.port.onmessage?.({ data: { type: 'pcm', buffer: extra.buffer } })
    expect(p.size).toBe(478399)
    const rej = posted.find((d) => (d as { type?: string }).type === 'rejected') as { samples: number }
    expect(rej).toBeDefined()
    expect(rej.samples).toBe(1600)
    // 消费一些空间后拒绝解除，后续块正常写入
    p.process([], [[new Float32Array(3200)]]) // 消费 3200
    expect(p.size).toBe(475199)
    p.port.onmessage?.({ data: { type: 'pcm', buffer: new Int16Array(1600).fill(300).buffer } })
    expect(p.size).toBe(476799)
  })
})

describe('capture 自适应输入增益（2026-11-05 真麦根修）', () => {
  function int16Frames(posted: unknown[]): Int16Array[] {
    return posted.filter((d): d is ArrayBuffer => d instanceof ArrayBuffer).map((b) => new Int16Array(b))
  }

  it('弱电平（RMS≈0.01）逐秒升档到 4 倍，送达电平进入过门档', () => {
    const { Processor, posted } = loadProcessor('captureProcessor.js', 16000)
    const p = new Processor() as unknown as { gain: number; process: (i: Float32Array[][], o: Float32Array[][]) => boolean }
    // 静默底噪直接喂说话段（幅值 0.03 正弦 ≈ RMS 0.021……取 0.014 正弦 RMS≈0.0099）
    const N = 16000 * 3
    for (let off = 0; off < N; off += 160) {
      const chunk = new Float32Array(160)
      for (let i = 0; i < 160; i++) chunk[i] = 0.014 * Math.sin((2 * Math.PI * 220 * (off + i)) / 16000)
      p.process([[chunk]], [])
    }
    // 3 秒评估：1→2→4
    expect(p.gain).toBe(4)
    // 送达帧应被放大 4 倍：满幅帧 int16 ≈ 0.014*4≈0.056→1836 附近（非 459）
    const frames = int16Frames(posted)
    const last = frames[frames.length - 1]
    const peak16 = Math.max(...Array.from(last, Math.abs))
    expect(peak16).toBeGreaterThan(1000)
  })

  it('正常电平（RMS 0.04 合成语料档）增益恒 1', () => {
    const { Processor } = loadProcessor('captureProcessor.js', 16000)
    const p = new Processor() as unknown as { gain: number; process: (i: Float32Array[][], o: Float32Array[][]) => boolean }
    const N = 16000 * 2
    for (let off = 0; off < N; off += 160) {
      const chunk = new Float32Array(160)
      for (let i = 0; i < 160; i++) chunk[i] = 0.06 * Math.sin((2 * Math.PI * 220 * (off + i)) / 16000)
      p.process([[chunk]], [])
    }
    expect(p.gain).toBe(1)
  })

  it('越峰保护：原始 peak>0.22（×4 逼近满幅）立即回 1', () => {
    const { Processor } = loadProcessor('captureProcessor.js', 16000)
    const p = new Processor() as unknown as { gain: number; process: (i: Float32Array[][], o: Float32Array[][]) => boolean }
    // 先喂弱电平 2 秒升到 4 倍（0.008 正弦 RMS≈0.0057，两次评估 1→2→4）
    for (let off = 0; off < 16000 * 2; off += 160) {
      const chunk = new Float32Array(160)
      for (let i = 0; i < 160; i++) chunk[i] = 0.008 * Math.sin((2 * Math.PI * 220 * (off + i)) / 16000)
      p.process([[chunk]], [])
    }
    expect(p.gain).toBe(4)
    // 再喂 peak 0.3 正弦（0.3×4=1.2 会削波）→ 回 1
    for (let off = 0; off < 16000; off += 160) {
      const chunk = new Float32Array(160)
      for (let i = 0; i < 160; i++) chunk[i] = 0.3 * Math.sin((2 * Math.PI * 220 * (off + i)) / 16000)
      p.process([[chunk]], [])
    }
    expect(p.gain).toBe(1)
  })
})
