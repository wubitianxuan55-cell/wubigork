// proposal.ts — ProposalBindings（AppBindings 分域接口之一，Go ProposalB 门面）：
// 方案库板块的五个绑定（2026-11-23 立项，2026-11-24 随一级板块「方案库」补齐）。
// 本树为办公树的镜像声明，界面不在文学树里；签名以 Go internal/app/
// bindings_proposal.go 为准，类型与 gaea/lib/types/proposal.ts 同构。
import type {
  ProposalChild,
  ProposalHit,
  ProposalSearchResult,
  ProposalSectionBody,
  ProposalSectionNode,
  ProposalSummary,
} from "../types/proposal";

export interface ProposalBindings {
  GaeaProposalList(): Promise<ProposalSummary[]>;
  GaeaProposalOutline(name: string): Promise<ProposalSectionNode[]>;
  GaeaProposalRead(sectionId: number): Promise<ProposalSectionBody>;
  GaeaProposalSearch(
    query: string,
    scope: string,
    level: number,
    projectType: string,
    year: string,
    proposal: string,
    limit: number,
  ): Promise<ProposalSearchResult>;
  GaeaProposalDelete(name: string): Promise<void>;
}

export type { ProposalChild, ProposalHit, ProposalSearchResult, ProposalSectionBody, ProposalSectionNode, ProposalSummary };
