import React from 'react'
import { Input, InputNumber, Button, Typography, Slider, Select } from 'antd'
import {
  CloudOutlined, DesktopOutlined, RocketOutlined, KeyOutlined,
  EditOutlined, SlidersOutlined, CloudServerOutlined, DashboardOutlined,
  RobotOutlined, ExperimentOutlined, PictureOutlined, ShakeOutlined, NumberOutlined,
  MinusCircleOutlined, PlusCircleOutlined, AppstoreOutlined,
  PlayCircleOutlined, PoweroffOutlined,
  UploadOutlined, CloseCircleOutlined, VideoCameraOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import type { SystemStats } from '../../api/image'
import { CollapsibleSection, PickerGroup, StatusDot } from './ui'
import type { ImageMode } from './types'

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

const VIDEO_SIZE_OPTIONS = [
  { label: '16:9', value: '640x384' },
  { label: '3:2', value: '768x512' },
  { label: '1:1', value: '704x704' },
  { label: '2:3', value: '512x768' },
  { label: '16:9 HD', value: '1280x736' },
  { label: '自定义', value: 'custom' },
]

const VIDEO_DURATION_OPTIONS = [
  { label: '约 6 秒', value: '49-8' },
  { label: '约 12 秒', value: '97-8' },
  { label: '约 6 秒@16', value: '97-16' },
  { label: '约 4 秒@24', value: '97-24' },
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

const labelStyle: React.CSSProperties = {
  color: C('color-text-secondary'), fontSize: 12, display: 'flex', alignItems: 'center', gap: 5,
  marginBottom: 6, fontWeight: 500,
}

const inputStyle: React.CSSProperties = {
  background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
  color: 'var(--color-text)', borderRadius: 10, fontSize: 13,
}

const loraHintStyle: React.CSSProperties = {
  fontSize: 11, color: C('color-text-secondary'), lineHeight: 1.5,
  padding: '7px 10px', borderRadius: 8, border: '1px dashed var(--border-subtle)',
  background: 'rgba(255,255,255,0.02)',
}

// WebView2 老问题：CSS 动画 tick 被挂起时 antd 下拉弹层卡在 opacity:0 首帧，
// 表现为“下拉弹不出来”。绘梦下拉统一挂到 body + 禁用弹层动画（imagegen.css
// 中 .ig-select-dropdown 规则），打开立即显示，不依赖 rAF 降级检测。
const selectPopupProps = {
  getPopupContainer: () => document.body,
  popupClassName: 'ig-select-dropdown',
}

export interface ControlPanelProps {
  mode: ImageMode
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
  initImage: string
  onInitImageChange: (v: string) => void
  denoise: number
  onDenoiseChange: (v: number) => void
  frames: number
  onFramesChange: (v: number) => void
  fps: number
  onFpsChange: (v: number) => void
  selectedLoras: string[]
  loraOptions: { label: string; value: string }[]
  loraLoading?: boolean
  loraError?: string
  onRefreshLoras?: () => void
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
}

export const ControlPanel: React.FC<ControlPanelProps> = ({
  mode,
  prompt, negative, onPromptChange, onNegativeChange, onOpenTemplatePicker,
  model, modelOptions, onModelChange,
  size, onSizeChange, customWidth, customHeight, onCustomWidthChange, onCustomHeightChange,
  seed, onSeedChange, count, onCountChange,
  initImage, onInitImageChange, denoise, onDenoiseChange,
  frames, onFramesChange, fps, onFpsChange,
  selectedLoras, loraOptions, loraLoading, loraError, onRefreshLoras, onLorasChange,
  backend, backendSwitching, engineRunning, engineStarting, engineModelCount,
  onSwitchBackend, onStartEngine, onStopEngine,
  sysStats,
}) => {
  const [showNegative, setShowNegative] = React.useState(false)
  const fileRef = React.useRef<HTMLInputElement>(null)
  const isLocal = ['comfyui', 'herdsman', 'ollama'].includes(backend)
  const needsComfy = (mode === 't2v' && backend !== 'comfyui')
    || (mode === 'img2img' && backend !== 'comfyui' && backend !== 'herdsman')

  const readFile = (file?: File | null) => {
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => onInitImageChange(String(reader.result || ''))
    reader.readAsDataURL(file)
  }

  return (
    <div className="ig-control-panel" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {/* ── ① 基础设置：提示词 / 负向 / 模板 / 参考图 ── */}
      <CollapsibleSection title="基础设置" icon={<EditOutlined />} defaultOpen>
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

        {mode === 'img2img' && (
          <>
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              style={{ display: 'none' }}
              onChange={(e) => { readFile(e.target.files?.[0]); e.target.value = '' }}
            />
            {initImage ? (
              <div style={{ position: 'relative' }}>
                <img
                  src={initImage}
                  alt="参考图"
                  style={{ width: '100%', maxHeight: 180, objectFit: 'contain', borderRadius: 12, border: '1px solid var(--border-subtle)' }}
                />
                <button
                  type="button"
                  onClick={() => { onInitImageChange('') }}
                  title="移除参考图"
                  className="img-picker-btn"
                  style={{
                    position: 'absolute', top: 6, right: 6, width: 26, height: 26, borderRadius: '50%',
                    border: '1px solid rgba(255,255,255,0.2)', background: 'rgba(0,0,0,0.55)',
                    color: '#fff', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                  }}
                >
                  <CloseCircleOutlined style={{ fontSize: 13 }} />
                </button>
              </div>
            ) : (
              <div
                className="ig-upload-zone"
                onClick={() => fileRef.current?.click()}
                onDragOver={(e) => e.preventDefault()}
                onDrop={(e) => { e.preventDefault(); readFile(e.dataTransfer.files?.[0]) }}
                role="button"
                tabIndex={0}
                onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); fileRef.current?.click() } }}
              >
                <UploadOutlined style={{ fontSize: 22, color: 'var(--color-primary)' }} />
                <div style={{ fontSize: 12.5, color: 'var(--color-text)' }}>点击上传或拖入参考图</div>
                <div style={{ fontSize: 11, color: C('color-text-secondary') }}>PNG / JPG / WebP，将作为构图基础</div>
              </div>
            )}
            {backend === 'comfyui' ? (
              <div>
                <div style={{ ...labelStyle, justifyContent: 'space-between' }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                    <SlidersOutlined />重绘幅度
                  </span>
                  <span style={{ fontSize: 11, color: 'var(--color-primary)', fontWeight: 600 }}>
                    {Math.round(denoise * 100)}%
                  </span>
                </div>
                <Slider
                  min={0.2}
                  max={1}
                  step={0.05}
                  value={denoise}
                  onChange={onDenoiseChange}
                  tooltip={{ formatter: (v) => `${Math.round((v || 0) * 100)}%` }}
                />
                <div style={{ fontSize: 10.5, color: C('color-text-secondary'), lineHeight: 1.5 }}>
                  低幅度保留构图微调风格，高幅度更接近全新创作
                </div>
              </div>
            ) : (
              <div style={{ fontSize: 10.5, color: C('color-text-secondary'), lineHeight: 1.5 }}>
                Herdsman 图生图直接按参考图重绘，不支持重绘幅度调节
              </div>
            )}
          </>
        )}
      </CollapsibleSection>

      {/* ── ② 模型与引擎 ── */}
      <CollapsibleSection title="模型与引擎" icon={<RobotOutlined />} defaultOpen>
        {needsComfy && (
          <div style={{
            padding: '9px 11px', borderRadius: 10, fontSize: 11.5, lineHeight: 1.6,
            border: '1px solid color-mix(in srgb, var(--md-sys-color-warning) 35%, transparent)',
            background: 'color-mix(in srgb, var(--md-sys-color-warning) 8%, transparent)',
            color: 'var(--md-sys-color-warning)',
          }}>
            <VideoCameraOutlined style={{ marginRight: 5 }} />
            {mode === 't2v' ? '文生视频需使用 ComfyUI 本地后端' : '图生图需使用 ComfyUI / Herdsman 本地后端'}，请切换引擎
          </div>
        )}

        <div>
          <Typography.Text style={labelStyle}><CloudServerOutlined />引擎</Typography.Text>
          <Select
            value={backend}
            onChange={onSwitchBackend}
            options={BACKEND_OPTIONS.map((o) => ({
              value: o.value,
              label: (
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                  {o.icon}{o.label}
                </span>
              ),
            }))}
            {...selectPopupProps}
            style={{ width: '100%' }}
          />
          {isLocal && (
            <div style={{
              display: 'flex', alignItems: 'center', gap: 8,
              background: 'var(--bg-elevated)', borderRadius: 10,
              border: '1px solid var(--border-subtle)', padding: '8px 10px', marginTop: 8,
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
        </div>

        <div>
          <Typography.Text style={labelStyle}><PictureOutlined />模型</Typography.Text>
          {modelOptions.length > 0 ? (
            <Select
              showSearch
              value={model}
              onChange={onModelChange}
              options={modelOptions.map((m) => ({ value: m.value, label: m.label }))}
              optionFilterProp="label"
              {...selectPopupProps}
              style={{ width: '100%' }}
            />
          ) : (
            <Input size="small" value={model} onChange={(e) => onModelChange(e.target.value)}
              placeholder="输入模型名称" style={{ ...inputStyle, height: 32 }} />
          )}
          {model === 'diagram' && (
            <div style={{ marginTop: 6, fontSize: 11, color: C('color-text-secondary'), lineHeight: 1.6 }}>
              ✨ 输入图表描述（如“订单处理流程图，含下单、支付、发货、售后”），
              AI 生成图表代码并渲染为图片，中文清晰，适合流程图 / 框架图 / 架构图。
            </div>
          )}
        </div>
      </CollapsibleSection>

      {/* ── ③ 画幅与输出 ── */}
      <CollapsibleSection title="画幅与输出" icon={<PictureOutlined />} defaultOpen>
        <div>
          <Select
            value={size}
            onChange={onSizeChange}
            options={(mode === 't2v' ? VIDEO_SIZE_OPTIONS : SIZE_OPTIONS).map((o) => ({
              value: String(o.value),
              label: o.label,
            }))}
            {...selectPopupProps}
            style={{ width: '100%' }}
          />
          {size === 'custom' && (
            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <InputNumber value={customWidth} onChange={(v) => onCustomWidthChange(v || 1024)}
                size="small" min={256} max={2048} step={64} addonBefore="宽" style={{ flex: 1 }} />
              <InputNumber value={customHeight} onChange={(v) => onCustomHeightChange(v || 1024)}
                size="small" min={256} max={2048} step={64} addonBefore="高" style={{ flex: 1 }} />
            </div>
          )}
        </div>

        {mode === 't2v' && (
          <div>
            <Typography.Text style={labelStyle}><VideoCameraOutlined />时长 / 帧率</Typography.Text>
            <Select
              value={`${frames}-${fps}`}
              onChange={(v) => {
                const [f, p] = String(v).split('-').map(Number)
                onFramesChange(f)
                onFpsChange(p)
              }}
              options={VIDEO_DURATION_OPTIONS.map((o) => ({ value: String(o.value), label: o.label }))}
              {...selectPopupProps}
              style={{ width: '100%' }}
            />
            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <InputNumber value={frames} onChange={(v) => onFramesChange(v || 49)}
                size="small" min={16} max={600} step={8} addonBefore="帧" style={{ flex: 1 }} />
              <InputNumber value={fps} onChange={(v) => onFpsChange(v || 8)}
                size="small" min={4} max={30} step={1} addonBefore="fps" style={{ flex: 1 }} />
            </div>
            <div style={{ fontSize: 10.5, color: C('color-text-secondary'), marginTop: 6 }}>
              输出为动画 WebP，时长约 {(frames / Math.max(fps, 1)).toFixed(1)} 秒（LTX-Video）
            </div>
          </div>
        )}
      </CollapsibleSection>

      {/* ── ④ 高级参数：种子 / 数量 / LoRA / 系统 ── */}
      <CollapsibleSection
        title="高级参数"
        icon={<SlidersOutlined />}
        defaultOpen={false}
        right={selectedLoras.length > 0 ? (
          <span className="ig-collapse-count">LoRA ×{selectedLoras.length}</span>
        ) : undefined}
      >
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
          {mode !== 't2v' && (
            <div>
              <Typography.Text style={labelStyle}><NumberOutlined />数量</Typography.Text>
              <PickerGroup options={COUNT_OPTIONS} value={count} onChange={onCountChange} columns={4} />
            </div>
          )}
        </div>

        {backend === 'comfyui' && model !== 'diagram' && (
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
            {!engineRunning ? (
              <div style={loraHintStyle}>
                {loraLoading ? '正在启动引擎…' : 'ComfyUI 未运行，启动后自动加载 LoRA'}
              </div>
            ) : loraLoading ? (
              <div style={loraHintStyle}>正在加载 LoRA…</div>
            ) : loraError ? (
              <div style={{ ...loraHintStyle, color: 'var(--color-destructive)', display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ flex: 1 }}>LoRA 加载失败：{loraError}</span>
                {onRefreshLoras && (
                  <Button size="small" type="text" onClick={onRefreshLoras}
                    style={{ color: 'var(--color-destructive)', fontSize: 11, padding: '0 4px' }}>
                    重试
                  </Button>
                )}
              </div>
            ) : loraOptions.length === 0 ? (
              <div style={loraHintStyle}>当前模型 {model} 暂无匹配的 LoRA</div>
            ) : (
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
            )}
          </div>
        )}

        {sysStats && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <Typography.Text style={labelStyle}><DashboardOutlined />系统</Typography.Text>
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
          </div>
        )}
      </CollapsibleSection>
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
