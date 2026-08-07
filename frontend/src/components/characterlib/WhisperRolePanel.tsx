// WhisperRolePanel.tsx — 轻语角色管理面板（角色库轻语 Tab）
// 管理虚拟助手（独立人格 + 微信通道 + 剧照）与人格切换，供角色库嵌入。
import React, { useState, useEffect, useCallback } from 'react'
import { Button, Input, Switch, Tag, Typography, Popconfirm, message, Select } from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, UserOutlined, ApiOutlined,
  CloseOutlined, CheckOutlined, QrcodeOutlined, LoadingOutlined, ReloadOutlined,
  PictureOutlined, ReadOutlined,
} from '@ant-design/icons'
import * as App from '../../../wailsjs/go/app/App'
import TisorRadar from '../TisorRadar'
import { CompanionAvatar } from '../CompanionAvatar'
import { generateImage } from '../../api/image'

const { Text } = Typography

export interface WhisperAssistant {
  id: string; name: string; personalityId: string
  wxToken: string; wxBotId: string; wxUserId: string; enabled: boolean
  portraitUrl?: string
  voiceGuide?: string; gender?: string; tags?: string[]; dims?: { T: number; I: number; S: number; O: number; R: number }
}

export interface WhisperPersonality {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}

// 人格 → 主题色
const PERSONALITY_COLORS: Record<string, string> = {
  gaea: '#34d399', deredere: '#e85388', tsundere: '#f59e0b', yandere: '#ef4444',
  kuudere: '#60a5fa', oneesan: '#a855f7', genki: '#22c55e', shitakiri: '#f97316',
  ice_queen: '#94a3b8', mommy: '#ec4899', mesugaki: '#fb7185', daddy: '#3b82f6',
  ceo_dom: '#8b5cf6', gentle_warmth: '#fbbf24', puppy: '#f472b6', iceberg: '#64748b',
}

interface Props {
  activePersonality: string
  /** 切换当前轻语人格（已写入 localStorage + 广播 gaea-persona-changed） */
  onSwitchPersonality: (id: string) => void
}

function emptyForm(): WhisperAssistant {
  return { id: '', name: '', personalityId: 'gaea', wxToken: '', wxBotId: '', wxUserId: '', enabled: true, portraitUrl: '' }
}

