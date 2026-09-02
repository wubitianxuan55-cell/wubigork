import React, { useMemo, useState } from 'react'
import { Typography, Button, Modal, Input, Empty } from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, AppstoreOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import {
  TEMPLATES, CATEGORIES, CUSTOM_CATEGORY_ID, ALL_CATEGORY_ID, ALL_CATEGORY, getCategory,
  type Template, type CustomTemplate, type TemplateCategory,
} from '../../data/imageTemplates'

interface Props {
  open: boolean
  onClose: () => void
  customTemplates: CustomTemplate[]
  onSelect: (t: Template) => void
  onAddCustom: () => void
  onEditCustom: (t: CustomTemplate) => void
  onDeleteCustom: (id: string) => void
}

/** hex → rgba，用于分类主题色淡底 */
function withAlpha(hex: string, a: number): string {
  const n = parseInt(hex.replace('#', ''), 16)
  if (Number.isNaN(n) || hex.length !== 7) return `rgba(45, 212, 191, ${a})`
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${a})`
}

const CUSTOM_CAT: TemplateCategory = { id: CUSTOM_CATEGORY_ID, label: '我的模板', color: 'var(--color-primary)' }

type CardItem = Template & {
  id?: string
  isCustom?: boolean
  catLabel?: string
  catColor?: string
}

const TemplateCard: React.FC<{
  t: CardItem
  accent: string
  hovered: boolean
  showCat: boolean
  onSelect: () => void
  onHover: () => void
  onLeave: () => void
  onEdit?: () => void
  onDelete?: () => void
}> = ({ t, accent, hovered, showCat, onSelect, onHover, onLeave, onEdit, onDelete }) => (
  <div
    onClick={onSelect}
    onMouseEnter={onHover}
    onMouseLeave={onLeave}
    style={{
      position: 'relative',
      padding: '12px 14px',
      borderRadius: 12,
      border: '1px solid',
      borderColor: hovered ? accent : 'var(--border-subtle)',
      background: hovered ? withAlpha(accent, 0.06) : 'var(--color-surface-container)',
      boxShadow: hovered ? '0 8px 20px rgba(0,0,0,0.10)' : '0 1px 3px rgba(0,0,0,0.05)',
      cursor: 'pointer',
      transition: 'all 0.15s ease',
      display: 'flex', flexDirection: 'column', gap: 7, minHeight: 96,
    }}
  >
    {/* 图标（圆角方底）+ 标题 + 自定义操作 */}
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
      <span style={{
        width: 28, height: 28, borderRadius: 8, flexShrink: 0,
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        background: withAlpha(accent, 0.12), fontSize: 15, lineHeight: 1,
      }}>
        {t.icon || <AppstoreOutlined style={{ fontSize: 13, color: accent }} />}
      </span>
      <Typography.Text
        strong
        style={{
          flex: 1, minWidth: 0, fontSize: 13, color: C('color-text'),
          whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
        }}
      >
        {t.label}
      </Typography.Text>
      {t.isCustom && (
        <span style={{ display: 'flex', gap: 2, flexShrink: 0 }}>
          <Button
            type="text" size="small" icon={<EditOutlined style={{ fontSize: 10 }} />}
            onClick={(e) => { e.stopPropagation(); onEdit?.() }}
            style={{ padding: 0, height: 16, width: 18, color: C('color-text-secondary') }}
          />
          <Button
            type="text" size="small" icon={<DeleteOutlined style={{ fontSize: 10 }} />}
            onClick={(e) => { e.stopPropagation(); onDelete?.() }}
            style={{ padding: 0, height: 16, width: 18, color: 'var(--color-destructive)' }}
          />
        </span>
      )}
    </div>

    {/* 用途描述 */}
    {t.description && (
      <Typography.Text
        style={{
          fontSize: 11, color: C('color-text-secondary'), lineHeight: 1.45,
          display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' as const,
          overflow: 'hidden', minHeight: 32,
        }}
      >
        {t.description}
      </Typography.Text>
    )}

    {showCat && t.catLabel && (
      <span style={{
        position: 'absolute', right: 10, bottom: 8,
        fontSize: 9.5, padding: '1px 7px', borderRadius: 999,
        background: withAlpha(accent, 0.12), color: accent,
        border: `1px solid ${withAlpha(accent, 0.35)}`, lineHeight: 1.5,
      }}>
        {t.catLabel}
      </span>
    )}
  </div>
)

const TemplatePickerModal: React.FC<Props> = ({
  open, onClose, customTemplates,
  onSelect, onAddCustom, onEditCustom, onDeleteCustom,
}) => {
  const defaultCat = ALL_CATEGORY_ID
  const [activeCatId, setActiveCatId] = useState(defaultCat)
  const [search, setSearch] = useState('')
  const [hoverIdx, setHoverIdx] = useState<number | null>(null)

  // 分类行：「全部模板」+ 预设 19 类 + 始终保留「我的模板」（空库时也可新建）
  const catRow: TemplateCategory[] = useMemo(() => {
    return [ALL_CATEGORY, ...CATEGORIES, CUSTOM_CAT]
  }, [])

  // 当前分类下的模板（自定义模板在「我的模板」分类）
  const baseTemplates: CardItem[] = useMemo(() => {
    if (activeCatId === CUSTOM_CATEGORY_ID) {
      return customTemplates.map((t) => ({ ...t, isCustom: true }))
    }
    if (activeCatId === ALL_CATEGORY_ID) {
      const all: CardItem[] = []
      for (const cat of CATEGORIES) {
        for (const t of TEMPLATES[cat.id] || []) {
          all.push({ ...t, catLabel: cat.label, catColor: cat.color })
        }
      }
      for (const t of customTemplates) {
        all.push({ ...t, isCustom: true, catLabel: CUSTOM_CAT.label, catColor: CUSTOM_CAT.color })
      }
      return all
    }
    return (TEMPLATES[activeCatId] || []).map((t) => ({ ...t }))
  }, [activeCatId, customTemplates])

  // 搜索：跨全库命中名称 / 描述 / 标签 / 提示词
  const searchHits = useMemo(() => {
    const kw = search.trim().toLowerCase()
    if (!kw) return null
    const matches = (t: Template) =>
      t.label.toLowerCase().includes(kw)
      || (t.description || '').toLowerCase().includes(kw)
      || (t.tags || []).some((tag) => tag.toLowerCase().includes(kw))
      || t.prompt.toLowerCase().includes(kw)
    const hits: CardItem[] = []
    for (const cat of CATEGORIES) {
      for (const t of TEMPLATES[cat.id] || []) {
        if (matches(t)) hits.push({ ...t, catLabel: cat.label, catColor: cat.color })
      }
    }
    for (const t of customTemplates) {
      if (matches(t)) hits.push({ ...t, isCustom: true, catLabel: CUSTOM_CAT.label, catColor: CUSTOM_CAT.color })
    }
    return hits
  }, [search, customTemplates])

  const searching = search.trim().length > 0
  const filtered: CardItem[] = searching ? (searchHits || []) : baseTemplates

  const handleSelect = (t: Template) => {
    // 保留模板推荐画幅（size），应用模板时同步画幅（刀3）
    onSelect({ label: t.label, prompt: t.prompt, negative: t.negative, size: t.size })
    onClose()
  }

  const activeColor = getCategory(activeCatId)?.color
    ?? CUSTOM_CAT.color

  return (
    <Modal
      title={
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8, fontSize: 15, fontWeight: 600 }}>
          <AppstoreOutlined style={{ color: 'var(--color-primary)' }} />
          选择图片模板
        </span>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{
        body: { background: 'transparent', padding: '12px 20px 20px' },
        header: { background: 'transparent' },
      }}
    >
      {/* 搜索框 */}
      <Input
        allowClear
        prefix={<SearchOutlined style={{ color: C('color-text-secondary'), fontSize: 13 }} />}
        placeholder="搜索模板名称、用途、标签或提示词…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        style={{
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-subtle)',
          color: 'var(--color-text)',
          borderRadius: 10, fontSize: 13, marginBottom: 10,
        }}
      />

      {/* 分类页签（参考 herdsman：横向滚动灰色胶囊，选中加深） */}
      <div className="ig-tpl-tabs" role="tablist" aria-label="模板分类">
        {catRow.map((cat) => {
          const selected = activeCatId === cat.id
          return (
            <button
              key={cat.id}
              type="button"
              role="tab"
              aria-selected={selected}
              onClick={() => { setActiveCatId(cat.id); setSearch('') }}
              className={`ig-tpl-tab${selected ? ' is-active' : ''}`}
            >
              {cat.label}
            </button>
          )
        })}
      </div>

      {/* 模板卡片网格 */}
      {filtered.length === 0 && !(activeCatId === CUSTOM_CATEGORY_ID && !searching) ? (
        <div style={{ padding: '30px 0' }}>
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <span style={{ color: C('color-text-secondary'), fontSize: 12 }}>
                {searching ? '没有匹配的模板' : '当前分类暂无模板'}
              </span>
            }
          >
            {searching && (
              <Button size="small" onClick={() => setSearch('')} style={{ borderRadius: 999, fontSize: 12 }}>
                清空搜索
              </Button>
            )}
          </Empty>
        </div>
      ) : (
        <div className="ig-tpl-grid">
          {filtered.map((t, i) => (
            <TemplateCard
              key={t.label + (t.id || '')}
              t={t}
              accent={(t as CardItem).catColor || activeColor}
              hovered={hoverIdx === i}
              showCat={searching || activeCatId === ALL_CATEGORY_ID}
              onSelect={() => handleSelect(t)}
              onHover={() => setHoverIdx(i)}
              onLeave={() => setHoverIdx(null)}
              onEdit={t.isCustom ? () => onEditCustom(t as CustomTemplate) : undefined}
              onDelete={t.isCustom ? () => onDeleteCustom((t as CustomTemplate).id) : undefined}
            />
          ))}

          {/* 我的模板：新建卡 */}
          {!searching && activeCatId === CUSTOM_CATEGORY_ID && (
            <div
              onClick={onAddCustom}
              style={{
                padding: '10px 12px', borderRadius: 'var(--radius-md)',
                border: '1px dashed var(--border-subtle)',
                background: 'rgba(255,255,255,0.02)', cursor: 'pointer',
                display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6, minHeight: 104,
                transition: 'all 0.15s',
              }}
            >
              <PlusOutlined style={{ color: C('color-text-secondary'), fontSize: 14 }} />
              <Typography.Text style={{ fontSize: 11.5, color: C('color-text-secondary') }}>
                新建模板
              </Typography.Text>
            </div>
          )}
        </div>
      )}
    </Modal>
  )
}

export default TemplatePickerModal
