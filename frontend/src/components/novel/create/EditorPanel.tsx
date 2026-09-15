import React, { useRef } from 'react'
import { Button, Spin, Typography, Input, Tooltip, Space, message } from 'antd'
import {
  EditOutlined, LoadingOutlined, ReloadOutlined, SaveOutlined,
  PlusOutlined, ThunderboltOutlined, StopOutlined,
} from '@ant-design/icons'
import { C } from '../../../utils/theme'
import { countTextChars } from '../../../utils/text'
import { chapterLabel } from './outlineTree'
import type { OutlineNode } from '../../../types'

const { TextArea } = Input

interface EditorPanelProps {
  activeNode: OutlineNode | null
  content: string
  onContentChange: (value: string) => void
  chapterLoading: boolean
  generating: boolean
  genPhase: string
  genPercent: number
  /** 停止请求进行中（按钮 loading，避免重复点击） */
  stopping: boolean
  saving: boolean
  onRegenerate: () => void
  onSave: () => void
  /** 停止生成：调用 NovelB.CancelCreateChapter（T6-7.2） */
  onStop: () => void
  hasChapters: boolean
  nextChapterNum: number
  onOpenWizard: (prevChapter: number) => void
  /** 正文编辑字号（px） */
  editorFontSize?: number
  onEditorFontSizeChange?: (value: number) => void
  /** 局部重写入口（t4-C3 余项）：start/end 为正文 rune 偏移（code-unit 选区已换算），text 为选中文本 */
  onPartialRewrite?: (sel: { start: number; end: number; text: string }) => void
}

/** 中部编辑器面板（T6-7.5 从 CreatePage 拆分）：标题/状态标签/字数 + 正文编辑 + 生成进度与停止按钮 + 局部重写入口 */
const EditorPanel: React.FC<EditorPanelProps> = ({
  activeNode, content, onContentChange, chapterLoading,
  generating, genPhase, genPercent, stopping, saving,
  onRegenerate, onSave, onStop, hasChapters, nextChapterNum, onOpenWizard,
  editorFontSize = 15, onEditorFontSizeChange, onPartialRewrite,
}) => {
  const editorRef = useRef<React.ComponentRef<typeof TextArea> | null>(null)

  // 局部重写入口：读原生 textarea 的 code-unit 选区（selectionStart/End 为 UTF-16
  // code-unit 偏移），换算成 rune 偏移后回调（后端选段校验按 rune 口径）；无选区给提示。
  const handlePartialRewrite = () => {
    const ta = editorRef.current?.resizableTextArea?.textArea
    if (!ta) return
    const s = ta.selectionStart
    const e = ta.selectionEnd
    if (s === e) {
      message.warning('请先选中要重写的文字段落')
      return
    }
    const toRune = (codeUnitPos: number): number => [...content.slice(0, codeUnitPos)].length
    onPartialRewrite?.({ start: toRune(s), end: toRune(e), text: content.slice(s, e) })
  }

  return (
  <section
    className="novel-editor-panel novel-workspace-col novel-editor-col"
    style={{ '--novel-editor-font-size': `${editorFontSize}px` } as React.CSSProperties}
  >
    <div className="novel-panel-head">
      <span className="novel-panel-title">
        {activeNode ? (activeNode.title || chapterLabel(activeNode.order_index)) : '正文编辑'}
      </span>
      {activeNode?.status === 'writing' && <span className="novel-tag-tone is-warning">生成中</span>}
      {activeNode?.status === 'done' && <span className="novel-tag-tone is-success">已写入</span>}
      <div style={{ flex: 1 }} />
      <span className="novel-setting-meta">{countTextChars(content).toLocaleString()} 字</span>
      <Space size={2}>
        <Tooltip title="减小字号">
          <Button
            size="small"
            aria-label="减小字号"
            disabled={editorFontSize <= 12}
            onClick={() => onEditorFontSizeChange?.(Math.max(12, editorFontSize - 1))}
            style={{ padding: '0 7px', fontSize: 11, color: C('color-text-secondary') }}
          >
            A−
          </Button>
        </Tooltip>
        <Tooltip title="增大字号">
          <Button
            size="small"
            aria-label="增大字号"
            disabled={editorFontSize >= 24}
            onClick={() => onEditorFontSizeChange?.(Math.min(24, editorFontSize + 1))}
            style={{ padding: '0 7px', fontSize: 11, color: C('color-text-secondary') }}
          >
            A+
          </Button>
        </Tooltip>
        <Tooltip title="正文字号">
          <span style={{ fontSize: 11, color: C('color-text-secondary'), minWidth: 34, textAlign: 'center', display: 'inline-block' }}>
            {editorFontSize}px
          </span>
        </Tooltip>
      </Space>
      {activeNode && (
        <Button size="small" icon={<ReloadOutlined />} onClick={onRegenerate}>重写</Button>
      )}
      {activeNode && !generating && (
        <Button size="small" onClick={handlePartialRewrite}>局部重写</Button>
      )}
      {activeNode && (
        <Button size="small" type="primary" icon={<SaveOutlined />} onClick={onSave} loading={saving}>保存</Button>
      )}
    </div>

    <div className="novel-editor-body">
      {chapterLoading ? (
        <div className="novel-editor-loading"><Spin /></div>
      ) : (activeNode || generating) ? (
        <TextArea className="novel-editor" ref={editorRef} value={content} onChange={e => onContentChange(e.target.value)}
          placeholder="AI 将在此流式呈现正文；也可直接手写后保存…"
        />
      ) : (
        <div className="novel-editor-empty">
          <EditOutlined className="novel-editor-empty-icon" />
          {hasChapters ? (
            <>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>
                选择左侧章节查看，或生成下一章
              </Typography.Text>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => onOpenWizard(nextChapterNum - 1)}>
                生成第 {nextChapterNum} 章
              </Button>
            </>
          ) : (
            <>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>
                开始创作你的第一部章节
              </Typography.Text>
              <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => onOpenWizard(0)}>
                生成第 1 章
              </Button>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                可先在右侧选择写作技能、设置目标字数与温度，再构思剧情方向
              </Typography.Text>
            </>
          )}
        </div>
      )}
    </div>

    {generating && (
      <div className="novel-gen-status">
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <LoadingOutlined />
          <span>{genPhase}</span>
          <div style={{ flex: 1 }} />
          <Button size="small" danger icon={<StopOutlined />} loading={stopping} onClick={onStop}>
            停止生成
          </Button>
        </div>
        <div className="novel-gen-progress-track">
          <div className="novel-gen-progress-fill" style={{ width: `${genPercent}%` }} />
        </div>
      </div>
    )}
  </section>
  )
}

export default EditorPanel
