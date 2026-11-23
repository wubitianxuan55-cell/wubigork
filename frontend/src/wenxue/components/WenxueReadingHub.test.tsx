import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WenxueReadingHub } from "./WenxueReadingHub";

// 数据面 mock：定稿目录两章（序号排序 + 0 号无序章），验目录导航/阅读窗/翻章。
const finals: Record<string, string> = {
  "定稿/001-雨夜.md": "# 第一章 雨夜\n\n雨夜。巷口的路灯忽明忽暗。",
  "定稿/002-追查.md": "# 第二章 追查\n\n清晨，废料站的证物。",
  "定稿/序章.md": "# 序章\n\n一切开始之前。",
};

vi.mock("../lib/bridge", () => ({
  app: {
    ListDir: vi.fn(async (rel: string) => {
      if (rel === "") return [{ name: "定稿", isDir: true, size: 0 }];
      return Object.keys(finals).map((p) => ({ name: p.split("/")[1], isDir: false, size: finals[p].length }));
    }),
    GaeaReadTextFile: vi.fn(async (rel: string) => ({
      path: rel, text: finals[rel], size: finals[rel].length,
      mtimeMs: 1000, encoding: "utf-8", eol: "LF",
    })),
  },
}));

describe("WenxueReadingHub", () => {
  // 排版偏好走 localStorage，先清防用例间串味。
  beforeEach(() => { localStorage.clear(); });

  it("依据定稿目录生成导航（序章在前，章号升序）并渲染首章阅读窗", async () => {
    render(<WenxueReadingHub />);
    await waitFor(() => expect(screen.getByTestId("wenxue-reading-item-1")).toBeTruthy());
    // 导航序：序章（无编号=0 在前）→ 1 → 2
    const container = document.querySelector('[data-testid="wenxue-reading-hub"]')!;
    const order = Array.from(container.querySelectorAll("[data-testid^='wenxue-reading-item-']"))
      .map((n) => n.getAttribute("data-testid"));
    expect(order).toEqual(["wenxue-reading-item-序章.md", "wenxue-reading-item-1", "wenxue-reading-item-2"]);
    // 首章（序章）自动选中的阅读窗（Markdown 异步渲染）
    await waitFor(() => expect(screen.getByText("一切开始之前。")).toBeTruthy());
    expect(screen.getByText(/定稿目录/)).toBeTruthy();
  });

  it("点目录条目切换章节，翻章按钮按序移动", async () => {
    render(<WenxueReadingHub />);
    await waitFor(() => expect(screen.getByTestId("wenxue-reading-item-2")).toBeTruthy());
    fireEvent.click(screen.getByTestId("wenxue-reading-item-2"));
    await waitFor(() => expect(screen.getByText("清晨，废料站的证物。")).toBeTruthy());
    // 第二章「下一章」禁用（末章）
    const nextBtn = screen.getByText("下一章").closest("button") as HTMLButtonElement;
    expect(nextBtn.disabled).toBe(true);
    fireEvent.click(screen.getByText("上一章").closest("button") as HTMLButtonElement);
    await waitFor(() => expect(screen.getByText("雨夜。巷口的路灯忽明忽暗。")).toBeTruthy());
  });

  it("字号滑动调节与字体切换生效（排版偏好）", async () => {
    render(<WenxueReadingHub />);
    await waitFor(() => screen.getByTestId("wenxue-reading-item-1"));
    // 字号滑动：14–24 步进 1，数值即时展示。
    const slider = screen.getByLabelText("正文字号") as HTMLInputElement;
    expect(slider.min).toBe("14");
    expect(slider.max).toBe("24");
    fireEvent.change(slider, { target: { value: "19" } });
    expect(screen.getByTitle("当前 19px")).toBeTruthy();
    const prose = document.querySelector<HTMLElement>(".wx-reading-prose")!;
    expect(prose.style.getPropertyValue("--wx-reading-size")).toBe("19px");
    // 字体切换：选中楷体后 prose 注入对应字体栈。
    fireEvent.change(screen.getByLabelText("正文字体"), { target: { value: "kai" } });
    expect(prose.style.getPropertyValue("--wx-reading-font")).toContain("KaiTi");
    // 偏好持久化进 localStorage（跨重启保留）。
    const saved = JSON.parse(localStorage.getItem("gaea.wenxue.reading.prefs") ?? "{}");
    expect(saved.fontSize).toBe(19);
    expect(saved.fontKey).toBe("kai");
  });

  it("沉浸专注模式开关", async () => {
    render(<WenxueReadingHub />);
    await waitFor(() => screen.getByTestId("wenxue-reading-item-1"));
    fireEvent.click(screen.getByRole("button", { name: /专注阅读/ }));
    expect(screen.getByRole("button", { name: /退出专注/ })).toBeTruthy();
    expect(document.querySelector('[data-testid="wenxue-reading-hub"]')?.className).toContain("is-focus");
  });

  it("书页 Ctrl+滚轮缩放字号，普通滚轮不劫持", async () => {
    render(<WenxueReadingHub />);
    await waitFor(() => expect(screen.getByText("一切开始之前。")).toBeTruthy());
    const scroll = document.querySelector(".wx-reading__scroll")!;
    // Ctrl+上滚放大：17 → 18，与滑杆同一偏好源。
    fireEvent.wheel(scroll, { deltaY: -120, ctrlKey: true });
    expect(screen.getByTitle("当前 18px")).toBeTruthy();
    expect(screen.getByTitle("正文字号 18px（Ctrl+滚轮也可缩放）")).toBeTruthy();
    // 普通滚轮（无 Ctrl）保持原生滚动语义，不改变字号。
    fireEvent.wheel(scroll, { deltaY: 120 });
    expect(screen.getByTitle("当前 18px")).toBeTruthy();
  });
});
