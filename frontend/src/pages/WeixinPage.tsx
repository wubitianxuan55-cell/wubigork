import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert, Button, DatePicker, Input, Modal, Popconfirm, Space, Switch, Tag, Tooltip, Typography, message,
} from 'antd'
import {
  QrcodeOutlined, ReloadOutlined, SendOutlined, BellOutlined, DeleteOutlined, CheckCircleOutlined, WarningOutlined, PlusOutlined,
  WechatOutlined, ClockCircleOutlined, MessageOutlined, BookOutlined, NotificationOutlined, SearchOutlined, EditOutlined,
  ApiOutlined, MobileOutlined, ArrowRightOutlined,
} from '@ant-design/icons'
import dayjs, { type Dayjs } from 'dayjs'
import { app } from '../gaea/lib/bridge'
import type { characterlib } from '../../wailsjs/go/models'
import type {
  WeixinAssistantStatusRow, WeixinAssistantView, WeixinReminderConfigView, WeixinReminderView,
} from '../gaea/lib/types'
import { usePollingGate } from '../hooks/usePollingGate'
import { useBoardActive } from '../lib/boardActive'
import { FRONTEND_EVENTS } from '../events'
import { takeWxFocusAssistant } from './wxFocus'
import { usePortraitUrl } from '../components/characterlib/usePortraitUrl'
import { ROLE_LABELS } from '../utils/theme'
import './weixin-page.css'

// WxPortraitAvatar：角色剧照头像——本地落盘路径须经 AttachmentDataURL 转
// data URL 上墙（WebView2 拒绝 file://，v4.361 观察池清账）；取数失败/无图
// 时回退到姓名首字。直接渲染受控 img，避免 Avatar 内部缓存旧 src。
const WxPortraitAvatar: React.FC<{ src?: string; name: string; size?: number }> = ({ src, name, size = 32 }) => {
  const { url } = usePortraitUrl(src)
  return (
    <span
      className="wx-avatar"
      style={{ '--wx-avatar-size': `${size}px` } as React.CSSProperties}
    >
      {url
        ? <img src={url} alt={`${name} 头像`} />
        : <span aria-hidden="true">{name.slice(0, 1)}</span>}
    </span>
  )
}

// WxPortraitImg：人格详情立绘同通道；无图/取数失败回退首字占位。
const WxPortraitImg: React.FC<{ src?: string; name: string }> = ({ src, name }) => {
  const { url } = usePortraitUrl(src)
  if (!url) return <span className="wx-pk-portrait-fallback" aria-hidden="true">{name.slice(0, 1)}</span>
  return <img src={url} alt={`${name} 立绘`} />
}

// WxPersonaStill：人物档案主剧照。直接消费角色库 portraitUrl / galleryImages；
// 内置人格无剧照时显示中性「预设形象」，自建角色无图时明确「剧照待补」。
const WxPresetPortrait: React.FC<{ gender?: string }> = ({ gender }) => {
  const female = gender === 'female'
  return (
    <svg className="wx-preset-portrait-art" viewBox="0 0 160 210" aria-hidden="true">
      <rect className="wx-preset-portrait-bg" width="160" height="210" />
      <circle className="wx-preset-portrait-ring" cx="80" cy="78" r="52" />
      <path className="wx-preset-portrait-body" d="M24 210c5-43 27-67 56-67s51 24 56 67z" />
      <path
        className="wx-preset-portrait-hair"
        d={female
          ? 'M39 76c0-31 17-51 42-51 31 0 45 24 43 60-2 32-11 52-17 58 2-27-2-44-7-54-13 9-34 11-49 2-2 16-3 30-1 52-9-9-11-34-11-54z'
          : 'M42 69c4-29 20-43 41-43 23 0 39 14 41 41-12-8-22-11-34-10-16 1-31 6-48 12z'}
      />
      <ellipse className="wx-preset-portrait-face" cx="80" cy="79" rx="33" ry="38" />
      <path className="wx-preset-portrait-neck" d="M68 108h24v23H68z" />
      <path className="wx-preset-portrait-collar" d="M48 154l32 19 32-19 10 56H38z" />
      <circle className="wx-preset-portrait-eye" cx="68" cy="79" r="2.3" />
      <circle className="wx-preset-portrait-eye" cx="92" cy="79" r="2.3" />
      <path className="wx-preset-portrait-mouth" d="M73 94c5 4 9 4 14 0" />
    </svg>
  )
}

const WxPersonaStill: React.FC<{
  src?: string
  name: string
  alt?: string
  gender?: string
  group?: PersonaOption['group']
}> = ({ src, name, alt, gender, group }) => {
  const { url } = usePortraitUrl(src)
  if (url) return <img className="wx-persona-photo-img" src={url} alt={alt || `${name} 人物剧照`} />
  return (
    <span
      className="wx-persona-photo-fallback"
      role="img"
      aria-label={`${name}${group === 'preset' ? ' 预设人物形象' : ' 人物剧照待补'}`}
    >
      <WxPresetPortrait gender={gender} />
      <small>{group === 'preset' ? '预设形象' : '剧照待补'}</small>
    </span>
  )
}

