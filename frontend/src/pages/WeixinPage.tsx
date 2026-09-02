import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert, Avatar, Button, DatePicker, Input, Modal, Popconfirm, Space, Switch, Tag, Tooltip, Typography, message,
} from 'antd'
import {
  QrcodeOutlined, ReloadOutlined, SendOutlined, BellOutlined, DeleteOutlined, CheckCircleOutlined, WarningOutlined, PlusOutlined,
  WechatOutlined, ClockCircleOutlined, MessageOutlined, BookOutlined, NotificationOutlined, SearchOutlined,
} from '@ant-design/icons'
import dayjs, { type Dayjs } from 'dayjs'
import { app } from '../gaea/lib/bridge'
import type { whisper, characterlib } from '../../wailsjs/go/models'
import type {
  WeixinAssistantStatusRow, WeixinAssistantView, WeixinReminderConfigView, WeixinReminderView,
} from '../gaea/lib/types'
import { isPageVisible } from '../lib/pollingGate'
import './weixin-page.css'

/**
 * WeixinPage — 青鸟（微信助手）「通讯枢纽」工作台（v4.47 星枢化重构；
 * v4.48 板块更名「青鸟」+ 新增流人格选择器打通角色库）。
 *
 * 布局（Constellation OS 三分区语言）：顶部玻璃细条（板块名 + 通道遥测 meta +
 * 刷新）/ 左通道轨道（每助手一条 rail item：头像 + 名字 + 状态字 + 状态点，
 * 「+ 新增助手」挂组尾；支撑组 = 离线提醒 / 使用指南）/ 主区三视图（助手详情、
 * 离线代办、使用指南）。功能面与 v4.4 逐项对齐零删减：多助手管理（人格 Tag/
 * 状态徽标/启停/删除/逐助手扫码绑定 + 新增流）、离线提醒（全局开关 + 手动
 * 新建 + 列表）、使用说明、会话过期警示。
 *
 * 数据面全部走 app 代理 work 面：WhisperWeixinStatus（通道状态）与
 * WhisperAssistantList（完整字段）按 id merge 成助手行；除删除外保存一律传
 * List 完整对象（后端契约：空 token 字段保留现值，保存后自动重拉通道）。
 * id='gaea' 为核心助手：禁删禁停。
 */

const POLL_MS = 5000

type MainView = 'channel' | 'reminders' | 'guide'

const statusTag = (row: WeixinAssistantStatusRow) => {
  if (!row.hasToken) return <Tag>未绑定</Tag>
  if (row.wxSessionExpired) return <Tag color="warning" icon={<WarningOutlined />}>会话过期</Tag>
  if (row.wxRunning) return <Tag color="success" icon={<CheckCircleOutlined />}>运行中</Tag>
  return <Tag color="default">已停止</Tag>
}

// 通道三态投影（rail 状态字/点 + 详情文案共用；与 statusTag 同口径）
const channelStatusOf = (row: WeixinAssistantStatusRow): { kind: 'running' | 'expired' | 'stopped' | 'unbound'; text: string } => {
  if (!row.hasToken) return { kind: 'unbound', text: '未绑定' }
  if (row.wxSessionExpired) return { kind: 'expired', text: '会话过期' }
  if (row.wxRunning) return { kind: 'running', text: '运行中' }
  return { kind: 'stopped', text: '已停止' }
}

const reminderTag = (status: string) => {
  switch (status) {
    case 'pending': return <Tag color="processing">待触发</Tag>
    case 'done': return <Tag color="success">已推送</Tag>
    case 'failed': return <Tag color="error">推送失败</Tag>
    default: return <Tag>{status}</Tag>
  }
}

// ── 助手行：List 完整字段 + 挂载的 Status 通道状态 ──
interface AssistantRow extends WeixinAssistantView {
  status?: WeixinAssistantStatusRow
}

// ── 人格选择器选项：轻语预设 + 角色库可聊天角色统一投影 ──
interface PersonaOption {
  id: string
  name: string
  group: 'preset' | 'character'
  gender?: string
  tags?: string[]
  portraitUrl?: string
  desc?: string // 预设=voiceGuide / 角色=personality+background
}

