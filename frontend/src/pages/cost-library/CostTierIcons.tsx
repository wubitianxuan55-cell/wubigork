import type { ReactNode, SVGProps } from "react";

import type { Tier } from "./logic";

/**
 * 造价数据库三级图标。
 *
 * 与一级板块图标共用 24×24 画布、1.6px 圆角描边。三枚图形分别取真实工作对象：
 * 清单＝勾选台账，定额＝带卡尺的标准单元，工料机＝人工 / 材料 / 机械资源组。
 * 小尺寸下只靠外轮廓与粗关系线区分，不靠文字或颜色。
 */
interface CostTierIconProps extends Omit<SVGProps<SVGSVGElement>, "children"> {
  tier: Tier;
  size?: number | string;
}

function IconFrame({ tier, size = 22, children, ...props }: CostTierIconProps & { children: ReactNode }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      data-cost-tier-icon={tier}
      {...props}
    >
      {children}
    </svg>
  );
}

/** 清单：带装订线与勾选项的工程量台账。 */
function BillIcon(props: CostTierIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M6.1 4.25h9.55L19 7.6v12.15H6.1z" />
      <path d="M15.65 4.25V7.6H19" />
      <path d="M8.35 8.55h1.55M8.35 12.25h1.55M8.35 15.95h1.55" />
      <path d="m8.55 8.7.65.65 1.2-1.2M8.55 12.4l.65.65 1.2-1.2M8.55 16.1l.65.65 1.2-1.2" />
      <path d="M11.65 8.75h3.85M11.65 12.45h3.85M11.65 16.15h2.4" />
    </IconFrame>
  );
}

/** 定额：被四角卡尺包围的标准单元，内部十字表示按单位拆分含量。 */
function QuotaIcon(props: CostTierIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M4.9 4.1v2.7M4.9 5.45h2.7M19.1 4.1v2.7M16.4 5.45h2.7M4.9 19.9v-2.7M4.9 18.55h2.7M19.1 19.9v-2.7M16.4 18.55h2.7" />
      <rect x="7.25" y="7.25" width="9.5" height="9.5" rx="2.1" />
      <path d="M12 7.25v9.5M7.25 12h9.5" />
      <circle cx="12" cy="12" r="1.15" fill="currentColor" stroke="none" />
    </IconFrame>
  );
}

/** 工料机：人物、砖料与齿轮组成的资源组，底座把它们归为同一类主数据。 */
function ResourceIcon(props: CostTierIconProps) {
  return (
    <IconFrame {...props}>
      <circle cx="6.7" cy="9.15" r="1.45" />
      <path d="M4.45 17.15v-2.05c0-1.85 1-2.95 2.25-2.95s2.25 1.1 2.25 2.95v2.05" />
      <path d="M9.75 12.8h4.15v2.15H9.75zM10.65 14.95h4.15v2.2h-4.15z" />
      <circle cx="17.85" cy="13.65" r="2" />
      <path d="M17.85 10.65v1.05M17.85 15.6v1.05M14.85 13.65h1.05M19.8 13.65h1.05" />
      <path d="M3.75 19.65h16.5" />
    </IconFrame>
  );
}

export function CostTierIcon({ tier, ...props }: CostTierIconProps) {
  if (tier === "bill") return <BillIcon tier={tier} {...props} />;
  if (tier === "quota") return <QuotaIcon tier={tier} {...props} />;
  return <ResourceIcon tier={tier} {...props} />;
}
