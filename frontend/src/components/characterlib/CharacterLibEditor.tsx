// CharacterLibEditor.tsx — 角色库详情：档案册视图（相卡面板）
// 档案眉 → 立绘横幅 → 身份栏 + 卷宗正文 → 底部操作条
// 随机生成：顶部「随机补全 / 全部随机」，每个字段旁 ↻ 可单独随机（含性格）
import React, { useEffect, useMemo, useState } from 'react'
import {
  Modal, Input, Select, Switch, Slider, Typography, message, Button,
} from 'antd'
import {
  SaveOutlined, CloseOutlined, PictureOutlined, ThunderboltOutlined,
  ExperimentOutlined, RetweetOutlined, LoadingOutlined,
} from '@ant-design/icons'
import TisorRadar from '../TisorRadar'
import {
  saveCharacter, generateFill, generateRandom, generatePortrait,
  type LibraryCharacter,
} from '../../api/characterlib'
import './character-detail.css'

const { Text } = Typography
const { TextArea } = Input

const GENDER_OPTIONS = [
  { value: 'female', label: '女性' },
  { value: 'male', label: '男性' },
  { value: 'neutral', label: '中性其他' },
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

const KIND_META: Record<string, string> = {
  builtin: '内置',
  custom: '自定义',
  assistant: '助手',
}

const ROLE_LABELS: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '龙套',
}

const GENDER_LABELS: Record<string, string> = {
  female: '女性', male: '男性', neutral: '中性',
}

const STATUS_LABELS: Record<string, string> = {
  Alive: '存活', Dead: '已故', Missing: '失踪', Transformed: '变身',
}

// AI 可单独随机的字段 → 中文名（用于按钮 title 与提示）
const FIELD_LABELS: Record<string, string> = {
  personality: '性格',
  background: '背景',
  appearance: '外貌',
  figure: '身材',
  motivation: '动机',
  arc: '角色弧线',
  notes: '备注',
  tags: '标签',
  dialogueSamples: '对话样本',
  voiceGuide: '口吻指南',
  behaviorRules: '行为规则',
  emotionLogic: '情感逻辑',
}

// 本地即时随机的年龄池（无需 AI）
const AGE_POOL = [
  '17', '18', '19', '20', '21', '22', '23', '24', '25', '26',
  '27', '28', '29', '30', '31', '32', '34', '36', '38', '40', '45',
]

const pick = <T,>(arr: readonly T[]): T => arr[Math.floor(Math.random() * arr.length)]

