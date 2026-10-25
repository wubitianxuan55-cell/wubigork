// gaea AI 工作台 · 原型交互
// 办公是唯一工作台；CAD / 进度是临时工具面板，不是页签、不是独立板块。

const body = document.body;
const tabs = [...document.querySelectorAll(".tabs .tab[data-view]")];
const toolMenu = document.getElementById("tool-menu");

const CRUMBS = {
  chat: "图纸识别 · 阳光城 3# 楼",
  trajectory: "图纸识别 · 轨迹",
  context: "图纸识别 · 上下文",
  memory: "图纸识别 · 记忆",
  desk: "工作台",
};
const SCOPES = { chat: "对话", trajectory: "轨迹", context: "上下文", memory: "记忆", desk: "工作台" };
const TOOL_CRUMBS = { cad: "工具 · CAD 看图", schedule: "工具 · 进度计划" };
const TOOL_SCOPES = { cad: "图纸", schedule: "计划" };

let lastView = "chat";

function activeView() {
  return body.dataset.view || "chat";
}

function setView(view) {
  if (!["chat", "trajectory", "context", "memory", "desk"].includes(view)) return;
  closeTool({ restore: false });
  body.dataset.view = view;
  lastView = view === "desk" ? lastView : view;
  for (const tab of tabs) {
    const active = tab.dataset.view === view;
    tab.classList.toggle("is-active", active);
    tab.setAttribute("aria-selected", String(active));
    tab.tabIndex = active ? 0 : -1;
  }
  document.querySelectorAll(".work-item").forEach((el) => el.classList.remove("is-active"));
  const current = document.querySelector(`.work-item[data-action="${view}"]`);
  if (current) current.classList.add("is-active");
  const scope = document.getElementById("inspector-scope");
  if (scope) scope.textContent = SCOPES[view] || "检查器";
  const crumb = document.querySelector(".crumb-here");
  if (crumb) crumb.textContent = CRUMBS[view] || "工作台";
}

function openTool(kind) {
  if (!TOOL_CRUMBS[kind]) return;
  body.dataset.tool = kind;
  if (body.dataset.inspector === "closed") body.dataset.inspector = "open";
  const scope = document.getElementById("inspector-scope");
  if (scope) scope.textContent = TOOL_SCOPES[kind];
  const crumb = document.querySelector(".crumb-here");
  if (crumb) crumb.textContent = TOOL_CRUMBS[kind];
  closeToolMenu();
}

function closeTool({ restore = true } = {}) {
  if (!body.dataset.tool) return;
  delete body.dataset.tool;
  if (!restore) return;
  const crumb = document.querySelector(".crumb-here");
  if (crumb) crumb.textContent = CRUMBS[activeView()] || "工作台";
  const scope = document.getElementById("inspector-scope");
  if (scope) scope.textContent = SCOPES[activeView()] || "检查器";
}

function closeToolMenu() {
  if (!toolMenu) return;
  toolMenu.hidden = true;
}

function toggleToolMenu() {
  if (!toolMenu) return;
  toolMenu.hidden = !toolMenu.hidden;
}

document.addEventListener("click", (event) => {
  if (event.target.closest("[data-close-tool]")) {
    event.stopPropagation();
    closeTool();
    return;
  }

  const toolEl = event.target.closest("[data-tool]");
  if (toolEl) {
    event.preventDefault();
    event.stopPropagation();
    openTool(toolEl.dataset.tool);
    return;
  }

  const viewEl = event.target.closest("[data-action]");
  if (!viewEl) return;
  const action = viewEl.dataset.action;

  if (action === "toggle-inspector") {
    body.dataset.inspector = body.dataset.inspector === "closed" ? "open" : "closed";
    return;
  }
  if (action === "tools") {
    toggleToolMenu();
    return;
  }
  if (action === "confirm") {
    body.dataset.inspector = "open";
    const card = document.querySelector(".confirm-card");
    card?.scrollIntoView({ block: "center", behavior: "smooth" });
    card?.classList.add("flash");
    setTimeout(() => card?.classList.remove("flash"), 1600);
    return;
  }
  if (action === "approve") {
    const host = event.target.closest(".approval-card") || event.target.closest(".pending-dock");
    host?.classList.add("is-done");
    host?.querySelectorAll("button").forEach((b) => (b.disabled = true));
    const label = host?.querySelector(".pending-copy b");
    if (label) label.textContent = "已允许 · 已留痕";
    if (viewEl.classList.contains("btn")) viewEl.textContent = "已允许 · 已留痕";
    return;
  }
  if (action === "skip") {
    const card = event.target.closest(".approval-card");
    card?.classList.add("is-done");
    card?.querySelectorAll("button").forEach((b) => (b.disabled = true));
    return;
  }
  if (action === "ask") {
    document.querySelector(".canvas-hud input")?.focus();
    return;
  }
  if (["chat", "trajectory", "context", "memory", "desk"].includes(action)) {
    setView(action);
    return;
  }
  if (action === "open-chat") setView("chat");
});

