import React, { useEffect, useRef, useState } from 'react'
import { Button, Card, Input, Modal, Select, Space, Typography } from 'antd'
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
  /** 选角会议结果（受控：状态由页面层持有，弹窗收起后构思期间不丢） */
  cast: BranchCastEntry[]
  onCastChange: (cast: BranchCastEntry[]) => void
  /** 后台构思完成的分支；非空=直接进入分支步（构思由页面层编排，弹窗不阻塞） */
  preloadedBranches?: Branch[]
  /** 选角会议完成：把名单交给页面层后台构思（弹窗随即收起，可继续操作） */
  onStartBrainstorm: (cast: BranchCastEntry[]) => void
  onClose: () => void
  /** 确认生成：plotReq 由分支标题或用户输入组装 */
  onStart: (plotReq: string, overwriteChapter: number, branchFromID: string) => void
}

/**
 * 剧情方向向导弹窗（T6-7.5 从 CreatePage 拆分）。流程「先选角、后后台构思」：
 * 打开先停在选角会议（圈定登场角色+主角关系，可延续上一章/角色库挑入/简单
 * 新建）；点「构思分支」把名单交给页面层后台构思并收起弹窗——期间可继续
 * 操作其他内容；构思完成自动弹回分支步。分支步改选角会提示重新构思。
 */
const BranchWizardModal: React.FC<BranchWizardModalProps> = ({
  open, prevChapter, overwriteChapter, branchFromID, characters, libraryCharacters, prevChapterCast,
  cast, onCastChange, preloadedBranches, onStartBrainstorm, onClose, onStart,
}) => {
  const [selectedBranch, setSelectedBranch] = useState<number | null>(null)
  const [userInput, setUserInput] = useState('')
  // 分支生成时的选角快照：分支步里改了选角 → 提示「重新构思后生效」
  const castAtGenRef = useRef('')

  const sanitizedCast = (cs: BranchCastEntry[]) =>
    cs.filter(e => e.name.trim())
      .map(e => ({ name: e.name.trim(), relation: e.relation?.trim() || undefined, note: e.note?.trim() || undefined, isNew: e.isNew }))

  // 打开即按预载分支定步：有=分支步，无=选角会议。重开选角步时快照清零。
  useEffect(() => {
    if (!open) return
    setSelectedBranch(null)
    setUserInput('')
    if (preloadedBranches && preloadedBranches.length > 0) {
      castAtGenRef.current = JSON.stringify(sanitizedCast(cast))
    } else {
      castAtGenRef.current = ''
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, preloadedBranches])

  const atBranchesStep = !!preloadedBranches && preloadedBranches.length > 0
  const castDirty = atBranchesStep && castAtGenRef.current !== JSON.stringify(sanitizedCast(cast))

  const confirmGenerate = () => {
    const chosen = selectedBranch !== null ? preloadedBranches?.[selectedBranch] : undefined
    const plotReq = userInput.trim() || (chosen ? `${chosen.title}：${chosen.pitch}` : '')
    onStart(plotReq, overwriteChapter, branchFromID)
    onClose()
  }

  const addCast = (v: string) => {
    if (v.startsWith('lib:')) {
      const c = libraryCharacters.find(x => `lib:${x.name}` === v)
      if (c) onCastChange(cast.some(x => x.name === c.name) ? cast : [...cast, { name: c.name, note: c.note ?? '' }])
      return
    }
    onCastChange(cast.some(x => x.name === v) ? cast : [...cast, { name: v }])
  }
  const inCast = (name: string) => cast.some(x => x.name === name)

  const castEditor = (
    <div style={{ marginBottom: 12 }}>
      <Space wrap style={{ marginBottom: 6 }} align="center">
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>选角（留空=全量名册不设限）：</Typography.Text>
        {prevChapterCast.length > 0 && (
          <Button size="small" onClick={() => onCastChange(prevChapterCast.filter(Boolean).map(n => ({ name: n })))}>延续上一章</Button>
        )}
        <Button size="small" icon={<PlusOutlined />}
          onClick={() => onCastChange([...cast, { name: '', relation: '', isNew: true }])}>新建角色</Button>
      </Space>
      {cast.map((e, i) => (
        <Space key={`${i}-${e.name}`} align="center" style={{ display: 'flex', marginBottom: 4, flexWrap: 'wrap' }}>
          {e.isNew ? (
            <Input size="small" placeholder="新角色名字" value={e.name} style={{ width: 130 }}
              onChange={ev => onCastChange(cast.map((x, j) => j === i ? { ...x, name: ev.target.value } : x))} />
          ) : (
            <Typography.Text style={{ fontSize: 13, color: C('color-text'), display: 'inline-block', minWidth: 130 }}>
              {e.name}{e.note ? <Typography.Text type="secondary" style={{ fontSize: 11 }}>·{e.note}</Typography.Text> : null}
            </Typography.Text>
          )}
          <Input size="small" placeholder="与主角关系（师妹/死敌/恋人…）" value={e.relation ?? ''} style={{ width: 200 }}
            onChange={ev => onCastChange(cast.map((x, j) => j === i ? { ...x, relation: ev.target.value } : x))} />
          <Button type="text" size="small" aria-label={`移除 ${e.name || '角色'}`}
            onClick={() => onCastChange(cast.filter((_, j) => j !== i))}>✕</Button>
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
      {!atBranchesStep && (<React.Fragment>
        {castEditor}
        {manualInput}
        <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between' }}>
          <Button type="primary" onClick={() => onStartBrainstorm(sanitizedCast(cast))}>构思分支（后台进行）</Button>
          <Button onClick={confirmGenerate} disabled={!userInput.trim()}>跳过构思，直接生成</Button>
        </div>
      </React.Fragment>)}
      {atBranchesStep && (<React.Fragment>
        {castEditor}
        {castDirty && (
          <div style={{ marginBottom: 8, color: 'var(--color-warning, #faad14)', fontSize: 12 }}>选角已调整，点「重新构思」后生效</div>
        )}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 12 }}>
          {(preloadedBranches ?? []).map((b, i) => (
            <Card key={i} size="small" hoverable onClick={() => { setSelectedBranch(i); setUserInput('') }}
              style={{ cursor: 'pointer', border: selectedBranch === i ? '2px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)', background: selectedBranch === i ? 'var(--md-sys-color-primary-container)' : 'var(--bg-elevated)' }}>
              <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{i + 1}. {b.title}</Typography.Text>
              <Typography.Paragraph style={{ color: C('color-text-secondary'), fontSize: 12, margin: '4px 0 0' }}>{b.pitch}</Typography.Paragraph>
            </Card>
          ))}
        </div>
        {manualInput}
        <div style={{ marginTop: 12, textAlign: 'right' }}>
          <Space>
            <Button onClick={() => onStartBrainstorm(sanitizedCast(cast))}>重新构思</Button>
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
