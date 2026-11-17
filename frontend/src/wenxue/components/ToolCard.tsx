import { memo } from "react";
import {
  ToolCard as SharedToolCard,
  type ToolCardProps,
} from "../../gaea/components/ToolCard";
import { isDisabledWenxueTool } from "../lib/disabledDomains";

const toolVisible = (name: string) => !isDisabledWenxueTool(name);

export const ToolCard = memo(function WenxueToolCard(props: ToolCardProps) {
  return <SharedToolCard {...props} isToolVisible={toolVisible} />;
});
