// DictSelect — 规范值下拉（专业 / 地区 / 分类）。选项来自造价库字典，不再是「库里已有
// 取值」（2026-11-28）：字典里规划了、还没有行在用的值也列出来——定额专业下拉此前只有
// 「土壤修复」一项，字典里另外 9 个专业选不到。没有合适的可以就地新增（走字典写入，
// 规矩在存储层把关：地区只到省、重名幂等），不必绕回造价模式。
// 字典读不到（老版本后端 / 读取失败）时只有调用方给的回退选项、不出「新增」。
import { useState } from "react";
import { Plus, X } from "../../gaea/icons";

export interface DictSelectProps {
  /** aria-label。 */
  label: string;
  value: string;
  onChange: (next: string) => void;
  /** 空值那一项的文案（如「未标注」）；不给 = 不出空值项。 */
  emptyLabel?: string;
  hint?: string;
  testid?: string;
  disabled?: boolean;
  /** 选项（字典值在前、库里已有取值兜底，调用方并好）。 */
  options: string[];
  /** 就地新增：成功返回库里最终的写法（地区会被归一成省级全称）。不给 = 不出「新增」。 */
  add?: (name: string) => Promise<string>;
}

const ADD = "__add__";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export function DictSelect({ label, value, onChange, emptyLabel, hint, testid, disabled, options, add }: DictSelectProps) {
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const confirmAdd = async () => {
    const name = draft.trim();
    if (!name) {
      setError("先写上要新增的值");
      return;
    }
    setBusy(true);
    setError("");
    try {
      onChange(await add!(name));
      setAdding(false);
      setDraft("");
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="cl-dict">
      {adding ? (
        <div className="cl-edit-row">
          <input
            className="cl-edit-input"
            aria-label={`新增${label}`}
            value={draft}
            disabled={busy}
            autoFocus
            placeholder={hint ?? ""}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") void confirmAdd();
              // 只退「新增」这一行，不往外冒——外层编辑面也监听了 Esc（取消编辑）。
              if (e.key === "Escape") {
                e.stopPropagation();
                setAdding(false);
                setDraft("");
                setError("");
              }
            }}
          />
          <button type="button" className="cl-btn" disabled={busy} data-testid={`${testid ?? "dict"}-confirm`} onClick={() => void confirmAdd()}>
            <Plus size={12} aria-hidden />加入字典
          </button>
          <button
            type="button"
            className="cl-btn cl-btn-ghost"
            aria-label="取消新增"
            disabled={busy}
            onClick={() => { setAdding(false); setDraft(""); setError(""); }}
          >
            <X size={12} aria-hidden />取消
          </button>
        </div>
      ) : (
        <select
          className="cl-select"
          aria-label={label}
          data-testid={testid}
          value={value}
          disabled={disabled}
          onChange={(e) => {
            if (e.target.value === ADD) {
              setAdding(true);
              setError("");
              return;
            }
            onChange(e.target.value);
          }}
        >
          {emptyLabel && <option value="">{emptyLabel}</option>}
          {options.map((o) => <option key={o} value={o}>{o}</option>)}
          {value && !options.includes(value) && <option value={value}>{value}</option>}
          {add && <option value={ADD}>＋ 新增…</option>}
        </select>
      )}
      {error && <span className="cl-warn-text" role="alert">{error}</span>}
    </div>
  );
}
