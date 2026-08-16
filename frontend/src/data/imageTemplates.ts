// imageTemplates.ts — 绘梦结构化提示词模板库（内置 7 类 79 个 + herdsman 12 类 152 个）+ 自定义模板管理
// 提示词统一按「风格 → 主体 → 环境 → 光线 → 构图 → 质量」结构化书写
import { HERDSMAN_CATEGORIES, HERDSMAN_TEMPLATES } from './herdsmanTemplates'

export interface Template {
  /** 稳定标识（herdsman 模板为 "category:名称"，自定义模板为 custom_*） */
  id?: string
  /** 分类图标（herdsman 模板自带的 emoji） */
  icon?: string
  label: string            // 模板名称（无 emoji）
  description?: string     // 一句话用途说明
  prompt: string           // 正面提示词
  negative?: string        // 负面提示词
  size?: string            // 推荐画幅：1:1 / 16:9 / 9:16 / 3:4 / 4:3 / 2:3
  tags?: string[]          // 维度标签：风格 / 光影 / 构图 / 用途
}

/** 用户自定义模板 */
export interface CustomTemplate extends Template {
  id: string
}

/** 分类元数据（主题色用于分类辨识，非 emoji） */
export interface TemplateCategory {
  id: string
  label: string
  color: string
  /** 分类图标（可选，herdsman 分类自带 emoji） */
  icon?: string
}

export const CUSTOM_CATEGORY_ID = 'custom'
/** 「全部模板」虚拟分类（模板库入口，浏览/搜索全量） */
export const ALL_CATEGORY_ID = 'all'
export const ALL_CATEGORY: TemplateCategory = { id: ALL_CATEGORY_ID, label: '全部模板', color: '#94a3b8' }

// ─── 分类：7 大类，各配主题色 ───

export const CORE_CATEGORIES: TemplateCategory[] = [
  { id: 'enhance', label: '通用增强', color: '#2dd4bf' },
  { id: 'style', label: '艺术风格', color: '#a78bfa' },
  { id: 'photo', label: '摄影风格', color: '#60a5fa' },
  { id: 'creation', label: '创作题材', color: '#fb923c' },
  { id: 'compose', label: '构图视角', color: '#22d3ee' },
  { id: 'light', label: '光影氛围', color: '#fbbf24' },
  { id: 'scene', label: '应用场景', color: '#f472b6' },
]

export function getCategory(id: string): TemplateCategory | undefined {
  return CATEGORIES.find((c) => c.id === id)
}

// ─── 预设模板：7 大类 79 个 ───

