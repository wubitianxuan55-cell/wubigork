export interface ReadingProgress {
  nodeId: string
  chapterNum: number
  title: string
}

const READING_KEY = 'gaea.novel.reading.'

export function readReadingProgress(projectPath: string): ReadingProgress | null {
  try {
    const raw = localStorage.getItem(READING_KEY + projectPath)
    if (!raw) return null
    const value = JSON.parse(raw) as ReadingProgress
    if (!value || !value.nodeId) return null
    return value
  } catch {
    return null
  }
}

export function writeReadingProgress(projectPath: string, progress: ReadingProgress) {
  try {
    if (!projectPath || !progress.nodeId) return
    localStorage.setItem(READING_KEY + projectPath, JSON.stringify(progress))
  } catch { /* ignore */ }
}
