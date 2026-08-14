import React, { useCallback, useEffect, useState } from 'react'
import { Button, Card, Input, message, Modal, Space, Spin, Typography } from 'antd'
import { BulbOutlined } from '@ant-design/icons'
import { C } from '../../../utils/theme'

export interface Branch { title: string; pitch: string }

interface BranchWizardModalProps {
  open: boolean
  prevChapter: number
  overwriteChapter: number
  branchFromID: string
  onClose: () => void
  /** 拉取 AI 构思的分支列表（由页面层注入最新设定与前文摘要） */
  onFetchBranches: (prevChapter: number) => Promise<Branch[]>
  /** 确认生成：plotReq 由分支标题或用户输入组装 */
  onStart: (plotReq: string, overwriteChapter: number, branchFromID: string) => void
}

/**
 * 剧情方向向导弹窗（T6-7.5 从 CreatePage 拆分）：自持构思步骤/分支/选择状态，
 * 打开时自动拉取 AI 构思；「重新构思」与「生成」按钮行为与旧实现一致。
 */
const BranchWizardModal: React.FC<BranchWizardModalProps> = ({
  open, prevChapter, overwriteChapter, branchFromID, onClose, onFetchBranches, onStart,
}) => {
  const [wizStep, setWizStep] = useState<'loading' | 'branches'>('loading')
  const [branches, setBranches] = useState<Branch[]>([])
  const [selectedBranch, setSelectedBranch] = useState<number | null>(null)
  const [userInput, setUserInput] = useState('')

  const loadBranches = useCallback(async () => {
    setWizStep('loading')
    try {
      const list = await onFetchBranches(prevChapter)
      setBranches(list)
      setSelectedBranch(null)
      setUserInput('')
    } catch (err: unknown) {
      // 构思失败不静默：提示用户可手动输入剧情要求继续
      setBranches([])
      message.error(err instanceof Error ? err.message : '剧情构思失败，可手动输入剧情要求')
    } finally {
      setWizStep('branches')
    }
  }, [onFetchBranches, prevChapter])

  // 每次打开重新构思（初始即 loading → 拉取 → branches）
  useEffect(() => {
    if (open) {
      void loadBranches()
    }
  }, [open, loadBranches])

  const confirmGenerate = () => {
    const chosen = selectedBranch !== null ? branches[selectedBranch] : null
    const plotReq = userInput.trim() || (chosen ? `${chosen.title}：${chosen.pitch}` : '')
    onStart(plotReq, overwriteChapter, branchFromID)
    onClose()
  }

  return (
    <Modal title={<><BulbOutlined style={{ marginRight: 8 }} />剧情方向</>}
      open={open} onCancel={onClose} footer={null} width={620}
      destroyOnHidden transitionName="" maskTransitionName="">
      {wizStep === 'loading' && <div style={{ textAlign: 'center', padding: 24 }}><Spin size="large" /><div style={{ marginTop: 8, color: C('color-text-secondary'), fontSize: 12 }}>AI 正在分析设定，构思剧情分支…</div></div>}
      {wizStep === 'branches' && (<React.Fragment>
        {branches.length > 0 && <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 12 }}>
          {branches.map((b, i) => (
            <Card key={i} size="small" hoverable onClick={() => { setSelectedBranch(i); setUserInput('') }}
              style={{ cursor: 'pointer', border: selectedBranch === i ? '2px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)', background: selectedBranch === i ? 'var(--md-sys-color-primary-container)' : 'var(--bg-elevated)' }}>
              <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{i + 1}. {b.title}</Typography.Text>
              <Typography.Paragraph style={{ color: C('color-text-secondary'), fontSize: 12, margin: '4px 0 0' }}>{b.pitch}</Typography.Paragraph>
            </Card>
          ))}
        </div>}
        <Space direction="vertical" style={{ width: '100%' }}>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>或输入你自己的剧情要求：</Typography.Text>
          <Input.TextArea value={userInput} onChange={e => { setUserInput(e.target.value); setSelectedBranch(null) }} rows={2} placeholder="剧情要求…"
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 'var(--radius-md)' }} />
        </Space>
        <div style={{ marginTop: 12, textAlign: 'right' }}>
          <Space>
            <Button onClick={() => void loadBranches()}>重新构思</Button>
            <Button type="primary" onClick={confirmGenerate} disabled={selectedBranch === null && !userInput.trim()}>
              {overwriteChapter > 0 ? `重新生成第${overwriteChapter}章` : '生成'}
            </Button>
          </Space>
        </div>
      </React.Fragment>)}
    </Modal>
  )
}

export default BranchWizardModal
