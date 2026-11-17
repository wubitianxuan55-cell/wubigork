import type { AppBindings } from "./bridge";

/** Composer 在两壳之间的最小绑定面；避免要求文学壳实现办公专属方法。 */
export type ComposerBridge = Pick<
  AppBindings,
  | "Commands"
  | "SlashArgs"
  | "GaeaListDirBatch"
  | "ListDir"
  | "FileSearch"
  | "ListWorkspaces"
  | "AttachmentChunkBegin"
  | "AttachmentChunkAppend"
  | "AttachmentChunkFinish"
  | "AttachmentChunkAbort"
  | "SavePastedImage"
  | "SaveAttachmentFile"
  | "PickFiles"
  | "CaptureScreen"
  | "RecognizeImage"
  | "OCRText"
  | "SkillDraftFromSession"
  | "SkillDraftSave"
>;
