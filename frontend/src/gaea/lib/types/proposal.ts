// proposal.ts — 方案库板块的绑定面类型。与 Go 侧逐字段对齐：
// internal/app/gaea_proposal.go 的 *View（清单/标题树节点/小节正文）与
// internal/gaea/proposal 的 Hit / Child / SearchResult（JSON 键以 Go tag 为准）。
// Go 的空切片/空指针到 JSON 是 null，消费面按可空处理。

/** 方案清单的一行：元数据 + 规模 + 语义索引进度。 */
export interface ProposalSummary {
  name: string;
  title: string;
  projectType: string;
  year: string;
  /** 现行 / 录入中 */
  status: string;
  summary: string;
  source: string;
  sourceFile: string;
  updatedAt: string;
  sections: number;
  chars: number;
  /** 已向量化的段落块数 / 总块数（0/0 = 尚无可向量化的正文） */
  indexDone: number;
  indexTotal: number;
}

/** 标题树的一个节点（1~4 级；不含正文）。 */
export interface ProposalSectionNode {
  id: number;
  parentId: number;
  level: number;
  seq: number;
  title: string;
  /** 「第4章 安全保障 › 4.3 危险源辨识」 */
  titlePath: string;
  chars: number;
}

/** 子节的轻量描述。 */
export interface ProposalChild {
  id: number;
  level: number;
  title: string;
  chars: number;
}

/** 某小节的正文与定位信息。 */
export interface ProposalSectionBody {
  id: number;
  proposalTitle: string;
  status: string;
  title: string;
  titlePath: string;
  chars: number;
  body: string;
  children: ProposalChild[] | null;
}

/** 一次检索命中：定位到小节，带命中位置摘要（命中词以【】标出）。 */
export interface ProposalHit {
  proposalName: string;
  proposalTitle: string;
  projectType: string;
  year: string;
  status: string;
  sectionId: number;
  level: number;
  title: string;
  titlePath: string;
  chars: number;
  childCount: number;
  snippet: string;
  /** 全文路命中的词（纯语义命中为空） */
  matched: string[] | null;
  keywordScore?: number;
  semanticScore?: number;
  rerankScore?: number;
  /** 子节清单（无子节时省略） */
  children?: ProposalChild[] | null;
  /** 命中来源：keyword=全文路 / semantic=只被语义路召回 / both=两路都命中（纯全文检索不带） */
  source?: string;
}

/** 分面计数的一格：按方案时 key=方案名；按章时 key=章标题所在小节的 id。 */
export interface ProposalFacet {
  key: string;
  label: string;
  count: number;
}

/**
 * 检索结果。mode：title=只查标题树 / keyword=仅全文路 / hybrid=全文+语义。
 * total / byProposal / byChapter 供翻页与收窄（2026-11-28 起；老后端不带这些字段）。
 */
export interface ProposalSearchResult {
  hits: ProposalHit[] | null;
  mode: string;
  note: string;
  /** 满足全部过滤的命中总数 */
  total?: number;
  /** 本页跳过的条数 */
  offset?: number;
  /** 按方案计数（不受「仅本方案」与标题范围过滤影响） */
  byProposal?: ProposalFacet[] | null;
  /** 按章计数（限定在一份方案里时才有；不受标题范围过滤影响） */
  byChapter?: ProposalFacet[] | null;
  /** 多词全含一个都没有，已放宽为任一词 */
  relaxed?: boolean;
  /** 语义路失败的原因（本地模型不可用等）；此时只按全文路检索 */
  searchError?: string;
}

/** 结构操作：升级 / 降级 / 上移 / 下移 / 拆分 / 并入父节 / 并入上一同级小节 / 新增子节 / 新增同级小节 / 删除。 */
export type ProposalSectionOp =
  | "promote" | "demote" | "moveUp" | "moveDown" | "split"
  | "mergeParent" | "mergePrev" | "addChild" | "addSibling" | "remove";

/** 结构操作回执。focusId=操作完成后应落到的小节（0=整册已空）。 */
export interface ProposalStructureResult {
  op: string;
  focusId: number;
  affected: number;
  created: number;
  removed: number;
  removedChars: number;
  missingAssets: string[] | null;
}

/** 同题小节：别的方案里标题路径去掉编号后与本节相同的小节。 */
export interface ProposalTwin {
  proposalName: string;
  proposalTitle: string;
  projectType: string;
  year: string;
  sectionId: number;
  level: number;
  title: string;
  titlePath: string;
  chars: number;
}

/** 索引体检快照。待向量化的块（vecDone<vecTotal）不算问题：检索时自动补齐。 */
export interface ProposalIndexStatus {
  vecDone: number;
  vecTotal: number;
  /** 全文索引行对应的小节已不存在 */
  orphanFts: number;
  /** 有小节却没有全文索引行 */
  missingFts: number;
  /** 全文索引内容与当前的标题 / 路径 / 正文对不上 */
  staleFts: number;
  /** 有正文却没有语义块的小节 */
  missingChunks: number;
  /** 块前缀还是旧标题路径的语义块 */
  staleChunks: number;
  model: string;
}

/** 导出回执。 */
export interface ProposalExportResult {
  path: string;
  /** 托管图片落到的目录（没有图片为空串） */
  assetsDir: string;
  sections: number;
  chars: number;
  images: number;
  /** 托管文件缺失、保留了原引用的图片数 */
  missing: number;
}
