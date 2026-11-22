import React, { useEffect, useState } from 'react'
import {
  AppstoreOutlined, AudioOutlined, BgColorsOutlined, CloudSyncOutlined,
  ControlOutlined, DashboardOutlined, DatabaseOutlined, ExperimentOutlined, FontSizeOutlined,
  PictureOutlined, RobotOutlined, SafetyCertificateOutlined, SearchOutlined,
  SoundOutlined, StarOutlined, ThunderboltOutlined, ToolOutlined, UserOutlined,
} from '@ant-design/icons'
import ModelSettingsProvider from '../../pages/modelcenter/ModelSettingsProvider'
import { OverviewSection } from '../../pages/modelcenter/OverviewSection'
import { HerdsmanCatalogSection } from '../../pages/modelcenter/HerdsmanCatalogSection'
import { RecommendSection } from '../../pages/modelcenter/RecommendSection'
import { BenchmarkSection } from '../../pages/modelcenter/BenchmarkSection'
import { RetrievalEvalSection } from '../../pages/modelcenter/RetrievalEvalSection'
import { AttributionSection } from '../../pages/modelcenter/AttributionSection'
import { StatsSection } from '../../pages/modelcenter/StatsSection'
import { StrategySection } from '../../pages/modelcenter/StrategySection'
import { SchedulingSection } from '../../pages/modelcenter/SchedulingSection'
import { ImageSection } from '../../pages/modelcenter/ImageSection'
import { VoiceSection } from '../../pages/modelcenter/VoiceSection'
import { CapabilitiesInline } from '../../gaea/components/capabilities/CapabilitiesInline'
import { WORKSPACE_TABS } from '../../gaea/lib/workspaceTabs'
import { usePaneEnabledStore } from '../../gaea/lib/paneEnabled'
import { WORKSPACE_TABS as WENXUE_WORKSPACE_TABS } from '../../wenxue/lib/workspaceTabs'
import { usePaneEnabledStore as useWenxuePaneEnabledStore } from '../../wenxue/lib/paneEnabled'
import '../../pages/modelcenter/modelcenter.css'
import ModelPanel from './ModelPanel'
import ChatPanel, { ChatDataSection } from './ChatPanel'
import ChatVoiceSection from './ChatVoiceSection'
import VoiceSettingsPanel from '../VoiceSettingsPanel'
import OfficePanel from './OfficePanel'
import OfficeMemorySection from './OfficeMemorySection'
import PaneManagementSection from './PaneManagementSection'
import WorkbenchPrefsPanel from './WorkbenchPrefsPanel'
import ContextPrefsPanel from './ContextPrefsPanel'
import ImageGenPanel from './ImageGenPanel'
import DataPanel from './DataPanel'
import SecurityPanel from './SecurityPanel'
import SearchPanel from './SearchPanel'
import WenxuePromptSection from './WenxuePromptSection'
import AppearancePanel, {
  AccentPanel,
  AmbiencePanel,
  DarkModePanel,
  DensityPanel,
  FontPanel,
  LanguagePanel,
  MeteorPanel,
  MotionPanel,
  SurfacePanel,
} from './AppearancePanel'
import SettingsPanelTabs, { type SettingsPanelTab } from './SettingsPanelTabs'

interface QueryProps {
  query?: string
  initialTab?: string
  onTabChange?: (tab: string) => void
}

const GENERAL_TABS: SettingsPanelTab[] = [
  {
    key: 'appearance',
    label: '外观',
    description: '选择主题色系与明暗模式，并实时预览最终效果。',
    icon: <BgColorsOutlined />,
    keywords: ['外观', '主题', '色系', '暗色', '亮色', '明暗', '模式', '预览', '氛围', '流星', '星尘', '星辰', '密度', '速度', '纯色', '彩色'],
    content: (
      <>
        <AppearancePanel />
        <DarkModePanel />
        <SurfacePanel />
        <AmbiencePanel />
        <MeteorPanel />
      </>
    ),
  },
  {
    key: 'typography',
    label: '排版',
    description: '调整字体、字号与界面密度。',
    icon: <FontSizeOutlined />,
    keywords: ['排版', '字体', '字号', '密度', '紧凑', '宽松'],
    content: (
      <>
        <FontPanel />
        <DensityPanel />
      </>
    ),
  },
  {
    key: 'motion-language',
    label: '动效与语言',
    description: '控制界面动效、强调色与显示语言。',
    icon: <ThunderboltOutlined />,
    keywords: ['动效', '动画', '强调色', '颜色', '语言', '界面语言', '英文', '繁体', '简体', 'language'],
    content: (
      <>
        <MotionPanel />
        <AccentPanel />
        <LanguagePanel />
      </>
    ),
  },
]

