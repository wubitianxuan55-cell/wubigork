// useWenxuePrivacy — 图文私密保护面（2026-11-23 无限制档小说四形态）的状态
// 与动作单源：Overview 拉取/解锁/回锁/遮罩开关/单书隐藏；失焦遮罩的 armed
// 判定（当前书无限制档 && maskOnBlur）也在此，PrivacyMask 只吃状态渲染。
//
// 刷新时机：挂载、wenxue:workspace-changed（切书后档位/清单可能变）、每次
// 动作完成后。锁定态下 Overview 的隐藏书名字/路径已由后端置空。
import { useCallback, useEffect, useState } from "react";
import { app } from "../lib/bridge";
import type { WenxuePrivacyOverview } from "../lib/bridge/wenxue";

const UNRESTRICTED = "深度成人（无限制）";

const EMPTY: WenxuePrivacyOverview = {
  pinSet: false,
  unlocked: true,
  maskOnBlur: true,
  hiddenCount: 0,
  books: [],
};

export function useWenxuePrivacy() {
  const [overview, setOverview] = useState<WenxuePrivacyOverview>(EMPTY);

  const refresh = useCallback(() => {
    app.WenxuePrivacyOverview()
      .then((ov) => setOverview(ov ?? EMPTY))
      .catch(() => setOverview(EMPTY));
  }, []);

  useEffect(() => {
    refresh();
    window.addEventListener("wenxue:workspace-changed", refresh);
    return () => window.removeEventListener("wenxue:workspace-changed", refresh);
  }, [refresh]);

  // 当前书是否无限制档（Overview 锁定态下隐藏书路径为空——用 current 位找）。
  const currentBookUnrestricted = overview.books.some((b) => b.current && b.adultLevel === UNRESTRICTED);

  // 当前书是否在隐藏清单（顶栏书名 chip 据此显「私密作品」；现书会话仍可用，
  // 只是浏览面——侧栏/首页/切换菜单展示名——不再暴露）。
  const currentBookHidden = overview.books.some((b) => b.current && b.hidden);

  // 失焦遮罩布防：开关开 + 当前书无限制档。PIN 未设时也遮（点击即摘），
  // 设了 PIN 且锁定时点击先走解锁。
  const maskArmed = overview.maskOnBlur && currentBookUnrestricted;

  const unlock = useCallback(
    async (pin: string) => {
      try {
        await app.WenxuePrivacyUnlock(pin);
        refresh();
        return true;
      } catch {
        return false;
      }
    },
    [refresh],
  );

  const lock = useCallback(() => {
    // 返回 bridge promise 供调用方 await 落定后再广播（竞速根修）；内部仍自刷。
    return app.WenxuePrivacyLock().then(refresh).catch(() => {});
  }, [refresh]);

  const setPin = useCallback(
    async (pin: string, oldPin: string) => {
      await app.WenxuePrivacySetPin(pin, oldPin);
      refresh();
    },
    [refresh],
  );

  const setMaskOnBlur = useCallback(
    (on: boolean) => {
      app.WenxuePrivacySetMaskOnBlur(on).then(refresh).catch(() => {});
    },
    [refresh],
  );

  const setBookHidden = useCallback(
    (name: string, hidden: boolean) => {
      // 同上：透传 promise（await 后广播才不会拉到旧清单）。
      return app.WenxuePrivacySetBookHidden(name, hidden).then(refresh).catch(() => {});
    },
    [refresh],
  );

  return {
    overview,
    refresh,
    currentBookUnrestricted,
    currentBookHidden,
    maskArmed,
    locked: overview.pinSet && !overview.unlocked,
    unlock,
    lock,
    setPin,
    setMaskOnBlur,
    setBookHidden,
  };
}
