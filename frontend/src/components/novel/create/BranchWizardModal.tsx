import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Card, Input, message, Modal, Select, Space, Spin, Typography } from 'antd'
import { BulbOutlined } from '@ant-design/icons'
import { C } from '../../../utils/theme'

export interface Branch { title: string; pitch: string }

interface BranchWizardModalProps {
  open: boolean
  prevChapter: number
  overwriteChapter: number
  branchFromID: string
  /** 项目角色名册（选择器候选；空名册隐藏选择器） */
  characters: string[]
  onClose: () => void
  /** 拉取 AI 构思的分支列表（由页面层注入最新设定与前文摘要；characters=作者圈定的登场角色） */
  onFetchBranches: (prevChapter: number, characters: string[]) => Promise<Branch[]>
  /** 确认生成：plotReq 由分支标题或用户输入组装 */
  onStart: (plotReq: string, overwriteChapter: number, branchFromID: string) => void
  /** 打开时已后台构思完成的分支；为空则打开后自动构思（用于弹窗内「重新构思」） */
  preloadedBranches?: Branch[]
}

/**
 * 剧情方向向导弹窗（T6-7.5 从 CreatePage 拆分）：自持构思步骤/分支/选择状态，
 * 打开时自动拉取 AI 构思；「重新构思」与「生成」按钮行为与旧实现一致。
 * 角色选择器：作者圈定登场角色后构思只围绕他们展开（留空=不限，全量名册注入）。
 */
const BranchWizardModal: React.FC<BranchWizardModalProps> = ({
  open, prevChapter, overwriteChapter, branchFromID, characters, onClose, onFetchBranches, onStart, preloadedBranches,
}) => {
  const [wizStep, setWizStep] = useState<'loading' | 'branches'>('loading')
  const [branches, setBranches] = useState<Branch[]>([])
  const [selectedBranch, setSelectedBranch] = useState<number | null>(null)
  const [userInput, setUserInput] = useState('')
  const [selectedChars, setSelectedChars] = useState<string[]>([])
  const selectedCharsRef = useRef<string[]>([])
  selectedCharsRef.current = selectedChars

  const loadBranches = useCallback(async () => {
    setWizStep('loading')
    try {
      // 读 ref 而非 state：重新构思带走当前圈定，但选择变化本身不触发重新构思
      const list = await onFetchBranches(prevChapter, selectedCharsRef.current)
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
  // 每次打开重置角色圈定（上一章的名单不该静默带进这一章）。
  useEffect(() => {
    if (open) {
      setSelectedChars([])
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

  return (
    <Modal title={<><BulbOutlined style={{ marginRight: 8 }} />剧情方向</>}
      open={open} onCancel={onClose} footer={null} width={620}
      destroyOnHidden transitionName="" maskTransitionName="">
      {wizStep === 'loading' && <div style={{ textAlign: 'center', padding: 24 }}><Spin size="large" /><div style={{ marginTop: 8, color: C('color-text-secondary'), fontSize: 12 }}>AI 正在分析设定，构思剧情分支…</div></div>}
      {wizStep === 'branches' && (<React.Fragment>
        {characters.length > 0 && (
          <div style={{ marginBottom: 12 }}>
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>登场角色（可多选后「重新构思」，留空=不限）：</Typography.Text>
            <Select mode="multiple" allowClear value={selectedChars} onChange={setSelectedChars}
              options={characters.map(n => ({ label: n, value: n }))}
              placeholder="圈定本轮分支围绕哪些角色展开…" maxTagCount={6} size="small"
              style={{ width: '100%', marginTop: 4 }} />
          </div>
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
