package app

// wenxue_workspace.go — 文学板块（图文）工作空间与文件面（2026-11-15 二次改造隔离刀）。
//
// 图文所有工作空间限定在 C:\\AI\\tuwen（每部小说一个子目录=独立工作空间）；
// 本文件族全部为加性刀：方法体 fork 自办公对应绑定（gaeaCwd→wenxueWorkspace），
// 办公共享原语（workbench/wspath/gaeaAttachment/pins/spaces）按路径参数复用，
// 办公行为零改动。会话/聊天引擎仍与办公全共享（早前拍板），仅文件面隔离。

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/docmd"
	gaeaAttachment "github.com/gaea/gaea/internal/gaea/attachment"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/filefilter"
	"github.com/gaea/gaea/internal/gaea/pins"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/workbench"
	"github.com/gaea/gaea/internal/gaea/wspath"
	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/gaea/gaea/internal/office/xlsxpreview"
	"github.com/gaea/gaea/internal/schedule"
)

// wenxueWorkspaceRoot 图文工作空间根（测试可经 GAEA_TUWEN_ROOT 覆盖）。
func wenxueWorkspaceRoot() string {
	if env := os.Getenv("GAEA_TUWEN_ROOT"); env != "" {
		return filepath.Clean(env)
	}
	return `C:\AI\tuwen`
}

func wenxueSamePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// wenxueWorkspace 当前图文工作空间：持久化的小说目录（wenxue_workspace 键），
// 无效/未选时回退图文根本身（确保存在）。
func wenxueWorkspace() string {
	root := wenxueWorkspaceRoot()
	_ = os.MkdirAll(root, 0o755)
	if cfg := config.Load(); cfg != nil && cfg.WenxueWorkspace != "" {
		ws := filepath.Clean(cfg.WenxueWorkspace)
		rel, err := filepath.Rel(root, ws)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if info, err := os.Stat(ws); err == nil && info.IsDir() {
				return ws
			}
		}
	}
	return root
}

// wenxueWorkspaceRelToRoot 把绝对路径转成相对图文根的路径；越界返回空串。
func wenxueWorkspaceRelToRoot(abs string) string {
	rel, err := filepath.Rel(wenxueWorkspaceRoot(), abs)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return rel
}

// wenxuePersistWorkspace 持久化当前小说工作空间（appconfig 键，不动办公工作空间）。
func wenxuePersistWorkspace(abs string) {
	if err := config.Save(config.KeyWenxueWorkspace, abs); err != nil {
		slog.Warn("图文工作空间持久化失败", "error", err)
	}
}

