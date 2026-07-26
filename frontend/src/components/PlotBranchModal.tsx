import React from 'react'
import { Modal, message } from 'antd'
import { BranchesOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import { usePlotBranch } from '../hooks/usePlotBranch'
import BranchSelectorPanel from './BranchSelectorPanel'

interface PlotBranchModalProps {
  open: boolean
  onClose: () => void
  nodeID: string
  nodeTitle: string
  onApplied?: () => void
}

const PlotBranchModal: React.FC<PlotBranchModalProps> = ({ open, onClose, nodeID, nodeTitle, onApplied }) => {
  const { loading, branches, selected, setSelected, applying, handleBrainstorm, handleApply } = usePlotBranch(nodeID, () => {
    onApplied?.()
    onClose()
  })

  const handleApplyBranch = async (manualMode: boolean, input: string) => {
    if (selected < 0 && !manualMode) {
      message.warning('请选择分支或手工录入')
      return
    }
    await handleApply(manualMode, input)
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><BranchesOutlined style={{ color: '#c084fc', marginRight: 8 }} />剧情分支 · {nodeTitle}</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={640}
      styles={{ body: { background: C('color-bg-container'), maxHeight: '70vh', overflow: 'auto' }, header: { background: C('color-bg-container') } }}
    >
      <BranchSelectorPanel
        loading={loading}
        branches={branches}
        selected={selected}
        setSelected={setSelected}
        onBrainstorm={handleBrainstorm}
        applying={applying}
        onApply={handleApplyBranch}
      />
    </Modal>
  )
}

export default PlotBranchModal