interface Props {
  open: boolean
  character: LibraryCharacter | null
  projects: string[]
  onClose: () => void
  onSaved: (c: LibraryCharacter) => void
  isCurrentPersona?: boolean
  index?: number
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

const CharacterLibEditor: React.FC<Props> = ({
  open, character, projects, onClose, onSaved,
  isCurrentPersona = false, index = 0,
}) => {
  const [form, setForm] = useState<Partial<LibraryCharacter>>(() => toForm(character))
  const [saving, setSaving] = useState(false)
  const [filling, setFilling] = useState(false)
  const [genAll, setGenAll] = useState(false)
  const [fieldGen, setFieldGen] = useState<string | null>(null)
  const [genPortrait, setGenPortrait] = useState(false)

  useEffect(() => {
    if (open) setForm(toForm(character))
  }, [open, character])

  const isNew = !character
  const kindLabel = character?.kind ? KIND_META[character.kind] || KIND_META.custom : KIND_META.custom
  const busy = saving || filling || genAll || !!fieldGen

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

  const handleFill = async () => {
    if (busy) return
    if (!form.name?.trim()) {
      message.warning('角色名称不能为空')
      return
    }
    setFilling(true)
    try {
      const filled = await generateFill(form)
      let filledCount = 0
      const textKeys = ['roleType', 'gender', 'age', 'personality', 'appearance', 'figure',
        'background', 'motivation', 'arc', 'status', 'notes', 'behaviorRules', 'emotionLogic'] as const
      for (const k of textKeys) {
        const f = form[k] as string | undefined
        const g = filled[k] as string | undefined
        if (!f?.trim() && g?.trim()) filledCount++
      }
      if (!(form.tags ?? []).length && (filled.tags ?? []).length) filledCount++
      setForm(filled as Partial<LibraryCharacter>)
      message.success(filledCount > 0
        ? `已补齐 ${filledCount} 处空缺，检查后保存`
        : '没有空缺字段需要补齐')
    } catch (err: any) {
      message.error(`补齐失败：${err?.message || String(err)}`)
    } finally {
      setFilling(false)
    }
  }

  const handleRandomAll = async () => {
    if (busy) return
    if (!form.name?.trim()) {
      message.warning('角色名称不能为空')
      return
    }
    setGenAll(true)
    try {
      const next = await generateRandom(form, 'all')
      setForm(next as Partial<LibraryCharacter>)
      message.success('已重新随机全部设定（含性格，姓名不变），检查后保存')
    } catch (err: any) {
      message.error(`全部随机失败：${err?.message || String(err)}`)
    } finally {
      setGenAll(false)
    }
  }

  const randomizeDims = () => {
    if (busy) return
    const next = {} as LibraryCharacter['dims']
    ;(['T', 'I', 'S', 'O', 'R'] as const).forEach(k => {
      next[k] = 25 + Math.floor(Math.random() * 66)
    })
    patch({ dims: next })
    message.success('已随机五维数值')
  }

  const randomizeField = async (key: string) => {
    if (busy) return
    if (!form.name?.trim()) {
      message.warning('角色名称不能为空')
      return
    }
    // 枚举字段本地即时随机（无需 AI）
    if (key === 'gender') {
      patch({ gender: pick(GENDER_OPTIONS).value })
      message.success('已随机性别')
      return
    }
    if (key === 'roleType') {
      patch({ roleType: pick(ROLE_OPTIONS).value })
      message.success('已随机定位')
      return
    }
    if (key === 'status') {
      patch({ status: pick(STATUS_OPTIONS).value })
      message.success('已随机状态')
      return
    }
    if (key === 'age') {
      patch({ age: pick(AGE_POOL) })
      message.success('已随机年龄')
      return
    }
    // 文本字段走 AI 按字段随机
    const label = FIELD_LABELS[key] || key
    setFieldGen(key)
    try {
      const next = await generateRandom(form, key)
      setForm(next as Partial<LibraryCharacter>)
      message.success(`已重新随机${label}，检查后保存`)
    } catch (err: any) {
      message.error(`随机${label}失败：${err?.message || String(err)}`)
    } finally {
      setFieldGen(null)
    }
  }

  const handleGeneratePortrait = async () => {
    if (busy) return
    if (!form.name?.trim()) {
      message.warning('角色名称不能为空')
      return
    }
    setGenPortrait(true)
    try {
      const img = await generatePortrait(form)
      patch({ portraitUrl: img })
      message.success('剧照已生成，检查后保存')
    } catch (err: any) {
      message.error(`剧照生成失败：${err?.message || String(err)}`)
    } finally {
      setGenPortrait(false)
    }
  }

  const dims = useMemo(() => form.dims ?? emptyDims(), [form.dims])
  const heroMeta = [
    form.roleType ? ROLE_LABELS[form.roleType] || form.roleType : '',
    form.gender ? GENDER_LABELS[form.gender] || form.gender : '',
    form.age,
    form.status ? STATUS_LABELS[form.status] || form.status : '',
  ].filter(Boolean).join(' · ')

  const fieldLabel = (t: string, diceKey?: string) => (
    <div className="cd-label-row">
      <label className="cd-label">{t}</label>
      {diceKey && (
        <Button
          size="small"
          type="text"
          className="cd-dice"
          icon={fieldGen === diceKey ? <LoadingOutlined /> : <RetweetOutlined />}
          loading={fieldGen === diceKey}
          disabled={busy && fieldGen !== diceKey}
          title={`随机生成${FIELD_LABELS[diceKey] || t}`}
          onClick={() => randomizeField(diceKey)}
        />
      )}
    </div>
  )

  const section = (title: string, children: React.ReactNode, action?: React.ReactNode) => (
    <section className="cd-sec">
      <h4 className="cd-sec-title">{title}{action}</h4>
      {children}
    </section>
  )

  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      closable={false}
      width={900}
      className="cd-modal"
      styles={{
        content: { padding: 0, borderRadius: 16, overflow: 'hidden', background: 'var(--md-sys-color-surface-container)' },
        body: { padding: 0 },
      }}
    >
      <div className="cd">
        <div className="cd-scroll">
          {/* 档案眉 */}
          <div className="cd-head">
            <span className="cd-no">
              {isNew ? '新建档案' : `角色档案 · NO.${String(index + 1).padStart(3, '0')}`}
            </span>
            <div className="cd-head-right">
              {!isNew && <span className="cd-kind">{kindLabel}</span>}
              {form.chatEnabled && (
                <span className="cd-chat"><i className="cd-chat-dot" />可聊天</span>
              )}
              {isCurrentPersona && <span className="cd-current">当前人格</span>}
              <Button type="text" size="small" icon={<CloseOutlined />} title="关闭" onClick={onClose} className="cd-close" />
            </div>
          </div>

          <div className="cd-body">
            {/* 身份栏 */}
            <aside className="cd-rail">
              {/* 立绘档案卡（竖向 3:4，剧照完整可见） */}
              <div className="cd-hero cd-sheen">
                {form.portraitUrl ? (
                  <img className="cd-hero-img" src={form.portraitUrl} alt={form.name || '角色立绘'} />
                ) : (
                  <div className="cd-hero-ph">{form.name?.slice(0, 1) || '?'}</div>
                )}
                <div className="cd-hero-shade" />
                <Button
                  size="small"
                  icon={<PictureOutlined />}
                  loading={genPortrait}
                  disabled={busy}
                  onClick={handleGeneratePortrait}
                  className="cd-hero-gen"
                  title="按角色设定生成剧照"
                >
                  生成剧照
                </Button>
                <div className="cd-hero-info">
                  <h2 className="cd-hero-name">{form.name || '未命名角色'}</h2>
                  {heroMeta && <p className="cd-hero-meta">{heroMeta}</p>}
                </div>
              </div>

              <div className="cd-field">
                {fieldLabel('名称 *')}
                <Input size="small" value={form.name ?? ''} onChange={e => patch({ name: e.target.value })}
                  placeholder="角色名" />
              </div>
              <div className="cd-field">
                {fieldLabel('立绘 URL')}
                <Input size="small" value={form.portraitUrl ?? ''}
                  onChange={e => patch({ portraitUrl: e.target.value })}
                  placeholder="https://... 或 data:image/..." />
                {form.portraitUrl && (
                  <img className="cd-thumb" src={form.portraitUrl} alt="立绘预览" />
                )}
              </div>
              <div className="cd-grid2">
                <div className="cd-field">
                  {fieldLabel('性别', 'gender')}
                  <Select size="small" value={form.gender ?? ''} style={{ width: '100%' }}
                    onChange={v => patch({ gender: v })} options={GENDER_OPTIONS} allowClear />
                </div>
                <div className="cd-field">
                  {fieldLabel('年龄', 'age')}
                  <Input size="small" value={form.age ?? ''} onChange={e => patch({ age: e.target.value })}
                    placeholder="23 / 外观二十五六" />
                </div>
                <div className="cd-field">
                  {fieldLabel('定位', 'roleType')}
                  <Select size="small" value={form.roleType ?? ''} style={{ width: '100%' }}
                    onChange={v => patch({ roleType: v })} options={ROLE_OPTIONS} allowClear />
                </div>
                <div className="cd-field">
                  {fieldLabel('状态', 'status')}
                  <Select size="small" value={form.status ?? ''} style={{ width: '100%' }}
                    onChange={v => patch({ status: v })} options={STATUS_OPTIONS} allowClear />
                </div>
              </div>
              <div className="cd-field">
                {fieldLabel('标签', 'tags')}
                <Input size="small" value={(form.tags ?? []).join('，')} onChange={e => patchTags(e.target.value)}
                  placeholder="女主，剑修" />
                {(form.tags ?? []).length > 0 && (
                  <div className="cd-tags">
                    {(form.tags ?? []).slice(0, 6).map(t => <span key={t} className="cd-tag">#{t}</span>)}
                  </div>
                )}
              </div>
              <div className="cd-radar">
                <div className="cd-radar-head">
                  <Text className="cd-radar-title">五维人格</Text>
                  <Button size="small" type="text" className="cd-dice" icon={<RetweetOutlined />}
                    title="随机五维数值" disabled={busy} onClick={randomizeDims} />
                </div>
                <TisorRadar dims={dims} size={132} color="var(--gaea-glow)" />
                <div className="cd-dims">
                  {DIM_META.map(d => (
                    <div key={d.key} className="cd-dim">
                      <Text className="cd-dim-label" title={d.desc}>{d.label}</Text>
                      <Slider min={0} max={100} value={dims[d.key] ?? 50}
                        onChange={v => patchDims(d.key, Number(v))} tooltip={{ formatter: (v?: number) => `${v}` }} />
                      <Text className="cd-dim-val">{dims[d.key] ?? 50}</Text>
                    </div>
                  ))}
                </div>
              </div>
              <div className="cd-chat-toggle">
                <div>
                  <Text className="cd-toggle-title">可聊天</Text>
                  <div className="cd-toggle-desc">出现在聊天板块的人格列表</div>
                </div>
                <Switch size="small" checked={!!form.chatEnabled} onChange={v => patch({ chatEnabled: v })} />
              </div>
            </aside>

            {/* 卷宗正文 */}
            <main className="cd-main">
              {/* 随机生成工具条 */}
              <div className="cd-genbar">
                <span className="cd-genbar-title">随机生成</span>
                <Button size="small" icon={<ThunderboltOutlined />} loading={filling}
                  disabled={busy && !filling} onClick={handleFill}
                  title="仅补齐空缺字段，保留已有内容">随机补齐</Button>
                <Button size="small" icon={<ExperimentOutlined />} loading={genAll}
                  disabled={busy && !genAll} onClick={handleRandomAll}
                  title="重新随机全部设定（含性格），姓名不变">全部随机</Button>
                <span className="cd-genbar-hint">字段旁 ↻ 可单独随机</span>
              </div>

              {section('小说设定', (
                <div className="cd-fields">
                  {[
                    { key: 'personality' as const, label: '性格', rows: 2 },
                    { key: 'background' as const, label: '背景', rows: 2 },
                    { key: 'appearance' as const, label: '外貌', rows: 2 },
                    { key: 'figure' as const, label: '身材体型', rows: 1 },
                    { key: 'motivation' as const, label: '动机', rows: 2 },
                    { key: 'arc' as const, label: '角色弧线', rows: 2 },
                  ].map(f => (
                    <div key={f.key} className="cd-field">
                      {fieldLabel(f.label, f.key)}
                      <TextArea className="cd-area" size="small" rows={f.rows} value={form[f.key] ?? ''}
                        onChange={e => patch({ [f.key]: e.target.value } as any)} />
                    </div>
                  ))}
                </div>
              ))}
              {section('对话样本', (
                <div className="cd-field">
                  {fieldLabel('每行一条，供 AI 说话节拍', 'dialogueSamples')}
                  <TextArea className="cd-area" size="small" rows={3} value={(form.dialogueSamples ?? []).join('\n')}
                    onChange={e => patchSamples(e.target.value)}
                    placeholder={'「你……你才不是为我来的吧。」\n「剑修不问红尘。」'} />
                </div>
              ))}
              {section('备注', (
                <div className="cd-field">
                  <TextArea className="cd-area" size="small" rows={2} value={form.notes ?? ''}
                    onChange={e => patch({ notes: e.target.value })} />
                </div>
              ), (
                <Button size="small" type="text" className="cd-dice" icon={<RetweetOutlined />}
                  title="随机生成备注" disabled={busy} onClick={() => randomizeField('notes')} />
              ))}
              {section('聊天设定', (
                <div className="cd-fields">
                  {[
                    { key: 'voiceGuide' as const, label: '口吻指南', rows: 3, ph: '角色怎么说话：语气、用词、节奏…' },
                    { key: 'behaviorRules' as const, label: '行为规则', rows: 2, ph: '互动中的行为边界与习惯' },
                    { key: 'emotionLogic' as const, label: '情感逻辑', rows: 2, ph: '对用户的情绪反应模式' },
                  ].map(f => (
                    <div key={f.key} className="cd-field">
                      {fieldLabel(f.label, f.key)}
                      <TextArea className="cd-area" size="small" rows={f.rows} value={form[f.key] ?? ''} placeholder={f.ph}
                        onChange={e => patch({ [f.key]: e.target.value } as any)} />
                    </div>
                  ))}
                </div>
              ))}
              {form.assistantId && (
                <div className="cd-note">
                  聊天通道：assistantId={form.assistantId}（微信通道配置以助手记录为准）
                </div>
              )}
            </main>
          </div>
        </div>

        {/* 底部操作条 */}
        <div className="cd-foot">
          <div className="cd-foot-meta">
            {projects.length > 0 && (
              <span>被 {projects.length} 个项目引用：{projects.join('、')}</span>
            )}
          </div>
          <div className="cd-foot-actions">
            <Button icon={<CloseOutlined />} onClick={onClose}>取消</Button>
            <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSave} className="cd-save">
              保存
            </Button>
          </div>
        </div>
      </div>
    </Modal>
  )
}

export default CharacterLibEditor
