/* eslint-disable react-refresh/only-export-components -- CompactContext 与 useCompact hook 同文件（Provider 模式），非组件文件 */
import { createContext, useContext } from "react";

const CompactContext = createContext(false);
export function useCompact() { return useContext(CompactContext); }
export default CompactContext;