/**
 * WeixinPage — 微信助手「通讯枢纽」工作台（v4.47 星枢化重构；
 * 2026-10-31 去掉隐喻名「青鸟」，导航与页面统一为「微信助手」；
 * v4.49 通道详情「编辑」助手：改名/换人格，选择器抽成 PersonaPickerPanel
 * 供新增/编辑两流复用；v4.51 深链聚焦：角色库「创建微信助手」成功后跳转
 * 本页，自动选中该助手并直进扫码绑定，见 wxFocus.ts）。
 * 2026-10-31「通讯航路」重设计：保留全部数据与交互，将详情重构为可判读的
 * 微信 → 助手内核 → 桌面能力链路由 + 操作控制台；提醒页增加真实状态统计与
 * 下一条待办，指南改为非对称手册版式。
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

// ── 人格选项：角色库可聊天角色（内置 + 自建）的统一投影 ──
interface PersonaOption {
  id: string
  name: string
  group: 'preset' | 'character'
  gender?: string
  age?: string
  tags?: string[]
  portraitUrl?: string
  galleryImages?: string[]
  roleType?: string
  personality?: string
  background?: string
  voiceGuide?: string
  desc?: string // 选择器摘要：预设=voiceGuide / 角色=personality+background
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

const genderText = (gender?: string): string => {
  if (gender === 'male') return '男'
  if (gender === 'female') return '女'
  return gender || '未标注'
}

const personaSourceText = (persona?: PersonaOption): string => {
  if (!persona) return '角色库未匹配'
  return persona.group === 'preset' ? '角色库 · 内置' : '角色库 · 自建'
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

  // ── 新增助手表单（人格选择器直接读取角色库可聊天角色）──
  const [addOpen, setAddOpen] = useState(false)
  const [addName, setAddName] = useState('')
  const [personaOpts, setPersonaOpts] = useState<PersonaOption[]>([])
  const [personaQuery, setPersonaQuery] = useState('')
  const [personaSel, setPersonaSel] = useState('gaea')
  const [personaLoadedOnce, setPersonaLoadedOnce] = useState(false)
  const [personaLoadFailed, setPersonaLoadFailed] = useState(false)
  const personaRequestRef = useRef(0)

  // ── 编辑已有助手（改名/换人格；核心锚点 gaea 不开放编辑入口）──
  const [editOpen, setEditOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<AssistantRow | null>(null)
  const [editName, setEditName] = useState('')
  // 编辑流独立选中态：打开时预填为当前人格；自定义角色 id 不在选项里时保留
  // 原值（不重置 gaea），仅当用户点选其他选项才改变
  const [editSel, setEditSel] = useState('')

  // 人物档案与人格选择器共用角色库这一份事实源：CharacterList 同时提供
  // kind/personality/background/roleType/portraitUrl/galleryImages，不再先读
  // WhisperGetPersonalities 的窄投影再二次拼装。页面可见时拉取，返回本板块
  // 时重拉；请求序号防止快速切换时迟到响应覆盖新结果。
  const loadPersonaOptions = useCallback(async () => {
    const request = ++personaRequestRef.current
    setPersonaLoadFailed(false)
    const charRes = await app.CharacterList('', '', true, 1, 500)
      .catch(() => null as Record<string, unknown> | null)
    if (request !== personaRequestRef.current) return
    if (charRes === null) {
      setPersonaLoadedOnce(true)
      setPersonaLoadFailed(true)
      return
    }
    const items = ((charRes?.items ?? []) as characterlib.Character[])
    const personaFromCharacter = (c: characterlib.Character): PersonaOption => ({
      id: c.id,
      name: c.name,
      group: c.kind === 'custom' ? 'character' : 'preset',
      gender: c.gender || undefined,
      age: c.age,
      tags: c.tags,
      portraitUrl: c.portraitUrl,
      galleryImages: c.galleryImages,
      roleType: c.roleType,
      personality: c.personality,
      background: c.background,
      voiceGuide: c.voiceGuide,
      desc: [c.personality, c.background].filter(Boolean).join('\n'),
    })
    setPersonaOpts(items.map(personaFromCharacter))
    setPersonaLoadedOnce(true)
  }, [])

  // 弹窗打开只重置搜索与新增默认项；不再重复清空目录，避免档案短暂闪空。
  useEffect(() => {
    if (!addOpen && !editOpen) return
    setPersonaQuery('')
    if (addOpen) setPersonaSel('gaea')
  }, [addOpen, editOpen])

  // 选择器派生：搜索过滤（分组与选中项投影收进 PersonaPickerPanel）
  const personaFiltered = useMemo(() => {
    const q = personaQuery.trim().toLowerCase()
    if (!q) return personaOpts
    return personaOpts.filter((o) =>
      o.name.toLowerCase().includes(q) || (o.tags ?? []).some((t) => t.toLowerCase().includes(q)))
  }, [personaOpts, personaQuery])
  const personaById = useMemo(
    () => new Map(personaOpts.map((persona) => [persona.id, persona])),
    [personaOpts],
  )
  const selectedPersona = selected ? personaById.get(selected.personalityId) : undefined

  // ── 手动新建提醒 ──
  const [newText, setNewText] = useState('')
  const [newTime, setNewTime] = useState<Dayjs | null>(null)
  const [adding, setAdding] = useState(false)
  const [assistantsLoadFailed, setAssistantsLoadFailed] = useState(false)
  // v4.361：首拉完成标志——首拉到达前不再把「载入中」渲染成「暂无助手」空态；
  // 提醒列表同款失败可见化（照 v4.351 loadAssistants 范式）。
  const [assistantsLoadedOnce, setAssistantsLoadedOnce] = useState(false)
  const [remindersLoadFailed, setRemindersLoadFailed] = useState(false)
  const [togglingId, setTogglingId] = useState<string | null>(null)

  const loadAssistants = useCallback(async () => {
    // 两路并行：状态用 Status 行、字段用 List 行；单路失败兜底空数组，
    // 另一路照常渲染（Status 独有行仍有兜底视图）。两路全失败=读不到数据
    // 而非「没有助手」——置 loadFailed 显示错误条（v4.351：此前伪装成空态，
    // 用户误以为助手被删）。
    const [statuses, list] = await Promise.all([
      app.WhisperWeixinStatus().then(
        (v) => v,
        () => null as WeixinAssistantStatusRow[] | null,
      ),
      app.WhisperAssistantList().then(
        (v) => v,
        () => null as WeixinAssistantView[] | null,
      ),
    ])
    if (statuses === null && list === null) {
      setAssistantsLoadFailed(true)
      setAssistantsLoadedOnce(true)
      return
    }
    setAssistantsLoadFailed(false)
    setAssistantsLoadedOnce(true)
    setRows(mergeAssistantRows(list ?? [], statuses ?? []))
  }, [])

  const loadReminders = useCallback(async () => {
    // v4.361：失败可见化——此前静默吞错，失败时显示「暂无提醒」误导用户
    // 去微信创建；两路全失败置 failed 走错误条（照 loadAssistants v4.351 范式）。
    const [list, config] = await Promise.all([
      app.WeixinReminderList().then((v) => v, () => null),
      app.WeixinReminderConfig().then((v) => v, () => null),
    ])
    if (list === null && config === null) {
      setRemindersLoadFailed(true)
      return
    }
    setRemindersLoadFailed(false)
    if (list !== null) setReminders(list)
    if (config !== null) setCfg(config)
  }, [])

  // 可见时轮询助手管理数据 + 提醒列表（keepAlive 页面隐藏时空转）。
  // FE5-05：门控要判两个正交维度（lib/boardActive.ts 文件头约定「后台轮询应
  // 各自判一次」）——①窗口不可见（usePollingGate，最小化/切走）；②本板块被
  // 壳层 keepAlive 切到后台（useBoardActive('weixin')：MainLayout 对访问过的
  // 板块只 display:none，组件不卸载）。此前只判 ①，于是进过一次微信助手页后停在
  // 任意板块都会每 5s 打 WhisperWeixinStatus/WhisperAssistantList/
  // WeixinReminderList/WeixinReminderConfig。形态照 modelcenter 系轮询组件
  // （usePollingGate + useBoardActive 双闸）。
  const pollable = usePollingGate()
  const boardVisible = useBoardActive('weixin')
  useEffect(() => {
    if (!boardVisible) return
    loadPersonaOptions()
  }, [boardVisible, loadPersonaOptions])

  useEffect(() => {
    const tick = () => {
      if (!pollable || !boardVisible) return
      loadAssistants()
      loadReminders()
    }
    tick()
    const timer = window.setInterval(tick, POLL_MS)
    return () => window.clearInterval(timer)
  }, [loadAssistants, loadReminders, pollable, boardVisible])

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

  // 逐助手扫码绑定：target 为要绑定的助手（新增助手此时才落库）。useCallback：
  // 深链聚焦 effect 依赖其身份（依赖均为 setState/模块量，天然稳定）
  const startBinding = useCallback(async (target?: WeixinAssistantView) => {
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
  }, [])

  // ── v4.51 深链聚焦：角色库「创建微信助手」成功 → NAVIGATE 跳微信助手 → 自动
  // 选中新助手并直进扫码绑定。本页可能尚未挂载（lazy 首载，事件先于订阅——
  // 焦点经 sessionStorage 传递，见 wxFocus.ts），也可能 keepAlive 已挂载
  // （壳层只切 display 不重挂载）——两条路都要接得住：挂载时读一次 +
  // 常驻监听 NAVIGATE(page=weixin) 再读一次；takeWxFocusAssistant 读后即清，
  // 两路天然互斥不重复消费。（放在 loadAssistants/startBinding 定义之后：
  // effect deps 数组在渲染期求值，不能前引未初始化的 const）
  const [pendingFocus, setPendingFocus] = useState<string | null>(null)
  const focusReloadRef = useRef(false)
  useEffect(() => {
    const id = takeWxFocusAssistant()
    if (id) setPendingFocus(id)
    const onNavigate = (e: Event) => {
      if (((e as CustomEvent).detail as { page?: string } | undefined)?.page !== 'weixin') return
      const fid = takeWxFocusAssistant()
      if (fid) setPendingFocus(fid)
    }
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, onNavigate)
    return () => window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, onNavigate)
  }, [])
  // rows 落定后消费焦点：命中 → 选中该通道，未绑定 → 复用 startBinding 直接
  // 打开其扫码绑定流；已绑定只选中。keepAlive 场景 rows 可能还是创建前的旧
  // 列表（新助手要等下个轮询）——先补拉一次再判；补拉后仍无（助手已不存在）
  // → 放弃焦点保持既有选中（优雅降级）。本 effect 声明在上方选中回退 effect
  // 之后：同一次提交内后跑，覆盖回退选中的首条。
  useEffect(() => {
    if (!pendingFocus) return
    const hit = rows.find((r) => r.id === pendingFocus)
    if (hit) {
      setPendingFocus(null)
      focusReloadRef.current = false
      setView('channel')
      setSelectedId(hit.id)
      if (!rowStatus(hit).hasToken) startBinding(viewOf(hit))
      return
    }
    if (rows.length === 0) return // 首拉未归，继续等
    if (!focusReloadRef.current) {
      focusReloadRef.current = true
      loadAssistants() // 旧列表未含焦点助手 → 补拉一次
      return
    }
    setPendingFocus(null) // 补拉后仍无 → 放弃
    focusReloadRef.current = false
  }, [pendingFocus, rows, loadAssistants, startBinding])

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

  // 启停：传 List 完整对象 + 翻转后的 enabled（空 token 字段后端保留现值）。
  // v4.354：busy 闸——Switch 受控到轮询才翻转，「点了没反应」诱发连点=两条
  // 相反的 Save 并发，最终启停与显示相反。
  const toggleAssistant = async (row: AssistantRow, enabled: boolean) => {
    if (togglingId) return
    setTogglingId(row.id)
    try {
      await app.WhisperAssistantSave({ ...viewOf(row), enabled })
      message.success(enabled ? '助手已启用' : '助手已停用')
      loadAssistants()
    } catch (e) {
      message.error(`保存失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setTogglingId(null)
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

  // 「新增微信助手」表单确定 → 本地暂存新助手并进入扫码流（confirmed 时 Save 落库；
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

  // 打开「编辑助手」：名字预填现名、人格预填当前值（自定义角色 id 可能不在
  // 加载出的选项里——保留原值，详情面板显示占位提示）
  const openEdit = (row: AssistantRow) => {
    setEditTarget(row)
    setEditName(row.name || row.id)
    setEditSel(row.personalityId)
    setEditOpen(true)
  }

  // 「编辑助手」保存：以 viewOf 为底（enabled/wxToken 等原样保留，空 token
  // 字段后端按契约保留现值），叠加新名字/人格/立绘。立绘契约：仅当用户新选
  // 了带立绘的选项才覆写，否则透传原值、绝不传空串（空值后端不覆写）。
  const [savingEdit, setSavingEdit] = useState(false)
  const saveEdit = async () => {
    if (!editTarget || savingEdit) return
    const base = viewOf(editTarget)
    const sel = personaOpts.find((o) => o.id === editSel)
    setSavingEdit(true)
    try {
      await app.WhisperAssistantSave({
        ...base,
        name: editName.trim() || base.name || base.id,
        personalityId: editSel || base.personalityId,
        portraitUrl: sel?.portraitUrl || base.portraitUrl || undefined,
      })
      message.success('助手已更新')
      setEditOpen(false)
      setEditTarget(null)
      loadAssistants()
    } catch (e) {
      message.error(`保存失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setSavingEdit(false)
    }
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
  const sentCount = reminders.filter((r) => r.status === 'done').length
  const failedCount = reminders.filter((r) => r.status === 'failed').length
  const nextReminder = reminders
    .filter((r) => r.status === 'pending')
    .slice()
    .sort((a, b) => dayjs(a.fireAt).valueOf() - dayjs(b.fireAt).valueOf())[0]
  const refreshAll = () => {
    loadAssistants()
    loadReminders()
    loadPersonaOptions()
  }

  // 扫码三步指示
  const qrSteps = ['扫码', '确认', '完成']
  const qrCurrent = qrStepIndex(qrPhase)

  return (
    <div className="wx-workspace">
      <div className="wx-bg" aria-hidden="true" />
      <div className="wx-grid" aria-hidden="true" />

      {/* 顶部细条：品牌锚点 + 通道遥测 + 刷新 */}
      <header className="wx-strip">
        <span className="wx-strip-mark" aria-hidden="true"><WechatOutlined /></span>
        <span className="wx-strip-copy">
          <span className="wx-strip-title">微信助手 <em>WECHAT LINK</em></span>
          <span className="wx-strip-sub">微信触达桌面能力，提醒到点回推</span>
        </span>
        <span className="wx-strip-meta" role="status">
          <span className="wx-strip-stat"><strong>{rows.length}</strong> 通道</span>
          <span className="wx-strip-stat"><i className={runningCount > 0 ? 'is-live' : ''} aria-hidden="true" /><strong>{runningCount}</strong> 运行</span>
          <span className="wx-strip-stat"><strong>{pendingCount}</strong> 待提醒</span>
          {cfg && <span className="wx-strip-switch">回推 {cfg.remindersEnabled ? '开' : '关'}</span>}
        </span>
        <button type="button" className="wx-refresh-btn" aria-label="刷新微信助手数据" title="刷新助手、提醒与角色库" onClick={refreshAll}>
          <ReloadOutlined aria-hidden="true" />
        </button>
      </header>

      <div className="wx-body">
        {/* 左：通道轨道（助手清单即导航；支撑组挂提醒/指南） */}
        <aside className="wx-rail v3-panel" aria-label="微信助手导航">
          <div className="v3-panel-head wx-rail-head">
            <span className="wx-rail-heading">
              <span className="v3-panel-title">通讯通道</span>
              <span className="wx-rail-count">{rows.length} 个连接</span>
            </span>
            <span className="v3-panel-spacer" />
            <button
              type="button" className="wx-refresh-btn" aria-label="新增助手" title="新增微信助手"
              onClick={() => setAddOpen(true)}
            >
              <PlusOutlined aria-hidden="true" />
            </button>
          </div>
          <nav className="wx-rail-nav">
            {rows.map((row) => {
              const st = channelStatusOf(rowStatus(row))
              const active = view === 'channel' && row.id === selectedId
              const persona = personaById.get(row.personalityId)
              return (
                <button
                  key={row.id}
                  type="button"
                  className={`wx-rail-item wx-rail-ch${active ? ' is-active' : ''}`}
                  aria-label={`通道 ${row.name || row.id}`}
                  aria-current={active ? 'page' : undefined}
                  title={`${row.name || row.id} · ${persona?.name || row.personalityId} · ${st.text}`}
                  onClick={() => { setSelectedId(row.id); setView('channel') }}
                >
                  <WxPortraitAvatar
                    src={persona?.portraitUrl || persona?.galleryImages?.[0] || row.portraitUrl}
                    name={row.name || row.id}
                  />
                  <span className="wx-rail-copy">
                    <span className="wx-rail-name">{row.name || row.id}</span>
                    <span className="wx-rail-persona">
                      {persona?.name || `人格 ${row.personalityId}`}
                      {persona ? ` · ${personaSourceText(persona)}` : ''}
                    </span>
                    <span className="wx-rail-status">{st.text}<span className={`wx-dot is-${st.kind}`} aria-hidden="true" /></span>
                  </span>
                  <ArrowRightOutlined className="wx-rail-arrow" aria-hidden="true" />
                </button>
              )
            })}
            {rows.length === 0 && !assistantsLoadFailed && assistantsLoadedOnce && (
              <div className="wx-rail-empty">暂无微信助手——点上方 + 新增</div>
            )}
            {rows.length === 0 && !assistantsLoadFailed && !assistantsLoadedOnce && (
              <div className="wx-rail-empty" role="status">正在载入助手…</div>
            )}
            {assistantsLoadFailed && (
              <div className="wx-rail-empty" role="status">助手列表载入失败，稍后自动重试</div>
            )}
            <button type="button" className="wx-rail-item wx-rail-add" onClick={() => setAddOpen(true)}>
              <PlusOutlined aria-hidden="true" />
              <span className="wx-rail-name">新增微信助手</span>
            </button>

            <div className="wx-rail-group">工作台</div>
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
                  persona={selectedPersona}
                  personaLoading={!personaLoadedOnce}
                  personaLoadFailed={personaLoadFailed}
                  onBind={() => startBinding(viewOf(selected))}
                  onToggle={(v) => void toggleAssistant(selected, v)}
                  toggling={togglingId === selected.id}
                  onDelete={() => removeAssistant(selected)}
                  onEdit={() => openEdit(selected)}
                  onOpenReminders={() => setView('reminders')}
                  onOpenGuide={() => setView('guide')}
                />
              ) : assistantsLoadedOnce ? (
                <div className="wx-empty">
                  <span className="wx-empty-icon"><WechatOutlined aria-hidden="true" /></span>
                  <span className="wx-empty-title">还没有微信助手</span>
                  <span className="wx-empty-hint">
                    新增助手并扫码绑定微信后，即可在微信里与 gaea 对话、设提醒、收推送。
                  </span>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>新增微信助手</Button>
                </div>
              ) : (
                <div className="wx-empty" role="status">
                  <span className="wx-empty-title">正在载入助手…</span>
                </div>
              )
            )}

            {view === 'reminders' && (
              <>
                <div className="wx-view-head">
                  <div className="wx-view-heading">
                    <div className="wx-view-kicker">异步任务</div>
                    <div className="wx-main-title">离线代办提醒</div>
                    <div className="wx-main-sub">到点由桌面端经微信回推——桌面常驻，人不在电脑前也能收到。</div>
                  </div>
                  {cfg && (
                    <span className="wx-rem-config">
                      <span className="wx-rem-config-copy">
                        <strong>到点回推微信</strong>
                        <small>{cfg.remindersEnabled ? '回推服务正在监听' : '回推服务已暂停'}</small>
                      </span>
                      <Switch
                        size="small" aria-label="到点回推微信" checked={cfg.remindersEnabled}
                        onChange={toggleReminders}
                      />
                    </span>
                  )}
                </div>
                <div className="wx-rem-overview" aria-label="提醒概览">
                  <div className="wx-rem-metric is-primary">
                    <span>等待触达</span>
                    <strong>{pendingCount}</strong>
                    <small>{nextReminder ? `最近 ${dayjs(nextReminder.fireAt).format('M月D日 HH:mm')}` : '暂无排期'}</small>
                  </div>
                  <div className="wx-rem-metric">
                    <span>已送微信</span>
                    <strong>{sentCount}</strong>
                    <small>桌面端回推</small>
                  </div>
                  <div className={`wx-rem-metric${failedCount > 0 ? ' is-warning' : ''}`}>
                    <span>需要处理</span>
                    <strong>{failedCount}</strong>
                    <small>{failedCount > 0 ? '请稍后重试' : '当前无异常'}</small>
                  </div>
                  <div className="wx-next-reminder">
                    <span className="wx-next-label"><ClockCircleOutlined aria-hidden="true" /> 下一条提醒</span>
                    <strong>{nextReminder ? `待办 · ${nextReminder.text}` : '还没有安排'}</strong>
                    <span>{nextReminder ? dayjs(nextReminder.fireAt).format('M月D日 HH:mm') : '在下方创建一条，或直接在微信里告诉助手'}</span>
                  </div>
                </div>
                <div className="wx-compose-card">
                  <div className="wx-compose-head">
                    <span><PlusOutlined aria-hidden="true" /> 快速添加</span>
                    <small>创建后会出现在右侧时间线</small>
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
                </div>
                {reminders.length > 0 ? (
                  <div className="wx-rem-list">
                    {reminders.map((r) => (
                      <div key={r.id} className="wx-rem-item">
                        <span className={`wx-rem-signal is-${r.status}`} aria-hidden="true" />
                        <div className="wx-rem-main">
                          <div className="wx-rem-title-row">
                            <span className="wx-rem-text">{r.text}</span>
                            {reminderTag(r.status)}
                          </div>
                          <div className="wx-rem-meta">
                            {r.source === 'weixin' ? <Tag>微信下达</Tag> : <Tag>桌面创建</Tag>}
                            {r.failCount > 0 && r.status === 'pending' && <Tag color="warning">重试 {r.failCount}/5</Tag>}
                          </div>
                        </div>
                        <time className="wx-rem-time" dateTime={r.fireAt}>{dayjs(r.fireAt).format('M月D日 HH:mm')}</time>
                        <Popconfirm
                          title="删除这条提醒？"
                          description="已下达微信的提醒删除后无法撤回"
                          okText="删除"
                          cancelText="取消"
                          okButtonProps={{ danger: true, 'data-testid': 'wx-rem-del-confirm' }}
                          onConfirm={() => void removeReminder(r.id)}
                        >
                          <button
                            type="button" className="wx-rem-del" aria-label={`删除提醒 ${r.text}`}
                            title="删除提醒"
                          >
                            <DeleteOutlined aria-hidden="true" />
                          </button>
                        </Popconfirm>
                      </div>
                    ))}
                  </div>
                ) : remindersLoadFailed ? (
                  <Typography.Text type="warning" role="status" className="wx-rem-feedback" data-testid="wx-reminders-failed">
                    提醒列表载入失败，稍后自动重试。
                  </Typography.Text>
                ) : (
                  <div className="wx-rem-empty">
                    <span className="wx-rem-empty-icon"><BellOutlined aria-hidden="true" /></span>
                    <span className="wx-rem-empty-copy">
                      <strong>暂无提醒</strong>
                      <small>在微信里对助手说「提醒我 30分钟后 喝水」即可创建。</small>
                    </span>
                  </div>
                )}
              </>
            )}

            {view === 'guide' && (
              <>
                <div className="wx-view-head">
                  <div className="wx-view-heading">
                    <div className="wx-view-kicker">操作手册</div>
                    <div className="wx-main-title">使用指南</div>
                    <div className="wx-main-sub">
                      在微信里给助手发消息即可触发桌面端能力；提醒到点后由桌面端经微信回推——桌面常驻，人不在电脑前也能收到。
                    </div>
                  </div>
                  <div className="wx-guide-live" role="status">
                    <span className={`wx-dot ${runningCount > 0 ? 'is-running' : 'is-stopped'}`} aria-hidden="true" />
                    <span>
                      <strong>{runningCount > 0 ? '桌面链路在线' : '等待通道运行'}</strong>
                      <small>{runningCount > 0 ? `${runningCount} 个助手可回推` : '绑定并启用助手后可用'}</small>
                    </span>
                  </div>
                </div>
                <div className="wx-guide">
                  <article className="wx-guide-card wx-guide-card-featured">
                    <div className="wx-guide-step">01</div>
                    <div className="wx-guide-label"><ClockCircleOutlined aria-hidden="true" /> 设提醒</div>
                    <div className="wx-guide-desc">直接说清事项和时间，助手会把它排进回推队列。</div>
                    <div className="wx-cmd-row">
                      <span className="wx-cmd">提醒我 30分钟后 喝水</span>
                      <span className="wx-cmd">明天早上9点 开站会</span>
                      <span className="wx-cmd">18:30 接孩子</span>
                    </div>
                  </article>
                  <article className="wx-guide-card">
                    <div className="wx-guide-step">02</div>
                    <div className="wx-guide-label"><NotificationOutlined aria-hidden="true" /> 收提醒</div>
                    <div className="wx-guide-desc">到点桌面端自动经微信回推（需助手在线且你至少发过一条消息）。</div>
                  </article>
                  <article className="wx-guide-card">
                    <div className="wx-guide-step">03</div>
                    <div className="wx-guide-label"><MessageOutlined aria-hidden="true" /> 聊天</div>
                    <div className="wx-guide-desc">其他消息照常与助手对话（联网搜索可用）。</div>
                  </article>
                </div>
                <div className="wx-guide-footnote">
                  <WarningOutlined aria-hidden="true" />
                  <span>无时间表达的提醒请求会收到格式提示；「今天 + 已过时刻」会收到确认询问，不会误设。</span>
                </div>
              </>
            )}
          </div>
        </main>
      </div>

      <Modal
        title="新增微信助手" open={addOpen} width={680}
        okText="下一步：扫码绑定" cancelText="取消"
        onOk={createAssistant} onCancel={() => setAddOpen(false)}
      >
        <Space direction="vertical" size={10} style={{ width: '100%', marginTop: 8 }}>
          <Input
            placeholder="助手名字（必填）" value={addName} maxLength={20}
            onChange={(e) => setAddName(e.target.value)} onPressEnter={createAssistant}
          />
          <PersonaPickerPanel
            opts={personaFiltered} query={personaQuery} sel={personaSel}
            onQuery={setPersonaQuery} onPick={setPersonaSel}
          />
          <Typography.Text type="secondary">
            人格 = 助手在微信里的身份，直接使用角色库名称、设定与剧照；内置人格和自建角色都可选。
          </Typography.Text>
        </Space>
      </Modal>

      {/* 编辑已有助手：改名/换人格（复用人格选择器；保存不改绑定与启停） */}
      <Modal
        title={editTarget ? `编辑助手 · ${editTarget.name || editTarget.id}` : '编辑助手'}
        open={editOpen} width={680}
        okText="保存修改" cancelText="取消"
        confirmLoading={savingEdit} onOk={saveEdit} onCancel={() => { if (!savingEdit) setEditOpen(false) }}
      >
        <Space direction="vertical" size={10} style={{ width: '100%', marginTop: 8 }}>
          <Input
            placeholder="助手名字" value={editName} maxLength={20}
            onChange={(e) => setEditName(e.target.value)}
          />
          <PersonaPickerPanel
            opts={personaFiltered} query={personaQuery} sel={editSel}
            onQuery={setPersonaQuery} onPick={setEditSel}
          />
          <Typography.Text type="secondary">
            修改名字或人格后保存即生效；微信绑定与启停状态不受影响（选中带立绘的角色时才更新立绘）。
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

