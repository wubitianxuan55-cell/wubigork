import React, { useState, useEffect, useRef } from 'react'
import { Typography, Button, Space, Tag, Modal, Input, Select, Popconfirm, message, Empty } from 'antd'
import { PlusOutlined, DeleteOutlined, BookOutlined, BulbOutlined } from '@ant-design/icons'

import { C } from '../utils/theme'

interface LorebookEntry {
  key: string
  content: string
  category: string
}

interface LorebookModalProps {
  open: boolean
  onClose: () => void
}

const categoryColors: Record<string, string> = {
  character: '#4ade80', location: '#60a5fa', item: '#f59e0b', concept: '#c084fc',
}
const categoryLabels: Record<string, string> = {
  character: '角色', location: '地点', item: '道具', concept: '概念',
}

const LorebookModal: React.FC<LorebookModalProps> = ({ open, onClose }) => {
  const [entries, setEntries] = useState<LorebookEntry[]>([])
  const [addKey, setAddKey] = useState('')
  const [addContent, setAddContent] = useState('')
  const [addCategory, setAddCategory] = useState('character')

  const mountedRef = useRef(true)
  useEffect(() => { mountedRef.current = true; return () => { mountedRef.current = false } }, [])

  useEffect(() => {
    if (open) loadEntries()
  }, [open])

  const loadEntries = async () => {
    try {
      // @ts-ignore
      const data = await window.go.app.App.GetLorebookEntries()
      if (!mountedRef.current) return
      if (data?.entries) setEntries(data.entries)
    } catch (_) {}
  }

  const handleAdd = async () => {
    if (!addKey.trim() || !addContent.trim()) { message.warning('请填写词条名和设定内容'); return }
    try {
      // @ts-ignore
      await window.go.app.App.SaveLorebookEntry(JSON.stringify({
        key: addKey.trim(), content: addContent.trim(), category: addCategory,
      }))
      setAddKey(''); setAddContent('')
      loadEntries()
      message.success(`已添加词条「${addKey}」`)
    } catch (err: any) { message.error(err.message || '添加失败') }
  }

  const handleDelete = async (key: string) => {
    try {
      // @ts-ignore
      await window.go.app.App.DeleteLorebookEntry(key)
      loadEntries()
    } catch (_) {}
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><BookOutlined style={{ color: '#c084fc', marginRight: 8 }} />Lorebook 词条管理</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={640}
      styles={{
        body: { maxHeight: '65vh', overflow: 'auto' },
      }}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {/* 说明 */}
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
          <BulbOutlined style={{ marginRight: 4 }} />定义关键词条（角色名、地名、道具等），AI 写作时会自动加载匹配的设定。
        </Typography.Text>

        {/* 添加区 */}
        <div style={{ background: C('color-bg-layout'), borderRadius: 8, padding: 12, border: '1px solid ' + C('color-border') }}>
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            <div style={{ display: 'flex', gap: 8 }}>
              <Input
                placeholder="词条名（如：青云宗）"
                value={addKey}
                onChange={(e) => setAddKey(e.target.value)}
                onPressEnter={handleAdd}
                style={{ flex: 1, background: C('color-bg-container'), borderColor: C('color-border'), color: C('color-text') }}
              />
              <Select value={addCategory} onChange={setAddCategory} style={{ width: 90 }}>
                {Object.entries(categoryLabels).map(([k, v]) => (
                  <Select.Option key={k} value={k}>{v}</Select.Option>
                ))}
              </Select>
            </div>
            <Input.TextArea
              placeholder="设定内容（50-300字）——会在写作时注入 AI 上下文"
              value={addContent}
              onChange={(e) => setAddContent(e.target.value)}
              rows={3}
              style={{ background: C('color-bg-container'), borderColor: C('color-border'), color: C('color-text') }}
            />
            <Button type="primary" size="small" icon={<PlusOutlined />} onClick={handleAdd}
              style={{ background: C('color-primary'), borderColor: C('color-primary') }}>
              添加词条
            </Button>
          </Space>
        </div>

        {/* 已有词条列表 */}
        {entries.length === 0 ? (
          <Empty description="暂无词条" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            {entries.map((e) => (
              <div
                key={e.key}
                style={{
                  display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
                  background: C('color-bg-layout'), borderRadius: 6,
                  padding: '8px 12px', border: '1px solid ' + C('color-border'),
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Space size={4}>
                    <Tag color={categoryColors[e.category]} style={{ fontSize: 10 }}>
                      {categoryLabels[e.category] || e.category}
                    </Tag>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
                      {e.key}
                    </Typography.Text>
                  </Space>
                  <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 4, lineHeight: 1.5 }}>
                    {e.content}
                  </div>
                </div>
                <Popconfirm title={`删除词条「${e.key}」？`} onConfirm={() => handleDelete(e.key)}>
                  <Button type="text" size="small" danger icon={<DeleteOutlined />}
                    style={{ flexShrink: 0, marginLeft: 8 }} />
                </Popconfirm>
              </div>
            ))}
          </Space>
        )}
      </Space>
    </Modal>
  )
}

export default LorebookModal
