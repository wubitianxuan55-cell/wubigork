// image.ts — 图片生成 API 层，封装 Go 后端 bridge 调用
import { app } from "./bridge";
import type { ImageGenResponse, ComfyUIStatus } from "./types";

/** 生成自由图片 */
export async function generateFreeImage(
  prompt: string,
  negative: string,
  size: string,
  model: string,
  seed: number,
  n: number,
): Promise<ImageGenResponse> {
  return app.GenerateFreeImage(prompt, negative, size, model, seed, n);
}

/** 获取 ComfyUI 运行状态 */
export async function getComfyUIStatus(): Promise<ComfyUIStatus> {
  return app.GetComfyUIStatus();
}

/** 启动 ComfyUI 服务 */
export async function startComfyUI(): Promise<void> {
  return app.StartComfyUI();
}

/** 停止 ComfyUI 服务 */
export async function stopComfyUI(): Promise<void> {
  return app.StopComfyUI();
}

/** 保存 ComfyUI 配置 */
export async function saveComfyUIConfig(
  comfyUIURL: string,
  imageModel: string,
  comfyUIPath: string,
  comfyUIPythonPath: string,
): Promise<void> {
  return app.SaveComfyUIConfig(comfyUIURL, imageModel, comfyUIPath, comfyUIPythonPath);
}

/** 获取 ComfyUI 完整配置 */
export async function getComfyUIConfig(): Promise<{url: string; model: string; path: string; pythonPath: string}> {
  return app.GetComfyUIConfig();
}

/** 持久化图片生成结果到历史 */
export async function saveImageResults(
  results: Record<string, unknown>[],
): Promise<void> {
  return app.SaveImageResults(results);
}

/** 加载图片生成历史 */
export async function loadImageResults(): Promise<Record<string, unknown>[]> {
  return app.LoadImageResults();
}

/** 按索引删除历史中的某条图片记录 */
export async function deleteImageResult(index: number): Promise<void> {
  return app.DeleteImageResult(index);
}

/** 打开图片历史所在文件夹 */
export async function openImageHistoryDir(): Promise<void> {
  return app.OpenImageHistoryDir();
}
