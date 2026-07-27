import React, { useState } from 'react'
import { Typography, Button, Modal, Tag } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { TEMPLATES, CATEGORIES, getAllCategories, type Template, type CustomTemplate } from '../../data/imageTemplates'

interface Props {
  open: boolean
  onClose: () => void
  customTemplates: CustomTemplate[]
  onSelect: (t: Template) => void
  onAddCustom: () => void
  onEditCustom: (t: CustomTemplate) => void
  onDeleteCustom: (id: string) => void
}

const TemplatePickerModal: React.FC<Props> = ({
  open, onClose, customTemplates,
  onSelect, onAddCustom, onEditCustom, onDeleteCustom,
}) => {
  const categories = getAllCategories(customTemplates.length)
  const [activeCat, setActiveCat] = useState(categories[0])
  const [hoverIdx, setHoverIdx] = useState<number | null>(null)

  // 合并预设 + 自定义
  const allTemplates: (Template & { id?: string; isCustom?: boolean })[] = [
    ...(TEMPLATES[activeCat] || []).map(t => ({ ...t })),
    ...(activeCat === '⭐ 自定义'
      ? customTemplates.map(t => ({ ...t, isCustom: true }))
      : []),
  ]

  const handleSelect = (t: Template & { id?: string }) => {
    // 去掉可能的 isCustom / id
    onSelect({ label: t.label, prompt: t.prompt, negative: t.negative })
    onClose()
  }

  return (
    <Modal
      title="📐 选择图片模板"
      open={open}
      onCancel={onClose}
      footer={null}
      width={560}
      styles={{
        body: { background: C('color-bg-container'), padding: '16px 20px' },
        header: { background: C('color-bg-container') },
      }}
    >
      {/* 分类 pill */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 14 }}>
        {categories.map(cat => (
          <Tag
            key={cat}
            onClick={() => { setActiveCat(cat); setHoverIdx(null) }}
            style={{
              cursor: 'pointer',
              borderRadius: 'var(--radius-sm)',
              fontSize: 12,
              padding: '2px 10px',
              border: activeCat === cat
                ? '1px solid var(--color-primary)'
                : '1px solid var(--border-subtle)',
              background: activeCat === cat
                ? 'rgba(99, 102, 241, 0.12)'
                : 'transparent',
              color: activeCat === cat ? 'var(--color-primary)' : C('color-text-secondary'),
              fontWeight: activeCat === cat ? 600 : 400,
            }}
          >
            {cat}
          </Tag>
        ))}
      </div>

      {/* 自适应网格 */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))',
        gap: 8,
        maxHeight: 320,
        overflowY: 'auto',
        paddingRight: 4,
      }}>
        {allTemplates.map((t, i) => (
          <div
            key={t.label + (t.id || '')}
            onClick={() => handleSelect(t)}
            onMouseEnter={() => setHoverIdx(i)}
            onMouseLeave={() => setHoverIdx(null)}
            style={{
              padding: '10px 12px',
              borderRadius: 'var(--radius-md)',
              border: '1px solid var(--border-subtle)',
              background: hoverIdx === i
                ? 'rgba(99, 102, 241, 0.08)'
                : 'rgba(255,255,255,0.03)',
              cursor: 'pointer',
              transition: 'all 0.15s',
              position: 'relative' as const,
            }}
          >
            {/* 标签 + 自定义按钮组 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
              <Typography.Text style={{ fontSize: 12, fontWeight: 600, color: C('color-text') }}>
                {t.label}
              </Typography.Text>
              {t.isCustom && (
                <span style={{ display: 'flex', gap: 4 }}>
                  <Button
                    type="text" size="small" icon={<EditOutlined style={{ fontSize: 10 }} />}
                    onClick={(e) => { e.stopPropagation(); onEditCustom(t as CustomTemplate) }}
                    style={{ padding: 0, height: 16, color: C('color-text-secondary') }}
                  />
                  <Button
                    type="text" size="small" icon={<DeleteOutlined style={{ fontSize: 10 }} />}
                    onClick={(e) => { e.stopPropagation(); onDeleteCustom((t as CustomTemplate).id) }}
                    style={{ padding: 0, height: 16, color: '#f87171' }}
                  />
                </span>
              )}
            </div>

            {/* hover 时显示 prompt 预览 */}
            <Typography.Text
              style={{
                fontSize: 10,
                color: C('color-text-secondary'),
                display: '-webkit-box',
                WebkitLineClamp: hoverIdx === i ? undefined : 2,
                WebkitBoxOrient: 'vertical' as const,
                overflow: 'hidden',
                lineHeight: '1.4',
              }}
            >
              {t.prompt}
            </Typography.Text>
          </div>
        ))}

        {/* 自定义模板快捷添加 */}
        {activeCat === '⭐ 自定义' && (
          <div
            onClick={onAddCustom}
            style={{
              padding: '10px 12px',
              borderRadius: 'var(--radius-md)',
              border: '1px dashed var(--border-subtle)',
              background: 'rgba(255,255,255,0.02)',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 6,
              minHeight: 60,
            }}
          >
            <PlusOutlined style={{ color: C('color-text-secondary'), fontSize: 14 }} />
            <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>
              新建模板
            </Typography.Text>
          </div>
        )}
      </div>
    </Modal>
  )
}

export default TemplatePickerModal
