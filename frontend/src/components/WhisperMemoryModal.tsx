// WhisperMemoryModal.tsx — 轻语记忆管理弹窗 v2
import React, { useState, useMemo, useCallback } from 'react'
import { Input, Button, Modal, Tag, Select, Tooltip, message, Popconfirm } from 'antd'
import { SearchOutlined, DeleteOutlined, EditOutlined, CloseOutlined, CheckOutlined, StarFilled } from '@ant-design/icons'
import { App } from '../../wailsjs/go/app/App'

interface EmotionalCtx { valence?: number; intensity?: number; trust?: number; relStage?: string }
interface MemoryFact {
  id: string; domain: string; subcategory?: string; subject: string; summary: string
  weight: number; confidence: number; tier?: string; createdAt: string; updatedAt?: string
  triggers?: string[]; sensitivity?: string; privacyLevel?: string; emotionalContext?: EmotionalCtx
}
interface Props { facts: MemoryFact[]; personalityID: string; onFactsChange?: (facts: MemoryFact[]) => void }

const DOMAIN_LABELS: Record<string, string> = {
  personal: '👤 Personal', preference: '⭐ Preference', relationship: '💕 Relationship',
  shared_bond: '🤝 Bond', health: '💊 Health', work: '💼 Work',
  user_behavior: '🎯 Behavior', user_state: '💭 State', companion_reply: '💬 Reply',
  SOCIAL: '💕 Social', DAILY_LIFE: '🏠 Daily', INNER_WORLD: '🧘 Inner', PURSUITS: '🎯 Pursuits', TEMPORAL: '⏰ Time', IDENTITY: '🪪 Identity',
}
const VALENCE_COLOR = (v: number) => v > 0.2 ? '#52c41a' : v < -0.2 ? '#ff4d4f' : '#8c8c8c'
const VALENCE_LABEL = (v: number) => v > 0.2 ? '😊' : v < -0.2 ? '😔' : '😐'

