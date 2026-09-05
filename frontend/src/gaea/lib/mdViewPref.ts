// mdViewPref.ts — markdown 双视图（文档/导图）偏好（M1）。
// localStorage 记忆用户在 FilePreview/FilePreviewModal 的视图选择，默认文档视图
// （既有行为不变）。独立于组件文件存放，避免 react-refresh 混合导出告警。

export const MD_VIEW_KEY = "gaea.preview.mdView";

export type MdViewMode = "doc" | "mindmap";

export function readMdViewPref(): MdViewMode {
  try {
    return localStorage.getItem(MD_VIEW_KEY) === "mindmap" ? "mindmap" : "doc";
  } catch {
    return "doc";
  }
}

export function writeMdViewPref(v: MdViewMode): void {
  try {
    localStorage.setItem(MD_VIEW_KEY, v);
  } catch {
    // 偏好写失败静默（隐私模式）；不影响当次会话的视图状态
  }
}
