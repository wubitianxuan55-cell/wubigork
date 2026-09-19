import React from 'react'
import { Typography, Input } from 'antd'
import { C } from '../../../utils/theme'

/** L 标签文字 */
export function L({ children }: { children: React.ReactNode }) {
  return <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>{children}</Typography.Text>
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
      style={{ background: 'color-mix(in srgb, var(--color-text) 5%, transparent)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)' }} />
  )
  return (
    <div>
      <L>{l}</L>
      {input}
    </div>
  )
}
