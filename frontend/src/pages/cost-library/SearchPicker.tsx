// SearchPicker — 搜出来再选（2026-11-28）。挂接定额、含量行加工料机不再要求知道编码，
// 也不再只列前 500 条工料机。输入即搜（防抖 250ms），走板块已有的列表绑定——语义索引
// 就绪时后端是混合检索，名称不含关键词、意思相近的也会回来。直接敲编码回车照常可用。
import { useEffect, useRef, useState } from "react";

export interface PickerOption {
  code: string;
  title: string;
  spec?: string;
}

export function SearchPicker({ label, placeholder, text, onText, search, onPick, testid }: {
  /** aria-label（也是测试锚点）。 */
  label: string;
  placeholder: string;
  /** 输入框里的字（调用方持有：外面的「挂接 / 加行」按钮要用它）。 */
  text: string;
  onText: (next: string) => void;
  /** 按关键词取候选（调用方接绑定；函数身份要稳定）。 */
  search: (kw: string) => Promise<PickerOption[]>;
  /** 选定一个编码（点候选，或回车按输入的字直接当编码）。 */
  onPick: (code: string) => void;
  testid?: string;
}) {
  const [options, setOptions] = useState<PickerOption[]>([]);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const seq = useRef(0);

  useEffect(() => {
    const kw = text.trim();
    if (!kw) {
      setOptions([]);
      setOpen(false);
      return;
    }
    const mine = ++seq.current;
    setBusy(true);
    const timer = window.setTimeout(async () => {
      let got: PickerOption[] = [];
      try {
        got = (await search(kw)) ?? [];
      } catch {
        got = [];
      }
      if (mine !== seq.current) return;
      setOptions(got.slice(0, 12));
      setOpen(true);
      setBusy(false);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [text, search]);

  const pick = (code: string) => {
    setOpen(false);
    onPick(code);
  };

  return (
    <div className="cl-picker" data-testid={testid}>
      <input
        className="cl-edit-input"
        aria-label={label}
        placeholder={placeholder}
        value={text}
        onChange={(e) => onText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            pick(text.trim());
          } else if (e.key === "Escape") {
            // 只收候选，不往外冒——外层编辑面也监听了 Esc（取消编辑），不能一次按键两件事。
            e.stopPropagation();
            setOpen(false);
          }
        }}
      />
      {open && (
        <ul className="cl-picker-list" role="listbox" aria-label={`${label}候选`}>
          {options.length === 0 && !busy && <li className="cl-picker-empty">没有匹配的；直接敲编码再点按钮也可以</li>}
          {options.map((o) => (
            <li key={o.code} role="option" aria-selected={false}>
              <button type="button" className="cl-picker-item" onClick={() => pick(o.code)}>
                <span className="cl-code">{o.code}</span> {o.title}
                {o.spec ? <span className="cl-dim"> · {o.spec}</span> : null}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
