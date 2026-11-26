// useTaskCards — 子代理 task 卡 live 预览/跳转/歧义选择器的注入点接线。
// 依赖 subRunsCacheRef（useSubagentTabs 共享快照）与 openSubagentAside；
// 单一 useEffect 按原 deps/执行时机注册，卸载时清空全部注入点。
import { useEffect } from "react";
import {
  matchRunningCandidates,
  matchRunningRun,
  setTaskCardActivityProvider,
  setTaskCardAmbiguityHandler,
  setTaskCardAmbiguityResolver,
  setTaskCardOpenHandler,
  setTaskCardOpenTarget,
} from "../lib/taskActivity";
import type { SubagentRunView } from "../lib/types";
import type { MutableRefObject } from "react";

interface UseTaskCardsParams {
  subRunsCacheRef: MutableRefObject<SubagentRunView[]>;
  setTaskPickCandidates: (v: SubagentRunView[] | null) => void;
  currentSessionPath: string | undefined;
  // 2026-11-18 对齐右栏：task 卡/活行动/歧义选择器点击一律开右栏子会话
  // pane（编队卡 e665c777 同管道）——主对话不被切走；主区会话 Tab 链路
  // （openSubagentThread）保留给目录树/面包屑等既有消费方。
  openSubagentAside: (p: { ref: string; task?: string }) => void;
}

export function useTaskCards({ subRunsCacheRef, setTaskPickCandidates, currentSessionPath, openSubagentAside }: UseTaskCardsParams) {
  // ② 子代理 task 卡 live 预览——运行期间 5s 轮询 GaeaSubagentRuns 喂
  //    taskActivity 注入点；派发期 args 不带 ref（ref 只在 tool_result 出现）。
  //    空 ref 回退：单个 running 直接绑定（原逻辑不变）；并行多个 running 时
  //    用 ToolCard 透传的 args 任务描述文本与各 run.task 做唯一命中匹配——
  //    0 或 ≥2 命中都返回 undefined（宁缺勿错，绝不把别的子代理动态安到
  //    错误卡片上）（taskActivity 头注释契约）。
  //    注：setTaskPickCandidates/subRunsCacheRef 为 App 侧稳定引用（dispatch/ref），
  //    原 deps [currentSessionPath, openSubagentThread] 语义保留（下方 disable 同因）。
  useEffect(() => {
    setTaskCardActivityProvider((ref, args) => {
      const runs = subRunsCacheRef.current;
      const pick = (r: SubagentRunView) =>
        r ? { lastText: r.lastText, lastTool: r.lastTool, state: r.status } : undefined;
      if (ref) {
        const hit = runs.find((r) => r.ref === ref);
        return hit ? pick(hit) : undefined;
      }
      const runningRuns = runs.filter((r) => r.status === "running");
      if (runningRuns.length !== 1) {
        // 并行多子代理同时 running：文本唯一命中才绑定，其余情形维持现状（undefined）
        const m = matchRunningRun(args, runningRuns);
        return m ? pick(m) : undefined;
      }
      return pick(runningRuns[0]);
    });
    // v4.63：task/run_skill 卡整卡可点 → 打开对应子代理会话（2026-11-18 起
    // 开右栏 pane，与编队卡同管道；v4.63~v4.68 为主区会话 tab）。目标解析：
    // args/output 已带 ref 直接放行；空 ref 回退「唯一 running 命中」（宁缺
    // 勿错，历史完成卡无 ref 不可点，如实）。pane 标题由 runs 缓存预填，
    // 打开后 5s 轮询自校正。
    setTaskCardOpenTarget((ref, args) => {
      if (ref) return ref;
      // 2026-11-18 实弹修复：后台委派完成后卡也要可点——打开目标在**全部
      // 运行**里唯一文本命中（旧逻辑只匹配 running，后台完成卡永不可点）。
      return matchRunningRun(args, subRunsCacheRef.current)?.ref ?? "";
    });
    // v4.68：空 ref 多候选（≥2 文本匹配 running）时 matchRunningRun 按宁缺
    // 勿错返回 ""，卡此前不可点、用户没有入口。歧义两槽位补上：渲染期判定
    // （仅 ≥2 候选为真，0/1 候选维持现状）+ 点击弹选择器人工挑（候选在点击
    // 瞬间现算，数据最新鲜；0 候选不弹，绝不自动跳转）。
    setTaskCardAmbiguityResolver((ref, args) => {
      if (ref) return false; // 已有唯一目标：不歧义
      return matchRunningCandidates(args, subRunsCacheRef.current).length > 1;
    });
    setTaskCardAmbiguityHandler((args) => {
      const cands = matchRunningCandidates(args, subRunsCacheRef.current);
      if (cands.length === 0) return; // 点击瞬间已无候选：维持现状，不弹空壳
      setTaskPickCandidates(cands);
    });
    setTaskCardOpenHandler((ref) => {
      const hit = subRunsCacheRef.current.find((r) => r.ref === ref);
      openSubagentAside({ ref, task: hit?.task });
    });
    return () => {
      setTaskCardActivityProvider(null);
      setTaskCardOpenTarget(null);
      setTaskCardOpenHandler(null);
      setTaskCardAmbiguityResolver(null);
      setTaskCardAmbiguityHandler(null);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setTaskPickCandidates/subRunsCacheRef 稳定引用，deps 如上注释
  }, [currentSessionPath, openSubagentAside]);
}
