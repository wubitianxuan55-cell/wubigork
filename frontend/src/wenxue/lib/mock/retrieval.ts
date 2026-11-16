// mock/retrieval.ts — 检索/索引/测评域（T6-10.1 拆分自 lib/mock.ts）。
// 从 office.ts 独立拆分（原 office 域超过 400 行预算）；方法体除
// RetrievalEvalRun 契约对齐（T6-10.4）外零改动。
import type { AppBindings } from "../bridge";
import type { IntentResultView, RetrievalEvalReport, WorkspaceSearchHit } from "../types";
import { taskView } from "./shared";
import type { MakeMockState } from "./state";

type RetrievalMethods = Pick<
  AppBindings,
  | "SemanticSearch" | "UnifiedSearch" | "RouteIntent" | "RetrievalEvalRun"
  | "FileIndexRebuild" | "FileSemanticSearch"
>;

/** WorkspaceSearch 实现在 office 域，运行时经 mock 聚合后同对象可用；按方法形态收窄引用。 */
type WithWorkspaceSearch = {
  WorkspaceSearch(query: string, limit?: number): Promise<WorkspaceSearchHit[]>;
};

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
  { query: "灯塔守夜人 职责", expected: ["file:章节/001-雾港夜航.md"], topHits: ["file:章节/001-雾港夜航.md", "file:设定/世界观-雾港.md", "file:角色/沈青梧.md"] },
  { query: "沈青梧 性格 背景", expected: ["file:角色/沈青梧.md"], topHits: ["file:角色/沈青梧.md", "file:章节/001-雾港夜航.md", "file:设定/世界观-雾港.md"] },
  { query: "三十年前 沉船 名单", expected: ["file:章节/002-灯塔来信.md", "file:时间线/时间线.md"], topHits: ["file:章节/002-灯塔来信.md", "file:时间线/时间线.md", "file:设定/世界观-雾港.md"] },
  { query: "灯语 力量体系 规则", expected: ["file:设定/力量体系-灯语.md"], topHits: ["file:设定/力量体系-灯语.md", "file:设定/世界观-雾港.md", "file:章节/002-灯塔来信.md"] },
  { query: "陆知远 与 沈青梧 关系", expected: ["file:角色/陆知远.md", "file:角色/沈青梧.md"], topHits: ["file:角色/陆知远.md", "file:角色/沈青梧.md", "file:章节/002-灯塔来信.md"] },
  { query: "雾港 地理 码头", expected: ["file:设定/世界观-雾港.md"], topHits: ["file:设定/世界观-雾港.md", "file:章节/001-雾港夜航.md", "file:设定/力量体系-灯语.md"] },
  { query: "第2章 事件 时间线", expected: ["file:时间线/时间线.md"], topHits: ["file:时间线/时间线.md", "file:章节/002-灯塔来信.md"] },
  { query: "雾港夜航 定稿 开篇", expected: ["file:定稿/001-雾港夜航.md"], topHits: ["file:定稿/001-雾港夜航.md", "file:章节/001-雾港夜航.md"] },
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
          kind: "knowledge", name: "雾港-世界观", score: 0.86,
          text: "雾港世界观 南方海岸手工灯塔城市 渔业与船舶维修 灯塔光「灯语」传讯…",
        },
        {
          kind: "knowledge", name: "灯语-力量体系", score: 0.71,
          text: "灯语编码 长亮=平安 两短一长=有人到港 连续急闪=封港 掌握完整灯语表者仅剩黑匣子…",
        },
      ];
    },
    // ── 跨库统一检索（mock：关键词 + 语义 + 三脑 + 文件语义各 1-2 条，
    // 与后端 GaeaUnifiedSearch 四组视图一致；S1.2-B 起带 scope 参数：
    // ""=全部/旧行为，"work"/"play"=只搜对应空间）──
    // 过滤规则对齐设计 docs/gaea-memory-isolation-design.md 检索面地图：
    //   · brain：brain.left/main 属 work，brain.right（轻语）属 play；
    //     work scope 滤 right，play scope 只留 right。
    //   · semantic：只对最终 hits 过滤（严禁动索引源）——演示数据里
    //     cost/knowledge/office 为 work 域，play scope 返回 play 分区命中。
    //   · keyword/files：工作区单根目录=共享面，不过滤。
    async UnifiedSearch(query: string, topN = 10, scope = "") {
      if (!query.trim()) return { keyword: [], semantic: [], brain: [], files: [] };
      const kw = await (this as unknown as WithWorkspaceSearch).WorkspaceSearch(query, topN);
      const sem = await this.SemanticSearch(query);
      const files = await this.FileSemanticSearch(query, topN);
      const workBrain = [
        {
          brain: "brain.left",
          entity: "雾港来信-写作偏好",
          text: "图文脑写作偏好：悬疑基调，短句造雾感；每章先摸时间线再落笔。",
          score: 0.9,
        },
      ];
      const playBrain = [
        {
          brain: "brain.right",
          entity: "游戏偏好",
          text: "轻语记忆：偏好策略与模拟经营类游戏。",
          score: 0.88,
        },
      ];
      const workSemantic = sem.length ? sem.slice(0, 2) : [];
      const playSemantic = [
        {
          kind: "office" as const,
          name: "娱乐偏好-策略模拟",
          score: 0.77,
          text: "乐园分区记忆：偏好策略与模拟经营类游戏，周末常玩。",
        },
      ];
      return {
        keyword: kw.length ? kw.slice(0, 2) : [],
        semantic:
          scope === "play" ? playSemantic : scope === "work" ? workSemantic : [...workSemantic, ...playSemantic],
        brain: scope === "play" ? playBrain : scope === "work" ? workBrain : [...workBrain, ...playBrain],
        files: files.length ? files.slice(0, 2) : [],
      };
    },
    // ── 统一意图路由（v4.7 S4.6 命令面板接内核）：演示规则与 internal/intent
    // Parse 同方向（导航>生图>状态>提醒；宁漏勿误）；dryRun=true 零副作用，
    // 只回「将发生什么」预览语 + action/target（与 Go intentPreview 同构）。
    async RouteIntent(text: string, dryRun: boolean): Promise<IntentResultView> {
      const q = text.trim();
      const boards: Array<[string, string]> = [
        ["首页", "home"], ["轻语", "chat"], ["小说", "novel"], ["绘梦", "imagegen"],
        ["办公", "gaea"], ["造价", "cost"], ["记忆", "memoryhub"], ["模型", "modelcenter"],
        ["角色", "characterlib"], ["设置", "settings"], ["微信", "weixin"],
      ];
      const mk = (action: string, target: string, preview: string, execReply: string): IntentResultView =>
        dryRun ? { reply: preview, handled: true, action, target } : { reply: execReply, handled: true, action, target };
      const hit = boards.find(([alias]) => q.includes(alias));
      if (hit && /打开|进入|切换到|切到|跳到|转到|回到|回|返回|去/.test(q)) {
        const [label, id] = hit;
        return mk("navigate", id, `将打开「${label}」板块`, `好，已打开${label}。`);
      }
      const img = /^(?:请|麻烦|帮我)?(?:给我)?画(?:一张|一幅|一个|张|出)?(.+)$/.exec(q)
        ?? /^(?:请|麻烦|帮我)?(?:给我)?生成(?:一张|一幅|一个)?(?:的)?(?:图片|图像|画|一张图)(?:[，,：:]|一下)?(.*)$/.exec(q);
      if (img?.[1]?.trim()) {
        const desc = img[1].trim();
        return mk("generate_image", desc, `将生成图片：${desc}（默认模型与尺寸，完成后到绘梦查看）`, `好，开始生成：${desc}。完成后到绘梦板块查看。`);
      }
      if (/模型|引擎/.test(q)) {
        return mk("status", "model", "将查询当前模型引擎状态", "当前状态：Herdsman（本地）。");
      }
      if (/(?:读|念|看|识别).{0,2}屏幕|屏幕.{0,2}(?:读|念|看)|屏幕上.{0,4}(?:什么|啥)|读屏/.test(q)) {
        return mk("read_screen", "screen", "将截取屏幕并识别屏幕上的文字（OCR）", "屏幕上的文字如下：（演示）这是 mock 环境的屏幕文字。");
      }
      if (/提醒|叫我/.test(q)) {
        return mk("reminder", "", "将设提醒（时间解析后确认）", "好，已设提醒（演示）。到点我用微信叫你。");
      }
      return { reply: "", handled: false };
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
    // topN 形参对齐 Go 门面签名（X1-11 契约测试）：mock 固定两条样例，只收不用。
    async FileSemanticSearch(query: string, _topN: number) {
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
