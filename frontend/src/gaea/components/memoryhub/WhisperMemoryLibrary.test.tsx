import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { WhisperMemoryLibrary } from "./WhisperMemoryLibrary";

const { memMock, epMock, replayMock, anchorsMock, anchorReplayMock, retellMock, pickDirMock, exportMock } = vi.hoisted(() => ({
  memMock: vi.fn(),
  epMock: vi.fn(),
  replayMock: vi.fn(),
  anchorsMock: vi.fn(),
  anchorReplayMock: vi.fn(),
  retellMock: vi.fn(),
  pickDirMock: vi.fn(),
  exportMock: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    WhisperMemories: memMock,
    WhisperEpisodes: epMock,
    WhisperEpisodeReplay: replayMock,
    WhisperAnchors: anchorsMock,
    WhisperAnchorReplay: anchorReplayMock,
    WhisperMemoryRetell: retellMock,
    PickDirectory: pickDirMock,
    WhisperExportArchive: exportMock,
  },
  openExternal: vi.fn(),
}));

// 3 条事实：IDENTITY ×2、SOCIAL ×1、未知 domain（OTHER → 归入「其他」组）。
const FACTS = [
  { id: "f1", domain: "IDENTITY", subcategory: "职业", subject: "用户是软件工程师", summary: "喜欢 TypeScript 与函数式编程，偏好深夜工作", weight: 0.9, confidence: 0.95, tier: "core", status: "active", updatedAt: "2026-08-01T10:00:00+08:00" },
  { id: "f2", domain: "IDENTITY", subcategory: "习惯", subject: "深夜编程效率最高", summary: "夜间注意力更集中", weight: 0.7, confidence: 0.8, tier: "normal", status: "active", updatedAt: "2026-08-02T10:00:00+08:00" },
  { id: "f3", domain: "SOCIAL", subcategory: "好友", subject: "好友阿黎", summary: "每周日一起爬山", weight: 0.6, confidence: 0.75, tier: "normal", status: "active", updatedAt: "2026-08-03T10:00:00+08:00" },
  { id: "f4", domain: "OTHER", subcategory: "宠物", subject: "橘猫年糕", summary: "家里养了一只橘猫", weight: 0.5, confidence: 0.6, tier: "normal", status: "active", updatedAt: "2026-08-04T10:00:00+08:00" },
];

const EPISODES = [
  { id: "e1", summary: "深夜一起改 bug 到天亮", dominantEmotion: "兴奋", emotionalIntensity: 0.85, keywords: ["debug", "熬夜"], startTurn: 3, endTurn: 8, createdAt: "2026-08-05T02:00:00+08:00", sourceSessionId: "s1" },
  { id: "e2", summary: "爬山途中的闲聊", dominantEmotion: "平静", emotionalIntensity: 0.4, keywords: ["爬山", "风景"], startTurn: 0, endTurn: 0, createdAt: "2026-08-03T09:30:00+08:00", sourceSessionId: "s2" },
];

const REPLAY = {
  id: "e1",
  summary: "深夜一起改 bug 到天亮",
  dominantEmotion: "兴奋",
  emotionalIntensity: 0.85,
  keywords: ["debug", "熬夜"],
  createdAt: "2026-08-05T02:00:00+08:00",
  sourceSessionId: "s1",
  startTurn: 3,
  endTurn: 8,
  replayable: true,
  dialogue: [
    { turnIndex: 3, role: "user", text: "这个 bug 搞不定了" },
    { turnIndex: 3, role: "assistant", text: "把报错贴给我看看" },
    { turnIndex: 4, role: "user", text: "找到了，是时区问题" },
  ],
};

const ANCHORS = [
  {
    id: "a1",
    anchorDate: "2026-05-20",
    anchorType: "recurring",
    recurrenceRule: "",
    domain: "user_profile",
    summary: "我的生日是 5 月 20 日",
    emotionalValence: 0.4,
    emotionalIntensity: 0.6,
    linkedFactIds: ["factBirthday"],
  },
];

const ANCHOR_REPLAY = {
  anchorId: "a1",
  anchorDate: "2026-05-20",
  anchorType: "recurring",
  domain: "user_profile",
  summary: "我的生日是 5 月 20 日",
  emotionalValence: 0.4,
  emotionalIntensity: 0.6,
  linkedFactSummaries: ["我的生日是 5 月 20 日"],
  replayable: true,
  episodeReplay: {
    id: "epAnchorA",
    summary: "聊到生日那天",
    dominantEmotion: "开心",
    emotionalIntensity: 0.6,
    keywords: ["生日"],
    createdAt: "2026-05-20T20:00:00+08:00",
    sourceSessionId: "s1",
    startTurn: 2,
    endTurn: 4,
    replayable: true,
    dialogue: [
      { turnIndex: 3, role: "user", text: "其实是 5 月 20 日" },
      { turnIndex: 3, role: "assistant", text: "我记下了" },
    ],
  },
};

