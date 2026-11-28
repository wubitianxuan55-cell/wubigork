// proposal.ts — ProposalBindings（AppBindings 分域接口之一）：方案库板块的
// 十三个绑定。方法名即 Go 绑定名（无短名映射，proxy 按 Gaea 前缀直路由到
// ProposalB 门面）。签名以 Go internal/app/bindings_proposal.go 为准。
//
// 录入（整册重录、从文件导入）只在办公方案模式里由模型经 proposal_save 完成；
// 这里给人看、检索、直接修正入库错误（2026-11-27 元数据与小节编辑，2026-11-28
// 结构修正 / 同题对照 / 索引体检与重建 / 导出）。
import type {
  ProposalExportResult,
  ProposalIndexStatus,
  ProposalSearchResult,
  ProposalSectionBody,
  ProposalSectionNode,
  ProposalSectionOp,
  ProposalStructureResult,
  ProposalSummary,
  ProposalTwin,
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
   * under：限定在某个标题之下（含其全部子孙小节）；offset：翻页跳过前几条；
   * sort："doc" 按文档顺序列，空串按相关度。
   */
  GaeaProposalSearch(
    query: string,
    scope: string,
    level: number,
    projectType: string,
    year: string,
    proposal: string,
    under: number,
    offset: number,
    sort: string,
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
  /**
   * 结构修正：整批一个事务，标题里的编号不改。split 用 offset（正文 rune 偏移，
   * 光标所在行及之后切出）与 title（新子节标题）；addChild / addSibling 用 title。
   */
  GaeaProposalSectionOp(sectionId: number, op: ProposalSectionOp, offset: number, title: string): Promise<ProposalStructureResult>;
  /** 其他方案里与本节同题的小节。 */
  GaeaProposalTwins(sectionId: number): Promise<ProposalTwin[]>;
  /** 索引体检（只读）。 */
  GaeaProposalIndexStatus(): Promise<ProposalIndexStatus>;
  /** 修复并重建索引（清孤儿行、补缺失索引、补齐向量）。 */
  GaeaProposalReindex(): Promise<ProposalIndexStatus>;
  /** 整册导出 Markdown 到绝对路径（由另存对话框选定）。 */
  GaeaProposalExport(name: string, outputPath: string): Promise<ProposalExportResult>;
}
