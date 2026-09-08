// 由 scripts/gen-herdsman-templates.mjs 生成（模板类别表），勿手改。
// 来源：herdsman.exe embedded template library（.tmp/herdsman_templates.json）
// 瘦身 P3：类别表自 data/herdsmanTemplates.ts 分片至此，主文件 re-export 聚合。
import type { TemplateCategory } from '../imageTemplates'

export const HERDSMAN_CATEGORIES: TemplateCategory[] = [
  { id: "portrait", label: "人像摄影", color: "#ec4899", icon: "📷" }, // hex-exempt 模板数据色
  { id: "poster", label: "海报视觉", color: "#ef4444", icon: "📰" }, // hex-exempt 模板数据色
  { id: "product", label: "产品与静物", color: "#f97316", icon: "🧴" }, // hex-exempt 模板数据色
  { id: "hm-scene", label: "场景氛围", color: "#14b8a6", icon: "🏞" }, // hex-exempt 模板数据色
  { id: "creative", label: "创意实验", color: "#8b5cf6", icon: "✨" }, // hex-exempt 模板数据色
  { id: "illustration", label: "插画艺术", color: "#f59e0b", icon: "🎨" }, // hex-exempt 模板数据色
  { id: "anime", label: "动漫二次元", color: "#f43f5e", icon: "🌸" }, // hex-exempt 模板数据色
  { id: "food", label: "美食摄影", color: "#eab308", icon: "🍜" }, // hex-exempt 模板数据色
  { id: "architecture", label: "建筑空间", color: "#0ea5e9", icon: "🏛️" }, // hex-exempt 模板数据色
  { id: "fashion", label: "时尚潮流", color: "#d946ef", icon: "👗" }, // hex-exempt 模板数据色
  { id: "game", label: "游戏美术", color: "#6366f1", icon: "🎮" }, // hex-exempt 模板数据色
  { id: "ui", label: "UI 界面", color: "#10b981", icon: "🧩" }, // hex-exempt 模板数据色
]