// WeixinPage.test.tsx — 青鸟（微信助手）「通讯枢纽」工作台测试。
// 桥接经 vi.mock('../gaea/lib/bridge') 注入确定性数据（app.* 方法逐个 mock），
// 覆盖：①多助手通道轨道渲染（状态字/状态点/头像回退/详情人格 Tag）②启停翻转
// 调用 Save 且 enabled 正确（携带完整对象）③gaea 核心助手禁删禁停（亦无编辑）
// ④删除 Popconfirm → WhisperAssistantDelete ⑤新增微信助手表单 → 本地暂存 →
// 扫码流确认后 Save ⑥离线提醒视图（列表渲染 + 全局开关 + 删除）⑦使用指南
// ⑧会话过期警示 ⑨人格选择器分组/详情/立绘 ⑩编辑助手（预填 + 改名换人格 →
// Save 合并 viewOf 携带新名字/人格与原 token）。

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'

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
  WhisperGetPersonalities: vi.fn(),
  CharacterList: vi.fn(),
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
const REMINDER_ROWS = [
  { id: 'r1', text: '交周报', fireAt: '2026-09-03T09:00:00+08:00', status: 'pending', source: 'weixin', failCount: 0 },
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

/** 点击左轨道通道项选中该助手（主区切到通道详情）。 */
async function selectChannel(name: string) {
  fireEvent.click(await screen.findByRole('button', { name: `通道 ${name}` }, LOAD))
  await flushAsync()
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
  mocks.WhisperGetPersonalities.mockResolvedValue([
    { id: 'gaea', label: '盖亚', gender: 'female', tags: ['秘书', '工作'], voiceGuide: '专业严谨的办公秘书口吻' },
    { id: 'muse', label: '缪斯', gender: 'female', tags: ['创作'], voiceGuide: '灵感充沛的写作伙伴' },
    { id: 'fixer', label: '修哥', gender: 'male', tags: ['执行'], voiceGuide: '干脆利落' },
  ])
  mocks.CharacterList.mockResolvedValue({
    items: [
      {
        id: 'c_lin', name: '林晚', kind: 'custom', gender: 'female', tags: ['女主角', '清冷'],
        portraitUrl: 'http://example.com/lin.png', personality: '清冷聪慧，外冷内热',
        background: '前朝遗孤，隐姓埋名于市井', chatEnabled: true,
      },
    ],
    total: 1,
  })
})

