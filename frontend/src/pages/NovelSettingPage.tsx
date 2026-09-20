// NovelSettingPage.tsx — 小说「设定」面板
// 纯 Markdown 文本编辑 + 直接渲染：编辑 / 分屏 / 渲染三种模式，
// 不做结构化维度拆分与任何反解析；导入导出、设定 Agent 对话直接操作整篇文本。
// v4.3e/f：新增「维度化」模式（6 维度卡片分卡片编辑）与伏笔登记表 / 一致性检查面板。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert, Button, Input, Modal, Segmented, Space, Spin, Tag, message,
} from 'antd'
import {
  ColumnWidthOutlined, EditOutlined, ExportOutlined, EyeOutlined,
  FileTextOutlined, ImportOutlined, SaveOutlined, AppstoreOutlined,
} from '@ant-design/icons'
import ChatPanel from '../components/ChatPanel'
import type { Message } from '../components/ChatPanel'
import { MarkdownContent, mdStyles } from '../components/MarkdownContent'
import WorldviewSectionsEditor from '../components/novel/WorldviewSectionsEditor'
import V3Empty from '../components/V3Empty'
import ForeshadowPanel from '../components/novel/ForeshadowPanel'
import ConsistencyPanel from '../components/novel/ConsistencyPanel'
import { useAppStore } from '../stores/appStore'
import { countTextChars, extractSettingText } from '../utils/text'
import { app } from '../gaea/lib/bridge'
import { inShellEnv, pickFileAsFile } from '../gaea/lib/pickFile'
import { saveExportBlob } from '../gaea/lib/saveFile'

type EditorMode = 'edit' | 'split' | 'preview' | 'sections'

const NovelSettingPage: React.FC = () => {
  const projectPath = useAppStore((s) => s.projectPath)
  const projectOpen = useAppStore((s) => s.projectOpen)

  const [content, setContent] = useState('')
  const [savedSnapshot, setSavedSnapshot] = useState('')
  const [loading, setLoading] = useState(true)
  // v4.361：读取失败可见化——此前失败被吞成空编辑器，用户随手输入再 Ctrl+S
  // 会把真实设定文件覆盖成残文。失败期间禁用保存，横幅提供重试。
  const [loadFailed, setLoadFailed] = useState(false)
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
      setLoadFailed(false)
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const text = await app.GetWorldview()
      if (token !== loadToken.current) return
      setContent(text || '')
      setSavedSnapshot(text || '')
      setLoadFailed(false)
    } catch {
      if (token !== loadToken.current) return
      setContent('')
      setSavedSnapshot('')
      setLoadFailed(true)
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
    if (loadFailed) {
      message.warning('设定读取失败，已暂停保存以防覆盖，请先重试')
      return
    }
    setSaving(true)
    try {
      await app.SaveWorldview(content)
      setSavedSnapshot(content)
      setLastSavedAt(new Date().toLocaleTimeString())
      message.success('设定已保存')
    } catch (err: unknown) {
      message.error('保存失败: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setSaving(false)
    }
  }, [content, loadFailed])

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

  /** 导入内容进编辑器（FileReader 读文本回填）——壳内 PickFiles 与浏览器 input 共用管线 */
  const readFileIntoContent = (file: File) => {
    const reader = new FileReader()
    reader.onload = () => {
      setContent((reader.result as string) || '')
      message.success(`已导入「${file.name}」`)
    }
    reader.readAsText(file)
  }

  // 审计刀B b：壳内 <input type=file> 不弹框（v4.162）→ pickFileAsFile 系统
  // 对话框（扩展名同 input accept 口径）还原 File 喂原 FileReader 管线；
  // 浏览器回退隐藏 input.click()（原生弹框，口径不变）。
  const handleImport = async () => {
    if (!inShellEnv()) { fileInputRef.current?.click(); return }
    try {
      const file = await pickFileAsFile(['md', 'txt', 'json'])
      if (file) readFileIntoContent(file)
    } catch (err: unknown) {
      message.error('导入失败: ' + (err instanceof Error ? err.message : String(err)))
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    readFileIntoContent(file)
    e.target.value = ''
  }

  // 审计刀B b：壳内 <a download> 不落盘（v4.162）→ saveExportBlob 系统另存为；
  // 浏览器回退原 <a download>（saveExportBlob 内置）。
  const handleExport = async () => {
    const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
    const saved = await saveExportBlob(blob, 'novel_setting.md')
    if (saved) message.success('已导出')
  }

  const handleChatSend = async (userMsg: string): Promise<string> => {
    try {
      const result = await app.ChatWorldview(userMsg, content)
      // ChatWorldview 返回结构化 Record（reply/worldview 均为未知字段）——
      // reply 按 string 收窄取用，worldview 非空 string 时回填编辑器
      const reply = typeof result?.reply === 'string' ? result.reply : ''
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
    <V3Empty style={{ margin: 'auto' }}
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
              <Button size="small" type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving} disabled={loadFailed}>
                保存
              </Button>
            </Space>
          </div>

          {loadFailed && (
            <Alert
              type="error" showIcon style={{ margin: '8px 12px 0' }}
              data-testid="novel-setting-load-failed"
              message="设定读取失败"
              description="为防把真实设定文件覆盖成空稿，保存已暂停。"
              action={<Button size="small" danger onClick={() => loadContent()}>重试</Button>}
            />
          )}

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
