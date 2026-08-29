/**
 * 阅读划线/高亮/想法（按项目持久化，条目归属章节）
 *  - text: 选中摘录（渲染时按文本在段落中回定位高亮）
 *  - color: yellow / green / blue / pink
 *  - note: 想法批注（可空）
 */
export type AnnotationColor = 'yellow' | 'green' | 'blue' | 'pink'

export interface ReadingAnnotation {
  id: string
  nodeId: string
  title: string
  color: AnnotationColor
  text: string
  note: string
  createdAt: number
}

export const ANNOTATION_COLORS: Record<AnnotationColor, string> = {
  yellow: '#f6c445', // hex-exempt 用户标注数据色（持久化色板）
  green: '#7bc96f', // hex-exempt 用户标注数据色（持久化色板）
  blue: '#6fa8dc', // hex-exempt 用户标注数据色（持久化色板）
  pink: '#e58aa6', // hex-exempt 用户标注数据色（持久化色板）
}

const ANNOTATIONS_KEY = 'gaea.novel.readingAnnotations.'

export function readAnnotations(projectPath: string): ReadingAnnotation[] {
  try {
    if (!projectPath) return []
    const raw = localStorage.getItem(ANNOTATIONS_KEY + projectPath)
    if (!raw) return []
    const list = JSON.parse(raw) as ReadingAnnotation[]
    if (!Array.isArray(list)) return []
    return list.filter(
      (a) => a && typeof a.id === 'string' && typeof a.nodeId === 'string' && typeof a.text === 'string' && a.text.length > 0,
    )
  } catch {
    return []
  }
}

export function writeAnnotations(projectPath: string, annotations: ReadingAnnotation[]) {
  try {
    if (!projectPath) return
    localStorage.setItem(ANNOTATIONS_KEY + projectPath, JSON.stringify(annotations))
  } catch { /* ignore */ }
}
