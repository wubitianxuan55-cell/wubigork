import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'
import { Modal } from 'antd'

/**
 * 复现 WebView2 冻结 rAF 的场景：
 * antd Modal 关闭时由 rc-motion 的 rAF 步进推进 leave 动画，rAF 被冻结后
 * 动画永不结束，rc-dialog 的 onVisibleChanged(false) 永不回调 → 全屏
 * .ant-modal-wrap 一直留在 DOM 且 display 未置 none，拦截所有点击
 * （界面看似已关闭、实际全部点不了）。
 * 修复（与 CreateNovelModal / CharacterLibEditor 同款）：
 * destroyOnHidden + transitionName="" + maskTransitionName="" 关闭即卸载。
 */
function freezeRAF() {
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
}

function ModalHarness({ open, fixed }: { open: boolean; fixed: boolean }) {
  return (
    <Modal
      open={open}
      onCancel={() => {}}
      footer={null}
      width={680}
      {...(fixed ? { destroyOnHidden: true, transitionName: '', maskTransitionName: '' } : {})}
    >
      <div>弹窗内容</div>
    </Modal>
  )
}

describe('角色面板 Modal 关闭残留（WebView2 rAF 冻结）', () => {
  it('未修复的 Modal 在 rAF 冻结时关闭后残留全屏 wrap 拦截点击', () => {
    freezeRAF()
    try {
      const { rerender } = render(<ModalHarness open fixed={false} />)
      expect(document.querySelector('.ant-modal-wrap')).toBeTruthy()

      rerender(<ModalHarness open={false} fixed={false} />)

      const wrap = document.querySelector('.ant-modal-wrap') as HTMLElement | null
      expect(wrap).toBeTruthy()
      expect(wrap!.style.display).not.toBe('none')
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('修复后（destroyOnHidden + 禁用过渡）关闭即卸载，无 wrap 残留', () => {
    freezeRAF()
    try {
      const { rerender } = render(<ModalHarness open fixed />)
      expect(document.querySelector('.ant-modal-wrap')).toBeTruthy()

      rerender(<ModalHarness open={false} fixed />)

      expect(document.querySelector('.ant-modal-wrap')).toBeNull()
      expect(document.querySelector('.ant-modal-root')).toBeNull()
    } finally {
      vi.unstubAllGlobals()
    }
  })
})
