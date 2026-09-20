import { useCallback, useEffect, useMemo, useState } from "react";
import { Button, Input, Modal, Popconfirm, Tabs, Tag, Typography, message } from "antd";
import { Inbox, Loader } from "../icons";
import { app } from "../lib/bridge";
import { useT, type Translator } from "../lib/i18n";
import type { DictKey } from "../locales/en";
import type { ShellSpace } from "../../boards/space";
import type { TaskInboxStatus, TaskInboxView } from "../lib/types";
import V3Empty from "../../components/V3Empty";

// TaskInboxPanel — 任务收件箱（阶段七 7.3-1）：四入口（Ctrl+K/命令面板/语音/
// 微信）「存为任务」的持久卡视图 + 手动新建兜底。状态机四档 tab（待处理/
// 进行中/已完成/已放弃）带计数；行动作按 taskinbox.CanTransition 前端镜像
// 出现（pending→开始/放弃、doing→完成/放弃、终态只读），全态可删除。
// 零常驻判据：open 时拉一次 + 每次变更（新建/迁移/删除）后重拉——无轮询、
// 无定时器、无事件订阅，读写均由用户动作触发。风格对齐 TaskCenter
// （v3-panel-head 细条头；语义色 + 文字双传达）。

const { Text } = Typography;

/** 状态档序（tab 顺序 = 行动优先：待办在前） */
const STATUS_ORDER: TaskInboxStatus[] = ["pending", "doing", "done", "abandoned"];

const STATUS_LABEL: Record<TaskInboxStatus, DictKey> = {
  pending: "tasks.inbox.pending",
  doing: "tasks.inbox.doing",
  done: "tasks.inbox.done",
  abandoned: "tasks.inbox.abandoned",
};

/** 来源展示名（未知来源回退原始值——后端新增来源无需等前端发版） */
const SOURCE_LABEL: Record<string, DictKey> = {
  ctrlk: "tasks.inbox.source.ctrlk",
  palette: "tasks.inbox.source.palette",
  voice: "tasks.inbox.source.voice",
  weixin: "tasks.inbox.source.weixin",
  inbox: "tasks.inbox.source.inbox",
};

function sourceLabel(source: string, t: Translator): string {
  const key = SOURCE_LABEL[source];
  return key ? t(key) : source;
}

/** 相对时间（unix 毫秒 → 刚刚/N 分钟前/…；复用 launcher 既有字典键） */
function fmtRel(ms: number, t: Translator): string {
  if (!ms) return "—";
  const diff = Date.now() - ms;
  const min = Math.floor(diff / 60000);
  if (min < 1) return t("shell.launcher.fmtJustNow");
  if (min < 60) return t("shell.launcher.fmtMin", { n: min });
  const h = Math.floor(min / 60);
  if (h < 24) return t("shell.launcher.fmtHour", { n: h });
  const d = Math.floor(h / 24);
  if (d < 30) return t("shell.launcher.fmtDay", { n: d });
  return new Date(ms).toLocaleDateString();
}

/**
 * 状态机动作镜像（taskinbox.CanTransition 纯函数的前端展示逻辑）：
 * pending→doing（开始）；doing→done（完成）；pending|doing→abandoned（放弃）；
 * 终态（done/abandoned）不接受任何迁移 → 无状态按钮，仅保留删除。
 * 非法组合的按钮不渲染（而非禁用）——出现性即迁移合法性，测试钉死。
 */
function nextActions(status: TaskInboxStatus): { to: TaskInboxStatus; labelKey: DictKey; testid: string }[] {
  switch (status) {
    case "pending":
      return [
        { to: "doing", labelKey: "tasks.inbox.start", testid: "task-inbox-start" },
        { to: "abandoned", labelKey: "tasks.inbox.abandon", testid: "task-inbox-abandon" },
      ];
    case "doing":
      return [
        { to: "done", labelKey: "tasks.inbox.finish", testid: "task-inbox-finish" },
        { to: "abandoned", labelKey: "tasks.inbox.abandon", testid: "task-inbox-abandon" },
      ];
    default:
      return [];
  }
}

