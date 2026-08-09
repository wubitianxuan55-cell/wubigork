import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DeliverableCards } from "./DeliverableCards";
import { usePreviewStore, useUpdatedFilesStore } from "../lib/store";

describe("DeliverableCards 交付文件卡片", () => {
  it("把正文中的办公文件渲染为可点击预览卡片并去重", () => {
    usePreviewStore.setState({ previewFile: null });
    render(
      <DeliverableCards
        text="已生成：exports/成本测算.xlsx，方案见 .gaea/exports/方案.docx，图表 exports/趋势.png。再次提到 exports/成本测算.xlsx。"
      />,
    );
    const cards = screen.getAllByRole("button", { name: /成本测算\.xlsx|方案\.docx|趋势\.png/ });
    expect(cards).toHaveLength(3);

    fireEvent.click(screen.getByRole("button", { name: /成本测算\.xlsx/ }));
    expect(usePreviewStore.getState().previewFile).toBe("exports/成本测算.xlsx");
  });

  it("代码文件不算交付物，纯文本无文件时不渲染", () => {
    const { container } = render(<DeliverableCards text="已生成 exports/main.ts，无其他交付" />);
    expect(container.querySelectorAll("button")).toHaveLength(0);
  });

  it("预览内编辑过的文件显示「已更新」徽标", () => {
    useUpdatedFilesStore.setState({ updatedAt: { "exports/成本测算.xlsx": Date.now() } });
    render(<DeliverableCards text="已生成：exports/成本测算.xlsx" />);
    expect(screen.getByText("已更新")).toBeTruthy();
    useUpdatedFilesStore.setState({ updatedAt: {} });
  });
});
