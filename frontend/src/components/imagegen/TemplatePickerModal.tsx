import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Button, Empty, Input, Modal, Popconfirm } from 'antd'
import {
  AppstoreOutlined,
  ArrowRightOutlined,
  CloseOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import {
  TEMPLATES,
  CATEGORIES,
  CUSTOM_CATEGORY_ID,
  ALL_CATEGORY_ID,
  ALL_CATEGORY,
  type Template,
  type CustomTemplate,
  type TemplateCategory,
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

const CUSTOM_CAT: TemplateCategory = {
  id: CUSTOM_CATEGORY_ID,
  label: '我的模板',
  color: 'var(--color-primary)',
}

const CORE_CATEGORY_IDS = new Set([
  'enhance',
  'style',
  'photo',
  'creation',
  'compose',
  'light',
  'scene',
])

type CardItem = Template & {
  id?: string
  isCustom?: boolean
  catLabel?: string
  catColor?: string
}

const templateKey = (t: CardItem, fallbackCategory = 'all'): string =>
  t.id || `${t.catLabel || fallbackCategory}:${t.label}`

const TemplateCard: React.FC<{
  template: CardItem
  index: number
  selected: boolean
  showCategory: boolean
  onSelect: () => void
  onApply: () => void
}> = ({ template, index, selected, showCategory, onSelect, onApply }) => {
  const accent = template.catColor || 'var(--color-primary)'
  return (
    <button
      type="button"
      className={`igt-template-card${selected ? ' is-selected' : ''}`}
      aria-pressed={selected}
      onClick={onSelect}
      onDoubleClick={onApply}
      style={{ '--igt-accent': accent } as React.CSSProperties}
      title="单击查看详情，双击直接套用"
    >
      <span className="igt-card-index" aria-hidden>
        {String(index + 1).padStart(2, '0')}
      </span>
      <span className="igt-card-topline">
        <span className="igt-card-category">
          <i aria-hidden />
          {showCategory ? template.catLabel : template.size || '自由画幅'}
        </span>
        <span>{template.size || '自由画幅'}</span>
      </span>
      <strong>{template.label}</strong>
      <span className="igt-card-description">
        {template.description || template.prompt}
      </span>
      <span className="igt-card-tags">
        {(template.tags || [template.size || '通用']).slice(0, 2).map((tag) => (
          <em key={tag}>{tag}</em>
        ))}
      </span>
      <span className="igt-card-arrow" aria-hidden>
        <ArrowRightOutlined />
      </span>
    </button>
  )
}

const TemplatePickerModal: React.FC<Props> = ({
  open,
  onClose,
  customTemplates,
  onSelect,
  onAddCustom,
  onEditCustom,
  onDeleteCustom,
}) => {
  const [activeCatId, setActiveCatId] = useState(ALL_CATEGORY_ID)
  const [search, setSearch] = useState('')
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const resultsRef = useRef<HTMLDivElement>(null)

  const totalTemplates = useMemo(
    () => CATEGORIES.reduce((sum, category) => sum + (TEMPLATES[category.id]?.length || 0), customTemplates.length),
    [customTemplates.length],
  )

  const coreCategories = useMemo(
    () => CATEGORIES.filter((category) => CORE_CATEGORY_IDS.has(category.id)),
    [],
  )
  const collectionCategories = useMemo(
    () => CATEGORIES.filter((category) => !CORE_CATEGORY_IDS.has(category.id)),
    [],
  )

  const categoryCount = (id: string): number => {
    if (id === ALL_CATEGORY_ID) return totalTemplates
    if (id === CUSTOM_CATEGORY_ID) return customTemplates.length
    return TEMPLATES[id]?.length || 0
  }

  const activeCategory = activeCatId === ALL_CATEGORY_ID
    ? ALL_CATEGORY
    : activeCatId === CUSTOM_CATEGORY_ID
      ? CUSTOM_CAT
      : CATEGORIES.find((category) => category.id === activeCatId) || ALL_CATEGORY

  const baseTemplates: CardItem[] = useMemo(() => {
    if (activeCatId === CUSTOM_CATEGORY_ID) {
      return customTemplates.map((template) => ({
        ...template,
        isCustom: true,
        catLabel: CUSTOM_CAT.label,
        catColor: CUSTOM_CAT.color,
      }))
    }
    if (activeCatId === ALL_CATEGORY_ID) {
      const all: CardItem[] = []
      for (const category of CATEGORIES) {
        for (const template of TEMPLATES[category.id] || []) {
          all.push({
            ...template,
            catLabel: category.label,
            catColor: category.color,
          })
        }
      }
      for (const template of customTemplates) {
        all.push({
          ...template,
          isCustom: true,
          catLabel: CUSTOM_CAT.label,
          catColor: CUSTOM_CAT.color,
        })
      }
      return all
    }
    const category = CATEGORIES.find((item) => item.id === activeCatId)
    return (TEMPLATES[activeCatId] || []).map((template) => ({
      ...template,
      catLabel: category?.label,
      catColor: category?.color,
    }))
  }, [activeCatId, customTemplates])

  const searchHits = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    if (!keyword) return null
    const matches = (template: Template) =>
      template.label.toLowerCase().includes(keyword)
      || (template.description || '').toLowerCase().includes(keyword)
      || (template.tags || []).some((tag) => tag.toLowerCase().includes(keyword))
      || template.prompt.toLowerCase().includes(keyword)
    const hits: CardItem[] = []
    for (const category of CATEGORIES) {
      for (const template of TEMPLATES[category.id] || []) {
        if (matches(template)) {
          hits.push({
            ...template,
            catLabel: category.label,
            catColor: category.color,
          })
        }
      }
    }
    for (const template of customTemplates) {
      if (matches(template)) {
        hits.push({
          ...template,
          isCustom: true,
          catLabel: CUSTOM_CAT.label,
          catColor: CUSTOM_CAT.color,
        })
      }
    }
    return hits
  }, [search, customTemplates])

  const searching = search.trim().length > 0
  const filtered = searching ? searchHits || [] : baseTemplates
  const selected = filtered.find((template) =>
    templateKey(template, activeCatId) === selectedKey,
  ) || filtered[0] || null
  const selectedAccent = selected?.catColor || activeCategory.color
  const showNewCard = !searching && activeCatId === CUSTOM_CATEGORY_ID

  useEffect(() => {
    setSelectedKey(null)
    const results = resultsRef.current
    if (!results) return
    if (typeof results.scrollTo === 'function') results.scrollTo({ top: 0 })
    else results.scrollTop = 0
  }, [activeCatId, search])

  const chooseCategory = (id: string) => {
    setActiveCatId(id)
    setSearch('')
  }

  const applyTemplate = (template: Template) => {
    onSelect({
      label: template.label,
      prompt: template.prompt,
      negative: template.negative,
      size: template.size,
    })
    onClose()
  }

  const collectionTitle = searching
    ? '搜索结果'
    : activeCatId === ALL_CATEGORY_ID
      ? '全库浏览'
      : activeCategory.label

  const renderCategory = (category: TemplateCategory) => {
    const selectedCategory = activeCatId === category.id
    return (
      <button
        key={category.id}
        type="button"
        role="tab"
        aria-selected={selectedCategory}
        aria-controls="igt-template-results"
        className={`igt-category${selectedCategory ? ' is-active' : ''}`}
        onClick={() => chooseCategory(category.id)}
        style={{ '--igt-category': category.color } as React.CSSProperties}
      >
        <i aria-hidden />
        <span>{category.label}</span>
        <b>{categoryCount(category.id)}</b>
      </button>
    )
  }

  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      closable={false}
      width={1120}
      className="ig-template-library-modal"
      rootClassName="ig-template-library-root"
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      style={{ top: 24 }}
      aria-labelledby="igt-library-title"
    >
      <div className="igt-library">
        <header className="igt-library-head">
          <div className="igt-library-identity">
            <span className="igt-library-mark" aria-hidden>
              <AppstoreOutlined />
            </span>
            <div>
              <span className="igt-kicker">Dream atelier · Prompt atlas</span>
              <h2 id="igt-library-title">绘梦模板库</h2>
            </div>
          </div>
          <p>从成熟的画面语言出发，再把模板改成属于你的那一幕。</p>
          <div className="igt-library-total">
            <strong>{totalTemplates}</strong>
            <span>组提示词</span>
          </div>
          <button type="button" className="igt-library-close" aria-label="关闭模板库" onClick={onClose}>
            <CloseOutlined />
          </button>
        </header>

        <div className="igt-library-body">
          <nav className="igt-category-panel" aria-label="模板分类">
            <div className="igt-category-scroll">
              <div className="igt-category-group" role="tablist" aria-label="全部模板">
                <button
                  type="button"
                  role="tab"
                  aria-selected={activeCatId === ALL_CATEGORY_ID}
                  aria-controls="igt-template-results"
                  className={`igt-category igt-category--all${activeCatId === ALL_CATEGORY_ID ? ' is-active' : ''}`}
                  onClick={() => chooseCategory(ALL_CATEGORY_ID)}
                >
                  <i aria-hidden />
                  <span>全部模板</span>
                  <b>{categoryCount(ALL_CATEGORY_ID)}</b>
                </button>
              </div>

              <div className="igt-category-group" role="tablist" aria-label="基础与风格">
                <span className="igt-category-label">基础与风格</span>
                {coreCategories.map(renderCategory)}
              </div>

              <div className="igt-category-group" role="tablist" aria-label="场景与视觉">
                <span className="igt-category-label">场景与视觉</span>
                {collectionCategories.map(renderCategory)}
              </div>

              <div className="igt-category-group igt-category-group--mine" role="tablist" aria-label="个人收藏">
                <span className="igt-category-label">个人收藏</span>
                {renderCategory(CUSTOM_CAT)}
              </div>
            </div>

            <Button
              type="primary"
              icon={<PlusOutlined />}
              className="igt-new-template"
              onClick={onAddCustom}
            >
              新建模板
            </Button>
          </nav>

          <section className="igt-results" aria-labelledby="igt-results-title">
            <div className="igt-search-row">
              <Input
                allowClear
                prefix={<SearchOutlined />}
                placeholder="搜索名称、用途、标签或提示词"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                aria-label="搜索模板"
                className="igt-search"
              />
              <span aria-live="polite">{filtered.length} 项</span>
            </div>

            <header className="igt-results-head">
              <div>
                <span className="igt-kicker">{searching ? 'Search result' : 'Collection'}</span>
                <h3 id="igt-results-title">{collectionTitle}</h3>
              </div>
              <small>单击查看详情 · 双击直接套用</small>
            </header>

            <div
              id="igt-template-results"
              ref={resultsRef}
              className="igt-results-scroll"
              role="tabpanel"
            >
              {filtered.length === 0 && !showNewCard ? (
                <div className="igt-empty">
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={searching ? '没有找到匹配的模板' : '当前分类还没有模板'}
                  >
                    {searching && (
                      <Button size="small" onClick={() => setSearch('')}>
                        清空搜索
                      </Button>
                    )}
                  </Empty>
                </div>
              ) : (
                <div className="igt-card-grid">
                  {filtered.map((template, index) => {
                    const key = templateKey(template, activeCatId)
                    return (
                      <TemplateCard
                        key={`${key}-${index}`}
                        template={template}
                        index={index}
                        selected={selected ? templateKey(selected, activeCatId) === key : false}
                        showCategory={searching || activeCatId === ALL_CATEGORY_ID}
                        onSelect={() => setSelectedKey(key)}
                        onApply={() => applyTemplate(template)}
                      />
                    )
                  })}

                  {showNewCard && (
                    <button
                      type="button"
                      className="igt-template-card igt-template-card--new"
                      onClick={onAddCustom}
                    >
                      <span className="igt-new-icon" aria-hidden><PlusOutlined /></span>
                      <strong>写下自己的模板</strong>
                      <span className="igt-card-description">
                        保存常用提示词、负向词与推荐画幅，建立自己的风格库。
                      </span>
                      <span className="igt-card-arrow" aria-hidden><ArrowRightOutlined /></span>
                    </button>
                  )}
                </div>
              )}
            </div>
          </section>

          <aside
            className="igt-preview"
            aria-label="模板详情"
            style={{ '--igt-accent': selectedAccent } as React.CSSProperties}
          >
            {selected ? (
              <>
                <div className="igt-preview-art" aria-hidden>
                  <span>{selected.catLabel || activeCategory.label}</span>
                  <strong>{Array.from(selected.label)[0] || '绘'}</strong>
                  <small>{selected.size || '自由画幅'}</small>
                </div>
                <div className="igt-preview-copy">
                  <span className="igt-preview-category">
                    {selected.catLabel || activeCategory.label}
                  </span>
                  <h3>{selected.label}</h3>
                  <p>{selected.description || '从这组提示词开始，继续补充主体与环境。'}</p>
                </div>

                <div className="igt-preview-meta">
                  <span>{selected.size || '自由画幅'}</span>
                  <span>{selected.tags?.length || 0} 个标签</span>
                  <span>{selected.isCustom ? '自定义' : '内置'}</span>
                </div>

                {(selected.tags?.length || 0) > 0 && (
                  <div className="igt-preview-tags">
                    {selected.tags?.map((tag) => <span key={tag}>{tag}</span>)}
                  </div>
                )}

                <section className="igt-prompt-preview" aria-labelledby="igt-prompt-title">
                  <header>
                    <span id="igt-prompt-title">提示词</span>
                    <small>{selected.prompt.length} 字</small>
                  </header>
                  <p>{selected.prompt}</p>
                </section>

                {selected.negative && (
                  <details className="igt-negative-preview">
                    <summary>负向提示词</summary>
                    <p>{selected.negative}</p>
                  </details>
                )}

                <div className="igt-preview-actions">
                  <Button
                    type="primary"
                    icon={<ArrowRightOutlined />}
                    iconPosition="end"
                    className="igt-apply"
                    onClick={() => applyTemplate(selected)}
                  >
                    套用此模板
                  </Button>
                  {selected.isCustom && (
                    <>
                      <Button
                        icon={<EditOutlined />}
                        onClick={() => onEditCustom(selected as CustomTemplate)}
                      >
                        编辑
                      </Button>
                      <Popconfirm
                        title="删除这个自定义模板？"
                        okText="删除"
                        cancelText="取消"
                        okButtonProps={{ danger: true, 'data-testid': 'imgtpl-delete-confirm' }}
                        onConfirm={() => onDeleteCustom((selected as CustomTemplate).id)}
                      >
                        <Button danger icon={<DeleteOutlined />}>
                          删除
                        </Button>
                      </Popconfirm>
                    </>
                  )}
                </div>
                <p className="igt-preview-note">套用后会同步推荐画幅与负向提示词。</p>
              </>
            ) : (
              <div className="igt-preview-empty">
                <AppstoreOutlined aria-hidden />
                <strong>选择一个模板</strong>
                <span>在这里查看完整提示词、画幅与标签。</span>
              </div>
            )}
          </aside>
        </div>
      </div>
    </Modal>
  )
}

export default TemplatePickerModal
