import React, { useEffect, useState } from 'react'
import { Modal, Button, Input, Space, Tag, Typography, Alert, Spin } from 'antd'
import { C } from '../../utils/theme'
import { app } from '../../gaea/lib/bridge'
import type { NovelTechnique, NovelTechniqueSearchResult } from '../../gaea/lib/bridge/novel'

interface TechniquePanelProps {
  open: boolean
  onClose: () => void
}

/**
 * 技法库面板（webnovel-writer 九表思想机制重推导蒸馏批 N11；GPL 只重推导，
 * 账本内容全自写）：零 LLM 确定性检索——关键词/分类名命中计分。只读不落盘
 * （建议制）；章节生成时命中本章计划/大纲要点的条目会作为「技法提示」参考
 * 区段注入（advisory，非创作约束）。空查询=分类目录初始态。
 */
const TechniquePanel: React.FC<TechniquePanelProps> = ({ open, onClose }) => {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<NovelTechnique[]>([])
  const [categories, setCategories] = useState<NovelTechniqueSearchResult['categories']>([])
  const [total, setTotal] = useState(0)
  const [searched, setSearched] = useState(false)
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  const runSearch = async (q: string) => {
    setLoading(true)
    setErr('')
    try {
      const res = await app.NovelTechniqueSearch(q)
      setResults(res.results ?? [])
      setCategories(res.categories ?? [])
      setTotal(res.total ?? 0)
      setSearched(true)
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (open) {
      setQuery('')
      setErr('')
      void runSearch('') // 打开即拉分类目录初始态
    }
  }, [open])

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>技法库（网文写作技法检索）</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{ body: { background: 'transparent' }, header: { background: 'transparent' } }}
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block' }}>
          内置技法账本（{total} 条，九类）：检索零 LLM、只读不落盘；生成章节时命中本章计划/大纲要点的条目会自动作为参考注入。
        </Typography.Text>

        <Space wrap size={8}>
          <Input
            data-testid="technique-query"
            placeholder="输入场景或意图，如：打脸 / 误会 / 伏笔 / 反派"
            style={{ width: 320 }}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onPressEnter={() => void runSearch(query)}
            allowClear
          />
          <Button size="small" type="primary" data-testid="technique-search" loading={loading} onClick={() => void runSearch(query)}>
            检索
          </Button>
        </Space>

        {categories.length > 0 && (
          <Space wrap size={4} data-testid="technique-cats">
            {categories.map((c) => (
              <Tag key={c.category} style={{ cursor: 'pointer' }} onClick={() => { setQuery(c.category); void runSearch(c.category) }}>
                {c.category}·{c.count}
              </Tag>
            ))}
          </Space>
        )}

        {err && <Alert type="warning" showIcon data-testid="technique-error" message={err} />}
        {loading && <Spin size="small" />}

        {searched && !loading && results.length === 0 && !err && (
          <Typography.Text type="secondary" data-testid="technique-empty" style={{ fontSize: 12 }}>
            没有命中的技法——换个词试试（账本按关键词与分类名匹配）
          </Typography.Text>
        )}

        {results.length > 0 && (
          <div data-testid="technique-results" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {results.map((t) => (
              <div key={t.id} style={{ border: `1px solid ${C('color-border')}`, borderRadius: 6, padding: '8px 10px' }}>
                <Space wrap size={4} style={{ marginBottom: 2 }}>
                  <Tag color="blue">{t.category}</Tag>
                  {(t.genres ?? []).map((g) => (
                    <Tag key={g}>{g}</Tag>
                  ))}
                </Space>
                <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block' }}>
                  {t.summary}
                </Typography.Text>
                <Typography.Text style={{ color: C('color-text'), fontSize: 12, display: 'block', marginTop: 2 }}>
                  {t.detail}
                </Typography.Text>
                <Typography.Text type="warning" style={{ fontSize: 12, display: 'block', marginTop: 2 }}>
                  毒点：{t.pitfall}
                </Typography.Text>
              </div>
            ))}
          </div>
        )}
      </Space>
    </Modal>
  )
}

export default TechniquePanel