describe('青鸟工作台 · 通道轨道与详情', () => {
  it('① 轨道渲染多助手：名字/状态字/状态点/头像（有图用图、无图回退首字），详情显示人格 Tag', async () => {
    const { container } = await renderPage()
    // 通道轨道 3 条助手项（新增/提醒/指南按钮除外）
    const channelItems = container.querySelectorAll('.wx-rail .wx-rail-ch')
    expect(channelItems.length).toBe(3)
    // 轨道状态字与顺序：运行中（gaea）/ 未绑定（小雨）/ 已停止（阿修）
    const statuses = Array.from(container.querySelectorAll('.wx-rail .wx-rail-status')).map((el) => el.textContent)
    expect(statuses).toEqual(['运行中', '未绑定', '已停止'])
    // 状态点三态 class
    expect(container.querySelector('.wx-dot.is-running')).toBeTruthy()
    expect(container.querySelector('.wx-dot.is-unbound')).toBeTruthy()
    expect(container.querySelector('.wx-dot.is-stopped')).toBeTruthy()
    // 小雨有 portraitUrl → 头像为图片；gaea 无 → 名字首字回退
    // （ant-avatar-image 类在 Avatar 的 span 宿主上，img 为其子元素）
    const xiaoyuAvatar = container.querySelector('.wx-rail .ant-avatar-image img') as HTMLImageElement
    expect(xiaoyuAvatar?.getAttribute('src')).toBe('http://example.com/xiaoyu.png')
    const gaeaAvatar = container.querySelector('.wx-rail-item .ant-avatar')
    expect(gaeaAvatar?.textContent).toBe('g')
    // 默认选中首条通道 gaea → 详情显示人格 Tag
    expect(await screen.findByText('人格 gaea', undefined, LOAD)).toBeTruthy()
    // 切到小雨 → 人格 muse
    await selectChannel('小雨')
    expect(screen.getByText('人格 muse')).toBeTruthy()
  })

  it('② 启停翻转：选中阿修后拨开关，WhisperAssistantSave 携带完整对象与翻转后的 enabled，并立即刷新两路数据', async () => {
    await renderPage()
    await selectChannel('阿修')
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
    expect(await screen.findByRole('switch', { name: '启停 gaea' }, LOAD)).toBeTruthy()
    expect((screen.getByRole('switch', { name: '启停 gaea' }) as HTMLButtonElement).disabled).toBe(true)
    // 核心助手详情无删除入口、无编辑入口（核心锚点禁删禁停禁改）
    expect(screen.queryByRole('button', { name: '删除 gaea' })).toBeNull()
    expect(screen.queryByRole('button', { name: '编辑 gaea' })).toBeNull()
    // 阿修（非核心）：可启停、有删除入口
    await selectChannel('阿修')
    expect((screen.getByRole('switch', { name: '启停 阿修' }) as HTMLButtonElement).disabled).toBe(false)
    expect(screen.getByRole('button', { name: '删除 阿修' })).toBeTruthy()
    // 小雨（非核心未绑定）同样有删除入口
    await selectChannel('小雨')
    expect(screen.getByRole('button', { name: '删除 小雨' })).toBeTruthy()
  })

  it('④ 删除：选中小雨 → Popconfirm 确认后调用 WhisperAssistantDelete 并刷新', async () => {
    await renderPage()
    await selectChannel('小雨')
    const statusCalls = mocks.WhisperWeixinStatus.mock.calls.length
    fireEvent.click(await screen.findByRole('button', { name: '删除 小雨' }, LOAD))
    // Popconfirm 确认按钮「删除」（antd 两字按钮自动插空格，用正则匹配）
    fireEvent.click(await screen.findByRole('button', { name: /^删\s*除$/ }, LOAD))
    await waitFor(() => expect(mocks.WhisperAssistantDelete).toHaveBeenCalledWith('wx_muse'), LOAD)
    await waitFor(() => expect(mocks.WhisperWeixinStatus.mock.calls.length).toBeGreaterThan(statusCalls), LOAD)
  })

  it('⑤ 新增微信助手：表单确定 → 本地暂存进入扫码流，确认后 Save 落库', async () => {
    // 先 wait_scan 验「扫码」步，再切 confirmed 验「完成」步与保存
    mocks.WhisperWeixinQRStatus.mockResolvedValue({ status: 'wait_scan' })
    await renderPage()
    fireEvent.click(await screen.findByRole('button', { name: '新增青鸟助手' }, LOAD))
    // 人格选择器加载：默认选中 gaea 预设（盖亚）
    await waitFor(() => {
      expect(document.querySelector('.wx-pk-item.is-active')?.textContent).toContain('盖亚')
    }, LOAD)
    fireEvent.change(screen.getByPlaceholderText('助手名字（必填）'), { target: { value: '小北' } })
    fireEvent.click(screen.getByRole('button', { name: '下一步：扫码绑定' }))

    // 进入扫码流：二维码拉取 + 动态标题指向新助手 + 三步指示在「扫码」
    await screen.findByText('扫码绑定 · 小北', undefined, LOAD)
    expect(mocks.WhisperWeixinGetQR).toHaveBeenCalled()
    await waitFor(() => expect(mocks.WhisperWeixinQRStatus).toHaveBeenCalledWith('qr-token'), LOAD)
    expect(document.querySelector('.wx-qr-step.is-current')?.textContent).toContain('扫码')

    // 扫码确认 → 步进到「完成」→ Save 携带暂存对象 + 新 token/botId/userId
    mocks.WhisperWeixinQRStatus.mockResolvedValue({ status: 'confirmed', botToken: 'tok-n', botId: 'bot-n', userId: 'uid-n' })
    fireEvent.click(await screen.findByRole('button', { name: '保存绑定并启动通道' }, LOAD))
    expect(document.querySelector('.wx-qr-step.is-current')?.textContent).toContain('完成')
    await waitFor(() => {
      expect(mocks.WhisperAssistantSave).toHaveBeenCalledWith(expect.objectContaining({
        id: expect.stringMatching(/^wx_/),
        name: '小北', personalityId: 'gaea', enabled: true,
        wxToken: 'tok-n', wxBotId: 'bot-n', wxUserId: 'uid-n',
      }))
    }, LOAD)
  })

  it('⑨ 人格选择器：分组渲染（预设/角色库）+ 详情面板 + 立绘随 Save 带出', async () => {
    mocks.WhisperWeixinQRStatus.mockResolvedValue({ status: 'confirmed', botToken: 'tok-n', botId: 'bot-n', userId: 'uid-n' })
    await renderPage()
    fireEvent.click(await screen.findByRole('button', { name: '新增青鸟助手' }, LOAD))
    // 分组列表：预设（盖亚/缪斯）+ 角色库（林晚）；文案断言收在 listbox 内
    //（footer 说明句含同词，全屏 getByText 会多匹配）
    const listbox = await screen.findByRole('listbox', { name: '人格选择' }, LOAD)
    expect(listbox.querySelectorAll('.wx-pk-group').length).toBe(2)
    expect(listbox.querySelector('.wx-pk-group')?.textContent).toBe('轻语预设')
    expect(within(listbox).getByRole('option', { name: '选择人格 盖亚' })).toBeTruthy()
    expect(within(listbox).getByRole('option', { name: '选择人格 缪斯' })).toBeTruthy()
    expect(within(listbox).getByRole('option', { name: '选择人格 林晚' })).toBeTruthy()

    // 选中林晚 → 详情面板显示人格/背景 + 立绘
    fireEvent.click(screen.getByRole('option', { name: '选择人格 林晚' }))
    const detail = document.querySelector('.wx-pk-detail') as HTMLElement
    expect(detail.textContent).toContain('清冷聪慧')
    expect(detail.textContent).toContain('前朝遗孤')
    expect(detail.querySelector('img')?.getAttribute('src')).toBe('http://example.com/lin.png')

    // 确认 → Save 携带角色库 id 与立绘 URL
    fireEvent.change(screen.getByPlaceholderText('助手名字（必填）'), { target: { value: '晚晚' } })
    fireEvent.click(screen.getByRole('button', { name: '下一步：扫码绑定' }))
    fireEvent.click(await screen.findByRole('button', { name: '保存绑定并启动通道' }, LOAD))
    await waitFor(() => {
      expect(mocks.WhisperAssistantSave).toHaveBeenCalledWith(expect.objectContaining({
        personalityId: 'c_lin', portraitUrl: 'http://example.com/lin.png',
      }))
    }, LOAD)
  })

  it('⑩ 编辑助手：详情头部「编辑」→ 弹窗预填名字与现人格，改名换人格后 Save 合并原绑定字段', async () => {
    await renderPage()
    await selectChannel('阿修')
    // 详情头部操作区有「编辑 阿修」按钮 → 打开编辑弹窗
    fireEvent.click(await screen.findByRole('button', { name: '编辑 阿修' }, LOAD))
    expect(await screen.findByText('编辑助手 · 阿修', undefined, LOAD)).toBeTruthy()
    // 名字预填现名；人格选择器激活行为当前人格（fixer → 修哥）
    expect(screen.getByDisplayValue('阿修')).toBeTruthy()
    await waitFor(() => {
      expect(document.querySelector('.wx-pk-item.is-active')?.textContent).toContain('修哥')
    }, LOAD)

    // 改名 + 换人格（缪斯）→ 保存修改
    fireEvent.change(screen.getByDisplayValue('阿修'), { target: { value: '阿修2号' } })
    fireEvent.click(screen.getByRole('option', { name: '选择人格 缪斯' }))
    fireEvent.click(screen.getByRole('button', { name: '保存修改' }))

    // Save 以 viewOf 为底：原 id/token 绑定字段原样保留，叠加新名字与新人格
    const statusCalls = mocks.WhisperWeixinStatus.mock.calls.length
    const listCalls = mocks.WhisperAssistantList.mock.calls.length
    await waitFor(() => {
      expect(mocks.WhisperAssistantSave).toHaveBeenCalledWith(expect.objectContaining({
        id: 'wx_fix', name: '阿修2号', personalityId: 'muse', wxToken: 'tok-fix',
      }))
    }, LOAD)
    // 保存成功后立即重拉 Status + List（弹窗 DOM 因 rc-dialog 离场动画在 jsdom
    // 不结束而冻结，故关闭态不按文本断言，沿用 ② 的刷新口径）
    await waitFor(() => expect(mocks.WhisperWeixinStatus.mock.calls.length).toBeGreaterThan(statusCalls), LOAD)
    await waitFor(() => expect(mocks.WhisperAssistantList.mock.calls.length).toBeGreaterThan(listCalls), LOAD)
  })
})

