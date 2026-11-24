// mock/proposal.ts — 方案库域（浏览器开发模式的内存实现，无 Go 内核）。
// demo 场景给一份可交互的示例技术标（1~4 级标题树）；fresh 场景为空库。
// 检索只做关键词包含匹配（dev 态没有本地 embedding / reranker），回执标注 keyword。
import type { AppBindings } from "../bridge";
import type {
  ProposalHit,
  ProposalSearchResult,
  ProposalSectionNode,
  ProposalSummary,
} from "../types/proposal";
import type { MakeMockState } from "./state";

type ProposalMethods = Pick<
  AppBindings,
  "GaeaProposalList" | "GaeaProposalOutline" | "GaeaProposalRead" | "GaeaProposalSearch" | "GaeaProposalDelete" |
  "GaeaProposalAssetDataURL"
>;

interface MockSection {
  id: number;
  parentId: number;
  level: number;
  title: string;
  chars: number;
  body: string;
}

interface MockProposal extends ProposalSummary {
  sectionsData: MockSection[];
}

const mockBody = (lines: string[]): string => lines.join("\n\n");

// 示例方案：章节取技术标常见骨架，正文为简短的示意文字。
const demoProposal: MockProposal = {
  name: "示例技术标 沿江地块修复项目",
  title: "沿江地块污染土壤修复项目技术标",
  projectType: "技术标",
  year: "2026",
  status: "现行",
  summary: "污染土壤与地下水修复技术标：原位化学氧化 + 异位热脱附，含施工组织与质量安全环保保证措施。",
  source: "沿江地块技术标_终稿.docx",
  sourceFile: "D:/工作/沿江地块/沿江地块技术标_终稿.md",
  updatedAt: "2026-11-20 09:30",
  sections: 0,
  chars: 0,
  indexDone: 0,
  indexTotal: 0,
  sectionsData: [
    { id: 1, parentId: 0, level: 1, title: "第一章 项目概况", chars: 0, body: mockBody(["项目位于沿江地块南岸，占地约 4.2 公顷，历史为化工企业生产区。", "主要污染物为苯系物与石油烃，最大污染深度 8.5 米。"]) },
    { id: 2, parentId: 1, level: 2, title: "1.1 场地污染状况", chars: 0, body: "详调共布设采样点 86 个，超标点位 31 个，集中在原储罐区与污水处理站周边。" },
    { id: 3, parentId: 1, level: 2, title: "1.2 修复目标与范围", chars: 0, body: "修复目标值执行 GB 36600-2018 第一类用地筛选值；修复土方量约 3.6 万立方米，地下水修复范围约 1.1 万平方米。" },
    { id: 4, parentId: 0, level: 1, title: "第二章 修复技术方案", chars: 0, body: "按污染物类型与分布分区分策：浅层挥发酚污染采用原位化学氧化，深层重污染土采用异位热脱附。" },
    { id: 5, parentId: 4, level: 2, title: "2.1 技术路线比选", chars: 0, body: mockBody(["对原位化学氧化、异位热脱附、固化稳定化三条路线从适用性、工期、造价、二次污染风险四个维度比选。", "结论：浅层原位氧化、深层异位热脱附组合方案综合最优。"]) },
    { id: 6, parentId: 4, level: 2, title: "2.2 施工降水", chars: 0, body: mockBody(["基坑降水采用管井降水，井间距 12 米，降水期间监测周边建筑物沉降。", "如发现沉降超限，立即回灌并停止降水。"]) },
    { id: 7, parentId: 6, level: 3, title: "2.2.1 降水井布置", chars: 0, body: "降水井沿基坑周边环形布置，共 48 口，井深 14 米。" },
    { id: 8, parentId: 0, level: 1, title: "第三章 质量安全环保保证措施", chars: 0, body: "建立质量安全环保三级管理体系，实行样板引路与首件验收制度。" },
    { id: 9, parentId: 8, level: 2, title: "3.1 危险源辨识与防控", chars: 0, body: mockBody(["基坑边坡失稳、扬尘、噪声为主要危险源。", "边坡失稳的防控措施：分层开挖、及时支护、坡顶严禁堆载；扬尘采取洒水覆盖与围挡封闭。"]) },
    { id: 10, parentId: 9, level: 3, title: "3.1.1 危险源清单", chars: 0, body: ["| 序号 | 危险源 | 控制措施 |", "|---|---|---|", "| 1 | 边坡失稳 | 分层开挖、及时支护 |", "| 2 | 扬尘 | 洒水覆盖、围挡封闭 |", "| 3 | 噪声 | 低噪设备、限时施工 |"].join("\n") },
    { id: 11, parentId: 8, level: 2, title: "3.2 环境监测", chars: 0, body: "施工期大气、噪声每周监测一次，地下水每半月监测一次；预警值按环评批复执行。" },
  ],
};

const secondProposal: MockProposal = {
  name: "示例调查报告 百景路地块",
  title: "百景路地块土壤污染状况调查报告",
  projectType: "调查报告",
  year: "2025",
  status: "录入中",
  summary: "百景路地块初步调查与详细调查成果，含污染识别、采样布点与风险评估。",
  source: "百景路地块调查报告.pdf",
  sourceFile: "D:/工作/百景路/调查报告.md",
  updatedAt: "2026-11-18 16:02",
  sections: 0,
  chars: 0,
  indexDone: 0,
  indexTotal: 0,
  sectionsData: [
    { id: 101, parentId: 0, level: 1, title: "1 概述", chars: 0, body: "调查目的、范围与依据。" },
    { id: 102, parentId: 0, level: 1, title: "2 场地污染识别", chars: 0, body: "场地历史上为机械加工厂，重点关注原机加工车间与油料库。" },
  ],
};