export const GeneralSettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <SettingsPanelTabs
    ariaLabel="通用设置任务"
    tabs={GENERAL_TABS}
    query={query}
    defaultKey={initialTab}
    onActiveChange={onTabChange}
  />
)

const RuntimeStrategyPanel: React.FC<{ forceOpen?: boolean }> = ({ forceOpen = false }) => {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    if (forceOpen) setOpen(true)
  }, [forceOpen])

  return (
    <>
      <SchedulingSection />
      <details
        className="settings-runtime-advanced"
        open={open}
        onToggle={(event) => setOpen(event.currentTarget.open)}
      >
        <summary>
          <span className="settings-runtime-advanced__icon" aria-hidden="true"><ControlOutlined /></span>
          <span className="settings-runtime-advanced__copy">
            <strong>空间装配与权限</strong>
            <span>办公/娱乐空间的模型覆写、审批默认与内容护栏；高级运行配置</span>
          </span>
          <span className="settings-runtime-advanced__badge">高级</span>
        </summary>
        {open && (
          <div className="settings-runtime-advanced__body">
            <StrategySection />
          </div>
        )}
      </details>
    </>
  )
}

const buildModelTabs = (query = ''): SettingsPanelTab[] => {
  const q = query.trim().toLowerCase()
  const spacePolicyHit = !!q && ['空间', '空间策略', '覆写', '护栏', '权限', 'space'].some((keyword) => (
    keyword.includes(q) || q.includes(keyword)
  ))

  return [
  {
    key: 'overview',
    label: '概览',
    description: '全局模型参数与运行状态：当前模型、推理强度、费用折算和引擎健康。',
    icon: <DashboardOutlined />,
    keywords: ['概览', '当前模型', '全局模型', '推理', '强度', '汇率', '美元', '人民币', '引擎健康', '调用健康', 'reasoning', 'usd', 'cny'],
    content: (
      <>
        <ModelPanel />
        <OverviewSection showActions={false} />
      </>
    ),
  },
  {
    key: 'catalog',
    label: '模型库',
    description: '浏览 Herdsman 模型目录、安装状态与能力，决定哪些模型进入日常路由。',
    icon: <AppstoreOutlined />,
    keywords: ['模型库', '模型目录', 'herdsman', '安装', '下载', '卸载', '量化', '能力', '目录'],
    content: <HerdsmanCatalogSection />,
  },
  {
    key: 'selection',
    label: '选型参考',
    description: '按社区实战反馈查看模型谱系、量化建议与本机模型适用面。',
    icon: <StarOutlined />,
    keywords: ['选型', '选型参考', '推荐', '模型推荐', '量化', 'q4', 'q8', '文笔', '智力', '长程'],
    content: <RecommendSection />,
  },
  {
    key: 'quality-cost',
    label: '质量与成本',
    description: '集中查看受控测评、检索质量与调用成本归因。',
    icon: <ExperimentOutlined />,
    keywords: ['测评', '质量', '检索', '受控测评', '成本', '归因', '费用', '调用统计', 'benchmark', 'retrieval', 'cost'],
    content: (
      <>
        <BenchmarkSection />
        <RetrievalEvalSection />
        <AttributionSection />
        <StatsSection />
      </>
    ),
  },
  {
    key: 'scheduling',
    label: '运行策略',
    description: '管理保活、自动预载与故障转移；空间装配、权限和护栏收纳在高级项中。',
    icon: <CloudSyncOutlined />,
    keywords: ['运行策略', '本地调度', '调度', '保活', '预载', '故障转移', '运行中', '重试', 'keepwarm', 'preload', 'failover', '空间策略', '空间', '办公空间', '娱乐空间', '覆写', '权限', '护栏', 'space'],
    content: <RuntimeStrategyPanel forceOpen={spacePolicyHit} />,
  },
  ]
}

