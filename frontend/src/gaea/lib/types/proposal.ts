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
}

/** 检索结果。mode：title=只查标题树 / keyword=仅全文路 / hybrid=全文+语义。 */
export interface ProposalSearchResult {
  hits: ProposalHit[] | null;
  mode: string;
  note: string;
}
