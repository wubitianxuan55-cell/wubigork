import { createRef } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MarkdownToc } from "./MarkdownToc";
import type { MarkdownHeading } from "../lib/markdownOutline";

describe("MarkdownToc", () => {
  it("assigns heading ids, collapses and jumps to a section", () => {
    const root = document.createElement("div");
    root.innerHTML = "<div class='md'><h1>一</h1><h2>二</h2></div>";
    document.body.appendChild(root);
    const scrollIntoView = vi.fn();
    root.querySelectorAll<HTMLElement>("h1,h2").forEach((node) => {
      node.scrollIntoView = scrollIntoView;
    });
    const ref = createRef<HTMLElement>();
    Object.defineProperty(ref, "current", { value: root });
    const headings: MarkdownHeading[] = [
      { id: "md-heading-1", level: 1, line: 1, text: "一" },
      { id: "md-heading-2", level: 2, line: 2, text: "二" },
    ];
    render(<MarkdownToc headings={headings} scrollRef={ref} contentKey="doc" />);
    expect(root.querySelector("h2")?.id).toBe("md-heading-2");
    fireEvent.click(screen.getByRole("button", { name: "二" }));
    expect(scrollIntoView).toHaveBeenCalledWith({ block: "start" });
    const toggle = screen.getByRole("button", { name: /目录 · 2/ });
    fireEvent.click(toggle);
    expect(screen.queryByRole("navigation", { name: "Markdown 目录" })).toBeNull();
    root.remove();
  });
});
