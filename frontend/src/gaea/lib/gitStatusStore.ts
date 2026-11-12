import { create } from "zustand";
import { app } from "./bridge";
import type { GitStatusView } from "./types";

const GIT_POLL_MS = 30_000;

export interface GitStatusState {
  status: GitStatusView | null;
  loading: boolean;
  error: string;
  revision: number;
  refresh: () => Promise<void>;
}

let refCount = 0;
let timer: ReturnType<typeof setInterval> | null = null;
let focusBound = false;
let inFlight: Promise<void> | null = null;
let generation = 0;

export const useGitStatusStore = create<GitStatusState>()(() => ({
  status: null,
  loading: false,
  error: "",
  revision: 0,
  refresh: () => refreshGitStatus(),
}));

export function refreshGitStatus(): Promise<void> {
  if (inFlight) return inFlight;
  const fn = app.GaeaGitStatus;
  if (typeof fn !== "function") {
    useGitStatusStore.setState({ status: null, loading: false, error: "Git 绑定不可用" });
    return Promise.resolve();
  }
  const seq = generation;
  useGitStatusStore.setState({ loading: true, error: "" });
  inFlight = Promise.resolve()
    .then(() => fn())
    .then((status) => {
      if (seq !== generation) return;
      useGitStatusStore.setState((s) => ({
        status,
        loading: false,
        error: status?.isRepo ? "" : (status?.error ?? "当前工作区不是 Git 仓库"),
        revision: s.revision + 1,
      }));
    })
    .catch((error: unknown) => {
      if (seq !== generation) return;
      useGitStatusStore.setState((s) => ({
        status: { isRepo: false, files: [], error: error instanceof Error ? error.message : String(error) },
        loading: false,
        error: error instanceof Error ? error.message : String(error),
        revision: s.revision + 1,
      }));
    })
    .finally(() => {
      inFlight = null;
    });
  return inFlight;
}

function onFocus(): void {
  void refreshGitStatus();
}

/** Acquire one shared status poller; release on tab unmount. */
export function acquireGitStatus(): () => void {
  refCount++;
  if (refCount === 1) {
    generation++;
    void refreshGitStatus();
    timer = setInterval(() => {
      void refreshGitStatus();
    }, GIT_POLL_MS);
    if (typeof window !== "undefined") {
      window.addEventListener("focus", onFocus);
      focusBound = true;
    }
  }
  let released = false;
  return () => {
    if (released) return;
    released = true;
    refCount = Math.max(0, refCount - 1);
    if (refCount > 0) return;
    if (timer !== null) clearInterval(timer);
    timer = null;
    if (focusBound && typeof window !== "undefined") window.removeEventListener("focus", onFocus);
    focusBound = false;
    generation++;
    inFlight = null;
    useGitStatusStore.setState({ status: null, loading: false, error: "", revision: 0 });
  };
}

export function resetGitStatusStoreForTest(): void {
  if (timer !== null) clearInterval(timer);
  timer = null;
  if (focusBound && typeof window !== "undefined") window.removeEventListener("focus", onFocus);
  focusBound = false;
  refCount = 0;
  inFlight = null;
  generation++;
  useGitStatusStore.setState({ status: null, loading: false, error: "", revision: 0 });
}
