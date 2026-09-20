// InstructionEditModal.tsx — 指令编辑弹窗（绘梦阶段一刀 C，规格
// 进度计划/gaea-instruct-edit-20260917.md）：原图 + 人话指令 → 局部语义编辑
// （OpenAI 兼容 /images/edits；云端先行）。与既有「改图」（=把结果填回图生图
// 整幅重绘）互补。对照区原图|新图并排；「用到画布」把编辑结果并入 results/
// history（后端已落盘+登记台账，走既有保存/溯源链）。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useState } from 'react'
import { Alert, Button, Input, Modal, Spin, Typography, message } from 'antd'
import { generateMedia } from '../../api/image'
import type { GenResult } from './types'


export default function InstructionEditModal({ open, source, onClose, onApply }: {
  open: boolean
  /** 编辑源图（结果卡点「指令编辑」的那张）。 */
  source: GenResult | null
  onClose: () => void
  /** 「用到画布」：编辑结果并入画布与历史（页面级镜像 queue 成功路径）。 */
  onApply: (r: GenResult) => void
}) {
  const [instruction, setInstruction] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [edited, setEdited] = useState<GenResult | null>(null)

  const run = async () => {
    if (!source || !instruction.trim()) {
      message.warning('请输入编辑指令（如：把外套改成红色）')
      return
    }
    setBusy(true)
    setError('')
    try {
      const res = await generateMedia({
        prompt: instruction.trim(),
        negative: '',
        size: source.size || '',
        model: source.model || '',
        seed: 0,
        lora: '',
        count: 1,
        mode: 'edit',
        initImage: source.image,
      })
      if (res.error) {
        setError(res.error)
        return
      }
      const first = res.results?.[0]
      if (!first) {
        setError('编辑返回空结果')
        return
      }
      setEdited({ ...first, prompt: `【编辑】${instruction.trim()}` })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const reset = () => {
    setInstruction('')
    setEdited(null)
    setError('')
    setBusy(false)
  }

  const imgBox: React.CSSProperties = {
    flex: 1, minWidth: 0, borderRadius: 8, overflow: 'hidden',
    background: 'rgba(0,0,0,0.04)', display: 'flex', alignItems: 'center', justifyContent: 'center',
  }

  return (
    <Modal
      open={open} title="指令编辑（原图 + 一句话指令 → 语义改图）" width={720} destroyOnHidden
      onCancel={() => { reset(); onClose() }}
      footer={[
        <Button key="run" size="small" type="primary" loading={busy} data-testid="instruct-edit-run"
          onClick={() => void run()}>生成编辑</Button>,
        edited && (
          <Button key="apply" size="small" data-testid="instruct-edit-apply"
            onClick={() => { if (edited) { onApply(edited); reset(); onClose() } }}>用到画布</Button>
        ),
        <Button key="close" size="small" onClick={() => { reset(); onClose() }}>关闭</Button>,
      ].filter(Boolean)}
    >
      {!source ? (
        <Typography.Text type="secondary">未选择源图</Typography.Text>
      ) : (
        <div data-testid="instruct-edit-body" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {/* 指令输入 */}
          <div>
            <div style={{ ...softTextStyle, marginBottom: 4 }}>编辑指令（描述要改什么，其余保持原样）</div>
            <Input.TextArea
              data-testid="instruct-edit-input" rows={2} value={instruction} maxLength={600}
              placeholder="例如：把外套改成红色；移除背景里的路人；让她看向镜头"
              onChange={e => setInstruction(e.target.value)} />
          </div>
          {/* 对照区：原图 | 新图 */}
          <div style={{ display: 'flex', gap: 10, alignItems: 'stretch', minHeight: 220 }}>
            <div style={imgBox}>
              <img src={source.image} alt="原图" style={{ maxWidth: '100%', maxHeight: 300, objectFit: 'contain' }} />
            </div>
            <div style={{ ...imgBox, border: edited ? '1px dashed rgba(0,0,0,0.18)' : undefined }}>
              {busy ? (
                <div style={{ padding: 24, textAlign: 'center' }}><Spin /><div style={{ ...softTextStyle, marginTop: 6 }}>正在按指令编辑…</div></div>
              ) : edited ? (
                <img src={edited.image} alt="编辑结果" data-testid="instruct-edit-result"
                  style={{ maxWidth: '100%', maxHeight: 300, objectFit: 'contain' }} />
              ) : (
                <div style={{ ...softTextStyle, padding: 24, textAlign: 'center' }}>编辑结果会出现在这里</div>
              )}
            </div>
          </div>
          {edited && (
            <div style={softTextStyle} data-testid="instruct-edit-meta">
              模型 {edited.model || '—'} · {edited.time ?? '—'}s{edited.size ? ` · ${edited.size}` : ''}
            </div>
          )}
          {error && <Alert type="error" showIcon data-testid="instruct-edit-error" message={error} />}
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            指令编辑走云端 OpenAI 兼容引擎的 /images/edits（如 Qwen-Image-Edit 系）；本地 ComfyUI 与 GLM 档暂不支持（会如实报错）。
          </Typography.Text>
        </div>
      )}
    </Modal>
  )
}
