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
  /** 本地保存路径（后端 T6-4.3 写入；历史图片经此恢复，下载/剧照优先取此） */
  file_path?: string
  mode?: 'txt2img' | 'img2img' | 't2v'
  count?: number
  selectedLoras?: string[]
  denoise?: number
  frames?: number
  fps?: number
  /** T2 角色参考槽：本次生成引用的角色 ID（进溯源登记） */
  characterId?: string
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
  /** T2 角色参考槽：角色 ID 与全部参考图（data URL；首张作图生图种子） */
  characterId?: string
  refImages?: string[]
}

export type QueueStatus = 'pending' | 'running' | 'done' | 'failed' | 'canceled'

export interface QueueEntry {
  id: number
  task: GenTask
  status: QueueStatus
}
