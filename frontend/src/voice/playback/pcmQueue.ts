/**
 * voice/playback/pcmQueue.ts — 下行 PCM 播放队列（M2，纯逻辑无 DOM）。
 *
 * V2 方案 §10.2：播放器只认 (response_id, PCM16, 到达序)，barge-in 只冲刷
 * 未播块；播放完成回执必须带 response id，不用全局布尔。
 * 代际生命周期由协议事件驱动：response.created → open(id)；音频 delta →
 * append(id)；response.done → markClosed(id)。回执=该响应闭合且其音频排空，
 * 一次性。
 *
 * 排队语义（2026-11-07 用户实测「一大段散文中间被打断」+ 拍板「应该排队」，
 * 上游 PlaybackQueue 全局 FIFO 同位——audio_client.py 播放队列无代际丢弃，
 * 新响应音频排在旧响应之后，仅 barge-in/取消的 flush 才清队）：open 新代际
 * 不再作废旧代际存量；旧「新代际作废」在半双工/关打断模式下会把上一响应
 * 未播音频掐掉=回复断续。本模块是「调度面」：真正的声音输出由 sink
 * （AudioWorklet / 测试桩）拉取。
 */

export interface PlaybackStats {
  /** 已入队 PCM 字节数（全会话累计） */
  enqueuedBytes: number
  /** 已被 sink 拉走的字节数 */
  playedBytes: number
  /** 因冲刷被丢弃的字节数 */
  droppedBytes: number
  /** 当前持有 response id */
  currentResponseId: string
  /** 当前 response 是否已终态闭合 */
  closed: boolean
  /** 当前 response 闭合且排空后一次性置位（回执） */
  playbackDone: boolean
  /** 未播块数 */
  pendingChunkCount: number
}

export type PcmSink = (pcm: Int16Array, responseId: string) => void
export type PcmSinkDone = (responseId: string) => void

interface queueChunk {
  id: string
  pcm: Int16Array
}

export class PlaybackQueue {
  private chunks: queueChunk[] = []
  private pendingBytes = 0
  private pendingById = new Map<string, number>()
  private playedBytes = 0
  private droppedBytes = 0
  private currentId = ''
  private closedIds = new Set<string>()
  private doneFiredIds = new Set<string>()
  private doneCb: PcmSinkDone | null = null

  /**
   * response.created(id) → 开启新代际。旧代际存量保留排队（上游 PlaybackQueue
   * 全局 FIFO 同位：新响应音频排在旧响应之后，不作废）。
   */
  open(responseId: string): void {
    if (responseId === this.currentId) return
    this.currentId = responseId
  }

  /** 服务端 output_audio.delta 入队（base64 已在上层解码为 Int16）。未 open 过的会话期前置帧防御性丢弃记账。 */
  append(responseId: string, pcm: Int16Array): void {
    if (this.currentId === '') {
      this.droppedBytes += pcm.byteLength
      return
    }
    this.chunks.push({ id: responseId, pcm })
    this.pendingBytes += pcm.byteLength
    this.pendingById.set(responseId, (this.pendingById.get(responseId) ?? 0) + pcm.byteLength)
  }

  /** response.done → 该响应闭合。其音频排空后回执（FIFO 到达序，逐响应一次性）。 */
  markClosed(responseId: string): void {
    this.closedIds.add(responseId)
    this.maybeFireDone(responseId)
  }

  /** barge-in / response.cancel：冲刷全部未播块。已闭合未回执的响应在此收口。返回冲刷字节数。 */
  flush(): number {
    const dropped = this.pendingBytes
    this.droppedBytes += dropped
    this.chunks = []
    this.pendingBytes = 0
    this.pendingById.clear()
    for (const id of [...this.closedIds]) this.fireDone(id)
    return dropped
  }

  /** 播放完成回执订阅（闭合且该响应音频排空时一次性触发；含 flush 收尾路径） */
  onDone(cb: PcmSinkDone): void {
    this.doneCb = cb
  }

  /** 探测器用：响应是否已终态闭合 */
  isClosed(responseId: string): boolean {
    return this.closedIds.has(responseId)
  }

  /** sink 拉取一块（到达序=播放序）；无数据返回 null。某响应末块拉走且已闭合时触发其回执。 */
  pull(): Int16Array | null {
    return this.pullChunk()?.pcm ?? null
  }

  /** 会话重建：清一切 */
  reset(): void {
    this.flush()
    this.currentId = ''
    this.playedBytes = 0
    this.droppedBytes = 0
  }

  /** 便捷：一次性拉空并喂给 sink（测试/非流式场景） */
  drainTo(sink: PcmSink): void {
    for (let c = this.pullChunk(); c; c = this.pullChunk()) {
      sink(c.pcm, c.id)
    }
  }

  private pullChunk(): queueChunk | null {
    const c = this.chunks.shift()
    if (!c) return null
    this.pendingBytes -= c.pcm.byteLength
    const left = (this.pendingById.get(c.id) ?? 0) - c.pcm.byteLength
    if (left > 0) this.pendingById.set(c.id, left)
    else this.pendingById.delete(c.id)
    this.playedBytes += c.pcm.byteLength
    this.maybeFireDone(c.id)
    return c
  }

  private fireDone(id: string): void {
    if (this.doneFiredIds.has(id)) return
    this.doneFiredIds.add(id)
    this.doneCb?.(id)
  }

  private maybeFireDone(id: string): void {
    if (!this.closedIds.has(id) || this.doneFiredIds.has(id)) return
    if (this.pendingById.has(id)) return // 该响应还有未播存量
    this.fireDone(id)
  }

  get stats(): PlaybackStats {
    return {
      enqueuedBytes: this.playedBytes + this.pendingBytes,
      playedBytes: this.playedBytes,
      droppedBytes: this.droppedBytes,
      currentResponseId: this.currentId,
      closed: this.closedIds.has(this.currentId),
      playbackDone: this.doneFiredIds.has(this.currentId),
      pendingChunkCount: this.chunks.length,
    }
  }
}
