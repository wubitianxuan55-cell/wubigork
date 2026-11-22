// CapabilitiesPanel（T6-10.1 巨型组件拆分后的编排层，行为零变化）
// 职责：抽屉壳（ResizableDrawer+DrawerHeader）；三 tab 主体（MCP 服务器/工具/
// 技能）已随设置中枢批 4 拆至 capabilities/CapabilitiesInline 单源——本壳与
// 设置页 office 组共用同一主体，summary+reload 工具行在主体顶部（原头部
// reload 按钮随主体走，交互不变）。
//
// CapabilitiesPanel is the desktop MCP & Skills drawer — the GUI counterpart to
// the CLI's /mcp + /skill, aligning with Claude Code's Customize → Connectors:
// each server shows a connected/failed dot, transport, and tool/prompt/resource
// counts, with add / remove / retry; skills list their scope and run mode.
import { ResizableDrawer } from "./ResizableDrawer";
import { DrawerHeader } from "./DrawerHeader";
import { Blocks } from "../icons";
import { useT } from "../lib/i18n";
import { CapabilitiesInline } from "./capabilities/CapabilitiesInline";

export function CapabilitiesPanel({
  onClose,
  toolCounts = {},
  skillCounts = {},
}: {
  onClose: () => void;
  toolCounts?: Record<string, number>;
  skillCounts?: Record<string, number>;
}) {
  const t = useT();
  return (
    <ResizableDrawer onClose={onClose} subtle label={t("caps.title")}>
      <DrawerHeader onClose={onClose}>
        <div className="caps-title-lockup">
          <span className="caps-title-lockup__mark" aria-hidden="true">
            <Blocks size={15} />
          </span>
          <div className="caps-title-lockup__copy">
            <span className="caps-title-lockup__title">{t("caps.title")}</span>
            <span className="caps-title-lockup__hint">连接器 · 工具目录 · 工作流</span>
          </div>
        </div>
      </DrawerHeader>
      <CapabilitiesInline toolCounts={toolCounts} skillCounts={skillCounts} variant="drawer" />
    </ResizableDrawer>
  );
}