export default function WhisperRolePanel({ activePersonality, onSwitchPersonality }: Props) {
  const [assistants, setAssistants] = useState<WhisperAssistant[]>([])
  const [personalities, setPersonalities] = useState<WhisperPersonality[]>([])
  const [wxStatuses, setWxStatuses] = useState<Record<string, boolean>>({})
  const [wxSessionExpired, setWxSessionExpired] = useState<Record<string, boolean>>({})
  const [editing, setEditing] = useState<WhisperAssistant | null>(null)
  const [detail, setDetail] = useState<WhisperAssistant | null>(null)
  const [form, setForm] = useState<WhisperAssistant>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [showPersonalityPicker, setShowPersonalityPicker] = useState(false)
  const [generatingPortrait, setGeneratingPortrait] = useState(false)
  const [portraitModel, setPortraitModel] = useState('')
  const [portraitAvailModels, setPortraitAvailModels] = useState<{ engine: string; model: string }[]>([])
  // QR 扫码
  const [qrImage, setQrImage] = useState('')
  const [qrCode, setQrCode] = useState('')
  const [qrStatus, setQrStatus] = useState('')
  const [qrPolling, setQrPolling] = useState(false)
  const [needVerify, setNeedVerify] = useState(false)
  const [verifyInput, setVerifyInput] = useState('')

  const reload = useCallback(async () => {
    try {
      const list: WhisperAssistant[] = await (App as any).WhisperAssistantList()
      setAssistants(list || [])
      const statuses: any[] = await (App as any).WhisperWeixinStatus()
      const map: Record<string, boolean> = {}
      const expMap: Record<string, boolean> = {}
      if (statuses) statuses.forEach((s: any) => {
        map[s.id] = s.wxRunning
        expMap[s.id] = !!s.wxSessionExpired
      })
      setWxStatuses(map)
      setWxSessionExpired(expMap)
    } catch (_) {}
  }, [])

  useEffect(() => {
    reload()
    App.WhisperGetPersonalities().then(setPersonalities).catch(() => {})
    ;(async () => {
      try {
        const cfg: any = await App.GetImageBackendConfig()
        const list = cfg?.availableModels || []
        setPortraitAvailModels(list)
        if (!portraitModel && cfg?.currentModel) setPortraitModel(cfg.currentModel)
      } catch (_) {}
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reload])

  const handleSave = async () => {
    if (!form.name.trim()) return message.warning('请输入助手名称')
    setSaving(true)
    try {
      const token = form.wxToken.includes('*') ? (editing?.wxToken || '') : form.wxToken
      await (App as any).WhisperAssistantSave({
        id: form.id || `ast_${Date.now()}`,
        name: form.name.trim(),
        personalityId: form.personalityId || 'gaea',
        wxToken: token || '',
        wxBotId: form.wxBotId || '',
        wxUserId: form.wxUserId || '',
        enabled: form.enabled,
      })
      message.success(editing ? '已更新' : '已创建')
      setEditing(null)
      reload()
    } catch (e: any) {
      message.error(e?.message || '保存失败')
    }
    setSaving(false)
  }

  const handleQRScan = async () => {
    try {
      setQrStatus('')
      const qr: any = await (App as any).WhisperWeixinGetQR()
      setQrImage(qr.imageUrl)
      setQrCode(qr.qrcode)
      setQrStatus('wait')
      setQrPolling(true)
      const poll = setInterval(async () => {
        try {
          const s: any = await (App as any).WhisperWeixinQRStatus(qr.qrcode)
          setQrStatus(s.status || 'wait')
          if (s.status === 'confirmed' && s.botToken) {
            clearInterval(poll); setQrPolling(false)
            setForm(f => ({ ...f, wxToken: s.botToken, wxBotId: s.botId || '' }))
            message.success('微信绑定成功')
            setQrImage('')
          } else if (s.status === 'need_verifycode') {
            clearInterval(poll); setQrPolling(false); setNeedVerify(true); setQrStatus('need_verifycode')
          } else if (s.status === 'expired' || s.status === 'verify_code_blocked') {
            clearInterval(poll); setQrPolling(false); setQrStatus(s.status)
          }
        } catch (_) {}
      }, 3000)
    } catch (_) {
      message.error('获取二维码失败')
    }
  }

  const handleVerifyCode = async () => {
    if (!qrCode || !verifyInput.trim()) return
    setQrPolling(true)
    try {
      const s: any = await (App as any).WhisperWeixinQRStatusWithCode(qrCode, verifyInput.trim())
      setQrStatus(s.status || 'wait')
      if (s.status === 'confirmed' && s.botToken) {
        setQrPolling(false); setNeedVerify(false)
        setForm(f => ({ ...f, wxToken: s.botToken, wxBotId: s.botId || '' }))
        message.success('微信绑定成功')
        setQrImage('')
        return
      }
      if (s.status === 'expired' || s.status === 'verify_code_blocked') {
        setQrPolling(false); setQrStatus(s.status); return
      }
      const poll = setInterval(async () => {
        try {
          const s2: any = await (App as any).WhisperWeixinQRStatus(qrCode)
          setQrStatus(s2.status || 'wait')
          if (s2.status === 'confirmed' && s2.botToken) {
            clearInterval(poll); setQrPolling(false); setNeedVerify(false)
            setForm(f => ({ ...f, wxToken: s2.botToken, wxBotId: s2.botId || '' }))
            message.success('微信绑定成功')
            setQrImage('')
          } else if (s2.status === 'expired' || s2.status === 'verify_code_blocked') {
            clearInterval(poll); setQrPolling(false); setQrStatus(s2.status)
          }
        } catch (_) {}
      }, 3000)
    } catch (_) {
      message.error('提交配对码失败')
      setQrPolling(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await (App as any).WhisperAssistantDelete(id)
      message.success('已删除')
      reload()
    } catch (e: any) {
      message.error(e?.message || '删除失败')
    }
  }

  const startEdit = (ast: WhisperAssistant) => {
    setForm({ ...ast })
    setEditing(ast)
  }

  const startNew = () => {
    setForm(emptyForm())
    setEditing({ id: '', name: '', personalityId: 'gaea', wxToken: '', wxBotId: '', wxUserId: '', enabled: true })
  }

  const getPersonality = (id: string) => personalities.find(p => p.id === id)

  /** 切换当前轻语人格：本地持久化 + 广播（角色库与聊天页联动） */
  const switchTo = (id: string) => {
    try { localStorage.setItem('gaea_whisper_personality', id) } catch (_) {}
    window.dispatchEvent(new CustomEvent('gaea-persona-changed', { detail: { id } }))
    onSwitchPersonality(id)
    setDetail(null)
  }

  const renderPersonalityPicker = () => (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(100px, 1fr))', gap: 6, maxHeight: 200, overflow: 'auto', padding: '4px 0' }}>
      {personalities.map(p => {
        const selected = form.personalityId === p.id
        return (
          <div key={p.id}
            onClick={() => { setForm({ ...form, personalityId: p.id }); setShowPersonalityPicker(false) }}
            style={{
              padding: '6px 8px', borderRadius: 10, cursor: 'pointer', textAlign: 'center',
              background: selected ? 'rgba(232,83,136,0.12)' : 'rgba(255,255,255,0.03)',
              border: selected ? '1.5px solid rgba(232,83,136,0.35)' : '1px solid rgba(255,255,255,0.06)',
              transition: 'all 200ms',
            }}
          >
            <TisorRadar dims={p.dims} size={40} color={selected ? '#e85388' : '#666'} showLabels={false} />
            <div style={{ fontSize: 10, marginTop: 2, color: selected ? '#e85388' : 'rgba(255,255,255,0.6)', fontWeight: selected ? 600 : 400 }}>
              {p.label}
            </div>
          </div>
        )
      })}
    </div>
  )

  const renderEditor = () => {
    const selPersonality = getPersonality(form.personalityId)
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <Button type="text" icon={<CloseOutlined />} onClick={() => setEditing(null)}
          style={{ alignSelf: 'flex-start', color: 'rgba(255,255,255,0.5)', fontSize: 12 }}>
          返回列表
        </Button>
        <div>
          <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', display: 'block', marginBottom: 4 }}>助手名称</Text>
          <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })}
            placeholder="如：小秘书、知心姐姐…"
            style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', color: '#fff', borderRadius: 10 }} />
        </div>
        <div>
          <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', display: 'block', marginBottom: 4 }}>绑定人格</Text>
          <div onClick={() => setShowPersonalityPicker(!showPersonalityPicker)}
            style={{
              padding: '10px 14px', borderRadius: 12, cursor: 'pointer',
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)',
              display: 'flex', alignItems: 'center', gap: 12,
            }}>
            {selPersonality ? (
              <>
                <TisorRadar dims={selPersonality.dims} size={48} color="#e85388" showLabels={false} />
                <div>
                  <Text strong style={{ color: '#fff', fontSize: 14 }}>{selPersonality.label}</Text>
                  <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)' }}>
                    T{selPersonality.dims.T} · I{selPersonality.dims.I} · S{selPersonality.dims.S} · O{selPersonality.dims.O} · R{selPersonality.dims.R}
                  </div>
                </div>
              </>
            ) : (
              <Text style={{ color: 'rgba(255,255,255,0.3)' }}>点击选择人格…</Text>
            )}
            <span style={{ marginLeft: 'auto', color: 'rgba(255,255,255,0.3)' }}>{showPersonalityPicker ? '▲' : '▼'}</span>
          </div>
          {showPersonalityPicker && renderPersonalityPicker()}
        </div>
        <div>
          <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', display: 'block', marginBottom: 4 }}>
            <ApiOutlined style={{ marginRight: 4 }} />微信 ClawBot Token
          </Text>
          <div style={{ display: 'flex', gap: 8 }}>
            <Input.Password
              value={form.wxToken}
              onChange={e => setForm({ ...form, wxToken: e.target.value })}
              placeholder="bot@... 或扫码绑定"
              style={{ flex: 1, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', color: '#fff', borderRadius: 10 }}
            />
            <Button
              icon={qrPolling ? <LoadingOutlined /> : <QrcodeOutlined />}
              onClick={handleQRScan}
              disabled={qrPolling}
              style={{ borderRadius: 10, border: '1px solid rgba(255,255,255,0.08)', background: 'rgba(255,255,255,0.04)', color: 'rgba(255,255,255,0.6)' }}
            >
              扫码
            </Button>
          </div>
          <div style={{ marginTop: 10 }}>
            <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', display: 'block', marginBottom: 4 }}>
              <ApiOutlined style={{ marginRight: 4 }} />微信 Bot ID
            </Text>
            <Input
              value={form.wxBotId}
              onChange={e => setForm({ ...form, wxBotId: e.target.value })}
              placeholder="扫码自动填入，回复消息必填"
              style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', color: '#fff', borderRadius: 10 }}
            />
          </div>
          {qrImage && (
            <div style={{
              marginTop: 8, padding: 12, borderRadius: 12, textAlign: 'center',
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)',
            }}>
              <img src={qrImage} alt="微信绑定二维码" style={{ width: 180, height: 180, borderRadius: 8, background: '#fff', padding: 8 }} />
              <div style={{ marginTop: 8, fontSize: 11, color: 'rgba(255,255,255,0.5)' }}>
                {qrStatus === 'wait' && <><LoadingOutlined spin style={{ marginRight: 4 }} />等待扫码…</>}
                {qrStatus === 'scaned' && '已扫码，请在手机上确认'}
                {qrStatus === 'scaned_but_redirect' && '已扫码，请在手机确认并允许登录'}
                {qrStatus === 'need_verifycode' && (
                  <div style={{ marginTop: 6 }}>
                    <div style={{ marginBottom: 4 }}>请在手机上查看配对码并输入：</div>
                    <div style={{ display: 'flex', gap: 6, justifyContent: 'center' }}>
                      <Input
                        value={verifyInput}
                        onChange={e => setVerifyInput(e.target.value)}
                        placeholder="手机显示的配对码"
                        style={{ width: 140, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', color: '#fff', borderRadius: 8 }}
                      />
                      <Button size="small" type="primary" loading={qrPolling} onClick={handleVerifyCode}
                        style={{ background: 'linear-gradient(135deg, #e85388, #c02660)', border: 'none', borderRadius: 8 }}>
                        提交
                      </Button>
                    </div>
                  </div>
                )}
                {qrStatus === 'verify_code_blocked' && (
                  <span style={{ color: '#ff4d4f' }}>配对码错误次数过多 <Button type="link" size="small" icon={<ReloadOutlined />} onClick={handleQRScan} style={{ color: '#e85388', padding: 0 }}>重新扫码</Button></span>
                )}
                {qrStatus === 'expired' && <span style={{ color: '#ff4d4f' }}>二维码已过期 <Button type="link" size="small" icon={<ReloadOutlined />} onClick={handleQRScan} style={{ color: '#e85388', padding: 0 }}>重新获取</Button></span>}
              </div>
            </div>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 14px', borderRadius: 12, background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)' }}>
          <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.6)' }}>启用助手</Text>
          <Switch checked={form.enabled} onChange={v => setForm({ ...form, enabled: v })} />
        </div>
        <Button type="primary" onClick={handleSave} loading={saving} icon={<CheckOutlined />}
          style={{
            background: 'linear-gradient(135deg, #e85388, #c02660)', border: 'none',
            borderRadius: 10, height: 40, fontWeight: 600,
            boxShadow: '0 4px 16px rgba(232,83,136,0.3)',
          }}>
          {editing ? '保存修改' : '创建助手'}
        </Button>
      </div>
    )
  }

  const handleGeneratePortrait = async (ast: WhisperAssistant) => {
    const p = getPersonality(ast.personalityId)
    setGeneratingPortrait(true)
    try {
      const genderWord = p?.gender === 'female' ? '女性' : p?.gender === 'male' ? '男性' : ''
      const guide = (p?.voiceGuide || '').split('：')[1] || p?.voiceGuide || '温柔可靠'
      const prompt = `${genderWord}角色 ${ast.name}。人格：${p?.label || '助手'}。性格设定：${guide.slice(0, 60)}。精致服饰，梦幻唯美背景，电影级光影，8K超高清，半身肖像。`
      const res = await generateImage(prompt, '', '1024x1024', portraitModel, 0, 1)
      if (res?.error) { message.error(res.error); return }
      const url = res?.images?.[0]?.image
      if (!url) { message.error('生成失败'); return }
      await (App as any).WhisperAssistantSave({ ...ast, portraitUrl: url })
      message.success('角色剧照已生成')
      setDetail({ ...ast, portraitUrl: url })
      reload()
    } catch (err: any) {
      message.error(err?.message || '剧照生成失败')
    } finally {
      setGeneratingPortrait(false)
    }
  }

  const handleExportToNovel = async (ast: WhisperAssistant) => {
    const p = getPersonality(ast.personalityId)
    try {
      const ch = {
        id: `whisper_${ast.id}`,
        name: ast.name,
        role_type: 'supporting',
        gender: ast.gender || p?.gender || '',
        age: '',
        personality: ast.voiceGuide || p?.voiceGuide || `${p?.label || ast.name}人格`,
        background: '',
        appearance: '',
        figure: '',
        motivation: '',
        arc: '',
        status: 'Alive',
        portrait_url: ast.portraitUrl || '',
      }
      await (App as any).SaveCharacter(JSON.stringify(ch))
      message.success(`已导出「${ast.name}」到小说角色`)
    } catch (err: any) {
      message.error(err?.message || '导出失败（请先打开小说项目）')
    }
  }

  const renderList = () => {
    const sorted = [...assistants].sort((a, b) => {
      const rank = (x: WhisperAssistant) => {
        if (x.personalityId === 'gaea' || x.name === 'gaea') return 0
        if (x.personalityId === activePersonality) return 1
        return x.enabled ? 2 : 3
      }
      return rank(a) - rank(b)
    })
    return (
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(208px, 1fr))', gap: 14 }}>
        {sorted.map(ast => {
          const p = getPersonality(ast.personalityId)
          const wxOn = wxStatuses[ast.id] || false
          const isActive = ast.personalityId === activePersonality
          const accent = PERSONALITY_COLORS[ast.personalityId] || '#a855f7'
          return (
            <div
              key={ast.id}
              onClick={() => setDetail(ast)}
              title="查看详情与参数"
              style={{
                position: 'relative',
                background: 'linear-gradient(160deg, rgba(255,255,255,0.05), rgba(255,255,255,0.015))',
                backdropFilter: 'blur(14px)', WebkitBackdropFilter: 'blur(14px)',
                border: isActive
                  ? `1px solid ${accent}55`
                  : `1px solid ${ast.enabled ? 'rgba(255,255,255,0.09)' : 'rgba(255,255,255,0.04)'}`,
                borderRadius: 16,
                borderLeft: `3px solid ${accent}`,
                boxShadow: isActive ? `0 0 22px ${accent}30` : '0 6px 20px rgba(0,0,0,0.25)',
                overflow: 'hidden',
                cursor: ast.enabled ? 'pointer' : 'not-allowed',
                opacity: ast.enabled ? 1 : 0.45,
                display: 'flex', flexDirection: 'column',
                transition: 'transform 0.25s, box-shadow 0.25s, border-color 0.25s',
              }}
              onMouseEnter={(e) => { if (ast.enabled) { e.currentTarget.style.transform = 'translateY(-3px)'; e.currentTarget.style.boxShadow = `0 12px 30px rgba(0,0,0,0.35), 0 0 24px ${accent}30` } }}
              onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = isActive ? `0 0 22px ${accent}30` : '0 6px 20px rgba(0,0,0,0.25)' }}
            >
              <div style={{
                height: 108, position: 'relative', flexShrink: 0,
                background: ast.portraitUrl ? 'none' : `linear-gradient(135deg, ${accent}33, ${accent}14 55%, rgba(255,255,255,0.02))`,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                overflow: 'hidden',
              }}>
                {ast.portraitUrl ? (
                  <img src={ast.portraitUrl} alt={ast.name} style={{ width: '100%', height: '100%', objectFit: 'cover', objectPosition: 'top center', display: 'block' }} />
                ) : (
                  <CompanionAvatar size={88} state="idle" emotionColor={ast.enabled ? accent : '#666'} />
                )}
                {isActive && (
                  <span style={{
                    position: 'absolute', top: 8, left: 8,
                    fontSize: 9, padding: '2px 8px', borderRadius: 8, fontWeight: 600,
                    background: `${accent}26`, color: accent,
                    border: `1px solid ${accent}44`, letterSpacing: '0.05em',
                  }}>
                    ● 当前对话
                  </span>
                )}
                {ast.wxToken && (
                  <span style={{
                    position: 'absolute', top: 8, right: 8, fontSize: 9,
                    display: 'inline-flex', alignItems: 'center', gap: 3,
                    color: wxOn ? '#4ade80' : wxSessionExpired[ast.id] ? '#fbbf24' : '#94a3b8',
                  }}>
                    <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'currentColor', boxShadow: `0 0 6px currentColor` }} />
                    微信{wxOn ? '在线' : wxSessionExpired[ast.id] ? '过期' : '离线'}
                  </span>
                )}
              </div>

              <div style={{ padding: '10px 12px 12px', display: 'flex', flexDirection: 'column', gap: 4, flex: 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                  <Text strong style={{ fontSize: 14, color: ast.enabled ? '#fff' : 'rgba(255,255,255,0.5)' }}>
                    {ast.name}
                  </Text>
                  {ast.name === 'gaea' || ast.personalityId === 'gaea'
                    ? <Tag color="green" style={{ fontSize: 9, margin: 0 }}>AI 助手</Tag>
                    : <Tag color="geekblue" style={{ fontSize: 9, margin: 0 }}>角色</Tag>}
                  {!ast.enabled && <Tag style={{ fontSize: 9, margin: 0 }}>已禁用</Tag>}
                </div>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.45)' }}>
                  {p ? `${p.label} · ${p.gender === 'female' ? '♀' : p.gender === 'male' ? '♂' : '✦'}` : ast.personalityId}
                </div>
                <div style={{ height: 1, background: 'rgba(255,255,255,0.07)', margin: '3px 0' }} />
                {(ast.tags?.length || p?.tags?.length) ? (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 2 }}>
                    {(ast.tags || p?.tags || []).slice(0, 3).map((t) => (
                      <span key={t} style={{
                        fontSize: 9, padding: '1px 7px', borderRadius: 999,
                        background: `${accent}1f`, color: accent, border: `1px solid ${accent}33`,
                        whiteSpace: 'nowrap',
                      }}>{t}</span>
                    ))}
                  </div>
                ) : (
                  <div style={{ height: 2 }} />
                )}
                <Text style={{
                  color: 'rgba(255,255,255,0.4)', fontSize: 10, lineHeight: 1.5,
                  overflow: 'hidden', textOverflow: 'ellipsis',
                  display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical',
                  flex: 1,
                }}>
                  {p?.voiceGuide || '暂无人格描述'}
                </Text>
                {p && (
                  <div style={{ display: 'flex', gap: 3, marginTop: 2 }}>
                    {[{ k: 'T', v: p.dims.T }, { k: 'I', v: p.dims.I }, { k: 'S', v: p.dims.S }, { k: 'O', v: p.dims.O }, { k: 'R', v: p.dims.R }].map((d) => (
                      <div key={d.k} style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
                        <div style={{ height: 2, borderRadius: 1, background: 'rgba(255,255,255,0.08)', overflow: 'hidden' }}>
                          <div style={{ height: '100%', width: `${Math.min(100, Math.max(0, d.v))}%`, background: accent, opacity: 0.85 }} />
                        </div>
                        <span style={{ fontSize: 7.5, color: 'rgba(255,255,255,0.3)', textAlign: 'center', lineHeight: 1 }}>{d.k}</span>
                      </div>
                    ))}
                  </div>
                )}
                <div style={{ display: 'flex', gap: 2, marginTop: 6 }}>
                  <Button type="text" size="small" icon={<EditOutlined />}
                    onClick={(e) => { e.stopPropagation(); startEdit(ast) }}
                    style={{ color: 'rgba(255,255,255,0.45)', width: 26, height: 26, fontSize: 12 }} />
                  <Popconfirm title="删除此助手？" onConfirm={(e) => { e?.stopPropagation?.(); handleDelete(ast.id) }} okText="删除" cancelText="取消">
                    <Button type="text" size="small" danger icon={<DeleteOutlined />}
                      onClick={(e) => e.stopPropagation()}
                      style={{ width: 26, height: 26, fontSize: 12 }} />
                  </Popconfirm>
                  <span style={{ marginLeft: 'auto', fontSize: 9, color: 'rgba(255,255,255,0.25)', alignSelf: 'center' }}>
                    点击切换
                  </span>
                </div>
              </div>
            </div>
          )
        })}

        <div
          onClick={startNew}
          style={{
            borderRadius: 16, border: '1.5px dashed rgba(255,255,255,0.12)',
            background: 'rgba(255,255,255,0.015)',
            minHeight: 210, display: 'flex', flexDirection: 'column',
            alignItems: 'center', justifyContent: 'center', gap: 8, cursor: 'pointer',
            color: 'rgba(255,255,255,0.4)',
            transition: 'transform 0.25s, border-color 0.25s, color 0.25s',
          }}
          onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-3px)'; e.currentTarget.style.borderColor = 'rgba(232,83,136,0.4)'; e.currentTarget.style.color = 'rgba(255,255,255,0.7)' }}
          onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.borderColor = 'rgba(255,255,255,0.12)'; e.currentTarget.style.color = 'rgba(255,255,255,0.4)' }}
        >
          <span style={{ width: 44, height: 44, borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(255,255,255,0.05)', fontSize: 20 }}>
            <PlusOutlined />
          </span>
          <Text style={{ fontSize: 12, fontWeight: 500 }}>新建角色</Text>
          <Text style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>独立人格 · 可绑微信</Text>
        </div>
      </div>
    )
  }

  const renderDetail = (ast: WhisperAssistant) => {
    const p = getPersonality(ast.personalityId)
    const wxOn = wxStatuses[ast.id] || false
    const accent = PERSONALITY_COLORS[ast.personalityId] || '#a855f7'
    const isActive = ast.personalityId === activePersonality
    const dimKeys = [
      { k: 'T', label: '温柔度', v: p?.dims.T ?? 0 },
      { k: 'I', label: '主动性', v: p?.dims.I ?? 0 },
      { k: 'S', label: '顺从度', v: p?.dims.S ?? 0 },
      { k: 'O', label: '独特度', v: p?.dims.O ?? 0 },
      { k: 'R', label: '矜持度', v: p?.dims.R ?? 0 },
    ]
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <Button type="text" icon={<CloseOutlined />} onClick={() => setDetail(null)}
          style={{ alignSelf: 'flex-start', color: 'rgba(255,255,255,0.5)', fontSize: 12 }}>
          返回角色列表
        </Button>

        <div style={{
          display: 'flex', alignItems: 'center', gap: 18,
          padding: '18px 20px', borderRadius: 18,
          background: `linear-gradient(135deg, ${accent}2e, ${accent}12 55%, rgba(255,255,255,0.02))`,
          border: `1px solid ${accent}33`,
        }}>
          <div style={{
            width: 128, height: 148, borderRadius: 16, flexShrink: 0, overflow: 'hidden',
            background: 'rgba(255,255,255,0.03)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            border: `1px solid ${accent}33`,
          }}>
            {ast.portraitUrl ? (
              <img src={ast.portraitUrl} alt={ast.name} style={{ width: '100%', height: '100%', objectFit: 'cover', objectPosition: 'top center', display: 'block' }} />
            ) : (
              <CompanionAvatar size={124} state="idle" emotionColor={ast.enabled ? accent : '#666'} />
            )}
          </div>
          <div style={{ minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Text style={{ fontSize: 20, fontWeight: 700, color: '#fff' }}>{ast.name}</Text>
              {isActive && <Tag color={accent} style={{ margin: 0, fontWeight: 600 }}>● 当前对话</Tag>}
              {!ast.enabled && <Tag style={{ margin: 0 }}>已禁用</Tag>}
            </div>
            <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.5)', marginTop: 4 }}>
              {p ? `${p.label} 人格 · ${p.gender === 'female' ? '♀ 女性' : p.gender === 'male' ? '♂ 男性' : '✦ 中性'}` : ast.personalityId}
            </div>
            {(ast.tags?.length || p?.tags?.length) ? (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 8 }}>
                {(ast.tags || p?.tags || []).map((t) => (
                  <span key={t} style={{
                    fontSize: 10, padding: '2px 10px', borderRadius: 999,
                    background: `${accent}1f`, color: accent, border: `1px solid ${accent}44`,
                  }}>{t}</span>
                ))}
              </div>
            ) : null}
            {ast.wxToken && (
              <div style={{ fontSize: 11, color: wxOn ? '#4ade80' : wxSessionExpired[ast.id] ? '#fbbf24' : '#94a3b8', marginTop: 6, display: 'flex', alignItems: 'center', gap: 5 }}>
                <span style={{ width: 7, height: 7, borderRadius: '50%', background: 'currentColor', boxShadow: '0 0 6px currentColor' }} />
                微信通道 {wxOn ? '在线' : wxSessionExpired[ast.id] ? '会话过期 · 需重新绑定' : '离线'}
              </div>
            )}
          </div>
        </div>

        <div style={{
          padding: '14px 18px', borderRadius: 16,
          background: 'rgba(255,255,255,0.025)', border: '1px solid rgba(255,255,255,0.07)',
          display: 'flex', gap: 18, alignItems: 'center',
        }}>
          <div style={{ flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            {p && <TisorRadar dims={p.dims} size={132} color={ast.enabled ? accent : '#666'} showLabels={false} />}
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', display: 'block', marginBottom: 10, letterSpacing: '0.08em' }}>
              人格五维参数
            </Text>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {dimKeys.map(d => (
                <div key={d.k} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{ width: 40, fontSize: 10, color: 'rgba(255,255,255,0.45)', flexShrink: 0 }}>{d.label}</span>
                  <div style={{ flex: 1, height: 4, borderRadius: 2, background: 'rgba(255,255,255,0.08)', overflow: 'hidden' }}>
                    <div style={{ width: `${Math.min(100, Math.max(0, d.v))}%`, height: '100%', borderRadius: 2, background: `linear-gradient(90deg, ${accent}66, ${accent})`, transition: 'width 600ms cubic-bezier(0.4,0,0.2,1)' }} />
                  </div>
                  <span style={{ width: 26, fontSize: 12, fontWeight: 700, color: accent, textAlign: 'right', flexShrink: 0 }}>{Math.round(d.v)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div style={{ padding: '14px 18px', borderRadius: 16, background: 'rgba(255,255,255,0.025)', border: '1px solid rgba(255,255,255,0.07)' }}>
          <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', display: 'block', marginBottom: 8, letterSpacing: '0.08em' }}>
            人格设定 · VoiceGuide
          </Text>
          <Text style={{ fontSize: 12.5, color: 'rgba(255,255,255,0.7)', lineHeight: 1.8 }}>
            {p?.voiceGuide || '暂无人格描述'}
          </Text>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: p?.tags?.length ? '1fr 1fr' : '1fr', gap: 12 }}>
          {p?.tags?.length ? (
            <div style={{ padding: '14px 18px', borderRadius: 16, background: 'rgba(255,255,255,0.025)', border: '1px solid rgba(255,255,255,0.07)' }}>
              <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', display: 'block', marginBottom: 8, letterSpacing: '0.08em' }}>标签</Text>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                {p.tags.map(t => <Tag key={t} color={accent} style={{ margin: 0, fontSize: 10 }}>{t}</Tag>)}
              </div>
            </div>
          ) : null}
          <div style={{ padding: '14px 18px', borderRadius: 16, background: 'rgba(255,255,255,0.025)', border: '1px solid rgba(255,255,255,0.07)' }}>
            <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', display: 'block', marginBottom: 8, letterSpacing: '0.08em' }}>微信参数</Text>
            {ast.wxToken ? (
              <>
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.55)' }}>Token：{ast.wxToken.slice(0, 8)}…</div>
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.55)', marginTop: 4 }}>Bot ID：{ast.wxBotId || '未绑定'}</div>
              </>
            ) : (
              <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>未绑定微信</Text>
            )}
          </div>
        </div>

        <div style={{ display: 'flex', gap: 10, marginTop: 4, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', gap: 6 }}>
            <Select
              size="middle" placeholder="出图模型" value={portraitModel || undefined}
              onChange={setPortraitModel}
              style={{ width: 168, height: 40 }}
              popupMatchSelectWidth={false}
              options={(portraitAvailModels.length ? portraitAvailModels : [{ engine: 'ComfyUI', model: 'krea2' }, { engine: 'ComfyUI', model: 'z-image-turbo' }]).map(m => ({ value: m.model, label: `${m.model} (${m.engine})` }))}
            />
            <Button
              icon={generatingPortrait ? <LoadingOutlined /> : <PictureOutlined />}
              loading={generatingPortrait}
              onClick={() => handleGeneratePortrait(ast)}
              style={{ height: 40, borderRadius: 12, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.1)', color: 'rgba(255,255,255,0.6)' }}>
              {ast.portraitUrl ? '重新生成剧照' : '生成剧照'}
            </Button>
          </div>
          <Button
            icon={<ReadOutlined />}
            onClick={() => handleExportToNovel(ast)}
            style={{ height: 40, borderRadius: 12, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.1)', color: 'rgba(255,255,255,0.6)' }}>
            导出到小说
          </Button>
          <Button type="primary" style={{
            flex: 1, minWidth: 220, height: 40, borderRadius: 12, fontWeight: 600,
            background: `linear-gradient(135deg, ${accent}, ${accent}cc)`, border: 'none',
          }}
            disabled={!ast.enabled}
            onClick={() => switchTo(ast.personalityId)}>
            {isActive ? '正在与此角色对话' : `切换为「${ast.name}」对话`}
          </Button>
          <Button icon={<EditOutlined />} onClick={() => { setDetail(null); startEdit(ast) }}
            style={{ height: 40, borderRadius: 12, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.1)', color: 'rgba(255,255,255,0.6)' }}>
            编辑
          </Button>
          <Popconfirm title="删除此助手？" onConfirm={() => { handleDelete(ast.id); setDetail(null) }} okText="删除" cancelText="取消">
            <Button danger icon={<DeleteOutlined />} style={{ height: 40, borderRadius: 12 }} />
          </Popconfirm>
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      {/* 面板头 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12,
        padding: '10px 14px', marginBottom: 14,
        borderRadius: 14,
        background: 'rgba(255,255,255,0.025)', border: '1px solid rgba(255,255,255,0.07)',
        backdropFilter: 'blur(18px)',
      }}>
        <span style={{
          fontSize: 16, width: 34, height: 34, borderRadius: 11, flexShrink: 0,
          background: 'linear-gradient(135deg, rgba(232,83,136,0.16), rgba(168,85,247,0.1))',
          border: '1px solid rgba(232,83,136,0.22)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <UserOutlined style={{ color: '#e85388', fontSize: 14 }} />
        </span>
        <div style={{ minWidth: 0 }}>
          <div style={{ fontSize: 14, fontWeight: 700, color: 'rgba(255,255,255,0.92)' }}>
            {detail ? `${detail.name} · 角色详情` : editing ? (editing.id ? `编辑 ${editing.name}` : '新建角色') : `轻语角色（${assistants.length}）`}
          </div>
          <Text style={{ fontSize: 10.5, color: 'rgba(255,255,255,0.4)' }}>
            {detail ? '角色参数设定 · 参照小说角色卡' : editing ? '' : 'gaea 为核心 AI 助手，其余为角色 · 点击卡片切换对话人格'}
          </Text>
        </div>
        <div style={{ flex: 1 }} />
        {!detail && !editing && (
          <Button size="small" icon={<PlusOutlined />} onClick={startNew}
            style={{ borderRadius: 10, background: 'rgba(232,83,136,0.14)', border: '1px solid rgba(232,83,136,0.28)', color: '#f472b6', fontWeight: 600 }}>
            新建角色
          </Button>
        )}
      </div>

      <div style={{ flex: 1, overflow: 'auto', minHeight: 0, padding: '0 2px 12px' }}>
        {detail ? renderDetail(detail) : editing ? renderEditor() : renderList()}
      </div>
    </div>
  )
}
