// 由 scripts/gen-herdsman-templates.mjs 生成（game 类模板），勿手改。
// 来源：herdsman.exe embedded template library（.tmp/herdsman_templates.json）
// 瘦身 P3：自 data/herdsmanTemplates.ts 分片，主文件按类别聚合 re-export。
import type { Template } from '../imageTemplates'

export const GAME_TEMPLATES: Template[] = [
  {
    id: "game:像素地下城房间",
    icon: "🎮",
    label: "像素地下城房间",
    description: "俯视、宝箱、火把",
    prompt: "像素风地下城游戏房间，俯视角，石砖地面、木门、宝箱、火把和小怪物分布清楚，墙角有金币和药水，16-bit 色彩，适合作为 RPG 地图素材",
  },
  {
    id: "game:奇幻 RPG 主城",
    icon: "🎮",
    label: "奇幻 RPG 主城",
    description: "等距、商铺、冒险者",
    prompt: "奇幻 RPG 游戏主城等距场景，石板街、武器店、魔法商店、喷泉和公告板组成热闹街区，冒险者与商人穿行其中，明亮手绘风格，细节丰富",
  },
  {
    id: "game:科幻游戏角色设定",
    icon: "🎮",
    label: "科幻游戏角色设定",
    description: "全身、装甲、武器",
    prompt: "科幻游戏角色设定图，一位轻装侦察兵穿模块化外骨骼装甲，手持能量步枪，正面全身站姿，旁边有背包和头盔小视图，干净灰色背景，概念设计稿",
  },
  {
    id: "game:手游抽卡界面",
    icon: "🎮",
    label: "手游抽卡界面",
    description: "角色卡、稀有度、华丽",
    prompt: "幻想手游抽卡结果界面，中央是一张发光 SSR 角色卡，周围有金色粒子、宝石按钮和十连抽信息，背景是深蓝魔法阵，UI 华丽但层级清晰",
  },
  {
    id: "game:赛车游戏车库",
    icon: "🎮",
    label: "赛车游戏车库",
    description: "跑车、霓虹、改装",
    prompt: "赛车游戏车库界面场景，一辆改装跑车停在地下车库中央，地面反射霓虹灯，周围有轮胎、工具箱和性能面板 UI，暗色高对比，适合游戏宣传图",
  },
  {
    id: "game:治愈农场游戏",
    icon: "🎮",
    label: "治愈农场游戏",
    description: "等距、田地、动物",
    prompt: "治愈系农场经营游戏画面，等距视角，小木屋、整齐田地、鸡舍、奶牛和小河组成可爱村庄，角色正在浇水，色彩明亮柔和，适合休闲游戏美术",
  },
  {
    id: "game:平台跳跃关卡",
    icon: "🎮",
    label: "平台跳跃关卡",
    description: "横版、机关、森林",
    prompt: "横版平台跳跃游戏关卡，森林遗迹中有苔藓石台、移动木桥、尖刺陷阱和发光收集物，主角站在起点，背景有多层树影，卡通手绘风格",
  },
  {
    id: "game:战棋游戏地图",
    icon: "🎮",
    label: "战棋游戏地图",
    description: "六边格、雪原、据点",
    prompt: "战棋游戏雪原地图，六边形格子覆盖冰面和松林，地图中央有敌方据点和旗帜，边缘有山脉和冻结河流，单位图标清晰，策略游戏 UI 视角",
  },
]