import React, { useCallback, useEffect, useRef, useState } from 'react'
import {
  Alert, Avatar, Button, Card, DatePicker, Descriptions, Input, List, Modal, Popconfirm, Space, Switch, Tag, Tooltip, Typography, message,
} from 'antd'
import {
  QrcodeOutlined, ReloadOutlined, SendOutlined, BellOutlined, DeleteOutlined, LinkOutlined, CheckCircleOutlined, WarningOutlined, PlusOutlined,
} from '@ant-design/icons'
import dayjs, { type Dayjs } from 'dayjs'
import { app } from '../gaea/lib/bridge'
import type {
  WeixinAssistantStatusRow, WeixinAssistantView, WeixinReminderConfigView, WeixinReminderView,
} from '../gaea/lib/types'
import { isPageVisible } from '../lib/pollingGate'

/**
 * WeixinPage — 微信助手管理台（v4.4 触点·书房板块）。
 *
 * 三卡布局：连接（助手卡列表：头像/人格/通道状态徽标/启停/逐助手扫码绑定/
 * 删除 + 新增微信助手表单）/ 离线代办（提醒列表 + 手动新建 + 开关）/ 使用
 * 说明（微信文本指令示例）。页面内容层 zh 单语（i18n 决策）。数据面全部走
 * app 代理 work 面：WhisperWeixinStatus（通道状态）与 WhisperAssistantList
 * （完整字段）按 id merge 成助手行；除删除外保存一律传 List 完整对象（后端
 * 契约：空 token 字段保留现值，保存后自动重拉通道）。id='gaea' 为核心助手：
 * 禁删禁停。
 */

const POLL_MS = 5000

