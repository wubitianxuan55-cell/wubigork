// sin/SinBookSourcePanel.tsx — 右栏「书源」页签（sin 书源线 t3）。
//
// 搜书 → 候选表（HasRule=可下载；免规则候选「仅参考」不可选，与后端 fail-closed
// 边界同口径的诚实 UI）→ 目录预览（总数+首末样例）→ 范围下载（进度/取消）→
// 成书清单（删除）。下载中关面板不打断（后台继续，完成后清单可见）；
// 事件 `sin-booksource:<jobId>`（progress/done/error），终态后必退订。
// 浏览器 dev mock 下为诚实空态（mock/sin.ts）。

import { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Input, InputNumber, Popconfirm, Progress, message } from 'antd'
import { DeleteOutlined, ExportOutlined, SearchOutlined, SendOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../../events'
import {
  beginBookDownload, bookDownloadOutcome, bookDownloadSnapshot, cancelBookDownloadTracking,
  subscribeBookDownload, __resetBookDownloadForTest,
} from './sinBookDownload'
import type { NovelBookSourceCandidate } from '../../gaea/lib/bridge/novel'
import type { SinBookSourceBook, SinBookSourceDownloadStart } from '../../gaea/lib/bridge/sin'
import type { NovelBookSourceTocPreview } from '../../gaea/lib/bridge/novel'
import V3Empty from '../../components/V3Empty'

function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

function fmtSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  if (bytes >= 1024) return `${Math.round(bytes / 1024)} KB`
  return `${bytes} B`
}

