import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CopyableTable } from "./CopyableTable";

const copyMock = vi.hoisted(() => vi.fn());
vi.mock("../gaea/lib/clipboard", () => ({ copyToClipboard: copyMock }));

describe("CopyableTable", () => {
  beforeEach(() => {
    copyMock.mockReset();
    copyMock.mockResolvedValue("rich");
  });

  it("copies GFM source plus sanitized HTML representation", async () => {
    render(
      <CopyableTable>
        <thead><tr><th align="left">名称</th><th>值</th></tr></thead>
        <tbody><tr><td>a|b</td><td>1</td></tr></tbody>
      </CopyableTable>,
    );
    fireEvent.click(screen.getByRole("button", { name: /复制表格/ }));
    await waitFor(() => expect(copyMock).toHaveBeenCalledTimes(1));
    const [plain, options] = copyMock.mock.calls[0];
    expect(plain).toBe("| 名称 | 值 |\n|:--|---|\n| a\\|b | 1 |");
    expect(options.html).toContain("<table>");
    expect(options.html).toContain("<th");
    expect(await screen.findByText("已复制")).toBeTruthy();
  });

  it("states plain-only fallback honestly", async () => {
    copyMock.mockResolvedValue("plain");
    render(<CopyableTable><tbody><tr><td>1</td></tr></tbody></CopyableTable>);
    fireEvent.click(screen.getByRole("button", { name: /复制表格/ }));
    expect(await screen.findByText("已复制为纯文本")).toBeTruthy();
  });
});
