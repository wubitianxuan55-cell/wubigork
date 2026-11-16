import { app } from "./bridge";
import type { TaskTemplate } from "./types";

// 内置任务模板兜底（2026-11-15 刀4 小说化）：后端命令库为空或加载失败时
// （首启/离线），命令面板仍有常用写作任务。图文板块模板=长篇小说创作八条，
// 与图文专属工具四族（wenxue_*）和内嵌技能（story-long-write/story-deslop/
// wenxue-workbench）同源配套。
export const FALLBACK_TEMPLATES: TaskTemplate[] = [
  {
    name: "chapter-first",
    title: "开篇钩子",
    description: "黄金三章第一章：500 字内起冲突，行动显性格，末尾留悬念",
    prompt:
      "为当前小说撰写第一章（或重写开篇）：先读 角色卡与时间线 把握设定，然后按黄金三章原则写正文——前 500 字内出现矛盾/冲突，主角性格通过行动体现（不可旁白说明），章末留一个让读者想继续的悬念。写完把新事件追加到时间线、更新相关角色卡的「当前状态」。写作用 story-deslop 口径自查一遍再交稿。",
  },
  {
    name: "chapter-continue",
    title: "续写推进",
    description: "衔接上一章续写：冲突推进 + 小高潮 + 章末钩子",
    prompt:
      "续写下一章：先用 wenxue_chapter_list 看进度、wenxue_chapter_read 读上一章结尾与相关角色卡当前状态、wenxue_timeline_read 梳理最近事件，然后续写约 2000-3000 字——每 2000 字至少一个小高潮，对话占比 35-45%，章末留钩子。写完回写时间线与角色状态。",
  },
  {
    name: "outline-push",
    title: "大纲推演",
    description: "未来 10 章走向：冲突升级 + 分支 + 伏笔登记",
    prompt:
      "基于当前时间线与角色卡，推演未来 10 章的走向：每章一行（章号/核心冲突/爽点/章末钩子），标出需要埋设与回收的伏笔，指出角色关系的变化点。落成 设定/大纲-推演.md 供后续逐章消费。",
  },
  {
    name: "setting-expand",
    title: "设定扩充",
    description: "世界观/体系/势力设定文档化",
    prompt:
      "整理并扩充小说设定：读 设定/ 目录既有文档与已写章节，把散落在正文里的设定（体系规则/势力关系/地理/道具）文档化归位到 设定/ 下，标注「已正文实证」与「待正文落地」两类，避免与已有章节矛盾。",
  },
  {
    name: "character-card",
    title: "角色卡建立",
    description: "建/更新角色卡：设定/外貌/性格/口头禅/关系/当前状态",
    prompt:
      "为指定角色建卡或更新（先用 wenxue_character_import 查角色库：同名命中即全字段导入再本地化，未命中才从零写；再用 wenxue_character_list 查工作空间查重）：按「设定 / 外貌 / 性格 / 口头禅 / 关系 / 当前状态」六节撰写，口头禅与句式要独特，当前状态必须与最新章节一致；有关系变动的章节要同步更新其他相关角色卡的「关系」节。",
  },
  {
    name: "timeline-tidy",
    title: "时间线整理",
    description: "从已写章节反推时间线，补事件、纠章号",
    prompt:
      "整理时间线：通读现有章节（wenxue_chapter_list + 逐章读），把漏记的事件按「第N章 · 一句话事件」补进时间线（wenxue_timeline_append），修正章号错标，标记可疑的时序矛盾（前后章事件倒挂）并给出修正建议。",
  },
  {
    name: "polish-deslop",
    title: "润色去味",
    description: "按去 AI 味口径润色指定章节",
    prompt:
      "润色指定章节（先问清章号或用 wenxue_chapter_list 确认）：按 story-deslop 口径逐段自查——删 AI 套话、调句子呼吸（长短交替、极短句至多一处）、情绪词换身体反应、对话加长度差与角色腔；保持情节与字数大体不变，润完用 wenxue_chapter_write 覆写（overwrite=true）。",
  },
  {
    name: "recap-review",
    title: "阶段合账",
    description: "每 10 章一次：剧情线/人物弧/伏笔对账",
    prompt:
      "做一次阶段合账（每 10 章惯例）：读时间线与角色卡，输出「剧情主线摘要 / 各角色弧线进度 / 伏笔已埋未收清单 / 节奏问题（连续平淡或冲突密度不足）」四段，落成 设定/合账-第N阶段.md，并给出下一阶段的推进建议。",
  },
];

export function resolveTemplates(remote: TaskTemplate[] | null | undefined): TaskTemplate[] {
  return remote && remote.length > 0 ? remote : FALLBACK_TEMPLATES;
}

let templatesCache: TaskTemplate[] | null = null;

/** 拉取任务模板（带缓存）：远端成功 → 缓存；失败/空 → 内置模板。 */
export async function loadTemplates(): Promise<TaskTemplate[]> {
  if (templatesCache) return templatesCache;
  try {
    const ts = await app.TaskTemplates();
    templatesCache = resolveTemplates(ts);
  } catch {
    templatesCache = FALLBACK_TEMPLATES;
  }
  return templatesCache;
}

export function resetTemplatesCacheForTest(): void {
  templatesCache = null;
}
