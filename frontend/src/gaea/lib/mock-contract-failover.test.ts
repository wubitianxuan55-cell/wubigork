// B 线欠账「dev mock 补 Get/SetEngineFailover」契约层 mock 冒烟测试：
// 模型中心「调度」段故障转移开关（SchedulingSection → api/engines
// getEngineFailover/setEngineFailover → Go ModelB 门面 GetEngineFailover/
// SetEngineFailover，同名 legacy 绑定面）。锁定 mock 契约：Get 初值 false
// （默认关，对齐后端默认值）、Set 写回、Get 回读，以及会话内重读一致
// （「重进页面保持」在 mock 单例下的语义）。
//
// 说明：两名不在 AppBindings 联合里（bridge.ts LegacySurfaceNames 显式排除，
// AppBindings 认领待后端合入后收口），故这里经 bridge 单例 app 以窄化视图
// 同名调用——与 window.go.app.App 兼容代理的路由方法名一致。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

/** legacy 绑定面窄化视图：仅暴露故障转移两名（mock 实现见 mock/model.ts）。 */
const failover = app as unknown as {
  GetEngineFailover(): Promise<boolean>;
  SetEngineFailover(enabled: boolean): Promise<void>;
};

describe("mock 契约 · 故障转移开关（Get/SetEngineFailover）", () => {
  it("GetEngineFailover 存在于 mock 绑定面（契约：可调用、不 undefined）", () => {
    expect(typeof failover.GetEngineFailover).toBe("function");
    expect(typeof failover.SetEngineFailover).toBe("function");
  });

  it("Get 初值 false（默认关，对齐后端默认值）", async () => {
    await expect(failover.GetEngineFailover()).resolves.toBe(false);
  });

  it("Set 写回 → Get 回读：开再关，布尔往返一致", async () => {
    await failover.SetEngineFailover(true);
    await expect(failover.GetEngineFailover()).resolves.toBe(true);
    await failover.SetEngineFailover(false);
    await expect(failover.GetEngineFailover()).resolves.toBe(false);
  });

  it("重进页面（重新 Get）仍返回最近写入值（会话内保持）", async () => {
    await failover.SetEngineFailover(true);
    // 模拟重进页面：组件重挂载后重新 load()，mock 单例状态仍在 → 读回 true
    await expect(failover.GetEngineFailover()).resolves.toBe(true);
    await failover.SetEngineFailover(false); // 还原默认，避免污染同文件后续读取
    await expect(failover.GetEngineFailover()).resolves.toBe(false);
  });
});
