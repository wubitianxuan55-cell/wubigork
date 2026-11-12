// mock/office/methods_ops.ts — office 运维/任务侧四块（批 31 FE3-15 续刀，自
// methods_office.ts 原位搬移，零逻辑改动）：Git 演示族（GaeaGit*）、任务族
// （TaskList/Cancel/Kill/Retry/Output，taskMock+监听器）、证据链（GaeaJournal/
// Verify/Rollback，与 Preview 的 MOCK_XLSX_BODY 声明样本对齐）+ WriteFile、
// 数据备份族（DataBackup*，批次三a legacy 直调转正）。
// 组合侧 methods_office.ts 以 `...opsMethods` 原位展开（键序不变）。
import { mockTaskListeners, taskMock } from "../shared";
import { mockFileBodies } from "./state";
import type { OfficeMethods } from "./types";

export const opsMethods: Pick<
  OfficeMethods,
  | "GaeaGitStatus"
  | "GaeaGitDiff"
  | "GaeaGitStage"
  | "GaeaGitUnstage"
  | "GaeaGitDiscard"
  | "GaeaGitCommit"
  | "GaeaGitLog"
  | "GaeaGitWorktrees"
  | "GaeaGitGraph"
  | "GaeaGitCommitDetail"
  | "TaskList"
  | "TaskCancel"
  | "TaskKill"
  | "TaskRetry"
  | "TaskOutput"
  | "GaeaJournalList"
  | "VerifyRecord"
  | "RollbackRecord"
  | "WriteFile"
  | "DataBackupInfo"
  | "DataBackupCreate"
  | "DataBackupRestore"
  | "DataBackupCancel"
  | "DataBackupRollback"
  | "DataBackupRestoreResult"
