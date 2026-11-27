// proposal.ts — ProposalBindings（AppBindings 分域接口之一）：方案库板块的
// 六个绑定。方法名即 Go 绑定名（无短名映射，proxy 按 Gaea 前缀直路由到
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
  /** 修改方案元数据（标题/类型/年度/摘要，全量替换）。标题即方案名，改名后返回新方案名。 */
  GaeaProposalUpdateMeta(name: string, title: string, projectType: string, year: string, summary: string): Promise<string>;
  /**
   * 增量修正一个小节：改标题连带子树标题路径，改正文整节替换（语义向量置空待
   * 重嵌）。两个 update* 全 false 后端会拒绝；没改的那半边不传，避免白重嵌向量。
   */
  GaeaProposalUpdateSection(sectionId: number, title: string, updateTitle: boolean, body: string, updateBody: boolean): Promise<void>;
  /** 把正文里的 gaea-asset://<sha256> 解析成可显示的 data URL。 */
  GaeaProposalAssetDataURL(ref: string): Promise<string>;
}
