import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// KnowledgeSummary is the lightweight view of a knowledge entry (without body).
export interface KnowledgeSummary {
  name: string;
  title: string;
  category: string;
  tags: string[];
  status: string;
  // 新建/导入时前端不发送时间戳（Go 端 time.Time 不接受空串），留空由后端置零。
  updatedAt?: string;
}

// KnowledgeEntry is the full knowledge entry including body.
export interface KnowledgeEntry extends KnowledgeSummary {
  body: string;
  phase: string;
  discipline: string;
  source: string;
  version: number;
  author: string;
  reviewer: string;
  // 新建/导入时前端不发送时间戳（Go 端 time.Time 不接受空串），留空由后端置零。
  createdAt?: string;
}

/** Alias for clarity when saving. */
export type KnowledgeSaveRequest = KnowledgeEntry;
// ── 知识库导入（无确认不落库）────────────────────────────────
export interface KnowledgeImportRow {
  name: string;
  title: string;
  category: string;
  phase: string;
  discipline: string;
  tags: string[];
  status: string;
  source: string;
  body: string;
  existingName: string;
  matchNote: string; // 新增 / 将覆盖更新
  similarName: string;
  similarNote: string; // 与「xxx」相似 87%，建议合并
  raw: string;
  skip: boolean;
  skipReason: string;
}

export type KnowledgeImportPreview = WireShape<AppModels.KnowledgeImportPreview>;

// 知识条目版本历史快照。
export type KnowledgeHistoryView = WireShape<AppModels.KnowledgeHistoryView>;

// 查重命中的相似条目。
export type SimilarView = WireShape<AppModels.SimilarView>;
