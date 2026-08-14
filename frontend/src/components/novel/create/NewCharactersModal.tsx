import React, { useEffect, useState } from 'react'
import { Button, Checkbox, Input, Modal, Tag, Typography, message } from 'antd'
import * as App from '../../../../src/wailsjsCompat'
import { associateToProject, syncProjectCharacters } from '../../../api/characterlib'

// 新角色条目（可编辑名称 + 可选择）
interface NewCharEntry {
  original: string   // AI 提取的原始名
  name: string       // 编辑后的名字
  selected: boolean
}

// 角色库已有同名角色（直接关联，不新建）
interface LibMatchEntry {
  id: string
  name: string
  roleType?: string
  portraitUrl?: string
  selected: boolean
}

/** new-characters-discovered 事件动态载荷（最小消费面） */
interface NewCharactersPayload {
  characters?: string[]
  libraryMatches?: Array<{ id?: string; name?: string; roleType?: string; portraitUrl?: string }>
  chapterNum?: number
}

const roleLabels: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '次要',
}

/**
 * 新角色发现弹窗（T6-7.5 从 CreatePage 拆分）：自订阅 'new-characters-discovered'
 * 事件，自持弹窗/条目/选中/保存中状态，页面仅需渲染 <NewCharactersModal />。
 */
const NewCharactersModal: React.FC = () => {
  const [open, setOpen] = useState(false)
  const [newCharsList, setNewCharsList] = useState<NewCharEntry[]>([])
  const [newCharsChapter, setNewCharsChapter] = useState(0)
  const [libMatches, setLibMatches] = useState<LibMatchEntry[]>([])
  const [adding, setAdding] = useState(false)

  // 监听新角色发现事件
  useEffect(() => {
    const handler = (event: { detail?: NewCharactersPayload } | NewCharactersPayload) => {
      const raw = event as { detail?: NewCharactersPayload } | null | undefined
      const data = (raw?.detail || raw) as NewCharactersPayload | undefined
      if (data?.characters?.length > 0 || data?.libraryMatches?.length > 0) {
        setNewCharsList((data.characters || []).map((name: string) => ({
          original: name, name, selected: true,
        })))
        setLibMatches((data.libraryMatches || []).map((m) => ({
          id: m.id ?? '', name: m.name ?? '', roleType: m.roleType || '', portraitUrl: m.portraitUrl || '', selected: true,
        })))
        setNewCharsChapter(data.chapterNum || 0)
        setOpen(true)
      }
    }
    try { window.runtime?.EventsOn?.('new-characters-discovered', handler) } catch { /* 事件通道缺失时弹窗不可用 */ }
    return () => { try { window.runtime?.EventsOff?.('new-characters-discovered') } catch { /* 清理失败无害 */ } }
  }, [])

  const selectedCount = newCharsList.filter(c => c.selected).length + libMatches.filter(m => m.selected).length

  return (
    <Modal
      title={<>🔍 第{newCharsChapter}章发现了 {selectedCount} 个新角色</>}
      open={open}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      confirmLoading={adding}
      onOk={async () => {
        const selected = newCharsList.filter(c => c.selected).map(c => c.name)
        const libSelected = libMatches.filter(m => m.selected)
        if (selected.length === 0 && libSelected.length === 0) {
          message.warning('请至少选择一个角色')
          return
        }
        setAdding(true)
        try {
          if (selected.length > 0) {
            await App.SaveCharactersBatch(JSON.stringify(selected))
            message.success(`已添加 ${selected.length} 个新角色（含 AI 完整档案）`)
          }
          if (libSelected.length > 0) {
            for (const m of libSelected) {
              await associateToProject(m.id, m.roleType || 'supporting')
            }
            await syncProjectCharacters()
            message.success(`已关联 ${libSelected.length} 个角色库已有角色`)
          }
        } catch (err: unknown) {
          message.error(err instanceof Error ? err.message : '添加失败')
        } finally {
          setAdding(false)
          setOpen(false)
          // 角色面板常驻挂载：通知其重新读取全局库与项目引用
          try { window.dispatchEvent(new CustomEvent('gaea-project-chars-changed')) } catch { /* 通知失败无害 */ }
        }
      }}
      onCancel={() => setOpen(false)}
      okText={adding ? 'AI 生成档案中…' : `确认添加 (${selectedCount})`}
      cancelText="稍后处理"
      width={480}
    >
      {libMatches.length > 0 && (
        <>
          <div style={{ marginBottom: 4, fontSize: 12, fontWeight: 600, color: '#60a5fa' }}>
            角色库已有同名角色（直接关联，不新建）
          </div>
          <div style={{
            maxHeight: 180, overflow: 'auto', border: '1px solid rgba(96,165,250,0.25)',
            borderRadius: 8, padding: '2px 8px', marginBottom: 10,
          }}>
            {libMatches.map((m, i) => (
              <div key={m.id} style={{
                display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0',
                borderBottom: i < libMatches.length - 1 ? '1px solid var(--border-subtle)' : 'none',
              }}>
                <Checkbox
                  checked={m.selected}
                  onChange={e => setLibMatches(prev => prev.map((x, j) => j === i ? { ...x, selected: e.target.checked } : x))}
                />
                <span style={{ fontSize: 13, color: 'var(--color-text)' }}>{m.name}</span>
                {m.roleType && (
                  <Tag color="blue" style={{ marginInlineEnd: 0, fontSize: 11 }}>{roleLabels[m.roleType] || m.roleType}</Tag>
                )}
                <div style={{ flex: 1 }} />
                <Typography.Text type="secondary" style={{ fontSize: 11 }}>库内角色</Typography.Text>
              </div>
            ))}
          </div>
        </>
      )}
      <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Checkbox
          disabled={newCharsList.length === 0}
          checked={newCharsList.every(c => c.selected)}
          indeterminate={newCharsList.some(c => c.selected) && !newCharsList.every(c => c.selected)}
          onChange={e => setNewCharsList(prev => prev.map(c => ({ ...c, selected: e.target.checked })))}
        >
          全选
        </Checkbox>
        <Typography.Text type="secondary" style={{ fontSize: 11 }}>
          💡 可编辑名字合并重复称呼
        </Typography.Text>
      </div>
      {newCharsList.length > 0 && (
        <div style={{ maxHeight: 260, overflow: 'auto' }}>
          {newCharsList.map((entry, i) => (
            <div key={i} style={{
              display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0',
              borderBottom: '1px solid var(--border-subtle)'
            }}>
              <Checkbox
                checked={entry.selected}
                onChange={e => {
                  setNewCharsList(prev => prev.map((c, j) => j === i ? { ...c, selected: e.target.checked } : c))
                }}
              />
              <Input
                size="small"
                value={entry.name}
                onChange={e => {
                  setNewCharsList(prev => prev.map((c, j) => j === i ? { ...c, name: e.target.value } : c))
                }}
                style={{
                  flex: 1, background: entry.name !== entry.original ? 'rgba(245,158,11,0.08)' : 'transparent',
                  border: entry.name !== entry.original ? '1px solid rgba(245,158,11,0.4)' : '1px solid transparent',
                  color: 'var(--color-text)', fontSize: 13
                }}
              />
              {entry.name !== entry.original && (
                <Button type="text" size="small"
                  onClick={() => setNewCharsList(prev => prev.map((c, j) => j === i ? { ...c, name: c.original } : c))}
                  style={{ fontSize: 10, padding: '0 2px', height: 20, color: '#f59e0b' }}>
                  还原
                </Button>
              )}
            </div>
          ))}
        </div>
      )}
      <Typography.Text type="secondary" style={{ fontSize: 12, marginTop: 8, display: 'block' }}>
        新角色将 AI 生成完整档案（性格/背景/外貌）并标记为「配角·存活」；角色库已有角色直接关联，不新建。
      </Typography.Text>
    </Modal>
  )
}

export default NewCharactersModal
