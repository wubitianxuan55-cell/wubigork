export interface GenResult {
  image: string
  seed: number
  time: number
  prompt: string
  negative?: string
  model: string
  size: string
  style?: string
  /** image | video（视频为 mp4/webm，动画 webp/gif 也标为 image） */
  kind?: 'image' | 'video'
  mode?: 'txt2img' | 'img2img' | 't2v'
  count?: number
  selectedLoras?: string[]
  denoise?: number
  frames?: number
  fps?: number
  customWidth?: number
  customHeight?: number
}

export type ImageMode = 'txt2img' | 'img2img' | 't2v'

export interface GenTask {
  prompt: string
  negative: string
  size: string
  customWidth: number
  customHeight: number
  model: string
  seed: number
  count: number
  selectedLoras: string[]
  mode: ImageMode
  initImage: string
  denoise: number
  frames: number
  fps: number
}

export type QueueStatus = 'pending' | 'running' | 'done' | 'failed' | 'canceled'

export interface QueueEntry {
  id: number
  task: GenTask
  status: QueueStatus
}
