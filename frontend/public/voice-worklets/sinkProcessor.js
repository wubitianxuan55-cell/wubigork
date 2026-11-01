/**
 * voice/worklets/sinkProcessor.js — 下行 PCM 播放 AudioWorklet（M2）。
 *
 * 主线程按到达序 postMessage(Int16Array.buffer)（PlaybackQueue 是唯一排序面），
 * worklet 内写环形缓冲，process() 从环读出填输出声道；欠载输出静音。
 * 20ms 内缓冲超过容量（4s@16k）时丢最旧数据并上报 overflow——真实链路下
 * PlaybackQueue 冲刷语义应让 overflow 恒为 0，出现即上游泄漏信号。
 */
class SinkProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    this.cap = 65536 // 4s @16k
    this.ring = new Float32Array(this.cap)
    this.rw = 0
    this.ww = 0
    this.size = 0
    this.overflow = 0
    this.port.onmessage = (e) => {
      const int16 = new Int16Array(e.data)
      for (let i = 0; i < int16.length; i++) {
        if (this.size >= this.cap) {
          // 环满丢最旧：读指针前移
          this.rw = (this.rw + 1) % this.cap
          this.size--
          this.overflow++
        }
        this.ring[this.ww] = int16[i] / 32768
        this.ww = (this.ww + 1) % this.cap
        this.size++
      }
    }
  }

  process(_inputs, outputs) {
    const out = outputs[0]
    if (!out || out.length === 0) return true
    const ch = out[0]
    for (let i = 0; i < ch.length; i++) {
      if (this.size > 0) {
        ch[i] = this.ring[this.rw]
        this.rw = (this.rw + 1) % this.cap
        this.size--
      } else {
        ch[i] = 0
      }
    }
    // 其余声道复制第 0 声道（单声道源 → 立体声设备）
    for (let c = 1; c < out.length; c++) {
      out[c].set(ch)
    }
    if (this.overflow > 0) {
      this.port.postMessage({ overflow: this.overflow })
      this.overflow = 0
    }
    return true
  }
}

registerProcessor('gaea-voice-sink', SinkProcessor)