// WenxueListWorkspaces 图文工作空间列表：tuwen 根的子目录（每部小说一个）。
func (a *App) WenxueListWorkspaces() []WorkspaceView {
	root := wenxueWorkspaceRoot()
	_ = os.MkdirAll(root, 0o755)
	current := wenxueWorkspace()
	out := []WorkspaceView{}
	entries, err := os.ReadDir(root)
	if err != nil {
		slog.Warn("图文工作空间根读取失败", "root", root, "error", err)
		return out
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		abs := filepath.Join(root, e.Name())
		out = append(out, WorkspaceView{Path: abs, Name: e.Name(), Current: wenxueSamePath(abs, current)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Current != out[j].Current {
			return out[i].Current
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// WenxueListSessions 返回当前小说工作空间的会话。办公工作空间的会话不会
// 出现在图文侧栏；没有选择小说时返回空列表。
func (a *App) WenxueListSessions() []SessionMeta {
	current := wenxueWorkspace()
	if current == wenxueWorkspaceRoot() {
		return []SessionMeta{}
	}
	curSession := ""
	if c := gaeaCtrl(); c != nil {
		curSession = c.SessionPath()
	}
	return a.listSessionsForDir(gaeaConfig.WorkspaceSessionDir(current, ""), curSession, true)
}

// WenxueListProjectSessions 以小说目录为项目分组。列表只扫描 tuwen 根下的
// 一级子目录，并包含尚未产生会话的新建小说；办公最近工作区不参与。
func (a *App) WenxueListProjectSessions() []ProjectGroup {
	current := wenxueWorkspace()
	curSession := ""
	if c := gaeaCtrl(); c != nil {
		curSession = c.SessionPath()
	}
	groups := make([]ProjectGroup, 0)
	for _, ws := range a.WenxueListWorkspaces() {
		sessions := a.listSessionsForDir(
			gaeaConfig.WorkspaceSessionDir(ws.Path, ""),
			curSession,
			ws.Current,
		)
		archived := a.archivedSessionsForDir(gaeaConfig.WorkspaceSessionDir(ws.Path, ""))
		if len(sessions) > maxProjectSessionsPerGroup {
			sessions = sessions[:maxProjectSessionsPerGroup]
		}
		if len(archived) > maxProjectSessionsPerGroup {
			archived = archived[:maxProjectSessionsPerGroup]
		}
		mod := int64(0)
		for _, s := range sessions {
			if s.ModTime > mod {
				mod = s.ModTime
			}
		}
		for _, s := range archived {
			if s.ModTime > mod {
				mod = s.ModTime
			}
		}
		if mod == 0 {
			if info, err := os.Stat(ws.Path); err == nil {
				mod = info.ModTime().UnixMilli()
			}
		}
		groups = append(groups, ProjectGroup{
			Path:     ws.Path,
			Name:     ws.Name,
			Current:  wenxueSamePath(ws.Path, current),
			Sessions: sessions,
			Archived: archived,
			ModTime:  mod,
		})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Current != groups[j].Current {
			return groups[i].Current
		}
		return groups[i].ModTime > groups[j].ModTime
	})
	return groups
}

// WenxueMeta 返回图文侧看到的会话元信息；工作目录取当前小说，而不是办公
// 工作区。聊天引擎本身仍沿用既有共享控制器。
func (a *App) WenxueMeta() Meta {
	meta := a.GaeaMeta()
	meta.Cwd = wenxueWorkspace()
	return meta
}

// WenxueNewSession 在当前小说目录下新建会话。控制器原本按办公工作区装配；
// 新会话前仅替换新会话落盘目录，不改办公工作区持久化配置。
func (a *App) WenxueNewSession() error {
	ws := wenxueWorkspace()
	if ws == wenxueWorkspaceRoot() {
		return errors.New("请先新建或选择一部小说")
	}
	if err := a.GaeaInit(); err != nil {
		return err
	}
	c := gaeaCtrl()
	if c == nil {
		return errors.New("图文引擎未初始化")
	}
	if c.Running() {
		return errors.New("回合正在运行，请先停止当前回合再新建会话")
	}
	c.SetSessionDir(gaeaConfig.WorkspaceSessionDir(ws, ""))
	return a.GaeaNewSession()
}

// wenxueValidateNovelName 校验小说名（即目录名）：拒绝非法文件名字符与越界形态。
func wenxueValidateNovelName(name string) error {
	n := strings.TrimSpace(name)
	if n == "" {
		return errors.New("小说名不能为空")
	}
	if len([]rune(n)) > 64 {
		return errors.New("小说名过长（上限 64 字）")
	}
	if strings.ContainsAny(n, `\/:*?"<>|`) {
		return errors.New("小说名含非法字符")
	}
	if strings.HasPrefix(n, ".") || strings.HasSuffix(n, ".") || strings.HasSuffix(n, " ") {
		return errors.New("小说名不能以点开头或以点/空格结尾")
	}
	for _, r := range n {
		if r < 32 {
			return errors.New("小说名含控制字符")
		}
	}
	reserved := strings.ToUpper(strings.SplitN(n, ".", 2)[0])
	switch reserved {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return errors.New("小说名是 Windows 保留名称")
	}
	if n == "." || n == ".." {
		return errors.New("小说名非法")
	}
	return nil
}

// WenxueCreateNovel 新建小说工作空间：tuwen 根/<name>（已存在即报错），
// 创建成功后自动切换为当前工作空间。
func (a *App) WenxueCreateNovel(name string) (WorkspaceView, error) {
	n := strings.TrimSpace(name)
	if err := wenxueValidateNovelName(n); err != nil {
		return WorkspaceView{}, err
	}
	root := wenxueWorkspaceRoot()
	_ = os.MkdirAll(root, 0o755)
	dir := filepath.Join(root, n)
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return WorkspaceView{}, errors.New("同名小说已存在：" + n)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return WorkspaceView{}, fmt.Errorf("创建工作空间失败: %w", err)
	}
	slog.Info("图文新建小说工作空间", "dir", dir)
	if got := a.WenxueSwitchWorkspace(dir); got != dir {
		return WorkspaceView{}, errors.New("工作空间切换失败")
	}
	return WorkspaceView{Path: dir, Name: n, Current: true}, nil
}

// WenxueDeleteNovel 删除 tuwen 根下的一部小说及其整个工作空间。
// 仅接受根目录的直接子目录；删除当前小说时先切回根目录，避免后续会话写入
// 已删除路径。
func (a *App) WenxueDeleteNovel(path string) error {
	root := filepath.Clean(wenxueWorkspaceRoot())
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return fmt.Errorf("小说路径无效: %w", err)
	}
	abs = filepath.Clean(abs)
	if wenxueSamePath(abs, root) {
		return errors.New("不能删除 tuwen 根目录")
	}
	if !wenxueSamePath(filepath.Dir(abs), root) {
		return errors.New("只能删除 tuwen 根目录下的小说")
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return fmt.Errorf("小说不存在: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("小说工作空间不是普通目录")
	}
	isCurrent := wenxueSamePath(abs, wenxueWorkspace())
	if isCurrent {
		if c := gaeaCtrl(); c != nil {
			if c.Running() {
				return errors.New("回合正在运行，请先停止当前回合再删除小说")
			}
			c.SetSessionDir(gaeaConfig.WorkspaceSessionDir(root, ""))
			if err := a.GaeaNewSession(); err != nil {
				return fmt.Errorf("退出当前小说失败: %w", err)
			}
		}
		wenxuePersistWorkspace(root)
		wenxueCloseFileTreeWatch()
	}
	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("删除小说失败: %w", err)
	}
	slog.Info("图文删除小说工作空间", "dir", abs, "current", isCurrent)
	return nil
}

// WenxueSwitchWorkspace 切换图文工作空间：只接受 tuwen 根内的目录
// （根本身或其子目录），越界/无效路径保持当前工作空间。
func (a *App) WenxueSwitchWorkspace(path string) string {
	if strings.TrimSpace(path) == "" {
		return wenxueWorkspace()
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return wenxueWorkspace()
	}
	abs = filepath.Clean(abs)
	root := wenxueWorkspaceRoot()
	if !wenxueSamePath(abs, root) && wenxueWorkspaceRelToRoot(abs) == "" {
		slog.Warn("图文工作空间越界（限制在 tuwen 根内）", "path", abs, "root", root)
		return wenxueWorkspace()
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return wenxueWorkspace()
	}
	// 确保小说目录结构就绪（.gaea 子目录懒建）
	_ = os.MkdirAll(filepath.Join(abs, ".gaea"), 0o755)
	wenxuePersistWorkspace(abs)
	// 释放图文文件树监视（旧根句柄），树重开时按新根重建。
	wenxueCloseFileTreeWatch()
	if c := gaeaCtrl(); c != nil && !c.Running() {
		c.SetSessionDir(gaeaConfig.WorkspaceSessionDir(abs, ""))
	}
	return abs
}

// WenxuePickWorkspace 图文不提供任意目录选择（工作空间限定 tuwen 根）；
// 保留绑定仅返回空串（UI 已移除入口，防旧调用路径误触）。
func (a *App) WenxuePickWorkspace() string {
	slog.Info("WenxuePickWorkspace 不可用：图文工作空间限定在 tuwen 根（用 WenxueCreateNovel 新建）")
	return ""
}

// ── 图文文件树监视槽（与办公 officeState.fileTreeWatch 物理分离）────

var (
	wenxueFileTreeMu    sync.Mutex
	wenxueFileTreeWatch *workbench.DirWatch
)

// wenxueFileTreeEventChannel 图文文件树变更专用事件通道。
const wenxueFileTreeEventChannel = "gaea-wenxue-filetree"

// wenxueCloseFileTreeWatch 关闭并清空图文文件树监视（切换工作空间时调用）。
func wenxueCloseFileTreeWatch() {
	wenxueFileTreeMu.Lock()
	w := wenxueFileTreeWatch
	wenxueFileTreeWatch = nil
	wenxueFileTreeMu.Unlock()
	if w != nil {
		_ = w.Close()
	}
}

// WenxueFileTreeWatch replaces the watched directory set with exactly the levels
// the tree currently has expanded.
func (a *App) WenxueFileTreeWatch(paths, exclude []string) error {
	wenxueFileTreeMu.Lock()
	old := wenxueFileTreeWatch
	wenxueFileTreeWatch = nil
	wenxueFileTreeMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	if len(paths) == 0 {
		return nil
	}
	w, err := workbench.NewDirWatch(wenxueWorkspace(), paths, exclude, func(changed []string) {
		a.emit(wenxueFileTreeEventChannel, map[string]interface{}{"paths": changed})
	})
	if err != nil {
		return err
	}
	wenxueFileTreeMu.Lock()
	if wenxueFileTreeWatch != nil {
		_ = wenxueFileTreeWatch.Close()
	}
	wenxueFileTreeWatch = w
	wenxueFileTreeMu.Unlock()
	return nil
}

// ── 以下为办公对应绑定的 fork（wenxueWorkspace 换根）─────────────

// WenxueAttachmentChunkBegin 开一个分块上传会话，返回会话 token。
func (a *App) WenxueAttachmentChunkBegin(fileName string) (string, error) {
	return gaeaAttachment.BeginChunk(filepath.Join(gaeaCwd(), ".gaea", "uploads"), fileName)
}

// WenxueAttachmentDataURL 读取附件为 dataURL。
// 2026-09-19 审计 P1 收口：无约束任意路径读盘转 base64 → 限定可读根
// （工作区 + 数据根：剧照 portraits/绘梦资产/附件 uploads 均落在这两根内）
// + 32MB 上限（头像/资产/截图远小于此，防注入读大文件撑爆内存）。
func (a *App) WenxueAttachmentDataURL(path string) (string, error) {
	resolved := wenxueResolveAgainstWorkspace(path)
	if !a.wenxueWithinReadRoots(resolved) {
		return "", fmt.Errorf("路径不在可读数据范围内（工作区/应用数据根）: %s", path)
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	if len(b) > 32<<20 {
		return "", fmt.Errorf("文件过大（上限 32MB）: %s", path)
	}
	mime := "application/octet-stream"
	switch strings.ToLower(filepath.Ext(resolved)) {
	case ".png":
		mime = "image/png"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	case ".pdf":
		mime = "application/pdf"
	case ".txt":
		mime = "text/plain"
	case ".md":
		mime = "text/markdown"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}

// WenxueConvertToPdf 把工作区文档转换为 PDF（LibreOffice 无头转换），
// PDF 落 .gaea/exports/ 并返回相对路径。md/markdown 先经 create_docx.py
// 出 docx 再转 PDF（「任何文档 → PDF」统一出口）。
func (a *App) WenxueConvertToPdf(rel string) (ConvertPdfResult, error) {
	if rel == "" {
		return ConvertPdfResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		// v4.363：相对路径拒 .. 穿越；审计 2026-10-02 AP5-09：判据改调唯一原语
		// wspath.ResolveRelWithin（Clean + 包含性检查，与全仓同类站点同源）。
		p, err := wspath.ResolveRelWithin(gaeaCwd(), rel)
		if err != nil {
			return ConvertPdfResult{}, fmt.Errorf("非法工作区相对路径: %s", rel)
		}
		path = p
	}
	// v4.363：读侧收口——本绑定是「读任意路径喂外部转换进程」的原语，此前
	// 绝对路径直通无约束（v4.358 审计挂池）。统一门 = 工作区/数据根白名单
	// ∪ 文件对话框登记（附件 chip 的工作区外素材正是经 GaeaPickFiles 进来、
	// 登记时已入表），fail-closed 对齐 GaeaReadFileB64。写侧 exports 目录
	// 与文件名全服务端拼装，无越界面。
	if !a.wenxueWithinReadRoots(path) && !isPickedFile(path) {
		return ConvertPdfResult{}, fmt.Errorf("文件不在可读范围（工作区/数据根），且未经文件对话框选取: %s", rel)
	}
	if _, err := os.Stat(path); err != nil {
		return ConvertPdfResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	ext := strings.ToLower(filepath.Ext(path))

	src := path
	tmpDir := ""
	switch ext {
	case ".md", ".markdown":
		// Markdown → docx（复用交付模板链路）→ PDF
		raw, err := os.ReadFile(path)
		if err != nil {
			return ConvertPdfResult{}, err
		}
		var err2 error
		tmpDir, err2 = os.MkdirTemp("", "gaea-pdf-*")
		if err2 != nil {
			return ConvertPdfResult{}, err2
		}
		base := safeDeliverableName(strings.TrimSuffix(filepath.Base(path), ext))
		docxPath := filepath.Join(tmpDir, base+".docx")
		in := ExportDeliverableInput{
			Markdown: string(raw),
			Format:   "docx",
			Title:    base,
			Template: "通用",
			Footer:   "第 {page} 页",
		}
		if err := wenxueExportDocx(in, base, docxPath); err != nil {
			_ = os.RemoveAll(tmpDir)
			return ConvertPdfResult{}, err
		}
		src = docxPath
	case ".doc":
		return ConvertPdfResult{}, fmt.Errorf("暂不支持 .doc 转 PDF（请先另存为 .docx）")
	default:
		if !pdfDirectExts[ext] {
			return ConvertPdfResult{}, fmt.Errorf(
				"暂不支持将 %s 转换为 PDF（支持 docx/xlsx/pptx/odt/html/txt/csv/md）", ext)
		}
	}
	if tmpDir != "" {
		defer func() { _ = os.RemoveAll(tmpDir) }()
	}

	// S4 产物路径分区：work 恒 .gaea/exports（现状不动），play 落 .gaea/play/exports。
	exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return ConvertPdfResult{}, err
	}
	srcBase := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	outPath := filepath.Join(exportsDir, safeDeliverableName(srcBase)+"-"+time.Now().Format("20060102-150405")+".pdf")
	if err := convertToPdfFile(src, outPath); err != nil {
		return ConvertPdfResult{}, err
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return ConvertPdfResult{}, err
	}
	outRel, _ := filepath.Rel(gaeaCwd(), outPath)
	srcRel, _ := filepath.Rel(gaeaCwd(), path)
	return ConvertPdfResult{
		Path:   filepath.ToSlash(outRel),
		Name:   filepath.Base(outPath),
		Size:   info.Size(),
		Source: filepath.ToSlash(srcRel),
	}, nil
}

// WenxueCreateWorkspaceFile creates an empty file inside an existing row.
func (a *App) WenxueCreateWorkspaceFile(parent, name string) (workbench.MutationResult, error) {
	return workbench.CreateFile(gaeaCwd(), parent, name)
}

// WenxueCreateWorkspaceFolder creates a directory inside an existing row.
func (a *App) WenxueCreateWorkspaceFolder(parent, name string) (workbench.MutationResult, error) {
	return workbench.CreateFolder(gaeaCwd(), parent, name)
}

// WenxueDeleteWorkspacePaths permanently removes selected rows.
func (a *App) WenxueDeleteWorkspacePaths(paths []string) error {
	return workbench.DeletePaths(gaeaCwd(), paths)
}

// WenxueExportDeliverable 统一交付出口：事实底座/对话成果的受控 Markdown →
// docx / pptx / xlsx / md / pdf，一稿多用、多形态一致。
func (a *App) WenxueExportDeliverable(in ExportDeliverableInput) (ExportDeliverableResult, error) {
	if strings.TrimSpace(in.Markdown) == "" {
		return ExportDeliverableResult{}, fmt.Errorf("交付内容为空")
	}
	format := strings.ToLower(strings.TrimPrefix(in.Format, "."))
	switch format {
	case "docx", "pptx", "xlsx", "md", "pdf":
	default:
		return ExportDeliverableResult{}, fmt.Errorf("不支持的交付格式 %q（docx/pptx/xlsx/md/pdf）", in.Format)
	}
	if in.Template == "" {
		in.Template = "通用"
	}
	switch in.Template {
	case "通用", "公文", "报告", "合同":
	default:
		return ExportDeliverableResult{}, fmt.Errorf("不支持的模板 %q（通用/公文/报告/合同）", in.Template)
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		if m := headingRe.FindStringSubmatch(in.Markdown); m != nil {
			title = strings.TrimSpace(m[1])
		} else {
			title = "gaea 交付物"
		}
	}
	if in.Footer == "" {
		in.Footer = "第 {page} 页"
	}

	// 输出路径双分支（P1a 声明入口）：OutputPath 非空=声明路径（工作区内校验 +
	// 扩展名与 format 一致闸，声明路径=实际落点逐字节一致）；空=历史默认分支
	// （S4 产物路径分区：work 恒 .gaea/exports 现状不动，play 落 .gaea/play/exports）。
	var outPath string
	if strings.TrimSpace(in.OutputPath) != "" {
		p, perr := wspath.ResolveRelWithin(gaeaCwd(), in.OutputPath)
		if perr != nil {
			return ExportDeliverableResult{}, fmt.Errorf("非法 output_path（须为工作区相对路径）: %s", in.OutputPath)
		}
		if ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(p), ".")); ext != format {
			return ExportDeliverableResult{}, fmt.Errorf("output_path 扩展名 %q 与 format %q 不一致", ext, format)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return ExportDeliverableResult{}, err
		}
		outPath = p
	} else {
		exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
		if err := os.MkdirAll(exportsDir, 0o755); err != nil {
			return ExportDeliverableResult{}, err
		}
		stamp := time.Now().Format("20060102-150405")
		base := safeDeliverableName(title)
		outPath = filepath.Join(exportsDir, base+"-"+stamp+"."+format)
	}

	switch format {
	case "md":
		if err := os.WriteFile(outPath, []byte(in.Markdown), 0o644); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "docx":
		if err := wenxueExportDocx(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "pptx":
		if err := wenxueExportPptx(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "xlsx":
		if err := exportXlsx(in, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "pdf":
		// Markdown → docx（临时，复用交付模板链路）→ LibreOffice PDF
		if err := wenxueMarkdownToPdf(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return ExportDeliverableResult{}, err
	}
	rel, _ := filepath.Rel(gaeaCwd(), outPath)
	return ExportDeliverableResult{
		Path:   filepath.ToSlash(rel),
		Name:   filepath.Base(outPath),
		Format: format,
		Size:   info.Size(),
	}, nil
}

// WenxueFileSearch 工作区文件名搜索（跨目录定位资料）：按名称子串匹配、
// 不区分大小写，限制深度与数量，与文件树共用默认/用户排除规则。
func (a *App) WenxueFileSearch(query string, limit int, exclude []string) []FileSearchHit {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := strings.ToLower(strings.TrimSpace(query))
	root := gaeaCwd()
	matcher := filefilter.Compile(exclude)
	out := []FileSearchHit{}
	matched := 0

	var walk func(dir, rel string, depth int)
	walk = func(dir, rel string, depth int) {
		if matched >= limit || depth > 6 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if matched >= limit {
				return
			}
			name := e.Name()
			relPath := name
			if rel != "" {
				relPath = rel + "/" + name
			}
			if matcher.Match(relPath, name) {
				continue
			}
			if e.IsDir() {
				if q == "" || strings.Contains(strings.ToLower(name), q) {
					out = append(out, FileSearchHit{Path: relPath, Name: name, IsDir: true})
					matched++
				}
				walk(filepath.Join(dir, name), relPath, depth+1)
				continue
			}
			if q == "" || strings.Contains(strings.ToLower(name), q) {
				info, _ := e.Info()
				size := int64(0)
				if info != nil {
					size = info.Size()
				}
				out = append(out, FileSearchHit{Path: relPath, Name: name, IsDir: false, Size: size})
				matched++
			}
		}
	}
	walk(root, "", 0)
	return out
}

// WenxueGitCommit 提交暂存区（不代 add：暂存什么提交什么，与源一致）；
// 返回新提交短 hash。
func (a *App) WenxueGitCommit(message string) (string, error) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return "", errGit("提交说明为空")
	}
	if _, err := gitRunWrite(wenxueGitWorkspace(), "commit", "-m", msg); err != nil {
		return "", err
	}
	out, err := gitRunAt(wenxueGitWorkspace(), "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", nil // 提交已成功，短 hash 拿不到不回滚提交（诚实降级）
	}
	return strings.TrimSpace(out), nil
}

// WenxueGitCommitDetail 返回单个提交详情：消息/引用/numstat 文件统计/全量
// unified diff（diff 走读预算与截断标记）。合并提交 diff-tree 默认无输出
// → Diff 为空串（前端诚实提示）。hash 只接受十六进制 rev（防注入）。
func (a *App) WenxueGitCommitDetail(hash string) (GitCommitDetail, error) {
	hash = strings.TrimSpace(strings.ToLower(hash))
	if !gitHashSafe(hash) {
		return GitCommitDetail{}, errGit("hash 不合法")
	}
	repoRoot, err := gitRepoRoot(wenxueGitWorkspace())
	if err != nil {
		return GitCommitDetail{}, err
	}
	detail := GitCommitDetail{Hash: hash, Files: []GitCommitFile{}}

	// 消息/引用/parents：单次 log 定长解析（%b=消息体，可跨行）
	out, err := gitRunAt(repoRoot, "-c", "core.quotepath=false", "log", "-n", "1",
		"--pretty=format:%H%x1f%s%x1f%an%x1f%at%x1f%P%x1f%D%x1f%b", hash, "--")
	if err != nil {
		return GitCommitDetail{}, err
	}
	parts := strings.Split(out, "\x1f")
	if len(parts) > 0 {
		detail.Hash = parts[0]
	}
	if len(parts) > 1 {
		detail.Subject = parts[1]
	}
	if len(parts) > 2 {
		detail.Author = parts[2]
	}
	if len(parts) > 3 {
		detail.Ts, _ = strconv.ParseInt(parts[3], 10, 64)
	}
	if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
		detail.Parents = strings.Fields(parts[4])
	}
	if len(parts) > 5 {
		detail.Refs = parseGitRefs(parts[5])
	}
	if len(parts) > 6 {
		detail.Body = strings.TrimSpace(parts[6])
	}

	// numstat：--root 使首提交也出统计；二进制行为 "-\t-" → -1（前端显「二进制」）
	stats, err := gitRunAt(repoRoot, "-c", "core.quotepath=false", "diff-tree", "-r", "--root", "--no-commit-id", "--numstat", hash)
	if err != nil {
		return GitCommitDetail{}, err
	}
	for _, line := range strings.Split(stats, "\n") {
		fields := strings.SplitN(strings.TrimRight(line, "\r"), "\t", 3)
		if len(fields) != 3 || strings.TrimSpace(fields[0]) == "" {
			continue
		}
		f := GitCommitFile{Path: fields[2]}
		if fields[0] == "-" || fields[1] == "-" {
			f.Add, f.Del = -1, -1
		} else {
			f.Add, _ = strconv.Atoi(fields[0])
			f.Del, _ = strconv.Atoi(fields[1])
		}
		detail.Files = append(detail.Files, f)
	}

	// diff：展示层预算（超预算=截断标记，不当错误炸面板）
	diffOut, derr := gitRunBudgeted(repoRoot, gitReadTimeout, gitDiffBudget,
		"-c", "core.quotepath=false", "diff-tree", "-p", "--root", "--no-commit-id", hash)
	if errors.Is(derr, errGitOutputBudget) {
		diffOut += "\n\n[diff 输出超过 4MB，已截断]"
	} else if derr != nil {
		return GitCommitDetail{}, derr
	}
	detail.Diff = diffOut
	return detail, nil
}

// WenxueGitDiff 返回单个已跟踪文件的 unified diff 文本（staged=true 取暂存区
// 与 HEAD 的差异）。未跟踪文件没有 diff 语义，前端走内容预览。
func (a *App) WenxueGitDiff(path string, staged bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errGit("path 为空")
	}
	// --no-ext-diff/--no-textconv（v4.375，Reasonix gitcmd 基线）：外置 diff
	// 驱动与 textconv 会执行 .gitattributes 配置的外部程序，查看面板 diff
	// 时一律关闭（clean filter 由 wenxueGitRun 层统一中和）。
	args := []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	out, err := gitRunBudgeted(wenxueGitWorkspace(), gitReadTimeout, gitDiffBudget, args...)
	if errors.Is(err, errGitOutputBudget) {
		// 展示层诚实截断：保留已收内容 + 截断标记，不当错误炸面板。
		return out + "\n\n[diff 输出超过 4MB，已截断]", nil
	}
	return out, err
}

// WenxueGitDiscard 丢弃单个文件的工作区改动（危险操作，前端两击确认；
// 只接受已跟踪文件，未跟踪文件走删除语义不在本刀）。
func (a *App) WenxueGitDiscard(path string) error {
	if strings.TrimSpace(path) == "" {
		return errGit("path 为空")
	}
	_, err := gitRunWrite(wenxueGitWorkspace(), "checkout", "-q", "--", path)
	return err
}

// WenxueGitGraph 返回提交图一页：分页提交（含 parents/装饰，lane 拓扑由前端
// 纯函数计算）+ 本地分支清单。branch 非空时按该分支取历史（严格校验，exec
// 列表无 shell 面，此处防的是 rev/选项注入）。只读：不 checkout/fetch/push。
func (a *App) WenxueGitGraph(skip, limit int, branch string) (GitGraphPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if skip < 0 {
		skip = 0
	}
	branch = strings.TrimSpace(branch)
	if branch != "" && !gitRefSafe(branch) {
		return GitGraphPage{}, errGit("分支名不合法")
	}
	repoRoot, err := gitRepoRoot(wenxueGitWorkspace())
	if err != nil {
		return GitGraphPage{}, err
	}
	argv := []string{"-c", "core.quotepath=false", "log",
		"-n", strconv.Itoa(limit), "--skip", strconv.Itoa(skip),
		"--pretty=format:%H%x1f%h%x1f%s%x1f%an%x1f%at%x1f%P%x1f%D%x1e"}
	if branch != "" {
		argv = append(argv, branch, "--")
	}
	out, err := gitRunAt(repoRoot, argv...)
	if err != nil {
		return GitGraphPage{}, err
	}
	page := GitGraphPage{Commits: []GitGraphNode{}}
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.TrimLeft(rec, "\n")
		if strings.TrimSpace(rec) == "" {
			continue
		}
		parts := strings.Split(rec, "\x1f")
		c := GitGraphNode{}
		if len(parts) > 0 {
			c.Hash = parts[0]
		}
		if len(parts) > 1 {
			c.Short = parts[1]
		}
		if len(parts) > 2 {
			c.Subject = parts[2]
		}
		if len(parts) > 3 {
			c.Author = parts[3]
		}
		if len(parts) > 4 {
			c.Ts, _ = strconv.ParseInt(parts[4], 10, 64)
		}
		if len(parts) > 5 && strings.TrimSpace(parts[5]) != "" {
			c.Parents = strings.Fields(parts[5])
		}
		if len(parts) > 6 {
			c.Refs = parseGitRefs(parts[6])
		}
		page.Commits = append(page.Commits, c)
	}
	branches, err := gitLocalBranches(repoRoot)
	if err != nil {
		return GitGraphPage{}, err
	}
	page.Branches = branches
	return page, nil
}

