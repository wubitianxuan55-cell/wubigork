/**
 * ReadingOverlays — 阅读模式浮层组（划词浮动工具条 / 想法编辑弹窗 / AI 问书弹窗），
 * 自 ChapterPage 原样搬移：纯受控展示组件，全部状态与回调经 props 传入；
 * 顺序语义（清选区 → 收起工具条 → 开问书）与拆分前一致。
 */
import React from 'react'
import { Button, Input, Modal } from 'antd'
import { CommentOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { ANNOTATION_COLORS, type ReadingAnnotation, type AnnotationColor } from '../../utils/readingAnnotations'
import type { ReadingAskMessage } from './readingAskSession'

export interface ReadingOverlaysProps {
  readMode: boolean
  selToolbar: { x: number; y: number } | null
  selText: string
  onClearSel: () => void
  onHighlight: (color: AnnotationColor, withNote: boolean, text: string) => void
  onAskSelection: (sel: string) => void
  noteTarget: ReadingAnnotation | null
  onNoteClose: () => void
  noteDraft: string
  onNoteDraftChange: (v: string) => void
  onSaveNote: () => void
  onDeleteAnnotation: (id: string) => void
  askTarget: { selection: string } | null
  onAskClose: () => void
  askMessages: ReadingAskMessage[]
  askLoading: boolean
  askError: string | null
  askQuestion: string
  onAskQuestionChange: (v: string) => void
  onRunAsk: () => void
  onClearAsk: () => void
  askThreadRef: React.Ref<HTMLDivElement>
}

const ReadingOverlays: React.FC<ReadingOverlaysProps> = ({
  readMode,
  selToolbar,
  selText,
  onClearSel,
  onHighlight,
  onAskSelection,
  noteTarget,
  onNoteClose,
  noteDraft,
  onNoteDraftChange,
  onSaveNote,
  onDeleteAnnotation,
  askTarget,
  onAskClose,
  askMessages,
  askLoading,
  askError,
  askQuestion,
  onAskQuestionChange,
  onRunAsk,
  onClearAsk,
  askThreadRef,
}) => (
  <>
    {/* 划词工具条：选中文字后浮动在选区上方 */}
    {readMode && selToolbar && (
      <div
        className="novel-read-selbar"
        style={{ left: selToolbar.x, top: selToolbar.y }}
        role="toolbar"
        aria-label="划词工具"
      >
        {(Object.keys(ANNOTATION_COLORS) as AnnotationColor[]).map((c) => (
          <button
            key={c}
            type="button"
            className="novel-read-selbar-swatch"
            style={{ background: ANNOTATION_COLORS[c] }}
            title={`高亮（${c}）`}
            aria-label={`高亮（${c}）`}
            onClick={() => onHighlight(c, false, selText)}
          />
        ))}
        <span className="novel-read-selbar-divider" aria-hidden />
        <button type="button" className="novel-read-selbar-note" onClick={() => onHighlight('yellow', true, selText)}>
          <CommentOutlined /> 想法
        </button>
        <span className="novel-read-selbar-divider" aria-hidden />
        <button
          type="button"
          className="novel-read-selbar-note"
          onClick={() => { window.getSelection()?.removeAllRanges(); onClearSel(); onAskSelection(selText) }}
        >
          <ThunderboltOutlined /> 问书
        </button>
      </div>
    )}

    {/* 想法编辑弹窗 */}
    <Modal
      open={!!noteTarget}
      onCancel={onNoteClose}
      title={noteTarget ? `想法 · ${noteTarget.title}` : '想法'}
      footer={null}
      width={420}
    >
      {noteTarget && (
        <div className="novel-read-note">
          <blockquote className="novel-read-note-quote">{noteTarget.text}</blockquote>
          <Input.TextArea
            rows={4}
            value={noteDraft}
            onChange={(e) => onNoteDraftChange(e.target.value)}
            placeholder="写点什么…（保存后随高亮展示）"
          />
          <div className="novel-read-note-actions">
            <Button danger size="small" onClick={() => onDeleteAnnotation(noteTarget.id)}>删除高亮</Button>
            <div style={{ flex: 1 }} />
            <Button size="small" onClick={onNoteClose}>取消</Button>
            <Button type="primary" size="small" onClick={onSaveNote}>保存想法</Button>
          </div>
        </div>
      )}
    </Modal>

    {/* AI 问书弹窗（会话式：同一章内连续追问，历史随请求回传） */}
    <Modal
      open={!!askTarget}
      onCancel={onAskClose}
      title="AI 问书"
      footer={null}
      width={560}
    >
      {askTarget && (
        <div className="novel-read-ask">
          <blockquote className="novel-read-note-quote">{askTarget.selection}</blockquote>
          {askMessages.length > 0 && (
            <div className="novel-read-ask-thread" ref={askThreadRef}>
              {askMessages.map((m, i) => (
                <div key={i} className={`novel-read-ask-msg is-${m.role}`}>
                  <span className="novel-read-ask-msg-role">{m.role === 'user' ? '我' : 'AI'}</span>
                  <div className="novel-read-ask-msg-body">{m.content}</div>
                </div>
              ))}
              {askLoading && (
                <div className="novel-read-ask-msg is-assistant is-pending">
                  <span className="novel-read-ask-msg-role">AI</span>
                  <div className="novel-read-ask-msg-body">正在思考…</div>
                </div>
              )}
            </div>
          )}
          {askError && (
            <div className="novel-read-ask-error">
              <span>{askError}</span>
              <Button size="small" type="text" onClick={() => void onRunAsk()}>重试</Button>
            </div>
          )}
          <Input.TextArea
            rows={2}
            value={askQuestion}
            onChange={(e) => onAskQuestionChange(e.target.value)}
            placeholder={askMessages.length > 0 ? '继续追问，例如：那他后来呢？' : '针对摘选内容提问，例如：这句话暗示了什么？'}
          />
          <div className="novel-read-ask-actions">
            {askMessages.length > 0 && (
              <Button size="small" type="text" onClick={onClearAsk}>清空会话</Button>
            )}
            <div style={{ flex: 1 }} />
            <Button size="small" onClick={onAskClose}>关闭</Button>
            <Button
              type="primary"
              size="small"
              loading={askLoading}
              disabled={!askQuestion.trim()}
              onClick={() => void onRunAsk()}
            >
              {askMessages.length > 0 ? '追问' : '提问'}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  </>
)

export default ReadingOverlays