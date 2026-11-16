import { useMemo, useState } from "react";
import { BookOpen, ChevronRight, Clock, Plus, Shield } from "../icons";
import { relativeTime } from "../lib/time";
import type { ProjectGroup } from "../lib/types";

// WenxueHome is the no-project state. Once a work is selected, App renders a
// focused writing welcome around the composer instead of a project dashboard.

function projectModified(project: ProjectGroup): number {
  const latestSession = project.sessions.reduce(
    (latest, session) => Math.max(latest, session.modTime),
    0,
  );
  return Math.max(project.modTime, latestSession);
}

function coverVariant(name: string): number {
  let hash = 0;
  for (const char of name) hash = (hash * 31 + char.codePointAt(0)!) | 0;
  return Math.abs(hash) % 4;
}

function PrivacyButton({
  enabled,
  onToggle,
}: {
  enabled: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      className={`wx-home-library-action${enabled ? " is-active" : ""}`}
      aria-pressed={enabled}
      onClick={onToggle}
      title={enabled ? "关闭首页隐私" : "隐藏作品名与最近记录"}
    >
      <Shield size={14} aria-hidden />
      <span>{enabled ? "已隐身" : "首页隐私"}</span>
    </button>
  );
}

export function WenxueHome({
  projects,
  projectsLoaded,
  onOpenProject,
  onCreateNovel,
  onOpenHistory,
  privateMode,
  onTogglePrivacy,
}: {
  projects: ProjectGroup[];
  projectsLoaded: boolean;
  onOpenProject: (path: string) => Promise<string>;
  onCreateNovel: () => void;
  onOpenHistory: () => void;
  privateMode: boolean;
  onTogglePrivacy: () => void;
}) {
  const sortedProjects = useMemo(
    () => [...projects].sort((a, b) => projectModified(b) - projectModified(a)),
    [projects],
  );
  const totalSessions = projects.reduce(
    (total, project) => total + project.sessions.length,
    0,
  );
  const [opening, setOpening] = useState("");
  const [error, setError] = useState("");
  const hasProjects = sortedProjects.length > 0;

  const openProject = async (path: string) => {
    setOpening(path);
    setError("");
    try {
      const picked = await onOpenProject(path);
      if (!picked) setError("未能打开这部作品，请稍后重试。");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setOpening("");
    }
  };

  return (
    <section
      className="wx-home wx-home--library"
      data-testid="wenxue-home-library"
    >
      <div className="wx-home-library-shell">
        <header className="wx-home-library-header">
          <div className="wx-home-library-kicker">
            <BookOpen size={14} aria-hidden />
            <span>文学作品</span>
          </div>
          <h1>
            {!projectsLoaded
              ? "正在打开作品库"
              : hasProjects
                ? "选择一部作品"
                : "开始第一部作品"}
          </h1>
          <p>
            {!projectsLoaded
              ? "正在读取本机作品与会话。"
              : hasProjects
                ? "从最近修改的作品继续，或新建一部小说。"
                : "先建一本书。人物、设定、章节与时间线都会留在同一作品目录中。"}
          </p>
          {projectsLoaded && (
            <div className="wx-home-library-actions" aria-label="作品库操作">
              <button
                type="button"
                className="wx-home-library-action is-primary"
                onClick={onCreateNovel}
              >
                <Plus size={14} aria-hidden />
                <span>新建小说</span>
              </button>
              {hasProjects && (
                <button
                  type="button"
                  className="wx-home-library-action"
                  onClick={onOpenHistory}
                >
                  <Clock size={14} aria-hidden />
                  <span>全部历史</span>
                </button>
              )}
              <PrivacyButton enabled={privateMode} onToggle={onTogglePrivacy} />
            </div>
          )}
          {error && (
            <div className="wx-home-library-error" role="alert">
              {error}
            </div>
          )}
        </header>

        {!projectsLoaded ? (
          <div className="wx-home-library-loading" role="status">
            <span />
            <span />
            <span />
          </div>
        ) : hasProjects ? (
          <section className="wx-home-works" aria-labelledby="wx-home-works-title">
            <div className="wx-home-works-head">
              <h2 id="wx-home-works-title">最近作品</h2>
              <span>
                {sortedProjects.length} 部作品 · {totalSessions} 个会话
              </span>
            </div>
            <div className="wx-home-work-grid">
              {sortedProjects.map((project, index) => {
                const visibleName = privateMode
                  ? `作品 ${String(index + 1).padStart(2, "0")}`
                  : project.name;
                const variant = coverVariant(project.name);
                return (
                  <button
                    key={project.path}
                    type="button"
                    className="wx-home-work-card"
                    onClick={() => void openProject(project.path)}
                    disabled={!!opening}
                    aria-busy={opening === project.path}
                  >
                    <span
                      className={`wx-home-work-cover wx-home-work-cover--v${variant}`}
                      aria-hidden
                    >
                      <span className="wx-home-work-cover__mark">文学</span>
                      <strong>{visibleName}</strong>
                      <span className="wx-home-work-cover__foot">长篇小说</span>
                    </span>
                    <span className="wx-home-work-meta">
                      <strong>{visibleName}</strong>
                      <small>
                        {project.sessions.length} 个会话 ·{" "}
                        {relativeTime(projectModified(project))}
                      </small>
                      <span className="wx-home-work-open">
                        {opening === project.path ? "正在打开" : "打开作品"}
                        <ChevronRight size={12} aria-hidden />
                      </span>
                    </span>
                  </button>
                );
              })}
            </div>
          </section>
        ) : (
          <div className="wx-home-library-empty">
            <BookOpen size={22} aria-hidden />
            <p>还没有作品。创建后会自动建立独立的写作目录。</p>
          </div>
        )}
      </div>
    </section>
  );
}
