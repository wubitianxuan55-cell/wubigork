import { Composer as SharedComposer, type ComposerProps } from "../../gaea/components/Composer";
import { BoardPortsContext } from "../../gaea/components/boardPorts";
import { app } from "../lib/bridge";
import { wenxueBoardPorts } from "../lib/boardPorts";
import { isDisabledWenxueCommand } from "../lib/disabledDomains";
import { useComposerInsertStore } from "../lib/store";

type WenxueComposerProps = Omit<ComposerProps, "variant" | "bridge" | "insertStore" | "filterCommand">;

const keepWenxueCommand = (name: string) => !isDisabledWenxueCommand(name);

export function Composer(props: WenxueComposerProps) {
  return (
    // 板块读口缝（2026-11-26）：共享 Composer 的附件条点开预览须经缝落到文学自己的
    // 预览库（见 lib/boardPorts.ts）。
    <BoardPortsContext.Provider value={wenxueBoardPorts}>
      <SharedComposer
        {...props}
        variant="wenxue"
        bridge={app}
        insertStore={useComposerInsertStore}
        filterCommand={keepWenxueCommand}
      />
    </BoardPortsContext.Provider>
  );
}
