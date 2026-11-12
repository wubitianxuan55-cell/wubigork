import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { UnifiedChangesTree } from "./UnifiedChangesTree";
import { buildUnifiedChangeTree } from "../lib/unifiedChangesTree";

describe("UnifiedChangesTree", () => {
  it("renders directory hierarchy and Git/Agent source badges", () => {
    const nodes = buildUnifiedChangeTree({
      isRepo: true,
      files: [{ path: "docs/a.md", x: "M", y: " " }],
    }, [{ path: "docs/a.md", count: 2 }, { path: "docs/b.md", count: 1 }]);
    const onOpen = vi.fn();
    render(<UnifiedChangesTree nodes={nodes} onOpenFile={onOpen} />);
    expect(screen.getByText("docs")).toBeTruthy();
    expect(screen.getByText("a.md")).toBeTruthy();
    expect(screen.getByText("暂 M")).toBeTruthy();
    expect(screen.getByText("AI 2")).toBeTruthy();
    fireEvent.click(screen.getByText("a.md"));
    expect(onOpen).toHaveBeenCalledWith("docs/a.md");
  });

  it("offers stage, unstage and two-click discard actions without opening the file", () => {
    const nodes = buildUnifiedChangeTree({
      isRepo: true,
      files: [{ path: "docs/a.md", x: "M", y: "M", modified: true }],
    }, []);
    const onOpen = vi.fn();
    const onStage = vi.fn();
    const onUnstage = vi.fn();
    const onDiscard = vi.fn();
    render(
      <UnifiedChangesTree
        nodes={nodes}
        onOpenFile={onOpen}
        onStage={onStage}
        onUnstage={onUnstage}
        onDiscard={onDiscard}
      />,
    );
    fireEvent.click(screen.getByLabelText("取消暂存 docs/a.md"));
    fireEvent.click(screen.getByLabelText("暂存 docs/a.md"));
    fireEvent.click(screen.getByLabelText("丢弃 docs/a.md"));
    expect(onDiscard).not.toHaveBeenCalled();
    fireEvent.click(screen.getByLabelText("确认丢弃 docs/a.md"));
    expect(onUnstage).toHaveBeenCalledWith("docs/a.md");
    expect(onStage).toHaveBeenCalledWith("docs/a.md");
    expect(onDiscard).toHaveBeenCalledWith("docs/a.md");
    expect(onOpen).not.toHaveBeenCalled();
  });

  it("embeds row diff through the supplied renderer", () => {
    const nodes = buildUnifiedChangeTree({
      isRepo: true,
      files: [{ path: "docs/a.md", x: " ", y: "M" }],
    }, []);
    let expanded: string | null = null;
    const { rerender } = render(
      <UnifiedChangesTree
        nodes={nodes}
        expandedDiffPath={expanded}
        onToggleDiff={(file) => { expanded = file.path; }}
        renderDiff={(file) => <div data-testid="inline-diff">{file.path}</div>}
      />,
    );
    fireEvent.click(screen.getByLabelText("查看 docs/a.md 的 diff"));
    rerender(
      <UnifiedChangesTree
        nodes={nodes}
        expandedDiffPath={expanded}
        onToggleDiff={(file) => { expanded = file.path; }}
        renderDiff={(file) => <div data-testid="inline-diff">{file.path}</div>}
      />,
    );
    expect(screen.getByTestId("inline-diff").textContent).toBe("docs/a.md");
  });
});
