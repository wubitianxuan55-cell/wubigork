// CharacterLibFilterBar.tsx — 角色档案库横向检索带
// 搜索、类型、聊天通道和数据运维收进同一条工具带，把整列空间让给人物画廊。
import React from 'react'
import { Button, Dropdown, Input } from 'antd'
import {
  SearchOutlined, FilterOutlined, ThunderboltOutlined, ImportOutlined,
  SyncOutlined, ToolOutlined, DownOutlined, CloseOutlined,
} from '@ant-design/icons'

const KIND_CHIPS = [
  { value: '', label: '全部' },
  { value: 'builtin', label: '内置' },
  { value: 'custom', label: '自定义' },
  { value: 'assistant', label: '助手' },
]

interface Props {
  query: string
  kind: string
  chatOnly: boolean
  total: number
  hasProject: boolean
  fillingAll: boolean
  onQueryChange: (q: string) => void
  onKindChange: (k: string) => void
  onChatOnlyChange: (v: boolean) => void
  onFillAll: () => void
  onImportProject: () => void
  onSyncProject: () => void
}

const CharacterLibFilterBar: React.FC<Props> = ({
  query, kind, chatOnly, total, hasProject, fillingAll,
  onQueryChange, onKindChange, onChatOnlyChange, onFillAll, onImportProject, onSyncProject,
}) => {
  const hasFilter = !!(query.trim() || kind || chatOnly)
  const dataMenu = {
    items: [
      {
        key: 'fill',
        icon: <ThunderboltOutlined />,
        label: fillingAll ? '补齐中…' : '一键补齐空缺字段',
        disabled: fillingAll,
        onClick: onFillAll,
      },
      {
        key: 'import',
        icon: <ImportOutlined />,
        label: '从当前小说导入',
        disabled: !hasProject,
        onClick: onImportProject,
      },
      {
        key: 'sync',
        icon: <SyncOutlined />,
        label: '同步引用到当前小说',
        disabled: !hasProject,
        onClick: onSyncProject,
      },
    ],
  }

  return (
    <section className="clib-filter-deck" aria-label="角色检索与筛选">
      <label className="clib-filter-search">
        <SearchOutlined className="clib-filter-search-icon" aria-hidden="true" />
        <Input
          allowClear
          variant="borderless"
          aria-label="搜索角色名称、标签或性格"
          placeholder="搜索角色名称、标签或性格…"
          value={query}
          onChange={e => onQueryChange(e.target.value)}
          className="clib-search"
        />
      </label>

      <div className="clib-filter-kinds" role="group" aria-label="按角色类型筛选">
        <FilterOutlined className="clib-filter-label-icon" aria-hidden="true" />
        {KIND_CHIPS.map(k => (
          <button
            key={k.value || 'all'}
            type="button"
            className={`clib-chip${kind === k.value ? ' is-active' : ''}`}
            aria-pressed={kind === k.value}
            onClick={() => onKindChange(k.value)}
          >
            {k.label}
          </button>
        ))}
      </div>

      <button
        type="button"
        className={`clib-chat-filter${chatOnly ? ' is-active' : ''}`}
        aria-pressed={chatOnly}
        onClick={() => onChatOnlyChange(!chatOnly)}
      >
        <i className="clib-chip-dot" aria-hidden="true" />
        仅可聊天
      </button>

      <div className="clib-filter-summary">
        <strong>{total}</strong>
        <span>份档案</span>
      </div>

      {hasFilter && (
        <button
          type="button"
          className="clib-filter-clear"
          aria-label="清除全部筛选"
          title="清除全部筛选"
          onClick={() => {
            onQueryChange('')
            onKindChange('')
            onChatOnlyChange(false)
          }}
        >
          <CloseOutlined aria-hidden="true" />
        </button>
      )}

      <Dropdown menu={dataMenu} trigger={['click']} placement="bottomRight">
        <Button className="clib-data-tools" size="small" icon={<ToolOutlined />}>
          数据工具 <DownOutlined />
        </Button>
      </Dropdown>
    </section>
  )
}

export default CharacterLibFilterBar
