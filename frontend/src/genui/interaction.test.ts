// 审计 2026-09 #7：会话删除时按 sessionKey 前缀清理 genui 交互状态。
// stateKey 形态见 fingerprint.ts：消息槽位 genui:{scope}:{sessionKey}:{slot}:{fp}
// 与办公面板 genui:{scope}:panel:{sessionKey}:{panelKey}。
import { beforeEach, describe, expect, it } from "vitest";
import {
  clearBlockStatesForSession,
  loadBlockState,
  resetInteractionStore,
  saveBlockState,
} from "./interaction";

beforeEach(() => {
  resetInteractionStore();
});

describe("clearBlockStatesForSession", () => {
  it("清理该会话的消息槽位与面板键，其他会话保留", () => {
    saveBlockState("genui:office:s1:h0:abc", { answers: { q: "a" } });
    saveBlockState("genui:office:panel:s1:main", { fields: { f: "v" } });
    saveBlockState("genui:chat:s1:0:xyz", { locked: true });
    saveBlockState("genui:office:s2:h0:def", { answers: { q: "b" } });
    saveBlockState("genui:office:panel:s2:main", { locked: true });

    clearBlockStatesForSession("s1");

    expect(loadBlockState("genui:office:s1:h0:abc")).toBeNull();
    expect(loadBlockState("genui:office:panel:s1:main")).toBeNull();
    expect(loadBlockState("genui:chat:s1:0:xyz")).toBeNull();
    expect(loadBlockState("genui:office:s2:h0:def")).toEqual({ answers: { q: "b" } });
    expect(loadBlockState("genui:office:panel:s2:main")).toEqual({ locked: true });
  });

  it("冒号边界匹配：前缀相似的会话 key 不被误伤", () => {
    saveBlockState("genui:office:s10:h0:fp", { locked: true });
    saveBlockState("genui:office:panel:s1extra:main", { locked: true });

    clearBlockStatesForSession("s1");

    expect(loadBlockState("genui:office:s10:h0:fp")).not.toBeNull();
    expect(loadBlockState("genui:office:panel:s1extra:main")).not.toBeNull();
  });

  it("空 sessionKey 为 no-op，store 不变", () => {
    saveBlockState("genui:office:s1:h0:fp", { locked: true });
    clearBlockStatesForSession("");
    expect(loadBlockState("genui:office:s1:h0:fp")).not.toBeNull();
  });

  it("无匹配键时是安全的 no-op", () => {
    expect(() => clearBlockStatesForSession("missing")).not.toThrow();
  });
});
