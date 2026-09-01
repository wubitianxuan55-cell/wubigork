import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { render, fireEvent, screen } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { ExportMenu } from "./ExportMenu";

// v4.29「化繁为简」：顶栏「导出 / Word / PDF」三常驻文字钮收进单钮下拉。
// 功能红线：三个出口与管线原样保留，本测试锁「收拢后仍三出口可达」。

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

describe("ExportMenu 导出收拢菜单（v4.29 化繁为简）", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    // jsdom navigator.language 默认 en-US → 强制中文断言与生产文案一致
    localStorage.setItem("gaea-lang", "zh");
  });

  it("触发钮显示「导出」且 aria-haspopup=menu", () => {
    render(wrap(<ExportMenu onPick={() => {}} />));
    const trigger = screen.getByTestId("export-menu-trigger");
    expect(trigger.textContent).toContain("导出");
    expect(trigger.getAttribute("aria-haspopup")).toBe("menu");
    expect(trigger.getAttribute("aria-expanded")).toBe("false");
  });

  it("点击展开三个出口（Markdown/Word/PDF），点选后回调并收起", () => {
    const onPick = vi.fn();
    render(wrap(<ExportMenu onPick={onPick} />));
    fireEvent.click(screen.getByTestId("export-menu-trigger"));
    const menu = screen.getByTestId("export-menu");
    expect(menu.getAttribute("role")).toBe("menu");
    const items = screen.getAllByRole("menuitem");
    expect(items).toHaveLength(3);
    expect(items[0].textContent).toContain("Markdown");
    expect(items[1].textContent).toContain("Word");
    expect(items[2].textContent).toContain("PDF");
    fireEvent.click(items[1]);
    expect(onPick).toHaveBeenCalledWith("docx");
    expect(screen.queryByTestId("export-menu")).toBeNull(); // 选后收起
  });

  it("三个出口逐个回调（md/docx/pdf 全可达，功能零删除）", () => {
    const onPick = vi.fn();
    render(wrap(<ExportMenu onPick={onPick} />));
    for (const [label, format] of [["Markdown", "md"], ["Word", "docx"], ["PDF", "pdf"]] as const) {
      fireEvent.click(screen.getByTestId("export-menu-trigger"));
      fireEvent.click(screen.getAllByRole("menuitem").find((el) => el.textContent?.includes(label)) as HTMLElement);
      expect(onPick).toHaveBeenLastCalledWith(format);
    }
    expect(onPick).toHaveBeenCalledTimes(3);
  });

  it("再次点击触发钮收起；点击遮罩收起；Esc 收起", () => {
    render(wrap(<ExportMenu onPick={() => {}} />));
    fireEvent.click(screen.getByTestId("export-menu-trigger"));
    expect(screen.getByTestId("export-menu")).toBeTruthy();
    fireEvent.click(screen.getByTestId("export-menu-trigger"));
    expect(screen.queryByTestId("export-menu")).toBeNull();
    fireEvent.click(screen.getByTestId("export-menu-trigger"));
    fireEvent.click(document.querySelector(".fixed.inset-0") as HTMLElement);
    expect(screen.queryByTestId("export-menu")).toBeNull();
    fireEvent.click(screen.getByTestId("export-menu-trigger"));
    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByTestId("export-menu")).toBeNull();
  });

  it("会话无内容时整钮禁用（与原导出钮 disabled 语义一致）", () => {
    render(wrap(<ExportMenu disabled onPick={() => {}} />));
    const trigger = screen.getByTestId("export-menu-trigger") as HTMLButtonElement;
    expect(trigger.disabled).toBe(true);
    fireEvent.click(trigger);
    expect(screen.queryByTestId("export-menu")).toBeNull();
  });
});
