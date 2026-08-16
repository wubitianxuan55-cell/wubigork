// 从 herdsman 嵌入式模板库导出文件（.tmp/herdsman_templates.json）生成前端数据模块。
// 用法：node scripts/gen-herdsman-templates.mjs
// 产物：frontend/src/data/herdsmanTemplates.ts（勿手改，herdsman 库更新后重新生成）
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const src = join(root, '.tmp', 'herdsman_templates.json')
const out = join(root, 'frontend', 'src', 'data', 'herdsmanTemplates.ts')

const data = JSON.parse(readFileSync(src, 'utf8'))

// 与内置分类冲突的 id 加前缀，避免合并库后互相覆盖（内置：enhance/style/photo/creation/compose/light/scene）
const CORE_IDS = new Set(['enhance', 'style', 'photo', 'creation', 'compose', 'light', 'scene'])

// 分类中文名与主题色（emoj 沿用 herdsman 自带）
const CATEGORY_META = {
  portrait: { label: '人像摄影', color: '#ec4899' },
  poster: { label: '海报视觉', color: '#ef4444' },
  product: { label: '产品与静物', color: '#f97316' },
  scene: { label: '场景氛围', color: '#14b8a6' },
  creative: { label: '创意实验', color: '#8b5cf6' },
  illustration: { label: '插画艺术', color: '#f59e0b' },
  anime: { label: '动漫二次元', color: '#f43f5e' },
  food: { label: '美食摄影', color: '#eab308' },
  architecture: { label: '建筑空间', color: '#0ea5e9' },
  fashion: { label: '时尚潮流', color: '#d946ef' },
  game: { label: '游戏美术', color: '#6366f1' },
  ui: { label: 'UI 界面', color: '#10b981' },
}

const cats = []
for (const c of data.categories || []) {
  const meta = CATEGORY_META[c.category] || {}
  const id = CORE_IDS.has(c.category) ? `hm-${c.category}` : c.category
  cats.push({ id, label: meta.label || c.category, color: meta.color || '#94a3b8', icon: c.emoji || '' })
}

const lines = []
lines.push('// 由 scripts/gen-herdsman-templates.mjs 从 herdsman 嵌入式模板库生成，勿手改。')
lines.push('// 来源：herdsman.exe embedded template library（.tmp/herdsman_templates.json）')
lines.push("import type { Template, TemplateCategory } from './imageTemplates'")
lines.push('')
lines.push('export const HERDSMAN_CATEGORIES: TemplateCategory[] = [')
for (const c of cats) {
  lines.push(`  { id: ${JSON.stringify(c.id)}, label: ${JSON.stringify(c.label)}, color: ${JSON.stringify(c.color)}, icon: ${JSON.stringify(c.icon)} },`)
}
lines.push(']')
lines.push('')
lines.push('export const HERDSMAN_TEMPLATES: Record<string, Template[]> = {')
for (const group of data.templates || []) {
  const key = CORE_IDS.has(group.category) ? `hm-${group.category}` : group.category
  lines.push(`  ${JSON.stringify(key)}: [`)
  for (const t of group.items || []) {
    lines.push('    {')
    lines.push(`      id: ${JSON.stringify(t.id)},`)
    lines.push(`      icon: ${JSON.stringify(t.icon || '')},`)
    lines.push(`      label: ${JSON.stringify(t.zhTitle)},`)
    lines.push(`      description: ${JSON.stringify(t.zhDescription || '')},`)
    lines.push(`      prompt: ${JSON.stringify(t.zhPrompt)},`)
    lines.push('    },')
  }
  lines.push('  ],')
}
lines.push('}')
lines.push('')

writeFileSync(out, lines.join('\n'), 'utf8')
console.log(`已生成 ${out}（${cats.length} 类 / ${data.templates?.reduce((n, g) => n + (g.items?.length || 0), 0) || 0} 个模板）`)
