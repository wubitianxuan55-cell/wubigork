// annotationMarks.ts — 标注 → 正文 code-unit 高亮段（t7：分析面板高亮视图与
// 编辑器持久 overlay 共用的唯一实现； rune 偏移 ↔ code-unit 换算口径收敛于此）。
import type { ChapterAnnotation } from '../../../gaea/lib/bridge/novel'

/** rune 偏移 → code-unit 下标（代理对按 codePoint 步进；EditorPanel toRune 的逆函数）。 */
export function runeToCodeUnit(text: string, runePos: number): number {
  let runes = 0
  let i = 0
  while (i < text.length && runes < runePos) {
    const cp = text.codePointAt(i) ?? 0
    i += cp > 0xffff ? 2 : 1
    runes += 1
  }
  return i
}

export interface AnnSegment {
  start: number // code-unit
  end: number // code-unit
  ann: ChapterAnnotation
}

/**
 * 高亮段（Q5：跳过 pos<0/length≤0；pos 升序；交叠后段起点钳到前段终点；
 * 越界裁剪）。输入 anns 顺序无关，输出段保留原 ann 引用（列表行反查用）。
 */
export function buildAnnSegments(text: string, anns: ChapterAnnotation[]): AnnSegment[] {
  const totalRunes = [...text].length
  const hits = anns
    .filter(a => a.pos >= 0 && (a.length ?? 0) > 0)
    .slice()
    .sort((x, y) => x.pos - y.pos)
  const segs: AnnSegment[] = []
  let cursorRunes = 0
  for (const a of hits) {
    const startRune = Math.max(a.pos, cursorRunes)
    const endRune = Math.min(a.pos + (a.length ?? 0), totalRunes)
    if (endRune <= startRune) continue
    const start = runeToCodeUnit(text, startRune)
    const end = runeToCodeUnit(text, endRune)
    if (end <= start) continue
    segs.push({ start, end, ann: a })
    cursorRunes = endRune
  }
  return segs
}
