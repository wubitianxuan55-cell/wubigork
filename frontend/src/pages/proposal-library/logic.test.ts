// logic.test.ts — 方案库板块纯逻辑：折叠/可见行/祖先、命中摘要切分、文案格式；
// 2026-11-28 起：检索词高亮与标题树筛选、编号与层级核对、结构操作、拆分预览、索引体检结论。
import { describe, expect, it } from 'vitest'
import {
  ancestorsOf,
  changeLabel,
  expandAncestors,
  filterTree,
  formatChars,
  hasChildren,
  indexLabel,
  indexSummary,
  initialCollapsed,
  mismatchHint,
  numberDepth,
  numberMismatch,
  parentOf,
  prevSiblingIndex,
  safeFileName,
  searchSummary,
  sourceLabel,
  splitByTerms,
  splitPreview,
  splitSnippet,
  stepRow,
  structureFlags,
  structureReceipt,
  subtreeEnd,
  visibleRows,
} from './logic'

// 一份 1~4 级的小标题树：
//   1 第一章 ─ 2 1.1 ─ 3 1.1.1 ─ 4 1.1.1.1
//   5 第二章 ─ 6 2.1
const tree = [
  { id: 1, level: 1 }, { id: 2, level: 2 }, { id: 3, level: 3 }, { id: 4, level: 4 },
  { id: 5, level: 1 }, { id: 6, level: 2 },
]
const ids = (rows: { node: { id: number } }[]) => rows.map((r) => r.node.id)

describe('标题树的父子与祖先', () => {
  it('hasChildren：紧随其后出现更深一级即为父节点', () => {
    expect(hasChildren(tree, 1)).toBe(true)
    expect(hasChildren(tree, 2)).toBe(true)
    expect(hasChildren(tree, 3)).toBe(true)
    expect(hasChildren(tree, 4)).toBe(false)
    expect(hasChildren(tree, 5)).toBe(true)
    expect(hasChildren(tree, 6)).toBe(false)
    expect(hasChildren(tree, 99)).toBe(false)
  })

  it('parentOf：往前找最近的更高级标题；一级标题与未知 id 为 null', () => {
    expect(parentOf(tree, 2)).toBe(1)
    expect(parentOf(tree, 4)).toBe(3)
    expect(parentOf(tree, 6)).toBe(5)
    expect(parentOf(tree, 1)).toBeNull()
    expect(parentOf(tree, 99)).toBeNull()
  })

  it('ancestorsOf：自近及远，跳级的文档也能找到链', () => {
    expect(ancestorsOf(tree, 4)).toEqual([3, 2, 1])
    expect(ancestorsOf(tree, 6)).toEqual([5])
    expect(ancestorsOf(tree, 1)).toEqual([])
  })
})

describe('折叠与可见行', () => {
  it('初始折叠：章展开到节，2 级及更深的有子节者收起', () => {
    expect([...initialCollapsed(tree)].sort()).toEqual([2, 3])
    // 首屏：第一章、1.1、第二章、2.1
    expect(ids(visibleRows(tree, initialCollapsed(tree)))).toEqual([1, 2, 5, 6])
  })

  it('折叠一级标题隐藏它的全部后代，直到下一个同级或更高级标题', () => {
    expect(ids(visibleRows(tree, new Set([1])))).toEqual([1, 5, 6])
  })

  it('缩进层数 = level - 1', () => {
    const rows = visibleRows(tree, new Set())
    expect(rows.map((r) => r.depth)).toEqual([0, 1, 2, 3, 0, 1])
  })

  it('expandAncestors 展开目标节点的全部祖先且不改动原折叠集', () => {
    const collapsed = new Set([1, 2, 3])
    const next = expandAncestors(collapsed, tree, 4)
    expect([...next]).toEqual([])
    expect([...collapsed]).toEqual([1, 2, 3])
    expect(ids(visibleRows(tree, next))).toEqual([1, 2, 3, 4, 5, 6])
  })

  it('键盘上下移：在可见行里走，到头停住', () => {
    const rows = visibleRows(tree, new Set([2]))
    expect(rows.map((r) => r.node.id)).toEqual([1, 2, 5, 6])
    expect(stepRow(rows, null, 1)).toBe(1)
    expect(stepRow(rows, 2, 1)).toBe(5)
    expect(stepRow(rows, 6, 1)).toBe(6)
    expect(stepRow(rows, 1, -1)).toBe(1)
    expect(stepRow([], 1, 1)).toBeNull()
  })
})

