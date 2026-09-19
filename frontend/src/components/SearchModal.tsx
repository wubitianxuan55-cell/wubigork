import { wailsApp } from '../lib/wailsApp';
import React, { useRef, useState } from 'react'
import { Typography, Space, Tag, Modal, Input, Spin, Button } from 'antd'
import { SearchOutlined, FileTextOutlined, UserOutlined, ThunderboltOutlined } from '@ant-design/icons'

import { C } from '../utils/theme'
import V3Empty from './V3Empty'
import { app } from '../gaea/lib/bridge'
import { useT } from '../gaea/lib/i18n'
import type { SearchScope, UnifiedSearchView, IntentResultView } from '../gaea/lib/types'
import type { ShellSpace } from '../boards/space'
import type { DictKey } from '../gaea/locales/en'

interface SearchResult {
  file: string
  context: string
}

interface SearchModalProps {
  open: boolean
  onClose: () => void
  /** S2.1：当前壳层空间——搜索 scope 默认跟随（docs/gaea-space-shell-design.md §4.8） */
  space: ShellSpace
}

const categoryIcons: Record<string, React.ReactNode> = {
  chapters: <FileTextOutlined style={{ color: 'var(--color-success)' }} />,
  characters: <UserOutlined style={{ color: '#60a5fa' /* hex-exempt 分类识别色 */ }} />,
}
const categoryLabels: Record<string, DictKey> = {
  chapters: 'shell.search.catChapters', characters: 'shell.search.catCharacters',
}

/** S4.6 指令动作 → 标签 i18n key（指令预览卡用；与后端 intent.Action 对齐） */
const intentActionLabels: Record<string, DictKey> = {
  navigate: 'shell.search.intentNavigate',
  generate_image: 'shell.search.intentGenerateImage',
  status: 'shell.search.intentStatus',
  reminder: 'shell.search.intentReminder',
  read_screen: 'shell.search.intentReadScreen',
}

/** S2.1 scope 三档（工位/乐园/全部；默认=当前空间，「全部」仅显式选择，红线不默认跨空间） */
const SCOPE_OPTIONS: { value: SearchScope; labelKey: DictKey; titleKey: DictKey }[] = [
  { value: 'work', labelKey: 'shell.search.scope.work', titleKey: 'shell.search.scope.workTitle' },
  { value: 'play', labelKey: 'shell.search.scope.play', titleKey: 'shell.search.scope.playTitle' },
  { value: '', labelKey: 'shell.search.scope.all', titleKey: 'shell.search.scope.allTitle' },
]

/** 在文本中用 <mark> 高亮关键词 */
function highlightText(text: string, query: string): React.ReactNode {
  if (!query) return text
  const idx = text.toLowerCase().indexOf(query.toLowerCase())
  if (idx < 0) return text
  return (
    <>
      {text.slice(0, idx)}
      <mark style={{ background: 'color-mix(in srgb, var(--color-warning) 20%, transparent)', color: 'var(--color-warning)', borderRadius: 2, padding: '0 2px' }}>
        {text.slice(idx, idx + query.length)}
      </mark>
      {text.slice(idx + query.length)}
    </>
  )
}

interface SearchSection {
  key: string
  label: string
  icon?: React.ReactNode
  rows: SearchResult[]
}

/** 小说项目搜索 → 章节/角色分节 */
function sectionsFromNovel(
  results: Record<string, SearchResult[]>,
  t: (k: DictKey) => string,
): SearchSection[] {
  return Object.keys(results)
    .filter((k) => results[k].length > 0)
    .map((cat) => ({
      key: cat,
      label: categoryLabels[cat] ? t(categoryLabels[cat]) : cat,
      icon: categoryIcons[cat],
      rows: results[cat],
    }))
}

/** 统一检索（工位记忆/文件）→ 工作区文件 + 记忆语义分节 */
function sectionsFromUnified(
  v: UnifiedSearchView,
  t: (k: DictKey) => string,
): SearchSection[] {
  const out: SearchSection[] = []
  if (v.keyword?.length) {
    out.push({
      key: 'files',
      label: t('shell.search.files'),
      icon: <FileTextOutlined style={{ color: 'var(--color-success)' }} />,
      rows: v.keyword.map((h) => ({ file: h.path, context: h.snippet })),
    })
  }
  if (v.semantic?.length) {
    out.push({
      key: 'memory',
      label: t('shell.search.memory'),
      icon: <FileTextOutlined style={{ color: '#a78bfa' /* hex-exempt 分类识别色 */ }} />,
      rows: v.semantic.map((h) => ({ file: h.kind, context: h.text })),
    })
  }
  if (v.brain?.length) {
    out.push({
      key: 'brain',
      label: t('shell.search.brain'),
      icon: <FileTextOutlined style={{ color: '#f472b6' /* hex-exempt 分类识别色 */ }} />,
      rows: v.brain.map((h) => ({ file: `${h.brain} · ${h.entity}`, context: h.text })),
    })
  }
  return out
}

