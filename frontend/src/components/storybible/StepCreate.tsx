import React from 'react'
import { Button, Input, Typography, Space, message } from 'antd'
import { ThunderboltOutlined, UploadOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

interface StepCreateProps {
  title: string
  onTitleChange: (v: string) => void
  style: string
  onStyleChange: (v: string) => void
  reference: string
  onReferenceChange: (v: string) => void
  importName: string
  onImportFile: () => void
  loading: boolean
  onBootstrap: () => void
}

const StepCreate: React.FC<StepCreateProps> = ({
  title, onTitleChange, style, onStyleChange,
  reference, onReferenceChange, importName, onImportFile,
  loading, onBootstrap,
}) => (
  <div>
    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
      小说标题
    </Typography.Text>
    <Input
      value={title}
      onChange={(e) => onTitleChange(e.target.value)}
      placeholder="你的小说标题"
      size="large"
      style={{
        marginBottom: 12,
        background: 'rgba(255,255,255,0.03)',
        border: '1px solid var(--border-subtle)',
        color: 'var(--color-text)',
        borderRadius: 'var(--radius-md)',
      }}
    />
    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
      文风参考（可选）
    </Typography.Text>
    <Input.TextArea
      value={style}
      onChange={(e) => onStyleChange(e.target.value)}
      placeholder="例如：文风类似《凡人修仙传》，偏写实，注重修炼体系细节..."
      rows={2}
      style={{
        marginBottom: 12,
        background: 'rgba(255,255,255,0.03)',
        border: '1px solid var(--border-subtle)',
        color: 'var(--color-text)',
        borderRadius: 'var(--radius-md)',
      }}
    />
    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
      参考素材（可选）
    </Typography.Text>
    <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
      <Input.TextArea
        value={reference}
        onChange={(e) => onReferenceChange(e.target.value)}
        placeholder="粘贴参考文本..."
        rows={2}
        style={{
          flex: 1,
          background: 'rgba(255,255,255,0.03)',
          border: '1px solid var(--border-subtle)',
          color: 'var(--color-text)',
          borderRadius: 'var(--radius-md)',
        }}
      />
      <Button icon={<UploadOutlined />} onClick={onImportFile}
        style={{ display: 'flex', alignItems: 'center', flexDirection: 'column', justifyContent: 'center', height: 'auto', minHeight: 50 }}>
        导入文件<br /><small>{importName || ''}</small>
      </Button>
    </div>
    <Button
      type="primary" size="large"
      icon={<ThunderboltOutlined />}
      onClick={onBootstrap} loading={loading}
      style={{ width: '100%' }}
    >
      {loading ? 'AI 正在创建小说...' : '一键生成全部'}
    </Button>
  </div>
)


export default StepCreate
