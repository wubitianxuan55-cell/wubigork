import React from 'react'
import { Typography, Space, Input, Modal, Checkbox } from 'antd'
import { C } from '../../utils/theme'
import { GENRE_OPTIONS, STYLE_OPTIONS } from './novelOptions'

interface ImportNovelModalProps {
  open: boolean
  fileName: string
  title: string
  genre: string[]
  style: string[]
  importing: boolean
  onTitleChange: (v: string) => void
  onGenreChange: (v: string[]) => void
  onStyleChange: (v: string[]) => void
  onImport: () => void
  onClose: () => void
}

/** 导入成品小说：选择文件后填写书名 / 题材 / 文风，确认后解析入库 */
const ImportNovelModal: React.FC<ImportNovelModalProps> = ({
  open, fileName, title, genre, style, importing,
  onTitleChange, onGenreChange, onStyleChange, onImport, onClose,
}) => (
  <Modal
    title={<span style={{ color: C('color-text') }}>导入成品小说</span>}
    open={open}
    onOk={onImport}
    onCancel={onClose}
    okText={importing ? '导入中…' : '导入'}
    cancelText="取消"
    confirmLoading={importing}
    okButtonProps={{ disabled: !title.trim() }}
    width={520}
    destroyOnHidden
    transitionName=""
    maskTransitionName=""
    styles={{
      body: { background: 'transparent' },
      header: { background: 'transparent' },
    }}
  >
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block' }}>
        文件：<span style={{ wordBreak: 'break-all' }}>{fileName}</span>
      </Typography.Text>
      <Input
        placeholder="小说标题（必填）"
        value={title}
        onChange={(e) => onTitleChange(e.target.value)}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
      />

      <div>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 6, display: 'block' }}>
          题材（可多选，支持 TXT / Markdown / EPUB）
        </Typography.Text>
        <Checkbox.Group
          options={GENRE_OPTIONS}
          value={genre}
          onChange={(v) => onGenreChange(v as string[])}
          style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
        />
      </div>

      <div>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 6, display: 'block' }}>
          文风（可多选）
        </Typography.Text>
        <Checkbox.Group
          options={STYLE_OPTIONS}
          value={style}
          onChange={(v) => onStyleChange(v as string[])}
          style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
        />
      </div>
    </Space>
  </Modal>
)

export default ImportNovelModal
