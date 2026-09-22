// InstructionEditModal.tsx — 指令编辑弹窗（绘梦阶段一刀 C，规格
// 进度计划/gaea-instruct-edit-20260917.md；阶段二刀 A 补 ComfyUI 本地档，规格
// 进度计划/gaea-comfyui-edit-20260922.md；阶段二刀 B 补蒙版局部重绘，规格
// 进度计划/gaea-mask-inpaint-20260923.md；阶段二刀 C 补扩图，规格
// 进度计划/gaea-outpaint-20260923.md）：原图 + 人话指令 → 语义改图
// （云端 OpenAI 兼容 /images/edits；本地 ComfyUI Qwen-Image-Edit 2511 官方工作流）；
// 范围可切「全图 / 局部（涂选）/ 扩图（加画布）」。与既有「改图」（=把结果填回
// 图生图整幅重绘）互补。对照区原图|新图并排；「用到画布」把编辑结果并入 results/
// history（后端已落盘+登记台账，走既有保存/溯源链）。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useState } from 'react'
import { Alert, Button, Input, InputNumber, Modal, Radio, Spin, Typography, message } from 'antd'
import { generateMedia } from '../../api/image'
import MaskBrushEditor from './MaskBrushEditor'
import type { GenResult } from './types'

type ScopeMode = 'full' | 'partial' | 'outpaint'
type ExpandEdges = { left: number; top: number; right: number; bottom: number }

// 扩图预设（阶段二刀 C）：chips 一键写入四边扩展量；补正方由前端按源图宽高比算。
const OUTPAINT_PRESETS: Array<{ key: string; label: string; edges: ExpandEdges | 'square' }> = [
  { key: 'left', label: '左 50%', edges: { left: 50, top: 0, right: 0, bottom: 0 } },
  { key: 'right', label: '右 50%', edges: { left: 0, top: 0, right: 50, bottom: 0 } },
  { key: 'top', label: '上 50%', edges: { left: 0, top: 50, right: 0, bottom: 0 } },
  { key: 'bottom', label: '下 50%', edges: { left: 0, top: 0, right: 0, bottom: 50 } },
  { key: 'all25', label: '四边 25%', edges: { left: 25, top: 25, right: 25, bottom: 25 } },
  { key: 'square', label: '补正方', edges: 'square' },
]


