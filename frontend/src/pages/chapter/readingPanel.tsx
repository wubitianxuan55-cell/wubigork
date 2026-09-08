/**
 * ReadingPanel — 阅读模式正文面板（进度条 / 滚动容器 / AI 摘要 / 段落渲染 / 页脚导航），
 * 自 ChapterPage 原样搬移：纯受控展示组件，滚动根 ref 与 readMode 无关的回调经 props
 * 传入；正文段落切分（空行 → 段）与摘要交互与拆分前逐字节一致。
 */
import React from 'react'
import { Button, Tooltip } from 'antd'
import {
  LeftOutlined, RightOutlined, ThunderboltOutlined, DownOutlined, LoadingOutlined, ExportOutlined,
} from '@ant-design/icons'
import TTSPlayer from '../../components/TTSPlayer'
import { READING_COLUMN_WIDTH, type ReadingSettings } from '../../utils/readingSettings'

export interface ReadingPanelProps {
  scrollRef: React.Ref<HTMLDivElement>
  onScroll: () => void
  onMouseUp: () => void
  progress: number
  prefs: ReadingSettings
  title: string
  scenes: string[]
  summaryOpen: boolean
  onToggleSummary: () => void
  summaryLoading: boolean
  summaryText: string | null
  summaryError: string | null
  onRetrySummary: () => void
  onTtsSentence: (sentence: string) => void
  onTtsClear: () => void
  atFirst: boolean
  atLast: boolean
  onPrev: () => void
  onNext: () => void
  totalWords: number
  onExport: () => void
}

const ReadingPanel: React.FC<ReadingPanelProps> = ({
  scrollRef,
  onScroll,
  onMouseUp,
  progress,
  prefs,
  title,
  scenes,
  summaryOpen,
  onToggleSummary,
  summaryLoading,
  summaryText,
  summaryError,
  onRetrySummary,
  onTtsSentence,
  onTtsClear,
  atFirst,
  atLast,
  onPrev,
  onNext,
  totalWords,
  onExport,
}) => (
  <>
    <div className="novel-reading-progress" aria-hidden>
      <i style={{ width: `${progress}%` }} />
    </div>
    <div
      className="novel-reading-scroll"
      ref={scrollRef}
      onScroll={onScroll}
      onMouseUp={onMouseUp}
      data-read-theme={prefs.theme}
      style={{ filter: `brightness(${prefs.brightness}%)` }}
    >
      <div
        className="novel-reading-column"
        style={{
          fontSize: prefs.fontSize,
          lineHeight: prefs.lineHeight,
          maxWidth: READING_COLUMN_WIDTH[prefs.column],
        }}
      >
        <h2 className="novel-reading-title">{title || '未命名章节'}</h2>
        <div className="novel-read-summary">
          <button
            type="button"
            className="novel-read-summary-head"
            onClick={onToggleSummary}
            aria-expanded={summaryOpen}
          >
            <ThunderboltOutlined className="novel-read-summary-ic" />
            <span>AI 摘要</span>
            {summaryLoading
              ? <LoadingOutlined className="novel-read-summary-loading" />
              : <DownOutlined className={`novel-read-summary-chev${summaryOpen ? ' is-open' : ''}`} />}
          </button>
          {summaryOpen && (
            <div className="novel-read-summary-body">
              {summaryLoading ? (
                <span className="novel-read-summary-hint">AI 正在阅读本章…</span>
              ) : summaryText ? (
                <div className="novel-read-summary-text">{summaryText}</div>
              ) : summaryError ? (
                <div className="novel-read-summary-error">
                  <span>{summaryError}</span>
                  <Button size="small" type="text" onClick={() => void onRetrySummary()}>重试</Button>
                </div>
              ) : (
                <span className="novel-read-summary-hint">展开即生成，仅使用本章本地文本</span>
              )}
            </div>
          )}
        </div>
        {scenes.map((scene, i) => {
          const paras = scene.split(/\n\s*\n/).map((s) => s.trim()).filter(Boolean)
          return (
            <React.Fragment key={i}>
              {i > 0 && <div className="novel-reading-scene-sep" aria-hidden>＊ ＊ ＊</div>}
              {paras.length === 0
                ? <p className="novel-reading-p">{scene || '（本章暂无内容）'}</p>
                : paras.map((p, j) => <p key={j} className="novel-reading-p">{p}</p>)}
            </React.Fragment>
          )
        })}
      </div>
    </div>
    {/* 阅读页脚：章节导航 */}
    <div className="novel-reading-foot">
      <TTSPlayer
        getText={() => scenes.join('\n\n') || ''}
        onSentence={onTtsSentence}
        onClear={onTtsClear}
      />
      <Button size="small" icon={<LeftOutlined />} onClick={onPrev} disabled={atFirst}>上一章</Button>
      <span style={{ fontSize: 11, color: 'var(--color-text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
        {title || '未命名章节'} · {totalWords.toLocaleString()} 字
      </span>
      <Button size="small" onClick={onNext} disabled={atLast}>下一章<RightOutlined /></Button>
      <Tooltip title="导出全部格式">
        <Button size="small" type="text" icon={<ExportOutlined />} aria-label="导出小说" onClick={onExport} />
      </Tooltip>
    </div>
  </>
)

export default ReadingPanel