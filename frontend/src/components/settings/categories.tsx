import React from 'react'
import {
  AppstoreOutlined, MessageOutlined, PictureOutlined,
  SettingOutlined, ApiOutlined, InfoCircleOutlined,
  SafetyCertificateOutlined, DatabaseOutlined, ClusterOutlined,
  ReadOutlined, WechatOutlined, ThunderboltOutlined, CodeOutlined,
} from '@ant-design/icons'
import type { DictKey } from '../../gaea/locales/en'
import WeixinServiceSection from './WeixinServiceSection'
import PantheonAgentsSection from './PantheonAgentsSection'
import AdvancedSection from './AdvancedSection'
import BoardIndexSection from './BoardIndexSection'
import { GaeaScheduleIcon, GaeaMemoryIcon } from '../../boards/boardIcons'
import CodeAtlasPanel from './CodeAtlasPanel'
import AboutPanel from './AboutPanel'
import {
  ChatSettingsPanel,
  DataSettingsPanel,
  GeneralSettingsPanel,
  ImageSettingsPanel,
  ModelSettingsPanel,
  OfficeSettingsPanel,
  SecuritySettingsPanel,
  WenxueSettingsPanel,
} from './DenseSettingsPanels'

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
  /** 静态面板，或按全局搜索词返回面板（密集组据此自动切换内部任务页签）。 */
  panel: React.ReactNode | ((
    query: string,
    initialTab?: string,
    onTabChange?: (tab: string) => void,
  ) => React.ReactNode)
}

/** 导航分区（尾批回切：zone/组名三语键已入 locales，zh 兜底退役——
 *  label 仍留作搜索关键词与键缺失时的最后兜底）。 */
