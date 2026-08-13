/** 滚动跟随工具：判断滚动容器是否接近底部，用于“用户上翻时不强制吸底”。 */

export function isNearBottom(distanceFromBottom: number, threshold = 80): boolean {
  return distanceFromBottom < threshold;
}