/** 人格选择器行（角色库内置/自建通用；role=option 挂在 listbox 容器上）。 */
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
    <WxPortraitAvatar src={opt.portraitUrl || opt.galleryImages?.[0]} name={opt.name} size={30} />
    <span className="wx-pk-item-name">{opt.name}</span>
    <span className="wx-pk-item-kind">{opt.group === 'preset' ? '内置' : '自建'}</span>
  </button>
)

/**
 * 人格选择器双栏面板（新增/编辑助手弹窗共用）：左 = 搜索 + 角色库分组列表
 * （内置人格/自建角色），右 = 详情预览。选中态与回调由调用方持有——新增流传
 * personaSel，编辑流传 editSel（打开时预填当前人格；不在选项里时保留原值，
 * 详情面板仅显示占位提示）。
 */
const PersonaPickerPanel: React.FC<{
  opts: PersonaOption[]
  query: string
  sel: string
  onQuery: (q: string) => void
  onPick: (id: string) => void
}> = ({ opts, query, sel, onQuery, onPick }) => {
  const presetGroup = opts.filter((o) => o.group === 'preset')
  const charGroup = opts.filter((o) => o.group === 'character')
  const selOpt = opts.find((o) => o.id === sel) ?? null
  return (
    <div className="wx-pk">
      <div className="wx-pk-side">
        <Input
          size="small" allowClear prefix={<SearchOutlined aria-hidden="true" />}
          placeholder="搜索名字 / 标签" value={query}
          onChange={(e) => onQuery(e.target.value)}
        />
        <div className="wx-pk-list" role="listbox" aria-label="人格选择">
          {presetGroup.length > 0 && <div className="wx-pk-group">角色库 · 内置人格</div>}
          {presetGroup.map((o) => (
            <PersonaRow key={o.id} opt={o} active={o.id === sel} onPick={onPick} />
          ))}
          {charGroup.length > 0 && <div className="wx-pk-group">角色库 · 自建角色</div>}
          {charGroup.map((o) => (
            <PersonaRow key={o.id} opt={o} active={o.id === sel} onPick={onPick} />
          ))}
          {opts.length === 0 && (
            <div className="wx-pk-empty">没有匹配的角色——可先去角色库创建可聊天角色</div>
          )}
        </div>
      </div>
      <div className="wx-pk-detail">
        {selOpt ? (
          <>
            <div className="wx-pk-portrait">
              <WxPortraitImg src={selOpt.portraitUrl || selOpt.galleryImages?.[0]} name={selOpt.name} />
            </div>
            <div className="wx-pk-detail-name">
              {selOpt.name}
              <Tag color={selOpt.group === 'preset' ? 'gold' : 'geekblue'}>
                {selOpt.group === 'preset' ? '内置人格' : '自建角色'}
              </Tag>
              {selOpt.gender && (
                <Tag>{selOpt.gender === 'male' ? '男' : selOpt.gender === 'female' ? '女' : selOpt.gender}</Tag>
              )}
            </div>
            {(selOpt.tags ?? []).length > 0 && (
              <div className="wx-pk-detail-tags">
                {selOpt.tags!.slice(0, 6).map((t) => <Tag key={t}>{t}</Tag>)}
              </div>
            )}
            {selOpt.desc && <div className="wx-pk-detail-desc">{selOpt.desc}</div>}
          </>
        ) : sel && opts.length > 0 ? (
          // 编辑流：当前人格不在可选清单（自定义角色/已下架）——保留原值不重置
          <div className="wx-pk-empty">当前人格不在可选列表中——将保留原设置；点选左侧其他人格可替换</div>
        ) : (
          <div className="wx-pk-empty">左侧选择人格——详情与立绘会在这里显示</div>
        )}
      </div>
    </div>
  )
}

