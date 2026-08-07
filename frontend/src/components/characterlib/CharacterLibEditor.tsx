import React, { useEffect, useMemo, useState } from 'react'
import {
  Modal, Input, Select, Switch, Slider, Tag, Typography, message, Space, Divider,
} from 'antd'
import { SaveOutlined, CloseOutlined } from '@ant-design/icons'
import TisorRadar from '../TisorRadar'
import { C } from '../../utils/theme'
import { saveCharacter, type LibraryCharacter } from '../../api/characterlib'

const { Text } = Typography
const { TextArea } = Input

const GENDER_OPTIONS = [
  { value: 'female', label: '女性' },
  { value: 'male', label: '男性' },
  { value: 'neutral', label: '中性/其他' },
]

const ROLE_OPTIONS = [
  { value: 'protagonist', label: '主角' },
  { value: 'antagonist', label: '反派' },
  { value: 'supporting', label: '配角' },
  { value: 'minor', label: '龙套' },
]

const STATUS_OPTIONS = [
  { value: 'Alive', label: '存活' },
  { value: 'Dead', label: '已故' },
  { value: 'Missing', label: '失踪' },
  { value: 'Transformed', label: '变身' },
]

const DIM_META: { key: keyof LibraryCharacter['dims']; label: string; desc: string }[] = [
  { key: 'T', label: 'T 温柔', desc: '体贴与温度' },
  { key: 'I', label: 'I 主动', desc: '发起与推进' },
  { key: 'S', label: 'S 顺从', desc: '配合与依赖' },
  { key: 'O', label: 'O 独特', desc: '个性与表达' },
  { key: 'R', label: 'R 矜持', desc: '克制与距离' },
]

interface Props {
  open: boolean
  character: LibraryCharacter | null
  projects: string[]
  onClose: () => void
  onSaved: (c: LibraryCharacter) => void
}

function emptyDims() {
  return { T: 50, I: 50, S: 50, O: 50, R: 50 }
}

function toForm(c: LibraryCharacter | null): Partial<LibraryCharacter> {
  if (!c) return { tags: [], dialogueSamples: [], dims: emptyDims() }
  return {
    ...c,
    tags: c.tags ?? [],
    dialogueSamples: c.dialogueSamples ?? [],
    dims: c.dims ?? emptyDims(),
  }
}

