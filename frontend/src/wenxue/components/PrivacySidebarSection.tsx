// PrivacySidebarSection.tsx — 左栏私密保护区（2026-11-23 无限制档小说四形态；
// 2026-11-24 用户拍板收敛为单按钮）。
//
// 常驻面 = 一个「私密保护」按钮，零书名零数量（白隐藏问题：之前常驻眼睛行
// 直接列书名，旁人扫侧栏即见）。点击展开管理面板（用户主动行为，短暂显示）：
//   · 锁定态（PIN 已设未解锁）：面板只有 PIN 输入，无任何书名；
//   · 解锁态：逐书眼睛开关、回锁、设置/修改 PIN、失焦遮罩开关。
// 解锁/回锁/隐藏开关后广播 wenxue:workspace-changed，侧栏书单重拉。
import { useState } from "react";
import { Eye, EyeOff, KeyRound, Lock, LockOpen } from "../icons";
import { useT } from "../lib/i18n";
import { useWenxuePrivacy } from "../app/useWenxuePrivacy";

export function PrivacySidebarSection() {
  const t = useT();
  const { overview, locked, unlock, lock, setPin, setMaskOnBlur, setBookHidden } = useWenxuePrivacy();
  const [open, setOpen] = useState(false);
  const [pinInput, setPinInput] = useState("");
  const [pinWrong, setPinWrong] = useState(false);
  const [setPinOpen, setSetPinOpen] = useState(false);
  const [oldPin, setOldPin] = useState("");
  const [setPinErr, setSetPinErr] = useState("");

  const hiddenBooks = overview.books.filter((b) => b.hidden);
  // 面板书行 = 无限制档书全集（含已隐藏——眼睛开关双向）。锁定态后端已把
  // 书名置空（book.name==""），面板行退化为「私密书籍」占位。
  const unrestrictedBooks = overview.books.filter(
    (b) => b.adultLevel === "深度成人（无限制）" || b.hidden,
  );
  const notifyChanged = () => window.dispatchEvent(new Event("wenxue:workspace-changed"));

  const doUnlock = async () => {
    const ok = await unlock(pinInput);
    if (ok) {
      setPinInput("");
      setPinWrong(false);
      notifyChanged();
    } else {
      setPinWrong(true);
      setPinInput("");
    }
  };

  const doSetPin = async () => {
    try {
      await setPin(pinInput, oldPin);
      setSetPinOpen(false);
      setPinInput("");
      setOldPin("");
      setSetPinErr("");
      notifyChanged();
    } catch (e) {
      setSetPinErr(e instanceof Error ? e.message : String(e));
    }
  };

  // 有无限制档书（或已设 PIN/有隐藏书）才渲染按钮：全年龄书库不出隐私区。
  const hasUnrestricted = overview.books.some((b) => b.adultLevel === "深度成人（无限制）");
  if (!overview.pinSet && hiddenBooks.length === 0 && !hasUnrestricted) return null;

  return (
    <div className="wx-privacy-sidebar" data-testid="wenxue-privacy-sidebar">
      {/* 常驻单按钮：零书名零数量。 */}
      <button
        className="wx-privacy-entry"
        data-testid="wenxue-privacy-entry"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        {locked ? <Lock size={13} aria-hidden /> : <EyeOff size={13} aria-hidden />}
        <span>{t("wenxue.privacy.sidebarTitle")}</span>
      </button>

      {open && (
        <div className="wx-privacy-panel" data-testid="wenxue-privacy-panel">
          {locked ? (
            <div className="wx-privacy-locked">
              <div className="wx-privacy-locked-row" data-testid="wenxue-privacy-locked-row">
                <Lock size={12} aria-hidden />
                <span>
                  {overview.hiddenCount} {t("wenxue.privacy.lockedRow")}
                </span>
              </div>
              <div className="wx-privacy-pin-inline">
                <input
                  data-testid="wenxue-privacy-sidebar-pin"
                  type="password"
                  inputMode="numeric"
                  autoComplete="off"
                  maxLength={12}
                  placeholder={t("wenxue.privacy.pinPlaceholder")}
                  value={pinInput}
                  aria-invalid={pinWrong}
                  onChange={(e) => {
                    setPinInput(e.target.value.replace(/\D/g, ""));
                    setPinWrong(false);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") void doUnlock();
                  }}
                />
                {pinWrong && <span className="wx-privacy-pin-wrong">{t("wenxue.privacy.pinWrong")}</span>}
                <button data-testid="wenxue-privacy-sidebar-unlock" onClick={() => void doUnlock()}>
                  <LockOpen size={12} aria-hidden />
                  {t("wenxue.privacy.unlockRow")}
                </button>
              </div>
            </div>
          ) : (
            <>
              {unrestrictedBooks.length > 0 && (
                <div className="wx-privacy-books" data-testid="wenxue-privacy-books">
                  {unrestrictedBooks.map((b, i) => (
                    <button
                      key={b.name || b.path}
                      className="wx-privacy-book-row"
                      data-testid="wenxue-privacy-book-row"
                      title={b.hidden ? t("wenxue.privacy.unlockRow") : t("wenxue.privacy.maskTitle")}
                      onClick={async () => {
                        // 先等 bridge 落定再广播：同步广播会让 listener 重拉到
                        // 旧清单（竞速——2026-11-23 沙箱实弹逮住）。
                        await setBookHidden(b.name, !b.hidden);
                        notifyChanged();
                      }}
                    >
                      {b.hidden ? <EyeOff size={12} aria-hidden /> : <Eye size={12} aria-hidden />}
                      <span className="wx-privacy-book-name">
                        {b.name || `${t("wenxue.privacy.sidebarTitle")} ${String(i + 1).padStart(2, "0")}`}
                      </span>
                    </button>
                  ))}
                </div>
              )}
              <div className="wx-privacy-actions">
                <button
                  data-testid="wenxue-privacy-lock-now"
                  onClick={async () => {
                    // lock 的 bridge 调用也等落定再广播（同隐藏行竞速口径）。
                    await lock();
                    notifyChanged();
                  }}
                  disabled={!overview.pinSet}
                >
                  <Lock size={12} aria-hidden />
                  {t("wenxue.privacy.lockNow")}
                </button>
                <button data-testid="wenxue-privacy-setpin-toggle" onClick={() => setSetPinOpen((v) => !v)}>
                  <KeyRound size={12} aria-hidden />
                  {t("wenxue.privacy.setPin")}
                </button>
              </div>
              <label className="wx-privacy-mask-toggle" data-testid="wenxue-privacy-mask-toggle">
                <input
                  type="checkbox"
                  checked={overview.maskOnBlur}
                  onChange={(e) => setMaskOnBlur(e.target.checked)}
                />
                <span>{t("wenxue.privacy.maskToggle")}</span>
              </label>
              {setPinOpen && (
                <div className="wx-privacy-setpin">
                  {overview.pinSet && (
                    <input
                      type="password"
                      inputMode="numeric"
                      autoComplete="off"
                      maxLength={12}
                      placeholder={t("wenxue.privacy.pinTitle")}
                      value={oldPin}
                      onChange={(e) => setOldPin(e.target.value.replace(/\D/g, ""))}
                    />
                  )}
                  <input
                    data-testid="wenxue-privacy-setpin-input"
                    type="password"
                    inputMode="numeric"
                    autoComplete="off"
                    maxLength={12}
                    placeholder={t("wenxue.privacy.pinPlaceholder")}
                    value={pinInput}
                    onChange={(e) => setPinInput(e.target.value.replace(/\D/g, ""))}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") void doSetPin();
                    }}
                  />
                  {setPinErr && <span className="wx-privacy-pin-wrong">{setPinErr}</span>}
                  <button data-testid="wenxue-privacy-setpin-save" onClick={() => void doSetPin()}>
                    {t("wenxue.privacy.setPin")}
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
