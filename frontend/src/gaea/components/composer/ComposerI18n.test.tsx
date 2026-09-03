// v4.57 i18n 收尾冒烟：composer 家族（Composer/输入行/工具栏/拖放指示器）
// 的三语字典接线。钉住 zh 断言原硬编码文案；en 抽查一条验证键值生效。
import { describe, expect, it, vi, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import { Composer } from "../Composer";
import { ComposerInputRow } from "./ComposerInputRow";
import { ComposerToolbar } from "./ComposerToolbar";
import { ComposerDragOverlay } from "./ComposerDragOverlay";
import { LocaleProvider } from "../../lib/i18n";
import { ToastProvider } from "../Toast";

// Composer 编排层经 hooks 触达 bridge（Commands/SlashArgs/ListDir/ListWorkspaces…）：
// 统一给一个「任何方法都 resolve(null)」的桩，动作未触发时不产生行为。
// vi.hoisted：vi.mock 工厂被提升到文件顶，桩必须随提升创建。
const appStub = vi.hoisted(() => new Proxy({}, { get: () => vi.fn().mockResolvedValue(null) }));

vi.mock("../../lib/bridge", () => ({
  app: appStub,
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
  onReady: vi.fn(() => () => {}),
}));

const renderT = (ui: React.ReactNode, lang: "zh" | "en" = "zh") => {
  localStorage.setItem("gaea-lang", lang);
  return render(
    <LocaleProvider>
      <ToastProvider>{ui}</ToastProvider>
    </LocaleProvider>,
  );
};

const noop = () => {};
const nullRef = { current: null } as React.RefObject<HTMLTextAreaElement>;

describe("ComposerInputRow i18n 冒烟", () => {
  afterEach(cleanup);

  it("运行中发送按钮 title：插话/排队/纠正三态走字典（zh）", () => {
    renderT(
      <ComposerInputRow
        taRef={nullRef}
        text="hi"
        onTextChange={noop}
        onPaste={noop}
        onKeyDown={noop}
        placeholder="p"
        disabled={false}
        running
        composerHeightFixed={false}
        dragOver={false}
        shiftHeld={false}
        queueLen={2}
        pendingPaste={0}
        attachmentsCount={0}
        onDrop={noop}
        onDragOver={noop}
        onDragLeave={noop}
        onStop={noop}
        onSubmit={noop}
        onQueue={noop}
      />,
    );
    expect(screen.getByTitle("排队发送 (2)")).toBeTruthy();
    expect(screen.getByTitle("排队发送（当前回合结束后执行，与插话不同：插话立即调整当前任务）")).toBeTruthy();
  });

  it("Shift 按住时 title 切纠正发送；空闲时回退 composer.send", () => {
    renderT(
      <ComposerInputRow
        taRef={nullRef}
        text="hi"
        onTextChange={noop}
        onPaste={noop}
        onKeyDown={noop}
        placeholder="p"
        disabled={false}
        running
        composerHeightFixed={false}
        dragOver={false}
        shiftHeld
        queueLen={0}
        pendingPaste={0}
        attachmentsCount={0}
        onDrop={noop}
        onDragOver={noop}
        onDragLeave={noop}
        onStop={noop}
        onSubmit={noop}
        onQueue={noop}
      />,
    );
    expect(screen.getByTitle("纠正发送（Shift+Enter）")).toBeTruthy();
    cleanup();
    renderT(
      <ComposerInputRow
        taRef={nullRef}
        text="hi"
        onTextChange={noop}
        onPaste={noop}
        onKeyDown={noop}
        placeholder="p"
        disabled={false}
        running={false}
        composerHeightFixed={false}
        dragOver={false}
        shiftHeld={false}
        queueLen={0}
        pendingPaste={0}
        attachmentsCount={0}
        onDrop={noop}
        onDragOver={noop}
        onDragLeave={noop}
        onStop={noop}
        onSubmit={noop}
        onQueue={noop}
      />,
    );
    expect(screen.getByTitle("发送（Enter）")).toBeTruthy();
  });
});

describe("ComposerToolbar i18n 冒烟", () => {
  afterEach(cleanup);

  it("权限级别与思考深度徽标、截图 title 走字典（zh）", () => {
    renderT(
      <ComposerToolbar
        cwd="/ws"
        workspaceName="ws"
        workspaceMenuOpen={false}
        onToggleWorkspaceMenu={noop}
        workspaceAnchorRef={{ current: null } as React.RefObject<HTMLDivElement>}
        running={false}
        pendingPaste={0}
        captureBusy={false}
        onPickFiles={noop}
        onScreenshot={noop}
        permLevel="ask"
        onSetPermLevel={noop}
        thinkLevel="normal"
        onSetThinkLevel={noop}
      />,
    );
    expect(screen.getByText("询问")).toBeTruthy();
    expect(screen.getByText("自动")).toBeTruthy();
    expect(screen.getByText("标准")).toBeTruthy();
    expect(screen.getByTitle("截图：捕获屏幕并裁剪附加")).toBeTruthy();
    expect(screen.getByText("/ 命令")).toBeTruthy();
  });
});

describe("ComposerDragOverlay i18n 冒烟", () => {
  afterEach(cleanup);

  it("show=true 显示释放提示（zh/en 各渲染一次）", () => {
    renderT(<ComposerDragOverlay show />);
    expect(screen.getByText("释放以添加文件")).toBeTruthy();
    cleanup();
    renderT(<ComposerDragOverlay show />, "en");
    expect(screen.getByText("Release to add files")).toBeTruthy();
  });
});

describe("Composer i18n 冒烟", () => {
  afterEach(cleanup);

  it("项目感知 placeholder 走字典：空闲带工作区名（zh）/运行中提示（zh/en）", () => {
    renderT(
      <Composer running={false} cwd="/ws" onSend={noop} onCancel={() => undefined} onPickFolder={async () => "/ws"} />,
    );
    expect(screen.getByPlaceholderText("在 ws/ 中提问…")).toBeTruthy();
    cleanup();

    renderT(
      <Composer running cwd="/ws" onSend={noop} onCancel={() => undefined} onPickFolder={async () => "/ws"} />,
    );
    expect(screen.getByPlaceholderText("任务执行中… Enter 插话调整 · Shift+Enter 纠正")).toBeTruthy();
    cleanup();

    renderT(
      <Composer running={false} cwd="/ws" onSend={noop} onCancel={() => undefined} onPickFolder={async () => "/ws"} />,
      "en",
    );
    expect(screen.getByPlaceholderText("Ask in ws/…")).toBeTruthy();
  });
});
