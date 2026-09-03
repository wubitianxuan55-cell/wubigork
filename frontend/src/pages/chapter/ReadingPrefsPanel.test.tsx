// ReadingPrefsPanel.test.tsx — 阅读排版面板（自 ChapterPage 拆分第四批）：
// 纯受控展示组件，锁各排布档位点击 → onChange 增量载荷正确（含钳制），与拆分前行为一致。
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import ReadingPrefsPanel from './ReadingPrefsPanel'
import { DEFAULT_READING_SETTINGS, type ReadingSettings } from '../../utils/readingSettings'

const prefs = (over: Partial<ReadingSettings>): ReadingSettings => ({
  ...DEFAULT_READING_SETTINGS,
  ...over,
})

describe('ReadingPrefsPanel（阅读排版面板）', () => {
  it('渲染全部六行排布项', () => {
    render(<ReadingPrefsPanel prefs={prefs({})} onChange={() => {}} />)
    for (const label of ['字号', '行距', '版宽', '主题', '亮度', '滚屏']) {
      expect(screen.getByText(label)).toBeTruthy()
    }
  })

  it('字号 A+/A−：onChange 载荷为钳制后的增量', () => {
    const onChange = vi.fn()
    render(<ReadingPrefsPanel prefs={prefs({ fontSize: 17 })} onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: '增大字号' }))
    expect(onChange).toHaveBeenLastCalledWith({ fontSize: 18 })
    fireEvent.click(screen.getByRole('button', { name: '减小字号' }))
    expect(onChange).toHaveBeenLastCalledWith({ fontSize: 16 })
  })

  it('字号边界：已达上下限时对应按钮禁用（钳制不在 onChange 后）', () => {
    render(<ReadingPrefsPanel prefs={prefs({ fontSize: 24 })} onChange={() => {}} />)
    expect((screen.getByRole('button', { name: '增大字号' }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: '减小字号' }) as HTMLButtonElement).disabled).toBe(false)
  })

  it('行距/版宽/主题/滚屏：onChange 载荷与拆分前各档位一致', () => {
    const onChange = vi.fn()
    render(<ReadingPrefsPanel prefs={prefs({})} onChange={onChange} />)
    fireEvent.click(screen.getByText('宽松'))
    expect(onChange).toHaveBeenLastCalledWith({ lineHeight: 2.3 })
    fireEvent.click(screen.getByText('窄')) // 默认即铺满（wide），点已选项不触发 onChange
    expect(onChange).toHaveBeenLastCalledWith({ column: 'narrow' })
    fireEvent.click(screen.getByText('夜间'))
    expect(onChange).toHaveBeenLastCalledWith({ theme: 'dark' })
    fireEvent.click(screen.getByText('快'))
    expect(onChange).toHaveBeenLastCalledWith({ autoScrollSpeed: 5 })
  })

  it('亮度展示值跟随 prefs', () => {
    render(<ReadingPrefsPanel prefs={prefs({ brightness: 90 })} onChange={() => {}} />)
    expect(screen.getByText('90%')).toBeTruthy()
  })
})
