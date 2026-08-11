// CharacterMemoryModal.tsx — 角色库：查看角色的状态 / 记忆 / 追踪
// 聊天面板不再展示角色状态，统一归集到这里（只读查看 + 记忆管理）。
import React, { useCallback, useEffect, useState } from 'react'
import { Modal, Tabs, Tag, Button } from 'antd'
import { HeartOutlined, InboxOutlined, RadarChartOutlined } from '@ant-design/icons'
import * as App from '../../../wailsjs/go/app/App'
import type { characterlib } from '../../../wailsjs/go/models'
import { C } from '../../utils/theme'
import { WhisperEmotionPanel } from '../WhisperEmotionPanel'
import WhisperDesirePanel from '../WhisperDesirePanel'
import WhisperTracePanel from '../WhisperTracePanel'
import WhisperMemoryList from '../WhisperMemoryList'
import WhisperMemoryModal from '../WhisperMemoryModal'

interface Props {
  open: boolean
  character: characterlib.Character | null
  onClose: () => void
}

const CharacterMemoryModal: React.FC<Props> = ({ open, character, onClose }) => {
  const [state, setState] = useState<Record<string, any>>({})
  const [facts, setFacts] = useState<any[]>([])
  const [traces, setTraces] = useState<any[]>([])
  const [manageOpen, setManageOpen] = useState(false)

  const load = useCallback(async () => {
    if (!character) return
    setState({}); setFacts([]); setTraces([])
    try {
      const s = await App.WhisperGetState(character.id)
      setState((s as Record<string, any>) || {})
    } catch (_) {}
    try {
      const f = await App.WhisperGetFacts(character.id)
      setFacts(Array.isArray(f) ? f : [])
    } catch (_) {}
    try {
      const t = await App.WhisperGetTraces(character.id)
      setTraces(Array.isArray(t) ? t : [])
    } catch (_) {}
  }, [character])

  useEffect(() => {
    if (open && character) load()
  }, [open, character, load])

  if (!character) return null

  const rel = state?.relationship || {}
  const emo = state?.emotion || {}
  const personality = state?.personality || {}
  const desireSlots = state?.desireStack?.slots || []
  const totalTurns = state?.totalTurns || 0

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
