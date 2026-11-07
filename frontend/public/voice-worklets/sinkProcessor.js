/**
 * voice/worklets/sinkProcessor.js — 下行 PCM 播放 AudioWorklet（M2）。
 *
 * 主线程按到达序 postMessage(Int16Array.buffer)（PlaybackQueue 是唯一排序面），
 * worklet 内先把 sidecar 的 16k PCM 线性重采样到设备实际 sampleRate，再写
 * 环形缓冲；process() 从环读出填输出声道，欠载输出静音。
 *
 * 2026-11-07 重叠线修复（tap 落盘声纹解析定罪：中段 26.5~35.7s 字被顶替残缺
 * =「语速加快+杂音」真身）：
 * - 容量 4s→30s：对齐上游 PlaybackBuffer 无上限+声卡按需拉取的背压语义
 *   （audio_client.py:245,258-266），泵水位灌注下 ring 常态远小于上限；
 * - 环满**整块拒绝**（消息级预检 + write 停写），上报 rejected——此前逐样本
 *   丢最旧（write 循环每个样本顶掉一个）=时间轴压缩+波形毛刺，即用户听感
 *   的加速与杂音；整块拒绝的听感是单次顿挫且正常路径（泵记账）永不触发；
 * - rejected 上报走独立计数，主线程可观测。
 */
class SinkProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    const outRate = typeof sampleRate === 'number' && sampleRate > 0 ? sampleRate : 16000
    this.cap = Math.max(8192, Math.round(outRate * 30))
    this.ring = new Float32Array(this.cap)
    this.rw = 0
    this.ww = 0
    this.size = 0
    this.overflow = 0 // 残留计数（write 层防御；正常路径应为 0）
    this.rejected = 0 // 整块拒绝的样本数（16k 域）
    // 水位报告节奏（约 10 次/秒，2026-11-07 提频：泵灌注参考更新鲜=双灌窗口减半）
    this.fillReportEvery = Math.max(1, Math.round(outRate / 128 / 10))
    this.fillCounter = 0
    this.resampleStep = 16000 / outRate
    this.resamplePos = 0
    this.prevSample = 0
    this.havePrev = false
    // 播放面诊断 tap（2026-11-07 重叠线复开）：process() 实际外放样本攒批回传，
    // 主线程 POST /v1/diag/playback 落盘 → 分窗声纹解析定位杂音/重叠/加速。
    // 开销面：Int16 化 + 1s 一条消息，默认开启（诊断刀，修复定形后撤）。
    this.tapOn = true
    this.tapRate = outRate
    this.tapBatchSamples = outRate // ≈1s 一批
    this.tapBuf = null
    this.tapLen = 0
    this.port.onmessage = (e) => {
      // 控制面：barge-in 必须清掉已经泵入 worklet、尚未发出的 PCM。
      // 仅清 PlaybackQueue 不够——主线程每 100ms 会提前把多块音频交给本 ring。
      if (e.data && e.data.type === 'flush') {
        this.rw = 0
        this.ww = 0
        this.size = 0
        this.resamplePos = 0
        this.havePrev = false
        this.port.postMessage({ flushed: true, buffered: 0, cap: this.cap })
        return
      }
      if (e.data && e.data.type === 'tap') {
        this.tapOn = e.data.on !== false
        if (!this.tapOn) this.tapLen = 0
        return
      }
      const raw = e.data && e.data.type === 'pcm' ? e.data.buffer : e.data
      if (!(raw instanceof ArrayBuffer)) return
      const int16 = new Int16Array(raw)
      // 整块拒绝预检（2026-11-07 修复 B）：本块重采样后样本量放不下即整块拒收。
      // 估算 n×(outRate/16000)+2 余量；块不写入=播放侧单次顿挫，绝不逐样本
      // 顶替（那是时间轴压缩+毛刺=「加速+杂音」的波形签名）。
      const need = Math.ceil(int16.length * (outRate / 16000)) + 2
      if (int16.length > 0 && this.size + need > this.cap) {
        this.rejected += int16.length
        this.port.postMessage({ type: 'rejected', samples: int16.length })
        return
      }
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
      // 防御层（预检漏网才可能到这）：停写防溢出；不逐样本顶替。
      this.overflow++
      return
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
    // 诊断 tap：实际外放样本（含欠载补零——声学真相原样）攒批回传主线程
    if (this.tapOn) {
      if (!this.tapBuf) this.tapBuf = new Int16Array(this.tapBatchSamples)
      for (let i = 0; i < ch.length; i++) {
        let s = ch[i]
        if (s > 1) s = 1
        else if (s < -1) s = -1
        this.tapBuf[this.tapLen++] = s < 0 ? s * 32768 : s * 32767
        if (this.tapLen >= this.tapBatchSamples) {
          this.port.postMessage({ type: 'tap', rate: this.tapRate, buffer: this.tapBuf.buffer.slice(0) })
          this.tapBuf = new Int16Array(this.tapBatchSamples)
          this.tapLen = 0
        }
      }
    }
    if (this.overflow > 0) {
      this.port.postMessage({ overflow: this.overflow })
      this.overflow = 0
    }
    if (this.rejected > 0) {
      this.port.postMessage({ rejected: this.rejected })
      this.rejected = 0
    }
    if (++this.fillCounter >= this.fillReportEvery) {
      this.fillCounter = 0
      this.port.postMessage({ buffered: this.size, cap: this.cap })
    }
    return true
  }
}

registerProcessor('gaea-voice-sink', SinkProcessor)
