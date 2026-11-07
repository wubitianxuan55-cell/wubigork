import React from 'react'
import { DownOutlined, GlobalOutlined, LinkOutlined } from '@ant-design/icons'
import { parseChatSearchSummary } from './searchCardModel'
import type { ChatSearchReference, ChatSearchSummary } from './searchCardModel'

const STATE_LABEL: Record<string, string> = {
  ok: '已完成',
  degraded: '降级结果',
  empty: '无结果',
  all_failed: '搜索失败',
}

const REASON_LABEL: Record<string, string> = {
  'primary-timeout': '主搜索源超时，当前为备用来源。',
  'primary-error': '主搜索源不可用，当前为备用来源。',
  'coverage-generic-query': '查询缺少足够实体，结果相关性可能有限。',
  'coverage-miss': '搜索结果与查询的相关性存疑。',
}

function safeExternalUrl(raw: string): string | null {
  try {
    const parsed = new URL(raw)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.toString() : null
  } catch {
    return null
  }
}

function hostLabel(raw: string): string {
  try {
    return new URL(raw).hostname.replace(/^www\./i, '')
  } catch {
    return raw
  }
}

function stateClass(state: string): string {
  if (state === 'all_failed') return 'is-error'
  if (state === 'degraded' || state === 'empty') return 'is-degraded'
  return 'is-ok'
}

function emptyMessage(summary: ChatSearchSummary): string {
  if (summary.state === 'empty') return '已检索，但没有找到与该查询相关的结果。'
  if (summary.state === 'all_failed') return '搜索源暂不可用，本次回答未取得实时信息。'
  if (summary.state === 'degraded') return '主搜索源未返回可用结果，当前没有可展示的来源。'
  return '本次检索没有可展示的来源。'
}

function metaText(summary: ChatSearchSummary): string {
  const parts: string[] = []
  if (summary.searches > 0) parts.push(`${summary.searches} 次检索`)
  const sources = Math.max(summary.sources, summary.references.length)
  if (sources > 0) parts.push(`${sources} 个来源`)
  if (summary.elapsedMs > 0) {
    const seconds = summary.elapsedMs / 1000
    parts.push(`${seconds < 10 ? seconds.toFixed(1) : Math.round(seconds)} 秒`)
  }
  return parts.length > 0 ? parts.join(' · ') : '未取得可用来源'
}

export const SearchCard: React.FC<{ value: unknown }> = ({ value }) => {
  const summary = parseChatSearchSummary(value)
  if (!summary || summary.state === 'not_requested') return null

  const validReferences = summary.references
    .map(ref => ({ ...ref, safeUrl: safeExternalUrl(ref.url) }))
    .filter((ref): ref is ChatSearchReference & { safeUrl: string } => Boolean(ref.safeUrl))
    .slice(0, 5)
  const reason = REASON_LABEL[summary.degradedReason] || (
    summary.degradedReason && !/^[a-z0-9-]+$/i.test(summary.degradedReason)
      ? summary.degradedReason
      : ''
  )

  return (
    <details className={`chat-search-card ${stateClass(summary.state)}`}>
      <summary className="chat-search-card-summary">
        <span className="chat-search-card-icon" aria-hidden><GlobalOutlined /></span>
        <span className="chat-search-card-copy">
          <strong>联网搜索</strong>
          <small>{metaText(summary)}</small>
        </span>
        <span className="chat-search-card-state">{STATE_LABEL[summary.state] || '已检索'}</span>
        <DownOutlined className="chat-search-card-chevron" aria-hidden />
      </summary>
      <div className="chat-search-card-body">
        {summary.queries.length > 0 && (
          <div className="chat-search-queries" aria-label="实际检索词">
            {summary.queries.map(query => <span key={query} className="chat-search-query">{query}</span>)}
          </div>
        )}
        {reason && <p className="chat-search-degraded-note">{reason}</p>}
        {validReferences.length > 0 ? (
          <ol className="chat-search-references">
            {validReferences.map((ref, index) => (
              <li key={`${ref.safeUrl}-${index}`}>
                <a href={ref.safeUrl} target="_blank" rel="noreferrer" className="chat-search-reference">
                  <span className="chat-search-reference-index">{index + 1}</span>
                  <span className="chat-search-reference-copy">
                    <strong>{ref.title || hostLabel(ref.url)}</strong>
                    <small>
                      <LinkOutlined aria-hidden />
                      {hostLabel(ref.url)}
                      {ref.source && <span> · {ref.source}</span>}
                    </small>
                    {ref.snippet && <span className="chat-search-reference-snippet">{ref.snippet}</span>}
                  </span>
                </a>
              </li>
            ))}
          </ol>
        ) : (
          <p className="chat-search-empty">{emptyMessage(summary)}</p>
        )}
        {summary.references.length > validReferences.length && (
          <p className="chat-search-more">仅展示前 {validReferences.length} 个有效来源</p>
        )}
      </div>
    </details>
  )
}
