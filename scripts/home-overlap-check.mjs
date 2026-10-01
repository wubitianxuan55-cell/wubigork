// 首页卡片重叠检查（v11.4 一屏自适应配套，2026-10-01）
// 用法：node scripts/home-overlap-check.mjs [--cdp 9333 | --url http://127.0.0.1:9344]
//       [--viewport WxH]   # 缺省 1600x1000；可多次传入扫多档
// 原理：首页所有卡片元素（.ml-card/.w-mod/.p-poster/…）两两做矩形相交（含容跳过），
//       部分相交 >2px 即报。配合 Emulation override 可在真机//dev 任意窗口档扫描。
// 背景：一屏自适应 grid 的 fr 轨道若无显式下限，矮窗下压瘪且 scrollHeight 不涨，
//       表现为重叠而非滚动——目检不可靠，必须程序化扫描（v11.4~7.14 三族实锤）。
const args = process.argv.slice(2);
const argOf = (k, d) => { const i = args.indexOf(k); return i > -1 ? args[i + 1] : d; };
const cdpPort = argOf("--cdp", "");
const url = argOf("--url", "");
const viewports = args.filter((a) => /^\d+x\d+$/.test(a)).map((s) => s.split("x").map(Number));

let wsBase, targetUrl;
if (cdpPort) {
  wsBase = `http://127.0.0.1:${cdpPort}`;
  const list = await (await fetch(`${wsBase}/json`)).json();
  const page = list.find((t) => t.type === "page");
  if (!page) throw new Error("CDP 无 page target");
  targetUrl = page.webSocketDebuggerUrl;
} else if (url) {
  // dev 直连：借 ZCode IAB/CDP 不具备时，退化为仅提示（dev 走浏览器工具更顺手）
  console.error("dev URL 模式请配合 CDP 端口使用；dev 下建议 --cdp 9333 或浏览器工具");
  process.exit(2);
} else {
  wsBase = "http://127.0.0.1:9333";
  const list = await (await fetch(`${wsBase}/json`)).json();
  const page = list.find((t) => t.type === "page");
  if (!page) throw new Error("CDP 9333 无 page target（壳未开或调试口未启）");
  targetUrl = page.webSocketDebuggerUrl;
}

const ws = new WebSocket(targetUrl);
let id = 0;
const pending = new Map();
const send = (method, params = {}) => new Promise((res, rej) => {
  const mid = ++id;
  pending.set(mid, { res, rej });
  ws.send(JSON.stringify({ id: mid, method, params }));
  setTimeout(() => { if (pending.has(mid)) { pending.delete(mid); rej(new Error("timeout: " + method)); } }, 10000);
});
ws.addEventListener("message", (ev) => {
  const m = JSON.parse(ev.data);
  if (m.id && pending.has(m.id)) { const { res, rej } = pending.get(m.id); pending.delete(m.id); m.error ? rej(new Error(m.error.message)) : res(m.result); }
});
await new Promise((res, rej) => { ws.addEventListener("open", res); ws.addEventListener("error", rej); });

const SIZES = viewports.length ? viewports : [[1600, 1000]];
const FN = `(() => {
  const els = Array.from(document.querySelectorAll('.ml-card, .w-mod, .p-poster, .ml-strip, .w-cap, .p-wall, .t-foot'));
  const rects = els.map(e => ({ cls: String(e.className).split(' ').slice(0,2).join('.'), r: e.getBoundingClientRect() }))
    .filter(x => x.r.width > 40 && x.r.height > 20);
  const bad = [];
  for (let i = 0; i < rects.length; i++) for (let j = i + 1; j < rects.length; j++) {
    const a = rects[i].r, b = rects[j].r;
    const contains = (x, y) => x.left <= y.left + 1 && x.right >= y.right - 1 && x.top <= y.top + 1 && x.bottom >= y.bottom - 1;
    if (contains(a, b) || contains(b, a)) continue;
    const ox = Math.min(a.right, b.right) - Math.max(a.left, b.left);
    const oy = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top);
    if (ox > 2 && oy > 2) bad.push(rects[i].cls + ' × ' + rects[j].cls + ' (' + Math.round(ox) + 'x' + Math.round(oy) + ')');
  }
  const ml = document.querySelector('.ml');
  return { space: document.querySelector('.ml-work') ? 'work' : (document.querySelector('.ml-play') ? 'play' : 'tasks'),
           over: ml ? ml.scrollHeight - ml.clientHeight : -1, overlaps: bad };
})()`;

let failed = false;
for (const [w, h] of SIZES) {
  await send("Emulation.setDeviceMetricsOverride", { width: w, height: h, deviceScaleFactor: 0, mobile: false });
  await new Promise((r) => setTimeout(r, 600));
  const r = await send("Runtime.evaluate", { expression: FN, returnByValue: true });
  const v = r.result.value;
  const tag = v.overlaps.length ? "FAIL" : "ok";
  if (v.overlaps.length) failed = true;
  console.log(`[${tag}] ${w}x${h} ${v.space} over=${v.over}${v.overlaps.length ? " → " + v.overlaps.join("; ") : ""}`);
}
await send("Emulation.clearDeviceMetricsOverride");
ws.close();
process.exit(failed ? 1 : 0);
