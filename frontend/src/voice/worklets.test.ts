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