describe('青鸟工作台 · 提醒与指南', () => {
  it('⑥ 离线提醒视图：列表渲染（状态/来源/时间）+ 全局开关翻转 + 删除', async () => {
    mocks.WeixinReminderList.mockResolvedValue(REMINDER_ROWS)
    const { container } = await renderPage()
    // 有待触发时按钮可访问名带徽标计数，用前缀匹配
    fireEvent.click(await screen.findByRole('button', { name: /^离线提醒/ }, LOAD))
    await flushAsync()

    // 列表行：文本 + 待触发 + 微信下达 + 等宽时间
    expect(await screen.findByText('交周报', undefined, LOAD)).toBeTruthy()
    expect(screen.getByText('待触发')).toBeTruthy()
    expect(screen.getByText('微信下达')).toBeTruthy()
    expect(container.querySelectorAll('.wx-rem-item').length).toBe(1)
    // 轨道徽标：1 条待触发
    expect(screen.getByRole('button', { name: '离线提醒，1 条待触发' })).toBeTruthy()

    // 全局开关翻转 → SetConfig 携带 false
    fireEvent.click(screen.getByRole('switch', { name: '到点回推微信' }))
    await waitFor(() => expect(mocks.WeixinReminderSetConfig).toHaveBeenCalledWith('{"remindersEnabled":false}'), LOAD)

    // 删除提醒
    fireEvent.click(screen.getByRole('button', { name: '删除提醒 交周报' }))
    await waitFor(() => expect(mocks.WeixinReminderDelete).toHaveBeenCalledWith('r1'), LOAD)
  })

  it('⑦ 使用指南视图：指令示例与边界说明可见', async () => {
    await renderPage()
    fireEvent.click(await screen.findByRole('button', { name: '使用指南' }, LOAD))
    expect(await screen.findByText('提醒我 30分钟后 喝水', undefined, LOAD)).toBeTruthy()
    expect(screen.getByText('明天早上9点 开站会')).toBeTruthy()
    expect(screen.getByText('18:30 接孩子')).toBeTruthy()
    expect(screen.getByText(/无时间表达的提醒请求会收到格式提示/)).toBeTruthy()
  })

  it('⑧ 会话过期：过期通道的详情内联警示', async () => {
    mocks.WhisperWeixinStatus.mockResolvedValue([
      { ...STATUS_ROWS[0], wxSessionExpired: true },
    ])
    await renderPage()
    expect(await screen.findByText('该助手微信会话已过期', undefined, LOAD)).toBeTruthy()
    // 轨道状态字同步为会话过期
    expect(document.querySelector('.wx-rail .wx-rail-status')?.textContent).toContain('会话过期')
  })
})