export const SETTINGS_ZONES: { key: SettingsZone; label: string; labelKey: DictKey }[] = [
  { key: 'global', label: '全局', labelKey: 'settings.zone.global' },
  { key: 'boards', label: '板块', labelKey: 'settings.zone.boards' },
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
    panel: (query, tab, onTabChange) => (
      <GeneralSettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
  },
  {
    key: 'chat',
    zone: 'global',
    icon: <MessageOutlined />,
    label: '聊天',
    labelKey: 'settings.cat.chat',
    desc: '伴侣 · 语音',
    descKey: 'settings.cat.chatDesc',
    // 聊天组的四个旧设置面按任务拆为内部页签：角色与人格 / 默认音色 /
    // 朗读引擎 / 语音模型。状态保留在当前面板内，不再纵向堆叠。
    keywords: ['聊天', '伴侣', '称呼', '性别', '人格', '角色', '语音', '朗读', '音色', '克隆', '麦克风', '确认词', 'breeze', '实时', '对话', '识别', 'tts', 'asr', 'ocr', 'voice'],
    panel: (query, tab, onTabChange) => (
      <ChatSettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
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
    // 图像生成配置与底层后端/模型分为两个任务页签，同键同源。
    panel: (query, tab, onTabChange) => (
      <ImageSettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
  },
  {
    key: 'office',
    zone: 'boards',
    icon: <SettingOutlined />,
    label: '办公',
    labelKey: 'settings.cat.office',
    desc: '引擎 · 方案',
    descKey: 'settings.cat.officeDesc',
    keywords: ['办公', '工作台', '偏好', '自动展开', '自动弹出', '自动切换', '任务视图', '上下文', '粒度', '排序', '趋势', '浏览器', '文件活动', '引擎', '方案', '模型', '权限', '沙箱', 'agent', '工具', '技能', '热加载', '招标', '撰写', '温度', '记忆', '晨报', '项目本体', '做梦', '面板', 'cad', '图纸', 'mcp', '服务器', 'context7', 'auto-open', 'auto expand', 'context'],
    // 办公设置按工作台 / 办公引擎 / 记忆与面板 / 工具与技能四域分栏。
    panel: (query, tab, onTabChange) => (
      <OfficeSettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
  },
  {
    key: 'model',
    zone: 'global',
    icon: <ApiOutlined />,
    label: '模型',
    labelKey: 'settings.cat.model',
    desc: '全局模型',
    descKey: 'settings.cat.modelDesc',
    // 模型组承载模型中心迁出的低频配置：概览 / 模型库 / 选型参考 / 质量与成本 / 运行策略。
    // 高频操作（模型一览、功能绑定、引擎管理）只保留在模型中心，避免双入口漂移。
    keywords: ['模型', '推理', '强度', 'grok', 'deepseek', 'herdsman', 'xai', '模型库', '目录', '选型', '推荐', '量化', '测评', '质量', '检索', '成本', '归因', '调度', '保活', '预载', '故障转移', '空间策略', '权限', '护栏', 'glm', 'opencode'],
    panel: (query, tab, onTabChange) => (
      <ModelSettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
  },
  {
    key: 'security',
    zone: 'global',
    icon: <SafetyCertificateOutlined />,
    label: '安全',
    labelKey: 'settings.cat.security',
    desc: '隐私 · 网络',
    descKey: 'settings.cat.securityDesc',
    // 批 6 重组：SearchPanel（网络搜索主源/Key）从办公组归位本组=隐私与网络同域
    //（desc i18n 键文案「隐私 · 调试」待 locales 解锁后随新键更新）。
    keywords: ['安全', '隐私', '敏感', '本地化', '本地', '局域网', '暴露', 'lan', '调试', 'webview', '远程', 'token', '报价', '成本', '网络搜索', '搜索', '联网', '主源', 'tavily', 'brave', 'searxng'],
    panel: (query, tab, onTabChange) => (
      <SecuritySettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
  },
  {
    key: 'data',
    zone: 'global',
    icon: <DatabaseOutlined />,
    label: '数据',
    labelKey: 'settings.cat.data',
    desc: '备份 · 恢复',
    descKey: 'settings.cat.dataDesc',
    // 批 6 归位：ChatDataSection（聊天历史 ZIP 导出/导入）从 chat 组归位本组
    //（数据操作与备份恢复同域）。
    keywords: ['数据', '备份', '恢复', '迁移', '导出', '导入', 'zip', '换机', '重装', '防丢', '存档', '快照', '聊天', '历史'],
    panel: (query, tab, onTabChange) => (
      <DataSettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
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
    labelKey: 'settings.cat.wenxue',
    desc: '人格 · 面板',
    descKey: 'settings.cat.wenxueDesc',
    keywords: ['文学', '书斋', '面板', '工作台', '阅读', '小说', '提示词', '人格', '分板', 'system_prompt'],
    // 尾批：身份提示词双键收编（此前仅手改 ga.cfg，CoreB WenxuePromptGet/Set 绑对）
    panel: (query, tab, onTabChange) => (
      <WenxueSettingsPanel query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
  },
  {
    key: 'weixin',
    zone: 'boards',
    icon: <WechatOutlined />,
    label: '微信',
    labelKey: 'settings.cat.weixin',
    desc: '回推服务',
    descKey: 'settings.cat.weixinDesc',
    keywords: ['微信', '回推', '提醒', '服务', '监听', '助手', 'wechat'],
    panel: <WeixinServiceSection />,
  },
  {
    key: 'pantheon',
    zone: 'boards',
    icon: <ThunderboltOutlined />,
    label: '众神殿',
    labelKey: 'settings.cat.pantheon',
    desc: '外部参与者',
    descKey: 'settings.cat.pantheonDesc',
    keywords: ['众神殿', '参与者', 'agent', '路径', '可执行', '外部', '讨论'],
    panel: <PantheonAgentsSection />,
  },
  {
    key: 'schedule',
    zone: 'boards',
    icon: <GaeaScheduleIcon />,
    label: '进度计划',
    labelKey: 'settings.cat.schedule',
    desc: '网络图样式 · 项目设置',
    descKey: 'settings.cat.scheduleDesc',
    keywords: ['进度计划', '进度', '网络图', '图上内容', '节点时间', '工期', '六参数', '起止时间', '关系线', '横道', '甘特', '日历', '工作日历', '基线', '检视', '资源与成本', '项目设置', 'schedule'],
    // 批 7 拍板挂账收口：进度板块显示/样式偏好属「使用中即时调」型，
    // 边界留域（偏好面留驻板块=单一事实源），设置页给索引卡直达。
    panel: (
      <BoardIndexSection
        title="进度计划偏好（板块内即时调）"
        desc="进度板块的显示与样式偏好留驻板块内（单一事实源），设置页不复制副本；下方为板块内真实入口的索引直达。"
        items={[
          '网络图 · 图上内容（节点时间/工期/六参数/起止时间）',
          '关系线布局与显隐',
          '工作日历',
          '检视 · 基线对比 · 资源与成本',
          '项目设置',
        ]}
        page="schedule"
        openLabel="打开进度计划"
      />
    ),
  },
  {
    key: 'memoryhub',
    zone: 'boards',
    icon: <GaeaMemoryIcon />,
    label: '记忆中枢',
    labelKey: 'settings.cat.memoryhub',
    desc: '星图显示设置',
    descKey: 'settings.cat.memoryhubDesc',
    keywords: ['记忆中枢', '记忆', '星图', '显示设置', '轨道环', '星尘', '自动旋转', '脑网格', '知识库', '画像', '记忆图谱', '关系', 'memory', 'atlas'],
    // 同上：星图显示设置属「使用中即时调」型，边界留域+索引卡直达。
    panel: (
      <BoardIndexSection
        title="记忆中枢偏好（板块内即时调）"
        desc="星图显示设置留驻记忆中枢（单一事实源），设置页不复制副本；下方为板块内真实入口的索引直达。"
        items={[
          '星图显示设置：轨道环 / 星尘 / 自动旋转',
          '交互：拖拽旋转 · 滚轮缩放 · 点击星辰',
        ]}
        page="memoryhub"
        openLabel="打开记忆中枢"
      />
    ),
  },
  {
    key: 'advanced',
    zone: 'global',
    icon: <CodeOutlined />,
    label: '高级',
    labelKey: 'settings.cat.advanced',
    desc: '路径 · 超时 · 兜底',
    descKey: 'settings.cat.advancedDesc',
    keywords: ['高级', '路径', '引擎路径', 'strata', 'whirl', 'breeze', 'comfyui', '空闲退出', '超时', '读屏', '意图', '兜底', 'glm', '目录', 'config', '配置文件'],
    // 尾批：无 UI config 键收编（读 AdvancedConfigGet/写 SaveConfig 宽写面）；
    // 本轮再按运行策略与路径目录拆为内部任务页签。
    panel: (query, tab, onTabChange) => (
      <AdvancedSection query={query} initialTab={tab} onTabChange={onTabChange} />
    ),
  },
]
