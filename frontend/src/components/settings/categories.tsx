import React from 'react'
import {
  AppstoreOutlined, MessageOutlined, PictureOutlined,
  SettingOutlined, ApiOutlined, InfoCircleOutlined,
  SafetyCertificateOutlined, DatabaseOutlined, ClusterOutlined,
} from '@ant-design/icons'
import type { DictKey } from '../../gaea/locales/en'
import AppearancePanel, { DarkModePanel, FontPanel, DensityPanel, MotionPanel, AccentPanel, LanguagePanel } from './AppearancePanel'
import ChatPanel from './ChatPanel'
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
  /** zh 兜底文案（搜索关键词用，数据层不可 use hook） */
  label: string
  desc: string
  /** i18n 渲染 key */
  labelKey: DictKey
  descKey: DictKey
  keywords: string[]
  panel: React.ReactNode
}

/** 导航分区（批 1 仅结构就位：board 区为空时整区不渲染，单 zone 不出标题）。
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
    keywords: ['聊天', '伴侣', '称呼', '性别', '人格', '角色', '语音', '朗读', '音色', '对话', '识别', 'tts', 'asr'],
    panel: <ChatPanel />,
  },
  {
    key: 'imagegen',
    zone: 'global',
    icon: <PictureOutlined />,
    label: '绘梦',
    labelKey: 'settings.cat.imagegen',
    desc: '图像后端',
    descKey: 'settings.cat.imagegenDesc',
    keywords: ['绘梦', '图像', '图片', '生成', 'comfyui', '后端', '模型', '保存', 'xai', 'grok-imagine'],
    panel: <ImageGenPanel />,
  },
  {
    key: 'office',
    zone: 'global',
    icon: <SettingOutlined />,
    label: '办公',
    labelKey: 'settings.cat.office',
    desc: '引擎 · 方案',
    descKey: 'settings.cat.officeDesc',
    keywords: ['办公', '工作台', '偏好', '自动展开', '自动弹出', '自动切换', '任务视图', '上下文', '粒度', '排序', '趋势', '浏览器', '文件活动', '引擎', '方案', '模型', '权限', '沙箱', 'agent', '工具', '技能', '热加载', '招标', '撰写', '温度', 'auto-open', 'auto expand', 'context', '网络搜索', '搜索', '联网', '主源', 'tavily', 'brave', 'searxng', 'web'],
    // v4.65 欠账收口：办公工作台「自动展开/自动切换」偏好卡置顶（入口可发现性），
    // 2.5d 补上下文页偏好卡（默认粒度/模式/排序），后接办公引擎设置（config.toml）。
    panel: (<><WorkbenchPrefsPanel /><ContextPrefsPanel /><OfficePanel /><SearchPanel /></>),
  },
  {
    key: 'model',
    zone: 'global',
    icon: <ApiOutlined />,
    label: '模型',
    labelKey: 'settings.cat.model',
    desc: '全局模型',
    descKey: 'settings.cat.modelDesc',
    keywords: ['模型', '推理', '强度', '引擎', 'grok', 'deepseek', 'herdsman', 'xai', 'api', 'key'],
    panel: <ModelPanel />,
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
]
