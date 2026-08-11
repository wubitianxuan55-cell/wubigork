import React, { useState, useEffect } from 'react'
import { Typography, Button, Modal, message } from 'antd'
import { BranchesOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import type { OutlineNode } from '../../types'
import { usePlotBranch } from './hooks/usePlotBranch'
import BranchSelectorPanel from './BranchSelectorPanel'

interface NextChapterModalProps {
  open: boolean
  onClose: () => void
  currentNode: OutlineNode | null
  nextNode: OutlineNode | null
  onGenerate: (nodeID: string, keyInfo: string) => Promise<void>
}

const NextChapterModal: React.FC<NextChapterModalProps> = ({
  open, onClose, currentNode, nextNode, onGenerate,
}) => {
  const nodeID = nextNode?.id || ''
  const { loading, branches, selected, setSelected, handleBrainstorm } = usePlotBranch(nodeID)

  const [keyInfo, setKeyInfo] = useState('')
  const [generating, setGenerating] = useState(false)

  useEffect(() => {
    if (open) {
      setKeyInfo(''); setGenerating(false)
    }
  }, [open])

  const handleGenerate = async (_manualMode: boolean, input: string) => {
    if (!nextNode) return
    setGenerating(true)
    try {
      await onGenerate(nextNode.id, input)
      message.success('正在生成下一章')
      onClose()
    } catch (err: any) {
      message.error(err.message || '生成失败')
    } finally { setGenerating(false) }
  }

  const handleBranchClick = (branch: any) => {
    setKeyInfo(branch.summary)
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><BranchesOutlined style={{ color: '#c084fc', marginRight: 8 }} />下一章方向</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={600}
      styles={{ body: { background: 'transparent', maxHeight: '70vh', overflow: 'auto' }, header: { background: 'transparent' } }}
    >
      <BranchSelectorPanel
        loading={loading}
        branches={branches}
        selected={selected}
        setSelected={setSelected}
        onBrainstorm={handleBrainstorm}
        applying={generating}
        onApply={handleGenerate}
        onBranchClick={handleBranchClick}
        autoBrainstorm
        brainstormText="AI 推理续写方向"
        brainstormLoadingText="AI 正在分析剧情走向..."
        branchListTitle="AI 推荐的下一章方向（点击选择）"
        applyText="以此方向生成下一章"
        applyLoadingText="正在生成..."
        manualTriggerText="或手工输入续写方向"
        manualPlaceholder="描述下一章的情节走向..."
        manualApplyText="确认生成"
        header={
          currentNode ? (
            <div style={{ padding: '8px 12px', background: 'rgba(192,132,252,0.06)', borderRadius: 'var(--radius-md)', borderLeft: '2px solid #c084fc' }}>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>已完成：{currentNode.title}</Typography.Text>
            </div>
          ) : undefined
        }
      />
    </Modal>
  )
}

export default NextChapterModal