describe('文案', () => {
  it('formatChars：不足一万显示字数，过万折成万字', () => {
    expect(formatChars(0)).toBe('0 字')
    expect(formatChars(9999)).toBe('9999 字')
    expect(formatChars(12000)).toBe('1.2 万字')
    expect(formatChars(285000)).toBe('28.5 万字')
  })

  it('indexLabel：索引完成或没有正文不显示，未完成提示进度', () => {
    expect(indexLabel(12, 12)).toBe('')
    expect(indexLabel(0, 0)).toBe('')
    expect(indexLabel(0, 7)).toBe('语义索引待建立')
    expect(indexLabel(3, 7)).toBe('语义索引 3/7')
  })

  it('searchSummary：模式与备注拼成一句；降级说明原样带出', () => {
    expect(searchSummary('hybrid', '已本地精排')).toBe('全文 + 语义 · 已本地精排')
    expect(searchSummary('title', '')).toBe('标题树')
    expect(searchSummary('keyword', '本地 embedding 不可用，本次只按全文路检索')).toBe(
      '全文 · 本地 embedding 不可用，本次只按全文路检索',
    )
  })
})

describe('命中摘要的【】切分', () => {
  it('括号内是命中词，括号外是普通文本；相邻命中连成一段', () => {
    expect(splitSnippet('基坑【边坡失稳】、扬尘为主要【危险源】。')).toEqual([
      { text: '基坑', hit: false },
      { text: '边坡失稳', hit: true },
      { text: '、扬尘为主要', hit: false },
      { text: '危险源', hit: true },
      { text: '。', hit: false },
    ])
  })

  it('无标记与空串', () => {
    expect(splitSnippet('纯文本')).toEqual([{ text: '纯文本', hit: false }])
    expect(splitSnippet('')).toEqual([])
  })
})

// ── 2026-11-28 索引与编辑优化批 ──────────────────────────────────────

// 带标题的小标题树：
//   1 第1章 总体(30) ─ 2 1.1 部署(20) ─ 3 1.1.1 细则(15) ─ 4 1.1.2 补充(10)
//                    └ 5 1.2 安全(25)
//   6 第2章 保障(12) ─ 7 2.1 质量(8)
const titled = [
  { id: 1, parentId: 0, level: 1, seq: 0, title: '第1章 总体', titlePath: '第1章 总体', chars: 30 },
  { id: 2, parentId: 1, level: 2, seq: 1, title: '1.1 部署', titlePath: '第1章 总体 › 1.1 部署', chars: 20 },
  { id: 3, parentId: 2, level: 3, seq: 2, title: '1.1.1 细则', titlePath: '第1章 总体 › 1.1 部署 › 1.1.1 细则', chars: 15 },
  { id: 4, parentId: 2, level: 3, seq: 3, title: '1.1.2 补充', titlePath: '第1章 总体 › 1.1 部署 › 1.1.2 补充', chars: 10 },
  { id: 5, parentId: 1, level: 2, seq: 4, title: '1.2 安全', titlePath: '第1章 总体 › 1.2 安全', chars: 25 },
  { id: 6, parentId: 0, level: 1, seq: 5, title: '第2章 保障', titlePath: '第2章 保障', chars: 12 },
  { id: 7, parentId: 6, level: 2, seq: 6, title: '2.1 质量', titlePath: '第2章 保障 › 2.1 质量', chars: 8 },
]