export default function InstructionEditModal({ open, source, onClose, onApply }: {
  open: boolean
  /** 编辑源图（结果卡点「指令编辑」的那张）。 */
  source: GenResult | null
  onClose: () => void
  /** 「用到画布」：编辑结果并入画布与历史（页面级镜像 queue 成功路径）。 */
  onApply: (r: GenResult) => void
}) {
  const [instruction, setInstruction] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [edited, setEdited] = useState<GenResult | null>(null)
  const [scope, setScope] = useState<ScopeMode>('full')
  const [mask, setMask] = useState<string | null>(null)
  const [expand, setExpand] = useState<ExpandEdges>({ left: 0, top: 0, right: 0, bottom: 0 })

  const expandTotal = expand.left + expand.top + expand.right + expand.bottom

  // 补正方：按源图自然宽高比把短边补齐（长边不动、短边均分差额）
  const edgesToSquare = (): ExpandEdges => {
    const img = new Image()
    img.src = source?.image ?? ''
    const w = img.naturalWidth || 1
    const h = img.naturalHeight || 1
    if (w === h) return { left: 0, top: 0, right: 0, bottom: 0 }
    if (w > h) {
      const pct = Math.round(((w - h) / h) * 100)
      return { left: 0, top: Math.floor(pct / 2), right: 0, bottom: pct - Math.floor(pct / 2) }
    }
    const pct = Math.round(((h - w) / w) * 100)
    return { left: Math.floor(pct / 2), top: 0, right: pct - Math.floor(pct / 2), bottom: 0 }
  }

  const applyPreset = (edges: ExpandEdges | 'square') => {
    setExpand(edges === 'square' ? edgesToSquare() : edges)
  }

  const run = async () => {
    if (!source || !instruction.trim()) {
      message.warning('请输入编辑指令（如：把外套改成红色）')
      return
    }
    if (scope === 'partial' && !mask) {
      message.warning('局部模式请先在图上涂抹要修改的区域')
      return
    }
    if (scope === 'outpaint' && expandTotal <= 0) {
      message.warning('扩图模式请选择扩展方向或输入扩展量（至少一边大于 0）')
      return
    }
    setBusy(true)
    setError('')
    try {
      const res = await generateMedia({
        prompt: instruction.trim(),
        negative: '',
        size: source.size || '',
        model: source.model || '',
        seed: 0,
        lora: '',
        count: 1,
        mode: scope === 'outpaint' ? 'outpaint' : 'edit',
        initImage: source.image,
        mask: scope === 'partial' ? (mask ?? undefined) : undefined,
        expand: scope === 'outpaint' ? expand : undefined,
      })
      if (res.error) {
        setError(res.error)
        return
      }
      const first = res.results?.[0]
      if (!first) {
        setError('编辑返回空结果')
        return
      }
      setEdited({ ...first, prompt: `【编辑】${instruction.trim()}` })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const reset = () => {
    setInstruction('')
    setEdited(null)
    setError('')
    setBusy(false)
    setScope('full')
    setMask(null)
    setExpand({ left: 0, top: 0, right: 0, bottom: 0 })
  }

  const imgBox: React.CSSProperties = {
    flex: 1, minWidth: 0, borderRadius: 8, overflow: 'hidden',
    background: 'rgba(0,0,0,0.04)', display: 'flex', alignItems: 'center', justifyContent: 'center',
  }

  return (
    <Modal
      open={open} title="指令编辑（原图 + 一句话指令 → 语义改图）" width={720} destroyOnHidden
      onCancel={() => { reset(); onClose() }}
      footer={[
        <Button key="run" size="small" type="primary" loading={busy} data-testid="instruct-edit-run"
          onClick={() => void run()}>生成编辑</Button>,
        edited && (
          <Button key="apply" size="small" data-testid="instruct-edit-apply"
            onClick={() => { if (edited) { onApply(edited); reset(); onClose() } }}>用到画布</Button>
        ),
        <Button key="close" size="small" onClick={() => { reset(); onClose() }}>关闭</Button>,
      ].filter(Boolean)}
    >
      {!source ? (
        <Typography.Text type="secondary">未选择源图</Typography.Text>
      ) : (
        <div data-testid="instruct-edit-body" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {/* 指令输入 */}
          <div>
            <div style={{ ...softTextStyle, marginBottom: 4 }}>编辑指令（描述要改什么，其余保持原样）</div>
            <Input.TextArea
              data-testid="instruct-edit-input" rows={2} value={instruction} maxLength={600}
              placeholder="例如：把外套改成红色；移除背景里的路人；让她看向镜头"
              onChange={e => setInstruction(e.target.value)} />
          </div>
          {/* 范围切换（阶段二刀 B/C）：全图 / 局部（蒙版涂选）/ 扩图（加画布） */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>范围</Typography.Text>
            <Radio.Group
              size="small" value={scope} data-testid="instruct-edit-scope"
              onChange={e => setScope(e.target.value as ScopeMode)}>
              <Radio.Button value="full">全图</Radio.Button>
              <Radio.Button value="partial">局部（涂选）</Radio.Button>
              <Radio.Button value="outpaint">扩图</Radio.Button>
            </Radio.Group>
          </div>
          {scope === 'partial' && (
            <MaskBrushEditor src={source.image} onMaskChange={setMask} />
          )}
          {scope === 'outpaint' && (
            <div data-testid="outpaint-panel" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {/* 预览：原图 + 按扩展比例的虚线扩边框（padding 按百分比换算展示宽高） */}
              <div data-testid="outpaint-preview" style={{
                alignSelf: 'flex-start',
                border: '1px dashed rgba(127,127,127,0.6)', borderRadius: 6, padding: 2,
                paddingLeft: 2 + expand.left * 0.3, paddingTop: 2 + expand.top * 0.3,
                paddingRight: 2 + expand.right * 0.3, paddingBottom: 2 + expand.bottom * 0.3,
                background: 'repeating-linear-gradient(45deg, rgba(127,127,127,0.06) 0 6px, transparent 6px 12px)',
              }}>
                <img src={source.image} alt="扩图底图" style={{ maxWidth: 160, maxHeight: 120, objectFit: 'contain', display: 'block', borderRadius: 4 }} />
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                {OUTPAINT_PRESETS.map(p => (
                  <Button key={p.key} size="small" data-testid={`outpaint-preset-${p.key}`} onClick={() => applyPreset(p.edges)}>{p.label}</Button>
                ))}
              </div>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
                {(['left', 'top', 'right', 'bottom'] as const).map(k => (
                  <label key={k} style={{ ...softTextStyle, fontSize: 12, display: 'flex', alignItems: 'center', gap: 4 }}>
                    {{ left: '左', top: '上', right: '右', bottom: '下' }[k]}
                    <InputNumber
                      size="small" min={0} max={200} value={expand[k]} style={{ width: 72 }}
                      data-testid={`outpaint-${k}`}
                      onChange={v => setExpand(prev => ({ ...prev, [k]: Math.max(0, Math.min(200, Math.round(Number(v) || 0))) }))} />
                    %
                  </label>
                ))}
                <Typography.Text type="secondary" style={{ fontSize: 12 }} data-testid="outpaint-total">
                  {expandTotal > 0 ? `扩展 ${expandTotal}%（新增区域按指令补全，原图保持）` : '选择方向或输入扩展量'}
                </Typography.Text>
              </div>
            </div>
          )}
          {/* 对照区：原图 | 新图 */}
          <div style={{ display: 'flex', gap: 10, alignItems: 'stretch', minHeight: 220 }}>
            <div style={imgBox}>
              <img src={source.image} alt="原图" style={{ maxWidth: '100%', maxHeight: 300, objectFit: 'contain' }} />
            </div>
            <div style={{ ...imgBox, border: edited ? '1px dashed rgba(0,0,0,0.18)' : undefined }}>
              {busy ? (
                <div style={{ padding: 24, textAlign: 'center' }}><Spin /><div style={{ ...softTextStyle, marginTop: 6 }}>正在按指令编辑…</div></div>
              ) : edited ? (
                <img src={edited.image} alt="编辑结果" data-testid="instruct-edit-result"
                  style={{ maxWidth: '100%', maxHeight: 300, objectFit: 'contain' }} />
              ) : (
                <div style={{ ...softTextStyle, padding: 24, textAlign: 'center' }}>编辑结果会出现在这里</div>
              )}
            </div>
          </div>
          {edited && (
            <div style={softTextStyle} data-testid="instruct-edit-meta">
              模型 {edited.model || '—'} · {edited.time ?? '—'}s{edited.size ? ` · ${edited.size}` : ''}
            </div>
          )}
          {error && <Alert type="error" showIcon data-testid="instruct-edit-error" message={error} />}
          <Typography.Text type="secondary" style={{ fontSize: 12 }} data-testid="instruct-edit-hint">
            云端走 OpenAI 兼容引擎的 /images/edits（如 Qwen-Image-Edit 系）；本地 ComfyUI 走 Qwen-Image-Edit 2511 官方工作流（需在 ComfyUI models 目录放置模型文件，缺失时错误会列出所需文件与下载地址）；GLM 档暂不支持（会如实报错）。局部=只重绘涂红区域；扩图=画布加边、原图保持。
          </Typography.Text>
        </div>
      )}
    </Modal>
  )
}
