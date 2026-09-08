/**
 * ReadingChrome — 阅读模式顶部 chrome（书签 / 划线想法 / 全文搜索 / 排版 / 章节导航），
 * 自 ChapterPage 原样搬移：纯受控展示组件，全部状态与回调经 props 传入（与
 * ReadingPrefsPanel 同款拆法）；classNames/data-testid 与拆分前逐字节一致。
 */
import React from 'react'
import { Button, Input, Popover, Tooltip } from 'antd'
import {
  LeftOutlined, RightOutlined, PushpinOutlined, PushpinFilled,
  PlayCircleOutlined, PauseCircleOutlined, CloseOutlined, HighlightOutlined,
  CommentOutlined, SearchOutlined, FontSizeOutlined, EditOutlined,
  ReadOutlined, LoadingOutlined,
} from '@ant-design/icons'
import ReadingPrefsPanel from './ReadingPrefsPanel'
import { splitSnippet, type NovelSearchHitData, type SearchSummary } from './novelSearchUtils'
import { ANNOTATION_COLORS, type ReadingAnnotation } from '../../utils/readingAnnotations'
import type { ReadingBookmark } from '../../utils/readingBookmarks'
import type { ReadingSettings } from '../../utils/readingSettings'

export interface ReadingChromeProps {
  title: string
  totalWords: number
  saved: boolean
  atFirst: boolean
  atLast: boolean
  onPrev: () => void
  onNext: () => void
  bookmarks: ReadingBookmark[]
  bookmarkOpen: boolean
  onBookmarkOpenChange: (open: boolean) => void
  onToggleBookmark: () => void
  onJumpBookmark: (b: ReadingBookmark) => void
  onRemoveBookmark: (b: ReadingBookmark) => void
  autoScrolling: boolean
  onToggleAutoScroll: () => void
  anns: ReadingAnnotation[]
  onJumpAnn: (a: ReadingAnnotation) => void
  onDeleteAnn: (id: string) => void
  searchOpen: boolean
  onSearchOpenChange: (open: boolean) => void
  searchQuery: string
  onSearchQueryChange: (v: string) => void
  onSearchSubmit: () => void
  searchSummary: SearchSummary
  searchHits: NovelSearchHitData[]
  searchLoading: boolean
  searchError: string | null
  onOpenSearchHit: (h: NovelSearchHitData) => void
  onAddSearchHitHighlight: (h: NovelSearchHitData) => void
  prefs: ReadingSettings
  onPrefsChange: (p: Partial<ReadingSettings>) => void
  onEdit: () => void
}

