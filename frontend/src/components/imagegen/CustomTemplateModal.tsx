import React from 'react'
import { Typography, Button, Input, Space, Modal } from 'antd'
import { C } from '../../utils/theme'

const { TextArea } = Input

interface CustomTemplateModalProps {
  open: boolean
  editing: boolean
  label: string
  onLabelChange: (v: string) => void
  prompt: string
  onPromptChange: (v: string) => void
  negative: string
  onNegativeChange: (v: string) => void
  onSave: () => void
  onCancel: () => void
}

/** CustomTemplateModal — 自定义模板编辑弹窗 */
const CustomTemplateModal: React.FC<CustomTemplateModalProps> = ({
  open, editing, label, onLabelChange,
  prompt, onPromptChange, negative, onNegativeChange,
  onSave, onCancel,
}) => (
  <Modal
    title={editing ? '编辑自定义模板' : '新建自定义模板'}
    open={open}
    onOk={onSave}
    onCancel={onCancel}
    okText="保存"
    cancelText="取消"
    width={420}
    styles={{
      body: { background: C('color-bg-container') },
      header: { background: C('color-bg-container') },
    }}
  >
    <Space direction="vertical" size={10} style={{ width: '100%' }}>
      <Input placeholder="模板名称（如：古风武侠）" value={label}
        onChange={(e) => onLabelChange(e.target.value)}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
      <TextArea
        placeholder="正面 Prompt（点击模板时追加到主 prompt）"
        value={prompt} onChange={(e) => onPromptChange(e.target.value)}
        rows={3}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text'), resize: 'none' }} />
      <TextArea
        placeholder="负面 Prompt（可选，追加到不想出现的内容）"
        value={negative} onChange={(e) => onNegativeChange(e.target.value)}
        rows={2}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text'), resize: 'none' }} />
    </Space>
  </Modal>
)

export default CustomTemplateModal
