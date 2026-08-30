// SearchModal.test.tsx — v4.7 S4.6 命令面板接统一意图路由（指令预览卡 + 预览-确认制）
// UI 契约：dry-run 命中出「指令」卡、点「执行」才真跑（回执内联）、未命中检索行为零变化。
// 后端语义（宁漏勿误 / dry-run 零副作用 / 执行层一致性）由 Go TestGaeaRouteIntent* 覆盖；
// 这里走真实 mock 层（mock/retrieval.ts 的 RouteIntent 演示规则与 internal/intent 同向）。
import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import SearchModal from "./SearchModal";
import { LocaleProvider } from "../gaea/lib/i18n";

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