// WenxueGitLog 返回最近提交历史（不含 push 相关语义）。
func (a *App) WenxueGitLog(limit int) ([]GitCommitInfo, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	out, err := wenxueGitRun("-c", "core.quotepath=false", "log", "-n", strconv.Itoa(limit),
		"--pretty=format:%h%x1f%s%x1f%an%x1f%at%x1e")
	if err != nil {
		return nil, err
	}
	commits := []GitCommitInfo{}
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.TrimLeft(rec, "\n")
		if strings.TrimSpace(rec) == "" {
			continue
		}
		parts := strings.Split(rec, "\x1f")
		c := GitCommitInfo{}
		if len(parts) > 0 {
			c.Hash = parts[0]
		}
		if len(parts) > 1 {
			c.Subject = parts[1]
		}
		if len(parts) > 2 {
			c.Author = parts[2]
		}
		if len(parts) > 3 {
			c.Ts, _ = strconv.ParseInt(parts[3], 10, 64)
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// WenxueGitStage 把文件加入暂存区（新增文件同样适用）。
func (a *App) WenxueGitStage(paths []string) error {
	clean := nonEmptyPaths(paths)
	if len(clean) == 0 {
		return errGit("未选择文件")
	}
	_, err := gitRunWrite(wenxueGitWorkspace(), append([]string{"add", "--"}, clean...)...)
	return err
}

// WenxueGitStatus 返回仓库状态：分支 + ahead/behind + 文件列表（porcelain v1），
// 并标注工作区是否位于 linked worktree（G1 深化）。
func (a *App) WenxueGitStatus() GitStatus {
	workspace := wenxueGitWorkspace()
	repoRoot, err := gitRepoRoot(workspace)
	if err != nil {
		return GitStatus{Files: []GitFileStatus{}, Error: err.Error()}
	}
	out, err := gitRunAt(repoRoot, "status", "--porcelain=v1", "-z", "-b")
	if err != nil {
		return GitStatus{Files: []GitFileStatus{}, Error: err.Error()}
	}
	st := GitStatus{Files: []GitFileStatus{}}
	records := strings.Split(out, "\x00")
	for i := 0; i < len(records); i++ {
		line := strings.TrimRight(records[i], "\r\n")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			st.Branch, st.Ahead, st.Behind = parseGitStatusBranch(line)
			continue
		}
		if len(line) < 4 {
			continue
		}
		x, y := string(line[0]), string(line[1])
		path := line[3:]
		var origPath string
		if x == "R" || x == "C" || y == "R" || y == "C" {
			// porcelain -z stores the destination first, followed by the source
			// path as its own NUL-delimited field（G1.7 起保留来源供前端溯源）。
			if i+1 < len(records) {
				i++
				origPath = strings.TrimRight(records[i], "\r\n")
			}
		}
		path, ok := workspaceStatusPath(repoRoot, workspace, path)
		if !ok {
			continue
		}
		f := GitFileStatus{Path: path, X: x, Y: y}
		f.Untracked = x == "?" && y == "?"
		f.Conflict = isConflictXY(x, y)
		f.Staged = !f.Untracked && !f.Conflict && x != " "
		f.Deleted = y == "D"
		f.Modified = y == "M"
		f.Renamed = x == "R" || y == "R"
		// 来源路径走同一工作区归一/过滤口径：工作区外来源不上树（不溯源）。
		if origPath != "" {
			if o, ok2 := workspaceStatusPath(repoRoot, workspace, origPath); ok2 {
				f.OrigPath = o
			}
		}
		st.Files = append(st.Files, f)
	}
	st.IsRepo = true
	// linked worktree 判定（G1 深化）：--absolute-git-dir 与
	// --git-common-dir 不同 → 工作区在 linked worktree（非主检出）。
	// --path-format=absolute 需 git ≥2.31，取不到时尽力而为不标注，不阻塞状态面。
	if gitDir, err := gitRunAt(workspace, "rev-parse", "--absolute-git-dir"); err == nil {
		if commonDir, err2 := gitRunAt(workspace, "rev-parse", "--path-format=absolute", "--git-common-dir"); err2 == nil {
			gitDir = strings.TrimSpace(strings.Split(gitDir, "\n")[0])
			commonDir = strings.TrimSpace(strings.Split(commonDir, "\n")[0])
			if gitDir != "" && commonDir != "" && !sameDirPath(gitDir, commonDir) {
				st.IsLinkedWorktree = true
				st.MainWorktreePath = filepath.Dir(filepath.FromSlash(commonDir))
			}
		}
	}
	return st
}

// WenxueGitUnstage 把文件移出暂存区（不动工作区内容；首条提交前仓库无 HEAD
// 时 git 会报错，原样透出为诚实错误）。
func (a *App) WenxueGitUnstage(paths []string) error {
	clean := nonEmptyPaths(paths)
	if len(clean) == 0 {
		return errGit("未选择文件")
	}
	_, err := gitRunWrite(wenxueGitWorkspace(), append([]string{"reset", "-q", "HEAD", "--"}, clean...)...)
	return err
}

// WenxueGitWorktrees 列出仓库全部 worktree（只读展示）。gaea 全应用单一
// workspace 锚（ga.cfg.Workspace），FileTree/Git/Agent 共用它——worktree
// 「切换」= 工作区切换器（G2/W1 级产品门），不在本刀；也不做 add/remove/prune。
func (a *App) WenxueGitWorktrees() ([]GitWorktree, error) {
	workspace := wenxueGitWorkspace()
	repoRoot, err := gitRepoRoot(workspace)
	if err != nil {
		return nil, err
	}
	out, err := gitRunAt(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseGitWorktrees(out, repoRoot), nil
}

// WenxueListDir 列出工作区目录：rel 为空 = 工作区根，相对路径 Join 工作区根
// （旧行为不变），绝对路径直接使用（v4.96 登记缺口补齐）。错误不再吞：
// 带 GAEADIR_* 结构化错误码透传（实现与码表见 gaea_listdir.go）。签名例外
// （[]DirEntry → ([]DirEntry, error)，主代理拍板）：方法名/绑定数 581 不变，
// 失败从 resolve 空切片改为 reject 错误串（前端调用点均有 catch）。
func (a *App) WenxueListDir(rel string) ([]DirEntry, error) {
	return wenxueListDirEntries(rel)
}

// WenxueListDirBatch lists all expanded tree levels in one binding round trip.
func (a *App) WenxueListDirBatch(paths, exclude []string) []workbench.Level {
	return workbench.ListBatch(gaeaCwd(), paths, exclude)
}

// WenxueMaterials 工作区资料概览：office/文本类文件按修改时间倒序返回
// （开工前「有哪些资料可用」视图，配合一键 @ 引用）。
func (a *App) WenxueMaterials(limit int) []FileSearchHit {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	root := gaeaCwd()
	matcher := filefilter.Compile(nil)
	out := []FileSearchHit{}
	var walk func(dir, rel string, depth int)
	walk = func(dir, rel string, depth int) {
		if len(out) >= limit || depth > 5 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if len(out) >= limit {
				return
			}
			name := e.Name()
			relPath := name
			if rel != "" {
				relPath = rel + "/" + name
			}
			if matcher.Match(relPath, name) {
				continue
			}
			if e.IsDir() {
				dirRel := name
				if rel != "" {
					dirRel = rel + "/" + name
				}
				walk(filepath.Join(dir, name), dirRel, depth+1)
				continue
			}
			if !materialExts[strings.ToLower(filepath.Ext(name))] {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			out = append(out, FileSearchHit{
				Path:    relPath,
				Name:    name,
				IsDir:   false,
				Size:    info.Size(),
				ModTime: info.ModTime().UnixMilli(),
			})
		}
	}
	walk(root, "", 0)
	// 修改时间倒序（最新在前）
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ModTime > out[j-1].ModTime; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// WenxueOpenWorkspacePath/WenxueRevealWorkspacePath 在文件管理器中打开/定位。
func (a *App) WenxueOpenWorkspacePath(rel string) error {
	path, _ := wenxueResolvePreviewPath(rel)
	return exec.Command("explorer", path).Start()
}

// WenxuePickWorkspaceUploadFiles chooses files natively and copies them into the
// tree without sending bytes through the JS bridge.
func (a *App) WenxuePickWorkspaceUploadFiles(dir string) ([]workbench.UploadResult, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("应用未就绪")
	}
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "上传文件到工作区",
		DefaultDirectory: gaeaCwd(),
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return []workbench.UploadResult{}, nil
	}
	return workbench.ImportFiles(gaeaCwd(), dir, files)
}

// WenxuePickWorkspaceUploadFolder chooses one directory natively and recursively
// copies it into the tree without bridging its bytes through JavaScript.
func (a *App) WenxuePickWorkspaceUploadFolder(dir string) ([]workbench.UploadResult, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("应用未就绪")
	}
	source, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "上传文件夹到工作区",
		DefaultDirectory: gaeaCwd(),
	})
	if err != nil {
		return nil, err
	}
	if source == "" {
		return []workbench.UploadResult{}, nil
	}
	return workbench.ImportFolder(gaeaCwd(), dir, source)
}