describe('检索词高亮与标题树筛选', () => {
  it('splitByTerms：按词切分、大小写不敏感、重叠的短词不重复标', () => {
    expect(splitByTerms('边坡失稳的防控', ['失稳'])).toEqual([
      { text: '边坡', hit: false }, { text: '失稳', hit: true }, { text: '的防控', hit: false },
    ])
    expect(splitByTerms('执行 HJ 25.1 标准', ['hj'])).toEqual([
      { text: '执行 ', hit: false }, { text: 'HJ', hit: true }, { text: ' 25.1 标准', hit: false },
    ])
    expect(splitByTerms('降水井点', ['降水', '降水井'])).toEqual([
      { text: '降水井', hit: true }, { text: '点', hit: false },
    ])
    expect(splitByTerms('无命中', ['监测'])).toEqual([{ text: '无命中', hit: false }])
    expect(splitByTerms('', [])).toEqual([])
  })

  it('filterTree：命中节点连同全部祖先留下；多个词全含才算；空关键词不筛', () => {
    const keep = filterTree(titled, '细则')
    expect([...keep].sort()).toEqual([1, 2, 3])
    expect([...filterTree(titled, '总体 安全')].sort()).toEqual([1, 5]) // 路径里带父节标题：子节靠路径命中
    expect(filterTree(titled, '总体 没有')).toEqual(new Set())
    expect(filterTree(titled, '   ')).toEqual(new Set())
    expect([...filterTree(titled, '第2章 质量')].sort()).toEqual([6, 7])
  })
})

describe('编号与层级核对', () => {
  it('numberDepth：点分数字数层数、第X章是 1 级、其余判断不了', () => {
    expect(numberDepth('1.4.7.2 备选修复技术介绍')).toBe(4)
    expect(numberDepth('3 质量保障方案')).toBe(0) // 纯数字分不清是章号还是名称的一部分
    expect(numberDepth('第一章 施工降水')).toBe(1)
    expect(numberDepth('（一）降水措施')).toBe(0)
    expect(numberDepth('一、总体部署')).toBe(0)
    expect(numberDepth('2026 年度监测计划')).toBe(0)
  })

  it('numberMismatch：层级对上不算，对不上才算；提示里带上两种层数', () => {
    expect(numberMismatch('1.1.2 补充', 3)).toBe(false)
    expect(numberMismatch('1.1.2 补充', 2)).toBe(true) // 升级后编号没改
    expect(numberMismatch('一、总体部署', 3)).toBe(false)
    expect(numberMismatch('第二章 安全', 2)).toBe(true)
    expect(mismatchHint('1.1.2 补充', 2)).toBe('编号与层级不符：标题里的编号是 3 级写法，现在在第 2 层。可以改标题里的编号，或用「调整结构」改回去')
  })
})

describe('结构操作的可用性', () => {
  it('subtreeEnd / prevSiblingIndex（参数是下标）', () => {
    expect(subtreeEnd(titled, 0)).toBe(5) // 第1章 连同 4 个子孙
    expect(subtreeEnd(titled, 1)).toBe(4) // 1.1 部署 连同 2 个子节
    expect(subtreeEnd(titled, 4)).toBe(5) // 1.2 安全 没有子节
    expect(subtreeEnd(titled, 5)).toBe(7)
    expect(prevSiblingIndex(titled, 0)).toBe(-1)
    expect(prevSiblingIndex(titled, 1)).toBe(-1) // 本层第一个
    expect(prevSiblingIndex(titled, 4)).toBe(1)
    expect(prevSiblingIndex(titled, 6)).toBe(-1)
  })

  it('structureFlags：1 级标题不能升、第一个子节不能降也不能并入上一级', () => {
    expect(structureFlags(titled, 1)).toMatchObject({
      canPromote: false, canDemote: false, canMoveUp: false, canMoveDown: true,
      canMergeParent: false, canMergePrev: false, canAddChild: true, childCount: 2, subtreeCount: 5, subtreeChars: 100,
    })
    // 1.1 部署：有子节、是第一个子节、后面有同级小节
    expect(structureFlags(titled, 2)).toMatchObject({
      canPromote: true, canDemote: false, canMoveUp: false, canMoveDown: true,
      canMergeParent: true, canMergePrev: false, canAddChild: true, childCount: 2, subtreeCount: 3, subtreeChars: 45,
    })
    // 1.2 安全：不是第一个子节——并入父节会把正文排到前面几节之前，不给
    expect(structureFlags(titled, 5)).toMatchObject({
      canPromote: true, canDemote: true, canMoveUp: true, canMoveDown: false,
      canMergeParent: false, canMergePrev: true, canAddChild: true, childCount: 0,
    })
    expect(structureFlags(titled, 999)).toMatchObject({ canPromote: false, canAddChild: false, subtreeCount: 0 })
  })

  it('structureReceipt：升降级带上编号提醒，删除带上子节数', () => {
    expect(structureReceipt('promote', { affected: 3, created: 0, removed: 0 })).toBe('已升级，2 个小节的标题路径已同步。标题里的编号没有改，与新层级不符时标题旁会有提示')
    expect(structureReceipt('demote', { affected: 1, created: 0, removed: 0 })).toContain('归到上一同级小节名下')
    expect(structureReceipt('moveUp', { affected: 4, created: 0, removed: 0 })).toBe('已上移。标题里的编号没有改，与新层级不符时标题旁会有提示')
    expect(structureReceipt('remove', { affected: 0, created: 0, removed: 6 })).toBe('已删除本节及 5 个子节')
    expect(structureReceipt('addChild', { affected: 0, created: 1, removed: 0 })).toBe('已新增小节，可以开始写正文')
  })
})

