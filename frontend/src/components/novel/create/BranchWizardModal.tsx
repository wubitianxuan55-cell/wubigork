import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Card, Input, message, Modal, Select, Space, Spin, Typography } from 'antd'
import { BulbOutlined, PlusOutlined } from '@ant-design/icons'
import { C } from '../../../utils/theme'

export interface Branch { title: string; pitch: string }

/** 选角会议单条 cast：name 必填；relation=与主角关系；note=来源标注（库卡摘要）；isNew=本次新建 */
export interface BranchCastEntry { name: string; relation?: string; note?: string; isNew?: boolean }

interface BranchWizardModalProps {
  open: boolean
  prevChapter: number
  overwriteChapter: number
  branchFromID: string
  /** 项目角色名册（选角候选） */
  characters: string[]
  /** 角色库候选（note=一句话档案摘要） */
  libraryCharacters: { name: string; note?: string }[]
  /** 上一章出场角色（「延续上一章」一键带入；空=隐藏按钮） */
  prevChapterCast: string[]
  onClose: () => void
  /** 拉取 AI 构思的分支列表（cast=选角会议结果，空=全量名册不设限） */
  onFetchBranches: (prevChapter: number, cast: BranchCastEntry[]) => Promise<Branch[]>
  /** 确认生成：plotReq 由分支标题或用户输入组装 */
  onStart: (plotReq: string, overwriteChapter: number, branchFromID: string) => void
}

/**
 * 剧情方向向导弹窗（T6-7.5 从 CreatePage 拆分）。流程是「先选角、后构思」：
 * 打开先停在选角会议（圈定登场角色+主角关系，可延续上一章/角色库挑入/简单
 * 新建），作者点「构思分支」才带着名单去生成；分支出来后改选角须重新构思
 * （有可见提示），直接生成则用当前选角不再重跑。
 */