/** 主区 · 通道详情：身份头 + 通讯链路由 + 操作控制台 + 使用入口。 */
const ChannelDetail: React.FC<{
  row: AssistantRow
  core: boolean
  persona?: PersonaOption
  personaLoading: boolean
  personaLoadFailed: boolean
  onBind: () => void
  onToggle: (v: boolean) => void
  toggling?: boolean
  onDelete: () => void
  onEdit: () => void
  onOpenReminders: () => void
  onOpenGuide: () => void
}> = ({
  row, core, persona, personaLoading, personaLoadFailed,
  onBind, onToggle, toggling, onDelete, onEdit, onOpenReminders, onOpenGuide,
}) => {
  const st = channelStatusOf(rowStatus(row))
  const bound = rowStatus(row).hasToken
  const name = row.name || row.id
  const personaName = persona?.name || `人格 ${row.personalityId}`
  const personaPortraitSrc = persona?.portraitUrl || persona?.galleryImages?.[0] || row.portraitUrl || ''
  const personaGallery = Array.from(new Set((persona?.galleryImages ?? []).filter(Boolean)))
    .filter((src) => src !== personaPortraitSrc)
    .slice(0, 4)
  const personaSource = personaSourceText(persona)
  const personaDescription = persona?.personality || persona?.voiceGuide || persona?.desc || ''
  const personaVoiceGuide = persona?.personality && persona.voiceGuide ? persona.voiceGuide : ''
  const personaBackground = persona?.background
    || (persona?.group === 'preset' ? '轻语内置人格，不绑定具体人物背景，主要服务于当前微信工作场景。' : '')
  const personaFallback = personaLoading
    ? '正在读取人物档案…'
    : personaLoadFailed
      ? '人物档案载入失败，稍后切换板块或刷新页面重试。'
      : '未找到该人格的详细资料；可在编辑助手时重新选择人格。'
  const personaRole = persona?.roleType ? (ROLE_LABELS[persona.roleType] || persona.roleType) : ''
  const personaTags = Array.from(new Set((persona?.tags ?? []).filter(Boolean)))
  return (
    <>
      <div className="wx-channel-hero">
        <div className="wx-identity-block">
          <span className="wx-avatar-halo">
            <WxPortraitAvatar
              src={personaPortraitSrc}
              name={name}
              size={52}
            />
            <i className={`wx-avatar-status is-${st.kind}`} aria-hidden="true" />
          </span>
          <div className="wx-identity-copy">
            <div className="wx-view-kicker">当前通道 · {bound ? '已接入微信' : '等待微信接入'}</div>
            <div className="wx-main-title">
              {name}
              {core && <Tag className="wx-tag-core">核心</Tag>}
              <Tag className="wx-tag-persona">人格 {row.personalityId}</Tag>
            </div>
            <div className="wx-main-sub">
              {bound ? '微信通道已就绪：发消息即可对话与设提醒，通道状态由桌面端实时维护。' : '尚未绑定微信——扫码绑定后，此助手即可在微信里收发消息。'}
            </div>
          </div>
        </div>
        <div className="wx-detail-head-actions">
          {!core && (
            <>
              <Button size="small" icon={<EditOutlined />} aria-label={`编辑 ${name}`} onClick={onEdit}>
                编辑
              </Button>
              <Popconfirm
                title="删除助手"
                description={`删除「${name}」后其微信通道一并停止。`}
                okText="删除" cancelText="取消"
                onConfirm={onDelete}
              >
                <Button aria-label={`删除 ${name}`} type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>
            </>
          )}
        </div>
      </div>

      {/* 会话过期警示（全局：任一通道过期即提示，指回对应轨道项） */}
      {st.kind === 'expired' && (
        <Alert
          className="wx-detail-alert" type="warning" showIcon
          message="该助手微信会话已过期"
          description="重新扫码绑定即可恢复（桌面端无需重启）。"
          action={<Button size="small" onClick={onBind}>重新扫码</Button>}
        />
      )}

      <section
        className={`wx-persona-dossier${persona ? '' : ' is-unresolved'}`}
        aria-label={`人物档案 ${personaName}`}
      >
        <div className="wx-persona-visual">
          <span className="wx-persona-orbit" aria-hidden="true" />
          <div className="wx-persona-photo-frame">
            <WxPersonaStill
              src={personaPortraitSrc}
              name={personaName}
              gender={persona?.gender}
              group={persona?.group}
            />
          </div>
          <span className="wx-persona-source">{personaSource}</span>
        </div>
        <div className="wx-persona-body">
          <div className="wx-persona-head">
            <div className="wx-persona-title-wrap">
              <div className="wx-view-kicker">人格档案 · {personaSource}</div>
              <div className="wx-persona-name">{personaName}</div>
              <div className="wx-persona-serving">
                当前作为「{name}」在微信里与你对话
              </div>
            </div>
            {persona && (
              <div className="wx-persona-facts">
                <span><small>性别</small><strong>{genderText(persona.gender)}</strong></span>
                {personaRole && <span><small>人物定位</small><strong>{personaRole}</strong></span>}
                {persona.age && <span><small>年龄</small><strong>{persona.age}</strong></span>}
              </div>
            )}
          </div>
          <div className="wx-persona-tags">
            <Tag className="wx-persona-kind">{personaSource}</Tag>
            {personaTags.map((tag) => <Tag key={tag}>{tag}</Tag>)}
            {!personaLoading && !personaLoadFailed && !persona && (
              <Tag className="wx-persona-kind">人格 ID · {row.personalityId}</Tag>
            )}
          </div>
          {personaGallery.length > 0 && (
            <div className="wx-persona-stills" aria-label={`${personaName} 其他剧照`}>
              <span className="wx-persona-stills-label">其他剧照</span>
              {personaGallery.map((src, index) => (
                <span className="wx-persona-still" key={src}>
                  <WxPersonaStill
                    src={src}
                    name={personaName}
                    alt={`${personaName} 人物剧照 ${index + 2}`}
                  />
                </span>
              ))}
            </div>
          )}
          <div className="wx-persona-copy">
            <div className="wx-persona-field">
              <span>人格设定</span>
              <p>{persona ? (personaDescription || '暂未填写人格说明。') : personaFallback}</p>
            </div>
            {persona && (
              <div className="wx-persona-field">
                <span>人物背景</span>
                <p>{personaBackground || '暂未填写人物背景。'}</p>
              </div>
            )}
            {personaVoiceGuide && (
              <div className="wx-persona-field">
                <span>沟通口吻</span>
                <p>{personaVoiceGuide}</p>
              </div>
            )}
          </div>
        </div>
      </section>

      <div className="wx-channel-grid">
        <section className="wx-route-card" aria-label="通讯链路状态">
          <div className="wx-section-head">
            <div>
              <div className="wx-view-kicker">实时链路由</div>
              <div className="wx-section-title">微信 ↔ 助手内核 ↔ 桌面</div>
            </div>
            <span className={`wx-status-pill is-${st.kind}`}>
              <i aria-hidden="true" />
              {st.text}
            </span>
          </div>

          <div className={`wx-signal-map is-${st.kind}`} aria-label={`通讯链路：${st.text}`}>
            <div className="wx-signal-node">
              <span className="wx-signal-icon"><MobileOutlined aria-hidden="true" /></span>
              <strong>微信端</strong>
              <small>{bound ? '身份已绑定' : '等待扫码'}</small>
            </div>
            <div className="wx-signal-link">
              <span className="wx-signal-track" />
              <span className="wx-signal-packet" aria-hidden="true" />
            </div>
            <div className="wx-signal-node is-hub">
              <span className="wx-signal-icon"><WechatOutlined aria-hidden="true" /></span>
              <strong>助手内核</strong>
              <small>{st.text}</small>
            </div>
            <div className="wx-signal-link">
              <span className="wx-signal-track" />
              <span className="wx-signal-packet" aria-hidden="true" />
            </div>
            <div className="wx-signal-node">
              <span className="wx-signal-icon"><ApiOutlined aria-hidden="true" /></span>
              <strong>桌面能力</strong>
              <small>{row.enabled ? '常驻待命' : '已暂停'}</small>
            </div>
          </div>

          <div className="wx-route-facts">
            <div>
              <span>消息范围</span>
              <strong>私聊对话</strong>
            </div>
            <div>
              <span>触发能力</span>
              <strong>提醒 · 搜索</strong>
            </div>
            <div>
              <span>运行依赖</span>
              <strong>桌面端常驻</strong>
            </div>
          </div>
        </section>

        <aside className="wx-channel-control" aria-label="通道控制">
          <div className="wx-section-head">
            <div>
              <div className="wx-view-kicker">控制台</div>
              <div className="wx-section-title">通道控制</div>
            </div>
            {statusTag(rowStatus(row))}
          </div>
          <div className="wx-control-power">
            <span>
              <strong>接收微信消息</strong>
              <small>{row.enabled ? '新消息会送入助手，并触发桌面能力' : '已停用，不收发微信消息'}</small>
            </span>
            {core ? (
              <Tooltip title="核心助手，不可停用">
                <span>
                  <Switch size="small" aria-label={`启停 ${name}`} checked={row.enabled} disabled />
                </span>
              </Tooltip>
            ) : (
              <Switch
                size="small" aria-label={`启停 ${name}`} checked={row.enabled}
                loading={toggling}
                onChange={onToggle}
              />
            )}
          </div>
          <div className="wx-control-bind">
            <span>
              <small>微信身份</small>
              <strong>{bound ? '已安全绑定' : '尚未绑定'}</strong>
            </span>
            <Button
              type={bound ? 'default' : 'primary'}
              size="small" icon={<QrcodeOutlined />} onClick={onBind}
            >
              {bound ? '重新绑定' : '扫码绑定'}
            </Button>
          </div>
          <div className="wx-control-lines">
            <span>绑定方式 <strong>微信扫码</strong></span>
            <span>生效时间 <strong>立即生效</strong></span>
          </div>
        </aside>
      </div>

      <section className="wx-playbook" aria-label="微信助手使用入口">
        <div className="wx-section-head">
          <div>
            <div className="wx-view-kicker">下一步</div>
            <div className="wx-section-title">把微信变成快捷指令入口</div>
          </div>
        </div>
        <div className="wx-playbook-grid">
          <button type="button" className="wx-playbook-item is-primary" onClick={onOpenReminders}>
            <span className="wx-playbook-icon"><ClockCircleOutlined aria-hidden="true" /></span>
            <span className="wx-playbook-copy">
              <strong>让微信助手到点提醒你</strong>
              <small>“提醒我 30分钟后 喝水”</small>
            </span>
            <ArrowRightOutlined className="wx-playbook-arrow" aria-hidden="true" />
          </button>
          <button type="button" className="wx-playbook-item" onClick={onOpenGuide}>
            <span className="wx-playbook-icon"><BookOutlined aria-hidden="true" /></span>
            <span className="wx-playbook-copy">
              <strong>查看完整使用指南</strong>
              <small>提醒、回推与聊天边界</small>
            </span>
            <ArrowRightOutlined className="wx-playbook-arrow" aria-hidden="true" />
          </button>
        </div>
      </section>
    </>
  )
}

export default WeixinPage
