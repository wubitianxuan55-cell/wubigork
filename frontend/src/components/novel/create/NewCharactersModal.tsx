import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Checkbox, Input, Modal, Tag, Typography, message } from 'antd'
import { app } from '../../../gaea/lib/bridge'
import { subscribeWailsEvent } from '../../../gaea/lib/wailsEvents'
import { associateToProject, syncProjectCharacters } from '../../../api/characterlib'

/** 新角色发现通道（对齐 events.ts BACKEND_EVENTS.NEW_CHARACTERS_DISCOVERED）。 */
const NEW_CHARACTERS_CHANNEL = 'new-characters-discovered'

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

/** 待弹的发现（事件载荷归一而来的可编辑条目，active=false 期间挂起） */
interface PendingDiscovery {
  list: NewCharEntry[]
  libs: LibMatchEntry[]
  chapter: number
}

/** 事件载荷 → 可编辑条目（新角色默认全选，库内同名角色默认关联）。 */
function toDiscovery(data: NewCharactersPayload | undefined): PendingDiscovery {
  return {
    list: (data?.characters || []).map((name: string) => ({ original: name, name, selected: true })),
    libs: (data?.libraryMatches || []).map((m) => ({
      id: m.id ?? '', name: m.name ?? '', roleType: m.roleType || '', portraitUrl: m.portraitUrl || '', selected: true,
    })),
    chapter: data?.chapterNum || 0,
  }
}

/**
 * 挂起期间可能连发多章（切页期间事件照发），按名字 / 角色 id 去重合并——
 * 只留最后一批会把更早的发现永久丢掉，正是 A8 要避免的。
 */
function mergeDiscovery(prev: PendingDiscovery | null, next: PendingDiscovery): PendingDiscovery {
  if (!prev) return next
  const list = [...prev.list]
  for (const e of next.list) if (!list.some((x) => x.name === e.name)) list.push(e)
  const libs = [...prev.libs]
  for (const m of next.libs) if (!libs.some((x) => x.id === m.id)) libs.push(m)
  return { list, libs, chapter: next.chapter || prev.chapter }
}

const roleLabels: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '次要',
}

/**
 * 新角色发现弹窗（T6-7.5 从 CreatePage 拆分）：自订阅 'new-characters-discovered'
 * 事件，自持弹窗/条目/选中/保存中状态，页面仅需渲染 <NewCharactersModal active={active} />。
 *
 * A8：小说子页**常驻挂载**（NovelPage 全 pane 同挂 + CSS 隐藏），本组件的事件通道因此
 * 一直在听。旧实现收到事件立即 setOpen(true)：带遮罩的 Modal 走 portal，父页的 CSS 隐藏
 * 盖不住它，于是「新角色发现」会凭空盖在**别的子页**上抢焦点（同页 novel:auto-reconstruct
 * 监听早已按 active 门控，此处是漏掉的一半）。故：
 *   - `active=false` 时事件**挂起**（合并进 pending，不丢弃——切页期间的角色发现不能永久丢）；
 *   - `active` 变真时再弹。
 * （不做「已在屏上的弹窗随 active 变假自动收起」：遮罩本身拦截指针事件，作者在该状态下
 * 根本点不动标签/板块，该路径不可达；多一条隐含路径反而让门控不可测。）
 */
const NewCharactersModal: React.FC<{ active?: boolean }> = ({ active = true }) => {
  const [open, setOpen] = useState(false)
  const [newCharsList, setNewCharsList] = useState<NewCharEntry[]>([])
  const [newCharsChapter, setNewCharsChapter] = useState(0)
  const [libMatches, setLibMatches] = useState<LibMatchEntry[]>([])
  const [adding, setAdding] = useState(false)
  // 事件回调只注册一次，故 active 走 ref 读最新值（同 CreatePage 的 activeRef 纪律）
  const activeRef = useRef(active)
  activeRef.current = active
  /** active=false 期间挂起的发现（切回创作页再弹，绝不丢弃） */
  const pendingRef = useRef<PendingDiscovery | null>(null)

  const applyDiscovery = useCallback((d: PendingDiscovery) => {
    setNewCharsList(d.list)
    setLibMatches(d.libs)
    setNewCharsChapter(d.chapter)
    setOpen(true)
  }, [])

  // 监听新角色发现事件（v4.421.0：退订走 subscribeWailsEvent 的「只摘除自己」
  // 清理函数——此前裸 EventsOff('new-characters-discovered') 会把同通道别人
  // 的监听一起炸掉，见 gaea/lib/wailsEvents.ts 事故纪律）
  useEffect(() => {
    const rt = window.runtime
    if (!rt?.EventsOn) return // 事件通道缺失时弹窗不可用
    const handler = (raw: unknown) => {
      const ev = raw as { detail?: NewCharactersPayload } | NewCharactersPayload | null | undefined
      const data = ((ev as { detail?: NewCharactersPayload } | null)?.detail || ev) as NewCharactersPayload | undefined
      if ((data?.characters?.length ?? 0) > 0 || (data?.libraryMatches?.length ?? 0) > 0) {
        const d = toDiscovery(data)
        if (activeRef.current) applyDiscovery(d)
        else pendingRef.current = mergeDiscovery(pendingRef.current, d)
      }
    }
    try {
      return subscribeWailsEvent(rt, NEW_CHARACTERS_CHANNEL, handler)
    } catch {
      // 订阅失败：无监听可退（不抛给 React 渲染链）
      return undefined
    }
  }, [applyDiscovery])

  // active 变真：把挂起的发现补弹出来（切页期间攒下的角色发现一笔都不能少）
  useEffect(() => {
    if (!active) return
    const pending = pendingRef.current
    if (!pending) return
    pendingRef.current = null
    applyDiscovery(pending)
  }, [active, applyDiscovery])

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
            await app.SaveCharactersBatch(JSON.stringify(selected))
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
          <div style={{ marginBottom: 4, fontSize: 12, fontWeight: 600, color: 'var(--color-primary)' }}>
            角色库已有同名角色（直接关联，不新建）
          </div>
          <div style={{
            maxHeight: 180, overflow: 'auto', border: '1px solid color-mix(in srgb, var(--color-primary) 25%, transparent)',
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
                  flex: 1, background: entry.name !== entry.original ? 'color-mix(in srgb, var(--color-warning) 8%, transparent)' : 'transparent',
                  border: entry.name !== entry.original ? '1px solid color-mix(in srgb, var(--color-warning) 40%, transparent)' : '1px solid transparent',
                  color: 'var(--color-text)', fontSize: 13
                }}
              />
              {entry.name !== entry.original && (
                <Button type="text" size="small"
                  onClick={() => setNewCharsList(prev => prev.map((c, j) => j === i ? { ...c, name: c.original } : c))}
                  style={{ fontSize: 10, padding: '0 2px', height: 20, color: 'var(--color-warning)' }}>
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
