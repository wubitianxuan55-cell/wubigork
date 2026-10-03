// mock/contract.test.ts — dev mock × Go 绑定签名契约测试（审计 X1-11「前端 mock
// 是第三份手写契约，已实际分叉咬人」）。
//
// 背景：dev mock（本目录按「前端怎么调」手写，~5300 行行为面）与 Go 绑定签名
// 之间没有任何机器比对——tsc 对 mock 的完备性与形参个数都测不出：
//   · 完备性：makeMockApp 用 Object.assign 聚合 16 个源，超出 TS 内建重载数，
//     落到 `assign(target: object, ...sources: any[]): any`——返回 any，
//     `: AppBindings` 注解形同虚设，漏实现的绑定 tsc 不红，浏览器 dev 一调
//     即 `undefined is not a function`；
//   · 形参个数：少声明形参的函数对接口签名仍可赋值（TS 函数赋值规则），行为
//     测试按 mock 自己的形参序自洽地过——错位拖到真机才炸。
// 本测试用 scripts/gen_bindings -signatures-ts 生成的 Go 侧签名清单
// （./bindingSignatures）逐名钉死两条缝：
//
//   1. 签名清单 ↔ bindingNames 同源同步——清单过期（Go 改了没再生）即红
//      （bindingNames 由 check-bindings-drift CI 锚定 Go 侧，不另建单源）；
//   2. mock 每个方法必须对得上一个 Go 签名（经 gaeaToGaea 映射；无 Go 绑定的
//      mock-only 名以 drift.ts 的 MOCK_ONLY_NAMES 单源豁免）；
//   3. mock 每个方法的声明形参个数与门面 argc 逐名一致；原生变参（...T 已被
//      门面收窄为单值，v4.285）按「至少 argc-1」放宽；逐名豁免见
//      ARITY_ALLOWANCES（条目过期即红，防豁免腐化）；
//   4. 清单内、legacy 面外、mock 缺失 = 漏 mock。有意不 mock 的名字逐条登记在
//      NOT_MOCKED（带理由；条目过期/已被实现即红）。legacy 绑定面不经
//      AppBindings 消费、浏览器 dev 不路由到 mock，属结构性不 mock，以
//      legacyBindings 单源清单为准——勿在此手抄副本：手抄就成了本审计要杀的
//      第四份手写契约。
import { describe, expect, it } from "vitest";
import { bindingNames } from "../bindingNames";
import { legacyBindings } from "../legacyBindings";
import { MOCK_ONLY_NAMES } from "../bridge/drift";
import { gaeaToGaea } from "../bridge/mappings";
import { makeMockApp } from "../mock";
import { bindingSignatureCount, bindingSignatures } from "./bindingSignatures";

type AnyFn = (...args: unknown[]) => unknown;

// mockName → Go 绑定名（与 bridge/proxy.ts realApp 的路由口径逐字一致：
// 映射表命中用映射目标，否则同名直调）。
function goTargetOf(mockName: string): string {
  return (gaeaToGaea as Record<string, string>)[mockName] ?? mockName;
}

