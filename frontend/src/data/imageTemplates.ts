export interface Template {
  label: string
  prompt: string
  negative: string
}

/** 用户自定义模板 */
export interface CustomTemplate extends Template {
  id: string
}

// ── 预设模板（4大类，含原风格中独有项） ──

export const TEMPLATES: Record<string, Template[]> = {
  '📕 创作类': [
    { label: '📕 小说封面', prompt: '精致小说封面设计，专业排版构图，电影级光影，浮雕标题质感，8K超高清', negative: '低质量, 模糊, 文字扭曲, 错别字, 非中文' },
    { label: '👤 角色肖像', prompt: '精致角色肖像，柔和棚拍光线，半身构图，精致的面部细节，8K超清', negative: '变形手指, 多余肢体, 低分辨率, 模糊' },
    { label: '🏔️ 场景概念', prompt: '史诗级场景概念艺术，广阔视野，戏剧性光影，氛围感强，丰富细节', negative: '模糊, 平面感, 简陋, 低质量' },
    { label: '🎨 插画内页', prompt: '精美插画风格，细腻线条，柔和色调，叙事性构图，艺术感', negative: '照片感, 3D渲染感, 低分辨率' },
    { label: '📖 章节插图', prompt: '小说章节插图，与文字搭配的叙事画面，适合排版，留白设计，精致', negative: '混乱, 模糊, 低质量, 过度复杂' },
  ],
  '📸 写实类': [
    { label: '📸 写实摄影', prompt: '写实摄影风格，自然光线，超高分辨率，逼真质感，细节丰富，8K', negative: '模糊, 低质量, 动漫风格, 绘画质感, 3D渲染感' },
    { label: '🎬 电影剧照', prompt: '电影剧照风格，宽银幕构图，戏剧性光影，胶片质感，电影级调色', negative: '模糊, 低质量, 平面感, 电视画质' },
    { label: '👗 时尚大片', prompt: '时尚杂志大片风格，精致妆造，高级灯光，完美构图，奢侈品质感', negative: '模糊, 低质量, 廉价感, 过时服装' },
    { label: '🔍 微距特写', prompt: '微距摄影特写，极致细节，浅景深，柔美散景，清晰焦点，8K超清', negative: '模糊, 大景深, 噪点, 低分辨率' },
    { label: '🚶 纪实街拍', prompt: '纪实街拍风格，自然抓拍感，真实光影，生活气息，人文情怀', negative: '摆拍感, 过度修饰, 滤镜感, 3D渲染' },
  ],
  '🎨 风格类': [
    { label: '🖌️ 中国水墨', prompt: '中国水墨画风格，写意笔触，留白构图，淡雅墨色，传统韵味', negative: '色彩鲜艳, 油画质感, 照片感, 西式构图' },
    { label: '🎪 日系动漫', prompt: '日系动漫风格，鲜艳色彩，精致角色，明亮光影，干净线条', negative: '写实感, 照片感, 暗黑风格, 3D渲染' },
    { label: '🌆 赛博朋克', prompt: '赛博朋克风格，霓虹灯光，雨夜都市，高科技感，蓝紫冷色调', negative: '自然风光, 暖色调, 古代风格, 简陋' },
    { label: '🧙 奇幻风格', prompt: '奇幻艺术风格，魔法氛围，史诗感，精细纹理，丰富想象力', negative: '现代元素, 科技感, 简约风格, 照片感' },
    { label: '🎨 数字油画', prompt: '数字油画风格，电影级光影，高细节，8K，丰富色彩层次', negative: '模糊, 低质量, 线稿, 扁平风格' },
    { label: '🖊️ 线稿插画', prompt: '精致线稿风格，干净利落的线条，扁平色彩，插画风', negative: '模糊, 3D渲染, 照片感, 过度写实' },
    { label: '🌌 概念艺术', prompt: '概念艺术风格，史诗级场景，戏剧性光影，氛围感强，想象力丰富', negative: '照片感, 写实感, 平淡, 简陋' },
  ],
  '📐 构图类': [
    { label: '🧍 半身肖像', prompt: '精致半身肖像构图，柔光，干净背景，人物突出，8K超清', negative: '全身, 多人, 混乱背景, 低分辨率' },
    { label: '🧎 全身立绘', prompt: '全身角色立绘，完整人物展示，清晰服装设计，白色背景，角色设计图', negative: '半身, 裁切, 模糊面部, 复杂背景' },
    { label: '🌄 广角场景', prompt: '广角场景构图，宏大视野，深远景深，丰富层次，电影级', negative: '特写, 浅景深, 扁平, 低分辨率' },
    { label: '🚁 俯瞰航拍', prompt: '俯瞰航拍视角，上帝视角，全景构图，丰富细节，震撼', negative: '仰视, 平视, 特写, 低清晰度' },
    { label: '👁️ 特写面部', prompt: '面部大特写，极致细节，柔光，眼神表达，8K超清', negative: '全身, 远景, 遮挡面部, 模糊' },
  ],
}

// ── 自定义模板 ──

const STORAGE_KEY = 'wubigork-image-templates'

export function loadCustomTemplates(): CustomTemplate[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
  } catch {
    return []
  }
}

export function saveCustomTemplates(templates: CustomTemplate[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(templates))
  } catch {
    // localStorage 满或不可用
  }
}

export function generateTemplateId(): string {
  return `custom_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`
}

// ── 将所有类别名导出为数组 ──

export const CATEGORIES: string[] = [
  '📕 创作类',
  '📸 写实类',
  '🎨 风格类',
  '📐 构图类',
]

/** 如果用户有自定义模板，追加自定义类别 */
export function getAllCategories(customCount: number): string[] {
  const cats = [...CATEGORIES]
  if (customCount > 0) cats.push('⭐ 自定义')
  return cats
}
