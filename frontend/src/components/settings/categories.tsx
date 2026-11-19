import React from 'react'
import {
  AppstoreOutlined, MessageOutlined, PictureOutlined,
  SettingOutlined, ApiOutlined, InfoCircleOutlined,
  SafetyCertificateOutlined, DatabaseOutlined, ClusterOutlined,
  ReadOutlined, WechatOutlined, ThunderboltOutlined,
} from '@ant-design/icons'
import type { DictKey } from '../../gaea/locales/en'
import AppearancePanel, { DarkModePanel, FontPanel, DensityPanel, MotionPanel, AccentPanel, LanguagePanel } from './AppearancePanel'
import ChatPanel from './ChatPanel'
import ChatVoiceSection from './ChatVoiceSection'
import VoiceSettingsPanel from '../VoiceSettingsPanel'
import OfficeMemorySection from './OfficeMemorySection'
import PaneManagementSection from './PaneManagementSection'
import ModelSettingsProvider from '../../pages/modelcenter/ModelSettingsProvider'
import { BindSection } from '../../pages/modelcenter/BindSection'
import { StrategySection } from '../../pages/modelcenter/StrategySection'
import { SchedulingSection } from '../../pages/modelcenter/SchedulingSection'
import { EngineSection } from '../../pages/modelcenter/EngineSection'
import { ImageSection } from '../../pages/modelcenter/ImageSection'
import { VoiceSection } from '../../pages/modelcenter/VoiceSection'
import { CapabilitiesInline } from '../../gaea/components/capabilities/CapabilitiesInline'
import { WORKSPACE_TABS } from '../../gaea/lib/workspaceTabs'
import { usePaneEnabledStore } from '../../gaea/lib/paneEnabled'
import { WORKSPACE_TABS as WENXUE_TABS } from '../../wenxue/lib/workspaceTabs'
import { usePaneEnabledStore as useWenxuePaneEnabled } from '../../wenxue/lib/paneEnabled'
import WeixinServiceSection from './WeixinServiceSection'
import PantheonAgentsSection from './PantheonAgentsSection'
import ImageGenPanel from './ImageGenPanel'
import OfficePanel from './OfficePanel'
import SearchPanel from './SearchPanel'
import WorkbenchPrefsPanel from './WorkbenchPrefsPanel'
import ContextPrefsPanel from './ContextPrefsPanel'
import ModelPanel from './ModelPanel'
import SecurityPanel from './SecurityPanel'
import DataPanel from './DataPanel'
import CodeAtlasPanel from './CodeAtlasPanel'
import AboutPanel from './AboutPanel'

/**
 * 设置中枢注册表（设置板块重规划批 1）：分组定义单源，SettingsPage 只负责
 * 渲染（左导航两级「全局/板块」+ 搜索 + 锚点直达）。后续批次收编各板块设置
 * 时在此挂组，不动页面骨架。
 */

export type SettingsZone = 'global' | 'boards'

export interface Category {
  key: string
  zone: SettingsZone
  icon: React.ReactNode
  /** zh 兜底文案（搜索关键词用，数据层不可 use hook；批 5 起亦作 labelKey 缺省时的渲染兜底） */
  label: string
  desc: string
  /** i18n 渲染 key（可选：locales 并行线在途期间新增组暂缺键，渲染走 zh 兜底，
   *  键待并行线收官后补齐回切——与批 1 zone 标题同款过渡模式） */
  labelKey?: DictKey
  descKey?: DictKey
  keywords: string[]
  panel: React.ReactNode
}

/** 导航分区（批 5 点亮：office/imagegen 归位 boards 区 + 文学/微信/众神殿入驻，
 *  两级标题自此常驻；搜索过滤到单 zone 时自动隐藏标题）。
 *  zone 标题暂用 zh 兜底常量——locales 三文件当前有并行线在途改动（遥测/
 * 启动文案段），键（settings.zone.global/boards）待该线收官后补，避免混提交。 */
export const SETTINGS_ZONES: { key: SettingsZone; label: string }[] = [
  { key: 'global', label: '全局' },
  { key: 'boards', label: '板块' },
]

/** 设置分组：现有 10 组全部驻 global 区（含代码图谱，用户拍板保留）；
 *  板块区（办公/文学/绘梦/众神殿/微信/进度/记忆中枢）由批 5 逐组挂入。 */
