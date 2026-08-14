/**
 * historyMeta.ts — 绘梦历史元数据序列化 / 恢复（纯逻辑，可单测）
 *
 * T6-4.3 历史图片可恢复：
 * - 后端生成时已把落盘路径写入 file_path；历史元数据保存时必须保留该字段。
 * - 重启加载时：无 base64 但有 file_path 的条目，通过后端文件读取绑定
 *   （App.GaeaAttachmentDataURL）把路径转 dataURL 回填。
 * - localStorage 容量保护（分级策略）：base64 超过 INLINE_IMAGE_MAX 时
 *   只保存 file_path，小图才内联。
 */

import type { GenResult } from './types'

export const HISTORY_META_KEY = 'gaea.imagegen.historyMeta'

/** 内联 base64 长度阈值（字符数）：超过则历史元数据只保存 file_path，避免 localStorage 膨胀 */
export const INLINE_IMAGE_MAX = 200_000

/** 是否为可内联的 data URL（远程 URL 不内联） */
export function isInlineDataUrl(image: string): boolean {
  return typeof image === 'string' && image.startsWith('data:')
}

/** 历史元数据序列化：小图内联 base64，大图只保留 file_path（分级策略） */
export function serializeHistoryMeta(history: GenResult[]): GenResult[] {
  return history.map(({ image, ...rest }) => {
    if (image && isInlineDataUrl(image) && image.length <= INLINE_IMAGE_MAX) {
      return { ...rest, image }
    }
    return { ...rest, image: '' }
  })
}

/** 需要从本地文件回填：无 base64 但有 file_path */
export function needsFileRestore(item: GenResult): boolean {
  return !item.image && !!item.file_path
}

export type FileReader = (path: string) => Promise<string>

/**
 * 批量恢复历史图片：无 base64 但有 file_path 的条目，调用后端文件读取绑定
 * 把路径转 dataURL 回填 image。读取失败保留原条目（前端显示占位），
 * 失败详情记录到 console（不静默吞错，调用方合并时无需处理失败项）。
 */
export async function restoreHistoryImages(
  history: GenResult[],
  readFile: FileReader,
): Promise<GenResult[]> {
  const restored: GenResult[] = []
  for (const item of history) {
    if (!needsFileRestore(item)) continue
    try {
      const dataUrl = await readFile(item.file_path as string)
      restored.push({ ...item, image: dataUrl })
    } catch (err) {
      console.warn(`[imagegen] 历史图片恢复失败: ${item.file_path}`, err)
    }
  }
  return restored
}
