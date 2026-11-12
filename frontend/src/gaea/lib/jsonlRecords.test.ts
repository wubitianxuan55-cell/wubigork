import { describe, expect, it } from "vitest";
import { JSONL_RECORD_LIMIT, parseJsonlRecords } from "./jsonlRecords";

describe("parseJsonlRecords", () => {
  it("keeps valid records and preserves invalid lines", () => {
    const records = parseJsonlRecords([
      '{"type":"progress","message":"第一步"}',
      "{bad json}",
      '["a","b"]',
      "",
    ].join("\n"));
    expect(records).toHaveLength(3);
    expect(records[0]).toMatchObject({ line: 1, valid: true, summary: "type: progress · message: 第一步" });
    expect(records[1]).toMatchObject({ line: 2, valid: false, summary: "无效 JSON" });
    expect(records[2]).toMatchObject({ line: 3, valid: true, summary: "数组 · 2 项" });
  });

  it("bounds record count and per-record output", () => {
    const text = Array.from({ length: JSONL_RECORD_LIMIT + 20 }, (_, i) =>
      JSON.stringify({ id: i, body: "x".repeat(80) }),
    ).join("\n");
    const records = parseJsonlRecords(text);
    expect(records).toHaveLength(JSONL_RECORD_LIMIT);
    expect(records[0].formatted.length).toBeLessThanOrEqual(4000);
  });

  it("summarizes scalar and object values deterministically", () => {
    expect(parseJsonlRecords("42")[0].summary).toBe("42");
    expect(parseJsonlRecords('{"title":"报告","status":"done","id":7}')[0].summary)
      .toBe("title: 报告 · status: done · id: 7");
  });
});
