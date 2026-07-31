import React, { useState, useEffect, useRef, useCallback } from 'react'
import { Button, Card, Space, Typography, Input, Tag, Spin, Empty, message, Tabs, Tooltip, Collapse, Divider } from 'antd'
import {
  SendOutlined, RobotOutlined, ToolOutlined, BookOutlined, ClearOutlined,
  CalculatorOutlined, FileTextOutlined, BarChartOutlined, AuditOutlined, MoneyCollectOutlined, AppstoreOutlined, LoadingOutlined
} from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import '../whisper-theme.css'

const { Title, Text, Paragraph } = Typography
const { TextArea } = Input

interface GMsg {
  id: string
  role: 'user' | 'assistant' | 'tool' | 'system'
  content: string
  reasoning?: string
  toolName?: string
  time: string
}

interface GTool {
  name: string
  description: string
  schema: string
}

interface GSkill {
  name: string
  description: string
}

const gb = 'var(--whisper-glass-bg)'
const cst: React.CSSProperties = { background: gb, backdropFilter: 'blur(20px)', height: '100%', overflow: 'auto' }

// 工具分类（按前缀归类，用于 UI 面板分组）
function toolCat(name: string): string {
  if (name.startsWith('calc_') || name === 'material_query') return '计算'
  if (name.includes('csv') || name.includes('xlsx') || name.includes('docx') || name.includes('pdf') || name.includes('doc_')) return '文档'
  if (name.includes('chart') || name.includes('gantt')) return '图表'
  if (name.startsWith('spec')) return '规范'
  if (name.includes('cost')) return '造价'
  if (name.includes('survey') || name.includes('bid') || name.includes('project') || name.includes('imple')) return '工程业务'
  return '通用'
}

const catColor: Record<string, string> = {
  计算: 'blue', 文档: 'green', 图表: 'purple', 规范: 'orange', 造价: 'cyan', 工程业务: 'magenta', 通用: 'default',
}

