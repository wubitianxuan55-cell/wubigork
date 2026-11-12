package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/gaea/gaea/internal/gaea/workbench"
)

const fileTreeEventChannel = "gaea-filetree"

// GaeaListDirBatch lists all expanded tree levels in one binding round trip.
func (a *App) GaeaListDirBatch(paths, exclude []string) []workbench.Level {
	return workbench.ListBatch(gaeaCwd(), paths, exclude)
}

// GaeaRenameWorkspacePath renames one file or directory row.
func (a *App) GaeaRenameWorkspacePath(rel, name string) (workbench.MutationResult, error) {
	return workbench.Rename(gaeaCwd(), rel, name)
}

// GaeaCreateWorkspaceFolder creates a directory inside an existing row.
func (a *App) GaeaCreateWorkspaceFolder(parent, name string) (workbench.MutationResult, error) {
	return workbench.CreateFolder(gaeaCwd(), parent, name)
}

// GaeaCreateWorkspaceFile creates an empty file inside an existing row.
func (a *App) GaeaCreateWorkspaceFile(parent, name string) (workbench.MutationResult, error) {
	return workbench.CreateFile(gaeaCwd(), parent, name)
}

// GaeaDeleteWorkspacePaths permanently removes selected rows.
func (a *App) GaeaDeleteWorkspacePaths(paths []string) error {
	return workbench.DeletePaths(gaeaCwd(), paths)
}

// GaeaWorkspaceUploadBegin starts a chunked upload below dir.
func (a *App) GaeaWorkspaceUploadBegin(dir, relativePath string) (string, error) {
	return a.workspaceUploadManager().Begin(gaeaCwd(), dir, relativePath)
}

// GaeaWorkspaceUploadAppend appends one independently-decodable base64 chunk.
func (a *App) GaeaWorkspaceUploadAppend(token, base64Chunk string) error {
	return a.workspaceUploadManager().Append(token, base64Chunk)
}

// GaeaWorkspaceUploadFinish publishes the uploaded file.
func (a *App) GaeaWorkspaceUploadFinish(token string) (workbench.UploadResult, error) {
	return a.workspaceUploadManager().Finish(token)
}

// GaeaWorkspaceUploadAbort discards an incomplete upload.
func (a *App) GaeaWorkspaceUploadAbort(token string) error {
	return a.workspaceUploadManager().Abort(token)
}

// GaeaPickWorkspaceUploadFiles chooses files natively and copies them into the
// tree without sending bytes through the JS bridge.
func (a *App) GaeaPickWorkspaceUploadFiles(dir string) ([]workbench.UploadResult, error) {
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

// GaeaPickWorkspaceUploadFolder chooses one directory natively and recursively
// copies it into the tree without bridging its bytes through JavaScript.
func (a *App) GaeaPickWorkspaceUploadFolder(dir string) ([]workbench.UploadResult, error) {
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

// GaeaFileTreeWatch replaces the watched directory set with exactly the levels
// the tree currently has expanded.
func (a *App) GaeaFileTreeWatch(paths, exclude []string) error {
	if a == nil || a.officeState == nil {
		return fmt.Errorf("文件工作台尚未初始化")
	}
	a.officeState.fileTreeWatchMu.Lock()
	old := a.officeState.fileTreeWatch
	a.officeState.fileTreeWatch = nil
	a.officeState.fileTreeWatchMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	if len(paths) == 0 {
		return nil
	}
	w, err := workbench.NewDirWatch(gaeaCwd(), paths, exclude, func(changed []string) {
		a.emit(fileTreeEventChannel, map[string]interface{}{"paths": changed})
	})
	if err != nil {
		return err
	}
	a.officeState.fileTreeWatchMu.Lock()
	if a.officeState.fileTreeWatch != nil {
		_ = a.officeState.fileTreeWatch.Close()
	}
	a.officeState.fileTreeWatch = w
	a.officeState.fileTreeWatchMu.Unlock()
	return nil
}

// GaeaReadTextFile returns one full editable document with its mtime baseline,
// detected encoding and line-ending convention.
func (a *App) GaeaReadTextFile(rel string) (workbench.TextFile, error) {
	return workbench.ReadText(gaeaCwd(), rel)
}

// GaeaReadFilePage returns one line-aligned page from a large text file.
func (a *App) GaeaReadFilePage(rel string, offset int64) (workbench.TextPage, error) {
	return workbench.ReadPage(gaeaCwd(), rel, offset)
}

// GaeaWriteWorkspaceFile saves an editor draft using the file's original
// encoding/EOL, rejecting a stale mtime baseline.
func (a *App) GaeaWriteWorkspaceFile(rel, content string, expectedMtimeMs int64) (workbench.WriteResult, error) {
	return workbench.WriteText(gaeaCwd(), rel, content, expectedMtimeMs)
}

// GaeaSaveWorkspacePathAs exports one file, or one directory as ZIP, through
// the native Save dialog. An empty result means the user cancelled.
func (a *App) GaeaSaveWorkspacePathAs(rel string) (string, error) {
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

// GaeaSaveWorkspacePathsAs exports multiple selected rows as one ZIP.
func (a *App) GaeaSaveWorkspacePathsAs(paths []string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("未选择要保存的路径")
	}
	if len(paths) == 1 {
		return a.GaeaSaveWorkspacePathAs(paths[0])
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

func (a *App) workspaceUploadManager() *workbench.UploadManager {
	if a == nil || a.officeState == nil {
		return nil
	}
	a.officeState.fileTreeWatchMu.Lock()
	defer a.officeState.fileTreeWatchMu.Unlock()
	if a.officeState.workspaceUploads == nil {
		a.officeState.workspaceUploads = workbench.NewUploadManager()
	}
	return a.officeState.workspaceUploads
}

func (a *App) closeWorkspaceWorkbench() {
	if a == nil || a.officeState == nil {
		return
	}
	a.officeState.fileTreeWatchMu.Lock()
	w := a.officeState.fileTreeWatch
	m := a.officeState.workspaceUploads
	a.officeState.fileTreeWatch = nil
	a.officeState.workspaceUploads = nil
	a.officeState.fileTreeWatchMu.Unlock()
	if w != nil {
		_ = w.Close()
	}
	if m != nil {
		m.Close()
	}
}
