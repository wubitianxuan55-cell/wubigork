/**
 * 小说全文检索 API：按章节标题 + 正文扫描当前项目。
 */
import * as App from '../../../../src/wailsjsCompat'

export interface NovelSearchHit {
  node_id: string
  title: string
  chapter_num: number
  branch?: string
  snippet: string
  title_hit?: boolean
}

/** 全文检索（大小写不敏感；一章最多一个命中，上限 100 条） */
export async function searchNovel(query: string): Promise<NovelSearchHit[]> {
  return App.NovelSearch(query)
}
