import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { useComposerInsertStore } from "../lib/store";
import { Composer } from "./Composer";

function bridgeStub(commands: { name: string; kind: string; description: string }[] = []) {
  const calls: Record<string, ReturnType<typeof vi.fn>> = {};
  return new Proxy({} as Record<string, unknown>, {
    get(_target, prop) {
      const name = String(prop);
      if (!calls[name]) {
        calls[name] = vi.fn(async () => name === "Commands" ? commands : name === "ListWorkspaces" ? [] : null);
      }
      return calls[name];
    },
  });
}

const wenxueBridge = vi.hoisted(() => bridgeStub());
const gaeaBridge = vi.hoisted(() => bridgeStub());

vi.mock("../lib/bridge", () => ({ app: wenxueBridge }));
vi.mock("../../gaea/lib/bridge", () => ({ app: gaeaBridge }));

function renderComposer() {
  return render(
    <LocaleProvider>
      <Composer
        running={false}
        onSend={() => {}}
        onCancel={() => undefined}
        onPickFolder={async () => ""}
      />
    </LocaleProvider>,
  );
}

describe("Wenxue Composer 单源适配", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useComposerInsertStore.setState({ pendingAt: null, pendingText: null });
    localStorage.setItem("gaea-lang", "zh");
  });
  afterEach(cleanup);

  it("共享组件调用文学 bridge，不回落办公 bridge", async () => {
    renderComposer();

    await waitFor(() => expect(wenxueBridge.Commands).toHaveBeenCalled());
    expect(wenxueBridge.ListWorkspaces).toHaveBeenCalled();
    expect(gaeaBridge.Commands).not.toHaveBeenCalled();
    expect(gaeaBridge.ListWorkspaces).not.toHaveBeenCalled();
  });

  it("一键引用仍写入文学 Composer insert store", async () => {
    useComposerInsertStore.setState({ pendingText: "来自文学资料面板" });
    renderComposer();

    await waitFor(() => expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("来自文学资料面板"));
    expect(useComposerInsertStore.getState().pendingText).toBeNull();
  });
});
