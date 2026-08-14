/**
 * media.ts — 绘梦媒体判定与命名（纯逻辑，可单测）
 *
 * T6-4.2 格式修正：t2v 实际输出动画 WebP（data:image/webp），
 * 但下载曾按 kind==='video' 固定命名 .mp4；预览用 <video> 播 webp 也不播。
 * 这里统一按「实际 data URL 前缀」推断扩展名与渲染元素。
 */

import type { GenResult } from './types'

/** 根据实际媒体 data URL 推断扩展名（t2v 可能输出 webp 而非 mp4） */
export function mediaExtension(dataUrl: string): string {
  if (!dataUrl) return '.png'
  if (dataUrl.startsWith('data:video/mp4')) return '.mp4'
  if (dataUrl.startsWith('data:video/webm')) return '.webm'
  if (dataUrl.startsWith('data:video/webp')) return '.webp'
  if (dataUrl.startsWith('data:video/quicktime')) return '.mov'
  if (dataUrl.startsWith('data:image/webp')) return '.webp'
  if (dataUrl.startsWith('data:image/gif')) return '.gif'
  if (dataUrl.startsWith('data:image/jpeg') || dataUrl.startsWith('data:image/jpg')) return '.jpg'
  return '.png'
}

/** 下载文件名：按实际媒体类型命名（t2v webp → .webp，而非固定 .mp4） */
export function downloadFileName(item: Pick<GenResult, 'image' | 'seed'>, now: number = Date.now()): string {
  return `gaea-${now}-seed${item.seed}${mediaExtension(item.image || '')}`
}

/** 是否为视频数据（data:video/*）——用于选择 <video> 还是 <img> 渲染 */
export function mediaIsVideo(dataUrl: string): boolean {
  return typeof dataUrl === 'string' && dataUrl.startsWith('data:video/')
}
