// CostBackupCard.tsx — 造价概览「数据备份」卡（v4.467）。
//
// 用户定调（2026-10-07）：「数据库不需要导出5表，但需要数据备份」。备份能力
// 本身早在（P4-3 全库一键备份：Hephaestus.db 用 VACUUM INTO 出一致性快照，
// 连同轻语数据/配置/会话打包 zip；恢复走 设置→数据），此前唯一入口在设置页
// ——造价页面无感。本卡把「立即备份」带到数据所在的位置。
//
// 诚实口径：造价数据在 Hephaestus.db 单库里（与记忆/知识同库），不存在
// 「只备造价」的切面——备份就是全库快照，造价数据随之安全。info 读取失败
// 不喊失败（体量显示「—」），备份动作本身失败才显形。
import { useCallback, useEffect, useState } from "react";
import { Archive } from "../../icons";
import { app } from "../../lib/bridge";

const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";

function fmtSize(n: number): string {
  if (n >= 1073741824) return (n / 1073741824).toFixed(2) + " GB";
  if (n >= 1048576) return (n / 1048576).toFixed(1) + " MB";
  if (n >= 1024) return (n / 1024).toFixed(0) + " KB";
  return `${n} B`;
}

/**
 * CostBackupCard — 概览页数据备份卡：体量速览 + 一键全库快照备份。
 */
export function CostBackupCard() {
  const [totalBytes, setTotalBytes] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState("");

  const load = useCallback(async () => {
    try {
      const r = (await app.DataBackupInfo()) ?? {};
      if (typeof r.total_bytes === "number") {
        setTotalBytes(r.total_bytes);
      }
    } catch {
      setTotalBytes(null); // 体量拿不到不拦备份——按钮照常可用
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const backup = useCallback(async () => {
    setBusy(true);
    setToast("");
    try {
      const destDir = await app.PickDirectory();
      if (!destDir) {
        setBusy(false); // 用户取消选择目录
        return;
      }
      const res = (await app.DataBackupCreate(destDir)) ?? {};
      const size = typeof res.total_bytes === "number" ? fmtSize(res.total_bytes) : "?";
      const sha = typeof res.sha256 === "string" ? `，SHA256 ${res.sha256.slice(0, 12)}…` : "";
      setToast(`备份完成：${String(res.zip_path ?? "")}（${size}${sha}）。恢复在 设置 → 数据。`);
      await load();
    } catch (e) {
      setToast(`备份失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [load]);

  return (
    <section
      className="v3-panel rounded-2xl p-4 flex flex-wrap items-center gap-3"
      data-testid="cost-backup-card"
    >
      <span className="w-8 h-8 rounded-lg bg-bg-elev inline-flex items-center justify-center shrink-0">
        <Archive size={15} className="text-accent" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="text-fg text-[12.5px] font-semibold">
          数据备份
          {totalBytes !== null && (
            <span className="ml-2 text-[10.5px] text-fg-faint font-normal tabular-nums">
              库与数据共 {fmtSize(totalBytes)}
            </span>
          )}
        </div>
        <p className="mt-0.5 text-[10.5px] text-fg-faint leading-relaxed">
          一键打全库 zip 快照（造价库 + 记忆 + 轻语 + 配置 + 会话，运行中备份安全）；
          恢复备份在 设置 → 数据。建议定期备到另一块盘。
        </p>
      </div>
      <button type="button" className={solidBtn} onClick={() => void backup()} disabled={busy} title="选择备份目录，生成带时间戳的 zip 快照">
        {busy ? "备份中…" : "立即备份"}
      </button>
      {toast && (
        <div className="fixed bottom-6 right-6 z-50 max-w-md v3-panel rounded-xl px-4 py-3 text-[11.5px] text-fg-dim shadow-lg">
          {toast}
          <button type="button" className={ghostBtn + " ml-2"} onClick={() => setToast("")}>关闭</button>
        </div>
      )}
    </section>
  );
}