> = {
  async GaeaGitStatus() {
    // 2b 走查样例：三分组状态（暂存/未暂存/未跟踪）+ 分支 ahead + linked
    // worktree 标注（G1 深化，与 GaeaGitWorktrees 样例同源）。
    return {
      isRepo: true,
      branch: "feature/x",
      ahead: 1,
      behind: 0,
      files: [
        { path: "docs/调研结论.md", x: "M", y: " ", staged: true, modified: true },
        { path: "internal/gaea/config/config.go", x: " ", y: "M", modified: true },
        { path: "web/app.ts", x: " ", y: "M", modified: true },
        { path: "reports/季度报告.html", x: "?", y: "?", untracked: true },
        // G1.7 走查样例：冲突独立分组 + rename 溯源。
        { path: "src/合并冲突.txt", x: "U", y: "U", conflict: true },
        { path: "docs/改名后.md", x: "R", y: " ", staged: true, renamed: true, origPath: "docs/改名前.md" },
      ],
      isLinkedWorktree: true,
      mainWorktreePath: "C:\\demo\\gaea-workspace",
    };
  },
  async GaeaGitWorktrees() {
    // G1 深化走查样例：主检出 + 当前 linked worktree + 一条 detached。
    return [
      { path: "C:\\demo\\gaea-workspace", head: "1a2b3c4", branch: "main", isMain: true, isCurrent: false },
      {
        path: "C:\\demo\\gaea-workspace\\.worktrees\\feature-x",
        head: "5d6e7f8",
        branch: "feature/x",
        isCurrent: true,
      },
      { path: "C:\\demo\\gaea-workspace\\.worktrees\\experiment", head: "9f8e7d6", isDetached: true },
    ];
  },
  async GaeaGitGraph(skip: number, limit: number, branch: string) {
    // G2 走查样例：主干 + 分支合并拓扑（lane 拓扑由前端纯函数算）。
    void skip;
    void limit;
    void branch;
    const H = {
      m: "f91a25fc7896a79520eaa8d66e82f65e27e5e322",
      mg: "cc2a8ef80af9287d2c6e9c38c0b067d7a4dfac4f",
      main: "e7be2aeaeb785f10562313deac2a282f7056e30e",
      feat: "19b8c10a98b9e93b19e7ef14f3f3ea29667d9965",
      root: "501680bcb5db8196c18c2a75c9be699e53d6f02a",
      base: "1f2ea3a382177df60b9e9f636ce40884291c70dc",
    };
    return {
      commits: [
        { hash: H.m, short: "f91a25f", subject: "发布 v0.32.0", author: "walk", ts: 1794502086, parents: [H.mg], refs: ["HEAD -> main"] },
        { hash: H.mg, short: "cc2a8ef", subject: "merge feature/workbench", author: "walk", ts: 1794502000, parents: [H.main, H.feat] },
        { hash: H.main, short: "e7be2ae", subject: "workbench: 上传分块续传", author: "walk", ts: 1794501900, parents: [H.root] },
        { hash: H.feat, short: "19b8c10", subject: "feat: worktree 只读展示", author: "walk", ts: 1794501910, parents: [H.base], refs: ["origin/feature"] },
        { hash: H.root, short: "501680b", subject: "init", author: "walk", ts: 1794501800, parents: [], refs: ["tag: v0.1.0"] },
        { hash: H.base, short: "1f2ea3a", subject: "chore: 基线", author: "walk", ts: 1794501850, parents: [H.root] },
      ],
      branches: ["main", "feature", "wt/feature-x"],
    };
  },
  async GaeaGitCommitDetail(hash: string) {
    // G2 走查样例：与 GaeaGitGraph 样例同源；含普通文件与二进制文件。
    void hash;
    return {
      hash,
      subject: "merge feature/workbench",
      body: "合并 workbench 分块上传与 worktree 展示两线。",
      author: "walk",
      ts: 1794502000,
      parents: ["e7be2aeaeb785f10562313deac2a282f7056e30e", "19b8c10a98b9e93b19e7ef14f3f3ea29667d9965"],
      refs: [],
      files: [
        { path: "internal/gaea/workbench/fs.go", add: 120, del: 4 },
        { path: "assets/logo.png", add: -1, del: -1 },
      ],
      diff: [
        "diff --git a/internal/gaea/workbench/fs.go b/internal/gaea/workbench/fs.go",
        "--- a/internal/gaea/workbench/fs.go",
        "+++ b/internal/gaea/workbench/fs.go",
        "@@ -1,2 +1,3 @@",
        " 上下文行",
        "+新增行",
      ].join("\n"),
    };
  },
  async GaeaGitDiff(path: string, staged: boolean) {
    void staged;
    return [
      "diff --git a/" + path + " b/" + path,
      "index 111111..222222 100644",
      "--- a/" + path,
      "+++ b/" + path,
      "@@ -1,14 +1,15 @@",
      " 第一行（上下文）",
      " 上下文 2",
      " 上下文 3",
      " 上下文 4",
      " 上下文 5",
      " 上下文 6",
      " 上下文 7",
      " 上下文 8",
      " 上下文 9",
      " 上下文 10",
      " 上下文 11",
      " 上下文 12",
      "-被删除的一行",
      "+新增加的一行",
      "+第二个新增行",
    ].join("\n");
  },
  async GaeaGitStage(_paths: string[]) {},
  async GaeaGitUnstage(_paths: string[]) {},
  async GaeaGitDiscard(_path: string) {},
  async GaeaGitCommit(_message: string) {
    return "abc1234";
  },
  async GaeaGitLog(_limit: number) {
    return [
      { hash: "abc1234", subject: "release: 演示提交（mock）", author: "gaea", ts: 1750000000 },
      { hash: "def5678", subject: "fix: 上一笔演示修复（mock）", author: "gaea", ts: 1749990000 },
    ];
  },
  async TaskList() {
    return [...taskMock];
  },
  async TaskCancel(id: string) {
    const t = taskMock.find((x) => x.id === id);
    if (t && (t.status === "queued" || t.status === "running")) {
      t.status = "cancelled";
      t.finishedAt = Date.now();
      mockTaskListeners.forEach((l) => l(t));
    }
  },
  async TaskKill(id: string) {
    // mock：强杀 = 立即终态（真实实现 = 协作取消 + 进程树击杀钩子）。
    const t = taskMock.find((x) => x.id === id);
    if (t && (t.status === "queued" || t.status === "running" || t.status === "stopping")) {
      t.status = "cancelled";
      t.message = "已强制终止";
      t.finishedAt = Date.now();
      mockTaskListeners.forEach((l) => l(t));
    }
  },
  async TaskRetry(id: string) {
    const t = taskMock.find((x) => x.id === id);
    if (t && (t.status === "failed" || t.status === "cancelled")) {
      t.status = "succeeded";
      t.progress = 100;
      t.error = "";
      t.finishedAt = Date.now();
      mockTaskListeners.forEach((l) => l(t));
    }
  },
  async TaskOutput(id: string) {
    // mock：对已知任务返回样例输出尾（真实实现 = tasks 环形缓冲回放）。
    const t = taskMock.find((x) => x.id === id);
    if (!t) return { tail: "", truncated: false };
    const tail = [
      `[10:00:00] 开始 ${t.label}`,
      `[10:00:01] 处理中…（进度 ${t.progress}%）`,
      t.status === "running" ? "[10:00:02] 正在抓取四川造价信息网…" : `[10:00:03] 完成（${t.error || t.message || "ok"}）`,
    ];
    return { tail: tail.join("\n"), truncated: false };
  },
  // ── v4.8 证据链（Verifier 产品化）：证据卡 / 复核 / 回滚 —— 声明样本与
  // Preview 的 MOCK_XLSX_BODY 对齐（预算!B2=120.50、B4 公式 SUM(B2:B3)），
  // 保证「声明↔实况」diff 在浏览器开发态可演示。 ──
  async GaeaJournalList(limit: number) {
    const now = Date.now();
    const opsJson = JSON.stringify([
      { type: "set_value", sheet: "预算", target: "B2", value: 120.5 },
      { type: "set_formula", sheet: "预算", target: "B4", formula: "SUM(B2:B3)" },
      { type: "replace", sheet: "预算", range: "A1:A3", find: "设备", replace: "机械" },
    ]);
    return [
      {
        id: "ev_1003", sessionId: "mock-session", space: "work", turn: 3,
        tool: "xlsx_apply", target: "docs/成本测算.xlsx",
        beforeSummary: "成本测算表初稿（mock）", afterSummary: "已应用规划操作并重算公式（mock）",
        model: "deepseek-v4-flash", at: now - 60_000, status: "applied",
        baselinePath: ".gaea/snapshots/docs/成本测算.xlsx.snap", opsJson,
      },
      {
        id: "ev_1002", sessionId: "mock-session", space: "work", turn: 2,
        tool: "edit_file", target: "docs/说明.md",
        beforeSummary: "旧说明内容（v1，mock）", afterSummary: "新说明内容（v2，mock）",
        at: now - 120_000, status: "applied",
      },
      {
        id: "ev_1001", sessionId: "mock-session", space: "work", turn: 1,
        tool: "xlsx_apply", target: "docs/旧表.xlsx",
        beforeSummary: "旧表快照——历史卡无 opsJson，仅回放变更摘要（mock）",
        afterSummary: "已应用（mock）",
        at: now - 180_000, status: "applied",
      },
    ].slice(0, limit);
  },
  async VerifyRecord(id: string) {
    // 样本复核结论：ev_1003 通过（携带 v4.16 通道 B 结构化字段——像素差异率/
    // 渲染页数/产物目录，前端渲染「视觉复核」行 + 查看产物按钮）；ev_1001
    // 警告但保持旧形态（无 channelB 结构化字段，向后兼容样本）。
    if (id === "ev_1003") {
      return {
        id,
        status: "verified" as const,
        channelA: "结构完整",
        channelB: "视觉正常",
        note: "（mock）双通道复核通过",
        channelBRatio: 0.013,
        channelBPages: 3,
        channelBArtifacts: ".gaea/work/journal/verify/ev_1003",
        at: Date.now(),
      };
    }
    if (id === "ev_1002") {
      // P2 条件视觉首渲样本：无基线 office_generate 交付——首渲覆盖 8/12 页
      // （>8 抽样，跳过原因透明），走「首渲覆盖」行（与 ev_1003 的
      // 「视觉复核」diff 行互斥形态）。
      return {
        id,
        status: "verified" as const,
        channelA: "pass: 文档可解析，替换文本在文档中且 1 处待处理修订在册（引用级）",
        channelB: "first-review: 首渲覆盖 8/12 页（无基线，视觉比对不适用；逐页 PNG 落产物目录供目检）",
        note: "（mock）通道 A 通过；条件视觉首渲已执行",
        visualPagesReviewed: 8,
        visualPagesTotal: 12,
        visualSkippedReason: "超 8 页抽样（前 8 页已渲，余 4 页跳过）",
        channelBArtifacts: ".gaea/work/journal/verify/ev_1002",
        at: Date.now(),
      };
    }
    return {
      id,
      status: (id === "ev_1001" ? "warned" : "verified") as "verified" | "warned" | "failed",
      channelA: "结构完整",
      channelB: "视觉正常",
      note: "（mock）双通道复核通过",
      at: Date.now(),
    };
  },
  async RollbackRecord(_id: string) {
    // 成功路径：mock 不落盘，返回空即可（前端 toast 透出成功文案）。
  },
  async WriteFile(rel: string, content: string) {
    // mock：浏览器开发环境不落盘（真实实现 = GaeaWriteFile 原子写回工作区）。
    // 走查态：会话内记忆（导图编辑 Ctrl+S 后预览回读可见，刷新即复位）。
    mockFileBodies[rel] = content;
  },
  // ── 批次三a legacy 直调转正（Go OfficeB.GaeaDataBackup*，Gaea 前缀经 mappings）──
  async DataBackupInfo() {
    // 未做过备份：中性空态（DataPanel 初始状态；不编造 lastBackup 时间）。
    return { lastBackup: null, status: "idle" };
  },
  async DataBackupCreate(destDir: string) {
    // mock：会话内成功形状（ok:true），不真实落备份文件（path 回显目标目录）。
    return { ok: true, path: destDir };
  },
  async DataBackupRestore(_zipPath: string) {
    // mock：声明成功（ok:true），不真实写盘；回滚语义见 DataBackupRollback。
    return { ok: true };
  },
  async DataBackupCancel() {
    // mock: no-op——无进行中的备份/恢复可取消。
  },
  async DataBackupRollback() {
    // mock：从未真实恢复过 → 无可回滚（false）。
    return false;
  },
  async DataBackupRestoreResult() {
    // 从未执行恢复：status:none（DataPanel 轮询空态）。
    return { status: "none" };
  },
};