export const SETTINGS_CATEGORIES: Category[] = [
  {
    key: 'general',
    zone: 'global',
    icon: <AppstoreOutlined />,
    label: '通用',
    labelKey: 'settings.cat.general',
    desc: '外观',
    descKey: 'settings.cat.generalDesc',
    keywords: ['通用', '外观', '主题', '暗色', '亮色', '深色', '浅色', '模式', '字体', '字号', '密度', '动效', '动画', '强调色', '颜色', '显示', '语言', '界面语言', '英文', '繁体', 'language'],
    panel: (<><AppearancePanel /><DarkModePanel /><FontPanel /><DensityPanel /><MotionPanel /><AccentPanel /><LanguagePanel /></>),
  },
  {
    key: 'chat',
    zone: 'global',
    icon: <MessageOutlined />,
    label: '聊天',
    labelKey: 'settings.cat.chat',
    desc: '伴侣 · 语音',
    descKey: 'settings.cat.chatDesc',
    // 批 2 语音收编：ChatPage 语音设置 Modal 退役，VoiceSettingsPanel（引擎路由/
    // Breeze 克隆/麦克风/确认词）+ ChatVoiceSection（全局默认音色，从 ChatPanel
    // 拆出）并入本组；批 3 增挂 VoiceSection（TTS/ASR/OCR 激活引擎，模型中心迁入）。
    // 独立语音组待 locales 并行线收官建键后一行拆出。
    keywords: ['聊天', '伴侣', '称呼', '性别', '人格', '角色', '语音', '朗读', '音色', '克隆', '麦克风', '确认词', 'breeze', '实时', '对话', '识别', 'tts', 'asr', 'ocr', 'voice'],
    panel: (<>
      <ChatPanel /><ChatVoiceSection /><VoiceSettingsPanel />
      <ModelSettingsProvider><VoiceSection /></ModelSettingsProvider>
    </>),
  },
  {
    key: 'imagegen',
    zone: 'boards',
    icon: <PictureOutlined />,
    label: '绘梦',
    labelKey: 'settings.cat.imagegen',
    desc: '图像后端',
    descKey: 'settings.cat.imagegenDesc',
    keywords: ['绘梦', '图像', '图片', '生成', 'comfyui', '后端', '模型', '保存', 'xai', 'grok-imagine', 'python'],
    // 批 3：模型中心 ImageSection（图像后端+模型选择+ComfyUI 路径）迁入同组，
    // 与 ImageGenPanel 同键同源；与绘梦页快捷开关构成「设置页完整面+原位高频」。
    panel: (<>
      <ImageGenPanel />
      <ModelSettingsProvider><ImageSection /></ModelSettingsProvider>
    </>),
  },
  {
    key: 'office',
    zone: 'boards',
    icon: <SettingOutlined />,
    label: '办公',
    labelKey: 'settings.cat.office',
    desc: '引擎 · 方案',
    descKey: 'settings.cat.officeDesc',
    keywords: ['办公', '工作台', '偏好', '自动展开', '自动弹出', '自动切换', '任务视图', '上下文', '粒度', '排序', '趋势', '浏览器', '文件活动', '引擎', '方案', '模型', '权限', '沙箱', 'agent', '工具', '技能', '热加载', '招标', '撰写', '温度', '记忆', '晨报', '项目本体', '做梦', '面板', 'cad', '图纸', 'mcp', '服务器', 'context7', 'auto-open', 'auto expand', 'context', '网络搜索', '搜索', '联网', '主源', 'tavily', 'brave', 'searxng', 'web'],
    // v4.65 欠账收口：办公工作台「自动展开/自动切换」偏好卡置顶（入口可发现性），
    // 2.5d 补上下文页偏好卡（默认粒度/模式/排序），后接办公引擎设置（config.toml）。
    // 批 4：记忆四态（原右栏 MemoryPanel 胶囊）+ 工位面板启停（原「管理」弹层，
    // CAD 可选卡常驻入口）+ MCP/工具/技能（原 CapabilitiesPanel 抽屉主体）收编。
    panel: (<>
      <WorkbenchPrefsPanel /><ContextPrefsPanel /><OfficePanel /><SearchPanel />
      <OfficeMemorySection />
      <PaneManagementSection
        title="工位面板"
        desc="办公右栏工作台面板启停：已打开的页面不因停用强关，启用集只管入口可见性（CAD 图纸等可选卡在此常驻可开）"
        tabs={WORKSPACE_TABS}
        useStore={usePaneEnabledStore}
      />
      <CapabilitiesInline />
    </>),
  },
  {
    key: 'model',
    zone: 'global',
    icon: <ApiOutlined />,
    label: '模型',
    labelKey: 'settings.cat.model',
    desc: '全局模型',
    descKey: 'settings.cat.modelDesc',
    // 批 3 模型与引擎收编：模型中心设置类 Section（功能绑定/空间策略/本地调度/
    // 引擎管理+Key+自定义引擎/自管引擎生命周期）整体迁入，本组从「跳转空壳」变
    // 完整控制面（ModelPanel 全局默认+汇率置顶）。评测统计（Benchmark/推荐/成本）
    // 留模型中心；同键同源，跨页变更经 model-changed/feature-model-changed 事件互同步。
    keywords: ['模型', '推理', '强度', '引擎', 'grok', 'deepseek', 'herdsman', 'xai', 'api', 'key', '绑定', '功能绑定', '调度', '保活', '预载', '故障转移', '转移', '空间策略', '自定义引擎', '启停', 'strata', 'llama', 'whirl', 'glm', 'opencode'],
    panel: (<>
      <ModelPanel />
      <ModelSettingsProvider>
        <BindSection /><StrategySection /><SchedulingSection /><EngineSection />
      </ModelSettingsProvider>
    </>),
  },
  {
    key: 'security',
    zone: 'global',
    icon: <SafetyCertificateOutlined />,
    label: '安全',
    labelKey: 'settings.cat.security',
    desc: '隐私 · 调试',
    descKey: 'settings.cat.securityDesc',
    keywords: ['安全', '隐私', '敏感', '本地化', '本地', '局域网', '暴露', 'lan', '调试', 'webview', '远程', 'token', '报价', '成本'],
    panel: <SecurityPanel />,
  },
  {
    key: 'data',
    zone: 'global',
    icon: <DatabaseOutlined />,
    label: '数据',
    labelKey: 'settings.cat.data',
    desc: '备份 · 恢复',
    descKey: 'settings.cat.dataDesc',
    keywords: ['数据', '备份', '恢复', '迁移', '导出', '导入', 'zip', '换机', '重装', '防丢', '存档', '快照'],
    panel: <DataPanel />,
  },
  {
    key: 'codeatlas',
    zone: 'global',
    icon: <ClusterOutlined />,
    label: '代码图谱',
    labelKey: 'settings.cat.codeatlas',
    desc: '分层 · 依赖',
    descKey: 'settings.cat.codeatlasDesc',
    keywords: ['代码图谱', '结构', '架构', '依赖', '包', '分层', '内核', '壳层', 'atlas', 'graph', 'code'],
    panel: <CodeAtlasPanel />,
  },
  {
    key: 'about',
    zone: 'global',
    icon: <InfoCircleOutlined />,
    label: '关于',
    labelKey: 'settings.cat.about',
    desc: '版本 · 存储',
    descKey: 'settings.cat.aboutDesc',
    keywords: ['关于', '版本', '更新', '日志', '系统', '信息', '路径', '配置', '存储', 'token', '凭证'],
    panel: <AboutPanel />,
  },

  // ── 板块区（批 5）：板块域设置薄组。label/desc 暂走 zh 兜底（labelKey 缺省），
  //     locales 并行线收官后补三语键回切。进度计划/记忆中枢的显示偏好为
  //     「使用中即时调」型，按不收编边界留域（批 7 索引卡直达）。──
  {
    key: 'wenxue',
    zone: 'boards',
    icon: <ReadOutlined />,
    label: '文学',
    desc: '面板 · 书斋',
    keywords: ['文学', '书斋', '面板', '工作台', '阅读', '小说', '提示词', '分板'],
    panel: (
      <PaneManagementSection
        title="书斋面板"
        desc="文学壳层右栏工作台面板启停（与办公工位各自独立清单）"
        tabs={WENXUE_TABS}
        useStore={useWenxuePaneEnabled}
      />
    ),
  },
  {
    key: 'weixin',
    zone: 'boards',
    icon: <WechatOutlined />,
    label: '微信',
    desc: '回推服务',
    keywords: ['微信', '回推', '提醒', '服务', '监听', '助手', 'wechat'],
    panel: <WeixinServiceSection />,
  },
  {
    key: 'pantheon',
    zone: 'boards',
    icon: <ThunderboltOutlined />,
    label: '众神殿',
    desc: '外部参与者',
    keywords: ['众神殿', '参与者', 'agent', '路径', '可执行', '外部', '讨论'],
    panel: <PantheonAgentsSection />,
  },
]
