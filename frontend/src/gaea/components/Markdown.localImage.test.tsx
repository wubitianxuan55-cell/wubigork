import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const attachment = vi.hoisted(() => vi.fn(async (_path: string) => "data:image/png;base64,YWJj"));
vi.mock("../lib/bridge", () => ({
  app: { AttachmentDataURL: attachment },
  openExternal: vi.fn(),
  onEvent: () => () => {},
}));

import { Markdown } from "./Markdown";

describe("Markdown local images", () => {
  it("resolves a relative image against the Markdown file and renders the data URL", async () => {
    render(<Markdown text="![示意图](./images/a.png)" imageBasePath="docs/report.md" />);
    const img = await screen.findByRole("img", { name: "示意图" });
    await waitFor(() => expect(img.getAttribute("src")).toBe("data:image/png;base64,YWJj"));
    expect(attachment).toHaveBeenCalledWith("docs/images/a.png");
  });
});
