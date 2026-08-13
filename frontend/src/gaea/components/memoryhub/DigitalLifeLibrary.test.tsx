import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { DigitalLifeLibrary } from "./DigitalLifeLibrary";

const { lifeMock, opsMock } = vi.hoisted(() => ({ lifeMock: vi.fn(), opsMock: vi.fn() }));

vi.mock("../../lib/bridge", () => ({
  app: {
    HerdsmanDigitalLife: lifeMock,
    HerdsmanOperations: opsMock,
  },
}));

const LIFE = {
  available: true,
  source: "herdsman-digital-life",
  character_count: 1,
  timeline_events: 10,
  state_commits: 215240,
  world_events: 5303,
  memory_events: 79,
  memory_summaries: 1,
  relationships: 1,
  turn_traces: 49,
  characters: [
    {
      id: "c1",
      name: "林晚",
      gender: "女",
      identity: "品牌设计师",
      worldview: "现实都市",
      text_model: "Qwen3.5-35B-A3B-MTP",
      intimacy: 89,
      trust: 62,
      safety: 77,
      conflict: 0,
      last_interacted_at: "2026-08-09T00:17:07+08:00",
      memory_summary: "关系画像: 稳定互动 / 用户主动联系",
      highlights: ["CosyVoice 部署", "AMD ROCm"],
      memory_event_count: 12,
      reinforcement: 32,
      updated_at: "",
    },
  ],
  recent_timeline: [{ type: "system", title: "character created", summary: "林晚", occurred_at: "2026-07-30T21:48:04+08:00" }],
  recent_world: [{ type: "npc_actor", title: "妈妈问起近况", summary: "给了她一点现实里的稳定感。", occurred_at: "2026-07-30T21:51:52+08:00" }],
};

describe("DigitalLifeLibrary 数字生命库", () => {
  it("渲染角色/关系/记忆摘要/时间线/世界事件/最近操作", async () => {
    lifeMock.mockResolvedValue(LIFE);
    opsMock.mockResolvedValue({
      total: 1,
      items: [
        {
          id: "op1",
          kind: "image_generate",
          model: "zimage-turbo",
          status: "completed",
          stage: "completed",
          progress: 100,
          artifacts: 1,
          created_at: "2026-08-13T21:09:23+08:00",
          completed_at: "",
        },
      ],
      source: "herdsman-operations",
    });
    render(<DigitalLifeLibrary />);

    expect(await screen.findByText("数字生命 · Herdsman")).toBeTruthy();
    expect(screen.getAllByText("林晚").length).toBeGreaterThan(0);
    expect(screen.getByText("品牌设计师")).toBeTruthy();
    expect(screen.getByText("亲密度")).toBeTruthy();
    expect(screen.getByText(/稳定互动/)).toBeTruthy();
    expect(screen.getByText("最近时间线")).toBeTruthy();
    expect(screen.getByText("character created")).toBeTruthy();
    expect(screen.getByText("妈妈问起近况")).toBeTruthy();
    expect(screen.getByText("最近 Herdsman 操作 · 1")).toBeTruthy();
    expect(screen.getByText("生图")).toBeTruthy();
  });

  it("数字生命不可用时展示错误空态", async () => {
    lifeMock.mockResolvedValue({ ...LIFE, available: false, error: "数字生命库不存在" });
    opsMock.mockResolvedValue({ total: 0, items: [], source: "x" });
    render(<DigitalLifeLibrary />);
    await waitFor(() => expect(screen.getByText(/数字生命库不存在/)).toBeTruthy());
  });
});
