// 由 scripts/gen-herdsman-templates.mjs 从 herdsman 嵌入式模板库生成，勿手改。
// 来源：herdsman.exe embedded template library（.tmp/herdsman_templates.json）
// 瘦身 P3：模板数据按类别分片于 data/templates/（categories + 12 类别文件），
// 本文件仅做聚合 re-export——导出面不变：HERDSMAN_CATEGORIES / HERDSMAN_TEMPLATES。
import type { Template } from './imageTemplates'
import { HERDSMAN_CATEGORIES } from './templates/categories'
import { PORTRAIT_TEMPLATES } from './templates/portrait'
import { POSTER_TEMPLATES } from './templates/poster'
import { PRODUCT_TEMPLATES } from './templates/product'
import { HM_SCENE_TEMPLATES } from './templates/hm-scene'
import { CREATIVE_TEMPLATES } from './templates/creative'
import { ILLUSTRATION_TEMPLATES } from './templates/illustration'
import { ANIME_TEMPLATES } from './templates/anime'
import { FOOD_TEMPLATES } from './templates/food'
import { ARCHITECTURE_TEMPLATES } from './templates/architecture'
import { FASHION_TEMPLATES } from './templates/fashion'
import { GAME_TEMPLATES } from './templates/game'
import { UI_TEMPLATES } from './templates/ui'

export { HERDSMAN_CATEGORIES }

export const HERDSMAN_TEMPLATES: Record<string, Template[]> = {
  portrait: PORTRAIT_TEMPLATES,
  poster: POSTER_TEMPLATES,
  product: PRODUCT_TEMPLATES,
  'hm-scene': HM_SCENE_TEMPLATES,
  creative: CREATIVE_TEMPLATES,
  illustration: ILLUSTRATION_TEMPLATES,
  anime: ANIME_TEMPLATES,
  food: FOOD_TEMPLATES,
  architecture: ARCHITECTURE_TEMPLATES,
  fashion: FASHION_TEMPLATES,
  game: GAME_TEMPLATES,
  ui: UI_TEMPLATES,
}