const statusTag = (row: WeixinAssistantStatusRow) => {
  if (!row.hasToken) return <Tag>未绑定</Tag>
  if (row.wxSessionExpired) return <Tag color="warning" icon={<WarningOutlined />}>会话过期</Tag>
  if (row.wxRunning) return <Tag color="success" icon={<CheckCircleOutlined />}>运行中</Tag>
  return <Tag color="default">已停止</Tag>
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

// 行 → 状态行投影（Status 未回时按 wxToken 兜底 hasToken，供 statusTag 复用）
const rowStatus = (row: AssistantRow): WeixinAssistantStatusRow => ({
  id: row.id, name: row.name, personalityId: row.personalityId, enabled: row.enabled,
  hasToken: row.status?.hasToken ?? Boolean(row.wxToken),
  wxRunning: row.status?.wxRunning ?? false,
  wxSessionExpired: row.status?.wxSessionExpired,
})

const WeixinPage: React.FC = () => {
  const [rows, setRows] = useState<AssistantRow[]>([])
  const [reminders, setReminders] = useState<WeixinReminderView[]>([])
  const [cfg, setCfg] = useState<WeixinReminderConfigView | null>(null)

  // ── 扫码绑定流（target 为绑定目标助手；新增流传本地暂存对象）──
  const [qrTarget, setQrTarget] = useState<WeixinAssistantView | null>(null)
  const [qrOpen, setQrOpen] = useState(false)
  const [qrImage, setQrImage] = useState('')
  const [qrToken, setQrToken] = useState('')
  const [qrPhase, setQrPhase] = useState<'idle' | 'waiting' | 'scanned' | 'needVerify' | 'confirmed' | 'error'>('idle')
  const [verifyCode, setVerifyCode] = useState('')
  const [binding, setBinding] = useState(false)
  const pollRef = useRef<number | null>(null)

  // ── 新增微信助手表单 ──
  const [addOpen, setAddOpen] = useState(false)
  const [addName, setAddName] = useState('')
  const [addPersonality, setAddPersonality] = useState('gaea')

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

  // 逐助手扫码绑定：target 为要绑定的助手（新增流传本地暂存对象）
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

  // 「新增微信助手」表单确定 → 本地暂存新助手并进入扫码流（confirmed 时 Save 落库）
  const createAssistant = () => {
    const name = addName.trim()
    if (!name) {
      message.warning('请填写助手名字')
      return
    }
    const staged: WeixinAssistantView = {
      id: `wx_${Date.now().toString(36)}`,
      name,
      personalityId: addPersonality.trim() || 'gaea',
      enabled: true,
    }
    setAddOpen(false)
    setAddName('')
    setAddPersonality('gaea')
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

  return (
    <div style={{ padding: 20, maxWidth: 860, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card size="small">
        <Space direction="vertical" size={2} style={{ width: '100%' }}>
          <Typography.Text strong>微信助手 · 离线代办遥控器</Typography.Text>
          <Typography.Text type="secondary">
            把微信变成 gaea 的遥控器：在微信里给助手发消息即可触发桌面端能力；
            提醒到点后由桌面端经微信回推——桌面常驻，人不在电脑前也能收到。
          </Typography.Text>
        </Space>
      </Card>

      <Card
        size="small"
        title={<Space><LinkOutlined />连接</Space>}
        extra={(
          <Space size={4}>
            <Button aria-label="刷新助手" type="text" size="small" icon={<ReloadOutlined />} onClick={loadAssistants} />
            <Button icon={<PlusOutlined />} type="primary" size="small" onClick={() => setAddOpen(true)}>新增微信助手</Button>
          </Space>
        )}
      >
        <List
          size="small"
          dataSource={rows}
          locale={{ emptyText: '暂无助手——点右上角「新增微信助手」创建' }}
          renderItem={(row) => {
            const core = row.id === 'gaea' // 核心助手：禁删禁停
            const bound = rowStatus(row).hasToken
            return (
              <List.Item
                actions={[
                  <Button
                    key="bind" type="text" size="small" icon={<QrcodeOutlined />}
                    onClick={() => startBinding(viewOf(row))}
                  >
                    {bound ? '重新绑定' : '扫码绑定'}
                  </Button>,
                  ...(core ? [] : [
                    <Popconfirm
                      key="del" title="删除助手"
                      description={`删除「${row.name || row.id}」后其微信通道一并停止。`}
                      okText="删除" cancelText="取消"
                      onConfirm={() => removeAssistant(row)}
                    >
                      <Button aria-label={`删除 ${row.name || row.id}`} type="text" size="small" danger icon={<DeleteOutlined />} />
                    </Popconfirm>,
                  ]),
                ]}
              >
                <Space size={12}>
                  <Avatar size={32} src={row.portraitUrl || undefined}>{(row.name || row.id).slice(0, 1)}</Avatar>
                  <Typography.Text strong>{row.name || row.id}</Typography.Text>
                  <Tag color="purple">人格 {row.personalityId}</Tag>
                  {statusTag(rowStatus(row))}
                  {core ? (
                    <Tooltip title="核心助手，不可停用">
                      <span>
                        <Switch size="small" aria-label={`启停 ${row.name || row.id}`} checked={row.enabled} disabled />
                      </span>
                    </Tooltip>
                  ) : (
                    <Switch
                      size="small" aria-label={`启停 ${row.name || row.id}`} checked={row.enabled}
                      onChange={(v) => toggleAssistant(row, v)}
                    />
                  )}
                </Space>
              </List.Item>
            )
          }}
        />
        {rows.some((r) => rowStatus(r).wxSessionExpired) && (
          <Alert
            style={{ marginTop: 8 }} type="warning" showIcon
            message="微信会话已过期"
            description="重新扫码绑定即可恢复（桌面端无需重启）。"
          />
        )}
      </Card>

      <Card
        size="small"
        title={<Space><BellOutlined />离线代办提醒</Space>}
        extra={cfg && (
          <Space size={8}>
            <Typography.Text type="secondary">到点回推微信</Typography.Text>
            <Switch checked={cfg.remindersEnabled} onChange={toggleReminders} />
          </Space>
        )}
      >
        <Space.Compact style={{ width: '100%', marginBottom: 12 }}>
          <Input
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
        </Space.Compact>
        <List
          size="small"
          dataSource={reminders}
          locale={{ emptyText: '暂无提醒——在微信里对助手说「提醒我 30分钟后 喝水」即可创建' }}
          renderItem={(r) => (
            <List.Item
              actions={[
                <Button key="del" type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => removeReminder(r.id)} />,
              ]}
            >
              <Space size={12}>
                {reminderTag(r.status)}
                <Typography.Text strong>{r.text}</Typography.Text>
                <Typography.Text type="secondary">{dayjs(r.fireAt).format('M月D日 HH:mm')}</Typography.Text>
                {r.source === 'weixin' && <Tag>微信下达</Tag>}
                {r.failCount > 0 && r.status === 'pending' && <Tag color="warning">重试 {r.failCount}/5</Tag>}
              </Space>
            </List.Item>
          )}
        />
      </Card>

      <Card size="small" title="怎么用">
        <Descriptions column={1} size="small" bordered>
          <Descriptions.Item label="设提醒">「提醒我 30分钟后 喝水」「明天早上9点 开站会」「18:30 接孩子」</Descriptions.Item>
          <Descriptions.Item label="收提醒">到点桌面端自动经微信回推（需助手在线且你至少发过一条消息）</Descriptions.Item>
          <Descriptions.Item label="聊天">其他消息照常与助手对话（联网搜索可用）</Descriptions.Item>
        </Descriptions>
        <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
          无时间表达的提醒请求会收到格式提示；「今天 + 已过时刻」会收到确认询问，不会误设。
        </Typography.Text>
      </Card>

      <Modal
        title="新增微信助手" open={addOpen} width={420}
        okText="下一步：扫码绑定" cancelText="取消"
        onOk={createAssistant} onCancel={() => setAddOpen(false)}
      >
        <Space direction="vertical" size={12} style={{ width: '100%', marginTop: 8 }}>
          <Input
            placeholder="助手名字（必填）" value={addName} maxLength={20}
            onChange={(e) => setAddName(e.target.value)} onPressEnter={createAssistant}
          />
          <Input addonBefore="人格 ID" value={addPersonality} onChange={(e) => setAddPersonality(e.target.value)} />
          <Typography.Text type="secondary">
            人格 ID 为轻语预设或角色库角色；确定后扫码绑定，扫码确认时保存并启动通道。
          </Typography.Text>
        </Space>
      </Modal>

      <Modal
        title={qrTarget ? `扫码绑定 · ${qrTarget.name || qrTarget.id}` : '扫码绑定微信'}
        open={qrOpen} footer={null} width={360}
        onCancel={() => { setQrOpen(false); setQrTarget(null) }}
      >
        <Space direction="vertical" size={12} style={{ width: '100%', textAlign: 'center' }}>
          {qrImage ? (
            <img src={qrImage} alt="微信二维码" style={{ width: 240, height: 240, margin: '0 auto', display: 'block' }} />
          ) : (
            <div style={{ width: 240, height: 240, lineHeight: '240px', margin: '0 auto', display: 'block', background: 'var(--md-sys-color-surface-variant, #f5f5f5)' }}>加载中…</div>
          )}
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

export default WeixinPage
