// mock/proposal.ts — 方案库域（浏览器开发模式的内存实现，无 Go 内核）。
// demo 场景给两份可交互的示例方案（1~4 级标题树）；fresh 场景为空库。
// 检索只做关键词包含匹配（dev 态没有本地 embedding / reranker），回执标注 keyword；
// 结构修正 / 同题对照 / 索引体检照真机的回执形态做；导出要文件系统，dev 态不支持。
import type { AppBindings } from "../bridge";
import type {
  ProposalExportResult,
  ProposalFacet,
  ProposalHit,
  ProposalIndexStatus,
  ProposalSearchResult,
  ProposalSectionNode,
  ProposalStructureResult,
  ProposalSummary,
  ProposalTwin,
} from "../types/proposal";
import type { MakeMockState } from "./state";

type ProposalMethods = Pick<
  AppBindings,
  "GaeaProposalList" | "GaeaProposalOutline" | "GaeaProposalRead" | "GaeaProposalSearch" | "GaeaProposalDelete" |
  "GaeaProposalUpdateMeta" | "GaeaProposalUpdateSection" |
  "GaeaProposalAssetDataURL" | "GaeaProposalSectionOp" | "GaeaProposalTwins" |
  "GaeaProposalIndexStatus" | "GaeaProposalReindex" | "GaeaProposalExport"
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

// ── 标题树结构（与 Go 侧 structure.go 同一套规则） ──

/** nodes[i] 的子树结束下标（不含）。 */
const subtreeEnd = (nodes: MockSection[], i: number): number => {
  for (let j = i + 1; j < nodes.length; j++) if (nodes[j].level <= nodes[i].level) return j;
  return nodes.length;
};

/** nodes[i] 上一个同级小节的下标；已是本层第一个时为 -1。 */
const prevSibling = (nodes: MockSection[], i: number): number => {
  for (let k = i - 1; k >= 0; k--) {
    if (nodes[k].level === nodes[i].level) return k;
    if (nodes[k].level < nodes[i].level) break;
  }
  return -1;
};

/** 结构改动后重连父节点（父=前面最近的更高级标题）。 */
const relink = (p: MockProposal): void => {
  const stack: number[] = [];
  for (const s of p.sectionsData) {
    while (stack.length >= s.level) stack.pop();
    s.parentId = stack.length ? stack[stack.length - 1] : 0;
    stack.push(s.id);
  }
};