/** 单行任务卡：标题 + 来源 Tag + 相对时间 + note 折叠 + 动作行 */
function TaskRow({
  task,
  busy,
  onSetStatus,
  onDelete,
  onNavigate,
  onResumeSession,
}: {
  task: TaskInboxView;
  busy: boolean;
  onSetStatus: (id: string, status: TaskInboxStatus) => void;
  onDelete: (id: string) => void;
  onNavigate?: (boardId: string) => void;
  /** 会话级回源（7.3-1 收口）：带会话路径精确恢复；缺省回退板块粒度。 */
  onResumeSession?: (path: string) => void;
}) {
  const t = useT();
  return (
    <div
      data-testid="task-inbox-row"
      className="rounded-[var(--radius-md)] px-2.5 py-2 space-y-1"
      style={{
        background: "var(--md-sys-color-surface-container)",
        border: "1px solid var(--md-sys-color-outline-variant)",
      }}
    >
      <div className="flex items-center gap-2">
        <Text strong className="!text-[12px]" style={{ color: "var(--md-sys-color-text)" }} ellipsis={{ tooltip: task.title }}>
          {task.title}
        </Text>
        <Tag className="!m-0 shrink-0" style={{ fontSize: 10, lineHeight: "16px" }}>
          {sourceLabel(task.source, t)}
        </Tag>
        <span className="ml-auto shrink-0 text-[10px] font-mono tabular-nums" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {fmtRel(task.updatedAt, t)}
        </span>
      </div>
      {task.note && (
        /* note 折叠：默认单行省略，展开后看全文（antd Paragraph ellipsis 行折叠） */
        <Typography.Paragraph
          type="secondary"
          className="!text-[10.5px] !mb-0"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          ellipsis={{ rows: 1, expandable: true }}
        >
          {task.note}
        </Typography.Paragraph>
      )}
      <div className="flex items-center gap-1.5 flex-wrap">
        {nextActions(task.status).map((a) => (
          <Button
            key={a.to}
            size="small"
            disabled={busy}
            data-testid={`${a.testid}-${task.id}`}
            onClick={() => onSetStatus(task.id, a.to)}
          >
            {t(a.labelKey)}
          </Button>
        ))}
        {/* 跳转：意图为导航且带目标板块 → 板块级跳回（「去板块」是导航不是触发） */}
        {task.action === "navigate" && task.target && onNavigate && (
          <Button size="small" type="dashed" data-testid={`task-inbox-go-${task.id}`} onClick={() => onNavigate(task.target!)}>
            {t("tasks.inbox.goBoard")}
          </Button>
        )}
        {/* 源会话：会话级回源（7.3-1 收口）——有回调且带路径走精确恢复；
            否则回退 V1 板块粒度回工作台（Session 落库走审计链） */}
        {task.session && (onResumeSession || onNavigate) && (
          <Button size="small" type="dashed" data-testid={`task-inbox-back-${task.id}`}
            onClick={() => (onResumeSession && task.session ? onResumeSession(task.session) : onNavigate?.("gaea"))}>
            {t("tasks.inbox.backSession")}
          </Button>
        )}
        <Popconfirm
          title={t("tasks.inbox.deleteConfirm")}
          okText={t("tasks.inbox.delete")}
          cancelText={t("common.cancel")}
          okButtonProps={{ danger: true, "data-testid": "task-inbox-delete-confirm" }}
          onConfirm={() => onDelete(task.id)}
        >
          <Button
            size="small"
            danger
            disabled={busy}
            className="ml-auto"
            data-testid={`task-inbox-delete-${task.id}`}
          >
            {t("tasks.inbox.delete")}
          </Button>
        </Popconfirm>
      </div>
    </div>
  );
}

