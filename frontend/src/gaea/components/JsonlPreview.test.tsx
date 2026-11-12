import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { JsonlPreview } from "./JsonlPreview";

describe("JsonlPreview", () => {
  it("renders record cards, keeps invalid lines visible and can switch to raw", () => {
    render(<JsonlPreview text={'{"type":"status","value":"ok"}\n{bad}\n'} />);
    expect(screen.getByText("JSONL · 2 条记录")).toBeTruthy();
    expect(screen.getByText("type: status · value: ok")).toBeTruthy();
    expect(screen.getByText("无效 JSON")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "原文" }));
    expect(screen.getByText(/type.*status/)).toBeTruthy();
  });
});
