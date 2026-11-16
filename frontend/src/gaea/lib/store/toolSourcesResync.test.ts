import { describe, expect, it } from "vitest";
import type { ControllerState } from "./controller_state";
import { applyEvent, initialState, parseResyncItems } from "./controller_state";

const source = {
  title: "Go 1.24",
  url: "https://go.dev/doc/go1.24",
  snippet: "release notes",
  source: "bing",
};

describe("tool source state", () => {
  it("实时 tool_result 将结构化来源挂到工具条目", () => {
    let state: ControllerState = { ...initialState, items: [], _dispatch: () => {} };
    state = applyEvent(state, {
      kind: "tool_dispatch",
      tool: { id: "t1", name: "web_search", args: "{\"query\":\"go\"}", readOnly: true },
    });
    state = applyEvent(state, {
      kind: "tool_result",
      tool: { id: "t1", name: "web_search", output: "[]", readOnly: true, sources: [source] },
    });
    expect(state.items[0]).toMatchObject({
      kind: "tool",
      id: "t1",
      sources: [source],
    });
  });

  it("补拉快照保留结构化来源并丢弃坏项", () => {
    const items = parseResyncItems([{
      kind: "tool",
      id: "t1",
      name: "web_search",
      args: "",
      readOnly: true,
      status: "done",
      sources: [source, { title: "", url: "https://bad" }, { title: "dup", url: source.url }],
    }]);
    expect(items?.[0]).toMatchObject({ kind: "tool", sources: [source] });
  });
});
