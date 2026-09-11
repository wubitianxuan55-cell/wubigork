#!/usr/bin/env node
/**
 * gaea 前端 E 系列回归守卫（无前端测试框架时的静态不变式检查）。
 *  - E16: QUICK_REPLIES 常量化必须在模块顶层（合并后指向 ChatPage，不得插进组件体内）
 *  - E22: 降级态必须禁用 AI 控制台自定义入场动画
 *  - E23: WebView2 rAF 帧率降级（main.tsx 检测 + index.css antd motion 禁用）
 *  - E24: 记忆中枢 3D 图谱必须用 3d-force-graph（误装底层库会白屏）
 *  - E25: 聊天×轻语合并不变量（单一入口 ChatSend / localStorage 迁移 / 菜单无独立轻语）
 *  - E26: 壳层空间切换=纯导航（1B 拍板 2026-09-10：switchSpace 零桥接，不打断在跑的活）
 *  - E27: Go 绑定面变参禁令（v4.237 定论：Wails v2.13 变参绑定不可用——unmarshal/reflect 双口径矛盾，静默坏死）
 *
 * 任一检查失败 → 非零退出，CI 拦截。
 */
import { readFileSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const read = (rel) => readFileSync(path.join(root, rel), "utf8");

let failed = false;
const ok = (msg) => console.log(`   PASS ${msg}`);
const bad = (msg) => {
  failed = true;
  console.error(`   FAIL ${msg}`);
};

const section = (name) => console.log(`\n[${name}]`);

// E16: QUICK_REPLIES 模块顶层（拆分后位于 ChatComposer，ChatPage 导入）
section("E16 QUICK_REPLIES 常量化位置");
{
  const src = read("frontend/src/components/chat/ChatComposer.tsx");
  const declIdx = src.indexOf("const QUICK_REPLIES = [");
  const compIdx = src.indexOf("export const ChatComposer:");
  if (declIdx < 0) bad("ChatComposer.tsx 找不到 QUICK_REPLIES 声明");
  else ok("QUICK_REPLIES 声明存在");
  if (compIdx < 0) bad("ChatComposer.tsx 找不到组件声明（无法校验作用域）");
  else if (declIdx >= 0 && declIdx > compIdx) {
    bad("QUICK_REPLIES 声明在组件之后——可能被插回组件体内");
  } else if (declIdx >= 0) {
    ok(`QUICK_REPLIES 在组件声明之前（模块顶层，行 ${src.slice(0, declIdx).split("\n").length}）`);
  }
}

// E22: 降级态禁用 AI 控制台入场动画
section("E22 AI 控制台降级动画");
{
  const css = read("frontend/src/index.css");
  if (css.includes("html.gaea-raf-degraded .ai-console-panel")) {
    ok("gaea-raf-degraded 覆盖 .ai-console-panel");
  } else {
    bad("index.css 缺少 .ai-console-panel 降级规则（控制台可能卡在入场首帧）");
  }
}

// E23: WebView2 rAF 帧率降级
section("E23 rAF 帧率降级");
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

// E24: 记忆中枢 3D 图谱依赖
section("E24 记忆中枢 3D 图谱");
{
  const gv = read("frontend/src/gaea/components/memoryhub/GraphView.tsx");
  const mhp = read("frontend/src/pages/MemoryHubPage.tsx");
  const pkg = read("frontend/package.json");
  if (gv.includes('import ForceGraph3D from "3d-force-graph"')) ok("GraphView 使用 3d-force-graph");
  else bad("GraphView 未使用 3d-force-graph（误装底层库会白屏）");
  if (gv.includes("const createGraph = ForceGraph3D as unknown as") && gv.includes("createGraph()(containerRef.current)")) {
    ok("ForceGraph3D 初始化写法正确");
  } else {
    bad("ForceGraph3D 初始化写法异常");
  }
  // v4.176 起 GraphView 为页内 React.lazy 挂载（动态 import，无 from 静态形态）。
  if (mhp.includes('import("../gaea/components/memoryhub/GraphView")') || mhp.includes('from "../gaea/components/memoryhub/GraphView"')) ok("MemoryHubPage 已挂载 GraphView");
  else bad("MemoryHubPage 未引用 GraphView");
  if (pkg.includes('"3d-force-graph"')) ok("package.json 声明 3d-force-graph 依赖");
  else bad("package.json 缺少 3d-force-graph 依赖");
}

// E25: 聊天×轻语合并不变量
section("E25 聊天×轻语合并");
{
  const chatPage = read("frontend/src/pages/ChatPage.tsx");
  const stream = read("frontend/src/hooks/useChatStream.ts");
  const utils = read("frontend/src/pages/chat/utils.ts");
  const layout = read("frontend/src/layouts/MainLayout.tsx");
  const models = read("frontend/wailsjs/go/models.ts");
  const bindings = read("frontend/wailsjs/go/app/ChatB.d.ts");

  // v4.174 bridge 双轨退役后生产代码走 bridge（app.ChatSend），直调 wailsjs 形态一并兼容。
  if (stream.includes("app.ChatSend(") || stream.includes("App.ChatSend(")) ok("useChatStream 使用统一入口 ChatSend");
  else bad("useChatStream 未使用 ChatSend（应走统一聊天入口）");

  if (
    (utils.includes("app.ChatImportTopic(") || utils.includes("App.ChatImportTopic(")) &&
    utils.includes("localStorage.removeItem(")
  ) {
    ok("旧 localStorage 话题迁移 chat.db 存在且清理本地键");
  } else {
    bad("chat/utils 缺少 localStorage → ChatImportTopic 迁移路径");
  }

  if (chatPage.includes("const QUICK_REPLIES = [") && chatPage.includes("const ChatPage:")) {
    ok("ChatPage 包含 QUICK_REPLIES 与组件声明");
  }

  if (layout.includes("'whisper'")) bad("MainLayout 仍残留独立 whisper 页面入口");
  else ok("MainLayout 已移除独立轻语页面入口");

  if (models.includes("preview: string") && models.includes("export class ChatMessageInput")) {
    ok("chat.Topic.preview + ChatMessageInput 绑定已生成");
  } else {
    bad("wailsjs 绑定缺少 chat.Topic.preview / ChatMessageInput（需 wails generate module）");
  }

  if (bindings.includes("ChatImportTopic") && bindings.includes("ChatTopicSetMode")) {
    ok("ChatImportTopic / ChatTopicSetMode 绑定已生成");
  } else {
    bad("wailsjs 绑定缺少 ChatImportTopic / ChatTopicSetMode");
  }
}

// E26: 壳层空间切换=纯导航（1B 拍板 2026-09-10「拆开」：切书斋/闲庭是界面导航，
// 不得触引擎——办公在跑时切小说，办公照常。守卫 switchSpace 函数体零桥接调用；
// 引擎空间唯一写入口=办公侧栏 SpaceChip 的 GaeaSpaceActivate，不在此处。）
section("E26 空间切换零桥接");
{
  const layout = read("frontend/src/layouts/MainLayout.tsx");
  const start = layout.indexOf("const switchSpace = useCallback");
  const end = layout.indexOf("const [, forceBoards]");
  if (start === -1 || end === -1 || end <= start) {
    bad("MainLayout 找不到 switchSpace 定义（结构变更，须人工复核 E26 守卫口径）");
  } else {
    const body = layout.slice(start, end);
    const forbidden = ["app.", "wailsApp(", "GaeaSpaceActivate", "noteSpaceActivated", ".Cancel(", "GaeaSend"];
    const hit = forbidden.filter((k) => body.includes(k));
    if (hit.length) bad(`switchSpace 含引擎/桥接调用（违反 1B 拍板）: ${hit.join(", ")}`);
    else ok("switchSpace 零桥接——壳层切换纯导航，不打断在跑的活");
  }
}

// E27: Go 绑定面变参禁令（v4.237 三刀定论：Wails v2.13 对 ...T 绑定，
// ParseArgs 按 []T 反序列化、reflect.Call 按 In(0)=T 校验——双口径矛盾，
// 任何传参形态要么立即报错要么回调永不送达（静默坏死数周）。
// 绑定门面（internal/app/bindings_*.go）一律禁用变参参数；
// 核心层（internal/gaea/**）可保留变参，由门面单参透传。）
section("E27 Go 绑定面变参禁令");
{
  const bindDir = path.join(root, "internal", "app");
  const files = readdirSync(bindDir).filter((f) => /^bindings_.*\.go$/.test(f));
  const hits = [];
  for (const f of files) {
    const lines = readFileSync(path.join(bindDir, f), "utf8").split(/\r?\n/);
    lines.forEach((line, i) => {
      if (/^\s*func \(\w+ \*\w+\) \w+\(.*\.\.\./.test(line)) {
        hits.push(`${f}:${i + 1} ${line.trim().slice(0, 70)}`);
      }
    });
  }
  if (hits.length) {
    for (const h of hits) bad(`变参绑定（Wails v2.13 不可用，改单参透传）: ${h}`);
  } else {
    ok("绑定面无变参参数（GaeaTaskList/GaeaAgentNetwork 族已单参化）");
  }
}

console.log("");
if (failed) {
  console.error("前端 E 系列回归守卫失败：请修复后重试。");
  process.exit(1);
}
console.log("前端 E 系列回归守卫 OK");
