// PrivacySidebarSection.test.tsx — 隐私区眼睛开关行为锁（2026-11-23 眼睛失效
// 根修回归，bridge 直接 mock——Hub 测试同款 vi.hoisted）：
//   1. 无限制书出眼睛行（普通书不出）；点眼睛 → SetBookHidden(true) +
//      wenxue:workspace-changed 广播（侧栏 projectGroups 重拉信号）；
//   2. 再点 → SetBookHidden(false) 再广播（还原向）；
//   3. 锁定态（PIN 已设未解锁）：书名置空占位行 + PIN 输入面。
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { PrivacySidebarSection } from "./PrivacySidebarSection";

const mocks = vi.hoisted(() => ({
  WenxuePrivacyOverview: vi.fn(),
  WenxuePrivacySetBookHidden: vi.fn(),
  WenxuePrivacyUnlock: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({ app: mocks }));

function wrap(ui: React.ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

const BOOK_OK = { name: "普通书", path: "C:/AI/tuwen/普通书", adultLevel: "全年龄", hidden: false, current: true };
const BOOK_R18 = { name: "无底线书", path: "C:/AI/tuwen/无底线书", adultLevel: "深度成人（无限制）", hidden: false, current: false };

beforeEach(() => {
  vi.clearAllMocks();
  mocks.WenxuePrivacySetBookHidden.mockImplementation(async () => {
    window.dispatchEvent(new Event("wenxue:workspace-changed"));
  });
});

describe("隐私区单按钮 · 常驻面零书名（2026-11-24 收敛）", () => {
  it("常驻面只有按钮无书名；点开面板才见眼睛行；点击隐藏→广播；再点还原→再广播", async () => {
    // 可变镜像：SetBookHidden 后 Overview 跟随（模拟后端真实状态翻转）。
    const r18 = { ...BOOK_R18 };
    mocks.WenxuePrivacyOverview.mockImplementation(async () => ({
      pinSet: false, unlocked: true, maskOnBlur: true, hiddenCount: r18.hidden ? 1 : 0,
      books: [BOOK_OK, { ...r18 }],
    }));
    mocks.WenxuePrivacySetBookHidden.mockImplementation(async (_name: string, h: boolean) => {
      r18.hidden = h;
      window.dispatchEvent(new Event("wenxue:workspace-changed"));
    });

    render(wrap(<PrivacySidebarSection />));

    // 常驻面：按钮在、书名不出现（白隐藏根修）。
    await waitFor(() => expect(screen.getByTestId("wenxue-privacy-entry")).toBeTruthy());
    expect(screen.queryByText("无底线书")).toBeNull();
    expect(screen.queryByTestId("wenxue-privacy-books")).toBeNull();

    // 点开面板：眼睛行只有无限制书。
    fireEvent.click(screen.getByTestId("wenxue-privacy-entry"));
    await waitFor(() => expect(screen.getByTestId("wenxue-privacy-books")).toBeTruthy());
    expect(screen.getByText("无底线书")).toBeTruthy();
    expect(screen.queryByText("普通书")).toBeNull();

    // 点眼睛隐藏 → SetBookHidden(true) + 事件广播（侧栏重拉信号）。
    const events: string[] = [];
    const onEvt = () => events.push("changed");
    window.addEventListener("wenxue:workspace-changed", onEvt);
    fireEvent.click(screen.getByTestId("wenxue-privacy-book-row"));
    await waitFor(() => expect(mocks.WenxuePrivacySetBookHidden).toHaveBeenCalledWith("无底线书", true));
    expect(events.length).toBeGreaterThanOrEqual(1);

    // Overview 翻转后（hook refresh 生效）再点还原。
    fireEvent.click(screen.getByTestId("wenxue-privacy-book-row"));
    await waitFor(() => expect(mocks.WenxuePrivacySetBookHidden).toHaveBeenCalledWith("无底线书", false));
    window.removeEventListener("wenxue:workspace-changed", onEvt);
  });

  it("锁定态：点开面板只有 PIN 输入与占位计数，无书名", async () => {
    mocks.WenxuePrivacyOverview.mockResolvedValue({
      pinSet: true, unlocked: false, maskOnBlur: true, hiddenCount: 1,
      books: [{ name: "", path: "", adultLevel: "深度成人（无限制）", hidden: true, current: false }],
    });

    render(wrap(<PrivacySidebarSection />));
    // 常驻面无书名；等 Overview 拉到后按钮才渲染（渲染守卫）。
    await waitFor(() => expect(screen.getByTestId("wenxue-privacy-entry")).toBeTruthy());
    expect(screen.queryByText("无底线书")).toBeNull();
    fireEvent.click(screen.getByTestId("wenxue-privacy-entry"));
    await waitFor(() => expect(screen.getByTestId("wenxue-privacy-locked-row")).toBeTruthy());
    expect(screen.queryByTestId("wenxue-privacy-books")).toBeNull();
    expect(screen.getByTestId("wenxue-privacy-sidebar-pin")).toBeTruthy();
    expect(screen.getByTestId("wenxue-privacy-sidebar-unlock")).toBeTruthy();
  });
});
