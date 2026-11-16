import { createBooleanPref } from "./booleanPref";

// 文学首页隐私偏好。开启后只隐藏首页上的人可读内容（书名、会话摘要、
// 最新章节），不改文件、不阻断工作流；完整路径在任何模式下都不直接展示。
export const wenxueHomePrivacyPref = createBooleanPref(
  "gaea.wenxueHomePrivacy",
  false,
);
