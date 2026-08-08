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
}
