import { describe, expect, it } from "vitest";
import { escOwner } from "./escLadder";

describe("escOwner", () => {
  it("modal/overlay win before running stop", () => {
    expect(escOwner({ modalOpen: true, running: true })).toBe("modal");
    expect(escOwner({ overlayOpen: true, running: true })).toBe("overlay");
  });

  it("input keeps Esc local; running outside input stops the turn", () => {
    expect(escOwner({ inInput: true, running: true })).toBe("composer");
    expect(escOwner({ running: true })).toBe("stop");
  });

  it("empty idle composer enters the backtrack entry point", () => {
    expect(escOwner({ composerEmpty: true })).toBe("backtrack");
    expect(escOwner({ composerEmpty: false })).toBe("composer");
  });

  it("warnings/questions/banner are explicit priorities", () => {
    expect(escOwner({ warningsOpen: true, questionsOpen: true })).toBe("warnings");
    expect(escOwner({ questionsOpen: true, overlayOpen: true })).toBe("questions");
    expect(escOwner({ bannerOpen: true, composerEmpty: true })).toBe("banner");
  });
});