const ReadingChrome: React.FC<ReadingChromeProps> = ({
  title,
  totalWords,
  saved,
  atFirst,
  atLast,
  onPrev,
  onNext,
  bookmarks,
  bookmarkOpen,
  onBookmarkOpenChange,
  onToggleBookmark,
  onJumpBookmark,
  onRemoveBookmark,
  autoScrolling,
  onToggleAutoScroll,
  anns,
  onJumpAnn,
  onDeleteAnn,
  searchOpen,
  onSearchOpenChange,
  searchQuery,
  onSearchQueryChange,
  onSearchSubmit,
  searchSummary,
  searchHits,
  searchLoading,
  searchError,
  onOpenSearchHit,
  onAddSearchHitHighlight,
  prefs,
  onPrefsChange,
  onEdit,
}) => (
  <>
    <span className="novel-chrome-title">
      <ReadOutlined aria-hidden />{title || '未命名章节'}
    </span>
    <span className="novel-chrome-sub">· {totalWords.toLocaleString()} 字</span>
    <span className="novel-chrome-spacer" />
    <span className="novel-chrome-save-state">
      <i className={`novel-dot ${saved ? 'ok' : 'dirty'}`} aria-hidden />
      <span style={{ color: saved ? 'var(--color-success)' : 'var(--color-warning)' }}>
        {saved ? '已保存' : '未保存'}
      </span>
    </span>
    <div className="novel-chrome-nav">
      <Tooltip title="上一章"><Button size="small" icon={<LeftOutlined />} onClick={onPrev} type="text" disabled={atFirst} aria-label="上一章" /></Tooltip>
      <Tooltip title="下一章"><Button size="small" icon={<RightOutlined />} onClick={onNext} type="text" disabled={atLast} aria-label="下一章" /></Tooltip>
    </div>
    <Popover
      trigger="click"
      placement="bottomRight"
      open={bookmarkOpen}
      onOpenChange={onBookmarkOpenChange}
      content={(
        <div className="novel-read-bookmarks">
          <div className="novel-read-bookmarks-head">
            <span>本章书签（{bookmarks.length}）</span>
            <Button size="small" type="text" onClick={onToggleBookmark} aria-label="在当前位置添加书签">＋ 此处</Button>
          </div>
          {bookmarks.length === 0 ? (
            <div className="novel-read-bookmarks-empty">滚动到想记住的位置，点「＋ 此处」添加书签</div>
          ) : (
            <div className="novel-read-bookmarks-list">
              {bookmarks.map((b) => (
                <div
                  key={b.createdAt}
                  className="novel-read-bookmark"
                  role="button"
                  tabIndex={0}
                  onClick={() => onJumpBookmark(b)}
                  onKeyDown={(e) => { if (e.key === 'Enter') onJumpBookmark(b) }}
                >
                  <span className="novel-read-bookmark-pct">{b.pct}%</span>
                  <span className="novel-read-bookmark-text">{b.text || '（无摘录）'}</span>
                  <Button
                    size="small"
                    type="text"
                    icon={<CloseOutlined />}
                    aria-label="删除书签"
                    onClick={(e) => { e.stopPropagation(); onRemoveBookmark(b) }}
                  />
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    >
      <Tooltip title={bookmarks.length > 0 ? `书签（${bookmarks.length}）` : '书签'}>
        <Button
          size="small"
          type="text"
          icon={bookmarks.length > 0 ? <PushpinFilled /> : <PushpinOutlined />}
          className={bookmarks.length > 0 ? 'is-active' : ''}
          aria-label="书签"
        />
      </Tooltip>
    </Popover>
    <Tooltip title={autoScrolling ? '停止自动滚屏' : '自动滚屏'}>
      <Button
        size="small"
        type="text"
        icon={autoScrolling ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
        className={autoScrolling ? 'is-active' : ''}
        aria-label="自动滚屏"
        onClick={onToggleAutoScroll}
      />
    </Tooltip>
    <Popover
      trigger="click"
      placement="bottomRight"
      content={(
        <div className="novel-read-anns">
          <div className="novel-read-anns-head">
            <span>本章划线 / 想法（{anns.length}）</span>
            <span className="novel-read-anns-hint">选中文字即可划线</span>
          </div>
          {anns.length === 0 ? (
            <div className="novel-read-anns-empty">拖动选中正文 → 高亮或写想法</div>
          ) : (
            <div className="novel-read-anns-list">
              {anns.map((a) => (
                <div
                  key={a.id}
                  className="novel-read-ann"
                  role="button"
                  tabIndex={0}
                  onClick={() => onJumpAnn(a)}
                  onKeyDown={(e) => { if (e.key === 'Enter') onJumpAnn(a) }}
                >
                  <i className="novel-read-ann-dot" style={{ background: ANNOTATION_COLORS[a.color] }} />
                  <span className="novel-read-ann-text">{a.text}</span>
                  {a.note && <CommentOutlined className="novel-read-ann-note" />}
                  <Button
                    size="small"
                    type="text"
                    icon={<CloseOutlined />}
                    aria-label="删除划线"
                    onClick={(e) => { e.stopPropagation(); onDeleteAnn(a.id) }}
                  />
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    >
      <Tooltip title={anns.length > 0 ? `划线 / 想法（${anns.length}）` : '划线 / 想法'}>
        <Button
          size="small"
          type="text"
          icon={<HighlightOutlined />}
          className={anns.length > 0 ? 'is-active' : ''}
          aria-label="划线 / 想法"
        />
      </Tooltip>
    </Popover>
    <Popover
      trigger="click"
      placement="bottomRight"
      open={searchOpen}
      onOpenChange={onSearchOpenChange}
      content={(
        <div className="novel-read-search">
          <Input
            size="small"
            value={searchQuery}
            onChange={(e) => onSearchQueryChange(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); onSearchSubmit() } }}
            placeholder="搜索全书（标题 + 正文）"
            allowClear
          />
          {searchHits.length > 0 && (
            <div className="novel-read-search-hint">
              共 {searchSummary.total} 处 · {searchSummary.chapters} 章
              {searchSummary.shown < searchSummary.total ? `（显示前 ${searchSummary.shown} 条）` : ''}
            </div>
          )}
          <div className="novel-read-search-body">
            {searchLoading ? (
              <div className="novel-read-search-hint"><LoadingOutlined spin /> 搜索中…</div>
            ) : searchError ? (
              <div className="novel-read-search-hint">{searchError}</div>
            ) : searchQuery.trim() && searchHits.length === 0 ? (
              <div className="novel-read-search-hint">没有找到「{searchQuery.trim()}」</div>
            ) : (
              <div className="novel-read-search-list">
                {searchHits.map((h) => (
                  <div
                    key={`${h.node_id}:${h.match_index}`}
                    className="novel-read-search-hit-row"
                    role="button"
                    tabIndex={0}
                    onClick={() => onOpenSearchHit(h)}
                    onKeyDown={(e) => { if (e.key === 'Enter') onOpenSearchHit(h) }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      <span className="novel-read-search-hit-title" style={{ flex: 1, minWidth: 0 }}>
                        {h.title}{h.paragraph_index >= 0 ? ` · 第${h.paragraph_index + 1}段` : ''}
                      </span>
                      <Tooltip title={h.paragraph_index >= 0 ? '把该命中文本写为永久划线标注' : '标题命中无法落为正文划线'}>
                        <Button
                          size="small"
                          type="text"
                          icon={<HighlightOutlined />}
                          disabled={h.paragraph_index < 0}
                          aria-label="落为划线"
                          onClick={(e) => { e.stopPropagation(); onAddSearchHitHighlight(h) }}
                          // 键盘 Enter 落划线时不冒泡触发行自身的定位跳转
                          onKeyDown={(e) => e.stopPropagation()}
                        >
                          落为划线
                        </Button>
                      </Tooltip>
                    </div>
                    <span className="novel-read-search-hit-snippet">
                      {splitSnippet(h.snippet, searchQuery.trim()).map((seg, i) => (
                        seg.match
                          ? <mark key={i}>{seg.text}</mark>
                          : <React.Fragment key={i}>{seg.text}</React.Fragment>
                      ))}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    >
      <Tooltip title="全文搜索">
        <Button size="small" type="text" icon={<SearchOutlined />} aria-label="全文搜索" />
      </Tooltip>
    </Popover>
    <Popover
      trigger="click"
      placement="bottomRight"
      content={<ReadingPrefsPanel prefs={prefs} onChange={onPrefsChange} />}
    >
      <Tooltip title="排版 / 主题 / 亮度 / 滚屏">
        <Button size="small" type="text" icon={<FontSizeOutlined />} aria-label="阅读排版" />
      </Tooltip>
    </Popover>
    <Tooltip title="返回编辑">
      <Button size="small" icon={<EditOutlined />} onClick={onEdit}>编辑</Button>
    </Tooltip>
  </>
)

export default ReadingChrome