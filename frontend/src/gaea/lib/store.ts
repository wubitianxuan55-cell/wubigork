// Zustand store — replaces useController's useReducer state machine.
// P4 结构刀：本文件瘦身为纯 re-export 聚合入口。按 store 域拆分见 ./store/
//（controller：主 reducer 状态机 / preview：预览队列 / commonts：小型 UI store），
// 全部消费方 `import ... from "../gaea/lib/store"` 与导出名零改动。

export * from "./store/controller";
export * from "./store/preview";
export * from "./store/commonts";

