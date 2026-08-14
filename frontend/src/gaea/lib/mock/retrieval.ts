// mock/retrieval.ts — 检索/索引/测评域（T6-10.1 拆分自 lib/mock.ts）。
// 从 office.ts 独立拆分（原 office 域超过 400 行预算）；方法体除
// RetrievalEvalRun 契约对齐（T6-10.4）外零改动。
import type { AppBindings } from "../bridge";
import type { RetrievalEvalReport } from "../types";
import { taskView } from "./shared";
import type { MakeMockState } from "./state";

type RetrievalMethods = Pick<
  AppBindings,
  | "SemanticSearch" | "UnifiedSearch" | "RetrievalEvalRun"
  | "FileIndexRebuild" | "FileSemanticSearch"
>;

// ── 检索质量测评（T6-10.4 契约对齐 Go GaeaRetrievalEvalRun）─────────────
// 查询集与字段形状与 internal/app/gaea_retrieval_eval.go + docs/retrieval-eval-set.md
// 一致：total=查询集条数（12）、threshold=0.8（Go retrievalEvalThreshold）、
// expected/topHits 为 "kind:name"、recall 按 Go evalHitMatched 同规则计算
// （同 kind 且 name 精确相等或互为子串）、recallAt10=各条 recall 均值、
// passed=recallAt10>=threshold。topHits 为演示命中，非真实测评结果。
const RETRIEVAL_EVAL_THRESHOLD = 0.8; // Go retrievalEvalThreshold

interface EvalRow {
  query: string;
  expected: string[];
  topHits: string[];
}

// docs/retrieval-eval-set.md 的 12 条真实查询（kind:name 化）。
const RETRIEVAL_EVAL_SET: EvalRow[] = [
  { query: "打桩设备 台班价", expected: ["cost:HP300 高频液压振动锤", "cost:桩机台班费"], topHits: ["cost:HP300 高频液压振动锤", "cost:桩机台班费", "knowledge:桩基检测规范要点"] },
  { query: "P.O 42.5 水泥 价格", expected: ["cost:P.O 42.5 水泥", "file:docs/材料价格信息价.md"], topHits: ["cost:P.O 42.5 水泥", "file:docs/材料价格信息价.md", "file:docs/成本测算.xlsx"] },
  { query: "振动锤选型要点", expected: ["knowledge:振动锤选型要点", "office:桩基施工-振动锤选型"], topHits: ["knowledge:振动锤选型要点", "file:docs/桩基施工方案.md", "knowledge:桩基检测规范要点"] },
  { query: "投标文件格式要求", expected: ["knowledge:投标文件格式要求", "file:docs/投标文件模板.md"], topHits: ["knowledge:投标文件格式要求", "file:docs/投标文件模板.md", "office:项目-投标文件"] },
  { query: "挖掘机 220 台班 租赁价", expected: ["cost:挖掘机 220 台班"], topHits: ["cost:挖掘机 220 台班", "cost:桩机台班费"] },
  { query: "三轴搅拌桩 水泥掺量 20%", expected: ["knowledge:三轴水泥搅拌桩工艺要点", "cost:桩机台班费"], topHits: ["knowledge:三轴水泥搅拌桩工艺要点", "cost:桩机台班费", "knowledge:抗浮设计要点"] },
  { query: "地下室抗浮水位 设计要求", expected: ["office:项目-地下室抗浮水位", "knowledge:抗浮设计要点"], topHits: ["knowledge:抗浮设计要点", "file:docs/基坑设计说明.md"] },
  { query: "桩基检测 低应变 规范", expected: ["knowledge:桩基检测规范要点"], topHits: ["knowledge:桩基检测规范要点", "knowledge:桩基施工要点"] },
  { query: "螺纹钢 HRB400 价格", expected: ["cost:螺纹钢 HRB400"], topHits: ["cost:螺纹钢 HRB400", "cost:螺纹钢 HRB400E", "file:docs/材料价格信息价.md"] },
  { query: "C30 泵送混凝土 单价", expected: ["cost:泵送商品混凝土 C30"], topHits: ["cost:泵送商品混凝土 C30", "cost:商品混凝土 C30（泵送）", "file:docs/材料价格信息价.md"] },
  { query: "清单计价 综合单价 组成", expected: ["knowledge:综合单价组成"], topHits: ["knowledge:综合单价组成", "knowledge:清单计价规范要点"] },
  { query: "土方开挖 放坡 安全要求", expected: ["knowledge:土方开挖放坡要求", "office:项目-土方开挖方案"], topHits: ["knowledge:土方开挖放坡要求", "office:项目-土方开挖方案", "file:docs/基坑支护方案.md"] },
];

