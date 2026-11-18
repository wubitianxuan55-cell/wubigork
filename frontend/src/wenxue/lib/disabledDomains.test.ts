import { describe, expect, it } from "vitest";
import { isDisabledWenxueCommand, isDisabledWenxueTool } from "./disabledDomains";
import { normalizeWorkspaceTabId, WORKSPACE_TABS, WORKSPACE_TAB_IDS } from "./workspaceTabs";

describe("Wenxue board isolation", () => {
  it("does not expose knowledge, CAD or schedule workspace tabs", () => {
    // 2026-11-18 起含 gallery（插图右栏 pane，wenxue 自有域；外部域仍禁）。
    expect(WORKSPACE_TAB_IDS).toEqual(["files", "deliverables", "browser", "git", "ui", "gallery"]);
    expect(WORKSPACE_TABS.map((tab) => tab.label)).not.toEqual(
      expect.arrayContaining(["CAD 图纸", "进度计划"]),
    );
    expect(normalizeWorkspaceTabId("cad")).toBeNull();
    expect(normalizeWorkspaceTabId("schedule")).toBeNull();
  });

  it("filters office-domain commands and tools from Wenxue surfaces", () => {
    for (const name of ["knowledge", "knowledge_search", "schedule", "schedule_apply", "cad", "cad_open"]) {
      expect(isDisabledWenxueCommand(name)).toBe(true);
    }
    for (const name of ["knowledge_search", "knowledge_add", "schedule_get", "schedule_apply", "cad_open"]) {
      expect(isDisabledWenxueTool(name)).toBe(true);
    }
    expect(isDisabledWenxueCommand("memory")).toBe(false);
    expect(isDisabledWenxueTool("write_file")).toBe(false);
  });
});
