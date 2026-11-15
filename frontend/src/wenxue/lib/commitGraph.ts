// commitGraph.ts — G2 只读提交图 lane 拓扑纯函数（与渲染解耦，单测钉拓扑）。
//
// 算法（经典 git log --graph 变体）：lanes[i] 持「待连接的父 hash」——某提交
// 出现时落在等待它的 lane 上（先清位再布父：首父默认继承提交所在 lane 继续
// 直线，其余父开新 lane 外扩；父已在别的 lane → 折线并入）。孤儿/截断历史的
// 父不进清单，through 标记其 lane 仍向下穿行（渲染层画到图底）。lane 数上限
// MAX_LANES（病理拓扑钳到末 lane，线可能交叉但渲染有界、不崩）。
//
// 本文件零依赖零 DOM：commitLanes 输入就是 GitGraphNodeView 的 hash/parents
// 子集，输出逐行 lane+连线+穿行集合，由提交图镜头的 SVG 直接消费。

export const MAX_LANES = 8;

export interface GraphCommit {
  hash: string;
  parents: string[];
}

export interface GraphEdge {
  from: number;
  to: number;
}

export interface GraphRow {
  /** 本行提交所在 lane（提交圆点画这里）。 */
  lane: number;
  /** 到各父的连线（from=本行 dot lane，to=父 lane；渲染画本行下半段，下一行上半段由 through/承接边覆盖）。 */
  edges: GraphEdge[];
  /** 处理本行后仍待连接的 lane 集合（行下方继续画竖线，画到图底或被消费）。 */
  through: number[];
}

export function commitLanes(commits: GraphCommit[]): GraphRow[] {
  const lanes: Array<string | null> = [];
  const rows: GraphRow[] = [];

  const laneOf = (h: string): number => lanes.indexOf(h);
  const freeLane = (): number => {
    const i = lanes.findIndex((v) => v === null);
    if (i >= 0) return i;
    if (lanes.length < MAX_LANES) {
      lanes.push(null);
      return lanes.length - 1;
    }
    return MAX_LANES - 1; // 病理拓扑钳到末 lane：有界不崩
  };

  for (const c of commits) {
    let lane = laneOf(c.hash);
    if (lane < 0) {
      lane = freeLane(); // 新根/未被任何子预占位的提交
    }
    lanes[lane] = null; // 先消费本位，再布父（首父继承时重新占用同槽）
    const edges: GraphEdge[] = [];
    c.parents.forEach((p, idx) => {
      let pl = laneOf(p);
      if (pl < 0) {
        pl = idx === 0 ? lane : freeLane();
        lanes[pl] = p;
      }
      edges.push({ from: lane, to: pl });
    });
    const through: number[] = [];
    lanes.forEach((v, i) => {
      if (v !== null) through.push(i);
    });
    rows.push({ lane, edges, through });
  }
  return rows;
}
