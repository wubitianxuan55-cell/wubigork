import React, { useState, useEffect, useRef } from 'react'
import { Button, Input, Space, message } from 'antd'
import {
  ImportOutlined, ExportOutlined, SaveOutlined,
} from '@ant-design/icons'
import ChatPanel from '../components/ChatPanel'
import type { Message } from '../components/ChatPanel'
import { C } from '../utils/theme'
import * as App from '../../wailsjs/go/app/App'

const NovelSettingPage: React.FC = () => {
  const [content, setContent] = useState('')
  const [messages, setMessages] = useState<Message[]>([])
  const [saving, setSaving] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // 挂载时加载已有设定
  useEffect(() => {
    ;(async () => {
      try {
        const text = await App.GetWorldview()
        if (text) setContent(text)
      } catch (_) { /* 无已有内容则保持空白 */ }
    })()
  }, [])

  const handleSave = async () => {
    setSaving(true)
    try {
      await App.SaveWorldview(content)
      message.success('已保存')
    } catch (err: any) {
      message.error('保存失败: ' + (err?.message || err))
    } finally { setSaving(false) }
  }

  const handleImport = () => fileInputRef.current?.click()

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => {
      const text = reader.result as string
      setContent(text)
      message.success(`已导入「${file.name}」`)
    }
    reader.readAsText(file)
    // 清空 input 以允许重复导入同一文件
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
      const reply = (result as any)?.reply || ''
      const worldview = (result as any)?.worldview
      // AI 返回更新后的设定文本，自动填入编辑器
      if (worldview && typeof worldview === 'string') {
        setContent(worldview)
      }
      return reply
    } catch (err: any) {
      const msg = typeof err === 'string' ? err : (err?.message || err?.toString?.() || '对话失败')
      throw new Error(msg)
    }
  }

  return (
    <div style={{
      display: 'flex', flexDirection: 'column', height: '100%',
      padding: '16px', maxWidth: 900, margin: '0 auto', gap: 12,
    }}>
      {/* 工具栏 */}
      <div style={{
        display: 'flex', justifyContent: 'flex-end', alignItems: 'center',
      }}>
        <Space size={8}>
          <Button icon={<ImportOutlined />} onClick={handleImport}
            style={{ borderColor: '#60a5fa', color: '#60a5fa' }}>
            导入
          </Button>
          <Button icon={<ExportOutlined />} onClick={handleExport}
            style={{ borderColor: '#f59e0b', color: '#f59e0b' }}>
            导出
          </Button>
          <Button icon={<SaveOutlined />} onClick={handleSave}
            loading={saving}
            style={{ borderColor: '#4ade80', color: '#4ade80' }}>
            保存
          </Button>
        </Space>
      </div>

      {/* 隐藏的文件导入 input */}
      <input
        ref={fileInputRef}
        type="file"
        accept=".txt,.md,.json"
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />

      {/* 编辑器 */}
      <Input.TextArea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="在此撰写或粘贴小说设定（世界观、剧情框架、人物关系、规则体系等）…"
        style={{
          flex: 1, minHeight: 300, resize: 'none',
          background: C('color-bg-container'),
          borderColor: C('color-border'),
          color: C('color-text'),
          fontFamily: 'var(--font-mono, "SF Mono", "Fira Code", monospace)',
          fontSize: 14,
          lineHeight: 1.8,
        }}
      />

      {/* AI 对话 */}
      <ChatPanel
        title="设定 Agent"
        messages={messages}
        onSend={handleChatSend}
        onMessagesChange={setMessages}
        placeholder="描述你想要的设定修改，AI 帮你调整…"
        defaultCollapsed
      />
    </div>
  )
}

export default NovelSettingPage