function fmtTime(rfc3339: string): string {
  const t = Date.parse(rfc3339)
  if (!Number.isFinite(t)) return ''
  const d = new Date(t)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

export function SinBookSourcePanel() {
  const [keyword, setKeyword] = useState('')
  const [searching, setSearching] = useState(false)
  // 下载会话：模块单例快照（卡切换重挂载即恢复进度显示）
  const [dl, setDl] = useState(bookDownloadSnapshot().state)
  const [candidates, setCandidates] = useState<NovelBookSourceCandidate[]>([])
  const [warnings, setWarnings] = useState<string[]>([])
  const [selected, setSelected] = useState<NovelBookSourceCandidate | null>(null)
  const [toc, setToc] = useState<NovelBookSourceTocPreview | null>(null)
  const [tocLoading, setTocLoading] = useState(false)
  const [rangeStart, setRangeStart] = useState<number | null>(null)
  const [rangeEnd, setRangeEnd] = useState<number | null>(null)
  const [downloadError, setDownloadError] = useState('')
  const [books, setBooks] = useState<SinBookSourceBook[]>([])
  const [booksLoading, setBooksLoading] = useState(false)
  // 起跑闸：SinBookSourceDownload 往返期间按钮仍可点——历史实现双击发两个 job
  const startingRef = useRef(false)

  // 订阅模块会话（快照同步；终态 tick 变化驱动清单刷新）
  const [tick, setTick] = useState(bookDownloadSnapshot().tick)
  useEffect(() => {
    const sync = () => {
      const snap = bookDownloadSnapshot()
      setDl({ ...snap.state })
      setTick(snap.tick)
    }
    sync()
    return subscribeBookDownload(sync)
  }, [])
  // 终态去向回填组件态：done 清选中（回到可再搜状态，与历史行为一致）；
  // error 回填行内原因（重试入口就在旁边）
  const prevTickRef = useRef(bookDownloadSnapshot().tick)
  useEffect(() => {
    if (tick === prevTickRef.current) return
    prevTickRef.current = tick
    const oc = bookDownloadOutcome()
    if (oc.outcome === 'done') {
      setSelected(null)
      setToc(null)
    } else if (oc.outcome === 'error') {
      setDownloadError(oc.error || '下载失败')
    }
  }, [tick])

  const refreshBooks = useCallback(() => {
    setBooksLoading(true)
    app.SinBookSourceBooksList()
      .then((list) => setBooks(list ?? []))
      .catch(() => setBooks([])) // 清单读失败不阻断面板（下载/搜索不受影响）
      .finally(() => setBooksLoading(false))
  }, [])

  useEffect(() => { refreshBooks() }, [refreshBooks])
  // 下载终态（done/error/取消）后刷新成书清单
  useEffect(() => { if (tick > 0) refreshBooks() }, [tick, refreshBooks])

  const handleSearch = useCallback(() => {
    const kw = keyword.trim()
    if (!kw || searching) return
    setSearching(true)
    setSelected(null)
    setToc(null)
    app.SinBookSourceSearch(kw)
      .then((res) => {
        setCandidates(res.candidates ?? [])
        setWarnings(res.warnings ?? [])
      })
      .catch((err: unknown) => {
        setCandidates([])
        setWarnings([errText(err, '搜索失败')])
      })
      .finally(() => setSearching(false))
  }, [keyword, searching])

  const selectCandidate = useCallback((c: NovelBookSourceCandidate) => {
    if (!c.hasRule) return
    setSelected(c)
    setToc(null)
    setTocLoading(true)
    setRangeStart(null)
    setRangeEnd(null)
    app.SinBookSourceToc(c.source, c.url)
      .then((pv) => {
        setToc(pv)
        setRangeStart(1)
        setRangeEnd(pv.total > 0 ? pv.total : null)
      })
      .catch((err: unknown) => {
        setSelected(null)
        message.warning(errText(err, '目录预览失败'))
      })
      .finally(() => setTocLoading(false))
  }, [])

  const startDownload = useCallback(() => {
    if (!selected || !toc || startingRef.current || dl.kind === 'downloading') return
    const total = toc.total
    const start = Math.min(Math.max(rangeStart ?? 1, 1), total)
    const end = Math.min(Math.max(rangeEnd ?? total, 1), total)
    if (start > end) {
      message.warning('起始章不能大于结束章')
      return
    }
    setDownloadError('')
    startingRef.current = true
    app.SinBookSourceDownload(selected.source, selected.url, start, end, selected.title)
      .then((res: SinBookSourceDownloadStart) => {
        startingRef.current = false
        beginBookDownload(res.jobId, Math.max(end - start + 1, 0))
      })
      .catch((err: unknown) => {
        startingRef.current = false
        setDownloadError(errText(err, '下载起跑失败'))
      })
  }, [selected, toc, dl.kind, rangeStart, rangeEnd])

  const cancelDownload = useCallback(() => {
    if (dl.kind !== 'downloading') return
    const jobId = dl.jobId
    app.SinBookSourceDownloadCancel(jobId)
      .then((ok) => {
        if (!ok) message.info('没有在途下载')
        refreshBooks()
      })
      .catch(() => refreshBooks())
    // 本地立即收尾（取消语义：后端中止，成书不落盘）；清单按取消落定后刷新
    cancelBookDownloadTracking()
  }, [dl, refreshBooks])

  const removeBook = useCallback((book: SinBookSourceBook) => {
    app.SinBookSourceBookDelete(book.path)
      .then(() => {
        message.success(`已删除「${book.title}」`)
        refreshBooks()
      })
      .catch((err: unknown) => message.error(errText(err, '删除失败')))
  }, [refreshBooks])

  // ── 成书两个后续动作（t5）：送小说书架（复用既有导入链，零新后端耦合）、导出 EPUB ──
  const [sendingBook, setSendingBook] = useState<string | null>(null)
  const [exportingKey, setExportingKey] = useState<string | null>(null)

  const sendToNovel = useCallback((book: SinBookSourceBook) => {
    if (sendingBook) return
    setSendingBook(book.path)
    app.ImportNovelBookEx(book.path, book.title, '未分类', '默认', 'full', 0)
      .then((res) => {
        message.success(`已导入书架「${res.title}」：${res.chapter_count} 章`)
        // 跨板块跳转走 NAVIGATE 白名单事件（MainLayout 校验空间可达性）
        emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: 'novel' })
      })
      .catch((err: unknown) => message.error(errText(err, '导入失败')))
      .finally(() => setSendingBook(null))
  }, [sendingBook])

  const exportEpub = useCallback((book: SinBookSourceBook) => {
    if (exportingKey) return
    setExportingKey(book.path)
    app.SinBookSourceBookExportEpub(book.path)
      .then(() => message.success('已导出 EPUB（与 TXT 同目录）'))
      .catch((err: unknown) => message.error(errText(err, '导出失败')))
      .finally(() => setExportingKey(null))
  }, [exportingKey])

  const downloading = dl.kind === 'downloading'
  const sample = toc?.sample ?? []
  const headCount = toc?.truncated ? Math.max(sample.length - 4, 0) : sample.length

  return (
    <section className="sin-card sin-bs" aria-label="书源取书">
      <div className="sin-bs-search">
        <Input
          size="small"
          allowClear
          placeholder="书名 / 作者关键字…"
          value={keyword}
          disabled={downloading}
          onChange={(e) => setKeyword(e.target.value)}
          onPressEnter={handleSearch}
          aria-label="书源搜索关键字"
        />
        <Button
          size="small"
          type="primary"
          icon={<SearchOutlined aria-hidden />}
          loading={searching}
          disabled={downloading || !keyword.trim()}
          onClick={handleSearch}
        >
          搜书
        </Button>
      </div>

      {warnings.length > 0 && (
        <div className="sin-bs-warnings">
          {warnings.slice(0, 3).map((w, i) => (
            <p className="sin-bs-warning" key={i}>{w}</p>
          ))}
        </div>
      )}

      {candidates.length > 0 && (
        <div className="sin-bs-cands" role="list" aria-label="搜书候选">
          {candidates.map((c, i) => (
            <button
              type="button"
              role="listitem"
              key={`${c.url}-${i}`}
              className={`sin-bs-cand${selected?.url === c.url ? ' is-active' : ''}${c.hasRule ? '' : ' is-ref'}`}
              disabled={downloading || !c.hasRule}
              title={c.hasRule ? `来源：${c.source}` : '免规则来源正文不可解析，仅参考'}
              onClick={() => selectCandidate(c)}
            >
              <span className="sin-bs-cand-title">{c.title}</span>
              <span className="sin-bs-cand-meta">
                {c.author ? `${c.author} · ` : ''}{c.source}
                <em className={`sin-bs-badge${c.hasRule ? ' is-ok' : ''}`}>{c.hasRule ? '可下载' : '仅参考'}</em>
              </span>
            </button>
          ))}
        </div>
      )}

      {selected && (
        <div className="sin-bs-toc" aria-label="目录预览与下载">
          <div className="sin-bs-toc-head">
            <span className="sin-bs-toc-title">{selected.title}</span>
            {tocLoading ? (
              <span className="sin-bs-toc-meta">目录读取中…</span>
            ) : toc ? (
              <span className="sin-bs-toc-meta">共 {toc.total} 章{toc.truncated ? '（样例为首末章）' : ''}</span>
            ) : null}
          </div>
          {toc && toc.total > 0 && (
            <>
              <div className="sin-bs-sample">
                {sample.slice(0, headCount).map((en) => (
                  <div className="sin-bs-sample-row" key={`h-${en.order}`}>{en.title}</div>
                ))}
                {toc.truncated && <div className="sin-bs-sample-omit">…… 省略 {toc.total - headCount - 4} 章 ……</div>}
                {toc.truncated && sample.slice(headCount).map((en) => (
                  <div className="sin-bs-sample-row" key={`t-${en.order}`}>{en.title}</div>
                ))}
              </div>
              <div className="sin-bs-range">
                <span>第</span>
                <InputNumber size="small" min={1} max={toc.total} value={rangeStart}
                  onChange={(v) => setRangeStart(v)} aria-label="起始章" />
                <span>—</span>
                <InputNumber size="small" min={1} max={toc.total} value={rangeEnd}
                  onChange={(v) => setRangeEnd(v)} aria-label="结束章" />
                <span>章</span>
                <Button
                  size="small" type="primary" disabled={downloading}
                  onClick={startDownload}
                >
                  {downloading ? '下载中…' : '下载成书'}
                </Button>
              </div>
            </>
          )}
          {toc && toc.total === 0 && !tocLoading && (
            <p className="sin-bs-toc-meta">目录为空（源站可能反爬或规则失配）</p>
          )}
        </div>
      )}

      {downloading && (
        <div className="sin-bs-progress" aria-label="下载进度">
          <Progress
            percent={dl.total > 0 ? Math.round((dl.done / dl.total) * 100) : 0}
            size="small"
          />
          <div className="sin-bs-progress-meta">
            <span>{dl.done}/{dl.total || '?'} 章（进度按每 20 章上报）</span>
            <Button size="small" onClick={cancelDownload}>取消</Button>
          </div>
        </div>
      )}

      {downloadError && <p className="sin-side-error">{downloadError}（可调整范围后重试）</p>}

      <div className="sin-bs-books-head">成书清单</div>
      {booksLoading ? null : books.length > 0 ? (
        <div className="sin-bs-books" role="list" aria-label="成书清单">
          {books.map((b) => (
            <div className="sin-bs-book" role="listitem" key={b.path}>
              <span className="sin-bs-book-title" title={b.path}>{b.title}</span>
              <span className="sin-bs-book-meta">{fmtSize(b.sizeBytes)} · {fmtTime(b.modifiedAt)}</span>
              <button
                type="button" className="sin-bs-book-act" title="导出 EPUB（与 TXT 同目录）"
                aria-label={`导出 ${b.title} 的 EPUB`}
                disabled={exportingKey !== null || sendingBook !== null}
                onClick={() => exportEpub(b)}
              >
                <ExportOutlined />
              </button>
              <Popconfirm
                title="导入到小说书架？"
                description="按 TXT 整本导入（书架新建项目），完成后跳转书架。"
                okText="导入"
                cancelText="取消"
                onConfirm={() => sendToNovel(b)}
              >
                <button
                  type="button" className="sin-bs-book-act" title="送入小说板块书架"
                  aria-label={`送 ${b.title} 入小说书架`}
                  disabled={exportingKey !== null || sendingBook !== null}
                >
                  {sendingBook === b.path ? '…' : <SendOutlined />}
                </button>
              </Popconfirm>
              <Popconfirm
                title="删除这本成书？"
                description="删除 TXT 与已导出的 EPUB，不影响任何项目。"
                okText="删除"
                cancelText="取消"
                onConfirm={() => removeBook(b)}
              >
                <button type="button" className="sin-bs-book-del" title="删除成书" aria-label={`删除 ${b.title}`}>
                  <DeleteOutlined />
                </button>
              </Popconfirm>
            </div>
          ))}
        </div>
      ) : (
        <V3Empty
          compact
          description="还没有成书"
          style={{ padding: '18px 0' }}
        >
          <span className="sin-side-empty-hint">搜到书后点「下载成书」，TXT 会收在这里。</span>
        </V3Empty>
      )}
    </section>
  )
}
