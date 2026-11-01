/**
 * voice/worklets/captureProcessor.js — 麦克风采集 AudioWorklet（M2）。
 *
 * 128 帧量子 → 攒满 320 样本（20ms@16k）→ Float→Int16 饱和量化 →
 * postMessage(ArrayBuffer, transfer) 给主线程（V2 方案 §10.1：
 * 不走 ScriptProcessor、不做逐块 base64——那是 V1 的 IndexSizeError 雷）。
 * 采样率由 AudioContext(16000) 强制；设备级 sampleRate 约束冗余且是 Chromium 雷，不加。
 */
class CaptureProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    this.target = 320
    this.buf = new Float32Array(this.target)
    this.n = 0
  }

  process(inputs) {
    const input = inputs[0]
    if (!input || input.length === 0) return true
    const ch = input[0]
    if (!ch) return true
    for (let i = 0; i < ch.length; i++) {
      this.buf[this.n++] = ch[i]
      if (this.n === this.target) {
        const int16 = new Int16Array(this.target)
        for (let j = 0; j < this.target; j++) {
          const s = Math.max(-1, Math.min(1, this.buf[j]))
          int16[j] = s < 0 ? s * 0x8000 : s * 0x7fff
        }
        this.port.postMessage(int16.buffer, [int16.buffer])
        this.n = 0
      }
    }
    return true
  }
}

registerProcessor('gaea-voice-capture', CaptureProcessor)
