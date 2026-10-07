// WorkcostRatesEditor.tsx — 项目费率录入区（企管/规费/利润/税率 + 利润基数
// 含规费 + 控制价）。全部是**录入数据**——不做任何计算展示（数据库不是计算器）。
import { useEffect, useState } from "react";
import { Coins } from "../../icons";
import { app } from "../../lib/bridge";
import type { WorkcostBillProject } from "../../lib/types";
import { fieldCls, solidBtn } from "./WorkcostBillView";

const round2 = (v: number) => Math.round(v * 100) / 100;

export function RatesEditor({ project, onSaved }: { project: WorkcostBillProject; onSaved: () => void }) {
  const [mgmt, setMgmt] = useState(round2(project.managementRate * 100));
  const [reg, setReg] = useState(round2(project.regulatoryRate * 100));
  const [profit, setProfit] = useState(round2(project.profitRate * 100));
  const [tax, setTax] = useState(round2(project.taxRate * 100));
  const [inclReg, setInclReg] = useState(project.profitIncludesRegulatory);
  const [control, setControl] = useState(project.controlPrice);
  const [busy, setBusy] = useState(false);

  // 项目切换时同步表单值。
  useEffect(() => {
    setMgmt(round2(project.managementRate * 100));
    setReg(round2(project.regulatoryRate * 100));
    setProfit(round2(project.profitRate * 100));
    setTax(round2(project.taxRate * 100));
    setInclReg(project.profitIncludesRegulatory);
    setControl(project.controlPrice);
  }, [project]);

  const save = async () => {
    setBusy(true);
    try {
      await app.WorkcostBillProjectRatesSave(
        project.id,
        { managementRate: mgmt / 100, regulatoryRate: reg / 100, profitRate: profit / 100, taxRate: tax / 100 },
        inclReg,
        control,
      );
      onSaved();
    } catch {
      // 本地库保存失败极罕见；错误静默，用户可重试。
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <div className="flex items-center gap-2 mb-2">
        <Coins size={12} className="text-fg-faint" />
        <span className="text-[11.5px] text-fg font-medium">取费费率（录入数据）</span>
        <span className="text-[10px] text-fg-faint">费率 0 = 显式不计取该段；加总在导出的五表里由 Excel 算</span>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-6 gap-2 items-end">
        <RateInput label="企管 %" value={mgmt} onChange={setMgmt} />
        <RateInput label="规费 %" value={reg} onChange={setReg} />
        <RateInput label="利润 %" value={profit} onChange={setProfit} />
        <RateInput label="增值税 %" value={tax} onChange={setTax} />
        <label className="block">
          <span className="block text-[10.5px] text-fg-faint mb-1">控制价（元）</span>
          <input className={fieldCls} type="number" min={0} value={control || ""} aria-label="控制价" onChange={(e) => setControl(Number(e.target.value))} />
        </label>
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-1.5 text-[11px] text-fg-dim cursor-pointer select-none" title="两份实测产物口径差异：百锦路不含规费 / 市政道路含规费">
            <input type="checkbox" checked={inclReg} onChange={(e) => setInclReg(e.target.checked)} />
            利润基数含规费
          </label>
          <button type="button" className={solidBtn} onClick={() => void save()} disabled={busy}>
            {busy ? "保存中…" : "保存费率"}
          </button>
        </div>
      </div>
    </div>
  );
}

function RateInput({ label, value, onChange }: { label: string; value: number; onChange: (v: number) => void }) {
  return (
    <label className="block">
      <span className="block text-[10.5px] text-fg-faint mb-1">{label}</span>
      <input className={fieldCls} type="number" min={0} step="0.1" aria-label={label} value={value} onChange={(e) => onChange(Number(e.target.value))} />
    </label>
  );
}
