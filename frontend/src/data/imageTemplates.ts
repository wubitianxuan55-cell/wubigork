export interface Template {
  label: string
  prompt: string
  negative: string
}

/** 用户自定义模板 */
export interface CustomTemplate extends Template {
  id: string
}

// ── 预设模板（6大类，含优化提示词） ──

export const TEMPLATES: Record<string, Template[]> = {
  '📕 创作类': [
    { label: '📕 小说封面', prompt: '精致小说封面设计，专业排版构图，电影级光影，浮雕标题质感，8K超高清', negative: '低质量, 模糊, 文字扭曲, 错别字, 非中文' },
    { label: '👤 角色肖像', prompt: '精致角色肖像，柔和棚拍光线，半身构图，精致的面部细节，8K超清', negative: '变形手指, 多余肢体, 低分辨率, 模糊' },
    { label: '🏔️ 场景概念', prompt: '史诗级场景概念艺术，广阔视野，戏剧性光影，氛围感强，丰富细节', negative: '模糊, 平面感, 简陋, 低质量' },
    { label: '🎨 插画内页', prompt: '精美插画风格，细腻线条，柔和色调，叙事性构图，艺术感', negative: '照片感, 3D渲染感, 低分辨率' },
    { label: '📖 章节插图', prompt: '小说章节插图，与文字搭配的叙事画面，适合排版，留白设计，精致', negative: '混乱, 模糊, 低质量, 过度复杂' },
    { label: '🐉 奇幻生物', prompt: '奇幻生物设计，精细鳞片/羽毛纹理，动态姿态，魔法氛围，史诗感', negative: '模糊, 比例失调, 低质量, 现代元素' },
    { label: '⚔️ 动作场景', prompt: '动态动作场景，激烈战斗或追逐，运动模糊效果，戏剧性构图，电影级光影', negative: '静态, 僵硬, 模糊, 低对比度' },
  ],
  '📸 写实类': [
    { label: '📸 写实摄影', prompt: '写实摄影风格，自然光线，超高分辨率，逼真质感，细节丰富，8K', negative: '模糊, 低质量, 动漫风格, 绘画质感, 3D渲染感' },
    { label: '🎬 电影剧照', prompt: '电影剧照风格，宽银幕构图，戏剧性光影，胶片质感，电影级调色', negative: '模糊, 低质量, 平面感, 电视画质' },
    { label: '👗 时尚大片', prompt: '时尚杂志大片风格，精致妆造，高级灯光，完美构图，奢侈品质感', negative: '模糊, 低质量, 廉价感, 过时服装' },
    { label: '🔍 微距特写', prompt: '微距摄影特写，极致细节，浅景深，柔美散景，清晰焦点，8K超清', negative: '模糊, 大景深, 噪点, 低分辨率' },
    { label: '🚶 纪实街拍', prompt: '纪实街拍风格，自然抓拍感，真实光影，生活气息，人文情怀', negative: '摆拍感, 过度修饰, 滤镜感, 3D渲染' },
    { label: '🍜 美食摄影', prompt: '精致美食摄影，诱人色泽，蒸汽或光泽质感，自然侧光，浅景深，高级餐厅摆盘', negative: '模糊, 低质量, 不新鲜, 凌乱, 廉价餐具' },
    { label: '🏛️ 建筑空间', prompt: '建筑空间摄影，完美对称或引导线构图，自然光与室内光融合，空间纵深感，专业级', negative: '模糊, 畸变, 杂乱, 低光, 手机画质' },
    { label: '🌿 自然风光', prompt: '壮丽自然风光摄影，黄金时刻光线，丰富色彩层次，广阔景深，国家地理风格', negative: '城市, 人造物, 模糊, 过曝, 低饱和度' },
  ],
  '🎨 风格类': [
    { label: '🖌️ 中国水墨', prompt: '中国水墨画风格，写意笔触，留白构图，淡雅墨色，传统韵味', negative: '色彩鲜艳, 油画质感, 照片感, 西式构图' },
    { label: '🎪 日系动漫', prompt: '日系动漫风格，鲜艳色彩，精致角色，明亮光影，干净线条', negative: '写实感, 照片感, 暗黑风格, 3D渲染' },
    { label: '🌆 赛博朋克', prompt: '赛博朋克风格，霓虹灯光，雨夜都市，高科技感，蓝紫冷色调', negative: '自然风光, 暖色调, 古代风格, 简陋' },
    { label: '🧙 奇幻风格', prompt: '奇幻艺术风格，魔法氛围，史诗感，精细纹理，丰富想象力', negative: '现代元素, 科技感, 简约风格, 照片感' },
    { label: '🎨 数字油画', prompt: '数字油画风格，电影级光影，高细节，8K，丰富色彩层次', negative: '模糊, 低质量, 线稿, 扁平风格' },
    { label: '🖊️ 线稿插画', prompt: '精致线稿风格，干净利落的线条，扁平色彩，插画风', negative: '模糊, 3D渲染, 照片感, 过度写实' },
    { label: '🌌 概念艺术', prompt: '概念艺术风格，史诗级场景，戏剧性光影，氛围感强，想象力丰富', negative: '照片感, 写实感, 平淡, 简陋' },
    { label: '🕯️ 暗黑美学', prompt: '暗黑美学风格，低调光影，戏剧性明暗对比，深色调，神秘氛围，巴洛克元素', negative: '明亮, 鲜艳色彩, 可爱风格, 简约' },
    { label: '🎭 复古胶片', prompt: '复古胶片风格，褪色质感，颗粒感，暖黄调，Vintage色调，70年代美学', negative: '现代感, 数码感, 冷色调, 过于锐利' },
    { label: '🤖 机甲科幻', prompt: '科幻机甲风格，精密机械结构，金属质感，未来感，动态战斗姿态，高细节', negative: '模糊, 有机生物感, 中世纪, 简陋设计' },
    { label: '💎 极简主义', prompt: '极简主义风格，大面积留白，几何构图，单一主体，高级灰调配色，建筑感', negative: '复杂, 杂乱, 鲜艳, 过度装饰, 繁荣' },
  ],
  '📐 构图类': [
    { label: '🧍 半身肖像', prompt: '精致半身肖像构图，柔光，干净背景，人物突出，8K超清', negative: '全身, 多人, 混乱背景, 低分辨率' },
    { label: '🧎 全身立绘', prompt: '全身角色立绘，完整人物展示，清晰服装设计，白色背景，角色设计图', negative: '半身, 裁切, 模糊面部, 复杂背景' },
    { label: '🌄 广角场景', prompt: '广角场景构图，宏大视野，深远景深，丰富层次，电影级', negative: '特写, 浅景深, 扁平, 低分辨率' },
    { label: '🚁 俯瞰航拍', prompt: '俯瞰航拍视角，上帝视角，全景构图，丰富细节，震撼', negative: '仰视, 平视, 特写, 低清晰度' },
    { label: '👁️ 特写面部', prompt: '面部大特写，极致细节，柔光，眼神表达，8K超清', negative: '全身, 远景, 遮挡面部, 模糊' },
    { label: '🔄 双人互动', prompt: '双人互动构图，自然姿态，情感交流，平衡的画面分布，故事感', negative: '单人, 僵硬, 无互动, 比例失调' },
    { label: '📱 竖屏适配', prompt: '9:16竖屏构图，主体居中偏上，适合手机壁纸，干净背景，留白设计', negative: '横构图, 主体偏离, 复杂背景, 裁切' },
  ],
  '🏷️ 产品商业': [
    { label: '💎 产品白底', prompt: '商业产品摄影，纯白背景，专业棚拍灯光，清晰产品细节，电商标准', negative: '阴影过重, 模糊, 杂色背景, 低分辨率' },
    { label: '🎁 场景展示', prompt: '产品场景展示，自然生活化布景，柔和窗光，生活方式摄影，高级感', negative: '白色背景, 棚拍感, 产品过小, 杂乱' },
    { label: '💄 美妆特写', prompt: '美妆产品特写，精致布光，微距质感，水珠或光泽细节，奢侈品牌调性', negative: '模糊, 廉价感, 粗糙, 低光' },
    { label: '🏠 室内设计', prompt: '室内设计效果图，自然采光，高级材质纹理，空间层次感，建筑可视化，8K', negative: '模糊, 低质量材质, 空间狭小, 杂乱' },
  ],
  '✨ 万能增强': [
    { label: '⭐ 通用高质量', prompt: '高质量杰作，极致细节，专业级构图，电影光影，清晰锐利，8K超高清', negative: '低质量, 模糊, 噪点, 变形, 多余肢体' },
    { label: '💡 光影增强', prompt: '戏剧性电影光影，强烈明暗对比，体积光，柔和高光，丰富阴影层次', negative: '平面光, 过曝, 死黑, 无阴影' },
    { label: '🎯 细节增强', prompt: '极致细节表现，清晰纹理质感，毛孔级清晰度，微距级别精度，超级分辨率', negative: '模糊, 涂抹感, 噪点, 低分辨率' },
    { label: '🌈 色彩增强', prompt: '丰富鲜艳的色彩，电影级调色，和谐配色，色彩层次丰富，视觉冲击力强', negative: '灰暗, 褪色, 单色调, 饱和度不足' },
  ],
}

// ── 自定义模板 ──

const STORAGE_KEY = 'gaea-image-templates'
const LEGACY_STORAGE_KEY = 'wubigork-image-templates'

export function loadCustomTemplates(): CustomTemplate[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY) ?? localStorage.getItem(LEGACY_STORAGE_KEY)
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
  '🏷️ 产品商业',
  '✨ 万能增强',
]

/** 如果用户有自定义模板，追加自定义类别 */
export function getAllCategories(customCount: number): string[] {
  const cats = [...CATEGORIES]
  if (customCount > 0) cats.push('⭐ 自定义')
  return cats
}