/** 同题键：标题路径各段去掉开头编号（首段的纯数字章号也算编号）。 */
const twinKey = (path: string): string =>
  path.split(" › ").map((seg, i) => {
    let core = seg.trim()
      .replace(/^第[一二三四五六七八九十百千零〇两\d]+(章|篇|部分|部|卷|编|册)[\s\u3000:：、.．]*/, "")
      .replace(/^\d{1,4}(?:\.\d{1,4})+[\s\u3000:：]*/, "")
      .replace(/^[一二三四五六七八九十]{1,3}[\s\u3000]*[、.．][\s\u3000]*/, "")
      .replace(/^[（(][\s\u3000]*[一二三四五六七八九十]{1,3}[\s\u3000]*[）)][\s\u3000]*/, "");
    if (i === 0) core = core.replace(/^\d{1,3}[\s\u3000]+(\D.+)$/, "$1");
    return core.replace(/\s+/g, " ");
  }).join(" › ");

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
  const indexStatus = (): ProposalIndexStatus => ({
    vecDone: proposals.reduce((n, p) => n + p.indexDone, 0),
    vecTotal: proposals.reduce((n, p) => n + p.indexTotal, 0),
    orphanFts: 0, missingFts: 0, staleFts: 0, missingChunks: 0, staleChunks: 0,
    model: "mock-embed",
  });

  return {
    async GaeaProposalList() {
      return proposals.map(summaryOf);
    },
    async GaeaProposalOutline(name: string) {
      const p = find(name);
      if (!p) throw new Error(`方案库里没有《${name}》`);
      relink(p);
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
    async GaeaProposalSearch(query, scope, level, projectType, year, proposal, under, offset, sort, limit) {
      const terms = query.split(/[\s,，、。；;：/|()（）]+/).filter(Boolean);
      if (terms.length === 0) return { hits: [], total: 0, offset, byProposal: [], byChapter: [], mode: "keyword", note: "" };
      const titleOnly = scope === "title";
      const top = limit > 0 ? limit : 10;
      // under：限定在某个标题之下（含它自己）→ 所在方案及其下标区间
      let scoped: MockProposal | undefined;
      let lo = -1;
      let hi = -1;
      if (under > 0) {
        for (const p of proposals) {
          const i = p.sectionsData.findIndex((s) => s.id === under);
          if (i >= 0) {
            scoped = p;
            lo = i;
            hi = subtreeEnd(p.sectionsData, i);
            break;
          }
        }
      }
      const collect = (needAll: boolean, opts: { scope: boolean; range: boolean }): ProposalHit[] => {
        const out: ProposalHit[] = [];
        for (const p of proposals) {
          if (opts.scope) {
            if (projectType && p.projectType !== projectType) continue;
            if (year && p.year !== year) continue;
            if (scoped ? p !== scoped : proposal && p.name !== proposal && p.title !== proposal) continue;
          }
          p.sectionsData.forEach((s, i) => {
            if (opts.scope && opts.range && scoped && (i < lo || i >= hi)) return;
            const node = nodeOf(p, i);
            const hay = titleOnly ? node.titlePath : `${node.titlePath}\n${s.body}`;
            const matched = terms.filter((t) => hay.includes(t));
            if (matched.length < (needAll ? terms.length : 1)) return;
            if (opts.scope && level && s.level !== level) return;
            out.push({
              proposalName: p.name, proposalTitle: p.title, projectType: p.projectType, year: p.year, status: p.status,
              sectionId: s.id, level: s.level, title: s.title, titlePath: node.titlePath, chars: s.body.length,
              childCount: childrenOf(p, i).length,
              snippet: titleOnly ? mark(node.titlePath, matched) : snippetOf(hay, matched),
              matched, source: "keyword",
            });
          });
        }
        return out;
      };
      // 多词全含一个都没有：放宽为任一词
      let relaxed = false;
      let hits = collect(true, { scope: true, range: true });
      if (hits.length === 0 && terms.length > 1) {
        hits = collect(false, { scope: true, range: true });
        relaxed = hits.length > 0;
      }
      // sort="doc" 与相关度在 dev 态都是文档顺序（没有排序模型）
      void sort;
      const byProposal: ProposalFacet[] = [];
      for (const h of collect(!relaxed, { scope: false, range: false })) {
        const f = byProposal.find((x) => x.key === h.proposalName);
        if (f) f.count += 1;
        else byProposal.push({ key: h.proposalName, label: h.proposalTitle, count: 1 });
      }
      const byChapter: ProposalFacet[] = [];
      const one = scoped ?? (proposal ? find(proposal) : undefined);
      if (one && !titleOnly) {
        for (const h of collect(!relaxed, { scope: true, range: false })) {
          if (h.proposalName !== one.name) continue;
          const at = one.sectionsData.findIndex((s) => s.id === h.sectionId);
          let chapterId = 0;
          let chapterTitle = "";
          for (let i = at; i >= 0; i--) {
            if (one.sectionsData[i].level === 1) {
              chapterId = one.sectionsData[i].id;
              chapterTitle = one.sectionsData[i].title;
              break;
            }
          }
          if (!chapterId) continue;
          const key = String(chapterId);
          const f = byChapter.find((x) => x.key === key);
          if (f) f.count += 1;
          else byChapter.push({ key, label: chapterTitle, count: 1 });
        }
      }
      return {
        hits: hits.slice(offset, offset + top), total: hits.length, offset, byProposal, byChapter, relaxed,
        mode: titleOnly ? "title" : "keyword", note: "浏览器开发模式无本地模型，仅按关键词匹配",
      } satisfies ProposalSearchResult;
    },
    async GaeaProposalDelete(name: string) {
      if (!find(name)) throw new Error(`方案 ${name} 不存在`);
      proposals = proposals.filter((p) => !(p.name === name || p.title === name));
    },
    async GaeaProposalUpdateMeta(name, title, projectType, year, summary) {
      const p = find(name);
      if (!p) throw new Error(`方案库里没有《${name}》`);
      const t = title.trim();
      if (!t) throw new Error("新标题不能为空");
      const clash = find(t);
      if (clash && clash !== p) throw new Error(`方案库里已有《${t}》，改名会撞名；换一个标题`);
      p.title = t;
      p.name = t; // 方案名=标题（与录入约定一致）
      p.projectType = projectType.trim();
      p.year = year.trim();
      p.summary = summary.trim();
      return p.name;
    },
    async GaeaProposalUpdateSection(sectionId, title, updateTitle, body, updateBody) {
      if (!updateTitle && !updateBody) throw new Error("没有要改的内容：标题与正文至少勾选一个");
      const p = proposals.find((x) => x.sectionsData.some((s) => s.id === sectionId));
      const idx = p ? p.sectionsData.findIndex((s) => s.id === sectionId) : -1;
      if (!p || idx < 0) throw new Error(`小节 ${sectionId} 不存在（方案可能已被覆盖重录）`);
      if (updateTitle) {
        const t = title.trim();
        if (!t) throw new Error("新标题不能为空");
        p.sectionsData[idx].title = t;
      }
      if (updateBody) p.sectionsData[idx].body = body;
    },
    async GaeaProposalAssetDataURL(ref: string) {
      if (!ref.startsWith("gaea-asset://")) throw new Error("方案图片引用无效");
      return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII=";
    },
    async GaeaProposalSectionOp(sectionId, op, offset, title): Promise<ProposalStructureResult> {
      const p = proposals.find((x) => x.sectionsData.some((s) => s.id === sectionId));
      const idx = p ? p.sectionsData.findIndex((s) => s.id === sectionId) : -1;
      if (!p || idx < 0) throw new Error(`小节 ${sectionId} 不存在（方案可能已被覆盖重录）`);
      const secs = p.sectionsData;
      const cur = secs[idx];
      const lv = cur.level;
      const end = subtreeEnd(secs, idx);
      const prev = prevSibling(secs, idx);
      const next = end < secs.length && secs[end].level === lv ? end : -1;
      const t = title.replace(/\s+/g, " ").trim();
      const needTitle = () => { if (!t) throw new Error("新小节的标题不能为空"); };
      const absorb = (into: MockSection, from: MockSection) => {
        const a = into.body.trim();
        const b = from.body.trim();
        into.body = !a ? b : !b ? a : `${a}\n\n${b}`;
      };
      let focus = cur.id;
      let created = 0;
      let removed = 0;
      let removedChars = 0;
      switch (op) {
        case "promote": {
          if (lv <= 1) throw new Error("它已是 1 级标题，不能再用「升级」");
          const sub = secs.splice(idx, end - idx);
          sub.forEach((s) => { s.level -= 1; });
          let at = idx; // 落点：原父节的子树末尾
          for (; at < secs.length; at++) if (secs[at].level <= lv - 1) break;
          secs.splice(at, 0, ...sub);
          break;
        }
        case "demote": {
          if (lv >= 4) throw new Error("它已是 4 级标题，不能再降级");
          if (prev < 0) throw new Error("它前面没有同级小节，降级后没有可挂靠的上一节；先把它上移 / 下移到合适的位置再降级");
          for (let k = idx; k < end; k++) secs[k].level += 1;
          break;
        }
        case "moveUp": {
          if (prev < 0) throw new Error("它已经是上一级标题下的第一个小节，不能再上移");
          const block = secs.splice(idx, end - idx);
          const mid = secs.splice(prev, idx - prev);
          secs.splice(prev, 0, ...block, ...mid);
          break;
        }
        case "moveDown": {
          if (next < 0) throw new Error("它已经是上一级标题下的最后一个小节，不能再下移");
          const nextEnd = subtreeEnd(secs, next);
          const mid = secs.splice(next, nextEnd - next);
          const block = secs.splice(idx, end - idx);
          secs.splice(idx, 0, ...mid, ...block);
          break;
        }
        case "split": {
          if (lv >= 4) throw new Error("它已是 4 级标题，下面不能再拆出子节；可以先给它升级");
          needTitle();
          let cut = Math.max(0, Math.min(offset, cur.body.length));
          while (cut > 0 && cur.body[cut - 1] !== "\n") cut -= 1;
          if (cut <= 0) throw new Error("光标在本节第一行，拆分后本节会没有正文；把光标移到要拆出去的那一段再拆");
          const tail = cur.body.slice(cut).replace(/^\n+/, "");
          if (!tail.trim()) throw new Error("光标之后没有正文可拆出去");
          cur.body = cur.body.slice(0, cut).replace(/[\n \t\r]+$/, "");
          const n: MockSection = { id: nextId++, parentId: cur.id, level: lv + 1, title: t, chars: 0, body: tail };
          secs.splice(idx + 1, 0, n);
          focus = n.id;
          created = 1;
          break;
        }
        case "mergeParent": {
          if (lv <= 1) throw new Error("它是 1 级标题，上面没有父节可并入");
          if (prev >= 0) throw new Error("它不是父节下的第一个小节，并入后正文会排到前面几节之前；要保住阅读顺序请用「并入上一同级小节」");
          absorb(secs[idx - 1], cur);
          for (let k = idx + 1; k < end; k++) secs[k].level -= 1;
          secs.splice(idx, 1);
          focus = secs[idx - 1].id;
          removed = 1;
          break;
        }
        case "mergePrev": {
          if (prev < 0) throw new Error("它前面没有同级小节可并入");
          absorb(secs[prev], cur);
          secs.splice(idx, 1);
          focus = secs[prev].id;
          removed = 1;
          break;
        }
        case "remove": {
          removed = end - idx;
          removedChars = secs.slice(idx, end).reduce((n, s) => n + s.body.length, 0);
          focus = prev >= 0 ? secs[prev].id : idx > 0 ? secs[idx - 1].id : end < secs.length ? secs[end].id : 0;
          secs.splice(idx, end - idx);
          break;
        }
        case "addChild":
        case "addSibling": {
          if (op === "addChild" && lv >= 4) throw new Error("它已是 4 级标题，下面不能再加子节；可以改用「新增同级小节」");
          needTitle();
          const n: MockSection = { id: nextId++, parentId: 0, level: op === "addChild" ? lv + 1 : lv, title: t, chars: 0, body: "" };
          secs.splice(end, 0, n);
          focus = n.id;
          created = 1;
          break;
        }
        default:
          throw new Error(`未知的结构操作 "${op}"`);
      }
      relink(p);
      return { op, focusId: focus, affected: secs.length, created, removed, removedChars, missingAssets: [] };
    },
    async GaeaProposalTwins(sectionId: number): Promise<ProposalTwin[]> {
      const owner = proposals.find((x) => x.sectionsData.some((s) => s.id === sectionId));
      const idx = owner ? owner.sectionsData.findIndex((s) => s.id === sectionId) : -1;
      if (!owner || idx < 0) throw new Error(`小节 ${sectionId} 不存在（方案可能已被覆盖重录）`);
      const key = twinKey(titlePathOf(owner, idx));
      const out: ProposalTwin[] = [];
      for (const p of proposals) {
        if (p === owner) continue;
        p.sectionsData.forEach((s, i) => {
          const node = nodeOf(p, i);
          if (twinKey(node.titlePath) !== key) return;
          out.push({
            proposalName: p.name, proposalTitle: p.title, projectType: p.projectType, year: p.year,
            sectionId: s.id, level: s.level, title: s.title, titlePath: node.titlePath, chars: s.body.length,
          });
        });
      }
      return out;
    },
    async GaeaProposalIndexStatus() {
      return indexStatus();
    },
    async GaeaProposalReindex() {
      return indexStatus();
    },
    async GaeaProposalExport(_name: string, _outputPath: string): Promise<ProposalExportResult> {
      throw new Error("浏览器开发模式没有文件系统，导出请用桌面端");
    },
  };
}