// 页签键盘漫游
document.querySelector(".tabs")?.addEventListener("keydown", (event) => {
  const list = [...document.querySelectorAll(".tabs .tab[data-view]")];
  const current = list.findIndex((el) => el === document.activeElement);
  if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
    event.preventDefault();
    const dir = event.key === "ArrowRight" ? 1 : -1;
    const next = list[(current + dir + list.length) % list.length];
    next?.focus();
    if (next?.dataset.view) setView(next.dataset.view);
  }
  if ((event.key === "Enter" || event.key === " ") && current >= 0) {
    event.preventDefault();
    if (list[current]?.dataset.view) setView(list[current].dataset.view);
  }
});

// 明暗切换
document.querySelector(".theme-toggle")?.addEventListener("click", () => {
  body.dataset.theme = body.dataset.theme === "dark" ? "light" : "dark";
});

// 图层显隐：检查器 checkbox ↔ 图纸 SVG 图层
document.querySelectorAll(".layer-row").forEach((row) => {
  const input = row.querySelector("input");
  const name = row.querySelector("span")?.textContent?.trim() || "";
  input?.addEventListener("change", () => {
    if (name === "A-墙体") document.querySelectorAll(".plan .walls").forEach((g) => (g.style.opacity = input.checked ? "1" : "0.12"));
    else if (name === "A-家具") document.querySelectorAll(".plan .furniture").forEach((g) => (g.style.opacity = input.checked ? "1" : "0.1"));
    else if (name === "DIM-标注") document.querySelectorAll(".plan .dimensions").forEach((g) => (g.style.opacity = input.checked ? "1" : "0.1"));
    else document.querySelectorAll(`.plan .layer[data-layer="${CSS.escape(name)}"]`).forEach((g) => {
      g.style.display = input.checked ? "" : "none";
    });
  });
});

// 输入框自适应高度 + Enter 发送（原型只做视觉反馈）
document.querySelectorAll("textarea").forEach((ta) => {
  ta.addEventListener("input", () => {
    ta.style.height = "auto";
    ta.style.height = Math.min(ta.scrollHeight, 140) + "px";
  });
  ta.addEventListener("keydown", (event) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      if (ta.value.trim()) {
        ta.value = "";
        ta.style.height = "";
        ta.blur();
      }
    }
  });
});

// 画布提问：Enter 之后轻反馈
document.querySelectorAll(".canvas-hud input, .mini-ask input").forEach((input) => {
  input.addEventListener("keydown", (event) => {
    if (event.key !== "Enter") return;
    event.preventDefault();
    if (!input.value.trim()) return;
    input.value = "";
    input.placeholder = "已发给 gaea，回答会出现在对话";
  });
});

// 检查器内部 tab（原型只做选中态）
document.querySelectorAll(".insp-tabs").forEach((bar) => {
  bar.addEventListener("click", (event) => {
    const btn = event.target.closest("button");
    if (!btn) return;
    bar.querySelectorAll("button").forEach((b) => b.classList.toggle("is-active", b === btn));
  });
});

// 点击空白关闭工具菜单
document.addEventListener("mousedown", (event) => {
  if (!toolMenu || toolMenu.hidden) return;
  if (event.target.closest("#tool-menu") || event.target.closest('[data-action="tools"]')) return;
  closeToolMenu();
});

// 初始状态
body.dataset.inspector = "open";
const params = new URLSearchParams(location.search);
if (params.get("theme") === "light") body.dataset.theme = "light";
const initial = params.get("view");
setView(["chat", "trajectory", "context", "memory", "desk"].includes(initial) ? initial : "chat");
const initialTool = params.get("tool");
if (initialTool === "cad" || initialTool === "schedule") openTool(initialTool);
document.querySelectorAll(".tab[data-view]").forEach((tab) => {
  tab.addEventListener("click", () => setView(tab.dataset.view));
});
document.addEventListener("keydown", (event) => {
  if (event.key !== "Escape") return;
  if (!toolMenu?.hidden) { closeToolMenu(); return; }
  closeTool();
});
// 工具菜单必须挂在 body 上：它在对话/空态都要能弹出，不能随对话视图隐藏
if (toolMenu) document.body.appendChild(toolMenu);
