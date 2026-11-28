// structureItems.ts — 「调整结构」菜单的条目表：升降级、上下移、并入、新增与删除小节。
// 可用态与后端 structure.go 的判定同口径（后端会再校验一遍）。拆分要取编辑框里的光标，
// 放在编辑态里，不在这张表里。
import type { ProposalSectionOp } from "../../gaea/lib/types/proposal";
import type { StructureFlags } from "./logic";

export interface StructureItem {
  op: ProposalSectionOp;
  label: string;
  hint: string;
  danger?: boolean;
  enabled: (flags: StructureFlags) => boolean;
  /** 不可用时的原因（做 title） */
  why: string;
}

export const STRUCTURE_ITEMS: StructureItem[] = [
  { op: "promote", label: "升级一级", hint: "变成上一级标题的同级、排在其后，子节跟着走", enabled: (f) => f.canPromote, why: "已是 1 级标题" },
  { op: "demote", label: "降级一级", hint: "归到上一同级小节名下，位置不变", enabled: (f) => f.canDemote, why: "已是 4 级标题，或前面没有同级小节可挂靠" },
  { op: "moveUp", label: "上移", hint: "与上一同级小节换位", enabled: (f) => f.canMoveUp, why: "已是本层第一个" },
  { op: "moveDown", label: "下移", hint: "与下一同级小节换位", enabled: (f) => f.canMoveDown, why: "已是本层最后一个" },
  { op: "mergePrev", label: "并入上一同级小节", hint: "本节正文接到它的末尾，本节标题移除", enabled: (f) => f.canMergePrev, why: "前面没有同级小节" },
  { op: "mergeParent", label: "并入父节", hint: "本节正文接到父节正文后面，子节升一级", enabled: (f) => f.canMergeParent, why: "只有父节下的第一个小节能并入父节（否则正文会排到前面几节之前）" },
  { op: "addChild", label: "新增子节", hint: "在本节下面加一个下级标题", enabled: (f) => f.canAddChild, why: "已是 4 级标题" },
  { op: "addSibling", label: "新增同级小节", hint: "在本节之后加一个同级标题", enabled: () => true, why: "" },
  { op: "remove", label: "删除本节", hint: "连同它的子节一起删除", danger: true, enabled: () => true, why: "" },
];
