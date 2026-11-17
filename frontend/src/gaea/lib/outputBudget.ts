import raw from "../../../../internal/gaea/strutil/output_budget.json";

export interface OutputBudget {
  schemaVersion: number;
  transport: {
    byteLimit: number;
    headRatio: number;
    serializationAllowance: number;
    agentPreviewByteLimit: number;
    repeatArgumentPreviewBytes: number;
    archiveContentByteLimit: number;
    controllerSummaryRunes: number;
  };
  display: {
    genericPreviewLines: number;
    toolCardPreviewLines: number;
    headLines: number;
    tailLines: number;
    lineWidth: number;
    byteLimit: number;
    maxRenderedRows: number;
  };
  frontend: {
    highlightByteLimit: number;
    highlightLineLimit: number;
    highlightLineByteLimit: number;
    genuiFragmentByteLimit: number;
    genuiPlaceholderOrigin: string;
  };
}

export const OUTPUT_BUDGET = raw as OutputBudget;

export const TOOL_OUTPUT_MAX_PREVIEW_LINES = OUTPUT_BUDGET.display.genericPreviewLines;
export const TOOL_CARD_OUTPUT_PREVIEW_LINES = OUTPUT_BUDGET.display.toolCardPreviewLines;
export const TOOL_OUTPUT_LINE_WIDTH = OUTPUT_BUDGET.display.lineWidth;
export const TOOL_OUTPUT_PREVIEW_BYTES = OUTPUT_BUDGET.display.byteLimit;
export const MAX_RENDERED_ROWS = OUTPUT_BUDGET.display.maxRenderedRows;