export const CORE_TEMPLATES: Record<string, Template[]> = {
  'enhance': [
    {
      label: '通用高质量', description: '万金油质量增强，任何画面都适用',
      prompt: '极致细节，专业级构图，电影级光影，8K 超高分辨率，清晰锐利，色彩自然，画面干净',
      negative: '低质量, 模糊, 噪点, 变形, 多余肢体, 画面杂乱',
      size: '1:1', tags: ['质量', '通用'],
    },
    {
      label: '电影质感', description: '胶片颗粒与宽银幕调色',
      prompt: '电影剧照质感，胶片颗粒，宽银幕构图，戏剧性光影，影院级调色，浅景深背景虚化，氛围厚重',
      negative: '低质量, 模糊, 平面感, 电视画质, 数码感',
      size: '16:9', tags: ['质感', '光影'],
    },
    {
      label: '光影增强', description: '强化体积感与明暗层次',
      prompt: '戏剧性电影光效，强烈明暗对比，体积光，柔和高光，丰富阴影层次，立体感强',
      negative: '平光, 过曝, 死黑, 无阴影, 低对比',
      size: '1:1', tags: ['光影'],
    },
    {
      label: '细节增强', description: '毛发、纹理、材质细节拉满',
      prompt: '极致细节表现，清晰纹理质感，毛孔级清晰度，微距级别精度，超强分辨率，材质真实',
      negative: '模糊, 涂抹感, 噪点, 低分辨率, 塑料感',
      size: '1:1', tags: ['细节', '质量'],
    },
    {
      label: '色彩增强', description: '鲜艳饱满但不过曝',
      prompt: '丰富鲜艳的色彩，电影级调色，和谐配色，色彩层次丰富，视觉冲击力强，饱和度自然',
      negative: '灰暗, 偏色, 单色调, 饱和度不足, 过曝',
      size: '1:1', tags: ['色彩'],
    },
    {
      label: '画面层次', description: '前景、中景、背景分明',
      prompt: '丰富的空间层次，清晰的前景中景背景分离，空气透视，纵深构图，氛围感强',
      negative: '平面, 杂乱, 无层次, 模糊',
      size: '16:9', tags: ['构图', '氛围'],
    },
    {
      label: '史诗氛围', description: '宏大叙事感的场面',
      prompt: '史诗级宏大氛围，气吞山河的场面，庄严神秘，戏剧化光线，电影级场面调度，震撼视觉',
      negative: '平淡, 简陋, 低质量, 现代元素, 廉价感',
      size: '16:9', tags: ['氛围', '场景'],
    },
    {
      label: '纯净画面', description: '干净无杂质的画面',
      prompt: '干净纯粹的构图，简洁背景，主体突出，画面无杂物，柔和过渡，精致细腻',
      negative: '杂乱, 多余物体, 噪点, 文字水印, 干扰元素',
      size: '1:1', tags: ['构图', '质量'],
    },
  ],
  'style': [
    {
      label: '中国水墨', description: '写意笔触与留白',
      prompt: '中国水墨画风格，写意笔触，留白构图，淡雅墨色，传统韵味，宣纸质感',
      negative: '色彩鲜艳, 油画质感, 照片感, 西式构图',
      size: '4:3', tags: ['国风', '写意'],
    },
    {
      label: '日系动漫', description: '明亮干净的日漫风',
      prompt: '日系动漫风格，鲜艳色彩，精致角色，明亮光线，干净线条，赛璐璐上色',
      negative: '写实感, 照片感, 暗黑风格, 3D渲染, 粗糙线条',
      size: '3:4', tags: ['二次元', '插画'],
    },
    {
      label: '赛博朋克', description: '霓虹都市与高科感',
      prompt: '赛博朋克风格，霓虹灯光，雨夜都市，高科技感，蓝紫冷色调，未来主义建筑，湿润路面反射',
      negative: '自然光, 暖色调, 古代风格, 简陋, 低科技感',
      size: '16:9', tags: ['未来', '霓虹'],
    },
    {
      label: '蒸汽波', description: '复古数字美学',
      prompt: '蒸汽波风格，复古像素与石膏像，粉紫渐变，故障艺术质感，1980 年代数字美学，梦幻怀旧',
      negative: '写实, 现代科技感, 明亮自然光, 单调',
      size: '1:1', tags: ['复古', '数字艺术'],
    },
    {
      label: '奇幻艺术', description: '魔法史诗插画',
      prompt: '奇幻艺术风格，魔法氛围，史诗感，精致纹理，丰富想象力，神秘元素，壮丽场景',
      negative: '现代元素, 科技感, 简约风格, 照片感, 平庸',
      size: '16:9', tags: ['奇幻', '史诗'],
    },
    {
      label: '数字油画', description: '电影级厚涂',
      prompt: '数字油画风格，电影级光影，高细节，8K，丰富色彩层次，厚重笔触，艺术感',
      negative: '模糊, 低质量, 线条感, 扁平风格, 草图',
      size: '1:1', tags: ['油画', '厚涂'],
    },
    {
      label: '线条插画', description: '干净利落的线稿',
      prompt: '精致线条风格，干净利落的线稿，扁平色彩，插画风，清晰描边，装饰性强',
      negative: '模糊, 3D渲染, 照片感, 过度写实, 脏乱线条',
      size: '1:1', tags: ['插画', '线稿'],
    },
    {
      label: '概念艺术', description: '设计前期的氛围图',
      prompt: '概念艺术风格，史诗级场景，戏剧性光影，氛围感强，想象力丰富，环境概念设计',
      negative: '照片感, 写实感, 平淡, 简陋, 儿童画',
      size: '16:9', tags: ['概念', '场景'],
    },
    {
      label: '暗黑美学', description: '哥特与巴洛克元素',
      prompt: '暗黑美学风格，低调光影，戏剧性明暗对比，深色调，神秘氛围，巴洛克元素，高级感',
      negative: '明亮, 鲜艳色彩, 可爱风格, 简约, 廉价',
      size: '1:1', tags: ['哥特', '神秘'],
    },
    {
      label: '复古胶片', description: '七十年代胶片质感',
      prompt: '复古胶片风格，褪色质感，颗粒感，暖黄调，Vintage 色调，1970 年代美学，怀旧氛围',
      negative: '现代感, 数码感, 冷色调, 过于锐利, 屏幕感',
      size: '4:3', tags: ['复古', '胶片'],
    },
    {
      label: '浮世绘', description: '日式传统版画',
      prompt: '浮世绘风格，日式传统版画，木刻线条，平面装饰性色彩，江户时代风情，海浪与富士元素',
      negative: '写实, 油画质感, 现代元素, 3D渲染',
      size: '3:4', tags: ['国风', '版画'],
    },
    {
      label: '水彩', description: '通透柔和的水彩晕染',
      prompt: '水彩画风格，通透柔和的色彩，自然晕染，纸张纹理，清新雅致，留白与边缘水痕',
      negative: '油画质感, 照片感, 厚重笔触, 3D渲染, 脏色',
      size: '1:1', tags: ['水彩', '清新'],
    },
    {
      label: '像素艺术', description: '复古游戏点阵',
      prompt: '像素艺术风格，复古游戏点阵，清晰像素块，低分辨率美学，8-bit 色调，精致像素构图',
      negative: '高清平滑, 照片感, 矢量感, 模糊抗锯齿',
      size: '1:1', tags: ['复古', '像素'],
    },
    {
      label: '低多边形', description: '几何切面风格',
      prompt: '低多边形风格，几何切面，简洁色块，现代扁平体积感，科技感构图，干净利落',
      negative: '写实, 细节过多, 杂乱, 油画质感',
      size: '16:9', tags: ['几何', '现代'],
    },
    {
      label: '3D 渲染', description: '皮克斯式三维',
      prompt: '3D 渲染风格，皮克斯式三维，柔软材质，体积光，全局光照，细腻表面，景深',
      negative: '2D扁平, 粗糙模型, 低质量贴图, 模糊, 线稿感',
      size: '1:1', tags: ['3D', '渲染'],
    },
  ],
  'photo': [
    {
      label: '写实摄影', description: '逼真自然的专业摄影',
      prompt: '写实摄影风格，自然光线，超高分辨率，逼真质感，细节丰富，8K，专业拍摄',
      negative: '模糊, 低质量, 动漫风格, 绘画质感, 3D渲染感',
      size: '1:1', tags: ['写实', '自然光'],
    },
    {
      label: '电影剧照', description: '宽银幕电影感',
      prompt: '电影剧照风格，宽银幕构图，戏剧性光影，胶片质感，电影级调色，叙事感',
      negative: '模糊, 低质量, 平面感, 电视画质, 摆拍感',
      size: '16:9', tags: ['电影', '叙事'],
    },
    {
      label: '微距特写', description: '极致细节的近摄',
      prompt: '微距摄影特写，极致细节，浅景深，柔美散景，清晰焦点，8K 超清，质感纹理',
      negative: '模糊, 大景深, 噪点, 低分辨率, 对焦失败',
      size: '1:1', tags: ['微距', '细节'],
    },
    {
      label: '纪实街拍', description: '生活气息的抓拍',
      prompt: '纪实街拍风格，自然抓拍感，真实光线，生活气息，人文情怀，街头场景，瞬间感',
      negative: '摆拍感, 过度修饰, 滤镜感, 3D渲染, 影棚光',
      size: '3:4', tags: ['街拍', '人文'],
    },
    {
      label: '美食摄影', description: '诱人的食物特写',
      prompt: '精致美食摄影，诱人色泽，蒸汽或光泽质感，自然侧光，浅景深，高级餐厅摆盘',
      negative: '模糊, 低质量, 不新鲜, 凌乱, 廉价餐具',
      size: '4:3', tags: ['美食', '商业'],
    },
    {
      label: '建筑空间', description: '对称透视的空间',
      prompt: '建筑空间摄影，完美对称或引导线构图，自然光与室内光融合，空间纵深感，专业级',
      negative: '模糊, 畸变, 杂乱, 低光, 手机画质',
      size: '16:9', tags: ['建筑', '空间'],
    },
    {
      label: '自然风光', description: '壮丽山河',
      prompt: '壮丽自然风光摄影，黄金时刻光线，丰富色彩层次，广阔景深，国家地理风格',
      negative: '城市, 人造物, 模糊, 过曝, 低饱和度',
      size: '16:9', tags: ['风光', '自然'],
    },
    {
      label: '人像摄影', description: '棚拍级人像',
      prompt: '专业人像摄影，柔和影棚灯光，干净背景，肤质细腻，眼神光，杂志级后期，浅景深',
      negative: '变形, 皮肤瑕疵, 模糊, 过度磨皮, 杂乱背景',
      size: '3:4', tags: ['人像', '影棚'],
    },
    {
      label: '黑白摄影', description: '纯粹的光影黑白',
      prompt: '黑白摄影风格，强烈明暗对比，影调层次丰富，颗粒质感，经典胶片黑白，光影造型',
      negative: '色彩, 过曝, 死黑, 模糊, 数码感',
      size: '4:3', tags: ['黑白', '光影'],
    },
    {
      label: '长曝光', description: '丝滑流水与车流',
      prompt: '长曝光摄影，丝滑水流或车流光轨，延时平滑效果，夜景灯光拖影，宁静氛围',
      negative: '运动模糊混乱, 过曝, 噪点, 白天',
      size: '16:9', tags: ['长曝光', '夜景'],
    },
    {
      label: '航拍俯瞰', description: '上帝视角的全景',
      prompt: '航拍俯瞰视角，上帝视角，全景构图，丰富细节，震撼场面，无人机摄影，开阔视野',
      negative: '仰视, 平视, 特写, 低清晰度, 杂乱',
      size: '16:9', tags: ['航拍', '俯瞰'],
    },
    {
      label: '双重曝光', description: '人景叠影的艺术',
      prompt: '双重曝光风格，人物与风景叠加，半透明混合，艺术剪影，梦幻层次，超现实',
      negative: '单层图像, 写实单曝, 杂乱叠加, 低对比',
      size: '4:3', tags: ['超现实', '艺术'],
    },
  ],
  'creation': [
    {
      label: '小说封面', description: '精装排版的封面',
      prompt: '精致小说封面设计，专业排版构图，电影级光影，浮雕标题质感，8K 超高清晰，高级感',
      negative: '低质量, 模糊, 文字扭曲, 错别字, 非中文',
      size: '3:4', tags: ['封面', '出版'],
    },
    {
      label: '角色立绘', description: '站姿全身角色展示',
      prompt: '精致角色立绘，完整全身展示，清晰服装设计，纯色背景，角色设定图，动态张力',
      negative: '半身, 裁切, 模糊面部, 复杂背景, 比例失调',
      size: '2:3', tags: ['角色', '立绘'],
    },
    {
      label: '章节插图', description: '小说章节配图',
      prompt: '小说章节插图，与文字搭配的叙事画面，适合排版，留白设计，精致细腻，故事感',
      negative: '混乱, 模糊, 低质量, 过度复杂, 文字水印',
      size: '4:3', tags: ['插画', '叙事'],
    },
    {
      label: '场景概念', description: '世界观大场景',
      prompt: '史诗级场景概念艺术，广阔视野，戏剧性光影，氛围感强，丰富细节，世界观设计',
      negative: '模糊, 平面感, 简陋, 低质量, 现代都市',
      size: '16:9', tags: ['概念', '场景'],
    },
    {
      label: '奇幻生物', description: '原创生物设计',
      prompt: '奇幻生物设计，精细鳞片与羽毛纹理，动态姿态，魔法氛围，史诗感，原创生物设定',
      negative: '模糊, 比例失调, 低质量, 现代元素, 抄袭感',
      size: '1:1', tags: ['生物', '奇幻'],
    },
    {
      label: '机甲设计', description: '精密机械结构',
      prompt: '科幻机甲风格，精密机械结构，金属质感，未来感，动态战斗姿态，高细节，工业设计感',
      negative: '模糊, 有机生物感, 中世纪, 简陋设计, 塑料感',
      size: '16:9', tags: ['机甲', '科幻'],
    },
    {
      label: '游戏原画', description: '游戏美术概念',
      prompt: '游戏原画风格，角色与场景概念设计，史诗光影，高细节材质，专业美术质量，战斗张力',
      negative: '模糊, 草图, 低质量, 粗糙, 手机画质',
      size: '16:9', tags: ['游戏', '原画'],
    },
    {
      label: '漫画分镜', description: '多格叙事分镜',
      prompt: '漫画分镜风格，多格叙事，清晰对话框留白，黑白或有限色，动态动作，漫画面板构图',
      negative: '照片感, 3D渲染, 单格插图, 复杂背景抢戏',
      size: '4:3', tags: ['漫画', '叙事'],
    },
    {
      label: '海报设计', description: '视觉冲击海报',
      prompt: '海报设计风格，强烈视觉冲击，大字排版留白，专业构图，高对比配色，商业广告质感',
      negative: '杂乱, 文字扭曲, 低质量, 排版混乱, 水印',
      size: '3:4', tags: ['海报', '商业'],
    },
    {
      label: '徽章纹章', description: '家族与组织徽记',
      prompt: '徽章纹章设计，对称构图，金属浮雕质感，精细图案，古典或奇幻风格，庄重威严',
      negative: '不对称, 粗糙, 现代元素, 文字错误, 塑料感',
      size: '1:1', tags: ['徽章', '纹章'],
    },
    {
      label: '绘本插画', description: '温暖治愈的绘本风',
      prompt: '绘本插画风格，温暖柔和色调，圆润造型，温馨氛围，儿童书籍插画，治愈感',
      negative: '写实, 暗黑, 恐怖元素, 粗糙线条, 廉价感',
      size: '4:3', tags: ['绘本', '治愈'],
    },
    {
      label: '图标套装', description: '统一风格的图标',
      prompt: '统一风格图标设计，圆角简洁，扁平或线性风格，一致视觉语言，干净背景，装饰性强',
      negative: '风格不统一, 复杂细节, 照片感, 模糊, 阴影混乱',
      size: '1:1', tags: ['图标', 'UI'],
    },
  ],
  'compose': [
    {
      label: '半身肖像', description: '腰上构图',
      prompt: '精致半身肖像构图，柔和影棚光，干净背景，人物突出，8K 超清，眼神表现力',
      negative: '全身, 多人, 杂乱背景, 低分辨率, 变形',
      size: '3:4', tags: ['人像', '半身'],
    },
    {
      label: '全身立绘', description: '完整角色展示',
      prompt: '全身角色立绘构图，完整人物展示，清晰服装设计，白色背景，角色设定图，站姿优雅',
      negative: '半身, 裁切, 模糊面部, 复杂背景, 比例失调',
      size: '2:3', tags: ['角色', '全身'],
    },
    {
      label: '面部特写', description: '大特写的情绪',
      prompt: '面部大特写，极致细节，柔和光线，眼神表达，8K 超清，肤质细腻，情绪饱满',
      negative: '全身, 远景, 遮挡面部, 模糊, 表情僵硬',
      size: '1:1', tags: ['特写', '情绪'],
    },
    {
      label: '广角场景', description: '开阔的视野',
      prompt: '广角场景构图，宏大视野，深远景深，丰富层次，电影级，开阔空间感',
      negative: '特写, 浅景深, 扁平, 低分辨率, 畸变严重',
      size: '16:9', tags: ['广角', '场景'],
    },
    {
      label: '俯视航拍', description: '上帝视角',
      prompt: '俯视航拍视角，上帝视角，全景构图，丰富细节，震撼场面，开阔透视',
      negative: '仰视, 平视, 特写, 低清晰度',
      size: '16:9', tags: ['俯瞰', '航拍'],
    },
    {
      label: '仰视视角', description: '英雄式仰望',
      prompt: '仰视视角构图，英雄式仰望，建筑或人物高大威严，透视强烈，天空背景，气势磅礴',
      negative: '俯视, 平视, 压缩感, 杂乱背景',
      size: '3:4', tags: ['仰视', '气势'],
    },
    {
      label: '对称构图', description: '完美平衡的秩序',
      prompt: '完美对称构图，中心轴线，镜像平衡，建筑或场景庄严感，秩序美学，干净画面',
      negative: '不对称, 杂乱, 偏移主体, 失衡, 多余元素',
      size: '1:1', tags: ['对称', '秩序'],
    },
    {
      label: '三分法', description: '经典黄金分割',
      prompt: '三分法构图，主体置于交点，黄金分割，留白得当，画面平衡，经典摄影构图',
      negative: '主体居中死板, 构图失衡, 拥挤, 杂乱',
      size: '4:3', tags: ['经典', '平衡'],
    },
    {
      label: '引导线', description: '视线延伸的纵深',
      prompt: '引导线构图，道路、栏杆或光线延伸至主体，纵深透视，视觉引导，空间感强',
      negative: '无纵深, 杂乱线条, 主体模糊, 平面感',
      size: '16:9', tags: ['透视', '引导'],
    },
    {
      label: '框架构图', description: '框中取景',
      prompt: '框架构图，门窗、拱形或树枝环绕主体，层次丰富，聚焦视线，画面有深度',
      negative: '无框架, 扁平, 杂乱前景, 主体过小',
      size: '4:3', tags: ['层次', '取景'],
    },
  ],
  'light': [
    {
      label: '黄金时刻', description: '日落暖光',
      prompt: '黄金时刻光线，低角度暖阳，长影，金色氛围，温暖色调，风景或人像皆宜',
      negative: '正午顶光, 冷色调, 过曝, 灰暗, 无阴影',
      size: '16:9', tags: ['日落', '暖光'],
    },
    {
      label: '体积光', description: '光束穿透空气',
      prompt: '体积光效果，光束穿透云层或窗户，丁达尔效应，光柱可见，神圣氛围，空气感强',
      negative: '平光, 无光源, 过曝, 雾霾混乱',
      size: '16:9', tags: ['丁达尔', '神圣'],
    },
    {
      label: '霓虹夜景', description: '赛博城市夜色',
      prompt: '霓虹夜景光线，彩色灯牌反光，湿润路面倒影，蓝紫冷调，城市夜色，赛博氛围',
      negative: '白天, 自然光, 暖黄单调, 无倒影, 灰暗',
      size: '16:9', tags: ['霓虹', '夜景'],
    },
    {
      label: '月光', description: '清冷的月色',
      prompt: '月光照明，清冷银蓝色调，柔和月晕，静谧夜晚，剪影与月光层次，神秘宁静',
      negative: '暖光, 白天, 过曝, 杂乱光源',
      size: '16:9', tags: ['夜晚', '清冷'],
    },
    {
      label: '烛光', description: '温暖摇曳的光',
      prompt: '烛光照明，温暖摇曳的光源，柔和明暗过渡，温馨或神秘氛围，火苗暖色，暗背景',
      negative: '强光, 冷色调, 过曝, 无明暗层次',
      size: '4:3', tags: ['暖光', '温馨'],
    },
    {
      label: '逆光', description: '轮廓发光',
      prompt: '逆光照明，主体轮廓发光，光晕与空气感，发丝光，剪影或半剪影，梦幻氛围',
      negative: '顺光, 平光, 过曝主体, 无轮廓光',
      size: '4:3', tags: ['逆光', '轮廓'],
    },
    {
      label: '剪影', description: '纯黑轮廓',
      prompt: '剪影构图，纯黑轮廓，背景渐变天空或光源，简洁有力，故事感，强烈对比',
      negative: '细节过多, 灰色轮廓, 杂乱背景, 过曝',
      size: '16:9', tags: ['剪影', '对比'],
    },
    {
      label: '戏剧性打光', description: '伦勃朗三角光',
      prompt: '戏剧性布光，伦勃朗三角光，强明暗对比，舞台感，人物立体，电影级打光',
      negative: '平光, 柔光过度, 无阴影, 呆板',
      size: '3:4', tags: ['布光', '人像'],
    },
    {
      label: '柔和漫射', description: '阴天柔光',
      prompt: '柔和漫射光，阴天质感，无硬阴影，色彩温柔，皮肤通透，静谧氛围',
      negative: '强光, 硬阴影, 过曝, 高对比',
      size: '1:1', tags: ['柔光', '清新'],
    },
    {
      label: '高对比', description: '极端的明暗',
      prompt: '高对比光影，极端明暗，黑色背景，主体明亮，极简有力，视觉冲击强',
      negative: '灰调, 低对比, 过曝, 杂乱',
      size: '1:1', tags: ['对比', '极简'],
    },
    {
      label: '蓝调时刻', description: '日落后的冷蓝',
      prompt: '蓝调时刻光线，日落后天空冷蓝，城市灯光初亮，静谧氛围，色彩层次丰富',
      negative: '白天, 全黑, 暖黄过度, 过曝',
      size: '16:9', tags: ['蓝调', '黄昏'],
    },
    {
      label: '舞台顶光', description: '聚光灯效果',
      prompt: '舞台顶光照明，主体顶部受光，面部阴影戏剧化，聚光灯效果，神秘或庄重氛围',
      negative: '顺光, 平光, 过曝, 无层次',
      size: '3:4', tags: ['舞台', '戏剧'],
    },
  ],
  'scene': [
    {
      label: '手机壁纸', description: '竖屏 9:16 构图',
      prompt: '9:16 竖屏构图，主体居中偏上，适合手机壁纸，干净背景，留白设计，高细节',
      negative: '横构图, 主体偏移, 复杂背景, 裁切, 文字水印',
      size: '9:16', tags: ['壁纸', '竖屏'],
    },
    {
      label: '桌面壁纸', description: '横屏 16:9 构图',
      prompt: '16:9 桌面壁纸构图，开阔场景，主体避让任务栏区域，色彩耐看，细节丰富，高分辨率',
      negative: '竖构图, 主体居中遮挡, 杂乱, 低分辨率',
      size: '16:9', tags: ['壁纸', '横屏'],
    },
    {
      label: '短视频封面', description: '高点击封面',
      prompt: '短视频封面构图，竖屏 9:16，主体突出，强视觉焦点，高对比，标题留白区域，抓眼球',
      negative: '横构图, 主体过小, 杂乱, 模糊, 文字水印',
      size: '9:16', tags: ['封面', '竖屏'],
    },
    {
      label: '电商白底', description: '产品主图',
      prompt: '商业产品摄影，纯白背景，专业棚拍灯光，清晰产品细节，电商标准，干净无影',
      negative: '阴影过重, 模糊, 杂色背景, 低分辨率, 反光脏乱',
      size: '1:1', tags: ['电商', '产品'],
    },
    {
      label: '场景展示', description: '生活化布景',
      prompt: '产品场景展示，自然生活化布景，柔和窗光，生活方式摄影，高级感，真实质感',
      negative: '白色背景, 棚拍感, 产品过小, 杂乱, 廉价',
      size: '4:3', tags: ['场景', '产品'],
    },
    {
      label: '头像', description: '方形社交头像',
      prompt: '方形头像构图，居中面部或主体，简洁背景，突出识别度，柔和光线，精致细节',
      negative: '全身, 复杂背景, 裁切面部, 模糊, 多人',
      size: '1:1', tags: ['头像', '社交'],
    },
    {
      label: '印刷海报', description: '印刷级 300dpi',
      prompt: '印刷级海报设计，高分辨率 300dpi，专业排版，色彩管理，留白与出血位意识，商业质感',
      negative: '低分辨率, 文字扭曲, 色彩溢出, 排版混乱, 水印',
      size: '3:4', tags: ['印刷', '海报'],
    },
    {
      label: '社交配图', description: '通用发布图',
      prompt: '社交平台配图，1:1 方形构图，视觉焦点集中，色彩明快，信息层级清晰，引人互动',
      negative: '杂乱, 文字水印, 过暗, 主体不突出, 低质量',
      size: '1:1', tags: ['社交', '配图'],
    },
    {
      label: '直播间背景', description: '横屏虚拟背景',
      prompt: '直播间虚拟背景，16:9 横屏，景深虚化，主体区域留空，氛围感强，不抢人物焦点',
      negative: '人物遮挡, 杂乱细节, 文字, 高饱和抢眼, 竖构图',
      size: '16:9', tags: ['直播', '背景'],
    },
    {
      label: 'PPT 配图', description: '商务汇报配图',
      prompt: '商务 PPT 配图，简洁现代，低饱和配色，大面积留白，抽象或概念化表达，专业克制',
      negative: '花哨, 高饱和, 复杂纹理, 文字水印, 卡通感',
      size: '16:9', tags: ['商务', '简洁'],
    },
  ],
}

// ─── 合并库：内置 7 类 + herdsman 12 类（共 231 个模板） ───

export const CATEGORIES: TemplateCategory[] = [...CORE_CATEGORIES, ...HERDSMAN_CATEGORIES]

export const TEMPLATES: Record<string, Template[]> = { ...CORE_TEMPLATES, ...HERDSMAN_TEMPLATES }

// ─── 自定义模板持久化（兼容旧 key） ───

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
