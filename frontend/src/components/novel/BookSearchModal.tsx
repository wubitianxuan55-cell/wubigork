// BookSearchModal.tsx — 书架「在线搜书」（书源取书→拆书导入 t2，规格
// docs/gaea-novel-booksource-import-2026-09.md §5）。
// 流程：搜索（书源聚合 + 泛搜索同表）→ 候选表（来源/可导入标注）→ 目录预览
// → 范围选择 → 后台下载（进度条 novel-import-progress:<jobId>）→ 完成回调
// onImported（HomePage 落「正在编辑」+ v4.279 报告文案）。
// 边界（规格 §8）：免规则来源正文不可解析——hasRule=false 只作参考不可导入；
// 泛搜索候选（kind=web）同 host 有启用中的书源规则（hasRule）才是可导入通道。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import {
  Alert, Button, Checkbox, Empty, Input, InputNumber, List, Modal,
  Progress, Space, Spin, Tag, Typography, message,
} from 'antd'
import { ArrowLeftOutlined, CloudDownloadOutlined, SearchOutlined, SettingOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import type {
  NovelBookSourceAppendResult, NovelBookSourceCandidate, NovelBookSourceTocPreview,
} from '../../gaea/lib/bridge/novel'
import { bookImportProgressChannel, subscribe } from '../../events'
import { GENRE_OPTIONS, STYLE_OPTIONS } from './novelOptions'
import type { ImportReportLike } from '../../utils/novelImportReport'
import { clearSearchHistory, loadSearchHistory, pushSearchHistory } from '../../utils/bookSearchHistory'
import BookSearchEnginesModal from './BookSearchEnginesModal'
import { C } from '../../utils/theme'

interface BookSearchModalProps {
  open: boolean
  onClose: () => void
  /** 导入完成（done 事件）：HomePage 打开项目 + 报告直显（与文件导入同款）。 */
  onImported: (res: ImportReportLike) => void
  /** 失败章补下完成（append-done 事件）：HomePage 刷新书架 + 提示。 */
  onAppended: (res: NovelBookSourceAppendResult) => void
}

/** 下载失败章（引擎重试穷尽后如实上报，不占位；对齐 booksource.FailedChapter）。 */
interface FailedChapterLike {
  title: string
  url: string
  error: string
}

/** 进度事件载荷（对齐 novel_booksource_handler.go emit 的三种 type）。 */
type ImportProgressEvent =
  | { type: 'progress'; done: number; total: number }
  | { type: 'done'; result: ImportReportLike; failed?: FailedChapterLike[] }
  | { type: 'error'; error: string; failed?: number }

const secondaryStyle = { color: C('color-text-secondary'), fontSize: 12 } as const

/** 失落兜底窗口：起跑成功后多久没收到任何进度事件就认定回调丢了（v4.421.0 P1）。 */
const PROGRESS_LOST_MS = 3000
/** 兜底态下按书架成书清单对账的轮询间隔。 */
const RECONCILE_INTERVAL_MS = 3000

/** 在线搜书 Modal：自包含 搜索→目录→范围→进度 流程；完成/失败经回调与全局提示上报。 */
const BookSearchModal: React.FC<BookSearchModalProps> = ({ open, onClose, onImported, onAppended }) => {
  // ── 搜索 ──
  const [keyword, setKeyword] = useState('')
  const [searching, setSearching] = useState(false)
  const [candidates, setCandidates] = useState<NovelBookSourceCandidate[]>([])
  const [searchWarnings, setSearchWarnings] = useState<string[]>([])

  // ── 目录预览 + 范围 ──
  const [selected, setSelected] = useState<NovelBookSourceCandidate | null>(null)
  const [tocLoading, setTocLoading] = useState(false)
  // v4.354：选书→取目录的 seq 守卫——网络秒级延迟窗口内换书，慢响应把 A 的目录
  // 落在 B 的界面上（章节范围用 B 的书源 URL+A 的 range 执行导入）
  const pickSeqRef = useRef(0)
  const [toc, setToc] = useState<NovelBookSourceTocPreview | null>(null)
  const [rangeStart, setRangeStart] = useState(1)
  const [rangeEnd, setRangeEnd] = useState(0)

  // ── 落库表单 ──
  const [title, setTitle] = useState('')
  const [genre, setGenre] = useState<string[]>([])
  const [style, setStyle] = useState<string[]>([])

  // ── 导入进度 ──
  const [jobId, setJobId] = useState('')
  const [progress, setProgress] = useState<{ done: number; total: number }>({ done: 0, total: 0 })
  const [importing, setImporting] = useState(false)
  const [startError, setStartError] = useState('')
  const unsubRef = useRef<(() => void) | null>(null)
  const cancelRequestedRef = useRef(false)

  // ── 失落兜底（v4.421.0 P1：进度事件可能在订阅建立前就已发出）──
  // 后端 NovelBookSourceImport 是「先起 goroutine 再返回 jobId」
  // （internal/app/novel_booksource_handler.go:332 起协程 / :359 回执），快速失败或
  // 瞬时完成的终态事件在 subscribe 之前就 emit 完毕 → UI 曾永远停在「后台下载中…」。
  // 前端没有 job 状态绑定，唯一可用于对账的既有绑定是书架成书清单（CoreB.ListProjects）：
  // PROGRESS_LOST_MS 内没有任何进度事件即进入兜底，轮询清单按书名对账；确认入库如实
  // 上报，查不到就如实提示未确认并给刷新按钮——既不假装成功，也不静默卡死。
  const [progressLost, setProgressLost] = useState(false)
  const [lostKind, setLostKind] = useState<'import' | 'retry'>('import')
  /** 清单对账命中时的章数（null=尚未确认入库）。 */
  const [shelfHit, setShelfHit] = useState<number | null>(null)
  const [shelfChecking, setShelfChecking] = useState(false)
  const [importedTitle, setImportedTitle] = useState('')
  const gotEventRef = useRef(false)
  const lostWatchRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const lostPollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // ── 搜索历史（t3 余项）+ 引擎规则编辑器入口 ──
  const [history, setHistory] = useState<string[]>([])
  const [enginesOpen, setEnginesOpen] = useState(false)

  // ── 失败章补下（t3）：done 带失败清单 → 面板内一键重试补下 ──
  const [lastImported, setLastImported] = useState<ImportReportLike | null>(null)
  const [failedList, setFailedList] = useState<FailedChapterLike[]>([])
  const [retrying, setRetrying] = useState(false)
  const [appendMsg, setAppendMsg] = useState('')

  /** 停掉失落兜底的计时器（关闭弹窗、终态事件到达、对账命中都要停）。 */
  const stopLostWatch = useCallback(() => {
    if (lostWatchRef.current !== null) {
      clearTimeout(lostWatchRef.current)
      lostWatchRef.current = null
    }
    if (lostPollRef.current !== null) {
      clearInterval(lostPollRef.current)
      lostPollRef.current = null
    }
  }, [])

  const detachProgress = useCallback(() => {
    unsubRef.current?.()
    unsubRef.current = null
    stopLostWatch()
  }, [stopLostWatch])

  /** 书架成书清单对账（唯一可用于对账的既有绑定）：命中返回该卡章数，否则 null。
   *  查询失败同样返回 null——宁可如实报「未确认」，也不假装成功。 */
  const findOnShelf = useCallback(async (wantTitle: string): Promise<number | null> => {
    try {
      const cards = await app.ListProjects()
      const hit = (cards ?? []).find(c => String(c?.title ?? '') === wantTitle)
      if (!hit) return null
      const n = Number(hit.chapter_count)
      return Number.isFinite(n) ? n : 0
    } catch {
      return null
    }
  }, [])

  /** 对账一次：命中→如实上报并停轮询；未命中且 announce→如实提示「还没入清单」。 */
  const checkShelf = useCallback(async (kind: 'import' | 'retry', wantTitle: string, announce: boolean) => {
    setShelfChecking(true)
    try {
      const chapters = await findOnShelf(wantTitle)
      if (chapters === null) {
        if (announce) {
          message.info(`书架清单里还没有《${wantTitle}》：请稍后刷新，或确认后台是否仍在下载`)
        }
        return null
      }
      stopLostWatch()
      setShelfHit(chapters)
      message.success(kind === 'retry'
        ? `未收到补下进度回调；书架清单显示《${wantTitle}》现有 ${chapters} 章（补下结果以章数为准）`
        : `未收到进度回调，已按书架成书清单确认《${wantTitle}》入库（${chapters} 章）`)
      return chapters
    } finally {
      setShelfChecking(false)
    }
  }, [findOnShelf, stopLostWatch])

  /** 起跑成功后起表：PROGRESS_LOST_MS 内无任何事件即进入兜底（轮询成书清单对账）。 */
  const armLostWatch = useCallback((kind: 'import' | 'retry', wantTitle: string) => {
    stopLostWatch()
    gotEventRef.current = false
    setLostKind(kind)
    lostWatchRef.current = setTimeout(() => {
      lostWatchRef.current = null
      if (gotEventRef.current) return // 订阅有效：事件已到，无需兜底
      setProgressLost(true)
      // 先挂轮询再首查：checkShelf 命中时会 stopLostWatch 把这个 interval 一并清掉
      lostPollRef.current = setInterval(() => { void checkShelf(kind, wantTitle, false) }, RECONCILE_INTERVAL_MS)
      void checkShelf(kind, wantTitle, false)
    }, PROGRESS_LOST_MS)
  }, [checkShelf, stopLostWatch])

  /** 兜底态下的逃生门：如实说明后台不因关窗而停止，并在关闭时清掉在途标记。 */
  const dismissLost = () => {
    setImporting(false)
    setRetrying(false)
    setProgressLost(false)
    setShelfHit(null)
    onClose()
  }

  // 打开时载入搜索历史
  useEffect(() => {
    if (open) setHistory(loadSearchHistory())
  }, [open])

  // 关闭即复位（订阅必退，杜绝悬挂监听）
  useEffect(() => {
    if (open) return
    detachProgress()
    setKeyword(''); setCandidates([]); setSearchWarnings([])
    setSelected(null); setToc(null); setTocLoading(false)
    setTitle(''); setGenre([]); setStyle([])
    setJobId(''); setProgress({ done: 0, total: 0 })
    setImporting(false); setSearching(false); setStartError('')
    setLastImported(null); setFailedList([]); setRetrying(false); setAppendMsg('')
    setEnginesOpen(false)
    cancelRequestedRef.current = false
    // 失落兜底态随弹窗关闭一并清掉（含在途标记，杜绝「重开后仍显示下载中」）
    setProgressLost(false); setShelfHit(null); setShelfChecking(false); setImportedTitle('')
    gotEventRef.current = false
  }, [open, detachProgress])
  useEffect(() => detachProgress, [detachProgress])

  const handleSearch = async (kwArg?: string) => {
    const kw = (kwArg ?? keyword).trim()
    if (!kw || searching) return
    setSearching(true)
    setSearchWarnings([])
    try {
      const res = await app.NovelBookSourceSearch(kw)
      setHistory(pushSearchHistory(kw))
      setCandidates(res.candidates ?? [])
      setSearchWarnings(res.warnings ?? [])
      if ((res.candidates ?? []).length === 0 && (res.warnings ?? []).length === 0) {
        message.info('没有找到候选，换个关键词或先在书源规则目录添加规则')
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '搜索失败')
    } finally {
      setSearching(false)
    }
  }

  const handlePick = async (c: NovelBookSourceCandidate) => {
    const seq = ++pickSeqRef.current
    setSelected(c)
    setToc(null)
    setTitle(c.title || '')
    setGenre([]); setStyle([])
    setTocLoading(true)
    try {
      const res = await app.NovelBookSourceToc(c.source, c.url)
      if (seq !== pickSeqRef.current) return
      setToc(res)
      setRangeStart(1)
      setRangeEnd(res.total)
    } catch (err: unknown) {
      if (seq !== pickSeqRef.current) return
      message.error(err instanceof Error ? err.message : '目录获取失败')
      setSelected(null)
    } finally {
      if (seq === pickSeqRef.current) setTocLoading(false)
    }
  }

  const handleImport = async () => {
    if (!selected || !toc || importing) return
    if (!title.trim()) { message.warning('请填写书名'); return }
    const start = Math.min(Math.max(1, rangeStart), toc.total)
    const end = Math.min(Math.max(start, rangeEnd), toc.total)
    setImporting(true)
    setStartError('')
    setProgress({ done: 0, total: end - start + 1 })
    cancelRequestedRef.current = false
    try {
      const started = await app.NovelBookSourceImport(
        selected.source, selected.url, start, end,
        title.trim(), genre.join('、') || '未分类', style.join('、') || '默认',
      )
      setJobId(started.jobId)
      setImportedTitle(title.trim())
      detachProgress()
      unsubRef.current = subscribe(bookImportProgressChannel(started.jobId), (data) => {
        const ev = data as ImportProgressEvent
        // 任何事件都证明订阅有效：撤销失落兜底（真实事件永远优先于清单对账）
        gotEventRef.current = true
        stopLostWatch()
        setProgressLost(false)
        setShelfHit(null)
        if (ev?.type === 'progress') {
          setProgress({ done: ev.done, total: ev.total })
          return
        }
        detachProgress()
        setImporting(false)
        if (ev?.type === 'done') {
          const failed = ev.failed ?? []
          onImported(ev.result)
          if (failed.length > 0) {
            // t3：失败章留面板可一键重试补下（清单仅存于本次会话，关闭即弃——
            // 不做持久化/断点续传，规格 §6 不做清单）。此时不关 Modal——父层
            // 也不再代关，关闭权在本组件（全部成功自动关 / 面板「完成」手动关）。
            setLastImported(ev.result)
            setFailedList(failed)
          } else {
            onClose()
          }
        } else if (ev?.type === 'error') {
          if (cancelRequestedRef.current) {
            message.info('已取消导入')
          } else {
            const suffix = ev.failed ? `（${ev.failed} 章未取到）` : ''
            setStartError(`${ev.error}${suffix}`)
          }
        }
      })
      // 订阅已建立 → 起「失落兜底」表（订阅之前就 emit 掉的终态事件由清单对账补上）
      armLostWatch('import', title.trim())
    } catch (err: unknown) {
      setImporting(false)
      message.error(err instanceof Error ? err.message : '导入起跑失败')
    }
  }

  const handleCancelImport = async () => {
    if (!jobId) return
    cancelRequestedRef.current = true
    try {
      const ok = await app.NovelBookSourceImportCancel(jobId)
      if (!ok) message.info('任务已结束，无需取消')
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '取消失败')
    }
  }

  // 失败章补下（t3）：done 事件 Failed 原样回传，引擎按清单抓章、项目端续编落库。
  const handleRetry = async () => {
    if (!selected || !lastImported || retrying || failedList.length === 0) return
    setRetrying(true)
    setStartError('')
    setAppendMsg('')
    setProgress({ done: 0, total: failedList.length })
    cancelRequestedRef.current = false
    try {
      const started = await app.NovelBookSourceImportChapters(
        selected.source, lastImported.path,
        JSON.stringify(failedList.map(({ title, url }) => ({ title, url }))),
      )
      setJobId(started.jobId)
      setImportedTitle(lastImported.title)
      detachProgress()
      unsubRef.current = subscribe(bookImportProgressChannel(started.jobId), (data) => {
        const ev = data as {
          type?: string; done?: number; total?: number
          error?: string; failed?: number; result?: NovelBookSourceAppendResult
        }
        // 同导入路径：任何事件都撤销失落兜底
        gotEventRef.current = true
        stopLostWatch()
        setProgressLost(false)
        setShelfHit(null)
        if (ev?.type === 'progress') {
          setProgress({ done: ev.done ?? 0, total: ev.total ?? 0 })
          return
        }
        detachProgress()
        setRetrying(false)
        if (ev?.type === 'append-done' && ev.result) {
          const remain = ev.result.failed ?? []
          setFailedList(remain)
          setAppendMsg(remain.length > 0
            ? `已补下 ${ev.result.appended} 章，仍有 ${remain.length} 章未取到`
            : `已补下 ${ev.result.appended} 章，全部章节已齐`)
          onAppended(ev.result)
        } else if (ev?.type === 'error') {
          if (cancelRequestedRef.current) {
            message.info('已取消补下')
          } else {
            setStartError(`${ev.error}${ev.failed ? `（${ev.failed} 章未取到）` : ''}`)
          }
        }
      })
      // 补下的终态事件（append-done）同款竞态：3s 无事件即按成书清单章数如实对账
      armLostWatch('retry', lastImported.title)
    } catch (err: unknown) {
      setRetrying(false)
      message.error(err instanceof Error ? err.message : '补下起跑失败')
    }
  }

  const backToSearch = () => {
    pickSeqRef.current += 1 // 失效在途目录响应（v4.354）
    setSelected(null); setToc(null); setStartError('')
  }

  // ── 渲染：候选条目 ──
  const renderCandidate = (c: NovelBookSourceCandidate) => (
    <List.Item
      actions={[(
        <Button
          key="pick"
          size="small"
          type={c.hasRule ? 'primary' : 'default'}
          disabled={!c.hasRule}
          onClick={() => void handlePick(c)}
        >
          {c.hasRule ? '选书' : '仅参考'}
        </Button>
      )]}
    >
      <div style={{ minWidth: 0 }}>
        <Space size={6} wrap>
          <Typography.Text strong style={{ color: C('color-text') }}>{c.title}</Typography.Text>
          {c.author ? <Typography.Text style={secondaryStyle}>{c.author}</Typography.Text> : null}
          {c.hasRule
            ? <Tag color="green">可导入</Tag>
            : <Tag>免规则 · 正文不可解析</Tag>}
        </Space>
        <div style={secondaryStyle}>
          来源：{c.source}（{c.kind === 'rule' ? '书源' : '网页搜索'}）
          {c.latestChapter ? ` · 最新：${c.latestChapter}` : ''}
        </div>
      </div>
    </List.Item>
  )

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>在线搜书</span>}
      open={open}
      onCancel={() => {
        // 导入/补下在途默认不许误关；进入失落兜底后必须留逃生门
        //（dismissLost 会清掉在途标记，避免重开后仍显示「后台下载中…」）
        if (progressLost || (!importing && !retrying)) dismissLost()
      }}
      footer={null}
      width={640}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{ body: { background: 'transparent' }, header: { background: 'transparent' } }}
    >
      {selected == null ? (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Input.Search
            placeholder="输入书名或关键字搜书…"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onSearch={() => void handleSearch()}
            enterButton="搜书"
            loading={searching}
            prefix={<SearchOutlined style={{ color: C('color-text-secondary') }} />}
            aria-label="在线搜书关键字"
          />
          {(
            <Space size={4} wrap align="center">
              <Button
                type="text"
                size="small"
                icon={<SettingOutlined aria-hidden />}
                onClick={() => setEnginesOpen(true)}
              >
                搜索引擎规则
              </Button>
              {history.slice(0, 8).map((kw) => (
                <Tag
                  key={kw}
                  style={{ cursor: 'pointer', color: C('color-text-secondary') }}
                  onClick={() => { setKeyword(kw); void handleSearch(kw) }}
                >
                  {kw}
                </Tag>
              ))}
              {history.length > 0 && (
                <Button type="text" size="small" onClick={() => { clearSearchHistory(); setHistory([]) }}>
                  清空历史
                </Button>
              )}
            </Space>
          )}
          {searchWarnings.length > 0 && (
            <Alert
              type="warning"
              showIcon
              message={`部分来源未返回：${searchWarnings.slice(0, 2).join('；')}${searchWarnings.length > 2 ? ` 等 ${searchWarnings.length} 条` : ''}`}
            />
          )}
          {searching ? (
            <div style={{ textAlign: 'center', padding: '24px 0' }}><Spin /></div>
          ) : candidates.length > 0 ? (
            <List
              size="small"
              dataSource={candidates}
              renderItem={renderCandidate}
              pagination={candidates.length > 6 ? { pageSize: 6, size: 'small' } : false}
            />
          ) : (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={keyword.trim()
                ? '没有找到候选（免规则站点只能作参考，正文无法解析）'
                : '搜索书源与网页，命中规则来源的书可整本导入书架'}
            />
          )}
        </Space>
      ) : (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <div>
            <Space size={6} wrap>
              <Typography.Text strong style={{ color: C('color-text') }}>{selected.title}</Typography.Text>
              <Tag color="green">来源：{selected.source}</Tag>
            </Space>
          </div>

          {tocLoading ? (
            <div style={{ textAlign: 'center', padding: '24px 0' }}><Spin /></div>
          ) : toc && (
            <>
              <Typography.Text style={secondaryStyle}>
                共 {toc.total} 章{toc.truncated ? '（目录较长，仅展示开头与结尾样例）' : ''}
              </Typography.Text>
              <List
                size="small"
                bordered
                dataSource={toc.sample ?? []}
                renderItem={(item, idx) => (
                  <List.Item style={{ padding: '4px 12px' }}>
                    <Typography.Text ellipsis style={{ color: C('color-text'), maxWidth: '100%' }}>
                      {toc.truncated && idx === 8 ? `……（共 ${toc.total} 章）` : item.title}
                    </Typography.Text>
                  </List.Item>
                )}
              />

              <Space size={8} wrap>
                <Typography.Text style={secondaryStyle}>下载范围</Typography.Text>
                <InputNumber min={1} max={toc.total} value={rangeStart} onChange={(v) => setRangeStart(Number(v) || 1)} aria-label="起始章" />
                <Typography.Text style={secondaryStyle}>至</Typography.Text>
                <InputNumber min={1} max={toc.total} value={rangeEnd} onChange={(v) => setRangeEnd(Number(v) || 1)} aria-label="结束章" />
                <Typography.Text style={secondaryStyle}>章（共 {Math.max(0, Math.min(toc.total, rangeEnd) - Math.max(1, rangeStart) + 1)} 章）</Typography.Text>
              </Space>

              <Input
                placeholder="书名（必填）"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
              />
              <div>
                <Typography.Text style={{ ...secondaryStyle, marginBottom: 6, display: 'block' }}>题材（可多选）</Typography.Text>
                <Checkbox.Group
                  options={GENRE_OPTIONS}
                  value={genre}
                  onChange={(v) => setGenre(v as string[])}
                  style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
                />
              </div>
              <div>
                <Typography.Text style={{ ...secondaryStyle, marginBottom: 6, display: 'block' }}>文风（可多选）</Typography.Text>
                <Checkbox.Group
                  options={STYLE_OPTIONS}
                  value={style}
                  onChange={(v) => setStyle(v as string[])}
                  style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
                />
              </div>

              {importing ? (
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  <Progress
                    percent={progress.total > 0 ? Math.round((progress.done / progress.total) * 100) : 0}
                    status="active"
                    format={() => `${progress.done}/${progress.total} 章`}
                  />
                  <Space>
                    <Button icon={<CloudDownloadOutlined aria-hidden />} disabled>后台下载中…</Button>
                    <Button onClick={() => void handleCancelImport()}>取消</Button>
                  </Space>
                  <Typography.Text style={secondaryStyle}>下载完成后自动入书架并打开</Typography.Text>
                </Space>
              ) : (
                <Space>
                  <Button type="primary" icon={<CloudDownloadOutlined aria-hidden />} onClick={() => void handleImport()}>
                    开始导入
                  </Button>
                  <Button icon={<ArrowLeftOutlined aria-hidden />} onClick={backToSearch}>返回搜索</Button>
                </Space>
              )}
              {startError && <Alert type="error" showIcon message={startError} />}

              {/* 失落兜底面板（v4.421.0）：进度事件可能在订阅建立前就已发出。
                  只按书架成书清单如实对账——命中报章数，查不到就报未确认。 */}
              {progressLost && (
                <Alert
                  type={shelfHit === null ? 'warning' : 'success'}
                  showIcon
                  data-testid="import-progress-lost"
                  message={shelfHit === null
                    ? `未收到${lostKind === 'retry' ? '补下' : '导入'}进度回调，无法确认后台状态`
                    : lostKind === 'retry'
                      ? `未收到补下进度回调；书架清单显示《${importedTitle}》现有 ${shelfHit} 章`
                      : `已按书架成书清单确认《${importedTitle}》入库（${shelfHit} 章）`}
                  description={shelfHit === null
                    ? '快速完成或快速失败的终态事件可能在进度订阅建立前就已发出。可刷新成书清单核对结果——本窗口不会假装成功，也不会静默卡住。'
                    : '结果以书架成书清单为准，可以关闭本窗口。'}
                  action={(
                    <Space size={6}>
                      <Button size="small" loading={shelfChecking} data-testid="import-shelf-refresh"
                        onClick={() => void checkShelf(lostKind, importedTitle, true)}>
                        刷新清单
                      </Button>
                      {shelfHit === null
                        ? <Button size="small" data-testid="import-shelf-close" onClick={dismissLost}>关闭（后台继续）</Button>
                        : <Button size="small" type="primary" data-testid="import-shelf-done" onClick={dismissLost}>完成</Button>}
                    </Space>
                  )}
                />
              )}

              {failedList.length > 0 && !importing && (
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  <Alert
                    type="warning"
                    showIcon
                    message={`${failedList.length} 章下载失败，未入库：${failedList.slice(0, 3).map((f) => f.title).join('、')}${failedList.length > 3 ? ' 等' : ''}`}
                  />
                  {retrying ? (
                    <>
                      <Progress
                        percent={progress.total > 0 ? Math.round((progress.done / progress.total) * 100) : 0}
                        status="active"
                        format={() => `${progress.done}/${progress.total} 章`}
                      />
                      <Space>
                        <Button disabled>补下中…</Button>
                        <Button onClick={() => void handleCancelImport()}>取消</Button>
                      </Space>
                    </>
                  ) : (
                    <Button type="primary" icon={<CloudDownloadOutlined aria-hidden />} onClick={() => void handleRetry()}>
                      重试补下 {failedList.length} 章
                    </Button>
                  )}
                </Space>
              )}
              {(appendMsg || (failedList.length === 0 && lastImported && !importing)) && (
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  {appendMsg && <Typography.Text style={secondaryStyle}>{appendMsg}</Typography.Text>}
                  <Button onClick={onClose}>完成</Button>
                </Space>
              )}
            </>
          )}
        </Space>
      )}

      <BookSearchEnginesModal open={enginesOpen} onClose={() => setEnginesOpen(false)} />
    </Modal>
  )
}

export default BookSearchModal
