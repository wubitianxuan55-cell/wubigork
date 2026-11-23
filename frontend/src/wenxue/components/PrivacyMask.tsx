// PrivacyMask.tsx — 图文板块失焦自动遮罩（2026-11-23 私密保护四形态之一）。
//
// 布防判定在 useWenxuePrivacy（maskArmed = maskOnBlur && 当前书无限制档）；
// 本组件只负责：window blur / visibilitychange(hidden) → 上罩；点击 →
// 未锁直接摘罩 / 已锁（PIN 已设未解锁）弹 PIN 输入，解锁成功才摘。
// 罩面 = 板块全屏毛玻璃 + 提示条，不遮壳层（办公板块无书稿内容）。
import { useCallback, useEffect, useRef, useState } from "react";
import { useT } from "../lib/i18n";

export function PrivacyMask({
  armed,
  locked,
  onUnlock,
}: {
  armed: boolean;
  locked: boolean;
  onUnlock: (pin: string) => Promise<boolean>;
}) {
  const t = useT();
  const [masked, setMasked] = useState(false);
  const [askPin, setAskPin] = useState(false);
  const [pin, setPin] = useState("");
  const [wrong, setWrong] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!armed) {
      // 布防解除（关开关/切到非无限制书）：同时收罩与 PIN 面。
      setMasked(false);
      setAskPin(false);
      return;
    }
    const arm = () => setMasked(true);
    const onVis = () => {
      if (document.visibilityState === "hidden") setMasked(true);
    };
    window.addEventListener("blur", arm);
    document.addEventListener("visibilitychange", onVis);
    return () => {
      window.removeEventListener("blur", arm);
      document.removeEventListener("visibilitychange", onVis);
    };
  }, [armed]);

  useEffect(() => {
    if (askPin) inputRef.current?.focus();
  }, [askPin]);

  const dismiss = useCallback(() => {
    if (locked) {
      setAskPin(true);
      setWrong(false);
      return;
    }
    setMasked(false);
  }, [locked]);

  const tryUnlock = useCallback(async () => {
    const ok = await onUnlock(pin);
    if (ok) {
      setMasked(false);
      setAskPin(false);
      setPin("");
      setWrong(false);
    } else {
      setWrong(true);
      setPin("");
      inputRef.current?.focus();
    }
  }, [onUnlock, pin]);

  if (!masked) return null;

  return (
    <div
      data-testid="wenxue-privacy-mask"
      className="wx-privacy-mask"
      role="button"
      tabIndex={0}
      aria-label={t("wenxue.privacy.maskTitle")}
      onClick={askPin ? undefined : dismiss}
      onKeyDown={(e) => {
        if (!askPin && (e.key === "Enter" || e.key === " ")) dismiss();
      }}
    >
      {askPin ? (
        <div
          className="wx-privacy-pin"
          role="dialog"
          aria-label={t("wenxue.privacy.pinTitle")}
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
          <div className="wx-privacy-pin-title">{t("wenxue.privacy.pinTitle")}</div>
          <input
            ref={inputRef}
            data-testid="wenxue-privacy-pin-input"
            type="password"
            inputMode="numeric"
            autoComplete="off"
            maxLength={12}
            placeholder={t("wenxue.privacy.pinPlaceholder")}
            value={pin}
            aria-invalid={wrong}
            onChange={(e) => {
              setPin(e.target.value.replace(/\D/g, ""));
              setWrong(false);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") void tryUnlock();
            }}
          />
          {wrong && <div className="wx-privacy-pin-wrong">{t("wenxue.privacy.pinWrong")}</div>}
          <div className="wx-privacy-pin-actions">
            <button
              data-testid="wenxue-privacy-pin-cancel"
              onClick={() => {
                setAskPin(false);
                setPin("");
                setWrong(false);
              }}
            >
              {t("wenxue.privacy.pinCancel")}
            </button>
            <button data-testid="wenxue-privacy-pin-unlock" onClick={() => void tryUnlock()}>
              {t("wenxue.privacy.pinUnlock")}
            </button>
          </div>
        </div>
      ) : (
        <div className="wx-privacy-mask-card" data-testid="wenxue-privacy-mask-card">
          <div className="wx-privacy-mask-title">{t("wenxue.privacy.maskTitle")}</div>
          <div className="wx-privacy-mask-hint">{t("wenxue.privacy.maskHint")}</div>
        </div>
      )}
    </div>
  );
}
