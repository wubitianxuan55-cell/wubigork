import { beforeEach, describe, expect, it } from 'vitest'
import { normalizeProject, useScheduleStore } from './store'

describe('store 图面参数（W4）', () => {
  beforeEach(() => {
    useScheduleStore.setState({
      project: normalizeProject({
        name: '图面样板',
        startDate: '2026-09-07',
        tasks: [{ id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 }],
        links: [],
        networkStyle: { mode: 'timescaled', dotStyle: 'rect', nodeRadius: 20 },
      }),
      past: [],
      future: [],
    })
  })

  it('patch 与既有样式合并，未知/越界值由 normalizeNetworkStyle 丢弃', () => {
    useScheduleStore.getState().setNetworkStyle({
      showDuration: false,
      showStartStopTime: true,
      nodeRadius: 400,
      headHeight: 12,
    })
    expect(useScheduleStore.getState().project.networkStyle).toEqual({
      mode: 'timescaled',
      dotStyle: 'rect',
      headHeight: 12,
      showDuration: false,
      showStartStopTime: true,
    })
  })

  it('一次 patch 一条历史；undo 恢复原图面参数', () => {
    useScheduleStore.getState().setNetworkStyle({ showSixPara: false })
    expect(useScheduleStore.getState().past).toHaveLength(1)
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.networkStyle).toEqual({
      mode: 'timescaled',
      dotStyle: 'rect',
      nodeRadius: 20,
    })
  })
})
