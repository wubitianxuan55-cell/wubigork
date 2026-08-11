import React, { useMemo, useState } from 'react'
import { Typography, Button, Modal, Input, Empty } from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, AppstoreOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import {
  TEMPLATES, CATEGORIES, CUSTOM_CATEGORY_ID, getCategory,
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

const CUSTOM_CAT: TemplateCategory = { id: CUSTOM_CATEGORY_ID, label: '我的模板', color: '#2dd4bf' }

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
      padding: '10px 12px',
      borderRadius: 'var(--radius-md)',
      border: '1px solid',
      borderColor: hovered ? accent : 'var(--border-subtle)',
      background: hovered ? withAlpha(accent, 0.07) : 'rgba(255,255,255,0.03)',
      cursor: 'pointer',
      transition: 'all 0.15s',
      display: 'flex', flexDirection: 'column', gap: 5, minHeight: 104,
    }}
  >
    {/* 标题行 + 自定义操作 */}
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 6 }}>
      <Typography.Text
        style={{ fontSize: 12.5, fontWeight: 600, color: C('color-text'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}
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
            style={{ padding: 0, height: 16, width: 18, color: '#f87171' }}
          />
        </span>
      )}
    </div>

    {/* 用途描述 */}
    {t.description && (
      <Typography.Text
        style={{
          fontSize: 10.5, color: C('color-text-secondary'), lineHeight: 1.4,
          display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' as const,
          overflow: 'hidden', minHeight: 29,
        }}
      >
        {t.description}
      </Typography.Text>
    )}

    {/* 分类 + 标签 + 画幅徽标 */}
    <div style={{ marginTop: 'auto', display: 'flex', alignItems: 'center', gap: 4, flexWrap: 'wrap' }}>
      {showCat && t.catLabel && (
        <span style={{
          fontSize: 9.5, padding: '1px 6px', borderRadius: 999,
          background: withAlpha(accent, 0.12), color: accent,
          border: `1px solid ${withAlpha(accent, 0.35)}`, lineHeight: 1.5,
        }}>
          {t.catLabel}
        </span>
      )}
      {(t.tags || []).slice(0, 3).map((tag) => (
        <span key={tag} style={{
          fontSize: 9.5, padding: '1px 6px', borderRadius: 999,
          border: '1px solid var(--border-subtle)', color: C('color-text-secondary'),
          background: 'rgba(255,255,255,0.04)', lineHeight: 1.5,
        }}>
          {tag}
        </span>
      ))}
      {t.size && (
        <span style={{
          fontSize: 9.5, padding: '1px 6px', borderRadius: 999, marginLeft: 'auto',
          background: withAlpha(accent, 0.12), color: accent,
          border: `1px solid ${withAlpha(accent, 0.35)}`, lineHeight: 1.5,
        }}>
          {t.size}
        </span>
      )}
    </div>
  </div>
)

const TemplatePickerModal: React.FC<Props> = ({
  open, onClose, customTemplates,
  onSelect, onAddCustom, onEditCustom, onDeleteCustom,
}) => {
  const defaultCat = CATEGORIES[0]?.id ?? 'enhance'
  const [activeCatId, setActiveCatId] = useState(defaultCat)
  const [search, setSearch] = useState('')
  const [hoverIdx, setHoverIdx] = useState<number | null>(null)

  // 分类行：预设 7 类 + 始终保留「我的模板」（空库时也可新建）
  const catRow: TemplateCategory[] = useMemo(() => {
    return [...CATEGORIES, CUSTOM_CAT]
  }, [])

  // 当前分类下的模板（自定义模板在「我的模板」分类）
  const baseTemplates: CardItem[] = useMemo(() => {
    if (activeCatId === CUSTOM_CATEGORY_ID) {
      return customTemplates.map((t) => ({ ...t, isCustom: true }))
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
    onSelect({ label: t.label, prompt: t.prompt, negative: t.negative })
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
      width={680}
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

      {/* 分类 pill（彩色圆点 + 主题色选中态） */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 12 }}>
        {catRow.map((cat) => {
          const selected = activeCatId === cat.id
          return (
            <button
              key={cat.id}
              type="button"
              onClick={() => { setActiveCatId(cat.id); setSearch('') }}
              style={{
                cursor: 'pointer', borderRadius: 999, fontSize: 12, padding: '3px 11px',
                border: '1px solid',
                borderColor: selected ? cat.color : 'var(--border-subtle)',
                background: selected ? withAlpha(cat.color, 0.14) : 'transparent',
                color: selected ? cat.color : C('color-text-secondary'),
                fontWeight: selected ? 600 : 400,
                display: 'inline-flex', alignItems: 'center', gap: 6, fontFamily: 'inherit',
                transition: 'all 0.15s',
              }}
            >
              <span style={{
                width: 7, height: 7, borderRadius: '50%', flexShrink: 0,
                background: cat.color, boxShadow: selected ? `0 0 5px ${cat.color}` : 'none',
              }} />
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
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(190px, 1fr))',
          gap: 8,
          maxHeight: 360,
          overflowY: 'auto',
          paddingRight: 4,
        }}>
          {filtered.map((t, i) => (
            <TemplateCard
              key={t.label + (t.id || '')}
              t={t}
              accent={(t as CardItem).catColor || activeColor}
              hovered={hoverIdx === i}
              showCat={searching}
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
