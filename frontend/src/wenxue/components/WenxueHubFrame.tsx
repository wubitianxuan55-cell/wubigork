import type { ReactNode } from "react";
import type { Icon } from "../icons";

// WenxueHubFrame — 角色 / 阅读 / 设定三个文学工作标签页的共享页眉。
// 只负责建立一致的“档案室”层级：细窄眉题、宋体主标题、摘要、指标与动作。
// 数据面仍由各 Hub 自己持有；这里不引入任何绑定或隐式写入。

export interface WenxueHubMetric {
  label: string;
  value: ReactNode;
  hint?: string;
}

export interface WenxueHubAction {
  key: string;
  label: string;
  icon: Icon;
  onClick: () => void;
  primary?: boolean;
  disabled?: boolean;
  title?: string;
  testId?: string;
}

export function WenxueHubFrame({
  tone,
  eyebrow,
  title,
  description,
  metrics,
  actions,
  children,
}: {
  tone: "characters" | "reading" | "settings";
  eyebrow: string;
  title: string;
  description: string;
  metrics: WenxueHubMetric[];
  actions?: WenxueHubAction[];
  children: ReactNode;
}) {
  return (
    <section className={`wx-hub wx-hub--${tone}`} aria-labelledby={`wx-hub-title-${tone}`}>
      <header className="wx-hub__masthead">
        <div className="wx-hub__heading">
          <div className="wx-hub__eyebrow">{eyebrow}</div>
          <h1 id={`wx-hub-title-${tone}`} className="wx-hub__title">{title}</h1>
          <p className="wx-hub__description">{description}</p>
        </div>

        {metrics.length > 0 && (
          <dl className="wx-hub__metrics" aria-label={`${title}概览`}>
            {metrics.map((metric) => (
              <div key={metric.label} className="wx-hub__metric">
                <dt>{metric.label}</dt>
                <dd>{metric.value}</dd>
                {metric.hint && <span>{metric.hint}</span>}
              </div>
            ))}
          </dl>
        )}

        {actions && actions.length > 0 && (
          <div className="wx-hub__actions">
            {actions.map(({ key, label, icon: Icon, onClick, primary, disabled, title, testId }) => (
              <button
                key={key}
                type="button"
                className={`wx-hub-action${primary ? " is-primary" : ""}`}
                onClick={onClick}
                disabled={disabled}
                title={title}
                data-testid={testId}
              >
                <Icon size={13} aria-hidden />
                <span>{label}</span>
              </button>
            ))}
          </div>
        )}
      </header>
      <div className="wx-hub__body">{children}</div>
    </section>
  );
}

export function WenxueHubEmpty({
  icon: Icon,
  title,
  description,
  action,
  children,
}: {
  icon: Icon;
  title: string;
  description: string;
  action?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="wx-hub-empty">
      <div className="wx-hub-empty__mark" aria-hidden>
        <Icon size={24} />
      </div>
      <h2>{title}</h2>
      <p>{description}</p>
      {action && <div className="wx-hub-empty__action">{action}</div>}
      {children}
    </div>
  );
}
