import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const assetDataURL = vi.hoisted(() => vi.fn(async (_ref: string) => "data:image/png;base64,YWJj"));
vi.mock("../gaea/lib/bridge", () => ({
  app: { GaeaProposalAssetDataURL: assetDataURL },
}));

import { MarkdownContent } from "./MarkdownContent";

describe("MarkdownContent proposal assets", () => {
  it("resolves gaea-asset image references through the proposal binding", async () => {
    const ref = `gaea-asset://${"a".repeat(64)}`;
    render(<MarkdownContent source={`![工艺流程图](${ref})`} />);
    const img = await screen.findByRole("img", { name: "工艺流程图" });
    await waitFor(() => expect(img.getAttribute("src")).toBe("data:image/png;base64,YWJj"));
    expect(img.getAttribute("data-managed-image")).toBe("true");
    expect(assetDataURL).toHaveBeenCalledWith(ref);
  });
});