// 状态行（WhisperWeixinStatus）与字段行（WhisperAssistantList）按 id merge：
// 字段以 List 为准、通道状态挂 status；Status 独有行（后端兜底自建未落库时）
// 仅以状态字段兜底展示。
const mergeAssistantRows = (list: WeixinAssistantView[], statuses: WeixinAssistantStatusRow[]): AssistantRow[] => {
  const statusById = new Map(statuses.map((s) => [s.id, s]))
  const rows: AssistantRow[] = list.map((a) => ({ ...a, status: statusById.get(a.id) }))
  for (const s of statuses) {
    if (!rows.some((r) => r.id === s.id)) {
      rows.push({ id: s.id, name: s.name, personalityId: s.personalityId, enabled: s.enabled, status: s })
    }
  }
  return rows
}

// 行 → 保存载荷：剥掉前端合并用的 status 字段（其余原样透传，空 token 字段由
// 后端按契约保留现值）
const viewOf = (row: AssistantRow): WeixinAssistantView => ({
  id: row.id, name: row.name, personalityId: row.personalityId,
  wxToken: row.wxToken, wxBotId: row.wxBotId, wxUserId: row.wxUserId,
  enabled: row.enabled, portraitUrl: row.portraitUrl,
})

// 行 → 状态行投影（Status 未回时按 wxToken 兜底 hasToken，供状态区复用）
const rowStatus = (row: AssistantRow): WeixinAssistantStatusRow => ({
  id: row.id, name: row.name, personalityId: row.personalityId, enabled: row.enabled,
  hasToken: row.status?.hasToken ?? Boolean(row.wxToken),
  wxRunning: row.status?.wxRunning ?? false,
  wxSessionExpired: row.status?.wxSessionExpired,
})

/** 扫码相位 → 三步指示（扫码 → 确认 → 完成）的当前步。 */
const qrStepIndex = (phase: string): number => {
  switch (phase) {
    case 'scanned': case 'needVerify': return 1
    case 'confirmed': return 2
    default: return 0
  }
}