export function TaskInboxPanel({
  open,
  onClose,
  space,
  onNavigate,
  onResumeSession,
}: {
  open: boolean;
  onClose: () => void;
  space: ShellSpace;
  /** 板块级跳转（ModuleLauncher 既有 onNavigate；缺省时跳转按钮不渲染） */
  onNavigate?: (boardId: string) => void;
  /** 会话级回源（7.3-1 收口）：透传 Board。 */
  onResumeSession?: (path: string) => void;
}) {
  const t = useT();
  // 待处理计数提升给 Modal 标题徽标（7.3-2：面板=Board 的 Modal 壳，DOM 零变化）
  const [pending, setPending] = useState(0);
  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      width={620}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      title={
        // v3-panel-head 细条头：图标 + 标题 + 待处理计数（对齐 TaskCenter 语言）
        <div className="v3-panel-head">
          <Inbox size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
          <span className="v3-panel-title">{t("tasks.inbox.title")}</span>
          {pending > 0 && (
            <span
              className="px-1.5 py-px rounded-full text-[10px]"
              style={{
                background: "color-mix(in srgb, var(--md-sys-color-primary-container) 55%, transparent)",
                color: "var(--gaea-glow)",
                border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
              }}
            >
              {t("tasks.inbox.pendingCount", { n: pending })}
            </span>
          )}
        </div>
      }
    >
      <TaskInboxBoard space={space} active={open} onNavigate={onNavigate} onPendingChange={setPending} onResumeSession={onResumeSession} />
    </Modal>
  );
}

/**
 * TaskInboxBoard 收件箱内联板（7.3-2 板块降级为任务视图）：原 TaskInboxPanel
 * 主体原样抽出——四档 tab + 新建 + 状态机动作 + 板块跳转。Modal 壳保留为薄
 * 包装（面板 props/DOM/既有测试零变化）；任务优先首页直接内嵌本组件。
 * active=false 时不拉取（Modal 关闭态）；onPendingChange 把待处理计数报给
 * 壳层（Modal 标题徽标 / 首页头部计数共用）。
 */