const BranchWizardModal: React.FC<BranchWizardModalProps> = ({
  open, prevChapter, overwriteChapter, branchFromID, characters, libraryCharacters, prevChapterCast,
  onClose, onFetchBranches, onStart,
}) => {
  const [wizStep, setWizStep] = useState<'cast' | 'loading' | 'branches'>('cast')
  const [branches, setBranches] = useState<Branch[]>([])
  const [selectedBranch, setSelectedBranch] = useState<number | null>(null)
  const [userInput, setUserInput] = useState('')
  const [cast, setCast] = useState<BranchCastEntry[]>([])
  const castRef = useRef<BranchCastEntry[]>([])
  castRef.current = cast
  // 分支生成时的选角快照：分支步骤里改了选角 → 提示「重新构思后生效」
  const castAtGenRef = useRef('')

  const sanitizedCast = (cs: BranchCastEntry[]) =>
    cs.filter(e => e.name.trim())
      .map(e => ({ name: e.name.trim(), relation: e.relation?.trim() || undefined, note: e.note?.trim() || undefined, isNew: e.isNew }))

  const loadBranches = useCallback(async (fromStep: 'cast' | 'branches') => {
    setWizStep('loading')
    try {
      const list = await onFetchBranches(prevChapter, sanitizedCast(castRef.current))
      castAtGenRef.current = JSON.stringify(sanitizedCast(castRef.current))
      setBranches(list)
      setSelectedBranch(null)
      setUserInput('')
      setWizStep('branches')
    } catch (err: unknown) {
      // 构思失败不静默：回到发起步骤，可改选角重试或手输剧情要求直接生成
      setWizStep(fromStep)
      message.error(err instanceof Error ? err.message : '剧情构思失败，可手动输入剧情要求')
    }
  }, [onFetchBranches, prevChapter])

  // 打开先停在选角会议（上一章的名单不该静默带进这一章，要带点「延续上一章」）
  useEffect(() => {
    if (open) {
      setCast([])
      setBranches([])
      setSelectedBranch(null)
      setUserInput('')
      castAtGenRef.current = ''
      setWizStep('cast')
    }
  }, [open])

  const confirmGenerate = () => {
    const chosen = selectedBranch !== null ? branches[selectedBranch] : null
    const plotReq = userInput.trim() || (chosen ? `${chosen.title}：${chosen.pitch}` : '')
    onStart(plotReq, overwriteChapter, branchFromID)
    onClose()
  }

  const addCast = (v: string) => {
    if (v.startsWith('lib:')) {
      const c = libraryCharacters.find(x => `lib:${x.name}` === v)
      if (c) setCast(cs => cs.some(x => x.name === c.name) ? cs : [...cs, { name: c.name, note: c.note ?? '' }])
      return
    }
    setCast(cs => cs.some(x => x.name === v) ? cs : [...cs, { name: v }])
  }
  const inCast = (name: string) => cast.some(x => x.name === name)

  const castEditor = (
    <div style={{ marginBottom: 12 }}>
      <Space wrap style={{ marginBottom: 6 }} align="center">
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>选角（留空=全量名册不设限）：</Typography.Text>
        {prevChapterCast.length > 0 && (
          <Button size="small" onClick={() => setCast(prevChapterCast.filter(Boolean).map(n => ({ name: n })))}>延续上一章</Button>
        )}
        <Button size="small" icon={<PlusOutlined />}
          onClick={() => setCast(cs => [...cs, { name: '', relation: '', isNew: true }])}>新建角色</Button>
      </Space>
      {cast.map((e, i) => (
        <Space key={`${i}-${e.name}`} align="center" style={{ display: 'flex', marginBottom: 4, flexWrap: 'wrap' }}>
          {e.isNew ? (
            <Input size="small" placeholder="新角色名字" value={e.name} style={{ width: 130 }}
              onChange={ev => setCast(cs => cs.map((x, j) => j === i ? { ...x, name: ev.target.value } : x))} />
          ) : (
            <Typography.Text style={{ fontSize: 13, color: C('color-text'), display: 'inline-block', minWidth: 130 }}>
              {e.name}{e.note ? <Typography.Text type="secondary" style={{ fontSize: 11 }}>·{e.note}</Typography.Text> : null}
            </Typography.Text>
          )}
          <Input size="small" placeholder="与主角关系（师妹/死敌/恋人…）" value={e.relation ?? ''} style={{ width: 200 }}
            onChange={ev => setCast(cs => cs.map((x, j) => j === i ? { ...x, relation: ev.target.value } : x))} />
          <Button type="text" size="small" aria-label={`移除 ${e.name || '角色'}`}
            onClick={() => setCast(cs => cs.filter((_, j) => j !== i))}>✕</Button>
        </Space>
      ))}
      <Select placeholder="添加角色…" style={{ width: 260, marginTop: cast.length ? 2 : 0 }} size="small" showSearch
        onSelect={(v: unknown) => addCast(String(v))}
        options={[
          { label: '项目角色', options: characters.filter(n => !inCast(n)).map(n => ({ label: n, value: n })) },
          { label: '角色库', options: libraryCharacters.filter(c => !inCast(c.name)).map(c => ({ label: `${c.name}·库`, value: `lib:${c.name}` })) },
        ]}
      />
    </div>
  )

  const manualInput = (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>或输入你自己的剧情要求：</Typography.Text>
      <Input.TextArea value={userInput} onChange={e => { setUserInput(e.target.value); setSelectedBranch(null) }} rows={2} placeholder="剧情要求…"
        style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 'var(--radius-md)' }} />
    </Space>
  )

  return (
    <Modal title={<><BulbOutlined style={{ marginRight: 8 }} />剧情方向</>}
      open={open} onCancel={onClose} footer={null} width={620}
      destroyOnHidden transitionName="" maskTransitionName="">
      {wizStep === 'loading' && <div style={{ textAlign: 'center', padding: 24 }}><Spin size="large" /><div style={{ marginTop: 8, color: C('color-text-secondary'), fontSize: 12 }}>AI 正在按当前选角构思剧情分支…</div></div>}
      {wizStep === 'cast' && (<React.Fragment>
        {castEditor}
        {manualInput}
        <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between' }}>
          <Button type="primary" onClick={() => void loadBranches('cast')}>构思分支</Button>
          <Button onClick={confirmGenerate} disabled={!userInput.trim()}>跳过构思，直接生成</Button>
        </div>
      </React.Fragment>)}
      {wizStep === 'branches' && (<React.Fragment>
        {castEditor}
        {castAtGenRef.current !== JSON.stringify(sanitizedCast(cast)) && (
          <div style={{ marginBottom: 8, color: 'var(--color-warning, #faad14)', fontSize: 12 }}>选角已调整，点「重新构思」后生效</div>
        )}
        {branches.length > 0 && <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 12 }}>
          {branches.map((b, i) => (
            <Card key={i} size="small" hoverable onClick={() => { setSelectedBranch(i); setUserInput('') }}
              style={{ cursor: 'pointer', border: selectedBranch === i ? '2px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)', background: selectedBranch === i ? 'var(--md-sys-color-primary-container)' : 'var(--bg-elevated)' }}>
              <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{i + 1}. {b.title}</Typography.Text>
              <Typography.Paragraph style={{ color: C('color-text-secondary'), fontSize: 12, margin: '4px 0 0' }}>{b.pitch}</Typography.Paragraph>
            </Card>
          ))}
        </div>}
        {manualInput}
        <div style={{ marginTop: 12, textAlign: 'right' }}>
          <Space>
            <Button onClick={() => void loadBranches('branches')}>重新构思</Button>
            <Button type="primary" onClick={confirmGenerate} disabled={selectedBranch === null && !userInput.trim()}>
              {overwriteChapter > 0 ? `重新生成第${overwriteChapter}章` : '生成'}
            </Button>
          </Space>
        </div>
      </React.Fragment>)}
    </Modal>
  )
}

export default BranchWizardModal
