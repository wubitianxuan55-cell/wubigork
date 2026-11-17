export type EscOwner =
  | "warnings"
  | "questions"
  | "modal"
  | "preview"
  | "overlay"
  | "banner"
  | "composer"
  | "stop"
  | "backtrack"
  | null;

export interface EscContext {
  warningsOpen?: boolean;
  questionsOpen?: boolean;
  modalOpen?: boolean;
  previewOpen?: boolean;
  overlayOpen?: boolean;
  bannerOpen?: boolean;
  inInput?: boolean;
  running?: boolean;
  composerEmpty?: boolean;
}

/** Single Esc arbitration table shared by both shells. */
export function escOwner(ctx: EscContext): EscOwner {
  if (ctx.warningsOpen) return "warnings";
  if (ctx.questionsOpen) return "questions";
  if (ctx.modalOpen) return "modal";
  if (ctx.previewOpen) return "preview";
  if (ctx.overlayOpen) return "overlay";
  if (ctx.bannerOpen) return "banner";
  if (ctx.inInput) return "composer";
  if (ctx.running) return "stop";
  if (ctx.composerEmpty) return "backtrack";
  return "composer";
}