function GaeaPage() {
  const [msgs, setMsgs] = useState<GMsg[]>([])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [model, setModel] = useState('')
  const [tools, setTools] = useState<GTool[]>([])
  const [skills, setSkills] = useState<GSkill[]>([])
  const [toolTab, setToolTab] = useState('tools')
  const bottomRef = useRef<HTMLDivElement>(null)

  const cur = useRef<{ role: 'assistant'; content: string; reasoning: string; id: string } | null>(null)

  // 事件监听：gaea 事件流
  useEffect(() => {
    const off = EventsOn('gaea-event', (e: any) => {
      const k = e?.kind
      if (k === 'turn_started') {
        setBusy(true)
        cur.current = { role: 'assistant', content: '', reasoning: '', id: 'm' + Date.now() }
      } else if (k === 'reasoning') {
        if (cur.current) cur.current.reasoning += e.text || ''
      } else if (k === 'text') {
        if (!cur.current) cur.current = { role: 'assistant', content: '', reasoning: '', id: 'm' + Date.now() }
        cur.current.content += e.text || ''
        setMsgs(prev => {
          const list = prev.filter(m => m.id !== cur.current!.id)
          return [...list, { ...cur.current!, time: new Date().toLocaleTimeString() }]
        })
      } else if (k === 'message') {
        if (cur.current) {
          cur.current.content = e.text || cur.current.content
          cur.current.reasoning = e.reasoning || cur.current.reasoning
        }
      } else if (k === 'tool_dispatch') {
        const t = e.tool || {}
        setMsgs(prev => [...prev, { id: 't' + Date.now() + Math.random(), role: 'tool', toolName: t.name, content: t.args || '', time: new Date().toLocaleTimeString() }])
      } else if (k === 'tool_result') {
        const t = e.tool || {}
        setMsgs(prev => {
          const list = prev.filter(m => m.id !== 'last-tool')
          return [...list, { id: 'last-tool', role: 'tool', toolName: t.name, content: String(t.output || '').slice(0, 400), time: new Date().toLocaleTimeString() }]
        })
      } else if (k === 'notice') {
        setMsgs(prev => [...prev, { id: 'n' + Date.now(), role: 'system', content: e.text || '', time: new Date().toLocaleTimeString() }])
      } else if (k === 'turn_done') {
        if (cur.current) {
          setMsgs(prev => {
            const list = prev.filter(m => m.id !== cur.current!.id)
            return [...list, { ...cur.current!, time: new Date().toLocaleTimeString() }]
          })
        }
        cur.current = null
        setBusy(false)
        if (e?.error) message.error(String(e.error))
      } else if (k === 'error') {
        setBusy(false)
        setMsgs(prev => [...prev, { id: 'e' + Date.now(), role: 'system', content: '❌ ' + (e.text || e.error || '未知错误'), time: new Date().toLocaleTimeString() }])
      }
    })
    return () => { if (typeof off === 'function') off() }
  }, [])

  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [msgs])

  // 初始化：拉工具/技能/模型名
  const init = useCallback(async () => {
    try {
      setModel((await App.GaeaModel()) || '')
      const ts = (await App.GaeaTools()) || []
      setTools(ts as GTool[])
      const sk = (await App.GaeaSkills()) || []
      setSkills(sk as GSkill[])
    } catch (e) { /* 引擎未初始化时静默 */ }
  }, [])
  useEffect(() => { init() }, [init])

  const send = async () => {
    const text = input.trim()
    if (!text || busy) return
    setMsgs(prev => [...prev, { id: 'u' + Date.now(), role: 'user', content: text, time: new Date().toLocaleTimeString() }])
    setInput('')
    try {
      await App.GaeaSend(text)
    } catch (e: any) {
      message.error(String(e))
      setBusy(false)
    }
  }

  const callTool = async (name: string, argsStr: string) => {
    try {
      const r = await App.GaeaCallTool(name, argsStr || '{}')
      setMsgs(prev => [...prev, { id: 'tc' + Date.now(), role: 'tool', toolName: name, content: String(r).slice(0, 600), time: new Date().toLocaleTimeString() }])
    } catch (e: any) {
      message.error(String(e))
    }
  }

  const newSession = async () => {
    try { await App.GaeaNewSession(); setMsgs([]) } catch (e: any) { message.error(String(e)) }
  }

  // 工具面板：分类分组
  const groups: Record<string, GTool[]> = {}
  for (const t of tools) {
    const c = toolCat(t.name)
    ;(groups[c] = groups[c] || []).push(t)
  }
  const groupNames = Object.keys(groups).sort()

  const items = [
    { key: 'chat', label: <span><RobotOutlined /> AI 办公对话</span>, children: chatPanel() },
    { key: 'tools', label: <span><ToolOutlined /> 工程工具箱</span>, children: toolsPanel() },
    { key: 'skills', label: <span><BookOutlined /> 技能模块</span>, children: skillsPanel() },
  ]

  function chatPanel() {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0, padding: 16 }}>
        <div style={{ flex: 1, overflow: 'auto', minHeight: 0, marginBottom: 12 }}>
          {msgs.length === 0 ? (
            <Empty description="土壤修复 / 岩土工程办公 AI 助手已就绪。描述需求即可调用 47 个工程工具（计算、文档、图表、规范、造价、投标）。" style={{ marginTop: 80 }} />
          ) : (
            msgs.map(m => (
              <div key={m.id} style={{ display: 'flex', justifyContent: m.role === 'user' ? 'flex-end' : 'flex-start', marginBottom: 10 }}>
                <div style={{
                  maxWidth: '86%', padding: '8px 12px', borderRadius: 10, fontSize: 13, lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                  background: m.role === 'user' ? 'var(--md-sys-color-primary-container)' : m.role === 'system' ? 'rgba(255,193,7,0.12)' : gb,
                  border: m.role === 'tool' ? '1px dashed var(--md-sys-color-outline-variant)' : '1px solid transparent',
                }}>
                  {m.role === 'tool' && m.toolName && (
                    <div style={{ marginBottom: 4 }}><Tag color="purple" style={{ fontSize: 10 }}>🔧 {m.toolName}</Tag></div>
                  )}
                  {m.role === 'assistant' && m.reasoning && (
                    <details style={{ marginBottom: 6, fontSize: 11, color: '#888' }}><summary>🧠 思考过程</summary><div style={{ marginTop: 4 }}>{m.reasoning}</div></details>
                  )}
                  {m.content || <Text type="secondary" style={{ fontSize: 12 }}>（空）</Text>}
                  {m.role === 'tool' && <div style={{ marginTop: 4 }}><Text type="secondary" style={{ fontSize: 10 }}>{m.time}</Text></div>}
                </div>
              </div>
            ))
          )}
          {busy && <div style={{ padding: '4px 0' }}><Spin indicator={<LoadingOutlined spin />} size="small" /> <Text type="secondary" style={{ fontSize: 12 }}>AI 思考中…</Text></div>}
          <div ref={bottomRef} />
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <TextArea
            value={input}
            onChange={e => setInput(e.target.value)}
            onPressEnter={e => { if (!e.shiftKey) { e.preventDefault(); send() } }}
            placeholder="描述任务：如「查询 GB 36600 中砷的超标限值」「计算修复成本测算」「生成调查布点图」…"
            autoSize={{ minRows: 2, maxRows: 6 }}
            style={{ flex: 1 }}
          />
          <Space direction="vertical" size={4}>
            <Button type="primary" icon={<SendOutlined />} onClick={send} loading={busy}>发送</Button>
            <Button icon={<ClearOutlined />} onClick={newSession} disabled={busy}>新会话</Button>
          </Space>
        </div>
      </div>
    )
  }

  function toolsPanel() {
    return (
      <div style={{ padding: 16, height: '100%', overflow: 'auto' }}>
        <Title level={5} style={{ marginTop: 0 }}>🧰 工程工具（{tools.length} 个，可直接调用）</Title>
        <Space wrap style={{ marginBottom: 12 }}>
          {groupNames.map(g => <Tag key={g} color={catColor[g] || 'default'}>{g} · {groups[g].length}</Tag>)}
        </Space>
        {groupNames.map(g => (
          <Collapse key={g} ghost style={{ marginBottom: 8 }} items={[{
            key: g,
            label: <Space><Tag color={catColor[g] || 'default'} icon={<AppstoreOutlined />}>{g}</Tag><Text strong>{groups[g].length} 个</Text></Space>,
            children: (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 8 }}>
                {groups[g].map(t => (
                  <Card key={t.name} size="small" style={{ ...cst, marginBottom: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                      <Text code strong style={{ fontSize: 12 }}>{t.name}</Text>
                      <Button size="small" icon={<RobotOutlined />} onClick={() => { navigator.clipboard.writeText(t.name); message.success('工具名已复制：' + t.name) }}>复制</Button>
                    </div>
                    <Text type="secondary" style={{ fontSize: 11, display: 'block', lineHeight: 1.6, maxHeight: 48, overflow: 'hidden' }}>{t.description}</Text>
                    <Divider style={{ margin: '8px 0' }} />
                    <Button size="small" type="primary" ghost icon={<ToolOutlined />} onClick={() => callTool(t.name, '{}')} style={{ fontSize: 11 }}>空参数调用</Button>
                  </Card>
                ))}
              </div>
            ),
          }]} />
        ))}
      </div>
    )
  }

  function skillsPanel() {
    const emojis: Record<string, string> = { 'site-survey': '🏞️', 'risk-assessment': '⚠️', 'remed-design': '🏗️', 'bid-package': '📋', 'data-report': '📊', 'skill-creator': '🛠️' }
    return (
      <div style={{ padding: 16, height: '100%', overflow: 'auto' }}>
        <Title level={5} style={{ marginTop: 0 }}>📚 工程技能模块（{skills.length} 个）</Title>
        {skills.length === 0 ? <Empty description="引擎初始化后展示技能列表" /> : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: 12 }}>
            {skills.map(s => (
              <Card key={s.name} size="small" style={cst}>
                <Space style={{ marginBottom: 8 }}>
                  <span style={{ fontSize: 20 }}>{emojis[s.name] || '📖'}</span>
                  <Text strong>{s.name}</Text>
                </Space>
                <Paragraph type="secondary" style={{ fontSize: 12, marginBottom: 8 }}>{s.description}</Paragraph>
                <Button size="small" icon={<RobotOutlined />} onClick={() => sendSkill(s)}>在对话中使用</Button>
              </Card>
            ))}
          </div>
        )}
      </div>
    )
  }

  function sendSkill(s: GSkill) {
    setInput('')
    setMsgs(prev => [...prev, { id: 'u' + Date.now(), role: 'user', content: `使用技能 ${s.name}：${s.description}`, time: new Date().toLocaleTimeString() }])
    App.GaeaSend(`请加载技能 ${s.name} 并按其流程处理我的任务。技能说明：${s.description}`)
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '10px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid var(--md-sys-color-outline-variant)' }}>
        <Space>
          <Title level={4} style={{ margin: 0 }}>🏗️ 工程办公</Title>
          <Tag color="blue" style={{ fontSize: 10 }}>{model || '模型待定'}</Tag>
          <Tag color="green" style={{ fontSize: 10 }}>gaeaW 移植</Tag>
        </Space>
      </div>
      <div style={{ flex: 1, minHeight: 0 }}>
        <Tabs activeKey={toolTab} onChange={setToolTab} items={items} destroyInactiveTabPane={false} style={{ height: '100%' }} tabBarStyle={{ padding: '0 16px', marginBottom: 0 }} />
      </div>
    </div>
  )
}

export default GaeaPage
