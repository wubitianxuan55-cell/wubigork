// sin/SinCastPicker.tsx — 从角色库挑角色（多选）。
//
// 复用面：头像走角色库既有 PortraitImg（远程 URL / data URL / 本地路径三态），
// 弹层走 antd Modal（与办公各面板同惯例）。选择只写「哪些角色进这个故事」，
// 不改角色库本身（角色库是共享资产，故事只是引用）。

import { useEffect, useMemo, useState } from 'react'
import { Button, Empty, Input, Modal, Tag } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import { PortraitImg } from '../../components/characterlib/PortraitImg'
import type { SinCastCharacter } from './useSinCast'

export interface SinCastPickerProps {
  open: boolean
  library: SinCastCharacter[]
  loading: boolean
  error: string
  selectedIds: string[]
  saving: boolean
  onSave: (ids: string[]) => void
  onClose: () => void
}

export function SinCastPicker({
  open, library, loading, error, selectedIds, saving, onSave, onClose,
}: SinCastPickerProps) {
  const [query, setQuery] = useState('')
  const [picks, setPicks] = useState<string[]>(selectedIds)

  // 每次打开以当前选择为起点（取消不留痕）
  useEffect(() => {
    if (open) {
      setPicks(selectedIds)
      setQuery('')
    }
  }, [open, selectedIds])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return library
    return library.filter((c) => (
      `${c.name} ${c.roleType ?? ''} ${(c.tags ?? []).join(' ')} ${c.personality ?? ''}`
        .toLowerCase()
        .includes(q)
    ))
  }, [library, query])

  const toggle = (id: string) => {
    setPicks((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  return (
    <Modal
      open={open}
      title="从角色库挑角色"
      width={720}
      onCancel={onClose}
      footer={[
        <Button key="clear" onClick={() => setPicks([])} disabled={picks.length === 0 || saving}>
          清空
        </Button>,
        <Button key="cancel" onClick={onClose} disabled={saving}>取消</Button>,
        <Button key="save" type="primary" loading={saving} onClick={() => onSave(picks)}>
          带入故事（{picks.length}）
        </Button>,
      ]}
    >
      <Input
        allowClear
        prefix={<SearchOutlined />}
        placeholder="搜名字 / 定位 / 标签 / 性格"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        style={{ marginBottom: 12 }}
      />
      {error && <div className="sin-cast-error">{error}</div>}
      {!loading && filtered.length === 0 && (
        <Empty
          description={library.length === 0 ? '角色库还没有角色，先去角色库建一个' : '没有匹配的角色'}
        />
      )}
      <div className="sin-cast-grid">
        {filtered.map((c) => {
          const picked = picks.includes(c.id)
          return (
            <button
              type="button"
              key={c.id}
              className={`sin-cast-item${picked ? ' is-picked' : ''}`}
              onClick={() => toggle(c.id)}
              aria-pressed={picked}
            >
              <div className="sin-cast-item-art">
                <PortraitImg src={c.portraitUrl} alt={c.name} />
              </div>
              <div className="sin-cast-item-body">
                <div className="sin-cast-item-name">
                  {c.name}
                  {c.roleType ? <span className="sin-cast-item-role">{c.roleType}</span> : null}
                </div>
                {c.personality && <div className="sin-cast-item-line">{c.personality}</div>}
                {c.background && <div className="sin-cast-item-line is-dim">{c.background}</div>}
                <div className="sin-cast-item-tags">
                  {(c.tags ?? []).slice(0, 3).map((t) => <Tag key={t}>{t}</Tag>)}
                </div>
              </div>
            </button>
          )
        })}
      </div>
    </Modal>
  )
}
