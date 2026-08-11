import React, { useState } from 'react'
import { Typography, Space, Tag, Modal, Input, Spin, Empty } from 'antd'
import { SearchOutlined, FileTextOutlined, UserOutlined, UnorderedListOutlined } from '@ant-design/icons'

import { C } from '../utils/theme'

interface SearchResult {
  file: string
  context: string
}

interface SearchModalProps {
  open: boolean
  onClose: () => void
}

const categoryIcons: Record<string, React.ReactNode> = {
  chapters: <FileTextOutlined style={{ color: '#4ade80' }} />,
  characters: <UserOutlined style={{ color: '#60a5fa' }} />,
}
const categoryLabels: Record<string, string> = {
  chapters: '章节', characters: '角色',
}

/** 在文本中用 <mark> 高亮关键词 */
function highlightText(text: string, query: string): React.ReactNode {
  if (!query) return text
  const idx = text.toLowerCase().indexOf(query.toLowerCase())
  if (idx < 0) return text
  return (
    <>
      {text.slice(0, idx)}
      <mark style={{ background: '#f59e0b33', color: '#f59e0b', borderRadius: 2, padding: '0 2px' }}>
        {text.slice(idx, idx + query.length)}
      </mark>
      {text.slice(idx + query.length)}
    </>
  )
}

const SearchModal: React.FC<SearchModalProps> = ({ open, onClose }) => {
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [results, setResults] = useState<Record<string, SearchResult[]>>({})
  const [searched, setSearched] = useState(false)
  const [filterCategory, setFilterCategory] = useState<string | null>(null)

  const handleSearch = async (value: string) => {
    const q = value.trim()
    if (!q) { setResults({}); setSearched(false); return }
    setLoading(true)
    setSearched(true)
    setFilterCategory(null)
    try {
      // @ts-ignore
      const data = await window.go.app.App.Search(q)
      setResults(data || {})
    } catch (_) {
      setResults({})
    } finally { setLoading(false) }
  }

  const totalResults = Object.values(results).flat().length
  const categories = Object.keys(results).filter((k) => results[k].length > 0)
  const filteredCategories = filterCategory
    ? categories.filter((c) => c === filterCategory)
    : categories

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><SearchOutlined style={{ color: C('color-primary'), marginRight: 8 }} />全文搜索</span>}
      open={open}
      onCancel={() => { onClose(); setQuery(''); setResults({}); setSearched(false) }}
      footer={null}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={680}
      styles={{
        body: { maxHeight: '70vh', overflow: 'auto' },
      }}
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Input.Search
          placeholder="搜索章节内容、角色、大纲..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onSearch={handleSearch}
          onPressEnter={() => handleSearch(query)}
          size="large"
          allowClear
          style={{ background: C('color-bg-layout') }}
        />

        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : searched ? (
          totalResults === 0 ? (
            <Empty description={`未找到「${query}」`} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          ) : (
            <>
              {/* 分类标签过滤 */}
              <Space size={4} wrap>
                <Tag
                  color={filterCategory === null ? 'blue' : 'default'}
                  style={{ cursor: 'pointer', fontSize: 11 }}
                  onClick={() => setFilterCategory(null)}
                >
                  全部 ({totalResults})
                </Tag>
                {categories.map((cat) => (
                  <Tag
                    key={cat}
                    color={filterCategory === cat ? 'blue' : 'default'}
                    style={{ cursor: 'pointer', fontSize: 11 }}
                    onClick={() => setFilterCategory(cat === filterCategory ? null : cat)}
                  >
                    {categoryIcons[cat]} {categoryLabels[cat] || cat} ({results[cat].length})
                  </Tag>
                ))}
              </Space>

              {/* 结果列表 */}
              {filteredCategories.map((cat) => (
                <div key={cat}>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 12, display: 'block', marginBottom: 6 }}>
                    {categoryIcons[cat]} {categoryLabels[cat] || cat} · {results[cat].length} 条
                  </Typography.Text>
                  <Space direction="vertical" size={6} style={{ width: '100%' }}>
                    {results[cat].map((r, i) => (
                      <div
                        key={i}
                        style={{
                          background: C('color-bg-layout'), borderRadius: 6,
                          padding: '8px 12px', border: '1px solid ' + C('color-border'),
                        }}
                      >
                        <Typography.Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 4 }}>
                          {r.file}
                        </Typography.Text>
                        <Typography.Text style={{ color: C('color-text'), fontSize: 12, lineHeight: 1.7 }}>
                          {highlightText(r.context, query)}
                        </Typography.Text>
                      </div>
                    ))}
                  </Space>
                </div>
              ))}
            </>
          )
        ) : (
          <div style={{ textAlign: 'center', padding: 20, color: C('color-text-secondary'), fontSize: 12 }}>
            输入关键词，搜索小说中的全部内容
          </div>
        )}
      </Space>
    </Modal>
  )
}

export default SearchModal
