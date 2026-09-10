// 临时真机走查工具：CDP 9333 → Runtime.evaluate / Page.captureScreenshot
// 用法：node .tmp/cdp-walk.mjs "<js 表达式>" [--shot out.png] [--await]
//      [--target <url 片段>]  # 多标签时选定目标页（缺省取第一个 page）
const list = await (await fetch("http://127.0.0.1:9333/json")).json();
const targetIdx = process.argv.indexOf("--target");
const targetHint = targetIdx > 0 ? process.argv[targetIdx + 1] : "";
const pages = list.filter((t) => t.type === "page");
const page = targetHint
  ? pages.find((t) => (t.url ?? "").includes(targetHint))
  : pages[0];
if (!page) {
  throw new Error(`no page target${targetHint ? ` matching ${targetHint}` : ""}（现有：${pages.map((p) => p.url).join(", ")}）`);
}
const ws = new WebSocket(page.webSocketDebuggerUrl);
let id = 0;
const pending = new Map();
function send(method, params = {}) {
  return new Promise((res, rej) => {
    const mid = ++id;
    pending.set(mid, { res, rej });
    ws.send(JSON.stringify({ id: mid, method, params }));
  });
}
ws.onmessage = (ev) => {
  const msg = JSON.parse(ev.data);
  if (msg.id && pending.has(msg.id)) {
    const { res, rej } = pending.get(msg.id);
    pending.delete(msg.id);
    msg.error ? rej(new Error(JSON.stringify(msg.error))) : res(msg.result);
  }
};
await new Promise((res, rej) => { ws.onopen = res; ws.onerror = rej; });

const arg = process.argv[2] ?? "";
// 以 @ 开头 = 从文件读取表达式（PowerShell 5.1 传原生命令参数会吃掉内层引号，
// 长表达式一律写文件再用 @路径 传入，避免被 shell 重写）
const expr = arg.startsWith("@")
  ? (await import("node:fs")).readFileSync(arg.slice(1), "utf8")
  : arg;
const shotIdx = process.argv.indexOf("--shot");
const awaitMode = process.argv.includes("--await");
const mouseIdx = process.argv.indexOf("--mouse");
const clickIdx = process.argv.indexOf("--click");
const keyIdx = process.argv.indexOf("--key");

if (mouseIdx > 0 || clickIdx > 0) {
  if (clickIdx > 0) {
    // 同连接内：先移到左缘热区展开 rail，再 move 到目标并按下
    await send("Input.dispatchMouseEvent", { type: "mouseMoved", x: 5, y: parseInt(process.argv[clickIdx + 2] ?? "400", 10) || 400 });
    await new Promise((r) => setTimeout(r, 500));
    const [x, y] = process.argv[clickIdx + 1].split(",").map(Number);
    await send("Input.dispatchMouseEvent", { type: "mouseMoved", x, y });
    await new Promise((r) => setTimeout(r, 120));
    await send("Input.dispatchMouseEvent", { type: "mousePressed", x, y, button: "left", clickCount: 1 });
    await send("Input.dispatchMouseEvent", { type: "mouseReleased", x, y, button: "left", clickCount: 1 });
  } else {
    const [x, y] = process.argv[mouseIdx + 1].split(",").map(Number);
    await send("Input.dispatchMouseEvent", { type: "mouseMoved", x, y, button: "none" });
  }
  await new Promise((r) => setTimeout(r, 600));
  console.log(`mouse${clickIdx > 0 ? "+click" : ""} done`);
} else if (keyIdx > 0) {
  const key = process.argv[keyIdx + 1];
  await send("Input.dispatchKeyEvent", { type: "keyDown", key, code: key, windowsVirtualKeyCode: 0 });
  await send("Input.dispatchKeyEvent", { type: "keyUp", key, code: key, windowsVirtualKeyCode: 0 });
  await new Promise((r) => setTimeout(r, 400));
  console.log(`key ${key}`);
} else if (shotIdx > 0) {
  const out = process.argv[shotIdx + 1];
  const r = await send("Page.captureScreenshot", { format: "png" });
  const { writeFileSync } = await import("node:fs");
  writeFileSync(out, Buffer.from(r.data, "base64"));
  console.log(`saved ${out}`);
} else if (arg) {
  const r = await send("Runtime.evaluate", {
    expression: expr,
    returnByValue: true,
    awaitPromise: awaitMode,
  });
  if (r.exceptionDetails) {
    console.error("EXCEPTION:", JSON.stringify(r.exceptionDetails.exception?.description ?? r.exceptionDetails, null, 2));
    process.exit(1);
  }
  console.log(JSON.stringify(r.result.value, null, 2));
}
ws.close();
