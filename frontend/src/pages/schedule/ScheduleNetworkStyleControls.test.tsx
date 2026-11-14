import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ScheduleNetworkStyleControls } from './ScheduleNetworkStyleControls'

describe('ScheduleNetworkStyleControls（W4 图面参数）', () => {
  it('默认保持旧图面：节点时间/工期开，六参数/起止时间关', () => {
    render(<ScheduleNetworkStyleControls networkView onChange={() => {}} />)
    expect(screen.getByTestId('sched-network-toggle-showDotTime').getAttribute('aria-checked')).toBe('true')
    expect(screen.getByTestId('sched-network-toggle-showDuration').getAttribute('aria-checked')).toBe('true')
    expect(screen.getByTestId('sched-network-toggle-showSixPara').getAttribute('aria-checked')).toBe('false')
    expect(screen.getByTestId('sched-network-toggle-showStartStopTime').getAttribute('aria-checked')).toBe('false')
  })

  it('四开关只提交对应 patch，不清空其他图面参数', () => {
    const onChange = vi.fn()
    render(
      <ScheduleNetworkStyleControls
        style={{ dotStyle: 'rect', nodeRadius: 20, showDuration: false }}
        networkView
        onChange={onChange}
      />,
    )
    fireEvent.click(screen.getByTestId('sched-network-toggle-showStartStopTime'))
    expect(onChange).toHaveBeenLastCalledWith({ showStartStopTime: true })
    fireEvent.click(screen.getByTestId('sched-network-toggle-showDuration'))
    expect(onChange).toHaveBeenLastCalledWith({ showDuration: true })
    expect(screen.getByTestId('sched-network-style-dot').textContent).toContain('矩形事件')
  })

  it('非网络页显示预配置提示，控件仍可操作', () => {
    const onChange = vi.fn()
    render(<ScheduleNetworkStyleControls networkView={false} onChange={onChange} />)
    expect(screen.getByTestId('sched-network-style-controls').textContent).toContain('切换视图后生效')
    fireEvent.click(screen.getByTestId('sched-network-toggle-showSixPara'))
    expect(onChange).toHaveBeenCalledWith({ showSixPara: true })
  })

  it('线条宽度与几何参数同入口，缺省 1.5px 回显', () => {
    render(<ScheduleNetworkStyleControls style={{ lineWidth: 2 }} networkView onChange={() => {}} />)
    expect(screen.getByTestId('sched-network-style-controls').textContent).toContain('线条宽度')
    expect((screen.getByTestId('sched-network-style-line') as HTMLInputElement).value).toBe('2.0')
  })

  it('布图参数（密度/层高/错层/隐藏非重要线）同入口且缺省不写文件', () => {
    const onChange = vi.fn()
    render(<ScheduleNetworkStyleControls style={{ density: 'loose', stagger: true }} networkView onChange={onChange} />)
    expect(screen.getByTestId('sched-network-style-controls').textContent).toContain('布图密度')
    expect(screen.getByTestId('sched-network-style-density').textContent).toContain('疏松')
    fireEvent.click(screen.getByTestId('sched-network-style-stagger'))
    expect(onChange).toHaveBeenLastCalledWith({ stagger: false })
    fireEvent.click(screen.getByTestId('sched-network-style-hidelink'))
    expect(onChange).toHaveBeenLastCalledWith({ hideUnimportantLink: true })
    expect((screen.getByTestId('sched-network-style-layer1') as HTMLInputElement).value).toBe('112')
  })
})
