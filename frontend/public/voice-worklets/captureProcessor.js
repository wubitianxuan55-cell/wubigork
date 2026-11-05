/**
 * voice/worklets/captureProcessor.js — 麦克风采集 AudioWorklet（M2）。
 *
 * 128 帧量子 → 攒满 320 样本（20ms@16k）→ Float→Int16 饱和量化 →
 * postMessage(ArrayBuffer, transfer) 给主线程（V2 方案 §10.1：
 * 不走 ScriptProcessor、不做逐块 base64——那是 V1 的 IndexSizeError 雷）。
 * 优先请求 16k AudioContext；设备拒绝时 worklet 按实际 sampleRate 抗混叠
 * 重采样回 16k，再切 320 样本帧。绝不能把 48k 的 320 帧直接冒充 16k。
 */
class CaptureProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    this.target = 320
    this.buf = new Float32Array(this.target)
    this.n = 0
    const sourceRate = typeof sampleRate === 'number' && sampleRate > 0 ? sampleRate : 16000
    this.ratio = sourceRate / 16000
    this.phase = 0
    this.sum = 0
    this.count = 0
    // 电平上报（哑麦诚实化）：每 12 帧（~240ms）回报一次帧内 RMS，
    // 主线程据此驱动真实电平可视化与「无信号」检测。
    this.levelEvery = 12
    this.levelCountdown = this.levelEvery
    // 自适应输入增益（2026-11-05 真麦根修第二刀）：弱电平麦克风（实弹 MIC-H
    // 说话 RMS≈0.01，silero 概率崩——阈值降到 0.5 仍全 discard，active 仅
    // 64~192ms；合成语料标定分界≈0.012，能过门的档≈0.04）自动抬增益，
    // 正常电平设备恒 1 不动。包络按「增益前」原始电平统计（防增益自激励
    // 振荡），每 1s@16k 评估一次：原始 RMS<0.012 升档 ×2 上限 4 倍，>0.06
    // 立即回 1 防削波；量化前施加，量化已有 clamp 收尾。
    this.gain = 1
    this.envSum = 0
    this.envCount = 0
    this.envPeak = 0
    this.evalCountdown = 16000
  }

  process(inputs) {
    const input = inputs[0]
    if (!input || input.length === 0) return true
    const ch = input[0]
    if (!ch) return true
    for (let i = 0; i < ch.length; i++) {
      const sample = ch[i]
      this.sum += sample
      this.count++
      this.phase += 1
      while (this.phase >= this.ratio) {
        this.phase -= this.ratio
        const down = this.count > 0 ? this.sum / this.count : sample
        this.sum = 0
        this.count = 0
        this.push(down)
      }
    }
    return true
  }

  push(sample) {
    // 包络统计用增益前原始电平；同步推进评估窗口（16k 域样本）。
    this.envSum += sample * sample
    this.envCount++
    const abs = sample < 0 ? -sample : sample
    if (abs > this.envPeak) this.envPeak = abs
    if (--this.evalCountdown <= 0) {
      this.evalCountdown = 16000
      if (this.envCount > 0) {
        const rms = Math.sqrt(this.envSum / this.envCount)
        if (rms < 0.012 && this.gain < 4) {
          this.gain = Math.min(4, this.gain * 2)
        } else if (this.envPeak > 0.22 && this.gain > 1) {
          // 越峰保护看原始 peak（削波=peak×gain>1；0.22×4=0.88 逼近满幅），
          // 且原始 peak>0.22 说明信号足够强本就过得了门，回 1 无损。
          this.gain = 1
        }
      }
      this.envSum = 0
      this.envCount = 0
      this.envPeak = 0
    }
    this.buf[this.n++] = sample
    if (this.n === this.target) {
      let sq = 0
      const int16 = new Int16Array(this.target)
      for (let j = 0; j < this.target; j++) {
        const s = Math.max(-1, Math.min(1, this.buf[j] * this.gain))
        sq += s * s
        int16[j] = s < 0 ? s * 0x8000 : s * 0x7fff
      }
      this.port.postMessage(int16.buffer, [int16.buffer])
      this.n = 0
      // RMS 电平（0..1 线性刻度，增益后=实际送达强度）：语音典型 0.02~0.35，静音 ≈0
      if (--this.levelCountdown <= 0) {
        this.levelCountdown = this.levelEvery
        const rms = Math.sqrt(sq / this.target)
        this.port.postMessage({ __gaeaLevel: Math.min(1, rms * 8) })
      }
    }
  }
}

registerProcessor('gaea-voice-capture', CaptureProcessor)
