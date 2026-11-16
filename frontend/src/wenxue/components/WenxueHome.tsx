import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ArrowUpRight, BookOpen, Clock, FileText, ListTree, MessageSquare,
  Plus, RefreshCw, ScrollText, Shield, Sparkles, Users,
} from "../icons";
import type { Icon } from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import { sessionTitle } from "../lib/session";
import { relativeTime } from "../lib/time";
import type { ProjectGroup, SessionMeta } from "../lib/types";

// WenxueHome — 文学板块对话页的两种首页。
// 1) 未进入作品：书斋作品库，负责选书、回到最近作品或创建第一部小说。
// 2) 已进入作品：作品控制台，读取现有目录的轻量计数，给出续写、角色、
//    设定、时间线和最近会话入口。这里只读目录元数据，不改任何文件。

export interface WenxueHomeQuickAction {
  key: string;
  label: string;
  Icon: Icon;
  prompt: string;
}

interface WorkspaceSnapshot {
  chapters: number;
  finalChapters: number;
  characters: number;
  settings: number;
  timelineFiles: number;
  characterNames: string[];
  settingNames: string[];
  latestChapter: { number: number; title: string } | null;
}

const EMPTY_SNAPSHOT: WorkspaceSnapshot = {
  chapters: 0,
  finalChapters: 0,
  characters: 0,
  settings: 0,
  timelineFiles: 0,
  characterNames: [],
  settingNames: [],
  latestChapter: null,
};

const CHAPTER_DIRS = ["章节", "chapters"];
const FINAL_DIRS = ["定稿", "final", "finalized"];
const CHARACTER_DIRS = ["角色", "characters", "人物"];
const SETTING_DIRS = ["设定", "worldview", "设定集"];
const TIMELINE_DIRS = ["时间线", "timeline"];

function shortName(name: string, max = 8): string {
  const chars = [...name.trim()];
  return chars.length > max ? `${chars.slice(0, max).join("")}…` : chars.join("");
}

function coverVariant(name: string): number {
  let hash = 0;
  for (const ch of name) hash = (hash * 31 + ch.codePointAt(0)!) | 0;
  return Math.abs(hash) % 4;
}

function projectModified(project: ProjectGroup): number {
  const latest = project.sessions.reduce((max, session) => Math.max(max, session.modTime), 0);
  return Math.max(project.modTime, latest);
}

function parseChapterFile(name: string): { number: number; title: string } {
  const stem = name.replace(/\.(md|markdown|txt)$/i, "");
  const match = /^(\d+)[-_\s.：:]*(.*)$/.exec(stem);
  if (match) return { number: parseInt(match[1], 10) || 0, title: match[2].trim() || stem };
  return { number: 0, title: stem };
}

function ProjectCover({
  name,
  variant,
  featured = false,
  privateMode = false,
}: {
  name: string;
  variant: number;
  featured?: boolean;
  privateMode?: boolean;
}) {
  const visibleName = privateMode ? "私密作品" : shortName(name, featured ? 12 : 9);
  return (
    <span
      className={`wx-home-cover wx-home-cover--v${variant}${featured ? " is-featured" : ""}`}
      aria-hidden
    >
      <span className="wx-home-cover__rule" />
      <span className="wx-home-cover__mark">gaea · 文学</span>
      <strong>{visibleName}</strong>
      <span className="wx-home-cover__foot">长篇小说工作空间</span>
    </span>
  );
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
      className={`wx-hub-action wx-home-privacy${enabled ? " is-active" : ""}`}
      aria-pressed={enabled}
      onClick={onToggle}
      title={enabled ? "关闭首页隐身模式" : "隐藏书名、会话摘要和最新章节"}
    >
      <Shield size={13} aria-hidden />
      <span>{enabled ? "已隐身" : "首页隐私"}</span>
    </button>
  );
}

