// mock/imagehub.ts — 图像域登记视图（T1 画室素材库走查样例）。
// ImageHubAssets 只读绑定：按空间/来源过滤静态样例；登记是辅助视图，
// mock 样例覆盖 生图/图示/视频 三形态与「模型/成本诚实留空」口径。
// 缩略 dataURL 复用 office.ts 的 AttachmentDataURL 占位色块。
import type { AppBindings } from "../bridge";

type ImageHubMethods = Pick<AppBindings, "ImageHubAssets" | "ChapterArtList">;

// type alias（非 interface）：MockAsset[] 要赋给 Record<string, unknown>[] 视图。
type MockAsset = {
  id: string;
  kind: string;
  path: string;
  mime?: string;
  space: string;
  source_board: string;
  capability: string;
  backend?: string;
  model?: string;
  cost?: string;
  created_at: string;
  prompt_truncate?: string;
  params?: Record<string, unknown>;
}

const ASSETS: MockAsset[] = [
  {
    id: "mock-a1",
    kind: "image",
    path: ".gaea/uploads/mock-cover.png",
    mime: "image/png",
    space: "work",
    source_board: "imagegen",
    capability: "media.generate",
    backend: "herdsman",
    model: "z-image-turbo",
    cost: "0.12",
    created_at: "2026-09-05T10:12:00+08:00",
    prompt_truncate: "赛博朋克城市夜景，霓虹反射，电影感构图",
    params: { character_id: "mara" },
  },
  {
    id: "mock-a2",
    kind: "image",
    path: ".gaea/uploads/mock-chapter.png",
    mime: "image/png",
    space: "work",
    source_board: "novel",
    capability: "media.generate",
    backend: "comfyui",
    model: "flux-dev",
    cost: "0.34",
    created_at: "2026-09-05T09:40:00+08:00",
    prompt_truncate: "雨夜站台，少女回望，胶片颗粒",
  },
  {
    id: "mock-a3",
    kind: "diagram",
    path: ".gaea/uploads/diagram-1757040000.mmd",
    space: "work",
    source_board: "imagegen",
    capability: "media.diagram",
    created_at: "2026-09-05T11:05:00+08:00",
    prompt_truncate: "登录鉴权时序图",
  },
  {
    id: "mock-a4",
    kind: "video",
    path: ".gaea/uploads/mock-clip.mp4",
    space: "play",
    source_board: "novel",
    capability: "media.generate",
    backend: "herdsman",
    model: "wan2.5",
    cost: "0.85",
    created_at: "2026-09-04T21:30:00+08:00",
    prompt_truncate: "片头运镜：从云层俯冲入山谷",
  },
  // T1 创作资产「近期作品」扩样例：work/play 各补一张 imagegen 生图登记。
  {
    id: "mock-a5",
    kind: "image",
    path: ".gaea/uploads/mock-poster.png",
    mime: "image/png",
    space: "work",
    source_board: "imagegen",
    capability: "media.generate",
    backend: "xai",
    model: "grok-imagine-image",
    cost: "0.05",
    created_at: "2026-09-05T12:02:00+08:00",
    prompt_truncate: "复古旅行海报，山间铁路，晨雾",
  },
  {
    id: "mock-a6",
    kind: "image",
    path: ".gaea/uploads/mock-avatar.png",
    mime: "image/png",
    space: "play",
    source_board: "imagegen",
    capability: "media.generate",
    backend: "herdsman",
    model: "krea2",
    cost: "0.02",
    created_at: "2026-09-05T13:20:00+08:00",
    prompt_truncate: "少女头像，银发红瞳，柔和顶光",
    params: { character_id: "mara" },
  },
];

export function buildImagehub(): ImageHubMethods {
  return {
    async ImageHubAssets(space: string, sourceBoard: string, limit: number) {
      const list = ASSETS.filter(
        (a) => a.space === space && (!sourceBoard || a.source_board === sourceBoard),
      );
      return list.slice(0, Math.max(0, limit));
    },
    async ChapterArtList(_chapterNum: number) {
      // 项目级按章清单语义，走查保持中性空态（各章历史配图属真数据域）。
      return [];
    },
  };
}
