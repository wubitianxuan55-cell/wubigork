// 精确匹配生成模型 → 别名（单一事实源 wailsjs/go/models.ts，scripts/check-types-drift.mjs --alias-exact 生成）

// WireShape 把 wails 生成类的实例形状剥成纯线格式数据：生成类含实例方法
// convertValues（构造时递归实例化嵌套对象），不属于 JSON 线协议字段；递归映射
// 同时处理嵌套类属性。别名统一用 WireShape<AppModels.X>，消费方拿到的仍是
// 与旧手写 interface 一致的结构类型。
export type WireShape<T> = T extends (...args: never[]) => unknown
  ? never
  : T extends (infer U)[]
    ? WireShape<U>[]
    : T extends object
      ? { [K in keyof T as K extends "convertValues" ? never : K]: WireShape<T[K]> }
      : T;
