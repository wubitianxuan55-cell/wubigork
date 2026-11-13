// WpsDockControl.test.tsx — P1-4 停靠控制行为测试（jsdom，app 全 mock）：
// 单文档窗→直接停靠；多文档窗→先弹点名菜单；还原；卸载兜底还原。
// jsdom 无几何：getBoundingClientRect 恒 0 → mock paneRect 换算只断言「被调」，
// 不断言坐标值（物理像素换算挂真机走查）。
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { WpsDockControl } from "./WpsDockControl";

const wpsMock = vi.hoisted(() => ({
  list: vi.fn(async (..._a: unknown[]) => [] as unknown[]),
  apply: vi.fn(async (..._a: unknown[]) => "已停靠"),
  restore: vi.fn(async (..._a: unknown[]) => "已还原"),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    WpsDockList: wpsMock.list,
    WpsDockApply: wpsMock.apply,
    WpsDockRestore: wpsMock.restore,
  },
}));

const DOC_WIN = {
  hwnd: 1001, proc: "wps.exe", class: "OpusApp",
  title: "施工成本.wps", visible: true, isDocWin: true,
};

function wrap() {
  // closest('.preview-pane') 需要真实祖先链：包一层带类名的容器。
  return (
    <div className="preview-pane">
      <WpsDockControl />
    </div>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  // jsdom 无几何：给面板一个真实矩形，让 paneRect 换算走通（坐标值不断言）。
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue({
    x: 100, y: 50, left: 100, top: 50, right: 500, bottom: 650,
    width: 400, height: 600, toJSON: () => ({}),
  } as DOMRect);
  Object.defineProperty(window, "screenX", { value: 0, configurable: true });
  Object.defineProperty(window, "screenY", { value: 0, configurable: true });
});

afterEach(async () => {
  // 卸载兜底还原的异步尾巴清完，防用例间串扰。
  await new Promise((r) => setTimeout(r, 0));
});

describe("WpsDockControl", () => {
  it("单文档窗口：点停靠直接 apply，出现还原钮", async () => {
    wpsMock.list.mockResolvedValue([DOC_WIN]);
    render(wrap());
    fireEvent.click(screen.getByLabelText("停靠 WPS 窗口"));
    await waitFor(() => expect(wpsMock.apply).toHaveBeenCalledTimes(1));
    expect(wpsMock.apply.mock.calls[0][0]).toBe(1001);
    expect(wpsMock.apply.mock.calls[0][5]).toBe(true); // borderless
    expect(screen.getByLabelText("还原 WPS 窗口")).toBeTruthy();
  });

  it("多文档窗口：先弹菜单再点名停靠", async () => {
    wpsMock.list.mockResolvedValue([
      DOC_WIN,
      { hwnd: 1002, proc: "et.exe", class: "XLMAIN", title: "预算表.xlsx", visible: true, isDocWin: true },
    ]);
    render(wrap());
    fireEvent.click(screen.getByLabelText("停靠 WPS 窗口"));
    const pick = await screen.findByTitle("预算表.xlsx");
    fireEvent.click(pick);
    await waitFor(() => expect(wpsMock.apply).toHaveBeenCalledTimes(1));
    expect(wpsMock.apply.mock.calls[0][0]).toBe(1002);
  });

  it("无文档窗口：报可执行下一步，不调 apply", async () => {
    wpsMock.list.mockResolvedValue([
      { hwnd: 9, proc: "wps.exe", class: "QQmlWindow", title: "", visible: true, isDocWin: false },
    ]);
    render(wrap());
    fireEvent.click(screen.getByLabelText("停靠 WPS 窗口"));
    await waitFor(() => expect(screen.getByText(/没找到 WPS 文档窗口/)).toBeTruthy());
    expect(wpsMock.apply).not.toHaveBeenCalled();
  });

  it("还原钮：调 restore 并清停靠态", async () => {
    wpsMock.list.mockResolvedValue([DOC_WIN]);
    render(wrap());
    fireEvent.click(screen.getByLabelText("停靠 WPS 窗口"));
    await waitFor(() => expect(screen.getByLabelText("还原 WPS 窗口")).toBeTruthy());
    fireEvent.click(screen.getByLabelText("还原 WPS 窗口"));
    await waitFor(() => expect(wpsMock.restore).toHaveBeenCalledWith(1001));
    expect(screen.getByLabelText("停靠 WPS 窗口")).toBeTruthy();
  });

  it("卸载兜底：停靠态下卸载自动 restore", async () => {
    wpsMock.list.mockResolvedValue([DOC_WIN]);
    const { unmount } = render(wrap());
    fireEvent.click(screen.getByLabelText("停靠 WPS 窗口"));
    await waitFor(() => expect(wpsMock.apply).toHaveBeenCalled());
    wpsMock.restore.mockClear();
    unmount();
    await waitFor(() => expect(wpsMock.restore).toHaveBeenCalledWith(1001));
  });
});
