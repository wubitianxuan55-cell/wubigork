import React from 'react'
import { Typography, Input, Select } from 'antd'
import { C } from '../../utils/theme'

/** Block 带标题的折叠块 */
export function Block({ title, extra, children }: { title: React.ReactNode; extra?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div style={{ marginBottom: 10, padding: '10px 14px', background: 'var(--bg-elevated)', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-subtle)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
        <Typography.Text strong style={{ color: 'var(--color-text-secondary)', fontSize: 11 }}>{title}</Typography.Text>
        {extra}
      </div>
      {children}
    </div>
  )
}

/** L 标签文字 */
export function L({ children }: { children: React.ReactNode }) {
  return <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>{children}</Typography.Text>
}

/** 从文本中提取关键词标签（按中文/英文逗号、分号、顿号分隔） */
export function extractTags(text: string): string[] {
  return text.split(/[，,、；;]/).map((s) => s.trim()).filter((s) => s.length > 1)
}

export interface FieldProps {
  l: string
  v?: string
  onChange: (s: string) => void
  type?: 'text' | 'textarea' | 'select'
  rows?: number
  options?: { value: string; label: string }[]
  value?: string
  noBlock?: boolean
}

/** Field 统一表单字段 */
export function Field({ l, v = '', onChange, type = 'text', rows = 3, options, value, noBlock = false }: FieldProps) {
  const actualValue = type === 'select' ? (value ?? v) : v
  const input = type === 'select' && options ? (
    <Select size="small" value={actualValue} onChange={onChange}
      style={{ width: '100%', background: 'rgba(255,255,255,0.05)', borderColor: 'var(--border-subtle)', borderRadius: 'var(--radius-md)' }}
      options={options} />
  ) : type === 'textarea' ? (
    <Input.TextArea value={v} onChange={(e) => onChange(e.target.value)} rows={rows}
      style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13, lineHeight: 1.7, boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)' }} />
  ) : (
    <Input size="small" value={v} onChange={(e) => onChange(e.target.value)}
      style={{ background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)' }} />
  )
  if (noBlock) return <div><L>{l}</L>{input}</div>
  return (
    <div style={{ marginBottom: 10, padding: '10px 14px', background: 'var(--bg-elevated)', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-subtle)' }}>
      <L>{l}</L>
      {input}
    </div>
  )
}

/** OrgField — 组织编辑用的简化字段组件 */
export function OrgField({ l, v = '', onChange, type = 'text', rows = 2 }: {
  l: string; v?: string; onChange: (s: string) => void; type?: 'text' | 'textarea'; rows?: number
}) {
  const input = type === 'textarea' ? (
    <Input.TextArea value={v} onChange={(e) => onChange(e.target.value)} rows={rows}
      style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13 }} />
  ) : (
    <Input value={v} onChange={(e) => onChange(e.target.value)}
      style={{ background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)' }} />
  )
  return (
    <div>
      <L>{l}</L>
      {input}
    </div>
  )
}
