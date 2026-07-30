// AssistantManagerModal.tsx — 虚拟助手管理中心
// 替代 WhisperPersonalityModal，管理多个助手（每人独立人格 + 微信）
import React, { useState, useEffect, useCallback } from 'react'
import { Modal, Button, Input, Switch, Tag, Typography, Popconfirm, message, Empty } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, UserOutlined, ApiOutlined, CloseOutlined, CheckOutlined, QrcodeOutlined, LoadingOutlined, ReloadOutlined } from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import TisorRadar from './TisorRadar'
import PersonalityPreview from './PersonalityPreview'

const { Text } = Typography

// ─── 类型 ────────────────────────────────────────────────────

interface Assistant {
  id: string; name: string; personalityId: string
  wxToken: string; wxUserId: string; enabled: boolean
}

interface PersonalityPreset {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}

interface Props {
  open: boolean
  activePersonality: string
  adultMode: boolean
  onClose: () => void
  onSwitchPersonality: (id: string) => void
}

// ─── 组件 ────────────────────────────────────────────────────

export default function AssistantManagerModal({ open, activePersonality, adultMode, onClose, onSwitchPersonality }: Props) {
  // 状态
  const [assistants, setAssistants] = useState<Assistant[]>([])
  const [personalities, setPersonalities] = useState<PersonalityPreset[]>([])
  const [wxStatuses, setWxStatuses] = useState<Record<string, boolean>>({})
  const [editing, setEditing] = useState<Assistant | null>(null) // null=列表视图, Assistant=编辑
  const [form, setForm] = useState<Assistant>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [showPersonalityPicker, setShowPersonalityPicker] = useState(false)
  // QR 扫码
  const [qrImage, setQrImage] = useState('')
  const [qrCode, setQrCode] = useState('')
  const [qrStatus, setQrStatus] = useState('') // wait | scanned | confirmed | expired
  const [qrPolling, setQrPolling] = useState(false)
  // 加载数据
  const reload = useCallback(async () => {
    try {
      const list: Assistant[] = await (App as any).WhisperAssistantList()
      setAssistants(list || [])
      const statuses: any[] = await (App as any).WhisperWeixinStatus()
      const map: Record<string, boolean> = {}
      if (statuses) statuses.forEach((s: any) => { map[s.id] = s.wxRunning })
      setWxStatuses(map)
    } catch (_) {}
  }, [])

  useEffect(() => {
    if (open) {
      reload()
      App.WhisperGetPersonalities().then(setPersonalities).catch(() => {})
    }
  }, [open, reload])

  // 保存助手
  const handleSave = async () => {
    if (!form.name.trim()) return message.warning('请输入助手名称')
    setSaving(true)
    try {
      await (App as any).WhisperAssistantSave({
        id: form.id || `ast_${Date.now()}`,
        name: form.name.trim(),
        personalityId: form.personalityId || 'deredere',
        wxToken: form.wxToken || '',
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
  // QR 扫码绑定
  const handleQRScan = async () => {
    try {
      setQrStatus('')
      const qr: any = await (App as any).WhisperWeixinGetQR()
      setQrImage(qr.imageUrl)
      setQrCode(qr.qrcode)
      setQrStatus('wait')
      setQrPolling(true)

      // 轮询扫码状态
      const poll = setInterval(async () => {
        try {
          const s: any = await (App as any).WhisperWeixinQRStatus(qr.qrcode)
          setQrStatus(s.status || 'wait')
          if (s.status === 'confirmed' && s.botToken) {
            clearInterval(poll)
            setQrPolling(false)
            setForm(f => ({ ...f, wxToken: s.botToken }))
            message.success('微信绑定成功！')
            setQrImage('')
          } else if (s.status === 'expired') {
            clearInterval(poll)
            setQrPolling(false)
            setQrStatus('expired')
          }
        } catch (_) {}
      }, 3000)
    } catch (e: any) {
      message.error('获取二维码失败')
    }
  }

  // 删除助手
  const handleDelete = async (id: string) => {
    try {
      await (App as any).WhisperAssistantDelete(id)
      message.success('已删除')
      reload()
    } catch (e: any) {
      message.error(e?.message || '删除失败')
    }
  }

  // 开始编辑
  const startEdit = (ast: Assistant) => {
    setForm({ ...ast })
    setEditing(ast)
  }

  // 新建
  const startNew = () => {
    setForm(emptyForm())
    setEditing({ id: '', name: '', personalityId: 'deredere', wxToken: '', wxUserId: '', enabled: true })
  }

  // 获取人格信息
  const getPersonality = (id: string) => personalities.find(p => p.id === id)

  // ─── 人格选择器（简化版）───────────────────────────────────

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

  // ─── 编辑视图 ──────────────────────────────────────────────

  const renderEditor = () => {
    const selPersonality = getPersonality(form.personalityId)
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 返回按钮 */}
        <Button type="text" icon={<CloseOutlined />} onClick={() => setEditing(null)}
          style={{ alignSelf: 'flex-start', color: 'rgba(255,255,255,0.5)', fontSize: 12 }}>
          返回列表
        </Button>

        {/* 名称 */}
        <div>
          <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', display: 'block', marginBottom: 4 }}>助手名称</Text>
          <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })}
            placeholder="如：小秘书、知心姐姐…"
            style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', color: '#fff', borderRadius: 10 }} />
        </div>

        {/* 人格选择 */}
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

        {/* 微信Token */}
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
              style={{
                borderRadius: 10, border: '1px solid rgba(255,255,255,0.08)',
                background: 'rgba(255,255,255,0.04)', color: 'rgba(255,255,255,0.6)',
              }}
            >
              扫码
            </Button>
          </div>
          {/* QR 码显示 */}
          {qrImage && (
            <div style={{
              marginTop: 8, padding: 12, borderRadius: 12, textAlign: 'center',
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)',
            }}>
              <img src={qrImage} alt="QR" style={{ width: 180, height: 180, borderRadius: 8, background: '#fff', padding: 8 }} />
              <div style={{ marginTop: 8, fontSize: 11, color: 'rgba(255,255,255,0.5)' }}>
                {qrStatus === 'wait' && <><LoadingOutlined spin style={{ marginRight: 4 }} />等待扫码…</>}
                {qrStatus === 'scanned' && '已扫码，请在手机上确认'}
                {qrStatus === 'expired' && <span style={{ color: '#ff4d4f' }}>二维码已过期 <Button type="link" size="small" icon={<ReloadOutlined />} onClick={handleQRScan} style={{ color: '#e85388', padding: 0 }}>重新获取</Button></span>}
              </div>
            </div>
          )}
        </div>

        {/* 启用 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 14px', borderRadius: 12, background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)' }}>
          <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.6)' }}>启用助手</Text>
          <Switch checked={form.enabled} onChange={v => setForm({ ...form, enabled: v })} />
        </div>

        {/* 保存 */}
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

  // ─── 列表视图 ──────────────────────────────────────────────

  const renderList = () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {assistants.length === 0 ? (
        <Empty description={<span style={{ color: 'rgba(255,255,255,0.3)' }}>还没有助手</span>} />
      ) : (
        assistants.map(ast => {
          const p = getPersonality(ast.personalityId)
          const wxOn = wxStatuses[ast.id] || false
          return (
            <div key={ast.id}
              style={{
                display: 'flex', alignItems: 'center', gap: 12, padding: '12px 14px', borderRadius: 14,
                background: ast.enabled ? 'rgba(255,255,255,0.03)' : 'rgba(255,255,255,0.01)',
                border: '1px solid rgba(255,255,255,0.06)',
                opacity: ast.enabled ? 1 : 0.5,
                transition: 'all 200ms',
              }}
            >
              {/* 雷达图 */}
              {p && <TisorRadar dims={p.dims} size={44} color={ast.enabled ? '#e85388' : '#666'} showLabels={false} />}

              {/* 信息 */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <Text strong style={{ fontSize: 13, color: '#fff' }}>{ast.name}</Text>
                  {!ast.enabled && <Tag style={{ fontSize: 9, margin: 0 }}>已禁用</Tag>}
                </div>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', marginTop: 1 }}>
                  {p ? p.label : ast.personalityId}
                  {ast.wxToken && (
                    <span style={{ marginLeft: 8 }}>
                      <span style={{
                        display: 'inline-block', width: 6, height: 6, borderRadius: '50%',
                        background: wxOn ? '#52c41a' : '#ff4d4f',
                        boxShadow: wxOn ? '0 0 6px #52c41a80' : 'none',
                        marginRight: 3, verticalAlign: 'middle',
                      }} />
                      微信 {wxOn ? '在线' : '离线'}
                    </span>
                  )}
                </div>
              </div>

              {/* 操作 */}
              <div style={{ display: 'flex', gap: 2 }}>
                <Button type="text" size="small" icon={<EditOutlined />}
                  onClick={() => startEdit(ast)}
                  style={{ color: 'rgba(255,255,255,0.4)', width: 28, height: 28 }} />
                <Popconfirm title="删除此助手？" onConfirm={() => handleDelete(ast.id)} okText="删除" cancelText="取消">
                  <Button type="text" size="small" danger icon={<DeleteOutlined />}
                    style={{ width: 28, height: 28 }} />
                </Popconfirm>
              </div>
            </div>
          )
        })
      )}

      {/* 新建按钮 */}
      <Button onClick={startNew} icon={<PlusOutlined />}
        style={{
          marginTop: 4, borderRadius: 10, height: 40,
          background: 'rgba(255,255,255,0.03)', border: '1px dashed rgba(255,255,255,0.1)',
          color: 'rgba(255,255,255,0.5)', fontWeight: 500,
        }}>
        新建虚拟助手
      </Button>
    </div>
  )

  // ─── 主弹窗 ────────────────────────────────────────────────

  return (
    <Modal
      title={null}
      open={open}
      onCancel={onClose}
      footer={null}
      width={640}
      centered
      styles={{
        body: {
          padding: 0, maxHeight: '72vh', overflow: 'auto',
          background: 'linear-gradient(180deg, #0d0d14 0%, #111119 100%)',
        },
      }}
      style={{
        background: 'linear-gradient(180deg, #0f0f18 0%, #13131e 100%)',
        border: '1px solid rgba(255,255,255,0.06)',
        borderRadius: 18,
        boxShadow: '0 24px 80px rgba(0,0,0,0.5)',
        overflow: 'hidden',
      }}
    >
      {/* 标题栏 */}
      <div style={{
        padding: '16px 24px 12px', borderBottom: '1px solid rgba(255,255,255,0.05)',
        background: 'rgba(255,255,255,0.015)', backdropFilter: 'blur(20px)',
        display: 'flex', alignItems: 'center', gap: 12,
      }}>
        <span style={{
          fontSize: 20, width: 40, height: 40, borderRadius: 12,
          background: 'linear-gradient(135deg, rgba(232,83,136,0.15), rgba(168,85,247,0.1))',
          border: '1px solid rgba(232,83,136,0.2)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <UserOutlined style={{ color: '#e85388', fontSize: 16 }} />
        </span>
        <div>
          <div style={{ fontSize: 16, fontWeight: 700, color: '#fff' }}>
            {editing ? (editing.id ? `编辑 ${editing.name}` : '新建虚拟助手') : '虚拟助手管理中心'}
          </div>
          <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
            {editing ? '' : `${assistants.length} 个助手 · 各绑定独立人格与微信`}
          </Text>
        </div>
      </div>

      {/* 内容区 */}
      <div style={{ padding: '20px 24px' }}>
        {editing ? renderEditor() : renderList()}
      </div>
    </Modal>
  )
}

function emptyForm(): Assistant {
  return { id: '', name: '', personalityId: 'deredere', wxToken: '', wxUserId: '', enabled: true }
}
