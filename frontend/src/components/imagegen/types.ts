export interface GenResult {
  image: string
  seed: number
  time: number
  prompt: string
  negative?: string
  model: string
  size: string
  style?: string
}
