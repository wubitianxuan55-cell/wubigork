/* eslint-disable react-refresh/only-export-components -- TEMPLATE_SIZE_OPTIONS 常量导出供生成流程复用 */
import React from 'react'
import { Typography, Input, Space, Modal, Select } from 'antd'
import { C } from '../../utils/theme'

const { TextArea } = Input

export const TEMPLATE_SIZE_OPTIONS = [
  { label: '不指定', value: '' },
  { label: '1:1 方形', value: '1:1' },
  { label: '16:9 横屏', value: '16:9' },
  { label: '9:16 竖屏', value: '9:16' },
  { label: '4:3', value: '4:3' },
  { label: '3:4', value: '3:4' },
  { label: '2:3 立绘', value: '2:3' },
]

interface CustomTemplateModalProps {
  open: boolean
  editing: boolean
  label: string
  onLabelChange: (v: string) => void
  description: string
  onDescriptionChange: (v: string) => void
  size: string
  onSizeChange: (v: string) => void
  prompt: string
  onPromptChange: (v: string) => void
  negative: string
  onNegativeChange: (v: string) => void
  onSave: () => void
  onCancel: () => void
}

/** CustomTemplateModal — 自定义模板编辑弹窗（含用途说明与推荐画幅） */
const CustomTemplateModal: React.FC<CustomTemplateModalProps> = ({
  open, editing, label, onLabelChange,
  description, onDescriptionChange, size, onSizeChange,
  prompt, onPromptChange, negative, onNegativeChange,
  onSave, onCancel,
}) => (
  <Modal
    title={editing ? '编辑自定义模板' : '新建自定义模板'}
    open={open}
    onOk={onSave}
    onCancel={onCancel}
    destroyOnHidden
    transitionName=""
    maskTransitionName=""
    okText="保存"
    cancelText="取消"
    width={440}
    styles={{
      body: { background: 'transparent' },
      header: { background: 'transparent' },
    }}
  >
    <Space direction="vertical" size={10} style={{ width: '100%' }}>
      <Input placeholder="模板名称（如：古风侠客）" value={label}
        onChange={(e) => onLabelChange(e.target.value)}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
      <Input placeholder="用途说明（可选，一句话描述适合什么场景）" value={description}
        onChange={(e) => onDescriptionChange(e.target.value)}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Typography.Text style={{ fontSize: 12, color: C('color-text-secondary'), flexShrink: 0 }}>
          推荐画幅
        </Typography.Text>
        <Select
          value={size}
          onChange={onSizeChange}
          options={TEMPLATE_SIZE_OPTIONS}
          size="small"
          style={{ width: 160 }}
          popupMatchSelectWidth={false}
        />
      </div>
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