const SearchModal: React.FC<SearchModalProps> = ({ open, onClose, space }) => {
  const t = useT()
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  // v4.354：检索序号——慢语义检索旧响应覆盖新结果、旧意图卡可被执行
  const searchSeqRef = useRef(0)
  const [sections, setSections] = useState<SearchSection[]>([])
  const [searched, setSearched] = useState(false)
  const [scope, setScope] = useState<SearchScope>(space)
  const [filterCategory, setFilterCategory] = useState<string | null>(null)
  // S4.6 命令面板接统一意图路由：intent=dry-run 预览（零副作用）；intentReply=
  // 真执行回执；intentBusy=执行中态。预览-确认制：命中只出卡，点「执行」才真跑。
  const [intent, setIntent] = useState<IntentResultView | null>(null)
  const [intentReply, setIntentReply] = useState<string | null>(null)
  const [intentBusy, setIntentBusy] = useState(false)

  // 每次打开：scope 默认跟随当前壳层空间（不持久化——默认不跨空间红线）
  React.useEffect(() => {
    if (open) setScope(space)
  }, [open, space])

  const resetQuery = () => {
    setQuery(''); setSections([]); setSearched(false)
    setIntent(null); setIntentReply(null)
  }

  const handleSearch = async (value: string) => {
    const q = value.trim()
    if (!q) { resetQuery(); return }
    const seq = ++searchSeqRef.current
    setLoading(true)
    setSearched(true)
    setFilterCategory(null)
    setIntentReply(null)
    try {
      // S4.6：dry-run 意图预览（RouteIntent(q,true) 零副作用）与检索并行——
      // 命中才出指令预览卡；未命中/调用失败都按无指令处理，检索行为零变化。
      const [intentHit, novel, unified] = await Promise.all([
        app.RouteIntent(q, true).catch(() => null),
        scope === 'work' ? Promise.resolve(null) : wailsApp().Search(q).catch(() => null),
        scope === 'play' ? Promise.resolve(null) : app.UnifiedSearch(q, scope === 'work' ? 'work' : '', 8).catch(() => null),
      ])
      if (seq !== searchSeqRef.current) return
      setIntent(intentHit?.handled ? intentHit : null)
      const all: SearchSection[] = []
      if (novel) all.push(...sectionsFromNovel(novel, t))
      if (unified) all.push(...sectionsFromUnified(unified, t))
      setSections(all)
    } catch (_) {
      if (seq !== searchSeqRef.current) return
      setSections([])
      setIntent(null)
    } finally {
      if (seq === searchSeqRef.current) setLoading(false)
    }
  }

  // S4.6：执行指令（用户显式点击「执行」= 确认）——dryRun=false 真跑能力，
  // 回执内联展示；导航类由后端 emit gaea-intent-navigate → MainLayout
  // navigateBoard 自动切板块，稍候收面板。
  // 7.3-1：save_task 不走 RouteIntent 真执行——前端入口带 shell 空间更准
  // （后端执行层无空间上下文只能取 gaeaEffectiveSpace），直接调同一 Save 内核
  // 落库（source='ctrlk'，两路落同一状态文件，来源可区分）。
  const saveIntentTask = async () => {
    const q = query.trim()
    if (!q || intentBusy || !intent) return
    setIntentBusy(true)
    try {
      await app.GaeaTaskInboxSave(JSON.stringify({
        title: q,
        space,
        source: 'ctrlk',
        action: intent.action,
        target: intent.target,
      }))
      setIntentReply(t('tasks.inbox.saved'))
    } catch (_) {
      /* 落库失败静默：按钮可再点（宁漏勿误，不打断搜索流） */
    } finally {
      setIntentBusy(false)
    }
  }

  const executeIntent = async () => {
    const q = query.trim()
    if (!q || intentBusy) return
    if (intent?.action === 'save_task') {
      await saveIntentTask()
      return
    }
    setIntentBusy(true)
    try {
      const res = await app.RouteIntent(q, false)
      if (res.handled) {
        setIntentReply(res.reply)
        if (res.action === 'navigate') {
          window.setTimeout(() => { onClose(); resetQuery() }, 600)
        }
      } else {
        setIntentReply(null)
        setIntent(null)
      }
    } finally { setIntentBusy(false) }
  }

  const totalResults = sections.reduce((n, s) => n + s.rows.length, 0)
  const filteredSections = filterCategory
    ? sections.filter((s) => s.key === filterCategory)
    : sections

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><SearchOutlined style={{ color: C('color-primary'), marginRight: 8 }} />{t('shell.search.title')}</span>}
      open={open}
      onCancel={() => { onClose(); resetQuery() }}
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
          placeholder={t('shell.search.placeholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          // v4.354：onPressEnter 已移除——antd Input.Search 回车本就触发 onSearch，
          // 双挂=一次回车两遍检索（旧意图卡残留可被执行）
          onSearch={handleSearch}
          size="large"
          allowClear
          style={{ background: C('color-bg-layout') }}
        />

        {/* S2.1 scope 切换：默认=当前空间；「全部」=scope '' 仅显式选择 */}
        <div role="radiogroup" aria-label={t('shell.search.scopeAria')} style={{ display: 'flex', gap: 8 }}>
          {SCOPE_OPTIONS.map((o) => (
            <button
              key={o.value}
              type="button"
              role="radio"
              aria-checked={scope === o.value}
              title={t(o.titleKey)}
              onClick={() => setScope(o.value)}
              style={{
                border: `1px solid ${scope === o.value ? C('color-primary') : C('color-border')}`,
                color: scope === o.value ? C('color-primary') : C('color-text-secondary'),
                background: 'transparent', borderRadius: 999, padding: '2px 12px',
                fontSize: 12, cursor: 'pointer',
              }}
            >
              {t(o.labelKey)}
            </button>
          ))}
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : searched ? (
          <>
            {/* S4.6 指令预览卡：dry-run 命中才显示；「执行」= 用户显式确认（宁漏勿误：
                搜索框不是整句指令入口，绝不因输入自动触发能力） */}
            {intent && (
              <div
                data-testid="intent-card"
                style={{
                  border: '1px solid ' + C('color-primary'), borderRadius: 8, padding: '10px 12px',
                  background: 'color-mix(in srgb, ' + C('color-primary') + ' 8%, transparent)',
                }}
              >
                <Space direction="vertical" size={6} style={{ width: '100%' }}>
                  <Space size={8} style={{ width: '100%', justifyContent: 'space-between' }}>
                    <Space size={8}>
                      <ThunderboltOutlined style={{ color: C('color-primary') }} />
                      <Tag color="blue" style={{ fontSize: 11, marginRight: 0 }}>{t('shell.search.intentTag')}</Tag>
                      {intent.action && intentActionLabels[intent.action] && (
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>
                          {t(intentActionLabels[intent.action])}
                        </Typography.Text>
                      )}
                    </Space>
                    {/* 7.3-1：〔执行〕旁次按钮〔存为任务〕——命中指令才出现（搜索词≠
                        指令时宁漏勿误，手动新建走收件箱面板）；执行按钮保持在首位
                        （既有测试锚定卡内第一个 button）。 */}
                    <Space size={8}>
                      <Button size="small" type="primary" loading={intentBusy} onClick={executeIntent}>
                        {t('shell.search.intentExec')}
                      </Button>
                      <Button size="small" data-testid="intent-save-task" disabled={intentBusy} onClick={saveIntentTask}>
                        {t('shell.search.intentSaveTask')}
                      </Button>
                    </Space>
                  </Space>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
                    {intent.reply}
                  </Typography.Text>
                  {intentReply && (
                    <Typography.Text style={{ color: C('color-success'), fontSize: 12 }}>
                      → {intentReply}
                    </Typography.Text>
                  )}
                </Space>
              </div>
            )}
            {totalResults === 0 ? (
              !intent && (
                <V3Empty description={t('shell.search.noResults', { q: query })} />
              )
            ) : (
              <>
                {/* 分类标签过滤 */}
                <Space size={4} wrap>
                <Tag
                  color={filterCategory === null ? 'blue' : 'default'}
                  style={{ cursor: 'pointer', fontSize: 11 }}
                  onClick={() => setFilterCategory(null)}
                >
                  {t('shell.search.allTag')} ({totalResults})
                </Tag>
                {sections.map((s) => (
                  <Tag
                    key={s.key}
                    color={filterCategory === s.key ? 'blue' : 'default'}
                    style={{ cursor: 'pointer', fontSize: 11 }}
                    onClick={() => setFilterCategory(s.key === filterCategory ? null : s.key)}
                  >
                    {s.icon} {s.label} ({s.rows.length})
                  </Tag>
                ))}
              </Space>

              {/* 结果列表 */}
              {filteredSections.map((s) => (
                <div key={s.key}>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 12, display: 'block', marginBottom: 6 }}>
                    {s.icon} {s.label} · {t('shell.search.count', { count: s.rows.length })}
                  </Typography.Text>
                  <Space direction="vertical" size={6} style={{ width: '100%' }}>
                    {s.rows.map((r, i) => (
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
              )}
            </>
          ) : (
          <div style={{ textAlign: 'center', padding: 20, color: C('color-text-secondary'), fontSize: 12 }}>
            {t('shell.search.empty')}
          </div>
        )}
      </Space>
    </Modal>
  )
}

export default SearchModal
