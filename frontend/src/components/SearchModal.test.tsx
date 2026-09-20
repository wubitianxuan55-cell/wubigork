// SearchModal.test.tsx — v4.7 S4.6 命令面板接统一意图路由（指令预览卡 + 预览-确认制）
// UI 契约：dry-run 命中出「指令」卡、点「执行」才真跑（回执内联）、未命中检索行为零变化。
// 后端语义（宁漏勿误 / dry-run 零副作用 / 执行层一致性）由 Go TestGaeaRouteIntent* 覆盖；
// 这里走真实 mock 层（mock/retrieval.ts 的 RouteIntent 演示规则与 internal/intent 同向）。
import { describe, expect, it, beforeEach, afterEach, beforeAll, vi } from "vitest";
import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import SearchModal from "./SearchModal";
import { LocaleProvider } from "../gaea/lib/i18n";
import { waitMockReady } from "../gaea/lib/bridge";

// 7.3-1 任务收件箱：收口前真实 mock 层没有 GaeaTaskInbox* 方法——用
// importOriginal 包装真实模块，仅给 app 补 Save 桩，其余导出（waitMockReady、
// RouteIntent 演示规则等）逐属性透传，既有用例零回归。
const taskInboxMocks = vi.hoisted(() => ({
  save: vi.fn(),
}));

// v4.361 失败可见化：可注入的 UnifiedSearch 覆盖（null=走真实 mock 层）。
const searchOverride = vi.hoisted(() => ({ unified: null as ((q: string, n: number, scope: string) => Promise<unknown>) | null }));

vi.mock("../gaea/lib/bridge", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../gaea/lib/bridge")>();
  const wrappedApp: typeof actual.app = new Proxy(actual.app, {
    get(target, prop) {
      if (prop === "GaeaTaskInboxSave") return taskInboxMocks.save;
      if (prop === "UnifiedSearch" && searchOverride.unified) return searchOverride.unified;
      return Reflect.get(target, prop);
    },
  });
  return { ...actual, app: wrappedApp };
});

// H6（P4 entry 懒加载）：mock 为异步 chunk，RouteIntent/UnifiedSearch 演示规则在
// chunk 加载后才可用——先等 mock 就绪再渲染（否则断言打到空结果）。
beforeAll(async () => {
  await waitMockReady();
});

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>;

// 固定 zh（断言 i18n 文案；preview reply 来自 mock 后端，与 UI locale 无关）
beforeEach(() => {
  localStorage.setItem("gaea-lang", "zh");
});

afterEach(() => {
  vi.useRealTimers();
});

function renderModal(onClose = vi.fn()) {
  render(wrap(<SearchModal open onClose={onClose} space="work" />));
  return onClose;
}

