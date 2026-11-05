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
    expect(posted).toHaveLength(1)
    expect((posted[0] as ArrayBuffer).byteLength).toBe(320 * 2)
  })

  it('capture 在原生 16k 输入上仍逐样本成帧', () => {
    const { Processor, posted } = loadProcessor('captureProcessor.js', 16000)
    const p = new Processor()
    p.process([[new Float32Array(160).fill(0.1)]], [])
    p.process([[new Float32Array(160).fill(0.2)]], [])
    expect(posted).toHaveLength(1)
    expect((posted[0] as ArrayBuffer).byteLength).toBe(640)
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
    expect(posted).toContainEqual({ flushed: true })
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
