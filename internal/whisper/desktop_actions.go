// Package whisper — desktop_actions.go
// 100% 对齐 ackem desktop-agent/actions.ts
// 桌面助手操作标签和分类
package whisper

// DesktopAgentAction 桌面助手操作类型
type DesktopAgentAction string

const (
	ActionListFolder         DesktopAgentAction = "list_folder"
	ActionSearchFiles        DesktopAgentAction = "search_files"
	ActionStatFile           DesktopAgentAction = "stat_file"
	ActionGrepText           DesktopAgentAction = "grep_text"
	ActionReadText           DesktopAgentAction = "read_text"
	ActionReadDocument       DesktopAgentAction = "read_document"
	ActionReadImage          DesktopAgentAction = "read_image"
	ActionOpenFolder         DesktopAgentAction = "open_folder"
	ActionOpenFile           DesktopAgentAction = "open_file"
	ActionOpenApp            DesktopAgentAction = "open_app"
	ActionCloseFile          DesktopAgentAction = "close_file"
	ActionCloseApp           DesktopAgentAction = "close_app"
	ActionCopyPath           DesktopAgentAction = "copy_path"
	ActionMovePath           DesktopAgentAction = "move_path"
	ActionMkdir              DesktopAgentAction = "mkdir"
	ActionWriteText          DesktopAgentAction = "write_text"
	ActionDeletePath         DesktopAgentAction = "delete_path"
	ActionDownloadFile       DesktopAgentAction = "download_file"
	ActionDownloadAndInstall DesktopAgentAction = "download_and_install"
	ActionRunInstaller       DesktopAgentAction = "run_installer"
	ActionImportToAckem      DesktopAgentAction = "import_to_ackem"
	ActionFocusApp           DesktopAgentAction = "focus_app"
)

// DesktopAgentActionLabels 22 个操作的中文标签
var DesktopAgentActionLabels = map[DesktopAgentAction]string{
	ActionListFolder:         "列出目录内容",
	ActionSearchFiles:        "搜索文件",
	ActionStatFile:           "查看文件信息",
	ActionGrepText:           "在目录中搜索文本",
	ActionReadText:           "读取文本文件",
	ActionReadDocument:       "读取文档",
	ActionReadImage:          "读取图片",
	ActionOpenFolder:         "打开文件夹",
	ActionOpenFile:           "打开文件",
	ActionOpenApp:            "打开应用程序",
	ActionCloseFile:          "关闭文件窗口",
	ActionCloseApp:           "关闭应用程序",
	ActionCopyPath:           "复制",
	ActionMovePath:           "移动或重命名",
	ActionMkdir:              "新建文件夹",
	ActionWriteText:          "写入文本文件",
	ActionDeletePath:         "删除",
	ActionDownloadFile:       "下载文件",
	ActionDownloadAndInstall: "下载并安装",
	ActionRunInstaller:       "运行安装包",
	ActionImportToAckem:      "导入到轻语",
	ActionFocusApp:           "将应用带到前台",
}

// CloseActions 关闭类操作集合
var CloseActions = map[DesktopAgentAction]bool{
	ActionCloseFile: true,
	ActionCloseApp:  true,
}

// AppActions 应用操作集合
var AppActions = map[DesktopAgentAction]bool{
	ActionOpenApp:   true,
	ActionCloseApp:  true,
	ActionCloseFile: true,
	ActionFocusApp:  true,
}

// WriteActions 写入操作集合
var WriteActions = map[DesktopAgentAction]bool{
	ActionCopyPath:   true,
	ActionMovePath:   true,
	ActionMkdir:      true,
	ActionWriteText:  true,
	ActionDeletePath: true,
}

// DownloadActions 下载操作集合
var DownloadActions = map[DesktopAgentAction]bool{
	ActionDownloadFile:       true,
	ActionDownloadAndInstall: true,
	ActionRunInstaller:       true,
}

// DocumentReadActions 文档读取操作集合
var DocumentReadActions = map[DesktopAgentAction]bool{
	ActionReadDocument: true,
	ActionReadImage:    true,
}

// ActionLabel 返回操作的中文标签
func ActionLabel(action DesktopAgentAction) string {
	if l, ok := DesktopAgentActionLabels[action]; ok {
		return l
	}
	return string(action)
}
