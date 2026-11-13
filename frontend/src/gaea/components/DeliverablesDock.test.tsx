import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { DeliverablesDock } from "./DeliverablesDock";

vi.mock("../lib/deliverablesTurn", () => ({
  ensureTurnRegistry: vi.fn(async () => ({
    entries: [
      { path: "docs/季度复盘.docx", tool: "office_save", updatedAt: 300 },
      { path: "docs/经营数据.xlsx", tool: "xlsx_apply", updatedAt: 200 },
    ],
  })),
  invalidateTurnCaches: vi.fn(),
  deliverablePathKey: (path: string) => path.replace(/\\/g, "/").toLowerCase(),
}));

const renderDock = () => {
  localStorage.setItem("gaea-lang", "zh");
  return render(
    <LocaleProvider>
      <DeliverablesDock />
    </LocaleProvider>,
  );
};

describe("DeliverablesDock 灵犀成果停靠条", () => {
  it("默认收起为单行摘要，展开后渲染类型徽标文件卡而非旧路径列表", async () => {
    const view = renderDock();
    const dock = await screen.findByTestId("deliverables-dock");
    const head = within(dock).getByRole("button", { name: /交付文件/ });
    expect(head.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByText("季度复盘.docx")).toBeNull();

    fireEvent.click(head);
    expect(head.getAttribute("aria-expanded")).toBe("true");
    const docCard = screen.getByText("季度复盘.docx").closest(".office-file-card");
    const xlsCard = screen.getByText("经营数据.xlsx").closest(".office-file-card");
    expect(docCard?.getAttribute("data-kind")).toBe("word");
    expect(docCard?.querySelector(".office-file-card__badge")?.textContent).toBe("DOCX");
    expect(xlsCard?.getAttribute("data-kind")).toBe("excel");
    expect(xlsCard?.querySelector(".office-file-card__badge")?.textContent).toBe("XLSX");
    expect(view.container.querySelectorAll(".office-file-card--compact")).toHaveLength(2);
  });
});
