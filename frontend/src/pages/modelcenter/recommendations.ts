// recommendations.ts — 模型推荐位三档静态数据（阶段十 10.2）
// 数据源：docs/gaea-competitor-survey-2026-10.md §5.3「NSFW 本地写作模型谱系」
//（2026-10 社区口碑，r/WritingWithAI / r/SillyTavernAI / r/LocalLLaMA 实测帖）。
// 标注口径诚实：口碑栏保留 survey 原文措辞（含负面：非英语变笨/量化敏感/需越
// 狱），不做成广告位。量化建议联动 10.1 量化共识：文风对低量化容忍度远高于知
// 识（知识低量化非线性崩塌）；agent 工具调用 <Q4 崩溃率高（见 10.1 红线）。

export type RecTier = 'nsfw' | 'writing' | 'intelligence'

export interface RecModel {
  /** 模型系名（survey 原文） */
  name: string
  /** 规模/基座 */
  base: string
  /** 口碑（survey 原文措辞，含负面） */
  notes: string
  /** 量化建议（survey 原文） */
  quant: string
  /** multilingual 标注 */
  multilingual?: string
}

export interface RecTierDef {
  tier: RecTier
  label: string
  /** 档位定位一句话 */
  blurb: string
  models: RecModel[]
}

export const REC_TIERS: RecTierDef[] = [
  {
    tier: 'nsfw',
    label: 'NSFW 特化',
    blurb: '成人向写作 go-to（ADULT_MODE 口径内自用）',
    models: [
      {
        name: 'TheDrummer 系',
        base: 'Behemoth 123B / Anubis 105B / Skyfall 31B / Cydonia 24B（Mistral/Llama 系）',
        notes: 'NSFW 特化 go-to',
        quant: 'Q4_K_M',
        multilingual: '非英语会变笨',
      },
    ],
  },
  {
    tier: 'writing',
    label: '文笔/创意写作',
    blurb: '文笔优先——量化共识：文风对低量化容忍度远高于知识',
    models: [
      {
        name: 'Magnum v4（anthracite）',
        base: '123B = Mistral Large',
        notes: '文笔 magnificent',
        quant: '≥Q4（量化敏感，Q2 崩）',
      },
      {
        name: 'Midnight Miqu',
        base: '70B / 103B（Llama mix）',
        notes: '「迄今最佳小说写作」；IQ2_S 仍出佳品=低量化写作可行背书',
        quant: 'IQ2_S 起可读',
      },
      {
        name: 'Gemma4 31B',
        base: 'Gemma4（非 NSFW 特化）',
        notes: '「创意写作第一，小量化大幅领先，frontier 级上下文连贯」',
        quant: '小量化也稳',
      },
    ],
  },
  {
    tier: 'intelligence',
    label: '智力/长程',
    blurb: '智力与长程伏笔最强，文风需配方',
    models: [
      {
        name: 'GLM-5.1 / Kimi K2 蒸馏路线',
        base: '大杯 API 蒸馏',
        notes: '智力/长程伏笔最强，文风需配方',
        quant: '3bit 可用',
        multilingual: '中文友好',
      },
    ],
  },
]

