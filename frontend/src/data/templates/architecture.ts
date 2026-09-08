// 由 scripts/gen-herdsman-templates.mjs 生成（architecture 类模板），勿手改。
// 来源：herdsman.exe embedded template library（.tmp/herdsman_templates.json）
// 瘦身 P3：自 data/herdsmanTemplates.ts 分片，主文件按类别聚合 re-export。
import type { Template } from '../imageTemplates'

export const ARCHITECTURE_TEMPLATES: Template[] = [
  {
    id: "architecture:极简住宅客厅",
    icon: "🏛️",
    label: "极简住宅客厅",
    description: "室内设计、自然光、木色",
    prompt: "极简住宅客厅室内设计，浅色木地板、低矮沙发、白色墙面和大面积落地窗，窗外有树影，家具线条克制，阳光形成柔和阴影，真实建筑摄影",
  },
  {
    id: "architecture:城市精品酒店大堂",
    icon: "🏛️",
    label: "城市精品酒店大堂",
    description: "石材、暖光、高端",
    prompt: "城市精品酒店大堂，米色石材墙面、深色木质接待台、线性吊灯和艺术装置，客人剪影轻微虚化，暖色环境光，高端现代建筑空间摄影",
  },
  {
    id: "architecture:山地美术馆",
    icon: "🏛️",
    label: "山地美术馆",
    description: "混凝土、自然、几何",
    prompt: "建在山坡上的现代美术馆，清水混凝土几何体量嵌入森林边缘，大面积玻璃反射天空，入口前有长台阶和少量行人，阴天柔光，建筑可视化效果",
  },
  {
    id: "architecture:未来地铁站",
    icon: "🏛️",
    label: "未来地铁站",
    description: "拱形空间、导视、冷光",
    prompt: "未来城市地铁站内部，连续拱形天花和发光导视线延伸到远处，站台干净宽阔，列车即将进站，银白和蓝色材质，强烈透视感，科幻建筑设计",
  },
  {
    id: "architecture:海边图书馆",
    icon: "🏛️",
    label: "海边图书馆",
    description: "公共建筑、玻璃、海景",
    prompt: "海边公共图书馆建筑，弧形玻璃幕墙面对蓝色海面，室内书架和阅读区透过玻璃可见，屋顶有绿色花园，晴朗日光，安静开放的建筑摄影",
  },
  {
    id: "architecture:工业风办公室",
    icon: "🏛️",
    label: "工业风办公室",
    description: "砖墙、钢梁、开放工位",
    prompt: "工业风办公室空间，裸露红砖墙、黑色钢梁、开放工位和会议盒子，桌面有电脑和绿植，顶部天窗洒下自然光，真实商业空间摄影，秩序清晰",
  },
  {
    id: "architecture:温泉度假酒店",
    icon: "🏛️",
    label: "温泉度假酒店",
    description: "木质、雾气、山景",
    prompt: "山间温泉度假酒店建筑，木质平台环绕露天温泉池，水面有轻雾，远处是层叠山景，室内暖光从落地窗透出，傍晚蓝调，奢华放松氛围",
  },
  {
    id: "architecture:儿童活动中心",
    icon: "🏛️",
    label: "儿童活动中心",
    description: "彩色、圆角、公共空间",
    prompt: "儿童活动中心室内，圆角墙面、彩色软垫、阅读角和攀爬设施组合在一起，天花有柔和灯带，空间安全明亮，现代教育建筑设计，色彩活泼但不刺眼",
  },
]