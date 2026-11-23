// proposal.ts — ProposalBindings（AppBindings 分域接口之一）：方案库板块的
// 五个绑定。方法名即 Go 绑定名（无短名映射，proxy 按 Gaea 前缀直路由到
// ProposalB 门面）。签名以 Go internal/app/bindings_proposal.go 为准。
//
// 录入与索引只在办公方案模式里由模型经 proposal_save 完成；这里只给人看：
// 清单 / 标题树 / 阅读 / 检索 / 删除。
import type {
  ProposalSearchResult,
  ProposalSectionBody,
  ProposalSectionNode,
  ProposalSummary,
} from "../types/proposal";

export interface ProposalBindings {
  /** 方案清单（更新时间降序），带规模与语义索引进度。 */
  GaeaProposalList(): Promise<ProposalSummary[]>;
  /** 某方案（name 或标题）的标题树，不含正文。 */
  GaeaProposalOutline(name: string): Promise<ProposalSectionNode[]>;
  /** 某小节的正文。 */
  GaeaProposalRead(sectionId: number): Promise<ProposalSectionBody>;
  /**
   * 检索：全文 + 本地语义召回 + 本地精排。scope："title" 只查标题树，空串为
   * 全文+语义；其余过滤项空值/0 表示不过滤；limit 0 = 默认 10。
   */
  GaeaProposalSearch(
    query: string,
    scope: string,
    level: number,
    projectType: string,
    year: string,
    proposal: string,
    limit: number,
  ): Promise<ProposalSearchResult>;
  /** 删除方案（小节/语义索引级联清理）。 */
  GaeaProposalDelete(name: string): Promise<void>;
}
