// CharacterMemoryModal.tsx — 角色库：查看角色的状态 / 记忆 / 追踪
// 聊天面板不再展示角色状态，统一归集到这里（只读查看 + 记忆管理）。
import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Modal, Tabs, Tag, Button } from 'antd'
import { HeartOutlined, InboxOutlined, RadarChartOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import type { characterlib, whisper } from '../../../wailsjs/go/models'
import { C } from '../../utils/theme'
import { WhisperEmotionPanel } from '../WhisperEmotionPanel'
import WhisperDesirePanel from '../WhisperDesirePanel'
import WhisperTracePanel from '../WhisperTracePanel'
import WhisperMemoryList from '../WhisperMemoryList'
import WhisperMemoryModal from '../WhisperMemoryModal'
import type { MemoryFact } from '../WhisperMemoryModal'

interface Props {
  open: boolean
  character: characterlib.Character | null
  onClose: () => void
}

const CharacterMemoryModal: React.FC<Props> = ({ open, character, onClose }) => {
  const [state, setState] = useState<Record<string, unknown>>({})
  const [facts, setFacts] = useState<MemoryFact[]>([])
  const [traces, setTraces] = useState<whisper.TurnTrace[]>([])
  const [manageOpen, setManageOpen] = useState(false)
  // v4.351：载入失败可见化——此前三路读取全部吞错，弹窗打开即全空白，
  // 与「该角色还没有任何记忆数据」不可区分。
  const [loadFailed, setLoadFailed] = useState(false)

  const load = useCallback(async () => {
    if (!character) return
    setState({}); setFacts([]); setTraces([]); setLoadFailed(false)
    let failed = 0
    try {
      const s = await app.WhisperGetState(character.id)
      setState((s as Record<string, unknown>) || {})
    } catch (_) { failed++ }
    try {
      const f = await app.WhisperGetFacts(character.id)
      setFacts((Array.isArray(f) ? f : []) as unknown as MemoryFact[])
    } catch (_) { failed++ }
    try {
      const t = await app.WhisperGetTraces(character.id)
      setTraces((Array.isArray(t) ? t : []) as unknown as whisper.TurnTrace[])
    } catch (_) { failed++ }
    setLoadFailed(failed > 0)
  }, [character])

  useEffect(() => {
    if (open && character) load()
  }, [open, character, load])

  if (!character) return null

  const rel = (state?.relationship || {}) as { stage: string; trust: number; rifts: number }
  const emo = (state?.emotion || {}) as { label: string; aff: number; sec: number; aro: number; dom: number }
  const personality = (state?.personality || {}) as { T: number; I: number; S: number; O: number; R: number }
  const desireSlots = ((state?.desireStack as { slots?: Array<{ id: string; topic: string; category: string; urgency: number; status: string } | null> } | undefined)?.slots) || []
  const totalTurns = (state?.totalTurns as number) || 0

  return (
    <>
      <Modal open={open} onCancel={onClose} footer={null} width={680}
        destroyOnHidden transitionName="" maskTransitionName=""
        title={
          <span style={{ color: C('color-text') }}>
            <HeartOutlined style={{ color: 'var(--gaea-glow)', marginRight: 8 }} />
            {character.name} · 状态 / 记忆 / 追踪
          </span>
        }
        styles={{ body: { maxHeight: '68vh', overflowY: 'auto' } }}
      >
        {loadFailed && (
          <Alert
            type="warning"
            showIcon
            message="记忆数据载入失败"
            description="部分或全部数据读取失败，当前内容可能不完整。可关闭后重试。"
            style={{ marginBottom: 12 }}
          />
        )}
        <Tabs
          size="small"
          items={[
            {
              key: 'status',
              label: <span><RadarChartOutlined /> 状态</span>,
              children: (
                <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
                  <WhisperEmotionPanel
                    emotion={emo.label} stage={rel.stage} trust={rel.trust} rifts={rel.rifts}
                    aff={emo.aff} sec={emo.sec} aro={emo.aro} dom={emo.dom}
                    T={personality.T ?? character.dims?.T ?? 50}
                    I={personality.I ?? character.dims?.I ?? 50}
                    S={personality.S ?? character.dims?.S ?? 50}
                    O={personality.O ?? character.dims?.O ?? 50}
                    R={personality.R ?? character.dims?.R ?? 50}
                    totalTurns={totalTurns}
                    personalityLabel={character.name}
                  />
                  <WhisperDesirePanel desireStack={{ slots: desireSlots }} sharedEventsCount={0} />
                </div>
              ),
            },
            {
              key: 'memory',
              label: (
                <span>
                  <InboxOutlined /> 记忆
                  {facts.length > 0 && <Tag style={{ fontSize: 9, margin: 0, padding: '0 4px', lineHeight: '14px' }}>{facts.length}</Tag>}
                </span>
              ),
              children: (
                <div style={{ position: 'relative', height: 420 }}>
                  <div style={{ position: 'absolute', right: 0, top: -36, zIndex: 2 }}>
                    <Button size="small" type="link" icon={<InboxOutlined />} onClick={() => setManageOpen(true)}
                      style={{ fontSize: 11.5, color: 'var(--gaea-glow)' }}>管理记忆</Button>
                  </div>
                  <WhisperMemoryList facts={facts} onOpenManage={() => setManageOpen(true)} />
                </div>
              ),
            },
            {
              key: 'trace',
              label: <span><RadarChartOutlined /> 追踪</span>,
              children: <WhisperTracePanel traces={traces} currentTurn={totalTurns} />,
            },
          ]}
        />
      </Modal>
      {manageOpen && (
        <WhisperMemoryModal
          facts={facts}
          personalityID={character.id}
          onFactsChange={setFacts}
        />
      )}
    </>
  )
}

export default CharacterMemoryModal
