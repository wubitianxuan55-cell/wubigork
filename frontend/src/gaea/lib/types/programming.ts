// ── 编程板块：DeepSeek Harness Web 进程管理 ─────────────────────
// GetProgrammingWebStatus 的返回契约：running=3080 端口在服务；
// owned=该实例由 gaea 自启（StopProgrammingWeb 只杀 owned 实例，不误杀外部进程）；
// uptime_s=自启实例已运行秒数（非自启/未运行恒 0）；log=自启日志路径。
export interface ProgrammingWebStatus {
  running: boolean;
  owned: boolean;
  pid: number;
  url: string; // Web UI 地址（默认 http://127.0.0.1:3080）
  root: string; // harness 仓库根目录
  log: string; // gaea 自启 dsh web 日志路径
  uptime_s: number; // 自启实例已运行秒数（非自启/未运行恒 0）
}

// GetProgrammingWebPreflight 的返回契约：启动前置条件逐项检查
// （harness 目录有效 / pnpm 可用 / 依赖已装 / Web 构建产物就绪 / 端口空闲），
// 启动引导视图据此渲染绿/红清单；all_ready 为真才建议一键启动。
export interface ProgrammingWebPreflight {
  harness_valid: boolean;
  pnpm_found: boolean;
  deps_ready: boolean;
  build_ready: boolean;
  port_free: boolean;
  all_ready: boolean;
  root: string; // 实际检查的 harness 目录
}

// ProgrammingWebLogTail 的返回契约：自启日志尾部（lines 最多 n 行，
// n 由调用方钳制 [1,200]）；日志尚未生成时 exists=false + error 提示。
export interface ProgrammingWebLogTail {
  exists: boolean;
  path: string;
  lines: string[];
  error: string;
}
