package builtin

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(archiveTool{}) }

type archiveTool struct{}

func (archiveTool) Name() string { return "archive" }

func (archiveTool) Description() string {
	return "压缩/解压工具：支持 zip 格式。压缩时将文件或目录打包为 zip；解压时提取 zip 到指定目录。"
}

func (archiveTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "action":{"type":"string","description":"操作：zip（压缩）、unzip（解压）"},
  "source":{"type":"string","description":"源路径（压缩时为文件/目录，解压时为 zip 文件）"},
  "target":{"type":"string","description":"目标路径（压缩时为 zip 输出路径，解压时为目标目录），不指定时自动生成"},
  "files":{"type":"array","items":{"type":"string"},"description":"压缩时指定文件列表（可选，不指定则压缩整个 source 目录）"}
},
"required":["action","source"]
}`)
}

func (archiveTool) ReadOnly() bool { return false }

func (archiveTool) CompactDescription() string     { return compactDesc["archive"] }
func (archiveTool) CompactSchema() json.RawMessage { return compactSchema["archive"] }

func (archiveTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action string   `json:"action"`
		Source string   `json:"source"`
		Target string   `json:"target,omitempty"`
		Files  []string `json:"files,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	p.Action = strings.TrimSpace(strings.ToLower(p.Action))
	if p.Action == "" {
		return "", fmt.Errorf("action 必须为 zip 或 unzip")
	}
	if p.Source == "" {
		return "", fmt.Errorf("source 不能为空")
	}

	switch p.Action {
	case "zip":
		return doZip(p.Source, p.Target, p.Files)
	case "unzip":
		return doUnzip(p.Source, p.Target)
	default:
		return "", fmt.Errorf("不支持的 action: %s（仅支持 zip/unzip）", p.Action)
	}
}

func doZip(source, target string, files []string) (string, error) {
	// 确定输出路径
	if target == "" {
		target = source + ".zip"
	}
	if !strings.HasSuffix(target, ".zip") {
		target += ".zip"
	}

	f, err := os.Create(target)
	if err != nil {
		return "", fmt.Errorf("创建 zip 文件失败: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	if len(files) > 0 {
		// 指定文件列表
		for _, file := range files {
			if err := addFileToZip(w, file, ""); err != nil {
				return "", fmt.Errorf("添加文件 %s 失败: %w", file, err)
			}
		}
	} else {
		// 打包整个 source
		info, err := os.Stat(source)
		if err != nil {
			return "", fmt.Errorf("无法访问 %s: %w", source, err)
		}
		if info.IsDir() {
			// baseDir removed
			err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if path == source {
					return nil
				}
				relPath, _ := filepath.Rel(filepath.Dir(source), path)
				return addEntryToZip(w, path, relPath, info)
			})
		} else {
			if err := addFileToZip(w, source, ""); err != nil {
				return "", err
			}
		}
		if err != nil {
			return "", fmt.Errorf("打包失败: %w", err)
		}
	}

	w.Close()
	info, _ := os.Stat(target)
	return tool.WrapText(fmt.Sprintf("✅ 已创建: %s（%d 字节，%d 个文件）", target, info.Size(), len(files))), nil
}

func addFileToZip(w *zip.Writer, path, prefix string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.Walk(path, func(subPath string, subInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(filepath.Dir(path), subPath)
			if prefix != "" {
				relPath = filepath.Join(prefix, relPath)
			}
			return addEntryToZip(w, subPath, relPath, subInfo)
		})
	}
	return addEntryToZip(w, path, filepath.Base(path), info)
}

func addEntryToZip(w *zip.Writer, srcPath, zipPath string, info os.FileInfo) error {
	if info.IsDir() {
		_, err := w.Create(zipPath + "/")
		return err
	}
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	header, _ := zip.FileInfoHeader(info)
	header.Name = zipPath
	header.Method = zip.Deflate

	zw, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(zw, f)
	return err
}

func doUnzip(source, target string) (string, error) {
	r, err := zip.OpenReader(source)
	if err != nil {
		return "", fmt.Errorf("打开 zip 文件失败: %w", err)
	}
	defer r.Close()

	if target == "" {
		target = strings.TrimSuffix(source, ".zip")
	}

	extracted := 0
	for _, f := range r.File {
		// 路径穿越安全检查
		cleanPath := filepath.Clean(f.Name)
		if strings.HasPrefix(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") {
			continue // 跳过不安全的路径
		}
		targetPath := filepath.Join(target, cleanPath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(targetPath, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(targetPath), 0755)

		rc, err := f.Open()
		if err != nil {
			continue // 跳过损坏的条目
		}

		out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			continue
		}

		io.Copy(out, rc)
		out.Close()
		rc.Close()
		extracted++
	}

	return tool.WrapText(fmt.Sprintf("✅ 已解压到: %s（%d 个文件/目录）", target, extracted)), nil
}