function withDerived(p: MockProposal): MockProposal {
  const chars = p.sectionsData.reduce((n, s) => n + s.body.length, 0);
  return { ...p, sections: p.sectionsData.length, chars, indexDone: p.status === "现行" ? 24 : 9, indexTotal: 24 };
}

const titlePathOf = (p: MockProposal, idx: number): string => {
  const stack: string[] = [];
  for (let i = 0; i <= idx; i++) {
    const s = p.sectionsData[i];
    stack[s.level - 1] = s.title;
    stack.length = s.level;
  }
  return stack.join(" › ");
};

const nodeOf = (p: MockProposal, idx: number): ProposalSectionNode => {
  const s = p.sectionsData[idx];
  return { id: s.id, parentId: s.parentId, level: s.level, seq: idx, title: s.title, titlePath: titlePathOf(p, idx), chars: s.body.length };
};

const childrenOf = (p: MockProposal, idx: number) =>
  p.sectionsData
    .map((s, i) => ({ s, i }))
    .filter(({ s }) => s.parentId === p.sectionsData[idx].id)
    .map(({ s }) => ({ id: s.id, level: s.level, title: s.title, chars: s.body.length }));

function mark(text: string, terms: string[]): string {
  let out = text;
  for (const t of terms) out = out.split(t).join(`【${t}】`);
  return out;
}

function snippetOf(body: string, terms: string[]): string {
  const at = terms.map((t) => body.indexOf(t)).filter((n) => n >= 0);
  const pos = at.length ? Math.min(...at) : 0;
  const start = Math.max(0, pos - 60);
  return mark(body.slice(start, start + 220), terms);
}

export function buildProposal(state: MakeMockState): ProposalMethods {
  let proposals: MockProposal[] = state.freshMock ? [] : [withDerived(demoProposal), withDerived(secondProposal)];
  let nextId = 1000;

  const find = (name: string) => proposals.find((p) => p.name === name || p.title === name);
  const summaryOf = ({ sectionsData: _s, ...rest }: MockProposal): ProposalSummary => rest;

  return {
    async GaeaProposalList() {
      return proposals.map(summaryOf);
    },
    async GaeaProposalOutline(name: string) {
      const p = find(name);
      if (!p) throw new Error(`方案库里没有《${name}》`);
      return p.sectionsData.map((_, i) => nodeOf(p, i));
    },
    async GaeaProposalRead(sectionId: number) {
      const p = proposals.find((x) => x.sectionsData.some((s) => s.id === sectionId));
      const idx = p ? p.sectionsData.findIndex((s) => s.id === sectionId) : -1;
      if (!p || idx < 0) throw new Error(`小节 ${sectionId} 不存在（方案可能已被覆盖重录）`);
      const node = nodeOf(p, idx);
      return {
        id: node.id,
        proposalTitle: p.title,
        status: p.status,
        title: node.title,
        titlePath: node.titlePath,
        chars: p.sectionsData[idx].body.length,
        body: p.sectionsData[idx].body,
        children: childrenOf(p, idx),
      };
    },
    async GaeaProposalSearch(query: string, scope: string, level: number, projectType: string, year: string, proposal: string, limit: number) {
      const terms = query.split(/[\s,，、。；;：/|()（）]+/).filter(Boolean);
      if (terms.length === 0) return { hits: [], mode: "keyword", note: "" };
      const titleOnly = scope === "title";
      const hits: ProposalHit[] = [];
      for (const p of proposals) {
        if (projectType && p.projectType !== projectType) continue;
        if (year && p.year !== year) continue;
        if (proposal && p.name !== proposal && p.title !== proposal) continue;
        p.sectionsData.forEach((s, i) => {
          const node = nodeOf(p, i);
          const hay = titleOnly ? node.titlePath : `${node.titlePath}\n${s.body}`;
          const matched = terms.filter((t) => hay.includes(t));
          if (matched.length !== terms.length) return;
          if (level && s.level !== level) return;
          hits.push({
            proposalName: p.name, proposalTitle: p.title, projectType: p.projectType, year: p.year, status: p.status,
            sectionId: s.id, level: s.level, title: s.title, titlePath: node.titlePath, chars: s.body.length,
            childCount: childrenOf(p, i).length,
            snippet: titleOnly ? mark(node.titlePath, matched) : snippetOf(hay, matched),
            matched,
          });
        });
      }
      const top = limit > 0 ? limit : 10;
      return { hits: hits.slice(0, top), mode: titleOnly ? "title" : "keyword", note: "浏览器开发模式无本地模型，仅按关键词匹配" } satisfies ProposalSearchResult;
    },
    async GaeaProposalDelete(name: string) {
      if (!find(name)) throw new Error(`方案 ${name} 不存在`);
      proposals = proposals.filter((p) => !(p.name === name || p.title === name));
      void nextId++;
    },
    async GaeaProposalAssetDataURL(ref: string) {
      if (!ref.startsWith("gaea-asset://")) throw new Error("方案图片引用无效");
      return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII=";
    },
  };
}
