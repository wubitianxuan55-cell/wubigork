import { Composer as SharedComposer, type ComposerProps } from "../../gaea/components/Composer";
import { app } from "../lib/bridge";
import { isDisabledWenxueCommand } from "../lib/disabledDomains";
import { useComposerInsertStore } from "../lib/store";

type WenxueComposerProps = Omit<ComposerProps, "variant" | "bridge" | "insertStore" | "filterCommand">;

const keepWenxueCommand = (name: string) => !isDisabledWenxueCommand(name);

export function Composer(props: WenxueComposerProps) {
  return (
    <SharedComposer
      {...props}
      variant="wenxue"
      bridge={app}
      insertStore={useComposerInsertStore}
      filterCommand={keepWenxueCommand}
    />
  );
}