const CharacterLibEditor: React.FC<Props> = ({ open, character, projects, onClose, onSaved }) => {
  const [form, setForm] = useState<Partial<LibraryCharacter>>(() => toForm(character))
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (open) setForm(toForm(character))
  }, [open, character])

  const isNew = !character
  const kindLabel = character?.kind === 'builtin' ? '内置人格' : character?.kind === 'assistant' ? '助手角色' : '自定义角色'

  const patch = (p: Partial<LibraryCharacter>) => setForm(prev => ({ ...prev, ...p }))
  const patchTags = (v: string) => patch({ tags: v.split(/[,，]/).map(s => s.trim()).filter(Boolean) })
  const patchSamples = (v: string) => patch({ dialogueSamples: v.split('\n').map(s => s.trim()).filter(Boolean) })
  const patchDims = (key: keyof LibraryCharacter['dims'], v: number) => {
    setForm(prev => ({ ...prev, dims: { ...(prev.dims ?? emptyDims()), [key]: v } }))
  }

  const handleSave = async () => {
    if (!form.name?.trim()) {
      message.warning('角色名称不能为空')
      return
    }
    setSaving(true)
    try {
      const saved = await saveCharacter({
        ...form,
        name: form.name.trim(),
        kind: form.kind || 'custom',
      })
      message.success(isNew ? '角色已创建' : '角色已保存')
      onSaved(saved)
      onClose()
    } catch (err: any) {
      message.error(`保存失败：${err?.message || String(err)}`)
    } finally {
      setSaving(false)
    }
  }

  const dims = useMemo(() => form.dims ?? emptyDims(), [form.dims])

  const sectionTitle = (t: string) => (
    <Text strong style={{ color: C('color-text'), fontSize: 12.5, display: 'block', marginBottom: 8 }}>
      {t}
    </Text>
  )

  return (
    <Modal
      open={open}
      onCancel={onClose}
      onOk={handleSave}
      okText="保存"
      cancelText="取消"
      okButtonProps={{ icon: <SaveOutlined />, loading: saving }}
      cancelButtonProps={{ icon: <CloseOutlined /> }}
      width={760}
      title={
        <Space>
          <span style={{ color: C('color-text') }}>{isNew ? '新建统一角色' : `编辑角色 · ${character?.name}`}</span>
          {!isNew && <Tag color={character?.kind === 'builtin' ? 'gold' : character?.kind === 'assistant' ? 'geekblue' : 'green'} style={{ margin: 0 }}>{kindLabel}</Tag>}
        </Space>
      }
      styles={{ body: { maxHeight: '68vh', overflowY: 'auto', paddingRight: 8 } }}
    >
      {/* ── 基础信息 ── */}
      {sectionTitle('基础信息')}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, marginBottom: 12 }}>
        <div>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>名称 *</Text>
          <Input size="small" value={form.name ?? ''} onChange={e => patch({ name: e.target.value })}
            placeholder="角色名" style={{ marginTop: 4 }} />
        </div>
        <div>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>性别</Text>
          <Select size="small" value={form.gender ?? ''} style={{ width: '100%', marginTop: 4 }}
            onChange={v => patch({ gender: v })} options={GENDER_OPTIONS} allowClear />
        </div>
        <div>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>年龄</Text>
          <Input size="small" value={form.age ?? ''} onChange={e => patch({ age: e.target.value })}
            placeholder="如：23 / 外观二十五六" style={{ marginTop: 4 }} />
        </div>
        <div>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>标签（逗号分隔）</Text>
          <Input size="small" value={(form.tags ?? []).join('，')} onChange={e => patchTags(e.target.value)}
            placeholder="女主，剑修" style={{ marginTop: 4 }} />
        </div>
      </div>
      <div style={{ marginBottom: 12 }}>
        <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>剧照 URL</Text>
        <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
          <Input size="small" value={form.portraitUrl ?? ''} onChange={e => patch({ portraitUrl: e.target.value })}
            placeholder="https://... 或 data:image/..." />
          {form.portraitUrl && (
            <img src={form.portraitUrl} alt="角色剧照" style={{ width: 44, height: 44, borderRadius: 8, objectFit: 'cover', flexShrink: 0 }} />
          )}
        </div>
      </div>

      <Divider style={{ margin: '12px 0' }} />

      {/* ── 小说设定 ── */}
      {sectionTitle('小说设定')}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, marginBottom: 10 }}>
        <div>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>定位</Text>
          <Select size="small" value={form.roleType ?? ''} style={{ width: '100%', marginTop: 4 }}
            onChange={v => patch({ roleType: v })} options={ROLE_OPTIONS} allowClear />
        </div>
        <div>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>状态</Text>
          <Select size="small" value={form.status ?? ''} style={{ width: '100%', marginTop: 4 }}
            onChange={v => patch({ status: v })} options={STATUS_OPTIONS} allowClear />
        </div>
      </div>
      {[
        { key: 'personality' as const, label: '性格', rows: 2 },
        { key: 'background' as const, label: '背景', rows: 2 },
        { key: 'appearance' as const, label: '外貌', rows: 2 },
        { key: 'figure' as const, label: '身材体型', rows: 1 },
        { key: 'motivation' as const, label: '动机', rows: 2 },
        { key: 'arc' as const, label: '角色弧线', rows: 2 },
      ].map(f => (
        <div key={f.key} style={{ marginBottom: 8 }}>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>{f.label}</Text>
          <TextArea size="small" rows={f.rows} value={form[f.key] ?? ''}
            onChange={e => patch({ [f.key]: e.target.value } as any)}
            style={{ marginTop: 4, fontSize: 12.5, background: 'rgba(0,0,0,0.18)', border: '1px solid var(--border-subtle)', color: C('color-text') }} />
        </div>
      ))}
      <div style={{ marginBottom: 8 }}>
        <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>备注</Text>
        <TextArea size="small" rows={2} value={form.notes ?? ''}
          onChange={e => patch({ notes: e.target.value })}
          style={{ marginTop: 4, fontSize: 12.5, background: 'rgba(0,0,0,0.18)', border: '1px solid var(--border-subtle)', color: C('color-text') }} />
      </div>
      <div style={{ marginBottom: 8 }}>
        <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>对话样本（每行一条，教 AI 说话节奏）</Text>
        <TextArea size="small" rows={3} value={(form.dialogueSamples ?? []).join('\n')}
          onChange={e => patchSamples(e.target.value)}
          placeholder={'“你…你才不是为我来的吧？”\n“剑修不问红尘。”'}
          style={{ marginTop: 4, fontSize: 12.5, background: 'rgba(0,0,0,0.18)', border: '1px solid var(--border-subtle)', color: C('color-text') }} />
      </div>

      <Divider style={{ margin: '12px 0' }} />

      {/* ── 聊天设定 ── */}
      {sectionTitle('聊天设定')}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
        <div>
          <Text style={{ fontSize: 12, color: C('color-text') }}>可聊天</Text>
          <div style={{ fontSize: 10, color: C('color-text-secondary') }}>开启后出现在聊天板块的人格列表，可直接对话</div>
        </div>
        <Switch size="small" checked={!!form.chatEnabled} onChange={v => patch({ chatEnabled: v })} />
      </div>
      <div style={{ display: 'flex', gap: 14, marginBottom: 10 }}>
        <div style={{ flexShrink: 0 }}>
          <TisorRadar dims={dims} size={110} color="#f472b6" />
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 5 }}>
          {DIM_META.map(d => (
            <div key={d.key} style={{ display: 'grid', gridTemplateColumns: '58px 1fr 26px', alignItems: 'center', gap: 6 }}>
              <Text style={{ fontSize: 10.5, color: C('color-text-secondary') }}>{d.label}</Text>
              <Slider min={0} max={100} value={dims[d.key] ?? 50}
                onChange={v => patchDims(d.key, Number(v))} tooltip={{ formatter: (v?: number) => `${v}` }} />
              <Text style={{ fontSize: 10.5, color: C('color-text'), textAlign: 'right' }}>{dims[d.key] ?? 50}</Text>
            </div>
          ))}
        </div>
      </div>
      {[
        { key: 'voiceGuide' as const, label: '口吻指南', rows: 3, ph: '角色怎么说话：语气、用词、节奏…' },
        { key: 'behaviorRules' as const, label: '行为规则', rows: 2, ph: '互动中的行为边界与习惯' },
        { key: 'emotionLogic' as const, label: '情感逻辑', rows: 2, ph: '对用户的情感反应模式' },
      ].map(f => (
        <div key={f.key} style={{ marginBottom: 8 }}>
          <Text style={{ fontSize: 11, color: C('color-text-secondary') }}>{f.label}</Text>
          <TextArea size="small" rows={f.rows} value={form[f.key] ?? ''} placeholder={f.ph}
            onChange={e => patch({ [f.key]: e.target.value } as any)}
            style={{ marginTop: 4, fontSize: 12.5, background: 'rgba(0,0,0,0.18)', border: '1px solid var(--border-subtle)', color: C('color-text') }} />
        </div>
      ))}

      {form.assistantId && (
        <div style={{ marginTop: 10, fontSize: 10.5, color: C('color-text-secondary') }}>
          聊天通道：assistantId={form.assistantId}（微信/通道配置以助手记录为准）
        </div>
      )}

      {projects.length > 0 && (
        <div style={{ marginTop: 10, fontSize: 10.5, color: C('color-text-secondary') }}>
          被 {projects.length} 个项目引用：{projects.join('、')}
        </div>
      )}
    </Modal>
  )
}

export default CharacterLibEditor