// evalHitMatched / evalRecall：与 Go evalHitMatched / evalRecall 同规则。
function evalHitMatched(expectedKind: string, expectedName: string, hitKind: string, hitName: string): boolean {
  if (expectedKind !== hitKind) return false;
  if (expectedName === "" || hitName === "") return expectedName === hitName;
  return expectedName === hitName || hitName.includes(expectedName) || expectedName.includes(hitName);
}

function evalRecall(expected: string[], topHits: string[]): number {
  if (expected.length === 0) return 0;
  let matched = 0;
  for (const exp of expected) {
    const ei = exp.indexOf(":");
    if (ei <= 0 || ei === exp.length - 1) continue;
    const ek = exp.slice(0, ei);
    const en = exp.slice(ei + 1);
    for (const hit of topHits) {
      const hi = hit.indexOf(":");
      if (hi <= 0 || hi === hit.length - 1) continue;
      if (evalHitMatched(ek, en, hit.slice(0, hi), hit.slice(hi + 1))) {
        matched++;
        break;
      }
    }
  }
  return matched / expected.length;
}

export function buildRetrieval(_s: MakeMockState): RetrievalMethods {
  return {
    async SemanticSearch(query: string) {
      if (!query.trim()) return [];
      return [
        {
          kind: "cost", name: "hp300", score: 0.86,
          text: "HP300 高频液压振动锤（300kW） 单位台班 单价3200元 分类机械 来源市场询价",
        },
        {
          kind: "knowledge", name: "桩基-施工要点", score: 0.71,
          text: "桩基施工要点 工程案例 振动锤选型需匹配地质条件…",
        },
      ];
    },
    // ── 跨库统一检索（mock：关键词 + 语义各 1-2 条，与后端 topN 参数一致）──
    async UnifiedSearch(query: string, topN = 10) {
      if (!query.trim()) return { keyword: [], semantic: [] };
      const kw = await this.WorkspaceSearch(query, topN);
      const sem = await this.SemanticSearch(query);
      return {
        keyword: kw.length ? kw.slice(0, 2) : [],
        semantic: sem.length ? sem.slice(0, 2) : [],
      };
    },
    // ── 检索质量测评（契约对齐 Go；topHits 为演示命中）──
    async RetrievalEvalRun(): Promise<RetrievalEvalReport> {
      const perQuery = RETRIEVAL_EVAL_SET.map((row) => ({
        query: row.query,
        expected: row.expected,
        topHits: row.topHits,
        recall: evalRecall(row.expected, row.topHits),
      }));
      const recallAt10 = perQuery.reduce((sum, q) => sum + q.recall, 0) / perQuery.length;
      return {
        total: perQuery.length,
        threshold: RETRIEVAL_EVAL_THRESHOLD,
        recallAt10,
        passed: recallAt10 >= RETRIEVAL_EVAL_THRESHOLD,
        perQuery,
        note: "匹配规则：expected 与 topHits 均为 kind:name；同 kind 且 name 精确相等或互为子串记命中",
      };
    },
    async FileIndexRebuild() {
      return taskView("file_index", "工作区语义索引", { total: 3, skipped: 0 });
    },
    async FileSemanticSearch(query: string) {
      if (!query.trim()) return [];
      return [
        {
          path: "docs/桩基施工方案.md", score: 0.82,
          snippet: "振动锤选型需匹配地质条件，HP300 高频液压振动锤…",
        },
        {
          path: "docs/成本测算.xlsx", score: 0.63,
          snippet: "机械台班单价明细：挖掘机、振动锤…",
        },
      ];
    },
  };
}
