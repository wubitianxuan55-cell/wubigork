// v4.15 消息级「由谁回答 / 为何 / 花了多少」回显 —— AnsweredByLine 组件测试：
//  - 有费用（cost_cny > 0）→ 渲染「约 ¥」段
//  - 费用 <= 0（本地引擎/未知）→ 隐藏费用段
//  - 有速度（tokens_per_second > 0）→ 渲染「tok/s」段；缺省/0/负值 → 隐藏
//  - 未知 source → 原样兜底展示
//  - feature/global/fallback 标签文案映射
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import AnsweredByLine, { answeredBySourceLabel, answeredBySpeedText } from './AnsweredByLine'
import type { AnsweredByInfo } from './AnsweredByLine'

function renderLine(info: Partial<AnsweredByInfo> & Pick<AnsweredByInfo, 'engine' | 'model'>) {
  return render(
    <AnsweredByLine
      info={{ source: 'feature', cost_cny: 0, ...info } as AnsweredByInfo}
    />,
  )
}

describe('AnsweredByLine 回显（v4.15）', () => {
  it('有费用：渲染「由 {engine}/{model} 回答 · {来源} · 约 ¥{费用}」', () => {
    renderLine({ engine: 'deepseek', model: 'deepseek-v4-flash', source: 'feature', cost_cny: 0.0123 })
    expect(screen.getByText('由 deepseek/deepseek-v4-flash 回答 · 功能绑定 · 约 ¥0.01')).toBeTruthy()
  })

  it('费用 <= 0（本地引擎/未知）：隐藏费用段，仅剩来源段', () => {
    renderLine({ engine: 'herdsman', model: 'qwen2.5-7b', source: 'fallback', cost_cny: 0 })
    expect(screen.getByText('由 herdsman/qwen2.5-7b 回答 · 兜底')).toBeTruthy()
    expect(screen.queryByText(/约 ¥/)).toBeNull()
  })

  it('有速度：费用与速度段并存（费用 · 速度）', () => {
    renderLine({ engine: 'deepseek', model: 'v4', source: 'global', cost_cny: 0.05, tokens_per_second: 42.3 })
    expect(screen.getByText('由 deepseek/v4 回答 · 全局路由 · 约 ¥0.05 · 42.3 tok/s')).toBeTruthy()
  })

  it('速度缺省/0/负值/非有限数：隐藏速度段（不虚报）', () => {
    expect(answeredBySpeedText(undefined)).toBeNull()
    expect(answeredBySpeedText(0)).toBeNull()
    expect(answeredBySpeedText(-3.2)).toBeNull()
    expect(answeredBySpeedText(Number.NaN)).toBeNull()
    renderLine({ engine: 'h', model: 'm', tokens_per_second: 0 })
    expect(screen.queryByText(/tok\/s/)).toBeNull()
  })

  it('未知 source：原样兜底展示', () => {
    renderLine({ engine: 'x', model: 'y', source: 'custom-source' })
    expect(screen.getByText('由 x/y 回答 · custom-source')).toBeTruthy()
  })

  it('source 标签映射：feature→功能绑定 / global→全局路由 / fallback→兜底', () => {
    expect(answeredBySourceLabel('feature')).toBe('功能绑定')
    expect(answeredBySourceLabel('global')).toBe('全局路由')
    expect(answeredBySourceLabel('fallback')).toBe('兜底')
    expect(answeredBySourceLabel('')).toBe('未知')
  })
})
