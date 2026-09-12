// sin/storyText.test.ts — 故事正文解析（插图标记 / 分段 / extra 映射）纯函数测试。
// 契约重点：cue 键 = 出现次序（0 起），必须与 Go 侧 sinCueKey 同规则，
// 否则导出与重开时插图映射对不上。
import { describe, expect, it } from 'vitest'
import {
  SIN_CUE_CLOSE, SIN_CUE_OPEN, parseIllustrations, parseReasoning,
  parseStorySegments, parseTools, pendingIllustrations, sinCueKey,
  stripCuesForDisplay, suggestStoryTitle, toToolViews,
} from './storyText'

describe('parseStorySegments', () => {
  it('按出现次序编号插图段，文本段与插图段交替', () => {
    const content = `雨落在窗上。\n${SIN_CUE_OPEN}女人站在灯下${SIN_CUE_CLOSE}\n她回头。\n${SIN_CUE_OPEN}近景${SIN_CUE_CLOSE}`
    const segs = parseStorySegments(content)
    expect(segs.map((s) => s.kind)).toEqual(['text', 'illustration', 'text', 'illustration'])
    expect(segs[1]).toEqual({ kind: 'illustration', cueKey: '0', prompt: '女人站在灯下' })
    expect(segs[3]).toEqual({ kind: 'illustration', cueKey: '1', prompt: '近景' })
    expect((segs[0] as { text: string }).text).toContain('雨落在窗上。')
  })

  it('未闭合标记按普通文本处理（流式中间态不冒插图位）', () => {
    const segs = parseStorySegments(`开头。\n${SIN_CUE_OPEN}画面描述还没写完`)
    expect(segs).toHaveLength(1)
    expect(segs[0].kind).toBe('text')
  })

  it('纯文本正文只有一段文本（无标记时不产生插图段）', () => {
    expect(parseStorySegments('只有正文。')).toEqual([{ kind: 'text', text: '只有正文。' }])
    expect(parseStorySegments('')).toEqual([])
  })

  it('cue 键规则稳定：出现次序即键（sinCueKey 同源）', () => {
    expect(sinCueKey(0)).toBe('0')
    expect(sinCueKey(7)).toBe('7')
  })
})

describe('extra 解析', () => {
  it('插图映射：JSON 字符串与已解析对象都认，坏 JSON 按空映射', () => {
    const json = JSON.stringify({ illustrations: { '0': 'C:/a.png' } })
    expect(parseIllustrations(json)).toEqual({ '0': 'C:/a.png' })
    expect(parseIllustrations({ illustrations: { '1': 'C:/b.png' } })).toEqual({ '1': 'C:/b.png' })
    expect(parseIllustrations('{坏 JSON')).toEqual({})
    expect(parseIllustrations('')).toEqual({})
    // 非字符串值丢弃（不把对象当路径）
    expect(parseIllustrations({ illustrations: { '0': 42 } })).toEqual({})
  })

  it('reasoning 取字符串字段，缺失/坏 JSON 返回空串', () => {
    expect(parseReasoning('{"reasoning":"想了想"}')).toBe('想了想')
    expect(parseReasoning('{}')).toBe('')
    expect(parseReasoning('not json')).toBe('')
  })
})

describe('pendingIllustrations', () => {
  it('只列出还没有产物且提示词非空的插图段', () => {
    const segs = parseStorySegments(
      `${SIN_CUE_OPEN}第一张${SIN_CUE_CLOSE}${SIN_CUE_OPEN}第二张${SIN_CUE_CLOSE}${SIN_CUE_OPEN}${SIN_CUE_CLOSE}`,
    )
    expect(pendingIllustrations(segs, { '0': 'C:/a.png' })).toEqual([{ cueKey: '1', prompt: '第二张' }])
  })
})

describe('展示辅助', () => {
  it('stripCuesForDisplay 把标记折成「（插图：…）」提示', () => {
    expect(stripCuesForDisplay(`前${SIN_CUE_OPEN}描述${SIN_CUE_CLOSE}后`)).toBe('前（插图：描述）后')
  })

  it('suggestStoryTitle 取首条指令前 16 字，空则「新故事」', () => {
    expect(suggestStoryTitle('写一个雨夜电车上的开场，主角是女记者和一个陌生男人')).toBe('写一个雨夜电车上的开场，主角是女')
    expect(suggestStoryTitle('   ')).toBe('新故事')
  })
})

describe('parseTools / toToolViews（工具轨迹）', () => {
  it('解析落库轨迹：JSON 字符串与已解析对象都认，键名按后端 snake_case', () => {
    const extra = JSON.stringify({
      reasoning: 'r',
      tools: [{
        id: 'call_1', name: 'web_search', args: '{"query":"唐末长安"}',
        output: '结果', error: '', elapsed_ms: 1240, read_only: true,
      }],
    })
    const [t] = parseTools(extra)
    expect(t).toEqual({
      id: 'call_1', name: 'web_search', args: '{"query":"唐末长安"}',
      output: '结果', error: '', elapsed_ms: 1240, read_only: true,
    })
    expect(parseTools(JSON.parse(extra))).toHaveLength(1)
  })

  it('容错：坏 JSON / 缺字段 / 畸形条目一律跳过，绝不抛', () => {
    expect(parseTools('not json')).toEqual([])
    expect(parseTools('')).toEqual([])
    expect(parseTools(null)).toEqual([])
    expect(parseTools('{"tools":"nope"}')).toEqual([])
    // 无名条目渲染不出信息 → 丢弃；其余字段缺失按默认值补齐（不编造）。
    expect(parseTools('{"tools":[{"args":"{}"},{"name":"sin_notes"}]}')).toEqual([
      { id: '', name: 'sin_notes', args: '', output: '', error: '', elapsed_ms: 0, read_only: false },
    ])
  })

  it('toToolViews：有 error 即失败，否则完成（历史消息没有运行态）', () => {
    const views = toToolViews([
      { id: 'a', name: 'web_search', error: '' },
      { id: 'b', name: 'sin_notes', error: '工具执行失败：x' },
    ])
    expect(views.map((v) => v.status)).toEqual(['done', 'failed'])
  })
})
