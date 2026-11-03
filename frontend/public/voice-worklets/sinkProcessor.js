/**
 * voice/worklets/sinkProcessor.js — 下行 PCM 播放 AudioWorklet（M2）。
 *
 * 主线程按到达序 postMessage(Int16Array.buffer)（PlaybackQueue 是唯一排序面），
 * worklet 内先把 sidecar 的 16k PCM 线性重采样到设备实际 sampleRate，再写
 * 环形缓冲；process() 从环读出填输出声道，欠载输出静音。
 * 缓冲超过 4s 时丢最旧数据并上报 overflow——真实链路下
 * PlaybackQueue 冲刷语义应让 overflow 恒为 0，出现即上游泄漏信号。
 */
class SinkProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    const outRate = typeof sampleRate === 'number' && sampleRate > 0 ? sampleRate : 16000
    this.cap = Math.max(8192, Math.round(outRate * 4))
    this.ring = new Float32Array(this.cap)
    this.rw = 0
    this.ww = 0
    this.size = 0
    this.overflow = 0
    this.resampleStep = 16000 / outRate
    this.resamplePos = 0
    this.prevSample = 0
    this.havePrev = false
    this.port.onmessage = (e) => {
      // 控制面：barge-in 必须清掉已经泵入 worklet、尚未发出的 PCM。
      // 仅清 PlaybackQueue 不够——主线程每 100ms 会提前把多块音频交给本 ring。
      if (e.data && e.data.type === 'flush') {
        this.rw = 0
        this.ww = 0
        this.size = 0
        this.resamplePos = 0
        this.havePrev = false
        this.port.postMessage({ flushed: true })
        return
      }
      const raw = e.data && e.data.type === 'pcm' ? e.data.buffer : e.data
      if (!(raw instanceof ArrayBuffer)) return
      const int16 = new Int16Array(raw)
      for (let i = 0; i < int16.length; i++) {
        const sample = int16[i] / 32768
        if (!this.havePrev) {
          this.prevSample = sample
          this.havePrev = true
          continue
        }
        let t = this.resamplePos
        while (t < 1) {
          this.write(this.prevSample + (sample - this.prevSample) * t)
          t += this.resampleStep
        }
        this.resamplePos = t - 1
        this.prevSample = sample
      }
    }
  }

  write(sample) {
    if (this.size >= this.cap) {
      // 环满丢最旧：读指针前移
      this.rw = (this.rw + 1) % this.cap
      this.size--
      this.overflow++
    }
    this.ring[this.ww] = sample
    this.ww = (this.ww + 1) % this.cap
    this.size++
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
