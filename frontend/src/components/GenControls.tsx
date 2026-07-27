import React from 'react'
import { Typography, InputNumber, Button, Input } from 'antd'
import { ShakeOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

const SIZE_OPTIONS = [
  { label: '🟦 1:1', value: '1024x1024' },
  { label: '🖼 4:3', value: '1024x768' },
  { label: '🎬 16:9', value: '1024x576' },
  { label: '📱 9:16', value: '576x1024' },
  { label: '📐 3:4', value: '768x1024' },
  { label: '🖥 21:9', value: '1280x544' },
  { label: '✏️ 自定义', value: 'custom' },
]

const COUNT_OPTIONS = [
  { label: '1', value: 1 },
  { label: '2', value: 2 },
  { label: '3', value: 3 },
  { label: '4', value: 4 },
]

// ── 卡片选择器（内联小组件） ──

interface CardPickerProps {
  options: { label: string; value: string | number }[]
  value: string | number
  onChange: (v: any) => void
  compact?: boolean
}

const CardPicker: React.FC<CardPickerProps> = ({ options, value, onChange, compact }) => (
  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
    {options.map((o) => {
      const selected = o.value === value
      return (
        <div
          key={String(o.value)}
          onClick={() => onChange(o.value)}
          title={o.label}
          style={{
            padding: compact ? '3px 8px' : '5px 10px',
            borderRadius: 'var(--radius-sm)',
            border: selected
              ? '1px solid var(--color-primary)'
              : '1px solid var(--border-subtle)',
            background: selected
              ? 'rgba(99, 102, 241, 0.12)'
              : 'rgba(255,255,255,0.03)',
            cursor: 'pointer',
            fontSize: 11,
            fontWeight: selected ? 600 : 400,
            color: selected ? 'var(--color-primary)' : C('color-text-secondary'),
            transition: 'all 0.15s',
            textAlign: 'center' as const,
            whiteSpace: 'nowrap' as const,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            maxWidth: compact ? 60 : 180,
            userSelect: 'none' as const,
          }}
        >
          {o.label}
        </div>
      )
    })}
  </div>
)

// ── Props ──

interface Props {
  size: string
  model: string
  seed: number
  count: number
  modelOptions: { label: string; value: string }[]
  customWidth: number
  customHeight: number
  onSizeChange: (v: string) => void
  onModelChange: (v: string) => void
  onSeedChange: (v: number) => void
  onCountChange: (v: number) => void
  onCustomWidthChange: (v: number) => void
  onCustomHeightChange: (v: number) => void
  backendSelector?: React.ReactNode
  // LoRA
  selectedLoras: string[]
  loraOptions: { label: string; value: string }[]
  onLorasChange: (v: string[]) => void
}

const GenControls: React.FC<Props> = ({
  size, model, seed, count,
  modelOptions, customWidth, customHeight,
  onSizeChange, onModelChange, onSeedChange, onCountChange,
  onCustomWidthChange, onCustomHeightChange,
  backendSelector,
  selectedLoras, loraOptions, onLorasChange,
}) => {
  const s = { color: C('color-text-secondary'), fontSize: 10, display: 'block', marginBottom: 6 } as const

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {backendSelector}

      {modelOptions.length > 0 && (
        <div>
          <Typography.Text style={s}>🧠 模型</Typography.Text>
          <CardPicker
            options={modelOptions}
            value={model}
            onChange={onModelChange}
          />
        </div>
      )}
      {modelOptions.length === 0 && (
        <div>
          <Typography.Text style={s}>🧠 模型</Typography.Text>
          <Input size="small" value={model} onChange={e => onModelChange(e.target.value)}
            placeholder="输入模型名称"
            style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: 'var(--color-text)', borderRadius: 'var(--radius-sm)' }} />
        </div>
      )}

      {loraOptions.length > 0 && (
        <div>
          <Typography.Text style={s}>🎨 LoRA（可多选）</Typography.Text>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {loraOptions.map((lo) => {
              const active = selectedLoras.includes(lo.value)
              return (
                <div
                  key={lo.value}
                  onClick={() => {
                    if (active) {
                      onLorasChange(selectedLoras.filter(v => v !== lo.value))
                    } else {
                      onLorasChange([...selectedLoras, lo.value])
                    }
                  }}
                  title={lo.label}
                  style={{
                    padding: '3px 8px',
                    borderRadius: 'var(--radius-sm)',
                    border: active
                      ? '1px solid var(--color-primary)'
                      : '1px solid var(--border-subtle)',
                    background: active
                      ? 'rgba(99, 102, 241, 0.12)'
                      : 'rgba(255,255,255,0.03)',
                    cursor: 'pointer',
                    fontSize: 11,
                    fontWeight: active ? 600 : 400,
                    color: active ? 'var(--color-primary)' : C('color-text-secondary'),
                    transition: 'all 0.15s',
                    userSelect: 'none' as const,
                  }}
                >
                  {lo.label}
                </div>
              )
            })}
          </div>
        </div>
      )}

      <div>
        <Typography.Text style={s}>📐 尺寸</Typography.Text>
        <CardPicker
          options={SIZE_OPTIONS}
          value={size}
          onChange={onSizeChange}
        />
        {size === 'custom' && (
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <InputNumber value={customWidth} onChange={(v) => onCustomWidthChange(v || 1024)}
              size="small" min={256} max={2048} step={64}
              addonBefore="宽" style={{ flex: 1 }} />
            <InputNumber value={customHeight} onChange={(v) => onCustomHeightChange(v || 1024)}
              size="small" min={256} max={2048} step={64}
              addonBefore="高" style={{ flex: 1 }} />
          </div>
        )}
      </div>

      <div style={{ display: 'flex', gap: 12 }}>
        <div style={{ flex: 1 }}>
          <Typography.Text style={s}>🎲 种子</Typography.Text>
          <InputNumber
            value={seed || undefined}
            onChange={(v) => onSeedChange(v || 0)}
            placeholder="随机"
            min={1} max={2147483647}
            style={{ width: '100%' }}
            addonAfter={
              <Button type="text" size="small" icon={<ShakeOutlined />}
                onClick={() => onSeedChange(0)}
                style={{ padding: 0, height: 18 }} />
            }
          />
        </div>
        <div>
          <Typography.Text style={s}>数量</Typography.Text>
          <CardPicker
            options={COUNT_OPTIONS}
            value={count}
            onChange={onCountChange}
            compact
          />
        </div>
      </div>
    </div>
  )
}

export default GenControls
