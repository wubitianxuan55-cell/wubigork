// CharacterLibFilterBar.tsx — 角色档案库「检索与筛选」侧栏（左侧 v3-panel 玻璃面板）
// 搜索（聚焦光晕）+ 类型筛选 chips（激活项 = 主色容器 + 左缘光条）+ 通道 chip + 数据操作入口。
import React from 'react'
import { Button, Input } from 'antd'
import {
  SearchOutlined, FilterOutlined, ThunderboltOutlined, ImportOutlined, SyncOutlined,
} from '@ant-design/icons'

const KIND_CHIPS = [
  { value: '', label: '全部类型' },
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
}) => (
  <aside className="clib-side v3-panel" aria-label="角色检索与筛选">
    <div className="v3-panel-head">
      <FilterOutlined className="clib-side-head-icon" aria-hidden />
      <span className="v3-panel-title">检索与筛选</span>
    </div>

    <div className="clib-side-body">
      <div className="clib-side-group">
        <span className="clib-side-label">搜索</span>
        <div className="clib-search-wrap">
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="名称 / 标签 / 性格"
            value={query}
            onChange={e => onQueryChange(e.target.value)}
            className="clib-search"
          />
        </div>
      </div>

      <div className="clib-side-group">
        <span className="clib-side-label">类型</span>
        <div className="clib-chip-row" role="group" aria-label="按角色类型筛选">
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
      </div>

      <div className="clib-side-group">
        <span className="clib-side-label">通道</span>
        <button
          type="button"
          className={`clib-chip${chatOnly ? ' is-active' : ''}`}
          aria-pressed={chatOnly}
          onClick={() => onChatOnlyChange(!chatOnly)}
        >
          <i className="clib-chip-dot" aria-hidden />
          仅可聊天
        </button>
      </div>

      <div className="clib-side-stat">
        <span className="clib-stat-key">档案总数</span>
        <span className="clib-stat-val">{total}</span>
      </div>

      <div className="clib-side-ops">
        <span className="clib-side-label">数据操作</span>
        <Button size="small" block icon={<ThunderboltOutlined />} loading={fillingAll}
          onClick={onFillAll}
          title="为所有角色补齐空缺字段（基于已有设定推断）">
          {fillingAll ? '补齐中…' : '一键补齐'}
        </Button>
        <Button size="small" block icon={<ImportOutlined />} disabled={!hasProject}
          onClick={onImportProject}
          title="把当前项目的 characters.json 导入全局库">导入项目</Button>
        <Button size="small" block icon={<SyncOutlined />} disabled={!hasProject}
          onClick={onSyncProject}
          title="把项目引用写回 characters.json，小说 Agent 生效">同步到项目</Button>
      </div>
    </div>
  </aside>
)

export default CharacterLibFilterBar
