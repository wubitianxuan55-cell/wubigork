/**
 * extractFence.test.ts — FE6-05 样板单源钉：ChatMarkdown / genuiAdapter /
 * Markdown 三条渲染缝的 code 组件共用 extractFence（language- 提取 + 去尾
 * 换行 + 块级判定）。本钉冻结该样板的逐字段行为：
 * - language- 提取含带连字符语言（dsh-ui）——[\w-] 字符类的连字符不可丢；
 * - 仅去「一个」尾换行（fenced 块 children 的固定尾巴），其余换行保留；
 * - isBlock = 提取到 language-* 或正文含换行（与三消费方原逐字行为一致）。
 */
import { describe, expect, it } from 'vitest'
import { extractFence } from './markdownFence'

describe('extractFence code 样板单源（FE6-05）', () => {
  it('language- 提取：普通语言与带连字符语言', () => {
    expect(extractFence('language-genui', 'x').lang).toBe('genui')
    expect(extractFence('language-dsh-ui', 'x').lang).toBe('dsh-ui')
    expect(extractFence('language-ts highlight', 'x').lang).toBe('ts')
  })

  it('无 language- 前缀：lang undefined', () => {
    expect(extractFence(undefined, 'x').lang).toBeUndefined()
    expect(extractFence('', 'x').lang).toBeUndefined()
    expect(extractFence('hljs js', 'x').lang).toBeUndefined()
  })

  it('children 去尾换行：只去一个 \n，其余保留；null/undefined → 空串', () => {
    expect(extractFence('language-ts', 'a\nb\n').text).toBe('a\nb')
    expect(extractFence('language-ts', 'a\nb').text).toBe('a\nb')
    expect(extractFence('language-ts', '\n').text).toBe('')
    expect(extractFence('language-ts', null).text).toBe('')
    expect(extractFence('language-ts', undefined).text).toBe('')
  })

  it('isBlock：有 language-* 即块级；无语言但正文含换行也块级；否则行内', () => {
    expect(extractFence('language-genui', 'x').isBlock).toBe(true)
    expect(extractFence(undefined, 'a\nb').isBlock).toBe(true)
    expect(extractFence(undefined, 'x').isBlock).toBe(false)
  })

  it('返回三元组同批给出（消费方解构用法）', () => {
    const r = extractFence('language-dsh-ui', '{...}\n')
    expect(r).toEqual({ text: '{...}', lang: 'dsh-ui', isBlock: true })
  })
})
