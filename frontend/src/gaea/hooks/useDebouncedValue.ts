import { useEffect, useState } from "react";

/**
 * 高频输入防抖：输入值仍即时更新（受控输入语义），仅下游过滤/搜索消费防抖后的值。
 * - delay 毫秒内连续变化只保留最后一次；
 * - 空串/空值即时同步（清空搜索框时立即生效，不残留旧结果）；
 * - 卸载时清理定时器。
 */
export function useDebouncedValue<T>(value: T, delay = 250): T {
  const [debounced, setDebounced] = useState<T>(value);

  useEffect(() => {
    if (value === "" || value == null) {
      // 即时清空：空串/空值不等待 delay
      setDebounced(value);
      return;
    }
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debounced;
}