// ── 声明形参计数（判据）─────────────────────────────────────────────
// 为什么不用 fn.length：JS 的 length 只数「首个带默认值/剩余参数之前的形参」，
// mock 里合法的默认值形参（如 FileSearch(query, limit = 30)）会被少算——按
// length 断言会逼人删默认值，反而改行为。这里从 fn.toString() 抠形参区
// （括号配平扫描，跳过字符串/模板字符串），按顶层逗号计数：数的是**声明了的
// 形参位置数**，即错位咬人的那个量。vitest 无压缩，源码文本保真。
function topLevelSplit(s: string): string[] {
  const out: string[] = [];
  let depth = 0;
  let quote: string | null = null;
  let cur = "";
  for (let i = 0; i < s.length; i++) {
    const c = s[i];
    if (quote) {
      cur += c;
      if (c === "\\") { cur += s[i + 1] ?? ""; i++; continue; }
      if (c === quote) quote = null;
      continue;
    }
    if (c === "'" || c === '"' || c === "`") { quote = c; cur += c; continue; }
    if (c === "(" || c === "[" || c === "{") depth++;
    else if (c === ")" || c === "]" || c === "}") depth--;
    if (c === "," && depth === 0) { out.push(cur.trim()); cur = ""; continue; }
    cur += c;
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}

function declaredParamText(fn: AnyFn): string {
  const src = fn.toString();
  const open = src.indexOf("(");
  if (open < 0) return "";
  let depth = 0;
  let quote: string | null = null;
  for (let i = open; i < src.length; i++) {
    const c = src[i];
    if (quote) {
      if (c === "\\") { i++; continue; }
      if (c === quote) quote = null;
      continue;
    }
    if (c === "'" || c === '"' || c === "`") { quote = c; continue; }
    if (c === "(") depth++;
    else if (c === ")") { depth--; if (depth === 0) return src.slice(open + 1, i); }
  }
  return "";
}

function declaredParamCount(fn: AnyFn): number {
  const text = declaredParamText(fn).trim();
  if (!text) return 0;
  return topLevelSplit(text).length;
}

// 失败报点用：形参清单原文（诊断展示，不作判据）。
function declaredParamsText(fn: AnyFn): string {
  return topLevelSplit(declaredParamText(fn)).join(" | ") || "（无参）";
}

const mockApp = makeMockApp();
const mockMethods = Object.entries(mockApp).filter(
  (entry): entry is [string, AnyFn] => typeof entry[1] === "function",
);
const mockKeys = new Set(mockMethods.map(([name]) => name));
const coveredTargets = new Set(mockMethods.map(([name]) => goTargetOf(name)));
const manifestKeys = Object.keys(bindingSignatures);
const legacySet = new Set<string>(legacyBindings);
const mockOnlySet = new Set<string>(MOCK_ONLY_NAMES);

// ── 形参个数逐名豁免（条目必须仍然「正在错位」，不再错位即红 → 删条目）──────
// GaeaSend：前端接口 SubmitDisplay(display, input) 双参、Go 门面 GaeaSend(input)
// 单参——这是桥接面（bridge/core.ts:65）与 Go 之间的既存口径分歧（App.tsx:145
// 「完整信封原文走 SubmitDisplay raw」），不是 mock 手误：mock 的调用方
// store/controller.ts send() 双参调用，mock 砍成单参会吞错位实参（display 落到
// input）。归属桥接面/Go 的线处置；mock 跟随调用方口径，此处豁免并留痕。
const ARITY_ALLOWANCES: Record<string, string> = {
  GaeaSend: "SubmitDisplay 桥接面双参 vs Go GaeaSend 单参（既存口径分歧，mock 跟随调用方；处置归桥接面线）",
};

// ── 有意不 mock 白名单（批 46 清空）──────────────────────────────────
// 历史语义：曾登记「被 AppBindings 认领但 dev mock 未实现」的名字（X1-11 实证
// 清单，浏览器调用即 TypeError）。批 43~46 分四刀补齐全部绑定（AI 面为诚实
// 拒绝 mock，读/状态面为 Go 口径空态/退化值），白名单清空——此后**漏 mock 锁
// 全量接管**：任何新绑定不补 mock 直接红（fail-closed），例外须在此重新登记
// 并写明理由。
const NOT_MOCKED: Record<string, string> = {};

describe("dev mock × Go 绑定签名契约（X1-11）", () => {
  it("签名清单与 bindingNames 同源同步（清单过期/漏再生即红）", () => {
    expect(Object.keys(bindingSignatures).length).toBe(bindingSignatureCount);
    expect(manifestKeys.sort()).toEqual([...bindingNames].sort());
  });

  it("mock 的每个方法都对得上一个 Go 签名（幽灵名即红）", () => {
    const stray: string[] = [];
    for (const name of mockKeys) {
      const target = goTargetOf(name);
      if (!(target in bindingSignatures) && !mockOnlySet.has(name)) {
        stray.push(`${name} → ${target}`);
      }
    }
    expect(stray).toEqual([]);
  });

  it("mock 每个方法的声明形参个数与 Go 门面签名一致（错位即红）", () => {
    const wrong: string[] = [];
    for (const [name, fn] of mockMethods) {
      const target = goTargetOf(name);
      const sig = bindingSignatures[target as keyof typeof bindingSignatures];
      if (!sig) continue; // mock-only 名，上一条已管
      // 非变参：精确相等——多声明（实参落错位）与少声明（吞参）都咬人。
      // 原生变参：门面把 ...T 收窄为单值 T（argc 含它计 1），mock 可不声明该尾参。
      const n = declaredParamCount(fn);
      const ok = sig.variadic ? n >= sig.argc - 1 : n === sig.argc;
      if (!ok && !(target in ARITY_ALLOWANCES)) {
        wrong.push(
          `${name}（mock ${n} 参 vs 门面 ${sig.argc} 参，variadic=${sig.variadic}；Go: ${sig.params || "无参"}；mock 声明: ${declaredParamsText(fn)}）`,
        );
      }
    }
    expect(wrong).toEqual([]);
  });

  it("清单内、legacy 面外的名字无漏 mock（漏 mock 即红）", () => {
    const missing = manifestKeys.filter(
      (n) => !coveredTargets.has(n) && !legacySet.has(n) && !(n in NOT_MOCKED),
    );
    expect(missing).toEqual([]);
  });

  it("白名单与豁免清单不过期（条目失效即红，防例外清单腐化）", () => {
    const stale: string[] = [];
    for (const [name, why] of Object.entries(NOT_MOCKED)) {
      if (!(name in bindingSignatures)) stale.push(`NOT_MOCKED[${name}] 已不是 Go 绑定（Go 侧删除/改名）——删条目：${why}`);
      else if (coveredTargets.has(name)) stale.push(`NOT_MOCKED[${name}] 已被 mock 实现——删条目让 arity/漏 mock 锁接管：${why}`);
    }
    for (const [name, why] of Object.entries(ARITY_ALLOWANCES)) {
      if (!(name in bindingSignatures)) {
        stale.push(`ARITY_ALLOWANCES[${name}] 已不是 Go 绑定——删条目：${why}`);
        continue;
      }
      // 条目必须仍然「正在错位」：mock 已对齐门面（或无人再映射到它）即过期。
      const stillOff = mockMethods.some(([mockName, fn]) => {
        if (goTargetOf(mockName) !== name) return false;
        const sig = bindingSignatures[name as keyof typeof bindingSignatures];
        const n = declaredParamCount(fn);
        return sig.variadic ? n < sig.argc - 1 : n !== sig.argc;
      });
      if (!stillOff) stale.push(`ARITY_ALLOWANCES[${name}] 已无错位（mock 对齐门面或映射已移除）——删条目：${why}`);
    }
    expect(stale).toEqual([]);
  });
});
