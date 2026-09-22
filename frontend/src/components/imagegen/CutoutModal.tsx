// CutoutModal.tsx — 抠图/透明底导出（绘梦阶段二刀 E，T3 收官，规格
// 进度计划/gaea-cutout-20260923.md）：涂选要保留的主体 → 蒙版即 alpha →
// 透明 PNG 落盘+台账登记。零模型调用。**抠图蒙版语义：涂=保留**（与编辑的
// 「涂=重绘」相反——蒙版产物同组件，语义由消费方定义）。导出产物不进画布
// results（透明图在结果流的展示语义挂观察池），成功态显示路径。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useState } from 'react'
import { Alert, Button, Modal, Typography, message } from 'antd'
import { imageCutout } from '../../api/image'
import MaskBrushEditor from './MaskBrushEditor'
import type { GenResult } from './types'

export default function CutoutModal({ open, source, onClose }: {
  open: boolean
  source: GenResult | null
  onClose: () => void
}) {
  const [mask, setMask] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [savedPath, setSavedPath] = useState('')

  const run = async () => {
    if (!source) return
    if (!mask) {
      message.warning('请先涂抹要保留的主体（其余区域将变透明）')
      return
    }
    setBusy(true)
    setError('')
    try {
      const res = await imageCutout(source.image, mask)
      if (res.error) {
        setError(res.error)
        return
      }
      setSavedPath(res.path ?? '')
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const reset = () => {
    setMask(null)
    setBusy(false)
    setError('')
    setSavedPath('')
  }

  return (
    <Modal
      open={open} title="抠图（涂选保留主体 → 透明底 PNG）" width={640} destroyOnHidden
      onCancel={() => { reset(); onClose() }}
      footer={[
        <Button key="run" size="small" type="primary" loading={busy} data-testid="cutout-run"
          onClick={() => void run()}>导出透明 PNG</Button>,
        <Button key="close" size="small" onClick={() => { reset(); onClose() }}>关闭</Button>,
      ]}
    >
      {!source ? (
        <Typography.Text type="secondary">未选择源图</Typography.Text>
      ) : savedPath ? (
        <Alert type="success" showIcon data-testid="cutout-saved"
          message="透明底 PNG 已导出"
          description={<div style={{ wordBreak: 'break-all' }}>
            <div style={softTextStyle}>{savedPath}</div>
            <div style={{ ...softTextStyle, marginTop: 4 }}>画室素材库与台账可查；回填角色库参考图可用此路径。</div>
          </div>} />
      ) : (
        <div data-testid="cutout-body" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div style={softTextStyle}>在图上涂抹<b>要保留的主体</b>（涂红区域保留原像素，其余区域转为透明）</div>
          <MaskBrushEditor src={source.image} onMaskChange={setMask} />
          {error && <Alert type="error" showIcon data-testid="cutout-error" message={error} />}
        </div>
      )}
    </Modal>
  )
}
