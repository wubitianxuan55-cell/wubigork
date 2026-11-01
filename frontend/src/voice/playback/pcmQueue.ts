/**
 * voice/playback/pcmQueue.ts — 下行 PCM 播放队列（M2，纯逻辑无 DOM）。
 *
 * V2 方案 §10.2：播放器只认 (response_id, PCM16, 到达序)，barge-in 只冲刷
 * 当前代际未播块；播放完成回执必须带 response id，不用全局布尔。
 * 代际生命周期由协议事件驱动：response.created → open(id)；音频 delta →
 * append(id)；response.done → markClosed(id)。回执=闭合且排空，一次性。
 * 本模块是「调度面」：真正的声音输出由 sink（AudioWorklet / 测试桩）拉取。
 */

export interface PlaybackStats {
  /** 已入队 PCM 字节数（当前代际） */
  enqueuedBytes: number
  /** 已被 sink 拉走的字节数 */
  playedBytes: number
  /** 因代际过期/冲刷被丢弃的字节数 */
  droppedBytes: number
  /** 当前持有 response id */
  currentResponseId: string
  /** 当前 response 是否已终态闭合 */
  closed: boolean
  /** 终态且排空后一次性置位（回执） */
  playbackDone: boolean
  /** 未播块数 */
  pendingChunkCount: number
}

export type PcmSink = (pcm: Int16Array, responseId: string) => void
export type PcmSinkDone = (responseId: string) => void

export class PlaybackQueue {
  private chunks: Int16Array[] = []
  private pendingBytes = 0
  private playedBytes = 0
  private droppedBytes = 0
  private currentId = ''
  private closed = false
  private doneFired = false
  private doneCb: PcmSinkDone | null = null

  /**
   * response.created(id) → 开启新代际。旧代际存量全部作废丢弃
   * （上游 cancel 已冲刷过则为 0；此处是队列自卫，不依赖上游纪律）。
   */
  open(responseId: string): void {
    if (responseId === this.currentId) return
    this.droppedBytes += this.pendingBytes
    this.chunks = []
    this.pendingBytes = 0
    this.currentId = responseId
    this.closed = false
    this.doneFired = false
  }

  /** 服务端 output_audio.delta 入队（base64 已在上层解码为 Int16）。非当前代际一律丢弃记账。 */
  append(responseId: string, pcm: Int16Array): void {
    if (responseId !== this.currentId) {
      this.droppedBytes += pcm.byteLength
      return
    }
    this.chunks.push(pcm)
    this.pendingBytes += pcm.byteLength
  }

  /** response.done → 关闭当前代际。队列排空后 playbackDone 置位（一次性回执）。 */
  markClosed(responseId: string): void {
    if (responseId !== this.currentId) return
    this.closed = true
    this.maybeFireDone()
  }

  /** barge-in / response.cancel：冲刷未播块。已闭合时冲刷即回执。返回冲刷字节数。 */
  flush(): number {
    const dropped = this.pendingBytes
    this.droppedBytes += dropped
    this.chunks = []
    this.pendingBytes = 0
    this.maybeFireDone()
    return dropped
  }

  /** 播放完成回执订阅（闭合且排空时一次性触发；含 flush 收尾路径） */
  onDone(cb: PcmSinkDone): void {
    this.doneCb = cb
  }

  /** sink 拉取一块（到达序=播放序）；无数据返回 null。末块拉走后按闭合态触发回执。 */
  pull(): Int16Array | null {
    const chunk = this.chunks.shift()
    if (!chunk) return null
    this.pendingBytes -= chunk.byteLength
    this.playedBytes += chunk.byteLength
    this.maybeFireDone()
    return chunk
  }

  /** 便捷：一次性拉空并喂给 sink（测试/非流式场景） */
  drainTo(sink: PcmSink): void {
    for (let chunk = this.pull(); chunk; chunk = this.pull()) {
      sink(chunk, this.currentId)
    }
  }

  /** 会话重建：清一切 */
  reset(): void {
    this.flush()
    this.currentId = ''
    this.closed = false
    this.doneFired = false
    this.playedBytes = 0
    this.droppedBytes = 0
  }

  private maybeFireDone(): void {
    if (this.closed && this.pendingBytes === 0 && this.currentId !== '' && !this.doneFired) {
      this.doneFired = true
      this.doneCb?.(this.currentId)
    }
  }

  get stats(): PlaybackStats {
    return {
      enqueuedBytes: this.playedBytes + this.pendingBytes,
      playedBytes: this.playedBytes,
      droppedBytes: this.droppedBytes,
      currentResponseId: this.currentId,
      closed: this.closed,
      playbackDone: this.doneFired,
      pendingChunkCount: this.chunks.length,
    }
  }
}