export function TaskInboxBoard({
  space,
  active,
  onNavigate,
  onPendingChange,
  onResumeSession,
}: {
  space: ShellSpace;
  /** 激活态（false=挂载但不拉取；内嵌首页恒 true） */
  active: boolean;
  onNavigate?: (boardId: string) => void;
  onPendingChange?: (pending: number) => void;
  /** 会话级回源（7.3-1 收口）：透传 TaskRow。 */
  onResumeSession?: (path: string) => void;
}) {
  const t = useT();
  const [tasks, setTasks] = useState<TaskInboxView[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [tab, setTab] = useState<TaskInboxStatus>("pending");
  const [draft, setDraft] = useState("");
  const [busyId, setBusyId] = useState<string | null>(null);

  // 拉取：open 时 + 每次变更后（用户动作触发，零轮询零定时器）
  const load = useCallback(() => {
    setLoading(true);
    let alive = true;
    app
      .GaeaTaskInboxList(space)
      .then((list: unknown) => {
        if (alive) {
          setTasks((list ?? []) as TaskInboxView[]);
          setLoadError(false);
        }
      })
      .catch(() => {
        // 文件缺失/损坏后端回空表；这里是桥接失败 → 空表 + 失败空态（可重试）
        if (alive) {
          setTasks([]);
          setLoadError(true);
        }
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [space]);

  useEffect(() => {
    if (!active) return;
    return load();
  }, [active, load]);

  // 手动新建：source='inbox'（第五来源兜底），回车或点「添加」提交
  const submitAdd = useCallback(() => {
    const title = draft.trim();
    if (!title || busyId) return;
    setBusyId("add");
    app
      .GaeaTaskInboxSave(JSON.stringify({ title, space, source: "inbox" }))
      .then(() => {
        setDraft("");
        load();
      })
      .catch(() => {
        /* 保存失败保持草稿，用户可再点添加 */
      })
      .finally(() => setBusyId(null));
  }, [draft, space, busyId, load]);

  const setStatus = useCallback(
    (id: string, status: TaskInboxStatus) => {
      if (busyId) return;
      setBusyId(id);
      app
        .GaeaTaskInboxSetStatus(id, status)
        .then(() => load())
        // v4.362：状态变更失败不再静默——原「点了没反应」UI 停旧状态。
        .catch(() => message.error(t("tasks.inbox.statusFail")))
        .finally(() => setBusyId(null));
    },
    [busyId, load, t],
  );

  const remove = useCallback(
    (id: string) => {
      if (busyId) return;
      setBusyId(id);
      app
        .GaeaTaskInboxDelete(id)
        .then(() => load())
        // v4.350：失败可见化——此前吞错，用户点删除后按钮闪一下毫无反应
        .catch(() => message.error(t("tasks.inbox.deleteFail")))
        .finally(() => setBusyId(null));
    },
    [busyId, load, t],
  );

  // 四档计数（tab 标签 = 状态名 + 计数）
  const counts = useMemo(() => {
    const c: Record<TaskInboxStatus, number> = { pending: 0, doing: 0, done: 0, abandoned: 0 };
    for (const tsk of tasks) c[tsk.status] += 1;
    return c;
  }, [tasks]);

  const rows = useMemo(() => tasks.filter((t2) => t2.status === tab), [tasks, tab]);

  useEffect(() => {
    onPendingChange?.(counts.pending);
  }, [counts.pending, onPendingChange]);

  return (
      <div data-testid="task-inbox-panel" className="flex flex-col gap-3">
        {/* 手动新建（回车提交） */}
        <div className="flex items-center gap-2">
          <Input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onPressEnter={submitAdd}
            placeholder={t("tasks.inbox.addPlaceholder")}
            size="small"
            allowClear
            data-testid="task-inbox-input"
          />
          <Button
            size="small"
            type="primary"
            data-testid="task-inbox-add"
            disabled={!draft.trim() || busyId !== null}
            loading={busyId === "add"}
            onClick={submitAdd}
          >
            {t("tasks.inbox.add")}
          </Button>
        </div>

        <Tabs
          size="small"
          activeKey={tab}
          onChange={(k) => setTab(k as TaskInboxStatus)}
          items={STATUS_ORDER.map((s) => ({
            key: s,
            label: (
              <span data-testid={`task-inbox-tab-${s}`}>
                {t(STATUS_LABEL[s])} {counts[s]}
              </span>
            ),
          }))}
        />

        <div className="max-h-[46vh] overflow-y-auto flex flex-col gap-2 pr-0.5">
          {loading && tasks.length === 0 ? (
            <div className="flex items-center justify-center gap-2 py-6 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              <Loader size={14} className="animate-spin" aria-hidden />
              {t("tasks.loading")}
            </div>
          ) : rows.length === 0 ? (
            loadError ? (
              // 空态两分之二：加载失败 → 可重试
              <V3Empty compact description={t("tasks.inbox.loadFail")}>
                <Button size="small" data-testid="task-inbox-retry" onClick={load}>
                  {t("tasks.inbox.retry")}
                </Button>
              </V3Empty>
            ) : (
              // 空态两分之一：无任务（含某档 tab 为空的局部空态）
              <V3Empty
                compact
                description={
                  <span>
                    {t("tasks.inbox.empty")}
                    {tasks.length === 0 && (
                      <span className="block mt-1 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        {t("tasks.inbox.emptyHint")}
                      </span>
                    )}
                  </span>
                }
              />
            )
          ) : (
            rows.map((task) => (
              <TaskRow
                key={task.id}
                task={task}
                busy={busyId !== null}
                onSetStatus={setStatus}
                onDelete={remove}
                onNavigate={onNavigate}
                onResumeSession={onResumeSession}
              />
            ))
          )}
        </div>
      </div>
  );
}
