import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { KnowledgePanel } from "./KnowledgePanel";
import { LocaleProvider } from "../lib/i18n";

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>;

describe("KnowledgePanel 知识库面板", () => {
  it("多选后可批量删除并清空选择", async () => {
    render(wrap(<KnowledgePanel onClose={() => {}} variant="page" />));

    expect(await screen.findByText(/建筑工程施工质量验收统一标准/)).toBeTruthy();
    const boxes = screen.getAllByTitle("多选（批量删除/改状态）");
    fireEvent.click(boxes[0]);
    fireEvent.click(boxes[1]);
    expect(screen.getByText("已选 2")).toBeTruthy();

    fireEvent.click(screen.getByText("批量删除"));
    await waitFor(() => expect(screen.queryByText("已选 2")).toBeNull());
  });

  it("提供导入入口", async () => {
    render(wrap(<KnowledgePanel onClose={() => {}} variant="page" />));
    await screen.findByText(/建筑工程施工质量验收统一标准/);
    expect(screen.getByTitle("导入 md/txt/docx/pdf/xlsx/csv")).toBeTruthy();
  });
});
