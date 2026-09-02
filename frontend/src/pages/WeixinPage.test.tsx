// WeixinPage.test.tsx — 微信助手管理台测试。
// 桥接经 vi.mock('../gaea/lib/bridge') 注入确定性数据（app.* 方法逐个 mock），
// 覆盖：①多助手列表渲染（状态徽标/人格 Tag/头像回退）②启停翻转调用 Save 且
// enabled 正确（携带完整对象）③gaea 核心助手禁删禁停 ④删除 Popconfirm →
// WhisperAssistantDelete ⑤新增微信助手表单 → 本地暂存 → 扫码流确认后 Save。

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'

// 屏蔽 bridge seam（vi.hoisted 避免 mock 提升导致的初始化顺序问题）
const mocks = vi.hoisted(() => ({
  WhisperWeixinStatus: vi.fn(),
  WhisperAssistantList: vi.fn(),
  WhisperAssistantSave: vi.fn(),
  WhisperAssistantDelete: vi.fn(),
  WhisperWeixinGetQR: vi.fn(),
  WhisperWeixinQRStatus: vi.fn(),
  WhisperWeixinQRStatusWithCode: vi.fn(),
  WeixinReminderList: vi.fn(),
  WeixinReminderConfig: vi.fn(),
  WeixinReminderAdd: vi.fn(),
  WeixinReminderDelete: vi.fn(),
  WeixinReminderSetConfig: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', () => ({ app: mocks }))

import WeixinPage from './WeixinPage'

// 负载 flake 治理（同 ProgrammingPage.test 先例）：RTL 默认 1s 超时在全量套件
// 高负载下不够，显式放宽到 5s（仍有上界，不会掩盖真回归）。
const LOAD = { timeout: 5000 }

// 三个助手覆盖三种通道状态：gaea 运行中（核心）/ 小雨 未绑定 / 阿修 已停止
const STATUS_ROWS = [
  { id: 'gaea', name: 'gaea', personalityId: 'gaea', enabled: true, hasToken: true, wxRunning: true },
  { id: 'wx_muse', name: '小雨', personalityId: 'muse', enabled: true, hasToken: false, wxRunning: false },
  { id: 'wx_fix', name: '阿修', personalityId: 'fixer', enabled: false, hasToken: true, wxRunning: false },
]
const LIST_ROWS = [
  { id: 'gaea', name: 'gaea', personalityId: 'gaea', enabled: true, wxToken: 'tok-gaea', wxBotId: 'bot-gaea', wxUserId: 'uid-gaea' },
  { id: 'wx_muse', name: '小雨', personalityId: 'muse', enabled: true, portraitUrl: 'http://example.com/xiaoyu.png' },
  { id: 'wx_fix', name: '阿修', personalityId: 'fixer', enabled: false, wxToken: 'tok-fix', wxBotId: 'bot-fix', wxUserId: 'uid-fix' },
]

/** 冲刷微任务链（挂载/操作链路的 promise 续延）。 */
async function flushAsync() {
  await act(async () => {
    for (let i = 0; i < 30; i++) await Promise.resolve()
  })
}

/** 挂载页面并等首屏两路数据落定。 */
async function renderPage() {
  const view = render(<WeixinPage />)
  await flushAsync()
  return view
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.WhisperWeixinStatus.mockResolvedValue(STATUS_ROWS)
  mocks.WhisperAssistantList.mockResolvedValue(LIST_ROWS)
  mocks.WhisperAssistantSave.mockResolvedValue(undefined)
  mocks.WhisperAssistantDelete.mockResolvedValue(undefined)
  mocks.WhisperWeixinGetQR.mockResolvedValue({ qrcode: 'qr-token', imageUrl: 'data:image/png;base64,AAAA' })
  mocks.WhisperWeixinQRStatus.mockResolvedValue({ status: 'wait_scan' })
  mocks.WhisperWeixinQRStatusWithCode.mockResolvedValue({ status: 'wait_scan' })
  mocks.WeixinReminderList.mockResolvedValue([])
  mocks.WeixinReminderConfig.mockResolvedValue({ remindersEnabled: true })
  mocks.WeixinReminderAdd.mockResolvedValue({ id: 'r1', fireAt: '', status: 'pending' })
  mocks.WeixinReminderDelete.mockResolvedValue(undefined)
  mocks.WeixinReminderSetConfig.mockResolvedValue(undefined)
})

describe('微信助手管理台 · 连接卡', () => {
  it('① 列表渲染多助手：名字/人格 Tag/状态徽标/头像（有图用图、无图回退首字）', async () => {
    const { container } = await renderPage()
    expect(await screen.findByText('小雨', undefined, LOAD)).toBeTruthy()
    expect(screen.getByText('阿修')).toBeTruthy()
    // 连接卡 3 行助手
    expect(container.querySelectorAll('li.ant-list-item').length).toBe(3)
    // 状态徽标：运行中（gaea）/ 未绑定（小雨）/ 已停止（阿修）
    expect(screen.getByText('运行中')).toBeTruthy()
    expect(screen.getByText('未绑定')).toBeTruthy()
    expect(screen.getByText('已停止')).toBeTruthy()
    // 人格 Tag（字段取自 WhisperAssistantList 行）
    expect(screen.getByText('人格 gaea')).toBeTruthy()
    expect(screen.getByText('人格 muse')).toBeTruthy()
    expect(screen.getByText('人格 fixer')).toBeTruthy()
    // 小雨有 portraitUrl → 头像为图片；gaea 无 → 名字首字回退
    // （ant-avatar-image 类在 Avatar 的 span 宿主上，img 为其子元素）
    const xiaoyuAvatar = container.querySelector('.ant-avatar-image img') as HTMLImageElement
    expect(xiaoyuAvatar?.getAttribute('src')).toBe('http://example.com/xiaoyu.png')
    const gaeaRow = screen.getByText('gaea').closest('li')
    expect(gaeaRow?.querySelector('.ant-avatar')?.textContent).toBe('g')
  })

  it('② 启停翻转：WhisperAssistantSave 携带完整对象与翻转后的 enabled，并立即刷新两路数据', async () => {
    await renderPage()
    expect(await screen.findByText('阿修', undefined, LOAD)).toBeTruthy()
    const statusCalls = mocks.WhisperWeixinStatus.mock.calls.length
    const listCalls = mocks.WhisperAssistantList.mock.calls.length

    // 阿修 enabled=false → 点击开关翻转为 true
    fireEvent.click(screen.getByRole('switch', { name: '启停 阿修' }))

    await waitFor(() => {
      expect(mocks.WhisperAssistantSave).toHaveBeenCalledWith(expect.objectContaining({
        id: 'wx_fix', name: '阿修', personalityId: 'fixer',
        wxToken: 'tok-fix', wxBotId: 'bot-fix', wxUserId: 'uid-fix',
        enabled: true,
      }))
    }, LOAD)
    // 保存成功后立即重拉 Status + List
    await waitFor(() => expect(mocks.WhisperWeixinStatus.mock.calls.length).toBeGreaterThan(statusCalls), LOAD)
    await waitFor(() => expect(mocks.WhisperAssistantList.mock.calls.length).toBeGreaterThan(listCalls), LOAD)
  })

  it('③ gaea 核心助手：Switch 禁用、无删除入口；其他助手不受限', async () => {
    await renderPage()
    expect(await screen.findByText('gaea', undefined, LOAD)).toBeTruthy()
    expect((screen.getByRole('switch', { name: '启停 gaea' }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('switch', { name: '启停 阿修' }) as HTMLButtonElement).disabled).toBe(false)
    // 删除入口只出现在非核心助手行
    const gaeaRow = screen.getByText('gaea').closest('li')
    expect(gaeaRow?.querySelector('[aria-label="删除 gaea"]')).toBeNull()
    expect(screen.getByRole('button', { name: '删除 阿修' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '删除 小雨' })).toBeTruthy()
  })

  it('④ 删除：Popconfirm 确认后调用 WhisperAssistantDelete 并刷新', async () => {
    await renderPage()
    const statusCalls = mocks.WhisperWeixinStatus.mock.calls.length
    fireEvent.click(await screen.findByRole('button', { name: '删除 小雨' }, LOAD))
    // Popconfirm 确认按钮「删除」（antd 两字按钮自动插空格，用正则匹配）
    fireEvent.click(await screen.findByRole('button', { name: /^删\s*除$/ }, LOAD))
    await waitFor(() => expect(mocks.WhisperAssistantDelete).toHaveBeenCalledWith('wx_muse'), LOAD)
    await waitFor(() => expect(mocks.WhisperWeixinStatus.mock.calls.length).toBeGreaterThan(statusCalls), LOAD)
  })

  it('⑤ 新增微信助手：表单确定 → 本地暂存进入扫码流，确认后 Save 落库', async () => {
    mocks.WhisperWeixinQRStatus.mockResolvedValue({ status: 'confirmed', botToken: 'tok-n', botId: 'bot-n', userId: 'uid-n' })
    await renderPage()
    fireEvent.click(await screen.findByRole('button', { name: /新增微信助手/ }, LOAD))
    // 名字必填、人格 ID 默认 gaea
    fireEvent.change(screen.getByPlaceholderText('助手名字（必填）'), { target: { value: '小北' } })
    expect((screen.getByDisplayValue('gaea') as HTMLInputElement).value).toBe('gaea')
    fireEvent.click(screen.getByRole('button', { name: '下一步：扫码绑定' }))

    // 进入扫码流：二维码拉取 + 动态标题指向新助手
    await screen.findByText('扫码绑定 · 小北', undefined, LOAD)
    expect(mocks.WhisperWeixinGetQR).toHaveBeenCalled()
    await waitFor(() => expect(mocks.WhisperWeixinQRStatus).toHaveBeenCalledWith('qr-token'), LOAD)

    // 扫码确认（mock 直接回 confirmed）→ Save 携带暂存对象 + 新 token/botId/userId
    fireEvent.click(await screen.findByRole('button', { name: '保存绑定并启动通道' }, LOAD))
    await waitFor(() => {
      expect(mocks.WhisperAssistantSave).toHaveBeenCalledWith(expect.objectContaining({
        id: expect.stringMatching(/^wx_/),
        name: '小北', personalityId: 'gaea', enabled: true,
        wxToken: 'tok-n', wxBotId: 'bot-n', wxUserId: 'uid-n',
      }))
    }, LOAD)
  })
})