// WenxuePinMaterial 固定一个工作区相对路径文件，返回更新后的固定清单。
func (a *App) WenxuePinMaterial(rel string) []FileSearchHit {
	if _, err := pins.Add(gaeaCwd(), rel); err != nil {
		return []FileSearchHit{}
	}
	return pinnedView(gaeaCwd())
}

// WenxuePinnedMaterials 返回当前工作区的固定常用资料（P1-②）。
// 固定后的文件在新会话启动时自动带入系统提示词（见 boot 系统提示词组装）。
func (a *App) WenxuePinnedMaterials() []FileSearchHit {
	return pinnedView(gaeaCwd())
}

// WenxuePreview 读取工作区相对路径并返回结构化预览。
func (a *App) WenxuePreview(rel string) PreviewResult {
	path, displayRel := wenxueResolvePreviewPath(rel)
	info, err := os.Stat(path)
	if err != nil {
		return PreviewResult{Path: displayRel, Kind: "error", Error: "文件不存在"}
	}
	if info.IsDir() {
		return PreviewResult{Path: displayRel, Kind: "error", Error: "目录无法预览"}
	}

	ext := strings.ToLower(filepath.Ext(rel))
	name := filepath.Base(rel)
	base := PreviewResult{
		Path: displayRel, Name: name, Ext: ext, Size: info.Size(),
		MTimeMs: info.ModTime().UnixMilli(),
	}

	if imageExts[ext] {
		// 32MB 封顶（2026-09-19 审计：整文件读入 base64 无上限=可被驱动读
		// 大文件撑爆内存；头像/截图/图产远小于此）。
		if info.Size() > 32<<20 {
			base.Kind = "error"
			base.Error = "图片过大（上限 32MB）"
			return base
		}
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "image"
		base.DataURL = "data:" + mimeFor(ext) + ";base64," + base64.StdEncoding.EncodeToString(b)
		return base
	}

	switch ext {
	case ".html", ".htm":
		// 1c HTML 沙箱预览：原文（截断口径与 markdown 同款）交前端沙箱
		// iframe 渲染（sandbox 无 allow-same-origin + CSP 白名单），绝不
		// 注入宿主 DOM；此前 .html 落 textExts 当纯文本展示。
		body, truncated, encoding, eol, err := readPreviewCapped(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "html"
		base.Body = body
		base.Truncated = truncated
		base.Encoding = encoding
		base.EOL = eol
		return base
	case ".md", ".markdown":
		body, truncated, encoding, eol, err := readPreviewCapped(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = previewTruncateNote(body, truncated)
		base.Truncated = truncated
		base.Encoding = encoding
		base.EOL = eol
		return base
	case ".mmd", ".mermaid":
		// Mermaid 图表源码：按 markdown 渲染，聊天/预览中的 ```mermaid
		// 代码块会直接渲染成图。
		body, truncated, encoding, eol, err := readPreviewCapped(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = "```mermaid\n" + body + "\n```" + previewTruncateNote("", truncated)
		base.Truncated = truncated
		base.Encoding = encoding
		base.EOL = eol
		return base
	case ".docx":
		// 原始 docx 交给前端 docx-preview 保真渲染（版式/表格/页眉页脚/修订）；
		// Body 同时附带轻量 Markdown 文本（截断头部），供交付卡片缩略图
		// 显示"看得见的文件内容"，完整版式仍走 dataUrl。
		if info.Size() > 32<<20 {
			base.Kind = "error"
			base.Error = "文档过大（上限 32MB）"
			return base
		}
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "docx"
		base.DataURL = "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64," +
			base64.StdEncoding.EncodeToString(b)
		if md, mdErr := docmd.Convert(path, ""); mdErr == nil {
			base.Body = previewThumbText(md)
		}
		return base
	case ".xlsx":
		// 公式无缓存值时先补一次 LibreOffice 重算，保证预览显示计算结果
		if need, _ := xlsxpreview.NeedsRecalc(path); need {
			_, _ = xlsxedit.Recalc(path, gaeaCwd())
		}
		j, err := xlsxpreview.Render(path)
		if err != nil {
			base.Kind = "unsupported"
			base.Error = err.Error()
			return base
		}
		base.Kind = "xlsx"
		base.Body = j
		return base
	case ".pptx":
		// v4.28 B2：pptx → soffice PDF（落 .gaea/cache 缓存）→ poppler 逐页
		// 缩略；kind=pdf + Pages + Hint="outline"（前端叠大纲卡）。实现见
		// gaea_pptx.go previewPptx。
		return previewPptx(path, info, base)
	case ".doc", ".xls", ".pdf":
		// 扫描件 OCR 逐页进度经事件通道回传前端（仅 Wails/HTTP 桥接环境真正发布）。
		// elapsedSec 与页数并显：批窗间无事件的空档期给时间感（同 knowledge-import-progress）。
		progressStart := time.Now()
		md, total, truncated, err := docmd.ConvertLimitProgress(path, "", docmd.DefaultMaxPDFPages,
			func(done, n int) {
				if n <= 0 {
					return
				}
				a.emit("gaea-event", map[string]interface{}{
					"kind": "preview_progress",
					"path": rel,
					"progress": map[string]interface{}{
						"path":       rel,
						"done":       done,
						"total":      n,
						"elapsedSec": int(time.Since(progressStart).Seconds()),
					},
				})
			})
		if err != nil {
			base.Kind = "unsupported"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = md
		base.TotalPages = total
		base.Truncated = truncated
		if truncated {
			base.Body += fmt.Sprintf("\n\n> ⚠️ 预览已截断：PDF 共 %d 页，仅显示前 %d 页。可让 AI 使用 summarize_file 获取全文摘要，或拆分文件后处理。",
				total, docmd.DefaultMaxPDFPages)
		}
		return base
	}

	if textExts[ext] {
		body, truncated, encoding, eol, err := readPreviewCapped(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "text"
		base.Body = previewTruncateNote(body, truncated)
		base.Truncated = truncated
		base.Encoding = encoding
		base.EOL = eol
		return base
	}

	base.Kind = "unsupported"
	base.Error = "该格式暂不支持内联预览，可点击右上角在外部程序中打开"
	return base
}

// WenxueReadFile 读取工作区相对路径的文件文本。
// 2026-09-19 审计 P1：原实现无任何路径约束（写端 GaeaWriteFile 有完整防线而
// 读端是没上锁的孪生兄弟）——相对路径穿越/绝对路径可读任意文件且无大小上限。
// 现对齐写端口径：拒绝绝对/.. 穿越 + wenxueWithinWriteRoots + maxTextEditBytes 截断。
func (a *App) WenxueReadFile(rel string) FilePreview {
	rel = strings.TrimSpace(rel)
	clean := filepath.Clean(filepath.FromSlash(rel))
	if rel == "" || clean == "." {
		return FilePreview{Path: rel, Markdown: "非法工作区相对路径: " + rel}
	}
	// 审计 2026-10-02 AP5-09：判据改调唯一原语（原为手写 IsAbs + Clean +
	// `..` 前缀/片段，与 GaeaWriteFile 逐字重复的两份之一）；绝对/卷相对/
	// 穿越一律由原语拒绝，措辞不变。
	abs, err := wspath.ResolveRelWithin(gaeaCwd(), rel)
	if err != nil {
		return FilePreview{Path: rel, Markdown: "非法工作区相对路径: " + rel}
	}
	if !wenxueWithinWriteRoots(abs) {
		return FilePreview{Path: rel, Markdown: "路径不在可读范围内（工作区/allow_write）: " + rel}
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return FilePreview{Path: rel}
	}
	truncated := false
	if len(b) > maxTextEditBytes {
		b = b[:maxTextEditBytes]
		truncated = true
	}
	body := string(b)
	if truncated {
		body += "\n\n…（内容过大，已截断至 2MB）"
	}
	return FilePreview{Path: rel, Markdown: body, Size: int64(len(b))}
}

// WenxueReadFilePage returns one line-aligned page from a large text file.
func (a *App) WenxueReadFilePage(rel string, offset int64) (workbench.TextPage, error) {
	return workbench.ReadPage(gaeaCwd(), rel, offset)
}

// WenxueReadTextFile returns one full editable document with its mtime baseline,
// detected encoding and line-ending convention.
func (a *App) WenxueReadTextFile(rel string) (workbench.TextFile, error) {
	return workbench.ReadText(gaeaCwd(), rel)
}

// WenxueRenameWorkspacePath renames one file or directory row.
func (a *App) WenxueRenameWorkspacePath(rel, name string) (workbench.MutationResult, error) {
	return workbench.Rename(gaeaCwd(), rel, name)
}

func (a *App) WenxueRevealWorkspacePath(rel string) error {
	path, _ := wenxueResolvePreviewPath(rel)
	return exec.Command("explorer", "/select,", path).Start()
}

// WenxueSaveAttachmentFile 保存附件文件到工作区 .gaea/uploads。
func (a *App) WenxueSaveAttachmentFile(fileName, base64Data string) (string, error) {
	store, err := gaeaAttachment.Open(filepath.Join(gaeaCwd(), ".gaea", "uploads"))
	if err != nil {
		return "", err
	}
	defer func() { _ = store.Close() }()
	return store.SaveFileBase64(fileName, base64Data)
}

// WenxueSavePastedImage 保存粘贴图片到工作区 .gaea/uploads。
func (a *App) WenxueSavePastedImage(dataURL string) (string, error) {
	store, err := gaeaAttachment.Open(filepath.Join(gaeaCwd(), ".gaea", "uploads"))
	if err != nil {
		return "", err
	}
	defer func() { _ = store.Close() }()
	return store.SaveImageDataURL(dataURL)
}

// WenxueSaveWorkspacePathAs exports one file, or one directory as ZIP, through
// the native Save dialog. An empty result means the user cancelled.
func (a *App) WenxueSaveWorkspacePathAs(rel string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	clean := strings.TrimSpace(rel)
	if clean == "" {
		return "", fmt.Errorf("路径为空")
	}
	defaultName := filepath.Base(filepath.FromSlash(clean))
	isDir, err := workbench.Kind(gaeaCwd(), clean)
	if err != nil {
		return "", err
	}
	if isDir {
		defaultName += ".zip"
	}
	dest, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存到...",
		DefaultFilename: defaultName,
	})
	if err != nil {
		return "", err
	}
	if dest == "" {
		return "", nil
	}
	if err := workbench.ExportPath(gaeaCwd(), clean, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// WenxueSaveWorkspacePathsAs exports multiple selected rows as one ZIP.
func (a *App) WenxueSaveWorkspacePathsAs(paths []string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("未选择要保存的路径")
	}
	if len(paths) == 1 {
		return a.WenxueSaveWorkspacePathAs(paths[0])
	}
	dest, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "打包保存所选项目",
		DefaultFilename: "gaea-selection.zip",
	})
	if err != nil {
		return "", err
	}
	if dest == "" {
		return "", nil
	}
	if err := workbench.ExportPaths(gaeaCwd(), paths, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// WenxueScheduleLoad 装载当前工程计划文件（索引 current 指针；索引不可用回落
// DefaultRelPath）。文件不存在返回 Exists=false（不视为错误）。
func (a *App) WenxueScheduleLoad() (ScheduleLoadResult, error) {
	rel := scheduleCurrentRel(gaeaCwd())
	// 审计 2026-10-02 AP5-09：rel 来自索引指针（.gaea/schedule/index.json，
	// 磁盘文件=不可信面），改调唯一原语——越界不再被 Join 静默清洗成工作区外
	// 路径后读取；按读取失败口径如实回报。
	path, perr := wspath.ResolveRelWithin(gaeaCwd(), rel)
	if perr != nil {
		return ScheduleLoadResult{}, fmt.Errorf("非法计划文件路径（越出工作区）：%s", rel)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ScheduleLoadResult{Path: rel, Exists: false}, nil
		}
		return ScheduleLoadResult{}, fmt.Errorf("读取计划文件失败：%w", err)
	}
	// 落盘前过引擎校验（坏文件不在板块里静默炸，给出可读错误）
	if _, err := schedule.Load(path); err != nil {
		return ScheduleLoadResult{}, err
	}
	return ScheduleLoadResult{Path: rel, Exists: true, Project: string(raw)}, nil
}

// WenxueScheduleProjectArchive 归档/反归档：索引标 archived，文件不动（用户
// 资产，agent 显式 path 引用不失效）；未知 rel 报错。
func (a *App) WenxueScheduleProjectArchive(rel string, archived bool) (ScheduleProjectsResult, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	cwd := gaeaCwd()
	idx, err := schedule.LoadScheduleIndex(cwd)
	if err != nil {
		return ScheduleProjectsResult{}, err
	}
	found := false
	for i := range idx.Projects {
		if idx.Projects[i].Rel == rel {
			idx.Projects[i].Archived = archived
			found = true
			break
		}
	}
	if !found {
		return ScheduleProjectsResult{}, fmt.Errorf("未知工程：%s", rel)
	}
	if err := schedule.SaveScheduleIndex(cwd, idx); err != nil {
		return ScheduleProjectsResult{}, err
	}
	return buildScheduleProjectsResult(cwd, idx), nil
}

// WenxueScheduleProjectCopy 复制工程（v4.145「另存为」）：源文件 schedule.Load
// 全套校验 → 新名 trim 非空 → SafeSlugName（冲突加序号，同 Create 口径，
// 登记条目与目录游离文件都算占用）→ 内容深拷贝仅改 name 落新文件（基线/
// AOA 布点/日历随行）→ 索引显式登记（未归档，UpdatedAt=now）。指针不动
// （文件级动作语义，同归档——复制品是快照，用户自行切换打开）。
// 回执含列表，前端取回执刷新即可。
func (a *App) WenxueScheduleProjectCopy(rel, name string) (ScheduleProjectsResult, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	name = strings.TrimSpace(name)
	if name == "" {
		return ScheduleProjectsResult{}, fmt.Errorf("工程名称为空")
	}
	if err := checkScheduleRel(rel); err != nil {
		return ScheduleProjectsResult{}, err
	}
	if rel == "" {
		return ScheduleProjectsResult{}, fmt.Errorf("源工程路径为空")
	}
	cwd := gaeaCwd()
	src := filepath.Join(cwd, filepath.FromSlash(rel))
	if _, err := os.Stat(src); err != nil {
		return ScheduleProjectsResult{}, fmt.Errorf("源工程不存在：%s", rel)
	}
	p, err := schedule.Load(src)
	if err != nil {
		return ScheduleProjectsResult{}, fmt.Errorf("源工程读取失败：%w", err)
	}
	idx, err := schedule.LoadScheduleIndex(cwd)
	if err != nil {
		return ScheduleProjectsResult{}, err
	}
	existing := map[string]bool{}
	for _, e := range idx.Projects {
		existing[scheduleRelBase(e.Rel)] = true
	}
	if scanned, err := schedule.ScanScheduleProjects(cwd); err == nil {
		for _, e := range scanned {
			existing[scheduleRelBase(e.Rel)] = true
		}
	}
	slug := schedule.SafeSlugName(name, existing)
	newRel := path.Join(schedule.ScheduleProjectsDir, slug+schedule.ScheduleFileSuffix)
	p.Name = name
	if err := schedule.Save(filepath.Join(cwd, filepath.FromSlash(newRel)), p); err != nil {
		return ScheduleProjectsResult{}, err
	}
	idx.Projects = append(idx.Projects, schedule.ScheduleIndexEntry{Rel: newRel, Name: name, UpdatedAt: time.Now()})
	if err := schedule.SaveScheduleIndex(cwd, idx); err != nil {
		return ScheduleProjectsResult{}, err
	}
	return buildScheduleProjectsResult(cwd, idx), nil
}

// WenxueScheduleProjectCreate 新建工程：name trim 非空 → SafeSlugName（冲突加
// 序号）→ 进度计划/<slug>.gsched.json 落空 Project（走 schedule.Save 全套
// 校验，name=用户工程名）→ 索引登记 → 自动切为当前。此后文件名不再改。
func (a *App) WenxueScheduleProjectCreate(name string) (ScheduleCreateResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ScheduleCreateResult{}, fmt.Errorf("工程名称为空")
	}
	cwd := gaeaCwd()
	// 文件名占用判定：登记条目与目录游离文件都算（此后文件名不再改，撞名代价高）
	existing := map[string]bool{}
	if idx, err := schedule.LoadScheduleIndex(cwd); err == nil {
		for _, e := range idx.Projects {
			existing[scheduleRelBase(e.Rel)] = true
		}
	}
	if scanned, err := schedule.ScanScheduleProjects(cwd); err == nil {
		for _, e := range scanned {
			existing[scheduleRelBase(e.Rel)] = true
		}
	}
	slug := schedule.SafeSlugName(name, existing)
	rel := path.Join(schedule.ScheduleProjectsDir, slug+schedule.ScheduleFileSuffix)
	p := schedule.Project{Name: name, Tasks: []schedule.Task{}, Links: []schedule.Link{}}
	if err := schedule.Save(filepath.Join(cwd, filepath.FromSlash(rel)), p); err != nil {
		return ScheduleCreateResult{}, err
	}
	if err := schedule.SetCurrentSchedule(cwd, rel); err != nil {
		return ScheduleCreateResult{}, err
	}
	return ScheduleCreateResult{Rel: rel, Current: rel}, nil
}

// WenxueScheduleProjectDelete 物理删文件+索引摘除（不可恢复，前端二次确认）。
// 删的是当前 → current 切到剩余第一个未归档工程；没有则回落 DefaultRelPath
// 并确保该文件存在（不存在则落空工程）。DefaultRelPath 文件允许删除，删后
// 同上规则。未知 rel（不在注册表且文件也不在盘）报错。
func (a *App) WenxueScheduleProjectDelete(rel string) (ScheduleProjectsResult, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if err := checkScheduleRel(rel); err != nil {
		return ScheduleProjectsResult{}, err
	}
	cwd := gaeaCwd()
	idx, err := schedule.LoadScheduleIndex(cwd)
	if err != nil {
		return ScheduleProjectsResult{}, err
	}
	kept := make([]schedule.ScheduleIndexEntry, 0, len(idx.Projects))
	found := false
	for _, e := range idx.Projects {
		if e.Rel == rel {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	abs := filepath.Join(cwd, filepath.FromSlash(rel))
	_, statErr := os.Stat(abs)
	if !found && statErr != nil {
		return ScheduleProjectsResult{}, fmt.Errorf("未知工程：%s", rel)
	}
	if statErr == nil {
		if err := os.Remove(abs); err != nil {
			return ScheduleProjectsResult{}, fmt.Errorf("删除计划文件失败：%w", err)
		}
	}
	idx.Projects = kept
	if idx.Current == rel {
		idx.Current = schedule.DefaultRelPath
		for _, e := range kept {
			if !e.Archived {
				idx.Current = e.Rel
				break
			}
		}
		if idx.Current == schedule.DefaultRelPath {
			if err := ensureSchedulePlanFile(cwd, schedule.DefaultRelPath); err != nil {
				return ScheduleProjectsResult{}, err
			}
		}
	}
	if err := schedule.SaveScheduleIndex(cwd, idx); err != nil {
		return ScheduleProjectsResult{}, err
	}
	return buildScheduleProjectsResult(cwd, idx), nil
}

// WenxueScheduleProjectOpen 打开工程：SetCurrentSchedule 语义（校验 rel 在注册
// 表或文件在盘且后缀合法 → 原子写索引 current）。不搬数据、不改工程文件。
func (a *App) WenxueScheduleProjectOpen(rel string) (ScheduleOpenResult, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if err := schedule.SetCurrentSchedule(gaeaCwd(), rel); err != nil {
		return ScheduleOpenResult{}, err
	}
	return ScheduleOpenResult{Current: rel}, nil
}

// WenxueScheduleProjects 工程列表+当前指针：逐文件 Load+Analyze 出摘要
// （工程数量级=个位数，开销可接受；单文件坏 → Ok=false 不计 Duration，
// 诚实呈现不静默吞）。含已归档条目（Archived 标记，切换器/管理列表各取所需）。
func (a *App) WenxueScheduleProjects() (ScheduleProjectsResult, error) {
	cwd := gaeaCwd()
	idx, err := schedule.LoadScheduleIndex(cwd)
	if err != nil {
		return ScheduleProjectsResult{}, err
	}
	return buildScheduleProjectsResult(cwd, idx), nil
}

// WenxueScheduleSave 保存板块计划（整量覆盖）：校验+CPM fail-closed+原子写。
// rel 为保存目标（板块装载时绑定，规避切换竞态——切换只改指针不动保存目标）；
// rel 为空回落索引 current（wails 缺参零值语义，老调用形态兜底）。
func (a *App) WenxueScheduleSave(projectJSON, rel string) (ScheduleSaveResult, error) {
	if strings.TrimSpace(projectJSON) == "" {
		return ScheduleSaveResult{}, fmt.Errorf("计划内容为空")
	}
	var p schedule.Project
	if err := json.Unmarshal([]byte(projectJSON), &p); err != nil {
		return ScheduleSaveResult{}, fmt.Errorf("计划 JSON 解析失败：%w", err)
	}
	if err := checkScheduleRel(rel); err != nil {
		return ScheduleSaveResult{}, err
	}
	cwd := gaeaCwd()
	rel = strings.TrimSpace(rel)
	if rel == "" {
		rel = scheduleCurrentRel(cwd)
	}
	rel = filepath.ToSlash(rel)
	// 审计 2026-10-02 AP5-09：落盘路径改调唯一原语。checkScheduleRel 仍是更严的
	// 段级前置门（`a/../b` 形态它拒、Clean 口径允许），本原语补上「归一后必须
	// 落在工作区内」这一条，堵索引 current 指针/rel 变体走私工作区外路径。
	absPath, perr := wspath.ResolveRelWithin(cwd, rel)
	if perr != nil {
		return ScheduleSaveResult{}, fmt.Errorf("非法计划文件路径（越出工作区）：%s", rel)
	}
	if err := schedule.Save(absPath, p); err != nil {
		return ScheduleSaveResult{}, err
	}
	// 保存成功后更新索引该条目 updatedAt（索引不存在则顺带建）；索引是缓存
	// 非权威，写失败不拦保存回执（下次 Load 轻扫会再收编）。
	touchScheduleIndexEntry(cwd, rel, p.Name)
	cpm, a2 := p.Analyze()
	dur, crit := 0, 0
	if cpm.OK {
		dur, crit = cpm.Duration, len(a2.Critical)
	}
	return ScheduleSaveResult{
		Path:     rel,
		SavedAt:  time.Now().Format("15:04"),
		Duration: dur,
		Critical: crit,
	}, nil
}

// WenxueUnpinMaterial 取消固定一个文件，返回更新后的固定清单。
func (a *App) WenxueUnpinMaterial(rel string) []FileSearchHit {
	if _, err := pins.Remove(gaeaCwd(), rel); err != nil {
		return []FileSearchHit{}
	}
	return pinnedView(gaeaCwd())
}

// WenxueWorkspaceUploadAbort discards an incomplete upload.
func (a *App) WenxueWorkspaceUploadAbort(token string) error {
	return a.workspaceUploadManager().Abort(token)
}

// WenxueWorkspaceUploadAppend appends one independently-decodable base64 chunk.
func (a *App) WenxueWorkspaceUploadAppend(token, base64Chunk string) error {
	return a.workspaceUploadManager().Append(token, base64Chunk)
}

// WenxueWorkspaceUploadBegin starts a chunked upload below dir.
func (a *App) WenxueWorkspaceUploadBegin(dir, relativePath string) (string, error) {
	return a.workspaceUploadManager().Begin(gaeaCwd(), dir, relativePath)
}

// WenxueWorkspaceUploadFinish publishes the uploaded file.
func (a *App) WenxueWorkspaceUploadFinish(token string) (workbench.UploadResult, error) {
	return a.workspaceUploadManager().Finish(token)
}

// WenxueWriteWorkspaceFile saves an editor draft using the file's original
// encoding/EOL, rejecting a stale mtime baseline.
func (a *App) WenxueWriteWorkspaceFile(rel, content string, expectedMtimeMs int64) (workbench.WriteResult, error) {
	return workbench.WriteText(gaeaCwd(), rel, content, expectedMtimeMs)
}

// WenxueZipDeliverables 会话产物一键打包：把本次会话交付的文件（工作区相对路径列表）
// 打成一个 zip 放进 .gaea/exports/，对标 Kimi 工作空间 / WorkBuddy 会话产物打包。
// 安全：只接受工作区内的相对路径（拒绝绝对路径与 .. 穿越），缺失/目录条目静默跳过。
func (a *App) WenxueZipDeliverables(paths []string) (ZipDeliverableResult, error) {
	if len(paths) == 0 {
		return ZipDeliverableResult{}, fmt.Errorf("没有可打包的会话产物")
	}

	// 收集存在的工作区文件（相对路径清洗 + 去重），zip 内保留相对路径结构，
	// 不同目录下的同名文件不会互相覆盖。
	type entry struct{ abs, rel string }
	var files []entry
	seen := map[string]bool{}
	for _, p := range paths {
		clean := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(p, "\\", "/")))
		clean = strings.TrimPrefix(clean, "./")
		if clean == "" || clean == "." || strings.HasPrefix(clean, "../") {
			continue // 拒绝 .. 穿越
		}
		if filepath.IsAbs(clean) || filepath.IsAbs(p) {
			continue // 拒绝绝对路径（产物面板路径均为工作区相对路径）
		}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		// 审计 2026-10-02 AP5-09：相对条目改调唯一原语（原为手写 ../ 前缀 +
		// IsAbs 判据；归一后越界的条目静默跳过，与既有「缺失条目跳过」语义一致）。
		abs, rerr := wspath.ResolveRelWithin(gaeaCwd(), clean)
		if rerr != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		files = append(files, entry{abs: abs, rel: clean})
	}
	if len(files) == 0 {
		return ZipDeliverableResult{}, fmt.Errorf("会话产物文件都不存在或不可访问")
	}

	// S4 产物路径分区：work 恒 .gaea/exports（现状不动），play 落 .gaea/play/exports。
	exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return ZipDeliverableResult{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	zipName := fmt.Sprintf("gaea-会话产物-%s.zip", stamp)
	zipPath := filepath.Join(exportsDir, zipName)

	fz, err := os.Create(zipPath)
	if err != nil {
		return ZipDeliverableResult{}, err
	}
	zw := zip.NewWriter(fz)
	var total int64
	for _, f := range files {
		info, err := os.Stat(f.abs)
		if err != nil {
			continue
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			continue
		}
		fh.Name = filepath.ToSlash(f.rel)
		fh.Method = zip.Deflate
		w, err := zw.CreateHeader(fh)
		if err != nil {
			continue
		}
		raw, err := os.ReadFile(f.abs)
		if err != nil {
			continue
		}
		if _, err := w.Write(raw); err != nil {
			continue
		}
		total += int64(len(raw))
	}
	if err := zw.Close(); err != nil {
		_ = fz.Close()
		_ = os.Remove(zipPath)
		return ZipDeliverableResult{}, err
	}
	if err := fz.Close(); err != nil {
		_ = os.Remove(zipPath)
		return ZipDeliverableResult{}, err
	}

	rel, _ := filepath.Rel(gaeaCwd(), zipPath)
	return ZipDeliverableResult{
		Path:    filepath.ToSlash(rel),
		Name:    filepath.Base(zipPath),
		Entries: len(files),
		Bytes:   total,
	}, nil
}

// wenxueExportDocx 经 create_docx.py（python-docx）从 Markdown 排版 Word。
func wenxueExportDocx(in ExportDeliverableInput, title, outPath string) error {
	script := wenxueFindSkillScript("docx", "scripts", "create_docx.py")
	if script == "" {
		return fmt.Errorf("未找到 create_docx.py（.gaea/skills/docx/scripts）")
	}
	mdPath := strings.TrimSuffix(outPath, ".docx") + ".md"
	if err := os.WriteFile(mdPath, []byte(in.Markdown), 0o644); err != nil {
		return err
	}
	defer func() { _ = os.Remove(mdPath) }()

	args := []string{script, mdPath, outPath, "--title", title}
	if in.Cover {
		args = append(args, "--cover")
	}
	if in.TOC {
		args = append(args, "--toc")
	}
	if in.Header != "" {
		args = append(args, "--header", in.Header)
	}
	if in.Footer != "" {
		args = append(args, "--footer", in.Footer)
	}
	args = append(args, "--template", in.Template)
	if err := runPython(args, 120); err != nil {
		return fmt.Errorf("Word 导出失败: %w", err)
	}
	return nil
}

// wenxueExportPptx 把 Markdown 转成简单 slides spec，经 create_pptx.py 出 16:9 演示文稿。
func wenxueExportPptx(in ExportDeliverableInput, title, outPath string) error {
	script := wenxueFindSkillScript("pptx", "scripts", "create_pptx.py")
	if script == "" {
		return fmt.Errorf("未找到 create_pptx.py（.gaea/skills/pptx/scripts）")
	}
	spec := markdownToSlides(in.Markdown, title)
	specPath := strings.TrimSuffix(outPath, ".pptx") + ".json"
	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(specPath, b, 0o644); err != nil {
		return err
	}
	defer func() { _ = os.Remove(specPath) }()
	if err := runPython([]string{script, specPath, outPath}, 120); err != nil {
		return fmt.Errorf("PPT 导出失败: %w", err)
	}
	return nil
}

// wenxueFindSkillScript 定位技能脚本（工作区 .gaea/skills 或用户 skills 镜像）。
func wenxueFindSkillScript(skill, sub, name string) string {
	candidates := []string{
		filepath.Join(gaeaCwd(), ".gaea", "skills", skill, sub, name),
		filepath.Join(gaeaCwd(), "skills", skill, sub, name),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".codex", "skills", skill, sub, name),
			filepath.Join(home, ".gaea", "skills", skill, sub, name),
		)
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// wenxueGitRun 在工作区 cwd 执行 git 子命令（读面默认档）。
func wenxueGitRun(args ...string) (string, error) {
	return gitRunAt(wenxueGitWorkspace(), args...)
}

// wenxueGitWorkspace 返回 git 面的工作区锚：gaeaCwd 优先，回退进程启动目录。
func wenxueGitWorkspace() string {
	cwd := gaeaCwd()
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return cwd
}

// wenxueListDirEntries 列出目录条目。rel 为空 = 工作区根；相对路径 Join 工作区根
// （拒 .. 穿越——2026-09-20 审计：全盘目录枚举原语，与其他读口的收口对齐；
// 前端四个调用方 FileTree/useComposerMenus/deliverablesTurn/
// VerifyArtifactsThumbs 全部传工作区相对路径，零误伤）；绝对路径 Clean 后
// 直接使用（保留——口径对齐 wenxueResolvePreviewPath 的 IsAbs 分支，含正斜杠写法
// 的 Windows 盘符路径，与前端 ToSlash 口径互通）。错误带 GAEADIR_* 码透传。
func wenxueListDirEntries(rel string) ([]DirEntry, error) {
	root := gaeaCwd()
	dir := root
	if rel != "" {
		if filepath.IsAbs(rel) {
			dir = filepath.Clean(rel)
		} else {
			// 审计 2026-10-02 AP5-09：相对分支改调唯一原语（原为手写
			// Clean + `..` 前缀/片段判据，第五份同族实现）。
			p, err := wspath.ResolveRelWithin(root, rel)
			if err != nil {
				return nil, listDirError(errCodeDirRead, "路径越出工作区: %s", rel)
			}
			dir = p
		}
	}
	info, err := osStat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, listDirError(errCodeDirNotFound, "目录不存在: %s（%v）", filepath.ToSlash(dir), err)
		}
		// Stat 本身失败（父目录无权限等）：同读失败口径透传原文
		return nil, listDirError(errCodeDirRead, "读取目录失败: %s（%v）", filepath.ToSlash(dir), err)
	}
	if !info.IsDir() {
		return nil, listDirError(errCodeDirNotDir, "不是目录: %s", filepath.ToSlash(dir))
	}
	entries, err := osReadDir(dir)
	if err != nil {
		// 权限等 OS 错误：码 + 原文一并透传（不再吞）
		return nil, listDirError(errCodeDirRead, "读取目录失败: %s（%v）", filepath.ToSlash(dir), err)
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		out = append(out, DirEntry{Name: e.Name(), IsDir: e.IsDir(), Size: size})
	}
	return out, nil
}

// wenxueMarkdownToPdf 受控 Markdown → docx（临时）→ soffice PDF，
// 供 WenxueExportDeliverable 的 pdf 格式复用交付模板链路。
func wenxueMarkdownToPdf(in ExportDeliverableInput, title, outPath string) error {
	tmpDir, err := os.MkdirTemp("", "gaea-pdf-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	docxPath := filepath.Join(tmpDir, safeDeliverableName(title)+".docx")
	if err := wenxueExportDocx(in, title, docxPath); err != nil {
		return err
	}
	return convertToPdfFile(docxPath, outPath)
}

// wenxueWithinReadRoots 判断绝对路径是否落在任一可读数据根内（工作区 + 应用数据
// 根 + 图片保存目录）。2026-09-19 审计新增：读侧附件/预览绑定共用的根白名单。
// v4.363 真机走查补：绘梦历史图实际落 cfg.ImageSaveDir（默认 %USERPROFILE%
// Pictures\gaea），不在工作区/数据根两根内——当时「绘梦资产均落两根内」的
// 论断不成立，聊天消息内联图/画廊读图被误拒，此处补第三根。
// wenxueResolveAgainstWorkspace 相对路径按工作区根解析（与 WenxueListDir 同惯例；
// v4.405.1 根修：WenxueAttachmentDataURL 曾把工作区相对路径直接拒之门外——
// .gaea/work/<文档>/插图/x.png 相对形态合法存在）。绝对路径原样返回。
func wenxueResolveAgainstWorkspace(p string) string {
	p = filepath.Clean(filepath.FromSlash(p))
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(gaeaCwd(), p))
}

// wenxueResolvePreviewPath 把工作区相对路径解析为绝对路径并返回展示用相对路径；
// 裸文件名（无目录分隔符）直接找不到时，在常见输出目录里查找同名文件，
// 使“输出文件：成本测算.xlsx”这类纯文件名引用也能直接预览。
func wenxueResolvePreviewPath(rel string) (path, displayRel string) {
	root := gaeaCwd()
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel), rel
	}
	// 相对路径拒 .. 穿越（2026-09-19 审计 P1：Join(root, rel) 此前直通；
	// 绝对路径分支保留——附件/素材预览有合法绝对路径用途，文件大小由各
	// 分支封顶）。
	//
	// 审计 2026-10-02 AP5-09 修复：原实现「拒 ..」分支返回的是
	// filepath.Join(root, cleanRel)（= 工作区**外**的逃逸路径），只是赌
	// 后续 os.Stat 失败——逃逸目标真实存在时预览会照读照渲染（真穿越）。
	// 现改调唯一原语：越界即返回空路径，WenxuePreview 的 os.Stat("") 必失败
	// → 前端得到既有「文件不存在」降级，逃逸路径不进任何读口。
	p, err := wspath.ResolveRelWithin(root, rel)
	if err != nil {
		return "", rel
	}
	display := filepath.ToSlash(rel)
	joined := p
	if fileExists(joined) {
		return joined, display
	}
	if !strings.ContainsAny(rel, `/\`) {
		if found := findInOutputDirs(root, rel); found != "" {
			if r, err := filepath.Rel(root, found); err == nil {
				display = filepath.ToSlash(r)
			}
			return found, display
		}
	}
	return joined, display
}

func (a *App) wenxueWithinReadRoots(abs string) bool {
	roots := []string{gaeaCwd()}
	if dr := config.DataRoot(); dr != "" {
		roots = append(roots, dr)
	}
	if a != nil && a.core != nil && a.cfg != nil {
		if sd := strings.TrimSpace(a.cfg.ImageSaveDir); sd != "" {
			roots = append(roots, sd)
		}
	}
	// 与 image_handler 的兜底一致：未配置时生成图仍会落 %USERPROFILE%\Pictures\gaea。
	if up := os.Getenv("USERPROFILE"); up != "" {
		roots = append(roots, filepath.Join(up, "Pictures", "gaea"))
	}
	abs = wenxueResolveAgainstWorkspace(abs)
	for _, r := range roots {
		// 审计 2026-10-02 AP5-09：包含性判据改调唯一原语 wspath.Within
		// （原为 strings.HasPrefix(abs, root+sep) 手写前缀法——大小写敏感，
		// Windows 上 `c:\ws\x` 会被误拒；Rel 口径与文件系统语义一致）。
		if wspath.Within(r, abs) {
			return true
		}
	}
	return false
}

// wenxueWithinWriteRoots 判断绝对路径是否落在任一可写根（工作区 + allow_write）内。
// 相对路径按工作区根解析（v4.405.1，与读侧同惯例）。
func wenxueWithinWriteRoots(abs string) bool {
	roots := []string{gaeaCwd()}
	if ga.cfg != nil {
		roots = ga.cfg.WriteRoots()
	}
	abs = wenxueResolveAgainstWorkspace(abs)
	for _, r := range roots {
		// 审计 2026-10-02 AP5-09：同读侧，改调 wspath.Within（root 本身仍算内）。
		if wspath.Within(r, abs) {
			return true
		}
	}
	return false
}