async function search(query: string) {
  // antd Modal 在 jsdom 下的 role 计算不稳定（textbox 查不到）——按类名直查输入框
  const input = document.querySelector("input.ant-input") as HTMLInputElement;
  expect(input).toBeTruthy();
  fireEvent.change(input, { target: { value: query } });
  const btn = document.querySelector(".ant-input-search-button") as HTMLButtonElement;
  fireEvent.click(btn);
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

// 指令卡内的「执行」按钮（antd 按钮的 role name 计算在 jsdom 同样不稳定，按 DOM 直查）
function execButton(card: Element): HTMLButtonElement {
  return card.querySelector("button") as HTMLButtonElement;
}

describe("SearchModal 命令面板接统一意图路由（S4.6）", () => {
  it("导航指令命中：出指令预览卡（动作标签 + 预览语 + 执行按钮），零结果时不再叠 Empty", async () => {
    renderModal();
    await search("打开绘梦");

    const card = screen.getByTestId("intent-card");
    expect(card).toBeTruthy();
    expect(card.textContent).toContain("打开板块"); // 动作标签（i18n）
    expect(card.textContent).toContain("将打开「绘梦」板块"); // dry-run 预览语
    expect(execButton(card)).toBeTruthy();
    // 有指令卡时不再叠放「无结果」Empty（面板不被误导）
    expect(document.querySelector(".ant-empty")).toBeNull();
  });

  it("预览-确认制：dry-run 只出卡不执行；点「执行」才真跑并内联回执（非导航不收面板）", async () => {
    const onClose = renderModal();
    await search("现在用什么模型");

    const card = screen.getByTestId("intent-card");
    expect(card.textContent).toContain("将查询当前模型引擎状态"); // 预览语
    expect(card.textContent).not.toContain("Herdsman"); // 尚未执行——没有真实回执
    expect(onClose).not.toHaveBeenCalled();

    fireEvent.click(execButton(card));
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(card.textContent).toContain("→ 当前状态：Herdsman（本地）。"); // 执行回执内联
    expect(onClose).not.toHaveBeenCalled(); // 非导航类保持面板打开供查看回执
  });

  it("导航指令执行：回执后自动收面板（后端 emit 导航事件，壳层切板块）", async () => {
    vi.useFakeTimers();
    const onClose = renderModal();
    await search("打开绘梦");

    const card = screen.getByTestId("intent-card");
    fireEvent.click(execButton(card));
    // 回执先内联（微任务刷新），600ms 后 onClose 收面板
    await act(async () => {
      await Promise.resolve();
    });
    expect(onClose).not.toHaveBeenCalled();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(700);
    });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("未命中：不出指令卡，检索行为零变化（结果区正常渲染）", async () => {
    renderModal();
    await search("今天天气怎么样");

    expect(screen.queryByTestId("intent-card")).toBeNull();
    // 检索照常执行：分类标签/结果区渲染（mock 演示数据对任意查询有命中）
    expect(document.querySelector(".ant-tag")).toBeTruthy();
  });

  it("闲聊口语不误出卡（宁漏勿误：搜索词不是整句指令入口）", async () => {
    renderModal();
    // 「画得不错」是口语夸赞（生图正则排除项）；「提醒我一下」无时间也会出卡
    // 但仅是预览——这里验证的是「夸赞」绝不触发生图预览卡。
    await search("这张图画得不错");
    expect(screen.queryByTestId("intent-card")).toBeNull();
  });
});

// 7.3-1 任务收件箱：指令预览卡〔存为任务〕次按钮——命中指令才出现（搜索词≠
// 指令宁漏勿误，手动新建走收件箱面板）；点击直接 GaeaTaskInboxSave 落库
// （source='ctrlk'、space=当前壳层空间、action/target 随指令），回执内联。
describe("SearchModal 存为任务（7.3-1）", () => {
  beforeEach(() => {
    taskInboxMocks.save.mockClear();
    taskInboxMocks.save.mockResolvedValue({
      id: "ti-save00000001",
      title: "打开绘梦",
      space: "work",
      status: "pending",
      source: "ctrlk",
      createdAt: 0,
      updatedAt: 0,
    });
  });

  it("指令预览卡出〔存为任务〕次按钮；「执行」仍为卡内首位按钮（既有锚定不变）", async () => {
    renderModal();
    await search("打开绘梦");

    const card = screen.getByTestId("intent-card");
    const saveBtn = card.querySelector<HTMLButtonElement>('[data-testid="intent-save-task"]');
    expect(saveBtn).toBeTruthy();
    expect(saveBtn!.textContent).toContain("存为任务");
    // 卡内第一个 button 仍是「执行」（既有 execButton() 语义零回归；antd 中文
    // 双字按钮自动插入字间空格，比较前去空白）
    expect(card.querySelector("button")!.textContent!.replace(/\s+/g, "")).toContain("执行");
  });

  it("点〔存为任务〕：调 Save 落库（title/space/source/action/target）且回执「已存入任务收件箱」", async () => {
    renderModal();
    await search("打开绘梦");

    const card = screen.getByTestId("intent-card");
    fireEvent.click(card.querySelector<HTMLButtonElement>('[data-testid="intent-save-task"]')!);
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(taskInboxMocks.save).toHaveBeenCalledTimes(1);
    expect(JSON.parse(taskInboxMocks.save.mock.calls[0][0] as string)).toEqual({
      title: "打开绘梦",
      space: "work",
      source: "ctrlk",
      action: "navigate",
      target: "imagegen",
    });
    // 回执内联（intentReply 位）
    expect(card.textContent).toContain("已存入任务收件箱");
  });

  it("未命中指令（无预览卡）时不出现〔存为任务〕按钮", async () => {
    renderModal();
    await search("这张图画得不错");

    expect(screen.queryByTestId("intent-card")).toBeNull();
    expect(document.querySelector('[data-testid="intent-save-task"]')).toBeNull();
    expect(taskInboxMocks.save).not.toHaveBeenCalled();
  });
});

describe("检索失败可见化（v4.361）", () => {
  beforeEach(() => {
    localStorage.setItem("gaea-lang", "zh");
    searchOverride.unified = null;
  });

  it("统一检索失败显示「检索失败，请重试」而非伪装「未找到」", async () => {
    searchOverride.unified = vi.fn().mockRejectedValue(new Error("backend down"));
    renderModal();
    await search("预算");
    expect(await screen.findByTestId("search-failed-empty")).toBeTruthy();
    expect(screen.getByTestId("search-failed-empty").textContent).toContain("检索失败");
    expect(screen.queryByText(/未找到/)).toBeNull();
  });

  it("检索正常时不误报失败", async () => {
    renderModal();
    await search("预算");
    await waitFor(() => expect(screen.queryByTestId("search-failed-empty")).toBeNull(), { timeout: 3000 });
  });
});
