import React, { useState, useCallback } from 'react'
import { Input, Button, Card, Space, Typography, Spin, Tag, Divider } from 'antd'
import { FolderOutlined, DesktopOutlined, SendOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined } from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import '../whisper-theme.css'

const { Title, Text, Paragraph } = Typography

interface ExecResult { success: boolean; action: string; path?: string; content?: string; summary: string; error?: string }
interface LogItem { id: number; type: 'command'|'result'|'error'; text: string; time: string; result?: ExecResult }

let logId = 0

const OfficePage: React.FC = () => {
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [logs, setLogs] = useState<LogItem[]>([])
  const [currentPath, setCurrentPath] = useState('C:\\')
  const quickActions = [
    { label: 'My Documents', action: 'list_folder', path: 'C:\\Users' },
    { label: 'C Drive', action: 'list_folder', path: 'C:\\' },
    { label: 'AI Novels', action: 'list_folder', path: 'C:\\AI\\xiaoshuo' },
    { label: 'wubigrok', action: 'list_folder', path: 'C:\\AI\\wubigrok' },
  ]

  const addLog = useCallback((type: LogItem['type'], text: string, result?: ExecResult) => {
    setLogs(prev => [...prev.slice(-200), { id: ++logId, type, text, time: new Date().toLocaleTimeString(), result }])
  }, [])

  const handleCommand = useCallback(async () => {
    const cmd = input.trim(); if (!cmd) return
    setInput(''); addLog('command', cmd)
    const isTask = await App.OfficeIsTask(cmd)
    if (isTask) {
      setLoading(true)
      try { const r = await App.OfficeRunTask(cmd) as any; addLog('result', r?.reply||'Done') } catch(e:any){ addLog('error', String(e)) }
      setLoading(false); return
    }
    if (cmd.match(/^[A-Z]:[\\/]/i)) {
      setLoading(true)
      try { const r = await App.OfficeListFolder(cmd); if(r.success){ setCurrentPath(cmd); addLog('result',r.summary,r) } else addLog('error',r.error||'Failed',r) } catch(e:any){ addLog('error',String(e)) }
      setLoading(false); return
    }
    setLoading(true)
    try { const r = await App.OfficeRunTask(cmd) as any; addLog('result', r?.reply||'Done') } catch(e:any){ addLog('error',String(e)) }
    setLoading(false)
  }, [input, addLog])

  const handleQuickAction = useCallback(async (action: string, path: string) => {
    addLog('command', `${action}: ${path}`); setLoading(true)
    try {
      let r: ExecResult
      if (action === 'list_folder') { r = await App.OfficeListFolder(path); if(r.success) setCurrentPath(path) }
      else r = await App.OfficeExecute(action, path, '', '', '', '')
      addLog('result', r.summary, r)
    } catch(e:any){ addLog('error', String(e)) }
    setLoading(false)
  }, [addLog])

  return (
    <div style={{display:'flex',height:'100%',gap:12,padding:16,background:'var(--whisper-glass-bg)'}}>
      <div style={{width:180,flexShrink:0}}>
        <Card size="small" title={<Text strong>📂 Quick</Text>} style={{background:'var(--whisper-glass-bg)',backdropFilter:'blur(20px)'}}>
          <Space direction="vertical" style={{width:'100%'}}>
            {quickActions.map((qa,i)=><Button key={i} block size="small" type="text" icon={<FolderOutlined/>} onClick={()=>handleQuickAction(qa.action,qa.path)} style={{textAlign:'left',justifyContent:'flex-start'}}>{qa.label}</Button>)}
            <Divider style={{margin:'4px 0'}}/>
            <Button block size="small" type="text" icon={<ReloadOutlined/>} onClick={()=>setLogs([])}>Clear</Button>
          </Space>
        </Card>
      </div>
      <div style={{flex:1,display:'flex',flexDirection:'column',minWidth:0}}>
        <div style={{display:'flex',alignItems:'center',gap:8,marginBottom:8,padding:'6px 12px',borderRadius:6,background:'rgba(255,255,255,0.5)',backdropFilter:'blur(10px)'}}>
          <FolderOutlined/><Text style={{fontFamily:'monospace',fontSize:13,flex:1}}>{currentPath}</Text>
          <Button size="small" icon={<ReloadOutlined/>} onClick={()=>handleQuickAction('list_folder',currentPath)}/>
        </div>
        <div style={{flex:1,overflow:'auto',background:'rgba(255,255,255,0.3)',borderRadius:8,padding:12}}>
          {logs.length===0 && <div style={{textAlign:'center',padding:40,opacity:.5}}><DesktopOutlined style={{fontSize:48,marginBottom:16}}/><Title level={5}>Desktop Office Assistant</Title><Paragraph type="secondary">Enter commands or use quick actions</Paragraph></div>}
          {logs.map(item => {
            if (item.type==='command') return <div key={item.id} style={{marginBottom:8}}><Tag color="blue" style={{fontFamily:'monospace'}}><DesktopOutlined/> {item.text}</Tag><Text type="secondary" style={{fontSize:11}}>{item.time}</Text></div>
            if (item.type==='error') return <Card key={item.id} size="small" style={{marginBottom:8,borderColor:'#ff4d4f'}}><Text type="danger"><CloseCircleOutlined/> {item.text}</Text></Card>
            return <Card key={item.id} size="small" style={{marginBottom:8}}><div style={{display:'flex',alignItems:'center',gap:8,marginBottom:4}}><CheckCircleOutlined style={{color:'#52c41a'}}/><Text>{item.result?.summary||item.text}</Text><Text type="secondary" style={{fontSize:11}}>{item.time}</Text></div>{item.result?.content&&<pre style={{background:'rgba(0,0,0,0.04)',padding:8,borderRadius:4,fontSize:12,maxHeight:300,overflow:'auto',whiteSpace:'pre-wrap'}}>{item.result.content}</pre>}</Card>
          })}
          {loading && <Spin indicator={<LoadingOutlined/>}/>}
        </div>
        <div style={{marginTop:8,display:'flex',gap:8,padding:8,borderRadius:8,background:'rgba(255,255,255,0.5)',backdropFilter:'blur(10px)'}}>
          <Input value={input} onChange={e=>setInput(e.target.value)} onKeyDown={e=>{if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();handleCommand()}}} placeholder="Path, command, or task description..." disabled={loading} style={{fontFamily:'monospace'}} prefix={<DesktopOutlined/>} suffix={<Button type="primary" size="small" icon={<SendOutlined/>} onClick={handleCommand} loading={loading}/>}/>
        </div>
      </div>
    </div>
  )
}
export default OfficePage
