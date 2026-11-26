import { memo } from "react";
import {
  ToolCard as SharedToolCard,
  type ToolCardProps,
} from "../../gaea/components/ToolCard";
import { BoardPortsContext } from "../../gaea/components/boardPorts";
import { wenxueBoardPorts } from "../lib/boardPorts";
import { isDisabledWenxueTool } from "../lib/disabledDomains";

const toolVisible = (name: string) => !isDisabledWenxueTool(name);

// 共享的办公 ToolCard 默认读办公那侧的任务卡注入点/文件打开入口（模块单例）；
// 这里把本树的读口经 Context 交给它（见 lib/boardPorts.ts）。
export const ToolCard = memo(function WenxueToolCard(props: ToolCardProps) {
  return (
    <BoardPortsContext.Provider value={wenxueBoardPorts}>
      <SharedToolCard {...props} isToolVisible={toolVisible} />
    </BoardPortsContext.Provider>
  );
});
