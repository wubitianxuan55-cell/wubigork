#!/usr/bin/env node
/**
 * gaea 前端 E 系列回归守卫（无前端测试框架时的静态不变式检查）。
 *
 * 覆盖 docs/evaluation-set.md 的前端项：
 *  - E16: QUICK_REPLIES 常量必须在模块顶层（不得插进组件体内）
 *  - E22: 降级态必须禁用 AI 控制台自定义入场动画
 *  - E23: WebView2 rAF 节流降级（main.tsx 检测 + index.css antd motion 禁用）
 *  - E24: 记忆中枢 3D 图谱必须用 3d-force-graph（误装底层库会白屏）
 *
 * 任一检查失败 → 非零退出，CI 拦截。
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const read = (rel) => readFileSync(path.join(root, rel), "utf8");

let failed = false;
const ok = (msg) => console.log(`  ✓ ${msg}`);
const bad = (msg) => {
  failed = true;
  console.error(`  ✗ ${msg}`);
};

const section = (name) => console.log(`\n[${name}]`);

// ── E16：QUICK_REPLIES 模块顶层 ─────────────────────────────
section("E16 QUICK_REPLIES 常量位置");
{
  const src = read("frontend/src/pages/WhisperPage.tsx");
  const declIdx = src.indexOf("const QUICK_REPLIES = [");
  const compIdx = src.indexOf("const WhisperPage:");
  if (declIdx < 0) bad("WhisperPage.tsx 找不到 QUICK_REPLIES 声明");
  else ok("QUICK_REPLIES 声明存在");
  if (compIdx < 0) bad("WhisperPage.tsx 找不到组件声明（无法校验作用域）");
  else if (declIdx >= 0 && declIdx > compIdx) {
    bad("QUICK_REPLIES 声明在组件之后——可能被插回组件体内");
  } else if (declIdx >= 0) {
    ok(`QUICK_REPLIES 在组件声明之前（模块顶层，行 ${src.slice(0, declIdx).split("\n").length}）`);
  }
}

// ── E22：降级态禁用 AI 控制台入场动画 ──────────────────────
section("E22 AI 控制台降级动画");
{
  const css = read("frontend/src/index.css");
  if (css.includes("html.gaea-raf-degraded .ai-console-panel")) {
    ok("gaea-raf-degraded 覆盖 .ai-console-panel");
  } else {
    bad("index.css 缺少 .ai-console-panel 降级规则（控制台可能卡在入场首帧）");
  }
}

// ── E23：WebView2 rAF 节流降级 ──────────────────────────────
section("E23 rAF 节流降级");
{
  const ts = read("frontend/src/main.tsx");
  const css = read("frontend/src/index.css");
  const hasDetect =
    ts.includes("gaea-raf-degraded") &&
    ts.includes("window.setTimeout(() => cb(start + 16), 16)") &&
    ts.includes("fps < 30");
  if (hasDetect) ok("main.tsx rAF 帧率检测 + setTimeout 模拟降级存在");
  else bad("main.tsx 缺少 rAF 降级探测/模拟（下拉可能打不开）");

  const hasEnter =
    css.includes("html.gaea-raf-degraded [class*='ant-slide-up-enter']") &&
    css.includes("animation: none !important");
  const hasLeave =
    css.includes("html.gaea-raf-degraded [class*='ant-slide-up-leave']") &&
    css.includes("display: none !important");
  if (hasEnter) ok("antd 弹层 enter 动画已禁用（立即显示）");
  else bad("index.css 缺少 antd 弹层 enter 降级规则");
  if (hasLeave) ok("antd 弹层 leave 动画已禁用（立即隐藏）");
  else bad("index.css 缺少 antd 弹层 leave 降级规则");
}

// ── E24：记忆中枢 3D 图谱依赖 ───────────────────────────────
section("E24 记忆中枢 3D 图谱");
{
  const gv = read("frontend/src/gaea/components/memoryhub/GraphView.tsx");
  const mhp = read("frontend/src/pages/MemoryHubPage.tsx");
  const pkg = read("frontend/package.json");
  if (gv.includes('import ForceGraph3D from "3d-force-graph"')) ok("GraphView 使用 3d-force-graph");
  else bad("GraphView 未使用 3d-force-graph（误装底层库会白屏）");
  if (gv.includes("(ForceGraph3D as any)()(containerRef.current)")) ok("ForceGraph3D 初始化写法正确");
  else bad("ForceGraph3D 初始化写法异常");
  if (mhp.includes('from "../gaea/components/memoryhub/GraphView"')) ok("MemoryHubPage 已挂载 GraphView");
  else bad("MemoryHubPage 未引用 GraphView");
  if (pkg.includes('"3d-force-graph"')) ok("package.json 声明 3d-force-graph 依赖");
  else bad("package.json 缺少 3d-force-graph 依赖");
}

console.log("");
if (failed) {
  console.error("前端 E 系列回归守卫失败：请修复后重试。");
  process.exit(1);
}
console.log("前端 E 系列回归守卫 OK");
