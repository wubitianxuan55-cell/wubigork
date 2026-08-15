// T7-4（v2.37.0）Toast：role 属性（error 档）+ error 档样式。
import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ToastProvider, useToast } from "./Toast";

function Harness() {
  const toast = useToast();
  return (
    <div>
      <button onClick={() => toast.show("普通提示")} type="button">info</button>
      <button onClick={() => toast.show("注意", "warn")} type="button">warn</button>
      <button onClick={() => toast.show("出错了", "error")} type="button">error</button>
    </div>
  );
}

describe("Toast（T7-4）", () => {
  it("info 档：短暂提示不设 role（避免与页面内状态框双匹配），普通样式", () => {
    render(<ToastProvider><Harness /></ToastProvider>);
    fireEvent.click(screen.getByText("info"));
    const toast = screen.getByText("普通提示");
    expect(toast.getAttribute("role")).toBeNull();
    expect(toast.className).toContain("border-l-info");
  });

  it("warn 档：短暂告警不设 role，warning 样式区分", () => {
    render(<ToastProvider><Harness /></ToastProvider>);
    fireEvent.click(screen.getByText("warn"));
    const toast = screen.getByText("注意");
    expect(toast.getAttribute("role")).toBeNull();
    expect(toast.className).toContain("border-l-warning");
  });

  it("error 档：role=alert（失败需读屏干预）+ 错误红色样式", () => {
    render(<ToastProvider><Harness /></ToastProvider>);
    fireEvent.click(screen.getByText("error"));
    const toast = screen.getByText("出错了");
    expect(toast.getAttribute("role")).toBe("alert");
    expect(toast.className).toContain("border-l-red-500");
    expect(toast.className).toContain("text-red-400");
    expect(toast.className).not.toContain("border-l-info");
  });
});