describe("WhisperMemoryLibrary 聊天记忆库", () => {
  beforeEach(() => {
    memMock.mockReset();
    epMock.mockReset();
    replayMock.mockReset();
    anchorsMock.mockReset();
    anchorReplayMock.mockReset();
    retellMock.mockReset();
    pickDirMock.mockReset();
    exportMock.mockReset();
    memMock.mockResolvedValue(FACTS);
    epMock.mockResolvedValue(EPISODES);
    replayMock.mockResolvedValue({ ...REPLAY, dialogue: [], replayable: false });
    anchorsMock.mockResolvedValue(ANCHORS);
    anchorReplayMock.mockResolvedValue({ ...ANCHOR_REPLAY, episodeReplay: undefined, replayable: false });
    retellMock.mockResolvedValue("（mock）那天的雨声、你说话的样子，我都还记得。");
  });

  it("默认事实 tab 按 domain 分组渲染：身份/社交/其他 组标题与条数", async () => {
    render(<WhisperMemoryLibrary />);

    // 3.0 Wave 4：领域分类 emoji → antd 图标，组标题为纯文本标签
    expect(await screen.findByText("身份")).toBeTruthy();
    // 组头 = 图标 + 标签 + 条数（用父容器 textContent 断言条数）
    expect(screen.getByText("身份").parentElement!.textContent).toContain("2");
    expect(screen.getByText("社交")).toBeTruthy();
    expect(screen.getByText("社交").parentElement!.textContent).toContain("1");
    expect(screen.getByText("其他")).toBeTruthy();
    expect(screen.getByText("其他").parentElement!.textContent).toContain("1");

    // 各事实主体渲染
    expect(screen.getByText("用户是软件工程师")).toBeTruthy();
    expect(screen.getByText("好友阿黎")).toBeTruthy();
    expect(screen.getByText("橘猫年糕")).toBeTruthy();
  });

  it("搜索框按 subject/summary/subcategory 关键词过滤事实", async () => {
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");
    const input = screen.getByPlaceholderText("搜索记忆…");

    // subject 命中
    fireEvent.change(input, { target: { value: "工程师" } });
    expect(screen.getByText("用户是软件工程师")).toBeTruthy();
    expect(screen.queryByText("好友阿黎")).toBeNull();
    expect(screen.queryByText("深夜编程效率最高")).toBeNull();

    // summary 命中
    fireEvent.change(input, { target: { value: "爬山" } });
    expect(screen.getByText("好友阿黎")).toBeTruthy();
    expect(screen.queryByText("用户是软件工程师")).toBeNull();

    // subcategory 命中
    fireEvent.change(input, { target: { value: "宠物" } });
    expect(screen.getByText("橘猫年糕")).toBeTruthy();
    expect(screen.queryByText("好友阿黎")).toBeNull();

    // 清空恢复全量
    fireEvent.change(input, { target: { value: "" } });
    expect(screen.getByText("深夜编程效率最高")).toBeTruthy();
  });

  it("切到「情节」tab 显示情节列表：情绪 emoji / 强度条 / 关键词 chips / 轮次", async () => {
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /情节/ }));

    expect(await screen.findByText("深夜一起改 bug 到天亮")).toBeTruthy();
    expect(screen.getByText("爬山途中的闲聊")).toBeTruthy();
    // 情绪 emoji + 标签
    expect(screen.getByText("🤩 兴奋")).toBeTruthy();
    expect(screen.getByText("🙂 平静")).toBeTruthy();
    // 强度条（title 带百分比）
    expect(screen.getByTitle("情绪强度 85%")).toBeTruthy();
    expect(screen.getByTitle("情绪强度 40%")).toBeTruthy();
    // 关键词 chips（摘要文本是独立节点，精确匹配只命中 chip 本身）
    expect(screen.getByText("debug")).toBeTruthy();
    expect(screen.getAllByText("爬山").length).toBeGreaterThanOrEqual(1);
    // 轮次标注（仅 startTurn > 0）
    expect(screen.getByText(/第3-8轮/)).toBeTruthy();
  });

  it("点击事实打开详情 Modal：含 subject 与领域/子类/层级等字段", async () => {
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByText("用户是软件工程师"));

    // Modal 标题 = 聊天记忆 · subject；正文含领域/子类/层级等字段（dialog 内断言）
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("用户是软件工程师");
    expect(dialog.textContent).toContain("IDENTITY");
    expect(dialog.textContent).toContain("职业");
    expect(dialog.textContent).toContain("核心");
    expect(dialog.textContent).toContain("0.95");
    expect(dialog.textContent).toContain("f1");
  });

  it("点击情节打开详情并回放原始对话（用户/gaea 气泡 + 轮次标注）", async () => {
    replayMock.mockResolvedValue(REPLAY);
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /情节/ }));
    await screen.findByText("深夜一起改 bug 到天亮");
    fireEvent.click(screen.getByText("深夜一起改 bug 到天亮"));

    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("回放原始对话");
    expect(await screen.findByText("这个 bug 搞不定了")).toBeTruthy();
    expect(screen.getByText("把报错贴给我看看")).toBeTruthy();
    expect(screen.getByText("找到了，是时区问题")).toBeTruthy();
    // 第 3 轮出现两次（你 + gaea 两行），用 getAllByText
    expect(screen.getAllByText(/第3轮/).length).toBeGreaterThanOrEqual(2);
  });

  it("无原始对话的情节显示摘要回退提示（不可逐字回放）", async () => {
    replayMock.mockResolvedValue({ ...REPLAY, id: "e2", dialogue: [], replayable: false });
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /情节/ }));
    await screen.findByText("爬山途中的闲聊");
    fireEvent.click(screen.getByText("爬山途中的闲聊"));

    expect(await screen.findByText(/原始对话已超出保留范围/)).toBeTruthy();
  });

  it("切到「纪念日」tab 显示时间锚点：日期/类型徽标/摘要", async () => {
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /纪念日/ }));

    expect(await screen.findByText("2026-05-20")).toBeTruthy();
    expect(screen.getByText("周期纪念日")).toBeTruthy();
    expect(screen.getByText("我的生日是 5 月 20 日")).toBeTruthy();
    expect(screen.getByText(/情绪 60%/)).toBeTruthy();
  });

  it("点击纪念日打开回放：锚点摘要 + 关联事实 + 原始对话气泡", async () => {
    anchorReplayMock.mockResolvedValue(ANCHOR_REPLAY);
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /纪念日/ }));
    await screen.findByText("2026-05-20");
    fireEvent.click(screen.getByText("2026-05-20"));

    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("纪念日回放");
    expect(await screen.findByText("回放原始对话")).toBeTruthy();
    expect(screen.getByText("其实是 5 月 20 日")).toBeTruthy();
    expect(screen.getByText("我记下了")).toBeTruthy();
  });

  it("情节详情中点击「让 gaea 重述这段记忆」展示 LLM 叙事", async () => {
    retellMock.mockResolvedValue("那晚的雨声，我一直记得。");
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /情节/ }));
    await screen.findByText("深夜一起改 bug 到天亮");
    fireEvent.click(screen.getByText("深夜一起改 bug 到天亮"));
    fireEvent.click(await screen.findByRole("button", { name: /让 gaea 重述这段记忆/ }));

    expect(await screen.findByText("那晚的雨声，我一直记得。")).toBeTruthy();
    await waitFor(() => expect(retellMock).toHaveBeenCalledWith("episode", "e1", expect.any(String)));
  });

  it("纪念日详情中点击重述展示 LLM 叙事", async () => {
    retellMock.mockResolvedValue("你的生日，我怎么会忘。");
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /纪念日/ }));
    await screen.findByText("2026-05-20");
    fireEvent.click(screen.getByText("2026-05-20"));
    fireEvent.click(await screen.findByRole("button", { name: /让 gaea 重述这段记忆/ }));

    expect(await screen.findByText("你的生日，我怎么会忘。")).toBeTruthy();
    await waitFor(() => expect(retellMock).toHaveBeenCalledWith("anchor", "a1", expect.any(String)));
  });

  it("点击情节打开详情 Modal：含 summary 与情绪/强度/关键词/会话", async () => {
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");
    fireEvent.click(screen.getByRole("button", { name: /情节/ }));
    await screen.findByText("深夜一起改 bug 到天亮");

    fireEvent.click(screen.getByText("深夜一起改 bug 到天亮"));

    // 列表中已有「🤩 兴奋」，弹窗内又有一份 → 用 dialog 容器断言
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("深夜一起改 bug 到天亮");
    expect(dialog.textContent).toContain("🤩 兴奋");
    expect(dialog.textContent).toContain("85%");
    expect(dialog.textContent).toContain("debug、熬夜");
    expect(dialog.textContent).toContain("s1");
  });

  it("导出归档：PickDirectory 后调用 WhisperExportArchive(dir)", async () => {
    pickDirMock.mockResolvedValue("/tmp/out");
    exportMock.mockResolvedValue(2);
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /导出归档/ }));

    await waitFor(() => expect(pickDirMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(exportMock).toHaveBeenCalledWith("/tmp/out"));
  });

  it("事实为空时展示「暂无聊天记忆」空态", async () => {
    memMock.mockResolvedValue([]);
    render(<WhisperMemoryLibrary />);

    expect(await screen.findByText(/暂无聊天记忆/)).toBeTruthy();
  });

  it("情节为空时切到情节 tab 展示「暂无情节记忆」空态", async () => {
    epMock.mockResolvedValue([]);
    render(<WhisperMemoryLibrary />);
    await screen.findByText("身份");

    fireEvent.click(screen.getByRole("button", { name: /情节/ }));

    expect(await screen.findByText(/暂无情节记忆/)).toBeTruthy();
  });
});
