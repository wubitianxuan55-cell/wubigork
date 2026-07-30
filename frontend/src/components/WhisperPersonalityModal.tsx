// WhisperPersonalityModal.tsx — 轻语人格选择弹窗 v2
import React, { useMemo } from 'react'
import { Modal, Card, Tag, Tooltip, Typography, message } from 'antd'
import { CheckCircleFilled, LockOutlined, ThunderboltOutlined } from '@ant-design/icons'

const { Text } = Typography

export interface PersonalityPreset {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
  hiddenPersona?: { T: number; I: number; S: number; O: number; R: number }
}
interface Props { open: boolean; personalities: PersonalityPreset[]; activePersonality: string; adultMode: boolean; onClose: () => void; onSwitch: (id: string) => void }

const DIM_LABELS: Record<string, string> = { T: 'Tender', I: 'Active', S: 'Sub', O: 'Unique', R: 'Reserved' }
const DIM_COLORS: Record<string, string> = { T: '#ff7eb3', I: '#ff6b35', S: '#52c41a', O: '#a855f7', R: '#3b82f6' }
const GROUPS = [
  { key: 'female', label: '👩 Female', ids: ['tsundere','yandere','oneesan','genki','kuudere','deredere','shitakiri','bokke','ice_queen','girl_next_door'] },
  { key: 'male', label: '🤵 Male', ids: ['ceo_dom','gentle_warmth','puppy','iceberg','schemer','loyal_knight','bad_boy','artistic','innocent_boy','boy_next_door'] },
  { key: 'ds', label: '🔥 D/s', ids: ['submissive','dominatrix','loyal_pup','tamer'] },
  { key: 'special', label: '✨ Special', ids: ['mommy','mesugaki','gap_moe_f','daddy','gap_moe_m'] },
]
const TAG_COLORS: Record<string, string> = { maternal:'magenta', nurturing:'magenta', bratty:'orange', 'provoke-submit':'volcano', 'dual-persona':'purple', paternal:'blue' }

function DimBars({ dims, mini }: { dims: PersonalityPreset['dims']; mini?: boolean }) {
  const entries = ['T','I','S','O','R'] as const
  const sz = mini ? { h:3,gap:2,fs:8 } : { h:6,gap:4,fs:10 }
  return <div style={{display:'grid',gridTemplateColumns:'repeat(5,1fr)',gap:sz.gap}}>
    {entries.map(k => <Tooltip key={k} title={`${DIM_LABELS[k]}: ${dims[k]}`}><div style={{textAlign:'center'}}><div style={{height:sz.h,borderRadius:sz.h,background:'rgba(0,0,0,0.08)',overflow:'hidden',marginBottom:1}}><div style={{height:'100%',width:`${dims[k]}%`,borderRadius:sz.h,background:DIM_COLORS[k]}}/></div>{!mini && <span style={{fontSize:sz.fs,color:'#8c8c8c'}}>{DIM_LABELS[k]}</span>}</div></Tooltip>)}
  </div>
}

export default function WhisperPersonalityModal({ open, personalities, activePersonality, adultMode, onClose, onSwitch }: Props) {
  const grouped = useMemo(() => GROUPS.map(g => ({...g,items:g.ids.map(id=>personalities.find(p=>p.id===id)).filter(Boolean) as PersonalityPreset[]})), [personalities])
  return (
    <Modal title={null} open={open} onCancel={onClose} footer={null} width={820} centered bodyStyle={{maxHeight:'70vh',overflow:'auto',padding:'16px 20px'}}>
      <div style={{fontSize:16,fontWeight:700,marginBottom:4}}>🎭 Select gaea Personality</div>
      <Text type="secondary" style={{fontSize:12}}>29 personalities, each with unique TISOR dimensions</Text>
      <div style={{marginTop:16}}>
        {grouped.map(group => (
          <div key={group.key} style={{marginBottom:20}}>
            <div style={{fontSize:13,fontWeight:600,marginBottom:8,padding:'4px 0',borderBottom:'1px solid rgba(0,0,0,0.06)',display:'flex',alignItems:'center',gap:6}}><span>{group.label}</span><Tag style={{marginLeft:4,fontSize:10}}>{group.items.length}</Tag></div>
            <div style={{display:'grid',gridTemplateColumns:'repeat(auto-fill,minmax(220px,1fr))',gap:8}}>
              {group.items.map(p => {
                const isActive = activePersonality === p.id
                const locked = Boolean(p.requiresAdult18 && !adultMode)
                const hasHidden = p.tags?.includes('dual-persona')
                return <Card key={p.id} size="small" hoverable={!locked} onClick={()=>{if(locked){message.warning('Requires adult mode');return};if(isActive)return;onSwitch(p.id)}}
                  style={{cursor:locked?'not-allowed':isActive?'default':'pointer',opacity:locked?.45:1,border:isActive?'2px solid #e85388':'1px solid rgba(0,0,0,0.08)',background:isActive?'rgba(232,83,136,0.04)':'#fff'}}
                  bodyStyle={{padding:'10px 12px'}}>
                  <div style={{display:'flex',alignItems:'center',gap:6,marginBottom:6}}>
                    <span style={{fontSize:16}}>{p.gender==='male'?'🤵':'👩'}</span>
                    <Text strong style={{fontSize:13,flex:1}}>{p.label}</Text>
                    {isActive && <CheckCircleFilled style={{color:'#e85388',fontSize:14}}/>}
                    {locked && <LockOutlined style={{color:'#8c8c8c',fontSize:12}}/>}
                    {hasHidden && <Tooltip title="Has hidden persona"><ThunderboltOutlined style={{color:'#a855f7',fontSize:12}}/></Tooltip>}
                  </div>
                  <DimBars dims={p.dims} mini/>
                  {p.tags && p.tags.length>0 && <div style={{marginTop:4,display:'flex',gap:3,flexWrap:'wrap'}}>{p.tags.map(t=><Tag key={t} color={TAG_COLORS[t]||'default'} style={{fontSize:9,margin:0}}>{t==='dual-persona'?'🎭':t==='maternal'?'🤱':t==='bratty'?'😈':t}</Tag>)}</div>}
                  {p.hiddenPersona && <div style={{marginTop:6,padding:'4px 6px',borderRadius:4,background:'rgba(168,85,247,0.06)',fontSize:9}}><span style={{color:'#a855f7'}}>🎭 Alt:</span> T{p.hiddenPersona.T} I{p.hiddenPersona.I} S{p.hiddenPersona.S} O{p.hiddenPersona.O} R{p.hiddenPersona.R}</div>}
                  {isActive && <div style={{marginTop:4,fontSize:10,color:'#e85388',fontWeight:500}}>✓ Active</div>}
                </Card>
              })}
            </div>
          </div>
        ))}
      </div>
      <div style={{marginTop:12,padding:'8px 12px',borderRadius:8,background:'rgba(0,0,0,0.02)',fontSize:10,color:'#8c8c8c',display:'flex',gap:16,flexWrap:'wrap'}}>
        <span>TISOR =</span>
        {Object.entries(DIM_LABELS).map(([k,v])=><span key={k} style={{display:'flex',alignItems:'center',gap:3}}><span style={{width:8,height:8,borderRadius:2,background:DIM_COLORS[k],display:'inline-block'}}/>{k}={v}</span>)}
      </div>
    </Modal>
  )
}
