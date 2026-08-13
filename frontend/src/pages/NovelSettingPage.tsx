// NovelSettingPage.tsx — 小说「设定」面板
// 纯 Markdown 文本编辑 + 直接渲染：编辑 / 分屏 / 渲染三种模式，
// 不做结构化维度拆分与任何反解析；导入导出、设定 Agent 对话直接操作整篇文本。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Button, Empty, Input, Segmented, Space, Spin, Tag, message,
} from 'antd'
import {
  ColumnWidthOutlined, EditOutlined, ExportOutlined, EyeOutlined,
  FileTextOutlined, ImportOutlined, SaveOutlined,
} from '@ant-design/icons'
import ChatPanel from '../components/ChatPanel'
import type { Message } from '../components/ChatPanel'
import { MarkdownContent, mdStyles } from '../components/MarkdownContent'
import { useAppStore } from '../stores/appStore'
import { countTextChars } from '../utils/text'
import * as App from '../../src/wailsjsCompat'

type EditorMode = 'edit' | 'split' | 'preview'

const NovelSettingPage: React.FC = () => {
  const projectPath = useAppStore((s) => s.projectPath)
  const projectOpen = useAppStore((s) => s.projectOpen)

  const [content, setContent] = useState('')
  const [savedSnapshot, setSavedSnapshot] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [mode, setMode] = useState<EditorMode>('split')
  const [messages, setMessages] = useState<Message[]>([])
  const [lastSavedAt, setLastSavedAt] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const loadToken = useRef(0)

  const loadContent = useCallback(async () => {
    const token = ++loadToken.current
    if (!projectPath) {
      setContent('')
      setSavedSnapshot('')
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const text = await App.GetWorldview()
      if (token !== loadToken.current) return
      setContent(text || '')
      setSavedSnapshot(text || '')
    } catch {
      if (token !== loadToken.current) return
      setContent('')
      setSavedSnapshot('')
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [projectPath])

  useEffect(() => { loadContent() }, [loadContent])
  useEffect(() => { setMessages([]) }, [projectPath])

  const needsProject = !projectOpen && !projectPath
  const dirty = useMemo(
    () => !needsProject && content !== savedSnapshot,
    [content, savedSnapshot, needsProject],
  )
  const wordCount = useMemo(() => countTextChars(content.trim()), [content])

  const handleSave = useCallback(async () => {
    setSaving(true)
    try {
      await App.SaveWorldview(content)
      setSavedSnapshot(content)
      setLastSavedAt(new Date().toLocaleTimeString())
      message.success('设定已保存')
    } catch (err: any) {
      message.error('保存失败: ' + (err?.message || err))
    } finally {
      setSaving(false)
    }
  }, [content])

  // Ctrl/Cmd+S 保存
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
        e.preventDefault()
        handleSave()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [handleSave])

  const handleImport = () => fileInputRef.current?.click()

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => {
      setContent((reader.result as string) || '')
      message.success(`已导入「${file.name}」`)
    }
    reader.readAsText(file)
    e.target.value = ''
  }

  const handleExport = () => {
    const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'novel_setting.md'
    a.click()
    URL.revokeObjectURL(url)
    message.success('已导出')
  }

  const handleChatSend = async (userMsg: string): Promise<string> => {
    try {
      const result: any = await App.ChatWorldview(userMsg, content)
      const reply = result?.reply || ''
      // AI 返回更新后的设定文本，直接回填编辑器（不解析、不拆分）
      if (typeof result?.worldview === 'string' && result.worldview) {
        setContent(result.worldview)
      }
      return reply
    } catch (err: any) {
      const msg = typeof err === 'string' ? err : (err?.message || err?.toString?.() || '对话失败')
      throw new Error(msg)
    }
  }

  const editorPane = (
    <Input.TextArea
      className="novel-editor"
      value={content}
      onChange={(e) => setContent(e.target.value)}
      placeholder="在此撰写或粘贴小说设定（世界观、剧情框架、人物关系、规则体系等）…"
      style={{ flex: 1, minHeight: 0, minWidth: 0, resize: 'none' }}
    />
  )

  const previewPane = (
    <div className="novel-setting-preview md-content">
      <MarkdownContent source={content} />
    </div>
  )

  return (
    <div style={{
      display: 'flex', flexDirection: 'row', width: '100%', height: '100%',
      padding: '16px', maxWidth: 1240, margin: '0 auto', gap: 14, minHeight: 0, minWidth: 0,
    }}>
      <style>{mdStyles}</style>

      {/* 左栏：设定编辑器（Markdown + 渲染） */}
      <div className="novel-panel" style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minHeight: 0 }}>
        <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 6 }}>
          <span className="novel-panel-title"><FileTextOutlined />设定编辑器</span>
          <div style={{ flex: 1 }} />
          <span className="novel-setting-meta">{wordCount.toLocaleString()} 字</span>
          <Space size={8}>
            <Button size="small" icon={<ImportOutlined />} onClick={handleImport}
              style={{ borderColor: '#60a5fa', color: '#60a5fa' }}>
              导入
            </Button>
            <Button size="small" icon={<ExportOutlined />} onClick={handleExport}
              style={{ borderColor: '#f59e0b', color: '#f59e0b' }}>
              导出
            </Button>
            <Button size="small" type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving}>
              保存
            </Button>
          </Space>
        </div>

        <input
          ref={fileInputRef}
          type="file"
          accept=".txt,.md,.json"
          style={{ display: 'none' }}
          onChange={handleFileChange}
        />

        <div className="novel-setting-toolbar">
          <Segmented
            size="small"
            value={mode}
            onChange={(val) => setMode(val as EditorMode)}
            options={[
              { label: '编辑', value: 'edit', icon: <EditOutlined /> },
              { label: '分屏', value: 'split', icon: <ColumnWidthOutlined /> },
              { label: '渲染', value: 'preview', icon: <EyeOutlined /> },
            ]}
          />
          <div style={{ flex: 1 }} />
          <Tag style={{ marginInlineEnd: 0 }} color={dirty ? 'warning' : 'success'}>
            {dirty ? '有未保存修改' : lastSavedAt ? `已保存 ${lastSavedAt}` : '无修改'}
          </Tag>
        </div>

        <div
          className="novel-setting-body"
          style={mode === 'split' ? { display: 'flex', flexDirection: 'row', gap: 12 } : { display: 'flex' }}
        >
          {needsProject ? (
            <Empty style={{ margin: 'auto' }} image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="请先在「书架」打开或创建一部小说项目" />
          ) : loading ? (
            <div style={{ margin: 'auto' }}><Spin size="large" /></div>
          ) : mode === 'edit' ? (
            editorPane
          ) : mode === 'preview' ? (
            previewPane
          ) : (
            <>
              {editorPane}
              {previewPane}
            </>
          )}
        </div>
      </div>

      {/* 右栏：设定 Agent 常驻对话 */}
      <div className="novel-panel" style={{ width: 380, flexShrink: 0, minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
        <ChatPanel
          title="设定 Agent"
          messages={messages}
          onSend={handleChatSend}
          onMessagesChange={setMessages}
          placeholder="描述你想要的设定修改，AI 帮你调整…"
          fillHeight
        />
      </div>
    </div>
  )
}

export default NovelSettingPage
