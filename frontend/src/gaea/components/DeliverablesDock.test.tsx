import { describe, expect, it } from "vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { DeliverablesDock } from "./DeliverablesDock";

const ITEMS = [
  { path: "方案.md" },
  { path: "季度总结.md" },
  { path: "docs/附件.pdf" },
  { path: "docs/汇报.pptx" },
  { path: "docs/经营数据.xlsx" },
  { path: "docs/季度复盘.docx" },
];

const renderDock = () => {
  localStorage.setItem("gaea-lang", "zh");
  return render(
    <LocaleProvider>
      <DeliverablesDock items={ITEMS} />
    </LocaleProvider>,
  );
};

describe("DeliverablesDock 灵犀成果停靠条", () => {
  it("按既定交互默认收起，摘要计数与右侧会话产物同源；展开后显示全部卡片", async () => {
    const view = renderDock();
    const dock = await screen.findByTestId("deliverables-dock");
    const head = within(dock).getByRole("button", { name: /交付文件/ });
    expect(head.getAttribute("aria-expanded")).toBe("false");
    expect(dock.querySelector(".office-delivery-dock__count")?.textContent).toBe("6");
    expect(screen.queryByText("季度复盘.docx")).toBeNull();

    fireEvent.click(head);
    expect(head.getAttribute("aria-expanded")).toBe("true");
    const docCard = screen.getByText("季度复盘.docx").closest(".office-file-card");
    const xlsCard = screen.getByText("经营数据.xlsx").closest(".office-file-card");
    expect(docCard?.getAttribute("data-kind")).toBe("word");
    expect(docCard?.querySelector(".office-file-card__badge")?.textContent).toBe("DOCX");
    expect(xlsCard?.getAttribute("data-kind")).toBe("excel");
    expect(xlsCard?.querySelector(".office-file-card__badge")?.textContent).toBe("XLSX");
    expect(view.container.querySelectorAll(".office-file-card--compact")).toHaveLength(6);
  });
});