export const ModelSettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <ModelSettingsProvider>
    <SettingsPanelTabs
      ariaLabel="模型设置任务"
      tabs={buildModelTabs(query)}
      query={query}
      defaultKey={initialTab}
      onActiveChange={onTabChange}
    />
  </ModelSettingsProvider>
)

const CHAT_TABS: SettingsPanelTab[] = [
  {
    key: 'persona',
    label: '角色与人格',
    description: '设置伴侣称呼、性别和当前对话人格。',
    icon: <UserOutlined />,
    keywords: ['角色', '人格', '伴侣', '称呼', '性别', '对话'],
    content: <ChatPanel />,
  },
  {
    key: 'default-voice',
    label: '默认音色',
    description: '选择聊天和全局朗读使用的默认合成音色。',
    icon: <SoundOutlined />,
    keywords: ['音色', '语音', '朗读', '合成', 'tts', 'voice'],
    content: <ChatVoiceSection />,
  },
  {
    key: 'speech-engine',
    label: '朗读引擎',
    description: '配置语音管道、Breeze 克隆、麦克风与确认词。',
    icon: <AudioOutlined />,
    keywords: ['朗读引擎', '语音引擎', '麦克风', '确认词', '克隆', 'breeze', 'stt'],
    content: <VoiceSettingsPanel />,
  },
  {
    key: 'speech-models',
    label: '语音模型',
    description: '为 TTS、ASR 与 OCR 指定各自使用的模型。',
    icon: <RobotOutlined />,
    keywords: ['语音模型', 'tts', 'asr', 'ocr', '识别', '引擎路由'],
    content: <VoiceSection />,
  },
]

export const ChatSettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <ModelSettingsProvider>
    <SettingsPanelTabs
      ariaLabel="聊天设置任务"
      tabs={CHAT_TABS}
      query={query}
      defaultKey={initialTab}
      onActiveChange={onTabChange}
    />
  </ModelSettingsProvider>
)

const OFFICE_PANE_SECTION = (
  <PaneManagementSection
    title="工位面板"
    desc="办公右栏工作台面板启停：已打开的页面不因停用强关，启用集只管入口可见性（CAD 图纸等可选卡在此常驻可开）"
    tabs={WORKSPACE_TABS}
    useStore={usePaneEnabledStore}
  />
)

const OFFICE_TABS: SettingsPanelTab[] = [
  {
    key: 'workbench',
    label: '工作台',
    description: '决定任务视图和上下文浏览器如何展开、排序与呈现。',
    icon: <AppstoreOutlined />,
    keywords: ['工作台', '偏好', '自动展开', '自动切换', '任务视图', '上下文', '粒度', '排序'],
    content: (
      <>
        <WorkbenchPrefsPanel />
        <ContextPrefsPanel />
      </>
    ),
  },
  {
    key: 'engine',
    label: '办公引擎',
    description: '设置办公 Agent 的模型、参数、权限、沙箱和实时协同。',
    icon: <ToolOutlined />,
    keywords: ['办公引擎', '引擎', '模型', '温度', '权限', '沙箱', '系统提示词', '实时协同'],
    content: <OfficePanel />,
  },
  {
    key: 'memory',
    label: '记忆与面板',
    description: '管理办公记忆、晨报预载和右栏工位入口。',
    icon: <DatabaseOutlined />,
    keywords: ['记忆', '晨报', '工位', '面板', 'cad', '右栏'],
    content: (
      <>
        <OfficeMemorySection />
        {OFFICE_PANE_SECTION}
      </>
    ),
  },
  {
    key: 'capabilities',
    label: '工具与技能',
    description: '查看并管理连接器、MCP 服务器、工具与技能。',
    icon: <ThunderboltOutlined />,
    keywords: ['工具', '技能', 'mcp', '连接器', '服务器', '能力', '重载', '热加载'],
    content: <CapabilitiesInline />,
  },
]

export const OfficeSettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <SettingsPanelTabs
    ariaLabel="办公设置任务"
    tabs={OFFICE_TABS}
    query={query}
    defaultKey={initialTab}
    onActiveChange={onTabChange}
  />
)