function LibraryHome({
  projects,
  recentSessions,
  loaded,
  privateMode,
  onTogglePrivacy,
  onOpenProject,
  onCreateNovel,
  onResumeSession,
  onOpenHistory,
}: {
  projects: ProjectGroup[];
  recentSessions: SessionMeta[];
  loaded: boolean;
  privateMode: boolean;
  onTogglePrivacy: () => void;
  onOpenProject: (path: string) => Promise<string>;
  onCreateNovel: () => void;
  onResumeSession: (path: string, projectPath: string) => void;
  onOpenHistory: () => void;
}) {
  const sortedProjects = useMemo(
    () => [...projects].sort((a, b) => projectModified(b) - projectModified(a)),
    [projects],
  );
  const featured = sortedProjects[0];
  const totalSessions = projects.reduce((sum, project) => sum + project.sessions.length, 0);
  const recent = useMemo(
    () => [...recentSessions].sort((a, b) => b.modTime - a.modTime).slice(0, 4),
    [recentSessions],
  );
  const [opening, setOpening] = useState("");
  const [error, setError] = useState("");

  const openProject = async (path: string) => {
    setOpening(path);
    setError("");
    try {
      const picked = await onOpenProject(path);
      if (!picked) setError("切换作品失败，请稍后重试");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setOpening("");
    }
  };

  if (!loaded) {
    return (
      <section className="wx-hub wx-home wx-home--library" data-testid="wenxue-home-library">
        <div className="wx-home-loading">
          <span className="wx-home-loading__mark"><BookOpen size={18} aria-hidden /></span>
          <span>正在整理书斋…</span>
        </div>
      </section>
    );
  }

  return (
    <section className="wx-hub wx-home wx-home--library" data-testid="wenxue-home-library">
      <div className="wx-home-library">
        <header className="wx-home-hero wx-home-hero--library">
          <div className="wx-home-hero__copy">
            <div className="wx-home-kicker">文学工作台 · 作品库</div>
            <h1>{projects.length > 0 ? "从书架上，继续你的故事。" : "先给故事一个可以生长的地方。"}</h1>
            <p>
              {projects.length > 0
                ? "每部小说有独立的人物、设定、章节与会话。先从最近写过的作品继续，也可以另起一本新书。"
                : "一部小说就是一座独立书斋。建好工作空间后，人物、设定、定稿和时间线都会在这里各归其位。"}
            </p>
            <div className="wx-home-hero__actions">
              <button type="button" className="wx-hub-action is-primary" onClick={onCreateNovel}>
                <Plus size={13} aria-hidden />
                <span>新建小说</span>
              </button>
              {featured && (
                <button
                  type="button"
                  className="wx-hub-action"
                  onClick={() => void openProject(featured.path)}
                  disabled={!!opening}
                >
                  <BookOpen size={13} aria-hidden />
                  <span>{opening === featured.path ? "正在打开…" : privateMode ? "继续最近作品" : `继续《${featured.name}》`}</span>
                </button>
              )}
              {projects.length > 0 && (
                <button type="button" className="wx-hub-action" onClick={onOpenHistory}>
                  <Clock size={13} aria-hidden />
                  <span>全部历史</span>
                </button>
              )}
              <PrivacyButton enabled={privateMode} onToggle={onTogglePrivacy} />
            </div>
            {error && <div className="wx-home-inline-error" role="alert">{error}</div>}
          </div>

          <div className="wx-home-cover-stage" aria-hidden>
            {featured ? (
              <>
                <ProjectCover
                  name={featured.name}
                  variant={coverVariant(featured.name)}
                  featured
                  privateMode={privateMode}
                />
                {sortedProjects.slice(1, 3).map((project, index) => (
                  <span
                    key={project.path}
                    className={`wx-home-cover-stack-item is-${index + 1}`}
                  >
                    <ProjectCover name={project.name} variant={coverVariant(project.name)} privateMode={privateMode} />
                  </span>
                ))}
              </>
            ) : (
              <div className="wx-home-blank-shelf">
                <span className="wx-home-blank-shelf__line" />
                <span className="wx-home-blank-shelf__copy">
                  <BookOpen size={21} aria-hidden />
                  <strong>你的第一部小说</strong>
                  <small>从空白稿纸开始</small>
                </span>
              </div>
            )}
          </div>
        </header>

        {projects.length > 0 ? (
          <div className="wx-home-library__grid">
            <main className="wx-home-shelf">
              <div className="wx-home-section-head">
                <div>
                  <span className="wx-home-section-kicker">作品书架</span>
                  <h2>我的小说</h2>
                </div>
                <span className="wx-home-section-meta">{projects.length} 部作品 · {totalSessions} 个会话</span>
              </div>
              <div className="wx-home-shelf__grid">
                {sortedProjects.map((project, index) => {
                  const isFeatured = index === 0;
                  const visibleName = privateMode ? `作品 · ${String(index + 1).padStart(2, "0")}` : project.name;
                  return (
                    <button
                      key={project.path}
                      type="button"
                      className={`wx-home-project-tile${isFeatured ? " is-featured" : ""}`}
                      onClick={() => void openProject(project.path)}
                      disabled={!!opening}
                    >
                      <ProjectCover
                        name={project.name}
                        variant={coverVariant(project.name)}
                        featured={isFeatured}
                        privateMode={privateMode}
                      />
                      <span className="wx-home-project-tile__copy">
                        <span className="wx-home-project-tile__title">{visibleName}</span>
                        <span className="wx-home-project-tile__meta">
                          {project.sessions.length} 个会话 · {relativeTime(projectModified(project))}
                        </span>
                        <span className="wx-home-project-tile__session">
                          {privateMode ? "会话内容已隐藏" : "最近写作记录已收起"}
                        </span>
                        <span className="wx-home-project-tile__open">
                          {opening === project.path ? "正在打开…" : "进入作品"}
                          <ArrowUpRight size={11} aria-hidden />
                        </span>
                      </span>
                    </button>
                  );
                })}
                <button type="button" className="wx-home-create-tile" onClick={onCreateNovel}>
                  <span className="wx-home-create-tile__mark"><Plus size={17} aria-hidden /></span>
                  <span>新建小说</span>
                  <small>建立下一个独立书斋</small>
                </button>
              </div>
            </main>

            <aside className="wx-home-library__aside">
              <section className="wx-hub-paper wx-home-side-card">
                <div className="wx-home-side-card__head">
                  <h2>最近打开</h2>
                  <MessageSquare size={12} aria-hidden />
                </div>
                {recent.length > 0 ? (
                  <div className="wx-home-session-list">
                    {recent.map((session) => {
                      const project = projects.find((item) =>
                        item.sessions.some((candidate) => candidate.path === session.path));
                      return (
                        <button
                          key={session.path}
                          type="button"
                          className="wx-home-session-row"
                          onClick={() => project && onResumeSession(session.path, project.path)}
                          disabled={!project}
                        >
                          <span className="wx-home-session-row__mark" aria-hidden><MessageSquare size={10} /></span>
                          <span className="wx-home-session-row__copy">
                            <strong>{privateMode ? "会话内容已隐藏" : project ? `继续《${project.name}》` : "写作会话"}</strong>
                            <small>{privateMode ? "隐私模式" : "最近写作"} · {relativeTime(session.modTime)}</small>
                          </span>
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <p className="wx-home-side-empty">还没有可恢复的写作会话。选中作品后从第一章开始。</p>
                )}
              </section>

              <section className="wx-hub-paper wx-home-side-card wx-home-library-note">
                <span className="wx-home-section-kicker">书斋约定</span>
                <p>会话、正文、角色、设定与时间线都留在对应小说目录中，不与其他作品混在一起。</p>
              </section>
            </aside>
          </div>
        ) : (
          <div className="wx-home-onboarding">
            <section className="wx-hub-paper wx-home-onboarding__main">
              <span className="wx-home-section-kicker">第一次使用</span>
              <h2>不需要先准备完整的世界。</h2>
              <p>先建一本书，写下一个角色和一个冲突。角色志、阅读室与世界设定志会随着稿件一起长大。</p>
              <button type="button" className="wx-hub-action is-primary" onClick={onCreateNovel}>
                <Plus size={13} aria-hidden />
                <span>创建第一部小说</span>
              </button>
            </section>
            <section className="wx-hub-paper wx-home-onboarding__steps">
              {[
                ["01", "命名作品", "建立独立的小说工作空间。"],
                ["02", "写下第一场戏", "从人物想要什么、什么阻止他开始。"],
                ["03", "逐步建立档案", "人物、设定与时间线在写作中自然补齐。"],
              ].map(([number, title, desc]) => (
                <div key={number} className="wx-home-step">
                  <span>{number}</span>
                  <div>
                    <strong>{title}</strong>
                    <p>{desc}</p>
                  </div>
                </div>
              ))}
            </section>
          </div>
        )}
      </div>
    </section>
  );
}

function ProjectHome({
  project,
  currentSessionPath,
  quickActions,
  privateMode,
  onTogglePrivacy,
  onInsertPrompt,
  onNewSession,
  onResumeSession,
  onSwitchTab,
  onOpenLibrary,
}: {
  project: ProjectGroup;
  currentSessionPath?: string;
  quickActions: WenxueHomeQuickAction[];
  privateMode: boolean;
  onTogglePrivacy: () => void;
  onInsertPrompt: (prompt: string) => void;
  onNewSession: () => void;
  onResumeSession: (path: string, projectPath: string) => void;
  onSwitchTab: (tab: "characters" | "reading" | "settings") => void;
  onOpenLibrary: () => void;
}) {
  const [snapshot, setSnapshot] = useState<WorkspaceSnapshot>(EMPTY_SNAPSHOT);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [reloadTick, setReloadTick] = useState(0);
  const variant = coverVariant(project.name);

  const loadSnapshot = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const rootEntries = await app.ListDir("");
      const findDir = (candidates: string[]) =>
        candidates.find((candidate) =>
          rootEntries.some((entry) => entry.isDir && entry.name === candidate)) ?? "";
      const listMd = async (candidates: string[]) => {
        const dir = findDir(candidates);
        if (!dir) return [] as Array<{ name: string; isDir: boolean }>;
        const entries = await app.ListDir(dir);
        return entries.filter((entry) =>
          !entry.isDir && /\.(md|markdown|txt)$/i.test(entry.name));
      };
      const [chapterFiles, finalFiles, characterFiles, settingFiles, timelineFiles] = await Promise.all([
        listMd(CHAPTER_DIRS),
        listMd(FINAL_DIRS),
        listMd(CHARACTER_DIRS),
        listMd(SETTING_DIRS),
        listMd(TIMELINE_DIRS),
      ]);
      const latestChapter = chapterFiles
        .map((entry) => ({ ...parseChapterFile(entry.name), file: entry.name }))
        .sort((a, b) => (a.number !== b.number
          ? b.number - a.number
          : b.file.localeCompare(a.file, "zh-Hans-CN")))[0] ?? null;
      setSnapshot({
        chapters: chapterFiles.length,
        finalChapters: finalFiles.length,
        characters: characterFiles.length,
        settings: settingFiles.length,
        timelineFiles: timelineFiles.length,
        characterNames: characterFiles.slice(0, 4).map((entry) =>
          entry.name.replace(/\.(md|markdown|txt)$/i, "")),
        settingNames: settingFiles.slice(0, 4).map((entry) =>
          entry.name.replace(/\.(md|markdown|txt)$/i, "")),
        latestChapter,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setSnapshot(EMPTY_SNAPSHOT);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void loadSnapshot(); }, [loadSnapshot, project.path, reloadTick]);

  useEffect(() => {
    return subscribe("gaea-wenxue-filetree", () => {
      setReloadTick((tick) => tick + 1);
    });
  }, []);

  const sessions = useMemo(
    () => [...project.sessions].sort((a, b) => b.modTime - a.modTime),
    [project.sessions],
  );
  const lastWritten = sessions.find((session) => session.turns > 0);
  const continueAction = quickActions.find((action) => action.key === "chapter-continue");
  const outlineAction = quickActions.find((action) => action.key === "outline-push");
  const latestLabel = privateMode
    ? "章节内容已隐藏"
    : snapshot.latestChapter
      ? snapshot.latestChapter.number > 0
        ? `第 ${snapshot.latestChapter.number} 章 · ${snapshot.latestChapter.title}`
        : snapshot.latestChapter.title
      : "章节尚未建立";

  return (
    <section className="wx-hub wx-home wx-home--project" data-testid="wenxue-home-project">
      <div className="wx-home-project">
        <header className="wx-home-hero wx-home-hero--project">
          <div className="wx-home-hero__copy">
            <div className="wx-home-kicker">作品控制台 · {relativeTime(projectModified(project))}编辑</div>
            <h1>{privateMode ? "作品已隐身" : project.name}</h1>
            <p>{loading ? "正在读取作品结构…" : latestLabel}</p>
            <div className="wx-home-hero__actions">
              <button
                type="button"
                className="wx-hub-action is-primary"
                onClick={() => onInsertPrompt(continueAction?.prompt ?? "请读取最近章节与角色状态，续写下一章。")}
              >
                <FileText size={13} aria-hidden />
                <span>继续正文</span>
              </button>
              <button
                type="button"
                className="wx-hub-action"
                onClick={() => {
                  if (lastWritten) onResumeSession(lastWritten.path, project.path);
                  else onNewSession();
                }}
              >
                <MessageSquare size={13} aria-hidden />
                <span>{lastWritten ? "回到最近会话" : "开始新会话"}</span>
              </button>
              <button type="button" className="wx-hub-action" onClick={onOpenLibrary}>
                <BookOpen size={13} aria-hidden />
                <span>作品书架</span>
              </button>
              <PrivacyButton enabled={privateMode} onToggle={onTogglePrivacy} />
            </div>
            {error && (
              <div className="wx-home-inline-error" role="alert">
                读取作品结构失败：{error}
                <button type="button" onClick={() => setReloadTick((tick) => tick + 1)}>重试</button>
              </div>
            )}
          </div>
          <div className="wx-home-project__cover" aria-hidden>
            <ProjectCover name={project.name} variant={variant} featured privateMode={privateMode} />
            <span className="wx-home-project__spine" />
          </div>
        </header>

        <div className="wx-home-project__metrics" aria-label="作品概况">
          {[
            ["章节", snapshot.chapters, "正文文件"],
            ["定稿", snapshot.finalChapters, "已入库"],
            ["角色", snapshot.characters, "人物卡"],
            ["设定", snapshot.settings, "世界档案"],
          ].map(([label, value, hint]) => (
            <div key={String(label)} className="wx-home-metric">
              <span>{label}</span>
              <strong>{loading ? "…" : value}</strong>
              <small>{hint}</small>
            </div>
          ))}
        </div>

        <div className="wx-home-project__grid">
          <main className="wx-home-project__main">
            <div className="wx-home-section-head">
              <div>
                <span className="wx-home-section-kicker">下一步</span>
                <h2>从当前作品继续</h2>
              </div>
              <button
                type="button"
                className="wx-hub-icon-btn"
                onClick={() => setReloadTick((tick) => tick + 1)}
                title="刷新作品结构"
                aria-label="刷新作品结构"
              >
                <RefreshCw size={11} aria-hidden />
              </button>
            </div>

            <div className="wx-home-action-grid">
              <button
                type="button"
                className="wx-home-action-card is-large"
                onClick={() => onInsertPrompt(continueAction?.prompt ?? "请读取最近章节与角色状态，续写下一章。")}
              >
                <span className="wx-home-action-card__icon"><FileText size={17} aria-hidden /></span>
                <span className="wx-home-action-card__copy">
                  <strong>{continueAction?.label ?? "续写推进"}</strong>
                  <span>衔接最近章节，推进冲突并留下章末钩子。</span>
                </span>
                <ArrowUpRight size={14} aria-hidden />
              </button>
              <button
                type="button"
                className="wx-home-action-card"
                onClick={() => onInsertPrompt("请检查主要角色的「当前状态」与最近三章是否一致，列出需要更新的角色卡并在确认后修订。")}
              >
                <span className="wx-home-action-card__icon"><Users size={15} aria-hidden /></span>
                <span className="wx-home-action-card__copy">
                  <strong>核对人物状态</strong>
                  <span>检查角色关系与当前状态。</span>
                </span>
                <ArrowUpRight size={12} aria-hidden />
              </button>
              <button
                type="button"
                className="wx-home-action-card"
                onClick={() => onInsertPrompt("请从最近章节反推时间线，补记漏掉的事件，并标注章号错标或时序矛盾。")}
              >
                <span className="wx-home-action-card__icon"><ListTree size={15} aria-hidden /></span>
                <span className="wx-home-action-card__copy">
                  <strong>整理时间线</strong>
                  <span>补记事件，核对章序。</span>
                </span>
                <ArrowUpRight size={12} aria-hidden />
              </button>
              <button
                type="button"
                className="wx-home-action-card"
                onClick={() => onInsertPrompt(outlineAction?.prompt ?? "请基于当前时间线与角色卡，推演未来十章走向。")}
              >
                <span className="wx-home-action-card__icon"><Sparkles size={15} aria-hidden /></span>
                <span className="wx-home-action-card__copy">
                  <strong>{outlineAction?.label ?? "推演大纲"}</strong>
                  <span>规划未来十章与伏笔。</span>
                </span>
                <ArrowUpRight size={12} aria-hidden />
              </button>
            </div>

            <section className="wx-hub-paper wx-home-library-map">
              <div className="wx-home-section-head">
                <div>
                  <span className="wx-home-section-kicker">作品结构</span>
                  <h2>常用工作区</h2>
                </div>
                <span className="wx-home-section-meta">
                  {snapshot.timelineFiles > 0 ? "时间线已建立" : "等待首次记录"}
                </span>
              </div>
              <div className="wx-home-library-map__rows">
                <button type="button" onClick={() => onSwitchTab("characters")}>
                  <span className="wx-home-library-map__icon"><Users size={15} aria-hidden /></span>
                  <span>
                    <strong>角色人物志</strong>
                    <small>{privateMode ? "人物名单已隐藏" : (snapshot.characterNames.join("、") || "尚无角色卡")}</small>
                  </span>
                  <span>{snapshot.characters}</span>
                </button>
                <button type="button" onClick={() => onSwitchTab("reading")}>
                  <span className="wx-home-library-map__icon"><BookOpen size={15} aria-hidden /></span>
                  <span>
                    <strong>沉浸阅读室</strong>
                    <small>{snapshot.finalChapters > 0 ? `${snapshot.finalChapters} 篇定稿可阅读` : "定稿目录尚未建立"}</small>
                  </span>
                  <span>{snapshot.finalChapters}</span>
                </button>
                <button type="button" onClick={() => onSwitchTab("settings")}>
                  <span className="wx-home-library-map__icon"><ScrollText size={15} aria-hidden /></span>
                  <span>
                    <strong>世界设定志</strong>
                    <small>{privateMode ? "设定目录已隐藏" : (snapshot.settingNames.join("、") || "尚无设定文档")}</small>
                  </span>
                  <span>{snapshot.settings}</span>
                </button>
              </div>
            </section>
          </main>

          <aside className="wx-home-project__aside">
            <section className="wx-hub-paper wx-home-side-card">
              <div className="wx-home-side-card__head">
                <h2>最近会话</h2>
                <span>{sessions.length}</span>
              </div>
              {sessions.length > 0 ? (
                <div className="wx-home-session-list">
                  {sessions.slice(0, 5).map((session) => (
                    <button
                      key={session.path}
                      type="button"
                      className={`wx-home-session-row${session.path === currentSessionPath ? " is-current" : ""}`}
                      onClick={() => onResumeSession(session.path, project.path)}
                    >
                      <span className="wx-home-session-row__mark" aria-hidden><MessageSquare size={10} /></span>
                      <span className="wx-home-session-row__copy">
                        <strong>{privateMode ? "会话内容已隐藏" : sessionTitle(session, session.path.split(/[/\\]/).pop() ?? "会话")}</strong>
                        <small>{session.turns} 轮 · {relativeTime(session.modTime)}</small>
                      </span>
                    </button>
                  ))}
                </div>
              ) : (
                <p className="wx-home-side-empty">这部作品还没有会话。可以直接在下方输入框开始。</p>
              )}
              <button type="button" className="wx-home-side-action" onClick={onNewSession}>
                <Plus size={11} aria-hidden />
                新开写作会话
              </button>
            </section>

            <section className="wx-hub-paper wx-home-side-card wx-home-project-note">
              <span className="wx-home-section-kicker">当前作品</span>
              <strong>{project.sessions.length} 个会话</strong>
              <p>正文、人物、设定与会话仅保存在本机作品库。</p>
            </section>
          </aside>
        </div>
      </div>
    </section>
  );
}

export function WenxueHome({
  projects,
  currentProject,
  projectsLoaded,
  recentSessions,
  currentSessionPath,
  quickActions,
  onOpenProject,
  onCreateNovel,
  onOpenHistory,
  onResumeSession,
  onNewSession,
  onInsertPrompt,
  onSwitchTab,
  onOpenLibrary,
  privateMode,
  onTogglePrivacy,
}: {
  projects: ProjectGroup[];
  currentProject?: ProjectGroup;
  projectsLoaded: boolean;
  recentSessions: SessionMeta[];
  currentSessionPath?: string;
  quickActions: WenxueHomeQuickAction[];
  onOpenProject: (path: string) => Promise<string>;
  onCreateNovel: () => void;
  onOpenHistory: () => void;
  onResumeSession: (path: string, projectPath: string) => void;
  onNewSession: () => void;
  onInsertPrompt: (prompt: string) => void;
  onSwitchTab: (tab: "characters" | "reading" | "settings") => void;
  onOpenLibrary: () => void;
  privateMode: boolean;
  onTogglePrivacy: () => void;
}) {
  if (currentProject) {
    return (
      <ProjectHome
        project={currentProject}
        currentSessionPath={currentSessionPath}
        quickActions={quickActions}
        privateMode={privateMode}
        onTogglePrivacy={onTogglePrivacy}
        onInsertPrompt={onInsertPrompt}
        onNewSession={onNewSession}
        onResumeSession={onResumeSession}
        onSwitchTab={onSwitchTab}
        onOpenLibrary={onOpenLibrary}
      />
    );
  }
  return (
    <LibraryHome
      projects={projects}
      recentSessions={recentSessions}
      loaded={projectsLoaded}
      privateMode={privateMode}
      onTogglePrivacy={onTogglePrivacy}
      onOpenProject={onOpenProject}
      onCreateNovel={onCreateNovel}
      onResumeSession={onResumeSession}
      onOpenHistory={onOpenHistory}
    />
  );
}