const WeixinPage: React.FC = () => {
  const [rows, setRows] = useState<AssistantRow[]>([])
  const [reminders, setReminders] = useState<WeixinReminderView[]>([])
  const [cfg, setCfg] = useState<WeixinReminderConfigView | null>(null)

  // 主区视图 + 选中通道（默认首条；数据刷新后选中 id 消失则回退首条）
  const [view, setView] = useState<MainView>('channel')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selected = useMemo(() => rows.find((r) => r.id === selectedId) ?? null, [rows, selectedId])
  useEffect(() => {
    if (selectedId && rows.some((r) => r.id === selectedId)) return
    setSelectedId(rows[0]?.id ?? null)
  }, [rows, selectedId])

  // ── 扫码绑定流（target 为绑定目标助手；新增流传本地暂存对象）──
  const [qrTarget, setQrTarget] = useState<WeixinAssistantView | null>(null)
  const [qrOpen, setQrOpen] = useState(false)
  const [qrImage, setQrImage] = useState('')
  const [qrToken, setQrToken] = useState('')
  const [qrPhase, setQrPhase] = useState<'idle' | 'waiting' | 'scanned' | 'needVerify' | 'confirmed' | 'error'>('idle')
  const [verifyCode, setVerifyCode] = useState('')
  const [binding, setBinding] = useState(false)
  const pollRef = useRef<number | null>(null)

  // ── 新增助手表单（人格选择器：轻语预设 + 角色库可聊天角色）──
  const [addOpen, setAddOpen] = useState(false)
  const [addName, setAddName] = useState('')
  const [personaOpts, setPersonaOpts] = useState<PersonaOption[]>([])
  const [personaQuery, setPersonaQuery] = useState('')
  const [personaSel, setPersonaSel] = useState('gaea')

  // 人格选择器数据：预设清单 + 角色库可聊天角色（打开弹窗时拉一次；
  // 默认选中 gaea 预设，加载失败时确认流仍可走——payload 照常 'gaea'）
  useEffect(() => {
    if (!addOpen) return
    let alive = true
    setPersonaOpts([])
    setPersonaQuery('')
    setPersonaSel('gaea')
    ;(async () => {
      const [presets, charRes] = await Promise.all([
        app.WhisperGetPersonalities().catch(() => [] as whisper.PersonalityPreset[]),
        app.CharacterList('', '', true, 1, 500).catch(() => null as Record<string, unknown> | null),
      ])
      if (!alive) return
      const presetOpts: PersonaOption[] = (presets ?? [])
        .filter((p) => !p.requiresAdult18) // 工作触点不列 18+ 人格
        .map((p) => ({
          id: p.id, name: p.label, group: 'preset' as const,
          gender: p.gender, tags: p.tags, desc: p.voiceGuide || '',
        }))
      const items = ((charRes?.items ?? []) as characterlib.Character[])
      const charOpts: PersonaOption[] = items.map((c) => ({
        id: c.id, name: c.name, group: 'character' as const,
        gender: c.gender || undefined, tags: c.tags, portraitUrl: c.portraitUrl,
        desc: [c.personality, c.background].filter(Boolean).join('\n'),
      }))
      setPersonaOpts([...presetOpts, ...charOpts])
    })()
    return () => { alive = false }
  }, [addOpen])

  // 选择器派生：搜索过滤 + 分组 + 当前选中项
  const personaFiltered = useMemo(() => {
    const q = personaQuery.trim().toLowerCase()
    if (!q) return personaOpts
    return personaOpts.filter((o) =>
      o.name.toLowerCase().includes(q) || (o.tags ?? []).some((t) => t.toLowerCase().includes(q)))
  }, [personaOpts, personaQuery])
  const presetGroup = personaFiltered.filter((o) => o.group === 'preset')
  const charGroup = personaFiltered.filter((o) => o.group === 'character')
  const personaSelOpt = personaOpts.find((o) => o.id === personaSel) ?? null

  // ── 手动新建提醒 ──
  const [newText, setNewText] = useState('')
  const [newTime, setNewTime] = useState<Dayjs | null>(null)
  const [adding, setAdding] = useState(false)

  const loadAssistants = useCallback(async () => {
    try {
      // 两路并行：状态用 Status 行、字段用 List 行；单路失败兜底空数组，
      // 另一路照常渲染（Status 独有行仍有兜底视图）。
      const [statuses, list] = await Promise.all([
        app.WhisperWeixinStatus().catch(() => [] as WeixinAssistantStatusRow[]),
        app.WhisperAssistantList().catch(() => [] as WeixinAssistantView[]),
      ])
      setRows(mergeAssistantRows(list, statuses))
    } catch { /* 后端未就绪时静默 */ }
  }, [])

  const loadReminders = useCallback(async () => {
    try {
      const [list, config] = await Promise.all([app.WeixinReminderList(), app.WeixinReminderConfig()])
      setReminders(list)
      setCfg(config)
    } catch { /* 静默 */ }
  }, [])

  // 可见时轮询助手管理数据 + 提醒列表（keepAlive 页面隐藏时空转）
  useEffect(() => {
    loadAssistants()
    loadReminders()
    const timer = window.setInterval(() => {
      if (!isPageVisible()) return
      loadAssistants()
      loadReminders()
    }, POLL_MS)
    return () => window.clearInterval(timer)
  }, [loadAssistants, loadReminders])

  // 扫码轮询：waiting → scanned → confirmed（携带 token）/ need_verifycode
  useEffect(() => {
    if (!qrOpen || !qrToken) return
    let alive = true
    const poll = async () => {
      try {
        const res = qrPhase === 'needVerify' && verifyCode
          ? await app.WhisperWeixinQRStatusWithCode(qrToken, verifyCode)
          : await app.WhisperWeixinQRStatus(qrToken)
        if (!alive) return
        if (res.botToken) {
          setQrPhase('confirmed')
          return // confirmed：停止轮询，等用户确认保存
        }
        const st = String(res.status ?? '')
        setQrPhase(st === 'need_verifycode' ? 'needVerify' : st === 'wait_scan' ? 'waiting' : 'scanned')
      } catch { /* 单次轮询失败忽略，下一轮重试 */ }
    }
    poll()
    pollRef.current = window.setInterval(poll, 2000)
    return () => {
      alive = false
      if (pollRef.current) window.clearInterval(pollRef.current)
      pollRef.current = null
    }
  }, [qrOpen, qrToken, qrPhase, verifyCode])

  // 逐助手扫码绑定：target 为要绑定的助手（新增助手此时才落库）
  const startBinding = async (target?: WeixinAssistantView) => {
    setQrTarget(target ?? null)
    setQrOpen(true)
    setQrPhase('waiting')
    setQrToken('')
    setVerifyCode('')
    setQrImage('')
    try {
      const qr = await app.WhisperWeixinGetQR()
      setQrImage(qr.imageUrl)
      setQrToken(qr.qrcode)
    } catch (e) {
      setQrPhase('error')
      message.error(`获取二维码失败：${e instanceof Error ? e.message : String(e)}`)
    }
  }

  // confirmed → 保存扫码结果到目标助手（新增助手此时才落库；后端自动重拉通道）
  const confirmBinding = async () => {
    if (!qrTarget) return
    setBinding(true)
    try {
      // WhisperWeixinQRStatus(confirmed) 已含 botToken/botId/userId，重取一次快照
      const res = await app.WhisperWeixinQRStatus(qrToken)
      await app.WhisperAssistantSave({
        ...qrTarget,
        wxToken: String(res.botToken ?? ''), wxBotId: String(res.botId ?? ''), wxUserId: String(res.userId ?? ''),
      })
      message.success('微信绑定已保存，通道已重启')
      setSelectedId(qrTarget.id) // 新增助手落库后主区直接落在该通道
      setView('channel')
      setQrOpen(false)
      loadAssistants()
    } catch (e) {
      message.error(`保存绑定失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setBinding(false)
    }
  }

  // 启停：传 List 完整对象 + 翻转后的 enabled（空 token 字段后端保留现值）
  const toggleAssistant = async (row: AssistantRow, enabled: boolean) => {
    try {
      await app.WhisperAssistantSave({ ...viewOf(row), enabled })
      message.success(enabled ? '助手已启用' : '助手已停用')
      loadAssistants()
    } catch (e) {
      message.error(`保存失败：${e instanceof Error ? e.message : String(e)}`)
    }
  }

  const removeAssistant = async (row: AssistantRow) => {
    try {
      await app.WhisperAssistantDelete(row.id)
      loadAssistants()
    } catch (e) {
      message.error(`删除失败：${e instanceof Error ? e.message : String(e)}`)
    }
  }

  // 「新增青鸟助手」表单确定 → 本地暂存新助手并进入扫码流（confirmed 时 Save 落库；
  // personalityId 取选择器选中项，角色库人物的立绘一并带出）
  const createAssistant = () => {
    const name = addName.trim()
    if (!name) {
      message.warning('请填写助手名字')
      return
    }
    const sel = personaOpts.find((o) => o.id === personaSel)
    const staged: WeixinAssistantView = {
      id: `wx_${Date.now().toString(36)}`,
      name,
      personalityId: personaSel || 'gaea',
      enabled: true,
      portraitUrl: sel?.portraitUrl || '',
    }
    setAddOpen(false)
    setAddName('')
    startBinding(staged)
  }

  const addReminder = async () => {
    if (!newText.trim() || !newTime) {
      message.warning('请填写提醒事项和时间')
      return
    }
    setAdding(true)
    try {
      await app.WeixinReminderAdd(newText.trim(), newTime.second(0).millisecond(0).format('YYYY-MM-DDTHH:mm:ssZ'))
      setNewText('')
      setNewTime(null)
      message.success('提醒已创建')
      loadReminders()
    } catch (e) {
      message.error(`创建失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setAdding(false)
    }
  }

  const removeReminder = async (id: string) => {
    try {
      await app.WeixinReminderDelete(id)
      loadReminders()
    } catch (e) {
      message.error(`删除失败：${e instanceof Error ? e.message : String(e)}`)
    }
  }

  const toggleReminders = async (on: boolean) => {
    try {
      await app.WeixinReminderSetConfig(JSON.stringify({ remindersEnabled: on }))
      loadReminders()
    } catch (e) {
      message.error(`保存配置失败：${e instanceof Error ? e.message : String(e)}`)
    }
  }

  // 通道遥测 meta（strip 右侧，等宽数字）
  const runningCount = rows.filter((r) => rowStatus(r).wxRunning && rowStatus(r).hasToken).length
  const pendingCount = reminders.filter((r) => r.status === 'pending').length

  // 扫码三步指示
  const qrSteps = ['扫码', '确认', '完成']
  const qrCurrent = qrStepIndex(qrPhase)

  return (
    <div className="wx-workspace">
      <div className="wx-bg" aria-hidden="true" />
      <div className="wx-grid" aria-hidden="true" />

      {/* 顶部细条：板块名 + 一句话定位 + 通道遥测 + 刷新 */}
      <header className="wx-strip">
        <span className="wx-strip-title"><WechatOutlined aria-hidden="true" /> 青鸟</span>
        <span className="wx-strip-sub">把微信变成 gaea 的遥控器：发消息触发桌面能力，提醒到点经微信回推</span>
        <span className="wx-strip-meta" role="status">
          {rows.length} 通道 · {runningCount} 运行中{cfg ? ` · 提醒 ${cfg.remindersEnabled ? '开' : '关'}` : ''}
        </span>
        <button type="button" className="wx-refresh-btn" aria-label="刷新助手" title="刷新" onClick={loadAssistants}>
          <ReloadOutlined aria-hidden="true" />
        </button>
      </header>

      <div className="wx-body">
        {/* 左：通道轨道（助手清单即导航；支撑组挂提醒/指南） */}
        <aside className="wx-rail v3-panel" aria-label="青鸟导航">
          <div className="v3-panel-head">
            <span className="v3-panel-title">通道</span>
            <span className="v3-panel-spacer" />
            <button
              type="button" className="wx-refresh-btn" aria-label="新增助手" title="新增青鸟助手"
              onClick={() => setAddOpen(true)}
            >
              <PlusOutlined aria-hidden="true" />
            </button>
          </div>
          <nav className="wx-rail-nav">
            {rows.map((row) => {
              const st = channelStatusOf(rowStatus(row))
              const active = view === 'channel' && row.id === selectedId
              return (
                <button
                  key={row.id}
                  type="button"
                  className={`wx-rail-item wx-rail-ch${active ? ' is-active' : ''}`}
                  aria-label={`通道 ${row.name || row.id}`}
                  aria-current={active ? 'page' : undefined}
                  title={`${row.name || row.id} · ${st.text}`}
                  onClick={() => { setSelectedId(row.id); setView('channel') }}
                >
                  <Avatar size={26} src={row.portraitUrl || undefined}>{(row.name || row.id).slice(0, 1)}</Avatar>
                  <span className="wx-rail-name">{row.name || row.id}</span>
                  <span className="wx-rail-status">{st.text}<span className={`wx-dot is-${st.kind}`} aria-hidden="true" /></span>
                </button>
              )
            })}
            {rows.length === 0 && (
              <div className="wx-rail-empty">暂无青鸟助手——点上方 + 新增</div>
            )}
            <button type="button" className="wx-rail-item wx-rail-add" onClick={() => setAddOpen(true)}>
              <PlusOutlined aria-hidden="true" />
              <span className="wx-rail-name">新增青鸟助手</span>
            </button>

            <div className="wx-rail-group">支持</div>
            <button
              type="button"
              className={`wx-rail-item${view === 'reminders' ? ' is-active' : ''}`}
              aria-label={pendingCount > 0 ? `离线提醒，${pendingCount} 条待触发` : '离线提醒'}
              aria-current={view === 'reminders' ? 'page' : undefined}
              onClick={() => setView('reminders')}
            >
              <BellOutlined aria-hidden="true" style={{ fontSize: 15 }} />
              <span className="wx-rail-name">离线提醒</span>
              {pendingCount > 0 && <span className="wx-rail-badge" aria-hidden="true">{pendingCount}</span>}
            </button>
            <button
              type="button"
              className={`wx-rail-item${view === 'guide' ? ' is-active' : ''}`}
              aria-current={view === 'guide' ? 'page' : undefined}
              onClick={() => setView('guide')}
            >
              <BookOutlined aria-hidden="true" style={{ fontSize: 15 }} />
              <span className="wx-rail-name">使用指南</span>
            </button>
          </nav>
        </aside>

        {/* 中：主区三视图 */}
        <main className="wx-main v3-panel" aria-label="主区视图">
          <div className="wx-main-scroll">
            {view === 'channel' && (
              selected ? (
                <ChannelDetail
                  row={selected}
                  core={selected.id === 'gaea'}
                  onBind={() => startBinding(viewOf(selected))}
                  onToggle={(v) => toggleAssistant(selected, v)}
                  onDelete={() => removeAssistant(selected)}
                />
              ) : (
                <div className="wx-empty">
                  <span className="wx-empty-icon"><WechatOutlined aria-hidden="true" /></span>
                  <span className="wx-empty-title">还没有青鸟助手</span>
                  <span className="wx-empty-hint">
                    新增助手并扫码绑定微信后，即可在微信里与 gaea 对话、设提醒、收推送。
                  </span>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>新增青鸟助手</Button>
                </div>
              )
            )}

            {view === 'reminders' && (
              <>
                <div className="wx-main-head">
                  <div>
                    <div className="wx-main-title">离线代办提醒</div>
                    <div className="wx-main-sub">到点由桌面端经微信回推——桌面常驻，人不在电脑前也能收到。</div>
                  </div>
                  {cfg && (
                    <span className="wx-rem-config">
                      到点回推微信
                      <Switch
                        size="small" aria-label="到点回推微信" checked={cfg.remindersEnabled}
                        onChange={toggleReminders}
                      />
                    </span>
                  )}
                </div>
                <div className="wx-compose">
                  <Input
                    className="wx-compose-input"
                    placeholder="提醒事项，如「交周报」"
                    value={newText}
                    onChange={(e) => setNewText(e.target.value)}
                    onPressEnter={addReminder}
                  />
                  <DatePicker
                    showTime={{ format: 'HH:mm' }} format="YYYY-MM-DD HH:mm"
                    placeholder="触发时间"
                    value={newTime}
                    onChange={setNewTime}
                    style={{ width: 190 }}
                  />
                  <Button type="primary" icon={<SendOutlined />} loading={adding} onClick={addReminder}>创建</Button>
                </div>
                {reminders.length > 0 ? (
                  <div className="wx-rem-list">
                    {reminders.map((r) => (
                      <div key={r.id} className="wx-rem-item">
                        {reminderTag(r.status)}
                        <span className="wx-rem-text">{r.text}</span>
                        {r.source === 'weixin' && <Tag>微信下达</Tag>}
                        {r.failCount > 0 && r.status === 'pending' && <Tag color="warning">重试 {r.failCount}/5</Tag>}
                        <span className="wx-rem-time">{dayjs(r.fireAt).format('M月D日 HH:mm')}</span>
                        <button
                          type="button" className="wx-rem-del" aria-label={`删除提醒 ${r.text}`}
                          title="删除提醒" onClick={() => removeReminder(r.id)}
                        >
                          <DeleteOutlined aria-hidden="true" />
                        </button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <Typography.Text type="secondary" style={{ display: 'block', marginTop: 14, fontSize: 12 }}>
                    暂无提醒——在微信里对助手说「提醒我 30分钟后 喝水」即可创建。
                  </Typography.Text>
                )}
              </>
            )}

            {view === 'guide' && (
              <>
                <div className="wx-main-head">
                  <div>
                    <div className="wx-main-title">使用指南</div>
                    <div className="wx-main-sub">
                      在微信里给助手发消息即可触发桌面端能力；提醒到点后由桌面端经微信回推——桌面常驻，人不在电脑前也能收到。
                    </div>
                  </div>
                </div>
                <div className="wx-guide">
                  <div className="wx-guide-card">
                    <div className="wx-guide-label"><ClockCircleOutlined aria-hidden="true" /> 设提醒</div>
                    <div className="wx-cmd-row">
                      <span className="wx-cmd">提醒我 30分钟后 喝水</span>
                      <span className="wx-cmd">明天早上9点 开站会</span>
                      <span className="wx-cmd">18:30 接孩子</span>
                    </div>
                  </div>
                  <div className="wx-guide-card">
                    <div className="wx-guide-label"><NotificationOutlined aria-hidden="true" /> 收提醒</div>
                    <div className="wx-guide-desc">到点桌面端自动经微信回推（需助手在线且你至少发过一条消息）。</div>
                  </div>
                  <div className="wx-guide-card">
                    <div className="wx-guide-label"><MessageOutlined aria-hidden="true" /> 聊天</div>
                    <div className="wx-guide-desc">其他消息照常与助手对话（联网搜索可用）。</div>
                  </div>
                </div>
                <div className="wx-guide-footnote">
                  无时间表达的提醒请求会收到格式提示；「今天 + 已过时刻」会收到确认询问，不会误设。
                </div>
              </>
            )}
          </div>
        </main>
      </div>

      <Modal
        title="新增青鸟助手" open={addOpen} width={680}
        okText="下一步：扫码绑定" cancelText="取消"
        onOk={createAssistant} onCancel={() => setAddOpen(false)}
      >
        <Space direction="vertical" size={10} style={{ width: '100%', marginTop: 8 }}>
          <Input
            placeholder="助手名字（必填）" value={addName} maxLength={20}
            onChange={(e) => setAddName(e.target.value)} onPressEnter={createAssistant}
          />
          <div className="wx-pk">
            <div className="wx-pk-side">
              <Input
                size="small" allowClear prefix={<SearchOutlined aria-hidden="true" />}
                placeholder="搜索名字 / 标签" value={personaQuery}
                onChange={(e) => setPersonaQuery(e.target.value)}
              />
              <div className="wx-pk-list" role="listbox" aria-label="人格选择">
                {presetGroup.length > 0 && <div className="wx-pk-group">轻语预设</div>}
                {presetGroup.map((o) => (
                  <PersonaRow key={o.id} opt={o} active={o.id === personaSel} onPick={setPersonaSel} />
                ))}
                {charGroup.length > 0 && <div className="wx-pk-group">角色库</div>}
                {charGroup.map((o) => (
                  <PersonaRow key={o.id} opt={o} active={o.id === personaSel} onPick={setPersonaSel} />
                ))}
                {personaFiltered.length === 0 && (
                  <div className="wx-pk-empty">没有匹配的人格——可先去角色库创建可聊天角色</div>
                )}
              </div>
            </div>
            <div className="wx-pk-detail">
              {personaSelOpt ? (
                <>
                  <div className="wx-pk-portrait">
                    {personaSelOpt.portraitUrl
                      ? <img src={personaSelOpt.portraitUrl} alt={`${personaSelOpt.name} 立绘`} />
                      : <span className="wx-pk-portrait-fallback" aria-hidden="true">{personaSelOpt.name.slice(0, 1)}</span>}
                  </div>
                  <div className="wx-pk-detail-name">
                    {personaSelOpt.name}
                    <Tag color={personaSelOpt.group === 'preset' ? 'gold' : 'geekblue'}>
                      {personaSelOpt.group === 'preset' ? '轻语预设' : '角色库'}
                    </Tag>
                    {personaSelOpt.gender && (
                      <Tag>{personaSelOpt.gender === 'male' ? '男' : personaSelOpt.gender === 'female' ? '女' : personaSelOpt.gender}</Tag>
                    )}
                  </div>
                  {(personaSelOpt.tags ?? []).length > 0 && (
                    <div className="wx-pk-detail-tags">
                      {personaSelOpt.tags!.slice(0, 6).map((t) => <Tag key={t}>{t}</Tag>)}
                    </div>
                  )}
                  {personaSelOpt.desc && <div className="wx-pk-detail-desc">{personaSelOpt.desc}</div>}
                </>
              ) : (
                <div className="wx-pk-empty">左侧选择人格——详情与立绘会在这里显示</div>
              )}
            </div>
          </div>
          <Typography.Text type="secondary">
            人格 = 助手在微信里的身份：轻语预设或角色库可聊天角色（18+ 人格不列出）；选中角色的立绘会自动带出。
          </Typography.Text>
        </Space>
      </Modal>

      <Modal
        title={qrTarget ? `扫码绑定 · ${qrTarget.name || qrTarget.id}` : '扫码绑定微信'}
        open={qrOpen} footer={null} width={360}
        onCancel={() => { setQrOpen(false); setQrTarget(null) }}
      >
        <Space direction="vertical" size={14} style={{ width: '100%', textAlign: 'center', marginTop: 8 }}>
          <div className="wx-qr-steps" aria-label="绑定进度">
            {qrSteps.map((label, i) => (
              <React.Fragment key={label}>
                {i > 0 && <span className="wx-qr-step-line" aria-hidden="true" />}
                <span className={`wx-qr-step${i === qrCurrent ? ' is-current' : ''}${i < qrCurrent ? ' is-done' : ''}`}>
                  <span className="wx-qr-step-dot" aria-hidden="true" />{label}
                </span>
              </React.Fragment>
            ))}
          </div>
          <div className="wx-qr-frame">
            {qrImage ? (
              <img src={qrImage} alt="微信二维码" />
            ) : (
              <span className="wx-qr-frame-tip">加载中…</span>
            )}
          </div>
          {qrPhase === 'waiting' && <Typography.Text type="secondary">请用微信扫码</Typography.Text>}
          {qrPhase === 'scanned' && <Typography.Text type="secondary">已扫码，请在手机上确认</Typography.Text>}
          {qrPhase === 'needVerify' && (
            <>
              <Alert type="info" showIcon message="需要手机配对码" description="请在微信手机端查看配对码并输入。" />
              <Input
                placeholder="配对码" value={verifyCode}
                onChange={(e) => setVerifyCode(e.target.value)}
                addonAfter={<Button type="link" size="small">重新校验</Button>}
              />
            </>
          )}
          {qrPhase === 'confirmed' && (
            <Space direction="vertical">
              <Alert type="success" showIcon message="扫码成功" />
              <Button type="primary" loading={binding} onClick={confirmBinding}>保存绑定并启动通道</Button>
            </Space>
          )}
          {qrPhase === 'error' && <Alert type="error" showIcon message="获取二维码失败，请重试" />}
        </Space>
      </Modal>
    </div>
  )
}

/** 人格选择器行（预设/角色库通用；role=option 挂在 listbox 容器上）。 */
const PersonaRow: React.FC<{
  opt: PersonaOption
  active: boolean
  onPick: (id: string) => void
}> = ({ opt, active, onPick }) => (
  <button
    type="button"
    role="option"
    aria-selected={active}
    className={`wx-pk-item${active ? ' is-active' : ''}`}
    aria-label={`选择人格 ${opt.name}`}
    onClick={() => onPick(opt.id)}
  >
    <Avatar size={30} src={opt.portraitUrl || undefined}>{opt.name.slice(0, 1)}</Avatar>
    <span className="wx-pk-item-name">{opt.name}</span>
    <span className="wx-pk-item-kind">{opt.group === 'preset' ? '预设' : '角色库'}</span>
  </button>
)

/** 主区 · 通道详情：身份头 + 键值栅格（通道状态/微信绑定/启停）+ 操作区。 */
const ChannelDetail: React.FC<{
  row: AssistantRow
  core: boolean
  onBind: () => void
  onToggle: (v: boolean) => void
  onDelete: () => void
}> = ({ row, core, onBind, onToggle, onDelete }) => {
  const st = channelStatusOf(rowStatus(row))
  const bound = rowStatus(row).hasToken
  const name = row.name || row.id
  return (
    <>
      <div className="wx-main-head">
        <Avatar size={44} src={row.portraitUrl || undefined}>{name.slice(0, 1)}</Avatar>
        <div style={{ minWidth: 0 }}>
          <div className="wx-main-title">
            {name}
            {core && <Tag color="gold">核心</Tag>}
            <Tag color="purple">人格 {row.personalityId}</Tag>
          </div>
          <div className="wx-main-sub">
            {bound ? '微信通道已就绪：在微信里给该助手发消息即可对话与设提醒。' : '尚未绑定微信——扫码绑定后此助手即可在微信里收发消息。'}
          </div>
        </div>
        <div className="wx-detail-head-actions">
          <Button type="primary" size="small" icon={<QrcodeOutlined />} onClick={onBind}>
            {bound ? '重新绑定' : '扫码绑定'}
          </Button>
          {!core && (
            <Popconfirm
              title="删除助手"
              description={`删除「${name}」后其微信通道一并停止。`}
              okText="删除" cancelText="取消"
              onConfirm={onDelete}
            >
              <Button aria-label={`删除 ${name}`} type="text" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
        </div>
      </div>

      {/* 会话过期警示（全局：任一通道过期即提示，指回对应轨道项） */}
      {st.kind === 'expired' && (
        <Alert
          className="wx-detail-alert" type="warning" showIcon
          message="该助手微信会话已过期"
          description="重新扫码绑定即可恢复（桌面端无需重启）。"
        />
      )}

      <div className="wx-kv-grid">
        <div className="wx-kv">
          <div className="wx-kv-label">通道状态</div>
          <div className="wx-kv-value">{statusTag(rowStatus(row))}</div>
        </div>
        <div className="wx-kv">
          <div className="wx-kv-label">微信绑定</div>
          <div className="wx-kv-value">{bound ? '已绑定' : '未绑定'}</div>
        </div>
        <div className="wx-kv">
          <div className="wx-kv-label">启用</div>
          <div className="wx-kv-value">
            {core ? (
              <Tooltip title="核心助手，不可停用">
                <span>
                  <Switch size="small" aria-label={`启停 ${name}`} checked={row.enabled} disabled />
                </span>
              </Tooltip>
            ) : (
              <Switch
                size="small" aria-label={`启停 ${name}`} checked={row.enabled}
                onChange={onToggle}
              />
            )}
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {row.enabled ? '接收微信消息' : '已停用，不收发'}
            </Typography.Text>
          </div>
        </div>
      </div>

      <Typography.Text
        type="secondary" style={{ display: 'block', marginTop: 18, fontSize: 12 }}
      >
        人格 ID 可在新增后于角色库调整；绑定与启停立即生效，无需重启桌面端。
      </Typography.Text>
    </>
  )
}

export default WeixinPage
