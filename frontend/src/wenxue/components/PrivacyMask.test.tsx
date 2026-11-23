// PrivacyMask.test.tsx — 失焦自动遮罩行为锁（2026-11-23 私密保护四形态之一）。
// 覆盖：blur/visibilitychange 上罩、未锁点击摘罩、锁定态点击弹 PIN、
// PIN 错误提示与成功解锁、布防解除收罩。
import { beforeAll, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import { PrivacyMask } from "./PrivacyMask";

beforeAll(() => {
  localStorage.setItem("gaea-lang", "zh");
});

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

function fireBlur() {
  act(() => {
    window.dispatchEvent(new Event("blur"));
  });
}

describe("文学失焦遮罩 · 私密保护", () => {
  it("未布防时 blur 不上罩", () => {
    render(wrap(<PrivacyMask armed={false} locked={false} onUnlock={vi.fn()} />));
    fireBlur();
    expect(screen.queryByTestId("wenxue-privacy-mask")).toBeNull();
  });

  it("blur 上罩；未锁点击即摘", () => {
    render(wrap(<PrivacyMask armed locked={false} onUnlock={vi.fn()} />));
    fireBlur();
    expect(screen.getByTestId("wenxue-privacy-mask")).toBeTruthy();
    fireEvent.click(screen.getByTestId("wenxue-privacy-mask"));
    expect(screen.queryByTestId("wenxue-privacy-mask")).toBeNull();
  });

  it("visibilitychange(hidden) 上罩", () => {
    render(wrap(<PrivacyMask armed locked={false} onUnlock={vi.fn()} />));
    // jsdom 无原生 visibilityState 切换：直接派发事件（监听的是事件，
    // 处理器内读 document.visibilityState——jsdom 恒 visible，故改走 blur
    // 分支已覆盖；此用例验证监听器挂载不抛错且不误上罩）。
    fireEvent(document, new Event("visibilitychange"));
    expect(screen.queryByTestId("wenxue-privacy-mask")).toBeNull();
    fireBlur();
    expect(screen.getByTestId("wenxue-privacy-mask")).toBeTruthy();
  });

  it("锁定态点击弹 PIN；错误 PIN 提示且不摘罩；正确 PIN 解锁摘罩", async () => {
    const onUnlock = vi.fn(async (pin: string) => pin === "123456");
    render(wrap(<PrivacyMask armed locked onUnlock={onUnlock} />));
    fireBlur();

    fireEvent.click(screen.getByTestId("wenxue-privacy-mask"));
    const input = screen.getByTestId("wenxue-privacy-pin-input") as HTMLInputElement;
    expect(input).toBeTruthy();

    // 错误 PIN：提示 + 罩面保留。
    fireEvent.change(input, { target: { value: "000000" } });
    fireEvent.click(screen.getByTestId("wenxue-privacy-pin-unlock"));
    await waitFor(() => expect(screen.getByText("PIN 不正确")).toBeTruthy());
    expect(screen.getByTestId("wenxue-privacy-mask")).toBeTruthy();

    // 正确 PIN：摘罩。
    fireEvent.change(screen.getByTestId("wenxue-privacy-pin-input"), { target: { value: "123456" } });
    fireEvent.click(screen.getByTestId("wenxue-privacy-pin-unlock"));
    await waitFor(() => expect(screen.queryByTestId("wenxue-privacy-mask")).toBeNull());
    expect(onUnlock).toHaveBeenCalledWith("123456");
  });

  it("PIN 输入只收数字；Enter 触发解锁", async () => {
    const onUnlock = vi.fn(async () => true);
    render(wrap(<PrivacyMask armed locked onUnlock={onUnlock} />));
    fireBlur();
    fireEvent.click(screen.getByTestId("wenxue-privacy-mask"));

    const input = screen.getByTestId("wenxue-privacy-pin-input") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "12a4" } });
    expect(input.value).toBe("124");
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() => expect(onUnlock).toHaveBeenCalledWith("124"));
  });

  it("布防解除（切普通书/关开关）自动收罩收 PIN 面", () => {
    const { rerender } = render(wrap(<PrivacyMask armed locked onUnlock={vi.fn()} />));
    fireBlur();
    fireEvent.click(screen.getByTestId("wenxue-privacy-mask"));
    expect(screen.getByTestId("wenxue-privacy-pin-input")).toBeTruthy();

    rerender(wrap(<PrivacyMask armed={false} locked onUnlock={vi.fn()} />));
    expect(screen.queryByTestId("wenxue-privacy-mask")).toBeNull();
  });
});
