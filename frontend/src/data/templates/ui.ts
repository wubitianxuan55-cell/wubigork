// 由 scripts/gen-herdsman-templates.mjs 生成（ui 类模板），勿手改。
// 来源：herdsman.exe embedded template library（.tmp/herdsman_templates.json）
// 瘦身 P3：自 data/herdsmanTemplates.ts 分片，主文件按类别聚合 re-export。
import type { Template } from '../imageTemplates'

export const UI_TEMPLATES: Template[] = [
  {
    id: "ui:AI 图片生成器界面",
    icon: "🧩",
    label: "AI 图片生成器界面",
    description: "深色、参数面板、预览区",
    prompt: "AI 图片生成器应用界面设计，左侧是 prompt 输入框和模型参数面板，右侧是四宫格图片预览，顶部有模型选择和历史入口，深色主题，霓虹点缀，布局清晰专业",
  },
  {
    id: "ui:健康数据仪表盘",
    icon: "🧩",
    label: "健康数据仪表盘",
    description: "图表、卡片、浅色商务",
    prompt: "健康管理仪表盘 UI，展示睡眠、心率、步数和压力趋势，使用折线图、环形进度和统计卡片，浅色背景，蓝绿配色，信息密度适中，适合桌面端",
  },
  {
    id: "ui:电商商品详情页",
    icon: "🧩",
    label: "电商商品详情页",
    description: "移动端、购买转化、干净",
    prompt: "移动端电商商品详情页 UI，顶部是大图轮播，中间展示价格、规格选择、评价摘要和优惠券，底部固定购买按钮，白色背景，品牌色点缀，转化导向清晰",
  },
  {
    id: "ui:金融资产看板",
    icon: "🧩",
    label: "金融资产看板",
    description: "暗色、资产曲线、安全感",
    prompt: "个人金融资产看板 UI，暗色主题，顶部展示总资产和收益率，中间是资产曲线图，下方是基金、股票、现金分类列表，使用稳重蓝绿配色，专业安全感",
  },
  {
    id: "ui:音乐播放器界面",
    icon: "🧩",
    label: "音乐播放器界面",
    description: "专辑封面、歌词、沉浸",
    prompt: "移动端音乐播放器 UI，大幅专辑封面作为背景虚化，中央显示唱片封面和播放控件，下方有歌词滚动和播放列表入口，色彩根据专辑封面自适应，沉浸感强",
  },
  {
    id: "ui:旅行规划应用",
    icon: "🧩",
    label: "旅行规划应用",
    description: "地图、行程、卡片",
    prompt: "旅行规划应用 UI，左侧为城市地图和路线点，右侧为每日行程卡片列表，包含景点、餐厅和交通时间，浅色主题，橙蓝配色，适合桌面网页工具",
  },
  {
    id: "ui:图标套装",
    icon: "🧩",
    label: "图标套装",
    description: "线性、工具类、统一风格",
    prompt: "一套线性风工具类图标设计，包含上传、下载、裁剪、画笔、图层、放大镜、设置、删除和收藏，统一 24px 网格，圆角线端，黑白版本，适合设计系统",
  },
  {
    id: "ui:SaaS 团队设置页",
    icon: "🧩",
    label: "SaaS 团队设置页",
    description: "表格、权限、企业工具",
    prompt: "SaaS 团队设置页面 UI，左侧导航包含成员、权限、账单和安全，主区域是成员表格、角色下拉和邀请按钮，布局紧凑，浅色企业风，适合高频操作",
  },
]