const IMAGE_TABS: SettingsPanelTab[] = [
  {
    key: 'settings',
    label: '生图配置',
    description: '选择绘梦后端、默认模型、保存位置和 ComfyUI 启停。',
    icon: <BgColorsOutlined />,
    keywords: ['生图', '绘梦', '图像', '后端', '模型', '保存', 'comfyui', 'xai'],
    content: <ImageGenPanel />,
  },
  {
    key: 'backends',
    label: '后端与模型',
    description: '配置图像引擎地址、模型清单和本地运行参数。',
    icon: <PictureOutlined />,
    keywords: ['图像模型', '后端', '引擎', '模型清单', '本地', 'python'],
    content: (
      <ModelSettingsProvider>
        <ImageSection />
      </ModelSettingsProvider>
    ),
  },
]

export const ImageSettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <SettingsPanelTabs
    ariaLabel="绘梦设置任务"
    tabs={IMAGE_TABS}
    query={query}
    defaultKey={initialTab}
    onActiveChange={onTabChange}
  />
)

const SECURITY_TABS: SettingsPanelTab[] = [
  {
    key: 'privacy',
    label: '隐私与路由',
    description: '决定敏感数据、办公内容和离线模式是否只在本机处理。',
    icon: <SafetyCertificateOutlined />,
    keywords: ['隐私', '敏感', '本地化', '本地', '离线', '路由', '局域网', '调试', 'webview'],
    content: <SecurityPanel />,
  },
  {
    key: 'web-search',
    label: '联网搜索',
    description: '配置聊天与办公共用的搜索主源和连接测试。',
    icon: <SearchOutlined />,
    keywords: ['联网', '网络搜索', '搜索', '主源', 'tavily', 'brave', 'searxng', 'api key'],
    content: <SearchPanel />,
  },
]

export const SecuritySettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <SettingsPanelTabs
    ariaLabel="安全与网络设置任务"
    tabs={SECURITY_TABS}
    query={query}
    defaultKey={initialTab}
    onActiveChange={onTabChange}
  />
)

const DATA_TABS: SettingsPanelTab[] = [
  {
    key: 'backup',
    label: '备份与恢复',
    description: '备份或恢复记忆、配置、会话等全部本地数据。',
    icon: <DatabaseOutlined />,
    keywords: ['备份', '恢复', '迁移', '换机', '数据', '配置', '快照', '回滚'],
    content: <DataPanel />,
  },
  {
    key: 'chat-history',
    label: '聊天历史',
    description: '单独导出或导入聊天会话历史。',
    icon: <CloudSyncOutlined />,
    keywords: ['聊天历史', '聊天', '历史', '导出', '导入', 'zip', '会话'],
    content: <ChatDataSection />,
  },
]

export const DataSettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <SettingsPanelTabs
    ariaLabel="数据设置任务"
    tabs={DATA_TABS}
    query={query}
    defaultKey={initialTab}
    onActiveChange={onTabChange}
  />
)

const WENXUE_TABS: SettingsPanelTab[] = [
  {
    key: 'prompt',
    label: '身份提示词',
    description: '设置文学图文工作台的身份提示词，并预览当前生效来源。',
    icon: <RobotOutlined />,
    keywords: ['文学', '身份提示词', '提示词', '人格', 'system prompt', '文件'],
    content: <WenxuePromptSection />,
  },
  {
    key: 'panes',
    label: '书斋面板',
    description: '管理文学右栏工作台面板入口。',
    icon: <AppstoreOutlined />,
    keywords: ['文学', '书斋', '面板', '工作台', '右栏', '入口'],
    content: (
      <PaneManagementSection
        title="书斋面板"
        desc="文学壳层右栏工作台面板启停（与办公工位各自独立清单）"
        tabs={WENXUE_WORKSPACE_TABS}
        useStore={useWenxuePaneEnabledStore}
      />
    ),
  },
]

export const WenxueSettingsPanel: React.FC<QueryProps> = ({ query, initialTab, onTabChange }) => (
  <SettingsPanelTabs
    ariaLabel="文学设置任务"
    tabs={WENXUE_TABS}
    query={query}
    defaultKey={initialTab}
    onActiveChange={onTabChange}
  />
)
