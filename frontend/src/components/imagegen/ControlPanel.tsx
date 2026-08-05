import React from 'react'
import { Input, InputNumber, Button, Typography } from 'antd'
import {
  CloudOutlined, DesktopOutlined, RocketOutlined, KeyOutlined,
  EditOutlined, SlidersOutlined, CloudServerOutlined, DashboardOutlined,
  RobotOutlined, ExperimentOutlined, PictureOutlined, ShakeOutlined, NumberOutlined,
  MinusCircleOutlined, PlusCircleOutlined, AppstoreOutlined,
  PlayCircleOutlined, PoweroffOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import type { SystemStats } from '../../api/image'
import { SectionBlock, SectionDivider, PickerGroup, StatusDot, ActionButton } from './ui'

const { TextArea } = Input

const SIZE_OPTIONS = [
  { label: '1:1', value: '1024x1024' },
  { label: '4:3', value: '1024x768' },
  { label: '16:9', value: '1024x576' },
  { label: '9:16', value: '576x1024' },
  { label: '3:4', value: '768x1024' },
  { label: '21:9', value: '1280x544' },
  { label: '自定义', value: 'custom' },
]

const COUNT_OPTIONS = [
  { label: '1', value: 1 },
  { label: '2', value: 2 },
  { label: '3', value: 3 },
  { label: '4', value: 4 },
]

const BACKEND_OPTIONS = [
  { label: 'xAI', value: 'xai', icon: <CloudOutlined /> },
  { label: 'ComfyUI', value: 'comfyui', icon: <DesktopOutlined /> },
  { label: 'Herdsman', value: 'herdsman', icon: <RocketOutlined /> },
  { label: 'Ollama', value: 'ollama', icon: <KeyOutlined /> },
]

const estimateTime = (backend: string, model: string, count: number) => {
  if (backend === 'xai') return count * 5
  if (model === 'z-image-turbo') return count * 20
  if (model.startsWith('krea2')) return count * 300
  return count * 60
}

const labelStyle: React.CSSProperties = {
  color: C('color-text-secondary'), fontSize: 12, display: 'flex', alignItems: 'center', gap: 5,
  marginBottom: 6, fontWeight: 500,
}

const inputStyle: React.CSSProperties = {
  background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
  color: 'var(--color-text)', borderRadius: 10, fontSize: 13,
}

export interface ControlPanelProps {
  prompt: string
  negative: string
  onPromptChange: (v: string) => void
  onNegativeChange: (v: string) => void
  onOpenTemplatePicker: () => void
  model: string
  modelOptions: { label: string; value: string }[]
  onModelChange: (v: string) => void
  size: string
  onSizeChange: (v: string) => void
  customWidth: number
  customHeight: number
  onCustomWidthChange: (v: number) => void
  onCustomHeightChange: (v: number) => void
  seed: number
  onSeedChange: (v: number) => void
  count: number
  onCountChange: (v: number) => void
  selectedLoras: string[]
  loraOptions: { label: string; value: string }[]
  onLorasChange: (v: string[]) => void
  backend: string
  backendSwitching: boolean
  engineRunning: boolean
  engineStarting: boolean
  engineModelCount: number
  onSwitchBackend: (v: string) => void
  onStartEngine: () => void
  onStopEngine: () => void
  sysStats: SystemStats | null
  generating: boolean
  elapsed: number
  lastTime: number
  onGenerate: () => void
}

export const ControlPanel: React.FC<ControlPanelProps> = ({
  prompt, negative, onPromptChange, onNegativeChange, onOpenTemplatePicker,
  model, modelOptions, onModelChange,
  size, onSizeChange, customWidth, customHeight, onCustomWidthChange, onCustomHeightChange,
  seed, onSeedChange, count, onCountChange,
  selectedLoras, loraOptions, onLorasChange,
  backend, backendSwitching, engineRunning, engineStarting, engineModelCount,
  onSwitchBackend, onStartEngine, onStopEngine,
  sysStats, generating, elapsed, lastTime, onGenerate,
}) => {
  const [showNegative, setShowNegative] = React.useState(false)
  const isLocal = ['comfyui', 'herdsman', 'ollama'].includes(backend)
  const est = estimateTime(backend, model, count)
  const hint = generating
    ? `已用时 ${elapsed}s`
    : lastTime > 0 ? `上次 ${lastTime}s · 预计约 ${est}s` : `预计约 ${est}s`

  return (
    <div style={{
      background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
      WebkitBackdropFilter: 'blur(18px) saturate(140%)',
      backdropFilter: 'blur(18px) saturate(140%)',
      borderRadius: 'var(--radius-lg)', border: '1px solid var(--md-sys-color-outline-variant)',
      padding: '14px 16px', display: 'flex', flexDirection: 'column', gap: 16,
    }}>
      {/* ── ① 提示词 ── */}
      <SectionBlock title="提示词" icon={<EditOutlined />}>
        <TextArea
          placeholder="描述你想要的画面，例如：一座悬浮在云端的东方仙侠城市，琉璃瓦宫殿，瀑布倾泻，落日余晖..."
          value={prompt}
          onChange={(e) => onPromptChange(e.target.value)}
          autoSize={{ minRows: 4, maxRows: 7 }}
          style={{
            ...inputStyle, resize: 'none', lineHeight: 1.6, fontSize: 13,
            padding: '10px 12px',
          }}
        />
        <button
          type="button"
          onClick={() => setShowNegative((v) => !v)}
          className="img-picker-btn"
          style={{
            display: 'flex', alignItems: 'center', gap: 5, border: 'none', background: 'none',
            cursor: 'pointer', padding: 0, fontSize: 12, color: C('color-text-secondary'),
            fontFamily: 'inherit',
          }}
        >
          {showNegative ? <MinusCircleOutlined /> : <PlusCircleOutlined />}
          {showNegative ? '收起不想出现的内容' : '添加不想出现的内容'}
        </button>
        {showNegative && (
          <TextArea
            placeholder="模糊, 低质量, 畸形手指, 多余肢体..."
            value={negative}
            onChange={(e) => onNegativeChange(e.target.value)}
            autoSize={{ minRows: 2, maxRows: 4 }}
            style={{ ...inputStyle, resize: 'none', lineHeight: 1.6, fontSize: 12, padding: '8px 12px' }}
          />
        )}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography.Text style={labelStyle}>模板</Typography.Text>
          <Button
            type="text" size="small" icon={<AppstoreOutlined />}
            onClick={onOpenTemplatePicker}
            style={{ color: 'var(--color-primary)', fontSize: 12, padding: '0 4px' }}
          >
            模板库
          </Button>
        </div>
      </SectionBlock>

      <SectionDivider />

      {/* ── ② 参数 ── */}
      <SectionBlock title="参数" icon={<SlidersOutlined />}>
        <div>
          <Typography.Text style={labelStyle}><RobotOutlined />模型</Typography.Text>
          {modelOptions.length > 0 ? (
            <PickerGroup options={modelOptions} value={model} onChange={onModelChange} />
          ) : (
            <Input size="small" value={model} onChange={(e) => onModelChange(e.target.value)}
              placeholder="输入模型名称" style={{ ...inputStyle, height: 32 }} />
          )}
        </div>

        {loraOptions.length > 0 && (
          <div>
            <div style={{ ...labelStyle, justifyContent: 'space-between' }}>
              <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                <ExperimentOutlined />LoRA
              </span>
              {selectedLoras.length > 0 && (
                <span style={{ fontSize: 11, color: 'var(--color-primary)', fontWeight: 600 }}>
                  已选 {selectedLoras.length} 个
                </span>
              )}
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 5 }}>
              {loraOptions.map((lo) => {
                const active = selectedLoras.includes(lo.value)
                return (
                  <button
                    key={lo.value}
                    type="button"
                    title={lo.label}
                    onClick={() => onLorasChange(
                      active ? selectedLoras.filter((v) => v !== lo.value) : [...selectedLoras, lo.value],
                    )}
                    className="img-picker-btn"
                    style={{
                      padding: '5px 9px', borderRadius: 999, cursor: 'pointer', fontSize: 11,
                      border: '1px solid',
                      borderColor: active ? 'var(--color-primary)' : 'var(--border-subtle)',
                      background: active ? 'rgba(var(--accent-rgb), 0.14)' : 'rgba(255,255,255,0.03)',
                      color: active ? 'var(--color-primary)' : C('color-text-secondary'),
                      fontWeight: active ? 600 : 400, fontFamily: 'inherit', whiteSpace: 'nowrap',
                    }}
                  >
                    {lo.label}
                  </button>
                )
              })}
            </div>
          </div>
        )}

        <div>
          <Typography.Text style={labelStyle}><PictureOutlined />尺寸</Typography.Text>
          <PickerGroup options={SIZE_OPTIONS} value={size} onChange={onSizeChange} columns={4} />
          {size === 'custom' && (
            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <InputNumber value={customWidth} onChange={(v) => onCustomWidthChange(v || 1024)}
                size="small" min={256} max={2048} step={64} addonBefore="宽" style={{ flex: 1 }} />
              <InputNumber value={customHeight} onChange={(v) => onCustomHeightChange(v || 1024)}
                size="small" min={256} max={2048} step={64} addonBefore="高" style={{ flex: 1 }} />
            </div>
          )}
        </div>

        <div style={{ display: 'flex', gap: 12 }}>
          <div style={{ flex: 1 }}>
            <Typography.Text style={labelStyle}><ShakeOutlined />种子</Typography.Text>
            <InputNumber
              value={seed || undefined}
              onChange={(v) => onSeedChange(v || 0)}
              placeholder="随机"
              min={1} max={2147483647}
              style={{ width: '100%' }}
              addonAfter={
                <Button type="text" size="small" icon={<ShakeOutlined />}
                  onClick={() => onSeedChange(0)} style={{ padding: 0, height: 18 }} />
              }
            />
          </div>
          <div>
            <Typography.Text style={labelStyle}><NumberOutlined />数量</Typography.Text>
            <PickerGroup options={COUNT_OPTIONS} value={count} onChange={onCountChange} columns={4} />
          </div>
        </div>
      </SectionBlock>

      <SectionDivider />

      {/* ── ③ 引擎 ── */}
      <SectionBlock title="引擎" icon={<CloudServerOutlined />}>
        <PickerGroup
          options={BACKEND_OPTIONS}
          value={backend}
          onChange={onSwitchBackend}
          columns={4}
        />
        {isLocal && (
          <div style={{
            display: 'flex', alignItems: 'center', gap: 8,
            background: 'var(--bg-elevated)', borderRadius: 10,
            border: '1px solid var(--border-subtle)', padding: '8px 10px',
          }}>
            <StatusDot tone={engineStarting ? 'warn' : engineRunning ? 'ok' : 'idle'} />
            <span style={{ fontSize: 12, color: C('color-text-secondary'), flex: 1 }}>
              {engineStarting
                ? '启动中，等待就绪...'
                : engineRunning
                  ? `运行中${engineModelCount > 0 ? ` · ${engineModelCount} 个模型` : ''}`
                  : '未启动'}
            </span>
            {engineRunning && !engineStarting ? (
              <Button size="small" danger icon={<PoweroffOutlined />} onClick={onStopEngine}
                style={{ borderRadius: 999, fontSize: 12, flexShrink: 0 }}>停止</Button>
            ) : (
              <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                loading={engineStarting || backendSwitching} onClick={onStartEngine}
                style={{ borderRadius: 999, fontSize: 12, flexShrink: 0 }}>启动</Button>
            )}
          </div>
        )}
      </SectionBlock>

      {sysStats && (
        <>
          <SectionDivider />
          <SectionBlock title="系统" icon={<DashboardOutlined />}>
            <MetricBar label="CPU" value={sysStats.cpu} detail={`${sysStats.cpu}%`} />
            <MetricBar
              label="内存"
              value={sysStats.memTotal > 0 ? (sysStats.memUsed / sysStats.memTotal) * 100 : 0}
              detail={`${sysStats.memUsed.toFixed(0)}/${sysStats.memTotal.toFixed(0)}GB`}
            />
            {sysStats.gpuName && (
              <MetricBar
                label={sysStats.gpuName.length > 18 ? sysStats.gpuName.slice(0, 18) + '…' : sysStats.gpuName}
                value={sysStats.vramTotal > 0 ? (sysStats.vramUsed / sysStats.vramTotal) * 100 : 0}
                detail={`${sysStats.vramUsed.toFixed(0)}/${sysStats.vramTotal.toFixed(0)}GB`}
              />
            )}
          </SectionBlock>
        </>
      )}

      {/* ── ⑤ 生成（常驻底部） ── */}
      <ActionButton
        loading={generating}
        label={generating ? '生成中' : `生成 ${count} 张`}
        hint={hint}
        onClick={onGenerate}
      />
    </div>
  )
}

/** 迷你指标条 */
const MetricBar: React.FC<{ label: string; value: number; detail: string }> = ({ label, value, detail }) => {
  const pct = Math.min(Math.max(value, 0), 100)
  const color = pct < 60 ? 'var(--color-success)' : pct < 85 ? 'var(--color-warning)' : '#f87171'
  return (
    <div style={{
      background: 'var(--bg-elevated)', borderRadius: 10, border: '1px solid var(--border-subtle)',
      padding: '6px 10px',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
        <span style={{ fontSize: 11, color: C('color-text-secondary') }}>{label}</span>
        <span style={{ fontSize: 11, fontWeight: 600, color }}>{detail}</span>
      </div>
      <div style={{ height: 4, background: 'rgba(255,255,255,0.08)', borderRadius: 2, overflow: 'hidden' }}>
        <div style={{
          width: `${pct}%`, height: '100%', background: color, borderRadius: 2,
          transition: 'width 0.6s cubic-bezier(0.32, 0.72, 0, 1)',
        }} />
      </div>
    </div>
  )
}
