// NovelSettingPage.tsx — 小说「设定」面板
// 纯 Markdown 文本编辑 + 直接渲染：编辑 / 分屏 / 渲染三种模式，
// 不做结构化维度拆分与任何反解析；导入导出、设定 Agent 对话直接操作整篇文本。
// v4.3e/f：新增「维度化」模式（6 维度卡片分卡片编辑）与伏笔登记表 / 一致性检查面板。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Button, Empty, Input, Modal, Segmented, Space, Spin, Tag, message,
} from 'antd'
import {
  ColumnWidthOutlined, EditOutlined, ExportOutlined, EyeOutlined,
  FileTextOutlined, ImportOutlined, SaveOutlined, AppstoreOutlined,
} from '@ant-design/icons'
import ChatPanel from '../components/ChatPanel'
import type { Message } from '../components/ChatPanel'
import { MarkdownContent, mdStyles } from '../components/MarkdownContent'
import WorldviewSectionsEditor from '../components/novel/WorldviewSectionsEditor'
import ForeshadowPanel from '../components/novel/ForeshadowPanel'
import ConsistencyPanel from '../components/novel/ConsistencyPanel'
import { useAppStore } from '../stores/appStore'
import { countTextChars, extractSettingText } from '../utils/text'
import * as App from '../../src/wailsjsCompat'

type EditorMode = 'edit' | 'split' | 'preview' | 'sections'

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
    } catch (err: unknown) {
      message.error('保存失败: ' + (err instanceof Error ? err.message : String(err)))
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
      const result = await App.ChatWorldview(userMsg, content)
      const reply = result?.reply || ''
      // AI 返回更新后的设定文本，直接回填编辑器（不解析、不拆分）
      if (typeof result?.worldview === 'string' && result.worldview) {
        setContent(result.worldview)
      }
      return reply
    } catch (err: unknown) {
      const msg = typeof err === 'string' ? err : (err instanceof Error ? (err.message || '对话失败') : '对话失败')
      throw new Error(msg)
    }
  }

  /** 手动将某条 AI 回复应用到设定编辑器（覆盖） */
  const handleApplyAiOutput = useCallback((msg: Message) => {
    const text = extractSettingText(msg.content)
    if (!text) {
      message.warning('这条消息没有可应用的内容')
      return
    }
    Modal.confirm({
      title: '将 AI 输出应用到设定',
      content: text !== msg.content.trim()
        ? '已提取回复中的 Markdown 代码块，将覆盖当前设定编辑器内容；未保存的修改会丢失。'
        : '将用这条 AI 回复的完整内容覆盖当前设定编辑器；未保存的修改会丢失。',
      okText: '覆盖设定',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => {
        setContent(text)
        message.success('已应用到设定编辑器，点击「保存」生效')
      },
    })
  }, [])

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

  const sectionsPane = (
    <WorldviewSectionsEditor disabled={needsProject} />
  )

  const editorBody = needsProject ? (
    <Empty style={{ margin: 'auto' }} image={Empty.PRESENTED_IMAGE_SIMPLE}
      description="请先在「书架」打开或创建一部小说项目" />
  ) : loading ? (
    <div style={{ margin: 'auto' }}><Spin size="large" /></div>
  ) : mode === 'edit' ? (
    editorPane
  ) : mode === 'preview' ? (
    previewPane
  ) : mode === 'sections' ? (
    sectionsPane
  ) : (
    <>
      {editorPane}
      {previewPane}
    </>
  )

  return (
    <div style={{
      display: 'flex', flexDirection: 'column', width: '100%', height: '100%',
      padding: '16px', maxWidth: 1240, margin: '0 auto', gap: 14, minHeight: 0, minWidth: 0,
    }}>
      <style>{mdStyles}</style>

      {/* 上行：设定编辑器（Markdown + 维度化 + 渲染）+ 设定 Agent */}
      <div style={{ display: 'flex', flexDirection: 'row', flex: 1, minHeight: 0, gap: 14 }}>
        {/* 左栏：设定编辑器（Markdown + 渲染） */}
        <div className="novel-panel" style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minHeight: 0 }}>
          <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 6 }}>
            <span className="novel-panel-title"><FileTextOutlined />设定编辑器</span>
            <div style={{ flex: 1 }} />
            <span className="novel-setting-meta">{wordCount.toLocaleString()} 字</span>
            <Space size={8}>
              <Button size="small" icon={<ImportOutlined />} onClick={handleImport}
                style={{ borderColor: 'var(--color-primary)', color: 'var(--color-primary)' }}>
                导入
              </Button>
              <Button size="small" icon={<ExportOutlined />} onClick={handleExport}
                style={{ borderColor: 'var(--color-warning)', color: 'var(--color-warning)' }}>
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
                { label: '维度化', value: 'sections', icon: <AppstoreOutlined /> },
              ]}
            />
            <div style={{ flex: 1 }} />
            <Tag style={{ marginInlineEnd: 0 }} color={dirty ? 'warning' : 'success'}>
              {dirty ? '有未保存修改' : lastSavedAt ? `已保存 ${lastSavedAt}` : '无修改'}
            </Tag>
          </div>

          <div
            className="novel-setting-body"
            style={mode === 'split' && !needsProject ? { display: 'flex', flexDirection: 'row', gap: 12 } : { display: 'flex' }}
          >
            {editorBody}
          </div>
        </div>

        {/* 右栏：设定 Agent 常驻对话 */}
        <div className="novel-panel" style={{ width: 380, flexShrink: 0, minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          <ChatPanel
            title="设定 Agent"
            messages={messages}
            onSend={handleChatSend}
            onMessagesChange={setMessages}
            onApply={handleApplyAiOutput}
            placeholder="描述你想要的设定修改，AI 帮你调整…"
            fillHeight
          />
        </div>
      </div>

      {/* 下行：伏笔登记表 + 一致性检查 */}
      {!needsProject && (
        <div style={{ display: 'flex', flexDirection: 'row', gap: 14, height: 300, flexShrink: 0, minHeight: 0 }}>
          <ForeshadowPanel />
          <ConsistencyPanel />
        </div>
      )}
    </div>
  )
}

export default NovelSettingPage
