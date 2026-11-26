/* eslint-disable react-refresh/only-export-components -- 转发办公 Toast（Provider 组件 + useToast hook 同模块，Context 模块惯例） */
// Toast — 轻提示 Provider + useToast。板块无关：直接转发办公单源组件，与本树复用的
// 办公组件（Composer 等）共用同一个 Context。此前这里是逐字拷贝、各自 createContext，
// 办公组件在文学界面里拿到的是缺省空实现，提示被静默丢弃。2026-11-26 同步审计改转发。
export * from "../../gaea/components/Toast";
