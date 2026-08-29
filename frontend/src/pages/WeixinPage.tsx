import React, { useCallback, useEffect, useRef, useState } from 'react'
import {
  Alert, Button, Card, DatePicker, Descriptions, Input, List, Modal, Space, Switch, Tag, Typography, message,
} from 'antd'
import {
  QrcodeOutlined, ReloadOutlined, SendOutlined, BellOutlined, DeleteOutlined, LinkOutlined, CheckCircleOutlined, WarningOutlined,
} from '@ant-design/icons'
import dayjs, { type Dayjs } from 'dayjs'
import { app } from '../gaea/lib/bridge'
import type {
  WeixinAssistantStatusRow, WeixinReminderConfigView, WeixinReminderView,
} from '../gaea/lib/types'
import { isPageVisible } from '../lib/pollingGate'

/**
 * WeixinPage — 微信助手（v4.4 触点·书房板块）。
 *
 * 三卡布局：连接（扫码绑定 + 通道状态）/ 离线代办（提醒列表 + 手动新建 +
 * 开关）/ 使用说明（微信文本指令示例）。页面内容层 zh 单语（i18n 决策）。
 * 数据面全部走 app 代理 work 面（WhisperWeixin* / WeixinReminder* /
 * WhisperAssistant*）。
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

const WeixinPage: React.FC = () => {
  const [assistants, setAssistants] = useState<WeixinAssistantStatusRow[]>([])
  const [reminders, setReminders] = useState<WeixinReminderView[]>([])
  const [cfg, setCfg] = useState<WeixinReminderConfigView | null>(null)

  // ── 扫码绑定流 ──
  const [qrOpen, setQrOpen] = useState(false)
  const [qrImage, setQrImage] = useState('')
  const [qrToken, setQrToken] = useState('')
  const [qrPhase, setQrPhase] = useState<'idle' | 'waiting' | 'scanned' | 'needVerify' | 'confirmed' | 'error'>('idle')
  const [verifyCode, setVerifyCode] = useState('')
  const [binding, setBinding] = useState(false)
  const pollRef = useRef<number | null>(null)

  // ── 手动新建提醒 ──
  const [newText, setNewText] = useState('')
  const [newTime, setNewTime] = useState<Dayjs | null>(null)
  const [adding, setAdding] = useState(false)

  const loadStatus = useCallback(async () => {
    try {
      setAssistants(await app.WhisperWeixinStatus())
    } catch { /* 后端未就绪时静默 */ }
  }, [])

  const loadReminders = useCallback(async () => {
    try {
      const [list, config] = await Promise.all([app.WeixinReminderList(), app.WeixinReminderConfig()])
      setReminders(list)
      setCfg(config)
    } catch { /* 静默 */ }
  }, [])

  // 可见时轮询通道状态 + 提醒列表（keepAlive 页面隐藏时空转）
  useEffect(() => {
    loadStatus()
    loadReminders()
    const timer = window.setInterval(() => {
      if (!isPageVisible()) return
      loadStatus()
      loadReminders()
    }, POLL_MS)
    return () => window.clearInterval(timer)
  }, [loadStatus, loadReminders])

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

  const startBinding = async () => {
    setQrOpen(true)
    setQrPhase('waiting')
    setQrToken('')
    setVerifyCode('')
    try {
      const qr = await app.WhisperWeixinGetQR()
      setQrImage(qr.imageUrl)
      setQrToken(qr.qrcode)
    } catch (e) {
      setQrPhase('error')
      message.error(`获取二维码失败：${e instanceof Error ? e.message : String(e)}`)
    }
  }

  // confirmed → 更新核心助手 gaea 的微信绑定（upsert，后端自动重拉通道）
  const confirmBinding = async () => {
    setBinding(true)
    try {
      // WhisperWeixinQRStatus(confirmed) 已含 botToken/botId/userId，重取一次快照
      const res = await app.WhisperWeixinQRStatus(qrToken)
      await app.WhisperAssistantSave({
        id: 'gaea', name: 'gaea', personalityId: 'gaea', enabled: true,
        wxToken: String(res.botToken ?? ''), wxBotId: String(res.botId ?? ''), wxUserId: String(res.userId ?? ''),
      })
      message.success('微信绑定已保存，通道已重启')
      setQrOpen(false)
      loadStatus()
    } catch (e) {
      message.error(`保存绑定失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setBinding(false)
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
        extra={<Button icon={<QrcodeOutlined />} type="primary" size="small" onClick={startBinding}>扫码绑定微信</Button>}
      >
        <List
          size="small"
          dataSource={assistants}
          locale={{ emptyText: '暂无助手（扫码绑定后自动创建）' }}
          renderItem={(row) => (
            <List.Item
              actions={[<Button key="refresh" type="text" size="small" icon={<ReloadOutlined />} onClick={loadStatus} />]}
            >
              <Space size={12}>
                <Typography.Text strong>{row.name || row.id}</Typography.Text>
                <Typography.Text type="secondary">人格 {row.personalityId}</Typography.Text>
                {statusTag(row)}
              </Space>
            </List.Item>
          )}
        />
        {assistants.some((r) => r.wxSessionExpired) && (
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
        title="扫码绑定微信" open={qrOpen} footer={null} width={360}
        onCancel={() => setQrOpen(false)}
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
