import React from 'react'
import { Typography, Input, Select } from 'antd'
import { C } from '../utils/theme'

interface SettingFieldProps {
  label: string
  value: string | number
  onChange: (v: unknown) => void
  type?: 'text' | 'number' | 'select'
  options?: { label: string; value: string }[]
  placeholder?: string
  width?: string | number
}

const inputStyle: React.CSSProperties = {
  background: 'var(--md-sys-color-surface-container)',
  border: '1px solid var(--border-subtle, var(--md-sys-color-outline-variant))',
  borderRadius: 'var(--radius-md)',
  color: 'var(--color-text)',
}

/** SettingField — 统一「标签 + 输入框/选择框」模式 */
const SettingField: React.FC<SettingFieldProps> = ({
  label, value, onChange, type = 'text',
  options, placeholder, width,
}) => (
  <div>
    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
      {label}
    </Typography.Text>
    {type === 'select' && options ? (
      <Select
        value={value as string}
        onChange={onChange}
        style={{ width: width || '100%' }}
        options={options}
      />
    ) : (
      <Input
        type={type}
        placeholder={placeholder}
        value={value as string}
        onChange={(e) => onChange(type === 'number' ? parseInt(e.target.value) || 0 : e.target.value)}
        style={{ ...inputStyle, width: width || '100%' }}
      />
    )}
  </div>
)

export default SettingField
