// 消息行（T-perf 性能优化）：从 MessageList 抽出的单行渲染组件，渲染契约与
// 原实现逐一对应——user/assistant 气泡、流式光标、推理块、错误块与复制/朗读操作。
// 保持「纯展示 + 浅比较友好」：props 全为原始值 / 稳定对象（msg）/ 恒稳回调，
// memo 边界在 MessageList 侧统一施加（React.memo(ChatRow)），流式 chunk 期间
// 非流式行可整行跳过重渲染。
import React from 'react'
import { Button, Tooltip } from 'antd'
import {
  CopyOutlined, CheckOutlined, SoundOutlined, BulbOutlined,
  ReloadOutlined, CloseCircleOutlined, BarChartOutlined, UpOutlined, DownOutlined,
  GlobalOutlined, EditOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import ChatMarkdown from '../ChatMarkdown'
import { MarkdownContent } from '../MarkdownContent'
import AnsweredByLine from './AnsweredByLine'
import type { AnsweredByInfo } from './AnsweredByLine'
import { SearchCard } from './SearchCard'
import type { ChatMsg } from '../../pages/chat/types'
import { useGenuiScope } from '../../genui/scope'
import { buildMarkdownGenuiOverrides } from './genuiAdapter'
import { findStableCut } from '../../gaea/lib/markdownCut'
import { useThrottledStreamText } from '../../hooks/useThrottledStreamText'
import { navigateToModelCenter } from '../../pages/chat/utils'
import { btOf, dbIdOf, type MsgVersions } from '../../pages/chat/branches'
import { BranchSwitcher } from './BranchSwitcher'
import type { ChatSearchReference } from './searchCardModel'

export interface ChatRowProps {
  /** 行对应的消息对象：未更新的行保持同一对象引用（updateMessage 仅替换被补丁的行），memo 才能命中 */
  msg: ChatMsg
  /** 展示文本：流式行 = 最新 streamText，其余行 = msg.content */
  text: string
  /** 当前流式回合的最小进度（理解 / 搜索 / 整理 / 降级）。 */
  streamStatus?: string
  /** 思考流实时文本（reasoning 帧节流 flush；流式思考期间展开渲染，done 后由 msg.reasoning 接管） */
  streamReasoning?: string
  /** 本行是否为当前流式行（msg.streaming && msg.key === streamKey） */
  isStreaming: boolean
  /** 是否与上一条消息（含窗口外相邻行）角色不同 → 显示 AI 名牌 */
  newGroup: boolean
  mode: string
  companionName: string
  /** 本行是否处于「已复制」态（由 copiedId === msg.key 派生为行级布尔，保证 memo 浅比较只影响相关行） */
  copied: boolean
  /** 本行是否处于「朗读中」态（由 speakingId === msg.key 派生） */
  speaking: boolean
  onCopy: (content: string, id: string) => void
  onSpeak: (content: string, id: string) => void
  onRetry: (msgKey: string) => void
  /** A 批分支树：本行兄弟版本（无/单版本为 null，不渲染切换器） */
  versions?: MsgVersions | null
  onSelectVersion?: (msgId: number) => void
  /** A 批编辑提问：提交某 db 用户行的新文本 */
  onEditSubmit?: (msgKey: string, text: string) => void
  /** A 批重新生成：某 db 回答行下挂新版本 */
  onRegenerate?: (msgKey: string) => void
  /** D 批：打开全量搜索来源面板（SearchCard「查看全部来源」入口） */
  onOpenSources?: (refs: ChatSearchReference[], queries: string[]) => void
  /** A 批续写：对已停止回答行发起 prefill 续写 */
  onContinue?: (msgKey: string) => void
  /** 会话级流式占线（切版本/编辑/重生成禁用，DeepSeek busy 同款语义） */
  busy?: boolean
}

export const ChatRow: React.FC<ChatRowProps> = ({
  msg, text, streamStatus, streamReasoning, isStreaming, newGroup, mode, companionName,
  copied, speaking, onCopy, onSpeak, onRetry,
  versions, onSelectVersion, onEditSubmit, onRegenerate, onOpenSources, onContinue, busy,
}) => {
  const display = useThrottledStreamText(text, isStreaming)
  // C1（chat-demo 配方）：流式行按渲染耗时自适应节流 display 更新——与下方
  // findStableCut 分段互补，重尾段（长表格/代码块）高频 reparse 被压到 ≤2s 一帧；
  // 流结束 enable=false 直通最新全文，终帧不延迟。非流式行（含 user 行）零影响。
  const [metricsOpen, setMetricsOpen] = React.useState(true)
  // A 批编辑提问行内编辑器（chat-demo UserMessage 同位）：仅 db 用户行可编辑
  //（乐观/流式行未落库无兄弟语义）；首条提问（bt.p===0）v1 不支持——根无父行
  // 存选择指针，见设计文档 §5。
  const [editing, setEditing] = React.useState(false)
  const [editDraft, setEditDraft] = React.useState('')
  const rowDbId = dbIdOf(msg.key)
  const rowBt = btOf(msg)
  const canEdit = msg.role === 'user' && rowDbId !== null && (rowBt?.p ?? 0) > 0
    && !!onEditSubmit && !busy && !isStreaming
  const canRegenerate = msg.role === 'assistant' && rowDbId !== null && (rowBt?.p ?? 0) > 0
    && !!onRegenerate && !busy && !isStreaming
  // A 批续写准入：已停止落库行（Go cancelled 轮打 extra.stopped）+ 非占线
  const canContinue = msg.role === 'assistant' && rowDbId !== null && !msg.error
    && msg.extra?.stopped === true && !!onContinue && !busy && !isStreaming
  const submitEdit = () => {
    const t = editDraft.trim()
    if (!t || !canEdit) return
    onEditSubmit?.(msg.key, t)
    setEditing(false)
  }
  // v4.368 流式分段渲染：把累积文本切成「稳定段（段落边界，不再变化）+ 增长尾段」，
  // 稳定段字符串不变时其 MarkdownContent 子树被 memo 跳过——每 chunk 实际重解析的
  // 只有尾段（通常一段），消除原「每 chunk 全量 reparse」的 O(n²)。
  const [stablePart, pendingPart] = React.useMemo(() => findStableCut(display), [display])
  const genuiScope = useGenuiScope()
  const genuiOverrides = React.useMemo(
    () => buildMarkdownGenuiOverrides(genuiScope, msg.key),
    [genuiScope, msg.key],
  )
  if (msg.role === 'user') {
    return (
      <div className="chat-row chat-row-user">
        {editing ? (
          <div className="chat-edit-editor">
            <textarea
              aria-label="编辑提问"
              autoFocus
              rows={2}
              value={editDraft}
              disabled={busy}
              onChange={(e) => setEditDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.nativeEvent.isComposing) return
                if (e.key === 'Escape') { setEditing(false) }
                else if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submitEdit() }
              }}
              style={{
                width: '100%', boxSizing: 'border-box', resize: 'vertical', minHeight: 52,
                maxHeight: 202, padding: '8px 10px', borderRadius: 10, fontSize: 13.5, lineHeight: 1.6,
                background: 'var(--md-sys-color-surface-variant)', color: C('color-text'),
                border: '1px solid var(--md-sys-color-outline-variant)', outline: 'none',
              }}
            />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 6 }}>
              <Button size="small" onClick={() => setEditing(false)}
                style={{ fontSize: 12, height: 26, borderRadius: 8 }}>取消</Button>
              <Button size="small" type="primary" disabled={busy || !editDraft.trim()} onClick={submitEdit}
                style={{ fontSize: 12, height: 26, borderRadius: 8 }}>发送</Button>
            </div>
          </div>
        ) : (
          <div className="chat-user-capsule">{display}</div>
        )}
        {msg.content && !msg.streaming && (
          <div className="chat-msg-actions">
            <Tooltip title={copied ? '已复制' : '复制'}>
              <Button type="text" size="small" aria-label={copied ? '已复制' : '复制'} icon={copied ? <CheckOutlined style={{ color: 'var(--md-sys-color-success)' }} /> : <CopyOutlined />}
                onClick={() => onCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
            </Tooltip>
            {canEdit && !editing && (
              <Tooltip title="编辑提问">
                <Button type="text" size="small" aria-label="编辑提问" icon={<EditOutlined />}
                  onClick={() => { setEditDraft(msg.content); setEditing(true) }}
                  style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
              </Tooltip>
            )}
            <BranchSwitcher ids={versions?.ids ?? []} index={versions?.index ?? 0}
              disabled={!!busy} onSelect={(id) => onSelectVersion?.(id)} />
          </div>
        )}
      </div>
    )
  }
  return (
    <div className="chat-row chat-row-assistant">
      {newGroup && (
        <div className="chat-assistant-name">
          <span className="ai-dot" />
          {mode === 'plain' ? 'gaea AI 助手' : `${companionName} · AI`}
        </div>
      )}
      {isStreaming && streamStatus && (
        <div className="chat-stream-status" role="status" aria-live="polite">
          <GlobalOutlined aria-hidden />
          <span>{streamStatus}</span>
        </div>
      )}
      {isStreaming && streamReasoning && (
        <details className="chat-reasoning" open>
          <summary><BulbOutlined style={{ marginRight: 4 }} />思考中…</summary>
          <div className="chat-reasoning-body">{streamReasoning}</div>
        </details>
      )}
      {!msg.error && msg.reasoning && (
        <details className="chat-reasoning">
          <summary><BulbOutlined style={{ marginRight: 4 }} />思考过程</summary>
          <div className="chat-reasoning-body">{msg.reasoning}</div>
        </details>
      )}
      {msg.error ? (
        <div className="chat-error-block">
          <CloseCircleOutlined style={{ marginTop: 2, flexShrink: 0 }} />
          <div style={{ flex: 1, minWidth: 0 }}>
            {display}
            {(() => {
              // B1（chat-demo describe.ts 对齐）：结构化错误码 → 引导动作；
              // 原始报错降级为小字详情保留排障线索。无码（旧消息/未分类）零变化。
              const errorCode = typeof msg.extra?.error_code === 'string' ? msg.extra.error_code : ''
              const errorDetail = typeof msg.extra?.error_detail === 'string' ? msg.extra.error_detail : ''
              const needsModelCenter = errorCode === 'engine_unreachable' || errorCode === 'auth'
              return (
                <>
                  {errorCode && errorDetail ? (
                    <div style={{ marginTop: 4, fontSize: 11, color: C('color-text-secondary'), wordBreak: 'break-all' }}>{errorDetail}</div>
                  ) : null}
                  <div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
                    {needsModelCenter && (
                      <Button size="small" onClick={() => navigateToModelCenter()}
                        style={{ fontSize: 12, height: 26, borderRadius: 8 }}>
                        打开模型中心
                      </Button>
                    )}
                    <Button size="small" icon={<ReloadOutlined />} onClick={() => onRetry(msg.key)}
                      style={{ fontSize: 12, height: 26, borderRadius: 8 }}>
                      重试
                    </Button>
                  </div>
                </>
              )
            })()}
          </div>
        </div>
      ) : (
        <div className="chat-assistant-text">
          {isStreaming ? (
            <span className="chat-streaming">
              {display ? <><MarkdownContent source={stablePart} className="md-content" /><MarkdownContent source={pendingPart} className="md-content" /><span className="cursor-blink" /></> : <span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span>}
            </span>
          ) : (
            mode === 'plain'
              ? <ChatMarkdown text={display} genuiKey={msg.key} />
              : <MarkdownContent source={display} className="md-content" components={genuiOverrides} />
          )}
        </div>
      )}
      {!msg.error && !msg.streaming && <SearchCard value={msg.extra?.search} onOpenSources={onOpenSources} />}
      {msg.content && !msg.streaming && (
        <div className={`chat-msg-actions${msg.extra?.answered_by != null ? ' has-metrics' : ''}`}>
          <Tooltip title={copied ? '已复制' : '复制'}>
            <Button type="text" size="small" aria-label={copied ? '已复制' : '复制'} icon={copied ? <CheckOutlined style={{ color: 'var(--md-sys-color-success)' }} /> : <CopyOutlined />}
              onClick={() => onCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
          </Tooltip>
          {!msg.error && (
            <Tooltip title={speaking ? '朗读中…' : '朗读'}>
              <Button type="text" size="small" aria-label={speaking ? '朗读中' : '朗读'} icon={<SoundOutlined />} loading={speaking}
                onClick={() => onSpeak(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
            </Tooltip>
          )}
          {/* A 批重新生成：db 回答行下挂新版本（chat-demo regenerateReply 同位） */}
          {canRegenerate && (
            <Tooltip title="重新生成">
              <Button type="text" size="small" aria-label="重新生成" icon={<ReloadOutlined />}
                onClick={() => onRegenerate?.(msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
            </Tooltip>
          )}
          {/* A 批续写：已停止回答行 prefill 续写（chat-demo continueReply 子集） */}
          {canContinue && (
            <Button size="small" onClick={() => onContinue?.(msg.key)}
              style={{ fontSize: 11.5, height: 22, borderRadius: 7, padding: '0 8px' }}>
              继续生成
            </Button>
          )}
          {!msg.error && msg.extra?.answered_by != null && (
            <button
              type="button"
              className="chat-metrics-toggle"
              aria-expanded={metricsOpen}
              onClick={() => setMetricsOpen(open => !open)}
            >
              <BarChartOutlined aria-hidden />
              {metricsOpen ? '收起指标' : '输出指标'}
              {metricsOpen ? <UpOutlined aria-hidden /> : <DownOutlined aria-hidden />}
            </button>
          )}
          <BranchSwitcher ids={versions?.ids ?? []} index={versions?.index ?? 0}
            disabled={!!busy} onSelect={(id) => onSelectVersion?.(id)} />
        </div>
      )}
      {/* v4.15：消息级「由谁回答/为何/花了多少」回显——仅当 extra.answered_by 存在时渲染
          （旧消息/旧事件无此字段 → 零渲染，向后兼容）；流式/错误行不渲染。
          （!= null 显式布尔化：extra 值为 unknown，避免 && 链类型污染） */}
      {!msg.error && !msg.streaming && metricsOpen && msg.extra?.answered_by != null && (
        <AnsweredByLine info={msg.extra.answered_by as AnsweredByInfo} />
      )}
    </div>
  )
}