export default function WhisperMemoryModal({ facts, personalityID, onFactsChange }: Props) {
  const [search, setSearch] = useState('')
  const [domainFilter, setDomainFilter] = useState<string>('')
  const [tierFilter, setTierFilter] = useState<string>('')
  const [sortBy, setSortBy] = useState<'weight' | 'date'>('weight')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editText, setEditText] = useState('')
  const [detailFact, setDetailFact] = useState<MemoryFact | null>(null)

  const domains = useMemo(() => { const set = new Set(facts.map(f => f.domain)); return Array.from(set).sort() }, [facts])

  const filtered = useMemo(() => {
    let list = [...facts]
    if (search.trim()) {
      const q = search.toLowerCase()
      list = list.filter(f => f.subject.toLowerCase().includes(q) || f.summary.toLowerCase().includes(q) || (f.triggers||[]).some(t=>t.toLowerCase().includes(q)) || (f.subcategory||'').toLowerCase().includes(q))
    }
    if (domainFilter) list = list.filter(f => f.domain === domainFilter)
    if (tierFilter) list = list.filter(f => f.tier === tierFilter)
    if (sortBy === 'weight') list.sort((a,b) => b.weight - a.weight)
    else list.sort((a,b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    return list
  }, [facts, search, domainFilter, tierFilter, sortBy])

  const coreCount = facts.filter(f => f.tier === 'core').length

  const handleDelete = useCallback(async (id: string) => {
    try { await App.WhisperDeleteFact(personalityID, id); message.success('Deleted'); if(onFactsChange) onFactsChange(facts.filter(f=>f.id!==id)) } catch(e:any){ message.error('Failed: '+e) }
  }, [personalityID, facts, onFactsChange])

  const startEdit = useCallback((f: MemoryFact) => { setEditingId(f.id); setEditText(f.summary) }, [])
  const confirmEdit = useCallback(async () => {
    if (!editingId || !editText.trim()) return
    try { await App.WhisperUpdateFact(personalityID, editingId, { summary: editText.trim() } as any); message.success('Updated'); if(onFactsChange) onFactsChange(facts.map(f=>f.id===editingId?{...f,summary:editText.trim()}:f)) } catch(e:any){ message.error('Failed: '+e) }
    setEditingId(null)
  }, [editingId, editText, personalityID, facts, onFactsChange])

  return (
    <div style={{display:'flex',flexDirection:'column',height:'100%',gap:10}}>
      <div style={{display:'flex',alignItems:'center',gap:8,flexWrap:'wrap'}}>
        <span style={{fontSize:15,fontWeight:700,color:'var(--whisper-ink)'}}>🧠 Memory</span>
        <Tag color="blue">{facts.length}</Tag>
        {coreCount>0 && <Tag color="gold" icon={<StarFilled />}>{coreCount} core</Tag>}
        <span style={{flex:1}}/>
        <Select size="small" value={sortBy} onChange={setSortBy} style={{width:90}} options={[{value:'weight',label:'Weight'},{value:'date',label:'Date'}]}/>
      </div>
      <div style={{display:'flex',gap:6}}>
        <Input prefix={<SearchOutlined />} placeholder="Search…" size="small" value={search} onChange={e=>setSearch(e.target.value)} style={{flex:1,borderRadius:8}} allowClear/>
        <Select size="small" placeholder="Domain" value={domainFilter||undefined} onChange={v=>setDomainFilter(v||'')} allowClear style={{width:120}} options={domains.map(d=>({value:d,label:(DOMAIN_LABELS[d]||d).replace(/[^\u4e00-\u9fa5]/g,'')}))}/>
        <Select size="small" placeholder="Tier" value={tierFilter||undefined} onChange={v=>setTierFilter(v||'')} allowClear style={{width:80}} options={[{value:'core',label:'⭐ Core'},{value:'memory',label:'Memory'}]}/>
      </div>
      <div style={{flex:1,overflow:'auto',minHeight:0}}>
        {filtered.length===0 ? (
          <div style={{textAlign:'center',padding:40,color:'var(--whisper-ink-muted)',fontSize:13}}>
            {facts.length===0 ? <><span style={{fontSize:32}}>🧠</span><br/><br/>No memories yet<br/>Chat more and I'll remember 💫</> : 'No matches'}
          </div>
        ) : filtered.map(f => (
          <div key={f.id} style={{padding:'10px 12px',marginBottom:6,borderRadius:10,background:'var(--whisper-surface)',border:f.tier==='core'?'1.5px solid var(--whisper-accent)':'1px solid var(--whisper-glass-border)'}}>
            <div style={{display:'flex',alignItems:'center',gap:6}}>
              {f.tier==='core' && <StarFilled style={{color:'#faad14',fontSize:12}}/>}
              <span style={{fontSize:13,fontWeight:600,color:'var(--whisper-ink)',flex:1,cursor:'pointer'}} onClick={()=>setDetailFact(f)}>{f.subject}</span>
              {f.subcategory && <Tag style={{fontSize:9,margin:0}}>{f.subcategory}</Tag>}
              <Tag style={{fontSize:9,margin:0,opacity:.7}}>{(DOMAIN_LABELS[f.domain]||f.domain).replace(/[^\u4e00-\u9fa5]/g,'')}</Tag>
              {editingId===f.id ? (<><Button type="text" size="small" icon={<CheckOutlined />} onClick={confirmEdit} style={{color:'#52c41a',padding:0,width:22}}/><Button type="text" size="small" icon={<CloseOutlined />} onClick={()=>setEditingId(null)} style={{color:'#ff4d4f',padding:0,width:22}}/></>) : (<><Tooltip title="Edit"><Button type="text" size="small" icon={<EditOutlined />} onClick={()=>startEdit(f)} style={{color:'var(--whisper-ink-muted)',padding:0,width:22}}/></Tooltip><Popconfirm title="Delete?" onConfirm={()=>handleDelete(f.id)}><Button type="text" size="small" danger icon={<DeleteOutlined />} style={{padding:0,width:22}}/></Popconfirm></>)}
            </div>
            {editingId===f.id ? <Input.TextArea size="small" value={editText} onChange={e=>setEditText(e.target.value)} autoSize={{minRows:2,maxRows:4}} style={{marginTop:6,fontSize:12}}/> : <div style={{fontSize:12,color:'var(--whisper-ink)',marginTop:4,lineHeight:1.6,cursor:'pointer'}} onClick={()=>setDetailFact(f)}>{f.summary.slice(0,200)}{f.summary.length>200?'…':''}</div>}
            <div style={{display:'flex',alignItems:'center',gap:8,marginTop:6,flexWrap:'wrap'}}>
              <span style={{fontSize:10,color:'var(--whisper-ink-muted)'}}>W{f.weight.toFixed(1)} · C{(f.confidence*100).toFixed(0)}%</span>
              {f.emotionalContext && <span style={{fontSize:10,color:VALENCE_COLOR(f.emotionalContext.valence||0)}}>{VALENCE_LABEL(f.emotionalContext.valence||0)}{f.emotionalContext.trust!=null&&` T${f.emotionalContext.trust.toFixed(0)}`}</span>}
              <span style={{fontSize:10,color:'var(--whisper-ink-muted)',marginLeft:'auto'}}>{f.createdAt?.slice(0,16)}</span>
            </div>
            {f.triggers && f.triggers.length>0 && <div style={{marginTop:4,display:'flex',gap:3,flexWrap:'wrap'}}>{f.triggers.slice(0,6).map((t,i)=><Tag key={i} style={{fontSize:9,margin:0,opacity:.5}}>#{t}</Tag>)}</div>}
          </div>
        ))}
      </div>
      <Modal title={detailFact?.subject||'Detail'} open={!!detailFact} onCancel={()=>setDetailFact(null)} footer={null} width={520}>
        {detailFact && (
          <div style={{fontSize:13,lineHeight:1.9,color:'var(--whisper-ink)'}}>
            <p style={{fontSize:14,whiteSpace:'pre-wrap'}}>{detailFact.summary}</p>
            <div style={{color:'var(--whisper-ink-muted)',fontSize:12,marginTop:16,display:'grid',gridTemplateColumns:'1fr 1fr',gap:'4px 16px'}}>
              <span>Domain: {DOMAIN_LABELS[detailFact.domain]||detailFact.domain}</span><span>Sub: {detailFact.subcategory||'—'}</span>
              <span>Weight: {detailFact.weight.toFixed(2)}</span><span>Confidence: {(detailFact.confidence*100).toFixed(0)}%</span>
              <span>Tier: {detailFact.tier||'memory'}</span><span>Sensitivity: {detailFact.sensitivity||'normal'}</span>
              <span>Created: {detailFact.createdAt}</span><span>Updated: {detailFact.updatedAt||'—'}</span>
            </div>
            {detailFact.emotionalContext && (
              <div style={{marginTop:12,padding:10,borderRadius:8,background:'rgba(0,0,0,0.03)'}}>
                <div style={{fontSize:12,fontWeight:600,marginBottom:6}}>📊 Emotional Snapshot</div>
                <div style={{display:'grid',gridTemplateColumns:'1fr 1fr',gap:4,fontSize:12}}>
                  <span>Valence: {detailFact.emotionalContext.valence?.toFixed(2)}</span><span>Intensity: {detailFact.emotionalContext.intensity?.toFixed(2)}</span>
                  <span>Trust: {detailFact.emotionalContext.trust?.toFixed(0)}</span><span>Stage: {detailFact.emotionalContext.relStage||'—'}</span>
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}
