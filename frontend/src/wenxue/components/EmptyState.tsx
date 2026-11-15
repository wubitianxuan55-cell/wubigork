import { memo } from "react";
import V3Empty from "../../components/V3Empty";

/** 空态统一（v4.261）：轨道环视觉收敛到全局 V3Empty（原 13px 灰字形态）。
 *  API 保持 {message} 不变，消费方（WhisperGraphPanel / sin StoryStream）零改动。 */
export const EmptyState = memo(function EmptyState(p: { message: string }) {
  return (
    <div className="py-14">
      <V3Empty description={p.message} />
    </div>
  );
});
