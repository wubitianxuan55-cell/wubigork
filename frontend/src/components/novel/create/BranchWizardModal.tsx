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
  /** 打开时已后台构思完成的分支；为空则打开后自动构思（用于弹窗内「重新构思」） */
  preloadedBranches?: Branch[]
}

/**
 * 剧情方向向导弹窗（T6-7.5 从 CreatePage 拆分）：自持构思步骤/分支/选择状态，
 * 打开时自动拉取 AI 构思；「重新构思」与「生成」按钮行为与旧实现一致。
 * 选角会议：规划前圈定登场角色并给「与主角关系」——可延续上一章、可从角色库
 * 挑入、可简单新建（名字+关系）；圈定后构思只围绕这批人展开，留空=全量名册。
 */
const BranchWizardModal: React.FC<BranchWizardModalProps> = ({
  open, prevChapter, overwriteChapter, branchFromID, characters, libraryCharacters, prevChapterCast,
  onClose, onFetchBranches, onStart, preloadedBranches,
}) => {
  const [wizStep, setWizStep] = useState<'loading' | 'branches'>('loading')
  const [branches, setBranches] = useState<Branch[]>([])
  const [selectedBranch, setSelectedBranch] = useState<number | null>(null)
  const [userInput, setUserInput] = useState('')
  const [cast, setCast] = useState<BranchCastEntry[]>([])
  const castRef = useRef<BranchCastEntry[]>([])
  castRef.current = cast

  const loadBranches = useCallback(async () => {
    setWizStep('loading')
    try {
      // 读 ref 而非 state：重新构思带走当前选角，但选角变化本身不触发重新构思
      const payload = castRef.current
        .filter(e => e.name.trim())
        .map(e => ({ name: e.name.trim(), relation: e.relation?.trim() || undefined, note: e.note?.trim() || undefined, isNew: e.isNew }))
      const list = await onFetchBranches(prevChapter, payload)
      setBranches(list)
      setSelectedBranch(null)
      setUserInput('')
    } catch (err: unknown) {
      // 构思失败不静默：提示用户可手动输入剧情要求继续
      setBranches([])
      message.error(err instanceof Error ? err.message : '剧情构思失败，可手动输入剧情要求')
    } finally {
      setWizStep('branches')
    }
  }, [onFetchBranches, prevChapter])

  // 打开时优先使用后台已构思完成的分支，直接进入选择确认；否则自动构思。
  // 每次打开重置选角（上一章的名单不该静默带进这一章，要带点「延续上一章」）。
  useEffect(() => {
    if (open) {
      setCast([])
      if (preloadedBranches && preloadedBranches.length > 0) {
        setBranches(preloadedBranches)
        setSelectedBranch(null)
        setUserInput('')
        setWizStep('branches')
      } else {
        void loadBranches()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, preloadedBranches])

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

  return (
    <Modal title={<><BulbOutlined style={{ marginRight: 8 }} />剧情方向</>}
      open={open} onCancel={onClose} footer={null} width={620}
      destroyOnHidden transitionName="" maskTransitionName="">
      {wizStep === 'loading' && <div style={{ textAlign: 'center', padding: 24 }}><Spin size="large" /><div style={{ marginTop: 8, color: C('color-text-secondary'), fontSize: 12 }}>AI 正在分析设定，构思剧情分支…</div></div>}
      {wizStep === 'branches' && (<React.Fragment>
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
        {branches.length > 0 && <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 12 }}>
          {branches.map((b, i) => (
            <Card key={i} size="small" hoverable onClick={() => { setSelectedBranch(i); setUserInput('') }}
              style={{ cursor: 'pointer', border: selectedBranch === i ? '2px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)', background: selectedBranch === i ? 'var(--md-sys-color-primary-container)' : 'var(--bg-elevated)' }}>
              <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{i + 1}. {b.title}</Typography.Text>
              <Typography.Paragraph style={{ color: C('color-text-secondary'), fontSize: 12, margin: '4px 0 0' }}>{b.pitch}</Typography.Paragraph>
            </Card>
          ))}
        </div>}
        <Space direction="vertical" style={{ width: '100%' }}>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>或输入你自己的剧情要求：</Typography.Text>
          <Input.TextArea value={userInput} onChange={e => { setUserInput(e.target.value); setSelectedBranch(null) }} rows={2} placeholder="剧情要求…"
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 'var(--radius-md)' }} />
        </Space>
        <div style={{ marginTop: 12, textAlign: 'right' }}>
          <Space>
            <Button onClick={() => void loadBranches()}>重新构思</Button>
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