describe('拆分预览与字数变化', () => {
  const body = '安全正文。\n\n第二段讲围挡。\n第三段讲警示。'
  it('splitPreview：光标所在行及之后切出；切点按码点数交给后端', () => {
    expect(splitPreview(body, 9)).toEqual({
      offset: 7, head: '安全正文。', tail: '第二段讲围挡。\n第三段讲警示。', suggest: '第二段讲围挡。',
    })
    // 光标在行首：该行归新小节；在末尾：最后一行归新小节
    expect(splitPreview(body, 7)?.head).toBe('安全正文。')
    expect(splitPreview(body, body.length)?.tail).toBe('第三段讲警示。')
    // 正文里混着 BMP 外的字符（emoji 占两个 UTF-16 单位、一个码点）时切点仍按码点数；
    // 光标停在第一行行尾（换行符之前）算在第一行上。
    expect(splitPreview('😀😀。\n后文。', 5)).toBeNull()
    expect(splitPreview('😀😀。\n后文。', 6)?.offset).toBe(4)
    // 新小节标题的底稿：拆出去那一段的第一行，去掉 Markdown 记号
    expect(splitPreview('前言。\n## 小节标题\n正文。', 5)?.suggest).toBe('小节标题')
  })
  it('光标在第一行、或之后没有正文：不能拆', () => {
    expect(splitPreview(body, 0)).toBeNull()
    expect(splitPreview(body, 2)).toBeNull()
    expect(splitPreview('正文。\n\n', 5)).toBeNull() // 末尾只有空行
    expect(splitPreview('正文。\n\n', 3)).toBeNull()
  })
  it('changeLabel / sourceLabel / safeFileName', () => {
    expect(changeLabel('abc', 'abc')).toBe('')
    expect(changeLabel('abc', 'ab')).toBe('−1 字')
    expect(changeLabel('abc', 'abcd')).toBe('+1 字')
    expect(changeLabel('abc', 'abd')).toBe('字数没变，内容有改动')
    expect(sourceLabel('both')).toBe('全文+语义')
    expect(sourceLabel('semantic')).toBe('语义')
    expect(sourceLabel('keyword')).toBe('全文')
    expect(sourceLabel(undefined)).toBe('')
    expect(safeFileName('包4技术标V1（南岸/弹子石：F7-19地块）')).toBe('包4技术标V1（南岸 弹子石：F7-19地块）')
    expect(safeFileName('  ')).toBe('方案')
  })
})

describe('索引体检结论', () => {
  it('没问题不列条目；有条目算坏；向量没补齐不算坏', () => {
    const clean = { vecDone: 10, vecTotal: 12, orphanFts: 0, missingFts: 0, staleFts: 0, missingChunks: 0, staleChunks: 0 }
    expect(indexSummary(clean)).toEqual({ broken: false, lines: [] })
    expect(indexSummary({ ...clean, orphanFts: 3, missingChunks: 1 })).toEqual({
      broken: true,
      lines: ['3 条全文索引对应的小节已不存在', '1 个小节缺语义块'],
    })
  })
})
